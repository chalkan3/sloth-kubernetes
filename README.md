<div align="center">

# 🦥 Sloth Kubernetes

### Multi-Cloud Kubernetes Deployment Made Simple

**Deploy production-ready Kubernetes clusters across DigitalOcean and Linode**
*with automated VPC creation, WireGuard VPN mesh, and zero external dependencies*

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Test Coverage](https://img.shields.io/badge/coverage-46.1%25-yellow?style=for-the-badge)](./TESTS_COVERAGE_REPORT.md)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE)
[![Pulumi](https://img.shields.io/badge/Pulumi-Embedded-8A3391?style=for-the-badge&logo=pulumi)](https://pulumi.com)

[Quick Start](#-quick-start) •
[Features](#-key-features) •
[Documentation](#-documentation) •
[CLI Reference](#-cli-reference) •
[Examples](#-configuration-examples)

</div>

---

## 📖 Table of Contents

- [Overview](#-overview)
- [Key Features](#-key-features)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Architecture](#-architecture)
- [CLI Reference](#-cli-reference)
- [Configuration Guide](#-configuration-guide)
- [Configuration Examples](#-configuration-examples)
- [Deployment Workflow](#-deployment-workflow)
- [Network Architecture](#-network-architecture)
- [Use Cases](#-use-cases)
- [Advanced Topics](#-advanced-topics)
- [Testing](#-testing)
- [Troubleshooting](#-troubleshooting)
- [FAQ](#-faq)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🌟 Overview

**Sloth Kubernetes** is a **single-binary CLI tool** that deploys production-grade Kubernetes clusters across multiple cloud providers with **zero external dependencies**. No Pulumi CLI, no Terraform, no complex setup—just one binary and you're ready to deploy.

### What Makes It Different?

```
┌─────────────────────────────────────────────────────────────────┐
│                    Traditional Approach                          │
├─────────────────────────────────────────────────────────────────┤
│  1. Install Pulumi CLI                                          │
│  2. Install kubectl                                             │
│  3. Install cloud provider CLIs                                 │
│  4. Write infrastructure code                                   │
│  5. Configure state backend                                     │
│  6. Run multiple commands                                       │
│  7. Manage dependencies between tools                           │
└─────────────────────────────────────────────────────────────────┘
                              ❌ Complex

┌─────────────────────────────────────────────────────────────────┐
│                    Sloth Kubernetes                              │
├─────────────────────────────────────────────────────────────────┤
│  1. Download single binary                                      │
│  2. Create YAML config                                          │
│  3. Run: sloth-kubernetes deploy                                │
└─────────────────────────────────────────────────────────────────┘
                              ✅ Simple
```

### Technology Stack

```
┌──────────────────────────────────────────────────────────────────┐
│                     Sloth Kubernetes Binary                       │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌────────────────┐  ┌──────────────┐  ┌──────────────────┐    │
│  │   CLI Layer    │  │  Validation  │  │  Orchestration   │    │
│  │   (Cobra)      │→ │   Engine     │→ │     Engine       │    │
│  └────────────────┘  └──────────────┘  └──────────────────┘    │
│                                                 ↓                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │         Pulumi Automation API (Embedded)                 │   │
│  │  • No CLI required                                       │   │
│  │  • Programmatic infrastructure                           │   │
│  │  • State management built-in                             │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                 ↓                 │
│  ┌──────────────────┐  ┌────────────────────┐                   │
│  │  DigitalOcean    │  │     Linode         │                   │
│  │  Provider SDK    │  │  Provider SDK      │                   │
│  └──────────────────┘  └────────────────────┘                   │
└──────────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────────┐
│                   Cloud Infrastructure                            │
│                                                                   │
│  VPCs • Droplets • Linodes • WireGuard VPN • RKE2 Kubernetes     │
└──────────────────────────────────────────────────────────────────┘
```

---

## ✨ Key Features

### 🎯 Zero External Dependencies

<table>
<tr>
<td width="50%">

**✅ What You Need**
- Go 1.23+ (for building only)
- Cloud provider API tokens
- SSH access

</td>
<td width="50%">

**❌ What You DON'T Need**
- Pulumi CLI
- Terraform
- kubectl (for deployment)
- Docker
- Ansible
- Any other IaC tools

</td>
</tr>
</table>

### 🌐 True Multi-Cloud Support

Deploy a single Kubernetes cluster with nodes across multiple providers:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Your Kubernetes Cluster                       │
│                                                                  │
│  ┌──────────────────────┐         ┌──────────────────────┐     │
│  │   DigitalOcean       │         │      Linode          │     │
│  │   Region: NYC3       │ ◄─────► │   Region: US-East    │     │
│  ├──────────────────────┤   VPN   ├──────────────────────┤     │
│  │ • 1 Master Node      │         │ • 2 Master Nodes     │     │
│  │ • 2 Worker Nodes     │         │ • 1 Worker Node      │     │
│  │ • VPC: 10.10.0.0/16  │         │ • VPC: 10.11.0.0/16  │     │
│  └──────────────────────┘         └──────────────────────┘     │
│           ↑                                    ↑                 │
│           └─────── WireGuard Mesh ────────────┘                │
│                  (10.8.0.0/24)                                   │
└─────────────────────────────────────────────────────────────────┘
```

**Why Multi-Cloud?**
- 🛡️ **High Availability** - Survive provider outages
- 💰 **Cost Optimization** - Use best pricing per region
- 🌍 **Geographic Distribution** - Reduce latency globally
- 🔄 **Avoid Vendor Lock-in** - Freedom to choose

### 🔐 Automated Networking

#### Sequential 3-Phase Deployment

```
Phase 1: VPC Creation
┌─────────────────────────────────────────────────────────┐
│  ✓ Create DigitalOcean VPC (10.10.0.0/16)             │
│  ✓ Create Linode VPC (10.11.0.0/16)                   │
│  ✓ Configure subnets and gateways                      │
└─────────────────────────────────────────────────────────┘
                         ↓
Phase 2: WireGuard VPN
┌─────────────────────────────────────────────────────────┐
│  ✓ Deploy VPN server (auto-created)                    │
│  ✓ Generate encryption keys                            │
│  ✓ Configure mesh networking                           │
│  ✓ Enable cross-provider routing                       │
└─────────────────────────────────────────────────────────┘
                         ↓
Phase 3: Kubernetes Cluster
┌─────────────────────────────────────────────────────────┐
│  ✓ Provision nodes (masters + workers)                 │
│  ✓ Install RKE2 Kubernetes                             │
│  ✓ Configure WireGuard on each node                    │
│  ✓ Join nodes to cluster                               │
│  ✓ Validate cluster health                             │
└─────────────────────────────────────────────────────────┘
```

**All in One Command:** `sloth-kubernetes deploy --config cluster.yaml`

### 🚀 Production-Ready Kubernetes

#### RKE2 Distribution Features

```yaml
├─ High Availability         # Odd-number master nodes (3, 5, 7)
├─ Automated Etcd Backups   # Scheduled snapshots with retention
├─ Secrets Encryption       # At-rest encryption for etcd
├─ Network Policies         # Calico/Cilium CNI support
├─ Security Hardening       # CIS benchmark compliance
├─ Rolling Updates          # Zero-downtime upgrades
└─ Multi-CNI Support        # Calico, Cilium, Canal, Flannel
```

### 🎯 GitOps-Native

```
┌─────────────────────────────────────────────────────────────┐
│  Your Git Repository                                        │
│  https://github.com/yourorg/k8s-gitops                      │
│                                                              │
│  ├── argocd/                                                │
│  │   └── install.yaml      ← ArgoCD self-manages itself     │
│  ├── apps/                                                  │
│  │   ├── cert-manager/                                      │
│  │   ├── ingress-nginx/                                     │
│  │   └── monitoring/                                        │
│  └── clusters/                                              │
│      └── production/                                        │
└─────────────────────────────────────────────────────────────┘
                         ↓
         sloth-kubernetes addons bootstrap \
           --repo https://github.com/yourorg/k8s-gitops
                         ↓
┌─────────────────────────────────────────────────────────────┐
│  Kubernetes Cluster                                         │
│  • ArgoCD auto-installed                                    │
│  • Watches Git repository                                   │
│  • Syncs all applications                                   │
│  • Self-healing and auto-sync                               │
└─────────────────────────────────────────────────────────────┘
```

---

## ⚡ Quick Start

### 3 Minutes to Your First Cluster

```bash
# 1. Clone and build (30 seconds)
git clone https://github.com/yourusername/sloth-kubernetes.git
cd sloth-kubernetes
go build -o sloth-kubernetes

# 2. Create config file (1 minute)
cat > cluster.yaml <<EOF
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: my-first-cluster

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.10.0.0/16

  network:
    wireguard:
      create: true              # Auto-create VPN server
      provider: digitalocean
      meshNetworking: true

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    - name: masters
      provider: digitalocean
      count: 3
      roles: [master]
      size: s-2vcpu-4gb

    - name: workers
      provider: digitalocean
      count: 3
      roles: [worker]
      size: s-2vcpu-4gb
EOF

# 3. Deploy! (5-10 minutes)
export DIGITALOCEAN_TOKEN="your-token-here"
./sloth-kubernetes deploy --config cluster.yaml

# 4. Access your cluster
./sloth-kubernetes kubeconfig > ~/.kube/config
kubectl get nodes
```

**Expected Output:**
```
NAME              STATUS   ROLES                  AGE   VERSION
do-master-1       Ready    control-plane,master   5m    v1.28.5+rke2r1
do-master-2       Ready    control-plane,master   5m    v1.28.5+rke2r1
do-master-3       Ready    control-plane,master   5m    v1.28.5+rke2r1
do-worker-1       Ready    worker                 4m    v1.28.5+rke2r1
do-worker-2       Ready    worker                 4m    v1.28.5+rke2r1
do-worker-3       Ready    worker                 4m    v1.28.5+rke2r1
```

---

## 📥 Installation

### Method 1: Build from Source (Recommended)

```bash
# Clone repository
git clone https://github.com/yourusername/sloth-kubernetes.git
cd sloth-kubernetes

# Build binary
go build -o sloth-kubernetes

# Install globally
sudo mv sloth-kubernetes /usr/local/bin/

# Verify installation
sloth-kubernetes version
```

### Method 2: Direct Go Install

```bash
go install github.com/yourusername/sloth-kubernetes@latest
```

### Method 3: Download Pre-built Binary (Coming Soon)

```bash
# Linux
curl -L https://github.com/yourusername/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes

# macOS Intel
curl -L https://github.com/yourusername/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-amd64 -o sloth-kubernetes

# macOS Apple Silicon
curl -L https://github.com/yourusername/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-arm64 -o sloth-kubernetes

chmod +x sloth-kubernetes
sudo mv sloth-kubernetes /usr/local/bin/
```

### Prerequisites

<table>
<tr>
<th>Component</th>
<th>Version</th>
<th>Required For</th>
</tr>
<tr>
<td>Go</td>
<td>1.23+</td>
<td>Building from source</td>
</tr>
<tr>
<td>DigitalOcean API Token</td>
<td>-</td>
<td>DigitalOcean resources</td>
</tr>
<tr>
<td>Linode API Token</td>
<td>-</td>
<td>Linode resources (optional)</td>
</tr>
<tr>
<td>SSH Key Pair</td>
<td>-</td>
<td>Node access</td>
</tr>
</table>

---

## 🏗️ Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Sloth Kubernetes CLI                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Command Layer (cmd/)                                  │    │
│  │  ┌──────┐ ┌───────┐ ┌───────┐ ┌──────┐ ┌─────────┐  │    │
│  │  │deploy│ │destroy│ │ nodes │ │ vpn  │ │ addons  │  │    │
│  │  └──────┘ └───────┘ └───────┘ └──────┘ └─────────┘  │    │
│  └────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Configuration Layer (pkg/config/)                     │    │
│  │  • YAML parsing                                        │    │
│  │  • Validation                                          │    │
│  │  • Schema enforcement                                  │    │
│  └────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Orchestration Layer (internal/orchestrator/)          │    │
│  │                                                         │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  │    │
│  │  │ SSH Keys    │→ │  Bastion     │→ │     VPC     │  │    │
│  │  └─────────────┘  └──────────────┘  └─────────────┘  │    │
│  │         ↓                                               │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  │    │
│  │  │   Nodes     │→ │  WireGuard   │→ │    RKE2     │  │    │
│  │  └─────────────┘  └──────────────┘  └─────────────┘  │    │
│  │         ↓                                               │    │
│  │  ┌─────────────┐                                       │    │
│  │  │     DNS     │                                       │    │
│  │  └─────────────┘                                       │    │
│  └────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Pulumi Automation API (Embedded)                      │    │
│  │  • Infrastructure as Code                              │    │
│  │  • State management                                    │    │
│  │  • Resource tracking                                   │    │
│  │  • Diff/preview capabilities                           │    │
│  └────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  Provider SDKs                                         │    │
│  │  ┌──────────────────┐    ┌───────────────────────┐   │    │
│  │  │  DigitalOcean    │    │      Linode           │   │    │
│  │  │  • Droplets      │    │  • Instances          │   │    │
│  │  │  • VPCs          │    │  • VPCs               │   │    │
│  │  │  • Firewalls     │    │  • Firewalls          │   │    │
│  │  │  • DNS           │    │  • NodeBalancers      │   │    │
│  │  └──────────────────┘    └───────────────────────┘   │    │
│  └────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Cloud Infrastructure                          │
│                                                                  │
│  ┌──────────────────────┐         ┌──────────────────────┐     │
│  │   DigitalOcean       │         │      Linode          │     │
│  │                      │◄───VPN──►│                      │     │
│  │  • VPCs              │         │  • VPCs              │     │
│  │  • Droplets          │         │  • Instances         │     │
│  │  • Load Balancers    │         │  • NodeBalancers     │     │
│  └──────────────────────┘         └──────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

### Deployment Flow

```
User Input (cluster.yaml)
          ↓
┌─────────────────────────────────────────┐
│ 1. Load & Validate Configuration        │
│    ✓ Parse YAML                         │
│    ✓ Validate providers                 │
│    ✓ Validate node distribution         │
│    ✓ Check network configuration        │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ 2. Initialize Pulumi Stack              │
│    ✓ Create/select stack                │
│    ✓ Configure backend                  │
│    ✓ Set config values                  │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ 3. Phase 1: VPC Creation                │
│    ✓ Create DigitalOcean VPC            │
│    ✓ Create Linode VPC                  │
│    ✓ Configure subnets                  │
│    ✓ Setup routing tables               │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ 4. Phase 2: WireGuard VPN               │
│    ✓ Deploy VPN server                  │
│    ✓ Generate server keys               │
│    ✓ Configure firewall rules           │
│    ✓ Setup routing                      │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ 5. Phase 3: Kubernetes Cluster          │
│    ✓ Generate SSH keys                  │
│    ✓ Create bastion host (optional)     │
│    ✓ Deploy master nodes                │
│    ✓ Deploy worker nodes                │
│    ✓ Install RKE2                       │
│    ✓ Configure WireGuard on nodes       │
│    ✓ Join nodes to cluster              │
│    ✓ Configure DNS                      │
└─────────────────────────────────────────┘
          ↓
┌─────────────────────────────────────────┐
│ 6. Export Outputs                       │
│    • Cluster name                       │
│    • Kubeconfig                         │
│    • API endpoint                       │
│    • SSH private key                    │
│    • VPC IDs                            │
│    • VPN configuration                  │
└─────────────────────────────────────────┘
          ↓
    Production Cluster Ready! 🎉
```

---

## 🎮 CLI Reference

### Global Flags

All commands support these global flags:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | `./cluster-config.yaml` | Path to configuration file |
| `--stack` | `-s` | `production` | Pulumi stack name for multi-environment support |
| `--verbose` | `-v` | `false` | Enable verbose output for debugging |
| `--yes` | `-y` | `false` | Auto-approve without confirmation prompts |

### Command Overview

```
sloth-kubernetes
├── deploy          Deploy a Kubernetes cluster
├── destroy         Destroy a Kubernetes cluster
├── status          Show cluster status and health
├── kubeconfig      Get kubeconfig for kubectl access
│
├── nodes           Node management
│   ├── list        List all cluster nodes
│   ├── add         Add nodes to existing pool
│   ├── remove      Remove node from cluster
│   ├── ssh         SSH into a node
│   └── upgrade     Upgrade Kubernetes version
│
├── vpn             VPN management
│   ├── status      Show VPN mesh status
│   ├── peers       List VPN peers
│   ├── config      Get node WireGuard config
│   ├── test        Test VPN connectivity
│   └── join        Add machine to VPN
│
├── addons          Addon management
│   ├── bootstrap   Bootstrap GitOps from repository
│   ├── list        List installed addons
│   └── install     Install specific addon
│
├── config          Configuration utilities
│   ├── generate    Generate example config file
│   └── validate    Validate configuration file
│
├── stacks          Stack management
│   ├── list        List all Pulumi stacks
│   ├── select      Switch active stack
│   └── delete      Delete a stack
│
└── version         Show version information
```

---

### 📦 deploy

Deploy a complete Kubernetes cluster with VPC and VPN infrastructure.

**Usage:**
```bash
sloth-kubernetes deploy [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--config`, `-c` | Path to cluster configuration file |
| `--stack`, `-s` | Pulumi stack name (default: "production") |
| `--dry-run` | Preview changes without applying |
| `--yes`, `-y` | Auto-approve without prompting |
| `--verbose`, `-v` | Show detailed output |

**Examples:**

```bash
# Basic deployment
sloth-kubernetes deploy --config cluster.yaml

# Preview changes before deploying (dry-run)
sloth-kubernetes deploy --config cluster.yaml --dry-run

# Deploy to specific stack
sloth-kubernetes deploy --config production.yaml --stack prod

# Auto-approve deployment (CI/CD)
sloth-kubernetes deploy --config cluster.yaml --yes

# Verbose output for debugging
sloth-kubernetes deploy --config cluster.yaml --verbose
```

**Deployment Phases:**

```
🚀 Deploying cluster: production-cluster

Phase 1/3: VPC Creation
  ✓ Creating DigitalOcean VPC (10.10.0.0/16)... Done
  ✓ Creating Linode VPC (10.11.0.0/16)... Done

Phase 2/3: WireGuard VPN Setup
  ✓ Deploying VPN server... Done
  ✓ Generating encryption keys... Done
  ✓ Configuring mesh networking... Done

Phase 3/3: Kubernetes Cluster
  ✓ Generating SSH keys... Done
  ✓ Creating master nodes (3)... Done
  ✓ Creating worker nodes (3)... Done
  ✓ Installing RKE2... Done
  ✓ Configuring WireGuard on nodes... Done
  ✓ Joining nodes to cluster... Done

✅ Cluster deployed successfully!

📊 Cluster Information:
  Name: production-cluster
  API Endpoint: 167.99.123.45:6443
  Kubernetes Version: v1.28.5+rke2r1

🌐 VPC Information:
  DigitalOcean VPC: vpc-abc123 (10.10.0.0/16)
  Linode VPC: vpc-def456 (10.11.0.0/16)

🔐 VPN Information:
  Server: 167.99.123.45:51820
  Subnet: 10.8.0.0/24

📋 Nodes:
  NAME              PROVIDER        ROLE     PUBLIC IP       VPN IP
  do-master-1       DigitalOcean    master   167.99.1.1      10.8.0.10
  do-master-2       DigitalOcean    master   167.99.1.2      10.8.0.11
  linode-master-1   Linode          master   172.104.1.1     10.8.0.12
  do-worker-1       DigitalOcean    worker   167.99.2.1      10.8.0.20
  do-worker-2       DigitalOcean    worker   167.99.2.2      10.8.0.21
  linode-worker-1   Linode          worker   172.104.2.1     10.8.0.22

⏱  Total time: 8m 32s

Next steps:
  • Get kubeconfig: sloth-kubernetes kubeconfig
  • Check status: sloth-kubernetes status
  • Bootstrap GitOps: sloth-kubernetes addons bootstrap
```

---

### 🗑️ destroy

Destroy the entire cluster including all infrastructure.

**Usage:**
```bash
sloth-kubernetes destroy [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--stack`, `-s` | Pulumi stack name to destroy |
| `--yes`, `-y` | Skip double confirmation |
| `--force` | Force destroy even if resources are protected |

**Examples:**

```bash
# Destroy with confirmation
sloth-kubernetes destroy --stack production

# Force destroy without confirmation (dangerous!)
sloth-kubernetes destroy --stack production --yes

# Destroy specific stack
sloth-kubernetes destroy --stack staging
```

**Output:**
```
⚠️  WARNING: This will destroy the entire cluster!

Cluster: production-cluster
Nodes: 6
VPCs: 2
Resources: 24

This action cannot be undone!

Type 'production-cluster' to confirm: production-cluster

🗑️  Destroying cluster...
  ✓ Removing nodes... Done
  ✓ Destroying VPN server... Done
  ✓ Deleting VPCs... Done
  ✓ Cleaning up DNS records... Done

✅ Cluster destroyed successfully
⏱  Total time: 2m 15s
```

---

### 📊 status

Display cluster health, node status, and resource information.

**Usage:**
```bash
sloth-kubernetes status [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--stack`, `-s` | Stack name |
| `--format` | Output format: table, json, yaml |
| `--watch`, `-w` | Watch mode (refresh every 5s) |

**Examples:**

```bash
# Show status
sloth-kubernetes status

# JSON output
sloth-kubernetes status --format json

# Watch mode
sloth-kubernetes status --watch

# Specific stack
sloth-kubernetes status --stack staging
```

**Output:**
```
┌─────────────────────────────────────────────────────────────────┐
│  Cluster: production-cluster                                    │
│  Status: ✅ Healthy                                             │
│  Uptime: 3d 12h 45m                                             │
└─────────────────────────────────────────────────────────────────┘

📊 Cluster Info
  Kubernetes Version: v1.28.5+rke2r1
  API Endpoint: https://167.99.123.45:6443
  CNI: Calico

🌐 Network
  VPN Status: ✅ Active
  Mesh Peers: 6/6 connected

  DigitalOcean VPC: vpc-abc123 (10.10.0.0/16)
  Linode VPC: vpc-def456 (10.11.0.0/16)
  VPN Subnet: 10.8.0.0/24

📋 Nodes (6)

┌──────────────────┬──────────────┬────────┬────────┬────────────────┬─────────────┬─────────┐
│ NAME             │ PROVIDER     │ ROLE   │ STATUS │ PUBLIC IP      │ VPN IP      │ UPTIME  │
├──────────────────┼──────────────┼────────┼────────┼────────────────┼─────────────┼─────────┤
│ do-master-1      │ DigitalOcean │ master │ ✅     │ 167.99.1.1     │ 10.8.0.10   │ 3d 12h  │
│ do-master-2      │ DigitalOcean │ master │ ✅     │ 167.99.1.2     │ 10.8.0.11   │ 3d 12h  │
│ linode-master-1  │ Linode       │ master │ ✅     │ 172.104.1.1    │ 10.8.0.12   │ 3d 12h  │
│ do-worker-1      │ DigitalOcean │ worker │ ✅     │ 167.99.2.1     │ 10.8.0.20   │ 3d 12h  │
│ do-worker-2      │ DigitalOcean │ worker │ ✅     │ 167.99.2.2     │ 10.8.0.21   │ 3d 12h  │
│ linode-worker-1  │ Linode       │ worker │ ✅     │ 172.104.2.1    │ 10.8.0.22   │ 3d 12h  │
└──────────────────┴──────────────┴────────┴────────┴────────────────┴─────────────┴─────────┘

💰 Estimated Monthly Cost: $120/month
```

---

### 🔑 kubeconfig

Retrieve kubeconfig for kubectl access.

**Usage:**
```bash
sloth-kubernetes kubeconfig [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--output`, `-o` | Output file (default: stdout) |
| `--stack`, `-s` | Stack name |
| `--merge` | Merge with existing kubeconfig |

**Examples:**

```bash
# Print to stdout
sloth-kubernetes kubeconfig

# Save to file
sloth-kubernetes kubeconfig -o ~/.kube/config

# Merge with existing kubeconfig
sloth-kubernetes kubeconfig --merge -o ~/.kube/config

# Specific stack
sloth-kubernetes kubeconfig --stack production -o prod-kubeconfig.yaml
```

**Usage:**
```bash
# Get kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Verify cluster access
kubectl get nodes
kubectl get pods --all-namespaces
```

---

### 🖥️ nodes

Manage cluster nodes (add, remove, SSH, upgrade).

#### nodes list

List all cluster nodes with detailed information.

**Usage:**
```bash
sloth-kubernetes nodes list [stack] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--format` | Output format: table, json, yaml |
| `--filter` | Filter by role, provider, or status |

**Examples:**

```bash
# List all nodes
sloth-kubernetes nodes list

# List for specific stack
sloth-kubernetes nodes list production

# JSON output
sloth-kubernetes nodes list --format json

# Filter by role
sloth-kubernetes nodes list --filter role=master

# Filter by provider
sloth-kubernetes nodes list --filter provider=digitalocean
```

**Output:**
```
┌──────────────────┬──────────────┬────────┬─────────────────┬─────────────┬──────────────┐
│ NAME             │ PROVIDER     │ ROLE   │ PUBLIC IP       │ PRIVATE IP  │ SIZE         │
├──────────────────┼──────────────┼────────┼─────────────────┼─────────────┼──────────────┤
│ do-master-1      │ DigitalOcean │ master │ 167.99.1.1      │ 10.10.0.2   │ s-2vcpu-4gb  │
│ do-master-2      │ DigitalOcean │ master │ 167.99.1.2      │ 10.10.0.3   │ s-2vcpu-4gb  │
│ linode-master-1  │ Linode       │ master │ 172.104.1.1     │ 10.11.0.2   │ g6-standard-2│
│ do-worker-1      │ DigitalOcean │ worker │ 167.99.2.1      │ 10.10.0.10  │ s-2vcpu-4gb  │
│ do-worker-2      │ DigitalOcean │ worker │ 167.99.2.2      │ 10.10.0.11  │ s-2vcpu-4gb  │
│ linode-worker-1  │ Linode       │ worker │ 172.104.2.1     │ 10.11.0.10  │ g6-standard-2│
└──────────────────┴──────────────┴────────┴─────────────────┴─────────────┴──────────────┘

Total: 6 nodes (3 masters, 3 workers)
```

#### nodes add

Add nodes to an existing node pool (horizontal scaling).

**Usage:**
```bash
sloth-kubernetes nodes add [stack] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--pool` | Node pool name to scale |
| `--count` | Number of nodes to add |
| `--yes`, `-y` | Auto-approve |

**Examples:**

```bash
# Add 2 workers to pool
sloth-kubernetes nodes add --pool workers --count 2

# Add nodes with auto-approve
sloth-kubernetes nodes add --pool workers --count 3 --yes
```

**Output:**
```
🚀 Adding 2 nodes to pool: workers

  ✓ Updating Pulumi stack... Done
  ✓ Creating nodes... Done
  ✓ Installing RKE2... Done
  ✓ Configuring WireGuard... Done
  ✓ Joining to cluster... Done

✅ Added 2 nodes successfully

New nodes:
  • do-worker-4 (167.99.2.4)
  • do-worker-5 (167.99.2.5)

⏱  Total time: 4m 12s
```

#### nodes remove

Remove a node from the cluster.

**Usage:**
```bash
sloth-kubernetes nodes remove [stack] [node-name] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--drain` | Drain node before removing |
| `--force` | Force removal without draining |
| `--yes`, `-y` | Auto-approve |

**Examples:**

```bash
# Remove node with drain
sloth-kubernetes nodes remove production do-worker-5 --drain

# Force remove
sloth-kubernetes nodes remove production do-worker-5 --force --yes
```

**Output:**
```
⚠️  Removing node: do-worker-5

  ✓ Draining node... Done (moved 12 pods)
  ✓ Removing from cluster... Done
  ✓ Deleting droplet... Done
  ✓ Cleaning up WireGuard config... Done

✅ Node removed successfully
```

#### nodes ssh

SSH into a cluster node.

**Usage:**
```bash
sloth-kubernetes nodes ssh [stack] [node-name] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--bastion` | Use bastion host for access |
| `--command`, `-c` | Execute command and exit |
| `--user`, `-u` | SSH user (default: root) |

**Examples:**

```bash
# SSH into node
sloth-kubernetes nodes ssh production do-master-1

# Execute command
sloth-kubernetes nodes ssh production do-master-1 -c "kubectl get nodes"

# Via bastion
sloth-kubernetes nodes ssh production do-worker-1 --bastion

# Custom user
sloth-kubernetes nodes ssh production do-worker-1 -u admin
```

#### nodes upgrade

Upgrade Kubernetes version on all nodes.

**Usage:**
```bash
sloth-kubernetes nodes upgrade [stack] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--version` | Target Kubernetes version |
| `--rolling` | Perform rolling upgrade (one node at a time) |
| `--yes`, `-y` | Auto-approve |

**Examples:**

```bash
# Upgrade to specific version
sloth-kubernetes nodes upgrade production --version v1.29.0+rke2r1 --rolling

# With auto-approve
sloth-kubernetes nodes upgrade production --version v1.29.0+rke2r1 --yes
```

---

### 🔐 vpn

Manage WireGuard VPN mesh network.

#### vpn status

Show VPN mesh status and connectivity.

**Usage:**
```bash
sloth-kubernetes vpn status [stack] [flags]
```

**Examples:**

```bash
# Show VPN status
sloth-kubernetes vpn status production
```

**Output:**
```
🔐 WireGuard VPN Status

Server: 167.99.123.45:51820
Subnet: 10.8.0.0/24
Mode: Mesh Networking

┌──────────────────┬─────────────┬────────────┬─────────────┬────────────┐
│ PEER             │ VPN IP      │ STATUS     │ LAST SEEN   │ TRANSFER   │
├──────────────────┼─────────────┼────────────┼─────────────┼────────────┤
│ do-master-1      │ 10.8.0.10   │ ✅ Active  │ 5s ago      │ ↓1.2GB ↑890MB │
│ do-master-2      │ 10.8.0.11   │ ✅ Active  │ 3s ago      │ ↓980MB ↑750MB │
│ linode-master-1  │ 10.8.0.12   │ ✅ Active  │ 2s ago      │ ↓1.1GB ↑820MB │
│ do-worker-1      │ 10.8.0.20   │ ✅ Active  │ 1s ago      │ ↓2.3GB ↑1.2GB │
│ do-worker-2      │ 10.8.0.21   │ ✅ Active  │ 4s ago      │ ↓1.8GB ↑980MB │
│ linode-worker-1  │ 10.8.0.22   │ ✅ Active  │ 6s ago      │ ↓1.5GB ↑890MB │
└──────────────────┴─────────────┴────────────┴─────────────┴────────────┘

Mesh Connectivity: 6/6 peers connected (100%)
```

#### vpn peers

List all VPN peers with connection details.

**Usage:**
```bash
sloth-kubernetes vpn peers [stack] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--format` | Output format: table, json, yaml |

**Examples:**

```bash
# List peers
sloth-kubernetes vpn peers production

# JSON output
sloth-kubernetes vpn peers production --format json
```

#### vpn config

Get WireGuard configuration for a specific node.

**Usage:**
```bash
sloth-kubernetes vpn config [stack] [node-name] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--output`, `-o` | Output to file |
| `--qr` | Generate QR code |

**Examples:**

```bash
# Get config
sloth-kubernetes vpn config production do-worker-1

# Save to file
sloth-kubernetes vpn config production do-worker-1 -o wg0.conf

# Generate QR code (for mobile)
sloth-kubernetes vpn config production do-worker-1 --qr
```

**Output:**
```
[Interface]
PrivateKey = <generated-private-key>
Address = 10.8.0.20/24
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = <server-public-key>
Endpoint = 167.99.123.45:51820
AllowedIPs = 10.8.0.0/24, 10.10.0.0/16, 10.11.0.0/16
PersistentKeepalive = 25
```

#### vpn test

Test VPN connectivity between nodes.

**Usage:**
```bash
sloth-kubernetes vpn test [stack] [flags]
```

**Examples:**

```bash
# Test connectivity
sloth-kubernetes vpn test production
```

**Output:**
```
🧪 Testing VPN connectivity...

┌──────────────────────────────────────────────────────────┐
│ Peer-to-Peer Connectivity Test                          │
├──────────────────────────────────────────────────────────┤
│ do-master-1 → do-master-2         ✅ 2ms               │
│ do-master-1 → linode-master-1     ✅ 45ms              │
│ do-master-2 → linode-worker-1     ✅ 48ms              │
│ linode-master-1 → do-worker-1     ✅ 43ms              │
└──────────────────────────────────────────────────────────┘

✅ All peers reachable
📊 Average latency: 34ms
```

#### vpn join

Add a local or remote machine to the VPN mesh.

**Usage:**
```bash
sloth-kubernetes vpn join [stack] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--name` | Machine name |
| `--output`, `-o` | Output config file |

**Examples:**

```bash
# Join local machine
sloth-kubernetes vpn join production --name my-laptop -o wg0.conf

# Then on your machine:
sudo cp wg0.conf /etc/wireguard/
sudo wg-quick up wg0
```

---

### 🎯 addons

Manage cluster addons and GitOps automation.

#### addons bootstrap

Bootstrap ArgoCD from a Git repository for GitOps workflow.

**Usage:**
```bash
sloth-kubernetes addons bootstrap [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--repo` | Git repository URL |
| `--branch` | Git branch (default: main) |
| `--path` | Path in repository (default: /) |
| `--yes`, `-y` | Auto-approve |

**Examples:**

```bash
# Bootstrap from Git repository
sloth-kubernetes addons bootstrap \
  --repo https://github.com/yourorg/k8s-gitops \
  --branch main

# Custom path
sloth-kubernetes addons bootstrap \
  --repo https://github.com/yourorg/k8s-gitops \
  --path clusters/production
```

**Output:**
```
🚀 Bootstrapping GitOps...

Repository: https://github.com/yourorg/k8s-gitops
Branch: main
Path: /

  ✓ Installing ArgoCD... Done
  ✓ Configuring repository access... Done
  ✓ Creating root application... Done
  ✓ Syncing applications... Done

✅ GitOps bootstrapped successfully!

🌐 Access ArgoCD:
  kubectl port-forward svc/argocd-server -n argocd 8080:443
  URL: https://localhost:8080
  User: admin
  Password: (run: kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath="{.data.password}" | base64 -d)

📋 Applications syncing:
  • cert-manager
  • ingress-nginx
  • monitoring-stack
  • vault
```

#### addons list

List all installed addons.

**Usage:**
```bash
sloth-kubernetes addons list [flags]
```

**Examples:**

```bash
# List addons
sloth-kubernetes addons list
```

**Output:**
```
┌─────────────────────┬────────────┬────────────┬────────────┐
│ ADDON               │ VERSION    │ STATUS     │ NAMESPACE  │
├─────────────────────┼────────────┼────────────┼────────────┤
│ argocd              │ v2.9.3     │ ✅ Synced  │ argocd     │
│ cert-manager        │ v1.13.3    │ ✅ Synced  │ cert-mgr   │
│ ingress-nginx       │ v1.9.5     │ ✅ Synced  │ ingress    │
│ prometheus          │ v2.48.0    │ ✅ Synced  │ monitoring │
│ grafana             │ v10.2.3    │ ✅ Synced  │ monitoring │
└─────────────────────┴────────────┴────────────┴────────────┘
```

#### addons install

Install a specific addon from catalog.

**Usage:**
```bash
sloth-kubernetes addons install [addon-name] [flags]
```

**Examples:**

```bash
# Install cert-manager
sloth-kubernetes addons install cert-manager

# Install with custom values
sloth-kubernetes addons install prometheus --values custom-values.yaml
```

---

### ⚙️ config

Configuration file utilities.

#### config generate

Generate example configuration file.

**Usage:**
```bash
sloth-kubernetes config generate [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--type` | Config type: minimal, basic, advanced, multi-cloud |
| `--output`, `-o` | Output file (default: stdout) |

**Examples:**

```bash
# Generate minimal config
sloth-kubernetes config generate --type minimal > cluster.yaml

# Generate multi-cloud config
sloth-kubernetes config generate --type multi-cloud -o production.yaml

# Generate with VPC/VPN
sloth-kubernetes config generate --type advanced -o advanced.yaml
```

#### config validate

Validate configuration file.

**Usage:**
```bash
sloth-kubernetes config validate [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--file`, `-f` | Config file to validate |

**Examples:**

```bash
# Validate config
sloth-kubernetes config validate -f cluster.yaml
```

**Output:**
```
✅ Configuration valid

Cluster: production-cluster
Providers: DigitalOcean, Linode
Node Pools: 4 (2 master, 2 worker)
Total Nodes: 6
VPC: Auto-create (2)
VPN: Auto-create (WireGuard)
```

---

### 📚 stacks

Manage Pulumi stacks for multi-environment support.

#### stacks list

List all available stacks.

**Usage:**
```bash
sloth-kubernetes stacks list
```

**Output:**
```
┌───────────────┬────────────┬─────────────────────────┐
│ STACK         │ ACTIVE     │ LAST UPDATE             │
├───────────────┼────────────┼─────────────────────────┤
│ production    │ ✅         │ 2025-01-15 14:30:22     │
│ staging       │            │ 2025-01-14 09:15:10     │
│ development   │            │ 2025-01-10 16:45:33     │
└───────────────┴────────────┴─────────────────────────┘
```

---

### 📌 version

Display version information.

**Usage:**
```bash
sloth-kubernetes version
```

**Output:**
```
Sloth Kubernetes v1.0.0

Build Information:
  Go Version: go1.23.1
  Commit: 2d605b4
  Built: 2025-01-15T10:30:00Z
  OS/Arch: darwin/arm64

Dependencies:
  Pulumi: v3.203.0
  Cobra: v1.10.1
```

---

## 📋 Configuration Guide

### Configuration File Structure

Sloth Kubernetes uses Kubernetes-style YAML configuration:

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster

metadata:
  name: cluster-name
  environment: production
  labels:
    key: value

spec:
  providers: {}
  network: {}
  kubernetes: {}
  nodePools: []
  security: {}
  addons: {}
```

### Complete Configuration Reference

<details>
<summary><b>📦 Providers Configuration</b></summary>

```yaml
providers:
  digitalocean:
    enabled: true
    token: ${DIGITALOCEAN_TOKEN}  # Environment variable
    region: nyc3                   # Default region
    monitoring: true               # Enable monitoring
    backups: false                 # Enable backups

    vpc:
      create: true                 # Auto-create VPC
      name: k8s-vpc-do
      cidr: 10.10.0.0/16
      region: nyc3
      enableDns: true
      tags:
        - kubernetes
        - production

  linode:
    enabled: true
    token: ${LINODE_TOKEN}
    region: us-east
    privateIp: true

    vpc:
      create: true
      name: k8s-vpc-linode
      cidr: 10.11.0.0/16
      region: us-east
      enableDns: true
      subnets:
        - label: k8s-subnet-1
          ipv4: 10.11.1.0/24
```

**Supported Regions:**

| Provider | Regions |
|----------|---------|
| **DigitalOcean** | nyc1, nyc3, sfo3, ams3, sgp1, lon1, fra1, tor1, blr1, syd1 |
| **Linode** | us-east, us-west, eu-west, eu-central, ap-south, ap-northeast |

</details>

<details>
<summary><b>🌐 Network Configuration</b></summary>

```yaml
network:
  mode: wireguard              # VPN mode
  cidr: 10.8.0.0/16           # Cluster network CIDR
  podCidr: 10.244.0.0/16      # Pod network CIDR
  serviceCidr: 10.96.0.0/12   # Service network CIDR
  crossProviderNetworking: true
  enableNodePorts: true

  # WireGuard VPN Configuration
  wireguard:
    create: true                    # Auto-create VPN server
    provider: digitalocean          # Provider to host VPN
    region: nyc3
    size: s-1vcpu-1gb              # Server size
    image: ubuntu-22-04-x64
    name: wireguard-vpn-server

    # VPN Settings
    enabled: true
    port: 51820
    clientIpBase: 10.8.0
    subnetCidr: 10.8.0.0/24
    mtu: 1420
    persistentKeepalive: 25
    autoConfig: true
    meshNetworking: true           # Enable full mesh
    allowedIps:
      - 10.8.0.0/24               # VPN subnet
      - 10.10.0.0/16              # DO VPC
      - 10.11.0.0/16              # Linode VPC
    dns:
      - 1.1.1.1
      - 8.8.8.8

  # DNS Configuration
  dns:
    provider: digitalocean
    domain: k8s.example.com
    records:
      - name: api
        type: A
        ttl: 300
      - name: "*.apps"
        type: A
        ttl: 300
```

</details>

<details>
<summary><b>🎯 Kubernetes Configuration</b></summary>

```yaml
kubernetes:
  distribution: rke2              # rke2 or k3s
  version: v1.28.5+rke2r1
  channel: stable                 # stable, latest, or specific version
  networkPlugin: calico           # calico, cilium, canal, flannel
  podCIDR: 10.42.0.0/16
  serviceCIDR: 10.43.0.0/16

  rke2:
    channel: stable
    clusterToken: your-secure-token  # Cluster join token

    # TLS SANs
    tlsSan:
      - api.k8s.example.com
      - 167.99.123.45

    # Disable default components
    disableComponents:
      - rke2-ingress-nginx         # Install via GitOps instead
      - rke2-metrics-server

    # Etcd Snapshots (Backups)
    snapshotScheduleCron: "0 */12 * * *"  # Every 12 hours
    snapshotRetention: 7                   # Keep 7 snapshots

    # Security
    secretsEncryption: true               # Encrypt secrets at rest

    # Profiles
    profiles:
      - cis-1.6                           # CIS benchmark compliance

    # Node taints (for masters)
    nodeTaints:
      - "node-role.kubernetes.io/control-plane:NoSchedule"

    # Additional server args
    serverArgs:
      - "--disable-cloud-controller"

    # Additional agent args
    agentArgs:
      - "--kubelet-arg=max-pods=200"
```

**Available CNI Plugins:**

| Plugin | Best For | Features |
|--------|----------|----------|
| **Calico** | Production, Network Policies | BGP routing, Network policies, Encryption |
| **Cilium** | Advanced networking, eBPF | eBPF-based, Service mesh, Security |
| **Canal** | Calico + Flannel | Simple, Reliable |
| **Flannel** | Simple setups | Basic overlay networking |

</details>

<details>
<summary><b>🖥️ Node Pools Configuration</b></summary>

```yaml
nodePools:
  # DigitalOcean Masters
  do-masters:
    provider: digitalocean
    region: nyc3
    size: s-2vcpu-4gb              # Droplet size
    image: ubuntu-22-04-x64        # OS image
    count: 1                       # Number of nodes
    role: master                   # master or worker

    # Kubernetes labels
    labels:
      node-role.kubernetes.io/master: "true"
      cloud-provider: digitalocean
      environment: production

    # Kubernetes taints
    taints:
      - key: node-role
        value: master
        effect: NoSchedule

    # DigitalOcean specific
    monitoring: true
    backups: false
    ipv6: false
    tags:
      - kubernetes
      - master

  # Linode Workers
  linode-workers:
    provider: linode
    region: us-east
    size: g6-standard-2            # Linode plan
    image: linode/ubuntu22.04
    count: 3
    role: worker

    labels:
      node-role.kubernetes.io/worker: "true"
      cloud-provider: linode
      workload: general

    taints: []

    # Linode specific
    privateIp: true
    backups: false
    tags:
      - kubernetes
      - worker
```

**Instance Sizes:**

| Provider | Size | vCPU | RAM | Price/mo |
|----------|------|------|-----|----------|
| **DigitalOcean** | s-2vcpu-2gb | 2 | 2GB | $18 |
| **DigitalOcean** | s-2vcpu-4gb | 2 | 4GB | $24 |
| **DigitalOcean** | s-4vcpu-8gb | 4 | 8GB | $48 |
| **Linode** | g6-standard-1 | 1 | 2GB | $12 |
| **Linode** | g6-standard-2 | 2 | 4GB | $24 |
| **Linode** | g6-standard-4 | 4 | 8GB | $48 |

</details>

<details>
<summary><b>🔐 Security Configuration</b></summary>

```yaml
security:
  # SSH Configuration
  ssh:
    generateKeys: true             # Auto-generate SSH keys
    keyPath: ~/.ssh/k8s-cluster   # Key path
    allowedIPs:                    # SSH access whitelist
      - 0.0.0.0/0                  # All (VPN protected)

  # Bastion Host (Jump Server)
  bastion:
    enabled: true
    provider: digitalocean
    region: nyc3
    size: s-1vcpu-1gb
    allowedIPs:
      - 1.2.3.4/32                # Your IP only

  # RBAC
  rbac:
    enabled: true

  # Pod Security
  podSecurity:
    enabled: true
    defaultPolicy: restricted      # restricted, baseline, privileged

  # Network Policies
  networkPolicies:
    enabled: true
    defaultDeny: true

  # TLS/Certificates
  tls:
    autoGenerate: true
```

</details>

<details>
<summary><b>🎯 Addons Configuration</b></summary>

```yaml
addons:
  # GitOps with ArgoCD
  gitops:
    enabled: true
    repository: https://github.com/yourorg/k8s-gitops
    branch: main
    path: addons/

    # ArgoCD configuration
    argocd:
      version: v2.9.3
      ha: true                     # High availability

  # Monitoring Stack
  monitoring:
    enabled: true
    prometheus:
      enabled: true
      retention: 15d
      storageSize: 50Gi

    grafana:
      enabled: true
      adminPassword: ${GRAFANA_PASSWORD}

    alertmanager:
      enabled: true

  # Storage
  storage:
    csi:
      digitalocean:
        enabled: true
      linode:
        enabled: true

    storageClasses:
      - name: fast
        provisioner: do-csi
        parameters:
          type: pd-ssd
      - name: standard
        provisioner: linode-csi
        parameters:
          type: pd-standard
```

</details>

### Environment Variables

Sensitive values can be referenced using environment variables:

```yaml
providers:
  digitalocean:
    token: ${DIGITALOCEAN_TOKEN}
  linode:
    token: ${LINODE_TOKEN}

addons:
  monitoring:
    grafana:
      adminPassword: ${GRAFANA_PASSWORD}
```

**Setting environment variables:**

```bash
export DIGITALOCEAN_TOKEN="dop_v1_xxxxx"
export LINODE_TOKEN="xxxxx"
export GRAFANA_PASSWORD="secure-password"
```

---

## 🎨 Configuration Examples

### Example 1: Minimal Single-Provider Cluster

Perfect for: Development, testing, small projects

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: dev-cluster

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    - name: masters
      provider: digitalocean
      count: 3
      roles: [master]
      size: s-2vcpu-4gb

    - name: workers
      provider: digitalocean
      count: 3
      roles: [worker]
      size: s-2vcpu-4gb
```

**Cost:** ~$144/month
**Nodes:** 6 (3 masters, 3 workers)
**Providers:** 1 (DigitalOcean)

---

### Example 2: Multi-Cloud Production Cluster

Perfect for: Production, high availability, geographic distribution

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: production-cluster
  environment: production

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.10.0.0/16

    linode:
      enabled: true
      token: ${LINODE_TOKEN}
      region: us-east
      vpc:
        create: true
        cidr: 10.11.0.0/16

  network:
    wireguard:
      create: true
      provider: digitalocean
      meshNetworking: true

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    rke2:
      secretsEncryption: true
      snapshotScheduleCron: "0 */12 * * *"
      snapshotRetention: 7

  nodePools:
    # 1 DO master + 2 Linode masters = 3 masters (HA)
    do-masters:
      provider: digitalocean
      count: 1
      roles: [master]
      size: s-2vcpu-4gb

    linode-masters:
      provider: linode
      count: 2
      roles: [master]
      size: g6-standard-2

    # Workers across both providers
    do-workers:
      provider: digitalocean
      count: 2
      roles: [worker]
      size: s-2vcpu-4gb

    linode-workers:
      provider: linode
      count: 2
      roles: [worker]
      size: g6-standard-2

  security:
    bastion:
      enabled: true
    podSecurity:
      enabled: true
      defaultPolicy: restricted
```

**Cost:** ~$180/month
**Nodes:** 7 (3 masters, 4 workers)
**Providers:** 2 (DigitalOcean + Linode)
**HA:** Yes (masters across providers)

---

### Example 3: Advanced with Monitoring & GitOps

Perfect for: Enterprise, full observability, GitOps workflow

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: enterprise-cluster
  environment: production

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.10.0.0/16

  network:
    wireguard:
      create: true
      provider: digitalocean
    dns:
      provider: digitalocean
      domain: k8s.example.com

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    networkPlugin: calico
    rke2:
      secretsEncryption: true
      profiles:
        - cis-1.6
      disableComponents:
        - rke2-ingress-nginx

  nodePools:
    masters:
      provider: digitalocean
      count: 3
      roles: [master]
      size: s-2vcpu-4gb

    workers:
      provider: digitalocean
      count: 5
      roles: [worker]
      size: s-4vcpu-8gb

  security:
    bastion:
      enabled: true
    podSecurity:
      enabled: true
      defaultPolicy: restricted
    networkPolicies:
      enabled: true

  addons:
    gitops:
      enabled: true
      repository: https://github.com/yourorg/k8s-gitops
      argocd:
        ha: true

    monitoring:
      enabled: true
      prometheus:
        retention: 30d
        storageSize: 100Gi
      grafana:
        enabled: true
```

**Cost:** ~$384/month
**Nodes:** 8 (3 masters, 5 workers)
**Features:** GitOps, Monitoring, CIS compliance, Bastion

---

## 🚀 Deployment Workflow

### Complete Deployment Example

```bash
# Step 1: Create configuration
cat > production.yaml <<EOF
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: production

spec:
  providers:
    digitalocean:
      enabled: true
      token: ${DIGITALOCEAN_TOKEN}
      region: nyc3
      vpc:
        create: true
        cidr: 10.10.0.0/16

  network:
    wireguard:
      create: true
      provider: digitalocean

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1

  nodePools:
    masters:
      provider: digitalocean
      count: 3
      roles: [master]
      size: s-2vcpu-4gb
    workers:
      provider: digitalocean
      count: 3
      roles: [worker]
      size: s-2vcpu-4gb
EOF

# Step 2: Validate configuration
sloth-kubernetes config validate -f production.yaml

# Step 3: Preview deployment (dry-run)
sloth-kubernetes deploy --config production.yaml --dry-run

# Step 4: Deploy cluster
export DIGITALOCEAN_TOKEN="dop_v1_xxxxx"
sloth-kubernetes deploy --config production.yaml

# Step 5: Get kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Step 6: Verify cluster
kubectl get nodes
kubectl get pods --all-namespaces

# Step 7: Check cluster status
sloth-kubernetes status

# Step 8: Bootstrap GitOps (optional)
sloth-kubernetes addons bootstrap \
  --repo https://github.com/yourorg/k8s-gitops

# Step 9: Deploy your applications
kubectl apply -f your-app.yaml
```

---

## 🌐 Network Architecture

### VPN Mesh Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                    WireGuard Mesh Network                        │
│                      (10.8.0.0/24)                               │
│                                                                  │
│                  ┌──────────────────────┐                       │
│                  │   VPN Server         │                       │
│                  │   167.99.123.45:51820│                       │
│                  │   IP: 10.8.0.1       │                       │
│                  └──────────┬───────────┘                       │
│                             │                                    │
│              ┌──────────────┼──────────────┐                    │
│              │              │              │                     │
│    ┌─────────▼──────┐  ┌───▼──────┐  ┌───▼──────────┐         │
│    │ DigitalOcean   │  │ Linode   │  │ DigitalOcean │         │
│    │ VPC            │  │ VPC      │  │ Nodes        │         │
│    │ 10.10.0.0/16   │  │10.11.0.0 │  │              │         │
│    │                │  │    /16   │  │              │         │
│    └────────────────┘  └──────────┘  └──────────────┘         │
│                                                                  │
│  All nodes communicate via encrypted WireGuard tunnel           │
│  • Full mesh: Every node can reach every other node            │
│  • Encrypted: All traffic encrypted with modern crypto         │
│  • Low latency: Direct peer-to-peer connections                │
│  • Cross-cloud: Transparent routing between providers          │
└─────────────────────────────────────────────────────────────────┘
```

### Network Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  Pod-to-Pod Communication (Cross-Provider)                       │
└─────────────────────────────────────────────────────────────────┘

Pod A (DO)                                          Pod B (Linode)
  ↓                                                       ↑
Calico CNI (10.42.0.10)                    Calico CNI (10.42.1.20)
  ↓                                                       ↑
Node A (DO)                                        Node B (Linode)
Private IP: 10.10.0.5                        Private IP: 10.11.0.8
VPN IP: 10.8.0.10                            VPN IP: 10.8.0.12
  ↓                                                       ↑
  └──────────► WireGuard Tunnel ─────────────────────────┘
             (Encrypted, 10.8.0.0/24)

✅ Encrypted end-to-end
✅ No public internet exposure
✅ Cross-provider routing
✅ Pod network isolation
```

### Security Layers

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 7: Application Layer                                     │
│  • Kubernetes Network Policies                                  │
│  • Pod Security Policies                                        │
│  • RBAC Authorization                                           │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4-6: Transport/Session Layer                             │
│  • WireGuard VPN (ChaCha20 encryption)                          │
│  • TLS for Kubernetes API                                       │
│  • Encrypted etcd storage                                       │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: Network Layer                                         │
│  • Private VPCs (isolated networks)                             │
│  • Firewall rules (UFW)                                         │
│  • Security groups                                              │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: Data Link Layer                                       │
│  • Provider network isolation                                   │
│  • VLAN separation                                              │
├─────────────────────────────────────────────────────────────────┤
│  Layer 1: Physical Layer                                        │
│  • Provider datacenter security                                 │
│  • Physical network isolation                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 💼 Use Cases

### 1. High Availability Production Cluster

**Scenario:** You need a highly available Kubernetes cluster that can survive a cloud provider outage.

**Solution:**
- Deploy masters across multiple providers
- Use WireGuard mesh for seamless communication
- Implement automated etcd backups

```yaml
nodePools:
  do-masters:
    provider: digitalocean
    count: 1
    roles: [master]

  linode-masters:
    provider: linode
    count: 2
    roles: [master]
```

**Benefits:**
- ✅ Survive entire provider outage
- ✅ Automatic failover
- ✅ Geographic distribution

---

### 2. Cost-Optimized Cluster

**Scenario:** You want to optimize costs by using the cheapest regions/instances.

**Solution:**
- Mix and match providers
- Use smaller instances for non-critical workloads
- Scale workers independently

```yaml
nodePools:
  linode-workers:
    provider: linode
    size: g6-standard-1  # Cheaper than DO
    count: 5
```

---

### 3. Development/Staging Environment

**Scenario:** Quick cluster for testing without complex setup.

**Solution:**
- Single provider
- Minimal configuration
- Fast deployment

```bash
sloth-kubernetes config generate --type minimal > dev.yaml
sloth-kubernetes deploy --config dev.yaml
```

---

### 4. GitOps-First Infrastructure

**Scenario:** Fully automated infrastructure with GitOps workflow.

**Solution:**
- Bootstrap ArgoCD automatically
- Self-managing applications
- Git as source of truth

```bash
# Deploy cluster
sloth-kubernetes deploy --config cluster.yaml

# Bootstrap GitOps
sloth-kubernetes addons bootstrap \
  --repo https://github.com/yourorg/k8s-gitops

# All applications auto-sync from Git
```

---

## 🔧 Advanced Topics

### State Management

Sloth Kubernetes uses Pulumi for state management. By default, state is stored locally in `~/.pulumi/`.

#### Local State (Default)

```bash
# State stored in: ~/.pulumi/stacks/
ls ~/.pulumi/stacks/
```

#### Remote State (S3)

```bash
# Set S3 backend
export PULUMI_BACKEND_URL="s3://my-bucket/sloth-kubernetes"

# Deploy
sloth-kubernetes deploy --config cluster.yaml
```

#### Remote State (Azure Blob)

```bash
export PULUMI_BACKEND_URL="azblob://my-container"
export AZURE_STORAGE_ACCOUNT="mystorageaccount"
export AZURE_STORAGE_KEY="xxxxx"
```

#### Remote State (Google Cloud Storage)

```bash
export PULUMI_BACKEND_URL="gs://my-bucket/sloth-kubernetes"
```

---

### Multi-Environment Management

Manage multiple clusters (dev, staging, production) with stacks:

```bash
# Development
sloth-kubernetes deploy --config dev.yaml --stack dev

# Staging
sloth-kubernetes deploy --config staging.yaml --stack staging

# Production
sloth-kubernetes deploy --config production.yaml --stack production

# List all stacks
sloth-kubernetes stacks list

# Switch between environments
kubectl config use-context dev
kubectl config use-context production
```

---

### Bastion Host (Jump Server)

For enhanced security, use a bastion host:

```yaml
security:
  bastion:
    enabled: true
    provider: digitalocean
    region: nyc3
    size: s-1vcpu-1gb
    allowedIPs:
      - 1.2.3.4/32  # Your IP only
```

**SSH via bastion:**

```bash
# Direct SSH (blocked)
ssh root@do-worker-1  # ❌ Blocked

# Via bastion
sloth-kubernetes nodes ssh production do-worker-1 --bastion  # ✅ Works
```

---

### Custom RKE2 Configuration

Advanced RKE2 options:

```yaml
kubernetes:
  rke2:
    # Custom registry
    privateRegistries:
      - url: registry.example.com
        username: ${REGISTRY_USER}
        password: ${REGISTRY_PASS}

    # Custom manifests (installed on bootstrap)
    customManifests:
      - https://raw.githubusercontent.com/yourorg/manifests/main/custom.yaml

    # SELinux
    selinux: true

    # Additional mount points
    extraMounts:
      - source: /host/path
        destination: /container/path
        type: bind
        options:
          - rbind
          - rw
```

---

### Monitoring Integration

Complete observability stack:

```yaml
addons:
  monitoring:
    enabled: true

    prometheus:
      enabled: true
      retention: 30d
      storageSize: 100Gi
      storageClass: fast
      replicas: 2

    grafana:
      enabled: true
      adminPassword: ${GRAFANA_PASSWORD}
      persistence:
        enabled: true
        size: 10Gi
      dashboards:
        - https://grafana.com/api/dashboards/12345/revisions/1/download

    alertmanager:
      enabled: true
      config:
        route:
          receiver: 'slack'
        receivers:
          - name: 'slack'
            slack_configs:
              - api_url: ${SLACK_WEBHOOK}
                channel: '#alerts'
```

---

## 🧪 Testing

### Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/config

# With coverage
go test ./pkg/config -cover

# Verbose
go test ./pkg/config -v

# Generate coverage report
go test ./pkg/config -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

Current test coverage: **46.1%** (71 tests)

| Package | Coverage | Tests |
|---------|----------|-------|
| `pkg/config` | 53.4% | 56 |
| `pkg/vpc` | 2.1% | 9 |
| `pkg/vpn` | 7.7% | 14 |

See [TESTS_COVERAGE_REPORT.md](./TESTS_COVERAGE_REPORT.md) for detailed coverage report.

---

## 🔍 Troubleshooting

### Common Issues

<details>
<summary><b>❌ Deployment fails: "token is invalid"</b></summary>

**Cause:** Invalid or expired API token

**Solution:**
```bash
# Verify token
export DIGITALOCEAN_TOKEN="dop_v1_xxxxx"
curl -X GET "https://api.digitalocean.com/v2/account" \
  -H "Authorization: Bearer $DIGITALOCEAN_TOKEN"

# If invalid, generate new token at:
# https://cloud.digitalocean.com/account/api/tokens
```

</details>

<details>
<summary><b>❌ WireGuard connection fails</b></summary>

**Cause:** Firewall blocking UDP port 51820

**Solution:**
```bash
# Check VPN status
sloth-kubernetes vpn status

# Test connectivity
sloth-kubernetes vpn test

# Verify firewall rules on VPN server
ssh root@vpn-server
ufw status
ufw allow 51820/udp
```

</details>

<details>
<summary><b>❌ kubectl: "Unable to connect to server"</b></summary>

**Cause:** Kubeconfig not set or incorrect

**Solution:**
```bash
# Get fresh kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Verify
kubectl cluster-info
kubectl get nodes

# Check API endpoint
grep server ~/.kube/config
```

</details>

<details>
<summary><b>❌ Nodes not joining cluster</b></summary>

**Cause:** Network connectivity or RKE2 installation issue

**Solution:**
```bash
# SSH into node
sloth-kubernetes nodes ssh production do-worker-1

# Check RKE2 status
systemctl status rke2-agent

# Check logs
journalctl -u rke2-agent -f

# Verify WireGuard
wg show
ping 10.8.0.1  # VPN server
```

</details>

<details>
<summary><b>❌ Out of disk space on nodes</b></summary>

**Cause:** Container images filling disk

**Solution:**
```bash
# SSH into node
sloth-kubernetes nodes ssh production do-worker-1

# Clean up Docker images
crictl rmi --prune

# Check disk usage
df -h
du -sh /var/lib/rancher/rke2
```

</details>

### Debug Mode

Enable verbose output for debugging:

```bash
# Verbose deployment
sloth-kubernetes deploy --config cluster.yaml --verbose

# Very verbose (includes Pulumi debug)
export PULUMI_DEBUG_COMMANDS=true
sloth-kubernetes deploy --config cluster.yaml --verbose
```

### Logs

Check logs for issues:

```bash
# RKE2 server logs (master)
ssh root@master-1
journalctl -u rke2-server -f

# RKE2 agent logs (worker)
ssh root@worker-1
journalctl -u rke2-agent -f

# WireGuard logs
journalctl -u wg-quick@wg0 -f
```

---

## ❓ FAQ

<details>
<summary><b>Do I need Pulumi CLI installed?</b></summary>

**No!** Sloth Kubernetes uses **Pulumi Automation API**, which is a Go library that embeds all Pulumi functionality into the binary. No external CLI needed.

</details>

<details>
<summary><b>Where is the infrastructure state stored?</b></summary>

By default, state is stored locally in `~/.pulumi/stacks/`. You can configure remote backends (S3, Azure Blob, GCS, Pulumi Cloud) using the `PULUMI_BACKEND_URL` environment variable.

</details>

<details>
<summary><b>Can I use my existing VPC?</b></summary>

Yes! Set `create: false` and provide the VPC ID:

```yaml
providers:
  digitalocean:
    vpc:
      create: false
      id: "vpc-existing-id"
```

</details>

<details>
<summary><b>Do I need a pre-existing WireGuard server?</b></summary>

No! Set `create: true` in the WireGuard configuration and Sloth Kubernetes will automatically deploy and configure a VPN server for you.

</details>

<details>
<summary><b>Can I add more nodes after initial deployment?</b></summary>

Yes! Use the `nodes add` command:

```bash
sloth-kubernetes nodes add --pool workers --count 2
```

</details>

<details>
<summary><b>How do I upgrade Kubernetes version?</b></summary>

Update the version in your config file and redeploy:

```yaml
kubernetes:
  version: v1.29.0+rke2r1  # Updated version
```

```bash
sloth-kubernetes deploy --config cluster.yaml
```

Or use the upgrade command:

```bash
sloth-kubernetes nodes upgrade --version v1.29.0+rke2r1 --rolling
```

</details>

<details>
<summary><b>What happens if one cloud provider goes down?</b></summary>

If you've distributed your master nodes across multiple providers (recommended), your cluster will continue to function. Pods on the affected provider's nodes will be rescheduled to healthy nodes on other providers automatically.

</details>

<details>
<summary><b>How much does it cost?</b></summary>

Cost depends on your configuration:

- **Minimal (6 nodes, DO only):** ~$144/month
- **Multi-cloud (7 nodes):** ~$180/month
- **Enterprise (8+ nodes):** ~$384/month+

VPN server adds minimal cost (~$6/month for s-1vcpu-1gb).

</details>

<details>
<summary><b>Can I use this in production?</b></summary>

Yes! Sloth Kubernetes is designed for production use with:
- High availability (multi-master)
- Automated backups (etcd snapshots)
- Security hardening (CIS profiles, secrets encryption)
- Multi-cloud resilience
- GitOps support

</details>

<details>
<summary><b>How do I backup my cluster?</b></summary>

RKE2 automatically creates etcd snapshots based on your configuration:

```yaml
kubernetes:
  rke2:
    snapshotScheduleCron: "0 */12 * * *"  # Every 12 hours
    snapshotRetention: 7                   # Keep 7 snapshots
```

Snapshots are stored on master nodes at `/var/lib/rancher/rke2/server/db/snapshots/`.

</details>

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

### 1. Fork the Repository

```bash
git clone https://github.com/yourusername/sloth-kubernetes.git
cd sloth-kubernetes
```

### 2. Create Feature Branch

```bash
git checkout -b feature/amazing-feature
```

### 3. Make Changes

```bash
# Make your changes
vim pkg/something/new.go

# Add tests
vim pkg/something/new_test.go

# Run tests
go test ./...
```

### 4. Commit Changes

```bash
git add .
git commit -m "Add amazing feature"
```

### 5. Push and Create PR

```bash
git push origin feature/amazing-feature
```

Then open a Pull Request on GitHub.

### Development Setup

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run linters
golangci-lint run

# Build
go build -o sloth-kubernetes
```

### Code Style

- Follow standard Go conventions
- Write tests for new features
- Document public APIs
- Keep commits atomic and descriptive

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Sloth Kubernetes is built on top of excellent open-source projects:

- **[Pulumi](https://pulumi.com)** - Infrastructure as Code framework
- **[RKE2](https://docs.rke2.io/)** - Production-ready Kubernetes distribution
- **[WireGuard](https://www.wireguard.com/)** - Fast, modern VPN protocol
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework for Go
- **[ArgoCD](https://argo-cd.readthedocs.io/)** - GitOps continuous delivery
- **[Calico](https://www.tigera.io/project-calico/)** - Kubernetes networking and security

Special thanks to the open-source community for making projects like this possible.

---

## 📧 Support & Community

- 📖 **Documentation:** [Full docs](./docs/)
- 🐛 **Issues:** [GitHub Issues](https://github.com/yourusername/sloth-kubernetes/issues)
- 💬 **Discussions:** [GitHub Discussions](https://github.com/yourusername/sloth-kubernetes/discussions)
- 📬 **Email:** support@example.com
- 🐦 **Twitter:** [@slothkubernetes](https://twitter.com/slothkubernetes)

---

<div align="center">

**🦥 Sloth Kubernetes - Deploy Kubernetes clusters, slowly but surely**

Made with ❤️ by the open-source community

[⭐ Star us on GitHub](https://github.com/yourusername/sloth-kubernetes) • [📖 Read the docs](./docs/) • [🚀 Get started](#-quick-start)

</div>
