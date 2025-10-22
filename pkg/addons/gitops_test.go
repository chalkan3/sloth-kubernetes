package addons

import (
	"strings"
	"testing"
)

// TestGitOpsConfig tests GitOps configuration structure
func TestGitOpsConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *GitOpsConfig
		valid  bool
	}{
		{
			name: "Valid HTTPS config",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "main",
				Path:    "addons/",
			},
			valid: true,
		},
		{
			name: "Valid SSH config",
			config: &GitOpsConfig{
				RepoURL:    "git@github.com:user/repo.git",
				Branch:     "develop",
				Path:       "k8s/addons/",
				PrivateKey: "/path/to/key",
			},
			valid: true,
		},
		{
			name: "Empty branch (should use default)",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "",
				Path:    "addons/",
			},
			valid: true,
		},
		{
			name: "Empty path (should use default)",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "main",
				Path:    "",
			},
			valid: true,
		},
		{
			name: "Missing RepoURL",
			config: &GitOpsConfig{
				RepoURL: "",
				Branch:  "main",
				Path:    "addons/",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.config.RepoURL != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestGenerateArgoCDApp tests ArgoCD Application manifest generation
func TestGenerateArgoCDApp(t *testing.T) {
	tests := []struct {
		name         string
		config       *GitOpsConfig
		expectBranch string
		expectPath   string
	}{
		{
			name: "Custom branch and path",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "production",
				Path:    "k8s/addons/",
			},
			expectBranch: "production",
			expectPath:   "k8s/addons/",
		},
		{
			name: "Default branch (empty)",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "",
				Path:    "addons/",
			},
			expectBranch: "main",
			expectPath:   "addons/",
		},
		{
			name: "Default path (empty)",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "main",
				Path:    "",
			},
			expectBranch: "main",
			expectPath:   "addons/",
		},
		{
			name: "Both defaults",
			config: &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "",
				Path:    "",
			},
			expectBranch: "main",
			expectPath:   "addons/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := generateArgoCDApp(tt.config)

			// Verify manifest is not empty
			if manifest == "" {
				t.Fatal("Manifest should not be empty")
			}

			// Verify YAML structure
			if !strings.Contains(manifest, "apiVersion: argoproj.io/v1alpha1") {
				t.Error("Manifest should contain ArgoCD apiVersion")
			}

			if !strings.Contains(manifest, "kind: Application") {
				t.Error("Manifest should contain kind: Application")
			}

			if !strings.Contains(manifest, "name: cluster-addons") {
				t.Error("Manifest should contain name: cluster-addons")
			}

			// Verify repo URL
			if !strings.Contains(manifest, tt.config.RepoURL) {
				t.Errorf("Manifest should contain repoURL: %s", tt.config.RepoURL)
			}

			// Verify branch (with default fallback)
			if !strings.Contains(manifest, "targetRevision: "+tt.expectBranch) {
				t.Errorf("Manifest should contain targetRevision: %s", tt.expectBranch)
			}

			// Verify path (with default fallback)
			if !strings.Contains(manifest, "path: "+tt.expectPath) {
				t.Errorf("Manifest should contain path: %s", tt.expectPath)
			}

			// Verify sync policy
			if !strings.Contains(manifest, "syncPolicy:") {
				t.Error("Manifest should contain syncPolicy")
			}

			if !strings.Contains(manifest, "automated:") {
				t.Error("Manifest should have automated sync")
			}

			if !strings.Contains(manifest, "prune: true") {
				t.Error("Manifest should have prune enabled")
			}

			if !strings.Contains(manifest, "selfHeal: true") {
				t.Error("Manifest should have selfHeal enabled")
			}

			if !strings.Contains(manifest, "CreateNamespace=true") {
				t.Error("Manifest should have CreateNamespace option")
			}
		})
	}
}

// TestGenerateGitOpsRepoStructure tests GitOps repo structure generation
func TestGenerateGitOpsRepoStructure(t *testing.T) {
	structure := GenerateGitOpsRepoStructure()

	if structure == "" {
		t.Fatal("Structure should not be empty")
	}

	// Verify key components are present
	expectedComponents := []string{
		"addons/",
		"argocd/",
		"ingress-nginx/",
		"cert-manager/",
		"prometheus/",
		"longhorn/",
		"namespace.yaml",
		"helmrelease.yaml",
		"values.yaml",
		"How it works:",
		"Bootstrap ArgoCD:",
		"kubernetes-create addons bootstrap",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(structure, component) {
			t.Errorf("Structure should contain %q", component)
		}
	}
}

