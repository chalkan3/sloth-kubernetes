# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added - Integrated VPC + VPN + Cluster Workflow

#### Complete Infrastructure Automation

**Novo fluxo integrado**: VPC → VPN → Cluster tudo em um único YAML!

```yaml
providers:
  digitalocean:
    vpc:
      create: true              # Auto-criar VPC
      cidr: 10.10.0.0/16

network:
  wireguard:
    create: true                # Auto-criar VPN
    provider: digitalocean

nodePools:
  masters:
    provider: digitalocean
    count: 3
```

Um único comando: `kubernetes-create deploy --config cluster.yaml`

#### New Features

1. **Auto-Create VPCs**
   - DigitalOcean VPC creation
   - Linode VPC creation with subnets
   - Configurable CIDR blocks
   - DNS and hostnames support
   - Internet/NAT gateways
   - Tags and labels

2. **Auto-Create WireGuard VPN Server**
   - Provision VPN server on any provider
   - Automatic WireGuard installation
   - Key generation and management
   - Mesh networking setup
   - Cross-provider connectivity
   - Configurable subnets and routes

3. **Integrated Workflow**
   - Single YAML configuration
   - Sequential deployment (VPC → VPN → Cluster)
   - Automatic network configuration
   - Cross-provider routing
   - VPN client auto-configuration on all nodes

#### New Packages

**pkg/vpc/vpc.go** - VPC Management
- `VPCManager` - Manages VPC creation across providers
- `CreateDigitalOceanVPC()` - DigitalOcean VPC creation
- `CreateLinodeVPC()` - Linode VPC creation with subnets
- `CreateAllVPCs()` - Create VPCs for all enabled providers
- `GetOrCreateVPC()` - Get existing or create new VPC

**pkg/vpn/wireguard.go** - WireGuard VPN Management
- `WireGuardManager` - Manages WireGuard server creation
- `CreateWireGuardServer()` - Create VPN server on any provider
- `createDigitalOceanWireGuard()` - DigitalOcean implementation
- `createLinodeWireGuard()` - Linode implementation
- `ConfigureWireGuardClient()` - Generate client configurations
- `GetWireGuardInstallCommand()` - Peer management commands

#### Configuration Types

**Enhanced types in pkg/config/types.go:**

- `VPCConfig` - Comprehensive VPC configuration
  - Creation settings (create, ID, name, CIDR, region)
  - Advanced settings (DNS, hostnames, gateways, subnets)
  - Provider-specific settings

- `WireGuardConfig` - Complete WireGuard configuration
  - Creation settings (create, provider, region, size)
  - Server settings (endpoint, keys, port)
  - Network settings (subnet, allowed IPs, MTU)
  - Mesh networking support

- `DOVPCConfig` - DigitalOcean VPC specifics
- `LinodeVPCConfig` - Linode VPC with subnets
- `LinodeSubnetConfig` - Linode subnet configuration

#### Documentation

**VPC_VPN_CLUSTER.md** - Complete guide (70+ pages)
- Architecture overview
- Configuration examples (minimal to advanced)
- Multi-cloud setups
- Security considerations
- Troubleshooting
- Cost estimates

**examples/cluster-with-vpc-vpn.yaml** - Full example
- Complete configuration with VPC + VPN + Cluster
- Multi-cloud setup (DigitalOcean + Linode)
- HA cluster (3 masters + 3 workers)
- Detailed comments

**examples/cluster-minimal-with-vpn.yaml** - Minimal example
- Simplest possible configuration
- Single provider
- Quick start

#### Workflow

```
1. Configure YAML
   └── Set VPC and VPN create: true

2. Run deploy
   └── kubernetes-create deploy --config cluster.yaml

3. Automated steps:
   ├── ✅ Create VPCs
   ├── ✅ Create WireGuard server
   ├── ✅ Configure VPN
   ├── ✅ Create cluster nodes
   ├── ✅ Setup VPN clients
   └── ✅ Install Kubernetes

4. Ready!
   └── kubectl get nodes
```

#### Benefits

- ✅ **Zero Manual Work** - Everything automated
- ✅ **Single Configuration** - One YAML for everything
- ✅ **Multi-Cloud** - Connect multiple providers seamlessly
- ✅ **Secure** - Encrypted VPN between all nodes
- ✅ **Private** - Nodes in isolated VPCs
- ✅ **Cross-Provider** - Communicate across clouds

### Added - GitOps Addons Management

#### New Commands

