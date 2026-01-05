package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/addons"
	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var (
	argocdNamespace  string
	argocdVersion    string
	gitopsRepoURL    string
	gitopsRepoBranch string
	gitopsAppsPath   string
	appOfAppsEnabled bool
	appOfAppsName    string
)

var argocdCmd = &cobra.Command{
	Use:   "argocd",
	Short: "Manage ArgoCD GitOps integration",
	Long: `Manage ArgoCD GitOps integration for your Kubernetes cluster.

ArgoCD enables declarative GitOps continuous delivery with automatic
synchronization of your applications from a Git repository.

Supports App of Apps pattern for managing multiple applications.`,
}

var argocdInstallCmd = &cobra.Command{
	Use:   "install [stack-name]",
	Short: "Install ArgoCD on the cluster",
	Long: `Install ArgoCD on your Kubernetes cluster.

This command will:
  1. Create the ArgoCD namespace
  2. Install ArgoCD from official manifests
  3. Wait for all pods to be ready
  4. Optionally configure GitOps repository
  5. Set up App of Apps pattern if enabled`,
	Example: `  # Install ArgoCD with defaults
  sloth-kubernetes argocd install my-cluster

  # Install with GitOps repo
  sloth-kubernetes argocd install my-cluster \
    --repo https://github.com/myorg/gitops.git \
    --branch main \
    --apps-path argocd/apps

  # Install with App of Apps pattern
  sloth-kubernetes argocd install my-cluster \
    --repo https://github.com/myorg/gitops.git \
    --app-of-apps \
    --app-of-apps-name root-app`,
	RunE: runArgocdInstall,
}

var argocdStatusCmd = &cobra.Command{
	Use:   "status [stack-name]",
	Short: "Check ArgoCD status",
	Long:  `Check the status of ArgoCD installation and applications.`,
	Example: `  # Check ArgoCD status
  sloth-kubernetes argocd status my-cluster`,
	RunE: runArgocdStatus,
}

var argocdPasswordCmd = &cobra.Command{
	Use:   "password [stack-name]",
	Short: "Get ArgoCD admin password",
	Long:  `Retrieve the ArgoCD admin password from the cluster.`,
	Example: `  # Get ArgoCD admin password
  sloth-kubernetes argocd password my-cluster`,
	RunE: runArgocdPassword,
}

var argocdAppsCmd = &cobra.Command{
	Use:   "apps [stack-name]",
	Short: "List ArgoCD applications",
	Long:  `List all ArgoCD applications and their sync status.`,
	Example: `  # List all applications
  sloth-kubernetes argocd apps my-cluster`,
	RunE: runArgocdApps,
}

var argocdSyncCmd = &cobra.Command{
	Use:   "sync [stack-name] [app-name]",
	Short: "Sync an ArgoCD application",
	Long:  `Trigger synchronization of an ArgoCD application.`,
	Example: `  # Sync all applications
  sloth-kubernetes argocd sync my-cluster --all

  # Sync specific application
  sloth-kubernetes argocd sync my-cluster my-app`,
	RunE: runArgocdSync,
}

func init() {
	rootCmd.AddCommand(argocdCmd)
	argocdCmd.AddCommand(argocdInstallCmd)
	argocdCmd.AddCommand(argocdStatusCmd)
	argocdCmd.AddCommand(argocdPasswordCmd)
	argocdCmd.AddCommand(argocdAppsCmd)
	argocdCmd.AddCommand(argocdSyncCmd)

	// Install command flags
	argocdInstallCmd.Flags().StringVar(&argocdNamespace, "namespace", "argocd", "ArgoCD namespace")
	argocdInstallCmd.Flags().StringVar(&argocdVersion, "version", "stable", "ArgoCD version (stable, v2.9.3, etc)")
	argocdInstallCmd.Flags().StringVar(&gitopsRepoURL, "repo", "", "GitOps repository URL")
	argocdInstallCmd.Flags().StringVar(&gitopsRepoBranch, "branch", "main", "GitOps repository branch")
	argocdInstallCmd.Flags().StringVar(&gitopsAppsPath, "apps-path", "argocd/apps", "Path to applications in GitOps repo")
	argocdInstallCmd.Flags().BoolVar(&appOfAppsEnabled, "app-of-apps", false, "Enable App of Apps pattern")
	argocdInstallCmd.Flags().StringVar(&appOfAppsName, "app-of-apps-name", "root-app", "Name for App of Apps")

	// Status command flags
	argocdStatusCmd.Flags().StringVar(&argocdNamespace, "namespace", "argocd", "ArgoCD namespace")

	// Password command flags
	argocdPasswordCmd.Flags().StringVar(&argocdNamespace, "namespace", "argocd", "ArgoCD namespace")

	// Apps command flags
	argocdAppsCmd.Flags().StringVar(&argocdNamespace, "namespace", "argocd", "ArgoCD namespace")

	// Sync command flags
	argocdSyncCmd.Flags().StringVar(&argocdNamespace, "namespace", "argocd", "ArgoCD namespace")
	argocdSyncCmd.Flags().Bool("all", false, "Sync all applications")
}

