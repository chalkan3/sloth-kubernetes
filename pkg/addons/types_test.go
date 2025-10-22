package addons

import (
	"testing"
)

func TestAddon_Struct(t *testing.T) {
	addon := &Addon{
		Name:         "nginx-ingress",
		DisplayName:  "NGINX Ingress Controller",
		Description:  "Ingress controller for Kubernetes",
		Category:     "ingress",
		Version:      "4.7.1",
		Chart:        "ingress-nginx",
		Repository:   "https://kubernetes.github.io/ingress-nginx",
		Namespace:    "ingress-nginx",
		Dependencies: []string{"cert-manager"},
		Values:       map[string]interface{}{"replicas": 3},
		InstallCmd:   "",
		Website:      "https://kubernetes.github.io/ingress-nginx/",
		Docs:         "https://kubernetes.github.io/ingress-nginx/deploy/",
	}

	if addon.Name != "nginx-ingress" {
		t.Errorf("Expected name 'nginx-ingress', got '%s'", addon.Name)
	}

	if addon.DisplayName != "NGINX Ingress Controller" {
		t.Errorf("Expected display name 'NGINX Ingress Controller', got '%s'", addon.DisplayName)
	}

	if addon.Category != "ingress" {
		t.Errorf("Expected category 'ingress', got '%s'", addon.Category)
	}

	if len(addon.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(addon.Dependencies))
	}

	if addon.Dependencies[0] != "cert-manager" {
		t.Errorf("Expected dependency 'cert-manager', got '%s'", addon.Dependencies[0])
	}

	if addon.Values["replicas"] != 3 {
		t.Errorf("Expected replicas value 3, got %v", addon.Values["replicas"])
	}
}

func TestAddonStatus_Struct(t *testing.T) {
	status := &AddonStatus{
		Name:      "nginx-ingress",
		Installed: true,
		Version:   "4.7.1",
		Namespace: "ingress-nginx",
		Status:    "Running",
		Pods:      3,
		Ready:     3,
	}

	if status.Name != "nginx-ingress" {
		t.Errorf("Expected name 'nginx-ingress', got '%s'", status.Name)
	}

	if !status.Installed {
		t.Error("Expected Installed to be true")
	}

	if status.Status != "Running" {
		t.Errorf("Expected status 'Running', got '%s'", status.Status)
	}

	if status.Pods != 3 {
		t.Errorf("Expected 3 pods, got %d", status.Pods)
	}

	if status.Ready != 3 {
		t.Errorf("Expected 3 ready pods, got %d", status.Ready)
	}
}

func TestCategory_Constants(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		expected string
	}{
		{"Ingress category", CategoryIngress, "ingress"},
		{"Storage category", CategoryStorage, "storage"},
		{"Monitoring category", CategoryMonitoring, "monitoring"},
		{"Security category", CategorySecurity, "security"},
		{"Networking category", CategoryNetworking, "networking"},
		{"CD category", CategoryCD, "cd"},
		{"Logging category", CategoryLogging, "logging"},
		{"Database category", CategoryDatabase, "database"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.category) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(tt.category))
			}
		})
	}
}

func TestAddon_EmptyValues(t *testing.T) {
	addon := &Addon{
		Name:   "test-addon",
		Values: make(map[string]interface{}),
	}

	if addon.Values == nil {
		t.Error("Values map should not be nil")
	}

	if len(addon.Values) != 0 {
		t.Errorf("Expected empty values map, got %d entries", len(addon.Values))
	}
}

func TestAddon_NilValues(t *testing.T) {
	addon := &Addon{
		Name:   "test-addon",
		Values: nil,
	}

	if addon.Values != nil {
		t.Error("Values should be nil when not initialized")
	}
}

func TestAddon_MultipleDependencies(t *testing.T) {
	addon := &Addon{
		Name:         "argocd",
		Dependencies: []string{"cert-manager", "ingress-nginx", "prometheus"},
	}

	if len(addon.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(addon.Dependencies))
	}

	expectedDeps := map[string]bool{
		"cert-manager":  true,
		"ingress-nginx": true,
		"prometheus":    true,
	}

	for _, dep := range addon.Dependencies {
		if !expectedDeps[dep] {
			t.Errorf("Unexpected dependency: %s", dep)
		}
	}
}

func TestAddon_NoDependencies(t *testing.T) {
	addon := &Addon{
		Name:         "standalone-addon",
		Dependencies: []string{},
	}

	if len(addon.Dependencies) != 0 {
		t.Errorf("Expected no dependencies, got %d", len(addon.Dependencies))
	}
}

func TestAddonStatus_NotInstalled(t *testing.T) {
	status := &AddonStatus{
		Name:      "prometheus",
		Installed: false,
		Status:    "Unknown",
		Pods:      0,
		Ready:     0,
	}

	if status.Installed {
		t.Error("Expected Installed to be false")
	}

	if status.Status != "Unknown" {
		t.Errorf("Expected status 'Unknown', got '%s'", status.Status)
	}

	if status.Pods != 0 {
		t.Errorf("Expected 0 pods for non-installed addon, got %d", status.Pods)
	}
}

