# Configuration Examples

Real-world cluster configurations for every scenario. Copy, paste, and customize!

---

## Simple Single-Cloud Cluster

Perfect for development or small projects.

```lisp
; cluster.lisp - Simple development cluster
(cluster
  (metadata
    (name "simple-dev")
    (environment "development"))

  (providers
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")))

  (node-pools
    (all-in-one
      (name "all-in-one")
      (provider "digitalocean")
      (count 1)
      (roles master worker)
      (size "s-2vcpu-4gb")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

**What you get:**
- 1 node serving as both master and worker
- No VPN (single node doesn't need it)
- Perfect for testing
- Cost: ~$24/month

---

## Production HA Multi-Cloud

High availability across multiple clouds.

```lisp
; cluster.lisp - Production HA multi-cloud
(cluster
  (metadata
    (name "production-ha")
    (environment "production"))

  (providers
    ; DigitalOcean for masters
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")
      (vpc
        (create true)
        (cidr "10.10.0.0/16")))
    ; Linode for masters and workers
    (linode
      (enabled true)
      (token "${LINODE_TOKEN}")
      (region "us-east")
      (vpc
        (create true)
        (cidr "10.11.0.0/16"))))

  ; Secure VPN mesh
  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)
      (subnet "10.8.0.0/24")
      (port 51820)))

  (node-pools
    ; Masters across clouds for HA
    (do-masters
      (name "do-masters")
      (provider "digitalocean")
      (count 1)
      (roles master etcd)
      (size "s-2vcpu-4gb"))
    (linode-masters
      (name "linode-masters")
      (provider "linode")
      (count 2)  ; 3 total masters (quorum)
      (roles master etcd)
      (size "g6-standard-2"))
    ; Workers for application workloads
    (do-workers
      (name "do-workers")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-4vcpu-8gb"))
    (linode-workers
      (name "linode-workers")
      (provider "linode")
      (count 2)
      (roles worker)
      (size "g6-standard-4")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (high-availability true)))
```

**What you get:**
- 3 master nodes (1 DO + 2 Linode) for HA
- 4 worker nodes across both clouds
- WireGuard VPN mesh
- Automatic failover
- Cost: ~$180/month

---

## Cost-Optimized Cluster

Maximum value for minimum spend.

```lisp
; cluster.lisp - Budget-friendly cluster
(cluster
  (metadata
    (name "budget-friendly")
    (environment "staging"))

  (providers
    ; Linode (generally cheaper)
    (linode
      (enabled true)
      (token "${LINODE_TOKEN}")
      (region "us-east")
      (vpc
        (create true)
        (cidr "10.20.0.0/16"))))

  (node-pools
    ; Single master (not HA, but cheap!)
    (master
      (name "master")
      (provider "linode")
      (count 1)
      (roles master etcd)
      (size "g6-nanode-1"))  ; Smallest size: $5/month
    ; 2 small workers
    (workers
      (name "workers")
      (provider "linode")
      (count 2)
      (roles worker)
      (size "g6-nanode-1")))  ; Also $5/month each

  (kubernetes
    (distribution "k3s")  ; Lighter than RKE2
    (version "v1.29.0+k3s1")))
```

**What you get:**
- 1 master + 2 workers
- Single cloud (no VPN overhead)
- K3s for lower resource usage
- Perfect for staging/testing
- Cost: ~$15/month

---

## AWS with Spot Instances

Cost savings with spot instances for workers.

```lisp
; cluster.lisp - AWS spot instance cluster
(cluster
  (metadata
    (name "aws-spot-cluster")
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
    ; On-demand masters for stability
    (masters
      (name "masters")
      (provider "aws")
      (count 3)
      (roles master etcd)
      (size "t3.medium"))
    ; Spot workers for cost savings
    (spot-workers
      (name "spot-workers")
      (provider "aws")
      (count 10)
      (roles worker)
      (size "t3.large")
      (spot-instance true)
      (spot-max-price "0.05")))  ; Max $0.05/hour

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (high-availability true)))
```

**What you get:**
- 3 stable on-demand masters
- 10 spot instance workers (up to 90% savings)
- Automatic spot instance management
- Cost: ~$150/month (vs ~$500 on-demand)

---

## GPU Workloads Cluster

For ML/AI and GPU-intensive workloads.

```lisp
; cluster.lisp - GPU cluster
(cluster
  (metadata
    (name "gpu-cluster")
    (environment "ml-training"))

  (providers
    ; DigitalOcean for control plane
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")
      (vpc
        (create true)
        (cidr "10.30.0.0/16")))
    ; Linode for GPU nodes
    (linode
      (enabled true)
      (token "${LINODE_TOKEN}")
      (region "us-east")
      (vpc
        (create true)
        (cidr "10.31.0.0/16"))))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    ; Control plane on DO
    (masters
      (name "masters")
      (provider "digitalocean")
      (count 3)
      (roles master etcd)
      (size "s-2vcpu-4gb"))
    ; CPU workers for system services
    (cpu-workers
      (name "cpu-workers")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-4vcpu-8gb")
      (labels
        (node-type "cpu")))
    ; GPU workers for ML workloads
    (gpu-workers
      (name "gpu-workers")
      (provider "linode")
      (count 2)
      (roles worker)
      (size "g1-gpu-rtx6000-1")  ; RTX 6000 GPU
      (labels
        (node-type "gpu")
        (accelerator "nvidia"))
      (taints
        (gpu
          (key "nvidia.com/gpu")
          (value "true")
          (effect "NoSchedule")))))  ; Only GPU pods here

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

