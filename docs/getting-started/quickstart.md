# Quick Start Guide

Deploy your first Kubernetes cluster in under 5 minutes with sloth-kubernetes.

---

## TL;DR (For the Impatient)

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash

# 2. Set credentials (pick your provider)
export DIGITALOCEAN_TOKEN="your-token"        # DigitalOcean
# OR
export LINODE_TOKEN="your-token"              # Linode
# OR
export AWS_ACCESS_KEY_ID="xxx" AWS_SECRET_ACCESS_KEY="yyy"  # AWS

# 3. Deploy (downloads example config automatically)
curl -sO https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/examples/minimal-do.lisp
sloth-kubernetes deploy my-cluster --config minimal-do.lisp

# 4. Use your cluster
sloth-kubernetes kubectl my-cluster get nodes
```

---

## Step-by-Step Guide

### Step 1: Install sloth-kubernetes

**One-liner (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash
```

**Verify installation:**

```bash
sloth-kubernetes version
# Output: sloth-kubernetes v0.6.1 (commit: abc123)
```

> **Troubleshooting:** If you get "command not found", add `/usr/local/bin` to your PATH or restart your terminal.

---

### Step 2: Configure Cloud Credentials

Choose your preferred cloud provider and set the required environment variables:

#### DigitalOcean (Easiest to Start)

```bash
# Get your token from: https://cloud.digitalocean.com/account/api/tokens
export DIGITALOCEAN_TOKEN="dop_v1_your_token_here"
```

#### Linode

```bash
# Get your token from: https://cloud.linode.com/profile/tokens
export LINODE_TOKEN="your_linode_token_here"
```

#### AWS

```bash
# Get credentials from: IAM Console > Users > Security credentials
export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export AWS_REGION="us-east-1"
```

#### State Encryption (Required)

```bash
# This passphrase encrypts your cluster state - don't lose it!
export PULUMI_CONFIG_PASSPHRASE="choose-a-strong-passphrase"
```

> **Tip:** Add these exports to your `~/.bashrc` or `~/.zshrc` for persistence.

---

### Step 3: Create Your First Cluster Configuration

Create a file named `cluster.lisp` with the configuration for your chosen provider:

#### DigitalOcean Configuration

