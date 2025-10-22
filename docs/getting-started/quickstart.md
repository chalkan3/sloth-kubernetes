# 🦥 Quick Start

Deploy a production-ready Kubernetes cluster in 5 minutes. Slowly, but surely!

---

## Overview

This quick start will guide you through deploying your first multi-cloud Kubernetes cluster with:

- ✅ 3 master nodes for high availability
- ✅ 2 worker nodes across multiple clouds
- ✅ WireGuard VPN mesh (automatic)
- ✅ RKE2 Kubernetes distribution
- ✅ Encrypted secrets at rest

**Time:** 5-8 minutes ☕🦥

---

## Step 1: Prerequisites

Make sure you have:

1. **Sloth Kubernetes binary** installed ([Installation Guide](installation.md))
2. **API tokens** configured as environment variables:

```bash
export DIGITALOCEAN_TOKEN="dop_v1_your_token_here"
export LINODE_TOKEN="your_linode_token_here"
```

---

## Step 2: Create Configuration File

Create a file named `my-first-cluster.yaml`:

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: my-first-cluster  # 🦥 Name your sloth cluster
  labels:
    environment: demo
    managed-by: sloth

spec:
  # Cloud providers configuration
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}  # 🦥 From environment
      region: nyc3
      vpc:
        create: true
        cidr: 10.10.0.0/16

    linode:
      enabled: true
      token: ${LINODE_TOKEN}  # 🦥 From environment
      region: us-east
      vpc:
        create: true
        cidr: 10.11.0.0/16

  # Network configuration
  network:
    wireguard:
      create: true  # 🦥 Auto-create VPN mesh
      meshNetworking: true
      subnet: 10.8.0.0/24

  # Kubernetes configuration
  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    rke2:
      secretsEncryption: true  # 🦥 Secure by default

  # Node pools
  nodePools:
    # DigitalOcean masters
    - name: do-masters
      provider: digitalocean
      count: 1
      roles: [master]
      size: s-2vcpu-4gb

    # Linode masters (for HA)
    - name: linode-masters
      provider: linode
      count: 2  # 🦥 3 total masters for quorum
      roles: [master]
      size: g6-standard-2

    # Workers across both clouds
    - name: do-workers
      provider: digitalocean
      count: 1
      roles: [worker]
      size: s-2vcpu-4gb

    - name: linode-workers
      provider: linode
      count: 1
      roles: [worker]
      size: g6-standard-2
```

!!! tip "Sloth Tip 🦥"
    This configuration creates a **true multi-cloud HA cluster** with masters and workers across both DigitalOcean and Linode!

---

## Step 3: Deploy! 🦥

Now let's deploy your cluster:

```bash
# Deploy the cluster
sloth-kubernetes deploy --config my-first-cluster.yaml

# The sloth will:
# 🦥 Create VPCs on both clouds
# 🦥 Deploy WireGuard VPN server
# 🦥 Provision 5 nodes (3 masters, 2 workers)
# 🦥 Install RKE2 on all nodes
# 🦥 Configure encrypted mesh networking
# 🦥 Generate kubeconfig
```

You'll see output like:

```
🦥 Sloth Kubernetes Deployment
Slowly, but surely deploying your cluster...

✓ Creating DigitalOcean VPC (10.10.0.0/16)
✓ Creating Linode VPC (10.11.0.0/16)
✓ Deploying WireGuard VPN server
✓ Provisioning master nodes (1/3)
✓ Provisioning master nodes (2/3)
✓ Provisioning master nodes (3/3)
✓ Installing RKE2 on masters
✓ Provisioning worker nodes (1/2)
✓ Provisioning worker nodes (2/2)
✓ Installing RKE2 on workers
✓ Configuring WireGuard mesh
✓ Generating kubeconfig

🦥 Cluster deployed successfully!
   Time elapsed: 7m 32s

   Kubeconfig: ./my-first-cluster-kubeconfig.yaml
```

!!! success "Deployment Complete! 🦥"
    Your multi-cloud Kubernetes cluster is now running! Time to relax like a sloth! 😴

---

## Step 4: Access Your Cluster

Get the kubeconfig and start using your cluster:

```bash
# Export kubeconfig
export KUBECONFIG=$(pwd)/my-first-cluster-kubeconfig.yaml

# Or copy to default location
mkdir -p ~/.kube
cp my-first-cluster-kubeconfig.yaml ~/.kube/config

