package vpn

import (
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

func TestNewWireGuardManager(t *testing.T) {
	manager := NewWireGuardManager(nil)
	if manager == nil {
		t.Error("NewWireGuardManager should not return nil")
	}
}

func TestWireGuardConfig_Creation(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.WireGuardConfig
		wantCreate bool
	}{
		{
			name: "Auto-create enabled",
			cfg: &config.WireGuardConfig{
				Create:     true,
				Provider:   "digitalocean",
				Region:     "nyc3",
				Size:       "s-1vcpu-1gb",
				Port:       51820,
				SubnetCIDR: "10.8.0.0/24",
			},
			wantCreate: true,
		},
		{
			name: "Auto-create disabled",
			cfg: &config.WireGuardConfig{
				Create:  false,
				Enabled: true,
			},
			wantCreate: false,
		},
		{
			name: "Existing server",
			cfg: &config.WireGuardConfig{
				Enabled:         true,
				ServerEndpoint:  "1.2.3.4:51820",
				ServerPublicKey: "test-key",
			},
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.Create != tt.wantCreate {
				t.Errorf("Expected Create=%v, got Create=%v", tt.wantCreate, tt.cfg.Create)
			}
		})
	}
}

func TestWireGuardConfig_DefaultValues(t *testing.T) {
	cfg := &config.WireGuardConfig{
		Create:   true,
		Provider: "digitalocean",
		Region:   "nyc3",
	}

	// Test default port
	if cfg.Port == 0 {
		cfg.Port = 51820
	}
	if cfg.Port != 51820 {
		t.Errorf("Expected default port 51820, got %d", cfg.Port)
	}

	// Test default subnet
	if cfg.SubnetCIDR == "" {
		cfg.SubnetCIDR = "10.8.0.0/24"
	}
	if cfg.SubnetCIDR != "10.8.0.0/24" {
		t.Errorf("Expected default subnet '10.8.0.0/24', got '%s'", cfg.SubnetCIDR)
	}

	// Test default image
	if cfg.Image == "" {
		cfg.Image = "ubuntu-22-04-x64"
	}
	if cfg.Image != "ubuntu-22-04-x64" {
		t.Errorf("Expected default image 'ubuntu-22-04-x64', got '%s'", cfg.Image)
	}

	// Test default name
	if cfg.Name == "" {
		cfg.Name = "wireguard-vpn"
	}
	if cfg.Name != "wireguard-vpn" {
		t.Errorf("Expected default name 'wireguard-vpn', got '%s'", cfg.Name)
	}
}

func TestWireGuardConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.WireGuardConfig
		wantErr bool
	}{
		{
			name: "Valid config for creation",
			cfg: &config.WireGuardConfig{
				Create:     true,
				Provider:   "digitalocean",
				Region:     "nyc3",
				Size:       "s-1vcpu-1gb",
				Port:       51820,
				SubnetCIDR: "10.8.0.0/24",
			},
			wantErr: false,
		},
		{
			name: "Missing provider",
			cfg: &config.WireGuardConfig{
				Create: true,
				Region: "nyc3",
			},
			wantErr: true,
		},
		{
			name: "Missing region",
			cfg: &config.WireGuardConfig{
				Create:   true,
				Provider: "digitalocean",
			},
			wantErr: true,
		},
		{
			name: "Valid existing server config",
			cfg: &config.WireGuardConfig{
				Enabled:         true,
				ServerEndpoint:  "1.2.3.4:51820",
				ServerPublicKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "Missing endpoint for existing",
			cfg: &config.WireGuardConfig{
				Enabled:         true,
				ServerPublicKey: "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := false
			if tt.cfg.Create {
				if tt.cfg.Provider == "" || tt.cfg.Region == "" {
					hasError = true
				}
			} else if tt.cfg.Enabled {
				if tt.cfg.ServerEndpoint == "" {
					hasError = true
				}
			}

			if tt.wantErr && !hasError {
				t.Error("Expected validation error but got none")
			}
			if !tt.wantErr && hasError {
				t.Error("Expected no validation error but got one")
			}
		})
	}
}

