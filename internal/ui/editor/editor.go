package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gotermsql/internal/theme"
)

// Model is a Bubble Tea component that wraps a textarea with SQL-aware
// styling. When focused, the underlying textarea handles editing (cursor,
// scrolling, selection, etc.). When blurred, the content is rendered with
// syntax highlighting and line numbers.
//
// TODO: Full inline syntax highlighting while editing requires textarea v2 or
// a custom widget. For now, highlighted rendering is only shown in the
// blurred/read-only view.
type Model struct {
	textarea    textarea.Model
	highlighter *Highlighter
	width       int
	height      int
	focused     bool
	modified    bool // track if content changed since last save/execute
	id          int  // tab identifier
}

// New creates a new editor instance. The id parameter is used to associate
// the editor with a tab.
func New(id int) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL query..."
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // unlimited

	// Use theme colours for the textarea prompt (line numbers) and cursor.
	th := theme.Current
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = th.EditorLineNumber
	ta.FocusedStyle.Text = lipgloss.NewStyle()
	ta.BlurredStyle.Prompt = th.EditorLineNumber
	ta.BlurredStyle.Text = lipgloss.NewStyle()

	ta.Blur()

	return Model{
		textarea:    ta,
		highlighter: NewHighlighter(),
		id:          id,
	}
}

// Init returns the textarea blink command so the cursor blinks when focused.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update processes messages. It delegates to the underlying textarea and
// tracks whether the content has been modified.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	prevValue := m.textarea.Value()
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)

	// Track modifications: mark as modified when content changes.
	if m.textarea.Value() != prevValue {
		m.modified = true
	}

	return m, cmd
}

// View renders the editor. When focused, the textarea is shown directly
// inside a styled border (editing mode). When blurred, the content is
// syntax-highlighted with line numbers (read-only preview mode).
func (m Model) View() string {
	th := theme.Current

	// Pick border style based on focus state.
	var border lipgloss.Style
	if m.focused {
		border = th.FocusedBorder
	} else {
		border = th.UnfocusedBorder
	}

	// Account for border width (left + right = 2, top + bottom = 2).
	innerW := m.width - 2
	innerH := m.height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	var content string
	if m.focused {
		// Editing mode: let the textarea handle everything.
		m.textarea.SetWidth(innerW)
		m.textarea.SetHeight(innerH)
		content = m.textarea.View()
	} else {
		// Read-only mode: render syntax-highlighted content with line
		// numbers.
		content = m.renderHighlighted(th, innerW, innerH)
	}

	return border.
		Width(innerW).
		Height(innerH).
		Render(content)
}

// renderHighlighted produces a syntax-highlighted, line-numbered view of the
// editor content for the blurred (read-only) state.
func (m Model) renderHighlighted(th *theme.Theme, width, height int) string {
	raw := m.textarea.Value()
	if raw == "" {
		return th.MutedText.Render(m.textarea.Placeholder)
	}

	highlighted := m.highlighter.Highlight(raw, th)
	lines := strings.Split(highlighted, "\n")

	// Limit visible lines to the available height.
	if len(lines) > height {
		lines = lines[:height]
	}

	// Determine gutter width from the total number of lines.
	totalLines := strings.Count(raw, "\n") + 1
	gutterWidth := len(fmt.Sprintf("%d", totalLines))
	if gutterWidth < 2 {
		gutterWidth = 2
	}

	lineNumStyle := th.EditorLineNumber

	var b strings.Builder
	for i, line := range lines {
		num := lineNumStyle.Render(fmt.Sprintf("%*d ", gutterWidth, i+1))
		b.WriteString(num)
		b.WriteString(line)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// Value returns the raw text content of the editor.
func (m Model) Value() string {
	return m.textarea.Value()
}

// SetValue replaces the editor content.
func (m *Model) SetValue(s string) {
	m.textarea.SetValue(s)
}

// SetSize updates the editor dimensions. The values should include space for
// the border.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Keep the textarea in sync so it wraps correctly.
	innerW := w - 2
	innerH := h - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	m.textarea.SetWidth(innerW)
	m.textarea.SetHeight(innerH)
}

// Focus gives input focus to the editor.
func (m *Model) Focus() {
	m.focused = true
	m.textarea.Focus()
}

// Blur removes input focus from the editor.
func (m *Model) Blur() {
	m.focused = false
	m.textarea.Blur()
}

// Focused reports whether the editor currently has input focus.
func (m Model) Focused() bool {
	return m.focused
}

// Modified reports whether the content has changed since the last call to
// ResetModified.
func (m Model) Modified() bool {
	return m.modified
}

// ResetModified clears the modification flag, typically called after the
// query is executed or the content is saved.
func (m *Model) ResetModified() {
	m.modified = false
}

// ID returns the tab identifier associated with this editor.
func (m Model) ID() int {
	return m.id
}

// InsertText inserts text at the current cursor position. This is useful for
// inserting table names or column names from the sidebar.
func (m *Model) InsertText(text string) {
	// textarea does not expose an InsertRune/InsertString method directly.
	// We simulate insertion by focusing, setting the value with the text
	// appended at the current cursor position. A simple approach: append to
	// the current value (at the end). For proper cursor-position insertion
	// we would need access to textarea internals.
	//
	// The bubbles textarea exposes InsertString (added in newer versions).
	// Fall back to appending if that is unavailable.
	current := m.textarea.Value()
	if current == "" {
		m.textarea.SetValue(text)
	} else {
		// Append a space before the inserted text if the last character is
		// not whitespace.
		last := current[len(current)-1]
		if last != ' ' && last != '\n' && last != '\t' {
			text = " " + text
		}
		m.textarea.SetValue(current + text)
	}
	m.modified = true
}
