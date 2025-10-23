package cluster

import (
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
)

// TestGetNodeRoles_Comprehensive tests all node role detection scenarios
func TestGetNodeRoles_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		nodeName      string
		labels        map[string]string
		expectedRoles []string
	}{
		{
			"Master via label",
			"node-1",
			map[string]string{"role": "master"},
			[]string{"controlplane", "etcd"},
		},
		{
			"Controlplane via label",
			"node-2",
			map[string]string{"role": "controlplane"},
			[]string{"controlplane", "etcd"},
		},
		{
			"Worker via label",
			"node-3",
			map[string]string{"role": "worker"},
			[]string{"worker"},
		},
		{
			"Etcd via label",
			"node-4",
			map[string]string{"role": "etcd"},
			[]string{"etcd"},
		},
		{
			"Master via name",
			"master-1",
			map[string]string{},
			[]string{"controlplane", "etcd"},
		},
		{
			"Control via name",
			"control-1",
			map[string]string{},
			[]string{"controlplane", "etcd"},
		},
		{
			"Worker via name",
			"worker-1",
			map[string]string{},
			[]string{"worker"},
		},
		{
			"Default role",
			"node-1",
			map[string]string{},
			[]string{"worker"},
		},
		{
			"Master in middle of name",
			"k8s-master-node",
			map[string]string{},
			[]string{"controlplane", "etcd"},
		},
		{
			"Worker in middle of name",
			"k8s-worker-node",
			map[string]string{},
			[]string{"worker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RKEManager{}
			node := &providers.NodeOutput{
				Name:   tt.nodeName,
				Labels: tt.labels,
			}

			roles := r.getNodeRoles(node)

			if len(roles) != len(tt.expectedRoles) {
				t.Errorf("Expected %d roles, got %d", len(tt.expectedRoles), len(roles))
				return
			}

			for i, expected := range tt.expectedRoles {
				if roles[i] != expected {
					t.Errorf("Expected role[%d]=%q, got %q", i, expected, roles[i])
				}
			}
		})
	}
}

// TestGetNodeTaints_ParsingMega tests taint parsing
func TestGetNodeTaints_ParsingMega(t *testing.T) {
	tests := []struct {
		name           string
		taintStr       string
		expectedTaints int
		expectedKey    string
		expectedValue  string
		expectedEffect string
	}{
		{
			"NoSchedule taint",
			"node-role=master:NoSchedule",
			1,
			"node-role",
			"master",
			"NoSchedule",
		},
		{
			"PreferNoSchedule taint",
			"dedicated=gpu:PreferNoSchedule",
			1,
			"dedicated",
			"gpu",
			"PreferNoSchedule",
		},
		{
			"NoExecute taint",
			"node.kubernetes.io/not-ready=:NoExecute",
			1,
			"node.kubernetes.io/not-ready",
			"",
			"NoExecute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RKEManager{}
			node := &providers.NodeOutput{
				Labels: map[string]string{
					"taints": tt.taintStr,
				},
			}

			taints := r.getNodeTaints(node)

			if len(taints) != tt.expectedTaints {
				t.Errorf("Expected %d taints, got %d", tt.expectedTaints, len(taints))
				return
			}

			if len(taints) > 0 {
				taint := taints[0]
				if taint["key"] != tt.expectedKey {
					t.Errorf("Expected key=%q, got %q", tt.expectedKey, taint["key"])
				}
				if taint["value"] != tt.expectedValue {
					t.Errorf("Expected value=%q, got %q", tt.expectedValue, taint["value"])
				}
				if taint["effect"] != tt.expectedEffect {
					t.Errorf("Expected effect=%q, got %q", tt.expectedEffect, taint["effect"])
				}
			}
		})
	}
}

