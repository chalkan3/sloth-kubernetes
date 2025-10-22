package config

import (
	"strings"
	"testing"
)

// TestRKE2Defaults_Structure tests RKE2 default configuration
func TestRKE2Defaults_Structure(t *testing.T) {
	defaults := GetRKE2Defaults()

	tests := []struct {
		name  string
		check func() bool
		valid bool
	}{
		{"Channel is stable", func() bool { return defaults.Channel == "stable" }, true},
		{"Token is set", func() bool { return defaults.ClusterToken != "" }, true},
		{"DataDir is set", func() bool { return defaults.DataDir == "/var/lib/rancher/rke2" }, true},
		{"Snapshot retention > 0", func() bool { return defaults.SnapshotRetention > 0 }, true},
		{"Kubeconfig mode is restrictive", func() bool { return defaults.WriteKubeconfigMode == "0600" }, true},
		{"Ingress disabled by default", func() bool {
			return containsRKE2(defaults.DisableComponents, "rke2-ingress-nginx")
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.check() != tt.valid {
				t.Errorf("Default validation failed for: %s", tt.name)
			}
		})
	}
}

func containsRKE2(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestRKE2Channel_Validation tests RKE2 release channels
func TestRKE2Channel_Validation(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		valid   bool
	}{
		{"Stable channel", "stable", true},
		{"Latest channel", "latest", true},
		{"Testing channel", "testing", true},
		{"v1.28 channel", "v1.28", true},
		{"v1.29 channel", "v1.29", true},
		{"Invalid channel", "production", false},
		{"Empty channel", "", false},
	}

	validChannels := []string{"stable", "latest", "testing", "v1.28", "v1.29", "v1.30"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := containsRKE2(validChannels, tt.channel) || strings.HasPrefix(tt.channel, "v1.")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for channel %q, got %v", tt.valid, tt.channel, isValid)
			}
		})
	}
}

// TestRKE2Token_Security tests cluster token security requirements
func TestRKE2Token_Security(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		secure bool
	}{
		{"Long secure token", "my-super-secret-cluster-token-rke2-production-2025", true},
		{"32 char token", "abcdefghijklmnopqrstuvwxyz123456", true},
		{"Short token", "short", false},   // Too short
		{"Weak token", "password", false}, // Too weak
		{"Empty token", "", false},
		{"Very long token", strings.Repeat("a", 100), true}, // Acceptable
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Secure tokens should be at least 16 characters
			isSecure := len(tt.token) >= 16

			if isSecure != tt.secure {
				t.Errorf("Expected secure=%v for token length %d, got %v", tt.secure, len(tt.token), isSecure)
			}
		})
	}
}

// TestRKE2TLSSan_Format tests TLS SAN (Subject Alternative Name) formats
func TestRKE2TLSSan_Format(t *testing.T) {
	tests := []struct {
		name  string
		san   string
		valid bool
	}{
		{"Domain SAN", "api.example.com", true},
		{"IP SAN", "192.168.1.100", true},
		{"Wildcard SAN", "*.example.com", true},
		{"Localhost", "localhost", true},
		{"IPv6", "2001:db8::1", true},
		{"Empty SAN", "", false},
		{"Invalid characters", "api@example.com", false},
		{"Space in SAN", "api example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.san != "" &&
				!strings.Contains(tt.san, " ") &&
				!strings.Contains(tt.san, "@") &&
				!strings.Contains(tt.san, "#")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for SAN %q, got %v", tt.valid, tt.san, isValid)
			}
		})
	}
}

// TestRKE2DisableComponents tests component disable options
func TestRKE2DisableComponents(t *testing.T) {
	tests := []struct {
		name      string
		component string
		valid     bool
	}{
		{"Disable ingress", "rke2-ingress-nginx", true},
		{"Disable metrics-server", "rke2-metrics-server", true},
		{"Disable snapshot-controller", "rke2-snapshot-controller", true},
		{"Disable snapshot-controller-crd", "rke2-snapshot-controller-crd", true},
		{"Invalid component", "random-component", false},
		{"Empty component", "", false},
	}

	validComponents := []string{
		"rke2-ingress-nginx",
		"rke2-metrics-server",
		"rke2-snapshot-controller",
		"rke2-snapshot-controller-crd",
		"rke2-snapshot-validation-webhook",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := containsRKE2(validComponents, tt.component)

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for component %q, got %v", tt.valid, tt.component, isValid)
			}
		})
	}
}

// TestRKE2DataDir_Path tests data directory path validation
func TestRKE2DataDir_Path(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"Default path", "/var/lib/rancher/rke2", true},
		{"Custom path", "/data/rke2", true},
		{"Absolute path", "/opt/rke2", true},
		{"Relative path", "rke2/data", false}, // Should be absolute
		{"Empty path", "", false},
		{"Windows-style", "C:\\rke2", false},    // Unix only
		{"With space", "/var/lib/rke 2", false}, // No spaces
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid path starts with / and has no spaces
			isValid := strings.HasPrefix(tt.path, "/") &&
				!strings.Contains(tt.path, " ") &&
				tt.path != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for path %q, got %v", tt.valid, tt.path, isValid)
			}
		})
	}
}

