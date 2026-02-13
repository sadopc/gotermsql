package sqlite

import (
	"context"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/sadopc/gotermsql/internal/adapter"
)

func TestSQLiteAdapter_Name(t *testing.T) {
	a := &sqliteAdapter{}
	if got := a.Name(); got != "sqlite" {
		t.Errorf("Name() = %q, want %q", got, "sqlite")
	}
}

func TestSQLiteAdapter_DefaultPort(t *testing.T) {
	a := &sqliteAdapter{}
	if got := a.DefaultPort(); got != 0 {
		t.Errorf("DefaultPort() = %d, want %d", got, 0)
	}
}

func TestSQLiteAdapter_Registration(t *testing.T) {
	a, ok := adapter.Registry["sqlite"]
	if !ok {
		t.Fatal("sqlite adapter not found in registry")
	}
	if a.Name() != "sqlite" {
		t.Errorf("registered adapter Name() = %q, want %q", a.Name(), "sqlite")
	}
	if a.DefaultPort() != 0 {
		t.Errorf("registered adapter DefaultPort() = %d, want %d", a.DefaultPort(), 0)
	}
}

func TestNormalizeDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "sqlite:// prefix stripped",
			dsn:  "sqlite:///path/to/file.db",
			want: "/path/to/file.db",
		},
		{
			name: "file: prefix stripped",
			dsn:  "file:test.db",
			want: "test.db",
		},
		{
			name: "memory unchanged",
			dsn:  ":memory:",
			want: ":memory:",
		},
		{
			name: "absolute path unchanged",
			dsn:  "/absolute/path.db",
			want: "/absolute/path.db",
		},
		{
			name: "relative path unchanged",
			dsn:  "relative/path.db",
			want: "relative/path.db",
		},
		{
			name: "sqlite:// relative path",
			dsn:  "sqlite://data.db",
			want: "data.db",
		},
		{
			name: "empty string",
			dsn:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDSN(tt.dsn)
			if got != tt.want {
				t.Errorf("normalizeDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// In-memory integration tests (no external database required)
// ---------------------------------------------------------------------------

func TestConnect_InMemory(t *testing.T) {
	a := &sqliteAdapter{}
	ctx := context.Background()

	conn, err := a.Connect(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Connect(:memory:) error: %v", err)
	}
	defer conn.Close()

	if err := conn.Ping(ctx); err != nil {
		t.Errorf("Ping() error: %v", err)
	}

	if got := conn.AdapterName(); got != "sqlite" {
		t.Errorf("AdapterName() = %q, want %q", got, "sqlite")
	}

	if got := conn.DatabaseName(); got != ":memory:" {
		t.Errorf("DatabaseName() = %q, want %q", got, ":memory:")
	}
}

func TestExecute_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	// Create table.
	result, err := conn.Execute(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}
	if result.IsSelect {
		t.Error("CREATE TABLE should not be IsSelect")
	}

	// Insert data.
	result, err = conn.Execute(ctx, "INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com')")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}
	if result.IsSelect {
		t.Error("INSERT should not be IsSelect")
	}
	if result.RowCount != 1 {
		t.Errorf("INSERT RowCount = %d, want 1", result.RowCount)
	}

	// Insert more data.
	_, err = conn.Execute(ctx, "INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com')")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	// SELECT data.
	result, err = conn.Execute(ctx, "SELECT id, name, email FROM users ORDER BY id")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}
	if !result.IsSelect {
		t.Error("SELECT should be IsSelect")
	}
	if result.RowCount != 2 {
		t.Errorf("SELECT RowCount = %d, want 2", result.RowCount)
	}
	if len(result.Columns) != 3 {
		t.Fatalf("SELECT returned %d columns, want 3", len(result.Columns))
	}
	if result.Columns[0].Name != "id" {
		t.Errorf("Column[0].Name = %q, want %q", result.Columns[0].Name, "id")
	}
	if result.Columns[1].Name != "name" {
		t.Errorf("Column[1].Name = %q, want %q", result.Columns[1].Name, "name")
	}
	if result.Columns[2].Name != "email" {
		t.Errorf("Column[2].Name = %q, want %q", result.Columns[2].Name, "email")
	}

	// Verify first row data.
	if len(result.Rows) < 2 {
		t.Fatalf("expected at least 2 rows, got %d", len(result.Rows))
	}
	if result.Rows[0][1] != "Alice" {
		t.Errorf("Row[0][1] = %q, want %q", result.Rows[0][1], "Alice")
	}
	if result.Rows[1][1] != "Bob" {
		t.Errorf("Row[1][1] = %q, want %q", result.Rows[1][1], "Bob")
	}
}

