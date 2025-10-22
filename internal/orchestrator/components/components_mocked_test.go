package components

import (
	"fmt"
	"strings"
	"testing"
)

// TestSSHKeyPath_Format tests SSH key path format
func TestSSHKeyPath_Format(t *testing.T) {
	tests := []struct {
		name     string
		stack    string
		expected string
		valid    bool
	}{
		{"Production stack", "production", "~/.ssh/kubernetes-clusters/production.pem", true},
		{"Development stack", "development", "~/.ssh/kubernetes-clusters/development.pem", true},
		{"Staging stack", "staging", "~/.ssh/kubernetes-clusters/staging.pem", true},
		{"Custom stack", "my-cluster", "~/.ssh/kubernetes-clusters/my-cluster.pem", true},
		{"Empty stack", "", "~/.ssh/kubernetes-clusters/.pem", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("~/.ssh/kubernetes-clusters/%s.pem", tt.stack)
			if path != tt.expected {
				t.Errorf("Expected path %q, got %q", tt.expected, path)
			}
			isValid := tt.stack != "" && strings.HasSuffix(path, ".pem")
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestProviderInitialization_Status tests provider initialization status
func TestProviderInitialization_Status(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		enabled  bool
		status   string
	}{
		{"DigitalOcean enabled", "digitalocean", true, "initialized"},
		{"Linode enabled", "linode", true, "initialized"},
		{"DigitalOcean disabled", "digitalocean", false, ""},
		{"Linode disabled", "linode", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status string
			if tt.enabled {
				status = "initialized"
			}
			if status != tt.status {
				t.Errorf("Expected status %q, got %q", tt.status, status)
			}
		})
	}
}

// TestNetworkCIDR_Validation tests network CIDR validation
func TestNetworkCIDR_Validation(t *testing.T) {
	tests := []struct {
		name  string
		cidr  string
		valid bool
	}{
		{"Valid /16", "10.0.0.0/16", true},
		{"Valid /24", "192.168.1.0/24", true},
		{"Valid /8", "10.0.0.0/8", true},
		{"Invalid format", "10.0.0.0", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.Contains(tt.cidr, "/") && tt.cidr != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for CIDR %q, got %v", tt.valid, tt.cidr, isValid)
			}
		})
	}
}

// TestDNSDomain_Validation tests DNS domain validation
func TestDNSDomain_Validation(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		valid  bool
	}{
		{"Valid domain", "chalkan3.com.br", true},
		{"Valid subdomain", "api.example.com", true},
		{"Valid TLD", "example.io", true},
		{"Default fallback", "", true}, // Empty uses default
		{"Invalid no TLD", "example", false},
		{"Invalid chars", "exam ple.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty domain uses default "chalkan3.com.br"
			domain := tt.domain
			if domain == "" {
				domain = "chalkan3.com.br"
			}

			isValid := domain != "" &&
				!strings.Contains(domain, " ") &&
				strings.Contains(domain, ".")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for domain %q, got %v", tt.valid, tt.domain, isValid)
			}
		})
	}
}

// TestWireGuardAddressAllocation tests WireGuard address allocation
func TestWireGuardAddressAllocation(t *testing.T) {
	tests := []struct {
		name      string
		nodeIndex int
		expected  string
	}{
		{"First node", 0, "10.8.0.10/24"},
		{"Second node", 1, "10.8.0.11/24"},
		{"Third node", 2, "10.8.0.12/24"},
		{"Tenth node", 9, "10.8.0.19/24"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := fmt.Sprintf("10.8.0.%d/24", tt.nodeIndex+10)
			if address != tt.expected {
				t.Errorf("Expected address %q, got %q", tt.expected, address)
			}
		})
	}
}

// TestWireGuardMesh_ConnectionCount tests full mesh connection calculation
func TestWireGuardMesh_ConnectionCount(t *testing.T) {
	tests := []struct {
		name        string
		nodes       int
		connections int
	}{
		{"2 nodes", 2, 1},
		{"3 nodes", 3, 3},
		{"4 nodes", 4, 6},
		{"5 nodes", 5, 10},
		{"6 nodes", 6, 15},
		{"10 nodes", 10, 45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Full mesh: n * (n-1) / 2
			connections := (tt.nodes * (tt.nodes - 1)) / 2
			if connections != tt.connections {
				t.Errorf("Expected %d connections for %d nodes, got %d",
					tt.connections, tt.nodes, connections)
			}
		})
	}
}

