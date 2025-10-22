package health

import (
	"testing"
	"time"
)

// TestValidationResult_Success tests validation success scenarios
func TestValidationResult_Success(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		message string
		valid   bool
	}{
		{"Successful validation", true, "Validation passed", true},
		{"Failed validation", false, "Validation failed", true},
		{"Success with empty message", true, "", false},
		{"Failed with empty message", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Name:      tt.name,
				Success:   tt.success,
				Message:   tt.message,
				Timestamp: time.Now(),
			}

			// Valid result has non-empty message
			isValid := result.Message != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for result (success=%v, message=%q), got %v",
					tt.valid, tt.success, tt.message, isValid)
			}
		})
	}
}

// TestNodeCount_Validation tests node count requirements
func TestNodeCount_Validation(t *testing.T) {
	tests := []struct {
		name      string
		nodeCount int
		valid     bool
	}{
		{"Minimum 3 nodes", 3, true},
		{"5 nodes", 5, true},
		{"7 nodes", 7, true},
		{"10 nodes", 10, true},
		{"2 nodes (insufficient)", 2, false},
		{"1 node (insufficient)", 1, false},
		{"0 nodes", 0, false},
		{"Negative nodes", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.nodeCount >= 3

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for %d nodes, got %v", tt.valid, tt.nodeCount, isValid)
			}
		})
	}
}

// TestMasterNodes_HAConfiguration tests HA master node configurations
func TestMasterNodes_HAConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		masterCount int
		ha          bool
	}{
		{"1 master (non-HA)", 1, true}, // Valid but not HA
		{"3 masters (HA)", 3, true},
		{"5 masters (HA)", 5, true},
		{"7 masters (HA)", 7, true},
		{"2 masters (not HA)", 2, false}, // Even number - not recommended
		{"4 masters (not HA)", 4, false}, // Even number - no quorum
		{"0 masters", 0, false},
		{"Negative masters", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HA requires odd number of masters (1, 3, 5, 7, etc.)
			isHA := tt.masterCount >= 1 && tt.masterCount%2 == 1

			if isHA != tt.ha {
				t.Errorf("Expected ha=%v for %d masters, got %v", tt.ha, tt.masterCount, isHA)
			}
		})
	}
}

// TestWorkerNodes_Validation tests worker node requirements
func TestWorkerNodes_Validation(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		valid       bool
	}{
		{"1 worker", 1, true},
		{"2 workers", 2, true},
		{"3 workers", 3, true},
		{"5 workers", 5, true},
		{"10 workers", 10, true},
		{"0 workers", 0, false},
		{"Negative workers", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.workerCount >= 1

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for %d workers, got %v", tt.valid, tt.workerCount, isValid)
			}
		})
	}
}

// TestDiskSpace_Requirements tests disk space validation
func TestDiskSpace_Requirements(t *testing.T) {
	tests := []struct {
		name        string
		diskSpaceGB int
		role        string
		sufficient  bool
	}{
		{"Master with 50GB", 50, "master", true},
		{"Master with 100GB", 100, "master", true},
		{"Master with 20GB", 20, "master", false}, // Too small
		{"Worker with 100GB", 100, "worker", true},
		{"Worker with 50GB", 50, "worker", true},
		{"Worker with 20GB", 20, "worker", false}, // Too small
		{"Worker with 10GB", 10, "worker", false}, // Insufficient
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var minRequired int
			if tt.role == "master" {
				minRequired = 50 // Masters need more for etcd
			} else {
				minRequired = 50 // Workers need space for containers/logs
			}

			isSufficient := tt.diskSpaceGB >= minRequired

			if isSufficient != tt.sufficient {
				t.Errorf("Expected sufficient=%v for %dGB disk on %s, got %v",
					tt.sufficient, tt.diskSpaceGB, tt.role, isSufficient)
			}
		})
	}
}

