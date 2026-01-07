package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optremove"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/chalkan3/sloth-kubernetes/internal/common"
)

var stacksCmd = &cobra.Command{
	Use:   "stacks",
	Short: "Manage deployment stacks",
	Long:  `List, inspect, and manage Pulumi stacks for different clusters`,
}

var listStacksCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployment stacks",
	Long:  `Display all available stacks with their status and last update time`,
	Example: `  # List all stacks
  sloth-kubernetes stacks list`,
	RunE: runListStacks,
}

var stackInfoCmd = &cobra.Command{
	Use:   "info [stack-name]",
	Short: "Show detailed stack information",
	Long:  `Display detailed information about a specific stack including resources and outputs`,
	Example: `  # Show stack info
  sloth-kubernetes stacks info production`,
	RunE: runStackInfo,
}

var deleteStackCmd = &cobra.Command{
	Use:   "delete [stack-name]",
	Short: "Delete a stack",
	Long:  `Delete a stack and optionally destroy all its resources`,
	Example: `  # Delete a stack
  sloth-kubernetes stacks delete old-cluster

  # Delete stack and destroy resources
  sloth-kubernetes stacks delete old-cluster --destroy`,
	RunE: runDeleteStack,
}

var outputCmd = &cobra.Command{
	Use:   "output [stack-name]",
	Short: "Show stack outputs",
	Long:  `Display all outputs from a stack, including cluster endpoints, IPs, and credentials`,
	Example: `  # Show all outputs
  sloth-kubernetes stacks output production

  # Show specific output
  sloth-kubernetes stacks output production --key kubeconfig

  # Export outputs as JSON
  sloth-kubernetes stacks output production --json`,
	RunE: runStackOutput,
}

var selectStackCmd = &cobra.Command{
	Use:   "select [stack-name]",
	Short: "Select current stack",
	Long:  `Set the current active stack for subsequent operations`,
	Example: `  # Select production stack
  sloth-kubernetes stacks select production`,
	RunE: runSelectStack,
}

var exportStackCmd = &cobra.Command{
	Use:   "export [stack-name]",
	Short: "Export stack state",
	Long:  `Export the complete stack state to a JSON file for backup or migration`,
	Example: `  # Export stack to file
  sloth-kubernetes stacks export production --output production-backup.json`,
	RunE: runExportStack,
}

var importStackCmd = &cobra.Command{
	Use:   "import [stack-name] [file]",
	Short: "Import stack state",
	Long:  `Import stack state from a previously exported JSON file`,
	Example: `  # Import stack from file
  sloth-kubernetes stacks import production production-backup.json`,
	RunE: runImportStack,
}

var currentStackCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current selected stack",
	Long:  `Display the currently selected stack name`,
	Example: `  # Show current stack
  sloth-kubernetes stacks current`,
	RunE: runCurrentStack,
}

var renameStackCmd = &cobra.Command{
	Use:   "rename [old-name] [new-name]",
	Short: "Rename a stack",
	Long:  `Rename an existing stack to a new name`,
	Example: `  # Rename stack
  sloth-kubernetes stacks rename old-name new-name`,
	RunE: runRenameStack,
}

var cancelCmd = &cobra.Command{
	Use:   "cancel [stack-name]",
	Short: "Cancel and unlock a stack",
	Long:  `Remove stale locks from a stack that was interrupted or crashed`,
	Example: `  # Cancel/unlock a stack
  sloth-kubernetes stacks cancel home-cluster

  # Cancel using shorthand
  sloth-kubernetes cancel home-cluster`,
	RunE: runCancel,
}

var createStackCmd = &cobra.Command{
	Use:   "create [stack-name]",
	Short: "Create a new stack with encryption (passphrase or KMS)",
	Long: `Create a new Pulumi stack with encrypted secrets.

Encryption options:
  1. AWS KMS (recommended for production):
     - --kms-key: Use AWS KMS key for encryption
     - Supports key ARN, key ID, or alias

  2. Passphrase-based (AES-256-GCM):
     - --password-stdin: Read from stdin
     - Environment variable: PULUMI_CONFIG_PASSPHRASE
     - Interactive prompt (if neither above is provided)

The encryption configuration is saved for subsequent operations.`,
	Example: `  # Create stack with AWS KMS encryption (recommended for production)
  sloth-kubernetes stacks create production --kms-key alias/sloth-secrets

  # Create stack with KMS key ARN
  sloth-kubernetes stacks create production --kms-key arn:aws:kms:us-east-1:123456789:key/abcd-1234

  # Create stack with password from stdin
  echo "my-secure-password" | sloth-kubernetes stacks create production --password-stdin

  # Create stack interactively (will prompt for password)
  sloth-kubernetes stacks create production`,
	RunE: runCreateStack,
}

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage stack state",
	Long:  `View and manipulate Pulumi stack state including resources`,
}

var stateDeleteCmd = &cobra.Command{
	Use:   "delete [stack-name] [urn]",
	Short: "Delete a resource from stack state",
	Long: `Remove a specific resource from the stack state by its URN.
This does NOT destroy the actual cloud resource, only removes it from Pulumi's state.

WARNING: This is a dangerous operation. Use with caution!`,
	Example: `  # Delete a resource by URN
  sloth-kubernetes stacks state delete production urn:pulumi:production::sloth-kubernetes::digitalocean:Droplet::master-1

  # Force delete without confirmation
  sloth-kubernetes stacks state delete production <urn> --force`,
	RunE: runStateDelete,
}

var stateListCmd = &cobra.Command{
	Use:   "list [stack-name]",
	Short: "List all resources in stack state",
	Long:  `Display all resources currently tracked in the stack state with their URNs and types`,
	Example: `  # List all resources in stack
  sloth-kubernetes stacks state list production

  # List with filtering
  sloth-kubernetes stacks state list production --type digitalocean:Droplet`,
	RunE: runStateList,
}

var stateDiffCmd = &cobra.Command{
	Use:   "diff [stack1] [stack2]",
	Short: "Compare state between two stacks",
	Long: `Compare resources between two stacks to identify differences.
Useful for comparing production vs staging, or before/after states.`,
	Example: `  # Compare two stacks
  sloth-kubernetes stacks state diff production staging

  # Compare with exported file
  sloth-kubernetes stacks state diff production --file backup.json`,
	RunE: runStateDiff,
}

var stateRepairCmd = &cobra.Command{
	Use:   "repair [stack-name]",
	Short: "Repair corrupted stack state",
	Long: `Attempt to repair a corrupted stack state by:
- Removing duplicate resources
- Fixing invalid URNs
- Cleaning up orphaned dependencies`,
	Example: `  # Repair stack state
  sloth-kubernetes stacks state repair production

  # Dry-run to see what would be fixed
  sloth-kubernetes stacks state repair production --dry-run`,
	RunE: runStateRepair,
}

var stateUnprotectCmd = &cobra.Command{
	Use:   "unprotect [stack-name] [urn]",
	Short: "Remove protection from a resource",
	Long: `Remove the 'protect' flag from a resource, allowing it to be deleted.
Some resources are marked as protected to prevent accidental deletion.`,
	Example: `  # Unprotect a specific resource
  sloth-kubernetes stacks state unprotect production urn:pulumi:...

  # Unprotect all resources
  sloth-kubernetes stacks state unprotect production --all`,
	RunE: runStateUnprotect,
}

