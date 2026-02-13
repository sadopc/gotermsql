package msg

import (
	"errors"
	"testing"
	"time"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

// ---------------------------------------------------------------------------
// KeyMode
// ---------------------------------------------------------------------------

func TestKeyMode_String(t *testing.T) {
	tests := []struct {
		mode KeyMode
		want string
	}{
		{KeyModeStandard, "standard"},
		{KeyModeVim, "vim"},
	}
	for _, tt := range tests {
		got := tt.mode.String()
		if got != tt.want {
			t.Errorf("KeyMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestParseKeyMode(t *testing.T) {
	tests := []struct {
		input string
		want  KeyMode
	}{
		{"vim", KeyModeVim},
		{"standard", KeyModeStandard},
		{"", KeyModeStandard},
		{"anything", KeyModeStandard},
		{"VIM", KeyModeStandard},   // case-sensitive: only lowercase "vim" matches
		{"Vim", KeyModeStandard},   // case-sensitive
		{"emacs", KeyModeStandard}, // unknown defaults to standard
	}
	for _, tt := range tests {
		got := ParseKeyMode(tt.input)
		if got != tt.want {
			t.Errorf("ParseKeyMode(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// VimState
// ---------------------------------------------------------------------------

func TestVimState_String(t *testing.T) {
	tests := []struct {
		state VimState
		want  string
	}{
		{VimNormal, "NORMAL"},
		{VimInsert, "INSERT"},
		{VimVisual, "VISUAL"},
	}
	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("VimState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestVimState_String_UnknownDefaultsToNormal(t *testing.T) {
	// Any state value beyond the defined constants should default to "NORMAL"
	unknown := VimState(99)
	got := unknown.String()
	if got != "NORMAL" {
		t.Errorf("VimState(99).String() = %q, want %q", got, "NORMAL")
	}
}

// ---------------------------------------------------------------------------
// Pane constants
// ---------------------------------------------------------------------------

func TestPaneConstants(t *testing.T) {
	// Verify the three pane constants are distinct
	if PaneSidebar == PaneEditor {
		t.Error("PaneSidebar should not equal PaneEditor")
	}
	if PaneSidebar == PaneResults {
		t.Error("PaneSidebar should not equal PaneResults")
	}
	if PaneEditor == PaneResults {
		t.Error("PaneEditor should not equal PaneResults")
	}
}

func TestPaneConstants_IotaOrder(t *testing.T) {
	// Verify expected iota ordering
	if PaneSidebar != 0 {
		t.Errorf("PaneSidebar = %d, want 0", PaneSidebar)
	}
	if PaneEditor != 1 {
		t.Errorf("PaneEditor = %d, want 1", PaneEditor)
	}
	if PaneResults != 2 {
		t.Errorf("PaneResults = %d, want 2", PaneResults)
	}
}

// ---------------------------------------------------------------------------
// Message types: verify struct fields are set correctly
// ---------------------------------------------------------------------------

func TestConnectMsg(t *testing.T) {
	m := ConnectMsg{
		Adapter: "postgres",
		DSN:     "postgres://user:pass@localhost:5432/db",
		Conn:    nil, // we don't create a real connection in unit tests
	}
	if m.Adapter != "postgres" {
		t.Errorf("ConnectMsg.Adapter = %q, want %q", m.Adapter, "postgres")
	}
	if m.DSN != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("ConnectMsg.DSN = %q, want full DSN string", m.DSN)
	}
	if m.Conn != nil {
		t.Error("ConnectMsg.Conn should be nil for this test")
	}
}

func TestConnectErrMsg(t *testing.T) {
	err := errors.New("connection refused")
	m := ConnectErrMsg{Err: err}
	if m.Err == nil {
		t.Fatal("ConnectErrMsg.Err should not be nil")
	}
	if m.Err.Error() != "connection refused" {
		t.Errorf("ConnectErrMsg.Err = %q, want %q", m.Err.Error(), "connection refused")
	}
}

func TestSchemaLoadedMsg(t *testing.T) {
	dbs := []schema.Database{
		{
			Name: "testdb",
			Schemas: []schema.Schema{
				{
					Name: "public",
					Tables: []schema.Table{
						{Name: "users"},
						{Name: "orders"},
					},
				},
			},
		},
		{
			Name: "analytics",
		},
	}
	m := SchemaLoadedMsg{Databases: dbs}
	if len(m.Databases) != 2 {
		t.Fatalf("SchemaLoadedMsg.Databases length = %d, want 2", len(m.Databases))
	}
	if m.Databases[0].Name != "testdb" {
		t.Errorf("Databases[0].Name = %q, want %q", m.Databases[0].Name, "testdb")
	}
	if len(m.Databases[0].Schemas) != 1 {
		t.Fatalf("Databases[0].Schemas length = %d, want 1", len(m.Databases[0].Schemas))
	}
	if len(m.Databases[0].Schemas[0].Tables) != 2 {
		t.Errorf("Databases[0].Schemas[0].Tables length = %d, want 2", len(m.Databases[0].Schemas[0].Tables))
	}
	if m.Databases[1].Name != "analytics" {
		t.Errorf("Databases[1].Name = %q, want %q", m.Databases[1].Name, "analytics")
	}
}

func TestSchemaLoadedMsg_Empty(t *testing.T) {
	m := SchemaLoadedMsg{Databases: nil}
	if m.Databases != nil {
		t.Error("SchemaLoadedMsg.Databases should be nil when not set")
	}
}

func TestExecuteQueryMsg(t *testing.T) {
	m := ExecuteQueryMsg{
		Query: "SELECT * FROM users WHERE id = 1",
		TabID: 3,
	}
	if m.Query != "SELECT * FROM users WHERE id = 1" {
		t.Errorf("ExecuteQueryMsg.Query = %q, want SQL query", m.Query)
	}
	if m.TabID != 3 {
		t.Errorf("ExecuteQueryMsg.TabID = %d, want 3", m.TabID)
	}
}

func TestQueryResultMsg(t *testing.T) {
	result := &adapter.QueryResult{
		Columns: []adapter.ColumnMeta{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "varchar"},
		},
		Rows: [][]string{
			{"1", "Alice"},
			{"2", "Bob"},
		},
		RowCount: 2,
		Duration: 150 * time.Millisecond,
		IsSelect: true,
		Message:  "2 rows returned",
	}
	m := QueryResultMsg{
		Result: result,
		TabID:  5,
	}
	if m.TabID != 5 {
		t.Errorf("QueryResultMsg.TabID = %d, want 5", m.TabID)
	}
	if m.Result == nil {
		t.Fatal("QueryResultMsg.Result should not be nil")
	}
	if m.Result.RowCount != 2 {
		t.Errorf("QueryResultMsg.Result.RowCount = %d, want 2", m.Result.RowCount)
	}
	if len(m.Result.Columns) != 2 {
		t.Errorf("QueryResultMsg.Result.Columns length = %d, want 2", len(m.Result.Columns))
	}
	if m.Result.Columns[0].Name != "id" {
		t.Errorf("QueryResultMsg.Result.Columns[0].Name = %q, want %q", m.Result.Columns[0].Name, "id")
	}
	if len(m.Result.Rows) != 2 {
		t.Errorf("QueryResultMsg.Result.Rows length = %d, want 2", len(m.Result.Rows))
	}
	if !m.Result.IsSelect {
		t.Error("QueryResultMsg.Result.IsSelect should be true")
	}
}

func TestQueryResultMsg_NilResult(t *testing.T) {
	m := QueryResultMsg{Result: nil, TabID: 0}
	if m.Result != nil {
		t.Error("QueryResultMsg.Result should be nil")
	}
}

func TestQueryErrMsg(t *testing.T) {
	err := errors.New("syntax error at position 42")
	m := QueryErrMsg{
		Err:   err,
		TabID: 2,
	}
	if m.Err == nil {
		t.Fatal("QueryErrMsg.Err should not be nil")
	}
	if m.Err.Error() != "syntax error at position 42" {
		t.Errorf("QueryErrMsg.Err = %q, want specific error message", m.Err.Error())
	}
	if m.TabID != 2 {
		t.Errorf("QueryErrMsg.TabID = %d, want 2", m.TabID)
	}
}

func TestNewTabMsg(t *testing.T) {
	m := NewTabMsg{Query: "SELECT 1"}
	if m.Query != "SELECT 1" {
		t.Errorf("NewTabMsg.Query = %q, want %q", m.Query, "SELECT 1")
	}
}

func TestNewTabMsg_Empty(t *testing.T) {
	m := NewTabMsg{}
	if m.Query != "" {
		t.Errorf("NewTabMsg.Query = %q, want empty", m.Query)
	}
}

func TestCloseTabMsg(t *testing.T) {
	m := CloseTabMsg{TabID: 7}
	if m.TabID != 7 {
		t.Errorf("CloseTabMsg.TabID = %d, want 7", m.TabID)
	}
}

func TestSwitchTabMsg(t *testing.T) {
	m := SwitchTabMsg{TabID: 4}
	if m.TabID != 4 {
		t.Errorf("SwitchTabMsg.TabID = %d, want 4", m.TabID)
	}
}

func TestStatusMsg(t *testing.T) {
	m := StatusMsg{
		Text:     "Query executed successfully",
		IsError:  false,
		Duration: 250 * time.Millisecond,
	}
	if m.Text != "Query executed successfully" {
		t.Errorf("StatusMsg.Text = %q, want success message", m.Text)
	}
	if m.IsError {
		t.Error("StatusMsg.IsError should be false")
	}
	if m.Duration != 250*time.Millisecond {
		t.Errorf("StatusMsg.Duration = %v, want 250ms", m.Duration)
	}
}

func TestStatusMsg_Error(t *testing.T) {
	m := StatusMsg{
		Text:     "Connection timeout",
		IsError:  true,
		Duration: 5 * time.Second,
	}
	if !m.IsError {
		t.Error("StatusMsg.IsError should be true for error messages")
	}
	if m.Duration != 5*time.Second {
		t.Errorf("StatusMsg.Duration = %v, want 5s", m.Duration)
	}
}

func TestExportRequestMsg(t *testing.T) {
	m := ExportRequestMsg{
		Format: "csv",
		Path:   "/tmp/export.csv",
	}
	if m.Format != "csv" {
		t.Errorf("ExportRequestMsg.Format = %q, want %q", m.Format, "csv")
	}
	if m.Path != "/tmp/export.csv" {
		t.Errorf("ExportRequestMsg.Path = %q, want %q", m.Path, "/tmp/export.csv")
	}
}

func TestExportRequestMsg_JSON(t *testing.T) {
	m := ExportRequestMsg{
		Format: "json",
		Path:   "/home/user/results.json",
	}
	if m.Format != "json" {
		t.Errorf("ExportRequestMsg.Format = %q, want %q", m.Format, "json")
	}
	if m.Path != "/home/user/results.json" {
		t.Errorf("ExportRequestMsg.Path = %q, want %q", m.Path, "/home/user/results.json")
	}
}

// ---------------------------------------------------------------------------
// Additional message types for coverage
// ---------------------------------------------------------------------------

func TestFocusMsg(t *testing.T) {
	m := FocusMsg{Pane: PaneEditor}
	if m.Pane != PaneEditor {
		t.Errorf("FocusMsg.Pane = %d, want PaneEditor (%d)", m.Pane, PaneEditor)
	}
}

func TestDisconnectMsg(t *testing.T) {
	// DisconnectMsg has no fields, just verify it can be created
	_ = DisconnectMsg{}
}

func TestSchemaErrMsg(t *testing.T) {
	err := errors.New("permission denied")
	m := SchemaErrMsg{Err: err}
	if m.Err.Error() != "permission denied" {
		t.Errorf("SchemaErrMsg.Err = %q, want %q", m.Err.Error(), "permission denied")
	}
}

func TestQueryStartedMsg(t *testing.T) {
	m := QueryStartedMsg{TabID: 10}
	if m.TabID != 10 {
		t.Errorf("QueryStartedMsg.TabID = %d, want 10", m.TabID)
	}
}

func TestExportCompleteMsg(t *testing.T) {
	m := ExportCompleteMsg{
		Path:     "/tmp/data.csv",
		RowCount: 500,
	}
	if m.Path != "/tmp/data.csv" {
		t.Errorf("ExportCompleteMsg.Path = %q, want %q", m.Path, "/tmp/data.csv")
	}
	if m.RowCount != 500 {
		t.Errorf("ExportCompleteMsg.RowCount = %d, want 500", m.RowCount)
	}
}

func TestExportErrMsg(t *testing.T) {
	err := errors.New("disk full")
	m := ExportErrMsg{Err: err}
	if m.Err.Error() != "disk full" {
		t.Errorf("ExportErrMsg.Err = %q, want %q", m.Err.Error(), "disk full")
	}
}

func TestInsertTextMsg(t *testing.T) {
	m := InsertTextMsg{Text: "SELECT * FROM "}
	if m.Text != "SELECT * FROM " {
		t.Errorf("InsertTextMsg.Text = %q, want %q", m.Text, "SELECT * FROM ")
	}
}

func TestToggleKeyModeMsg(t *testing.T) {
	// ToggleKeyModeMsg has no fields, just verify it can be created
	_ = ToggleKeyModeMsg{}
}

func TestRefreshSchemaMsg(t *testing.T) {
	// RefreshSchemaMsg has no fields
	_ = RefreshSchemaMsg{}
}

func TestOpenHistoryMsg(t *testing.T) {
	// OpenHistoryMsg has no fields
	_ = OpenHistoryMsg{}
}

// ---------------------------------------------------------------------------
// KeyMode and VimState constants are correct iota values
// ---------------------------------------------------------------------------

func TestKeyModeConstants(t *testing.T) {
	if KeyModeStandard != 0 {
		t.Errorf("KeyModeStandard = %d, want 0", KeyModeStandard)
	}
	if KeyModeVim != 1 {
		t.Errorf("KeyModeVim = %d, want 1", KeyModeVim)
	}
}

func TestVimStateConstants(t *testing.T) {
	if VimNormal != 0 {
		t.Errorf("VimNormal = %d, want 0", VimNormal)
	}
	if VimInsert != 1 {
		t.Errorf("VimInsert = %d, want 1", VimInsert)
	}
	if VimVisual != 2 {
		t.Errorf("VimVisual = %d, want 2", VimVisual)
	}
}
