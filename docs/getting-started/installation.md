# 🦥 Installation

Get Sloth Kubernetes up and running. Slowly, but surely!

---

## Prerequisites

!!! tip "The Sloth Way 🦥"
    We keep dependencies minimal. You only need API tokens from your cloud providers!

Before you start, make sure you have:

- ✅ **Cloud Provider Account** - DigitalOcean and/or Linode
- ✅ **API Tokens** - From your cloud provider(s)
- ✅ **SSH Access** - For node management (optional but recommended)

That's it! No kubectl, no Pulumi CLI, no Terraform. Just one lazy sloth binary! 🦥

---

## Download Binary

### Option 1: Latest Release (Recommended)

Download the pre-compiled binary for your platform:

=== "Linux (x64)"

    ```bash
    # Download latest release
    curl -L https://github.com/yourusername/sloth-kubernetes/releases/latest/download/sloth-kubernetes-linux-amd64 -o sloth-kubernetes

    # Make executable
    chmod +x sloth-kubernetes

    # Move to PATH
    sudo mv sloth-kubernetes /usr/local/bin/

    # Verify installation 🦥
    sloth-kubernetes version
    ```

=== "macOS (Intel)"

    ```bash
    # Download latest release
    curl -L https://github.com/yourusername/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-amd64 -o sloth-kubernetes

    # Make executable
    chmod +x sloth-kubernetes

    # Move to PATH
    sudo mv sloth-kubernetes /usr/local/bin/

    # Verify installation 🦥
    sloth-kubernetes version
    ```

=== "macOS (Apple Silicon)"

    ```bash
    # Download latest release
    curl -L https://github.com/yourusername/sloth-kubernetes/releases/latest/download/sloth-kubernetes-darwin-arm64 -o sloth-kubernetes

    # Make executable
    chmod +x sloth-kubernetes

    # Move to PATH
    sudo mv sloth-kubernetes /usr/local/bin/

    # Verify installation 🦥
    sloth-kubernetes version
    ```

### Option 2: Build from Source

For the adventurous sloths who want the latest features! 🦥

```bash
# Clone the repository
git clone https://github.com/yourusername/sloth-kubernetes.git
cd sloth-kubernetes

# Build (requires Go 1.23+)
go build -o sloth-kubernetes

# Move to PATH
sudo mv sloth-kubernetes /usr/local/bin/

# Verify 🦥
sloth-kubernetes version
```

---

## Configure API Tokens

Sloth Kubernetes needs API tokens to create resources in your cloud providers.

### DigitalOcean

1. Go to [DigitalOcean API Tokens](https://cloud.digitalocean.com/account/api/tokens)
2. Click "Generate New Token"
3. Name it "sloth-kubernetes" 🦥
4. Select Read & Write scope
5. Copy the token

```bash
# Set environment variable
export DIGITALOCEAN_TOKEN="dop_v1_abc123..."

# Or add to your shell profile (~/.bashrc, ~/.zshrc)
echo 'export DIGITALOCEAN_TOKEN="dop_v1_abc123..."' >> ~/.bashrc
```

### Linode

1. Go to [Linode API Tokens](https://cloud.linode.com/profile/tokens)
2. Click "Create a Personal Access Token"
3. Label it "sloth-kubernetes" 🦥
4. Select Read/Write for Linodes, VPCs
5. Copy the token

```bash
# Set environment variable
export LINODE_TOKEN="abc123..."

# Or add to your shell profile
echo 'export LINODE_TOKEN="abc123..."' >> ~/.bashrc
```

!!! warning "Keep Your Tokens Safe 🦥"
    Never commit API tokens to Git! Use environment variables or secret management tools.

---

## Verify Installation

Let's make sure everything is working:

```bash
# Check version
sloth-kubernetes version

# View help
sloth-kubernetes --help

# Test configuration (dry run)
sloth-kubernetes deploy --config examples/simple-cluster.yaml --dry-run
```

You should see output like:

```
🦥 Sloth Kubernetes v1.0.0
Slowly, but surely deploying your cluster...
```

---

## Optional: Shell Completion

Make your life easier with shell completion! 🦥

=== "Bash"

    ```bash
    # Generate completion script
    sloth-kubernetes completion bash > /tmp/sloth-completion.bash

    # Install system-wide
    sudo mv /tmp/sloth-completion.bash /etc/bash_completion.d/sloth-kubernetes

    # Or just for your user
    echo 'source <(sloth-kubernetes completion bash)' >> ~/.bashrc
    source ~/.bashrc
    ```

=== "Zsh"

    ```bash
    # Generate completion script
    sloth-kubernetes completion zsh > "${fpath[1]}/_sloth-kubernetes"

    # Or add to .zshrc
    echo 'source <(sloth-kubernetes completion zsh)' >> ~/.zshrc
    source ~/.zshrc
    ```

=== "Fish"

    ```bash
    # Generate completion script
    sloth-kubernetes completion fish > ~/.config/fish/completions/sloth-kubernetes.fish
    ```

---

## What's Next? 🦥

Now that you're all set up, let's deploy your first cluster!

<div class="grid cards" markdown>

-   📘 **Quick Start**

    ---

    Deploy your first cluster in 5 minutes! 🦥

    [:octicons-arrow-right-24: Quick Start](quickstart.md)

-   🎯 **First Cluster**

    ---

    Step-by-step guide to your first production cluster 🦥

    [:octicons-arrow-right-24: First Cluster](first-cluster.md)

-   📚 **Configuration**

    ---

    Learn about cluster configuration options 🦥

    [:octicons-arrow-right-24: Configuration](../configuration/file-structure.md)

</div>

---

!!! quote "Sloth Wisdom 🦥"
    *"The journey of a thousand miles begins with a single step... but take your time!"*
