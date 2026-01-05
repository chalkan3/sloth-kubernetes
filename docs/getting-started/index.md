---
title: Getting Started
description: Get started with sloth-kubernetes by installing the binary and deploying your first cluster
---

# Getting Started

Welcome to **sloth-kubernetes**! This guide will walk you through installing the tool and deploying your first multi-cloud Kubernetes cluster.

## What You'll Learn

In this section, you'll learn how to:

- Install sloth-kubernetes on Linux, macOS, or Windows
- Configure cloud provider credentials
- Create your first cluster configuration
- Deploy a production-grade Kubernetes cluster
- Access and manage your cluster

## Prerequisites

Before you begin, make sure you have:

### Required

- **Cloud provider account** - At least one of:
    - [DigitalOcean account](https://www.digitalocean.com/) with API token
    - [Linode account](https://www.linode.com/) with API token
    - [AWS account](https://aws.amazon.com/) with access keys (coming soon)
    - [Azure account](https://azure.microsoft.com/) with service principal (coming soon)
    - [GCP account](https://cloud.google.com/) with service account (coming soon)

- **SSH keys** - For secure access to cluster nodes
- **Terminal/Shell** - Command-line access on your local machine

### Optional

- **kubectl** - For Kubernetes management (embedded in sloth-kubernetes)
- **Helm** - For package management (wrapper provided)
- **Git** - For GitOps with ArgoCD

## Quick Installation

=== "Linux"

    ```bash
    # Download the latest release
    curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes

    # Make it executable
    chmod +x sloth-kubernetes

    # Move to PATH
    sudo mv sloth-kubernetes /usr/local/bin/

    # Verify installation
    sloth-kubernetes version
    ```

=== "macOS"

    ```bash
    # Download the latest release
    curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-amd64 -o sloth-kubernetes

    # Make it executable
    chmod +x sloth-kubernetes

    # Move to PATH
    sudo mv sloth-kubernetes /usr/local/bin/

    # Verify installation
    sloth-kubernetes version
    ```

=== "macOS (ARM64)"

    ```bash
    # Download the latest release for Apple Silicon
    curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-arm64 -o sloth-kubernetes

    # Make it executable
    chmod +x sloth-kubernetes

    # Move to PATH
    sudo mv sloth-kubernetes /usr/local/bin/

    # Verify installation
    sloth-kubernetes version
    ```

=== "Windows"

    ```powershell
    # Download the latest release
    Invoke-WebRequest -Uri "https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-windows-amd64.exe" -OutFile "sloth-kubernetes.exe"

    # Move to a directory in PATH
    Move-Item sloth-kubernetes.exe C:\Windows\System32\

    # Verify installation
    sloth-kubernetes version
    ```

## Next Steps

Once installed, continue with:

1. **[Installation Guide](installation.md)** - Detailed installation instructions and troubleshooting
2. **[Quick Start](quickstart.md)** - Deploy your first cluster in 5 minutes

## What's Next?

After deploying your first cluster, explore:

- **[User Guide](../user-guide/index.md)** - Learn all CLI commands and workflows
- **[Configuration Reference](../configuration/lisp-format.md)** - Complete LISP configuration options
- **[FAQ](../faq.md)** - Frequently asked questions
