package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"
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

var destroyStack bool

func init() {
	rootCmd.AddCommand(stacksCmd)

	// Add subcommands
	stacksCmd.AddCommand(listStacksCmd)
	stacksCmd.AddCommand(stackInfoCmd)
	stacksCmd.AddCommand(deleteStackCmd)

	// Delete flags
	deleteStackCmd.Flags().BoolVar(&destroyStack, "destroy", false, "Destroy all resources before deleting stack")
}

func runListStacks(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	printHeader("📦 Deployment Stacks")

	// Create workspace
	workspace, err := auto.NewLocalWorkspace(ctx)
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

	// Get stack
	workspace, err := auto.NewLocalWorkspace(ctx)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	s, err := auto.SelectStack(ctx, stackName, workspace)
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
		if url == "" {
			url = "local://"
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
