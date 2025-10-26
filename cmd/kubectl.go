package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
)

var kubectlCmd = &cobra.Command{
	Use:   "kubectl",
	Short: "Execute kubectl commands (embedded kubectl client)",
	Long: `Execute kubectl commands directly using the embedded Kubernetes client.

This command embeds the official kubectl client, providing full kubectl
functionality without requiring a separate kubectl installation.

The command automatically uses kubeconfig from:
  1. --kubeconfig flag
  2. KUBECONFIG environment variable
  3. ~/.kube/config (default)
  4. Kubeconfig from Pulumi stack (sloth-kubernetes kubeconfig get)

All standard kubectl commands and flags are supported.`,
	Example: `  # Get all nodes
  sloth-kubernetes kubectl get nodes

  # Get pods in all namespaces
  sloth-kubernetes kubectl get pods -A

  # Get pods in specific namespace
  sloth-kubernetes kubectl get pods -n kube-system

  # Describe a resource
  sloth-kubernetes kubectl describe pod nginx-123 -n default

  # Apply a manifest
  sloth-kubernetes kubectl apply -f deployment.yaml

  # Get logs
  sloth-kubernetes kubectl logs nginx-123 -n default

  # Execute command in pod
  sloth-kubernetes kubectl exec -it nginx-123 -- sh

  # Use custom kubeconfig
  sloth-kubernetes kubectl --kubeconfig=./my-kubeconfig get nodes`,
	DisableFlagParsing: true,
	RunE:               runKubectl,
}

func init() {
	rootCmd.AddCommand(kubectlCmd)
}

func runKubectl(cmd *cobra.Command, args []string) error {
	// Try to set KUBECONFIG environment if needed
	kubeconfigPath := getKubeconfigPath()
	if kubeconfigPath != "" {
		os.Setenv("KUBECONFIG", kubeconfigPath)
	}

	// Create the root kubectl command with all subcommands
	kubectlRootCmd := kubectlcmd.NewDefaultKubectlCommand()
	kubectlRootCmd.SetArgs(args)
	kubectlRootCmd.SetIn(os.Stdin)
	kubectlRootCmd.SetOut(os.Stdout)
	kubectlRootCmd.SetErr(os.Stderr)

	// Execute kubectl command
	return kubectlRootCmd.Execute()
}

// getKubeconfigPath returns the kubeconfig path in order of precedence
func getKubeconfigPath() string {
	// 1. Check KUBECONFIG environment variable
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// 2. Check default location
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	defaultKubeconfig := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(defaultKubeconfig); err == nil {
		return defaultKubeconfig
	}

	// 3. Try to get from stack (if available)
	stackKubeconfig := getKubeconfigFromStack()
	if stackKubeconfig != "" {
		return stackKubeconfig
	}

	return ""
}

// getKubeconfigFromStack attempts to retrieve kubeconfig from Pulumi stack
func getKubeconfigFromStack() string {
	// This will be implemented to fetch kubeconfig from stack outputs
	// For now, return empty - users can use 'sloth-kubernetes kubeconfig get' first
	return ""
}
