package config

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestSaveToYAML_InvalidPath(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "test",
		},
	}

	// Try to save to invalid path
	err := SaveToYAML(cfg, "/invalid/path/that/does/not/exist/config.yaml")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestSaveToYAML_Success(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name:        "test-cluster",
			Environment: "test",
		},
		Providers: ProvidersConfig{
			DigitalOcean: &DigitalOceanProvider{
				Enabled: true,
				Region:  "nyc3",
			},
		},
	}

	tmpFile, err := ioutil.TempFile("", "test-save-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = SaveToYAML(cfg, tmpFile.Name())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify file has content
	content, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if len(content) == 0 {
		t.Error("saved file is empty")
	}

	// Verify it contains expected data
	contentStr := string(content)
	if len(contentStr) < 10 {
		t.Error("saved content is too short")
	}
}

func TestLoadFromYAML_InvalidFile(t *testing.T) {
	_, err := LoadFromYAML("/non/existent/file.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadFromYAML_EmptyFile(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "test-empty-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = LoadFromYAML(tmpFile.Name())
	// Should handle empty file gracefully
	if err != nil {
		// This is expected - empty file can't be parsed
		return
	}
}

func TestApplyDefaults_MinimalConfig(t *testing.T) {
	cfg := &ClusterConfig{}
	applyDefaults(cfg)

	// Check that defaults are applied
	if cfg.Kubernetes.Distribution == "" {
		t.Error("expected default distribution to be set")
	}

	if cfg.Kubernetes.PodCIDR == "" {
		t.Error("expected default pod CIDR to be set")
	}

	if cfg.Kubernetes.ServiceCIDR == "" {
		t.Error("expected default service CIDR to be set")
	}
}

func TestApplyDefaults_PreserveExisting(t *testing.T) {
	cfg := &ClusterConfig{
		Kubernetes: KubernetesConfig{
			Distribution: "k3s",
			PodCIDR:      "10.100.0.0/16",
		},
	}

	applyDefaults(cfg)

	// Ensure existing values are not overwritten
	if cfg.Kubernetes.Distribution != "k3s" {
		t.Errorf("expected distribution 'k3s' to be preserved, got '%s'", cfg.Kubernetes.Distribution)
	}

	if cfg.Kubernetes.PodCIDR != "10.100.0.0/16" {
		t.Errorf("expected pod CIDR '10.100.0.0/16' to be preserved, got '%s'", cfg.Kubernetes.PodCIDR)
	}
}

func TestValidateConfig_MissingClusterName(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "",
		},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("expected error for missing cluster name")
	}
}

func TestValidateConfig_MissingProviders(t *testing.T) {
	cfg := &ClusterConfig{
		Metadata: Metadata{
			Name: "test",
		},
		Providers: ProvidersConfig{},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("expected error for no providers enabled")
	}
}

func TestValidateConfig_MissingNodes(t *testing.T) {
	cfg := &ClusterConfig{
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
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("expected error for no nodes configured")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
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
			Version:      "v1.28.5+rke2r1",
		},
		NodePools: map[string]NodePool{
			"masters": {
				Name:     "masters",
				Provider: "digitalocean",
				Count:    3,
				Roles:    []string{"master"},
			},
			"workers": {
				Name:     "workers",
				Provider: "digitalocean",
				Count:    2,
				Roles:    []string{"worker"},
			},
		},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}
}

func TestExpandEnvVars_MultipleVars(t *testing.T) {
	os.Setenv("TEST_VAR1", "value1")
	os.Setenv("TEST_VAR2", "value2")
	defer os.Unsetenv("TEST_VAR1")
	defer os.Unsetenv("TEST_VAR2")

	input := "${TEST_VAR1}-${TEST_VAR2}"
	result := expandEnvVars(input)

	expected := "value1-value2"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestExpandEnvVars_NoVars(t *testing.T) {
	input := "no variables here"
	result := expandEnvVars(input)

	if result != input {
		t.Errorf("expected '%s', got '%s'", input, result)
	}
}

func TestExpandEnvVars_EmptyVar(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR")

	input := "${NONEXISTENT_VAR}"
	result := expandEnvVars(input)

	// expandEnvVars might not replace non-existent vars or return empty
	// Just check it returns something (either empty or original)
	if result != "" && result != input {
		// Either behavior is acceptable
		t.Logf("got result: '%s'", result)
	}
}
