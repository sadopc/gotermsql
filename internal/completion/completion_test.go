package completion

import (
	"sort"
	"strings"
	"testing"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

// ---------------------------------------------------------------------------
// Helper: build a standard test schema
// ---------------------------------------------------------------------------

func testDatabases() []schema.Database {
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
								{Name: "id", Type: "integer", IsPK: true, Nullable: false},
								{Name: "name", Type: "text", Nullable: false},
								{Name: "email", Type: "text", Nullable: true},
								{Name: "created_at", Type: "timestamp", Nullable: false},
							},
						},
						{
							Name: "orders",
							Columns: []schema.Column{
								{Name: "id", Type: "integer", IsPK: true, Nullable: false},
								{Name: "user_id", Type: "integer", Nullable: false},
								{Name: "total", Type: "numeric", Nullable: false},
								{Name: "status", Type: "text", Nullable: true},
							},
						},
					},
					Views: []schema.View{
						{
							Name: "active_users",
							Columns: []schema.Column{
								{Name: "id", Type: "integer", IsPK: false, Nullable: false},
								{Name: "name", Type: "text", Nullable: false},
							},
						},
					},
				},
			},
		},
	}
}

func newTestEngine() *Engine {
	e := NewEngine("postgres")
	e.UpdateSchema(testDatabases())
	return e
}

// collectLabels extracts labels from a completion item slice for easy comparison.
func collectLabels(items []adapter.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.Label
	}
	return labels
}

// containsLabel returns true if label appears in items.
func containsLabel(items []adapter.CompletionItem, label string) bool {
	for _, it := range items {
		if it.Label == label {
			return true
		}
	}
	return false
}

// containsKind returns true if at least one item has the given kind.
func containsKind(items []adapter.CompletionItem, kind adapter.CompletionKind) bool {
	for _, it := range items {
		if it.Kind == kind {
			return true
		}
	}
	return false
}

// onlyKind returns true if every item in the slice has the given kind.
func onlyKind(items []adapter.CompletionItem, kind adapter.CompletionKind) bool {
	if len(items) == 0 {
		return false
	}
	for _, it := range items {
		if it.Kind != kind {
			return false
		}
	}
	return true
}

