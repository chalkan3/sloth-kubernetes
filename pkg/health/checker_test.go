package health

import (
	"errors"
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "DOCKER:PS:OK",
			substr: "DOCKER:PS:OK",
			want:   true,
		},
		{
			name:   "substring at beginning",
			s:      "SERVICE:docker:RUNNING is healthy",
			substr: "SERVICE:docker:RUNNING",
			want:   true,
		},
		{
			name:   "substring at end",
			s:      "The service is SERVICE:docker:RUNNING",
			substr: "SERVICE:docker:RUNNING",
			want:   true,
		},
		{
			name:   "substring in middle",
			s:      "Check SERVICE:docker:RUNNING complete",
			substr: "SERVICE:docker:RUNNING",
			want:   true,
		},
		{
			name:   "not found",
			s:      "DOCKER:VERSION:OK",
			substr: "WIREGUARD",
			want:   false,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "test",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "test string",
			substr: "",
			want:   false,
		},
		{
			name:   "both empty",
			s:      "",
			substr: "",
			want:   false,
		},
		{
			name:   "substring longer than string",
			s:      "OK",
			substr: "KUBERNETES:API:OK",
			want:   false,
		},
		{
			name:   "multiple occurrences",
			s:      "OK OK OK",
			substr: "OK",
			want:   true,
		},
		{
			name:   "case sensitive",
			s:      "service:docker:running",
			substr: "SERVICE:docker:RUNNING",
			want:   false,
		},
		{
			name:   "partial match at start",
			s:      "DOCKER",
			substr: "DOCKER:PS",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestContainsMiddle(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "found in middle",
			s:      "start middle end",
			substr: "middle",
			want:   true,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "foo",
			want:   false,
		},
		{
			name:   "substring at position 1",
			s:      "xmiddley",
			substr: "middle",
			want:   true,
		},
		{
			name:   "substring longer than string",
			s:      "hi",
			substr: "hello",
			want:   false,
		},
		{
			name:   "exact length match",
			s:      "test",
			substr: "test",
			want:   false, // containsMiddle doesn't check position 0
		},
		{
			name:   "single character in middle",
			s:      "abc",
			substr: "b",
			want:   true,
		},
		{
			name:   "near end",
			s:      "abcde",
			substr: "de",
			want:   false, // at the end, not middle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsMiddle(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsMiddle(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "simple error",
			err:  errors.New("connection timeout"),
			want: true,
		},
		{
			name: "ssh error",
			err:  errors.New("ssh: unable to authenticate"),
			want: true,
		},
		{
			name: "network error",
			err:  errors.New("network unreachable"),
			want: true,
		},
		{
			name: "generic error",
			err:  errors.New("something went wrong"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRecoverableError(tt.err)
			if got != tt.want {
				t.Errorf("isRecoverableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsRecoverableError_AlwaysRecoverable(t *testing.T) {
	// Based on the implementation, all non-nil errors are recoverable
	testErrors := []error{
		errors.New("timeout"),
		errors.New("connection refused"),
		errors.New("host unreachable"),
		errors.New("permission denied"),
		errors.New("service unavailable"),
	}

	for _, err := range testErrors {
		if !isRecoverableError(err) {
			t.Errorf("isRecoverableError(%v) should be true for all errors", err)
		}
	}

	// Nil error should not be recoverable
	if isRecoverableError(nil) {
		t.Error("isRecoverableError(nil) should be false")
	}
}

func TestContains_ServicePatterns(t *testing.T) {
	// Test with realistic service check outputs
	serviceOutput := `=== Node Health Check ===
Timestamp: Thu Jan 15 10:30:00 UTC 2025

SERVICE:docker:RUNNING
COMMAND:docker:AVAILABLE
DOCKER:VERSION:OK
DOCKER:PS:OK

WIREGUARD:CONFIG:EXISTS
WIREGUARD:INTERFACE:UP

KUBECONFIG:EXISTS
KUBERNETES:API:OK

=== Health Check Complete ===`

	tests := []struct {
		name   string
		substr string
		want   bool
	}{
		{"docker running", "SERVICE:docker:RUNNING", true},
		{"docker available", "COMMAND:docker:AVAILABLE", true},
		{"docker version ok", "DOCKER:VERSION:OK", true},
		{"docker ps ok", "DOCKER:PS:OK", true},
		{"wireguard config", "WIREGUARD:CONFIG:EXISTS", true},
		{"wireguard up", "WIREGUARD:INTERFACE:UP", true},
		{"kubeconfig", "KUBECONFIG:EXISTS", true},
		{"k8s api", "KUBERNETES:API:OK", true},
		{"not present", "NGINX:SERVICE:OK", false},
		{"partial match", "DOCKER:VERSION", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(serviceOutput, tt.substr)
			if got != tt.want {
				t.Errorf("contains(output, %q) = %v, want %v", tt.substr, got, tt.want)
			}
		})
	}
}

func TestContains_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "newline in string",
			s:      "line1\nline2\nline3",
			substr: "line2",
			want:   true,
		},
		{
			name:   "tab in string",
			s:      "col1\tcol2\tcol3",
			substr: "col2",
			want:   true,
		},
		{
			name:   "special characters",
			s:      "key:value|status=ok",
			substr: "status=ok",
			want:   true,
		},
		{
			name:   "unicode",
			s:      "test ✓ passed",
			substr: "✓",
			want:   true,
		},
		{
			name:   "repeated pattern",
			s:      "ababab",
			substr: "ab",
			want:   true,
		},
		{
			name:   "overlapping search",
			s:      "aaa",
			substr: "aa",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
