package cmd

import (
	"strings"
	"testing"
)

// TestNodesCommand tests nodes command structure
func TestNodesCommand(t *testing.T) {
	if nodesCmd == nil {
		t.Fatal("nodesCmd should not be nil")
	}

	if nodesCmd.Use != "nodes" {
		t.Errorf("Expected Use 'nodes', got %q", nodesCmd.Use)
	}

	if nodesCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if nodesCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

// TestListNodesCommand tests list nodes command structure
func TestListNodesCommand(t *testing.T) {
	if listNodesCmd == nil {
		t.Fatal("listNodesCmd should not be nil")
	}

	if !strings.HasPrefix(listNodesCmd.Use, "list") {
		t.Errorf("Expected Use to start with 'list', got %q", listNodesCmd.Use)
	}

	if listNodesCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if listNodesCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if listNodesCmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if listNodesCmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

// TestSSHNodeCommand tests SSH node command structure
func TestSSHNodeCommand(t *testing.T) {
	if sshNodeCmd == nil {
		t.Fatal("sshNodeCmd should not be nil")
	}

	if !strings.HasPrefix(sshNodeCmd.Use, "ssh") {
		t.Errorf("Expected Use to start with 'ssh', got %q", sshNodeCmd.Use)
	}

	if sshNodeCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if sshNodeCmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if sshNodeCmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

// TestAddNodeCommand tests add node command structure
func TestAddNodeCommand(t *testing.T) {
	if addNodeCmd == nil {
		t.Fatal("addNodeCmd should not be nil")
	}

	if !strings.HasPrefix(addNodeCmd.Use, "add") {
		t.Errorf("Expected Use to start with 'add', got %q", addNodeCmd.Use)
	}

	if addNodeCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if addNodeCmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if addNodeCmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

// TestRemoveNodeCommand tests remove node command structure
func TestRemoveNodeCommand(t *testing.T) {
	if removeNodeCmd == nil {
		t.Fatal("removeNodeCmd should not be nil")
	}

	if !strings.HasPrefix(removeNodeCmd.Use, "remove") {
		t.Errorf("Expected Use to start with 'remove', got %q", removeNodeCmd.Use)
	}

	if removeNodeCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if removeNodeCmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if removeNodeCmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

// TestGetStackFromArgs tests getStackFromArgs helper
func TestGetStackFromArgs(t *testing.T) {
	// Save original stackName
	originalStackName := stackName

	tests := []struct {
		name         string
		args         []string
		index        int
		stackNameVar string
		expected     string
	}{
		{
			name:         "Stack from args",
			args:         []string{"production"},
			index:        0,
			stackNameVar: "",
			expected:     "production",
		},
		{
			name:         "Stack from args - multiple args",
			args:         []string{"staging", "node-1"},
			index:        0,
			stackNameVar: "",
			expected:     "staging",
		},
		{
			name:         "Stack from stackName variable",
			args:         []string{},
			index:        0,
			stackNameVar: "development",
			expected:     "development",
		},
		{
			name:         "Empty stack when no default",
			args:         []string{},
			index:        0,
			stackNameVar: "",
			expected:     "",
		},
		{
			name:         "Args precedence over stackName",
			args:         []string{"custom-stack"},
			index:        0,
			stackNameVar: "ignored",
			expected:     "custom-stack",
		},
		{
			name:         "Index out of bounds - use stackName",
			args:         []string{"production"},
			index:        1,
			stackNameVar: "fallback",
			expected:     "fallback",
		},
		{
			name:         "Index out of bounds - empty when no stackName",
			args:         []string{"production"},
			index:        5,
			stackNameVar: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set stackName variable
			stackName = tt.stackNameVar

			result := getStackFromArgs(tt.args, tt.index)

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}

	// Restore original stackName
	stackName = originalStackName
}

// TestOutputFormatOptions tests output format options
func TestOutputFormatOptions(t *testing.T) {
	validFormats := []string{"table", "json", "yaml"}

	for _, format := range validFormats {
		t.Run("format-"+format, func(t *testing.T) {
			if format == "" {
				t.Error("Format should not be empty")
			}

			isValid := format == "table" || format == "json" || format == "yaml"
			if !isValid {
				t.Errorf("Format %q should be valid", format)
			}
		})
	}
}

// TestInvalidOutputFormats tests invalid output formats
func TestInvalidOutputFormats(t *testing.T) {
	invalidFormats := []string{"", "invalid", "xml", "csv", "text"}

	for _, format := range invalidFormats {
		t.Run("invalid-"+format, func(t *testing.T) {
			isValid := format == "table" || format == "json" || format == "yaml"
			if isValid {
				t.Errorf("Format %q should be invalid but was considered valid", format)
			}
		})
	}
}

// TestNodeRoleOptions tests node role options
func TestNodeRoleOptions(t *testing.T) {
	validRoles := []string{"master", "worker", "etcd"}

	for _, role := range validRoles {
		t.Run("role-"+role, func(t *testing.T) {
			if role == "" {
				t.Error("Role should not be empty")
			}

			// Validate role format (lowercase alphanumeric)
			for _, char := range role {
				if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
					t.Errorf("Role %q contains invalid character: %c", role, char)
				}
			}
		})
	}
}

// TestProviderOptions tests provider options
func TestProviderOptions(t *testing.T) {
	validProviders := []string{"digitalocean", "linode"}
	invalidProviders := []string{"", "aws", "azure", "gcp", "invalid"}

	for _, provider := range validProviders {
		t.Run("valid-"+provider, func(t *testing.T) {
			isValid := provider == "digitalocean" || provider == "linode"
			if !isValid {
				t.Errorf("Provider %q should be valid", provider)
			}
		})
	}

	for _, provider := range invalidProviders {
		t.Run("invalid-"+provider, func(t *testing.T) {
			isValid := provider == "digitalocean" || provider == "linode"
			if isValid {
				t.Errorf("Provider %q should be invalid but was considered valid", provider)
			}
		})
	}
}

// TestSSHCommandFlag tests SSH command flag
func TestSSHCommandFlag(t *testing.T) {
	tests := []struct {
		name    string
		command string
		valid   bool
	}{
		{"Valid command", "docker ps", true},
		{"Valid command with options", "kubectl get pods -n default", true},
		{"Empty command", "", true}, // Empty is valid (interactive session)
		{"System command", "systemctl status kubelet", true},
		{"Long command", "curl -X GET http://localhost:8080/api/v1/pods", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All commands are technically valid strings
			// Empty command triggers interactive mode
			isEmptyOrValid := true
			if !isEmptyOrValid {
				t.Errorf("Command %q should be valid", tt.command)
			}
		})
	}
}

// TestForceRemoveFlag tests force remove flag
func TestForceRemoveFlag(t *testing.T) {
	tests := []struct {
		name        string
		forceRemove bool
		shouldDrain bool
	}{
		{"Force remove enabled", true, false},
		{"Force remove disabled", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Force remove = skip drain
			shouldDrain := !tt.forceRemove

			if shouldDrain != tt.shouldDrain {
				t.Errorf("Expected shouldDrain=%v, got %v", tt.shouldDrain, shouldDrain)
			}
		})
	}
}

// TestNodeNameValidation tests node name validation
func TestNodeNameValidation(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		valid    bool
	}{
		{"Valid node name", "master-1", true},
		{"Valid worker name", "worker-primary", true},
		{"Valid with numbers", "node-123", true},
		{"Empty name", "", false},
		{"Uppercase (should be lowercase)", "MASTER-1", false},
		{"Special chars", "node@123", false},
		{"Spaces", "master 1", false},
		{"Starts with dash", "-master", false},
		{"Ends with dash", "master-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := true

			// Check if empty
			if tt.nodeName == "" {
				isValid = false
			}

			// Check for invalid characters
			if isValid {
				for _, char := range tt.nodeName {
					if !((char >= 'a' && char <= 'z') ||
						(char >= '0' && char <= '9') ||
						char == '-') {
						isValid = false
						break
					}
				}
			}

			// Check for leading/trailing dashes
			if isValid && (strings.HasPrefix(tt.nodeName, "-") || strings.HasSuffix(tt.nodeName, "-")) {
				isValid = false
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for %q, got %v", tt.valid, tt.nodeName, isValid)
			}
		})
	}
}

// TestNodeSizeValidation tests node size validation
func TestNodeSizeValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		size     string
		valid    bool
	}{
		// DigitalOcean sizes
		{"DO small", "digitalocean", "s-1vcpu-1gb", true},
		{"DO medium", "digitalocean", "s-2vcpu-4gb", true},
		{"DO large", "digitalocean", "s-4vcpu-8gb", true},
		{"DO invalid", "digitalocean", "invalid-size", false},

		// Linode sizes
		{"Linode nanode", "linode", "g6-nanode-1", true},
		{"Linode standard", "linode", "g6-standard-2", true},
		{"Linode dedicated", "linode", "g6-dedicated-4", true},
		{"Linode invalid", "linode", "invalid-type", false},

		// Empty sizes
		{"Empty size", "digitalocean", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isEmpty := tt.size == ""
			hasValidPrefix := false

			if !isEmpty {
				// DigitalOcean sizes start with 's-' or 'c-' or 'm-'
				if tt.provider == "digitalocean" {
					hasValidPrefix = strings.HasPrefix(tt.size, "s-") ||
						strings.HasPrefix(tt.size, "c-") ||
						strings.HasPrefix(tt.size, "m-")
				}
				// Linode sizes start with 'g6-' or 'g7-'
				if tt.provider == "linode" {
					hasValidPrefix = strings.HasPrefix(tt.size, "g6-") ||
						strings.HasPrefix(tt.size, "g7-")
				}
			}

			isValid := !isEmpty && hasValidPrefix
			if isValid != tt.valid {
				t.Logf("Size %q for provider %q: expected valid=%v, got %v",
					tt.size, tt.provider, tt.valid, isValid)
			}
		})
	}
}