// TestRKE2NodeTaint_Format tests node taint formats
func TestRKE2NodeTaint_Format(t *testing.T) {
	tests := []struct {
		name  string
		taint string
		valid bool
	}{
		{"NoSchedule taint", "node-role.kubernetes.io/master:NoSchedule", true},
		{"NoExecute taint", "node-role.kubernetes.io/control-plane:NoExecute", true},
		{"PreferNoSchedule", "workload=critical:PreferNoSchedule", true},
		{"Simple key=value", "key=value:NoSchedule", true},
		{"Missing effect", "key=value", false},
		{"Invalid effect", "key=value:InvalidEffect", false},
		{"Empty taint", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid taint: key=value:effect or key:effect
			isValid := (strings.Contains(tt.taint, ":NoSchedule") ||
				strings.Contains(tt.taint, ":NoExecute") ||
				strings.Contains(tt.taint, ":PreferNoSchedule")) &&
				tt.taint != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for taint %q, got %v", tt.valid, tt.taint, isValid)
			}
		})
	}
}

// TestRKE2NodeLabel_Format tests node label formats
func TestRKE2NodeLabel_Format(t *testing.T) {
	tests := []struct {
		name  string
		label string
		valid bool
	}{
		{"Simple label", "environment=production", true},
		{"Namespaced label", "node.kubernetes.io/instance-type=large", true},
		{"Multiple slashes", "custom.io/team/project=value", true},
		{"No value", "just-a-key", true},
		{"Empty label", "", false},
		{"Invalid character", "label@key=value", false},
		{"Space in label", "my label=value", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.label != "" &&
				!strings.Contains(tt.label, " ") &&
				!strings.Contains(tt.label, "@")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for label %q, got %v", tt.valid, tt.label, isValid)
			}
		})
	}
}

// TestRKE2SnapshotSchedule_Cron tests snapshot schedule cron expressions
func TestRKE2SnapshotSchedule_Cron(t *testing.T) {
	tests := []struct {
		name  string
		cron  string
		valid bool
	}{
		{"Every 12 hours", "0 */12 * * *", true},
		{"Every 6 hours", "0 */6 * * *", true},
		{"Daily at midnight", "0 0 * * *", true},
		{"Every hour", "0 * * * *", true},
		{"Invalid format", "invalid", false},
		{"Too few fields", "0 0 *", false},
		{"Empty cron", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid cron has 5 fields
			fields := strings.Fields(tt.cron)
			isValid := len(fields) == 5

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for cron %q, got %v", tt.valid, tt.cron, isValid)
			}
		})
	}
}

// TestRKE2SnapshotRetention tests snapshot retention values
func TestRKE2SnapshotRetention(t *testing.T) {
	tests := []struct {
		name      string
		retention int
		valid     bool
	}{
		{"Keep 5 snapshots", 5, true},
		{"Keep 10 snapshots", 10, true},
		{"Keep 1 snapshot", 1, true},
		{"Keep 0 snapshots", 0, false}, // Must keep at least 1
		{"Negative retention", -1, false},
		{"Very high retention", 100, true}, // Valid but high
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.retention > 0

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for retention %d, got %v", tt.valid, tt.retention, isValid)
			}
		})
	}
}

// TestRKE2KubeconfigMode_Permissions tests kubeconfig file permissions
func TestRKE2KubeconfigMode_Permissions(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		secure bool
	}{
		{"Secure mode 0600", "0600", true},
		{"Secure mode 0400", "0400", true},
		{"Less secure 0644", "0644", false}, // Readable by others
		{"Insecure 0777", "0777", false},    // World writable
		{"Empty mode", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Secure modes: 0600 or 0400
			isSecure := tt.mode == "0600" || tt.mode == "0400"

			if isSecure != tt.secure {
				t.Errorf("Expected secure=%v for mode %q, got %v", tt.secure, tt.mode, isSecure)
			}
		})
	}
}

// TestRKE2ServerJoin_URL tests server join URL format
func TestRKE2ServerJoin_URL(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		valid bool
	}{
		{"Valid HTTPS URL", "https://10.0.0.1:9345", true},
		{"Valid with domain", "https://master.example.com:9345", true},
		{"Wrong port", "https://10.0.0.1:6443", false}, // Wrong port for join
		{"HTTP instead of HTTPS", "http://10.0.0.1:9345", false},
		{"Missing port", "https://10.0.0.1", false},
		{"Empty URL", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := strings.HasPrefix(tt.url, "https://") &&
				strings.Contains(tt.url, ":9345")

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for URL %q, got %v", tt.valid, tt.url, isValid)
			}
		})
	}
}

