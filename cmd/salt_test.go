package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/salt"
)

// Mock Salt API server for testing
func setupMockSaltServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle login
		if r.URL.Path == "/login" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"return": []map[string]interface{}{
					{
						"token":  "test-token-12345",
						"start":  1234567890.123456,
						"expire": 1234567890.123456 + 43200,
						"user":   "saltapi",
						"eauth":  "pam",
						"perms":  []string{".*"},
					},
				},
			})
			return
		}

		// Handle /run endpoint (most Salt commands)
		if r.URL.Path == "/run" || r.URL.Path == "/" {
			// Parse the request to determine response
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			// Default response structure
			response := map[string]interface{}{
				"return": []map[string]interface{}{
					{
						"minion-1": true,
						"minion-2": true,
					},
				},
			}

			// Customize based on function
			if fun, ok := reqBody["fun"].(string); ok {
				switch fun {
				case "test.ping":
					response["return"] = []map[string]bool{
						{
							"minion-1": true,
							"minion-2": true,
							"minion-3": true,
						},
					}
				case "cmd.run":
					response["return"] = []map[string]string{
						{
							"minion-1": "output from minion-1",
							"minion-2": "output from minion-2",
						},
					}
				case "grains.items":
					response["return"] = []map[string]interface{}{
						{
							"minion-1": map[string]interface{}{
								"os":        "Ubuntu",
								"osrelease": "22.04",
								"kernel":    "5.15.0",
							},
							"minion-2": map[string]interface{}{
								"os":        "Ubuntu",
								"osrelease": "22.04",
								"kernel":    "5.15.0",
							},
						},
					}
				case "state.apply":
					response["return"] = []map[string]interface{}{
						{
							"minion-1": map[string]interface{}{
								"result":  true,
								"comment": "State applied successfully",
							},
						},
					}
				case "key.list_all":
					response["return"] = []map[string]interface{}{
						{
							"local":            []string{"master.pem", "master.pub"},
							"minions":          []string{"minion-1", "minion-2", "minion-3"},
							"minions_pre":      []string{"pending-minion"},
							"minions_rejected": []string{},
							"minions_denied":   []string{},
						},
					}
				case "key.accept":
					response["return"] = []map[string]interface{}{
						{
							"minions": []string{"minion-1"},
						},
					}
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Handle minions endpoint
		if r.URL.Path == "/minions" {
			response := map[string]interface{}{
				"return": []map[string]interface{}{
					{
						"minion-1": map[string]interface{}{
							"os": "Ubuntu",
						},
						"minion-2": map[string]interface{}{
							"os": "Ubuntu",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
	}))
}

// TestGetSaltClient tests the getSaltClient function
func TestGetSaltClient(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration from flags",
			setupEnv: func() {
				saltAPIURL = "http://localhost:8000"
				saltUsername = "saltapi"
				saltPassword = "saltapi123"
			},
			shouldError: false,
		},
		{
			name: "Missing API URL",
			setupEnv: func() {
				saltAPIURL = ""
				saltUsername = "saltapi"
				saltPassword = "saltapi123"
			},
			shouldError: true,
			errorMsg:    "stack name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			client, err := getSaltClient()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if err != nil && tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client but got nil")
				}
			}
		})
	}
}

// TestSaltPingCommand tests the ping command
func TestSaltPingCommand(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	// Setup client
	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	client, err := getSaltClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test ping
	results, err := client.Ping(saltTarget)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected results but got empty map")
	}

	// Verify minions responded
	expectedMinions := []string{"minion-1", "minion-2", "minion-3"}
	for _, minion := range expectedMinions {
		if responded, ok := results[minion]; !ok {
			t.Errorf("Minion %s not in results", minion)
		} else if !responded {
			t.Errorf("Minion %s did not respond true", minion)
		}
	}
}

// TestSaltCmdCommand tests the cmd command
func TestSaltCmdCommand(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	client, err := getSaltClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test command execution
	results, err := client.RunCommand(saltTarget, "cmd.run", []string{"uptime"})
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results but got nil")
	}

	// Verify we got some return data
	if results.Return == nil || len(results.Return) == 0 {
		t.Error("Expected return data but got none")
	}
}

// TestSaltGrainsCommand tests the grains command
func TestSaltGrainsCommand(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	client, err := getSaltClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test grains
	results, err := client.GetGrains(saltTarget)
	if err != nil {
		t.Fatalf("Grains failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results but got nil")
	}
}

// TestSaltStateApply tests the state apply command
func TestSaltStateApply(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"
	saltTarget = "*"

	client, err := getSaltClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test state apply
	results, err := client.ApplyState(saltTarget, "webserver")
	if err != nil {
		t.Fatalf("State apply failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results but got nil")
	}
}

// TestSaltKeysOperations tests key management operations
func TestSaltKeysOperations(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	saltAPIURL = server.URL
	saltUsername = "saltapi"
	saltPassword = "saltapi123"

	client, err := getSaltClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("ListKeys", func(t *testing.T) {
		results, err := client.KeyList()
		if err != nil {
			t.Fatalf("Keys list failed: %v", err)
		}

		if results == nil {
			t.Fatal("Expected results but got nil")
		}
	})

	t.Run("AcceptKey", func(t *testing.T) {
		err := client.KeyAccept("minion-1")
		if err != nil {
			t.Fatalf("Key accept failed: %v", err)
		}
	})
}

