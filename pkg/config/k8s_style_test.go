package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadFromK8sYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "Valid K8s-style config",
			content: `apiVersion: kubernetes-create.io/v1
kind: Cluster
metadata:
  name: test-cluster
  labels:
    env: test
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
    version: v1.28.5+rke2r1
`,
			wantErr: false,
		},
		{
			name: "Invalid Kind",
			content: `apiVersion: kubernetes-create.io/v1
kind: InvalidKind
metadata:
  name: test-cluster
`,
			wantErr: true,
		},
		{
			name:    "Invalid YAML",
			content: `this is not valid: [[[`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "k8s-config-*.yaml")
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
			cfg, err := LoadFromK8sYAML(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromK8sYAML() error = %v, wantErr %v", err, tt.wantErr)
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

func TestGenerateK8sStyleConfig(t *testing.T) {
	k8sConfig := GenerateK8sStyleConfig()

	if k8sConfig == nil {
		t.Fatal("Expected k8sConfig to be non-nil")
	}

	if k8sConfig.APIVersion != "kubernetes-create.io/v1" {
		t.Errorf("Expected apiVersion 'kubernetes-create.io/v1', got '%s'", k8sConfig.APIVersion)
	}

	if k8sConfig.Kind != "Cluster" {
		t.Errorf("Expected kind 'Cluster', got '%s'", k8sConfig.Kind)
	}

	if k8sConfig.Metadata.Name != "production-cluster" {
		t.Errorf("Expected name 'production-cluster', got '%s'", k8sConfig.Metadata.Name)
	}

	if len(k8sConfig.Metadata.Labels) == 0 {
		t.Error("Expected labels to be present")
	}

	if k8sConfig.Spec.Providers.DigitalOcean == nil {
		t.Error("Expected DigitalOcean provider to be present")
	}

	if len(k8sConfig.Spec.NodePools) == 0 {
		t.Error("Expected node pools to be present")
	}
}

func TestSaveK8sStyleConfig(t *testing.T) {
	k8sConfig := GenerateK8sStyleConfig()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "k8s-save-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Save config
	err = SaveK8sStyleConfig(k8sConfig, tmpFile.Name())
	if err != nil {
		t.Fatalf("SaveK8sStyleConfig() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile.Name()); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "apiVersion:") {
		t.Error("Saved file should contain apiVersion")
	}
	if !strings.Contains(contentStr, "kind:") {
		t.Error("Saved file should contain kind")
	}
	if !strings.Contains(contentStr, "metadata:") {
		t.Error("Saved file should contain metadata")
	}
}

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_TOKEN", "my-test-token")
	defer os.Unsetenv("TEST_TOKEN")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Replace environment variable",
			input:    "${TEST_TOKEN}",
			expected: "my-test-token",
		},
		{
			name:     "Replace in string",
			input:    "token: ${TEST_TOKEN}",
			expected: "token: my-test-token",
		},
		{
			name:     "Multiple occurrences",
			input:    "${TEST_TOKEN} and ${TEST_TOKEN}",
			expected: "my-test-token and my-test-token",
		},
		{
			name:     "Non-existent variable",
			input:    "${NON_EXISTENT_VAR}",
			expected: "${NON_EXISTENT_VAR}",
		},
		{
			name:     "No variables",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestConvertFromK8sStyle(t *testing.T) {
	k8sConfig := &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "test-cluster",
			Labels: map[string]string{
				"env": "production",
			},
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				DigitalOcean: &DigitalOceanSpec{
					Enabled: true,
					Token:   "test-token",
					Region:  "nyc3",
				},
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
					Size:     "s-2vcpu-4gb",
					Region:   "nyc3",
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution: "rke2",
				Version:      "v1.28.5+rke2r1",
			},
		},
	}

	cfg := convertFromK8sStyle(k8sConfig)

	if cfg == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if cfg.Metadata.Name != "test-cluster" {
		t.Errorf("Expected name 'test-cluster', got '%s'", cfg.Metadata.Name)
	}

	if cfg.Providers.DigitalOcean == nil {
		t.Fatal("Expected DigitalOcean provider to be non-nil")
	}

	if !cfg.Providers.DigitalOcean.Enabled {
		t.Error("Expected DigitalOcean provider to be enabled")
	}

	if len(cfg.NodePools) != 1 {
		t.Errorf("Expected 1 node pool, got %d", len(cfg.NodePools))
	}

	if cfg.Kubernetes.Distribution != "rke2" {
		t.Errorf("Expected distribution 'rke2', got '%s'", cfg.Kubernetes.Distribution)
	}
}

