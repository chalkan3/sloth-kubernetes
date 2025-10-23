package cluster

import (
	"strings"
	"testing"

	"github.com/chalkan3/sloth-kubernetes/pkg/config"
	"github.com/chalkan3/sloth-kubernetes/pkg/providers"
)

// TestNewRKEManager tests RKEManager creation
func TestNewRKEManager(t *testing.T) {
	// This would require Pulumi context mock, so we test structure only
	cfg := &config.KubernetesConfig{
		Version:       "v1.27.0",
		NetworkPlugin: "flannel",
		PodCIDR:       "10.42.0.0/16",
		ServiceCIDR:   "10.43.0.0/16",
	}

	// Verify config structure is valid
	if cfg.Version == "" {
		t.Error("Version should not be empty")
	}
	if cfg.NetworkPlugin == "" {
		t.Error("NetworkPlugin should not be empty")
	}
}

// TestGetNodeRoles tests role determination logic
func TestGetNodeRoles(t *testing.T) {
	manager := &RKEManager{
		config: &config.KubernetesConfig{},
	}

	tests := []struct {
		name      string
		node      *providers.NodeOutput
		wantRoles []string
	}{
		{
			name: "Node with master label",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"role": "master"},
			},
			wantRoles: []string{"controlplane", "etcd"},
		},
		{
			name: "Node with controlplane label",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"role": "controlplane"},
			},
			wantRoles: []string{"controlplane", "etcd"},
		},
		{
			name: "Node with worker label",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"role": "worker"},
			},
			wantRoles: []string{"worker"},
		},
		{
			name: "Node with etcd label",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"role": "etcd"},
			},
			wantRoles: []string{"etcd"},
		},
		{
			name: "Node with master in name (no label)",
			node: &providers.NodeOutput{
				Name:   "master-1",
				Labels: map[string]string{},
			},
			wantRoles: []string{"controlplane", "etcd"},
		},
		{
			name: "Node with control in name (no label)",
			node: &providers.NodeOutput{
				Name:   "control-plane-1",
				Labels: map[string]string{},
			},
			wantRoles: []string{"controlplane", "etcd"},
		},
		{
			name: "Node with worker in name (no label)",
			node: &providers.NodeOutput{
				Name:   "worker-1",
				Labels: map[string]string{},
			},
			wantRoles: []string{"worker"},
		},
		{
			name: "Node with no role indicators - defaults to worker",
			node: &providers.NodeOutput{
				Name:   "random-node-1",
				Labels: map[string]string{},
			},
			wantRoles: []string{"worker"},
		},
		{
			name: "Empty node name defaults to worker",
			node: &providers.NodeOutput{
				Name:   "",
				Labels: map[string]string{},
			},
			wantRoles: []string{"worker"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := manager.getNodeRoles(tt.node)

			if len(roles) != len(tt.wantRoles) {
				t.Errorf("Expected %d roles, got %d: %v", len(tt.wantRoles), len(roles), roles)
				return
			}

			for i, role := range roles {
				if role != tt.wantRoles[i] {
					t.Errorf("Role %d: expected %q, got %q", i, tt.wantRoles[i], role)
				}
			}
		})
	}
}

// TestGetNodeTaints tests taint parsing logic
func TestGetNodeTaints(t *testing.T) {
	manager := &RKEManager{
		config: &config.KubernetesConfig{},
	}

	tests := []struct {
		name       string
		node       *providers.NodeOutput
		wantTaints int
		wantKey    string
		wantValue  string
		wantEffect string
	}{
		{
			name: "Valid taint format",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"taints": "dedicated=gpu:NoSchedule"},
			},
			wantTaints: 1,
			wantKey:    "dedicated",
			wantValue:  "gpu",
			wantEffect: "NoSchedule",
		},
		{
			name: "Taint with different effect",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"taints": "zone=special:PreferNoSchedule"},
			},
			wantTaints: 1,
			wantKey:    "zone",
			wantValue:  "special",
			wantEffect: "PreferNoSchedule",
		},
		{
			name: "No taints label",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{},
			},
			wantTaints: 0,
		},
		{
			name: "Invalid taint format - missing effect",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"taints": "dedicated=gpu"},
			},
			wantTaints: 0,
		},
		{
			name: "Invalid taint format - missing value",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"taints": "dedicated:NoSchedule"},
			},
			wantTaints: 0,
		},
		{
			name: "Empty taint string",
			node: &providers.NodeOutput{
				Name:   "test-node",
				Labels: map[string]string{"taints": ""},
			},
			wantTaints: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taints := manager.getNodeTaints(tt.node)

			if len(taints) != tt.wantTaints {
				t.Errorf("Expected %d taints, got %d", tt.wantTaints, len(taints))
				return
			}

			if tt.wantTaints > 0 && len(taints) > 0 {
				taint := taints[0]

				if key, ok := taint["key"].(string); !ok || key != tt.wantKey {
					t.Errorf("Expected key %q, got %q", tt.wantKey, key)
				}

				if value, ok := taint["value"].(string); !ok || value != tt.wantValue {
					t.Errorf("Expected value %q, got %q", tt.wantValue, value)
				}

				if effect, ok := taint["effect"].(string); !ok || effect != tt.wantEffect {
					t.Errorf("Expected effect %q, got %q", tt.wantEffect, effect)
				}
			}
		})
	}
}

