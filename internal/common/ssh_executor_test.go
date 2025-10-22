package common

import (
	"strings"
	"testing"
)

func TestBuildRetryCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		retries     int
		wantContains []string
	}{
		{
			name:    "No retries (1)",
			command: "echo hello",
			retries: 1,
			wantContains: []string{
				"echo hello",
			},
		},
		{
			name:    "Two retries",
			command: "apt-get update",
			retries: 2,
			wantContains: []string{
				"for i in",
				"apt-get update",
				"&& break",
				"sleep 10",
			},
		},
		{
			name:    "Three retries",
			command: "docker pull nginx",
			retries: 3,
			wantContains: []string{
				"for i in",
				"docker pull nginx",
				"&& break",
				"|| sleep 10",
				"done",
			},
		},
		{
			name:    "Five retries",
			command: "wget https://example.com/file",
			retries: 5,
			wantContains: []string{
				"for i in",
				"wget https://example.com/file",
				"&& break",
				"|| sleep 10",
				"done",
			},
		},
		{
			name:    "Zero retries",
			command: "ls -la",
			retries: 0,
			wantContains: []string{
				"ls -la",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRetryCommand(tt.command, tt.retries)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("BuildRetryCommand() missing expected string: %q\nGot: %q", want, got)
				}
			}

			// If retries <= 1, should return command as-is
			if tt.retries <= 1 {
				if got != tt.command {
					t.Errorf("BuildRetryCommand() with retries <= 1 should return command as-is\nGot: %q\nWant: %q", got, tt.command)
				}
			} else {
				// Should contain retry logic
				if !strings.Contains(got, "for i in") {
					t.Error("BuildRetryCommand() with retries > 1 should contain 'for i in'")
				}
				if !strings.Contains(got, "&& break") {
					t.Error("BuildRetryCommand() should contain '&& break' for retry logic")
				}
				if !strings.Contains(got, "|| sleep 10") {
					t.Error("BuildRetryCommand() should contain '|| sleep 10' for retry delay")
				}
			}
		})
	}
}

func TestBuildRetryCommand_ExactFormat(t *testing.T) {
	// Test exact format for 3 retries
	command := "test-command"
	retries := 3
	result := BuildRetryCommand(command, retries)

	// Should follow the pattern: for i in {1..N}; do COMMAND && break || sleep 10; done
	expectedParts := []string{
		"for i in {1..",
		"}; do ",
		"test-command",
		" && break || sleep 10; done",
	}

	for _, part := range expectedParts {
		if !strings.Contains(result, part) {
			t.Errorf("BuildRetryCommand() missing part: %q in result: %q", part, result)
		}
	}
}

func TestBuildRetryCommand_SingleRetry(t *testing.T) {
	command := "echo test"
	result := BuildRetryCommand(command, 1)

	if result != command {
		t.Errorf("BuildRetryCommand() with 1 retry should return original command\nGot: %q\nWant: %q", result, command)
	}

	// Should NOT contain retry logic
	if strings.Contains(result, "for i in") {
		t.Error("BuildRetryCommand() with 1 retry should not contain retry loop")
	}
}

func TestBuildRetryCommand_ComplexCommand(t *testing.T) {
	command := "apt-get update && apt-get install -y docker.io"
	retries := 3
	result := BuildRetryCommand(command, retries)

	// Should wrap the entire complex command
	if !strings.Contains(result, command) {
		t.Errorf("BuildRetryCommand() should contain original command:\n%q\nGot:\n%q", command, result)
	}

	if !strings.Contains(result, "for i in") {
		t.Error("BuildRetryCommand() should contain retry loop")
	}
}

func TestBuildRetryCommand_CommandWithSpecialChars(t *testing.T) {
	tests := []struct {
		name    string
		command string
		retries int
	}{
		{
			name:    "Command with pipes",
			command: "cat file.txt | grep pattern",
			retries: 2,
		},
		{
			name:    "Command with redirection",
			command: "echo 'test' > /tmp/file",
			retries: 2,
		},
		{
			name:    "Command with quotes",
			command: "echo \"hello world\"",
			retries: 2,
		},
		{
			name:    "Command with backslashes",
			command: "find /path -name \\*.txt",
			retries: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildRetryCommand(tt.command, tt.retries)

			// Original command should be preserved
			if !strings.Contains(result, tt.command) {
				t.Errorf("BuildRetryCommand() should preserve command:\nWant: %q\nGot: %q", tt.command, result)
			}
		})
	}
}

