package security

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// WireGuardManager manages WireGuard VPN configuration
type WireGuardManager struct {
	config     *config.WireGuardConfig
	nodes      []*providers.NodeOutput
	ctx        *pulumi.Context
	privateKey pulumi.StringOutput
	publicKey  pulumi.StringOutput
}

// NewWireGuardManager creates a new WireGuard manager
func NewWireGuardManager(ctx *pulumi.Context, config *config.WireGuardConfig) *WireGuardManager {
	return &WireGuardManager{
		ctx:    ctx,
		config: config,
		nodes:  make([]*providers.NodeOutput, 0),
	}
}

// ConfigureNode configures WireGuard on a node
func (w *WireGuardManager) ConfigureNode(node *providers.NodeOutput) error {
	if !w.config.Enabled {
		return nil
	}

	w.nodes = append(w.nodes, node)

	// Generate WireGuard configuration for the node
	configContent := w.generateNodeConfig(node)

	// Configure WireGuard on the node
	_, err := remote.NewCommand(w.ctx, fmt.Sprintf("%s-wg-config", node.Name), &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       node.PublicIP,
			Port:       pulumi.Float64(22),
			User:       pulumi.String(node.SSHUser),
			PrivateKey: pulumi.String(w.getSSHPrivateKey()),
		},
		Create: pulumi.String(fmt.Sprintf(`
#!/bin/bash
set -e

# Wait for cloud-init to complete
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
    echo "Waiting for cloud-init to finish..."
    sleep 5
done

# Create WireGuard config directory
mkdir -p /etc/wireguard
chmod 700 /etc/wireguard

# Write WireGuard configuration
cat > /etc/wireguard/wg0.conf << 'EOF'
%s
EOF

chmod 600 /etc/wireguard/wg0.conf

# Enable and start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# Verify WireGuard is running
wg show

echo "WireGuard configured successfully on %s"
`, configContent, node.Name)),
		Update: pulumi.String(fmt.Sprintf(`
#!/bin/bash
set -e

# Update WireGuard configuration
cat > /etc/wireguard/wg0.conf << 'EOF'
%s
EOF

chmod 600 /etc/wireguard/wg0.conf

# Restart WireGuard
systemctl restart wg-quick@wg0

# Verify WireGuard is running
wg show

echo "WireGuard updated successfully on %s"
`, configContent, node.Name)),
		Delete: pulumi.String(`
#!/bin/bash
systemctl stop wg-quick@wg0 || true
systemctl disable wg-quick@wg0 || true
rm -f /etc/wireguard/wg0.conf
echo "WireGuard removed"
`),
	}, pulumi.DependsOn([]pulumi.Resource{}))

	if err != nil {
		return fmt.Errorf("failed to configure WireGuard on %s: %w", node.Name, err)
	}

	return nil
}

// generateNodeConfig generates WireGuard configuration for a node
func (w *WireGuardManager) generateNodeConfig(node *providers.NodeOutput) string {
	// Generate or use existing private key for the node
	privateKey := w.generatePrivateKey(node)

	config := fmt.Sprintf(`[Interface]
# Node: %s
Address = %s/24
PrivateKey = %s
ListenPort = %d
MTU = %d

# Enable IP forwarding
PostUp = sysctl -w net.ipv4.ip_forward=1
PostUp = sysctl -w net.ipv6.conf.all.forwarding=1
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT
PostUp = iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT
PostDown = iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# DNS
DNS = %s

[Peer]
# WireGuard Server
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = %s
PersistentKeepalive = %d
`,
		node.Name,
		node.WireGuardIP,
		privateKey,
		w.config.Port,
		w.config.MTU,
		strings.Join(w.config.DNS, ", "),
		w.config.ServerPublicKey,
		w.config.ServerEndpoint,
		w.config.Port,
		strings.Join(w.config.AllowedIPs, ", "),
		w.config.PersistentKeepalive,
	)

	// Add peer configurations for mesh networking if enabled
	if w.config.MeshNetworking {
		config += w.generateMeshPeers(node)
	}

	return config
}

// generateMeshPeers generates peer configurations for mesh networking
func (w *WireGuardManager) generateMeshPeers(currentNode *providers.NodeOutput) string {
	peers := ""

	// Add all other nodes as peers for full mesh
	for _, node := range w.nodes {
		if node.Name == currentNode.Name {
			continue
		}

		// Note: In production, this would need to properly handle pulumi.StringOutput
		// For now, using a placeholder for the endpoint IP
		endpointIP := "PLACEHOLDER_IP" // TODO: Handle pulumi.StringOutput properly

		peers += fmt.Sprintf(`
[Peer]
# Node: %s
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s:%d
PersistentKeepalive = %d
`,
			node.Name,
			w.getNodePublicKey(node),
			node.WireGuardIP,
			endpointIP,
			w.config.Port,
			w.config.PersistentKeepalive,
		)
	}

	return peers
}

// generatePrivateKey generates a private key for a node
func (w *WireGuardManager) generatePrivateKey(node *providers.NodeOutput) string {
	// In production, this would generate a unique key per node
	// For now, we'll use a placeholder that would be replaced by actual key generation
	return fmt.Sprintf("GENERATED_PRIVATE_KEY_FOR_%s", node.Name)
}

