// Package results provides a virtualized table component for displaying
// SQL query results. It supports both fully-loaded and streaming result
// sets with paginated fetching through adapter.RowIterator.
package results

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sadopc/gotermsql/internal/adapter"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/theme"
)

// FetchedPageMsg carries rows fetched asynchronously from an iterator.
type FetchedPageMsg struct {
	Rows    [][]string
	Forward bool // true = FetchNext, false = FetchPrev
	Err     error
	TabID   int
}

// maxBufferedRows is the maximum number of rows kept in memory for streamed
// results. When this limit is exceeded, the oldest rows are trimmed.
const maxBufferedRows = 5000

// Model is the results table component. It wraps bubbles/table with support
// for streaming large result sets via adapter.RowIterator.
type Model struct {
	table     table.Model
	columns   []adapter.ColumnMeta
	rows      [][]string          // current page of rows in memory
	allRows   [][]string          // all loaded rows (for non-streaming results)
	totalRows int64               // total row count (-1 if unknown)
	offset    int                 // current scroll offset in the full dataset
	pageSize  int                 // rows per page
	iterator  adapter.RowIterator // for streaming results
	tabID     int
	width     int
	height    int
	focused   bool
	loading   bool
	message   string // status message ("INSERT 0 1", etc.)
	queryTime time.Duration
	err       error
}

// New creates a new results model with sensible defaults.
func New(tabID int) Model {
	t := table.New(
		table.WithFocused(false),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.
		Bold(false)
	t.SetStyles(s)

	return Model{
		table:     t,
		tabID:     tabID,
		pageSize:  1000,
		totalRows: -1,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the results table.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch msg.String() {
		case "pgdown":
			// If we have an iterator and are near the end of loaded rows,
			// fetch the next page.
			if m.iterator != nil && m.table.Cursor() >= len(m.rows)-1 {
				m.loading = true
				iter := m.iterator
				return m, fetchNextPage(iter, m.tabID)
			}
		case "pgup":
			// If we have an iterator and are at the top, fetch previous page.
			if m.iterator != nil && m.offset > 0 && m.table.Cursor() == 0 {
				m.loading = true
				iter := m.iterator
				return m, fetchPrevPage(iter, m.tabID)
			}
		}

		// Delegate all other key handling to the underlying table.
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd

	case appmsg.QueryResultMsg:
		m.SetResults(msg.Result)
		return m, nil

	case FetchedPageMsg:
		if msg.TabID != m.tabID {
			return m, nil
		}
		m.loading = false
		if msg.Err != nil {
			if !adapter.SentinelEOF(msg.Err) {
				m.err = msg.Err
			}
			return m, nil
		}
		if msg.Forward {
			m.allRows = append(m.allRows, msg.Rows...)
			// Trim oldest rows if exceeding buffer limit
			if len(m.allRows) > maxBufferedRows {
				excess := len(m.allRows) - maxBufferedRows
				m.allRows = m.allRows[excess:]
				m.offset += excess
			}
			m.rows = m.allRows
			m.rebuildTableRows()
		} else {
			m.allRows = append(msg.Rows, m.allRows...)
			m.offset -= len(msg.Rows)
			if m.offset < 0 {
				m.offset = 0
			}
			// Trim newest rows if exceeding buffer limit
			if len(m.allRows) > maxBufferedRows {
				m.allRows = m.allRows[:maxBufferedRows]
			}
			m.rows = m.allRows
			m.rebuildTableRows()
		}
		return m, nil
	}

	// Pass through any other messages to the table.
	if m.focused {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the results component.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	th := theme.Current

	// Reserve space for the border (2 lines) and footer (1 line).
	contentHeight := m.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Loading state.
	if m.loading && len(m.rows) == 0 {
		msg := th.MutedText.Render("  Executing query...")
		return m.wrapBorder(msg, contentHeight)
	}

	// Error state.
	if m.err != nil {
		errText := th.ErrorText.Render("  Error: " + m.err.Error())
		return m.wrapBorder(errText, contentHeight)
	}

	// Non-SELECT result message (INSERT, UPDATE, CREATE TABLE, etc.).
	if m.message != "" && len(m.rows) == 0 {
		msgText := th.SuccessText.Render("  " + m.message)
		return m.wrapBorder(msgText, contentHeight)
	}

	// Empty result set.
	if len(m.columns) == 0 && len(m.rows) == 0 && m.message == "" {
		placeholder := th.MutedText.Render("  No results — write a query and press F5 to execute")
		return m.wrapBorder(placeholder, contentHeight)
	}

	// Render table.
	tableView := m.table.View()

	// Build footer.
	footer := m.buildFooter()

	content := lipgloss.JoinVertical(lipgloss.Left, tableView, footer)
	return m.wrapBorder(content, 0)
}

// SetResults loads a complete QueryResult into the table.
func (m *Model) SetResults(result *adapter.QueryResult) {
	m.err = nil
	m.loading = false
	if m.iterator != nil {
		m.iterator.Close()
		m.iterator = nil
	}
	m.offset = 0
	m.queryTime = result.Duration

	if !result.IsSelect {
		// Non-SELECT statement: show message only.
		m.message = result.Message
		m.columns = nil
		m.rows = nil
		m.allRows = nil
		m.totalRows = result.RowCount
		m.table.SetRows(nil)
		m.table.SetColumns(nil)
		return
	}

	m.message = ""
	m.columns = result.Columns
	m.allRows = result.Rows
	m.rows = result.Rows
	m.totalRows = result.RowCount
	if m.totalRows < 0 {
		m.totalRows = int64(len(result.Rows))
	}

	m.rebuildTable()
}

// SetIterator configures the model for streaming mode with the given iterator.
func (m *Model) SetIterator(iter adapter.RowIterator) {
	if m.iterator != nil {
		m.iterator.Close()
	}
	m.iterator = iter
	m.columns = iter.Columns()
	m.totalRows = iter.TotalRows()
	m.offset = 0
	m.err = nil
	m.message = ""
	m.allRows = nil
	m.rows = nil

	// Build column headers immediately so the table structure is visible.
	tableCols := autoSizeColumns(m.columns, nil, m.contentWidth())
	m.table.SetColumns(tableCols)
	m.table.SetRows(nil)
}

// SetSize updates the component dimensions and recalculates table layout.
func (m *Model) SetSize(w, h int) {
	if m.width == w && m.height == h {
		return
	}
	m.width = w
	m.height = h

	// Account for border.
	innerW := w - 2
	if innerW < 0 {
		innerW = 0
	}
	innerH := h - 3 // border top/bottom + footer
	if innerH < 1 {
		innerH = 1
	}

	m.table.SetWidth(innerW)
	m.table.SetHeight(innerH)

	// Recalculate column widths if we have data.
	if len(m.columns) > 0 {
		tableCols := autoSizeColumns(m.columns, m.rows, m.contentWidth())
		m.table.SetColumns(tableCols)
	}
}

// SetLoading sets the loading state.
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
	if loading {
		m.err = nil
	}
}

// SetError sets the error state.
func (m *Model) SetError(err error) {
	m.err = err
	m.loading = false
}

// SetMessage sets a status message with the associated query duration.
func (m *Model) SetMessage(msg string, duration time.Duration) {
	m.message = msg
	m.queryTime = duration
	m.err = nil
	m.loading = false
}

// Focus gives the results table keyboard focus.
func (m *Model) Focus() {
	m.focused = true
	m.table.Focus()
	m.applyStyles()
}

// Blur removes keyboard focus from the results table.
func (m *Model) Blur() {
	m.focused = false
	m.table.Blur()
	m.applyStyles()
}

// Focused reports whether the results table is currently focused.
func (m Model) Focused() bool {
	return m.focused
}

// SelectedRow returns the data for the currently selected row, or nil if
// no row is selected.
func (m Model) SelectedRow() []string {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return nil
	}
	return row
}