func TestBuildRetryCommand_NegativeRetries(t *testing.T) {
	command := "test command"
	result := BuildRetryCommand(command, -1)

	// Negative retries should be treated like <= 1
	if result != command {
		t.Errorf("BuildRetryCommand() with negative retries should return original command\nGot: %q\nWant: %q", result, command)
	}
}

func TestBuildRetryCommand_LargeRetries(t *testing.T) {
	command := "download-file"
	retries := 10
	result := BuildRetryCommand(command, retries)

	// Should contain retry logic
	if !strings.Contains(result, "for i in") {
		t.Error("BuildRetryCommand() with large retries should contain retry loop")
	}

	// Original command should be present
	if !strings.Contains(result, command) {
		t.Error("BuildRetryCommand() should contain original command")
	}
}

func TestBuildRetryCommand_EmptyCommand(t *testing.T) {
	command := ""
	retries := 3
	result := BuildRetryCommand(command, retries)

	// Even empty command should be wrapped if retries > 1
	if !strings.Contains(result, "for i in") {
		t.Error("BuildRetryCommand() should wrap even empty command")
	}
}

func TestBuildRetryCommand_WhitespaceCommand(t *testing.T) {
	command := "   "
	retries := 2
	result := BuildRetryCommand(command, retries)

	// Should preserve whitespace command
	if !strings.Contains(result, command) {
		t.Error("BuildRetryCommand() should preserve whitespace in command")
	}
}

func TestBuildRetryCommand_MultilineCommand(t *testing.T) {
	command := "echo 'line1'\necho 'line2'"
	retries := 2
	result := BuildRetryCommand(command, retries)

	// Should contain the multiline command
	if !strings.Contains(result, command) {
		t.Errorf("BuildRetryCommand() should handle multiline commands:\nWant: %q\nGot: %q", command, result)
	}
}

func TestBuildRetryCommand_RetryPattern(t *testing.T) {
	// Test that retry pattern follows expected format
	command := "test"
	retries := 3

	result := BuildRetryCommand(command, retries)

	// Check order of elements
	forIndex := strings.Index(result, "for i in")
	doIndex := strings.Index(result, "do ")
	breakIndex := strings.Index(result, "&& break")
	sleepIndex := strings.Index(result, "|| sleep")
	doneIndex := strings.Index(result, "done")

	if forIndex == -1 {
		t.Error("Missing 'for i in'")
	}
	if doIndex == -1 {
		t.Error("Missing 'do '")
	}
	if breakIndex == -1 {
		t.Error("Missing '&& break'")
	}
	if sleepIndex == -1 {
		t.Error("Missing '|| sleep'")
	}
	if doneIndex == -1 {
		t.Error("Missing 'done'")
	}

	// Check order is correct
	if !(forIndex < doIndex && doIndex < breakIndex && breakIndex < sleepIndex && sleepIndex < doneIndex) {
		t.Error("Retry pattern elements are not in correct order")
	}
}

func TestBuildRetryCommand_SleepDuration(t *testing.T) {
	command := "test"
	retries := 5
	result := BuildRetryCommand(command, retries)

	// Should always use "sleep 10" for retry delay
	if !strings.Contains(result, "sleep 10") {
		t.Error("BuildRetryCommand() should use 'sleep 10' for retry delay")
	}

	// Should NOT contain other sleep durations
	otherSleeps := []string{"sleep 5", "sleep 20", "sleep 30"}
	for _, sleep := range otherSleeps {
		if strings.Contains(result, sleep) {
			t.Errorf("BuildRetryCommand() should not contain %q", sleep)
		}
	}
}

func TestBuildRetryCommand_BreakOnSuccess(t *testing.T) {
	command := "test"
	retries := 3
	result := BuildRetryCommand(command, retries)

	// Should break on successful command execution
	if !strings.Contains(result, "&& break") {
		t.Error("BuildRetryCommand() should contain '&& break' to exit on success")
	}

	// The pattern should be: command && break || sleep
	if !strings.Contains(result, "test && break || sleep") {
		t.Error("BuildRetryCommand() should follow pattern: 'command && break || sleep'")
	}
}
