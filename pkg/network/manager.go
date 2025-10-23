package network

import (
	"fmt"
	"net"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Manager handles network orchestration across providers
type Manager struct {
	config    *config.NetworkConfig
	providers map[string]providers.Provider
	networks  map[string]*providers.NetworkOutput
	ctx       *pulumi.Context
}

// NewManager creates a new network manager
func NewManager(ctx *pulumi.Context, config *config.NetworkConfig) *Manager {
	return &Manager{
		ctx:       ctx,
		config:    config,
		providers: make(map[string]providers.Provider),
		networks:  make(map[string]*providers.NetworkOutput),
	}
}

// RegisterProvider registers a provider for network management
func (m *Manager) RegisterProvider(name string, provider providers.Provider) {
	m.providers[name] = provider
}

// CreateNetworks creates network infrastructure for all providers
func (m *Manager) CreateNetworks() error {
	for name, provider := range m.providers {
		m.ctx.Log.Info("Creating network for provider", nil)

		network, err := provider.CreateNetwork(m.ctx, m.config)
		if err != nil {
			return fmt.Errorf("failed to create network for %s: %w", name, err)
		}

		m.networks[name] = network
	}

	// If cross-provider networking is enabled, create peering
	if m.config.CrossProviderNetworking {
		if err := m.createCrossProviderPeering(); err != nil {
			return fmt.Errorf("failed to create cross-provider peering: %w", err)
		}
	}

	return nil
}

// CreateFirewalls creates firewall rules for nodes
func (m *Manager) CreateFirewalls(nodes map[string][]*providers.NodeOutput) error {
	for providerName, nodeList := range nodes {
		provider, ok := m.providers[providerName]
		if !ok {
			return fmt.Errorf("provider %s not registered", providerName)
		}

		// Collect node IDs
		nodeIds := make([]pulumi.IDOutput, len(nodeList))
		for i, node := range nodeList {
			nodeIds[i] = node.ID
		}

		// Create firewall config
		firewallConfig := m.createFirewallConfig(providerName)

		if err := provider.CreateFirewall(m.ctx, firewallConfig, nodeIds); err != nil {
			return fmt.Errorf("failed to create firewall for %s: %w", providerName, err)
		}
	}

	return nil
}

// createFirewallConfig creates firewall configuration for a provider
func (m *Manager) createFirewallConfig(providerName string) *config.FirewallConfig {
	firewallConfig := &config.FirewallConfig{
		Name:          fmt.Sprintf("%s-firewall", m.ctx.Stack()),
		InboundRules:  []config.FirewallRule{},
		OutboundRules: []config.FirewallRule{},
	}

	// Add WireGuard rules if enabled
	if m.config.WireGuard != nil && m.config.WireGuard.Enabled {
		firewallConfig.InboundRules = append(firewallConfig.InboundRules, config.FirewallRule{
			Protocol:    "udp",
			Port:        fmt.Sprintf("%d", m.config.WireGuard.Port),
			Source:      []string{"0.0.0.0/0"},
			Description: "WireGuard VPN",
		})

		// Allow all traffic from WireGuard network
		firewallConfig.InboundRules = append(firewallConfig.InboundRules, config.FirewallRule{
			Protocol:    "tcp",
			Port:        "1-65535",
			Source:      []string{"10.8.0.0/24"},
			Description: "Allow all from WireGuard network",
		})

		firewallConfig.InboundRules = append(firewallConfig.InboundRules, config.FirewallRule{
			Protocol:    "udp",
			Port:        "1-65535",
			Source:      []string{"10.8.0.0/24"},
			Description: "Allow all UDP from WireGuard network",
		})
	}

	// Add Kubernetes-specific rules
	firewallConfig.InboundRules = append(firewallConfig.InboundRules, m.getKubernetesFirewallRules()...)

	// Add custom rules from config
	if m.config.Firewall != nil {
		firewallConfig.InboundRules = append(firewallConfig.InboundRules, m.config.Firewall.InboundRules...)
		firewallConfig.OutboundRules = append(firewallConfig.OutboundRules, m.config.Firewall.OutboundRules...)
	}

	return firewallConfig
}

// getKubernetesFirewallRules returns Kubernetes-specific firewall rules
func (m *Manager) getKubernetesFirewallRules() []config.FirewallRule {
	rules := []config.FirewallRule{}

	// Internal cluster communication (only from private networks)
	rules = append(rules, config.FirewallRule{
		Protocol:    "tcp",
		Port:        "6443",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "Kubernetes API server",
	})

	rules = append(rules, config.FirewallRule{
		Protocol:    "tcp",
		Port:        "2379-2380",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "etcd server client API",
	})

	rules = append(rules, config.FirewallRule{
		Protocol:    "tcp",
		Port:        "10250",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "Kubelet API",
	})

	rules = append(rules, config.FirewallRule{
		Protocol:    "tcp",
		Port:        "10251",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "kube-scheduler",
	})

	rules = append(rules, config.FirewallRule{
		Protocol:    "tcp",
		Port:        "10252",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "kube-controller-manager",
	})

	// Calico/Canal networking
	rules = append(rules, config.FirewallRule{
		Protocol:    "udp",
		Port:        "8472",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "Flannel VXLAN",
	})

	rules = append(rules, config.FirewallRule{
		Protocol:    "tcp",
		Port:        "179",
		Source:      []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
		Description: "Calico BGP",
	})

	// NodePort services (if enabled)
	if m.config.EnableNodePorts {
		rules = append(rules, config.FirewallRule{
			Protocol:    "tcp",
			Port:        "30000-32767",
			Source:      []string{"10.8.0.0/24"}, // Only from WireGuard
			Description: "NodePort Services",
		})
	}

	return rules
}

// createCrossProviderPeering creates network peering between providers
func (m *Manager) createCrossProviderPeering() error {
	// This would implement VPC peering or VPN connections between providers
	// For now, we rely on WireGuard for cross-provider connectivity
	m.ctx.Log.Info("Cross-provider networking enabled via WireGuard", nil)
	return nil
}

// GetNetworkByProvider returns the network output for a provider
func (m *Manager) GetNetworkByProvider(provider string) (*providers.NetworkOutput, error) {
	network, ok := m.networks[provider]
	if !ok {
		return nil, fmt.Errorf("network not found for provider %s", provider)
	}
	return network, nil
}

// ValidateCIDRs validates that network CIDRs don't overlap
func (m *Manager) ValidateCIDRs() error {
	cidrs := []string{
		m.config.CIDR,
	}

	// Add Kubernetes CIDRs
	if m.config.PodCIDR != "" {
		cidrs = append(cidrs, m.config.PodCIDR)
	}
	if m.config.ServiceCIDR != "" {
		cidrs = append(cidrs, m.config.ServiceCIDR)
	}

	// Check for overlaps
	for i := 0; i < len(cidrs); i++ {
		for j := i + 1; j < len(cidrs); j++ {
			if overlap, err := cidrOverlap(cidrs[i], cidrs[j]); err != nil {
				return fmt.Errorf("invalid CIDR: %w", err)
			} else if overlap {
				return fmt.Errorf("CIDR overlap detected between %s and %s", cidrs[i], cidrs[j])
			}
		}
	}

	return nil
}

// cidrOverlap checks if two CIDR ranges overlap
func cidrOverlap(cidr1, cidr2 string) (bool, error) {
	_, net1, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR %s: %w", cidr1, err)
	}

	_, net2, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR %s: %w", cidr2, err)
	}

	return net1.Contains(net2.IP) || net2.Contains(net1.IP), nil
}

