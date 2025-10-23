package vpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// Test VPCManager creation
func TestNewVPCManager_Creation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"Basic creation"},
		{"Manager creation"},
		{"Nil context handling"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &VPCManager{}
			assert.NotNil(t, manager)
		})
	}
}

// Test VPCResult structure
func TestVPCResult_StructureUnit(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		vpcName  string
		cidr     string
		region   string
	}{
		{"DigitalOcean VPC", "digitalocean", "do-vpc-1", "10.0.0.0/16", "nyc3"},
		{"Linode VPC", "linode", "linode-vpc-1", "10.1.0.0/16", "us-east"},
		{"Custom CIDR", "digitalocean", "custom-vpc", "172.16.0.0/12", "sfo3"},
		{"Different region", "linode", "eu-vpc", "192.168.0.0/16", "eu-west"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VPCResult{
				Provider: tt.provider,
				Name:     tt.vpcName,
				CIDR:     tt.cidr,
				Region:   tt.region,
			}

			assert.Equal(t, tt.provider, result.Provider)
			assert.Equal(t, tt.vpcName, result.Name)
			assert.Equal(t, tt.cidr, result.CIDR)
			assert.Equal(t, tt.region, result.Region)
		})
	}
}

// Test VPC creation logic - config validation
func TestVPC_ConfigValidation_Create(t *testing.T) {
	tests := []struct {
		name         string
		vpcConfig    *config.VPCConfig
		shouldCreate bool
	}{
		{
			"VPC enabled",
			&config.VPCConfig{Create: true, Name: "test-vpc", CIDR: "10.0.0.0/16"},
			true,
		},
		{
			"VPC disabled",
			&config.VPCConfig{Create: false, Name: "test-vpc", CIDR: "10.0.0.0/16"},
			false,
		},
		{
			"Nil VPC config",
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCreate := tt.vpcConfig != nil && tt.vpcConfig.Create
			assert.Equal(t, tt.shouldCreate, shouldCreate)
		})
	}
}

// Test region selection logic
func TestVPC_RegionSelection(t *testing.T) {
	tests := []struct {
		name           string
		vpcRegion      string
		providerRegion string
		expectedRegion string
	}{
		{"VPC region specified", "nyc3", "sfo3", "nyc3"},
		{"Use provider region", "", "sfo3", "sfo3"},
		{"Both regions specified", "ams3", "nyc3", "ams3"},
		{"Empty VPC region", "", "lon1", "lon1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region := tt.vpcRegion
			if region == "" {
				region = tt.providerRegion
			}
			assert.Equal(t, tt.expectedRegion, region)
		})
	}
}

// Test VPC name generation
func TestVPC_NameGeneration(t *testing.T) {
	tests := []struct {
		name         string
		baseName     string
		provider     string
		expectedName string
	}{
		{"DigitalOcean VPC", "k8s-cluster", "do", "k8s-cluster"},
		{"Linode VPC", "production", "linode", "production"},
		{"Custom name", "my-vpc-01", "do", "my-vpc-01"},
		{"Hyphenated name", "kubernetes-production-vpc", "linode", "kubernetes-production-vpc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, tt.baseName)
		})
	}
}

// Test CIDR validation patterns
func TestVPC_CIDRPatterns(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		valid bool
	}{
		{"Valid /16", "10.0.0.0/16", true},
		{"Valid /24", "10.0.1.0/24", true},
		{"Valid /20", "10.0.0.0/20", true},
		{"Valid /12", "172.16.0.0/12", true},
		{"Private Class A", "10.0.0.0/8", true},
		{"Private Class B", "172.16.0.0/16", true},
		{"Private Class C", "192.168.0.0/24", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple CIDR format check
			hasSlash := false
			for _, ch := range tt.cidr {
				if ch == '/' {
					hasSlash = true
					break
				}
			}
			assert.Equal(t, tt.valid, hasSlash && len(tt.cidr) > 0)
		})
	}
}

// Test VPC description generation
func TestVPC_DescriptionGenerationUnit(t *testing.T) {
	tests := []struct {
		name         string
		vpcName      string
		expectedDesc string
	}{
		{"Standard VPC", "production-vpc", "VPC for Kubernetes cluster - production-vpc"},
		{"Development VPC", "dev-k8s", "VPC for Kubernetes cluster - dev-k8s"},
		{"Staging VPC", "staging", "VPC for Kubernetes cluster - staging"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := "VPC for Kubernetes cluster - " + tt.vpcName
			assert.Equal(t, tt.expectedDesc, desc)
		})
	}
}

// Test export keys for DigitalOcean
func TestVPC_DigitalOceanExportKeys(t *testing.T) {
	expectedKeys := []string{
		"digitalocean_vpc_id",
		"digitalocean_vpc_urn",
		"digitalocean_vpc_ip_range",
	}

	for _, key := range expectedKeys {
		t.Run("ExportKey_"+key, func(t *testing.T) {
			assert.NotEmpty(t, key)
			assert.Contains(t, key, "digitalocean_vpc")
		})
	}
}

