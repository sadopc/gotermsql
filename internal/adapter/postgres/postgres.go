package postgres

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

func init() {
	adapter.Register(&postgresAdapter{})
}

// postgresAdapter implements adapter.Adapter for PostgreSQL.
type postgresAdapter struct{}

func (a *postgresAdapter) Name() string     { return "postgres" }
func (a *postgresAdapter) DefaultPort() int { return 5432 }

func (a *postgresAdapter) Connect(ctx context.Context, dsn string) (adapter.Connection, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	dbName := extractDBName(dsn)

	return &pgConn{
		pool:   pool,
		dsn:    dsn,
		dbName: dbName,
	}, nil
}

// extractDBName parses the database name from the DSN.
func extractDBName(dsn string) string {
	if dsn == "" {
		return ""
	}
	// Try URL format first (postgres://... or postgresql://...)
	u, err := url.Parse(dsn)
	if err == nil && u.Scheme != "" {
		return strings.TrimPrefix(u.Path, "/")
	}
	// Fallback: keyword=value format (e.g. "host=localhost dbname=myapp")
	for _, part := range strings.Fields(dsn) {
		if strings.HasPrefix(part, "dbname=") {
			return strings.TrimPrefix(part, "dbname=")
		}
	}
	return ""
}

// pgConn implements adapter.Connection for PostgreSQL.
type pgConn struct {
	pool     *pgxpool.Pool
	dsn      string
	dbName   string
	cancelMu sync.Mutex
	cancelFn context.CancelFunc
}

func (c *pgConn) DatabaseName() string { return c.dbName }
func (c *pgConn) AdapterName() string  { return "postgres" }

func (c *pgConn) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *pgConn) Close() error {
	c.pool.Close()
	return nil
}

// Cancel cancels the currently running query, if any.
func (c *pgConn) Cancel() error {
	c.cancelMu.Lock()
	fn := c.cancelFn
	c.cancelMu.Unlock()
	if fn != nil {
		fn()
	}
	return nil
}

func (c *pgConn) setCancel(fn context.CancelFunc) {
	c.cancelMu.Lock()
	c.cancelFn = fn
	c.cancelMu.Unlock()
}

func (c *pgConn) clearCancel() {
	c.cancelMu.Lock()
	c.cancelFn = nil
	c.cancelMu.Unlock()
}

// ---------------------------------------------------------------------------
// Introspection
// ---------------------------------------------------------------------------

func (c *pgConn) Databases(ctx context.Context) ([]schema.Database, error) {
	// List all non-template databases.
	dbRows, err := c.pool.Query(ctx,
		`SELECT datname FROM pg_database
		 WHERE datistemplate = false
		 ORDER BY datname`)
	if err != nil {
		return nil, fmt.Errorf("databases: %w", err)
	}
	defer dbRows.Close()

	var dbNames []string
	for dbRows.Next() {
		var name string
		if err := dbRows.Scan(&name); err != nil {
			return nil, fmt.Errorf("databases scan: %w", err)
		}
		dbNames = append(dbNames, name)
	}
	if err := dbRows.Err(); err != nil {
		return nil, err
	}

	// For the connected database, load schemas and tables.
	// PostgreSQL only allows querying information_schema for the current database.
	var dbs []schema.Database
	for _, name := range dbNames {
		db := schema.Database{Name: name}

		if name == c.dbName {
			schemas, err := c.loadSchemas(ctx, name)
			if err == nil {
				db.Schemas = schemas
			}
		}

		dbs = append(dbs, db)
	}
	return dbs, nil
}

