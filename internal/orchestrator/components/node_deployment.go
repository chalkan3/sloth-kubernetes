package components

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/cloudinit"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	azurecompute "github.com/pulumi/pulumi-azure-native-sdk/compute/v2"
	azurenetwork "github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	azureresources "github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

// ProviderNodeCreator defines the function signature for creating nodes on different providers
type ProviderNodeCreator func(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent, extras map[string]interface{}) error

// providerNodeCreators maps provider names to their node creation functions
var providerNodeCreators = map[string]ProviderNodeCreator{
	"digitalocean": createDigitalOceanNode,
	"linode":       createLinodeNode,
	"azure":        createAzureNode,
	"aws":          createAWSNode,
	"hetzner":      createHetznerNode,
}

// RegisterProviderCreator allows registering new provider creators dynamically
func RegisterProviderCreator(provider string, creator ProviderNodeCreator) {
	providerNodeCreators[provider] = creator
}

// GetProviderCreator returns the creator function for a provider
func GetProviderCreator(provider string) (ProviderNodeCreator, bool) {
	creator, ok := providerNodeCreators[provider]
	return creator, ok
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
	if bastionEnabled {
		ctx.Log.Info("🔒 Bastion enabled - SSH access restricted to bastion only", nil)
		ctx.Log.Info("   ℹ️  Note: Nodes get public IPs (cloud provider limitation)", nil)
		ctx.Log.Info("   ℹ️  Public IPs needed for K8s API, ingress traffic, WireGuard VPN", nil)
		ctx.Log.Info("   ℹ️  UFW firewall will block direct SSH - use bastion as jump host", nil)
	} else {
		ctx.Log.Info("🌍 Bastion disabled - nodes have direct SSH access", nil)
	}

	// Detect which providers are used
	usesDigitalOcean := false
	for _, node := range clusterConfig.Nodes {
		if node.Provider == "digitalocean" {
			usesDigitalOcean = true
			break
		}
	}
	if !usesDigitalOcean {
		for _, pool := range clusterConfig.NodePools {
			if pool.Provider == "digitalocean" {
				usesDigitalOcean = true
				break
			}
		}
	}

	// Create ONE shared SSH key for all DigitalOcean Droplets (only if DO is used)
	var sharedDOSshKey *digitalocean.SshKey
	if usesDigitalOcean {
		var err error
		sharedDOSshKey, err = digitalocean.NewSshKey(ctx, fmt.Sprintf("%s-shared-key", name), &digitalocean.SshKeyArgs{
			Name:      pulumi.Sprintf("kubernetes-cluster-production-key"),
			PublicKey: sshKeyOutput,
		}, pulumi.Parent(component))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create shared DO SSH key: %w", err)
		}
	}

	// NOTE: cloud-init installs prerequisites (WireGuard, packages) on all nodes
	// K3s installation is handled by remote commands AFTER WireGuard is configured
	// Linode supports cloud-init natively via Metadatas.UserData field
	// This is cleaner and more consistent with DigitalOcean's UserData approach

	realNodeComponents := []*RealNodeComponent{}
	nodesArray := []pulumi.Output{}

	// Create individual nodes
	for _, nodeConfig := range clusterConfig.Nodes {
		nodeComp, err := newRealNodeComponent(ctx, fmt.Sprintf("%s-%s", name, nodeConfig.Name), &nodeConfig, sshKeyOutput, sshPrivateKey, sharedDOSshKey, nil, doToken, linodeToken, vpcComponent, bastionComponent, component)
		if err != nil {
			return nil, nil, err
		}
		realNodeComponents = append(realNodeComponents, nodeComp)
		nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
	}

	// Create nodes from pools IN DETERMINISTIC ORDER
	// CRITICAL: Go maps have random iteration order, which causes K3s to assign
	// master/worker roles incorrectly. Process pools in explicit order: masters first!
	nodeIndex := len(realNodeComponents)

	// Build deterministic pool order: ALL masters first, then ALL workers
	// This allows for dynamic providers (DigitalOcean, Linode, Azure, AWS, GCP)
	poolOrder := []string{}

	// DEBUG: Log all node pools
	ctx.Log.Info(fmt.Sprintf("🔍 DEBUG: Total node pools in config: %d", len(clusterConfig.NodePools)), nil)
	for poolName, pool := range clusterConfig.NodePools {
		ctx.Log.Info(fmt.Sprintf("🔍 DEBUG: Pool '%s' - provider=%s, count=%d", poolName, pool.Provider, pool.Count), nil)
	}

	// First pass: add all master pools
	for poolName, pool := range clusterConfig.NodePools {
		for _, role := range pool.Roles {
			if role == "master" || role == "controlplane" {
				poolOrder = append(poolOrder, poolName)
				break
			}
		}
	}

	// Second pass: add all worker pools
	for poolName, pool := range clusterConfig.NodePools {
		isMaster := false
		for _, role := range pool.Roles {
			if role == "master" || role == "controlplane" {
				isMaster = true
				break
			}
		}
		if !isMaster {
			poolOrder = append(poolOrder, poolName)
		}
	}

	for _, poolName := range poolOrder {
		poolConfig := clusterConfig.NodePools[poolName]

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

			nodeComp, err := newRealNodeComponent(ctx, fmt.Sprintf("%s-%s-%s", name, poolName, nodeName), &nodeConfig, sshKeyOutput, sshPrivateKey, sharedDOSshKey, nil, doToken, linodeToken, vpcComponent, bastionComponent, component)
			if err != nil {
				return nil, nil, err
			}
			realNodeComponents = append(realNodeComponents, nodeComp)
			nodesArray = append(nodesArray, pulumi.ToOutput(nodeComp))
			nodeIndex++
		}
	}

	component.Nodes = pulumi.ToArrayOutput(nodesArray)

	ctx.Log.Info(fmt.Sprintf("✅ All %d VMs created, starting PARALLEL provisioning...", len(realNodeComponents)), nil)

	// CLOUD-INIT PROVISIONING (OPTIMIZED)
	// Docker and WireGuard are now installed via cloud-init user-data during VM boot
	// This eliminates the need for SSH provisioning and saves ~2-3 minutes per node
	// The cloud-init validator (in cluster_orchestrator.go) waits for installation to complete
	//
	// NOTE: SSH provisioning is DISABLED because cloud-init handles everything:
	// - DigitalOcean: Uses UserData field with cloud-init script
	// - Linode: TODO - Add Stackscript support for cloud-init
	//
	// OLD CODE (COMMENTED OUT - kept for reference):
	// provisioningComponents := []*RealNodeProvisioningComponent{}
	// var bastionProvisioningDep pulumi.Resource = component
	// ... (SSH provisioning logic removed)

	ctx.Log.Info("✅ Node provisioning handled by cloud-init (UserData) - SSH provisioning disabled", nil)

	component.Status = pulumi.Sprintf("Deployed %d VMs with cloud-init provisioning",
		len(realNodeComponents))

	// Store real node components for later use (WireGuard, RKE, etc)
	secrets.Export(ctx,"__realNodes", pulumi.ToOutput(realNodeComponents))

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
func newRealNodeComponent(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, sshPrivateKey pulumi.StringOutput, sharedDOSshKey *digitalocean.SshKey, sharedLinodeStackscript *linode.StackScript, doToken pulumi.StringInput, linodeToken pulumi.StringInput, vpcComponent *VPCComponent, bastionComponent *BastionComponent, parent pulumi.Resource) (*RealNodeComponent, error) {
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

	// Determine if bastion is enabled and get Salt Master IP
	bastionEnabled := bastionComponent != nil && bastionComponent.BastionName.ToStringOutput() != pulumi.String("").ToStringOutput()
	saltMasterIP := ""
	if bastionEnabled {
		// Use the fixed WireGuard IP of the bastion (10.8.0.5)
		saltMasterIP = "10.8.0.5"
	}

	// Create real cloud resource based on provider using the provider map
	creator, ok := GetProviderCreator(nodeConfig.Provider)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: digitalocean, linode, azure, aws)", nodeConfig.Provider)
	}

	// Build extras map with provider-specific dependencies
	extras := map[string]interface{}{
		"sharedDOSshKey":          sharedDOSshKey,
		"sharedLinodeStackscript": sharedLinodeStackscript,
		"doToken":                 doToken,
		"linodeToken":             linodeToken,
		"vpcComponent":            vpcComponent,
		"bastionComponent":        bastionComponent,
	}

	err = creator(ctx, name, nodeConfig, sshKeyOutput, bastionEnabled, saltMasterIP, component, extras)

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
func createDigitalOceanDroplet(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sharedSshKey *digitalocean.SshKey, doToken pulumi.StringInput, vpcComponent *VPCComponent, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent) error {
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
			pulumi.String(strings.ReplaceAll(ctx.Stack(), ".", "-")),
		},
		Ipv6:       pulumi.Bool(true),
		Monitoring: pulumi.Bool(true),
		// Cloud-init user-data: Install prerequisites (WireGuard, packages, Salt Minion) during VM boot
		// K3s installation is handled by remote commands AFTER WireGuard is configured
		// Set unique hostname to avoid etcd "duplicate node name" errors
		// If Salt Master IP is provided, Salt Minion will be installed and configured
		UserData: pulumi.String(cloudinit.GenerateUserDataWithHostnameAndSalt(nodeConfig.Name, saltMasterIP)),
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
func createLinodeInstance(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, sharedStackscript *linode.StackScript, linodeToken pulumi.StringInput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent) error {
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

	// Create Linode Instance with cloud-init UserData
	// CRITICAL FIX: Linode supports cloud-init via Metadatas.UserData field
	// Previously, we were only using StackscriptId, which doesn't support cloud-init
	// This caused RKE2 to NOT be installed on Linode machines!
	//
	// NOTE: UserData must be base64-encoded per Linode API requirements
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
			pulumi.String(strings.ReplaceAll(ctx.Stack(), ".", "-")),
		},
		PrivateIp: pulumi.Bool(true),
		// CRITICAL: Linode supports cloud-init via Metadatas.UserData field
		// This is the SAME cloud-init format as DigitalOcean's UserData field
		// Linode's cloud-init support is native (not via Stackscripts)
		// UserData must be base64-encoded
		// Set unique hostname to avoid etcd "duplicate node name" errors
		// K3s installation is handled by remote commands AFTER WireGuard is configured
		// If Salt Master IP is provided, Salt Minion will be installed and configured
		Metadatas: linode.InstanceMetadataArray{
			&linode.InstanceMetadataArgs{
				UserData: pulumi.String(base64.StdEncoding.EncodeToString([]byte(cloudinit.GenerateUserDataWithHostnameAndSalt(nodeConfig.Name, saltMasterIP)))),
			},
		},
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

