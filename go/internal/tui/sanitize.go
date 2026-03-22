package tui

import (
	"regexp"
	"strings"
)

var (
	// oscRe strips OSC, DCS, PM, APC sequences: ESC ] ... BEL/ST
	oscRe = regexp.MustCompile(`\x1b[\x5d\x50\x5e\x5f][\x08-\xff]*?(?:\x07|\x1b\\)`)
	// c0Re strips dangerous C0 control chars but preserves \t \n \r
	c0Re = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]`)
)

// StripControlSequences removes ANSI/VT escape sequences, OSC sequences,
// and dangerous C0 control characters from untrusted terminal content.
// Preserves newlines (\n), carriage returns (\r), and tabs (\t).
// Mitigates CWE-150 terminal escape injection.
func StripControlSequences(s string) string {
	s = oscRe.ReplaceAllString(s, "")
	s = ansiRe.ReplaceAllString(s, "")
	s = c0Re.ReplaceAllString(s, "")
	return s
}

// sanitizeField strips control sequences and collapses newlines/carriage
// returns to spaces. Use for single-line display fields: branch names,
// project names, task summaries.
func sanitizeField(s string) string {
	s = StripControlSequences(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}
