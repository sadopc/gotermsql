package sidebar

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/schema"
	"github.com/sadopc/gotermsql/internal/theme"
)

// useSimpleIcons returns true when running inside Neovim's terminal emulator,
// which has emoji width rendering issues in libvterm.
var useSimpleIcons = os.Getenv("NVIM") != ""

// NodeKind represents the type of tree node.
type NodeKind int

const (
	NodeDatabase NodeKind = iota
	NodeSchema
	NodeTableGroup
	NodeTable
	NodeViewGroup
	NodeView
	NodeColumn
)

// TreeNode represents a node in the schema tree.
type TreeNode struct {
	Label    string
	Kind     NodeKind
	Children []*TreeNode
	Expanded bool
	Depth    int

	// Metadata for generating queries
	Database string
	Schema   string
	Table    string
	Column   string
	ColType  string
	IsPK     bool
}

// Model is the schema browser sidebar.
type Model struct {
	nodes   []*TreeNode
	flat    []*TreeNode // flattened visible nodes
	cursor  int
	offset  int
	width   int
	height  int
	focused bool
	loading bool
}

// New creates a new sidebar.
func New() Model {
	return Model{}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles sidebar messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case appmsg.SchemaLoadedMsg:
		m.nodes = buildTree(msg.Databases)
		m.flatten()
		m.loading = false

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "down", "j":
			if m.cursor < len(m.flat)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "enter", "right", "l":
			return m, m.toggleOrSelect()
		case "left", "h":
			if m.cursor < len(m.flat) {
				node := m.flat[m.cursor]
				if node.Expanded {
					node.Expanded = false
					m.flatten()
				}
			}
		case "home", "g":
			m.cursor = 0
			m.offset = 0
		case "end", "G":
			m.cursor = len(m.flat) - 1
			m.ensureVisible()
		}
	}

	return m, nil
}

// View renders the sidebar.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	th := theme.Current

	// Account for border (left + right = 2, top + bottom = 2).
	innerW := m.width - 2
	innerH := m.height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	// Title
	title := " Schema Browser "
	var titleStyle lipgloss.Style
	if m.focused {
		titleStyle = th.SidebarTitle.Copy().Background(lipgloss.Color("#569CD6"))
	} else {
		titleStyle = th.SidebarTitle
	}
	titleLine := titleStyle.Width(innerW).Render(title)

	if m.loading {
		content := titleLine + "\n\n  Loading schema..."
		return m.borderStyle().Width(innerW).Height(innerH).Render(content)
	}

	if len(m.flat) == 0 {
		content := titleLine + "\n\n  No schema loaded.\n  Connect to a database."
		return m.borderStyle().Width(innerW).Height(innerH).Render(content)
	}

	// Render visible nodes: innerH - 1 for the title line.
	contentHeight := innerH - 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	var lines []string
	end := m.offset + contentHeight
	if end > len(m.flat) {
		end = len(m.flat)
	}

	for i := m.offset; i < end; i++ {
		node := m.flat[i]
		line := m.renderNode(node, i == m.cursor, th)
		lines = append(lines, line)
	}

	content := titleLine + "\n" + strings.Join(lines, "\n")
	return m.borderStyle().Width(innerW).Height(innerH).Render(content)
}

func (m Model) renderNode(node *TreeNode, selected bool, th *theme.Theme) string {
	indent := strings.Repeat("  ", node.Depth)

	var icon string
	if useSimpleIcons {
		switch node.Kind {
		case NodeDatabase:
			icon = "■ "
		case NodeSchema:
			icon = "▪ "
		case NodeTableGroup:
			icon = "≡ "
		case NodeTable:
			icon = "◆ "
		case NodeViewGroup:
			icon = "◎ "
		case NodeView:
			icon = "◇ "
		case NodeColumn:
			icon = "  "
		}
	} else {
		switch node.Kind {
		case NodeDatabase:
			icon = "🗄 "
		case NodeSchema:
			icon = "📁 "
		case NodeTableGroup:
			icon = "📋 "
		case NodeTable:
			icon = "📊 "
		case NodeViewGroup:
			icon = "👁 "
		case NodeView:
			icon = "📄 "
		case NodeColumn:
			icon = "  "
		}
	}

	// Expand/collapse indicator for parent nodes
	expandIcon := "  "
	if len(node.Children) > 0 {
		if node.Expanded {
			expandIcon = "▼ "
		} else {
			expandIcon = "▶ "
		}
	}

	label := node.Label
	if node.Kind == NodeColumn && node.ColType != "" {
		label = fmt.Sprintf("%s %s", node.Label, node.ColType)
	}

	line := indent + expandIcon + icon + label

	// Truncate to width
	maxW := m.width - 4
	if len(line) > maxW {
		line = line[:maxW-1] + "…"
	}
	// Pad
	for len(line) < maxW {
		line += " "
	}

	if selected {
		return th.SidebarSelected.Render(line)
	}

	switch node.Kind {
	case NodeDatabase:
		return th.SidebarDatabase.Render(line)
	case NodeSchema:
		return th.SidebarSchema.Render(line)
	case NodeTable:
		return th.SidebarTable.Render(line)
	case NodeView:
		return th.SidebarView.Render(line)
	case NodeColumn:
		if node.IsPK {
			return th.SidebarColumn.Bold(true).Render(line)
		}
		return th.SidebarColumn.Render(line)
	default:
		return th.SidebarColumn.Render(line)
	}
}

