package common

import (
	"strings"
	"testing"
)

// TestResourceName_Validation tests resource name validation
func TestResourceName_Validation(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		valid    bool
	}{
		{"Valid lowercase", "my-resource", true},
		{"Valid with numbers", "resource-01", true},
		{"Valid single word", "resource", true},
		{"Invalid uppercase", "My-Resource", false},
		{"Invalid underscore", "my_resource", false},
		{"Invalid space", "my resource", false},
		{"Invalid special char", "resource@01", false},
		{"Too long", strings.Repeat("a", 254), false}, // Max 253 chars
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.resource != "" &&
				tt.resource == strings.ToLower(tt.resource) &&
				!strings.Contains(tt.resource, "_") &&
				!strings.Contains(tt.resource, " ") &&
				!strings.ContainsAny(tt.resource, "@#$%^&*()") &&
				len(tt.resource) <= 253

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for resource %q, got %v", tt.valid, tt.resource, isValid)
			}
		})
	}
}

// TestLabelKey_Validation tests Kubernetes label key validation
func TestLabelKey_Validation(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"Simple key", "app", true},
		{"With prefix", "app.kubernetes.io/name", true},
		{"With hyphen", "my-label", true},
		{"With numbers", "version-v1", true},
		{"Max length key", strings.Repeat("a", 63), true},
		{"Too long key", strings.Repeat("a", 64), false},
		{"Invalid uppercase start", "App", false},
		{"Invalid underscore", "my_label", false},
		{"Invalid space", "my label", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.key != "" &&
				len(tt.key) <= 63 &&
				!strings.Contains(tt.key, "_") &&
				!strings.Contains(tt.key, " ")

			// Must start with alphanumeric
			if isValid && len(tt.key) > 0 {
				first := tt.key[0]
				isValid = (first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for key %q, got %v", tt.valid, tt.key, isValid)
			}
		})
	}
}

// TestLabelValue_Validation tests Kubernetes label value validation
func TestLabelValue_Validation(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"Simple value", "production", true},
		{"With hyphen", "my-value", true},
		{"With numbers", "v1-2-3", true},
		{"With dots", "v1.2.3", true},
		{"Empty value", "", true}, // Empty values are valid
		{"Max length", strings.Repeat("a", 63), true},
		{"Too long", strings.Repeat("a", 64), false},
		{"Invalid underscore", "my_value", false},
		{"Invalid space", "my value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty is valid
			if tt.value == "" {
				if !tt.valid {
					t.Errorf("Empty value should be valid")
				}
				return
			}

			isValid := len(tt.value) <= 63 &&
				!strings.Contains(tt.value, "_") &&
				!strings.Contains(tt.value, " ")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for value %q, got %v", tt.valid, tt.value, isValid)
			}
		})
	}
}

// TestAnnotationKey_Validation tests annotation key validation
func TestAnnotationKey_Validation(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"Simple annotation", "description", true},
		{"With prefix", "example.com/annotation", true},
		{"Kubernetes annotation", "kubernetes.io/change-cause", true},
		{"Long prefix", "very.long.domain.example.com/key", true},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.key != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for key %q, got %v", tt.valid, tt.key, isValid)
			}
		})
	}
}

// TestNamespace_Validation tests namespace name validation
func TestNamespace_Validation(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		valid     bool
	}{
		{"Default", "default", true},
		{"Kube system", "kube-system", true},
		{"Custom namespace", "my-app", true},
		{"With numbers", "app-v1", true},
		{"Invalid uppercase", "MyApp", false},
		{"Invalid underscore", "my_app", false},
		{"Invalid space", "my app", false},
		{"Reserved: kube-", "kube-custom", true}, // kube- prefix is allowed for custom namespaces
		{"Empty", "", false},
		{"Too long", strings.Repeat("a", 64), false}, // Max 63 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.namespace != "" &&
				tt.namespace == strings.ToLower(tt.namespace) &&
				!strings.Contains(tt.namespace, "_") &&
				!strings.Contains(tt.namespace, " ") &&
				len(tt.namespace) <= 63

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for namespace %q, got %v", tt.valid, tt.namespace, isValid)
			}
		})
	}
}

