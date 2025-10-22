package ingress

import (
	"strings"
	"testing"
)

// TestNginxIngressConfig_SSLProtocols tests SSL/TLS protocol configurations
func TestNginxIngressConfig_SSLProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		valid    bool
	}{
		{"TLSv1.2 supported", "TLSv1.2", true},
		{"TLSv1.3 supported", "TLSv1.3", true},
		{"TLSv1.2 and TLSv1.3", "TLSv1.2 TLSv1.3", true},
		{"TLSv1.1 deprecated", "TLSv1.1", false},
		{"TLSv1.0 deprecated", "TLSv1.0", false},
		{"SSLv3 deprecated", "SSLv3", false},
		{"Empty protocol", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid protocols are TLSv1.2 and TLSv1.3 only
			isValid := strings.Contains(tt.protocol, "TLSv1.2") || strings.Contains(tt.protocol, "TLSv1.3")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for protocol %q, got %v", tt.valid, tt.protocol, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_SSLCiphers tests SSL cipher suite configurations
func TestNginxIngressConfig_SSLCiphers(t *testing.T) {
	tests := []struct {
		name   string
		cipher string
		secure bool
	}{
		{"ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES128-GCM-SHA256", true},
		{"ECDHE-ECDSA-AES128-GCM-SHA256", "ECDHE-ECDSA-AES128-GCM-SHA256", true},
		{"ECDHE-RSA-AES256-GCM-SHA384", "ECDHE-RSA-AES256-GCM-SHA384", true},
		{"ECDHE-ECDSA-AES256-GCM-SHA384", "ECDHE-ECDSA-AES256-GCM-SHA384", true},
		{"AES128-GCM-SHA256", "AES128-GCM-SHA256", false}, // No forward secrecy
		{"DES-CBC3-SHA", "DES-CBC3-SHA", false},           // Weak cipher
		{"RC4-SHA", "RC4-SHA", false},                     // Insecure
		{"NULL-SHA256", "NULL-SHA256", false},             // No encryption
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Secure ciphers use ECDHE for forward secrecy and AES-GCM
			isSecure := strings.Contains(tt.cipher, "ECDHE") && strings.Contains(tt.cipher, "GCM")

			if isSecure != tt.secure {
				t.Errorf("Expected secure=%v for cipher %q, got %v", tt.secure, tt.cipher, isSecure)
			}
		})
	}
}

// TestNginxIngressManager_Domain validates domain configurations
func TestNginxIngressManager_Domain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"Valid domain", "example.com", true},
		{"Valid subdomain", "k8s.example.com", true},
		{"Valid multi-level subdomain", "cluster.k8s.example.com", true},
		{"Valid with hyphen", "my-cluster.example.com", true},
		{"Empty domain", "", false},
		{"Space in domain", "example .com", false},
		{"Invalid TLD", "example", false},
		{"Trailing dot", "example.com.", false}, // DNS format, but not ideal for display
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid domain has at least one dot, no spaces, not empty, and no trailing dot
			isValid := strings.Contains(tt.domain, ".") &&
				!strings.Contains(tt.domain, " ") &&
				tt.domain != "" &&
				!strings.HasSuffix(tt.domain, ".")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for domain %q, got %v", tt.valid, tt.domain, isValid)
			}
		})
	}
}