var stateBulkDeleteCmd = &cobra.Command{
	Use:   "bulk-delete [stack-name]",
	Short: "Delete multiple resources from state",
	Long: `Remove multiple resources from stack state at once.
Supports filtering by type, pattern matching, or providing a list of URNs.`,
	Example: `  # Delete all resources of a type
  sloth-kubernetes stacks state bulk-delete production --type digitalocean:Droplet

  # Delete resources matching pattern
  sloth-kubernetes stacks state bulk-delete production --pattern "worker-*"

  # Delete from file with URN list
  sloth-kubernetes stacks state bulk-delete production --file urns.txt`,
	RunE: runStateBulkDelete,
}

var stateMoveCmd = &cobra.Command{
	Use:   "move [source-stack] [target-stack] [urn]",
	Short: "Move a resource between stacks",
	Long: `Move a resource from one stack to another.
The resource is removed from the source stack and added to the target stack.`,
	Example: `  # Move a single resource
  sloth-kubernetes stacks state move staging production urn:pulumi:...

  # Move multiple resources by type
  sloth-kubernetes stacks state move staging production --type digitalocean:Droplet`,
	RunE: runStateMove,
}

var (
	destroyStack  bool
	outputKey     string
	outputJSON    bool
	exportOutput  string
	forceDelete   bool
	resourceType  string
	diffFile      string
	stateDryRun   bool
	unprotectAll  bool
	bulkPattern   string
	bulkFile      string
	moveType      string
	passwordStdin bool
	kmsKey        string
)

func init() {
	// Add stacks command to root for direct access
	rootCmd.AddCommand(stacksCmd)

	// Add subcommands
	stacksCmd.AddCommand(listStacksCmd)
	stacksCmd.AddCommand(stackInfoCmd)
	stacksCmd.AddCommand(createStackCmd)
	stacksCmd.AddCommand(deleteStackCmd)
	stacksCmd.AddCommand(outputCmd)
	stacksCmd.AddCommand(selectStackCmd)
	stacksCmd.AddCommand(currentStackCmd)
	stacksCmd.AddCommand(exportStackCmd)
	stacksCmd.AddCommand(importStackCmd)
	stacksCmd.AddCommand(renameStackCmd)
	stacksCmd.AddCommand(cancelCmd)
	stacksCmd.AddCommand(stateCmd)

	// State subcommands
	stateCmd.AddCommand(stateDeleteCmd)
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(stateDiffCmd)
	stateCmd.AddCommand(stateRepairCmd)
	stateCmd.AddCommand(stateUnprotectCmd)
	stateCmd.AddCommand(stateBulkDeleteCmd)
	stateCmd.AddCommand(stateMoveCmd)

	// Delete flags
	deleteStackCmd.Flags().BoolVar(&destroyStack, "destroy", false, "Destroy all resources before deleting stack")
	deleteStackCmd.Flags().BoolVar(&forceDelete, "force", false, "Force delete stack even if it has resources")

	// Output flags
	outputCmd.Flags().StringVar(&outputKey, "key", "", "Show specific output key")
	outputCmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	// Export flags
	exportStackCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (default: <stack-name>-state.json)")

	// State delete flags
	stateDeleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete without confirmation")

	// State list flags
	stateListCmd.Flags().StringVar(&resourceType, "type", "", "Filter by resource type (e.g., digitalocean:Droplet)")

	// State diff flags
	stateDiffCmd.Flags().StringVarP(&diffFile, "file", "f", "", "Compare with exported state file instead of another stack")

	// State repair flags
	stateRepairCmd.Flags().BoolVar(&stateDryRun, "dry-run", false, "Show what would be repaired without making changes")

	// State unprotect flags
	stateUnprotectCmd.Flags().BoolVar(&unprotectAll, "all", false, "Unprotect all resources in the stack")

	// State bulk-delete flags
	stateBulkDeleteCmd.Flags().StringVar(&resourceType, "type", "", "Delete all resources of this type")
	stateBulkDeleteCmd.Flags().StringVar(&bulkPattern, "pattern", "", "Delete resources matching this URN pattern")
	stateBulkDeleteCmd.Flags().StringVarP(&bulkFile, "file", "f", "", "File containing list of URNs to delete")
	stateBulkDeleteCmd.Flags().BoolVar(&forceDelete, "force", false, "Force delete without confirmation")

	// State move flags
	stateMoveCmd.Flags().StringVar(&moveType, "type", "", "Move all resources of this type")

	// Create stack flags
	createStackCmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "Read passphrase from stdin")
	createStackCmd.Flags().StringVar(&kmsKey, "kms-key", "", "AWS KMS key ARN or alias for encryption (e.g., alias/my-key or arn:aws:kms:...)")
}

// createWorkspaceWithS3Support creates a Pulumi workspace with S3/MinIO backend support
func createWorkspaceWithS3Support(ctx context.Context) (auto.Workspace, error) {
	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	projectName := "sloth-kubernetes"

	// Build project configuration with optional backend
	project := workspace.Project{
		Name:    tokens.PackageName(projectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}

	// If PULUMI_BACKEND_URL is set, configure backend in the project
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		project.Backend = &workspace.ProjectBackend{
			URL: backendURL,
		}
	}

	workspaceOpts := []auto.LocalWorkspaceOption{
		auto.Project(project),
	}

	// Collect all AWS/S3 environment variables to pass to Pulumi subprocess
	envVars := make(map[string]string)
	awsEnvKeys := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
		"AWS_S3_ENDPOINT",
		"AWS_S3_USE_PATH_STYLE",
		"AWS_S3_FORCE_PATH_STYLE",
		"PULUMI_BACKEND_URL",
		"PULUMI_CONFIG_PASSPHRASE",
	}
	for _, key := range awsEnvKeys {
		if val := os.Getenv(key); val != "" {
			envVars[key] = val
		}
	}

	// Add environment variables to workspace options
	if len(envVars) > 0 {
		workspaceOpts = append(workspaceOpts, auto.EnvVars(envVars))
	}

	// If PULUMI_BACKEND_URL is set, use passphrase secrets provider
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		workspaceOpts = append(workspaceOpts, auto.SecretsProvider("passphrase"))
		if os.Getenv("PULUMI_CONFIG_PASSPHRASE") == "" {
			os.Setenv("PULUMI_CONFIG_PASSPHRASE", "")
			envVars["PULUMI_CONFIG_PASSPHRASE"] = ""
		}
	}

	return auto.NewLocalWorkspace(ctx, workspaceOpts...)
}

func runListStacks(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	printHeader("📦 Deployment Stacks")

	// Create workspace with S3 support
	ws, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// List stacks
	stacks, err := ws.ListStacks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	if len(stacks) == 0 {
		color.Yellow("\n⚠️  No stacks found")
		fmt.Println()
		color.Cyan("Create a new stack with:")
		fmt.Println("  sloth-kubernetes deploy <stack-name> --config cluster.lisp")
		return nil
	}

	fmt.Println()
	printStacksTable(stacks)

	return nil
}

func runStackInfo(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks info <stack-name>")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("📊 Stack Info: %s", stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	s, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Display basic info
	fmt.Println()
	color.New(color.Bold).Println("Stack Information:")
	fmt.Printf("  • Name: %s\n", stackName)

	// Get outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		color.Yellow("⚠️  Could not get stack outputs")
	} else {
		fmt.Println()
		printStackOutputs(outputs)
	}

	return nil
}