func runArgocdInstall(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	printHeader("Installing ArgoCD GitOps")

	// Get stack name
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	// Get stack info including SSH credentials
	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get stack info: %w", err)
	}

	if stackInfo.MasterIP == "" || stackInfo.SSHKeyPath == "" {
		return fmt.Errorf("could not get master IP or SSH key from stack '%s'", targetStack)
	}

	// Read SSH key
	sshKey, err := readSSHKey(stackInfo.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key: %w", err)
	}

	// Create minimal config for addons
	cfg := &config.ClusterConfig{
		Addons: config.AddonsConfig{
			ArgoCD: &config.ArgoCDConfig{
				Enabled:          true,
				Namespace:        argocdNamespace,
				Version:          argocdVersion,
				GitOpsRepoURL:    gitopsRepoURL,
				GitOpsRepoBranch: gitopsRepoBranch,
				AppsPath:         gitopsAppsPath,
			},
		},
	}

	// Install ArgoCD
	if err := addons.InstallArgoCD(cfg, stackInfo.MasterIP, sshKey); err != nil {
		return fmt.Errorf("failed to install ArgoCD: %w", err)
	}

	// Install App of Apps if enabled
	if appOfAppsEnabled && gitopsRepoURL != "" {
		fmt.Println()
		color.Cyan("Setting up App of Apps pattern...")
		if err := addons.SetupAppOfApps(stackInfo.MasterIP, sshKey, &addons.AppOfAppsConfig{
			Name:       appOfAppsName,
			Namespace:  argocdNamespace,
			RepoURL:    gitopsRepoURL,
			Branch:     gitopsRepoBranch,
			Path:       gitopsAppsPath,
			SyncPolicy: "automated",
		}); err != nil {
			color.Yellow("Warning: Failed to setup App of Apps: %v", err)
		} else {
			color.Green("App of Apps '%s' created successfully!", appOfAppsName)
		}
	}

	fmt.Println()
	printSuccess("ArgoCD installation completed!")
	printArgocdAccessInfo(argocdNamespace)

	// Record the operation
	details := fmt.Sprintf("Version: %s, Namespace: %s", argocdVersion, argocdNamespace)
	if gitopsRepoURL != "" {
		details += fmt.Sprintf(", GitOps Repo: %s", gitopsRepoURL)
	}
	if appOfAppsEnabled {
		details += fmt.Sprintf(", App of Apps: %s", appOfAppsName)
	}
	operations.RecordArgoCDOperation(targetStack, "install", "", argocdNamespace, "success", "", "", details, time.Since(startTime), nil)

	return nil
}

func runArgocdStatus(cmd *cobra.Command, args []string) error {
	printHeader("ArgoCD Status")

	// Get stack name
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	// Get stack info
	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get stack info: %w", err)
	}

	if stackInfo.MasterIP == "" || stackInfo.SSHKeyPath == "" {
		return fmt.Errorf("could not get master IP or SSH key from stack '%s'", targetStack)
	}

	sshKey, err := readSSHKey(stackInfo.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key: %w", err)
	}

	fmt.Println()
	color.Cyan("Checking ArgoCD pods...")
	fmt.Println()

	// Check pod status
	status, err := addons.GetArgoCDStatus(stackInfo.MasterIP, sshKey, argocdNamespace)
	if err != nil {
		return fmt.Errorf("failed to get ArgoCD status: %w", err)
	}

	// Display status
	fmt.Printf("Namespace: %s\n", argocdNamespace)
	fmt.Println()

	if status.Healthy {
		color.Green("Status: Healthy")
	} else {
		color.Red("Status: Unhealthy")
	}
	fmt.Println()

	fmt.Println("Pods:")
	for _, pod := range status.Pods {
		statusIcon := "✅"
		if pod.Status != "Running" {
			statusIcon = "❌"
		}
		fmt.Printf("  %s %s: %s (%s)\n", statusIcon, pod.Name, pod.Status, pod.Ready)
	}

	fmt.Println()
	fmt.Printf("Applications: %d total, %d synced, %d out-of-sync\n",
		status.AppsTotal, status.AppsSynced, status.AppsOutOfSync)

	return nil
}

