package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"

	_ "modernc.org/sqlite"
)

func init() {
	adapter.Register(&sqliteAdapter{})
}

// sqliteAdapter implements adapter.Adapter for SQLite databases.
type sqliteAdapter struct{}

func (a *sqliteAdapter) Name() string     { return "sqlite" }
func (a *sqliteAdapter) DefaultPort() int { return 0 }

func (a *sqliteAdapter) Connect(ctx context.Context, dsn string) (adapter.Connection, error) {
	dsn = normalizeDSN(dsn)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}

	// Enable foreign keys.
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite enable foreign keys: %w", err)
	}

	dbName := dsn
	if dsn != ":memory:" {
		dbName = filepath.Base(dsn)
	}

	return &sqliteConn{
		db:     db,
		dsn:    dsn,
		dbName: dbName,
	}, nil
}

// normalizeDSN strips common SQLite URI prefixes.
func normalizeDSN(dsn string) string {
	if strings.HasPrefix(dsn, "sqlite://") {
		return strings.TrimPrefix(dsn, "sqlite://")
	}
	if strings.HasPrefix(dsn, "file:") {
		return strings.TrimPrefix(dsn, "file:")
	}
	return dsn
}

// sqliteConn implements adapter.Connection.
type sqliteConn struct {
	db     *sql.DB
	dsn    string
	dbName string

	mu       sync.Mutex
	cancelFn context.CancelFunc
}

func (c *sqliteConn) AdapterName() string  { return "sqlite" }
func (c *sqliteConn) DatabaseName() string { return c.dbName }

func (c *sqliteConn) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *sqliteConn) Close() error {
	return c.db.Close()
}

// Databases returns a single database entry for the opened SQLite file.
func (c *sqliteConn) Databases(ctx context.Context) ([]schema.Database, error) {
	tables, err := c.Tables(ctx, c.dbName, "main")
	if err != nil {
		return nil, err
	}
	return []schema.Database{
		{
			Name: c.dbName,
			Schemas: []schema.Schema{
				{
					Name:   "main",
					Tables: tables,
				},
			},
		},
	}, nil
}

// Tables returns all user tables in the database.
func (c *sqliteConn) Tables(ctx context.Context, db, schemaName string) ([]schema.Table, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("sqlite tables: %w", err)
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("sqlite tables scan: %w", err)
		}
		tables = append(tables, schema.Table{Name: name})
	}
	return tables, rows.Err()
}

// Columns returns column metadata for the given table using PRAGMA table_info.
func (c *sqliteConn) Columns(ctx context.Context, db, schemaName, table string) ([]schema.Column, error) {
	rows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite columns: %w", err)
	}
	defer rows.Close()

	var columns []schema.Column
	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("sqlite columns scan: %w", err)
		}
		col := schema.Column{
			Name:     name,
			Type:     colType,
			Nullable: notNull == 0,
			IsPK:     pk > 0,
		}
		if dfltValue.Valid {
			col.Default = dfltValue.String
		}
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// Indexes returns index information for the given table.
func (c *sqliteConn) Indexes(ctx context.Context, db, schemaName, table string) ([]schema.Index, error) {
	listRows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list(%q)", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite index_list: %w", err)
	}
	defer listRows.Close()

	type indexEntry struct {
		name   string
		unique bool
	}
	var entries []indexEntry
	for listRows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := listRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("sqlite index_list scan: %w", err)
		}
		entries = append(entries, indexEntry{name: name, unique: unique == 1})
	}
	if err := listRows.Err(); err != nil {
		return nil, err
	}

	var indexes []schema.Index
	for _, entry := range entries {
		infoRows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info(%q)", entry.name))
		if err != nil {
			return nil, fmt.Errorf("sqlite index_info: %w", err)
		}

		var cols []string
		for infoRows.Next() {
			var (
				seqno int
				cid   int
				name  string
			)
			if err := infoRows.Scan(&seqno, &cid, &name); err != nil {
				infoRows.Close()
				return nil, fmt.Errorf("sqlite index_info scan: %w", err)
			}
			cols = append(cols, name)
		}
		infoRows.Close()
		if err := infoRows.Err(); err != nil {
			return nil, err
		}

		indexes = append(indexes, schema.Index{
			Name:    entry.name,
			Columns: cols,
			Unique:  entry.unique,
		})
	}
	return indexes, nil
}