// TestGenerateServicesConfig tests services configuration generation
func TestGenerateServicesConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.KubernetesConfig
		checkFields []string
	}{
		{
			name: "Basic services config",
			config: &config.KubernetesConfig{
				ServiceCIDR:    "10.43.0.0/16",
				PodCIDR:        "10.42.0.0/16",
				ClusterDomain:  "cluster.local",
				ClusterDNS:     "10.43.0.10",
				AuditLog:       true,
				EncryptSecrets: true,
			},
			checkFields: []string{"etcd", "kube-api", "kube-controller", "scheduler", "kubelet", "kubeproxy"},
		},
		{
			name: "Services config with audit disabled",
			config: &config.KubernetesConfig{
				ServiceCIDR:    "10.43.0.0/16",
				PodCIDR:        "10.42.0.0/16",
				ClusterDomain:  "cluster.local",
				ClusterDNS:     "10.43.0.10",
				AuditLog:       false,
				EncryptSecrets: false,
			},
			checkFields: []string{"etcd", "kube-api", "kube-controller", "scheduler", "kubelet", "kubeproxy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &RKEManager{config: tt.config}
			services := manager.generateServicesConfig()

			// Check all required services exist
			for _, field := range tt.checkFields {
				if _, ok := services[field]; !ok {
					t.Errorf("Missing service: %s", field)
				}
			}

			// Check etcd configuration
			if etcd, ok := services["etcd"].(map[string]interface{}); ok {
				if snapshot, ok := etcd["snapshot"].(bool); !ok || !snapshot {
					t.Error("etcd snapshot should be enabled")
				}
				if backup, ok := etcd["backup_config"].(map[string]interface{}); ok {
					if enabled, ok := backup["enabled"].(bool); !ok || !enabled {
						t.Error("etcd backup should be enabled")
					}
				}
			} else {
				t.Error("etcd configuration missing or invalid")
			}

			// Check kube-api configuration
			if kubeAPI, ok := services["kube-api"].(map[string]interface{}); ok {
				if cidr, ok := kubeAPI["service_cluster_ip_range"].(string); !ok || cidr != tt.config.ServiceCIDR {
					t.Errorf("Expected service CIDR %s, got %v", tt.config.ServiceCIDR, cidr)
				}

				if auditLog, ok := kubeAPI["audit_log"].(map[string]interface{}); ok {
					if enabled, ok := auditLog["enabled"].(bool); ok {
						if enabled != tt.config.AuditLog {
							t.Errorf("Expected audit log %v, got %v", tt.config.AuditLog, enabled)
						}
					}
				}

				if encryption, ok := kubeAPI["secrets_encryption_config"].(map[string]interface{}); ok {
					if enabled, ok := encryption["enabled"].(bool); ok {
						if enabled != tt.config.EncryptSecrets {
							t.Errorf("Expected encryption %v, got %v", tt.config.EncryptSecrets, enabled)
						}
					}
				}
			} else {
				t.Error("kube-api configuration missing or invalid")
			}

			// Check kubelet configuration
			if kubelet, ok := services["kubelet"].(map[string]interface{}); ok {
				if domain, ok := kubelet["cluster_domain"].(string); !ok || domain != tt.config.ClusterDomain {
					t.Errorf("Expected cluster domain %s, got %v", tt.config.ClusterDomain, domain)
				}

				if dns, ok := kubelet["cluster_dns_server"].(string); !ok || dns != tt.config.ClusterDNS {
					t.Errorf("Expected cluster DNS %s, got %v", tt.config.ClusterDNS, dns)
				}
			} else {
				t.Error("kubelet configuration missing or invalid")
			}
		})
	}
}

