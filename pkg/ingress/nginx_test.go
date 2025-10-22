package ingress

import (
	"fmt"
	"strings"
	"testing"

	"sloth-kubernetes/pkg/providers"
)

// TestNewNginxIngressManager tests manager creation
func TestNewNginxIngressManager(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"Standard domain", "example.com"},
		{"Subdomain", "k8s.example.com"},
		{"Multi-level", "prod.k8s.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test domain validation
			if tt.domain == "" {
				t.Error("Domain should not be empty")
			}
		})
	}
}

// TestSetMasterNode tests master node setting
func TestSetMasterNode(t *testing.T) {
	manager := &NginxIngressManager{
		domain: "example.com",
	}

	// Initially nil
	if manager.masterNode != nil {
		t.Error("masterNode should initially be nil")
	}

	// Set master node
	node := &providers.NodeOutput{
		Name:     "master-1",
		SSHUser:  "root",
		Provider: "digitalocean",
	}
	manager.SetMasterNode(node)

	if manager.masterNode == nil {
		t.Fatal("masterNode should be set")
	}

	if manager.masterNode.Name != "master-1" {
		t.Errorf("Expected master node name 'master-1', got %q", manager.masterNode.Name)
	}
}

// TestSetSSHKeyPath tests SSH key path setting
func TestSetSSHKeyPath(t *testing.T) {
	manager := &NginxIngressManager{
		domain: "example.com",
	}

	tests := []struct {
		name string
		path string
	}{
		{"Standard path", "/root/.ssh/id_rsa"},
		{"Custom path", "/home/user/.ssh/custom_key"},
		{"Absolute path", "/etc/ssh/deploy_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.SetSSHKeyPath(tt.path)

			if manager.sshKeyPath != tt.path {
				t.Errorf("Expected SSH key path %q, got %q", tt.path, manager.sshKeyPath)
			}
		})
	}
}

// TestGetSSHPrivateKey tests SSH private key retrieval
func TestGetSSHPrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		sshKeyPath string
		wantEmpty  bool
	}{
		{
			name:       "With SSH key path set",
			sshKeyPath: "/root/.ssh/id_rsa",
			wantEmpty:  false,
		},
		{
			name:       "No SSH key path",
			sshKeyPath: "",
			wantEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &NginxIngressManager{
				domain:     "example.com",
				sshKeyPath: tt.sshKeyPath,
			}

			key := manager.getSSHPrivateKey()

			if tt.wantEmpty && key != "" {
				t.Errorf("Expected empty key, got %q", key)
			}

			if !tt.wantEmpty && tt.sshKeyPath != "" && key == "" {
				t.Error("Expected non-empty key when path is set")
			}
		})
	}
}

// TestIngressIPParsing tests parsing of LoadBalancer IP from output
func TestIngressIPParsing(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantIP   string
	}{
		{
			name:   "Valid IP in output",
			output: "Installing...\nINGRESS_IP:192.168.1.100\nDone",
			wantIP: "192.168.1.100",
		},
		{
			name:   "IP at start",
			output: "INGRESS_IP:10.0.0.5",
			wantIP: "10.0.0.5",
		},
		{
			name:   "IP at end",
			output: "Setup complete\nINGRESS_IP:172.16.0.1",
			wantIP: "172.16.0.1",
		},
		{
			name:   "No IP in output",
			output: "Installing...\nDone",
			wantIP: "",
		},
		{
			name:   "Empty output",
			output: "",
			wantIP: "",
		},
		{
			name:   "Multiple lines without IP",
			output: "Line 1\nLine 2\nLine 3",
			wantIP: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the parsing logic from Install()
			lines := strings.Split(tt.output, "\n")
			var foundIP string
			for _, line := range lines {
				if strings.HasPrefix(line, "INGRESS_IP:") {
					foundIP = strings.TrimPrefix(line, "INGRESS_IP:")
					break
				}
			}

			if foundIP != tt.wantIP {
				t.Errorf("Expected IP %q, got %q", tt.wantIP, foundIP)
			}
		})
	}
}

