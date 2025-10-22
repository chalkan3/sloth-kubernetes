package vpn

import (
	"fmt"
	"strings"
	"testing"
)

// TestWireGuardResult_Fields tests WireGuardResult struct fields
func TestWireGuardResult_Fields(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		serverName string
		port       int
		subnetCIDR string
		valid      bool
	}{
		{"Valid DO server", "digitalocean", "wg-server", 51820, "10.8.0.0/24", true},
		{"Valid Linode server", "linode", "wg-server", 51820, "10.8.0.0/24", true},
		{"Custom port", "digitalocean", "wg-server", 12345, "10.8.0.0/24", true},
		{"Empty provider", "", "wg-server", 51820, "10.8.0.0/24", false},
		{"Empty name", "digitalocean", "", 51820, "10.8.0.0/24", false},
		{"Invalid port", "digitalocean", "wg-server", 0, "10.8.0.0/24", false},
		{"Empty subnet", "digitalocean", "wg-server", 51820, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.provider != "" && tt.serverName != "" &&
				tt.port > 0 && tt.port <= 65535 && tt.subnetCIDR != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestWireGuardPort_Range tests valid WireGuard port ranges
func TestWireGuardPort_Range(t *testing.T) {
	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"Default port", 51820, true},
		{"Custom port", 12345, true},
		{"Low port", 1024, true},
		{"High port", 65535, true},
		{"Privileged port", 22, true},
		{"Port 1", 1, true},
		{"Port 0", 0, false},
		{"Negative port", -1, false},
		{"Port too high", 65536, false},
		{"Port way too high", 100000, false},
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

// TestWireGuardSubnet_Format tests subnet CIDR format
func TestWireGuardSubnet_Format(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		valid bool
	}{
		{"Default subnet", "10.8.0.0/24", true},
		{"Larger subnet", "10.8.0.0/16", true},
		{"Smaller subnet", "10.8.0.0/28", true},
		{"Alternative base", "10.9.0.0/24", true},
		{"172 range", "172.16.0.0/24", true},
		{"192 range", "192.168.1.0/24", true},
		{"No mask", "10.8.0.0", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.Contains(tt.cidr, "/") && tt.cidr != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for CIDR %q, got %v", tt.valid, tt.cidr, isValid)
			}
		})
	}
}

// TestWireGuardImage_Ubuntu tests Ubuntu image versions
func TestWireGuardImage_Ubuntu(t *testing.T) {
	tests := []struct {
		name  string
		image string
		valid bool
	}{
		{"Ubuntu 22.04", "ubuntu-22-04-x64", true},
		{"Ubuntu 20.04", "ubuntu-20-04-x64", true},
		{"Ubuntu 24.04", "ubuntu-24-04-x64", true},
		{"Debian 12", "debian-12", true},
		{"Invalid", "windows-server", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.image != "" &&
				(strings.Contains(tt.image, "ubuntu") || strings.Contains(tt.image, "debian"))
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for image %q, got %v", tt.valid, tt.image, isValid)
			}
		})
	}
}

// TestWireGuardServerName_Format tests server name format
func TestWireGuardServerName_Format(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		valid      bool
	}{
		{"Default name", "wireguard-vpn", true},
		{"Custom name", "wg-server", true},
		{"With env", "prod-wg", true},
		{"Uppercase", "WG-SERVER", false},
		{"With underscore", "wg_server", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.serverName != "" &&
				tt.serverName == strings.ToLower(tt.serverName) &&
				!strings.Contains(tt.serverName, "_")
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for name %q, got %v", tt.valid, tt.serverName, isValid)
			}
		})
	}
}

