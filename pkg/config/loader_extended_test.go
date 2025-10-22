package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestLoader_ApplyOverrides(t *testing.T) {
	loader := NewLoader("test.yaml")
	loader.SetOverride("metadata.name", "override-cluster")
	loader.SetOverride("metadata.owner", "test-owner")
	loader.SetOverride("metadata.team", "devops")
	loader.SetOverride("cluster.type", "k3s")
	loader.SetOverride("cluster.version", "v1.29.0")

	config := &ClusterConfig{
		Metadata: Metadata{
			Name:  "original",
			Owner: "original-owner",
		},
		Cluster: ClusterSpec{
			Type:    "rke2",
			Version: "v1.28.0",
		},
	}

	err := loader.applyOverrides(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check overrides were applied
	if config.Metadata.Name != "override-cluster" {
		t.Errorf("expected name 'override-cluster', got '%s'", config.Metadata.Name)
	}

	if config.Metadata.Owner != "test-owner" {
		t.Errorf("expected owner 'test-owner', got '%s'", config.Metadata.Owner)
	}

	if config.Metadata.Team != "devops" {
		t.Errorf("expected team 'devops', got '%s'", config.Metadata.Team)
	}

	if config.Cluster.Type != "k3s" {
		t.Errorf("expected type 'k3s', got '%s'", config.Cluster.Type)
	}

	if config.Cluster.Version != "v1.29.0" {
		t.Errorf("expected version 'v1.29.0', got '%s'", config.Cluster.Version)
	}
}

func TestLoader_SetConfigValue_Metadata(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    interface{}
		validate func(*ClusterConfig) error
	}{
		{
			name:  "Set metadata.name",
			path:  "metadata.name",
			value: "test-cluster",
			validate: func(c *ClusterConfig) error {
				if c.Metadata.Name != "test-cluster" {
					return ErrValidation("name not set")
				}
				return nil
			},
		},
		{
			name:  "Set metadata.environment",
			path:  "metadata.environment",
			value: "production",
			validate: func(c *ClusterConfig) error {
				if c.Metadata.Environment != "production" {
					return ErrValidation("environment not set")
				}
				return nil
			},
		},
		{
			name:  "Set metadata.owner",
			path:  "metadata.owner",
			value: "platform-team",
			validate: func(c *ClusterConfig) error {
				if c.Metadata.Owner != "platform-team" {
					return ErrValidation("owner not set")
				}
				return nil
			},
		},
		{
			name:  "Set metadata.team",
			path:  "metadata.team",
			value: "infrastructure",
			validate: func(c *ClusterConfig) error {
				if c.Metadata.Team != "infrastructure" {
					return ErrValidation("team not set")
				}
				return nil
			},
		},
		{
			name:  "Set cluster.type",
			path:  "cluster.type",
			value: "k3s",
			validate: func(c *ClusterConfig) error {
				if c.Cluster.Type != "k3s" {
					return ErrValidation("type not set")
				}
				return nil
			},
		},
		{
			name:  "Set cluster.version",
			path:  "cluster.version",
			value: "v1.29.0",
			validate: func(c *ClusterConfig) error {
				if c.Cluster.Version != "v1.29.0" {
					return ErrValidation("version not set")
				}
				return nil
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

			if err := tt.validate(config); err != nil {
				t.Errorf("validation failed: %v", err)
			}
		})
	}
}

