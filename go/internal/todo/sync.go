package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fabiobrady/tact/internal/model"
)

// ReadClaudeTodos reads Claude Code's internal todo files for a session (read-only).
func ReadClaudeTodos(sessionID string) []model.TodoItem {
	pattern := filepath.Join(model.ClaudeHome, "todos", sessionID+"-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var items []model.TodoItem
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var raw []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		}
		if json.Unmarshal(data, &raw) != nil {
			continue
		}
		for _, r := range raw {
			if r.Content == "" {
				continue
			}
			var s model.TodoStatus
			switch r.Status {
			case "completed":
				s = model.TodoDone
			case "in_progress":
				s = model.TodoInProgress
			default:
				s = model.TodoPending
			}
			items = append(items, model.TodoItem{
				Text:          r.Content,
				Status:        s,
				SourceSession: sessionID,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})
		}
	}
	return items
}
