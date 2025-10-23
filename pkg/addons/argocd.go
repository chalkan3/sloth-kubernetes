package addons

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
)

// InstallArgoCD installs ArgoCD and applies GitOps applications
func InstallArgoCD(cfg *config.ClusterConfig, masterNodeIP string, sshPrivateKey string) error {
	if cfg.Addons.ArgoCD == nil || !cfg.Addons.ArgoCD.Enabled {
		return nil // ArgoCD not enabled, skip
	}

	argocdConfig := cfg.Addons.ArgoCD

	// Set defaults
	if argocdConfig.Namespace == "" {
		argocdConfig.Namespace = "argocd"
	}
	if argocdConfig.GitOpsRepoBranch == "" {
		argocdConfig.GitOpsRepoBranch = "main"
	}
	if argocdConfig.AppsPath == "" {
		argocdConfig.AppsPath = "argocd/apps"
	}
	if argocdConfig.Version == "" {
		argocdConfig.Version = "stable" // or "v2.9.3" for specific version
	}

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("🚀 Installing ArgoCD GitOps")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println()

	// Step 1: Install ArgoCD
	fmt.Println("📦 Step 1: Installing ArgoCD...")
	if err := installArgoCDManifests(masterNodeIP, sshPrivateKey, argocdConfig); err != nil {
		return fmt.Errorf("failed to install ArgoCD: %w", err)
	}

	// Step 2: Wait for ArgoCD to be ready
	fmt.Println("⏳ Step 2: Waiting for ArgoCD to be ready...")
	if err := waitForArgoCDReady(masterNodeIP, sshPrivateKey, argocdConfig.Namespace); err != nil {
		return fmt.Errorf("failed to wait for ArgoCD: %w", err)
	}

	// Step 3: Clone GitOps repo and apply applications
	fmt.Println("📂 Step 3: Applying GitOps applications from repository...")
	if err := applyGitOpsApplications(masterNodeIP, sshPrivateKey, argocdConfig); err != nil {
		return fmt.Errorf("failed to apply GitOps applications: %w", err)
	}

	// Step 4: Get ArgoCD admin password
	fmt.Println()
	fmt.Println("✅ ArgoCD installation completed successfully!")
	fmt.Println()

	if argocdConfig.AdminPassword == "" {
		password, err := getArgoCDAdminPassword(masterNodeIP, sshPrivateKey, argocdConfig.Namespace)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not retrieve ArgoCD admin password: %v\n", err)
		} else {
			fmt.Println("═══════════════════════════════════════════════════════")
			fmt.Println("🔐 ArgoCD Access Information")
			fmt.Println("═══════════════════════════════════════════════════════")
			fmt.Printf("  Username: admin\n")
			fmt.Printf("  Password: %s\n", password)
			fmt.Println()
			fmt.Println("  To access ArgoCD UI, port-forward:")
			fmt.Printf("  kubectl port-forward svc/argocd-server -n %s 8080:443\n", argocdConfig.Namespace)
			fmt.Println("  Then access: https://localhost:8080")
			fmt.Println("═══════════════════════════════════════════════════════")
		}
	}

	fmt.Println()
	fmt.Printf("📋 GitOps Repository: %s\n", argocdConfig.GitOpsRepoURL)
	fmt.Printf("📂 Applications Path: %s\n", argocdConfig.AppsPath)
	fmt.Println()

	return nil
}

// installArgoCDManifests installs ArgoCD using official manifests
func installArgoCDManifests(masterNodeIP string, sshPrivateKey string, argocdConfig *config.ArgoCDConfig) error {
	installScript := fmt.Sprintf(`
set -e

# Install kubectl if not present
if ! command -v kubectl &> /dev/null; then
    echo "Installing kubectl..."
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    rm kubectl
    echo "kubectl installed successfully"
else
    echo "kubectl already installed"
fi

# Create ArgoCD namespace
kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -

# Install ArgoCD
kubectl apply -n %s -f https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml

echo "ArgoCD installed successfully"
`, argocdConfig.Namespace, argocdConfig.Namespace, argocdConfig.Version)

	return runSSHCommand(masterNodeIP, sshPrivateKey, installScript)
}

// waitForArgoCDReady waits for ArgoCD pods to be ready
func waitForArgoCDReady(masterNodeIP string, sshPrivateKey string, namespace string) error {
	waitScript := fmt.Sprintf(`
set -e

echo "Waiting for ArgoCD pods to be ready..."
kubectl wait --for=condition=Ready pods --all -n %s --timeout=300s

echo "ArgoCD is ready"
`, namespace)

	return runSSHCommand(masterNodeIP, sshPrivateKey, waitScript)
}

// applyGitOpsApplications clones the GitOps repo and applies application manifests
func applyGitOpsApplications(masterNodeIP string, sshPrivateKey string, argocdConfig *config.ArgoCDConfig) error {
	applyScript := fmt.Sprintf(`
set -e

# Create temporary directory for GitOps repo
TEMP_DIR=$(mktemp -d)

# Ensure cleanup happens even on error
trap "cd / && rm -rf $TEMP_DIR" EXIT

cd $TEMP_DIR

# Clone the GitOps repository
echo "Cloning GitOps repository: %s"
git clone -b %s %s gitops-repo

# Apply all YAML files from the apps path
if [ -d "gitops-repo/%s" ]; then
	echo "Applying applications from %s..."
	kubectl apply -f gitops-repo/%s/ -n %s
	echo "Applications applied successfully"
else
	echo "Warning: Apps path 'gitops-repo/%s' not found"
fi

echo "GitOps applications deployed"
echo "Cleaning up temporary directory..."
`, argocdConfig.GitOpsRepoURL, argocdConfig.GitOpsRepoBranch, argocdConfig.GitOpsRepoURL,
		argocdConfig.AppsPath, argocdConfig.AppsPath, argocdConfig.AppsPath, argocdConfig.Namespace,
		argocdConfig.AppsPath)

	return runSSHCommand(masterNodeIP, sshPrivateKey, applyScript)
}

// getArgoCDAdminPassword retrieves the ArgoCD admin password
func getArgoCDAdminPassword(masterNodeIP string, sshPrivateKey string, namespace string) (string, error) {
	getPasswordScript := fmt.Sprintf(`
set -e
kubectl -n %s get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
`, namespace)

	output, err := runSSHCommandWithOutput(masterNodeIP, sshPrivateKey, getPasswordScript)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// runSSHCommand executes a command on the remote node via SSH
func runSSHCommand(host string, privateKey string, command string) error {
	_, err := runSSHCommandWithOutput(host, privateKey, command)
	return err
}

// runSSHCommandWithOutput executes a command on the remote node via SSH and returns output
func runSSHCommandWithOutput(host string, privateKey string, command string) (string, error) {
	// Save private key to temporary file
	tmpKeyFile := fmt.Sprintf("/tmp/ssh-key-%d", time.Now().UnixNano())
	if err := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > %s && chmod 600 %s", privateKey, tmpKeyFile, tmpKeyFile)).Run(); err != nil {
		return "", fmt.Errorf("failed to save SSH key: %w", err)
	}
	defer exec.Command("rm", "-f", tmpKeyFile).Run()

	// Execute SSH command
	sshCmd := fmt.Sprintf(`ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s root@%s '%s'`, tmpKeyFile, host, strings.ReplaceAll(command, "'", "'\\''"))

	cmd := exec.Command("bash", "-c", sshCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}
