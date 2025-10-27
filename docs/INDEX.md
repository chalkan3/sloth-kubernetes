---
layout: default
title: Home
nav_order: 1
description: "sloth-kubernetes is a unified tool for deploying and managing multi-cloud Kubernetes clusters with RKE2, WireGuard VPN, and integrated tooling."
permalink: /
---

# sloth-kubernetes
{: .fs-9 }

Multi-Cloud Kubernetes Orchestration in a Single Binary
{: .fs-6 .fw-300 }

[Get Started](getting-started/installation){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/chalkan3/sloth-kubernetes){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## What is sloth-kubernetes?

**sloth-kubernetes** is a comprehensive tool that unifies infrastructure provisioning, configuration management, and Kubernetes operations into a single binary. It eliminates the need for multiple external dependencies and provides a seamless experience for deploying production-grade Kubernetes clusters across multiple cloud providers.

### Five Tools in One

- **Pulumi Automation API** - Infrastructure as Code without external CLI
- **Salt API Client** - 100+ remote operations for node management
- **kubectl** - Complete Kubernetes CLI embedded
- **Helm** - Chart management and deployments
- **Kustomize** - Configuration customization

## Key Features

### 🌐 True Multi-Cloud Support
Deploy clusters across DigitalOcean, Linode, and Azure with unified configuration. AWS and GCP support coming soon.

### 🔐 Security-First Design
- Bastion host with MFA and SSH audit logging
- Encrypted WireGuard VPN mesh between cloud providers
- Private nodes without public IPs
- RBAC, Network Policies, TLS, and CIS compliance

### ⚡ 50+ CLI Commands
Comprehensive command-line interface covering deployment, node management, VPN operations, Salt commands, GitOps, and more.

### 🏗️ Automated Orchestration
8-phase automated deployment: SSH keys → Bastion → VPCs → WireGuard → Nodes → RKE2 → VPN config → DNS

### 🚀 Enterprise-Grade Kubernetes
- RKE2 with security hardening
- High availability with odd-number masters
- Automatic etcd backups
- Zero-downtime rolling updates

## Quick Start

\`\`\`bash
# Install
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes
chmod +x sloth-kubernetes
sudo mv sloth-kubernetes /usr/local/bin/

# Deploy a cluster
sloth-kubernetes deploy --config cluster.yaml

# Check status
sloth-kubernetes status

# Access with kubectl
sloth-kubernetes kubectl get nodes
\`\`\`

## Architecture Overview

\`\`\`
┌─────────────────────────────────────────────────────────────┐
│                     sloth-kubernetes CLI                     │
├──────────────┬──────────────┬──────────────┬────────────────┤
│   Pulumi     │     Salt     │   kubectl    │  Helm/Kustomize│
│  (Embedded)  │  (Embedded)  │  (Embedded)  │   (Wrappers)   │
└──────────────┴──────────────┴──────────────┴────────────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────┐
│              Cloud Providers (DO/Linode/Azure)              │
├─────────────────────────────────────────────────────────────┤
│  Bastion → VPCs → WireGuard Mesh → RKE2 Cluster → DNS      │
└─────────────────────────────────────────────────────────────┘
\`\`\`

## Use Cases

✅ **Local Development** - Minimal cost clusters for testing
✅ **Multi-Cloud Staging** - Test across different providers
✅ **Distributed Production HA** - Resilient multi-region deployments
✅ **Disaster Recovery** - Cross-cloud backup and failover
✅ **Cost Optimization** - Mix spot/preemptible instances across clouds

## Why sloth-kubernetes?

| Feature | sloth-kubernetes | Terraform + Ansible | Raw Pulumi |
|---------|------------------|---------------------|------------|
| **Single Binary** | ✅ | ❌ Multiple tools | ❌ Requires CLI |
| **kubectl Embedded** | ✅ | ❌ | ❌ |
| **Salt Integration** | ✅ | ❌ | ❌ |
| **Multi-Cloud VPN** | ✅ Built-in | ⚠️ Manual setup | ⚠️ Manual setup |
| **Config Management** | ✅ Unified | ❌ Separate tools | ❌ Separate tools |

## Community & Support

- [GitHub Issues](https://github.com/chalkan3/sloth-kubernetes/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/chalkan3/sloth-kubernetes/discussions) - Questions and community support
- [Contributing Guide](https://github.com/chalkan3/sloth-kubernetes/blob/main/CONTRIBUTING.md) - How to contribute

## License

sloth-kubernetes is open source software licensed under the [MIT License](https://github.com/chalkan3/sloth-kubernetes/blob/main/LICENSE).