// filterByKind returns only items of the given kind.
func filterByKind(items []adapter.CompletionItem, kind adapter.CompletionKind) []adapter.CompletionItem {
	var result []adapter.CompletionItem
	for _, it := range items {
		if it.Kind == kind {
			result = append(result, it)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// 1. NewEngine
// ---------------------------------------------------------------------------

func TestNewEngine(t *testing.T) {
	dialects := []struct {
		name          string
		extraKeywords []string // a sample keyword unique to that dialect
	}{
		{"postgres", PostgresKeywords[:1]},
		{"mysql", MySQLKeywords[:1]},
		{"sqlite", SQLiteKeywords[:1]},
		{"duckdb", DuckDBKeywords[:1]},
		{"sql", nil}, // generic -- no extra keywords
	}

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			e := NewEngine(d.name)

			if e.dialect != d.name {
				t.Errorf("dialect = %q, want %q", e.dialect, d.name)
			}
			if e.tables == nil {
				t.Error("tables map should be initialized, got nil")
			}

			// Every dialect must include the common keywords.
			for _, kw := range CommonKeywords {
				found := false
				for _, k := range e.keywords {
					if k == kw {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("common keyword %q missing for dialect %q", kw, d.name)
				}
			}

			// Dialect-specific keywords should be present.
			for _, kw := range d.extraKeywords {
				found := false
				for _, k := range e.keywords {
					if k == kw {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("dialect keyword %q missing for dialect %q", kw, d.name)
				}
			}

			// Functions should be populated.
			if len(e.functions) == 0 {
				t.Error("functions should be populated")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. UpdateSchema
// ---------------------------------------------------------------------------

func TestUpdateSchema_SingleDatabase(t *testing.T) {
	e := NewEngine("postgres")
	e.UpdateSchema(testDatabases())

	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.databases) != 1 || e.databases[0] != "testdb" {
		t.Errorf("databases = %v, want [testdb]", e.databases)
	}
	if len(e.schemas) != 1 || e.schemas[0] != "public" {
		t.Errorf("schemas = %v, want [public]", e.schemas)
	}

	// Should store both qualified and unqualified table names.
	if _, ok := e.tables["users"]; !ok {
		t.Error("expected unqualified key 'users'")
	}
	if _, ok := e.tables["public.users"]; !ok {
		t.Error("expected qualified key 'public.users'")
	}
	if _, ok := e.tables["orders"]; !ok {
		t.Error("expected unqualified key 'orders'")
	}

	// Views should also be stored.
	if _, ok := e.tables["active_users"]; !ok {
		t.Error("expected unqualified key 'active_users' (view)")
	}
	if _, ok := e.tables["public.active_users"]; !ok {
		t.Error("expected qualified key 'public.active_users' (view)")
	}

	// Column count.
	if cols := e.tables["users"]; len(cols) != 4 {
		t.Errorf("users columns = %d, want 4", len(cols))
	}
}

func TestUpdateSchema_MultipleDatabases(t *testing.T) {
	dbs := []schema.Database{
		{
			Name: "db1",
			Schemas: []schema.Schema{
				{
					Name: "public",
					Tables: []schema.Table{
						{Name: "accounts", Columns: []schema.Column{{Name: "id", Type: "int"}}},
					},
				},
				{
					Name: "analytics",
					Tables: []schema.Table{
						{Name: "events", Columns: []schema.Column{{Name: "event_id", Type: "int"}}},
					},
				},
			},
		},
		{
			Name: "db2",
			Schemas: []schema.Schema{
				{
					Name: "main",
					Tables: []schema.Table{
						{Name: "products", Columns: []schema.Column{{Name: "sku", Type: "text"}}},
					},
				},
			},
		},
	}

	e := NewEngine("postgres")
	e.UpdateSchema(dbs)

	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.databases) != 2 {
		t.Errorf("databases count = %d, want 2", len(e.databases))
	}
	if len(e.schemas) != 3 {
		t.Errorf("schemas count = %d, want 3", len(e.schemas))
	}

	for _, key := range []string{"accounts", "public.accounts", "events", "analytics.events", "products", "main.products"} {
		if _, ok := e.tables[key]; !ok {
			t.Errorf("expected table key %q", key)
		}
	}
}

func TestUpdateSchema_EmptySchema(t *testing.T) {
	e := NewEngine("sqlite")
	e.UpdateSchema(nil)

	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.tables) != 0 {
		t.Errorf("tables should be empty, got %d entries", len(e.tables))
	}
	if len(e.databases) != 0 {
		t.Errorf("databases should be empty, got %v", e.databases)
	}
}

func TestUpdateSchema_ReplacePrevious(t *testing.T) {
	e := NewEngine("postgres")

	// First schema load.
	e.UpdateSchema(testDatabases())
	if _, ok := e.tables["users"]; !ok {
		t.Fatal("expected 'users' after first load")
	}

	// Second schema load with different data; old data should be gone.
	e.UpdateSchema([]schema.Database{
		{
			Name: "newdb",
			Schemas: []schema.Schema{
				{
					Name: "public",
					Tables: []schema.Table{
						{Name: "products", Columns: []schema.Column{{Name: "sku", Type: "text"}}},
					},
				},
			},
		},
	})

	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, ok := e.tables["users"]; ok {
		t.Error("old table 'users' should have been replaced")
	}
	if _, ok := e.tables["products"]; !ok {
		t.Error("new table 'products' should be present")
	}
	if len(e.databases) != 1 || e.databases[0] != "newdb" {
		t.Errorf("databases = %v, want [newdb]", e.databases)
	}
}

// ---------------------------------------------------------------------------
// 3. Complete - Keyword completion
// ---------------------------------------------------------------------------

func TestComplete_KeywordsAtStartOfStatement(t *testing.T) {
	e := newTestEngine()
	items := e.Complete("", 0)

	if len(items) == 0 {
		t.Fatal("expected completions at start of statement")
	}
	if !containsKind(items, adapter.CompletionKeyword) {
		t.Error("expected keyword completions at start of statement")
	}
}

func TestComplete_KeywordAfterSemicolon(t *testing.T) {
	e := newTestEngine()
	text := "SELECT 1; "
	items := e.Complete(text, len(text))

	if len(items) == 0 {
		t.Fatal("expected completions after semicolon")
	}
	// After a semicolon and space, the context is general, so keywords should appear.
	if !containsKind(items, adapter.CompletionKeyword) {
		t.Error("expected keyword completions after semicolon")
	}
}

func TestComplete_PartialKeyword(t *testing.T) {
	e := newTestEngine()

	t.Run("uppercase_SEL", func(t *testing.T) {
		text := "SEL"
		items := e.Complete(text, len(text))
		if !containsLabel(items, "SELECT") {
			t.Errorf("expected SELECT in results, got %v", collectLabels(items))
		}
	})

	t.Run("lowercase_sel", func(t *testing.T) {
		text := "sel"
		items := e.Complete(text, len(text))
		if !containsLabel(items, "SELECT") {
			t.Errorf("expected SELECT (case insensitive) in results, got %v", collectLabels(items))
		}
	})

	t.Run("mixed_case_Sel", func(t *testing.T) {
		text := "Sel"
		items := e.Complete(text, len(text))
		if !containsLabel(items, "SELECT") {
			t.Errorf("expected SELECT (mixed case) in results, got %v", collectLabels(items))
		}
	})
}

// ---------------------------------------------------------------------------
// 4. Complete - Table completion
// ---------------------------------------------------------------------------

func TestComplete_TableAfterFROM(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM "
	items := e.Complete(text, len(text))

	if !containsLabel(items, "users") {
		t.Error("expected 'users' table after FROM")
	}
	if !containsLabel(items, "orders") {
		t.Error("expected 'orders' table after FROM")
	}
	if !containsLabel(items, "active_users") {
		t.Error("expected 'active_users' view after FROM")
	}
}

func TestComplete_TableAfterJOIN(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users JOIN "
	items := e.Complete(text, len(text))

	if !containsLabel(items, "orders") {
		t.Error("expected 'orders' table after JOIN")
	}
}

func TestComplete_TableAfterINTO(t *testing.T) {
	e := newTestEngine()
	text := "INSERT INTO "
	items := e.Complete(text, len(text))

	if !containsLabel(items, "users") {
		t.Error("expected 'users' table after INTO")
	}
}

func TestComplete_TableAfterUPDATE(t *testing.T) {
	e := newTestEngine()
	text := "UPDATE "
	items := e.Complete(text, len(text))

	if !containsLabel(items, "users") {
		t.Error("expected 'users' table after UPDATE")
	}
}

func TestComplete_PartialTableName(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM us"
	items := e.Complete(text, len(text))

	if !containsLabel(items, "users") {
		t.Errorf("expected 'users' for prefix 'us', got %v", collectLabels(items))
	}
}

func TestComplete_TableAfterLEFTJOIN(t *testing.T) {
	e := newTestEngine()
	// "LEFT" is in fromKeywords, so after "LEFT " we expect table completions.
	text := "SELECT * FROM users LEFT "
	items := e.Complete(text, len(text))

	if !containsLabel(items, "orders") {
		t.Error("expected 'orders' after LEFT (join context)")
	}
}

func TestComplete_TableAfterTABLE(t *testing.T) {
	e := newTestEngine()
	text := "CREATE TABLE "
	items := e.Complete(text, len(text))

	// TABLE triggers contextFrom, which suggests table names.
	if !containsKind(items, adapter.CompletionTable) {
		t.Error("expected table completions after CREATE TABLE")
	}
}

// ---------------------------------------------------------------------------
// 5. Complete - Column completion
// ---------------------------------------------------------------------------

func TestComplete_ColumnsAfterSELECT(t *testing.T) {
	e := newTestEngine()
	text := "SELECT "
	items := e.Complete(text, len(text))

	// After SELECT (contextColumn), we get columns from FROM tables, table names, and functions.
	// Since there's no FROM clause yet, columns come from an empty table list,
	// but we still expect table names and functions.
	if !containsKind(items, adapter.CompletionTable) {
		t.Error("expected table completions after SELECT")
	}
	if !containsKind(items, adapter.CompletionFunction) {
		t.Error("expected function completions after SELECT")
	}
}

func TestComplete_ColumnsAfterSELECTWithFROM(t *testing.T) {
	e := newTestEngine()
	// Full statement with FROM; cursor is after SELECT.
	text := "SELECT  FROM users"
	// Cursor at position 7 (after "SELECT ").
	items := e.Complete(text, 7)

	// Should include columns from the users table.
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' column from users, got %v", collectLabels(items))
	}
	if !containsLabel(items, "email") {
		t.Errorf("expected 'email' column from users, got %v", collectLabels(items))
	}
}

func TestComplete_ColumnsAfterWHERE(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users WHERE "
	items := e.Complete(text, len(text))

	// After WHERE (contextColumn), columns from FROM tables should appear.
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' column after WHERE, got %v", collectLabels(items))
	}
	if !containsLabel(items, "name") {
		t.Errorf("expected 'name' column after WHERE, got %v", collectLabels(items))
	}
}

func TestComplete_DotAccessColumns(t *testing.T) {
	e := newTestEngine()
	text := "SELECT users."
	items := e.Complete(text, len(text))

	labels := collectLabels(items)
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' in dot access results, got %v", labels)
	}
	if !containsLabel(items, "email") {
		t.Errorf("expected 'email' in dot access results, got %v", labels)
	}
	if !containsLabel(items, "name") {
		t.Errorf("expected 'name' in dot access results, got %v", labels)
	}
	if !containsLabel(items, "created_at") {
		t.Errorf("expected 'created_at' in dot access results, got %v", labels)
	}

	// All results should be column kind.
	if !onlyKind(items, adapter.CompletionColumn) {
		t.Error("dot access should return only column completions")
	}
}

func TestComplete_DotAccessWithPrefix(t *testing.T) {
	e := newTestEngine()
	text := "SELECT users.em"
	items := e.Complete(text, len(text))

	if !containsLabel(items, "email") {
		t.Errorf("expected 'email' for 'users.em', got %v", collectLabels(items))
	}
}

func TestComplete_ColumnsAfterORDERBY(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users ORDER BY "
	items := e.Complete(text, len(text))

	// "BY" is in columnKeywords.
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' after ORDER BY, got %v", collectLabels(items))
	}
	if !containsLabel(items, "name") {
		t.Errorf("expected 'name' after ORDER BY, got %v", collectLabels(items))
	}
}

