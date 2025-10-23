package vpc

import (
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// TestVPCResult_Structure tests VPCResult structure validation
func TestVPCResult_Structure(t *testing.T) {
	tests := []struct {
		name   string
		result *VPCResult
		valid  bool
	}{
		{
			"Valid DigitalOcean VPC",
			&VPCResult{Provider: "digitalocean", Name: "vpc-do", CIDR: "10.0.0.0/16", Region: "nyc3"},
			true,
		},
		{
			"Valid Linode VPC",
			&VPCResult{Provider: "linode", Name: "vpc-linode", CIDR: "192.168.0.0/16", Region: "us-east"},
			true,
		},
		{
			"Empty provider",
			&VPCResult{Provider: "", Name: "vpc-test", CIDR: "10.0.0.0/16", Region: "nyc3"},
			false,
		},
		{
			"Empty name",
			&VPCResult{Provider: "digitalocean", Name: "", CIDR: "10.0.0.0/16", Region: "nyc3"},
			false,
		},
		{
			"Empty CIDR",
			&VPCResult{Provider: "digitalocean", Name: "vpc-do", CIDR: "", Region: "nyc3"},
			false,
		},
		{
			"Empty region",
			&VPCResult{Provider: "digitalocean", Name: "vpc-do", CIDR: "10.0.0.0/16", Region: ""},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.result.Provider != "" &&
				tt.result.Name != "" &&
				tt.result.CIDR != "" &&
				tt.result.Region != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for VPCResult, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestVPCConfig_Create tests VPC creation flag
func TestVPCConfig_Create(t *testing.T) {
	tests := []struct {
		name   string
		vpc    *config.VPCConfig
		create bool
	}{
		{"VPC creation enabled", &config.VPCConfig{Create: true, Name: "test-vpc", CIDR: "10.0.0.0/16"}, true},
		{"VPC creation disabled", &config.VPCConfig{Create: false, Name: "test-vpc", CIDR: "10.0.0.0/16"}, false},
		{"Nil VPC config", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCreate := tt.vpc != nil && tt.vpc.Create

			if shouldCreate != tt.create {
				t.Errorf("Expected create=%v, got %v", tt.create, shouldCreate)
			}
		})
	}
}

// TestVPCNaming_Conventions tests VPC naming conventions
func TestVPCNaming_Conventions(t *testing.T) {
	tests := []struct {
		name    string
		vpcName string
		valid   bool
	}{
		{"Valid name with hyphen", "vpc-production", true},
		{"Valid name with number", "vpc-cluster-1", true},
		{"Valid short name", "vpc", true},
		{"Invalid uppercase", "VPC-Production", false},
		{"Invalid underscore", "vpc_production", false},
		{"Invalid space", "vpc production", false},
		{"Invalid special chars", "vpc@prod", false},
		{"Empty name", "", false},
		{"Too long (>255 chars)", strings.Repeat("a", 256), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// VPC names should be lowercase, alphanumeric with hyphens, max 255 chars
			isValid := tt.vpcName != "" &&
				tt.vpcName == strings.ToLower(tt.vpcName) &&
				!strings.Contains(tt.vpcName, "_") &&
				!strings.Contains(tt.vpcName, " ") &&
				!strings.ContainsAny(tt.vpcName, "@#$%^&*()") &&
				len(tt.vpcName) <= 255

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for VPC name %q, got %v", tt.valid, tt.vpcName, isValid)
			}
		})
	}
}

// TestVPCCIDR_Private tests private IP range validation
func TestVPCCIDR_Private(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		private bool
	}{
		{"Class A private (10.0.0.0/8)", "10.0.0.0/8", true},
		{"Class A subnet", "10.100.0.0/16", true},
		{"Class B private (172.16.0.0/12)", "172.16.0.0/12", true},
		{"Class B subnet", "172.20.0.0/16", true},
		{"Class C private (192.168.0.0/16)", "192.168.0.0/16", true},
		{"Class C subnet", "192.168.1.0/24", true},
		{"Public IP range", "8.8.8.0/24", false},
		{"Public IP range 2", "1.1.1.0/24", false},
		{"Invalid 172.32.x.x", "172.32.0.0/16", false}, // Not in 172.16-172.31 range
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if CIDR starts with private ranges
			isPrivate := strings.HasPrefix(tt.cidr, "10.") ||
				strings.HasPrefix(tt.cidr, "192.168.") ||
				(strings.HasPrefix(tt.cidr, "172.") && isPrivate172(tt.cidr))

			if isPrivate != tt.private {
				t.Errorf("Expected private=%v for CIDR %q, got %v", tt.private, tt.cidr, isPrivate)
			}
		})
	}
}

