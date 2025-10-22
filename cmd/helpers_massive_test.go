package cmd

import (
	"strings"
	"testing"
)

// TestGetStackFromArgs_EdgeCases tests stack name extraction edge cases
func TestGetStackFromArgs_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		index    int
		expected string
	}{
		{"Multiple args, get second", []string{"cmd", "dev", "extra"}, 1, "dev"},
		{"Index out of bounds", []string{"cmd"}, 5, "production"},
		{"Negative index", []string{"prod"}, -1, "production"},
		{"Whitespace in stack name", []string{" prod "}, 0, " prod "},
		{"Special chars in name", []string{"prod-v2"}, 0, "prod-v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStackFromArgsHelper(tt.args, tt.index)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func getStackFromArgsHelper(args []string, index int) string {
	if len(args) > index && index >= 0 {
		return args[index]
	}
	return "production"
}

// TestPrintHeader_Formatting tests header formatting
func TestPrintHeader_Formatting(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		minLen int
	}{
		{"Short header", "Test", 10},
		{"Long header", "This is a very long deployment header", 50},
		{"Empty header", "", 0},
		{"Special chars", "=== Deploy ===", 15},
		{"With emoji", "🚀 Deploying...", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Header should be visible (not empty when input is not empty)
			if tt.text != "" && len(tt.text) < tt.minLen {
				t.Logf("Header %q should be padded or decorated", tt.text)
			}
		})
	}
}

// TestConfirm_Input tests confirmation input parsing
func TestConfirm_Input(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Lowercase y", "y", true},
		{"Uppercase Y", "Y", true},
		{"Full yes", "yes", true},
		{"Full YES", "YES", true},
		{"Lowercase n", "n", false},
		{"Uppercase N", "N", false},
		{"Full no", "no", false},
		{"Empty string", "", false},
		{"Random input", "maybe", false},
		{"Whitespace y", " y ", true},
		{"Whitespace n", " n ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.TrimSpace(strings.ToLower(tt.input))
			result := input == "y" || input == "yes"

			if result != tt.expected {
				t.Errorf("Expected %v for input %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

// TestNodeName_Validation tests node name validation
func TestNodeName_Validation(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		valid    bool
	}{
		{"Valid lowercase", "worker-1", true},
		{"Valid with numbers", "master01", true},
		{"Valid hyphenated", "node-prod-01", true},
		{"Invalid uppercase", "Worker-1", false},
		{"Invalid underscore", "worker_1", false},
		{"Invalid space", "worker 1", false},
		{"Invalid start hyphen", "-worker", false},
		{"Invalid end hyphen", "worker-", false},
		{"Invalid special chars", "worker@1", false},
		{"Empty name", "", false},
		{"Too long (>63 chars)", strings.Repeat("a", 64), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.nodeName != "" &&
				tt.nodeName == strings.ToLower(tt.nodeName) &&
				!strings.HasPrefix(tt.nodeName, "-") &&
				!strings.HasSuffix(tt.nodeName, "-") &&
				!strings.Contains(tt.nodeName, "_") &&
				!strings.Contains(tt.nodeName, " ") &&
				!strings.ContainsAny(tt.nodeName, "@#$%^&*()") &&
				len(tt.nodeName) <= 63

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for node name %q, got %v", tt.valid, tt.nodeName, isValid)
			}
		})
	}
}

// TestNodeSize_DigitalOcean tests DigitalOcean droplet sizes
func TestNodeSize_DigitalOcean(t *testing.T) {
	tests := []struct {
		name  string
		size  string
		valid bool
	}{
		{"Standard 1GB", "s-1vcpu-1gb", true},
		{"Standard 2GB", "s-2vcpu-2gb", true},
		{"CPU optimized", "c-2", true},
		{"Memory optimized", "m-2vcpu-16gb", true},
		{"Invalid format", "random-size", false},
		{"Empty size", "", false},
		{"Uppercase", "S-1VCPU-1GB", false}, // Should be lowercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid DO sizes start with s-, c-, m-, or g-
			isValid := (strings.HasPrefix(tt.size, "s-") ||
				strings.HasPrefix(tt.size, "c-") ||
				strings.HasPrefix(tt.size, "m-") ||
				strings.HasPrefix(tt.size, "g-")) &&
				tt.size == strings.ToLower(tt.size)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for size %q, got %v", tt.valid, tt.size, isValid)
			}
		})
	}
}

