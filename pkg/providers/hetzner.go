// Package providers implements cloud provider integrations
package providers

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/pulumi/pulumi-hcloud/sdk/go/hcloud"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// idToInt converts a pulumi.IDOutput to pulumi.IntOutput for hcloud SDK compatibility
func idToInt(id pulumi.IDOutput) pulumi.IntOutput {
	return id.ApplyT(func(idVal pulumi.ID) int {
		intVal, _ := strconv.Atoi(string(idVal))
		return intVal
	}).(pulumi.IntOutput)
}

// idToIntPtr converts a pulumi.IDOutput to pulumi.IntPtrOutput for optional int fields
func idToIntPtr(id pulumi.IDOutput) pulumi.IntPtrOutput {
	return id.ApplyT(func(idVal pulumi.ID) *int {
		intVal, _ := strconv.Atoi(string(idVal))
		return &intVal
	}).(pulumi.IntPtrOutput)
}

// HetznerProvider implements the Provider interface for Hetzner Cloud
type HetznerProvider struct {
	config         *config.HetznerProvider
	network        *hcloud.Network
	firewall       *hcloud.Firewall
	sshKey         *hcloud.SshKey
	placementGroup *hcloud.PlacementGroup
	nodes          []*NodeOutput
	ctx            *pulumi.Context
	clusterConfig  *config.ClusterConfig
}

// NewHetznerProvider creates a new Hetzner provider instance
func NewHetznerProvider() Provider {
	return &HetznerProvider{
		nodes: make([]*NodeOutput, 0),
	}
}

// GetName returns the provider name
func (p *HetznerProvider) GetName() string {
	return "hetzner"
}

// Initialize sets up the Hetzner provider
func (p *HetznerProvider) Initialize(ctx *pulumi.Context, cfg *config.ClusterConfig) error {
	p.ctx = ctx
	p.clusterConfig = cfg

	if cfg.Providers.Hetzner == nil || !cfg.Providers.Hetzner.Enabled {
		return fmt.Errorf("Hetzner provider is not enabled in configuration")
	}

	p.config = cfg.Providers.Hetzner

	if p.config.Token == "" {
		return fmt.Errorf("Hetzner API token is required")
	}

	ctx.Log.Info("Initializing Hetzner Cloud provider...", nil)

	// Setup SSH keys
	if err := p.setupSSHKeys(ctx); err != nil {
		return fmt.Errorf("failed to setup SSH keys: %w", err)
	}

	// Setup placement group if configured
	if err := p.setupPlacementGroup(ctx); err != nil {
		return fmt.Errorf("failed to setup placement group: %w", err)
	}

	ctx.Log.Info("Hetzner provider initialized successfully", nil)
	return nil
}