func (m Model) borderStyle() lipgloss.Style {
	th := theme.Current
	if m.focused {
		return th.FocusedBorder
	}
	return th.UnfocusedBorder
}

func (m *Model) toggleOrSelect() tea.Cmd {
	if m.cursor >= len(m.flat) {
		return nil
	}
	node := m.flat[m.cursor]

	// Toggle expand/collapse for parent nodes
	if len(node.Children) > 0 {
		node.Expanded = !node.Expanded
		m.flatten()
		return nil
	}

	// For table nodes, generate a SELECT query
	if node.Kind == NodeTable {
		tableName := quoteIdentifier(node.Table)
		if node.Schema != "" && node.Schema != "main" {
			tableName = quoteIdentifier(node.Schema) + "." + tableName
		}
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 100;", tableName)
		return func() tea.Msg {
			return appmsg.NewTabMsg{Query: query}
		}
	}

	return nil
}

func (m *Model) flatten() {
	m.flat = nil
	for _, node := range m.nodes {
		m.flattenNode(node)
	}
	if m.cursor >= len(m.flat) {
		m.cursor = len(m.flat) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) flattenNode(node *TreeNode) {
	m.flat = append(m.flat, node)
	if node.Expanded {
		for _, child := range node.Children {
			m.flattenNode(child)
		}
	}
}

func (m *Model) ensureVisible() {
	contentHeight := m.height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+contentHeight {
		m.offset = m.cursor - contentHeight + 1
	}
}

// SetSize sets the sidebar dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Focus focuses the sidebar.
func (m *Model) Focus() { m.focused = true }

// Blur unfocuses the sidebar.
func (m *Model) Blur() { m.focused = false }

// Focused returns whether the sidebar is focused.
func (m Model) Focused() bool { return m.focused }

// SetLoading sets the loading state.
func (m *Model) SetLoading(loading bool) { m.loading = loading }

// quoteIdentifier wraps a SQL identifier in double-quotes (ANSI style),
// escaping any embedded double-quotes by doubling them.
func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func buildTree(databases []schema.Database) []*TreeNode {
	var nodes []*TreeNode

	for _, db := range databases {
		dbNode := &TreeNode{
			Label:    db.Name,
			Kind:     NodeDatabase,
			Database: db.Name,
			Expanded: len(databases) == 1, // auto-expand if single database
		}

		for _, s := range db.Schemas {
			schemaNode := &TreeNode{
				Label:    s.Name,
				Kind:     NodeSchema,
				Database: db.Name,
				Schema:   s.Name,
				Depth:    1,
				Expanded: s.Name == "public" || s.Name == "main",
			}

			// Tables group
			if len(s.Tables) > 0 {
				tablesGroup := &TreeNode{
					Label:    fmt.Sprintf("Tables (%d)", len(s.Tables)),
					Kind:     NodeTableGroup,
					Database: db.Name,
					Schema:   s.Name,
					Depth:    2,
					Expanded: true,
				}
				for _, t := range s.Tables {
					tableNode := &TreeNode{
						Label:    t.Name,
						Kind:     NodeTable,
						Database: db.Name,
						Schema:   s.Name,
						Table:    t.Name,
						Depth:    3,
					}
					for _, c := range t.Columns {
						colNode := &TreeNode{
							Label:    c.Name,
							Kind:     NodeColumn,
							Database: db.Name,
							Schema:   s.Name,
							Table:    t.Name,
							Column:   c.Name,
							ColType:  c.Type,
							IsPK:     c.IsPK,
							Depth:    4,
						}
						tableNode.Children = append(tableNode.Children, colNode)
					}
					tablesGroup.Children = append(tablesGroup.Children, tableNode)
				}
				schemaNode.Children = append(schemaNode.Children, tablesGroup)
			}

			// Views group
			if len(s.Views) > 0 {
				viewsGroup := &TreeNode{
					Label:    fmt.Sprintf("Views (%d)", len(s.Views)),
					Kind:     NodeViewGroup,
					Database: db.Name,
					Schema:   s.Name,
					Depth:    2,
				}
				for _, v := range s.Views {
					viewNode := &TreeNode{
						Label:    v.Name,
						Kind:     NodeView,
						Database: db.Name,
						Schema:   s.Name,
						Table:    v.Name,
						Depth:    3,
					}
					viewsGroup.Children = append(viewsGroup.Children, viewNode)
				}
				schemaNode.Children = append(schemaNode.Children, viewsGroup)
			}

			dbNode.Children = append(dbNode.Children, schemaNode)
		}

		nodes = append(nodes, dbNode)
	}

	return nodes
}
