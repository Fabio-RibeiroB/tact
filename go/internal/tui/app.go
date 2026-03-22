package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fabiobrady/tact/internal/model"
	"github.com/fabiobrady/tact/internal/notify"
	"github.com/fabiobrady/tact/internal/parser"
	"github.com/fabiobrady/tact/internal/tmux"
	"github.com/fabiobrady/tact/internal/todo"
)

// Messages
type animTickMsg time.Time // fast tick for spinner/blink (~150ms)
type pollTickMsg time.Time // slow tick for pane polling (~2s)
type discoveryMsg []model.SessionInfo
type paneUpdateMsg []model.SessionInfo
type costUpdateMsg []model.SessionInfo
type todoStrikeMsg string // carries todo ID to remove after strike animation

// Focus panels
const (
	panelSessions = iota
	panelTodos
	panelDetail
	panelCount
)

// App is the root Bubble Tea model.
type App struct {
	sessions      []model.SessionInfo
	selectedIdx   int
	width, height int
	focusPanel    int  // panelSessions, panelTodos, panelDetail
	prevStatuses  map[string]model.SessionStatus
	discovering   bool
	notifyEnabled bool

	// Animation state (driven by fast animTick)
	blinkOn    bool
	spinnerIdx int

	// Pane insert mode
	insertMode bool

	// Todo state
	todos        []model.TodoItem // current project's todos
	todoSlug     string           // current project slug
	todoIdx      int              // selected todo index
	todoInput    string           // text being typed in todo insert mode
	todoInsert   bool             // true = typing a new todo
	strikingID   string           // todo ID currently showing strikethrough
}

func newApp() App {
	return App{
		focusPanel:    panelSessions,
		notifyEnabled: true,
		prevStatuses:  make(map[string]model.SessionStatus),
		blinkOn:       true,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		doDiscovery,
		animTick(),
		pollTick(),
	)
}

const animInterval = 150 * time.Millisecond

func animTick() tea.Cmd {
	return tea.Tick(animInterval, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func pollTick() tea.Cmd {
	return tea.Tick(time.Duration(model.PanePollInterval)*time.Second, func(t time.Time) tea.Msg {
		return pollTickMsg(t)
	})
}

func doDiscovery() tea.Msg {
	return discoveryMsg(tmux.DiscoverSessions())
}

func doPaneUpdate(sessions []model.SessionInfo) tea.Cmd {
	return func() tea.Msg {
		for i := range sessions {
			s := &sessions[i]
			content := tmux.CapturePane(s.PaneID)
			title := tmux.GetPaneTitle(s.PaneID)
			if content == "" {
				s.Status = model.StatusDisconnected
				continue
			}
			// Store cleaned content for preview (strip ANSI codes)
			s.PaneContent = ansiRe.ReplaceAllString(content, "")
			clean := s.PaneContent
			s.Status = parser.DetectStatus(clean, title, s.ProcessType)
			if s.ProcessType == model.ProcessClaude {
				cs := parser.ParseClaudeStatusline(clean)
				if cs.CostUSD > 0 {
					s.CostUSD = cs.CostUSD
				}
				if cs.ContextPct > 0 {
					s.ContextPct = cs.ContextPct
					s.ContextTokens = cs.ContextTokens
					s.ContextMax = cs.ContextMax
				}
				if cs.GitBranch != "" {
					s.GitBranch = cs.GitBranch
				}
				if cs.ProjectName != "" {
					s.ProjectName = cs.ProjectName
				}
			}
			if s.ProcessType == model.ProcessKiro {
				if pct := parser.ParseKiroContext(clean); pct > 0 {
					s.ContextPct = pct
				}
			}
		if s.ProcessType == model.ProcessCodex {
			if pct := parser.ParseCodexContext(clean); pct > 0 {
				s.ContextPct = pct
			}
		}
		if s.ProcessType == model.ProcessOpencode {
			if pct := parser.ParseOpencodeContext(clean); pct > 0 {
				s.ContextPct = pct
			}
		}
		if task := parser.ExtractTaskSummary(clean, s.ProcessType); task != "" {
			s.TaskSummary = task
		}
			s.LastChecked = time.Now()
		}
		return paneUpdateMsg(sessions)
	}
}

func doCostUpdate(sessions []model.SessionInfo) tea.Cmd {
	return func() tea.Msg {
		for i := range sessions {
			s := &sessions[i]
			if s.SessionID != "" && s.Cwd != "" {
				data := parser.ParseSessionJSONL(s.SessionID, s.Cwd)
				if data.Cost.TotalUSD > 0 && s.CostUSD == 0 {
					s.CostUSD = data.Cost.TotalUSD
				}
				if data.LastMessage != "" {
					s.LastActivity = data.LastMessage
				}
				if s.TaskSummary == "" {
					if data.FirstHumanMessage != "" {
						s.TaskSummary = data.FirstHumanMessage
					} else if data.LastHumanMessage != "" {
						s.TaskSummary = data.LastHumanMessage
					}
				}
			}
		}
		return costUpdateMsg(sessions)
	}
}

func todoStrikeAfter(id string, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return todoStrikeMsg(id)
	})
}

