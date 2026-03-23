package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fabiobrady/tact/internal/model"
)

var attentionRe = regexp.MustCompile(
	`(?i)Do you want to proceed\?|Allow\s+(once|always)\?|Allow command\?|Allow execution\?|\(y/n\)|\(Y/n\)|Yes/No\?|Waiting for your`)

// ── Too small ────────────────────────────────────────────────────────

func renderTooSmall(w, h int) string {
	msg := fmt.Sprintf(
		"Terminal too small (%dx%d)\nMinimum size: %dx%d",
		w, h, minWidth, minHeight,
	)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f7768e")).
		Padding(1, 2).
		Render(msg)
}

// ── Header ──────────────────────────────────────────────────────────

func renderHeader(sessions []model.SessionInfo, width int, notifyEnabled bool, selected *model.SessionInfo) string {
	attn := 0
	var totalCost float64
	working := 0
	for _, s := range sessions {
		if s.Status == model.StatusNeedsAttention {
			attn++
		}
		if s.Status == model.StatusWorking {
			working++
		}
		totalCost += s.CostUSD
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(tokenFgAccent).
		Render("⚡ TACT")

	sessionPill := headerPill("Sessions", fmt.Sprintf("%d", len(sessions)), tokenFgDefault)

	var attnPill string
	if attn > 0 {
		attnPill = headerPill("", fmt.Sprintf("⚠ %d NEED ATTENTION", attn), tokenFgDanger)
	} else {
		attnPill = headerPill("", "✓ all clear", tokenFgSuccess)
	}

	costPill := headerPill("Today", fmt.Sprintf("$%.2f", totalCost), tokenFgWarning)
	clockPill := headerPill("", time.Now().Format("15:04:05"), tokenFgMuted)

	notifyStr := lipgloss.NewStyle().Foreground(tokenFgSuccess).Render("🔔")
	if !notifyEnabled {
		notifyStr = lipgloss.NewStyle().Foreground(tokenFgMuted).Render("🔕")
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center,
		title, "  ", sessionPill, " ", attnPill, " ", costPill)
	right := lipgloss.JoinHorizontal(lipgloss.Center, notifyStr, " ", clockPill)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := lipgloss.NewStyle().
		Background(lipgloss.Color("#1f2335")).
		Width(width).
		Render(left + strings.Repeat(" ", gap) + right)

	// Summary line
	var summaryParts []string
	if working > 0 {
		summaryParts = append(summaryParts,
			lipgloss.NewStyle().Foreground(tokenFgInfo).Render(fmt.Sprintf("⟳ %d working", working)))
	}
	if selected != nil && selected.TaskSummary != "" {
		activity := sanitizeField(selected.TaskSummary)
		maxLen := width - 40
		if maxLen < 30 {
			maxLen = 30
		}
		if len(activity) > maxLen {
			activity = activity[:maxLen] + "…"
		}
		label := selected.DisplayName() + ": "
		summaryParts = append(summaryParts,
			lipgloss.NewStyle().Foreground(tokenFgMuted).Render(label)+
				lipgloss.NewStyle().Foreground(tokenFgDefault).Italic(true).Render(activity))
	}
	summaryLine := ""
	if len(summaryParts) > 0 {
		summaryLine = lipgloss.NewStyle().
			Background(lipgloss.Color("#1f2335")).
			Width(width).
			Render("  " + strings.Join(summaryParts, "  │  "))
	}

	if summaryLine != "" {
		return bar + "\n" + summaryLine
	}
	return bar
}

// ── Tab bar ─────────────────────────────────────────────────────────

func renderTabBar(activeTab, width int) string {
	tabs := []struct {
		key   string
		label string
		tab   int
	}{
		{"1", "Sessions", tabSessions},
		{"2", "Todos", tabTodos},
		{"3", "Output", tabOutput},
	}

	var parts []string
	for _, t := range tabs {
		keyStr := lipgloss.NewStyle().Foreground(tokenFgMuted).Render("[" + t.key + "]")
		if t.tab == activeTab {
			label := lipgloss.NewStyle().
				Foreground(tokenFgAccent).
				Bold(true).
				Underline(true).
				Render(t.label)
			parts = append(parts, keyStr+" "+label)
		} else {
			label := lipgloss.NewStyle().Foreground(tokenFgMuted).Render(t.label)
			parts = append(parts, keyStr+" "+label)
		}
	}

	bar := "  " + strings.Join(parts, "   ")
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1a1b26")).
		Width(width).
		Render(bar)
}

// ── Filter bar ──────────────────────────────────────────────────────

func renderFilterBar(active bool, text string, total, shown int, width int) string {
	if !active && text == "" {
		return ""
	}

	var prompt string
	if active {
		cursor := lipgloss.NewStyle().Foreground(tokenFgWarning).Render("/")
		inputText := lipgloss.NewStyle().Foreground(tokenFgDefault).Render(text + "█")
		prompt = cursor + " " + inputText
	} else {
		// Filter applied but bar hidden
		badge := lipgloss.NewStyle().Foreground(tokenFgWarning).Render("[filter: " + text + "]")
		prompt = badge
	}

	count := lipgloss.NewStyle().Foreground(tokenFgMuted).
		Render(fmt.Sprintf("Showing %d of %d", shown, total))

	gap := width - lipgloss.Width(prompt) - lipgloss.Width(count) - 4
	if gap < 1 {
		gap = 1
	}
	line := "  " + prompt + strings.Repeat(" ", gap) + count
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1f2335")).
		Width(width).
		Render(line)
}

// ── Session table (k9s-style, 1 row per session) ─────────────────────

var sortLabels = []string{"", "↑ status", "↑ cost", "↑ age", "↑ name"}

const typeColWidth = 8 // "Claude  " fits in 8 chars

func renderSessionTableHeader(width, sortMode int) string {
	// indicator(2) + ST(2) + TYPE(typeColWidth) + AGO(5) + space(1)
	nameWidth := width - 2 - 2 - typeColWidth - 5 - 1
	if nameWidth < 8 {
		nameWidth = 8
	}

	nameLabel := "NAME"
	if sortMode > 0 && sortMode < len(sortLabels) {
		nameLabel = "NAME (" + sortLabels[sortMode] + ")"
	}

	dim := lipgloss.NewStyle().Foreground(tokenFgMuted)
	row := dim.Width(2).Render("") +
		dim.Width(2).Render("") +
		dim.Width(typeColWidth).Render("AI") +
		dim.Width(nameWidth).Render(nameLabel) +
		" " + dim.Width(4).Render("AGO")

	return row
}

func renderSessionTableRow(s model.SessionInfo, selected, blinkOn bool, spinnerIdx, width int) string {
	// Selected indicator: bright left arrow (2 chars) vs padding
	var indicator string
	if selected {
		indicator = lipgloss.NewStyle().Foreground(colorBorderHi).Bold(true).Render("▶ ")
	} else {
		indicator = "  "
	}

	icon := statusIcon(s.Status, blinkOn, spinnerIdx)
	tStr := typeTag(s.ProcessType)

	// indicator(2) + ST(2) + TYPE(typeColWidth) + AGO(5) + space(1)
	nameWidth := width - 2 - 2 - typeColWidth - 5 - 1
	if nameWidth < 8 {
		nameWidth = 8
	}

	name := s.DisplayName()
	if len(name) > nameWidth {
		name = name[:nameWidth-1] + "…"
	}

	nameStyle := lipgloss.NewStyle().Width(nameWidth).Foreground(tokenFgDefault)
	if selected {
		nameStyle = nameStyle.Foreground(colorText).Bold(true)
	}
	if s.Status == model.StatusNeedsAttention {
		nameStyle = nameStyle.Foreground(tokenFgDanger).Bold(true)
	} else if s.Status == model.StatusWorking && !selected {
		nameStyle = nameStyle.Foreground(tokenFgInfo)
	}

	ago := formatAge(s.LastChecked)
	agoStr := lipgloss.NewStyle().Width(4).Foreground(tokenFgMuted).Render(ago)

	stCol := lipgloss.NewStyle().Width(2).Render(icon)
	typeCol := lipgloss.NewStyle().Width(typeColWidth).Render(tStr)
	nameCol := nameStyle.Render(name)

	row := indicator + stCol + typeCol + nameCol + " " + agoStr

	if selected {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("#1e3a5f")).
			Width(width).
			Render(row)
	}
	if s.Status == model.StatusNeedsAttention {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("#2d1b1b")).
			Width(width).
			Render(row)
	}
	return lipgloss.NewStyle().Width(width).Render(row)
}

