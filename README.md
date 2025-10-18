# Kube RKE Multi-Cloud

Production-grade Kubernetes cluster deployment across multiple cloud providers using RKE2, WireGuard VPN mesh, and Pulumi Infrastructure as Code.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Pulumi](https://img.shields.io/badge/Pulumi-v3-5430ED?logo=pulumi)](https://www.pulumi.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.33.5-326CE5?logo=kubernetes)](https://kubernetes.io/)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)

## Overview

This project automates the deployment of a highly available Kubernetes cluster spanning multiple cloud providers (DigitalOcean and Linode) with a secure WireGuard VPN mesh network. Built with Pulumi and Go, it creates a production-ready cluster with:

- **High Availability**: 3 master nodes with etcd cluster
- **Multi-Cloud**: Nodes distributed across DigitalOcean and Linode
- **Private Networking**: Full WireGuard VPN mesh between all nodes
- **Security-First**: VPN-only access, no public exposure
- **Fully Automated**: Complete infrastructure as code
- **Production Ready**: ArgoCD, Nginx Ingress, Calico CNI

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Control Plane (HA)                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  master-1   │  │  master-2   │  │  master-3   │            │
│  │ 10.8.0.10   │  │ 10.8.0.11   │  │ 10.8.0.12   │            │
│  │ DigitalOcean│  │   Linode    │  │   Linode    │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ WireGuard VPN Mesh
                              │
┌─────────────────────────────────────────────────────────────────┐
│                        Worker Nodes                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  worker-1   │  │  worker-2   │  │  worker-3   │            │
│  │ 10.8.0.13   │  │ 10.8.0.14   │  │ 10.8.0.15   │            │
│  │DigitalOcean │  │DigitalOcean │  │   Linode    │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Infrastructure
- **Multi-Cloud Support**: Seamlessly deploy across DigitalOcean and Linode
- **Automated Provisioning**: Fully automated VM creation and configuration
- **Smart Retries**: Intelligent retry logic for apt-get operations to handle unattended-upgrades
- **Dependency Validation**: Automatic validation of all prerequisites before proceeding

### Networking
- **WireGuard VPN Mesh**: Full mesh VPN with direct tunnels between all nodes
- **Private Cluster**: All inter-node communication over encrypted WireGuard tunnels
- **Automated DNS**: Automatic DNS record creation for cluster endpoints
- **Calico CNI**: Production-grade pod networking

### Kubernetes
- **RKE2 Distribution**: Rancher's next-generation Kubernetes distribution
- **High Availability**: 3 master nodes with etcd cluster
- **Node Taints**: Worker nodes pre-configured with workload-specific taints
  - `worker-1`, `worker-2`: `workload=tools` (for CI/CD, monitoring)
  - `worker-3`: `workload=misc` (for experiments, testing)

### Operations
- **GitOps Ready**: Pre-installed ArgoCD for GitOps workflows
- **Ingress Controller**: Nginx Ingress with SSL passthrough
- **Secure Access**: VPN-only access, no public exposure
- **Complete Documentation**: Extensive docs with examples and diagrams

## Prerequisites

- **Pulumi**: v3.x or higher
- **Go**: 1.21 or higher
- **Cloud Accounts**:
  - DigitalOcean account with API token
  - Linode account with API token
- **WireGuard VPN Server**: Pre-configured WireGuard server
- **Domain**: DNS domain for cluster endpoints (e.g., via DigitalOcean DNS)

## Quick Start

### 1. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/kube-rke-multi-cloud.git
cd kube-rke-multi-cloud

# Install dependencies
go mod download

# Configure Pulumi stack
pulumi stack init production

# Set required configuration
pulumi config set digitaloceanToken <YOUR_DO_TOKEN> --secret
pulumi config set linodeToken <YOUR_LINODE_TOKEN> --secret
pulumi config set wireguardServerEndpoint <YOUR_VPN_SERVER:51820>
pulumi config set wireguardServerPublicKey <YOUR_VPN_PUBLIC_KEY>
```

### 2. Deploy the Cluster

```bash
# Preview the deployment
pulumi preview

# Deploy
pulumi up
```

The deployment will:
1. Generate SSH keys for cluster access
2. Create 6 VMs (3 masters + 3 workers) across clouds
3. Install and configure Docker, WireGuard, and prerequisites
4. Set up WireGuard mesh VPN between all nodes
5. Install RKE2 Kubernetes cluster
6. Create DNS records for cluster endpoints
7. Output kubeconfig and connection details

### 3. Access the Cluster

```bash
# Get kubeconfig
pulumi stack output kubeConfig --show-secrets > ~/.kube/config

# Verify cluster
kubectl get nodes

# Should show 6 nodes all Ready
```

## Configuration

### Pulumi Configuration

| Key | Description | Required | Secret |
|-----|-------------|----------|--------|
| `digitaloceanToken` | DigitalOcean API token | Yes | Yes |
| `linodeToken` | Linode API token | Yes | Yes |
| `wireguardServerEndpoint` | WireGuard VPN server endpoint | Yes | No |
| `wireguardServerPublicKey` | WireGuard VPN server public key | Yes | No |
| `rke2ClusterToken` | RKE2 cluster join token (auto-generated if not set) | No | Yes |

### Node Pools

The default configuration creates:

| Pool | Provider | Count | Size | Role |
|------|----------|-------|------|------|
| do-masters | DigitalOcean | 1 | s-2vcpu-4gb | Master |
| do-workers | DigitalOcean | 2 | s-2vcpu-4gb | Worker (tools) |
| linode-masters | Linode | 2 | g6-standard-2 | Master |
| linode-workers | Linode | 1 | g6-standard-2 | Worker (misc) |

You can customize these in `main.go` or implement configuration file support.

## Project Structure

```
.
├── main.go                          # Main Pulumi program
├── go.mod                           # Go dependencies
├── internal/
│   └── orchestrator/               # Orchestration logic
│       ├── cluster_orchestrator.go  # Main orchestrator
│       ├── node_deployment.go       # Node provisioning
│       ├── node_provisioning.go     # Dependency installation
│       ├── rke2_installer.go        # RKE2 deployment
│       ├── wireguard_mesh.go        # WireGuard mesh setup
│       ├── dns_manager.go           # DNS record management
│       └── ...
├── pkg/
│   ├── config/                     # Configuration structs
│   ├── providers/                  # Cloud provider implementations
│   ├── security/                   # SSH key generation
│   └── network/                    # Network utilities
└── docs/                           # Complete documentation
    ├── README.md                   # Main documentation
    ├── QUICK_START.md             # Quick reference
    ├── NETWORK_DIAGRAM.md         # Network architecture
    └── examples/                   # Deployment examples
```

## Usage Examples

### Deploy Application to Tools Workers

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 2
  template:
    spec:
      nodeSelector:
        workload: tools
      tolerations:
      - key: workload
        operator: Equal
        value: tools
        effect: NoSchedule
      containers:
      - name: app
        image: my-app:latest
```

### Expose Service via Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: myapp.kube.yourdomain.com
    http:
      paths:
      - path: /
        backend:
          service:
            name: my-app
            port:
              number: 80
```

See `docs/examples/` for more complete examples.

## Documentation

Complete documentation is available in the `docs/` directory:

- **[docs/README.md](docs/README.md)** - Complete cluster documentation
- **[docs/QUICK_START.md](docs/QUICK_START.md)** - Quick start guide
- **[docs/NETWORK_DIAGRAM.md](docs/NETWORK_DIAGRAM.md)** - Network architecture diagrams
- **[docs/examples/](docs/examples/)** - Deployment examples

## Features Roadmap

### Code Improvements Needed
- [ ] Remove duplicate code (files with `_real`, `_granular` suffixes)
- [ ] Rename components to more descriptive names
- [ ] Extract configuration to YAML files
- [ ] Add comprehensive error handling
- [ ] Implement configuration validation schema
- [ ] Add unit tests for core components

### Feature Enhancements
- [ ] Support for additional cloud providers (AWS, GCP, Azure)
- [ ] Automated backup and restore for etcd
- [ ] Cluster autoscaling support
- [ ] Monitoring stack (Prometheus + Grafana)
- [ ] Logging stack (ELK or Loki)
- [ ] Cert-manager for automatic TLS
- [ ] External-DNS for automated DNS management
- [ ] StorageClass configurations for persistent volumes

## Security Considerations

⚠️ **Important Security Notes**:

1. **Sensitive Files**: The following files are gitignored and contain sensitive data:
   - `Pulumi.yaml` and `Pulumi.*.yaml` (contain configuration)
   - `.pulumi/` (Pulumi state)
   - `*.pem`, `*.key` (SSH keys)
   - kubeconfig files

2. **Secrets Management**: All secrets should be stored in Pulumi config with `--secret` flag:
   ```bash
   pulumi config set mySecret "value" --secret
   ```

3. **VPN Access**: The cluster is accessible ONLY via WireGuard VPN. Ensure your VPN server is properly secured.

4. **SSH Keys**: Generated SSH keys are stored securely and should never be committed to version control.

## Troubleshooting

### Common Issues

**Issue**: apt-get lock errors during provisioning
**Solution**: The code includes intelligent retry logic that handles unattended-upgrades. Wait for retries to complete.

**Issue**: Cannot access cluster via kubectl
**Solution**: Ensure you're connected to the WireGuard VPN and have the correct kubeconfig.

**Issue**: Pods not scheduling on workers
**Solution**: Ensure pods have the correct `nodeSelector` and `tolerations` for tainted nodes.

See `docs/README.md#troubleshooting` for complete troubleshooting guide.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- **Pulumi**: For the excellent Infrastructure as Code platform
- **RKE2**: Rancher's production-grade Kubernetes distribution
- **WireGuard**: For the modern, fast VPN protocol
- **DigitalOcean & Linode**: For reliable cloud infrastructure

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- Check the [documentation](docs/)
- Review [examples](docs/examples/)

---

**Built with ❤️ using Pulumi, Go, and Kubernetes**
