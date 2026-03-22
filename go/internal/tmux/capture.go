package tmux

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// CapturePane returns the visible content of a tmux pane.
func CapturePane(paneID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tmux", "capture-pane", "-t", paneID, "-e", "-p").Output()
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}

// GetPaneTitle returns the title of a tmux pane.
func GetPaneTitle(paneID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tmux", "display-message", "-t", paneID, "-p", "#{pane_title}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// SwitchToPane focuses the tmux window and pane.
func SwitchToPane(paneID string) bool {
	windowID := paneID
	if i := strings.LastIndex(paneID, "."); i >= 0 {
		windowID = paneID[:i]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if exec.CommandContext(ctx, "tmux", "select-window", "-t", windowID).Run() != nil {
		return false
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	return exec.CommandContext(ctx2, "tmux", "select-pane", "-t", paneID).Run() == nil
}

// ActivePaneID returns the pane ID of the currently active tmux pane.
func ActivePaneID() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tmux", "display-message", "-p",
		"#{session_name}:#{window_index}.#{pane_index}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// SendKeys sends keystrokes to a tmux pane.
func SendKeys(paneID string, keys ...string) bool {
	args := append([]string{"send-keys", "-t", paneID}, keys...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "tmux", args...).Run() == nil
}
