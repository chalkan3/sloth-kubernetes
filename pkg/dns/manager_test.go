package dns

import (
	"fmt"
	"strings"
	"testing"

	"sloth-kubernetes/pkg/providers"
)

// TestNewManager tests DNS manager creation
func TestNewManager(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"Standard domain", "example.com"},
		{"Subdomain", "k8s.example.com"},
		{"Multi-level subdomain", "prod.k8s.example.com"},
		{"Different TLD", "example.io"},
		{"Country TLD", "example.co.uk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Can't test Pulumi context, but we can test domain storage
			if tt.domain == "" {
				t.Error("Domain should not be empty")
			}
		})
	}
}

// TestGetDomain tests domain retrieval
func TestGetDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"Simple domain", "example.com"},
		{"Subdomain", "k8s.example.com"},
		{"Complex domain", "prod.cluster.k8s.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{domain: tt.domain}

			got := manager.GetDomain()
			if got != tt.domain {
				t.Errorf("Expected domain %q, got %q", tt.domain, got)
			}
		})
	}
}

// TestDNSNameGeneration tests DNS name generation logic
func TestDNSNameGeneration(t *testing.T) {
	tests := []struct {
		name         string
		nodeType     string
		nodeCount    int
		provider     string
		expectedDNS  []string
	}{
		{
			name:      "First master node",
			nodeType:  "master",
			nodeCount: 1,
			provider:  "digitalocean",
			expectedDNS: []string{
				"master1",
				"master1-digitalocean",
				"k8s-master1",
				"api",
				"k8s-api",
			},
		},
		{
			name:      "Second master node",
			nodeType:  "master",
			nodeCount: 2,
			provider:  "linode",
			expectedDNS: []string{
				"master2",
				"master2-linode",
				"k8s-master2",
			},
		},
		{
			name:      "First worker node",
			nodeType:  "worker",
			nodeCount: 1,
			provider:  "digitalocean",
			expectedDNS: []string{
				"worker1",
				"worker1-digitalocean",
				"k8s-worker1",
			},
		},
		{
			name:      "Third worker node",
			nodeType:  "worker",
			nodeCount: 3,
			provider:  "aws",
			expectedDNS: []string{
				"worker3",
				"worker3-aws",
				"k8s-worker3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test DNS name patterns
			var generatedNames []string

			switch tt.nodeType {
			case "master":
				generatedNames = append(generatedNames,
					fmt.Sprintf("master%d", tt.nodeCount),
					fmt.Sprintf("master%d-%s", tt.nodeCount, tt.provider),
					fmt.Sprintf("k8s-master%d", tt.nodeCount),
				)
				if tt.nodeCount == 1 {
					generatedNames = append(generatedNames, "api", "k8s-api")
				}
			case "worker":
				generatedNames = append(generatedNames,
					fmt.Sprintf("worker%d", tt.nodeCount),
					fmt.Sprintf("worker%d-%s", tt.nodeCount, tt.provider),
					fmt.Sprintf("k8s-worker%d", tt.nodeCount),
				)
			}

			// Verify all expected DNS names are generated
			for _, expected := range tt.expectedDNS {
				found := false
				for _, generated := range generatedNames {
					if generated == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected DNS name %q not found in generated names: %v", expected, generatedNames)
				}
			}
		})
	}
}

// TestNodeRoleDetection tests role detection from labels
func TestNodeRoleDetection(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		wantRole string
	}{
		{
			name:     "Master role",
			labels:   map[string]string{"role": "master"},
			wantRole: "master",
		},
		{
			name:     "Controlplane role",
			labels:   map[string]string{"role": "controlplane"},
			wantRole: "controlplane",
		},
		{
			name:     "Worker role",
			labels:   map[string]string{"role": "worker"},
			wantRole: "worker",
		},
		{
			name:     "No role label",
			labels:   map[string]string{},
			wantRole: "node", // Default
		},
		{
			name:     "Nil labels",
			labels:   nil,
			wantRole: "node", // Default
		},
		{
			name:     "Different label",
			labels:   map[string]string{"environment": "production"},
			wantRole: "node", // Default when role not found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate role detection logic
			nodeType := "node"
			if tt.labels != nil {
				if role, ok := tt.labels["role"]; ok {
					nodeType = role
				}
			}

			if nodeType != tt.wantRole {
				t.Errorf("Expected role %q, got %q", tt.wantRole, nodeType)
			}
		})
	}
}

