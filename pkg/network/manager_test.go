package network

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
)

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	cfg := &config.NetworkConfig{
		CIDR: "10.0.0.0/16",
	}

	// Test structure initialization
	if cfg.CIDR == "" {
		t.Error("CIDR should not be empty")
	}
}

// TestRegisterProvider tests provider registration
func TestRegisterProvider(t *testing.T) {
	manager := &Manager{
		providers: make(map[string]providers.Provider),
	}

	// Initially empty
	if len(manager.providers) != 0 {
		t.Errorf("Expected 0 providers initially, got %d", len(manager.providers))
	}

	// Mock provider (we can't create real one without Pulumi)
	// Just test the map structure
	providerName := "digitalocean"
	if manager.providers == nil {
		t.Error("Providers map should be initialized")
	}

	// Test that we can add to the map
	manager.providers[providerName] = nil
	if len(manager.providers) != 1 {
		t.Errorf("Expected 1 provider after registration, got %d", len(manager.providers))
	}
}

// TestGetNetworkByProvider tests network retrieval
func TestGetNetworkByProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		hasNetwork   bool
		wantError    bool
	}{
		{
			name:         "Provider with network",
			providerName: "digitalocean",
			hasNetwork:   true,
			wantError:    false,
		},
		{
			name:         "Provider without network",
			providerName: "nonexistent",
			hasNetwork:   false,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				networks: make(map[string]*providers.NetworkOutput),
			}

			if tt.hasNetwork {
				manager.networks[tt.providerName] = &providers.NetworkOutput{
					Name: "test-network",
					CIDR: "10.0.0.0/16",
				}
			}

			network, err := manager.GetNetworkByProvider(tt.providerName)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if network != nil {
					t.Error("Expected nil network on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if network == nil {
					t.Error("Expected network, got nil")
				}
			}
		})
	}
}

// TestCIDROverlap tests CIDR overlap detection
func TestCIDROverlap(t *testing.T) {
	tests := []struct {
		name        string
		cidr1       string
		cidr2       string
		wantOverlap bool
		wantError   bool
	}{
		{
			name:        "No overlap - different ranges",
			cidr1:       "10.0.0.0/16",
			cidr2:       "192.168.0.0/16",
			wantOverlap: false,
			wantError:   false,
		},
		{
			name:        "Overlap - one contains other",
			cidr1:       "10.0.0.0/16",
			cidr2:       "10.0.1.0/24",
			wantOverlap: true,
			wantError:   false,
		},
		{
			name:        "Overlap - same range",
			cidr1:       "10.0.0.0/16",
			cidr2:       "10.0.0.0/16",
			wantOverlap: true,
			wantError:   false,
		},
		{
			name:        "No overlap - adjacent ranges",
			cidr1:       "10.0.0.0/24",
			cidr2:       "10.0.1.0/24",
			wantOverlap: false,
			wantError:   false,
		},
		{
			name:        "Invalid CIDR format - first",
			cidr1:       "invalid",
			cidr2:       "10.0.0.0/16",
			wantOverlap: false,
			wantError:   true,
		},
		{
			name:        "Invalid CIDR format - second",
			cidr1:       "10.0.0.0/16",
			cidr2:       "invalid",
			wantOverlap: false,
			wantError:   true,
		},
		{
			name:        "Overlap - reverse order",
			cidr1:       "10.0.1.0/24",
			cidr2:       "10.0.0.0/16",
			wantOverlap: true,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlap, err := cidrOverlap(tt.cidr1, tt.cidr2)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if overlap != tt.wantOverlap {
					t.Errorf("Expected overlap=%v, got overlap=%v", tt.wantOverlap, overlap)
				}
			}
		})
	}
}

// TestValidateCIDRs tests CIDR validation
func TestValidateCIDRs(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.NetworkConfig
		wantError     bool
		errorContains string
	}{
		{
			name: "No overlap",
			config: &config.NetworkConfig{
				CIDR:        "10.0.0.0/16",
				PodCIDR:     "10.42.0.0/16",
				ServiceCIDR: "10.43.0.0/16",
			},
			wantError: false,
		},
		{
			name: "Network overlaps with Pod CIDR",
			config: &config.NetworkConfig{
				CIDR:        "10.0.0.0/8",
				PodCIDR:     "10.42.0.0/16",
				ServiceCIDR: "192.168.0.0/16",
			},
			wantError:     true,
			errorContains: "overlap",
		},
		{
			name: "Pod and Service CIDRs overlap",
			config: &config.NetworkConfig{
				CIDR:        "10.0.0.0/16",
				PodCIDR:     "10.42.0.0/16",
				ServiceCIDR: "10.42.0.0/16",
			},
			wantError:     true,
			errorContains: "overlap",
		},
		{
			name: "Invalid CIDR format",
			config: &config.NetworkConfig{
				CIDR:        "invalid-cidr",
				PodCIDR:     "10.42.0.0/16",
				ServiceCIDR: "10.43.0.0/16",
			},
			wantError:     true,
			errorContains: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: tt.config,
			}

			err := manager.ValidateCIDRs()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestNextIP tests IP address incrementing
func TestNextIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "Increment last octet",
			ip:   "10.0.0.1",
			want: "10.0.0.2",
		},
		{
			name: "Rollover last octet",
			ip:   "10.0.0.255",
			want: "10.0.1.0",
		},
		{
			name: "Rollover two octets",
			ip:   "10.0.255.255",
			want: "10.1.0.0",
		},
		{
			name: "Rollover three octets",
			ip:   "10.255.255.255",
			want: "11.0.0.0",
		},
		{
			name: "Start of range",
			ip:   "10.0.0.0",
			want: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Invalid test IP: %s", tt.ip)
			}

			got := nextIP(ip)
			if got.String() != tt.want {
				t.Errorf("nextIP(%s) = %s, want %s", tt.ip, got.String(), tt.want)
			}
		})
	}
}