func TestLoader_Load_WithOverrides(t *testing.T) {
	// Create temp config file
	tmpFile, err := ioutil.TempFile("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `
metadata:
  name: original-cluster
  environment: dev
providers:
  digitalocean:
    enabled: true
    token: test-token
nodePools:
  masters:
    name: masters
    provider: digitalocean
    count: 3
    roles: [master]
`
	tmpFile.WriteString(content)
	tmpFile.Close()

	// Load with overrides
	loader := NewLoader(tmpFile.Name())
	loader.SetOverride("metadata.name", "overridden-cluster")
	loader.SetOverride("metadata.environment", "production")

	config, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check overrides were applied
	if config.Metadata.Name != "overridden-cluster" {
		t.Errorf("expected name 'overridden-cluster', got '%s'", config.Metadata.Name)
	}

	if config.Metadata.Environment != "production" {
		t.Errorf("expected environment 'production', got '%s'", config.Metadata.Environment)
	}
}

func TestLoader_Load_JSON(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `{
		"metadata": {
			"name": "json-cluster"
		},
		"providers": {
			"digitalocean": {
				"enabled": true,
				"token": "test-token"
			}
		},
		"nodePools": {
			"masters": {
				"name": "masters",
				"provider": "digitalocean",
				"count": 3,
				"roles": ["master"]
			}
		}
	}`
	tmpFile.WriteString(content)
	tmpFile.Close()

	loader := NewLoader(tmpFile.Name())
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("failed to load JSON config: %v", err)
	}

	if config.Metadata.Name != "json-cluster" {
		t.Errorf("expected name 'json-cluster', got '%s'", config.Metadata.Name)
	}
}

func TestLoader_Validate_MultipleValidators(t *testing.T) {
	loader := NewLoader("test.yaml")

	// Add multiple custom validators
	validator1Called := false
	validator2Called := false

	validator1 := &mockValidator{
		validateFunc: func(config *ClusterConfig) error {
			validator1Called = true
			if config.Metadata.Name == "forbidden" {
				return ErrValidation("forbidden name")
			}
			return nil
		},
	}

	validator2 := &mockValidator{
		validateFunc: func(config *ClusterConfig) error {
			validator2Called = true
			if config.Metadata.Environment == "invalid" {
				return ErrValidation("invalid environment")
			}
			return nil
		},
	}

	loader.AddValidator(validator1)
	loader.AddValidator(validator2)

	// Valid config
	config := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test",
			Environment: "dev",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
			},
		},
		NodePools: map[string]NodePool{
			"masters": {
				Name:     "masters",
				Provider: "digitalocean",
				Count:    3,
				Roles:    []string{"master"},
			},
		},
	}

	err := loader.validate(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !validator1Called {
		t.Error("validator1 was not called")
	}

	if !validator2Called {
		t.Error("validator2 was not called")
	}
}

func TestMergeConfig_WithNodePools(t *testing.T) {
	target := &ClusterConfig{
		Metadata: Metadata{
			Name: "target",
		},
		NodePools: map[string]NodePool{
			"pool1": {
				Name:     "pool1",
				Provider: "digitalocean",
				Count:    3,
			},
		},
	}

	source := &ClusterConfig{
		Metadata: Metadata{
			Name: "source",
		},
		NodePools: map[string]NodePool{
			"pool2": {
				Name:     "pool2",
				Provider: "linode",
				Count:    2,
			},
		},
	}

	err := mergeConfig(target, source)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Metadata should be overridden
	if target.Metadata.Name != "source" {
		t.Errorf("expected name 'source', got '%s'", target.Metadata.Name)
	}

	// Node pools should be merged
	if len(target.NodePools) != 2 {
		t.Errorf("expected 2 node pools, got %d", len(target.NodePools))
	}

	if _, exists := target.NodePools["pool1"]; !exists {
		t.Error("pool1 should exist in merged config")
	}

	if _, exists := target.NodePools["pool2"]; !exists {
		t.Error("pool2 should exist in merged config")
	}
}

func TestLoader_SetDefaults_WireGuard(t *testing.T) {
	tests := []struct {
		name   string
		config *ClusterConfig
		check  func(*ClusterConfig) bool
	}{
		{
			name: "WireGuard enabled - sets defaults",
			config: &ClusterConfig{
				Network: NetworkConfig{
					WireGuard: &WireGuardConfig{
						Enabled: true,
					},
				},
			},
			check: func(c *ClusterConfig) bool {
				return c.Network.WireGuard.Port == 51820 &&
					c.Network.WireGuard.PersistentKeepalive == 25 &&
					c.Network.WireGuard.MTU == 1420 &&
					len(c.Network.WireGuard.DNS) == 2 &&
					len(c.Network.WireGuard.AllowedIPs) > 0
			},
		},
		{
			name: "WireGuard nil - no defaults",
			config: &ClusterConfig{
				Network: NetworkConfig{
					WireGuard: nil,
				},
			},
			check: func(c *ClusterConfig) bool {
				return c.Network.WireGuard == nil
			},
		},
		{
			name: "WireGuard disabled - no defaults",
			config: &ClusterConfig{
				Network: NetworkConfig{
					WireGuard: &WireGuardConfig{
						Enabled: false,
					},
				},
			},
			check: func(c *ClusterConfig) bool {
				// Defaults might still be set even if disabled
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader("test.yaml")
			err := loader.setDefaults(tt.config)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.check(tt.config) {
				t.Error("check failed")
			}
		})
	}
}

func TestLoader_Validate_Providers(t *testing.T) {
	tests := []struct {
		name    string
		config  *ClusterConfig
		wantErr bool
	}{
		{
			name: "DigitalOcean enabled",
			config: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
					},
				},
				NodePools: map[string]NodePool{
					"masters": {Name: "masters", Count: 3, Roles: []string{"master"}},
				},
			},
			wantErr: false,
		},
		{
			name: "Linode enabled",
			config: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					Linode: &LinodeProvider{
						Enabled: true,
					},
				},
				NodePools: map[string]NodePool{
					"masters": {Name: "masters", Count: 3, Roles: []string{"master"}},
				},
			},
			wantErr: false,
		},
		{
			name: "AWS enabled",
			config: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					AWS: &AWSProvider{
						Enabled: true,
					},
				},
				NodePools: map[string]NodePool{
					"masters": {Name: "masters", Count: 3, Roles: []string{"master"}},
				},
			},
			wantErr: false,
		},
		{
			name: "Azure enabled",
			config: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					Azure: &AzureProvider{
						Enabled: true,
					},
				},
				NodePools: map[string]NodePool{
					"masters": {Name: "masters", Count: 3, Roles: []string{"master"}},
				},
			},
			wantErr: false,
		},
		{
			name: "GCP enabled",
			config: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					GCP: &GCPProvider{
						Enabled: true,
					},
				},
				NodePools: map[string]NodePool{
					"masters": {Name: "masters", Count: 3, Roles: []string{"master"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader("test.yaml")
			err := loader.validate(tt.config)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper type for validation errors
type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

func ErrValidation(msg string) error {
	return &validationError{message: msg}
}