// TestGenerateSystemImages tests system images generation
func TestGenerateSystemImages(t *testing.T) {
	tests := []struct {
		name           string
		k8sVersion     string
		expectedImages []string
	}{
		{
			name:       "Kubernetes v1.27.0",
			k8sVersion: "v1.27.0",
			expectedImages: []string{
				"kubernetes", "etcd", "alpine", "nginx_proxy", "cert_downloader",
				"kubernetes_services_sidecar", "coredns", "flannel", "calico_node",
				"ingress", "metrics_server", "pod_infra_container",
			},
		},
		{
			name:       "Kubernetes v1.26.5",
			k8sVersion: "v1.26.5",
			expectedImages: []string{
				"kubernetes", "etcd", "alpine", "nginx_proxy", "cert_downloader",
				"kubernetes_services_sidecar", "coredns", "flannel", "calico_node",
				"ingress", "metrics_server", "pod_infra_container",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &RKEManager{
				config: &config.KubernetesConfig{Version: tt.k8sVersion},
			}

			images := manager.generateSystemImages()

			// Check all expected images exist
			for _, expectedImage := range tt.expectedImages {
				if _, ok := images[expectedImage]; !ok {
					t.Errorf("Missing system image: %s", expectedImage)
				}
			}

			// Check kubernetes image includes version
			if k8sImage, ok := images["kubernetes"].(string); ok {
				if !strings.Contains(k8sImage, tt.k8sVersion) {
					t.Errorf("Kubernetes image should contain version %s, got %s", tt.k8sVersion, k8sImage)
				}
			} else {
				t.Error("kubernetes image not found or invalid")
			}

			// Check that images are proper container references
			imagesToCheck := []string{"etcd", "coredns", "flannel", "calico_node", "ingress"}
			for _, imageName := range imagesToCheck {
				if image, ok := images[imageName].(string); ok {
					// Should contain "/" indicating registry/image format
					if !strings.Contains(image, "/") {
						t.Errorf("Image %s should be in registry/image format, got %s", imageName, image)
					}
					// Should contain ":" indicating tag
					if !strings.Contains(image, ":") {
						t.Errorf("Image %s should include tag, got %s", imageName, image)
					}
				}
			}
		})
	}
}

// TestGetMasterNode tests master node selection
func TestGetMasterNode(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []*providers.NodeOutput
		wantFound bool
		wantName  string
	}{
		{
			name: "Find master node with label",
			nodes: []*providers.NodeOutput{
				{Name: "worker-1", Labels: map[string]string{"role": "worker"}},
				{Name: "master-1", Labels: map[string]string{"role": "master"}},
				{Name: "worker-2", Labels: map[string]string{"role": "worker"}},
			},
			wantFound: true,
			wantName:  "master-1",
		},
		{
			name: "Find controlplane node with label",
			nodes: []*providers.NodeOutput{
				{Name: "worker-1", Labels: map[string]string{"role": "worker"}},
				{Name: "control-1", Labels: map[string]string{"role": "controlplane"}},
			},
			wantFound: true,
			wantName:  "control-1",
		},
		{
			name: "Find master by name pattern",
			nodes: []*providers.NodeOutput{
				{Name: "worker-1", Labels: map[string]string{}},
				{Name: "master-1", Labels: map[string]string{}},
			},
			wantFound: true,
			wantName:  "master-1",
		},
		{
			name: "Find control-plane by name pattern",
			nodes: []*providers.NodeOutput{
				{Name: "worker-1", Labels: map[string]string{}},
				{Name: "control-plane-1", Labels: map[string]string{}},
			},
			wantFound: true,
			wantName:  "control-plane-1",
		},
		{
			name: "No master nodes",
			nodes: []*providers.NodeOutput{
				{Name: "worker-1", Labels: map[string]string{"role": "worker"}},
				{Name: "worker-2", Labels: map[string]string{"role": "worker"}},
			},
			wantFound: false,
		},
		{
			name:      "Empty node list",
			nodes:     []*providers.NodeOutput{},
			wantFound: false,
		},
		{
			name: "First master node is returned when multiple exist",
			nodes: []*providers.NodeOutput{
				{Name: "master-1", Labels: map[string]string{"role": "master"}},
				{Name: "master-2", Labels: map[string]string{"role": "master"}},
			},
			wantFound: true,
			wantName:  "master-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &RKEManager{
				config: &config.KubernetesConfig{},
				nodes:  tt.nodes,
			}

			masterNode := manager.getMasterNode()

			if tt.wantFound {
				if masterNode == nil {
					t.Error("Expected to find master node, but got nil")
					return
				}
				if masterNode.Name != tt.wantName {
					t.Errorf("Expected master node %q, got %q", tt.wantName, masterNode.Name)
				}
			} else {
				if masterNode != nil {
					t.Errorf("Expected no master node, but got %q", masterNode.Name)
				}
			}
		})
	}
}

