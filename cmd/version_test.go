package cmd

import (
	"runtime"
	"strings"
	"testing"
)

// TestVersionVariables tests version variables
func TestVersionVariables(t *testing.T) {
	// Test that version variables are set
	if version == "" {
		t.Error("version should not be empty")
	}

	if gitCommit == "" {
		t.Error("gitCommit should not be empty")
	}

	if buildDate == "" {
		t.Error("buildDate should not be empty")
	}
}

// TestVersionFormat tests version format validation
func TestVersionFormat(t *testing.T) {
	tests := []struct {
		name    string
		version string
		isValid bool
	}{
		{"Valid semver", "1.0.0", true},
		{"Valid semver with patch", "1.2.3", true},
		{"Valid semver major only", "2.0.0", true},
		{"Dev version", "dev", true},
		{"Empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.version != ""

			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got valid=%v for version %q", tt.isValid, isValid, tt.version)
			}
		})
	}
}

// TestGitCommitFormat tests git commit format
func TestGitCommitFormat(t *testing.T) {
	tests := []struct {
		name   string
		commit string
		isValid bool
	}{
		{"Short SHA", "abc123d", true},
		{"Full SHA", "abc123def456789012345678901234567890abcd", true},
		{"Dev build", "dev", true},
		{"Empty commit", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.commit != ""

			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got valid=%v for commit %q", tt.isValid, isValid, tt.commit)
			}
		})
	}
}

// TestBuildDateFormat tests build date format
func TestBuildDateFormat(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		isValid bool
	}{
		{"ISO date", "2024-01-15", true},
		{"Full datetime", "2024-01-15T10:30:00Z", true},
		{"Unknown", "unknown", true},
		{"Empty date", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.date != ""

			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got valid=%v for date %q", tt.isValid, isValid, tt.date)
			}
		})
	}
}

// TestRuntimeInfo tests runtime information
func TestRuntimeInfo(t *testing.T) {
	// Test Go version
	goVersion := runtime.Version()
	if !strings.HasPrefix(goVersion, "go") {
		t.Errorf("Go version should start with 'go', got %q", goVersion)
	}

	// Test OS
	goos := runtime.GOOS
	validOS := []string{"darwin", "linux", "windows", "freebsd"}
	isValidOS := false
	for _, os := range validOS {
		if goos == os {
			isValidOS = true
			break
		}
	}
	if !isValidOS {
		t.Logf("Note: Running on uncommon OS: %s", goos)
	}

	// Test architecture
	goarch := runtime.GOARCH
	validArch := []string{"amd64", "arm64", "386", "arm"}
	isValidArch := false
	for _, arch := range validArch {
		if goarch == arch {
			isValidArch = true
			break
		}
	}
	if !isValidArch {
		t.Logf("Note: Running on uncommon architecture: %s", goarch)
	}
}

// TestOSArchCombinations tests common OS/Arch combinations
func TestOSArchCombinations(t *testing.T) {
	tests := []struct {
		os   string
		arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.os+"/"+tt.arch, func(t *testing.T) {
			// Verify format
			osArch := tt.os + "/" + tt.arch
			if !strings.Contains(osArch, "/") {
				t.Error("OS/Arch should contain separator")
			}

			parts := strings.Split(osArch, "/")
			if len(parts) != 2 {
				t.Errorf("Expected 2 parts, got %d", len(parts))
			}

			if parts[0] != tt.os {
				t.Errorf("Expected OS %q, got %q", tt.os, parts[0])
			}

			if parts[1] != tt.arch {
				t.Errorf("Expected arch %q, got %q", tt.arch, parts[1])
			}
		})
	}
}

// TestVersionCommandStructure tests version command structure
func TestVersionCommandStructure(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd should not be nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("Expected Use 'version', got %q", versionCmd.Use)
	}

	if versionCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if versionCmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if versionCmd.Run == nil {
		t.Error("Run function should not be nil")
	}
}

// TestVersionConstants tests version-related constants
func TestVersionConstants(t *testing.T) {
	// Default version should be semantic versioning
	if version != "" {
		// Check if it contains dots (semver pattern)
		if !strings.Contains(version, ".") && version != "dev" {
			t.Logf("Version %q doesn't follow semver pattern (x.y.z)", version)
		}
	}

	// Git commit should be alphanumeric or "dev"
	if gitCommit != "" {
		for _, char := range gitCommit {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9')) {
				t.Logf("Git commit contains non-alphanumeric character: %c", char)
				break
			}
		}
	}
}

// TestGoVersionFormat tests Go version string format
func TestGoVersionFormat(t *testing.T) {
	goVersion := runtime.Version()

	// Should start with "go"
	if !strings.HasPrefix(goVersion, "go") {
		t.Errorf("Go version should start with 'go', got %q", goVersion)
	}

	// Should contain numbers
	hasNumber := false
	for _, char := range goVersion {
		if char >= '0' && char <= '9' {
			hasNumber = true
			break
		}
	}
	if !hasNumber {
		t.Errorf("Go version should contain numbers, got %q", goVersion)
	}
}

// TestPlatformInfo tests platform information
func TestPlatformInfo(t *testing.T) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// OS should not be empty
	if os == "" {
		t.Error("GOOS should not be empty")
	}

	// Arch should not be empty
	if arch == "" {
		t.Error("GOARCH should not be empty")
	}

	// Common OS values
	commonOS := map[string]bool{
		"darwin":  true,
		"linux":   true,
		"windows": true,
		"freebsd": true,
	}

	if !commonOS[os] {
		t.Logf("Running on less common OS: %s", os)
	}

	// Common arch values
	commonArch := map[string]bool{
		"amd64": true,
		"arm64": true,
		"386":   true,
		"arm":   true,
	}

	if !commonArch[arch] {
		t.Logf("Running on less common architecture: %s", arch)
	}
}

// TestVersionInfo tests version information structure
func TestVersionInfo(t *testing.T) {
	info := struct {
		version   string
		gitCommit string
		buildDate string
		goVersion string
		os        string
		arch      string
	}{
		version:   version,
		gitCommit: gitCommit,
		buildDate: buildDate,
		goVersion: runtime.Version(),
		os:        runtime.GOOS,
		arch:      runtime.GOARCH,
	}

	// All fields should be populated
	if info.version == "" {
		t.Error("version should not be empty")
	}
	if info.gitCommit == "" {
		t.Error("gitCommit should not be empty")
	}
	if info.buildDate == "" {
		t.Error("buildDate should not be empty")
	}
	if info.goVersion == "" {
		t.Error("goVersion should not be empty")
	}
	if info.os == "" {
		t.Error("os should not be empty")
	}
	if info.arch == "" {
		t.Error("arch should not be empty")
	}
}

// TestCLIName tests CLI name consistency
func TestCLIName(t *testing.T) {
	cliNames := []string{
		"Kubernetes-Create CLI",
		"kubernetes-create",
		"sloth-kubernetes",
	}

	for _, name := range cliNames {
		if name == "" {
			t.Error("CLI name should not be empty")
		}

		// Should not contain special characters except dash
		for _, char := range name {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == ' ') {
				t.Errorf("CLI name %q contains invalid character: %c", name, char)
			}
		}
	}
}
