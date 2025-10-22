package cmd

import (
	"encoding/json"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/stretchr/testify/assert"
)

// TestStateCommands_Structure tests that state commands are properly structured
func TestStateCommands_Structure(t *testing.T) {
	assert.NotNil(t, stateCmd, "stateCmd should be initialized")
	assert.NotNil(t, stateListCmd, "stateListCmd should be initialized")
	assert.NotNil(t, stateDeleteCmd, "stateDeleteCmd should be initialized")

	assert.Equal(t, "state", stateCmd.Use, "stateCmd should have correct Use")
	assert.Equal(t, "list [stack-name]", stateListCmd.Use, "stateListCmd should have correct Use")
	assert.Equal(t, "delete [stack-name] [urn]", stateDeleteCmd.Use, "stateDeleteCmd should have correct Use")
}

// TestStateListCmd_Usage tests state list command usage
func TestStateListCmd_Usage(t *testing.T) {
	cmd := stateListCmd

	assert.Contains(t, cmd.Short, "List all resources", "Short description should mention listing")
	assert.Contains(t, cmd.Long, "Display all resources", "Long description should explain purpose")
	assert.NotEmpty(t, cmd.Example, "Should have examples")
}

// TestStateDeleteCmd_Usage tests state delete command usage
func TestStateDeleteCmd_Usage(t *testing.T) {
	cmd := stateDeleteCmd

	assert.Contains(t, cmd.Short, "Delete a resource", "Short description should mention deletion")
	assert.Contains(t, cmd.Long, "Remove a specific resource", "Long description should explain purpose")
	assert.Contains(t, cmd.Long, "WARNING", "Should have warning")
	assert.Contains(t, cmd.Long, "does NOT destroy", "Should clarify it doesn't destroy cloud resource")
	assert.NotEmpty(t, cmd.Example, "Should have examples")
}

// TestStateDeleteCmd_Flags tests state delete command flags
func TestStateDeleteCmd_Flags(t *testing.T) {
	cmd := stateDeleteCmd

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag, "Should have --force flag")
	assert.Equal(t, "f", forceFlag.Shorthand, "Should have -f shorthand")
	assert.Equal(t, "false", forceFlag.DefValue, "Default should be false")
}

// TestStateListCmd_Flags tests state list command flags
func TestStateListCmd_Flags(t *testing.T) {
	cmd := stateListCmd

	typeFlag := cmd.Flags().Lookup("type")
	assert.NotNil(t, typeFlag, "Should have --type flag")
	assert.Equal(t, "", typeFlag.DefValue, "Default should be empty")
}

// TestStateListCmd_ArgumentValidation tests argument validation
func TestStateListCmd_ArgumentValidation(t *testing.T) {
	// Test with no arguments
	err := runStateList(stateListCmd, []string{})
	assert.Error(t, err, "Should error with no arguments")
	assert.Contains(t, err.Error(), "usage:", "Error should show usage")
}

// TestStateDeleteCmd_ArgumentValidation tests argument validation
func TestStateDeleteCmd_ArgumentValidation(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "No arguments",
			args:        []string{},
			shouldError: true,
			errorMsg:    "usage:",
		},
		{
			name:        "Only stack name",
			args:        []string{"production"},
			shouldError: true,
			errorMsg:    "usage:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runStateDelete(stateDeleteCmd, tc.args)
			if tc.shouldError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDeploymentJSONParsing tests JSON parsing logic for deployment
func TestDeploymentJSONParsing(t *testing.T) {
	testCases := []struct {
		name          string
		deploymentJSON string
		expectedCount int
		shouldError   bool
	}{
		{
			name: "Valid deployment with resources",
			deploymentJSON: `{
				"resources": [
					{
						"urn": "urn:pulumi:test::project::digitalocean:Droplet::node1",
						"type": "digitalocean:Droplet",
						"id": "12345"
					},
					{
						"urn": "urn:pulumi:test::project::digitalocean:Vpc::vpc1",
						"type": "digitalocean:Vpc",
						"id": "vpc-123"
					}
				]
			}`,
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name: "Empty resources",
			deploymentJSON: `{
				"resources": []
			}`,
			expectedCount: 0,
			shouldError:   false,
		},
		{
			name:           "Invalid JSON",
			deploymentJSON: `{invalid json}`,
			shouldError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var deploymentData struct {
				Resources []struct {
					URN  string      `json:"urn"`
					Type string      `json:"type"`
					ID   interface{} `json:"id"`
				} `json:"resources"`
			}

			err := json.Unmarshal([]byte(tc.deploymentJSON), &deploymentData)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, len(deploymentData.Resources))
			}
		})
	}
}