// loadSchemas queries the user-visible schemas and their tables for the connected database.
func (c *pgConn) loadSchemas(ctx context.Context, dbName string) ([]schema.Schema, error) {
	rows, err := c.pool.Query(ctx,
		`SELECT schema_name FROM information_schema.schemata
		 WHERE catalog_name = $1
		   AND schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		 ORDER BY schema_name`, dbName)
	if err != nil {
		return nil, fmt.Errorf("schemas: %w", err)
	}
	defer rows.Close()

	var schemas []schema.Schema
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("schemas scan: %w", err)
		}

		tables, _ := c.Tables(ctx, dbName, name)
		schemas = append(schemas, schema.Schema{
			Name:   name,
			Tables: tables,
		})
	}
	return schemas, rows.Err()
}

func (c *pgConn) Tables(ctx context.Context, db, schemaName string) ([]schema.Table, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	rows, err := c.pool.Query(ctx,
		`SELECT table_name
		 FROM information_schema.tables
		 WHERE table_catalog = $1
		   AND table_schema  = $2
		   AND table_type    = 'BASE TABLE'
		 ORDER BY table_name`, db, schemaName)
	if err != nil {
		return nil, fmt.Errorf("tables: %w", err)
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("tables scan: %w", err)
		}
		tables = append(tables, schema.Table{Name: name})
	}
	return tables, rows.Err()
}

func (c *pgConn) Columns(ctx context.Context, db, schemaName, table string) ([]schema.Column, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	// Fetch primary key column names for this table.
	pkSet, err := c.primaryKeyColumns(ctx, schemaName, table)
	if err != nil {
		return nil, err
	}

	rows, err := c.pool.Query(ctx,
		`SELECT column_name,
		        data_type,
		        is_nullable,
		        COALESCE(column_default, '')
		 FROM information_schema.columns
		 WHERE table_catalog = $1
		   AND table_schema  = $2
		   AND table_name    = $3
		 ORDER BY ordinal_position`, db, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("columns: %w", err)
	}
	defer rows.Close()

	var cols []schema.Column
	for rows.Next() {
		var (
			name, dtype, nullable, dflt string
		)
		if err := rows.Scan(&name, &dtype, &nullable, &dflt); err != nil {
			return nil, fmt.Errorf("columns scan: %w", err)
		}
		cols = append(cols, schema.Column{
			Name:     name,
			Type:     dtype,
			Nullable: nullable == "YES",
			Default:  dflt,
			IsPK:     pkSet[name],
		})
	}
	return cols, rows.Err()
}

// primaryKeyColumns returns a set of column names that belong to the primary key.
func (c *pgConn) primaryKeyColumns(ctx context.Context, schemaName, table string) (map[string]bool, error) {
	rows, err := c.pool.Query(ctx,
		`SELECT a.attname
		 FROM pg_index i
		 JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		 WHERE i.indrelid = ($1 || '.' || $2)::regclass
		   AND i.indisprimary`, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("primary keys: %w", err)
	}
	defer rows.Close()

	pk := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("primary keys scan: %w", err)
		}
		pk[name] = true
	}
	return pk, rows.Err()
}

func (c *pgConn) Indexes(ctx context.Context, db, schemaName, table string) ([]schema.Index, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	rows, err := c.pool.Query(ctx,
		`SELECT i.relname                        AS index_name,
		        array_agg(a.attname ORDER BY k.n) AS columns,
		        ix.indisunique                     AS is_unique
		 FROM pg_index ix
		 JOIN pg_class  t ON t.oid  = ix.indrelid
		 JOIN pg_class  i ON i.oid  = ix.indexrelid
		 JOIN pg_namespace n ON n.oid = t.relnamespace
		 JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n) ON true
		 JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
		 WHERE n.nspname = $1
		   AND t.relname = $2
		 GROUP BY i.relname, ix.indisunique
		 ORDER BY i.relname`, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("indexes: %w", err)
	}
	defer rows.Close()

	var indexes []schema.Index
	for rows.Next() {
		var (
			name   string
			cols   []string
			unique bool
		)
		if err := rows.Scan(&name, &cols, &unique); err != nil {
			return nil, fmt.Errorf("indexes scan: %w", err)
		}
		indexes = append(indexes, schema.Index{
			Name:    name,
			Columns: cols,
			Unique:  unique,
		})
	}
	return indexes, rows.Err()
}

