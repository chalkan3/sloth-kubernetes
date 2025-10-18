package ingress

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/providers"
)

// NginxIngressManager manages NGINX Ingress Controller installation
type NginxIngressManager struct {
	ctx         *pulumi.Context
	domain      string
	masterNode  *providers.NodeOutput
	sshKeyPath  string
}

// NewNginxIngressManager creates a new NGINX Ingress manager
func NewNginxIngressManager(ctx *pulumi.Context, domain string) *NginxIngressManager {
	return &NginxIngressManager{
		ctx:    ctx,
		domain: domain,
	}
}

// SetMasterNode sets the master node for installation
func (n *NginxIngressManager) SetMasterNode(node *providers.NodeOutput) {
	n.masterNode = node
}

// SetSSHKeyPath sets the SSH key path
func (n *NginxIngressManager) SetSSHKeyPath(path string) {
	n.sshKeyPath = path
}

// Install installs NGINX Ingress Controller on the cluster
func (n *NginxIngressManager) Install() (pulumi.StringOutput, error) {
	if n.masterNode == nil {
		return pulumi.StringOutput{}, fmt.Errorf("master node not set")
	}

	n.ctx.Log.Info("Installing NGINX Ingress Controller", nil)

	// Install NGINX Ingress via Helm
	installIngress, err := remote.NewCommand(n.ctx, "install-nginx-ingress", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       n.masterNode.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(n.masterNode.SSHUser),
			PrivateKey: pulumi.String(n.getSSHPrivateKey()),
		},
		Create: pulumi.String(fmt.Sprintf(`
#!/bin/bash
set -e

echo "Installing NGINX Ingress Controller..."

# Wait for cluster to be ready
export KUBECONFIG=/root/kube_config_cluster.yml
kubectl wait --for=condition=Ready nodes --all --timeout=600s || true

# Install Helm if not present
if ! command -v helm &> /dev/null; then
    echo "Installing Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# Add NGINX Ingress repository
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

# Create ingress-nginx namespace
kubectl create namespace ingress-nginx --dry-run=client -o yaml | kubectl apply -f -

# Install NGINX Ingress Controller with custom values
cat > /tmp/nginx-ingress-values.yaml <<EOF
controller:
  service:
    type: LoadBalancer
    annotations:
      # DigitalOcean Load Balancer annotations
      service.beta.kubernetes.io/do-loadbalancer-protocol: "tcp"
      service.beta.kubernetes.io/do-loadbalancer-algorithm: "round_robin"
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-port: "10254"
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-protocol: "tcp"
      service.beta.kubernetes.io/do-loadbalancer-healthcheck-interval-seconds: "10"
      service.beta.kubernetes.io/do-loadbalancer-size-slug: "lb-small"
      service.beta.kubernetes.io/do-loadbalancer-hostname: "kube-ingress.%s"
    externalTrafficPolicy: Local

  replicaCount: 2

  metrics:
    enabled: true
    serviceMonitor:
      enabled: true

  config:
    use-forwarded-headers: "true"
    compute-full-forwarded-for: "true"
    use-proxy-protocol: "false"
    ssl-protocols: "TLSv1.2 TLSv1.3"
    ssl-ciphers: "ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256"

  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app.kubernetes.io/name: ingress-nginx
        topologyKey: kubernetes.io/hostname

  resources:
    limits:
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi

  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 4
    targetCPUUtilizationPercentage: 80
    targetMemoryUtilizationPercentage: 80

defaultBackend:
  enabled: true
  replicaCount: 1
  resources:
    limits:
      memory: 64Mi
    requests:
      cpu: 10m
      memory: 32Mi

tcp:
  22: "default/ssh:22"

udp: {}
EOF

# Install or upgrade NGINX Ingress
helm upgrade --install nginx-ingress ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --values /tmp/nginx-ingress-values.yaml \
  --wait \
  --timeout 10m

# Wait for the LoadBalancer to get an external IP
echo "Waiting for LoadBalancer IP..."
for i in {1..60}; do
  INGRESS_IP=$(kubectl get svc nginx-ingress-controller -n ingress-nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
  if [ ! -z "$INGRESS_IP" ]; then
    echo "LoadBalancer IP: $INGRESS_IP"
    break
  fi
  echo "Waiting for LoadBalancer IP... ($i/60)"
  sleep 10
done

# Get the final LoadBalancer IP
INGRESS_IP=$(kubectl get svc nginx-ingress-controller -n ingress-nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}' || \
             kubectl get svc nginx-ingress-controller -n ingress-nginx -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' || \
             echo "pending")

echo "INGRESS_IP:$INGRESS_IP"

# Create a test ingress
cat > /tmp/test-ingress.yaml <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ingress-test
---
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: ingress-test
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: ingress-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
        env:
        - name: MESSAGE
          value: "Kubernetes cluster is working! Ingress at kube-ingress.%s"
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: ingress-test
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: kube-ingress.%s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service
            port:
              number: 80
  - host: test.k8s.%s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service
            port:
              number: 80
EOF

kubectl apply -f /tmp/test-ingress.yaml

echo "NGINX Ingress Controller installed successfully!"
echo "Ingress endpoint: kube-ingress.%s"
echo "LoadBalancer IP: $INGRESS_IP"
`, n.domain, n.domain, n.domain, n.domain, n.domain)),
		Update: pulumi.String(`
#!/bin/bash
helm upgrade nginx-ingress ingress-nginx/ingress-nginx --namespace ingress-nginx --reuse-values
`),
		Delete: pulumi.String(`
#!/bin/bash
helm uninstall nginx-ingress --namespace ingress-nginx || true
kubectl delete namespace ingress-nginx || true
`),
	})

	if err != nil {
		return pulumi.StringOutput{}, fmt.Errorf("failed to install NGINX Ingress: %w", err)
	}

	// Extract the LoadBalancer IP from the output
	ingressIP := installIngress.Stdout.ApplyT(func(output string) string {
		// Parse the output to find INGRESS_IP:xxx.xxx.xxx.xxx
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "INGRESS_IP:") {
				return strings.TrimPrefix(line, "INGRESS_IP:")
			}
		}
		return ""
	}).(pulumi.StringOutput)

	// Export ingress information
	n.ctx.Export("nginx_ingress_installed", pulumi.Bool(true))
	n.ctx.Export("nginx_ingress_ip", ingressIP)
	n.ctx.Export("nginx_ingress_url", pulumi.Sprintf("http://kube-ingress.%s", n.domain))
	n.ctx.Export("nginx_ingress_https_url", pulumi.Sprintf("https://kube-ingress.%s", n.domain))

	return ingressIP, nil
}

