package sidebar

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/schema"
	"github.com/sadopc/gotermsql/internal/theme"
)

func init() {
	theme.Current = theme.Default()
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType}
}

func TestNew(t *testing.T) {
	m := New()

	if len(m.nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(m.nodes))
	}
	if len(m.flat) != 0 {
		t.Fatalf("expected 0 flat nodes, got %d", len(m.flat))
	}
	if m.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", m.cursor)
	}
	if m.focused {
		t.Fatal("expected focused=false")
	}
	if m.loading {
		t.Fatal("expected loading=false")
	}
}

func singleDBSchema() []schema.Database {
	return []schema.Database{
		{
			Name: "testdb",
			Schemas: []schema.Schema{
				{
					Name: "public",
					Tables: []schema.Table{
						{
							Name: "users",
							Columns: []schema.Column{
								{Name: "id", Type: "integer", IsPK: true},
								{Name: "name", Type: "text"},
								{Name: "email", Type: "text", Nullable: true},
							},
						},
						{
							Name: "orders",
							Columns: []schema.Column{
								{Name: "id", Type: "integer", IsPK: true},
								{Name: "user_id", Type: "integer"},
								{Name: "total", Type: "numeric"},
							},
						},
					},
					Views: []schema.View{
						{Name: "active_users"},
					},
				},
			},
		},
	}
}

func multiDBSchema() []schema.Database {
	return []schema.Database{
		{
			Name: "db1",
			Schemas: []schema.Schema{
				{
					Name: "main",
					Tables: []schema.Table{
						{Name: "table1"},
					},
				},
			},
		},
		{
			Name: "db2",
			Schemas: []schema.Schema{
				{
					Name: "public",
					Tables: []schema.Table{
						{Name: "table2"},
					},
				},
			},
		},
	}
}

func TestUpdate_SchemaLoaded(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	m.SetLoading(true)

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	if m.loading {
		t.Fatal("expected loading=false after SchemaLoadedMsg")
	}
	if len(m.nodes) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(m.nodes))
	}
	// With a single database, it should be auto-expanded.
	if !m.nodes[0].Expanded {
		t.Fatal("expected single database to be auto-expanded")
	}
	// Flat list should contain visible (expanded) nodes.
	if len(m.flat) == 0 {
		t.Fatal("expected flat list to have nodes after schema load")
	}
}