func (c *pgConn) ForeignKeys(ctx context.Context, db, schemaName, table string) ([]schema.ForeignKey, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	rows, err := c.pool.Query(ctx,
		`SELECT tc.constraint_name,
		        kcu.column_name,
		        ccu.table_name  AS ref_table,
		        ccu.column_name AS ref_column
		 FROM information_schema.table_constraints tc
		 JOIN information_schema.key_column_usage kcu
		      ON kcu.constraint_name = tc.constraint_name
		     AND kcu.table_schema    = tc.table_schema
		 JOIN information_schema.constraint_column_usage ccu
		      ON ccu.constraint_name = tc.constraint_name
		     AND ccu.table_schema    = tc.table_schema
		 WHERE tc.constraint_type = 'FOREIGN KEY'
		   AND tc.table_schema    = $1
		   AND tc.table_name      = $2
		 ORDER BY tc.constraint_name, kcu.ordinal_position`, schemaName, table)
	if err != nil {
		return nil, fmt.Errorf("foreign keys: %w", err)
	}
	defer rows.Close()

	// Group by constraint name.
	fkMap := make(map[string]*schema.ForeignKey)
	var fkOrder []string
	for rows.Next() {
		var cname, col, refTable, refCol string
		if err := rows.Scan(&cname, &col, &refTable, &refCol); err != nil {
			return nil, fmt.Errorf("foreign keys scan: %w", err)
		}
		fk, ok := fkMap[cname]
		if !ok {
			fk = &schema.ForeignKey{Name: cname, RefTable: refTable}
			fkMap[cname] = fk
			fkOrder = append(fkOrder, cname)
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
// Query Execution
// ---------------------------------------------------------------------------

func (c *pgConn) Execute(ctx context.Context, query string) (*adapter.QueryResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	c.setCancel(cancel)
	defer c.clearCancel()

	start := time.Now()
	isSelect := isSelectQuery(query)

	if isSelect {
		return c.executeSelect(ctx, query, start)
	}
	return c.executeNonSelect(ctx, query, start)
}

func (c *pgConn) executeSelect(ctx context.Context, query string, start time.Time) (*adapter.QueryResult, error) {
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	cols := fieldDescToMeta(rows.FieldDescriptions())

	var result [][]string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("execute values: %w", err)
		}
		result = append(result, valuesToStrings(vals))
	}
	if err := rows.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("execute rows: %w", err)
	}

	return &adapter.QueryResult{
		Columns:  cols,
		Rows:     result,
		RowCount: int64(len(result)),
		Duration: time.Since(start),
		IsSelect: true,
	}, nil
}

func (c *pgConn) executeNonSelect(ctx context.Context, query string, start time.Time) (*adapter.QueryResult, error) {
	tag, err := c.pool.Exec(ctx, query)
	if err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &adapter.QueryResult{
		RowCount: tag.RowsAffected(),
		Duration: time.Since(start),
		IsSelect: false,
		Message:  tag.String(),
	}, nil
}

// ---------------------------------------------------------------------------
// Streaming with server-side cursors
// ---------------------------------------------------------------------------