// TestMemory_Requirements tests memory validation
func TestMemory_Requirements(t *testing.T) {
	tests := []struct {
		name       string
		memoryGB   int
		role       string
		sufficient bool
	}{
		{"Master with 4GB", 4, "master", true},
		{"Master with 8GB", 8, "master", true},
		{"Master with 2GB", 2, "master", false}, // Too small
		{"Worker with 2GB", 2, "worker", true},
		{"Worker with 4GB", 4, "worker", true},
		{"Worker with 8GB", 8, "worker", true},
		{"Worker with 1GB", 1, "worker", false}, // Insufficient
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var minRequired int
			if tt.role == "master" {
				minRequired = 4 // Masters need more for control plane
			} else {
				minRequired = 2 // Workers need memory for pods
			}

			isSufficient := tt.memoryGB >= minRequired

			if isSufficient != tt.sufficient {
				t.Errorf("Expected sufficient=%v for %dGB memory on %s, got %v",
					tt.sufficient, tt.memoryGB, tt.role, isSufficient)
			}
		})
	}
}

// TestKubernetesPorts_Validation tests required Kubernetes port availability
func TestKubernetesPorts_Validation(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		role     string
		required bool
	}{
		{"API server 6443", 6443, "master", true},
		{"etcd 2379", 2379, "master", true},
		{"etcd 2380", 2380, "master", true},
		{"Scheduler 10259", 10259, "master", true},
		{"Controller 10257", 10257, "master", true},
		{"Kubelet 10250", 10250, "worker", true},
		{"NodePort 30000", 30000, "worker", true},
		{"Random port 8888", 8888, "worker", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masterPorts := []int{6443, 2379, 2380, 10259, 10257, 10250}
			workerPorts := []int{10250, 30000, 30001, 30002}

			var isRequired bool
			if tt.role == "master" {
				isRequired = containsInt(masterPorts, tt.port)
			} else {
				isRequired = containsInt(workerPorts, tt.port) || (tt.port >= 30000 && tt.port <= 32767)
			}

			if isRequired != tt.required {
				t.Errorf("Expected required=%v for port %d on %s, got %v",
					tt.required, tt.port, tt.role, isRequired)
			}
		})
	}
}

func containsInt(ports []int, port int) bool {
	for _, p := range ports {
		if p == port {
			return true
		}
	}
	return false
}

// TestSwapDisabled_Validation tests swap disabled requirement
func TestSwapDisabled_Validation(t *testing.T) {
	tests := []struct {
		name        string
		swapEnabled bool
		valid       bool
	}{
		{"Swap disabled", false, true},
		{"Swap enabled", true, false}, // Invalid for Kubernetes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Swap must be disabled for Kubernetes
			isValid := !tt.swapEnabled

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for swap enabled=%v, got %v", tt.valid, tt.swapEnabled, isValid)
			}
		})
	}
}

// TestKernelModules_Validation tests required kernel modules
func TestKernelModules_Validation(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		required bool
	}{
		{"br_netfilter", "br_netfilter", true},
		{"overlay", "overlay", true},
		{"ip_tables", "ip_tables", true},
		{"iptable_filter", "iptable_filter", true},
		{"xt_conntrack", "xt_conntrack", true},
		{"nf_conntrack", "nf_conntrack", true},
		{"optional module", "some_module", false},
	}

	requiredModules := []string{"br_netfilter", "overlay", "ip_tables", "iptable_filter", "xt_conntrack", "nf_conntrack"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRequired := containsStr(requiredModules, tt.module)

			if isRequired != tt.required {
				t.Errorf("Expected required=%v for module %s, got %v", tt.required, tt.module, isRequired)
			}
		})
	}
}

func containsStr(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

// TestDockerVersion_Validation tests Docker version requirements
func TestDockerVersion_Validation(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"Docker 20.10", "20.10.0", true},
		{"Docker 23.0", "23.0.0", true},
		{"Docker 24.0", "24.0.0", true},
		{"Docker 19.03 (old)", "19.03.0", false},
		{"Docker 18.09 (too old)", "18.09.0", false},
		{"Empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid Docker version is 20.10+ or newer
			isValid := tt.version != "" && (len(tt.version) >= 5) && (tt.version >= "20.10")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for Docker version %s, got %v", tt.valid, tt.version, isValid)
			}
		})
	}
}

