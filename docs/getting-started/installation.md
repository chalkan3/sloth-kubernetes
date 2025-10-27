---
title: Installation
description: Detailed installation instructions for sloth-kubernetes on all platforms
---

# Installation

This guide provides detailed installation instructions for **sloth-kubernetes** across different platforms, package managers, and build methods.

## Binary Releases

The recommended method is to download pre-built binaries from GitHub Releases.

### Linux (amd64)

```bash
# Download
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes

# Make executable
chmod +x sloth-kubernetes

# Move to PATH
sudo mv sloth-kubernetes /usr/local/bin/

# Verify
sloth-kubernetes version
```

### Linux (arm64)

```bash
# Download for ARM64 (e.g., Raspberry Pi 4, AWS Graviton)
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-arm64 -o sloth-kubernetes

# Make executable
chmod +x sloth-kubernetes

# Move to PATH
sudo mv sloth-kubernetes /usr/local/bin/

# Verify
sloth-kubernetes version
```

### macOS (Intel)

```bash
# Download
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-amd64 -o sloth-kubernetes

# Make executable
chmod +x sloth-kubernetes

# Remove quarantine attribute (macOS Gatekeeper)
xattr -d com.apple.quarantine sloth-kubernetes

# Move to PATH
sudo mv sloth-kubernetes /usr/local/bin/

# Verify
sloth-kubernetes version
```

### macOS (Apple Silicon / M1/M2/M3)

```bash
# Download for ARM64
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-arm64 -o sloth-kubernetes

# Make executable
chmod +x sloth-kubernetes

# Remove quarantine attribute
xattr -d com.apple.quarantine sloth-kubernetes

# Move to PATH
sudo mv sloth-kubernetes /usr/local/bin/

# Verify
sloth-kubernetes version
```

### Windows (PowerShell)

```powershell
# Download
Invoke-WebRequest -Uri "https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-windows-amd64.exe" -OutFile "sloth-kubernetes.exe"

# Move to PATH (requires admin)
Move-Item sloth-kubernetes.exe C:\Windows\System32\

# Verify
sloth-kubernetes version
```

## Package Managers

### Homebrew (macOS / Linux)

```bash
# Coming soon
# brew install chalkan3/tap/sloth-kubernetes
```

### Snap (Linux)

```bash
# Coming soon
# sudo snap install sloth-kubernetes --classic
```

### Chocolatey (Windows)

```powershell
# Coming soon
# choco install sloth-kubernetes
```

## Build from Source

### Prerequisites

- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Git** - For cloning repository
- **Make** (optional) - For using Makefile

### Clone and Build

```bash
# Clone repository
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes

# Build binary
go build -o sloth-kubernetes .

# Install to PATH
sudo mv sloth-kubernetes /usr/local/bin/

# Verify
sloth-kubernetes version
```

### Using Make

```bash
# Clone repository
git clone https://github.com/chalkan3/sloth-kubernetes.git
cd sloth-kubernetes

# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install

# Run tests
make test
```

## Container Image

Run sloth-kubernetes in a Docker container:

```bash
# Pull image
docker pull chalkan3/sloth-kubernetes:latest

# Run interactively
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.ssh:/root/.ssh:ro \
  -e DO_TOKEN="${DO_TOKEN}" \
  chalkan3/sloth-kubernetes:latest

# Deploy cluster
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.ssh:/root/.ssh:ro \
  -e DO_TOKEN="${DO_TOKEN}" \
  chalkan3/sloth-kubernetes:latest \
  deploy --config /workspace/cluster.yaml
```

## Shell Completion

sloth-kubernetes supports shell completion for Bash, Zsh, Fish, and PowerShell.

### Bash

```bash
# Generate completion script
sloth-kubernetes completion bash > /etc/bash_completion.d/sloth-kubernetes

# Or for current session only
source <(sloth-kubernetes completion bash)

# Add to .bashrc for persistence
echo 'source <(sloth-kubernetes completion bash)' >> ~/.bashrc
```

### Zsh

```bash
# Generate completion script
sloth-kubernetes completion zsh > "${fpath[1]}/_sloth-kubernetes"

# Or add to .zshrc
echo 'source <(sloth-kubernetes completion zsh)' >> ~/.zshrc

# Reload shell
exec zsh
```