var (
	pollCount    int
	discoveryDue int
	costDue      int
	ansiRe       = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.todoInsert {
			return a.handleTodoInsertKey(msg)
		}
		if a.insertMode {
			return a.handleInsertKey(msg)
		}
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case animTickMsg:
		a.blinkOn = !a.blinkOn
		a.spinnerIdx++
		return a, animTick()

	case pollTickMsg:
		pollCount++
		var cmds []tea.Cmd
		if len(a.sessions) > 0 && !a.discovering {
			cmds = append(cmds, doPaneUpdate(copySessionSlice(a.sessions)))
		}
		discoveryDue++
		if discoveryDue >= model.DiscoveryPollInterval/model.PanePollInterval {
			discoveryDue = 0
			a.discovering = true
			cmds = append(cmds, doDiscovery)
		}
		costDue++
		if costDue >= model.CostPollInterval/model.PanePollInterval {
			costDue = 0
			if len(a.sessions) > 0 {
				cmds = append(cmds, doCostUpdate(copySessionSlice(a.sessions)))
			}
		}
		cmds = append(cmds, pollTick())
		return a, tea.Batch(cmds...)

	case discoveryMsg:
		a.discovering = false
		a.mergeSessions([]model.SessionInfo(msg))
		if len(a.sessions) > 0 {
			return a, doCostUpdate(copySessionSlice(a.sessions))
		}
		return a, nil

	case paneUpdateMsg:
		a.mergeSessions([]model.SessionInfo(msg))
		a.checkNotifications()
		a.refreshTodos()
		return a, nil

	case costUpdateMsg:
		a.mergeSessions([]model.SessionInfo(msg))
		for i := range a.sessions {
			s := &a.sessions[i]
			if s.CostUSD > 0 {
				s.CostHistory = append(s.CostHistory, s.CostUSD)
				if len(s.CostHistory) > model.MaxCostHistory {
					s.CostHistory = s.CostHistory[len(s.CostHistory)-model.MaxCostHistory:]
				}
			}
		}
		a.refreshTodos()
		return a, nil

	case todoStrikeMsg:
		id := string(msg)
		if a.strikingID == id {
			a.strikingID = ""
			a.removeTodo(id)
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "tab":
		a.focusPanel = (a.focusPanel + 1) % panelCount
		a.refreshTodos()
	case "shift+tab":
		a.focusPanel = (a.focusPanel + panelCount - 1) % panelCount
		a.refreshTodos()
	case "r":
		return *a, doDiscovery
	case "n":
		a.notifyEnabled = !a.notifyEnabled
	}

	// Panel-specific keys
	switch a.focusPanel {
	case panelSessions:
		return a.handleSessionKey(msg)
	case panelTodos:
		return a.handleTodoKey(msg)
	case panelDetail:
		return a.handleDetailKey(msg)
	}
	return *a, nil
}

func (a *App) handleSessionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if a.selectedIdx < len(a.sessions)-1 {
			a.selectedIdx++
			a.refreshTodos()
		}
	case "k", "up":
		if a.selectedIdx > 0 {
			a.selectedIdx--
			a.refreshTodos()
		}
	case "g", "home":
		a.selectedIdx = 0
		a.refreshTodos()
	case "G", "end":
		if len(a.sessions) > 0 {
			a.selectedIdx = len(a.sessions) - 1
			a.refreshTodos()
		}
	case "enter":
		if a.selectedIdx < len(a.sessions) {
			tmux.SwitchToPane(a.sessions[a.selectedIdx].PaneID)
		}
	case "y":
		if s := a.selectedSession(); s != nil && s.Status == model.StatusNeedsAttention {
			tmux.SendKeys(s.PaneID, "Enter")
		}
	case "a":
		if s := a.selectedSession(); s != nil && s.Status == model.StatusNeedsAttention {
			tmux.SendKeys(s.PaneID, "a")
		}
	case "!":
		if s := a.selectedSession(); s != nil && s.Status == model.StatusNeedsAttention {
			tmux.SendKeys(s.PaneID, "Escape")
		}
	case "i":
		if a.selectedSession() != nil {
			a.insertMode = true
		}
	}
	return *a, nil
}

