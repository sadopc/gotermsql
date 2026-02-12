package dialog

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gotermsql/internal/theme"
)

// Button represents a dialog button.
type Button struct {
	Label  string
	Action func() tea.Msg
}

// Model is a reusable modal dialog component.
type Model struct {
	title     string
	body      string
	buttons   []Button
	active    int
	visible   bool
	width     int
	maxWidth  int
	maxHeight int
}

// New creates a new dialog.
func New(title, body string, buttons ...Button) Model {
	return Model{
		title:    title,
		body:     body,
		buttons:  buttons,
		maxWidth: 60,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles dialog messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "shift+tab":
			if m.active > 0 {
				m.active--
			}
		case "right", "tab":
			if m.active < len(m.buttons)-1 {
				m.active++
			}
		case "enter":
			if m.active < len(m.buttons) && m.buttons[m.active].Action != nil {
				m.visible = false
				return m, m.buttons[m.active].Action
			}
		case "esc":
			m.visible = false
		}
	}

	return m, nil
}

// View renders the dialog as a centered overlay.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	th := theme.Current

	// Title
	title := th.DialogTitle.Render(m.title)

	// Body
	body := lipgloss.NewStyle().
		Width(m.maxWidth - 4).
		Render(m.body)

	// Buttons
	var btns []string
	for i, btn := range m.buttons {
		var style lipgloss.Style
		if i == m.active {
			style = th.DialogButtonActive
		} else {
			style = th.DialogButton
		}
		btns = append(btns, style.Render(" "+btn.Label+" "))
	}
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, btns...)
	buttonRow = lipgloss.NewStyle().Width(m.maxWidth - 4).Align(lipgloss.Center).Render(buttonRow)

	// Compose
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		body,
		"",
		buttonRow,
	)

	return th.DialogBorder.Render(content)
}

// Show makes the dialog visible.
func (m *Model) Show() {
	m.visible = true
	m.active = 0
}

// Hide makes the dialog invisible.
func (m *Model) Hide() {
	m.visible = false
}

// Visible returns whether the dialog is shown.
func (m Model) Visible() bool {
	return m.visible
}

// SetSize sets the available space for centering.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.maxHeight = height
	if m.maxWidth > width-4 {
		m.maxWidth = width - 4
	}
}

// Overlay renders the dialog centered over the given background content.
func (m Model) Overlay(background string) string {
	if !m.visible {
		return background
	}

	dialog := m.View()
	bgLines := strings.Split(background, "\n")
	dlgLines := strings.Split(dialog, "\n")

	bgH := len(bgLines)
	dlgH := len(dlgLines)
	dlgW := lipgloss.Width(dialog)

	startY := (bgH - dlgH) / 2
	startX := (m.width - dlgW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	for i, dlgLine := range dlgLines {
		y := startY + i
		if y >= bgH {
			break
		}
		line := bgLines[y]
		// Overlay dialog line onto background
		lineRunes := []rune(line)
		prefix := ""
		if startX < len(lineRunes) {
			prefix = string(lineRunes[:startX])
		} else {
			prefix = line + strings.Repeat(" ", startX-len(lineRunes))
		}
		suffix := ""
		endX := startX + lipgloss.Width(dlgLine)
		if endX < len(lineRunes) {
			suffix = string(lineRunes[endX:])
		}
		bgLines[y] = prefix + dlgLine + suffix
	}

	return strings.Join(bgLines, "\n")
}
