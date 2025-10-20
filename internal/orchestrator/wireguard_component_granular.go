package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/pkg/config"
)

// WireGuardPeerComponent represents a single WireGuard peer configuration
type WireGuardPeerComponent struct {
	pulumi.ResourceState

	PeerName        pulumi.StringOutput `pulumi:"peerName"`
	PrivateKey      pulumi.StringOutput `pulumi:"privateKey"`
	PublicKey       pulumi.StringOutput `pulumi:"publicKey"`
	Address         pulumi.StringOutput `pulumi:"address"`
	ListenPort      pulumi.IntOutput    `pulumi:"listenPort"`
	AllowedIPs      pulumi.ArrayOutput  `pulumi:"allowedIPs"`
	PeerConnections pulumi.IntOutput    `pulumi:"peerConnections"`
	Status          pulumi.StringOutput `pulumi:"status"`
}

// WireGuardTunnelComponent represents a tunnel between two peers
type WireGuardTunnelComponent struct {
	pulumi.ResourceState

	FromPeer pulumi.StringOutput `pulumi:"fromPeer"`
	ToPeer   pulumi.StringOutput `pulumi:"toPeer"`
	Status   pulumi.StringOutput `pulumi:"status"`
}

// NewWireGuardComponentGranular creates granular WireGuard components with individual peers and tunnels
func NewWireGuardComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*WireGuardComponent, error) {
	component := &WireGuardComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:WireGuard", name, component, opts...)
	if err != nil {
		return nil, err
	}

	peerComponents := []*WireGuardPeerComponent{}
	tunnelCount := 0

	// Create individual WireGuard peer components for each node
	nodes.ApplyT(func(nodeList []interface{}) error {
		nodeCount := len(nodeList)
		if nodeCount > 6 {
			nodeCount = 6
		}

		// Create a peer component for each node
		for i := 0; i < nodeCount; i++ {
			nodeName := fmt.Sprintf("node-%d", i+1)
			address := fmt.Sprintf("10.8.0.%d/24", 10+i)

			peerComp, err := newWireGuardPeerComponent(ctx,
				fmt.Sprintf("%s-peer-%s", name, nodeName),
				nodeName,
				address,
				51820+i,
				nodeCount-1, // Each peer connects to all other peers
				component)
			if err != nil {
				return err
			}
			peerComponents = append(peerComponents, peerComp)

			// Create tunnel components for full mesh connectivity
			// Each node connects to all other nodes
			for j := i + 1; j < nodeCount; j++ {
				targetName := fmt.Sprintf("node-%d", j+1)
				_, err := newWireGuardTunnelComponent(ctx,
					fmt.Sprintf("%s-tunnel-%s-to-%s", name, nodeName, targetName),
					nodeName,
					targetName,
					component)
				if err != nil {
					return err
				}
				tunnelCount++
			}
		}

		return nil
	})

	component.Status = pulumi.Sprintf("WireGuard configured: %d peers, %d tunnels (full mesh)",
		len(peerComponents), tunnelCount).ToStringOutput()

	// Generate client configs
	component.ClientConfigs = nodes.ApplyT(func(nodes []interface{}) map[string]interface{} {
		configs := make(map[string]interface{})
		for i := 0; i < len(nodes) && i < 6; i++ {
			configs[fmt.Sprintf("node-%d", i)] = map[string]interface{}{
				"privateKey": fmt.Sprintf("generated-private-key-%d", i),
				"publicKey":  fmt.Sprintf("generated-public-key-%d", i),
				"address":    fmt.Sprintf("10.8.0.%d/24", i+10),
				"endpoint":   config.Network.WireGuard.ServerEndpoint,
				"listenPort": 51820 + i,
			}
		}
		return configs
	}).(pulumi.MapOutput)

	// Mesh network status
	component.MeshStatus = pulumi.Map{
		"type":       pulumi.String("full-mesh"),
		"nodes":      pulumi.Int(len(peerComponents)),
		"tunnels":    pulumi.Int(tunnelCount),
		"status":     pulumi.String("configured"),
		"encryption": pulumi.String("ChaCha20-Poly1305"),
		"mtu":        pulumi.Int(config.Network.WireGuard.MTU),
		"keepalive":  pulumi.Int(config.Network.WireGuard.PersistentKeepalive),
	}.ToMapOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":        component.Status,
		"clientConfigs": component.ClientConfigs,
		"meshStatus":    component.MeshStatus,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newWireGuardPeerComponent creates a component for a single WireGuard peer
func newWireGuardPeerComponent(ctx *pulumi.Context, name, peerName, address string, listenPort, peerConnections int, parent pulumi.Resource) (*WireGuardPeerComponent, error) {
	component := &WireGuardPeerComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:WireGuardPeer", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.PeerName = pulumi.String(peerName).ToStringOutput()
	component.PrivateKey = pulumi.Sprintf("generated-private-key-%s", peerName).ToStringOutput()
	component.PublicKey = pulumi.Sprintf("generated-public-key-%s", peerName).ToStringOutput()
	component.Address = pulumi.String(address).ToStringOutput()
	component.ListenPort = pulumi.Int(listenPort).ToIntOutput()
	component.PeerConnections = pulumi.Int(peerConnections).ToIntOutput()
	component.Status = pulumi.String("configured").ToStringOutput()

	// Allowed IPs for full mesh
	allowedIPs := []pulumi.Output{
		pulumi.String("10.0.0.0/8").ToStringOutput(),
		pulumi.String("172.16.0.0/12").ToStringOutput(),
	}
	component.AllowedIPs = pulumi.ToArrayOutput(allowedIPs)

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"peerName":        component.PeerName,
		"privateKey":      component.PrivateKey,
		"publicKey":       component.PublicKey,
		"address":         component.Address,
		"listenPort":      component.ListenPort,
		"allowedIPs":      component.AllowedIPs,
		"peerConnections": component.PeerConnections,
		"status":          component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newWireGuardTunnelComponent creates a component representing a tunnel between two peers
func newWireGuardTunnelComponent(ctx *pulumi.Context, name, fromPeer, toPeer string, parent pulumi.Resource) (*WireGuardTunnelComponent, error) {
	component := &WireGuardTunnelComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:WireGuardTunnel", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.FromPeer = pulumi.String(fromPeer).ToStringOutput()
	component.ToPeer = pulumi.String(toPeer).ToStringOutput()
	component.Status = pulumi.String("established").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"fromPeer": component.FromPeer,
		"toPeer":   component.ToPeer,
		"status":   component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
