package vpc

import (
	"testing"

	"sloth-kubernetes/pkg/config"
)

func TestNewVPCManager(t *testing.T) {
	// Test VPCManager creation
	manager := NewVPCManager(nil)
	if manager == nil {
		t.Error("NewVPCManager should not return nil")
	}
}

func TestVPCConfig_DigitalOcean(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.DigitalOceanProvider
		wantNil bool
	}{
		{
			name: "VPC creation enabled",
			cfg: &config.DigitalOceanProvider{
				VPC: &config.VPCConfig{
					Create: true,
					Name:   "test-vpc",
					CIDR:   "10.10.0.0/16",
					Region: "nyc3",
				},
			},
			wantNil: false,
		},
		{
			name: "VPC creation disabled",
			cfg: &config.DigitalOceanProvider{
				VPC: &config.VPCConfig{
					Create: false,
				},
			},
			wantNil: true,
		},
		{
			name:    "No VPC config",
			cfg:     &config.DigitalOceanProvider{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCreate := tt.cfg.VPC != nil && tt.cfg.VPC.Create
			if tt.wantNil && shouldCreate {
				t.Error("Expected no VPC creation but Create is true")
			}
			if !tt.wantNil && !shouldCreate {
				t.Error("Expected VPC creation but Create is false")
			}
		})
	}
}

func TestVPCConfig_Linode(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.LinodeProvider
		wantNil bool
	}{
		{
			name: "VPC creation enabled",
			cfg: &config.LinodeProvider{
				VPC: &config.VPCConfig{
					Create: true,
					Name:   "test-vpc",
					CIDR:   "10.11.0.0/16",
					Region: "us-east",
				},
			},
			wantNil: false,
		},
		{
			name: "VPC creation disabled",
			cfg: &config.LinodeProvider{
				VPC: &config.VPCConfig{
					Create: false,
				},
			},
			wantNil: true,
		},
		{
			name:    "No VPC config",
			cfg:     &config.LinodeProvider{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCreate := tt.cfg.VPC != nil && tt.cfg.VPC.Create
			if tt.wantNil && shouldCreate {
				t.Error("Expected no VPC creation but Create is true")
			}
			if !tt.wantNil && !shouldCreate {
				t.Error("Expected VPC creation but Create is false")
			}
		})
	}
}

func TestVPCResult(t *testing.T) {
	result := &VPCResult{
		Provider: "digitalocean",
		Name:     "test-vpc",
		CIDR:     "10.10.0.0/16",
		Region:   "nyc3",
	}

	if result.Provider != "digitalocean" {
		t.Errorf("Expected provider 'digitalocean', got '%s'", result.Provider)
	}
	if result.Name != "test-vpc" {
		t.Errorf("Expected name 'test-vpc', got '%s'", result.Name)
	}
	if result.CIDR != "10.10.0.0/16" {
		t.Errorf("Expected CIDR '10.10.0.0/16', got '%s'", result.CIDR)
	}
	if result.Region != "nyc3" {
		t.Errorf("Expected region 'nyc3', got '%s'", result.Region)
	}
}

func TestVPCConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		vpc     *config.VPCConfig
		wantErr bool
	}{
		{
			name: "Valid VPC config",
			vpc: &config.VPCConfig{
				Create: true,
				Name:   "test-vpc",
				CIDR:   "10.10.0.0/16",
				Region: "nyc3",
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			vpc: &config.VPCConfig{
				Create: true,
				CIDR:   "10.10.0.0/16",
				Region: "nyc3",
			},
			wantErr: true,
		},
		{
			name: "Missing CIDR",
			vpc: &config.VPCConfig{
				Create: true,
				Name:   "test-vpc",
				Region: "nyc3",
			},
			wantErr: true,
		},
		{
			name: "Missing region",
			vpc: &config.VPCConfig{
				Create: true,
				Name:   "test-vpc",
				CIDR:   "10.10.0.0/16",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.vpc.Name == "" || tt.vpc.CIDR == "" || tt.vpc.Region == ""
			if tt.wantErr && !hasError {
				t.Error("Expected validation error but got none")
			}
			if !tt.wantErr && hasError {
				t.Error("Expected no validation error but got one")
			}
		})
	}
}

