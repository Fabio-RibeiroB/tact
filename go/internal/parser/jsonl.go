package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fabiobrady/tact/internal/model"
)

var nonAlnumRe = regexp.MustCompile(`[^a-zA-Z0-9-]`)

// ParseSessionJSONL reads a Claude Code session JSONL file for cost and activity data.
func ParseSessionJSONL(sessionID, cwd string) model.SessionData {
	if data := parseCodexSessionJSONL(sessionID, cwd); data.MessageCount > 0 || data.LastMessage != "" || data.FirstHumanMessage != "" {
		return data
	}

	return parseClaudeSessionJSONL(sessionID, cwd)
}

func parseClaudeSessionJSONL(sessionID, cwd string) model.SessionData {
	var data model.SessionData
	if cwd == "" || sessionID == "" {
		return data
	}
	escaped := nonAlnumRe.ReplaceAllString(cwd, "-")
	jsonlPath := filepath.Join(model.ClaudeHome, "projects", escaped, sessionID+".jsonl")

	f, err := os.Open(jsonlPath)
	if err != nil {
		return data
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // handle large lines
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content json.RawMessage `json:"content"`
			Message *struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
				Usage   *struct {
					InputTokens              int `json:"input_tokens"`
					OutputTokens             int `json:"output_tokens"`
					CacheReadInputTokens     int `json:"cache_read_input_tokens"`
					CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage *struct {
				InputTokens              int `json:"input_tokens"`
				OutputTokens             int `json:"output_tokens"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		// Resolve role and content: top-level or nested under message
		role := entry.Role
		content := entry.Content
		var usage = entry.Usage
		if entry.Message != nil {
			if role == "" {
				role = entry.Message.Role
			}
			if role == "" {
				role = entry.Type // "user" or "assistant" at top level
			}
			if content == nil {
				content = entry.Message.Content
			}
			if usage == nil {
				usage = entry.Message.Usage
			}
		}
		if role == "" {
			role = entry.Type
		}
		if usage != nil {
			data.Cost.InputTokens += usage.InputTokens
			data.Cost.OutputTokens += usage.OutputTokens
			data.Cost.CacheReadTokens += usage.CacheReadInputTokens
			data.Cost.CacheCreationTokens += usage.CacheCreationInputTokens
		}
		if role == "assistant" {
			data.MessageCount++
			text := extractText(content)
			if text != "" {
				if len(text) > 200 {
					text = text[:200]
				}
				data.LastMessage = text
			}
		}
		if role == "human" || role == "user" {
			text := extractText(content)
			if text != "" && !strings.Contains(text, "[Request interrupted") {
				if len(text) > 120 {
					text = text[:120]
				}
				if data.FirstHumanMessage == "" {
					data.FirstHumanMessage = text
				}
				data.LastHumanMessage = text
			}
		}
	}
	data.Cost.Compute()
	return data
}

func parseCodexSessionJSONL(sessionID, cwd string) model.SessionData {
	var data model.SessionData
	if cwd == "" || sessionID == "" {
		return data
	}
	jsonlPath := filepath.Join(model.CodexHome, "sessions", "*", "*", "*", "*"+sessionID+"*.jsonl")
	matches, err := filepath.Glob(jsonlPath)
	if err != nil || len(matches) == 0 {
		return data
	}
	latest := matches[0]
	for _, match := range matches[1:] {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		current, err := os.Stat(latest)
		if err != nil || info.ModTime().After(current.ModTime()) {
			latest = match
		}
	}

	f, err := os.Open(latest)
	if err != nil {
		return data
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry struct {
			Type    string `json:"type"`
			Payload struct {
				Type    string `json:"type"`
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}

		if entry.Type != "response_item" || entry.Payload.Type != "message" {
			continue
		}
		text := joinContentBlocks(entry.Payload.Content)
		if text == "" {
			continue
		}
		switch entry.Payload.Role {
		case "assistant":
			data.MessageCount++
			if len(text) > 200 {
				text = text[:200]
			}
			data.LastMessage = text
		case "user":
			if len(text) > 120 {
				text = text[:120]
			}
			if data.FirstHumanMessage == "" {
				data.FirstHumanMessage = text
			}
			data.LastHumanMessage = text
		}
	}

	if data.FirstHumanMessage == "" || data.LastHumanMessage == "" {
		historyPath := filepath.Join(model.CodexHome, "history.jsonl")
		fallback := readCodexHistory(historyPath, sessionID)
		if data.FirstHumanMessage == "" {
			data.FirstHumanMessage = fallback.FirstHumanMessage
		}
		if data.LastHumanMessage == "" {
			data.LastHumanMessage = fallback.LastHumanMessage
		}
	}

	return data
}

func joinContentBlocks(blocks []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	var parts []string
	for _, block := range blocks {
		if block.Type == "output_text" || block.Type == "input_text" || block.Type == "text" {
			if block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
	}
	return strings.Join(parts, " ")
}

func readCodexHistory(path, sessionID string) model.SessionData {
	var data model.SessionData
	f, err := os.Open(path)
	if err != nil {
		return data
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var entry struct {
			SessionID string `json:"session_id"`
			Text      string `json:"text"`
		}
		if json.Unmarshal(line, &entry) != nil || entry.SessionID != sessionID || entry.Text == "" {
			continue
		}
		text := entry.Text
		if len(text) > 120 {
			text = text[:120]
		}
		if data.FirstHumanMessage == "" {
			data.FirstHumanMessage = text
		}
		data.LastHumanMessage = text
	}
	return data
}

func extractText(raw json.RawMessage) string {
	// Try string first
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// Try array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}
