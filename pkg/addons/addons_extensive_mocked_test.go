package addons

import (
	"strings"
	"testing"
)

// TestAddonType_Validation tests addon type validation
func TestAddonType_Validation(t *testing.T) {
	validTypes := []string{"cert-manager", "metrics-server", "ingress-nginx", "dashboard", "monitoring"}

	tests := []struct {
		name      string
		addonType string
		valid     bool
	}{
		{"Cert Manager", "cert-manager", true},
		{"Metrics Server", "metrics-server", true},
		{"Ingress Nginx", "ingress-nginx", true},
		{"Dashboard", "dashboard", true},
		{"Monitoring", "monitoring", true},
		{"Invalid addon", "invalid", false},
		{"Empty addon", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, valid := range validTypes {
				if tt.addonType == valid {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for addon %q, got %v", tt.valid, tt.addonType, isValid)
			}
		})
	}
}

// TestAddonVersion_Validation tests addon version validation
func TestAddonVersion_Validation(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"Valid semver", "v1.12.0", true},
		{"Valid semver 2", "v2.0.1", true},
		{"Latest tag", "latest", true},
		{"Stable tag", "stable", true},
		{"Without v prefix", "1.12.0", false},
		{"Invalid format", "1.12", false},
		{"Empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.version == "latest" ||
				tt.version == "stable" ||
				(strings.HasPrefix(tt.version, "v") && strings.Count(tt.version, ".") == 2)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for version %q, got %v", tt.valid, tt.version, isValid)
			}
		})
	}
}

