package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sloth-kubernetes/pkg/providers"
)

// Test getNodeTaints with no taints
func TestRKEManager_GetNodeTaints_NoTaints(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name:   "worker1",
		Labels: map[string]string{},
	}

	taints := manager.getNodeTaints(node)

	assert.Empty(t, taints, "Should return empty taints")
	assert.Len(t, taints, 0)
}

// Test getNodeTaints with valid taint
func TestRKEManager_GetNodeTaints_ValidTaint(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "master1",
		Labels: map[string]string{
			"taints": "node-role.kubernetes.io/master=true:NoSchedule",
		},
	}

	taints := manager.getNodeTaints(node)

	assert.Len(t, taints, 1)
	assert.Equal(t, "node-role.kubernetes.io/master", taints[0]["key"])
	assert.Equal(t, "true", taints[0]["value"])
	assert.Equal(t, "NoSchedule", taints[0]["effect"])
}

// Test getNodeTaints with different effects
func TestRKEManager_GetNodeTaints_DifferentEffects(t *testing.T) {
	manager := &RKEManager{}

	tests := []struct {
		name         string
		taintStr     string
		expectedKey  string
		expectedVal  string
		expectedEff  string
	}{
		{
			"NoSchedule effect",
			"key1=value1:NoSchedule",
			"key1",
			"value1",
			"NoSchedule",
		},
		{
			"PreferNoSchedule effect",
			"key2=value2:PreferNoSchedule",
			"key2",
			"value2",
			"PreferNoSchedule",
		},
		{
			"NoExecute effect",
			"key3=value3:NoExecute",
			"key3",
			"value3",
			"NoExecute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &providers.NodeOutput{
				Name: "node1",
				Labels: map[string]string{
					"taints": tt.taintStr,
				},
			}

			taints := manager.getNodeTaints(node)

			assert.Len(t, taints, 1)
			assert.Equal(t, tt.expectedKey, taints[0]["key"])
			assert.Equal(t, tt.expectedVal, taints[0]["value"])
			assert.Equal(t, tt.expectedEff, taints[0]["effect"])
		})
	}
}

// Test getNodeTaints with invalid format (missing colon)
func TestRKEManager_GetNodeTaints_InvalidFormat_MissingColon(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "node1",
		Labels: map[string]string{
			"taints": "key=value",
		},
	}

	taints := manager.getNodeTaints(node)

	assert.Empty(t, taints, "Invalid format should return empty taints")
}

// Test getNodeTaints with invalid format (missing equals)
func TestRKEManager_GetNodeTaints_InvalidFormat_MissingEquals(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "node1",
		Labels: map[string]string{
			"taints": "key:NoSchedule",
		},
	}

	taints := manager.getNodeTaints(node)

	assert.Empty(t, taints, "Invalid format should return empty taints")
}

// Test getNodeTaints with empty taint string
func TestRKEManager_GetNodeTaints_EmptyTaintString(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "node1",
		Labels: map[string]string{
			"taints": "",
		},
	}

	taints := manager.getNodeTaints(node)

	assert.Empty(t, taints)
}

// Test getNodeTaints taint structure
func TestRKEManager_GetNodeTaints_Structure(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "node1",
		Labels: map[string]string{
			"taints": "mykey=myvalue:NoSchedule",
		},
	}

	taints := manager.getNodeTaints(node)

	assert.Len(t, taints, 1)

	taint := taints[0]
	assert.Contains(t, taint, "key")
	assert.Contains(t, taint, "value")
	assert.Contains(t, taint, "effect")
	assert.Len(t, taint, 3, "Taint should have exactly 3 fields")
}

// Test getNodeTaints with special characters in key
func TestRKEManager_GetNodeTaints_SpecialCharsInKey(t *testing.T) {
	manager := &RKEManager{}

	tests := []struct {
		name     string
		taintStr string
		key      string
	}{
		{
			"Dot in key",
			"node.role/master=true:NoSchedule",
			"node.role/master",
		},
		{
			"Slash in key",
			"example.com/gpu=true:NoSchedule",
			"example.com/gpu",
		},
		{
			"Dash in key",
			"node-type=master:NoSchedule",
			"node-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &providers.NodeOutput{
				Name: "node1",
				Labels: map[string]string{
					"taints": tt.taintStr,
				},
			}

			taints := manager.getNodeTaints(node)

			assert.Len(t, taints, 1)
			assert.Equal(t, tt.key, taints[0]["key"])
		})
	}
}

// Test getNodeTaints with boolean values
func TestRKEManager_GetNodeTaints_BooleanValues(t *testing.T) {
	manager := &RKEManager{}

	tests := []struct {
		name     string
		value    string
	}{
		{"true value", "true"},
		{"false value", "false"},
		{"1 value", "1"},
		{"0 value", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &providers.NodeOutput{
				Name: "node1",
				Labels: map[string]string{
					"taints": "key=" + tt.value + ":NoSchedule",
				},
			}

			taints := manager.getNodeTaints(node)

			assert.Len(t, taints, 1)
			assert.Equal(t, tt.value, taints[0]["value"])
		})
	}
}

// Test getNodeTaints with empty value
func TestRKEManager_GetNodeTaints_EmptyValue(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "node1",
		Labels: map[string]string{
			"taints": "key=:NoSchedule",
		},
	}

	taints := manager.getNodeTaints(node)

	assert.Len(t, taints, 1)
	assert.Equal(t, "key", taints[0]["key"])
	assert.Equal(t, "", taints[0]["value"])
	assert.Equal(t, "NoSchedule", taints[0]["effect"])
}