func TestWireGuardResult(t *testing.T) {
	result := &WireGuardResult{
		Provider:   "digitalocean",
		ServerName: "wireguard-vpn",
		Port:       51820,
		SubnetCIDR: "10.8.0.0/24",
	}

	if result.Provider != "digitalocean" {
		t.Errorf("Expected provider 'digitalocean', got '%s'", result.Provider)
	}
	if result.ServerName != "wireguard-vpn" {
		t.Errorf("Expected name 'wireguard-vpn', got '%s'", result.ServerName)
	}
	if result.Port != 51820 {
		t.Errorf("Expected port 51820, got %d", result.Port)
	}
	if result.SubnetCIDR != "10.8.0.0/24" {
		t.Errorf("Expected subnet '10.8.0.0/24', got '%s'", result.SubnetCIDR)
	}
}

func TestConfigureWireGuardClient(t *testing.T) {
	manager := &WireGuardManager{}

	config := manager.ConfigureWireGuardClient("1.2.3.4", 51820, "10.8.0.2")

	if !strings.Contains(config, "1.2.3.4:51820") {
		t.Error("Client config should contain server endpoint")
	}
	if !strings.Contains(config, "10.8.0.2/24") {
		t.Error("Client config should contain client IP")
	}
	if !strings.Contains(config, "[Interface]") {
		t.Error("Client config should contain [Interface] section")
	}
	if !strings.Contains(config, "[Peer]") {
		t.Error("Client config should contain [Peer] section")
	}
	if !strings.Contains(config, "AllowedIPs") {
		t.Error("Client config should contain AllowedIPs")
	}
	if !strings.Contains(config, "PersistentKeepalive") {
		t.Error("Client config should contain PersistentKeepalive")
	}
}

func TestGetWireGuardInstallCommand(t *testing.T) {
	manager := &WireGuardManager{}

	cmd := manager.GetWireGuardInstallCommand("peer1", "pubkey123", "10.8.0.2")

	if !strings.Contains(cmd, "peer1") {
		t.Error("Command should contain peer name")
	}
	if !strings.Contains(cmd, "pubkey123") {
		t.Error("Command should contain public key")
	}
	if !strings.Contains(cmd, "10.8.0.2/32") {
		t.Error("Command should contain peer IP with /32")
	}
	if !strings.Contains(cmd, "wg set") {
		t.Error("Command should contain 'wg set'")
	}
	if !strings.Contains(cmd, "wg-quick save") {
		t.Error("Command should contain 'wg-quick save'")
	}
	if !strings.Contains(cmd, "systemctl restart") {
		t.Error("Command should contain 'systemctl restart'")
	}
}

func TestWireGuardConfig_MeshNetworking(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.WireGuardConfig
		wantMesh bool
	}{
		{
			name: "Mesh enabled",
			cfg: &config.WireGuardConfig{
				MeshNetworking: true,
			},
			wantMesh: true,
		},
		{
			name: "Mesh disabled",
			cfg: &config.WireGuardConfig{
				MeshNetworking: false,
			},
			wantMesh: false,
		},
		{
			name:     "Default (mesh disabled)",
			cfg:      &config.WireGuardConfig{},
			wantMesh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg.MeshNetworking != tt.wantMesh {
				t.Errorf("Expected MeshNetworking=%v, got %v", tt.wantMesh, tt.cfg.MeshNetworking)
			}
		})
	}
}

func TestWireGuardConfig_AllowedIPs(t *testing.T) {
	cfg := &config.WireGuardConfig{
		AllowedIPs: []string{
			"10.8.0.0/24",
			"10.10.0.0/16",
			"10.11.0.0/16",
		},
	}

	if len(cfg.AllowedIPs) != 3 {
		t.Errorf("Expected 3 allowed IPs, got %d", len(cfg.AllowedIPs))
	}

	expectedIPs := map[string]bool{
		"10.8.0.0/24":  true,
		"10.10.0.0/16": true,
		"10.11.0.0/16": true,
	}

	for _, ip := range cfg.AllowedIPs {
		if !expectedIPs[ip] {
			t.Errorf("Unexpected IP in AllowedIPs: %s", ip)
		}
	}
}