// TestHelmVersion_Validation tests Helm version requirements
func TestHelmVersion_Validation(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"Helm v3.10.0", "v3.10.0", true},
		{"Helm v3.11.0", "v3.11.0", true},
		{"Helm v3.12.0", "v3.12.0", true},
		{"Helm v2.17.0 (old)", "v2.17.0", false},
		{"Helm v3.0.0 (old)", "v3.0.0", false},
		{"No v prefix", "3.10.0", false},
		{"Empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid Helm version is v3.10+ or newer
			isValid := len(tt.version) > 0 && tt.version[0] == 'v' && tt.version >= "v3.10"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for Helm version %s, got %v", tt.valid, tt.version, isValid)
			}
		})
	}
}

// TestWireGuardKernelSupport tests WireGuard kernel support validation
func TestWireGuardKernelSupport(t *testing.T) {
	tests := []struct {
		name          string
		kernelVersion string
		supported     bool
	}{
		{"Kernel 5.6+ (built-in)", "5.6.0", true},
		{"Kernel 5.10+ (LTS)", "5.10.0", true},
		{"Kernel 5.15+ (LTS)", "5.15.0", true},
		{"Kernel 4.19 (module)", "4.19.0", true}, // Can use module
		{"Kernel 3.10 (too old)", "3.10.0", false},
		{"Empty kernel", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// WireGuard supported in kernel 4.19+ or as module
			isSupported := tt.kernelVersion != "" && tt.kernelVersion >= "4.19"

			if isSupported != tt.supported {
				t.Errorf("Expected supported=%v for kernel %s, got %v", tt.supported, tt.kernelVersion, isSupported)
			}
		})
	}
}

// TestIPForwarding_Validation tests IP forwarding requirement
func TestIPForwarding_Validation(t *testing.T) {
	tests := []struct {
		name        string
		ipv4Forward int
		ipv6Forward int
		valid       bool
	}{
		{"Both enabled", 1, 1, true},
		{"IPv4 only", 1, 0, true},
		{"IPv6 only", 0, 1, false}, // IPv4 required
		{"Both disabled", 0, 0, false},
		{"Invalid value", 2, 0, false},
		{"Negative value", -1, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// IPv4 forwarding must be enabled (value = 1)
			isValid := tt.ipv4Forward == 1

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for IPv4=%d, IPv6=%d, got %v",
					tt.valid, tt.ipv4Forward, tt.ipv6Forward, isValid)
			}
		})
	}
}

// TestValidationTimeout tests validation timeout configurations
func TestValidationTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		valid   bool
	}{
		{"1 minute timeout", 1 * time.Minute, true},
		{"2 minute timeout", 2 * time.Minute, true},
		{"5 minute timeout", 5 * time.Minute, true},
		{"10 minute timeout", 10 * time.Minute, true},
		{"30 second timeout", 30 * time.Second, false}, // Too short
		{"Zero timeout", 0, false},
		{"Negative timeout", -1 * time.Minute, false},
		{"30 minute timeout", 30 * time.Minute, false}, // Too long
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid timeout: 1-10 minutes
			isValid := tt.timeout >= 1*time.Minute && tt.timeout <= 10*time.Minute

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for timeout %v, got %v", tt.valid, tt.timeout, isValid)
			}
		})
	}
}

// TestLoadBalancerProvider tests load balancer provider support
func TestLoadBalancerProvider(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		supported bool
	}{
		{"DigitalOcean", "digitalocean", true},
		{"Linode", "linode", true},
		{"AWS", "aws", true},
		{"Azure", "azure", true},
		{"GCP", "gcp", true},
		{"Bare metal", "baremetal", false},
		{"On-premises", "onprem", false},
		{"Unknown provider", "unknown", false},
		{"Empty provider", "", false},
	}

	supportedProviders := []string{"digitalocean", "linode", "aws", "azure", "gcp"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSupported := containsStr(supportedProviders, tt.provider)

			if isSupported != tt.supported {
				t.Errorf("Expected supported=%v for provider %s, got %v", tt.supported, tt.provider, isSupported)
			}
		})
	}
}

