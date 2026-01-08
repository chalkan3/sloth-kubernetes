# CLI Reference

Complete reference for all sloth-kubernetes commands.

---

## Global Flags

These flags work with all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--help, -h` | Show help for command | - |
| `--version, -v` | Show version | - |
| `--debug` | Enable debug logging | `false` |
| `--config, -c` | Path to config file | `cluster.lisp` |

---

## Commands Overview

```bash
sloth-kubernetes [command] [stack-name] [flags]
```

Most commands require a **stack name** as the first argument. The stack name identifies your Pulumi stack (e.g., `production`, `staging`, `my-cluster`). Commands that depend on kubeconfig automatically retrieve it from the stack.

Available Commands:

**Deployment & Configuration:**
- [`deploy`](#deploy) - Deploy a Kubernetes cluster
- [`destroy`](#destroy) - Destroy a cluster
- [`validate`](#validate) - Validate configuration
- [`config`](#config) - Generate example configuration
- [`export-config`](#export-config) - Export config from Pulumi state
- [`login`](#login) - Configure S3 state backend

**Cluster Operations:**
- [`kubectl`](#kubectl) - Kubernetes operations (stack-aware)
- [`helm`](#helm) - Helm chart management (stack-aware)
- [`kustomize`](#kustomize) - Kustomize operations (stack-aware)
- [`kubeconfig`](#kubeconfig) - Generate kubeconfig

**Node & Infrastructure:**
- [`nodes`](#nodes) - Manage cluster nodes
- [`salt`](#salt) - Node management with SaltStack
- [`vpn`](#vpn) - VPN management (WireGuard or Tailscale/Headscale)

**Monitoring & Operations:**
- [`health`](#health) - Cluster health checks (stack-aware)
- [`backup`](#backup) - Velero backup management (stack-aware)
- [`benchmark`](#benchmark) - Cluster benchmarks (stack-aware)
- [`upgrade`](#upgrade) - Cluster upgrades (stack-aware)
- [`history`](#history) - View operation history

**GitOps & Addons:**
- [`argocd`](#argocd) - ArgoCD GitOps (stack-aware)
- [`addons`](#addons) - Manage cluster addons via GitOps

**State Management:**
- [`stacks`](#stacks) - Manage Pulumi stacks
- [`pulumi`](#pulumi) - Direct Pulumi operations (no CLI required)

**Utility:**
- [`version`](#version) - Show version info
- [`list`](#list) - List deployed clusters
- [`status`](#status) - Show cluster status

---

## `deploy`

Deploy a new Kubernetes cluster or update an existing one.

### Usage

```bash
sloth-kubernetes deploy [flags]
```

### Flags

| Flag | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| `--config, -c` | string | Path to cluster config file | Yes | `cluster.lisp` |
| `--dry-run` | bool | Preview changes without applying | No | `false` |
| `--auto-approve` | bool | Skip confirmation prompt | No | `false` |
| `--parallel` | int | Max parallel operations | No | `10` |
| `--timeout` | duration | Deployment timeout | No | `30m` |

### Examples

```bash
# Deploy with default config
sloth-kubernetes deploy --config cluster.lisp

# Deploy with custom config
sloth-kubernetes deploy --config production.lisp

# Preview changes (dry run)
sloth-kubernetes preview --config cluster.lisp

# Auto-approve without confirmation
sloth-kubernetes deploy --config cluster.lisp --auto-approve

# Deploy with timeout
sloth-kubernetes deploy --config cluster.lisp --timeout 45m
```

### Output

```
Sloth Kubernetes Deployment
Slowly, but surely deploying your cluster...

Stack: my-cluster
Config: cluster.yaml

Preview:
  + 2 VPCs
  + 1 WireGuard VPN server
  + 3 Master nodes
  + 2 Worker nodes
  + 5 DNS records

Continue with deployment? [y/N]: y

✓ Creating resources... (5m 32s)
✓ Installing Kubernetes... (3m 45s)
✓ Configuring VPN mesh... (1m 12s)

🦥 Deployment complete!
   Time: 10m 29s
   Kubeconfig: ./my-cluster-kubeconfig.yaml
