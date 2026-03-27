package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var SessionNamesPath = filepath.Join(DataDir, "session-names.json")

type SessionNames map[string]string

func LoadSessionNames() SessionNames {
	data, err := os.ReadFile(SessionNamesPath)
	if err != nil {
		return SessionNames{}
	}

	var names SessionNames
	if err := json.Unmarshal(data, &names); err != nil {
		return SessionNames{}
	}
	if names == nil {
		return SessionNames{}
	}
	return names
}

func SaveSessionNames(names SessionNames) error {
	EnsureDirs()

	normalized := SessionNames{}
	for key, value := range names {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			normalized[key] = trimmed
		}
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(DataDir, ".tact-session-names-*.tmp")
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

	return os.Rename(tmpName, SessionNamesPath)
}
