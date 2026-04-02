package tui

import "testing"

func TestNormalizeTaskSummaryRejectsMetadataOnlyInput(t *testing.T) {
	input := `<environment_context><cwd>/tmp/project</cwd><shell>bash</shell></environment_context>`
	if got := normalizeTaskSummary(input); got != "" {
		t.Fatalf("normalizeTaskSummary() = %q, want empty", got)
	}
}

func TestNormalizeTaskSummaryStripsTaskLabels(t *testing.T) {
	for _, input := range []string{"Working: patch navigation", "Prompt: patch navigation", "Task: patch navigation"} {
		if got := normalizeTaskSummary(input); got != "patch navigation" {
			t.Fatalf("normalizeTaskSummary(%q) = %q, want %q", input, got, "patch navigation")
		}
	}
}
