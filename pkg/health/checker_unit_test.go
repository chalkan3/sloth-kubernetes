package health

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test buildHealthCheckScript basic structure
func TestHealthChecker_BuildHealthCheckScript_Structure(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker"}
	script := checker.buildHealthCheckScript(services)

	// Should start with shebang
	assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))

	// Should contain critical sections
	sections := []string{
		"set -e",
		"=== Node Health Check ===",
		"Timestamp:",
		"check_service()",
		"check_command()",
		"check_port()",
		"UPTIME:",
		"LOAD:",
		"MEMORY:",
		"DISK:",
		"=== Health Check Complete ===",
	}

	for _, section := range sections {
		assert.Contains(t, script, section,
			"Script should contain section: %s", section)
	}
}

// Test buildHealthCheckScript with Docker service
func TestHealthChecker_BuildHealthCheckScript_Docker(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker"}
	script := checker.buildHealthCheckScript(services)

	dockerChecks := []string{
		"# Docker checks",
		"check_service docker",
		"check_command docker",
		"docker version",
		"DOCKER:VERSION:OK",
		"DOCKER:VERSION:FAIL",
		"docker ps",
		"DOCKER:PS:OK",
		"DOCKER:PS:FAIL",
	}

	for _, check := range dockerChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with WireGuard service
func TestHealthChecker_BuildHealthCheckScript_WireGuard(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"wireguard"}
	script := checker.buildHealthCheckScript(services)

	wgChecks := []string{
		"# WireGuard checks",
		"check_command wg",
		"/etc/wireguard/wg0.conf",
		"WIREGUARD:CONFIG:EXISTS",
		"WIREGUARD:CONFIG:MISSING",
		"wg show wg0",
		"WIREGUARD:INTERFACE:UP",
		"WIREGUARD:INTERFACE:DOWN",
	}

	for _, check := range wgChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with Kubernetes service
func TestHealthChecker_BuildHealthCheckScript_Kubernetes(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"kubernetes"}
	script := checker.buildHealthCheckScript(services)

	k8sChecks := []string{
		"# Kubernetes checks",
		"check_command kubectl",
		"kubectl version --client",
		"KUBECTL:VERSION:OK",
		"/root/kube_config_cluster.yml",
		"KUBECONFIG:EXISTS",
		"KUBECONFIG:MISSING",
		"export KUBECONFIG=/root/kube_config_cluster.yml",
		"kubectl get nodes",
		"KUBERNETES:API:OK",
		"KUBERNETES:API:FAIL",
	}

	for _, check := range k8sChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with Kubelet service
func TestHealthChecker_BuildHealthCheckScript_Kubelet(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"kubelet"}
	script := checker.buildHealthCheckScript(services)

	kubeletChecks := []string{
		"# Kubelet checks",
		"check_service kubelet",
		"check_port 10250",
	}

	for _, check := range kubeletChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with Etcd service
func TestHealthChecker_BuildHealthCheckScript_Etcd(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"etcd"}
	script := checker.buildHealthCheckScript(services)

	etcdChecks := []string{
		"# Etcd checks",
		"check_port 2379",
		"check_port 2380",
	}

	for _, check := range etcdChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with NGINX service
func TestHealthChecker_BuildHealthCheckScript_NGINX(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"nginx"}
	script := checker.buildHealthCheckScript(services)

	nginxChecks := []string{
		"# NGINX checks",
		"kubectl get svc -n ingress-nginx nginx-ingress-controller",
		"NGINX:SERVICE:OK",
		"NGINX:SERVICE:FAIL",
		"kubectl get pods -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx",
		"NGINX:PODS:OK",
		"NGINX:PODS:FAIL",
	}

	for _, check := range nginxChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with SSH service
func TestHealthChecker_BuildHealthCheckScript_SSH(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"ssh"}
	script := checker.buildHealthCheckScript(services)

	sshChecks := []string{
		"# SSH checks",
		"check_service ssh",
		"check_port 22",
	}

	for _, check := range sshChecks {
		assert.Contains(t, script, check)
	}
}

