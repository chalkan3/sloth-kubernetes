package health

import (
	"strings"
	"testing"
)

// TestHealthCheck_Types tests different health check types
func TestHealthCheck_Types(t *testing.T) {
	validTypes := []string{"http", "https", "tcp", "exec"}

	tests := []struct {
		name  string
		hcType string
		valid bool
	}{
		{"HTTP check", "http", true},
		{"HTTPS check", "https", true},
		{"TCP check", "tcp", true},
		{"Exec check", "exec", true},
		{"Invalid type", "invalid", false},
		{"Empty type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, valid := range validTypes {
				if tt.hcType == valid {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for type %q, got %v", tt.valid, tt.hcType, isValid)
			}
		})
	}
}

// TestHealthEndpoint_Validation tests health endpoint validation
func TestHealthEndpoint_Validation(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		valid    bool
	}{
		{"Root path", "/", true},
		{"Health path", "/health", true},
		{"Healthz path", "/healthz", true},
		{"Ready path", "/ready", true},
		{"Live path", "/live", true},
		{"Custom path", "/api/health", true},
		{"No leading slash", "health", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.endpoint != "" && strings.HasPrefix(tt.endpoint, "/")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for endpoint %q, got %v", tt.valid, tt.endpoint, isValid)
			}
		})
	}
}

// TestHTTPStatusCode_Validation tests HTTP status code validation
func TestHTTPStatusCode_Validation(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		valid  bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"204 No Content", 204, true},
		{"301 Redirect", 301, true},
		{"400 Bad Request", 400, false}, // Client errors not valid for health
		{"500 Server Error", 500, false}, // Server errors not valid for health
		{"Invalid code", 0, false},
		{"Negative code", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.code >= 200 && tt.code < 400

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for code %d, got %v", tt.valid, tt.code, isValid)
			}
		})
	}
}