func (a *App) handleTodoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if a.todoIdx < len(a.todos)-1 {
			a.todoIdx++
		}
	case "k", "up":
		if a.todoIdx > 0 {
			a.todoIdx--
		}
	case "i":
		a.todoInsert = true
		a.todoInput = ""
	case "enter":
		// Strike out then dismiss
		if a.todoIdx < len(a.todos) && a.strikingID == "" {
			id := a.todos[a.todoIdx].ID
			a.strikingID = id
			return *a, todoStrikeAfter(id, 400*time.Millisecond)
		}
	case "d", "x":
		// Immediate delete
		if a.todoIdx < len(a.todos) {
			a.removeTodo(a.todos[a.todoIdx].ID)
		}
	}
	return *a, nil
}

func (a *App) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		if a.selectedSession() != nil {
			a.insertMode = true
		}
	case "y":
		if s := a.selectedSession(); s != nil && s.Status == model.StatusNeedsAttention {
			tmux.SendKeys(s.PaneID, "Enter")
		}
	case "a":
		if s := a.selectedSession(); s != nil && s.Status == model.StatusNeedsAttention {
			tmux.SendKeys(s.PaneID, "a")
		}
	}
	return *a, nil
}

func (a *App) handleInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := a.selectedSession()
	if s == nil {
		a.insertMode = false
		return *a, nil
	}
	switch msg.Type {
	case tea.KeyEscape:
		a.insertMode = false
	case tea.KeyEnter:
		tmux.SendKeys(s.PaneID, "Enter")
	case tea.KeyBackspace:
		tmux.SendKeys(s.PaneID, "BSpace")
	case tea.KeyTab:
		tmux.SendKeys(s.PaneID, "Tab")
	case tea.KeySpace:
		tmux.SendKeys(s.PaneID, "Space")
	case tea.KeyUp:
		tmux.SendKeys(s.PaneID, "Up")
	case tea.KeyDown:
		tmux.SendKeys(s.PaneID, "Down")
	default:
		if r := msg.String(); len(r) == 1 {
			tmux.SendKeys(s.PaneID, r)
		}
	}
	return *a, nil
}

func (a *App) handleTodoInsertKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		a.todoInsert = false
		a.todoInput = ""
	case tea.KeyEnter:
		text := strings.TrimSpace(a.todoInput)
		if text != "" {
			a.addTodo(text)
		}
		a.todoInput = ""
		// Stay in insert mode for rapid entry — Esc to exit
	case tea.KeyBackspace:
		if len(a.todoInput) > 0 {
			a.todoInput = a.todoInput[:len(a.todoInput)-1]
		}
	default:
		if r := msg.String(); len(r) >= 1 {
			a.todoInput += r
		}
	}
	return *a, nil
}

// ── Todo helpers ─────────────────────────────────────────────────

func (a *App) currentProjectSlug() string {
	s := a.selectedSession()
	if s == nil {
		return ""
	}
	name := s.ProjectName
	if name == "" {
		name = s.DisplayName()
	}
	return todo.Slug(name)
}

func (a *App) refreshTodos() {
	slug := a.currentProjectSlug()
	if slug == "" {
		a.todos = nil
		a.todoSlug = ""
		return
	}
	a.todoSlug = slug
	pt := todo.LoadProjectTodos(slug)
	// Filter out done items (they've been dismissed)
	var active []model.TodoItem
	for _, item := range pt.Items {
		if item.Status != model.TodoDone {
			active = append(active, item)
		}
	}
	a.todos = active
	if a.todoIdx >= len(a.todos) {
		a.todoIdx = max(0, len(a.todos)-1)
	}
}

func (a *App) addTodo(text string) {
	s := a.selectedSession()
	if s == nil {
		return
	}
	name := s.ProjectName
	if name == "" {
		name = s.DisplayName()
	}
	todo.AddTodo(name, text, "", nil)
	a.refreshTodos()
	a.todoIdx = len(a.todos) - 1
}

