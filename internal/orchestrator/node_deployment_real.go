package orchestrator

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"kubernetes-create/pkg/config"
)

// RealNodeComponent represents a real cloud instance (Droplet or Linode)
type RealNodeComponent struct {
	pulumi.ResourceState

	NodeName    pulumi.StringOutput    `pulumi:"nodeName"`
	Provider    pulumi.StringOutput    `pulumi:"provider"`
	Region      pulumi.StringOutput    `pulumi:"region"`
	Size        pulumi.StringOutput    `pulumi:"size"`
	PublicIP    pulumi.StringOutput    `pulumi:"publicIP"`
	PrivateIP   pulumi.StringOutput    `pulumi:"privateIP"`
	WireGuardIP pulumi.StringOutput    `pulumi:"wireGuardIP"`
	Roles       pulumi.ArrayOutput     `pulumi:"roles"`
	Status      pulumi.StringOutput    `pulumi:"status"`
	DropletID   pulumi.IDOutput        `pulumi:"dropletId"`    // For DigitalOcean
	InstanceID  pulumi.IntOutput       `pulumi:"instanceId"`   // For Linode
}

// NewRealNodeDeploymentComponent creates real cloud resources
// Returns NodeDeploymentComponent and list of RealNodeComponents for WireGuard/RKE
func NewRealNodeDeploymentComponent(ctx *pulumi.Context, name string, clusterConfig *config.ClusterConfig, sshKeyOutput pulumi.StringOutput, sshPrivateKey pulumi.StringOutput, doToken pulumi.StringInput, linodeToken pulumi.StringInput, opts ...pulumi.ResourceOption) (*NodeDeploymentComponent, []*RealNodeComponent, error) {
	component := &NodeDeploymentComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:compute:NodeDeployment", name, component, opts...)
	if err != nil {
		return nil, nil, err
	}

	// Create ONE shared SSH key for all DigitalOcean Droplets (DO doesn't allow duplicate keys)
	sharedDOSshKey, err := digitalocean.NewSshKey(ctx, fmt.Sprintf("%s-shared-key", name), &digitalocean.SshKeyArgs{
		Name:      pulumi.Sprintf("kubernetes-cluster-production-key"),
		PublicKey: sshKeyOutput,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create shared DO SSH key: %w", err)
	}

	realNodeComponents := []*RealNodeComponent{}
	nodesArray := []pulumi.Output{}

	// Create individual nodes
	for _, nodeConfig := range clusterConfig.Nodes {
		nodeComp, err := newRealNodeComponent(ctx, fmt.Sprintf("%s-%s", name, nodeConfig.Name), &nodeConfig, sshKeyOutput, sshPrivateKey, sharedDOSshKey, doToken, linodeToken, component)
		if err != nil {
			return nil, nil, err
		}
		realNodeComponents = append(realNodeComponents, nodeComp)
		nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
	}

	// Create nodes from pools IN DETERMINISTIC ORDER
	// CRITICAL: Go maps have random iteration order, which causes RKE2 to assign
	// master/worker roles incorrectly. Process pools in explicit order: masters first!
	nodeIndex := len(realNodeComponents)

	// Define deterministic pool order: ALL masters first, then ALL workers
	poolOrder := []string{"do-masters", "linode-masters", "do-workers", "linode-workers"}

	for _, poolName := range poolOrder {
		poolConfig, exists := clusterConfig.NodePools[poolName]
		if !exists {
			continue // Skip if pool not defined
		}

		for i := 0; i < poolConfig.Count; i++ {
			nodeName := fmt.Sprintf("%s-%d", poolConfig.Name, i+1)

			nodeConfig := config.NodeConfig{
				Name:        nodeName,
				Provider:    poolConfig.Provider,
				Region:      poolConfig.Region,
				Size:        poolConfig.Size,
				Image:       poolConfig.Image,
				Roles:       poolConfig.Roles,
				Labels:      poolConfig.Labels,
				Taints:      poolConfig.Taints,
				PrivateIP:   fmt.Sprintf("10.0.1.%d", nodeIndex+1),
				WireGuardIP: fmt.Sprintf("10.8.0.%d", 10+nodeIndex),
			}

			nodeComp, err := newRealNodeComponent(ctx, fmt.Sprintf("%s-%s-%s", name, poolName, nodeName), &nodeConfig, sshKeyOutput, sshPrivateKey, sharedDOSshKey, doToken, linodeToken, component)
			if err != nil {
				return nil, nil, err
			}
			realNodeComponents = append(realNodeComponents, nodeComp)
			nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
			nodeIndex++
		}
	}

	component.Nodes = pulumi.ToArrayOutput(nodesArray)
	component.Status = pulumi.Sprintf("Deployed %d real cloud instances", len(realNodeComponents))

	// Store real node components for later use (WireGuard, RKE, etc)
	ctx.Export("__realNodes", pulumi.ToOutput(realNodeComponents))

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodes":  component.Nodes,
		"status": component.Status,
	}); err != nil {
		return nil, nil, err
	}

	// Return both the component and the list of real nodes
	return component, realNodeComponents, nil
}

