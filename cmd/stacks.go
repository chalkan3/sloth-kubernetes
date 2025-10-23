package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/spf13/cobra"

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

var (
	destroyStack bool
	outputKey    string
	outputJSON   bool
	exportOutput string
	forceDelete  bool
	resourceType string
)

func init() {
	rootCmd.AddCommand(stacksCmd)

	// Add subcommands
	stacksCmd.AddCommand(listStacksCmd)
	stacksCmd.AddCommand(stackInfoCmd)
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

	// Delete flags
	deleteStackCmd.Flags().BoolVar(&destroyStack, "destroy", false, "Destroy all resources before deleting stack")

	// Output flags
	outputCmd.Flags().StringVar(&outputKey, "key", "", "Show specific output key")
	outputCmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	// Export flags
	exportStackCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (default: <stack-name>-state.json)")

	// State delete flags
	stateDeleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete without confirmation")

	// State list flags
	stateListCmd.Flags().StringVar(&resourceType, "type", "", "Filter by resource type (e.g., digitalocean:Droplet)")
}

// createWorkspaceWithS3Support creates a Pulumi workspace with S3/MinIO backend support
func createWorkspaceWithS3Support(ctx context.Context) (auto.Workspace, error) {
	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	projectName := "sloth-kubernetes"
	workspaceOpts := []auto.LocalWorkspaceOption{
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		}),
	}

	// Collect all AWS/S3 environment variables to pass to Pulumi subprocess
	envVars := make(map[string]string)
	awsEnvKeys := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
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
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// List stacks
	stacks, err := workspace.ListStacks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stacks: %w", err)
	}

	if len(stacks) == 0 {
		color.Yellow("\n⚠️  No stacks found")
		fmt.Println()
		color.Cyan("Create a new stack with:")
		fmt.Println("  sloth-kubernetes deploy <stack-name> --config cluster.yaml")
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
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes stacks delete <stack-name>")
	}

	stackName := args[0]

	printHeader(fmt.Sprintf("🗑️  Deleting Stack: %s", stackName))

	if destroyStack {
		color.Red("⚠️  This will DESTROY all resources in the stack!")
		color.Yellow("⚠️  Stack destruction will be implemented in next phase")
	} else {
		color.Yellow("Stack will be deleted but resources will remain (use --destroy to remove resources)")
	}

	fmt.Println()
	color.Cyan(fmt.Sprintf("Stack: %s", stackName))

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

	stackName := args[0]

	printHeader(fmt.Sprintf("💾 Exporting Stack: %s", stackName))

	fmt.Println()
	color.Yellow("⚠️  Stack export/import functionality requires Pulumi CLI")
	fmt.Println()
	color.Cyan("Alternative: Use 'pulumi stack export' command:")
	fmt.Printf("  pulumi stack export --stack %s > %s-state.json\n", stackName, stackName)
	fmt.Println()
	color.Cyan("To import later:")
	fmt.Printf("  pulumi stack import --stack %s < %s-state.json\n", stackName, stackName)

	return nil
}

func runImportStack(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sloth-kubernetes stacks import <stack-name> <file>")
	}

	stackName := args[0]
	filePath := args[1]

	printHeader(fmt.Sprintf("📥 Importing Stack: %s", stackName))

	fmt.Println()
	color.Yellow("⚠️  Stack export/import functionality requires Pulumi CLI")
	fmt.Println()
	color.Cyan("Use 'pulumi stack import' command:")
	fmt.Printf("  pulumi stack import --stack %s < %s\n", stackName, filePath)
	fmt.Println()
	color.Cyan("Or create the stack and import:")
	fmt.Printf("  pulumi stack init %s\n", stackName)
	fmt.Printf("  pulumi stack import < %s\n", filePath)

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
	fmt.Println("  3. Or import it back with: pulumi import <type> <name> <id>")

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
