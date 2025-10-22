package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("test.yaml")
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
	if loader.configPath != "test.yaml" {
		t.Errorf("expected config path 'test.yaml', got '%s'", loader.configPath)
	}
	if loader.overrides == nil {
		t.Error("overrides map not initialized")
	}
	if loader.validators == nil {
		t.Error("validators slice not initialized")
	}
}

func TestLoader_Load(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		fileExt       string
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid YAML",
			fileContent: `
metadata:
  name: test-cluster
  environment: test
providers:
  digitalocean:
    enabled: true
    region: nyc3
nodes:
  - name: test-node
    provider: digitalocean
    roles:
      - master
`,
			fileExt: ".yaml",
			wantErr: false,
		},
		{
			name: "Valid JSON",
			fileContent: `{
				"metadata": {
					"name": "test-cluster",
					"environment": "test"
				},
				"providers": {
					"digitalocean": {
						"enabled": true,
						"region": "nyc3"
					}
				},
				"nodes": [
					{
						"name": "test-node",
						"provider": "digitalocean",
						"roles": ["master"]
					}
				]
			}`,
			fileExt: ".json",
			wantErr: false,
		},
		{
			name:          "Invalid YAML",
			fileContent:   "invalid: yaml: content:",
			fileExt:       ".yaml",
			wantErr:       true,
			errorContains: "failed to parse YAML",
		},
		{
			name:          "Invalid JSON",
			fileContent:   `{"invalid json`,
			fileExt:       ".json",
			wantErr:       true,
			errorContains: "failed to parse JSON",
		},
		{
			name:          "Unsupported format",
			fileContent:   "content",
			fileExt:       ".txt",
			wantErr:       true,
			errorContains: "unsupported configuration format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := ioutil.TempFile("", "test-config-*"+tt.fileExt)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content
			if _, err := tmpFile.WriteString(tt.fileContent); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			tmpFile.Close()

			// Create loader and load
			loader := NewLoader(tmpFile.Name())
			config, err := loader.Load()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if config == nil {
				t.Error("expected config, got nil")
			}
			if config != nil && config.Metadata.Name == "" {
				t.Error("config name not set (default should be applied)")
			}
		})
	}
}

func TestLoader_LoadNonExistentFile(t *testing.T) {
	loader := NewLoader("/non/existent/file.yaml")
	_, err := loader.Load()
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
	if !containsString(err.Error(), "configuration file not found") {
		t.Errorf("error should mention file not found, got: %v", err)
	}
}

func TestLoader_SetOverride(t *testing.T) {
	loader := NewLoader("test.yaml")
	loader.SetOverride("metadata.name", "override-name")
	loader.SetOverride("cluster.version", "v1.29.0")

	if len(loader.overrides) != 2 {
		t.Errorf("expected 2 overrides, got %d", len(loader.overrides))
	}
	if loader.overrides["metadata.name"] != "override-name" {
		t.Error("override not set correctly")
	}
}

func TestLoader_AddValidator(t *testing.T) {
	loader := NewLoader("test.yaml")
	validator := &mockValidator{}

	loader.AddValidator(validator)

	if len(loader.validators) != 1 {
		t.Errorf("expected 1 validator, got %d", len(loader.validators))
	}
}

func TestLoader_GetConfig(t *testing.T) {
	loader := NewLoader("test.yaml")
	if loader.GetConfig() != nil {
		t.Error("GetConfig should return nil before loading")
	}

	// Create a simple config file
	tmpFile, _ := ioutil.TempFile("", "test-config-*.yaml")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`
metadata:
  name: test
providers:
  digitalocean:
    enabled: true
nodes:
  - name: test
    provider: digitalocean
    roles: [master]
`)
	tmpFile.Close()

	loader = NewLoader(tmpFile.Name())
	loader.Load()

	if loader.GetConfig() == nil {
		t.Error("GetConfig should return config after loading")
	}
}

func TestLoader_SaveConfig(t *testing.T) {
	tests := []struct {
		name    string
		ext     string
		wantErr bool
	}{
		{
			name:    "Save as YAML",
			ext:     ".yaml",
			wantErr: false,
		},
		{
			name:    "Save as YML",
			ext:     ".yml",
			wantErr: false,
		},
		{
			name:    "Save as JSON",
			ext:     ".json",
			wantErr: false,
		},
		{
			name:    "Unsupported format",
			ext:     ".txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader("test.yaml")
			loader.config = &ClusterConfig{
				Metadata: Metadata{
					Name:        "test-cluster",
					Environment: "test",
				},
			}

			tmpFile := filepath.Join(os.TempDir(), "test-save"+tt.ext)
			defer os.Remove(tmpFile)

			err := loader.SaveConfig(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify file exists and has content
			if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
				t.Error("saved file does not exist")
			}

			content, _ := ioutil.ReadFile(tmpFile)
			if len(content) == 0 {
				t.Error("saved file is empty")
			}
		})
	}
}