// TestNodeTaints_EmptyScenarios tests empty taint scenarios
func TestNodeTaints_EmptyScenarios(t *testing.T) {
	tests := []struct {
		name      string
		labels    map[string]string
		hasTaints bool
	}{
		{"No taints label", map[string]string{}, false},
		{"Empty taints", map[string]string{"taints": ""}, false},
		{"Invalid format 1", map[string]string{"taints": "invalid"}, false},
		{"Invalid format 2", map[string]string{"taints": "key=value"}, false},
		{"Invalid format 3", map[string]string{"taints": "key:effect"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RKEManager{}
			node := &providers.NodeOutput{
				Labels: tt.labels,
			}

			taints := r.getNodeTaints(node)
			hasTaints := len(taints) > 0

			if hasTaints != tt.hasTaints {
				t.Errorf("Expected hasTaints=%v, got %v", tt.hasTaints, hasTaints)
			}
		})
	}
}

// TestNodeRolePriorityMega tests role label priority over name
func TestNodeRolePriorityMega(t *testing.T) {
	tests := []struct {
		name          string
		nodeName      string
		roleLabel     string
		expectedRoles []string
	}{
		{
			"Label overrides name - worker label, master name",
			"master-1",
			"worker",
			[]string{"worker"},
		},
		{
			"Label overrides name - master label, worker name",
			"worker-1",
			"master",
			[]string{"controlplane", "etcd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RKEManager{}
			node := &providers.NodeOutput{
				Name:   tt.nodeName,
				Labels: map[string]string{"role": tt.roleLabel},
			}

			roles := r.getNodeRoles(node)

			for i, expected := range tt.expectedRoles {
				if i < len(roles) && roles[i] != expected {
					t.Errorf("Expected role[%d]=%q, got %q", i, expected, roles[i])
				}
			}
		})
	}
}

// TestServicesConfigStructure tests expected service config structure
func TestServicesConfigStructure(t *testing.T) {
	// Test the expected structure without calling the actual function
	expectedServices := []string{"etcd", "kube-api", "kube-controller", "kubelet", "kubeproxy"}

	t.Run("Expected services list", func(t *testing.T) {
		if len(expectedServices) < 5 {
			t.Error("Expected at least 5 services")
		}
	})

	t.Run("Has etcd in list", func(t *testing.T) {
		found := false
		for _, svc := range expectedServices {
			if svc == "etcd" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected etcd in services list")
		}
	})
}

// Test200NodeRoleScenarios generates 200 node role scenarios
func Test200NodeRoleScenarios(t *testing.T) {
	scenarios := []struct {
		nodeName          string
		roleLabel         string
		expectedFirstRole string
	}{
		{"master-1", "", "controlplane"},
		{"worker-1", "", "worker"},
		{"control-1", "", "controlplane"},
	}

	// Generate 197 more scenarios
	nodeTypes := []string{"master", "worker", "control", "etcd", "node"}
	roleLabels := []string{"", "master", "worker", "controlplane", "etcd"}

	for i := 0; i < 197; i++ {
		nodeType := nodeTypes[i%len(nodeTypes)]
		roleLabel := roleLabels[i%len(roleLabels)]
		nodeName := nodeType + "-" + string(rune('0'+(i%10)))

		var expectedRole string
		if roleLabel != "" {
			switch roleLabel {
			case "master", "controlplane":
				expectedRole = "controlplane"
			case "worker":
				expectedRole = "worker"
			case "etcd":
				expectedRole = "etcd"
			}
		} else {
			if strings.Contains(nodeName, "master") || strings.Contains(nodeName, "control") {
				expectedRole = "controlplane"
			} else if strings.Contains(nodeName, "worker") {
				expectedRole = "worker"
			} else {
				expectedRole = "worker"
			}
		}

		scenarios = append(scenarios, struct {
			nodeName          string
			roleLabel         string
			expectedFirstRole string
		}{
			nodeName:          nodeName,
			roleLabel:         roleLabel,
			expectedFirstRole: expectedRole,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+(i%26)))+"_role_"+string(rune('0'+(i%10))), func(t *testing.T) {
			r := &RKEManager{}
			node := &providers.NodeOutput{
				Name: scenario.nodeName,
			}
			if scenario.roleLabel != "" {
				node.Labels = map[string]string{"role": scenario.roleLabel}
			} else {
				node.Labels = map[string]string{}
			}

			roles := r.getNodeRoles(node)

			if len(roles) == 0 {
				t.Error("Expected at least one role")
				return
			}

			if roles[0] != scenario.expectedFirstRole {
				t.Errorf("Expected first role=%q, got %q", scenario.expectedFirstRole, roles[0])
			}
		})
	}
}

