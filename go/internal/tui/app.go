package tui

import (
	"fmt"
	"os"
	"regexp"
	"sort"
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

// Tabs
const (
	tabSessions = iota
	tabTodos
	tabOutput
)

// Sort modes
const (
	sortDefault = iota // discovery order
	sortStatus         // attention → working → idle → disconnected
	sortCost           // highest cost first
	sortAge            // most recently active first
	sortName           // alphabetical
)

// App is the root Bubble Tea model.
type App struct {
	sessions      []model.SessionInfo
	selectedIdx   int
	width, height int
	prevStatuses  map[string]model.SessionStatus
	discovering   bool
	notifyEnabled bool

	// Animation state (driven by fast animTick)
	blinkOn    bool
	spinnerIdx int

	// Pane insert mode
	insertMode bool

	// Todo state
	todos      []model.TodoItem // current project's todos
	todoSlug   string           // current project slug
	todoIdx    int              // selected todo index
	todoInput  string           // text being typed in todo insert mode
	todoInsert bool             // true = typing a new todo
	strikingID string           // todo ID currently showing strikethrough

	// Tabs
	activeTab int // tabSessions, tabTodos, tabOutput

	// Filter and sort
	filterActive bool
	filterText   string
	sortMode     int // sortDefault..sortName

	// Overlays
	showHelp   bool
	confirmMsg string // non-empty = confirm modal visible
	confirmCmd string // action to execute on confirm

	// Environment
	sshSafe bool
}

func newApp() App {
	a := App{
		activeTab:     tabSessions,
		notifyEnabled: true,
		prevStatuses:  make(map[string]model.SessionStatus),
		blinkOn:       true,
	}
	if isSSH() {
		a.sshSafe = true
		animIntervalDuration = 100 * time.Millisecond // 10 FPS cap
	}
	return a
}

func isSSH() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != ""
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		doDiscovery,
		animTick(),
		pollTick(),
	)
}

var animIntervalDuration = 150 * time.Millisecond