func runDeleteStack(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks delete <stack-name>")
	}

	targetStackName := args[0]

	printHeader(fmt.Sprintf("🗑️  Deleting Stack: %s", targetStackName))

	if destroyStack {
		color.Red("⚠️  This will DESTROY all resources in the stack!")
		color.Yellow("Resources will be destroyed before stack removal.")
	} else if forceDelete {
		color.Red("⚠️  Using --force: Stack will be removed even if it still has resources!")
		color.Yellow("This may leave orphaned cloud resources that need manual cleanup.")
	} else {
		color.Yellow("Stack will be deleted but resources will remain (use --destroy to remove resources)")
		color.Yellow("Use --force to remove stacks with orphaned resources")
	}

	// Confirm unless --yes is passed
	if !autoApprove {
		fmt.Println()
		if !confirm(fmt.Sprintf("Are you sure you want to delete stack '%s'?", targetStackName)) {
			color.Yellow("Deletion cancelled")
			return nil
		}
	}

	fmt.Println()

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// If --destroy flag is set, destroy resources first
	if destroyStack {
		fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", targetStackName)
		stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
		if err != nil {
			return fmt.Errorf("failed to select stack for destroy: %w", err)
		}

		fmt.Println("Destroying resources...")
		_, err = stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
		if err != nil {
			return fmt.Errorf("failed to destroy resources: %w", err)
		}
		printSuccess("Resources destroyed")
	}

	// Remove the stack
	fmt.Printf("Removing stack '%s'...\n", targetStackName)
	if forceDelete {
		color.Yellow("Using --force: Stack will be removed even if it has resources")
		err = workspace.RemoveStack(ctx, targetStackName, optremove.Force())
	} else {
		err = workspace.RemoveStack(ctx, targetStackName)
	}
	if err != nil {
		return fmt.Errorf("failed to remove stack: %w", err)
	}

	fmt.Println()
	printSuccess(fmt.Sprintf("Stack '%s' deleted successfully", targetStackName))

	return nil
}

func printStacksTable(stacks []auto.StackSummary) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	color.New(color.Bold).Fprintln(w, "NAME\tLAST UPDATE\tRESOURCE COUNT\tURL")
	fmt.Fprintln(w, "----\t-----------\t--------------\t---")

	// Get backend URL from environment (set by createWorkspaceWithS3Support)
	backendURL := os.Getenv("PULUMI_BACKEND_URL")

	for _, stack := range stacks {
		lastUpdate := "Never"
		if stack.LastUpdate != "" {
			// Parse if possible, otherwise just show the string
			lastUpdate = stack.LastUpdate
		}

		resourceCount := "?"
		if stack.ResourceCount != nil {
			resourceCount = fmt.Sprintf("%d", *stack.ResourceCount)
		}

		url := stack.URL
		if url == "" || url == "local://" {
			// If URL is empty and we have a backend URL, use it
			if backendURL != "" {
				// Extract just the S3 bucket part for cleaner display
				// Format: s3://bucket?endpoint=...&params
				if strings.HasPrefix(backendURL, "s3://") {
					// Extract bucket name
					parts := strings.Split(strings.TrimPrefix(backendURL, "s3://"), "?")
					if len(parts) > 0 {
						url = "s3://" + parts[0]
					} else {
						url = backendURL
					}
				} else {
					url = backendURL
				}
			} else {
				url = "local://"
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", stack.Name, lastUpdate, resourceCount, url)
	}
}

func printStackOutputs(outputs auto.OutputMap) {
	if len(outputs) == 0 {
		color.Yellow("No outputs available")
		return
	}

	color.New(color.Bold).Println("Outputs:")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	color.New(color.Bold).Fprintln(w, "  KEY\tVALUE\tSECRET")
	fmt.Fprintln(w, "  ---\t-----\t------")

	for key, output := range outputs {
		value := fmt.Sprintf("%v", output.Value)
		if len(value) > 60 {
			value = value[:57] + "..."
		}

		secret := ""
		if output.Secret {
			secret = "🔒"
			value = "***REDACTED***"
		}

		fmt.Fprintf(w, "  %s\t%s\t%s\n", key, value, secret)
	}
}

func formatTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

func runStackOutput(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks output <stack-name>")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("📤 Stack Outputs: %s", stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Use fully qualified stack name for S3 backend
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	s, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Get outputs
	outputs, err := s.Outputs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get outputs: %w", err)
	}

	if len(outputs) == 0 {
		color.Yellow("\n⚠️  No outputs available for this stack")
		return nil
	}

	fmt.Println()

	// Show specific key
	if outputKey != "" {
		output, exists := outputs[outputKey]
		if !exists {
			return fmt.Errorf("output key '%s' not found", outputKey)
		}

		if outputJSON {
			fmt.Printf("{\n  \"%s\": %v\n}\n", outputKey, output.Value)
		} else {
			value := fmt.Sprintf("%v", output.Value)
			if output.Secret {
				value = "***REDACTED***"
			}
			fmt.Printf("%s: %s\n", outputKey, value)
		}
		return nil
	}

	// Show all outputs
	if outputJSON {
		fmt.Println("{")
		i := 0
		for key, output := range outputs {
			value := output.Value
			if output.Secret {
				value = "***REDACTED***"
			}
			if i > 0 {
				fmt.Println(",")
			}
			fmt.Printf("  \"%s\": %v", key, value)
			i++
		}
		fmt.Println("\n}")
	} else {
		printStackOutputs(outputs)
	}

	return nil
}

