# Documentation Index

Welcome to the complete documentation for the Production Kubernetes Cluster.

## 📚 Documentation Structure

```
docs/
├── README.md                  # Complete cluster documentation (main reference)
├── QUICK_START.md            # Quick start guide for common tasks
├── NETWORK_DIAGRAM.md        # Network architecture diagrams
├── INDEX.md                  # This file
└── examples/
    ├── deployment-tools.yaml        # Deploy to tools workers
    ├── deployment-misc.yaml         # Deploy to misc worker
    └── statefulset-with-pvc.yaml   # Stateful app with storage
```

## 🚀 Start Here

**New to the cluster?**
1. Read [QUICK_START.md](QUICK_START.md) for immediate productivity
2. Explore [examples/](examples/) for deployment templates
3. Reference [README.md](README.md) for comprehensive details

**Need specific information?**
- Architecture & Infrastructure → [README.md#architecture](README.md#architecture)
- Network & VPN → [NETWORK_DIAGRAM.md](NETWORK_DIAGRAM.md)
- Deployment Examples → [examples/](examples/)
- Troubleshooting → [README.md#troubleshooting](README.md#troubleshooting)

## 📖 Documentation Guide

### [README.md](README.md) - Complete Reference
**Sections:**
- Overview & Key Features
- Architecture Topology
- Infrastructure Specifications
- Node Configuration (Labels & Taints)
- Networking & VPN (WireGuard Mesh)
- Deployed Services (ArgoCD, Nginx Ingress)
- Access & Authentication
- Deployment Guide
- Maintenance & Operations
- Troubleshooting
- Security Considerations
- Quick Reference
- Appendix

**Use when:** You need detailed information about any aspect of the cluster

### [QUICK_START.md](QUICK_START.md) - Quick Reference
**Contents:**
- Prerequisites checklist
- Cluster access verification
- Common tasks with commands
- Node targeting examples
- Service access URLs
- SSH shortcuts
- Troubleshooting tips

**Use when:** You need to quickly accomplish a specific task

### [NETWORK_DIAGRAM.md](NETWORK_DIAGRAM.md) - Network Architecture
**Diagrams:**
- Complete network topology
- Kubernetes cluster detail
- Traffic flow (external → services)
- kubectl → API flow
- WireGuard peer connections
- DNS resolution flow
- Port reference table

**Use when:** You need to understand network connectivity and routing

### [examples/](examples/) - Deployment Templates

#### [deployment-tools.yaml](examples/deployment-tools.yaml)
- Deploy applications to tools workers (worker-1, worker-2)
- Includes: Deployment, Service, Ingress
- Use for: CI/CD tools, monitoring, development tools

#### [deployment-misc.yaml](examples/deployment-misc.yaml)
- Deploy applications to misc worker (worker-3)
- Includes: Deployment, Service, Ingress
- Use for: Experimental apps, background jobs, testing

#### [statefulset-with-pvc.yaml](examples/statefulset-with-pvc.yaml)
- Deploy stateful applications with persistent storage
- Includes: StatefulSet, PVC, Storage Class
- Use for: Databases, message queues, data persistence

## 🎯 Common Use Cases

### "I want to deploy a new service"
1. Copy template from [examples/deployment-tools.yaml](examples/deployment-tools.yaml)
2. Modify image, labels, and ingress hostname
3. Apply with `kubectl apply -f`
4. Add DNS record with `doctl`
5. Access via https://yourservice.kube.chalkan3.com.br

### "I need to understand the network"
1. Read [NETWORK_DIAGRAM.md](NETWORK_DIAGRAM.md)
2. Check [README.md#networking--vpn](README.md#networking--vpn)
3. Test connectivity: `ping 10.8.0.10`

### "Something is not working"
1. Check [QUICK_START.md#troubleshooting-quick-tips](QUICK_START.md#troubleshooting-quick-tips)
2. Reference [README.md#troubleshooting](README.md#troubleshooting)
3. Check logs: `kubectl logs <pod> -n <namespace>`
4. Review events: `kubectl get events --sort-by='.lastTimestamp'`

### "I want to SSH to a node"
1. Use shortcuts: `ssh-k8s-worker-1` (from zsh functions)
2. Or: `ssh -i ~/.ssh/kubernetes-clusters/production.pem root@10.8.0.13`
3. List all hosts: `vpn-hosts`

## 📊 Cluster Overview

```
┌─────────────────────────────────────────────────┐
│         Production Kubernetes Cluster            │
│                                                   │
│  Masters: 3 (HA)    Workers: 3 (2 tools + 1 misc)│
│  Kubernetes: v1.33.5+rke2r1                       │
│  Network: WireGuard VPN Mesh (10.8.0.0/24)        │
│  Providers: DigitalOcean + Linode (Multi-cloud)  │
│                                                   │
│  Services:                                        │
│  - ArgoCD v3.1.9 (GitOps)                        │
│  - Nginx Ingress v1.11.1                         │
│  - Calico CNI                                    │
│                                                   │
│  Access: VPN-only (No public exposure)           │
│  API: https://api.chalkan3.com.br:6443           │
└─────────────────────────────────────────────────┘
```

## 🔗 Quick Links

| Resource                | URL / Command                              |
|-------------------------|--------------------------------------------|
| Kubernetes API          | https://api.chalkan3.com.br:6443           |
| ArgoCD UI               | https://argocd.kube.chalkan3.com.br        |
| Cluster Status          | `kubectl get nodes`                        |
| All Pods                | `kubectl get pods --all-namespaces`        |
| ArgoCD Login            | `argocd login argocd.kube.chalkan3.com.br` |
| SSH to Master-1         | `ssh-k8s-master-1`                         |
| SSH to Worker-1         | `ssh-k8s-worker-1`                         |
| VPN Hosts List          | `vpn-hosts`                                |
| Pulumi Project          | `cd ~/.projects/do-droplet-create`         |

## 🆘 Need Help?

**Quick diagnostics:**
```bash
# Cluster health
kubectl get nodes
kubectl get pods --all-namespaces

# VPN connectivity
ping 10.8.0.10

# Service status
kubectl get svc --all-namespaces

# Recent events
kubectl get events --all-namespaces --sort-by='.lastTimestamp' | tail -20
```

**Documentation feedback:**
If you find errors or have suggestions, update the docs in:
`~/.projects/do-droplet-create/docs/`

---

**Last Updated:** October 18, 2025
**Version:** 1.0
