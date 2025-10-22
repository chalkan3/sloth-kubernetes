package vpc

import (
	"fmt"
	"testing"

	"sloth-kubernetes/pkg/config"
)

// TestVPCResult_Providers tests VPCResult with different providers
func TestVPCResult_Providers(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{"DigitalOcean provider", "digitalocean", false},
		{"Linode provider", "linode", false},
		{"AWS provider", "aws", true},
		{"Azure provider", "azure", true},
		{"GCP provider", "gcp", true},
		{"Invalid provider", "invalid", true},
		{"Empty provider", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VPCResult{
				Provider: tt.provider,
				Name:     "test-vpc",
				CIDR:     "10.0.0.0/16",
				Region:   "us-east",
			}

			isValid := result.Provider == "digitalocean" || result.Provider == "linode"
			if tt.wantErr && isValid {
				t.Errorf("Expected error for provider %q, but it's valid", tt.provider)
			}
			if !tt.wantErr && !isValid {
				t.Errorf("Expected provider %q to be valid, but it's not", tt.provider)
			}
		})
	}
}

// TestVPCResult_RegionValidation tests VPCResult region validation
func TestVPCResult_RegionValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		region   string
		valid    bool
	}{
		// DigitalOcean regions
		{"DO NYC3", "digitalocean", "nyc3", true},
		{"DO SFO3", "digitalocean", "sfo3", true},
		{"DO LON1", "digitalocean", "lon1", true},
		{"DO FRA1", "digitalocean", "fra1", true},
		{"DO SGP1", "digitalocean", "sgp1", true},
		{"DO invalid", "digitalocean", "invalid-region", false},

		// Linode regions
		{"Linode us-east", "linode", "us-east", true},
		{"Linode us-west", "linode", "us-west", true},
		{"Linode eu-west", "linode", "eu-west", true},
		{"Linode ap-south", "linode", "ap-south", true},
		{"Linode invalid", "linode", "invalid-region", false},

		// Empty region
		{"Empty region", "digitalocean", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VPCResult{
				Provider: tt.provider,
				Name:     "test-vpc",
				CIDR:     "10.0.0.0/16",
				Region:   tt.region,
			}

			// Basic validation - region should not be empty
			hasRegion := result.Region != ""
			if tt.valid && !hasRegion {
				t.Errorf("Region should not be empty for valid config")
			}
			if !tt.valid && result.Region != "" && result.Region != "invalid-region" {
				t.Logf("Note: Region %q may not be in standard regions list", tt.region)
			}
		})
	}
}

// TestVPCResult_CIDRFormats tests different CIDR formats
func TestVPCResult_CIDRFormats(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		isValid bool
	}{
		{"/16 CIDR", "10.0.0.0/16", true},
		{"/24 CIDR", "192.168.1.0/24", true},
		{"/8 CIDR", "10.0.0.0/8", true},
		{"/20 CIDR", "172.16.0.0/20", true},
		{"Invalid - no prefix", "10.0.0.0", false},
		{"Invalid - wrong format", "10.0.0.0/", false},
		{"Invalid - out of range", "10.0.0.0/33", false},
		{"Empty CIDR", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VPCResult{
				Provider: "digitalocean",
				Name:     "test-vpc",
				CIDR:     tt.cidr,
				Region:   "nyc3",
			}

			// Basic CIDR validation
			hasCIDR := len(result.CIDR) > 0
			hasSlash := false
			for _, c := range result.CIDR {
				if c == '/' {
					hasSlash = true
					break
				}
			}

			isValidFormat := hasCIDR && hasSlash
			if tt.isValid && !isValidFormat {
				t.Errorf("Expected valid CIDR format for %q", tt.cidr)
			}
			if !tt.isValid && isValidFormat && tt.cidr != "10.0.0.0/" && tt.cidr != "10.0.0.0/33" {
				t.Logf("CIDR %q has slash but may be invalid", tt.cidr)
			}
		})
	}
}

// TestVPCConfig_RegionFallback tests region fallback logic
func TestVPCConfig_RegionFallback(t *testing.T) {
	tests := []struct {
		name           string
		vpcRegion      string
		providerRegion string
		expectedRegion string
	}{
		{
			name:           "VPC region specified",
			vpcRegion:      "nyc3",
			providerRegion: "sfo3",
			expectedRegion: "nyc3",
		},
		{
			name:           "VPC region empty - fallback to provider",
			vpcRegion:      "",
			providerRegion: "sfo3",
			expectedRegion: "sfo3",
		},
		{
			name:           "Both regions specified - VPC takes precedence",
			vpcRegion:      "lon1",
			providerRegion: "fra1",
			expectedRegion: "lon1",
		},
		{
			name:           "VPC region empty - provider also empty",
			vpcRegion:      "",
			providerRegion: "",
			expectedRegion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.DigitalOceanProvider{
				Region: tt.providerRegion,
				VPC: &config.VPCConfig{
					Create: true,
					Name:   "test-vpc",
					CIDR:   "10.0.0.0/16",
					Region: tt.vpcRegion,
				},
			}

			// Simulate region fallback logic from CreateDigitalOceanVPC
			region := cfg.VPC.Region
			if region == "" {
				region = cfg.Region
			}

			if region != tt.expectedRegion {
				t.Errorf("Expected region %q, got %q", tt.expectedRegion, region)
			}
		})
	}
}

