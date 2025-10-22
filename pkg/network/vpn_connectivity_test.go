package network

import (
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
			s:      "hello",
			substr: "hello",
			want:   true,
		},
		{
			name:   "substring at start",
			s:      "hello world",
			substr: "hello",
			want:   true,
		},
		{
			name:   "substring at end",
			s:      "hello world",
			substr: "world",
			want:   true,
		},
		{
			name:   "substring in middle",
			s:      "hello world",
			substr: "lo wo",
			want:   true,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "foo",
			want:   false,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "hello",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "hello",
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
			s:      "hi",
			substr: "hello",
			want:   false,
		},
		{
			name:   "case sensitive - no match",
			s:      "Hello World",
			substr: "hello",
			want:   false,
		},
		{
			name:   "case sensitive - match",
			s:      "Hello World",
			substr: "Hello",
			want:   true,
		},
		{
			name:   "single character match",
			s:      "hello",
			substr: "e",
			want:   true,
		},
		{
			name:   "single character no match",
			s:      "hello",
			substr: "x",
			want:   false,
		},
		{
			name:   "repeated substring",
			s:      "hello hello",
			substr: "hello",
			want:   true,
		},
		{
			name:   "partial match not enough",
			s:      "helloworld",
			substr: "world!",
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

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "found in middle",
			s:      "hello world",
			substr: "lo wo",
			want:   true,
		},
		{
			name:   "not found",
			s:      "hello world",
			substr: "xyz",
			want:   false,
		},
		{
			name:   "substring longer than string",
			s:      "hi",
			substr: "hello",
			want:   false,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "test",
			want:   false,
		},
		{
			name:   "single char found",
			s:      "hello",
			substr: "l",
			want:   true,
		},
		{
			name:   "exact match",
			s:      "test",
			substr: "test",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsSubstring(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestExtractValue(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		prefix string
		want   string
	}{
		{
			name:   "extract simple value",
			s:      "PACKET_LOSS:10",
			prefix: "PACKET_LOSS:",
			want:   "10",
		},
		{
			name:   "extract until newline",
			s:      "PACKET_LOSS:10\nOTHER:20",
			prefix: "PACKET_LOSS:",
			want:   "10",
		},
		{
			name:   "extract until space",
			s:      "PACKET_LOSS:10 OTHER",
			prefix: "PACKET_LOSS:",
			want:   "10",
		},
		{
			name:   "prefix not found",
			s:      "SOME_OTHER_VALUE:10",
			prefix: "PACKET_LOSS:",
			want:   "",
		},
		{
			name:   "extract floating point",
			s:      "AVG_LATENCY:15.5",
			prefix: "AVG_LATENCY:",
			want:   "15.5",
		},
		{
			name:   "extract with multiple occurrences",
			s:      "FOO:bar FOO:baz",
			prefix: "FOO:",
			want:   "bar",
		},
		{
			name:   "extract empty value",
			s:      "VALUE: next",
			prefix: "VALUE:",
			want:   "",
		},
		{
			name:   "extract value at end",
			s:      "RESULT:success",
			prefix: "RESULT:",
			want:   "success",
		},
		{
			name:   "empty string",
			s:      "",
			prefix: "KEY:",
			want:   "",
		},
		{
			name:   "prefix at start",
			s:      "KEY:value",
			prefix: "KEY:",
			want:   "value",
		},
		{
			name:   "extract alphanumeric",
			s:      "STATUS:ok123",
			prefix: "STATUS:",
			want:   "ok123",
		},
		{
			name:   "extract until special char",
			s:      "DATA:test|other",
			prefix: "DATA:",
			want:   "test|other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractValue(tt.s, tt.prefix)
			if got != tt.want {
				t.Errorf("extractValue(%q, %q) = %q, want %q", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestExtractValue_MultilineInput(t *testing.T) {
	input := `PING_STATUS:SUCCESS
PACKET_LOSS:5
AVG_LATENCY:12.3
HANDSHAKE:ACTIVE`

	tests := []struct {
		prefix string
		want   string
	}{
		{"PING_STATUS:", "SUCCESS"},
		{"PACKET_LOSS:", "5"},
		{"AVG_LATENCY:", "12.3"},
		{"HANDSHAKE:", "ACTIVE"},
		{"NOTFOUND:", ""},
	}

	for _, tt := range tests {
		got := extractValue(input, tt.prefix)
		if got != tt.want {
			t.Errorf("extractValue(multiline, %q) = %q, want %q", tt.prefix, got, tt.want)
		}
	}
}

func TestExtractValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		prefix string
		want   string
	}{
		{
			name:   "value with colon",
			s:      "TIME:12:30:45",
			prefix: "TIME:",
			want:   "12:30:45",
		},
		{
			name:   "value with equals",
			s:      "EQUATION:x=10",
			prefix: "EQUATION:",
			want:   "x=10",
		},
		{
			name:   "value with dash",
			s:      "UUID:123-456-789",
			prefix: "UUID:",
			want:   "123-456-789",
		},
		{
			name:   "value with underscore",
			s:      "VAR:some_value",
			prefix: "VAR:",
			want:   "some_value",
		},
		{
			name:   "prefix appears twice",
			s:      "KEY:first KEY:second",
			prefix: "KEY:",
			want:   "first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractValue(tt.s, tt.prefix)
			if got != tt.want {
				t.Errorf("extractValue(%q, %q) = %q, want %q", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}
