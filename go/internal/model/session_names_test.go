package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionInfoNameMethods(t *testing.T) {
	s := SessionInfo{
		ProcessType: ProcessCodex,
		SessionID:   "abc123",
		Cwd:         "/tmp/project",
		ProjectName: "project",
	}

	if got := s.BaseName(); got != "project" {
		t.Fatalf("BaseName() = %q, want %q", got, "project")
	}
	if got := s.DisplayName(); got != "project" {
		t.Fatalf("DisplayName() = %q, want %q", got, "project")
	}
	s.CustomName = "wordle-fix"
	if got := s.DisplayName(); got != "wordle-fix" {
		t.Fatalf("DisplayName() with custom name = %q, want %q", got, "wordle-fix")
	}
	if got := s.RenameKey(); got != "codex:session:abc123" {
		t.Fatalf("RenameKey() = %q, want %q", got, "codex:session:abc123")
	}
}

func TestSaveAndLoadSessionNames(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	oldSessionNamesPath := SessionNamesPath
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
		SessionNamesPath = oldSessionNamesPath
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")
	SessionNamesPath = filepath.Join(DataDir, "session-names.json")

	names := SessionNames{
		"codex:session:abc123": "wordle-fix",
		"codex:session:def456": "   ",
	}
	if err := SaveSessionNames(names); err != nil {
		t.Fatalf("SaveSessionNames() error = %v", err)
	}

	loaded := LoadSessionNames()
	if got := loaded["codex:session:abc123"]; got != "wordle-fix" {
		t.Fatalf("LoadSessionNames() kept name = %q, want %q", got, "wordle-fix")
	}
	if _, ok := loaded["codex:session:def456"]; ok {
		t.Fatalf("LoadSessionNames() should drop blank names")
	}
}

func TestLoadSessionNamesInvalidJSON(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	oldSessionNamesPath := SessionNamesPath
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
		SessionNamesPath = oldSessionNamesPath
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")
	SessionNamesPath = filepath.Join(DataDir, "session-names.json")

	if err := os.MkdirAll(DataDir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(SessionNamesPath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if loaded := LoadSessionNames(); len(loaded) != 0 {
		t.Fatalf("LoadSessionNames() len = %d, want 0", len(loaded))
	}
}
