# SaltStack API Integration

This document describes how to use the SaltStack integration in sloth-kubernetes.

## Overview

SaltStack is automatically installed on the bastion host during cluster deployment. The Salt Master runs on the bastion and provides a REST API for remote execution and configuration management.

## Configuration

### Environment Variables

Set these environment variables to configure Salt API access:

```bash
export SALT_API_URL="http://your-bastion-ip:8000"
export SALT_USERNAME="saltapi"
export SALT_PASSWORD="saltapi123"
```

### Command Line Flags

Alternatively, use flags with each command:

```bash
sloth-kubernetes salt ping \
  --url "http://bastion-ip:8000" \
  --username "saltapi" \
  --password "saltapi123"
```

## Available Commands

### 1. Ping Minions

Test connectivity to Salt minions:

```bash
# Ping all minions
sloth-kubernetes salt ping

# Ping specific minions
sloth-kubernetes salt ping --target "master*"
sloth-kubernetes salt ping --target "worker*"
```

### 2. List Minions

List all connected minions:

```bash
sloth-kubernetes salt minions
```

### 3. Execute Shell Commands

Run shell commands on minions:

```bash
# Check uptime on all nodes
sloth-kubernetes salt cmd "uptime"

# Check disk usage
sloth-kubernetes salt cmd "df -h"

# Check K3s status on masters
sloth-kubernetes salt cmd "systemctl status k3s" --target "master*"

# View system logs
sloth-kubernetes salt cmd "journalctl -u k3s -n 50 --no-pager" --target "master*"

# Check WireGuard status
sloth-kubernetes salt cmd "wg show"
```

### 4. Get System Information (Grains)

Retrieve detailed system information:

```bash
# Get grains from all minions
sloth-kubernetes salt grains

# Get grains from specific minions
sloth-kubernetes salt grains --target "worker*"

# Output as JSON
sloth-kubernetes salt grains --json > grains.json
```

### 5. Apply Salt States

Apply configuration states to minions:

```bash
# Apply a specific state
sloth-kubernetes salt state apply nginx --target "web*"

# Apply highstate (all states)
sloth-kubernetes salt state highstate

# Apply highstate to specific minions
sloth-kubernetes salt state highstate --target "master*"
```

### 6. Manage Minion Keys

Manage Salt minion authentication keys:

```bash
# List all keys
sloth-kubernetes salt keys list

# Accept a pending minion key
sloth-kubernetes salt keys accept node-1

# Accept all pending keys
sloth-kubernetes salt keys accept "*"
```

## Targeting Minions

Salt supports various targeting methods using the `--target` flag:

### Glob (default)
```bash
# All minions
--target "*"

# All masters
--target "master*"

# All workers
--target "worker*"

# Specific node
--target "node-1"
```

### Examples by Role

```bash
# Target all master nodes
sloth-kubernetes salt cmd "kubectl get nodes" --target "master*"

# Target all worker nodes
sloth-kubernetes salt cmd "systemctl restart kubelet" --target "worker*"

# Target specific node
sloth-kubernetes salt cmd "reboot" --target "node-3"
```

## Common Use Cases

### 1. Cluster Health Check

```bash
# Ping all nodes
sloth-kubernetes salt ping

# Check K3s status on all nodes
sloth-kubernetes salt cmd "systemctl status k3s"

# Check WireGuard connectivity
sloth-kubernetes salt cmd "wg show"
```

### 2. Update Packages

```bash
# Update package lists
sloth-kubernetes salt cmd "apt-get update" --target "*"

# Upgrade packages (dry-run)
sloth-kubernetes salt cmd "apt-get upgrade --dry-run" --target "*"

# Install specific package
sloth-kubernetes salt cmd "apt-get install -y htop" --target "*"
```

### 3. Monitor Resources

```bash
# Check CPU usage
sloth-kubernetes salt cmd "top -bn1 | head -20"

# Check memory usage
sloth-kubernetes salt cmd "free -h"

# Check disk I/O
sloth-kubernetes salt cmd "iostat"
```

### 4. Manage Services