func TestLoader_SaveConfig_NoConfig(t *testing.T) {
	loader := NewLoader("test.yaml")
	err := loader.SaveConfig("/tmp/test.yaml")
	if err == nil {
		t.Error("expected error when saving without loaded config")
	}
	if !containsString(err.Error(), "no configuration loaded") {
		t.Errorf("error should mention no configuration, got: %v", err)
	}
}

func TestLoader_ApplyEnvironmentOverrides(t *testing.T) {
	// Set environment variable
	os.Setenv("CLUSTER_METADATA_NAME", "env-override")
	os.Setenv("CLUSTER_METADATA_ENVIRONMENT", "production")
	defer os.Unsetenv("CLUSTER_METADATA_NAME")
	defer os.Unsetenv("CLUSTER_METADATA_ENVIRONMENT")

	config := &ClusterConfig{
		Metadata: Metadata{
			Name:        "original",
			Environment: "dev",
		},
	}

	loader := NewLoader("test.yaml")
	err := loader.applyEnvironmentOverrides(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if config.Metadata.Name != "env-override" {
		t.Errorf("expected name 'env-override', got '%s'", config.Metadata.Name)
	}
	if config.Metadata.Environment != "production" {
		t.Errorf("expected environment 'production', got '%s'", config.Metadata.Environment)
	}
}

func TestLoader_SetDefaults(t *testing.T) {
	config := &ClusterConfig{}
	loader := NewLoader("test.yaml")

	err := loader.setDefaults(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check metadata defaults
	if config.Metadata.Name != "kubernetes-cluster" {
		t.Errorf("expected default name 'kubernetes-cluster', got '%s'", config.Metadata.Name)
	}
	if config.Metadata.Environment != "development" {
		t.Errorf("expected default environment 'development', got '%s'", config.Metadata.Environment)
	}
	if config.Metadata.Version != "1.0.0" {
		t.Errorf("expected default version '1.0.0', got '%s'", config.Metadata.Version)
	}

	// Check cluster defaults
	if config.Cluster.Type != "rke" {
		t.Errorf("expected default type 'rke', got '%s'", config.Cluster.Type)
	}

	// Check network defaults
	if config.Network.Mode != "vpc" {
		t.Errorf("expected default mode 'vpc', got '%s'", config.Network.Mode)
	}
	if config.Network.CIDR != "10.0.0.0/16" {
		t.Errorf("expected default CIDR '10.0.0.0/16', got '%s'", config.Network.CIDR)
	}

	// Check Kubernetes defaults
	if config.Kubernetes.NetworkPlugin != "canal" {
		t.Errorf("expected default network plugin 'canal', got '%s'", config.Kubernetes.NetworkPlugin)
	}
	if config.Kubernetes.PodCIDR != "10.42.0.0/16" {
		t.Errorf("expected default pod CIDR '10.42.0.0/16', got '%s'", config.Kubernetes.PodCIDR)
	}

	// Check SSH defaults
	if config.Security.SSHConfig.Port != 22 {
		t.Errorf("expected default SSH port 22, got %d", config.Security.SSHConfig.Port)
	}
}

func TestLoader_SetDefaults_WithWireGuard(t *testing.T) {
	config := &ClusterConfig{
		Network: NetworkConfig{
			WireGuard: &WireGuardConfig{
				Enabled: true,
			},
		},
	}

	loader := NewLoader("test.yaml")
	err := loader.setDefaults(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check WireGuard defaults
	if config.Network.WireGuard.Port != 51820 {
		t.Errorf("expected default port 51820, got %d", config.Network.WireGuard.Port)
	}
	if config.Network.WireGuard.PersistentKeepalive != 25 {
		t.Errorf("expected default keepalive 25, got %d", config.Network.WireGuard.PersistentKeepalive)
	}
	if config.Network.WireGuard.MTU != 1420 {
		t.Errorf("expected default MTU 1420, got %d", config.Network.WireGuard.MTU)
	}
	if len(config.Network.WireGuard.DNS) != 2 {
		t.Errorf("expected 2 default DNS servers, got %d", len(config.Network.WireGuard.DNS))
	}
	if len(config.Network.WireGuard.AllowedIPs) == 0 {
		t.Error("expected default allowed IPs to be set")
	}
}

func TestLoader_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        *ClusterConfig
		wantErr       bool
		errorContains string
	}{
		{
			name: "Valid config",
			config: &ClusterConfig{
				Metadata: Metadata{
					Name: "test-cluster",
				},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
					},
				},
				Nodes: []NodeConfig{
					{
						Name:     "master-1",
						Provider: "digitalocean",
						Roles:    []string{"master"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing cluster name",
			config: &ClusterConfig{
				Metadata: Metadata{
					Name: "",
				},
			},
			wantErr:       true,
			errorContains: "cluster name is required",
		},
		{
			name: "No providers enabled",
			config: &ClusterConfig{
				Metadata: Metadata{
					Name: "test",
				},
				Providers: ProvidersConfig{},
			},
			wantErr:       true,
			errorContains: "at least one cloud provider must be enabled",
		},
		{
			name: "No nodes configured",
			config: &ClusterConfig{
				Metadata: Metadata{
					Name: "test",
				},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
					},
				},
				Nodes:     []NodeConfig{},
				NodePools: map[string]NodePool{},
			},
			wantErr:       true,
			errorContains: "at least one node or node pool must be configured",
		},
		{
			name: "No master nodes",
			config: &ClusterConfig{
				Metadata: Metadata{
					Name: "test",
				},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
					},
				},
				Nodes: []NodeConfig{
					{
						Name:     "worker-1",
						Provider: "digitalocean",
						Roles:    []string{"worker"},
					},
				},
			},
			wantErr:       true,
			errorContains: "at least one control plane node is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader("test.yaml")
			err := loader.validate(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("error '%v' does not contain '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoader_ValidateWithCustomValidator(t *testing.T) {
	loader := NewLoader("test.yaml")
	validator := &mockValidator{
		validateFunc: func(config *ClusterConfig) error {
			if config.Metadata.Environment == "forbidden" {
				return fmt.Errorf("forbidden environment")
			}
			return nil
		},
	}
	loader.AddValidator(validator)

	config := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test",
			Environment: "forbidden",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
			},
		},
		Nodes: []NodeConfig{
			{
				Name:     "master-1",
				Provider: "digitalocean",
				Roles:    []string{"master"},
			},
		},
	}

	err := loader.validate(config)
	if err == nil {
		t.Error("expected custom validator error")
	}
	if !containsString(err.Error(), "forbidden environment") {
		t.Errorf("error should be from custom validator, got: %v", err)
	}
}

