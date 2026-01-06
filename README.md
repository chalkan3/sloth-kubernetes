# Sloth Kubernetes

<p align="center">
  <strong>Multi-Cloud Kubernetes Deployment Platform</strong><br>
  Single binary, zero external dependencies, enterprise-grade state management.
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#state-management">State Management</a> •
  <a href="#versioning-system">Versioning</a> •
  <a href="#manifest-registry">Manifest Registry</a> •
  <a href="#audit-logging">Audit Logging</a>
</p>

---

## Installation

**One-line install (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash
```

<details>
<summary>Other installation methods</summary>

**Install specific version:**
```bash
curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash -s v0.6.0
```

**Build from source:**
```bash
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes
go build -o sloth .
sudo mv sloth /usr/local/bin/
```

</details>

---

Deploy production-ready Kubernetes clusters across **AWS**, **Hetzner Cloud**, **DigitalOcean**, **Linode**, and **Azure** with embedded Pulumi, Salt, and kubectl. Features advanced **state management**, **configuration versioning**, **manifest tracking**, and comprehensive **audit logging**.

## Key Features

| Feature | Description |
|---------|-------------|
| **Multi-Cloud** | Deploy across AWS, Hetzner, DigitalOcean, Linode, Azure in a single cluster |
| **Zero Dependencies** | Single binary with embedded Pulumi, Salt, kubectl |
| **Lisp Configuration** | Human-readable S-expression configuration format |
| **State as Database** | Pulumi state stores complete deployment history |
| **Config Versioning** | Schema migrations, version tracking, rollback support |
| **Manifest Registry** | Track all Kubernetes manifests with history |
| **Audit Logging** | Complete audit trail of all changes |
| **WireGuard VPN** | Automatic mesh networking across providers |

---

## Quick Start

### 1. Set Cloud Credentials

```bash
# AWS
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# Hetzner Cloud
export HETZNER_TOKEN="your-token"

# DigitalOcean
export DIGITALOCEAN_TOKEN="your-token"

# Linode
export LINODE_TOKEN="your-token"

# Pulumi passphrase (for state encryption)
export PULUMI_CONFIG_PASSPHRASE="your-secure-passphrase"
```

### 2. Create Configuration

```lisp
; cluster.lisp
(cluster
  (metadata
    (name "my-cluster")
    (environment "production"))

  (providers
    (aws
      (enabled true)
      (region "us-east-1")
      (vpc
        (create true)
        (cidr "10.0.0.0/16"))))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    (masters
      (name "masters")
      (provider "aws")
      (count 3)
      (roles master etcd)
      (size "t3.medium"))
    (workers
      (name "workers")
      (provider "aws")
      (count 5)
      (roles worker)
      (size "t3.large")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

### 3. Deploy

```bash
sloth-kubernetes deploy production --config cluster.lisp --verbose
```

---

## Lisp Built-in Functions

Sloth Kubernetes provides a rich set of built-in functions for dynamic configuration.

### Environment Variables

| Function | Description | Example |
|----------|-------------|---------|
| `(env "VAR")` | Get environment variable | `(env "AWS_REGION")` |
| `(env "VAR" "default")` | Get env var with default | `(env "CLUSTER_ENV" "production")` |
| `(env-or "VAR" "fallback")` | Get env var or fallback | `(env-or "TOKEN" "default-token")` |
| `(env? "VAR")` | Check if env var exists | `(if (env? "DEBUG") "yes" "no")` |

### String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `(concat a b ...)` | Concatenate strings | `(concat "prefix-" (env "NAME"))` |
| `(format "template" args...)` | Format string | `(format "cluster-%s" (env "ENV"))` |
| `(upper str)` | Uppercase | `(upper "hello")` → `"HELLO"` |
| `(lower str)` | Lowercase | `(lower "HELLO")` → `"hello"` |
| `(trim str)` | Trim whitespace | `(trim "  hello  ")` → `"hello"` |
| `(replace str old new)` | Replace all occurrences | `(replace "foo-bar" "-" "_")` |
| `(substring str start end)` | Extract substring | `(substring "hello" 0 2)` → `"he"` |
| `(split str sep)` | Split into list | `(split "a,b,c" ",")` |
| `(join list sep)` | Join list to string | `(join (list "a" "b") "-")` |

### Control Flow

| Function | Description | Example |
|----------|-------------|---------|
| `(if cond then else)` | Conditional | `(if (env? "PROD") "prod" "dev")` |
| `(when cond body...)` | When true | `(when (env? "DEBUG") (set "log" "verbose"))` |
| `(unless cond body...)` | When false | `(unless (env? "CI") "local")` |
| `(cond (c1 v1) (c2 v2))` | Multiple conditions | `(cond ((eq x 1) "one") ((eq x 2) "two"))` |
| `(default val fallback)` | Default if nil/empty | `(default (env "SIZE") "t3.medium")` |
| `(or a b ...)` | First truthy | `(or (env "A") (env "B") "default")` |
| `(and a b ...)` | All truthy | `(and (env? "A") (env? "B"))` |
| `(not x)` | Negate | `(not false)` → `true` |

### Comparison

| Function | Description | Example |
|----------|-------------|---------|
| `(eq a b)` or `(= a b)` | Equal | `(eq (env "ENV") "prod")` |
| `(!= a b)` | Not equal | `(!= (env "ENV") "dev")` |
| `(< a b)` | Less than | `(< count 10)` |
| `(> a b)` | Greater than | `(> count 0)` |
| `(<= a b)` | Less or equal | `(<= count 100)` |
| `(>= a b)` | Greater or equal | `(>= count 3)` |

### Arithmetic

| Function | Description | Example |
|----------|-------------|---------|
| `(+ a b ...)` | Add | `(+ 1 2 3)` → `6` |
| `(- a b)` | Subtract | `(- 10 3)` → `7` |
| `(* a b ...)` | Multiply | `(* 2 3 4)` → `24` |
| `(/ a b)` | Divide | `(/ 10 2)` → `5` |
| `(mod a b)` | Modulo | `(mod 10 3)` → `1` |

### Encoding & Hashing

| Function | Description | Example |
|----------|-------------|---------|
| `(base64-encode str)` | Base64 encode | `(base64-encode "hello")` |
| `(base64-decode str)` | Base64 decode | `(base64-decode "aGVsbG8=")` |
| `(sha256 str)` | SHA256 hash | `(sha256 "password")` |

### UUID & Random

| Function | Description | Example |
|----------|-------------|---------|
| `(uuid)` | Generate UUID | `(uuid)` → `"550e8400-..."` |
| `(random-string len)` | Random string | `(random-string 16)` |

### Time & Date

| Function | Description | Example |
|----------|-------------|---------|
| `(now)` | Current RFC3339 time | `(now)` → `"2026-01-05T..."` |
| `(timestamp)` | Unix timestamp | `(timestamp)` → `1736078400` |
| `(date "format")` | Formatted date | `(date "2006-01-02")` |
| `(time "format")` | Formatted time | `(time "15:04:05")` |

### System

| Function | Description | Example |
|----------|-------------|---------|
| `(hostname)` | System hostname | `(hostname)` |
| `(user)` | Current user | `(user)` |
| `(home)` | Home directory | `(home)` → `"/home/user"` |
| `(cwd)` | Current directory | `(cwd)` |

### File Operations

| Function | Description | Example |
|----------|-------------|---------|
| `(read-file path)` | Read file content | `(read-file "~/.ssh/id_rsa.pub")` |
| `(file-exists? path)` | Check file exists | `(file-exists? "/etc/hosts")` |
| `(dirname path)` | Directory name | `(dirname "/a/b/c.txt")` → `"/a/b"` |
| `(basename path)` | Base name | `(basename "/a/b/c.txt")` → `"c.txt"` |
| `(expand-path path)` | Expand ~ and relative | `(expand-path "~/file")` |

### Variables

| Function | Description | Example |
|----------|-------------|---------|
| `(let ((v1 e1) ...) body)` | Local bindings | `(let ((x 1) (y 2)) (+ x y))` |
| `(set name value)` | Set variable | `(set "region" "us-east-1")` |
| `(var name)` | Get variable | `(var "region")` |

### List Operations

| Function | Description | Example |
|----------|-------------|---------|
| `(list a b ...)` | Create list | `(list 1 2 3)` |
| `(first lst)` | First element | `(first (list 1 2 3))` → `1` |
| `(rest lst)` | All but first | `(rest (list 1 2 3))` → `(2 3)` |
| `(nth lst n)` | Nth element | `(nth (list "a" "b") 1)` → `"b"` |
| `(len lst)` | Length | `(len (list 1 2 3))` → `3` |
| `(append a b)` | Concatenate lists | `(append (list 1) (list 2))` |
| `(range n)` | Generate 0..n-1 | `(range 5)` → `(0 1 2 3 4)` |

### Type Checking

| Function | Description | Example |
|----------|-------------|---------|
| `(string? x)` | Is string? | `(string? "hello")` → `true` |
| `(number? x)` | Is number? | `(number? 42)` → `true` |
| `(bool? x)` | Is boolean? | `(bool? true)` → `true` |
| `(list? x)` | Is list? | `(list? (list 1))` → `true` |
| `(nil? x)` | Is nil? | `(nil? nil)` → `true` |
| `(empty? x)` | Is empty? | `(empty? "")` → `true` |

### Type Conversion

| Function | Description | Example |
|----------|-------------|---------|
| `(to-string x)` | Convert to string | `(to-string 42)` → `"42"` |
| `(to-int x)` | Convert to integer | `(to-int "42")` → `42` |
| `(to-bool x)` | Convert to boolean | `(to-bool "true")` → `true` |

### Regex

| Function | Description | Example |
|----------|-------------|---------|
| `(match pattern str)` | Match and capture | `(match "v([0-9]+)" "v123")` |
| `(match? pattern str)` | Test if matches | `(match? "^prod" "production")` |

### Example: Dynamic Configuration

```lisp
(cluster
  (metadata
    (name (concat "cluster-" (env "CLUSTER_ENV" "dev")))
    (environment (env "CLUSTER_ENV" "development")))

  (providers
    (aws
      (enabled true)
      (region (env "AWS_REGION" "us-east-1"))
      (access-key-id (env "AWS_ACCESS_KEY_ID"))
      (secret-access-key (env "AWS_SECRET_ACCESS_KEY"))))

  (node-pools
    (masters
      (count (if (eq (env "CLUSTER_ENV") "production") 3 1))
      (size (default (env "MASTER_SIZE") "t3.medium"))))

  (kubernetes
    (version (concat "v1." (env "K8S_MINOR" "29") ".0+rke2r1"))))
```

### 4. Access Cluster

```bash
# Get kubeconfig
sloth-kubernetes kubeconfig production > ~/.kube/config

# Verify nodes (stack-aware command)
sloth-kubernetes kubectl production get nodes
```

---

## State Management

Sloth Kubernetes uses **Pulumi state as a database**, storing complete deployment information including configuration, metadata, and history. This enables powerful features like configuration recovery, audit trails, and rollback.

### What's Stored in State

```
┌─────────────────────────────────────────────────────────────────┐
│                     PULUMI STATE                                │
├─────────────────────────────────────────────────────────────────┤
│  configJson        │ Full cluster configuration (sanitized)    │
│  lispManifest      │ Original Lisp configuration file          │
│  deploymentMeta    │ Deployment metadata with versioning       │
│  kubeConfig        │ Kubernetes access configuration           │
│  nodes             │ Detailed node information                  │
│  salt_master       │ Salt API credentials and status           │
└─────────────────────────────────────────────────────────────────┘
```

### Export Configuration from State

```bash
# Export as Lisp (recover lost config files!)
sloth-kubernetes export-config production --format lisp

# Export as JSON
sloth-kubernetes export-config production --format json

# Export deployment metadata
sloth-kubernetes export-config production --format meta

# Save to file
sloth-kubernetes export-config production --format lisp --output recovered.lisp
```

### Backend Options

#### Local Backend (Development)

```bash
# State stored in ~/.pulumi (default)
sloth-kubernetes deploy --config cluster.lisp
```

#### S3 Backend (Production)

```bash
export PULUMI_BACKEND_URL="s3://my-bucket?region=us-east-1"
export PULUMI_CONFIG_PASSPHRASE="your-passphrase"
sloth-kubernetes deploy --config cluster.lisp
```

#### Persistent Configuration

```bash
mkdir -p ~/.sloth-kubernetes
cat > ~/.sloth-kubernetes/config << 'EOF'
PULUMI_BACKEND_URL=s3://my-bucket?region=us-east-1
PULUMI_CONFIG_PASSPHRASE=my-passphrase
EOF
```

---

## Versioning System

The versioning system (`pkg/versioning`) provides **schema versioning**, **migrations**, and **change tracking** for cluster configurations.

### DeploymentMetadata Structure

Every deployment stores comprehensive metadata:

```json
{
  "createdAt": "2026-01-05T11:42:41Z",
  "updatedAt": "2026-01-05T11:42:41Z",
  "lastDeployedAt": "2026-01-05T11:42:41Z",
  "deploymentId": "deploy-1-2026-01-05",
  "deploymentCount": 1,

  "configVersion": {
    "schema_version": "2.0",
    "config_hash": "02c6ff2ebc979d1026e3cbb66627afcf552202f9caa6aea71b5fa2cfa1d0e2f0",
    "created_at": "2026-01-05T11:42:41.69762Z",
    "updated_at": "2026-01-05T11:42:41.69762Z"
  },
  "schemaVersion": "2.0",
  "configChecksum": "fcf6d2664734ffeb623fade4b67a0846ff0cbfa32bcf50ce739513113c790cc1",

  "currentNodeCount": 3,
  "previousNodeCount": 0,
  "lastScaleOperation": "initial",

  "nodePoolsAdded": ["masters", "workers"],
  "nodePoolsRemoved": [],
  "nodePoolsScaled": [],

  "changeLog": [
    {
      "timestamp": "2026-01-05T11:42:41Z",
      "changeType": "cluster_created",
      "resourceId": "my-cluster",
      "description": "Initial cluster deployment with 3 nodes",
      "actor": "sloth-kubernetes"
    },
    {
      "timestamp": "2026-01-05T11:42:41Z",
      "changeType": "pool_added",
      "resourceId": "masters",
      "description": "Initial node pool 'masters' created",
      "actor": "sloth-kubernetes"
    }
  ],

  "previousDeployments": [],
  "stateSnapshotId": "state-deploy-1-2026-01-05-fcf6d266",
  "parentStateId": "",

  "slothVersion": "1.0.0",
  "pulumiVersion": "3.x"
}
```

### Schema Versioning

The system tracks schema versions and supports migrations:

```go
// Current schema version
const CurrentSchema = "2.0"

// Schema versions
SchemaV1 = "1.0"  // Original schema
SchemaV2 = "2.0"  // Added manifest_tracking, versioning fields
```

### ConfigVersion Structure

```go
type ConfigVersion struct {
    SchemaVersion SchemaVersion     // Schema version (e.g., "2.0")
    ConfigHash    string            // SHA256 hash of configuration
    ParentHash    string            // Hash of previous version
    CreatedAt     time.Time         // When version was created
    UpdatedAt     time.Time         // Last update time
    Migrations    []string          // Applied migrations
    Metadata      map[string]string // Additional metadata
}
```

### Change Log Entry Types

| Change Type | Description |
|-------------|-------------|
| `cluster_created` | Initial cluster deployment |
| `scale_up` | Node count increased |
| `scale_down` | Node count decreased |
| `config_updated` | Configuration changed |
| `pool_added` | New node pool created |
| `pool_removed` | Node pool deleted |
| `pool_scaled` | Node pool count changed |

---

## Manifest Registry

The manifest registry (`pkg/manifests`) tracks all Kubernetes manifests with **versioning**, **history**, and **diff capabilities**.

### Manifest Types

| Type | Description |
|------|-------------|
| `rke` | RKE cluster configuration |
| `rke2` | RKE2 cluster configuration |
| `k3s` | K3s cluster configuration |
| `argocd` | ArgoCD application |
| `helm` | Helm release |
| `kustomize` | Kustomize configuration |
| `raw` | Raw Kubernetes manifest |

### Manifest Status

| Status | Description |
|--------|-------------|
| `pending` | Registered but not applied |
| `applied` | Successfully applied to cluster |
| `failed` | Application failed |
| `deleted` | Marked as deleted |
| `out_of_sync` | Differs from live state |

### Manifest Structure

```go
type Manifest struct {
    ID           string            // Unique identifier
    Name         string            // Human-readable name
    Type         ManifestType      // Type of manifest
    Version      string            // Semantic version (v1, v2, etc.)
    Hash         string            // SHA256 of content
    Content      string            // YAML/JSON content
    Status       ManifestStatus    // Current status
    CreatedAt    time.Time         // Creation time
    UpdatedAt    time.Time         // Last update time
    AppliedAt    *time.Time        // When applied (if applicable)
    Metadata     map[string]string // Additional metadata
    Dependencies []string          // Dependent manifest IDs
    ParentHash   string            // Previous version hash
}
```

### Registry Operations

```go
// Create registry
registry := manifests.NewRegistry(50) // Max 50 versions in history

// Register manifest
manifest, err := registry.Register(
    "nginx-deployment",
    manifests.ManifestTypeRaw,
    `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 3`,
    map[string]string{"env": "production"},
)

// Get manifest
m, exists := registry.Get("nginx-deployment")

// List by type
argoApps := registry.ListByType(manifests.ManifestTypeArgoCD)

// List by status
pending := registry.ListByStatus(manifests.StatusPending)

// Update status
registry.UpdateStatus("nginx-deployment", manifests.StatusApplied)

// Get history
history, _ := registry.GetHistory("nginx-deployment")

// Compare registries
diff := registry1.Diff(registry2)
// diff.Added, diff.Removed, diff.Modified

// Export/Import
export, _ := registry.Export()
registry2.Import(export)

// JSON serialization
jsonData, _ := registry.ToJSON()
registry.FromJSON(jsonData)
```

---

## State Manager

The state manager (`internal/state`) provides **snapshot management**, **diff capabilities**, and **rollback support**.

### StateSnapshot Structure

```go
type StateSnapshot struct {
    ID            string                    // Unique snapshot ID
    DeploymentID  string                    // Associated deployment
    Timestamp     time.Time                 // When snapshot was taken
    Status        string                    // active, archived, rolled_back
    ConfigHash    string                    // Configuration hash
    ResourceCount int                       // Number of resources
    NodeCount     int                       // Number of nodes
    Resources     map[string]*ResourceState // Resource states
    Metadata      map[string]string         // Additional metadata
    ParentID      string                    // Previous snapshot ID
}
```

### State Manager Operations

```go
// Create manager
manager := state.NewInMemoryManager(100) // Max 100 snapshots

// Save snapshot
snapshot := &state.StateSnapshot{
    DeploymentID:  "deploy-1",
    Status:        "active",
    ConfigHash:    "abc123",
    ResourceCount: 10,
    NodeCount:     3,
}
manager.SaveSnapshot(snapshot)

// Get snapshots
latest, _ := manager.GetLatestSnapshot()
snapshot, _ := manager.GetSnapshot("snapshot-id")

// List with filters
filter := &state.SnapshotFilter{
    Status:       "active",
    DeploymentID: "deploy-1",
    Limit:        10,
}
snapshots := manager.ListSnapshots(filter)

// Diff snapshots
diff, _ := manager.DiffSnapshots("old-id", "new-id")
// diff.PoolChanges, diff.ResourceChanges

// Get rollback points
rollbackPoints := manager.GetRollbackPoints()

// Rollback
newSnapshot, _ := manager.Rollback("target-snapshot-id")

// Export/Import
export, _ := manager.Export()
manager.Import(export)
```

---

## Audit Logging

The audit logger (`internal/audit`) provides comprehensive **event tracking** for all deployment operations.

### Event Types

| Event Type | Description |
|------------|-------------|
| `deployment` | Deployment-related events |
| `configuration` | Configuration changes |
| `manifest` | Manifest operations |
| `rollback` | Rollback events |
| `state` | State changes |
| `error` | Error events |

### Event Actions

| Action | Description |
|--------|-------------|
| `create` | Resource created |
| `update` | Resource updated |
| `delete` | Resource deleted |
| `apply` | Resource applied |
| `rollback` | Rollback performed |
| `validate` | Validation occurred |
| `migrate` | Migration executed |

### Event Severity

| Severity | Description |
|----------|-------------|
| `info` | Informational |
| `warning` | Warning condition |
| `error` | Error occurred |
| `critical` | Critical issue |

### AuditEvent Structure

```go
type AuditEvent struct {
    ID            string        // Unique event ID
    Timestamp     time.Time     // When event occurred
    Type          EventType     // Event type
    Action        EventAction   // Action taken
    Severity      EventSeverity // Event severity
    ResourceID    string        // Affected resource
    ResourceType  string        // Type of resource
    Actor         string        // Who triggered event
    Description   string        // Human-readable description
    OldValue      interface{}   // Previous value (for updates)
    NewValue      interface{}   // New value
    Metadata      map[string]string
    CorrelationID string        // Links related events
    Duration      time.Duration // Operation duration
    Success       bool          // Did action succeed
    ErrorMessage  string        // Error details if failed
}
```

### Logger Operations

```go
// Create logger
logger := audit.NewInMemoryLogger(10000) // Max 10000 events

// Log deployment event
event, _ := logger.LogDeployment(
    "cluster-1",           // Resource ID
    "admin",               // Actor
    "Deployed production", // Description
    audit.ActionApply,     // Action
    true,                  // Success
    map[string]string{"env": "prod"},
)

// Log configuration change
logger.LogConfiguration(
    "config-1",
    "admin",
    oldConfig,  // Old value
    newConfig,  // New value
    nil,
)

// Log manifest operation
logger.LogManifest(
    "nginx-deployment",
    "system",
    "Applied nginx deployment",
    audit.ActionApply,
    true,
    nil,
)

// Log rollback
logger.LogRollback(
    "cluster-1",
    "admin",
    "v2",      // From version
    "v1",      // To version
    true,      // Success
    nil,
)

// Log error
logger.LogError(
    "cluster-1",
    "system",
    "Connection timeout",
    map[string]string{"retry": "3"},
)

// Query events
filter := &audit.AuditFilter{
    Types:       []audit.EventType{audit.EventTypeDeployment},
    Actions:     []audit.EventAction{audit.ActionApply},
    Actor:       "admin",
    SuccessOnly: true,
    Limit:       100,
}
events := logger.Query(filter)

// Get summary
summary := logger.GetSummary()
// summary.TotalEvents, summary.EventsByType, summary.SuccessCount, etc.

// Get correlated events
related := logger.GetByCorrelation("correlation-id")

// Prune old events
pruned := logger.Prune(time.Now().Add(-30 * 24 * time.Hour))

// Export/Import
export, _ := logger.Export()
logger.Import(export)
```

### Audit Summary

```go
type AuditSummary struct {
    TotalEvents      int
    EventsByType     map[EventType]int
    EventsByAction   map[EventAction]int
    EventsBySeverity map[EventSeverity]int
    SuccessCount     int
    FailureCount     int
    FirstEvent       *time.Time
    LastEvent        *time.Time
    AverageDuration  time.Duration
    TopActors        []ActorStat
    TopResources     []ResourceStat
}
```

---

## CLI Commands

### Infrastructure Management

| Command | Description |
|---------|-------------|
| `deploy <stack> --config <file>` | Deploy cluster |
| `destroy <stack>` | Destroy cluster |
| `refresh <stack>` | Sync state with cloud |
| `preview <stack> --config <file>` | Preview changes |

### Stack Management

| Command | Description |
|---------|-------------|
| `stacks list` | List all stacks |
| `stacks info <name>` | Show stack details |
| `stacks output <name>` | Show stack outputs |
| `stacks remove <name>` | Delete a stack |
| `stacks cancel <name>` | Unlock stuck stack |

### Configuration Export

| Command | Description |
|---------|-------------|
| `export-config <stack> --format lisp` | Export as Lisp |
| `export-config <stack> --format json` | Export as JSON |
| `export-config <stack> --format meta` | Export metadata |
| `export-config <stack> --output <file>` | Save to file |

### Node Management

| Command | Description |
|---------|-------------|
| `nodes list <stack>` | List cluster nodes |
| `nodes ssh <node>` | SSH to node |
| `nodes drain <node>` | Drain node |
| `nodes cordon <node>` | Cordon node |

### Salt Management

| Command | Description |
|---------|-------------|
| `salt login` | Authenticate with Salt |
| `salt ping` | Ping all nodes |
| `salt cmd "<command>"` | Run command on all nodes |
| `salt system disk` | Check disk usage |
| `salt system memory` | Check memory usage |

### Kubernetes (Embedded kubectl - Stack-Aware)

All kubectl commands require the stack name as the first argument:

| Command | Description |
|---------|-------------|
| `kubectl <stack> get nodes` | List nodes |
| `kubectl <stack> get pods -A` | List all pods |
| `kubectl <stack> apply -f <file>` | Apply manifest |
| `kubectl <stack> logs <pod>` | View pod logs |

### Health, Backup, Upgrade, Benchmark (Stack-Aware)

| Command | Description |
|---------|-------------|
| `health <stack>` | Check cluster health |
| `backup status <stack>` | Check Velero status |
| `backup create <stack> <name>` | Create backup |
| `upgrade versions <stack>` | List available versions |
| `upgrade apply <stack> --to <version>` | Upgrade cluster |
| `benchmark run <stack>` | Run benchmarks |
| `argocd install <stack>` | Install ArgoCD |

---

## Configuration Examples

### AWS Production Cluster

```lisp
(cluster
  (metadata
    (name "aws-production")
    (environment "production")
    (owner "platform-team"))

  (providers
    (aws
      (enabled true)
      (region "us-east-1")
      (vpc
        (create true)
        (cidr "10.0.0.0/16"))))

  (network
    (mode "wireguard")
    (pod-cidr "10.42.0.0/16")
    (service-cidr "10.43.0.0/16")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)
      (subnet-cidr "10.8.0.0/24")
      (port 51820)))

  (security
    (bastion
      (enabled true)
      (provider "aws")
      (region "us-east-1")
      (size "t3.micro")))

  (node-pools
    (masters
      (name "masters")
      (provider "aws")
      (region "us-east-1")
      (count 3)
      (roles master etcd)
      (size "t3.large"))
    (workers
      (name "workers")
      (provider "aws")
      (region "us-east-1")
      (count 10)
      (roles worker)
      (size "t3.xlarge")
      (spot-instance true)))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (network-plugin "calico"))

  (addons
    (salt
      (enabled true)
      (api-enabled true)
      (api-port 8000)
      (secure-auth true)
      (auto-join true))))
```

### Multi-Cloud HA Cluster (AWS + Hetzner)

```lisp
(cluster
  (metadata
    (name "aws-hetzner-cluster")
    (environment "production"))

  (providers
    (aws
      (enabled true)
      (region "us-east-1")
      (vpc
        (create true)
        (cidr "10.0.0.0/16")))
    (hetzner
      (enabled true)
      (location "nbg1")))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)
      (subnet-cidr "10.8.0.0/24")))

  (security
    (bastion
      (enabled true)
      (provider "hetzner")
      (location "nbg1")
      (size "cx22")))

  (node-pools
    (aws-master
      (name "aws-master")
      (provider "aws")
      (region "us-east-1")
      (count 1)
      (roles master etcd)
      (size "t3.medium"))
    (aws-workers
      (name "aws-workers")
      (provider "aws")
      (region "us-east-1")
      (count 2)
      (roles worker)
      (size "t3.medium"))
    (hetzner-workers
      (name "hetzner-workers")
      (provider "hetzner")
      (location "nbg1")
      (count 2)
      (roles worker)
      (size "cx22")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (network-plugin "calico")))
```

### Multi-Cloud HA Cluster (Legacy)

```lisp
(cluster
  (metadata
    (name "multi-cloud-ha")
    (environment "production"))

  (providers
    (aws
      (enabled true)
      (region "us-east-1"))
    (digitalocean
      (enabled true)
      (region "nyc3"))
    (linode
      (enabled true)
      (region "us-east")))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    (aws-masters
      (name "aws-masters")
      (provider "aws")
      (count 1)
      (roles master etcd)
      (size "t3.medium"))
    (do-masters
      (name "do-masters")
      (provider "digitalocean")
      (count 1)
      (roles master etcd)
      (size "s-2vcpu-4gb"))
    (linode-masters
      (name "linode-masters")
      (provider "linode")
      (count 1)
      (roles master etcd)
      (size "g6-standard-2"))
    (workers
      (name "workers")
      (provider "aws")
      (count 5)
      (roles worker)
      (size "t3.large")))

  (kubernetes
    (distribution "rke2")
    (high-availability true)))
```

### Development Cluster

```lisp
(cluster
  (metadata
    (name "dev-cluster")
    (environment "development"))

  (providers
    (digitalocean
      (enabled true)
      (region "nyc3")))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    (single-node
      (name "dev-node")
      (provider "digitalocean")
      (count 1)
      (roles master etcd worker)
      (size "s-4vcpu-8gb")))

  (kubernetes
    (distribution "k3s")))
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SLOTH KUBERNETES                                    │
│                        (Single Binary CLI)                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Deploy     │  │   Destroy    │  │   Export     │  │   Kubectl    │    │
│  │   Command    │  │   Command    │  │   Config     │  │   Commands   │    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘    │
│         │                 │                 │                 │            │
│  ┌──────┴─────────────────┴─────────────────┴─────────────────┴──────┐     │
│  │                     ORCHESTRATOR LAYER                            │     │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │     │
│  │  │  Versioning │  │  Manifest   │  │   Audit     │               │     │
│  │  │   System    │  │  Registry   │  │   Logger    │               │     │
│  │  └─────────────┘  └─────────────┘  └─────────────┘               │     │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │     │
│  │  │   State     │  │ Deployment  │  │   Config    │               │     │
│  │  │  Manager    │  │  Metadata   │  │   Loader    │               │     │
│  │  └─────────────┘  └─────────────┘  └─────────────┘               │     │
│  └───────────────────────────┬───────────────────────────────────────┘     │
│                              │                                             │
│  ┌───────────────────────────┴───────────────────────────────────────┐     │
│  │                    PULUMI AUTOMATION API                          │     │
│  │              (Infrastructure State & Orchestration)               │     │
│  └───────────────────────────┬───────────────────────────────────────┘     │
│                              │                                             │
│  ┌───────────────────────────┴───────────────────────────────────────┐     │
│  │                      PROVIDER LAYER                               │     │
│  │  ┌───────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │     │
│  │  │  AWS  │  │ Hetzner │  │   DO    │  │ Linode  │  │  Azure  │  │     │
│  │  └───────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘  │     │
│  └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐     │
│  │                      COMPONENT LAYER                              │     │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐ │     │
│  │  │  Nodes  │  │WireGuard│  │  RKE2   │  │  Salt   │  │  ArgoCD │ │     │
│  │  │         │  │  VPN    │  │  K3s    │  │ Master  │  │         │ │     │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘ │     │
│  └───────────────────────────────────────────────────────────────────┘     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         STATE STORAGE                                       │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐          │
│  │   Local State    │  │    S3 Backend    │  │   Pulumi Cloud   │          │
│  │   (~/.pulumi)    │  │   (Production)   │  │    (Optional)    │          │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘          │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Package Reference

### pkg/versioning

Configuration versioning with schema migrations.

```go
import "github.com/chalkan3/sloth-kubernetes/pkg/versioning"

vm := versioning.NewVersionManager()
version, _ := vm.CreateVersion(config, parentHash)
migrated, _, _ := vm.Migrate(config, versioning.SchemaV1, versioning.SchemaV2)
```

### pkg/manifests

Kubernetes manifest registry with versioning.

```go
import "github.com/chalkan3/sloth-kubernetes/pkg/manifests"

registry := manifests.NewRegistry(50)
manifest, _ := registry.Register("name", manifests.ManifestTypeRaw, content, nil)
```

### internal/state

State snapshot management with rollback support.

```go
import "github.com/chalkan3/sloth-kubernetes/internal/state"

manager := state.NewInMemoryManager(100)
manager.SaveSnapshot(snapshot)
diff, _ := manager.DiffSnapshots(oldID, newID)
```

### internal/audit

Comprehensive audit logging.

```go
import "github.com/chalkan3/sloth-kubernetes/internal/audit"

logger := audit.NewInMemoryLogger(10000)
logger.LogDeployment(resourceID, actor, description, action, success, metadata)
```

---

## Development

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./pkg/versioning/...
go test -v ./pkg/manifests/...
go test -v ./internal/state/...
go test -v ./internal/audit/...

# Build
go build -o sloth-kubernetes .

# Run E2E tests (requires cloud credentials)
go test -tags=e2e ./test/e2e/ -timeout 30m
```

---

## Troubleshooting

### Stack is Locked

```bash
sloth-kubernetes stacks cancel <stack-name>
```

### View Detailed Logs

```bash
sloth-kubernetes deploy --config cluster.lisp --verbose
```

### Export Lost Configuration

```bash
# Recover Lisp config from state
sloth-kubernetes export-config <stack> --format lisp --output recovered.lisp
```

### View Deployment History

```bash
sloth-kubernetes export-config <stack> --format meta
```

### Force Backend Override

```bash
export PULUMI_BACKEND_URL="s3://my-bucket?region=us-east-1"
sloth-kubernetes stacks list
```

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit changes: `git commit -m "Add my feature"`
4. Push to branch: `git push origin feature/my-feature`
5. Open a Pull Request

---

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

---

<p align="center">
  <strong>Built with Go, Pulumi, Salt, and WireGuard</strong>
</p>
