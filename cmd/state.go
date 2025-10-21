package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage Pulumi state backend",
	Long:  `Configure and manage where Pulumi stores your infrastructure state`,
}

var stateLocateCmd = &cobra.Command{
	Use:   "locate [backend-url]",
	Short: "Configure state backend location (equivalent to pulumi login)",
	Long: `Configure where Pulumi stores infrastructure state.

Supported backends:
  - Local filesystem:  file://~/.pulumi (default)
  - S3:               s3://bucket-name
  - Azure Blob:       azblob://container-name
  - Google Cloud:     gs://bucket-name
  - Pulumi Cloud:     (leave empty or use 'cloud')

Examples:
  # Use local filesystem (default)
  sloth-kubernetes state locate file://~/.pulumi
  
  # Use Pulumi Cloud
  sloth-kubernetes state locate
  
  # Use S3 bucket
  sloth-kubernetes state locate s3://my-pulumi-state`,
	Example: `  # Show current backend
  sloth-kubernetes state locate
  
  # Set local backend
  sloth-kubernetes state locate file://~/.pulumi
  
  # Set S3 backend
  sloth-kubernetes state locate s3://my-state-bucket`,
	RunE: runStateLocate,
}

var stateCancelCmd = &cobra.Command{
	Use:   "cancel [stack-name]",
	Short: "Cancel an in-progress operation (equivalent to pulumi cancel)",
	Long: `Cancel an in-progress update or destroy operation for a stack.

This command removes the lock file that prevents concurrent operations.
Use this if a deployment was interrupted and the stack is stuck in a locked state.`,
	Example: `  # Cancel operation on production stack
  sloth-kubernetes state cancel production
  
  # Cancel with auto-confirm
  sloth-kubernetes state cancel production --yes`,
	RunE: runStateCancel,
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateLocateCmd)
	stateCmd.AddCommand(stateCancelCmd)
}

func runStateLocate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// If no arguments, show current backend
	if len(args) == 0 {
		return showCurrentBackend(ctx)
	}

	backendURL := args[0]

	// Handle "cloud" alias
	if backendURL == "cloud" {
		backendURL = ""
	}

	// Expand home directory in file:// URLs
	if strings.HasPrefix(backendURL, "file://~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		backendURL = "file://" + filepath.Join(home, backendURL[8:])
	}

	printHeader(fmt.Sprintf("🔧 Configuring state backend"))

	// The backend is configured via PULUMI_BACKEND_URL environment variable
	if backendURL != "" {
		printInfo(fmt.Sprintf("Setting backend to: %s", backendURL))
		
		// Set the backend URL in environment for future operations
		if err := os.Setenv("PULUMI_BACKEND_URL", backendURL); err != nil {
			return fmt.Errorf("failed to set backend URL: %w", err)
		}

		// Create a .pulumi directory if using local backend
		if strings.HasPrefix(backendURL, "file://") {
			localPath := strings.TrimPrefix(backendURL, "file://")
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return fmt.Errorf("failed to create backend directory: %w", err)
			}
			printSuccess(fmt.Sprintf("Created backend directory: %s", localPath))
		}
	} else {
		printInfo("Setting backend to: Pulumi Cloud")
		os.Unsetenv("PULUMI_BACKEND_URL")
	}

	// Show the current backend
	fmt.Println()
	return showCurrentBackend(ctx)
}