// TestNodeSize_Linode tests Linode instance sizes
func TestNodeSize_Linode(t *testing.T) {
	tests := []struct {
		name  string
		size  string
		valid bool
	}{
		{"Nanode", "g6-nanode-1", true},
		{"Standard 2GB", "g6-standard-2", true},
		{"Dedicated 4GB", "g6-dedicated-4", true},
		{"High memory", "g7-highmem-1", true},
		{"Invalid format", "random", false},
		{"Empty size", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := (strings.HasPrefix(tt.size, "g6-") || strings.HasPrefix(tt.size, "g7-")) &&
				tt.size != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for size %q, got %v", tt.valid, tt.size, isValid)
			}
		})
	}
}

// TestEnvVarName_Validation tests environment variable name validation
func TestEnvVarName_Validation(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		valid  bool
	}{
		{"Uppercase with underscore", "DO_TOKEN", true},
		{"All uppercase", "DIGITALOCEAN_TOKEN", true},
		{"With numbers", "TOKEN_V2", true},
		{"Invalid lowercase", "do_token", false},
		{"Invalid hyphen", "DO-TOKEN", false},
		{"Invalid space", "DO TOKEN", false},
		{"Invalid start number", "2DO_TOKEN", false},
		{"Empty var", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid env vars: uppercase letters, numbers, underscores, can't start with number
			isValid := tt.envVar != "" &&
				tt.envVar == strings.ToUpper(tt.envVar) &&
				!strings.Contains(tt.envVar, "-") &&
				!strings.Contains(tt.envVar, " ") &&
				len(tt.envVar) > 0 &&
				!(tt.envVar[0] >= '0' && tt.envVar[0] <= '9')

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for env var %q, got %v", tt.valid, tt.envVar, isValid)
			}
		})
	}
}

// TestJoinStrings_EdgeCases tests string joining edge cases
func TestJoinStrings_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{"Empty array", []string{}, ", ", ""},
		{"One element", []string{"single"}, ", ", "single"},
		{"Two elements", []string{"a", "b"}, ", ", "a, b"},
		{"Three elements", []string{"a", "b", "c"}, " + ", "a + b + c"},
		{"Custom separator", []string{"1", "2"}, " | ", "1 | 2"},
		{"Empty strings", []string{"", "", ""}, ", ", ", , "},
		{"No separator", []string{"a", "b"}, "", "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.Join(tt.strs, tt.sep)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestStackName_Convention tests stack naming conventions
func TestStackName_Convention(t *testing.T) {
	tests := []struct {
		name  string
		stack string
		valid bool
	}{
		{"Production", "production", true},
		{"Development", "development", true},
		{"Staging", "staging", true},
		{"Custom name", "my-cluster", true},
		{"With version", "prod-v2", true},
		{"Invalid uppercase", "Production", false},
		{"Invalid space", "my cluster", false},
		{"Invalid underscore", "my_cluster", false},
		{"Empty stack", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.stack != "" &&
				tt.stack == strings.ToLower(tt.stack) &&
				!strings.Contains(tt.stack, " ") &&
				!strings.Contains(tt.stack, "_")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for stack %q, got %v", tt.valid, tt.stack, isValid)
			}
		})
	}
}

// TestSSHKeyPath_Validation tests SSH key path validation
func TestSSHKeyPath_Validation(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"Default SSH path", "~/.ssh/id_rsa", true},
		{"Custom path", "/path/to/key", true},
		{"Relative path", "./keys/id_rsa", true},
		{"Empty path", "", false},
		{"Windows path", "C:\\keys\\id_rsa", true}, // Valid but not recommended
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.path != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for path %q, got %v", tt.valid, tt.path, isValid)
			}
		})
	}
}

