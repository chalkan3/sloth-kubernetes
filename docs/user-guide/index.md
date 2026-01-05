---
title: User Guide
description: Complete reference for all sloth-kubernetes commands and features
---

# User Guide

Complete guide to using **sloth-kubernetes** for deploying and managing multi-cloud Kubernetes clusters.

## Overview

**sloth-kubernetes** is a single binary that embeds all tools needed for multi-cloud Kubernetes:

- Pulumi Automation API - No external Pulumi CLI required
- kubectl - Full Kubernetes client embedded
- Helm support - Package management via external helm binary
- SaltStack - 100+ remote execution modules
- WireGuard VPN - Secure mesh networking
- RKE2 - CNCF-certified Kubernetes distribution

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

---

## Detailed Command Reference

### Core Commands

#### `deploy`

Deploy a new Kubernetes cluster or update an existing one.

**Synopsis:**
```bash
sloth-kubernetes deploy [stack-name] --config <file>
```

**Features:**
- Multi-cloud deployment across DigitalOcean, Linode, AWS (in progress)
- RKE2 Kubernetes with automatic WireGuard VPN mesh
- 8-phase orchestrated deployment
- Comprehensive pre-deployment validation
- Dry-run mode for previewing changes

**Flags:**
- `--config <file>` - Configuration YAML file (required)
- `--dry-run` - Preview changes without applying
- `--yes` - Auto-approve without confirmation

**Examples:**
```bash
# Deploy new cluster
sloth-kubernetes deploy production --config prod.yaml

# Update existing cluster
sloth-kubernetes deploy production --config prod-updated.yaml

# Preview changes
sloth-kubernetes deploy staging --config staging.yaml --dry-run
```

**Deployment Phases:**
1. SSH keys generation & upload
2. Bastion host (if enabled)
3. VPC networks per provider
4. WireGuard encrypted mesh
5. Master & worker nodes
6. RKE2 Kubernetes bootstrap
7. VPN routing configuration
8. DNS service discovery

---

#### `destroy`

Destroy an existing cluster and all associated resources.

**Synopsis:**
```bash
sloth-kubernetes destroy [stack-name]
```

**Features:**
- Destroys all VMs, DNS records, and configurations
- Automatic VPN disconnect and Salt session cleanup
- Double confirmation for safety
- Force destroy option

**Flags:**
- `--yes` - Skip confirmation
- `--force` - Force destroy even with dependencies

**Examples:**
```bash
# Destroy with confirmation
sloth-kubernetes destroy production

# Force destroy without confirmation
sloth-kubernetes destroy staging --yes --force
```

**Warning:** This action cannot be undone!

---

#### `validate`

Validate cluster configuration before deployment.

**Synopsis:**
```bash
sloth-kubernetes validate --config <file>
```

**Validation Checks:**
- YAML syntax and structure
- Required fields and metadata
- Node distribution (masters/workers)
- Provider configuration and credentials
- Network and WireGuard VPN settings
- DNS configuration
- Resource limits and quotas
- SSH configuration

**Examples:**
```bash
# Validate configuration
sloth-kubernetes validate --config cluster.yaml

# Validate with detailed output
sloth-kubernetes validate --config production.yaml --verbose
```

---

#### `status`

Show current cluster status and health.

**Synopsis:**
```bash
sloth-kubernetes status [stack-name]
```

**Displays:**
- Cluster overview
- Node count and distribution
- Provider information
- Network configuration
- VPN status
- Kubernetes version
- Component health

---

#### `kubeconfig`

Retrieve kubeconfig for kubectl access.

**Synopsis:**
```bash
sloth-kubernetes kubeconfig [stack-name]
```

**Flags:**
- `-o, --output <file>` - Save to file (default: stdout)
- `--merge` - Merge with existing kubeconfig (not implemented)

**Examples:**
```bash
# Print to stdout
sloth-kubernetes kubeconfig

# Save to default location
sloth-kubernetes kubeconfig -o ~/.kube/config

# Export and use immediately
sloth-kubernetes kubeconfig -o ~/.kube/config
export KUBECONFIG=~/.kube/config
kubectl get nodes
```

---

### Node Management Commands

#### `nodes list`

List all nodes in the cluster.

**Synopsis:**
```bash
sloth-kubernetes nodes list [stack-name]
```

**Displays:**
- Node name, IP address, provider
- Role (master/worker)
- Status, age
- Resource allocation

---

#### `nodes ssh`

SSH into a cluster node via bastion.

**Synopsis:**
```bash
sloth-kubernetes nodes ssh [stack-name] [node-name]
```