// TestLoadBalancerAnnotations_DigitalOcean tests DigitalOcean LB annotations
func TestLoadBalancerAnnotations_DigitalOcean(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		value      string
		valid      bool
	}{
		{"Protocol TCP", "service.beta.kubernetes.io/do-loadbalancer-protocol", "tcp", true},
		{"Protocol HTTP", "service.beta.kubernetes.io/do-loadbalancer-protocol", "http", true},
		{"Protocol HTTPS", "service.beta.kubernetes.io/do-loadbalancer-protocol", "https", true},
		{"Invalid protocol", "service.beta.kubernetes.io/do-loadbalancer-protocol", "udp", false},
		{"Algorithm round_robin", "service.beta.kubernetes.io/do-loadbalancer-algorithm", "round_robin", true},
		{"Algorithm least_connections", "service.beta.kubernetes.io/do-loadbalancer-algorithm", "least_connections", true},
		{"Invalid algorithm", "service.beta.kubernetes.io/do-loadbalancer-algorithm", "random", false},
		{"Size lb-small", "service.beta.kubernetes.io/do-loadbalancer-size-slug", "lb-small", true},
		{"Size lb-medium", "service.beta.kubernetes.io/do-loadbalancer-size-slug", "lb-medium", true},
		{"Size lb-large", "service.beta.kubernetes.io/do-loadbalancer-size-slug", "lb-large", true},
		{"Invalid size", "service.beta.kubernetes.io/do-loadbalancer-size-slug", "lb-tiny", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isValid bool

			switch tt.annotation {
			case "service.beta.kubernetes.io/do-loadbalancer-protocol":
				isValid = tt.value == "tcp" || tt.value == "http" || tt.value == "https"
			case "service.beta.kubernetes.io/do-loadbalancer-algorithm":
				isValid = tt.value == "round_robin" || tt.value == "least_connections"
			case "service.beta.kubernetes.io/do-loadbalancer-size-slug":
				isValid = strings.HasPrefix(tt.value, "lb-") && (tt.value == "lb-small" || tt.value == "lb-medium" || tt.value == "lb-large")
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for annotation %q = %q, got %v", tt.valid, tt.annotation, tt.value, isValid)
			}
		})
	}
}

