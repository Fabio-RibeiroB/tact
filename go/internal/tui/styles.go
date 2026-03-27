package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fabiobrady/tact/internal/model"
)

const defaultThemeName = "tokyo-night"
const defaultStyleName = "card-stack"

type themeDefinition struct {
	Name string

	bg              lipgloss.Color
	surface         lipgloss.Color
	border          lipgloss.Color
	borderHi        lipgloss.Color
	text            lipgloss.Color
	dim             lipgloss.Color
	success         lipgloss.Color
	info            lipgloss.Color
	danger          lipgloss.Color
	warning         lipgloss.Color
	magenta         lipgloss.Color
	accent          lipgloss.Color
	orange          lipgloss.Color
	selectedBg      lipgloss.Color
	attentionBg     lipgloss.Color
	headerBg        lipgloss.Color
	tabBarBg        lipgloss.Color
	tooSmall        lipgloss.Color
	contextGradient []string
}

type styleDefinition struct {
	Name string

	displayName  string
	title        string
	panelBorder  lipgloss.Border
	activeBorder lipgloss.Border
	previewBorder lipgloss.Border
	helpBorder    lipgloss.Border
	confirmBorder lipgloss.Border
	dividerRune   string
	tabOpen       string
	tabClose      string
	rowPrefix     string
	selectedPrefix string
	cardPadding    int
}

var themes = []themeDefinition{
	{
		Name:            "tokyo-night",
		bg:              lipgloss.Color("#1a1b26"),
		surface:         lipgloss.Color("#24283b"),
		border:          lipgloss.Color("#3b4261"),
		borderHi:        lipgloss.Color("#7aa2f7"),
		text:            lipgloss.Color("#c0caf5"),
		dim:             lipgloss.Color("#565f89"),
		success:         lipgloss.Color("#9ece6a"),
		info:            lipgloss.Color("#7aa2f7"),
		danger:          lipgloss.Color("#f7768e"),
		warning:         lipgloss.Color("#e0af68"),
		magenta:         lipgloss.Color("#bb9af7"),
		accent:          lipgloss.Color("#7dcfff"),
		orange:          lipgloss.Color("#ff9e64"),
		selectedBg:      lipgloss.Color("#1e3a5f"),
		attentionBg:     lipgloss.Color("#2d1b1b"),
		headerBg:        lipgloss.Color("#1f2335"),
		tabBarBg:        lipgloss.Color("#1a1b26"),
		tooSmall:        lipgloss.Color("#f7768e"),
		contextGradient: []string{"#9ece6a", "#b5e06a", "#d4c96a", "#e0af68", "#f7768e"},
	},
	{
		Name:            "sunset-grid",
		bg:              lipgloss.Color("#1d1411"),
		surface:         lipgloss.Color("#2b1d18"),
		border:          lipgloss.Color("#6a4a3c"),
		borderHi:        lipgloss.Color("#ff9b54"),
		text:            lipgloss.Color("#ffe7d1"),
		dim:             lipgloss.Color("#b68d78"),
		success:         lipgloss.Color("#c7d36f"),
		info:            lipgloss.Color("#ff9b54"),
		danger:          lipgloss.Color("#ff6b6b"),
		warning:         lipgloss.Color("#ffd166"),
		magenta:         lipgloss.Color("#f4a6ff"),
		accent:          lipgloss.Color("#58d6c5"),
		orange:          lipgloss.Color("#ff7f50"),
		selectedBg:      lipgloss.Color("#55342a"),
		attentionBg:     lipgloss.Color("#4a2022"),
		headerBg:        lipgloss.Color("#35231d"),
		tabBarBg:        lipgloss.Color("#241915"),
		tooSmall:        lipgloss.Color("#ff6b6b"),
		contextGradient: []string{"#58d6c5", "#c7d36f", "#ffd166", "#ff9b54", "#ff6b6b"},
	},
	{
		Name:            "arctic-pulse",
		bg:              lipgloss.Color("#0f1c24"),
		surface:         lipgloss.Color("#162933"),
		border:          lipgloss.Color("#315465"),
		borderHi:        lipgloss.Color("#71e7ff"),
		text:            lipgloss.Color("#dff8ff"),
		dim:             lipgloss.Color("#7ea2b1"),
		success:         lipgloss.Color("#96f7c1"),
		info:            lipgloss.Color("#71e7ff"),
		danger:          lipgloss.Color("#ff7f9f"),
		warning:         lipgloss.Color("#ffe28a"),
		magenta:         lipgloss.Color("#d0a7ff"),
		accent:          lipgloss.Color("#8df0ff"),
		orange:          lipgloss.Color("#ffb86c"),
		selectedBg:      lipgloss.Color("#163b4a"),
		attentionBg:     lipgloss.Color("#412534"),
		headerBg:        lipgloss.Color("#13242d"),
		tabBarBg:        lipgloss.Color("#101c23"),
		tooSmall:        lipgloss.Color("#ff7f9f"),
		contextGradient: []string{"#96f7c1", "#71e7ff", "#8df0ff", "#ffe28a", "#ff7f9f"},
	},
}

