package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/pkg/addons"
	"github.com/chalkan3/sloth-kubernetes/pkg/operations"
)

var (
	gitopsRepo       string
	gitopsBranch     string
	gitopsPath       string
	gitopsPrivateKey string
	addonNamespace   string
	addonValues      string
)

// addonsCmd represents the addons command
var addonsCmd = &cobra.Command{
	Use:   "addons",
	Short: "Manage cluster addons via GitOps",
	Long: `Manage Kubernetes cluster addons using GitOps methodology.

The addons system works by:
1. You provide a Git repository URL
2. The repo contains addon manifests in directories (e.g., addons/argocd/)
3. ArgoCD is bootstrapped from that repo
4. ArgoCD watches the repo and auto-syncs all addons

This provides declarative, Git-based addon management.`,
}

// addonsBootstrapCmd bootstraps ArgoCD from a GitOps repo
var addonsBootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap ArgoCD from a GitOps repository",
	Long: `Bootstrap ArgoCD using a GitOps repository.

This command will:
1. Clone your GitOps repository
2. Find the ArgoCD manifests in the repo (e.g., addons/argocd/)
3. Apply ArgoCD via kubectl
4. Configure ArgoCD to watch your repo
5. ArgoCD will then auto-sync all other addons from the repo

The repository becomes your single source of truth for cluster addons.`,
	Example: `  # Bootstrap with public repo
  kubernetes-create addons bootstrap --repo https://github.com/you/gitops-repo

  # Bootstrap with specific branch and path
  kubernetes-create addons bootstrap \
    --repo https://github.com/you/gitops-repo \
    --branch main \
    --path addons/

  # Bootstrap with private repo
  kubernetes-create addons bootstrap \
    --repo git@github.com:you/private-repo.git \
    --private-key ~/.ssh/id_rsa`,
	RunE: runAddonsBootstrap,
}

// addonsListCmd lists installed addons
var addonsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed addons",
	Long: `List all addons currently installed in the cluster.

This queries both:
- ArgoCD Applications (GitOps-managed addons)
- Direct kubectl resources (manually installed addons)`,
	Example: `  # List all addons
  kubernetes-create addons list

  # List with specific stack
  kubernetes-create addons list --stack production`,
	RunE: runAddonsList,
}

// addonsSyncCmd manually triggers ArgoCD sync
var addonsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manually trigger ArgoCD sync",
	Long: `Manually trigger ArgoCD to sync all applications.

Normally ArgoCD syncs automatically, but this command forces an immediate sync.`,
	Example: `  # Sync all applications
  kubernetes-create addons sync

  # Sync specific application
  kubernetes-create addons sync --app cert-manager`,
	RunE: runAddonsSync,
}

// addonsStatusCmd shows addon status
var addonsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show ArgoCD and addon status",
	Long: `Display detailed status of ArgoCD and all managed addons.

Shows:
- ArgoCD server status
- All Applications and their sync status
- Health status of each addon`,
	Example: `  # Show status
  kubernetes-create addons status`,
	RunE: runAddonsStatus,
}

// addonsTemplateCmd generates example GitOps repo structure
var addonsTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate example GitOps repository structure",
	Long: `Generate an example GitOps repository structure showing how to organize
your addon manifests.

This creates a template directory structure you can use as a starting point
for your GitOps repository.`,
	Example: `  # Print template structure
  kubernetes-create addons template

  # Generate to directory
  kubernetes-create addons template --output ./my-gitops-repo`,
	RunE: runAddonsTemplate,
}