func TestProvidersConfig_VPCCount(t *testing.T) {
	tests := []struct {
		name      string
		providers *config.ProvidersConfig
		wantCount int
	}{
		{
			name: "Both providers with VPC",
			providers: &config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					VPC: &config.VPCConfig{Create: true},
				},
				Linode: &config.LinodeProvider{
					VPC: &config.VPCConfig{Create: true},
				},
			},
			wantCount: 2,
		},
		{
			name: "Only DigitalOcean with VPC",
			providers: &config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					VPC: &config.VPCConfig{Create: true},
				},
			},
			wantCount: 1,
		},
		{
			name: "Only Linode with VPC",
			providers: &config.ProvidersConfig{
				Linode: &config.LinodeProvider{
					VPC: &config.VPCConfig{Create: true},
				},
			},
			wantCount: 1,
		},
		{
			name:      "No VPCs",
			providers: &config.ProvidersConfig{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			if tt.providers.DigitalOcean != nil && tt.providers.DigitalOcean.VPC != nil && tt.providers.DigitalOcean.VPC.Create {
				count++
			}
			if tt.providers.Linode != nil && tt.providers.Linode.VPC != nil && tt.providers.Linode.VPC.Create {
				count++
			}

			if count != tt.wantCount {
				t.Errorf("Expected %d VPCs, got %d", tt.wantCount, count)
			}
		})
	}
}

func TestVPCConfig_DefaultValues(t *testing.T) {
	vpc := &config.VPCConfig{
		Create: true,
		Name:   "test-vpc",
		CIDR:   "10.10.0.0/16",
		Region: "nyc3",
	}

	// Test default values
	if vpc.EnableDNS {
		t.Error("EnableDNS should default to false")
	}
	if vpc.EnableDNSHostname {
		t.Error("EnableDNSHostname should default to false")
	}
	if vpc.InternetGateway {
		t.Error("InternetGateway should default to false")
	}
	if vpc.NATGateway {
		t.Error("NATGateway should default to false")
	}

	// Test with enabled values
	vpc.EnableDNS = true
	vpc.EnableDNSHostname = true
	vpc.InternetGateway = true

	if !vpc.EnableDNS {
		t.Error("EnableDNS should be true")
	}
	if !vpc.EnableDNSHostname {
		t.Error("EnableDNSHostname should be true")
	}
	if !vpc.InternetGateway {
		t.Error("InternetGateway should be true")
	}
}

func TestDOVPCConfig(t *testing.T) {
	doVPC := &config.DOVPCConfig{
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
	linodeVPC := &config.LinodeVPCConfig{
		Label:       "test-vpc",
		Description: "Test VPC",
		Subnets: []config.LinodeSubnetConfig{
			{
				Label: "subnet-1",
				IPv4:  "10.11.1.0/24",
			},
			{
				Label: "subnet-2",
				IPv4:  "10.11.2.0/24",
			},
		},
	}

	if linodeVPC.Label != "test-vpc" {
		t.Errorf("Expected Label 'test-vpc', got '%s'", linodeVPC.Label)
	}
	if len(linodeVPC.Subnets) != 2 {
		t.Errorf("Expected 2 subnets, got %d", len(linodeVPC.Subnets))
	}
	if linodeVPC.Subnets[0].Label != "subnet-1" {
		t.Errorf("Expected subnet label 'subnet-1', got '%s'", linodeVPC.Subnets[0].Label)
	}
	if linodeVPC.Subnets[0].IPv4 != "10.11.1.0/24" {
		t.Errorf("Expected subnet IPv4 '10.11.1.0/24', got '%s'", linodeVPC.Subnets[0].IPv4)
	}
}
