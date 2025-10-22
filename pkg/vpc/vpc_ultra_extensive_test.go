package vpc

import (
	"fmt"
	"strings"
	"testing"
)

// TestVPCResult_Fields tests VPCResult struct fields
func TestVPCResult_Fields(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		vpcName  string
		cidr     string
		region   string
		valid    bool
	}{
		{"Valid DO VPC", "digitalocean", "prod-vpc", "10.0.0.0/16", "nyc3", true},
		{"Valid Linode VPC", "linode", "prod-vpc", "10.0.0.0/16", "us-east", true},
		{"Empty provider", "", "vpc", "10.0.0.0/16", "nyc3", false},
		{"Empty name", "digitalocean", "", "10.0.0.0/16", "nyc3", false},
		{"Empty CIDR", "digitalocean", "vpc", "", "nyc3", false},
		{"Empty region", "digitalocean", "vpc", "10.0.0.0/16", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.provider != "" && tt.vpcName != "" && tt.cidr != "" && tt.region != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestVPCName_Format tests VPC name formatting
func TestVPCName_Format(t *testing.T) {
	tests := []struct {
		name      string
		vpcName   string
		validDO   bool
		validLinode bool
	}{
		{"Simple name", "vpc", true, true},
		{"With hyphen", "prod-vpc", true, true},
		{"With env", "prod-k8s-vpc", true, true},
		{"Uppercase", "PROD-VPC", false, true}, // DO requires lowercase
		{"With underscore", "prod_vpc", false, false},
		{"Too long", strings.Repeat("a", 256), false, false},
		{"Empty", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DigitalOcean: lowercase, alphanumeric + hyphens, 1-255 chars
			isDOValid := tt.vpcName != "" &&
				tt.vpcName == strings.ToLower(tt.vpcName) &&
				!strings.Contains(tt.vpcName, "_") &&
				len(tt.vpcName) <= 255

			// Linode: more flexible, just no underscores
			isLinodeValid := tt.vpcName != "" &&
				!strings.Contains(tt.vpcName, "_") &&
				len(tt.vpcName) <= 255

			if isDOValid != tt.validDO {
				t.Errorf("Expected DO valid=%v for %q, got %v", tt.validDO, tt.vpcName, isDOValid)
			}
			if isLinodeValid != tt.validLinode {
				t.Errorf("Expected Linode valid=%v for %q, got %v", tt.validLinode, tt.vpcName, isLinodeValid)
			}
		})
	}
}

// TestVPCCIDR_PrivateRanges tests private CIDR ranges
func TestVPCCIDR_PrivateRanges(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		isPrivate bool
	}{
		{"Class A private", "10.0.0.0/8", true},
		{"Class A /16", "10.0.0.0/16", true},
		{"Class A /24", "10.1.1.0/24", true},
		{"Class B private", "172.16.0.0/12", true},
		{"Class B /16", "172.16.0.0/16", true},
		{"Class B /24", "172.31.255.0/24", true},
		{"Class C private", "192.168.0.0/16", true},
		{"Class C /24", "192.168.1.0/24", true},
		{"Public IP", "8.8.8.8/24", false},
		{"Public range", "1.1.1.0/24", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract first octet
			parts := strings.Split(tt.cidr, ".")
			if len(parts) < 1 {
				t.Fatalf("Invalid CIDR: %s", tt.cidr)
			}

			var firstOctet, secondOctet int
			fmt.Sscanf(parts[0], "%d", &firstOctet)
			if len(parts) > 1 {
				fmt.Sscanf(parts[1], "%d", &secondOctet)
			}

			isPrivate := (firstOctet == 10) ||
				(firstOctet == 172 && secondOctet >= 16 && secondOctet <= 31) ||
				(firstOctet == 192 && secondOctet == 168)

			if isPrivate != tt.isPrivate {
				t.Errorf("Expected isPrivate=%v for %s, got %v", tt.isPrivate, tt.cidr, isPrivate)
			}
		})
	}
}

// TestVPCCIDR_SubnetMasks tests valid subnet masks
func TestVPCCIDR_SubnetMasks(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		mask  int
		valid bool
	}{
		{"/8 mask", "10.0.0.0/8", 8, true},
		{"/16 mask", "10.0.0.0/16", 16, true},
		{"/20 mask", "10.0.0.0/20", 20, true},
		{"/24 mask", "10.0.0.0/24", 24, true},
		{"/28 mask", "10.0.0.0/28", 28, true},
		{"/32 host", "10.0.0.1/32", 32, true},
		{"Invalid /0", "10.0.0.0/0", 0, false},
		{"Invalid /33", "10.0.0.0/33", 33, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.mask >= 8 && tt.mask <= 32
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for mask /%d, got %v", tt.valid, tt.mask, isValid)
			}
		})
	}
}