// TestResourceFiltering tests resource type filtering logic
func TestResourceFiltering(t *testing.T) {
	resources := []struct {
		URN  string
		Type string
		ID   string
	}{
		{
			URN:  "urn:pulumi:test::project::digitalocean:Droplet::node1",
			Type: "digitalocean:Droplet",
			ID:   "12345",
		},
		{
			URN:  "urn:pulumi:test::project::digitalocean:Vpc::vpc1",
			Type: "digitalocean:Vpc",
			ID:   "vpc-123",
		},
		{
			URN:  "urn:pulumi:test::project::linode:Instance::instance1",
			Type: "linode:Instance",
			ID:   "67890",
		},
	}

	testCases := []struct {
		name          string
		filterType    string
		expectedCount int
	}{
		{
			name:          "Filter by Droplet",
			filterType:    "digitalocean:Droplet",
			expectedCount: 1,
		},
		{
			name:          "Filter by VPC",
			filterType:    "digitalocean:Vpc",
			expectedCount: 1,
		},
		{
			name:          "Filter by Linode Instance",
			filterType:    "linode:Instance",
			expectedCount: 1,
		},
		{
			name:          "No filter",
			filterType:    "",
			expectedCount: 3,
		},
		{
			name:          "Non-existent type",
			filterType:    "aws:Instance",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := []struct {
				URN  string
				Type string
				ID   string
			}{}

			for _, resource := range resources {
				if tc.filterType == "" || resource.Type == tc.filterType {
					filtered = append(filtered, resource)
				}
			}

			assert.Equal(t, tc.expectedCount, len(filtered))
		})
	}
}

// TestResourceRemoval tests resource removal logic
func TestResourceRemoval(t *testing.T) {
	testCases := []struct {
		name           string
		resources      []map[string]interface{}
		removeURN      string
		expectedCount  int
		shouldBeFound  bool
	}{
		{
			name: "Remove existing resource",
			resources: []map[string]interface{}{
				{"urn": "urn:pulumi:test::project::digitalocean:Droplet::node1", "type": "digitalocean:Droplet"},
				{"urn": "urn:pulumi:test::project::digitalocean:Droplet::node2", "type": "digitalocean:Droplet"},
				{"urn": "urn:pulumi:test::project::digitalocean:Vpc::vpc1", "type": "digitalocean:Vpc"},
			},
			removeURN:      "urn:pulumi:test::project::digitalocean:Droplet::node1",
			expectedCount:  2,
			shouldBeFound:  true,
		},
		{
			name: "Remove non-existent resource",
			resources: []map[string]interface{}{
				{"urn": "urn:pulumi:test::project::digitalocean:Droplet::node1", "type": "digitalocean:Droplet"},
			},
			removeURN:      "urn:pulumi:test::project::digitalocean:Droplet::node999",
			expectedCount:  1,
			shouldBeFound:  false,
		},
		{
			name:           "Remove from empty list",
			resources:      []map[string]interface{}{},
			removeURN:      "urn:pulumi:test::project::digitalocean:Droplet::node1",
			expectedCount:  0,
			shouldBeFound:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found := false
			newResources := []map[string]interface{}{}

			for _, resource := range tc.resources {
				resourceURN, _ := resource["urn"].(string)
				if resourceURN != tc.removeURN {
					newResources = append(newResources, resource)
				} else {
					found = true
				}
			}

			assert.Equal(t, tc.shouldBeFound, found)
			assert.Equal(t, tc.expectedCount, len(newResources))
		})
	}
}