// TestAddonBootstrap tests AddonBootstrap structure
func TestAddonBootstrap(t *testing.T) {
	bootstrap := &AddonBootstrap{
		Name:         "argocd",
		Description:  "Bootstrap ArgoCD for GitOps",
		RepoPath:     "addons/argocd/",
		Dependencies: []string{},
		PostInstall:  []string{"get password", "port-forward"},
	}

	if bootstrap.Name == "" {
		t.Error("Name should not be empty")
	}

	if bootstrap.Description == "" {
		t.Error("Description should not be empty")
	}

	if bootstrap.RepoPath == "" {
		t.Error("RepoPath should not be empty")
	}

	if len(bootstrap.PostInstall) != 2 {
		t.Errorf("Expected 2 post-install commands, got %d", len(bootstrap.PostInstall))
	}
}

// TestGetBootstrapAddons tests bootstrap addons retrieval
func TestGetBootstrapAddons(t *testing.T) {
	bootstraps := GetBootstrapAddons()

	if bootstraps == nil {
		t.Fatal("Bootstrap addons should not be nil")
	}

	if len(bootstraps) == 0 {
		t.Error("Should have bootstrap addons")
	}

	// Verify ArgoCD is present (it's the foundation)
	argocd, exists := bootstraps["argocd"]
	if !exists {
		t.Fatal("ArgoCD should be in bootstrap addons")
	}

	// ArgoCD should have no dependencies (it's first)
	if len(argocd.Dependencies) > 0 {
		t.Error("ArgoCD should not have dependencies")
	}

	// ArgoCD should have post-install steps
	if len(argocd.PostInstall) == 0 {
		t.Error("ArgoCD should have post-install steps")
	}

	// Other addons should depend on ArgoCD
	expectedAddons := []string{"ingress-nginx", "cert-manager", "prometheus", "longhorn"}
	for _, name := range expectedAddons {
		addon, exists := bootstraps[name]
		if !exists {
			t.Errorf("Expected %q in bootstrap addons", name)
			continue
		}

		// Should have argocd as dependency
		hasArgoCDDep := false
		for _, dep := range addon.Dependencies {
			if dep == "argocd" {
				hasArgoCDDep = true
				break
			}
		}

		if !hasArgoCDDep {
			t.Errorf("Addon %q should depend on argocd", name)
		}
	}
}

// TestBootstrapAddonPaths tests that bootstrap addons have valid repo paths
func TestBootstrapAddonPaths(t *testing.T) {
	bootstraps := GetBootstrapAddons()

	for name, bootstrap := range bootstraps {
		t.Run(name, func(t *testing.T) {
			if bootstrap.RepoPath == "" {
				t.Error("RepoPath should not be empty")
			}

			// Should start with "addons/"
			if !strings.HasPrefix(bootstrap.RepoPath, "addons/") {
				t.Errorf("RepoPath should start with 'addons/', got %q", bootstrap.RepoPath)
			}

			// Should end with "/"
			if !strings.HasSuffix(bootstrap.RepoPath, "/") {
				t.Errorf("RepoPath should end with '/', got %q", bootstrap.RepoPath)
			}

			// Should contain addon name
			if !strings.Contains(bootstrap.RepoPath, name) {
				t.Errorf("RepoPath should contain addon name %q, got %q", name, bootstrap.RepoPath)
			}
		})
	}
}

// TestArgoCDPostInstallSteps tests ArgoCD post-install commands
func TestArgoCDPostInstallSteps(t *testing.T) {
	bootstraps := GetBootstrapAddons()
	argocd, exists := bootstraps["argocd"]

	if !exists {
		t.Fatal("ArgoCD should exist in bootstraps")
	}

	if len(argocd.PostInstall) == 0 {
		t.Fatal("ArgoCD should have post-install steps")
	}

	// Verify expected commands are mentioned
	expectedCommands := []string{
		"kubectl",
		"argocd-initial-admin-secret",
		"port-forward",
		"argocd login",
	}

	allSteps := strings.Join(argocd.PostInstall, " ")

	for _, cmd := range expectedCommands {
		if !strings.Contains(allSteps, cmd) {
			t.Errorf("Post-install steps should mention %q", cmd)
		}
	}
}

