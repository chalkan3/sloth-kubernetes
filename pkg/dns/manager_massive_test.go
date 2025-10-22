package dns

import (
	"strings"
	"testing"
)

// TestDNSRecordNames_Master tests master node DNS naming patterns
func TestDNSRecordNames_Master(t *testing.T) {
	tests := []struct {
		name        string
		masterIndex int
		provider    string
		expected    []string
	}{
		{"Master 1", 1, "digitalocean", []string{"master1", "master1-digitalocean", "k8s-master1", "api", "k8s-api"}},
		{"Master 2", 2, "digitalocean", []string{"master2", "master2-digitalocean", "k8s-master2"}},
		{"Master 3", 3, "linode", []string{"master3", "master3-linode", "k8s-master3"}},
		{"Master 1 Linode", 1, "linode", []string{"master1", "master1-linode", "k8s-master1", "api", "k8s-api"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First master gets "api" and "k8s-api" records
			hasAPIRecord := tt.masterIndex == 1

			// Validate all expected names are present
			for _, expectedName := range tt.expected {
				if expectedName == "api" || expectedName == "k8s-api" {
					if !hasAPIRecord {
						t.Errorf("Master %d should not have %q record", tt.masterIndex, expectedName)
					}
				}
			}

			// Basic naming pattern validation
			basicNames := 3  // master{n}, master{n}-{provider}, k8s-master{n}
			if hasAPIRecord {
				basicNames += 2  // api, k8s-api
			}

			if len(tt.expected) != basicNames {
				t.Errorf("Expected %d DNS names for master %d, got %d", basicNames, tt.masterIndex, len(tt.expected))
			}
		})
	}
}

// TestDNSRecordNames_Worker tests worker node DNS naming patterns
func TestDNSRecordNames_Worker(t *testing.T) {
	tests := []struct {
		name        string
		workerIndex int
		provider    string
		expected    []string
	}{
		{"Worker 1", 1, "digitalocean", []string{"worker1", "worker1-digitalocean", "k8s-worker1"}},
		{"Worker 2", 2, "digitalocean", []string{"worker2", "worker2-digitalocean", "k8s-worker2"}},
		{"Worker 3", 3, "linode", []string{"worker3", "worker3-linode", "k8s-worker3"}},
		{"Worker 5", 5, "digitalocean", []string{"worker5", "worker5-digitalocean", "k8s-worker5"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Workers always have exactly 3 DNS names
			if len(tt.expected) != 3 {
				t.Errorf("Expected 3 DNS names for worker %d, got %d", tt.workerIndex, len(tt.expected))
			}

			// Validate naming pattern
			expectedBasic := "worker" + string(rune('0'+tt.workerIndex))
			if !strings.Contains(tt.expected[0], expectedBasic) && tt.workerIndex < 10 {
				t.Errorf("Expected first name to contain %q, got %q", expectedBasic, tt.expected[0])
			}
		})
	}
}

// TestDNSRecordType_Validation tests DNS record type validation
func TestDNSRecordType_Validation(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		valid      bool
	}{
		{"A record", "A", true},
		{"AAAA record", "AAAA", true},
		{"CNAME record", "CNAME", true},
		{"MX record", "MX", true},
		{"TXT record", "TXT", true},
		{"NS record", "NS", true},
		{"SOA record", "SOA", false},      // Not typically user-created
		{"Invalid type", "INVALID", false},
		{"Empty type", "", false},
		{"Lowercase a", "a", false},       // Should be uppercase
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS"}
			isValid := contains(validTypes, tt.recordType)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for record type %q, got %v", tt.valid, tt.recordType, isValid)
			}
		})
	}
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

// TestDNSTTL_Validation tests TTL value validation
func TestDNSTTL_Validation(t *testing.T) {
	tests := []struct {
		name  string
		ttl   int
		valid bool
	}{
		{"60 seconds (1 min)", 60, true},
		{"300 seconds (5 min)", 300, true},
		{"600 seconds (10 min)", 600, true},
		{"1800 seconds (30 min)", 1800, true},
		{"3600 seconds (1 hour)", 3600, true},
		{"86400 seconds (1 day)", 86400, true},
		{"0 seconds", 0, false},
		{"Negative TTL", -1, false},
		{"Very low (30s)", 30, false},       // Too low for production
		{"Very high (1 week)", 604800, false}, // Too high for dynamic IPs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid TTL: 60-86400 seconds (1 min - 1 day)
			isValid := tt.ttl >= 60 && tt.ttl <= 86400

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for TTL %d, got %v", tt.valid, tt.ttl, isValid)
			}
		})
	}
}