// getNodePublicKey gets the public key for a node
func (w *WireGuardManager) getNodePublicKey(node *providers.NodeOutput) string {
	// In production, this would derive from the private key
	return fmt.Sprintf("GENERATED_PUBLIC_KEY_FOR_%s", node.Name)
}

// getSSHPrivateKey gets the SSH private key for connecting to nodes
func (w *WireGuardManager) getSSHPrivateKey() string {
	// This should be configured in the security config
	if w.config.SSHPrivateKeyPath != "" {
		// Read from file in production
		return "SSH_PRIVATE_KEY_CONTENT"
	}
	return ""
}

// ConfigureServerPeers configures peers on the WireGuard server
func (w *WireGuardManager) ConfigureServerPeers() error {
	if !w.config.Enabled || w.config.ServerEndpoint == "" {
		return nil
	}

	// Generate peer configurations for the server
	serverPeers := w.generateServerPeerConfig()

	// Connect to WireGuard server and add peers
	_, err := remote.NewCommand(w.ctx, "wg-server-peers", &remote.CommandArgs{
		Connection: &remote.ConnectionArgs{
			Host:       pulumi.String(w.config.ServerEndpoint),
			Port:       pulumi.Float64(22),
			User:       pulumi.String("root"),
			PrivateKey: pulumi.String(w.getSSHPrivateKey()),
		},
		Create: pulumi.String(fmt.Sprintf(`
#!/bin/bash
set -e

# Backup existing configuration
cp /etc/wireguard/wg0.conf /etc/wireguard/wg0.conf.backup

# Add peer configurations
cat >> /etc/wireguard/wg0.conf << 'EOF'

# Kubernetes Cluster Nodes
%s
EOF

# Reload WireGuard configuration
wg syncconf wg0 <(wg-quick strip wg0)

echo "Server peers configured successfully"
`, serverPeers)),
		Delete: pulumi.String(`
#!/bin/bash
# Restore backup configuration
if [ -f /etc/wireguard/wg0.conf.backup ]; then
    mv /etc/wireguard/wg0.conf.backup /etc/wireguard/wg0.conf
    wg syncconf wg0 <(wg-quick strip wg0)
fi
echo "Server peers removed"
`),
	})

	if err != nil {
		return fmt.Errorf("failed to configure server peers: %w", err)
	}

	return nil
}

// generateServerPeerConfig generates peer configuration for the WireGuard server
func (w *WireGuardManager) generateServerPeerConfig() string {
	config := ""

	for _, node := range w.nodes {
		config += fmt.Sprintf(`
[Peer]
# %s
PublicKey = %s
AllowedIPs = %s/32
`,
			node.Name,
			w.getNodePublicKey(node),
			node.WireGuardIP,
		)
	}

	return config
}

// ValidateConfiguration validates WireGuard configuration
func (w *WireGuardManager) ValidateConfiguration() error {
	if !w.config.Enabled {
		return nil
	}

	if w.config.ServerEndpoint == "" {
		return fmt.Errorf("WireGuard server endpoint is required")
	}

	if w.config.ServerPublicKey == "" {
		return fmt.Errorf("WireGuard server public key is required")
	}

	if len(w.config.AllowedIPs) == 0 {
		return fmt.Errorf("WireGuard allowed IPs must be specified")
	}

	return nil
}

// GenerateKeyPair generates a WireGuard key pair
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	// In production, this would use wg genkey and wg pubkey
	// For now, returning placeholder values
	privateKey = base64.StdEncoding.EncodeToString([]byte("private-key-placeholder"))
	publicKey = base64.StdEncoding.EncodeToString([]byte("public-key-placeholder"))
	return privateKey, publicKey, nil
}

// ExportWireGuardInfo exports WireGuard information to Pulumi stack
func (w *WireGuardManager) ExportWireGuardInfo() {
	secrets.Export(w.ctx, "wireguard_configured", pulumi.Bool(w.config.Enabled))

	if w.config.Enabled {
		secrets.Export(w.ctx, "wireguard_server_endpoint", pulumi.String(w.config.ServerEndpoint))
		secrets.Export(w.ctx, "wireguard_network", pulumi.String("10.8.0.0/24"))
		secrets.Export(w.ctx, "wireguard_port", pulumi.Int(w.config.Port))

		// Export node WireGuard IPs
		nodeIPs := pulumi.Map{}
		for _, node := range w.nodes {
			nodeIPs[node.Name] = pulumi.String(node.WireGuardIP)
		}
		secrets.Export(w.ctx, "wireguard_node_ips", nodeIPs)
	}
}

// GetNodeByWireGuardIP returns a node by its WireGuard IP
func (w *WireGuardManager) GetNodeByWireGuardIP(ip string) (*providers.NodeOutput, error) {
	for _, node := range w.nodes {
		if node.WireGuardIP == ip {
			return node, nil
		}
	}
	return nil, fmt.Errorf("node with WireGuard IP %s not found", ip)
}

// IsNodeReachable checks if a node is reachable via WireGuard
func (w *WireGuardManager) IsNodeReachable(node *providers.NodeOutput) pulumi.BoolOutput {
	return pulumi.All(node.PublicIP).ApplyT(func(args []interface{}) bool {
		// In production, this would actually test connectivity
		// For now, we assume nodes are reachable once configured
		return true
	}).(pulumi.BoolOutput)
}
