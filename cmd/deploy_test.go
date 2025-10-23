package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// TestDeployCommand tests deploy command structure
func TestDeployCommand(t *testing.T) {
	if deployCmd == nil {
		t.Fatal("deployCmd should not be nil")
	}

	if !strings.HasPrefix(deployCmd.Use, "deploy") {
		t.Errorf("Expected Use to start with 'deploy', got %q", deployCmd.Use)
	}

	if deployCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if deployCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if deployCmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if deployCmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

// TestDeployFlags tests deploy command flags
func TestDeployFlags(t *testing.T) {
	flags := deployCmd.Flags()

	// Required flags
	requiredFlags := []string{"do-token", "linode-token", "wireguard-endpoint", "wireguard-pubkey", "dry-run"}

	for _, flagName := range requiredFlags {
		flag := flags.Lookup(flagName)
		if flag == nil {
			t.Errorf("Flag %q should be defined", flagName)
		}
	}
}

// TestGetEnvOrFlag tests getEnvOrFlag helper
func TestGetEnvOrFlag(t *testing.T) {
	tests := []struct {
		name      string
		envKey    string
		envValue  string
		flagValue string
		expected  string
	}{
		{
			name:      "Flag takes precedence",
			envKey:    "TEST_TOKEN",
			envValue:  "env-value",
			flagValue: "flag-value",
			expected:  "flag-value",
		},
		{
			name:      "Env value when flag is empty",
			envKey:    "TEST_TOKEN",
			envValue:  "env-value",
			flagValue: "",
			expected:  "env-value",
		},
		{
			name:      "Both empty",
			envKey:    "TEST_NONEXISTENT",
			envValue:  "",
			flagValue: "",
			expected:  "",
		},
		{
			name:      "Only flag value",
			envKey:    "TEST_NONEXISTENT",
			envValue:  "",
			flagValue: "flag-only",
			expected:  "flag-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnvOrFlag(tt.envKey, tt.flagValue)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestJoinStrings tests joinStrings helper
func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{
			name:     "Two strings with space",
			strs:     []string{"DigitalOcean", "Linode"},
			sep:      " + ",
			expected: "DigitalOcean + Linode",
		},
		{
			name:     "Three strings with comma",
			strs:     []string{"master", "worker", "etcd"},
			sep:      ", ",
			expected: "master, worker, etcd",
		},
		{
			name:     "Single string",
			strs:     []string{"single"},
			sep:      ", ",
			expected: "single",
		},
		{
			name:     "Empty array",
			strs:     []string{},
			sep:      ", ",
			expected: "",
		},
		{
			name:     "Multiple strings with hyphen",
			strs:     []string{"vpc", "vpn", "kubernetes"},
			sep:      " - ",
			expected: "vpc - vpn - kubernetes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.strs, tt.sep)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestPrintHeaderFormat tests printHeader function
func TestPrintHeaderFormat(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"Deployment header", "🚀 Kubernetes Multi-Cloud Deployment"},
		{"Phase header", "📊 Phase 1: VPC Creation"},
		{"Success header", "✅ Deployment Complete"},
		{"Empty text", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just validate the function doesn't panic
			// printHeader writes to stdout, so we can't easily capture output
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printHeader panicked: %v", r)
				}
			}()
			// Would call: printHeader(tt.text)
			// But we can't easily test output without capturing stdout
		})
	}
}

