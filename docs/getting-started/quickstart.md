# Quick Start

Deploy a Kubernetes cluster in 5 minutes.

## Prerequisites

- [sloth-kubernetes installed](installation.md)
- Cloud provider credentials set

## Step 1: Create Configuration

Create `cluster.lisp`:

```lisp
; cluster.lisp - Minimal cluster configuration
(cluster
  (metadata
    (name "quickstart-cluster")
    (environment "development"))

  (providers
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")
      (vpc
        (create true)
        (cidr "10.10.0.0/16"))))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    (masters
      (name "masters")
      (provider "digitalocean")
      (count 1)
      (roles master etcd)
      (size "s-2vcpu-4gb")
      (region "nyc3"))
    (workers
      (name "workers")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-2vcpu-4gb")
      (region "nyc3")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

## Step 2: Set Credentials

```bash
export DIGITALOCEAN_TOKEN="your-token-here"
```

## Step 3: Deploy

```bash
sloth-kubernetes deploy --config cluster.lisp
```

Expected output:

```
Deploying cluster: quickstart-cluster

Creating resources:
  ✓ VPC (10.10.0.0/16)
  ✓ WireGuard VPN server
  ✓ Master node (masters-1)
  ✓ Worker node (workers-1)
  ✓ Worker node (workers-2)

Installing Kubernetes:
  ✓ RKE2 on masters-1
  ✓ Joining workers-1
  ✓ Joining workers-2

Cluster ready!
```

## Step 4: Access Cluster

```bash
# Get kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Verify nodes (using embedded kubectl)
sloth-kubernetes kubectl get nodes
```

Output:

```
NAME        STATUS   ROLES                  AGE   VERSION
masters-1   Ready    control-plane,master   5m    v1.29.0+rke2r1
workers-1   Ready    worker                 4m    v1.29.0+rke2r1
workers-2   Ready    worker                 4m    v1.29.0+rke2r1
```

## Step 5: Deploy Application

```bash
# Create deployment
sloth-kubernetes kubectl create deployment nginx --image=nginx

# Expose service
sloth-kubernetes kubectl expose deployment nginx --port=80 --type=LoadBalancer

# Check status
sloth-kubernetes kubectl get pods,svc
```

## Step 6: Manage Nodes with Salt

```bash
# Login to Salt master
sloth-kubernetes salt login

# Ping all nodes
sloth-kubernetes salt ping

# Check disk usage
sloth-kubernetes salt system disk

# Run command on all nodes
sloth-kubernetes salt cmd "uptime"
```

## Cleanup

```bash
# Destroy cluster
sloth-kubernetes destroy --config cluster.lisp
```

## Next Steps

- [LISP Configuration](../configuration/lisp-format.md) - Full config syntax
- [CLI Reference](../user-guide/cli-reference.md) - All commands
- [Examples](../configuration/examples.md) - More configurations
- [FAQ](../faq.md) - Frequently asked questions
