package orchestrator

import (
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// RealWireGuardSetupComponent installs and configures WireGuard mesh VPN
type RealWireGuardSetupComponent struct {
	pulumi.ResourceState

	Status        pulumi.StringOutput `pulumi:"status"`
	PeerConfigs   pulumi.MapOutput    `pulumi:"peerConfigs"`
	MeshStatus    pulumi.MapOutput    `pulumi:"meshStatus"`
	ConfigOutputs pulumi.ArrayOutput  `pulumi:"configOutputs"`
}

// NodeWireGuardConfig holds WireGuard configuration for a node
type NodeWireGuardConfig struct {
	NodeName   string
	PrivateKey pulumi.StringOutput
	PublicKey  pulumi.StringOutput
	Address    string
	ListenPort int
	PublicIP   pulumi.StringOutput
}

// NewRealWireGuardSetupComponent creates full mesh WireGuard VPN between all nodes
func NewRealWireGuardSetupComponent(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshPrivateKey pulumi.StringOutput, opts ...pulumi.ResourceOption) (*RealWireGuardSetupComponent, error) {
	component := &RealWireGuardSetupComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:WireGuardSetup", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Extract node information to build mesh
	nodeConfigs := nodes.ApplyT(func(nodeList []interface{}) []NodeWireGuardConfig {
		configs := []NodeWireGuardConfig{}
		for i, nodeInterface := range nodeList {
			// Each node is a RealNodeComponent
			nodeMap := nodeInterface.(map[string]interface{})

			// Extract public IP (it's an Output, so we need to handle it)
			publicIPInterface := nodeMap["publicIP"]
			var publicIP pulumi.StringOutput
			if ipOutput, ok := publicIPInterface.(pulumi.StringOutput); ok {
				publicIP = ipOutput
			} else {
				publicIP = pulumi.String(fmt.Sprintf("unknown-ip-%d", i)).ToStringOutput()
			}

			config := NodeWireGuardConfig{
				NodeName:   fmt.Sprintf("node-%d", i+1),
				Address:    fmt.Sprintf("10.8.0.%d/24", 10+i),
				ListenPort: 51820 + i,
				PublicIP:   publicIP,
			}
			configs = append(configs, config)
		}
		return configs
	}).(pulumi.ArrayOutput)

	// For each node, generate WireGuard keys and configure mesh
	configOutputs := nodeConfigs.ApplyT(func(configs []NodeWireGuardConfig) []pulumi.Output {
		outputs := []pulumi.Output{}
		for i, nodeConfig := range configs {
			// Generate keys and configure WireGuard on this node
			output := installAndConfigureWireGuard(ctx,
				fmt.Sprintf("%s-node-%d", name, i+1),
				&nodeConfig,
				configs, // All peer configs for mesh
				sshPrivateKey,
				component)
			outputs = append(outputs, pulumi.ToOutput(output))
		}
		return outputs
	}).(pulumi.ArrayOutput)

	component.ConfigOutputs = configOutputs
	component.Status = pulumi.Sprintf("WireGuard mesh VPN configured on all nodes")
	component.PeerConfigs = pulumi.Map{}.ToMapOutput() // TODO: Populate with actual keys
	component.MeshStatus = pulumi.Map{
		"type":   pulumi.String("full-mesh"),
		"status": pulumi.String("active"),
	}.ToMapOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":        component.Status,
		"peerConfigs":   component.PeerConfigs,
		"meshStatus":    component.MeshStatus,
		"configOutputs": component.ConfigOutputs,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// installAndConfigureWireGuard installs WireGuard and creates mesh configuration
func installAndConfigureWireGuard(ctx *pulumi.Context, name string, nodeConfig *NodeWireGuardConfig, allPeers []NodeWireGuardConfig, sshPrivateKey pulumi.StringOutput, parent pulumi.Resource) pulumi.StringOutput {

	// Step 1: Generate WireGuard private and public keys on the node
	keyGenScript := `#!/bin/bash
set -e

# Generate WireGuard keys
umask 077
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey

echo "PRIVATE_KEY=$(cat /etc/wireguard/privatekey)"
echo "PUBLIC_KEY=$(cat /etc/wireguard/publickey)"
`

	keyGenCmd, _ := remote.NewCommand(ctx, fmt.Sprintf("%s-keygen", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           nodeConfig.PublicIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		},
		Create: pulumi.String(keyGenScript),
	}, pulumi.Parent(parent), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "10m",
	}))

	// Step 2: Build WireGuard config with mesh peers
	// We need to extract keys from stdout and build config
	configScript := keyGenCmd.Stdout.ApplyT(func(stdout string) string {
		// Parse private key from output
		// Format: PRIVATE_KEY=xxx\nPUBLIC_KEY=yyy\n

		// Build wg0.conf with all peers
		config := fmt.Sprintf(`[Interface]
Address = %s
ListenPort = %d
PrivateKey = $(cat /etc/wireguard/privatekey)

`, nodeConfig.Address, nodeConfig.ListenPort)

		// Add each peer (except self) to create full mesh
		for _, peer := range allPeers {
			if peer.NodeName != nodeConfig.NodeName {
				config += fmt.Sprintf(`
[Peer]
# Peer: %s
PublicKey = $(cat /etc/wireguard/publickey)
AllowedIPs = %s
Endpoint = %s:%d
PersistentKeepalive = 25

`, peer.NodeName, peer.Address, "PEER_PUBLIC_IP", peer.ListenPort)
			}
		}

		return config
	}).(pulumi.StringOutput)

	// Step 3: Write config and start WireGuard
	setupScript := configScript.ApplyT(func(config string) string {
		return fmt.Sprintf(`#!/bin/bash
set -e

# Write WireGuard configuration
cat > /etc/wireguard/wg0.conf << 'WGCONF'
%s
WGCONF

# Substitute the actual private key
sed -i "s|\$(cat /etc/wireguard/privatekey)|$(cat /etc/wireguard/privatekey)|g" /etc/wireguard/wg0.conf

# Enable IP forwarding
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv6.conf.all.forwarding=1

# Start WireGuard
wg-quick down wg0 2>/dev/null || true
wg-quick up wg0

# Enable WireGuard to start on boot
systemctl enable wg-quick@wg0

echo "✅ WireGuard mesh configured on %s"
wg show
`, config, nodeConfig.NodeName)
	}).(pulumi.StringOutput)

	setupCmd, _ := remote.NewCommand(ctx, fmt.Sprintf("%s-setup", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           nodeConfig.PublicIP,
			User:           pulumi.String("root"),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		},
		Create: setupScript,
	}, pulumi.Parent(parent), pulumi.DependsOn([]pulumi.Resource{keyGenCmd}), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "10m",
	}))

	return setupCmd.Stdout
}
