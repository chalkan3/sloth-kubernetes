package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestExtractBastionIP tests the extraction of bastion IP from various Pulumi output formats
func TestExtractBastionIP(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expected  string
		shouldErr bool
	}{
		{
			name:      "Direct string IP",
			input:     "192.168.1.100",
			expected:  "192.168.1.100",
			shouldErr: false,
		},
		{
			name: "Map with public_ip",
			input: map[string]interface{}{
				"public_ip":  "203.0.113.10",
				"private_ip": "10.0.0.5",
				"name":       "bastion",
			},
			expected:  "203.0.113.10",
			shouldErr: false,
		},
		{
			name: "Map with ip key",
			input: map[string]interface{}{
				"ip":   "198.51.100.20",
				"name": "bastion",
			},
			expected:  "198.51.100.20",
			shouldErr: false,
		},
		{
			name: "Map with ipv4 key",
			input: map[string]interface{}{
				"ipv4": "192.0.2.30",
				"name": "bastion",
			},
			expected:  "192.0.2.30",
			shouldErr: false,
		},
		{
			name: "Map with address key",
			input: map[string]interface{}{
				"address": "198.18.0.40",
				"name":    "bastion",
			},
			expected:  "198.18.0.40",
			shouldErr: false,
		},
		{
			name: "Array with bastion by name",
			input: []interface{}{
				map[string]interface{}{
					"name":       "master-1",
					"public_ip":  "203.0.113.50",
					"private_ip": "10.0.0.10",
				},
				map[string]interface{}{
					"name":       "bastion",
					"public_ip":  "203.0.113.60",
					"private_ip": "10.0.0.5",
				},
				map[string]interface{}{
					"name":       "worker-1",
					"public_ip":  "203.0.113.70",
					"private_ip": "10.0.0.20",
				},
			},
			expected:  "203.0.113.60",
			shouldErr: false,
		},
		{
			name: "Array with bastion by role",
			input: []interface{}{
				map[string]interface{}{
					"name":       "master-1",
					"public_ip":  "203.0.113.80",
					"roles":      []interface{}{"master", "controlplane"},
					"private_ip": "10.0.0.10",
				},
				map[string]interface{}{
					"name":       "bastion-host",
					"public_ip":  "203.0.113.90",
					"roles":      []interface{}{"bastion"},
					"private_ip": "10.0.0.5",
				},
			},
			expected:  "203.0.113.90",
			shouldErr: false,
		},
		{
			name: "Map with empty public_ip",
			input: map[string]interface{}{
				"public_ip":  "",
				"private_ip": "10.0.0.5",
			},
			expected:  "",
			shouldErr: true,
		},
		{
			name: "Map with no IP fields",
			input: map[string]interface{}{
				"name": "bastion",
				"id":   "12345",
			},
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "Empty array",
			input:     []interface{}{},
			expected:  "",
			shouldErr: true,
		},
		{
			name: "Array without bastion",
			input: []interface{}{
				map[string]interface{}{
					"name":      "master-1",
					"public_ip": "203.0.113.100",
				},
				map[string]interface{}{
					"name":      "worker-1",
					"public_ip": "203.0.113.110",
				},
			},
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "Invalid type - integer",
			input:     12345,
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "Invalid type - boolean",
			input:     true,
			expected:  "",
			shouldErr: true,
		},
		{
			name:      "Nil input",
			input:     nil,
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractBastionIP(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none. Result: %s", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected IP %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// TestSaveSaltConfig tests saving and loading Salt configuration
func TestSaveSaltConfig(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sloth-k8s-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config subdirectory
	configDir := filepath.Join(tempDir, ".sloth-kubernetes")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	tests := []struct {
		name   string
		config SaltConfig
	}{
		{
			name: "Basic configuration",
			config: SaltConfig{
				APIURL:     "http://192.168.1.100:8000",
				Username:   "saltapi",
				Password:   "saltapi123",
				BastionIP:  "192.168.1.100",
				StackName:  "dev",
				ConfigFile: "cluster.yaml",
			},
		},
		{
			name: "Configuration with special characters",
			config: SaltConfig{
				APIURL:     "http://203.0.113.50:8000",
				Username:   "salt-admin",
				Password:   "P@ssw0rd!#$%",
				BastionIP:  "203.0.113.50",
				StackName:  "production-cluster-001",
				ConfigFile: "/path/to/cluster.yaml",
			},
		},
		{
			name: "Minimal configuration",
			config: SaltConfig{
				APIURL:    "http://10.0.0.5:8000",
				Username:  "admin",
				Password:  "pass",
				BastionIP: "10.0.0.5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override home directory for test
			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", originalHome)

			// Save configuration
			err := saveSaltConfig(tt.config)
			if err != nil {
				t.Fatalf("Failed to save config: %v", err)
			}

			// Verify file was created
			configFile := filepath.Join(configDir, "salt-config.json")
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				t.Fatalf("Config file was not created: %v", err)
			}

			// Check file permissions
			fileInfo, err := os.Stat(configFile)
			if err != nil {
				t.Fatalf("Failed to stat config file: %v", err)
			}
			expectedPerms := os.FileMode(0600)
			if fileInfo.Mode().Perm() != expectedPerms {
				t.Errorf("Expected file permissions %v, got %v", expectedPerms, fileInfo.Mode().Perm())
			}

			// Read and verify content
			data, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}

			var savedConfig SaltConfig
			if err := json.Unmarshal(data, &savedConfig); err != nil {
				t.Fatalf("Failed to parse saved config: %v", err)
			}

			// Verify all fields
			if savedConfig.APIURL != tt.config.APIURL {
				t.Errorf("APIURL: expected %q, got %q", tt.config.APIURL, savedConfig.APIURL)
			}
			if savedConfig.Username != tt.config.Username {
				t.Errorf("Username: expected %q, got %q", tt.config.Username, savedConfig.Username)
			}
			if savedConfig.Password != tt.config.Password {
				t.Errorf("Password: expected %q, got %q", tt.config.Password, savedConfig.Password)
			}
			if savedConfig.BastionIP != tt.config.BastionIP {
				t.Errorf("BastionIP: expected %q, got %q", tt.config.BastionIP, savedConfig.BastionIP)
			}
			if savedConfig.StackName != tt.config.StackName {
				t.Errorf("StackName: expected %q, got %q", tt.config.StackName, savedConfig.StackName)
			}
			if savedConfig.ConfigFile != tt.config.ConfigFile {
				t.Errorf("ConfigFile: expected %q, got %q", tt.config.ConfigFile, savedConfig.ConfigFile)
			}
		})
	}
}

// TestLoadSaltConfig tests loading Salt configuration
func TestLoadSaltConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		shouldError bool
		expected    *SaltConfig
	}{
		{
			name: "Valid configuration",
			configJSON: `{
				"api_url": "http://192.168.1.100:8000",
				"username": "saltapi",
				"password": "saltapi123",
				"bastion_ip": "192.168.1.100",
				"stack_name": "dev",
				"config_file": "cluster.yaml"
			}`,
			shouldError: false,
			expected: &SaltConfig{
				APIURL:     "http://192.168.1.100:8000",
				Username:   "saltapi",
				Password:   "saltapi123",
				BastionIP:  "192.168.1.100",
				StackName:  "dev",
				ConfigFile: "cluster.yaml",
			},
		},
		{
			name: "Minimal valid configuration",
			configJSON: `{
				"api_url": "http://10.0.0.5:8000",
				"username": "admin",
				"password": "pass",
				"bastion_ip": "10.0.0.5"
			}`,
			shouldError: false,
			expected: &SaltConfig{
				APIURL:    "http://10.0.0.5:8000",
				Username:  "admin",
				Password:  "pass",
				BastionIP: "10.0.0.5",
			},
		},
		{
			name:        "Invalid JSON",
			configJSON:  `{invalid json`,
			shouldError: true,
			expected:    nil,
		},
		{
			name:        "Empty JSON",
			configJSON:  `{}`,
			shouldError: false,
			expected:    &SaltConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := os.MkdirTemp("", "sloth-k8s-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create config directory and file
			configDir := filepath.Join(tempDir, ".sloth-kubernetes")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatalf("Failed to create config dir: %v", err)
			}

			configFile := filepath.Join(configDir, "salt-config.json")
			if err := os.WriteFile(configFile, []byte(tt.configJSON), 0600); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Override home directory for test
			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", originalHome)

			// Load configuration
			config, err := loadSaltConfig()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				if tt.expected != nil {
					if config.APIURL != tt.expected.APIURL {
						t.Errorf("APIURL: expected %q, got %q", tt.expected.APIURL, config.APIURL)
					}
					if config.Username != tt.expected.Username {
						t.Errorf("Username: expected %q, got %q", tt.expected.Username, config.Username)
					}
					if config.Password != tt.expected.Password {
						t.Errorf("Password: expected %q, got %q", tt.expected.Password, config.Password)
					}
					if config.BastionIP != tt.expected.BastionIP {
						t.Errorf("BastionIP: expected %q, got %q", tt.expected.BastionIP, config.BastionIP)
					}
					if config.StackName != tt.expected.StackName {
						t.Errorf("StackName: expected %q, got %q", tt.expected.StackName, config.StackName)
					}
					if config.ConfigFile != tt.expected.ConfigFile {
						t.Errorf("ConfigFile: expected %q, got %q", tt.expected.ConfigFile, config.ConfigFile)
					}
				}
			}
		})
	}
}

