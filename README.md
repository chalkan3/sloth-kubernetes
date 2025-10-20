# 🦥 Sloth Kubernetes

> Multi-cloud Kubernetes cluster deployment tool with RKE2, WireGuard VPN, and automated networking

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Test Coverage](https://img.shields.io/badge/coverage-46.1%25-yellow)](./TESTS_COVERAGE_REPORT.md)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Sloth Kubernetes is a **single-binary CLI tool** for deploying production-ready Kubernetes clusters across multiple cloud providers with automated VPC creation, WireGuard VPN mesh networking, and GitOps-based addon management.

## ⚡ Quick Start (No Dependencies!)

```bash
# Clone and build
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes
go build -o sloth-kubernetes

# Deploy cluster
./sloth-kubernetes deploy --config cluster.yaml
```

**That's it!** No need to install Pulumi CLI, Terraform, or any other tools. Everything is embedded in the single binary.

---

## ✨ Key Features

### 🎯 Zero External Dependencies
- ✅ **Single binary** - No Pulumi CLI installation required
- ✅ **Embedded Pulumi** - Uses Pulumi Automation API (library only)
- ✅ **Self-contained** - All logic built into the Go binary
- ✅ **State Management** - Automatic state handling (local or remote)

### 🌐 Multi-Cloud Support
- **DigitalOcean** - Droplets, VPCs, DNS
- **Linode** - Instances, VPCs, NodeBalancers
- Deploy nodes across both providers in a single cluster

### 🔒 Automated Networking
- **Auto VPC Creation** - Automatic VPC provisioning with custom CIDRs
- **WireGuard VPN Mesh** - Secure multi-cloud networking with automatic peer configuration
- **Sequential Deployment** - VPC → VPN → Cluster in one command

### 🚀 Kubernetes Distribution
- **RKE2** - Production-ready Kubernetes with 40+ configuration options
- **K3s** - Lightweight Kubernetes for smaller deployments
- **HA Masters** - Odd-number master nodes for high availability
- **Etcd Snapshots** - Automatic backup scheduling

### 🎯 GitOps Ready
- **ArgoCD Bootstrap** - Automatic ArgoCD installation from Git repository
- **Self-Managed** - GitOps repo manages its own deployment
- **Declarative Config** - Kubernetes-style YAML configuration

---

## 📦 Installation

### Option 1: From Source (Recommended)

```bash
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes
go build -o sloth-kubernetes
sudo mv sloth-kubernetes /usr/local/bin/
```

### Option 2: Quick Build

```bash
go install github.com/chalkan3/sloth-kubernetes@latest
```

### Prerequisites

**Only these are required:**
- ✅ Go 1.23+ (for building)
- ✅ Cloud provider tokens (DigitalOcean and/or Linode)
- ✅ SSH key pair

**NOT required:**
- ❌ Pulumi CLI
- ❌ Terraform
- ❌ kubectl (for deployment, but needed later to manage cluster)
- ❌ Docker

---

## 🎮 Usage

### 1. Create Configuration

Create `cluster.yaml`:

```yaml
apiVersion: sloth-kubernetes.io/v1
kind: ClusterConfig
metadata:
  name: production-cluster

providers:
  digitalocean:
    enabled: true
    token: ${DIGITALOCEAN_TOKEN}
    region: nyc3
    vpc:
      create: true              # ✨ Auto-create VPC
      name: k8s-vpc
      cidr: 10.10.0.0/16

network:
  mode: wireguard
  wireguard:
    create: true                # ✨ Auto-create VPN
    provider: digitalocean
    region: nyc3
    port: 51820
    subnetCidr: 10.8.0.0/24
    meshNetworking: true

kubernetes:
  distribution: rke2
  version: v1.28.5+rke2r1
  networkPlugin: calico

nodePools:
  masters:
    provider: digitalocean
    count: 3                    # HA with 3 masters
    size: s-2vcpu-4gb
    roles: [master]
  workers:
    provider: digitalocean
    count: 3
    size: s-2vcpu-4gb
    roles: [worker]
```

### 2. Deploy

```bash
# Preview what will be created
sloth-kubernetes deploy --config cluster.yaml --dry-run

# Deploy for real
sloth-kubernetes deploy --config cluster.yaml
```

### 3. Access Cluster

```bash
# Get kubeconfig
sloth-kubernetes kubeconfig -o ~/.kube/config

# Verify cluster
kubectl get nodes
```

---

## 🔧 CLI Commands

### Cluster Management

```bash
# Deploy cluster
sloth-kubernetes deploy --config cluster.yaml

# Preview changes (dry-run)
sloth-kubernetes deploy --config cluster.yaml --dry-run

# Check status
sloth-kubernetes status

# Destroy cluster
sloth-kubernetes destroy
```

### Node Management

```bash
# List nodes
sloth-kubernetes nodes list

# Add workers
sloth-kubernetes nodes add --count 2 --pool workers

# Remove node
sloth-kubernetes nodes remove node-name

# SSH into node
sloth-kubernetes nodes ssh master-1

# Upgrade Kubernetes
sloth-kubernetes nodes upgrade --version v1.29.0+rke2r1
```

### GitOps Addons

```bash
# Bootstrap ArgoCD from Git repo
sloth-kubernetes addons bootstrap --repo https://github.com/user/gitops-repo

# List addons
sloth-kubernetes addons list

# Install addon
sloth-kubernetes addons install cert-manager
```

### Configuration

```bash
# Generate example config
sloth-kubernetes config generate > cluster.yaml

# Validate config
sloth-kubernetes config validate -f cluster.yaml
```

---

## 🏗️ How It Works

### No Pulumi CLI Required!

Sloth Kubernetes uses **Pulumi Automation API**, which is a Go library that embeds all Pulumi functionality directly into the binary:

```go
// This is what happens internally:
stack, err := auto.UpsertStackInlineSource(ctx, stackName, "sloth-kubernetes", program)
result, err := stack.Up(ctx)
```

**Benefits:**
- ✅ No external CLI installation
- ✅ No `pulumi` command needed
- ✅ No separate Pulumi.yaml files
- ✅ Everything is programmatic
- ✅ State managed automatically

### State Storage

By default, state is stored locally in `~/.pulumi/`. You can configure remote state:

```yaml
# In your config
pulumi:
  backend: "s3://my-bucket/sloth-kubernetes"
```

Or set environment variable:
```bash
export PULUMI_BACKEND_URL="s3://my-bucket"
```

---

## 🌐 Architecture

### Deployment Flow

```
┌────────────────────────────────────────────────┐
│ 1. VPC Creation                                │
│    ├── DigitalOcean VPC (10.10.0.0/16)        │
│    └── Linode VPC (10.11.0.0/16)              │
└────────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────────┐
│ 2. VPN Setup                                   │
│    ├── WireGuard server deployment            │
│    ├── Key generation                          │
│    └── Mesh networking configuration           │
└────────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────────┐
│ 3. Cluster Deployment                          │
│    ├── SSH key provisioning                    │
│    ├── Master nodes (3 for HA)                 │
│    ├── Worker nodes                            │
│    ├── RKE2 installation                       │
│    └── Cluster bootstrap                       │
└────────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────────┐
│ 4. Network Configuration                       │
│    ├── WireGuard clients on all nodes         │
│    ├── Peer-to-peer mesh                      │
│    └── Cross-cloud connectivity                │
└────────────────────────────────────────────────┘
```

### Network Topology

```
┌─────────────────────────────────────────────────────┐
│            WireGuard VPN Mesh (10.8.0.0/24)         │
│                                                      │
│  ┌──────────────────┐      ┌──────────────────┐    │
│  │  DigitalOcean    │      │     Linode       │    │
│  │  VPC             │◄────►│     VPC          │    │
│  │  10.10.0.0/16    │ VPN  │  10.11.0.0/16    │    │
│  │                  │      │                  │    │
│  │  • 3 Masters     │      │  • Optional      │    │
│  │  • 3 Workers     │      │    nodes         │    │
│  └──────────────────┘      └──────────────────┘    │
│                                                      │
│  All nodes communicate via encrypted WireGuard      │
└─────────────────────────────────────────────────────┘
```

---

## 📖 Documentation

Comprehensive documentation available:

- [**VPC + VPN + Cluster Guide**](./VPC_VPN_CLUSTER.md) - 70+ pages complete guide
- [**Test Coverage Report**](./TESTS_COVERAGE_REPORT.md) - 46.1% coverage, 71 tests
- [**Node Management**](./NODES_MANAGEMENT.md) - Node operations guide
- [**GitOps Addons**](./ADDONS_GITOPS.md) - ArgoCD and addon management
- [**Configuration Examples**](./examples/) - Sample configurations
- [**State Management**](./STATE_MANAGEMENT.md) - Pulumi state guide
- [**Changelog**](./CHANGELOG.md) - Version history

---

## 🔧 Configuration Examples

### Multi-Cloud Cluster

```yaml
providers:
  digitalocean:
    enabled: true
    vpc:
      create: true
      cidr: 10.10.0.0/16

  linode:
    enabled: true
    vpc:
      create: true
      cidr: 10.11.0.0/16

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

### With RKE2 Options

```yaml
kubernetes:
  distribution: rke2
  rke2:
    channel: stable
    snapshotScheduleCron: "0 */12 * * *"
    snapshotRetention: 5
    secretsEncryption: true
    disableComponents:
      - rke2-ingress-nginx
    profiles:
      - cis-1.6
```

See [examples/](./examples/) for more.

---

## 🧪 Testing

**46.1% test coverage** with **71 tests** (all passing ✅)

```bash
# Run all tests
go test ./pkg/vpc ./pkg/vpn ./pkg/config

# With coverage
go test ./pkg/vpc ./pkg/vpn ./pkg/config -cover

# HTML report
go test ./pkg/vpc ./pkg/vpn ./pkg/config -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Coverage by Package

- `pkg/config`: 53.4% (56 tests)
- `pkg/vpc`: 2.1% (9 tests)
- `pkg/vpn`: 7.7% (14 tests)

See [TESTS_COVERAGE_REPORT.md](./TESTS_COVERAGE_REPORT.md) for details.

---

## ❓ FAQ

### Do I need Pulumi CLI installed?

**No!** Sloth Kubernetes uses Pulumi Automation API (a Go library), not the CLI. Everything is embedded in the binary.

### Where is the state stored?

By default: `~/.pulumi/stacks/`. You can configure S3, Azure Blob, GCS, or Pulumi Cloud.

### Can I use my existing VPC?

Yes! Set `create: false` and provide the VPC ID:

```yaml
vpc:
  create: false
  id: "vpc-existing"
```

### Do I need a WireGuard server?

No! Set `create: true` and it will be created automatically:

```yaml
wireguard:
  create: true
  provider: digitalocean
```

### How do I update the cluster?

```bash
# Update config file
vim cluster.yaml

# Preview changes
sloth-kubernetes deploy --config cluster.yaml --dry-run

# Apply
sloth-kubernetes deploy --config cluster.yaml
```

---

## 🚀 Roadmap

- [ ] AWS, GCP, Azure support
- [ ] Terraform backend support
- [ ] Cluster autoscaling
- [ ] Monitoring stack (Prometheus/Grafana)
- [ ] Backup/restore automation
- [ ] Web UI dashboard

---

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repo
2. Create feature branch
3. Commit changes
4. Push and open PR

---

## 📝 License

MIT License - see [LICENSE](LICENSE)

---

## 🙏 Acknowledgments

- [Pulumi](https://pulumi.com) - Infrastructure as Code
- [RKE2](https://docs.rke2.io/) - Kubernetes distribution
- [WireGuard](https://www.wireguard.com/) - VPN protocol
- [ArgoCD](https://argo-cd.readthedocs.io/) - GitOps

---

## 📧 Support

- 📖 [Documentation](./VPC_VPN_CLUSTER.md)
- 🐛 [Issues](https://github.com/chalkan3/sloth-kubernetes/issues)
- 💬 [Discussions](https://github.com/chalkan3/sloth-kubernetes/discussions)

---

**🦥 Built with Sloth Kubernetes - Deploy Kubernetes slowly but surely!**