func TestDatabases_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	dbs, err := conn.Databases(ctx)
	if err != nil {
		t.Fatalf("Databases() error: %v", err)
	}

	if len(dbs) != 1 {
		t.Fatalf("Databases() returned %d databases, want 1", len(dbs))
	}

	if dbs[0].Name != ":memory:" {
		t.Errorf("Database name = %q, want %q", dbs[0].Name, ":memory:")
	}

	// Verify schemas include "main".
	if len(dbs[0].Schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(dbs[0].Schemas))
	}
	if dbs[0].Schemas[0].Name != "main" {
		t.Errorf("Schema name = %q, want %q", dbs[0].Schemas[0].Name, "main")
	}
}

func TestTables_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	// Initially no user tables.
	tables, err := conn.Tables(ctx, ":memory:", "main")
	if err != nil {
		t.Fatalf("Tables() error: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("Tables() initially returned %d tables, want 0", len(tables))
	}

	// Create a table.
	_, err = conn.Execute(ctx, "CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	// Create another table.
	_, err = conn.Execute(ctx, "CREATE TABLE orders (id INTEGER PRIMARY KEY, product_id INTEGER)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	tables, err = conn.Tables(ctx, ":memory:", "main")
	if err != nil {
		t.Fatalf("Tables() error: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("Tables() returned %d tables, want 2", len(tables))
	}

	// Tables should be ordered by name.
	if tables[0].Name != "orders" {
		t.Errorf("Tables()[0].Name = %q, want %q", tables[0].Name, "orders")
	}
	if tables[1].Name != "products" {
		t.Errorf("Tables()[1].Name = %q, want %q", tables[1].Name, "products")
	}
}

func TestColumns_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	_, err := conn.Execute(ctx, `CREATE TABLE items (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		price REAL,
		quantity INTEGER DEFAULT 0,
		description TEXT
	)`)
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	cols, err := conn.Columns(ctx, ":memory:", "main", "items")
	if err != nil {
		t.Fatalf("Columns() error: %v", err)
	}

	if len(cols) != 5 {
		t.Fatalf("Columns() returned %d columns, want 5", len(cols))
	}

	// Verify column properties.
	expected := []struct {
		name     string
		colType  string
		nullable bool
		isPK     bool
	}{
		// SQLite's PRAGMA table_info reports notNull=0 for INTEGER PRIMARY KEY
		// because it is the rowid alias and technically allows NULL in some edge cases.
		{"id", "INTEGER", true, true},
		{"name", "TEXT", false, false},
		{"price", "REAL", true, false},
		{"quantity", "INTEGER", true, false},
		{"description", "TEXT", true, false},
	}

	for i, exp := range expected {
		col := cols[i]
		if col.Name != exp.name {
			t.Errorf("Column[%d].Name = %q, want %q", i, col.Name, exp.name)
		}
		if col.Type != exp.colType {
			t.Errorf("Column[%d].Type = %q, want %q", i, col.Type, exp.colType)
		}
		if col.Nullable != exp.nullable {
			t.Errorf("Column[%d].Nullable = %v, want %v (column: %s)", i, col.Nullable, exp.nullable, exp.name)
		}
		if col.IsPK != exp.isPK {
			t.Errorf("Column[%d].IsPK = %v, want %v (column: %s)", i, col.IsPK, exp.isPK, exp.name)
		}
	}
}

func TestExecute_NonSelect(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	// Create table.
	_, err := conn.Execute(ctx, "CREATE TABLE counters (id INTEGER PRIMARY KEY, val INTEGER)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	// INSERT
	result, err := conn.Execute(ctx, "INSERT INTO counters (val) VALUES (10)")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}
	if result.IsSelect {
		t.Error("INSERT result should have IsSelect=false")
	}
	if result.RowCount != 1 {
		t.Errorf("INSERT RowCount = %d, want 1", result.RowCount)
	}
	if !strings.Contains(result.Message, "1") {
		t.Errorf("INSERT Message = %q, expected to contain '1'", result.Message)
	}

	// Insert more rows.
	_, err = conn.Execute(ctx, "INSERT INTO counters (val) VALUES (20)")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}
	_, err = conn.Execute(ctx, "INSERT INTO counters (val) VALUES (30)")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	// UPDATE
	result, err = conn.Execute(ctx, "UPDATE counters SET val = val + 1")
	if err != nil {
		t.Fatalf("UPDATE error: %v", err)
	}
	if result.IsSelect {
		t.Error("UPDATE result should have IsSelect=false")
	}
	if result.RowCount != 3 {
		t.Errorf("UPDATE RowCount = %d, want 3", result.RowCount)
	}

	// DELETE
	result, err = conn.Execute(ctx, "DELETE FROM counters WHERE val > 20")
	if err != nil {
		t.Fatalf("DELETE error: %v", err)
	}
	if result.IsSelect {
		t.Error("DELETE result should have IsSelect=false")
	}
	if result.RowCount != 2 {
		t.Errorf("DELETE RowCount = %d, want 2", result.RowCount)
	}
}

func TestExecute_NullHandling(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	_, err := conn.Execute(ctx, "CREATE TABLE nullable_test (id INTEGER, val TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	_, err = conn.Execute(ctx, "INSERT INTO nullable_test VALUES (1, NULL)")
	if err != nil {
		t.Fatalf("INSERT error: %v", err)
	}

	result, err := conn.Execute(ctx, "SELECT id, val FROM nullable_test")
	if err != nil {
		t.Fatalf("SELECT error: %v", err)
	}

	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
	if result.Rows[0][0] != "1" {
		t.Errorf("Row[0][0] = %q, want %q", result.Rows[0][0], "1")
	}
	if result.Rows[0][1] != "NULL" {
		t.Errorf("Row[0][1] = %q, want %q (NULL representation)", result.Rows[0][1], "NULL")
	}
}

func TestExecute_PragmaIsSelect(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	// PRAGMA should be treated as a SELECT-like query.
	result, err := conn.Execute(ctx, "PRAGMA table_info('sqlite_master')")
	if err != nil {
		t.Fatalf("PRAGMA error: %v", err)
	}
	if !result.IsSelect {
		t.Error("PRAGMA should be treated as IsSelect=true")
	}
}

func TestIndexes_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	_, err := conn.Execute(ctx, "CREATE TABLE indexed_table (id INTEGER PRIMARY KEY, name TEXT, email TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	_, err = conn.Execute(ctx, "CREATE UNIQUE INDEX idx_email ON indexed_table(email)")
	if err != nil {
		t.Fatalf("CREATE INDEX error: %v", err)
	}

	_, err = conn.Execute(ctx, "CREATE INDEX idx_name ON indexed_table(name)")
	if err != nil {
		t.Fatalf("CREATE INDEX error: %v", err)
	}

	indexes, err := conn.Indexes(ctx, ":memory:", "main", "indexed_table")
	if err != nil {
		t.Fatalf("Indexes() error: %v", err)
	}

	if len(indexes) < 2 {
		t.Fatalf("Indexes() returned %d indexes, want at least 2", len(indexes))
	}

	// Find the unique email index.
	found := false
	for _, idx := range indexes {
		if idx.Name == "idx_email" {
			found = true
			if !idx.Unique {
				t.Error("idx_email should be unique")
			}
			if len(idx.Columns) != 1 || idx.Columns[0] != "email" {
				t.Errorf("idx_email columns = %v, want [email]", idx.Columns)
			}
		}
	}
	if !found {
		t.Error("idx_email not found in indexes")
	}
}

func TestForeignKeys_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	_, err := conn.Execute(ctx, "CREATE TABLE parent (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE parent error: %v", err)
	}

	_, err = conn.Execute(ctx, "CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id))")
	if err != nil {
		t.Fatalf("CREATE TABLE child error: %v", err)
	}

	fks, err := conn.ForeignKeys(ctx, ":memory:", "main", "child")
	if err != nil {
		t.Fatalf("ForeignKeys() error: %v", err)
	}

	if len(fks) != 1 {
		t.Fatalf("ForeignKeys() returned %d, want 1", len(fks))
	}

	fk := fks[0]
	if fk.RefTable != "parent" {
		t.Errorf("FK RefTable = %q, want %q", fk.RefTable, "parent")
	}
	if len(fk.Columns) != 1 || fk.Columns[0] != "parent_id" {
		t.Errorf("FK Columns = %v, want [parent_id]", fk.Columns)
	}
	if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
		t.Errorf("FK RefColumns = %v, want [id]", fk.RefColumns)
	}
}