func TestBuildTree_SingleDB(t *testing.T) {
	dbs := singleDBSchema()
	nodes := buildTree(dbs)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 database node, got %d", len(nodes))
	}

	dbNode := nodes[0]
	if dbNode.Label != "testdb" {
		t.Fatalf("expected label 'testdb', got %q", dbNode.Label)
	}
	if dbNode.Kind != NodeDatabase {
		t.Fatalf("expected NodeDatabase, got %v", dbNode.Kind)
	}
	if !dbNode.Expanded {
		t.Fatal("expected single database to be auto-expanded")
	}

	// Should have 1 schema child.
	if len(dbNode.Children) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(dbNode.Children))
	}

	schemaNode := dbNode.Children[0]
	if schemaNode.Label != "public" {
		t.Fatalf("expected schema 'public', got %q", schemaNode.Label)
	}
	if schemaNode.Kind != NodeSchema {
		t.Fatalf("expected NodeSchema, got %v", schemaNode.Kind)
	}
	// "public" schema should be auto-expanded.
	if !schemaNode.Expanded {
		t.Fatal("expected 'public' schema to be auto-expanded")
	}

	// Should have 2 children: Tables group and Views group.
	if len(schemaNode.Children) != 2 {
		t.Fatalf("expected 2 groups (tables+views), got %d", len(schemaNode.Children))
	}

	tablesGroup := schemaNode.Children[0]
	if tablesGroup.Kind != NodeTableGroup {
		t.Fatalf("expected NodeTableGroup, got %v", tablesGroup.Kind)
	}
	if tablesGroup.Label != "Tables (2)" {
		t.Fatalf("expected 'Tables (2)', got %q", tablesGroup.Label)
	}
	if !tablesGroup.Expanded {
		t.Fatal("expected tables group to be expanded")
	}
	if len(tablesGroup.Children) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tablesGroup.Children))
	}

	// Verify table nodes.
	usersTable := tablesGroup.Children[0]
	if usersTable.Label != "users" {
		t.Fatalf("expected 'users', got %q", usersTable.Label)
	}
	if usersTable.Kind != NodeTable {
		t.Fatalf("expected NodeTable, got %v", usersTable.Kind)
	}
	if len(usersTable.Children) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(usersTable.Children))
	}

	// Verify PK marking.
	idCol := usersTable.Children[0]
	if idCol.Label != "id" {
		t.Fatalf("expected column 'id', got %q", idCol.Label)
	}
	if !idCol.IsPK {
		t.Fatal("expected id column to be PK")
	}
	if idCol.ColType != "integer" {
		t.Fatalf("expected ColType 'integer', got %q", idCol.ColType)
	}

	nameCol := usersTable.Children[1]
	if nameCol.IsPK {
		t.Fatal("expected name column to NOT be PK")
	}

	// Views group.
	viewsGroup := schemaNode.Children[1]
	if viewsGroup.Kind != NodeViewGroup {
		t.Fatalf("expected NodeViewGroup, got %v", viewsGroup.Kind)
	}
	if len(viewsGroup.Children) != 1 {
		t.Fatalf("expected 1 view, got %d", len(viewsGroup.Children))
	}
	if viewsGroup.Children[0].Label != "active_users" {
		t.Fatalf("expected view 'active_users', got %q", viewsGroup.Children[0].Label)
	}
	if viewsGroup.Children[0].Kind != NodeView {
		t.Fatalf("expected NodeView, got %v", viewsGroup.Children[0].Kind)
	}
}

func TestBuildTree_MultipleDBs(t *testing.T) {
	dbs := multiDBSchema()
	nodes := buildTree(dbs)

	if len(nodes) != 2 {
		t.Fatalf("expected 2 database nodes, got %d", len(nodes))
	}

	// With multiple databases, none should be auto-expanded.
	if nodes[0].Expanded {
		t.Fatal("expected first database NOT auto-expanded with multiple DBs")
	}
	if nodes[1].Expanded {
		t.Fatal("expected second database NOT auto-expanded with multiple DBs")
	}
}

func TestBuildTree_EmptySchema(t *testing.T) {
	dbs := []schema.Database{
		{
			Name: "emptydb",
			Schemas: []schema.Schema{
				{Name: "public"},
			},
		},
	}
	nodes := buildTree(dbs)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	schemaNode := nodes[0].Children[0]
	// No tables or views, so no groups.
	if len(schemaNode.Children) != 0 {
		t.Fatalf("expected 0 children for empty schema, got %d", len(schemaNode.Children))
	}
}

func TestNavigation(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	m.Focus()

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	initialCursor := m.cursor
	flatLen := len(m.flat)
	if flatLen < 2 {
		t.Fatalf("expected at least 2 flat nodes, got %d", flatLen)
	}

	// Move down.
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != initialCursor+1 {
		t.Fatalf("expected cursor=%d after down, got %d", initialCursor+1, m.cursor)
	}

	// Move up.
	m, _ = m.Update(keyMsg("k"))
	if m.cursor != initialCursor {
		t.Fatalf("expected cursor=%d after up, got %d", initialCursor, m.cursor)
	}

	// Move up at top: should stay at 0.
	m.cursor = 0
	m, _ = m.Update(keyMsg("k"))
	if m.cursor != 0 {
		t.Fatalf("expected cursor=0 at top boundary, got %d", m.cursor)
	}

	// Move down to bottom, then try moving past.
	m.cursor = flatLen - 1
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != flatLen-1 {
		t.Fatalf("expected cursor=%d at bottom boundary, got %d", flatLen-1, m.cursor)
	}

	// Test "up" key.
	m.cursor = 2
	m, _ = m.Update(specialKeyMsg(tea.KeyUp))
	if m.cursor != 1 {
		t.Fatalf("expected cursor=1 after up arrow, got %d", m.cursor)
	}

	// Test "down" key.
	m, _ = m.Update(specialKeyMsg(tea.KeyDown))
	if m.cursor != 2 {
		t.Fatalf("expected cursor=2 after down arrow, got %d", m.cursor)
	}
}

