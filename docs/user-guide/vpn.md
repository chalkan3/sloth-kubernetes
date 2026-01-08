---
title: VPN Networking
description: Secure mesh networking with WireGuard or Tailscale/Headscale
sidebar_position: 10
---

# VPN Networking

sloth-kubernetes provides secure mesh networking between all cluster nodes and your local machine. You can choose between two VPN providers:

| Provider | Description | Best For |
|----------|-------------|----------|
| **WireGuard** | Direct peer-to-peer mesh network | Self-managed, full control |
| **Tailscale/Headscale** | Managed mesh with coordination server | Easier management, NAT traversal |

---

## Choosing a VPN Provider

### WireGuard

- **Pros**: No external dependencies, full control, minimal overhead
- **Cons**: Manual peer management, requires open UDP port

### Tailscale/Headscale

- **Pros**: Automatic peer discovery, NAT traversal, embedded client
- **Cons**: Requires Headscale coordination server

---

## Configuration

### WireGuard Mode

```lisp
(cluster
  (name "my-cluster")

  (network
    (mode "wireguard")
    (cidr "10.8.0.0/24")

    (wireguard
      (enabled true)
      (create true)
      (mesh-networking true)
      (port 51820))))
```

### Tailscale/Headscale Mode

```lisp
(cluster
  (name "my-cluster")

  (network
    (mode "tailscale")
    (cidr "100.64.0.0/10")

    (tailscale
      (enabled true)
      (namespace "kubernetes")
      (tags ("tag:k8s-node" "tag:production"))
      (accept-routes true))))
```

---

# Tailscale/Headscale

When using Tailscale mode, sloth-kubernetes automatically deploys a Headscale coordination server and configures all nodes to join the mesh network.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Tailscale Mesh Network                       │
│                     (100.64.0.0/10)                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│              ┌──────────────┐                                   │
│              │  Headscale   │  (Coordination Server)            │
│              │   Server     │                                   │
│              └──────┬───────┘                                   │
│                     │                                           │
│     ┌───────────────┼───────────────┐                           │
│     │               │               │                           │
│  ┌──┴───┐       ┌───┴──┐       ┌────┴─┐                         │
│  │master│◄─────►│worker│◄─────►│worker│  (Cluster Nodes)        │
│  │  -1  │       │  -1  │       │  -2  │                         │
│  └──┬───┘       └──────┘       └──────┘                         │
│     │                                                           │
│  ┌──┴───────┐                                                   │
│  │  laptop  │  (Your Machine - Embedded Client)                 │
│  │100.64.x.x│                                                   │
│  └──────────┘                                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Commands

### vpn connect

Connect your local machine to the Tailscale mesh using the embedded client. No system-wide Tailscale installation required.

```bash
# Connect in foreground
sloth-kubernetes vpn connect my-cluster

# Connect in background (daemon mode) - recommended
sloth-kubernetes vpn connect my-cluster --daemon

# Connect with custom hostname
sloth-kubernetes vpn connect my-cluster --daemon --hostname my-laptop
```

**What happens:**
1. Reads Headscale URL and API key from Pulumi state
2. Creates an auth key for your machine
3. Starts embedded Tailscale client (tsnet)
4. Starts SOCKS5 proxy for kubectl routing
5. Saves connection state to `~/.sloth/vpn/<cluster>/`

**Output:**
```
🔌 VPN Connect (Daemon) - Stack: my-cluster

Starting VPN daemon in background...
Waiting for VPN connection to establish...
✓ VPN daemon started (PID: 12345)
SOCKS5 proxy running on 127.0.0.1:64172

  kubectl commands will automatically use the VPN tunnel
  Use 'sloth vpn disconnect my-cluster' to stop
```

### vpn disconnect

Disconnect from the Tailscale mesh.

```bash
sloth-kubernetes vpn disconnect my-cluster
```

**Output:**
```
🔌 VPN Disconnect - Stack: my-cluster

Stopping VPN daemon (PID: 12345)...
✓ VPN daemon stopped
Cleaning up connection state...
✓ Disconnected and cleaned up VPN state
```

### kubectl (Automatic VPN Routing)

When connected to VPN, kubectl commands automatically route through the VPN tunnel:

```bash
# These commands work through VPN automatically
sloth-kubernetes kubectl my-cluster get nodes
sloth-kubernetes kubectl my-cluster get pods -A
sloth-kubernetes kubectl my-cluster apply -f deployment.yaml
```

The embedded kubectl detects the running VPN daemon and configures the SOCKS5 proxy automatically.

## Embedded Client Features

The embedded Tailscale client provides:

- **No Installation Required**: Uses tsnet library, no system Tailscale needed
- **Isolated Network Stack**: Runs in userspace, doesn't affect system networking
- **SOCKS5 Proxy**: Allows other tools (kubectl) to route through VPN
- **Automatic Auth**: Creates ephemeral auth keys from Headscale
- **State Persistence**: Saves connection state for reconnection

## Files and State

Connection state is stored in `~/.sloth/vpn/<cluster-name>/`:

| File | Description |
|------|-------------|
| `connection.json` | Connection metadata (Headscale URL, hostname) |
| `daemon.pid` | PID of running daemon process |
| `proxy.port` | SOCKS5 proxy port number |
| `tailscaled.state` | Tailscale client state |