func runSelectStack(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks select <stack-name>")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("🎯 Selecting Stack: %s", stackName))

	// Verify stack exists
	workspace, err := auto.NewLocalWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	_, err = auto.SelectStack(ctx, stackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w\n\nAvailable stacks:\n  Use 'sloth-kubernetes stacks list' to see all stacks", stackName, err)
	}

	// Save to config file
	configPath := ".sloth-stack"
	if err := os.WriteFile(configPath, []byte(stackName), 0644); err != nil {
		return fmt.Errorf("failed to save stack selection: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack '%s' is now selected", stackName)
	fmt.Println()
	color.Cyan("All subsequent commands will use this stack by default")
	fmt.Println("  (You can override with --stack flag)")

	return nil
}

func runExportStack(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks export <stack-name>")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("💾 Exporting Stack: %s", stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Select stack using fully qualified name
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Export stack state
	fmt.Println()
	color.Cyan("⏳ Exporting stack state...")

	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// Determine output file
	outputFile := exportOutput
	if outputFile == "" {
		outputFile = fmt.Sprintf("%s-state.json", stackName)
	}

	// Create export wrapper with metadata
	exportData := map[string]interface{}{
		"version":    deployment.Version,
		"deployment": json.RawMessage(deployment.Deployment),
		"metadata": map[string]interface{}{
			"stack":      stackName,
			"exportedAt": time.Now().UTC().Format(time.RFC3339),
			"exportedBy": "sloth-kubernetes",
		},
	}

	// Marshal with indentation for readability
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack exported successfully")
	fmt.Printf("\n  File: %s\n", outputFile)
	fmt.Printf("  Size: %d bytes\n", len(jsonData))
	fmt.Println()
	color.Cyan("To import later:")
	fmt.Printf("  sloth-kubernetes stacks import %s %s\n", stackName, outputFile)

	return nil
}

func runImportStack(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes stacks import <stack-name> <file>")
	}

	ctx := context.Background()
	stackName := args[0]
	filePath := args[1]

	printHeader(fmt.Sprintf("📥 Importing Stack: %s", stackName))

	// Read import file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse export wrapper
	var exportData struct {
		Version    int                    `json:"version"`
		Deployment json.RawMessage        `json:"deployment"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(fileData, &exportData); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	// Show metadata if available
	if exportData.Metadata != nil {
		fmt.Println()
		color.Cyan("📋 Import Metadata:")
		if stack, ok := exportData.Metadata["stack"].(string); ok {
			fmt.Printf("  Original Stack: %s\n", stack)
		}
		if exportedAt, ok := exportData.Metadata["exportedAt"].(string); ok {
			fmt.Printf("  Exported At: %s\n", exportedAt)
		}
	}

	// Confirm import
	fmt.Println()
	color.Yellow("⚠️  This will REPLACE the entire stack state!")
	color.Yellow("   All existing resources in the stack will be overwritten.")
	fmt.Println()

	if !autoApprove {
		if !confirm(fmt.Sprintf("Import state into stack '%s'?", stackName)) {
			color.Yellow("Import cancelled")
			return nil
		}
	}

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Try to select existing stack, or create new one
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		// Stack doesn't exist, create it
		color.Cyan("Stack doesn't exist, creating new stack...")
		stack, err = auto.NewStack(ctx, fullyQualifiedStackName, workspace)
		if err != nil {
			return fmt.Errorf("failed to create stack '%s': %w", stackName, err)
		}
	}

	// Create deployment object
	deployment := apitype.UntypedDeployment{
		Version:    exportData.Version,
		Deployment: exportData.Deployment,
	}

	// Import state
	fmt.Println()
	color.Cyan("⏳ Importing stack state...")

	if err := stack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import stack state: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack imported successfully")
	fmt.Printf("\n  Stack: %s\n", stackName)
	fmt.Printf("  Source: %s\n", filePath)
	fmt.Println()
	color.Cyan("Next steps:")
	fmt.Println("  • Run 'sloth-kubernetes stacks info " + stackName + "' to verify")
	fmt.Println("  • Run 'sloth-kubernetes refresh --stack " + stackName + "' to sync with cloud")

	return nil
}

func runCurrentStack(cmd *cobra.Command, args []string) error {
	printHeader("🎯 Current Stack")

	// Try to read from config file
	configPath := ".sloth-stack"
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			color.Yellow("\n⚠️  No stack currently selected")
			fmt.Println()
			color.Cyan("Select a stack with:")
			fmt.Println("  sloth-kubernetes stacks select <stack-name>")
			fmt.Println()
			color.Cyan("Or use --stack flag in commands:")
			fmt.Println("  sloth-kubernetes deploy --stack production")
			return nil
		}
		return fmt.Errorf("failed to read stack selection: %w", err)
	}

	currentStack := string(data)

	fmt.Println()
	color.Green("✅ Current stack: %s", currentStack)
	fmt.Println()
	color.Cyan("💡 Commands will use this stack by default")
	fmt.Println("  (Override with --stack flag)")

	return nil
}

func runRenameStack(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes stacks rename <old-name> <new-name>")
	}

	ctx := context.Background()
	oldName := args[0]
	newName := args[1]

	printHeader(fmt.Sprintf("✏️  Renaming Stack: %s → %s", oldName, newName))

	// Get workspace
	workspace, err := auto.NewLocalWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Verify old stack exists
	oldStack, err := auto.SelectStack(ctx, oldName, workspace)
	if err != nil {
		return fmt.Errorf("failed to find stack '%s': %w", oldName, err)
	}

	// Export old stack
	deployment, err := oldStack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export old stack: %w", err)
	}

	// Create new stack
	newStack, err := auto.NewStack(ctx, newName, workspace)
	if err != nil {
		return fmt.Errorf("failed to create new stack: %w", err)
	}

	// Import into new stack
	if err := newStack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import into new stack: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack renamed successfully")
	fmt.Printf("\n  Old name: %s\n", oldName)
	fmt.Printf("  New name: %s\n", newName)
	fmt.Println()
	color.Yellow("⚠️  The old stack still exists. To remove it:")
	fmt.Printf("  sloth-kubernetes stacks delete %s\n", oldName)

	return nil
}

func runStateList(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state list <stack-name>")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("📋 Stack State: %s", stackName))

	// Get workspace and stack
	workspace, err := auto.NewLocalWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	stack, err := auto.SelectStack(ctx, stackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Export stack to get state
	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// The deployment is stored as JSON, we need to parse it
	var deploymentData struct {
		Resources []struct {
			URN  string      `json:"urn"`
			Type string      `json:"type"`
			ID   interface{} `json:"id"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(deployment.Deployment, &deploymentData); err != nil {
		return fmt.Errorf("failed to parse deployment: %w", err)
	}

	resources := deploymentData.Resources

	if len(resources) == 0 {
		color.Yellow("\n⚠️  No resources found in stack")
		return nil
	}

	fmt.Println()
	color.New(color.Bold).Printf("Total resources: %d\n\n", len(resources))

	// Print resources table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	color.New(color.Bold).Fprintln(w, "URN\tTYPE\tID")
	fmt.Fprintln(w, "---\t----\t--")

	for _, resource := range resources {
		// Filter by type if specified
		if resourceType != "" && resource.Type != resourceType {
			continue
		}

		idStr := fmt.Sprintf("%v", resource.ID)
		if len(idStr) > 60 {
			idStr = idStr[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", resource.URN, resource.Type, idStr)
	}

	return nil
}

func runStateDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state delete <stack-name> <urn>")
	}

	ctx := context.Background()
	stackName := args[0]
	urn := args[1]

	printHeader(fmt.Sprintf("🗑️  Delete Resource from State: %s", stackName))

	fmt.Println()
	color.Red("⚠️  WARNING: This will remove the resource from Pulumi state!")
	fmt.Println()
	color.Yellow("This operation:")
	fmt.Println("  ✓ Removes the resource from Pulumi's tracking")
	fmt.Println("  ✗ Does NOT destroy the actual cloud resource")
	fmt.Println("  ⚠️  The resource will become unmanaged by Pulumi")
	fmt.Println()
	fmt.Printf("Stack: %s\n", stackName)
	fmt.Printf("URN:   %s\n", urn)
	fmt.Println()

	// Confirm unless --force
	if !forceDelete {
		fmt.Print("Are you sure you want to continue? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			color.Yellow("\n❌ Operation cancelled")
			return nil
		}
	}

	// Get workspace and stack
	workspace, err := auto.NewLocalWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	stack, err := auto.SelectStack(ctx, stackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Use pulumi CLI to delete state
	fmt.Println()
	color.Cyan("Deleting resource from state...")

	// The Automation API doesn't have direct state delete, so we use the CLI
	// Export, modify, and import back
	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse deployment JSON
	var deploymentData struct {
		Resources []map[string]interface{} `json:"resources"`
	}

	if err := json.Unmarshal(deployment.Deployment, &deploymentData); err != nil {
		return fmt.Errorf("failed to parse deployment: %w", err)
	}

	// Find and remove the resource
	found := false
	newResources := []map[string]interface{}{}
	for _, resource := range deploymentData.Resources {
		resourceURN, _ := resource["urn"].(string)
		if resourceURN != urn {
			newResources = append(newResources, resource)
		} else {
			found = true
			resourceType, _ := resource["type"].(string)
			color.Yellow("  Found resource: %s (Type: %s)", resourceURN, resourceType)
		}
	}

	if !found {
		return fmt.Errorf("resource with URN '%s' not found in stack", urn)
	}

	// Update deployment
	deploymentData.Resources = newResources

	// Marshal back to JSON
	modifiedDeployment, err := json.Marshal(deploymentData)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	deployment.Deployment = modifiedDeployment

	// Import modified state back
	if err := stack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import modified state: %w", err)
	}

	fmt.Println()
	color.Green("✅ Resource removed from state successfully")
	fmt.Println()
	color.Cyan("Next steps:")
	fmt.Println("  1. The cloud resource still exists and is now unmanaged")
	fmt.Println("  2. You can manually delete it from the cloud provider console")
	fmt.Println("  3. Or import it back with: sloth-kubernetes pulumi import <type> <name> <id>")

	return nil
}

func runCancel(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("stack name required")
	}

	stackName := args[0]
	ctx := context.Background()

	printHeader("🔓 Canceling Stack Operations")
	fmt.Printf("Stack: %s\n\n", color.CyanString(stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Select the stack using the fully qualified stack name format
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	color.Yellow("⏳ Canceling ongoing operations and removing locks...")

	// Cancel the stack - this removes locks
	err = stack.Cancel(ctx)
	if err != nil {
		return fmt.Errorf("failed to cancel stack: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack unlocked successfully")
	fmt.Println()
	color.Cyan("Next steps:")
	fmt.Println("  • You can now run deploy, destroy, or other operations on this stack")
	fmt.Println("  • If there were running operations, they have been cancelled")

	return nil
}

func runCreateStack(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks create <stack-name> [--password-stdin | --kms-key <key>]")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("🔐 Creating Stack: %s", stackName))

	var secretsProvider string
	var encryptionType string

	// Check if KMS key is provided
	if kmsKey != "" {
		// Use AWS KMS encryption
		secretsProvider = formatKMSProvider(kmsKey)
		encryptionType = "AWS KMS"

		fmt.Println()
		color.Cyan("⏳ Creating stack with AWS KMS encryption...")
		color.Cyan("   KMS Key: %s", kmsKey)

		// Save KMS key to config
		if err := saveKMSKey(kmsKey); err != nil {
			color.Yellow("⚠️  Warning: Could not save KMS key to config: %v", err)
		}
	} else {
		// Use passphrase-based encryption
		passphrase, err := getPassphrase()
		if err != nil {
			return fmt.Errorf("failed to get passphrase: %w", err)
		}

		if passphrase == "" {
			return fmt.Errorf("passphrase cannot be empty")
		}

		// Validate passphrase strength
		if len(passphrase) < 8 {
			color.Yellow("⚠️  Warning: Passphrase is short (less than 8 characters)")
			fmt.Println("   Consider using a longer passphrase for better security")
			fmt.Println()
		}

		// Set the passphrase as environment variable for Pulumi
		os.Setenv("PULUMI_CONFIG_PASSPHRASE", passphrase)
		secretsProvider = "passphrase"
		encryptionType = "Passphrase (AES-256-GCM)"

		// Save passphrase to config file
		if err := savePassphrase(passphrase); err != nil {
			color.Yellow("⚠️  Warning: Could not save passphrase to config: %v", err)
			fmt.Println("   You will need to provide it again for future operations")
		}

		fmt.Println()
		color.Cyan("⏳ Creating stack with passphrase encryption...")
	}

	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	// Create workspace with secrets provider
	workspace, err := createWorkspaceWithSecretsProvider(ctx, secretsProvider)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create the stack
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	_, err = auto.NewStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		// Check if stack already exists
		if strings.Contains(err.Error(), "already exists") {
			color.Yellow("\n⚠️  Stack '%s' already exists", stackName)
			fmt.Println()
			color.Cyan("To use this stack:")
			fmt.Printf("  sloth-kubernetes deploy %s --config your-config.lisp\n", stackName)
			return nil
		}
		return fmt.Errorf("failed to create stack: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack '%s' created successfully", stackName)
	fmt.Println()
	color.New(color.Bold).Println("Stack Details:")
	fmt.Printf("  • Name: %s\n", stackName)
	fmt.Printf("  • Encryption: %s\n", encryptionType)
	if kmsKey != "" {
		fmt.Printf("  • KMS Key: %s\n", kmsKey)
	}
	fmt.Printf("  • Config saved: ~/.sloth-kubernetes/config\n")
	fmt.Println()
	color.Cyan("Next steps:")
	fmt.Printf("  1. Deploy your cluster:\n")
	fmt.Printf("     sloth-kubernetes deploy %s --config your-config.lisp\n", stackName)
	fmt.Println()

	if kmsKey != "" {
		color.Yellow("⚠️  Important: Ensure AWS credentials have access to the KMS key!")
		fmt.Println("   The key must allow Encrypt and Decrypt operations.")
	} else {
		color.Yellow("⚠️  Important: Keep your passphrase safe!")
		fmt.Println("   You will need it to access encrypted outputs and manage this stack.")
	}

	return nil
}

// formatKMSProvider formats the KMS key into a Pulumi secrets provider URL
func formatKMSProvider(key string) string {
	// If it's a full ARN, extract key ID and region
	// ARN format: arn:aws:kms:<region>:<account-id>:key/<key-id>
	if strings.HasPrefix(key, "arn:aws:kms:") {
		parts := strings.Split(key, ":")
		if len(parts) >= 6 {
			region := parts[3]
			keyPart := parts[5] // "key/<key-id>"
			if strings.HasPrefix(keyPart, "key/") {
				keyID := strings.TrimPrefix(keyPart, "key/")
				return fmt.Sprintf("awskms://%s?region=%s", keyID, region)
			}
		}
		// Fallback: use just the key ID with region from ARN
		return "awskms://" + key
	}
	// If it's an alias, format it properly with region query param
	if strings.HasPrefix(key, "alias/") {
		// Check if AWS_REGION or AWS_DEFAULT_REGION is set
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
		}
		if region != "" {
			return fmt.Sprintf("awskms://%s?region=%s", key, region)
		}
		return "awskms://" + key
	}
	// If it's just a key ID (UUID format), use it with region
	if len(key) == 36 && strings.Count(key, "-") == 4 {
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = os.Getenv("AWS_DEFAULT_REGION")
		}
		if region != "" {
			return fmt.Sprintf("awskms://%s?region=%s", key, region)
		}
		return "awskms://" + key
	}
	// Default: treat as alias
	return "awskms://alias/" + key
}

// saveKMSKey saves the KMS key to the config file
func saveKMSKey(kmsKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".sloth-kubernetes")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config")

	// Read existing config
	existingContent := ""
	if data, err := os.ReadFile(configFile); err == nil {
		existingContent = string(data)
	}

	// Update or add KMS key, remove passphrase if present
	lines := strings.Split(existingContent, "\n")
	var newLines []string
	foundKMS := false
	for _, line := range lines {
		// Skip existing KMS key and passphrase lines
		if strings.HasPrefix(line, "PULUMI_SECRETS_PROVIDER=") ||
			strings.HasPrefix(line, "PULUMI_CONFIG_PASSPHRASE=") {
			continue
		}
		if line != "" {
			newLines = append(newLines, line)
		}
	}

	// Add the KMS key
	newLines = append(newLines, fmt.Sprintf("PULUMI_SECRETS_PROVIDER=awskms://%s", kmsKey))
	foundKMS = true
	_ = foundKMS

	content := strings.Join(newLines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(configFile, []byte(content), 0600)
}

// createWorkspaceWithSecretsProvider creates a workspace with the specified secrets provider
func createWorkspaceWithSecretsProvider(ctx context.Context, secretsProvider string) (auto.Workspace, error) {
	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	projectName := "sloth-kubernetes"

	// Build project configuration with optional backend
	project := workspace.Project{
		Name:    tokens.PackageName(projectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	}

	// If PULUMI_BACKEND_URL is set, configure backend in the project
	if backendURL := os.Getenv("PULUMI_BACKEND_URL"); backendURL != "" {
		project.Backend = &workspace.ProjectBackend{
			URL: backendURL,
		}
	}

	workspaceOpts := []auto.LocalWorkspaceOption{
		auto.Project(project),
	}

	// Collect all AWS/S3 environment variables to pass to Pulumi subprocess
	envVars := make(map[string]string)
	awsEnvKeys := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"AWS_REGION",
		"AWS_DEFAULT_REGION",
		"AWS_PROFILE",
	}

	for _, key := range awsEnvKeys {
		if val := os.Getenv(key); val != "" {
			envVars[key] = val
		}
	}

	// Add passphrase if using passphrase provider
	if secretsProvider == "passphrase" {
		if passphrase := os.Getenv("PULUMI_CONFIG_PASSPHRASE"); passphrase != "" {
			envVars["PULUMI_CONFIG_PASSPHRASE"] = passphrase
		}
	}

	// Set secrets provider
	if secretsProvider != "" {
		workspaceOpts = append(workspaceOpts, auto.SecretsProvider(secretsProvider))
	}

	// Add environment variables if we have any
	if len(envVars) > 0 {
		workspaceOpts = append(workspaceOpts, auto.EnvVars(envVars))
	}

	return auto.NewLocalWorkspace(ctx, workspaceOpts...)
}

// getPassphrase gets the passphrase from stdin, environment, or interactive prompt
func getPassphrase() (string, error) {
	// Option 1: Read from stdin if --password-stdin flag is set
	if passwordStdin {
		reader := bufio.NewReader(os.Stdin)
		passphrase, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read passphrase from stdin: %w", err)
		}
		return strings.TrimSpace(passphrase), nil
	}

	// Option 2: Check environment variable
	if envPass := os.Getenv("PULUMI_CONFIG_PASSPHRASE"); envPass != "" {
		color.Cyan("Using passphrase from PULUMI_CONFIG_PASSPHRASE environment variable")
		return envPass, nil
	}

	// Option 3: Interactive prompt
	fmt.Println()
	fmt.Print("Enter encryption passphrase: ")
	passBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}
	fmt.Println() // New line after hidden input

	// Confirm passphrase
	fmt.Print("Confirm passphrase: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to read confirmation: %w", err)
	}
	fmt.Println()

	if string(passBytes) != string(confirmBytes) {
		return "", fmt.Errorf("passphrases do not match")
	}

	return string(passBytes), nil
}