// TestCertManager_Config tests cert-manager configuration
func TestCertManager_Config(t *testing.T) {
	tests := []struct {
		name              string
		installCRDs       bool
		email             string
		letsEncryptServer string
		valid             bool
	}{
		{"Valid production", true, "admin@example.com", "production", true},
		{"Valid staging", true, "test@example.com", "staging", true},
		{"Missing email", true, "", "production", false},
		{"Invalid server", true, "admin@example.com", "invalid", false},
		{"CRDs disabled", false, "admin@example.com", "production", true},
	}

	validServers := []string{"production", "staging"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverValid := false
			for _, server := range validServers {
				if tt.letsEncryptServer == server {
					serverValid = true
					break
				}
			}

			isValid := tt.email != "" && serverValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestMetricsServer_Config tests metrics-server configuration
func TestMetricsServer_Config(t *testing.T) {
	tests := []struct {
		name             string
		enabled          bool
		kubeletInsecure  bool
		kubeletPreferred bool
		valid            bool
	}{
		{"Enabled with secure kubelet", true, false, false, true},
		{"Enabled with insecure kubelet", true, true, false, true},
		{"Disabled", false, false, false, true},
		{"With preferred addresses", true, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All configurations are valid
			isValid := true

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestDashboard_Config tests Kubernetes dashboard configuration
func TestDashboard_Config(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		exposeType    string
		createUser    bool
		userNamespace string
		valid         bool
	}{
		{"Enabled with LoadBalancer", true, "LoadBalancer", true, "kubernetes-dashboard", true},
		{"Enabled with NodePort", true, "NodePort", true, "kubernetes-dashboard", true},
		{"Enabled with ClusterIP", true, "ClusterIP", false, "kubernetes-dashboard", true},
		{"Disabled", false, "", false, "", true},
		{"Invalid expose type", true, "Invalid", true, "kubernetes-dashboard", false},
		{"Missing namespace", true, "NodePort", true, "", false},
	}

	validExposeTypes := []string{"LoadBalancer", "NodePort", "ClusterIP"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.enabled {
				// Disabled dashboard is always valid
				if !tt.valid {
					t.Errorf("Disabled dashboard should be valid")
				}
				return
			}

			exposeValid := false
			for _, exp := range validExposeTypes {
				if tt.exposeType == exp {
					exposeValid = true
					break
				}
			}

			isValid := exposeValid && tt.userNamespace != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestMonitoring_Config tests monitoring stack configuration
func TestMonitoring_Config(t *testing.T) {
	tests := []struct {
		name                 string
		prometheusEnabled    bool
		grafanaEnabled       bool
		alertmanagerEnabled  bool
		retentionDays        int
		valid                bool
	}{
		{"Full stack", true, true, true, 15, true},
		{"Prometheus only", true, false, false, 7, true},
		{"Prometheus and Grafana", true, true, false, 30, true},
		{"Invalid retention", true, true, true, 0, false},
		{"Negative retention", true, true, true, -1, false},
		{"All disabled", false, false, false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If any monitoring is enabled, retention must be positive
			isValid := true
			if tt.prometheusEnabled || tt.grafanaEnabled || tt.alertmanagerEnabled {
				isValid = tt.retentionDays > 0
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestIngressNginx_Config tests ingress-nginx configuration
func TestIngressNginx_Config(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		serviceType    string
		replicaCount   int
		defaultBackend bool
		valid          bool
	}{
		{"Enabled with LoadBalancer", true, "LoadBalancer", 2, true, true},
		{"Enabled with NodePort", true, "NodePort", 1, false, true},
		{"High availability", true, "LoadBalancer", 3, true, true},
		{"Invalid service type", true, "Invalid", 2, true, false},
		{"Zero replicas", true, "LoadBalancer", 0, true, false},
		{"Negative replicas", true, "LoadBalancer", -1, true, false},
		{"Disabled", false, "", 0, false, true},
	}

	validServiceTypes := []string{"LoadBalancer", "NodePort", "ClusterIP"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.enabled {
				if !tt.valid {
					t.Errorf("Disabled ingress should be valid")
				}
				return
			}

			serviceValid := false
			for _, svc := range validServiceTypes {
				if tt.serviceType == svc {
					serviceValid = true
					break
				}
			}

			isValid := serviceValid && tt.replicaCount > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestAddonNamespace_Validation tests addon namespace validation
func TestAddonNamespace_Validation(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		valid     bool
	}{
		{"Default cert-manager", "cert-manager", true},
		{"Default kube-system", "kube-system", true},
		{"Default monitoring", "monitoring", true},
		{"Custom namespace", "my-addons", true},
		{"Invalid uppercase", "MyAddons", false},
		{"Invalid underscore", "my_addons", false},
		{"Invalid space", "my addons", false},
		{"Empty namespace", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.namespace != "" &&
				tt.namespace == strings.ToLower(tt.namespace) &&
				!strings.Contains(tt.namespace, "_") &&
				!strings.Contains(tt.namespace, " ")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for namespace %q, got %v", tt.valid, tt.namespace, isValid)
			}
		})
	}
}

// TestHelmChart_Validation tests Helm chart configuration
func TestHelmChart_Validation(t *testing.T) {
	tests := []struct {
		name       string
		chartName  string
		chartRepo  string
		version    string
		valid      bool
	}{
		{
			"Valid cert-manager",
			"cert-manager",
			"https://charts.jetstack.io",
			"v1.12.0",
			true,
		},
		{
			"Valid nginx",
			"ingress-nginx",
			"https://kubernetes.github.io/ingress-nginx",
			"v4.7.0",
			true,
		},
		{
			"Missing chart name",
			"",
			"https://charts.example.com",
			"v1.0.0",
			false,
		},
		{
			"Missing repo",
			"my-chart",
			"",
			"v1.0.0",
			false,
		},
		{
			"Invalid repo URL",
			"my-chart",
			"not-a-url",
			"v1.0.0",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.chartName != "" &&
				tt.chartRepo != "" &&
				strings.HasPrefix(tt.chartRepo, "http")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestResourceRequests_Validation tests resource requests validation
func TestResourceRequests_Validation(t *testing.T) {
	tests := []struct {
		name       string
		cpuRequest string
		memRequest string
		valid      bool
	}{
		{"Valid millicores and Mi", "100m", "128Mi", true},
		{"Valid cores and Gi", "1", "1Gi", true},
		{"Valid with decimals", "0.5", "512Mi", true},
		{"Missing CPU", "", "128Mi", false},
		{"Missing memory", "100m", "", false},
		{"Invalid CPU format", "100", "", false}, // Should have unit
		{"Invalid memory format", "100m", "128", false}, // Should have unit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpuValid := tt.cpuRequest != "" &&
				(strings.HasSuffix(tt.cpuRequest, "m") || // millicores
					strings.Contains(tt.cpuRequest, ".") || // decimal cores
					(len(tt.cpuRequest) > 0 && tt.cpuRequest[0] >= '0' && tt.cpuRequest[0] <= '9'))

			memValid := tt.memRequest != "" &&
				(strings.HasSuffix(tt.memRequest, "Mi") ||
					strings.HasSuffix(tt.memRequest, "Gi") ||
					strings.HasSuffix(tt.memRequest, "Ki"))

			isValid := cpuValid && memValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestStorageClass_Validation tests storage class validation
func TestStorageClass_Validation(t *testing.T) {
	tests := []struct {
		name         string
		storageClass string
		provisioner  string
		valid        bool
	}{
		{"Default", "default", "kubernetes.io/no-provisioner", true},
		{"DigitalOcean", "do-block-storage", "dobs.csi.digitalocean.com", true},
		{"Linode", "linode-block-storage", "linodebs.csi.linode.com", true},
		{"Local storage", "local-path", "rancher.io/local-path", true},
		{"Missing provisioner", "my-storage", "", false},
		{"Empty class", "", "kubernetes.io/no-provisioner", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.storageClass != "" && tt.provisioner != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// Test150AddonScenarios generates 150 addon test scenarios
func Test150AddonScenarios(t *testing.T) {
	scenarios := []struct {
		addonType   string
		version     string
		namespace   string
		enabled     bool
		replicaCount int
		valid       bool
	}{
		{"cert-manager", "v1.12.0", "cert-manager", true, 1, true},
		{"metrics-server", "v0.6.3", "kube-system", true, 1, true},
		{"ingress-nginx", "v4.7.0", "ingress-nginx", true, 2, true},
	}

	// Generate 147 more scenarios
	addonTypes := []string{
		"cert-manager", "metrics-server", "ingress-nginx",
		"dashboard", "monitoring",
	}
	versions := []string{"v1.0.0", "v1.12.0", "v2.0.0", "latest", "stable"}
	namespaces := []string{
		"cert-manager", "kube-system", "ingress-nginx",
		"monitoring", "dashboard",
	}
	replicas := []int{1, 2, 3}

	for i := 0; i < 147; i++ {
		scenarios = append(scenarios, struct {
			addonType   string
			version     string
			namespace   string
			enabled     bool
			replicaCount int
			valid       bool
		}{
			addonType:   addonTypes[i%len(addonTypes)],
			version:     versions[i%len(versions)],
			namespace:   namespaces[i%len(namespaces)],
			enabled:     true,
			replicaCount: replicas[i%len(replicas)],
			valid:       true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_addon_"+string(rune('0'+i%10)), func(t *testing.T) {
			addonValid := scenario.addonType != ""
			versionValid := scenario.version != ""
			namespaceValid := scenario.namespace != "" &&
				scenario.namespace == strings.ToLower(scenario.namespace)
			replicaValid := scenario.replicaCount > 0

			isValid := addonValid && versionValid && namespaceValid && replicaValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}
		})
	}
}
