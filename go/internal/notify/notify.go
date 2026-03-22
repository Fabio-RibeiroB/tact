package notify

import (
	"fmt"
	"os"
	"os/exec"
)

const towerIcon = "/System/Library/CoreServices/CoreTypes.bundle/Contents/Resources/com.apple.airport-extreme-tower.icns"

// Notify sends OS-level notifications when a session needs attention.
func Notify(projectName, processType string) {
	label := processType
	if projectName != "" {
		label = projectName + " (" + processType + ")"
	}
	msg := fmt.Sprintf("%s needs attention", label)

	// Prefer terminal-notifier (supports custom icon)
	if tn, err := exec.LookPath("terminal-notifier"); err == nil {
		exec.Command(tn,
			"-title", "Tact",
			"-message", msg,
			"-sound", "Ping",
			"-appIcon", towerIcon,
		).Start()
	} else {
		// Fallback to osascript (shows Script Editor scroll icon)
		exec.Command("osascript", "-e",
			fmt.Sprintf(`display notification "%s" with title "Tact" sound name "Ping"`, msg)).Start()
	}

	// OSC 9 escape sequence (iTerm2, Kitty, WezTerm)
	fmt.Fprintf(os.Stdout, "\033]9;%s\007", msg)

	// tmux display-message (5 second duration)
	exec.Command("tmux", "display-message", "-d", "5000", fmt.Sprintf("⚠ %s", msg)).Start()

	// Terminal bell
	fmt.Fprint(os.Stdout, "\a")
}