// Global Azure shared resources (created once, reused by all VMs)
var (
	azureResourceGroup *azureresources.ResourceGroup
	azureVNet          *azurenetwork.VirtualNetwork
	azureSubnet        *azurenetwork.Subnet
	azureNSG           *azurenetwork.NetworkSecurityGroup
)

// createAzureVM creates a real Azure VM with all required infrastructure
func createAzureVM(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent) error {
	location := nodeConfig.Region
	if location == "" {
		location = "eastus"
	}

	if bastionEnabled {
		ctx.Log.Info(fmt.Sprintf("🔒 Creating Azure VM %s (SSH restricted to bastion only)", nodeConfig.Name), nil)
	} else {
		ctx.Log.Info(fmt.Sprintf("🌍 Creating PUBLIC Azure VM %s", nodeConfig.Name), nil)
	}

	// Create shared Azure infrastructure (only once for all VMs)
	if azureResourceGroup == nil {
		rgName := "sloth-k8s-rg"
		rg, err := azureresources.NewResourceGroup(ctx, rgName, &azureresources.ResourceGroupArgs{
			ResourceGroupName: pulumi.String(rgName),
			Location:          pulumi.String(location),
			Tags: pulumi.StringMap{
				"Environment": pulumi.String("production"),
				"ManagedBy":   pulumi.String("sloth-kubernetes"),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Azure resource group: %w", err)
		}
		azureResourceGroup = rg

		// Create VNet
		vnetName := "sloth-k8s-azure-vnet"
		vnet, err := azurenetwork.NewVirtualNetwork(ctx, vnetName, &azurenetwork.VirtualNetworkArgs{
			ResourceGroupName:  rg.Name,
			Location:           pulumi.String(location),
			VirtualNetworkName: pulumi.String(vnetName),
			AddressSpace: &azurenetwork.AddressSpaceArgs{
				AddressPrefixes: pulumi.StringArray{pulumi.String("10.14.0.0/16")},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Azure VNet: %w", err)
		}
		azureVNet = vnet

		// Create Subnet
		subnetName := "sloth-k8s-subnet"
		subnet, err := azurenetwork.NewSubnet(ctx, subnetName, &azurenetwork.SubnetArgs{
			ResourceGroupName:  rg.Name,
			VirtualNetworkName: vnet.Name,
			SubnetName:         pulumi.String(subnetName),
			AddressPrefix:      pulumi.String("10.14.1.0/24"),
		})
		if err != nil {
			return fmt.Errorf("failed to create Azure subnet: %w", err)
		}
		azureSubnet = subnet

		// Create NSG
		nsgName := "sloth-k8s-nsg"
		nsg, err := azurenetwork.NewNetworkSecurityGroup(ctx, nsgName, &azurenetwork.NetworkSecurityGroupArgs{
			ResourceGroupName:        rg.Name,
			Location:                 pulumi.String(location),
			NetworkSecurityGroupName: pulumi.String(nsgName),
			SecurityRules: azurenetwork.SecurityRuleTypeArray{
				&azurenetwork.SecurityRuleTypeArgs{
					Name:                     pulumi.String("AllowSSH"),
					Priority:                 pulumi.Int(1000),
					Direction:                pulumi.String("Inbound"),
					Access:                   pulumi.String("Allow"),
					Protocol:                 pulumi.String("Tcp"),
					SourcePortRange:          pulumi.String("*"),
					DestinationPortRange:     pulumi.String("22"),
					SourceAddressPrefix:      pulumi.String("*"),
					DestinationAddressPrefix: pulumi.String("*"),
				},
				&azurenetwork.SecurityRuleTypeArgs{
					Name:                     pulumi.String("AllowWireGuard"),
					Priority:                 pulumi.Int(1010),
					Direction:                pulumi.String("Inbound"),
					Access:                   pulumi.String("Allow"),
					Protocol:                 pulumi.String("Udp"),
					SourcePortRange:          pulumi.String("*"),
					DestinationPortRange:     pulumi.String("51820"),
					SourceAddressPrefix:      pulumi.String("*"),
					DestinationAddressPrefix: pulumi.String("*"),
				},
				&azurenetwork.SecurityRuleTypeArgs{
					Name:                     pulumi.String("AllowKubernetesAPI"),
					Priority:                 pulumi.Int(1020),
					Direction:                pulumi.String("Inbound"),
					Access:                   pulumi.String("Allow"),
					Protocol:                 pulumi.String("Tcp"),
					SourcePortRange:          pulumi.String("*"),
					DestinationPortRange:     pulumi.String("6443"),
					SourceAddressPrefix:      pulumi.String("*"),
					DestinationAddressPrefix: pulumi.String("*"),
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Azure NSG: %w", err)
		}
		azureNSG = nsg
	}

	// Create Public IP for this VM
	publicIPName := fmt.Sprintf("%s-pip", nodeConfig.Name)
	publicIP, err := azurenetwork.NewPublicIPAddress(ctx, publicIPName, &azurenetwork.PublicIPAddressArgs{
		ResourceGroupName:        azureResourceGroup.Name,
		Location:                 pulumi.String(location),
		PublicIpAddressName:      pulumi.String(publicIPName),
		PublicIPAllocationMethod: pulumi.String("Static"),
		Sku: &azurenetwork.PublicIPAddressSkuArgs{
			Name: pulumi.String("Standard"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create public IP: %w", err)
	}

	// Create Network Interface
	nicName := fmt.Sprintf("%s-nic", nodeConfig.Name)
	nic, err := azurenetwork.NewNetworkInterface(ctx, nicName, &azurenetwork.NetworkInterfaceArgs{
		ResourceGroupName:    azureResourceGroup.Name,
		Location:             pulumi.String(location),
		NetworkInterfaceName: pulumi.String(nicName),
		IpConfigurations: azurenetwork.NetworkInterfaceIPConfigurationArray{
			&azurenetwork.NetworkInterfaceIPConfigurationArgs{
				Name:                      pulumi.String("ipconfig1"),
				PrivateIPAllocationMethod: pulumi.String("Dynamic"),
				Subnet: &azurenetwork.SubnetTypeArgs{
					Id: azureSubnet.ID(),
				},
				PublicIPAddress: &azurenetwork.PublicIPAddressTypeArgs{
					Id: publicIP.ID(),
				},
			},
		},
		NetworkSecurityGroup: &azurenetwork.NetworkSecurityGroupTypeArgs{
			Id: azureNSG.ID(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	// Generate cloud-init user data with Salt Minion if master IP is provided
	userData := cloudinit.GenerateUserDataWithHostnameAndSalt(nodeConfig.Name, saltMasterIP)
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userData))

	// Map image name to Azure image reference
	imageReference := &azurecompute.ImageReferenceArgs{
		Publisher: pulumi.String("Canonical"),
		Offer:     pulumi.String("0001-com-ubuntu-server-jammy"),
		Sku:       pulumi.String("22_04-lts-gen2"),
		Version:   pulumi.String("latest"),
	}

	// Generate a secure password (required by Azure but we use SSH keys)
	adminPassword := generateSecurePassword()

	// Create Virtual Machine
	vmArgs := &azurecompute.VirtualMachineArgs{
		ResourceGroupName: azureResourceGroup.Name,
		Location:          pulumi.String(location),
		VmName:            pulumi.String(nodeConfig.Name),
		NetworkProfile: &azurecompute.NetworkProfileArgs{
			NetworkInterfaces: azurecompute.NetworkInterfaceReferenceArray{
				&azurecompute.NetworkInterfaceReferenceArgs{
					Id:      nic.ID(),
					Primary: pulumi.Bool(true),
				},
			},
		},
		HardwareProfile: &azurecompute.HardwareProfileArgs{
			VmSize: pulumi.String(nodeConfig.Size),
		},
		OsProfile: &azurecompute.OSProfileArgs{
			ComputerName:  pulumi.String(nodeConfig.Name),
			AdminUsername: pulumi.String("azureuser"),
			AdminPassword: pulumi.String(adminPassword),
			CustomData:    pulumi.String(userDataEncoded),
			LinuxConfiguration: &azurecompute.LinuxConfigurationArgs{
				DisablePasswordAuthentication: pulumi.Bool(true),
				Ssh: &azurecompute.SshConfigurationArgs{
					PublicKeys: azurecompute.SshPublicKeyTypeArray{
						&azurecompute.SshPublicKeyTypeArgs{
							KeyData: sshKeyOutput,
							Path:    pulumi.String("/home/azureuser/.ssh/authorized_keys"),
						},
					},
				},
			},
		},
		StorageProfile: &azurecompute.StorageProfileArgs{
			ImageReference: imageReference,
			OsDisk: &azurecompute.OSDiskArgs{
				Name:         pulumi.String(fmt.Sprintf("%s-osdisk", nodeConfig.Name)),
				CreateOption: pulumi.String("FromImage"),
				ManagedDisk: &azurecompute.ManagedDiskParametersArgs{
					StorageAccountType: pulumi.String("Premium_LRS"),
				},
				DiskSizeGB: pulumi.Int(30),
			},
		},
	}

	vm, err := azurecompute.NewVirtualMachine(ctx, nodeConfig.Name, vmArgs)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// Set component outputs
	component.PublicIP = publicIP.IpAddress.Elem()
	component.PrivateIP = nic.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress().Elem()
	component.DropletID = vm.ID()

	ctx.Log.Info(fmt.Sprintf("   ✅ Azure VM %s created successfully", nodeConfig.Name), nil)

	return nil
}

// generateSecurePassword generates a secure random password for Azure VMs
func generateSecurePassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()"
	const length = 16

	password := make([]byte, length)
	randomBytes := make([]byte, length)
	rand.Read(randomBytes)

	for i := 0; i < length; i++ {
		password[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(password)
}

// ============================================================================
// Provider Node Creator Wrappers (matching ProviderNodeCreator signature)
// ============================================================================

// createDigitalOceanNode wraps createDigitalOceanDroplet with the standard signature
func createDigitalOceanNode(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent, extras map[string]interface{}) error {
	sharedDOSshKey, _ := extras["sharedDOSshKey"].(*digitalocean.SshKey)
	doToken, _ := extras["doToken"].(pulumi.StringInput)
	vpcComponent, _ := extras["vpcComponent"].(*VPCComponent)
	return createDigitalOceanDroplet(ctx, name, nodeConfig, sharedDOSshKey, doToken, vpcComponent, bastionEnabled, saltMasterIP, component)
}

// createLinodeNode wraps createLinodeInstance with the standard signature
func createLinodeNode(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent, extras map[string]interface{}) error {
	sharedLinodeStackscript, _ := extras["sharedLinodeStackscript"].(*linode.StackScript)
	linodeToken, _ := extras["linodeToken"].(pulumi.StringInput)
	return createLinodeInstance(ctx, name, nodeConfig, sshKeyOutput, sharedLinodeStackscript, linodeToken, bastionEnabled, saltMasterIP, component)
}

// createAzureNode wraps createAzureVM with the standard signature
func createAzureNode(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent, extras map[string]interface{}) error {
	return createAzureVM(ctx, name, nodeConfig, sshKeyOutput, bastionEnabled, saltMasterIP, component)
}

// ============================================================================
// AWS EC2 Instance Creation
// ============================================================================

// AWS shared resources (created once, reused by all instances)
var (
	awsProvider      *aws.Provider
	awsVpc           *ec2.Vpc
	awsSubnet        *ec2.Subnet
	awsSecurityGroup *ec2.SecurityGroup
	awsKeyPair       *ec2.KeyPair
)

// Hetzner shared resources (created once, reused by all instances)
var (
	hetznerProvider *hcloud.Provider
	hetznerSshKey   *hcloud.SshKey
)

// createAWSNode creates an AWS EC2 instance
func createAWSNode(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent, extras map[string]interface{}) error {
	region := nodeConfig.Region
	if region == "" {
		region = "us-east-1"
	}

	if bastionEnabled {
		ctx.Log.Info(fmt.Sprintf("🔒 Creating AWS EC2 instance %s (SSH restricted to bastion only)", nodeConfig.Name), nil)
	} else {
		ctx.Log.Info(fmt.Sprintf("🌍 Creating PUBLIC AWS EC2 instance %s", nodeConfig.Name), nil)
	}

	// Create AWS provider with explicit region and credentials (only once for all instances)
	if awsProvider == nil {
		providerArgs := &aws.ProviderArgs{
			Region: pulumi.String(region),
		}
		// Explicitly pass credentials if available (needed for temporary STS credentials)
		if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
			providerArgs.AccessKey = pulumi.StringPtr(accessKey)
		}
		if secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
			providerArgs.SecretKey = pulumi.StringPtr(secretKey)
		}
		if sessionToken := os.Getenv("AWS_SESSION_TOKEN"); sessionToken != "" {
			providerArgs.Token = pulumi.StringPtr(sessionToken)
		}
		provider, err := aws.NewProvider(ctx, fmt.Sprintf("%s-aws-node-provider", ctx.Stack()), providerArgs)
		if err != nil {
			return fmt.Errorf("failed to create AWS provider: %w", err)
		}
		awsProvider = provider
	}

	// Create shared AWS infrastructure (only once for all instances)
	if awsVpc == nil {
		// Create VPC
		vpcName := fmt.Sprintf("%s-vpc", ctx.Stack())
		vpc, err := ec2.NewVpc(ctx, vpcName, &ec2.VpcArgs{
			CidrBlock:          pulumi.String("10.0.0.0/16"),
			EnableDnsHostnames: pulumi.Bool(true),
			EnableDnsSupport:   pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(vpcName),
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to create AWS VPC: %w", err)
		}
		awsVpc = vpc

		// Create Internet Gateway
		igwName := fmt.Sprintf("%s-igw", ctx.Stack())
		igw, err := ec2.NewInternetGateway(ctx, igwName, &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(igwName),
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to create AWS Internet Gateway: %w", err)
		}

		// Create public subnet
		subnetName := fmt.Sprintf("%s-subnet", ctx.Stack())
		subnet, err := ec2.NewSubnet(ctx, subnetName, &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String("10.0.1.0/24"),
			MapPublicIpOnLaunch: pulumi.Bool(true),
			AvailabilityZone:    pulumi.String(fmt.Sprintf("%sa", region)),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(subnetName),
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to create AWS subnet: %w", err)
		}
		awsSubnet = subnet

		// Create route table with internet access
		rtName := fmt.Sprintf("%s-rt", ctx.Stack())
		rt, err := ec2.NewRouteTable(ctx, rtName, &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igw.ID(),
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String(rtName),
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to create AWS route table: %w", err)
		}

		// Associate route table with subnet
		_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-rta", ctx.Stack()), &ec2.RouteTableAssociationArgs{
			SubnetId:     subnet.ID(),
			RouteTableId: rt.ID(),
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to associate route table: %w", err)
		}

		ctx.Log.Info("   ✅ AWS VPC infrastructure created (VPC, IGW, Subnet, Route Table)", nil)
	}

	if awsSecurityGroup == nil {
		sgName := fmt.Sprintf("%s-sg", ctx.Stack())
		sg, err := ec2.NewSecurityGroup(ctx, sgName, &ec2.SecurityGroupArgs{
			Description: pulumi.String("Security group for Kubernetes cluster nodes"),
			VpcId:       awsVpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("SSH"),
					FromPort:    pulumi.Int(22),
					ToPort:      pulumi.Int(22),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("Kubernetes API"),
					FromPort:    pulumi.Int(6443),
					ToPort:      pulumi.Int(6443),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("WireGuard VPN"),
					FromPort:    pulumi.Int(51820),
					ToPort:      pulumi.Int(51820),
					Protocol:    pulumi.String("udp"),
					CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("HTTP"),
					FromPort:    pulumi.Int(80),
					ToPort:      pulumi.Int(80),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("HTTPS"),
					FromPort:    pulumi.Int(443),
					ToPort:      pulumi.Int(443),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("Internal cluster traffic"),
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(0),
					Protocol:    pulumi.String("-1"),
					Self:        pulumi.Bool(true),
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					Protocol:   pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String(sgName),
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to create AWS security group: %w", err)
		}
		awsSecurityGroup = sg

		// Create shared key pair
		keyPairName := fmt.Sprintf("%s-keypair", ctx.Stack())
		kp, err := ec2.NewKeyPair(ctx, keyPairName, &ec2.KeyPairArgs{
			KeyName:   pulumi.String(keyPairName),
			PublicKey: sshKeyOutput,
			Tags: pulumi.StringMap{
				"Name": pulumi.String(keyPairName),
			},
		}, pulumi.Provider(awsProvider))
		if err != nil {
			return fmt.Errorf("failed to create AWS key pair: %w", err)
		}
		awsKeyPair = kp
	}

	// Get Ubuntu AMI for the region
	ami, err := getUbuntuAMIForRegion(ctx, region)
	if err != nil {
		return fmt.Errorf("failed to get Ubuntu AMI: %w", err)
	}

	// Generate cloud-init user data
	userData := cloudinit.GenerateUserDataWithHostnameAndSalt(nodeConfig.Name, saltMasterIP)

	// Create EC2 instance in our VPC subnet
	instance, err := ec2.NewInstance(ctx, name, &ec2.InstanceArgs{
		Ami:                      pulumi.String(ami),
		InstanceType:             pulumi.String(nodeConfig.Size),
		KeyName:                  awsKeyPair.KeyName,
		SubnetId:                 awsSubnet.ID(),
		VpcSecurityGroupIds:      pulumi.StringArray{awsSecurityGroup.ID()},
		AssociatePublicIpAddress: pulumi.Bool(true),
		UserData:                 pulumi.String(userData),
		Tags: pulumi.StringMap{
			"Name":       pulumi.String(nodeConfig.Name),
			"kubernetes": pulumi.String("true"),
			"stack":      pulumi.String(ctx.Stack()),
		},
	}, pulumi.Parent(component), pulumi.Provider(awsProvider))
	if err != nil {
		return fmt.Errorf("failed to create EC2 instance: %w", err)
	}

	// Set component outputs
	component.PublicIP = instance.PublicIp
	component.PrivateIP = instance.PrivateIp
	component.DropletID = instance.ID()

	ctx.Log.Info(fmt.Sprintf("   ✅ AWS EC2 instance %s created successfully", nodeConfig.Name), nil)

	return nil
}

// getUbuntuAMIForRegion returns the Ubuntu 22.04 LTS AMI ID for the given region
func getUbuntuAMIForRegion(ctx *pulumi.Context, region string) (string, error) {
	// Ubuntu 22.04 LTS AMIs by region (canonical owner: 099720109477)
	ubuntuAMIs := map[string]string{
		"us-east-1":      "ami-0c7217cdde317cfec", // Ubuntu 22.04 LTS
		"us-east-2":      "ami-05fb0b8c1424f266b",
		"us-west-1":      "ami-0ce2cb35386fc22e9",
		"us-west-2":      "ami-008fe2fc65df48dac",
		"eu-west-1":      "ami-0905a3c97561e0b69",
		"eu-west-2":      "ami-0e5f882be1900e43b",
		"eu-west-3":      "ami-01d21b7be69801c2f",
		"eu-central-1":   "ami-0faab6bdbac9486fb",
		"ap-southeast-1": "ami-078c1149d8ad719a7",
		"ap-southeast-2": "ami-04f5097681773b989",
		"ap-northeast-1": "ami-07c589821f2b353aa",
		"sa-east-1":      "ami-0fb4cf3a99aa89f72",
	}

	ami, ok := ubuntuAMIs[region]
	if !ok {
		// Fallback: use us-east-1 AMI and let AWS handle region compatibility
		ctx.Log.Warn(fmt.Sprintf("No pre-configured AMI for region %s, using us-east-1 default", region), nil)
		return ubuntuAMIs["us-east-1"], nil
	}

	return ami, nil
}

// createHetznerNode creates a Hetzner Cloud server
func createHetznerNode(ctx *pulumi.Context, name string, nodeConfig *config.NodeConfig, sshKeyOutput pulumi.StringOutput, bastionEnabled bool, saltMasterIP string, component *RealNodeComponent, extras map[string]interface{}) error {
	// Get provider-specific config
	hetznerConfig, _ := extras["hetznerConfig"].(*config.HetznerProvider)
	bastionComponent, _ := extras["bastionComponent"].(*BastionComponent)

	// Determine server type
	serverType := nodeConfig.Size
	if serverType == "" {
		serverType = "cpx22" // Default: 2 vCPU, 4GB RAM (AMD shared)
	}

	// Determine image
	image := nodeConfig.Image
	if image == "" {
		image = "ubuntu-22.04"
	}

	// Determine location
	location := nodeConfig.Region
	if location == "" && hetznerConfig != nil {
		location = hetznerConfig.Location
	}
	if location == "" {
		location = "nbg1" // Default to Nuremberg (more reliable than fsn1)
	}

	// Reuse bastion's Hetzner provider and SSH key if available (avoids duplicate key error)
	if hetznerProvider == nil && bastionComponent != nil && bastionComponent.HetznerProvider != nil {
		hetznerProvider = bastionComponent.HetznerProvider
		hetznerSshKey = bastionComponent.HetznerSSHKey
		ctx.Log.Info("   ✅ Reusing bastion's Hetzner provider and SSH key for cluster nodes", nil)
	}

	// Create shared Hetzner provider with token (only once for all instances)
	if hetznerProvider == nil {
		hetznerToken := os.Getenv("HETZNER_TOKEN")
		if hetznerToken == "" {
			return fmt.Errorf("HETZNER_TOKEN environment variable is required for Hetzner nodes")
		}

		provider, err := hcloud.NewProvider(ctx, fmt.Sprintf("%s-hetzner-node-provider", ctx.Stack()), &hcloud.ProviderArgs{
			Token: pulumi.String(hetznerToken),
		})
		if err != nil {
			return fmt.Errorf("failed to create Hetzner provider: %w", err)
		}
		hetznerProvider = provider

		// Create shared SSH key for all Hetzner nodes
		sshKey, err := hcloud.NewSshKey(ctx, fmt.Sprintf("%s-hetzner-node-ssh-key", ctx.Stack()), &hcloud.SshKeyArgs{
			Name:      pulumi.Sprintf("sloth-k8s-nodes-%s", ctx.Stack()),
			PublicKey: sshKeyOutput,
			Labels: pulumi.StringMap{
				"managed": pulumi.String("sloth-kubernetes"),
				"stack":   pulumi.String(ctx.Stack()),
			},
		}, pulumi.Provider(hetznerProvider))
		if err != nil {
			return fmt.Errorf("failed to create Hetzner SSH key: %w", err)
		}
		hetznerSshKey = sshKey

		ctx.Log.Info("   ✅ Hetzner provider and SSH key created for cluster nodes", nil)
	}

	// Generate cloud-init user data
	userDataScript := cloudinit.GenerateUserDataWithHostnameAndSalt(name, saltMasterIP)

	// Build labels
	labels := pulumi.StringMap{
		"managed": pulumi.String("sloth-kubernetes"),
		"role":    pulumi.String(strings.Join(nodeConfig.Roles, "-")),
		"stack":   pulumi.String(ctx.Stack()),
	}
	if nodeConfig.Labels != nil {
		for k, v := range nodeConfig.Labels {
			labels[k] = pulumi.String(v)
		}
	}

	if bastionEnabled {
		ctx.Log.Info(fmt.Sprintf("🔒 Creating Hetzner server %s (SSH restricted to bastion only)", nodeConfig.Name), nil)
	} else {
		ctx.Log.Info(fmt.Sprintf("🌍 Creating PUBLIC Hetzner server %s", nodeConfig.Name), nil)
	}

	// Build server args with shared SSH key
	serverArgs := &hcloud.ServerArgs{
		Name:       pulumi.String(name),
		ServerType: pulumi.String(serverType),
		Image:      pulumi.String(image),
		Location:   pulumi.String(location),
		UserData:   pulumi.String(userDataScript),
		Labels:     labels,
		SshKeys: pulumi.StringArray{
			hetznerSshKey.ID().ToStringOutput(),
		},
		PublicNets: hcloud.ServerPublicNetArray{
			&hcloud.ServerPublicNetArgs{
				Ipv4Enabled: pulumi.Bool(true),
				Ipv6Enabled: pulumi.Bool(true),
			},
		},
	}

	// Create the server with explicit provider
	server, err := hcloud.NewServer(ctx, name, serverArgs, pulumi.Parent(component), pulumi.Provider(hetznerProvider))
	if err != nil {
		return fmt.Errorf("failed to create Hetzner server %s: %w", name, err)
	}

	// Set component outputs
	component.PublicIP = server.Ipv4Address
	component.PrivateIP = server.Ipv4Address // Hetzner uses public IP as primary
	component.NodeName = pulumi.String(name).ToStringOutput()
	component.Provider = pulumi.String("hetzner").ToStringOutput()
	component.WireGuardIP = pulumi.String(nodeConfig.WireGuardIP).ToStringOutput()
	component.Region = pulumi.String(location).ToStringOutput()
	component.Size = pulumi.String(serverType).ToStringOutput()
	component.Status = server.Status

	// Export outputs
	secrets.Export(ctx,fmt.Sprintf("%s_id", name), server.ID())
	secrets.Export(ctx,fmt.Sprintf("%s_public_ip", name), server.Ipv4Address)
	secrets.Export(ctx,fmt.Sprintf("%s_status", name), server.Status)

	ctx.Log.Info(fmt.Sprintf("   ✅ Hetzner server %s created (%s) in %s", name, serverType, location), nil)

	return nil
}
