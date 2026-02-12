package autocomplete

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/completion"
	"github.com/sadopc/gotermsql/internal/theme"
)

const maxVisible = 5

// SelectedMsg is sent when an autocomplete item is selected.
type SelectedMsg struct {
	Text string
}

// DismissMsg is sent when autocomplete is dismissed.
type DismissMsg struct{}

// Model is the autocomplete dropdown overlay.
type Model struct {
	items    []adapter.CompletionItem
	filtered []adapter.CompletionItem
	selected int
	visible  bool
	prefix   string // current word prefix being completed
	engine   *completion.Engine
	posX     int // cursor X position for overlay placement
	posY     int // cursor Y position
	width    int
}

// New creates a new autocomplete model.
func New(engine *completion.Engine) Model {
	return Model{
		engine: engine,
		width:  40,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles autocomplete interactions.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case "down", "ctrl+n":
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
			return m, nil

		case "enter", "tab":
			if m.selected < len(m.filtered) {
				item := m.filtered[m.selected]
				text := item.Label
				// Remove the prefix already typed
				if len(m.prefix) > 0 && strings.HasPrefix(strings.ToLower(text), strings.ToLower(m.prefix)) {
					text = text[len(m.prefix):]
				}
				m.visible = false
				return m, func() tea.Msg { return SelectedMsg{Text: text} }
			}

		case "esc", "ctrl+c":
			m.visible = false
			return m, func() tea.Msg { return DismissMsg{} }
		}
	}

	return m, nil
}

// View renders the autocomplete dropdown.
func (m Model) View() string {
	if !m.visible || len(m.filtered) == 0 {
		return ""
	}

	th := theme.Current

	visible := m.filtered
	offset := 0
	if len(visible) > maxVisible {
		if m.selected >= maxVisible {
			offset = m.selected - maxVisible + 1
		}
		end := offset + maxVisible
		if end > len(visible) {
			end = len(visible)
		}
		visible = visible[offset:end]
	}

	var lines []string
	for i, item := range visible {
		idx := offset + i
		icon := kindIcon(item.Kind)
		label := icon + " " + item.Label
		if item.Detail != "" {
			label += "  " + item.Detail
		}
		// Truncate to width
		if len(label) > m.width-2 {
			label = label[:m.width-5] + "..."
		}
		// Pad to width
		for len(label) < m.width-2 {
			label += " "
		}

		if idx == m.selected {
			lines = append(lines, th.AutocompleteSelected.Render(label))
		} else {
			lines = append(lines, th.AutocompleteItem.Render(label))
		}
	}

	content := strings.Join(lines, "\n")
	return th.AutocompleteBorder.Render(content)
}

// Trigger computes completions for the given text and cursor position.
func (m *Model) Trigger(text string, cursorPos int) {
	if m.engine == nil {
		return
	}

	items := m.engine.Complete(text, cursorPos)
	if len(items) == 0 {
		m.visible = false
		return
	}

	m.filtered = items
	m.selected = 0
	m.visible = true

	// Extract prefix
	m.prefix = extractPrefix(text, cursorPos)
}

// TriggerForced forces autocomplete display (Ctrl+Space).
func (m *Model) TriggerForced(text string, cursorPos int) {
	m.Trigger(text, cursorPos)
}

// Dismiss hides the autocomplete.
func (m *Model) Dismiss() {
	m.visible = false
}

// Visible returns whether autocomplete is shown.
func (m Model) Visible() bool {
	return m.visible
}

// SetPosition sets the overlay position hint.
func (m *Model) SetPosition(x, y int) {
	m.posX = x
	m.posY = y
}

// SetEngine sets the completion engine.
func (m *Model) SetEngine(engine *completion.Engine) {
	m.engine = engine
}

func extractPrefix(text string, cursorPos int) string {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}
	before := text[:cursorPos]
	// Walk backward to find word start
	i := len(before) - 1
	for i >= 0 && !isWordBreak(before[i]) {
		i--
	}
	return before[i+1:]
}

func isWordBreak(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '(' || b == ')' ||
		b == ',' || b == ';' || b == '.' || b == '=' || b == '<' || b == '>'
}

func kindIcon(k adapter.CompletionKind) string {
	switch k {
	case adapter.CompletionTable:
		return "T"
	case adapter.CompletionColumn:
		return "C"
	case adapter.CompletionKeyword:
		return "K"
	case adapter.CompletionFunction:
		return "F"
	case adapter.CompletionSchema:
		return "S"
	case adapter.CompletionDatabase:
		return "D"
	case adapter.CompletionView:
		return "V"
	default:
		return " "
	}
}