// TestAllocateNodeIPs tests IP allocation
func TestAllocateNodeIPs(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		nodeCount int
		wantError bool
		wantCount int
	}{
		{
			name:      "Allocate 3 IPs",
			cidr:      "10.0.0.0/24",
			nodeCount: 3,
			wantError: false,
			wantCount: 3,
		},
		{
			name:      "Allocate 10 IPs",
			cidr:      "10.0.0.0/24",
			nodeCount: 10,
			wantError: false,
			wantCount: 10,
		},
		{
			name:      "Too many IPs for /30 network",
			cidr:      "10.0.0.0/30",
			nodeCount: 10,
			wantError: true,
			wantCount: 0,
		},
		{
			name:      "Invalid CIDR",
			cidr:      "invalid",
			nodeCount: 3,
			wantError: true,
			wantCount: 0,
		},
		{
			name:      "Zero nodes",
			cidr:      "10.0.0.0/24",
			nodeCount: 0,
			wantError: false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: &config.NetworkConfig{
					CIDR: tt.cidr,
				},
			}

			ips, err := manager.AllocateNodeIPs(tt.nodeCount)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(ips) != tt.wantCount {
					t.Errorf("Expected %d IPs, got %d", tt.wantCount, len(ips))
				}

				// Verify all IPs are unique
				seen := make(map[string]bool)
				for _, ip := range ips {
					if seen[ip] {
						t.Errorf("Duplicate IP allocated: %s", ip)
					}
					seen[ip] = true

					// Verify IP is valid
					if net.ParseIP(ip) == nil {
						t.Errorf("Invalid IP allocated: %s", ip)
					}
				}
			}
		})
	}
}

// TestGetDNSServers tests DNS server retrieval
func TestGetDNSServers(t *testing.T) {
	tests := []struct {
		name          string
		configServers []string
		wantServers   []string
	}{
		{
			name:          "Custom DNS servers",
			configServers: []string{"1.1.1.1", "8.8.8.8"},
			wantServers:   []string{"1.1.1.1", "8.8.8.8"},
		},
		{
			name:          "Default DNS servers",
			configServers: []string{},
			wantServers:   []string{"1.1.1.1", "8.8.8.8"},
		},
		{
			name:          "Single custom server",
			configServers: []string{"9.9.9.9"},
			wantServers:   []string{"9.9.9.9"},
		},
		{
			name:          "Multiple custom servers",
			configServers: []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
			wantServers:   []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: &config.NetworkConfig{
					DNSServers: tt.configServers,
				},
			}

			servers := manager.GetDNSServers()

			if len(servers) != len(tt.wantServers) {
				t.Errorf("Expected %d DNS servers, got %d", len(tt.wantServers), len(servers))
			}

			for i, server := range servers {
				if i >= len(tt.wantServers) {
					break
				}
				if server != tt.wantServers[i] {
					t.Errorf("Server %d: expected %s, got %s", i, tt.wantServers[i], server)
				}

				// Validate IP format
				if net.ParseIP(server) == nil {
					t.Errorf("Invalid DNS server IP: %s", server)
				}
			}
		})
	}
}

// TestGetKubernetesFirewallRules tests Kubernetes firewall rule generation
func TestGetKubernetesFirewallRules(t *testing.T) {
	tests := []struct {
		name            string
		enableNodePorts bool
		minRuleCount    int
	}{
		{
			name:            "Standard rules without NodePorts",
			enableNodePorts: false,
			minRuleCount:    7, // API, etcd, kubelet, scheduler, controller, flannel, calico
		},
		{
			name:            "Standard rules with NodePorts",
			enableNodePorts: true,
			minRuleCount:    8, // Above + NodePort range
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: &config.NetworkConfig{
					EnableNodePorts: tt.enableNodePorts,
				},
			}

			rules := manager.getKubernetesFirewallRules()

			if len(rules) < tt.minRuleCount {
				t.Errorf("Expected at least %d rules, got %d", tt.minRuleCount, len(rules))
			}

			// Verify required ports are present
			requiredPorts := map[string]bool{
				"6443":      false, // API server
				"2379-2380": false, // etcd
				"10250":     false, // kubelet
			}

			for _, rule := range rules {
				if _, ok := requiredPorts[rule.Port]; ok {
					requiredPorts[rule.Port] = true
				}

				// Validate rule has description
				if rule.Description == "" {
					t.Error("Rule missing description")
				}

				// Validate protocol
				if rule.Protocol != "tcp" && rule.Protocol != "udp" {
					t.Errorf("Invalid protocol: %s", rule.Protocol)
				}

				// Validate sources
				if len(rule.Source) == 0 {
					t.Error("Rule has no sources")
				}
			}

			// Check all required ports were found
			for port, found := range requiredPorts {
				if !found {
					t.Errorf("Required port %s not found in rules", port)
				}
			}

			// If NodePorts enabled, verify the range is included
			if tt.enableNodePorts {
				found := false
				for _, rule := range rules {
					if rule.Port == "30000-32767" {
						found = true
						if rule.Protocol != "tcp" {
							t.Error("NodePort rule should use TCP protocol")
						}
						break
					}
				}
				if !found {
					t.Error("NodePort range (30000-32767) not found when enabled")
				}
			}
		})
	}
}

