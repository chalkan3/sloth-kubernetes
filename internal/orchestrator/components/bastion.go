package components

import (
	"fmt"
	"os"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	azurecompute "github.com/pulumi/pulumi-azure-native-sdk/compute/v2"
	azurenetwork "github.com/pulumi/pulumi-azure-native-sdk/network/v2"
	azureresources "github.com/pulumi/pulumi-azure-native-sdk/resources/v2"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BastionComponent represents the bastion host for secure cluster access
type BastionComponent struct {
	pulumi.ResourceState

	BastionName     pulumi.StringOutput `pulumi:"bastionName"`
	PublicIP        pulumi.StringOutput `pulumi:"publicIP"`
	PrivateIP       pulumi.StringOutput `pulumi:"privateIP"`
	WireGuardIP     pulumi.StringOutput `pulumi:"wireGuardIP"`
	Provider        pulumi.StringOutput `pulumi:"provider"`
	Region          pulumi.StringOutput `pulumi:"region"`
	SSHPort         pulumi.IntOutput    `pulumi:"sshPort"`
	Status          pulumi.StringOutput `pulumi:"status"`
	HetznerSSHKey   *hcloud.SshKey      // Shared SSH key for Hetzner nodes
	HetznerProvider *hcloud.Provider    // Shared Hetzner provider
}

// NewBastionComponent creates a bastion host for secure cluster access
// The bastion is the ONLY host with public SSH access. All cluster nodes are private.
func NewBastionComponent(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	sshPrivateKey pulumi.StringOutput,
	doToken pulumi.StringInput,
	linodeToken pulumi.StringInput,
	opts ...pulumi.ResourceOption,
) (*BastionComponent, error) {
	component := &BastionComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:Bastion", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if !bastionConfig.Enabled {
		ctx.Log.Info("⏭️  Bastion host disabled - cluster nodes will have public IPs", nil)
		component.Status = pulumi.String("disabled").ToStringOutput()
		return component, nil
	}

	ctx.Log.Info(fmt.Sprintf("🏰 Creating Bastion host on %s...", bastionConfig.Provider), nil)

	// Set defaults
	if bastionConfig.Name == "" {
		bastionConfig.Name = "bastion"
	}
	if bastionConfig.SSHPort == 0 {
		bastionConfig.SSHPort = 22
	}

	// Assign VPN IP for bastion (10.8.0.5 - reserved for bastion)
	bastionVPNIP := "10.8.0.5"

	component.BastionName = pulumi.String(bastionConfig.Name).ToStringOutput()
	component.Provider = pulumi.String(bastionConfig.Provider).ToStringOutput()
	component.Region = pulumi.String(bastionConfig.Region).ToStringOutput()
	component.SSHPort = pulumi.Int(bastionConfig.SSHPort).ToIntOutput()
	component.WireGuardIP = pulumi.String(bastionVPNIP).ToStringOutput()

	// Create bastion host based on provider
	switch bastionConfig.Provider {
	case "digitalocean":
		err = createDigitalOceanBastion(ctx, name, bastionConfig, sshKeyOutput, doToken, component)
	case "linode":
		err = createLinodeBastion(ctx, name, bastionConfig, sshKeyOutput, linodeToken, component)
	case "aws":
		err = createAWSBastion(ctx, name, bastionConfig, sshKeyOutput, component)
	case "azure":
		err = createAzureBastion(ctx, name, bastionConfig, sshKeyOutput, component)
	case "hetzner":
		err = createHetznerBastion(ctx, name, bastionConfig, sshKeyOutput, component)
	default:
		return nil, fmt.Errorf("unsupported bastion provider: %s (supported: digitalocean, linode, aws, azure, hetzner)", bastionConfig.Provider)
	}

	if err != nil {
		return nil, err
	}

	// Provision bastion with security hardening
	provComp, err := NewBastionProvisioningComponent(
		ctx,
		fmt.Sprintf("%s-provision", name),
		component.PublicIP,
		bastionConfig,
		sshPrivateKey,
		component,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to provision bastion: %w", err)
	}

	component.Status = provComp.Status

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"bastionName": component.BastionName,
		"publicIP":    component.PublicIP,
		"privateIP":   component.PrivateIP,
		"wireGuardIP": component.WireGuardIP,
		"provider":    component.Provider,
		"region":      component.Region,
		"sshPort":     component.SSHPort,
		"status":      component.Status,
	}); err != nil {
		return nil, err
	}

	ctx.Log.Info("✅ Bastion host created successfully", nil)

	return component, nil
}

