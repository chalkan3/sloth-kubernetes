package addons

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateArgoCDApp_DefaultValues tests ArgoCD app generation with defaults
func TestGenerateArgoCDApp_DefaultValues(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops-repo",
	}

	manifest := generateArgoCDApp(config)

	assert.Contains(t, manifest, "apiVersion: argoproj.io/v1alpha1")
	assert.Contains(t, manifest, "kind: Application")
	assert.Contains(t, manifest, "name: cluster-addons")
	assert.Contains(t, manifest, "namespace: argocd")
	assert.Contains(t, manifest, "repoURL: https://github.com/example/gitops-repo")
	assert.Contains(t, manifest, "targetRevision: main") // default branch
	assert.Contains(t, manifest, "path: addons/")         // default path
	assert.Contains(t, manifest, "prune: true")
	assert.Contains(t, manifest, "selfHeal: true")
	assert.Contains(t, manifest, "CreateNamespace=true")
}

// TestGenerateArgoCDApp_CustomBranch tests ArgoCD app with custom branch
func TestGenerateArgoCDApp_CustomBranch(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops-repo",
		Branch:  "develop",
	}

	manifest := generateArgoCDApp(config)

	assert.Contains(t, manifest, "targetRevision: develop")
}

// TestGenerateArgoCDApp_CustomPath tests ArgoCD app with custom path
func TestGenerateArgoCDApp_CustomPath(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops-repo",
		Path:    "manifests/production/",
	}

	manifest := generateArgoCDApp(config)

	assert.Contains(t, manifest, "path: manifests/production/")
}

// TestGenerateArgoCDApp_CompleteConfig tests ArgoCD app with all fields
func TestGenerateArgoCDApp_CompleteConfig(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL:    "git@github.com:example/gitops-repo.git",
		Branch:     "production",
		Path:       "k8s/addons/",
		PrivateKey: "/path/to/key",
	}

	manifest := generateArgoCDApp(config)

	assert.Contains(t, manifest, "repoURL: git@github.com:example/gitops-repo.git")
	assert.Contains(t, manifest, "targetRevision: production")
	assert.Contains(t, manifest, "path: k8s/addons/")
}

// TestGenerateArgoCDApp_Structure tests ArgoCD app YAML structure
func TestGenerateArgoCDApp_Structure(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops-repo",
		Branch:  "main",
		Path:    "addons/",
	}

	manifest := generateArgoCDApp(config)

	// Check YAML structure
	assert.Contains(t, manifest, "apiVersion:")
	assert.Contains(t, manifest, "kind:")
	assert.Contains(t, manifest, "metadata:")
	assert.Contains(t, manifest, "spec:")
	assert.Contains(t, manifest, "project: default")
	assert.Contains(t, manifest, "source:")
	assert.Contains(t, manifest, "destination:")
	assert.Contains(t, manifest, "server: https://kubernetes.default.svc")
	assert.Contains(t, manifest, "syncPolicy:")
	assert.Contains(t, manifest, "automated:")
}

// TestGenerateGitOpsRepoStructure_Content tests GitOps repo structure generation
func TestGenerateGitOpsRepoStructure_Content(t *testing.T) {
	structure := GenerateGitOpsRepoStructure()

	// Check structure contains key directories
	assert.Contains(t, structure, "addons/")
	assert.Contains(t, structure, "argocd/")
	assert.Contains(t, structure, "ingress-nginx/")
	assert.Contains(t, structure, "cert-manager/")
	assert.Contains(t, structure, "prometheus/")
	assert.Contains(t, structure, "longhorn/")
	assert.Contains(t, structure, "apps/")

	// Check YAML file references
	assert.Contains(t, structure, "namespace.yaml")
	assert.Contains(t, structure, "application.yaml")
	assert.Contains(t, structure, "helmrelease.yaml")
	assert.Contains(t, structure, "values.yaml")
}

// TestGenerateGitOpsRepoStructure_Instructions tests instructions
func TestGenerateGitOpsRepoStructure_Instructions(t *testing.T) {
	structure := GenerateGitOpsRepoStructure()

	// Check instructions are present
	assert.Contains(t, structure, "How it works:")
	assert.Contains(t, structure, "Bootstrap ArgoCD:")
	assert.Contains(t, structure, "ArgoCD watches")
	assert.Contains(t, structure, "To add a new addon:")
	assert.Contains(t, structure, "To remove an addon:")
}