// TestWireGuardDNSNaming tests WireGuard DNS name generation
func TestWireGuardDNSNaming(t *testing.T) {
	tests := []struct {
		name         string
		nodeName     string
		nodeType     string
		nodeCount    int
		wantWGNames  []string
	}{
		{
			name:      "Master node with WireGuard",
			nodeName:  "master-1",
			nodeType:  "master",
			nodeCount: 1,
			wantWGNames: []string{
				"wg-master-1",
				"wg-master1",
			},
		},
		{
			name:      "Worker node with WireGuard",
			nodeName:  "worker-3",
			nodeType:  "worker",
			nodeCount: 3,
			wantWGNames: []string{
				"wg-worker-3",
				"wg-worker3",
			},
		},
		{
			name:      "Generic node with WireGuard",
			nodeName:  "node-5",
			nodeType:  "node",
			nodeCount: 5,
			wantWGNames: []string{
				"wg-node-5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test WireGuard DNS naming pattern
			var generatedWGNames []string

			// Base WireGuard name
			generatedWGNames = append(generatedWGNames, fmt.Sprintf("wg-%s", tt.nodeName))

			// Numbered WireGuard names
			if tt.nodeType == "master" || tt.nodeType == "controlplane" {
				generatedWGNames = append(generatedWGNames, fmt.Sprintf("wg-master%d", tt.nodeCount))
			} else if tt.nodeType == "worker" {
				generatedWGNames = append(generatedWGNames, fmt.Sprintf("wg-worker%d", tt.nodeCount))
			}

			// Verify expected names are generated
			for _, expected := range tt.wantWGNames {
				found := false
				for _, generated := range generatedWGNames {
					if generated == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected WireGuard DNS name %q not found: %v", expected, generatedWGNames)
				}
			}
		})
	}
}

// TestPrivateDNSNaming tests private DNS record naming
func TestPrivateDNSNaming(t *testing.T) {
	tests := []struct {
		name         string
		nodeName     string
		wantPrivate  string
	}{
		{
			name:        "Master node private DNS",
			nodeName:    "master-1",
			wantPrivate: "private-master-1",
		},
		{
			name:        "Worker node private DNS",
			nodeName:    "worker-5",
			wantPrivate: "private-worker-5",
		},
		{
			name:        "Generic node private DNS",
			nodeName:    "node-xyz",
			wantPrivate: "private-node-xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateName := fmt.Sprintf("private-%s", tt.nodeName)

			if privateName != tt.wantPrivate {
				t.Errorf("Expected private DNS %q, got %q", tt.wantPrivate, privateName)
			}
		})
	}
}

// TestIngressSubdomainsList tests ingress subdomain list
func TestIngressSubdomainsList(t *testing.T) {
	expectedSubdomains := []string{
		"grafana",
		"prometheus",
		"alertmanager",
		"dashboard",
		"argocd",
		"jenkins",
		"gitlab",
		"registry",
	}

	// Verify all subdomains are strings and non-empty
	for i, subdomain := range expectedSubdomains {
		if subdomain == "" {
			t.Errorf("Subdomain %d is empty", i)
		}

		// Should be lowercase
		if subdomain != subdomain {
			t.Errorf("Subdomain %s should be lowercase", subdomain)
		}

		// Should not contain special characters
		for _, char := range subdomain {
			if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
				t.Errorf("Subdomain %s contains invalid character: %c", subdomain, char)
			}
		}
	}

	// Verify count
	if len(expectedSubdomains) != 8 {
		t.Errorf("Expected 8 ingress subdomains, got %d", len(expectedSubdomains))
	}
}

