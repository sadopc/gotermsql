package tabs

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/theme"
)

// Tab represents a single query tab.
type Tab struct {
	ID       int
	Title    string
	Modified bool
}

// Model is the tab bar component.
type Model struct {
	tabs      []Tab
	active    int
	nextID    int
	width     int
}

// New creates a new tab bar with one default tab.
func New() Model {
	m := Model{
		nextID: 1,
	}
	m.tabs = []Tab{{ID: 0, Title: "Query 1"}}
	return m
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles tab bar messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case appmsg.NewTabMsg:
		tab := Tab{
			ID:    m.nextID,
			Title: fmt.Sprintf("Query %d", m.nextID+1),
		}
		m.nextID++
		m.tabs = append(m.tabs, tab)
		m.active = len(m.tabs) - 1
		return m, func() tea.Msg { return appmsg.SwitchTabMsg{TabID: tab.ID} }

	case appmsg.CloseTabMsg:
		if len(m.tabs) <= 1 {
			return m, nil // don't close last tab
		}
		idx := m.indexByID(msg.TabID)
		if idx < 0 {
			return m, nil
		}
		m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
		if m.active >= len(m.tabs) {
			m.active = len(m.tabs) - 1
		}
		return m, func() tea.Msg { return appmsg.SwitchTabMsg{TabID: m.tabs[m.active].ID} }

	case appmsg.SwitchTabMsg:
		idx := m.indexByID(msg.TabID)
		if idx >= 0 {
			m.active = idx
		}
	}

	return m, nil
}

// View renders the tab bar.
func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	th := theme.Current

	var tabs []string
	for i, tab := range m.tabs {
		title := tab.Title
		if tab.Modified {
			title += " *"
		}

		var style lipgloss.Style
		if i == m.active {
			style = th.TabActive
		} else {
			style = th.TabInactive
		}
		tabs = append(tabs, style.Render(title))
	}

	newTabBtn := th.TabInactive.Render(" + ")
	tabs = append(tabs, newTabBtn)

	bar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	return th.TabBar.Width(m.width).Render(bar)
}

// SetSize sets the tab bar width.
func (m *Model) SetSize(width int) {
	m.width = width
}

// ActiveTab returns the active tab.
func (m Model) ActiveTab() Tab {
	if m.active < len(m.tabs) {
		return m.tabs[m.active]
	}
	return Tab{}
}

// ActiveID returns the active tab ID.
func (m Model) ActiveID() int {
	return m.ActiveTab().ID
}

// SetModified marks a tab as modified.
func (m *Model) SetModified(tabID int, modified bool) {
	idx := m.indexByID(tabID)
	if idx >= 0 {
		m.tabs[idx].Modified = modified
	}
}

// NextTab switches to the next tab.
func (m *Model) NextTab() tea.Cmd {
	if len(m.tabs) == 0 {
		return nil
	}
	m.active = (m.active + 1) % len(m.tabs)
	id := m.tabs[m.active].ID
	return func() tea.Msg { return appmsg.SwitchTabMsg{TabID: id} }
}

// PrevTab switches to the previous tab.
func (m *Model) PrevTab() tea.Cmd {
	if len(m.tabs) == 0 {
		return nil
	}
	m.active--
	if m.active < 0 {
		m.active = len(m.tabs) - 1
	}
	id := m.tabs[m.active].ID
	return func() tea.Msg { return appmsg.SwitchTabMsg{TabID: id} }
}

// Tabs returns all tabs.
func (m Model) Tabs() []Tab {
	return m.tabs
}

// Count returns the number of tabs.
func (m Model) Count() int {
	return len(m.tabs)
}

func (m Model) indexByID(id int) int {
	for i, t := range m.tabs {
		if t.ID == id {
			return i
		}
	}
	return -1
}