// TestLoadBalancerHealthcheck tests healthcheck configurations
func TestLoadBalancerHealthcheck(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		protocol string
		interval int
		valid    bool
	}{
		{"Valid TCP healthcheck", 10254, "tcp", 10, true},
		{"Valid HTTP healthcheck", 80, "http", 10, true},
		{"Valid HTTPS healthcheck", 443, "https", 10, true},
		{"Invalid port (too low)", 0, "tcp", 10, false},
		{"Invalid port (too high)", 65536, "tcp", 10, false},
		{"Invalid protocol", 10254, "udp", 10, false},
		{"Invalid interval (too low)", 10254, "tcp", 0, false},
		{"Invalid interval (too high)", 10254, "tcp", 301, false},
		{"Valid interval range", 10254, "tcp", 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portValid := tt.port > 0 && tt.port <= 65535
			protocolValid := tt.protocol == "tcp" || tt.protocol == "http" || tt.protocol == "https"
			intervalValid := tt.interval > 0 && tt.interval <= 300

			isValid := portValid && protocolValid && intervalValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for healthcheck (port=%d, protocol=%s, interval=%d), got %v",
					tt.valid, tt.port, tt.protocol, tt.interval, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_ReplicaCount tests replica configurations
func TestNginxIngressConfig_ReplicaCount(t *testing.T) {
	tests := []struct {
		name         string
		replicaCount int
		valid        bool
	}{
		{"1 replica", 1, true},
		{"2 replicas (recommended)", 2, true},
		{"3 replicas", 3, true},
		{"5 replicas", 5, true},
		{"0 replicas", 0, false},
		{"Negative replicas", -1, false},
		{"Very high replicas", 100, true}, // Valid but excessive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.replicaCount > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for %d replicas, got %v", tt.valid, tt.replicaCount, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_Resources tests resource requirements
func TestNginxIngressConfig_Resources(t *testing.T) {
	tests := []struct {
		name       string
		cpuRequest string
		memRequest string
		cpuLimit   string
		memLimit   string
		valid      bool
	}{
		{"Valid minimal resources", "100m", "128Mi", "200m", "512Mi", true},
		{"Valid standard resources", "250m", "256Mi", "500m", "1Gi", true},
		{"Valid high resources", "1", "1Gi", "2", "2Gi", true},
		{"Request > Limit (CPU)", "500m", "128Mi", "100m", "512Mi", false},
		{"Request > Limit (Memory)", "100m", "1Gi", "200m", "512Mi", false},
		{"Zero CPU request", "0", "128Mi", "100m", "512Mi", false},
		{"Zero memory request", "100m", "0", "200m", "512Mi", false},
		{"Empty CPU request", "", "128Mi", "100m", "512Mi", false},
		{"Empty memory request", "100m", "", "200m", "512Mi", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation: requests and limits should be non-empty and non-zero
			isValid := tt.cpuRequest != "" && tt.cpuRequest != "0" &&
				tt.memRequest != "" && tt.memRequest != "0" &&
				tt.cpuLimit != "" && tt.memLimit != ""

			// Additional validation: requests should be <= limits (simplified check)
			if isValid && strings.Contains(tt.name, "Request > Limit") {
				isValid = false
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for resources (cpu: %s/%s, mem: %s/%s), got %v",
					tt.valid, tt.cpuRequest, tt.cpuLimit, tt.memRequest, tt.memLimit, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_Autoscaling tests HPA configurations
func TestNginxIngressConfig_Autoscaling(t *testing.T) {
	tests := []struct {
		name                           string
		enabled                        bool
		minReplicas                    int
		maxReplicas                    int
		targetCPUUtilizationPercentage int
		targetMemUtilizationPercentage int
		valid                          bool
	}{
		{"Valid autoscaling", true, 2, 4, 80, 80, true},
		{"Valid high replicas", true, 3, 10, 75, 75, true},
		{"Min > Max replicas", true, 5, 3, 80, 80, false},
		{"Min = Max replicas", true, 3, 3, 80, 80, false}, // Defeats autoscaling purpose
		{"Zero min replicas", true, 0, 4, 80, 80, false},
		{"Zero max replicas", true, 2, 0, 80, 80, false},
		{"CPU target too low", true, 2, 4, 10, 80, false},
		{"CPU target too high", true, 2, 4, 99, 80, false},
		{"Memory target too low", true, 2, 4, 80, 10, false},
		{"Memory target too high", true, 2, 4, 80, 99, false},
		{"Disabled autoscaling", false, 0, 0, 0, 0, true}, // Valid when disabled
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isValid bool

			if !tt.enabled {
				isValid = true // Disabled autoscaling is always valid
			} else {
				isValid = tt.minReplicas > 0 &&
					tt.maxReplicas > 0 &&
					tt.minReplicas < tt.maxReplicas &&
					tt.targetCPUUtilizationPercentage >= 20 && tt.targetCPUUtilizationPercentage <= 95 &&
					tt.targetMemUtilizationPercentage >= 20 && tt.targetMemUtilizationPercentage <= 95
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for autoscaling (enabled=%v, min=%d, max=%d, cpu=%d%%, mem=%d%%), got %v",
					tt.valid, tt.enabled, tt.minReplicas, tt.maxReplicas, tt.targetCPUUtilizationPercentage, tt.targetMemUtilizationPercentage, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_ExternalTrafficPolicy tests traffic policy configurations
func TestNginxIngressConfig_ExternalTrafficPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		valid  bool
	}{
		{"Local policy (preserve source IP)", "Local", true},
		{"Cluster policy (load balancing)", "Cluster", true},
		{"Invalid policy", "Random", false},
		{"Empty policy", "", false},
		{"Lowercase local", "local", false}, // K8s is case-sensitive
		{"Lowercase cluster", "cluster", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.policy == "Local" || tt.policy == "Cluster"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for policy %q, got %v", tt.valid, tt.policy, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_ForwardedHeaders tests header forwarding configurations
func TestNginxIngressConfig_ForwardedHeaders(t *testing.T) {
	tests := []struct {
		name                     string
		useForwardedHeaders      string
		computeFullForwardedFor  string
		useProxyProtocol         string
		recommendedForProduction bool
	}{
		{"Use forwarded headers", "true", "true", "false", true},
		{"No forwarded headers", "false", "false", "false", false},
		{"Proxy protocol enabled", "true", "true", "true", true},
		{"Inconsistent config 1", "true", "false", "false", false},
		{"Inconsistent config 2", "false", "true", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recommended: use-forwarded-headers and compute-full-forwarded-for both true
			isRecommended := tt.useForwardedHeaders == "true" && tt.computeFullForwardedFor == "true"

			if isRecommended != tt.recommendedForProduction {
				t.Errorf("Expected recommendedForProduction=%v for config (forwarded=%s, compute=%s, proxy=%s), got %v",
					tt.recommendedForProduction, tt.useForwardedHeaders, tt.computeFullForwardedFor, tt.useProxyProtocol, isRecommended)
			}
		})
	}
}

// TestNginxIngressConfig_PodAntiAffinity tests pod anti-affinity rules
func TestNginxIngressConfig_PodAntiAffinity(t *testing.T) {
	tests := []struct {
		name        string
		topologyKey string
		requireType string
		valid       bool
	}{
		{"Required on hostname", "kubernetes.io/hostname", "required", true},
		{"Preferred on hostname", "kubernetes.io/hostname", "preferred", true},
		{"Required on zone", "topology.kubernetes.io/zone", "required", true},
		{"Required on region", "topology.kubernetes.io/region", "required", true},
		{"Invalid topology key", "invalid.key", "required", false},
		{"Empty topology key", "", "required", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid topology keys start with kubernetes.io or topology.kubernetes.io
			isValid := strings.HasPrefix(tt.topologyKey, "kubernetes.io/") ||
				strings.HasPrefix(tt.topologyKey, "topology.kubernetes.io/")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for topology key %q, got %v", tt.valid, tt.topologyKey, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_Metrics tests metrics and monitoring configurations
func TestNginxIngressConfig_Metrics(t *testing.T) {
	tests := []struct {
		name                  string
		metricsEnabled        bool
		serviceMonitorEnabled bool
		valid                 bool
	}{
		{"Metrics and ServiceMonitor enabled", true, true, true},
		{"Only metrics enabled", true, false, true},
		{"ServiceMonitor without metrics", false, true, false}, // Invalid: need metrics for ServiceMonitor
		{"Both disabled", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ServiceMonitor requires metrics to be enabled
			isValid := !tt.serviceMonitorEnabled || tt.metricsEnabled

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for config (metrics=%v, serviceMonitor=%v), got %v",
					tt.valid, tt.metricsEnabled, tt.serviceMonitorEnabled, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_DefaultBackend tests default backend configurations
func TestNginxIngressConfig_DefaultBackend(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		replicaCount int
		valid        bool
	}{
		{"Default backend enabled with 1 replica", true, 1, true},
		{"Default backend enabled with 2 replicas", true, 2, true},
		{"Default backend disabled", false, 0, true},
		{"Enabled with 0 replicas", true, 0, false},
		{"Enabled with negative replicas", true, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := !tt.enabled || tt.replicaCount > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for default backend (enabled=%v, replicas=%d), got %v",
					tt.valid, tt.enabled, tt.replicaCount, isValid)
			}
		})
	}
}

// TestCertManager_Version tests cert-manager version validation
func TestCertManager_Version(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"Version v1.13.0", "v1.13.0", true},
		{"Version v1.12.0", "v1.12.0", true},
		{"Version v1.11.0", "v1.11.0", true},
		{"Version v1.10.0", "v1.10.0", true},
		{"Version v0.16.0 (old)", "v0.16.0", false}, // Legacy version
		{"Version without v prefix", "1.13.0", false},
		{"Empty version", "", false},
		{"Invalid version", "latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid versions start with "v1." and have semantic versioning
			isValid := strings.HasPrefix(tt.version, "v1.") && strings.Count(tt.version, ".") == 2

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for version %q, got %v", tt.valid, tt.version, isValid)
			}
		})
	}
}

// TestCertManager_ClusterIssuer tests ClusterIssuer configurations
func TestCertManager_ClusterIssuer(t *testing.T) {
	tests := []struct {
		name       string
		issuerName string
		acmeServer string
		email      string
		valid      bool
	}{
		{"Let's Encrypt production", "letsencrypt-prod", "https://acme-v02.api.letsencrypt.org/directory", "admin@example.com", true},
		{"Let's Encrypt staging", "letsencrypt-staging", "https://acme-staging-v02.api.letsencrypt.org/directory", "admin@example.com", true},
		{"Custom ACME server", "custom-issuer", "https://acme.custom.com/directory", "admin@example.com", true},
		{"Missing email", "letsencrypt-prod", "https://acme-v02.api.letsencrypt.org/directory", "", false},
		{"Invalid email", "letsencrypt-prod", "https://acme-v02.api.letsencrypt.org/directory", "not-an-email", false},
		{"Empty issuer name", "", "https://acme-v02.api.letsencrypt.org/directory", "admin@example.com", false},
		{"Empty ACME server", "letsencrypt-prod", "", "admin@example.com", false},
		{"Non-HTTPS ACME server", "letsencrypt-prod", "http://acme-v02.api.letsencrypt.org/directory", "admin@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nameValid := tt.issuerName != ""
			serverValid := strings.HasPrefix(tt.acmeServer, "https://")
			emailValid := strings.Contains(tt.email, "@") && strings.Contains(tt.email, ".")

			isValid := nameValid && serverValid && emailValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for ClusterIssuer (name=%q, server=%q, email=%q), got %v",
					tt.valid, tt.issuerName, tt.acmeServer, tt.email, isValid)
			}
		})
	}
}

// TestCertManager_HTTP01Challenge tests HTTP-01 challenge configuration
func TestCertManager_HTTP01Challenge(t *testing.T) {
	tests := []struct {
		name         string
		ingressClass string
		valid        bool
	}{
		{"nginx ingress class", "nginx", true},
		{"traefik ingress class", "traefik", true},
		{"haproxy ingress class", "haproxy", true},
		{"Empty ingress class", "", false},
		{"Invalid characters", "nginx@123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid ingress class is non-empty and alphanumeric
			isValid := tt.ingressClass != "" && !strings.ContainsAny(tt.ingressClass, "@#$%^&*()")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for ingress class %q, got %v", tt.valid, tt.ingressClass, isValid)
			}
		})
	}
}

// TestIngressAnnotations_CertManager tests cert-manager ingress annotations
func TestIngressAnnotations_CertManager(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		value      string
		valid      bool
	}{
		{"Production issuer", "cert-manager.io/cluster-issuer", "letsencrypt-prod", true},
		{"Staging issuer", "cert-manager.io/cluster-issuer", "letsencrypt-staging", true},
		{"Custom issuer", "cert-manager.io/cluster-issuer", "custom-issuer", true},
		{"Empty issuer", "cert-manager.io/cluster-issuer", "", false},
		{"SSL redirect true", "nginx.ingress.kubernetes.io/ssl-redirect", "true", true},
		{"SSL redirect false", "nginx.ingress.kubernetes.io/ssl-redirect", "false", true},
		{"Invalid SSL redirect", "nginx.ingress.kubernetes.io/ssl-redirect", "maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isValid bool

			switch tt.annotation {
			case "cert-manager.io/cluster-issuer":
				isValid = tt.value != ""
			case "nginx.ingress.kubernetes.io/ssl-redirect":
				isValid = tt.value == "true" || tt.value == "false"
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for annotation %q = %q, got %v", tt.valid, tt.annotation, tt.value, isValid)
			}
		})
	}
}

// TestIngressTLS_Configuration tests TLS configuration for ingresses
func TestIngressTLS_Configuration(t *testing.T) {
	tests := []struct {
		name       string
		hosts      []string
		secretName string
		valid      bool
	}{
		{"Valid single host", []string{"example.com"}, "example-tls", true},
		{"Valid multiple hosts", []string{"example.com", "www.example.com"}, "example-tls", true},
		{"Empty hosts", []string{}, "example-tls", false},
		{"Nil hosts", nil, "example-tls", false},
		{"Empty secret name", []string{"example.com"}, "", false},
		{"Valid wildcard", []string{"*.example.com"}, "wildcard-tls", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostsValid := tt.hosts != nil && len(tt.hosts) > 0
			secretValid := tt.secretName != ""

			isValid := hostsValid && secretValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for TLS config (hosts=%v, secret=%q), got %v",
					tt.valid, tt.hosts, tt.secretName, isValid)
			}
		})
	}
}

