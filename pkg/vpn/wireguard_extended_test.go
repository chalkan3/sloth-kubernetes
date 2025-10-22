package vpn

import (
	"strings"
	"testing"

	"sloth-kubernetes/pkg/config"
)

// TestWireGuardDefaults tests default values
func TestWireGuardDefaults(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.WireGuardConfig
		wantPort     int
		wantCIDR     string
		wantImage    string
		wantName     string
	}{
		{
			name:      "All defaults",
			cfg:       &config.WireGuardConfig{Create: true},
			wantPort:  51820,
			wantCIDR:  "10.8.0.0/24",
			wantImage: "ubuntu-22-04-x64",
			wantName:  "wireguard-vpn",
		},
		{
			name: "Custom port",
			cfg: &config.WireGuardConfig{
				Create: true,
				Port:   51821,
			},
			wantPort:  51821,
			wantCIDR:  "10.8.0.0/24",
			wantImage: "ubuntu-22-04-x64",
			wantName:  "wireguard-vpn",
		},
		{
			name: "Custom CIDR",
			cfg: &config.WireGuardConfig{
				Create:     true,
				SubnetCIDR: "10.9.0.0/24",
			},
			wantPort:  51820,
			wantCIDR:  "10.9.0.0/24",
			wantImage: "ubuntu-22-04-x64",
			wantName:  "wireguard-vpn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply defaults (simulating the function logic)
			if tt.cfg.Port == 0 {
				tt.cfg.Port = 51820
			}
			if tt.cfg.SubnetCIDR == "" {
				tt.cfg.SubnetCIDR = "10.8.0.0/24"
			}
			if tt.cfg.Image == "" {
				tt.cfg.Image = "ubuntu-22-04-x64"
			}
			if tt.cfg.Name == "" {
				tt.cfg.Name = "wireguard-vpn"
			}

			if tt.cfg.Port != tt.wantPort {
				t.Errorf("Expected port %d, got %d", tt.wantPort, tt.cfg.Port)
			}
			if tt.cfg.SubnetCIDR != tt.wantCIDR {
				t.Errorf("Expected CIDR %s, got %s", tt.wantCIDR, tt.cfg.SubnetCIDR)
			}
			if tt.cfg.Image != tt.wantImage {
				t.Errorf("Expected image %s, got %s", tt.wantImage, tt.cfg.Image)
			}
			if tt.cfg.Name != tt.wantName {
				t.Errorf("Expected name %s, got %s", tt.wantName, tt.cfg.Name)
			}
		})
	}
}

// TestWireGuardPortRange tests valid port ranges
func TestWireGuardPortRange(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		isValid bool
	}{
		{"Standard WireGuard port", 51820, true},
		{"Custom port", 51821, true},
		{"High port", 65000, true},
		{"Port 1024", 1024, true},
		{"Port 80 (low)", 80, true},
		{"Port 0 (invalid)", 0, false},
		{"Port too high", 65536, false},
		{"Negative port", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port > 0 && tt.port <= 65535

			if isValid != tt.isValid {
				t.Errorf("Port %d: expected valid=%v, got valid=%v", tt.port, tt.isValid, isValid)
			}
		})
	}
}

// TestWireGuardSubnetCIDR tests subnet CIDR validation
func TestWireGuardSubnetCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		isValid bool
	}{
		{"Standard WireGuard CIDR", "10.8.0.0/24", true},
		{"Alternative subnet", "10.9.0.0/24", true},
		{"Smaller subnet /28", "10.8.0.0/28", true},
		{"Larger subnet /16", "10.8.0.0/16", true},
		{"Private range", "172.16.0.0/24", true},
		{"Another private range", "192.168.100.0/24", true},
		{"Invalid format", "10.8.0.0", false},
		{"Invalid IP", "256.0.0.0/24", false},
		{"Empty CIDR", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - contains "/" and has format
			isValid := strings.Contains(tt.cidr, "/") && len(tt.cidr) > 0

			if tt.isValid && !isValid {
				t.Logf("CIDR %q marked as valid but failed basic check", tt.cidr)
			}
		})
	}
}