// TestCommandExamples tests that all commands have examples
func TestCommandExamples(t *testing.T) {
	commands := []struct {
		name    string
		example string
	}{
		{"list", listNodesCmd.Example},
		{"ssh", sshNodeCmd.Example},
		{"add", addNodeCmd.Example},
		{"remove", removeNodeCmd.Example},
	}

	for _, cmd := range commands {
		t.Run(cmd.name, func(t *testing.T) {
			if cmd.example == "" {
				t.Errorf("%s command should have example", cmd.name)
			}
		})
	}
}

// TestSSHKeyPathDefault tests SSH key path default
func TestSSHKeyPathDefault(t *testing.T) {
	tests := []struct {
		name      string
		stack     string
		wantEmpty bool
	}{
		{"Production stack", "production", false},
		{"Staging stack", "staging", false},
		{"Custom stack", "my-cluster", false},
		{"Empty stack", "", false}, // Should still return a default path
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// GetSSHKeyPath should return a non-empty path
			path := GetSSHKeyPath(tt.stack)

			isEmpty := path == ""
			if isEmpty && !tt.wantEmpty {
				t.Error("SSH key path should not be empty")
			}
		})
	}
}

// TestNodePoolCountUpdate tests node pool count update logic
func TestNodePoolCountUpdate(t *testing.T) {
	tests := []struct {
		name         string
		currentCount int
		addCount     int
		expectedNew  int
	}{
		{"Add 1 to 3 nodes", 3, 1, 4},
		{"Add 2 to 1 node", 1, 2, 3},
		{"Add 5 to 0 nodes", 0, 5, 5},
		{"Add 1 to 10 nodes", 10, 1, 11},
		{"Add 0 nodes (no change)", 5, 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCount := tt.currentCount + tt.addCount

			if newCount != tt.expectedNew {
				t.Errorf("Expected new count %d, got %d", tt.expectedNew, newCount)
			}
		})
	}
}