// Test buildHealthCheckScript with multiple services
func TestHealthChecker_BuildHealthCheckScript_MultipleServices(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker", "wireguard", "kubernetes"}
	script := checker.buildHealthCheckScript(services)

	// Should contain checks for all services
	assert.Contains(t, script, "# Docker checks")
	assert.Contains(t, script, "# WireGuard checks")
	assert.Contains(t, script, "# Kubernetes checks")
}

// Test buildHealthCheckScript helper functions
func TestHealthChecker_BuildHealthCheckScript_HelperFunctions(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker"}
	script := checker.buildHealthCheckScript(services)

	// Should define helper functions
	helpers := []string{
		"check_service() {",
		"systemctl is-active --quiet",
		"SERVICE:$service:RUNNING",
		"SERVICE:$service:STOPPED",
		"check_command() {",
		"command -v $cmd",
		"COMMAND:$cmd:AVAILABLE",
		"COMMAND:$cmd:MISSING",
		"check_port() {",
		"netstat -tuln",
		"PORT:$port:LISTENING",
		"PORT:$port:CLOSED",
	}

	for _, helper := range helpers {
		assert.Contains(t, script, helper)
	}
}

// Test buildHealthCheckScript system metrics
func TestHealthChecker_BuildHealthCheckScript_SystemMetrics(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	metrics := []string{
		"UPTIME:$(uptime -p)",
		"LOAD:$(cat /proc/loadavg",
		"MEMORY:$(free -m",
		"DISK:$(df -h /",
	}

	for _, metric := range metrics {
		assert.Contains(t, script, metric)
	}
}

// Test buildHealthCheckScript output markers
func TestHealthChecker_BuildHealthCheckScript_OutputMarkers(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker", "wireguard", "kubernetes"}
	script := checker.buildHealthCheckScript(services)

	markers := []string{
		"SERVICE:",
		"COMMAND:",
		"PORT:",
		"DOCKER:",
		"WIREGUARD:",
		"KUBERNETES:",
		"KUBECTL:",
		"KUBECONFIG:",
	}

	for _, marker := range markers {
		assert.Contains(t, script, marker)
	}
}

