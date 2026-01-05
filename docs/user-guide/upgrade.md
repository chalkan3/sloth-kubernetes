---
title: Cluster Upgrades
description: Upgrade Kubernetes cluster versions with multiple strategies
sidebar_position: 6
---

# Cluster Upgrades

sloth-kubernetes provides comprehensive cluster upgrade capabilities with multiple strategies, rollback support, and detailed planning.

## Overview

The upgrade system supports:
- **Multiple strategies**: Rolling, blue-green, canary, in-place
- **Pre-flight checks**: Cluster health, etcd backup, node readiness
- **Automatic rollback**: On failure detection
- **Node filtering**: Upgrade specific nodes only
- **Dry-run mode**: Preview changes without applying

## Commands

### upgrade plan

Create an upgrade plan without executing it.

```bash
sloth-kubernetes upgrade plan --to v1.29.0
```

The plan shows:
- Current cluster state
- Nodes to be upgraded and their order
- Pre-flight checks to be performed
- Estimated downtime per node
- Potential risks identified

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target Kubernetes version (required) | - |
| `--strategy` | Upgrade strategy | `rolling` |
| `--kubeconfig` | Path to kubeconfig file | - |

### upgrade apply

Execute the upgrade plan on the cluster.

```bash
# Execute upgrade with rolling strategy
sloth-kubernetes upgrade apply --to v1.29.0 --strategy rolling

# Dry-run to see what would happen
sloth-kubernetes upgrade apply --to v1.29.0 --dry-run

# Upgrade specific nodes only
sloth-kubernetes upgrade apply --to v1.29.0 --nodes master-1,worker-1

# Force upgrade without confirmation
sloth-kubernetes upgrade apply --to v1.29.0 --force
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target Kubernetes version (required) | - |
| `--strategy` | Upgrade strategy | `rolling` |
| `--dry-run` | Simulate without making changes | `false` |
| `--verbose`, `-v` | Show verbose output | `false` |
| `--force` | Skip confirmation prompts | `false` |
| `--kubeconfig` | Path to kubeconfig file | - |
| `--nodes` | Specific nodes to upgrade (comma-separated) | - |
| `--backup-dir` | Directory for etcd backups | `/var/lib/sloth-kubernetes/backups` |
| `--timeout` | Timeout in seconds per node | `600` |

### upgrade rollback

Rollback to the previous Kubernetes version.

```bash
sloth-kubernetes upgrade rollback

# Force rollback without confirmation
sloth-kubernetes upgrade rollback --force
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Skip confirmation prompts | `false` |
| `--verbose`, `-v` | Show verbose output | `false` |
| `--kubeconfig` | Path to kubeconfig file | - |

### upgrade versions

List current cluster version and available upgrade targets.

```bash
sloth-kubernetes upgrade versions
```

**Example output:**

```
Current Version:
  v1.28.2+rke2r1

Available Upgrade Targets:
  v1.29.0+rke2r1
  v1.29.1+rke2r1

All Supported Versions:
  v1.27.5+rke2r1 (downgrade)
  v1.28.2+rke2r1 (current)
  v1.29.0+rke2r1 (upgrade available)
  v1.29.1+rke2r1 (upgrade available)
```

### upgrade status

Show the status of an ongoing or recent upgrade.

```bash
sloth-kubernetes upgrade status
```

**Example output:**

```
Cluster Version: v1.28.2+rke2r1

Node Status:
NODE                           VERSION         STATUS          ROLE
---------------------------------------------------------------------------
master-1                       v1.28.2+rke2r1  Ready           master
master-2                       v1.28.2+rke2r1  Ready           master
worker-1                       v1.28.2+rke2r1  Ready           worker
worker-2                       v1.28.2+rke2r1  Ready           worker
```

---

## Upgrade Strategies

### Rolling (Default)

Upgrades one node at a time. Safest strategy with minimal disruption.

```bash
sloth-kubernetes upgrade apply --to v1.29.0 --strategy rolling
```

**Process:**
1. Run pre-flight checks
2. Create etcd backup
3. For each node:
   - Cordon node
   - Drain workloads
   - Upgrade Kubernetes components
   - Uncordon node
   - Wait for node to be ready
