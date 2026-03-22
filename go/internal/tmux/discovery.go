package tmux

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fabiobrady/tact/internal/model"
	"github.com/shirou/gopsutil/v4/process"
)

var (
	claudePaneRe = []string{
		"❯ ",
		"Do you want to proceed?",
		"Bash command",
		"⏵⏵ accept edits",
		"Crunched",
		"Cooked",
		"Baked",
	}
	codexPaneRe = []string{
		"gpt-",
		"codex",
		"Would you like to make the following edits?",
		"Do you want to make this edit",
		"This command requires approval",
		"Yes, proceed (y)",
		"Press enter to confirm or esc to cancel",
		"and don't ask again for these files",
		"Reason: command failed; retry without sandbox?",
	}
)

// PaneInfo holds raw data from tmux list-panes.
type PaneInfo struct {
	PaneID    string
	PanePID   int
	PaneTitle string
	PaneCmd   string
}

// ListPanes returns all tmux panes.
func ListPanes() ([]PaneInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tmux", "list-panes", "-a", "-F",
		"#{session_name}:#{window_index}.#{pane_index}\t#{pane_pid}\t#{pane_title}\t#{pane_current_command}").Output()
	if err != nil {
		return nil, err
	}
	var panes []PaneInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		pid, _ := strconv.Atoi(parts[1])
		panes = append(panes, PaneInfo{PaneID: parts[0], PanePID: pid, PaneTitle: parts[2], PaneCmd: parts[3]})
	}
	return panes, nil
}

// FindAIProcess walks the process tree from panePID via BFS (max depth 5).
// Claude is checked first, then Codex, then Kiro, then Opencode.
func FindAIProcess(panePID int) (model.ProcessType, int) {
	// Collect all PIDs in the tree
	var allPIDs []int32
	queue := []struct {
		pid   int32
		depth int
	}{{int32(panePID), 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth > 5 {
			continue
		}
		allPIDs = append(allPIDs, cur.pid)
		p, err := process.NewProcess(cur.pid)
		if err != nil {
			continue
		}
		children, err := p.Children()
		if err != nil {
			continue
		}
		for _, c := range children {
			queue = append(queue, struct {
				pid   int32
				depth int
			}{c.Pid, cur.depth + 1})
		}
	}

	// Check Claude first
	for _, pid := range allPIDs {
		if isClaudeProcess(pid) {
			return model.ProcessClaude, int(pid)
		}
	}
	// Then Codex
	for _, pid := range allPIDs {
		if isCodexProcess(pid) {
			return model.ProcessCodex, int(pid)
		}
	}
	// Then Kiro
	for _, pid := range allPIDs {
		if isKiroProcess(pid) {
			return model.ProcessKiro, int(pid)
		}
	}
	// Then Opencode
	for _, pid := range allPIDs {
		if isOpencodeProcess(pid) {
			return model.ProcessOpencode, int(pid)
		}
	}
	return model.ProcessUnknown, 0
}

func isClaudeProcess(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}
	args, err := p.CmdlineSlice()
	if err != nil || len(args) == 0 {
		return false
	}
	// Exclude tact (our own TUI)
	for _, arg := range args[:min(len(args), 2)] {
		base := filepath.Base(arg)
		if base == "tact" {
			return false
		}
	}
	limit := min(len(args), 2)
	for _, arg := range args[:limit] {
		lower := strings.ToLower(arg)
		if strings.Contains(lower, "claude") || strings.Contains(lower, "claude-code") {
			return true
		}
		if strings.HasSuffix(arg, "/claude") {
			return true
		}
	}
	// Definitive: session file exists (Claude's process name is its version number)
	sessionFile := filepath.Join(model.ClaudeHome, "sessions", fmt.Sprintf("%d.json", pid))
	if _, err := os.Stat(sessionFile); err == nil {
		return true
	}
	return false
}

func isKiroProcess(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}
	args, err := p.CmdlineSlice()
	if err != nil || len(args) == 0 {
		return false
	}
	limit := min(len(args), 2)
	for _, arg := range args[:limit] {
		base := filepath.Base(arg)
		// Must be the actual binary, NOT shells named "zsh (kiro-cli-term)"
		if base == "kiro" || base == "kiro-cli" {
			return true
		}
		if arg == "kiro" || arg == "kiro-cli" {
			return true
		}
	}
	return false
}

func isCodexProcess(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}
	name, err := p.Name()
	if err == nil {
		lower := strings.ToLower(name)
		if lower == "codex" || strings.Contains(lower, "codex") {
			return true
		}
	}
	args, err := p.CmdlineSlice()
	if err != nil || len(args) == 0 {
		return false
	}
	limit := min(len(args), 3)
	for _, arg := range args[:limit] {
		base := strings.ToLower(filepath.Base(arg))
		lower := strings.ToLower(arg)
		if base == "codex" || strings.HasPrefix(base, "codex-") {
			return true
		}
		if strings.Contains(lower, "codex") && !strings.Contains(lower, "tact") {
			return true
		}
	}
	return false
}

func isOpencodeProcess(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		return false
	}
	name, err := p.Name()
	if err == nil {
		lower := strings.ToLower(name)
		if lower == "opencode" || strings.Contains(lower, "opencode") {
			return true
		}
	}
	args, err := p.CmdlineSlice()
	if err != nil || len(args) == 0 {
		return false
	}
	limit := min(len(args), 3)
	for _, arg := range args[:limit] {
		base := strings.ToLower(filepath.Base(arg))
		lower := strings.ToLower(arg)
		if base == "opencode" || strings.HasPrefix(base, "opencode-") {
			return true
		}
		if strings.Contains(lower, "opencode") && !strings.Contains(lower, "tact") {
			return true
		}
	}
	return false
}