// setupSSHKeys creates or imports SSH keys
func (p *HetznerProvider) setupSSHKeys(ctx *pulumi.Context) error {
	// Get SSH public key from security config
	sshPublicKey := ""
	if p.clusterConfig.Security.SSHConfig.PublicKeyPath != "" {
		sshPublicKey = p.clusterConfig.Security.SSHConfig.PublicKeyPath
	}

	// Check if SSHPublicKey was set programmatically
	if p.config.SSHPublicKey != nil {
		switch v := p.config.SSHPublicKey.(type) {
		case string:
			sshPublicKey = v
		case pulumi.StringOutput:
			// Will be resolved later
		}
	}

	if sshPublicKey == "" {
		return fmt.Errorf("SSH public key is required")
	}

	// Create SSH key
	sshKey, err := hcloud.NewSshKey(ctx, "cluster-ssh-key", &hcloud.SshKeyArgs{
		Name:      pulumi.String(fmt.Sprintf("%s-key", p.clusterConfig.Metadata.Name)),
		PublicKey: pulumi.String(sshPublicKey),
		Labels: pulumi.StringMap{
			"cluster": pulumi.String(p.clusterConfig.Metadata.Name),
			"managed": pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create SSH key: %w", err)
	}

	p.sshKey = sshKey
	ctx.Export("hetzner_ssh_key_id", sshKey.ID())
	ctx.Log.Info("SSH key created/imported successfully", nil)

	return nil
}

// setupPlacementGroup creates a placement group for anti-affinity
func (p *HetznerProvider) setupPlacementGroup(ctx *pulumi.Context) error {
	if p.config.PlacementGroup == nil || !p.config.PlacementGroup.Create {
		return nil
	}

	pgName := p.config.PlacementGroup.Name
	if pgName == "" {
		pgName = fmt.Sprintf("%s-pg", p.clusterConfig.Metadata.Name)
	}

	pgType := p.config.PlacementGroup.Type
	if pgType == "" {
		pgType = "spread"
	}

	pg, err := hcloud.NewPlacementGroup(ctx, "cluster-placement-group", &hcloud.PlacementGroupArgs{
		Name: pulumi.String(pgName),
		Type: pulumi.String(pgType),
		Labels: pulumi.StringMap{
			"cluster": pulumi.String(p.clusterConfig.Metadata.Name),
			"managed": pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create placement group: %w", err)
	}

	p.placementGroup = pg
	ctx.Export("hetzner_placement_group_id", pg.ID())
	ctx.Log.Info("Placement group created successfully", nil)

	return nil
}

// CreateNetwork creates a Hetzner Cloud network
func (p *HetznerProvider) CreateNetwork(ctx *pulumi.Context, network *config.NetworkConfig) (*NetworkOutput, error) {
	// Use provider network config if available
	netConfig := p.config.Network
	if netConfig == nil || !netConfig.Create {
		ctx.Log.Info("Network creation not enabled, skipping...", nil)
		return nil, nil
	}

	networkName := netConfig.Name
	if networkName == "" {
		networkName = fmt.Sprintf("%s-network", p.clusterConfig.Metadata.Name)
	}

	ipRange := netConfig.IPRange
	if ipRange == "" {
		ipRange = "10.0.0.0/16"
	}

	// Create the network
	hzNetwork, err := hcloud.NewNetwork(ctx, "cluster-network", &hcloud.NetworkArgs{
		Name:    pulumi.String(networkName),
		IpRange: pulumi.String(ipRange),
		Labels: pulumi.StringMap{
			"cluster": pulumi.String(p.clusterConfig.Metadata.Name),
			"managed": pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	p.network = hzNetwork
	ctx.Export("hetzner_network_id", hzNetwork.ID())

	// Create subnets
	var subnets []SubnetOutput
	for i, subnetCfg := range netConfig.Subnets {
		subnetType := subnetCfg.Type
		if subnetType == "" {
			subnetType = "cloud"
		}

		networkZone := subnetCfg.NetworkZone
		if networkZone == "" {
			networkZone = "eu-central"
		}

		subnet, err := hcloud.NewNetworkSubnet(ctx, fmt.Sprintf("subnet-%d", i), &hcloud.NetworkSubnetArgs{
			NetworkId:   idToInt(hzNetwork.ID()),
			Type:        pulumi.String(subnetType),
			IpRange:     pulumi.String(subnetCfg.IPRange),
			NetworkZone: pulumi.String(networkZone),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create subnet %d: %w", i, err)
		}

		subnets = append(subnets, SubnetOutput{
			ID:   subnet.ID(),
			CIDR: subnetCfg.IPRange,
			Zone: networkZone,
		})
	}

	// If no subnets configured, create a default one
	if len(netConfig.Subnets) == 0 {
		subnet, err := hcloud.NewNetworkSubnet(ctx, "subnet-default", &hcloud.NetworkSubnetArgs{
			NetworkId:   idToInt(hzNetwork.ID()),
			Type:        pulumi.String("cloud"),
			IpRange:     pulumi.String("10.0.1.0/24"),
			NetworkZone: pulumi.String("eu-central"),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create default subnet: %w", err)
		}

		subnets = append(subnets, SubnetOutput{
			ID:   subnet.ID(),
			CIDR: "10.0.1.0/24",
			Zone: "eu-central",
		})
	}

	ctx.Log.Info(fmt.Sprintf("Network created: %s with %d subnets", networkName, len(subnets)), nil)

	return &NetworkOutput{
		ID:      hzNetwork.ID(),
		Name:    networkName,
		CIDR:    ipRange,
		Region:  p.config.Location,
		Subnets: subnets,
	}, nil
}

// CreateNode creates a single Hetzner server
func (p *HetznerProvider) CreateNode(ctx *pulumi.Context, node *config.NodeConfig) (*NodeOutput, error) {
	// Determine server type
	serverType := node.Size
	if serverType == "" {
		serverType = "cpx22" // Default: 2 vCPU, 4GB RAM (AMD shared)
	}

	// Determine image
	image := node.Image
	if image == "" {
		image = "ubuntu-22.04"
	}

	// Determine location
	location := node.Region
	if location == "" {
		location = p.config.Location
		if location == "" {
			location = "fsn1"
		}
	}

	// Generate user data
	userData := p.generateUserData(node)

	// Build labels
	labels := pulumi.StringMap{
		"cluster": pulumi.String(p.clusterConfig.Metadata.Name),
		"managed": pulumi.String("sloth-kubernetes"),
		"role":    pulumi.String(strings.Join(node.Roles, "-")),
	}

	// Add custom labels
	if len(node.Labels) > 0 {
		for k, v := range node.Labels {
			labels[k] = pulumi.String(v)
		}
	}

	// Build server args
	serverArgs := &hcloud.ServerArgs{
		Name:       pulumi.String(node.Name),
		ServerType: pulumi.String(serverType),
		Image:      pulumi.String(image),
		Location:   pulumi.String(location),
		UserData:   pulumi.String(userData),
		Labels:     labels,
		SshKeys: pulumi.StringArray{
			p.sshKey.ID().ToStringOutput(),
		},
		PublicNets: hcloud.ServerPublicNetArray{
			&hcloud.ServerPublicNetArgs{
				Ipv4Enabled: pulumi.Bool(true),
				Ipv6Enabled: pulumi.Bool(true),
			},
		},
	}

	// Add placement group if available
	if p.placementGroup != nil {
		serverArgs.PlacementGroupId = idToIntPtr(p.placementGroup.ID())
	}

	// Create the server
	server, err := hcloud.NewServer(ctx, node.Name, serverArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create server %s: %w", node.Name, err)
	}

	// Attach to network if available (separate resource)
	if p.network != nil {
		_, err := hcloud.NewServerNetwork(ctx, fmt.Sprintf("%s-network", node.Name), &hcloud.ServerNetworkArgs{
			ServerId:  idToInt(server.ID()),
			NetworkId: idToInt(p.network.ID()),
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to attach server to network: %v", err), nil)
		}
	}

	// Build node output
	output := &NodeOutput{
		ID:          server.ID(),
		Name:        node.Name,
		PublicIP:    server.Ipv4Address,
		PrivateIP:   server.Ipv4Address, // Will be updated if network is attached
		Provider:    "hetzner",
		Region:      location,
		Size:        serverType,
		Status:      server.Status,
		Labels:      node.Labels,
		WireGuardIP: node.WireGuardIP,
		SSHUser:     "root", // Hetzner uses root by default
		SSHKeyPath:  p.clusterConfig.Security.SSHConfig.KeyPath,
	}

	p.nodes = append(p.nodes, output)

	// Export outputs
	ctx.Export(fmt.Sprintf("%s_id", node.Name), server.ID())
	ctx.Export(fmt.Sprintf("%s_public_ip", node.Name), server.Ipv4Address)
	ctx.Export(fmt.Sprintf("%s_status", node.Name), server.Status)

	ctx.Log.Info(fmt.Sprintf("Server created: %s (%s) in %s", node.Name, serverType, location), nil)

	return output, nil
}

// CreateNodePool creates a pool of Hetzner servers
func (p *HetznerProvider) CreateNodePool(ctx *pulumi.Context, pool *config.NodePool) ([]*NodeOutput, error) {
	outputs := make([]*NodeOutput, 0, pool.Count)

	// Determine locations for distribution
	locations := pool.Zones
	if len(locations) == 0 {
		locations = []string{"fsn1", "nbg1", "hel1"}
	}

	ctx.Log.Info(fmt.Sprintf("Creating node pool %s with %d nodes across %v", pool.Name, pool.Count, locations), nil)

	for i := 0; i < pool.Count; i++ {
		nodeName := fmt.Sprintf("%s-%d", pool.Name, i)
		location := locations[i%len(locations)]

		// Assign WireGuard IPs based on role
		var wireGuardIP string
		if containsRole(pool.Roles, "master") || containsRole(pool.Roles, "control-plane") {
			wireGuardIP = fmt.Sprintf("10.8.0.%d", 20+i)
		} else if containsRole(pool.Roles, "worker") {
			wireGuardIP = fmt.Sprintf("10.8.0.%d", 30+i)
		} else {
			wireGuardIP = fmt.Sprintf("10.8.0.%d", 50+i)
		}

		nodeConfig := &config.NodeConfig{
			Name:        nodeName,
			Provider:    "hetzner",
			Roles:       pool.Roles,
			Size:        pool.Size,
			Image:       pool.Image,
			Region:      location,
			Labels:      pool.Labels,
			WireGuardIP: wireGuardIP,
		}

		output, err := p.CreateNode(ctx, nodeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", nodeName, err)
		}
		outputs = append(outputs, output)
	}

	ctx.Log.Info(fmt.Sprintf("Node pool %s created with %d nodes", pool.Name, len(outputs)), nil)

	return outputs, nil
}

// CreateFirewall creates firewall rules for the cluster
func (p *HetznerProvider) CreateFirewall(ctx *pulumi.Context, firewall *config.FirewallConfig, nodeIds []pulumi.IDOutput) error {
	if firewall == nil {
		// Create default firewall
		firewall = &config.FirewallConfig{
			Name: fmt.Sprintf("%s-firewall", p.clusterConfig.Metadata.Name),
		}
	}

	// Build firewall rules
	rules := hcloud.FirewallRuleArray{
		// SSH
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("22"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// Kubernetes API
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("6443"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// WireGuard VPN
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("udp"),
			Port:      pulumi.String("51820"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// HTTP
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("80"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// HTTPS
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("443"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// NodePort range
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("30000-32767"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// ICMP (ping)
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("icmp"),
			SourceIps: pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")},
		},
		// etcd
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("2379-2380"),
			SourceIps: pulumi.StringArray{pulumi.String("10.0.0.0/8"), pulumi.String("10.8.0.0/24")},
		},
		// Kubelet
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("10250-10252"),
			SourceIps: pulumi.StringArray{pulumi.String("10.0.0.0/8"), pulumi.String("10.8.0.0/24")},
		},
		// Salt API
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("8000"),
			SourceIps: pulumi.StringArray{pulumi.String("10.0.0.0/8"), pulumi.String("10.8.0.0/24")},
		},
		// Salt Master
		&hcloud.FirewallRuleArgs{
			Direction: pulumi.String("in"),
			Protocol:  pulumi.String("tcp"),
			Port:      pulumi.String("4505-4506"),
			SourceIps: pulumi.StringArray{pulumi.String("10.0.0.0/8"), pulumi.String("10.8.0.0/24")},
		},
	}

	// Add custom inbound rules
	for _, rule := range firewall.InboundRules {
		sourceIPs := pulumi.StringArray{}
		for _, src := range rule.Source {
			sourceIPs = append(sourceIPs, pulumi.String(src))
		}
		if len(sourceIPs) == 0 {
			sourceIPs = pulumi.StringArray{pulumi.String("0.0.0.0/0"), pulumi.String("::/0")}
		}

		rules = append(rules, &hcloud.FirewallRuleArgs{
			Direction:   pulumi.String("in"),
			Protocol:    pulumi.String(strings.ToLower(rule.Protocol)),
			Port:        pulumi.String(rule.Port),
			SourceIps:   sourceIPs,
			Description: pulumi.String(rule.Description),
		})
	}

	// Create firewall
	fw, err := hcloud.NewFirewall(ctx, firewall.Name, &hcloud.FirewallArgs{
		Name:  pulumi.String(firewall.Name),
		Rules: rules,
		Labels: pulumi.StringMap{
			"cluster": pulumi.String(p.clusterConfig.Metadata.Name),
			"managed": pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create firewall: %w", err)
	}

	p.firewall = fw
	ctx.Export("hetzner_firewall_id", fw.ID())

	// Attach firewall to servers using separate attachment resources
	for i, node := range p.nodes {
		_, err := hcloud.NewFirewallAttachment(ctx, fmt.Sprintf("fw-attach-%d", i), &hcloud.FirewallAttachmentArgs{
			FirewallId: idToInt(fw.ID()),
			ServerIds: pulumi.IntArray{
				idToInt(node.ID),
			},
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to attach firewall to node %d: %v", i, err), nil)
		}
	}

	ctx.Log.Info(fmt.Sprintf("Firewall %s created with %d rules", firewall.Name, len(rules)), nil)

	return nil
}

// CreateLoadBalancer creates a Hetzner load balancer
func (p *HetznerProvider) CreateLoadBalancer(ctx *pulumi.Context, lb *config.LoadBalancerConfig) (*LoadBalancerOutput, error) {
	if lb == nil {
		return nil, nil
	}

	lbName := lb.Name
	if lbName == "" {
		lbName = fmt.Sprintf("%s-lb", p.clusterConfig.Metadata.Name)
	}

	location := p.config.Location
	if location == "" {
		location = "fsn1"
	}

	// Create load balancer
	loadBalancer, err := hcloud.NewLoadBalancer(ctx, lbName, &hcloud.LoadBalancerArgs{
		Name:             pulumi.String(lbName),
		LoadBalancerType: pulumi.String("lb11"), // Smallest type
		Location:         pulumi.String(location),
		Labels: pulumi.StringMap{
			"cluster": pulumi.String(p.clusterConfig.Metadata.Name),
			"managed": pulumi.String("sloth-kubernetes"),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	// Add services
	for i, port := range lb.Ports {
		protocol := strings.ToLower(port.Protocol)
		if protocol == "" {
			protocol = "tcp"
		}

		_, err := hcloud.NewLoadBalancerService(ctx, fmt.Sprintf("%s-service-%d", lbName, i), &hcloud.LoadBalancerServiceArgs{
			LoadBalancerId:  loadBalancer.ID().ToStringOutput(),
			Protocol:        pulumi.String(protocol),
			ListenPort:      pulumi.Int(port.Port),
			DestinationPort: pulumi.Int(port.TargetPort),
			HealthCheck: &hcloud.LoadBalancerServiceHealthCheckArgs{
				Protocol: pulumi.String(protocol),
				Port:     pulumi.Int(port.TargetPort),
				Interval: pulumi.Int(15),
				Timeout:  pulumi.Int(10),
				Retries:  pulumi.Int(3),
			},
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to add service %d to load balancer: %v", i, err), nil)
		}
	}

	// Add default Kubernetes API service if no ports specified
	if len(lb.Ports) == 0 {
		_, err := hcloud.NewLoadBalancerService(ctx, fmt.Sprintf("%s-k8s-api", lbName), &hcloud.LoadBalancerServiceArgs{
			LoadBalancerId:  loadBalancer.ID().ToStringOutput(),
			Protocol:        pulumi.String("tcp"),
			ListenPort:      pulumi.Int(6443),
			DestinationPort: pulumi.Int(6443),
			HealthCheck: &hcloud.LoadBalancerServiceHealthCheckArgs{
				Protocol: pulumi.String("tcp"),
				Port:     pulumi.Int(6443),
				Interval: pulumi.Int(15),
				Timeout:  pulumi.Int(10),
				Retries:  pulumi.Int(3),
			},
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to add K8s API service to load balancer: %v", err), nil)
		}
	}

	// Attach to network if available
	if p.network != nil {
		_, err := hcloud.NewLoadBalancerNetwork(ctx, fmt.Sprintf("%s-network", lbName), &hcloud.LoadBalancerNetworkArgs{
			LoadBalancerId: idToInt(loadBalancer.ID()),
			NetworkId:      idToInt(p.network.ID()),
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to attach load balancer to network: %v", err), nil)
		}
	}

	// Add targets
	for i, node := range p.nodes {
		_, err := hcloud.NewLoadBalancerTarget(ctx, fmt.Sprintf("%s-target-%d", lbName, i), &hcloud.LoadBalancerTargetArgs{
			LoadBalancerId: idToInt(loadBalancer.ID()),
			Type:           pulumi.String("server"),
			ServerId:       idToIntPtr(node.ID),
		})
		if err != nil {
			ctx.Log.Warn(fmt.Sprintf("Failed to add target %d to load balancer: %v", i, err), nil)
		}
	}

	ctx.Export("hetzner_lb_id", loadBalancer.ID())
	ctx.Export("hetzner_lb_ipv4", loadBalancer.Ipv4)
	ctx.Export("hetzner_lb_ipv6", loadBalancer.Ipv6)

	ctx.Log.Info(fmt.Sprintf("Load balancer %s created in %s", lbName, location), nil)

	return &LoadBalancerOutput{
		ID:       loadBalancer.ID(),
		IP:       loadBalancer.Ipv4,
		Hostname: loadBalancer.Ipv4, // Hetzner LBs don't have hostnames
		Status:   pulumi.String("active").ToStringOutput(),
	}, nil
}

// GetRegions returns available Hetzner Cloud locations
func (p *HetznerProvider) GetRegions() []string {
	return []string{
		"fsn1", // Falkenstein, Germany
		"nbg1", // Nuremberg, Germany
		"hel1", // Helsinki, Finland
		"ash",  // Ashburn, VA, USA
		"hil",  // Hillsboro, OR, USA
		"sin",  // Singapore
	}
}

// GetSizes returns available Hetzner server types
func (p *HetznerProvider) GetSizes() []string {
	return []string{
		// Shared vCPU AMD (CPX)
		"cpx11", // 2 vCPU, 2 GB RAM
		"cpx21", // 3 vCPU, 4 GB RAM
		"cpx31", // 4 vCPU, 8 GB RAM
		"cpx41", // 8 vCPU, 16 GB RAM
		"cpx51", // 16 vCPU, 32 GB RAM
		// Shared vCPU Intel (CX - new naming)
		"cx23", // 2 vCPU, 4 GB RAM
		"cx33", // 4 vCPU, 8 GB RAM
		"cx43", // 8 vCPU, 16 GB RAM
		"cx53", // 16 vCPU, 32 GB RAM
		// Dedicated vCPU (CCX)
		"ccx13", // 2 vCPU, 8 GB RAM
		"ccx23", // 4 vCPU, 16 GB RAM
		"ccx33", // 8 vCPU, 32 GB RAM
		"ccx43", // 16 vCPU, 64 GB RAM
		"ccx53", // 32 vCPU, 128 GB RAM
		"ccx63", // 48 vCPU, 192 GB RAM
		// ARM64 (CAX)
		"cax11", // 2 vCPU, 4 GB RAM
		"cax21", // 4 vCPU, 8 GB RAM
		"cax31", // 8 vCPU, 16 GB RAM
		"cax41", // 16 vCPU, 32 GB RAM
	}
}

// Cleanup cleans up any resources (Pulumi handles this)
func (p *HetznerProvider) Cleanup(ctx *pulumi.Context) error {
	return nil
}

// generateUserData generates cloud-init user data for server provisioning
func (p *HetznerProvider) generateUserData(node *config.NodeConfig) string {
	script := `#!/bin/bash
set -e

echo "=== Hetzner Cloud Server Provisioning ==="
echo "Node: ` + node.Name + `"
echo "Roles: ` + strings.Join(node.Roles, ", ") + `"

# Update system
apt-get update -qq
apt-get upgrade -y -qq

# Install essential packages
apt-get install -y -qq \
    curl \
    wget \
    git \
    htop \
    vim \
    jq \
    net-tools \
    ca-certificates \
    gnupg \
    lsb-release \
    apt-transport-https \
    software-properties-common

# Enable IP forwarding
echo 'net.ipv4.ip_forward = 1' | tee -a /etc/sysctl.conf
echo 'net.ipv6.conf.all.forwarding = 1' | tee -a /etc/sysctl.conf
sysctl -p

# Disable swap
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab

# Load required kernel modules
cat <<EOF | tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

modprobe overlay
modprobe br_netfilter

# Sysctl settings for Kubernetes
cat <<EOF | tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sysctl --system

# Setup hostname
hostnamectl set-hostname ` + node.Name + `

echo "=== Server provisioning complete ==="
`

	// Check if custom user data is provided
	if node.UserData != "" {
		script = script + "\n# Custom user data\n" + node.UserData
	}

	return base64.StdEncoding.EncodeToString([]byte(script))
}

// containsRole checks if a slice contains a role string (Hetzner-specific to avoid redeclaration)
func containsRole(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
