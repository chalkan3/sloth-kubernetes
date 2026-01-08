package cmd

import (
	"fmt"
	"os"

	"github.com/chalkan3/sloth-kubernetes/pkg/vpn/tailscale"
	"github.com/spf13/cobra"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
)

var kubectlCmd = &cobra.Command{
	Use:   "kubectl [stack-name] [kubectl-args...]",
	Short: "Execute kubectl commands using kubeconfig from stack",
	Long: `Execute kubectl commands directly using the embedded Kubernetes client.

This command embeds the official kubectl client, providing full kubectl
functionality without requiring a separate kubectl installation.

The kubeconfig is automatically retrieved from the specified Pulumi stack,
so you don't need to manage kubeconfig files manually.

All standard kubectl commands and flags are supported after the stack name.`,
	Example: `  # Get all nodes
  sloth-kubernetes kubectl my-cluster get nodes

  # Get pods in all namespaces
  sloth-kubernetes kubectl my-cluster get pods -A

  # Get pods in specific namespace
  sloth-kubernetes kubectl my-cluster get pods -n kube-system

  # Describe a resource
  sloth-kubernetes kubectl my-cluster describe pod nginx-123 -n default

  # Apply a manifest
  sloth-kubernetes kubectl my-cluster apply -f deployment.yaml

  # Get logs
  sloth-kubernetes kubectl my-cluster logs nginx-123 -n default

  # Execute command in pod
  sloth-kubernetes kubectl my-cluster exec -it nginx-123 -- sh

  # Using --stack flag instead of positional argument
  sloth-kubernetes kubectl --stack my-cluster get nodes`,
	DisableFlagParsing: true,
	RunE:               runKubectl,
}

func init() {
	rootCmd.AddCommand(kubectlCmd)
}

func runKubectl(cmd *cobra.Command, args []string) error {
	// Parse stack name from args
	// First arg should be stack name, rest are kubectl args
	if len(args) == 0 {
		return fmt.Errorf("stack name is required\nUsage: sloth-kubernetes kubectl <stack-name> [kubectl-args...]")
	}

	// Check if first arg is a flag (starts with -)
	targetStack := ""
	kubectlArgs := args

	if args[0][0] != '-' {
		targetStack = args[0]
		kubectlArgs = args[1:]
	} else {
		// Try to get from global flag
		targetStack = stackName
	}

	if targetStack == "" {
		return fmt.Errorf("stack name is required\nUsage: sloth-kubernetes kubectl <stack-name> [kubectl-args...]")
	}

	// Validate stack exists
	if err := EnsureStackExists(targetStack); err != nil {
		return err
	}

	// Get kubeconfig from stack
	kubeconfigPath, err := GetKubeconfigFromStack(targetStack)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig from stack '%s': %w", targetStack, err)
	}

	// Set KUBECONFIG environment
	os.Setenv("KUBECONFIG", kubeconfigPath)

	// Check if VPN daemon is running and configure SOCKS proxy
	if tailscale.IsDaemonRunning(targetStack) {
		proxyPort := tailscale.GetSavedProxyPort(targetStack)
		if proxyPort > 0 {
			proxyURL := fmt.Sprintf("socks5://127.0.0.1:%d", proxyPort)
			os.Setenv("HTTPS_PROXY", proxyURL)
			os.Setenv("https_proxy", proxyURL)
		}
	}

	// Create the root kubectl command with all subcommands
	kubectlRootCmd := kubectlcmd.NewDefaultKubectlCommand()
	kubectlRootCmd.SetArgs(kubectlArgs)
	kubectlRootCmd.SetIn(os.Stdin)
	kubectlRootCmd.SetOut(os.Stdout)
	kubectlRootCmd.SetErr(os.Stderr)

	// Execute kubectl command
	return kubectlRootCmd.Execute()
}