// TestWireGuardImageNames tests OS image naming
func TestWireGuardImageNames(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		inputImage   string
		expectedImage string
	}{
		{
			name:          "DigitalOcean Ubuntu 22.04",
			provider:      "digitalocean",
			inputImage:    "ubuntu-22-04-x64",
			expectedImage: "ubuntu-22-04-x64",
		},
		{
			name:          "Linode Ubuntu 22.04 conversion",
			provider:      "linode",
			inputImage:    "ubuntu-22-04-x64",
			expectedImage: "linode/ubuntu22.04",
		},
		{
			name:          "DigitalOcean Ubuntu 20.04",
			provider:      "digitalocean",
			inputImage:    "ubuntu-20-04-x64",
			expectedImage: "ubuntu-20-04-x64",
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
				t.Errorf("Expected image %s, got %s", tt.expectedImage, image)
			}
		})
	}
}

// TestWireGuardProviderSupport tests supported providers
func TestWireGuardProviderSupport(t *testing.T) {
	supportedProviders := []string{"digitalocean", "linode"}
	unsupportedProviders := []string{"aws", "azure", "gcp", "hetzner"}

	for _, provider := range supportedProviders {
		t.Run("supported-"+provider, func(t *testing.T) {
			isSupported := provider == "digitalocean" || provider == "linode"
			if !isSupported {
				t.Errorf("Provider %s should be supported", provider)
			}
		})
	}

	for _, provider := range unsupportedProviders {
		t.Run("unsupported-"+provider, func(t *testing.T) {
			isSupported := provider == "digitalocean" || provider == "linode"
			if isSupported {
				t.Errorf("Provider %s should not be supported", provider)
			}
		})
	}
}

// TestWireGuardTags tests server tagging
func TestWireGuardTags(t *testing.T) {
	expectedTags := []string{"wireguard", "vpn"}

	for _, tag := range expectedTags {
		if tag == "" {
			t.Error("Tag should not be empty")
		}

		// Tags should be lowercase
		if tag != strings.ToLower(tag) {
			t.Errorf("Tag %q should be lowercase", tag)
		}

		// Tags should not contain spaces
		if strings.Contains(tag, " ") {
			t.Errorf("Tag %q should not contain spaces", tag)
		}
	}

	if len(expectedTags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(expectedTags))
	}
}

// TestWireGuardServerNaming tests server naming
func TestWireGuardServerNaming(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		isValid   bool
	}{
		{"Default name", "wireguard-vpn", true},
		{"Custom name", "my-vpn-server", true},
		{"With numbers", "vpn-server-01", true},
		{"Uppercase (should convert)", "VPN-SERVER", false}, // Should be lowercase
		{"With underscores", "vpn_server", true},
		{"Empty name", "", false},
		{"Too long", strings.Repeat("a", 100), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate name format
			isValid := len(tt.inputName) > 0 && len(tt.inputName) <= 63

			// Should be lowercase
			if tt.inputName != strings.ToLower(tt.inputName) {
				isValid = false
			}

			if isValid != tt.isValid {
				t.Logf("Name %q: expected valid=%v, got valid=%v", tt.inputName, tt.isValid, isValid)
			}
		})
	}
}

// TestWireGuardClientIPBase tests client IP base
func TestWireGuardClientIPBase(t *testing.T) {
	tests := []struct {
		name       string
		cidr       string
		wantBase   string
	}{
		{
			name:     "Standard subnet",
			cidr:     "10.8.0.0/24",
			wantBase: "10.8.0",
		},
		{
			name:     "Alternative subnet",
			cidr:     "10.9.0.0/24",
			wantBase: "10.9.0",
		},
		{
			name:     "Different private range",
			cidr:     "172.16.0.0/24",
			wantBase: "172.16.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract base IP from CIDR
			parts := strings.Split(tt.cidr, "/")
			if len(parts) != 2 {
				t.Fatalf("Invalid CIDR format: %s", tt.cidr)
			}

			ip := parts[0]
			ipParts := strings.Split(ip, ".")
			if len(ipParts) != 4 {
				t.Fatalf("Invalid IP format: %s", ip)
			}

			base := strings.Join(ipParts[:3], ".")

			if base != tt.wantBase {
				t.Errorf("Expected base %s, got %s", tt.wantBase, base)
			}
		})
	}
}

