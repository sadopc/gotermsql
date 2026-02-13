package statusbar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sadopc/gotermsql/internal/adapter"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/schema"
	"github.com/sadopc/gotermsql/internal/theme"
)

func init() {
	theme.Current = theme.Default()
}

// mockConnection implements adapter.Connection for testing.
type mockConnection struct {
	dbName      string
	adapterName string
}

func (c *mockConnection) DatabaseName() string { return c.dbName }
func (c *mockConnection) AdapterName() string  { return c.adapterName }
func (c *mockConnection) Databases(_ context.Context) ([]schema.Database, error) {
	return nil, nil
}
func (c *mockConnection) Tables(_ context.Context, _, _ string) ([]schema.Table, error) {
	return nil, nil
}
func (c *mockConnection) Columns(_ context.Context, _, _, _ string) ([]schema.Column, error) {
	return nil, nil
}
func (c *mockConnection) Indexes(_ context.Context, _, _, _ string) ([]schema.Index, error) {
	return nil, nil
}
func (c *mockConnection) ForeignKeys(_ context.Context, _, _, _ string) ([]schema.ForeignKey, error) {
	return nil, nil
}
func (c *mockConnection) Execute(_ context.Context, _ string) (*adapter.QueryResult, error) {
	return nil, nil
}
func (c *mockConnection) Cancel() error { return nil }
func (c *mockConnection) ExecuteStreaming(_ context.Context, _ string, _ int) (adapter.RowIterator, error) {
	return nil, nil
}
func (c *mockConnection) Completions(_ context.Context) ([]adapter.CompletionItem, error) {
	return nil, nil
}
func (c *mockConnection) Ping(_ context.Context) error { return nil }
func (c *mockConnection) Close() error                 { return nil }

func TestNew(t *testing.T) {
	m := New()

	if m.rowCount != -1 {
		t.Fatalf("expected rowCount=-1, got %d", m.rowCount)
	}
	if m.keyMode != appmsg.KeyModeStandard {
		t.Fatalf("expected KeyModeStandard, got %v", m.keyMode)
	}
	if m.connected {
		t.Fatal("expected connected=false")
	}
	if m.message != "" {
		t.Fatalf("expected empty message, got %q", m.message)
	}
}

func TestUpdate_ConnectMsg(t *testing.T) {
	m := New()

	conn := &mockConnection{dbName: "testdb", adapterName: "postgres"}
	m, _ = m.Update(appmsg.ConnectMsg{
		Conn:    conn,
		Adapter: "postgres",
		DSN:     "postgres://localhost/testdb",
	})

	if !m.connected {
		t.Fatal("expected connected=true after ConnectMsg")
	}
	if m.adapterName != "postgres" {
		t.Fatalf("expected adapter 'postgres', got %q", m.adapterName)
	}
	if m.databaseName != "testdb" {
		t.Fatalf("expected database 'testdb', got %q", m.databaseName)
	}
	if m.dsn != "postgres://localhost/testdb" {
		t.Fatalf("expected DSN 'postgres://localhost/testdb', got %q", m.dsn)
	}
	if m.message != "" {
		t.Fatalf("expected message cleared, got %q", m.message)
	}
	if m.isError {
		t.Fatal("expected isError=false after connect")
	}
}

func TestUpdate_DisconnectMsg(t *testing.T) {
	m := New()

	// First connect.
	conn := &mockConnection{dbName: "testdb", adapterName: "sqlite"}
	m, _ = m.Update(appmsg.ConnectMsg{
		Conn:    conn,
		Adapter: "sqlite",
		DSN:     "file:test.db",
	})

	// Then disconnect.
	m, _ = m.Update(appmsg.DisconnectMsg{})

	if m.connected {
		t.Fatal("expected connected=false after DisconnectMsg")
	}
	if m.adapterName != "" {
		t.Fatalf("expected empty adapterName, got %q", m.adapterName)
	}
	if m.databaseName != "" {
		t.Fatalf("expected empty databaseName, got %q", m.databaseName)
	}
	if m.dsn != "" {
		t.Fatalf("expected empty DSN, got %q", m.dsn)
	}
}

