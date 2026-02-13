package historybrowser

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gotermsql/internal/history"
	"github.com/sadopc/gotermsql/internal/theme"
)

// SelectQueryMsg is sent when the user picks a history entry.
type SelectQueryMsg struct {
	Query string
}

// Model is the history browser modal.
type Model struct {
	hist    *history.History
	entries []history.HistoryEntry
	cursor  int
	offset  int // scroll offset
	visible bool
	width   int
	height  int
	search  textinput.Model
}

// New creates a new history browser.
func New(hist *history.History) Model {
	ti := textinput.New()
	ti.Placeholder = "Search queries..."
	ti.Prompt = "  > "
	ti.Width = 50
	return Model{
		hist:   hist,
		search: ti,
	}
}

// Show makes the history browser visible and loads entries.
func (m *Model) Show() {
	m.visible = true
	m.cursor = 0
	m.offset = 0
	m.search.SetValue("")
	m.search.Focus()
	m.loadEntries()
}

// Hide hides the history browser.
func (m *Model) Hide() {
	m.visible = false
	m.search.Blur()
}

// Visible returns whether the history browser is shown.
func (m Model) Visible() bool { return m.visible }

// SetSize sets the available space.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles history browser messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+h":
			m.visible = false
			m.search.Blur()
			return m, nil
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
				m.ensureVisible()
			}
			return m, nil
		case "pgup":
			visible := m.visibleCount()
			m.cursor -= visible
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
			return m, nil
		case "pgdown":
			visible := m.visibleCount()
			m.cursor += visible
			if m.cursor >= len(m.entries) {
				m.cursor = len(m.entries) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
			return m, nil
		case "enter":
			if m.cursor < len(m.entries) {
				query := m.entries[m.cursor].Query
				m.visible = false
				m.search.Blur()
				return m, func() tea.Msg {
					return SelectQueryMsg{Query: query}
				}
			}
			return m, nil
		}

		// Delegate all other keys to the search input
		prevVal := m.search.Value()
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		if m.search.Value() != prevVal {
			m.cursor = 0
			m.offset = 0
			m.loadEntries()
		}
		return m, cmd
	}

	// Non-key messages (e.g. blink)
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

// View renders the history browser.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	th := theme.Current
	w := m.dialogWidth()

	title := th.DialogTitle.Render("  Query History  ")
	searchView := "  " + m.search.View()

	visible := m.visibleCount()
	var lines []string
	end := m.offset + visible
	if end > len(m.entries) {
		end = len(m.entries)
	}

	for i := m.offset; i < end; i++ {
		e := m.entries[i]
		line := m.formatEntry(e, w-6)
		if i == m.cursor {
			lines = append(lines, th.SidebarSelected.Render(line))
		} else if e.IsError {
			lines = append(lines, th.ErrorText.Render("  "+line))
		} else {
			lines = append(lines, "  "+line)
		}
	}

	if len(m.entries) == 0 {
		lines = append(lines, th.MutedText.Render("  No history entries"))
	}

	countText := fmt.Sprintf("  %d entries", len(m.entries))
	help := th.MutedText.Render("  enter:select  esc:close  up/down:navigate")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		searchView,
		"",
		strings.Join(lines, "\n"),
		"",
		th.MutedText.Render(countText),
		help,
	)

	return th.DialogBorder.Width(w).Render(content)
}

func (m Model) dialogWidth() int {
	w := 80
	if m.width > 0 && w > m.width-4 {
		w = m.width - 4
	}
	return w
}

// visibleCount returns how many entries fit in the visible area.
func (m Model) visibleCount() int {
	// Title + search + blank + blank + count + help = 6 lines of chrome
	// Plus 2 for border
	avail := m.height - 8
	if avail < 3 {
		avail = 3
	}
	return avail
}

func (m *Model) ensureVisible() {
	visible := m.visibleCount()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
}

func (m *Model) loadEntries() {
	if m.hist == nil {
		m.entries = nil
		return
	}

	var err error
	searchText := m.search.Value()
	if searchText != "" {
		m.entries, err = m.hist.Search("%"+searchText+"%", 200)
	} else {
		m.entries, err = m.hist.Recent(200)
	}
	if err != nil {
		m.entries = nil
	}
}

func (m Model) formatEntry(e history.HistoryEntry, maxWidth int) string {
	// First line of query, truncated
	query := firstLine(e.Query)
	queryMax := maxWidth - 30 // leave room for metadata
	if queryMax < 10 {
		queryMax = 10
	}
	if len(query) > queryMax {
		query = query[:queryMax-3] + "..."
	}

	// Metadata
	var meta []string
	if e.Adapter != "" {
		meta = append(meta, e.Adapter)
	}
	if e.DurationMS > 0 {
		meta = append(meta, formatDuration(e.DurationMS))
	}
	meta = append(meta, RelativeTime(e.ExecutedAt))

	return fmt.Sprintf("%-*s  %s", queryMax, query, strings.Join(meta, " | "))
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// RelativeTime formats a timestamp as a human-readable relative time.
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 48*time.Hour:
		return "yesterday"
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}
