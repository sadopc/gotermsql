package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/completion"
	"github.com/sadopc/gotermsql/internal/config"
	"github.com/sadopc/gotermsql/internal/history"
	"github.com/sadopc/gotermsql/internal/schema"
	"github.com/sadopc/gotermsql/internal/theme"
	"github.com/sadopc/gotermsql/internal/ui/autocomplete"
	"github.com/sadopc/gotermsql/internal/ui/connmgr"
	"github.com/sadopc/gotermsql/internal/ui/editor"
	"github.com/sadopc/gotermsql/internal/ui/results"
	"github.com/sadopc/gotermsql/internal/ui/sidebar"
	"github.com/sadopc/gotermsql/internal/ui/statusbar"
	"github.com/sadopc/gotermsql/internal/ui/tabs"
)

// TabState holds per-tab state.
type TabState struct {
	Editor  editor.Model
	Results results.Model
	Query   string
	RunID   uint64
}

// Model is the root application model.
type Model struct {
	// Layout
	width        int
	height       int
	sidebarWidth int
	editorHeight int // percentage of main area for editor (rest for results)
	showSidebar  bool

	// Focus
	focusedPane Pane

	// Components
	sidebar   sidebar.Model
	tabs      tabs.Model
	statusbar statusbar.Model
	connMgr   connmgr.Model
	autocomp  autocomplete.Model
	help      help.Model
	spinner   spinner.Model

	// Per-tab state
	tabStates map[int]*TabState

	// Database
	conn       adapter.Connection
	cancelFunc context.CancelFunc
	connGen    uint64

	// Engine
	compEngine *completion.Engine

	// Config
	cfg     *config.Config
	history *history.History

	// Keybinding
	keyMap   KeyMap
	keyMode  KeyMode
	vimState VimState

	// State
	showHelp    bool
	showConnMgr bool
	executing   bool
	quitting    bool
}

// New creates a new app model.
func New(cfg *config.Config, hist *history.History) Model {
	keyMode := ParseKeyMode(cfg.KeyMode)
	var km KeyMap
	if keyMode == KeyModeVim {
		km = VimKeyMap()
	} else {
		km = StandardKeyMap()
	}

	// Set theme
	if t := theme.Get(cfg.Theme); t != nil {
		theme.Current = t
	}

	s := spinner.New()
	s.Spinner = spinner.Dot

	compEngine := completion.NewEngine("sql")

	m := Model{
		sidebarWidth: 30,
		editorHeight: 50,
		showSidebar:  true,
		focusedPane:  PaneEditor,

		sidebar:   sidebar.New(),
		tabs:      tabs.New(),
		statusbar: statusbar.New(),
		connMgr:   connmgr.New(cfg.Connections),
		autocomp:  autocomplete.New(compEngine),
		help:      help.New(),
		spinner:   s,

		tabStates:  make(map[int]*TabState),
		compEngine: compEngine,
		cfg:        cfg,
		history:    hist,
		keyMap:     km,
		keyMode:    keyMode,
	}

	// Initialize first tab state
	ed := editor.New(0)
	ed.Focus()
	m.tabStates[0] = &TabState{
		Editor:  ed,
		Results: results.New(),
	}

	m.statusbar.SetKeyMode(keyMode)
	return m
}

