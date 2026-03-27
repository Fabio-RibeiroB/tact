package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadSessionTasks(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	oldSessionTasksPath := SessionTasksPath
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
		SessionTasksPath = oldSessionTasksPath
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")
	SessionTasksPath = filepath.Join(DataDir, "session-tasks.json")

	tasks := SessionTasks{
		"codex:session:abc123": "patch navigation glitch",
		"codex:session:def456": "   ",
	}
	if err := SaveSessionTasks(tasks); err != nil {
		t.Fatalf("SaveSessionTasks() error = %v", err)
	}

	loaded := LoadSessionTasks()
	if got := loaded["codex:session:abc123"]; got != "patch navigation glitch" {
		t.Fatalf("LoadSessionTasks() kept task = %q, want %q", got, "patch navigation glitch")
	}
	if _, ok := loaded["codex:session:def456"]; ok {
		t.Fatalf("LoadSessionTasks() should drop blank tasks")
	}
}

func TestLoadSessionTasksInvalidJSON(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	oldSessionTasksPath := SessionTasksPath
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
		SessionTasksPath = oldSessionTasksPath
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")
	SessionTasksPath = filepath.Join(DataDir, "session-tasks.json")

	if err := os.MkdirAll(DataDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(SessionTasksPath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if loaded := LoadSessionTasks(); len(loaded) != 0 {
		t.Fatalf("LoadSessionTasks() len = %d, want 0", len(loaded))
	}
}

func TestLoadSessionTasksStripsLegacyPrefixes(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	oldSessionTasksPath := SessionTasksPath
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
		SessionTasksPath = oldSessionTasksPath
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")
	SessionTasksPath = filepath.Join(DataDir, "session-tasks.json")

	if err := os.MkdirAll(DataDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	body := []byte("{\n  \"codex:session:abc123\": \"Working: patch navigation\"\n}\n")
	if err := os.WriteFile(SessionTasksPath, body, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded := LoadSessionTasks()
	if got := loaded["codex:session:abc123"]; got != "patch navigation" {
		t.Fatalf("LoadSessionTasks() = %q, want %q", got, "patch navigation")
	}
}