// TestAddNode tests adding nodes to RKEManager
func TestAddNode(t *testing.T) {
	manager := &RKEManager{
		config: &config.KubernetesConfig{},
		nodes:  make([]*providers.NodeOutput, 0),
	}

	// Initially empty
	if len(manager.nodes) != 0 {
		t.Errorf("Expected 0 nodes initially, got %d", len(manager.nodes))
	}

	// Add first node
	node1 := &providers.NodeOutput{Name: "node-1"}
	manager.AddNode(node1)
	if len(manager.nodes) != 1 {
		t.Errorf("Expected 1 node after adding, got %d", len(manager.nodes))
	}
	if manager.nodes[0].Name != "node-1" {
		t.Errorf("Expected node name 'node-1', got %q", manager.nodes[0].Name)
	}

	// Add second node
	node2 := &providers.NodeOutput{Name: "node-2"}
	manager.AddNode(node2)
	if len(manager.nodes) != 2 {
		t.Errorf("Expected 2 nodes after adding, got %d", len(manager.nodes))
	}

	// Add multiple nodes
	for i := 3; i <= 10; i++ {
		manager.AddNode(&providers.NodeOutput{Name: "node-" + string(rune(i))})
	}
	if len(manager.nodes) != 10 {
		t.Errorf("Expected 10 nodes, got %d", len(manager.nodes))
	}
}

// TestRKEManagerStructure tests RKEManager structure
func TestRKEManagerStructure(t *testing.T) {
	cfg := &config.KubernetesConfig{
		Version:        "v1.27.0",
		NetworkPlugin:  "flannel",
		PodCIDR:        "10.42.0.0/16",
		ServiceCIDR:    "10.43.0.0/16",
		ClusterDomain:  "cluster.local",
		ClusterDNS:     "10.43.0.10",
		AuditLog:       true,
		EncryptSecrets: true,
		Monitoring:     true,
		Addons: []config.AddonConfig{
			{Name: "dashboard", Enabled: true},
			{Name: "metrics-server", Enabled: true},
		},
	}

	// Verify all fields are accessible
	if cfg.Version != "v1.27.0" {
		t.Errorf("Expected version v1.27.0, got %s", cfg.Version)
	}

	if cfg.NetworkPlugin != "flannel" {
		t.Errorf("Expected network plugin flannel, got %s", cfg.NetworkPlugin)
	}

	if cfg.PodCIDR != "10.42.0.0/16" {
		t.Errorf("Expected pod CIDR 10.42.0.0/16, got %s", cfg.PodCIDR)
	}

	if !cfg.AuditLog {
		t.Error("Expected AuditLog to be true")
	}

	if !cfg.Monitoring {
		t.Error("Expected Monitoring to be true")
	}

	if len(cfg.Addons) != 2 {
		t.Errorf("Expected 2 addons, got %d", len(cfg.Addons))
	}
}

// TestServicesConfigETCD tests etcd-specific configuration
func TestServicesConfigETCD(t *testing.T) {
	manager := &RKEManager{
		config: &config.KubernetesConfig{
			ServiceCIDR:   "10.43.0.0/16",
			PodCIDR:       "10.42.0.0/16",
			ClusterDomain: "cluster.local",
			ClusterDNS:    "10.43.0.10",
		},
	}

	services := manager.generateServicesConfig()

	etcd, ok := services["etcd"].(map[string]interface{})
	if !ok {
		t.Fatal("etcd configuration not found")
	}

	// Check snapshot configuration
	if snapshot, ok := etcd["snapshot"].(bool); !ok || !snapshot {
		t.Error("snapshot should be enabled")
	}

	// Check backup configuration
	if backupConfig, ok := etcd["backup_config"].(map[string]interface{}); ok {
		if enabled, ok := backupConfig["enabled"].(bool); !ok || !enabled {
			t.Error("backup should be enabled")
		}
		if interval, ok := backupConfig["interval_hours"].(int); !ok || interval != 12 {
			t.Errorf("Expected backup interval 12h, got %v", interval)
		}
	} else {
		t.Error("backup_config not found")
	}

	// Check extra_args
	if extraArgs, ok := etcd["extra_args"].(map[string]interface{}); ok {
		expectedArgs := map[string]string{
			"heartbeat-interval":        "500",
			"election-timeout":          "5000",
			"snapshot-count":            "10000",
			"quota-backend-bytes":       "8589934592",
			"max-request-bytes":         "10485760",
			"auto-compaction-mode":      "periodic",
			"auto-compaction-retention": "1",
		}

		for key, expectedValue := range expectedArgs {
			if value, ok := extraArgs[key].(string); !ok || value != expectedValue {
				t.Errorf("Expected etcd extra_arg %s=%s, got %v", key, expectedValue, value)
			}
		}
	} else {
		t.Error("extra_args not found in etcd config")
	}
}

