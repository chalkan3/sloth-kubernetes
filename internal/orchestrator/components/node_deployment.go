package components

import (
	"fmt"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"sloth-kubernetes/pkg/config"
)

// NodeDeploymentComponent handles node deployment
type NodeDeploymentComponent struct {
	pulumi.ResourceState

	Nodes  pulumi.ArrayOutput  `pulumi:"nodes"`
	Status pulumi.StringOutput `pulumi:"status"`
}

// RealNodeComponent represents a real cloud instance (Droplet or Linode)
type RealNodeComponent struct {
	pulumi.ResourceState

	NodeName    pulumi.StringOutput `pulumi:"nodeName"`
	Provider    pulumi.StringOutput `pulumi:"provider"`
	Region      pulumi.StringOutput `pulumi:"region"`
	Size        pulumi.StringOutput `pulumi:"size"`
	PublicIP    pulumi.StringOutput `pulumi:"publicIP"`
	PrivateIP   pulumi.StringOutput `pulumi:"privateIP"`
	WireGuardIP pulumi.StringOutput `pulumi:"wireGuardIP"`
	Roles       pulumi.ArrayOutput  `pulumi:"roles"`
	Status      pulumi.StringOutput `pulumi:"status"`
	DropletID   pulumi.IDOutput     `pulumi:"dropletId"`  // For DigitalOcean
	InstanceID  pulumi.IntOutput    `pulumi:"instanceId"` // For Linode
}

