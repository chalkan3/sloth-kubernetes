# Sloth Kubernetes - Configuration Examples

This directory contains example configuration files for deploying multi-cloud Kubernetes clusters.

## Configuration Files

### cluster-with-bastion.yaml
Complete production-ready configuration with a bastion host for secure access.

**Features:**
- Bastion host for secure SSH access to private nodes
- WireGuard VPN mesh networking
- VPC configuration for DigitalOcean and Linode
- RKE2 Kubernetes distribution
- Private nodes (no public IPs)

**Use case:** Production deployments requiring enhanced security

### cluster-simple.yaml
Simple multi-cloud cluster configuration with VPN.

**Features:**
- WireGuard VPN for cross-cloud networking
- Multiple master and worker nodes across providers
- VPC configuration
- Public node access (no bastion)

**Use case:** Development and testing environments

### cluster-advanced.yaml
Advanced configuration with granular node control.

**Features:**
- Individual node definitions with custom names
- Custom WireGuard IP assignments
- Node pools for workers
- Custom labels and monitoring

**Use case:** Complex production deployments requiring fine-grained control

### cluster-k8s-style.yaml
Kubernetes-style YAML configuration (apiVersion/kind format).

**Features:**
- Familiar Kubernetes YAML structure
- Complete cluster specification
- Node pool definitions

**Use case:** Users familiar with Kubernetes manifests

## Environment Variables

All configurations use environment variables for sensitive data:

```bash
export DIGITALOCEAN_TOKEN="your-do-token-here"
export LINODE_TOKEN="your-linode-token-here"
```

## Usage

Deploy a cluster using any of these examples:

```bash
# Basic deployment
./sloth-kubernetes deploy my-stack --config examples/cluster-simple.yaml \
  --do-token "$DIGITALOCEAN_TOKEN" \
  --linode-token "$LINODE_TOKEN" -y

# With bastion host
./sloth-kubernetes deploy prod-stack --config examples/cluster-with-bastion.yaml \
  --do-token "$DIGITALOCEAN_TOKEN" \
  --linode-token "$LINODE_TOKEN" -y
```

## Customization

Replace the following placeholders in the examples:

1. **Cluster tokens:** Change `change-this-to-secure-random-token` to a secure random string
2. **Domain names:** Replace `example.com` with your actual domain
3. **Node counts:** Adjust `count` values in node pools based on your needs
4. **Node sizes:** Modify `size` fields to match your resource requirements
5. **Regions:** Update `region` fields to deploy closer to your users

## Network Configuration

All examples include:
- VPC with custom CIDR ranges
- WireGuard VPN for secure multi-cloud networking
- Private networking between nodes
- DNS configuration support

## Security Notes

- Store tokens as environment variables, never commit them to version control
- Use bastion host configuration for production deployments
- Restrict bastion `allowedCIDRs` to known IP ranges in production
- Enable MFA on bastion when deploying to production
- Rotate cluster tokens regularly