var styles = []styleDefinition{
	{
		Name:          "card-stack",
		displayName:   "Card Stack",
		title:         "TACT",
		panelBorder:   lipgloss.RoundedBorder(),
		activeBorder:  lipgloss.RoundedBorder(),
		previewBorder: lipgloss.RoundedBorder(),
		helpBorder:    lipgloss.RoundedBorder(),
		confirmBorder: lipgloss.RoundedBorder(),
		dividerRune:   "─",
		tabOpen:       "‹",
		tabClose:      "›",
		rowPrefix:     "  ",
		selectedPrefix:"▶ ",
		cardPadding:   1,
	},
	{
		Name:          "retro-bracket",
		displayName:   "Retro Bracket",
		title:         "[ TACT ]",
		panelBorder: lipgloss.Border{
			Top: "-", Bottom: "-", Left: "|", Right: "|",
			TopLeft: "[", TopRight: "]", BottomLeft: "[", BottomRight: "]",
		},
		activeBorder: lipgloss.Border{
			Top: "=", Bottom: "=", Left: "|", Right: "|",
			TopLeft: "[", TopRight: "]", BottomLeft: "[", BottomRight: "]",
		},
		previewBorder: lipgloss.Border{
			Top: "-", Bottom: "-", Left: "|", Right: "|",
			TopLeft: "[", TopRight: "]", BottomLeft: "[", BottomRight: "]",
		},
		helpBorder: lipgloss.Border{
			Top: "=", Bottom: "=", Left: "|", Right: "|",
			TopLeft: "[", TopRight: "]", BottomLeft: "[", BottomRight: "]",
		},
		confirmBorder: lipgloss.Border{
			Top: "=", Bottom: "=", Left: "|", Right: "|",
			TopLeft: "[", TopRight: "]", BottomLeft: "[", BottomRight: "]",
		},
		dividerRune:   "=",
		tabOpen:       "[",
		tabClose:      "]",
		rowPrefix:     "  ",
		selectedPrefix:"> ",
		cardPadding:   0,
	},
	{
		Name:          "signal-grid",
		displayName:   "Signal Grid",
		title:         "TACT//GRID",
		panelBorder:   lipgloss.NormalBorder(),
		activeBorder:  lipgloss.ThickBorder(),
		previewBorder: lipgloss.NormalBorder(),
		helpBorder:    lipgloss.ThickBorder(),
		confirmBorder: lipgloss.ThickBorder(),
		dividerRune:   "━",
		tabOpen:       "⟦",
		tabClose:      "⟧",
		rowPrefix:     "│ ",
		selectedPrefix:"┃ ",
		cardPadding:   0,
	},
}

var (
	currentTheme themeDefinition
	currentStyle styleDefinition

	colorBg       lipgloss.Color
	colorSurface  lipgloss.Color
	colorBorder   lipgloss.Color
	colorBorderHi lipgloss.Color
	colorText     lipgloss.Color
	colorDim      lipgloss.Color
	colorGreen    lipgloss.Color
	colorBlue     lipgloss.Color
	colorRed      lipgloss.Color
	colorYellow   lipgloss.Color
	colorMagenta  lipgloss.Color
	colorCyan     lipgloss.Color
	colorOrange   lipgloss.Color

	tokenFgDefault      lipgloss.Color
	tokenFgMuted        lipgloss.Color
	tokenFgDanger       lipgloss.Color
	tokenFgWarning      lipgloss.Color
	tokenFgSuccess      lipgloss.Color
	tokenFgInfo         lipgloss.Color
	tokenFgAccent       lipgloss.Color
	tokenBgSelected     lipgloss.Color
	tokenBgAttention    lipgloss.Color
	tokenBgSurface      lipgloss.Color
	tokenBgHeader       lipgloss.Color
	tokenBgTabBar       lipgloss.Color
	tokenFgTooSmall     lipgloss.Color
	tokenBorderFocused  lipgloss.Color
	tokenBorderInactive lipgloss.Color

	panelBorder       lipgloss.Style
	activePanelBorder lipgloss.Style
	labelStyle        lipgloss.Style
	helpStyle         lipgloss.Style
	panelHeadingStyle lipgloss.Style
	previewBorder     lipgloss.Style
	dividerStyle      lipgloss.Style

	gradientColors []string
	appStyle       lipgloss.Style
)

func init() {
	applyTheme(themeByName(defaultThemeName))
	applyStyle(styleByName(defaultStyleName))
}

func themeByName(name string) themeDefinition {
	for _, theme := range themes {
		if theme.Name == name {
			return theme
		}
	}
	return themes[0]
}

