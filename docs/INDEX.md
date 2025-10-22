# 🦥 Welcome to Sloth Kubernetes

<div align="center">

**Deploy Kubernetes clusters across multiple clouds**
***Slowly, but surely*** 🐌

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![Test Coverage](https://img.shields.io/badge/coverage-90%25-brightgreen?style=for-the-badge)](contributing/testing.md)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE.md)

[Get Started](getting-started/quickstart.md){ .md-button .md-button--primary }
[View on GitHub](https://github.com/yourusername/sloth-kubernetes){ .md-button }

</div>

---

## 🌟 What is Sloth Kubernetes?

**Sloth Kubernetes** is a single-binary CLI tool that deploys production-ready Kubernetes clusters across **multiple cloud providers** with **zero external dependencies**. No Pulumi CLI, no Terraform, no kubectl required for deployment—just one lazy sloth doing all the heavy lifting! 🦥💪

!!! tip "The Sloth Philosophy"
    Why rush? Good things take time. We'll get there... eventually. 🦥

<div class="grid cards" markdown>

-   🚀 **Zero Dependencies**

    ---

    Everything you need in one binary. No Pulumi CLI, no complex setup. Just download and deploy! 🦥

    [:octicons-arrow-right-24: Installation](getting-started/installation.md)

-   🌍 **True Multi-Cloud**

    ---

    Deploy across DigitalOcean and Linode. Your cluster, your choice, multiple clouds! 🦥

    [:octicons-arrow-right-24: Multi-Cloud Guide](advanced/multi-cloud.md)

-   🔐 **Secure by Default**

    ---

    WireGuard VPN mesh, encrypted secrets, CIS benchmarks. Security while you sleep! 😴🦥

    [:octicons-arrow-right-24: Security](configuration/security.md)

-   🌳 **GitOps Native**

    ---

    ArgoCD auto-bootstrap, Git as source of truth. Set it and forget it! 🦥

    [:octicons-arrow-right-24: GitOps](advanced/gitops.md)

</div>

---

## ⚡ Quick Start

!!! success "3 Minutes to Production Cluster"
    Have a production-ready cluster in the time it takes to make coffee! ☕🦥

```bash
# Step 1: Download (pick your platform)
curl -L https://github.com/user/sloth-kubernetes/releases/latest/download/sloth-kubernetes -o sloth-kubernetes
chmod +x sloth-kubernetes

# Step 2: Create config
cat > cluster.yaml <<EOF
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: my-cluster
spec:
  providers:
    digitalocean:
      enabled: true
      token: \${DIGITALOCEAN_TOKEN}
  kubernetes:
    version: v1.28.5+rke2r1
  nodePools:
    - name: masters
      count: 3
      roles: [master]
EOF

# Step 3: Deploy! 🦥
export DIGITALOCEAN_TOKEN="your-token"
./sloth-kubernetes deploy --config cluster.yaml

# Step 4: Access your cluster
./sloth-kubernetes kubeconfig > ~/.kube/config
kubectl get nodes
```

[Full Quick Start Guide →](getting-started/quickstart.md){ .md-button }

---

## 🎯 Why Sloth Kubernetes?

### Traditional vs. Sloth Way

| Aspect | Traditional | Sloth Way 🦥 |
|--------|------------|--------------|
| Installation | Multiple CLIs + tools | Single binary |
| Dependencies | Pulumi + kubectl + more | None |
| Setup Time | 30-60 minutes | 3-5 minutes |
| Multi-Cloud | Complex manual | Built-in |
| VPN Setup | Hours of config | Automatic |
| Learning Curve | Steep ⛰️ | Gentle 🦥 |

!!! quote "Ancient Sloth Proverb"
    *"The best time to deploy was yesterday. The second best time is now... but take your time!"* 🦥

---

## 🚀 Key Features

### 🎯 Single Binary Simplicity

Everything embedded in one binary:

- ✅ Pulumi Automation API (no CLI needed)
- ✅ State management
- ✅ VPN configuration
- ✅ GitOps bootstrap
- ✅ Kubeconfig generation

### 🌍 Multi-Cloud Support

```
        🦥 Your Kubernetes Cluster 🦥

┌────────────────────┐  ┌────────────────────┐
│   DigitalOcean     │  │      Linode        │
│   Region: NYC3     │  │  Region: US-East   │
│                    │  │                    │
│ • Master 1    🦥   │  │ • Master 2    🦥   │
│ • Worker 1    🦥   │  │ • Master 3    🦥   │
│ • Worker 2    🦥   │  │ • Worker 3    🦥   │
│                    │  │                    │
│ VPC: 10.10.0.0/16  │  │ VPC: 10.11.0.0/16  │
└─────────┬──────────┘  └──────────┬─────────┘
          │                        │
          └────────► VPN ◄─────────┘
             WireGuard 🔐
            10.8.0.0/24
```

### 🔐 Automated Security

- WireGuard VPN mesh (automatic setup)
- Encrypted secrets at rest
- CIS benchmark compliance
- Pod security policies
- Network policies
- Bastion host support

### 🌳 GitOps Ready

```bash
# Deploy cluster
sloth-kubernetes deploy --config cluster.yaml

# Bootstrap ArgoCD
sloth-kubernetes addons bootstrap \
  --repo https://github.com/yourorg/k8s-gitops

# Everything auto-syncs from Git! 🦥🌳
```

---

## 📚 Documentation

<div class="grid cards" markdown>

-   📖 **Getting Started**

    ---

    New to Sloth? Start here! 🦥

    - [Installation](getting-started/installation.md)
    - [Quick Start](getting-started/quickstart.md)
    - [First Cluster](getting-started/first-cluster.md)

-   💻 **User Guide**

    ---

    Day-to-day operations 🦥

    - [CLI Reference](user-guide/cli-reference.md)
    - [Deploy](user-guide/deploy.md)
    - [Manage Nodes](user-guide/nodes.md)
    - [VPN Management](user-guide/vpn.md)

-   ⚙️ **Configuration**

    ---

    Configure your cluster 🦥

    - [File Structure](configuration/file-structure.md)
    - [Providers](configuration/providers.md)
    - [Network](configuration/network.md)
    - [Examples](configuration/examples.md)

-   🎓 **Advanced**

    ---

    Pro tips and tricks 🦥

    - [Architecture](advanced/architecture.md)
    - [Multi-Cloud](advanced/multi-cloud.md)
    - [State Management](advanced/state-management.md)
    - [GitOps](advanced/gitops.md)

</div>

---

## 🦥 Real-World Example

Production-grade multi-cloud HA cluster:

```yaml
apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: production-ha
  labels:
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
      create: true  # 🦥 Auto-create VPN!
      meshNetworking: true

  kubernetes:
    distribution: rke2
    version: v1.28.5+rke2r1
    rke2:
      secretsEncryption: true
      snapshotScheduleCron: "0 */12 * * *"
      profiles:
        - cis-1.6

  nodePools:
    do-masters:
      provider: digitalocean
      count: 1
      roles: [master]
      size: s-2vcpu-4gb

    linode-masters:
      provider: linode
      count: 2  # Odd number for quorum
      roles: [master]
      size: g6-standard-2

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

  addons:
    gitops:
      enabled: true
      repository: https://github.com/yourorg/k8s-gitops
```

!!! success "Deploy in One Command"
    ```bash
    sloth-kubernetes deploy --config production-ha.yaml
    ```

    The sloth will: 🦥

    - Create VPCs on both clouds
    - Deploy WireGuard VPN server
    - Provision 7 nodes (3 masters, 4 workers)
    - Install RKE2 Kubernetes
    - Configure encrypted mesh
    - Bootstrap ArgoCD
    - Set up automated backups
    - Apply security policies

    **Time: 8-10 minutes** ☕

---

## 🎓 What Makes Us Different?

✅ **No External Dependencies** - One binary, that's it
✅ **Multi-Cloud Native** - Not bolted on, built in
✅ **Auto VPN** - WireGuard mesh configured automatically
✅ **GitOps First** - Bootstrap in one command
✅ **Production Ready** - HA, backups, security hardening
✅ **Simple** - Kubernetes-style YAML config
✅ **Fast** - Deploy in 5-10 minutes
✅ **Sloth Approved** - Slow is smooth, smooth is fast 🦥

---

## 🌟 Community

Join the sloth family! 🦥

- :fontawesome-brands-github: [GitHub](https://github.com/yourusername/sloth-kubernetes) - Star us!
- :fontawesome-brands-slack: [Slack](https://sloth-kubernetes.slack.com) - Chat with us!
- :fontawesome-brands-twitter: [Twitter](https://twitter.com/slothkubernetes) - Follow for updates!
- :material-email: [Email](mailto:support@sloth-kubernetes.io) - Get help!

---

## 🦥 Ready?

Take it slow, take it steady, take it with a sloth!

<div align="center">

[Install Now](getting-started/installation.md){ .md-button .md-button--primary .md-button--lg }
[Quick Start](getting-started/quickstart.md){ .md-button .md-button--lg }
[View Examples](configuration/examples.md){ .md-button .md-button--lg }

---

**Made with 🦥 and ❤️ by the Sloth Kubernetes community**

</div>