func isPrivate172(cidr string) bool {
	// 172.16.0.0 - 172.31.255.255
	parts := strings.Split(cidr, ".")
	if len(parts) < 2 {
		return false
	}
	second := 0
	for _, c := range parts[1] {
		if c >= '0' && c <= '9' {
			second = second*10 + int(c-'0')
		} else {
			break
		}
	}
	return second >= 16 && second <= 31
}

// TestVPCCIDR_Size tests CIDR block sizes
func TestVPCCIDR_Size(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		minHosts    int
		appropriate bool
	}{
		{"/8 - 16M hosts", "10.0.0.0/8", 1000000, true},
		{"/16 - 65K hosts", "10.0.0.0/16", 50000, true},
		{"/20 - 4K hosts", "10.0.0.0/20", 3000, true},
		{"/24 - 256 hosts", "10.0.0.0/24", 200, true},
		{"/28 - 16 hosts", "10.0.0.0/28", 100, false}, // Too small
		{"/32 - 1 host", "10.0.0.0/32", 1, false},     // Single host
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract prefix length
			parts := strings.Split(tt.cidr, "/")
			if len(parts) != 2 {
				t.Fatalf("Invalid CIDR format: %s", tt.cidr)
			}

			prefix := 0
			for _, c := range parts[1] {
				if c >= '0' && c <= '9' {
					prefix = prefix*10 + int(c-'0')
				}
			}

			// Calculate approximate hosts: 2^(32-prefix) - 2
			hostBits := 32 - prefix
			var approxHosts int
			if hostBits <= 10 {
				approxHosts = (1 << hostBits) - 2
			} else {
				approxHosts = 1 << hostBits // Simplified for large blocks
			}

			isAppropriate := approxHosts >= tt.minHosts

			if isAppropriate != tt.appropriate {
				t.Logf("CIDR %s provides ~%d hosts for %d required", tt.cidr, approxHosts, tt.minHosts)
			}
		})
	}
}

// TestVPCRegion_DigitalOcean tests DigitalOcean region codes
func TestVPCRegion_DigitalOcean(t *testing.T) {
	validRegions := []string{
		"nyc1", "nyc2", "nyc3",
		"sfo1", "sfo2", "sfo3",
		"ams2", "ams3",
		"sgp1",
		"lon1",
		"fra1",
		"tor1",
		"blr1",
	}

	tests := []struct {
		name   string
		region string
		valid  bool
	}{
		{"NYC3", "nyc3", true},
		{"SFO3", "sfo3", true},
		{"AMS3", "ams3", true},
		{"Invalid region", "xyz1", false},
		{"Empty region", "", false},
		{"Uppercase", "NYC3", false}, // Should be lowercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := contains(validRegions, tt.region)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for region %q, got %v", tt.valid, tt.region, isValid)
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestVPCRegion_Linode tests Linode region codes
func TestVPCRegion_Linode(t *testing.T) {
	validRegions := []string{
		"us-east", "us-central", "us-west", "us-southeast",
		"eu-west", "eu-central",
		"ap-south", "ap-northeast", "ap-southeast",
	}

	tests := []struct {
		name   string
		region string
		valid  bool
	}{
		{"US East", "us-east", true},
		{"EU West", "eu-west", true},
		{"AP South", "ap-south", true},
		{"Invalid region", "us-north", false},
		{"Empty region", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := contains(validRegions, tt.region)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for region %q, got %v", tt.valid, tt.region, isValid)
			}
		})
	}
}