func TestCompletions_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	_, err := conn.Execute(ctx, "CREATE TABLE comp_test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	items, err := conn.Completions(ctx)
	if err != nil {
		t.Fatalf("Completions() error: %v", err)
	}

	// Should have at least 1 table + 2 columns = 3 items.
	if len(items) < 3 {
		t.Fatalf("Completions() returned %d items, want at least 3", len(items))
	}

	// Check the table completion.
	foundTable := false
	for _, item := range items {
		if item.Label == "comp_test" && item.Kind == adapter.CompletionTable {
			foundTable = true
		}
	}
	if !foundTable {
		t.Error("expected to find 'comp_test' table in completions")
	}

	// Check column completions.
	foundID := false
	foundName := false
	for _, item := range items {
		if item.Label == "id" && item.Kind == adapter.CompletionColumn {
			foundID = true
		}
		if item.Label == "name" && item.Kind == adapter.CompletionColumn {
			foundName = true
		}
	}
	if !foundID {
		t.Error("expected to find 'id' column in completions")
	}
	if !foundName {
		t.Error("expected to find 'name' column in completions")
	}
}

func TestExecuteStreaming_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	_, err := conn.Execute(ctx, "CREATE TABLE stream_test (id INTEGER PRIMARY KEY, val TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	// Insert 10 rows.
	for i := 1; i <= 10; i++ {
		_, err = conn.Execute(ctx, "INSERT INTO stream_test VALUES ("+itoa(i)+", 'row-"+itoa(i)+"')")
		if err != nil {
			t.Fatalf("INSERT error: %v", err)
		}
	}

	// Stream with page size 3.
	iter, err := conn.ExecuteStreaming(ctx, "SELECT * FROM stream_test ORDER BY id", 3)
	if err != nil {
		t.Fatalf("ExecuteStreaming error: %v", err)
	}
	defer iter.Close()

	// Check columns.
	cols := iter.Columns()
	if len(cols) != 2 {
		t.Fatalf("Columns() returned %d, want 2", len(cols))
	}

	// Total rows should be -1 (unknown for streaming).
	if iter.TotalRows() != -1 {
		t.Errorf("TotalRows() = %d, want -1", iter.TotalRows())
	}

	// Fetch first page.
	page1, err := iter.FetchNext(ctx)
	if err != nil {
		t.Fatalf("FetchNext page 1 error: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page 1 has %d rows, want 3", len(page1))
	}

	// Fetch second page.
	page2, err := iter.FetchNext(ctx)
	if err != nil {
		t.Fatalf("FetchNext page 2 error: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page 2 has %d rows, want 3", len(page2))
	}
}