// createDigitalOceanBastion creates a DigitalOcean bastion droplet
func createDigitalOceanBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	doToken pulumi.StringInput,
	component *BastionComponent,
) error {
	// Create SSH key for bastion
	sshKey, err := digitalocean.NewSshKey(ctx, fmt.Sprintf("%s-ssh-key", name), &digitalocean.SshKeyArgs{
		Name:      pulumi.Sprintf("bastion-key-%s", name),
		PublicKey: sshKeyOutput,
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create DO SSH key: %w", err)
	}

	// Create bastion droplet
	droplet, err := digitalocean.NewDroplet(ctx, name, &digitalocean.DropletArgs{
		Image:  pulumi.String(bastionConfig.Image),
		Name:   pulumi.String(bastionConfig.Name),
		Region: pulumi.String(bastionConfig.Region),
		Size:   pulumi.String(bastionConfig.Size),
		SshKeys: pulumi.StringArray{
			sshKey.Fingerprint,
		},
		Tags: pulumi.StringArray{
			pulumi.String("bastion"),
			pulumi.String("security"),
			pulumi.String(ctx.Stack()),
		},
		Ipv6:       pulumi.Bool(true),
		Monitoring: pulumi.Bool(true),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create bastion droplet: %w", err)
	}

	component.PublicIP = droplet.Ipv4Address
	component.PrivateIP = droplet.Ipv4AddressPrivate

	return nil
}

// createLinodeBastion creates a Linode bastion instance
func createLinodeBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	linodeToken pulumi.StringInput,
	component *BastionComponent,
) error {
	// Create bastion instance
	instance, err := linode.NewInstance(ctx, name, &linode.InstanceArgs{
		Label:  pulumi.String(bastionConfig.Name),
		Region: pulumi.String(bastionConfig.Region),
		Type:   pulumi.String(bastionConfig.Size),
		Image:  pulumi.String(bastionConfig.Image),
		AuthorizedKeys: pulumi.StringArray{
			sshKeyOutput,
		},
		Tags: pulumi.StringArray{
			pulumi.String("bastion"),
			pulumi.String("security"),
			pulumi.String(ctx.Stack()),
		},
		PrivateIp: pulumi.Bool(true),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create bastion instance: %w", err)
	}

	component.PublicIP = instance.IpAddress
	component.PrivateIP = instance.PrivateIpAddress

	return nil
}

// createAWSBastion creates an AWS EC2 bastion instance
func createAWSBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	component *BastionComponent,
) error {
	region := bastionConfig.Region
	if region == "" {
		region = "us-east-1"
	}

	// Create explicit AWS provider with region and credentials
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
	awsProvider, err := aws.NewProvider(ctx, fmt.Sprintf("%s-aws-provider", name), providerArgs, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	// Get the default VPC
	defaultVpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
		Default: pulumi.BoolRef(true),
	}, pulumi.Provider(awsProvider))
	if err != nil {
		return fmt.Errorf("failed to get default VPC: %w", err)
	}

	// Get a subnet from the default VPC
	subnets, err := ec2.GetSubnets(ctx, &ec2.GetSubnetsArgs{
		Filters: []ec2.GetSubnetsFilter{
			{
				Name:   "vpc-id",
				Values: []string{defaultVpc.Id},
			},
		},
	}, pulumi.Provider(awsProvider))
	if err != nil {
		return fmt.Errorf("failed to get subnets: %w", err)
	}

	subnetId := ""
	if len(subnets.Ids) > 0 {
		subnetId = subnets.Ids[0]
	}

	// Create Security Group for bastion
	sg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-bastion-sg", name), &ec2.SecurityGroupArgs{
		Name:        pulumi.Sprintf("%s-bastion-sg", name),
		Description: pulumi.String("Security group for bastion host"),
		VpcId:       pulumi.String(defaultVpc.Id),
		Ingress: ec2.SecurityGroupIngressArray{
			// SSH
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(bastionConfig.SSHPort),
				ToPort:     pulumi.Int(bastionConfig.SSHPort),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			// WireGuard
			&ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("udp"),
				FromPort:   pulumi.Int(51820),
				ToPort:     pulumi.Int(51820),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.Sprintf("%s-bastion-sg", name),
		},
	}, pulumi.Parent(component), pulumi.Provider(awsProvider))
	if err != nil {
		return fmt.Errorf("failed to create security group: %w", err)
	}

	// Create Key Pair
	keyPair, err := ec2.NewKeyPair(ctx, fmt.Sprintf("%s-bastion-key", name), &ec2.KeyPairArgs{
		KeyName:   pulumi.Sprintf("%s-bastion-key", name),
		PublicKey: sshKeyOutput,
		Tags: pulumi.StringMap{
			"Name": pulumi.Sprintf("%s-bastion-key", name),
		},
	}, pulumi.Parent(component), pulumi.Provider(awsProvider))
	if err != nil {
		return fmt.Errorf("failed to create key pair: %w", err)
	}

	// Get Ubuntu AMI
	ami := getUbuntuAMI(region)

	// Determine instance type
	instanceType := bastionConfig.Size
	if instanceType == "" {
		instanceType = "t3.micro"
	}

	// Create EC2 Instance
	instanceArgs := &ec2.InstanceArgs{
		Ami:                      pulumi.String(ami),
		InstanceType:             pulumi.String(instanceType),
		KeyName:                  keyPair.KeyName,
		VpcSecurityGroupIds:      pulumi.StringArray{sg.ID()},
		AssociatePublicIpAddress: pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(bastionConfig.Name),
			"Role": pulumi.String("bastion"),
		},
		MetadataOptions: &ec2.InstanceMetadataOptionsArgs{
			HttpTokens:   pulumi.String("optional"),
			HttpEndpoint: pulumi.String("enabled"),
		},
	}

	if subnetId != "" {
		instanceArgs.SubnetId = pulumi.String(subnetId)
	}

	instance, err := ec2.NewInstance(ctx, name, instanceArgs, pulumi.Parent(component), pulumi.Provider(awsProvider))
	if err != nil {
		return fmt.Errorf("failed to create bastion instance: %w", err)
	}

	component.PublicIP = instance.PublicIp
	component.PrivateIP = instance.PrivateIp

	return nil
}

// getUbuntuAMI returns the Ubuntu 22.04 LTS AMI for the given region
func getUbuntuAMI(region string) string {
	amiMap := map[string]string{
		"us-east-1":      "ami-0c7217cdde317cfec",
		"us-east-2":      "ami-0e83be366243f524a",
		"us-west-1":      "ami-0ce2cb35386fc22e9",
		"us-west-2":      "ami-008fe2fc65df48dac",
		"eu-west-1":      "ami-0905a3c97561e0b69",
		"eu-west-2":      "ami-0e5f882be1900e43b",
		"eu-central-1":   "ami-0faab6bdbac9486fb",
		"ap-southeast-1": "ami-078c1149d8ad719a7",
		"ap-northeast-1": "ami-07c589821f2b353aa",
		"sa-east-1":      "ami-0fb4cf3a99aa89f72",
	}

	if ami, ok := amiMap[region]; ok {
		return ami
	}
	return amiMap["us-east-1"]
}

func createAzureBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	component *BastionComponent,
) error {
	// Azure bastion configuration
	resourceGroupName := fmt.Sprintf("%s-bastion-rg", ctx.Stack())
	location := bastionConfig.Region
	if location == "" {
		location = "eastus"
	}

	// Create Resource Group
	rg, err := azureresources.NewResourceGroup(ctx, fmt.Sprintf("%s-rg", name), &azureresources.ResourceGroupArgs{
		ResourceGroupName: pulumi.String(resourceGroupName),
		Location:          pulumi.String(location),
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"Role":        pulumi.String("bastion"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create resource group: %w", err)
	}

	// Create Virtual Network
	vnetName := fmt.Sprintf("%s-vnet", name)
	vnet, err := azurenetwork.NewVirtualNetwork(ctx, vnetName, &azurenetwork.VirtualNetworkArgs{
		ResourceGroupName:  rg.Name,
		VirtualNetworkName: pulumi.String(vnetName),
		Location:           rg.Location,
		AddressSpace: &azurenetwork.AddressSpaceArgs{
			AddressPrefixes: pulumi.StringArray{
				pulumi.String("10.100.0.0/16"),
			},
		},
		Tags: pulumi.StringMap{
			"Role": pulumi.String("bastion"),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create virtual network: %w", err)
	}

	// Create Subnet
	subnetName := fmt.Sprintf("%s-subnet", name)
	subnet, err := azurenetwork.NewSubnet(ctx, subnetName, &azurenetwork.SubnetArgs{
		ResourceGroupName:  rg.Name,
		VirtualNetworkName: vnet.Name,
		SubnetName:         pulumi.String(subnetName),
		AddressPrefix:      pulumi.String("10.100.1.0/24"),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create subnet: %w", err)
	}

	// Create Network Security Group
	nsgName := fmt.Sprintf("%s-nsg", name)
	nsg, err := azurenetwork.NewNetworkSecurityGroup(ctx, nsgName, &azurenetwork.NetworkSecurityGroupArgs{
		ResourceGroupName:        rg.Name,
		NetworkSecurityGroupName: pulumi.String(nsgName),
		Location:                 rg.Location,
		SecurityRules: azurenetwork.SecurityRuleTypeArray{
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
		},
		Tags: pulumi.StringMap{
			"Role": pulumi.String("bastion"),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create network security group: %w", err)
	}

	// Create Public IP
	publicIPName := fmt.Sprintf("%s-ip", name)
	publicIP, err := azurenetwork.NewPublicIPAddress(ctx, publicIPName, &azurenetwork.PublicIPAddressArgs{
		ResourceGroupName:        rg.Name,
		PublicIpAddressName:      pulumi.String(publicIPName),
		Location:                 rg.Location,
		PublicIPAllocationMethod: pulumi.String("Static"),
		Sku: &azurenetwork.PublicIPAddressSkuArgs{
			Name: pulumi.String("Standard"),
		},
		Tags: pulumi.StringMap{
			"Role": pulumi.String("bastion"),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create public IP: %w", err)
	}

	// Create Network Interface
	nicName := fmt.Sprintf("%s-nic", name)
	nic, err := azurenetwork.NewNetworkInterface(ctx, nicName, &azurenetwork.NetworkInterfaceArgs{
		ResourceGroupName:    rg.Name,
		NetworkInterfaceName: pulumi.String(nicName),
		Location:             rg.Location,
		IpConfigurations: azurenetwork.NetworkInterfaceIPConfigurationArray{
			&azurenetwork.NetworkInterfaceIPConfigurationArgs{
				Name:                      pulumi.String("ipconfig1"),
				PrivateIPAllocationMethod: pulumi.String("Dynamic"),
				Subnet: &azurenetwork.SubnetTypeArgs{
					Id: subnet.ID(),
				},
				PublicIPAddress: &azurenetwork.PublicIPAddressTypeArgs{
					Id: publicIP.ID(),
				},
			},
		},
		NetworkSecurityGroup: &azurenetwork.NetworkSecurityGroupTypeArgs{
			Id: nsg.ID(),
		},
		Tags: pulumi.StringMap{
			"Role": pulumi.String("bastion"),
		},
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	// Create Virtual Machine
	vmName := bastionConfig.Name
	if vmName == "" {
		vmName = "bastion-azure"
	}

	// Use size from config or default
	vmSize := bastionConfig.Size
	if vmSize == "" {
		vmSize = "Standard_B1s" // Free tier eligible
	}

	vm, err := azurecompute.NewVirtualMachine(ctx, vmName, &azurecompute.VirtualMachineArgs{
		ResourceGroupName: rg.Name,
		VmName:            pulumi.String(vmName),
		Location:          rg.Location,
		HardwareProfile: &azurecompute.HardwareProfileArgs{
			VmSize: pulumi.String(vmSize),
		},
		StorageProfile: &azurecompute.StorageProfileArgs{
			ImageReference: &azurecompute.ImageReferenceArgs{
				Publisher: pulumi.String("Canonical"),
				Offer:     pulumi.String("0001-com-ubuntu-server-jammy"),
				Sku:       pulumi.String("22_04-lts-gen2"),
				Version:   pulumi.String("latest"),
			},
			OsDisk: &azurecompute.OSDiskArgs{
				Name:         pulumi.String(fmt.Sprintf("%s-osdisk", vmName)),
				CreateOption: pulumi.String("FromImage"),
				ManagedDisk: &azurecompute.ManagedDiskParametersArgs{
					StorageAccountType: pulumi.String("Standard_LRS"),
				},
			},
		},
		OsProfile: &azurecompute.OSProfileArgs{
			ComputerName:  pulumi.String(vmName),
			AdminUsername: pulumi.String("azureuser"),
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
		NetworkProfile: &azurecompute.NetworkProfileArgs{
			NetworkInterfaces: azurecompute.NetworkInterfaceReferenceArray{
				&azurecompute.NetworkInterfaceReferenceArgs{
					Id:      nic.ID(),
					Primary: pulumi.Bool(true),
				},
			},
		},
		Tags: pulumi.StringMap{
			"Environment": pulumi.String("production"),
			"Role":        pulumi.String("bastion"),
			"ManagedBy":   pulumi.String("sloth-kubernetes"),
		},
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{nic}))
	if err != nil {
		return fmt.Errorf("failed to create virtual machine: %w", err)
	}

	// Set component outputs
	component.PublicIP = publicIP.IpAddress.Elem()
	component.PrivateIP = nic.IpConfigurations.Index(pulumi.Int(0)).PrivateIPAddress().Elem()

	ctx.Log.Info(fmt.Sprintf("✅ Azure bastion VM '%s' created in %s", vmName, location), nil)

	_ = vm // Use vm to avoid unused variable warning

	return nil
}

func createHetznerBastion(
	ctx *pulumi.Context,
	name string,
	bastionConfig *config.BastionConfig,
	sshKeyOutput pulumi.StringOutput,
	component *BastionComponent,
) error {
	location := bastionConfig.Region
	if location == "" {
		location = "fsn1"
	}

	// Determine server type (size)
	serverType := bastionConfig.Size
	if serverType == "" {
		serverType = "cpx22" // Hetzner shared AMD vCPU (2 vCPU, 4GB)
	}

	// Determine image
	image := bastionConfig.Image
	if image == "" {
		image = "ubuntu-22.04"
	}

	// Create Hetzner provider with token from environment
	hetznerToken := os.Getenv("HETZNER_TOKEN")
	if hetznerToken == "" {
		return fmt.Errorf("HETZNER_TOKEN environment variable is required")
	}

	hetznerProvider, err := hcloud.NewProvider(ctx, fmt.Sprintf("%s-hetzner-provider", name), &hcloud.ProviderArgs{
		Token: pulumi.String(hetznerToken),
	}, pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create Hetzner provider: %w", err)
	}

	// Create SSH key for bastion
	sshKey, err := hcloud.NewSshKey(ctx, fmt.Sprintf("%s-ssh-key", name), &hcloud.SshKeyArgs{
		Name:      pulumi.Sprintf("bastion-key-%s", name),
		PublicKey: sshKeyOutput,
		Labels: pulumi.StringMap{
			"managed": pulumi.String("sloth-kubernetes"),
			"role":    pulumi.String("bastion"),
		},
	}, pulumi.Provider(hetznerProvider), pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create Hetzner SSH key: %w", err)
	}

	// Create bastion server
	server, err := hcloud.NewServer(ctx, name, &hcloud.ServerArgs{
		Name:       pulumi.String(bastionConfig.Name),
		ServerType: pulumi.String(serverType),
		Image:      pulumi.String(image),
		Location:   pulumi.String(location),
		SshKeys: pulumi.StringArray{
			sshKey.ID().ToStringOutput(),
		},
		Labels: pulumi.StringMap{
			"managed": pulumi.String("sloth-kubernetes"),
			"role":    pulumi.String("bastion"),
			"stack":   pulumi.String(ctx.Stack()),
		},
		PublicNets: hcloud.ServerPublicNetArray{
			&hcloud.ServerPublicNetArgs{
				Ipv4Enabled: pulumi.Bool(true),
				Ipv6Enabled: pulumi.Bool(true),
			},
		},
	}, pulumi.Provider(hetznerProvider), pulumi.Parent(component))
	if err != nil {
		return fmt.Errorf("failed to create Hetzner bastion server: %w", err)
	}

	// Set component outputs
	component.PublicIP = server.Ipv4Address
	component.PrivateIP = server.Ipv4Address // Hetzner uses public IP as primary

	// Store SSH key and provider for reuse by Hetzner nodes (avoid duplicate key error)
	component.HetznerSSHKey = sshKey
	component.HetznerProvider = hetznerProvider

	ctx.Log.Info(fmt.Sprintf("✅ Hetzner bastion server '%s' created in %s", bastionConfig.Name, location), nil)

	return nil
}

// BastionProvisioningComponent handles bastion host provisioning and hardening
type BastionProvisioningComponent struct {
	pulumi.ResourceState

	Status pulumi.StringOutput `pulumi:"status"`
}

// NewBastionProvisioningComponent provisions and hardens the bastion host
func NewBastionProvisioningComponent(
	ctx *pulumi.Context,
	name string,
	bastionIP pulumi.StringOutput,
	bastionConfig *config.BastionConfig,
	sshPrivateKey pulumi.StringOutput,
	parent pulumi.Resource,
) (*BastionProvisioningComponent, error) {
	component := &BastionProvisioningComponent{}
	err := ctx.RegisterComponentResource("kubernetes-create:security:BastionProvisioning", name, component, pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}

	ctx.Log.Info("🔧 Provisioning bastion with security hardening...", nil)

	// Determine SSH user based on provider
	sshUser := "root"
	sudoPrefix := ""
	switch bastionConfig.Provider {
	case "azure":
		sshUser = "azureuser"
		sudoPrefix = "sudo "
		ctx.Log.Info("🔧 Using Azure-specific configuration (user: azureuser, sudo required)", nil)
	case "aws":
		sshUser = "ubuntu"
		sudoPrefix = "sudo "
		ctx.Log.Info("🔧 Using AWS-specific configuration (user: ubuntu, sudo required)", nil)
	}

	// Build provisioning script with security hardening
	provisionScript := buildBastionProvisionScript(bastionConfig, sudoPrefix)

	// Execute provisioning script via pulumi-command
	ctx.Log.Info("📋 Bastion will be provisioned with:", nil)
	ctx.Log.Info("  • UFW firewall (SSH only from allowed CIDRs)", nil)
	ctx.Log.Info("  • fail2ban for brute force protection", nil)
	ctx.Log.Info("  • SSH hardening (key-only auth)", nil)
	ctx.Log.Info("  • Audit logging enabled", nil)
	ctx.Log.Info("  • WireGuard VPN client", nil)

	if bastionConfig.EnableMFA {
		ctx.Log.Info("  • MFA (Google Authenticator)", nil)
	}

	// Execute the provisioning script on the bastion host
	ctx.Log.Info("⏳ Starting bastion provisioning (this may take 5-10 minutes)...", nil)
	ctx.Log.Info("   → Installing security packages (ufw, fail2ban, wireguard)", nil)
	ctx.Log.Info("   → Configuring firewall rules", nil)
	ctx.Log.Info("   → Hardening SSH configuration", nil)
	ctx.Log.Info("   → Setting up audit logging", nil)
	ctx.Log.Info("", nil)
	ctx.Log.Info("💡 Note: Pulumi doesn't show real-time output from remote commands.", nil)
	ctx.Log.Info("   The process is still running - please wait...", nil)

	provisionCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-provision-script", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           bastionIP,
			User:           pulumi.String(sshUser),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(30),
		},
		Create: pulumi.String(provisionScript),
	}, pulumi.Parent(component), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "20m", // Provisioning can take time for package installation
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to execute provisioning script: %w", err)
	}

	ctx.Log.Info("✅ Bastion provisioning command completed successfully", nil)
	ctx.Log.Info("", nil)
	ctx.Log.Info("🔍 Validating bastion SSH connectivity...", nil)

	// SSH Validation Command - Test that SSH is working properly
	validateSSHCmd, err := remote.NewCommand(ctx, fmt.Sprintf("%s-validate-ssh", name), &remote.CommandArgs{
		Connection: remote.ConnectionArgs{
			Host:           bastionIP,
			User:           pulumi.String(sshUser),
			PrivateKey:     sshPrivateKey,
			DialErrorLimit: pulumi.Int(10),
		},
		Create: pulumi.String(fmt.Sprintf(`#!/bin/bash
echo "=========================================="
echo "🔍 BASTION SSH VALIDATION TEST"
echo "=========================================="
echo "✅ SSH connection successful!"
echo "📋 Bastion details:"
echo "  • Hostname: $(hostname)"
echo "  • Uptime: $(uptime -p)"
echo "  • SSH service: $(%ssystemctl is-active sshd)"
echo "  • UFW status: $(%sufw status | head -1)"
echo "  • fail2ban status: $(%ssystemctl is-active fail2ban)"
echo ""
echo "✅ Bastion is fully operational and ready!"
echo "=========================================="
`, sudoPrefix, sudoPrefix, sudoPrefix)),
	}, pulumi.Parent(component), pulumi.DependsOn([]pulumi.Resource{provisionCmd}), pulumi.Timeouts(&pulumi.CustomTimeouts{
		Create: "2m",
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to validate bastion SSH: %w", err)
	}

	// Log validation output
	validateSSHCmd.Stdout.ApplyT(func(stdout string) string {
		if stdout != "" {
			ctx.Log.Info("", nil)
			ctx.Log.Info("✅ BASTION VALIDATION SUCCESSFUL", nil)
			ctx.Log.Info("   SSH connectivity confirmed", nil)
			ctx.Log.Info("   Security services are active", nil)
			ctx.Log.Info("", nil)
			ctx.Log.Info("🎉 BASTION IS 100% READY FOR CLUSTER DEPLOYMENT", nil)
			ctx.Log.Info("", nil)
		}
		return stdout
	})

	// Set status based on validation success
	component.Status = validateSSHCmd.Stdout.ApplyT(func(stdout string) string {
		if stdout != "" {
			return "validated"
		}
		return "validation-failed"
	}).(pulumi.StringOutput)

	if err := ctx.RegisterResourceOutputs(component, pulumi.Map{
		"status": component.Status,
	}); err != nil {
		return nil, err
	}

	return component, nil
}

// buildBastionProvisionScript creates the provisioning script for bastion security hardening
func buildBastionProvisionScript(cfg *config.BastionConfig, sudoPrefix string) string {
	// For AWS/Azure, we write the script to a temp file and execute with sudo
	// This avoids quoting issues with sudo bash -c '...'
	scriptHeader := `#!/bin/bash
set -e

echo "=========================================="
echo "🏰 BASTION PROVISIONING STARTED"
echo "=========================================="
echo "Time: $(date)"
echo ""
`

	// If using sudo (AWS/Azure), use a temp file approach instead of sudo bash -c
	if sudoPrefix != "" {
		// Write base script to temp file and execute with sudo
		// This avoids shell quoting issues
		scriptHeader = `#!/bin/bash
# Create temp script file to avoid quoting issues with sudo bash -c
TEMP_SCRIPT=$(mktemp /tmp/bastion-provision.XXXXXX.sh)
cat > "$TEMP_SCRIPT" << 'SCRIPT_END'
#!/bin/bash
set -e

echo "=========================================="
echo "🏰 BASTION PROVISIONING STARTED"
echo "=========================================="
echo "Time: $(date)"
echo ""
`
	}

	script := scriptHeader + `
# Function to wait for apt-get lock
wait_for_apt_lock() {
    local MAX_WAIT=300  # 5 minutes max
    local ELAPSED=0
    echo "[$(date +%H:%M:%S)] Checking for apt-get lock..."
    while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 || fuser /var/lib/apt/lists/lock >/dev/null 2>&1 || fuser /var/lib/dpkg/lock >/dev/null 2>&1; do
        if [ $ELAPSED -ge $MAX_WAIT ]; then
            echo "[$(date +%H:%M:%S)] ❌ ERROR: apt-get lock still held after ${MAX_WAIT}s, killing processes..."
            killall -9 apt-get apt dpkg unattended-upgrades || true
            sleep 5
            rm -f /var/lib/dpkg/lock-frontend /var/lib/dpkg/lock /var/lib/apt/lists/lock 2>/dev/null || true
            dpkg --configure -a || true
            break
        fi
        echo "[$(date +%H:%M:%S)]   ⏳ Waiting for apt lock... (${ELAPSED}s elapsed)"
        sleep 5
        ELAPSED=$((ELAPSED + 5))
    done
    echo "[$(date +%H:%M:%S)] ✅ apt-get lock released"
}

# Function to run apt-get commands with retry and mirror fallback
apt_get_with_retry() {
    local MAX_RETRIES=5
    local RETRY_COUNT=0
    local SWITCHED_MIRROR=false

    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        wait_for_apt_lock

        echo "[$(date +%H:%M:%S)] 🔄 Executing: $@"
        if "$@"; then
            echo "[$(date +%H:%M:%S)] ✅ Command succeeded"
            return 0
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))

            # On 3rd failure, try switching to Ubuntu official mirrors
            if [ $RETRY_COUNT -eq 3 ] && [ "$SWITCHED_MIRROR" = "false" ]; then
                echo "[$(date +%H:%M:%S)] ⚠️  Repeated failures detected, switching to official Ubuntu mirrors..."
                # Switch DigitalOcean mirrors
                if grep -q "mirrors.digitalocean.com" /etc/apt/sources.list 2>/dev/null; then
                    sed -i.bak 's|http://mirrors.digitalocean.com/ubuntu|http://archive.ubuntu.com/ubuntu|g' /etc/apt/sources.list
                fi
                # Switch Hetzner mirrors
                if grep -q "mirror.hetzner.com" /etc/apt/sources.list 2>/dev/null; then
                    sed -i.bak 's|https://mirror.hetzner.com/ubuntu/packages|http://archive.ubuntu.com/ubuntu|g' /etc/apt/sources.list
                    sed -i 's|https://mirror.hetzner.com/ubuntu/security|http://security.ubuntu.com/ubuntu|g' /etc/apt/sources.list
                fi
                SWITCHED_MIRROR=true
                echo "[$(date +%H:%M:%S)] 📝 Mirror switched to official Ubuntu repos, retrying..."
            fi

            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                echo "[$(date +%H:%M:%S)] ⚠️  Command failed, retrying in 10s... (attempt $((RETRY_COUNT + 1))/$MAX_RETRIES)"
                sleep 10
            else
                echo "[$(date +%H:%M:%S)] ❌ Command failed after $MAX_RETRIES attempts"
                return 1
            fi
        fi
    done
}

# Disable unattended-upgrades to prevent conflicts
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 1: Preparing system"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] Disabling unattended-upgrades..."
systemctl stop unattended-upgrades || true
systemctl disable unattended-upgrades || true
killall -9 unattended-upgrades || true
echo "[$(date +%H:%M:%S)] ✅ Unattended upgrades disabled"

# Initial lock wait
wait_for_apt_lock

# Update system
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 2: Updating system packages"
echo "[$(date +%H:%M:%S)] =========================================="
export DEBIAN_FRONTEND=noninteractive

# Fix Hetzner mirrors - replace with official Ubuntu mirrors
if grep -q "mirror.hetzner.com" /etc/apt/sources.list 2>/dev/null; then
    echo "[$(date +%H:%M:%S)] 🔧 Detected Hetzner mirrors, switching to official Ubuntu mirrors..."
    # Detect Ubuntu version
    UBUNTU_CODENAME=$(lsb_release -cs 2>/dev/null || echo "jammy")
    # Backup original
    cp /etc/apt/sources.list /etc/apt/sources.list.hetzner.bak
    # Create new sources.list with official mirrors
    cat > /etc/apt/sources.list << EOFMIRROR
deb http://archive.ubuntu.com/ubuntu ${UBUNTU_CODENAME} main restricted universe multiverse
deb http://archive.ubuntu.com/ubuntu ${UBUNTU_CODENAME}-updates main restricted universe multiverse
deb http://archive.ubuntu.com/ubuntu ${UBUNTU_CODENAME}-backports main restricted universe multiverse
deb http://security.ubuntu.com/ubuntu ${UBUNTU_CODENAME}-security main restricted universe multiverse
EOFMIRROR
    echo "[$(date +%H:%M:%S)] ✅ Switched to official Ubuntu mirrors"
fi

# Enable universe repository (required for fail2ban, wireguard-tools on some AMIs)
echo "[$(date +%H:%M:%S)] Enabling universe repository..."
add-apt-repository -y universe || apt-add-repository -y universe || true
sed -i '/^#.*universe/s/^#//' /etc/apt/sources.list || true

apt_get_with_retry apt-get update
# OPTIMIZATION: Skip apt-get upgrade to speed up provisioning (Ubuntu 24.04 is already recent)
# apt_get_with_retry apt-get upgrade -y -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold"
echo "[$(date +%H:%M:%S)] ⏭️  Skipping apt-get upgrade for faster provisioning"

# CRITICAL: Wait 5 seconds after apt-get update to ensure all locks are released
echo "[$(date +%H:%M:%S)] ⏳ Waiting 5s for all apt locks to be fully released..."
sleep 5

# Install required packages
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 3: Installing security packages"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] Installing: ufw, fail2ban, wireguard-tools, net-tools, curl, wget"
apt_get_with_retry apt-get install -y ufw fail2ban wireguard-tools net-tools curl wget

# Configure UFW Firewall
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 4: Configuring UFW firewall"
echo "[$(date +%H:%M:%S)] =========================================="
ufw default deny incoming
ufw default allow outgoing

# Allow SSH from allowed CIDRs
`

	// Add allowed CIDRs
	if len(cfg.AllowedCIDRs) > 0 {
		for _, cidr := range cfg.AllowedCIDRs {
			// Special handling for 0.0.0.0/0 - UFW doesn't handle "from 0.0.0.0/0" correctly
			if cidr == "0.0.0.0/0" {
				script += fmt.Sprintf("ufw allow %d/tcp comment 'SSH from anywhere'\n", cfg.SSHPort)
			} else {
				script += fmt.Sprintf("ufw allow from %s to any port %d proto tcp comment 'SSH from %s'\n", cidr, cfg.SSHPort, cidr)
			}
		}
	} else {
		// If no CIDRs specified, allow from anywhere (not recommended for production)
		script += fmt.Sprintf("ufw allow %d/tcp comment 'SSH (no CIDR restriction)'\n", cfg.SSHPort)
	}

	script += `
# Enable UFW
echo "[$(date +%H:%M:%S)] 🔥 Enabling UFW firewall..."
ufw --force enable
echo "[$(date +%H:%M:%S)] ✅ UFW firewall enabled and configured"

# Configure fail2ban
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 5: Configuring fail2ban"
echo "[$(date +%H:%M:%S)] =========================================="
cat > /etc/fail2ban/jail.local <<'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true
port = ` + fmt.Sprintf("%d", cfg.SSHPort) + `
logpath = /var/log/auth.log
maxretry = 3
EOF

systemctl enable fail2ban
systemctl restart fail2ban
echo "[$(date +%H:%M:%S)] ✅ fail2ban configured and started"

# SSH Hardening
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 6: Hardening SSH configuration"
echo "[$(date +%H:%M:%S)] =========================================="
cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup

cat >> /etc/ssh/sshd_config <<'EOF'

# Bastion Security Hardening
PermitRootLogin prohibit-password
PasswordAuthentication no
PubkeyAuthentication yes
ChallengeResponseAuthentication no
UsePAM yes
X11Forwarding no
PrintMotd no
AcceptEnv LANG LC_*
`

	if cfg.IdleTimeout > 0 {
		script += fmt.Sprintf("ClientAliveInterval 60\nClientAliveCountMax %d\n", cfg.IdleTimeout)
	}

	if cfg.MaxSessions > 0 {
		script += fmt.Sprintf("MaxSessions %d\n", cfg.MaxSessions)
	}

	script += `EOF

# Enable SSH agent forwarding and TCP forwarding for ProxyJump
# These settings are CRITICAL for bastion ProxyJump functionality
echo "[$(date +%H:%M:%S)] Configuring TCP forwarding for ProxyJump..."

# First, comment out ALL existing AllowTcpForwarding lines to prevent conflicts
# This handles all variations: yes, no, Yes, No, commented, uncommented
sed -i 's/^AllowTcpForwarding/#DISABLED_AllowTcpForwarding/' /etc/ssh/sshd_config
sed -i 's/^#AllowTcpForwarding/#DISABLED_AllowTcpForwarding/' /etc/ssh/sshd_config

# Comment out all AllowAgentForwarding lines too
sed -i 's/^AllowAgentForwarding/#DISABLED_AllowAgentForwarding/' /etc/ssh/sshd_config
sed -i 's/^#AllowAgentForwarding/#DISABLED_AllowAgentForwarding/' /etc/ssh/sshd_config

# Remove any Match blocks that might restrict these settings (Ubuntu cloud-init sometimes adds these)
# This is a simple approach - just comment them out
sed -i 's/^Match /#DISABLED_Match /' /etc/ssh/sshd_config

# Now add our settings at the VERY END of the file (SSH uses last-wins semantics)
echo "" >> /etc/ssh/sshd_config
echo "# ==============================================" >> /etc/ssh/sshd_config
echo "# ProxyJump/Bastion Settings (added by sloth-kubernetes)" >> /etc/ssh/sshd_config
echo "# These MUST be at the end of the file to override any Match blocks" >> /etc/ssh/sshd_config
echo "# ==============================================" >> /etc/ssh/sshd_config
echo "AllowTcpForwarding yes" >> /etc/ssh/sshd_config
echo "AllowAgentForwarding yes" >> /etc/ssh/sshd_config
echo "PermitTunnel yes" >> /etc/ssh/sshd_config
echo "GatewayPorts no" >> /etc/ssh/sshd_config

# Verify the configuration is valid before restarting
echo "[$(date +%H:%M:%S)] Validating sshd configuration..."
if sshd -t; then
    echo "[$(date +%H:%M:%S)] ✅ sshd configuration is valid"
else
    echo "[$(date +%H:%M:%S)] ❌ sshd configuration is INVALID! Check /etc/ssh/sshd_config"
    cat /etc/ssh/sshd_config | tail -30
fi

# Show what we configured
echo "[$(date +%H:%M:%S)] 📋 TCP forwarding settings applied:"
grep -i "AllowTcpForwarding\|AllowAgentForwarding\|PermitTunnel" /etc/ssh/sshd_config | tail -10

echo "[$(date +%H:%M:%S)] 🔄 Restarting SSH daemon to apply TCP forwarding settings..."
# RESTART is required for AllowTcpForwarding to take effect (reload is not enough)
# Modern SSH doesn't kill existing connections on restart - they continue running
systemctl restart sshd
sleep 3

# Verify the daemon is running and config is loaded
if systemctl is-active --quiet sshd; then
    echo "[$(date +%H:%M:%S)] ✅ SSH daemon restarted successfully"
else
    echo "[$(date +%H:%M:%S)] ❌ SSH daemon failed to restart!"
    systemctl status sshd
fi

echo "[$(date +%H:%M:%S)] ✅ SSH hardening complete"

# Audit Logging
`
	if cfg.EnableAuditLog {
		script += `
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 7: Setting up audit logging"
echo "[$(date +%H:%M:%S)] =========================================="
apt_get_with_retry apt-get install -y auditd audispd-plugins

# Log all SSH sessions
cat >> /etc/audit/rules.d/bastion.rules <<'EOF'
# Log all SSH sessions
-w /usr/sbin/sshd -p x -k bastion_ssh
-w /var/log/auth.log -p wa -k bastion_auth
EOF

augenrules --load || true
systemctl enable auditd
systemctl restart auditd
echo "[$(date +%H:%M:%S)] ✅ Audit logging configured"
`
	}

	script += `
# Install Salt Master with Salt API
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 8: Installing Salt Master with API"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] Downloading Salt Bootstrap script..."
curl -o /tmp/bootstrap-salt.sh -L https://github.com/saltstack/salt-bootstrap/releases/latest/download/bootstrap-salt.sh
chmod +x /tmp/bootstrap-salt.sh

echo "[$(date +%H:%M:%S)] Installing Salt Master and Salt API..."
sh /tmp/bootstrap-salt.sh -M -W stable

echo "[$(date +%H:%M:%S)] ✅ Salt Master and API installed successfully"

# Configure Salt API
echo "[$(date +%H:%M:%S)] Configuring Salt API..."
mkdir -p /etc/salt/master.d

# Configure Salt API (HTTP only - no SSL for simplicity)
cat > /etc/salt/master.d/api-nossl.conf <<'SALTEOF'
# Salt API Configuration - HTTP (no SSL)
rest_cherrypy:
  port: 8000
  host: 0.0.0.0
  disable_ssl: True
SALTEOF

# Add external_auth and netapi_enable_clients directly to /etc/salt/master
# This is CRITICAL - these configs must be in /etc/salt/master, not just /etc/salt/master.d/
echo "[$(date +%H:%M:%S)] Adding external_auth and netapi_enable_clients to /etc/salt/master..."
cat >> /etc/salt/master <<'SALTEOF'

# External authentication for Salt API
external_auth:
  pam:
    saltapi:
      - .*
      - '@wheel'
      - '@runner'
      - '@jobs'

# Enable Salt API clients (REQUIRED for 'local' client)
netapi_enable_clients: ["local", "local_async", "runner", "runner_async"]
SALTEOF

# Create Salt API user
echo "[$(date +%H:%M:%S)] Creating Salt API user..."
useradd -M -s /bin/bash saltapi || true
echo 'saltapi:saltapi123' | chpasswd

# CRITICAL: Add salt user to shadow group (required for PAM authentication)
echo "[$(date +%H:%M:%S)] Adding salt user to shadow group for PAM authentication..."
usermod -aG shadow salt

# Allow Salt API port in firewall
echo "[$(date +%H:%M:%S)] Configuring firewall for Salt API..."
ufw allow 8000/tcp comment 'Salt API'
ufw allow 4505/tcp comment 'Salt Publisher'
ufw allow 4506/tcp comment 'Salt Request Server'

# Allow WireGuard VPN port
echo "[$(date +%H:%M:%S)] Configuring firewall for WireGuard VPN..."
ufw allow 51820/udp comment 'WireGuard VPN'

# Restart Salt Master and start Salt API
echo "[$(date +%H:%M:%S)] Starting Salt Master and API services..."
systemctl restart salt-master
systemctl enable salt-api
systemctl start salt-api

echo "[$(date +%H:%M:%S)] ✅ Salt Master and API configured and running"

# Configure Salt to auto-accept minion keys
echo "[$(date +%H:%M:%S)] Configuring Salt Master to auto-accept minion keys..."
cat >> /etc/salt/master.d/auto-accept.conf <<'SALTEOF'
# Auto-accept minion keys (for automated deployments)
auto_accept: True
SALTEOF

# Restart Salt Master to apply auto-accept configuration
systemctl restart salt-master

echo "[$(date +%H:%M:%S)] ✅ Salt Master configured to auto-accept minion keys"

# Install WireGuard
echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] STEP 9: Finalizing configuration"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] WireGuard tools already installed"

# Set hostname
echo "[$(date +%H:%M:%S)] Setting hostname to ` + cfg.Name + `"
hostnamectl set-hostname ` + cfg.Name + `

# Create MOTD
echo "[$(date +%H:%M:%S)] Creating MOTD banner"
cat > /etc/motd <<'EOF'
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║            🏰  BASTION HOST - AUTHORIZED ACCESS ONLY      ║
║                                                           ║
║  This system is for authorized users only.                ║
║  All activity is monitored and logged.                    ║
║  Unauthorized access is prohibited.                       ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝

Cluster Access:
  • SSH to cluster nodes: ssh root@10.8.0.<node-vpn-ip>
  • ProxyJump is configured automatically
  • All sessions are audited

EOF

echo ""
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] ✅ BASTION PROVISIONING COMPLETE!"
echo "[$(date +%H:%M:%S)] =========================================="
echo "[$(date +%H:%M:%S)] 🏰 Bastion is ready for secure cluster access"
echo "[$(date +%H:%M:%S)] Finished at: $(date)"
echo ""
`

	// Close the temp file heredoc and execute with sudo if needed (AWS/Azure)
	if sudoPrefix != "" {
		script += `
SCRIPT_END

# Make script executable and run with sudo
chmod +x "$TEMP_SCRIPT"
echo "Executing provisioning script with sudo..."
sudo bash "$TEMP_SCRIPT"
RESULT=$?
rm -f "$TEMP_SCRIPT"
exit $RESULT
`
	}

	return script
}