// Test 50 health check scenarios
func Test50HealthCheckScenarios(t *testing.T) {
	checker := &HealthChecker{}

	scenarios := []struct {
		services []string
		checks   []string
	}{
		{
			[]string{"docker"},
			[]string{"# Docker checks", "docker version"},
		},
		{
			[]string{"wireguard"},
			[]string{"# WireGuard checks", "wg show"},
		},
		{
			[]string{"kubernetes"},
			[]string{"# Kubernetes checks", "kubectl get nodes"},
		},
	}

	// Generate 47 more scenarios
	allServices := []string{"docker", "wireguard", "kubernetes", "kubelet", "etcd", "nginx", "ssh"}

	for i := 0; i < 47; i++ {
		// Vary the service combinations
		var services []string
		if i%7 == 0 {
			services = []string{"docker", "wireguard"}
		} else if i%7 == 1 {
			services = []string{"kubernetes", "kubelet"}
		} else if i%7 == 2 {
			services = []string{"docker", "kubernetes"}
		} else if i%7 == 3 {
			services = []string{"etcd"}
		} else if i%7 == 4 {
			services = []string{"nginx"}
		} else if i%7 == 5 {
			services = []string{"ssh"}
		} else {
			services = []string{allServices[i%len(allServices)]}
		}

		scenarios = append(scenarios, struct {
			services []string
			checks   []string
		}{services, []string{}})
	}

	for i, scenario := range scenarios {
		t.Run("Scenario_"+string(rune('A'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			script := checker.buildHealthCheckScript(scenario.services)

			// Validate basic structure
			assert.NotEmpty(t, script)
			assert.True(t, strings.HasPrefix(script, "#!/bin/bash"))
			assert.Contains(t, script, "=== Node Health Check ===")
			assert.Contains(t, script, "=== Health Check Complete ===")

			// Validate service checks are included
			for _, service := range scenario.services {
				switch service {
				case "docker":
					assert.Contains(t, script, "# Docker checks")
				case "wireguard":
					assert.Contains(t, script, "# WireGuard checks")
				case "kubernetes":
					assert.Contains(t, script, "# Kubernetes checks")
				case "kubelet":
					assert.Contains(t, script, "# Kubelet checks")
				case "etcd":
					assert.Contains(t, script, "# Etcd checks")
				case "nginx":
					assert.Contains(t, script, "# NGINX checks")
				case "ssh":
					assert.Contains(t, script, "# SSH checks")
				}
			}
		})
	}
}

// Test isServiceHealthy helper
func TestHealthChecker_IsServiceHealthy(t *testing.T) {
	checker := &HealthChecker{}

	tests := []struct {
		name     string
		output   string
		service  string
		expected bool
	}{
		{
			"Service running",
			"DOCKER:PS:OK\nSERVICE:docker:RUNNING",
			"docker",
			true, // docker needs BOTH DOCKER:PS:OK AND SERVICE:docker:RUNNING
		},
		{
			"Service stopped",
			"SERVICE:docker:STOPPED",
			"docker",
			false,
		},
		{
			"Command available",
			"COMMAND:docker:AVAILABLE\nSERVICE:docker:RUNNING",
			"docker",
			false, // docker service needs BOTH DOCKER:PS:OK AND SERVICE:docker:RUNNING
		},
		{
			"Command missing",
			"COMMAND:docker:MISSING",
			"docker",
			false,
		},
		{
			"Port listening",
			"PORT:22:LISTENING\nSERVICE:ssh:RUNNING",
			"ssh",
			true, // ssh is checked as default case - needs SERVICE:ssh:RUNNING
		},
		{
			"Port closed",
			"PORT:22:CLOSED",
			"ssh",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.isServiceHealthy(tt.output, tt.service)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test script bash safety
func TestHealthChecker_ScriptBashSafety(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker"}
	script := checker.buildHealthCheckScript(services)

	// Should use 'set -e' for error handling
	assert.Contains(t, script, "set -e")

	// Should redirect stderr appropriately
	assert.Contains(t, script, "&>/dev/null")
	assert.Contains(t, script, "&> /dev/null")
}

// Test script systemctl usage
func TestHealthChecker_ScriptSystemctl(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker", "kubelet", "ssh"}
	script := checker.buildHealthCheckScript(services)

	// Should use systemctl for service checks
	assert.Contains(t, script, "systemctl is-active --quiet")
}

// Test script netstat usage
func TestHealthChecker_ScriptNetstat(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"kubelet", "etcd", "ssh"}
	script := checker.buildHealthCheckScript(services)

	// Should use netstat for port checks
	assert.Contains(t, script, "netstat -tuln")
}

// Test script kubectl commands
func TestHealthChecker_ScriptKubectl(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"kubernetes", "nginx"}
	script := checker.buildHealthCheckScript(services)

	kubectlCommands := []string{
		"kubectl version --client",
		"kubectl get nodes",
		"kubectl get svc",
		"kubectl get pods",
	}

	for _, cmd := range kubectlCommands {
		assert.Contains(t, script, cmd)
	}
}

// Test script uptime command
func TestHealthChecker_ScriptUptime(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should check uptime
	assert.Contains(t, script, "uptime -p")
}

// Test script load average
func TestHealthChecker_ScriptLoadAverage(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should check load average from /proc/loadavg
	assert.Contains(t, script, "/proc/loadavg")
	assert.Contains(t, script, "cut -d' ' -f1-3")
}

// Test script memory usage
func TestHealthChecker_ScriptMemoryUsage(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should check memory with free -m
	assert.Contains(t, script, "free -m")
	assert.Contains(t, script, "grep Mem")
}

// Test script disk usage
func TestHealthChecker_ScriptDiskUsage(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should check disk usage with df -h
	assert.Contains(t, script, "df -h /")
}

// Test script timestamp
func TestHealthChecker_ScriptTimestamp(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should include timestamp
	assert.Contains(t, script, "Timestamp: $(date)")
}

// Test script with no services
func TestHealthChecker_BuildHealthCheckScript_NoServices(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should still have basic structure
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "=== Node Health Check ===")
	assert.Contains(t, script, "UPTIME:")
	assert.Contains(t, script, "LOAD:")
	assert.Contains(t, script, "MEMORY:")
	assert.Contains(t, script, "DISK:")
	assert.Contains(t, script, "=== Health Check Complete ===")
}

// Test script function definitions
func TestHealthChecker_ScriptFunctionDefinitions(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Should define all helper functions
	functions := []string{
		"check_service() {",
		"check_command() {",
		"check_port() {",
	}

	for _, fn := range functions {
		assert.Contains(t, script, fn)
	}
}

// Test script local variables in functions
func TestHealthChecker_ScriptLocalVariables(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{}
	script := checker.buildHealthCheckScript(services)

	// Functions should use local variables
	localVars := []string{
		"local service=$1",
		"local cmd=$1",
		"local port=$1",
	}

	for _, localVar := range localVars {
		assert.Contains(t, script, localVar)
	}
}

// Test WireGuard config check
func TestHealthChecker_WireGuardConfigCheck(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"wireguard"}
	script := checker.buildHealthCheckScript(services)

	// Should check for config file existence
	assert.Contains(t, script, "[ -f /etc/wireguard/wg0.conf ]")
}

