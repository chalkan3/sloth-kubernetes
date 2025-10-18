# Quick Start Guide

## Prerequisites

- ✅ WireGuard VPN connected
- ✅ kubectl installed
- ✅ `~/.kube/config` configured

## Verify Cluster Access

```bash
# Check VPN connectivity
ping 10.8.0.10

# Verify kubectl access
kubectl get nodes

# Should show 6 nodes (3 masters + 3 workers) all Ready
```

## Common Tasks

### 1. Deploy an Application to Tools Workers

```bash
# Create deployment
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      nodeSelector:
        workload: tools
      tolerations:
      - key: workload
        operator: Equal
        value: tools
        effect: NoSchedule
      containers:
      - name: my-app
        image: nginx:latest
        ports:
        - containerPort: 80
EOF

# Check deployment
kubectl get pods -o wide
```

### 2. Expose Service via Ingress

```bash
# Create service
kubectl expose deployment my-app --port=80 --type=ClusterIP

# Create ingress
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app-ingress
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  ingressClassName: nginx
  rules:
  - host: myapp.kube.chalkan3.com.br
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-app
            port:
              number: 80
EOF

# Add DNS record (replace with actual IP of a worker)
doctl compute domain records create chalkan3.com.br \
  --record-type A \
  --record-name myapp.kube \
  --record-data 10.8.0.13 \
  --record-ttl 300

# Test access (wait ~30s for DNS propagation)
curl http://myapp.kube.chalkan3.com.br
```

### 3. Check Application Logs

```bash
# Get pod name
kubectl get pods

# Follow logs
kubectl logs -f <pod-name>

# View last 100 lines
kubectl logs <pod-name> --tail=100
```

### 4. Execute Commands in Pod

```bash
# Get shell in pod
kubectl exec -it <pod-name> -- /bin/bash

# Run single command
kubectl exec <pod-name> -- ls -la /app
```

### 5. Scale Application

```bash
# Scale up to 3 replicas
kubectl scale deployment my-app --replicas=3

# Scale down to 1 replica
kubectl scale deployment my-app --replicas=1
```

## Node Targeting

### Deploy to Tools Workers (worker-1, worker-2)

```yaml
nodeSelector:
  workload: tools
tolerations:
- key: workload
  operator: Equal
  value: tools
  effect: NoSchedule
```

### Deploy to Misc Worker (worker-3)

```yaml
nodeSelector:
  workload: misc
tolerations:
- key: workload
  operator: Equal
  value: misc
  effect: NoSchedule
```

## Access Services

### ArgoCD
```bash
# Web UI
open https://argocd.kube.chalkan3.com.br

# CLI login
argocd login argocd.kube.chalkan3.com.br \
  --username admin \
  --password w-13KcdiqsQwruLs \
  --insecure
```

### Kubernetes Dashboard (if installed)
```bash
kubectl proxy
open http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
```

## Troubleshooting Quick Tips

```bash
# Node not ready?
kubectl describe node <node-name>

# Pod not starting?
kubectl describe pod <pod-name>
kubectl logs <pod-name>

# Ingress not working?
kubectl get ingress
kubectl describe ingress <ingress-name>

# Check events
kubectl get events --sort-by='.lastTimestamp'
```

## SSH to Nodes

```bash
# Masters
ssh-k8s-master-1  # or: ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.10
ssh-k8s-master-2
ssh-k8s-master-3

# Workers
ssh-k8s-worker-1
ssh-k8s-worker-2
ssh-k8s-worker-3
```

## Important URLs

- **Kubernetes API:** https://api.chalkan3.com.br:6443
- **ArgoCD:** https://argocd.kube.chalkan3.com.br
- **Documentation:** `~/.projects/do-droplet-create/docs/`

## Get More Help

```bash
# Full documentation
cat ~/.projects/do-droplet-create/docs/README.md

# Deployment examples
ls ~/.projects/do-droplet-create/docs/examples/

# Cluster info
kubectl cluster-info
```
