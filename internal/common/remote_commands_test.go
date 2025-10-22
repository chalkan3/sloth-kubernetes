package common

import (
	"strings"
	"testing"
)

func TestBuildAptInstallCommand(t *testing.T) {
	tests := []struct {
		name         string
		packages     []string
		wantContains []string
	}{
		{
			name:     "single package",
			packages: []string{"docker.io"},
			wantContains: []string{
				"apt-get update -y",
				"apt-get install -y docker.io",
				"Waiting for apt locks",
				"fuser /var/lib/dpkg/lock-frontend",
			},
		},
		{
			name:     "multiple packages",
			packages: []string{"docker.io", "wireguard", "curl"},
			wantContains: []string{
				"apt-get update -y",
				"apt-get install -y docker.io wireguard curl",
				"DEBIAN_FRONTEND=noninteractive",
			},
		},
		{
			name:     "package with version",
			packages: []string{"nginx=1.18.0"},
			wantContains: []string{
				"apt-get install -y nginx=1.18.0",
			},
		},
		{
			name:     "no packages",
			packages: []string{},
			wantContains: []string{
				"apt-get update -y",
				"apt-get install -y ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAptInstallCommand(tt.packages...)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("BuildAptInstallCommand() missing expected string: %q", want)
				}
			}

			// Verify it's a valid bash script
			if !strings.Contains(got, "while fuser") {
				t.Error("BuildAptInstallCommand() should include lock waiting logic")
			}

			if !strings.Contains(got, "sleep 5") {
				t.Error("BuildAptInstallCommand() should include sleep in lock wait loop")
			}
		})
	}
}

func TestBuildAptInstallCommand_Format(t *testing.T) {
	result := BuildAptInstallCommand("test-package")

	// Should start with newline (consistent formatting)
	if !strings.HasPrefix(result, "\n") {
		t.Error("BuildAptInstallCommand() should start with newline")
	}

	// Should contain comment
	if !strings.Contains(result, "# Wait for apt locks") {
		t.Error("BuildAptInstallCommand() should contain explanatory comment")
	}

	// Should handle unattended-upgrades
	if !strings.Contains(result, "/var/lib/apt/lists/lock") {
		t.Error("BuildAptInstallCommand() should check for apt lists lock")
	}
}

func TestBuildSystemdEnableStart(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		want        string
	}{
		{
			name:        "docker service",
			serviceName: "docker",
			want:        "systemctl enable docker && systemctl start docker",
		},
		{
			name:        "nginx service",
			serviceName: "nginx",
			want:        "systemctl enable nginx && systemctl start nginx",
		},
		{
			name:        "service with suffix",
			serviceName: "kubelet.service",
			want:        "systemctl enable kubelet.service && systemctl start kubelet.service",
		},
		{
			name:        "empty service name",
			serviceName: "",
			want:        "systemctl enable  && systemctl start ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSystemdEnableStart(tt.serviceName)
			if got != tt.want {
				t.Errorf("BuildSystemdEnableStart(%q) = %q, want %q", tt.serviceName, got, tt.want)
			}
		})
	}
}

func TestBuildSystemdEnableStart_Components(t *testing.T) {
	result := BuildSystemdEnableStart("test-service")

	// Should contain enable command
	if !strings.Contains(result, "systemctl enable test-service") {
		t.Error("BuildSystemdEnableStart() should contain enable command")
	}

	// Should contain start command
	if !strings.Contains(result, "systemctl start test-service") {
		t.Error("BuildSystemdEnableStart() should contain start command")
	}

	// Should use && for chaining
	if !strings.Contains(result, " && ") {
		t.Error("BuildSystemdEnableStart() should chain commands with &&")
	}
}

func TestBuildFileWrite(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		content      string
		wantContains []string
	}{
		{
			name:    "simple config file",
			path:    "/etc/config.conf",
			content: "key=value",
			wantContains: []string{
				"cat > /etc/config.conf <<'EOF'",
				"key=value",
				"EOF",
			},
		},
		{
			name:    "multi-line content",
			path:    "/tmp/test.txt",
			content: "line1\nline2\nline3",
			wantContains: []string{
				"cat > /tmp/test.txt",
				"line1\nline2\nline3",
				"EOF",
			},
		},
		{
			name:    "wireguard config",
			path:    "/etc/wireguard/wg0.conf",
			content: "[Interface]\nPrivateKey=xxx",
			wantContains: []string{
				"cat > /etc/wireguard/wg0.conf",
				"[Interface]",
				"PrivateKey=xxx",
			},
		},
		{
			name:    "empty content",
			path:    "/tmp/empty.txt",
			content: "",
			wantContains: []string{
				"cat > /tmp/empty.txt",
				"EOF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFileWrite(tt.path, tt.content)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("BuildFileWrite() missing expected string: %q", want)
				}
			}
		})
	}
}

func TestBuildFileWrite_HereDoc(t *testing.T) {
	result := BuildFileWrite("/test/path", "content")

	// Should use heredoc syntax
	if !strings.Contains(result, "<<'EOF'") {
		t.Error("BuildFileWrite() should use heredoc with single quotes")
	}

	// Should end with EOF
	if !strings.HasSuffix(result, "EOF") {
		t.Error("BuildFileWrite() should end with EOF")
	}

	// Heredoc should be on same line as cat command
	if !strings.Contains(result, "cat > /test/path <<'EOF'") {
		t.Error("BuildFileWrite() heredoc should be on same line as cat")
	}
}