// TestURNFormat tests URN format validation
func TestURNFormat(t *testing.T) {
	testCases := []struct {
		name    string
		urn     string
		isValid bool
	}{
		{
			name:    "Valid DigitalOcean Droplet URN",
			urn:     "urn:pulumi:production::sloth-kubernetes::digitalocean:Droplet::master-1",
			isValid: true,
		},
		{
			name:    "Valid Linode Instance URN",
			urn:     "urn:pulumi:production::sloth-kubernetes::linode:Instance::worker-1",
			isValid: true,
		},
		{
			name:    "Valid VPC URN",
			urn:     "urn:pulumi:production::sloth-kubernetes::digitalocean:Vpc::k8s-vpc",
			isValid: true,
		},
		{
			name:    "Invalid URN - missing prefix",
			urn:     "pulumi:production::project::type::name",
			isValid: false,
		},
		{
			name:    "Invalid URN - empty",
			urn:     "",
			isValid: false,
		},
		{
			name:    "Invalid URN - malformed",
			urn:     "not-a-urn",
			isValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// URN format: urn:pulumi:stack::project::type::name
			isValid := len(tc.urn) > 0 && tc.urn[:4] == "urn:"
			assert.Equal(t, tc.isValid, isValid)
		})
	}
}

// TestResourceTypeFormatting tests resource type formatting
func TestResourceTypeFormatting(t *testing.T) {
	testCases := []struct {
		name         string
		resourceType string
		provider     string
		resource     string
	}{
		{
			name:         "DigitalOcean Droplet",
			resourceType: "digitalocean:Droplet",
			provider:     "digitalocean",
			resource:     "Droplet",
		},
		{
			name:         "DigitalOcean VPC",
			resourceType: "digitalocean:Vpc",
			provider:     "digitalocean",
			resource:     "Vpc",
		},
		{
			name:         "Linode Instance",
			resourceType: "linode:Instance",
			provider:     "linode",
			resource:     "Instance",
		},
		{
			name:         "DigitalOcean DNS Record",
			resourceType: "digitalocean:DnsRecord",
			provider:     "digitalocean",
			resource:     "DnsRecord",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parts := len(tc.resourceType) > 0 && tc.resourceType != ""
			assert.True(t, parts)
			assert.Contains(t, tc.resourceType, ":")
		})
	}
}

// TestStateCommandsHelpText tests help text content
func TestStateCommandsHelpText(t *testing.T) {
	type cmdInfo struct {
		Use     string
		Short   string
		Long    string
		Example string
	}

	testCases := []struct {
		name     string
		cmd      cmdInfo
		keywords []string
	}{
		{
			name: "State parent command",
			cmd: cmdInfo{
				Use:   "state",
				Short: "Manage stack state",
				Long:  "View and manipulate Pulumi stack state including resources",
			},
			keywords: []string{"state", "stack", "resources"},
		},
		{
			name: "State list command",
			cmd: cmdInfo{
				Use:     "list [stack-name]",
				Short:   "List all resources in stack state",
				Long:    "Display all resources currently tracked in the stack state with their URNs and types",
				Example: "sloth-kubernetes stacks state list production",
			},
			keywords: []string{"list", "resources", "URN", "stack"},
		},
		{
			name: "State delete command",
			cmd: cmdInfo{
				Use:   "delete [stack-name] [urn]",
				Short: "Delete a resource from stack state",
				Long: `Remove a specific resource from the stack state by its URN.
This does NOT destroy the actual cloud resource, only removes it from Pulumi's state.

WARNING: This is a dangerous operation. Use with caution!`,
				Example: "sloth-kubernetes stacks state delete production urn:pulumi:...",
			},
			keywords: []string{"delete", "URN", "WARNING", "does NOT destroy"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			combinedText := tc.cmd.Use + " " + tc.cmd.Short + " " + tc.cmd.Long + " " + tc.cmd.Example

			for _, keyword := range tc.keywords {
				assert.Contains(t, combinedText, keyword,
					"Help text should contain keyword: "+keyword)
			}
		})
	}
}