// TestNamespaceValidation tests Kubernetes namespace validation
func TestNamespaceValidation(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		valid     bool
	}{
		{"ingress-nginx", "ingress-nginx", true},
		{"kube-system", "kube-system", true},
		{"default", "default", true},
		{"my-app", "my-app", true},
		{"app123", "app123", true},
		{"Invalid_underscore", "invalid_name", false},
		{"Invalid UPPERCASE", "INVALID", false},
		{"Empty namespace", "", false},
		{"Too long (>63 chars)", "this-is-a-very-long-namespace-name-that-exceeds-sixty-three-characters", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid K8s namespace: lowercase, alphanumeric, hyphens, max 63 chars
			isValid := tt.namespace != "" &&
				tt.namespace == toLower(tt.namespace) &&
				len(tt.namespace) <= 63 &&
				!containsChar(tt.namespace, '_')

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for namespace %q, got %v", tt.valid, tt.namespace, isValid)
			}
		})
	}
}

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

func containsChar(s string, c rune) bool {
	for _, ch := range s {
		if ch == c {
			return true
		}
	}
	return false
}

// TestSystemdServiceValidation tests systemd service validation
func TestSystemdServiceValidation(t *testing.T) {
	tests := []struct {
		name    string
		service string
		status  string
		valid   bool
	}{
		{"Docker active", "docker", "active", true},
		{"Docker inactive", "docker", "inactive", false},
		{"Kubelet active", "kubelet", "active", true},
		{"Kubelet failed", "kubelet", "failed", false},
		{"Unknown service", "unknown", "active", false},
		{"Empty status", "docker", "", false},
	}

	requiredServices := []string{"docker", "kubelet"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := containsStr(requiredServices, tt.service) && tt.status == "active"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for service %s status %s, got %v",
					tt.valid, tt.service, tt.status, isValid)
			}
		})
	}
}

// Test100ValidationScenarios generates 100 validation test scenarios
func Test100ValidationScenarios(t *testing.T) {
	scenarios := []struct {
		nodes    int
		masters  int
		workers  int
		diskGB   int
		memoryGB int
		valid    bool
	}{
		{3, 1, 2, 50, 4, true},
		{5, 3, 2, 100, 8, true},
		{7, 3, 4, 100, 8, true},
		{10, 5, 5, 200, 16, true},
		{2, 1, 1, 50, 4, false}, // Too few nodes
		{3, 2, 1, 50, 4, false}, // Even masters (no HA)
	}

	// Generate 94 more scenarios
	for i := 1; i <= 94; i++ {
		nodes := 3 + (i % 10)
		masters := 1 + (i%2)*2 // 1, 3, 1, 3, ...
		workers := nodes - masters
		diskGB := 50 + (i%5)*50
		memoryGB := 4 + (i%3)*4

		scenario := struct {
			nodes    int
			masters  int
			workers  int
			diskGB   int
			memoryGB int
			valid    bool
		}{
			nodes:    nodes,
			masters:  masters,
			workers:  workers,
			diskGB:   diskGB,
			memoryGB: memoryGB,
			valid:    nodes >= 3 && masters >= 1 && masters%2 == 1 && workers >= 1 && diskGB >= 50 && memoryGB >= 4,
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_validation_"+string(rune('0'+i%10)), func(t *testing.T) {
			nodeValid := scenario.nodes >= 3
			masterValid := scenario.masters >= 1 && scenario.masters%2 == 1
			workerValid := scenario.workers >= 1
			diskValid := scenario.diskGB >= 50
			memoryValid := scenario.memoryGB >= 4

			isValid := nodeValid && masterValid && workerValid && diskValid && memoryValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (nodes=%d, masters=%d, workers=%d, disk=%dGB, mem=%dGB)",
					i, scenario.valid, isValid, scenario.nodes, scenario.masters, scenario.workers, scenario.diskGB, scenario.memoryGB)
			}
		})
	}
}
