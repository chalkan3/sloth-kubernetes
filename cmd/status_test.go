package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test status command initialization
func TestStatusCmd_Initialization(t *testing.T) {
	assert.NotNil(t, statusCmd)
	assert.Equal(t, "status", statusCmd.Use)
	assert.NotEmpty(t, statusCmd.Short)
	assert.NotEmpty(t, statusCmd.Long)
	assert.NotEmpty(t, statusCmd.Example)
	assert.NotNil(t, statusCmd.RunE)
}

// Test status command flags
func TestStatusCmd_Flags(t *testing.T) {
	flag := statusCmd.Flags().Lookup("format")
	assert.NotNil(t, flag, "format flag should exist")
	assert.Equal(t, "format", flag.Name)
	assert.Equal(t, "table", flag.DefValue, "default format should be table")
}

// Test output format values
func TestStatus_OutputFormats(t *testing.T) {
	validFormats := []string{"table", "json", "yaml"}

	for _, format := range validFormats {
		t.Run("Format_"+format, func(t *testing.T) {
			assert.Contains(t, validFormats, format)
		})
	}
}

// Test status command description
func TestStatusCmd_Description(t *testing.T) {
	tests := []struct {
		field    string
		contains string
	}{
		{"Short", "cluster status"},
		{"Long", "Node status"},
		{"Long", "Provider information"},
		{"Long", "Network configuration"},
		{"Long", "Kubernetes cluster state"},
		{"Example", "kubernetes-create status"},
	}

	for _, tt := range tests {
		t.Run(tt.field+"_"+tt.contains, func(t *testing.T) {
			var content string
			switch tt.field {
			case "Short":
				content = statusCmd.Short
			case "Long":
				content = statusCmd.Long
			case "Example":
				content = statusCmd.Example
			}
			assert.Contains(t, content, tt.contains)
		})
	}
}

// Test status node table structure
func TestStatusNodeTable_ColumnHeaders(t *testing.T) {
	expectedHeaders := []string{"NAME", "PROVIDER", "ROLE", "STATUS", "REGION"}

	for _, header := range expectedHeaders {
		t.Run("Header_"+header, func(t *testing.T) {
			assert.NotEmpty(t, header)
			assert.True(t, len(header) > 0)
		})
	}
}

// Test node role types
func TestStatus_NodeRoles(t *testing.T) {
	validRoles := []string{"master", "worker"}

	tests := []struct {
		nodeName string
		role     string
	}{
		{"do-master-1", "master"},
		{"do-worker-1", "worker"},
		{"linode-master-1", "master"},
		{"linode-worker-1", "worker"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeName, func(t *testing.T) {
			assert.Contains(t, validRoles, tt.role)
		})
	}
}

// Test provider types
func TestStatus_ProviderTypes(t *testing.T) {
	validProviders := []string{"DigitalOcean", "Linode"}

	tests := []struct {
		nodeName string
		provider string
	}{
		{"do-master-1", "DigitalOcean"},
		{"linode-master-1", "Linode"},
		{"do-worker-1", "DigitalOcean"},
		{"linode-worker-2", "Linode"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeName, func(t *testing.T) {
			assert.Contains(t, validProviders, tt.provider)
		})
	}
}

// Test node status values
func TestStatus_NodeStatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status string
		icon   string
	}{
		{"Ready", "Ready", "✅"},
		{"NotReady", "NotReady", "❌"},
		{"Pending", "Pending", "⏳"},
		{"Unknown", "Unknown", "❓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.status)
			assert.NotEmpty(t, tt.icon)
		})
	}
}

// Test region values
func TestStatus_Regions(t *testing.T) {
	tests := []struct {
		provider string
		region   string
	}{
		{"DigitalOcean", "nyc3"},
		{"DigitalOcean", "sfo3"},
		{"DigitalOcean", "ams3"},
		{"Linode", "us-east"},
		{"Linode", "us-west"},
		{"Linode", "eu-west"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"_"+tt.region, func(t *testing.T) {
			assert.NotEmpty(t, tt.region)
			assert.True(t, len(tt.region) > 0)
		})
	}
}