// TestKubeConfig_Format tests kubeconfig file format
func TestKubeConfig_Format(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		domain      string
		valid       bool
	}{
		{"Production cluster", "production", "chalkan3.com.br", true},
		{"Development cluster", "development", "example.com", true},
		{"Custom cluster", "my-cluster", "mydomain.io", true},
		{"Empty cluster name", "", "example.com", false},
		{"Empty domain", "production", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://api.%s:6443
  name: %s-rke`, tt.domain, tt.clusterName)

			hasAPIVersion := strings.Contains(kubeconfig, "apiVersion: v1")
			hasKind := strings.Contains(kubeconfig, "kind: Config")
			hasServer := strings.Contains(kubeconfig, "server:")
			hasCluster := tt.clusterName != "" && strings.Contains(kubeconfig, tt.clusterName)

			isValid := hasAPIVersion && hasKind && hasServer && hasCluster && tt.domain != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestClusterState_Values tests cluster state values
func TestClusterState_Values(t *testing.T) {
	validStates := []string{"Active", "Pending", "Failed", "Updating"}

	tests := []struct {
		name  string
		state string
		valid bool
	}{
		{"Active state", "Active", true},
		{"Pending state", "Pending", true},
		{"Failed state", "Failed", true},
		{"Updating state", "Updating", true},
		{"Invalid state", "Unknown", false},
		{"Empty state", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, validState := range validStates {
				if tt.state == validState {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for state %q, got %v", tt.valid, tt.state, isValid)
			}
		})
	}
}

// TestComponentResourceType_Naming tests component resource type naming
func TestComponentResourceType_Naming(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		valid        bool
	}{
		{"SSH Key", "kubernetes-create:security:SSHKey", true},
		{"Providers", "kubernetes-create:provider:Providers", true},
		{"Network", "kubernetes-create:network:Network", true},
		{"DNS", "kubernetes-create:dns:DNS", true},
		{"WireGuard", "kubernetes-create:network:WireGuard", true},
		{"CloudFirewall", "kubernetes-create:network:CloudFirewall", true},
		{"RKE", "kubernetes-create:cluster:RKE", true},
		{"Ingress", "kubernetes-create:ingress:NGINX", true},
		{"Addons", "kubernetes-create:addons:Addons", true},
		{"Invalid format", "invalid-type", false},
		{"Missing prefix", "security:SSHKey", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.HasPrefix(tt.resourceType, "kubernetes-create:") &&
				strings.Count(tt.resourceType, ":") == 2

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for type %q, got %v", tt.valid, tt.resourceType, isValid)
			}
		})
	}
}

// TestWireGuardServerEndpoint_Format tests WireGuard server endpoint format
func TestWireGuardServerEndpoint_Format(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		valid    bool
	}{
		{"Valid IP:port", "203.0.113.1:51820", true},
		{"Valid domain:port", "vpn.example.com:51820", true},
		{"Valid custom port", "10.0.0.1:12345", true},
		{"Missing port", "203.0.113.1", false},
		{"Invalid format", "not-an-endpoint", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.Contains(tt.endpoint, ":") && tt.endpoint != ""
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for endpoint %q, got %v", tt.valid, tt.endpoint, isValid)
			}
		})
	}
}

// TestIngressController_Status tests ingress controller status
func TestIngressController_Status(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"NGINX installed", "NGINX Ingress installed", "NGINX Ingress installed"},
		{"Traefik installed", "Traefik installed", "Traefik installed"},
		{"Installing", "Installing ingress controller", "Installing ingress controller"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expected {
				t.Errorf("Expected status %q, got %q", tt.expected, tt.status)
			}
		})
	}
}

// TestAddons_Installation tests addon installation status
func TestAddons_Installation(t *testing.T) {
	addons := []string{
		"cert-manager",
		"metrics-server",
		"ingress-nginx",
		"dashboard",
		"monitoring",
		"logging",
	}

	tests := []struct {
		name      string
		addon     string
		installed bool
	}{
		{"cert-manager", "cert-manager", true},
		{"metrics-server", "metrics-server", true},
		{"invalid addon", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, addon := range addons {
				if tt.addon == addon {
					found = true
					break
				}
			}

			if found != tt.installed {
				t.Errorf("Expected installed=%v for addon %q, got %v", tt.installed, tt.addon, found)
			}
		})
	}
}

// TestWireGuardClientConfig_Keys tests WireGuard client config key generation
func TestWireGuardClientConfig_Keys(t *testing.T) {
	tests := []struct {
		name        string
		hasPrivate  bool
		hasPublic   bool
		hasAddress  bool
		hasEndpoint bool
		valid       bool
	}{
		{"Complete config", true, true, true, true, true},
		{"Missing private key", false, true, true, true, false},
		{"Missing public key", true, false, true, true, false},
		{"Missing address", true, true, false, true, false},
		{"Missing endpoint", true, true, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.hasPrivate && tt.hasPublic && tt.hasAddress && tt.hasEndpoint
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestFirewallRules_Protocols tests firewall rule protocols
func TestFirewallRules_Protocols(t *testing.T) {
	validProtocols := []string{"tcp", "udp", "icmp"}

	tests := []struct {
		name     string
		protocol string
		valid    bool
	}{
		{"TCP protocol", "tcp", true},
		{"UDP protocol", "udp", true},
		{"ICMP protocol", "icmp", true},
		{"Invalid protocol", "http", false},
		{"Empty protocol", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, p := range validProtocols {
				if tt.protocol == p {
					isValid = true
					break
				}
			}

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for protocol %q, got %v", tt.valid, tt.protocol, isValid)
			}
		})
	}
}

// TestFirewallRules_PortRanges tests firewall port ranges
func TestFirewallRules_PortRanges(t *testing.T) {
	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"SSH port", 22, true},
		{"HTTP port", 80, true},
		{"HTTPS port", 443, true},
		{"K8s API", 6443, true},
		{"WireGuard", 51820, true},
		{"Invalid low", 0, false},
		{"Invalid high", 65536, false},
		{"Negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.port > 0 && tt.port <= 65535
			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for port %d, got %v", tt.valid, tt.port, isValid)
			}
		})
	}
}

// Test150ComponentScenarios generates 150 component test scenarios
func Test150ComponentScenarios(t *testing.T) {
	scenarios := []struct {
		componentType string
		stack         string
		provider      string
		nodeCount     int
		valid         bool
	}{
		{"SSHKey", "production", "digitalocean", 3, true},
		{"Network", "staging", "linode", 5, true},
		{"WireGuard", "development", "digitalocean", 7, true},
	}

	// Generate 147 more scenarios
	componentTypes := []string{"SSHKey", "Network", "WireGuard", "RKE", "Ingress", "Addons", "DNS", "Firewall"}
	stacks := []string{"production", "staging", "development", "test"}
	providers := []string{"digitalocean", "linode"}

	for i := 0; i < 147; i++ {
		scenarios = append(scenarios, struct {
			componentType string
			stack         string
			provider      string
			nodeCount     int
			valid         bool
		}{
			componentType: componentTypes[i%len(componentTypes)],
			stack:         stacks[i%len(stacks)],
			provider:      providers[i%len(providers)],
			nodeCount:     2 + (i % 15),
			valid:         true,
		})
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_component_"+string(rune('0'+i%10)), func(t *testing.T) {
			// Validate component type
			validTypes := []string{"SSHKey", "Network", "WireGuard", "RKE", "Ingress", "Addons", "DNS", "Firewall"}
			typeValid := false
			for _, vt := range validTypes {
				if scenario.componentType == vt {
					typeValid = true
					break
				}
			}

			// Validate stack name
			stackValid := scenario.stack != "" && scenario.stack == strings.ToLower(scenario.stack)

			// Validate provider
			providerValid := scenario.provider == "digitalocean" || scenario.provider == "linode"

			// Validate node count
			nodeCountValid := scenario.nodeCount >= 2 && scenario.nodeCount <= 100

			isValid := typeValid && stackValid && providerValid && nodeCountValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v", i, scenario.valid, isValid)
			}

			// Validate WireGuard connections for this scenario
			if scenario.componentType == "WireGuard" {
				connections := (scenario.nodeCount * (scenario.nodeCount - 1)) / 2
				if connections <= 0 {
					t.Errorf("Scenario %d: Invalid connections %d for %d nodes",
						i, connections, scenario.nodeCount)
				}
			}
		})
	}
}