// Init initializes the application.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		// Connection manager takes priority
		if m.connMgr.Visible() {
			var cmd tea.Cmd
			m.connMgr, cmd = m.connMgr.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Help overlay consumes all keys except toggle/close
		if m.showHelp {
			if msg.String() == "f1" || msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
				m.showHelp = false
			}
			return m, nil
		}

		// Autocomplete takes priority when visible
		if m.autocomp.Visible() {
			switch msg.String() {
			case "up", "down", "enter", "tab", "esc", "ctrl+p", "ctrl+n":
				var cmd tea.Cmd
				m.autocomp, cmd = m.autocomp.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return m, tea.Batch(cmds...)
			}
		}

		// Global keybindings
		cmd := m.handleGlobalKeys(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Route to focused pane
		cmd = m.handleFocusedPaneKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ConnectMsg:
		if m.conn != nil {
			m.conn.Close()
		}
		m.conn = msg.Conn
		m.connGen++
		m.showConnMgr = false
		m.connMgr.Hide()
		var cmd tea.Cmd
		m.statusbar, cmd = m.statusbar.Update(msg)
		cmds = append(cmds, cmd)
		// Load schema
		m.sidebar.SetLoading(true)
		cmds = append(cmds, m.loadSchema())

	case ConnectErrMsg:
		var sbCmd tea.Cmd
		m.statusbar, sbCmd = m.statusbar.Update(StatusMsg{
			Text: "Connection failed: " + msg.Err.Error(), IsError: true,
		})
		cmds = append(cmds, sbCmd)

	case SchemaLoadedMsg:
		if msg.ConnGen != m.connGen {
			break // stale schema from previous connection
		}
		m.sidebar.SetLoading(false)
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		cmds = append(cmds, cmd)
		// Update completion engine
		if m.conn != nil {
			m.compEngine = completion.NewEngine(m.conn.AdapterName())
			m.compEngine.UpdateSchema(msg.Databases)
			m.autocomp.SetEngine(m.compEngine)
		} else {
			m.compEngine.UpdateSchema(msg.Databases)
		}

	case SchemaErrMsg:
		if msg.ConnGen != m.connGen {
			break // stale error from previous connection
		}
		m.sidebar.SetLoading(false)
		var sbCmd tea.Cmd
		m.statusbar, sbCmd = m.statusbar.Update(StatusMsg{
			Text: "Schema load failed: " + msg.Err.Error(), IsError: true,
		})
		cmds = append(cmds, sbCmd)

	case ExecuteQueryMsg:
		cmds = append(cmds, m.executeQuery(msg.Query, msg.TabID))

	case QueryStartedMsg:
		ts := m.tabStates[msg.TabID]
		if ts != nil && msg.RunID == ts.RunID {
			m.executing = true
			ts.Results.SetLoading(true)
		}

	case QueryResultMsg:
		ts := m.tabStates[msg.TabID]
		if ts != nil && msg.RunID == ts.RunID {
			m.executing = false
			ts.Results.SetLoading(false)
			if msg.Result != nil {
				ts.Results.SetResults(msg.Result)
			}
			// Save to history
			if m.history != nil && m.conn != nil && msg.Result != nil {
				m.history.Add(history.HistoryEntry{
					Query:        ts.Query,
					Adapter:      m.conn.AdapterName(),
					DatabaseName: m.conn.DatabaseName(),
					ExecutedAt:   time.Now(),
					DurationMS:   msg.Result.Duration.Milliseconds(),
					RowCount:     msg.Result.RowCount,
				})
			}
			var sbCmd tea.Cmd
			m.statusbar, sbCmd = m.statusbar.Update(msg)
			cmds = append(cmds, sbCmd)
		}

	case QueryErrMsg:
		ts := m.tabStates[msg.TabID]
		if ts != nil && msg.RunID == ts.RunID {
			m.executing = false
			ts.Results.SetLoading(false)
			ts.Results.SetError(msg.Err)
			// Save error to history
			if m.history != nil && m.conn != nil {
				m.history.Add(history.HistoryEntry{
					Query:        ts.Query,
					Adapter:      m.conn.AdapterName(),
					DatabaseName: m.conn.DatabaseName(),
					ExecutedAt:   time.Now(),
					IsError:      true,
				})
			}
			var sbCmd tea.Cmd
			m.statusbar, sbCmd = m.statusbar.Update(msg)
			cmds = append(cmds, sbCmd)
		}

	case NewTabMsg:
		// Blur current editor before switching
		if ts := m.activeTabState(); ts != nil {
			ts.Editor.Blur()
		}
		var cmd tea.Cmd
		m.tabs, cmd = m.tabs.Update(msg)
		cmds = append(cmds, cmd)
		tabID := m.tabs.ActiveID()
		ed := editor.New(tabID)
		ed.Focus()
		if msg.Query != "" {
			ed.SetValue(msg.Query)
		}
		m.tabStates[tabID] = &TabState{
			Editor:  ed,
			Results: results.New(),
		}
		m.updateLayout()
		m.focusedPane = PaneEditor

	case CloseTabMsg:
		delete(m.tabStates, msg.TabID)
		var cmd tea.Cmd
		m.tabs, cmd = m.tabs.Update(msg)
		cmds = append(cmds, cmd)

	case SwitchTabMsg:
		m.tabs, _ = m.tabs.Update(msg)
		m.updateLayout()

	case ToggleKeyModeMsg:
		if m.keyMode == KeyModeStandard {
			m.keyMode = KeyModeVim
			m.keyMap = VimKeyMap()
			m.vimState = VimNormal
		} else {
			m.keyMode = KeyModeStandard
			m.keyMap = StandardKeyMap()
		}
		m.statusbar.SetKeyMode(m.keyMode)
		var sbCmd tea.Cmd
		m.statusbar, sbCmd = m.statusbar.Update(msg)
		cmds = append(cmds, sbCmd)

	case autocomplete.SelectedMsg:
		ts := m.activeTabState()
		if ts != nil {
			ts.Editor.ReplaceWord(msg.Text, msg.PrefixLen)
		}

	case connmgr.ConnectRequestMsg:
		cmds = append(cmds, m.connect(msg.AdapterName, msg.DSN))

	case ExportCompleteMsg:
		var sbCmd tea.Cmd
		m.statusbar, sbCmd = m.statusbar.Update(StatusMsg{
			Text: fmt.Sprintf("Exported %d rows to %s", msg.RowCount, msg.Path),
		})
		cmds = append(cmds, sbCmd)

	case ExportErrMsg:
		var sbCmd tea.Cmd
		m.statusbar, sbCmd = m.statusbar.Update(StatusMsg{
			Text: "Export failed: " + msg.Err.Error(), IsError: true,
		})
		cmds = append(cmds, sbCmd)

	case statusbar.ClearStatusMsg:
		m.statusbar, _ = m.statusbar.Update(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case msg.String() == "ctrl+q":
		m.quitting = true
		return tea.Quit

	case msg.String() == "ctrl+c":
		if m.executing {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			if m.conn != nil {
				m.conn.Cancel()
			}
			m.executing = false
			var sbCmd tea.Cmd
			m.statusbar, sbCmd = m.statusbar.Update(StatusMsg{Text: "Query cancelled"})
			return sbCmd
		}
		return nil

	case msg.String() == "f1":
		m.showHelp = !m.showHelp
		return nil

	case msg.String() == "?" && m.focusedPane != PaneEditor:
		m.showHelp = !m.showHelp
		return nil

	case msg.String() == "f2":
		return func() tea.Msg { return ToggleKeyModeMsg{} }

	case msg.String() == "ctrl+b":
		m.showSidebar = !m.showSidebar
		m.updateLayout()
		return nil

	case msg.String() == "ctrl+r":
		if m.conn != nil {
			m.sidebar.SetLoading(true)
			return m.loadSchema()
		}
		return nil

	case msg.String() == "ctrl+e":
		return m.exportResults()

	case msg.String() == "ctrl+o":
		m.connMgr.Show()
		return nil

	case msg.String() == "ctrl+t":
		return func() tea.Msg { return NewTabMsg{} }

	case msg.String() == "ctrl+w":
		tabID := m.tabs.ActiveID()
		return func() tea.Msg { return CloseTabMsg{TabID: tabID} }

	case msg.String() == "ctrl+]":
		return m.tabs.NextTab()

	case msg.String() == "ctrl+[":
		return m.tabs.PrevTab()

	case msg.String() == "tab" && m.focusedPane != PaneEditor:
		m.cycleFocus(1)
		return nil

	case msg.String() == "shift+tab":
		m.cycleFocus(-1)
		return nil

	case msg.String() == "alt+1":
		m.setFocus(PaneSidebar)
		return nil

	case msg.String() == "alt+2":
		m.setFocus(PaneEditor)
		return nil

	case msg.String() == "alt+3":
		m.setFocus(PaneResults)
		return nil

	case msg.String() == "ctrl+left":
		if m.sidebarWidth > 15 {
			m.sidebarWidth -= 2
			m.updateLayout()
		}
		return nil

	case msg.String() == "ctrl+right":
		if m.sidebarWidth < m.width/2 {
			m.sidebarWidth += 2
			m.updateLayout()
		}
		return nil

	case msg.String() == "ctrl+up":
		if m.editorHeight > 20 {
			m.editorHeight -= 5
			m.updateLayout()
		}
		return nil

	case msg.String() == "ctrl+down":
		if m.editorHeight < 80 {
			m.editorHeight += 5
			m.updateLayout()
		}
		return nil
	}
	return nil
}

