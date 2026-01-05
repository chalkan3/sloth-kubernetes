package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKubectlCmd_Structure(t *testing.T) {
	assert.NotNil(t, kubectlCmd)
	assert.Equal(t, "kubectl [stack-name] [kubectl-args...]", kubectlCmd.Use)
	assert.NotEmpty(t, kubectlCmd.Short)
	assert.NotEmpty(t, kubectlCmd.Long)
	assert.NotEmpty(t, kubectlCmd.Example)
}

func TestKubectlCmd_DisableFlagParsing(t *testing.T) {
	assert.True(t, kubectlCmd.DisableFlagParsing, "kubectl command should have DisableFlagParsing=true")
}

func TestKubectlCmd_RunE(t *testing.T) {
	assert.NotNil(t, kubectlCmd.RunE, "kubectl command should have RunE function")
}

func TestKubectlCmd_Examples(t *testing.T) {
	examples := kubectlCmd.Example
	assert.Contains(t, examples, "get nodes")
	assert.Contains(t, examples, "get pods")
	assert.Contains(t, examples, "apply")
	assert.Contains(t, examples, "logs")
	assert.Contains(t, examples, "exec")
}

func TestKubectlCmd_LongDescription(t *testing.T) {
	long := kubectlCmd.Long
	assert.Contains(t, long, "kubectl")
	assert.Contains(t, long, "embedded")
	assert.Contains(t, long, "kubeconfig")
}

func TestKubectlCmd_StackAwareUsage(t *testing.T) {
	// Verify the command usage shows stack-name as first argument
	assert.Contains(t, kubectlCmd.Use, "stack-name")
	assert.Contains(t, kubectlCmd.Use, "kubectl-args")
}

func TestRunKubectl_RequiresStackName(t *testing.T) {
	// Test that running kubectl without arguments returns an error
	err := runKubectl(kubectlCmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stack name is required")
}

func TestRunKubectl_WithInvalidStack(t *testing.T) {
	// Test with a non-existent stack name
	err := runKubectl(kubectlCmd, []string{"non-existent-stack", "get", "nodes"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get kubeconfig")
}