func TestUpdate_QueryResultMsg(t *testing.T) {
	m := New()

	result := &adapter.QueryResult{
		Duration: 150 * time.Millisecond,
		RowCount: 42,
		Message:  "42 rows affected",
	}
	m, _ = m.Update(appmsg.QueryResultMsg{Result: result})

	if m.queryTime != 150*time.Millisecond {
		t.Fatalf("expected queryTime=150ms, got %v", m.queryTime)
	}
	if m.rowCount != 42 {
		t.Fatalf("expected rowCount=42, got %d", m.rowCount)
	}
	if m.message != "42 rows affected" {
		t.Fatalf("expected message '42 rows affected', got %q", m.message)
	}
	if m.isError {
		t.Fatal("expected isError=false for result message")
	}
}

func TestUpdate_QueryResultMsg_NilResult(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.QueryResultMsg{Result: nil})

	// Should not panic and should not change defaults.
	if m.rowCount != -1 {
		t.Fatalf("expected rowCount=-1 unchanged, got %d", m.rowCount)
	}
}

func TestUpdate_QueryResultMsg_NoMessage(t *testing.T) {
	m := New()

	result := &adapter.QueryResult{
		Duration: 5 * time.Second,
		RowCount: 1000,
	}
	m, _ = m.Update(appmsg.QueryResultMsg{Result: result})

	if m.queryTime != 5*time.Second {
		t.Fatalf("expected queryTime=5s, got %v", m.queryTime)
	}
	if m.rowCount != 1000 {
		t.Fatalf("expected rowCount=1000, got %d", m.rowCount)
	}
	// Message should remain empty when result has no message.
	if m.message != "" {
		t.Fatalf("expected empty message, got %q", m.message)
	}
}

func TestUpdate_QueryErrMsg(t *testing.T) {
	m := New()

	m, _ = m.Update(appmsg.QueryErrMsg{Err: errors.New("syntax error near 'SELEC'")})

	if m.message != "syntax error near 'SELEC'" {
		t.Fatalf("expected error message, got %q", m.message)
	}
	if !m.isError {
		t.Fatal("expected isError=true")
	}
}

func TestUpdate_StatusMsg(t *testing.T) {
	m := New()

	m, _ = m.Update(appmsg.StatusMsg{
		Text:     "Export complete",
		IsError:  false,
		Duration: 200 * time.Millisecond,
	})

	if m.message != "Export complete" {
		t.Fatalf("expected message 'Export complete', got %q", m.message)
	}
	if m.isError {
		t.Fatal("expected isError=false")
	}
	if m.queryTime != 200*time.Millisecond {
		t.Fatalf("expected queryTime=200ms, got %v", m.queryTime)
	}
}

func TestUpdate_StatusMsg_Error(t *testing.T) {
	m := New()

	m, _ = m.Update(appmsg.StatusMsg{
		Text:    "Connection failed",
		IsError: true,
	})

	if m.message != "Connection failed" {
		t.Fatalf("expected message 'Connection failed', got %q", m.message)
	}
	if !m.isError {
		t.Fatal("expected isError=true")
	}
}

func TestUpdate_StatusMsg_NoDuration(t *testing.T) {
	m := New()
	m.queryTime = 100 * time.Millisecond

	m, _ = m.Update(appmsg.StatusMsg{
		Text: "Info message",
	})

	// Duration should not change when StatusMsg.Duration is 0.
	if m.queryTime != 100*time.Millisecond {
		t.Fatalf("expected queryTime unchanged at 100ms, got %v", m.queryTime)
	}
}

func TestUpdate_ToggleKeyModeMsg(t *testing.T) {
	m := New()

	if m.keyMode != appmsg.KeyModeStandard {
		t.Fatalf("expected KeyModeStandard, got %v", m.keyMode)
	}

	// Toggle to vim.
	m, _ = m.Update(appmsg.ToggleKeyModeMsg{})
	if m.keyMode != appmsg.KeyModeVim {
		t.Fatalf("expected KeyModeVim after toggle, got %v", m.keyMode)
	}

	// Toggle back to standard.
	m, _ = m.Update(appmsg.ToggleKeyModeMsg{})
	if m.keyMode != appmsg.KeyModeStandard {
		t.Fatalf("expected KeyModeStandard after second toggle, got %v", m.keyMode)
	}
}

