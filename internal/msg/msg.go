package msg

import (
	"time"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
)

// Pane focus targets.
type Pane int

const (
	PaneSidebar Pane = iota
	PaneEditor
	PaneResults
)

// KeyMode represents the active keybinding mode.
type KeyMode int

const (
	KeyModeStandard KeyMode = iota
	KeyModeVim
)

func (m KeyMode) String() string {
	if m == KeyModeVim {
		return "vim"
	}
	return "standard"
}

// ParseKeyMode parses a string into a KeyMode.
func ParseKeyMode(s string) KeyMode {
	if s == "vim" {
		return KeyModeVim
	}
	return KeyModeStandard
}

// VimState tracks vim mode state.
type VimState int

const (
	VimNormal VimState = iota
	VimInsert
	VimVisual
)

func (s VimState) String() string {
	switch s {
	case VimInsert:
		return "INSERT"
	case VimVisual:
		return "VISUAL"
	default:
		return "NORMAL"
	}
}

// FocusMsg requests a pane focus change.
type FocusMsg struct {
	Pane Pane
}

// ConnectMsg is sent when a database connection is established.
type ConnectMsg struct {
	Conn    adapter.Connection
	Adapter string
	DSN     string
}

// ConnectErrMsg is sent when a connection attempt fails.
type ConnectErrMsg struct {
	Err error
}

// DisconnectMsg is sent when the connection is closed.
type DisconnectMsg struct{}

// SchemaLoadedMsg is sent when schema introspection completes.
type SchemaLoadedMsg struct {
	Databases []schema.Database
	ConnGen   uint64
	Warnings  []string
}

// SchemaErrMsg is sent when schema loading fails.
type SchemaErrMsg struct {
	Err     error
	ConnGen uint64
}

// ExecuteQueryMsg requests query execution.
type ExecuteQueryMsg struct {
	Query string
	TabID int
}

// QueryStartedMsg is sent when a query begins executing.
type QueryStartedMsg struct {
	TabID   int
	RunID   uint64
	ConnGen uint64
}

// QueryResultMsg is sent when query execution completes.
type QueryResultMsg struct {
	Result  *adapter.QueryResult
	TabID   int
	RunID   uint64
	ConnGen uint64
}

// QueryErrMsg is sent when query execution fails.
type QueryErrMsg struct {
	Err     error
	TabID   int
	RunID   uint64
	ConnGen uint64
}

// NewTabMsg requests creating a new query tab.
type NewTabMsg struct {
	Query string
}

// CloseTabMsg requests closing a tab.
type CloseTabMsg struct {
	TabID int
}

// SwitchTabMsg requests switching to a tab.
type SwitchTabMsg struct {
	TabID int
}

// StatusMsg updates the status bar text.
type StatusMsg struct {
	Text     string
	IsError  bool
	Duration time.Duration
}

// ToggleKeyModeMsg switches between vim and standard keybindings.
type ToggleKeyModeMsg struct{}

// ExportRequestMsg requests exporting results.
type ExportRequestMsg struct {
	Format string
	Path   string
}

// ExportCompleteMsg is sent when export finishes.
type ExportCompleteMsg struct {
	Path     string
	RowCount int64
}

// ExportErrMsg is sent when export fails.
type ExportErrMsg struct {
	Err error
}

// InsertTextMsg inserts text into the active editor.
type InsertTextMsg struct {
	Text string
}

// RefreshSchemaMsg triggers a schema refresh.
type RefreshSchemaMsg struct{}

// OpenHistoryMsg opens the query history panel.
type OpenHistoryMsg struct{}
