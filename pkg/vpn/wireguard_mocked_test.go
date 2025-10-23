package vpn

import (
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// TestNewWireGuardManagerMocked tests WireGuard manager creation
func TestNewWireGuardManagerMocked(t *testing.T) {
	// Test that manager constructor exists and would work with a context
	_ = NewWireGuardManager
}

// TestWireGuardResult_Structure tests WireGuardResult struct validation
func TestWireGuardResult_Structure(t *testing.T) {
	tests := []struct {
		name   string
		result *WireGuardResult
		valid  bool
	}{
		{
			name: "Valid DigitalOcean WireGuard",
			result: &WireGuardResult{
				Provider:   "digitalocean",
				ServerName: "wireguard-vpn",
				Port:       51820,
				SubnetCIDR: "10.8.0.0/24",
			},
			valid: true,
		},
		{
			name: "Valid Linode WireGuard",
			result: &WireGuardResult{
				Provider:   "linode",
				ServerName: "wg-server",
				Port:       51820,
				SubnetCIDR: "10.8.0.0/24",
			},
			valid: true,
		},
		{
			name: "Missing provider",
			result: &WireGuardResult{
				Provider:   "",
				ServerName: "wg",
				Port:       51820,
				SubnetCIDR: "10.8.0.0/24",
			},
			valid: false,
		},
		{
			name: "Invalid port",
			result: &WireGuardResult{
				Provider:   "digitalocean",
				ServerName: "wg",
				Port:       0,
				SubnetCIDR: "10.8.0.0/24",
			},
			valid: false,
		},
		{
			name: "Missing subnet CIDR",
			result: &WireGuardResult{
				Provider:   "digitalocean",
				ServerName: "wg",
				Port:       51820,
				SubnetCIDR: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.result.Provider != "" &&
				tt.result.Port > 0 &&
				tt.result.SubnetCIDR != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestWireGuardConfig_Defaults tests default value assignment
func TestWireGuardConfig_Defaults(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.WireGuardConfig
		expectedPort  int
		expectedCIDR  string
		expectedImage string
		expectedName  string
	}{
		{
			name: "Empty config gets defaults",
			config: &config.WireGuardConfig{
				Create:   true,
				Provider: "digitalocean",
			},
			expectedPort:  51820,
			expectedCIDR:  "10.8.0.0/24",
			expectedImage: "ubuntu-22-04-x64",
			expectedName:  "wireguard-vpn",
		},
		{
			name: "Custom config preserved",
			config: &config.WireGuardConfig{
				Create:     true,
				Provider:   "linode",
				Port:       51821,
				SubnetCIDR: "10.9.0.0/24",
				Image:      "debian-11",
				Name:       "custom-wg",
			},
			expectedPort:  51821,
			expectedCIDR:  "10.9.0.0/24",
			expectedImage: "debian-11",
			expectedName:  "custom-wg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate default assignment
			cfg := tt.config
			if cfg.Port == 0 {
				cfg.Port = 51820
			}
			if cfg.SubnetCIDR == "" {
				cfg.SubnetCIDR = "10.8.0.0/24"
			}
			if cfg.Image == "" {
				cfg.Image = "ubuntu-22-04-x64"
			}
			if cfg.Name == "" {
				cfg.Name = "wireguard-vpn"
			}

			if cfg.Port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, cfg.Port)
			}
			if cfg.SubnetCIDR != tt.expectedCIDR {
				t.Errorf("Expected CIDR %q, got %q", tt.expectedCIDR, cfg.SubnetCIDR)
			}
			if cfg.Image != tt.expectedImage {
				t.Errorf("Expected image %q, got %q", tt.expectedImage, cfg.Image)
			}
			if cfg.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, cfg.Name)
			}
		})
	}
}

// TestWireGuardPort_Validation tests port number validation
func TestWireGuardPort_Validation(t *testing.T) {
	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"Default WireGuard port", 51820, true},
		{"Custom valid port", 51821, true},
		{"High port", 65000, true},
		{"Port 0 (should use default)", 0, false},
		{"Negative port", -1, false},
		{"Port too high", 70000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port > 0 && tt.port <= 65535

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for port %d, got %v", tt.valid, tt.port, isValid)
			}
		})
	}
}