// TestGetBootstrapAddons_AllAddons tests all bootstrap addons
func TestGetBootstrapAddons_AllAddons(t *testing.T) {
	addons := GetBootstrapAddons()

	expectedAddons := []string{"argocd", "ingress-nginx", "cert-manager", "prometheus", "longhorn"}

	for _, name := range expectedAddons {
		addon, exists := addons[name]
		assert.True(t, exists, "Addon %s should exist", name)
		assert.NotNil(t, addon)
		assert.Equal(t, name, addon.Name)
		assert.NotEmpty(t, addon.Description)
		assert.NotEmpty(t, addon.RepoPath)
	}
}

// TestGetBootstrapAddons_ArgoCD tests ArgoCD addon specifically
func TestGetBootstrapAddons_ArgoCD(t *testing.T) {
	addons := GetBootstrapAddons()

	argocd, exists := addons["argocd"]
	assert.True(t, exists)
	assert.Equal(t, "argocd", argocd.Name)
	assert.Equal(t, "Bootstrap ArgoCD for GitOps", argocd.Description)
	assert.Equal(t, "addons/argocd/", argocd.RepoPath)
	assert.NotEmpty(t, argocd.PostInstall)
	assert.Len(t, argocd.PostInstall, 3)
}

// TestGetBootstrapAddons_Dependencies tests addon dependencies
func TestGetBootstrapAddons_Dependencies(t *testing.T) {
	addons := GetBootstrapAddons()

	tests := []struct {
		name         string
		hasDeps      bool
		expectedDeps []string
	}{
		{"argocd", false, []string{}},
		{"ingress-nginx", true, []string{"argocd"}},
		{"cert-manager", true, []string{"argocd"}},
		{"prometheus", true, []string{"argocd"}},
		{"longhorn", true, []string{"argocd"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon := addons[tt.name]
			if tt.hasDeps {
				assert.NotEmpty(t, addon.Dependencies)
				assert.Equal(t, tt.expectedDeps, addon.Dependencies)
			} else {
				assert.Empty(t, addon.Dependencies)
			}
		})
	}
}

// TestGetBootstrapAddons_PostInstall tests PostInstall commands
func TestGetBootstrapAddons_PostInstall(t *testing.T) {
	addons := GetBootstrapAddons()

	argocd := addons["argocd"]

	// ArgoCD should have post-install commands
	assert.NotEmpty(t, argocd.PostInstall)

	// Check commands contain expected strings
	allCommands := strings.Join(argocd.PostInstall, " ")
	assert.Contains(t, allCommands, "kubectl")
	assert.Contains(t, allCommands, "argocd-initial-admin-secret")
	assert.Contains(t, allCommands, "port-forward")
}

// TestGetBootstrapAddons_RepoPaths tests repo paths
func TestGetBootstrapAddons_RepoPaths(t *testing.T) {
	addons := GetBootstrapAddons()

	tests := []struct {
		name         string
		expectedPath string
	}{
		{"argocd", "addons/argocd/"},
		{"ingress-nginx", "addons/ingress-nginx/"},
		{"cert-manager", "addons/cert-manager/"},
		{"prometheus", "addons/prometheus/"},
		{"longhorn", "addons/longhorn/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addon := addons[tt.name]
			assert.Equal(t, tt.expectedPath, addon.RepoPath)
		})
	}
}

// TestGitOpsConfig_Structure tests GitOpsConfig struct fields
func TestGitOpsConfig_Structure(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL:    "https://github.com/example/gitops",
		Branch:     "main",
		Path:       "addons/",
		PrivateKey: "/path/to/key",
	}

	assert.Equal(t, "https://github.com/example/gitops", config.RepoURL)
	assert.Equal(t, "main", config.Branch)
	assert.Equal(t, "addons/", config.Path)
	assert.Equal(t, "/path/to/key", config.PrivateKey)
}