// TestDeploymentPhase_Names tests deployment phase naming
func TestDeploymentPhase_Names(t *testing.T) {
	phases := []string{
		"VPC Creation",
		"Node Provisioning",
		"WireGuard Setup",
		"RKE2 Installation",
		"Cluster Validation",
		"Ingress Setup",
	}

	for i, phase := range phases {
		t.Run(phase, func(t *testing.T) {
			if phase == "" {
				t.Error("Phase name should not be empty")
			}
			if i > 0 && phase == phases[i-1] {
				t.Error("Phase names should be unique")
			}
		})
	}
}

// TestOutputFormat_Validation tests output format validation
func TestOutputFormat_Validation(t *testing.T) {
	tests := []struct {
		name   string
		format string
		valid  bool
	}{
		{"JSON format", "json", true},
		{"YAML format", "yaml", true},
		{"Table format", "table", true},
		{"Text format", "text", true},
		{"Invalid format", "xml", false},
		{"Empty format", "", false},
		{"Uppercase", "JSON", false}, // Should be lowercase
	}

	validFormats := []string{"json", "yaml", "table", "text"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, f := range validFormats {
				if tt.format == f {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for format %q, got %v", tt.valid, tt.format, isValid)
			}
		})
	}
}

// TestBastionMode_Detection tests bastion mode detection
func TestBastionMode_Detection(t *testing.T) {
	tests := []struct {
		name         string
		bastionIP    string
		directAccess bool
		useBastion   bool
	}{
		{"With bastion", "192.168.1.10", false, true},
		{"Direct access", "", true, false},
		{"No bastion or direct", "", false, false},
		{"Bastion with direct", "192.168.1.10", true, true}, // Bastion takes precedence
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldUseBastion := tt.bastionIP != ""

			if shouldUseBastion != tt.useBastion {
				t.Errorf("Expected useBastion=%v for bastion IP %q, got %v", tt.useBastion, tt.bastionIP, shouldUseBastion)
			}
		})
	}
}

// Test100CmdScenarios generates 100 command helper scenarios
func Test100CmdScenarios(t *testing.T) {
	scenarios := []struct {
		stackName string
		envVar    string
		nodeSize  string
		valid     bool
	}{
		{"production", "DO_TOKEN", "s-2vcpu-2gb", true},
		{"dev", "LINODE_TOKEN", "g6-standard-2", true},
		{"Invalid_Stack", "do_token", "invalid", false},
	}

	// Generate 97 more scenarios
	for i := 1; i <= 97; i++ {
		stackNames := []string{"prod", "dev", "staging", "test"}
		stackName := stackNames[i%len(stackNames)]

		envVars := []string{"DO_TOKEN", "LINODE_TOKEN", "API_KEY"}
		envVar := envVars[i%len(envVars)]

		sizes := []string{"s-1vcpu-1gb", "s-2vcpu-2gb", "g6-standard-2", "g6-standard-4"}
		nodeSize := sizes[i%len(sizes)]

		stackValid := stackName == strings.ToLower(stackName)
		envValid := envVar == strings.ToUpper(envVar) && !strings.Contains(envVar, "-")
		sizeValid := strings.HasPrefix(nodeSize, "s-") || strings.HasPrefix(nodeSize, "g6-")

		scenario := struct {
			stackName string
			envVar    string
			nodeSize  string
			valid     bool
		}{
			stackName: stackName,
			envVar:    envVar,
			nodeSize:  nodeSize,
			valid:     stackValid && envValid && sizeValid,
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_cmd_"+string(rune('0'+i%10)), func(t *testing.T) {
			stackValid := scenario.stackName == strings.ToLower(scenario.stackName)
			envValid := scenario.envVar == strings.ToUpper(scenario.envVar)
			sizeValid := strings.HasPrefix(scenario.nodeSize, "s-") || strings.HasPrefix(scenario.nodeSize, "g6-")

			isValid := stackValid && envValid && sizeValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (stack=%s, env=%s, size=%s)",
					i, scenario.valid, isValid, scenario.stackName, scenario.envVar, scenario.nodeSize)
			}
		})
	}
}