func (a *App) removeTodo(id string) {
	slug := a.currentProjectSlug()
	if slug == "" {
		return
	}
	todo.RemoveTodo(slug, id)
	a.refreshTodos()
}

// ── Session management ───────────────────────────────────────────

func (a *App) mergeSessions(updated []model.SessionInfo) {
	if len(updated) == 0 {
		return
	}
	existing := make(map[string]*model.SessionInfo, len(a.sessions))
	for i := range a.sessions {
		existing[a.sessions[i].PaneID] = &a.sessions[i]
	}

	merged := make([]model.SessionInfo, 0, len(updated))
	for _, u := range updated {
		if prev, ok := existing[u.PaneID]; ok {
			if u.Status == model.StatusUnknown && prev.Status != model.StatusUnknown {
				u.Status = prev.Status
			}
			if u.CostUSD == 0 && prev.CostUSD > 0 {
				u.CostUSD = prev.CostUSD
			}
			if u.ContextPct == 0 && prev.ContextPct > 0 {
				u.ContextPct = prev.ContextPct
				u.ContextTokens = prev.ContextTokens
			}
			if u.GitBranch == "" && prev.GitBranch != "" {
				u.GitBranch = prev.GitBranch
			}
			if u.LastActivity == "" && prev.LastActivity != "" {
				u.LastActivity = prev.LastActivity
			}
			if u.TaskSummary == "" && prev.TaskSummary != "" {
				u.TaskSummary = prev.TaskSummary
			}
			if u.SessionID == "" && prev.SessionID != "" {
				u.SessionID = prev.SessionID
			}
			if u.ProjectName == "" && prev.ProjectName != "" {
				u.ProjectName = prev.ProjectName
			}
			if u.PaneContent == "" && prev.PaneContent != "" {
				u.PaneContent = prev.PaneContent
			}
			u.CostHistory = prev.CostHistory
		}
		merged = append(merged, u)
	}
	a.sessions = merged
	if a.selectedIdx >= len(a.sessions) {
		a.selectedIdx = max(0, len(a.sessions)-1)
	}
}

func (a *App) selectedSession() *model.SessionInfo {
	if a.selectedIdx < len(a.sessions) {
		return &a.sessions[a.selectedIdx]
	}
	return nil
}

func (a *App) checkNotifications() {
	activePaneID := tmux.ActivePaneID()

	for _, s := range a.sessions {
		if s.Status == model.StatusUnknown {
			continue
		}
		prev, existed := a.prevStatuses[s.PaneID]
		if s.Status == model.StatusNeedsAttention &&
			(!existed || prev == model.StatusUnknown || prev != model.StatusNeedsAttention) {
			if a.notifyEnabled && s.PaneID != activePaneID {
				notify.Notify(s.DisplayName(), s.ProcessType.String())
			}
		}
		a.prevStatuses[s.PaneID] = s.Status
	}
}

// ── View ─────────────────────────────────────────────────────────

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	header := renderHeader(a.sessions, a.width, a.notifyEnabled, a.selectedSession())

	leftWidth := a.width * 3 / 10
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := a.width - leftWidth - 4
	headerLines := strings.Count(header, "\n") + 1
	bodyHeight := a.height - headerLines - 2

	// Dynamic left panel split: sessions + todos
	todoFocused := a.focusPanel == panelTodos
	todoHeight := a.todoSectionHeight(bodyHeight, todoFocused)
	sessionHeight := bodyHeight - todoHeight - 2 // -2 for todo border

	// Session list
	var listLines []string
	for i, s := range a.sessions {
		listLines = append(listLines, renderSessionRow(s, i == a.selectedIdx && a.focusPanel == panelSessions, a.blinkOn, a.spinnerIdx, leftWidth-2))
	}
	if len(listLines) == 0 {
		listLines = append(listLines, lipgloss.NewStyle().Foreground(colorDim).Render("  No sessions found"))
	}
	listContent := strings.Join(listLines, "\n")

	sessionBorder := panelBorder
	if a.focusPanel == panelSessions {
		sessionBorder = activePanelBorder
	}
	sessionPanel := sessionBorder.Width(leftWidth).Height(sessionHeight).Render(listContent)

	// Todo panel
	todoContent := a.renderTodoPanel(leftWidth-2, todoHeight, todoFocused)
	todoBorder := panelBorder
	if todoFocused {
		todoBorder = activePanelBorder
	}
	todoPanel := todoBorder.Width(leftWidth).Height(todoHeight).Render(todoContent)

	leftPanel := lipgloss.JoinVertical(lipgloss.Left, sessionPanel, todoPanel)

	// Right panel: detail
	var selected *model.SessionInfo
	if a.selectedIdx < len(a.sessions) {
		s := a.sessions[a.selectedIdx]
		selected = &s
	}
	detailContent := renderDetail(selected, bodyHeight, a.insertMode)

	rightBorder := panelBorder
	if a.focusPanel == panelDetail {
		rightBorder = activePanelBorder
	}
	rightPanel := rightBorder.Width(rightWidth).Height(bodyHeight).Render(detailContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	return fmt.Sprintf("%s\n%s", header, body)
}