// NewRealNodeDeploymentComponent creates real cloud resources
// Returns NodeDeploymentComponent and list of RealNodeComponents for WireGuard/RKE
// bastionComponent is optional - if provided, SSH connections will use ProxyJump through the bastion
func NewRealNodeDeploymentComponent(ctx *pulumi.Context, name string, clusterConfig *config.ClusterConfig, sshKeyOutput pulumi.StringOutput, sshPrivateKey pulumi.StringOutput, doToken pulumi.StringInput, linodeToken pulumi.StringInput, vpcComponent *VPCComponent, bastionComponent *BastionComponent, opts ...pulumi.ResourceOption) (*NodeDeploymentComponent, []*RealNodeComponent, error) {
	component := &NodeDeploymentComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:compute:NodeDeployment", name, component, opts...)
	if err != nil {
		return nil, nil, err
	}

	// Check if bastion is enabled - if so, SSH access will be restricted to bastion only
	bastionEnabled := clusterConfig.Security.Bastion != nil && clusterConfig.Security.Bastion.Enabled
	var bastionIP *pulumi.StringOutput
	if bastionEnabled {
		ctx.Log.Info("🔒 Bastion enabled - SSH access restricted to bastion only", nil)
		ctx.Log.Info("   ℹ️  Note: Nodes get public IPs (cloud provider limitation)", nil)
		ctx.Log.Info("   ℹ️  Public IPs needed for K8s API, ingress traffic, WireGuard VPN", nil)
		ctx.Log.Info("   ℹ️  UFW firewall will block direct SSH - use bastion as jump host", nil)
		if bastionComponent != nil {
			ip := bastionComponent.PublicIP
			bastionIP = &ip
		}
	} else {
		ctx.Log.Info("🌍 Bastion disabled - nodes have direct SSH access", nil)
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
		nodeComp, err := newRealNodeComponent(ctx, fmt.Sprintf("%s-%s", name, nodeConfig.Name), &nodeConfig, sshKeyOutput, sshPrivateKey, sharedDOSshKey, doToken, linodeToken, vpcComponent, bastionEnabled, component)
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
			nodeName := fmt.Sprintf("%s-%d", poolName, i+1)

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

			nodeComp, err := newRealNodeComponent(ctx, fmt.Sprintf("%s-%s-%s", name, poolName, nodeName), &nodeConfig, sshKeyOutput, sshPrivateKey, sharedDOSshKey, doToken, linodeToken, vpcComponent, bastionEnabled, component)
			if err != nil {
				return nil, nil, err
			}
			realNodeComponents = append(realNodeComponents, nodeComp)
			nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
			nodeIndex++
		}
	}

	component.Nodes = pulumi.ToArrayOutput(nodesArray)

	ctx.Log.Info(fmt.Sprintf("✅ All %d VMs created, starting SEQUENTIAL provisioning...", len(realNodeComponents)), nil)

	// SEQUENTIAL PROVISIONING PHASE
	// When bastion is enabled, provision nodes ONE AT A TIME to avoid overwhelming the bastion host
	// Each node depends on the previous node's provisioning completion
	provisioningComponents := []*RealNodeProvisioningComponent{}
	var previousProvisioningComponent pulumi.Resource = component // First node depends on main component

	// Track nodes by index to get correct names
	provNodeIndex := 0

	// Provision individual nodes
	for _, nodeConfig := range clusterConfig.Nodes {
		if provNodeIndex >= len(realNodeComponents) {
			break
		}
		nodeComp := realNodeComponents[provNodeIndex]

		// Create provisioning component - each depends on the previous one (SEQUENTIAL!)
		provComp, err := NewRealNodeProvisioningComponent(ctx,
			fmt.Sprintf("%s-%s-provision", name, nodeConfig.Name),
			nodeComp.PublicIP,
			nodeConfig.Name,
			sshPrivateKey,
			bastionIP, // Use bastion ProxyJump if enabled
			pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{previousProvisioningComponent})) // CRITICAL: Depend on previous node!

		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to create provisioning for node %s: %v", nodeConfig.Name, err), nil)
		} else {
			provisioningComponents = append(provisioningComponents, provComp)
			previousProvisioningComponent = provComp // Next node will depend on this one
		}
		provNodeIndex++
	}

	// Provision pool nodes in deterministic order
	for _, poolName := range poolOrder {
		poolConfig, exists := clusterConfig.NodePools[poolName]
		if !exists || poolConfig.Count == 0 {
			continue
		}

		for i := 0; i < poolConfig.Count; i++ {
			if provNodeIndex >= len(realNodeComponents) {
				break
			}
			nodeComp := realNodeComponents[provNodeIndex]
			nodeName := fmt.Sprintf("%s-%d", poolName, i+1)

			// Create provisioning component - each depends on the previous one (SEQUENTIAL!)
			provComp, err := NewRealNodeProvisioningComponent(ctx,
				fmt.Sprintf("%s-%s-provision", name, nodeName),
				nodeComp.PublicIP,
				nodeName,
				sshPrivateKey,
				bastionIP, // Use bastion ProxyJump if enabled
				pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{previousProvisioningComponent})) // CRITICAL: Depend on previous node!

			if err != nil {
				ctx.Log.Warn(fmt.Sprintf("Failed to create provisioning for node %s: %v", nodeName, err), nil)
			} else {
				provisioningComponents = append(provisioningComponents, provComp)
				previousProvisioningComponent = provComp // Next node will depend on this one
			}
			provNodeIndex++
		}
	}

	ctx.Log.Info(fmt.Sprintf("📦 Provisioning %d nodes SEQUENTIALLY (1 at a time to avoid bastion overload)...", len(provisioningComponents)), nil)

	component.Status = pulumi.Sprintf("Deployed %d VMs, provisioning %d nodes SEQUENTIALLY",
		len(realNodeComponents), len(provisioningComponents))

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
func newRealNodeComponent(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, sshPrivateKey pulumi.StringOutput, sharedDOSshKey *digitalocean.SshKey, doToken pulumi.StringInput, linodeToken pulumi.StringInput, vpcComponent *VPCComponent, bastionEnabled bool, parent pulumi.Resource) (*RealNodeComponent, error) {
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
		err = createDigitalOceanDroplet(ctx, name, nodeConfig, sharedDOSshKey, doToken, vpcComponent, bastionEnabled, component)
	} else if nodeConfig.Provider == "linode" {
		err = createLinodeInstance(ctx, name, nodeConfig, sshKeyOutput, linodeToken, bastionEnabled, component)
	} else {
		return nil, fmt.Errorf("unknown provider: %s", nodeConfig.Provider)
	}

	if err != nil {
		return nil, err
	}

	// NOTE: Provisioning is now done in a separate parallel phase
	// This allows all VMs to be created first, then ALL provisioned in parallel
	// See NewRealNodeDeploymentComponent for the parallel provisioning phase

	component.Status = pulumi.String("created").ToStringOutput()

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
func createDigitalOceanDroplet(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sharedSshKey *digitalocean.SshKey, doToken pulumi.StringInput, vpcComponent *VPCComponent, bastionEnabled bool, component *RealNodeComponent) error {
	// Use the shared SSH key (already created, no duplication)

	// Build droplet args
	dropletArgs := &digitalocean.DropletArgs{
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
		Ipv6:       pulumi.Bool(true),
		Monitoring: pulumi.Bool(true),
	}

	// If bastion is enabled, attach to VPC and configure for bastion-only SSH access
	if bastionEnabled && vpcComponent != nil {
		ctx.Log.Info(fmt.Sprintf("🔒 Creating droplet %s (SSH restricted to bastion only)", nodeConfig.Name), nil)
		dropletArgs.VpcUuid = vpcComponent.VPCID
		// NOTE: DigitalOcean droplets always get public IPs (provider limitation)
		// Public IPs are required for:
		//   - Kubernetes API Server (port 6443)
		//   - HTTP/HTTPS Ingress (ports 80/443)
		//   - WireGuard VPN (port 51820)
		// SSH (port 22) will be restricted to bastion IP only via UFW firewall
		ctx.Log.Info(fmt.Sprintf("   → Public IP will be assigned (required for K8s API & ingress traffic)"), nil)
		ctx.Log.Info(fmt.Sprintf("   → SSH access will be restricted to bastion only"), nil)
	} else {
		ctx.Log.Info(fmt.Sprintf("🌍 Creating PUBLIC droplet %s (direct SSH access enabled)", nodeConfig.Name), nil)
	}

	// Create Droplet
	droplet, err := digitalocean.NewDroplet(ctx, name, dropletArgs, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create droplet: %w", err)
	}

	component.DropletID = droplet.ID()

	// Set public IP (DigitalOcean always assigns public IPs - provider limitation)
	component.PublicIP = droplet.Ipv4Address
	if bastionEnabled {
		ctx.Log.Info(fmt.Sprintf("   ✅ Droplet %s created (VPC attached, SSH via bastion)", nodeConfig.Name), nil)
	}

	component.PrivateIP = droplet.Ipv4AddressPrivate

	return nil
}