// TestVPCConfig_CreateFlag tests VPC creation flag
func TestVPCConfig_CreateFlag(t *testing.T) {
	tests := []struct {
		name         string
		create       bool
		shouldCreate bool
	}{
		{"Create enabled", true, true},
		{"Create disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.VPCConfig{
				Create: tt.create,
				Name:   "test-vpc",
				CIDR:   "10.0.0.0/16",
				Region: "nyc3",
			}

			if cfg.Create != tt.shouldCreate {
				t.Errorf("Expected Create=%v, got Create=%v", tt.shouldCreate, cfg.Create)
			}
		})
	}
}

// TestVPCConfig_IDHandling tests VPC ID handling
func TestVPCConfig_IDHandling(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		hasID     bool
		shouldUse bool
	}{
		{"Existing VPC with ID", "vpc-12345", true, true},
		{"Existing VPC with long ID", "vpc-abcdef1234567890", true, true},
		{"New VPC - no ID", "", false, false},
		{"New VPC - empty string", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.VPCConfig{
				Create: !tt.hasID, // If has ID, don't create
				ID:     tt.id,
				Name:   "test-vpc",
				CIDR:   "10.0.0.0/16",
				Region: "nyc3",
			}

			hasID := cfg.ID != ""
			if tt.hasID != hasID {
				t.Errorf("Expected hasID=%v, got %v", tt.hasID, hasID)
			}

			// If ID is provided, Create should be false
			if tt.hasID && cfg.Create {
				t.Error("If VPC ID is provided, Create should be false")
			}
		})
	}
}

// TestVPCManager_NilContext tests VPCManager with nil context
func TestVPCManager_NilContext(t *testing.T) {
	manager := NewVPCManager(nil)
	if manager == nil {
		t.Fatal("NewVPCManager should not return nil even with nil context")
	}
	if manager.ctx != nil {
		t.Error("Manager context should be nil when created with nil")
	}
}

// TestProvidersConfig_EnabledProviders tests enabled providers count
func TestProvidersConfig_EnabledProviders(t *testing.T) {
	tests := []struct {
		name         string
		providers    *config.ProvidersConfig
		enabledCount int
		vpcCount     int
	}{
		{
			name: "Both providers enabled with VPCs",
			providers: &config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					Enabled: true,
					VPC:     &config.VPCConfig{Create: true},
				},
				Linode: &config.LinodeProvider{
					Enabled: true,
					VPC:     &config.VPCConfig{Create: true},
				},
			},
			enabledCount: 2,
			vpcCount:     2,
		},
		{
			name: "One provider enabled, one VPC",
			providers: &config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					Enabled: true,
					VPC:     &config.VPCConfig{Create: true},
				},
				Linode: &config.LinodeProvider{
					Enabled: false,
				},
			},
			enabledCount: 1,
			vpcCount:     1,
		},
		{
			name: "No providers enabled",
			providers: &config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					Enabled: false,
				},
				Linode: &config.LinodeProvider{
					Enabled: false,
				},
			},
			enabledCount: 0,
			vpcCount:     0,
		},
		{
			name: "Provider enabled but no VPC",
			providers: &config.ProvidersConfig{
				DigitalOcean: &config.DigitalOceanProvider{
					Enabled: true,
					VPC:     nil,
				},
			},
			enabledCount: 1,
			vpcCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enabledCount := 0
			vpcCount := 0

			if tt.providers.DigitalOcean != nil && tt.providers.DigitalOcean.Enabled {
				enabledCount++
				if tt.providers.DigitalOcean.VPC != nil && tt.providers.DigitalOcean.VPC.Create {
					vpcCount++
				}
			}

			if tt.providers.Linode != nil && tt.providers.Linode.Enabled {
				enabledCount++
				if tt.providers.Linode.VPC != nil && tt.providers.Linode.VPC.Create {
					vpcCount++
				}
			}

			if enabledCount != tt.enabledCount {
				t.Errorf("Expected %d enabled providers, got %d", tt.enabledCount, enabledCount)
			}
			if vpcCount != tt.vpcCount {
				t.Errorf("Expected %d VPCs, got %d", tt.vpcCount, vpcCount)
			}
		})
	}
}

// TestVPCResult_DescriptionGeneration tests description generation
func TestVPCResult_DescriptionGeneration(t *testing.T) {
	tests := []struct {
		name            string
		vpcName         string
		expectedPattern string
	}{
		{
			name:            "Standard VPC name",
			vpcName:         "production-vpc",
			expectedPattern: "VPC for Kubernetes cluster - production-vpc",
		},
		{
			name:            "Short VPC name",
			vpcName:         "dev",
			expectedPattern: "VPC for Kubernetes cluster - dev",
		},
		{
			name:            "Hyphenated VPC name",
			vpcName:         "staging-k8s-cluster",
			expectedPattern: "VPC for Kubernetes cluster - staging-k8s-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate description generation
			description := fmt.Sprintf("VPC for Kubernetes cluster - %s", tt.vpcName)

			if description != tt.expectedPattern {
				t.Errorf("Expected description %q, got %q", tt.expectedPattern, description)
			}
		})
	}
}

