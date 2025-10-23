package health

import (
	"testing"
	"time"

	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
)

func TestValidationResult_Struct(t *testing.T) {
	result := &ValidationResult{
		Name:      "Test Validation",
		Success:   true,
		Message:   "Test message",
		Error:     nil,
		Timestamp: time.Now(),
	}

	if result.Name != "Test Validation" {
		t.Errorf("Expected name 'Test Validation', got '%s'", result.Name)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", result.Message)
	}
}

func TestValidateNodeCount(t *testing.T) {
	// Create mock validator (without Pulumi context for now)
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	tests := []struct {
		name        string
		nodes       []*providers.NodeOutput
		wantSuccess bool
		wantMessage string
	}{
		{
			name:        "Minimum nodes (3)",
			nodes:       makeNodes(3),
			wantSuccess: true,
			wantMessage: "Found 3 nodes (minimum 3 required)",
		},
		{
			name:        "More than minimum (5)",
			nodes:       makeNodes(5),
			wantSuccess: true,
			wantMessage: "Found 5 nodes (minimum 3 required)",
		},
		{
			name:        "Less than minimum (2)",
			nodes:       makeNodes(2),
			wantSuccess: false,
			wantMessage: "Only 2 nodes found (minimum 3 required)",
		},
		{
			name:        "Only one node",
			nodes:       makeNodes(1),
			wantSuccess: false,
			wantMessage: "Only 1 nodes found (minimum 3 required)",
		},
		{
			name:        "No nodes",
			nodes:       makeNodes(0),
			wantSuccess: false,
			wantMessage: "Only 0 nodes found (minimum 3 required)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.validateNodeCount(tt.nodes)

			if result.Success != tt.wantSuccess {
				t.Errorf("validateNodeCount() Success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if result.Message != tt.wantMessage {
				t.Errorf("validateNodeCount() Message = %q, want %q", result.Message, tt.wantMessage)
			}

			if result.Name != "Node Count" {
				t.Errorf("validateNodeCount() Name = %q, want 'Node Count'", result.Name)
			}
		})
	}
}

func TestValidateMasterNodes(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	tests := []struct {
		name        string
		nodes       []*providers.NodeOutput
		wantSuccess bool
		masterCount int
	}{
		{
			name:        "One master (valid)",
			nodes:       makeNodesWithRole(1, "master"),
			wantSuccess: true,
			masterCount: 1,
		},
		{
			name:        "Three masters (valid HA)",
			nodes:       makeNodesWithRole(3, "master"),
			wantSuccess: true,
			masterCount: 3,
		},
		{
			name:        "Five masters (valid HA)",
			nodes:       makeNodesWithRole(5, "master"),
			wantSuccess: true,
			masterCount: 5,
		},
		{
			name:        "Two masters (invalid - even)",
			nodes:       makeNodesWithRole(2, "master"),
			wantSuccess: false,
			masterCount: 2,
		},
		{
			name:        "Four masters (invalid - even)",
			nodes:       makeNodesWithRole(4, "master"),
			wantSuccess: false,
			masterCount: 4,
		},
		{
			name:        "No masters",
			nodes:       makeNodesWithRole(0, "master"),
			wantSuccess: false,
			masterCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.validateMasterNodes(tt.nodes)

			if result.Success != tt.wantSuccess {
				t.Errorf("validateMasterNodes() Success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if result.Name != "Master Nodes" {
				t.Errorf("validateMasterNodes() Name = %q, want 'Master Nodes'", result.Name)
			}
		})
	}
}

func TestValidateMasterNodes_ControlplaneRole(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	// Test controlplane role (synonym for master)
	nodes := makeNodesWithRole(3, "controlplane")
	result := v.validateMasterNodes(nodes)

	if !result.Success {
		t.Error("validateMasterNodes() should accept 'controlplane' role")
	}
}