func (m *Model) handleFocusedPaneKey(msg tea.KeyMsg) tea.Cmd {
	ts := m.activeTabState()
	if ts == nil {
		return nil
	}

	switch m.focusedPane {
	case PaneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return cmd

	case PaneEditor:
		// Execute query on ctrl+enter, F5, or ctrl+g
		if msg.String() == "ctrl+enter" || msg.String() == "f5" || msg.String() == "ctrl+g" {
			query := ts.Editor.Value()
			if query != "" {
				tabID := m.tabs.ActiveID()
				return func() tea.Msg { return ExecuteQueryMsg{Query: query, TabID: tabID} }
			}
			return nil
		}

		// Trigger autocomplete on ctrl+space
		if msg.String() == "ctrl+@" || msg.String() == "ctrl+ " {
			text := ts.Editor.Value()
			// Approximate cursor position from textarea
			m.autocomp.TriggerForced(text, len(text))
			return nil
		}

		var cmd tea.Cmd
		ts.Editor, cmd = ts.Editor.Update(msg)

		// Trigger autocomplete after typing
		if isTypingKey(msg) {
			text := ts.Editor.Value()
			m.autocomp.Trigger(text, len(text))
		}

		return cmd

	case PaneResults:
		var cmd tea.Cmd
		ts.Results, cmd = ts.Results.Update(msg)
		return cmd
	}

	return nil
}