func (c *pgConn) ExecuteStreaming(ctx context.Context, query string, pageSize int) (adapter.RowIterator, error) {
	ctx, cancel := context.WithCancel(ctx)
	c.setCancel(cancel)

	// Open a direct connection (not from the pool) for the cursor transaction.
	conn, err := pgx.Connect(ctx, c.dsn)
	if err != nil {
		cancel()
		c.clearCancel()
		return nil, fmt.Errorf("streaming connect: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Close(ctx)
		cancel()
		c.clearCancel()
		return nil, fmt.Errorf("streaming begin tx: %w", err)
	}

	cursorName := "gotermsql_cursor"
	_, err = tx.Exec(ctx, fmt.Sprintf("DECLARE %s SCROLL CURSOR FOR %s", cursorName, query))
	if err != nil {
		tx.Rollback(ctx)
		conn.Close(ctx)
		cancel()
		c.clearCancel()
		return nil, fmt.Errorf("declare cursor: %w", err)
	}

	// Fetch the first batch to obtain column metadata.
	rows, err := tx.Query(ctx, fmt.Sprintf("FETCH FORWARD %d FROM %s", pageSize, cursorName))
	if err != nil {
		tx.Rollback(ctx)
		conn.Close(ctx)
		cancel()
		c.clearCancel()
		return nil, fmt.Errorf("initial fetch: %w", err)
	}

	cols := fieldDescToMeta(rows.FieldDescriptions())

	var firstBatch [][]string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			rows.Close()
			tx.Rollback(ctx)
			conn.Close(ctx)
			cancel()
			c.clearCancel()
			return nil, fmt.Errorf("initial fetch values: %w", err)
		}
		firstBatch = append(firstBatch, valuesToStrings(vals))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		tx.Rollback(ctx)
		conn.Close(ctx)
		cancel()
		c.clearCancel()
		return nil, fmt.Errorf("initial fetch rows: %w", err)
	}

	iter := &pgRowIterator{
		conn:       conn,
		tx:         tx,
		cursorName: cursorName,
		pageSize:   pageSize,
		cols:       cols,
		ctx:        ctx,
		cancel:     cancel,
		parentConn: c,
		firstBatch: firstBatch,
	}

	return iter, nil
}

// pgRowIterator implements adapter.RowIterator using server-side cursors.
type pgRowIterator struct {
	conn       *pgx.Conn
	tx         pgx.Tx
	cursorName string
	pageSize   int
	cols       []adapter.ColumnMeta
	ctx        context.Context
	cancel     context.CancelFunc
	parentConn *pgConn
	closed     atomic.Bool

	// firstBatch holds data from the initial FETCH during construction.
	// It is returned on the first call to FetchNext and then set to nil.
	firstBatch [][]string
}

func (it *pgRowIterator) Columns() []adapter.ColumnMeta {
	return it.cols
}

func (it *pgRowIterator) TotalRows() int64 {
	return -1 // unknown for streaming
}

func (it *pgRowIterator) FetchNext(ctx context.Context) ([][]string, error) {
	if it.closed.Load() {
		return nil, io.EOF
	}

	// Return the first batch if available.
	if it.firstBatch != nil {
		batch := it.firstBatch
		it.firstBatch = nil
		if len(batch) == 0 {
			return nil, io.EOF
		}
		return batch, nil
	}

	return it.fetch(ctx, fmt.Sprintf("FETCH FORWARD %d FROM %s", it.pageSize, it.cursorName))
}

func (it *pgRowIterator) FetchPrev(ctx context.Context) ([][]string, error) {
	if it.closed.Load() {
		return nil, io.EOF
	}

	return it.fetch(ctx, fmt.Sprintf("FETCH BACKWARD %d FROM %s", it.pageSize, it.cursorName))
}

func (it *pgRowIterator) fetch(ctx context.Context, sql string) ([][]string, error) {
	rows, err := it.tx.Query(ctx, sql)
	if err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("cursor fetch: %w", err)
	}
	defer rows.Close()

	var batch [][]string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("cursor fetch values: %w", err)
		}
		batch = append(batch, valuesToStrings(vals))
	}
	if err := rows.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, adapter.ErrCancelled
		}
		return nil, fmt.Errorf("cursor fetch rows: %w", err)
	}

	if len(batch) == 0 {
		return nil, io.EOF
	}
	return batch, nil
}

