package tui

import (
	"testing"
)

func TestStripControlSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "CSI colour codes stripped",
			input: "\x1b[31mhello\x1b[0m",
			want:  "hello",
		},
		{
			name:  "OSC title sequence stripped",
			input: "\x1b]0;window title\x07visible",
			want:  "visible",
		},
		{
			name:  "OSC with ST terminator stripped",
			input: "\x1b]2;title\x1b\\text",
			want:  "text",
		},
		{
			name:  "C0 control chars stripped",
			input: "\x00\x01\x02hello\x08world\x7f",
			want:  "helloworld",
		},
		{
			name:  "newlines preserved",
			input: "line1\nline2\r\nline3",
			want:  "line1\nline2\r\nline3",
		},
		{
			name:  "tabs preserved",
			input: "col1\tcol2",
			want:  "col1\tcol2",
		},
		{
			name:  "plain text unchanged",
			input: "hello world 123",
			want:  "hello world 123",
		},
		{
			name:  "mixed ANSI and text",
			input: "\x1b[1;32m✓\x1b[0m Done",
			want:  "✓ Done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripControlSequences(tt.input)
			if got != tt.want {
				t.Errorf("StripControlSequences(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "newlines collapsed to space",
			input: "main\nfeat",
			want:  "main feat",
		},
		{
			name:  "carriage returns collapsed",
			input: "branch\rname",
			want:  "branch name",
		},
		{
			name:  "leading/trailing whitespace trimmed",
			input: "  main  ",
			want:  "main",
		},
		{
			name:  "ANSI codes stripped",
			input: "\x1b[33mmain\x1b[0m",
			want:  "main",
		},
		{
			name:  "plain branch name unchanged",
			input: "feat/add-filter",
			want:  "feat/add-filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeField(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeField(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
