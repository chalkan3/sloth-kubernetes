---
title: User Guide
description: Complete reference for all sloth-kubernetes commands and features
---

# User Guide

Complete guide to using **sloth-kubernetes** for deploying and managing multi-cloud Kubernetes clusters.

## CLI Command Categories

sloth-kubernetes provides **50+ commands** organized into categories:

### Cluster Lifecycle

```bash
# Deploy a new cluster
sloth-kubernetes deploy --config cluster.yaml

# Destroy cluster and all resources
sloth-kubernetes destroy --config cluster.yaml

# Refresh Pulumi state
sloth-kubernetes refresh --config cluster.yaml

# Check cluster status
sloth-kubernetes status

# Validate configuration
sloth-kubernetes validate --config cluster.yaml
```

### Node Management

```bash
# List all nodes
sloth-kubernetes nodes list

# SSH to a node (via bastion)
sloth-kubernetes nodes ssh <node-name>

# Add nodes to pool
sloth-kubernetes nodes add --pool workers --count 2

# Remove a node
sloth-kubernetes nodes remove <node-name>

# Drain node for maintenance
sloth-kubernetes nodes drain <node-name>

# Cordon node (prevent new pods)
sloth-kubernetes nodes cordon <node-name>

# Uncordon node
sloth-kubernetes nodes uncordon <node-name>
```

### Stack Operations (Multiple Clusters)

```bash
# List all stacks
sloth-kubernetes stacks list

# Show stack details
sloth-kubernetes stacks info

# Select active stack
sloth-kubernetes stacks select <stack-name>

# Delete a stack
sloth-kubernetes stacks delete <stack-name>

# Export stack state
sloth-kubernetes stacks export > stack-backup.json

# Import stack state
sloth-kubernetes stacks import stack-backup.json

# Show stack outputs
sloth-kubernetes stacks output
```

### SaltStack Operations

Over **100 remote execution modules**:

```bash
# Test connectivity
sloth-kubernetes salt ping

# List minions
sloth-kubernetes salt minions

# Run command on all nodes
sloth-kubernetes salt cmd.run "uptime"

# Run on specific node
sloth-kubernetes salt cmd.run "uptime" --target master-0

# Package management
sloth-kubernetes salt pkg.install htop
sloth-kubernetes salt pkg.remove htop
sloth-kubernetes salt pkg.upgrade

# Service management
sloth-kubernetes salt service.restart kubelet
sloth-kubernetes salt service.status kubelet
sloth-kubernetes salt service.enable nginx

# System information
sloth-kubernetes salt grains.items
sloth-kubernetes salt grains.get os
sloth-kubernetes salt status.diskusage

# State management
sloth-kubernetes salt state.apply
sloth-kubernetes salt state.highstate

# Key management
sloth-kubernetes salt keys list
sloth-kubernetes salt keys accept <minion-id>
sloth-kubernetes salt keys delete <minion-id>
```

### Kubernetes Tools

```bash
# Get kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Use embedded kubectl
sloth-kubernetes kubectl get nodes
sloth-kubernetes kubectl get pods -A
sloth-kubernetes kubectl apply -f manifest.yaml
sloth-kubernetes kubectl logs <pod-name>
sloth-kubernetes kubectl exec -it <pod-name> -- /bin/bash

# Helm operations
sloth-kubernetes helm install nginx bitnami/nginx
sloth-kubernetes helm upgrade nginx bitnami/nginx
sloth-kubernetes helm list
sloth-kubernetes helm uninstall nginx

# Kustomize
sloth-kubernetes kustomize build overlays/production
sloth-kubernetes kustomize build overlays/production | kubectl apply -f -
```

### GitOps & Addons

```bash
# Bootstrap ArgoCD
sloth-kubernetes addons bootstrap

# List available addons
sloth-kubernetes addons list

# Sync addons from Git
sloth-kubernetes addons sync

# Check addon status
sloth-kubernetes addons status

# Generate addon template
sloth-kubernetes addons template ingress-nginx > addons/ingress.yaml
```

### Authentication

```bash
# Login to cloud provider
sloth-kubernetes login digitalocean
sloth-kubernetes login linode
sloth-kubernetes login aws
```

## Configuration Reference

### Complete Cluster Configuration Example