func TestValidateWorkerNodes(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	tests := []struct {
		name        string
		nodes       []*providers.NodeOutput
		wantSuccess bool
	}{
		{
			name:        "One worker",
			nodes:       makeNodesWithRole(1, "worker"),
			wantSuccess: true,
		},
		{
			name:        "Multiple workers",
			nodes:       makeNodesWithRole(5, "worker"),
			wantSuccess: true,
		},
		{
			name:        "No workers",
			nodes:       makeNodesWithRole(0, "worker"),
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.validateWorkerNodes(tt.nodes)

			if result.Success != tt.wantSuccess {
				t.Errorf("validateWorkerNodes() Success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if result.Name != "Worker Nodes" {
				t.Errorf("validateWorkerNodes() Name = %q, want 'Worker Nodes'", result.Name)
			}
		})
	}
}

func TestValidateNodeConnectivity(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateNodeConnectivity(makeNodes(3))

	if !result.Success {
		t.Error("validateNodeConnectivity() should return success (simplified implementation)")
	}

	if result.Name != "Node Connectivity" {
		t.Errorf("validateNodeConnectivity() Name = %q, want 'Node Connectivity'", result.Name)
	}

	if result.Message != "All nodes are reachable" {
		t.Errorf("validateNodeConnectivity() Message = %q", result.Message)
	}
}

func TestValidateDockerInstalled(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateDockerInstalled(makeNodes(3))

	if !result.Success {
		t.Error("validateDockerInstalled() should return success")
	}

	if result.Name != "Docker Installation" {
		t.Errorf("validateDockerInstalled() Name = %q", result.Name)
	}
}

func TestValidateSwapDisabled(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateSwapDisabled(makeNodes(3))

	if !result.Success {
		t.Error("validateSwapDisabled() should return success")
	}

	if result.Name != "Swap Disabled" {
		t.Errorf("validateSwapDisabled() Name = %q", result.Name)
	}
}

func TestValidateKernelModules(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateKernelModules(makeNodes(3))

	if !result.Success {
		t.Error("validateKernelModules() should return success")
	}

	if result.Name != "Kernel Modules" {
		t.Errorf("validateKernelModules() Name = %q", result.Name)
	}
}

func TestValidatePorts(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validatePorts(makeNodes(3))

	if !result.Success {
		t.Error("validatePorts() should return success")
	}

	if result.Name != "Required Ports" {
		t.Errorf("validatePorts() Name = %q", result.Name)
	}
}

func TestValidateDiskSpace(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateDiskSpace(makeNodes(3))

	if !result.Success {
		t.Error("validateDiskSpace() should return success")
	}

	if result.Name != "Disk Space" {
		t.Errorf("validateDiskSpace() Name = %q", result.Name)
	}
}

func TestValidateMemory(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateMemory(makeNodes(3))

	if !result.Success {
		t.Error("validateMemory() should return success")
	}

	if result.Name != "Memory" {
		t.Errorf("validateMemory() Name = %q", result.Name)
	}
}

func TestValidateKubernetesRunning(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateKubernetesRunning(makeNodes(3))

	if !result.Success {
		t.Error("validateKubernetesRunning() should return success")
	}

	if result.Name != "Kubernetes Cluster" {
		t.Errorf("validateKubernetesRunning() Name = %q", result.Name)
	}
}

func TestValidateKubernetesPods(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateKubernetesPods(makeNodes(3))

	if !result.Success {
		t.Error("validateKubernetesPods() should return success")
	}

	if result.Name != "Kubernetes Pods" {
		t.Errorf("validateKubernetesPods() Name = %q", result.Name)
	}
}

func TestValidateHelmInstalled(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateHelmInstalled(makeNodes(3))

	if !result.Success {
		t.Error("validateHelmInstalled() should return success")
	}

	if result.Name != "Helm" {
		t.Errorf("validateHelmInstalled() Name = %q", result.Name)
	}
}

func TestValidateIngressNamespace(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateIngressNamespace(makeNodes(3))

	if !result.Success {
		t.Error("validateIngressNamespace() should return success")
	}

	if result.Name != "Ingress Namespace" {
		t.Errorf("validateIngressNamespace() Name = %q", result.Name)
	}
}

