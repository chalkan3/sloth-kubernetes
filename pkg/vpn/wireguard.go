package vpn

import (
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
)

// WireGuardManager handles WireGuard VPN server creation
type WireGuardManager struct {
	ctx *pulumi.Context
}

// NewWireGuardManager creates a new WireGuard manager
func NewWireGuardManager(ctx *pulumi.Context) *WireGuardManager {
	return &WireGuardManager{ctx: ctx}
}

// WireGuardResult contains created WireGuard server information
type WireGuardResult struct {
	Provider   string
	ServerID   pulumi.IDOutput
	ServerIP   pulumi.StringOutput
	ServerName string
	PublicKey  string
	PrivateKey string
	Port       int
	SubnetCIDR string
}

// CreateWireGuardServer creates a WireGuard VPN server
func (m *WireGuardManager) CreateWireGuardServer(cfg *config.WireGuardConfig, sshKey pulumi.StringOutput) (*WireGuardResult, error) {
	if !cfg.Create {
		return nil, nil // Not creating a server
	}

	// Default values
	if cfg.Port == 0 {
		cfg.Port = 51820
	}
	if cfg.SubnetCIDR == "" {
		cfg.SubnetCIDR = "10.8.0.0/24"
	}
	if cfg.Image == "" {
		cfg.Image = "ubuntu-22-04-x64"
	}
	if cfg.Name == "" {
		cfg.Name = "wireguard-vpn"
	}

	// WireGuard installation script
	installScript := pulumi.Sprintf(`#!/bin/bash
set -e

# Update system
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y

# Install WireGuard
apt-get install -y wireguard wireguard-tools

# Generate keys
umask 077
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey

# Configure WireGuard
cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
Address = %s.1/24
ListenPort = %d
PrivateKey = $(cat /etc/wireguard/privatekey)
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# Peers will be added dynamically
EOF

# Enable IP forwarding
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf
sysctl -p

# Start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# Save public key to accessible location
cp /etc/wireguard/publickey /root/wireguard-publickey

echo "WireGuard server installed successfully!"
`, cfg.ClientIPBase, cfg.Port)

	switch cfg.Provider {
	case "digitalocean":
		return m.createDigitalOceanWireGuard(cfg, sshKey, installScript)
	case "linode":
		return m.createLinodeWireGuard(cfg, sshKey, installScript)
	default:
		return nil, fmt.Errorf("unsupported provider for WireGuard: %s", cfg.Provider)
	}
}

func (m *WireGuardManager) createDigitalOceanWireGuard(cfg *config.WireGuardConfig, sshKey pulumi.StringOutput, installScript pulumi.StringOutput) (*WireGuardResult, error) {
	// Get SSH key ID
	sshKeyID := sshKey.ApplyT(func(key string) (string, error) {
		// The key should already be registered, we need to look it up
		return key, nil
	}).(pulumi.StringOutput)

	// Create WireGuard server droplet
	droplet, err := digitalocean.NewDroplet(m.ctx, cfg.Name, &digitalocean.DropletArgs{
		Name:       pulumi.String(cfg.Name),
		Region:     pulumi.String(cfg.Region),
		Size:       pulumi.String(cfg.Size),
		Image:      pulumi.String(cfg.Image),
		SshKeys:    pulumi.StringArray{sshKeyID},
		UserData:   installScript,
		Tags:       pulumi.StringArray{pulumi.String("wireguard"), pulumi.String("vpn")},
		Monitoring: pulumi.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WireGuard server: %w", err)
	}

	// Export server information
	secrets.Export(m.ctx, "wireguard_server_id", droplet.ID())
	secrets.Export(m.ctx, "wireguard_server_ip", droplet.Ipv4Address)
	secrets.Export(m.ctx, "wireguard_server_name", droplet.Name)
	secrets.Export(m.ctx, "wireguard_port", pulumi.Int(cfg.Port))

	return &WireGuardResult{
		Provider:   "digitalocean",
		ServerID:   droplet.ID(),
		ServerIP:   droplet.Ipv4Address,
		ServerName: cfg.Name,
		Port:       cfg.Port,
		SubnetCIDR: cfg.SubnetCIDR,
	}, nil
}

func (m *WireGuardManager) createLinodeWireGuard(cfg *config.WireGuardConfig, sshKey pulumi.StringOutput, installScript pulumi.StringOutput) (*WireGuardResult, error) {
	// Convert image name for Linode
	image := cfg.Image
	if image == "ubuntu-22-04-x64" {
		image = "linode/ubuntu22.04"
	}

	// Use the SSH key directly - it's already normalized in sshkeys.go
	// The key is in format: "ssh-rsa AAAAB3..." (type + key-data only, no comment)

	// Create WireGuard server instance
	instance, err := linode.NewInstance(m.ctx, cfg.Name, &linode.InstanceArgs{
		Label:          pulumi.String(cfg.Name),
		Region:         pulumi.String(cfg.Region),
		Type:           pulumi.String(cfg.Size),
		Image:          pulumi.String(image),
		AuthorizedKeys: pulumi.StringArray{sshKey},
		RootPass:       pulumi.String("TempPass123!ChangeMe"), // Will be changed via SSH key
		Tags:           pulumi.StringArray{pulumi.String("wireguard"), pulumi.String("vpn")},
		PrivateIp:      pulumi.Bool(true),
		// User data will be applied via cloud-init if supported
		StackscriptData: pulumi.StringMap{
			"user_data": installScript,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WireGuard server: %w", err)
	}

	// Export server information
	secrets.Export(m.ctx, "wireguard_server_id", instance.ID())
	secrets.Export(m.ctx, "wireguard_server_ip", instance.IpAddress)
	secrets.Export(m.ctx, "wireguard_server_name", instance.Label)
	secrets.Export(m.ctx, "wireguard_port", pulumi.Int(cfg.Port))

	return &WireGuardResult{
		Provider:   "linode",
		ServerID:   instance.ID(),
		ServerIP:   instance.IpAddress,
		ServerName: cfg.Name,
		Port:       cfg.Port,
		SubnetCIDR: cfg.SubnetCIDR,
	}, nil
}

// ConfigureWireGuardClient generates WireGuard client configuration
func (m *WireGuardManager) ConfigureWireGuardClient(serverIP string, serverPort int, clientIP string) string {
	return fmt.Sprintf(`[Interface]
Address = %s/24
PrivateKey = <CLIENT_PRIVATE_KEY>

[Peer]
PublicKey = <SERVER_PUBLIC_KEY>
Endpoint = %s:%d
AllowedIPs = 10.8.0.0/24, 10.10.0.0/16, 10.11.0.0/16
PersistentKeepalive = 25
`, clientIP, serverIP, serverPort)
}

// GetWireGuardInstallCommand returns the command to add a peer to the WireGuard server
func (m *WireGuardManager) GetWireGuardInstallCommand(peerName, peerPublicKey, peerIP string) string {
	return fmt.Sprintf(`
# Add peer %s to WireGuard server
wg set wg0 peer %s allowed-ips %s/32

# Persist configuration
wg-quick save wg0

# Restart WireGuard
systemctl restart wg-quick@wg0
`, peerName, peerPublicKey, peerIP)
}
