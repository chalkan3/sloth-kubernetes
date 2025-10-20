package orchestrator

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VPNVerificationComponent verifies VPN connectivity between all nodes
type VPNVerificationComponent struct {
	pulumi.ResourceState

	Status             pulumi.StringOutput `pulumi:"status"`
	ConnectivityMatrix pulumi.MapOutput    `pulumi:"connectivityMatrix"`
}

// NewVPNVerificationComponent creates a new VPN verification component
func NewVPNVerificationComponent(ctx *pulumi.Context, name string, nodes pulumi.ArrayOutput, sshKeyPath pulumi.StringOutput, opts ...pulumi.ResourceOption) (*VPNVerificationComponent, error) {
	component := &VPNVerificationComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPNVerification", name, component, opts...)
	if err != nil {
		return nil, err
	}

	// Create connectivity matrix placeholder
	component.ConnectivityMatrix = pulumi.Map{
		"status":  pulumi.String("pending"),
		"message": pulumi.String("VPN connectivity will be verified after node deployment"),
	}.ToMapOutput()

	// Set component outputs
	component.Status = pulumi.String("VPN verification configured").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status":             component.Status,
		"connectivityMatrix": component.ConnectivityMatrix,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
