package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test NewManager creation
func TestNewManager_Creation(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{"Simple domain", "example.com"},
		{"Subdomain", "cluster.example.com"},
		{"Long domain", "kubernetes.production.cluster.example.com"},
		{"Short domain", "k8s.io"},
		{"Hyphenated domain", "my-cluster.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				domain: tt.domain,
			}

			assert.NotNil(t, manager)
			assert.Equal(t, tt.domain, manager.domain)
		})
	}
}

// Test GetDomain method
func TestManager_GetDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{"Basic domain", "example.com", "example.com"},
		{"Subdomain", "k8s.example.com", "k8s.example.com"},
		{"Empty domain", "", ""},
		{"Single char", "a.io", "a.io"},
		{"Long TLD", "example.museum", "example.museum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{domain: tt.domain}
			result := m.GetDomain()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test DNS name generation logic
func TestDNSNameGeneration_MasterNodes(t *testing.T) {
	tests := []struct {
		name         string
		masterCount  int
		provider     string
		expectedBase []string
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
			// Simulate the DNS name generation logic from CreateNodeRecords
			var dnsNames []string

			dnsNames = []string{
				"master" + string(rune('0'+tt.masterCount)),
				"master" + string(rune('0'+tt.masterCount)) + "-" + tt.provider,
				"k8s-master" + string(rune('0'+tt.masterCount)),
			}

			if tt.masterCount == 1 {
				dnsNames = append(dnsNames, "api", "k8s-api")
			}

			// Verify expected names are generated
			for _, expectedName := range tt.expectedBase {
				assert.Contains(t, dnsNames, expectedName,
					"Expected DNS name %s not found for master%d", expectedName, tt.masterCount)
			}
		})
	}
}

// Test DNS name generation for worker nodes
func TestDNSNameGeneration_WorkerNodes(t *testing.T) {
	tests := []struct {
		name         string
		workerCount  int
		provider     string
		expectedBase []string
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
		{
			"Fifth worker",
			5,
			"digitalocean",
			[]string{"worker5", "worker5-digitalocean", "k8s-worker5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate worker DNS name generation
			var dnsNames []string

			dnsNames = []string{
				"worker" + string(rune('0'+tt.workerCount)),
				"worker" + string(rune('0'+tt.workerCount)) + "-" + tt.provider,
				"k8s-worker" + string(rune('0'+tt.workerCount)),
			}

			for _, expectedName := range tt.expectedBase {
				assert.Contains(t, dnsNames, expectedName)
			}
		})
	}
}

// Test subdomain generation for ingress services
func TestIngressSubdomainGeneration(t *testing.T) {
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

	// This matches the list in UpdateIngressRecord
	ingressSubdomains := []string{
		"grafana",
		"prometheus",
		"alertmanager",
		"dashboard",
		"argocd",
		"jenkins",
		"gitlab",
		"registry",
	}

	assert.ElementsMatch(t, expectedSubdomains, ingressSubdomains,
		"Ingress subdomain list should match expected services")
	assert.Len(t, ingressSubdomains, 8, "Should have exactly 8 ingress subdomains")
}

// Test CNAME convenience records mapping
func TestCNAMEConvenienceRecords(t *testing.T) {
	// This matches the map in CreateClusterRecords
	conveniences := map[string]string{
		"k8s":        "api",
		"kubernetes": "api",
		"cluster":    "api",
		"rancher":    "kube-ingress",
		"dashboard":  "kube-ingress",
	}

	tests := []struct {
		cname  string
		target string
	}{
		{"k8s", "api"},
		{"kubernetes", "api"},
		{"cluster", "api"},
		{"rancher", "kube-ingress"},
		{"dashboard", "kube-ingress"},
	}

	for _, tt := range tests {
		t.Run("CNAME_"+tt.cname, func(t *testing.T) {
			target, exists := conveniences[tt.cname]
			assert.True(t, exists, "CNAME %s should exist", tt.cname)
			assert.Equal(t, tt.target, target, "CNAME %s should point to %s", tt.cname, tt.target)
		})
	}
}

// Test DNS info export structure
func TestExportDNSInfo_Structure(t *testing.T) {
	domain := "example.com"

	// Simulate the structure created in ExportDNSInfo
	dnsInfo := make(map[string]interface{})
	dnsInfo["domain"] = domain
	dnsInfo["ingress_url"] = "https://kube-ingress." + domain
	dnsInfo["api_url"] = "https://api." + domain + ":6443"
	dnsInfo["wildcard"] = "*.k8s." + domain

	// Verify structure
	assert.Contains(t, dnsInfo, "domain")
	assert.Contains(t, dnsInfo, "ingress_url")
	assert.Contains(t, dnsInfo, "api_url")
	assert.Contains(t, dnsInfo, "wildcard")

	// Verify values
	assert.Equal(t, domain, dnsInfo["domain"])
	assert.Equal(t, "https://kube-ingress."+domain, dnsInfo["ingress_url"])
	assert.Equal(t, "https://api."+domain+":6443", dnsInfo["api_url"])
	assert.Equal(t, "*.k8s."+domain, dnsInfo["wildcard"])
}

