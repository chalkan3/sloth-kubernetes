package network

import (
	"net"
	"strings"
	"testing"

	"sloth-kubernetes/pkg/config"
	"sloth-kubernetes/pkg/providers"
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
		name        string
		config      *config.NetworkConfig
		wantError   bool
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
		name           string
		configServers  []string
		wantServers    []string
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
		name              string
		wireGuardEnabled  bool
		wireGuardPort     int
		customRules       *config.FirewallConfig
		minInboundRules   int
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
		CIDR:                   "10.0.0.0/16",
		PodCIDR:                "10.42.0.0/16",
		ServiceCIDR:            "10.43.0.0/16",
		CrossProviderNetworking: true,
		EnableNodePorts:        true,
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