// savePassphrase saves the passphrase to the config file
func savePassphrase(passphrase string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".sloth-kubernetes")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "config")

	// Read existing config
	existingConfig := make(map[string]string)
	if data, err := os.ReadFile(configFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existingConfig[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Update passphrase
	existingConfig["PULUMI_CONFIG_PASSPHRASE"] = passphrase

	// Write back
	var lines []string
	lines = append(lines, "# Sloth Kubernetes Configuration")
	lines = append(lines, "# Generated by sloth-kubernetes CLI")
	lines = append(lines, "")
	for key, value := range existingConfig {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return os.WriteFile(configFile, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

func runStateDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate arguments
	if diffFile == "" && len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state diff <stack1> <stack2>\n       or: sloth-kubernetes stacks state diff <stack> --file backup.json")
	}

	stack1Name := args[0]
	printHeader(fmt.Sprintf("🔍 State Diff: %s", stack1Name))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Get stack1 resources
	fullyQualifiedStack1 := fmt.Sprintf("organization/sloth-kubernetes/%s", stack1Name)
	stack1, err := auto.SelectStack(ctx, fullyQualifiedStack1, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stack1Name, err)
	}

	deployment1, err := stack1.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack '%s': %w", stack1Name, err)
	}

	resources1 := parseResourcesFromDeployment(deployment1.Deployment)

	var resources2 []resourceInfo
	var stack2Name string

	if diffFile != "" {
		// Compare with file
		stack2Name = diffFile
		fileData, err := os.ReadFile(diffFile)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", diffFile, err)
		}

		var exportData struct {
			Deployment json.RawMessage `json:"deployment"`
		}
		if err := json.Unmarshal(fileData, &exportData); err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}

		resources2 = parseResourcesFromDeployment(exportData.Deployment)
	} else {
		// Compare with another stack
		stack2Name = args[1]
		fullyQualifiedStack2 := fmt.Sprintf("organization/sloth-kubernetes/%s", stack2Name)
		stack2, err := auto.SelectStack(ctx, fullyQualifiedStack2, workspace)
		if err != nil {
			return fmt.Errorf("failed to select stack '%s': %w", stack2Name, err)
		}

		deployment2, err := stack2.Export(ctx)
		if err != nil {
			return fmt.Errorf("failed to export stack '%s': %w", stack2Name, err)
		}

		resources2 = parseResourcesFromDeployment(deployment2.Deployment)
	}

	// Build maps for comparison
	map1 := make(map[string]resourceInfo)
	map2 := make(map[string]resourceInfo)

	for _, r := range resources1 {
		map1[r.URN] = r
	}
	for _, r := range resources2 {
		map2[r.URN] = r
	}

	// Find differences
	var onlyIn1, onlyIn2, inBoth []string

	for urn := range map1 {
		if _, exists := map2[urn]; exists {
			inBoth = append(inBoth, urn)
		} else {
			onlyIn1 = append(onlyIn1, urn)
		}
	}

	for urn := range map2 {
		if _, exists := map1[urn]; !exists {
			onlyIn2 = append(onlyIn2, urn)
		}
	}

	// Sort for consistent output
	sort.Strings(onlyIn1)
	sort.Strings(onlyIn2)
	sort.Strings(inBoth)

	// Display results
	fmt.Println()
	fmt.Printf("Comparing: %s vs %s\n\n", color.CyanString(stack1Name), color.CyanString(stack2Name))

	color.New(color.Bold).Printf("Summary:\n")
	fmt.Printf("  Resources in %s: %d\n", stack1Name, len(resources1))
	fmt.Printf("  Resources in %s: %d\n", stack2Name, len(resources2))
	fmt.Printf("  Common resources: %d\n", len(inBoth))
	fmt.Println()

	if len(onlyIn1) > 0 {
		color.New(color.Bold, color.FgRed).Printf("Only in %s (%d):\n", stack1Name, len(onlyIn1))
		for _, urn := range onlyIn1 {
			r := map1[urn]
			fmt.Printf("  - %s (%s)\n", color.RedString(r.URN), r.Type)
		}
		fmt.Println()
	}

	if len(onlyIn2) > 0 {
		color.New(color.Bold, color.FgGreen).Printf("Only in %s (%d):\n", stack2Name, len(onlyIn2))
		for _, urn := range onlyIn2 {
			r := map2[urn]
			fmt.Printf("  + %s (%s)\n", color.GreenString(r.URN), r.Type)
		}
		fmt.Println()
	}

	if len(onlyIn1) == 0 && len(onlyIn2) == 0 {
		color.Green("✅ No differences found - stacks have identical resources")
	}

	return nil
}

