package model

import (
	"os"
	"path/filepath"
)

var (
	ClaudeHome   = filepath.Join(homeDir(), ".claude")
	CodexHome    = filepath.Join(homeDir(), ".codex")
	OpencodeHome = filepath.Join(homeDir(), ".opencode")
	TactHome     = filepath.Join(homeDir(), ".tact")
	DataDir      = filepath.Join(TactHome, "data")
	TodosDir     = filepath.Join(DataDir, "todos")
)

const (
	PanePollInterval      = 2  // seconds
	DiscoveryPollInterval = 10 // seconds
	CostPollInterval      = 30 // seconds
)

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func EnsureDirs() {
	os.MkdirAll(TodosDir, 0755)
}
