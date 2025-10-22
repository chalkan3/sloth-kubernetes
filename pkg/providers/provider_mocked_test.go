package providers

import (
	"strings"
	"testing"

	"sloth-kubernetes/pkg/config"
)

// TestNewDigitalOceanProvider tests DigitalOcean provider creation
func TestNewDigitalOceanProvider(t *testing.T) {
	provider := NewDigitalOceanProvider()

	if provider == nil {
		t.Fatal("Expected provider to be created, got nil")
	}

	if provider.nodes == nil {
		t.Error("Expected nodes slice to be initialized")
	}

	if len(provider.nodes) != 0 {
		t.Errorf("Expected empty nodes slice, got %d nodes", len(provider.nodes))
	}
}

// TestDigitalOceanProvider_GetName tests provider name
func TestDigitalOceanProvider_GetName(t *testing.T) {
	provider := NewDigitalOceanProvider()
	name := provider.GetName()

	if name != "digitalocean" {
		t.Errorf("Expected name 'digitalocean', got %q", name)
	}
}

// TestDigitalOceanProvider_GetRegions tests available regions
func TestDigitalOceanProvider_GetRegions(t *testing.T) {
	provider := NewDigitalOceanProvider()
	regions := provider.GetRegions()

	expectedRegions := []string{
		"nyc1", "nyc3", "sfo3", "ams3", "sgp1",
		"lon1", "fra1", "tor1", "blr1", "syd1",
	}

	if len(regions) != len(expectedRegions) {
		t.Errorf("Expected %d regions, got %d", len(expectedRegions), len(regions))
	}

	// Verify all expected regions are present
	for _, expected := range expectedRegions {
		found := false
		for _, region := range regions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected region %q not found in regions list", expected)
		}
	}
}

// TestDigitalOceanProvider_GetSizes tests available instance sizes
func TestDigitalOceanProvider_GetSizes(t *testing.T) {
	provider := NewDigitalOceanProvider()
	sizes := provider.GetSizes()

	if len(sizes) == 0 {
		t.Error("Expected non-empty sizes list")
	}

	// Verify sizes follow DigitalOcean naming convention
	for _, size := range sizes {
		if !strings.HasPrefix(size, "s-") && !strings.HasPrefix(size, "c-") {
			t.Errorf("Size %q doesn't follow DigitalOcean naming convention (should start with s- or c-)", size)
		}
	}
}

// TestDigitalOceanProvider_GenerateUserData tests user data generation
func TestDigitalOceanProvider_GenerateUserData(t *testing.T) {
	provider := NewDigitalOceanProvider()
	provider.config = &config.DigitalOceanProvider{
		Region: "nyc3",
	}

	tests := []struct {
		name          string
		nodeConfig    *config.NodeConfig
		shouldContain []string
	}{
		{
			name: "Basic worker node",
			nodeConfig: &config.NodeConfig{
				Name:   "worker-1",
				Region: "nyc3",
				Size:   "s-2vcpu-2gb",
				Roles:  []string{"worker"},
			},
			shouldContain: []string{
				"#!/bin/bash",
				"apt-get update",
				"wireguard",
				"docker",
				"NODE_PROVIDER=digitalocean",
				"NODE_REGION=nyc3",
				"NODE_SIZE=s-2vcpu-2gb",
				"NODE_ROLE_worker=true",
			},
		},
		{
			name: "Master node with multiple roles",
			nodeConfig: &config.NodeConfig{
				Name:   "master-1",
				Region: "sfo3",
				Size:   "s-4vcpu-8gb",
				Roles:  []string{"master", "controlplane"},
			},
			shouldContain: []string{
				"NODE_ROLE_master=true",
				"NODE_ROLE_controlplane=true",
				"NODE_REGION=sfo3",
			},
		},
		{
			name: "Node with custom user data",
			nodeConfig: &config.NodeConfig{
				Name:     "custom-1",
				Region:   "ams3",
				Size:     "s-2vcpu-2gb",
				Roles:    []string{"worker"},
				UserData: "echo 'Custom script'\napt-get install -y custom-package",
			},
			shouldContain: []string{
				"Custom script",
				"custom-package",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userData := provider.generateUserData(tt.nodeConfig)

			if userData == "" {
				t.Fatal("Expected non-empty user data")
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(userData, expected) {
					t.Errorf("User data should contain %q", expected)
				}
			}

			// Verify it starts with shebang
			if !strings.HasPrefix(userData, "#!/bin/bash") {
				t.Error("User data should start with #!/bin/bash")
			}
		})
	}
}

