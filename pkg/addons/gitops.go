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

// ArgoCDLocalStatus represents ArgoCD status queried via kubeconfig
type ArgoCDLocalStatus struct {
	Running      bool
	Version      string
	Namespace    string
	Applications []ArgoCDAppDetail
}

// ArgoCDAppDetail represents detailed ArgoCD application info
type ArgoCDAppDetail struct {
	Name       string
	SyncStatus string
	Health     string
	Namespace  string
	RepoURL    string
}

// SyncArgoCDApplications triggers sync for ArgoCD applications via kubeconfig
func SyncArgoCDApplications(kubeconfig string, appName string) error {
	var cmd *exec.Cmd

	if appName == "" || appName == "--all" {
		// Sync all applications by refreshing each one
		cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
			"get", "applications", "-n", "argocd", "-o", "jsonpath={.items[*].metadata.name}")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to list applications: %w", err)
		}

		apps := strings.Fields(string(output))
		for _, app := range apps {
			// Trigger refresh by patching the application
			patchCmd := exec.Command("kubectl", "--kubeconfig", kubeconfig,
				"patch", "application", app, "-n", "argocd",
				"--type", "merge", "-p", `{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}`)
			if err := patchCmd.Run(); err != nil {
				return fmt.Errorf("failed to sync application %s: %w", app, err)
			}
		}
	} else {
		// Sync specific application
		cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
			"patch", "application", appName, "-n", "argocd",
			"--type", "merge", "-p", `{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}`)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to sync application %s: %w", appName, err)
		}
	}

	return nil
}

// GetArgoCDLocalStatus gets the current status of ArgoCD using kubeconfig
func GetArgoCDLocalStatus(kubeconfig string) (*ArgoCDLocalStatus, error) {
	status := &ArgoCDLocalStatus{
		Namespace:    "argocd",
		Applications: []ArgoCDAppDetail{},
	}

	// Check if ArgoCD server is running
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"get", "pods", "-n", "argocd", "-l", "app.kubernetes.io/name=argocd-server",
		"-o", "jsonpath={.items[0].status.phase}")
	output, err := cmd.Output()
	if err != nil {
		status.Running = false
	} else {
		status.Running = strings.TrimSpace(string(output)) == "Running"
	}

	// Get ArgoCD version
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"get", "pods", "-n", "argocd", "-l", "app.kubernetes.io/name=argocd-server",
		"-o", "jsonpath={.items[0].spec.containers[0].image}")
	output, err = cmd.Output()
	if err == nil {
		// Extract version from image tag
		image := string(output)
		if parts := strings.Split(image, ":"); len(parts) > 1 {
			status.Version = parts[len(parts)-1]
		} else {
			status.Version = "unknown"
		}
	}

	// Get applications status
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig,
		"get", "applications", "-n", "argocd",
		"-o", "jsonpath={range .items[*]}{.metadata.name}|{.status.sync.status}|{.status.health.status}|{.spec.destination.namespace}|{.spec.source.repoURL}{\"\\n\"}{end}")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				status.Applications = append(status.Applications, ArgoCDAppDetail{
					Name:       parts[0],
					SyncStatus: parts[1],
					Health:     parts[2],
					Namespace:  parts[3],
					RepoURL:    parts[4],
				})
			}
		}
	}

	return status, nil
}

// InstalledAddon represents an addon installed in the cluster
type InstalledAddon struct {
	Name      string
	Category  string
	Status    string
	Version   string
	Namespace string
}

// GetInstalledAddons gets the list of installed addons from the cluster
func GetInstalledAddons(kubeconfig string) ([]InstalledAddon, error) {
	addons := []InstalledAddon{}

	// Define known addons and their namespaces
	knownAddons := []struct {
		name      string
		namespace string
		category  string
		label     string
	}{
		{"argocd", "argocd", "CD", "app.kubernetes.io/name=argocd-server"},
		{"ingress-nginx", "ingress-nginx", "Ingress", "app.kubernetes.io/name=ingress-nginx"},
		{"cert-manager", "cert-manager", "Security", "app.kubernetes.io/name=cert-manager"},
		{"prometheus", "monitoring", "Monitoring", "app=prometheus"},
		{"longhorn", "longhorn-system", "Storage", "app=longhorn-manager"},
		{"metallb", "metallb-system", "LoadBalancer", "app=metallb"},
		{"traefik", "traefik", "Ingress", "app.kubernetes.io/name=traefik"},
	}

	for _, addon := range knownAddons {
		// Check if namespace exists and has running pods
		cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig,
			"get", "pods", "-n", addon.namespace, "-l", addon.label,
			"-o", "jsonpath={.items[0].status.phase}|{.items[0].spec.containers[0].image}")
		output, err := cmd.Output()
		if err != nil {
			continue // Addon not installed
		}

		parts := strings.Split(string(output), "|")
		if len(parts) >= 1 && parts[0] != "" {
			status := "❌ Unknown"
			if strings.TrimSpace(parts[0]) == "Running" {
				status = "✅ Running"
			} else if strings.TrimSpace(parts[0]) == "Pending" {
				status = "⏳ Pending"
			} else if strings.TrimSpace(parts[0]) == "Failed" {
				status = "❌ Failed"
			}

			version := "unknown"
			if len(parts) > 1 {
				image := parts[1]
				if imgParts := strings.Split(image, ":"); len(imgParts) > 1 {
					version = imgParts[len(imgParts)-1]
				}
			}

			addons = append(addons, InstalledAddon{
				Name:      addon.name,
				Category:  addon.category,
				Status:    status,
				Version:   version,
				Namespace: addon.namespace,
			})
		}
	}

	return addons, nil
}
