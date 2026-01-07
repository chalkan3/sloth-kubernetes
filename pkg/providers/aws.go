package providers

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AWSProvider implements the Provider interface for Amazon Web Services
type AWSProvider struct {
	config        *config.AWSProvider
	vpc           *ec2.Vpc
	subnet        *ec2.Subnet
	subnets       []*ec2.Subnet
	securityGroup *ec2.SecurityGroup
	keyPair       *ec2.KeyPair
	nodes         []*NodeOutput
	ctx           *pulumi.Context
	clusterConfig *config.ClusterConfig
}

// NewAWSProvider creates a new AWS provider
func NewAWSProvider() *AWSProvider {
	return &AWSProvider{
		nodes:   make([]*NodeOutput, 0),
		subnets: make([]*ec2.Subnet, 0),
	}
}

// GetName returns the provider name
func (p *AWSProvider) GetName() string {
	return "aws"
}

// GetRegions returns available AWS regions
func (p *AWSProvider) GetRegions() []string {
	return []string{
		"us-east-1",      // N. Virginia
		"us-east-2",      // Ohio
		"us-west-1",      // N. California
		"us-west-2",      // Oregon
		"eu-west-1",      // Ireland
		"eu-west-2",      // London
		"eu-west-3",      // Paris
		"eu-central-1",   // Frankfurt
		"ap-southeast-1", // Singapore
		"ap-southeast-2", // Sydney
		"ap-northeast-1", // Tokyo
		"ap-northeast-2", // Seoul
		"ap-south-1",     // Mumbai
		"sa-east-1",      // Sao Paulo
		"ca-central-1",   // Canada
	}
}

// GetSizes returns available EC2 instance types
func (p *AWSProvider) GetSizes() []string {
	return []string{
		// T2 (Burstable)
		"t2.micro", "t2.small", "t2.medium", "t2.large", "t2.xlarge",
		// T3 (Burstable, newer)
		"t3.micro", "t3.small", "t3.medium", "t3.large", "t3.xlarge",
		// M5 (General Purpose)
		"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge",
		// C5 (Compute Optimized)
		"c5.large", "c5.xlarge", "c5.2xlarge", "c5.4xlarge",
		// R5 (Memory Optimized)
		"r5.large", "r5.xlarge", "r5.2xlarge",
	}
}

// Initialize initializes the AWS provider
func (p *AWSProvider) Initialize(ctx *pulumi.Context, cfg *config.ClusterConfig) error {
	p.ctx = ctx
	p.clusterConfig = cfg

	if cfg.Providers.AWS == nil || !cfg.Providers.AWS.Enabled {
		return fmt.Errorf("AWS provider is not enabled")
	}

	p.config = cfg.Providers.AWS

	// Validate required configuration
	if p.config.Region == "" {
		return fmt.Errorf("AWS region is required")
	}

	// Setup SSH key pair
	if err := p.setupKeyPair(ctx); err != nil {
		return fmt.Errorf("failed to setup key pair: %w", err)
	}

	ctx.Log.Info("AWS provider initialized", nil)
	return nil
}