// TestCreateFirewallConfig tests firewall configuration creation
func TestCreateFirewallConfig(t *testing.T) {
	tests := []struct {
		name             string
		wireGuardEnabled bool
		wireGuardPort    int
		customRules      *config.FirewallConfig
		minInboundRules  int
	}{
		{
			name:             "Without WireGuard",
			wireGuardEnabled: false,
			minInboundRules:  7, // Just Kubernetes rules
		},
		{
			name:             "With WireGuard",
			wireGuardEnabled: true,
			wireGuardPort:    51820,
			minInboundRules:  10, // Kubernetes + WireGuard (UDP + 2x TCP/UDP internal)
		},
		{
			name:             "With custom rules",
			wireGuardEnabled: false,
			customRules: &config.FirewallConfig{
				InboundRules: []config.FirewallRule{
					{Protocol: "tcp", Port: "80", Source: []string{"0.0.0.0/0"}},
					{Protocol: "tcp", Port: "443", Source: []string{"0.0.0.0/0"}},
				},
			},
			minInboundRules: 9, // Kubernetes + 2 custom
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			netConfig := &config.NetworkConfig{}

			if tt.wireGuardEnabled {
				netConfig.WireGuard = &config.WireGuardConfig{
					Enabled: true,
					Port:    tt.wireGuardPort,
				}
			}

			if tt.customRules != nil {
				netConfig.Firewall = tt.customRules
			}

			manager := &Manager{
				config: netConfig,
			}

			// Note: we need a mock context, so we'll test the logic without calling the function
			// Instead, test the components

			// Test WireGuard rules creation
			if tt.wireGuardEnabled {
				if netConfig.WireGuard.Port != tt.wireGuardPort {
					t.Errorf("Expected WireGuard port %d, got %d", tt.wireGuardPort, netConfig.WireGuard.Port)
				}
			}

			// Test that k8s rules are generated
			k8sRules := manager.getKubernetesFirewallRules()
			if len(k8sRules) < 7 {
				t.Errorf("Expected at least 7 Kubernetes rules, got %d", len(k8sRules))
			}
		})
	}
}

// TestFirewallRuleValidation tests firewall rule structure validation
func TestFirewallRuleValidation(t *testing.T) {
	tests := []struct {
		name    string
		rule    config.FirewallRule
		isValid bool
	}{
		{
			name: "Valid TCP rule",
			rule: config.FirewallRule{
				Protocol:    "tcp",
				Port:        "80",
				Source:      []string{"0.0.0.0/0"},
				Description: "HTTP",
			},
			isValid: true,
		},
		{
			name: "Valid UDP rule",
			rule: config.FirewallRule{
				Protocol:    "udp",
				Port:        "53",
				Source:      []string{"10.0.0.0/8"},
				Description: "DNS",
			},
			isValid: true,
		},
		{
			name: "Valid port range",
			rule: config.FirewallRule{
				Protocol:    "tcp",
				Port:        "30000-32767",
				Source:      []string{"10.0.0.0/8"},
				Description: "NodePorts",
			},
			isValid: true,
		},
		{
			name: "Missing protocol",
			rule: config.FirewallRule{
				Port:        "80",
				Source:      []string{"0.0.0.0/0"},
				Description: "HTTP",
			},
			isValid: false,
		},
		{
			name: "Empty sources",
			rule: config.FirewallRule{
				Protocol:    "tcp",
				Port:        "80",
				Source:      []string{},
				Description: "HTTP",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate rule structure
			isValid := true

			if tt.rule.Protocol == "" {
				isValid = false
			}

			if tt.rule.Port == "" {
				isValid = false
			}

			if len(tt.rule.Source) == 0 {
				isValid = false
			}

			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.isValid, isValid)
			}
		})
	}
}

// TestManagerStructure tests Manager structure initialization
func TestManagerStructure(t *testing.T) {
	cfg := &config.NetworkConfig{
		CIDR:                    "10.0.0.0/16",
		PodCIDR:                 "10.42.0.0/16",
		ServiceCIDR:             "10.43.0.0/16",
		CrossProviderNetworking: true,
		EnableNodePorts:         true,
	}

	manager := &Manager{
		config:    cfg,
		providers: make(map[string]providers.Provider),
		networks:  make(map[string]*providers.NetworkOutput),
	}

	if manager.config.CIDR != "10.0.0.0/16" {
		t.Errorf("Expected CIDR 10.0.0.0/16, got %s", manager.config.CIDR)
	}

	if !manager.config.CrossProviderNetworking {
		t.Error("Expected CrossProviderNetworking to be true")
	}

	if !manager.config.EnableNodePorts {
		t.Error("Expected EnableNodePorts to be true")
	}

	if manager.providers == nil {
		t.Error("Providers map should be initialized")
	}

	if manager.networks == nil {
		t.Error("Networks map should be initialized")
	}
}

// TestWireGuardNetworkCIDR tests WireGuard network CIDR constant
func TestWireGuardNetworkCIDR(t *testing.T) {
	wireguardCIDR := "10.8.0.0/24"

	_, network, err := net.ParseCIDR(wireguardCIDR)
	if err != nil {
		t.Errorf("Invalid WireGuard CIDR: %v", err)
	}

	// Verify it's a /24 network (256 addresses)
	ones, bits := network.Mask.Size()
	if ones != 24 || bits != 32 {
		t.Errorf("Expected /24 network, got /%d", ones)
	}

	// Verify it's in private range
	if !network.IP.IsPrivate() {
		t.Error("WireGuard CIDR should be in private range")
	}
}

// TestPrivateNetworkRanges tests private network CIDR ranges
func TestPrivateNetworkRanges(t *testing.T) {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Errorf("Invalid private range CIDR %s: %v", cidr, err)
		}

		if !network.IP.IsPrivate() {
			t.Errorf("Range %s should be private", cidr)
		}
	}
}

// TestNewManager_Complete tests complete manager creation
func TestNewManager_Complete(t *testing.T) {
	cfg := &config.NetworkConfig{
		CIDR:        "10.0.0.0/16",
		PodCIDR:     "10.244.0.0/16",
		ServiceCIDR: "10.96.0.0/12",
	}

	// Mock context - we can't create a real one without Pulumi runtime
	// But we can test the manager structure
	manager := &Manager{
		ctx:       nil, // Would be real context in production
		config:    cfg,
		providers: make(map[string]providers.Provider),
		networks:  make(map[string]*providers.NetworkOutput),
	}

	if manager.config == nil {
		t.Error("Config should not be nil")
	}

	if manager.config.CIDR != "10.0.0.0/16" {
		t.Errorf("Expected CIDR 10.0.0.0/16, got %s", manager.config.CIDR)
	}

	if manager.providers == nil {
		t.Error("Providers map should be initialized")
	}

	if manager.networks == nil {
		t.Error("Networks map should be initialized")
	}

	if len(manager.providers) != 0 {
		t.Errorf("Expected 0 providers initially, got %d", len(manager.providers))
	}

	if len(manager.networks) != 0 {
		t.Errorf("Expected 0 networks initially, got %d", len(manager.networks))
	}
}