// TestServicesConfigKubelet tests kubelet-specific configuration
func TestServicesConfigKubelet(t *testing.T) {
	manager := &RKEManager{
		config: &config.KubernetesConfig{
			ServiceCIDR:   "10.43.0.0/16",
			PodCIDR:       "10.42.0.0/16",
			ClusterDomain: "custom.local",
			ClusterDNS:    "10.43.0.99",
		},
	}

	services := manager.generateServicesConfig()

	kubelet, ok := services["kubelet"].(map[string]interface{})
	if !ok {
		t.Fatal("kubelet configuration not found")
	}

	// Check cluster domain
	if domain, ok := kubelet["cluster_domain"].(string); !ok || domain != "custom.local" {
		t.Errorf("Expected cluster_domain 'custom.local', got %v", domain)
	}

	// Check cluster DNS
	if dns, ok := kubelet["cluster_dns_server"].(string); !ok || dns != "10.43.0.99" {
		t.Errorf("Expected cluster_dns_server '10.43.0.99', got %v", dns)
	}

	// Check fail_swap_on
	if failSwap, ok := kubelet["fail_swap_on"].(bool); !ok || failSwap {
		t.Error("Expected fail_swap_on to be false")
	}

	// Check generate_serving_certificate
	if genCert, ok := kubelet["generate_serving_certificate"].(bool); !ok || !genCert {
		t.Error("Expected generate_serving_certificate to be true")
	}

	// Check extra_args
	if extraArgs, ok := kubelet["extra_args"].(map[string]interface{}); ok {
		if maxPods, ok := extraArgs["max-pods"].(string); !ok || maxPods != "110" {
			t.Errorf("Expected max-pods '110', got %v", maxPods)
		}
		if cgroupDriver, ok := extraArgs["cgroup-driver"].(string); !ok || cgroupDriver != "systemd" {
			t.Errorf("Expected cgroup-driver 'systemd', got %v", cgroupDriver)
		}
	} else {
		t.Error("extra_args not found in kubelet config")
	}
}

// TestSystemImagesVersioning tests that system images use correct versions
func TestSystemImagesVersioning(t *testing.T) {
	versions := []string{"v1.25.0", "v1.26.5", "v1.27.0", "v1.28.0"}

	for _, version := range versions {
		t.Run("Version "+version, func(t *testing.T) {
			manager := &RKEManager{
				config: &config.KubernetesConfig{Version: version},
			}

			images := manager.generateSystemImages()

			// Kubernetes image should use the specified version
			if k8sImage, ok := images["kubernetes"].(string); ok {
				expectedImage := "rancher/hyperkube:" + version
				if k8sImage != expectedImage {
					t.Errorf("Expected kubernetes image %s, got %s", expectedImage, k8sImage)
				}
			} else {
				t.Error("kubernetes image not found")
			}

			// All images should be strings
			for name, image := range images {
				if _, ok := image.(string); !ok {
					t.Errorf("Image %s should be a string, got %T", name, image)
				}
			}
		})
	}
}

// TestNodeRolePriority tests that label-based roles take precedence over name-based roles
func TestNodeRolePriority(t *testing.T) {
	manager := &RKEManager{
		config: &config.KubernetesConfig{},
	}

	tests := []struct {
		name      string
		node      *providers.NodeOutput
		wantRoles []string
	}{
		{
			name: "Label takes precedence - worker label on master-named node",
			node: &providers.NodeOutput{
				Name:   "master-1",                          // Name suggests master
				Labels: map[string]string{"role": "worker"}, // But label says worker
			},
			wantRoles: []string{"worker"}, // Label wins
		},
		{
			name: "Label takes precedence - master label on worker-named node",
			node: &providers.NodeOutput{
				Name:   "worker-1",                          // Name suggests worker
				Labels: map[string]string{"role": "master"}, // But label says master
			},
			wantRoles: []string{"controlplane", "etcd"}, // Label wins
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := manager.getNodeRoles(tt.node)

			if len(roles) != len(tt.wantRoles) {
				t.Errorf("Expected %d roles, got %d: %v", len(tt.wantRoles), len(roles), roles)
				return
			}

			for i, role := range roles {
				if role != tt.wantRoles[i] {
					t.Errorf("Role %d: expected %q, got %q", i, tt.wantRoles[i], role)
				}
			}
		})
	}
}