func runArgocdPassword(cmd *cobra.Command, args []string) error {
	// Get stack name
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get stack info: %w", err)
	}

	if stackInfo.MasterIP == "" || stackInfo.SSHKeyPath == "" {
		return fmt.Errorf("could not get master IP or SSH key from stack '%s'", targetStack)
	}

	sshKey, err := readSSHKey(stackInfo.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key: %w", err)
	}

	password, err := addons.GetArgoCDPassword(stackInfo.MasterIP, sshKey, argocdNamespace)
	if err != nil {
		return fmt.Errorf("failed to get ArgoCD password: %w", err)
	}

	fmt.Println()
	color.Cyan("ArgoCD Admin Credentials")
	fmt.Println("========================")
	fmt.Printf("Username: admin\n")
	fmt.Printf("Password: %s\n", password)
	fmt.Println()

	return nil
}

func runArgocdApps(cmd *cobra.Command, args []string) error {
	printHeader("ArgoCD Applications")

	// Get stack name
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get stack info: %w", err)
	}

	if stackInfo.MasterIP == "" || stackInfo.SSHKeyPath == "" {
		return fmt.Errorf("could not get master IP or SSH key from stack '%s'", targetStack)
	}

	sshKey, err := readSSHKey(stackInfo.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key: %w", err)
	}

	apps, err := addons.ListArgoCDApps(stackInfo.MasterIP, sshKey, argocdNamespace)
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	if len(apps) == 0 {
		fmt.Println("No applications found.")
		return nil
	}

	fmt.Println()
	fmt.Printf("%-30s %-15s %-15s %-20s\n", "NAME", "SYNC STATUS", "HEALTH", "REPO")
	fmt.Println(strings.Repeat("-", 80))

	for _, app := range apps {
		syncIcon := "✅"
		if app.SyncStatus != "Synced" {
			syncIcon = "⚠️"
		}
		healthIcon := "💚"
		if app.Health != "Healthy" {
			healthIcon = "❤️"
		}

		fmt.Printf("%-30s %s %-13s %s %-13s %-20s\n",
			truncate(app.Name, 30),
			syncIcon, app.SyncStatus,
			healthIcon, app.Health,
			truncate(app.RepoURL, 20))
	}

	fmt.Println()
	return nil
}

func runArgocdSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	syncAll, _ := cmd.Flags().GetBool("all")

	// Get stack name
	targetStack, err := RequireStackArg(args)
	if err != nil {
		return err
	}

	// Get app name if provided (second argument after stack)
	var appName string
	if len(args) > 1 {
		appName = args[1]
	}

	stackInfo, err := GetStackInfo(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get stack info: %w", err)
	}

	if stackInfo.MasterIP == "" || stackInfo.SSHKeyPath == "" {
		return fmt.Errorf("could not get master IP or SSH key from stack '%s'", targetStack)
	}

	sshKey, err := readSSHKey(stackInfo.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH key: %w", err)
	}

	if syncAll {
		color.Cyan("Syncing all applications...")
		if err := addons.SyncAllApps(stackInfo.MasterIP, sshKey, argocdNamespace); err != nil {
			operations.RecordArgoCDOperation(targetStack, "sync", "--all", argocdNamespace, "failed", "", "", "Sync all applications failed", time.Since(startTime), err)
			return fmt.Errorf("failed to sync applications: %w", err)
		}
		printSuccess("All applications synced!")
		operations.RecordArgoCDOperation(targetStack, "sync", "--all", argocdNamespace, "success", "Synced", "", "All applications synced", time.Since(startTime), nil)
	} else if appName != "" {
		color.Cyan("Syncing application: %s", appName)
		if err := addons.SyncApp(stackInfo.MasterIP, sshKey, argocdNamespace, appName); err != nil {
			operations.RecordArgoCDOperation(targetStack, "sync", appName, argocdNamespace, "failed", "", "", fmt.Sprintf("Sync application %s failed", appName), time.Since(startTime), err)
			return fmt.Errorf("failed to sync application %s: %w", appName, err)
		}
		printSuccess(fmt.Sprintf("Application '%s' synced!", appName))
		operations.RecordArgoCDOperation(targetStack, "sync", appName, argocdNamespace, "success", "Synced", "", fmt.Sprintf("Application %s synced", appName), time.Since(startTime), nil)
	} else {
		return fmt.Errorf("please specify an application name or use --all")
	}

	return nil
}

// Helper functions

func readSSHKey(path string) (string, error) {
	if path == "" {
		path = os.ExpandEnv("$HOME/.ssh/id_rsa")
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = home + path[1:]
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func printArgocdAccessInfo(namespace string) {
	fmt.Println()
	color.Cyan("Access Information:")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("To access ArgoCD UI:")
	fmt.Printf("  kubectl port-forward svc/argocd-server -n %s 8080:443\n", namespace)
	fmt.Println("  Then open: https://localhost:8080")
	fmt.Println()
	fmt.Println("To get admin password:")
	fmt.Println("  sloth-kubernetes argocd password <stack-name>")
	fmt.Println()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