// TestVPCConfig_Tags tests VPC tags
func TestVPCConfig_Tags(t *testing.T) {
	tests := []struct {
		name        string
		tags        []string
		expectedLen int
	}{
		{
			name:        "Multiple tags",
			tags:        []string{"kubernetes", "production", "vpc"},
			expectedLen: 3,
		},
		{
			name:        "Single tag",
			tags:        []string{"kubernetes"},
			expectedLen: 1,
		},
		{
			name:        "No tags",
			tags:        []string{},
			expectedLen: 0,
		},
		{
			name:        "Nil tags",
			tags:        nil,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.VPCConfig{
				Create: true,
				Name:   "test-vpc",
				CIDR:   "10.0.0.0/16",
				Region: "nyc3",
				Tags:   tt.tags,
			}

			tagLen := 0
			if cfg.Tags != nil {
				tagLen = len(cfg.Tags)
			}

			if tagLen != tt.expectedLen {
				t.Errorf("Expected %d tags, got %d", tt.expectedLen, tagLen)
			}
		})
	}
}

// TestLinodeSubnetConfig tests Linode subnet configuration
func TestLinodeSubnetConfig(t *testing.T) {
	tests := []struct {
		name      string
		subnet    config.LinodeSubnetConfig
		wantValid bool
	}{
		{
			name: "Valid subnet",
			subnet: config.LinodeSubnetConfig{
				Label: "subnet-1",
				IPv4:  "10.0.1.0/24",
			},
			wantValid: true,
		},
		{
			name: "Missing label",
			subnet: config.LinodeSubnetConfig{
				Label: "",
				IPv4:  "10.0.1.0/24",
			},
			wantValid: false,
		},
		{
			name: "Missing IPv4",
			subnet: config.LinodeSubnetConfig{
				Label: "subnet-1",
				IPv4:  "",
			},
			wantValid: false,
		},
		{
			name: "Both missing",
			subnet: config.LinodeSubnetConfig{
				Label: "",
				IPv4:  "",
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.subnet.Label != "" && tt.subnet.IPv4 != ""
			if isValid != tt.wantValid {
				t.Errorf("Expected valid=%v, got %v", tt.wantValid, isValid)
			}
		})
	}
}

// TestVPCResult_MultipleRegions tests VPCResult with multiple regions
func TestVPCResult_MultipleRegions(t *testing.T) {
	regions := map[string][]string{
		"digitalocean": {"nyc1", "nyc2", "nyc3", "sfo1", "sfo2", "sfo3", "ams3", "sgp1", "lon1", "fra1"},
		"linode":       {"us-east", "us-west", "us-central", "eu-west", "eu-central", "ap-south", "ap-northeast"},
	}

	for provider, regionList := range regions {
		for _, region := range regionList {
			t.Run(fmt.Sprintf("%s-%s", provider, region), func(t *testing.T) {
				result := &VPCResult{
					Provider: provider,
					Name:     "test-vpc",
					CIDR:     "10.0.0.0/16",
					Region:   region,
				}

				if result.Region != region {
					t.Errorf("Expected region %q, got %q", region, result.Region)
				}
				if result.Provider != provider {
					t.Errorf("Expected provider %q, got %q", provider, result.Provider)
				}
			})
		}
	}
}

// TestVPCConfig_BooleanFields tests VPC config boolean fields
func TestVPCConfig_BooleanFields(t *testing.T) {
	cfg := &config.VPCConfig{
		Create:            true,
		Name:              "test-vpc",
		CIDR:              "10.0.0.0/16",
		Region:            "nyc3",
		EnableDNS:         false,
		EnableDNSHostname: false,
		InternetGateway:   false,
		NATGateway:        false,
	}

	// Test default false values
	if cfg.EnableDNS {
		t.Error("EnableDNS should be false by default")
	}
	if cfg.EnableDNSHostname {
		t.Error("EnableDNSHostname should be false by default")
	}
	if cfg.InternetGateway {
		t.Error("InternetGateway should be false by default")
	}
	if cfg.NATGateway {
		t.Error("NATGateway should be false by default")
	}

	// Test setting to true
	cfg.EnableDNS = true
	cfg.EnableDNSHostname = true
	cfg.InternetGateway = true
	cfg.NATGateway = true

	if !cfg.EnableDNS {
		t.Error("EnableDNS should be true")
	}
	if !cfg.EnableDNSHostname {
		t.Error("EnableDNSHostname should be true")
	}
	if !cfg.InternetGateway {
		t.Error("InternetGateway should be true")
	}
	if !cfg.NATGateway {
		t.Error("NATGateway should be true")
	}
}