// RowCount returns the total number of rows in the result set. Returns -1
// if the total is unknown (streaming mode before completion).
func (m Model) RowCount() int64 {
	return m.totalRows
}

// QueryDuration returns how long the query took to execute.
func (m Model) QueryDuration() time.Duration {
	return m.queryTime
}

// Columns returns the current column metadata.
func (m Model) Columns() []adapter.ColumnMeta {
	return m.columns
}

// Rows returns all loaded rows.
func (m Model) Rows() [][]string {
	return m.allRows
}

// CloseIterator closes the current iterator if any, releasing resources.
func (m *Model) CloseIterator() {
	if m.iterator != nil {
		m.iterator.Close()
		m.iterator = nil
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// rebuildTable recalculates columns and repopulates the table widget.
func (m *Model) rebuildTable() {
	tableCols := autoSizeColumns(m.columns, m.rows, m.contentWidth())
	m.table.SetColumns(tableCols)
	m.rebuildTableRows()
}

// rebuildTableRows converts [][]string rows into table.Row and sets them.
func (m *Model) rebuildTableRows() {
	tableRows := make([]table.Row, len(m.rows))
	for i, row := range m.rows {
		tableRows[i] = table.Row(row)
	}
	m.table.SetRows(tableRows)
}

// contentWidth returns the usable width inside the border.
func (m *Model) contentWidth() int {
	w := m.width - 2 // border left + right
	if w < 10 {
		w = 10
	}
	return w
}

// buildFooter constructs the row count and timing footer line.
func (m Model) buildFooter() string {
	th := theme.Current
	var parts []string

	// Row count.
	switch {
	case m.totalRows >= 0:
		parts = append(parts, fmt.Sprintf("%d rows", m.totalRows))
	case len(m.allRows) > 0:
		parts = append(parts, fmt.Sprintf("%d rows loaded", len(m.allRows)))
	}

	// Query duration.
	if m.queryTime > 0 {
		parts = append(parts, fmt.Sprintf("%s", formatDuration(m.queryTime)))
	}

	// Loading indicator.
	if m.loading {
		parts = append(parts, "loading...")
	}

	if len(parts) == 0 {
		return ""
	}

	footer := "  " + strings.Join(parts, " | ")
	return th.MutedText.Render(footer)
}

// wrapBorder renders the content inside a themed border frame.
func (m Model) wrapBorder(content string, minHeight int) string {
	th := theme.Current

	var borderStyle lipgloss.Style
	if m.focused {
		borderStyle = th.FocusedBorder
	} else {
		borderStyle = th.UnfocusedBorder
	}

	innerW := m.width - 2
	if innerW < 0 {
		innerW = 0
	}

	style := borderStyle.Width(innerW)
	if minHeight > 0 {
		style = style.Height(minHeight)
	}

	return style.Render(content)
}

// applyStyles updates the table styles based on the current theme and focus.
func (m *Model) applyStyles() {
	th := theme.Current
	s := table.DefaultStyles()

	s.Header = th.ResultsHeader.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240"))

	s.Cell = th.ResultsCell

	s.Selected = th.ResultsSelectedRow

	m.table.SetStyles(s)
}

// formatDuration produces a human-readable duration string.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%d us", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%d ms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.2f s", d.Seconds())
	default:
		return fmt.Sprintf("%.1f min", d.Minutes())
	}
}