---

## Edge Computing Cluster

Distributed edge locations.

```lisp
; cluster.lisp - Edge distributed cluster
(cluster
  (metadata
    (name "edge-distributed")
    (environment "edge"))

  (providers
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")
      (vpc
        (create true)
        (cidr "10.40.0.0/16"))))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    ; Masters in primary region
    (central-masters
      (name "central-masters")
      (provider "digitalocean")
      (count 3)
      (roles master etcd)
      (size "s-2vcpu-4gb")
      (region "nyc3"))
    ; Edge workers in NYC
    (nyc-edge
      (name "nyc-edge")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-2vcpu-4gb")
      (region "nyc3")
      (labels
        (edge-location "nyc")))
    ; Edge workers in SF
    (sfo-edge
      (name "sfo-edge")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-2vcpu-4gb")
      (region "sfo3")
      (labels
        (edge-location "sfo")))
    ; Edge workers in Amsterdam
    (ams-edge
      (name "ams-edge")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-2vcpu-4gb")
      (region "ams3")
      (labels
        (edge-location "ams"))))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

---

## Hetzner Cloud Cluster

Cost-effective European cloud deployment with Hetzner.

```lisp
; cluster.lisp - Hetzner Cloud cluster
(cluster
  (metadata
    (name "hetzner-cluster")
    (environment "production"))

  (providers
    (hetzner
      (enabled true)
      (location "nbg1")))  ; Nuremberg, Germany

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
    (masters
      (name "masters")
      (provider "hetzner")
      (location "nbg1")
      (count 3)
      (roles master etcd)
      (size "cx22"))  ; 2 vCPU, 4GB RAM
    (workers
      (name "workers")
      (provider "hetzner")
      (location "nbg1")
      (count 5)
      (roles worker)
      (size "cx32")))  ; 4 vCPU, 8GB RAM

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (high-availability true)))
```

**What you get:**
- 3 master nodes with HA
- 5 worker nodes
- WireGuard VPN mesh
- Bastion host with Salt Master
- Cost: ~€80/month (very competitive)

**Hetzner locations:**
- `nbg1` - Nuremberg, Germany
- `fsn1` - Falkenstein, Germany
- `hel1` - Helsinki, Finland
- `ash` - Ashburn, USA

---

## AWS + Hetzner Multi-Cloud

Combine AWS reliability with Hetzner cost-effectiveness.

```lisp
; cluster.lisp - AWS + Hetzner multi-cloud
(cluster
  (metadata
    (name "aws-hetzner-hybrid")
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
    ; AWS masters for reliability
    (aws-masters
      (name "aws-masters")
      (provider "aws")
      (region "us-east-1")
      (count 1)
      (roles master etcd)
      (size "t3.medium"))
    ; AWS workers for US workloads
    (aws-workers
      (name "aws-workers")
      (provider "aws")
      (region "us-east-1")
      (count 2)
      (roles worker)
      (size "t3.medium"))
    ; Hetzner workers for EU workloads (cost-effective)
    (hetzner-workers
      (name "hetzner-workers")
      (provider "hetzner")
      (location "nbg1")
      (count 3)
      (roles worker)
      (size "cx32")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (network-plugin "calico")))
```

**What you get:**
- AWS master for reliability
- Workers split between AWS (US) and Hetzner (EU)
- WireGuard mesh connecting both clouds
- Bastion on Hetzner (cheaper)
- Cost: ~$100/month (AWS) + ~€30/month (Hetzner)

---

## Azure Cluster

Single-cloud Azure deployment.

```lisp
; cluster.lisp - Azure cluster
(cluster
  (metadata
    (name "azure-cluster")
    (environment "production"))

  (providers
    (azure
      (enabled true)
      (location "eastus")
      (resource-group "k8s-rg")
      (vnet
        (create true)
        (cidr "10.50.0.0/16"))))

  (network
    (mode "wireguard")
    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)))

  (node-pools
    (masters
      (name "masters")
      (provider "azure")
      (count 3)
      (roles master etcd)
      (size "Standard_D2s_v3"))
    (workers
      (name "workers")
      (provider "azure")
      (count 5)
      (roles worker)
      (size "Standard_D4s_v3")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (high-availability true)))
