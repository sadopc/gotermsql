package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

func init() {
	adapter.Register(&mysqlAdapter{})
}

// ---------------------------------------------------------------------------
// Adapter
// ---------------------------------------------------------------------------

type mysqlAdapter struct{}

func (a *mysqlAdapter) Name() string     { return "mysql" }
func (a *mysqlAdapter) DefaultPort() int { return 3306 }

func (a *mysqlAdapter) Connect(ctx context.Context, dsn string) (adapter.Connection, error) {
	goDriverDSN, dbName, err := normalizeDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: invalid dsn: %w", err)
	}

	db, err := sql.Open("mysql", goDriverDSN)
	if err != nil {
		return nil, fmt.Errorf("mysql: open: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("mysql: ping: %w", err)
	}

	return &mysqlConn{
		db:     db,
		dsn:    goDriverDSN,
		dbName: dbName,
	}, nil
}

// normalizeDSN converts a mysql:// URL-style DSN to go-sql-driver format, or
// passes through a DSN that is already in go-sql-driver format.
//
// Accepted forms:
//   - mysql://user:pass@host:port/dbname?params
//   - user:pass@tcp(host:port)/dbname?params
func normalizeDSN(dsn string) (goDriverDSN string, dbName string, err error) {
	if strings.HasPrefix(dsn, "mysql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", "", err
		}

		user := u.User.Username()
		pass, _ := u.User.Password()

		host := u.Hostname()
		port := u.Port()
		if port == "" {
			port = "3306"
		}

		dbName = strings.TrimPrefix(u.Path, "/")

		var userInfo string
		if pass != "" {
			userInfo = fmt.Sprintf("%s:%s", user, pass)
		} else if user != "" {
			userInfo = user
		}

		query := u.RawQuery
		// Ensure parseTime=true so time columns scan correctly.
		if query == "" {
			query = "parseTime=true"
		} else if !strings.Contains(query, "parseTime") {
			query += "&parseTime=true"
		}

		goDriverDSN = fmt.Sprintf("%s@tcp(%s:%s)/%s?%s", userInfo, host, port, dbName, query)
		return goDriverDSN, dbName, nil
	}

	// Already in go-sql-driver format. Extract dbName from the DSN.
	// Format: [user[:pass]@][tcp[(host:port)]]/dbname[?params]
	if !strings.Contains(dsn, "parseTime") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}

	// Extract database name: everything between the last "/" and "?" (or end).
	if idx := strings.LastIndex(dsn, "/"); idx >= 0 {
		rest := dsn[idx+1:]
		if qIdx := strings.Index(rest, "?"); qIdx >= 0 {
			dbName = rest[:qIdx]
		} else {
			dbName = rest
		}
	}

	return dsn, dbName, nil
}

// ---------------------------------------------------------------------------
// Connection
// ---------------------------------------------------------------------------

type mysqlConn struct {
	db     *sql.DB
	dsn    string
	dbName string

	mu           sync.Mutex
	cancel       context.CancelFunc
	activeConnID int64
}

func (c *mysqlConn) AdapterName() string  { return "mysql" }
func (c *mysqlConn) DatabaseName() string { return c.dbName }

func (c *mysqlConn) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *mysqlConn) Close() error {
	return c.db.Close()
}

// ---------------------------------------------------------------------------
// Introspection
// ---------------------------------------------------------------------------

func (c *mysqlConn) Databases(ctx context.Context) ([]schema.Database, error) {
	rows, err := c.db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []schema.Database
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		dbs = append(dbs, schema.Database{Name: name})
	}
	return dbs, rows.Err()
}

func (c *mysqlConn) Tables(ctx context.Context, db, schemaName string) ([]schema.Table, error) {
	if db == "" {
		db = c.dbName
	}

	const q = `
		SELECT TABLE_NAME
		FROM information_schema.tables
		WHERE TABLE_SCHEMA = ?
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`

	rows, err := c.db.QueryContext(ctx, q, db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, schema.Table{Name: name})
	}
	return tables, rows.Err()
}