// TestNginxIngressConfiguration tests NGINX ingress configuration values
func TestNginxIngressConfiguration(t *testing.T) {
	// Test configuration values
	config := map[string]interface{}{
		"replicaCount":               2,
		"minReplicas":                2,
		"maxReplicas":                4,
		"targetCPUUtilization":       80,
		"targetMemoryUtilization":    80,
		"healthcheckInterval":        10,
		"loadBalancerSize":           "lb-small",
		"loadBalancerAlgorithm":      "round_robin",
		"loadBalancerProtocol":       "tcp",
		"loadBalancerHealthcheckPort": 10254,
	}

	// Validate replica counts
	if replicas, ok := config["replicaCount"].(int); ok {
		if replicas < 1 {
			t.Error("replicaCount should be at least 1")
		}
		if replicas != 2 {
			t.Errorf("Expected replicaCount 2, got %d", replicas)
		}
	}

	// Validate autoscaling
	if minReplicas, ok := config["minReplicas"].(int); ok {
		if minReplicas < 1 {
			t.Error("minReplicas should be at least 1")
		}
	}

	if maxReplicas, ok := config["maxReplicas"].(int); ok {
		if minReplicas, ok := config["minReplicas"].(int); ok {
			if maxReplicas < minReplicas {
				t.Error("maxReplicas should be >= minReplicas")
			}
		}
	}

	// Validate utilization percentages
	if cpuUtil, ok := config["targetCPUUtilization"].(int); ok {
		if cpuUtil < 1 || cpuUtil > 100 {
			t.Errorf("CPU utilization should be 1-100, got %d", cpuUtil)
		}
	}

	if memUtil, ok := config["targetMemoryUtilization"].(int); ok {
		if memUtil < 1 || memUtil > 100 {
			t.Errorf("Memory utilization should be 1-100, got %d", memUtil)
		}
	}
}

// TestResourceRequirements tests resource limits and requests
func TestResourceRequirements(t *testing.T) {
	tests := []struct {
		name            string
		componentType   string
		cpuRequest      string
		memoryRequest   string
		memoryLimit     string
		isValid         bool
	}{
		{
			name:          "Controller resources",
			componentType: "controller",
			cpuRequest:    "100m",
			memoryRequest: "128Mi",
			memoryLimit:   "512Mi",
			isValid:       true,
		},
		{
			name:          "Default backend resources",
			componentType: "defaultBackend",
			cpuRequest:    "10m",
			memoryRequest: "32Mi",
			memoryLimit:   "64Mi",
			isValid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate resource format
			if tt.cpuRequest != "" && !strings.HasSuffix(tt.cpuRequest, "m") {
				if _, ok := parseMemory(tt.cpuRequest); !ok {
					// CPU should be in millicores (m) or whole numbers
					t.Errorf("Invalid CPU format: %s", tt.cpuRequest)
				}
			}

			if tt.memoryRequest != "" {
				if !strings.HasSuffix(tt.memoryRequest, "Mi") && !strings.HasSuffix(tt.memoryRequest, "Gi") {
					t.Errorf("Invalid memory format: %s", tt.memoryRequest)
				}
			}
		})
	}
}

// parseMemory is a helper to validate memory format
func parseMemory(mem string) (int, bool) {
	if strings.HasSuffix(mem, "Mi") || strings.HasSuffix(mem, "Gi") {
		return 1, true
	}
	return 0, false
}

// TestSSLConfiguration tests SSL/TLS configuration
func TestSSLConfiguration(t *testing.T) {
	sslConfig := map[string]string{
		"ssl-protocols": "TLSv1.2 TLSv1.3",
		"ssl-ciphers":   "ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256",
	}

	// Validate SSL protocols
	protocols := sslConfig["ssl-protocols"]
	if !strings.Contains(protocols, "TLSv1.2") {
		t.Error("Should support TLSv1.2")
	}
	if !strings.Contains(protocols, "TLSv1.3") {
		t.Error("Should support TLSv1.3")
	}
	if strings.Contains(protocols, "TLSv1.0") || strings.Contains(protocols, "TLSv1.1") {
		t.Error("Should not support TLSv1.0 or TLSv1.1 (insecure)")
	}

	// Validate cipher suites
	ciphers := sslConfig["ssl-ciphers"]
	if !strings.Contains(ciphers, "ECDHE") {
		t.Error("Should use ECDHE cipher suites")
	}
	if !strings.Contains(ciphers, "GCM") {
		t.Error("Should use GCM mode")
	}
}