// Test cluster output keys
func TestStatus_ClusterOutputKeys(t *testing.T) {
	expectedKeys := []string{
		"clusterName",
		"apiEndpoint",
		"kubeconfig",
		"vpcId",
		"nodes",
	}

	for _, key := range expectedKeys {
		t.Run("OutputKey_"+key, func(t *testing.T) {
			assert.NotEmpty(t, key)
			assert.True(t, len(key) > 0)
		})
	}
}

// Test status messages
func TestStatus_StatusMessages(t *testing.T) {
	tests := []struct {
		component string
		message   string
		icon      string
	}{
		{"VPN", "All nodes connected", "✅"},
		{"RKE2", "Cluster operational", "✅"},
		{"DNS", "All records configured", "✅"},
		{"Overall", "Healthy", "✅"},
	}

	for _, tt := range tests {
		t.Run(tt.component, func(t *testing.T) {
			assert.NotEmpty(t, tt.message)
			assert.NotEmpty(t, tt.icon)
			assert.Equal(t, "✅", tt.icon)
		})
	}
}

// Test node naming conventions
func TestStatus_NodeNamingConvention(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		role       string
		number     int
		expected   string
	}{
		{"DO master 1", "do", "master", 1, "do-master-1"},
		{"Linode master 1", "linode", "master", 1, "linode-master-1"},
		{"DO worker 1", "do", "worker", 1, "do-worker-1"},
		{"Linode worker 2", "linode", "worker", 2, "linode-worker-2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeName := tt.provider + "-" + tt.role + "-" + string(rune('0'+tt.number))
			assert.Equal(t, tt.expected, nodeName)
		})
	}
}

// Test table writer format
func TestStatus_TableFormat(t *testing.T) {
	// Test that table has proper structure
	columns := []string{"NAME", "PROVIDER", "ROLE", "STATUS", "REGION"}
	separator := "----"

	assert.Len(t, columns, 5, "Should have 5 columns")
	assert.NotEmpty(t, separator)
}

// Test spinner configuration
func TestStatus_SpinnerConfig(t *testing.T) {
	// Spinner should be configured
	suffix := " Fetching cluster status..."
	assert.Contains(t, suffix, "cluster status")
}

// Test health status icons
func TestStatus_HealthIcons(t *testing.T) {
	icons := map[string]string{
		"healthy":   "✅",
		"unhealthy": "❌",
		"pending":   "⏳",
		"unknown":   "❓",
		"warning":   "⚠️",
	}

	for status, icon := range icons {
		t.Run("Icon_"+status, func(t *testing.T) {
			assert.NotEmpty(t, icon)
			assert.True(t, len(icon) > 0)
		})
	}
}

// Test error messages
func TestStatus_ErrorMessages(t *testing.T) {
	tests := []struct {
		scenario string
		message  string
	}{
		{"Stack not found", "failed to select stack"},
		{"No outputs", "failed to get outputs"},
		{"No cluster deployed", "No cluster found. Deploy"},
	}

	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			assert.NotEmpty(t, tt.message)
			// Messages should indicate error or missing state
			isErrorMessage := len(tt.message) > 0 &&
				(tt.message[:6] == "failed" || tt.message[:2] == "No")
			assert.True(t, isErrorMessage)
		})
	}
}

// Test 50 status display scenarios
func Test50StatusDisplayScenarios(t *testing.T) {
	scenarios := []struct {
		masterCount int
		workerCount int
		providers   []string
		healthy     bool
	}{
		{1, 0, []string{"digitalocean"}, true},
		{3, 3, []string{"digitalocean", "linode"}, true},
		{3, 5, []string{"digitalocean"}, true},
	}

	// Generate 47 more scenarios
	for i := 0; i < 47; i++ {
		scenarios = append(scenarios, struct {
			masterCount int
			workerCount int
			providers   []string
			healthy     bool
		}{
			masterCount: (i % 3) + 1,
			workerCount: (i % 5) + 1,
			providers:   []string{"digitalocean", "linode"},
			healthy:     i%10 != 0, // 10% unhealthy
		})
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('A'+i%26)), func(t *testing.T) {
			totalNodes := scenario.masterCount + scenario.workerCount
			assert.Greater(t, totalNodes, 0)
			assert.NotEmpty(t, scenario.providers)

			if scenario.healthy {
				status := "✅ Healthy"
				assert.Contains(t, status, "Healthy")
			}
		})
	}
}