```

---

## `destroy`

Destroy a Kubernetes cluster and all associated resources.

### Usage

```bash
sloth-kubernetes destroy [flags]
```

### Flags

| Flag | Type | Description | Required | Default |
|------|------|-------------|----------|---------|
| `--config, -c` | string | Path to cluster config file | Yes | `cluster.lisp` |
| `--force, -f` | bool | Skip confirmation prompt | No | `false` |
| `--remove-state` | bool | Also remove state files | No | `false` |

### Examples

```bash
# Destroy clustersloth-kubernetes destroy

# Force destroy (no confirmation)
sloth-kubernetes destroy --force

# Destroy and remove state
sloth-kubernetes destroy --remove-state
```

### Output

```
🦥 Sloth Kubernetes Destruction
Slowly tearing down your cluster...

⚠ WARNING: This will destroy:
  - 2 VPCs
  - 1 VPN server
  - 5 nodes (3 masters, 2 workers)
  - All data and volumes

Type cluster name to confirm: my-cluster

✓ Removing nodes... (3m 12s)
✓ Destroying VPN... (45s)
✓ Deleting VPCs... (1m 5s)

🦥 Cluster destroyed successfully!
```

---

## `nodes`

Manage cluster nodes: list, add, remove, or drain.

### Subcommands

- `nodes list` - List all nodes- `nodes add` - Add nodes to cluster- `nodes remove` - Remove nodes from cluster- `nodes drain` - Drain a node for maintenance
### `nodes list`

List all nodes in the cluster.

```bash
sloth-kubernetes nodes list [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config, -c` | string | Cluster config | `cluster.lisp` |
| `--output, -o` | string | Output format: `table`, `json`, `yaml` | `table` |

**Example:**

```bash
# List nodessloth-kubernetes nodes list

# Output as JSON
sloth-kubernetes nodes list -o json
```

**Output:**

```
🦥 Cluster Nodes

NAME              PROVIDER        ROLE     STATUS   IP            REGION
do-master-1       digitalocean    master   Ready    10.10.1.5     nyc3
linode-master-1   linode          master   Ready    10.11.1.5     us-east
linode-master-2   linode          master   Ready    10.11.1.6     us-east
do-worker-1       digitalocean    worker   Ready    10.10.1.10    nyc3
linode-worker-1   linode          worker   Ready    10.11.1.10    us-east

Total: 5 nodes (3 masters, 2 workers)
```

### `nodes add`

Add new nodes to an existing cluster.

```bash
sloth-kubernetes nodes add --pool POOL_NAME --count COUNT [flags]
```

**Flags:**

| Flag | Type | Description | Required |
|------|------|-------------|----------|
| `--pool` | string | Node pool name from config | Yes |
| `--count` | int | Number of nodes to add | Yes |
| `--config, -c` | string | Cluster config | No |

**Example:**

```bash
# Add 2 workers to linode-workers poolsloth-kubernetes nodes add --pool linode-workers --count 2

# Add 1 master
sloth-kubernetes nodes add --pool do-masters --count 1
```

### `nodes remove`

Remove nodes from the cluster.

```bash
sloth-kubernetes nodes remove NODE_NAME [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--force, -f` | bool | Skip drain and delete immediately | `false` |
| `--drain-timeout` | duration | Timeout for draining | `5m` |

**Example:**

```bash
# Remove a node (with graceful drain)sloth-kubernetes nodes remove do-worker-2

# Force remove without drain
sloth-kubernetes nodes remove do-worker-2 --force
```

### `nodes drain`

Drain a node for maintenance.

```bash
sloth-kubernetes nodes drain NODE_NAME [flags]
```

**Example:**

```bash
# Drain node for maintenancesloth-kubernetes nodes drain do-worker-1
```

---

## `vpn`

Manage VPN networking with WireGuard or Tailscale/Headscale.

### Subcommands

**Tailscale/Headscale (Embedded Client):**
- `vpn connect` - Connect local machine to Tailscale mesh
- `vpn disconnect` - Disconnect from Tailscale mesh

**WireGuard:**
- `vpn status` - Show VPN status
- `vpn peers` - List VPN peers
- `vpn join` - Join WireGuard mesh
- `vpn leave` - Leave WireGuard mesh
- `vpn test` - Test VPN connectivity
- `vpn config` - Get node WireGuard config
- `vpn client-config` - Generate client config

---

### `vpn connect` (Tailscale)

Connect your local machine to the Tailscale mesh using the embedded client. No system-wide Tailscale installation required.

```bash
sloth-kubernetes vpn connect <stack-name> [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--daemon` | bool | Run in background | `false` |
| `--hostname` | string | Custom hostname in tailnet | Auto-generated |

**Example:**

```bash
# Connect in daemon mode (recommended)
sloth-kubernetes vpn connect my-cluster --daemon

# Connect with custom hostname
sloth-kubernetes vpn connect my-cluster --daemon --hostname my-laptop
```

**Output:**

```
🔌 VPN Connect (Daemon) - Stack: my-cluster

Starting VPN daemon in background...
Waiting for VPN connection to establish...
✓ VPN daemon started (PID: 12345)
SOCKS5 proxy running on 127.0.0.1:64172

  kubectl commands will automatically use the VPN tunnel
  Use 'sloth vpn disconnect my-cluster' to stop
```

**Note:** When connected, kubectl commands automatically route through the VPN:
```bash
sloth-kubernetes kubectl my-cluster get nodes  # Works through VPN
```

---

### `vpn disconnect` (Tailscale)

Disconnect from the Tailscale mesh and stop the daemon.

```bash
sloth-kubernetes vpn disconnect <stack-name>
```

**Example:**

```bash
sloth-kubernetes vpn disconnect my-cluster
```

**Output:**

```
🔌 VPN Disconnect - Stack: my-cluster

Stopping VPN daemon (PID: 12345)...
✓ VPN daemon stopped
Cleaning up connection state...
✓ Disconnected and cleaned up VPN state
```

---

### `vpn status` (WireGuard)

Show WireGuard VPN status and connected nodes.

```bash
sloth-kubernetes vpn status <stack-name>
```

**Output:**

```
═══════════════════════════════════════════════════════════════
           VPN STATUS - Stack: production
═══════════════════════════════════════════════════════════════

METRIC          VALUE
------          -----
VPN Mode        WireGuard Mesh
Total Nodes     6
Total Tunnels   15
VPN Subnet      10.8.0.0/24
Status          All tunnels active
```

---

### `vpn peers` (WireGuard)

List all VPN peers in the mesh.

```bash
sloth-kubernetes vpn peers <stack-name>
```

**Output:**

```
NODE         LABEL      VPN IP       PUBLIC KEY        LAST HANDSHAKE   TRANSFER
----         -----      ------       ----------        --------------   --------
master-1     -          10.8.0.10    ABC123def456...   30s ago          1.2MB / 2.4MB
worker-1     -          10.8.0.20    GHI789jkl012...   1m ago           3.5MB / 5.2MB
laptop       personal   10.8.0.100   MNO345pqr678...   2m ago           500KB / 1.2MB
```

---

### `vpn join` (WireGuard)

Join your local machine or a remote host to the WireGuard mesh.

```bash
sloth-kubernetes vpn join <stack-name> [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--remote` | string | Remote SSH host to add | - |
| `--vpn-ip` | string | Custom VPN IP address | Auto-assign |
| `--label` | string | Peer label/name | - |
| `--install` | bool | Auto-install WireGuard | `false` |

**Example:**

```bash
# Join local machine
sloth-kubernetes vpn join production --install

# Join with label
sloth-kubernetes vpn join production --label laptop --install

# Join remote host
sloth-kubernetes vpn join production --remote user@server.com
```

---

### `vpn leave` (WireGuard)

Remove a machine from the WireGuard mesh.

```bash
sloth-kubernetes vpn leave <stack-name> [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--vpn-ip` | string | VPN IP of peer to remove | Auto-detect |

**Example:**

```bash
# Leave VPN
sloth-kubernetes vpn leave production

# Remove specific peer
sloth-kubernetes vpn leave production --vpn-ip 10.8.0.100
```

---

### `vpn client-config` (WireGuard)

Generate WireGuard client configuration file.

```bash
sloth-kubernetes vpn client-config <stack-name> [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--output, -o` | string | Output file path | `wg0-client.conf` |
| `--qr` | bool | Generate QR code | `false` |

**Example:**

```bash
# Generate client config
sloth-kubernetes vpn client-config production

# Generate QR code for mobile
sloth-kubernetes vpn client-config production --qr
```

---

## `stacks`

Manage Pulumi stacks for cluster state.

### Subcommands

- `stacks list` - List all stacks- `stacks state list` - List stack resources- `stacks state delete` - Delete specific resources
### `stacks list`

List all Pulumi stacks.

```bash
sloth-kubernetes stacks list
```

**Example:**

```bash
# List stackssloth-kubernetes stacks list
```

**Output:**

```
🦥 Pulumi Stacks

NAME              LAST UPDATE       RESOURCE COUNT
my-cluster        2 hours ago       47 resources
staging-cluster   1 day ago         23 resources
```

### `stacks state list`

List all resources in a stack.

```bash
sloth-kubernetes stacks state list [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config, -c` | string | Cluster config | `cluster.lisp` |
| `--type` | string | Filter by resource type | - |

**Example:**

```bash
# List all resourcessloth-kubernetes stacks state list

# Filter by type
sloth-kubernetes stacks state list --type digitalocean:Droplet
```

---

## `kubeconfig`

Generate kubeconfig for cluster access from a stack.

### Usage

```bash
sloth-kubernetes kubeconfig <stack-name> [flags]
```

### Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--output, -o` | string | Output file | stdout |

### Examples

```bash
# Print kubeconfig for a stack
sloth-kubernetes kubeconfig my-cluster

# Save to file
sloth-kubernetes kubeconfig my-cluster > ~/.kube/config

# Use immediately with kubectl
export KUBECONFIG=~/.kube/config
sloth-kubernetes kubeconfig production > $KUBECONFIG
kubectl get nodes
```

---

## `version`

Show version information.

### Usage

```bash
sloth-kubernetes version
```

### Output

```
🦥 Sloth Kubernetes
Version: 1.0.0
Git Commit: abc123
Built: 2025-01-15T10:30:00Z
Go Version: go1.23.4
Platform: darwin/arm64
```

---

## `kubectl`

Run kubectl commands against a stack's cluster. The kubeconfig is automatically retrieved from the Pulumi stack.

### Usage

```bash
sloth-kubernetes kubectl <stack-name> <kubectl-command>
```

### Examples

```bash
# Get nodes
sloth-kubernetes kubectl my-cluster get nodes

# Get all pods
sloth-kubernetes kubectl my-cluster get pods -A

# Apply manifest
sloth-kubernetes kubectl my-cluster apply -f deployment.yaml

# View pod logs
sloth-kubernetes kubectl my-cluster logs -f nginx-pod
```

---

## `health`

Run health checks on a stack's cluster.

### Usage

```bash
sloth-kubernetes health <stack-name> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--verbose`, `-v` | Show verbose output | `false` |
| `--compact` | Show compact output (only issues) | `false` |
| `--checks` | Specific checks to run | All |

### Examples

```bash
# Full health check
sloth-kubernetes health my-cluster

# Verbose output
sloth-kubernetes health my-cluster --verbose

# Specific checks only
sloth-kubernetes health my-cluster --checks nodes,pods,dns
```

---

## `backup`

Manage Velero backups for a stack's cluster.

### Usage

```bash
sloth-kubernetes backup <subcommand> <stack-name> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `status` | Check Velero status |
| `create` | Create a backup |
| `list` | List backups |
| `describe` | Describe a backup |
| `delete` | Delete a backup |
| `restore` | Restore from backup |
| `schedule create` | Create backup schedule |
| `schedule list` | List schedules |
| `schedule delete` | Delete schedule |

### Examples

```bash
# Check Velero status
sloth-kubernetes backup status my-cluster

# Create backup
sloth-kubernetes backup create my-cluster my-backup

# List backups
sloth-kubernetes backup list my-cluster

# Restore from backup
sloth-kubernetes backup restore my-cluster --from-backup my-backup
```

---

## `benchmark`

Run performance benchmarks on a stack's cluster.

### Usage

```bash
sloth-kubernetes benchmark <subcommand> <stack-name> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `run` | Execute benchmarks |
| `quick` | Quick benchmark summary |
| `report` | Display saved report |
| `compare` | Compare two reports |

### Examples

```bash
# Run all benchmarks
sloth-kubernetes benchmark run my-cluster

# Run specific type
sloth-kubernetes benchmark run my-cluster --type network

# Quick benchmark
sloth-kubernetes benchmark quick my-cluster

# Save results
sloth-kubernetes benchmark run my-cluster --output json --save results.json
```

---

## `upgrade`

Manage Kubernetes version upgrades for a stack's cluster.

### Usage

```bash
sloth-kubernetes upgrade <subcommand> <stack-name> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `plan` | Create upgrade plan |
| `apply` | Execute upgrade |
| `rollback` | Rollback to previous version |
| `versions` | List available versions |
| `status` | Show upgrade status |

### Examples

```bash
# Check available versions
sloth-kubernetes upgrade versions my-cluster

# Plan upgrade
sloth-kubernetes upgrade plan my-cluster --to v1.29.0

# Execute upgrade
sloth-kubernetes upgrade apply my-cluster --to v1.29.0

# Rollback
sloth-kubernetes upgrade rollback my-cluster
```

---

## `argocd`

Manage ArgoCD GitOps integration for a stack's cluster.

### Usage

```bash
sloth-kubernetes argocd <subcommand> <stack-name> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `install` | Install ArgoCD |
| `status` | Check ArgoCD status |
| `password` | Get admin password |
| `apps` | List applications |
| `sync` | Sync applications |

### Examples

```bash
# Install ArgoCD
sloth-kubernetes argocd install my-cluster

# With GitOps repo
sloth-kubernetes argocd install my-cluster --repo https://github.com/org/gitops.git

# Check status
sloth-kubernetes argocd status my-cluster

# Get password
sloth-kubernetes argocd password my-cluster

# List apps
sloth-kubernetes argocd apps my-cluster

# Sync all apps
sloth-kubernetes argocd sync my-cluster --all
```

---

## `helm`

Execute Helm commands using kubeconfig from a stack. The kubeconfig is automatically retrieved from Pulumi state.

### Usage

```bash
sloth-kubernetes helm <stack-name> [helm-args...]
```

### Examples

```bash
# List all releases
sloth-kubernetes helm my-cluster list

# List releases in all namespaces
sloth-kubernetes helm my-cluster list -A

# Install a chart
sloth-kubernetes helm my-cluster install myapp bitnami/nginx

# Upgrade a release
sloth-kubernetes helm my-cluster upgrade myapp bitnami/nginx

# Add a repository
sloth-kubernetes helm my-cluster repo add bitnami https://charts.bitnami.com/bitnami

# Search for charts
sloth-kubernetes helm my-cluster search repo nginx

# Get release status
sloth-kubernetes helm my-cluster status myapp

# Uninstall a release
sloth-kubernetes helm my-cluster uninstall myapp

# Install with custom values
sloth-kubernetes helm my-cluster install redis bitnami/redis -f values.yaml

# Install in specific namespace
sloth-kubernetes helm my-cluster install nginx bitnami/nginx -n web --create-namespace
```

---

## `kustomize`

Execute Kustomize commands for declarative Kubernetes configuration management.

### Usage

```bash
sloth-kubernetes kustomize <stack-name> [kustomize-args...]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `build` | Build a kustomization target |
| `create` | Create a new kustomization |
| `edit` | Edit a kustomization file |

### Examples

```bash
# Build kustomization and apply to cluster
sloth-kubernetes kustomize my-cluster build ./overlays/production | \
  sloth-kubernetes kubectl my-cluster apply -f -

# Build with specific output
sloth-kubernetes kustomize my-cluster build ./base

# Create new kustomization
sloth-kubernetes kustomize my-cluster create --resources deployment.yaml,service.yaml

# Edit kustomization - add resource
sloth-kubernetes kustomize my-cluster edit add resource configmap.yaml

# Edit kustomization - set image
sloth-kubernetes kustomize my-cluster edit set image nginx=nginx:1.25
```

---

## `addons`

Manage Kubernetes cluster addons using GitOps methodology with ArgoCD.

### Usage

```bash
sloth-kubernetes addons <subcommand> <stack-name> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `bootstrap` | Bootstrap ArgoCD from a GitOps repository |
| `list` | List installed addons |
| `status` | Show ArgoCD and addon status |
| `sync` | Manually trigger ArgoCD sync |
| `template` | Generate example GitOps repository structure |

### Examples

```bash
# Bootstrap ArgoCD with GitOps repo
sloth-kubernetes addons bootstrap my-cluster --repo https://github.com/org/gitops.git

# List installed addons
sloth-kubernetes addons list my-cluster

# Check addon status
sloth-kubernetes addons status my-cluster

# Trigger sync
sloth-kubernetes addons sync my-cluster

# Generate example GitOps structure
sloth-kubernetes addons template --output ./my-gitops-repo
```

### GitOps Repository Structure

The addons system expects this directory structure in your GitOps repo:

```
gitops-repo/
├── argocd/
│   └── apps/           # ArgoCD Application manifests
│       ├── monitoring.yaml
│       ├── logging.yaml
│       └── ingress.yaml
├── addons/
│   ├── prometheus/     # Prometheus manifests
│   ├── grafana/        # Grafana manifests
│   ├── nginx-ingress/  # Ingress controller
│   └── cert-manager/   # Certificate management
└── README.md
```

---

## `history`

View operation history stored in Pulumi stack state. All CLI operations are automatically recorded.

### Usage

```bash
sloth-kubernetes history <stack-name> [type] [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output in JSON format | `false` |
| `--limit` | Number of records per type | `10` |

### Operation Types

| Type | Description |
|------|-------------|
| `backups` | Backup create/restore/delete operations |
| `upgrades` | Cluster upgrade operations |
| `health` | Health check results |
| `benchmarks` | Benchmark runs |
| `nodes` | Node add/remove/drain operations |
| `vpn` | VPN join/leave/test operations |
| `argocd` | ArgoCD install/sync operations |
| `addons` | Addons bootstrap/install operations |
| `salt` | Salt command executions |
| `validation` | Validation check results |

### Examples

```bash
# View all operation history
sloth-kubernetes history my-cluster

# View backup history only
sloth-kubernetes history my-cluster backups

# View upgrade history
sloth-kubernetes history my-cluster upgrades

# View health check history
sloth-kubernetes history my-cluster health

# Output as JSON
sloth-kubernetes history my-cluster --json

# Limit records
sloth-kubernetes history my-cluster --limit 5
```

### Output Example

```
Operation History for stack: my-cluster
════════════════════════════════════════════════════════════

Backup Operations (last 10):
────────────────────────────────────────
  2026-01-05 10:30:00  create   daily-backup-001    success   2m 15s
  2026-01-05 08:00:00  create   scheduled-backup    success   1m 45s

Upgrade Operations (last 10):
────────────────────────────────────────
  2026-01-04 15:00:00  upgrade  v1.28.0 → v1.29.0   success   15m 30s

Health Check History (last 10):
────────────────────────────────────────
  2026-01-05 11:00:00  healthy  12/12 passed        45s
```

---

## `login`

Configure the S3 bucket for storing Pulumi state. Similar to `pulumi login` but with S3-compatible storage support.

### Usage

```bash
sloth-kubernetes login [s3://bucket-name] [flags]
```

### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `-b, --bucket` | S3 bucket URL | Yes |
| `--access-key-id` | AWS Access Key ID | No |
| `--secret-access-key` | AWS Secret Access Key | No |
| `--region` | AWS Region | No |
| `--endpoint` | S3 endpoint for S3-compatible storage | No |

### Examples

```bash
# Login with S3 bucket URL
sloth-kubernetes login s3://my-pulumi-state-bucket

# Login with bucket flag
sloth-kubernetes login --bucket my-pulumi-state-bucket

# Login with credentials
sloth-kubernetes login s3://bucket \
  --access-key-id YOUR_KEY \
  --secret-access-key YOUR_SECRET \
  --region us-east-1

# Login to MinIO or S3-compatible storage
sloth-kubernetes login s3://bucket \
  --endpoint https://minio.example.com \
  --access-key-id minio \
  --secret-access-key minio123
```

### Configuration File

The backend configuration is stored in `~/.sloth-kubernetes/config`:

```json
{
  "backend_url": "s3://my-pulumi-state-bucket",
  "region": "us-east-1"
}
```

---

## `export-config`

Export cluster configuration stored in Pulumi state. Useful for recovering lost config files or auditing deployments.

### Usage

```bash
sloth-kubernetes export-config <stack-name> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-f, --format` | Output format: `lisp`, `json`, `yaml`, `meta` | `lisp` |
| `-o, --output` | Output file path | stdout |
| `--regenerate` | Regenerate Lisp from stored JSON | `false` |
| `--meta` | Also show deployment metadata | `false` |

### Examples

```bash
# Export config as Lisp (default)
sloth-kubernetes export-config production

# Export config as JSON
sloth-kubernetes export-config production --format json

# Export to a file
sloth-kubernetes export-config production --output recovered-config.lisp

# Regenerate Lisp from stored JSON
sloth-kubernetes export-config production --regenerate

# Export deployment metadata
sloth-kubernetes export-config production --format meta

# Export with metadata included
sloth-kubernetes export-config production --meta
```

### Output Formats

**Lisp (default):**
```lisp
(cluster
  (metadata
    (name "production")
    (environment "prod"))
  (providers
    (digitalocean
      (enabled true)
      (region "nyc3"))))
```

**JSON:**
```json
{
  "metadata": {
    "name": "production",
    "environment": "prod"
  },
  "providers": {
    "digitalocean": {
      "enabled": true,
      "region": "nyc3"
    }
  }
}
```

**Meta:**
```
Deployment Metadata
═══════════════════
Stack:        production
Deployed:     2026-01-05T10:30:00Z
Last Update:  2026-01-05T15:45:00Z
Nodes:        5 (3 masters, 2 workers)
K8s Version:  v1.29.0+rke2r1
```

---

## `pulumi`

Execute Pulumi operations using the embedded Automation API. No Pulumi CLI installation required.

### Usage

```bash
sloth-kubernetes pulumi <command> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `stack list` | List all stacks |
| `stack output` | Show stack outputs |
| `stack export` | Export stack state to JSON |
| `stack import` | Import stack state from JSON |
| `stack info` | Show detailed stack information |
| `stack delete` | Delete a stack |
| `stack select` | Select current stack |
| `stack current` | Show current selected stack |
| `stack rename` | Rename a stack |
| `stack cancel` | Cancel and unlock a stack |
| `preview` | Preview infrastructure changes |
| `refresh` | Refresh stack state from cloud |

### Examples

```bash
# List all stacks
sloth-kubernetes pulumi stack list

# Show stack outputs
sloth-kubernetes pulumi stack output --stack production

# Export stack state for backup
sloth-kubernetes pulumi stack export --stack production > backup.json

# Import stack state
sloth-kubernetes pulumi stack import --stack production < backup.json

# Preview changes
sloth-kubernetes pulumi preview --stack production

# Refresh state from cloud
sloth-kubernetes pulumi refresh --stack production

# Get detailed stack info
sloth-kubernetes pulumi stack info --stack production

# Cancel stuck operation
sloth-kubernetes pulumi stack cancel --stack production
```

---

## `config`

Manage cluster configuration files.

### Usage

```bash
sloth-kubernetes config <command> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `generate` | Generate example configuration file |

### Examples

```bash
# Generate example config
sloth-kubernetes config generate

# Generate with specific provider
sloth-kubernetes config generate --provider digitalocean

# Generate minimal config
sloth-kubernetes config generate --minimal

# Generate with output file
sloth-kubernetes config generate --output my-cluster.lisp
```

---

## `validate`

Validate cluster configuration before deployment.

### Usage

```bash
sloth-kubernetes validate [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --config` | Path to config file | `cluster.lisp` |
| `-v, --verbose` | Show detailed output | `false` |

### Validation Checks

- Lisp S-expression syntax
- Required fields and metadata
- Node distribution (masters/workers)
- Provider configuration
- Network and VPN settings
- DNS configuration
- Resource limits

### Examples

```bash
# Validate configuration
sloth-kubernetes validate --config cluster.lisp

# Validate with verbose output
sloth-kubernetes validate --config production.lisp --verbose
```

### Output

```
Validating: cluster.lisp
═══════════════════════════════════════════════════════════

✓ Lisp syntax valid
✓ Metadata complete
✓ Provider configuration valid
✓ Node pools configured (3 masters, 5 workers)
✓ Network settings valid
✓ VPN configuration valid
✓ DNS settings valid

Configuration is valid! Ready for deployment.
```

---

## `list`

List all deployed clusters.

### Usage

```bash
sloth-kubernetes list [flags]
```

### Examples

```bash
# List all clusters
sloth-kubernetes list

# Output as JSON
sloth-kubernetes list --json
```

### Output

```
Deployed Clusters
═════════════════════════════════════════════════════════════
NAME            NODES   STATUS    LAST UPDATE      PROVIDER
production      8       healthy   2h ago           digitalocean
staging         3       healthy   1d ago           linode
development     2       warning   5h ago           aws
```

---

## `status`

Show detailed cluster status and health information.

### Usage

```bash
sloth-kubernetes status <stack-name> [flags]
```

### Examples

```bash
# Show cluster status
sloth-kubernetes status my-cluster

# Show detailed status
sloth-kubernetes status my-cluster --verbose
```

### Output

```
Cluster Status: my-cluster
═══════════════════════════════════════════════════════════

Overview:
  Status:       Healthy
  K8s Version:  v1.29.0+rke2r1
  Nodes:        5 (3 ready, 0 not ready)
  Pods:         45 running, 2 pending

Nodes:
  NAME              STATUS   ROLES           VERSION
  master-1          Ready    control-plane   v1.29.0+rke2r1
  master-2          Ready    control-plane   v1.29.0+rke2r1
  master-3          Ready    control-plane   v1.29.0+rke2r1
  worker-1          Ready    worker          v1.29.0+rke2r1
  worker-2          Ready    worker          v1.29.0+rke2r1

Resources:
  CPU:      45% utilized (18/40 cores)
  Memory:   62% utilized (50/80 GB)
  Storage:  35% utilized (350/1000 GB)
```

---

## Environment Variables

Sloth Kubernetes supports these environment variables:

### Cloud Provider Credentials

| Variable | Description | Example |
|----------|-------------|---------|
| `DIGITALOCEAN_TOKEN` | DigitalOcean API token | `dop_v1_abc123...` |
| `LINODE_TOKEN` | Linode API token | `abc123...` |
| `AWS_ACCESS_KEY_ID` | AWS Access Key ID | `AKIA...` |
| `AWS_SECRET_ACCESS_KEY` | AWS Secret Access Key | `wJalr...` |
| `AWS_SESSION_TOKEN` | AWS Session Token (optional) | `FwoGZX...` |
| `AWS_REGION` | AWS Region | `us-east-1` |

### State Backend (S3)

| Variable | Description | Example |
|----------|-------------|---------|
| `PULUMI_BACKEND_URL` | S3 backend URL | `s3://my-bucket` |
| `PULUMI_CONFIG_PASSPHRASE` | Encryption passphrase | `mysecret` |
| `AWS_S3_ENDPOINT` | S3-compatible endpoint | `https://minio.example.com` |

### SSH Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `SSH_USER` | SSH username for nodes | `ubuntu` (AWS), `root` (DO) |
| `SSH_KEY_PATH` | Path to SSH private key | `~/.ssh/id_rsa` |

### Salt API

| Variable | Description | Example |
|----------|-------------|---------|
| `SALT_API_URL` | Salt API endpoint | `http://bastion:8000` |
| `SALT_USERNAME` | Salt API username | `saltapi` |
| `SALT_PASSWORD` | Salt API password | `saltapi123` |

### Debug & Logging

| Variable | Description | Example |
|----------|-------------|---------|
| `SLOTH_DEBUG` | Enable debug mode | `true` |
| `SLOTH_VERBOSE` | Verbose output | `true` |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success 🦥 |
| `1` | General error |
| `2` | Configuration error |
| `3` | Network error |
| `4` | API error |
| `5` | Timeout |

---

!!! quote "Sloth Wisdom 🦥"
    *"With great CLIs comes great responsibility... but take your time using them!"*
