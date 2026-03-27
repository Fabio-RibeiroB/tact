package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fabiobrady/tact/internal/model"
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

func TestActivityTimestamp(t *testing.T) {
	now := time.Date(2026, 3, 23, 20, 0, 0, 0, time.UTC)
	earlier := now.Add(-2 * time.Minute)

	tests := []struct {
		name string
		prev model.SessionInfo
		next model.SessionInfo
		want time.Time
	}{
		{
			name: "first attention detection resets age",
			prev: model.SessionInfo{Status: model.StatusWorking, LastChecked: earlier},
			next: model.SessionInfo{Status: model.StatusNeedsAttention},
			want: now,
		},
		{
			name: "continued attention preserves waiting age",
			prev: model.SessionInfo{Status: model.StatusNeedsAttention, LastChecked: earlier},
			next: model.SessionInfo{Status: model.StatusNeedsAttention},
			want: earlier,
		},
		{
			name: "pane content change counts as activity",
			prev: model.SessionInfo{Status: model.StatusWorking, LastChecked: earlier, PaneContent: "old"},
			next: model.SessionInfo{Status: model.StatusWorking, PaneContent: "new"},
			want: now,
		},
		{
			name: "unchanged pane content preserves age",
			prev: model.SessionInfo{Status: model.StatusIdle, LastChecked: earlier, PaneContent: "same"},
			next: model.SessionInfo{Status: model.StatusIdle, PaneContent: "same"},
			want: earlier,
		},
		{
			name: "disconnect preserves prior age",
			prev: model.SessionInfo{Status: model.StatusIdle, LastChecked: earlier},
			next: model.SessionInfo{Status: model.StatusDisconnected},
			want: earlier,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := activityTimestamp(tt.prev, tt.next, now)
			if !got.Equal(tt.want) {
				t.Fatalf("activityTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrappedIndex(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		delta   int
		want    int
	}{
		{name: "empty list stays zero", current: 0, total: 0, delta: 1, want: 0},
		{name: "single item wraps to itself", current: 0, total: 1, delta: 1, want: 0},
		{name: "wrap forward from end", current: 2, total: 3, delta: 1, want: 0},
		{name: "wrap backward from start", current: 0, total: 3, delta: -1, want: 2},
		{name: "move within bounds", current: 1, total: 3, delta: 1, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrappedIndex(tt.current, tt.total, tt.delta)
			if got != tt.want {
				t.Fatalf("wrappedIndex(%d, %d, %d) = %d, want %d", tt.current, tt.total, tt.delta, got, tt.want)
			}
		})
	}
}

func TestNewAppFallsBackToDefaultTheme(t *testing.T) {
	oldTactHome := model.TactHome
	oldConfigPath := model.ConfigPath
	oldDataDir := model.DataDir
	oldTodosDir := model.TodosDir
	oldTheme := currentTheme.Name
	t.Cleanup(func() {
		model.TactHome = oldTactHome
		model.ConfigPath = oldConfigPath
		model.DataDir = oldDataDir
		model.TodosDir = oldTodosDir
		applyThemeByName(oldTheme)
	})

	model.TactHome = t.TempDir()
	model.ConfigPath = filepath.Join(model.TactHome, "config.json")
	model.DataDir = filepath.Join(model.TactHome, "data")
	model.TodosDir = filepath.Join(model.DataDir, "todos")

	if err := os.WriteFile(model.ConfigPath, []byte("{\"theme\":\"not-a-theme\"}\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	app := newApp()
	if app.themeName != defaultThemeName {
		t.Fatalf("newApp() theme = %q, want %q", app.themeName, defaultThemeName)
	}
	if currentTheme.Name != defaultThemeName {
		t.Fatalf("currentTheme = %q, want %q", currentTheme.Name, defaultThemeName)
	}
}