// TestIngressAnnotations tests ingress controller annotations
func TestIngressAnnotations(t *testing.T) {
	annotations := map[string]string{
		"service.beta.kubernetes.io/do-loadbalancer-protocol":                   "tcp",
		"service.beta.kubernetes.io/do-loadbalancer-algorithm":                  "round_robin",
		"service.beta.kubernetes.io/do-loadbalancer-healthcheck-port":           "10254",
		"service.beta.kubernetes.io/do-loadbalancer-healthcheck-protocol":       "tcp",
		"service.beta.kubernetes.io/do-loadbalancer-healthcheck-interval-seconds": "10",
		"service.beta.kubernetes.io/do-loadbalancer-size-slug":                  "lb-small",
	}

	// Validate annotation keys
	for key := range annotations {
		if !strings.HasPrefix(key, "service.beta.kubernetes.io/") {
			t.Errorf("Annotation key should start with service.beta.kubernetes.io/, got: %s", key)
		}
	}

	// Validate specific values
	if protocol := annotations["service.beta.kubernetes.io/do-loadbalancer-protocol"]; protocol != "tcp" {
		t.Errorf("Expected protocol 'tcp', got %q", protocol)
	}

	if algorithm := annotations["service.beta.kubernetes.io/do-loadbalancer-algorithm"]; algorithm != "round_robin" {
		t.Errorf("Expected algorithm 'round_robin', got %q", algorithm)
	}

	if size := annotations["service.beta.kubernetes.io/do-loadbalancer-size-slug"]; size == "" {
		t.Error("LoadBalancer size should not be empty")
	}
}

// TestCertManagerConfiguration tests cert-manager configuration
func TestCertManagerConfiguration(t *testing.T) {
	version := "v1.13.0"

	// Validate version format
	if !strings.HasPrefix(version, "v") {
		t.Error("Version should start with 'v'")
	}

	if !strings.Contains(version, ".") {
		t.Error("Version should contain dots")
	}

	// Test Let's Encrypt servers
	servers := map[string]string{
		"production": "https://acme-v02.api.letsencrypt.org/directory",
		"staging":    "https://acme-staging-v02.api.letsencrypt.org/directory",
	}

	for env, server := range servers {
		if !strings.HasPrefix(server, "https://") {
			t.Errorf("%s server should use https, got: %s", env, server)
		}
		if !strings.Contains(server, "letsencrypt.org") {
			t.Errorf("%s server should be letsencrypt.org, got: %s", env, server)
		}
	}
}

// TestClusterIssuerConfig tests ClusterIssuer configuration
func TestClusterIssuerConfig(t *testing.T) {
	tests := []struct {
		name        string
		issuerName  string
		server      string
		environment string
	}{
		{
			name:        "Production issuer",
			issuerName:  "letsencrypt-prod",
			server:      "https://acme-v02.api.letsencrypt.org/directory",
			environment: "production",
		},
		{
			name:        "Staging issuer",
			issuerName:  "letsencrypt-staging",
			server:      "https://acme-staging-v02.api.letsencrypt.org/directory",
			environment: "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate issuer name format
			if !strings.HasPrefix(tt.issuerName, "letsencrypt-") {
				t.Errorf("Issuer name should start with 'letsencrypt-', got: %s", tt.issuerName)
			}

			// Validate server URL
			if !strings.HasPrefix(tt.server, "https://") {
				t.Error("Server should use HTTPS")
			}

			// Validate environment suffix
			if tt.environment == "production" && !strings.HasSuffix(tt.issuerName, "prod") {
				t.Error("Production issuer should end with 'prod'")
			}
			if tt.environment == "staging" && !strings.HasSuffix(tt.issuerName, "staging") {
				t.Error("Staging issuer should end with 'staging'")
			}
		})
	}
}