**Features:**
- Automatic bastion jump configuration
- Uses cluster SSH keys
- Direct access to private nodes

**Examples:**
```bash
# SSH to specific node
sloth-kubernetes nodes ssh production master-0

# SSH to worker
sloth-kubernetes nodes ssh production worker-2
```

---

#### `nodes add`

Add new nodes to an existing node pool.

**Synopsis:**
```bash
sloth-kubernetes nodes add [stack-name] --pool <name> --count <n>
```

**Flags:**
- `--pool <name>` - Target node pool
- `--count <n>` - Number of nodes to add

**Examples:**
```bash
# Add 2 workers
sloth-kubernetes nodes add production --pool workers --count 2

# Add master node
sloth-kubernetes nodes add staging --pool masters --count 1
```

---

#### `nodes remove`

Remove a node from the cluster.

**Synopsis:**
```bash
sloth-kubernetes nodes remove [stack-name] [node-name]
```

**Features:**
- Drains node before removal
- Updates cluster state
- Cleans up VPN configuration

---

#### `nodes drain`

Drain node for maintenance (evict all pods).

**Synopsis:**
```bash
sloth-kubernetes nodes drain [stack-name] [node-name]
```

---

#### `nodes cordon`

Mark node as unschedulable (prevent new pods).

**Synopsis:**
```bash
sloth-kubernetes nodes cordon [stack-name] [node-name]
```

---

#### `nodes uncordon`

Mark node as schedulable again.

**Synopsis:**
```bash
sloth-kubernetes nodes uncordon [stack-name] [node-name]
```

---

### SaltStack Commands

SaltStack provides 100+ remote execution modules for managing cluster nodes.

#### `salt ping`

Test connectivity to all or specific minions.

**Synopsis:**
```bash
sloth-kubernetes salt ping [--target <minion>]
```

---

#### `salt minions`

List all connected Salt minions.

**Synopsis:**
```bash
sloth-kubernetes salt minions
```

---

#### `salt cmd.run`

Execute shell commands on cluster nodes.

**Synopsis:**
```bash
sloth-kubernetes salt cmd.run "<command>" [--target <minion>]
```

**Examples:**
```bash
# Run on all nodes
sloth-kubernetes salt cmd.run "uptime"

# Run on specific node
sloth-kubernetes salt cmd.run "df -h" --target master-0

# Check memory usage
sloth-kubernetes salt cmd.run "free -h"
```

---

#### `salt pkg.*`

Package management commands.

**Synopsis:**
```bash
sloth-kubernetes salt pkg.install <package>
sloth-kubernetes salt pkg.remove <package>
sloth-kubernetes salt pkg.upgrade
sloth-kubernetes salt pkg.list_upgrades
```

**Examples:**
```bash
# Install package
sloth-kubernetes salt pkg.install htop

# Remove package
sloth-kubernetes salt pkg.remove nginx

# Upgrade all packages
sloth-kubernetes salt pkg.upgrade
```

---

#### `salt service.*`

Service management commands.

**Synopsis:**
```bash
sloth-kubernetes salt service.status <service>
sloth-kubernetes salt service.restart <service>
sloth-kubernetes salt service.start <service>
sloth-kubernetes salt service.stop <service>
sloth-kubernetes salt service.enable <service>
sloth-kubernetes salt service.disable <service>
```

**Examples:**
```bash
# Check kubelet status
sloth-kubernetes salt service.status kubelet

# Restart RKE2
sloth-kubernetes salt service.restart rke2-server --target master-0
```

---

#### `salt grains.*`

System information commands.

**Synopsis:**
```bash
sloth-kubernetes salt grains.items
sloth-kubernetes salt grains.get <key>
```

**Examples:**
```bash
# Get all system info
sloth-kubernetes salt grains.items

# Get OS info
sloth-kubernetes salt grains.get os

# Get kernel version
sloth-kubernetes salt grains.get kernelrelease
```

---

#### `salt state.*`

Configuration management.

**Synopsis:**
```bash
sloth-kubernetes salt state.apply [state]
sloth-kubernetes salt state.highstate
```

---

#### `salt keys`

Manage Salt minion keys.

**Synopsis:**
```bash
sloth-kubernetes salt keys list
sloth-kubernetes salt keys accept <minion-id>
sloth-kubernetes salt keys delete <minion-id>
sloth-kubernetes salt keys accept-all
```

---

### VPN Management Commands

#### `vpn status`

Show VPN status and tunnel information.

**Synopsis:**
```bash
sloth-kubernetes vpn status [stack-name]
```

**Displays:**
- VPN server status
- Connected peers
- Tunnel configuration
- Traffic statistics

---

#### `vpn peers`

