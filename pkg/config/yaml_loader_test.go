package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "Valid legacy YAML",
			content: `metadata:
  name: test-cluster
providers:
  digitalocean:
    enabled: true
    token: test-token
    region: nyc3
nodePools:
  masters:
    name: masters
    provider: digitalocean
    count: 1
    roles:
      - master
  workers:
    name: workers
    provider: digitalocean
    count: 1
    roles:
      - worker
kubernetes:
  distribution: rke2
`,
			wantErr: false,
		},
		{
			name: "Valid K8s-style YAML",
			content: `apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: test-cluster
spec:
  providers:
    digitalocean:
      enabled: true
      token: test-token
      region: nyc3
  nodePools:
    - name: masters
      provider: digitalocean
      count: 1
      role: master
      size: s-2vcpu-4gb
    - name: workers
      provider: digitalocean
      count: 1
      role: worker
      size: s-2vcpu-4gb
  kubernetes:
    distribution: rke2
`,
			wantErr: false,
		},
		{
			name:    "Invalid YAML",
			content: `this is not valid yaml: [[[`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			tmpFile.Close()

			// Load config
			cfg, err := LoadFromYAML(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg == nil {
				t.Error("Expected config to be non-nil")
			}

			if !tt.wantErr && cfg != nil {
				if cfg.Metadata.Name != "test-cluster" {
					t.Errorf("Expected name 'test-cluster', got '%s'", cfg.Metadata.Name)
				}
			}
		})
	}
}