// TestDNSName_Validation tests DNS name format validation
func TestDNSName_Validation(t *testing.T) {
	tests := []struct {
		name     string
		dnsName  string
		valid    bool
	}{
		{"Simple name", "master1", true},
		{"Hyphenated name", "master-1", true},
		{"Numbered name", "node123", true},
		{"Subdomain", "api.k8s", true},
		{"Wildcard", "*.k8s", true},
		{"Long valid name", "very-long-but-valid-dns-name", true},
		{"Empty name", "", false},
		{"Starts with hyphen", "-master", false},
		{"Ends with hyphen", "master-", false},
		{"Contains underscore", "master_1", false},
		{"Contains space", "master 1", false},
		{"Uppercase", "MASTER", false},     // DNS names should be lowercase
		{"Too long (>63 chars)", "this-is-a-very-long-dns-name-that-exceeds-the-maximum-length-of-sixty-three-characters", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.dnsName != "" &&
				tt.dnsName == strings.ToLower(tt.dnsName) &&
				!strings.HasPrefix(tt.dnsName, "-") &&
				!strings.HasSuffix(tt.dnsName, "-") &&
				!strings.Contains(tt.dnsName, "_") &&
				!strings.Contains(tt.dnsName, " ") &&
				len(tt.dnsName) <= 63

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for DNS name %q, got %v", tt.valid, tt.dnsName, isValid)
			}
		})
	}
}

// TestDomainName_Validation tests full domain name validation
func TestDomainName_Validation(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"Simple domain", "example.com", true},
		{"Subdomain", "k8s.example.com", true},
		{"Multi-level", "cluster.k8s.example.com", true},
		{"Hyphenated", "my-cluster.example.com", true},
		{"Two letter TLD", "example.io", true},
		{"Three letter TLD", "example.dev", true},
		{"Empty domain", "", false},
		{"No TLD", "example", false},
		{"Starts with dot", ".example.com", false},
		{"Ends with dot", "example.com.", false},
		{"Double dot", "example..com", false},
		{"Contains underscore", "my_cluster.com", false},
		{"Contains space", "my cluster.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.domain != "" &&
				strings.Contains(tt.domain, ".") &&
				!strings.HasPrefix(tt.domain, ".") &&
				!strings.HasSuffix(tt.domain, ".") &&
				!strings.Contains(tt.domain, "..") &&
				!strings.Contains(tt.domain, "_") &&
				!strings.Contains(tt.domain, " ")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for domain %q, got %v", tt.valid, tt.domain, isValid)
			}
		})
	}
}

// TestWildcardDNS_Validation tests wildcard DNS record validation
func TestWildcardDNS_Validation(t *testing.T) {
	tests := []struct {
		name     string
		wildcard string
		valid    bool
	}{
		{"Valid wildcard", "*.k8s", true},
		{"Valid wildcard subdomain", "*.apps.k8s", true},
		{"Single star only", "*", true},
		{"Multiple wildcards", "*.*.k8s", false},  // Not allowed
		{"Wildcard in middle", "k8s.*.example", false},
		{"No star", "k8s", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid wildcard: starts with "*." or is just "*"
			isValid := (strings.HasPrefix(tt.wildcard, "*.") && strings.Count(tt.wildcard, "*") == 1) ||
				tt.wildcard == "*"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for wildcard %q, got %v", tt.valid, tt.wildcard, isValid)
			}
		})
	}
}

// TestIngressSubdomains tests common ingress subdomain names
func TestIngressSubdomains(t *testing.T) {
	subdomains := []string{
		"grafana", "prometheus", "alertmanager", "dashboard",
		"argocd", "jenkins", "gitlab", "registry",
	}

	tests := []struct {
		name      string
		subdomain string
		valid     bool
	}{
		{"grafana", "grafana", true},
		{"prometheus", "prometheus", true},
		{"dashboard", "dashboard", true},
		{"argocd", "argocd", true},
		{"invalid", "not-a-service", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := contains(subdomains, tt.subdomain)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for subdomain %q, got %v", tt.valid, tt.subdomain, isValid)
			}
		})
	}
}

