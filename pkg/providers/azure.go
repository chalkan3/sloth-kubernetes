package providers

import (
	"encoding/base64"
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	azurecompute "github.com/pulumi/pulumi-azure-native-sdk/compute/v2"
	azurenetwork "github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	azureresources "github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AzureProvider implements the Provider interface for Microsoft Azure
type AzureProvider struct {
	config          *config.AzureProvider
	resourceGroup   *azureresources.ResourceGroup
	virtualNetwork  *azurenetwork.VirtualNetwork
	subnet          *azurenetwork.Subnet
	securityGroup   *azurenetwork.NetworkSecurityGroup
	nodes           []*NodeOutput
	ctx             *pulumi.Context
	resourceGroupID pulumi.IDOutput
}

// NewAzureProvider creates a new Azure provider
func NewAzureProvider() *AzureProvider {
	return &AzureProvider{
		nodes: make([]*NodeOutput, 0),
	}
}

// GetName returns the provider name
func (p *AzureProvider) GetName() string {
	return "azure"
}

// Initialize initializes the Azure provider
func (p *AzureProvider) Initialize(ctx *pulumi.Context, config *config.ClusterConfig) error {
	p.ctx = ctx

	if config.Providers.Azure == nil || !config.Providers.Azure.Enabled {
		return fmt.Errorf("Azure provider is not enabled")
	}

	p.config = config.Providers.Azure

	ctx.Log.Info("Azure provider initialized", nil)
	return nil
}

// CreateNode creates an Azure Virtual Machine
func (p *AzureProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	// Ensure resource group exists
	if p.resourceGroup == nil {
		return nil, fmt.Errorf("resource group not created - call CreateNetwork first")
	}

	// Ensure network exists
	if p.subnet == nil {
		return nil, fmt.Errorf("subnet not created - call CreateNetwork first")
	}

	location := p.config.Location
	if node.Region != "" {
		location = node.Region
	}

	// Generate user data script
	userData := p.generateUserData(node)
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userData))

	// Create Public IP
	publicIPName := fmt.Sprintf("%s-pip", node.Name)
	publicIP, err := azurenetwork.NewPublicIPAddress(ctx, publicIPName, &azurenetwork.PublicIPAddressArgs{
		ResourceGroupName:        p.resourceGroup.Name,
		Location:                 pulumi.String(location),
		PublicIpAddressName:      pulumi.String(publicIPName),
		PublicIPAllocationMethod: pulumi.String("Static"),
		Sku: &azurenetwork.PublicIPAddressSkuArgs{
			Name: pulumi.String("Standard"),
		},
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
			"Name":        pulumi.String(node.Name),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create public IP %s: %w", publicIPName, err)
	}

	// Create Network Interface
	nicName := fmt.Sprintf("%s-nic", node.Name)
	nic, err := azurenetwork.NewNetworkInterface(ctx, nicName, &azurenetwork.NetworkInterfaceArgs{
		ResourceGroupName:    p.resourceGroup.Name,
		Location:             pulumi.String(location),
		NetworkInterfaceName: pulumi.String(nicName),
		IpConfigurations: azurenetwork.NetworkInterfaceIPConfigurationArray{
			&azurenetwork.NetworkInterfaceIPConfigurationArgs{
				Name:                      pulumi.String("ipconfig1"),
				PrivateIPAllocationMethod: pulumi.String("Dynamic"),
				Subnet: &azurenetwork.SubnetTypeArgs{
					Id: p.subnet.ID(),
				},
				PublicIPAddress: &azurenetwork.PublicIPAddressTypeArgs{
					Id: publicIP.ID(),
				},
			},
		},
		NetworkSecurityGroup: &azurenetwork.NetworkSecurityGroupTypeArgs{
			Id: p.securityGroup.ID(),
		},
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
			"Name":        pulumi.String(node.Name),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network interface %s: %w", nicName, err)
	}

	// Prepare tags
	tags := pulumi.StringMap{
		"kubernetes":  pulumi.String("true"),
		"cluster":     pulumi.String(ctx.Stack()),
		"Environment": pulumi.String("production"),
		"ManagedBy":   pulumi.String("sloth-kubernetes"),
	}

	// Add role tags
	for _, role := range node.Roles {
		tags[fmt.Sprintf("role-%s", role)] = pulumi.String("true")
	}

	// Add custom labels as tags
	for k, v := range node.Labels {
		tags[k] = pulumi.String(v)
	}

	// Create Virtual Machine
	vmSize := node.Size
	if vmSize == "" {
		vmSize = "Standard_B2s" // Default size
	}

	// Map common image names to Azure publisher/offer/sku
	imageReference := p.getImageReference(node.Image)

	vmArgs := &azurecompute.VirtualMachineArgs{
		ResourceGroupName: p.resourceGroup.Name,
		Location:          pulumi.String(location),
		VmName:            pulumi.String(node.Name),
		NetworkProfile: &azurecompute.NetworkProfileArgs{
			NetworkInterfaces: azurecompute.NetworkInterfaceReferenceArray{
				&azurecompute.NetworkInterfaceReferenceArgs{
					Id:      nic.ID(),
					Primary: pulumi.Bool(true),
				},
			},
		},
		HardwareProfile: &azurecompute.HardwareProfileArgs{
			VmSize: pulumi.String(vmSize),
		},
		OsProfile: &azurecompute.OSProfileArgs{
			ComputerName:  pulumi.String(node.Name),
			AdminUsername: pulumi.String("azureuser"),
			AdminPassword: pulumi.String(generateSecurePassword()), // Required but we use SSH keys
			CustomData:    pulumi.String(userDataEncoded),
			LinuxConfiguration: &azurecompute.LinuxConfigurationArgs{
				DisablePasswordAuthentication: pulumi.Bool(true),
				Ssh: &azurecompute.SshConfigurationArgs{
					PublicKeys: azurecompute.SshPublicKeyTypeArray{
						&azurecompute.SshPublicKeyTypeArgs{
							KeyData: pulumi.String(p.config.SSHPublicKey),
							Path:    pulumi.String("/home/azureuser/.ssh/authorized_keys"),
						},
					},
				},
			},
		},
		StorageProfile: &azurecompute.StorageProfileArgs{
			ImageReference: imageReference,
			OsDisk: &azurecompute.OSDiskArgs{
				Name:         pulumi.String(fmt.Sprintf("%s-osdisk", node.Name)),
				CreateOption: pulumi.String("FromImage"),
				ManagedDisk: &azurecompute.ManagedDiskParametersArgs{
					StorageAccountType: pulumi.String("Premium_LRS"),
				},
				DiskSizeGB: pulumi.Int(30),
			},
		},
		Tags: tags,
	}

	vm, err := azurecompute.NewVirtualMachine(ctx, node.Name, vmArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM %s: %w", node.Name, err)
	}

	// Create node output
	output := &NodeOutput{
		ID:          vm.ID(),
		Name:        node.Name,
		PublicIP:    publicIP.IpAddress.Elem(),
		PrivateIP:   nic.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress().Elem(),
		Provider:    "azure",
		Region:      location,
		Size:        vmSize,
		Status:      pulumi.String("active").ToStringOutput(),
		Labels:      node.Labels,
		WireGuardIP: node.WireGuardIP,
		SSHUser:     "azureuser",
		SSHKeyPath:  "~/.ssh/id_rsa",
	}

	// Export node information
	secrets.Export(ctx,fmt.Sprintf("%s_public_ip", node.Name), publicIP.IpAddress)
	secrets.Export(ctx,fmt.Sprintf("%s_private_ip", node.Name), nic.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress())
	secrets.Export(ctx,fmt.Sprintf("%s_id", node.Name), vm.ID())

	p.nodes = append(p.nodes, output)
	return output, nil
}

// getImageReference returns the Azure image reference based on image name
func (p *AzureProvider) getImageReference(imageName string) *azurecompute.ImageReferenceArgs {
	// Default to Ubuntu 22.04 LTS
	defaultImage := &azurecompute.ImageReferenceArgs{
		Publisher: pulumi.String("Canonical"),
		Offer:     pulumi.String("0001-com-ubuntu-server-jammy"),
		Sku:       pulumi.String("22_04-lts-gen2"),
		Version:   pulumi.String("latest"),
	}

	// Map common image names
	imageMap := map[string]*azurecompute.ImageReferenceArgs{
		"Ubuntu 22.04 LTS": defaultImage,
		"ubuntu-22.04":     defaultImage,
		"Ubuntu 24.04 LTS": {
			Publisher: pulumi.String("Canonical"),
			Offer:     pulumi.String("ubuntu-24_04-lts"),
			Sku:       pulumi.String("server"),
			Version:   pulumi.String("latest"),
		},
		"ubuntu-24.04": {
			Publisher: pulumi.String("Canonical"),
			Offer:     pulumi.String("ubuntu-24_04-lts"),
			Sku:       pulumi.String("server"),
			Version:   pulumi.String("latest"),
		},
	}

	if image, ok := imageMap[imageName]; ok {
		return image
	}

	// Return default if not found
	return defaultImage
}

// CreateNodePool creates multiple nodes
func (p *AzureProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
	outputs := make([]*NodeOutput, 0, pool.Count)

	for i := 0; i < pool.Count; i++ {
		nodeName := fmt.Sprintf("%s-%d", pool.Name, i+1)

		// Determine zone/region
		region := pool.Region
		if len(pool.Zones) > 0 {
			// Distribute across zones
			region = pool.Zones[i%len(pool.Zones)]
		}

		// Create node config from pool
		nodeConfig := &config.NodeConfig{
			Name:       nodeName,
			Provider:   pool.Provider,
			Pool:       pool.Name,
			Roles:      pool.Roles,
			Size:       pool.Size,
			Image:      pool.Image,
			Region:     region,
			Labels:     pool.Labels,
			Taints:     pool.Taints,
			UserData:   pool.UserData,
			Monitoring: true,
		}

		// Set WireGuard IP based on role and index
		// Azure Master: 10.8.0.20
		// Azure Workers: 10.8.0.21, 10.8.0.22
		if contains(pool.Roles, "controlplane") || contains(pool.Roles, "master") {
			nodeConfig.WireGuardIP = "10.8.0.20"
		} else if contains(pool.Roles, "worker") {
			nodeConfig.WireGuardIP = fmt.Sprintf("10.8.0.%d", 21+i)
		}

		output, err := p.CreateNode(ctx, nodeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", nodeName, err)
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// CreateNetwork creates Azure network infrastructure (VNet, Subnet, NSG)
func (p *AzureProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	location := p.config.Location

	// Create Resource Group
	resourceGroupName := p.config.ResourceGroup
	if resourceGroupName == "" {
		resourceGroupName = fmt.Sprintf("%s-rg", ctx.Stack())
	}

	rg, err := azureresources.NewResourceGroup(ctx, resourceGroupName, &azureresources.ResourceGroupArgs{
		ResourceGroupName: pulumi.String(resourceGroupName),
		Location:          pulumi.String(location),
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
			"Cluster":     pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create resource group: %w", err)
	}

	p.resourceGroup = rg
	p.resourceGroupID = rg.ID()

	// Get VNet configuration
	vnetConfig := p.config.VirtualNetwork
	if vnetConfig == nil {
		vnetConfig = &config.AzureVirtualNetwork{
			Create: true,
			Name:   fmt.Sprintf("%s-vnet", ctx.Stack()),
			CIDR:   network.CIDR,
		}
	}

	if vnetConfig.CIDR == "" {
		vnetConfig.CIDR = network.CIDR
	}

	// Create Virtual Network
	vnet, err := azurenetwork.NewVirtualNetwork(ctx, vnetConfig.Name, &azurenetwork.VirtualNetworkArgs{
		ResourceGroupName:  rg.Name,
		Location:           pulumi.String(location),
		VirtualNetworkName: pulumi.String(vnetConfig.Name),
		AddressSpace: &azurenetwork.AddressSpaceArgs{
			AddressPrefixes: pulumi.StringArray{
				pulumi.String(vnetConfig.CIDR),
			},
		},
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual network: %w", err)
	}

	p.virtualNetwork = vnet

	// Create Subnet (use first /24 from VNet CIDR)
	subnetName := fmt.Sprintf("%s-subnet", ctx.Stack())
	subnetCIDR := calculateSubnetCIDR(vnetConfig.CIDR)

	subnet, err := azurenetwork.NewSubnet(ctx, subnetName, &azurenetwork.SubnetArgs{
		ResourceGroupName:  rg.Name,
		VirtualNetworkName: vnet.Name,
		SubnetName:         pulumi.String(subnetName),
		AddressPrefix:      pulumi.String(subnetCIDR),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	p.subnet = subnet

	// Create Network Security Group
	nsgName := fmt.Sprintf("%s-nsg", ctx.Stack())
	nsg, err := azurenetwork.NewNetworkSecurityGroup(ctx, nsgName, &azurenetwork.NetworkSecurityGroupArgs{
		ResourceGroupName:        rg.Name,
		Location:                 pulumi.String(location),
		NetworkSecurityGroupName: pulumi.String(nsgName),
		SecurityRules: azurenetwork.SecurityRuleTypeArray{
			// SSH from anywhere (restrict in production!)
			&azurenetwork.SecurityRuleTypeArgs{
				Name:                     pulumi.String("allow-ssh"),
				Priority:                 pulumi.Int(100),
				Direction:                pulumi.String("Inbound"),
				Access:                   pulumi.String("Allow"),
				Protocol:                 pulumi.String("Tcp"),
				SourcePortRange:          pulumi.String("*"),
				DestinationPortRange:     pulumi.String("22"),
				SourceAddressPrefix:      pulumi.String("*"),
				DestinationAddressPrefix: pulumi.String("*"),
			},
			// WireGuard VPN
			&azurenetwork.SecurityRuleTypeArgs{
				Name:                     pulumi.String("allow-wireguard"),
				Priority:                 pulumi.Int(110),
				Direction:                pulumi.String("Inbound"),
				Access:                   pulumi.String("Allow"),
				Protocol:                 pulumi.String("Udp"),
				SourcePortRange:          pulumi.String("*"),
				DestinationPortRange:     pulumi.String("51820"),
				SourceAddressPrefix:      pulumi.String("*"),
				DestinationAddressPrefix: pulumi.String("*"),
			},
			// Kubernetes API
			&azurenetwork.SecurityRuleTypeArgs{
				Name:                     pulumi.String("allow-k8s-api"),
				Priority:                 pulumi.Int(120),
				Direction:                pulumi.String("Inbound"),
				Access:                   pulumi.String("Allow"),
				Protocol:                 pulumi.String("Tcp"),
				SourcePortRange:          pulumi.String("*"),
				DestinationPortRange:     pulumi.String("6443"),
				SourceAddressPrefix:      pulumi.String("*"),
				DestinationAddressPrefix: pulumi.String("*"),
			},
			// Allow all internal traffic
			&azurenetwork.SecurityRuleTypeArgs{
				Name:                     pulumi.String("allow-internal"),
				Priority:                 pulumi.Int(200),
				Direction:                pulumi.String("Inbound"),
				Access:                   pulumi.String("Allow"),
				Protocol:                 pulumi.String("*"),
				SourcePortRange:          pulumi.String("*"),
				DestinationPortRange:     pulumi.String("*"),
				SourceAddressPrefix:      pulumi.String(vnetConfig.CIDR),
				DestinationAddressPrefix: pulumi.String("*"),
			},
		},
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network security group: %w", err)
	}

	p.securityGroup = nsg

	// Create output
	output := &NetworkOutput{
		ID:     vnet.ID(),
		Name:   vnetConfig.Name,
		CIDR:   vnetConfig.CIDR,
		Region: location,
	}

	// Export network info
	secrets.Export(ctx,"azure_resource_group_id", rg.ID())
	secrets.Export(ctx,"azure_resource_group_name", rg.Name)
	secrets.Export(ctx,"azure_vnet_id", vnet.ID())
	secrets.Export(ctx,"azure_vnet_name", vnet.Name)
	secrets.Export(ctx,"azure_subnet_id", subnet.ID())
	secrets.Export(ctx,"azure_nsg_id", nsg.ID())

	return output, nil
}

// calculateSubnetCIDR calculates a /24 subnet from a larger CIDR
func calculateSubnetCIDR(vnetCIDR string) string {
	// Simple implementation: take first /24
	// Example: 10.14.0.0/16 -> 10.14.1.0/24
	// For production, use proper IP calculation library
	if len(vnetCIDR) > 0 {
		// Extract base IP (e.g., "10.14" from "10.14.0.0/16")
		parts := vnetCIDR[:len(vnetCIDR)-3] // Remove "/16"
		lastDot := len(parts) - 1
		for lastDot >= 0 && parts[lastDot] != '.' {
			lastDot--
		}
		if lastDot > 0 {
			base := parts[:lastDot]
			return fmt.Sprintf("%s.1.0/24", base)
		}
	}
	return "10.14.1.0/24" // Default fallback
}

// CreateFirewall creates firewall rules (uses NSG)
func (p *AzureProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	// Azure uses Network Security Groups (NSG) for firewalling
	// NSG is already created in CreateNetwork and attached to NICs
	// Additional rules can be added here if needed

	ctx.Log.Info("Azure firewall rules configured via Network Security Group", nil)
	return nil
}

// CreateLoadBalancer creates an Azure Load Balancer
func (p *AzureProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	// Azure Load Balancer implementation
	// This is a placeholder - implement when needed
	return nil, fmt.Errorf("Azure Load Balancer not yet implemented")
}

// GetRegions returns available Azure regions
func (p *AzureProvider) GetRegions() []string {
	return []string{
		"eastus", "eastus2", "westus", "westus2", "westus3",
		"centralus", "northcentralus", "southcentralus",
		"northeurope", "westeurope", "uksouth", "ukwest",
		"francecentral", "germanywestcentral", "norwayeast",
		"switzerlandnorth", "swedencentral",
		"southeastasia", "eastasia", "australiaeast",
		"japaneast", "japanwest", "koreacentral",
		"southindia", "centralindia", "brazilsouth",
		"canadacentral", "canadaeast",
	}
}

// GetSizes returns available Azure VM sizes
func (p *AzureProvider) GetSizes() []string {
	return []string{
		"Standard_B1s", "Standard_B1ms", "Standard_B2s", "Standard_B2ms",
		"Standard_B4ms", "Standard_B8ms",
		"Standard_D2s_v3", "Standard_D4s_v3", "Standard_D8s_v3",
		"Standard_D16s_v3", "Standard_D32s_v3",
		"Standard_E2s_v3", "Standard_E4s_v3", "Standard_E8s_v3",
		"Standard_F2s_v2", "Standard_F4s_v2", "Standard_F8s_v2",
	}
}

// Cleanup performs cleanup operations
func (p *AzureProvider) Cleanup(ctx *pulumi.Context) error {
	// Cleanup is handled by Pulumi's resource management
	return nil
}

// generateUserData generates the cloud-init script for the VM
func (p *AzureProvider) generateUserData(node *config.NodeConfig) string {
	// Base user data with Docker and WireGuard
	baseScript := `#!/bin/bash
set -e

# Update system
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y

# Install required packages
apt-get install -y \
    curl \
    wget \
    git \
    vim \
    htop \
    net-tools \
    wireguard \
    wireguard-tools

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
usermod -aG docker azureuser

# Enable Docker service
systemctl enable docker
systemctl start docker

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/
kubectl version --client

# Disable swap (required for Kubernetes)
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Enable IP forwarding
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf
sysctl -p

# Configure WireGuard directory
mkdir -p /etc/wireguard
chmod 700 /etc/wireguard

# Generate WireGuard keys
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey
chmod 600 /etc/wireguard/privatekey

# Set node labels
echo "NODE_PROVIDER=azure" >> /etc/environment
echo "NODE_REGION=%s" >> /etc/environment
echo "NODE_SIZE=%s" >> /etc/environment
`

	// Add role-specific configuration
	for _, role := range node.Roles {
		baseScript += fmt.Sprintf("echo 'NODE_ROLE_%s=true' >> /etc/environment\n", role)
	}

	// Add custom user data if provided
	if node.UserData != "" {
		baseScript += "\n# Custom user data\n"
		baseScript += node.UserData
	} else if p.config.UserData != "" {
		baseScript += "\n# Provider custom user data\n"
		baseScript += p.config.UserData
	}

	baseScript += "\necho 'Azure node initialization complete'\n"

	return fmt.Sprintf(baseScript, node.Region, node.Size)
}