// View renders the entire application.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	th := theme.Current

	// Tab bar (top)
	tabBar := m.tabs.View()

	// Status bar (bottom)
	statusBar := m.statusbar.View()

	// Main content area
	mainHeight := m.height - lipgloss.Height(tabBar) - lipgloss.Height(statusBar)
	if mainHeight < 1 {
		mainHeight = 1
	}

	// Editor + Results
	ts := m.activeTabState()
	var editorView, resultsView string
	if ts != nil {
		editorH := mainHeight * m.editorHeight / 100
		resultsH := mainHeight - editorH
		if editorH < 3 {
			editorH = 3
		}
		if resultsH < 3 {
			resultsH = 3
		}

		mainWidth := m.width
		if m.showSidebar {
			mainWidth = m.width - m.sidebarWidth
		}

		ts.Editor.SetSize(mainWidth, editorH)
		ts.Results.SetSize(mainWidth, resultsH)

		editorView = ts.Editor.View()
		resultsView = ts.Results.View()

		// Autocomplete overlay - render within editor space to avoid pushing content off-screen
		if m.autocomp.Visible() {
			acView := m.autocomp.View()
			acHeight := lipgloss.Height(acView)
			editorLines := strings.Split(editorView, "\n")
			if acHeight < len(editorLines) {
				// Replace bottom lines of editor with autocomplete
				editorLines = editorLines[:len(editorLines)-acHeight]
				editorView = strings.Join(editorLines, "\n") + "\n" + acView
			} else {
				// Editor too small, just show autocomplete below first line
				if len(editorLines) > 1 {
					editorView = editorLines[0] + "\n" + acView
				}
			}
		}
	} else {
		editorView = "No active tab"
		resultsView = ""
	}

	mainContent := lipgloss.JoinVertical(lipgloss.Left, editorView, resultsView)

	// Sidebar + Main
	var content string
	if m.showSidebar {
		m.sidebar.SetSize(m.sidebarWidth, mainHeight)
		sidebarView := m.sidebar.View()
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainContent)
	} else {
		content = mainContent
	}

	// Assemble full view
	view := lipgloss.JoinVertical(lipgloss.Left, tabBar, content, statusBar)

	// Help overlay — full-screen centered
	if m.showHelp {
		helpContent := m.renderHelpScreen(th)
		view = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpContent)
	}

	// Connection manager overlay
	if m.connMgr.Visible() {
		connView := m.connMgr.View()
		// Center the connection manager
		centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, connView)
		return centered
	}

	return view
}