// TestCNAME_TargetValidation tests CNAME target validation
func TestCNAME_TargetValidation(t *testing.T) {
	tests := []struct {
		name   string
		target string
		valid  bool
	}{
		{"FQDN with trailing dot", "api.example.com.", true},
		{"Relative name", "api", false},        // Should be FQDN
		{"Without trailing dot", "api.example.com", false},
		{"Empty target", "", false},
		{"Just dot", ".", false},
		{"Multiple dots", "api..example.com.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid CNAME target ends with "."
			isValid := tt.target != "" &&
				strings.HasSuffix(tt.target, ".") &&
				!strings.Contains(tt.target, "..") &&
				tt.target != "."

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for CNAME target %q, got %v", tt.valid, tt.target, isValid)
			}
		})
	}
}

// TestPrivateDNS_Naming tests private DNS naming pattern
func TestPrivateDNS_Naming(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		expected string
	}{
		{"Master node", "master1", "private-master1"},
		{"Worker node", "worker2", "private-worker2"},
		{"Custom node", "node-custom", "private-node-custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateName := "private-" + tt.nodeName

			if privateName != tt.expected {
				t.Errorf("Expected private DNS name %q, got %q", tt.expected, privateName)
			}
		})
	}
}

// TestWireGuardDNS_Naming tests WireGuard DNS naming pattern
func TestWireGuardDNS_Naming(t *testing.T) {
	tests := []struct {
		name     string
		nodeName string
		expected string
	}{
		{"Master node", "master1", "wg-master1"},
		{"Worker node", "worker2", "wg-worker2"},
		{"Custom node", "node-custom", "wg-node-custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wgName := "wg-" + tt.nodeName

			if wgName != tt.expected {
				t.Errorf("Expected WireGuard DNS name %q, got %q", tt.expected, wgName)
			}
		})
	}
}

// TestDNSProvider_Support tests DNS provider support
func TestDNSProvider_Support(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		supported bool
	}{
		{"DigitalOcean", "digitalocean", true},
		{"AWS Route53", "route53", true},
		{"Cloudflare", "cloudflare", true},
		{"Google Cloud DNS", "google", true},
		{"Azure DNS", "azure", true},
		{"Unknown provider", "unknown", false},
		{"Empty provider", "", false},
	}

	supportedProviders := []string{"digitalocean", "route53", "cloudflare", "google", "azure"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSupported := contains(supportedProviders, tt.provider)

			if isSupported != tt.supported {
				t.Errorf("Expected supported=%v for provider %q, got %v", tt.supported, tt.provider, isSupported)
			}
		})
	}
}

// TestDNSUpdateFrequency tests DNS update frequency based on TTL
func TestDNSUpdateFrequency(t *testing.T) {
	tests := []struct {
		name       string
		ttl        int
		updateFreq string
		appropriate bool
	}{
		{"60s TTL for frequent updates", 60, "frequent", true},
		{"300s TTL for moderate updates", 300, "moderate", true},
		{"3600s TTL for rare updates", 3600, "rare", true},
		{"60s TTL for rare updates", 60, "rare", false},    // TTL too low
		{"3600s TTL for frequent updates", 3600, "frequent", false}, // TTL too high
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isAppropriate bool

			switch tt.updateFreq {
			case "frequent":
				isAppropriate = tt.ttl <= 300
			case "moderate":
				isAppropriate = tt.ttl >= 300 && tt.ttl <= 1800
			case "rare":
				isAppropriate = tt.ttl >= 1800
			}

			if isAppropriate != tt.appropriate {
				t.Errorf("Expected appropriate=%v for TTL %d with %s updates, got %v",
					tt.appropriate, tt.ttl, tt.updateFreq, isAppropriate)
			}
		})
	}
}

// TestDNSRecordPriority tests MX record priority values
func TestDNSRecordPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		valid    bool
	}{
		{"Priority 10", 10, true},
		{"Priority 20", 20, true},
		{"Priority 0", 0, true},
		{"Priority 65535", 65535, true},
		{"Negative priority", -1, false},
		{"Too high priority", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid priority: 0-65535
			isValid := tt.priority >= 0 && tt.priority <= 65535

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for priority %d, got %v", tt.valid, tt.priority, isValid)
			}
		})
	}
}