func showCurrentBackend(ctx context.Context) error {
	printHeader("📍 Current State Backend")
	
	// Check environment variable first
	backendURL := os.Getenv("PULUMI_BACKEND_URL")
	
	if backendURL == "" {
		// Try to get from workspace
		workspace, err := auto.NewLocalWorkspace(ctx, auto.WorkDir("."))
		if err == nil {
			// Get the current project settings
			project, err := workspace.ProjectSettings(ctx)
			if err == nil && project.Backend != nil {
				backendURL = project.Backend.URL
			}
		}
	}

	if backendURL == "" {
		backendURL = "Pulumi Cloud (default)"
	}

	color.New(color.FgCyan, color.Bold).Println("\nBackend URL:")
	fmt.Printf("  %s\n", backendURL)

	// Show backend type
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("Backend Type:")
	backendType := getBackendType(backendURL)
	fmt.Printf("  %s\n", backendType)

	// Show additional info
	if strings.HasPrefix(backendURL, "file://") {
		localPath := strings.TrimPrefix(backendURL, "file://")
		fmt.Println()
		color.New(color.FgCyan, color.Bold).Println("Local Path:")
		fmt.Printf("  %s\n", localPath)
		
		// Check if directory exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			color.Yellow("\n⚠️  Warning: Backend directory does not exist")
			printInfo(fmt.Sprintf("Run 'mkdir -p %s' to create it", localPath))
		} else {
			color.Green("\n✅ Backend directory exists")
		}
	}

	return nil
}

func getBackendType(backendURL string) string {
	switch {
	case backendURL == "Pulumi Cloud (default)" || backendURL == "":
		return "Pulumi Cloud (managed service)"
	case strings.HasPrefix(backendURL, "file://"):
		return "Local Filesystem"
	case strings.HasPrefix(backendURL, "s3://"):
		return "AWS S3"
	case strings.HasPrefix(backendURL, "azblob://"):
		return "Azure Blob Storage"
	case strings.HasPrefix(backendURL, "gs://"):
		return "Google Cloud Storage"
	default:
		return "Custom/Unknown"
	}
}

func runStateCancel(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: sloth-kubernetes state cancel <stack-name>")
	}

	stack := args[0]

	printHeader(fmt.Sprintf("🔓 Canceling operation for stack: %s", stack))

	// Confirm with user unless --yes flag is set
	if !autoApprove {
		color.Yellow("\n⚠️  This will unlock the stack and cancel any in-progress operations.")
		color.Yellow("   Only do this if you're sure no other process is using the stack.")
		fmt.Print("\nContinue? (yes/no): ")
		
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
			printInfo("Operation cancelled")
			return nil
		}
	}

	// Get the home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try to find and remove lock files
	// Pulumi stores locks in ~/.pulumi/locks/organization/project/stack/
	lockPaths := []string{
		// Local backend lock
		filepath.Join(home, ".pulumi", "locks", "organization", "kubernetes-create", stack),
		// Alternative paths
		filepath.Join(home, ".pulumi", "stacks", "organization", "kubernetes-create", stack, ".pulumi", "locks"),
	}

	lockFound := false
	for _, lockPath := range lockPaths {
		// Check if lock directory exists
		if info, err := os.Stat(lockPath); err == nil && info.IsDir() {
			// Remove all lock files in this directory
			files, err := os.ReadDir(lockPath)
			if err != nil {
				printWarning(fmt.Sprintf("Failed to read lock directory %s: %v", lockPath, err))
				continue
			}

			if len(files) > 0 {
				lockFound = true
				printInfo(fmt.Sprintf("Found %d lock file(s) in: %s", len(files), lockPath))
				
				for _, file := range files {
					if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
						lockFile := filepath.Join(lockPath, file.Name())
						if err := os.Remove(lockFile); err != nil {
							printWarning(fmt.Sprintf("Failed to remove lock file %s: %v", lockFile, err))
						} else {
							printSuccess(fmt.Sprintf("Removed lock file: %s", file.Name()))
						}
					}
				}
			}
		}
	}

	if !lockFound {
		printWarning("No lock files found for this stack")
		printInfo("The stack may not be locked, or it's using a different backend")
	} else {
		fmt.Println()
		printSuccess("Stack unlocked successfully!")
		printInfo("You can now run deploy or destroy operations on this stack")
	}

	return nil
}

func printWarning(msg string) {
	color.Yellow("⚠️  " + msg)
}