func TestNavigation_NotFocused(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	// Do NOT focus the sidebar.
	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	oldCursor := m.cursor
	m, _ = m.Update(keyMsg("j"))
	if m.cursor != oldCursor {
		t.Fatalf("expected cursor unchanged when not focused, got %d", m.cursor)
	}
}

func TestExpandCollapse(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	m.Focus()

	// Use multi-DB schema so databases are not auto-expanded.
	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: multiDBSchema()})

	// Initially only root nodes are visible (collapsed).
	if len(m.flat) != 2 {
		t.Fatalf("expected 2 flat nodes (collapsed DBs), got %d", len(m.flat))
	}

	// Press enter to expand first database.
	m, _ = m.Update(keyMsg("l"))
	if !m.flat[0].Expanded {
		t.Fatal("expected first db to be expanded after enter")
	}
	if len(m.flat) <= 2 {
		t.Fatal("expected more flat nodes after expanding db")
	}

	// Press left to collapse.
	m.cursor = 0
	m, _ = m.Update(keyMsg("h"))
	if m.flat[0].Expanded {
		t.Fatal("expected first db to be collapsed after left")
	}
}

func TestToggleOrSelect_Table(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	m.Focus()

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	// Find a table node in the flat list.
	tableIdx := -1
	for i, node := range m.flat {
		if node.Kind == NodeTable {
			tableIdx = i
			break
		}
	}
	if tableIdx < 0 {
		t.Fatal("expected to find a table node in flat list")
	}

	// Navigate to the table.
	m.cursor = tableIdx

	// Press enter to select the table.
	m, cmd := m.Update(keyMsg("l"))

	// The table node has children (columns), so enter should toggle expand.
	// Since the table node starts collapsed, it should expand.
	if m.flat[tableIdx].Kind == NodeTable && len(m.flat[tableIdx].Children) > 0 {
		// It should expand, not generate a query.
		if !m.flat[tableIdx].Expanded {
			t.Fatal("expected table with children to expand on enter")
		}
	}

	// Find a table node that has no children (or test with the leaf behavior).
	// Actually, table nodes DO have column children. Let us test the case where
	// the table is already expanded: pressing enter should collapse it.
	m.cursor = tableIdx
	m, cmd = m.Update(keyMsg("l"))
	_ = cmd

	// Now let's find a leaf table node to test select behavior.
	// Since all tables in our schema have columns, we need to test the actual
	// toggleOrSelect logic. Let's create a schema with tables that have no columns.
	noColSchema := []schema.Database{{
		Name: "testdb",
		Schemas: []schema.Schema{{
			Name: "main",
			Tables: []schema.Table{
				{Name: "simple_table"}, // no columns
			},
		}},
	}}

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: noColSchema})

	// Find the table node.
	tableIdx = -1
	for i, node := range m.flat {
		if node.Kind == NodeTable {
			tableIdx = i
			break
		}
	}
	if tableIdx < 0 {
		t.Fatal("expected to find table node")
	}

	m.cursor = tableIdx
	m, cmd = m.Update(keyMsg("l"))

	// The table has no children, so enter should generate a SELECT query.
	if cmd == nil {
		t.Fatal("expected cmd from selecting leaf table")
	}
	msg := cmd()
	newTabMsg, ok := msg.(appmsg.NewTabMsg)
	if !ok {
		t.Fatalf("expected NewTabMsg, got %T", msg)
	}
	if newTabMsg.Query != `SELECT * FROM "simple_table" LIMIT 100;` {
		t.Fatalf("unexpected query: %q", newTabMsg.Query)
	}
}

