package connmgr

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/config"
	"github.com/sadopc/gotermsql/internal/theme"
)

// State tracks the connection manager screen.
type State int

const (
	StateList State = iota
	StateForm
	StateTesting
)

// ConnectRequestMsg is sent when the user picks a connection.
type ConnectRequestMsg struct {
	AdapterName string
	DSN         string
}

// ConnectionsUpdatedMsg is sent when saved connections are modified.
type ConnectionsUpdatedMsg struct {
	Connections []config.SavedConnection
}

// Model is the connection manager modal.
type Model struct {
	state       State
	connections []config.SavedConnection
	cursor      int
	visible     bool
	width       int
	height      int

	// Form fields
	inputs    []textinput.Model
	formFocus int
	editing   int // index of connection being edited, -1 for new
	message   string
	isError   bool
}

const (
	fieldName = iota
	fieldAdapter
	fieldHost
	fieldPort
	fieldUser
	fieldPassword
	fieldDatabase
	fieldDSN
	fieldCount
)

// New creates a new connection manager.
func New(connections []config.SavedConnection) Model {
	m := Model{
		connections: connections,
		editing:     -1,
	}
	m.initForm()
	return m
}

func (m *Model) initForm() {
	m.inputs = make([]textinput.Model, fieldCount)

	labels := []string{"Name", "Adapter", "Host", "Port", "User", "Password", "Database", "DSN"}
	placeholders := []string{
		"my-database",
		"postgres|mysql|sqlite|duckdb",
		"localhost",
		"5432",
		"",
		"",
		"",
		"postgres://user:pass@host:5432/db",
	}

	for i := range m.inputs {
		t := textinput.New()
		t.Prompt = labels[i] + ": "
		t.Placeholder = placeholders[i]
		if i == fieldPassword {
			t.EchoMode = textinput.EchoPassword
		}
		t.Width = 40
		m.inputs[i] = t
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles connection manager messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch m.state {
	case StateList:
		return m.updateList(msg)
	case StateForm:
		return m.updateForm(msg)
	case StateTesting:
		return m.updateTesting(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.connections) {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.connections) {
				conn := m.connections[m.cursor]
				dsn := conn.DSN
				if dsn == "" {
					dsn = conn.BuildDSN()
				}
				m.visible = false
				return m, func() tea.Msg {
					return ConnectRequestMsg{
						AdapterName: conn.Adapter,
						DSN:         dsn,
					}
				}
			}
		case "n":
			m.state = StateForm
			m.editing = -1
			m.clearForm()
			m.inputs[0].Focus()
			return m, textinput.Blink
		case "e":
			if m.cursor < len(m.connections) {
				m.state = StateForm
				m.editing = m.cursor
				m.loadIntoForm(m.connections[m.cursor])
				m.inputs[0].Focus()
				return m, textinput.Blink
			}
		case "d":
			if m.cursor < len(m.connections) {
				m.connections = append(m.connections[:m.cursor], m.connections[m.cursor+1:]...)
				if m.cursor >= len(m.connections) && m.cursor > 0 {
					m.cursor--
				}
				conns := make([]config.SavedConnection, len(m.connections))
				copy(conns, m.connections)
				return m, func() tea.Msg { return ConnectionsUpdatedMsg{Connections: conns} }
			}
		case "esc", "q":
			m.visible = false
		}
	}
	return m, nil
}

func (m Model) updateForm(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = StateList
			return m, nil
		case "tab", "down":
			m.inputs[m.formFocus].Blur()
			m.formFocus = (m.formFocus + 1) % fieldCount
			m.inputs[m.formFocus].Focus()
			return m, textinput.Blink
		case "shift+tab", "up":
			m.inputs[m.formFocus].Blur()
			m.formFocus--
			if m.formFocus < 0 {
				m.formFocus = fieldCount - 1
			}
			m.inputs[m.formFocus].Focus()
			return m, textinput.Blink
		case "ctrl+s":
			conn := m.formToConnection()
			if m.editing >= 0 && m.editing < len(m.connections) {
				m.connections[m.editing] = conn
			} else {
				m.connections = append(m.connections, conn)
			}
			m.state = StateList
			conns := make([]config.SavedConnection, len(m.connections))
			copy(conns, m.connections)
			return m, func() tea.Msg { return ConnectionsUpdatedMsg{Connections: conns} }
		case "ctrl+t":
			m.state = StateTesting
			conn := m.formToConnection()
			return m, m.testConnection(conn)
		}
	}

	// Update focused input
	var cmd tea.Cmd
	m.inputs[m.formFocus], cmd = m.inputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m Model) updateTesting(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case testResultMsg:
		if msg.err != nil {
			m.message = "Connection failed: " + sanitizeError(msg.err.Error())
			m.isError = true
		} else {
			m.message = "Connection successful!"
			m.isError = false
		}
		m.state = StateForm
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.state = StateForm
		}
	}
	return m, nil
}

type testResultMsg struct{ err error }