func normalizeThemeName(name string) string {
	return themeByName(name).Name
}

func themeDisplayName(name string) string {
	switch normalizeThemeName(name) {
	case "sunset-grid":
		return "Sunset Grid"
	case "arctic-pulse":
		return "Arctic Pulse"
	default:
		return "Tokyo Night"
	}
}

func styleByName(name string) styleDefinition {
	for _, style := range styles {
		if style.Name == name {
			return style
		}
	}
	return styles[0]
}

func normalizeStyleName(name string) string {
	return styleByName(name).Name
}

func styleDisplayName(name string) string {
	return styleByName(name).displayName
}

func nextStyleName(name string) string {
	name = normalizeStyleName(name)
	for i, style := range styles {
		if style.Name == name {
			return styles[(i+1)%len(styles)].Name
		}
	}
	return styles[0].Name
}

func applyStyleByName(name string) string {
	style := styleByName(name)
	applyStyle(style)
	return style.Name
}

func nextThemeName(name string) string {
	name = normalizeThemeName(name)
	for i, theme := range themes {
		if theme.Name == name {
			return themes[(i+1)%len(themes)].Name
		}
	}
	return themes[0].Name
}

func applyThemeByName(name string) string {
	theme := themeByName(name)
	applyTheme(theme)
	return theme.Name
}

func applyTheme(theme themeDefinition) {
	currentTheme = theme

	colorBg = theme.bg
	colorSurface = theme.surface
	colorBorder = theme.border
	colorBorderHi = theme.borderHi
	colorText = theme.text
	colorDim = theme.dim
	colorGreen = theme.success
	colorBlue = theme.info
	colorRed = theme.danger
	colorYellow = theme.warning
	colorMagenta = theme.magenta
	colorCyan = theme.accent
	colorOrange = theme.orange

	tokenFgDefault = colorText
	tokenFgMuted = colorDim
	tokenFgDanger = colorRed
	tokenFgWarning = colorYellow
	tokenFgSuccess = colorGreen
	tokenFgInfo = colorBlue
	tokenFgAccent = colorCyan
	tokenBgSelected = theme.selectedBg
	tokenBgAttention = theme.attentionBg
	tokenBgSurface = colorSurface
	tokenBgHeader = theme.headerBg
	tokenBgTabBar = theme.tabBarBg
	tokenFgTooSmall = theme.tooSmall
	tokenBorderFocused = colorBorderHi
	tokenBorderInactive = colorBorder

	panelBorder = lipgloss.NewStyle().
		Border(currentStyle.panelBorder).
		BorderForeground(colorBorder).
		Background(colorSurface)

	activePanelBorder = lipgloss.NewStyle().
		Border(currentStyle.activeBorder).
		BorderForeground(colorBorderHi).
		Background(colorSurface)

	labelStyle = lipgloss.NewStyle().Foreground(colorDim).Width(10)
	helpStyle = lipgloss.NewStyle().Foreground(colorDim)
	panelHeadingStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorText)
	previewBorder = lipgloss.NewStyle().
		Border(currentStyle.previewBorder).
		BorderForeground(colorBorder).
		Background(colorSurface)
	dividerStyle = lipgloss.NewStyle().Foreground(colorBorder)
	gradientColors = theme.contextGradient
	appStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Background(colorBg)
}

func applyStyle(style styleDefinition) {
	currentStyle = style
	applyTheme(currentTheme)
}

var workingSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func statusIcon(s model.SessionStatus, blinkOn bool, spinnerIdx int) string {
	switch s {
	case model.StatusIdle:
		return lipgloss.NewStyle().Foreground(colorGreen).Render("●")
	case model.StatusWorking:
		frame := workingSpinner[spinnerIdx%len(workingSpinner)]
		return lipgloss.NewStyle().Foreground(colorBlue).Render(frame)
	case model.StatusNeedsAttention:
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

func headerPill(label, value string, fg lipgloss.Color) string {
	if currentStyle.Name == "retro-bracket" {
		content := value
		if label != "" {
			content = label + " " + value
		}
		return lipgloss.NewStyle().
			Foreground(fg).
			Render("[ " + content + " ]")
	}
	l := lipgloss.NewStyle().Foreground(colorDim).Render(label)
	v := lipgloss.NewStyle().Foreground(fg).Bold(true).Render(value)
	content := l + v
	if label != "" {
		content = l + " " + v
	}
	return lipgloss.NewStyle().
		Background(colorSurface).
		BorderStyle(currentStyle.panelBorder).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Render(content)
}

func divider(width int) string {
	return dividerStyle.Render(strings.Repeat(currentStyle.dividerRune, width))
}

func appTitle() string {
	return currentStyle.title
}

func styleTab(key string) string {
	return currentStyle.tabOpen + key + currentStyle.tabClose
}

func isRetroStyle() bool {
	return currentStyle.Name == "retro-bracket"
}

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
