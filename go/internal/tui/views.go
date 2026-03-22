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
		Foreground(colorCyan).
		Render("⚡ TACT")

	sessionPill := headerPill("Sessions", fmt.Sprintf("%d", len(sessions)), colorText)

	var attnPill string
	if attn > 0 {
		attnPill = headerPill("", fmt.Sprintf("⚠ %d NEED ATTENTION", attn), colorRed)
	} else {
		attnPill = headerPill("", "✓ all clear", colorGreen)
	}

	costPill := headerPill("Today", fmt.Sprintf("$%.2f", totalCost), colorYellow)
	clockPill := headerPill("", time.Now().Format("15:04:05"), colorDim)

	notifyStr := lipgloss.NewStyle().Foreground(colorGreen).Render("🔔")
	if !notifyEnabled {
		notifyStr = lipgloss.NewStyle().Foreground(colorDim).Render("🔕")
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

	// Summary line: what the selected session is doing + working count + todo hint
	var summaryParts []string
	if working > 0 {
		summaryParts = append(summaryParts,
			lipgloss.NewStyle().Foreground(colorBlue).Render(fmt.Sprintf("⟳ %d working", working)))
	}
	if selected != nil && selected.TaskSummary != "" {
		activity := selected.TaskSummary
		maxLen := width - 40
		if maxLen < 30 {
			maxLen = 30
		}
		if len(activity) > maxLen {
			activity = activity[:maxLen] + "…"
		}
		label := selected.DisplayName() + ": "
		summaryParts = append(summaryParts,
			lipgloss.NewStyle().Foreground(colorDim).Render(label)+
				lipgloss.NewStyle().Foreground(colorText).Italic(true).Render(activity))
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

// ── Session list row (two-line with left-border selection) ──────────

func renderSessionRow(s model.SessionInfo, selected bool, blinkOn bool, spinnerIdx int, width int) string {
	icon := statusIcon(s.Status, blinkOn, spinnerIdx)
	tIcon := typeIcon(s.ProcessType)

	name := s.DisplayName()
	if len(name) > 20 {
		name = name[:20]
	}

	line1 := fmt.Sprintf("%s %s %s",
		icon, tIcon,
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(name))

	// Second line: metadata
	var meta []string
	if s.GitBranch != "" {
		meta = append(meta, lipgloss.NewStyle().Foreground(colorDim).Render("⎇ "+s.GitBranch))
	}
	if s.ContextPct > 0 {
		pctColor := colorGreen
		if s.ContextPct >= 80 {
			pctColor = colorRed
		} else if s.ContextPct >= 60 {
			pctColor = colorYellow
		}
		meta = append(meta, lipgloss.NewStyle().Foreground(pctColor).Render(fmt.Sprintf("%d%%", s.ContextPct)))
	}
	if s.CostUSD > 0 {
		costStr := lipgloss.NewStyle().Foreground(colorYellow).Render(fmt.Sprintf("$%.2f", s.CostUSD))
		spark := sparkline(s.CostHistory)
		if spark != "" {
			costStr += " " + spark
		}
		meta = append(meta, costStr)
	}

	line2 := ""
	if len(meta) > 0 {
		line2 = lipgloss.NewStyle().PaddingLeft(4).Foreground(colorDim).
			Render(strings.Join(meta, "  "))
	}

	content := line1
	if line2 != "" {
		content += "\n" + line2
	}
	if s.TaskSummary != "" {
		task := s.TaskSummary
		maxLen := width - 6
		if maxLen < 10 {
			maxLen = 10
		}
		if len(task) > maxLen {
			task = task[:maxLen] + "…"
		}
		content += "\n" + lipgloss.NewStyle().PaddingLeft(4).Foreground(colorDim).Italic(true).
			Render("» "+task)
	}

	if selected {
		return lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(colorBlue).
			PaddingLeft(1).
			Width(width).
			Render(content)
	}
	return lipgloss.NewStyle().PaddingLeft(3).Width(width).Render(content)
}

// ── Detail panel ────────────────────────────────────────────────────

func renderDetail(s *model.SessionInfo, height int, insertMode bool) string {
	if s == nil {
		return lipgloss.NewStyle().Foreground(colorDim).
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

	// Title + badge
	title := lipgloss.NewStyle().Bold(true).Foreground(colorText).
		Render(s.DisplayName())
	lines := []string{
		title + "  " + statusBadge(s.Status),
		"",
	}

	// Metadata
	lines = append(lines,
		labelStyle.Render("Type:")+" "+lipgloss.NewStyle().Foreground(colorText).Render(typeName),
	)
	if s.Cwd != "" {
		lines = append(lines, labelStyle.Render("Dir:")+" "+lipgloss.NewStyle().Foreground(colorText).Render(s.Cwd))
	}
	if s.GitBranch != "" {
		lines = append(lines, labelStyle.Render("Branch:")+" "+
			lipgloss.NewStyle().Foreground(colorMagenta).Render("⎇ "+s.GitBranch))
	}

	// Divider + context bar
	lines = append(lines, "", divider(40))

	if s.ContextPct > 0 || s.ContextTokens > 0 {
		lines = append(lines, "")
		lines = append(lines,
			labelStyle.Render("Context:")+" "+renderContextBar(s.ContextPct, 24))
		lines = append(lines,
			strings.Repeat(" ", 10)+
				lipgloss.NewStyle().Foreground(colorDim).
					Render(fmt.Sprintf("%d / %d tokens", s.ContextTokens, s.ContextMax)))
	}

	if s.CostUSD > 0 {
		costStr := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).
			Render(fmt.Sprintf("$%.2f", s.CostUSD))
		spark := sparkline(s.CostHistory)
		if spark != "" {
			costStr += "  " + spark
		}
		lines = append(lines, labelStyle.Render("Cost:")+" "+costStr)
	}

	if s.LastActivity != "" {
		activity := s.LastActivity
		if len(activity) > 80 {
			activity = activity[:80] + "…"
		}
		lines = append(lines, "",
			lipgloss.NewStyle().Foreground(colorDim).Italic(true).
				Render("Last: "+activity))
	}

	// Task summary
	if s.TaskSummary != "" {
		task := s.TaskSummary
		if len(task) > 76 {
			task = task[:76] + "…"
		}
		lines = append(lines, "",
			lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("Task: ")+
				lipgloss.NewStyle().Foreground(colorText).Render(task))
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

		// Syntax-aware line coloring
		var colored []string
		for _, pl := range preview {
			if len(pl) > 76 {
				pl = pl[:76]
			}
			colored = append(colored, colorPreviewLine(pl))
		}
		for len(colored) < previewHeight {
			colored = append(colored, "")
		}

		boxStyle := previewBorder
		boxLabel := lipgloss.NewStyle().Foreground(colorDim).Render(" Preview ")
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
		parts := []string{"i:insert", "y/a/!:respond", "j/k:nav", "⏎:switch", "tab:panels", "n:notify", "r:refresh", "q:quit"}
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
		return lipgloss.NewStyle().Foreground(colorGreen).Render(line)
	case attentionRe.MatchString(line):
		return lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render(line)
	case strings.HasPrefix(trimmed, "│") || strings.HasPrefix(trimmed, "├") || strings.HasPrefix(trimmed, "└"):
		return lipgloss.NewStyle().Foreground(colorDim).Render(line)
	case strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "✔"):
		return lipgloss.NewStyle().Foreground(colorGreen).Render(line)
	case strings.HasPrefix(trimmed, "✗") || strings.HasPrefix(trimmed, "✘") || strings.HasPrefix(trimmed, "error"):
		return lipgloss.NewStyle().Foreground(colorRed).Render(line)
	case strings.HasPrefix(trimmed, "λ") || strings.HasPrefix(trimmed, ">"):
		return lipgloss.NewStyle().Foreground(colorCyan).Render(line)
	}
	return lipgloss.NewStyle().Foreground(colorText).Render(line)
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
		return lipgloss.NewStyle().Foreground(colorDim).
			Render(fmt.Sprintf("Todos: none for %s", projectName))
	}

	var lines []string
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Foreground(colorText).Render("Todos")+
			lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf(" (%s)", projectName)))

	if hasShared {
		for _, item := range shared.Items {
			icon := todoIcon(item.Status)
			text := item.Text
			if len(text) > 55 {
				text = text[:55]
			}
			switch item.Status {
			case model.TodoDone:
				text = lipgloss.NewStyle().Foreground(colorDim).Strikethrough(true).Render(text)
			case model.TodoInProgress:
				text = lipgloss.NewStyle().Foreground(colorCyan).Render(text)
			default:
				text = lipgloss.NewStyle().Foreground(colorText).Render(text)
			}
			tags := ""
			if len(item.Tags) > 0 {
				tags = lipgloss.NewStyle().Foreground(colorDim).
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
			lipgloss.NewStyle().Foreground(colorDim).Italic(true).
				Render("  Session internal todos:"))
		for _, item := range internal {
			icon := todoIcon(item.Status)
			text := item.Text
			if len(text) > 55 {
				text = text[:55]
			}
			lines = append(lines, fmt.Sprintf("  %s %s",
				icon, lipgloss.NewStyle().Foreground(colorDim).Render(text)))
		}
	}

	return strings.Join(lines, "\n")
}
