package statusbar

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/theme"
)

// ClearStatusMsg is sent after a timeout to revert the status bar to key hints.
type ClearStatusMsg struct{}

// Model is the status bar component.
type Model struct {
	width        int
	adapterName  string
	databaseName string
	dsn          string
	queryTime    time.Duration
	rowCount     int64
	keyMode      appmsg.KeyMode
	vimState     appmsg.VimState
	message      string
	isError      bool
	cursorLine   int
	cursorCol    int
	connected    bool
}

// New creates a new status bar.
func New() Model {
	return Model{
		rowCount: -1,
		keyMode:  appmsg.KeyModeStandard,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles status bar messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	clearAfter := func() tea.Cmd {
		return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
			return ClearStatusMsg{}
		})
	}

	switch msg := msg.(type) {
	case appmsg.ConnectMsg:
		m.adapterName = msg.Adapter
		m.dsn = msg.DSN
		m.databaseName = msg.Conn.DatabaseName()
		m.connected = true
		m.message = ""
		m.isError = false

	case appmsg.DisconnectMsg:
		m.connected = false
		m.adapterName = ""
		m.databaseName = ""
		m.dsn = ""

	case appmsg.QueryResultMsg:
		if msg.Result != nil {
			m.queryTime = msg.Result.Duration
			m.rowCount = msg.Result.RowCount
			if msg.Result.Message != "" {
				m.message = msg.Result.Message
				m.isError = false
			}
		}
		return m, clearAfter()

	case appmsg.QueryStreamingMsg:
		m.queryTime = msg.Duration
		m.rowCount = -1
		m.message = "streaming"
		m.isError = false
		return m, clearAfter()

	case appmsg.QueryErrMsg:
		if msg.Err != nil {
			m.message = msg.Err.Error()
		} else {
			m.message = "unknown error"
		}
		m.isError = true
		return m, clearAfter()

	case appmsg.StatusMsg:
		m.message = msg.Text
		m.isError = msg.IsError
		if msg.Duration > 0 {
			m.queryTime = msg.Duration
		}
		return m, clearAfter()

	case ClearStatusMsg:
		m.queryTime = 0
		m.rowCount = -1
		m.message = ""
		m.isError = false

	case appmsg.ToggleKeyModeMsg:
		if m.keyMode == appmsg.KeyModeStandard {
			m.keyMode = appmsg.KeyModeVim
		} else {
			m.keyMode = appmsg.KeyModeStandard
		}
	}

	return m, nil
}

// View renders the status bar.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	th := theme.Current

	// Left section: connection info
	var left string
	if m.connected {
		connStr := fmt.Sprintf(" %s://%s ", m.adapterName, m.databaseName)
		left = th.StatusBarKey.Render(connStr)
	} else {
		left = th.StatusBarKey.Render(" disconnected ")
	}

	// Center section: query time + row count or message or key hints
	var center string
	if m.message != "" {
		if m.isError {
			center = th.StatusBarError.Render(" " + truncate(m.message, m.width/2) + " ")
		} else {
			center = th.StatusBarSuccess.Render(" " + m.message + " ")
		}
	} else if m.queryTime > 0 {
		timeStr := formatDuration(m.queryTime)
		center = th.StatusBarValue.Render(fmt.Sprintf(" %s ", timeStr))
		if m.rowCount >= 0 {
			center += th.StatusBarValue.Render(fmt.Sprintf(" %s rows ", formatCount(m.rowCount)))
		}
	} else {
		// Show key hints when idle
		hintKey := th.StatusBarValue
		hintSep := th.StatusBar
		center = hintKey.Render("F5") +
			hintSep.Render(" Run ") +
			hintKey.Render("Ctrl+Q") +
			hintSep.Render(" Quit ") +
			hintKey.Render("Tab") +
			hintSep.Render(" Switch pane ") +
			hintKey.Render("F1") +
			hintSep.Render(" Help ")
	}

	// Right section: key mode + cursor position
	modeStr := fmt.Sprintf(" %s ", m.keyMode)
	if m.keyMode == appmsg.KeyModeVim {
		modeStr = fmt.Sprintf(" %s:%s ", m.keyMode, m.vimState)
	}
	right := th.StatusBarKey.Render(modeStr)
	if m.cursorLine > 0 {
		right += th.StatusBarValue.Render(fmt.Sprintf(" %d:%d ", m.cursorLine, m.cursorCol))
	}

	// Calculate spacing
	leftW := lipgloss.Width(left)
	centerW := lipgloss.Width(center)
	rightW := lipgloss.Width(right)
	gap := m.width - leftW - centerW - rightW
	if gap < 0 {
		gap = 0
	}

	leftGap := gap / 2
	rightGap := gap - leftGap

	bar := left +
		th.StatusBar.Render(spaces(leftGap)) +
		center +
		th.StatusBar.Render(spaces(rightGap)) +
		right

	return th.StatusBar.Width(m.width).Render(bar)
}

// SetSize sets the status bar width.
func (m *Model) SetSize(width int) {
	m.width = width
}

// SetCursor updates the cursor position display.
func (m *Model) SetCursor(line, col int) {
	m.cursorLine = line
	m.cursorCol = col
}

// SetVimState updates the vim state display.
func (m *Model) SetVimState(state appmsg.VimState) {
	m.vimState = state
}

// KeyMode returns the current key mode.
func (m Model) KeyMode() appmsg.KeyMode {
	return m.keyMode
}

// SetKeyMode sets the key mode.
func (m *Model) SetKeyMode(mode appmsg.KeyMode) {
	m.keyMode = mode
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func formatCount(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
