package tmux

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// controlSender is a long-lived tmux -C process whose stdin we write
// send-keys commands to. This avoids a fork+exec per keystroke.
type controlSender struct {
	mu    sync.Mutex
	stdin io.WriteCloser
	cmd   *exec.Cmd
}

var globalSender struct {
	mu sync.Mutex
	cs *controlSender
}

// getControlSender returns the global sender, starting it on first use.
// Returns nil if the tmux control mode process cannot be started.
func getControlSender() *controlSender {
	globalSender.mu.Lock()
	defer globalSender.mu.Unlock()
	if globalSender.cs != nil {
		return globalSender.cs
	}

	cmd := exec.Command("tmux", "-C", "attach")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	// Drain stdout/stderr so the pipe never blocks.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil
	}

	cs := &controlSender{cmd: cmd, stdin: stdin}
	globalSender.cs = cs
	go func() {
		_ = cmd.Wait()
		resetControlSender(cs)
	}()
	return cs
}

func resetControlSender(cs *controlSender) {
	globalSender.mu.Lock()
	defer globalSender.mu.Unlock()
	if globalSender.cs != cs {
		return
	}
	globalSender.cs = nil
	_ = cs.stdin.Close()
}

// SendKeyFast sends a single key to a pane via the persistent control
// connection, falling back to SendKeys if the connection is unavailable.
func SendKeyFast(paneID, key string) {
	cs := getControlSender()
	if cs == nil {
		SendKeys(paneID, key)
		return
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	// Quote the key so special characters are safe inside control mode.
	quoted := "'" + strings.ReplaceAll(key, "'", "'\\''") + "'"
	if _, err := fmt.Fprintf(cs.stdin, "send-keys -t %s %s\n", paneID, quoted); err != nil {
		resetControlSender(cs)
		SendKeys(paneID, key)
	}
}