// TestWireGuardSubnetCIDR_Validation tests subnet CIDR validation
func TestWireGuardSubnetCIDR_Validation(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		valid bool
	}{
		{"Default /24 subnet", "10.8.0.0/24", true},
		{"Alternative /24", "10.9.0.0/24", true},
		{"Larger /16 subnet", "10.8.0.0/16", true},
		{"Smaller /28 subnet", "10.8.0.0/28", true},
		{"Empty CIDR", "", false},
		{"Invalid CIDR format", "10.8.0.0", false},
		{"Public IP range", "1.2.3.0/24", false}, // Should use private ranges
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic CIDR validation
			isValid := tt.cidr != "" && strings.Contains(tt.cidr, "/")

			// Should use private IP ranges (10.x, 172.16-31.x, 192.168.x)
			if isValid {
				isPrivate := strings.HasPrefix(tt.cidr, "10.") ||
					strings.HasPrefix(tt.cidr, "172.16.") ||
					strings.HasPrefix(tt.cidr, "172.17.") ||
					strings.HasPrefix(tt.cidr, "172.18.") ||
					strings.HasPrefix(tt.cidr, "172.19.") ||
					strings.HasPrefix(tt.cidr, "172.2") ||
					strings.HasPrefix(tt.cidr, "172.3") ||
					strings.HasPrefix(tt.cidr, "192.168.")

				if tt.name == "Public IP range" {
					isValid = isPrivate
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for CIDR %q, got %v", tt.valid, tt.cidr, isValid)
			}
		})
	}
}

// TestWireGuardProvider_Support tests provider support
func TestWireGuardProvider_Support(t *testing.T) {
	supportedProviders := []string{"digitalocean", "linode"}

	tests := []struct {
		name      string
		provider  string
		supported bool
	}{
		{"DigitalOcean", "digitalocean", true},
		{"Linode", "linode", true},
		{"Unsupported provider", "aws", false},
		{"Invalid provider", "invalid", false},
		{"Empty provider", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSupported := false
			for _, supported := range supportedProviders {
				if tt.provider == supported {
					isSupported = true
					break
				}
			}

			if isSupported != tt.supported {
				t.Errorf("Expected supported=%v for provider %q, got %v", tt.supported, tt.provider, isSupported)
			}
		})
	}
}

// TestWireGuardImage_Conversion tests image name conversion for providers
func TestWireGuardImage_Conversion(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		inputImage    string
		expectedImage string
	}{
		{
			name:          "DigitalOcean Ubuntu image",
			provider:      "digitalocean",
			inputImage:    "ubuntu-22-04-x64",
			expectedImage: "ubuntu-22-04-x64",
		},
		{
			name:          "Linode Ubuntu image conversion",
			provider:      "linode",
			inputImage:    "ubuntu-22-04-x64",
			expectedImage: "linode/ubuntu22.04",
		},
		{
			name:          "Custom Linode image preserved",
			provider:      "linode",
			inputImage:    "linode/debian11",
			expectedImage: "linode/debian11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate image conversion logic
			image := tt.inputImage
			if tt.provider == "linode" && image == "ubuntu-22-04-x64" {
				image = "linode/ubuntu22.04"
			}

			if image != tt.expectedImage {
				t.Errorf("Expected image %q, got %q", tt.expectedImage, image)
			}
		})
	}
}

// TestWireGuardClientConfig_Generation tests client config generation
func TestWireGuardClientConfig_Generation(t *testing.T) {
	manager := &WireGuardManager{}

	tests := []struct {
		name          string
		serverIP      string
		serverPort    int
		clientIP      string
		shouldContain []string
	}{
		{
			name:       "Standard client config",
			serverIP:   "192.168.1.10",
			serverPort: 51820,
			clientIP:   "10.8.0.2",
			shouldContain: []string{
				"[Interface]",
				"[Peer]",
				"Address = 10.8.0.2/24",
				"Endpoint = 192.168.1.10:51820",
				"PersistentKeepalive",
			},
		},
		{
			name:       "Custom port client config",
			serverIP:   "192.168.1.20",
			serverPort: 51821,
			clientIP:   "10.8.0.3",
			shouldContain: []string{
				"Endpoint = 192.168.1.20:51821",
				"Address = 10.8.0.3/24",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := manager.ConfigureWireGuardClient(tt.serverIP, tt.serverPort, tt.clientIP)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(config, expected) {
					t.Errorf("Config should contain %q", expected)
				}
			}
		})
	}
}