# Verify cluster access
kubectl get nodes

# You should see:
NAME                    STATUS   ROLES                       AGE   VERSION
do-master-1             Ready    control-plane,etcd,master   7m    v1.28.5+rke2r1
linode-master-1         Ready    control-plane,etcd,master   7m    v1.28.5+rke2r1
linode-master-2         Ready    control-plane,etcd,master   6m    v1.28.5+rke2r1
do-worker-1             Ready    worker                      5m    v1.28.5+rke2r1
linode-worker-1         Ready    worker                      5m    v1.28.5+rke2r1
```

```bash
# Check cluster info
kubectl cluster-info

# View pods across all namespaces
kubectl get pods -A

# Deploy a test application 🦥
kubectl create deployment nginx --image=nginx
kubectl expose deployment nginx --port=80 --type=LoadBalancer
kubectl get svc
```

---

## Step 5: What Just Happened? 🦥

Let's understand what the sloth built for you:

### Architecture Diagram

```
        🦥 Your Multi-Cloud Cluster 🦥

┌─────────────────────────┐  ┌─────────────────────────┐
│   DigitalOcean NYC3     │  │     Linode US-East      │
│                         │  │                         │
│  • Master 1        🦥   │  │  • Master 2        🦥   │
│  • Worker 1        🦥   │  │  • Master 3        🦥   │
│                         │  │  • Worker 1        🦥   │
│  VPC: 10.10.0.0/16      │  │  VPC: 10.11.0.0/16      │
└───────────┬─────────────┘  └─────────┬───────────────┘
            │                          │
            └──────► WireGuard ◄───────┘
                   10.8.0.0/24
                      🔐
```

### What Was Created

| Component | Details |
|-----------|---------|
| **VPCs** | 2 VPCs (1 per cloud) with private networking |
| **VPN** | WireGuard mesh connecting all nodes |
| **Masters** | 3 control plane nodes across 2 clouds |
| **Workers** | 2 worker nodes for your workloads |
| **Kubernetes** | RKE2 v1.28.5 with encrypted secrets |
| **Networking** | Private mesh + public access |

---

## Next Steps 🦥

Now that your cluster is running, explore more features:

<div class="grid cards" markdown>

-   📖 **Add More Nodes**

    ---

    Scale your cluster up! 🦥

    [:octicons-arrow-right-24: Manage Nodes](../user-guide/nodes.md)

-   🔐 **Configure VPN**

    ---

    Access your cluster securely 🦥

    [:octicons-arrow-right-24: VPN Management](../user-guide/vpn.md)

-   🌳 **Enable GitOps**

    ---

    Bootstrap ArgoCD for GitOps 🦥

    [:octicons-arrow-right-24: GitOps Guide](../advanced/gitops.md)

-   ⚙️ **Advanced Config**

    ---

    Customize everything 🦥

    [:octicons-arrow-right-24: Configuration](../configuration/file-structure.md)

</div>

---

## Troubleshooting

### Common Issues

??? question "Deployment stuck at 'Provisioning nodes'"
    This is normal! Cloud providers can take 2-3 minutes to provision instances. The sloth is patient! 🦥

??? question "VPN connection failed"
    Check that:
    - Firewalls allow UDP port 51820
    - VPC CIDR ranges don't overlap
    - Nodes have public IPs for initial setup

??? question "kubectl connection refused"
    Verify:
    ```bash
    # Check kubeconfig path
    echo $KUBECONFIG

    # Test with explicit path
    kubectl --kubeconfig=my-first-cluster-kubeconfig.yaml get nodes
    ```

For more help, see [Troubleshooting Guide](../advanced/troubleshooting.md) 🦥

---

## Clean Up (Optional)

When you're done testing, clean up resources:

```bash
# Destroy the cluster
sloth-kubernetes destroy --config my-first-cluster.yaml

# The sloth will:
# 🦥 Remove all nodes
# 🦥 Delete VPCs
# 🦥 Clean up VPN server
# 🦥 Remove local state
```

!!! warning "Destruction is Permanent 🦥"
    This will permanently delete all cluster resources. Make sure you've backed up any data!

---

!!! quote "Ancient Sloth Wisdom 🦥"
    *"A cluster deployed slowly is a cluster deployed correctly!"*

**Congratulations!** You've deployed your first Sloth Kubernetes cluster! 🦥🎉
