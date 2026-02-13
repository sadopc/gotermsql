//go:build !duckdb

package duckdb

import (
	"context"
	"errors"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

var errDisabled = errors.New("DuckDB support not compiled in. Rebuild with -tags duckdb")

func init() {
	adapter.Register(&disabledAdapter{})
}

type disabledAdapter struct{}

func (d *disabledAdapter) Name() string     { return "duckdb" }
func (d *disabledAdapter) DefaultPort() int { return 0 }

func (d *disabledAdapter) Connect(_ context.Context, _ string) (adapter.Connection, error) {
	return nil, errDisabled
}

// disabledConnection is never instantiated but satisfies the interface at compile time.
var _ adapter.Connection = (*disabledConnection)(nil)

type disabledConnection struct{}

func (c *disabledConnection) Databases(_ context.Context) ([]schema.Database, error) {
	return nil, errDisabled
}
func (c *disabledConnection) Tables(_ context.Context, _, _ string) ([]schema.Table, error) {
	return nil, errDisabled
}
func (c *disabledConnection) Columns(_ context.Context, _, _, _ string) ([]schema.Column, error) {
	return nil, errDisabled
}
func (c *disabledConnection) Indexes(_ context.Context, _, _, _ string) ([]schema.Index, error) {
	return nil, errDisabled
}
func (c *disabledConnection) ForeignKeys(_ context.Context, _, _, _ string) ([]schema.ForeignKey, error) {
	return nil, errDisabled
}
func (c *disabledConnection) Execute(_ context.Context, _ string) (*adapter.QueryResult, error) {
	return nil, errDisabled
}
func (c *disabledConnection) ExecuteStreaming(_ context.Context, _ string, _ int) (adapter.RowIterator, error) {
	return nil, errDisabled
}
func (c *disabledConnection) Cancel() error { return errDisabled }
func (c *disabledConnection) Completions(_ context.Context) ([]adapter.CompletionItem, error) {
	return nil, errDisabled
}
func (c *disabledConnection) Ping(_ context.Context) error { return errDisabled }
func (c *disabledConnection) Close() error                 { return errDisabled }
func (c *disabledConnection) DatabaseName() string         { return "" }
func (c *disabledConnection) AdapterName() string          { return "duckdb" }
