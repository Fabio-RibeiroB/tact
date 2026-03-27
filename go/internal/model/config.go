package model

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	ClaudeHome   = filepath.Join(homeDir(), ".claude")
	CodexHome    = filepath.Join(homeDir(), ".codex")
	OpencodeHome = filepath.Join(homeDir(), ".opencode")
	TactHome     = filepath.Join(homeDir(), ".tact")
	ConfigPath   = filepath.Join(TactHome, "config.json")
	DataDir      = filepath.Join(TactHome, "data")
	TodosDir     = filepath.Join(DataDir, "todos")
)

const (
	PanePollInterval        = 1  // seconds
	DiscoveryPollInterval   = 10 // seconds
	SessionDataPollInterval = 30 // seconds
)

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func EnsureDirs() {
	os.MkdirAll(TactHome, 0700)
	os.MkdirAll(TodosDir, 0700)
}

type UIConfig struct {
	Theme string `json:"theme,omitempty"`
	Style string `json:"style,omitempty"`
}

func LoadConfig() UIConfig {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return UIConfig{}
	}

	var cfg UIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return UIConfig{}
	}
	return cfg
}

func SaveConfig(cfg UIConfig) error {
	EnsureDirs()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(ConfigPath, data, 0600)
}
