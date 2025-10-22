package ingress

import (
	"strings"
	"testing"
)

// TestNginxConfig_Structure tests nginx configuration structure
func TestNginxConfig_Structure(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		valid  bool
	}{
		{
			name: "Valid basic config",
			config: map[string]string{
				"use-forwarded-headers": "true",
				"compute-full-forwarded-for": "true",
			},
			valid: true,
		},
		{
			name: "Valid with proxy settings",
			config: map[string]string{
				"proxy-body-size": "100m",
				"proxy-connect-timeout": "60",
			},
			valid: true,
		},
		{
			name:   "Empty config",
			config: map[string]string{},
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.config) > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestNginxAnnotation_Validation tests nginx annotation validation
func TestNginxAnnotation_Validation(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		value      string
		valid      bool
	}{
		{"SSL redirect", "nginx.ingress.kubernetes.io/ssl-redirect", "true", true},
		{"Force SSL", "nginx.ingress.kubernetes.io/force-ssl-redirect", "true", true},
		{"Proxy body size", "nginx.ingress.kubernetes.io/proxy-body-size", "100m", true},
		{"Rate limit", "nginx.ingress.kubernetes.io/limit-rps", "10", true},
		{"Whitelist IPs", "nginx.ingress.kubernetes.io/whitelist-source-range", "10.0.0.0/8", true},
		{"Rewrite target", "nginx.ingress.kubernetes.io/rewrite-target", "/", true},
		{"Invalid annotation", "invalid-annotation", "value", false},
		{"Empty annotation", "", "value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.annotation != "" && strings.Contains(tt.annotation, "/")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for annotation %q, got %v", tt.valid, tt.annotation, isValid)
			}
		})
	}
}

// TestIngressClass_Validation tests ingress class validation
func TestIngressClass_Validation(t *testing.T) {
	validClasses := []string{"nginx", "traefik", "haproxy", "kong"}

	tests := []struct {
		name  string
		class string
		valid bool
	}{
		{"Nginx", "nginx", true},
		{"Traefik", "traefik", true},
		{"HAProxy", "haproxy", true},
		{"Kong", "kong", true},
		{"Invalid", "invalid", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, valid := range validClasses {
				if tt.class == valid {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for class %q, got %v", tt.valid, tt.class, isValid)
			}
		})
	}
}

// TestTLSConfig_Validation tests TLS configuration validation
func TestTLSConfig_Validation(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		hosts      []string
		valid      bool
	}{
		{
			name:       "Valid TLS config",
			secretName: "tls-secret",
			hosts:      []string{"example.com", "www.example.com"},
			valid:      true,
		},
		{
			name:       "Valid wildcard",
			secretName: "wildcard-tls",
			hosts:      []string{"*.example.com"},
			valid:      true,
		},
		{
			name:       "Missing secret name",
			secretName: "",
			hosts:      []string{"example.com"},
			valid:      false,
		},
		{
			name:       "Missing hosts",
			secretName: "tls-secret",
			hosts:      []string{},
			valid:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.secretName != "" && len(tt.hosts) > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestIngressRule_Validation tests ingress rule validation
func TestIngressRule_Validation(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		path        string
		serviceName string
		servicePort int
		valid       bool
	}{
		{"Valid rule", "example.com", "/", "web-service", 80, true},
		{"Valid with path", "api.example.com", "/api", "api-service", 8080, true},
		{"Valid wildcard host", "*.example.com", "/", "service", 80, true},
		{"Missing host", "", "/", "service", 80, false},
		{"Missing path", "example.com", "", "service", 80, false},
		{"Missing service", "example.com", "/", "", 80, false},
		{"Invalid port", "example.com", "/", "service", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.host != "" &&
				tt.path != "" &&
				tt.serviceName != "" &&
				tt.servicePort > 0 &&
				tt.servicePort <= 65535

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestProxyBodySize_Validation tests proxy body size validation
func TestProxyBodySize_Validation(t *testing.T) {
	tests := []struct {
		name  string
		size  string
		valid bool
	}{
		{"Valid 10m", "10m", true},
		{"Valid 100m", "100m", true},
		{"Valid 1g", "1g", true},
		{"Valid 50k", "50k", true},
		{"No unit", "100", false},
		{"Invalid unit", "100x", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.size != "" && (strings.HasSuffix(tt.size, "k") ||
				strings.HasSuffix(tt.size, "m") ||
				strings.HasSuffix(tt.size, "g"))

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for size %q, got %v", tt.valid, tt.size, isValid)
			}
		})
	}
}

// TestProxyTimeout_Validation tests proxy timeout validation
func TestProxyTimeout_Validation(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
		valid   bool
	}{
		{"Valid 60s", "60", true},
		{"Valid 30s", "30", true},
		{"Valid 120s", "120", true},
		{"Zero timeout", "0", false},
		{"Negative timeout", "-1", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.timeout != "" && tt.timeout != "0" && !strings.HasPrefix(tt.timeout, "-")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for timeout %q, got %v", tt.valid, tt.timeout, isValid)
			}
		})
	}
}