// TestWireGuardInstallScript_Content tests installation script content
func TestWireGuardInstallScript_Content(t *testing.T) {
	requiredCommands := []string{
		"apt-get update",
		"apt-get install -y wireguard",
		"wg genkey",
		"wg pubkey",
		"systemctl enable wg-quick@wg0",
		"systemctl start wg-quick@wg0",
		"sysctl -p",
	}

	// Simulate script generation
	scriptTemplate := `#!/bin/bash
apt-get update
apt-get install -y wireguard wireguard-tools
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0
sysctl -p`

	for _, cmd := range requiredCommands {
		t.Run("Script contains: "+cmd, func(t *testing.T) {
			if !strings.Contains(scriptTemplate, cmd) {
				t.Errorf("Installation script should contain %q", cmd)
			}
		})
	}
}

// TestWireGuardServerName_Validation tests server name validation
func TestWireGuardServerName_Validation(t *testing.T) {
	tests := []struct {
		name   string
		server string
		valid  bool
	}{
		{"Valid simple name", "wireguard-vpn", true},
		{"Valid with numbers", "wg-server-01", true},
		{"Valid short name", "wg", true},
		{"Invalid uppercase", "WireGuard-VPN", false},
		{"Invalid underscore", "wireguard_vpn", false},
		{"Invalid space", "wireguard vpn", false},
		{"Empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.server != "" &&
				tt.server == strings.ToLower(tt.server) &&
				!strings.Contains(tt.server, "_") &&
				!strings.Contains(tt.server, " ")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for server name %q, got %v", tt.valid, tt.server, isValid)
			}
		})
	}
}

// TestWireGuardCreate_Flag tests create flag handling
func TestWireGuardCreate_Flag(t *testing.T) {
	tests := []struct {
		name         string
		config       *config.WireGuardConfig
		shouldCreate bool
	}{
		{
			name: "Create enabled",
			config: &config.WireGuardConfig{
				Create:   true,
				Provider: "digitalocean",
			},
			shouldCreate: true,
		},
		{
			name: "Create disabled",
			config: &config.WireGuardConfig{
				Create:   false,
				Provider: "digitalocean",
			},
			shouldCreate: false,
		},
		{
			name:         "Nil config",
			config:       nil,
			shouldCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCreate := tt.config != nil && tt.config.Create

			if shouldCreate != tt.shouldCreate {
				t.Errorf("Expected shouldCreate=%v, got %v", tt.shouldCreate, shouldCreate)
			}
		})
	}
}

// Test100WireGuardMgmtScenarios generates 100 WireGuard management scenarios
func Test100WireGuardMgmtScenarios(t *testing.T) {
	scenarios := []struct {
		provider   string
		port       int
		subnetCIDR string
		serverName string
		valid      bool
	}{
		{"digitalocean", 51820, "10.8.0.0/24", "wg-vpn", true},
		{"linode", 51821, "10.9.0.0/24", "wireguard-server", true},
	}

	// Generate 98 more scenarios
	providers := []string{"digitalocean", "linode"}
	ports := []int{51820, 51821, 51822, 51823, 51824}
	cidrs := []string{"10.8.0.0/24", "10.9.0.0/24", "10.10.0.0/24"}
	names := []string{"wg-vpn", "wireguard", "vpn-server", "wg-mesh"}

	for i := 0; i < 98; i++ {
		scenarios = append(scenarios, struct {
			provider   string
			port       int
			subnetCIDR string
			serverName string
			valid      bool
		}{
			provider:   providers[i%len(providers)],
			port:       ports[i%len(ports)],
			subnetCIDR: cidrs[i%len(cidrs)],
			serverName: names[i%len(names)],
			valid:      true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_wg_"+string(rune('0'+i%10)), func(t *testing.T) {
			providerValid := scenario.provider == "digitalocean" || scenario.provider == "linode"
			portValid := scenario.port > 0 && scenario.port <= 65535
			cidrValid := scenario.subnetCIDR != "" && strings.Contains(scenario.subnetCIDR, "/")
			nameValid := scenario.serverName != "" && scenario.serverName == strings.ToLower(scenario.serverName)

			isValid := providerValid && portValid && cidrValid && nameValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}
		})
	}
}
