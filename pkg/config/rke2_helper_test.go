package config

import (
	"strings"
	"testing"
)

func TestGetRKE2Defaults(t *testing.T) {
	defaults := GetRKE2Defaults()

	if defaults == nil {
		t.Fatal("GetRKE2Defaults() returned nil")
	}

	if defaults.Channel != "stable" {
		t.Errorf("Expected channel 'stable', got '%s'", defaults.Channel)
	}

	if defaults.DataDir != "/var/lib/rancher/rke2" {
		t.Errorf("Expected data dir '/var/lib/rancher/rke2', got '%s'", defaults.DataDir)
	}

	if defaults.SnapshotScheduleCron != "0 */12 * * *" {
		t.Errorf("Expected snapshot schedule '0 */12 * * *', got '%s'", defaults.SnapshotScheduleCron)
	}

	if defaults.SnapshotRetention != 5 {
		t.Errorf("Expected snapshot retention 5, got %d", defaults.SnapshotRetention)
	}

	if defaults.WriteKubeconfigMode != "0600" {
		t.Errorf("Expected kubeconfig mode '0600', got '%s'", defaults.WriteKubeconfigMode)
	}

	if len(defaults.DisableComponents) == 0 {
		t.Error("Expected default disabled components")
	}
}

func TestBuildRKE2ServerConfig(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *RKE2Config
		nodeIP        string
		nodeName      string
		isFirstMaster bool
		firstMasterIP string
		k8sConfig     *KubernetesConfig
		wantContains  []string
	}{
		{
			name: "First master",
			cfg: &RKE2Config{
				ClusterToken:        "test-token",
				TLSSan:              []string{"api.example.com"},
				DisableComponents:   []string{"rke2-ingress-nginx"},
				SnapshotScheduleCron: "0 */6 * * *",
				SnapshotRetention:   3,
				SecretsEncryption:   true,
			},
			nodeIP:        "10.0.0.1",
			nodeName:      "master-1",
			isFirstMaster: true,
			firstMasterIP: "",
			k8sConfig: &KubernetesConfig{
				PodCIDR:       "10.42.0.0/16",
				ServiceCIDR:   "10.43.0.0/16",
				ClusterDNS:    "10.43.0.10",
				NetworkPlugin: "calico",
			},
			wantContains: []string{
				"token: test-token",
				"node-name: master-1",
				"node-ip: 10.0.0.1",
				"advertise-address: 10.0.0.1",
				"tls-san:",
				"api.example.com",
				"cluster-cidr: 10.42.0.0/16",
				"service-cidr: 10.43.0.0/16",
				"cluster-dns: 10.43.0.10",
				"cni:",
				"calico",
				"disable:",
				"rke2-ingress-nginx",
				"etcd-snapshot-schedule-cron: 0 */6 * * *",
				"etcd-snapshot-retention: 3",
				"secrets-encryption: true",
			},
		},
		{
			name: "Additional master",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
			},
			nodeIP:        "10.0.0.2",
			nodeName:      "master-2",
			isFirstMaster: false,
			firstMasterIP: "10.0.0.1",
			k8sConfig:     &KubernetesConfig{},
			wantContains: []string{
				"token: test-token",
				"server: https://10.0.0.1:9345",
				"node-name: master-2",
				"node-ip: 10.0.0.2",
			},
		},
		{
			name: "With node taints and labels",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
				NodeTaint:    []string{"node-role.kubernetes.io/master:NoSchedule"},
				NodeLabel:    []string{"node-type=master", "zone=us-east"},
			},
			nodeIP:        "10.0.0.1",
			nodeName:      "master-1",
			isFirstMaster: true,
			firstMasterIP: "",
			k8sConfig:     &KubernetesConfig{},
			wantContains: []string{
				"node-taint:",
				"node-role.kubernetes.io/master:NoSchedule",
				"node-label:",
				"node-type=master",
				"zone=us-east",
			},
		},
		{
			name: "With custom data dir",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
				DataDir:      "/custom/data/dir",
			},
			nodeIP:        "10.0.0.1",
			nodeName:      "master-1",
			isFirstMaster: true,
			firstMasterIP: "",
			k8sConfig:     &KubernetesConfig{},
			wantContains: []string{
				"data-dir: /custom/data/dir",
			},
		},
		{
			name: "With security settings",
			cfg: &RKE2Config{
				ClusterToken:          "test-token",
				SeLinux:               true,
				ProtectKernelDefaults: true,
				WriteKubeconfigMode:   "0640",
			},
			nodeIP:        "10.0.0.1",
			nodeName:      "master-1",
			isFirstMaster: true,
			firstMasterIP: "",
			k8sConfig:     &KubernetesConfig{},
			wantContains: []string{
				"selinux: true",
				"protect-kernel-defaults: true",
				"write-kubeconfig-mode: 0640",
			},
		},
		{
			name: "With system default registry",
			cfg: &RKE2Config{
				ClusterToken:          "test-token",
				SystemDefaultRegistry: "registry.example.com",
			},
			nodeIP:        "10.0.0.1",
			nodeName:      "master-1",
			isFirstMaster: true,
			firstMasterIP: "",
			k8sConfig:     &KubernetesConfig{},
			wantContains: []string{
				"system-default-registry: registry.example.com",
			},
		},
		{
			name: "With CIS profiles",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
				Profiles:     []string{"cis-1.6", "cis-1.23"},
			},
			nodeIP:        "10.0.0.1",
			nodeName:      "master-1",
			isFirstMaster: true,
			firstMasterIP: "",
			k8sConfig:     &KubernetesConfig{},
			wantContains: []string{
				"profile:",
				"cis-1.6",
				"cis-1.23",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BuildRKE2ServerConfig(tt.cfg, tt.nodeIP, tt.nodeName, tt.isFirstMaster, tt.firstMasterIP, tt.k8sConfig)

			for _, want := range tt.wantContains {
				if !strings.Contains(config, want) {
					t.Errorf("Config should contain '%s'\nGot:\n%s", want, config)
				}
			}
		})
	}
}