// TestAddonBootstrap_Structure tests AddonBootstrap struct fields
func TestAddonBootstrap_Structure(t *testing.T) {
	addon := &AddonBootstrap{
		Name:         "test-addon",
		Description:  "Test addon for testing",
		RepoPath:     "addons/test/",
		Dependencies: []string{"argocd"},
		PostInstall:  []string{"echo 'installed'"},
	}

	assert.Equal(t, "test-addon", addon.Name)
	assert.Equal(t, "Test addon for testing", addon.Description)
	assert.Equal(t, "addons/test/", addon.RepoPath)
	assert.Len(t, addon.Dependencies, 1)
	assert.Len(t, addon.PostInstall, 1)
}

// Test50GitOpsScenarios tests 50 GitOps scenarios
func Test50GitOpsScenarios(t *testing.T) {
	repoURLs := []string{
		"https://github.com/example/gitops",
		"git@github.com:example/gitops.git",
		"https://gitlab.com/example/gitops",
		"git@gitlab.com:example/gitops.git",
		"https://bitbucket.org/example/gitops",
	}

	branches := []string{"main", "master", "develop", "staging", "production"}
	paths := []string{"addons/", "manifests/", "k8s/addons/", "deploy/", "kubernetes/"}

	for i := 0; i < 50; i++ {
		repoURL := repoURLs[i%len(repoURLs)]
		branch := branches[i%len(branches)]
		path := paths[i%len(paths)]

		t.Run("Scenario_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: repoURL,
				Branch:  branch,
				Path:    path,
			}

			manifest := generateArgoCDApp(config)

			assert.Contains(t, manifest, "apiVersion: argoproj.io/v1alpha1")
			assert.Contains(t, manifest, "kind: Application")
			assert.Contains(t, manifest, repoURL)
			assert.Contains(t, manifest, branch)
			assert.Contains(t, manifest, path)
		})
	}
}

// TestArgoCDApp_AllRepoTypes tests different repository types
func TestArgoCDApp_AllRepoTypes(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
	}{
		{"GitHub HTTPS", "https://github.com/user/repo"},
		{"GitHub SSH", "git@github.com:user/repo.git"},
		{"GitLab HTTPS", "https://gitlab.com/user/repo"},
		{"GitLab SSH", "git@gitlab.com:user/repo.git"},
		{"Bitbucket HTTPS", "https://bitbucket.org/user/repo"},
		{"Bitbucket SSH", "git@bitbucket.org:user/repo.git"},
		{"Self-hosted Git", "https://git.example.com/user/repo"},
		{"Self-hosted SSH", "git@git.example.com:user/repo.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: tt.repoURL,
			}

			manifest := generateArgoCDApp(config)

			assert.Contains(t, manifest, tt.repoURL)
			assert.Contains(t, manifest, "apiVersion: argoproj.io/v1alpha1")
		})
	}
}

// TestArgoCDApp_BranchVariations tests different branch names
func TestArgoCDApp_BranchVariations(t *testing.T) {
	branches := []string{
		"main",
		"master",
		"develop",
		"feature/new-addon",
		"release/v1.0",
		"hotfix/critical-bug",
		"staging",
		"production",
	}

	for _, branch := range branches {
		t.Run("Branch_"+branch, func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: "https://github.com/example/gitops",
				Branch:  branch,
			}

			manifest := generateArgoCDApp(config)

			assert.Contains(t, manifest, "targetRevision: "+branch)
		})
	}
}

// TestArgoCDApp_PathVariations tests different path variations
func TestArgoCDApp_PathVariations(t *testing.T) {
	paths := []string{
		"addons/",
		"manifests/",
		"k8s/addons/",
		"deploy/production/",
		"kubernetes/apps/",
		"charts/",
		"base/",
		"overlays/production/",
	}

	for _, path := range paths {
		t.Run("Path_"+path, func(t *testing.T) {
			config := &GitOpsConfig{
				RepoURL: "https://github.com/example/gitops",
				Path:    path,
			}

			manifest := generateArgoCDApp(config)

			assert.Contains(t, manifest, "path: "+path)
		})
	}
}

// TestArgoCDApp_SyncPolicy tests sync policy configuration
func TestArgoCDApp_SyncPolicy(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
	}

	manifest := generateArgoCDApp(config)

	// Check sync policy settings
	assert.Contains(t, manifest, "syncPolicy:")
	assert.Contains(t, manifest, "automated:")
	assert.Contains(t, manifest, "prune: true")
	assert.Contains(t, manifest, "selfHeal: true")
	assert.Contains(t, manifest, "syncOptions:")
	assert.Contains(t, manifest, "CreateNamespace=true")
}