### Fish

```bash
# Generate completion script
sloth-kubernetes completion fish > ~/.config/fish/completions/sloth-kubernetes.fish
```

### PowerShell

```powershell
# Generate completion script
sloth-kubernetes completion powershell | Out-String | Invoke-Expression

# Add to profile for persistence
sloth-kubernetes completion powershell >> $PROFILE
```

## Configuration

### Environment Variables

sloth-kubernetes reads configuration from environment variables:

```bash
# Cloud provider tokens
export DO_TOKEN="your-digitalocean-token"
export LINODE_TOKEN="your-linode-token"
export AWS_ACCESS_KEY_ID="your-aws-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret"
export AZURE_SUBSCRIPTION_ID="your-azure-subscription"
export GCP_PROJECT_ID="your-gcp-project"

# Optional: Custom config directory
export SLOTH_CONFIG_DIR="$HOME/.config/sloth-kubernetes"

# Optional: Enable debug logging
export SLOTH_DEBUG="true"

# Optional: Pulumi backend URL
export PULUMI_BACKEND_URL="file://~/.pulumi"
```

### SSH Keys

sloth-kubernetes requires SSH access to cluster nodes:

```bash
# Generate SSH key if you don't have one
ssh-keygen -t ed25519 -C "sloth-kubernetes" -f ~/.ssh/sloth-k8s

# Add public key to SSH agent
ssh-add ~/.ssh/sloth-k8s

# The public key will be automatically uploaded to cloud providers
```

## Verify Installation

After installation, verify everything works:

```bash
# Check version
sloth-kubernetes version

# Show help
sloth-kubernetes --help

# Validate configuration (if you have a cluster.yaml)
sloth-kubernetes validate --config cluster.yaml

# Test cloud provider authentication
sloth-kubernetes login digitalocean
```

Expected output:

```
sloth-kubernetes version v1.0.0
Go version: go1.21.0
Git commit: abc1234
Built: 2024-01-15T12:00:00Z
```

## Troubleshooting

### macOS: "sloth-kubernetes cannot be opened because the developer cannot be verified"

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/sloth-kubernetes

# Or allow in System Preferences → Security & Privacy
```

### Linux: Permission denied

```bash
# Make binary executable
chmod +x sloth-kubernetes

# Ensure /usr/local/bin is in PATH
echo $PATH | grep /usr/local/bin

# Add to PATH if missing
export PATH="/usr/local/bin:$PATH"
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
```

### Windows: Command not found

```powershell
# Check if binary is in PATH
$env:Path -split ';' | Select-String "System32"

# Add current directory to PATH temporarily
$env:Path += ";$PWD"

# Or move to System32 (requires admin)
Move-Item sloth-kubernetes.exe C:\Windows\System32\
```

### Cloud provider authentication fails

```bash
# Verify token is set
echo $DO_TOKEN

# Test API access manually
curl -H "Authorization: Bearer $DO_TOKEN" \
  "https://api.digitalocean.com/v2/account"

# Re-export token
export DO_TOKEN="your-token-here"
```

## Updating

### Binary Releases

```bash
# Download latest version
curl -sSL https://github.com/chalkan3/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o /tmp/sloth-kubernetes

# Replace existing binary
sudo mv /tmp/sloth-kubernetes /usr/local/bin/sloth-kubernetes
sudo chmod +x /usr/local/bin/sloth-kubernetes

# Verify new version
sloth-kubernetes version
```

### From Source

```bash
cd sloth-kubernetes
git pull origin main
go build -o sloth-kubernetes .
sudo mv sloth-kubernetes /usr/local/bin/
```

## Uninstallation

```bash
# Remove binary
sudo rm /usr/local/bin/sloth-kubernetes

# Remove config directory
rm -rf ~/.config/sloth-kubernetes

# Remove Pulumi state (optional)
rm -rf ~/.pulumi

# Remove SSH keys (optional)
rm -f ~/.ssh/sloth-k8s*
```

## Next Steps

Now that sloth-kubernetes is installed, continue with:

- **[Quick Start](quickstart.md)** - Deploy your first cluster
- **[User Guide](../user-guide/index.md)** - Learn all commands