- `kubernetes-create addons bootstrap` - Bootstrap ArgoCD from a GitOps repository
  - Clones user's Git repository
  - Installs ArgoCD from repo manifests
  - Configures ArgoCD to watch the repo
  - Enables auto-sync for all addons
  - Supports both public and private repositories (SSH/HTTPS)
  - Configurable branch and path

- `kubernetes-create addons list` - List all installed addons
  - Shows addon name, category, status, version, and namespace
  - Summary statistics

- `kubernetes-create addons status` - Show ArgoCD and addon status
  - ArgoCD server status
  - All Applications sync and health status
  - Detailed view of GitOps state

- `kubernetes-create addons sync` - Manually trigger ArgoCD sync
  - Force immediate synchronization
  - Can sync all apps or specific app

- `kubernetes-create addons template` - Generate example GitOps repo structure
  - Print template structure to stdout
  - Generate actual directory structure with manifests
  - Includes README and example addons

#### New Package: pkg/addons

**pkg/addons/types.go**
- `Addon` struct - Complete addon definition
- `AddonStatus` struct - Runtime status information
- `Category` type - Addon categories (ingress, storage, monitoring, etc.)

**pkg/addons/catalog.go**
- Pre-defined catalog of 12 popular addons:
  - ingress-nginx - NGINX Ingress Controller
  - cert-manager - Certificate Management
  - prometheus - Monitoring Stack
  - longhorn - Distributed Storage
  - argocd - GitOps CD
  - loki - Log Aggregation
  - metallb - LoadBalancer for bare metal
  - postgres-operator - PostgreSQL Management
  - istio - Service Mesh
  - external-dns - DNS Management
  - velero - Backup/Restore
  - sealed-secrets - Encrypted Secrets

**pkg/addons/gitops.go**
- `GitOpsConfig` struct - GitOps repository configuration
- `AddonBootstrap` struct - Addon bootstrap configuration
- `BootstrapArgoCD()` - Main bootstrap function
  - Creates argocd namespace
  - Applies ArgoCD installation manifests
  - Waits for ArgoCD to be ready
  - Creates ArgoCD Application pointing to GitOps repo
- `generateArgoCDApp()` - Generates ArgoCD Application manifest
  - Configured for auto-sync
  - Self-heal enabled
  - Auto-create namespaces
- `CloneGitOpsRepo()` - Clone Git repository
  - Supports SSH with private key
  - Supports HTTPS
  - Branch checkout
- `ApplyAddonsFromRepo()` - Apply addon manifests via kubectl
- `GenerateGitOpsRepoStructure()` - Example repo structure
- `GetBootstrapAddons()` - Available bootstrap addons with dependencies

#### Documentation

**ADDONS_GITOPS.md** - Complete GitOps addons guide (50+ pages)
- Overview of GitOps workflow
- Quick start guide
- Detailed command reference with examples
- Repository structure recommendations
- Multi-environment setup
- Private repository configuration
- Monitoring and troubleshooting
- Best practices
- Popular addon catalog

#### Features

- **GitOps-First Approach**: User's Git repository is the single source of truth
- **Declarative Management**: Describe desired state in Git, ArgoCD maintains it
- **Auto-Sync**: Automatic synchronization of changes from Git
- **Self-Heal**: ArgoCD automatically corrects drift
- **Private Repository Support**: SSH keys and HTTPS tokens
- **Multi-Environment**: Support for different branches/paths per environment
- **Audit Trail**: Full Git history of all addon changes
- **Easy Rollback**: `git revert` to undo changes
- **Template Generation**: Quick start with example structures

### Changed

#### cmd/addons.go
- New file implementing all addons subcommands
- Integration with Pulumi stack for kubeconfig
- Spinner-based progress indicators
- Confirmation prompts (bypassable with --yes)
- Detailed output with next steps
- Error handling and user guidance

### Technical Details

#### GitOps Workflow

```
User Repo → kubernetes-create bootstrap → ArgoCD Installation → ArgoCD Watches Repo → Auto-Sync Addons
```

1. User creates Git repo with addon manifests in `addons/` directory
2. User runs `kubernetes-create addons bootstrap --repo <url>`
3. CLI clones repo and applies ArgoCD from `addons/argocd/`
4. ArgoCD Application is created pointing to the repo
5. ArgoCD continuously watches repo and syncs changes
6. User adds/updates addons by committing to Git
7. ArgoCD automatically applies changes to cluster

#### Benefits Over Traditional Approach