```bash
# Restart K3s on all nodes
sloth-kubernetes salt cmd "systemctl restart k3s"

# Check service status
sloth-kubernetes salt cmd "systemctl status wireguard-wg0"

# View logs
sloth-kubernetes salt cmd "journalctl -u k3s -f --no-pager -n 100"
```

### 5. Network Diagnostics

```bash
# Check network interfaces
sloth-kubernetes salt cmd "ip addr show"

# Test connectivity between nodes
sloth-kubernetes salt cmd "ping -c 4 10.8.0.1"

# Check firewall rules
sloth-kubernetes salt cmd "ufw status verbose"
```

### 6. Security Updates

```bash
# Check for security updates
sloth-kubernetes salt cmd "apt-get upgrade --dry-run | grep -i security"

# Apply security updates
sloth-kubernetes salt cmd "apt-get upgrade -y"
```

## JSON Output

All commands support JSON output for scripting:

```bash
# Get JSON output
sloth-kubernetes salt grains --json > output.json

# Process with jq
sloth-kubernetes salt grains --json | jq '.return[0]'
```

## Integration with Kubernetes

### Get Cluster Info

```bash
# Get nodes
sloth-kubernetes salt cmd "kubectl get nodes" --target "master-1"

# Get pods
sloth-kubernetes salt cmd "kubectl get pods -A" --target "master-1"

# Get cluster info
sloth-kubernetes salt cmd "kubectl cluster-info" --target "master-1"
```

### Deploy Applications

```bash
# Apply manifest
sloth-kubernetes salt cmd "kubectl apply -f /path/to/manifest.yaml" --target "master-1"

# Scale deployment
sloth-kubernetes salt cmd "kubectl scale deployment nginx --replicas=3" --target "master-1"
```

## Troubleshooting

### Connection Issues

```bash
# Check if Salt API is running
curl -k https://bastion-ip:8000

# Check Salt Master status
ssh bastion-ip "systemctl status salt-master"
ssh bastion-ip "systemctl status salt-api"

# View Salt Master logs
ssh bastion-ip "journalctl -u salt-master -n 100"
```

### Minion Not Responding

```bash
# Check if minion is running
ssh node-ip "systemctl status salt-minion"

# View minion logs
ssh node-ip "journalctl -u salt-minion -n 100"

# Restart minion
ssh node-ip "systemctl restart salt-minion"
```

### Key Management

```bash
# List pending keys
sloth-kubernetes salt keys list

# Accept pending keys
sloth-kubernetes salt keys accept "*"

# Re-generate minion key
ssh node-ip "rm -rf /etc/salt/pki/minion/*"
ssh node-ip "systemctl restart salt-minion"
```

## Security Notes

1. **Change Default Password**: The default Salt API password is `saltapi123`. Change it in production:
   ```bash
   ssh bastion-ip "echo 'saltapi:new-secure-password' | sudo chpasswd"
   ```

2. **Use HTTPS**: In production, configure proper SSL certificates for the Salt API.

3. **Firewall Rules**: The Salt API (port 8000) should only be accessible from trusted networks.

4. **Key Management**: Regularly audit and revoke unused minion keys.

## Advanced Usage

### Custom States

Create custom Salt states on the bastion:

```bash
# SSH to bastion
ssh bastion-ip

# Create state directory
sudo mkdir -p /srv/salt/webserver

# Create state file
sudo cat > /srv/salt/webserver/init.sls <<EOF
nginx:
  pkg.installed: []
  service.running:
    - enable: True
    - require:
      - pkg: nginx
EOF

# Apply the state
sloth-kubernetes salt state apply webserver --target "web*"
```

### Asynchronous Execution

For long-running commands, use async execution:

```bash
# The Salt API supports async execution via the 'local_async' client
# This can be added as an enhancement to the CLI
```

## Additional Resources

- [Salt API Documentation](https://docs.saltproject.io/en/latest/ref/netapi/all/salt.netapi.rest_cherrypy.html)
- [Salt Execution Modules](https://docs.saltproject.io/en/latest/ref/modules/all/index.html)
- [Salt States](https://docs.saltproject.io/en/latest/ref/states/all/index.html)