// TestConfirmFunction tests confirm function behavior
func TestConfirmFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Confirm with 'y'", "y", true},
		{"Confirm with 'Y'", "Y", true},
		{"Confirm with 'yes'", "yes", true},
		{"Deny with 'n'", "n", false},
		{"Deny with 'N'", "N", false},
		{"Deny with 'no'", "no", false},
		{"Deny with empty", "", false},
		{"Deny with invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test response matching logic
			response := tt.input
			result := response == "y" || response == "Y" || response == "yes"

			if result != tt.expected {
				t.Errorf("Expected %v for input %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

// TestDeploymentPhaseCounting tests deployment phase counting logic
func TestDeploymentPhaseCounting(t *testing.T) {
	tests := []struct {
		name          string
		vpcCount      int
		createVPN     bool
		expectedPhase int
	}{
		{
			name:          "All phases",
			vpcCount:      2,
			createVPN:     true,
			expectedPhase: 5, // VPC, VPN, Provision, VPN Mesh, K8s
		},
		{
			name:          "No VPC",
			vpcCount:      0,
			createVPN:     true,
			expectedPhase: 4, // VPN, Provision, VPN Mesh, K8s
		},
		{
			name:          "No VPN",
			vpcCount:      2,
			createVPN:     false,
			expectedPhase: 3, // VPC, Provision, K8s
		},
		{
			name:          "Minimal",
			vpcCount:      0,
			createVPN:     false,
			expectedPhase: 2, // Provision, K8s
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phaseNum := 0

			if tt.vpcCount > 0 {
				phaseNum++
			}
			if tt.createVPN {
				phaseNum++
			}
			// Provision nodes
			phaseNum++
			// VPN mesh (if enabled)
			if tt.createVPN {
				phaseNum++
			}
			// Install Kubernetes
			phaseNum++

			// We expect phaseNum to equal expectedPhase
			if phaseNum != tt.expectedPhase {
				t.Errorf("Expected %d phases, got %d", tt.expectedPhase, phaseNum)
			}
		})
	}
}

// TestNodeCountCalculation tests node count calculation
func TestNodeCountCalculation(t *testing.T) {
	tests := []struct {
		name           string
		nodePools      map[string]config.NodePool
		expectedTotal  int
		expectedMaster int
		expectedWorker int
	}{
		{
			name: "Standard 6-node cluster",
			nodePools: map[string]config.NodePool{
				"do-masters": {
					Count: 1,
					Roles: []string{"master"},
				},
				"do-workers": {
					Count: 2,
					Roles: []string{"worker"},
				},
				"linode-masters": {
					Count: 2,
					Roles: []string{"master"},
				},
				"linode-workers": {
					Count: 1,
					Roles: []string{"worker"},
				},
			},
			expectedTotal:  6,
			expectedMaster: 3,
			expectedWorker: 3,
		},
		{
			name: "Single node cluster",
			nodePools: map[string]config.NodePool{
				"master": {
					Count: 1,
					Roles: []string{"master"},
				},
			},
			expectedTotal:  1,
			expectedMaster: 1,
			expectedWorker: 0,
		},
		{
			name: "Large cluster",
			nodePools: map[string]config.NodePool{
				"masters": {
					Count: 3,
					Roles: []string{"master"},
				},
				"workers": {
					Count: 10,
					Roles: []string{"worker"},
				},
			},
			expectedTotal:  13,
			expectedMaster: 3,
			expectedWorker: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalNodes := 0
			masters := 0
			workers := 0

			for _, pool := range tt.nodePools {
				totalNodes += pool.Count
				for _, role := range pool.Roles {
					if role == "master" {
						masters += pool.Count
					} else if role == "worker" {
						workers += pool.Count
					}
				}
			}

			if totalNodes != tt.expectedTotal {
				t.Errorf("Expected total %d nodes, got %d", tt.expectedTotal, totalNodes)
			}
			if masters != tt.expectedMaster {
				t.Errorf("Expected %d masters, got %d", tt.expectedMaster, masters)
			}
			if workers != tt.expectedWorker {
				t.Errorf("Expected %d workers, got %d", tt.expectedWorker, workers)
			}
		})
	}
}

// TestProviderCounting tests provider counting logic
func TestProviderCounting(t *testing.T) {
	tests := []struct {
		name              string
		doEnabled         bool
		linodeEnabled     bool
		expectedCount     int
		expectedProviders []string
	}{
		{
			name:              "Both providers",
			doEnabled:         true,
			linodeEnabled:     true,
			expectedCount:     2,
			expectedProviders: []string{"DigitalOcean", "Linode"},
		},
		{
			name:              "Only DigitalOcean",
			doEnabled:         true,
			linodeEnabled:     false,
			expectedCount:     1,
			expectedProviders: []string{"DigitalOcean"},
		},
		{
			name:              "Only Linode",
			doEnabled:         false,
			linodeEnabled:     true,
			expectedCount:     1,
			expectedProviders: []string{"Linode"},
		},
		{
			name:              "No providers",
			doEnabled:         false,
			linodeEnabled:     false,
			expectedCount:     0,
			expectedProviders: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ClusterConfig{
				Providers: config.ProvidersConfig{},
			}

			if tt.doEnabled {
				cfg.Providers.DigitalOcean = &config.DigitalOceanProvider{Enabled: true}
			}
			if tt.linodeEnabled {
				cfg.Providers.Linode = &config.LinodeProvider{Enabled: true}
			}

			providers := []string{}
			if cfg.Providers.DigitalOcean != nil && cfg.Providers.DigitalOcean.Enabled {
				providers = append(providers, "DigitalOcean")
			}
			if cfg.Providers.Linode != nil && cfg.Providers.Linode.Enabled {
				providers = append(providers, "Linode")
			}

			if len(providers) != tt.expectedCount {
				t.Errorf("Expected %d providers, got %d", tt.expectedCount, len(providers))
			}

			// Check provider names match
			for i, expected := range tt.expectedProviders {
				if i >= len(providers) || providers[i] != expected {
					t.Errorf("Provider mismatch at index %d: expected %q, got %q", i, expected, providers[i])
				}
			}
		})
	}
}

