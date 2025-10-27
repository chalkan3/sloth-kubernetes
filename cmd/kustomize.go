package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var kustomizeCmd = &cobra.Command{
	Use:   "kustomize",
	Short: "Execute Kustomize commands (requires kustomize binary in PATH)",
	Long: `Execute Kustomize commands by calling the kustomize binary.

This command requires the 'kustomize' binary to be installed and available in your PATH.
You can install kustomize from: https://kubectl.docs.kubernetes.io/installation/kustomize/

Kustomize lets you customize raw, template-free YAML files for multiple
purposes, leaving the original YAML untouched and usable as is.`,
	Example: `  # Build kustomization
  sloth-kubernetes kustomize build ./overlays/production

  # Build and apply to cluster
  sloth-kubernetes kustomize build ./overlays/production | sloth-kubernetes kubectl apply -f -

  # Create kustomization.yaml
  sloth-kubernetes kustomize create --autodetect

  # Edit kustomization
  sloth-kubernetes kustomize edit set image nginx=nginx:1.21

  # Add a resource
  sloth-kubernetes kustomize edit add resource deployment.yaml

  # Add a ConfigMap generator
  sloth-kubernetes kustomize edit add configmap my-config --from-literal=key=value`,
	DisableFlagParsing: true,
	RunE:               runKustomize,
}

func init() {
	rootCmd.AddCommand(kustomizeCmd)
}

func runKustomize(cmd *cobra.Command, args []string) error {
	// Check if kustomize is available in PATH
	kustomizeBinary, err := exec.LookPath("kustomize")
	if err != nil {
		return fmt.Errorf("kustomize binary not found in PATH. Please install kustomize from https://kubectl.docs.kubernetes.io/installation/kustomize/")
	}

	// Create and execute kustomize command
	kustomizeExec := exec.Command(kustomizeBinary, args...)
	kustomizeExec.Stdin = os.Stdin
	kustomizeExec.Stdout = os.Stdout
	kustomizeExec.Stderr = os.Stderr
	kustomizeExec.Env = os.Environ()

	return kustomizeExec.Run()
}