// AllocateNodeIPs allocates IPs for nodes within the network
func (m *Manager) AllocateNodeIPs(nodeCount int) ([]string, error) {
	_, network, err := net.ParseCIDR(m.config.CIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid network CIDR: %w", err)
	}

	ips := []string{}
	ip := network.IP

	// Skip network and gateway addresses
	for i := 0; i < 2; i++ {
		ip = nextIP(ip)
	}

	for i := 0; i < nodeCount; i++ {
		ip = nextIP(ip)
		if !network.Contains(ip) {
			return nil, fmt.Errorf("ran out of IPs in network %s", m.config.CIDR)
		}
		ips = append(ips, ip.String())
	}

	return ips, nil
}

// nextIP returns the next IP address
func nextIP(ip net.IP) net.IP {
	next := net.IP(make([]byte, len(ip)))
	copy(next, ip)

	for j := len(next) - 1; j >= 0; j-- {
		next[j]++
		if next[j] > 0 {
			break
		}
	}

	return next
}

// GetDNSServers returns DNS servers for the network
func (m *Manager) GetDNSServers() []string {
	if len(m.config.DNSServers) > 0 {
		return m.config.DNSServers
	}

	// Default DNS servers
	return []string{
		"1.1.1.1",
		"8.8.8.8",
	}
}

// ExportNetworkOutputs exports network information to Pulumi stack
func (m *Manager) ExportNetworkOutputs() {
	for provider, network := range m.networks {
		m.ctx.Export(fmt.Sprintf("%s_network_id", provider), network.ID)
		m.ctx.Export(fmt.Sprintf("%s_network_cidr", provider), pulumi.String(network.CIDR))
		m.ctx.Export(fmt.Sprintf("%s_network_region", provider), pulumi.String(network.Region))

		for i, subnet := range network.Subnets {
			m.ctx.Export(fmt.Sprintf("%s_subnet_%d_id", provider, i), subnet.ID)
			m.ctx.Export(fmt.Sprintf("%s_subnet_%d_cidr", provider, i), pulumi.String(subnet.CIDR))
		}
	}

	// Export WireGuard info if enabled
	if m.config.WireGuard != nil && m.config.WireGuard.Enabled {
		m.ctx.Export("wireguard_enabled", pulumi.Bool(true))
		m.ctx.Export("wireguard_port", pulumi.Int(m.config.WireGuard.Port))
		m.ctx.Export("wireguard_network", pulumi.String("10.8.0.0/24"))
	}
}