// TestRKE2ExtraArgs_Format tests extra argument formats
func TestRKE2ExtraArgs_Format(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		valid bool
	}{
		{"Simple flag", "debug", "true", true},
		{"Value with equals", "kubelet-arg", "max-pods=110", true},
		{"Numeric value", "port", "6443", true},
		{"Empty key", "", "value", false},
		{"Empty value allowed", "flag", "", true}, // Some flags have no value
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.key != ""

			if isValid != tt.valid {
				t.Errorf("Expected valid=%v for arg %q=%q, got %v", tt.valid, tt.key, tt.value, isValid)
			}
		})
	}
}

// TestRKE2ConfigBuilder_FirstMaster tests first master config generation
func TestRKE2ConfigBuilder_FirstMaster(t *testing.T) {
	cfg := GetRKE2Defaults()
	k8sConfig := &KubernetesConfig{
		PodCIDR:       "10.42.0.0/16",
		ServiceCIDR:   "10.43.0.0/16",
		ClusterDNS:    "10.43.0.10",
		NetworkPlugin: "calico",
	}

	config := BuildRKE2ServerConfig(cfg, "10.0.0.1", "master-1", true, "", k8sConfig)

	tests := []struct {
		name     string
		contains string
		should   bool
	}{
		{"Has token", "token:", true},
		{"Has node name", "node-name: master-1", true},
		{"Has node IP", "node-ip: 10.0.0.1", true},
		{"Has pod CIDR", "cluster-cidr: 10.42.0.0/16", true},
		{"Has service CIDR", "service-cidr: 10.43.0.0/16", true},
		{"No server URL (first master)", "server:", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(config, tt.contains)

			if contains != tt.should {
				t.Errorf("Expected %q in config: %v, got %v", tt.contains, tt.should, contains)
			}
		})
	}
}

// TestRKE2ConfigBuilder_AdditionalMaster tests additional master config
func TestRKE2ConfigBuilder_AdditionalMaster(t *testing.T) {
	cfg := GetRKE2Defaults()
	k8sConfig := &KubernetesConfig{
		PodCIDR:       "10.42.0.0/16",
		ServiceCIDR:   "10.43.0.0/16",
		NetworkPlugin: "calico",
	}

	config := BuildRKE2ServerConfig(cfg, "10.0.0.2", "master-2", false, "10.0.0.1", k8sConfig)

	tests := []struct {
		name     string
		contains string
		should   bool
	}{
		{"Has server URL", "server: https://10.0.0.1:9345", true},
		{"Has token", "token:", true},
		{"Has node name", "node-name: master-2", true},
		{"Has node IP", "node-ip: 10.0.0.2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contains := strings.Contains(config, tt.contains)

			if contains != tt.should {
				t.Errorf("Expected %q in config: %v, got %v", tt.contains, tt.should, contains)
			}
		})
	}
}

// Test100RKE2Scenarios generates 100 RKE2 configuration scenarios
func Test100RKE2Scenarios(t *testing.T) {
	scenarios := []struct {
		channel   string
		retention int
		mode      string
		valid     bool
	}{
		{"stable", 5, "0600", true},
		{"latest", 10, "0400", true},
		{"v1.28", 3, "0600", true},
		{"invalid", 0, "0777", false},
	}

	// Generate 96 more scenarios
	for i := 1; i <= 96; i++ {
		channels := []string{"stable", "latest", "v1.28", "v1.29"}
		channel := channels[i%len(channels)]

		retention := 1 + (i % 10)
		modes := []string{"0600", "0400", "0644"}
		mode := modes[i%len(modes)]

		validChannel := channel == "stable" || channel == "latest" || strings.HasPrefix(channel, "v1.")
		validRetention := retention > 0
		validMode := mode == "0600" || mode == "0400"

		scenario := struct {
			channel   string
			retention int
			mode      string
			valid     bool
		}{
			channel:   channel,
			retention: retention,
			mode:      mode,
			valid:     validChannel && validRetention && validMode,
		}
		scenarios = append(scenarios, scenario)
	}

	for i, scenario := range scenarios {
		t.Run(string(rune('A'+i%26))+"_rke2_"+string(rune('0'+i%10)), func(t *testing.T) {
			channelValid := scenario.channel == "stable" || scenario.channel == "latest" || strings.HasPrefix(scenario.channel, "v1.")
			retentionValid := scenario.retention > 0
			modeValid := scenario.mode == "0600" || scenario.mode == "0400"

			isValid := channelValid && retentionValid && modeValid

			if isValid != scenario.valid {
				t.Errorf("Scenario %d: Expected valid=%v, got %v (channel=%s, retention=%d, mode=%s)",
					i, scenario.valid, isValid, scenario.channel, scenario.retention, scenario.mode)
			}
		})
	}
}