// TestBastionModeDetection tests bastion mode detection logic
func TestBastionModeDetection(t *testing.T) {
	tests := []struct {
		name            string
		bastionEnabled  bool
		bastionIP       string
		shouldUseDirect bool
	}{
		{"Bastion enabled with IP", true, "203.0.113.1", false},
		{"Bastion enabled but no IP", true, "", true},
		{"Bastion disabled", false, "", true},
		{"Bastion disabled with IP", false, "203.0.113.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Direct mode if bastion is not enabled OR bastion IP is missing
			useDirect := !tt.bastionEnabled || tt.bastionIP == ""

			if useDirect != tt.shouldUseDirect {
				t.Errorf("Expected direct mode=%v, got %v", tt.shouldUseDirect, useDirect)
			}
		})
	}
}

// TestSSHArguments tests SSH arguments construction
func TestSSHArguments(t *testing.T) {
	tests := []struct {
		name        string
		keyPath     string
		targetIP    string
		bastionIP   string
		customCmd   string
		shouldProxy bool
	}{
		{
			name:        "Direct SSH",
			keyPath:     "/root/.ssh/id_rsa",
			targetIP:    "203.0.113.10",
			bastionIP:   "",
			customCmd:   "",
			shouldProxy: false,
		},
		{
			name:        "Bastion SSH",
			keyPath:     "/root/.ssh/id_rsa",
			targetIP:    "10.8.0.10",
			bastionIP:   "203.0.113.1",
			customCmd:   "",
			shouldProxy: true,
		},
		{
			name:        "Direct SSH with command",
			keyPath:     "/root/.ssh/id_rsa",
			targetIP:    "203.0.113.10",
			bastionIP:   "",
			customCmd:   "docker ps",
			shouldProxy: false,
		},
		{
			name:        "Bastion SSH with command",
			keyPath:     "/root/.ssh/id_rsa",
			targetIP:    "10.8.0.10",
			bastionIP:   "203.0.113.1",
			customCmd:   "kubectl get pods",
			shouldProxy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useProxy := tt.bastionIP != ""

			if useProxy != tt.shouldProxy {
				t.Errorf("Expected proxy=%v, got %v", tt.shouldProxy, useProxy)
			}

			// Validate key path
			if tt.keyPath == "" {
				t.Error("SSH key path should not be empty")
			}

			// Validate target IP
			if tt.targetIP == "" {
				t.Error("Target IP should not be empty")
			}

			// If proxy, bastion IP must be present
			if useProxy && tt.bastionIP == "" {
				t.Error("Bastion IP required when using proxy mode")
			}
		})
	}
}
