package postgres

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/sadopc/gotermsql/internal/adapter"
)

// Default DSN for local Homebrew PostgreSQL.
// Override with GOTERMSQL_PG_DSN env var.
const defaultTestDSN = "postgres://localhost:5432/gotermsql_test?sslmode=disable"

func testDSN() string {
	if dsn := os.Getenv("GOTERMSQL_PG_DSN"); dsn != "" {
		return dsn
	}
	return defaultTestDSN
}

func connectForTest(t *testing.T) adapter.Connection {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a := &postgresAdapter{}
	conn, err := a.Connect(ctx, testDSN())
	if err != nil {
		t.Skipf("skipping: cannot connect to PostgreSQL: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestIntegration_ConnectAndPing(t *testing.T) {
	conn := connectForTest(t)

	ctx := context.Background()
	if err := conn.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if conn.AdapterName() != "postgres" {
		t.Errorf("AdapterName() = %q, want %q", conn.AdapterName(), "postgres")
	}
	if conn.DatabaseName() != "gotermsql_test" {
		t.Errorf("DatabaseName() = %q, want %q", conn.DatabaseName(), "gotermsql_test")
	}
}

func TestIntegration_Execute_DDL_and_DML(t *testing.T) {
	conn := connectForTest(t)
	ctx := context.Background()

	// Cleanup from any previous run
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_orders")
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_users")

	// CREATE TABLE
	res, err := conn.Execute(ctx, `
		CREATE TABLE test_users (
			id    SERIAL PRIMARY KEY,
			name  VARCHAR(100) NOT NULL,
			email VARCHAR(200) UNIQUE,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if res.IsSelect {
		t.Error("CREATE TABLE should not be a SELECT result")
	}

	// INSERT
	res, err = conn.Execute(ctx, `
		INSERT INTO test_users (name, email) VALUES
		('Alice', 'alice@example.com'),
		('Bob', 'bob@example.com'),
		('Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	if res.RowCount != 3 {
		t.Errorf("INSERT RowCount = %d, want 3", res.RowCount)
	}

	// SELECT
	res, err = conn.Execute(ctx, "SELECT id, name, email, active FROM test_users ORDER BY id")
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if !res.IsSelect {
		t.Error("SELECT should be a SELECT result")
	}
	if len(res.Rows) != 3 {
		t.Fatalf("SELECT returned %d rows, want 3", len(res.Rows))
	}
	if res.Rows[0][1] != "Alice" {
		t.Errorf("first row name = %q, want %q", res.Rows[0][1], "Alice")
	}
	if len(res.Columns) != 4 {
		t.Errorf("got %d columns, want 4", len(res.Columns))
	}

	// UPDATE
	res, err = conn.Execute(ctx, "UPDATE test_users SET active = false WHERE name = 'Bob'")
	if err != nil {
		t.Fatalf("UPDATE: %v", err)
	}
	if res.RowCount != 1 {
		t.Errorf("UPDATE RowCount = %d, want 1", res.RowCount)
	}

	// DELETE
	res, err = conn.Execute(ctx, "DELETE FROM test_users WHERE name = 'Charlie'")
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	if res.RowCount != 1 {
		t.Errorf("DELETE RowCount = %d, want 1", res.RowCount)
	}

	// Verify remaining data
	res, err = conn.Execute(ctx, "SELECT name, active FROM test_users ORDER BY id")
	if err != nil {
		t.Fatalf("final SELECT: %v", err)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(res.Rows))
	}
	if res.Rows[1][1] != "false" {
		t.Errorf("Bob active = %q, want %q", res.Rows[1][1], "false")
	}

	// Cleanup
	conn.Execute(ctx, "DROP TABLE test_users")
}

func TestIntegration_Introspection(t *testing.T) {
	conn := connectForTest(t)
	ctx := context.Background()

	// Setup
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_orders")
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_products")
	conn.Execute(ctx, `
		CREATE TABLE test_products (
			id    SERIAL PRIMARY KEY,
			name  VARCHAR(100) NOT NULL,
			price NUMERIC(10,2)
		)
	`)
	conn.Execute(ctx, `
		CREATE TABLE test_orders (
			id         SERIAL PRIMARY KEY,
			product_id INT REFERENCES test_products(id),
			quantity   INT NOT NULL DEFAULT 1
		)
	`)
	conn.Execute(ctx, "CREATE INDEX idx_test_orders_product ON test_orders(product_id)")

	t.Cleanup(func() {
		conn.Execute(ctx, "DROP TABLE IF EXISTS test_orders")
		conn.Execute(ctx, "DROP TABLE IF EXISTS test_products")
	})

	// Databases
	t.Run("Databases", func(t *testing.T) {
		dbs, err := conn.Databases(ctx)
		if err != nil {
			t.Fatalf("Databases: %v", err)
		}
		found := false
		for _, db := range dbs {
			if db.Name == "gotermsql_test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("gotermsql_test not found in Databases()")
		}
	})

	// Tables
	t.Run("Tables", func(t *testing.T) {
		tables, err := conn.Tables(ctx, "gotermsql_test", "public")
		if err != nil {
			t.Fatalf("Tables: %v", err)
		}
		names := map[string]bool{}
		for _, tbl := range tables {
			names[tbl.Name] = true
		}
		if !names["test_products"] {
			t.Error("test_products not found in Tables()")
		}
		if !names["test_orders"] {
			t.Error("test_orders not found in Tables()")
		}
	})

	// Columns
	t.Run("Columns", func(t *testing.T) {
		cols, err := conn.Columns(ctx, "gotermsql_test", "public", "test_products")
		if err != nil {
			t.Fatalf("Columns: %v", err)
		}
		if len(cols) != 3 {
			t.Fatalf("got %d columns, want 3", len(cols))
		}
		colMap := map[string]bool{}
		for _, c := range cols {
			colMap[c.Name] = true
			if c.Name == "id" && !c.IsPK {
				t.Error("id column should be PK")
			}
		}
		for _, name := range []string{"id", "name", "price"} {
			if !colMap[name] {
				t.Errorf("column %q not found", name)
			}
		}
	})

	// Indexes
	t.Run("Indexes", func(t *testing.T) {
		idxs, err := conn.Indexes(ctx, "", "public", "test_orders")
		if err != nil {
			t.Fatalf("Indexes: %v", err)
		}
		found := false
		for _, idx := range idxs {
			if idx.Name == "idx_test_orders_product" {
				found = true
				if len(idx.Columns) != 1 || idx.Columns[0] != "product_id" {
					t.Errorf("index columns = %v, want [product_id]", idx.Columns)
				}
			}
		}
		if !found {
			t.Error("idx_test_orders_product not found in Indexes()")
		}
	})

	// Foreign Keys
	t.Run("ForeignKeys", func(t *testing.T) {
		fks, err := conn.ForeignKeys(ctx, "", "public", "test_orders")
		if err != nil {
			t.Fatalf("ForeignKeys: %v", err)
		}
		if len(fks) == 0 {
			t.Fatal("expected at least 1 foreign key")
		}
		fk := fks[0]
		if fk.RefTable != "test_products" {
			t.Errorf("FK RefTable = %q, want %q", fk.RefTable, "test_products")
		}
		if len(fk.Columns) != 1 || fk.Columns[0] != "product_id" {
			t.Errorf("FK Columns = %v, want [product_id]", fk.Columns)
		}
		if len(fk.RefColumns) != 1 || fk.RefColumns[0] != "id" {
			t.Errorf("FK RefColumns = %v, want [id]", fk.RefColumns)
		}
	})
}

func TestIntegration_Streaming(t *testing.T) {
	conn := connectForTest(t)
	ctx := context.Background()

	// Setup
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_stream")
	conn.Execute(ctx, "CREATE TABLE test_stream (id INT, val TEXT)")
	conn.Execute(ctx, `
		INSERT INTO test_stream (id, val)
		SELECT g, 'row-' || g FROM generate_series(1, 50) AS g
	`)
	t.Cleanup(func() {
		conn.Execute(ctx, "DROP TABLE IF EXISTS test_stream")
	})

	iter, err := conn.ExecuteStreaming(ctx, "SELECT * FROM test_stream ORDER BY id", 10)
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	defer iter.Close()

	// Check columns
	cols := iter.Columns()
	if len(cols) != 2 {
		t.Fatalf("got %d columns, want 2", len(cols))
	}

	// Fetch first page
	rows, err := iter.FetchNext(ctx)
	if err != nil {
		t.Fatalf("FetchNext page 1: %v", err)
	}
	if len(rows) != 10 {
		t.Fatalf("page 1 got %d rows, want 10", len(rows))
	}
	if rows[0][0] != "1" {
		t.Errorf("first row id = %q, want %q", rows[0][0], "1")
	}

	// Fetch second page
	rows, err = iter.FetchNext(ctx)
	if err != nil {
		t.Fatalf("FetchNext page 2: %v", err)
	}
	if len(rows) != 10 {
		t.Fatalf("page 2 got %d rows, want 10", len(rows))
	}
	if rows[0][0] != "11" {
		t.Errorf("page 2 first row id = %q, want %q", rows[0][0], "11")
	}

	// Go back to previous page
	rows, err = iter.FetchPrev(ctx)
	if err != nil {
		t.Fatalf("FetchPrev: %v", err)
	}
	if len(rows) != 10 {
		t.Fatalf("prev page got %d rows, want 10", len(rows))
	}

	// Drain remaining pages to reach EOF
	for {
		rows, err = iter.FetchNext(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("FetchNext drain: %v", err)
		}
		if len(rows) == 0 {
			break
		}
	}
}

func TestIntegration_Completions(t *testing.T) {
	conn := connectForTest(t)
	ctx := context.Background()

	// Setup
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_comp")
	conn.Execute(ctx, "CREATE TABLE test_comp (id INT, description TEXT)")
	t.Cleanup(func() {
		conn.Execute(ctx, "DROP TABLE IF EXISTS test_comp")
	})

	items, err := conn.Completions(ctx)
	if err != nil {
		t.Fatalf("Completions: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected completions, got none")
	}

	foundTable := false
	foundColumn := false
	for _, item := range items {
		if item.Label == "test_comp" && item.Kind == adapter.CompletionTable {
			foundTable = true
		}
		if item.Label == "description" && item.Kind == adapter.CompletionColumn {
			foundColumn = true
		}
	}
	if !foundTable {
		t.Error("test_comp table not found in completions")
	}
	if !foundColumn {
		t.Error("description column not found in completions")
	}
}

func TestIntegration_DataTypes(t *testing.T) {
	conn := connectForTest(t)
	ctx := context.Background()

	// Setup
	conn.Execute(ctx, "DROP TABLE IF EXISTS test_types")
	conn.Execute(ctx, `
		CREATE TABLE test_types (
			c_bool     BOOLEAN,
			c_int      INT,
			c_bigint   BIGINT,
			c_float    DOUBLE PRECISION,
			c_numeric  NUMERIC(10,2),
			c_text     TEXT,
			c_varchar  VARCHAR(50),
			c_date     DATE,
			c_ts       TIMESTAMP,
			c_json     JSONB,
			c_uuid     UUID
		)
	`)
	conn.Execute(ctx, `
		INSERT INTO test_types VALUES (
			true, 42, 9999999999, 3.14, 99.99,
			'hello world', 'varchar val',
			'2024-06-15', '2024-06-15 14:30:00',
			'{"key": "value"}',
			'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
		)
	`)
	t.Cleanup(func() {
		conn.Execute(ctx, "DROP TABLE IF EXISTS test_types")
	})

	res, err := conn.Execute(ctx, "SELECT * FROM test_types")
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(res.Rows))
	}
	row := res.Rows[0]

	checks := []struct {
		idx  int
		name string
		want string
	}{
		{0, "bool", "true"},
		{1, "int", "42"},
		{2, "bigint", "9999999999"},
		{4, "numeric", "99.99"},
		{5, "text", "hello world"},
		{6, "varchar", "varchar val"},
		{7, "date", "2024-06-15"},
		{10, "uuid", "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"},
	}
	for _, c := range checks {
		if row[c.idx] != c.want {
			t.Errorf("%s: got %q, want %q", c.name, row[c.idx], c.want)
		}
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	conn := connectForTest(t)
	ctx := context.Background()

	// Invalid SQL
	_, err := conn.Execute(ctx, "SELECT * FROM nonexistent_table_xyz")
	if err == nil {
		t.Error("expected error for nonexistent table, got nil")
	}

	// Syntax error
	_, err = conn.Execute(ctx, "SELEC broken")
	if err == nil {
		t.Error("expected error for syntax error, got nil")
	}
}