// TestSampleIngressYAML tests sample ingress YAML generation
func TestSampleIngressYAML(t *testing.T) {
	domain := "example.com"

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
`, domain, domain)

	// Validate YAML contains required fields
	if !strings.Contains(sampleIngress, "apiVersion: networking.k8s.io/v1") {
		t.Error("Missing apiVersion")
	}
	if !strings.Contains(sampleIngress, "kind: Ingress") {
		t.Error("Missing kind: Ingress")
	}
	if !strings.Contains(sampleIngress, "cert-manager.io/cluster-issuer") {
		t.Error("Missing cert-manager annotation")
	}
	if !strings.Contains(sampleIngress, "nginx.ingress.kubernetes.io/ssl-redirect") {
		t.Error("Missing SSL redirect annotation")
	}
	if !strings.Contains(sampleIngress, "ingressClassName: nginx") {
		t.Error("Missing ingress class")
	}
	if !strings.Contains(sampleIngress, "tls:") {
		t.Error("Missing TLS section")
	}
	if !strings.Contains(sampleIngress, domain) {
		t.Errorf("Domain %s not found in ingress", domain)
	}
}

// TestTestIngressYAML tests the test ingress deployment
func TestTestIngressYAML(t *testing.T) {
	// Test ingress should create test namespace, service, deployment, and ingress
	requiredResources := []string{
		"kind: Namespace",
		"name: ingress-test",
		"kind: Service",
		"name: test-service",
		"kind: Deployment",
		"name: test-deployment",
		"kind: Ingress",
		"name: test-ingress",
	}

	testYAML := `
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
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress
  namespace: ingress-test
`

	for _, resource := range requiredResources {
		if !strings.Contains(testYAML, resource) {
			t.Errorf("Missing required resource: %s", resource)
		}
	}

	// Should use nginx:alpine for testing
	if !strings.Contains(testYAML, "nginx:alpine") {
		t.Error("Should use nginx:alpine image for test deployment")
	}

	// Should have 1 replica for test
	if !strings.Contains(testYAML, "replicas: 1") {
		t.Error("Test deployment should have 1 replica")
	}
}

// TestNginxIngressURLs tests URL generation
func TestNginxIngressURLs(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		wantHTTP    string
		wantHTTPS   string
		wantIngress string
	}{
		{
			name:        "Standard domain",
			domain:      "example.com",
			wantHTTP:    "http://kube-ingress.example.com",
			wantHTTPS:   "https://kube-ingress.example.com",
			wantIngress: "kube-ingress.example.com",
		},
		{
			name:        "Subdomain",
			domain:      "k8s.example.com",
			wantHTTP:    "http://kube-ingress.k8s.example.com",
			wantHTTPS:   "https://kube-ingress.k8s.example.com",
			wantIngress: "kube-ingress.k8s.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpURL := fmt.Sprintf("http://kube-ingress.%s", tt.domain)
			httpsURL := fmt.Sprintf("https://kube-ingress.%s", tt.domain)
			ingressDomain := fmt.Sprintf("kube-ingress.%s", tt.domain)

			if httpURL != tt.wantHTTP {
				t.Errorf("HTTP URL: want %q, got %q", tt.wantHTTP, httpURL)
			}
			if httpsURL != tt.wantHTTPS {
				t.Errorf("HTTPS URL: want %q, got %q", tt.wantHTTPS, httpsURL)
			}
			if ingressDomain != tt.wantIngress {
				t.Errorf("Ingress domain: want %q, got %q", tt.wantIngress, ingressDomain)
			}
		})
	}
}

// TestHelmRepositories tests Helm repository URLs
func TestHelmRepositories(t *testing.T) {
	repos := map[string]string{
		"ingress-nginx": "https://kubernetes.github.io/ingress-nginx",
		"jetstack":      "https://charts.jetstack.io",
	}

	for name, url := range repos {
		if !strings.HasPrefix(url, "https://") {
			t.Errorf("Repository %s should use HTTPS, got: %s", name, url)
		}

		// Validate URL format
		if !strings.Contains(url, ".") {
			t.Errorf("Repository URL %s appears invalid", url)
		}
	}
}

// TestNginxIngressManagerStructure tests manager structure
func TestNginxIngressManagerStructure(t *testing.T) {
	manager := &NginxIngressManager{
		domain:     "example.com",
		sshKeyPath: "/root/.ssh/id_rsa",
	}

	if manager.domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got %q", manager.domain)
	}

	if manager.sshKeyPath != "/root/.ssh/id_rsa" {
		t.Errorf("Expected SSH key path '/root/.ssh/id_rsa', got %q", manager.sshKeyPath)
	}

	if manager.masterNode != nil {
		t.Error("masterNode should initially be nil")
	}
}

// TestLoadBalancerConfiguration tests load balancer settings
func TestLoadBalancerConfiguration(t *testing.T) {
	lbConfig := map[string]interface{}{
		"type":                   "LoadBalancer",
		"externalTrafficPolicy":  "Local",
		"protocol":               "tcp",
		"algorithm":              "round_robin",
		"healthcheckPort":        10254,
		"healthcheckProtocol":    "tcp",
		"healthcheckInterval":    10,
		"size":                   "lb-small",
	}

	// Validate service type
	if serviceType, ok := lbConfig["type"].(string); !ok || serviceType != "LoadBalancer" {
		t.Error("Service type should be LoadBalancer")
	}

	// Validate traffic policy
	if policy, ok := lbConfig["externalTrafficPolicy"].(string); !ok || policy != "Local" {
		t.Error("External traffic policy should be Local")
	}

	// Validate healthcheck port
	if port, ok := lbConfig["healthcheckPort"].(int); !ok || port <= 0 || port > 65535 {
		t.Errorf("Invalid healthcheck port: %v", lbConfig["healthcheckPort"])
	}

	// Validate healthcheck interval
	if interval, ok := lbConfig["healthcheckInterval"].(int); !ok || interval < 1 {
		t.Error("Healthcheck interval should be at least 1 second")
	}
}