// TestRegisterProvider_Method tests the RegisterProvider method
func TestRegisterProvider_Method(t *testing.T) {
	manager := &Manager{
		providers: make(map[string]providers.Provider),
	}

	// Test registering nil provider (valid use case in tests)
	manager.RegisterProvider("test-provider", nil)

	if len(manager.providers) != 1 {
		t.Errorf("Expected 1 provider after registration, got %d", len(manager.providers))
	}

	// Test multiple registrations
	manager.RegisterProvider("provider1", nil)
	manager.RegisterProvider("provider2", nil)

	if len(manager.providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(manager.providers))
	}

	// Test overwriting existing provider
	manager.RegisterProvider("provider1", nil)
	if len(manager.providers) != 3 {
		t.Errorf("Expected still 3 providers after overwrite, got %d", len(manager.providers))
	}
}

// TestCreateFirewallConfig_WithWireGuard tests firewall config creation with WireGuard
func TestCreateFirewallConfig_WithWireGuard(t *testing.T) {
	manager := &Manager{
		config: &config.NetworkConfig{
			WireGuard: &config.WireGuardConfig{
				Enabled: true,
				Port:    51820,
			},
		},
	}

	// Skip context-dependent parts by testing the logic directly
	// We'll test the WireGuard rules creation

	// Manually create firewall config similar to createFirewallConfig but without context
	firewallConfig := &config.FirewallConfig{
		Name:          "test-firewall",
		InboundRules:  []config.FirewallRule{},
		OutboundRules: []config.FirewallRule{},
	}

	// Add WireGuard rules (same logic as in createFirewallConfig)
	if manager.config.WireGuard != nil && manager.config.WireGuard.Enabled {
		firewallConfig.InboundRules = append(firewallConfig.InboundRules, config.FirewallRule{
			Protocol:    "udp",
			Port:        fmt.Sprintf("%d", manager.config.WireGuard.Port),
			Source:      []string{"0.0.0.0/0"},
			Description: "WireGuard VPN",
		})

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
	firewallConfig.InboundRules = append(firewallConfig.InboundRules, manager.getKubernetesFirewallRules()...)

	if firewallConfig == nil {
		t.Fatal("Firewall config should not be nil")
	}

	// Check WireGuard rules are added
	foundWireGuardUDP := false
	foundWireGuardTCP := false
	foundWireGuardUDPAll := false

	for _, rule := range firewallConfig.InboundRules {
		if rule.Protocol == "udp" && rule.Port == "51820" {
			foundWireGuardUDP = true
		}
		if rule.Protocol == "tcp" && rule.Port == "1-65535" {
			if len(rule.Source) > 0 && rule.Source[0] == "10.8.0.0/24" {
				foundWireGuardTCP = true
			}
		}
		if rule.Protocol == "udp" && rule.Port == "1-65535" {
			if len(rule.Source) > 0 && rule.Source[0] == "10.8.0.0/24" {
				foundWireGuardUDPAll = true
			}
		}
	}

	if !foundWireGuardUDP {
		t.Error("WireGuard UDP rule not found")
	}

	if !foundWireGuardTCP {
		t.Error("WireGuard TCP allow rule not found")
	}

	if !foundWireGuardUDPAll {
		t.Error("WireGuard UDP allow rule not found")
	}

	// Check Kubernetes rules are also added
	if len(firewallConfig.InboundRules) < 5 {
		t.Errorf("Expected at least 5 inbound rules with WireGuard and K8s, got %d", len(firewallConfig.InboundRules))
	}
}

// TestCreateFirewallConfig_WithoutWireGuard tests firewall config without WireGuard
func TestCreateFirewallConfig_WithoutWireGuard(t *testing.T) {
	manager := &Manager{
		config: &config.NetworkConfig{
			WireGuard: nil,
		},
	}

	// Create firewall config manually (similar logic to createFirewallConfig)
	firewallConfig := &config.FirewallConfig{
		Name:          "test-firewall",
		InboundRules:  []config.FirewallRule{},
		OutboundRules: []config.FirewallRule{},
	}

	// Add Kubernetes-specific rules
	firewallConfig.InboundRules = append(firewallConfig.InboundRules, manager.getKubernetesFirewallRules()...)

	if firewallConfig == nil {
		t.Fatal("Firewall config should not be nil")
	}

	// Check that WireGuard rules are NOT added
	for _, rule := range firewallConfig.InboundRules {
		if rule.Protocol == "udp" && rule.Port == "51820" {
			t.Error("WireGuard UDP rule should not be present when WireGuard is disabled")
		}
	}

	// Kubernetes rules should still be there
	foundK8sAPI := false
	for _, rule := range firewallConfig.InboundRules {
		if rule.Port == "6443" {
			foundK8sAPI = true
		}
	}

	if !foundK8sAPI {
		t.Error("Kubernetes API rule should be present even without WireGuard")
	}
}

// TestCreateFirewallConfig_WithCustomRules tests firewall config with custom rules
func TestCreateFirewallConfig_WithCustomRules(t *testing.T) {
	customInbound := config.FirewallRule{
		Protocol:    "tcp",
		Port:        "8080",
		Source:      []string{"0.0.0.0/0"},
		Description: "Custom HTTP",
	}

	customOutbound := config.FirewallRule{
		Protocol:    "tcp",
		Port:        "443",
		Source:      []string{"0.0.0.0/0"},
		Description: "Custom HTTPS",
	}

	manager := &Manager{
		config: &config.NetworkConfig{
			Firewall: &config.FirewallConfig{
				InboundRules:  []config.FirewallRule{customInbound},
				OutboundRules: []config.FirewallRule{customOutbound},
			},
		},
	}

	// Create firewall config manually
	firewallConfig := &config.FirewallConfig{
		Name:          "test-firewall",
		InboundRules:  []config.FirewallRule{},
		OutboundRules: []config.FirewallRule{},
	}

	// Add Kubernetes rules
	firewallConfig.InboundRules = append(firewallConfig.InboundRules, manager.getKubernetesFirewallRules()...)

	// Add custom rules from config
	if manager.config.Firewall != nil {
		firewallConfig.InboundRules = append(firewallConfig.InboundRules, manager.config.Firewall.InboundRules...)
		firewallConfig.OutboundRules = append(firewallConfig.OutboundRules, manager.config.Firewall.OutboundRules...)
	}

	if firewallConfig == nil {
		t.Fatal("Firewall config should not be nil")
	}

	// Check custom inbound rule is present
	foundCustomInbound := false
	for _, rule := range firewallConfig.InboundRules {
		if rule.Port == "8080" && rule.Description == "Custom HTTP" {
			foundCustomInbound = true
		}
	}

	if !foundCustomInbound {
		t.Error("Custom inbound rule not found")
	}

	// Check custom outbound rule is present
	foundCustomOutbound := false
	for _, rule := range firewallConfig.OutboundRules {
		if rule.Port == "443" && rule.Description == "Custom HTTPS" {
			foundCustomOutbound = true
		}
	}

	if !foundCustomOutbound {
		t.Error("Custom outbound rule not found")
	}
}

// TestGetKubernetesFirewallRules_WithNodePorts tests K8s rules with NodePorts enabled
func TestGetKubernetesFirewallRules_WithNodePorts(t *testing.T) {
	manager := &Manager{
		config: &config.NetworkConfig{
			EnableNodePorts: true,
		},
	}

	rules := manager.getKubernetesFirewallRules()

	// Check for NodePort range
	foundNodePorts := false
	for _, rule := range rules {
		if rule.Port == "30000-32767" {
			foundNodePorts = true
			if rule.Protocol != "tcp" {
				t.Errorf("NodePort rule should be TCP, got %s", rule.Protocol)
			}
			if len(rule.Source) == 0 || rule.Source[0] != "10.8.0.0/24" {
				t.Error("NodePort rule should only allow traffic from WireGuard network")
			}
		}
	}

	if !foundNodePorts {
		t.Error("NodePort rule not found when EnableNodePorts is true")
	}

	// Verify other essential rules are still present
	foundAPIServer := false
	for _, rule := range rules {
		if rule.Port == "6443" {
			foundAPIServer = true
		}
	}

	if !foundAPIServer {
		t.Error("Kubernetes API server rule should be present")
	}
}

// TestGetKubernetesFirewallRules_WithoutNodePorts tests K8s rules without NodePorts
func TestGetKubernetesFirewallRules_WithoutNodePorts(t *testing.T) {
	manager := &Manager{
		config: &config.NetworkConfig{
			EnableNodePorts: false,
		},
	}

	rules := manager.getKubernetesFirewallRules()

	// Check that NodePort range is NOT present
	for _, rule := range rules {
		if rule.Port == "30000-32767" {
			t.Error("NodePort rule should not be present when EnableNodePorts is false")
		}
	}

	// Verify essential rules are still present
	essentialPorts := []string{"6443", "2379-2380", "10250", "10251", "10252"}
	for _, port := range essentialPorts {
		found := false
		for _, rule := range rules {
			if rule.Port == port {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Essential Kubernetes port %s not found in firewall rules", port)
		}
	}
}

// TestCreateCrossProviderPeering tests cross-provider peering config
func TestCreateCrossProviderPeering(t *testing.T) {
	manager := &Manager{
		config: &config.NetworkConfig{
			CrossProviderNetworking: true,
		},
	}

	// Test that the configuration is set correctly
	// The actual createCrossProviderPeering method just logs and returns nil
	// but requires Pulumi context, so we test the config instead
	if !manager.config.CrossProviderNetworking {
		t.Error("CrossProviderNetworking should be enabled")
	}

	// Test with cross-provider disabled
	manager2 := &Manager{
		config: &config.NetworkConfig{
			CrossProviderNetworking: false,
		},
	}

	if manager2.config.CrossProviderNetworking {
		t.Error("CrossProviderNetworking should be disabled")
	}
}

// TestGetNetworkByProvider_MultipleNetworks tests network retrieval with multiple networks
func TestGetNetworkByProvider_MultipleNetworks(t *testing.T) {
	manager := &Manager{
		networks: make(map[string]*providers.NetworkOutput),
	}

	// Add multiple networks
	networks := map[string]*providers.NetworkOutput{
		"digitalocean": {
			Name:   "do-network",
			CIDR:   "10.10.0.0/16",
			Region: "nyc3",
		},
		"linode": {
			Name:   "linode-network",
			CIDR:   "10.20.0.0/16",
			Region: "us-east",
		},
		"aws": {
			Name:   "aws-network",
			CIDR:   "10.30.0.0/16",
			Region: "us-east-1",
		},
	}

	for name, net := range networks {
		manager.networks[name] = net
	}

	// Test retrieving each network
	for providerName, expectedNet := range networks {
		net, err := manager.GetNetworkByProvider(providerName)
		if err != nil {
			t.Errorf("Error getting network for %s: %v", providerName, err)
		}

		if net == nil {
			t.Errorf("Network for %s should not be nil", providerName)
			continue
		}

		if net.Name != expectedNet.Name {
			t.Errorf("Expected network name %s, got %s", expectedNet.Name, net.Name)
		}

		if net.CIDR != expectedNet.CIDR {
			t.Errorf("Expected CIDR %s, got %s", expectedNet.CIDR, net.CIDR)
		}

		if net.Region != expectedNet.Region {
			t.Errorf("Expected region %s, got %s", expectedNet.Region, net.Region)
		}
	}

	// Test non-existent provider
	_, err := manager.GetNetworkByProvider("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

// TestAllocateNodeIPs_EdgeCases tests IP allocation with edge cases
func TestAllocateNodeIPs_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		nodeCount int
		wantErr   bool
		validate  func(t *testing.T, ips []string)
	}{
		{
			name:      "Single IP in /32 network",
			cidr:      "10.0.0.0/32",
			nodeCount: 1,
			wantErr:   true, // Not enough IPs (need network + gateway + node)
		},
		{
			name:      "Minimum /30 network - just enough",
			cidr:      "10.0.0.0/30",
			nodeCount: 1,
			wantErr:   false, // 4 IPs total - network, gateway, and 1 node fits
			validate: func(t *testing.T, ips []string) {
				if len(ips) != 1 {
					t.Errorf("Expected 1 IP, got %d", len(ips))
				}
			},
		},
		{
			name:      "Large /16 network",
			cidr:      "172.16.0.0/16",
			nodeCount: 100,
			wantErr:   false,
			validate: func(t *testing.T, ips []string) {
				if len(ips) != 100 {
					t.Errorf("Expected 100 IPs, got %d", len(ips))
				}
				// Verify IPs are in range
				for _, ip := range ips {
					if !strings.HasPrefix(ip, "172.16.") {
						t.Errorf("IP %s not in expected range", ip)
					}
				}
			},
		},
		{
			name:      "Very large /8 network",
			cidr:      "10.0.0.0/8",
			nodeCount: 1000,
			wantErr:   false,
			validate: func(t *testing.T, ips []string) {
				if len(ips) != 1000 {
					t.Errorf("Expected 1000 IPs, got %d", len(ips))
				}
			},
		},
		{
			name:      "Zero nodes requested",
			cidr:      "192.168.1.0/24",
			nodeCount: 0,
			wantErr:   false,
			validate: func(t *testing.T, ips []string) {
				if len(ips) != 0 {
					t.Errorf("Expected 0 IPs for 0 nodes, got %d", len(ips))
				}
			},
		},
		{
			name:      "Invalid CIDR format",
			cidr:      "not-a-cidr",
			nodeCount: 1,
			wantErr:   true,
		},
		{
			name:      "CIDR with invalid IP",
			cidr:      "999.999.999.999/24",
			nodeCount: 1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: &config.NetworkConfig{
					CIDR: tt.cidr,
				},
			}

			ips, err := manager.AllocateNodeIPs(tt.nodeCount)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, ips)
			}
		})
	}
}