```yaml
metadata:
  name: production-cluster
  region: nyc3
  labels:
    environment: production
    team: platform

providers:
  digitalocean:
    enabled: true
    token: ${DO_TOKEN}
    region: nyc3
    sshKeys:
      - "ssh-ed25519 AAAA..."

  linode:
    enabled: true
    token: ${LINODE_TOKEN}
    region: us-east
    sshKeys:
      - "ssh-ed25519 AAAA..."

  aws:
    enabled: false
    region: us-east-1
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}

network:
  vpcCIDR: 10.244.0.0/16
  serviceCIDR: 10.96.0.0/12
  podCIDR: 10.100.0.0/16
  wireguard:
    enabled: true
    port: 51820
    mtu: 1420

security:
  bastion:
    enabled: true
    provider: digitalocean
    size: s-1vcpu-1gb
    sshPort: 22
    mfa: true
    auditLogging: true
  
  firewall:
    enabled: true
    allowedSSHSources:
      - 0.0.0.0/0  # Restrict in production
    allowedAPIServerSources:
      - 0.0.0.0/0

kubernetes:
  version: "v1.28.2+rke2r1"
  distribution: rke2
  
  rke2:
    server:
      tlsSan:
        - api.example.com
      writeKubeconfigMode: "0644"
      etcdSnapshotScheduleCron: "0 */6 * * *"
      etcdSnapshotRetention: 10
    
    agent:
      kubeletArgs:
        - "max-pods=110"

nodePools:
  - name: do-masters
    provider: digitalocean
    role: master
    count: 3
    size: s-2vcpu-4gb
    region: nyc3
    labels:
      node-role.kubernetes.io/master: "true"
    taints:
      - key: node-role.kubernetes.io/master
        effect: NoSchedule

  - name: do-workers
    provider: digitalocean
    role: worker
    count: 5
    size: s-4vcpu-8gb
    region: nyc3
    labels:
      node-role.kubernetes.io/worker: "true"

  - name: linode-workers
    provider: linode
    role: worker
    count: 3
    size: g6-standard-4
    region: us-east

addons:
  argocd:
    enabled: true
    version: "v2.9.0"
    repository: "https://github.com/your-org/k8s-addons"
    path: "clusters/production"
    targetRevision: main
    syncPolicy:
      automated:
        prune: true
        selfHeal: true
      syncOptions:
        - CreateNamespace=true

salt:
  enabled: true
  masterPort: 4505
  publishPort: 4506
```

## Common Workflows

### Deploy Multi-Cloud HA Cluster

```bash
# 1. Create configuration
cat > multi-cloud.yaml <<EOF
metadata:
  name: multi-cloud-ha

providers:
  digitalocean:
    enabled: true
    token: \${DO_TOKEN}
  linode:
    enabled: true
    token: \${LINODE_TOKEN}

security:
  bastion:
    enabled: true
    provider: digitalocean

nodePools:
  - name: do-masters
    provider: digitalocean
    role: master
    count: 1

  - name: linode-masters
    provider: linode
    role: master
    count: 2

  - name: workers
    provider: digitalocean
    role: worker
    count: 5
EOF

# 2. Validate
sloth-kubernetes validate --config multi-cloud.yaml

# 3. Deploy
sloth-kubernetes deploy --config multi-cloud.yaml

# 4. Verify
sloth-kubernetes kubectl get nodes -o wide
```

### Scale Node Pool

```bash
# Add workers
sloth-kubernetes nodes add --pool workers --count 3

# Verify new nodes
sloth-kubernetes nodes list

# Check Kubernetes
sloth-kubernetes kubectl get nodes
```

### Upgrade Kubernetes Version

```bash
# 1. Update cluster.yaml
# Change kubernetes.version to new version

# 2. Refresh cluster
sloth-kubernetes deploy --config cluster.yaml

# 3. Verify
sloth-kubernetes kubectl version
```

### Backup and Restore

```bash
# Export stack state
sloth-kubernetes stacks export > backup.json

# Export etcd snapshot (on master node)
sloth-kubernetes nodes ssh master-0
sudo rke2 etcd-snapshot save --name backup-$(date +%Y%m%d)

# Restore from backup
sloth-kubernetes stacks import backup.json
```

## Best Practices

### Security

1. **Use Bastion Host**: Always enable bastion for production clusters
2. **Restrict SSH Access**: Limit `allowedSSHSources` to your IP ranges
3. **Enable MFA**: Set `security.bastion.mfa: true`
4. **Rotate Tokens**: Regularly rotate cloud provider tokens
5. **Use Private Nodes**: Keep worker nodes private with WireGuard VPN

### High Availability

1. **Odd Number Masters**: Use 3 or 5 masters for etcd quorum
2. **Multi-Cloud Masters**: Distribute across providers for resilience
3. **etcd Backups**: Enable automatic snapshots
4. **Monitor Health**: Use health check component

### Cost Optimization

1. **Right-Size Nodes**: Start small and scale up
2. **Use Spot Instances**: Mix spot and on-demand for workers
3. **Auto-Scaling**: Implement cluster-autoscaler addon
4. **Monitor Usage**: Track costs with cloud provider billing

## Troubleshooting

### Common Issues

**Deployment fails at node provisioning**:
```bash
# Check Pulumi logs
sloth-kubernetes status

# Verify credentials
sloth-kubernetes login digitalocean
```

**Nodes not joining cluster**:
```bash
# SSH to master
sloth-kubernetes nodes ssh master-0
sudo systemctl status rke2-server
sudo journalctl -u rke2-server -f

# Check worker
sloth-kubernetes nodes ssh worker-0
sudo systemctl status rke2-agent
```

**SaltStack connectivity issues**:
```bash
# Test ping
sloth-kubernetes salt ping

# Check minion keys
sloth-kubernetes salt keys list

# Accept pending keys
sloth-kubernetes salt keys accept-all
```

## Next Steps

- **[Architecture](../architecture/index.md)** - Understand how it all works
- **[FAQ](../faq.md)** - Frequently asked questions
