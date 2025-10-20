package cmd

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version   = "1.0.0"
	gitCommit = "dev"
	buildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display version information for kubernetes-create CLI",
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Println()
	color.Cyan("🚀 Kubernetes-Create CLI")
	fmt.Println()
	fmt.Printf("Version:    %s\n", version)
	fmt.Printf("Git Commit: %s\n", gitCommit)
	fmt.Printf("Build Date: %s\n", buildDate)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	color.Yellow("💡 Using Pulumi Automation API (no Pulumi CLI required!)")
	fmt.Println()
}