// GetClaudeSessionInfo reads session metadata from ~/.claude/sessions/<PID>.json.
func GetClaudeSessionInfo(processPID int) (sessionID, cwd string) {
	dir := filepath.Join(model.ClaudeHome, "sessions")
	// Direct lookup
	data, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%d.json", processPID)))
	if err == nil {
		var info struct {
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
		}
		if json.Unmarshal(data, &info) == nil && info.SessionID != "" {
			return info.SessionID, info.Cwd
		}
	}
	// Fallback: scan all files for matching pid field
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var info struct {
			Pid       int    `json:"pid"`
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
		}
		if json.Unmarshal(data, &info) == nil && info.Pid == processPID {
			return info.SessionID, info.Cwd
		}
	}
	return "", ""
}

// GetCodexSessionInfo finds the newest Codex session for a working directory.
// Codex stores session metadata in session JSONL files rather than per-PID files.
func GetCodexSessionInfo(cwd string) (sessionID, sessionCwd string) {
	if cwd == "" {
		return "", ""
	}
	pattern := filepath.Join(model.CodexHome, "sessions", "*", "*", "*", "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", ""
	}

	var newestPath string
	var newestTime time.Time
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		id, fileCwd := readCodexSessionMeta(match)
		if id == "" || fileCwd == "" || fileCwd != cwd {
			continue
		}
		if newestPath == "" || info.ModTime().After(newestTime) {
			newestPath = match
			newestTime = info.ModTime()
			sessionID = id
			sessionCwd = fileCwd
		}
	}
	return sessionID, sessionCwd
}

func readCodexSessionMeta(path string) (sessionID, cwd string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	var entry struct {
		Type    string `json:"type"`
		Payload struct {
			ID  string `json:"id"`
			Cwd string `json:"cwd"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(f).Decode(&entry); err != nil {
		return "", ""
	}
	if entry.Type != "session_meta" {
		return "", ""
	}
	return entry.Payload.ID, entry.Payload.Cwd
}

// GetOpencodeSessionInfo reads session metadata from ~/.opencode/sessions/<PID>.json.
func GetOpencodeSessionInfo(processPID int) (sessionID, cwd string) {
	sessionFile := filepath.Join(model.OpencodeHome, "sessions", fmt.Sprintf("%d.json", processPID))
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return "", ""
	}
	var info struct {
		SessionID string `json:"sessionId"`
		Cwd       string `json:"cwd"`
	}
	if json.Unmarshal(data, &info) == nil && info.SessionID != "" {
		return info.SessionID, info.Cwd
	}
	return "", ""
}

// GetProcessCwd returns the working directory of a process.
func GetProcessCwd(pid int) string {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return ""
	}
	cwd, err := p.Cwd()
	if err != nil {
		return ""
	}
	return cwd
}

// getPanePath returns the current working directory of a tmux pane.
func getPanePath(paneID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tmux", "display-message", "-t", paneID, "-p", "#{pane_current_path}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// DiscoverSessions finds all AI coding sessions running in tmux.
func DiscoverSessions() []model.SessionInfo {
	panes, err := ListPanes()
	if err != nil {
		return nil
	}
	var sessions []model.SessionInfo
	for _, pane := range panes {
		procType, procPID := FindAIProcess(pane.PanePID)
		if procType == model.ProcessUnknown && pane.PaneCmd == "ssh" {
			procType = inferRemoteAIProcess(pane.PaneID, pane.PaneTitle)
		}
		if procType == model.ProcessUnknown {
			continue
		}
		s := model.SessionInfo{
			PaneID:      pane.PaneID,
			PanePID:     pane.PanePID,
			ProcessPID:  procPID,
			ProcessType: procType,
			ContextMax:  200_000,
			LastChecked: time.Now(),
		}
		if procType == model.ProcessClaude && procPID > 0 {
			s.SessionID, s.Cwd = GetClaudeSessionInfo(procPID)
		}
		if s.Cwd == "" && procPID > 0 {
			s.Cwd = GetProcessCwd(procPID)
		}
		// Fallback: use tmux pane's current path
		if s.Cwd == "" {
			s.Cwd = getPanePath(pane.PaneID)
		}
		if procType == model.ProcessCodex {
			s.SessionID, s.Cwd = GetCodexSessionInfo(s.Cwd)
			if s.Cwd == "" && procPID > 0 {
				s.Cwd = GetProcessCwd(procPID)
			}
		}
		if procType == model.ProcessOpencode && procPID > 0 {
			s.SessionID, s.Cwd = GetOpencodeSessionInfo(procPID)
		}
		if s.Cwd == "" && procPID > 0 {
			s.Cwd = GetProcessCwd(procPID)
		}
		// Fallback: use tmux pane's current path
		if s.Cwd == "" {
			s.Cwd = getPanePath(pane.PaneID)
		}
		if s.Cwd != "" {
			s.ProjectName = filepath.Base(s.Cwd)
		}
		sessions = append(sessions, s)
	}
	return sessions
}

func inferRemoteAIProcess(paneID, paneTitle string) model.ProcessType {
	content := CapturePane(paneID)
	if content == "" {
		return model.ProcessUnknown
	}
	tail := tailLines(content, 80)
	if matchesAny(tail, codexPaneRe) {
		return model.ProcessCodex
	}
	if matchesAny(tail, claudePaneRe) {
		return model.ProcessClaude
	}
	if strings.Contains(paneTitle, "Claude") || strings.Contains(paneTitle, "✳") {
		return model.ProcessClaude
	}
	return model.ProcessUnknown
}

func matchesAny(content string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(content, needle) {
			return true
		}
	}
	return false
}

func tailLines(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	return strings.Join(lines[len(lines)-maxLines:], "\n")
}
