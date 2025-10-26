# Production Kubernetes Cluster - Complete Documentation

## 📋 Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Infrastructure](#infrastructure)
4. [Networking & VPN](#networking--vpn)
5. [Node Configuration](#node-configuration)
6. [Deployed Services](#deployed-services)
7. [Access & Authentication](#access--authentication)
8. [Deployment Guide](#deployment-guide)
9. [Maintenance & Operations](#maintenance--operations)
10. [Troubleshooting](#troubleshooting)

---

## Overview

**Cluster Name:** `production`
**Kubernetes Distribution:** RKE2 v1.33.5
**Container Runtime:** containerd 2.1.4-k3s2
**Infrastructure as Code:** Pulumi (Go)
**Cloud Providers:** DigitalOcean + Linode (Multi-cloud)
**Network:** WireGuard VPN Mesh
**API Endpoint:** https://api.chalkan3.com.br:6443

### Key Features

- ✅ **High Availability**: 3 master nodes with etcd cluster
- ✅ **Multi-cloud**: Nodes distributed across DigitalOcean and Linode
- ✅ **Private Network**: Full WireGuard VPN mesh between all nodes
- ✅ **Secure**: No public exposure, VPN-only access
- ✅ **Automated**: Provisioned via Pulumi with automatic dependency installation
- ✅ **Production Ready**: ArgoCD for GitOps, Nginx Ingress, SSL/TLS

---

## Architecture

### Cluster Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                     Control Plane (HA)                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  master-1   │  │  master-2   │  │  master-3   │            │
│  │ 10.8.0.10   │  │ 10.8.0.11   │  │ 10.8.0.12   │            │
│  │ DigitalOcean│  │   Linode    │  │   Linode    │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ WireGuard VPN Mesh
                              │
┌─────────────────────────────────────────────────────────────────┐
│                        Worker Nodes                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  worker-1   │  │  worker-2   │  │  worker-3   │            │
│  │ 10.8.0.13   │  │ 10.8.0.14   │  │ 10.8.0.15   │            │
│  │ (tools)     │  │ (tools)     │  │  (misc)     │            │
│  │DigitalOcean │  │DigitalOcean │  │   Linode    │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

### Network Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    VPN Server (AWS)                              │
│                     10.8.0.1                                     │
│                   3.93.242.31:51820                              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ WireGuard VPN
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   Your Client      K8s Master Nodes    K8s Worker Nodes
   10.8.0.2         10.8.0.10-12        10.8.0.13-15
```

---

## Infrastructure

### Node Specifications

#### Master Nodes (3x)

| Node      | Provider      | VPN IP     | Public IP       | Size/Type      | OS            | Location    |
|-----------|---------------|------------|-----------------|----------------|---------------|-------------|
| master-1  | DigitalOcean  | 10.8.0.10  | 159.203.130.45  | s-2vcpu-4gb    | Ubuntu 22.04  | NYC3        |
| master-2  | Linode        | 10.8.0.11  | 96.126.106.67   | g6-standard-2  | Ubuntu 22.04  | US-East     |
| master-3  | Linode        | 10.8.0.12  | 96.126.106.52   | g6-standard-2  | Ubuntu 22.04  | US-East     |

**Resources per Master:**
- 2 vCPUs
- 4 GB RAM
- 80 GB SSD
- Roles: `control-plane`, `etcd`, `master`

#### Worker Nodes (3x)

| Node      | Provider      | VPN IP     | Public IP        | Size/Type      | Workload | OS            | Location    |
|-----------|---------------|------------|------------------|----------------|----------|---------------|-------------|
| worker-1  | DigitalOcean  | 10.8.0.13  | 138.197.68.229   | s-2vcpu-4gb    | tools    | Ubuntu 22.04  | NYC3        |
| worker-2  | DigitalOcean  | 10.8.0.14  | 165.227.219.199  | s-2vcpu-4gb    | tools    | Ubuntu 22.04  | NYC3        |
| worker-3  | Linode        | 10.8.0.15  | 96.126.106.49    | g6-standard-2  | misc     | Ubuntu 22.04  | US-East     |

**Resources per Worker:**
- 2 vCPUs
- 4 GB RAM
- 80 GB SSD

### Node Labels & Taints

#### Tools Workers (worker-1, worker-2)
```yaml
Labels:
  workload: tools

Taints:
  - key: workload
    value: tools
    effect: NoSchedule
```

**Purpose:** Dedicated to development tools (ArgoCD, CI/CD, monitoring, etc.)

#### Misc Worker (worker-3)
```yaml
Labels:
  workload: misc

Taints:
  - key: workload
    value: misc
    effect: NoSchedule
```

**Purpose:** Miscellaneous workloads and experimental deployments

---

## Networking & VPN

### WireGuard VPN Configuration

All nodes are connected via a full-mesh WireGuard VPN:

- **VPN Server:** 3.93.242.31 (AWS EC2 - Amazon Linux 2023)
- **VPN Network:** 10.8.0.0/24
- **VPN Interface:** wg0
- **Port:** 51820 (UDP)
- **Encryption:** ChaCha20-Poly1305

#### VPN IP Allocation

| IP Range       | Purpose                    |
|----------------|----------------------------|
| 10.8.0.1       | VPN Server                 |
| 10.8.0.2       | Your client machine        |
| 10.8.0.10-12   | Kubernetes master nodes    |
| 10.8.0.13-15   | Kubernetes worker nodes    |

### Network Topology

- **Full Mesh:** Each Kubernetes node has direct WireGuard tunnels to:
  - All other Kubernetes nodes (5 peers)
  - VPN server (1 peer)
  - Total: 6 WireGuard peers per node

- **Persistent Keepalive:** 25 seconds (maintains NAT traversal)
- **Allowed Networks:**
  - `10.8.0.0/24` - VPN network
  - `10.0.0.0/16` - Additional private network
  - `10.11.0.0/16` - Reserved range
  - `10.20.0.0/24`, `10.21.0.0/24`, `10.22.0.0/24` - Application subnets

### DNS Configuration

All cluster DNS records point to VPN private IPs:

| DNS Record                    | IP         | Purpose          |
|-------------------------------|------------|------------------|
| api.chalkan3.com.br          | 10.8.0.10  | Kubernetes API   |
| argocd.kube.chalkan3.com.br  | 10.8.0.13  | ArgoCD UI        |

---

## Node Configuration

### Pre-installed Software

All nodes come with the following pre-installed:

- **Docker** (via get.docker.sh)
- **WireGuard** + wireguard-tools
- **Kernel Modules:** br_netfilter, overlay
- **NFS Client** (for persistent volumes)
- **sysctl Configurations:**
  ```
  net.bridge.bridge-nf-call-iptables = 1
  net.bridge.bridge-nf-call-ip6tables = 1
  net.ipv4.ip_forward = 1
  ```

### SSH Access

**SSH Key:** `~/.ssh/kubernetes-clusters/production.pem`

```bash
# Access master nodes
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.10  # master-1
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.11  # master-2
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.12  # master-3

# Access worker nodes
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.13  # worker-1
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.14  # worker-2
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.15  # worker-3
```

**Shell Shortcuts:**
```bash
# These functions are available in ~/.zsh/functions.zsh
ssh-k8s-master-1   # SSH to master-1
ssh-k8s-master-2   # SSH to master-2
ssh-k8s-master-3   # SSH to master-3
ssh-k8s-worker-1   # SSH to worker-1
ssh-k8s-worker-2   # SSH to worker-2
ssh-k8s-worker-3   # SSH to worker-3

# List all VPN hosts
vpn-hosts
```

---

## Deployed Services

### System Services

#### Calico CNI
- **Namespace:** `calico-system`, `tigera-operator`
- **Purpose:** Pod networking and network policies
- **CIDR:** Automatic IPAM

#### Nginx Ingress Controller
- **Namespace:** `ingress-nginx`
- **Version:** v1.11.1
- **Type:** DaemonSet with hostPort
- **Ports:**
  - HTTP: 80 (hostPort)
  - HTTPS: 443 (hostPort)
- **Features:**
  - SSL Passthrough: ✅ Enabled
  - Default Backend: ✅
  - Admission Webhook: ✅

### Application Services

#### ArgoCD
- **Namespace:** `argocd`
- **Version:** v3.1.9
- **Access:** https://argocd.kube.chalkan3.com.br
- **Service Type:** ClusterIP (VPN-only access)
- **Ingress:** Nginx with SSL passthrough
- **Authentication:**
  - Username: `admin`
  - Password: `w-13KcdiqsQwruLs`

**ArgoCD CLI:**
```bash
# Login
argocd login argocd.kube.chalkan3.com.br \
  --username admin \
  --password w-13KcdiqsQwruLs \
  --insecure

# Check status
argocd app list
```

---

## Access & Authentication

### Kubernetes API Access

**Kubeconfig Location:** `~/.kube/config`

**API Server:** https://api.chalkan3.com.br:6443

```bash
# Verify access
kubectl cluster-info
kubectl get nodes
kubectl get pods --all-namespaces
```

### VPN Connection

**VPN Client Configuration:**
```bash
# Check VPN status
sudo wg show

# VPN should show:
# - Interface: utun4
# - Your IP: 10.8.0.2
# - Peer: VPN Server (3.93.242.31)
# - Allowed IPs: 10.8.0.0/24, 10.0.0.0/16, etc.
```

**Testing VPN Connectivity:**
```bash
# Ping Kubernetes nodes
ping 10.8.0.10  # master-1
ping 10.8.0.13  # worker-1

# Should see ~140ms latency (via VPN gateway)
```

### Service Access

All services are accessible **only via VPN**:

| Service          | URL/Endpoint                            | Port | Protocol |
|------------------|-----------------------------------------|------|----------|
| Kubernetes API   | https://api.chalkan3.com.br:6443        | 6443 | HTTPS    |
| ArgoCD UI        | https://argocd.kube.chalkan3.com.br     | 443  | HTTPS    |
| kubectl          | Via kubeconfig                          | 6443 | HTTPS    |

---

## Deployment Guide

### Deploying to Tools Workers

For services that should run on `tools` workers (worker-1, worker-2):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-tool
  namespace: tools
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-tool
  template:
    metadata:
      labels:
        app: my-tool
    spec:
      # Node selector to target tools workers
      nodeSelector:
        workload: tools

      # Toleration to allow scheduling on tainted nodes
      tolerations:
      - key: workload
        operator: Equal
        value: tools
        effect: NoSchedule

      containers:
      - name: my-tool
        image: my-tool:latest
        ports:
        - containerPort: 8080
```

### Deploying to Misc Worker

For miscellaneous workloads (worker-3):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-misc-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-misc-app
  template:
    metadata:
      labels:
        app: my-misc-app
    spec:
      # Node selector to target misc worker
      nodeSelector:
        workload: misc

      # Toleration for misc taint
      tolerations:
      - key: workload
        operator: Equal
        value: misc
        effect: NoSchedule

      containers:
      - name: my-misc-app
        image: my-app:latest
        ports:
        - containerPort: 3000
```

### Exposing Services via Ingress

Create an Ingress to expose your service via HTTPS:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-service-ingress
  namespace: tools
  annotations:
    # Use Nginx ingress controller
    kubernetes.io/ingress.class: nginx
    # For HTTPS backends
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    # For SSL passthrough (if needed)
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
spec:
  ingressClassName: nginx
  rules:
  - host: myservice.kube.chalkan3.com.br
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 443
```

**Don't forget to add DNS record:**
```bash
doctl compute domain records create chalkan3.com.br \
  --record-type A \
  --record-name myservice.kube \
  --record-data 10.8.0.13 \
  --record-ttl 300
```

---

## Maintenance & Operations

### Cluster Management

#### Check Cluster Health
```bash
# Node status
kubectl get nodes -o wide

# Pod status across all namespaces
kubectl get pods --all-namespaces

# Component status
kubectl get componentstatuses

# Cluster info
kubectl cluster-info
```

#### View Node Resources
```bash
# Node resource usage
kubectl top nodes

# Pod resource usage
kubectl top pods --all-namespaces
```

#### Drain a Node (for maintenance)
```bash
# Safely evict all pods from a node
kubectl drain worker-1 --ignore-daemonsets --delete-emptydir-data

# Mark node as schedulable again
kubectl uncordon worker-1
```

### Pulumi Operations

#### View Stack Outputs
```bash
cd ~/.projects/do-droplet-create

# List all outputs
sloth-kubernetes pulumi stack output

# Get kubeconfig
sloth-kubernetes pulumi stack output kubeConfig --show-secrets

# Get SSH key
sloth-kubernetes pulumi stack output ssh_private_key --show-secrets

# Get connection instructions
sloth-kubernetes pulumi stack output connectionInstructions
```

#### Update Infrastructure
```bash
# Preview changes
sloth-kubernetes pulumi preview

# Apply changes
sloth-kubernetes deploy

# Destroy entire cluster (⚠️ DANGEROUS)
sloth-kubernetes destroy
```

### Backup & Recovery

#### Backup etcd
```bash
# SSH to any master node
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.10

# Create etcd snapshot
sudo rke2 etcd-snapshot save --name manual-backup-$(date +%Y%m%d-%H%M%S)

# List snapshots
sudo rke2 etcd-snapshot list
```

#### Backup Kubeconfig
```bash
# Local backup
cp ~/.kube/config ~/.kube/config.backup-$(date +%Y%m%d)

# From Pulumi
sloth-kubernetes pulumi stack output kubeConfig --show-secrets > ~/kubeconfig-backup-$(date +%Y%m%d).yaml
```

### Monitoring & Logs

#### View Pod Logs
```bash
# Follow logs
kubectl logs -f <pod-name> -n <namespace>

# Previous container logs (if crashed)
kubectl logs <pod-name> -n <namespace> --previous

# All pods in deployment
kubectl logs -l app=my-app -n tools --tail=100
```

#### View Events
```bash
# Cluster-wide events
kubectl get events --all-namespaces --sort-by='.lastTimestamp'

# Events in specific namespace
kubectl get events -n argocd
```

#### Describe Resources
```bash
# Detailed node info
kubectl describe node worker-1

# Detailed pod info
kubectl describe pod <pod-name> -n <namespace>
```

---

## Troubleshooting

### Common Issues

#### 1. Cannot Access Cluster
**Symptoms:** `kubectl` commands timeout or fail

**Troubleshooting:**
```bash
# Check VPN connectivity
sudo wg show
ping 10.8.0.10

# Verify kubeconfig
kubectl config current-context
kubectl config view

# Test API server
curl -k https://api.chalkan3.com.br:6443
```

**Solution:**
- Ensure VPN is connected
- Verify `~/.kube/config` is correct
- Check VPN allowed IPs include `10.8.0.0/24`

#### 2. Pods Not Scheduling
**Symptoms:** Pods stuck in `Pending` state

**Troubleshooting:**
```bash
# Check pod status
kubectl describe pod <pod-name> -n <namespace>

# Check node taints
kubectl describe node worker-1 | grep Taints

# Check pod tolerations
kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A5 tolerations
```

**Solution:**
- Add correct `nodeSelector` and `tolerations` to pod spec
- Verify worker nodes have capacity
- Check if taints are preventing scheduling

#### 3. Ingress Not Working
**Symptoms:** Cannot access service via domain

**Troubleshooting:**
```bash
# Check Ingress status
kubectl get ingress -n <namespace>
kubectl describe ingress <ingress-name> -n <namespace>

# Check Ingress Controller
kubectl get pods -n ingress-nginx

# Check DNS resolution
host myservice.kube.chalkan3.com.br

# Test direct access to NodePort
curl -k https://10.8.0.13:31328
```

**Solution:**
- Verify DNS points to correct VPN IP
- Check Ingress annotations are correct
- Ensure service is running: `kubectl get svc -n <namespace>`
- Verify Nginx Ingress Controller is healthy

#### 4. VPN Connectivity Issues
**Symptoms:** Cannot ping nodes at 10.8.0.x

**Troubleshooting:**
```bash
# Check WireGuard status
sudo wg show

# Check routes
netstat -rn | grep 10.8

# Ping VPN server
ping 10.8.0.1

# Check WireGuard on nodes
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.10 "sudo wg show"
```

**Solution:**
- Restart WireGuard: `sudo systemctl restart wg-quick@wg0` (on VPN server)
- Verify peers are configured on both sides
- Check firewall allows UDP port 51820

#### 5. Node Not Ready
**Symptoms:** Node shows `NotReady` status

**Troubleshooting:**
```bash
# Check node status
kubectl describe node <node-name>

# SSH to node and check kubelet
ssh -i ~/.ssh/kubernetes-clusters/production.pem root@<node-ip>
sudo systemctl status rke2-server  # for masters
sudo systemctl status rke2-agent   # for workers

# Check logs
sudo journalctl -u rke2-agent -f
```

**Solution:**
- Restart RKE2 service
- Check disk space: `df -h`
- Verify network connectivity
- Check container runtime: `sudo crictl ps`

### Getting Help

**Useful Commands:**
```bash
# Cluster diagnostics
kubectl cluster-info dump > cluster-dump.txt

# Node diagnostics
kubectl get nodes -o yaml > nodes.yaml

# Pod diagnostics
kubectl get pods --all-namespaces -o wide > pods.txt

# Events
kubectl get events --all-namespaces > events.txt
```

**Log Locations on Nodes:**
- RKE2 Server: `/var/lib/rancher/rke2/agent/logs/`
- Kubelet: `journalctl -u rke2-agent`
- Container logs: `sudo crictl logs <container-id>`

---

## Security Considerations

### Network Security
- ✅ All cluster communication via WireGuard VPN (encrypted)
- ✅ No public exposure of services
- ✅ DNS points to private IPs only
- ✅ Ingress Controller uses SSL/TLS

### Access Control
- ✅ SSH key-based authentication only
- ✅ Kubernetes RBAC enabled
- ✅ ArgoCD with authentication required
- ✅ API server certificate-based auth

### Best Practices
1. **Never commit secrets to Git** - Use ArgoCD with sealed secrets or external secret management
2. **Rotate credentials regularly** - Change ArgoCD admin password, regenerate SSH keys
3. **Monitor cluster activity** - Review logs and events regularly
4. **Keep nodes updated** - Apply security patches to Ubuntu and Kubernetes
5. **Backup regularly** - Automated etcd snapshots and config backups

---

## Quick Reference

### Essential Commands
```bash
# Cluster status
kubectl get nodes
kubectl get pods --all-namespaces

# Deploy application
kubectl apply -f deployment.yaml

# Check logs
kubectl logs -f <pod-name> -n <namespace>

# Execute in pod
kubectl exec -it <pod-name> -n <namespace> -- /bin/bash

# Port forward
kubectl port-forward -n <namespace> svc/<service> 8080:80

# Scale deployment
kubectl scale deployment <name> --replicas=3 -n <namespace>

# Delete resources
kubectl delete -f deployment.yaml
```

### Important Paths
- **Kubeconfig:** `~/.kube/config`
- **SSH Key:** `~/.ssh/kubernetes-clusters/production.pem`
- **Pulumi Project:** `~/.projects/do-droplet-create`
- **Shell Functions:** `~/.zsh/functions.zsh`

### Important URLs
- **API Server:** https://api.chalkan3.com.br:6443
- **ArgoCD:** https://argocd.kube.chalkan3.com.br
- **VPN Server:** 3.93.242.31 (10.8.0.1)

---

## Appendix

### Cluster Specifications Summary

| Component           | Details                                    |
|---------------------|--------------------------------------------|
| Kubernetes Version  | v1.33.5+rke2r1                             |
| Distribution        | RKE2                                       |
| Container Runtime   | containerd 2.1.4-k3s2                      |
| CNI                 | Calico                                     |
| Ingress             | Nginx v1.11.1                              |
| GitOps              | ArgoCD v3.1.9                              |
| Total Nodes         | 6 (3 masters + 3 workers)                  |
| Total vCPUs         | 12 vCPUs                                   |
| Total RAM           | 24 GB                                      |
| Total Storage       | ~480 GB SSD                                |
| Cloud Providers     | DigitalOcean (3 nodes) + Linode (3 nodes)  |
| Network             | WireGuard VPN full-mesh                    |

### Change Log

| Date       | Change                                              | Author  |
|------------|-----------------------------------------------------|---------|
| 2025-10-18 | Initial cluster deployment                          | Claude  |
| 2025-10-18 | ArgoCD installation and configuration               | Claude  |
| 2025-10-18 | VPN mesh configuration with all nodes               | Claude  |
| 2025-10-18 | Worker node labels and taints (tools/misc)          | Claude  |
| 2025-10-18 | Nginx Ingress Controller with hostPort 80/443       | Claude  |

---

**Documentation Version:** 1.0
**Last Updated:** October 18, 2025
**Maintained By:** Infrastructure Team