func typeTag(t model.ProcessType) string {
	switch t {
	case model.ProcessClaude:
		return lipgloss.NewStyle().Foreground(colorMagenta).Bold(true).Render("Claude")
	case model.ProcessKiro:
		return lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("Kiro")
	case model.ProcessCodex:
		return lipgloss.NewStyle().Foreground(colorBlue).Bold(true).Render("Codex")
	case model.ProcessOpencode:
		return lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("Open")
	}
	return lipgloss.NewStyle().Foreground(tokenFgMuted).Render("?")
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// ── Output tab ──────────────────────────────────────────────────────

func renderOutputTab(a App, width, height int) string {
	panelWidth := width - 4

	var selected *model.SessionInfo
	filtered := a.filteredSessions()
	if a.selectedIdx < len(filtered) {
		s := filtered[a.selectedIdx]
		selected = &s
	}

	if selected == nil || selected.PaneContent == "" {
		empty := lipgloss.NewStyle().Foreground(tokenFgMuted).Render("\n  No pane content available")
		return activePanelBorder.Width(panelWidth).Height(height).Render(empty)
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).
		Render(selected.DisplayName()) + "  " + statusBadge(selected.Status)

	paneLines := strings.Split(selected.PaneContent, "\n")
	for len(paneLines) > 0 && strings.TrimSpace(paneLines[len(paneLines)-1]) == "" {
		paneLines = paneLines[:len(paneLines)-1]
	}

	previewHeight := height - 4
	if previewHeight < 3 {
		previewHeight = 3
	}
	start := len(paneLines) - previewHeight
	if start < 0 {
		start = 0
	}
	preview := paneLines[start:]

	maxLineWidth := panelWidth - 4
	var colored []string
	for _, pl := range preview {
		pl = strings.ReplaceAll(pl, "\r", "")
		pl = strings.ReplaceAll(pl, "\t", "    ")
		if runes := []rune(pl); len(runes) > maxLineWidth {
			pl = string(runes[:maxLineWidth])
		}
		colored = append(colored, colorPreviewLine(pl))
	}

	help := helpStyle.Render("j/k:navigate  ⏎:switch  y:yes  a:auto  !:esc  i:insert  ?:help")
	content := title + "\n\n" + strings.Join(colored, "\n") + "\n\n" + help
	return activePanelBorder.Width(panelWidth).Height(height).Render(content)
}

// ── Detail panel ────────────────────────────────────────────────────

func renderDetail(s *model.SessionInfo, height int, insertMode bool) string {
	if s == nil {
		return panelHeadingStyle.Render("Overview") + "\n" +
			lipgloss.NewStyle().Foreground(tokenFgMuted).
				Render("\n  No session selected")
	}

	typeName := "Claude Code"
	if s.ProcessType == model.ProcessKiro {
		typeName = "Kiro CLI"
	} else if s.ProcessType == model.ProcessCodex {
		typeName = "Codex"
	} else if s.ProcessType == model.ProcessOpencode {
		typeName = "Opencode"
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(tokenFgDefault).
		Render(s.DisplayName())
	lines := []string{
		panelHeadingStyle.Render("Overview"),
		"",
		title + "  " + statusBadge(s.Status),
		"",
	}

	lines = append(lines,
		labelStyle.Render("Type:")+" "+lipgloss.NewStyle().Foreground(tokenFgDefault).Render(typeName),
	)
	if s.Cwd != "" {
		lines = append(lines, labelStyle.Render("Dir:")+" "+lipgloss.NewStyle().Foreground(tokenFgDefault).Render(s.Cwd))
	}
	if s.GitBranch != "" {
		lines = append(lines, labelStyle.Render("Branch:")+" "+
			lipgloss.NewStyle().Foreground(colorMagenta).Render("⎇ "+sanitizeField(s.GitBranch)))
	}

	lines = append(lines, "", divider(40))

	if s.ContextPct > 0 || s.ContextTokens > 0 {
		lines = append(lines, "")
		lines = append(lines,
			labelStyle.Render("Context:")+" "+renderContextBar(s.ContextPct, 24))
		lines = append(lines,
			strings.Repeat(" ", 10)+
				lipgloss.NewStyle().Foreground(tokenFgMuted).
					Render(fmt.Sprintf("%d / %d tokens", s.ContextTokens, s.ContextMax)))
	}

	if s.CostUSD > 0 {
		costStr := lipgloss.NewStyle().Foreground(tokenFgWarning).Bold(true).
			Render(fmt.Sprintf("$%.2f", s.CostUSD))
		spark := sparkline(s.CostHistory)
		if spark != "" {
			costStr += "  " + spark
		}
		lines = append(lines, labelStyle.Render("Cost:")+" "+costStr)
	}

	if s.LastActivity != "" {
		activity := sanitizeField(s.LastActivity)
		if len(activity) > 80 {
			activity = activity[:80] + "…"
		}
		lines = append(lines, "",
			lipgloss.NewStyle().Foreground(tokenFgMuted).Italic(true).
				Render("Last: "+activity))
	}

	if s.TaskSummary != "" {
		task := sanitizeField(s.TaskSummary)
		if len(task) > 76 {
			task = task[:76] + "…"
		}
		lines = append(lines, "",
			lipgloss.NewStyle().Foreground(tokenFgAccent).Bold(true).Render("Task: ")+
				lipgloss.NewStyle().Foreground(tokenFgDefault).Render(task))
	}

	// Pane preview
	if s.PaneContent != "" {
		lines = append(lines, "", divider(40))

		paneLines := strings.Split(s.PaneContent, "\n")
		for len(paneLines) > 0 && strings.TrimSpace(paneLines[len(paneLines)-1]) == "" {
			paneLines = paneLines[:len(paneLines)-1]
		}
		previewHeight := height - len(lines) - 6
		if previewHeight < 3 {
			previewHeight = 3
		}
		if previewHeight > 20 {
			previewHeight = 20
		}
		start := len(paneLines) - previewHeight
		if start < 0 {
			start = 0
		}
		preview := paneLines[start:]

		var colored []string
		for _, pl := range preview {
			pl = strings.ReplaceAll(pl, "\r", "")
			pl = strings.ReplaceAll(pl, "\t", "    ")
			if runes := []rune(pl); len(runes) > 76 {
				pl = string(runes[:76])
			}
			colored = append(colored, colorPreviewLine(pl))
		}
		for len(colored) < previewHeight {
			colored = append(colored, "")
		}

		boxStyle := previewBorder
		boxLabel := lipgloss.NewStyle().Foreground(tokenFgMuted).Render(" Preview ")
		if insertMode {
			boxStyle = boxStyle.BorderForeground(colorYellow)
			boxLabel = lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(" INSERT ")
		}
		box := boxStyle.Width(78).Height(previewHeight).
			Render(strings.Join(colored, "\n"))

		lines = append(lines, boxLabel, box)
	}

	// Help / insert mode indicator
	if insertMode {
		lines = append(lines, "",
			lipgloss.NewStyle().Bold(true).Foreground(colorYellow).
				Render("── INSERT ──  type to send to pane  Esc: exit"))
	} else {
		parts := []string{"i:insert", "y/a/!:respond", "j/k:nav", "⏎:switch", "?:help", "1-3:tabs", "q:quit"}
		lines = append(lines, "", helpStyle.Render(strings.Join(parts, "  ")))
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// colorPreviewLine applies syntax-aware dimming to a preview line.
func colorPreviewLine(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.Contains(line, "❯"):
		return lipgloss.NewStyle().Foreground(tokenFgSuccess).Render(line)
	case attentionRe.MatchString(line):
		return lipgloss.NewStyle().Foreground(tokenFgDanger).Bold(true).Render(line)
	case strings.HasPrefix(trimmed, "│") || strings.HasPrefix(trimmed, "├") || strings.HasPrefix(trimmed, "└"):
		return lipgloss.NewStyle().Foreground(tokenFgMuted).Render(line)
	case strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "✔"):
		return lipgloss.NewStyle().Foreground(tokenFgSuccess).Render(line)
	case strings.HasPrefix(trimmed, "✗") || strings.HasPrefix(trimmed, "✘") || strings.HasPrefix(trimmed, "error"):
		return lipgloss.NewStyle().Foreground(tokenFgDanger).Render(line)
	case strings.HasPrefix(trimmed, "λ") || strings.HasPrefix(trimmed, ">"):
		return lipgloss.NewStyle().Foreground(tokenFgAccent).Render(line)
	}
	return lipgloss.NewStyle().Foreground(tokenFgDefault).Render(line)
}

// ── Help overlay ────────────────────────────────────────────────────

func renderHelpOverlay(width, height int) string {
	col1 := []string{
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Global"),
		kv("q / Ctrl+C", "quit"),
		kv("1 / 2 / 3", "switch tabs"),
		kv("Tab", "next tab"),
		kv("?", "toggle help"),
		kv("r", "refresh sessions"),
		kv("n", "toggle notify"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Session Actions"),
		kv("j / k", "navigate"),
		kv("g / G", "top / bottom"),
		kv("Enter", "switch to pane"),
		kv("y", "send Enter (yes)"),
		kv("a", "send 'a' (auto)"),
		kv("!", "send Escape *"),
		kv("i", "insert mode"),
	}
	col2 := []string{
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Filter & Sort"),
		kv("/", "open filter"),
		kv("Esc", "close filter (keep)"),
		kv("s", "cycle sort order"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Todos (tab 2)"),
		kv("j / k", "navigate"),
		kv("i", "add new todo"),
		kv("Enter", "mark done"),
		kv("d / x", "delete"),
		kv("Esc", "exit insert"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Insert Mode"),
		kv("type", "send key to pane"),
		kv("Enter", "send Enter"),
		kv("Esc", "exit insert"),
		lipgloss.NewStyle().Foreground(tokenFgMuted).Render("* requires confirmation"),
	}

	maxRows := len(col1)
	if len(col2) > maxRows {
		maxRows = len(col2)
	}

	colW := 32
	var rows []string
	for i := 0; i < maxRows; i++ {
		left := ""
		if i < len(col1) {
			left = col1[i]
		}
		right := ""
		if i < len(col2) {
			right = col2[i]
		}
		leftPadded := lipgloss.NewStyle().Width(colW).Render(left)
		rows = append(rows, leftPadded+"  "+right)
	}

	content := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tokenBorderFocused).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("  Help  ") + "\n\n" +
				strings.Join(rows, "\n") + "\n\n" +
				lipgloss.NewStyle().Foreground(tokenFgMuted).Render("? or Esc to close"),
		)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func kv(key, desc string) string {
	k := lipgloss.NewStyle().Foreground(tokenFgWarning).Bold(true).Render(fmt.Sprintf("%-12s", key))
	d := lipgloss.NewStyle().Foreground(tokenFgDefault).Render(desc)
	return k + " " + d
}

// ── Confirm modal ────────────────────────────────────────────────────

func renderConfirmModal(msg string, width, height int) string {
	line1 := lipgloss.NewStyle().Foreground(tokenFgDefault).Render(msg)
	line2 := lipgloss.NewStyle().Foreground(tokenFgMuted).
		Render("    [y] confirm   [Esc / n] cancel")

	content := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tokenFgDanger).
		Padding(1, 3).
		Render(line1 + "\n\n" + line2)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// ── Todos ───────────────────────────────────────────────────────────

func renderTodos(s *model.SessionInfo, shared *model.ProjectTodos, internal []model.TodoItem) string {
	projectName := s.ProjectName
	if projectName == "" {
		projectName = s.DisplayName()
	}

	hasShared := shared != nil && len(shared.Items) > 0
	hasInternal := len(internal) > 0

	if !hasShared && !hasInternal {
		return lipgloss.NewStyle().Foreground(tokenFgMuted).
			Render(fmt.Sprintf("Todos: none for %s", projectName))
	}

	var lines []string
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgDefault).Render("Todos")+
			lipgloss.NewStyle().Foreground(tokenFgMuted).Render(fmt.Sprintf(" (%s)", projectName)))

	if hasShared {
		for _, item := range shared.Items {
			icon := todoIcon(item.Status)
			text := item.Text
			if len(text) > 55 {
				text = text[:55]
			}
			switch item.Status {
			case model.TodoDone:
				text = lipgloss.NewStyle().Foreground(tokenFgMuted).Strikethrough(true).Render(text)
			case model.TodoInProgress:
				text = lipgloss.NewStyle().Foreground(tokenFgAccent).Render(text)
			default:
				text = lipgloss.NewStyle().Foreground(tokenFgDefault).Render(text)
			}
			tags := ""
			if len(item.Tags) > 0 {
				tags = lipgloss.NewStyle().Foreground(tokenFgMuted).
					Render(" [" + strings.Join(item.Tags, ", ") + "]")
			}
			lines = append(lines, fmt.Sprintf("  %s %s%s", icon, text, tags))
		}
	}

	if hasInternal {
		if hasShared {
			lines = append(lines, "")
		}
		lines = append(lines,
			lipgloss.NewStyle().Foreground(tokenFgMuted).Italic(true).
				Render("  Session internal todos:"))
		for _, item := range internal {
			icon := todoIcon(item.Status)
			text := item.Text
			if len(text) > 55 {
				text = text[:55]
			}
			lines = append(lines, fmt.Sprintf("  %s %s",
				icon, lipgloss.NewStyle().Foreground(tokenFgMuted).Render(text)))
		}
	}

	return strings.Join(lines, "\n")
}