func (c *mysqlConn) Columns(ctx context.Context, db, schemaName, table string) ([]schema.Column, error) {
	if db == "" {
		db = c.dbName
	}

	const q = `
		SELECT
			c.COLUMN_NAME,
			c.COLUMN_TYPE,
			c.IS_NULLABLE,
			COALESCE(c.COLUMN_DEFAULT, ''),
			CASE WHEN kcu.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON  kcu.TABLE_SCHEMA    = c.TABLE_SCHEMA
			AND kcu.TABLE_NAME      = c.TABLE_NAME
			AND kcu.COLUMN_NAME     = c.COLUMN_NAME
			AND kcu.CONSTRAINT_NAME = 'PRIMARY'
		WHERE c.TABLE_SCHEMA = ?
		  AND c.TABLE_NAME   = ?
		ORDER BY c.ORDINAL_POSITION`

	rows, err := c.db.QueryContext(ctx, q, db, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []schema.Column
	for rows.Next() {
		var (
			col      schema.Column
			nullable string
			isPKInt  int
		)
		if err := rows.Scan(&col.Name, &col.Type, &nullable, &col.Default, &isPKInt); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		col.IsPK = isPKInt == 1
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (c *mysqlConn) Indexes(ctx context.Context, db, schemaName, table string) ([]schema.Index, error) {
	if db == "" {
		db = c.dbName
	}

	const q = `
		SELECT
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE
		FROM information_schema.statistics
		WHERE TABLE_SCHEMA = ?
		  AND TABLE_NAME   = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`

	rows, err := c.db.QueryContext(ctx, q, db, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*schema.Index)
	var order []string

	for rows.Next() {
		var (
			idxName   string
			colName   string
			nonUnique int
		)
		if err := rows.Scan(&idxName, &colName, &nonUnique); err != nil {
			return nil, err
		}
		idx, ok := indexMap[idxName]
		if !ok {
			idx = &schema.Index{
				Name:   idxName,
				Unique: nonUnique == 0,
			}
			indexMap[idxName] = idx
			order = append(order, idxName)
		}
		idx.Columns = append(idx.Columns, colName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	indexes := make([]schema.Index, 0, len(order))
	for _, name := range order {
		indexes = append(indexes, *indexMap[name])
	}
	return indexes, nil
}

func (c *mysqlConn) ForeignKeys(ctx context.Context, db, schemaName, table string) ([]schema.ForeignKey, error) {
	if db == "" {
		db = c.dbName
	}

	const q = `
		SELECT
			kcu.CONSTRAINT_NAME,
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
			ON  rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
			AND rc.CONSTRAINT_NAME   = kcu.CONSTRAINT_NAME
		WHERE kcu.TABLE_SCHEMA          = ?
		  AND kcu.TABLE_NAME            = ?
		  AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`

	rows, err := c.db.QueryContext(ctx, q, db, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkMap := make(map[string]*schema.ForeignKey)
	var order []string

	for rows.Next() {
		var (
			fkName   string
			colName  string
			refTable string
			refCol   string
		)
		if err := rows.Scan(&fkName, &colName, &refTable, &refCol); err != nil {
			return nil, err
		}
		fk, ok := fkMap[fkName]
		if !ok {
			fk = &schema.ForeignKey{
				Name:     fkName,
				RefTable: refTable,
			}
			fkMap[fkName] = fk
			order = append(order, fkName)
		}
		fk.Columns = append(fk.Columns, colName)
		fk.RefColumns = append(fk.RefColumns, refCol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	fks := make([]schema.ForeignKey, 0, len(order))
	for _, name := range order {
		fks = append(fks, *fkMap[name])
	}
	return fks, nil
}

// ---------------------------------------------------------------------------
// Batch Introspection (implements adapter.BatchIntrospector)
// ---------------------------------------------------------------------------

func (c *mysqlConn) AllColumns(ctx context.Context, db, schemaName string) (map[string][]schema.Column, error) {
	if db == "" {
		db = c.dbName
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT c.TABLE_NAME, c.COLUMN_NAME, c.COLUMN_TYPE, c.IS_NULLABLE,
		       COALESCE(c.COLUMN_DEFAULT, ''),
		       CASE WHEN kcu.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON  kcu.TABLE_SCHEMA    = c.TABLE_SCHEMA
			AND kcu.TABLE_NAME      = c.TABLE_NAME
			AND kcu.COLUMN_NAME     = c.COLUMN_NAME
			AND kcu.CONSTRAINT_NAME = 'PRIMARY'
		WHERE c.TABLE_SCHEMA = ?
		ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION`, db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]schema.Column)
	for rows.Next() {
		var (
			table    string
			col      schema.Column
			nullable string
			isPKInt  int
		)
		if err := rows.Scan(&table, &col.Name, &col.Type, &nullable, &col.Default, &isPKInt); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		col.IsPK = isPKInt == 1
		result[table] = append(result[table], col)
	}
	return result, rows.Err()
}

func (c *mysqlConn) AllIndexes(ctx context.Context, db, schemaName string) (map[string][]schema.Index, error) {
	if db == "" {
		db = c.dbName
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT TABLE_NAME, INDEX_NAME, COLUMN_NAME, NON_UNIQUE
		FROM information_schema.statistics
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX`, db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type idxKey struct{ table, name string }
	indexMap := make(map[idxKey]*schema.Index)
	var order []idxKey

	for rows.Next() {
		var table, idxName, colName string
		var nonUnique int
		if err := rows.Scan(&table, &idxName, &colName, &nonUnique); err != nil {
			return nil, err
		}
		key := idxKey{table, idxName}
		idx, ok := indexMap[key]
		if !ok {
			idx = &schema.Index{Name: idxName, Unique: nonUnique == 0}
			indexMap[key] = idx
			order = append(order, key)
		}
		idx.Columns = append(idx.Columns, colName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[string][]schema.Index)
	for _, key := range order {
		result[key.table] = append(result[key.table], *indexMap[key])
	}
	return result, nil
}

func (c *mysqlConn) AllForeignKeys(ctx context.Context, db, schemaName string) (map[string][]schema.ForeignKey, error) {
	if db == "" {
		db = c.dbName
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT kcu.TABLE_NAME, kcu.CONSTRAINT_NAME, kcu.COLUMN_NAME,
		       kcu.REFERENCED_TABLE_NAME, kcu.REFERENCED_COLUMN_NAME
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.referential_constraints rc
			ON  rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
			AND rc.CONSTRAINT_NAME   = kcu.CONSTRAINT_NAME
		WHERE kcu.TABLE_SCHEMA          = ?
		  AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY kcu.TABLE_NAME, kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`, db)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type fkKey struct{ table, name string }
	fkMap := make(map[fkKey]*schema.ForeignKey)
	var fkOrder []fkKey

	for rows.Next() {
		var table, fkName, colName, refTable, refCol string
		if err := rows.Scan(&table, &fkName, &colName, &refTable, &refCol); err != nil {
			return nil, err
		}
		key := fkKey{table, fkName}
		fk, ok := fkMap[key]
		if !ok {
			fk = &schema.ForeignKey{Name: fkName, RefTable: refTable}
			fkMap[key] = fk
			fkOrder = append(fkOrder, key)
		}
		fk.Columns = append(fk.Columns, colName)
		fk.RefColumns = append(fk.RefColumns, refCol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[string][]schema.ForeignKey)
	for _, key := range fkOrder {
		result[key.table] = append(result[key.table], *fkMap[key])
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

func (c *mysqlConn) Execute(ctx context.Context, query string) (*adapter.QueryResult, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Pin to a dedicated connection from the pool so that CONNECTION_ID()
	// accurately identifies the session running our query.
	sqlConn, err := c.db.Conn(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("mysql: acquire conn: %w", err)
	}

	var connID int64
	if err := sqlConn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&connID); err != nil {
		sqlConn.Close()
		cancel()
		return nil, fmt.Errorf("mysql: connection_id: %w", err)
	}

	c.mu.Lock()
	c.cancel = cancel
	c.activeConnID = connID
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.cancel = nil
		c.activeConnID = 0
		c.mu.Unlock()
		sqlConn.Close()
		cancel()
	}()

	start := time.Now()

	if isSelectQuery(query) {
		return c.executeSelectOnConn(ctx, sqlConn, query, start)
	}
	return c.executeExecOnConn(ctx, sqlConn, query, start)
}

func (c *mysqlConn) executeSelectOnConn(ctx context.Context, conn *sql.Conn, query string, start time.Time) (*adapter.QueryResult, error) {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	columns := make([]adapter.ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		columns[i].Name = ct.Name()
		columns[i].Type = ct.DatabaseTypeName()
		if n, ok := ct.Nullable(); ok {
			columns[i].Nullable = n
		}
	}

	var resultRows [][]string
	nCols := len(columns)
	truncated := false

	for rows.Next() {
		if len(resultRows) >= adapter.DefaultMaxRows {
			truncated = true
			break
		}
		values := make([]sql.NullString, nCols)
		ptrs := make([]any, nCols)
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, nCols)
		for i, v := range values {
			if v.Valid {
				row[i] = v.String
			} else {
				row[i] = "NULL"
			}
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &adapter.QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  int64(len(resultRows)),
		Duration:  time.Since(start),
		IsSelect:  true,
		Truncated: truncated,
	}, nil
}

func (c *mysqlConn) executeExecOnConn(ctx context.Context, conn *sql.Conn, query string, start time.Time) (*adapter.QueryResult, error) {
	result, err := conn.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}

	affected, _ := result.RowsAffected()

	return &adapter.QueryResult{
		RowCount: affected,
		Duration: time.Since(start),
		IsSelect: false,
		Message:  fmt.Sprintf("%d row(s) affected", affected),
	}, nil
}

// Cancel kills the currently running query via KILL QUERY on a separate
// connection.
func (c *mysqlConn) Cancel() error {
	c.mu.Lock()
	cancel := c.cancel
	connID := c.activeConnID
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if connID == 0 {
		return nil // no active query
	}

	// Open a short-lived connection to issue KILL QUERY.
	killDB, err := sql.Open("mysql", c.dsn)
	if err != nil {
		return fmt.Errorf("mysql: cancel open: %w", err)
	}
	defer killDB.Close()

	ctx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer killCancel()

	_, err = killDB.ExecContext(ctx, fmt.Sprintf("KILL QUERY %d", connID))
	if err != nil {
		return fmt.Errorf("mysql: kill query: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Streaming (LIMIT/OFFSET pagination)
// ---------------------------------------------------------------------------

func (c *mysqlConn) ExecuteStreaming(ctx context.Context, query string, pageSize int) (adapter.RowIterator, error) {
	// Probe columns by running the query with LIMIT 0.
	probeQuery := fmt.Sprintf("SELECT * FROM (%s) AS _t LIMIT 0", strings.TrimRight(query, "; \t\n"))

	rows, err := c.db.QueryContext(ctx, probeQuery)
	if err != nil {
		return nil, err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	columns := make([]adapter.ColumnMeta, len(colTypes))
	for i, ct := range colTypes {
		columns[i].Name = ct.Name()
		columns[i].Type = ct.DatabaseTypeName()
		if n, ok := ct.Nullable(); ok {
			columns[i].Nullable = n
		}
	}

	return &rowIterator{
		conn:      c,
		baseQuery: strings.TrimRight(query, "; \t\n"),
		pageSize:  pageSize,
		columns:   columns,
		offset:    0,
	}, nil
}

type rowIterator struct {
	conn      *mysqlConn
	baseQuery string
	pageSize  int
	columns   []adapter.ColumnMeta
	offset    int64
}

func (it *rowIterator) Columns() []adapter.ColumnMeta { return it.columns }
func (it *rowIterator) TotalRows() int64              { return -1 }
func (it *rowIterator) Close() error                  { return nil }

func (it *rowIterator) FetchNext(ctx context.Context) ([][]string, error) {
	q := fmt.Sprintf("SELECT * FROM (%s) AS _t LIMIT %d OFFSET %d",
		it.baseQuery, it.pageSize, it.offset)

	rows, err := it.conn.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	page, err := scanPage(rows, len(it.columns))
	if err != nil {
		return nil, err
	}

	if len(page) == 0 {
		return nil, io.EOF
	}

	it.offset += int64(len(page))
	return page, nil
}

func (it *rowIterator) FetchPrev(ctx context.Context) ([][]string, error) {
	newOffset := it.offset - int64(it.pageSize)*2
	if newOffset < 0 {
		// If we haven't scrolled enough pages to go back, clamp or error.
		if it.offset-int64(it.pageSize) <= 0 {
			return nil, adapter.ErrNoBidirectional
		}
		newOffset = 0
	}

	it.offset = newOffset

	q := fmt.Sprintf("SELECT * FROM (%s) AS _t LIMIT %d OFFSET %d",
		it.baseQuery, it.pageSize, it.offset)

	rows, err := it.conn.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	page, err := scanPage(rows, len(it.columns))
	if err != nil {
		return nil, err
	}

	if len(page) == 0 {
		return nil, io.EOF
	}

	it.offset += int64(len(page))
	return page, nil
}

func scanPage(rows *sql.Rows, nCols int) ([][]string, error) {
	var page [][]string
	for rows.Next() {
		values := make([]sql.NullString, nCols)
		ptrs := make([]any, nCols)
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, nCols)
		for i, v := range values {
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

func (c *mysqlConn) Completions(ctx context.Context) ([]adapter.CompletionItem, error) {
	var items []adapter.CompletionItem

	// Tables (and views) in the current database.
	tableRows, err := c.db.QueryContext(ctx, `
		SELECT TABLE_NAME, TABLE_TYPE
		FROM information_schema.tables
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME`, c.dbName)
	if err != nil {
		return nil, err
	}
	defer tableRows.Close()

	var tableNames []string
	for tableRows.Next() {
		var name, ttype string
		if err := tableRows.Scan(&name, &ttype); err != nil {
			return nil, err
		}
		kind := adapter.CompletionTable
		if ttype == "VIEW" {
			kind = adapter.CompletionView
		}
		items = append(items, adapter.CompletionItem{
			Label:  name,
			Kind:   kind,
			Detail: ttype,
		})
		tableNames = append(tableNames, name)
	}
	if err := tableRows.Err(); err != nil {
		return nil, err
	}

	// Columns for all tables in the current database.
	if len(tableNames) > 0 {
		colRows, err := c.db.QueryContext(ctx, `
			SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE
			FROM information_schema.columns
			WHERE TABLE_SCHEMA = ?
			ORDER BY TABLE_NAME, ORDINAL_POSITION`, c.dbName)
		if err != nil {
			return nil, err
		}
		defer colRows.Close()

		for colRows.Next() {
			var tableName, colName, colType string
			if err := colRows.Scan(&tableName, &colName, &colType); err != nil {
				return nil, err
			}
			items = append(items, adapter.CompletionItem{
				Label:  colName,
				Kind:   adapter.CompletionColumn,
				Detail: fmt.Sprintf("%s.%s (%s)", tableName, colName, colType),
			})
		}
		if err := colRows.Err(); err != nil {
			return nil, err
		}
	}

	return items, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isSelectQuery returns true if the trimmed, uppercased query starts with a
// keyword that produces a result set.
func isSelectQuery(query string) bool {
	q := strings.TrimSpace(query)
	upper := strings.ToUpper(q)
	for _, prefix := range []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}
