---
title: FAQ
description: Frequently asked questions about sloth-kubernetes
---

# Frequently Asked Questions

## General

### What is sloth-kubernetes?

sloth-kubernetes is a unified tool for deploying production-grade Kubernetes clusters across multiple cloud providers. It embeds infrastructure provisioning (Pulumi), configuration management (SaltStack), and Kubernetes tooling (kubectl, Helm, Kustomize) into a single binary.

### Why use sloth-kubernetes instead of Terraform + Ansible?

**Single Binary**: No need to install and manage multiple tools. Everything is embedded in one executable.

**Simplified Workflow**: Declarative YAML configuration covers infrastructure, networking, security, and Kubernetes setup.

**Built-in Multi-Cloud VPN**: Automatic WireGuard mesh networking between clouds - no manual setup required.

**Integrated Management**: SaltStack for remote operations, kubectl embedded, Helm wrapper - all in one tool.

### Do I need to install Pulumi CLI?

**No**. sloth-kubernetes uses the Pulumi Automation API, which embeds the entire Pulumi engine in the binary. No external Pulumi CLI installation required.

### Which cloud providers are supported?

Currently supported:
- **DigitalOcean** (fully supported)
- **Linode** (fully supported)

Coming soon:
- **AWS** (in development)
- **Azure** (in development)
- **GCP** (in development)

### Is sloth-kubernetes production-ready?

Yes. sloth-kubernetes deploys:
- **RKE2** - CNCF-certified Kubernetes distribution
- **Security hardening** - CIS Benchmark compliance
- **High availability** - Multi-master etcd clusters
- **Automatic backups** - etcd snapshots
- **GitOps** - ArgoCD integration

## Installation

### How do I install sloth-kubernetes?

Download the latest binary from GitHub Releases:

```bash
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes
chmod +x sloth-kubernetes
sudo mv sloth-kubernetes /usr/local/bin/
```

See [Installation Guide](getting-started/installation.md) for details.

### Can I build from source?

Yes:

```bash
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes
go build -o sloth-kubernetes .
```

Requires Go 1.21+.

### Does it work on Windows?

Yes. Windows binaries are available in releases. However, the best experience is on Linux or macOS.

## Configuration

### How do I configure multiple cloud providers?

Enable multiple providers in your `cluster.yaml`:

```yaml
providers:
  digitalocean:
    enabled: true
    token: ${DO_TOKEN}
  
  linode:
    enabled: true
    token: ${LINODE_TOKEN}

nodePools:
  - name: do-masters
    provider: digitalocean
    role: master
    count: 1

  - name: linode-masters
    provider: linode
    role: master
    count: 2
```

### Can I use different instance sizes per node pool?

Yes:

```yaml
nodePools:
  - name: masters
    size: s-2vcpu-4gb
    count: 3

  - name: workers-small
    size: s-2vcpu-4gb
    count: 5

  - name: workers-large
    size: s-8vcpu-16gb
    count: 2
```

### How do I specify SSH keys?

Either:

1. **Let sloth-kubernetes generate them** (automatic)
2. **Provide existing keys**:

```yaml
providers:
  digitalocean:
    sshKeys:
      - "ssh-ed25519 AAAA... user@host"
```

## Deployment

### How long does deployment take?

Typical times:
- **Single node**: ~3 minutes
- **3 masters + 5 workers**: ~5-7 minutes
- **Multi-cloud cluster**: ~8-10 minutes

### Can I deploy to multiple regions?

Yes, specify region per node pool:

```yaml
nodePools:
  - name: nyc-masters
    provider: digitalocean
    region: nyc3
    count: 1

  - name: sfo-masters
    provider: digitalocean
    region: sfo3
    count: 1

  - name: lon-masters
    provider: linode
    region: eu-west
    count: 1
```

### What if deployment fails?

sloth-kubernetes preserves state and allows resume:

```bash
# Check status
sloth-kubernetes status

# Retry deployment
sloth-kubernetes deploy --config cluster.yaml
```

Pulumi handles idempotency - only missing resources are created.

### Can I update a running cluster?

Yes. Modify `cluster.yaml` and re-run:

```bash
sloth-kubernetes deploy --config cluster.yaml
```

Changes are applied incrementally.

## Security

### Is traffic between clouds encrypted?

Yes. WireGuard VPN automatically encrypts all traffic between nodes across clouds.

### Do nodes have public IPs?

By default, **only the bastion host has a public IP**. All cluster nodes use private IPs and are accessed via the bastion.

### How does bastion authentication work?

- **SSH keys** - Automatic key distribution
- **MFA** - Optional Google Authenticator
- **Audit logging** - Complete session recording

### Can I use my own VPN?

Yes, disable WireGuard and configure your own:

```yaml
network:
  wireguard:
    enabled: false
```

## Operations

### How do I access nodes?

Via bastion jump host:

```bash
# List nodes
sloth-kubernetes nodes list

# SSH to node (automatically via bastion)
sloth-kubernetes nodes ssh master-0
```

### How do I run commands on all nodes?

Use SaltStack:

```bash
# Test connectivity
sloth-kubernetes salt ping

# Run command
sloth-kubernetes salt cmd.run "uptime"

# Install package
sloth-kubernetes salt pkg.install htop
```

### How do I scale workers?

```bash
# Add 3 workers
sloth-kubernetes nodes add --pool workers --count 3

# Or update cluster.yaml and redeploy
```

### Can I manage multiple clusters?

Yes, using **stacks**:

```bash
# List stacks
sloth-kubernetes stacks list

# Switch stack
sloth-kubernetes stacks select production

# Each stack is an independent cluster
```

## Kubernetes

### Which Kubernetes version is installed?

RKE2 with the version specified in your config:

```yaml
kubernetes:
  version: "v1.28.2+rke2r1"
  distribution: rke2
```

### Can I use kubectl directly?

Yes:

```bash
# Export kubeconfig
sloth-kubernetes kubeconfig > ~/.kube/config

# Use kubectl normally
kubectl get nodes
```

Or use embedded kubectl:

```bash
sloth-kubernetes kubectl get nodes
```

### Is Helm supported?

Yes, via wrapper:

```bash
sloth-kubernetes helm install nginx bitnami/nginx
```

Or use Helm directly with exported kubeconfig.

### How do I deploy applications?

Multiple ways:

1. **kubectl**:
```bash
sloth-kubernetes kubectl apply -f app.yaml
```

2. **Helm**:
```bash
sloth-kubernetes helm install myapp ./chart
```

3. **GitOps (ArgoCD)**:
```yaml
addons:
  argocd:
    enabled: true
    repository: "https://github.com/org/k8s-apps"
```

## Troubleshooting

### Deployment fails with "insufficient quota"

Your cloud provider account has quota limits. Either:
- Increase quota in provider dashboard
- Use smaller instances
- Reduce node count

### Nodes not joining cluster

Check RKE2 status:

```bash
sloth-kubernetes nodes ssh master-0
sudo systemctl status rke2-server
sudo journalctl -u rke2-server -f
```

Common causes:
- Network connectivity issues
- Insufficient resources
- Firewall blocking ports

### SaltStack commands timeout

```bash
# Check minion connectivity
sloth-kubernetes salt ping

# Check keys
sloth-kubernetes salt keys list

# Accept pending keys
sloth-kubernetes salt keys accept-all
```

### How do I get logs?

```bash
# Cluster logs
sloth-kubernetes status

# Node logs
sloth-kubernetes nodes ssh master-0
sudo journalctl -u rke2-server

# Kubernetes logs
sloth-kubernetes kubectl logs <pod-name>
```

## Cost

### How much does it cost?

Costs depend on cloud provider and instance sizes. Example DigitalOcean cluster:

- 3 masters (s-2vcpu-4gb): $54/month
- 5 workers (s-4vcpu-8gb): $240/month
- Bastion (s-1vcpu-1gb): $6/month
- **Total**: ~$300/month

### Can I use spot/preemptible instances?

Coming soon. Will support:
- AWS Spot Instances
- GCP Preemptible VMs
- Azure Spot VMs

### How do I minimize costs?

- Start with smaller instances
- Use fewer nodes
- Enable cluster autoscaler
- Shut down dev/test clusters when not in use

## Advanced

### Can I customize cloud-init?

Yes, provide custom user data:

```yaml
nodePools:
  - name: workers
    cloudInit: |
      #cloud-config
      packages:
        - docker
      runcmd:
        - systemctl enable docker
```

### How do I backup etcd?

Automatic backups enabled by default:

```yaml
kubernetes:
  rke2:
    server:
      etcdSnapshotScheduleCron: "0 */6 * * *"
      etcdSnapshotRetention: 10
```

Manual backup:

```bash
sloth-kubernetes nodes ssh master-0
sudo rke2 etcd-snapshot save --name manual-backup
```

### Can I use a custom Kubernetes distribution?

Currently only RKE2 is supported. Support for k3s and kubeadm is planned.

### How do I contribute?

- **GitHub Issues**: [Report bugs](https://github.com/chalkan3/sloth-kubernetes/issues)
- **Pull Requests**: Submit improvements
- **Discussions**: Ask questions in [Discussions](https://github.com/chalkan3/sloth-kubernetes/discussions)

See [Contributing Guide](https://github.com/chalkan3/sloth-kubernetes/blob/main/CONTRIBUTING.md).

## Getting Help

### Where can I get support?

- **Documentation**: [https://chalkan3.github.io/sloth-kubernetes](https://chalkan3.github.io/sloth-kubernetes)
- **GitHub Issues**: [Bug reports](https://github.com/chalkan3/sloth-kubernetes/issues)
- **GitHub Discussions**: [Community support](https://github.com/chalkan3/sloth-kubernetes/discussions)

### How do I report a bug?

[Open an issue](https://github.com/chalkan3/sloth-kubernetes/issues/new) with:
- sloth-kubernetes version
- Cloud provider(s)
- Configuration (sanitized)
- Error messages
- Steps to reproduce
