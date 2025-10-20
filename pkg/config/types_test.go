package config

import (
	"testing"
)

func TestVPCConfig(t *testing.T) {
	vpc := &VPCConfig{
		Create:            true,
		Name:              "test-vpc",
		CIDR:              "10.10.0.0/16",
		Region:            "nyc3",
		EnableDNS:         true,
		EnableDNSHostname: true,
		InternetGateway:   true,
	}

	if vpc.Create != true {
		t.Error("Create should be true")
	}
	if vpc.Name != "test-vpc" {
		t.Errorf("Expected name 'test-vpc', got '%s'", vpc.Name)
	}
	if vpc.CIDR != "10.10.0.0/16" {
		t.Errorf("Expected CIDR '10.10.0.0/16', got '%s'", vpc.CIDR)
	}
	if !vpc.EnableDNS {
		t.Error("EnableDNS should be true")
	}
}

func TestWireGuardConfig(t *testing.T) {
	wg := &WireGuardConfig{
		Create:     true,
		Provider:   "digitalocean",
		Region:     "nyc3",
		Size:       "s-1vcpu-1gb",
		Port:       51820,
		SubnetCIDR: "10.8.0.0/24",
		Enabled:    true,
	}

	if wg.Create != true {
		t.Error("Create should be true")
	}
	if wg.Provider != "digitalocean" {
		t.Errorf("Expected provider 'digitalocean', got '%s'", wg.Provider)
	}
	if wg.Port != 51820 {
		t.Errorf("Expected port 51820, got %d", wg.Port)
	}
}

func TestDOVPCConfig(t *testing.T) {
	doVPC := &DOVPCConfig{
		IPRange:     "10.10.0.0/16",
		Description: "Test VPC",
	}

	if doVPC.IPRange != "10.10.0.0/16" {
		t.Errorf("Expected IPRange '10.10.0.0/16', got '%s'", doVPC.IPRange)
	}
	if doVPC.Description != "Test VPC" {
		t.Errorf("Expected Description 'Test VPC', got '%s'", doVPC.Description)
	}
}

func TestLinodeVPCConfig(t *testing.T) {
	linodeVPC := &LinodeVPCConfig{
		Label:       "test-vpc",
		Description: "Test VPC",
		Subnets: []LinodeSubnetConfig{
			{
				Label: "subnet-1",
				IPv4:  "10.11.1.0/24",
			},
		},
	}

	if linodeVPC.Label != "test-vpc" {
		t.Errorf("Expected Label 'test-vpc', got '%s'", linodeVPC.Label)
	}
	if len(linodeVPC.Subnets) != 1 {
		t.Errorf("Expected 1 subnet, got %d", len(linodeVPC.Subnets))
	}
}

func TestLinodeSubnetConfig(t *testing.T) {
	subnet := LinodeSubnetConfig{
		Label: "test-subnet",
		IPv4:  "10.11.1.0/24",
	}

	if subnet.Label != "test-subnet" {
		t.Errorf("Expected Label 'test-subnet', got '%s'", subnet.Label)
	}
	if subnet.IPv4 != "10.11.1.0/24" {
		t.Errorf("Expected IPv4 '10.11.1.0/24', got '%s'", subnet.IPv4)
	}
}

func TestProvidersConfig(t *testing.T) {
	providers := ProvidersConfig{
		DigitalOcean: &DigitalOceanProvider{
			Enabled: true,
			Token:   "test-token",
			Region:  "nyc3",
			VPC: &VPCConfig{
				Create: true,
				Name:   "do-vpc",
				CIDR:   "10.10.0.0/16",
			},
		},
		Linode: &LinodeProvider{
			Enabled: true,
			Token:   "test-token",
			Region:  "us-east",
			VPC: &VPCConfig{
				Create: true,
				Name:   "linode-vpc",
				CIDR:   "10.11.0.0/16",
			},
		},
	}

	if !providers.DigitalOcean.Enabled {
		t.Error("DigitalOcean should be enabled")
	}
	if !providers.Linode.Enabled {
		t.Error("Linode should be enabled")
	}
	if providers.DigitalOcean.VPC == nil {
		t.Error("DigitalOcean VPC should not be nil")
	}
	if providers.Linode.VPC == nil {
		t.Error("Linode VPC should not be nil")
	}
}

func TestNetworkConfig(t *testing.T) {
	network := NetworkConfig{
		Mode: "wireguard",
		CIDR: "10.8.0.0/16",
		WireGuard: &WireGuardConfig{
			Create:   true,
			Provider: "digitalocean",
		},
	}

	if network.Mode != "wireguard" {
		t.Errorf("Expected mode 'wireguard', got '%s'", network.Mode)
	}
	if network.WireGuard == nil {
		t.Error("WireGuard config should not be nil")
	}
	if !network.WireGuard.Create {
		t.Error("WireGuard Create should be true")
	}
}

func TestWireGuardPeer(t *testing.T) {
	peer := WireGuardPeer{
		Name:       "peer1",
		PublicKey:  "test-key",
		AllowedIPs: []string{"10.8.0.2/32"},
		Endpoint:   "1.2.3.4:51820",
	}

	if peer.Name != "peer1" {
		t.Errorf("Expected name 'peer1', got '%s'", peer.Name)
	}
	if peer.PublicKey != "test-key" {
		t.Errorf("Expected public key 'test-key', got '%s'", peer.PublicKey)
	}
	if len(peer.AllowedIPs) == 0 {
		t.Error("AllowedIPs should not be empty")
	}
}

