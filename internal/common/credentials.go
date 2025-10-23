package common

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadSavedCredentials loads credentials from ~/.sloth-kubernetes/credentials
// and sets them as environment variables if not already set
func LoadSavedCredentials() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil // Silently skip if we can't get home dir
	}

	credsFile := filepath.Join(home, ".sloth-kubernetes", "credentials")

	// Check if file exists
	if _, err := os.Stat(credsFile); os.IsNotExist(err) {
		return nil // No saved credentials, that's ok
	}

	// Load credentials
	creds, err := loadCredentialsFile(credsFile)
	if err != nil {
		return nil // Silently skip on error
	}

	// Set environment variables if not already set
	for key, value := range creds {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return nil
}

func loadCredentialsFile(path string) (map[string]string, error) {
	creds := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return creds, err
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
			creds[key] = value
		}
	}

	return creds, scanner.Err()
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
