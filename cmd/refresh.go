package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/spf13/cobra"
)

var (
	expectNoChanges bool
	showSecrets     bool
	skipPreview     bool
)

var refreshCmd = &cobra.Command{
	Use:   "refresh <stack-name>",
	Short: "Refresh the stack state to match real infrastructure",
	Long: `Refresh synchronizes your Pulumi stack state with the actual state
of your infrastructure in the cloud providers.

This command compares the current state file with the real resources
in DigitalOcean and Linode, updating the state to reflect reality.

Use this when:
  • Resources were modified outside of Pulumi (manual changes)
  • You suspect drift between state and reality
  • After recovering from a failed deployment
  • Before running 'pulumi up' to ensure state accuracy`,
	Example: `  # Refresh a specific stack
  sloth-kubernetes refresh production

  # Refresh using --stack flag
  sloth-kubernetes refresh --stack production

  # Refresh and show secrets
  sloth-kubernetes refresh production --show-secrets

  # Refresh and expect no changes (exits with error if changes found)
  sloth-kubernetes refresh production --expect-no-changes

  # Skip preview and refresh directly
  sloth-kubernetes refresh production --skip-preview --yes`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRefresh,
}

func init() {
	rootCmd.AddCommand(refreshCmd)
	refreshCmd.Flags().BoolVar(&expectNoChanges, "expect-no-changes", false, "Return error if any changes are detected")
	refreshCmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "Show secret values in output")
	refreshCmd.Flags().BoolVar(&skipPreview, "skip-preview", false, "Skip preview and refresh directly")
}

func runRefresh(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine target stack
	targetStack := stackName
	if len(args) > 0 {
		targetStack = args[0]
	}

	// Validate stack name
	if targetStack == "" {
		return fmt.Errorf("stack name is required. Use: sloth-kubernetes refresh <stack-name> or --stack <name>")
	}

	// Print header
	fmt.Println()
	printHeader("🔄 Refreshing Stack State")
	fmt.Println()
	color.Cyan("Stack: %s", targetStack)
	fmt.Println()
	color.Yellow("This will synchronize your Pulumi state with actual cloud resources.")
	color.Yellow("No resources will be created, modified, or deleted.")
	fmt.Println()

	// Confirm refresh unless auto-approved
	if !autoApprove && !skipPreview {
		if !confirm("Do you want to proceed with the refresh?") {
			color.Yellow("Refresh cancelled")
			return nil
		}
	}

	// Get stack with S3 backend support
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Connecting to Pulumi stack..."
	s.Start()

	workspace, err := createWorkspaceWithS3Support(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fullyQualifiedStackName := fmt.Sprintf("organization/sloth-kubernetes/%s", targetStack)
	stack, err := auto.SelectStack(ctx, fullyQualifiedStackName, workspace)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to select stack: %w", err)
	}
	s.Stop()
	printSuccess("Connected to stack")

	// Get current outputs before refresh
	fmt.Println()
	printHeader("📊 Current State Summary")
	fmt.Println()

	outputs, err := stack.Outputs(ctx)
	if err != nil {
		color.Yellow("⚠️  Could not retrieve current outputs: %v", err)
	} else {
		printOutputsSummary(outputs)
	}

	// Refresh
	fmt.Println()
	printHeader("🔄 Refreshing state from cloud providers...")
	fmt.Println()

	// Build refresh options
	refreshOpts := []optrefresh.Option{
		optrefresh.ProgressStreams(os.Stdout),
	}

	if expectNoChanges {
		refreshOpts = append(refreshOpts, optrefresh.ExpectNoChanges())
	}

	if showSecrets {
		refreshOpts = append(refreshOpts, optrefresh.ShowSecrets(true))
	}

	// Perform refresh
	refreshResult, err := stack.Refresh(ctx, refreshOpts...)
	if err != nil {
		return fmt.Errorf("failed to refresh: %w", err)
	}

	// Check summary
	fmt.Println()
	printHeader("📋 Refresh Summary")
	fmt.Println()

	if refreshResult.Summary.ResourceChanges != nil {
		changes := *refreshResult.Summary.ResourceChanges

		hasChanges := false
		if changes["update"] > 0 || changes["delete"] > 0 || changes["create"] > 0 {
			hasChanges = true
		}

		if hasChanges {
			color.Yellow("⚠️  State changes detected:")
			fmt.Println()
			if changes["update"] > 0 {
				color.Yellow("  • %d resource(s) updated", changes["update"])
			}
			if changes["create"] > 0 {
				color.Yellow("  • %d resource(s) created", changes["create"])
			}
			if changes["delete"] > 0 {
				color.Red("  • %d resource(s) deleted", changes["delete"])
			}
			if changes["same"] > 0 {
				color.Green("  • %d resource(s) unchanged", changes["same"])
			}
			fmt.Println()

			if expectNoChanges {
				return fmt.Errorf("refresh detected changes when expecting none")
			}

			// Show drift warning
			color.Yellow("💡 Drift detected between Pulumi state and actual infrastructure.")
			color.Yellow("   Consider running 'sloth-kubernetes deploy %s' to reconcile.", targetStack)
		} else {
			color.Green("✅ No changes detected - state matches infrastructure!")
		}
	}

	// Print updated outputs
	fmt.Println()
	printHeader("📊 Updated State Summary")
	fmt.Println()

	updatedOutputs, err := stack.Outputs(ctx)
	if err != nil {
		color.Yellow("⚠️  Could not retrieve updated outputs: %v", err)
	} else {
		printOutputsSummary(updatedOutputs)
	}

	// Success
	fmt.Println()
	color.Green("✅ Stack state refreshed successfully")
	fmt.Println()

	return nil
}

// printOutputsSummary prints a summary of stack outputs
func printOutputsSummary(outputs auto.OutputMap) {
	if len(outputs) == 0 {
		color.Yellow("No outputs found in stack")
		return
	}

	// Count resources by type
	nodeCount := 0
	networkCount := 0

	for key := range outputs {
		switch {
		case containsString(key, "node_id") || containsString(key, "droplet") || containsString(key, "linode"):
			nodeCount++
		case containsString(key, "network") || containsString(key, "vpc"):
			networkCount++
		}
	}

	fmt.Printf("  • Total outputs: %d\n", len(outputs))
	if nodeCount > 0 {
		fmt.Printf("  • Node-related: %d\n", nodeCount)
	}
	if networkCount > 0 {
		fmt.Printf("  • Network-related: %d\n", networkCount)
	}

	// Show key outputs
	keyOutputs := []string{
		"cluster_name",
		"kubernetes_version",
		"master_count",
		"worker_count",
		"wireguard_enabled",
	}

	fmt.Println()
	color.Cyan("Key Outputs:")
	for _, key := range keyOutputs {
		if output, exists := outputs[key]; exists {
			fmt.Printf("  • %s: %v\n", key, output.Value)
		}
	}
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOf(s, substr) >= 0)))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