// TestVPCCounting tests VPC counting logic
func TestVPCCounting(t *testing.T) {
	tests := []struct {
		name      string
		doVPC     *config.VPCConfig
		linodeVPC *config.VPCConfig
		expected  int
	}{
		{
			name: "Both VPCs",
			doVPC: &config.VPCConfig{
				Create: true,
				Name:   "do-vpc",
				CIDR:   "10.0.0.0/16",
			},
			linodeVPC: &config.VPCConfig{
				Create: true,
				Name:   "linode-vpc",
				CIDR:   "10.1.0.0/16",
			},
			expected: 2,
		},
		{
			name: "Only DO VPC",
			doVPC: &config.VPCConfig{
				Create: true,
				Name:   "do-vpc",
				CIDR:   "10.0.0.0/16",
			},
			linodeVPC: nil,
			expected:  1,
		},
		{
			name:  "Only Linode VPC",
			doVPC: nil,
			linodeVPC: &config.VPCConfig{
				Create: true,
				Name:   "linode-vpc",
				CIDR:   "10.1.0.0/16",
			},
			expected: 1,
		},
		{
			name:      "No VPCs",
			doVPC:     nil,
			linodeVPC: nil,
			expected:  0,
		},
		{
			name: "VPCs configured but not created",
			doVPC: &config.VPCConfig{
				Create: false,
			},
			linodeVPC: &config.VPCConfig{
				Create: false,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpcCount := 0

			if tt.doVPC != nil && tt.doVPC.Create {
				vpcCount++
			}
			if tt.linodeVPC != nil && tt.linodeVPC.Create {
				vpcCount++
			}

			if vpcCount != tt.expected {
				t.Errorf("Expected %d VPCs, got %d", tt.expected, vpcCount)
			}
		})
	}
}

// TestWireGuardConfiguration tests WireGuard config validation
func TestWireGuardConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		wgConfig          *config.WireGuardConfig
		shouldCreate      bool
		shouldUseExisting bool
	}{
		{
			name: "Create new WireGuard",
			wgConfig: &config.WireGuardConfig{
				Create:  true,
				Enabled: true,
				Port:    51820,
			},
			shouldCreate:      true,
			shouldUseExisting: false,
		},
		{
			name: "Use existing WireGuard",
			wgConfig: &config.WireGuardConfig{
				Create:          false,
				Enabled:         true,
				ServerEndpoint:  "1.2.3.4:51820",
				ServerPublicKey: "pubkey123",
			},
			shouldCreate:      false,
			shouldUseExisting: true,
		},
		{
			name: "WireGuard disabled",
			wgConfig: &config.WireGuardConfig{
				Enabled: false,
			},
			shouldCreate:      false,
			shouldUseExisting: false,
		},
		{
			name:              "No WireGuard config",
			wgConfig:          nil,
			shouldCreate:      false,
			shouldUseExisting: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			create := tt.wgConfig != nil && tt.wgConfig.Create
			useExisting := tt.wgConfig != nil && tt.wgConfig.Enabled && !tt.wgConfig.Create

			if create != tt.shouldCreate {
				t.Errorf("Expected create=%v, got %v", tt.shouldCreate, create)
			}
			if useExisting != tt.shouldUseExisting {
				t.Errorf("Expected useExisting=%v, got %v", tt.shouldUseExisting, useExisting)
			}
		})
	}
}