func TestWireGuardConfig_MTU(t *testing.T) {
	tests := []struct {
		name    string
		mtu     int
		wantMTU int
	}{
		{
			name:    "Default MTU",
			mtu:     0,
			wantMTU: 1420,
		},
		{
			name:    "Custom MTU",
			mtu:     1280,
			wantMTU: 1280,
		},
		{
			name:    "Maximum MTU",
			mtu:     1500,
			wantMTU: 1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.WireGuardConfig{
				MTU: tt.mtu,
			}

			// Apply default if 0
			if cfg.MTU == 0 {
				cfg.MTU = 1420
			}

			if cfg.MTU != tt.wantMTU {
				t.Errorf("Expected MTU %d, got %d", tt.wantMTU, cfg.MTU)
			}
		})
	}
}

func TestWireGuardConfig_PersistentKeepalive(t *testing.T) {
	tests := []struct {
		name          string
		keepalive     int
		wantKeepalive int
	}{
		{
			name:          "Default keepalive",
			keepalive:     0,
			wantKeepalive: 25,
		},
		{
			name:          "Custom keepalive",
			keepalive:     15,
			wantKeepalive: 15,
		},
		{
			name:          "Disabled keepalive",
			keepalive:     0,
			wantKeepalive: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.WireGuardConfig{
				PersistentKeepalive: tt.keepalive,
			}

			// Apply default if 0
			if cfg.PersistentKeepalive == 0 {
				cfg.PersistentKeepalive = 25
			}

			if cfg.PersistentKeepalive != tt.wantKeepalive {
				t.Errorf("Expected keepalive %d, got %d", tt.wantKeepalive, cfg.PersistentKeepalive)
			}
		})
	}
}

func TestWireGuardPeer(t *testing.T) {
	peer := config.WireGuardPeer{
		Name:       "peer1",
		PublicKey:  "pubkey123",
		AllowedIPs: []string{"10.8.0.2/32"},
		Endpoint:   "1.2.3.4:51820",
	}

	if peer.Name != "peer1" {
		t.Errorf("Expected name 'peer1', got '%s'", peer.Name)
	}
	if peer.PublicKey != "pubkey123" {
		t.Errorf("Expected public key 'pubkey123', got '%s'", peer.PublicKey)
	}
	if len(peer.AllowedIPs) != 1 {
		t.Errorf("Expected 1 allowed IP, got %d", len(peer.AllowedIPs))
	}
	if peer.Endpoint != "1.2.3.4:51820" {
		t.Errorf("Expected endpoint '1.2.3.4:51820', got '%s'", peer.Endpoint)
	}
}

func TestWireGuardConfig_DNS(t *testing.T) {
	cfg := &config.WireGuardConfig{
		DNS: []string{
			"1.1.1.1",
			"8.8.8.8",
		},
	}

	if len(cfg.DNS) != 2 {
		t.Errorf("Expected 2 DNS servers, got %d", len(cfg.DNS))
	}
	if cfg.DNS[0] != "1.1.1.1" {
		t.Errorf("Expected DNS '1.1.1.1', got '%s'", cfg.DNS[0])
	}
	if cfg.DNS[1] != "8.8.8.8" {
		t.Errorf("Expected DNS '8.8.8.8', got '%s'", cfg.DNS[1])
	}
}

func TestWireGuardConfig_AutoConfig(t *testing.T) {
	tests := []struct {
		name           string
		autoConfig     bool
		wantAutoConfig bool
	}{
		{
			name:           "AutoConfig enabled",
			autoConfig:     true,
			wantAutoConfig: true,
		},
		{
			name:           "AutoConfig disabled",
			autoConfig:     false,
			wantAutoConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.WireGuardConfig{
				AutoConfig: tt.autoConfig,
			}

			if cfg.AutoConfig != tt.wantAutoConfig {
				t.Errorf("Expected AutoConfig=%v, got %v", tt.wantAutoConfig, cfg.AutoConfig)
			}
		})
	}
}
