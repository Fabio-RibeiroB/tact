package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMissingFile(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")

	cfg := LoadConfig()
	if cfg.Theme != "" {
		t.Fatalf("LoadConfig() theme = %q, want empty default", cfg.Theme)
	}
	if cfg.Style != "" {
		t.Fatalf("LoadConfig() style = %q, want empty default", cfg.Style)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")

	if err := SaveConfig(UIConfig{Theme: "sunset-grid", Style: "retro-bracket"}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	cfg := LoadConfig()
	if cfg.Theme != "sunset-grid" {
		t.Fatalf("LoadConfig() theme = %q, want %q", cfg.Theme, "sunset-grid")
	}
	if cfg.Style != "retro-bracket" {
		t.Fatalf("LoadConfig() style = %q, want %q", cfg.Style, "retro-bracket")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	oldTactHome := TactHome
	oldConfigPath := ConfigPath
	oldDataDir := DataDir
	oldTodosDir := TodosDir
	t.Cleanup(func() {
		TactHome = oldTactHome
		ConfigPath = oldConfigPath
		DataDir = oldDataDir
		TodosDir = oldTodosDir
	})

	TactHome = t.TempDir()
	ConfigPath = filepath.Join(TactHome, "config.json")
	DataDir = filepath.Join(TactHome, "data")
	TodosDir = filepath.Join(DataDir, "todos")

	if err := os.WriteFile(ConfigPath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := LoadConfig()
	if cfg.Theme != "" {
		t.Fatalf("LoadConfig() theme = %q, want empty default on invalid JSON", cfg.Theme)
	}
	if cfg.Style != "" {
		t.Fatalf("LoadConfig() style = %q, want empty default on invalid JSON", cfg.Style)
	}
}