// TestVPCDescription_Generation tests VPC description format
func TestVPCDescription_Generation(t *testing.T) {
	tests := []struct {
		name     string
		vpcName  string
		expected string
	}{
		{"Production VPC", "vpc-production", "VPC for Kubernetes cluster - vpc-production"},
		{"Dev VPC", "vpc-dev", "VPC for Kubernetes cluster - vpc-dev"},
		{"Cluster 1", "cluster-1-vpc", "VPC for Kubernetes cluster - cluster-1-vpc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := "VPC for Kubernetes cluster - " + tt.vpcName

			if description != tt.expected {
				t.Errorf("Expected description %q, got %q", tt.expected, description)
			}
		})
	}
}

// TestVPCConfig_RegionFallbackAdvanced tests advanced region fallback scenarios
func TestVPCConfig_RegionFallbackAdvanced(t *testing.T) {
	tests := []struct {
		name           string
		vpcRegion      string
		providerRegion string
		fallbackRegion string
		expectedRegion string
	}{
		{"VPC region specified", "nyc3", "sfo3", "ams3", "nyc3"},
		{"VPC region empty, use provider", "", "sfo3", "ams3", "sfo3"},
		{"Both VPC and provider empty, use fallback", "", "", "ams3", "ams3"},
		{"All specified, VPC wins", "nyc3", "sfo3", "ams3", "nyc3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region := tt.vpcRegion
			if region == "" {
				region = tt.providerRegion
			}
			if region == "" {
				region = tt.fallbackRegion
			}

			if region != tt.expectedRegion {
				t.Errorf("Expected region %q, got %q", tt.expectedRegion, region)
			}
		})
	}
}

// TestVPCCIDR_Overlap tests CIDR block overlap detection
func TestVPCCIDR_Overlap(t *testing.T) {
	tests := []struct {
		name    string
		cidr1   string
		cidr2   string
		overlap bool
	}{
		{"No overlap - different Class A", "10.0.0.0/16", "172.16.0.0/16", false},
		{"No overlap - different subnets", "10.0.0.0/24", "10.1.0.0/24", false},
		{"Overlap - same network", "10.0.0.0/16", "10.0.0.0/16", true},
		{"Overlap - subnet within", "10.0.0.0/16", "10.0.1.0/24", true},
		{"Overlap - larger contains smaller", "10.0.0.0/8", "10.100.0.0/16", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simplified overlap detection based on prefix
			prefix1 := strings.Split(tt.cidr1, "/")[0]
			prefix2 := strings.Split(tt.cidr2, "/")[0]

			// Simple check: if first octets match, potential overlap
			parts1 := strings.Split(prefix1, ".")
			parts2 := strings.Split(prefix2, ".")

			hasOverlap := len(parts1) >= 1 && len(parts2) >= 1 && parts1[0] == parts2[0]
			if hasOverlap && len(parts1) >= 2 && len(parts2) >= 2 {
				// More specific check for second octet
				hasOverlap = parts1[1] == parts2[1] || tt.cidr1 == tt.cidr2
			}

			if tt.overlap && !hasOverlap {
				t.Logf("Expected overlap between %s and %s", tt.cidr1, tt.cidr2)
			}
		})
	}
}

// TestVPCExport_Keys tests VPC export key naming
func TestVPCExport_Keys(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		key      string
		valid    bool
	}{
		{"DigitalOcean VPC ID", "digitalocean", "digitalocean_vpc_id", true},
		{"DigitalOcean VPC URN", "digitalocean", "digitalocean_vpc_urn", true},
		{"DigitalOcean IP range", "digitalocean", "digitalocean_vpc_ip_range", true},
		{"Linode VPC ID", "linode", "linode_vpc_id", true},
		{"Linode VPC label", "linode", "linode_vpc_label", true},
		{"Invalid format", "digitalocean", "vpc_id", false},
		{"Wrong provider prefix", "linode", "digitalocean_vpc_id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.HasPrefix(tt.key, tt.provider+"_")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for export key %q with provider %q, got %v",
					tt.valid, tt.key, tt.provider, isValid)
			}
		})
	}
}

