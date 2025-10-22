package components

import (
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// VPCComponent manages VPC creation for private networking
type VPCComponent struct {
	pulumi.ResourceState

	VPCID   pulumi.StringOutput `pulumi:"vpcId"`
	VPCName pulumi.StringOutput `pulumi:"vpcName"`
	Region  pulumi.StringOutput `pulumi:"region"`
	IPRange pulumi.StringOutput `pulumi:"ipRange"`
}

// NewVPCComponent creates a VPC for private networking
// When bastion is enabled, this VPC will be used for all cluster nodes
func NewVPCComponent(
	ctx *pulumi.Context,
	name string,
	region string,
	ipRange string,
	opts ...pulumi.ResourceOption,
) (*VPCComponent, error) {
	component := &VPCComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPC", name, component, opts...)
	if err != nil {
		return nil, err
	}

	ctx.Log.Info(fmt.Sprintf("🌐 Creating VPC for private networking in %s...", region), nil)

	// Create VPC for private networking
	vpc, err := digitalocean.NewVpc(ctx, name, &digitalocean.VpcArgs{
		Name:        pulumi.String(fmt.Sprintf("kubernetes-vpc-%s", ctx.Stack())),
		Region:      pulumi.String(region),
		IpRange:     pulumi.String(ipRange),
		Description: pulumi.String("Private VPC for Kubernetes cluster nodes"),
	}, pulumi.Parent(component))
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}

	component.VPCID = vpc.ID().ToStringOutput()
	component.VPCName = vpc.Name
	component.Region = pulumi.String(region).ToStringOutput()
	component.IPRange = vpc.IpRange

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"vpcId":   component.VPCID,
		"vpcName": component.VPCName,
		"region":  component.Region,
		"ipRange": component.IPRange,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ VPC created for private networking", nil)

	return component, nil
}