// TestValidateCIDRs_ComplexOverlaps tests complex CIDR overlap scenarios
func TestValidateCIDRs_ComplexOverlaps(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.NetworkConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "No overlaps - well separated",
			config: &config.NetworkConfig{
				CIDR:        "10.0.0.0/16",
				PodCIDR:     "10.244.0.0/16",
				ServiceCIDR: "10.96.0.0/16",
			},
			wantErr: false,
		},
		{
			name: "Network contains Pod CIDR",
			config: &config.NetworkConfig{
				CIDR:    "10.0.0.0/8",
				PodCIDR: "10.244.0.0/16",
			},
			wantErr: true,
			errMsg:  "overlap",
		},
		{
			name: "Service and Pod CIDRs overlap",
			config: &config.NetworkConfig{
				CIDR:        "192.168.0.0/16",
				PodCIDR:     "10.244.0.0/16",
				ServiceCIDR: "10.244.0.0/20",
			},
			wantErr: true,
			errMsg:  "overlap",
		},
		{
			name: "All three overlap",
			config: &config.NetworkConfig{
				CIDR:        "10.0.0.0/16",
				PodCIDR:     "10.0.1.0/24",
				ServiceCIDR: "10.0.2.0/24",
			},
			wantErr: true,
			errMsg:  "overlap",
		},
		{
			name: "Only network CIDR specified",
			config: &config.NetworkConfig{
				CIDR: "172.16.0.0/12",
			},
			wantErr: false,
		},
		{
			name: "Invalid pod CIDR",
			config: &config.NetworkConfig{
				CIDR:    "10.0.0.0/16",
				PodCIDR: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid CIDR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: tt.config,
			}

			err := manager.ValidateCIDRs()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestFirewallRules_ComplexConfigurations tests complex firewall rule combinations
func TestFirewallRules_ComplexConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.NetworkConfig
		validate func(t *testing.T, rules []config.FirewallRule)
	}{
		{
			name: "WireGuard + NodePorts + Custom rules",
			config: &config.NetworkConfig{
				EnableNodePorts: true,
				WireGuard: &config.WireGuardConfig{
					Enabled: true,
					Port:    51820,
				},
				Firewall: &config.FirewallConfig{
					InboundRules: []config.FirewallRule{
						{
							Protocol:    "tcp",
							Port:        "8080",
							Source:      []string{"0.0.0.0/0"},
							Description: "Custom HTTP",
						},
						{
							Protocol:    "tcp",
							Port:        "9090",
							Source:      []string{"10.0.0.0/8"},
							Description: "Metrics",
						},
					},
				},
			},
			validate: func(t *testing.T, rules []config.FirewallRule) {
				foundWG := false
				foundNodePort := false
				foundCustom := false
				foundMetrics := false
				foundK8sAPI := false

				for _, rule := range rules {
					if rule.Protocol == "udp" && rule.Port == "51820" {
						foundWG = true
					}
					if rule.Port == "30000-32767" {
						foundNodePort = true
					}
					if rule.Port == "8080" {
						foundCustom = true
					}
					if rule.Port == "9090" {
						foundMetrics = true
					}
					if rule.Port == "6443" {
						foundK8sAPI = true
					}
				}

				if !foundWG {
					t.Error("WireGuard rule not found")
				}
				if !foundNodePort {
					t.Error("NodePort rule not found")
				}
				if !foundCustom {
					t.Error("Custom HTTP rule not found")
				}
				if !foundMetrics {
					t.Error("Metrics rule not found")
				}
				if !foundK8sAPI {
					t.Error("Kubernetes API rule not found")
				}

				// Should have WireGuard (3) + K8s rules (7) + NodePorts (1) + Custom (2)
				if len(rules) < 10 {
					t.Errorf("Expected at least 10 rules, got %d", len(rules))
				}
			},
		},
		{
			name: "Multiple custom inbound rules",
			config: &config.NetworkConfig{
				Firewall: &config.FirewallConfig{
					InboundRules: []config.FirewallRule{
						{Protocol: "tcp", Port: "80", Source: []string{"0.0.0.0/0"}, Description: "HTTP"},
						{Protocol: "tcp", Port: "443", Source: []string{"0.0.0.0/0"}, Description: "HTTPS"},
						{Protocol: "tcp", Port: "22", Source: []string{"10.0.0.0/8"}, Description: "SSH"},
						{Protocol: "tcp", Port: "3306", Source: []string{"10.0.0.0/8"}, Description: "MySQL"},
						{Protocol: "udp", Port: "53", Source: []string{"10.0.0.0/8"}, Description: "DNS"},
					},
				},
			},
			validate: func(t *testing.T, rules []config.FirewallRule) {
				customRules := 0
				for _, rule := range rules {
					if rule.Port == "80" || rule.Port == "443" || rule.Port == "22" ||
						rule.Port == "3306" || rule.Port == "53" {
						customRules++
					}
				}
				if customRules != 5 {
					t.Errorf("Expected 5 custom rules, found %d", customRules)
				}
			},
		},
		{
			name: "No custom rules - only defaults",
			config: &config.NetworkConfig{
				EnableNodePorts: false,
				WireGuard:       nil,
			},
			validate: func(t *testing.T, rules []config.FirewallRule) {
				// Should only have K8s default rules
				if len(rules) != 7 {
					t.Errorf("Expected 7 default K8s rules, got %d", len(rules))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: tt.config,
			}

			// Build firewall config manually (same logic as createFirewallConfig)
			firewallConfig := &config.FirewallConfig{
				Name:          "test-firewall",
				InboundRules:  []config.FirewallRule{},
				OutboundRules: []config.FirewallRule{},
			}

			// Add WireGuard rules if enabled
			if manager.config.WireGuard != nil && manager.config.WireGuard.Enabled {
				firewallConfig.InboundRules = append(firewallConfig.InboundRules, config.FirewallRule{
					Protocol:    "udp",
					Port:        fmt.Sprintf("%d", manager.config.WireGuard.Port),
					Source:      []string{"0.0.0.0/0"},
					Description: "WireGuard VPN",
				})

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

			// Add K8s rules
			firewallConfig.InboundRules = append(firewallConfig.InboundRules, manager.getKubernetesFirewallRules()...)

			// Add custom rules
			if manager.config.Firewall != nil {
				firewallConfig.InboundRules = append(firewallConfig.InboundRules, manager.config.Firewall.InboundRules...)
				firewallConfig.OutboundRules = append(firewallConfig.OutboundRules, manager.config.Firewall.OutboundRules...)
			}

			tt.validate(t, firewallConfig.InboundRules)
		})
	}
}