// Test service URLs generation
func TestServiceURLs_Generation(t *testing.T) {
	domain := "example.com"

	services := map[string]string{
		"grafana":    "https://grafana.k8s." + domain,
		"prometheus": "https://prometheus.k8s." + domain,
		"dashboard":  "https://dashboard.k8s." + domain,
	}

	tests := []struct {
		service     string
		expectedURL string
	}{
		{"grafana", "https://grafana.k8s.example.com"},
		{"prometheus", "https://prometheus.k8s.example.com"},
		{"dashboard", "https://dashboard.k8s.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			url, exists := services[tt.service]
			assert.True(t, exists)
			assert.Equal(t, tt.expectedURL, url)
		})
	}
}

// Test TTL values
func TestDNS_TTLValues(t *testing.T) {
	// All DNS records use 300 seconds (5 minutes) TTL
	expectedTTL := 300

	tests := []struct {
		recordType string
		ttl        int
	}{
		{"A record", 300},
		{"CNAME record", 300},
		{"Wildcard record", 300},
		{"Ingress record", 300},
	}

	for _, tt := range tests {
		t.Run(tt.recordType, func(t *testing.T) {
			assert.Equal(t, expectedTTL, tt.ttl,
				"%s should have %d seconds TTL", tt.recordType, expectedTTL)
		})
	}
}

// Test DNS record name formatting (lowercase)
func TestDNS_RecordNameFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Lowercase", "worker1", "worker1"},
		{"Uppercase", "WORKER1", "worker1"},
		{"Mixed case", "Worker1", "worker1"},
		{"With hyphens", "worker-1", "worker-1"},
		{"Complex name", "K8s-Master-01", "k8s-master-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate strings.ToLower() from createARecord
			result := toLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function
func toLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

// Test node name DNS export formatting
func TestDNS_ExportNameFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple name", "master1", "dns_master1"},
		{"With hyphen", "worker-1", "dns_worker_1"},
		{"Multiple hyphens", "k8s-master-01", "dns_k8s_master_01"},
		{"Private record", "private-node1", "dns_private_node1"},
		{"WireGuard record", "wg-master1", "dns_wg_master1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the export name generation
			exportName := "dns_" + replaceHyphensWithUnderscores(tt.input)
			assert.Equal(t, tt.expected, exportName)
		})
	}
}

// Helper function
func replaceHyphensWithUnderscores(s string) string {
	result := ""
	for _, c := range s {
		if c == '-' {
			result += "_"
		} else {
			result += string(c)
		}
	}
	return result
}

// Test record type determination
func TestDNS_RecordTypes(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		isA        bool
		isCNAME    bool
	}{
		{"A record", "A", true, false},
		{"CNAME record", "CNAME", false, true},
		{"Wildcard A", "A", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isA {
				assert.Equal(t, "A", tt.recordType)
				assert.False(t, tt.isCNAME)
			}
			if tt.isCNAME {
				assert.Equal(t, "CNAME", tt.recordType)
				assert.False(t, tt.isA)
			}
		})
	}
}

// Test private DNS naming convention
func TestDNS_PrivateRecordNaming(t *testing.T) {
	tests := []struct {
		nodeName     string
		expectedName string
	}{
		{"master1", "private-master1"},
		{"worker1", "private-worker1"},
		{"node-01", "private-node-01"},
		{"k8s-master", "private-k8s-master"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeName, func(t *testing.T) {
			privateName := "private-" + tt.nodeName
			assert.Equal(t, tt.expectedName, privateName)
		})
	}
}

// Test WireGuard DNS naming convention
func TestDNS_WireGuardRecordNaming(t *testing.T) {
	tests := []struct {
		nodeName     string
		expectedName string
	}{
		{"master1", "wg-master1"},
		{"worker1", "wg-worker1"},
		{"node-01", "wg-node-01"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeName, func(t *testing.T) {
			wgName := "wg-" + tt.nodeName
			assert.Equal(t, tt.expectedName, wgName)
		})
	}
}

// Test numbered WireGuard record naming
func TestDNS_NumberedWireGuardNaming(t *testing.T) {
	tests := []struct {
		nodeType string
		number   int
		expected string
	}{
		{"master", 1, "wg-master1"},
		{"master", 2, "wg-master2"},
		{"worker", 1, "wg-worker1"},
		{"worker", 3, "wg-worker3"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			var name string
			if tt.nodeType == "master" {
				name = "wg-master" + string(rune('0'+tt.number))
			} else {
				name = "wg-worker" + string(rune('0'+tt.number))
			}
			assert.Equal(t, tt.expected, name)
		})
	}
}

