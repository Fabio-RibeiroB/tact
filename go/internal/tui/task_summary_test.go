package tui

import "testing"

func TestChooseTaskSummaryPrefersSubstantiveFirstMessage(t *testing.T) {
	got := chooseTaskSummary("fix sunset-grid color bleed", "yes", "")
	if got != "fix sunset-grid color bleed" {
		t.Fatalf("chooseTaskSummary() = %q, want %q", got, "fix sunset-grid color bleed")
	}
}

func TestChooseTaskSummaryFallsBackToSubstantiveLastMessage(t *testing.T) {
	got := chooseTaskSummary("ok", "patch session list jumping", "")
	if got != "patch session list jumping" {
		t.Fatalf("chooseTaskSummary() = %q, want %q", got, "patch session list jumping")
	}
}

func TestIsSubstantiveTaskRejectsShortReplies(t *testing.T) {
	for _, value := range []string{"yes", "ok", "continue", "push"} {
		if isSubstantiveTask(value) {
			t.Fatalf("isSubstantiveTask(%q) = true, want false", value)
		}
	}
}

func TestChooseTaskSummaryStripsEnvironmentContext(t *testing.T) {
	first := `<environment_context>
  <cwd>/home/fabio/Documents/GitHub/tact</cwd>
  <shell>bash</shell>
</environment_context>

why is the ui jumping up and down now when I hit j/k?`
	got := chooseTaskSummary(first, "", "")
	want := "why is the ui jumping up and down now when I hit j/k?"
	if got != want {
		t.Fatalf("chooseTaskSummary() = %q, want %q", got, want)
	}
}

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
