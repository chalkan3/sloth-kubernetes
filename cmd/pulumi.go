package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/spf13/cobra"

	"github.com/chalkan3/sloth-kubernetes/internal/common"
)

var (
	pulumiStackName string
)

var pulumiCmd = &cobra.Command{
	Use:   "pulumi [command]",
	Short: "Execute Pulumi operations using embedded Automation API",
	Long: `Execute Pulumi operations using the embedded Automation API (no CLI required).

This command provides direct access to common Pulumi operations without requiring
the Pulumi CLI to be installed. All operations use the Pulumi Automation API
that is already embedded in sloth-kubernetes.

Available operations:
  • stack list           - List all stacks
  • stack output         - Show stack outputs
  • stack export         - Export stack state to JSON
  • stack import         - Import stack state from JSON
  • stack info           - Show detailed stack information
  • stack delete         - Delete a stack
  • stack select         - Select current stack
  • stack current        - Show current selected stack
  • stack rename         - Rename a stack
  • stack cancel         - Cancel and unlock a stack
  • stack state          - Manage stack state
  • preview              - Preview infrastructure changes
  • refresh              - Refresh stack state from cloud

No Pulumi CLI installation required! 🦥`,
	Example: `  # Show stack outputs
  sloth-kubernetes pulumi stack output

  # List all stacks
  sloth-kubernetes pulumi stack list

  # Export stack state
  sloth-kubernetes pulumi stack export --stack production > backup.json

  # Import stack state
  sloth-kubernetes pulumi stack import --stack production < backup.json

  # Refresh stack state
  sloth-kubernetes pulumi refresh --stack production`,
	RunE: runPulumiCommand,
}

var previewPulumiCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview infrastructure changes",
	Long:  `Preview what changes would be made to infrastructure (requires deployment context)`,
	Example: `  # Preview changes
  sloth-kubernetes pulumi preview --stack production`,
	RunE: runPreview,
}

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Stack operations",
	Long:  `Perform operations on Pulumi stacks`,
}

func init() {
	rootCmd.AddCommand(pulumiCmd)

	// Add stack subcommand with all stack operations from stacks.go
	pulumiCmd.AddCommand(stackCmd)

	// Add all stack operations from stacks.go under 'pulumi stack'
	stackCmd.AddCommand(listStacksCmd)   // list
	stackCmd.AddCommand(stackInfoCmd)    // info
	stackCmd.AddCommand(deleteStackCmd)  // delete
	stackCmd.AddCommand(outputCmd)       // output
	stackCmd.AddCommand(selectStackCmd)  // select
	stackCmd.AddCommand(currentStackCmd) // current
	stackCmd.AddCommand(exportStackCmd)  // export
	stackCmd.AddCommand(importStackCmd)  // import
	stackCmd.AddCommand(renameStackCmd)  // rename
	stackCmd.AddCommand(cancelCmd)       // cancel
	stackCmd.AddCommand(stateCmd)        // state

	// State subcommands
	stateCmd.AddCommand(stateDeleteCmd)
	stateCmd.AddCommand(stateListCmd)

	// Add top-level Pulumi operations
	pulumiCmd.AddCommand(previewPulumiCmd)
	pulumiCmd.AddCommand(refreshCmd)  // Use the comprehensive refresh from refresh.go

	// Flags for Pulumi operations (preview only, refresh uses its own flags from refresh.go)
	previewPulumiCmd.Flags().StringVar(&pulumiStackName, "stack", "", "Stack name")
}

func runPulumiCommand(cmd *cobra.Command, args []string) error {
	// If no args provided, show help
	if len(args) == 0 {
		fmt.Println()
		color.Cyan("🦥 Pulumi Automation API")
		fmt.Println()
		color.Green("✅ No Pulumi CLI required - using embedded Automation API!")
		fmt.Println()
		fmt.Println("Usage: sloth-kubernetes pulumi [command]")
		fmt.Println()
		color.Cyan("Available commands:")
		fmt.Println("  stack list                List all stacks")
		fmt.Println("  stack output              Show stack outputs")
		fmt.Println("  stack export              Export stack state to JSON")
		fmt.Println("  stack import              Import stack state from JSON")
		fmt.Println("  stack info                Show detailed stack information")
		fmt.Println("  stack delete              Delete a stack")
		fmt.Println("  stack select              Select current stack")
		fmt.Println("  stack current             Show current selected stack")
		fmt.Println("  stack rename              Rename a stack")
		fmt.Println("  stack cancel              Cancel and unlock a stack")
		fmt.Println("  stack state               Manage stack state")
		fmt.Println("  preview                   Preview infrastructure changes")
		fmt.Println("  refresh                   Refresh stack state from cloud")
		fmt.Println()
		color.Cyan("Examples:")
		fmt.Println("  sloth-kubernetes pulumi stack list")
		fmt.Println("  sloth-kubernetes pulumi stack output --stack production")
		fmt.Println("  sloth-kubernetes pulumi stack export --stack production > backup.json")
		fmt.Println("  sloth-kubernetes pulumi refresh --stack production")
		fmt.Println()
		color.Yellow("💡 All operations use the embedded Pulumi Automation API")
		fmt.Println()
		return nil
	}

	return cmd.Help()
}

func runPreview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load saved S3 backend configuration
	_ = common.LoadSavedConfig()

	// Determine stack name
	targetStack := pulumiStackName
	if targetStack == "" && len(args) > 0 {
		targetStack = args[0]
	}
	if targetStack == "" {
		targetStack = stackName
	}
	if targetStack == "" {
		return fmt.Errorf("stack name required. Use: --stack <name>")
	}

	// Create workspace
	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Select stack
	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", targetStack)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		return fmt.Errorf("failed to select stack '%s': %w", targetStack, err)
	}

	fmt.Println()
	color.Cyan("🔍 Previewing changes for stack: %s", targetStack)
	fmt.Println()

	// Preview
	previewOpts := []optpreview.Option{
		optpreview.ProgressStreams(os.Stdout),
	}

	_, err = stack.Preview(ctx, previewOpts...)
	if err != nil {
		return fmt.Errorf("preview failed: %w", err)
	}

	return nil
}