- ✅ **Version Control**: All changes tracked in Git
- ✅ **Declarative**: No imperative `helm install` commands
- ✅ **Automated**: No manual sync needed
- ✅ **Auditable**: Git log shows who changed what and when
- ✅ **Rollback**: Simple `git revert` to undo
- ✅ **Multi-Cluster**: Same repo can manage multiple clusters
- ✅ **Self-Healing**: Cluster auto-corrects to match Git state

#### Example Usage

```bash
# 1. Generate template repository
kubernetes-create addons template --output my-gitops-repo

# 2. Customize and push to Git
cd my-gitops-repo
# ... edit manifests ...
git init && git add . && git commit -m "Initial"
git remote add origin git@github.com:me/k8s-gitops.git
git push -u origin main

# 3. Bootstrap ArgoCD
kubernetes-create addons bootstrap --repo git@github.com:me/k8s-gitops.git

# 4. Add new addon by committing to Git
mkdir addons/redis
# ... create manifests ...
git add addons/redis && git commit -m "Add Redis"
git push
# ArgoCD automatically syncs! ✅

# 5. Monitor status
kubernetes-create addons status
```

### Dependencies

No new external dependencies added. Uses existing:
- kubectl (for applying manifests)
- git (for cloning repositories)
- Pulumi Automation API (for stack management)

---

## Previous Changes

### [v1.0.0] - Initial Release

#### Core Features
- Multi-cloud support (DigitalOcean, Linode)
- RKE2 Kubernetes cluster deployment
- WireGuard VPN mesh networking
- Pulumi Automation API (no Pulumi CLI needed)
- Kubernetes-style YAML configuration
- Environment variable expansion in configs
- Node management (list, add, remove, ssh, upgrade)
- State management via Pulumi
- Dry-run/preview functionality
- Multi-stack support (production, staging, etc.)

#### Commands
- `kubernetes-create deploy` - Deploy/update cluster
- `kubernetes-create destroy` - Destroy cluster
- `kubernetes-create status` - Show cluster status
- `kubernetes-create config generate` - Generate config templates
- `kubernetes-create nodes list` - List cluster nodes
- `kubernetes-create nodes add` - Add nodes
- `kubernetes-create nodes remove` - Remove nodes
- `kubernetes-create nodes ssh` - SSH into nodes
- `kubernetes-create nodes upgrade` - Upgrade Kubernetes

#### Configuration
- RKE2Config with 40+ options
- Support for both flags and YAML
- Auto-detect config format
- Comprehensive examples

#### Documentation
- README.md - Project overview
- STATE_MANAGEMENT.md - Pulumi state guide
- DRY_RUN_USAGE.md - Preview functionality guide
- NODES_MANAGEMENT.md - Complete node management guide
- IMPROVEMENTS_SUMMARY.md - All improvements summary
- examples/ - Example configurations

---

## Migration Guide

### From Manual Addon Management to GitOps

If you were previously installing addons manually:

**Before:**
```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx
```

**After:**
```bash
# 1. Create GitOps repo with manifests
kubernetes-create addons template -o gitops-repo

# 2. Bootstrap
kubernetes-create addons bootstrap --repo <url>

# 3. Add addons by committing to Git
# ArgoCD handles the rest!
```

### Benefits

- 📦 **No more Helm commands** - Just Git commits
- 🔄 **Auto-sync** - Changes apply automatically
- 📊 **Visibility** - See all addons in one place
- 🔙 **Easy rollback** - Git revert
- 📝 **Audit trail** - Git history

---

## Roadmap

### Planned Features

- [ ] **Real-time sync monitoring** - Watch ArgoCD sync in real-time
- [ ] **Addon marketplace** - Browse and install popular addons
- [ ] **Health checks** - Validate addon health
- [ ] **Dependency resolution** - Auto-install addon dependencies
- [ ] **Multi-cluster support** - Manage addons across clusters
- [ ] **Secrets management** - Sealed Secrets integration
- [ ] **Backup/Restore** - Velero integration
- [ ] **Cost estimation** - Estimate addon resource costs
- [ ] **Compliance checks** - Policy enforcement
- [ ] **Addon recommendations** - Suggest addons based on workload

---

## Breaking Changes

None. This release adds new functionality without breaking existing features.

---

## Contributors

- Initial implementation of GitOps addon management
- Documentation and examples
- Testing and validation

---

## Notes

This release introduces a **GitOps-first approach** for addon management, following industry best practices and providing a more declarative, auditable, and automated way to manage cluster addons.

The traditional Helm-based addon catalog (pkg/addons/catalog.go) is still available but the GitOps approach is now the recommended way to manage addons.