func TestBuildRKE2AgentConfig(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *RKE2Config
		nodeIP       string
		nodeName     string
		serverIP     string
		wantContains []string
	}{
		{
			name: "Basic agent config",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
			},
			nodeIP:   "10.0.0.10",
			nodeName: "worker-1",
			serverIP: "10.0.0.1",
			wantContains: []string{
				"token: test-token",
				"server: https://10.0.0.1:9345",
				"node-name: worker-1",
				"node-ip: 10.0.0.10",
			},
		},
		{
			name: "Agent with taints and labels",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
				NodeTaint:    []string{"workload=gpu:NoSchedule"},
				NodeLabel:    []string{"node-type=worker", "gpu=nvidia"},
			},
			nodeIP:   "10.0.0.10",
			nodeName: "worker-gpu-1",
			serverIP: "10.0.0.1",
			wantContains: []string{
				"node-taint:",
				"workload=gpu:NoSchedule",
				"node-label:",
				"node-type=worker",
				"gpu=nvidia",
			},
		},
		{
			name: "Agent with custom data dir",
			cfg: &RKE2Config{
				ClusterToken: "test-token",
				DataDir:      "/custom/agent/dir",
			},
			nodeIP:   "10.0.0.10",
			nodeName: "worker-1",
			serverIP: "10.0.0.1",
			wantContains: []string{
				"data-dir: /custom/agent/dir",
			},
		},
		{
			name: "Agent with security settings",
			cfg: &RKE2Config{
				ClusterToken:          "test-token",
				SeLinux:               true,
				ProtectKernelDefaults: true,
			},
			nodeIP:   "10.0.0.10",
			nodeName: "worker-1",
			serverIP: "10.0.0.1",
			wantContains: []string{
				"selinux: true",
				"protect-kernel-defaults: true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BuildRKE2AgentConfig(tt.cfg, tt.nodeIP, tt.nodeName, tt.serverIP)

			for _, want := range tt.wantContains {
				if !strings.Contains(config, want) {
					t.Errorf("Config should contain '%s'\nGot:\n%s", want, config)
				}
			}
		})
	}
}

func TestGetRKE2InstallCommand(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *RKE2Config
		isServer     bool
		wantContains []string
	}{
		{
			name: "Server installation with version",
			cfg: &RKE2Config{
				Version: "v1.28.5+rke2r1",
			},
			isServer: true,
			wantContains: []string{
				"curl -sfL https://get.rke2.io",
				"INSTALL_RKE2_TYPE=server",
				"INSTALL_RKE2_VERSION=v1.28.5+rke2r1",
			},
		},
		{
			name: "Agent installation with version",
			cfg: &RKE2Config{
				Version: "v1.28.5+rke2r1",
			},
			isServer: false,
			wantContains: []string{
				"curl -sfL https://get.rke2.io",
				"INSTALL_RKE2_TYPE=agent",
				"INSTALL_RKE2_VERSION=v1.28.5+rke2r1",
			},
		},
		{
			name: "Server installation with channel",
			cfg: &RKE2Config{
				Channel: "stable",
			},
			isServer: true,
			wantContains: []string{
				"curl -sfL https://get.rke2.io",
				"INSTALL_RKE2_TYPE=server",
				"INSTALL_RKE2_CHANNEL=stable",
			},
		},
		{
			name:     "Default installation",
			cfg:      &RKE2Config{},
			isServer: true,
			wantContains: []string{
				"curl -sfL https://get.rke2.io",
				"INSTALL_RKE2_TYPE=server",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := GetRKE2InstallCommand(tt.cfg, tt.isServer)

			for _, want := range tt.wantContains {
				if !strings.Contains(cmd, want) {
					t.Errorf("Command should contain '%s'\nGot: %s", want, cmd)
				}
			}
		})
	}
}