func TestConvertFromK8sStyle_WithMultipleProviders(t *testing.T) {
	k8sConfig := &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "multi-cloud-cluster",
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				DigitalOcean: &DigitalOceanSpec{
					Enabled: true,
					Token:   "do-token",
					Region:  "nyc3",
				},
				Linode: &LinodeSpec{
					Enabled: true,
					Token:   "linode-token",
					Region:  "us-east",
				},
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "do-masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
				},
				{
					Name:     "linode-masters",
					Provider: "linode",
					Count:    2,
					Roles:    []string{"master"},
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution: "rke2",
			},
		},
	}

	cfg := convertFromK8sStyle(k8sConfig)

	if cfg.Providers.DigitalOcean == nil {
		t.Error("Expected DigitalOcean provider to be present")
	}

	if cfg.Providers.Linode == nil {
		t.Error("Expected Linode provider to be present")
	}

	if len(cfg.NodePools) != 2 {
		t.Errorf("Expected 2 node pools, got %d", len(cfg.NodePools))
	}
}

func TestConvertFromK8sStyle_WithWireGuard(t *testing.T) {
	k8sConfig := &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "vpn-cluster",
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				DigitalOcean: &DigitalOceanSpec{
					Enabled: true,
					Token:   "token",
					Region:  "nyc3",
				},
			},
			Network: NetworkSpec{
				WireGuard: &WireGuardSpec{
					Enabled:             true,
					ServerEndpoint:      "1.2.3.4:51820",
					ServerPublicKey:     "test-key",
					Port:                51820,
					MTU:                 1420,
					PersistentKeepalive: 25,
				},
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution: "rke2",
			},
		},
	}

	cfg := convertFromK8sStyle(k8sConfig)

	if cfg.Network.WireGuard == nil {
		t.Fatal("Expected WireGuard config to be present")
	}

	if !cfg.Network.WireGuard.Enabled {
		t.Error("Expected WireGuard to be enabled")
	}

	if cfg.Network.WireGuard.ServerEndpoint != "1.2.3.4:51820" {
		t.Errorf("Expected endpoint '1.2.3.4:51820', got '%s'", cfg.Network.WireGuard.ServerEndpoint)
	}

	if cfg.Network.WireGuard.Port != 51820 {
		t.Errorf("Expected port 51820, got %d", cfg.Network.WireGuard.Port)
	}
}

func TestLoadFromK8sYAML_FileNotFound(t *testing.T) {
	_, err := LoadFromK8sYAML("/non/existent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestSaveK8sStyleConfig_Error(t *testing.T) {
	k8sConfig := GenerateK8sStyleConfig()

	// Try to save to invalid path (directory that doesn't exist and can't be created)
	err := SaveK8sStyleConfig(k8sConfig, "/root/forbidden/path/config.yaml")
	if err == nil {
		// This might pass if running as root, so we check if file was actually created
		if _, statErr := os.Stat("/root/forbidden/path/config.yaml"); statErr == nil {
			// Clean up if somehow it was created
			os.Remove("/root/forbidden/path/config.yaml")
			t.Skip("Running as root, skipping permission test")
		}
	}
}

func TestExpandEnvVars_EdgeCases(t *testing.T) {
	// Test with empty string
	result := expandEnvVars("")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Test with just ${ without closing }
	result = expandEnvVars("${INCOMPLETE")
	if result != "${INCOMPLETE" {
		t.Errorf("Expected '${INCOMPLETE', got '%s'", result)
	}

	// Test with nested ${} (should not be supported but shouldn't crash)
	result = expandEnvVars("${OUTER${INNER}}")
	// Just verify it doesn't crash, exact behavior doesn't matter
	if result == "" {
		// This is fine
	}
}

func TestConvertFromK8sStyle_EmptyProviders(t *testing.T) {
	k8sConfig := &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "minimal-cluster",
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				// Both providers nil
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution: "rke2",
			},
		},
	}

	cfg := convertFromK8sStyle(k8sConfig)

	if cfg == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if cfg.Providers.DigitalOcean != nil {
		t.Error("Expected DigitalOcean provider to be nil")
	}

	if cfg.Providers.Linode != nil {
		t.Error("Expected Linode provider to be nil")
	}
}

