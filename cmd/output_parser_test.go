package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFormatNodesAsJSON(t *testing.T) {
	nodes := []NodeInfo{
		{
			Name:        "master-1",
			Provider:    "digitalocean",
			Region:      "nyc3",
			Size:        "s-2vcpu-4gb",
			PublicIP:    "1.2.3.4",
			PrivateIP:   "10.0.0.1",
			WireGuardIP: "10.8.0.1",
			Roles:       []string{"master"},
			Status:      "running",
		},
		{
			Name:        "worker-1",
			Provider:    "linode",
			Region:      "us-east",
			Size:        "g6-standard-2",
			PublicIP:    "5.6.7.8",
			PrivateIP:   "10.0.0.2",
			WireGuardIP: "10.8.0.2",
			Roles:       []string{"worker"},
			Status:      "running",
		},
	}

	result, err := FormatNodesAsJSON(nodes)
	if err != nil {
		t.Fatalf("FormatNodesAsJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed []NodeInfo
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}

	// Verify content
	if len(parsed) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(parsed))
	}

	if parsed[0].Name != "master-1" {
		t.Errorf("Expected first node name 'master-1', got '%s'", parsed[0].Name)
	}

	if parsed[1].Provider != "linode" {
		t.Errorf("Expected second node provider 'linode', got '%s'", parsed[1].Provider)
	}

	// Check formatting (indented)
	if !strings.Contains(result, "  ") {
		t.Error("Result should be indented")
	}
}

func TestFormatNodesAsJSON_Empty(t *testing.T) {
	nodes := []NodeInfo{}

	result, err := FormatNodesAsJSON(nodes)
	if err != nil {
		t.Fatalf("FormatNodesAsJSON() error = %v", err)
	}

	expected := "[]"
	if strings.TrimSpace(result) != expected {
		t.Errorf("Expected %q, got %q", expected, strings.TrimSpace(result))
	}
}

func TestFormatClusterAsJSON(t *testing.T) {
	cluster := &ClusterInfo{
		Name: "test-cluster",
		Nodes: []NodeInfo{
			{
				Name:     "master-1",
				Provider: "digitalocean",
				Roles:    []string{"master"},
			},
		},
		KubeConfig:  "/path/to/kubeconfig",
		APIEndpoint: "https://api.cluster.com",
		Status:      "ready",
	}

	result, err := FormatClusterAsJSON(cluster)
	if err != nil {
		t.Fatalf("FormatClusterAsJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed ClusterInfo
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("Result is not valid JSON: %v", err)
	}

	// Verify content
	if parsed.Name != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", parsed.Name)
	}

	if parsed.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", parsed.Status)
	}

	if len(parsed.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(parsed.Nodes))
	}
}

func TestFormatNodesAsYAML(t *testing.T) {
	nodes := []NodeInfo{
		{
			Name:        "master-1",
			Provider:    "digitalocean",
			Region:      "nyc3",
			PublicIP:    "1.2.3.4",
			WireGuardIP: "10.8.0.1",
			Roles:       []string{"master"},
			Status:      "running",
		},
	}

	result, err := FormatNodesAsYAML(nodes)
	if err != nil {
		t.Fatalf("FormatNodesAsYAML() error = %v", err)
	}

	// Verify it's valid YAML
	var parsed []NodeInfo
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("Result is not valid YAML: %v", err)
	}

	// Verify content
	if len(parsed) != 1 {
		t.Errorf("Expected 1 node, got %d", len(parsed))
	}

	if parsed[0].Name != "master-1" {
		t.Errorf("Expected node name 'master-1', got '%s'", parsed[0].Name)
	}

	// YAML should contain specific fields
	if !strings.Contains(result, "name:") {
		t.Error("YAML should contain 'name:' field")
	}

	if !strings.Contains(result, "provider:") {
		t.Error("YAML should contain 'provider:' field")
	}
}

func TestFormatNodesAsYAML_Empty(t *testing.T) {
	nodes := []NodeInfo{}

	result, err := FormatNodesAsYAML(nodes)
	if err != nil {
		t.Fatalf("FormatNodesAsYAML() error = %v", err)
	}

	// Empty YAML array should be []
	if strings.TrimSpace(result) != "[]" {
		t.Errorf("Expected '[]', got %q", strings.TrimSpace(result))
	}
}

func TestFormatClusterAsYAML(t *testing.T) {
	cluster := &ClusterInfo{
		Name: "production-cluster",
		Nodes: []NodeInfo{
			{Name: "master-1", Roles: []string{"master"}},
			{Name: "worker-1", Roles: []string{"worker"}},
		},
		Status: "ready",
	}

	result, err := FormatClusterAsYAML(cluster)
	if err != nil {
		t.Fatalf("FormatClusterAsYAML() error = %v", err)
	}

	// Verify it's valid YAML
	var parsed ClusterInfo
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("Result is not valid YAML: %v", err)
	}

	// Verify content
	if parsed.Name != "production-cluster" {
		t.Errorf("Expected cluster name 'production-cluster', got '%s'", parsed.Name)
	}

	if len(parsed.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(parsed.Nodes))
	}

	// YAML should contain expected fields
	expectedFields := []string{"name:", "nodes:", "status:"}
	for _, field := range expectedFields {
		if !strings.Contains(result, field) {
			t.Errorf("YAML should contain '%s' field", field)
		}
	}
}