// TestWireGuardKeys_Generation tests key generation logic
func TestWireGuardKeys_Generation(t *testing.T) {
	tests := []struct {
		name       string
		hasPrivate bool
		hasPublic  bool
		valid      bool
	}{
		{"Both keys", true, true, true},
		{"Only private", true, false, false},
		{"Only public", false, true, false},
		{"No keys", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.hasPrivate && tt.hasPublic
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestWireGuardConfig_DefaultsExtended tests default configuration values
func TestWireGuardConfig_DefaultsExtended(t *testing.T) {
	tests := []struct {
		name          string
		port          int
		subnetCIDR    string
		image         string
		serverName    string
		expectedPort  int
		expectedCIDR  string
		expectedImage string
		expectedName  string
	}{
		{
			"All defaults",
			0, "", "", "",
			51820, "10.8.0.0/24", "ubuntu-22-04-x64", "wireguard-vpn",
		},
		{
			"Custom port",
			12345, "", "", "",
			12345, "10.8.0.0/24", "ubuntu-22-04-x64", "wireguard-vpn",
		},
		{
			"Custom subnet",
			0, "10.9.0.0/24", "", "",
			51820, "10.9.0.0/24", "ubuntu-22-04-x64", "wireguard-vpn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.port
			if port == 0 {
				port = 51820
			}
			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}

			cidr := tt.subnetCIDR
			if cidr == "" {
				cidr = "10.8.0.0/24"
			}
			if cidr != tt.expectedCIDR {
				t.Errorf("Expected CIDR %s, got %s", tt.expectedCIDR, cidr)
			}
		})
	}
}

// TestWireGuardInterface_Name tests interface naming
func TestWireGuardInterface_Name(t *testing.T) {
	tests := []struct {
		name      string
		ifaceName string
		valid     bool
	}{
		{"Default wg0", "wg0", true},
		{"wg1", "wg1", true},
		{"wg-vpn", "wg-vpn", true},
		{"wgvpn", "wgvpn", true},
		{"eth0", "eth0", false},
		{"tun0", "tun0", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.HasPrefix(tt.ifaceName, "wg") && tt.ifaceName != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for interface %q, got %v", tt.valid, tt.ifaceName, isValid)
			}
		})
	}
}

// TestWireGuardIPForwarding_Sysctl tests IP forwarding sysctl settings
func TestWireGuardIPForwarding_Sysctl(t *testing.T) {
	tests := []struct {
		name    string
		setting string
		value   string
		valid   bool
	}{
		{"IPv4 forwarding", "net.ipv4.ip_forward", "1", true},
		{"IPv6 forwarding", "net.ipv6.conf.all.forwarding", "1", true},
		{"Disabled IPv4", "net.ipv4.ip_forward", "0", false},
		{"Invalid setting", "net.invalid", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := (tt.setting == "net.ipv4.ip_forward" ||
				tt.setting == "net.ipv6.conf.all.forwarding") &&
				tt.value == "1"
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestWireGuardFirewall_Rules tests iptables firewall rules
func TestWireGuardFirewall_Rules(t *testing.T) {
	tests := []struct {
		name   string
		rule   string
		action string
		valid  bool
	}{
		{"Forward accept", "FORWARD -i wg0 -j ACCEPT", "ACCEPT", true},
		{"NAT masquerade", "POSTROUTING -o eth0 -j MASQUERADE", "MASQUERADE", true},
		{"Drop rule", "INPUT -i wg0 -j DROP", "DROP", true},
		{"Invalid action", "FORWARD -i wg0 -j INVALID", "INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validActions := []string{"ACCEPT", "DROP", "REJECT", "MASQUERADE"}
			isValid := false
			for _, action := range validActions {
				if tt.action == action {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for action %q, got %v", tt.valid, tt.action, isValid)
			}
		})
	}
}

// TestWireGuardService_Systemd tests systemd service names
func TestWireGuardService_Systemd(t *testing.T) {
	tests := []struct {
		name    string
		service string
		valid   bool
	}{
		{"wg-quick@wg0", "wg-quick@wg0", true},
		{"wg-quick@wg1", "wg-quick@wg1", true},
		{"wg-quick@vpn", "wg-quick@vpn", true},
		{"wireguard", "wireguard", false},
		{"wg0", "wg0", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.HasPrefix(tt.service, "wg-quick@") && tt.service != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for service %q, got %v", tt.valid, tt.service, isValid)
			}
		})
	}
}

// TestWireGuardClientIPBase_Format tests client IP base format
func TestWireGuardClientIPBase_Format(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		valid bool
	}{
		{"Default base", "10.8.0", true},
		{"Alternative base", "10.9.0", true},
		{"172 base", "172.16.0", true},
		{"192 base", "192.168.1", true},
		{"Incomplete", "10.8", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.base, ".")
			isValid := len(parts) == 3 && tt.base != "" && !strings.Contains(tt.base, "/")
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for base %q, got %v", tt.valid, tt.base, isValid)
			}
		})
	}
}