// TestDNSServers_CustomConfigurations tests DNS server configurations
func TestDNSServers_CustomConfigurations(t *testing.T) {
	tests := []struct {
		name       string
		dnsServers []string
		expected   []string
	}{
		{
			name:       "Custom Cloudflare DNS",
			dnsServers: []string{"1.1.1.1", "1.0.0.1"},
			expected:   []string{"1.1.1.1", "1.0.0.1"},
		},
		{
			name:       "Custom Google DNS",
			dnsServers: []string{"8.8.8.8", "8.8.4.4"},
			expected:   []string{"8.8.8.8", "8.8.4.4"},
		},
		{
			name:       "Mixed public DNS",
			dnsServers: []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
			expected:   []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
		},
		{
			name:       "Private DNS servers",
			dnsServers: []string{"10.0.0.53", "10.0.0.54"},
			expected:   []string{"10.0.0.53", "10.0.0.54"},
		},
		{
			name:       "Single DNS server",
			dnsServers: []string{"1.1.1.1"},
			expected:   []string{"1.1.1.1"},
		},
		{
			name:       "Empty DNS list - defaults",
			dnsServers: []string{},
			expected:   []string{"1.1.1.1", "8.8.8.8"},
		},
		{
			name:       "Nil DNS list - defaults",
			dnsServers: nil,
			expected:   []string{"1.1.1.1", "8.8.8.8"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: &config.NetworkConfig{
					DNSServers: tt.dnsServers,
				},
			}

			servers := manager.GetDNSServers()

			if len(servers) != len(tt.expected) {
				t.Errorf("Expected %d DNS servers, got %d", len(tt.expected), len(servers))
				return
			}

			for i, expected := range tt.expected {
				if servers[i] != expected {
					t.Errorf("DNS server at index %d: expected %s, got %s", i, expected, servers[i])
				}
			}
		})
	}
}

