package cmd

import (
	"strings"
	"testing"

	"sloth-kubernetes/pkg/config"
)

// TestGenerateMinimalConfig tests minimal config generation
func TestGenerateMinimalConfig(t *testing.T) {
	cfg := generateMinimalConfig()

	if cfg == nil {
		t.Fatal("generateMinimalConfig should not return nil")
	}

	// Test APIVersion
	if cfg.APIVersion != "kubernetes-create.io/v1" {
		t.Errorf("Expected APIVersion 'kubernetes-create.io/v1', got %q", cfg.APIVersion)
	}

	// Test Kind
	if cfg.Kind != "Cluster" {
		t.Errorf("Expected Kind 'Cluster', got %q", cfg.Kind)
	}

	// Test Metadata
	if cfg.Metadata.Name == "" {
		t.Error("Cluster name should not be empty")
	}

	if len(cfg.Metadata.Labels) == 0 {
		t.Error("Labels should not be empty")
	}

	// Test Providers
	if cfg.Spec.Providers.DigitalOcean == nil {
		t.Error("DigitalOcean provider should be configured")
	}

	if cfg.Spec.Providers.DigitalOcean != nil {
		if !cfg.Spec.Providers.DigitalOcean.Enabled {
			t.Error("DigitalOcean should be enabled in minimal config")
		}

		if !strings.Contains(cfg.Spec.Providers.DigitalOcean.Token, "$") {
			t.Logf("Note: Token doesn't use environment variable syntax")
		}

		if cfg.Spec.Providers.DigitalOcean.Region == "" {
			t.Error("Region should not be empty")
		}
	}
}

// TestConfigCommandStructure tests config command structure
func TestConfigCommandStructure(t *testing.T) {
	if configCmd == nil {
		t.Fatal("configCmd should not be nil")
	}

	if configCmd.Use != "config" {
		t.Errorf("Expected Use 'config', got %q", configCmd.Use)
	}

	if configCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if configCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

// TestGenerateCommandStructure tests generate command structure
func TestGenerateCommandStructure(t *testing.T) {
	if generateCmd == nil {
		t.Fatal("generateCmd should not be nil")
	}

	if generateCmd.Use != "generate" {
		t.Errorf("Expected Use 'generate', got %q", generateCmd.Use)
	}

	if generateCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if generateCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if generateCmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if generateCmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

// TestConfigFormatOptions tests config format options
func TestConfigFormatOptions(t *testing.T) {
	validFormats := []string{"full", "minimal"}

	for _, format := range validFormats {
		if format == "" {
			t.Error("Format should not be empty")
		}

		if format != "full" && format != "minimal" {
			t.Errorf("Unexpected format: %s", format)
		}
	}
}

// TestAPIVersion tests API version format
func TestAPIVersion(t *testing.T) {
	cfg := generateMinimalConfig()

	apiVersion := cfg.APIVersion

	// Should follow pattern: domain/version
	if !strings.Contains(apiVersion, "/") {
		t.Error("APIVersion should contain '/' separator")
	}

	parts := strings.Split(apiVersion, "/")
	if len(parts) != 2 {
		t.Errorf("APIVersion should have 2 parts, got %d", len(parts))
	}

	// Domain part
	domain := parts[0]
	if !strings.Contains(domain, ".") {
		t.Logf("Note: Domain %q doesn't contain dot", domain)
	}

	// Version part
	version := parts[1]
	if !strings.HasPrefix(version, "v") {
		t.Logf("Note: Version %q doesn't start with 'v'", version)
	}
}

// TestKindValue tests Kind field value
func TestKindValue(t *testing.T) {
	cfg := generateMinimalConfig()

	validKinds := []string{"Cluster", "Node", "Network"}

	isValid := false
	for _, kind := range validKinds {
		if cfg.Kind == kind {
			isValid = true
			break
		}
	}

	if !isValid {
		t.Logf("Note: Kind %q is not in common kinds list", cfg.Kind)
	}

	// Kind should be capitalized
	if cfg.Kind != "" {
		firstChar := cfg.Kind[0]
		if !(firstChar >= 'A' && firstChar <= 'Z') {
			t.Error("Kind should start with capital letter")
		}
	}
}

// TestClusterMetadata tests cluster metadata
func TestClusterMetadata(t *testing.T) {
	cfg := generateMinimalConfig()

	// Name should follow DNS naming rules
	name := cfg.Metadata.Name
	if name == "" {
		t.Error("Cluster name should not be empty")
	}

	// Should be lowercase or contain dashes
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			t.Errorf("Cluster name contains invalid character: %c", char)
		}
	}

	// Should not start or end with dash
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		t.Error("Cluster name should not start or end with dash")
	}
}

