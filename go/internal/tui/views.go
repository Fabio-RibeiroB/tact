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
		Background(colorBg).
		Foreground(tokenFgTooSmall).
		Padding(1, 2).
		Render(msg)
}

// ── Header ──────────────────────────────────────────────────────────

func renderHeader(sessions []model.SessionInfo, width int, notifyEnabled bool, _ *model.SessionInfo, themeName, styleName string) string {
	attn := 0
	for _, s := range sessions {
		if s.Status == model.StatusNeedsAttention {
			attn++
		}
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(tokenFgAccent).
		Render("⚡ " + appTitle())

	sessionPill := headerPill("Sessions", fmt.Sprintf("%d", len(sessions)), tokenFgDefault)
	themePill := headerPill("Theme", themeDisplayName(themeName), tokenFgAccent)
	stylePill := headerPill("Style", styleDisplayName(styleName), tokenFgInfo)

	var attnPill string
	if attn > 0 {
		attnPill = headerPill("", fmt.Sprintf("⚠ %d NEED ATTENTION", attn), tokenFgDanger)
	} else {
		attnPill = headerPill("", "✓ all clear", tokenFgSuccess)
	}

	clockPill := headerPill("", time.Now().Format("15:04:05"), tokenFgMuted)

	notifyStr := lipgloss.NewStyle().Foreground(tokenFgSuccess).Render("🔔")
	if !notifyEnabled {
		notifyStr = lipgloss.NewStyle().Foreground(tokenFgMuted).Render("🔕")
	}

	if isRetroStyle() {
		left := strings.Join([]string{
			title,
			sessionPill,
			themePill,
			stylePill,
			attnPill,
		}, " ")
		right := lipgloss.NewStyle().Foreground(tokenFgMuted).Render(
			fmt.Sprintf("[%s %s]", notifyStr, time.Now().Format("15:04:05")),
		)
		gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 1
		if gap < 1 {
			gap = 1
		}
		return lipgloss.NewStyle().
			Background(tokenBgHeader).
			Width(width).
			Render(left + strings.Repeat(" ", gap) + right)
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center,
		title, "  ", sessionPill, " ", themePill, " ", stylePill, " ", attnPill)
	right := lipgloss.JoinHorizontal(lipgloss.Center, notifyStr, " ", clockPill)

	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	bar := lipgloss.NewStyle().
		Background(tokenBgHeader).
		Width(width).
		Render(left + strings.Repeat(" ", gap) + right)

	return bar
}

// ── Tab bar ─────────────────────────────────────────────────────────

func renderTabBar(activeTab, width int, insertMode bool) string {
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
		keyStr := lipgloss.NewStyle().Foreground(tokenFgMuted).Render(styleTab(t.key))
		if t.tab == activeTab {
			style := lipgloss.NewStyle().
				Foreground(tokenFgAccent).
				Bold(true).
				Underline(true)
			if insertMode && t.tab == tabOutput {
				style = style.Foreground(colorYellow)
			}
			label := style.Render(t.label)
			parts = append(parts, keyStr+" "+label)
		} else {
			style := lipgloss.NewStyle().Foreground(tokenFgMuted)
			if insertMode && t.tab == tabOutput {
				style = style.Foreground(colorYellow).Bold(true)
			}
			label := style.Render(t.label)
			parts = append(parts, keyStr+" "+label)
		}
	}

	bar := "  " + strings.Join(parts, "   ")
	return lipgloss.NewStyle().
		Background(tokenBgTabBar).
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
		label := "[filter: " + text + "]"
		if isRetroStyle() {
			label = "[ FILTER " + text + " ]"
		}
		badge := lipgloss.NewStyle().Foreground(tokenFgWarning).Render(label)
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
		Background(tokenBgHeader).
		Width(width).
		Render(line)
}

// ── Session table (k9s-style, 1 row per session) ─────────────────────

var sortLabels = []string{"", "↑ status", "↑ age", "↑ name"}

const typeColWidth = 8 // "Claude  " fits in 8 chars

func renderSessionListHeader(width, sortMode int) string {
	switch currentStyle.Name {
	case "card-stack":
		label := "Session Stack"
		if sortMode > 0 && sortMode < len(sortLabels) {
			label = "Session Stack · " + sortLabels[sortMode]
		}
		return lipgloss.NewStyle().
			Background(tokenBgSurface).
			Foreground(tokenFgMuted).
			Bold(true).
			Padding(0, currentStyle.cardPadding).
			Width(width).
			Render(label)
	case "retro-bracket":
		label := "[SESSIONS]"
		if sortMode > 0 && sortMode < len(sortLabels) {
			label += " [SORT " + strings.ToUpper(strings.TrimPrefix(sortLabels[sortMode], "↑ ")) + "]"
		}
		return lipgloss.NewStyle().
			Background(tokenBgSurface).
			Foreground(tokenFgMuted).
			Bold(true).
			Width(width).
			Render(label)
	default:
		return renderSessionTableHeader(width, sortMode)
	}
}

