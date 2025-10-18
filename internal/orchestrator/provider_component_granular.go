package orchestrator

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/config"
)

// DigitalOceanProviderComponent represents DigitalOcean provider initialization
type DigitalOceanProviderComponent struct {
	pulumi.ResourceState

	Token      pulumi.StringOutput `pulumi:"token"`
	Region     pulumi.StringOutput `pulumi:"region"`
	VPCConfig  pulumi.MapOutput    `pulumi:"vpcConfig"`
	SSHKeys    pulumi.ArrayOutput  `pulumi:"sshKeys"`
	Status     pulumi.StringOutput `pulumi:"status"`
}

// LinodeProviderComponent represents Linode provider initialization
type LinodeProviderComponent struct {
	pulumi.ResourceState

	Token        pulumi.StringOutput `pulumi:"token"`
	Region       pulumi.StringOutput `pulumi:"region"`
	VPCConfig    pulumi.MapOutput    `pulumi:"vpcConfig"`
	RootPassword pulumi.StringOutput `pulumi:"rootPassword"`
	Status       pulumi.StringOutput `pulumi:"status"`
}

// NewProviderComponentGranular creates granular provider components
func NewProviderComponentGranular(ctx *pulumi.Context, name string, config *config.ClusterConfig, sshKey pulumi.StringOutput, opts ...pulumi.ResourceOption) (*ProviderComponent, error) {
	component := &ProviderComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provider:Providers", name, component, opts...)
	if err != nil {
		return nil, err
	}

	providersMap := pulumi.Map{}

	// Create DigitalOcean provider component
	if config.Providers.DigitalOcean != nil && config.Providers.DigitalOcean.Enabled {
		doProvider, err := newDigitalOceanProviderComponent(ctx,
			fmt.Sprintf("%s-digitalocean", name),
			config.Providers.DigitalOcean,
			sshKey,
			component)
		if err != nil {
			return nil, err
		}
		providersMap["digitalocean"] = pulumi.String("initialized")

		// Create VPC component for DigitalOcean
		if config.Providers.DigitalOcean.VPC != nil {
			_, err = newVPCComponent(ctx,
				fmt.Sprintf("%s-digitalocean-vpc", name),
				"digitalocean",
				config.Providers.DigitalOcean.VPC,
				doProvider)
			if err != nil {
				return nil, err
			}
		}
	}

	// Create Linode provider component
	if config.Providers.Linode != nil && config.Providers.Linode.Enabled {
		linodeProvider, err := newLinodeProviderComponent(ctx,
			fmt.Sprintf("%s-linode", name),
			config.Providers.Linode,
			sshKey,
			component)
		if err != nil {
			return nil, err
		}
		providersMap["linode"] = pulumi.String("initialized")

		// Create VPC component for Linode
		if config.Providers.Linode.VPC != nil {
			_, err = newVPCComponent(ctx,
				fmt.Sprintf("%s-linode-vpc", name),
				"linode",
				config.Providers.Linode.VPC,
				linodeProvider)
			if err != nil {
				return nil, err
			}
		}
	}

	component.Providers = providersMap.ToMapOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"providers": component.Providers,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newDigitalOceanProviderComponent creates a DigitalOcean provider component
func newDigitalOceanProviderComponent(ctx *pulumi.Context, name string, providerConfig *config.DigitalOceanProvider, sshKey pulumi.StringOutput, parent pulumi.Resource) (*DigitalOceanProviderComponent, error) {
	component := &DigitalOceanProviderComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provider:DigitalOcean", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Token = pulumi.String("***REDACTED***").ToStringOutput()
	component.Region = pulumi.String(providerConfig.Region).ToStringOutput()
	component.Status = pulumi.String("initialized").ToStringOutput()

	// SSH Keys
	sshKeysArray := make([]pulumi.Output, len(providerConfig.SSHKeys)+1)
	for i, key := range providerConfig.SSHKeys {
		sshKeysArray[i] = pulumi.String(key).ToStringOutput()
	}
	sshKeysArray[len(providerConfig.SSHKeys)] = sshKey
	component.SSHKeys = pulumi.ToArrayOutput(sshKeysArray)

	// VPC Config
	if providerConfig.VPC != nil {
		component.VPCConfig = pulumi.Map{
			"name":    pulumi.String(providerConfig.VPC.Name),
			"cidr":    pulumi.String(providerConfig.VPC.CIDR),
			"region":  pulumi.String(providerConfig.VPC.Region),
			"private": pulumi.Bool(providerConfig.VPC.Private),
		}.ToMapOutput()
	} else {
		component.VPCConfig = pulumi.Map{}.ToMapOutput()
	}

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"token":     component.Token,
		"region":    component.Region,
		"vpcConfig": component.VPCConfig,
		"sshKeys":   component.SSHKeys,
		"status":    component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// newLinodeProviderComponent creates a Linode provider component
func newLinodeProviderComponent(ctx *pulumi.Context, name string, providerConfig *config.LinodeProvider, sshKey pulumi.StringOutput, parent pulumi.Resource) (*LinodeProviderComponent, error) {
	component := &LinodeProviderComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:provider:Linode", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Token = pulumi.String("***REDACTED***").ToStringOutput()
	component.Region = pulumi.String(providerConfig.Region).ToStringOutput()
	component.RootPassword = pulumi.String("***REDACTED***").ToStringOutput()
	component.Status = pulumi.String("initialized").ToStringOutput()

	// VPC Config
	if providerConfig.VPC != nil {
		component.VPCConfig = pulumi.Map{
			"name":   pulumi.String(providerConfig.VPC.Name),
			"cidr":   pulumi.String(providerConfig.VPC.CIDR),
			"region": pulumi.String(providerConfig.VPC.Region),
		}.ToMapOutput()
	} else {
		component.VPCConfig = pulumi.Map{}.ToMapOutput()
	}

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"token":        component.Token,
		"region":       component.Region,
		"vpcConfig":    component.VPCConfig,
		"rootPassword": component.RootPassword,
		"status":       component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// VPCComponent represents a VPC configuration
type VPCComponent struct {
	pulumi.ResourceState

	Provider pulumi.StringOutput `pulumi:"provider"`
	Name     pulumi.StringOutput `pulumi:"name"`
	CIDR     pulumi.StringOutput `pulumi:"cidr"`
	Region   pulumi.StringOutput `pulumi:"region"`
	Status   pulumi.StringOutput `pulumi:"status"`
}

// newVPCComponent creates a VPC component
func newVPCComponent(ctx *pulumi.Context, name, provider string, vpcConfig *config.VPCConfig, parent pulumi.Resource) (*VPCComponent, error) {
	component := &VPCComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:network:VPC", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.Provider = pulumi.String(provider).ToStringOutput()
	component.Name = pulumi.String(vpcConfig.Name).ToStringOutput()
	component.CIDR = pulumi.String(vpcConfig.CIDR).ToStringOutput()
	component.Region = pulumi.String(vpcConfig.Region).ToStringOutput()
	component.Status = pulumi.String("configured").ToStringOutput()

	// Register outputs
	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"provider": component.Provider,
		"name":     component.Name,
		"cidr":     component.CIDR,
		"region":   component.Region,
		"status":   component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}