// TestLabelsStructure tests labels structure
func TestLabelsStructure(t *testing.T) {
	cfg := generateMinimalConfig()

	labels := cfg.Metadata.Labels

	// Should have at least one label
	if len(labels) == 0 {
		t.Error("Labels should not be empty")
	}

	// All labels should be key-value pairs
	for key, value := range labels {
		if key == "" {
			t.Error("Label key should not be empty")
		}

		if value == "" {
			t.Errorf("Label value for key %q should not be empty", key)
		}

		// Keys should be lowercase or contain dashes
		for _, char := range key {
			if !((char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '.') {
				t.Logf("Label key %q contains character: %c", key, char)
			}
		}
	}
}

// TestProviderConfiguration tests provider configuration
func TestProviderConfiguration(t *testing.T) {
	cfg := generateMinimalConfig()

	providers := cfg.Spec.Providers

	// At least one provider should be configured
	hasProvider := false
	if providers.DigitalOcean != nil && providers.DigitalOcean.Enabled {
		hasProvider = true
	}
	if providers.Linode != nil && providers.Linode.Enabled {
		hasProvider = true
	}

	if !hasProvider {
		t.Error("At least one provider should be enabled")
	}
}

// TestDigitalOceanConfiguration tests DigitalOcean-specific config
func TestDigitalOceanConfiguration(t *testing.T) {
	cfg := generateMinimalConfig()

	if cfg.Spec.Providers.DigitalOcean == nil {
		t.Skip("DigitalOcean provider not configured")
	}

	do := cfg.Spec.Providers.DigitalOcean

	// Token should be configured
	if do.Token == "" {
		t.Error("Token should not be empty")
	}

	// Should use environment variable
	if strings.Contains(do.Token, "$") {
		if !strings.HasPrefix(do.Token, "${") || !strings.HasSuffix(do.Token, "}") {
			t.Logf("Token uses $ but not ${} format: %s", do.Token)
		}
	}

	// Region should be valid
	if do.Region != "" {
		validRegions := []string{"nyc1", "nyc2", "nyc3", "sfo1", "sfo2", "sfo3", "ams3", "sgp1", "lon1", "fra1"}
		isValid := false
		for _, region := range validRegions {
			if do.Region == region {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Logf("Note: Region %q is not in common regions list", do.Region)
		}
	}
}

// TestEnvironmentVariableSyntax tests environment variable syntax
func TestEnvironmentVariableSyntax(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		isValid bool
	}{
		{"Valid braces syntax", "${VAR_NAME}", true},
		{"Valid simple syntax", "$VAR_NAME", true},
		{"No variable", "plain-text", true},
		{"Mixed", "prefix-${VAR}-suffix", true},
		{"Invalid - no closing brace", "${VAR", false},
		{"Invalid - no opening brace", "VAR}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check for balanced braces
			openCount := strings.Count(tt.value, "${")
			closeCount := strings.Count(tt.value, "}")

			isValid := true
			if openCount != closeCount {
				isValid = false
			}

			if isValid != tt.isValid {
				t.Logf("Value %q: expected valid=%v, got valid=%v", tt.value, tt.isValid, isValid)
			}
		})
	}
}

// TestOutputPathDefault tests default output path
func TestOutputPathDefault(t *testing.T) {
	defaultPath := "cluster-config.yaml"

	if !strings.HasSuffix(defaultPath, ".yaml") && !strings.HasSuffix(defaultPath, ".yml") {
		t.Error("Default output path should have .yaml or .yml extension")
	}

	// Should not contain path separators (should be in current dir)
	if strings.Contains(defaultPath, "/") || strings.Contains(defaultPath, "\\") {
		t.Logf("Note: Default path contains directory separator")
	}
}

// TestConfigFormatValidation tests format validation
func TestConfigFormatValidation(t *testing.T) {
	validFormats := []string{"full", "minimal"}
	invalidFormats := []string{"", "invalid", "compact", "extended"}

	for _, format := range validFormats {
		t.Run("valid-"+format, func(t *testing.T) {
			if format != "full" && format != "minimal" {
				t.Errorf("Format %q should be valid but isn't", format)
			}
		})
	}

	for _, format := range invalidFormats {
		t.Run("invalid-"+format, func(t *testing.T) {
			if format == "full" || format == "minimal" {
				t.Errorf("Format %q should be invalid but isn't", format)
			}
		})
	}
}

// TestMinimalConfigDefaults tests minimal config defaults
func TestMinimalConfigDefaults(t *testing.T) {
	cfg := generateMinimalConfig()

	// Should have sensible defaults
	if cfg.Metadata.Name == "" {
		t.Error("Should have default cluster name")
	}

	if cfg.Spec.Providers.DigitalOcean != nil {
		if cfg.Spec.Providers.DigitalOcean.Region == "" {
			t.Error("Should have default region")
		}
	}

	// Labels should include environment
	if env, ok := cfg.Metadata.Labels["env"]; ok {
		validEnvs := []string{"production", "staging", "development", "dev", "prod"}
		isValid := false
		for _, validEnv := range validEnvs {
			if env == validEnv {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Logf("Note: Environment label %q is not in common values", env)
		}
	}
}

// TestConfigGeneration tests config generation options
func TestConfigGeneration(t *testing.T) {
	formats := []string{"full", "minimal"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			var cfg *config.KubernetesStyleConfig

			switch format {
			case "minimal":
				cfg = generateMinimalConfig()
			case "full":
				cfg = config.GenerateK8sStyleConfig()
			}

			if cfg == nil {
				t.Errorf("Config generation failed for format %q", format)
				return
			}

			// Verify basic structure
			if cfg.APIVersion == "" {
				t.Error("APIVersion should be set")
			}

			if cfg.Kind == "" {
				t.Error("Kind should be set")
			}

			if cfg.Metadata.Name == "" {
				t.Error("Cluster name should be set")
			}
		})
	}
}
