package providers

import (
	"encoding/base64"
	"fmt"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DigitalOceanProvider implements the Provider interface for DigitalOcean
type DigitalOceanProvider struct {
	config   *config.DigitalOceanProvider
	vpc      *digitalocean.Vpc
	firewall *digitalocean.Firewall
	sshKeys  pulumi.StringArray
	nodes    []*NodeOutput
	ctx      *pulumi.Context
}

// NewDigitalOceanProvider creates a new DigitalOcean provider
func NewDigitalOceanProvider() *DigitalOceanProvider {
	return &DigitalOceanProvider{
		nodes: make([]*NodeOutput, 0),
	}
}

// GetName returns the provider name
func (p *DigitalOceanProvider) GetName() string {
	return "digitalocean"
}

// Initialize initializes the DigitalOcean provider
func (p *DigitalOceanProvider) Initialize(ctx *pulumi.Context, config *config.ClusterConfig) error {
	p.ctx = ctx

	if config.Providers.DigitalOcean == nil || !config.Providers.DigitalOcean.Enabled {
		return fmt.Errorf("DigitalOcean provider is not enabled")
	}

	p.config = config.Providers.DigitalOcean

	// Generate or setup SSH keys
	if err := p.setupSSHKeys(ctx); err != nil {
		return fmt.Errorf("failed to setup SSH keys: %w", err)
	}

	ctx.Log.Info("DigitalOcean provider initialized", nil)
	return nil
}

// setupSSHKeys generates or imports SSH keys for DigitalOcean
func (p *DigitalOceanProvider) setupSSHKeys(ctx *pulumi.Context) error {
	// Check if we have a public key from the orchestrator
	if sshKey, ok := p.config.SSHPublicKey.(string); ok && sshKey != "" {
		// Create SSH key in DigitalOcean
		doSSHKey, err := digitalocean.NewSshKey(ctx, fmt.Sprintf("%s-do-key", ctx.Stack()), &digitalocean.SshKeyArgs{
			Name:      pulumi.String(fmt.Sprintf("%s-kubernetes", ctx.Stack())),
			PublicKey: pulumi.String(sshKey),
		})
		if err != nil {
			return fmt.Errorf("failed to create SSH key in DigitalOcean: %w", err)
		}

		// Store the SSH key fingerprint
		p.sshKeys = pulumi.StringArray{doSSHKey.Fingerprint}

		// Export SSH key info
		secrets.Export(ctx,"do_ssh_key_id", doSSHKey.ID())
		secrets.Export(ctx,"do_ssh_key_fingerprint", doSSHKey.Fingerprint)
		secrets.Export(ctx,"do_ssh_key_name", doSSHKey.Name)
	} else if len(p.config.SSHKeys) > 0 {
		// Use existing SSH keys
		keys := make(pulumi.StringArray, len(p.config.SSHKeys))
		for i, key := range p.config.SSHKeys {
			keys[i] = pulumi.String(key)
		}
		p.sshKeys = keys
	} else {
		return fmt.Errorf("no SSH keys configured")
	}

	return nil
}

// CreateNode creates a DigitalOcean droplet
func (p *DigitalOceanProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	// Generate user data script
	userData := p.generateUserData(node)

	// Encode user data to base64
	userDataEncoded := pulumi.String(userData).ToStringOutput().ApplyT(func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}).(pulumi.StringOutput)

	// Prepare tags
	tags := pulumi.StringArray{
		pulumi.String("kubernetes"),
		pulumi.String(p.ctx.Stack()),
	}

	// Add role tags
	for _, role := range node.Roles {
		tags = append(tags, pulumi.String(fmt.Sprintf("role:%s", role)))
	}

	// Add custom tags from config
	for _, tag := range p.config.Tags {
		tags = append(tags, pulumi.String(tag))
	}

	// Add node labels as tags
	for k, v := range node.Labels {
		tags = append(tags, pulumi.String(fmt.Sprintf("%s:%s", k, v)))
	}

	// Create droplet args
	dropletArgs := &digitalocean.DropletArgs{
		Name:       pulumi.String(node.Name),
		Region:     pulumi.String(node.Region),
		Size:       pulumi.String(node.Size),
		Image:      pulumi.String(node.Image),
		UserData:   userDataEncoded,
		SshKeys:    p.sshKeys,
		Monitoring: pulumi.Bool(p.config.Monitoring),
		Ipv6:       pulumi.Bool(p.config.IPv6),
		Tags:       tags,
	}

	// Add to VPC if configured
	if p.vpc != nil {
		dropletArgs.VpcUuid = p.vpc.ID()
	}

	// Create the droplet
	droplet, err := digitalocean.NewDroplet(ctx, node.Name, dropletArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create droplet %s: %w", node.Name, err)
	}

	// Create node output
	output := &NodeOutput{
		ID:          droplet.ID(),
		Name:        node.Name,
		PublicIP:    droplet.Ipv4Address,
		PrivateIP:   droplet.Ipv4AddressPrivate,
		Provider:    "digitalocean",
		Region:      node.Region,
		Size:        node.Size,
		Status:      droplet.Status,
		Labels:      node.Labels,
		WireGuardIP: node.WireGuardIP,
		SSHUser:     "root",
		SSHKeyPath:  "~/.ssh/id_rsa",
	}

	// Export node information
	secrets.Export(ctx,fmt.Sprintf("%s_public_ip", node.Name), droplet.Ipv4Address)
	secrets.Export(ctx,fmt.Sprintf("%s_private_ip", node.Name), droplet.Ipv4AddressPrivate)
	secrets.Export(ctx,fmt.Sprintf("%s_id", node.Name), droplet.ID())
	secrets.Export(ctx,fmt.Sprintf("%s_status", node.Name), droplet.Status)

	p.nodes = append(p.nodes, output)
	return output, nil
}