func renderSessionTableHeader(width, sortMode int) string {
	// indicator(2) + TYPE(typeColWidth) + NAME + STATUS(3)
	nameWidth := width - 2 - typeColWidth - 3
	if nameWidth < 8 {
		nameWidth = 8
	}

	nameLabel := "NAME"
	if sortMode > 0 && sortMode < len(sortLabels) {
		nameLabel = "NAME (" + sortLabels[sortMode] + ")"
	}

	dim := lipgloss.NewStyle().Foreground(tokenFgMuted)
	row := dim.Width(2).Render("") +
		dim.Width(typeColWidth).Render("AI") +
		dim.Width(nameWidth).Render(nameLabel) +
		dim.Width(3).Render(" ST")

	return lipgloss.NewStyle().
		Background(tokenBgSurface).
		Width(width).
		Render(row)
}

func renderSessionListRow(s model.SessionInfo, selected, blinkOn bool, spinnerIdx, width int) string {
	switch currentStyle.Name {
	case "card-stack":
		return renderSessionCardRow(s, selected, blinkOn, spinnerIdx, width)
	case "retro-bracket":
		return renderSessionRetroRow(s, selected, blinkOn, spinnerIdx, width)
	default:
		return renderSessionTableRow(s, selected, blinkOn, spinnerIdx, width)
	}
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

	// indicator(2) + TYPE(typeColWidth) + NAME + STATUS(3)
	nameWidth := width - 2 - typeColWidth - 3
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

	statusCol := lipgloss.NewStyle().Width(3).Render(icon)
	typeCol := lipgloss.NewStyle().Width(typeColWidth).Render(tStr)
	nameCol := nameStyle.Render(name)

	row := indicator + typeCol + nameCol + statusCol

	if selected {
		return lipgloss.NewStyle().
			Background(tokenBgSelected).
			Width(width).
			Render(row)
	}
	if s.Status == model.StatusNeedsAttention {
		return lipgloss.NewStyle().
			Background(tokenBgAttention).
			Width(width).
			Render(row)
	}
	return lipgloss.NewStyle().
		Background(tokenBgSurface).
		Width(width).
		Render(row)
}

func renderSessionCardRow(s model.SessionInfo, selected, blinkOn bool, spinnerIdx, width int) string {
	status := statusBadge(s.Status)
	titleWidth := max(8, width-lipgloss.Width(status)-4)
	name := sanitizeField(s.DisplayName())
	if len([]rune(name)) > titleWidth {
		name = string([]rune(name)[:titleWidth-1]) + "…"
	}

	meta := typeTag(s.ProcessType) + "  " + lipgloss.NewStyle().Foreground(tokenFgMuted).Render(formatLastCheck(s.LastPolled))
	if s.GitBranch != "" {
		branch := sanitizeField(s.GitBranch)
		if len([]rune(branch)) > max(6, width-12) {
			branch = string([]rune(branch)[:max(5, width-13)]) + "…"
		}
		meta += "  " + lipgloss.NewStyle().Foreground(colorMagenta).Render("⎇ "+branch)
	}

	rowStyle := lipgloss.NewStyle().
		Background(tokenBgSurface).
		Border(currentStyle.panelBorder).
		BorderForeground(tokenBorderInactive).
		Padding(0, currentStyle.cardPadding).
		Width(width)
	if selected {
		rowStyle = rowStyle.Background(tokenBgSelected).BorderForeground(tokenBorderFocused)
	} else if s.Status == model.StatusNeedsAttention {
		rowStyle = rowStyle.Background(tokenBgAttention).BorderForeground(tokenFgDanger)
	}

	title := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.NewStyle().Foreground(tokenFgAccent).Render(currentStyle.selectedPrefix),
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgDefault).Width(titleWidth).Render(name),
		" ",
		status,
	)
	if !selected {
		title = lipgloss.JoinHorizontal(lipgloss.Center,
			lipgloss.NewStyle().Foreground(tokenFgMuted).Render(currentStyle.rowPrefix),
			lipgloss.NewStyle().Bold(true).Foreground(tokenFgDefault).Width(titleWidth).Render(name),
			" ",
			status,
		)
	}

	iconLine := lipgloss.NewStyle().Foreground(tokenFgMuted).
		Render(statusIcon(s.Status, blinkOn, spinnerIdx) + " " + meta)
	if s.TaskSummary != "" {
		task := sanitizeField(s.TaskSummary)
		if len([]rune(task)) > max(8, width-4) {
			task = string([]rune(task)[:max(7, width-5)]) + "…"
		}
		return rowStyle.Render(title + "\n" + iconLine + "\n" +
			lipgloss.NewStyle().Foreground(tokenFgMuted).Render(task))
	}
	return rowStyle.Render(title + "\n" + iconLine)
}