type resourceInfo struct {
	URN       string
	Type      string
	ID        string
	Protected bool
}

func parseResourcesFromDeployment(deployment json.RawMessage) []resourceInfo {
	var data struct {
		Resources []struct {
			URN     string      `json:"urn"`
			Type    string      `json:"type"`
			ID      interface{} `json:"id"`
			Protect bool        `json:"protect"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(deployment, &data); err != nil {
		return nil
	}

	var resources []resourceInfo
	for _, r := range data.Resources {
		resources = append(resources, resourceInfo{
			URN:       r.URN,
			Type:      r.Type,
			ID:        fmt.Sprintf("%v", r.ID),
			Protected: r.Protect,
		})
	}

	return resources
}

func runStateRepair(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state repair <stack-name>")
	}

	ctx := context.Background()
	stackName := args[0]

	printHeader(fmt.Sprintf("🔧 Repairing Stack State: %s", stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Export current state
	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse deployment
	var data struct {
		Resources []map[string]interface{} `json:"resources"`
	}

	if err := json.Unmarshal(deployment.Deployment, &data); err != nil {
		return fmt.Errorf("failed to parse deployment: %w", err)
	}

	fmt.Println()
	color.Cyan("Analyzing stack state...")
	fmt.Println()

	// Track issues
	var issues []string
	seenURNs := make(map[string]bool)
	var cleanedResources []map[string]interface{}
	duplicatesRemoved := 0
	invalidRemoved := 0

	for _, resource := range data.Resources {
		urn, _ := resource["urn"].(string)

		// Check for empty/invalid URN
		if urn == "" {
			issues = append(issues, "Found resource with empty URN")
			invalidRemoved++
			continue
		}

		// Check for duplicates
		if seenURNs[urn] {
			issues = append(issues, fmt.Sprintf("Duplicate URN: %s", urn))
			duplicatesRemoved++
			continue
		}

		seenURNs[urn] = true
		cleanedResources = append(cleanedResources, resource)
	}

	// Report findings
	color.New(color.Bold).Println("Analysis Results:")
	fmt.Printf("  Total resources: %d\n", len(data.Resources))
	fmt.Printf("  Duplicate URNs: %d\n", duplicatesRemoved)
	fmt.Printf("  Invalid resources: %d\n", invalidRemoved)
	fmt.Println()

	if len(issues) == 0 {
		color.Green("✅ No issues found - stack state is healthy")
		return nil
	}

	color.Yellow("Issues found:")
	for _, issue := range issues {
		fmt.Printf("  • %s\n", issue)
	}
	fmt.Println()

	if stateDryRun {
		color.Cyan("Dry-run mode: No changes made")
		fmt.Printf("\nWould remove %d resources from state\n", duplicatesRemoved+invalidRemoved)
		return nil
	}

	// Confirm repair
	if !autoApprove {
		if !confirm(fmt.Sprintf("Repair stack state by removing %d problematic resources?", duplicatesRemoved+invalidRemoved)) {
			color.Yellow("Repair cancelled")
			return nil
		}
	}

	// Apply changes
	data.Resources = cleanedResources
	modifiedDeployment, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal repaired state: %w", err)
	}

	deployment.Deployment = modifiedDeployment

	color.Cyan("Applying repaired state...")
	if err := stack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import repaired state: %w", err)
	}

	fmt.Println()
	color.Green("✅ Stack state repaired successfully")
	fmt.Printf("  Removed %d duplicate/invalid resources\n", duplicatesRemoved+invalidRemoved)

	return nil
}

func runStateUnprotect(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state unprotect <stack-name> [urn]")
	}

	ctx := context.Background()
	stackName := args[0]

	var targetURN string
	if len(args) > 1 && !unprotectAll {
		targetURN = args[1]
	}

	if targetURN == "" && !unprotectAll {
		return fmt.Errorf("specify a URN or use --all to unprotect all resources")
	}

	printHeader(fmt.Sprintf("🔓 Unprotecting Resources: %s", stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Export current state
	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse deployment
	var data struct {
		Resources []map[string]interface{} `json:"resources"`
	}

	if err := json.Unmarshal(deployment.Deployment, &data); err != nil {
		return fmt.Errorf("failed to parse deployment: %w", err)
	}

	// Find and unprotect resources
	unprotectedCount := 0
	var protectedResources []string

	for i, resource := range data.Resources {
		urn, _ := resource["urn"].(string)
		protected, _ := resource["protect"].(bool)

		if protected {
			protectedResources = append(protectedResources, urn)
		}

		if unprotectAll {
			if protected {
				data.Resources[i]["protect"] = false
				unprotectedCount++
				fmt.Printf("  • %s\n", urn)
			}
		} else if urn == targetURN {
			if !protected {
				color.Yellow("Resource is not protected: %s", targetURN)
				return nil
			}
			data.Resources[i]["protect"] = false
			unprotectedCount++
			fmt.Printf("  • %s\n", urn)
		}
	}

	if unprotectedCount == 0 {
		if targetURN != "" {
			return fmt.Errorf("resource not found: %s", targetURN)
		}
		color.Yellow("\n⚠️  No protected resources found")
		return nil
	}

	fmt.Println()

	// Confirm
	if !autoApprove {
		if !confirm(fmt.Sprintf("Unprotect %d resource(s)?", unprotectedCount)) {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	// Apply changes
	modifiedDeployment, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	deployment.Deployment = modifiedDeployment

	color.Cyan("Applying changes...")
	if err := stack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import state: %w", err)
	}

	fmt.Println()
	color.Green("✅ Successfully unprotected %d resource(s)", unprotectedCount)
	fmt.Println()
	color.Yellow("⚠️  These resources can now be deleted by 'destroy' operations")

	return nil
}

func runStateBulkDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state bulk-delete <stack-name> --type <type>|--pattern <pattern>|--file <file>")
	}

	ctx := context.Background()
	stackName := args[0]

	if resourceType == "" && bulkPattern == "" && bulkFile == "" {
		return fmt.Errorf("must specify --type, --pattern, or --file")
	}

	printHeader(fmt.Sprintf("🗑️  Bulk Delete from State: %s", stackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", stackName)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", stackName, err)
	}

	// Export current state
	deployment, err := stack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export stack: %w", err)
	}

	// Parse deployment
	var data struct {
		Resources []map[string]interface{} `json:"resources"`
	}

	if err := json.Unmarshal(deployment.Deployment, &data); err != nil {
		return fmt.Errorf("failed to parse deployment: %w", err)
	}

	// Build list of URNs to delete
	urnsToDelete := make(map[string]bool)

	if bulkFile != "" {
		// Read URNs from file
		fileData, err := os.ReadFile(bulkFile)
		if err != nil {
			return fmt.Errorf("failed to read file '%s': %w", bulkFile, err)
		}
		lines := strings.Split(string(fileData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				urnsToDelete[line] = true
			}
		}
	}

	// Find matching resources
	var toDelete []string
	var remaining []map[string]interface{}

	for _, resource := range data.Resources {
		urn, _ := resource["urn"].(string)
		resType, _ := resource["type"].(string)

		shouldDelete := false

		// Check by type
		if resourceType != "" && resType == resourceType {
			shouldDelete = true
		}

		// Check by pattern
		if bulkPattern != "" && matchesPattern(urn, bulkPattern) {
			shouldDelete = true
		}

		// Check from file
		if urnsToDelete[urn] {
			shouldDelete = true
		}

		if shouldDelete {
			toDelete = append(toDelete, urn)
		} else {
			remaining = append(remaining, resource)
		}
	}

	if len(toDelete) == 0 {
		color.Yellow("\n⚠️  No matching resources found")
		return nil
	}

	// Show what will be deleted
	fmt.Println()
	color.New(color.Bold).Printf("Resources to delete (%d):\n", len(toDelete))
	for _, urn := range toDelete {
		fmt.Printf("  • %s\n", color.RedString(urn))
	}
	fmt.Println()

	color.Red("⚠️  WARNING: This will remove resources from Pulumi state only!")
	fmt.Println("   Cloud resources will NOT be destroyed and will become unmanaged.")
	fmt.Println()

	// Confirm
	if !forceDelete {
		if !confirm(fmt.Sprintf("Delete %d resource(s) from state?", len(toDelete))) {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	// Apply changes
	data.Resources = remaining
	modifiedDeployment, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	deployment.Deployment = modifiedDeployment

	color.Cyan("Applying changes...")
	if err := stack.Import(ctx, deployment); err != nil {
		return fmt.Errorf("failed to import state: %w", err)
	}

	fmt.Println()
	color.Green("✅ Successfully removed %d resource(s) from state", len(toDelete))

	return nil
}

func matchesPattern(urn, pattern string) bool {
	// Simple wildcard matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.Contains(urn, prefix)
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(urn, suffix)
	}
	return strings.Contains(urn, pattern)
}

func runStateMove(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes stacks state move <source-stack> <target-stack> [urn]")
	}

	ctx := context.Background()
	sourceStackName := args[0]
	targetStackName := args[1]

	var targetURN string
	if len(args) > 2 {
		targetURN = args[2]
	}

	if targetURN == "" && moveType == "" {
		return fmt.Errorf("specify a URN or use --type to move resources by type")
	}

	printHeader(fmt.Sprintf("📦 Moving Resources: %s → %s", sourceStackName, targetStackName))

	// Create workspace with S3 support
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Get source stack
	fullyQualifiedSource := fmt.Sprintf("organization/sloth-kubernetes/%s", sourceStackName)
	sourceStack, err := auto.SelectStack(ctx, fullyQualifiedSource, workspace)
	if err != nil {
		return fmt.Errorf("failed to select source stack '%s': %w", sourceStackName, err)
	}

	// Get target stack
	fullyQualifiedTarget := fmt.Sprintf("organization/sloth-kubernetes/%s", targetStackName)
	targetStack, err := auto.SelectStack(ctx, fullyQualifiedTarget, workspace)
	if err != nil {
		return fmt.Errorf("failed to select target stack '%s': %w", targetStackName, err)
	}

	// Export both stacks
	sourceDeployment, err := sourceStack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export source stack: %w", err)
	}

	targetDeployment, err := targetStack.Export(ctx)
	if err != nil {
		return fmt.Errorf("failed to export target stack: %w", err)
	}

	// Parse deployments
	var sourceData struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	var targetData struct {
		Resources []map[string]interface{} `json:"resources"`
	}

	if err := json.Unmarshal(sourceDeployment.Deployment, &sourceData); err != nil {
		return fmt.Errorf("failed to parse source deployment: %w", err)
	}
	if err := json.Unmarshal(targetDeployment.Deployment, &targetData); err != nil {
		return fmt.Errorf("failed to parse target deployment: %w", err)
	}

	// Find resources to move
	var toMove []map[string]interface{}
	var remaining []map[string]interface{}

	for _, resource := range sourceData.Resources {
		urn, _ := resource["urn"].(string)
		resType, _ := resource["type"].(string)

		shouldMove := false

		if targetURN != "" && urn == targetURN {
			shouldMove = true
		}
		if moveType != "" && resType == moveType {
			shouldMove = true
		}

		if shouldMove {
			toMove = append(toMove, resource)
		} else {
			remaining = append(remaining, resource)
		}
	}

	if len(toMove) == 0 {
		color.Yellow("\n⚠️  No matching resources found in source stack")
		return nil
	}

	// Show what will be moved
	fmt.Println()
	color.New(color.Bold).Printf("Resources to move (%d):\n", len(toMove))
	for _, resource := range toMove {
		urn, _ := resource["urn"].(string)
		resType, _ := resource["type"].(string)
		fmt.Printf("  • %s (%s)\n", color.CyanString(urn), resType)
	}
	fmt.Println()

	// Confirm
	if !autoApprove {
		if !confirm(fmt.Sprintf("Move %d resource(s) from %s to %s?", len(toMove), sourceStackName, targetStackName)) {
			color.Yellow("Operation cancelled")
			return nil
		}
	}

	// Update URNs to reflect new stack
	for i, resource := range toMove {
		urn, _ := resource["urn"].(string)
		// Replace stack name in URN
		// URN format: urn:pulumi:stack::project::type::name
		parts := strings.Split(urn, "::")
		if len(parts) >= 2 {
			parts[0] = strings.Replace(parts[0], sourceStackName, targetStackName, 1)
			toMove[i]["urn"] = strings.Join(parts, "::")
		}
	}

	// Add to target
	targetData.Resources = append(targetData.Resources, toMove...)

	// Remove from source
	sourceData.Resources = remaining

	// Marshal and import both
	sourceModified, err := json.Marshal(sourceData)
	if err != nil {
		return fmt.Errorf("failed to marshal source state: %w", err)
	}

	targetModified, err := json.Marshal(targetData)
	if err != nil {
		return fmt.Errorf("failed to marshal target state: %w", err)
	}

	sourceDeployment.Deployment = sourceModified
	targetDeployment.Deployment = targetModified

	color.Cyan("Updating target stack...")
	if err := targetStack.Import(ctx, targetDeployment); err != nil {
		return fmt.Errorf("failed to import to target stack: %w", err)
	}

	color.Cyan("Updating source stack...")
	if err := sourceStack.Import(ctx, sourceDeployment); err != nil {
		return fmt.Errorf("failed to update source stack: %w", err)
	}

	fmt.Println()
	color.Green("✅ Successfully moved %d resource(s)", len(toMove))
	fmt.Printf("  From: %s\n", sourceStackName)
	fmt.Printf("  To:   %s\n", targetStackName)
	fmt.Println()
	color.Cyan("Next steps:")
	fmt.Printf("  • Verify target: sloth-kubernetes stacks state list %s\n", targetStackName)
	fmt.Printf("  • Verify source: sloth-kubernetes stacks state list %s\n", sourceStackName)

	return nil
}