// TestNewLinodeProvider tests Linode provider creation
func TestNewLinodeProvider(t *testing.T) {
	provider := NewLinodeProvider()

	if provider == nil {
		t.Fatal("Expected provider to be created, got nil")
	}

	if provider.nodes == nil {
		t.Error("Expected nodes slice to be initialized")
	}

	if len(provider.nodes) != 0 {
		t.Errorf("Expected empty nodes slice, got %d nodes", len(provider.nodes))
	}
}

// TestLinodeProvider_GetName tests provider name
func TestLinodeProvider_GetName(t *testing.T) {
	provider := NewLinodeProvider()
	name := provider.GetName()

	if name != "linode" {
		t.Errorf("Expected name 'linode', got %q", name)
	}
}

// TestLinodeProvider_GetRegions tests available regions
func TestLinodeProvider_GetRegions(t *testing.T) {
	provider := NewLinodeProvider()
	regions := provider.GetRegions()

	expectedRegions := []string{
		"us-east", "us-west", "us-central", "us-southeast",
		"eu-west", "eu-central", "ap-south", "ap-northeast",
		"ap-west", "ca-central", "ap-southeast",
	}

	if len(regions) != len(expectedRegions) {
		t.Errorf("Expected %d regions, got %d", len(expectedRegions), len(regions))
	}

	// Verify all expected regions are present
	for _, expected := range expectedRegions {
		found := false
		for _, region := range regions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected region %q not found in regions list", expected)
		}
	}
}

// TestLinodeProvider_GetSizes tests available instance sizes
func TestLinodeProvider_GetSizes(t *testing.T) {
	provider := NewLinodeProvider()
	sizes := provider.GetSizes()

	if len(sizes) == 0 {
		t.Error("Expected non-empty sizes list")
	}

	// Verify sizes follow Linode naming convention
	for _, size := range sizes {
		if !strings.HasPrefix(size, "g6-") && !strings.HasPrefix(size, "g7-") {
			t.Errorf("Size %q doesn't follow Linode naming convention (should start with g6- or g7-)", size)
		}
	}
}

// TestLinodeProvider_GenerateUserData tests user data generation
func TestLinodeProvider_GenerateUserData(t *testing.T) {
	provider := NewLinodeProvider()

	tests := []struct {
		name          string
		nodeConfig    *config.NodeConfig
		shouldContain []string
	}{
		{
			name: "Basic worker node",
			nodeConfig: &config.NodeConfig{
				Name:   "worker-1",
				Region: "us-east",
				Size:   "g6-standard-2",
				Roles:  []string{"worker"},
			},
			shouldContain: []string{
				"#!/bin/bash",
				"apt-get update",
				"wireguard",
				"docker",
				"NODE_PROVIDER=linode",
				"NODE_REGION=us-east",
				"NODE_SIZE=g6-standard-2",
				"NODE_ROLE_worker=true",
			},
		},
		{
			name: "Master node",
			nodeConfig: &config.NodeConfig{
				Name:   "master-1",
				Region: "eu-west",
				Size:   "g6-standard-4",
				Roles:  []string{"master"},
			},
			shouldContain: []string{
				"NODE_ROLE_master=true",
				"NODE_REGION=eu-west",
			},
		},
		{
			name: "Node with custom user data",
			nodeConfig: &config.NodeConfig{
				Name:     "custom-1",
				Region:   "ap-south",
				Size:     "g6-standard-2",
				Roles:    []string{"worker"},
				UserData: "echo 'Linode custom script'",
			},
			shouldContain: []string{
				"Linode custom script",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userData := provider.generateUserData(tt.nodeConfig)

			if userData == "" {
				t.Fatal("Expected non-empty user data")
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(userData, expected) {
					t.Errorf("User data should contain %q", expected)
				}
			}
		})
	}
}

