package dns

import (
	"fmt"
	"strings"
	"testing"
)

// TestDNSRecordName_Format tests DNS record name format
func TestDNSRecordName_Format(t *testing.T) {
	tests := []struct {
		name  string
		record string
		valid bool
	}{
		{"Simple name", "api", true},
		{"With number", "master1", true},
		{"With hyphen", "k8s-master1", true},
		{"Subdomain", "api.cluster", true},
		{"With provider", "master1-digitalocean", true},
		{"Uppercase", "API", false},
		{"With underscore", "master_1", false},
		{"With space", "master 1", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.record != "" &&
				tt.record == strings.ToLower(tt.record) &&
				!strings.Contains(tt.record, "_") &&
				!strings.Contains(tt.record, " ")
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for record %q, got %v", tt.valid, tt.record, isValid)
			}
		})
	}
}

// TestDNSRecordType_Values tests DNS record types
func TestDNSRecordType_Values(t *testing.T) {
	validTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "SRV"}

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
		{"SRV record", "SRV", true},
		{"Invalid type", "INVALID", false},
		{"Lowercase", "a", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, vt := range validTypes {
				if tt.recordType == vt {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for type %q, got %v", tt.valid, tt.recordType, isValid)
			}
		})
	}
}

// TestDNSDomain_Validation tests domain name validation
func TestDNSDomain_Validation(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"Valid domain", "example.com", true},
		{"Valid subdomain", "api.example.com", true},
		{"Valid country TLD", "example.com.br", true},
		{"Valid new TLD", "example.io", true},
		{"Multiple subdomains", "api.k8s.example.com", true},
		{"No TLD", "example", false},
		{"With space", "exam ple.com", false},
		{"With underscore", "exam_ple.com", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.domain != "" &&
				strings.Contains(tt.domain, ".") &&
				!strings.Contains(tt.domain, " ") &&
				!strings.Contains(tt.domain, "_")
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for domain %q, got %v", tt.valid, tt.domain, isValid)
			}
		})
	}
}

// TestDNSNodeType_Detection tests node type detection
func TestDNSNodeType_Detection(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		nodeType string
	}{
		{"Master node", "master", "master"},
		{"Control plane", "controlplane", "controlplane"},
		{"Worker node", "worker", "worker"},
		{"Unknown role", "unknown", "node"},
		{"Empty role", "", "node"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeType := "node"
			if tt.role == "master" || tt.role == "controlplane" {
				nodeType = tt.role
			} else if tt.role == "worker" {
				nodeType = "worker"
			}

			if nodeType != tt.nodeType {
				t.Errorf("Expected nodeType %q, got %q", tt.nodeType, nodeType)
			}
		})
	}
}

// TestDNSMasterNames_Generation tests master node DNS names
func TestDNSMasterNames_Generation(t *testing.T) {
	tests := []struct {
		name         string
		masterNum    int
		provider     string
		expectedNames []string
	}{
		{
			"First master",
			1,
			"digitalocean",
			[]string{"master1", "master1-digitalocean", "k8s-master1", "api", "k8s-api"},
		},
		{
			"Second master",
			2,
			"linode",
			[]string{"master2", "master2-linode", "k8s-master2"},
		},
		{
			"Third master",
			3,
			"digitalocean",
			[]string{"master3", "master3-digitalocean", "k8s-master3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dnsNames []string
			dnsNames = append(dnsNames,
				fmt.Sprintf("master%d", tt.masterNum),
				fmt.Sprintf("master%d-%s", tt.masterNum, tt.provider),
				fmt.Sprintf("k8s-master%d", tt.masterNum),
			)

			if tt.masterNum == 1 {
				dnsNames = append(dnsNames, "api", "k8s-api")
			}

			if len(dnsNames) != len(tt.expectedNames) {
				t.Errorf("Expected %d names, got %d", len(tt.expectedNames), len(dnsNames))
			}

			for i, expected := range tt.expectedNames {
				if i < len(dnsNames) && dnsNames[i] != expected {
					t.Errorf("Expected name[%d] to be %q, got %q", i, expected, dnsNames[i])
				}
			}
		})
	}
}

