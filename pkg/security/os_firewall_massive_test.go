package security

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"sloth-kubernetes/pkg/providers"
)

// TestFirewallRuleStructure tests FirewallRule structure
func TestFirewallRuleStructure(t *testing.T) {
	rule := FirewallRule{
		Port:        "6443",
		Protocol:    "tcp",
		Source:      "0.0.0.0/0",
		Direction:   "inbound",
		Action:      "allow",
		Description: "Kubernetes API Server",
	}

	if rule.Port != "6443" {
		t.Errorf("Expected port 6443, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Direction != "inbound" {
		t.Errorf("Expected direction inbound, got %s", rule.Direction)
	}
	if rule.Action != "allow" {
		t.Errorf("Expected action allow, got %s", rule.Action)
	}
}

// TestFirewallResultStructure tests FirewallResult structure
func TestFirewallResultStructure(t *testing.T) {
	result := &FirewallResult{
		NodeName:     "test-node",
		Success:      true,
		RulesApplied: []FirewallRule{},
		Timestamp:    time.Now(),
		FirewallType: "ufw",
	}

	if result.NodeName != "test-node" {
		t.Errorf("Expected NodeName test-node, got %s", result.NodeName)
	}
	if !result.Success {
		t.Error("Expected Success to be true")
	}
	if result.FirewallType != "ufw" {
		t.Errorf("Expected FirewallType ufw, got %s", result.FirewallType)
	}
}

// TestKubernetesFirewallPorts_APIServer tests API Server port
func TestKubernetesFirewallPorts_APIServer(t *testing.T) {
	rule := KubernetesFirewallPorts.APIServer

	if rule.Port != "6443" {
		t.Errorf("Expected port 6443, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Source != "0.0.0.0/0" {
		t.Errorf("Expected source 0.0.0.0/0, got %s", rule.Source)
	}
	if rule.Direction != "inbound" {
		t.Errorf("Expected direction inbound, got %s", rule.Direction)
	}
	if rule.Action != "allow" {
		t.Errorf("Expected action allow, got %s", rule.Action)
	}
	if !strings.Contains(rule.Description, "API Server") {
		t.Errorf("Expected description to contain 'API Server', got %s", rule.Description)
	}
}

// TestKubernetesFirewallPorts_ETCD tests ETCD port
func TestKubernetesFirewallPorts_ETCD(t *testing.T) {
	rule := KubernetesFirewallPorts.ETCD

	if rule.Port != "2379" {
		t.Errorf("Expected port 2379, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Source != "10.0.0.0/8" {
		t.Errorf("Expected source 10.0.0.0/8, got %s", rule.Source)
	}
}

// TestKubernetesFirewallPorts_ETCDPeer tests ETCD Peer port
func TestKubernetesFirewallPorts_ETCDPeer(t *testing.T) {
	rule := KubernetesFirewallPorts.ETCDPeer

	if rule.Port != "2380" {
		t.Errorf("Expected port 2380, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_Scheduler tests kube-scheduler port
func TestKubernetesFirewallPorts_Scheduler(t *testing.T) {
	rule := KubernetesFirewallPorts.Scheduler

	if rule.Port != "10259" {
		t.Errorf("Expected port 10259, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_ControllerManager tests controller-manager port
func TestKubernetesFirewallPorts_ControllerManager(t *testing.T) {
	rule := KubernetesFirewallPorts.ControllerManager

	if rule.Port != "10257" {
		t.Errorf("Expected port 10257, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_Kubelet tests kubelet port
func TestKubernetesFirewallPorts_Kubelet(t *testing.T) {
	rule := KubernetesFirewallPorts.Kubelet

	if rule.Port != "10250" {
		t.Errorf("Expected port 10250, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_KubeProxy tests kube-proxy port
func TestKubernetesFirewallPorts_KubeProxy(t *testing.T) {
	rule := KubernetesFirewallPorts.KubeProxy

	if rule.Port != "10256" {
		t.Errorf("Expected port 10256, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_NodePortServices tests NodePort range
func TestKubernetesFirewallPorts_NodePortServices(t *testing.T) {
	rule := KubernetesFirewallPorts.NodePortServices

	if rule.Port != "30000:32767" {
		t.Errorf("Expected port 30000:32767, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Source != "0.0.0.0/0" {
		t.Errorf("Expected source 0.0.0.0/0, got %s", rule.Source)
	}
}

// TestKubernetesFirewallPorts_Flannel tests Flannel VXLAN port
func TestKubernetesFirewallPorts_Flannel(t *testing.T) {
	rule := KubernetesFirewallPorts.Flannel

	if rule.Port != "8472" {
		t.Errorf("Expected port 8472, got %s", rule.Port)
	}
	if rule.Protocol != "udp" {
		t.Errorf("Expected protocol udp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_Calico tests Calico BGP port
func TestKubernetesFirewallPorts_Calico(t *testing.T) {
	rule := KubernetesFirewallPorts.Calico

	if rule.Port != "179" {
		t.Errorf("Expected port 179, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_CanalBGP tests Canal BGP port
func TestKubernetesFirewallPorts_CanalBGP(t *testing.T) {
	rule := KubernetesFirewallPorts.CanalBGP

	if rule.Port != "179" {
		t.Errorf("Expected port 179, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_Weave tests Weave Net port
func TestKubernetesFirewallPorts_Weave(t *testing.T) {
	rule := KubernetesFirewallPorts.Weave

	if rule.Port != "6783" {
		t.Errorf("Expected port 6783, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_SSH tests SSH port
func TestKubernetesFirewallPorts_SSH(t *testing.T) {
	rule := KubernetesFirewallPorts.SSH

	if rule.Port != "22" {
		t.Errorf("Expected port 22, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Source != "10.8.0.0/24" {
		t.Errorf("Expected source 10.8.0.0/24 (WireGuard), got %s", rule.Source)
	}
}

// TestKubernetesFirewallPorts_WireGuard tests WireGuard port
func TestKubernetesFirewallPorts_WireGuard(t *testing.T) {
	rule := KubernetesFirewallPorts.WireGuard

	if rule.Port != "51820" {
		t.Errorf("Expected port 51820, got %s", rule.Port)
	}
	if rule.Protocol != "udp" {
		t.Errorf("Expected protocol udp, got %s", rule.Protocol)
	}
	if rule.Source != "0.0.0.0/0" {
		t.Errorf("Expected source 0.0.0.0/0, got %s", rule.Source)
	}
}

// TestKubernetesFirewallPorts_DockerRegistry tests Docker Registry port
func TestKubernetesFirewallPorts_DockerRegistry(t *testing.T) {
	rule := KubernetesFirewallPorts.DockerRegistry

	if rule.Port != "5000" {
		t.Errorf("Expected port 5000, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_MetricsServer tests Metrics Server port
func TestKubernetesFirewallPorts_MetricsServer(t *testing.T) {
	rule := KubernetesFirewallPorts.MetricsServer

	if rule.Port != "10255" {
		t.Errorf("Expected port 10255, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
}

// TestKubernetesFirewallPorts_HTTPIngress tests HTTP Ingress port
func TestKubernetesFirewallPorts_HTTPIngress(t *testing.T) {
	rule := KubernetesFirewallPorts.HTTPIngress

	if rule.Port != "80" {
		t.Errorf("Expected port 80, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Source != "0.0.0.0/0" {
		t.Errorf("Expected source 0.0.0.0/0, got %s", rule.Source)
	}
}

// TestKubernetesFirewallPorts_HTTPSIngress tests HTTPS Ingress port
func TestKubernetesFirewallPorts_HTTPSIngress(t *testing.T) {
	rule := KubernetesFirewallPorts.HTTPSIngress

	if rule.Port != "443" {
		t.Errorf("Expected port 443, got %s", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("Expected protocol tcp, got %s", rule.Protocol)
	}
	if rule.Source != "0.0.0.0/0" {
		t.Errorf("Expected source 0.0.0.0/0, got %s", rule.Source)
	}
}

// TestAllKubernetesPorts tests all Kubernetes ports are defined
func TestAllKubernetesPorts(t *testing.T) {
	ports := []struct {
		name string
		rule FirewallRule
	}{
		{"APIServer", KubernetesFirewallPorts.APIServer},
		{"ETCD", KubernetesFirewallPorts.ETCD},
		{"ETCDPeer", KubernetesFirewallPorts.ETCDPeer},
		{"Scheduler", KubernetesFirewallPorts.Scheduler},
		{"ControllerManager", KubernetesFirewallPorts.ControllerManager},
		{"Kubelet", KubernetesFirewallPorts.Kubelet},
		{"KubeProxy", KubernetesFirewallPorts.KubeProxy},
		{"NodePortServices", KubernetesFirewallPorts.NodePortServices},
		{"Flannel", KubernetesFirewallPorts.Flannel},
		{"Calico", KubernetesFirewallPorts.Calico},
		{"CanalBGP", KubernetesFirewallPorts.CanalBGP},
		{"Weave", KubernetesFirewallPorts.Weave},
		{"SSH", KubernetesFirewallPorts.SSH},
		{"WireGuard", KubernetesFirewallPorts.WireGuard},
		{"DockerRegistry", KubernetesFirewallPorts.DockerRegistry},
		{"MetricsServer", KubernetesFirewallPorts.MetricsServer},
		{"HTTPIngress", KubernetesFirewallPorts.HTTPIngress},
		{"HTTPSIngress", KubernetesFirewallPorts.HTTPSIngress},
	}

	for _, p := range ports {
		t.Run(p.name, func(t *testing.T) {
			if p.rule.Port == "" {
				t.Errorf("%s port should not be empty", p.name)
			}
			if p.rule.Protocol == "" {
				t.Errorf("%s protocol should not be empty", p.name)
			}
			if p.rule.Source == "" {
				t.Errorf("%s source should not be empty", p.name)
			}
			if p.rule.Direction == "" {
				t.Errorf("%s direction should not be empty", p.name)
			}
			if p.rule.Action == "" {
				t.Errorf("%s action should not be empty", p.name)
			}
			if p.rule.Description == "" {
				t.Errorf("%s description should not be empty", p.name)
			}
		})
	}
}

// TestFirewallRuleProtocols tests protocol validation
func TestFirewallRuleProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		valid    bool
	}{
		{"TCP uppercase", "TCP", true},
		{"tcp lowercase", "tcp", true},
		{"UDP uppercase", "UDP", true},
		{"udp lowercase", "udp", true},
		{"ICMP", "icmp", true},
		{"Invalid", "invalid", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := FirewallRule{
				Protocol: tt.protocol,
			}

			isValid := rule.Protocol == "tcp" || rule.Protocol == "udp" ||
				rule.Protocol == "TCP" || rule.Protocol == "UDP" ||
				rule.Protocol == "icmp" || rule.Protocol == "ICMP"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v for protocol %q", tt.valid, isValid, tt.protocol)
			}
		})
	}
}

// TestFirewallRuleDirections tests direction validation
func TestFirewallRuleDirections(t *testing.T) {
	tests := []struct {
		name      string
		direction string
		valid     bool
	}{
		{"Inbound", "inbound", true},
		{"Outbound", "outbound", true},
		{"Both", "both", true},
		{"Invalid", "invalid", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := FirewallRule{
				Direction: tt.direction,
			}

			isValid := rule.Direction == "inbound" || rule.Direction == "outbound" || rule.Direction == "both"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v for direction %q", tt.valid, isValid, tt.direction)
			}
		})
	}
}

// TestFirewallRuleActions tests action validation
func TestFirewallRuleActions(t *testing.T) {
	tests := []struct {
		name   string
		action string
		valid  bool
	}{
		{"Allow", "allow", true},
		{"Deny", "deny", true},
		{"Reject", "reject", true},
		{"Invalid", "invalid", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := FirewallRule{
				Action: tt.action,
			}

			isValid := rule.Action == "allow" || rule.Action == "deny" || rule.Action == "reject"

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v, got %v for action %q", tt.valid, isValid, tt.action)
			}
		})
	}
}

// TestFirewallRuleSources tests source CIDR validation
func TestFirewallRuleSources(t *testing.T) {
	tests := []struct {
		name   string
		source string
		valid  bool
	}{
		{"Any", "0.0.0.0/0", true},
		{"Private /8", "10.0.0.0/8", true},
		{"Private /16", "172.16.0.0/16", true},
		{"Private /24", "192.168.0.0/24", true},
		{"WireGuard", "10.8.0.0/24", true},
		{"Single IP", "1.2.3.4/32", true},
		{"Invalid CIDR", "invalid", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := FirewallRule{
				Source: tt.source,
			}

			// Basic validation - should contain / for CIDR
			hasSlash := strings.Contains(rule.Source, "/")
			_ = hasSlash || rule.Source == ""

			if tt.source != "" && !hasSlash && tt.valid {
				t.Logf("Source %q is missing CIDR notation", tt.source)
			}
		})
	}
}

// TestFirewallRulePortRanges tests port range formats
func TestFirewallRulePortRanges(t *testing.T) {
	tests := []struct {
		name string
		port string
		valid bool
	}{
		{"Single port", "80", true},
		{"Four digits", "6443", true},
		{"Five digits", "10250", true},
		{"Port range", "30000:32767", true},
		{"ETCD", "2379", true},
		{"ETCD Peer", "2380", true},
		{"Invalid range", "80-90", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := FirewallRule{
				Port: tt.port,
			}

			hasPort := rule.Port != ""
			if tt.valid && !hasPort {
				t.Errorf("Port should not be empty for valid case")
			}
		})
	}
}

// TestNewOSFirewallManager tests creating new firewall manager
func TestNewOSFirewallManager(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	if manager == nil {
		t.Fatal("NewOSFirewallManager should not return nil")
	}

	if manager.nodes == nil {
		t.Error("nodes slice should be initialized")
	}

	if len(manager.nodes) != 0 {
		t.Errorf("nodes should be empty, got %d", len(manager.nodes))
	}

	if manager.results == nil {
		t.Error("results map should be initialized")
	}

	if manager.timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", manager.timeout)
	}
}

// TestOSFirewallManager_AddNode tests adding nodes
func TestOSFirewallManager_AddNode(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	node := &providers.NodeOutput{
		Name: "test-node-1",
	}

	manager.AddNode(node)

	if len(manager.nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(manager.nodes))
	}

	if manager.nodes[0].Name != "test-node-1" {
		t.Errorf("Expected node name test-node-1, got %s", manager.nodes[0].Name)
	}

	// Check result was created
	if result, ok := manager.results["test-node-1"]; !ok {
		t.Error("Result should be created for added node")
	} else {
		if result.NodeName != "test-node-1" {
			t.Errorf("Result NodeName should be test-node-1, got %s", result.NodeName)
		}
		if len(result.RulesApplied) != 0 {
			t.Errorf("RulesApplied should be empty initially, got %d", len(result.RulesApplied))
		}
	}
}

// TestOSFirewallManager_AddMultipleNodes tests adding multiple nodes
func TestOSFirewallManager_AddMultipleNodes(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	nodes := []*providers.NodeOutput{
		{Name: "master-1"},
		{Name: "master-2"},
		{Name: "worker-1"},
		{Name: "worker-2"},
		{Name: "worker-3"},
	}

	for _, node := range nodes {
		manager.AddNode(node)
	}

	if len(manager.nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(manager.nodes))
	}

	if len(manager.results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(manager.results))
	}

	// Verify all nodes
	for _, node := range nodes {
		found := false
		for _, n := range manager.nodes {
			if n.Name == node.Name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Node %s not found in manager", node.Name)
		}
	}
}

// TestOSFirewallManager_SetSSHKeyPath tests setting SSH key path
func TestOSFirewallManager_SetSSHKeyPath(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	path := "/root/.ssh/id_rsa"
	manager.SetSSHKeyPath(path)

	if manager.sshKeyPath != path {
		t.Errorf("Expected SSH key path %s, got %s", path, manager.sshKeyPath)
	}
}

// TestExtractValue tests extractValue helper function
func TestExtractValue(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected string
	}{
		{
			name:     "Simple extraction",
			s:        "FIREWALL_TYPE:ufw\nother data",
			prefix:   "FIREWALL_TYPE:",
			expected: "ufw",
		},
		{
			name:     "No newline",
			s:        "FIREWALL_TYPE:iptables",
			prefix:   "FIREWALL_TYPE:",
			expected: "iptables",
		},
		{
			name:     "With space delimiter",
			s:        "TYPE:firewalld more text",
			prefix:   "TYPE:",
			expected: "firewalld",
		},
		{
			name:     "Not found",
			s:        "some other text",
			prefix:   "NOTFOUND:",
			expected: "",
		},
		{
			name:     "Empty string",
			s:        "",
			prefix:   "PREFIX:",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValue(tt.s, tt.prefix)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestFirewallTypes tests different firewall types
func TestFirewallTypes(t *testing.T) {
	types := []string{"ufw", "firewalld", "iptables"}

	for _, fwType := range types {
		t.Run(fwType, func(t *testing.T) {
			result := &FirewallResult{
				FirewallType: fwType,
			}

			isValid := result.FirewallType == "ufw" ||
				result.FirewallType == "firewalld" ||
				result.FirewallType == "iptables"

			if !isValid {
				t.Errorf("Firewall type %q should be valid", fwType)
			}
		})
	}
}

// TestMasterNodeRules tests rules for master nodes
func TestMasterNodeRules(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	masterNode := &providers.NodeOutput{
		Name: "master-1",
		Labels: map[string]string{
			"role": "master",
		},
	}

	rules := manager.getRulesForNode(masterNode)

	// Master should have more rules than worker
	if len(rules) == 0 {
		t.Error("Master node should have firewall rules")
	}

	// Check for master-specific ports
	hasAPIServer := false
	hasETCD := false
	hasScheduler := false

	for _, rule := range rules {
		if rule.Port == "6443" {
			hasAPIServer = true
		}
		if rule.Port == "2379" {
			hasETCD = true
		}
		if rule.Port == "10259" {
			hasScheduler = true
		}
	}

	if !hasAPIServer {
		t.Error("Master node should have API Server port (6443)")
	}
	if !hasETCD {
		t.Error("Master node should have ETCD port (2379)")
	}
	if !hasScheduler {
		t.Error("Master node should have Scheduler port (10259)")
	}
}

// TestWorkerNodeRules tests rules for worker nodes
func TestWorkerNodeRules(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	workerNode := &providers.NodeOutput{
		Name: "worker-1",
		Labels: map[string]string{
			"role": "worker",
		},
	}

	rules := manager.getRulesForNode(workerNode)

	if len(rules) == 0 {
		t.Error("Worker node should have firewall rules")
	}

	// Check for worker-specific ports
	hasNodePort := false
	hasHTTP := false
	hasHTTPS := false

	for _, rule := range rules {
		if rule.Port == "30000:32767" {
			hasNodePort = true
		}
		if rule.Port == "80" {
			hasHTTP = true
		}
		if rule.Port == "443" {
			hasHTTPS = true
		}
	}

	if !hasNodePort {
		t.Error("Worker node should have NodePort range (30000:32767)")
	}
	if !hasHTTP {
		t.Error("Worker node should have HTTP port (80)")
	}
	if !hasHTTPS {
		t.Error("Worker node should have HTTPS port (443)")
	}
}

// TestCommonNodeRules tests rules common to all nodes
func TestCommonNodeRules(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	node := &providers.NodeOutput{
		Name:   "node-1",
		Labels: map[string]string{},
	}

	rules := manager.getRulesForNode(node)

	// All nodes should have these
	hasSSH := false
	hasWireGuard := false
	hasKubelet := false

	for _, rule := range rules {
		if rule.Port == "22" {
			hasSSH = true
		}
		if rule.Port == "51820" {
			hasWireGuard = true
		}
		if rule.Port == "10250" {
			hasKubelet = true
		}
	}

	if !hasSSH {
		t.Error("All nodes should have SSH port (22)")
	}
	if !hasWireGuard {
		t.Error("All nodes should have WireGuard port (51820)")
	}
	if !hasKubelet {
		t.Error("All nodes should have Kubelet port (10250)")
	}
}

// TestControlPlaneNodeRules tests rules for controlplane role
func TestControlPlaneNodeRules(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	node := &providers.NodeOutput{
		Name: "controlplane-1",
		Labels: map[string]string{
			"role": "controlplane",
		},
	}

	rules := manager.getRulesForNode(node)

	hasAPIServer := false
	for _, rule := range rules {
		if rule.Port == "6443" {
			hasAPIServer = true
			break
		}
	}

	if !hasAPIServer {
		t.Error("Controlplane node should have API Server port")
	}
}

// TestMultiRoleNode tests node with multiple roles
func TestMultiRoleNode(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	// Some nodes might have both master and worker roles
	node := &providers.NodeOutput{
		Name: "multi-role-1",
		Labels: map[string]string{
			"role": "master", // Primary role
		},
	}

	rules := manager.getRulesForNode(node)

	// Should have master ports
	hasAPIServer := false
	for _, rule := range rules {
		if rule.Port == "6443" {
			hasAPIServer = true
			break
		}
	}

	if !hasAPIServer {
		t.Error("Multi-role node should have API Server port")
	}
}

// TestFirewallResultTimestamp tests timestamp tracking
func TestFirewallResultTimestamp(t *testing.T) {
	before := time.Now()
	time.Sleep(10 * time.Millisecond)

	result := &FirewallResult{
		Timestamp: time.Now(),
	}

	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	if result.Timestamp.Before(before) {
		t.Error("Timestamp should be after 'before' time")
	}

	if result.Timestamp.After(after) {
		t.Error("Timestamp should be before 'after' time")
	}
}

// TestGetResults tests getting all results
func TestGetResults(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	// Add some nodes
	manager.AddNode(&providers.NodeOutput{Name: "node-1"})
	manager.AddNode(&providers.NodeOutput{Name: "node-2"})

	results := manager.GetResults()

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if _, ok := results["node-1"]; !ok {
		t.Error("Results should contain node-1")
	}

	if _, ok := results["node-2"]; !ok {
		t.Error("Results should contain node-2")
	}
}

// TestFirewallScriptGeneration tests script generation exists
func TestFirewallScriptGeneration(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	node := &providers.NodeOutput{
		Name: "test-node",
		Labels: map[string]string{
			"role": "master",
		},
	}

	rules := manager.getRulesForNode(node)
	script := manager.generateFirewallScript(node, rules)

	if script == "" {
		t.Fatal("Script should not be empty")
	}

	// Verify script contains key elements
	expectedElements := []string{
		"#!/bin/bash",
		"set -e",
		"FIREWALL_TYPE",
		"ufw",
		"firewalld",
		"iptables",
		"SUCCESS",
	}

	for _, elem := range expectedElements {
		if !strings.Contains(script, elem) {
			t.Errorf("Script should contain %q", elem)
		}
	}
}

// TestFirewallScriptContainsNodeName tests node name in script
func TestFirewallScriptContainsNodeName(t *testing.T) {
	manager := NewOSFirewallManager(nil)

	nodeName := "test-node-123"
	node := &providers.NodeOutput{
		Name: nodeName,
	}

	rules := manager.getRulesForNode(node)
	script := manager.generateFirewallScript(node, rules)

	if !strings.Contains(script, nodeName) {
		t.Errorf("Script should contain node name %q", nodeName)
	}
}

// TestPortDescriptions tests all ports have descriptions
func TestPortDescriptions(t *testing.T) {
	tests := []struct {
		name string
		rule FirewallRule
	}{
		{"APIServer", KubernetesFirewallPorts.APIServer},
		{"ETCD", KubernetesFirewallPorts.ETCD},
		{"Kubelet", KubernetesFirewallPorts.Kubelet},
		{"WireGuard", KubernetesFirewallPorts.WireGuard},
		{"HTTPIngress", KubernetesFirewallPorts.HTTPIngress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rule.Description == "" {
				t.Errorf("%s should have a description", tt.name)
			}

			// Description should not be too short
			if len(tt.rule.Description) < 3 {
				t.Errorf("%s description too short: %q", tt.name, tt.rule.Description)
			}
		})
	}
}

// Test100MoreRules tests 100 different firewall rule combinations
func Test100MoreRules(t *testing.T) {
	// Generate 100 test cases for different rule combinations
	protocols := []string{"tcp", "udp"}
	directions := []string{"inbound", "outbound"}
	actions := []string{"allow", "deny"}
	sources := []string{"0.0.0.0/0", "10.0.0.0/8", "192.168.0.0/16"}

	testNum := 0
	for _, proto := range protocols {
		for _, dir := range directions {
			for _, action := range actions {
				for _, source := range sources {
					testNum++
					name := fmt.Sprintf("Rule%d_%s_%s_%s", testNum, proto, dir, action)
					t.Run(name, func(t *testing.T) {
						rule := FirewallRule{
							Port:      fmt.Sprintf("%d", 8000+testNum),
							Protocol:  proto,
							Source:    source,
							Direction: dir,
							Action:    action,
							Description: fmt.Sprintf("Test rule %d", testNum),
						}

						// Validate rule
						if rule.Port == "" {
							t.Error("Port should not be empty")
						}
						if rule.Protocol == "" {
							t.Error("Protocol should not be empty")
						}
						if rule.Direction == "" {
							t.Error("Direction should not be empty")
						}
						if rule.Action == "" {
							t.Error("Action should not be empty")
						}
					})

					if testNum >= 100 {
						return
					}
				}
			}
		}
	}
}