// setupKeyPair creates or imports an EC2 key pair
func (p *AWSProvider) setupKeyPair(ctx *pulumi.Context) error {
	// Check if we have a public key path configured
	publicKeyPath := p.clusterConfig.Security.SSHConfig.PublicKeyPath
	if publicKeyPath != "" {
		// Read public key from file
		sshKey, err := readSSHPublicKey(publicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read SSH public key: %w", err)
		}
		// Create key pair in AWS
		keyPair, err := ec2.NewKeyPair(ctx, fmt.Sprintf("%s-aws-key", ctx.Stack()), &ec2.KeyPairArgs{
			KeyName:   pulumi.String(fmt.Sprintf("%s-kubernetes", ctx.Stack())),
			PublicKey: pulumi.String(sshKey),
			Tags: pulumi.StringMap{
				"Name":    pulumi.String(fmt.Sprintf("%s-kubernetes-key", ctx.Stack())),
				"Cluster": pulumi.String(ctx.Stack()),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create key pair: %w", err)
		}

		p.keyPair = keyPair

		// Export key pair info
		secrets.Export(ctx,"aws_key_pair_id", keyPair.ID())
		secrets.Export(ctx,"aws_key_pair_name", keyPair.KeyName)
		secrets.Export(ctx,"aws_key_pair_fingerprint", keyPair.Fingerprint)
	} else if p.config.KeyPair != "" {
		// Use existing key pair name - we'll reference it by name
		ctx.Log.Info(fmt.Sprintf("Using existing key pair: %s", p.config.KeyPair), nil)
	} else {
		return fmt.Errorf("no SSH key configured - provide security.ssh.publicKey or providers.aws.keyPair")
	}

	return nil
}

// CreateNetwork creates VPC, subnets, internet gateway, and route tables
func (p *AWSProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	vpcConfig := p.config.VPC
	if vpcConfig == nil {
		vpcConfig = &config.VPCConfig{
			Create: true,
			CIDR:   "10.0.0.0/16",
		}
	}

	// Create VPC
	vpc, err := ec2.NewVpc(ctx, fmt.Sprintf("%s-vpc", ctx.Stack()), &ec2.VpcArgs{
		CidrBlock:          pulumi.String(vpcConfig.CIDR),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-vpc", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}
	p.vpc = vpc

	// Create Internet Gateway
	igw, err := ec2.NewInternetGateway(ctx, fmt.Sprintf("%s-igw", ctx.Stack()), &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-igw", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create internet gateway: %w", err)
	}

	// Calculate subnet CIDRs from VPC CIDR
	// For a /16 VPC (e.g., 10.100.0.0/16), create /24 subnets (e.g., 10.100.1.0/24, 10.100.2.0/24)
	subnetCIDRs := calculateSubnetCIDRs(vpcConfig.CIDR, 2)
	if len(subnetCIDRs) < 2 {
		return nil, fmt.Errorf("failed to calculate subnet CIDRs from VPC CIDR %s", vpcConfig.CIDR)
	}

	// Create public subnet
	subnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-subnet-public", ctx.Stack()), &ec2.SubnetArgs{
		VpcId:                       vpc.ID(),
		CidrBlock:                   pulumi.String(subnetCIDRs[0]),
		MapPublicIpOnLaunch:         pulumi.Bool(true),
		AvailabilityZone:            pulumi.String(fmt.Sprintf("%sa", p.config.Region)),
		AssignIpv6AddressOnCreation: pulumi.Bool(false),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-subnet-public-1", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
			"Type":    pulumi.String("public"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}
	p.subnet = subnet
	p.subnets = append(p.subnets, subnet)

	// Create second subnet in different AZ for high availability
	subnet2, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-subnet-public-2", ctx.Stack()), &ec2.SubnetArgs{
		VpcId:               vpc.ID(),
		CidrBlock:           pulumi.String(subnetCIDRs[1]),
		MapPublicIpOnLaunch: pulumi.Bool(true),
		AvailabilityZone:    pulumi.String(fmt.Sprintf("%sb", p.config.Region)),
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-subnet-public-2", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
			"Type":    pulumi.String("public"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create second subnet: %w", err)
	}
	p.subnets = append(p.subnets, subnet2)

	// Create route table
	routeTable, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-rt", ctx.Stack()), &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: igw.ID(),
			},
		},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-rt-public", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create route table: %w", err)
	}

	// Associate route table with subnets
	_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-rta-1", ctx.Stack()), &ec2.RouteTableAssociationArgs{
		SubnetId:     subnet.ID(),
		RouteTableId: routeTable.ID(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to associate route table with subnet 1: %w", err)
	}

	_, err = ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-rta-2", ctx.Stack()), &ec2.RouteTableAssociationArgs{
		SubnetId:     subnet2.ID(),
		RouteTableId: routeTable.ID(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to associate route table with subnet 2: %w", err)
	}

	// Export network info
	secrets.Export(ctx,"aws_vpc_id", vpc.ID())
	secrets.Export(ctx,"aws_vpc_cidr", vpc.CidrBlock)
	secrets.Export(ctx,"aws_subnet_id", subnet.ID())
	secrets.Export(ctx,"aws_igw_id", igw.ID())

	return &NetworkOutput{
		ID:     vpc.ID(),
		Name:   fmt.Sprintf("%s-vpc", ctx.Stack()),
		CIDR:   vpcConfig.CIDR,
		Region: p.config.Region,
		Subnets: []SubnetOutput{
			{ID: subnet.ID(), CIDR: "10.0.1.0/24", Zone: fmt.Sprintf("%sa", p.config.Region)},
			{ID: subnet2.ID(), CIDR: "10.0.2.0/24", Zone: fmt.Sprintf("%sb", p.config.Region)},
		},
	}, nil
}

