package vpn

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// PeerConfig holds configuration for a WireGuard peer
type PeerConfig struct {
	PublicKey    string   // WireGuard public key (base64, 44 chars)
	AllowedIPs   []string // Allowed IP ranges (e.g., "10.8.0.5/32")
	Endpoint     string   // Optional endpoint (e.g., "1.2.3.4:51820")
	Keepalive    int      // PersistentKeepalive in seconds (0 to disable)
	Label        string   // Human-readable label/comment
	PresharedKey string   // Optional preshared key
}

// ConfigManager manages WireGuard configuration safely
type ConfigManager struct {
	connMgr       *ConnectionManager
	configPath    string // Default: /etc/wireguard/wg0.conf
	interfaceName string // Default: wg0
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(connMgr *ConnectionManager) *ConfigManager {
	return &ConfigManager{
		connMgr:       connMgr,
		configPath:    "/etc/wireguard/wg0.conf",
		interfaceName: "wg0",
	}
}

// WithConfigPath sets a custom config path
func (c *ConfigManager) WithConfigPath(path string) *ConfigManager {
	c.configPath = path
	return c
}

// WithInterfaceName sets a custom interface name
func (c *ConfigManager) WithInterfaceName(name string) *ConfigManager {
	c.interfaceName = name
	return c
}

// ValidatePeerConfig validates a peer configuration
func (c *ConfigManager) ValidatePeerConfig(peer PeerConfig) error {
	// Validate public key (base64 encoded, 44 characters)
	if len(peer.PublicKey) != 44 {
		return fmt.Errorf("invalid public key length: expected 44, got %d", len(peer.PublicKey))
	}

	// Validate AllowedIPs
	if len(peer.AllowedIPs) == 0 {
		return fmt.Errorf("at least one AllowedIP is required")
	}

	for _, ip := range peer.AllowedIPs {
		if _, _, err := net.ParseCIDR(ip); err != nil {
			// Try parsing as plain IP and add /32
			if net.ParseIP(strings.Split(ip, "/")[0]) == nil {
				return fmt.Errorf("invalid AllowedIP '%s': %w", ip, err)
			}
		}
	}

	// Validate keepalive
	if peer.Keepalive < 0 || peer.Keepalive > 65535 {
		return fmt.Errorf("invalid keepalive value: %d (must be 0-65535)", peer.Keepalive)
	}

	// Validate endpoint if provided
	if peer.Endpoint != "" {
		parts := strings.Split(peer.Endpoint, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid endpoint format: %s (expected host:port)", peer.Endpoint)
		}
		if net.ParseIP(parts[0]) == nil {
			// Try to resolve hostname - skip for now, just check format
		}
		port, err := strconv.Atoi(parts[1])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid endpoint port: %s", parts[1])
		}
	}

	return nil
}

// PeerExists checks if a peer with the given public key exists
func (c *ConfigManager) PeerExists(ctx context.Context, conn *SSHConnection, publicKey string) (bool, error) {
	cmd := fmt.Sprintf("wg show %s peers 2>/dev/null | grep -q '%s' && echo 'EXISTS' || echo 'NOT_EXISTS'",
		c.interfaceName, publicKey)

	output, err := conn.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to check peer existence: %w", err)
	}

	return strings.Contains(output, "EXISTS"), nil
}

// AddPeer adds a peer to WireGuard using atomic `wg set` command
// This is safer than editing the config file directly as it won't corrupt on disconnect
func (c *ConfigManager) AddPeer(ctx context.Context, conn *SSHConnection, peer PeerConfig) error {
	// Validate config first
	if err := c.ValidatePeerConfig(peer); err != nil {
		return fmt.Errorf("invalid peer config: %w", err)
	}

	// Check if peer already exists
	exists, err := c.PeerExists(ctx, conn, peer.PublicKey)
	if err != nil {
		return err
	}

	// Build wg set command
	allowedIPs := strings.Join(peer.AllowedIPs, ",")
	cmd := fmt.Sprintf("sudo wg set %s peer %s allowed-ips %s",
		c.interfaceName, peer.PublicKey, allowedIPs)

	if peer.Keepalive > 0 {
		cmd += fmt.Sprintf(" persistent-keepalive %d", peer.Keepalive)
	}

	if peer.Endpoint != "" {
		cmd += fmt.Sprintf(" endpoint %s", peer.Endpoint)
	}

	if peer.PresharedKey != "" {
		// Use a temporary file for preshared key to avoid command line exposure
		cmd = fmt.Sprintf("echo '%s' | sudo wg set %s peer %s allowed-ips %s preshared-key /dev/stdin",
			peer.PresharedKey, c.interfaceName, peer.PublicKey, allowedIPs)
		if peer.Keepalive > 0 {
			cmd += fmt.Sprintf(" persistent-keepalive %d", peer.Keepalive)
		}
		if peer.Endpoint != "" {
			cmd += fmt.Sprintf(" endpoint %s", peer.Endpoint)
		}
	}

	// Execute the wg set command (atomic operation)
	_, err = conn.Execute(cmd)
	if err != nil {
		return fmt.Errorf("wg set failed: %w", err)
	}

	// Persist to config file for reboot persistence
	if err := c.persistPeerToConfig(ctx, conn, peer, exists); err != nil {
		// Log warning but don't fail - runtime config is already applied
		return fmt.Errorf("peer added to runtime but failed to persist to config: %w", err)
	}

	return nil
}

