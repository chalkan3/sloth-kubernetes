package cmd

import (
	"fmt"
	"os"

	"github.com/chalkan3/sloth-kubernetes/internal/common"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	stackName   string
	verbose     bool
	autoApprove bool

	// Version information - set by main.go
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "unknown"
)

// SetVersionInfo sets the version information from main.go
func SetVersionInfo(version, commit, date, builtBy string) {
	Version = version
	Commit = commit
	Date = date
	BuiltBy = builtBy
	// Also set in common package for use by internal packages
	common.SetVersionInfo(version, commit, date, builtBy)
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "sloth",
	Short: "Multi-cloud Kubernetes cluster deployment tool",
	Long: `Sloth Kubernetes is a CLI tool for deploying production-grade
Kubernetes clusters across multiple cloud providers (AWS, DigitalOcean, Linode, Azure)
with RKE2, WireGuard VPN mesh, and automated configuration.

This tool uses Pulumi Automation API internally - no Pulumi CLI required!
Stack-based deployment enables managing multiple independent clusters.`,
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
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default: ./cluster-config.lisp)")
	rootCmd.PersistentFlags().StringVarP(&stackName, "stack", "s", "production", "Pulumi stack name")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVarP(&autoApprove, "yes", "y", false, "Auto-approve without prompting")

	// Version template
	rootCmd.SetVersionTemplate(fmt.Sprintf(`Sloth Kubernetes %s
  Commit:    %s
  Built:     %s
  Built by:  %s
`, Version, Commit, Date, BuiltBy))
	rootCmd.Version = Version
}

func initConfig() {
	// Load saved credentials from ~/.sloth-kubernetes/credentials
	// This runs before every command, allowing saved credentials to be used
	_ = common.LoadSavedCredentials()
}
