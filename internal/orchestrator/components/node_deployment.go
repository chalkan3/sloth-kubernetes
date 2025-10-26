package components

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/cloudinit"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi-azure-native-sdk/compute/v2"
	aznetwork "github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	"github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
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

	// Create ONE shared SSH key for all DigitalOcean Droplets (DO doesn't allow duplicate keys)
	sharedDOSshKey, err := digitalocean.NewSshKey(ctx, fmt.Sprintf("%s-shared-key", name), &digitalocean.SshKeyArgs{
		Name:      pulumi.Sprintf("kubernetes-cluster-production-key"),
		PublicKey: sshKeyOutput,
	}, pulumi.Parent(component))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create shared DO SSH key: %w", err)
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

	// Create real cloud resource based on provider
	if nodeConfig.Provider == "digitalocean" {
		err = createDigitalOceanDroplet(ctx, name, nodeConfig, sharedDOSshKey, doToken, vpcComponent, bastionEnabled, saltMasterIP, component)
	} else if nodeConfig.Provider == "linode" {
		err = createLinodeInstance(ctx, name, nodeConfig, sshKeyOutput, sharedLinodeStackscript, linodeToken, bastionEnabled, saltMasterIP, component)
	} else if nodeConfig.Provider == "azure" {
		err = createAzureVM(ctx, name, nodeConfig, sshKeyOutput, bastionEnabled, saltMasterIP, component)
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
	azureResourceGroup *resources.ResourceGroup
	azureVNet          *aznetwork.VirtualNetwork
	azureSubnet        *aznetwork.Subnet
	azureNSG           *aznetwork.NetworkSecurityGroup
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
		rg, err := resources.NewResourceGroup(ctx, rgName, &resources.ResourceGroupArgs{
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
		vnet, err := aznetwork.NewVirtualNetwork(ctx, vnetName, &aznetwork.VirtualNetworkArgs{
			ResourceGroupName:  rg.Name,
			Location:           pulumi.String(location),
			VirtualNetworkName: pulumi.String(vnetName),
			AddressSpace: &aznetwork.AddressSpaceArgs{
				AddressPrefixes: pulumi.StringArray{pulumi.String("10.14.0.0/16")},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Azure VNet: %w", err)
		}
		azureVNet = vnet

		// Create Subnet
		subnetName := "sloth-k8s-subnet"
		subnet, err := aznetwork.NewSubnet(ctx, subnetName, &aznetwork.SubnetArgs{
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
		nsg, err := aznetwork.NewNetworkSecurityGroup(ctx, nsgName, &aznetwork.NetworkSecurityGroupArgs{
			ResourceGroupName:        rg.Name,
			Location:                 pulumi.String(location),
			NetworkSecurityGroupName: pulumi.String(nsgName),
			SecurityRules: aznetwork.SecurityRuleTypeArray{
				&aznetwork.SecurityRuleTypeArgs{
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
				&aznetwork.SecurityRuleTypeArgs{
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
				&aznetwork.SecurityRuleTypeArgs{
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
	publicIP, err := aznetwork.NewPublicIPAddress(ctx, publicIPName, &aznetwork.PublicIPAddressArgs{
		ResourceGroupName:        azureResourceGroup.Name,
		Location:                 pulumi.String(location),
		PublicIpAddressName:      pulumi.String(publicIPName),
		PublicIPAllocationMethod: pulumi.String("Static"),
		Sku: &aznetwork.PublicIPAddressSkuArgs{
			Name: pulumi.String("Standard"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create public IP: %w", err)
	}

	// Create Network Interface
	nicName := fmt.Sprintf("%s-nic", nodeConfig.Name)
	nic, err := aznetwork.NewNetworkInterface(ctx, nicName, &aznetwork.NetworkInterfaceArgs{
		ResourceGroupName:   azureResourceGroup.Name,
		Location:            pulumi.String(location),
		NetworkInterfaceName: pulumi.String(nicName),
		IpConfigurations: aznetwork.NetworkInterfaceIPConfigurationArray{
			&aznetwork.NetworkInterfaceIPConfigurationArgs{
				Name:                      pulumi.String("ipconfig1"),
				PrivateIPAllocationMethod: pulumi.String("Dynamic"),
				Subnet: &aznetwork.SubnetTypeArgs{
					Id: azureSubnet.ID(),
				},
				PublicIPAddress: &aznetwork.PublicIPAddressTypeArgs{
					Id: publicIP.ID(),
				},
			},
		},
		NetworkSecurityGroup: &aznetwork.NetworkSecurityGroupTypeArgs{
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
	imageReference := &compute.ImageReferenceArgs{
		Publisher: pulumi.String("Canonical"),
		Offer:     pulumi.String("0001-com-ubuntu-server-jammy"),
		Sku:       pulumi.String("22_04-lts-gen2"),
		Version:   pulumi.String("latest"),
	}

	// Generate a secure password (required by Azure but we use SSH keys)
	adminPassword := generateSecurePassword()

	// Create Virtual Machine
	vmArgs := &compute.VirtualMachineArgs{
		ResourceGroupName: azureResourceGroup.Name,
		Location:          pulumi.String(location),
		VmName:            pulumi.String(nodeConfig.Name),
		NetworkProfile: &compute.NetworkProfileArgs{
			NetworkInterfaces: compute.NetworkInterfaceReferenceArray{
				&compute.NetworkInterfaceReferenceArgs{
					Id:      nic.ID(),
					Primary: pulumi.Bool(true),
				},
			},
		},
		HardwareProfile: &compute.HardwareProfileArgs{
			VmSize: pulumi.String(nodeConfig.Size),
		},
		OsProfile: &compute.OSProfileArgs{
			ComputerName:  pulumi.String(nodeConfig.Name),
			AdminUsername: pulumi.String("azureuser"),
			AdminPassword: pulumi.String(adminPassword),
			CustomData:    pulumi.String(userDataEncoded),
			LinuxConfiguration: &compute.LinuxConfigurationArgs{
				DisablePasswordAuthentication: pulumi.Bool(true),
				Ssh: &compute.SshConfigurationArgs{
					PublicKeys: compute.SshPublicKeyTypeArray{
						&compute.SshPublicKeyTypeArgs{
							KeyData: sshKeyOutput,
							Path:    pulumi.String("/home/azureuser/.ssh/authorized_keys"),
						},
					},
				},
			},
		},
		StorageProfile: &compute.StorageProfileArgs{
			ImageReference: imageReference,
			OsDisk: &compute.OSDiskArgs{
				Name:         pulumi.String(fmt.Sprintf("%s-osdisk", nodeConfig.Name)),
				CreateOption: pulumi.String("FromImage"),
				ManagedDisk: &compute.ManagedDiskParametersArgs{
					StorageAccountType: pulumi.String("Premium_LRS"),
				},
				DiskSizeGB: pulumi.Int(30),
			},
		},
	}

	vm, err := compute.NewVirtualMachine(ctx, nodeConfig.Name, vmArgs)
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
