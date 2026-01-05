---
slug: /
title: Introduction
sidebar_position: 1
---

# sloth-kubernetes

## Multi-Cloud Kubernetes Orchestration in a Single Binary

Deploy secure, production-grade Kubernetes clusters across **DigitalOcean**, **Linode**, **AWS**, **Azure**, and **GCP** with unified configuration, automated orchestration, and zero external dependencies.

[Get Started](/getting-started/installation) | [View on GitHub](https://github.com/chalkan3/sloth-kubernetes)

---

## What is sloth-kubernetes?

**sloth-kubernetes** eliminates the complexity of multi-cloud Kubernetes deployment by embedding **infrastructure provisioning**, **configuration management**, and **Kubernetes tooling** into a single binary. No Pulumi CLI, no Terraform, no Ansible - just one executable.

### Five Essential Tools, Zero Dependencies

| Tool | Description |
|------|-------------|
| **Pulumi Automation API** | Infrastructure as Code **embedded** in the binary. No external Pulumi CLI required. |
| **SaltStack Integration** | **100+ remote operations** for node management, including cmd.run, pkg.install, service.restart. |
| **kubectl Embedded** | Complete Kubernetes CLI **built-in**. Manage workloads through `sloth-kubernetes kubectl`. |
| **Helm Support** | Chart management and deployments. Install, upgrade, rollback Helm releases. |
| **Kustomize** | Configuration customization for Kubernetes manifests. |

---

## Complete Feature Set

### Multi-Cloud Infrastructure

Deploy across **5 cloud providers** with unified LISP configuration:

- **DigitalOcean** - Droplets, VPCs, Floating IPs, Cloud Firewalls
- **Linode** - Instances, VLANs, NodeBalancers
- **AWS** - EC2, VPC, Route53
- **Azure** - VMs, VNets, Load Balancers
- **GCP** - Compute Engine, VPC, Cloud DNS

### Security-First Architecture

**Bastion Host**
- SSH jump host for private cluster access
- MFA support via Google Authenticator
- Complete SSH session audit logging

**WireGuard VPN Mesh**
- Automatic WireGuard tunnel creation between all nodes
- Full mesh topology for HA and performance
- Private IP routing across cloud providers

**Hardened Kubernetes**
- CIS Kubernetes Benchmark alignment
- RBAC enabled by default
- Network Policies for pod isolation

### 50+ CLI Commands

```bash
# Cluster Lifecycle
sloth-kubernetes deploy --config cluster.lisp
sloth-kubernetes destroy
sloth-kubernetes preview

# Node Management
sloth-kubernetes nodes list
sloth-kubernetes nodes ssh <name>
sloth-kubernetes nodes add --pool workers --count 2

# Stack Operations
sloth-kubernetes stacks list
sloth-kubernetes stacks info
sloth-kubernetes stacks select <name>

# SaltStack (100+ operations)
sloth-kubernetes salt ping
sloth-kubernetes salt cmd.run "uptime"
sloth-kubernetes salt pkg.install nginx

# Kubernetes Tools
sloth-kubernetes kubectl get nodes
sloth-kubernetes helm install nginx bitnami/nginx
```

---

## Quick Start

### Installation

```bash
curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash
```

### Deploy Your First Cluster

```lisp
(cluster
  (metadata
    (name "production")
    (environment "production"))

  (providers
    (digitalocean
      (enabled true)
      (token "${DIGITALOCEAN_TOKEN}")
      (region "nyc3")))

  (node-pools
    (masters
      (name "masters")
      (provider "digitalocean")
      (count 3)
      (roles master etcd)
      (size "s-2vcpu-4gb"))
    (workers
      (name "workers")
      (provider "digitalocean")
      (count 5)
      (roles worker)
      (size "s-4vcpu-8gb")))

  (kubernetes
    (distribution "rke2")
    (version "v1.29.0+rke2r1")))
```

```bash
export DIGITALOCEAN_TOKEN="your-digitalocean-token"
sloth-kubernetes deploy --config cluster.lisp
```

---

## Comparison Matrix

| Feature | sloth-kubernetes | Terraform + Ansible | Raw Pulumi | Rancher |
|---------|------------------|---------------------|------------|---------|
| **Single Binary** | ✅ All-in-one | ❌ 3+ tools | ❌ Requires CLI | ❌ Server required |
| **kubectl Embedded** | ✅ Built-in | ❌ External | ❌ External | ✅ Web UI |
| **Multi-Cloud VPN** | ✅ Automated | ⚠️ Manual | ⚠️ Manual | ❌ Not included |
| **SaltStack** | ✅ 100+ ops | ❌ | ❌ | ❌ |
| **GitOps (ArgoCD)** | ✅ Integrated | ⚠️ Separate | ⚠️ Separate | ⚠️ Fleet |

---

## Next Steps

- [Installation Guide](/getting-started/installation)
- [Quick Start Tutorial](/getting-started/quickstart)
- [CLI Reference](/user-guide/cli-reference)