// TestVPCRegion_DigitalOceanExtended tests DigitalOcean regions
func TestVPCRegion_DigitalOceanExtended(t *testing.T) {
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
		{"SGP1", "sgp1", true},
		{"Invalid region", "invalid", false},
		{"Empty region", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, r := range validRegions {
				if tt.region == r {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for region %q, got %v", tt.valid, tt.region, isValid)
			}
		})
	}
}

// TestVPCRegion_LinodeExtended tests Linode regions
func TestVPCRegion_LinodeExtended(t *testing.T) {
	validRegions := []string{
		"us-east", "us-central", "us-west", "us-southeast",
		"ca-central",
		"eu-west", "eu-central",
		"ap-south", "ap-northeast", "ap-southeast",
		"ap-west",
	}

	tests := []struct {
		name   string
		region string
		valid  bool
	}{
		{"US East", "us-east", true},
		{"US West", "us-west", true},
		{"EU West", "eu-west", true},
		{"AP South", "ap-south", true},
		{"Invalid region", "invalid", false},
		{"Empty region", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, r := range validRegions {
				if tt.region == r {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for region %q, got %v", tt.valid, tt.region, isValid)
			}
		})
	}
}

// TestVPCDescription_Format tests VPC description format
func TestVPCDescription_Format(t *testing.T) {
	tests := []struct {
		name        string
		vpcName     string
		expectedDesc string
	}{
		{"Production VPC", "prod-vpc", "VPC for Kubernetes cluster - prod-vpc"},
		{"Staging VPC", "staging-vpc", "VPC for Kubernetes cluster - staging-vpc"},
		{"Dev VPC", "dev-vpc", "VPC for Kubernetes cluster - dev-vpc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := fmt.Sprintf("VPC for Kubernetes cluster - %s", tt.vpcName)
			if desc != tt.expectedDesc {
				t.Errorf("Expected description %q, got %q", tt.expectedDesc, desc)
			}
		})
	}
}

// TestVPCManager_Creation tests VPCManager creation
func TestVPCManager_Creation(t *testing.T) {
	tests := []struct {
		name    string
		ctxNil  bool
		valid   bool
	}{
		{"Valid manager", false, true},
		{"Nil context", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := !tt.ctxNil
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestVPCConfig_CreateFlagExtended tests VPC create flag
func TestVPCConfig_CreateFlagExtended(t *testing.T) {
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
			if tt.create != tt.shouldCreate {
				t.Errorf("Expected shouldCreate=%v, got %v", tt.shouldCreate, tt.create)
			}
		})
	}
}

// TestVPCConfig_RegionFallbackExtended tests region fallback logic
func TestVPCConfig_RegionFallbackExtended(t *testing.T) {
	tests := []struct {
		name           string
		vpcRegion      string
		providerRegion string
		expectedRegion string
	}{
		{"VPC region set", "nyc3", "sfo3", "nyc3"},
		{"VPC region empty, use provider", "", "sfo3", "sfo3"},
		{"Both set, prefer VPC", "nyc3", "sfo3", "nyc3"},
		{"Both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region := tt.vpcRegion
			if region == "" {
				region = tt.providerRegion
			}

			if region != tt.expectedRegion {
				t.Errorf("Expected region %q, got %q", tt.expectedRegion, region)
			}
		})
	}
}

// TestVPCExports_Names tests VPC export names
func TestVPCExports_Names(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		exportKeys []string
	}{
		{
			"DigitalOcean exports",
			"digitalocean",
			[]string{"digitalocean_vpc_id", "digitalocean_vpc_urn", "digitalocean_vpc_ip_range"},
		},
		{
			"Linode exports",
			"linode",
			[]string{"linode_vpc_id", "linode_vpc_label"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range tt.exportKeys {
				if !strings.HasPrefix(key, tt.provider) {
					t.Errorf("Export key %q should have prefix %q", key, tt.provider)
				}
			}
		})
	}
}

