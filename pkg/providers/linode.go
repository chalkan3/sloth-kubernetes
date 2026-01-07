package providers

import (
	"fmt"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/secrets"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// LinodeProvider implements the Provider interface for Linode/Akamai
type LinodeProvider struct {
	config   *config.LinodeProvider
	firewall *linode.Firewall
	nodes    []*NodeOutput
	ctx      *pulumi.Context
}

// NewLinodeProvider creates a new Linode provider
func NewLinodeProvider() *LinodeProvider {
	return &LinodeProvider{
		nodes: make([]*NodeOutput, 0),
	}
}

// GetName returns the provider name
func (p *LinodeProvider) GetName() string {
	return "linode"
}

// Initialize initializes the Linode provider
func (p *LinodeProvider) Initialize(ctx *pulumi.Context, config *config.ClusterConfig) error {
	p.ctx = ctx

	if config.Providers.Linode == nil || !config.Providers.Linode.Enabled {
		return fmt.Errorf("Linode provider is not enabled")
	}

	p.config = config.Providers.Linode

	ctx.Log.Info("Linode provider initialized", nil)
	return nil
}

// CreateNode creates a Linode instance
func (p *LinodeProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	// Generate user data script
	// userData := p.generateUserData(node) // Will be used later when Linode supports cloud-init better

	// Prepare tags
	tags := []string{
		"kubernetes",
		ctx.Stack(),
	}

	// Add role tags
	for _, role := range node.Roles {
		tags = append(tags, fmt.Sprintf("role-%s", role))
	}

	// Add custom tags from config
	tags = append(tags, p.config.Tags...)

	// Add node labels as tags
	for k, v := range node.Labels {
		tags = append(tags, fmt.Sprintf("%s-%s", k, v))
	}

	// Convert tags to pulumi.StringArray
	pulumiTags := make(pulumi.StringArray, len(tags))
	for i, tag := range tags {
		pulumiTags[i] = pulumi.String(tag)
	}

	// Convert authorized keys to pulumi.StringArray
	authorizedKeys := make(pulumi.StringArray, len(p.config.AuthorizedKeys))
	for i, key := range p.config.AuthorizedKeys {
		authorizedKeys[i] = pulumi.String(key)
	}

	// Create instance args
	instanceArgs := &linode.InstanceArgs{
		Label:           pulumi.String(node.Name),
		Region:          pulumi.String(node.Region),
		Type:            pulumi.String(node.Size),
		Image:           pulumi.String(node.Image),
		RootPass:        pulumi.String(generateSecurePassword()),
		AuthorizedKeys:  authorizedKeys,
		PrivateIp:       pulumi.Bool(p.config.PrivateIP),
		Tags:            pulumiTags,
		WatchdogEnabled: pulumi.Bool(true),
		BootConfigLabel: pulumi.String(fmt.Sprintf("%s-config", node.Name)),
	}

	// Create the instance
	instance, err := linode.NewInstance(ctx, node.Name, instanceArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create Linode instance %s: %w", node.Name, err)
	}

	// Get network information

	// Create node output
	output := &NodeOutput{
		ID:          instance.ID(),
		Name:        node.Name,
		PublicIP:    instance.IpAddress,
		PrivateIP:   instance.PrivateIpAddress,
		Provider:    "linode",
		Region:      node.Region,
		Size:        node.Size,
		Status:      instance.Status,
		Labels:      node.Labels,
		WireGuardIP: node.WireGuardIP,
		SSHUser:     "root",
		SSHKeyPath:  "~/.ssh/id_rsa",
	}

	// Export node information
	secrets.Export(ctx,fmt.Sprintf("%s_public_ip", node.Name), instance.IpAddress)
	secrets.Export(ctx,fmt.Sprintf("%s_private_ip", node.Name), instance.PrivateIpAddress)
	secrets.Export(ctx,fmt.Sprintf("%s_id", node.Name), instance.ID())
	secrets.Export(ctx,fmt.Sprintf("%s_status", node.Name), instance.Status)

	p.nodes = append(p.nodes, output)
	return output, nil
}

// CreateNodePool creates multiple nodes
func (p *LinodeProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
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
		// Masters: 10.8.0.12, 10.8.0.13 (Linode)
		// Worker: 10.8.0.16 (Linode)
		if contains(pool.Roles, "controlplane") || contains(pool.Roles, "master") {
			nodeConfig.WireGuardIP = fmt.Sprintf("10.8.0.%d", 12+i)
		} else if contains(pool.Roles, "worker") {
			nodeConfig.WireGuardIP = "10.8.0.16"
		}

		output, err := p.CreateNode(ctx, nodeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", nodeName, err)
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// CreateNetwork creates network infrastructure
func (p *LinodeProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	// Linode uses private IPs for internal networking
	// No explicit network creation needed - instances with private_ip enabled
	// automatically get a private network interface

	// Create a dummy ID output for the network
	networkName := fmt.Sprintf("%s-network", ctx.Stack())

	output := &NetworkOutput{
		ID:     pulumi.ID(networkName).ToIDOutput(),
		Name:   networkName,
		CIDR:   network.CIDR,
		Region: p.config.Region,
	}

	secrets.Export(ctx,"linode_network_name", pulumi.String(output.Name))
	secrets.Export(ctx,"linode_network_cidr", pulumi.String(output.CIDR))

	return output, nil
}

// CreateFirewall creates firewall rules
func (p *LinodeProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	// Convert IDs to int array for Linode
	linodeIds := make(pulumi.IntArray, len(nodeIds))
	for i, id := range nodeIds {
		linodeIds[i] = id.ApplyT(func(id pulumi.ID) int {
			var idInt int
			fmt.Sscanf(string(id), "%d", &idInt)
			return idInt
		}).(pulumi.IntOutput)
	}

	// Build inbound rules
	inboundRules := linode.FirewallInboundArray{}

	// Allow all internal traffic (private network)
	inboundRules = append(inboundRules, &linode.FirewallInboundArgs{
		Label:    pulumi.String("internal-all"),
		Protocol: pulumi.String("TCP"),
		Ports:    pulumi.String("1-65535"),
		Ipv4s: pulumi.StringArray{
			pulumi.String("10.0.0.0/8"),
			pulumi.String("172.16.0.0/12"),
			pulumi.String("192.168.0.0/16"),
		},
	})

	// Allow WireGuard
	inboundRules = append(inboundRules, &linode.FirewallInboundArgs{
		Label:    pulumi.String("wireguard"),
		Protocol: pulumi.String("UDP"),
		Ports:    pulumi.String("51820"),
		Ipv4s:    pulumi.StringArray{pulumi.String("10.8.0.0/24")},
	})

	// Convert custom firewall rules
	for _, rule := range firewall.InboundRules {
		ipv4Sources := make(pulumi.StringArray, len(rule.Source))
		for i, src := range rule.Source {
			ipv4Sources[i] = pulumi.String(src)
		}

		inboundRules = append(inboundRules, &linode.FirewallInboundArgs{
			Label:    pulumi.String(fmt.Sprintf("custom-%s-%s", rule.Protocol, rule.Port)),
			Protocol: pulumi.String(strings.ToUpper(rule.Protocol)),
			Ports:    pulumi.String(rule.Port),
			Action:   pulumi.String("ACCEPT"),
			Ipv4s:    ipv4Sources,
		})
	}

	// Build outbound rules (allow all by default)
	outboundRules := linode.FirewallOutboundArray{
		&linode.FirewallOutboundArgs{
			Label:    pulumi.String("allow-all"),
			Protocol: pulumi.String("TCP"),
			Ports:    pulumi.String("1-65535"),
			Ipv4s:    pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		&linode.FirewallOutboundArgs{
			Label:    pulumi.String("allow-all-udp"),
			Protocol: pulumi.String("UDP"),
			Ports:    pulumi.String("1-65535"),
			Ipv4s:    pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		&linode.FirewallOutboundArgs{
			Label:    pulumi.String("allow-icmp"),
			Protocol: pulumi.String("ICMP"),
			Ipv4s:    pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
	}

	// Create firewall
	fw, err := linode.NewFirewall(ctx, firewall.Name, &linode.FirewallArgs{
		Label:          pulumi.String(firewall.Name),
		Linodes:        linodeIds,
		InboundPolicy:  pulumi.String("DROP"),
		OutboundPolicy: pulumi.String("ACCEPT"),
		Inbounds:       inboundRules,
		Outbounds:      outboundRules,
		Tags:           pulumi.StringArray{pulumi.String("kubernetes"), pulumi.String(ctx.Stack())},
	})
	if err != nil {
		return fmt.Errorf("failed to create firewall: %w", err)
	}

	p.firewall = fw
	secrets.Export(ctx,"linode_firewall_id", fw.ID())

	return nil
}

// CreateLoadBalancer creates a NodeBalancer
func (p *LinodeProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	// Create NodeBalancer
	nodeBalancer, err := linode.NewNodeBalancer(ctx, lb.Name, &linode.NodeBalancerArgs{
		Label:  pulumi.String(lb.Name),
		Region: pulumi.String(p.config.Region),
		Tags:   pulumi.StringArray{pulumi.String("kubernetes"), pulumi.String(ctx.Stack())},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create NodeBalancer: %w", err)
	}

	// Create configs for each port
	for _, port := range lb.Ports {
		configName := fmt.Sprintf("%s-%d", lb.Name, port.Port)

		nbConfig, err := linode.NewNodeBalancerConfig(ctx, configName, &linode.NodeBalancerConfigArgs{
			NodebalancerId: nodeBalancer.ID().ApplyT(func(id pulumi.ID) int {
				var idInt int
				fmt.Sscanf(string(id), "%d", &idInt)
				return idInt
			}).(pulumi.IntOutput),
			Port:          pulumi.Int(port.Port),
			Protocol:      pulumi.String(strings.ToLower(port.Protocol)),
			Algorithm:     pulumi.String("roundrobin"),
			Check:         pulumi.String("http"),
			CheckInterval: pulumi.Int(30),
			CheckTimeout:  pulumi.Int(5),
			CheckAttempts: pulumi.Int(3),
			Stickiness:    pulumi.String("table"),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create NodeBalancer config: %w", err)
		}

		// Add nodes to the config
		for i, node := range p.nodes {
			nodeName := fmt.Sprintf("%s-%d-node-%d", lb.Name, port.Port, i)

			_, err := linode.NewNodeBalancerNode(ctx, nodeName, &linode.NodeBalancerNodeArgs{
				NodebalancerId: nodeBalancer.ID().ApplyT(func(id pulumi.ID) int {
					var idInt int
					fmt.Sscanf(string(id), "%d", &idInt)
					return idInt
				}).(pulumi.IntOutput),
				ConfigId: nbConfig.ID().ApplyT(func(id pulumi.ID) int {
					var idInt int
					fmt.Sscanf(string(id), "%d", &idInt)
					return idInt
				}).(pulumi.IntOutput),
				Address: node.PrivateIP.ApplyT(func(ip string) string {
					return fmt.Sprintf("%s:%d", ip, port.TargetPort)
				}).(pulumi.StringOutput),
				Label:  pulumi.String(nodeName),
				Mode:   pulumi.String("accept"),
				Weight: pulumi.Int(100),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add node to NodeBalancer: %w", err)
			}
		}
	}

	// Get the first IPv4 address
	ipv4 := nodeBalancer.Ipv4

	output := &LoadBalancerOutput{
		ID:       nodeBalancer.ID(),
		IP:       ipv4,
		Hostname: nodeBalancer.Hostname,
		Status:   pulumi.String("active").ToStringOutput(),
	}

	secrets.Export(ctx,fmt.Sprintf("%s_ip", lb.Name), ipv4)
	secrets.Export(ctx,fmt.Sprintf("%s_hostname", lb.Name), nodeBalancer.Hostname)

	return output, nil
}

// GetRegions returns available Linode regions
func (p *LinodeProvider) GetRegions() []string {
	return []string{
		"us-east", "us-west", "us-central", "us-southeast",
		"eu-west", "eu-central", "ap-south", "ap-northeast",
		"ap-west", "ca-central", "ap-southeast",
	}
}

// GetSizes returns available instance sizes
func (p *LinodeProvider) GetSizes() []string {
	return []string{
		"g6-nanode-1", "g6-standard-1", "g6-standard-2", "g6-standard-4",
		"g6-standard-6", "g6-standard-8", "g6-standard-16", "g6-standard-20",
		"g6-dedicated-2", "g6-dedicated-4", "g6-dedicated-8", "g6-dedicated-16",
	}
}

// Cleanup performs cleanup operations
func (p *LinodeProvider) Cleanup(ctx *pulumi.Context) error {
	// Cleanup is handled by Pulumi's resource management
	return nil
}

// generateUserData generates the user data script for the instance
func (p *LinodeProvider) generateUserData(node *config.NodeConfig) string {
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
echo "NODE_PROVIDER=linode" >> /etc/environment
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
	}

	baseScript += "\necho 'Node initialization complete'\n"

	return fmt.Sprintf(baseScript, node.Region, node.Size)
}

// generateSecurePassword generates a secure random password
func generateSecurePassword() string {
	// In production, use a proper password generator
	// This is just a placeholder
	return "ChangeMe123!@#$%^&*()"
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