List all VPN peers in the mesh.

**Synopsis:**
```bash
sloth-kubernetes vpn peers [stack-name]
```

---

#### `vpn config`

Get VPN configuration for a specific node.

**Synopsis:**
```bash
sloth-kubernetes vpn config [stack-name] [node-name]
```

**Output:** WireGuard configuration file

---

#### `vpn test`

Test VPN connectivity across the mesh.

**Synopsis:**
```bash
sloth-kubernetes vpn test [stack-name]
```

**Tests:**
- Ping all nodes via VPN
- Latency measurements
- Throughput tests

---

#### `vpn join`

Join this machine or remote host to the VPN mesh.

**Synopsis:**
```bash
sloth-kubernetes vpn join [stack-name]
```

**Features:**
- Generates client configuration
- Configures local WireGuard interface
- Adds routes to cluster networks

---

#### `vpn leave`

Remove this machine from the VPN mesh.

**Synopsis:**
```bash
sloth-kubernetes vpn leave [stack-name]
```

---

### Stack Management Commands

Manage multiple cluster stacks (multiple clusters).

#### `stacks list`

List all available stacks.

**Synopsis:**
```bash
sloth-kubernetes stacks list
```

---

#### `stacks info`

Show detailed stack information.

**Synopsis:**
```bash
sloth-kubernetes stacks info [stack-name]
```

---

#### `stacks select`

Select active stack.

**Synopsis:**
```bash
sloth-kubernetes stacks select <stack-name>
```

---

#### `stacks delete`

Delete a stack (after destroying resources).

**Synopsis:**
```bash
sloth-kubernetes stacks delete <stack-name>
```

---

#### `stacks export`

Export stack state to JSON.

**Synopsis:**
```bash
sloth-kubernetes stacks export [stack-name]
```

**Examples:**
```bash
# Export to file
sloth-kubernetes stacks export production > backup.json
```

---

#### `stacks import`

Import stack state from JSON.

**Synopsis:**
```bash
sloth-kubernetes stacks import <file>
```

---

#### `stacks output`

Show stack outputs (IPs, endpoints, etc).

**Synopsis:**
```bash
sloth-kubernetes stacks output [stack-name]
```

---

### Kubernetes Tools

#### `kubectl`

Embedded kubectl client with full functionality.

**Synopsis:**
```bash
sloth-kubernetes kubectl <kubectl-args>
```

**Features:**
- Full kubectl functionality embedded
- No separate kubectl installation required
- Automatic kubeconfig detection

**Examples:**
```bash
# Get nodes
sloth-kubernetes kubectl get nodes

# Get pods in all namespaces
sloth-kubernetes kubectl get pods -A

# Apply manifest
sloth-kubernetes kubectl apply -f deployment.yaml

# Get logs
sloth-kubernetes kubectl logs nginx-123 -n default

# Execute in pod
sloth-kubernetes kubectl exec -it nginx-123 -- sh

# Port forward
sloth-kubernetes kubectl port-forward svc/nginx 8080:80
```

---

#### `helm`

Helm package manager (requires external helm binary).

**Synopsis:**
```bash
sloth-kubernetes helm <helm-args>
```

**Prerequisites:** Helm 3.x must be installed in PATH

**Examples:**
```bash
# List releases
sloth-kubernetes helm list

# Install chart
sloth-kubernetes helm install nginx bitnami/nginx

# Upgrade release
sloth-kubernetes helm upgrade nginx bitnami/nginx

# Add repository
sloth-kubernetes helm repo add bitnami https://charts.bitnami.com/bitnami

# Uninstall release
sloth-kubernetes helm uninstall nginx
```

---

#### `kustomize`

Kustomize template rendering (requires external kustomize binary).

**Synopsis:**
```bash
sloth-kubernetes kustomize <kustomize-args>
```

**Examples:**
```bash
# Build kustomization
sloth-kubernetes kustomize build overlays/production

# Apply kustomization
sloth-kubernetes kustomize build overlays/production | kubectl apply -f -
```

---

### GitOps & Addons

#### `addons bootstrap`

Bootstrap ArgoCD from a GitOps repository.

**Synopsis:**
```bash
sloth-kubernetes addons bootstrap --repo <url>
```

**Flags:**
- `--repo <url>` - Git repository URL (required)
- `--branch <name>` - Git branch (default: main)
- `--path <path>` - Path within repo (default: addons/)
- `--private-key <file>` - SSH key for private repos

**Features:**
- Clones GitOps repository
- Installs ArgoCD from repo manifests
- Configures ArgoCD to watch repo
- Auto-syncs all other addons