// TestArgoCDApp_Destination tests destination configuration
func TestArgoCDApp_Destination(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
	}

	manifest := generateArgoCDApp(config)

	// Check destination settings
	assert.Contains(t, manifest, "destination:")
	assert.Contains(t, manifest, "server: https://kubernetes.default.svc")
	assert.Contains(t, manifest, "namespace: argocd")
}

// TestArgoCDApp_Metadata tests metadata configuration
func TestArgoCDApp_Metadata(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
	}

	manifest := generateArgoCDApp(config)

	// Check metadata settings
	assert.Contains(t, manifest, "metadata:")
	assert.Contains(t, manifest, "name: cluster-addons")
	assert.Contains(t, manifest, "namespace: argocd")
}

// TestArgoCDApp_Project tests project configuration
func TestArgoCDApp_Project(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
	}

	manifest := generateArgoCDApp(config)

	// Check project setting
	assert.Contains(t, manifest, "project: default")
}

// TestGitOpsRepoStructure_AllAddons tests all addons in structure
func TestGitOpsRepoStructure_AllAddons(t *testing.T) {
	structure := GenerateGitOpsRepoStructure()

	addons := []string{
		"argocd",
		"ingress-nginx",
		"cert-manager",
		"prometheus",
		"longhorn",
	}

	for _, addon := range addons {
		assert.Contains(t, structure, addon+"/", "Structure should contain %s addon", addon)
	}
}

// TestGitOpsRepoStructure_FileTypes tests different file types
func TestGitOpsRepoStructure_FileTypes(t *testing.T) {
	structure := GenerateGitOpsRepoStructure()

	fileTypes := []string{
		"namespace.yaml",
		"application.yaml",
		"helmrelease.yaml",
		"values.yaml",
		"issuer.yaml",
		"storageclass.yaml",
	}

	for _, fileType := range fileTypes {
		assert.Contains(t, structure, fileType, "Structure should contain %s", fileType)
	}
}

// TestBootstrapAddons_Count tests addon count
func TestBootstrapAddons_Count(t *testing.T) {
	addons := GetBootstrapAddons()

	assert.Len(t, addons, 5, "Should have 5 bootstrap addons")
}

// TestBootstrapAddons_NoDuplicates tests no duplicate addon names
func TestBootstrapAddons_NoDuplicates(t *testing.T) {
	addons := GetBootstrapAddons()

	names := make(map[string]bool)
	for name := range addons {
		assert.False(t, names[name], "Duplicate addon name: %s", name)
		names[name] = true
	}
}

// TestBootstrapAddons_ValidPaths tests all paths are valid
func TestBootstrapAddons_ValidPaths(t *testing.T) {
	addons := GetBootstrapAddons()

	for name, addon := range addons {
		assert.NotEmpty(t, addon.RepoPath, "Addon %s should have a repo path", name)
		assert.True(t, strings.HasPrefix(addon.RepoPath, "addons/"), "Addon %s path should start with 'addons/'", name)
		assert.True(t, strings.HasSuffix(addon.RepoPath, "/"), "Addon %s path should end with '/'", name)
	}
}

// TestArgoCDApp_EmptyBranch tests default branch behavior
func TestArgoCDApp_EmptyBranch(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
		Branch:  "", // Empty should default to "main"
	}

	manifest := generateArgoCDApp(config)

	assert.Contains(t, manifest, "targetRevision: main")
}

// TestArgoCDApp_EmptyPath tests default path behavior
func TestArgoCDApp_EmptyPath(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
		Path:    "", // Empty should default to "addons/"
	}

	manifest := generateArgoCDApp(config)

	assert.Contains(t, manifest, "path: addons/")
}

// TestArgoCDApp_ValidYAML tests YAML format validity
func TestArgoCDApp_ValidYAML(t *testing.T) {
	config := &GitOpsConfig{
		RepoURL: "https://github.com/example/gitops",
		Branch:  "main",
		Path:    "addons/",
	}

	manifest := generateArgoCDApp(config)

	// Basic YAML validity checks
	lines := strings.Split(manifest, "\n")

	// Should have multiple lines
	assert.Greater(t, len(lines), 10)

	// Should not have tabs (YAML uses spaces)
	assert.NotContains(t, manifest, "\t")
}