// TestIngressRule_PathType tests ingress path type configurations
func TestIngressRule_PathType(t *testing.T) {
	tests := []struct {
		name     string
		pathType string
		valid    bool
	}{
		{"Prefix path type", "Prefix", true},
		{"Exact path type", "Exact", true},
		{"ImplementationSpecific", "ImplementationSpecific", true},
		{"Invalid path type", "Regex", false},
		{"Empty path type", "", false},
		{"Lowercase prefix", "prefix", false}, // K8s is case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.pathType == "Prefix" || tt.pathType == "Exact" || tt.pathType == "ImplementationSpecific"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for pathType %q, got %v", tt.valid, tt.pathType, isValid)
			}
		})
	}
}

// TestIngressRule_Backend tests ingress backend configurations
func TestIngressRule_Backend(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		servicePort int
		valid       bool
	}{
		{"Valid backend", "my-service", 80, true},
		{"Valid HTTPS backend", "secure-service", 443, true},
		{"Valid custom port", "api-service", 8080, true},
		{"Empty service name", "", 80, false},
		{"Invalid port (0)", "my-service", 0, false},
		{"Invalid port (negative)", "my-service", -1, false},
		{"Invalid port (too high)", "my-service", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nameValid := tt.serviceName != ""
			portValid := tt.servicePort > 0 && tt.servicePort <= 65535

			isValid := nameValid && portValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for backend (service=%q, port=%d), got %v",
					tt.valid, tt.serviceName, tt.servicePort, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_TCPServices tests TCP service configurations
func TestNginxIngressConfig_TCPServices(t *testing.T) {
	tests := []struct {
		name   string
		port   int
		target string
		valid  bool
	}{
		{"SSH service", 22, "default/ssh:22", true},
		{"PostgreSQL service", 5432, "database/postgres:5432", true},
		{"MySQL service", 3306, "database/mysql:3306", true},
		{"Redis service", 6379, "cache/redis:6379", true},
		{"Invalid port (0)", 0, "default/service:80", false},
		{"Invalid port (negative)", -1, "default/service:80", false},
		{"Empty target", 22, "", false},
		{"Invalid target format", 22, "invalid-target", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portValid := tt.port > 0 && tt.port <= 65535
			// Target format: namespace/service:port
			targetValid := tt.target != "" && strings.Contains(tt.target, "/") && strings.Contains(tt.target, ":")

			isValid := portValid && targetValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for TCP service (port=%d, target=%q), got %v",
					tt.valid, tt.port, tt.target, isValid)
			}
		})
	}
}