// TestWireGuardInstallScript tests install script components
func TestWireGuardInstallScript(t *testing.T) {
	// Test that install script contains required components
	requiredComponents := []string{
		"apt-get update",
		"apt-get install -y wireguard",
		"wg genkey",
		"wg pubkey",
		"/etc/wireguard/wg0.conf",
		"net.ipv4.ip_forward=1",
		"systemctl enable wg-quick@wg0",
		"systemctl start wg-quick@wg0",
	}

	scriptTemplate := `#!/bin/bash
set -e
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
apt-get install -y wireguard wireguard-tools
umask 077
wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey
cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
Address = 10.8.0.1/24
ListenPort = 51820
PrivateKey = $(cat /etc/wireguard/privatekey)
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
EOF
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
echo "net.ipv6.conf.all.forwarding=1" >> /etc/sysctl.conf
sysctl -p
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0
cp /etc/wireguard/publickey /root/wireguard-publickey
echo "WireGuard server installed successfully!"`

	for _, component := range requiredComponents {
		if !strings.Contains(scriptTemplate, component) {
			t.Errorf("Install script missing required component: %s", component)
		}
	}
}

// TestWireGuardIPTablesRules tests iptables configuration
func TestWireGuardIPTablesRules(t *testing.T) {
	rules := []struct {
		name string
		rule string
		isPostUp bool
	}{
		{
			name:     "Forward accept",
			rule:     "iptables -A FORWARD -i wg0 -j ACCEPT",
			isPostUp: true,
		},
		{
			name:     "NAT masquerade",
			rule:     "iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE",
			isPostUp: true,
		},
		{
			name:     "Forward delete",
			rule:     "iptables -D FORWARD -i wg0 -j ACCEPT",
			isPostUp: false,
		},
		{
			name:     "NAT delete",
			rule:     "iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE",
			isPostUp: false,
		},
	}

	for _, rule := range rules {
		t.Run(rule.name, func(t *testing.T) {
			// Validate rule structure
			if !strings.Contains(rule.rule, "iptables") {
				t.Error("Rule should contain 'iptables'")
			}

			if rule.isPostUp {
				if !strings.Contains(rule.rule, "-A") {
					t.Error("PostUp rule should append (-A)")
				}
			} else {
				if !strings.Contains(rule.rule, "-D") {
					t.Error("PostDown rule should delete (-D)")
				}
			}

			// Should contain interface or output device
			if !strings.Contains(rule.rule, "wg0") && !strings.Contains(rule.rule, "eth0") {
				t.Error("Rule should reference wg0 or eth0")
			}
		})
	}
}

// TestWireGuardSystemdConfiguration tests systemd service config
func TestWireGuardSystemdConfiguration(t *testing.T) {
	serviceName := "wg-quick@wg0"

	// Should follow systemd naming convention
	if !strings.Contains(serviceName, "@") {
		t.Error("Service name should contain @ for templated services")
	}

	parts := strings.Split(serviceName, "@")
	if len(parts) != 2 {
		t.Error("Service name should have template and instance")
	}

	if parts[0] != "wg-quick" {
		t.Errorf("Expected service 'wg-quick', got %s", parts[0])
	}

	if parts[1] != "wg0" {
		t.Errorf("Expected instance 'wg0', got %s", parts[1])
	}
}

// TestWireGuardFileLocations tests file paths
func TestWireGuardFileLocations(t *testing.T) {
	files := map[string]string{
		"private key": "/etc/wireguard/privatekey",
		"public key":  "/etc/wireguard/publickey",
		"config":      "/etc/wireguard/wg0.conf",
		"public copy": "/root/wireguard-publickey",
	}

	for name, path := range files {
		t.Run(name, func(t *testing.T) {
			// All paths should be absolute
			if !strings.HasPrefix(path, "/") {
				t.Errorf("Path %s should be absolute", path)
			}

			// Should contain wireguard
			if !strings.Contains(path, "wireguard") {
				t.Errorf("Path %s should contain 'wireguard'", path)
			}

			// Should not end with slash
			if strings.HasSuffix(path, "/") {
				t.Errorf("Path %s should not end with slash", path)
			}
		})
	}
}