---

# WireGuard

WireGuard provides direct peer-to-peer mesh networking with minimal overhead.

## Architecture

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

## IP Address Scheme

| Range | Purpose |
|-------|---------|
| 10.8.0.1-9 | Reserved (bastion, gateways) |
| 10.8.0.10-99 | Cluster nodes |
| 10.8.0.100-254 | External clients |

## Commands

### vpn status

Show VPN status and tunnel information.

```bash
sloth-kubernetes vpn status production
```

**Output:**
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

**Output:**
```
NODE         LABEL      VPN IP       PUBLIC KEY        ENDPOINT            LAST HANDSHAKE   TRANSFER
----         -----      ------       ----------        --------            --------------   --------
master-1     -          10.8.0.10    ABC123def456...   167.71.1.1:51820    30s ago          1.2MB / 2.4MB
master-2     -          10.8.0.11    DEF456ghi789...   167.71.1.2:51820    45s ago          800KB / 1.5MB
worker-1     -          10.8.0.20    GHI789jkl012...   172.236.1.1:51820   1m ago           3.5MB / 5.2MB
laptop       personal   10.8.0.100   MNO345pqr678...   N/A                 2m ago           500KB / 1.2MB

Found 4 peers in VPN mesh
```

### vpn config

Get WireGuard configuration for a specific node.

```bash
sloth-kubernetes vpn config production master-1
```

### vpn test

Test VPN connectivity between all nodes.

```bash
sloth-kubernetes vpn test production
```

**Output:**
```
Test 1/3: Testing ping connectivity via VPN...
  [OK] master-1 -> master-2 (10.8.0.11)
  [OK] master-1 -> worker-1 (10.8.0.20)
  ...

Test 2/3: Checking WireGuard handshake status...
  [OK] master-1 - 3 active peers
  [OK] master-2 - 3 active peers
  ...

Test 3/3: Summary
  Ping Tests          12/12 passed (100.0%)
  Handshake Checks    4/4 nodes responding
  Overall Status      All tests passed
```

### vpn join

Join your local machine or a remote host to the WireGuard mesh.

```bash
# Join local machine
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
| `--remote` | Remote SSH host to add | - |
| `--vpn-ip` | Custom VPN IP address | Auto-assign |
| `--label` | Peer label/name | - |
| `--install` | Auto-install WireGuard | `false` |

### vpn leave

Remove a machine from the VPN mesh.

```bash
# Remove local machine
sloth-kubernetes vpn leave production

# Remove specific peer by VPN IP
sloth-kubernetes vpn leave production --vpn-ip 10.8.0.100
```

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

## Manual WireGuard Setup

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
# Install WireGuard
brew install wireguard-tools

# Copy configuration
sudo mkdir -p /opt/homebrew/etc/wireguard
sudo cp wg0-client.conf /opt/homebrew/etc/wireguard/wg0.conf

# Start VPN
sudo wg-quick up wg0

# Or use WireGuard app from App Store
```

### Windows

1. Download WireGuard from https://www.wireguard.com/install/
2. Open WireGuard application
3. Click "Import tunnel(s) from file"
4. Select the `wg0-client.conf` file
5. Click "Activate"

### Mobile (iOS/Android)

1. Install WireGuard app
2. Generate QR code: `sloth-kubernetes vpn client-config production --qr`
3. Scan QR code in app
4. Activate tunnel

---

## Comparison: WireGuard vs Tailscale

| Feature | WireGuard | Tailscale/Headscale |
|---------|-----------|---------------------|
| **Setup** | Manual peer config | Automatic discovery |
| **NAT Traversal** | Requires open port | Built-in (DERP relays) |
| **Client Install** | System WireGuard | Embedded (no install) |
| **kubectl Integration** | Manual proxy setup | Automatic SOCKS5 |
| **Peer Management** | Manual add/remove | Automatic via Headscale |
| **Coordination** | None (peer-to-peer) | Headscale server |
| **Overhead** | Minimal | Slightly higher |

**Recommendation:**
- Use **Tailscale/Headscale** for easier management and kubectl integration
- Use **WireGuard** for minimal dependencies and full control

---

## Troubleshooting

### Tailscale: Connection Timeout

```bash
# Check if Headscale server is reachable
curl -k https://<headscale-url>/health

# Check daemon logs
cat ~/.sloth/vpn/<cluster>/tailscaled.log1.txt
```

### Tailscale: kubectl Not Working

```bash
# Verify daemon is running
ps aux | grep "vpn connect"

# Check proxy port
cat ~/.sloth/vpn/<cluster>/proxy.port

# Test proxy manually
curl --socks5 127.0.0.1:<port> https://<kubernetes-api>:6443/healthz
```

### WireGuard: Handshake Not Completing

```bash
# Verify UDP port 51820 is open
sloth-kubernetes nodes ssh master-1
sudo iptables -L -n | grep 51820

# Check WireGuard interface
sudo wg show
```

### WireGuard: Peer Not Reachable

```bash
# Check peer configuration
sloth-kubernetes vpn config production master-1 | grep -A5 "worker-1"

# Test connectivity
sloth-kubernetes nodes ssh master-1
ping 10.8.0.20  # Worker VPN IP
```