func (a *App) todoSectionHeight(totalHeight int, focused bool) int {
	if focused {
		// Expand: at least 40% of left panel or enough for items + input
		h := totalHeight * 4 / 10
		needed := len(a.todos) + 3 // items + header + input + padding
		if needed > h {
			h = needed
		}
		if h > totalHeight-6 {
			h = totalHeight - 6 // leave room for at least a few session rows
		}
		return max(h, 4)
	}
	// Collapsed: show header + up to 3 items
	items := len(a.todos)
	if items > 3 {
		items = 3
	}
	return max(items+2, 3) // header + items + hint
}

func (a *App) renderTodoPanel(width, height int, focused bool) string {
	var lines []string

	// Header
	count := len(a.todos)
	headerText := fmt.Sprintf("Todos (%d)", count)
	if a.todoSlug != "" {
		headerText = fmt.Sprintf("Todos · %s (%d)", a.todoSlug, count)
	}
	if len(headerText) > width-2 {
		headerText = fmt.Sprintf("Todos (%d)", count)
	}
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(colorDim)
	if focused {
		hdrStyle = hdrStyle.Foreground(colorCyan)
	}
	lines = append(lines, hdrStyle.Render(headerText))

	if len(a.todos) == 0 && !a.todoInsert {
		lines = append(lines, lipgloss.NewStyle().Foreground(colorDim).Italic(true).
			Render("  (empty)"))
	}

	// Items
	maxItems := height - 2 // header + input line
	if a.todoInsert {
		maxItems--
	}
	if maxItems < 0 {
		maxItems = 0
	}
	visible := a.todos
	if len(visible) > maxItems {
		// Scroll to keep selected visible
		start := a.todoIdx - maxItems + 1
		if start < 0 {
			start = 0
		}
		visible = visible[start:]
		if len(visible) > maxItems {
			visible = visible[:maxItems]
		}
	}

	for _, item := range visible {
		selected := focused && item.ID == a.todoID()
		striking := item.ID == a.strikingID

		bullet := "○"
		style := lipgloss.NewStyle().Foreground(colorText)

		if striking {
			bullet = "●"
			style = style.Foreground(colorDim).Strikethrough(true)
		} else if selected {
			bullet = "›"
			style = style.Foreground(colorCyan)
		}

		text := item.Text
		maxLen := width - 4
		if maxLen < 5 {
			maxLen = 5
		}
		if len(text) > maxLen {
			text = text[:maxLen-1] + "…"
		}

		prefix := "  "
		if selected {
			prefix = lipgloss.NewStyle().Foreground(colorCyan).Render("▎ ")
		}
		lines = append(lines, prefix+lipgloss.NewStyle().Foreground(colorDim).Render(bullet)+" "+style.Render(text))
	}

	// Insert line
	if a.todoInsert {
		cursor := lipgloss.NewStyle().Foreground(colorYellow).Render("▎ + ")
		input := a.todoInput
		if len(input) > width-6 {
			input = input[len(input)-width+6:]
		}
		lines = append(lines, cursor+lipgloss.NewStyle().Foreground(colorText).Render(input+"█"))
	} else if focused && len(lines) < height {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorDim).Render("  i:add  ⏎:done  d:del"))
	}

	return strings.Join(lines, "\n")
}

func (a *App) todoID() string {
	if a.todoIdx < len(a.todos) {
		return a.todos[a.todoIdx].ID
	}
	return ""
}

func copySessionSlice(src []model.SessionInfo) []model.SessionInfo {
	dst := make([]model.SessionInfo, len(src))
	copy(dst, src)
	return dst
}

// Run starts the TUI application.
func Run() error {
	model.EnsureDirs()
	p := tea.NewProgram(newApp(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