4. Run post-upgrade validation

**Best for:** Production clusters where availability is critical.

### Blue-Green

Creates new nodes with the new version, migrates workloads, then removes old nodes.

```bash
sloth-kubernetes upgrade apply --to v1.29.0 --strategy blue-green
```

**Process:**
1. Provision new nodes with target version
2. Migrate workloads to new nodes
3. Validate new nodes are healthy
4. Remove old nodes

**Best for:** Zero-downtime upgrades when you have capacity for temporary extra nodes.

### Canary

Upgrades a subset of nodes first, validates, then proceeds with the rest.

```bash
sloth-kubernetes upgrade apply --to v1.29.0 --strategy canary
```

**Process:**
1. Upgrade 1-2 nodes as canaries
2. Run workloads on canary nodes for validation period
3. If successful, proceed with remaining nodes
4. If issues detected, rollback canary nodes

**Best for:** Risk-averse upgrades where you want to test before full rollout.

### In-Place

Upgrades all nodes simultaneously. Fastest but riskiest.

```bash
sloth-kubernetes upgrade apply --to v1.29.0 --strategy in-place
```

**Process:**
1. Run pre-flight checks
2. Create etcd backup
3. Upgrade all nodes at once
4. Wait for cluster to stabilize

**Best for:** Development/test clusters where downtime is acceptable.

---

## Pre-flight Checks

Before any upgrade, the following checks are performed:

1. **Cluster Health** - Verify all nodes are Ready
2. **etcd Backup** - Create backup of etcd data
3. **Pod Status** - Ensure no stuck or failing pods
4. **Resource Availability** - Check for sufficient resources
5. **API Server** - Verify API server is responsive
6. **Certificate Expiry** - Check certificates aren't expiring soon

---

## Examples

### Standard Production Upgrade

```bash
# 1. Check available versions
sloth-kubernetes upgrade versions

# 2. Create upgrade plan
sloth-kubernetes upgrade plan --to v1.29.0

# 3. Review the plan, then apply
sloth-kubernetes upgrade apply --to v1.29.0 --strategy rolling

# 4. Verify upgrade
sloth-kubernetes upgrade status
```

### Upgrade Specific Nodes

```bash
# Upgrade only master nodes first
sloth-kubernetes upgrade apply --to v1.29.0 --nodes master-1,master-2,master-3

# Then upgrade workers
sloth-kubernetes upgrade apply --to v1.29.0 --nodes worker-1,worker-2,worker-3
```

### Dry-Run Before Upgrade

```bash
# See what would happen without making changes
sloth-kubernetes upgrade apply --to v1.29.0 --dry-run
```

**Example dry-run output:**

```
[DRY-RUN] Would execute the following upgrade:

Planned Steps:
  1. Run pre-flight checks
  2. Create etcd backup
  3. Cordon node master-1
  4. Drain node master-1
  5. Upgrade node master-1 to v1.29.0
  6. Uncordon node master-1
  7. Wait for node master-1 to be ready
  ...
```

### Rollback After Failed Upgrade

```bash
# If upgrade fails, rollback
sloth-kubernetes upgrade rollback

# Force rollback without confirmation
sloth-kubernetes upgrade rollback --force
```

---

## Troubleshooting

### Upgrade Stuck

If an upgrade appears stuck:

```bash
# Check upgrade status
sloth-kubernetes upgrade status

# Check node status directly
sloth-kubernetes kubectl get nodes

# Check for stuck pods
sloth-kubernetes kubectl get pods --all-namespaces | grep -v Running
```

### Node Won't Drain

If a node won't drain during upgrade:

```bash
# Check what's blocking drain
sloth-kubernetes kubectl get pods -o wide | grep <node-name>

# Force drain if necessary (may cause data loss)
sloth-kubernetes kubectl drain <node-name> --force --ignore-daemonsets --delete-emptydir-data
```

### Rollback Failed

If automatic rollback fails:

```bash
# Manual rollback using etcd backup
sloth-kubernetes nodes ssh master-1
sudo rke2-killall.sh
sudo systemctl stop rke2-server
# Restore etcd from backup directory
```