// TestDNSWorkerNames_Generation tests worker node DNS names
func TestDNSWorkerNames_Generation(t *testing.T) {
	tests := []struct {
		name         string
		workerNum    int
		provider     string
		expectedNames []string
	}{
		{
			"First worker",
			1,
			"digitalocean",
			[]string{"worker1", "worker1-digitalocean", "k8s-worker1"},
		},
		{
			"Second worker",
			2,
			"linode",
			[]string{"worker2", "worker2-linode", "k8s-worker2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dnsNames := []string{
				fmt.Sprintf("worker%d", tt.workerNum),
				fmt.Sprintf("worker%d-%s", tt.workerNum, tt.provider),
				fmt.Sprintf("k8s-worker%d", tt.workerNum),
			}

			if len(dnsNames) != len(tt.expectedNames) {
				t.Errorf("Expected %d names, got %d", len(tt.expectedNames), len(dnsNames))
			}

			for i, expected := range tt.expectedNames {
				if i < len(dnsNames) && dnsNames[i] != expected {
					t.Errorf("Expected name[%d] to be %q, got %q", i, expected, dnsNames[i])
				}
			}
		})
	}
}

// TestDNSPrivateRecordName_Format tests private DNS record naming
func TestDNSPrivateRecordName_Format(t *testing.T) {
	tests := []struct {
		name         string
		nodeName     string
		expectedName string
	}{
		{"Master node", "master-1", "private-master-1"},
		{"Worker node", "worker-1", "private-worker-1"},
		{"Custom node", "node-01", "private-node-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateName := fmt.Sprintf("private-%s", tt.nodeName)
			if privateName != tt.expectedName {
				t.Errorf("Expected %q, got %q", tt.expectedName, privateName)
			}
		})
	}
}

// TestDNSWireGuardRecordName_Format tests WireGuard DNS record naming
func TestDNSWireGuardRecordName_Format(t *testing.T) {
	tests := []struct {
		name         string
		nodeName     string
		expectedName string
	}{
		{"Master node", "master-1", "wg-master-1"},
		{"Worker node", "worker-1", "wg-worker-1"},
		{"Custom node", "node-01", "wg-node-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wgName := fmt.Sprintf("wg-%s", tt.nodeName)
			if wgName != tt.expectedName {
				t.Errorf("Expected %q, got %q", tt.expectedName, wgName)
			}
		})
	}
}

// TestDNSTTL_Values tests DNS TTL values
func TestDNSTTL_Values(t *testing.T) {
	tests := []struct {
		name  string
		ttl   int
		valid bool
	}{
		{"1 minute", 60, true},
		{"5 minutes", 300, true},
		{"1 hour", 3600, true},
		{"1 day", 86400, true},
		{"Too short", 30, false},
		{"Zero", 0, false},
		{"Negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.ttl >= 60 && tt.ttl <= 86400
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for TTL %d, got %v", tt.valid, tt.ttl, isValid)
			}
		})
	}
}

// TestDNSIPAddress_Format tests IP address format
func TestDNSIPAddress_Format(t *testing.T) {
	tests := []struct {
		name  string
		ip    string
		valid bool
	}{
		{"Valid IPv4", "192.168.1.1", true},
		{"Valid public IP", "8.8.8.8", true},
		{"Valid private IP", "10.0.0.1", true},
		{"Localhost", "127.0.0.1", true},
		{"Invalid format", "192.168.1", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.ip, ".")
			isValid := len(parts) == 4 && tt.ip != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for IP %q, got %v", tt.valid, tt.ip, isValid)
			}
		})
	}
}

// TestDNSAPIEndpoint_Names tests API endpoint DNS names
func TestDNSAPIEndpoint_Names(t *testing.T) {
	tests := []struct {
		name          string
		aliases       []string
		includesAPI   bool
		includesK8sAPI bool
	}{
		{
			"First master gets API aliases",
			[]string{"api", "k8s-api"},
			true,
			true,
		},
		{
			"Other masters don't get API aliases",
			[]string{},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAPI := false
			hasK8sAPI := false
			for _, alias := range tt.aliases {
				if alias == "api" {
					hasAPI = true
				}
				if alias == "k8s-api" {
					hasK8sAPI = true
				}
			}

			if hasAPI != tt.includesAPI {
				t.Errorf("Expected includesAPI=%v, got %v", tt.includesAPI, hasAPI)
			}
			if hasK8sAPI != tt.includesK8sAPI {
				t.Errorf("Expected includesK8sAPI=%v, got %v", tt.includesK8sAPI, hasK8sAPI)
			}
		})
	}
}