func animTick() tea.Cmd {
	return tea.Tick(animIntervalDuration, func(t time.Time) tea.Msg {
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
			// Store cleaned content for preview (strip ANSI and control sequences)
			s.PaneContent = StripControlSequences(content)
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
					s.GitBranch = sanitizeField(cs.GitBranch)
				}
				if cs.ProjectName != "" {
					s.ProjectName = sanitizeField(cs.ProjectName)
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
				s.TaskSummary = sanitizeField(task)
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
					s.LastActivity = sanitizeField(data.LastMessage)
				}
				if s.TaskSummary == "" {
					if data.FirstHumanMessage != "" {
						s.TaskSummary = sanitizeField(data.FirstHumanMessage)
					} else if data.LastHumanMessage != "" {
						s.TaskSummary = sanitizeField(data.LastHumanMessage)
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
	ansiRe       = regexp.MustCompile(`\x1b(?:[@-Z\\-_]|\[[?!>]?[0-9;]*[a-zA-Z~])`)
)

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Priority: confirm modal > help overlay > filter > todo insert > insert > normal
		if a.confirmMsg != "" {
			return a.handleConfirmKey(msg)
		}
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}
		if a.filterActive {
			return a.handleFilterKey(msg)
		}
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
		a.updateSessions([]model.SessionInfo(msg))
		a.checkNotifications()
		a.refreshTodos()
		return a, nil

	case costUpdateMsg:
		a.updateSessions([]model.SessionInfo(msg))
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
	k := msg.String()

	// Global keys
	switch k {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "tab":
		a.activeTab = (a.activeTab + 1) % 3
		a.refreshTodos()
		return *a, nil
	case "shift+tab":
		a.activeTab = (a.activeTab + 2) % 3
		a.refreshTodos()
		return *a, nil
	case "1":
		a.activeTab = tabSessions
		a.refreshTodos()
		return *a, nil
	case "2":
		a.activeTab = tabTodos
		a.refreshTodos()
		return *a, nil
	case "3":
		a.activeTab = tabOutput
		if s := a.selectedSession(); s != nil {
			return *a, doPaneUpdate([]model.SessionInfo{*s})
		}
		return *a, nil
	case "r":
		return *a, doDiscovery
	case "n":
		a.notifyEnabled = !a.notifyEnabled
		return *a, nil
	case "?":
		a.showHelp = !a.showHelp
		return *a, nil
	case "/":
		a.filterActive = true
		a.filterText = ""
		return *a, nil
	case "s":
		if !a.filterActive {
			a.sortMode = (a.sortMode + 1) % 5
		}
		return *a, nil
	}

	// Tab-specific keys
	switch a.activeTab {
	case tabSessions, tabOutput:
		return a.handleSessionKey(msg)
	case tabTodos:
		return a.handleTodoKey(msg)
	}
	return *a, nil
}

func (a *App) handleSessionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := a.filteredSessions()
	switch msg.String() {
	case "j", "down":
		if a.selectedIdx < len(filtered)-1 {
			a.selectedIdx++
			a.refreshTodos()
			return *a, doPaneUpdate([]model.SessionInfo{filtered[a.selectedIdx]})
		}
	case "k", "up":
		if a.selectedIdx > 0 {
			a.selectedIdx--
			a.refreshTodos()
			return *a, doPaneUpdate([]model.SessionInfo{filtered[a.selectedIdx]})
		}
	case "g", "home":
		a.selectedIdx = 0
		a.refreshTodos()
		if len(filtered) > 0 {
			return *a, doPaneUpdate([]model.SessionInfo{filtered[0]})
		}
	case "G", "end":
		if len(filtered) > 0 {
			a.selectedIdx = len(filtered) - 1
			a.refreshTodos()
			return *a, doPaneUpdate([]model.SessionInfo{filtered[a.selectedIdx]})
		}
	case "enter":
		if a.selectedIdx < len(filtered) {
			tmux.SwitchToPane(filtered[a.selectedIdx].PaneID)
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
			a.confirmMsg = fmt.Sprintf("Send Escape to '%s'?", s.DisplayName())
			a.confirmCmd = "send_escape"
		}
	case "i":
		if a.selectedIdx < len(filtered) {
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
		if a.todoIdx < len(a.todos) && a.strikingID == "" {
			id := a.todos[a.todoIdx].ID
			a.strikingID = id
			return *a, todoStrikeAfter(id, 400*time.Millisecond)
		}
	case "d", "x":
		if a.todoIdx < len(a.todos) {
			a.removeTodo(a.todos[a.todoIdx].ID)
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
	paneID := s.PaneID
	switch msg.Type {
	case tea.KeyEscape:
		a.insertMode = false
	case tea.KeyEnter:
		tmux.SendKeyFast(paneID, "Enter")
	case tea.KeyBackspace:
		tmux.SendKeyFast(paneID, "BSpace")
	case tea.KeyTab:
		tmux.SendKeyFast(paneID, "Tab")
	case tea.KeySpace:
		tmux.SendKeyFast(paneID, "Space")
	case tea.KeyUp:
		tmux.SendKeyFast(paneID, "Up")
	case tea.KeyDown:
		tmux.SendKeyFast(paneID, "Down")
	default:
		if r := msg.String(); len(r) == 1 {
			tmux.SendKeyFast(paneID, r)
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

func (a *App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		a.filterActive = false // hide bar but keep filter applied
	case tea.KeyEnter:
		a.filterActive = false
	case tea.KeyBackspace:
		if len(a.filterText) > 0 {
			a.filterText = a.filterText[:len(a.filterText)-1]
		}
	default:
		if r := msg.String(); len(r) == 1 {
			a.filterText += r
		}
	}
	// Clamp selected index to filtered list
	filtered := a.filteredSessions()
	if a.selectedIdx >= len(filtered) && len(filtered) > 0 {
		a.selectedIdx = len(filtered) - 1
	}
	return *a, nil
}

func (a *App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if a.confirmCmd == "send_escape" {
			if s := a.selectedSession(); s != nil {
				tmux.SendKeys(s.PaneID, "Escape")
			}
		}
		a.confirmMsg = ""
		a.confirmCmd = ""
	case "n", "escape":
		a.confirmMsg = ""
		a.confirmCmd = ""
	}
	return *a, nil
}

// ── Filter and sort ──────────────────────────────────────────────

func (a *App) filteredSessions() []model.SessionInfo {
	sessions := make([]model.SessionInfo, len(a.sessions))
	copy(sessions, a.sessions)

	if a.filterText != "" {
		lower := strings.ToLower(a.filterText)
		var filtered []model.SessionInfo
		for _, s := range sessions {
			if strings.Contains(strings.ToLower(s.DisplayName()), lower) ||
				strings.Contains(strings.ToLower(s.GitBranch), lower) ||
				strings.Contains(strings.ToLower(s.TaskSummary), lower) {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	switch a.sortMode {
	case sortStatus:
		sort.SliceStable(sessions, func(i, j int) bool {
			return statusPriority(sessions[i].Status) < statusPriority(sessions[j].Status)
		})
	case sortCost:
		sort.SliceStable(sessions, func(i, j int) bool {
			return sessions[i].CostUSD > sessions[j].CostUSD
		})
	case sortAge:
		sort.SliceStable(sessions, func(i, j int) bool {
			return sessions[i].LastChecked.After(sessions[j].LastChecked)
		})
	case sortName:
		sort.SliceStable(sessions, func(i, j int) bool {
			return sessions[i].DisplayName() < sessions[j].DisplayName()
		})
	}
	return sessions
}

func statusPriority(s model.SessionStatus) int {
	switch s {
	case model.StatusNeedsAttention:
		return 0
	case model.StatusWorking:
		return 1
	case model.StatusIdle:
		return 2
	case model.StatusDisconnected:
		return 3
	}
	return 4
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

// mergeFields copies preserved fields from prev into u.
func mergeFields(u *model.SessionInfo, prev *model.SessionInfo) {
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

// mergeSessions handles discovery results: adds new sessions, updates existing
// ones, and marks sessions missing from discovery as Disconnected (never drops them).
func (a *App) mergeSessions(discovered []model.SessionInfo) {
	if len(discovered) == 0 {
		return
	}
	existing := make(map[string]*model.SessionInfo, len(a.sessions))
	for i := range a.sessions {
		existing[a.sessions[i].PaneID] = &a.sessions[i]
	}

	inDiscovery := make(map[string]bool, len(discovered))
	merged := make([]model.SessionInfo, 0, len(discovered))
	for _, u := range discovered {
		inDiscovery[u.PaneID] = true
		if prev, ok := existing[u.PaneID]; ok {
			mergeFields(&u, prev)
		}
		merged = append(merged, u)
	}

	// Keep sessions not found by discovery as Disconnected rather than dropping them.
	for _, prev := range a.sessions {
		if !inDiscovery[prev.PaneID] {
			prev.Status = model.StatusDisconnected
			merged = append(merged, prev)
		}
	}

	a.sessions = merged
	if a.selectedIdx >= len(a.sessions) {
		a.selectedIdx = max(0, len(a.sessions)-1)
	}
}

// updateSessions handles pane/cost update results: only updates existing sessions,
// never adds or removes entries (discovery is the sole authority for that).
func (a *App) updateSessions(updated []model.SessionInfo) {
	if len(updated) == 0 {
		return
	}
	index := make(map[string]model.SessionInfo, len(updated))
	for _, u := range updated {
		index[u.PaneID] = u
	}
	for i := range a.sessions {
		u, ok := index[a.sessions[i].PaneID]
		if !ok {
			continue
		}
		prev := a.sessions[i]
		mergeFields(&u, &prev)
		a.sessions[i] = u
	}
}

func (a *App) selectedSession() *model.SessionInfo {
	filtered := a.filteredSessions()
	if a.selectedIdx < len(filtered) {
		// Find the matching session in a.sessions
		paneID := filtered[a.selectedIdx].PaneID
		for i := range a.sessions {
			if a.sessions[i].PaneID == paneID {
				return &a.sessions[i]
			}
		}
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

const minWidth = 60
const minHeight = 12

func (a App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	if a.width < minWidth || a.height < minHeight {
		return renderTooSmall(a.width, a.height)
	}

	header := renderHeader(a.sessions, a.width, a.notifyEnabled, a.selectedSession())
	tabBar := renderTabBar(a.activeTab, a.width)
	headerLines := strings.Count(header, "\n") + 1

	filtered := a.filteredSessions()
	filterBar := renderFilterBar(a.filterActive, a.filterText, len(a.sessions), len(filtered), a.width)
	filterBarLines := 0
	if filterBar != "" {
		filterBarLines = 1
	}

	// bodyHeight accounts for: header + tab bar (1) + filter bar (0-1) + borders (2)
	bodyHeight := a.height - headerLines - 1 - filterBarLines - 2

	var body string
	switch a.activeTab {
	case tabTodos:
		body = a.renderTodosFullTab(bodyHeight)
	case tabOutput:
		body = renderOutputTab(a, a.width, bodyHeight)
	default: // tabSessions
		body = a.renderSessionsBody(filtered, bodyHeight)
	}

	parts := []string{header, tabBar}
	if filterBar != "" {
		parts = append(parts, filterBar)
	}
	parts = append(parts, body)
	view := strings.Join(parts, "\n")

	if a.showHelp {
		return renderHelpOverlay(a.width, a.height)
	}
	if a.confirmMsg != "" {
		return renderConfirmModal(a.confirmMsg, a.width, a.height)
	}
	return view
}

func (a App) renderSessionsBody(filtered []model.SessionInfo, bodyHeight int) string {
	leftWidth := a.width * 3 / 10
	if leftWidth < 25 {
		leftWidth = 25
	}
	rightWidth := a.width - leftWidth - 4

	// Small todo section always collapsed at bottom of left panel
	todoHeight := a.todoSectionHeight(bodyHeight, false)
	sessionHeight := bodyHeight - todoHeight - 2

	// Sessions table
	var listLines []string
	listLines = append(listLines, renderSessionTableHeader(leftWidth-2, a.sortMode))
	for i, s := range filtered {
		listLines = append(listLines, renderSessionTableRow(s, i == a.selectedIdx, a.blinkOn, a.spinnerIdx, leftWidth-2))
	}
	if len(filtered) == 0 {
		noSess := lipgloss.NewStyle().Foreground(tokenFgMuted).Render("  No sessions found")
		if a.filterText != "" {
			noSess = lipgloss.NewStyle().Foreground(tokenFgMuted).Render("  No sessions match filter")
		}
		listLines = append(listLines, noSess)
	}
	listContent := strings.Join(listLines, "\n")
	sessionPanel := activePanelBorder.Width(leftWidth).Height(sessionHeight).Render(listContent)

	// Todo section (read-only in sessions tab; press 2 to manage)
	todoContent := a.renderTodoPanel(leftWidth-2, todoHeight, false)
	todoPanel := panelBorder.Width(leftWidth).Height(todoHeight).Render(todoContent)

	leftPanel := lipgloss.JoinVertical(lipgloss.Left, sessionPanel, todoPanel)

	// Right panel: detail
	var selected *model.SessionInfo
	if a.selectedIdx < len(filtered) {
		s := filtered[a.selectedIdx]
		selected = &s
	}
	detailContent := renderDetail(selected, bodyHeight, a.insertMode)
	rightPanel := panelBorder.Width(rightWidth).Height(bodyHeight).Render(detailContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (a App) renderTodosFullTab(height int) string {
	width := a.width - 4
	content := a.renderTodoPanel(width-2, height-2, true)
	return activePanelBorder.Width(width).Height(height).Render(content)
}

func (a *App) todoSectionHeight(totalHeight int, focused bool) int {
	if focused {
		h := totalHeight * 4 / 10
		needed := len(a.todos) + 3
		if needed > h {
			h = needed
		}
		if h > totalHeight-6 {
			h = totalHeight - 6
		}
		return max(h, 4)
	}
	// Collapsed: show header + up to 3 items
	items := len(a.todos)
	if items > 3 {
		items = 3
	}
	return max(items+2, 3)
}

func (a *App) renderTodoPanel(width, height int, focused bool) string {
	var lines []string

	count := len(a.todos)
	headerText := fmt.Sprintf("Todos (%d)", count)
	if a.todoSlug != "" {
		headerText = fmt.Sprintf("Todos · %s (%d)", a.todoSlug, count)
	}
	if len(headerText) > width-2 {
		headerText = fmt.Sprintf("Todos (%d)", count)
	}
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(tokenFgMuted)
	if focused {
		hdrStyle = hdrStyle.Foreground(tokenFgAccent)
	}
	lines = append(lines, hdrStyle.Render(headerText))

	if !focused {
		// In sessions tab: show hint to switch to todos tab
		lines = append(lines, lipgloss.NewStyle().Foreground(tokenFgMuted).Render("  2:manage todos"))
	}

	if len(a.todos) == 0 && !a.todoInsert {
		lines = append(lines, lipgloss.NewStyle().Foreground(tokenFgMuted).Italic(true).
			Render("  (empty)"))
	}

	maxItems := height - 2
	if a.todoInsert {
		maxItems--
	}
	if !focused {
		maxItems = 3 // always collapsed in sessions tab
	}
	if maxItems < 0 {
		maxItems = 0
	}
	visible := a.todos
	if len(visible) > maxItems {
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
		style := lipgloss.NewStyle().Foreground(tokenFgDefault)

		if striking {
			bullet = "●"
			style = style.Foreground(tokenFgMuted).Strikethrough(true)
		} else if selected {
			bullet = "›"
			style = style.Foreground(tokenFgAccent)
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
			prefix = lipgloss.NewStyle().Foreground(tokenFgAccent).Render("▎ ")
		}
		lines = append(lines, prefix+lipgloss.NewStyle().Foreground(tokenFgMuted).Render(bullet)+" "+style.Render(text))
	}

	if a.todoInsert {
		cursor := lipgloss.NewStyle().Foreground(colorYellow).Render("▎ + ")
		input := a.todoInput
		if len(input) > width-6 {
			input = input[len(input)-width+6:]
		}
		lines = append(lines, cursor+lipgloss.NewStyle().Foreground(tokenFgDefault).Render(input+"█"))
	} else if focused && len(lines) < height {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(tokenFgMuted).Render("  i:add  ⏎:done  d:del"))
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