func TestAddonStatus_PartiallyReady(t *testing.T) {
	status := &AddonStatus{
		Name:      "monitoring",
		Installed: true,
		Status:    "Pending",
		Pods:      5,
		Ready:     3,
	}

	if status.Pods != 5 {
		t.Errorf("Expected 5 total pods, got %d", status.Pods)
	}

	if status.Ready != 3 {
		t.Errorf("Expected 3 ready pods, got %d", status.Ready)
	}

	if status.Status != "Pending" {
		t.Errorf("Expected status 'Pending', got '%s'", status.Status)
	}
}

func TestAddonStatus_Failed(t *testing.T) {
	status := &AddonStatus{
		Name:      "failed-addon",
		Installed: true,
		Status:    "Failed",
		Pods:      2,
		Ready:     0,
	}

	if status.Status != "Failed" {
		t.Errorf("Expected status 'Failed', got '%s'", status.Status)
	}

	if status.Ready != 0 {
		t.Error("Expected 0 ready pods for failed addon")
	}
}

func TestAddon_ComplexValues(t *testing.T) {
	addon := &Addon{
		Name: "complex-addon",
		Values: map[string]interface{}{
			"replicas": 3,
			"enabled":  true,
			"config": map[string]interface{}{
				"host": "example.com",
				"port": 8080,
			},
			"resources": map[string]interface{}{
				"limits": map[string]string{
					"cpu":    "1000m",
					"memory": "512Mi",
				},
			},
		},
	}

	if addon.Values["replicas"] != 3 {
		t.Errorf("Expected replicas 3, got %v", addon.Values["replicas"])
	}

	if addon.Values["enabled"] != true {
		t.Errorf("Expected enabled true, got %v", addon.Values["enabled"])
	}

	config, ok := addon.Values["config"].(map[string]interface{})
	if !ok {
		t.Error("Expected config to be map[string]interface{}")
	}

	if config["host"] != "example.com" {
		t.Errorf("Expected host 'example.com', got %v", config["host"])
	}
}

func TestAddon_InstallCmdAlternative(t *testing.T) {
	addon := &Addon{
		Name:       "custom-addon",
		InstallCmd: "kubectl apply -f https://example.com/manifest.yaml",
		Chart:      "", // No Helm chart
	}

	if addon.InstallCmd == "" {
		t.Error("Expected non-empty InstallCmd")
	}

	if addon.Chart != "" {
		t.Error("Expected empty Chart when using InstallCmd")
	}
}

func TestAddon_HelmChart(t *testing.T) {
	addon := &Addon{
		Name:       "helm-addon",
		Chart:      "stable/nginx-ingress",
		Repository: "https://charts.helm.sh/stable",
		InstallCmd: "", // Empty when using Helm
	}

	if addon.Chart == "" {
		t.Error("Expected non-empty Chart")
	}

	if addon.Repository == "" {
		t.Error("Expected non-empty Repository")
	}

	if addon.InstallCmd != "" {
		t.Error("Expected empty InstallCmd when using Helm")
	}
}

func TestAddonStatus_AllStatuses(t *testing.T) {
	statuses := []string{"Running", "Pending", "Failed", "Unknown"}

	for _, s := range statuses {
		status := &AddonStatus{
			Name:   "test",
			Status: s,
		}

		if status.Status != s {
			t.Errorf("Expected status %q, got %q", s, status.Status)
		}
	}
}

func TestAddon_FullyPopulated(t *testing.T) {
	addon := &Addon{
		Name:         "argocd",
		DisplayName:  "Argo CD",
		Description:  "Declarative GitOps CD for Kubernetes",
		Category:     "cd",
		Version:      "5.36.1",
		Chart:        "argo-cd",
		Repository:   "https://argoproj.github.io/argo-helm",
		Namespace:    "argocd",
		Dependencies: []string{"ingress-nginx"},
		Values: map[string]interface{}{
			"server": map[string]interface{}{
				"ingress": map[string]interface{}{
					"enabled": true,
				},
			},
		},
		InstallCmd: "",
		Website:    "https://argo-cd.readthedocs.io/",
		Docs:       "https://argo-cd.readthedocs.io/en/stable/getting_started/",
	}

	// Verify all fields are populated
	if addon.Name == "" {
		t.Error("Name should not be empty")
	}
	if addon.DisplayName == "" {
		t.Error("DisplayName should not be empty")
	}
	if addon.Description == "" {
		t.Error("Description should not be empty")
	}
	if addon.Category == "" {
		t.Error("Category should not be empty")
	}
	if addon.Version == "" {
		t.Error("Version should not be empty")
	}
	if addon.Chart == "" {
		t.Error("Chart should not be empty")
	}
	if addon.Repository == "" {
		t.Error("Repository should not be empty")
	}
	if addon.Namespace == "" {
		t.Error("Namespace should not be empty")
	}
	if addon.Website == "" {
		t.Error("Website should not be empty")
	}
	if addon.Docs == "" {
		t.Error("Docs should not be empty")
	}
}

func TestCategory_TypeAssertion(t *testing.T) {
	var cat Category = CategoryIngress

	// Should be able to convert to string
	str := string(cat)
	if str != "ingress" {
		t.Errorf("Expected 'ingress', got %q", str)
	}

	// Should be able to compare
	if cat != CategoryIngress {
		t.Error("Category comparison failed")
	}
}