func TestSetCursor(t *testing.T) {
	m := New()
	m.SetCursor(10, 25)

	if m.cursorLine != 10 {
		t.Fatalf("expected cursorLine=10, got %d", m.cursorLine)
	}
	if m.cursorCol != 25 {
		t.Fatalf("expected cursorCol=25, got %d", m.cursorCol)
	}
}

func TestKeyMode(t *testing.T) {
	m := New()

	if m.KeyMode() != appmsg.KeyModeStandard {
		t.Fatalf("expected KeyModeStandard")
	}

	m.SetKeyMode(appmsg.KeyModeVim)
	if m.KeyMode() != appmsg.KeyModeVim {
		t.Fatalf("expected KeyModeVim after SetKeyMode")
	}
}

func TestSetVimState(t *testing.T) {
	m := New()
	m.SetVimState(appmsg.VimInsert)

	if m.vimState != appmsg.VimInsert {
		t.Fatalf("expected VimInsert, got %v", m.vimState)
	}
}

func TestView_ZeroWidth(t *testing.T) {
	m := New()
	view := m.View()
	if view != "" {
		t.Fatalf("expected empty view when width=0, got %q", view)
	}
}

func TestView(t *testing.T) {
	m := New()
	m.SetSize(120)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view when width is set")
	}
}

func TestView_Connected(t *testing.T) {
	m := New()
	m.SetSize(120)

	conn := &mockConnection{dbName: "mydb", adapterName: "postgres"}
	m, _ = m.Update(appmsg.ConnectMsg{
		Conn:    conn,
		Adapter: "postgres",
		DSN:     "postgres://localhost/mydb",
	})

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view when connected")
	}
}

func TestView_WithQueryTime(t *testing.T) {
	m := New()
	m.SetSize(120)

	result := &adapter.QueryResult{
		Duration: 42 * time.Millisecond,
		RowCount: 100,
	}
	m, _ = m.Update(appmsg.QueryResultMsg{Result: result})

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view with query time")
	}
}

func TestView_WithError(t *testing.T) {
	m := New()
	m.SetSize(120)

	m, _ = m.Update(appmsg.QueryErrMsg{Err: errors.New("test error")})

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view with error message")
	}
}

func TestView_VimMode(t *testing.T) {
	m := New()
	m.SetSize(120)
	m.SetKeyMode(appmsg.KeyModeVim)
	m.SetVimState(appmsg.VimInsert)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view in vim mode")
	}
}

func TestView_WithCursorPosition(t *testing.T) {
	m := New()
	m.SetSize(120)
	m.SetCursor(5, 10)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view with cursor position")
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd from Init")
	}
}

func TestUpdate_ClearStatusMsg_StaleIgnored(t *testing.T) {
	m := New()

	m, cmd1 := m.Update(appmsg.StatusMsg{Text: "first"})
	if cmd1 == nil {
		t.Fatal("expected clear timer command for first status")
	}
	msg1 := cmd1()
	clear1, ok := msg1.(ClearStatusMsg)
	if !ok {
		t.Fatalf("expected ClearStatusMsg from first timer, got %T", msg1)
	}

	m, cmd2 := m.Update(appmsg.StatusMsg{Text: "second"})
	if cmd2 == nil {
		t.Fatal("expected clear timer command for second status")
	}
	msg2 := cmd2()
	clear2, ok := msg2.(ClearStatusMsg)
	if !ok {
		t.Fatalf("expected ClearStatusMsg from second timer, got %T", msg2)
	}
	if clear1.Gen == clear2.Gen {
		t.Fatalf("expected different generations, got %d and %d", clear1.Gen, clear2.Gen)
	}

	m, _ = m.Update(clear1)
	if m.message != "second" {
		t.Fatalf("stale timer cleared newer message: got %q, want %q", m.message, "second")
	}

	m, _ = m.Update(clear2)
	if m.message != "" {
		t.Fatalf("fresh timer should clear message, got %q", m.message)
	}
}