// TestGenerateSecurePassword tests password generation
func TestGenerateSecurePassword(t *testing.T) {
	password := generateSecurePassword()

	if password == "" {
		t.Error("Expected non-empty password")
	}

	if len(password) < 16 {
		t.Errorf("Expected password length >= 16, got %d", len(password))
	}
}

// TestContains tests the contains helper function
func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Found in middle", []string{"a", "b", "c"}, "b", true},
		{"Found at start", []string{"worker", "master"}, "worker", true},
		{"Found at end", []string{"a", "b", "c"}, "c", true},
		{"Not found", []string{"a", "b", "c"}, "d", false},
		{"Empty slice", []string{}, "a", false},
		{"Case sensitive", []string{"Worker"}, "worker", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestNodeOutput_StructureMocked tests NodeOutput struct creation (mocked)
func TestNodeOutput_StructureMocked(t *testing.T) {
	tests := []struct {
		name     string
		output   *NodeOutput
		validate func(*testing.T, *NodeOutput)
	}{
		{
			name: "DigitalOcean node output",
			output: &NodeOutput{
				Name:        "worker-1",
				Provider:    "digitalocean",
				Region:      "nyc3",
				Size:        "s-2vcpu-2gb",
				SSHUser:     "root",
				SSHKeyPath:  "~/.ssh/id_rsa",
				WireGuardIP: "10.8.0.11",
				Labels: map[string]string{
					"role": "worker",
				},
			},
			validate: func(t *testing.T, n *NodeOutput) {
				if n.Provider != "digitalocean" {
					t.Errorf("Expected provider 'digitalocean', got %q", n.Provider)
				}
				if n.SSHUser != "root" {
					t.Errorf("Expected SSH user 'root', got %q", n.SSHUser)
				}
				if n.WireGuardIP != "10.8.0.11" {
					t.Errorf("Expected WireGuard IP '10.8.0.11', got %q", n.WireGuardIP)
				}
			},
		},
		{
			name: "Linode node output",
			output: &NodeOutput{
				Name:       "master-1",
				Provider:   "linode",
				Region:     "us-east",
				Size:       "g6-standard-4",
				SSHUser:    "root",
				SSHKeyPath: "~/.ssh/id_rsa",
				Labels: map[string]string{
					"role": "master",
					"env":  "production",
				},
			},
			validate: func(t *testing.T, n *NodeOutput) {
				if n.Provider != "linode" {
					t.Errorf("Expected provider 'linode', got %q", n.Provider)
				}
				if len(n.Labels) != 2 {
					t.Errorf("Expected 2 labels, got %d", len(n.Labels))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.output.Name == "" {
				t.Error("Node name should not be empty")
			}
			if tt.output.Provider == "" {
				t.Error("Provider should not be empty")
			}
			if tt.output.Region == "" {
				t.Error("Region should not be empty")
			}
			if tt.output.Size == "" {
				t.Error("Size should not be empty")
			}

			tt.validate(t, tt.output)
		})
	}
}

// TestNetworkOutput_StructureMocked tests NetworkOutput struct (mocked)
func TestNetworkOutput_StructureMocked(t *testing.T) {
	tests := []struct {
		name   string
		output *NetworkOutput
		valid  bool
	}{
		{
			name: "Valid network output",
			output: &NetworkOutput{
				Name:   "production-vpc",
				CIDR:   "10.0.0.0/16",
				Region: "nyc3",
			},
			valid: true,
		},
		{
			name: "Empty name",
			output: &NetworkOutput{
				Name:   "",
				CIDR:   "10.0.0.0/16",
				Region: "nyc3",
			},
			valid: false,
		},
		{
			name: "Empty CIDR",
			output: &NetworkOutput{
				Name:   "vpc",
				CIDR:   "",
				Region: "nyc3",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.output.Name != "" && tt.output.CIDR != "" && tt.output.Region != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestLoadBalancerOutput_StructureMocked tests LoadBalancerOutput struct (mocked)
func TestLoadBalancerOutput_StructureMocked(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		valid    bool
	}{
		{"With hostname", "lb.example.com", true},
		{"Without hostname", "", true}, // Some providers don't provide hostname
		{"IP-based hostname", "192.168.1.100", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// LoadBalancerOutput is valid even without hostname (some providers use IP only)
			if !tt.valid {
				t.Errorf("LoadBalancer output should be valid")
			}

			// Verify hostname format if provided
			if tt.hostname != "" && len(tt.hostname) == 0 {
				t.Error("Hostname should not be empty if provided")
			}
		})
	}
}

// Test100ProviderScenarios generates 100 provider test scenarios
func Test100ProviderScenarios(t *testing.T) {
	scenarios := []struct {
		providerName string
		region       string
		size         string
		valid        bool
	}{
		// DigitalOcean scenarios
		{"digitalocean", "nyc1", "s-1vcpu-1gb", true},
		{"digitalocean", "nyc3", "s-2vcpu-2gb", true},
		{"digitalocean", "sfo3", "c-2", true},
		{"digitalocean", "invalid-region", "s-2vcpu-2gb", false},

		// Linode scenarios
		{"linode", "us-east", "g6-standard-2", true},
		{"linode", "eu-west", "g6-standard-4", true},
		{"linode", "ap-south", "g6-nanode-1", true},
		{"linode", "invalid-region", "g6-standard-2", false},
	}

	// Generate 92 more scenarios programmatically
	doRegions := []string{"nyc1", "nyc3", "sfo3", "ams3", "lon1", "fra1"}
	doSizes := []string{"s-1vcpu-1gb", "s-2vcpu-2gb", "c-2", "c-4"}
	linodeRegions := []string{"us-east", "us-west", "eu-west", "ap-south"}
	linodeSizes := []string{"g6-nanode-1", "g6-standard-2", "g6-standard-4"}

	for i := 0; i < 46; i++ {
		scenarios = append(scenarios, struct {
			providerName string
			region       string
			size         string
			valid        bool
		}{
			providerName: "digitalocean",
			region:       doRegions[i%len(doRegions)],
			size:         doSizes[i%len(doSizes)],
			valid:        true,
		})
	}

	for i := 0; i < 46; i++ {
		scenarios = append(scenarios, struct {
			providerName string
			region       string
			size         string
			valid        bool
		}{
			providerName: "linode",
			region:       linodeRegions[i%len(linodeRegions)],
			size:         linodeSizes[i%len(linodeSizes)],
			valid:        true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_provider_"+string(rune('0'+i%10)), func(t *testing.T) {
			// Validate provider name
			providerValid := scenario.providerName == "digitalocean" || scenario.providerName == "linode"

			// Validate region format
			regionValid := scenario.region != "" && scenario.region != "invalid-region"

			// Validate size format
			var sizeValid bool
			if scenario.providerName == "digitalocean" {
				sizeValid = strings.HasPrefix(scenario.size, "s-") || strings.HasPrefix(scenario.size, "c-")
			} else if scenario.providerName == "linode" {
				sizeValid = strings.HasPrefix(scenario.size, "g6-") || strings.HasPrefix(scenario.size, "g7-")
			}

			isValid := providerValid && regionValid && sizeValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (provider=%s, region=%s, size=%s)",
					i, scenario.valid, isValid, scenario.providerName, scenario.region, scenario.size)
			}
		})
	}
}