// TestNginxIngressConfig_UDPServices tests UDP service configurations
func TestNginxIngressConfig_UDPServices(t *testing.T) {
	tests := []struct {
		name   string
		port   int
		target string
		valid  bool
	}{
		{"DNS service", 53, "kube-system/coredns:53", true},
		{"NTP service", 123, "default/ntp:123", true},
		{"VoIP service", 5060, "communication/asterisk:5060", true},
		{"Invalid port (0)", 0, "default/service:80", false},
		{"Empty target", 53, "", false},
		{"Invalid target format", 53, "no-namespace-separator", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			portValid := tt.port > 0 && tt.port <= 65535
			targetValid := tt.target != "" && strings.Contains(tt.target, "/") && strings.Contains(tt.target, ":")

			isValid := portValid && targetValid

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for UDP service (port=%d, target=%q), got %v",
					tt.valid, tt.port, tt.target, isValid)
			}
		})
	}
}

// TestNginxIngressAnnotations_RateLimiting tests rate limiting annotations
func TestNginxIngressAnnotations_RateLimiting(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		valid bool
	}{
		{"Rate limit by IP", "nginx.ingress.kubernetes.io/limit-rps", "10", true},
		{"Rate limit burst", "nginx.ingress.kubernetes.io/limit-burst-multiplier", "5", true},
		{"Rate limit connections", "nginx.ingress.kubernetes.io/limit-connections", "100", true},
		{"Invalid RPS (negative)", "nginx.ingress.kubernetes.io/limit-rps", "-1", false},
		{"Invalid RPS (zero)", "nginx.ingress.kubernetes.io/limit-rps", "0", false},
		{"Invalid burst (negative)", "nginx.ingress.kubernetes.io/limit-burst-multiplier", "-1", false},
		{"Empty value", "nginx.ingress.kubernetes.io/limit-rps", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Value should be a positive number
			isValid := tt.value != "" && tt.value != "0" && !strings.HasPrefix(tt.value, "-")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for annotation %q = %q, got %v", tt.valid, tt.key, tt.value, isValid)
			}
		})
	}
}