// CreateFirewall creates a security group with the specified rules
func (p *AWSProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	if p.vpc == nil {
		return fmt.Errorf("VPC must be created before creating firewall")
	}

	// Build ingress rules
	ingressRules := ec2.SecurityGroupIngressArray{
		// SSH
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("tcp"),
			FromPort:   pulumi.Int(22),
			ToPort:     pulumi.Int(22),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		// Kubernetes API
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("tcp"),
			FromPort:   pulumi.Int(6443),
			ToPort:     pulumi.Int(6443),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		// WireGuard
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("udp"),
			FromPort:   pulumi.Int(51820),
			ToPort:     pulumi.Int(51820),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		// All traffic within VPC
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("-1"),
			FromPort:   pulumi.Int(0),
			ToPort:     pulumi.Int(0),
			CidrBlocks: pulumi.StringArray{pulumi.String("10.0.0.0/16")},
		},
		// NodePort range
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("tcp"),
			FromPort:   pulumi.Int(30000),
			ToPort:     pulumi.Int(32767),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		// HTTP/HTTPS
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("tcp"),
			FromPort:   pulumi.Int(80),
			ToPort:     pulumi.Int(80),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		&ec2.SecurityGroupIngressArgs{
			Protocol:   pulumi.String("tcp"),
			FromPort:   pulumi.Int(443),
			ToPort:     pulumi.Int(443),
			CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
	}

	// Add custom firewall rules
	if firewall != nil {
		for _, rule := range firewall.InboundRules {
			port, _ := strconv.Atoi(rule.Port)
			// Convert source CIDRs
			cidrBlocks := pulumi.StringArray{}
			for _, src := range rule.Source {
				cidrBlocks = append(cidrBlocks, pulumi.String(src))
			}
			if len(cidrBlocks) == 0 {
				cidrBlocks = pulumi.StringArray{pulumi.String("0.0.0.0/0")}
			}
			ingressRules = append(ingressRules, &ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String(rule.Protocol),
				FromPort:   pulumi.Int(port),
				ToPort:     pulumi.Int(port),
				CidrBlocks: cidrBlocks,
			})
		}
	}

	// Create security group
	sg, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-sg", ctx.Stack()), &ec2.SecurityGroupArgs{
		Name:        pulumi.String(fmt.Sprintf("%s-kubernetes-sg", ctx.Stack())),
		Description: pulumi.String("Security group for Kubernetes cluster"),
		VpcId:       p.vpc.ID(),
		Ingress:     ingressRules,
		Egress: ec2.SecurityGroupEgressArray{
			// Allow all outbound
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-kubernetes-sg", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create security group: %w", err)
	}

	p.securityGroup = sg

	// Export security group info
	secrets.Export(ctx,"aws_security_group_id", sg.ID())
	secrets.Export(ctx,"aws_security_group_name", sg.Name)

	return nil
}