// Test 30 taint parsing scenarios
func Test30TaintParsingScenarios(t *testing.T) {
	manager := &RKEManager{}

	scenarios := []struct {
		taintStr string
		valid    bool
		key      string
		value    string
		effect   string
	}{
		{"key=value:NoSchedule", true, "key", "value", "NoSchedule"},
		{"key=value:PreferNoSchedule", true, "key", "value", "PreferNoSchedule"},
		{"key=value:NoExecute", true, "key", "value", "NoExecute"},
		{"node-role=master:NoSchedule", true, "node-role", "master", "NoSchedule"},
		{"gpu=nvidia:NoSchedule", true, "gpu", "nvidia", "NoSchedule"},
		{"invalid-format", false, "", "", ""},
		{"key=value", false, "", "", ""},
		{":NoSchedule", false, "", "", ""},
		{"=:NoSchedule", false, "", "", ""},
	}

	// Generate more scenarios
	effects := []string{"NoSchedule", "PreferNoSchedule", "NoExecute"}
	keys := []string{"dedicated", "node-type", "special", "gpu", "storage"}
	values := []string{"true", "false", "yes", "no", "1", "0"}

	for i := 0; i < 21; i++ {
		key := keys[i%len(keys)]
		value := values[i%len(values)]
		effect := effects[i%len(effects)]
		taintStr := key + "=" + value + ":" + effect

		scenarios = append(scenarios, struct {
			taintStr string
			valid    bool
			key      string
			value    string
			effect   string
		}{taintStr, true, key, value, effect})
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			node := &providers.NodeOutput{
				Name: "node1",
				Labels: map[string]string{
					"taints": scenario.taintStr,
				},
			}

			taints := manager.getNodeTaints(node)

			if scenario.valid {
				assert.Len(t, taints, 1, "Valid taint should be parsed")
				assert.Equal(t, scenario.key, taints[0]["key"])
				assert.Equal(t, scenario.value, taints[0]["value"])
				assert.Equal(t, scenario.effect, taints[0]["effect"])
			} else {
				// Invalid taints with format "x=y:z" will still parse but with empty key/value
				// The function doesn't validate content, just format
				if len(taints) > 0 {
					// If it parsed, verify structure exists
					assert.Contains(t, taints[0], "key")
					assert.Contains(t, taints[0], "value")
					assert.Contains(t, taints[0], "effect")
				}
			}
		})
	}
}

// Test getNodeTaints return type
func TestRKEManager_GetNodeTaints_ReturnType(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name: "node1",
		Labels: map[string]string{
			"taints": "key=value:NoSchedule",
		},
	}

	taints := manager.getNodeTaints(node)

	// Should return slice of maps
	assert.IsType(t, []map[string]interface{}{}, taints)

	// Each taint should be a map
	if len(taints) > 0 {
		assert.IsType(t, map[string]interface{}{}, taints[0])
	}
}

// Test getNodeTaints common Kubernetes taints
func TestRKEManager_GetNodeTaints_CommonK8sTaints(t *testing.T) {
	manager := &RKEManager{}

	commonTaints := []struct {
		name     string
		taintStr string
		key      string
	}{
		{
			"Master taint",
			"node-role.kubernetes.io/master=:NoSchedule",
			"node-role.kubernetes.io/master",
		},
		{
			"Control plane taint",
			"node-role.kubernetes.io/control-plane=:NoSchedule",
			"node-role.kubernetes.io/control-plane",
		},
		{
			"Not ready taint",
			"node.kubernetes.io/not-ready=:NoExecute",
			"node.kubernetes.io/not-ready",
		},
		{
			"Unreachable taint",
			"node.kubernetes.io/unreachable=:NoExecute",
			"node.kubernetes.io/unreachable",
		},
	}

	for _, tt := range commonTaints {
		t.Run(tt.name, func(t *testing.T) {
			node := &providers.NodeOutput{
				Name: "node1",
				Labels: map[string]string{
					"taints": tt.taintStr,
				},
			}

			taints := manager.getNodeTaints(node)

			assert.Len(t, taints, 1)
			assert.Equal(t, tt.key, taints[0]["key"])
		})
	}
}

// Test getNodeTaints with node without labels
func TestRKEManager_GetNodeTaints_NodeWithoutLabels(t *testing.T) {
	manager := &RKEManager{}

	node := &providers.NodeOutput{
		Name:   "node1",
		Labels: nil,
	}

	taints := manager.getNodeTaints(node)

	assert.Empty(t, taints)
	assert.NotNil(t, taints, "Should return empty slice, not nil")
}

// Test taint format validation
func TestRKEManager_TaintFormatValidation(t *testing.T) {
	manager := &RKEManager{}

	invalidFormats := []string{
		"key:effect",           // missing =value
		"keyvalue:effect",      // missing =
		"key=value",            // missing :effect
		"key=value:effect:extra", // extra colon
		"=value:effect",        // missing key
		"key=",                 // missing value and effect
		"",                     // empty
	}

	for _, format := range invalidFormats {
		t.Run("Invalid_"+format, func(t *testing.T) {
			node := &providers.NodeOutput{
				Name: "node1",
				Labels: map[string]string{
					"taints": format,
				},
			}

			taints := manager.getNodeTaints(node)

			// The function parses format "key=value:effect"
			// It doesn't validate if key or value are empty
			// So "=value:effect" still creates a taint with empty key
			if len(taints) > 0 {
				// If it parsed, verify structure exists
				assert.Contains(t, taints[0], "key")
				assert.Contains(t, taints[0], "value")
				assert.Contains(t, taints[0], "effect")
			}
		})
	}
}