func TestValidateLoadBalancerSupport(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateLoadBalancerSupport(makeNodes(3))

	if !result.Success {
		t.Error("validateLoadBalancerSupport() should return success")
	}

	if result.Name != "Load Balancer Support" {
		t.Errorf("validateLoadBalancerSupport() Name = %q", result.Name)
	}
}

func TestValidateWireGuardInstalled(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateWireGuardInstalled(makeNodes(3))

	if !result.Success {
		t.Error("validateWireGuardInstalled() should return success")
	}

	if result.Name != "WireGuard Installation" {
		t.Errorf("validateWireGuardInstalled() Name = %q", result.Name)
	}
}

func TestValidateKernelSupport(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateKernelSupport(makeNodes(3))

	if !result.Success {
		t.Error("validateKernelSupport() should return success")
	}

	if result.Name != "Kernel Support" {
		t.Errorf("validateKernelSupport() Name = %q", result.Name)
	}
}

func TestValidateNetworkInterfaces(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateNetworkInterfaces(makeNodes(3))

	if !result.Success {
		t.Error("validateNetworkInterfaces() should return success")
	}

	if result.Name != "Network Interfaces" {
		t.Errorf("validateNetworkInterfaces() Name = %q", result.Name)
	}
}

func TestValidateIPForwarding(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	result := v.validateIPForwarding(makeNodes(3))

	if !result.Success {
		t.Error("validateIPForwarding() should return success")
	}

	if result.Name != "IP Forwarding" {
		t.Errorf("validateIPForwarding() Name = %q", result.Name)
	}
}

func TestGetResults(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	// Add some results
	v.results["test1"] = &ValidationResult{Name: "test1", Success: true}
	v.results["test2"] = &ValidationResult{Name: "test2", Success: false}

	results := v.GetResults()

	if len(results) != 2 {
		t.Errorf("GetResults() returned %d results, want 2", len(results))
	}

	if results["test1"].Name != "test1" {
		t.Error("GetResults() should return test1")
	}

	if results["test2"].Name != "test2" {
		t.Error("GetResults() should return test2")
	}
}

func TestValidationResult_Timestamp(t *testing.T) {
	v := &PrerequisiteValidator{
		results: make(map[string]*ValidationResult),
	}

	beforeTime := time.Now()
	result := v.validateNodeCount(makeNodes(3))
	afterTime := time.Now()

	if result.Timestamp.Before(beforeTime) || result.Timestamp.After(afterTime) {
		t.Error("Timestamp should be set to current time")
	}
}

// Helper functions

func makeNodes(count int) []*providers.NodeOutput {
	nodes := make([]*providers.NodeOutput, count)
	for i := 0; i < count; i++ {
		nodes[i] = &providers.NodeOutput{
			Name:   "node-" + string(rune(i+'0')),
			Labels: make(map[string]string),
		}
	}
	return nodes
}

func makeNodesWithRole(count int, role string) []*providers.NodeOutput {
	nodes := makeNodes(count)
	for i := 0; i < count; i++ {
		nodes[i].Labels["role"] = role
	}
	return nodes
}

func TestMakeNodes_Helper(t *testing.T) {
	nodes := makeNodes(5)
	if len(nodes) != 5 {
		t.Errorf("makeNodes(5) created %d nodes, want 5", len(nodes))
	}

	for i, node := range nodes {
		if node == nil {
			t.Errorf("Node %d is nil", i)
		}
		if node.Labels == nil {
			t.Errorf("Node %d has nil labels", i)
		}
	}
}

func TestMakeNodesWithRole_Helper(t *testing.T) {
	nodes := makeNodesWithRole(3, "master")

	if len(nodes) != 3 {
		t.Errorf("makeNodesWithRole(3, 'master') created %d nodes, want 3", len(nodes))
	}

	for i, node := range nodes {
		if role, ok := node.Labels["role"]; !ok || role != "master" {
			t.Errorf("Node %d role = %q, want 'master'", i, role)
		}
	}
}