func TestClusterConfig(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test-cluster",
			Environment: "production",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				VPC: &VPCConfig{
					Create: true,
				},
			},
		},
		Network: NetworkConfig{
			Mode: "wireguard",
			WireGuard: &WireGuardConfig{
				Create: true,
			},
		},
	}

	if cfg.Metadata.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", cfg.Metadata.Name)
	}
	if cfg.Providers.DigitalOcean == nil {
		t.Error("DigitalOcean provider should not be nil")
	}
	if cfg.Network.WireGuard == nil {
		t.Error("WireGuard config should not be nil")
	}
}

func TestVPCConfigWithProviderSpecific(t *testing.T) {
	// Test with DigitalOcean specific config
	doVPC := &VPCConfig{
		Create: true,
		Name:   "do-vpc",
		CIDR:   "10.10.0.0/16",
		DigitalOcean: &DOVPCConfig{
			IPRange:     "10.10.0.0/16",
			Description: "DO VPC",
		},
	}

	if doVPC.DigitalOcean == nil {
		t.Error("DigitalOcean config should not be nil")
	}
	if doVPC.DigitalOcean.IPRange != "10.10.0.0/16" {
		t.Errorf("Expected IPRange '10.10.0.0/16', got '%s'", doVPC.DigitalOcean.IPRange)
	}

	// Test with Linode specific config
	linodeVPC := &VPCConfig{
		Create: true,
		Name:   "linode-vpc",
		CIDR:   "10.11.0.0/16",
		Linode: &LinodeVPCConfig{
			Label: "linode-vpc",
			Subnets: []LinodeSubnetConfig{
				{Label: "subnet-1", IPv4: "10.11.1.0/24"},
			},
		},
	}

	if linodeVPC.Linode == nil {
		t.Error("Linode config should not be nil")
	}
	if len(linodeVPC.Linode.Subnets) != 1 {
		t.Errorf("Expected 1 subnet, got %d", len(linodeVPC.Linode.Subnets))
	}
}

func TestWireGuardConfigDefaults(t *testing.T) {
	wg := &WireGuardConfig{
		Create:   true,
		Provider: "digitalocean",
		Region:   "nyc3",
	}

	// Port should default to 51820
	if wg.Port == 0 {
		wg.Port = 51820
	}
	if wg.Port != 51820 {
		t.Errorf("Expected default port 51820, got %d", wg.Port)
	}

	// SubnetCIDR should default to 10.8.0.0/24
	if wg.SubnetCIDR == "" {
		wg.SubnetCIDR = "10.8.0.0/24"
	}
	if wg.SubnetCIDR != "10.8.0.0/24" {
		t.Errorf("Expected default subnet '10.8.0.0/24', got '%s'", wg.SubnetCIDR)
	}

	// Image should default to ubuntu-22-04-x64
	if wg.Image == "" {
		wg.Image = "ubuntu-22-04-x64"
	}
	if wg.Image != "ubuntu-22-04-x64" {
		t.Errorf("Expected default image 'ubuntu-22-04-x64', got '%s'", wg.Image)
	}
}

func TestVPCConfigTags(t *testing.T) {
	vpc := &VPCConfig{
		Create: true,
		Name:   "test-vpc",
		CIDR:   "10.10.0.0/16",
		Tags:   []string{"kubernetes", "production"},
	}

	if len(vpc.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(vpc.Tags))
	}
	if vpc.Tags[0] != "kubernetes" {
		t.Errorf("Expected tag 'kubernetes', got '%s'", vpc.Tags[0])
	}
}

func TestWireGuardConfigAllowedIPs(t *testing.T) {
	wg := &WireGuardConfig{
		AllowedIPs: []string{
			"10.8.0.0/24",
			"10.10.0.0/16",
			"10.11.0.0/16",
		},
	}

	if len(wg.AllowedIPs) != 3 {
		t.Errorf("Expected 3 allowed IPs, got %d", len(wg.AllowedIPs))
	}

	expectedIPs := []string{"10.8.0.0/24", "10.10.0.0/16", "10.11.0.0/16"}
	for i, ip := range wg.AllowedIPs {
		if ip != expectedIPs[i] {
			t.Errorf("Expected IP '%s' at index %d, got '%s'", expectedIPs[i], i, ip)
		}
	}
}

func TestDigitalOceanProvider(t *testing.T) {
	provider := &DigitalOceanProvider{
		Enabled:    true,
		Token:      "test-token",
		Region:     "nyc3",
		Tags:       []string{"k8s"},
		Monitoring: true,
		IPv6:       false,
		VPC: &VPCConfig{
			Create: true,
			Name:   "test-vpc",
		},
	}

	if !provider.Enabled {
		t.Error("Provider should be enabled")
	}
	if provider.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", provider.Token)
	}
	if !provider.Monitoring {
		t.Error("Monitoring should be enabled")
	}
	if provider.IPv6 {
		t.Error("IPv6 should be disabled")
	}
}

func TestLinodeProvider(t *testing.T) {
	provider := &LinodeProvider{
		Enabled:   true,
		Token:     "test-token",
		Region:    "us-east",
		PrivateIP: true,
		VPC: &VPCConfig{
			Create: true,
			Name:   "test-vpc",
		},
	}

	if !provider.Enabled {
		t.Error("Provider should be enabled")
	}
	if !provider.PrivateIP {
		t.Error("PrivateIP should be enabled")
	}
}