func TestComplete_ColumnsAfterGROUPBY(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users GROUP BY "
	items := e.Complete(text, len(text))

	// "BY" triggers contextColumn.
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' after GROUP BY, got %v", collectLabels(items))
	}
}

func TestComplete_ColumnsAfterHAVING(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users GROUP BY name HAVING "
	items := e.Complete(text, len(text))

	// "HAVING" is in columnKeywords.
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' after HAVING, got %v", collectLabels(items))
	}
}

func TestComplete_ColumnsAfterON(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users JOIN orders ON "
	items := e.Complete(text, len(text))

	// "ON" is in columnKeywords.
	colItems := filterByKind(items, adapter.CompletionColumn)
	if len(colItems) == 0 {
		t.Error("expected column completions after ON")
	}
}

func TestComplete_ColumnsAfterAND(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users WHERE id = 1 AND "
	items := e.Complete(text, len(text))

	// "AND" is in columnKeywords.
	if !containsLabel(items, "name") {
		t.Errorf("expected 'name' after AND, got %v", collectLabels(items))
	}
}

func TestComplete_ColumnsAfterSET(t *testing.T) {
	e := newTestEngine()
	text := "UPDATE users SET "
	items := e.Complete(text, len(text))

	// "SET" is in columnKeywords; parseFromTables picks up "users" from UPDATE.
	// Note: the regex matches FROM/JOIN, and UPDATE is not FROM. Columns won't appear
	// from the UPDATE target unless FROM/JOIN references it. However, table names and functions
	// will still appear.
	if !containsKind(items, adapter.CompletionTable) {
		t.Error("expected table completions after SET")
	}
}

