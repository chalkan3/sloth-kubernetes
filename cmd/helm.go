package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var helmCmd = &cobra.Command{
	Use:   "helm",
	Short: "Execute Helm commands (requires helm binary in PATH)",
	Long: `Execute Helm commands by calling the helm binary.

This command requires the 'helm' binary to be installed and available in your PATH.
You can install Helm from: https://helm.sh/docs/intro/install/

The command automatically uses kubeconfig from:
  1. --kubeconfig flag
  2. KUBECONFIG environment variable
  3. ~/.kube/config (default)

All standard Helm v3 commands and flags are supported.`,
	Example: `  # List all releases
  sloth-kubernetes helm list

  # Install a chart
  sloth-kubernetes helm install myapp bitnami/nginx

  # Upgrade a release
  sloth-kubernetes helm upgrade myapp bitnami/nginx

  # Add a repository
  sloth-kubernetes helm repo add bitnami https://charts.bitnami.com/bitnami

  # Search for charts
  sloth-kubernetes helm search repo nginx

  # Get release status
  sloth-kubernetes helm status myapp

  # Uninstall a release
  sloth-kubernetes helm uninstall myapp

  # Use custom kubeconfig
  sloth-kubernetes helm --kubeconfig=./my-kubeconfig list`,
	DisableFlagParsing: true,
	RunE:               runHelm,
}

func init() {
	rootCmd.AddCommand(helmCmd)
}

func runHelm(cmd *cobra.Command, args []string) error {
	// Check if helm is available in PATH
	helmBinary, err := exec.LookPath("helm")
	if err != nil {
		return fmt.Errorf("helm binary not found in PATH. Please install Helm from https://helm.sh/docs/intro/install/")
	}

	// Try to set KUBECONFIG environment if needed
	kubeconfigPath := getKubeconfigPath()
	if kubeconfigPath != "" {
		os.Setenv("KUBECONFIG", kubeconfigPath)
	}

	// Create and execute helm command
	helmExec := exec.Command(helmBinary, args...)
	helmExec.Stdin = os.Stdin
	helmExec.Stdout = os.Stdout
	helmExec.Stderr = os.Stderr
	helmExec.Env = os.Environ()

	return helmExec.Run()
}
