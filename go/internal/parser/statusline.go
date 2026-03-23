package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/fabiobrady/tact/internal/model"
)

var (
	contextRe         = regexp.MustCompile(`Context:.*?(\d+)k/(\d+)k\s*\((\d+)%\)`)
	branchRe          = regexp.MustCompile(`⎇\s*(\S+)`)
	codexContextRe    = regexp.MustCompile(`(?i)\bcontext\b.*?(\d+)%`)
	opencodeContextRe = regexp.MustCompile(`(?i)\bcontext\b.*?(\d+)%`)
)

// ParseClaudeStatusline extracts context, branch, and project from pane content.
func ParseClaudeStatusline(paneContent string) model.ClaudeStatus {
	var cs model.ClaudeStatus
	// CRITICAL: normalize non-breaking spaces to regular spaces
	normalized := strings.ReplaceAll(paneContent, "\u00a0", " ")
	lines := strings.Split(normalized, "\n")

	for _, line := range lines {
		if m := contextRe.FindStringSubmatch(line); m != nil {
			cs.ContextTokens, _ = strconv.Atoi(m[1])
			cs.ContextTokens *= 1000
			cs.ContextMax, _ = strconv.Atoi(m[2])
			cs.ContextMax *= 1000
			cs.ContextPct, _ = strconv.Atoi(m[3])
		}
		if m := branchRe.FindStringSubmatch(line); m != nil {
			if m[1] != "no" { // filter "no git"
				cs.GitBranch = m[1]
			}
		}
	}

	// Project name: line with both "|" and "⎇", take text before first "|"
	for _, line := range lines {
		if strings.Contains(line, "|") && strings.Contains(line, "⎇") {
			project := strings.TrimSpace(strings.SplitN(line, "|", 2)[0])
			if project != "" && project != "no git" && project != "no" &&
				!strings.HasPrefix(project, "v") && !strings.HasPrefix(project, "Context") {
				cs.ProjectName = project
			}
			break
		}
	}

	return cs
}

// ParseKiroContext extracts context percentage from Kiro's prompt line (e.g. "14% λ >").
func ParseKiroContext(paneContent string) int {
	lines := strings.Split(paneContent, "\n")
	for i := len(lines) - 1; i >= max(0, len(lines)-10); i-- {
		if m := kiroContextRe.FindStringSubmatch(lines[i]); m != nil {
			pct, _ := strconv.Atoi(m[1])
			return pct
		}
	}
	return 0
}

var kiroContextRe = regexp.MustCompile(`(\d+)%\s*λ`)

// ParseCodexContext extracts a context percentage from visible Codex UI text.
func ParseCodexContext(paneContent string) int {
	lines := strings.Split(paneContent, "\n")
	for i := len(lines) - 1; i >= max(0, len(lines)-12); i-- {
		if m := codexContextRe.FindStringSubmatch(lines[i]); m != nil {
			pct, _ := strconv.Atoi(m[1])
			return pct
		}
	}
	return 0
}

// ParseOpencodeContext extracts a context percentage from visible opencode UI text.
func ParseOpencodeContext(paneContent string) int {
	lines := strings.Split(paneContent, "\n")
	for i := len(lines) - 1; i >= max(0, len(lines)-12); i-- {
		if m := opencodeContextRe.FindStringSubmatch(lines[i]); m != nil {
			pct, _ := strconv.Atoi(m[1])
			return pct
		}
	}
	return 0
}

var (
	claudePromptRe       = regexp.MustCompile(`^❯\s+(.+)`)
	claudeSuggestionRe   = regexp.MustCompile(`^❯\s+Try\s+"`)
	kiroPromptLineRe     = regexp.MustCompile(`\d+%\s*λ\s*>\s*(.+)`)
	codexPromptLineRe    = regexp.MustCompile(`^[›>] ?(.+)`)
	opencodePromptLineRe = regexp.MustCompile(`^[>›]\s*(.+)`)
)

// ExtractTaskSummary finds the last user prompt from pane content.
func ExtractTaskSummary(paneContent string, procType model.ProcessType) string {
	lines := strings.Split(paneContent, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || trimmed == "❯" || trimmed == "❯ " {
			continue
		}
		switch procType {
		case model.ProcessClaude:
			if claudePromptRe.MatchString(trimmed) {
				// Skip Claude suggestions like: ❯ Try "refactor eda.ipynb"
				if claudeSuggestionRe.MatchString(trimmed) {
					continue
				}
				return joinWrappedPrompt(lines, i)
			}
		case model.ProcessKiro:
			if kiroPromptLineRe.MatchString(trimmed) {
				return joinWrappedPrompt(lines, i)
			}
		case model.ProcessCodex:
			if codexPromptLineRe.MatchString(trimmed) {
				return joinWrappedPrompt(lines, i)
			}
		case model.ProcessOpencode:
			if opencodePromptLineRe.MatchString(trimmed) {
				return joinWrappedPrompt(lines, i)
			}
		}
	}
	return ""
}

// joinWrappedPrompt joins a prompt line with continuation lines below it
// (Claude/Kiro wrap long prompts across multiple lines).
func joinWrappedPrompt(lines []string, startIdx int) string {
	first := strings.TrimSpace(lines[startIdx])
	// Extract just the user text after the prompt symbol
	if m := claudePromptRe.FindStringSubmatch(first); m != nil {
		first = m[1]
	} else if m := kiroPromptLineRe.FindStringSubmatch(first); m != nil {
		first = m[1]
	} else if m := codexPromptLineRe.FindStringSubmatch(first); m != nil {
		first = m[1]
	} else if m := opencodePromptLineRe.FindStringSubmatch(first); m != nil {
		first = m[1]
	}
	parts := []string{first}
	// Grab continuation lines (indented, no prompt symbol)
	for j := startIdx + 1; j < len(lines) && j < startIdx+4; j++ {
		cont := strings.TrimSpace(lines[j])
		if cont == "" || strings.HasPrefix(cont, "─") || strings.HasPrefix(cont, "❯") ||
			strings.HasPrefix(cont, "›") || strings.HasPrefix(cont, ">") ||
			strings.Contains(cont, "⎇") || strings.HasPrefix(cont, "v2.") {
			break
		}
		parts = append(parts, cont)
	}
	result := strings.Join(parts, " ")
	if len(result) > 120 {
		result = result[:120] + "…"
	}
	return result
}