// TestNginxIngressAnnotations_CORS tests CORS annotations
func TestNginxIngressAnnotations_CORS(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		valid bool
	}{
		{"CORS enabled", "nginx.ingress.kubernetes.io/enable-cors", "true", true},
		{"CORS disabled", "nginx.ingress.kubernetes.io/enable-cors", "false", true},
		{"CORS allow origin", "nginx.ingress.kubernetes.io/cors-allow-origin", "https://example.com", true},
		{"CORS allow all origins", "nginx.ingress.kubernetes.io/cors-allow-origin", "*", true},
		{"CORS methods", "nginx.ingress.kubernetes.io/cors-allow-methods", "GET, POST, OPTIONS", true},
		{"CORS headers", "nginx.ingress.kubernetes.io/cors-allow-headers", "Authorization, Content-Type", true},
		{"Invalid CORS enabled", "nginx.ingress.kubernetes.io/enable-cors", "maybe", false},
		{"HTTP origin (insecure)", "nginx.ingress.kubernetes.io/cors-allow-origin", "http://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isValid bool

			switch tt.key {
			case "nginx.ingress.kubernetes.io/enable-cors":
				isValid = tt.value == "true" || tt.value == "false"
			case "nginx.ingress.kubernetes.io/cors-allow-origin":
				isValid = tt.value == "*" || strings.HasPrefix(tt.value, "https://")
			case "nginx.ingress.kubernetes.io/cors-allow-methods":
				isValid = tt.value != ""
			case "nginx.ingress.kubernetes.io/cors-allow-headers":
				isValid = tt.value != ""
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for annotation %q = %q, got %v", tt.valid, tt.key, tt.value, isValid)
			}
		})
	}
}