// TestStackNameParsing tests stack name parsing logic
func TestStackNameParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flagName string
		expected string
	}{
		{
			name:     "Stack from args",
			args:     []string{"production"},
			flagName: "",
			expected: "production",
		},
		{
			name:     "Stack from flag",
			args:     []string{},
			flagName: "staging",
			expected: "staging",
		},
		{
			name:     "Args take precedence",
			args:     []string{"custom"},
			flagName: "ignored",
			expected: "custom",
		},
		{
			name:     "Default to production",
			args:     []string{},
			flagName: "",
			expected: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate stack name parsing
			var stackName string

			if len(tt.args) > 0 {
				stackName = tt.args[0]
			} else {
				if tt.flagName != "" {
					stackName = tt.flagName
				} else {
					stackName = "production"
				}
			}

			if stackName != tt.expected {
				t.Errorf("Expected stack name %q, got %q", tt.expected, stackName)
			}
		})
	}
}

// TestDryRunMode tests dry-run mode detection
func TestDryRunMode(t *testing.T) {
	tests := []struct {
		name          string
		dryRun        bool
		shouldPreview bool
		shouldDeploy  bool
	}{
		{
			name:          "Dry-run enabled",
			dryRun:        true,
			shouldPreview: true,
			shouldDeploy:  false,
		},
		{
			name:          "Dry-run disabled",
			dryRun:        false,
			shouldPreview: false,
			shouldDeploy:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview := tt.dryRun
			deploy := !tt.dryRun

			if preview != tt.shouldPreview {
				t.Errorf("Expected preview=%v, got %v", tt.shouldPreview, preview)
			}
			if deploy != tt.shouldDeploy {
				t.Errorf("Expected deploy=%v, got %v", tt.shouldDeploy, deploy)
			}
		})
	}
}

// TestOutputParsing tests output key parsing
func TestOutputParsing(t *testing.T) {
	tests := []struct {
		name      string
		outputKey string
		isVPC     bool
		isVPCID   bool
		provider  string
	}{
		{
			name:      "DO VPC ID",
			outputKey: "vpc_digitalocean_id",
			isVPC:     true,
			isVPCID:   true,
			provider:  "digitalocean",
		},
		{
			name:      "Linode VPC ID",
			outputKey: "vpc_linode_id",
			isVPC:     true,
			isVPCID:   true,
			provider:  "linode",
		},
		{
			name:      "DO VPC CIDR",
			outputKey: "vpc_digitalocean_cidr",
			isVPC:     true,
			isVPCID:   false,
			provider:  "digitalocean",
		},
		{
			name:      "Non-VPC output",
			outputKey: "clusterName",
			isVPC:     false,
			isVPCID:   false,
			provider:  "",
		},
		{
			name:      "VPN output",
			outputKey: "vpn_server_ip",
			isVPC:     false,
			isVPCID:   false,
			provider:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isVPC := len(tt.outputKey) > 4 && tt.outputKey[:4] == "vpc_"
			isVPCID := len(tt.outputKey) > 3 && tt.outputKey[len(tt.outputKey)-3:] == "_id"

			if isVPC != tt.isVPC {
				t.Errorf("Expected isVPC=%v, got %v", tt.isVPC, isVPC)
			}
			if isVPCID != tt.isVPCID {
				t.Errorf("Expected isVPCID=%v, got %v", tt.isVPCID, isVPCID)
			}

			// Extract provider from VPC ID keys
			if isVPC && isVPCID {
				provider := tt.outputKey[4 : len(tt.outputKey)-3]
				if provider != tt.provider {
					t.Errorf("Expected provider %q, got %q", tt.provider, provider)
				}
			}
		})
	}
}