**Examples:**
```bash
# Bootstrap with public repo
sloth-kubernetes addons bootstrap --repo https://github.com/you/gitops-repo

# Bootstrap with private repo
sloth-kubernetes addons bootstrap \
  --repo git@github.com:you/private-repo.git \
  --private-key ~/.ssh/id_rsa

# Custom branch and path
sloth-kubernetes addons bootstrap \
  --repo https://github.com/you/gitops-repo \
  --branch production \
  --path cluster-addons/
```

---

#### `addons list`

List all installed addons.

**Synopsis:**
```bash
sloth-kubernetes addons list [stack-name]
```

**Displays:**
- Addon name, category, status
- Version, namespace
- Sync status (if managed by ArgoCD)

---

#### `addons sync`

Manually trigger ArgoCD sync.

**Synopsis:**
```bash
sloth-kubernetes addons sync [--app <name>]
```

**Flags:**
- `--app <name>` - Sync specific application (default: all)

---

#### `addons status`

Show ArgoCD and addon status.

**Synopsis:**
```bash
sloth-kubernetes addons status
```

**Displays:**
- ArgoCD server status
- Application sync status
- Health status of each addon

---

#### `addons template`

Generate example GitOps repository structure.

**Synopsis:**
```bash
sloth-kubernetes addons template [-o <dir>]
```

**Flags:**
- `-o, --output <dir>` - Output directory (default: print to stdout)

**Examples:**
```bash
# Print template structure
sloth-kubernetes addons template

# Generate to directory
sloth-kubernetes addons template --output ./my-gitops-repo
```

---

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
  masterNode: "0"           # Node index to run Salt Master
  apiEnabled: true          # Enable Salt REST API
  apiPort: 8000             # Salt API port
  secureAuth: true          # Use SHA256 hash-based minion authentication
  autoJoin: true            # Automatically join minions to master
  auditLogging: true        # Enable authentication audit logging
```

---

## Salt Configuration

### Secure Hash-Based Authentication

sloth-kubernetes implements a secure minion authentication system using SHA256 hash tokens instead of the insecure `auto_accept: True` approach.

**How it works:**

1. A unique cluster token is generated using SHA256 hash of cluster name + stack name + timestamp
2. Salt Master is configured with `auto_accept: False` and `autosign_grains_dir`
3. Each minion is configured with the cluster token in its grains
4. Salt Master validates the grain token before accepting the minion key
5. All authentication events are logged for audit

**Security Configuration:**

```yaml
salt:
  enabled: true
  secureAuth: true         # Enables hash-based authentication
  auditLogging: true       # Logs all key auth events
```

**Master Security Settings:**

When `secureAuth` is enabled, the Salt Master is configured with:

```yaml
auto_accept: False              # Never auto-accept minion keys
open_mode: False                # Disable open mode
autosign_grains_dir: /etc/salt/autosign_grains  # Grain-based autosign
```

**Minion Grain Token:**

Each minion receives a grain configuration with the cluster token:

```yaml
grains:
  cluster_token: <sha256-token>
  node_type: master|worker
  cluster: sloth-kubernetes
```

### Salt Configuration in Lisp

For Lisp-based configuration files:

```lisp
(addons
  (salt
    (enabled true)
    (master-node "0")       ; Node index for Salt Master
    (api-enabled true)      ; Enable REST API
    (api-port 8000)         ; API port
    (secure-auth true)      ; SHA256 token authentication
    (auto-join true)        ; Auto-join minions
    (audit-logging true)))  ; Authentication audit logs
```

### Salt API

When `apiEnabled` is true, Salt REST API is available for programmatic access:

```bash
# Login to Salt API
curl -sSk http://<master-ip>:8000/login \
  -d username=saltapi \
  -d password=<api-password> \
  -d eauth=pam

# Run commands via API
curl -sSk http://<master-ip>:8000/ \
  -H "X-Auth-Token: <token>" \
  -d client=local \
  -d tgt='*' \
  -d fun=cmd.run \
  -d arg='uptime'
```

### Reactor Audit Logging

When `auditLogging` is enabled, all authentication events are logged:

```yaml
# Salt reactor configuration
reactor:
  - 'salt/auth':
    - /srv/reactor/auth_log.sls
```

Authentication events are logged to `/var/log/salt/auth_audit.log`:

```
[2024-01-15 10:30:45] Key auth event: worker-0 - accept
[2024-01-15 10:30:46] Key auth event: worker-1 - accept
[2024-01-15 10:31:00] Key auth event: unknown-node - reject
```

---

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

- **[Configuration](../configuration/lisp-format.md)** - LISP configuration reference
- **[FAQ](../faq.md)** - Frequently asked questions
