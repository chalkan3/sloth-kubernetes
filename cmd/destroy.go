package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/spf13/cobra"
)

var force bool

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy an existing Kubernetes cluster",
	Long: `Destroy an existing Kubernetes cluster and all associated resources.
This will delete all VMs, DNS records, and configurations.

WARNING: This action cannot be undone!`,
	Example: `  # Destroy with confirmation
  kubernetes-create destroy

  # Force destroy without confirmation
  kubernetes-create destroy --yes --force`,
	RunE: runDestroy,
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&force, "force", false, "Force destroy even if there are dependencies")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Print warning header
	fmt.Println()
	color.Red("⚠️  WARNING: Cluster Destruction")
	fmt.Println()
	color.Yellow("This will destroy the entire cluster and all resources:")
	fmt.Println("  • All virtual machines (6 nodes)")
	fmt.Println("  • All data and configurations")
	fmt.Println("  • DNS records")
	fmt.Println("  • SSH keys")
	fmt.Println()
	color.Red("This action CANNOT be undone!")
	fmt.Println()

	// Confirm destruction
	if !autoApprove {
		if !confirm("Are you ABSOLUTELY SURE you want to destroy the cluster?") {
			color.Yellow("Destruction cancelled")
			return nil
		}

		// Double confirmation
		fmt.Println()
		color.Red("⚠️  FINAL CONFIRMATION")
		if !confirm("Type 'yes' to confirm destruction") {
			color.Yellow("Destruction cancelled")
			return nil
		}
	}

	// Get stack
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Connecting to Pulumi stack..."
	s.Start()

	stack, err := auto.SelectStackInlineSource(ctx, stackName, "kubernetes-create", func(ctx *pulumi.Context) error {
		return nil
	})
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to select stack: %w", err)
	}
	s.Stop()
	printSuccess("Connected to stack")

	// Destroy
	fmt.Println()
	printHeader("🔥 Destroying cluster...")
	fmt.Println()

	_, err = stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
	if err != nil {
		return fmt.Errorf("failed to destroy: %w", err)
	}

	// Success
	fmt.Println()
	color.Green("✅ Cluster destroyed successfully")
	fmt.Println()
	color.Yellow("💡 The Pulumi stack still exists. To remove it completely:")
	fmt.Println("   kubernetes-create stack remove --stack " + stackName)
	fmt.Println()

	return nil
}
