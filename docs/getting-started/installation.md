---
layout: default
title: Installation
parent: Getting Started
nav_order: 1
---

# Installation

sloth-kubernetes is distributed as a single binary with no external dependencies (except for Helm and Kustomize if you want to use those features).

## Download Pre-built Binary

### Linux

```bash
# AMD64
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes

# ARM64
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-arm64 -o sloth-kubernetes

# Make executable and install
chmod +x sloth-kubernetes
sudo mv sloth-kubernetes /usr/local/bin/
```

### macOS

```bash
# AMD64 (Intel)
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-amd64 -o sloth-kubernetes

# ARM64 (Apple Silicon)
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-arm64 -o sloth-kubernetes

# Make executable and install
chmod +x sloth-kubernetes
sudo mv sloth-kubernetes /usr/local/bin/
```

### Windows

Download from [GitHub Releases](https://github.com/chalkan3/sloth-kubernetes/releases/latest):
- `sloth-kubernetes-windows-amd64.exe`
- `sloth-kubernetes-windows-arm64.exe`

Add the binary to your PATH.

## Build from Source

### Prerequisites

- Go 1.23 or later
- Git

### Steps

```bash
# Clone the repository
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes

# Build
go build -o sloth-kubernetes .

# Install (optional)
sudo mv sloth-kubernetes /usr/local/bin/
```

## Verify Installation

```bash
sloth-kubernetes version
```

You should see output similar to:

```
sloth-kubernetes version v1.0.0
```

## Optional Dependencies

### Helm (for helm commands)

```bash
# Linux/macOS
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify
helm version
```

### Kustomize (for kustomize commands)

```bash
# Linux
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
sudo mv kustomize /usr/local/bin/

# macOS
brew install kustomize

# Verify
kustomize version
```

## Shell Completion

### Bash

```bash
sloth-kubernetes completion bash > /etc/bash_completion.d/sloth-kubernetes
```

### Zsh

```bash
sloth-kubernetes completion zsh > "${fpath[1]}/_sloth-kubernetes"
```

### Fish

```bash
sloth-kubernetes completion fish > ~/.config/fish/completions/sloth-kubernetes.fish
```

## Next Steps

- [Quick Start Guide](quickstart) - Deploy your first cluster
- [Configuration](../user-guide/configuration) - Learn about cluster configuration
- [CLI Reference](../cli-reference/commands) - Explore all available commands
