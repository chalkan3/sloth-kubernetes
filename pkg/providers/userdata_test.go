package providers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sloth-kubernetes/pkg/config"
)

// Test generateUserData for DigitalOcean
func TestDigitalOceanProvider_GenerateUserDataDetailed(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	tests := []struct {
		name     string
		node     *config.NodeConfig
		contains []string
	}{
		{
			"Basic node",
			&config.NodeConfig{
				Name:   "node1",
				Region: "nyc3",
				Size:   "s-2vcpu-4gb",
				Roles:  []string{},
			},
			[]string{
				"#!/bin/bash",
				"apt-get update",
				"Install Docker",
				"wireguard",
				"NODE_PROVIDER=digitalocean",
				"NODE_REGION=nyc3",
				"NODE_SIZE=s-2vcpu-4gb",
			},
		},
		{
			"Master node with role",
			&config.NodeConfig{
				Name:   "master1",
				Region: "sfo3",
				Size:   "s-4vcpu-8gb",
				Roles:  []string{"master", "controlplane"},
			},
			[]string{
				"NODE_ROLE_master=true",
				"NODE_ROLE_controlplane=true",
				"NODE_REGION=sfo3",
				"NODE_SIZE=s-4vcpu-8gb",
			},
		},
		{
			"Node with custom userData",
			&config.NodeConfig{
				Name:     "worker1",
				Region:   "ams3",
				Size:     "s-2vcpu-2gb",
				Roles:    []string{"worker"},
				UserData: "echo 'Custom script here'",
			},
			[]string{
				"# Custom user data",
				"echo 'Custom script here'",
				"NODE_ROLE_worker=true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userData := provider.generateUserData(tt.node)

			assert.NotEmpty(t, userData)
			for _, expected := range tt.contains {
				assert.Contains(t, userData, expected,
					"UserData should contain: %s", expected)
			}
		})
	}
}

// Test generateUserData with provider-level userData
func TestDigitalOceanProvider_GenerateUserData_ProviderLevel(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region:   "nyc3",
			UserData: "echo 'Provider-level custom script'",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	assert.Contains(t, userData, "# Provider custom user data")
	assert.Contains(t, userData, "echo 'Provider-level custom script'")
}

// Test generateUserData node-level overrides provider-level
func TestDigitalOceanProvider_GenerateUserData_NodeOverridesProvider(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region:   "nyc3",
			UserData: "echo 'Provider script'",
		},
	}

	node := &config.NodeConfig{
		Name:     "node1",
		Region:   "nyc3",
		Size:     "s-2vcpu-4gb",
		Roles:    []string{},
		UserData: "echo 'Node script'",
	}

	userData := provider.generateUserData(node)

	// Node userData should be present
	assert.Contains(t, userData, "echo 'Node script'")
	// Provider userData should NOT be present (overridden)
	assert.NotContains(t, userData, "echo 'Provider script'")
}

// Test UserData script structure
func TestDigitalOceanProvider_UserDataStructure(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "test-node",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{"master"},
	}

	userData := provider.generateUserData(node)

	// Should start with shebang
	assert.True(t, strings.HasPrefix(userData, "#!/bin/bash"))

	// Should contain critical sections
	sections := []string{
		"set -e",
		"apt-get update",
		"Install required packages",
		"Install Docker",
		"Disable swap",
		"Enable IP forwarding",
		"Configure WireGuard",
		"Generate WireGuard keys",
		"Set node labels",
		"Node initialization complete",
	}

	for _, section := range sections {
		assert.Contains(t, userData, section)
	}
}

// Test UserData role handling
func TestDigitalOceanProvider_UserDataRoles(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	tests := []struct {
		name          string
		roles         []string
		expectedLines []string
	}{
		{
			"No roles",
			[]string{},
			[]string{},
		},
		{
			"Single role",
			[]string{"master"},
			[]string{"NODE_ROLE_master=true"},
		},
		{
			"Multiple roles",
			[]string{"master", "controlplane", "etcd"},
			[]string{
				"NODE_ROLE_master=true",
				"NODE_ROLE_controlplane=true",
				"NODE_ROLE_etcd=true",
			},
		},
		{
			"Worker role",
			[]string{"worker"},
			[]string{"NODE_ROLE_worker=true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &config.NodeConfig{
				Name:   "node1",
				Region: "nyc3",
				Size:   "s-2vcpu-4gb",
				Roles:  tt.roles,
			}

			userData := provider.generateUserData(node)

			for _, expectedLine := range tt.expectedLines {
				assert.Contains(t, userData, expectedLine)
			}
		})
	}
}

// Test UserData required packages
func TestDigitalOceanProvider_UserDataPackages(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	requiredPackages := []string{
		"curl",
		"wget",
		"git",
		"vim",
		"htop",
		"net-tools",
		"wireguard",
		"wireguard-tools",
	}

	for _, pkg := range requiredPackages {
		assert.Contains(t, userData, pkg,
			"UserData should install package: %s", pkg)
	}
}

// Test UserData system configuration
func TestDigitalOceanProvider_UserDataSystemConfig(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	// Check critical system configurations
	configs := []string{
		"swapoff -a",                           // Disable swap
		"sed -i '/ swap / s/^\\(.*\\)$/#\\1/g' /etc/fstab", // Comment swap in fstab
		"net.ipv4.ip_forward=1",                // IPv4 forwarding
		"net.ipv6.conf.all.forwarding=1",       // IPv6 forwarding
		"sysctl -p",                            // Apply sysctl changes
	}

	for _, cfg := range configs {
		assert.Contains(t, userData, cfg)
	}
}