func (it *pgRowIterator) Close() error {
	if !it.closed.CompareAndSwap(false, true) {
		return nil // already closed
	}

	// Close cursor, rollback transaction, close connection.
	ctx := context.Background()

	it.tx.Exec(ctx, fmt.Sprintf("CLOSE %s", it.cursorName))
	it.tx.Rollback(ctx)
	err := it.conn.Close(ctx)

	it.cancel()
	it.parentConn.clearCancel()
	return err
}

// ---------------------------------------------------------------------------
// Completions
// ---------------------------------------------------------------------------

func (c *pgConn) Completions(ctx context.Context) ([]adapter.CompletionItem, error) {
	var items []adapter.CompletionItem

	// Schemas
	schemaRows, err := c.pool.Query(ctx,
		`SELECT schema_name
		 FROM information_schema.schemata
		 WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		 ORDER BY schema_name`)
	if err != nil {
		return nil, fmt.Errorf("completions schemas: %w", err)
	}
	defer schemaRows.Close()

	var schemas []string
	for schemaRows.Next() {
		var name string
		if err := schemaRows.Scan(&name); err != nil {
			return nil, fmt.Errorf("completions schemas scan: %w", err)
		}
		schemas = append(schemas, name)
		items = append(items, adapter.CompletionItem{
			Label:  name,
			Kind:   adapter.CompletionSchema,
			Detail: "schema",
		})
	}
	if err := schemaRows.Err(); err != nil {
		return nil, err
	}

	// Tables and views per schema
	for _, s := range schemas {
		tableRows, err := c.pool.Query(ctx,
			`SELECT table_name, table_type
			 FROM information_schema.tables
			 WHERE table_schema = $1
			 ORDER BY table_name`, s)
		if err != nil {
			return nil, fmt.Errorf("completions tables: %w", err)
		}

		type tableInfo struct {
			name      string
			tableType string
		}
		var tables []tableInfo

		for tableRows.Next() {
			var name, ttype string
			if err := tableRows.Scan(&name, &ttype); err != nil {
				tableRows.Close()
				return nil, fmt.Errorf("completions tables scan: %w", err)
			}
			tables = append(tables, tableInfo{name, ttype})

			kind := adapter.CompletionTable
			detail := "table"
			if ttype == "VIEW" {
				kind = adapter.CompletionView
				detail = "view"
			}

			label := name
			if s != "public" {
				label = s + "." + name
			}
			items = append(items, adapter.CompletionItem{
				Label:  label,
				Kind:   kind,
				Detail: detail,
			})
		}
		tableRows.Close()
		if err := tableRows.Err(); err != nil {
			return nil, err
		}

		// Columns for each table/view
		for _, t := range tables {
			colRows, err := c.pool.Query(ctx,
				`SELECT column_name, data_type
				 FROM information_schema.columns
				 WHERE table_schema = $1 AND table_name = $2
				 ORDER BY ordinal_position`, s, t.name)
			if err != nil {
				return nil, fmt.Errorf("completions columns: %w", err)
			}
			for colRows.Next() {
				var cname, ctype string
				if err := colRows.Scan(&cname, &ctype); err != nil {
					colRows.Close()
					return nil, fmt.Errorf("completions columns scan: %w", err)
				}
				items = append(items, adapter.CompletionItem{
					Label:  cname,
					Kind:   adapter.CompletionColumn,
					Detail: fmt.Sprintf("%s.%s (%s)", t.name, cname, ctype),
				})
			}
			colRows.Close()
			if err := colRows.Err(); err != nil {
				return nil, err
			}
		}
	}

	return items, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isSelectQuery determines if a query is a SELECT-like statement.
func isSelectQuery(query string) bool {
	q := strings.TrimSpace(query)
	// Strip leading comments (-- and /* */)
	for {
		if strings.HasPrefix(q, "--") {
			if idx := strings.Index(q, "\n"); idx >= 0 {
				q = strings.TrimSpace(q[idx+1:])
				continue
			}
			return false
		}
		if strings.HasPrefix(q, "/*") {
			if idx := strings.Index(q, "*/"); idx >= 0 {
				q = strings.TrimSpace(q[idx+2:])
				continue
			}
			return false
		}
		break
	}
	upper := strings.ToUpper(q)
	return strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "VALUES") ||
		strings.HasPrefix(upper, "TABLE") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "EXPLAIN")
}

