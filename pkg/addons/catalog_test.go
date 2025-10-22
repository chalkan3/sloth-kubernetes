package addons

import (
	"strings"
	"testing"
)

// TestGetAddonCatalog tests the addon catalog retrieval
func TestGetAddonCatalog(t *testing.T) {
	catalog := GetAddonCatalog()

	if catalog == nil {
		t.Fatal("GetAddonCatalog should not return nil")
	}

	// Should have multiple addons
	if len(catalog) == 0 {
		t.Error("Catalog should contain addons")
	}

	// Verify expected addons exist
	expectedAddons := []string{
		"ingress-nginx",
		"cert-manager",
		"prometheus",
		"longhorn",
		"argocd",
		"loki",
		"metallb",
		"postgres-operator",
		"istio",
		"external-dns",
		"velero",
		"sealed-secrets",
	}

	for _, name := range expectedAddons {
		if _, exists := catalog[name]; !exists {
			t.Errorf("Expected addon %q to be in catalog", name)
		}
	}
}

// TestGetAddon tests retrieving specific addons
func TestGetAddon(t *testing.T) {
	tests := []struct {
		name       string
		addonName  string
		shouldFind bool
	}{
		{"Find ingress-nginx", "ingress-nginx", true},
		{"Find cert-manager", "cert-manager", true},
		{"Find prometheus", "prometheus", true},
		{"Find argocd", "argocd", true},
		{"Not find invalid", "invalid-addon", false},
		{"Not find empty", "", false},
		{"Not find random", "random-name-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon, exists := GetAddon(tt.addonName)

			if tt.shouldFind && !exists {
				t.Errorf("Expected to find addon %q", tt.addonName)
			}

			if !tt.shouldFind && exists {
				t.Errorf("Expected NOT to find addon %q", tt.addonName)
			}

			if exists && addon == nil {
				t.Error("Found addon should not be nil")
			}

			if exists && addon.Name != tt.addonName {
				t.Errorf("Expected addon name %q, got %q", tt.addonName, addon.Name)
			}
		})
	}
}

// TestAddonStructure tests addon structure completeness
func TestAddonStructure(t *testing.T) {
	catalog := GetAddonCatalog()

	for name, addon := range catalog {
		t.Run(name, func(t *testing.T) {
			// Name should match key
			if addon.Name != name {
				t.Errorf("Addon key %q doesn't match name %q", name, addon.Name)
			}

			// Required fields
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

			// Repository should be HTTPS URL
			if !strings.HasPrefix(addon.Repository, "https://") {
				t.Errorf("Repository should start with https://, got %q", addon.Repository)
			}

			// Website should be HTTPS URL (if present)
			if addon.Website != "" && !strings.HasPrefix(addon.Website, "https://") {
				t.Errorf("Website should start with https://, got %q", addon.Website)
			}

			// Docs should be HTTPS URL (if present)
			if addon.Docs != "" && !strings.HasPrefix(addon.Docs, "https://") {
				t.Errorf("Docs should start with https://, got %q", addon.Docs)
			}
		})
	}
}

// TestGetAddonsByCategory tests filtering by category
func TestGetAddonsByCategory(t *testing.T) {
	tests := []struct {
		name           string
		category       Category
		minExpected    int
		expectedAddons []string
	}{
		{
			name:           "Ingress category",
			category:       CategoryIngress,
			minExpected:    1,
			expectedAddons: []string{"ingress-nginx"},
		},
		{
			name:           "Security category",
			category:       CategorySecurity,
			minExpected:    2,
			expectedAddons: []string{"cert-manager", "sealed-secrets"},
		},
		{
			name:           "Monitoring category",
			category:       CategoryMonitoring,
			minExpected:    1,
			expectedAddons: []string{"prometheus"},
		},
		{
			name:           "Storage category",
			category:       CategoryStorage,
			minExpected:    2,
			expectedAddons: []string{"longhorn", "velero"},
		},
		{
			name:           "CD category",
			category:       CategoryCD,
			minExpected:    1,
			expectedAddons: []string{"argocd"},
		},
		{
			name:           "Logging category",
			category:       CategoryLogging,
			minExpected:    1,
			expectedAddons: []string{"loki"},
		},
		{
			name:           "Networking category",
			category:       CategoryNetworking,
			minExpected:    3,
			expectedAddons: []string{"metallb", "istio", "external-dns"},
		},
		{
			name:           "Database category",
			category:       CategoryDatabase,
			minExpected:    1,
			expectedAddons: []string{"postgres-operator"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addons := GetAddonsByCategory(tt.category)

			if len(addons) < tt.minExpected {
				t.Errorf("Expected at least %d addons, got %d", tt.minExpected, len(addons))
			}

			// Verify expected addons are present
			foundAddons := make(map[string]bool)
			for _, addon := range addons {
				foundAddons[addon.Name] = true

				// Verify category matches
				if addon.Category != string(tt.category) {
					t.Errorf("Addon %q has category %q, expected %q",
						addon.Name, addon.Category, tt.category)
				}
			}

			// Check expected addons are found
			for _, expectedName := range tt.expectedAddons {
				if !foundAddons[expectedName] {
					t.Errorf("Expected addon %q not found in category %q",
						expectedName, tt.category)
				}
			}
		})
	}
}

