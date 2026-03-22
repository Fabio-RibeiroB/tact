package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fabiobrady/tact/internal/model"
)

// ── Tokyo Night palette ─────────────────────────────────────────────
var (
	colorBg       = lipgloss.Color("#1a1b26")
	colorSurface  = lipgloss.Color("#24283b")
	colorBorder   = lipgloss.Color("#3b4261")
	colorBorderHi = lipgloss.Color("#7aa2f7")
	colorText     = lipgloss.Color("#c0caf5")
	colorDim      = lipgloss.Color("#565f89")
	colorGreen    = lipgloss.Color("#9ece6a")
	colorBlue     = lipgloss.Color("#7aa2f7")
	colorRed      = lipgloss.Color("#f7768e")
	colorYellow   = lipgloss.Color("#e0af68")
	colorMagenta  = lipgloss.Color("#bb9af7")
	colorCyan     = lipgloss.Color("#7dcfff")
	colorOrange   = lipgloss.Color("#ff9e64")
)

// ── Semantic design tokens ───────────────────────────────────────────
// Map semantic intent to palette. Change a token here to change it everywhere.
var (
	tokenFgDefault      = colorText
	tokenFgMuted        = colorDim
	tokenFgDanger       = colorRed
	tokenFgWarning      = colorYellow
	tokenFgSuccess      = colorGreen
	tokenFgInfo         = colorBlue
	tokenFgAccent       = colorCyan
	tokenBgSelected     = lipgloss.Color("#2a2c3d")
	tokenBorderFocused  = colorBorderHi
	tokenBorderInactive = colorBorder
)

// ── Reusable styles ─────────────────────────────────────────────────
var (
	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	activePanelBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderHi)

	labelStyle = lipgloss.NewStyle().Foreground(colorDim).Width(10)

	helpStyle = lipgloss.NewStyle().Foreground(colorDim)

	panelHeadingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText)

	previewBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	dividerStyle = lipgloss.NewStyle().Foreground(colorBorder)
)

// ── Status icons ────────────────────────────────────────────────────

var workingSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func statusIcon(s model.SessionStatus, blinkOn bool, spinnerIdx int) string {
	switch s {
	case model.StatusIdle:
		return lipgloss.NewStyle().Foreground(colorGreen).Render("●")
	case model.StatusWorking:
		frame := workingSpinner[spinnerIdx%len(workingSpinner)]
		return lipgloss.NewStyle().Foreground(colorBlue).Render(frame)
	case model.StatusNeedsAttention:
		// Always red — blink between bold ◉ and dim ◉ so it's never invisible
		if blinkOn {
			return lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("◉")
		}
		return lipgloss.NewStyle().Foreground(colorRed).Render("◉")
	case model.StatusDisconnected:
		return lipgloss.NewStyle().Foreground(colorDim).Render("○")
	}
	return lipgloss.NewStyle().Foreground(colorDim).Render("○")
}

func typeIcon(t model.ProcessType) string {
	switch t {
	case model.ProcessClaude:
		return lipgloss.NewStyle().Foreground(colorMagenta).Bold(true).Render("C")
	case model.ProcessKiro:
		return lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("K")
	case model.ProcessCodex:
		return lipgloss.NewStyle().Foreground(colorBlue).Bold(true).Render("X")
	case model.ProcessOpencode:
		return lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("O")
	}
	return lipgloss.NewStyle().Foreground(colorDim).Render("?")
}

// ── Status badge ────────────────────────────────────────────────────

func statusBadge(s model.SessionStatus) string {
	var bg, fg lipgloss.Color
	switch s {
	case model.StatusIdle:
		bg, fg = colorGreen, colorBg
	case model.StatusWorking:
		bg, fg = colorBlue, colorBg
	case model.StatusNeedsAttention:
		bg, fg = colorRed, colorBg
	case model.StatusDisconnected:
		bg, fg = colorBorder, colorDim
	default:
		bg, fg = colorBorder, colorDim
	}
	label := strings.ToUpper(s.String())
	if s == model.StatusUnknown {
		label = "POLLING..."
	}
	return lipgloss.NewStyle().
		Background(bg).Foreground(fg).
		Bold(true).Padding(0, 1).
		Render(label)
}

// ── Gradient context bar ────────────────────────────────────────────

var gradientColors = []string{"#9ece6a", "#b5e06a", "#d4c96a", "#e0af68", "#f7768e"}

func renderContextBar(pct, width int) string {
	filled := pct * width / 100
	if filled > width {
		filled = width
	}
	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			idx := i * (len(gradientColors) - 1) / max(width-1, 1)
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(gradientColors[idx])).
				Render("━"))
		} else {
			b.WriteString(lipgloss.NewStyle().
				Foreground(colorBorder).
				Render("─"))
		}
	}
	pctColor := colorGreen
	if pct >= 80 {
		pctColor = colorRed
	} else if pct >= 60 {
		pctColor = colorYellow
	}
	b.WriteString(" ")
	b.WriteString(lipgloss.NewStyle().Foreground(pctColor).Bold(true).
		Render(fmt.Sprintf("%d%%", pct)))
	return b.String()
}

// ── Sparkline ───────────────────────────────────────────────────────

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func sparkline(values []float64) string {
	if len(values) < 2 {
		return ""
	}
	mx := 0.0
	for _, v := range values {
		mx = math.Max(mx, v)
	}
	if mx == 0 {
		return ""
	}
	var b strings.Builder
	for _, v := range values {
		idx := int(v / mx * float64(len(sparkChars)-1))
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		b.WriteRune(sparkChars[idx])
	}
	return lipgloss.NewStyle().Foreground(colorYellow).Render(b.String())
}

// ── Header pill ─────────────────────────────────────────────────────

func headerPill(label, value string, fg lipgloss.Color) string {
	l := lipgloss.NewStyle().Foreground(colorDim).Render(label)
	v := lipgloss.NewStyle().Foreground(fg).Bold(true).Render(value)
	content := l + v
	if label != "" {
		content = l + " " + v
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1f2335")).
		Padding(0, 1).
		Render(content)
}

// ── Divider ─────────────────────────────────────────────────────────

func divider(width int) string {
	return dividerStyle.Render(strings.Repeat("─", width))
}

// ── Todo icons ──────────────────────────────────────────────────────

func todoIcon(s model.TodoStatus) string {
	switch s {
	case model.TodoPending:
		return "○"
	case model.TodoInProgress:
		return lipgloss.NewStyle().Foreground(colorCyan).Render("◑")
	case model.TodoDone:
		return lipgloss.NewStyle().Foreground(colorDim).Render("●")
	}
	return "?"
}