// fieldDescToMeta converts pgx field descriptions to adapter ColumnMeta.
func fieldDescToMeta(fds []pgconn.FieldDescription) []adapter.ColumnMeta {
	cols := make([]adapter.ColumnMeta, len(fds))
	for i, fd := range fds {
		cols[i] = adapter.ColumnMeta{
			Name: fd.Name,
			Type: pgTypeOIDToName(fd.DataTypeOID),
		}
	}
	return cols
}

// valuesToStrings converts a row of interface{} values to strings.
func valuesToStrings(vals []any) []string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = valueToString(v)
	}
	return out
}

// valueToString converts a single database value to a string representation.
func valueToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case time.Time:
		if val.Hour() == 0 && val.Minute() == 0 && val.Second() == 0 && val.Nanosecond() == 0 {
			return val.Format("2006-01-02")
		}
		return val.Format("2006-01-02 15:04:05")
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int8:
		return fmt.Sprintf("%d", val)
	case int16:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float32:
		return fmt.Sprintf("%g", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case [16]byte:
		// UUID
		return fmt.Sprintf("%x-%x-%x-%x-%x", val[0:4], val[4:6], val[6:8], val[8:10], val[10:16])
	case []string:
		return "{" + strings.Join(val, ",") + "}"
	case []int32:
		parts := make([]string, len(val))
		for i, n := range val {
			parts[i] = fmt.Sprintf("%d", n)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []int64:
		parts := make([]string, len(val))
		for i, n := range val {
			parts[i] = fmt.Sprintf("%d", n)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []float64:
		parts := make([]string, len(val))
		for i, n := range val {
			parts[i] = fmt.Sprintf("%g", n)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []bool:
		parts := make([]string, len(val))
		for i, b := range val {
			if b {
				parts[i] = "true"
			} else {
				parts[i] = "false"
			}
		}
		return "{" + strings.Join(parts, ",") + "}"
	case pgtype.Numeric:
		dv, err := val.Value()
		if err != nil || dv == nil {
			return ""
		}
		if s, ok := dv.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", dv)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// pgTypeOIDToName maps common PostgreSQL type OIDs to human-readable names.
func pgTypeOIDToName(oid uint32) string {
	switch oid {
	case 16:
		return "bool"
	case 17:
		return "bytea"
	case 18:
		return "char"
	case 20:
		return "int8"
	case 21:
		return "int2"
	case 23:
		return "int4"
	case 25:
		return "text"
	case 26:
		return "oid"
	case 114:
		return "json"
	case 142:
		return "xml"
	case 700:
		return "float4"
	case 701:
		return "float8"
	case 790:
		return "money"
	case 1000:
		return "bool[]"
	case 1005:
		return "int2[]"
	case 1007:
		return "int4[]"
	case 1009:
		return "text[]"
	case 1016:
		return "int8[]"
	case 1021:
		return "float4[]"
	case 1022:
		return "float8[]"
	case 1042:
		return "bpchar"
	case 1043:
		return "varchar"
	case 1082:
		return "date"
	case 1083:
		return "time"
	case 1114:
		return "timestamp"
	case 1184:
		return "timestamptz"
	case 1186:
		return "interval"
	case 1266:
		return "timetz"
	case 1700:
		return "numeric"
	case 2249:
		return "record"
	case 2278:
		return "void"
	case 2950:
		return "uuid"
	case 3802:
		return "jsonb"
	case 3807:
		return "jsonb[]"
	default:
		return fmt.Sprintf("oid:%d", oid)
	}
}
