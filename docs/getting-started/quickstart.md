---
layout: default
title: Quick Start
parent: Getting Started
nav_order: 2
---

# Quick Start Guide

This guide will help you deploy your first multi-cloud Kubernetes cluster with sloth-kubernetes in under 10 minutes.

## Prerequisites

- sloth-kubernetes installed ([Installation Guide](installation))
- Cloud provider account(s):
  - DigitalOcean (with API token)
  - Linode (with API token)
  - Azure (with credentials)
- SSH key pair

## Step 1: Create Configuration File

Create a file named `cluster.yaml`:

```yaml
apiVersion: v1
kind: Cluster
metadata:
  name: my-first-cluster
  
spec:
  # Cloud providers
  providers:
    - type: digitalocean
      region: nyc3
      token: ${DIGITALOCEAN_TOKEN}
    
  # Node pools
  nodePools:
    - name: masters
      role: master
      count: 3
      size: s-2vcpu-4gb
      provider: digitalocean
      
    - name: workers
      role: worker
      count: 2
      size: s-2vcpu-4gb
      provider: digitalocean
  
  # Networking
  networking:
    vpnMesh: true
    cni: calico
    
  # Security
  security:
    bastion:
      enabled: true
      size: s-1vcpu-1gb
```

## Step 2: Set Environment Variables

```bash
export DIGITALOCEAN_TOKEN="your-do-token-here"
```

## Step 3: Validate Configuration

```bash
sloth-kubernetes validate --config cluster.yaml
```

## Step 4: Deploy Cluster

```bash
sloth-kubernetes deploy --config cluster.yaml
```

This will:
1. ✅ Generate and upload SSH keys
2. ✅ Create bastion host
3. ✅ Create VPCs in each provider
4. ✅ Setup WireGuard VPN mesh
5. ✅ Provision nodes
6. ✅ Install RKE2 Kubernetes
7. ✅ Configure VPN on nodes
8. ✅ Setup DNS records

Expected time: 5-8 minutes.

## Step 5: Verify Deployment

```bash
# Check cluster status
sloth-kubernetes status

# Get nodes
sloth-kubernetes kubectl get nodes

# Get all pods
sloth-kubernetes kubectl get pods -A
```

## Step 6: Deploy a Test Application

```bash
# Create a namespace
sloth-kubernetes kubectl create namespace demo

# Deploy nginx
sloth-kubernetes kubectl create deployment nginx --image=nginx -n demo

# Expose it
sloth-kubernetes kubectl expose deployment nginx --port=80 --type=LoadBalancer -n demo

# Check the service
sloth-kubernetes kubectl get svc -n demo
```

## Step 7: Use Salt for Node Management

```bash
# Check node status
sloth-kubernetes salt cmd "systemctl status kubelet" --target "worker*"

# Update packages
sloth-kubernetes salt pkg.upgrade --target "all"

# Docker operations
sloth-kubernetes salt docker.ps --target "worker*"
```

## Step 8: Use Helm (Optional)

```bash
# Add bitnami repo
sloth-kubernetes helm repo add bitnami https://charts.bitnami.com/bitnami

# Install a chart
sloth-kubernetes helm install myapp bitnami/nginx -n demo

# List releases
sloth-kubernetes helm list -A
```

## Step 9: Cleanup

When you're done testing:

```bash
sloth-kubernetes destroy --config cluster.yaml
```

{: .warning }
This will delete all resources. Make sure you've backed up any important data first.

## Next Steps

- [Configuration Guide](../user-guide/configuration) - Learn about all configuration options
- [Architecture](../architecture/overview) - Understand how sloth-kubernetes works
- [Examples](../examples/) - See more complex deployment scenarios
- [CLI Reference](../cli-reference/commands) - Explore all available commands

## Troubleshooting

### Deployment Failed

```bash
# Check detailed logs
sloth-kubernetes deploy --config cluster.yaml --log-level debug

# Check status
sloth-kubernetes status --verbose
```

### Can't Connect to Nodes

```bash
# SSH to bastion
sloth-kubernetes nodes ssh bastion

# From bastion, SSH to any node
ssh worker-1
```

### VPN Issues

```bash
# Check VPN status
sloth-kubernetes vpn status

# Test connectivity
sloth-kubernetes vpn test

# View peer configuration
sloth-kubernetes vpn peers
```