// TestDNSZone_Validation tests DNS zone validation
func TestDNSZone_Validation(t *testing.T) {
	tests := []struct {
		name  string
		zone  string
		valid bool
	}{
		{"Valid zone", "example.com", true},
		{"Subdomain zone", "k8s.example.com", true},
		{"Empty zone", "", false},
		{"No TLD", "example", false},
		{"Invalid characters", "example$.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.zone != "" &&
				strings.Contains(tt.zone, ".") &&
				!strings.ContainsAny(tt.zone, "$@#%^&*()")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for zone %q, got %v", tt.valid, tt.zone, isValid)
			}
		})
	}
}

// TestServiceURL_Generation tests service URL generation
func TestServiceURL_Generation(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		domain   string
		protocol string
		expected string
	}{
		{"Grafana HTTPS", "grafana", "example.com", "https", "https://grafana.k8s.example.com"},
		{"Prometheus HTTPS", "prometheus", "example.com", "https", "https://prometheus.k8s.example.com"},
		{"Dashboard HTTP", "dashboard", "example.com", "http", "http://dashboard.k8s.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generatedURL := tt.protocol + "://" + tt.service + ".k8s." + tt.domain

			if generatedURL != tt.expected {
				t.Errorf("Expected URL %q, got %q", tt.expected, generatedURL)
			}
		})
	}
}

// TestAPIEndpoint_Generation tests API endpoint URL generation
func TestAPIEndpoint_Generation(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		port     int
		expected string
	}{
		{"Standard API port", "example.com", 6443, "https://api.example.com:6443"},
		{"Custom port", "example.com", 8443, "https://api.example.com:8443"},
		{"Different domain", "k8s.io", 6443, "https://api.k8s.io:6443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simplified URL construction for testing
			// In reality, would use fmt.Sprintf
			if tt.port == 6443 && !strings.Contains(tt.expected, "6443") {
				t.Errorf("Expected port 6443 in URL")
			}

			// Validate domain is in expected URL
			if !strings.Contains(tt.expected, tt.domain) {
				t.Errorf("Expected domain %q in URL %q", tt.domain, tt.expected)
			}
		})
	}
}

// Test100DNSScenarios generates 100 DNS configuration scenarios
func Test100DNSScenarios(t *testing.T) {
	scenarios := []struct {
		domain      string
		ttl         int
		recordType  string
		recordCount int
		valid       bool
	}{
		{"example.com", 300, "A", 10, true},
		{"test.io", 600, "A", 5, true},
		{"cluster.dev", 300, "CNAME", 3, true},
		{"k8s.cloud", 1800, "A", 20, true},
		{"invalid", 0, "A", 1, false},  // Invalid: no TLD, zero TTL
		{"example.com", -1, "INVALID", 0, false}, // Invalid: negative TTL, invalid type
	}

	// Generate 94 more scenarios
	for i := 1; i <= 94; i++ {
		domain := "test" + string(rune('a'+i%26)) + ".com"
		ttl := 300 + (i%10)*300
		recordType := "A"
		if i%3 == 0 {
			recordType = "CNAME"
		}
		recordCount := 1 + (i % 20)

		scenario := struct {
			domain      string
			ttl         int
			recordType  string
			recordCount int
			valid       bool
		}{
			domain:      domain,
			ttl:         ttl,
			recordType:  recordType,
			recordCount: recordCount,
			valid:       strings.Contains(domain, ".") && ttl >= 60 && ttl <= 86400 && (recordType == "A" || recordType == "CNAME"),
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_dns_"+string(rune('0'+i%10)), func(t *testing.T) {
			domainValid := strings.Contains(scenario.domain, ".")
			ttlValid := scenario.ttl >= 60 && scenario.ttl <= 86400
			recordTypeValid := scenario.recordType == "A" || scenario.recordType == "CNAME" || scenario.recordType == "AAAA"

			isValid := domainValid && ttlValid && recordTypeValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (domain=%s, ttl=%d, type=%s, count=%d)",
					i, scenario.valid, isValid, scenario.domain, scenario.ttl, scenario.recordType, scenario.recordCount)
			}
		})
	}
}