func TestExecuteStreaming_10MillionRows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10M row test in short mode")
	}

	conn := openMemory(t)
	defer conn.Close()

	ctx := context.Background()

	// Create table and bulk-insert 10M rows via recursive CTE.
	_, err := conn.Execute(ctx, "CREATE TABLE big_test (id INTEGER PRIMARY KEY, val TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE error: %v", err)
	}

	const totalRows = 10_000_000
	t.Logf("inserting %d rows...", totalRows)
	_, err = conn.Execute(ctx, `
		WITH RECURSIVE cnt(x) AS (
			VALUES(1)
			UNION ALL
			SELECT x+1 FROM cnt WHERE x < 10000000
		)
		INSERT INTO big_test SELECT x, 'row-' || x FROM cnt
	`)
	if err != nil {
		t.Fatalf("bulk INSERT error: %v", err)
	}
	t.Log("insert complete, starting streaming test")

	// Force GC and record baseline memory.
	runtime.GC()
	var baseline runtime.MemStats
	runtime.ReadMemStats(&baseline)

	// Stream with page size 1000 (same as production).
	const pageSize = 1000
	iter, err := conn.ExecuteStreaming(ctx, "SELECT * FROM big_test ORDER BY id", pageSize)
	if err != nil {
		t.Fatalf("ExecuteStreaming error: %v", err)
	}
	defer iter.Close()

	if len(iter.Columns()) != 2 {
		t.Fatalf("Columns() = %d, want 2", len(iter.Columns()))
	}

	// Drain all pages, keeping only the latest page in scope.
	var rowCount int64
	var pageCount int
	var peakAlloc uint64
	for {
		page, err := iter.FetchNext(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("FetchNext error after %d rows: %v", rowCount, err)
		}
		if len(page) == 0 {
			break
		}
		rowCount += int64(len(page))
		pageCount++

		// Sample memory every 1000 pages.
		if pageCount%1000 == 0 {
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			if mem.Alloc > peakAlloc {
				peakAlloc = mem.Alloc
			}
		}
	}

	// Final memory snapshot.
	runtime.GC()
	var final runtime.MemStats
	runtime.ReadMemStats(&final)
	if final.Alloc > peakAlloc {
		peakAlloc = final.Alloc
	}

	t.Logf("streamed %d rows in %d pages", rowCount, pageCount)
	t.Logf("baseline alloc: %d MB, peak alloc: %d MB",
		baseline.Alloc/1024/1024, peakAlloc/1024/1024)

	// Verify all rows were fetched.
	if rowCount != totalRows {
		t.Errorf("fetched %d rows, want %d", rowCount, totalRows)
	}

	// Each page should have been exactly pageSize (except possibly the last).
	expectedPages := totalRows / pageSize
	if totalRows%pageSize != 0 {
		expectedPages++
	}
	if pageCount != expectedPages {
		t.Errorf("got %d pages, want %d", pageCount, expectedPages)
	}

	// Memory guard: streaming should not hold all 10M rows in memory.
	// 10M rows × ~20 bytes each ≈ 200 MB if all held at once.
	// With streaming, peak overhead above baseline should stay well under 100 MB.
	overhead := peakAlloc - baseline.Alloc
	const maxOverhead = 100 * 1024 * 1024 // 100 MB
	if overhead > maxOverhead {
		t.Errorf("memory overhead = %d MB, want < 100 MB (streaming should not buffer all rows)",
			overhead/1024/1024)
	}
}

func TestCancel_InMemory(t *testing.T) {
	conn := openMemory(t)
	defer conn.Close()

	// Cancel should not error even when no query is running.
	if err := conn.Cancel(); err != nil {
		t.Errorf("Cancel() error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// openMemory creates an in-memory SQLite connection for testing.
func openMemory(t *testing.T) adapter.Connection {
	t.Helper()
	a := &sqliteAdapter{}
	conn, err := a.Connect(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Connect(:memory:) error: %v", err)
	}
	return conn
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