func (m *Model) updateLayout() {
	// Tab bar
	m.tabs.SetSize(m.width)

	// Status bar
	m.statusbar.SetSize(m.width)

	// Connection manager
	m.connMgr.SetSize(m.width, m.height)

	// Resize active tab state
	ts := m.activeTabState()
	if ts != nil {
		mainHeight := m.height - 3 // tab bar + status bar estimate
		mainWidth := m.width
		if m.showSidebar {
			mainWidth = m.width - m.sidebarWidth
		}
		editorH := mainHeight * m.editorHeight / 100
		resultsH := mainHeight - editorH
		ts.Editor.SetSize(mainWidth, editorH)
		ts.Results.SetSize(mainWidth, resultsH)
	}
}

func (m *Model) cycleFocus(direction int) {
	panes := []Pane{PaneEditor, PaneResults}
	if m.showSidebar {
		panes = []Pane{PaneSidebar, PaneEditor, PaneResults}
	}

	current := 0
	for i, p := range panes {
		if p == m.focusedPane {
			current = i
			break
		}
	}

	next := (current + direction + len(panes)) % len(panes)
	m.setFocus(panes[next])
}

func (m *Model) setFocus(pane Pane) {
	// Blur current
	switch m.focusedPane {
	case PaneSidebar:
		m.sidebar.Blur()
	case PaneEditor:
		if ts := m.activeTabState(); ts != nil {
			ts.Editor.Blur()
		}
	case PaneResults:
		if ts := m.activeTabState(); ts != nil {
			ts.Results.Blur()
		}
	}

	m.focusedPane = pane

	// Focus new
	switch pane {
	case PaneSidebar:
		m.sidebar.Focus()
	case PaneEditor:
		if ts := m.activeTabState(); ts != nil {
			ts.Editor.Focus()
		}
	case PaneResults:
		if ts := m.activeTabState(); ts != nil {
			ts.Results.Focus()
		}
	}
}

func (m Model) activeTabState() *TabState {
	return m.tabStates[m.tabs.ActiveID()]
}

func (m *Model) connect(adapterName, dsn string) tea.Cmd {
	return func() tea.Msg {
		a, ok := adapter.Registry[adapterName]
		if !ok {
			return ConnectErrMsg{Err: fmt.Errorf("unknown adapter: %s", adapterName)}
		}
		ctx := context.Background()
		conn, err := a.Connect(ctx, dsn)
		if err != nil {
			return ConnectErrMsg{Err: err}
		}
		if err := conn.Ping(ctx); err != nil {
			conn.Close()
			return ConnectErrMsg{Err: err}
		}
		return ConnectMsg{Conn: conn, Adapter: adapterName, DSN: dsn}
	}
}

