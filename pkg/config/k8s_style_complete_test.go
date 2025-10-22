package config

import (
	"os"
	"testing"
)

func TestExpandEnvVars_BothFormats(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("ANOTHER_VAR", "another_value")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("ANOTHER_VAR")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Braces format ${VAR}",
			input: "Value is ${TEST_VAR}",
			want:  "Value is test_value",
		},
		{
			name:  "Simple format $VAR",
			input: "Value is $TEST_VAR",
			want:  "Value is test_value",
		},
		{
			name:  "Multiple variables braces",
			input: "${TEST_VAR} and ${ANOTHER_VAR}",
			want:  "test_value and another_value",
		},
		{
			name:  "Multiple variables simple",
			input: "$TEST_VAR and $ANOTHER_VAR",
			want:  "test_value and another_value",
		},
		{
			name:  "Mixed formats",
			input: "${TEST_VAR} and $ANOTHER_VAR",
			want:  "test_value and another_value",
		},
		{
			name:  "Variable not set with braces",
			input: "${NOTSET}",
			want:  "${NOTSET}",
		},
		{
			name:  "Variable not set simple",
			input: "$NOTSET",
			want:  "$NOTSET",
		},
		{
			name:  "Variable in middle",
			input: "prefix_${TEST_VAR}_suffix",
			want:  "prefix_test_value_suffix",
		},
		{
			name:  "Multiple same variable",
			input: "${TEST_VAR} ${TEST_VAR} ${TEST_VAR}",
			want:  "test_value test_value test_value",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "No variables",
			input: "just a string",
			want:  "just a string",
		},
		{
			name:  "Dollar without variable",
			input: "cost is $100",
			want:  "cost is $100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvVars_EmptyValue(t *testing.T) {
	os.Setenv("EMPTY_VAR", "")
	defer os.Unsetenv("EMPTY_VAR")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Empty value with braces",
			input: "${EMPTY_VAR}",
			want:  "${EMPTY_VAR}", // Should keep original if empty
		},
		{
			name:  "Empty value simple",
			input: "$EMPTY_VAR",
			want:  "$EMPTY_VAR", // Should keep original if empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvVars_SpecialCharsInValue(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		envVal string
		input  string
		want   string
	}{
		{
			name:   "Value with spaces",
			envVar: "SPACE_VAR",
			envVal: "hello world",
			input:  "${SPACE_VAR}",
			want:   "hello world",
		},
		{
			name:   "Value with slashes",
			envVar: "PATH_VAR",
			envVal: "/usr/local/bin",
			input:  "${PATH_VAR}",
			want:   "/usr/local/bin",
		},
		{
			name:   "Value with dots",
			envVar: "DOMAIN_VAR",
			envVal: "example.com",
			input:  "${DOMAIN_VAR}",
			want:   "example.com",
		},
		{
			name:   "Value with dashes",
			envVar: "UUID_VAR",
			envVal: "123-456-789",
			input:  "${UUID_VAR}",
			want:   "123-456-789",
		},
		{
			name:   "Value with equals",
			envVar: "EQUATION_VAR",
			envVal: "x=10",
			input:  "${EQUATION_VAR}",
			want:   "x=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envVal)
			defer os.Unsetenv(tt.envVar)

			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvVars_VariableNaming(t *testing.T) {
	// Test different variable name formats
	tests := []struct {
		name   string
		envVar string
		envVal string
		input  string
		want   string
	}{
		{
			name:   "Lowercase",
			envVar: "lowercase",
			envVal: "value",
			input:  "${lowercase}",
			want:   "value",
		},
		{
			name:   "Uppercase",
			envVar: "UPPERCASE",
			envVal: "value",
			input:  "${UPPERCASE}",
			want:   "value",
		},
		{
			name:   "Mixed case",
			envVar: "MixedCase",
			envVal: "value",
			input:  "${MixedCase}",
			want:   "value",
		},
		{
			name:   "With underscores",
			envVar: "VAR_WITH_UNDERSCORES",
			envVal: "value",
			input:  "${VAR_WITH_UNDERSCORES}",
			want:   "value",
		},
		{
			name:   "With numbers",
			envVar: "VAR123",
			envVal: "value",
			input:  "${VAR123}",
			want:   "value",
		},
		{
			name:   "Numbers and underscores",
			envVar: "VAR_123_TEST",
			envVal: "value",
			input:  "${VAR_123_TEST}",
			want:   "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envVal)
			defer os.Unsetenv(tt.envVar)

			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvVars_AdvancedEdgeCases(t *testing.T) {
	os.Setenv("TEST", "value")
	defer os.Unsetenv("TEST")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Empty braces",
			input: "${}",
			want:  "${}",
		},
		{
			name:  "Just dollar",
			input: "$",
			want:  "$",
		},
		{
			name:  "Unclosed brace",
			input: "${TEST",
			want:  "${TEST",
		},
		{
			name:  "Only opening brace",
			input: "${",
			want:  "${",
		},
		{
			name:  "Nested braces",
			input: "${${TEST}}",
			want:  "${value}", // Inner ${TEST} gets expanded first
		},
		{
			name:  "Variable at start",
			input: "${TEST} rest",
			want:  "value rest",
		},
		{
			name:  "Variable at end",
			input: "start ${TEST}",
			want:  "start value",
		},
		{
			name:  "Only variable",
			input: "${TEST}",
			want:  "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvVars_BracesVsSimple(t *testing.T) {
	os.Setenv("VAR", "value")
	defer os.Unsetenv("VAR")

	// When both formats appear, ${VAR} gets expanded but $VAR is skipped if ${VAR} exists in original
	input := "${VAR} and $VAR"
	result := expandEnvVars(input)

	// Based on the implementation, $VAR is not expanded if ${VAR} exists in the string
	if result != "value and $VAR" {
		t.Errorf("expandEnvVars(%q) = %q, want 'value and $VAR'", input, result)
	}
}

func TestExpandEnvVars_OnlyExpandsIfNotEmpty(t *testing.T) {
	// Variable exists but is empty
	os.Setenv("EMPTY", "")
	// Variable doesn't exist
	os.Unsetenv("NOTEXIST")
	defer os.Unsetenv("EMPTY")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Empty variable braces",
			input: "${EMPTY}",
			want:  "${EMPTY}",
		},
		{
			name:  "Nonexistent variable braces",
			input: "${NOTEXIST}",
			want:  "${NOTEXIST}",
		},
		{
			name:  "Empty variable simple",
			input: "$EMPTY",
			want:  "$EMPTY",
		},
		{
			name:  "Nonexistent variable simple",
			input: "$NOTEXIST",
			want:  "$NOTEXIST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvVars_ComplexStrings(t *testing.T) {
	os.Setenv("HOST", "example.com")
	os.Setenv("PORT", "8080")
	os.Setenv("PROTOCOL", "https")
	defer os.Unsetenv("HOST")
	defer os.Unsetenv("PORT")
	defer os.Unsetenv("PROTOCOL")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "URL construction",
			input: "${PROTOCOL}://${HOST}:${PORT}/path",
			want:  "https://example.com:8080/path",
		},
		{
			name:  "Mixed text and variables",
			input: "Connect to ${HOST} on port ${PORT} using ${PROTOCOL}",
			want:  "Connect to example.com on port 8080 using https",
		},
		{
			name:  "Multiple occurrences",
			input: "${HOST} ${HOST} ${HOST}",
			want:  "example.com example.com example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