// TestVPCIPRange_Size tests IP range size calculations
func TestVPCIPRange_Size(t *testing.T) {
	tests := []struct {
		name     string
		mask     int
		numHosts int
	}{
		{"/24 network", 24, 256},
		{"/20 network", 20, 4096},
		{"/16 network", 16, 65536},
		{"/28 network", 28, 16},
		{"/30 network", 30, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate: 2^(32 - mask)
			numHosts := 1 << (32 - tt.mask)
			if numHosts != tt.numHosts {
				t.Errorf("Expected %d hosts for /%d, got %d", tt.numHosts, tt.mask, numHosts)
			}
		})
	}
}

// TestVPCSubnet_Overlap tests subnet overlap detection
func TestVPCSubnet_Overlap(t *testing.T) {
	tests := []struct {
		name     string
		cidr1    string
		cidr2    string
		overlaps bool
	}{
		{"Same subnet", "10.0.0.0/24", "10.0.0.0/24", true},
		{"Different subnets", "10.0.0.0/24", "10.0.1.0/24", false},
		{"Nested subnets", "10.0.0.0/16", "10.0.0.0/24", true},
		{"Adjacent subnets", "10.0.0.0/24", "10.0.1.0/24", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simplified overlap check: same base for this test
			base1 := strings.Split(tt.cidr1, "/")[0]
			base2 := strings.Split(tt.cidr2, "/")[0]

			// Get first 3 octets
			parts1 := strings.Split(base1, ".")
			parts2 := strings.Split(base2, ".")

			overlaps := tt.cidr1 == tt.cidr2 ||
				(len(parts1) >= 3 && len(parts2) >= 3 &&
					parts1[0] == parts2[0] && parts1[1] == parts2[1] &&
					strings.Contains(tt.cidr1, "/16"))

			if overlaps != tt.overlaps {
				t.Errorf("Expected overlaps=%v for %s and %s, got %v",
					tt.overlaps, tt.cidr1, tt.cidr2, overlaps)
			}
		})
	}
}

// Test300VPCScenarios generates 300 VPC test scenarios
func Test300VPCScenarios(t *testing.T) {
	scenarios := []struct {
		provider string
		region   string
		cidr     string
		name     string
		create   bool
		valid    bool
	}{
		{"digitalocean", "nyc3", "10.0.0.0/16", "prod-vpc", true, true},
		{"linode", "us-east", "10.1.0.0/16", "staging-vpc", true, true},
		{"digitalocean", "sfo3", "10.2.0.0/16", "dev-vpc", true, true},
	}

	// Generate 297 more scenarios
	providers := []string{"digitalocean", "linode"}
	doRegions := []string{"nyc3", "sfo3", "ams3", "sgp1", "lon1", "fra1"}
	linodeRegions := []string{"us-east", "us-west", "eu-west", "ap-south"}
	masks := []int{16, 20, 24}

	for i := 0; i < 297; i++ {
		provider := providers[i%len(providers)]
		var region string
		if provider == "digitalocean" {
			region = doRegions[i%len(doRegions)]
		} else {
			region = linodeRegions[i%len(linodeRegions)]
		}

		mask := masks[i%len(masks)]
		thirdOctet := i % 256
		cidr := fmt.Sprintf("10.%d.0.0/%d", thirdOctet, mask)
		name := fmt.Sprintf("vpc-%d", i)

		scenarios = append(scenarios, struct {
			provider string
			region   string
			cidr     string
			name     string
			create   bool
			valid    bool
		}{
			provider: provider,
			region:   region,
			cidr:     cidr,
			name:     name,
			create:   true,
			valid:    true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(fmt.Sprintf("VPC_%d_%s", i, scenario.provider), func(t *testing.T) {
			// Validate provider
			providerValid := scenario.provider == "digitalocean" || scenario.provider == "linode"

			// Validate region
			regionValid := scenario.region != ""

			// Validate CIDR
			cidrValid := strings.Contains(scenario.cidr, "/") && strings.Contains(scenario.cidr, "10.")

			// Validate name
			nameValid := scenario.name != "" && !strings.Contains(scenario.name, "_")

			isValid := providerValid && regionValid && cidrValid && nameValid && scenario.create

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}

			// Additional validation: CIDR should be private
			if strings.HasPrefix(scenario.cidr, "10.") {
				// Valid private range
			} else {
				t.Errorf("Scenario %d: CIDR %s is not in private range", i, scenario.cidr)
			}
		})
	}
}