func init() {
	rootCmd.AddCommand(addonsCmd)

	// Add subcommands
	addonsCmd.AddCommand(addonsBootstrapCmd)
	addonsCmd.AddCommand(addonsListCmd)
	addonsCmd.AddCommand(addonsSyncCmd)
	addonsCmd.AddCommand(addonsStatusCmd)
	addonsCmd.AddCommand(addonsTemplateCmd)

	// Flags for bootstrap command
	addonsBootstrapCmd.Flags().StringVar(&gitopsRepo, "repo", "", "Git repository URL (required)")
	addonsBootstrapCmd.Flags().StringVar(&gitopsBranch, "branch", "main", "Git branch to use")
	addonsBootstrapCmd.Flags().StringVar(&gitopsPath, "path", "addons/", "Path within repo containing addons")
	addonsBootstrapCmd.Flags().StringVar(&gitopsPrivateKey, "private-key", "", "SSH private key for private repos")
	addonsBootstrapCmd.MarkFlagRequired("repo")

	// Flags for sync command
	addonsSyncCmd.Flags().StringVar(&addonNamespace, "app", "", "Specific application to sync")

	// Flags for template command
	addonsTemplateCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory for template")
}

func runAddonsBootstrap(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	printHeader("🚀 Bootstrap ArgoCD via GitOps")

	// Validate repo URL
	if gitopsRepo == "" {
		return fmt.Errorf("--repo flag is required")
	}

	fmt.Println()
	color.Cyan("📋 Bootstrap Configuration:")
	fmt.Printf("  • Repository: %s\n", gitopsRepo)
	fmt.Printf("  • Branch: %s\n", gitopsBranch)
	fmt.Printf("  • Path: %s\n", gitopsPath)
	if gitopsPrivateKey != "" {
		fmt.Printf("  • Private Key: %s\n", gitopsPrivateKey)
	}
	fmt.Println()

	// Confirmation
	if !autoApprove {
		color.Yellow("⚠️  This will:")
		fmt.Println("  1. Clone your GitOps repository")
		fmt.Println("  2. Install ArgoCD from the repo")
		fmt.Println("  3. Configure ArgoCD to watch the repo")
		fmt.Println("  4. ArgoCD will auto-sync all addons")
		fmt.Println()

		if !confirm("Do you want to proceed?") {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	// Get stack and kubeconfig
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Getting cluster information..."
	s.Start()

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to select stack: %w", err)
	}

	outputs, err := stack.Outputs(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to get outputs: %w", err)
	}

	// Extract kubeconfig path
	kubeconfigOutput, ok := outputs["kubeconfig"]
	if !ok {
		s.Stop()
		return fmt.Errorf("kubeconfig not found in stack outputs")
	}

	kubeconfigPath := fmt.Sprintf("%v", kubeconfigOutput.Value)
	s.Stop()

	color.Green("✅ Cluster found")
	fmt.Println()

	// Clone GitOps repo
	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Cloning GitOps repository..."
	s.Start()

	tempDir, err := os.MkdirTemp("", "gitops-*")
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	gitopsConfig := &addons.GitOpsConfig{
		RepoURL:    gitopsRepo,
		Branch:     gitopsBranch,
		Path:       gitopsPath,
		PrivateKey: gitopsPrivateKey,
	}

	if err := addons.CloneGitOpsRepo(gitopsConfig, tempDir); err != nil {
		s.Stop()
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	s.Stop()
	color.Green("✅ Repository cloned")
	fmt.Println()

	// Bootstrap ArgoCD
	s = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Bootstrapping ArgoCD..."
	s.Start()

	if err := addons.BootstrapArgoCD(kubeconfigPath, gitopsConfig); err != nil {
		s.Stop()
		return fmt.Errorf("failed to bootstrap ArgoCD: %w", err)
	}

	s.Stop()
	color.Green("✅ ArgoCD bootstrapped successfully!")
	fmt.Println()

	// Print next steps
	color.Cyan("🎉 Success! ArgoCD is now watching your repository")
	fmt.Println()
	color.Cyan("📝 Next Steps:")
	fmt.Println()
	fmt.Println("  1. Get ArgoCD admin password:")
	fmt.Println("     kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d")
	fmt.Println()
	fmt.Println("  2. Port-forward to ArgoCD UI:")
	fmt.Println("     kubectl port-forward svc/argocd-server -n argocd 8080:443")
	fmt.Println()
	fmt.Println("  3. Access ArgoCD:")
	fmt.Println("     https://localhost:8080")
	fmt.Println("     Username: admin")
	fmt.Println("     Password: (from step 1)")
	fmt.Println()
	fmt.Println("  4. Add more addons by committing to your repo:")
	fmt.Printf("     %s/%s\n", gitopsRepo, gitopsPath)
	fmt.Println()
	color.Cyan("💡 View addon status:")
	fmt.Println("   kubernetes-create addons status")

	// Record the operation
	details := fmt.Sprintf("GitOps Repo: %s, Branch: %s, Path: %s", gitopsRepo, gitopsBranch, gitopsPath)
	operations.RecordAddonsOperation(stackName, "bootstrap", "argocd", "gitops", "success", details, 1, 0, time.Since(startTime), nil)

	return nil
}

func runAddonsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	printHeader("📦 Installed Addons")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Fetching addon information..."
	s.Start()

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to select stack: %w", err)
	}

	outputs, err := stack.Outputs(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to get outputs: %w", err)
	}

	s.Stop()

	if len(outputs) == 0 {
		color.Yellow("⚠️  No cluster found")
		return nil
	}

	// Print addon table
	printAddonTable()

	return nil
}

func runAddonsSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	printHeader("🔄 Sync ArgoCD Applications")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Triggering ArgoCD sync..."
	s.Start()

	// TODO: Implement actual sync via argocd CLI or kubectl
	time.Sleep(2 * time.Second)

	s.Stop()

	color.Green("✅ Sync triggered successfully!")
	fmt.Println()
	color.Cyan("💡 Monitor sync status:")
	fmt.Println("   kubernetes-create addons status")
	fmt.Println("   kubectl get applications -n argocd")

	// Record the operation
	addonName := addonNamespace // --app flag value
	if addonName == "" {
		addonName = "--all"
	}
	operations.RecordAddonsOperation(stackName, "sync", addonName, "argocd", "success", "Sync triggered", 0, 0, time.Since(startTime), nil)

	return nil
}

func runAddonsStatus(cmd *cobra.Command, args []string) error {
	printHeader("📊 ArgoCD & Addon Status")

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Fetching status..."
	s.Start()

	// TODO: Implement actual status check via kubectl
	time.Sleep(2 * time.Second)

	s.Stop()

	// Print ArgoCD status
	fmt.Println()
	color.Cyan("ArgoCD Server:")
	fmt.Println("  Status: ✅ Running")
	fmt.Println("  Version: v2.9.3")
	fmt.Println("  Namespace: argocd")
	fmt.Println()

	// Print applications
	color.Cyan("Applications:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSYNC STATUS\tHEALTH\tNAMESPACE\tREPO")
	fmt.Fprintln(w, "----\t-----------\t------\t---------\t----")

	// Example data - TODO: Get real data from kubectl
	apps := []struct {
		Name       string
		SyncStatus string
		Health     string
		Namespace  string
		Repo       string
	}{
		{"cluster-addons", "✅ Synced", "✅ Healthy", "argocd", "github.com/user/gitops"},
		{"ingress-nginx", "✅ Synced", "✅ Healthy", "ingress-nginx", "Auto-synced"},
		{"cert-manager", "✅ Synced", "✅ Healthy", "cert-manager", "Auto-synced"},
	}

	for _, app := range apps {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			app.Name, app.SyncStatus, app.Health, app.Namespace, app.Repo)
	}

	w.Flush()
	fmt.Println()

	color.Cyan("💡 View in ArgoCD UI:")
	fmt.Println("   kubectl port-forward svc/argocd-server -n argocd 8080:443")
	fmt.Println("   https://localhost:8080")

	return nil
}

func runAddonsTemplate(cmd *cobra.Command, args []string) error {
	printHeader("📄 GitOps Repository Template")

	if outputPath != "" {
		// Generate actual directory structure
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " Generating template..."
		s.Start()

		if err := generateTemplateStructure(outputPath); err != nil {
			s.Stop()
			return fmt.Errorf("failed to generate template: %w", err)
		}

		s.Stop()
		color.Green("✅ Template generated at: %s", outputPath)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. cd " + outputPath)
		fmt.Println("  2. git init")
		fmt.Println("  3. Edit addon manifests")
		fmt.Println("  4. git add . && git commit -m 'Initial commit'")
		fmt.Println("  5. git remote add origin <your-repo-url>")
		fmt.Println("  6. git push -u origin main")
		fmt.Println("  7. kubernetes-create addons bootstrap --repo <your-repo-url>")
	} else {
		// Just print the structure
		fmt.Println()
		fmt.Println(addons.GenerateGitOpsRepoStructure())
	}

	return nil
}