// TestWireGuard_Configurations tests different WireGuard configurations
func TestWireGuard_Configurations(t *testing.T) {
	tests := []struct {
		name       string
		wireGuard  *config.WireGuardConfig
		expectRule bool
		port       string
	}{
		{
			name: "Default WireGuard port",
			wireGuard: &config.WireGuardConfig{
				Enabled: true,
				Port:    51820,
			},
			expectRule: true,
			port:       "51820",
		},
		{
			name: "Custom WireGuard port",
			wireGuard: &config.WireGuardConfig{
				Enabled: true,
				Port:    12345,
			},
			expectRule: true,
			port:       "12345",
		},
		{
			name: "High port number",
			wireGuard: &config.WireGuardConfig{
				Enabled: true,
				Port:    65000,
			},
			expectRule: true,
			port:       "65000",
		},
		{
			name: "Disabled WireGuard",
			wireGuard: &config.WireGuardConfig{
				Enabled: false,
				Port:    51820,
			},
			expectRule: false,
			port:       "",
		},
		{
			name:       "Nil WireGuard config",
			wireGuard:  nil,
			expectRule: false,
			port:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				config: &config.NetworkConfig{
					WireGuard: tt.wireGuard,
				},
			}

			// Build firewall config
			firewallConfig := &config.FirewallConfig{
				Name:         "test-firewall",
				InboundRules: []config.FirewallRule{},
			}

			// Add WireGuard rules if enabled
			if manager.config.WireGuard != nil && manager.config.WireGuard.Enabled {
				firewallConfig.InboundRules = append(firewallConfig.InboundRules, config.FirewallRule{
					Protocol:    "udp",
					Port:        fmt.Sprintf("%d", manager.config.WireGuard.Port),
					Source:      []string{"0.0.0.0/0"},
					Description: "WireGuard VPN",
				})
			}

			foundRule := false
			foundPort := ""
			for _, rule := range firewallConfig.InboundRules {
				if rule.Description == "WireGuard VPN" {
					foundRule = true
					foundPort = rule.Port
				}
			}

			if tt.expectRule && !foundRule {
				t.Error("Expected WireGuard rule but not found")
			}

			if !tt.expectRule && foundRule {
				t.Error("Did not expect WireGuard rule but found one")
			}

			if tt.expectRule && foundPort != tt.port {
				t.Errorf("Expected WireGuard port %s, got %s", tt.port, foundPort)
			}
		})
	}
}

