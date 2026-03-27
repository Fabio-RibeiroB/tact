package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// SessionStatus represents the current state of an AI session.
type SessionStatus int

const (
	StatusUnknown SessionStatus = iota // zero value = not yet polled
	StatusIdle
	StatusWorking
	StatusNeedsAttention
	StatusDisconnected
)

func (s SessionStatus) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusWorking:
		return "working"
	case StatusNeedsAttention:
		return "needs_attention"
	case StatusDisconnected:
		return "disconnected"
	}
	return "unknown"
}

func (s SessionStatus) Icon() string {
	switch s {
	case StatusIdle:
		return "●"
	case StatusWorking:
		return "◉"
	case StatusNeedsAttention:
		return "◉"
	case StatusDisconnected:
		return "○"
	}
	return "○"
}

// ProcessType identifies the AI tool running in a session.
type ProcessType int

const (
	ProcessUnknown ProcessType = iota
	ProcessClaude
	ProcessKiro
	ProcessCodex
	ProcessOpencode
)

func (p ProcessType) String() string {
	switch p {
	case ProcessClaude:
		return "claude"
	case ProcessKiro:
		return "kiro"
	case ProcessCodex:
		return "codex"
	case ProcessOpencode:
		return "opencode"
	}
	return "unknown"
}

func (p ProcessType) Icon() string {
	switch p {
	case ProcessClaude:
		return "C"
	case ProcessKiro:
		return "K"
	case ProcessCodex:
		return "X"
	case ProcessOpencode:
		return "O"
	}
	return "?"
}

// SessionInfo holds all data about a discovered AI session.
type SessionInfo struct {
	PaneID        string
	PanePID       int
	ProcessPID    int
	ProcessType   ProcessType
	SessionID     string
	Cwd           string
	ProjectName   string
	CustomName    string
	ManualTask    string
	GitBranch     string
	Status        SessionStatus
	ContextPct    int
	ContextTokens int
	ContextMax    int
	LastActivity  string
	TaskSummary   string    // inferred working summary/topic
	LastChecked   time.Time // semantic activity age used by the UI's AGO column
	LastPolled    time.Time // last successful TUI pane poll for this session
	PaneContent   string    // last captured pane output for preview
}

func (s *SessionInfo) BaseName() string {
	if s.ProjectName != "" {
		return s.ProjectName
	}
	if s.Cwd != "" {
		return filepath.Base(strings.TrimRight(s.Cwd, "/"))
	}
	return s.PaneID
}

func (s *SessionInfo) DisplayName() string {
	if s.CustomName != "" {
		return s.CustomName
	}
	return s.BaseName()
}

func (s *SessionInfo) WorkingTask() string {
	if task := normalizeStoredTask(s.ManualTask); task != "" {
		return task
	}
	return normalizeStoredTask(s.TaskSummary)
}

func (s *SessionInfo) ManualWorkingTask() string {
	return normalizeStoredTask(s.ManualTask)
}

func (s *SessionInfo) RenameKey() string {
	if s.SessionID != "" {
		return fmt.Sprintf("%s:session:%s", s.ProcessType.String(), s.SessionID)
	}
	if s.Cwd != "" {
		return fmt.Sprintf("%s:cwd:%s:pane:%s", s.ProcessType.String(), s.Cwd, s.PaneID)
	}
	return fmt.Sprintf("%s:pane:%s", s.ProcessType.String(), s.PaneID)
}

// TodoStatus represents the state of a todo item.
type TodoStatus int

const (
	TodoPending TodoStatus = iota
	TodoInProgress
	TodoDone
)

var todoStatusStrings = map[TodoStatus]string{
	TodoPending:    "pending",
	TodoInProgress: "in_progress",
	TodoDone:       "done",
}

var todoStatusFromString = map[string]TodoStatus{
	"pending":     TodoPending,
	"in_progress": TodoInProgress,
	"done":        TodoDone,
}

func (t TodoStatus) String() string { return todoStatusStrings[t] }

func (t TodoStatus) Icon() string {
	switch t {
	case TodoPending:
		return "○"
	case TodoInProgress:
		return "◑"
	case TodoDone:
		return "●"
	}
	return "?"
}

func (t TodoStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TodoStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, ok := todoStatusFromString[s]
	if !ok {
		return fmt.Errorf("unknown todo status: %s", s)
	}
	*t = v
	return nil
}

// NewTodoID generates a random 8-character hex ID.
func NewTodoID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b) // crypto/rand.Read never fails on Linux/macOS (reads /dev/urandom)
	return hex.EncodeToString(b)
}

// TodoItem is a single todo entry.
type TodoItem struct {
	ID            string     `json:"id"`
	Text          string     `json:"text"`
	Status        TodoStatus `json:"status"`
	SourceSession string     `json:"source_session,omitempty"`
	Project       string     `json:"project,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ProjectTodos is the top-level container for a project's todo list.
type ProjectTodos struct {
	Project     string     `json:"project"`
	ProjectPath string     `json:"project_path,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Items       []TodoItem `json:"items"`
}

// ClaudeStatus holds parsed statusline data from a Claude Code pane.
type ClaudeStatus struct {
	ContextPct    int
	ContextTokens int
	ContextMax    int
	GitBranch     string
	ProjectName   string
}

// SessionData holds enriched data parsed from a session's JSONL file.
type SessionData struct {
	LastMessage       string
	FirstHumanMessage string
	LastHumanMessage  string
	MessageCount      int
}