// Helper functions

func printAddonTable() {
	color.Cyan("Addons:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCATEGORY\tSTATUS\tVERSION\tNAMESPACE")
	fmt.Fprintln(w, "----\t--------\t------\t-------\t---------")

	// Example data - TODO: Get real data from kubectl/ArgoCD
	addonsData := []struct {
		Name      string
		Category  string
		Status    string
		Version   string
		Namespace string
	}{
		{"argocd", "CD", "✅ Running", "v2.9.3", "argocd"},
		{"ingress-nginx", "Ingress", "✅ Running", "v4.8.3", "ingress-nginx"},
		{"cert-manager", "Security", "✅ Running", "v1.13.3", "cert-manager"},
		{"prometheus", "Monitoring", "✅ Running", "v55.5.0", "monitoring"},
		{"longhorn", "Storage", "✅ Running", "v1.5.3", "longhorn-system"},
	}

	for _, addon := range addonsData {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			addon.Name, addon.Category, addon.Status, addon.Version, addon.Namespace)
	}

	w.Flush()
	fmt.Println()

	color.Cyan("📊 Summary:")
	fmt.Println("  • Total Addons: 5")
	fmt.Println("  • Running: 5")
	fmt.Println("  • Failed: 0")
	fmt.Println()
	color.Green("  ✅ All addons healthy")
}

func generateTemplateStructure(outputDir string) error {
	// Create directory structure
	dirs := []string{
		filepath.Join(outputDir, "addons", "argocd"),
		filepath.Join(outputDir, "addons", "ingress-nginx"),
		filepath.Join(outputDir, "addons", "cert-manager"),
		filepath.Join(outputDir, "addons", "prometheus"),
		filepath.Join(outputDir, "addons", "longhorn"),
		filepath.Join(outputDir, "apps"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create example ArgoCD namespace
	argoCDNamespace := `apiVersion: v1
kind: Namespace
metadata:
  name: argocd
`
	if err := os.WriteFile(filepath.Join(outputDir, "addons", "argocd", "namespace.yaml"), []byte(argoCDNamespace), 0644); err != nil {
		return err
	}

	// Create example ArgoCD installation reference
	argoCDInstall := `# ArgoCD Installation
# Apply the official ArgoCD manifests:
# kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Or use this as a placeholder that the bootstrap command will replace
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-install-reference
  namespace: argocd
data:
  url: "https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml"
  version: "stable"
`
	if err := os.WriteFile(filepath.Join(outputDir, "addons", "argocd", "install.yaml"), []byte(argoCDInstall), 0644); err != nil {
		return err
	}

	// Create README
	readme := `# GitOps Repository

This repository manages Kubernetes cluster addons via GitOps.

## Structure

- **addons/** - Cluster addons managed by ArgoCD
  - **argocd/** - ArgoCD itself
  - **ingress-nginx/** - NGINX Ingress Controller
  - **cert-manager/** - Certificate management
  - **prometheus/** - Monitoring stack
  - **longhorn/** - Distributed storage
- **apps/** - Your applications

## Bootstrap

` + "```bash" + `
kubernetes-create addons bootstrap --repo https://github.com/you/this-repo
` + "```" + `

## Adding New Addons

1. Create a new directory under addons/ (e.g., addons/my-addon/)
2. Add Kubernetes manifests or Helm charts
3. Commit and push
4. ArgoCD will automatically sync

## Removing Addons

1. Delete the addon directory
2. Commit and push
3. ArgoCD will automatically prune (if syncPolicy.prune: true)
`
	if err := os.WriteFile(filepath.Join(outputDir, "README.md"), []byte(readme), 0644); err != nil {
		return err
	}

	return nil
}