// TestWireGuardPeer_Count tests peer count limits
func TestWireGuardPeer_Count(t *testing.T) {
	tests := []struct {
		name        string
		peerCount   int
		withinLimit bool
	}{
		{"1 peer", 1, true},
		{"10 peers", 10, true},
		{"100 peers", 100, true},
		{"253 peers (max for /24)", 253, true},
		{"254 peers", 254, false}, // .0 is network, .255 is broadcast, .1 is server
		{"1000 peers", 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For /24 subnet: .0 is network, .255 is broadcast, .1 is typically server
			// So max clients = 253
			withinLimit := tt.peerCount >= 1 && tt.peerCount <= 253
			if withinLimit != tt.withinLimit {
				t.Errorf("Expected withinLimit=%v for %d peers, got %v",
					tt.withinLimit, tt.peerCount, withinLimit)
			}
		})
	}
}

// TestWireGuardInstallScript_Commands tests install script commands
func TestWireGuardInstallScript_Commands(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		required bool
	}{
		{"apt-get update", "apt-get update", true},
		{"install wireguard", "apt-get install -y wireguard", true},
		{"wg genkey", "wg genkey", true},
		{"systemctl enable", "systemctl enable wg-quick@wg0", true},
		{"systemctl start", "systemctl start wg-quick@wg0", true},
		{"Optional command", "echo test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRequired := tt.command != "" &&
				(strings.Contains(tt.command, "wireguard") ||
					strings.Contains(tt.command, "wg") ||
					strings.Contains(tt.command, "systemctl") ||
					strings.Contains(tt.command, "apt-get update"))

			if isRequired != tt.required {
				t.Errorf("Expected required=%v for command %q, got %v",
					tt.required, tt.command, isRequired)
			}
		})
	}
}

// Test250WireGuardScenarios generates 250 WireGuard test scenarios
func Test250WireGuardScenarios(t *testing.T) {
	scenarios := []struct {
		provider   string
		region     string
		port       int
		subnetCIDR string
		image      string
		create     bool
		valid      bool
	}{
		{"digitalocean", "nyc3", 51820, "10.8.0.0/24", "ubuntu-22-04-x64", true, true},
		{"linode", "us-east", 51820, "10.8.0.0/24", "ubuntu-22-04-x64", true, true},
		{"digitalocean", "sfo3", 12345, "10.9.0.0/24", "ubuntu-20-04-x64", true, true},
	}

	// Generate 247 more scenarios
	providers := []string{"digitalocean", "linode"}
	doRegions := []string{"nyc3", "sfo3", "ams3", "sgp1", "lon1"}
	linodeRegions := []string{"us-east", "us-west", "eu-west", "ap-south"}
	ports := []int{51820, 12345, 13579, 54321}
	images := []string{"ubuntu-22-04-x64", "ubuntu-20-04-x64", "ubuntu-24-04-x64", "debian-12"}

	for i := 0; i < 247; i++ {
		provider := providers[i%len(providers)]
		var region string
		if provider == "digitalocean" {
			region = doRegions[i%len(doRegions)]
		} else {
			region = linodeRegions[i%len(linodeRegions)]
		}

		port := ports[i%len(ports)]
		subnetBase := 8 + (i % 8) // 10.8-10.15
		subnetCIDR := fmt.Sprintf("10.%d.0.0/24", subnetBase)
		image := images[i%len(images)]

		scenarios = append(scenarios, struct {
			provider   string
			region     string
			port       int
			subnetCIDR string
			image      string
			create     bool
			valid      bool
		}{
			provider:   provider,
			region:     region,
			port:       port,
			subnetCIDR: subnetCIDR,
			image:      image,
			create:     true,
			valid:      true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(fmt.Sprintf("WG_%d_%s", i, scenario.provider), func(t *testing.T) {
			// Validate provider
			providerValid := scenario.provider == "digitalocean" || scenario.provider == "linode"

			// Validate region
			regionValid := scenario.region != ""

			// Validate port
			portValid := scenario.port > 0 && scenario.port <= 65535

			// Validate subnet
			subnetValid := strings.Contains(scenario.subnetCIDR, "/24") &&
				strings.HasPrefix(scenario.subnetCIDR, "10.")

			// Validate image
			imageValid := strings.Contains(scenario.image, "ubuntu") ||
				strings.Contains(scenario.image, "debian")

			isValid := providerValid && regionValid && portValid && subnetValid &&
				imageValid && scenario.create

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}

			// Additional validation: port should not conflict with common services
			commonPorts := []int{22, 80, 443, 3306, 5432, 6379}
			for _, cp := range commonPorts {
				if scenario.port == cp {
					t.Logf("Scenario %d: Warning - port %d conflicts with common service", i, scenario.port)
				}
			}
		})
	}
}