func TestMergeRKE2Config(t *testing.T) {
	tests := []struct {
		name string
		user *RKE2Config
		want func(*RKE2Config) bool
	}{
		{
			name: "Nil user config returns defaults",
			user: nil,
			want: func(cfg *RKE2Config) bool {
				return cfg.Channel == "stable" && cfg.DataDir == "/var/lib/rancher/rke2"
			},
		},
		{
			name: "User config overrides defaults",
			user: &RKE2Config{
				Version:      "v1.29.0+rke2r1",
				Channel:      "latest",
				ClusterToken: "custom-token",
			},
			want: func(cfg *RKE2Config) bool {
				return cfg.Version == "v1.29.0+rke2r1" &&
					cfg.Channel == "latest" &&
					cfg.ClusterToken == "custom-token"
			},
		},
		{
			name: "User TLSSan overrides defaults",
			user: &RKE2Config{
				TLSSan: []string{"api.example.com", "api2.example.com"},
			},
			want: func(cfg *RKE2Config) bool {
				return len(cfg.TLSSan) == 2 &&
					cfg.TLSSan[0] == "api.example.com"
			},
		},
		{
			name: "User DisableComponents overrides defaults",
			user: &RKE2Config{
				DisableComponents: []string{"rke2-ingress-nginx", "rke2-metrics-server"},
			},
			want: func(cfg *RKE2Config) bool {
				return len(cfg.DisableComponents) == 2
			},
		},
		{
			name: "User DataDir overrides default",
			user: &RKE2Config{
				DataDir: "/custom/data/dir",
			},
			want: func(cfg *RKE2Config) bool {
				return cfg.DataDir == "/custom/data/dir"
			},
		},
		{
			name: "User snapshot settings override defaults",
			user: &RKE2Config{
				SnapshotScheduleCron: "0 */6 * * *",
				SnapshotRetention:   10,
			},
			want: func(cfg *RKE2Config) bool {
				return cfg.SnapshotScheduleCron == "0 */6 * * *" &&
					cfg.SnapshotRetention == 10
			},
		},
		{
			name: "User security settings override defaults",
			user: &RKE2Config{
				SeLinux:               true,
				SecretsEncryption:     true,
				ProtectKernelDefaults: true,
				WriteKubeconfigMode:   "0640",
			},
			want: func(cfg *RKE2Config) bool {
				return cfg.SeLinux &&
					cfg.SecretsEncryption &&
					cfg.ProtectKernelDefaults &&
					cfg.WriteKubeconfigMode == "0640"
			},
		},
		{
			name: "User system default registry overrides default",
			user: &RKE2Config{
				SystemDefaultRegistry: "registry.example.com",
			},
			want: func(cfg *RKE2Config) bool {
				return cfg.SystemDefaultRegistry == "registry.example.com"
			},
		},
		{
			name: "User profiles override defaults",
			user: &RKE2Config{
				Profiles: []string{"cis-1.6"},
			},
			want: func(cfg *RKE2Config) bool {
				return len(cfg.Profiles) == 1 && cfg.Profiles[0] == "cis-1.6"
			},
		},
		{
			name: "User extra args override defaults",
			user: &RKE2Config{
				ExtraServerArgs: map[string]string{"key1": "value1"},
				ExtraAgentArgs:  map[string]string{"key2": "value2"},
			},
			want: func(cfg *RKE2Config) bool {
				return len(cfg.ExtraServerArgs) == 1 &&
					len(cfg.ExtraAgentArgs) == 1 &&
					cfg.ExtraServerArgs["key1"] == "value1" &&
					cfg.ExtraAgentArgs["key2"] == "value2"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := MergeRKE2Config(tt.user)

			if merged == nil {
				t.Fatal("MergeRKE2Config() returned nil")
			}

			if !tt.want(merged) {
				t.Errorf("MergeRKE2Config() result doesn't match expectations")
			}
		})
	}
}
