# 🦥 Getting Started

Welcome to Sloth Kubernetes! Let's get you up and running. Slowly, but surely!

---

## Your Journey Begins 🦥

Whether you're new to Kubernetes or a seasoned pro, Sloth Kubernetes makes multi-cloud cluster deployment simple. Just follow these steps at your own pace!

!!! tip "The Sloth Philosophy 🦥"
    Why rush? Good clusters are deployed slowly and carefully. We'll get there together!

---

## Quick Navigation

<div class="grid cards" markdown>

-   📦 **Installation**

    ---

    Download and install Sloth Kubernetes

    **Time:** 2 minutes ⏱️

    [:octicons-arrow-right-24: Install Now](installation.md)

-   🚀 **Quick Start**

    ---

    Deploy your first cluster

    **Time:** 5 minutes ⏱️

    [:octicons-arrow-right-24: Quick Start](quickstart.md)

-   🎯 **First Cluster**

    ---

    Detailed walkthrough with explanations

    **Time:** 15 minutes ⏱️

    [:octicons-arrow-right-24: First Cluster](first-cluster.md)

-   🔮 **What's Next?**

    ---

    Explore advanced features

    **Time:** Whenever you're ready! 🦥

    [:octicons-arrow-right-24: Next Steps](whats-next.md)

</div>

---

## Learning Path

Follow this path to become a Sloth Kubernetes expert! 🦥

```mermaid
graph LR
    A[📦 Installation] --> B[🚀 Quick Start]
    B --> C[🎯 First Cluster]
    C --> D[⚙️ Configuration]
    D --> E[🎓 Advanced Topics]
    E --> F[🦥 Sloth Master!]

    style A fill:#8B4513,stroke:#D2691E,color:#fff
    style B fill:#8B4513,stroke:#D2691E,color:#fff
    style C fill:#8B4513,stroke:#D2691E,color:#fff
    style D fill:#8B4513,stroke:#D2691E,color:#fff
    style E fill:#8B4513,stroke:#D2691E,color:#fff
    style F fill:#228B22,stroke:#32CD32,color:#fff
```

---

## Prerequisites

Before you start, make sure you have:

### Required ✅

- **Cloud Provider Account** - DigitalOcean and/or Linode
- **API Tokens** - Read/Write access from your provider
- **Basic Linux Knowledge** - Understanding of SSH and command line

### Optional (but helpful) 🦥

- **kubectl** - For managing your cluster after deployment
- **SSH Keys** - For accessing nodes directly
- **Git** - For GitOps workflows

!!! success "Zero Installation Dependencies! 🦥"
    Unlike other tools, Sloth Kubernetes doesn't require Pulumi CLI, Terraform, or any other external tools. Just one binary!

---

## What You'll Learn

By the end of this section, you'll be able to:

- [x] Install Sloth Kubernetes on your system
- [x] Configure cloud provider credentials
- [x] Deploy a multi-cloud Kubernetes cluster
- [x] Access and manage your cluster
- [x] Scale nodes up and down
- [x] Configure WireGuard VPN mesh
- [x] Bootstrap GitOps with ArgoCD

---

## Getting Help

Need assistance? We've got you! 🦥

<div class="grid cards" markdown>

-   💬 **Community Slack**

    ---

    Chat with other sloths!

    [Join Slack](https://sloth-kubernetes.slack.com)

-   📖 **Documentation**

    ---

    Comprehensive guides

    [Browse Docs](../user-guide/index.md)

-   🐛 **GitHub Issues**

    ---

    Report bugs or request features

    [Open Issue](https://github.com/yourusername/sloth-kubernetes/issues)

-   📧 **Email Support**

    ---

    Get help from the team

    [support@sloth-kubernetes.io](mailto:support@sloth-kubernetes.io)

</div>

---

## Typical Timeline

Here's how long each step typically takes:

| Step | Time | Details |
|------|------|---------|
| **Installation** | 2 min | Download binary, configure tokens |
| **Quick Start** | 5 min | Deploy simple cluster |
| **First Cluster** | 15 min | Detailed walkthrough |
| **Advanced Config** | 30 min | Custom configurations |
| **Production Deploy** | 45 min | Full HA production cluster |

!!! tip "Sloth Speed 🦥"
    These times are estimates. Take your time! The sloth way is to do things slowly and correctly.

---

## Architecture Overview

Before diving in, here's what Sloth Kubernetes will build for you:

```
🦥 Multi-Cloud Kubernetes Cluster 🦥

┌──────────────────────────────────────────────────────────┐
│                    Your Kubernetes Cluster                │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  ┌────────────────────┐      ┌────────────────────┐     │
│  │  DigitalOcean      │      │     Linode         │     │
│  │  ┌──────────────┐  │      │  ┌──────────────┐  │     │
│  │  │ Master Node  │  │      │  │ Master Node  │  │     │
│  │  │   (etcd)  🦥 │  │      │  │   (etcd)  🦥 │  │     │
│  │  └──────────────┘  │      │  └──────────────┘  │     │
│  │  ┌──────────────┐  │      │  ┌──────────────┐  │     │
│  │  │ Worker Node  │  │      │  │ Worker Node  │  │     │
│  │  │   (apps)  🦥 │  │      │  │   (apps)  🦥 │  │     │
│  │  └──────────────┘  │      │  └──────────────┘  │     │
│  └────────────────────┘      └────────────────────┘     │
│            │                           │                 │
│            └──────► WireGuard VPN ◄────┘                 │
│                    (10.8.0.0/24) 🔐                      │
└──────────────────────────────────────────────────────────┘
```

**Key Features:**

- 🌍 **Multi-Cloud** - Nodes across multiple providers
- 🔐 **Encrypted** - WireGuard VPN mesh
- 🎯 **High Availability** - Multiple masters with etcd
- 🌳 **GitOps Ready** - ArgoCD bootstrap support
- 🦥 **Simple** - One binary, one config file

---

## Ready to Start? 🦥

Let's begin your sloth journey! Choose your path:

!!! success "Recommended Path"
    Start with [Installation](installation.md) → [Quick Start](quickstart.md) → [First Cluster](first-cluster.md)

<div align="center">

[Install Now](installation.md){ .md-button .md-button--primary .md-button--lg }
[Quick Start](quickstart.md){ .md-button .md-button--lg }

</div>

---

!!! quote "Sloth Proverb 🦥"
    *"Every expert was once a beginner. Take your time, learn slowly, succeed surely!"*