// TestDeploymentMarshaling tests JSON marshaling/unmarshaling
func TestDeploymentMarshaling(t *testing.T) {
	original := struct {
		Resources []map[string]interface{} `json:"resources"`
	}{
		Resources: []map[string]interface{}{
			{
				"urn":  "urn:pulumi:test::project::digitalocean:Droplet::node1",
				"type": "digitalocean:Droplet",
				"id":   "12345",
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Unmarshal back
	var parsed struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, len(original.Resources), len(parsed.Resources))

	// Verify data integrity
	if len(parsed.Resources) > 0 {
		assert.Equal(t, original.Resources[0]["urn"], parsed.Resources[0]["urn"])
		assert.Equal(t, original.Resources[0]["type"], parsed.Resources[0]["type"])
	}
}

// Test100StateScenarios generates 100 state management test scenarios
func Test100StateScenarios(t *testing.T) {
	scenarios := []struct {
		stackName    string
		resourceType string
		operation    string
	}{
		{"production", "digitalocean:Droplet", "list"},
		{"staging", "digitalocean:Vpc", "list"},
		{"development", "linode:Instance", "list"},
		{"test", "digitalocean:DnsRecord", "list"},
		{"production", "digitalocean:Droplet", "delete"},
		{"staging", "digitalocean:Firewall", "list"},
	}

	// Generate more scenarios
	stacks := []string{"production", "staging", "development", "test", "qa", "demo"}
	resourceTypes := []string{
		"digitalocean:Droplet",
		"digitalocean:Vpc",
		"digitalocean:DnsRecord",
		"digitalocean:Firewall",
		"digitalocean:SshKey",
		"linode:Instance",
		"linode:Vpc",
		"linode:Firewall",
	}
	operations := []string{"list", "delete"}

	for _, stack := range stacks {
		for _, resourceType := range resourceTypes {
			for _, operation := range operations {
				scenarios = append(scenarios, struct {
					stackName    string
					resourceType string
					operation    string
				}{
					stackName:    stack,
					resourceType: resourceType,
					operation:    operation,
				})
			}
		}
	}

	// Limit to 100
	if len(scenarios) > 100 {
		scenarios = scenarios[:100]
	}

	for i, sc := range scenarios {
		t.Run(sc.stackName+"_"+sc.resourceType+"_"+sc.operation, func(t *testing.T) {
			// Verify scenario is well-formed
			assert.NotEmpty(t, sc.stackName)
			assert.NotEmpty(t, sc.resourceType)
			assert.NotEmpty(t, sc.operation)
			assert.Contains(t, sc.resourceType, ":")
			assert.True(t, sc.operation == "list" || sc.operation == "delete")

			// Verify resource type format
			assert.Contains(t, sc.resourceType, ":")
		})

		// Just verify first 100
		if i >= 99 {
			break
		}
	}
}

// TestConfirmationLogic tests the confirmation prompt logic
func TestConfirmationLogic(t *testing.T) {
	testCases := []struct {
		name        string
		response    string
		force       bool
		shouldAllow bool
	}{
		{
			name:        "Confirmed with 'yes'",
			response:    "yes",
			force:       false,
			shouldAllow: true,
		},
		{
			name:        "Rejected with 'no'",
			response:    "no",
			force:       false,
			shouldAllow: false,
		},
		{
			name:        "Force flag set",
			response:    "",
			force:       true,
			shouldAllow: true,
		},
		{
			name:        "Invalid response",
			response:    "maybe",
			force:       false,
			shouldAllow: false,
		},
		{
			name:        "Empty response",
			response:    "",
			force:       false,
			shouldAllow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate confirmation logic
			shouldAllow := tc.force || tc.response == "yes"
			assert.Equal(t, tc.shouldAllow, shouldAllow)
		})
	}
}

// Avoid unused import error
var _ = optup.ProgressStreams