func TestBuildFileWrite_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "dollar signs",
			content: "PATH=$HOME/bin",
		},
		{
			name:    "backticks",
			content: "date=`date`",
		},
		{
			name:    "quotes",
			content: `echo "hello" 'world'`,
		},
		{
			name:    "backslashes",
			content: "path=C:\\Windows\\System32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFileWrite("/tmp/test", tt.content)

			// Content should be preserved as-is
			if !strings.Contains(got, tt.content) {
				t.Errorf("BuildFileWrite() did not preserve content: %q", tt.content)
			}

			// Should use single-quoted heredoc to prevent variable expansion
			if !strings.Contains(got, "<<'EOF'") {
				t.Error("BuildFileWrite() should use single-quoted heredoc for safety")
			}
		})
	}
}

func TestBuildDirectoryCreate(t *testing.T) {
	tests := []struct {
		name string
		path string
		mode string
		want string
	}{
		{
			name: "standard directory",
			path: "/var/lib/data",
			mode: "755",
			want: "mkdir -p /var/lib/data && chmod 755 /var/lib/data",
		},
		{
			name: "secure directory",
			path: "/etc/secrets",
			mode: "700",
			want: "mkdir -p /etc/secrets && chmod 700 /etc/secrets",
		},
		{
			name: "world writable",
			path: "/tmp/shared",
			mode: "777",
			want: "mkdir -p /tmp/shared && chmod 777 /tmp/shared",
		},
		{
			name: "nested path",
			path: "/opt/app/config/data",
			mode: "750",
			want: "mkdir -p /opt/app/config/data && chmod 750 /opt/app/config/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDirectoryCreate(tt.path, tt.mode)
			if got != tt.want {
				t.Errorf("BuildDirectoryCreate(%q, %q) = %q, want %q", tt.path, tt.mode, got, tt.want)
			}
		})
	}
}

func TestBuildDirectoryCreate_Components(t *testing.T) {
	result := BuildDirectoryCreate("/test/path", "755")

	// Should use mkdir -p
	if !strings.Contains(result, "mkdir -p") {
		t.Error("BuildDirectoryCreate() should use 'mkdir -p' for recursive creation")
	}

	// Should include chmod
	if !strings.Contains(result, "chmod 755") {
		t.Error("BuildDirectoryCreate() should include chmod command")
	}

	// Should chain commands with &&
	if !strings.Contains(result, " && ") {
		t.Error("BuildDirectoryCreate() should chain mkdir and chmod with &&")
	}

	// Path should appear twice (mkdir and chmod)
	pathCount := strings.Count(result, "/test/path")
	if pathCount != 2 {
		t.Errorf("BuildDirectoryCreate() path should appear twice, got %d times", pathCount)
	}
}

func TestConstants(t *testing.T) {
	// Test that constants are defined and not empty
	constants := map[string]string{
		"InstallDocker":         InstallDocker,
		"InstallWireGuard":      InstallWireGuard,
		"GenerateWireGuardKeys": GenerateWireGuardKeys,
		"EnableIPForwarding":    EnableIPForwarding,
		"CheckDockerStatus":     CheckDockerStatus,
		"CheckWireGuardStatus":  CheckWireGuardStatus,
	}

	for name, value := range constants {
		if value == "" {
			t.Errorf("Constant %s should not be empty", name)
		}
	}
}

func TestInstallDocker_Content(t *testing.T) {
	// Verify InstallDocker constant contains required commands
	required := []string{
		"apt-get update",
		"apt-get install",
		"docker.io",
		"systemctl enable docker",
		"systemctl start docker",
	}

	for _, cmd := range required {
		if !strings.Contains(InstallDocker, cmd) {
			t.Errorf("InstallDocker should contain %q", cmd)
		}
	}
}

func TestInstallWireGuard_Content(t *testing.T) {
	// Verify InstallWireGuard constant
	required := []string{
		"apt-get update",
		"apt-get install",
		"wireguard",
		"wireguard-tools",
	}

	for _, cmd := range required {
		if !strings.Contains(InstallWireGuard, cmd) {
			t.Errorf("InstallWireGuard should contain %q", cmd)
		}
	}
}

func TestGenerateWireGuardKeys_Content(t *testing.T) {
	// Verify GenerateWireGuardKeys constant
	required := []string{
		"wg genkey",
		"wg pubkey",
		"/etc/wireguard/private.key",
		"/etc/wireguard/public.key",
		"chmod 600",
	}

	for _, cmd := range required {
		if !strings.Contains(GenerateWireGuardKeys, cmd) {
			t.Errorf("GenerateWireGuardKeys should contain %q", cmd)
		}
	}
}

func TestEnableIPForwarding_Content(t *testing.T) {
	// Verify EnableIPForwarding constant
	required := []string{
		"net.ipv4.ip_forward=1",
		"/etc/sysctl.conf",
		"sysctl -p",
	}

	for _, cmd := range required {
		if !strings.Contains(EnableIPForwarding, cmd) {
			t.Errorf("EnableIPForwarding should contain %q", cmd)
		}
	}
}