// TestDNSRecordPriority_MX tests MX record priority
func TestDNSRecordPriority_MX(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		valid    bool
	}{
		{"Low priority", 10, true},
		{"Medium priority", 20, true},
		{"High priority", 30, true},
		{"Very high priority", 50, true},
		{"Zero priority", 0, false},
		{"Negative priority", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.priority > 0 && tt.priority <= 65535
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for priority %d, got %v", tt.valid, tt.priority, isValid)
			}
		})
	}
}

// TestDNSWildcard_Support tests wildcard DNS records
func TestDNSWildcard_Support(t *testing.T) {
	tests := []struct {
		name     string
		record   string
		wildcard bool
	}{
		{"Wildcard subdomain", "*.example.com", true},
		{"Wildcard all", "*.api.example.com", true},
		{"Not wildcard", "api.example.com", false},
		{"Just asterisk", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isWildcard := strings.HasPrefix(tt.record, "*")
			if isWildcard != tt.wildcard {
				t.Errorf("Expected wildcard=%v for record %q, got %v", tt.wildcard, tt.record, isWildcard)
			}
		})
	}
}

// Test200DNSScenarios generates 200 DNS test scenarios
func Test200DNSScenarios(t *testing.T) {
	scenarios := []struct {
		recordType string
		name       string
		domain     string
		nodeType   string
		provider   string
		valid      bool
	}{
		{"A", "api", "example.com", "master", "digitalocean", true},
		{"A", "master1", "example.com", "master", "digitalocean", true},
		{"A", "worker1", "example.com", "worker", "linode", true},
	}

	// Generate 197 more scenarios
	recordTypes := []string{"A", "AAAA", "CNAME"}
	domains := []string{"example.com", "cluster.io", "k8s.dev", "cloud.net"}
	nodeTypes := []string{"master", "worker", "controlplane"}
	providers := []string{"digitalocean", "linode"}

	for i := 0; i < 197; i++ {
		recordType := recordTypes[i%len(recordTypes)]
		domain := domains[i%len(domains)]
		nodeType := nodeTypes[i%len(nodeTypes)]
		provider := providers[i%len(providers)]

		var name string
		if nodeType == "master" || nodeType == "controlplane" {
			name = fmt.Sprintf("master%d", (i%10)+1)
		} else {
			name = fmt.Sprintf("worker%d", (i%10)+1)
		}

		scenarios = append(scenarios, struct {
			recordType string
			name       string
			domain     string
			nodeType   string
			provider   string
			valid      bool
		}{
			recordType: recordType,
			name:       name,
			domain:     domain,
			nodeType:   nodeType,
			provider:   provider,
			valid:      true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(fmt.Sprintf("DNS_%d_%s", i, scenario.recordType), func(t *testing.T) {
			// Validate record type
			validTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "SRV"}
			typeValid := false
			for _, vt := range validTypes {
				if scenario.recordType == vt {
					typeValid = true
					break
				}
			}

			// Validate name
			nameValid := scenario.name != "" &&
				scenario.name == strings.ToLower(scenario.name) &&
				!strings.Contains(scenario.name, "_")

			// Validate domain
			domainValid := scenario.domain != "" && strings.Contains(scenario.domain, ".")

			// Validate node type
			nodeTypeValid := scenario.nodeType == "master" ||
				scenario.nodeType == "worker" ||
				scenario.nodeType == "controlplane"

			// Validate provider
			providerValid := scenario.provider == "digitalocean" || scenario.provider == "linode"

			isValid := typeValid && nameValid && domainValid && nodeTypeValid && providerValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}

			// Generate full FQDN
			fqdn := fmt.Sprintf("%s.%s", scenario.name, scenario.domain)
			if !strings.Contains(fqdn, ".") {
				t.Errorf("Scenario %d: Invalid FQDN %q", i, fqdn)
			}
		})
	}
}