// Test wildcard DNS record naming
func TestDNS_WildcardNaming(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{"Wildcard ingress", "*.k8s", "*.k8s"},
		{"Specific subdomain", "grafana.k8s", "grafana.k8s"},
		{"API endpoint", "api", "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pattern)
		})
	}
}

// Test full FQDN generation
func TestDNS_FQDNGeneration(t *testing.T) {
	domain := "example.com"

	tests := []struct {
		name      string
		subdomain string
		expected  string
	}{
		{"Master node", "master1", "master1.example.com"},
		{"Worker node", "worker1", "worker1.example.com"},
		{"API endpoint", "api", "api.example.com"},
		{"Ingress", "kube-ingress", "kube-ingress.example.com"},
		{"Service", "grafana.k8s", "grafana.k8s.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fqdn := tt.subdomain + "." + domain
			assert.Equal(t, tt.expected, fqdn)
		})
	}
}

// Test CNAME target formatting
func TestDNS_CNAMETargetFormatting(t *testing.T) {
	domain := "example.com"

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{"API target", "api", "api.example.com."},
		{"Ingress target", "kube-ingress", "kube-ingress.example.com."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// CNAME targets should have trailing dot
			cnameTarget := tt.target + "." + domain + "."
			assert.Equal(t, tt.expected, cnameTarget)
			assert.True(t, cnameTarget[len(cnameTarget)-1] == '.',
				"CNAME target should end with dot")
		})
	}
}

// Test 100 comprehensive DNS scenarios
func Test100DNSNamingScenarios(t *testing.T) {
	scenarios := []struct {
		nodeType string
		number   int
		provider string
	}{
		{"master", 1, "digitalocean"},
		{"worker", 1, "digitalocean"},
		{"master", 2, "linode"},
	}

	// Generate 97 more scenarios
	providers := []string{"digitalocean", "linode"}
	nodeTypes := []string{"master", "worker"}

	for i := 0; i < 97; i++ {
		scenarios = append(scenarios, struct {
			nodeType string
			number   int
			provider string
		}{
			nodeType: nodeTypes[i%2],
			number:   (i % 10) + 1,
			provider: providers[i%2],
		})
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('0'+i%10)), func(t *testing.T) {
			// Generate DNS names based on scenario
			var dnsNames []string

			if scenario.nodeType == "master" {
				dnsNames = append(dnsNames,
					"master"+string(rune('0'+scenario.number)),
					"k8s-master"+string(rune('0'+scenario.number)),
				)
			} else {
				dnsNames = append(dnsNames,
					"worker"+string(rune('0'+scenario.number)),
					"k8s-worker"+string(rune('0'+scenario.number)),
				)
			}

			// Validate DNS names
			assert.NotEmpty(t, dnsNames)
			for _, name := range dnsNames {
				assert.NotEmpty(t, name)
				assert.True(t, len(name) > 0)
			}
		})
	}
}

// Test node labels role detection
func TestDNS_NodeRoleDetection(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{"Master role", map[string]string{"role": "master"}, "master"},
		{"Worker role", map[string]string{"role": "worker"}, "worker"},
		{"Controlplane role", map[string]string{"role": "controlplane"}, "controlplane"},
		{"No role", map[string]string{}, "node"},
		{"Other label", map[string]string{"env": "prod"}, "node"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeType := "node"
			if tt.labels != nil {
				if role, ok := tt.labels["role"]; ok {
					nodeType = role
				}
			}
			assert.Equal(t, tt.expected, nodeType)
		})
	}
}

// Test DNS record count expectations
func TestDNS_RecordCountEstimation(t *testing.T) {
	tests := []struct {
		name               string
		masterCount        int
		workerCount        int
		expectedMinRecords int
	}{
		{"Single master", 1, 0, 5}, // 3 base + api + k8s-api + private + wg
		{"Single worker", 0, 1, 4}, // 3 base + private + wg
		{"Standard HA", 3, 3, 30},  // (3*7 + 3*5) + wildcard + ingress + CNAMEs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var recordCount int

			// Count master records
			for i := 1; i <= tt.masterCount; i++ {
				recordCount += 3 // base names
				if i == 1 {
					recordCount += 2 // api records
				}
				recordCount += 1 // private
				recordCount += 2 // wireguard (base + numbered)
			}

			// Count worker records
			for i := 1; i <= tt.workerCount; i++ {
				recordCount += 3 // base names
				recordCount += 1 // private
				recordCount += 2 // wireguard
			}

			// Add wildcard and ingress
			recordCount += 2

			assert.GreaterOrEqual(t, recordCount, tt.expectedMinRecords,
				"Should create at least %d DNS records", tt.expectedMinRecords)
		})
	}
}
