package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"
)

var (
	outputFile string
	merge      bool
)

var kubeconfigCmd = &cobra.Command{
	Use:   "kubeconfig",
	Short: "Get kubeconfig for kubectl access",
	Long: `Retrieve the kubeconfig file for accessing the Kubernetes cluster.
The kubeconfig can be printed to stdout or saved to a file.`,
	Example: `  # Print to stdout
  kubernetes-create kubeconfig

  # Save to file
  kubernetes-create kubeconfig -o ~/.kube/config

  # Save to default location
  kubernetes-create kubeconfig -o ~/.kube/config`,
	RunE: runKubeconfig,
}

func init() {
	rootCmd.AddCommand(kubeconfigCmd)
	kubeconfigCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	kubeconfigCmd.Flags().BoolVar(&merge, "merge", false, "Merge with existing kubeconfig (not implemented yet)")
}

func runKubeconfig(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Retrieving kubeconfig..."
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

	// Get outputs
	outputs, err := stack.Outputs(ctx)
	if err != nil {
		s.Stop()
		return fmt.Errorf("failed to get outputs: %w", err)
	}

	kubeConfigOutput, ok := outputs["kubeConfig"]
	if !ok {
		s.Stop()
		return fmt.Errorf("kubeconfig not found in stack outputs")
	}

	s.Stop()

	kubeConfigStr := fmt.Sprintf("%v", kubeConfigOutput.Value)

	// Output to file or stdout
	if outputFile != "" {
		// Expand home directory
		if outputFile[:2] == "~/" {
			home, _ := os.UserHomeDir()
			outputFile = filepath.Join(home, outputFile[2:])
		}

		// Create directory if needed
		dir := filepath.Dir(outputFile)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Write file
		if err := os.WriteFile(outputFile, []byte(kubeConfigStr), 0600); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}

		printSuccess(fmt.Sprintf("Kubeconfig saved to %s", outputFile))
		fmt.Println()
		color.Green("🎯 You can now use kubectl:")
		fmt.Printf("   export KUBECONFIG=%s\n", outputFile)
		fmt.Println("   kubectl get nodes")
	} else {
		// Print to stdout
		fmt.Println(kubeConfigStr)
	}

	return nil
}