func TestMergeConfigs(t *testing.T) {
	config1 := &ClusterConfig{
		Metadata: Metadata{
			Name:        "cluster1",
			Environment: "dev",
		},
		Nodes: []NodeConfig{
			{Name: "node1", Provider: "digitalocean"},
		},
	}

	config2 := &ClusterConfig{
		Metadata: Metadata{
			Name:        "cluster2",
			Environment: "prod",
		},
		Nodes: []NodeConfig{
			{Name: "node2", Provider: "linode"},
		},
	}

	merged, err := MergeConfigs(config1, config2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if merged.Metadata.Name != "cluster2" {
		t.Errorf("expected merged name 'cluster2', got '%s'", merged.Metadata.Name)
	}
	if merged.Metadata.Environment != "prod" {
		t.Errorf("expected merged environment 'prod', got '%s'", merged.Metadata.Environment)
	}
	if len(merged.Nodes) != 2 {
		t.Errorf("expected 2 nodes after merge, got %d", len(merged.Nodes))
	}
}

func TestMergeConfigs_Empty(t *testing.T) {
	_, err := MergeConfigs()
	if err == nil {
		t.Error("expected error for empty configs")
	}
	if !containsString(err.Error(), "no configurations to merge") {
		t.Errorf("error should mention no configurations, got: %v", err)
	}
}

func TestLoader_SetConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    interface{}
		validate func(*ClusterConfig) bool
	}{
		{
			name:  "Set metadata.name",
			path:  "metadata.name",
			value: "new-name",
			validate: func(c *ClusterConfig) bool {
				return c.Metadata.Name == "new-name"
			},
		},
		{
			name:  "Set metadata.environment",
			path:  "metadata.environment",
			value: "production",
			validate: func(c *ClusterConfig) bool {
				return c.Metadata.Environment == "production"
			},
		},
		{
			name:  "Set metadata.owner",
			path:  "metadata.owner",
			value: "devops-team",
			validate: func(c *ClusterConfig) bool {
				return c.Metadata.Owner == "devops-team"
			},
		},
		{
			name:  "Set cluster.type",
			path:  "cluster.type",
			value: "k3s",
			validate: func(c *ClusterConfig) bool {
				return c.Cluster.Type == "k3s"
			},
		},
		{
			name:  "Set cluster.version",
			path:  "cluster.version",
			value: "v1.29.0",
			validate: func(c *ClusterConfig) bool {
				return c.Cluster.Version == "v1.29.0"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader("test.yaml")
			config := &ClusterConfig{}

			err := loader.setConfigValue(config, tt.path, tt.value)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.validate(config) {
				t.Errorf("config value not set correctly for path '%s'", tt.path)
			}
		})
	}
}

// Mock validator for testing
type mockValidator struct {
	validateFunc func(config *ClusterConfig) error
}

func (m *mockValidator) Validate(config *ClusterConfig) error {
	if m.validateFunc != nil {
		return m.validateFunc(config)
	}
	return nil
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
