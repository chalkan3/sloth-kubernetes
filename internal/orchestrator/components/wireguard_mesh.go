package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// getSSHUserForProvider returns the correct SSH username for the given cloud provider
// Azure uses "azureuser", while other providers use "root" or "ubuntu"
func getSSHUserForWireGuard(provider pulumi.StringOutput) pulumi.StringOutput {
	return provider.ApplyT(func(p string) string {
		switch p {
		case "azure":
			return "azureuser"
		case "aws":
			return "ubuntu" // AWS Ubuntu AMIs use "ubuntu"
		case "gcp":
			return "ubuntu" // GCP uses "ubuntu" for Ubuntu images
		default:
			return "root" // DigitalOcean, Linode, and others use "root"
		}
	}).(pulumi.StringOutput)
}

// getSudoPrefixForUser returns "sudo " if user needs sudo, empty string if root
func getSudoPrefixForUser(provider pulumi.StringOutput) pulumi.StringOutput {
	return provider.ApplyT(func(p string) string {
		switch p {
		case "azure", "aws", "gcp":
			return "sudo " // Non-root users need sudo
		default:
			return "" // root doesn't need sudo
		}
	}).(pulumi.StringOutput)
}

// WireGuardMeshComponent configures full mesh WireGuard VPN
type WireGuardMeshComponent struct {
	pulumi.ResourceState

	Status      pulumi.StringOutput `pulumi:"status"`
	PeerCount   pulumi.IntOutput    `pulumi:"peerCount"`
	TunnelCount pulumi.IntOutput    `pulumi:"tunnelCount"`
}

