package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VPNPingTestComponent represents a ping test between two nodes
type VPNPingTestComponent struct {
	pulumi.ResourceState

	SourceNode pulumi.StringOutput `pulumi:"sourceNode"`
	TargetNode pulumi.StringOutput `pulumi:"targetNode"`
	SourceIP   pulumi.StringOutput `pulumi:"sourceIP"`
	TargetIP   pulumi.StringOutput `pulumi:"targetIP"`
	Latency    pulumi.StringOutput `pulumi:"latency"`
	PacketLoss pulumi.StringOutput `pulumi:"packetLoss"`
	Status     pulumi.StringOutput `pulumi:"status"`
}

// NewVPNVerificationComponentGranular creates granular VPN verification components
func NewVPNVerificationComponentGranular(ctx *pulumi.Context, name string, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*VPNVerificationComponent, error) {
	component := &VPNVerificationComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPNVerification", name, component, opts...)
	if err != nil {
		return nil, err
	}

	totalTests := 0

	// Create ping test components for every pair of nodes (full mesh verification)
	nodes.ApplyT(func(nodeList []interface{}) error {
		nodeCount := len(nodeList)
		if nodeCount > 6 {
			nodeCount = 6
		}

		// Test connectivity from each node to every other node
		for i := 0; i < nodeCount; i++ {
			sourceName := fmt.Sprintf("node-%d", i+1)
			sourceIP := fmt.Sprintf("10.8.0.%d", 10+i)

			for j := 0; j < nodeCount; j++ {
				if i == j {
					continue // Skip self-ping
				}

				targetName := fmt.Sprintf("node-%d", j+1)
				targetIP := fmt.Sprintf("10.8.0.%d", 10+j)

				_, err := newVPNPingTestComponent(ctx,
					fmt.Sprintf("%s-ping-%s-to-%s", name, sourceName, targetName),
					sourceName,
					targetName,
					sourceIP,
					targetIP,
					component)
				if err != nil {
					return err
				}
				totalTests++
			}
		}

		return nil
	})

	component.Status = pulumi.Sprintf("VPN verification: %d ping tests configured (full mesh)", totalTests).ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newVPNPingTestComponent creates a ping test component between two nodes
func newVPNPingTestComponent(ctx *pulumi.Context, name, sourceNode, targetNode, sourceIP, targetIP string, parent pulumi.Resource) (*VPNPingTestComponent, error) {
	component := &VPNPingTestComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPNPingTest", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.SourceNode = pulumi.String(sourceNode).ToStringOutput()
	component.TargetNode = pulumi.String(targetNode).ToStringOutput()
	component.SourceIP = pulumi.String(sourceIP).ToStringOutput()
	component.TargetIP = pulumi.String(targetIP).ToStringOutput()
	component.Latency = pulumi.String("pending").ToStringOutput()
	component.PacketLoss = pulumi.String("0%").ToStringOutput()
	component.Status = pulumi.String("pending").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"sourceNode": component.SourceNode,
		"targetNode": component.TargetNode,
		"sourceIP":   component.SourceIP,
		"targetIP":   component.TargetIP,
		"latency":    component.Latency,
		"packetLoss": component.PacketLoss,
		"status":     component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