// ---------------------------------------------------------------------------
// 6. Complete - Fuzzy matching
// ---------------------------------------------------------------------------

func TestComplete_FuzzyTableMatch(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM usr"
	items := e.Complete(text, len(text))

	if !containsLabel(items, "users") {
		t.Errorf("fuzzy: expected 'users' for prefix 'usr', got %v", collectLabels(items))
	}
}

func TestComplete_FuzzyColumnMatch(t *testing.T) {
	e := newTestEngine()
	text := "SELECT eml FROM users"
	// Cursor at position 10 (after "SELECT eml").
	items := e.Complete(text, 10)

	if !containsLabel(items, "email") {
		t.Errorf("fuzzy: expected 'email' for prefix 'eml', got %v", collectLabels(items))
	}
}

func TestComplete_FuzzyDotAccessColumn(t *testing.T) {
	e := newTestEngine()
	text := "SELECT users.crt"
	items := e.Complete(text, len(text))

	if !containsLabel(items, "created_at") {
		t.Errorf("fuzzy: expected 'created_at' for 'users.crt', got %v", collectLabels(items))
	}
}

// ---------------------------------------------------------------------------
// 7. Complete - No completions / edge cases
// ---------------------------------------------------------------------------

func TestComplete_InsideStringLiteral(t *testing.T) {
	e := newTestEngine()

	tests := []struct {
		name string
		text string
		pos  int
	}{
		{"single_quoted", "SELECT * FROM users WHERE name = 'test", 38},
		{"mid_string", "SELECT 'hello ", 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := e.Complete(tt.text, tt.pos)
			if len(items) != 0 {
				t.Errorf("inside string literal: expected no completions, got %v", collectLabels(items))
			}
		})
	}
}

func TestComplete_EmptyInput(t *testing.T) {
	e := newTestEngine()
	items := e.Complete("", 0)

	// Empty input with cursor at 0 should give general completions (keywords + tables + functions).
	if len(items) == 0 {
		t.Error("expected completions for empty input")
	}
	if !containsKind(items, adapter.CompletionKeyword) {
		t.Error("expected keywords for empty input")
	}
}

func TestComplete_CursorAtZero(t *testing.T) {
	e := newTestEngine()
	items := e.Complete("SELECT * FROM users", 0)

	// Cursor at position 0 means we have empty 'before', so general context.
	if len(items) == 0 {
		t.Error("expected completions at cursor position 0")
	}
	if !containsKind(items, adapter.CompletionKeyword) {
		t.Error("expected keyword completions at cursor position 0")
	}
}