// Test UserData WireGuard setup
func TestDigitalOceanProvider_UserDataWireGuard(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	// Check WireGuard configuration
	wgSteps := []string{
		"mkdir -p /etc/wireguard",
		"chmod 700 /etc/wireguard",
		"wg genkey",
		"tee /etc/wireguard/privatekey",
		"wg pubkey",
		"/etc/wireguard/publickey",
		"chmod 600 /etc/wireguard/privatekey",
	}

	for _, step := range wgSteps {
		assert.Contains(t, userData, step)
	}
}

// Test UserData Docker installation
func TestDigitalOceanProvider_UserDataDocker(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	// Check Docker installation steps
	dockerSteps := []string{
		"curl -fsSL https://get.docker.com -o get-docker.sh",
		"sh get-docker.sh",
		"usermod -aG docker root",
		"systemctl enable docker",
		"systemctl start docker",
	}

	for _, step := range dockerSteps {
		assert.Contains(t, userData, step)
	}
}

// Test 50 UserData generation scenarios
func Test50UserDataScenarios(t *testing.T) {
	scenarios := []struct {
		region   string
		size     string
		roles    []string
		userData string
	}{
		{"nyc3", "s-2vcpu-4gb", []string{"master"}, ""},
		{"sfo3", "s-4vcpu-8gb", []string{"worker"}, "echo 'custom'"},
		{"ams3", "s-2vcpu-2gb", []string{"master", "controlplane"}, ""},
	}

	// Generate 47 more scenarios
	regions := []string{"nyc3", "sfo3", "ams3", "sgp1", "lon1", "fra1"}
	sizes := []string{"s-1vcpu-1gb", "s-2vcpu-2gb", "s-2vcpu-4gb", "s-4vcpu-8gb"}
	roleSets := [][]string{
		{"master"},
		{"worker"},
		{"master", "controlplane"},
		{"master", "controlplane", "etcd"},
		{},
	}

	for i := 0; i < 47; i++ {
		scenario := struct {
			region   string
			size     string
			roles    []string
			userData string
		}{
			region:   regions[i%len(regions)],
			size:     sizes[i%len(sizes)],
			roles:    roleSets[i%len(roleSets)],
			userData: "",
		}
		if i%5 == 0 {
			scenario.userData = "echo 'custom script'"
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			provider := &DigitalOceanProvider{
				config: &config.DigitalOceanProvider{
					Region: "nyc3",
				},
			}

			node := &config.NodeConfig{
				Name:     "node-" + string(rune('0'+i)),
				Region:   scenario.region,
				Size:     scenario.size,
				Roles:    scenario.roles,
				UserData: scenario.userData,
			}

			userData := provider.generateUserData(node)

			// Validate basic structure
			assert.NotEmpty(t, userData)
			assert.Contains(t, userData, "#!/bin/bash")
			assert.Contains(t, userData, "NODE_REGION="+scenario.region)
			assert.Contains(t, userData, "NODE_SIZE="+scenario.size)

			// Validate roles
			for _, role := range scenario.roles {
				assert.Contains(t, userData, "NODE_ROLE_"+role+"=true")
			}

			// Validate custom userData
			if scenario.userData != "" {
				assert.Contains(t, userData, scenario.userData)
			}
		})
	}
}

// Test UserData completion message
func TestDigitalOceanProvider_UserDataCompletion(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	// Should end with completion message
	assert.Contains(t, userData, "Node initialization complete")
}

// Test UserData environment variables format
func TestDigitalOceanProvider_UserDataEnvVars(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "sfo3",
		Size:   "s-4vcpu-8gb",
		Roles:  []string{"master"},
	}

	userData := provider.generateUserData(node)

	// Check environment variable format
	envVars := []string{
		"NODE_PROVIDER=digitalocean",
		"NODE_REGION=sfo3",
		"NODE_SIZE=s-4vcpu-8gb",
		"NODE_ROLE_master=true",
	}

	for _, envVar := range envVars {
		// Should append to /etc/environment (can be with single or double quotes)
		assert.True(t,
			strings.Contains(userData, "echo \""+envVar+"\" >> /etc/environment") ||
			strings.Contains(userData, "echo '"+envVar+"' >> /etc/environment"),
			"UserData should contain env var: %s", envVar)
	}
}

// Test UserData security settings
func TestDigitalOceanProvider_UserDataSecurity(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
	}

	userData := provider.generateUserData(node)

	// Check security-related configurations
	securitySettings := []string{
		"chmod 700 /etc/wireguard",       // Secure WireGuard directory
		"chmod 600 /etc/wireguard/privatekey", // Secure private key
	}

	for _, setting := range securitySettings {
		assert.Contains(t, userData, setting)
	}
}

// Test UserData with special characters in custom script
func TestDigitalOceanProvider_UserDataSpecialCharacters(t *testing.T) {
	provider := &DigitalOceanProvider{
		config: &config.DigitalOceanProvider{
			Region: "nyc3",
		},
	}

	node := &config.NodeConfig{
		Name:   "node1",
		Region: "nyc3",
		Size:   "s-2vcpu-4gb",
		Roles:  []string{},
		UserData: "echo 'Test with \"quotes\" and $variables'",
	}

	userData := provider.generateUserData(node)

	// Custom userData should be included as-is
	assert.Contains(t, userData, "echo 'Test with \"quotes\" and $variables'")
}
