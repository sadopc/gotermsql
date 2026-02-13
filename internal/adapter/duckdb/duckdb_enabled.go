//go:build duckdb

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

func init() {
	adapter.Register(&duckdbAdapter{})
}

// ---------------------------------------------------------------------------
// Adapter
// ---------------------------------------------------------------------------

type duckdbAdapter struct{}

func (a *duckdbAdapter) Name() string     { return "duckdb" }
func (a *duckdbAdapter) DefaultPort() int { return 0 }

func (a *duckdbAdapter) Connect(ctx context.Context, dsn string) (adapter.Connection, error) {
	// Strip the "duckdb://" prefix if present.
	dsn = strings.TrimPrefix(dsn, "duckdb://")
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdb: open: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("duckdb: ping: %w", err)
	}

	return &duckdbConn{
		db:  db,
		dsn: dsn,
	}, nil
}

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

type duckdbConn struct {
	db  *sql.DB
	dsn string

	mu     sync.Mutex
	cancel context.CancelFunc
}

func (c *duckdbConn) DatabaseName() string { return c.dsn }
func (c *duckdbConn) AdapterName() string  { return "duckdb" }

func (c *duckdbConn) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *duckdbConn) Close() error {
	return c.db.Close()
}

// Cancel cancels the currently running query, if any.
func (c *duckdbConn) Cancel() error {
	c.mu.Lock()
	fn := c.cancel
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Introspection
// ---------------------------------------------------------------------------

func (c *duckdbConn) Databases(ctx context.Context) ([]schema.Database, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT database_name FROM duckdb_databases() ORDER BY database_name`)
	if err != nil {
		return nil, fmt.Errorf("duckdb: databases: %w", err)
	}
	defer rows.Close()

	var dbs []schema.Database
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("duckdb: databases scan: %w", err)
		}
		dbs = append(dbs, schema.Database{Name: name})
	}
	return dbs, rows.Err()
}

func (c *duckdbConn) Tables(ctx context.Context, db, schemaName string) ([]schema.Table, error) {
	query := `SELECT table_name
		FROM information_schema.tables
		WHERE table_catalog = ? AND table_schema = ?
		ORDER BY table_name`
	rows, err := c.db.QueryContext(ctx, query, db, schemaName)
	if err != nil {
		return nil, fmt.Errorf("duckdb: tables: %w", err)
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("duckdb: tables scan: %w", err)
		}
		tables = append(tables, schema.Table{Name: name})
	}
	return tables, rows.Err()
}

func (c *duckdbConn) Columns(ctx context.Context, db, schemaName, table string) ([]schema.Column, error) {
	query := `SELECT column_name,
			data_type,
			CASE WHEN is_nullable = 'YES' THEN true ELSE false END,
			COALESCE(column_default, ''),
			CASE WHEN column_name IN (
				SELECT kcu.column_name
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
				  ON tc.constraint_name = kcu.constraint_name
				  AND tc.table_catalog = kcu.table_catalog
				  AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'PRIMARY KEY'
				  AND tc.table_catalog = ?
				  AND tc.table_schema = ?
				  AND tc.table_name = ?
			) THEN true ELSE false END
		FROM information_schema.columns
		WHERE table_catalog = ? AND table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`
	rows, err := c.db.QueryContext(ctx, query, db, schemaName, table, db, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("duckdb: columns: %w", err)
	}
	defer rows.Close()

	var cols []schema.Column
	for rows.Next() {
		var col schema.Column
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Default, &col.IsPK); err != nil {
			return nil, fmt.Errorf("duckdb: columns scan: %w", err)
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (c *duckdbConn) Indexes(ctx context.Context, db, schemaName, table string) ([]schema.Index, error) {
	query := `SELECT index_name, is_unique, sql
		FROM duckdb_indexes()
		WHERE database_name = ? AND schema_name = ? AND table_name = ?
		ORDER BY index_name`
	rows, err := c.db.QueryContext(ctx, query, db, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("duckdb: indexes: %w", err)
	}
	defer rows.Close()

	var indexes []schema.Index
	for rows.Next() {
		var idx schema.Index
		var isUnique bool
		var sqlStr sql.NullString
		if err := rows.Scan(&idx.Name, &isUnique, &sqlStr); err != nil {
			return nil, fmt.Errorf("duckdb: indexes scan: %w", err)
		}
		idx.Unique = isUnique
		// Extract column names from the index SQL if available.
		idx.Columns = parseIndexColumns(sqlStr.String)
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

// parseIndexColumns extracts column names from a CREATE INDEX SQL statement.
// Example: "CREATE INDEX idx ON tbl (col1, col2)" -> ["col1", "col2"]
func parseIndexColumns(sqlStr string) []string {
	if sqlStr == "" {
		return nil
	}
	start := strings.LastIndex(sqlStr, "(")
	end := strings.LastIndex(sqlStr, ")")
	if start < 0 || end <= start {
		return nil
	}
	inner := sqlStr[start+1 : end]
	parts := strings.Split(inner, ",")
	var cols []string
	for _, p := range parts {
		col := strings.TrimSpace(p)
		if col != "" {
			cols = append(cols, col)
		}
	}
	return cols
}

func (c *duckdbConn) ForeignKeys(ctx context.Context, db, schemaName, table string) ([]schema.ForeignKey, error) {
	query := `SELECT
			rc.constraint_name,
			kcu.column_name,
			kcu2.table_name AS ref_table,
			kcu2.column_name AS ref_column
		FROM information_schema.referential_constraints rc
		JOIN information_schema.key_column_usage kcu
		  ON rc.constraint_catalog = kcu.constraint_catalog
		  AND rc.constraint_schema = kcu.constraint_schema
		  AND rc.constraint_name = kcu.constraint_name
		JOIN information_schema.key_column_usage kcu2
		  ON rc.unique_constraint_catalog = kcu2.constraint_catalog
		  AND rc.unique_constraint_schema = kcu2.constraint_schema
		  AND rc.unique_constraint_name = kcu2.constraint_name
		  AND kcu.ordinal_position = kcu2.ordinal_position
		WHERE kcu.table_catalog = ? AND kcu.table_schema = ? AND kcu.table_name = ?
		ORDER BY rc.constraint_name, kcu.ordinal_position`
	rows, err := c.db.QueryContext(ctx, query, db, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("duckdb: foreign keys: %w", err)
	}
	defer rows.Close()

	fkMap := map[string]*schema.ForeignKey{}
	var fkOrder []string
	for rows.Next() {
		var name, col, refTable, refCol string
		if err := rows.Scan(&name, &col, &refTable, &refCol); err != nil {
			return nil, fmt.Errorf("duckdb: foreign keys scan: %w", err)
		}
		fk, ok := fkMap[name]
		if !ok {
			fk = &schema.ForeignKey{Name: name, RefTable: refTable}
			fkMap[name] = fk
			fkOrder = append(fkOrder, name)
		}
		fk.Columns = append(fk.Columns, col)
		fk.RefColumns = append(fk.RefColumns, refCol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	fks := make([]schema.ForeignKey, 0, len(fkOrder))
	for _, name := range fkOrder {
		fks = append(fks, *fkMap[name])
	}
	return fks, nil
}

// ---------------------------------------------------------------------------
// Query execution
// ---------------------------------------------------------------------------

func (c *duckdbConn) Execute(ctx context.Context, query string) (*adapter.QueryResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()
	defer func() {
		cancel()
		c.mu.Lock()
		c.cancel = nil
		c.mu.Unlock()
	}()

	start := time.Now()
	trimmed := strings.TrimSpace(query)
	isSelect := isSelectQuery(trimmed)

	if isSelect {
		return c.executeSelect(ctx, query, start)
	}
	return c.executeExec(ctx, query, start)
}

func isSelectQuery(q string) bool {
	upper := strings.ToUpper(q)
	for _, prefix := range []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "EXPLAIN", "PRAGMA", "FROM"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

func (c *duckdbConn) executeSelect(ctx context.Context, query string, start time.Time) (*adapter.QueryResult, error) {
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("duckdb: query: %w", err)
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("duckdb: column types: %w", err)
	}

	cols := make([]adapter.ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		nullable, _ := ct.Nullable()
		cols[i] = adapter.ColumnMeta{
			Name:     ct.Name(),
			Type:     ct.DatabaseTypeName(),
			Nullable: nullable,
		}
	}

	var resultRows [][]string
	nCols := len(cols)
	truncated := false
	for rows.Next() {
		if len(resultRows) >= adapter.DefaultMaxRows {
			truncated = true
			break
		}
		vals := make([]sql.NullString, nCols)
		ptrs := make([]any, nCols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("duckdb: scan: %w", err)
		}
		row := make([]string, nCols)
		for i, v := range vals {
			if v.Valid {
				row[i] = v.String
			} else {
				row[i] = "NULL"
			}
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdb: rows iteration: %w", err)
	}

	return &adapter.QueryResult{
		Columns:   cols,
		Rows:      resultRows,
		RowCount:  int64(len(resultRows)),
		Duration:  time.Since(start),
		IsSelect:  true,
		Truncated: truncated,
	}, nil
}

func (c *duckdbConn) executeExec(ctx context.Context, query string, start time.Time) (*adapter.QueryResult, error) {
	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("duckdb: exec: %w", err)
	}

	affected, _ := result.RowsAffected()
	return &adapter.QueryResult{
		RowCount: affected,
		Duration: time.Since(start),
		IsSelect: false,
		Message:  fmt.Sprintf("%d row(s) affected", affected),
	}, nil
}

// ---------------------------------------------------------------------------
// Streaming (LIMIT/OFFSET pagination)
// ---------------------------------------------------------------------------

func (c *duckdbConn) ExecuteStreaming(ctx context.Context, query string, pageSize int) (adapter.RowIterator, error) {
	// Run a quick query to discover columns without fetching all rows.
	probeCtx, probeCancel := context.WithCancel(ctx)
	defer probeCancel()

	probeQuery := fmt.Sprintf("SELECT * FROM (%s) AS __probe LIMIT 0", query)
	probeRows, err := c.db.QueryContext(probeCtx, probeQuery)
	if err != nil {
		return nil, fmt.Errorf("duckdb: streaming probe: %w", err)
	}

	colTypes, err := probeRows.ColumnTypes()
	if err != nil {
		probeRows.Close()
		return nil, fmt.Errorf("duckdb: streaming column types: %w", err)
	}
	probeRows.Close()

	cols := make([]adapter.ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		nullable, _ := ct.Nullable()
		cols[i] = adapter.ColumnMeta{
			Name:     ct.Name(),
			Type:     ct.DatabaseTypeName(),
			Nullable: nullable,
		}
	}

	return &duckdbIterator{
		db:       c.db,
		query:    query,
		pageSize: pageSize,
		cols:     cols,
		offset:   0,
	}, nil
}

type duckdbIterator struct {
	db       *sql.DB
	query    string
	pageSize int
	cols     []adapter.ColumnMeta
	offset   int
	done     bool
}

func (it *duckdbIterator) Columns() []adapter.ColumnMeta { return it.cols }
func (it *duckdbIterator) TotalRows() int64              { return -1 }
func (it *duckdbIterator) Close() error                  { return nil }

func (it *duckdbIterator) FetchNext(ctx context.Context) ([][]string, error) {
	if it.done {
		return nil, io.EOF
	}

	paged := fmt.Sprintf("SELECT * FROM (%s) AS __paged LIMIT %d OFFSET %d", it.query, it.pageSize, it.offset)
	rows, err := it.db.QueryContext(ctx, paged)
	if err != nil {
		return nil, fmt.Errorf("duckdb: fetch next: %w", err)
	}
	defer rows.Close()

	page, err := scanPage(rows, len(it.cols))
	if err != nil {
		return nil, err
	}

	if len(page) < it.pageSize {
		it.done = true
	}
	if len(page) == 0 {
		return nil, io.EOF
	}
	it.offset += len(page)
	return page, nil
}

func (it *duckdbIterator) FetchPrev(ctx context.Context) ([][]string, error) {
	if it.offset <= 0 {
		return nil, io.EOF
	}

	newOffset := it.offset - 2*it.pageSize
	if newOffset < 0 {
		newOffset = 0
	}

	paged := fmt.Sprintf("SELECT * FROM (%s) AS __paged LIMIT %d OFFSET %d", it.query, it.pageSize, newOffset)
	rows, err := it.db.QueryContext(ctx, paged)
	if err != nil {
		return nil, fmt.Errorf("duckdb: fetch prev: %w", err)
	}
	defer rows.Close()

	page, err := scanPage(rows, len(it.cols))
	if err != nil {
		return nil, err
	}

	if len(page) == 0 {
		return nil, io.EOF
	}
	it.offset = newOffset + len(page)
	it.done = false
	return page, nil
}

func scanPage(rows *sql.Rows, nCols int) ([][]string, error) {
	var page [][]string
	for rows.Next() {
		vals := make([]sql.NullString, nCols)
		ptrs := make([]any, nCols)
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("duckdb: scan page: %w", err)
		}
		row := make([]string, nCols)
		for i, v := range vals {
			if v.Valid {
				row[i] = v.String
			} else {
				row[i] = "NULL"
			}
		}
		page = append(page, row)
	}
	return page, rows.Err()
}

// ---------------------------------------------------------------------------
// Completions
// ---------------------------------------------------------------------------

func (c *duckdbConn) Completions(ctx context.Context) ([]adapter.CompletionItem, error) {
	var items []adapter.CompletionItem

	// Tables and views
	tableRows, err := c.db.QueryContext(ctx,
		`SELECT table_catalog, table_schema, table_name, table_type
		 FROM information_schema.tables
		 ORDER BY table_catalog, table_schema, table_name`)
	if err != nil {
		return nil, fmt.Errorf("duckdb: completions tables: %w", err)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var catalog, sch, name, typ string
		if err := tableRows.Scan(&catalog, &sch, &name, &typ); err != nil {
			return nil, fmt.Errorf("duckdb: completions tables scan: %w", err)
		}
		kind := adapter.CompletionTable
		if strings.Contains(strings.ToUpper(typ), "VIEW") {
			kind = adapter.CompletionView
		}
		items = append(items, adapter.CompletionItem{
			Label:  name,
			Kind:   kind,
			Detail: fmt.Sprintf("%s.%s (%s)", catalog, sch, typ),
		})
	}
	if err := tableRows.Err(); err != nil {
		return nil, err
	}

	// Columns
	colRows, err := c.db.QueryContext(ctx,
		`SELECT table_name, column_name, data_type
		 FROM information_schema.columns
		 ORDER BY table_name, ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("duckdb: completions columns: %w", err)
	}
	defer colRows.Close()

	for colRows.Next() {
		var tbl, col, dtype string
		if err := colRows.Scan(&tbl, &col, &dtype); err != nil {
			return nil, fmt.Errorf("duckdb: completions columns scan: %w", err)
		}
		items = append(items, adapter.CompletionItem{
			Label:  col,
			Kind:   adapter.CompletionColumn,
			Detail: fmt.Sprintf("%s.%s %s", tbl, col, dtype),
		})
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
