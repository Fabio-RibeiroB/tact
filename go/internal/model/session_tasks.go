package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var SessionTasksPath = filepath.Join(DataDir, "session-tasks.json")

type SessionTasks map[string]string

func normalizeStoredTask(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	prefixes := []string{"working:", "prompt:", "task:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			value = strings.TrimSpace(value[len(prefix):])
			break
		}
	}
	return strings.TrimSpace(value)
}

func LoadSessionTasks() SessionTasks {
	data, err := os.ReadFile(SessionTasksPath)
	if err != nil {
		return SessionTasks{}
	}

	var tasks SessionTasks
	if err := json.Unmarshal(data, &tasks); err != nil {
		return SessionTasks{}
	}
	if tasks == nil {
		return SessionTasks{}
	}
	for key, value := range tasks {
		if normalized := normalizeStoredTask(value); normalized != "" {
			tasks[key] = normalized
		} else {
			delete(tasks, key)
		}
	}
	return tasks
}

func SaveSessionTasks(tasks SessionTasks) error {
	EnsureDirs()

	normalized := SessionTasks{}
	for key, value := range tasks {
		if trimmed := normalizeStoredTask(value); trimmed != "" {
			normalized[key] = trimmed
		}
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(DataDir, ".tact-session-tasks-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, SessionTasksPath)
}
