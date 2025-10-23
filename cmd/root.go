package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sloth-kubernetes/internal/common"
)

var (
	cfgFile     string
	stackName   string
	verbose     bool
	autoApprove bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "sloth-kubernetes",
	Short: "Multi-cloud Kubernetes cluster deployment tool",
	Long: `Sloth Kubernetes is a CLI tool for deploying production-grade
Kubernetes clusters across multiple cloud providers (DigitalOcean and Linode)
with RKE2, WireGuard VPN mesh, and automated configuration.

This tool uses Pulumi Automation API internally - no Pulumi CLI required!
Stack-based deployment enables managing multiple independent clusters.`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Load saved credentials before running any command
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default: ./cluster-config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&stackName, "stack", "s", "production", "Pulumi stack name")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&autoApprove, "yes", "y", false, "Auto-approve without prompting")
}

func initConfig() {
	// Load saved credentials from ~/.sloth-kubernetes/credentials
	// This runs before every command, allowing saved credentials to be used
	_ = common.LoadSavedCredentials()
}