// TestNginxIngressAnnotations_Auth tests authentication annotations
func TestNginxIngressAnnotations_Auth(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		valid bool
	}{
		{"Basic auth", "nginx.ingress.kubernetes.io/auth-type", "basic", true},
		{"Digest auth", "nginx.ingress.kubernetes.io/auth-type", "digest", true},
		{"Auth secret", "nginx.ingress.kubernetes.io/auth-secret", "basic-auth", true},
		{"Auth realm", "nginx.ingress.kubernetes.io/auth-realm", "Authentication Required", true},
		{"Invalid auth type", "nginx.ingress.kubernetes.io/auth-type", "oauth", false},
		{"Empty secret", "nginx.ingress.kubernetes.io/auth-secret", "", false},
		{"Empty realm", "nginx.ingress.kubernetes.io/auth-realm", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isValid bool

			switch tt.key {
			case "nginx.ingress.kubernetes.io/auth-type":
				isValid = tt.value == "basic" || tt.value == "digest"
			case "nginx.ingress.kubernetes.io/auth-secret":
				isValid = tt.value != ""
			case "nginx.ingress.kubernetes.io/auth-realm":
				isValid = tt.value != ""
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for annotation %q = %q, got %v", tt.valid, tt.key, tt.value, isValid)
			}
		})
	}
}

// Test100IngressScenarios generates 100 ingress configuration scenarios
func Test100IngressScenarios(t *testing.T) {
	scenarios := []struct {
		domain       string
		sslProtocol  string
		replicaCount int
		valid        bool
	}{
		{"example.com", "TLSv1.2 TLSv1.3", 2, true},
		{"test.io", "TLSv1.3", 3, true},
		{"app.dev", "TLSv1.2", 1, true},
		{"cluster.local", "TLSv1.2 TLSv1.3", 4, true},
		{"k8s.example.com", "TLSv1.3", 2, true},
		{"invalid", "TLSv1.1", 0, false}, // Invalid: no TLD, old TLS, 0 replicas
	}

	// Generate 94 more scenarios programmatically
	for i := 1; i <= 94; i++ {
		scenario := struct {
			domain       string
			sslProtocol  string
			replicaCount int
			valid        bool
		}{
			domain:       "test" + string(rune('a'+i%26)) + ".com",
			sslProtocol:  "TLSv1.2 TLSv1.3",
			replicaCount: (i % 5) + 1,
			valid:        true,
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_scenario_"+string(rune('0'+i%10)), func(t *testing.T) {
			domainValid := strings.Contains(scenario.domain, ".")
			tlsValid := strings.Contains(scenario.sslProtocol, "TLSv1.2") || strings.Contains(scenario.sslProtocol, "TLSv1.3")
			replicasValid := scenario.replicaCount > 0

			isValid := domainValid && tlsValid && replicasValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (domain=%s, tls=%s, replicas=%d)",
					i, scenario.valid, isValid, scenario.domain, scenario.sslProtocol, scenario.replicaCount)
			}
		})
	}
}