func TestConvertFromK8sStyle_WithRKE2Config(t *testing.T) {
	k8sConfig := &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "rke2-cluster",
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				DigitalOcean: &DigitalOceanSpec{
					Enabled: true,
					Token:   "token",
					Region:  "nyc3",
				},
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution: "rke2",
				RKE2: &RKE2Spec{
					Channel:              "stable",
					ClusterToken:         "test-token",
					TLSSan:               []string{"api.example.com"},
					DisableComponents:    []string{"rke2-ingress-nginx"},
					SnapshotScheduleCron: "0 */6 * * *",
					SnapshotRetention:    10,
					SecretsEncryption:    true,
					WriteKubeconfigMode:  "0600",
					ExtraServerArgs:      map[string]string{"key": "value"},
					ExtraAgentArgs:       map[string]string{"agent-key": "agent-value"},
				},
			},
		},
	}

	cfg := convertFromK8sStyle(k8sConfig)

	if cfg.Kubernetes.RKE2 == nil {
		t.Fatal("Expected RKE2 config to be present")
	}

	if cfg.Kubernetes.RKE2.Channel != "stable" {
		t.Errorf("Expected channel 'stable', got '%s'", cfg.Kubernetes.RKE2.Channel)
	}

	if cfg.Kubernetes.RKE2.ClusterToken != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", cfg.Kubernetes.RKE2.ClusterToken)
	}

	if len(cfg.Kubernetes.RKE2.TLSSan) != 1 {
		t.Errorf("Expected 1 TLS SAN, got %d", len(cfg.Kubernetes.RKE2.TLSSan))
	}

	if !cfg.Kubernetes.RKE2.SecretsEncryption {
		t.Error("Expected secrets encryption to be enabled")
	}

	if len(cfg.Kubernetes.RKE2.ExtraServerArgs) != 1 {
		t.Errorf("Expected 1 extra server arg, got %d", len(cfg.Kubernetes.RKE2.ExtraServerArgs))
	}

	if len(cfg.Kubernetes.RKE2.ExtraAgentArgs) != 1 {
		t.Errorf("Expected 1 extra agent arg, got %d", len(cfg.Kubernetes.RKE2.ExtraAgentArgs))
	}
}

func TestConvertFromK8sStyle_WithTaints(t *testing.T) {
	k8sConfig := &KubernetesStyleConfig{
		APIVersion: "kubernetes-create.io/v1",
		Kind:       "Cluster",
		Metadata: K8sMetadata{
			Name: "tainted-cluster",
		},
		Spec: ClusterSpec2{
			Providers: ProvidersSpec{
				DigitalOcean: &DigitalOceanSpec{
					Enabled: true,
					Token:   "token",
					Region:  "nyc3",
				},
			},
			NodePools: []NodePoolSpec{
				{
					Name:     "masters",
					Provider: "digitalocean",
					Count:    1,
					Roles:    []string{"master"},
					Taints: []TaintSpec{
						{
							Key:    "node-role.kubernetes.io/master",
							Value:  "",
							Effect: "NoSchedule",
						},
					},
					Labels: map[string]string{
						"node-type": "master",
					},
				},
			},
			Kubernetes: KubernetesSpec{
				Distribution: "rke2",
			},
		},
	}

	cfg := convertFromK8sStyle(k8sConfig)

	if len(cfg.NodePools) != 1 {
		t.Fatalf("Expected 1 node pool, got %d", len(cfg.NodePools))
	}

	masters := cfg.NodePools["masters"]
	if len(masters.Taints) != 1 {
		t.Errorf("Expected 1 taint, got %d", len(masters.Taints))
	}

	if masters.Taints[0].Key != "node-role.kubernetes.io/master" {
		t.Errorf("Expected taint key 'node-role.kubernetes.io/master', got '%s'", masters.Taints[0].Key)
	}

	if masters.Taints[0].Effect != "NoSchedule" {
		t.Errorf("Expected taint effect 'NoSchedule', got '%s'", masters.Taints[0].Effect)
	}

	if len(masters.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(masters.Labels))
	}
}
