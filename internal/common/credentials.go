package common

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadSavedConfig loads config from ~/.sloth-kubernetes/config
// and sets them as environment variables if not already set
func LoadSavedConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil // Silently skip if we can't get home dir
	}

	configFile := filepath.Join(home, ".sloth-kubernetes", "config")

	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil // No saved config, that's ok
	}

	// Load config
	config, err := loadConfigFile(configFile)
	if err != nil {
		return nil // Silently skip on error
	}

	// Set environment variables from config file
	// IMPORTANT: Always override existing environment variables with values from config file
	// This ensures the S3 backend credentials from ~/.sloth-kubernetes/config take precedence
	for key, value := range config {
		os.Setenv(key, value)
	}

	return nil
}

// Deprecated: Use LoadSavedConfig instead
func LoadSavedCredentials() error {
	return LoadSavedConfig()
}

func loadConfigFile(path string) (map[string]string, error) {
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

// GetCredentialsStatus returns information about saved credentials
func GetCredentialsStatus() (bool, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, "", err
	}

	credsFile := filepath.Join(home, ".sloth-kubernetes", "credentials")

	if _, err := os.Stat(credsFile); os.IsNotExist(err) {
		return false, credsFile, nil
	}

	return true, credsFile, nil
}
