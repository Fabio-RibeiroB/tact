package parser

import (
	"testing"

	"github.com/fabiobrady/tact/internal/model"
)

func TestDetectStatusCodexWorkingFooter(t *testing.T) {
	pane := "diff preview\n\nWorking (45s • esc to interrupt)\n\n› Run /review on my current changes\n"
	got := DetectStatus(pane, "", model.ProcessCodex)
	if got != model.StatusWorking {
		t.Fatalf("DetectStatus() = %v, want %v", got, model.StatusWorking)
	}
}

func TestDetectStatusCodexPromptIsIdle(t *testing.T) {
	pane := "context 78%\n› review this diff\n"
	got := DetectStatus(pane, "", model.ProcessCodex)
	if got != model.StatusIdle {
		t.Fatalf("DetectStatus() = %v, want %v", got, model.StatusIdle)
	}
}
