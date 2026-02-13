package adapter

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/sadopc/gotermsql/internal/schema"
)

var (
	ErrNoBidirectional = errors.New("adapter does not support bidirectional scrolling")
	ErrNotConnected    = errors.New("not connected to database")
	ErrCancelled       = errors.New("query cancelled")
)

// Adapter creates database connections.
type Adapter interface {
	Connect(ctx context.Context, dsn string) (Connection, error)
	Name() string
	DefaultPort() int
}

// Connection represents an active database connection.
type Connection interface {
	// Introspection
	Databases(ctx context.Context) ([]schema.Database, error)
	Tables(ctx context.Context, db, schemaName string) ([]schema.Table, error)
	Columns(ctx context.Context, db, schemaName, table string) ([]schema.Column, error)
	Indexes(ctx context.Context, db, schemaName, table string) ([]schema.Index, error)
	ForeignKeys(ctx context.Context, db, schemaName, table string) ([]schema.ForeignKey, error)

	// Query execution
	Execute(ctx context.Context, query string) (*QueryResult, error)
	Cancel() error

	// Streaming for large results
	ExecuteStreaming(ctx context.Context, query string, pageSize int) (RowIterator, error)

	// Completions
	Completions(ctx context.Context) ([]CompletionItem, error)

	// Lifecycle
	Ping(ctx context.Context) error
	Close() error

	// Info
	DatabaseName() string
	AdapterName() string
}

// RowIterator provides paginated access to query results.
type RowIterator interface {
	FetchNext(ctx context.Context) ([][]string, error)
	FetchPrev(ctx context.Context) ([][]string, error)
	Columns() []ColumnMeta
	TotalRows() int64 // -1 if unknown
	Close() error
}

// QueryResult holds the result of a query execution.
type QueryResult struct {
	Columns  []ColumnMeta
	Rows     [][]string
	RowCount int64 // -1 if unknown
	Duration time.Duration
	IsSelect bool
	Message  string
}

// ColumnMeta holds metadata about a result column.
type ColumnMeta struct {
	Name     string
	Type     string
	Nullable bool
}

// CompletionItem is a schema-aware autocomplete candidate.
type CompletionItem struct {
	Label  string
	Kind   CompletionKind
	Detail string
}

// CompletionKind categorizes autocomplete items.
type CompletionKind int

const (
	CompletionTable CompletionKind = iota
	CompletionColumn
	CompletionKeyword
	CompletionFunction
	CompletionSchema
	CompletionDatabase
	CompletionView
)

// SentinelEOF returns true if err is io.EOF.
func SentinelEOF(err error) bool {
	return errors.Is(err, io.EOF)
}

// Registry holds registered adapters by name.
var Registry = map[string]Adapter{}

// Register adds an adapter to the global registry.
func Register(a Adapter) {
	Registry[a.Name()] = a
}