// NewWireGuardMeshComponent sets up WireGuard mesh between nodes
// This configures a REAL full mesh VPN where every node connects to every other node
// If bastionComponent is provided, it's added to the mesh with VPN IP 10.8.0.5
func NewWireGuardMeshComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*WireGuardMeshComponent, error) {
	component := &WireGuardMeshComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:WireGuardMesh", name, component, opts...)
	if err != nil {
		return nil, err
	}

	peerCount := len(nodes)
	totalPeers := peerCount
	if bastionComponent != nil {
		totalPeers++ // Include bastion in peer count
		ctx.Log.Info("🏰 Including bastion host in WireGuard mesh (10.8.0.5)", nil)
	}
	tunnelCount := (totalPeers * (totalPeers - 1)) / 2

	ctx.Log.Info(fmt.Sprintf("🔧 Configuring WireGuard mesh: %d total peers (%d nodes + bastion), %d tunnels", totalPeers, peerCount, tunnelCount), nil)

	// STEP 1: Generate WireGuard keypairs on each node
	type nodeKeys struct {
		publicKey pulumi.StringOutput
		publicIP  pulumi.StringOutput
		wgIP      string
		name      string
	}

	allNodeKeys := make([]*nodeKeys, totalPeers)
	var keyGenCommands []pulumi.Resource

	// Generate keys for bastion if present
	if bastionComponent != nil {
		bastionWgIP := "10.8.0.5"
		keyCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-keygen-bastion", name), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           bastionComponent.PublicIP,
				User:           pulumi.String("root"),
				PrivateKey:     sshPrivateKey,
				DialErrorLimit: pulumi.Int(30),
			},
			Create: pulumi.String(`#!/bin/bash
set -e
umask 077
mkdir -p /etc/wireguard
cd /etc/wireguard
wg genkey | tee privatekey | wg pubkey > publickey
cat publickey
`),
		}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "10m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to generate keys for bastion: %v", err), nil)
		} else {
			keyGenCommands = append(keyGenCommands, keyCmd)

			publicKey := keyCmd.Stdout.ApplyT(func(s string) string {
				result := ""
				for _, c := range s {
					if c != '\n' && c != '\r' && c != ' ' && c != '\t' {
						result += string(c)
					}
				}
				return result
			}).(pulumi.StringOutput)

			allNodeKeys[0] = &nodeKeys{
				publicKey: publicKey,
				publicIP:  bastionComponent.PublicIP,
				wgIP:      bastionWgIP,
				name:      "bastion",
			}

			ctx.Log.Info("✅ Generated WireGuard keys on bastion", nil)
		}
	}

	// Generate keys for cluster nodes
	nodeOffset := 0
	if bastionComponent != nil {
		nodeOffset = 1 // Bastion is at index 0
	}

	for i, node := range nodes {
		wgIP := fmt.Sprintf("10.8.0.%d", 10+i)

		// Generate keys on each node
		// When bastion is present, use ProxyJump to connect through it
		// Use provider-specific SSH user (azureuser for Azure, root for others)
		sshUser := getSSHUserForWireGuard(node.Provider)
		sudoPrefix := getSudoPrefixForUser(node.Provider)

		connectionArgs := remote.ConnectionArgs{
			Host:           node.PublicIP,
			User:           sshUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}

		// Add ProxyJump when bastion is enabled
		if bastionComponent != nil {
			connectionArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       pulumi.String("root"),
				PrivateKey: sshPrivateKey,
			}
		}

		// Build keygen script with sudo if needed
		keygenScript := pulumi.All(sudoPrefix).ApplyT(func(args []interface{}) string {
			sudo := args[0].(string)
			if sudo != "" {
				// For non-root users, wrap entire script in sudo bash -c
				return fmt.Sprintf(`%sbash -c 'set -e && umask 077 && mkdir -p /etc/wireguard && wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey && cat /etc/wireguard/publickey'`, sudo)
			}
			// For root users, execute commands directly
			return `#!/bin/bash
set -e
umask 077
mkdir -p /etc/wireguard
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey
cat /etc/wireguard/publickey`
		}).(pulumi.StringOutput)

		keyCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-keygen-%d", name, i), &remote.CommandArgs{
			Connection: connectionArgs,
			Create:     keygenScript,
		}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "10m",
		}))
		if err != nil {
			return nil, fmt.Errorf("failed to generate keys for node %d: %w", i, err)
		}

		keyGenCommands = append(keyGenCommands, keyCmd)

		// The public key is in stdout (trimmed)
		publicKey := keyCmd.Stdout.ApplyT(func(s string) string {
			// Trim whitespace and newlines
			result := ""
			for _, c := range s {
				if c != '\n' && c != '\r' && c != ' ' && c != '\t' {
					result += string(c)
				}
			}
			return result
		}).(pulumi.StringOutput)

		allNodeKeys[nodeOffset+i] = &nodeKeys{
			publicKey: publicKey,
			publicIP:  node.PublicIP,
			wgIP:      wgIP,
			name:      fmt.Sprintf("node-%d", i),
		}

		ctx.Log.Info(fmt.Sprintf("✅ Generated WireGuard keys on node %d", i), nil)
	}

	// STEP 2: Configure WireGuard mesh on each node
	// Each node gets a config with ALL other peers (including bastion)

	// Configure bastion if present
	if bastionComponent != nil && allNodeKeys[0] != nil {
		myIdx := 0
		myWgIP := allNodeKeys[myIdx].wgIP

		peerConfigs := []pulumi.StringOutput{}

		// Add all nodes as peers to bastion
		for j := 1; j < totalPeers; j++ {
			peerKeys := allNodeKeys[j]

			peerConfig := pulumi.All(peerKeys.publicKey, peerKeys.publicIP).ApplyT(func(args []interface{}) string {
				pubKey := args[0].(string)
				peerIP := args[1].(string)
				peerWgIP := allNodeKeys[j].wgIP

				return fmt.Sprintf(`
[Peer]
# %s (%s)
PublicKey = %s
AllowedIPs = %s/32, 10.0.0.0/8
Endpoint = %s:51820
PersistentKeepalive = 25
`, allNodeKeys[j].name, peerWgIP, pubKey, peerWgIP, peerIP)
			}).(pulumi.StringOutput)

			peerConfigs = append(peerConfigs, peerConfig)
		}

		// Combine all peer configs
		var allPeerConfigsOutput pulumi.StringOutput
		if len(peerConfigs) > 0 {
			allPeerConfigsOutput = peerConfigs[0]
			for j := 1; j < len(peerConfigs); j++ {
				currentIdx := j
				allPeerConfigsOutput = pulumi.All(allPeerConfigsOutput, peerConfigs[currentIdx]).ApplyT(func(args []interface{}) string {
					existing := args[0].(string)
					newPeer := args[1].(string)
					return existing + newPeer
				}).(pulumi.StringOutput)
			}
		} else {
			allPeerConfigsOutput = pulumi.String("").ToStringOutput()
		}

		// Build complete WireGuard config for bastion
		fullConfig := allPeerConfigsOutput.ApplyT(func(peerSection string) string {
			interfaceSection := fmt.Sprintf(`[Interface]
Address = %s/24
ListenPort = 51820
PrivateKey = $(cat /etc/wireguard/privatekey)
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE; sysctl -w net.ipv4.ip_forward=1
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
`, myWgIP)

			return interfaceSection + peerSection
		}).(pulumi.StringOutput)

		// Deploy configuration to bastion
		deployScript := fullConfig.ApplyT(func(config string) string {
			return fmt.Sprintf(`#!/bin/bash
set -e

# Install WireGuard if not present
if ! command -v wg &> /dev/null; then
    apt-get update
    apt-get install -y wireguard-tools
fi

# Write WireGuard configuration
cat > /tmp/wg0.conf.template << 'WGEOF'
%s
WGEOF

# Expand the privatekey variable
export PRIVKEY=$(cat /etc/wireguard/privatekey)
sed "s|\$(cat /etc/wireguard/privatekey)|$PRIVKEY|g" /tmp/wg0.conf.template > /etc/wireguard/wg0.conf
chmod 600 /etc/wireguard/wg0.conf

# Enable IP forwarding permanently
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
sysctl -w net.ipv4.ip_forward=1

# Stop any existing WireGuard interface
wg-quick down wg0 2>/dev/null || true
sleep 2

# Start WireGuard
wg-quick up wg0

# Enable on boot
systemctl enable wg-quick@wg0 2>/dev/null || true

echo "✅ WireGuard mesh configured on bastion"
wg show
`, config)
		}).(pulumi.StringOutput)

		// Execute deployment on bastion
		_, err := remote.NewCommand(ctx, fmt.Sprintf("%s-deploy-bastion", name), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           bastionComponent.PublicIP,
				User:           pulumi.String("root"),
				PrivateKey:     sshPrivateKey,
				DialErrorLimit: pulumi.Int(30),
			},
			Create: deployScript,
		}, pulumi.Parent(component), pulumi.DependsOn(keyGenCommands), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "15m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to deploy WireGuard on bastion: %v", err), nil)
		} else {
			ctx.Log.Info("✅ WireGuard mesh configured on bastion", nil)
		}
	}

	// Configure cluster nodes
	for i, node := range nodes {
		myIdx := nodeOffset + i
		myWgIP := allNodeKeys[myIdx].wgIP

		// Get sudo prefix for this node (Azure/AWS/GCP need sudo, others don't)
		sudoPrefix := getSudoPrefixForUser(node.Provider)

		// Build peer list dynamically by combining all peer outputs (including bastion)
		peerConfigs := []pulumi.StringOutput{}

		for j := 0; j < totalPeers; j++ {
			if myIdx != j {
				peerKeys := allNodeKeys[j]

				// Build peer config section
				peerConfig := pulumi.All(peerKeys.publicKey, peerKeys.publicIP).ApplyT(func(args []interface{}) string {
					pubKey := args[0].(string)
					peerIP := args[1].(string)
					peerWgIP := allNodeKeys[j].wgIP
					peerName := allNodeKeys[j].name

					return fmt.Sprintf(`
[Peer]
# %s (%s)
PublicKey = %s
AllowedIPs = %s/32, 10.0.0.0/8
Endpoint = %s:51820
PersistentKeepalive = 25
`, peerName, peerWgIP, pubKey, peerWgIP, peerIP)
				}).(pulumi.StringOutput)

				peerConfigs = append(peerConfigs, peerConfig)
			}
		}

		// Combine all peer configs into final WireGuard configuration
		var allPeerConfigsOutput pulumi.StringOutput
		if len(peerConfigs) > 0 {
			// Merge all peer configs
			allPeerConfigsOutput = peerConfigs[0]
			for j := 1; j < len(peerConfigs); j++ {
				currentIdx := j
				allPeerConfigsOutput = pulumi.All(allPeerConfigsOutput, peerConfigs[currentIdx]).ApplyT(func(args []interface{}) string {
					existing := args[0].(string)
					newPeer := args[1].(string)
					return existing + newPeer
				}).(pulumi.StringOutput)
			}
		} else {
			allPeerConfigsOutput = pulumi.String("").ToStringOutput()
		}

		// Build complete WireGuard config with interface + all peers
		fullConfig := allPeerConfigsOutput.ApplyT(func(peerSection string) string {
			// Read the private key from the node
			interfaceSection := fmt.Sprintf(`[Interface]
Address = %s/24
ListenPort = 51820
PrivateKey = $(cat /etc/wireguard/privatekey)
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE; sysctl -w net.ipv4.ip_forward=1
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
`, myWgIP)

			return interfaceSection + peerSection
		}).(pulumi.StringOutput)

		// Deploy configuration to node - build script with sudo if needed
		deployScript := pulumi.All(fullConfig, sudoPrefix).ApplyT(func(args []interface{}) string {
			config := args[0].(string)
			sudo := args[1].(string)
			return fmt.Sprintf(`#!/bin/bash
set -e

# Write WireGuard configuration (expanding privatekey variable)
cat > /tmp/wg0.conf.template << 'WGEOF'
%s
WGEOF

# Expand the privatekey variable
export PRIVKEY=$(%scat /etc/wireguard/privatekey)
sed "s|\$(cat /etc/wireguard/privatekey)|$PRIVKEY|g" /tmp/wg0.conf.template | %stee /etc/wireguard/wg0.conf > /dev/null
%schmod 600 /etc/wireguard/wg0.conf

# Enable IP forwarding permanently
echo "net.ipv4.ip_forward=1" | %stee -a /etc/sysctl.conf > /dev/null
%ssysctl -w net.ipv4.ip_forward=1

# Stop any existing WireGuard interface
%swg-quick down wg0 2>/dev/null || true
sleep 2

# Start WireGuard
%swg-quick up wg0

# Enable on boot
%ssystemctl enable wg-quick@wg0 2>/dev/null || true

echo "✅ WireGuard mesh configured"
%swg show
`, config, sudo, sudo, sudo, sudo, sudo, sudo, sudo, sudo, sudo)
		}).(pulumi.StringOutput)

		// Execute deployment
		// When bastion is present, use ProxyJump to connect through it
		// Use provider-specific SSH user (azureuser for Azure, root for others)
		deploySSHUser := getSSHUserForWireGuard(node.Provider)

		deployConnectionArgs := remote.ConnectionArgs{
			Host:           node.PublicIP,
			User:           deploySSHUser,
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		}

		// Add ProxyJump when bastion is enabled
		if bastionComponent != nil {
			deployConnectionArgs.Proxy = &remote.ProxyConnectionArgs{
				Host:       bastionComponent.PublicIP,
				User:       pulumi.String("root"),
				PrivateKey: sshPrivateKey,
			}
		}

		_, err := remote.NewCommand(ctx, fmt.Sprintf("%s-deploy-%d", name, i), &remote.CommandArgs{
			Connection: deployConnectionArgs,
			Create:     deployScript,
		}, pulumi.Parent(component), pulumi.DependsOn(keyGenCommands), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "15m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to deploy WireGuard on node %d: %v", i, err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ WireGuard mesh configured on node %d", i), nil)
		}
	}

	statusMsg := fmt.Sprintf("WireGuard mesh: %d total peers, %d tunnels", totalPeers, tunnelCount)
	if bastionComponent != nil {
		statusMsg = fmt.Sprintf("WireGuard mesh: %d total peers (%d nodes + bastion), %d tunnels", totalPeers, peerCount, tunnelCount)
	}

	component.Status = pulumi.String(statusMsg).ToStringOutput()
	component.PeerCount = pulumi.Int(totalPeers).ToIntOutput()
	component.TunnelCount = pulumi.Int(tunnelCount).ToIntOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":      component.Status,
		"peerCount":   component.PeerCount,
		"tunnelCount": component.TunnelCount,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("✅ WireGuard mesh COMPLETE: %d total peers, %d tunnels", totalPeers, tunnelCount), nil)

	return component, nil
}
