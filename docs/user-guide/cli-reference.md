# 🦥 CLI Reference

Complete reference for all Sloth Kubernetes commands. Slowly, but thoroughly documented!

---

## Global Flags

These flags work with all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--help, -h` | Show help for command | - |
| `--version, -v` | Show version | - |
| `--debug` | Enable debug logging | `false` |
| `--config, -c` | Path to config file | `cluster.yaml` |

---

## Commands Overview

```bash
sloth-kubernetes [command] [flags]
```

Available Commands:

- [`deploy`](#deploy) - Deploy a Kubernetes cluster 🦥
- [`destroy`](#destroy) - Destroy a cluster 🦥
- [`nodes`](#nodes) - Manage cluster nodes 🦥
- [`vpn`](#vpn) - Manage WireGuard VPN 🦥
- [`stacks`](#stacks) - Manage Pulumi stacks 🦥
- [`kubeconfig`](#kubeconfig) - Generate kubeconfig 🦥
- [`version`](#version) - Show version info 🦥

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
| `--config, -c` | string | Path to cluster config file | Yes | `cluster.yaml` |
| `--dry-run` | bool | Preview changes without applying | No | `false` |
| `--auto-approve` | bool | Skip confirmation prompt | No | `false` |
| `--parallel` | int | Max parallel operations | No | `10` |
| `--timeout` | duration | Deployment timeout | No | `30m` |

### Examples

```bash
# Deploy with default config 🦥
sloth-kubernetes deploy

# Deploy with custom config
sloth-kubernetes deploy --config production.yaml

# Dry run (preview changes)
sloth-kubernetes deploy --dry-run

# Auto-approve without confirmation
sloth-kubernetes deploy --auto-approve

# Deploy with timeout
sloth-kubernetes deploy --timeout 45m
```

### Output

```
🦥 Sloth Kubernetes Deployment
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
| `--config, -c` | string | Path to cluster config file | Yes | `cluster.yaml` |
| `--force, -f` | bool | Skip confirmation prompt | No | `false` |
| `--remove-state` | bool | Also remove state files | No | `false` |

### Examples

```bash
# Destroy cluster 🦥
sloth-kubernetes destroy

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

- `nodes list` - List all nodes 🦥
- `nodes add` - Add nodes to cluster 🦥
- `nodes remove` - Remove nodes from cluster 🦥
- `nodes drain` - Drain a node for maintenance 🦥

### `nodes list`

List all nodes in the cluster.

```bash
sloth-kubernetes nodes list [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config, -c` | string | Cluster config | `cluster.yaml` |
| `--output, -o` | string | Output format: `table`, `json`, `yaml` | `table` |

**Example:**

```bash
# List nodes 🦥
sloth-kubernetes nodes list

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
# Add 2 workers to linode-workers pool 🦥
sloth-kubernetes nodes add --pool linode-workers --count 2

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
# Remove a node (with graceful drain) 🦥
sloth-kubernetes nodes remove do-worker-2

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
# Drain node for maintenance 🦥
sloth-kubernetes nodes drain do-worker-1
```

---

## `vpn`

Manage WireGuard VPN configuration and client access.

### Subcommands

- `vpn status` - Show VPN status 🦥
- `vpn client-config` - Generate client config 🦥
- `vpn add-client` - Add new VPN client 🦥
- `vpn remove-client` - Remove VPN client 🦥

### `vpn status`

Show WireGuard VPN status and connected clients.

```bash
sloth-kubernetes vpn status [flags]
```

**Example:**

```bash
# Check VPN status 🦥
sloth-kubernetes vpn status
```

**Output:**

```
🦥 WireGuard VPN Status

Server: 203.0.113.10 (nyc3)
Subnet: 10.8.0.0/24
Port: 51820

Connected Nodes:
  do-master-1      10.8.0.2   ✓ Connected
  linode-master-1  10.8.0.3   ✓ Connected
  linode-master-2  10.8.0.4   ✓ Connected
  do-worker-1      10.8.0.10  ✓ Connected
  linode-worker-1  10.8.0.11  ✓ Connected

Clients:
  my-laptop        10.8.0.100 ✓ Connected
```

### `vpn client-config`

Generate WireGuard client configuration.

```bash
sloth-kubernetes vpn client-config --name CLIENT_NAME [flags]
```

**Flags:**

| Flag | Type | Description | Required |
|------|------|-------------|----------|
| `--name` | string | Client name | Yes |
| `--output, -o` | string | Output file path | No |

**Example:**

```bash
# Generate client config 🦥
sloth-kubernetes vpn client-config --name my-laptop

# Save to file
sloth-kubernetes vpn client-config --name my-laptop -o laptop.conf
```

**Output:**

```
🦥 WireGuard Client Configuration

[Interface]
PrivateKey = <generated-private-key>
Address = 10.8.0.100/24
DNS = 10.8.0.1

[Peer]
PublicKey = <server-public-key>
Endpoint = 203.0.113.10:51820
AllowedIPs = 10.8.0.0/24, 10.10.0.0/16, 10.11.0.0/16
PersistentKeepalive = 25

Saved to: my-laptop.conf
```

---

## `stacks`

Manage Pulumi stacks for cluster state.

### Subcommands

- `stacks list` - List all stacks 🦥
- `stacks state list` - List stack resources 🦥
- `stacks state delete` - Delete specific resources 🦥

### `stacks list`

List all Pulumi stacks.

```bash
sloth-kubernetes stacks list
```

**Example:**

```bash
# List stacks 🦥
sloth-kubernetes stacks list
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
| `--config, -c` | string | Cluster config | `cluster.yaml` |
| `--type` | string | Filter by resource type | - |

**Example:**

```bash
# List all resources 🦥
sloth-kubernetes stacks state list

# Filter by type
sloth-kubernetes stacks state list --type digitalocean:Droplet
```

---

## `kubeconfig`

Generate kubeconfig for cluster access.

### Usage

```bash
sloth-kubernetes kubeconfig [flags]
```

### Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config, -c` | string | Cluster config | `cluster.yaml` |
| `--output, -o` | string | Output file | stdout |

### Examples

```bash
# Print kubeconfig 🦥
sloth-kubernetes kubeconfig

# Save to file
sloth-kubernetes kubeconfig -o ~/.kube/config

# Use immediately with kubectl
export KUBECONFIG=$(sloth-kubernetes kubeconfig -o /tmp/kubeconfig.yaml)
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

## Environment Variables

Sloth Kubernetes supports these environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `DIGITALOCEAN_TOKEN` | DigitalOcean API token | `dop_v1_abc123...` |
| `LINODE_TOKEN` | Linode API token | `abc123...` |
| `SLOTH_DEBUG` | Enable debug mode | `true` |
| `SLOTH_STATE_DIR` | State directory | `~/.sloth` |

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