// TestLoadSaltConfigMissingFile tests loading when config file doesn't exist
func TestLoadSaltConfigMissingFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sloth-k8s-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory (no config file created)
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Try to load non-existent config
	_, err = loadSaltConfig()
	if err == nil {
		t.Error("Expected error when loading missing config file, got none")
	}
}

// TestSaltConfigRoundTrip tests saving and loading together
func TestSaltConfigRoundTrip(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sloth-k8s-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create original config
	originalConfig := SaltConfig{
		APIURL:     "http://198.51.100.75:8000",
		Username:   "testuser",
		Password:   "testpass123",
		BastionIP:  "198.51.100.75",
		StackName:  "test-stack",
		ConfigFile: "/tmp/test.yaml",
	}

	// Save config
	if err := saveSaltConfig(originalConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := loadSaltConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Compare all fields
	if loadedConfig.APIURL != originalConfig.APIURL {
		t.Errorf("APIURL mismatch: expected %q, got %q", originalConfig.APIURL, loadedConfig.APIURL)
	}
	if loadedConfig.Username != originalConfig.Username {
		t.Errorf("Username mismatch: expected %q, got %q", originalConfig.Username, loadedConfig.Username)
	}
	if loadedConfig.Password != originalConfig.Password {
		t.Errorf("Password mismatch: expected %q, got %q", originalConfig.Password, loadedConfig.Password)
	}
	if loadedConfig.BastionIP != originalConfig.BastionIP {
		t.Errorf("BastionIP mismatch: expected %q, got %q", originalConfig.BastionIP, loadedConfig.BastionIP)
	}
	if loadedConfig.StackName != originalConfig.StackName {
		t.Errorf("StackName mismatch: expected %q, got %q", originalConfig.StackName, loadedConfig.StackName)
	}
	if loadedConfig.ConfigFile != originalConfig.ConfigFile {
		t.Errorf("ConfigFile mismatch: expected %q, got %q", originalConfig.ConfigFile, loadedConfig.ConfigFile)
	}
}

// TestSaltConfigMultipleSaves tests that subsequent saves overwrite previous ones
func TestSaltConfigMultipleSaves(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sloth-k8s-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Save first config
	config1 := SaltConfig{
		APIURL:    "http://192.168.1.100:8000",
		Username:  "user1",
		Password:  "pass1",
		BastionIP: "192.168.1.100",
		StackName: "stack1",
	}
	if err := saveSaltConfig(config1); err != nil {
		t.Fatalf("Failed to save first config: %v", err)
	}

	// Save second config (should overwrite)
	config2 := SaltConfig{
		APIURL:    "http://10.0.0.50:8000",
		Username:  "user2",
		Password:  "pass2",
		BastionIP: "10.0.0.50",
		StackName: "stack2",
	}
	if err := saveSaltConfig(config2); err != nil {
		t.Fatalf("Failed to save second config: %v", err)
	}

	// Load config
	loadedConfig, err := loadSaltConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Should match config2, not config1
	if loadedConfig.APIURL != config2.APIURL {
		t.Errorf("Expected config2 APIURL %q, got %q", config2.APIURL, loadedConfig.APIURL)
	}
	if loadedConfig.Username != config2.Username {
		t.Errorf("Expected config2 Username %q, got %q", config2.Username, loadedConfig.Username)
	}
	if loadedConfig.BastionIP != config2.BastionIP {
		t.Errorf("Expected config2 BastionIP %q, got %q", config2.BastionIP, loadedConfig.BastionIP)
	}
	if loadedConfig.StackName != config2.StackName {
		t.Errorf("Expected config2 StackName %q, got %q", config2.StackName, loadedConfig.StackName)
	}
}

// BenchmarkExtractBastionIP benchmarks the bastion IP extraction
func BenchmarkExtractBastionIP(b *testing.B) {
	testCases := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "DirectString",
			input: "192.168.1.100",
		},
		{
			name: "SimpleMap",
			input: map[string]interface{}{
				"public_ip": "203.0.113.10",
			},
		},
		{
			name: "ComplexArray",
			input: []interface{}{
				map[string]interface{}{
					"name":      "master-1",
					"public_ip": "203.0.113.50",
				},
				map[string]interface{}{
					"name":      "bastion",
					"public_ip": "203.0.113.60",
				},
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = extractBastionIP(tc.input)
			}
		})
	}
}

// BenchmarkSaveSaltConfig benchmarks saving Salt configuration
func BenchmarkSaveSaltConfig(b *testing.B) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sloth-k8s-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	config := SaltConfig{
		APIURL:     "http://192.168.1.100:8000",
		Username:   "saltapi",
		Password:   "saltapi123",
		BastionIP:  "192.168.1.100",
		StackName:  "dev",
		ConfigFile: "cluster.yaml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = saveSaltConfig(config)
	}
}

// BenchmarkLoadSaltConfig benchmarks loading Salt configuration
func BenchmarkLoadSaltConfig(b *testing.B) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "sloth-k8s-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create config file
	config := SaltConfig{
		APIURL:     "http://192.168.1.100:8000",
		Username:   "saltapi",
		Password:   "saltapi123",
		BastionIP:  "192.168.1.100",
		StackName:  "dev",
		ConfigFile: "cluster.yaml",
	}
	if err := saveSaltConfig(config); err != nil {
		b.Fatalf("Failed to save config: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loadSaltConfig()
	}
}