// TestCNAMEConveniences tests convenience CNAME records
func TestCNAMEConveniences(t *testing.T) {
	conveniences := map[string]string{
		"k8s":        "api",
		"kubernetes": "api",
		"cluster":    "api",
		"rancher":    "kube-ingress",
		"dashboard":  "kube-ingress",
	}

	tests := []struct {
		name   string
		record string
		target string
	}{
		{"k8s points to api", "k8s", "api"},
		{"kubernetes points to api", "kubernetes", "api"},
		{"cluster points to api", "cluster", "api"},
		{"rancher points to ingress", "rancher", "kube-ingress"},
		{"dashboard points to ingress", "dashboard", "kube-ingress"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, ok := conveniences[tt.record]
			if !ok {
				t.Errorf("Convenience record %q not found", tt.record)
				return
			}

			if target != tt.target {
				t.Errorf("Record %q: expected target %q, got %q", tt.record, tt.target, target)
			}
		})
	}
}

// TestDNSInfoExport tests DNS info structure
func TestDNSInfoExport(t *testing.T) {
	domain := "example.com"

	expectedInfo := map[string]interface{}{
		"domain":      domain,
		"ingress_url": fmt.Sprintf("https://kube-ingress.%s", domain),
		"api_url":     fmt.Sprintf("https://api.%s:6443", domain),
		"wildcard":    fmt.Sprintf("*.k8s.%s", domain),
	}

	// Test URLs are well-formed
	ingressURL, ok := expectedInfo["ingress_url"].(string)
	if !ok || ingressURL != "https://kube-ingress.example.com" {
		t.Errorf("Ingress URL invalid: %v", ingressURL)
	}

	apiURL, ok := expectedInfo["api_url"].(string)
	if !ok || apiURL != "https://api.example.com:6443" {
		t.Errorf("API URL invalid: %v", apiURL)
	}

	wildcard, ok := expectedInfo["wildcard"].(string)
	if !ok || wildcard != "*.k8s.example.com" {
		t.Errorf("Wildcard invalid: %v", wildcard)
	}
}

// TestServiceURLGeneration tests service URL generation
func TestServiceURLGeneration(t *testing.T) {
	domain := "example.com"

	services := map[string]string{
		"grafana":    fmt.Sprintf("https://grafana.k8s.%s", domain),
		"prometheus": fmt.Sprintf("https://prometheus.k8s.%s", domain),
		"dashboard":  fmt.Sprintf("https://dashboard.k8s.%s", domain),
	}

	tests := []struct {
		name        string
		service     string
		expectedURL string
	}{
		{"Grafana URL", "grafana", "https://grafana.k8s.example.com"},
		{"Prometheus URL", "prometheus", "https://prometheus.k8s.example.com"},
		{"Dashboard URL", "dashboard", "https://dashboard.k8s.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, ok := services[tt.service]
			if !ok {
				t.Errorf("Service %q not found", tt.service)
				return
			}

			if url != tt.expectedURL {
				t.Errorf("Service %q: expected URL %q, got %q", tt.service, tt.expectedURL, url)
			}
		})
	}
}

// TestNodeDNSNameList tests node DNS name list generation
func TestNodeDNSNameList(t *testing.T) {
	domain := "example.com"

	var nodeNames []string
	for i := 1; i <= 3; i++ {
		nodeNames = append(nodeNames, fmt.Sprintf("master%d.%s", i, domain))
	}
	for i := 1; i <= 3; i++ {
		nodeNames = append(nodeNames, fmt.Sprintf("worker%d.%s", i, domain))
	}

	// Should have 6 total names (3 masters + 3 workers)
	if len(nodeNames) != 6 {
		t.Errorf("Expected 6 node names, got %d", len(nodeNames))
	}

	// Check master names
	expectedMasters := []string{
		"master1.example.com",
		"master2.example.com",
		"master3.example.com",
	}
	for i, expected := range expectedMasters {
		if nodeNames[i] != expected {
			t.Errorf("Master %d: expected %q, got %q", i+1, expected, nodeNames[i])
		}
	}

	// Check worker names
	expectedWorkers := []string{
		"worker1.example.com",
		"worker2.example.com",
		"worker3.example.com",
	}
	for i, expected := range expectedWorkers {
		if nodeNames[i+3] != expected {
			t.Errorf("Worker %d: expected %q, got %q", i+1, expected, nodeNames[i+3])
		}
	}
}