```

---

## Three-Cloud HA

Maximum resilience across three cloud providers.

```lisp
; cluster.lisp - Three-cloud HA cluster
(cluster
  (metadata
    (name "ultra-ha")
    (environment "production"))

  (providers
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")
      (vpc
        (create true)
        (cidr "10.10.0.0/16")))
    (linode
      (enabled true)
      (token "${LINODE_TOKEN}")
      (region "us-east")
      (vpc
        (create true)
        (cidr "10.11.0.0/16")))
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
    ; One master per cloud
    (do-master
      (name "do-master")
      (provider "digitalocean")
      (count 1)
      (roles master etcd)
      (size "s-4vcpu-8gb"))
    (linode-master
      (name "linode-master")
      (provider "linode")
      (count 1)
      (roles master etcd)
      (size "g6-standard-4"))
    (aws-master
      (name "aws-master")
      (provider "aws")
      (count 1)
      (roles master etcd)
      (size "t3.large"))
    ; Workers distributed
    (do-workers
      (name "do-workers")
      (provider "digitalocean")
      (count 2)
      (roles worker)
      (size "s-4vcpu-8gb"))
    (linode-workers
      (name "linode-workers")
      (provider "linode")
      (count 2)
      (roles worker)
      (size "g6-standard-4"))
    (aws-workers
      (name "aws-workers")
      (provider "aws")
      (count 2)
      (roles worker)
      (size "t3.large")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")
    (high-availability true)))
```

**What you get:**
- Survives complete cloud provider outage
- 3 masters (one per cloud)
- 6 workers (two per cloud)
- Full WireGuard mesh across all clouds

---

## Environment Variables

Reference environment variables in your configurations:

```lisp
(providers
  (digitalocean
    (token "${DIGITALOCEAN_TOKEN}")  ; From environment
    (region "${DO_REGION:-nyc3}")))  ; With default value
```

Set before deploying:

```bash
# AWS
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# Hetzner Cloud
export HETZNER_TOKEN="your-hetzner-token"

# DigitalOcean
export DIGITALOCEAN_TOKEN="dop_v1_..."

# Linode
export LINODE_TOKEN="..."

# Pulumi state encryption
export PULUMI_CONFIG_PASSPHRASE="your-passphrase"

sloth-kubernetes deploy --config cluster.lisp
```

---

## Tips for Writing Configs

1. **Start small** - Begin with a simple config and add features gradually
2. **Test in dev first** - Always test new configurations in development
3. **Version control** - Keep your configs in Git for tracking and rollback
4. **Use environment variables** - Never hardcode credentials

```bash
# Good structure
k8s-clusters/
├── production.lisp
├── staging.lisp
├── development.lisp
└── examples/
    ├── simple.lisp
    ├── ha.lisp
    └── multi-cloud.lisp
```

---

## Next Steps

- [LISP Format Reference](lisp-format.md) - Complete syntax documentation
- [Backend Configuration](backend.md) - S3 and local state storage
- [CLI Reference](../user-guide/cli-reference.md) - All available commands