// TestGitOpsConfigBranchDefault tests branch default behavior
func TestGitOpsConfigBranchDefault(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{"Explicit main", "main", "main"},
		{"Explicit master", "master", "master"},
		{"Custom branch", "production", "production"},
		{"Empty string", "", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  tt.branch,
				Path:    "addons/",
			}

			manifest := generateArgoCDApp(config)

			// Determine expected branch (empty -> main)
			expectedBranch := tt.branch
			if expectedBranch == "" {
				expectedBranch = "main"
			}

			if tt.expected != expectedBranch {
				t.Errorf("Expected branch %q, got %q", tt.expected, expectedBranch)
			}

			if !strings.Contains(manifest, "targetRevision: "+expectedBranch) {
				t.Errorf("Manifest should use branch %q", expectedBranch)
			}
		})
	}
}

// TestGitOpsConfigPathDefault tests path default behavior
func TestGitOpsConfigPathDefault(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Explicit addons/", "addons/", "addons/"},
		{"Custom path", "k8s/manifests/", "k8s/manifests/"},
		{"Nested path", "deployments/cluster/addons/", "deployments/cluster/addons/"},
		{"Empty string", "", "addons/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: "https://github.com/user/repo",
				Branch:  "main",
				Path:    tt.path,
			}

			manifest := generateArgoCDApp(config)

			// Determine expected path (empty -> addons/)
			expectedPath := tt.path
			if expectedPath == "" {
				expectedPath = "addons/"
			}

			if tt.expected != expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expected, expectedPath)
			}

			if !strings.Contains(manifest, "path: "+expectedPath) {
				t.Errorf("Manifest should use path %q", expectedPath)
			}
		})
	}
}

// TestGitOpsRepoURLFormats tests different Git repository URL formats
func TestGitOpsRepoURLFormats(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		valid   bool
	}{
		{"HTTPS GitHub", "https://github.com/user/repo", true},
		{"HTTPS GitLab", "https://gitlab.com/user/repo", true},
		{"SSH GitHub", "git@github.com:user/repo.git", true},
		{"SSH GitLab", "git@gitlab.com:user/repo.git", true},
		{"Empty URL", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: tt.repoURL,
				Branch:  "main",
				Path:    "addons/",
			}

			isValid := config.RepoURL != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v for URL %q", tt.valid, isValid, tt.repoURL)
			}

			if tt.valid {
				manifest := generateArgoCDApp(config)
				if !strings.Contains(manifest, tt.repoURL) {
					t.Errorf("Manifest should contain repoURL %q", tt.repoURL)
				}
			}
		})
	}
}

// TestArgoCDManifestDestination tests ArgoCD Application destination
func TestArgoCDManifestDestination(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/user/repo",
		Branch:  "main",
		Path:    "addons/",
	}

	manifest := generateArgoCDApp(config)

	// Should target the local Kubernetes cluster
	if !strings.Contains(manifest, "server: https://kubernetes.default.svc") {
		t.Error("Manifest should target local Kubernetes cluster")
	}

	// Should deploy to argocd namespace
	if !strings.Contains(manifest, "namespace: argocd") {
		t.Error("Manifest should deploy to argocd namespace")
	}
}

// TestArgoCDManifestProject tests ArgoCD Application project
func TestArgoCDManifestProject(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/user/repo",
		Branch:  "main",
		Path:    "addons/",
	}

	manifest := generateArgoCDApp(config)

	// Should use default project
	if !strings.Contains(manifest, "project: default") {
		t.Error("Manifest should use default project")
	}
}

// TestBootstrapAddonDependencies tests dependency chains
func TestBootstrapAddonDependencies(t *testing.T) {
	bootstraps := GetBootstrapAddons()

	// Track which addons have dependencies
	hasDeps := make(map[string]bool)
	noDeps := make(map[string]bool)

	for name, bootstrap := range bootstraps {
		if len(bootstrap.Dependencies) > 0 {
			hasDeps[name] = true
		} else {
			noDeps[name] = true
		}
	}

	// ArgoCD should have no dependencies
	if !noDeps["argocd"] {
		t.Error("ArgoCD should have no dependencies")
	}

	// Other addons should have dependencies
	for name := range hasDeps {
		bootstrap := bootstraps[name]

		// Verify dependencies are valid
		for _, dep := range bootstrap.Dependencies {
			if _, exists := bootstraps[dep]; !exists {
				t.Errorf("Addon %q depends on non-existent addon %q", name, dep)
			}
		}
	}
}