func renderSessionRetroRow(s model.SessionInfo, selected, blinkOn bool, spinnerIdx, width int) string {
	prefix := currentStyle.rowPrefix
	if selected {
		prefix = currentStyle.selectedPrefix
	}
	status := strings.ToUpper(s.Status.String())
	if s.Status == model.StatusUnknown {
		status = "POLL"
	}
	typeLabel := strings.ToUpper(s.ProcessType.String())
	if typeLabel == "" {
		typeLabel = "AI"
	}
	icon := statusIcon(s.Status, blinkOn, spinnerIdx)
	meta := fmt.Sprintf("[%s|%s] %s ", typeLabel, status, icon)
	nameWidth := max(8, width-lipgloss.Width(prefix)-lipgloss.Width(meta))
	name := truncateRunes(sanitizeField(s.DisplayName()), nameWidth)
	line := prefix + meta + name
	style := lipgloss.NewStyle().Background(tokenBgSurface).Width(width)
	if selected {
		style = style.Background(tokenBgSelected).Foreground(tokenFgAccent).Bold(true)
	} else if s.Status == model.StatusNeedsAttention {
		style = style.Background(tokenBgAttention)
	}
	return style.Render(line)
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
		return lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("OpenCode")
	}
	return lipgloss.NewStyle().Foreground(tokenFgMuted).Render("?")
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		minutes := int(d.Minutes())
		if minutes < 1 {
			return "just now"
		}
		return fmt.Sprintf("%dm", minutes)
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

func formatLastCheck(t time.Time) string {
	if t.IsZero() {
		return "checked --:--"
	}
	return "checked " + t.Format("15:04")
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

	help := helpStyle.Render("j/k:navigate  R:rename  T:theme  S:style  ⏎:switch  ?:help")
	content := title + "\n\n" + strings.Join(colored, "\n") + "\n\n" + help
	return activePanelBorder.Width(panelWidth).Height(height).Render(content)
}

// ── Detail panel ────────────────────────────────────────────────────

