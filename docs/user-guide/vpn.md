---
title: WireGuard VPN
description: Manage WireGuard VPN mesh network for secure cluster connectivity
sidebar_position: 10
---

# WireGuard VPN

sloth-kubernetes automatically configures a WireGuard VPN mesh network between all cluster nodes, enabling secure encrypted communication across multiple clouds and regions.

## Overview

The VPN system provides:
- **Automatic mesh networking**: All nodes interconnected via WireGuard
- **Multi-cloud connectivity**: Secure tunnels between different cloud providers
- **Client access**: Join your local machine or CI servers to the VPN
- **Full mesh topology**: Direct peer-to-peer connections between all nodes
- **Low overhead**: Minimal latency and CPU usage with WireGuard

## Commands

### vpn status

Show VPN status and tunnel information.

```bash
sloth-kubernetes vpn status production
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
           VPN STATUS - Stack: production
═══════════════════════════════════════════════════════════════

METRIC          VALUE
------          -----
VPN Mode        WireGuard Mesh
Total Nodes     6
Total Tunnels   15
VPN Subnet      10.8.0.0/24
Status          All tunnels active
```

### vpn peers

List all VPN peers in the mesh network.

```bash
sloth-kubernetes vpn peers production
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
           VPN PEERS - Stack: production
═══════════════════════════════════════════════════════════════

Fetching peer information from cluster nodes...

NODE         LABEL      VPN IP       PUBLIC KEY        ENDPOINT            LAST HANDSHAKE   TRANSFER
----         -----      ------       ----------        --------            --------------   --------
master-1     -          10.8.0.10    ABC123def456...   167.71.1.1:51820    30s ago          1.2MB / 2.4MB
master-2     -          10.8.0.11    DEF456ghi789...   167.71.1.2:51820    45s ago          800KB / 1.5MB
worker-1     -          10.8.0.20    GHI789jkl012...   172.236.1.1:51820   1m ago           3.5MB / 5.2MB
worker-2     -          10.8.0.21    JKL012mno345...   172.236.1.2:51820   25s ago          2.1MB / 3.8MB
laptop       personal   10.8.0.100   MNO345pqr678...   N/A                 2m ago           500KB / 1.2MB

Found 5 peers in VPN mesh
```

### vpn config

Get WireGuard configuration for a specific node.

```bash
sloth-kubernetes vpn config production master-1
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
           VPN CONFIG - Node: master-1
═══════════════════════════════════════════════════════════════

Fetching WireGuard configuration from master-1...

WireGuard Configuration:

[Interface]
PrivateKey = <private-key>
Address = 10.8.0.10/24
ListenPort = 51820

[Peer]
# master-2
PublicKey = DEF456ghi789...
AllowedIPs = 10.8.0.11/32
Endpoint = 167.71.1.2:51820
PersistentKeepalive = 25

[Peer]
# worker-1
PublicKey = GHI789jkl012...
AllowedIPs = 10.8.0.20/32
Endpoint = 172.236.1.1:51820
PersistentKeepalive = 25

Node: master-1
Public IP: 167.71.1.1
VPN IP: 10.8.0.10
Provider: digitalocean
```

### vpn test

Test VPN connectivity between all nodes.

```bash
sloth-kubernetes vpn test production
```

**Example output:**

```
═══════════════════════════════════════════════════════════════
      TESTING VPN CONNECTIVITY - Stack: production
═══════════════════════════════════════════════════════════════

Found 4 nodes to test

Test 1/3: Testing ping connectivity via VPN...

  [OK] master-1 -> master-2 (10.8.0.11)
  [OK] master-1 -> worker-1 (10.8.0.20)
  [OK] master-1 -> worker-2 (10.8.0.21)
  [OK] master-2 -> master-1 (10.8.0.10)
  [OK] master-2 -> worker-1 (10.8.0.20)
  [OK] master-2 -> worker-2 (10.8.0.21)
  ...

Test 2/3: Checking WireGuard handshake status...

  [OK] master-1 - 3 active peers
  [OK] master-2 - 3 active peers
  [OK] worker-1 - 3 active peers
  [OK] worker-2 - 3 active peers

Test 3/3: Summary

METRIC              RESULT
------              ------
Total Nodes         4
Ping Tests          12/12 passed (100.0%)
Handshake Checks    4/4 nodes responding
Overall Status      All tests passed
```

### vpn join

Join your local machine or a remote host to the VPN mesh.

```bash
# Join local machine to VPN
sloth-kubernetes vpn join production

# Join with custom VPN IP
sloth-kubernetes vpn join production --vpn-ip 10.8.0.100

# Join with label
sloth-kubernetes vpn join production --label laptop

# Join a remote SSH host
sloth-kubernetes vpn join production --remote user@host.com

# Join and auto-install WireGuard config
sloth-kubernetes vpn join production --install
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--remote` | Remote SSH host to add (e.g., user@host.com) | - |
| `--vpn-ip` | Custom VPN IP address | Auto-assign |
| `--label` | Peer label/name (e.g., 'laptop', 'ci-server') | - |
| `--install` | Auto-install WireGuard configuration | `false` |

**What happens during join:**
1. Generates a new WireGuard keypair
2. Auto-assigns a VPN IP (10.8.0.100-254 range)
3. Adds the peer to all cluster nodes
4. Updates existing VPN clients
5. Generates client configuration file
6. Optionally installs and activates WireGuard

### vpn leave

Remove a machine from the VPN mesh.