// newRealNodeComponent creates a real DigitalOcean Droplet or Linode Instance AND provisions it
func newRealNodeComponent(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, sshPrivateKey pulumi.StringOutput, sharedDOSshKey *digitalocean.SshKey, doToken pulumi.StringInput, linodeToken pulumi.StringInput, parent pulumi.Resource) (*RealNodeComponent, error) {
	component := &RealNodeComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:compute:RealNode", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	component.NodeName = pulumi.String(nodeConfig.Name).ToStringOutput()
	component.Provider = pulumi.String(nodeConfig.Provider).ToStringOutput()
	component.Region = pulumi.String(nodeConfig.Region).ToStringOutput()
	component.Size = pulumi.String(nodeConfig.Size).ToStringOutput()
	component.WireGuardIP = pulumi.String(nodeConfig.WireGuardIP).ToStringOutput()

	// Convert roles
	rolesArray := make([]pulumi.Output, len(nodeConfig.Roles))
	for i, role := range nodeConfig.Roles {
		rolesArray[i] = pulumi.String(role).ToStringOutput()
	}
	component.Roles = pulumi.ToArrayOutput(rolesArray)

	// Create real cloud resource based on provider
	if nodeConfig.Provider == "digitalocean" {
		err = createDigitalOceanDroplet(ctx, name, nodeConfig, sharedDOSshKey, doToken, component)
	} else if nodeConfig.Provider == "linode" {
		err = createLinodeInstance(ctx, name, nodeConfig, sshKeyOutput, linodeToken, component)
	} else {
		return nil, fmt.Errorf("unknown provider: %s", nodeConfig.Provider)
	}

	if err != nil {
		return nil, err
	}

	// PROVISION THIS NODE with Docker and Kubernetes prerequisites
	// Now we have direct access to component.PublicIP!
	_, err = NewRealNodeProvisioningComponent(ctx,
		fmt.Sprintf("%s-provision", name),
		component.PublicIP,
		nodeConfig.Name,
		sshPrivateKey,
		component)
	if err != nil {
		ctx.Log.Warn(fmt.Sprintf("Failed to provision node %s: %v", nodeConfig.Name, err), nil)
		// Don't fail the entire deployment if provisioning fails - we can fix it later
	}

	component.Status = pulumi.String("provisioned").ToStringOutput()

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"nodeName":    component.NodeName,
		"provider":    component.Provider,
		"region":      component.Region,
		"size":        component.Size,
		"publicIP":    component.PublicIP,
		"privateIP":   component.PrivateIP,
		"wireGuardIP": component.WireGuardIP,
		"roles":       component.Roles,
		"status":      component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// createDigitalOceanDroplet creates a real DigitalOcean Droplet
func createDigitalOceanDroplet(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sharedSshKey *digitalocean.SshKey, doToken pulumi.StringInput, component *RealNodeComponent) error {
	// Use the shared SSH key (already created, no duplication)
	// Create Droplet
	droplet, err := digitalocean.NewDroplet(ctx, name, &digitalocean.DropletArgs{
		Image:  pulumi.String(nodeConfig.Image),
		Name:   pulumi.String(nodeConfig.Name),
		Region: pulumi.String(nodeConfig.Region),
		Size:   pulumi.String(nodeConfig.Size),
		SshKeys: pulumi.StringArray{
			sharedSshKey.Fingerprint,
		},
		Tags: pulumi.StringArray{
			pulumi.String("kubernetes"),
			pulumi.String(ctx.Stack()),
		},
		Ipv6:               pulumi.Bool(true),
		Monitoring:         pulumi.Bool(true),
		PrivateNetworking:  pulumi.Bool(true),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create droplet: %w", err)
	}

	component.DropletID = droplet.ID()
	component.PublicIP = droplet.Ipv4Address
	component.PrivateIP = droplet.Ipv4AddressPrivate

	return nil
}

// createLinodeInstance creates a real Linode Instance
func createLinodeInstance(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, linodeToken pulumi.StringInput, component *RealNodeComponent) error {
	// Linode requires SSH key in single line format (remove newlines)
	singleLineKey := sshKeyOutput.ApplyT(func(key string) string {
		// Remove all newlines and ensure it's a single line
		return strings.ReplaceAll(strings.ReplaceAll(key, "\n", ""), "\r", "")
	}).(pulumi.StringOutput)

	// Create Linode Instance
	instance, err := linode.NewInstance(ctx, name, &linode.InstanceArgs{
		Label:  pulumi.String(nodeConfig.Name),
		Region: pulumi.String(nodeConfig.Region),
		Type:   pulumi.String(nodeConfig.Size),
		Image:  pulumi.String(nodeConfig.Image),
		AuthorizedKeys: pulumi.StringArray{
			singleLineKey,
		},
		Tags: pulumi.StringArray{
			pulumi.String("kubernetes"),
			pulumi.String(ctx.Stack()),
		},
		PrivateIp: pulumi.Bool(true),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create linode instance: %w", err)
	}

	component.InstanceID = instance.ID().ApplyT(func(id pulumi.ID) int {
		// Linode IDs are integers, but Pulumi returns IDOutput
		return 0 // Placeholder
	}).(pulumi.IntOutput)
	component.PublicIP = instance.IpAddress

	// Get private IP from instance configs
	component.PrivateIP = instance.PrivateIpAddress

	return nil
}