// ForeignKeys returns foreign key constraints for the given table.
func (c *sqliteConn) ForeignKeys(ctx context.Context, db, schemaName, table string) ([]schema.ForeignKey, error) {
	rows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%q)", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite foreign_key_list: %w", err)
	}
	defer rows.Close()

	// Group by id since a single FK can span multiple columns.
	type fkEntry struct {
		refTable   string
		columns    []string
		refColumns []string
	}
	fkMap := make(map[int]*fkEntry)
	var fkOrder []int

	for rows.Next() {
		var (
			id       int
			seq      int
			refTable string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, fmt.Errorf("sqlite foreign_key_list scan: %w", err)
		}
		entry, ok := fkMap[id]
		if !ok {
			entry = &fkEntry{refTable: refTable}
			fkMap[id] = entry
			fkOrder = append(fkOrder, id)
		}
		entry.columns = append(entry.columns, from)
		entry.refColumns = append(entry.refColumns, to)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var fks []schema.ForeignKey
	for _, id := range fkOrder {
		entry := fkMap[id]
		fks = append(fks, schema.ForeignKey{
			Name:       fmt.Sprintf("fk_%s_%d", table, id),
			Columns:    entry.columns,
			RefTable:   entry.refTable,
			RefColumns: entry.refColumns,
		})
	}
	return fks, nil
}

// Execute runs a query and returns the result.
func (c *sqliteConn) Execute(ctx context.Context, query string) (*adapter.QueryResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancelFn = cancel
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.cancelFn = nil
		c.mu.Unlock()
		cancel()
	}()

	trimmed := strings.TrimSpace(strings.ToUpper(query))
	isSelect := strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "PRAGMA") ||
		strings.HasPrefix(trimmed, "EXPLAIN") ||
		strings.HasPrefix(trimmed, "WITH")

	start := time.Now()

	if isSelect {
		return c.executeQuery(ctx, query, start)
	}
	return c.executeExec(ctx, query, start)
}

func (c *sqliteConn) executeQuery(ctx context.Context, query string, start time.Time) (*adapter.QueryResult, error) {
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("sqlite query: %w", err)
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("sqlite column types: %w", err)
	}

	cols := make([]adapter.ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		cols[i] = adapter.ColumnMeta{
			Name: ct.Name(),
			Type: ct.DatabaseTypeName(),
		}
		if nullable, ok := ct.Nullable(); ok {
			cols[i].Nullable = nullable
		}
	}

	var resultRows [][]string
	scanDest := make([]any, len(cols))
	for i := range scanDest {
		scanDest[i] = new(sql.NullString)
	}

	for rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("sqlite scan: %w", err)
		}
		row := make([]string, len(cols))
		for i, v := range scanDest {
			ns := v.(*sql.NullString)
			if ns.Valid {
				row[i] = ns.String
			} else {
				row[i] = "NULL"
			}
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("sqlite rows: %w", err)
	}

	return &adapter.QueryResult{
		Columns:  cols,
		Rows:     resultRows,
		RowCount: int64(len(resultRows)),
		Duration: time.Since(start),
		IsSelect: true,
	}, nil
}

func (c *sqliteConn) executeExec(ctx context.Context, query string, start time.Time) (*adapter.QueryResult, error) {
	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("sqlite exec: %w", err)
	}

	affected, _ := result.RowsAffected()
	duration := time.Since(start)

	return &adapter.QueryResult{
		RowCount: affected,
		Duration: duration,
		IsSelect: false,
		Message:  fmt.Sprintf("%d row(s) affected", affected),
	}, nil
}

// Cancel cancels any in-flight query.
func (c *sqliteConn) Cancel() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancelFn != nil {
		c.cancelFn()
	}
	return nil
}