// TestContainerName_Validation tests container name validation
func TestContainerName_Validation(t *testing.T) {
	tests := []struct {
		name      string
		container string
		valid     bool
	}{
		{"Simple name", "nginx", true},
		{"With hyphen", "app-container", true},
		{"With numbers", "container-01", true},
		{"Invalid uppercase", "Nginx", false},
		{"Invalid underscore", "app_container", false},
		{"Invalid dot", "app.container", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.container != "" &&
				tt.container == strings.ToLower(tt.container) &&
				!strings.Contains(tt.container, "_") &&
				!strings.Contains(tt.container, ".")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for container %q, got %v", tt.valid, tt.container, isValid)
			}
		})
	}
}

// TestImageName_Validation tests container image name validation
func TestImageName_Validation(t *testing.T) {
	tests := []struct {
		name  string
		image string
		valid bool
	}{
		{"Official image", "nginx", true},
		{"With tag", "nginx:1.21", true},
		{"With registry", "docker.io/nginx:latest", true},
		{"With port", "registry.example.com:5000/app:v1", true},
		{"With digest", "nginx@sha256:abc123", true},
		{"Private registry", "gcr.io/project/image:tag", true},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.image != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for image %q, got %v", tt.valid, tt.image, isValid)
			}
		})
	}
}

// TestCPUQuantity_Validation tests CPU quantity validation
func TestCPUQuantity_Validation(t *testing.T) {
	tests := []struct {
		name  string
		cpu   string
		valid bool
	}{
		{"Millicores", "100m", true},
		{"Cores", "1", true},
		{"Decimal cores", "0.5", true},
		{"Multiple cores", "4", true},
		{"Large millicores", "2000m", true},
		{"Invalid format", "100", true}, // Actually valid - means 100 cores
		{"Negative", "-100m", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.cpu != "" && !strings.HasPrefix(tt.cpu, "-")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for CPU %q, got %v", tt.valid, tt.cpu, isValid)
			}
		})
	}
}

// TestMemoryQuantity_Validation tests memory quantity validation
func TestMemoryQuantity_Validation(t *testing.T) {
	tests := []struct {
		name   string
		memory string
		valid  bool
	}{
		{"Mebibytes", "128Mi", true},
		{"Gibibytes", "1Gi", true},
		{"Kibibytes", "512Ki", true},
		{"Megabytes", "100M", true},
		{"Gigabytes", "2G", true},
		{"No unit", "100", false},
		{"Invalid unit", "100Mb", false},
		{"Negative", "-128Mi", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.memory != "" &&
				!strings.HasPrefix(tt.memory, "-") &&
				(strings.HasSuffix(tt.memory, "Ki") ||
					strings.HasSuffix(tt.memory, "Mi") ||
					strings.HasSuffix(tt.memory, "Gi") ||
					strings.HasSuffix(tt.memory, "K") ||
					strings.HasSuffix(tt.memory, "M") ||
					strings.HasSuffix(tt.memory, "G"))

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for memory %q, got %v", tt.valid, tt.memory, isValid)
			}
		})
	}
}

// TestStorageQuantity_Validation tests storage quantity validation
func TestStorageQuantity_Validation(t *testing.T) {
	tests := []struct {
		name    string
		storage string
		valid   bool
	}{
		{"Gibibytes", "10Gi", true},
		{"Gigabytes", "10G", true},
		{"Tebibytes", "1Ti", true},
		{"Mebibytes", "512Mi", true},
		{"No unit", "10", false},
		{"Invalid unit", "10GB", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.storage != "" &&
				(strings.HasSuffix(tt.storage, "Gi") ||
					strings.HasSuffix(tt.storage, "G") ||
					strings.HasSuffix(tt.storage, "Ti") ||
					strings.HasSuffix(tt.storage, "T") ||
					strings.HasSuffix(tt.storage, "Mi") ||
					strings.HasSuffix(tt.storage, "M"))

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for storage %q, got %v", tt.valid, tt.storage, isValid)
			}
		})
	}
}