func TestComplete_CursorBeyondTextLength(t *testing.T) {
	e := newTestEngine()
	text := "SEL"
	// cursor far beyond end -- should be clamped.
	items := e.Complete(text, 100)
	if !containsLabel(items, "SELECT") {
		t.Errorf("cursor beyond text: expected SELECT, got %v", collectLabels(items))
	}
}

func TestComplete_NegativeCursorPosition(t *testing.T) {
	e := newTestEngine()
	items := e.Complete("SELECT", -5)
	// Negative cursor clamped to 0 => empty before => general context.
	if len(items) == 0 {
		t.Error("negative cursor: expected completions")
	}
}

func TestComplete_NoSchemaLoaded(t *testing.T) {
	e := NewEngine("postgres")
	// No UpdateSchema called.
	text := "SELECT * FROM "
	items := e.Complete(text, len(text))

	// Should have no table completions since no schema loaded, but should not panic.
	tables := filterByKind(items, adapter.CompletionTable)
	if len(tables) != 0 {
		t.Errorf("expected no table completions without schema, got %v", collectLabels(tables))
	}
}

// ---------------------------------------------------------------------------
// 8. parseFromTables
// ---------------------------------------------------------------------------

func TestParseFromTables(t *testing.T) {
	e := newTestEngine()

	tests := []struct {
		name   string
		sql    string
		expect []string
	}{
		{
			name:   "simple_FROM",
			sql:    "SELECT * FROM users",
			expect: []string{"users"},
		},
		{
			name:   "FROM_with_alias",
			sql:    "SELECT * FROM users u",
			expect: []string{"users"},
		},
		{
			name:   "FROM_with_AS_alias",
			sql:    "SELECT * FROM users AS u",
			expect: []string{"users"},
		},
		{
			name:   "FROM_with_JOIN",
			sql:    "SELECT * FROM users u JOIN orders o ON u.id = o.user_id",
			expect: []string{"users", "orders"},
		},
		{
			name:   "schema_qualified_table",
			sql:    "SELECT * FROM public.users",
			expect: []string{"public.users"},
		},
		{
			name:   "multiple_FROM_tables_comma",
			sql:    "SELECT * FROM users, orders",
			expect: []string{"users", "orders"},
		},
		{
			name:   "LEFT_JOIN",
			sql:    "SELECT * FROM users LEFT JOIN orders ON users.id = orders.user_id",
			expect: []string{"users", "orders"},
		},
		{
			name:   "no_FROM_clause",
			sql:    "SELECT 1",
			expect: nil,
		},
		{
			name:   "empty",
			sql:    "",
			expect: nil,
		},
		{
			name:   "subquery_multiple_FROMs",
			sql:    "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			expect: []string{"users", "orders"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.parseFromTables(tt.sql)

			if len(got) != len(tt.expect) {
				t.Fatalf("parseFromTables(%q) = %v (len %d), want %v (len %d)",
					tt.sql, got, len(got), tt.expect, len(tt.expect))
			}

			// Sort both for stable comparison.
			sortedGot := make([]string, len(got))
			copy(sortedGot, got)
			sort.Strings(sortedGot)
			sortedExpect := make([]string, len(tt.expect))
			copy(sortedExpect, tt.expect)
			sort.Strings(sortedExpect)

			for i := range sortedGot {
				if sortedGot[i] != sortedExpect[i] {
					t.Errorf("parseFromTables(%q) = %v, want %v", tt.sql, got, tt.expect)
					break
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 9. KeywordsForDialect
// ---------------------------------------------------------------------------

func TestKeywordsForDialect(t *testing.T) {
	tests := []struct {
		dialect        string
		mustContain    []string
		mustNotContain []string
	}{
		{
			dialect:     "postgres",
			mustContain: []string{"SELECT", "SERIAL", "RETURNING", "MATERIALIZED"},
			mustNotContain: []string{"AUTO_INCREMENT", "PRAGMA", "PIVOT"},
		},
		{
			dialect:     "postgresql",
			mustContain: []string{"SELECT", "SERIAL"},
		},
		{
			dialect:     "mysql",
			mustContain: []string{"SELECT", "AUTO_INCREMENT", "ENGINE", "SHOW"},
			mustNotContain: []string{"SERIAL", "PRAGMA", "PIVOT"},
		},
		{
			dialect:     "sqlite",
			mustContain: []string{"SELECT", "PRAGMA", "AUTOINCREMENT", "ROWID"},
			mustNotContain: []string{"SERIAL", "AUTO_INCREMENT", "PIVOT"},
		},
		{
			dialect:     "duckdb",
			mustContain: []string{"SELECT", "PIVOT", "UNPIVOT", "QUALIFY"},
			mustNotContain: []string{"SERIAL", "AUTO_INCREMENT", "PRAGMA"},
		},
		{
			dialect:        "sql",
			mustContain:    []string{"SELECT", "FROM", "WHERE"},
			mustNotContain: []string{"SERIAL", "AUTO_INCREMENT", "PRAGMA", "PIVOT"},
		},
		{
			dialect:     "unknown",
			mustContain: []string{"SELECT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.dialect, func(t *testing.T) {
			kw := KeywordsForDialect(tt.dialect)

			kwSet := map[string]bool{}
			for _, k := range kw {
				kwSet[k] = true
			}

			for _, must := range tt.mustContain {
				if !kwSet[must] {
					t.Errorf("KeywordsForDialect(%q): missing %q", tt.dialect, must)
				}
			}
			for _, mustNot := range tt.mustNotContain {
				if kwSet[mustNot] {
					t.Errorf("KeywordsForDialect(%q): should not contain %q", tt.dialect, mustNot)
				}
			}
		})
	}
}

func TestKeywordsForDialect_ReturnsNewSlice(t *testing.T) {
	kw1 := KeywordsForDialect("postgres")
	kw2 := KeywordsForDialect("postgres")

	// Mutating one should not affect the other.
	kw1[0] = "MODIFIED"
	if kw2[0] == "MODIFIED" {
		t.Error("KeywordsForDialect should return a new slice each time")
	}
}

// ---------------------------------------------------------------------------
// 10. FunctionsForDialect
// ---------------------------------------------------------------------------

func TestFunctionsForDialect(t *testing.T) {
	dialects := []string{"postgres", "mysql", "sqlite", "duckdb", "sql", "unknown"}

	for _, d := range dialects {
		t.Run(d, func(t *testing.T) {
			fns := FunctionsForDialect(d)

			if len(fns) == 0 {
				t.Errorf("FunctionsForDialect(%q) returned empty", d)
			}

			fnSet := map[string]bool{}
			for _, f := range fns {
				fnSet[f] = true
			}

			// All dialects should have the common functions.
			for _, cf := range CommonFunctions {
				if !fnSet[cf] {
					t.Errorf("FunctionsForDialect(%q): missing common function %q", d, cf)
				}
			}
		})
	}
}

func TestFunctionsForDialect_ReturnsNewSlice(t *testing.T) {
	fn1 := FunctionsForDialect("postgres")
	fn2 := FunctionsForDialect("postgres")

	fn1[0] = "MODIFIED"
	if fn2[0] == "MODIFIED" {
		t.Error("FunctionsForDialect should return a new slice each time")
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func TestInsideStringLiteral(t *testing.T) {
	tests := []struct {
		text   string
		expect bool
	}{
		{"", false},
		{"SELECT", false},
		{"SELECT 'hello'", false},          // matched pair
		{"SELECT 'hello", true},            // unmatched single
		{"SELECT 'it''s'", false},          // escaped single quotes (even count)
		{"SELECT 'it''s", true},            // odd count
		{"WHERE name = '", true},           // opening quote
		{"WHERE name = 'test' AND x = '", true},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := insideStringLiteral(tt.text)
			if got != tt.expect {
				t.Errorf("insideStringLiteral(%q) = %v, want %v", tt.text, got, tt.expect)
			}
		})
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		before     string
		wantPrefix string
		wantDot    string
	}{
		{"", "", ""},
		{"sel", "sel", ""},
		{"SELECT ", "", ""},
		{"SELECT us", "us", ""},
		{"users.", "", "users"},
		{"users.na", "na", "users"},
		{"SELECT users.em", "em", "users"},
		{"public.users.", "", "public.users"},
		{"public.users.na", "na", "public.users"},
		{"(us", "us", ""},
	}

	for _, tt := range tests {
		t.Run(tt.before, func(t *testing.T) {
			prefix, dot := extractPrefix(tt.before)
			if prefix != tt.wantPrefix || dot != tt.wantDot {
				t.Errorf("extractPrefix(%q) = (%q, %q), want (%q, %q)",
					tt.before, prefix, dot, tt.wantPrefix, tt.wantDot)
			}
		})
	}
}

func TestDetectContext(t *testing.T) {
	tests := []struct {
		before string
		prefix string
		want   contextKind
	}{
		{"", "", contextGeneral},
		{"SELECT ", "", contextColumn},
		{"SELECT * FROM ", "", contextFrom},
		{"SELECT * FROM users JOIN ", "", contextFrom},
		{"SELECT * FROM users WHERE ", "", contextColumn},
		{"INSERT INTO ", "", contextFrom},
		{"UPDATE ", "", contextFrom},
		{"SELECT * FROM users ORDER BY ", "", contextColumn},
		{"SELECT * FROM users GROUP BY ", "", contextColumn},
		// After a comma in a FROM list.
		{"SELECT * FROM users, ", "", contextFrom},
		// After a comma in a SELECT list.
		{"SELECT id, ", "", contextColumn},
		// General context after something like a value.
		{"SELECT * FROM users WHERE id = 1 ", "", contextGeneral},
	}

	for _, tt := range tests {
		name := tt.before
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			got := detectContext(tt.before, tt.prefix)
			if got != tt.want {
				t.Errorf("detectContext(%q, %q) = %v, want %v", tt.before, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestIsWordBreak(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', false},
		{'Z', false},
		{'0', false},
		{'_', false},
		{'.', false},
		{' ', true},
		{',', true},
		{'(', true},
		{')', true},
		{';', true},
		{'\t', true},
		{'\n', true},
	}

	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
			got := isWordBreak(tt.r)
			if got != tt.want {
				t.Errorf("isWordBreak(%q) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"SELECT", []string{"SELECT"}},
		{"SELECT * FROM users", []string{"SELECT", "*", "FROM", "users"}},
		{"  spaces  everywhere  ", []string{"spaces", "everywhere"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tokenize(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("tokenize(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Column detail formatting
// ---------------------------------------------------------------------------

func TestColumnsToItems_DetailFormat(t *testing.T) {
	cols := []schema.Column{
		{Name: "id", Type: "integer", IsPK: true, Nullable: false},
		{Name: "email", Type: "text", IsPK: false, Nullable: true},
		{Name: "age", Type: "int", IsPK: false, Nullable: false},
	}

	items := columnsToItems("users", cols)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// id: PK + NOT NULL
	if items[0].Label != "id" {
		t.Errorf("item[0].Label = %q, want 'id'", items[0].Label)
	}
	if !strings.Contains(items[0].Detail, "PK") {
		t.Errorf("item[0].Detail should contain 'PK', got %q", items[0].Detail)
	}
	if !strings.Contains(items[0].Detail, "NOT NULL") {
		t.Errorf("item[0].Detail should contain 'NOT NULL', got %q", items[0].Detail)
	}
	if items[0].Kind != adapter.CompletionColumn {
		t.Errorf("item[0].Kind = %v, want CompletionColumn", items[0].Kind)
	}

	// email: nullable, no PK
	if strings.Contains(items[1].Detail, "PK") {
		t.Errorf("email should not have PK in detail, got %q", items[1].Detail)
	}
	if strings.Contains(items[1].Detail, "NOT NULL") {
		t.Errorf("email is nullable, should not have NOT NULL, got %q", items[1].Detail)
	}

	// age: NOT NULL, no PK
	if strings.Contains(items[2].Detail, "PK") {
		t.Errorf("age should not have PK, got %q", items[2].Detail)
	}
	if !strings.Contains(items[2].Detail, "NOT NULL") {
		t.Errorf("age should have NOT NULL, got %q", items[2].Detail)
	}

	// All details should start with the table name.
	for _, item := range items {
		if !strings.HasPrefix(item.Detail, "users - ") {
			t.Errorf("Detail should start with 'users - ', got %q", item.Detail)
		}
	}
}

// ---------------------------------------------------------------------------
// Complete - comma continuation
// ---------------------------------------------------------------------------

func TestComplete_CommaInFROMList(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM users, "
	items := e.Complete(text, len(text))

	// After a comma in a FROM clause, should suggest more tables.
	if !containsLabel(items, "orders") {
		t.Errorf("expected 'orders' after comma in FROM, got %v", collectLabels(items))
	}
}

func TestComplete_CommaInSELECTList(t *testing.T) {
	e := newTestEngine()
	text := "SELECT id,  FROM users"
	// Cursor at position 11 (after "SELECT id, ").
	items := e.Complete(text, 11)

	// After comma in SELECT list, context is column.
	if !containsKind(items, adapter.CompletionColumn) || !containsKind(items, adapter.CompletionTable) || !containsKind(items, adapter.CompletionFunction) {
		t.Error("expected column/table/function completions after comma in SELECT list")
	}
}

// ---------------------------------------------------------------------------
// Complete - Views
// ---------------------------------------------------------------------------

func TestComplete_ViewsAppearAsTables(t *testing.T) {
	e := newTestEngine()
	text := "SELECT * FROM "
	items := e.Complete(text, len(text))

	if !containsLabel(items, "active_users") {
		t.Error("expected view 'active_users' in table completions")
	}
}

func TestComplete_DotAccessView(t *testing.T) {
	e := newTestEngine()
	text := "SELECT active_users."
	items := e.Complete(text, len(text))

	if !containsLabel(items, "id") {
		t.Error("expected 'id' column from active_users view")
	}
	if !containsLabel(items, "name") {
		t.Error("expected 'name' column from active_users view")
	}
}

// ---------------------------------------------------------------------------
// Complete - Schema-qualified dot access
// ---------------------------------------------------------------------------

func TestComplete_SchemaQualifiedDotAccess(t *testing.T) {
	e := newTestEngine()
	text := "SELECT public.users."
	items := e.Complete(text, len(text))

	// "public.users" is stored as a key, so dot access should resolve columns.
	if !containsLabel(items, "id") {
		t.Errorf("expected 'id' from public.users dot access, got %v", collectLabels(items))
	}
}

// ---------------------------------------------------------------------------
// FuzzyMatch
// ---------------------------------------------------------------------------

func TestFuzzyMatch_EmptyItems(t *testing.T) {
	result := fuzzyMatch("sel", nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil items, got %v", result)
	}
}

func TestFuzzyMatch_EmptyPrefix(t *testing.T) {
	items := []adapter.CompletionItem{
		{Label: "SELECT", Kind: adapter.CompletionKeyword},
	}
	// Empty prefix should match everything via fuzzy (empty string is a prefix of all).
	result := fuzzyMatch("", items)
	// fuzzy.FindFrom with empty string may or may not match; implementation-dependent.
	// We just ensure no panic.
	_ = result
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	items := []adapter.CompletionItem{
		{Label: "SELECT", Kind: adapter.CompletionKeyword},
		{Label: "SET", Kind: adapter.CompletionKeyword},
		{Label: "users", Kind: adapter.CompletionTable},
	}

	result := fuzzyMatch("sel", items)
	if !containsLabel(result, "SELECT") {
		t.Errorf("fuzzyMatch case insensitive: expected SELECT, got %v", collectLabels(result))
	}
}

func TestFuzzyMatch_CapsResult(t *testing.T) {
	// Create more than 50 items to verify the cap at 50.
	items := make([]adapter.CompletionItem, 100)
	for i := range items {
		items[i] = adapter.CompletionItem{Label: "item", Kind: adapter.CompletionKeyword}
	}

	result := fuzzyMatch("ite", items)
	if len(result) > 50 {
		t.Errorf("fuzzyMatch should cap at 50, got %d", len(result))
	}
}

// ---------------------------------------------------------------------------
// tableCompletions - avoid duplicates
// ---------------------------------------------------------------------------

func TestTableCompletions_NoDuplicates(t *testing.T) {
	e := newTestEngine()
	items := e.tableCompletions()

	seen := map[string]bool{}
	for _, it := range items {
		if seen[it.Label] {
			t.Errorf("duplicate table completion: %q", it.Label)
		}
		seen[it.Label] = true
	}

	// Verify unqualified names are present but schema-qualified duplicates are not.
	if !seen["users"] {
		t.Error("expected 'users' in table completions")
	}
	if seen["public.users"] {
		t.Error("should not include 'public.users' when 'users' is already present")
	}
}

// ---------------------------------------------------------------------------
// Complete - multiple join types
// ---------------------------------------------------------------------------

func TestComplete_JoinVariants(t *testing.T) {
	e := newTestEngine()

	joinKeywords := []string{"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL", "CROSS"}
	for _, jk := range joinKeywords {
		t.Run(jk, func(t *testing.T) {
			text := "SELECT * FROM users " + jk + " "
			items := e.Complete(text, len(text))
			if !containsKind(items, adapter.CompletionTable) {
				t.Errorf("expected table completions after %s", jk)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Complete - Function completions in SELECT context
// ---------------------------------------------------------------------------

func TestComplete_FunctionCompletionsInSELECT(t *testing.T) {
	e := newTestEngine()
	text := "SELECT COU"
	items := e.Complete(text, len(text))

	if !containsLabel(items, "COUNT") {
		t.Errorf("expected COUNT for prefix 'COU', got %v", collectLabels(items))
	}
}

func TestComplete_FunctionSUM(t *testing.T) {
	e := newTestEngine()
	text := "SELECT SU"
	items := e.Complete(text, len(text))

	if !containsLabel(items, "SUM") {
		t.Errorf("expected SUM for prefix 'SU', got %v", collectLabels(items))
	}
}

// ---------------------------------------------------------------------------
// Complete - large result capping
// ---------------------------------------------------------------------------

func TestComplete_ResultsCappedAt50(t *testing.T) {
	e := NewEngine("postgres")

	// Create a schema with many tables.
	var tables []schema.Table
	for i := 0; i < 100; i++ {
		tables = append(tables, schema.Table{
			Name:    strings.Repeat("t", i+1), // unique names
			Columns: []schema.Column{{Name: "id", Type: "int"}},
		})
	}
	e.UpdateSchema([]schema.Database{
		{Name: "db", Schemas: []schema.Schema{{Name: "public", Tables: tables}}},
	})

	// Empty prefix after FROM should be capped.
	text := "SELECT * FROM "
	items := e.Complete(text, len(text))
	if len(items) > 50 {
		t.Errorf("results should be capped at 50, got %d", len(items))
	}
}