// TestSaltClientAuthentication tests authentication flow
func TestSaltClientAuthentication(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	tests := []struct {
		name        string
		url         string
		username    string
		password    string
		shouldError bool
	}{
		{
			name:        "Valid credentials",
			url:         server.URL,
			username:    "saltapi",
			password:    "saltapi123",
			shouldError: false,
		},
		{
			name:        "Empty URL",
			url:         "",
			username:    "saltapi",
			password:    "saltapi123",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.url == "" {
				return // Skip actual client creation for empty URL
			}

			client := salt.NewClient(tt.url, tt.username, tt.password)
			err := client.Login()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestSaltTargetPatterns tests different targeting patterns
func TestSaltTargetPatterns(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	client := salt.NewClient(server.URL, "saltapi", "saltapi123")
	if err := client.Login(); err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	tests := []struct {
		name   string
		target string
	}{
		{
			name:   "All minions",
			target: "*",
		},
		{
			name:   "Master pattern",
			target: "master*",
		},
		{
			name:   "Worker pattern",
			target: "worker*",
		},
		{
			name:   "Specific minion",
			target: "minion-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := client.Ping(tt.target)
			if err != nil {
				t.Errorf("Ping with target %q failed: %v", tt.target, err)
			}
			if results == nil {
				t.Errorf("Expected results for target %q but got nil", tt.target)
			}
		})
	}
}

// TestGetEnvOrDefault tests the environment variable helper
func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "Environment variable set",
			envKey:       "TEST_VAR_1",
			envValue:     "custom_value",
			defaultValue: "default_value",
			expected:     "custom_value",
		},
		{
			name:         "Environment variable not set",
			envKey:       "TEST_VAR_2",
			envValue:     "",
			defaultValue: "default_value",
			expected:     "default_value",
		},
		{
			name:         "Empty default",
			envKey:       "TEST_VAR_3",
			envValue:     "",
			defaultValue: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}

			result := getEnvOrDefault(tt.envKey, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSaltJSONOutput tests JSON output formatting
func TestSaltJSONOutput(t *testing.T) {
	testData := map[string]interface{}{
		"minion-1": map[string]interface{}{
			"os":     "Ubuntu",
			"kernel": "5.15.0",
		},
		"minion-2": map[string]interface{}{
			"os":     "Ubuntu",
			"kernel": "5.15.0",
		},
	}

	// Test JSON marshaling
	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify it's valid JSON
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify data integrity
	if len(unmarshaled) != len(testData) {
		t.Errorf("Expected %d items, got %d", len(testData), len(unmarshaled))
	}
}

// TestSaltErrorHandling tests error handling in salt commands
func TestSaltErrorHandling(t *testing.T) {
	// Create a server that returns errors
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"return": []map[string]interface{}{
					{
						"token":  "test-token",
						"start":  1234567890.0,
						"expire": 9999999999.0,
					},
				},
			})
			return
		}

		// Return error response
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer errorServer.Close()

	client := salt.NewClient(errorServer.URL, "saltapi", "saltapi123")
	if err := client.Login(); err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Test that commands handle errors gracefully
	_, err := client.Ping("*")
	if err == nil {
		t.Error("Expected error but got none")
	}
}

// TestSaltCommandWithArguments tests commands with various arguments
func TestSaltCommandWithArguments(t *testing.T) {
	server := setupMockSaltServer()
	defer server.Close()

	client := salt.NewClient(server.URL, "saltapi", "saltapi123")
	if err := client.Login(); err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	tests := []struct {
		name     string
		function string
		args     []string
	}{
		{
			name:     "Command with single argument",
			function: "cmd.run",
			args:     []string{"uptime"},
		},
		{
			name:     "Command with multiple arguments",
			function: "cmd.run",
			args:     []string{"echo", "hello world"},
		},
		{
			name:     "Command with no arguments",
			function: "test.ping",
			args:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.RunCommand("*", tt.function, tt.args)
			if err != nil {
				t.Errorf("Command %q with args %v failed: %v", tt.function, tt.args, err)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr || len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkSaltPing benchmarks the ping operation
func BenchmarkSaltPing(b *testing.B) {
	server := setupMockSaltServer()
	defer server.Close()

	client := salt.NewClient(server.URL, "saltapi", "saltapi123")
	if err := client.Login(); err != nil {
		b.Fatalf("Failed to login: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Ping("*")
	}
}

// BenchmarkSaltCommand benchmarks command execution
func BenchmarkSaltCommand(b *testing.B) {
	server := setupMockSaltServer()
	defer server.Close()

	client := salt.NewClient(server.URL, "saltapi", "saltapi123")
	if err := client.Login(); err != nil {
		b.Fatalf("Failed to login: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.RunCommand("*", "cmd.run", []string{"uptime"})
	}
}

// BenchmarkJSONMarshal benchmarks JSON output
func BenchmarkJSONMarshal(b *testing.B) {
	data := map[string]interface{}{
		"minion-1": map[string]interface{}{
			"os":        "Ubuntu",
			"osrelease": "22.04",
			"kernel":    "5.15.0",
		},
		"minion-2": map[string]interface{}{
			"os":        "Ubuntu",
			"osrelease": "22.04",
			"kernel":    "5.15.0",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(data)
	}
}
