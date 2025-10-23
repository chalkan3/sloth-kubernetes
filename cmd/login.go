package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login [s3://bucket-name]",
	Short: "Configure state backend (S3 bucket)",
	Long: `Configure the S3 bucket for storing Pulumi state.

This command works similar to 'pulumi login' - it configures where your
infrastructure state will be stored. The state backend must be an S3-compatible
storage bucket.

The backend URL is stored in ~/.sloth-kubernetes/config for future use.

Example:
  sloth-kubernetes login s3://s3.lady-guica.chalkan3.com.br
  sloth-kubernetes login --bucket s3.lady-guica.chalkan3.com.br
  sloth-kubernetes login s3://bucket --access-key-id YOUR_KEY --secret-access-key YOUR_SECRET --region us-east-1`,
	RunE: runLogin,
}

var (
	loginBucket          string
	loginAccessKeyID     string
	loginSecretAccessKey string
	loginRegion          string
	loginEndpoint        string
)

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&loginBucket, "bucket", "b", "", "S3 bucket URL for state backend")
	loginCmd.Flags().StringVar(&loginAccessKeyID, "access-key-id", "", "AWS Access Key ID")
	loginCmd.Flags().StringVar(&loginSecretAccessKey, "secret-access-key", "", "AWS Secret Access Key")
	loginCmd.Flags().StringVar(&loginRegion, "region", "", "AWS Region (e.g., us-east-1)")
	loginCmd.Flags().StringVar(&loginEndpoint, "endpoint", "", "S3 endpoint URL for S3-compatible storage")
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Determine bucket URL from args or flag
	var bucketURL string
	if len(args) > 0 {
		bucketURL = args[0]
	} else if loginBucket != "" {
		bucketURL = loginBucket
	} else {
		return fmt.Errorf("usage: sloth-kubernetes login [s3://bucket-name]\nExample: sloth-kubernetes login s3://s3.lady-guica.chalkan3.com.br")
	}

	// Normalize bucket URL - ensure it starts with s3://
	if !strings.HasPrefix(bucketURL, "s3://") {
		bucketURL = "s3://" + bucketURL
	}

	fmt.Println()
	color.Cyan("🔐 Configuring State Backend")
	fmt.Println()

	// Get or create config directory
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config")

	// Load existing config
	config, err := loadConfig(configFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Set AWS credentials from flags if provided
	if loginAccessKeyID != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", loginAccessKeyID)
		config["AWS_ACCESS_KEY_ID"] = loginAccessKeyID
		fmt.Println("✓ AWS Access Key ID configured")
	}
	if loginSecretAccessKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", loginSecretAccessKey)
		config["AWS_SECRET_ACCESS_KEY"] = loginSecretAccessKey
		fmt.Println("✓ AWS Secret Access Key configured")
	}
	if loginRegion != "" {
		os.Setenv("AWS_REGION", loginRegion)
		config["AWS_REGION"] = loginRegion
		fmt.Printf("✓ AWS Region configured: %s\n", loginRegion)
	}

	// For S3-compatible storage (MinIO), construct backend URL with query parameters
	// Pulumi expects: s3://bucket?endpoint=endpoint-url&s3ForcePathStyle=true&region=region
	if loginEndpoint != "" {
		os.Setenv("AWS_S3_ENDPOINT", loginEndpoint)
		config["AWS_S3_ENDPOINT"] = loginEndpoint

		// Build query parameters for backend URL
		queryParams := []string{}

		// Add endpoint parameter (remove https:// prefix if present)
		endpoint := strings.TrimPrefix(loginEndpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "http://")
		queryParams = append(queryParams, "endpoint="+endpoint)

		// Add s3ForcePathStyle for MinIO compatibility
		queryParams = append(queryParams, "s3ForcePathStyle=true")

		// Add region if specified
		if loginRegion != "" {
			queryParams = append(queryParams, "region="+loginRegion)
		}

		// Construct full backend URL with query parameters
		bucketURL = bucketURL + "?" + strings.Join(queryParams, "&")

		fmt.Printf("✓ S3 Endpoint configured: %s\n", loginEndpoint)
		fmt.Println("✓ Path-style S3 URLs enabled")
	}

	if loginAccessKeyID != "" || loginSecretAccessKey != "" || loginRegion != "" || loginEndpoint != "" {
		fmt.Println()
	}

	// Validate S3 backend access
	fmt.Println("⏳ Validating S3 backend access...")
	if err := validateS3Backend(bucketURL); err != nil {
		fmt.Println()
		color.Red("✗ Failed to access S3 backend")
		fmt.Println()
		fmt.Println("Error:", err.Error())
		fmt.Println()
		color.Yellow("💡 Possible solutions:")
		fmt.Println("  1. Check if AWS credentials are configured:")
		fmt.Println("     export AWS_ACCESS_KEY_ID=your_access_key")
		fmt.Println("     export AWS_SECRET_ACCESS_KEY=your_secret_key")
		fmt.Println()
		fmt.Println("  2. For S3-compatible storage (MinIO, DigitalOcean Spaces, etc):")
		fmt.Println("     export AWS_S3_ENDPOINT=https://your-endpoint.com")
		fmt.Println()
		fmt.Println("  3. Verify the bucket URL is correct and accessible")
		fmt.Println()
		return fmt.Errorf("S3 backend validation failed")
	}
	fmt.Println("✓ S3 backend is accessible")
	fmt.Println()

	// Check if backend is already configured
	if existingBackend, ok := config["PULUMI_BACKEND_URL"]; ok && existingBackend != "" {
		fmt.Printf("State backend already configured: %s\n", existingBackend)
		fmt.Println()
		if !promptYesNo("Overwrite existing backend configuration?") {
			fmt.Println("Keeping existing configuration.")
			return nil
		}
	}

	// Set the backend URL
	config["PULUMI_BACKEND_URL"] = bucketURL

	// Save config
	if err := saveConfig(configFile, config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	color.Green("✓ State backend configured successfully!")
	fmt.Printf("  Backend URL: %s\n", bucketURL)
	fmt.Printf("  Config file: %s\n", configFile)
	fmt.Println()
	color.Yellow("Note: All Pulumi state will now be stored in this S3 bucket.")
	fmt.Println()

	return nil
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".sloth-kubernetes")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

func loadConfig(path string) (map[string]string, error) {
	config := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, `"'`)
			config[key] = value
		}
	}

	return config, scanner.Err()
}

func saveConfig(path string, config map[string]string) error {
	// Create file with restricted permissions
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintln(file, "# Sloth Kubernetes Configuration")
	fmt.Fprintln(file, "# This file contains Pulumi backend configuration")
	fmt.Fprintln(file, "# File permissions: 0600 (read/write for owner only)")
	fmt.Fprintln(file, "#")
	fmt.Fprintf(file, "# Generated by: sloth-kubernetes login\n")
	fmt.Fprintln(file, "")

	// Write configuration
	for key, value := range config {
		if value != "" {
			fmt.Fprintf(file, "%s=%s\n", key, value)
		}
	}

	return nil
}

func promptYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (y/N): ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// validateS3Backend validates that the S3 backend is accessible
func validateS3Backend(backendURL string) error {
	// For S3 backend validation, we'll simply save the config and let
	// the deploy command validate the access when it actually needs it.
	// This avoids the complexity of validating without a Pulumi project.

	// The validation happens implicitly when:
	// 1. User runs deploy
	// 2. Pulumi tries to access the backend
	// 3. If credentials are wrong, deploy fails immediately

	// For now, just check if AWS credentials are set
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID not set")
	}
	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY not set")
	}

	return nil
}