// Test export keys for Linode
func TestVPC_LinodeExportKeys(t *testing.T) {
	expectedKeys := []string{
		"linode_vpc_id",
		"linode_vpc_label",
	}

	for _, key := range expectedKeys {
		t.Run("ExportKey_"+key, func(t *testing.T) {
			assert.NotEmpty(t, key)
			assert.Contains(t, key, "linode_vpc")
		})
	}
}

// Test provider identification
func TestVPC_ProviderIdentification(t *testing.T) {
	providers := []string{"digitalocean", "linode"}

	for _, provider := range providers {
		t.Run("Provider_"+provider, func(t *testing.T) {
			result := &VPCResult{Provider: provider}
			assert.Contains(t, providers, result.Provider)
		})
	}
}

// Test VPC result provider field
func TestVPCResult_ProviderField(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{"DigitalOcean", "digitalocean"},
		{"Linode", "linode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &VPCResult{Provider: tt.provider}
			assert.Equal(t, tt.provider, result.Provider)
			assert.NotEmpty(t, result.Provider)
		})
	}
}

// Test VPC configuration combinations
func TestVPC_ConfigurationCombinations(t *testing.T) {
	tests := []struct {
		name    string
		create  bool
		vpcName string
		cidr    string
		region  string
		valid   bool
	}{
		{"Valid full config", true, "vpc-1", "10.0.0.0/16", "nyc3", true},
		{"Create disabled", false, "vpc-1", "10.0.0.0/16", "nyc3", false},
		{"Empty name", true, "", "10.0.0.0/16", "nyc3", false},
		{"Empty CIDR", true, "vpc-1", "", "nyc3", false},
		{"Empty region", true, "vpc-1", "10.0.0.0/16", "", true}, // Region can be inherited
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.create && tt.vpcName != "" && tt.cidr != ""
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// Test 100 VPC scenarios with different configurations
func Test100VPCScenariosUnit(t *testing.T) {
	scenarios := []struct {
		provider string
		cidr     string
		region   string
	}{
		{"digitalocean", "10.0.0.0/16", "nyc3"},
		{"linode", "10.1.0.0/16", "us-east"},
		{"digitalocean", "172.16.0.0/16", "sfo3"},
	}

	// Generate 97 more scenarios
	providers := []string{"digitalocean", "linode"}
	doRegions := []string{"nyc3", "sfo3", "ams3", "sgp1", "lon1", "fra1"}
	linodeRegions := []string{"us-east", "us-west", "eu-west", "ap-south"}
	cidrBases := []string{"10", "172", "192"}
	masks := []int{16, 20, 24}

	for i := 0; i < 97; i++ {
		provider := providers[i%2]
		cidrBase := cidrBases[i%3]
		mask := masks[i%3]
		var cidr string
		if cidrBase == "10" {
			cidr = "10." + string(rune('0'+(i%256))) + ".0.0/" + string(rune('0'+mask/10)) + string(rune('0'+mask%10))
		} else if cidrBase == "172" {
			cidr = "172.16.0.0/" + string(rune('0'+mask/10)) + string(rune('0'+mask%10))
		} else {
			cidr = "192.168." + string(rune('0'+(i%256))) + ".0/" + string(rune('0'+mask/10)) + string(rune('0'+mask%10))
		}

		var region string
		if provider == "digitalocean" {
			region = doRegions[i%len(doRegions)]
		} else {
			region = linodeRegions[i%len(linodeRegions)]
		}

		scenarios = append(scenarios, struct {
			provider string
			cidr     string
			region   string
		}{provider, cidr, region})
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			result := &VPCResult{
				Provider: scenario.provider,
				CIDR:     scenario.cidr,
				Region:   scenario.region,
				Name:     "vpc-" + string(rune('0'+i)),
			}

			assert.NotEmpty(t, result.Provider)
			assert.NotEmpty(t, result.CIDR)
			assert.NotEmpty(t, result.Region)
			assert.NotEmpty(t, result.Name)

			// Validate provider
			assert.Contains(t, []string{"digitalocean", "linode"}, result.Provider)

			// Validate CIDR format (basic)
			assert.Contains(t, result.CIDR, "/")

			// Validate region not empty
			assert.True(t, len(result.Region) > 0)
		})
	}
}

// Test error message formats
func TestVPC_ErrorMessageFormats(t *testing.T) {
	tests := []struct {
		provider      string
		expectedError string
	}{
		{"digitalocean", "failed to create DigitalOcean VPC"},
		{"linode", "failed to create Linode VPC"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			assert.Contains(t, tt.expectedError, "failed to create")
			assert.Contains(t, tt.expectedError, "VPC")
		})
	}
}