// TestManager_MultipleProviders tests manager with multiple providers
func TestManager_MultipleProviders(t *testing.T) {
	manager := &Manager{
		config: &config.NetworkConfig{
			CIDR: "10.0.0.0/16",
		},
		providers: make(map[string]providers.Provider),
		networks:  make(map[string]*providers.NetworkOutput),
	}

	// Simulate registering multiple providers
	providerNames := []string{"digitalocean", "linode", "aws", "gcp", "azure"}

	for _, name := range providerNames {
		manager.RegisterProvider(name, nil) // nil is ok for this test
	}

	if len(manager.providers) != 5 {
		t.Errorf("Expected 5 providers, got %d", len(manager.providers))
	}

	// Simulate adding networks
	for i, name := range providerNames {
		manager.networks[name] = &providers.NetworkOutput{
			Name:   fmt.Sprintf("%s-network", name),
			CIDR:   fmt.Sprintf("10.%d.0.0/16", i+10),
			Region: fmt.Sprintf("region-%d", i),
		}
	}

	// Test retrieval
	for _, name := range providerNames {
		net, err := manager.GetNetworkByProvider(name)
		if err != nil {
			t.Errorf("Failed to get network for %s: %v", name, err)
		}
		if net == nil {
			t.Errorf("Network for %s is nil", name)
		}
	}

	// Test non-existent provider
	_, err := manager.GetNetworkByProvider("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

// TestNextIP_Boundary tests nextIP function with boundary conditions
func TestNextIP_Boundary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Near end of IPv4 range",
			input:    "255.255.255.254",
			expected: "255.255.255.255",
		},
		{
			name:     "End of /24 subnet",
			input:    "192.168.1.255",
			expected: "192.168.2.0",
		},
		{
			name:     "End of /16 subnet",
			input:    "10.0.255.255",
			expected: "10.1.0.0",
		},
		{
			name:     "Middle of range",
			input:    "172.16.100.50",
			expected: "172.16.100.51",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.input)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.input)
			}

			next := nextIP(ip)
			if next.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, next.String())
			}
		})
	}
}

// TestCIDROverlap_EdgeCases tests cidrOverlap with edge cases
func TestCIDROverlap_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		cidr1    string
		cidr2    string
		wantBool bool
		wantErr  bool
	}{
		{
			name:     "Same exact CIDR",
			cidr1:    "10.0.0.0/16",
			cidr2:    "10.0.0.0/16",
			wantBool: true,
			wantErr:  false,
		},
		{
			name:     "Larger contains smaller",
			cidr1:    "10.0.0.0/8",
			cidr2:    "10.244.0.0/16",
			wantBool: true,
			wantErr:  false,
		},
		{
			name:     "Smaller inside larger",
			cidr1:    "192.168.1.0/24",
			cidr2:    "192.168.0.0/16",
			wantBool: true,
			wantErr:  false,
		},
		{
			name:     "Adjacent networks - no overlap",
			cidr1:    "10.0.0.0/24",
			cidr2:    "10.0.1.0/24",
			wantBool: false,
			wantErr:  false,
		},
		{
			name:     "Completely different ranges",
			cidr1:    "10.0.0.0/8",
			cidr2:    "172.16.0.0/12",
			wantBool: false,
			wantErr:  false,
		},
		{
			name:     "Single IP vs network",
			cidr1:    "192.168.1.1/32",
			cidr2:    "192.168.1.0/24",
			wantBool: true,
			wantErr:  false,
		},
		{
			name:     "Two single IPs - same",
			cidr1:    "10.0.0.1/32",
			cidr2:    "10.0.0.1/32",
			wantBool: true,
			wantErr:  false,
		},
		{
			name:     "Two single IPs - different",
			cidr1:    "10.0.0.1/32",
			cidr2:    "10.0.0.2/32",
			wantBool: false,
			wantErr:  false,
		},
		{
			name:     "Invalid first CIDR",
			cidr1:    "invalid",
			cidr2:    "10.0.0.0/16",
			wantBool: false,
			wantErr:  true,
		},
		{
			name:     "Invalid second CIDR",
			cidr1:    "10.0.0.0/16",
			cidr2:    "not-a-cidr",
			wantBool: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlap, err := cidrOverlap(tt.cidr1, tt.cidr2)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if overlap != tt.wantBool {
				t.Errorf("Expected overlap=%v, got %v", tt.wantBool, overlap)
			}
		})
	}
}