func (m Model) testConnection(conn config.SavedConnection) tea.Cmd {
	return func() tea.Msg {
		dsn := conn.DSN
		if dsn == "" {
			dsn = conn.BuildDSN()
		}
		a, ok := adapter.Registry[conn.Adapter]
		if !ok {
			return testResultMsg{err: fmt.Errorf("unknown adapter: %s", conn.Adapter)}
		}
		ctx := context.Background()
		c, err := a.Connect(ctx, dsn)
		if err != nil {
			return testResultMsg{err: err}
		}
		err = c.Ping(ctx)
		c.Close()
		return testResultMsg{err: err}
	}
}

// View renders the connection manager.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	th := theme.Current

	switch m.state {
	case StateList:
		return m.viewList(th)
	case StateForm:
		return m.viewForm(th)
	case StateTesting:
		return th.DialogBorder.Render("\n  Testing connection...\n")
	}
	return ""
}

func (m Model) viewList(th *theme.Theme) string {
	title := th.DialogTitle.Render("  Connection Manager  ")

	var lines []string
	for i, conn := range m.connections {
		line := fmt.Sprintf("  %s  (%s)", conn.Name, conn.DisplayString())
		if i == m.cursor {
			lines = append(lines, th.SidebarSelected.Render(line))
		} else {
			lines = append(lines, "  "+line)
		}
	}

	// "New connection" option
	newLine := "  + New Connection"
	if m.cursor == len(m.connections) {
		lines = append(lines, th.SidebarSelected.Render(newLine))
	} else {
		lines = append(lines, "  "+newLine)
	}

	help := th.MutedText.Render("  enter:connect  n:new  e:edit  d:delete  esc:close")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		strings.Join(lines, "\n"),
		"",
		help,
	)

	return th.DialogBorder.Width(m.dialogWidth()).Render(content)
}

func (m Model) viewForm(th *theme.Theme) string {
	title := "  New Connection  "
	if m.editing >= 0 {
		title = "  Edit Connection  "
	}

	var lines []string
	lines = append(lines, th.DialogTitle.Render(title))
	lines = append(lines, "")

	for i := range m.inputs {
		lines = append(lines, "  "+m.inputs[i].View())
	}

	if m.message != "" {
		if m.isError {
			lines = append(lines, "", th.ErrorText.Render("  "+m.message))
		} else {
			lines = append(lines, "", th.SuccessText.Render("  "+m.message))
		}
	}

	lines = append(lines, "")
	lines = append(lines, th.MutedText.Render("  ctrl+s:save  ctrl+t:test  esc:back"))

	content := strings.Join(lines, "\n")
	return th.DialogBorder.Width(m.dialogWidth()).Render(content)
}

func (m Model) dialogWidth() int {
	w := 60
	if m.width > 0 && w > m.width-4 {
		w = m.width - 4
	}
	return w
}

func (m *Model) clearForm() {
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}
	m.formFocus = 0
	m.message = ""
}

func (m *Model) loadIntoForm(conn config.SavedConnection) {
	m.inputs[fieldName].SetValue(conn.Name)
	m.inputs[fieldAdapter].SetValue(conn.Adapter)
	m.inputs[fieldHost].SetValue(conn.Host)
	if conn.Port > 0 {
		m.inputs[fieldPort].SetValue(fmt.Sprintf("%d", conn.Port))
	}
	m.inputs[fieldUser].SetValue(conn.User)
	m.inputs[fieldPassword].SetValue(conn.Password)
	m.inputs[fieldDatabase].SetValue(conn.Database)
	m.inputs[fieldDSN].SetValue(conn.DSN)
	m.formFocus = 0
	m.message = ""
}

func (m Model) formToConnection() config.SavedConnection {
	port := 0
	fmt.Sscanf(m.inputs[fieldPort].Value(), "%d", &port)
	return config.SavedConnection{
		Name:     m.inputs[fieldName].Value(),
		Adapter:  m.inputs[fieldAdapter].Value(),
		Host:     m.inputs[fieldHost].Value(),
		Port:     port,
		User:     m.inputs[fieldUser].Value(),
		Password: m.inputs[fieldPassword].Value(),
		Database: m.inputs[fieldDatabase].Value(),
		DSN:      m.inputs[fieldDSN].Value(),
	}
}

// Show makes the connection manager visible.
func (m *Model) Show() {
	m.visible = true
	m.state = StateList
	m.cursor = 0
}

// Hide hides the connection manager.
func (m *Model) Hide() {
	m.visible = false
}

// Visible returns whether the connection manager is shown.
func (m Model) Visible() bool { return m.visible }

// SetSize sets the available space.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Connections returns the current saved connections.
func (m Model) Connections() []config.SavedConnection {
	return m.connections
}

// SetConnections updates the saved connections list.
func (m *Model) SetConnections(conns []config.SavedConnection) {
	m.connections = conns
}

// sanitizeError strips credentials from error messages that may contain DSN URLs.
func sanitizeError(msg string) string {
	for _, prefix := range []string{"postgres://", "postgresql://", "mysql://", "duckdb://"} {
		for {
			idx := strings.Index(msg, prefix)
			if idx < 0 {
				break
			}
			rest := msg[idx+len(prefix):]
			atIdx := strings.Index(rest, "@")
			if atIdx < 0 {
				break
			}
			msg = msg[:idx+len(prefix)] + "***" + msg[idx+len(prefix)+atIdx:]
		}
	}
	return msg
}