// ---------------------------------------------------------------------------
// Async fetch commands
// ---------------------------------------------------------------------------

// fetchNextPage returns a tea.Cmd that fetches the next page from an iterator.
func fetchNextPage(iter adapter.RowIterator, tabID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rows, err := iter.FetchNext(ctx)
		return FetchedPageMsg{Rows: rows, Forward: true, Err: err, TabID: tabID}
	}
}

// fetchPrevPage returns a tea.Cmd that fetches the previous page from an iterator.
func fetchPrevPage(iter adapter.RowIterator, tabID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rows, err := iter.FetchPrev(ctx)
		return FetchedPageMsg{Rows: rows, Forward: false, Err: err, TabID: tabID}
	}
}

// ---------------------------------------------------------------------------
// Column auto-sizing
// ---------------------------------------------------------------------------

// autoSizeColumns calculates column widths based on header names and data
// content, distributing available space proportionally and capping individual
// columns at maxWidth.
func autoSizeColumns(cols []adapter.ColumnMeta, rows [][]string, maxWidth int) []table.Column {
	if len(cols) == 0 {
		return nil
	}

	numCols := len(cols)

	// Start with header lengths as minimum widths.
	widths := make([]int, numCols)
	for i, c := range cols {
		widths[i] = len(c.Name)
		if widths[i] < 4 {
			widths[i] = 4 // minimum column width
		}
	}

	// Sample up to 100 rows to estimate content widths.
	sampleSize := len(rows)
	if sampleSize > 100 {
		sampleSize = 100
	}
	for i := 0; i < sampleSize; i++ {
		for j := 0; j < numCols && j < len(rows[i]); j++ {
			cellLen := len(rows[i][j])
			if cellLen > widths[j] {
				widths[j] = cellLen
			}
		}
	}

	// Cap individual column widths at 50 characters.
	const maxColWidth = 50
	for i := range widths {
		if widths[i] > maxColWidth {
			widths[i] = maxColWidth
		}
	}

	// Calculate total desired width. The bubbles/table component adds no
	// separator between columns; spacing comes from the Cell style's
	// Padding(0, 1) which adds 2 characters per column (1 left + 1 right).
	paddingWidth := numCols * 2
	totalDesired := paddingWidth
	for _, w := range widths {
		totalDesired += w
	}

	// If the total exceeds the available width, scale columns down
	// proportionally.
	available := maxWidth - paddingWidth
	if available < numCols {
		available = numCols
	}

	if totalDesired > maxWidth {
		totalColWidth := totalDesired - paddingWidth
		for i := range widths {
			widths[i] = (widths[i] * available) / totalColWidth
			if widths[i] < 2 {
				widths[i] = 2
			}
		}
	}

	// Build table.Column slice.
	tableCols := make([]table.Column, numCols)
	for i, c := range cols {
		title := c.Name
		tableCols[i] = table.Column{
			Title: title,
			Width: widths[i],
		}
	}

	return tableCols
}
