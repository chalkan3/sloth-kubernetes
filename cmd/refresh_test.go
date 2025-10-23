package cmd

import (
	"testing"
)

func TestRefreshCommand(t *testing.T) {
	// Test that refresh command is registered
	cmd := refreshCmd
	if cmd == nil {
		t.Fatal("refresh command should not be nil")
	}

	if cmd.Use != "refresh <stack-name>" {
		t.Errorf("Expected Use to be 'refresh <stack-name>', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE function should not be nil")
	}
}

func TestRefreshFlags(t *testing.T) {
	cmd := refreshCmd

	// Test expect-no-changes flag
	flag := cmd.Flags().Lookup("expect-no-changes")
	if flag == nil {
		t.Error("expect-no-changes flag should exist")
	} else {
		if flag.Usage != "Return error if any changes are detected" {
			t.Errorf("Unexpected usage for expect-no-changes: %s", flag.Usage)
		}
		if flag.DefValue != "false" {
			t.Errorf("Expected default value 'false', got '%s'", flag.DefValue)
		}
	}

	// Test show-secrets flag
	flag = cmd.Flags().Lookup("show-secrets")
	if flag == nil {
		t.Error("show-secrets flag should exist")
	} else {
		if flag.Usage != "Show secret values in output" {
			t.Errorf("Unexpected usage for show-secrets: %s", flag.Usage)
		}
		if flag.DefValue != "false" {
			t.Errorf("Expected default value 'false', got '%s'", flag.DefValue)
		}
	}

	// Test skip-preview flag
	flag = cmd.Flags().Lookup("skip-preview")
	if flag == nil {
		t.Error("skip-preview flag should exist")
	} else {
		if flag.Usage != "Skip preview and refresh directly" {
			t.Errorf("Unexpected usage for skip-preview: %s", flag.Usage)
		}
		if flag.DefValue != "false" {
			t.Errorf("Expected default value 'false', got '%s'", flag.DefValue)
		}
	}
}

func TestRefreshCommand_MaxArgs(t *testing.T) {
	cmd := refreshCmd

	// Test with valid number of args (0 or 1)
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "No arguments",
			args: []string{},
			want: true, // Valid - will use default stack
		},
		{
			name: "One argument",
			args: []string{"production"},
			want: true, // Valid
		},
		{
			name: "Two arguments",
			args: []string{"production", "extra"},
			want: false, // Invalid - too many args
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cmd.Args(cmd, tt.args)
			if (err == nil) != tt.want {
				t.Errorf("Args validation failed for %s: got error=%v, want valid=%v", tt.name, err, tt.want)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "Contains at start",
			s:      "node_id_123",
			substr: "node",
			want:   true,
		},
		{
			name:   "Contains at end",
			s:      "master_node",
			substr: "node",
			want:   true,
		},
		{
			name:   "Contains in middle",
			s:      "my_node_id",
			substr: "node",
			want:   true,
		},
		{
			name:   "Does not contain",
			s:      "worker_123",
			substr: "node",
			want:   false,
		},
		{
			name:   "Exact match",
			s:      "node",
			substr: "node",
			want:   true,
		},
		{
			name:   "Empty substring",
			s:      "anything",
			substr: "",
			want:   true,
		},
		{
			name:   "Substring longer than string",
			s:      "abc",
			substr: "abcdef",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsString(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{
			name:   "Found at start",
			s:      "node_id_123",
			substr: "node",
			want:   0,
		},
		{
			name:   "Found in middle",
			s:      "my_node_id",
			substr: "node",
			want:   3,
		},
		{
			name:   "Found at end",
			s:      "master_node",
			substr: "node",
			want:   7,
		},
		{
			name:   "Not found",
			s:      "worker_123",
			substr: "node",
			want:   -1,
		},
		{
			name:   "Exact match",
			s:      "node",
			substr: "node",
			want:   0,
		},
		{
			name:   "Empty substring",
			s:      "anything",
			substr: "",
			want:   0,
		},
		{
			name:   "Substring longer",
			s:      "abc",
			substr: "abcdef",
			want:   -1,
		},
		{
			name:   "Multiple occurrences returns first",
			s:      "node_node_node",
			substr: "node",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexOf(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("indexOf(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestRefreshCommand_Integration(t *testing.T) {
	// This is an integration test that would require actual Pulumi stack
	// Skip in unit tests
	t.Skip("Integration test - requires actual Pulumi stack")

	// Example of how to test with actual stack:
	// - Set up test stack
	// - Run refresh command
	// - Verify output
	// - Clean up
}