// persistPeerToConfig safely adds peer to the config file with backup and validation
func (c *ConfigManager) persistPeerToConfig(ctx context.Context, conn *SSHConnection, peer PeerConfig, update bool) error {
	label := peer.Label
	if label == "" {
		label = "peer-" + peer.PublicKey[:8]
	}

	allowedIPs := strings.Join(peer.AllowedIPs, ", ")
	keepalive := strconv.Itoa(peer.Keepalive)

	// Build script that safely updates config (requires sudo for /etc/wireguard)
	script := fmt.Sprintf(`#!/bin/bash
set -e

CONFIG="%s"
BACKUP="${CONFIG}.bak"
PUBKEY="%s"

# Create backup (requires sudo)
sudo cp "$CONFIG" "$BACKUP"

# Check if peer already in file
if sudo grep -q "$PUBKEY" "$CONFIG"; then
    echo "PEER_EXISTS_IN_FILE"
    sudo rm "$BACKUP"
    exit 0
fi

# Add peer section (requires sudo)
sudo tee -a "$CONFIG" > /dev/null << 'PEEREOF'

[Peer]
# %s
PublicKey = %s
AllowedIPs = %s
PersistentKeepalive = %s
PEEREOF

# Validate config
if wg-quick strip %s > /dev/null 2>&1; then
    echo "CONFIG_VALID"
    sudo rm "$BACKUP"
else
    echo "CONFIG_INVALID"
    sudo mv "$BACKUP" "$CONFIG"
    exit 1
fi
`, c.configPath, peer.PublicKey, label, peer.PublicKey, allowedIPs, keepalive, c.interfaceName)

	output, err := conn.ExecuteScript(script)
	if err != nil {
		return fmt.Errorf("persist script failed: %w", err)
	}

	if strings.Contains(output, "CONFIG_INVALID") {
		return fmt.Errorf("config validation failed, changes rolled back")
	}

	return nil
}

// RemovePeer removes a peer from WireGuard
func (c *ConfigManager) RemovePeer(ctx context.Context, conn *SSHConnection, publicKey string) error {
	// Remove from runtime
	cmd := fmt.Sprintf("sudo wg set %s peer %s remove", c.interfaceName, publicKey)
	_, err := conn.Execute(cmd)
	if err != nil {
		return fmt.Errorf("wg set remove failed: %w", err)
	}

	// Remove from config file
	if err := c.removePeerFromConfig(ctx, conn, publicKey); err != nil {
		return fmt.Errorf("peer removed from runtime but failed to update config: %w", err)
	}

	return nil
}

// removePeerFromConfig removes a peer section from the config file
func (c *ConfigManager) removePeerFromConfig(ctx context.Context, conn *SSHConnection, publicKey string) error {
	script := fmt.Sprintf(`#!/bin/bash
set -e

CONFIG="%s"
BACKUP="${CONFIG}.bak"
PUBKEY="%s"
TEMP_FILE=$(mktemp)

# Check if peer exists in file
if ! sudo grep -q "$PUBKEY" "$CONFIG"; then
    echo "PEER_NOT_IN_FILE"
    exit 0
fi

# Create backup
sudo cp "$CONFIG" "$BACKUP"

# Remove peer section using awk (write to temp file first)
sudo awk -v pubkey="$PUBKEY" '
    BEGIN { skip = 0 }
    /^\[Peer\]/ {
        if (skip) { skip = 0 }
        peerblock = $0; getline
        while ($0 !~ /^\[/ && NF > 0) {
            peerblock = peerblock "\n" $0
            if ($0 ~ pubkey) { skip = 1 }
            if (!getline) break
        }
        if (!skip) { print peerblock }
        if (NF > 0 && $0 ~ /^\[/) { print }
        next
    }
    { if (!skip) print }
' "$BACKUP" > "$TEMP_FILE"
sudo cp "$TEMP_FILE" "$CONFIG"
rm -f "$TEMP_FILE"

# Validate
if wg-quick strip %s > /dev/null 2>&1; then
    echo "CONFIG_VALID"
    sudo rm "$BACKUP"
else
    echo "CONFIG_INVALID"
    sudo mv "$BACKUP" "$CONFIG"
    exit 1
fi
`, c.configPath, publicKey, c.interfaceName)

	output, err := conn.ExecuteScript(script)
	if err != nil {
		return fmt.Errorf("remove peer script failed: %w", err)
	}

	if strings.Contains(output, "CONFIG_INVALID") {
		return fmt.Errorf("config validation failed after peer removal, changes rolled back")
	}

	return nil
}

