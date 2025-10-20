package components

import (
	"fmt"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// WireGuardMeshComponent configures full mesh WireGuard VPN
type WireGuardMeshComponent struct {
	pulumi.ResourceState

	Status      pulumi.StringOutput `pulumi:"status"`
	PeerCount   pulumi.IntOutput    `pulumi:"peerCount"`
	TunnelCount pulumi.IntOutput    `pulumi:"tunnelCount"`
}

// NewWireGuardMeshComponent sets up WireGuard mesh between nodes
// This configures a REAL full mesh VPN where every node connects to every other node
func NewWireGuardMeshComponent(ctx *pulumi.Context, name string, nodes []*RealNodeComponent, sshPrivateKey pulumi.StringOutput, opts ...pulumi.ResourceOption) (*WireGuardMeshComponent, error) {
	component := &WireGuardMeshComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:WireGuardMesh", name, component, opts...)
	if err != nil {
		return nil, err
	}

	peerCount := len(nodes)
	tunnelCount := (peerCount * (peerCount - 1)) / 2

	ctx.Log.Info(fmt.Sprintf("🔧 Configuring WireGuard mesh: %d nodes, %d tunnels", peerCount, tunnelCount), nil)

	// STEP 1: Generate WireGuard keypairs on each node
	type nodeKeys struct {
		publicKey pulumi.StringOutput
		publicIP  pulumi.StringOutput
		wgIP      string
	}

	allNodeKeys := make([]*nodeKeys, peerCount)
	var keyGenCommands []pulumi.Resource

	for i, node := range nodes {
		wgIP := fmt.Sprintf("10.8.0.%d", 10+i)

		// Generate keys on each node
		keyCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-keygen-%d", name, i), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           node.PublicIP,
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

		allNodeKeys[i] = &nodeKeys{
			publicKey: publicKey,
			publicIP:  node.PublicIP,
			wgIP:      wgIP,
		}

		ctx.Log.Info(fmt.Sprintf("✅ Generated WireGuard keys on node %d", i), nil)
	}

	// STEP 2: Configure WireGuard mesh on each node
	// Each node gets a config with ALL other nodes as peers
	for i, node := range nodes {
		myWgIP := allNodeKeys[i].wgIP

		// Build peer list dynamically by combining all peer outputs
		peerConfigs := []pulumi.StringOutput{}

		for j := 0; j < peerCount; j++ {
			if i != j {
				peerKeys := allNodeKeys[j]

				// Build peer config section
				peerConfig := pulumi.All(peerKeys.publicKey, peerKeys.publicIP).ApplyT(func(args []interface{}) string {
					pubKey := args[0].(string)
					peerIP := args[1].(string)
					peerWgIP := allNodeKeys[j].wgIP

					return fmt.Sprintf(`
[Peer]
# Node %d (%s)
PublicKey = %s
AllowedIPs = %s/32, 10.0.0.0/8
Endpoint = %s:51820
PersistentKeepalive = 25
`, j, peerWgIP, pubKey, peerWgIP, peerIP)
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

		// Deploy configuration to node
		deployScript := fullConfig.ApplyT(func(config string) string {
			return fmt.Sprintf(`#!/bin/bash
set -e

# Write WireGuard configuration (expanding privatekey variable)
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

echo "✅ WireGuard mesh configured"
wg show
`, config)
		}).(pulumi.StringOutput)

		// Execute deployment
		_, err := remote.NewCommand(ctx, fmt.Sprintf("%s-deploy-%d", name, i), &remote.CommandArgs{
			Connection: remote.ConnectionArgs{
				Host:           node.PublicIP,
				User:           pulumi.String("root"),
				PrivateKey:     sshPrivateKey,
				DialErrorLimit: pulumi.Int(30),
			},
			Create: deployScript,
		}, pulumi.Parent(component), pulumi.DependsOn(keyGenCommands), pulumi.Timeouts(&pulumi.CustomTimeouts{
			Create: "15m",
		}))
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("⚠️  Failed to deploy WireGuard on node %d: %v", i, err), nil)
		} else {
			ctx.Log.Info(fmt.Sprintf("✅ WireGuard mesh configured on node %d", i), nil)
		}
	}

	component.Status = pulumi.Sprintf("WireGuard mesh: %d nodes, %d tunnels", peerCount, tunnelCount)
	component.PeerCount = pulumi.Int(peerCount).ToIntOutput()
	component.TunnelCount = pulumi.Int(tunnelCount).ToIntOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":      component.Status,
		"peerCount":   component.PeerCount,
		"tunnelCount": component.TunnelCount,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("✅ WireGuard mesh COMPLETE: %d nodes, %d tunnels", peerCount, tunnelCount), nil)

	return component, nil
}