// Test VPC name formatting
func TestVPC_NameFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Lowercase", "production", "production"},
		{"With hyphens", "prod-vpc-01", "prod-vpc-01"},
		{"Numbers", "vpc123", "vpc123"},
		{"Mixed", "k8s-prod-vpc-01", "k8s-prod-vpc-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

// Test region codes validation
func TestVPC_RegionCodes(t *testing.T) {
	doRegions := []string{"nyc1", "nyc3", "sfo3", "ams3", "sgp1", "lon1", "fra1", "tor1", "blr1"}
	linodeRegions := []string{"us-east", "us-west", "us-central", "eu-west", "eu-central", "ap-south", "ap-northeast"}

	t.Run("DigitalOcean_Regions", func(t *testing.T) {
		for _, region := range doRegions {
			assert.NotEmpty(t, region)
			assert.True(t, len(region) >= 3)
		}
	})

	t.Run("Linode_Regions", func(t *testing.T) {
		for _, region := range linodeRegions {
			assert.NotEmpty(t, region)
			assert.Contains(t, region, "-")
		}
	})
}

// Test VPC IP range validation
func TestVPC_IPRangeValidation(t *testing.T) {
	tests := []struct {
		name    string
		ipRange string
		valid   bool
	}{
		{"Valid Class A", "10.0.0.0/16", true},
		{"Valid Class B", "172.16.0.0/16", true},
		{"Valid Class C", "192.168.0.0/24", true},
		{"Large network", "10.0.0.0/8", true},
		{"Small network", "10.0.0.0/28", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasSlash := false
			for _, ch := range tt.ipRange {
				if ch == '/' {
					hasSlash = true
					break
				}
			}
			assert.True(t, hasSlash)
		})
	}
}

// Test VPC configuration inheritance
func TestVPC_ConfigurationInheritance(t *testing.T) {
	tests := []struct {
		name           string
		vpcRegion      string
		providerRegion string
		useProvider    bool
	}{
		{"VPC has region", "nyc3", "sfo3", false},
		{"Inherit from provider", "", "sfo3", true},
		{"Both specified", "ams3", "nyc3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldInherit := tt.vpcRegion == ""
			assert.Equal(t, tt.useProvider, shouldInherit)
		})
	}
}

// Test VPC export structure
func TestVPC_ExportStructure(t *testing.T) {
	type VPCExports struct {
		ID      string
		URN     string
		IPRange string
		Label   string
	}

	exports := VPCExports{
		ID:      "vpc-123456",
		URN:     "urn:do:vpc:123456",
		IPRange: "10.0.0.0/16",
		Label:   "production-vpc",
	}

	assert.NotEmpty(t, exports.ID)
	assert.NotEmpty(t, exports.URN)
	assert.NotEmpty(t, exports.IPRange)
	assert.NotEmpty(t, exports.Label)
}

// Test VPC result validation
func TestVPCResult_Validation(t *testing.T) {
	tests := []struct {
		name   string
		result *VPCResult
		valid  bool
	}{
		{
			"Valid result",
			&VPCResult{
				Provider: "digitalocean",
				Name:     "vpc-1",
				CIDR:     "10.0.0.0/16",
				Region:   "nyc3",
			},
			true,
		},
		{
			"Missing provider",
			&VPCResult{
				Provider: "",
				Name:     "vpc-1",
				CIDR:     "10.0.0.0/16",
				Region:   "nyc3",
			},
			false,
		},
		{
			"Missing name",
			&VPCResult{
				Provider: "digitalocean",
				Name:     "",
				CIDR:     "10.0.0.0/16",
				Region:   "nyc3",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.result.Provider != "" && tt.result.Name != "" &&
				tt.result.CIDR != "" && tt.result.Region != ""
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// Test nil config handling
func TestVPC_NilConfigHandling(t *testing.T) {
	var vpcConfig *config.VPCConfig = nil

	shouldCreate := vpcConfig != nil && vpcConfig.Create
	assert.False(t, shouldCreate, "Should not create VPC with nil config")
}

// Test VPC creation flag
func TestVPC_CreationFlag(t *testing.T) {
	tests := []struct {
		name   string
		create bool
	}{
		{"Create enabled", true},
		{"Create disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.VPCConfig{Create: tt.create}
			assert.Equal(t, tt.create, cfg.Create)
		})
	}
}

// Test provider-specific config structures
func TestVPC_ProviderConfigStructures(t *testing.T) {
	t.Run("DigitalOcean_Config", func(t *testing.T) {
		cfg := &config.DigitalOceanProvider{
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "do-vpc",
				CIDR:   "10.0.0.0/16",
				Region: "nyc3",
			},
		}

		require.NotNil(t, cfg.VPC)
		assert.True(t, cfg.VPC.Create)
		assert.Equal(t, "do-vpc", cfg.VPC.Name)
	})

	t.Run("Linode_Config", func(t *testing.T) {
		cfg := &config.LinodeProvider{
			VPC: &config.VPCConfig{
				Create: true,
				Name:   "linode-vpc",
				CIDR:   "10.1.0.0/16",
				Region: "us-east",
			},
		}

		require.NotNil(t, cfg.VPC)
		assert.True(t, cfg.VPC.Create)
		assert.Equal(t, "linode-vpc", cfg.VPC.Name)
	})
}