// CreateNode creates an EC2 instance
func (p *AWSProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	if p.securityGroup == nil {
		return nil, fmt.Errorf("security group must be created before creating nodes")
	}

	// Generate user data script
	userData := p.generateUserData(node)
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userData))

	// Get AMI for Ubuntu
	ami := p.getUbuntuAMI(node.Image)

	// Determine key name
	var keyName pulumi.StringPtrInput
	if p.keyPair != nil {
		keyName = p.keyPair.KeyName
	} else if p.config.KeyPair != "" {
		keyName = pulumi.StringPtr(p.config.KeyPair)
	}

	// Select subnet (round-robin between available subnets)
	subnetIndex := len(p.nodes) % len(p.subnets)
	subnet := p.subnets[subnetIndex]

	// Build tags
	tags := pulumi.StringMap{
		"Name":    pulumi.String(node.Name),
		"Cluster": pulumi.String(ctx.Stack()),
	}
	for _, role := range node.Roles {
		tags[fmt.Sprintf("Role-%s", role)] = pulumi.String("true")
	}
	for k, v := range node.Labels {
		tags[k] = pulumi.String(v)
	}

	var instance *ec2.Instance
	var err error

	// Check if this is a spot instance request
	if node.SpotInstance {
		// Create Spot Instance Request
		spotRequest, err := ec2.NewSpotInstanceRequest(ctx, node.Name, &ec2.SpotInstanceRequestArgs{
			Ami:                      pulumi.String(ami),
			InstanceType:             pulumi.String(node.Size),
			KeyName:                  keyName,
			SubnetId:                 subnet.ID(),
			VpcSecurityGroupIds:      pulumi.StringArray{p.securityGroup.ID()},
			UserData:                 pulumi.String(userDataEncoded),
			AssociatePublicIpAddress: pulumi.Bool(true),
			SpotType:                 pulumi.String("one-time"),
			WaitForFulfillment:       pulumi.Bool(true),
			Tags:                     tags,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create spot instance request %s: %w", node.Name, err)
		}

		// Create node output for spot instance
		output := &NodeOutput{
			ID:          spotRequest.ID(),
			Name:        node.Name,
			PublicIP:    spotRequest.PublicIp,
			PrivateIP:   spotRequest.PrivateIp,
			Provider:    "aws",
			Region:      p.config.Region,
			Size:        node.Size,
			Status:      spotRequest.SpotRequestState,
			Labels:      node.Labels,
			WireGuardIP: node.WireGuardIP,
			SSHUser:     "ubuntu",
			SSHKeyPath:  p.clusterConfig.Security.SSHConfig.KeyPath,
		}

		p.nodes = append(p.nodes, output)

		// Export node info
		secrets.Export(ctx,fmt.Sprintf("%s_public_ip", node.Name), spotRequest.PublicIp)
		secrets.Export(ctx,fmt.Sprintf("%s_private_ip", node.Name), spotRequest.PrivateIp)
		secrets.Export(ctx,fmt.Sprintf("%s_id", node.Name), spotRequest.SpotInstanceId)

		return output, nil
	}

	// Create regular EC2 instance
	instance, err = ec2.NewInstance(ctx, node.Name, &ec2.InstanceArgs{
		Ami:                      pulumi.String(ami),
		InstanceType:             pulumi.String(node.Size),
		KeyName:                  keyName,
		SubnetId:                 subnet.ID(),
		VpcSecurityGroupIds:      pulumi.StringArray{p.securityGroup.ID()},
		UserDataBase64:           pulumi.String(userDataEncoded),
		AssociatePublicIpAddress: pulumi.Bool(true),
		RootBlockDevice: &ec2.InstanceRootBlockDeviceArgs{
			VolumeSize:          pulumi.Int(50),
			VolumeType:          pulumi.String("gp3"),
			DeleteOnTermination: pulumi.Bool(true),
		},
		Tags: tags,
		MetadataOptions: &ec2.InstanceMetadataOptionsArgs{
			HttpTokens:   pulumi.String("optional"),
			HttpEndpoint: pulumi.String("enabled"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instance %s: %w", node.Name, err)
	}

	// Create node output
	output := &NodeOutput{
		ID:          instance.ID(),
		Name:        node.Name,
		PublicIP:    instance.PublicIp,
		PrivateIP:   instance.PrivateIp,
		Provider:    "aws",
		Region:      p.config.Region,
		Size:        node.Size,
		Status:      instance.InstanceState,
		Labels:      node.Labels,
		WireGuardIP: node.WireGuardIP,
		SSHUser:     "ubuntu",
		SSHKeyPath:  p.clusterConfig.Security.SSHConfig.KeyPath,
	}

	p.nodes = append(p.nodes, output)

	// Export node info
	secrets.Export(ctx,fmt.Sprintf("%s_public_ip", node.Name), instance.PublicIp)
	secrets.Export(ctx,fmt.Sprintf("%s_private_ip", node.Name), instance.PrivateIp)
	secrets.Export(ctx,fmt.Sprintf("%s_id", node.Name), instance.ID())
	secrets.Export(ctx,fmt.Sprintf("%s_status", node.Name), instance.InstanceState)

	return output, nil
}

// CreateNodePool creates a pool of EC2 instances
func (p *AWSProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
	outputs := make([]*NodeOutput, 0, pool.Count)

	// Get available zones
	zones := pool.Zones
	if len(zones) == 0 {
		zones = []string{
			fmt.Sprintf("%sa", p.config.Region),
			fmt.Sprintf("%sb", p.config.Region),
		}
	}

	for i := 0; i < pool.Count; i++ {
		// Distribute across zones
		zone := zones[i%len(zones)]

		// Generate node name
		nodeName := fmt.Sprintf("%s-%d", pool.Name, i+1)

		// Assign WireGuard IP based on role
		var wireGuardIP string
		isMaster := false
		for _, role := range pool.Roles {
			if role == "master" || role == "controlplane" {
				isMaster = true
				break
			}
		}

		if isMaster {
			wireGuardIP = fmt.Sprintf("10.8.0.%d", 20+i)
		} else {
			wireGuardIP = fmt.Sprintf("10.8.0.%d", 30+i)
		}

		// Create node config
		nodeConfig := &config.NodeConfig{
			Name:         nodeName,
			Provider:     "aws",
			Pool:         pool.Name,
			Roles:        pool.Roles,
			Size:         pool.Size,
			Image:        pool.Image,
			Region:       p.config.Region,
			Zone:         zone,
			Labels:       pool.Labels,
			Taints:       pool.Taints,
			WireGuardIP:  wireGuardIP,
			SpotInstance: pool.SpotInstance,
			UserData:     pool.UserData,
		}

		output, err := p.CreateNode(ctx, nodeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", nodeName, err)
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// CreateLoadBalancer creates a Network Load Balancer
func (p *AWSProvider) CreateLoadBalancer(ctx *pulumi.Context, lbConfig *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	if p.vpc == nil || len(p.subnets) == 0 {
		return nil, fmt.Errorf("VPC and subnets must be created before creating load balancer")
	}

	// Collect subnet IDs
	subnetIds := pulumi.StringArray{}
	for _, subnet := range p.subnets {
		subnetIds = append(subnetIds, subnet.ID())
	}

	// Create Network Load Balancer
	nlb, err := lb.NewLoadBalancer(ctx, fmt.Sprintf("%s-nlb", ctx.Stack()), &lb.LoadBalancerArgs{
		Name:             pulumi.String(fmt.Sprintf("%s-nlb", ctx.Stack())),
		LoadBalancerType: pulumi.String("network"),
		Subnets:          subnetIds,
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-nlb", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Create target group for Kubernetes API
	tg, err := lb.NewTargetGroup(ctx, fmt.Sprintf("%s-tg-k8s-api", ctx.Stack()), &lb.TargetGroupArgs{
		Name:       pulumi.String(fmt.Sprintf("%s-k8s-api", ctx.Stack())),
		Port:       pulumi.Int(6443),
		Protocol:   pulumi.String("TCP"),
		VpcId:      p.vpc.ID(),
		TargetType: pulumi.String("instance"),
		HealthCheck: &lb.TargetGroupHealthCheckArgs{
			Protocol:           pulumi.String("TCP"),
			Port:               pulumi.String("6443"),
			HealthyThreshold:   pulumi.Int(3),
			UnhealthyThreshold: pulumi.Int(3),
			Interval:           pulumi.Int(30),
		},
		Tags: pulumi.StringMap{
			"Name":    pulumi.String(fmt.Sprintf("%s-tg-k8s-api", ctx.Stack())),
			"Cluster": pulumi.String(ctx.Stack()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create target group: %w", err)
	}

	// Register master nodes with target group
	for i, node := range p.nodes {
		// Only register master nodes
		isMaster := false
		for k := range node.Labels {
			if k == "role" && node.Labels[k] == "master" {
				isMaster = true
				break
			}
		}
		// Check node name for master indication
		if !isMaster {
			for _, role := range []string{"master", "controlplane"} {
				if len(node.Name) > len(role) && node.Name[:len(role)] == role {
					isMaster = true
					break
				}
			}
		}

		if isMaster || i < 3 { // Register first 3 nodes as masters if role not clear
			_, err := lb.NewTargetGroupAttachment(ctx, fmt.Sprintf("%s-tga-%d", ctx.Stack(), i), &lb.TargetGroupAttachmentArgs{
				TargetGroupArn: tg.Arn,
				TargetId:       node.ID.ToStringOutput(),
				Port:           pulumi.Int(6443),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to attach node to target group: %w", err)
			}
		}
	}

	// Create listener
	_, err = lb.NewListener(ctx, fmt.Sprintf("%s-listener-k8s", ctx.Stack()), &lb.ListenerArgs{
		LoadBalancerArn: nlb.Arn,
		Port:            pulumi.Int(6443),
		Protocol:        pulumi.String("TCP"),
		DefaultActions: lb.ListenerDefaultActionArray{
			&lb.ListenerDefaultActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: tg.Arn,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Export load balancer info
	secrets.Export(ctx,"aws_nlb_dns_name", nlb.DnsName)
	secrets.Export(ctx,"aws_nlb_arn", nlb.Arn)
	secrets.Export(ctx,"aws_nlb_zone_id", nlb.ZoneId)

	return &LoadBalancerOutput{
		ID:       nlb.ID(),
		IP:       nlb.DnsName,
		Hostname: nlb.DnsName,
		Status:   pulumi.String("active").ToStringOutput(),
	}, nil
}

// Cleanup performs cleanup operations
func (p *AWSProvider) Cleanup(ctx *pulumi.Context) error {
	ctx.Log.Info("AWS cleanup completed", nil)
	return nil
}

// generateUserData creates the cloud-init user data script
func (p *AWSProvider) generateUserData(node *config.NodeConfig) string {
	rolesStr := ""
	for _, role := range node.Roles {
		rolesStr += fmt.Sprintf("export NODE_ROLE_%s=true\n", role)
	}

	labelsStr := ""
	for k, v := range node.Labels {
		labelsStr += fmt.Sprintf("export NODE_LABEL_%s=%s\n", k, v)
	}

	customUserData := ""
	if node.UserData != "" {
		customUserData = node.UserData
	} else if p.config.Custom != nil {
		if ud, ok := p.config.Custom["userData"].(string); ok {
			customUserData = ud
		}
	}

	return fmt.Sprintf(`#!/bin/bash
set -e

# Node Configuration
export NODE_NAME=%s
export NODE_PROVIDER=aws
export NODE_REGION=%s
export WIREGUARD_IP=%s
%s
%s

# Update system
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y

# Install essential packages
DEBIAN_FRONTEND=noninteractive apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    software-properties-common \
    jq \
    net-tools \
    wireguard \
    wireguard-tools

# Install Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io

# Install kubectl
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y kubectl

# Disable swap
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

# Enable IP forwarding
cat <<EOF | tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system

# Load kernel modules
modprobe br_netfilter
modprobe overlay
echo -e "br_netfilter\noverlay" > /etc/modules-load.d/k8s.conf

# Setup WireGuard
mkdir -p /etc/wireguard
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey
chmod 600 /etc/wireguard/privatekey

# Save node info
mkdir -p /opt/kubernetes
cat <<EOF > /opt/kubernetes/node-info.json
{
  "name": "$NODE_NAME",
  "provider": "aws",
  "region": "%s",
  "wireguard_ip": "$WIREGUARD_IP"
}
EOF

# Run custom user data
%s

echo "Node initialization complete"
`, node.Name, node.Region, node.WireGuardIP, rolesStr, labelsStr, node.Region, customUserData)
}

// getUbuntuAMI returns the Ubuntu AMI ID for the given region
func (p *AWSProvider) getUbuntuAMI(image string) string {
	// Ubuntu 22.04 LTS AMIs by region (updated periodically)
	// These are official Canonical AMIs for x86_64
	amiMap := map[string]string{
		"us-east-1":      "ami-0c7217cdde317cfec", // Ubuntu 22.04 LTS
		"us-east-2":      "ami-0e83be366243f524a",
		"us-west-1":      "ami-0ce2cb35386fc22e9",
		"us-west-2":      "ami-008fe2fc65df48dac",
		"eu-west-1":      "ami-0905a3c97561e0b69",
		"eu-west-2":      "ami-0e5f882be1900e43b",
		"eu-west-3":      "ami-01d21b7be69801c2f",
		"eu-central-1":   "ami-0faab6bdbac9486fb",
		"ap-southeast-1": "ami-078c1149d8ad719a7",
		"ap-southeast-2": "ami-04f5097681773b989",
		"ap-northeast-1": "ami-07c589821f2b353aa",
		"ap-northeast-2": "ami-0c9c942bd7bf113a2",
		"ap-south-1":     "ami-03f4878755434977f",
		"sa-east-1":      "ami-0fb4cf3a99aa89f72",
		"ca-central-1":   "ami-0a2e7efb4257c0907",
	}

	if ami, ok := amiMap[p.config.Region]; ok {
		return ami
	}

	// Default to us-east-1 AMI
	return amiMap["us-east-1"]
}

// readSSHPublicKey reads the SSH public key from a file
func readSSHPublicKey(path string) (string, error) {
	// Expand home directory if needed
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = home + path[1:]
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return strings.TrimSpace(string(content)), nil
}

// calculateSubnetCIDRs calculates subnet CIDRs from a VPC CIDR
// For example, given "10.100.0.0/16" and count=2, returns ["10.100.1.0/24", "10.100.2.0/24"]
func calculateSubnetCIDRs(vpcCIDR string, count int) []string {
	parts := strings.Split(vpcCIDR, "/")
	if len(parts) != 2 {
		return nil
	}

	ipParts := strings.Split(parts[0], ".")
	if len(ipParts) != 4 {
		return nil
	}

	// Parse the first two octets
	firstOctet := ipParts[0]
	secondOctet := ipParts[1]

	// Create /24 subnets within the VPC CIDR
	var subnets []string
	for i := 1; i <= count; i++ {
		subnet := fmt.Sprintf("%s.%s.%d.0/24", firstOctet, secondOctet, i)
		subnets = append(subnets, subnet)
	}

	return subnets
}