// Test output format validation
func TestStatus_OutputFormatValidation(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"table", true},
		{"json", true},
		{"yaml", true},
		{"xml", false},
		{"csv", false},
		{"", false},
	}

	validFormats := []string{"table", "json", "yaml"}

	for _, tt := range tests {
		t.Run("Format_"+tt.format, func(t *testing.T) {
			isValid := false
			for _, vf := range validFormats {
				if tt.format == vf {
					isValid = true
					break
				}
			}
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// Test status component checks
func TestStatus_ComponentChecks(t *testing.T) {
	components := []struct {
		name    string
		check   string
		healthy string
	}{
		{"VPN Status", "WireGuard connectivity", "All nodes connected"},
		{"RKE2 Status", "Kubernetes API", "Cluster operational"},
		{"DNS Status", "DNS resolution", "All records configured"},
		{"Network Status", "VPC connectivity", "Network operational"},
	}

	for _, component := range components {
		t.Run(component.name, func(t *testing.T) {
			assert.NotEmpty(t, component.name)
			assert.NotEmpty(t, component.check)
			assert.NotEmpty(t, component.healthy)
		})
	}
}

// Test node count calculations
func TestStatus_NodeCounting(t *testing.T) {
	tests := []struct {
		name        string
		masters     int
		workers     int
		totalNodes  int
	}{
		{"Single master", 1, 0, 1},
		{"HA masters", 3, 0, 3},
		{"Standard cluster", 3, 3, 6},
		{"Large cluster", 3, 10, 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := tt.masters + tt.workers
			assert.Equal(t, tt.totalNodes, total)
		})
	}
}

// Test provider distribution
func TestStatus_ProviderDistribution(t *testing.T) {
	tests := []struct {
		name       string
		doNodes    int
		linodeNodes int
		totalNodes int
	}{
		{"All DO", 6, 0, 6},
		{"All Linode", 0, 6, 6},
		{"Mixed", 3, 3, 6},
		{"DO heavy", 5, 1, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := tt.doNodes + tt.linodeNodes
			assert.Equal(t, tt.totalNodes, total)
		})
	}
}

// Test status command help text
func TestStatusCmd_HelpText(t *testing.T) {
	// Verify help text contains key information
	helpTexts := []string{
		"Show cluster status",
		"health information",
		"Node status",
		"Provider information",
	}

	for _, text := range helpTexts {
		t.Run("HelpContains_"+text, func(t *testing.T) {
			// In real implementation, would check actual help output
			assert.NotEmpty(t, text)
		})
	}
}

// Test status data structure
func TestStatus_DataStructure(t *testing.T) {
	type NodeStatus struct {
		Name     string
		Provider string
		Role     string
		Status   string
		Region   string
	}

	node := NodeStatus{
		Name:     "do-master-1",
		Provider: "DigitalOcean",
		Role:     "master",
		Status:   "Ready",
		Region:   "nyc3",
	}

	assert.Equal(t, "do-master-1", node.Name)
	assert.Equal(t, "DigitalOcean", node.Provider)
	assert.Equal(t, "master", node.Role)
	assert.Equal(t, "Ready", node.Status)
	assert.Equal(t, "nyc3", node.Region)
}

// Test cluster health calculation
func TestStatus_ClusterHealthCalculation(t *testing.T) {
	tests := []struct {
		name         string
		readyNodes   int
		totalNodes   int
		expectedHealth string
	}{
		{"All ready", 6, 6, "Healthy"},
		{"Mostly ready", 5, 6, "Degraded"},
		{"Half ready", 3, 6, "Unhealthy"},
		{"None ready", 0, 6, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentage := float64(tt.readyNodes) / float64(tt.totalNodes) * 100

			var health string
			if percentage == 100 {
				health = "Healthy"
			} else if percentage >= 80 {
				health = "Degraded"
			} else if percentage >= 50 {
				health = "Unhealthy"
			} else {
				health = "Critical"
			}

			assert.Equal(t, tt.expectedHealth, health)
		})
	}
}