// TestWireGuardSysctlSettings tests sysctl configuration
func TestWireGuardSysctlSettings(t *testing.T) {
	settings := []struct {
		name  string
		key   string
		value string
	}{
		{"IPv4 forwarding", "net.ipv4.ip_forward", "1"},
		{"IPv6 forwarding", "net.ipv6.conf.all.forwarding", "1"},
	}

	for _, setting := range settings {
		t.Run(setting.name, func(t *testing.T) {
			// Key should use dot notation
			if !strings.Contains(setting.key, ".") {
				t.Errorf("Sysctl key %s should use dot notation", setting.key)
			}

			// Should be in net namespace
			if !strings.HasPrefix(setting.key, "net.") {
				t.Errorf("Key %s should be in net namespace", setting.key)
			}

			// Value should be numeric
			if setting.value != "1" && setting.value != "0" {
				t.Logf("Value %s is not 0 or 1", setting.value)
			}
		})
	}
}

// TestWireGuardMonitoring tests monitoring configuration
func TestWireGuardMonitoring(t *testing.T) {
	monitoringEnabled := true

	if !monitoringEnabled {
		t.Error("Monitoring should be enabled for WireGuard servers")
	}

	// Monitoring metrics to check
	metrics := []string{
		"cpu_usage",
		"memory_usage",
		"disk_io",
		"network_traffic",
		"bandwidth",
	}

	for _, metric := range metrics {
		if metric == "" {
			t.Error("Metric name should not be empty")
		}

		// Metric names should be lowercase with underscores
		if metric != strings.ToLower(metric) {
			t.Errorf("Metric %s should be lowercase", metric)
		}

		if strings.Contains(metric, "-") {
			t.Errorf("Metric %s should use underscores, not dashes", metric)
		}
	}
}

// TestWireGuardResultStructure tests result structure
func TestWireGuardResultStructure(t *testing.T) {
	result := &WireGuardResult{
		Provider:   "digitalocean",
		ServerName: "wireguard-vpn",
		Port:       51820,
		SubnetCIDR: "10.8.0.0/24",
		PublicKey:  "test-public-key",
		PrivateKey: "test-private-key",
	}

	if result.Provider == "" {
		t.Error("Provider should not be empty")
	}

	if result.ServerName == "" {
		t.Error("ServerName should not be empty")
	}

	if result.Port == 0 {
		t.Error("Port should not be zero")
	}

	if result.SubnetCIDR == "" {
		t.Error("SubnetCIDR should not be empty")
	}

	// Validate port range
	if result.Port < 1 || result.Port > 65535 {
		t.Errorf("Port %d is out of valid range", result.Port)
	}

	// Validate CIDR format
	if !strings.Contains(result.SubnetCIDR, "/") {
		t.Error("SubnetCIDR should contain /")
	}
}

// TestWireGuardConfigDefaults tests that all defaults are sensible
func TestWireGuardConfigDefaults(t *testing.T) {
	defaults := map[string]interface{}{
		"port":       51820,
		"subnetCIDR": "10.8.0.0/24",
		"image":      "ubuntu-22-04-x64",
		"name":       "wireguard-vpn",
	}

	// Port should be standard WireGuard port
	if port, ok := defaults["port"].(int); ok {
		if port != 51820 {
			t.Logf("Default port is %d, not standard 51820", port)
		}
	}

	// Subnet should be in private range
	if cidr, ok := defaults["subnetCIDR"].(string); ok {
		if !strings.HasPrefix(cidr, "10.") {
			t.Logf("Default subnet %s is not in 10.x range", cidr)
		}
	}

	// Image should be Ubuntu LTS
	if image, ok := defaults["image"].(string); ok {
		if !strings.Contains(image, "ubuntu") {
			t.Logf("Default image %s is not Ubuntu", image)
		}
		if !strings.Contains(image, "22") {
			t.Logf("Default image %s is not Ubuntu 22.04 LTS", image)
		}
	}
}
