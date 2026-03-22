package parser

import (
	"regexp"
	"strings"

	"github.com/fabiobrady/tact/internal/model"
)

var (
	attentionRe = regexp.MustCompile(
		`Do you want to proceed\??|Would you like to make the following edits\??|Do you want to make this edit\??|This command requires approval|Allow (once|always)\??|Allow command\??|Allow execution\??|\(y/n\)|\(Y/n\)|Yes/No\??|Yes, proceed\s*\(y\)|Press enter to confirm|don't ask again for these files|retry without sandbox|Waiting for your|❯\s*\d+\.\s*(Yes|No|Allow|Deny)`)
	claudeWorkingRe   = regexp.MustCompile(`✽\s*Vibing|⏺|Running\s+\w+\s+tool`)
	kiroWorkingRe     = regexp.MustCompile(`Thinking\.\.\.|Processing\.\.\.|Generating\.\.\.|⠋|⠙|⠹|⠸|⠼|⠴|⠦|⠧|⠇|⠏`)
	kiroIdleRe        = regexp.MustCompile(`\d+%\s*λ\s*>\s*$|^λ\s*>\s*$|^>\s*$`)
	kiroPromptRe      = regexp.MustCompile(`(\d+)%\s*λ`)
	codexWorkingRe    = regexp.MustCompile(`(?i)(thinking\.{3}|processing\.{3}|generating\.{3}|running\s+(bash|command|tool)|tool call|spawning|executing\s+\w+)`)
	codexIdleRe       = regexp.MustCompile(`^[›>] ?$|esc to interrupt|shift\+tab`)
	opencodeWorkingRe = regexp.MustCompile(`(?i)^\s*(thinking|processing|generating)\s*[.⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]+|^[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`)
	opencodeIdleRe    = regexp.MustCompile(`^[>›]\s*$|^[>›]\s+\S|^>\s*$`)
	opencodePromptRe  = regexp.MustCompile(`^[>›]\s*`)
	brailleSpinners   = "⠐⠂⠄⠈⠑⠒⠔⠘"
)

// DetectStatus determines session status from pane content and title.
func DetectStatus(paneContent, paneTitle string, procType model.ProcessType) model.SessionStatus {
	lines := strings.Split(paneContent, "\n")
	tailStart := 0
	if len(lines) > 20 {
		tailStart = len(lines) - 20
	}
	tail := strings.Join(lines[tailStart:], "\n")

	// 1. Needs attention (highest priority) — skip last 3 lines (statusline area)
	attentionEnd := len(lines)
	if procType == model.ProcessClaude && attentionEnd > 3 {
		attentionEnd = len(lines) - 3
	}
	attailStart := attentionEnd - 20
	if attailStart < 0 {
		attailStart = 0
	}
	attentionTail := strings.Join(lines[attailStart:attentionEnd], "\n")
	if attentionRe.MatchString(attentionTail) {
		return model.StatusNeedsAttention
	}

	// 2. Braille spinners in pane title = working
	if strings.ContainsAny(paneTitle, brailleSpinners) {
		return model.StatusWorking
	}

	// 2b. ✳ in pane title = idle (Claude sets this when done)
	if strings.Contains(paneTitle, "✳") {
		return model.StatusIdle
	}

	// 3. Process-specific checks
	switch procType {
	case model.ProcessClaude:
		// Check idle FIRST
		idleStart := 0
		if len(lines) > 10 {
			idleStart = len(lines) - 10
		}
		for _, line := range lines[idleStart:] {
			stripped := strings.TrimSpace(line)
			// ❯ with or without text = idle prompt (Claude shows suggestions like ❯ Try "...")
			if strings.HasPrefix(stripped, "❯") {
				return model.StatusIdle
			}
		}
		if strings.Contains(tail, "Cooked") || strings.Contains(tail, "Baked") {
			return model.StatusIdle
		}
		if strings.Contains(tail, "⏵⏵ accept edits") {
			return model.StatusIdle
		}
		if claudeWorkingRe.MatchString(tail) {
			return model.StatusWorking
		}

	case model.ProcessKiro:
		end := len(lines)
		start := end - 5
		if start < 0 {
			start = 0
		}
		// Check idle: prompt line like "14% λ > " at end (possibly with user input after)
		for i := end - 1; i >= start; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				continue
			}
			// Exact idle: prompt with no input or user input visible
			if kiroIdleRe.MatchString(trimmed) {
				return model.StatusIdle
			}
			// Also idle if the line contains the λ prompt (user may have typed something)
			if kiroPromptRe.MatchString(trimmed) {
				return model.StatusIdle
			}
			break // only check the last non-empty line
		}
		if kiroWorkingRe.MatchString(tail) {
			return model.StatusWorking
		}
		// Kiro default: idle (opposite of Claude — Kiro shows clear working indicators)
		return model.StatusIdle

	case model.ProcessCodex:
		end := len(lines)
		start := end - 8
		if start < 0 {
			start = 0
		}
		for i := end - 1; i >= start; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				continue
			}
			if codexIdleRe.MatchString(trimmed) {
				return model.StatusIdle
			}
			break
		}
		if codexWorkingRe.MatchString(tail) {
			return model.StatusWorking
		}
		return model.StatusIdle

	case model.ProcessOpencode:
		end := len(lines)
		start := end - 8
		if start < 0 {
			start = 0
		}
		// Check idle first: prompt like ">" or "›" at end
		for i := end - 1; i >= start; i-- {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" {
				continue
			}
			// Idle: bare prompt or prompt with text after it
			if opencodeIdleRe.MatchString(trimmed) || opencodePromptRe.MatchString(trimmed) {
				return model.StatusIdle
			}
			break
		}
		// Check for spinner characters in title
		if strings.ContainsAny(paneTitle, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
			return model.StatusWorking
		}
		if opencodeWorkingRe.MatchString(tail) {
			return model.StatusWorking
		}
		// Default to idle for opencode
		return model.StatusIdle
	}

	return model.StatusWorking // default if unsure
}