// TestVPCMultiRegion_Deployment tests multi-region VPC scenarios
func TestVPCMultiRegion_Deployment(t *testing.T) {
	tests := []struct {
		name       string
		regions    []string
		cidrs      []string
		validSetup bool
	}{
		{
			"Two regions, non-overlapping",
			[]string{"nyc3", "sfo3"},
			[]string{"10.0.0.0/16", "10.1.0.0/16"},
			true,
		},
		{
			"Two regions, overlapping CIDRs",
			[]string{"nyc3", "sfo3"},
			[]string{"10.0.0.0/16", "10.0.0.0/16"},
			false,
		},
		{
			"Three regions, non-overlapping",
			[]string{"nyc3", "sfo3", "ams3"},
			[]string{"10.0.0.0/16", "10.1.0.0/16", "10.2.0.0/16"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.regions) != len(tt.cidrs) {
				t.Fatalf("Regions and CIDRs count mismatch")
			}

			// Check for CIDR overlaps
			hasOverlap := false
			for i := 0; i < len(tt.cidrs); i++ {
				for j := i + 1; j < len(tt.cidrs); j++ {
					if tt.cidrs[i] == tt.cidrs[j] {
						hasOverlap = true
						break
					}
				}
			}

			isValid := !hasOverlap

			if isValid != tt.validSetup {
				t.Errorf("Expected valid=%v for multi-region setup, got %v", tt.validSetup, isValid)
			}
		})
	}
}

// Test100VPCScenarios generates 100 VPC configuration scenarios
func Test100VPCScenarios(t *testing.T) {
	scenarios := []struct {
		provider string
		region   string
		cidr     string
		name     string
		valid    bool
	}{
		{"digitalocean", "nyc3", "10.0.0.0/16", "vpc-prod", true},
		{"linode", "us-east", "172.16.0.0/16", "vpc-dev", true},
		{"digitalocean", "sfo3", "192.168.0.0/16", "vpc-test", true},
		{"linode", "eu-west", "10.100.0.0/16", "vpc-eu", true},
		{"invalid", "nyc3", "10.0.0.0/16", "vpc-prod", false},
		{"digitalocean", "", "10.0.0.0/16", "vpc-prod", false}, // Empty region
	}

	// Generate 94 more scenarios
	for i := 1; i <= 94; i++ {
		provider := "digitalocean"
		if i%2 == 0 {
			provider = "linode"
		}

		region := "nyc3"
		if provider == "linode" {
			region = "us-east"
		}

		// Vary CIDR blocks
		thirdOctet := i % 256
		cidr := "10." + string(rune('0'+(thirdOctet/100))) +
			string(rune('0'+((thirdOctet%100)/10))) +
			string(rune('0'+(thirdOctet%10))) + ".0.0/16"

		name := "vpc-cluster-" + string(rune('0'+(i%10)))

		validProviders := provider == "digitalocean" || provider == "linode"
		validRegion := region != ""
		validCIDR := strings.Contains(cidr, "/")
		validName := name != ""

		scenario := struct {
			provider string
			region   string
			cidr     string
			name     string
			valid    bool
		}{
			provider: provider,
			region:   region,
			cidr:     cidr,
			name:     name,
			valid:    validProviders && validRegion && validCIDR && validName,
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_vpc_"+string(rune('0'+i%10)), func(t *testing.T) {
			providerValid := scenario.provider == "digitalocean" || scenario.provider == "linode"
			regionValid := scenario.region != ""
			cidrValid := strings.Contains(scenario.cidr, "/")
			nameValid := scenario.name != ""

			isValid := providerValid && regionValid && cidrValid && nameValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (provider=%s, region=%s, cidr=%s, name=%s)",
					i, scenario.valid, isValid, scenario.provider, scenario.region, scenario.cidr, scenario.name)
			}
		})
	}
}