// CreateNodePool creates multiple nodes
func (p *DigitalOceanProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
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

		// Set WireGuard IP if using WireGuard
		if i < 3 { // First 3 nodes get specific IPs
			if i == 0 {
				nodeConfig.WireGuardIP = "10.8.0.11" // Master DO
			} else {
				nodeConfig.WireGuardIP = fmt.Sprintf("10.8.0.%d", 14+i) // Workers DO
			}
		}

		output, err := p.CreateNode(ctx, nodeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", nodeName, err)
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// CreateNetwork creates VPC infrastructure
func (p *DigitalOceanProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	if p.config.VPC == nil {
		// Create default VPC
		p.config.VPC = &config.VPCConfig{
			Name:    fmt.Sprintf("%s-vpc", ctx.Stack()),
			CIDR:    network.CIDR,
			Region:  p.config.Region,
			Private: true,
		}
	}

	vpc, err := digitalocean.NewVpc(ctx, p.config.VPC.Name, &digitalocean.VpcArgs{
		Name:    pulumi.String(p.config.VPC.Name),
		Region:  pulumi.String(p.config.VPC.Region),
		IpRange: pulumi.String(p.config.VPC.CIDR),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}

	p.vpc = vpc

	output := &NetworkOutput{
		ID:     vpc.ID(),
		Name:   p.config.VPC.Name,
		CIDR:   p.config.VPC.CIDR,
		Region: p.config.VPC.Region,
	}

	secrets.Export(ctx,"do_vpc_id", vpc.ID())
	secrets.Export(ctx,"do_vpc_cidr", pulumi.String(p.config.VPC.CIDR))

	return output, nil
}

// CreateFirewall creates firewall rules
func (p *DigitalOceanProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	// Convert IDs to int array
	dropletIds := make([]pulumi.IntOutput, len(nodeIds))
	for i, id := range nodeIds {
		dropletIds[i] = id.ApplyT(func(idAny interface{}) int {
			id := idAny.(pulumi.ID)
			var idInt int
			fmt.Sscanf(string(id), "%d", &idInt)
			return idInt
		}).(pulumi.IntOutput)
	}

	// Build inbound rules
	inboundRules := digitalocean.FirewallInboundRuleArray{}

	// Default: Allow all traffic within VPC
	if p.vpc != nil {
		inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
			Protocol:        pulumi.String("tcp"),
			PortRange:       pulumi.String("1-65535"),
			SourceAddresses: pulumi.StringArray{pulumi.String(p.config.VPC.CIDR)},
		})
		inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
			Protocol:        pulumi.String("udp"),
			PortRange:       pulumi.String("1-65535"),
			SourceAddresses: pulumi.StringArray{pulumi.String(p.config.VPC.CIDR)},
		})
	}

	// Add WireGuard rule if needed
	inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
		Protocol:        pulumi.String("udp"),
		PortRange:       pulumi.String("51820"),
		SourceAddresses: pulumi.StringArray{pulumi.String("10.8.0.0/24")},
	})

	// Convert custom firewall rules
	for _, rule := range firewall.InboundRules {
		sources := make(pulumi.StringArray, len(rule.Source))
		for i, src := range rule.Source {
			sources[i] = pulumi.String(src)
		}

		inboundRules = append(inboundRules, &digitalocean.FirewallInboundRuleArgs{
			Protocol:        pulumi.String(rule.Protocol),
			PortRange:       pulumi.String(rule.Port),
			SourceAddresses: sources,
		})
	}

	// Build outbound rules (allow all by default)
	outboundRules := digitalocean.FirewallOutboundRuleArray{
		&digitalocean.FirewallOutboundRuleArgs{
			Protocol:             pulumi.String("tcp"),
			PortRange:            pulumi.String("1-65535"),
			DestinationAddresses: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		&digitalocean.FirewallOutboundRuleArgs{
			Protocol:             pulumi.String("udp"),
			PortRange:            pulumi.String("1-65535"),
			DestinationAddresses: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		&digitalocean.FirewallOutboundRuleArgs{
			Protocol:             pulumi.String("icmp"),
			DestinationAddresses: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
	}

	// Convert dropletIds to pulumi.IntArray
	dropletIntArray := make(pulumi.IntArray, len(dropletIds))
	for i, id := range dropletIds {
		dropletIntArray[i] = id
	}

	// Create firewall
	fw, err := digitalocean.NewFirewall(ctx, firewall.Name, &digitalocean.FirewallArgs{
		Name:          pulumi.String(firewall.Name),
		DropletIds:    dropletIntArray,
		InboundRules:  inboundRules,
		OutboundRules: outboundRules,
	})
	if err != nil {
		return fmt.Errorf("failed to create firewall: %w", err)
	}

	p.firewall = fw
	secrets.Export(ctx,"do_firewall_id", fw.ID())

	return nil
}

// CreateLoadBalancer creates a load balancer
func (p *DigitalOceanProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	// Get droplet IDs for the load balancer
	dropletIds := make(pulumi.IntArray, 0)
	for _, node := range p.nodes {
		dropletIds = append(dropletIds, node.ID.ApplyT(func(id pulumi.ID) int {
			var idInt int
			fmt.Sscanf(string(id), "%d", &idInt)
			return idInt
		}).(pulumi.IntOutput))
	}

	// Create forwarding rules
	forwardingRules := make(digitalocean.LoadBalancerForwardingRuleArray, len(lb.Ports))
	for i, port := range lb.Ports {
		forwardingRules[i] = &digitalocean.LoadBalancerForwardingRuleArgs{
			EntryPort:      pulumi.Int(port.Port),
			TargetPort:     pulumi.Int(port.TargetPort),
			EntryProtocol:  pulumi.String(port.Protocol),
			TargetProtocol: pulumi.String(port.Protocol),
		}
	}

	// Create load balancer args
	lbArgs := &digitalocean.LoadBalancerArgs{
		Name:                pulumi.String(lb.Name),
		Region:              pulumi.String(p.config.Region),
		Size:                pulumi.String("lb-small"),
		ForwardingRules:     forwardingRules,
		DropletIds:          dropletIds,
		RedirectHttpToHttps: pulumi.Bool(true),
	}

	// Add VPC if available
	if p.vpc != nil {
		lbArgs.VpcUuid = p.vpc.ID()
	}

	// Create load balancer
	loadBalancer, err := digitalocean.NewLoadBalancer(ctx, lb.Name, lbArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	output := &LoadBalancerOutput{
		ID:     loadBalancer.ID(),
		IP:     loadBalancer.Ip,
		Status: loadBalancer.Status,
	}

	secrets.Export(ctx,fmt.Sprintf("%s_ip", lb.Name), loadBalancer.Ip)
	secrets.Export(ctx,fmt.Sprintf("%s_status", lb.Name), loadBalancer.Status)

	return output, nil
}

// GetRegions returns available regions
func (p *DigitalOceanProvider) GetRegions() []string {
	return []string{
		"nyc1", "nyc3", "sfo3", "ams3", "sgp1",
		"lon1", "fra1", "tor1", "blr1", "syd1",
	}
}

// GetSizes returns available instance sizes
func (p *DigitalOceanProvider) GetSizes() []string {
	return []string{
		"s-1vcpu-1gb", "s-1vcpu-2gb", "s-2vcpu-2gb", "s-2vcpu-4gb",
		"s-4vcpu-8gb", "s-8vcpu-16gb", "c-2", "c-4", "c-8", "c-16",
	}
}

// Cleanup performs cleanup operations
func (p *DigitalOceanProvider) Cleanup(ctx *pulumi.Context) error {
	// Cleanup is handled by Pulumi's resource management
	return nil
}

// generateUserData generates the user data script for the droplet
func (p *DigitalOceanProvider) generateUserData(node *config.NodeConfig) string {
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
usermod -aG docker root

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
echo "NODE_PROVIDER=digitalocean" >> /etc/environment
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

	baseScript += "\necho 'Node initialization complete'\n"

	return fmt.Sprintf(baseScript, node.Region, node.Size)
}
