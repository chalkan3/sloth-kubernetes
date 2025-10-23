package orchestrator

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
)

// HealthCheckComponent performs health checks on all nodes
type HealthCheckComponent struct {
	pulumi.ResourceState

	Status     pulumi.StringOutput `pulumi:"status"`
	NodeHealth pulumi.MapOutput    `pulumi:"nodeHealth"`
}

// NewHealthCheckComponent creates a new health check component with correct API usage
func NewHealthCheckComponent(ctx *pulumi.Context, name string, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*HealthCheckComponent, error) {
	component := &HealthCheckComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:health:HealthCheck", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// For now, create a simple status without remote commands
	// Remote commands will be added once nodes are actually created
	component.Status = pulumi.String("Health checks configured").ToStringOutput()

	// Create mock health results
	component.NodeHealth = nodes.ApplyT(func(nodes []interface{}) map[string]interface{} {
		healthResults := make(map[string]interface{})
		for _, n := range nodes {
			node := n.(*providers.NodeOutput)
			healthResults[node.Name] = map[string]interface{}{
				"status":  "pending",
				"healthy": false,
			}
		}
		return healthResults
	}).(pulumi.MapOutput)

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":     component.Status,
		"nodeHealth": component.NodeHealth,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
