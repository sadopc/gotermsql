//go:build !duckdb

package duckdb

import (
	"context"
	"strings"
	"testing"

	"github.com/sadopc/gotermsql/internal/adapter"
)

func TestDuckDBDisabled_Name(t *testing.T) {
	a := &disabledAdapter{}
	if got := a.Name(); got != "duckdb" {
		t.Errorf("Name() = %q, want %q", got, "duckdb")
	}
}

func TestDuckDBDisabled_DefaultPort(t *testing.T) {
	a := &disabledAdapter{}
	if got := a.DefaultPort(); got != 0 {
		t.Errorf("DefaultPort() = %d, want %d", got, 0)
	}
}

func TestDuckDBDisabled_Connect(t *testing.T) {
	a := &disabledAdapter{}
	conn, err := a.Connect(context.Background(), "test.db")

	if conn != nil {
		t.Error("Connect() should return nil connection when disabled")
	}
	if err == nil {
		t.Fatal("Connect() should return an error when disabled")
	}
	if !strings.Contains(err.Error(), "not compiled in") {
		t.Errorf("Connect() error = %q, expected to contain 'not compiled in'", err.Error())
	}
	if err != errDisabled {
		t.Errorf("Connect() error should be errDisabled, got %v", err)
	}
}

func TestDuckDBDisabled_Registration(t *testing.T) {
	a, ok := adapter.Registry["duckdb"]
	if !ok {
		t.Fatal("duckdb adapter not found in registry")
	}
	if a.Name() != "duckdb" {
		t.Errorf("registered adapter Name() = %q, want %q", a.Name(), "duckdb")
	}
	if a.DefaultPort() != 0 {
		t.Errorf("registered adapter DefaultPort() = %d, want %d", a.DefaultPort(), 0)
	}
}

func TestDuckDBDisabled_ErrorMessage(t *testing.T) {
	// Verify the error message is descriptive and mentions the build tag.
	msg := errDisabled.Error()
	if !strings.Contains(msg, "DuckDB") {
		t.Errorf("errDisabled message = %q, expected to contain 'DuckDB'", msg)
	}
	if !strings.Contains(msg, "duckdb") {
		t.Errorf("errDisabled message = %q, expected to contain 'duckdb' (build tag)", msg)
	}
}

func TestDuckDBDisabled_ConnectionInterface(t *testing.T) {
	// Verify that disabledConnection satisfies adapter.Connection at compile time.
	// This is already checked by the var _ line in the source, but we can
	// also verify the methods return errDisabled consistently.
	c := &disabledConnection{}

	ctx := context.Background()

	if _, err := c.Databases(ctx); err != errDisabled {
		t.Errorf("Databases() error = %v, want errDisabled", err)
	}
	if _, err := c.Tables(ctx, "", ""); err != errDisabled {
		t.Errorf("Tables() error = %v, want errDisabled", err)
	}
	if _, err := c.Columns(ctx, "", "", ""); err != errDisabled {
		t.Errorf("Columns() error = %v, want errDisabled", err)
	}
	if _, err := c.Indexes(ctx, "", "", ""); err != errDisabled {
		t.Errorf("Indexes() error = %v, want errDisabled", err)
	}
	if _, err := c.ForeignKeys(ctx, "", "", ""); err != errDisabled {
		t.Errorf("ForeignKeys() error = %v, want errDisabled", err)
	}
	if _, err := c.Execute(ctx, ""); err != errDisabled {
		t.Errorf("Execute() error = %v, want errDisabled", err)
	}
	if _, err := c.ExecuteStreaming(ctx, "", 0); err != errDisabled {
		t.Errorf("ExecuteStreaming() error = %v, want errDisabled", err)
	}
	if err := c.Cancel(); err != errDisabled {
		t.Errorf("Cancel() error = %v, want errDisabled", err)
	}
	if _, err := c.Completions(ctx); err != errDisabled {
		t.Errorf("Completions() error = %v, want errDisabled", err)
	}
	if err := c.Ping(ctx); err != errDisabled {
		t.Errorf("Ping() error = %v, want errDisabled", err)
	}
	if err := c.Close(); err != errDisabled {
		t.Errorf("Close() error = %v, want errDisabled", err)
	}
	if got := c.DatabaseName(); got != "" {
		t.Errorf("DatabaseName() = %q, want empty string", got)
	}
	if got := c.AdapterName(); got != "duckdb" {
		t.Errorf("AdapterName() = %q, want %q", got, "duckdb")
	}
}