func TestGetSSHKeyPath(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tests := []struct {
		name      string
		stackName string
		homeEnv   string
		want      string
	}{
		{
			name:      "standard stack",
			stackName: "dev-cluster",
			homeEnv:   "/home/user",
			want:      "/home/user/.ssh/kubernetes-clusters/dev-cluster.pem",
		},
		{
			name:      "production stack",
			stackName: "prod-cluster",
			homeEnv:   "/home/devops",
			want:      "/home/devops/.ssh/kubernetes-clusters/prod-cluster.pem",
		},
		{
			name:      "stack with dashes",
			stackName: "my-test-stack",
			homeEnv:   "/Users/developer",
			want:      "/Users/developer/.ssh/kubernetes-clusters/my-test-stack.pem",
		},
		{
			name:      "no HOME env",
			stackName: "test",
			homeEnv:   "",
			want:      "~/.ssh/kubernetes-clusters/test.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("HOME", tt.homeEnv)

			got := GetSSHKeyPath(tt.stackName)
			if got != tt.want {
				t.Errorf("GetSSHKeyPath(%q) = %q, want %q", tt.stackName, got, tt.want)
			}
		})
	}
}

func TestGetSSHKeyPath_PathComponents(t *testing.T) {
	result := GetSSHKeyPath("test-stack")

	// Should contain .ssh directory
	if !strings.Contains(result, ".ssh") {
		t.Error("Path should contain .ssh directory")
	}

	// Should contain kubernetes-clusters directory
	if !strings.Contains(result, "kubernetes-clusters") {
		t.Error("Path should contain kubernetes-clusters directory")
	}

	// Should end with stack name and .pem extension
	if !strings.HasSuffix(result, "test-stack.pem") {
		t.Error("Path should end with stackname.pem")
	}
}

func TestFormatNodesAsJSON_AllFields(t *testing.T) {
	nodes := []NodeInfo{
		{
			Name:        "full-node",
			Provider:    "digitalocean",
			Region:      "nyc3",
			Size:        "s-4vcpu-8gb",
			PublicIP:    "1.2.3.4",
			PrivateIP:   "10.0.0.1",
			WireGuardIP: "10.8.0.1",
			Roles:       []string{"master", "worker"},
			Status:      "active",
		},
	}

	result, err := FormatNodesAsJSON(nodes)
	if err != nil {
		t.Fatalf("FormatNodesAsJSON() error = %v", err)
	}

	// Verify all fields are present
	requiredFields := []string{
		"name", "provider", "region", "size",
		"publicIP", "privateIP", "wireGuardIP",
		"roles", "status",
	}

	for _, field := range requiredFields {
		if !strings.Contains(result, field) {
			t.Errorf("JSON should contain field %q", field)
		}
	}
}

func TestFormatClusterAsJSON_WithOptionalFields(t *testing.T) {
	tests := []struct {
		name    string
		cluster *ClusterInfo
		wantFields []string
	}{
		{
			name: "with kubeconfig",
			cluster: &ClusterInfo{
				Name:       "test",
				KubeConfig: "/path/to/config",
				Status:     "ready",
			},
			wantFields: []string{"name", "kubeConfig", "status"},
		},
		{
			name: "with api endpoint",
			cluster: &ClusterInfo{
				Name:        "test",
				APIEndpoint: "https://api.example.com",
				Status:      "ready",
			},
			wantFields: []string{"name", "apiEndpoint", "status"},
		},
		{
			name: "minimal",
			cluster: &ClusterInfo{
				Name:   "test",
				Status: "unknown",
			},
			wantFields: []string{"name", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatClusterAsJSON(tt.cluster)
			if err != nil {
				t.Fatalf("FormatClusterAsJSON() error = %v", err)
			}

			for _, field := range tt.wantFields {
				if !strings.Contains(result, field) {
					t.Errorf("JSON should contain field %q", field)
				}
			}
		})
	}
}

func TestNodeInfo_JSONRoundTrip(t *testing.T) {
	original := NodeInfo{
		Name:        "test-node",
		Provider:    "digitalocean",
		Region:      "nyc3",
		Size:        "s-2vcpu-4gb",
		PublicIP:    "1.2.3.4",
		PrivateIP:   "10.0.0.1",
		WireGuardIP: "10.8.0.1",
		Roles:       []string{"master", "etcd"},
		Status:      "running",
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal back
	var parsed NodeInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare
	if parsed.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", parsed.Name, original.Name)
	}
	if parsed.Provider != original.Provider {
		t.Errorf("Provider mismatch: got %q, want %q", parsed.Provider, original.Provider)
	}
	if len(parsed.Roles) != len(original.Roles) {
		t.Errorf("Roles length mismatch: got %d, want %d", len(parsed.Roles), len(original.Roles))
	}
}

func TestClusterInfo_YAMLRoundTrip(t *testing.T) {
	original := ClusterInfo{
		Name: "test-cluster",
		Nodes: []NodeInfo{
			{Name: "node-1", Roles: []string{"master"}},
		},
		KubeConfig:  "/path/to/config",
		APIEndpoint: "https://api.test.com",
		Status:      "ready",
	}

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal back
	var parsed ClusterInfo
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare
	if parsed.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", parsed.Name, original.Name)
	}
	if len(parsed.Nodes) != len(original.Nodes) {
		t.Errorf("Nodes length mismatch: got %d, want %d", len(parsed.Nodes), len(original.Nodes))
	}
	if parsed.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", parsed.Status, original.Status)
	}
}