func TestToggleOrSelect_TableWithSchema(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	m.Focus()

	// Use a schema that is not "main" to test schema-qualified name.
	dbs := []schema.Database{{
		Name: "testdb",
		Schemas: []schema.Schema{{
			Name: "custom_schema",
			Tables: []schema.Table{
				{Name: "my_table"}, // no columns, leaf node
			},
		}},
	}}

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: dbs})

	// Expand nodes to reach the table.
	// The database is auto-expanded (single DB).
	// The schema "custom_schema" is NOT auto-expanded (not "public" or "main").
	// Expand the schema.
	for i, node := range m.flat {
		if node.Kind == NodeSchema {
			m.cursor = i
			m, _ = m.Update(keyMsg("l")) // expand schema
			break
		}
	}
	// The tables group is already auto-expanded by buildTree,
	// so the table should now be visible in the flat list.

	// Find the table node.
	tableIdx := -1
	for i, node := range m.flat {
		if node.Kind == NodeTable {
			tableIdx = i
			break
		}
	}
	if tableIdx < 0 {
		t.Fatal("expected to find table node")
	}

	m.cursor = tableIdx
	m, cmd := m.Update(keyMsg("l"))

	if cmd == nil {
		t.Fatal("expected cmd from selecting table with schema")
	}
	msg := cmd()
	newTabMsg, ok := msg.(appmsg.NewTabMsg)
	if !ok {
		t.Fatalf("expected NewTabMsg, got %T", msg)
	}
	expected := `SELECT * FROM "custom_schema"."my_table" LIMIT 100;`
	if newTabMsg.Query != expected {
		t.Fatalf("expected query %q, got %q", expected, newTabMsg.Query)
	}
}

func TestFocusBlur(t *testing.T) {
	m := New()

	if m.Focused() {
		t.Fatal("expected not focused initially")
	}

	m.Focus()
	if !m.Focused() {
		t.Fatal("expected focused after Focus()")
	}

	m.Blur()
	if m.Focused() {
		t.Fatal("expected not focused after Blur()")
	}
}

func TestSetLoading(t *testing.T) {
	m := New()

	m.SetLoading(true)
	if !m.loading {
		t.Fatal("expected loading=true")
	}

	m.SetLoading(false)
	if m.loading {
		t.Fatal("expected loading=false")
	}
}

func TestView_ZeroDimensions(t *testing.T) {
	m := New()
	view := m.View()
	if view != "" {
		t.Fatalf("expected empty view with zero dimensions, got %q", view)
	}
}

func TestView_Loading(t *testing.T) {
	m := New()
	m.SetSize(40, 20)
	m.SetLoading(true)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view when loading")
	}
}

func TestView_NoSchema(t *testing.T) {
	m := New()
	m.SetSize(40, 20)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view with no schema")
	}
}

func TestView_WithSchema(t *testing.T) {
	m := New()
	m.SetSize(40, 20)

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view with schema")
	}
}

func TestView_Focused(t *testing.T) {
	m := New()
	m.SetSize(40, 20)
	m.Focus()

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view when focused")
	}
}

func TestHomeEnd(t *testing.T) {
	m := New()
	m.SetSize(40, 30)
	m.Focus()

	m, _ = m.Update(appmsg.SchemaLoadedMsg{Databases: singleDBSchema()})

	flatLen := len(m.flat)
	if flatLen < 2 {
		t.Fatalf("need at least 2 nodes, got %d", flatLen)
	}

	// Go to end.
	m, _ = m.Update(keyMsg("G"))
	if m.cursor != flatLen-1 {
		t.Fatalf("expected cursor at end (%d), got %d", flatLen-1, m.cursor)
	}

	// Go to home.
	m, _ = m.Update(keyMsg("g"))
	if m.cursor != 0 {
		t.Fatalf("expected cursor=0 at home, got %d", m.cursor)
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd from Init")
	}
}