```bash
# Remove local machine
sloth-kubernetes vpn leave production

# Remove specific peer by VPN IP
sloth-kubernetes vpn leave production --vpn-ip 10.8.0.100
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--vpn-ip` | VPN IP of peer to remove | Auto-detect local |

### vpn client-config

Generate a WireGuard client configuration file.

```bash
# Generate client config
sloth-kubernetes vpn client-config production

# Save to specific file
sloth-kubernetes vpn client-config production --output client.conf

# Generate QR code for mobile devices
sloth-kubernetes vpn client-config production --qr
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--output` | Output file path | `./wg0-client.conf` |
| `--qr` | Generate QR code for mobile | `false` |

---

## VPN Architecture

### IP Address Scheme

| Range | Purpose |
|-------|---------|
| 10.8.0.1-9 | Reserved (bastion, gateways) |
| 10.8.0.10-99 | Cluster nodes |
| 10.8.0.100-254 | External clients |

### Network Topology

```
┌─────────────────────────────────────────────────────────────────┐
│                     WireGuard VPN Mesh                          │
│                      (10.8.0.0/24)                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  DigitalOcean          Linode              AWS                  │
│  ┌──────────┐         ┌──────────┐        ┌──────────┐          │
│  │ master-1 │◄───────►│ master-2 │◄──────►│ worker-1 │          │
│  │10.8.0.10 │         │10.8.0.11 │        │10.8.0.20 │          │
│  └────┬─────┘         └────┬─────┘        └────┬─────┘          │
│       │                    │                   │                │
│       └──────────┬─────────┴───────────────────┘                │
│                  │                                              │
│            ┌─────┴─────┐                                        │
│            │  laptop   │   (External Client)                    │
│            │10.8.0.100 │                                        │
│            └───────────┘                                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Examples

### Join Development Machine

```bash
# Join your laptop to the VPN
sloth-kubernetes vpn join production --label dev-laptop --install

# After installation, verify connectivity
ping 10.8.0.10  # Should reach master-1

# Access cluster services directly
kubectl --server=https://10.8.0.10:6443 get nodes
```

### Join CI/CD Server

```bash
# Join a CI server to the VPN
sloth-kubernetes vpn join production \
  --remote ci@build-server.example.com \
  --label ci-server \
  --install

# The CI server can now access the cluster directly
ssh ci@build-server.example.com
kubectl get nodes  # Works via VPN
```

### Troubleshoot Connectivity

```bash
# Run full connectivity test
sloth-kubernetes vpn test production

# Check specific node configuration
sloth-kubernetes vpn config production worker-1

# List all peers to see handshake times
sloth-kubernetes vpn peers production
```

### Remove Client Access

```bash
# Remove your machine from VPN
sloth-kubernetes vpn leave production

# Remove a specific client by IP
sloth-kubernetes vpn leave production --vpn-ip 10.8.0.105
```

---

## Manual Client Setup

If you prefer to set up WireGuard manually:

### Linux

```bash
# Install WireGuard
sudo apt install wireguard

# Copy configuration
sudo cp wg0-client.conf /etc/wireguard/wg0.conf

# Start VPN
sudo wg-quick up wg0

# Enable on boot
sudo systemctl enable wg-quick@wg0
```

### macOS

```bash
# Install WireGuard (via Homebrew)
brew install wireguard-tools

# Copy configuration
sudo mkdir -p /opt/homebrew/etc/wireguard
sudo cp wg0-client.conf /opt/homebrew/etc/wireguard/wg0.conf

# Start VPN
sudo wg-quick up wg0

# Or use WireGuard app from App Store
# Import the .conf file
```

### Windows

1. Download WireGuard from https://www.wireguard.com/install/
2. Open WireGuard application
3. Click "Import tunnel(s) from file"
4. Select the `wg0-client.conf` file
5. Click "Activate" to connect

### Mobile (iOS/Android)

1. Install WireGuard app from App Store / Play Store
2. Generate QR code: `sloth-kubernetes vpn client-config production --qr`
3. In app, tap "+" and "Create from QR code"
4. Scan the QR code
5. Activate the tunnel

---

## Troubleshooting

### VPN Handshake Not Completing

Check firewall rules on all nodes:

```bash
# Verify UDP port 51820 is open
sloth-kubernetes nodes ssh master-1
sudo iptables -L -n | grep 51820

# Check WireGuard interface
sudo wg show
```

### Peer Not Reachable

Verify peer configuration:

```bash
# Check if peer is in config
sloth-kubernetes vpn config production master-1 | grep -A5 "worker-1"

# Test connectivity directly
sloth-kubernetes nodes ssh master-1
ping 10.8.0.20  # Worker-1 VPN IP
```

### Join Command Fails

Ensure SSH access is working:

```bash
# Test SSH to nodes
sloth-kubernetes nodes ssh master-1

# Check SSH key path
ls -la ~/.sloth-kubernetes/keys/
```

### Local WireGuard Won't Start

Check for conflicts:

```bash
# Stop any existing WireGuard
sudo wg-quick down wg0

# Check for interface conflicts
ip link show

# Verify configuration syntax
sudo wg-quick strip wg0 < wg0-client.conf
```

### Performance Issues

Check MTU settings:

```bash
# On nodes, verify MTU
sloth-kubernetes nodes ssh master-1
cat /etc/wireguard/wg0.conf | grep MTU

# Typical MTU for WireGuard: 1420
# Adjust if seeing packet fragmentation
```