// ExecuteStreaming returns a RowIterator for paginated access to query results.
func (c *sqliteConn) ExecuteStreaming(ctx context.Context, query string, pageSize int) (adapter.RowIterator, error) {
	// First, execute a probe query to discover column metadata.
	probeQuery := fmt.Sprintf("SELECT * FROM (%s) LIMIT 0", query)
	rows, err := c.db.QueryContext(ctx, probeQuery)
	if err != nil {
		return nil, fmt.Errorf("sqlite streaming probe: %w", err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		rows.Close()
		return nil, fmt.Errorf("sqlite streaming column types: %w", err)
	}

	cols := make([]adapter.ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		cols[i] = adapter.ColumnMeta{
			Name: ct.Name(),
			Type: ct.DatabaseTypeName(),
		}
		if nullable, ok := ct.Nullable(); ok {
			cols[i].Nullable = nullable
		}
	}
	rows.Close()

	return &rowIterator{
		db:       c.db,
		query:    query,
		pageSize: pageSize,
		offset:   0,
		cols:     cols,
	}, nil
}

// Completions returns autocomplete items for tables and their columns.
func (c *sqliteConn) Completions(ctx context.Context) ([]adapter.CompletionItem, error) {
	var items []adapter.CompletionItem

	tables, err := c.Tables(ctx, c.dbName, "main")
	if err != nil {
		return nil, err
	}

	for _, t := range tables {
		items = append(items, adapter.CompletionItem{
			Label:  t.Name,
			Kind:   adapter.CompletionTable,
			Detail: "table",
		})

		columns, err := c.Columns(ctx, c.dbName, "main", t.Name)
		if err != nil {
			continue
		}
		for _, col := range columns {
			items = append(items, adapter.CompletionItem{
				Label:  col.Name,
				Kind:   adapter.CompletionColumn,
				Detail: fmt.Sprintf("%s.%s (%s)", t.Name, col.Name, col.Type),
			})
		}
	}

	return items, nil
}

// rowIterator implements adapter.RowIterator with LIMIT/OFFSET pagination.
type rowIterator struct {
	db       *sql.DB
	query    string
	pageSize int
	offset   int
	cols     []adapter.ColumnMeta
}

func (it *rowIterator) Columns() []adapter.ColumnMeta {
	return it.cols
}

func (it *rowIterator) TotalRows() int64 {
	return -1
}

func (it *rowIterator) FetchNext(ctx context.Context) ([][]string, error) {
	paged := fmt.Sprintf("SELECT * FROM (%s) LIMIT %d OFFSET %d", it.query, it.pageSize, it.offset)
	rows, err := it.db.QueryContext(ctx, paged)
	if err != nil {
		return nil, fmt.Errorf("sqlite fetch next: %w", err)
	}
	defer rows.Close()

	data, err := scanAllRows(rows, len(it.cols))
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, io.EOF
	}

	it.offset += len(data)
	return data, nil
}

func (it *rowIterator) FetchPrev(ctx context.Context) ([][]string, error) {
	newOffset := it.offset - 2*it.pageSize
	if newOffset < 0 {
		newOffset = 0
	}
	if it.offset <= 0 {
		return nil, adapter.ErrNoBidirectional
	}

	paged := fmt.Sprintf("SELECT * FROM (%s) LIMIT %d OFFSET %d", it.query, it.pageSize, newOffset)
	rows, err := it.db.QueryContext(ctx, paged)
	if err != nil {
		return nil, fmt.Errorf("sqlite fetch prev: %w", err)
	}
	defer rows.Close()

	data, err := scanAllRows(rows, len(it.cols))
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, io.EOF
	}

	it.offset = newOffset + len(data)
	return data, nil
}

func (it *rowIterator) Close() error {
	return nil
}

// scanAllRows scans all rows from a result set into string slices.
func scanAllRows(rows *sql.Rows, colCount int) ([][]string, error) {
	scanDest := make([]any, colCount)
	for i := range scanDest {
		scanDest[i] = new(sql.NullString)
	}

	var result [][]string
	for rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("sqlite scan: %w", err)
		}
		row := make([]string, colCount)
		for i, v := range scanDest {
			ns := v.(*sql.NullString)
			if ns.Valid {
				row[i] = ns.String
			} else {
				row[i] = "NULL"
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