```lisp
; cluster.lisp - Minimal DigitalOcean cluster
(cluster
  (metadata
    (name "my-first-cluster")
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
    (control-plane
      (name "control-plane")
      (provider "digitalocean")
      (region "nyc3")
      (count 1)
      (roles master etcd)
      (size "s-2vcpu-4gb"))
    (workers
      (name "workers")
      (provider "digitalocean")
      (region "nyc3")
      (count 2)
      (roles worker)
      (size "s-2vcpu-4gb")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

#### Linode Configuration

```lisp
; cluster.lisp - Minimal Linode cluster
(cluster
  (metadata
    (name "my-first-cluster")
    (environment "development"))

  (providers
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
    (control-plane
      (name "control-plane")
      (provider "linode")
      (region "us-east")
      (count 1)
      (roles master etcd)
      (size "g6-standard-2"))
    (workers
      (name "workers")
      (provider "linode")
      (region "us-east")
      (count 2)
      (roles worker)
      (size "g6-standard-2")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

#### AWS Configuration

```lisp
; cluster.lisp - Minimal AWS cluster
(cluster
  (metadata
    (name "my-first-cluster")
    (environment "development"))

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
    (control-plane
      (name "control-plane")
      (provider "aws")
      (region "us-east-1")
      (count 1)
      (roles master etcd)
      (size "t3.medium"))
    (workers
      (name "workers")
      (provider "aws")
      (region "us-east-1")
      (count 2)
      (roles worker)
      (size "t3.medium")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

---

### Step 4: Deploy the Cluster

Run the deploy command with your **stack name** and configuration file:

```bash
sloth-kubernetes deploy my-cluster --config cluster.lisp
```

**What happens:**

1. Creates cloud infrastructure (VPC, security groups, nodes)
2. Sets up WireGuard VPN mesh between all nodes
3. Installs RKE2 Kubernetes distribution
4. Configures Salt for node management
5. Stores all state and configuration in Pulumi

**Expected output:**

```
Deploying stack: my-cluster
Configuration: cluster.lisp

[1/6] Initializing providers...
  ✓ DigitalOcean provider ready

[2/6] Creating network infrastructure...
  ✓ VPC created (10.10.0.0/16)
  ✓ Firewall rules configured

[3/6] Provisioning nodes...
  ✓ control-plane-1 (s-2vcpu-4gb) - 164.90.xxx.xxx
  ✓ workers-1 (s-2vcpu-4gb) - 164.90.xxx.xxx
  ✓ workers-2 (s-2vcpu-4gb) - 164.90.xxx.xxx

[4/6] Configuring WireGuard VPN...
  ✓ Mesh network established

[5/6] Installing Kubernetes (RKE2)...
  ✓ Control plane initialized
  ✓ Workers joined cluster

[6/6] Finalizing...
  ✓ Salt master configured
  ✓ Kubeconfig generated

Cluster ready! Time elapsed: 4m 32s

Stack: my-cluster
Nodes: 3
Distribution: RKE2 v1.29.0
```

> **Note:** First deployment takes 4-6 minutes. Subsequent operations are faster.

---

### Step 5: Access Your Cluster

All kubectl commands in sloth-kubernetes are **stack-aware** - you specify the stack name first:

```bash
# Check nodes are ready
sloth-kubernetes kubectl my-cluster get nodes
```

**Output:**

```
NAME              STATUS   ROLES                       AGE   VERSION
control-plane-1   Ready    control-plane,etcd,master   5m    v1.29.0+rke2r1
workers-1         Ready    worker                      4m    v1.29.0+rke2r1
workers-2         Ready    worker                      4m    v1.29.0+rke2r1
```

```bash
# Check system pods
sloth-kubernetes kubectl my-cluster get pods -n kube-system
```

```bash
# Export kubeconfig for external tools
sloth-kubernetes kubeconfig my-cluster > ~/.kube/config

# Now you can use standard kubectl
kubectl get nodes
```

---

### Step 6: Deploy a Test Application

Let's deploy nginx to verify everything works:

```bash
# Create deployment
sloth-kubernetes kubectl my-cluster create deployment nginx --image=nginx:latest

# Expose as LoadBalancer service
sloth-kubernetes kubectl my-cluster expose deployment nginx --port=80 --type=LoadBalancer

# Watch until external IP is assigned
sloth-kubernetes kubectl my-cluster get svc nginx -w
```

**Output:**

```
NAME    TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)        AGE
nginx   LoadBalancer   10.43.12.45    164.90.xxx.xxx   80:31234/TCP   30s
```

```bash
# Test the application
curl http://164.90.xxx.xxx
# Should return: Welcome to nginx!
```

---

### Step 7: Manage Nodes with Salt

sloth-kubernetes includes Salt for powerful node management:

```bash
# Ping all nodes
sloth-kubernetes salt my-cluster ping

# Output:
# control-plane-1: True
# workers-1: True
# workers-2: True

# Check disk usage on all nodes
sloth-kubernetes salt my-cluster system disk

# Check memory usage
sloth-kubernetes salt my-cluster system memory

# Run any command on all nodes
sloth-kubernetes salt my-cluster cmd "uptime"

# Run command on specific node
sloth-kubernetes salt my-cluster cmd "df -h" --target="workers-1"
```

---

### Step 8: Explore Your Stack

View information about your deployed cluster:

```bash
# List all stacks
sloth-kubernetes stacks list

# Get detailed stack info
sloth-kubernetes stacks info my-cluster

# View stack outputs (IPs, kubeconfig, etc.)
sloth-kubernetes stacks output my-cluster

# Export configuration (useful for backup/recovery)
sloth-kubernetes export-config my-cluster --format lisp
```

---

## Cleanup

When you're done, destroy the cluster to stop billing:

```bash
sloth-kubernetes destroy my-cluster
```

**Output:**

```
Destroying stack: my-cluster

This will permanently delete:
  - 3 compute instances
  - 1 VPC
  - All associated resources

Type 'yes' to confirm: yes

[1/3] Removing Kubernetes resources...
  ✓ Workloads removed

[2/3] Destroying infrastructure...
  ✓ workers-2 deleted
  ✓ workers-1 deleted
  ✓ control-plane-1 deleted
  ✓ VPC deleted

[3/3] Cleaning up state...
  ✓ Stack removed

Destruction complete! Time elapsed: 2m 15s
```

---

## Common Issues & Solutions

### "Stack is locked"

Another operation is in progress or a previous operation failed:

```bash
sloth-kubernetes stacks cancel my-cluster
```

### "Credentials not found"

Ensure environment variables are set in your current shell:

```bash
echo $DIGITALOCEAN_TOKEN  # Should show your token
```

### "Connection timeout during deployment"

Cloud provider API may be slow. Retry the deployment:

```bash
sloth-kubernetes deploy my-cluster --config cluster.lisp
```

The operation is idempotent - it will continue from where it left off.

### "Node not joining cluster"

Check node health and WireGuard connectivity:

```bash
sloth-kubernetes health my-cluster
```

---

## Next Steps

Now that your cluster is running:

| Topic | Description |
|-------|-------------|
| [Lisp Configuration](../configuration/lisp-format.md) | Full configuration syntax and options |
| [Built-in Functions](../configuration/builtin-functions.md) | Dynamic config with 70+ Lisp functions |
| [CLI Reference](../user-guide/cli-reference.md) | Complete command documentation |
| [Multi-Cloud Setup](../configuration/examples.md) | Deploy across multiple providers |
| [ArgoCD Integration](../user-guide/argocd.md) | GitOps workflow setup |
| [Backup & Restore](../user-guide/backup.md) | Velero backup configuration |

---

## Example Configurations

### Single-Node Development Cluster

Perfect for local development and testing:

```lisp
(cluster
  (metadata
    (name "dev")
    (environment "development"))

  (providers
    (digitalocean
      (enabled true)
      (region "nyc3")))

  (network
    (mode "wireguard")
    (wireguard (enabled true) (create true) (mesh-networking true)))

  (node-pools
    (all-in-one
      (name "dev-node")
      (provider "digitalocean")
      (region "nyc3")
      (count 1)
      (roles master etcd worker)  ; All roles on one node
      (size "s-4vcpu-8gb")))

  (kubernetes
    (distribution "k3s")))  ; Lightweight distribution
```

### Production-Ready HA Cluster

High availability setup with 3 control plane nodes:

```lisp
(cluster
  (metadata
    (name "production")
    (environment "production"))

  (providers
    (digitalocean
      (enabled true)
      (region "nyc3")))

  (network
    (mode "wireguard")
    (wireguard (enabled true) (create true) (mesh-networking true)))

  (node-pools
    (control-plane
      (name "control-plane")
      (provider "digitalocean")
      (region "nyc3")
      (count 3)  ; HA requires 3 control plane nodes
      (roles master etcd)
      (size "s-4vcpu-8gb"))
    (workers
      (name "workers")
      (provider "digitalocean")
      (region "nyc3")
      (count 5)
      (roles worker)
      (size "s-4vcpu-8gb")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (high-availability true)))
```

### Dynamic Configuration with Environment Variables

Use Lisp functions for environment-aware configs:

```lisp
(cluster
  (metadata
    (name (concat "cluster-" (env "CLUSTER_ENV" "dev")))
    (environment (env "CLUSTER_ENV" "development")))

  (providers
    (digitalocean
      (enabled true)
      (region (env "DO_REGION" "nyc3"))))

  (node-pools
    (control-plane
      (name "control-plane")
      (provider "digitalocean")
      (region (env "DO_REGION" "nyc3"))
      (count (if (eq (env "CLUSTER_ENV") "production") 3 1))
      (roles master etcd)
      (size (env "MASTER_SIZE" "s-2vcpu-4gb")))
    (workers
      (name "workers")
      (provider "digitalocean")
      (region (env "DO_REGION" "nyc3"))
      (count (if (eq (env "CLUSTER_ENV") "production") 5 2))
      (roles worker)
      (size (env "WORKER_SIZE" "s-2vcpu-4gb"))))

  (kubernetes
    (distribution "rke2")))
```

Deploy with different environments:

```bash
# Development (1 master, 2 workers)
CLUSTER_ENV=dev sloth-kubernetes deploy dev --config cluster.lisp

# Production (3 masters, 5 workers)
CLUSTER_ENV=production sloth-kubernetes deploy prod --config cluster.lisp
```