// TestGetCategories tests getting all categories
func TestGetCategories(t *testing.T) {
	categories := GetCategories()

	if len(categories) == 0 {
		t.Error("Should have categories")
	}

	expectedCategories := []string{
		string(CategoryIngress),
		string(CategoryStorage),
		string(CategoryMonitoring),
		string(CategorySecurity),
		string(CategoryNetworking),
		string(CategoryCD),
		string(CategoryLogging),
		string(CategoryDatabase),
	}

	if len(categories) != len(expectedCategories) {
		t.Errorf("Expected %d categories, got %d", len(expectedCategories), len(categories))
	}

	// Verify all expected categories are present
	categoryMap := make(map[string]bool)
	for _, cat := range categories {
		categoryMap[cat] = true
	}

	for _, expected := range expectedCategories {
		if !categoryMap[expected] {
			t.Errorf("Expected category %q not found", expected)
		}
	}
}

// TestAddonVersions tests that versions are in expected format
func TestAddonVersions(t *testing.T) {
	catalog := GetAddonCatalog()

	for name, addon := range catalog {
		t.Run(name, func(t *testing.T) {
			version := addon.Version

			// Version should not be empty
			if version == "" {
				t.Error("Version should not be empty")
			}

			// Most versions should contain a number
			hasNumber := false
			for _, char := range version {
				if char >= '0' && char <= '9' {
					hasNumber = true
					break
				}
			}

			if !hasNumber {
				t.Errorf("Version %q should contain at least one number", version)
			}
		})
	}
}

// TestAddonNamespaces tests that namespaces are valid Kubernetes names
func TestAddonNamespaces(t *testing.T) {
	catalog := GetAddonCatalog()

	for name, addon := range catalog {
		t.Run(name, func(t *testing.T) {
			ns := addon.Namespace

			// Namespace should not be empty
			if ns == "" {
				t.Error("Namespace should not be empty")
			}

			// Should be lowercase
			if strings.ToLower(ns) != ns {
				t.Errorf("Namespace %q should be lowercase", ns)
			}

			// Should not start or end with dash
			if strings.HasPrefix(ns, "-") || strings.HasSuffix(ns, "-") {
				t.Errorf("Namespace %q should not start or end with dash", ns)
			}

			// Should only contain valid characters
			for _, char := range ns {
				if !((char >= 'a' && char <= 'z') ||
					(char >= '0' && char <= '9') ||
					char == '-') {
					t.Errorf("Namespace %q contains invalid character: %c", ns, char)
				}
			}
		})
	}
}

// TestSpecificAddons tests specific addon configurations
func TestSpecificAddons(t *testing.T) {
	tests := []struct {
		name      string
		addonName string
		category  Category
		chart     string
		namespace string
	}{
		{
			name:      "ingress-nginx",
			addonName: "ingress-nginx",
			category:  CategoryIngress,
			chart:     "ingress-nginx",
			namespace: "ingress-nginx",
		},
		{
			name:      "cert-manager",
			addonName: "cert-manager",
			category:  CategorySecurity,
			chart:     "cert-manager",
			namespace: "cert-manager",
		},
		{
			name:      "prometheus",
			addonName: "prometheus",
			category:  CategoryMonitoring,
			chart:     "kube-prometheus-stack",
			namespace: "monitoring",
		},
		{
			name:      "argocd",
			addonName: "argocd",
			category:  CategoryCD,
			chart:     "argo-cd",
			namespace: "argocd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon, exists := GetAddon(tt.addonName)

			if !exists {
				t.Fatalf("Addon %q should exist", tt.addonName)
			}

			if addon.Category != string(tt.category) {
				t.Errorf("Expected category %q, got %q", tt.category, addon.Category)
			}

			if addon.Chart != tt.chart {
				t.Errorf("Expected chart %q, got %q", tt.chart, addon.Chart)
			}

			if addon.Namespace != tt.namespace {
				t.Errorf("Expected namespace %q, got %q", tt.namespace, addon.Namespace)
			}
		})
	}
}

// TestCatalogImmutability tests that catalog is consistently returned
func TestCatalogImmutability(t *testing.T) {
	catalog1 := GetAddonCatalog()
	catalog2 := GetAddonCatalog()

	if len(catalog1) != len(catalog2) {
		t.Error("Catalog should return consistent results")
	}

	// Verify same addons are present
	for name := range catalog1 {
		if _, exists := catalog2[name]; !exists {
			t.Errorf("Addon %q should be in both catalogs", name)
		}
	}
}

// TestEmptyCategoryReturnsEmpty tests filtering by invalid category
func TestEmptyCategoryReturnsEmpty(t *testing.T) {
	// Create an invalid category
	invalidCategory := Category("invalid-category-name")

	addons := GetAddonsByCategory(invalidCategory)

	// Can be nil or empty slice - both are acceptable
	if len(addons) != 0 {
		t.Errorf("Expected 0 addons for invalid category, got %d", len(addons))
	}
}