// InstallCertManager installs cert-manager for automatic TLS certificates
func (n *NginxIngressManager) InstallCertManager() error {
	if n.masterNode == nil {
		return fmt.Errorf("master node not set")
	}

	n.ctx.Log.Info("Installing cert-manager for TLS certificates", nil)

	_, err := remote.NewCommand(n.ctx, "install-cert-manager", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       n.masterNode.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(n.masterNode.SSHUser),
			PrivateKey: pulumi.String(n.getSSHPrivateKey()),
		},
		Create: pulumi.String(fmt.Sprintf(`
#!/bin/bash
set -e

echo "Installing cert-manager..."

export KUBECONFIG=/root/kube_config_cluster.yml

# Add Jetstack Helm repository
helm repo add jetstack https://charts.jetstack.io
helm repo update

# Create cert-manager namespace
kubectl create namespace cert-manager --dry-run=client -o yaml | kubectl apply -f -

# Install cert-manager CRDs
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.crds.yaml

# Install cert-manager
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version v1.13.0 \
  --set installCRDs=false \
  --set global.leaderElection.namespace=cert-manager \
  --wait

# Create ClusterIssuer for Let's Encrypt
cat > /tmp/letsencrypt-issuer.yaml <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@%s
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: admin@%s
    privateKeySecretRef:
      name: letsencrypt-staging
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

kubectl apply -f /tmp/letsencrypt-issuer.yaml

echo "cert-manager installed successfully!"
`, n.domain, n.domain)),
	})

	if err != nil {
		return fmt.Errorf("failed to install cert-manager: %w", err)
	}

	n.ctx.Export("cert_manager_installed", pulumi.Bool(true))

	return nil
}

// getSSHPrivateKey retrieves the SSH private key
func (n *NginxIngressManager) getSSHPrivateKey() string {
	if n.sshKeyPath != "" {
		// Read from file in production
		return "SSH_PRIVATE_KEY_CONTENT"
	}
	return ""
}

// CreateSampleIngress creates a sample ingress resource
func (n *NginxIngressManager) CreateSampleIngress() error {
	sampleIngress := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: sample-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - kube-ingress.%s
    secretName: kube-ingress-tls
  rules:
  - host: kube-ingress.%s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: sample-service
            port:
              number: 80
`, n.domain, n.domain)

	n.ctx.Export("sample_ingress_yaml", pulumi.String(sampleIngress))

	return nil
}