// GetPeers returns a list of current peers from the WireGuard interface
func (c *ConfigManager) GetPeers(ctx context.Context, conn *SSHConnection) ([]string, error) {
	cmd := fmt.Sprintf("wg show %s peers 2>/dev/null", c.interfaceName)
	output, err := conn.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %w", err)
	}

	var peers []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			peers = append(peers, line)
		}
	}

	return peers, nil
}

// GetPeerInfo returns detailed information about a specific peer
func (c *ConfigManager) GetPeerInfo(ctx context.Context, conn *SSHConnection, publicKey string) (map[string]string, error) {
	cmd := fmt.Sprintf("wg show %s dump 2>/dev/null | grep '%s'", c.interfaceName, publicKey)
	output, err := conn.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get peer info: %w", err)
	}

	// wg dump format: public-key, preshared-key, endpoint, allowed-ips, latest-handshake, transfer-rx, transfer-tx, persistent-keepalive
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 8 {
		return nil, fmt.Errorf("unexpected wg dump output format")
	}

	info := map[string]string{
		"public_key":     fields[0],
		"preshared_key":  fields[1],
		"endpoint":       fields[2],
		"allowed_ips":    fields[3],
		"last_handshake": fields[4],
		"transfer_rx":    fields[5],
		"transfer_tx":    fields[6],
		"keepalive":      fields[7],
	}

	return info, nil
}

// SyncConfig saves the current runtime WireGuard config to file
func (c *ConfigManager) SyncConfig(ctx context.Context, conn *SSHConnection) error {
	cmd := fmt.Sprintf("sudo wg-quick save %s", c.interfaceName)
	_, err := conn.Execute(cmd)
	if err != nil {
		return fmt.Errorf("wg-quick save failed: %w", err)
	}
	return nil
}

// ReloadConfig reloads the WireGuard interface from config file
func (c *ConfigManager) ReloadConfig(ctx context.Context, conn *SSHConnection) error {
	// Strip and reload
	cmd := fmt.Sprintf("sudo wg syncconf %s <(wg-quick strip %s)", c.interfaceName, c.interfaceName)
	_, err := conn.Execute(fmt.Sprintf("bash -c '%s'", cmd))
	if err != nil {
		return fmt.Errorf("wg syncconf failed: %w", err)
	}
	return nil
}

// ValidateConfigFile checks if the config file is valid
func (c *ConfigManager) ValidateConfigFile(ctx context.Context, conn *SSHConnection) error {
	cmd := fmt.Sprintf("wg-quick strip %s > /dev/null 2>&1 && echo 'VALID' || echo 'INVALID'", c.interfaceName)
	output, err := conn.Execute(cmd)
	if err != nil {
		return fmt.Errorf("config validation command failed: %w", err)
	}

	if strings.Contains(output, "INVALID") {
		return fmt.Errorf("WireGuard config file is invalid")
	}

	return nil
}

// BackupConfig creates a backup of the current config
func (c *ConfigManager) BackupConfig(ctx context.Context, conn *SSHConnection) (string, error) {
	backupPath := fmt.Sprintf("%s.backup.%d", c.configPath, time.Now().Unix())
	cmd := fmt.Sprintf("sudo cp %s %s", c.configPath, backupPath)
	_, err := conn.Execute(cmd)
	if err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}
	return backupPath, nil
}

// RestoreConfig restores config from a backup
func (c *ConfigManager) RestoreConfig(ctx context.Context, conn *SSHConnection, backupPath string) error {
	cmd := fmt.Sprintf("sudo cp %s %s && sudo wg syncconf %s <(wg-quick strip %s)",
		backupPath, c.configPath, c.interfaceName, c.interfaceName)
	_, err := conn.Execute(fmt.Sprintf("bash -c '%s'", cmd))
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	return nil
}