func renderDetail(s *model.SessionInfo, width, height int, insertMode bool) string {
	if width < 20 {
		width = 20
	}
	if s == nil {
		return panelHeadingStyle.Render(detailHeading("Overview")) + "\n" +
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
		panelHeadingStyle.Render(detailHeading("Overview")),
		"",
		title + "  " + statusBadge(s.Status),
		"",
	}

	lines = append(lines,
		labelStyle.Render("Type:")+" "+lipgloss.NewStyle().Foreground(tokenFgDefault).Render(typeName),
	)
	if s.CustomName != "" {
		lines = append(lines,
			labelStyle.Render("Name:")+" "+lipgloss.NewStyle().Foreground(tokenFgAccent).Bold(true).Render(s.CustomName))
	}
	if s.Cwd != "" {
		lines = append(lines, labelStyle.Render("Dir:")+" "+lipgloss.NewStyle().Foreground(tokenFgDefault).Render(s.Cwd))
	}
	if s.GitBranch != "" {
		lines = append(lines, labelStyle.Render("Branch:")+" "+
			lipgloss.NewStyle().Foreground(colorMagenta).Render("⎇ "+sanitizeField(s.GitBranch)))
	}

	lines = append(lines, "", divider(min(width, 40)))

	if s.ContextPct > 0 || s.ContextTokens > 0 {
		lines = append(lines, "")
		lines = append(lines,
			labelStyle.Render("Context:")+" "+renderContextBar(s.ContextPct, 24))
		lines = append(lines,
			strings.Repeat(" ", 10)+
				lipgloss.NewStyle().Foreground(tokenFgMuted).
					Render(fmt.Sprintf("%d / %d tokens", s.ContextTokens, s.ContextMax)))
	}

	if s.LastActivity != "" {
		activity := sanitizeField(s.LastActivity)
		if len(activity) > 80 {
			activity = activity[:min(width, 80)] + "…"
		}
		lines = append(lines, "",
			lipgloss.NewStyle().Foreground(tokenFgMuted).Italic(true).
				Render("Last: "+activity))
	}

	if s.TaskSummary != "" {
		task := sanitizeField(s.TaskSummary)
		if len(task) > 76 {
			task = task[:min(width, 76)] + "…"
		}
		lines = append(lines, "",
			lipgloss.NewStyle().Foreground(tokenFgAccent).Bold(true).Render("Task: ")+
				lipgloss.NewStyle().Foreground(tokenFgDefault).Render(task))
	}

	// Pane preview
	if s.PaneContent != "" {
		lines = append(lines, "", divider(min(width, 40)))

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
		previewWidth := max(16, width-2)
		for _, pl := range preview {
			pl = strings.ReplaceAll(pl, "\r", "")
			pl = strings.ReplaceAll(pl, "\t", "    ")
			if runes := []rune(pl); len(runes) > previewWidth {
				pl = string(runes[:previewWidth])
			}
			colored = append(colored, colorPreviewLine(pl))
		}
		for len(colored) < previewHeight {
			colored = append(colored, "")
		}

		boxStyle := previewBorder
		boxLabel := lipgloss.NewStyle().Foreground(tokenFgAccent).Bold(true).Render("Preview")
		if insertMode {
			boxStyle = boxStyle.BorderForeground(colorYellow)
			boxLabel = lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render(" INSERT ")
		}
		if isRetroStyle() {
			lines = append(lines, boxLabel)
			lines = append(lines, colored...)
		} else {
			box := boxStyle.Width(previewWidth).Height(previewHeight).
				Render(strings.Join(colored, "\n"))
			lines = append(lines, boxLabel, box)
		}
	}

	// Help / insert mode indicator
	if insertMode {
		lines = append(lines, "",
			lipgloss.NewStyle().Bold(true).Foreground(colorYellow).
				Render(detailHeading("Insert") + "  type to send to pane  Esc: exit"))
	} else {
		parts := []string{"R:rename", "i:insert", "T:theme", "S:style", "y/a/!:respond", "j/k:nav", "⏎:switch", "?:help", "1-3:tabs", "q:quit"}
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
		kv("R", "rename session"),
		kv("n", "toggle notify"),
		kv("T", "cycle theme"),
		kv("S", "cycle style"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Session Actions"),
		kv("j / k", "navigate"),
		kv("g / G", "top / bottom"),
		kv("Enter", "switch to pane"),
		kv("y", "confirm/continue"),
		kv("t", "send 't' to Kiro"),
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
		Border(currentStyle.helpBorder).
		BorderForeground(tokenBorderFocused).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("  Help  ") + "\n\n" +
				strings.Join(rows, "\n") + "\n\n" +
				lipgloss.NewStyle().Foreground(tokenFgMuted).
					Render(fmt.Sprintf("Theme: %s   Style: %s   ? or Esc to close", themeDisplayName(currentTheme.Name), styleDisplayName(currentStyle.Name))),
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

	style := lipgloss.NewStyle().
		Border(currentStyle.confirmBorder).
		BorderForeground(tokenFgDanger).
		Padding(1, 3)
	if isRetroStyle() {
		style = lipgloss.NewStyle().
			Border(currentStyle.confirmBorder).
			BorderForeground(tokenFgDanger).
			Padding(0, 2)
	}
	content := style.Render(line1 + "\n\n" + line2)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func renderRenameModal(input, baseName string, width, height int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(tokenFgAccent).Render("Rename Session")
	current := lipgloss.NewStyle().Foreground(tokenFgMuted).
		Render("Default: " + baseName)
	entry := lipgloss.NewStyle().
		Border(currentStyle.confirmBorder).
		BorderForeground(tokenBorderFocused).
		Padding(0, 1).
		Render(input + "█")
	hint := lipgloss.NewStyle().Foreground(tokenFgMuted).
		Render("Enter: save   Esc: cancel   clear or match default to reset")

	content := lipgloss.NewStyle().
		Border(currentStyle.confirmBorder).
		BorderForeground(tokenBorderFocused).
		Padding(1, 2).
		Render(title + "\n\n" + current + "\n\n" + entry + "\n\n" + hint)

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

func trimToLines(content string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	return strings.Join(lines[:maxLines], "\n")
}

func detailHeading(label string) string {
	switch currentStyle.Name {
	case "retro-bracket":
		return "[" + strings.ToUpper(label) + "]"
	case "signal-grid":
		return strings.ToUpper(label) + " //"
	default:
		return label
	}
}

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string(r[:width-1]) + "…"
}