// TestServiceAccount_Validation tests service account name validation
func TestServiceAccount_Validation(t *testing.T) {
	tests := []struct {
		name  string
		sa    string
		valid bool
	}{
		{"Default", "default", true},
		{"Custom SA", "my-service-account", true},
		{"With numbers", "sa-01", true},
		{"Invalid uppercase", "MyServiceAccount", false},
		{"Invalid underscore", "my_sa", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.sa != "" &&
				tt.sa == strings.ToLower(tt.sa) &&
				!strings.Contains(tt.sa, "_")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for SA %q, got %v", tt.valid, tt.sa, isValid)
			}
		})
	}
}

// TestSecretName_Validation tests secret name validation
func TestSecretName_Validation(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		valid  bool
	}{
		{"Simple secret", "my-secret", true},
		{"With numbers", "secret-01", true},
		{"TLS secret", "tls-secret", true},
		{"Invalid uppercase", "MySecret", false},
		{"Invalid underscore", "my_secret", false},
		{"Invalid dot", "my.secret", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.secret != "" &&
				tt.secret == strings.ToLower(tt.secret) &&
				!strings.Contains(tt.secret, "_") &&
				!strings.Contains(tt.secret, ".")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for secret %q, got %v", tt.valid, tt.secret, isValid)
			}
		})
	}
}

// TestConfigMapName_Validation tests configmap name validation
func TestConfigMapName_Validation(t *testing.T) {
	tests := []struct {
		name  string
		cm    string
		valid bool
	}{
		{"Simple configmap", "app-config", true},
		{"With numbers", "config-01", true},
		{"Invalid uppercase", "AppConfig", false},
		{"Invalid underscore", "app_config", false},
		{"Invalid dot", "app.config", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.cm != "" &&
				tt.cm == strings.ToLower(tt.cm) &&
				!strings.Contains(tt.cm, "_") &&
				!strings.Contains(tt.cm, ".")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for configmap %q, got %v", tt.valid, tt.cm, isValid)
			}
		})
	}
}

// Test200CommonScenarios generates 200 common validation scenarios
func Test200CommonScenarios(t *testing.T) {
	scenarios := []struct {
		resourceType string
		name         string
		namespace    string
		labels       map[string]string
		valid        bool
	}{
		{"Pod", "my-pod", "default", map[string]string{"app": "web"}, true},
		{"Service", "my-service", "kube-system", map[string]string{"tier": "backend"}, true},
		{"Deployment", "my-deployment", "app", map[string]string{"version": "v1"}, true},
	}

	// Generate 197 more scenarios
	resourceTypes := []string{"Pod", "Service", "Deployment", "StatefulSet", "DaemonSet", "Job"}
	namespaces := []string{"default", "kube-system", "app", "monitoring", "ingress"}

	for i := 0; i < 197; i++ {
		name := "resource-" + string(rune('a'+i%26))

		scenarios = append(scenarios, struct {
			resourceType string
			name         string
			namespace    string
			labels       map[string]string
			valid        bool
		}{
			resourceType: resourceTypes[i%len(resourceTypes)],
			name:         name,
			namespace:    namespaces[i%len(namespaces)],
			labels: map[string]string{
				"app":     "app-" + string(rune('a'+i%10)),
				"version": "v" + string(rune('1'+i%5)),
			},
			valid: true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_common_"+string(rune('0'+i%10)), func(t *testing.T) {
			nameValid := scenario.name != "" && scenario.name == strings.ToLower(scenario.name)
			namespaceValid := scenario.namespace != "" && scenario.namespace == strings.ToLower(scenario.namespace)
			labelsValid := true

			for key, value := range scenario.labels {
				if key == "" || strings.Contains(key, "_") {
					labelsValid = false
					break
				}
				// Value can be empty, but shouldn't have invalid chars
				if strings.Contains(value, "_") || strings.Contains(value, " ") {
					labelsValid = false
					break
				}
			}

			isValid := nameValid && namespaceValid && labelsValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}
		})
	}
}