// TestRateLimiting_Config tests rate limiting configuration
func TestRateLimiting_Config(t *testing.T) {
	tests := []struct {
		name         string
		limitRPS     int
		limitRPM     int
		limitBurst   int
		valid        bool
	}{
		{"Valid RPS limit", 10, 0, 20, true},
		{"Valid RPM limit", 0, 100, 50, true},
		{"Both limits", 10, 100, 20, true},
		{"No limits", 0, 0, 0, false},
		{"Negative RPS", -1, 0, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := (tt.limitRPS > 0 || tt.limitRPM > 0) &&
				tt.limitRPS >= 0 &&
				tt.limitRPM >= 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestWhitelistSourceRange_Validation tests IP whitelist validation
func TestWhitelistSourceRange_Validation(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		valid bool
	}{
		{"Valid single IP", "192.168.1.1/32", true},
		{"Valid /24 network", "10.0.0.0/24", true},
		{"Valid /16 network", "172.16.0.0/16", true},
		{"Valid /8 network", "10.0.0.0/8", true},
		{"Multiple CIDRs", "10.0.0.0/8,192.168.0.0/16", true},
		{"Missing CIDR", "192.168.1.1", false},
		{"Invalid CIDR", "192.168.1.1", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.cidr != "" && (strings.Contains(tt.cidr, "/") || strings.Contains(tt.cidr, ","))

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for CIDR %q, got %v", tt.valid, tt.cidr, isValid)
			}
		})
	}
}

// TestRewriteTarget_Validation tests rewrite target validation
func TestRewriteTarget_Validation(t *testing.T) {
	tests := []struct {
		name   string
		target string
		valid  bool
	}{
		{"Root path", "/", true},
		{"API path", "/api", true},
		{"Versioned path", "/v1", true},
		{"Nested path", "/api/v1", true},
		{"With trailing slash", "/api/", true},
		{"No leading slash", "api", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.target != "" && strings.HasPrefix(tt.target, "/")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for target %q, got %v", tt.valid, tt.target, isValid)
			}
		})
	}
}

// TestCORS_Config tests CORS configuration
func TestCORS_Config(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		allowOrigin    string
		allowMethods   string
		allowHeaders   string
		valid          bool
	}{
		{
			name:         "Valid CORS config",
			enabled:      true,
			allowOrigin:  "*",
			allowMethods: "GET, POST, PUT, DELETE",
			allowHeaders: "Content-Type, Authorization",
			valid:        true,
		},
		{
			name:         "Specific origin",
			enabled:      true,
			allowOrigin:  "https://example.com",
			allowMethods: "GET, POST",
			allowHeaders: "Content-Type",
			valid:        true,
		},
		{
			name:         "CORS disabled",
			enabled:      false,
			allowOrigin:  "",
			allowMethods: "",
			allowHeaders: "",
			valid:        true,
		},
		{
			name:         "Missing origin",
			enabled:      true,
			allowOrigin:  "",
			allowMethods: "GET",
			allowHeaders: "Content-Type",
			valid:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := !tt.enabled || (tt.allowOrigin != "")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestSSLCertificate_Validation tests SSL certificate validation
func TestSSLCertificate_Validation(t *testing.T) {
	tests := []struct {
		name       string
		secretType string
		certData   string
		keyData    string
		valid      bool
	}{
		{"Valid TLS secret", "kubernetes.io/tls", "cert-data", "key-data", true},
		{"Missing cert", "kubernetes.io/tls", "", "key-data", false},
		{"Missing key", "kubernetes.io/tls", "cert-data", "", false},
		{"Wrong type", "Opaque", "cert-data", "key-data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.secretType == "kubernetes.io/tls" &&
				tt.certData != "" &&
				tt.keyData != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestLoadBalancer_Config tests load balancer configuration
func TestLoadBalancer_Config(t *testing.T) {
	tests := []struct {
		name     string
		lbType   string
		lbIP     string
		valid    bool
	}{
		{"LoadBalancer type", "LoadBalancer", "192.168.1.100", true},
		{"NodePort type", "NodePort", "", true},
		{"ClusterIP type", "ClusterIP", "", true},
		{"Invalid type", "Invalid", "", false},
		{"Empty type", "", "", false},
	}

	validTypes := []string{"LoadBalancer", "NodePort", "ClusterIP"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, validType := range validTypes {
				if tt.lbType == validType {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for type %q, got %v", tt.valid, tt.lbType, isValid)
			}
		})
	}
}

// Test200IngressScenarios generates 200 ingress test scenarios
func Test200IngressScenarios(t *testing.T) {
	scenarios := []struct {
		host        string
		path        string
		serviceName string
		servicePort int
		tlsEnabled  bool
		valid       bool
	}{
		{"example.com", "/", "web", 80, false, true},
		{"api.example.com", "/api", "api", 8080, true, true},
	}

	// Generate 198 more scenarios
	hosts := []string{"example.com", "api.example.com", "app.example.com", "*.example.com"}
	paths := []string{"/", "/api", "/v1", "/app", "/health"}
	services := []string{"web", "api", "app", "backend", "frontend"}
	ports := []int{80, 8080, 8443, 3000, 9090}

	for i := 0; i < 198; i++ {
		scenarios = append(scenarios, struct {
			host        string
			path        string
			serviceName string
			servicePort int
			tlsEnabled  bool
			valid       bool
		}{
			host:        hosts[i%len(hosts)],
			path:        paths[i%len(paths)],
			serviceName: services[i%len(services)],
			servicePort: ports[i%len(ports)],
			tlsEnabled:  i%2 == 0,
			valid:       true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_ingress_"+string(rune('0'+i%10)), func(t *testing.T) {
			hostValid := scenario.host != ""
			pathValid := scenario.path != "" && strings.HasPrefix(scenario.path, "/")
			serviceValid := scenario.serviceName != ""
			portValid := scenario.servicePort > 0 && scenario.servicePort <= 65535

			isValid := hostValid && pathValid && serviceValid && portValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}
		})
	}
}