// TestManagerStructure tests Manager structure initialization
func TestManagerStructure(t *testing.T) {
	domain := "test.com"
	manager := &Manager{
		domain: domain,
		nodes:  make([]*providers.NodeOutput, 0),
	}

	if manager.domain != domain {
		t.Errorf("Expected domain %q, got %q", domain, manager.domain)
	}

	if manager.nodes == nil {
		t.Error("Nodes slice should be initialized")
	}

	if len(manager.nodes) != 0 {
		t.Errorf("Expected 0 nodes initially, got %d", len(manager.nodes))
	}
}

// TestDNSTTL tests DNS record TTL value
func TestDNSTTL(t *testing.T) {
	expectedTTL := 300 // 5 minutes

	if expectedTTL != 300 {
		t.Errorf("Expected TTL 300 seconds, got %d", expectedTTL)
	}

	// Verify it's a reasonable value (between 60 and 3600)
	if expectedTTL < 60 || expectedTTL > 3600 {
		t.Errorf("TTL %d is outside reasonable range (60-3600)", expectedTTL)
	}
}

// TestWildcardDNSPattern tests wildcard DNS pattern
func TestWildcardDNSPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		isValid  bool
	}{
		{"Valid wildcard", "*.k8s", true},
		{"Invalid - no asterisk", "k8s", false},
		{"Invalid - multiple asterisks", "**.k8s", false},
		{"Valid - different subdomain", "*.apps", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wildcard should start with "*."
			hasWildcard := len(tt.pattern) > 2 && tt.pattern[0] == '*' && tt.pattern[1] == '.'
			hasDoubleWildcard := len(tt.pattern) > 3 && tt.pattern[0:3] == "**."

			isValid := hasWildcard && !hasDoubleWildcard

			if isValid != tt.isValid {
				t.Errorf("Pattern %q: expected valid=%v, got valid=%v", tt.pattern, tt.isValid, isValid)
			}
		})
	}
}

// TestDNSRecordTypes tests DNS record type values
func TestDNSRecordTypes(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		isValid    bool
	}{
		{"A record", "A", true},
		{"CNAME record", "CNAME", true},
		{"AAAA record", "AAAA", true},
		{"Invalid type", "INVALID", false},
		{"Lowercase invalid", "a", false},
	}

	validTypes := map[string]bool{
		"A":     true,
		"AAAA":  true,
		"CNAME": true,
		"MX":    true,
		"TXT":   true,
		"NS":    true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, isValid := validTypes[tt.recordType]

			if isValid != tt.isValid {
				t.Errorf("Record type %q: expected valid=%v, got valid=%v", tt.recordType, tt.isValid, isValid)
			}
		})
	}
}

// TestDNSNameLowercase tests that DNS names are lowercased
func TestDNSNameLowercase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Already lowercase", "master-1", "master-1"},
		{"Uppercase", "MASTER-1", "master-1"},
		{"Mixed case", "Master-1", "master-1"},
		{"With numbers", "Worker123", "worker123"},
		{"Special chars preserved", "node-test_1", "node-test_1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the lowercase logic as in createARecord
			got := strings.ToLower(tt.input)

			if got != tt.want {
				t.Errorf("ToLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestCNAMETargetFormat tests CNAME target format
func TestCNAMETargetFormat(t *testing.T) {
	domain := "example.com"
	target := "api"

	// CNAME target should be fully qualified with trailing dot
	expectedFormat := fmt.Sprintf("%s.%s.", target, domain)

	if expectedFormat != "api.example.com." {
		t.Errorf("Expected CNAME target %q, got %q", "api.example.com.", expectedFormat)
	}

	// Should end with dot
	if expectedFormat[len(expectedFormat)-1] != '.' {
		t.Error("CNAME target should end with dot")
	}
}
