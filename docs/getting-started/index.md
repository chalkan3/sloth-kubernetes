---
title: Getting Started
description: Get started with sloth-kubernetes - deploy production-grade Kubernetes clusters in minutes
---

# Getting Started

Welcome to **sloth-kubernetes** - the multi-cloud Kubernetes deployment platform that fits in a single binary.

---

## What is sloth-kubernetes?

sloth-kubernetes is a CLI tool that deploys production-ready Kubernetes clusters across multiple cloud providers with:

- **Zero external dependencies** - Single binary includes Pulumi, Salt, and kubectl
- **Multi-cloud support** - AWS, DigitalOcean, Linode, Azure (GCP coming soon)
- **Lisp-based configuration** - Dynamic, readable configs with 70+ built-in functions
- **Enterprise features** - State management, audit logging, config versioning, manifest tracking
- **Automatic networking** - WireGuard VPN mesh across all providers

---

## Quick Installation

```bash
# One-line install (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/chalkan3/sloth-kubernetes/main/install.sh | bash

# Verify
sloth-kubernetes version
```

**Other platforms:** See [Installation Guide](installation.md) for Windows, manual install, and building from source.

---

## 5-Minute Quick Start

**1. Set credentials:**

```bash
export DIGITALOCEAN_TOKEN="your-token"
export PULUMI_CONFIG_PASSPHRASE="your-passphrase"
```

**2. Create `cluster.lisp`:**

```lisp
(cluster
  (metadata (name "my-cluster") (environment "dev"))
  (providers (digitalocean (enabled true) (region "nyc3")))
  (network (mode "wireguard") (wireguard (enabled true) (create true) (mesh-networking true)))
  (node-pools
    (main
      (name "main")
      (provider "digitalocean")
      (region "nyc3")
      (count 1)
      (roles master etcd worker)
      (size "s-4vcpu-8gb")))
  (kubernetes (distribution "k3s")))
```

**3. Deploy:**

```bash
sloth-kubernetes deploy my-cluster --config cluster.lisp
```

**4. Use:**

```bash
sloth-kubernetes kubectl my-cluster get nodes
```

**Full guide:** [Quick Start](quickstart.md)

---

## Why sloth-kubernetes?

| Challenge | sloth-kubernetes Solution |
|-----------|---------------------------|
| Multiple tools required (Terraform, kubectl, helm, etc.) | Single binary with everything embedded |
| Static configuration files | Dynamic Lisp configs with env vars, conditionals, functions |
| Vendor lock-in | Multi-cloud support with same config format |
| State management complexity | Pulumi state as database with full audit trail |
| Networking across providers | Automatic WireGuard VPN mesh |

---

## How It Works

```
┌────────────────────────────────────────────────────────────────┐
│                    sloth-kubernetes CLI                        │
│                     (Single Binary)                            │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  Lisp    │  │  Pulumi  │  │  Salt    │  │   kubectl    │   │
│  │  Config  │  │  Engine  │  │  Master  │  │   Embedded   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘   │
├────────────────────────────────────────────────────────────────┤
│            Cloud Providers (AWS, DO, Linode, Azure)            │
├────────────────────────────────────────────────────────────────┤
│          WireGuard VPN Mesh | RKE2/K3s | Node Management       │
└────────────────────────────────────────────────────────────────┘
```

---

## What You'll Learn

In this section:

1. **[Installation](installation.md)** - Install on Linux, macOS, or Windows
2. **[Quick Start](quickstart.md)** - Deploy your first cluster in 5 minutes

After getting started:

- **[User Guide](../user-guide/index.md)** - All CLI commands and workflows
- **[Configuration](../configuration/lisp-format.md)** - Complete Lisp syntax reference
- **[Built-in Functions](../configuration/builtin-functions.md)** - 70+ functions for dynamic configs
- **[Examples](../configuration/examples.md)** - Multi-cloud and production configurations
- **[FAQ](../faq.md)** - Common questions answered

---

## Prerequisites

### Required

- **Cloud provider account** with API credentials:
  - [DigitalOcean](https://cloud.digitalocean.com/account/api/tokens) - Easiest to start
  - [Linode](https://cloud.linode.com/profile/tokens)
  - [AWS](https://console.aws.amazon.com/iam/) - IAM credentials
  - [Azure](https://portal.azure.com/) - Service principal

- **Terminal** - Command-line access (bash, zsh, PowerShell)

### Optional (but embedded)

- kubectl - Embedded in sloth-kubernetes
- Pulumi - Embedded in sloth-kubernetes
- Salt - Automatically configured

---

## Support

- **Issues:** [GitHub Issues](https://github.com/chalkan3/sloth-kubernetes/issues)
- **Discussions:** [GitHub Discussions](https://github.com/chalkan3/sloth-kubernetes/discussions)

---

Ready to get started? Head to the **[Quick Start Guide](quickstart.md)** to deploy your first cluster.