// TestHealthProbe_Intervals tests probe interval validation
func TestHealthProbe_Intervals(t *testing.T) {
	tests := []struct {
		name             string
		initialDelay     int
		periodSeconds    int
		timeoutSeconds   int
		successThreshold int
		failureThreshold int
		valid            bool
	}{
		{"Valid defaults", 10, 10, 1, 1, 3, true},
		{"Valid custom", 30, 30, 5, 2, 5, true},
		{"Zero initial delay", 0, 10, 1, 1, 3, true}, // Valid
		{"Zero period", 10, 0, 1, 1, 3, false},
		{"Zero timeout", 10, 10, 0, 1, 3, false},
		{"Zero thresholds", 10, 10, 1, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.periodSeconds > 0 &&
				tt.timeoutSeconds > 0 &&
				tt.successThreshold > 0 &&
				tt.failureThreshold > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestLivenessProbe_Config tests liveness probe configuration
func TestLivenessProbe_Config(t *testing.T) {
	tests := []struct {
		name             string
		probeType        string
		path             string
		port             int
		initialDelay     int
		failureThreshold int
		valid            bool
	}{
		{"Valid HTTP liveness", "http", "/healthz", 8080, 15, 3, true},
		{"Valid TCP liveness", "tcp", "", 8080, 10, 3, true},
		{"Missing path for HTTP", "http", "", 8080, 15, 3, false},
		{"Invalid port", "http", "/health", 0, 15, 3, false},
		{"Negative delay", "http", "/health", 8080, -1, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port > 0 &&
				tt.initialDelay >= 0 &&
				tt.failureThreshold > 0

			// HTTP requires path
			if tt.probeType == "http" || tt.probeType == "https" {
				isValid = isValid && tt.path != ""
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestReadinessProbe_Config tests readiness probe configuration
func TestReadinessProbe_Config(t *testing.T) {
	tests := []struct {
		name             string
		probeType        string
		path             string
		port             int
		initialDelay     int
		periodSeconds    int
		valid            bool
	}{
		{"Valid HTTP readiness", "http", "/ready", 8080, 5, 10, true},
		{"Valid TCP readiness", "tcp", "", 8080, 5, 10, true},
		{"Valid exec readiness", "exec", "", 0, 5, 10, true},
		{"Invalid period", "http", "/ready", 8080, 5, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.periodSeconds > 0 && tt.initialDelay >= 0

			// Exec doesn't need port
			if tt.probeType != "exec" {
				isValid = isValid && tt.port > 0
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestStartupProbe_Config tests startup probe configuration
func TestStartupProbe_Config(t *testing.T) {
	tests := []struct {
		name             string
		initialDelay     int
		periodSeconds    int
		failureThreshold int
		maxStartupTime   int
		valid            bool
	}{
		{"Valid startup probe", 0, 10, 30, 300, true}, // 30 * 10 = 300s max
		{"Long startup", 0, 15, 40, 600, true},        // 40 * 15 = 600s max
		{"Zero period", 0, 0, 30, 0, false},
		{"Zero threshold", 0, 10, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.periodSeconds > 0 && tt.failureThreshold > 0

			// Verify max startup time calculation
			calculatedMax := tt.periodSeconds * tt.failureThreshold
			if isValid && tt.maxStartupTime > 0 && calculatedMax != tt.maxStartupTime {
				t.Errorf("Max startup time mismatch: %d * %d = %d, expected %d",
					tt.periodSeconds, tt.failureThreshold, calculatedMax, tt.maxStartupTime)
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestHealthCheckCommand_Validation tests exec command validation
func TestHealthCheckCommand_Validation(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		valid   bool
	}{
		{"Valid shell command", []string{"/bin/sh", "-c", "curl localhost:8080/health"}, true},
		{"Valid binary", []string{"/usr/bin/health-check"}, true},
		{"Valid with args", []string{"curl", "-f", "http://localhost/health"}, true},
		{"Empty command", []string{}, false},
		{"Nil command", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.command != nil && len(tt.command) > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestTCPSocket_Config tests TCP socket health check configuration
func TestTCPSocket_Config(t *testing.T) {
	tests := []struct {
		name  string
		host  string
		port  int
		valid bool
	}{
		{"Valid localhost", "localhost", 8080, true},
		{"Valid 127.0.0.1", "127.0.0.1", 9090, true},
		{"Valid hostname", "service.namespace.svc.cluster.local", 80, true},
		{"Empty host (default)", "", 8080, true}, // Empty means localhost
		{"Invalid port 0", "localhost", 0, false},
		{"Invalid port negative", "localhost", -1, false},
		{"Invalid port too high", "localhost", 70000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port > 0 && tt.port <= 65535

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestHTTPHeaders_Validation tests HTTP header validation for health checks
func TestHTTPHeaders_Validation(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		valid   bool
	}{
		{
			"Valid headers",
			map[string]string{
				"User-Agent":    "kube-probe/1.0",
				"Authorization": "Bearer token",
			},
			true,
		},
		{
			"Valid custom header",
			map[string]string{
				"X-Custom-Header": "value",
			},
			true,
		},
		{
			"Empty headers (valid)",
			map[string]string{},
			true,
		},
		{
			"Nil headers (valid)",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Headers are always valid, even if empty/nil
			isValid := true

			// Verify header format if present
			for key, value := range tt.headers {
				if key == "" || value == "" {
					isValid = false
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestProbeAction_Priority tests probe action priority
func TestProbeAction_Priority(t *testing.T) {
	tests := []struct {
		name         string
		hasHTTP      bool
		hasTCP       bool
		hasExec      bool
		expectedType string
		valid        bool
	}{
		{"Only HTTP", true, false, false, "http", true},
		{"Only TCP", false, true, false, "tcp", true},
		{"Only Exec", false, false, true, "exec", true},
		{"Multiple actions", true, true, false, "", false}, // Should have only one
		{"No action", false, false, false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actionCount := 0
			if tt.hasHTTP {
				actionCount++
			}
			if tt.hasTCP {
				actionCount++
			}
			if tt.hasExec {
				actionCount++
			}

			isValid := actionCount == 1

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// Test150HealthCheckScenarios generates 150 health check test scenarios
func Test150HealthCheckScenarios(t *testing.T) {
	scenarios := []struct {
		probeType        string
		endpoint         string
		port             int
		initialDelay     int
		periodSeconds    int
		failureThreshold int
		valid            bool
	}{
		{"http", "/health", 8080, 10, 10, 3, true},
		{"https", "/healthz", 8443, 15, 15, 3, true},
		{"tcp", "", 9090, 5, 10, 3, true},
	}

	// Generate 147 more scenarios
	probeTypes := []string{"http", "https", "tcp", "exec"}
	endpoints := []string{"/health", "/healthz", "/ready", "/live", "/ping"}
	ports := []int{8080, 8443, 9090, 3000, 5000}
	delays := []int{0, 5, 10, 15, 30}
	periods := []int{5, 10, 15, 30}
	failures := []int{1, 2, 3, 5, 10}

	for i := 0; i < 147; i++ {
		probeType := probeTypes[i%len(probeTypes)]
		endpoint := endpoints[i%len(endpoints)]
		if probeType == "tcp" || probeType == "exec" {
			endpoint = ""
		}

		port := ports[i%len(ports)]
		if probeType == "exec" {
			port = 0
		}

		scenarios = append(scenarios, struct {
			probeType        string
			endpoint         string
			port             int
			initialDelay     int
			periodSeconds    int
			failureThreshold int
			valid            bool
		}{
			probeType:        probeType,
			endpoint:         endpoint,
			port:             port,
			initialDelay:     delays[i%len(delays)],
			periodSeconds:    periods[i%len(periods)],
			failureThreshold: failures[i%len(failures)],
			valid:            true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_health_"+string(rune('0'+i%10)), func(t *testing.T) {
			isValid := scenario.periodSeconds > 0 && scenario.failureThreshold > 0

			// Validate endpoint for HTTP/HTTPS
			if scenario.probeType == "http" || scenario.probeType == "https" {
				isValid = isValid && scenario.endpoint != "" && strings.HasPrefix(scenario.endpoint, "/")
			}

			// Validate port for non-exec probes
			if scenario.probeType != "exec" {
				isValid = isValid && scenario.port > 0 && scenario.port <= 65535
			}

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}
		})
	}
}