// createLinodeInstance creates a real Linode Instance
func createLinodeInstance(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, linodeToken pulumi.StringInput, bastionEnabled bool, component *RealNodeComponent) error {
	// Use the SSH key directly - it's already normalized in sshkeys.go
	// The key is in format: "ssh-rsa AAAAB3..." (type + key-data only, no comment)

	if bastionEnabled {
		ctx.Log.Info(fmt.Sprintf("🔒 Creating Linode instance %s (SSH restricted to bastion only)", nodeConfig.Name), nil)
		// NOTE: Linode instances always get public IPs (provider limitation)
		// Public IPs are required for K8s API, ingress traffic, and WireGuard VPN
		// SSH access will be restricted to bastion IP only via UFW firewall
		ctx.Log.Info(fmt.Sprintf("   → Public IP will be assigned (required for K8s API & ingress traffic)"), nil)
		ctx.Log.Info(fmt.Sprintf("   → SSH access will be restricted to bastion only"), nil)
	} else {
		ctx.Log.Info(fmt.Sprintf("🌍 Creating PUBLIC Linode instance %s (direct SSH access enabled)", nodeConfig.Name), nil)
	}

	// Create Linode Instance
	instance, err := linode.NewInstance(ctx, name, &linode.InstanceArgs{
		Label:  pulumi.String(nodeConfig.Name),
		Region: pulumi.String(nodeConfig.Region),
		Type:   pulumi.String(nodeConfig.Size),
		Image:  pulumi.String(nodeConfig.Image),
		AuthorizedKeys: pulumi.StringArray{
			sshKeyOutput,
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

	// Set public IP (Linode always assigns public IPs - provider limitation)
	component.PublicIP = instance.IpAddress
	if bastionEnabled {
		ctx.Log.Info(fmt.Sprintf("   ✅ Linode instance %s created (SSH via bastion)", nodeConfig.Name), nil)
	}

	// Get private IP from instance configs
	component.PrivateIP = instance.PrivateIpAddress

	return nil
}