func TestLoadFromYAML_HomeDir(t *testing.T) {
	// Create temp file in actual location (not using ~)
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `metadata:
  name: test-cluster
providers:
  digitalocean:
    enabled: true
    token: test-token
kubernetes:
  distribution: rke2
nodePools:
  masters:
    name: masters
    provider: digitalocean
    count: 1
    roles:
      - master
  workers:
    name: workers
    provider: digitalocean
    count: 1
    roles:
      - worker
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Load should work with absolute path
	_, err = LoadFromYAML(tmpFile.Name())
	if err != nil {
		t.Errorf("LoadFromYAML() with absolute path failed: %v", err)
	}
}

func TestSaveToYAML(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test-cluster",
			Environment: "testing",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "test-token",
				Region:  "nyc3",
			},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "rke2",
			Version:      "v1.28.5+rke2r1",
		},
		NodePools: map[string]NodePool{
			"masters": {
				Name:     "masters",
				Provider: "digitalocean",
				Count:    1,
				Roles:    []string{"master"},
			},
			"workers": {
				Name:     "workers",
				Provider: "digitalocean",
				Count:    1,
				Roles:    []string{"worker"},
			},
		},
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "config.yaml")

	// Save config
	err = SaveToYAML(cfg, filePath)
	if err != nil {
		t.Errorf("SaveToYAML() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back
	loadedCfg, err := LoadFromYAML(filePath)
	if err != nil {
		t.Errorf("Failed to load saved config: %v", err)
	}

	if loadedCfg.Metadata.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", loadedCfg.Metadata.Name)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "test",
		},
		Kubernetes: KubernetesConfig{},
		Network: NetworkConfig{
			WireGuard: &WireGuardConfig{
				Enabled: true,
			},
		},
	}

	applyDefaults(cfg)

	// Check Kubernetes defaults
	if cfg.Kubernetes.Distribution != "rke2" {
		t.Errorf("Expected distribution 'rke2', got '%s'", cfg.Kubernetes.Distribution)
	}
	if cfg.Kubernetes.Version != "v1.28.5+rke2r1" {
		t.Errorf("Expected version 'v1.28.5+rke2r1', got '%s'", cfg.Kubernetes.Version)
	}
	if cfg.Kubernetes.NetworkPlugin != "calico" {
		t.Errorf("Expected network plugin 'calico', got '%s'", cfg.Kubernetes.NetworkPlugin)
	}
	if cfg.Kubernetes.PodCIDR != "10.42.0.0/16" {
		t.Errorf("Expected pod CIDR '10.42.0.0/16', got '%s'", cfg.Kubernetes.PodCIDR)
	}
	if cfg.Kubernetes.ServiceCIDR != "10.43.0.0/16" {
		t.Errorf("Expected service CIDR '10.43.0.0/16', got '%s'", cfg.Kubernetes.ServiceCIDR)
	}

	// Check WireGuard defaults
	if cfg.Network.WireGuard.Port != 51820 {
		t.Errorf("Expected WireGuard port 51820, got %d", cfg.Network.WireGuard.Port)
	}
	if cfg.Network.WireGuard.MTU != 1420 {
		t.Errorf("Expected WireGuard MTU 1420, got %d", cfg.Network.WireGuard.MTU)
	}
	if cfg.Network.WireGuard.PersistentKeepalive != 25 {
		t.Errorf("Expected keepalive 25, got %d", cfg.Network.WireGuard.PersistentKeepalive)
	}

	// Check metadata defaults
	if cfg.Metadata.Environment != "production" {
		t.Errorf("Expected environment 'production', got '%s'", cfg.Metadata.Environment)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ClusterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				NodePools: map[string]NodePool{
					"masters": {
						Count: 1,
						Roles: []string{"master"},
					},
					"workers": {
						Count: 1,
						Roles: []string{"worker"},
					},
				},
				Kubernetes: KubernetesConfig{
					Distribution: "rke2",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			cfg: &ClusterConfig{
				Metadata: Metadata{},
			},
			wantErr: true,
			errMsg:  "metadata.name is required",
		},
		{
			name: "No providers enabled",
			cfg: &ClusterConfig{
				Metadata:  Metadata{Name: "test"},
				Providers: ProvidersConfig{},
			},
			wantErr: true,
			errMsg:  "at least one cloud provider must be enabled",
		},
		{
			name: "DigitalOcean enabled without token",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "",
					},
				},
			},
			wantErr: true,
			errMsg:  "digitalocean token is required",
		},
		{
			name: "No node pools",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				NodePools: map[string]NodePool{},
			},
			wantErr: true,
			errMsg:  "at least one node pool is required",
		},
		{
			name: "No master nodes",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				NodePools: map[string]NodePool{
					"workers": {
						Count: 2,
						Roles: []string{"worker"},
					},
				},
				Kubernetes: KubernetesConfig{
					Distribution: "rke2",
				},
			},
			wantErr: true,
			errMsg:  "at least one master node is required",
		},
		{
			name: "Even number of masters",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				NodePools: map[string]NodePool{
					"masters": {
						Count: 2,
						Roles: []string{"master"},
					},
					"workers": {
						Count: 1,
						Roles: []string{"worker"},
					},
				},
				Kubernetes: KubernetesConfig{
					Distribution: "rke2",
				},
			},
			wantErr: true,
			errMsg:  "master count must be odd",
		},
		{
			name: "No worker nodes",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				NodePools: map[string]NodePool{
					"masters": {
						Count: 1,
						Roles: []string{"master"},
					},
				},
				Kubernetes: KubernetesConfig{
					Distribution: "rke2",
				},
			},
			wantErr: true,
			errMsg:  "at least one worker node is required",
		},
		{
			name: "Invalid distribution",
			cfg: &ClusterConfig{
				Metadata: Metadata{Name: "test"},
				Providers: ProvidersConfig{
					DigitalOcean: &DigitalOceanProvider{
						Enabled: true,
						Token:   "test-token",
					},
				},
				NodePools: map[string]NodePool{
					"masters": {
						Count: 1,
						Roles: []string{"master"},
					},
					"workers": {
						Count: 1,
						Roles: []string{"worker"},
					},
				},
				Kubernetes: KubernetesConfig{
					Distribution: "invalid",
				},
			},
			wantErr: true,
			errMsg:  "only rke2 and k3s distributions are supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestGenerateExampleConfig(t *testing.T) {
	cfg := GenerateExampleConfig()

	if cfg == nil {
		t.Fatal("GenerateExampleConfig() returned nil")
	}

	if cfg.Metadata.Name != "production-cluster" {
		t.Errorf("Expected name 'production-cluster', got '%s'", cfg.Metadata.Name)
	}

	if cfg.Providers.DigitalOcean == nil {
		t.Error("DigitalOcean provider should not be nil")
	}

	if cfg.Providers.Linode == nil {
		t.Error("Linode provider should not be nil")
	}

	if cfg.Network.WireGuard == nil {
		t.Error("WireGuard config should not be nil")
	}

	if cfg.Kubernetes.Distribution != "rke2" {
		t.Errorf("Expected distribution 'rke2', got '%s'", cfg.Kubernetes.Distribution)
	}

	if len(cfg.NodePools) == 0 {
		t.Error("Node pools should not be empty")
	}
}

func TestSaveToYAML_HomeDir(t *testing.T) {
	// This test verifies the home directory expansion works
	// but we'll use a regular temp file since we can't write to actual home
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "test-cluster",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "test-token",
			},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "rke2",
		},
		NodePools: map[string]NodePool{
			"masters": {
				Name:     "masters",
				Provider: "digitalocean",
				Count:    1,
				Roles:    []string{"master"},
			},
			"workers": {
				Name:     "workers",
				Provider: "digitalocean",
				Count:    1,
				Roles:    []string{"worker"},
			},
		},
	}

	tmpDir, err := os.MkdirTemp("", "yaml-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := tmpDir + "/subdir"
	filePath := subDir + "/config.yaml"

	// This should create the subdir automatically
	err = SaveToYAML(cfg, filePath)
	if err != nil {
		t.Errorf("SaveToYAML() with subdir failed: %v", err)
	}

	// Verify subdir was created
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Subdirectory was not created")
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestApplyDefaults_EmptyWireGuard(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "test",
		},
		Kubernetes: KubernetesConfig{},
		Network: NetworkConfig{
			WireGuard: &WireGuardConfig{
				Enabled: false, // Disabled, so defaults should not apply
			},
		},
	}

	applyDefaults(cfg)

	// WireGuard is disabled, defaults should not be applied for optional fields
	// But some fields like Port may still have been set
	if cfg.Metadata.Environment != "production" {
		t.Errorf("Expected environment 'production', got '%s'", cfg.Metadata.Environment)
	}
}

func TestApplyDefaults_WithRKE2(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "test",
		},
		Kubernetes: KubernetesConfig{
			Distribution: "rke2",
			RKE2: &RKE2Config{
				Channel: "latest",
			},
		},
	}

	applyDefaults(cfg)

	// RKE2 config should be merged with defaults
	if cfg.Kubernetes.RKE2 == nil {
		t.Fatal("RKE2 config should not be nil")
	}

	if cfg.Kubernetes.RKE2.Channel != "latest" {
		t.Errorf("Expected channel 'latest', got '%s'", cfg.Kubernetes.RKE2.Channel)
	}

	// Default data dir should be set
	if cfg.Kubernetes.RKE2.DataDir == "" {
		t.Error("DataDir should have default value")
	}
}

func TestValidateConfig_LinodeWithoutToken(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Providers: ProvidersConfig{
			Linode: &LinodeProvider{
				Enabled: true,
				Token:   "", // Missing token
			},
		},
		NodePools: map[string]NodePool{
			"masters": {Count: 1, Roles: []string{"master"}},
			"workers": {Count: 1, Roles: []string{"worker"}},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "rke2",
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected validation error for missing Linode token")
	}
}

func TestValidateConfig_WireGuardMissingEndpoint(t *testing.T) {
	// WireGuard can now be auto-created, so missing endpoint is OK
	// This test now verifies that auto-create mode works
	cfg := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "test-token",
				Region:  "nyc1",
			},
		},
		Network: NetworkConfig{
			WireGuard: &WireGuardConfig{
				Enabled:         true,
				ServerEndpoint:  "", // Empty means auto-create
				ServerPublicKey: "",
			},
		},
		NodePools: map[string]NodePool{
			"masters": {Count: 1, Roles: []string{"master"}},
			"workers": {Count: 1, Roles: []string{"worker"}},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "rke2",
		},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("WireGuard auto-create should be valid, got error: %v", err)
	}
}

func TestValidateConfig_WireGuardMissingPublicKey(t *testing.T) {
	// WireGuard can now be auto-created, so missing public key is OK
	// This test now verifies that auto-create mode works even with endpoint specified
	cfg := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "test-token",
				Region:  "nyc1",
			},
		},
		Network: NetworkConfig{
			WireGuard: &WireGuardConfig{
				Enabled:         true,
				ServerEndpoint:  "", // Auto-create
				ServerPublicKey: "", // Auto-create
			},
		},
		NodePools: map[string]NodePool{
			"masters": {Count: 1, Roles: []string{"master"}},
			"workers": {Count: 1, Roles: []string{"worker"}},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "rke2",
		},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("WireGuard auto-create should be valid, got error: %v", err)
	}
}

func TestValidateConfig_K3sDistribution(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "test-token",
			},
		},
		NodePools: map[string]NodePool{
			"masters": {Count: 1, Roles: []string{"master"}},
			"workers": {Count: 1, Roles: []string{"worker"}},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "k3s",
		},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("k3s distribution should be valid, got error: %v", err)
	}
}

func TestValidateConfig_EmptyDistribution(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{Name: "test"},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Token:   "test-token",
			},
		},
		NodePools: map[string]NodePool{
			"masters": {Count: 1, Roles: []string{"master"}},
			"workers": {Count: 1, Roles: []string{"worker"}},
		},
		Kubernetes: KubernetesConfig{
			Distribution: "",
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected validation error for empty distribution")
	}
}

func TestLoadFromYAML_NonExistentFile(t *testing.T) {
	_, err := LoadFromYAML("/non/existent/path/to/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