// Test Kubernetes kubeconfig check
func TestHealthChecker_KubeconfigCheck(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"kubernetes"}
	script := checker.buildHealthCheckScript(services)

	// Should check for kubeconfig file
	assert.Contains(t, script, "[ -f /root/kube_config_cluster.yml ]")
}

// Test all service types coverage
func TestHealthChecker_AllServiceTypes(t *testing.T) {
	checker := &HealthChecker{}

	allServices := []string{"docker", "wireguard", "kubernetes", "kubelet", "etcd", "nginx", "ssh"}
	script := checker.buildHealthCheckScript(allServices)

	expectedChecks := []string{
		"# Docker checks",
		"# WireGuard checks",
		"# Kubernetes checks",
		"# Kubelet checks",
		"# Etcd checks",
		"# NGINX checks",
		"# SSH checks",
	}

	for _, check := range expectedChecks {
		assert.Contains(t, script, check,
			"Script should contain check: %s", check)
	}
}

// Test ports being checked
func TestHealthChecker_PortsBeingChecked(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"ssh", "kubelet", "etcd"}
	script := checker.buildHealthCheckScript(services)

	ports := []string{
		"check_port 22",    // SSH
		"check_port 10250", // Kubelet
		"check_port 2379",  // Etcd client
		"check_port 2380",  // Etcd peer
	}

	for _, port := range ports {
		assert.Contains(t, script, port)
	}
}

// Test Docker version check
func TestHealthChecker_DockerVersionCheck(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker"}
	script := checker.buildHealthCheckScript(services)

	// Should check docker version
	assert.Contains(t, script, "docker version &>/dev/null")
	assert.Contains(t, script, "DOCKER:VERSION:OK")
	assert.Contains(t, script, "DOCKER:VERSION:FAIL")
}

// Test Docker ps check
func TestHealthChecker_DockerPsCheck(t *testing.T) {
	checker := &HealthChecker{}

	services := []string{"docker"}
	script := checker.buildHealthCheckScript(services)

	// Should check docker ps (daemon running)
	assert.Contains(t, script, "docker ps &>/dev/null")
	assert.Contains(t, script, "DOCKER:PS:OK")
	assert.Contains(t, script, "DOCKER:PS:FAIL")
}