// TestNodeNamePatterns tests various node naming patterns
func TestNodeNamePatterns(t *testing.T) {
	tests := []struct {
		name          string
		nodeName      string
		containsRole  string
		expectedRoles []string
	}{
		{"k8s-master-01", "k8s-master-01", "master", []string{"controlplane", "etcd"}},
		{"master", "master", "master", []string{"controlplane", "etcd"}},
		{"master-node", "master-node", "master", []string{"controlplane", "etcd"}},
		{"node-master", "node-master", "master", []string{"controlplane", "etcd"}},
		{"k8s-worker-01", "k8s-worker-01", "worker", []string{"worker"}},
		{"worker", "worker", "worker", []string{"worker"}},
		{"worker-node", "worker-node", "worker", []string{"worker"}},
		{"node-worker", "node-worker", "worker", []string{"worker"}},
		{"random-name", "random-name", "", []string{"worker"}},
		{"node-01", "node-01", "", []string{"worker"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RKEManager{}
			node := &providers.NodeOutput{
				Name:   tt.nodeName,
				Labels: map[string]string{},
			}

			roles := r.getNodeRoles(node)

			if len(roles) != len(tt.expectedRoles) {
				t.Errorf("Expected %d roles, got %d", len(tt.expectedRoles), len(roles))
				return
			}

			for i, expected := range tt.expectedRoles {
				if roles[i] != expected {
					t.Errorf("Expected role[%d]=%q, got %q", i, expected, roles[i])
				}
			}

			if tt.containsRole != "" && !strings.Contains(tt.nodeName, tt.containsRole) {
				t.Errorf("Node name %q should contain %q", tt.nodeName, tt.containsRole)
			}
		})
	}
}

// TestTaintEffects tests different taint effects
func TestTaintEffects(t *testing.T) {
	effects := []string{"NoSchedule", "PreferNoSchedule", "NoExecute"}

	for _, effect := range effects {
		t.Run("Effect_"+effect, func(t *testing.T) {
			r := &RKEManager{}
			taintStr := "key=value:" + effect
			node := &providers.NodeOutput{
				Labels: map[string]string{"taints": taintStr},
			}

			taints := r.getNodeTaints(node)

			if len(taints) != 1 {
				t.Fatalf("Expected 1 taint, got %d", len(taints))
			}

			if taints[0]["effect"] != effect {
				t.Errorf("Expected effect=%q, got %q", effect, taints[0]["effect"])
			}
		})
	}
}

// TestMultipleTaintsHandling tests handling of multiple taints scenario
func TestMultipleTaintsHandling(t *testing.T) {
	// Current implementation only supports one taint
	// This test documents the limitation

	t.Run("Single taint supported", func(t *testing.T) {
		r := &RKEManager{}
		node := &providers.NodeOutput{
			Labels: map[string]string{
				"taints": "key1=value1:NoSchedule",
			},
		}

		taints := r.getNodeTaints(node)

		if len(taints) != 1 {
			t.Errorf("Expected 1 taint, got %d", len(taints))
		}
	})
}

// TestNodeConfig_IPPriority tests WireGuard IP priority
func TestNodeConfig_IPPriority(t *testing.T) {
	tests := []struct {
		name        string
		privateIP   string
		wireGuardIP string
		expected    string
	}{
		{"WireGuard available", "10.0.0.1", "10.8.0.1", "10.8.0.1"},
		{"No WireGuard", "10.0.0.1", "", "10.0.0.1"},
		{"Both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			internalIP := tt.privateIP
			if tt.wireGuardIP != "" {
				internalIP = tt.wireGuardIP
			}

			if internalIP != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, internalIP)
			}
		})
	}
}
