package addons

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitOpsConfig represents GitOps configuration
type GitOpsConfig struct {
	RepoURL    string // Git repository URL
	Branch     string // Branch to use (default: main)
	Path       string // Path within repo (default: addons/)
	PrivateKey string // SSH private key for private repos
}

// AddonBootstrap represents an addon bootstrap configuration
type AddonBootstrap struct {
	Name         string
	Description  string
	RepoPath     string // Path within GitOps repo (e.g., "addons/argocd/")
	Dependencies []string
	PostInstall  []string // Commands to run after install
}

// BootstrapArgoCD bootstraps ArgoCD and configures it to watch the GitOps repo
func BootstrapArgoCD(kubeconfig string, gitopsConfig *GitOpsConfig) error {
	// 1. Install ArgoCD using kubectl
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"create", "namespace", "argocd", "--dry-run=client", "-o", "yaml")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create argocd namespace: %w", err)
	}

	// Apply ArgoCD manifests
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"apply", "-n", "argocd",
		"-f", "https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install ArgoCD: %w", err)
	}

	// 2. Wait for ArgoCD to be ready
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"wait", "--for=condition=ready", "pod",
		"-l", "app.kubernetes.io/name=argocd-server",
		"-n", "argocd", "--timeout=300s")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ArgoCD pods not ready: %w", err)
	}

	// 3. Create ArgoCD Application pointing to the GitOps repo
	appManifest := generateArgoCDApp(gitopsConfig)
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"apply", "-n", "argocd", "-f", "-")
	cmd.Stdin = strings.NewReader(appManifest)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %w", err)
	}

	return nil
}

// generateArgoCDApp generates ArgoCD Application manifest
func generateArgoCDApp(config *GitOpsConfig) string {
	branch := config.Branch
	if branch == "" {
		branch = "main"
	}

	path := config.Path
	if path == "" {
		path = "addons/"
	}

	return fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: cluster-addons
  namespace: argocd
spec:
  project: default
  source:
    repoURL: %s
    targetRevision: %s
    path: %s
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
`, config.RepoURL, branch, path)
}

// GenerateGitOpsRepoStructure generates example GitOps repo structure
func GenerateGitOpsRepoStructure() string {
	return `# GitOps Repository Structure

your-gitops-repo/
├── addons/
│   ├── argocd/
│   │   ├── namespace.yaml
│   │   └── application.yaml
│   │
│   ├── ingress-nginx/
│   │   ├── namespace.yaml
│   │   ├── helmrelease.yaml
│   │   └── values.yaml
│   │
│   ├── cert-manager/
│   │   ├── namespace.yaml
│   │   ├── helmrelease.yaml
│   │   └── issuer.yaml
│   │
│   ├── prometheus/
│   │   ├── namespace.yaml
│   │   ├── helmrelease.yaml
│   │   └── values.yaml
│   │
│   └── longhorn/
│       ├── namespace.yaml
│       ├── helmrelease.yaml
│       └── storageclass.yaml
│
├── apps/
│   └── [your applications]
│
└── README.md

## How it works:

1. Bootstrap ArgoCD:
   kubernetes-create addons bootstrap --repo https://github.com/you/gitops-repo

2. ArgoCD watches the 'addons/' directory in your repo

3. Any manifests you add to addons/* are automatically applied

4. To add a new addon:
   - Add manifests to addons/<addon-name>/
   - Commit and push
   - ArgoCD automatically syncs

5. To remove an addon:
   - Delete the directory from repo
   - ArgoCD automatically prunes
`
}

// GetBootstrapAddons returns addons that can be bootstrapped
func GetBootstrapAddons() map[string]*AddonBootstrap {
	return map[string]*AddonBootstrap{
		"argocd": {
			Name:        "argocd",
			Description: "Bootstrap ArgoCD for GitOps",
			RepoPath:    "addons/argocd/",
			PostInstall: []string{
				"Get ArgoCD password: kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d",
				"Port-forward: kubectl port-forward svc/argocd-server -n argocd 8080:443",
				"Login: argocd login localhost:8080",
			},
		},
		"ingress-nginx": {
			Name:         "ingress-nginx",
			Description:  "NGINX Ingress Controller via GitOps",
			RepoPath:     "addons/ingress-nginx/",
			Dependencies: []string{"argocd"},
		},
		"cert-manager": {
			Name:         "cert-manager",
			Description:  "Cert Manager via GitOps",
			RepoPath:     "addons/cert-manager/",
			Dependencies: []string{"argocd"},
		},
		"prometheus": {
			Name:         "prometheus",
			Description:  "Prometheus Stack via GitOps",
			RepoPath:     "addons/prometheus/",
			Dependencies: []string{"argocd"},
		},
		"longhorn": {
			Name:         "longhorn",
			Description:  "Longhorn Storage via GitOps",
			RepoPath:     "addons/longhorn/",
			Dependencies: []string{"argocd"},
		},
	}
}

// CloneGitOpsRepo clones the GitOps repository locally
func CloneGitOpsRepo(config *GitOpsConfig, destDir string) error {
	var cmd *exec.Cmd

	if config.PrivateKey != "" {
		// Use SSH with private key
		cmd = exec.Command("git", "clone", config.RepoURL, destDir)
		cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", config.PrivateKey))
	} else {
		// Public repo or HTTPS with credentials
		cmd = exec.Command("git", "clone", config.RepoURL, destDir)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Checkout specific branch if specified
	if config.Branch != "" && config.Branch != "main" && config.Branch != "master" {
		cmd = exec.Command("git", "-C", destDir, "checkout", config.Branch)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout branch %s: %w", config.Branch, err)
		}
	}

	return nil
}

// ApplyAddonsFromRepo applies all addons from a GitOps repo path
func ApplyAddonsFromRepo(kubeconfig string, repoPath string, addonPath string) error {
	fullPath := fmt.Sprintf("%s/%s", repoPath, addonPath)

	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"apply", "-R", "-f", fullPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply addons: %w\nOutput: %s", err, string(output))
	}

	return nil
}