func (m *Model) renderHelpScreen(th *theme.Theme) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#569CD6")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#DCDCAA")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#CE9178"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D4D4D4"))

	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6A9955"))

	line := func(key, desc string) string {
		return fmt.Sprintf("  %s  %s", keyStyle.Render(fmt.Sprintf("%-16s", key)), descStyle.Render(desc))
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("  gotermsql - Keyboard Shortcuts"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("  Query"))
	b.WriteString("\n")
	b.WriteString(line("F5 / Ctrl+G", "Execute query"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+C", "Cancel running query"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+Space", "Trigger autocomplete"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+E", "Export results"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("  Navigation"))
	b.WriteString("\n")
	b.WriteString(line("Tab", "Next pane"))
	b.WriteString("\n")
	b.WriteString(line("Shift+Tab", "Previous pane"))
	b.WriteString("\n")
	b.WriteString(line("Alt+1 / 2 / 3", "Jump to sidebar / editor / results"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("  Tabs"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+T", "New tab"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+W", "Close tab"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+] / Ctrl+[", "Next / previous tab"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("  Application"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+O", "Connection manager"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+B", "Toggle sidebar"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+R", "Refresh schema"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+H", "Query history"))
	b.WriteString("\n")
	b.WriteString(line("F2", "Toggle vim / standard mode"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+Q", "Quit"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("  Resize Panes"))
	b.WriteString("\n")
	b.WriteString(line("Ctrl+Arrow keys", "Resize sidebar / editor split"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("  Sidebar"))
	b.WriteString("\n")
	b.WriteString(line("Enter / Right", "Expand node / open table"))
	b.WriteString("\n")
	b.WriteString(line("Left", "Collapse node"))
	b.WriteString("\n")
	b.WriteString(line("Up / Down", "Navigate"))
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("  Press ? / F1 / Esc to close"))

	return th.DialogBorder.Render(b.String())
}

func (m *Model) loadSchema() tea.Cmd {
	conn := m.conn
	gen := m.connGen
	return func() tea.Msg {
		if conn == nil {
			return SchemaErrMsg{Err: adapter.ErrNotConnected, ConnGen: gen}
		}
		ctx := context.Background()
		dbs, err := conn.Databases(ctx)
		if err != nil {
			return SchemaErrMsg{Err: err, ConnGen: gen}
		}

		// Load full schema for each database
		var databases []schema.Database
		for _, db := range dbs {
			for si := range db.Schemas {
				s := &db.Schemas[si]
				for ti := range s.Tables {
					t := &s.Tables[ti]
					cols, err := conn.Columns(ctx, db.Name, s.Name, t.Name)
					if err == nil {
						t.Columns = cols
					}
					idxs, err := conn.Indexes(ctx, db.Name, s.Name, t.Name)
					if err == nil {
						t.Indexes = idxs
					}
					fks, err := conn.ForeignKeys(ctx, db.Name, s.Name, t.Name)
					if err == nil {
						t.FKs = fks
					}
				}
			}
			databases = append(databases, db)
		}

		return SchemaLoadedMsg{Databases: databases, ConnGen: gen}
	}
}

func (m *Model) executeQuery(query string, tabID int) tea.Cmd {
	conn := m.conn
	ts := m.tabStates[tabID]
	if ts == nil {
		return nil
	}
	ts.Query = query
	ts.RunID++
	runID := ts.RunID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	m.cancelFunc = cancel

	return tea.Batch(
		func() tea.Msg { return QueryStartedMsg{TabID: tabID, RunID: runID} },
		func() tea.Msg {
			defer cancel()
			if conn == nil {
				return QueryErrMsg{Err: adapter.ErrNotConnected, TabID: tabID, RunID: runID}
			}

			result, err := conn.Execute(ctx, query)
			if err != nil {
				return QueryErrMsg{Err: err, TabID: tabID, RunID: runID}
			}
			return QueryResultMsg{Result: result, TabID: tabID, RunID: runID}
		},
	)
}

// SetConnection sets the initial database connection.
func (m *Model) SetConnection(conn adapter.Connection, adapterName, dsn string) {
	m.conn = conn
}

// Connection returns the current database connection, or nil if not connected.
func (m Model) Connection() adapter.Connection {
	return m.conn
}

// InitialConnect returns a command to connect on startup.
func (m Model) InitialConnect(adapterName, dsn string) tea.Cmd {
	return m.connect(adapterName, dsn)
}

// ShowConnManager shows the connection manager on startup.
func (m *Model) ShowConnManager() {
	m.connMgr.Show()
}

func (m *Model) exportResults() tea.Cmd {
	ts := m.activeTabState()
	if ts == nil {
		return nil
	}
	cols := ts.Results.Columns()
	rows := ts.Results.Rows()
	if len(cols) == 0 || len(rows) == 0 {
		return func() tea.Msg {
			return ExportErrMsg{Err: fmt.Errorf("no results to export")}
		}
	}

	return func() tea.Msg {
		dir, err := os.Getwd()
		if err != nil {
			return ExportErrMsg{Err: err}
		}
		path := filepath.Join(dir, fmt.Sprintf("export_%s.csv", time.Now().Format("20060102_150405")))
		if err := results.ExportCSV(path, cols, rows); err != nil {
			return ExportErrMsg{Err: err}
		}
		return ExportCompleteMsg{Path: path, RowCount: int64(len(rows))}
	}
}

func isTypingKey(msg tea.KeyMsg) bool {
	s := msg.String()
	if len(s) == 1 && s[0] >= 32 && s[0] <= 126 {
		return true
	}
	return s == "backspace" || s == "delete"
}
