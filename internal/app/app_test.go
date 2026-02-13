package app

import (
	"context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/config"
	"github.com/sadopc/gotermsql/internal/schema"
)

// ---------------------------------------------------------------------------
// TestNew: default config
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg, nil, nil)

	t.Run("focusedPane is PaneEditor", func(t *testing.T) {
		if m.focusedPane != PaneEditor {
			t.Errorf("focusedPane = %d, want PaneEditor (%d)", m.focusedPane, PaneEditor)
		}
	})

	t.Run("showSidebar is true", func(t *testing.T) {
		if !m.showSidebar {
			t.Error("showSidebar should be true by default")
		}
	})

	t.Run("keyMode matches config standard", func(t *testing.T) {
		if m.keyMode != KeyModeStandard {
			t.Errorf("keyMode = %d, want KeyModeStandard (%d)", m.keyMode, KeyModeStandard)
		}
	})

	t.Run("tabStates has one entry for tab 0", func(t *testing.T) {
		if len(m.tabStates) != 1 {
			t.Fatalf("tabStates length = %d, want 1", len(m.tabStates))
		}
		ts, ok := m.tabStates[0]
		if !ok {
			t.Fatal("tabStates[0] does not exist")
		}
		if ts == nil {
			t.Fatal("tabStates[0] is nil")
		}
	})

	t.Run("sidebarWidth has default", func(t *testing.T) {
		if m.sidebarWidth != 30 {
			t.Errorf("sidebarWidth = %d, want 30", m.sidebarWidth)
		}
	})

	t.Run("editorHeight has default", func(t *testing.T) {
		if m.editorHeight != 50 {
			t.Errorf("editorHeight = %d, want 50", m.editorHeight)
		}
	})

	t.Run("conn is nil", func(t *testing.T) {
		if m.conn != nil {
			t.Error("conn should be nil initially")
		}
	})

	t.Run("not quitting", func(t *testing.T) {
		if m.quitting {
			t.Error("quitting should be false initially")
		}
	})

	t.Run("not executing", func(t *testing.T) {
		if m.executing {
			t.Error("executing should be false initially")
		}
	})

	t.Run("showHelp is false", func(t *testing.T) {
		if m.showHelp {
			t.Error("showHelp should be false initially")
		}
	})

	t.Run("config is stored", func(t *testing.T) {
		if m.cfg != cfg {
			t.Error("cfg pointer does not match the config passed to New")
		}
	})

	t.Run("history is nil when passed nil", func(t *testing.T) {
		if m.history != nil {
			t.Error("history should be nil when nil was passed")
		}
	})

	t.Run("compEngine is not nil", func(t *testing.T) {
		if m.compEngine == nil {
			t.Error("compEngine should not be nil after New")
		}
	})

	t.Run("standard keymap is used", func(t *testing.T) {
		// Verify that VimUp has no keys (standard mode)
		if len(m.keyMap.VimUp.Keys()) != 0 {
			t.Errorf("keyMap.VimUp should have no keys in standard mode, got %v", m.keyMap.VimUp.Keys())
		}
	})
}

// ---------------------------------------------------------------------------
// TestNew_VimMode
// ---------------------------------------------------------------------------

func TestNew_VimMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.KeyMode = "vim"
	m := New(cfg, nil, nil)

	t.Run("keyMode is KeyModeVim", func(t *testing.T) {
		if m.keyMode != KeyModeVim {
			t.Errorf("keyMode = %d, want KeyModeVim (%d)", m.keyMode, KeyModeVim)
		}
	})

	t.Run("VimKeyMap is used", func(t *testing.T) {
		// Vim keymap should have vim-specific bindings
		if len(m.keyMap.VimUp.Keys()) == 0 {
			t.Error("VimKeyMap.VimUp should have keys")
		}
		if !containsKey(m.keyMap.VimUp, "k") {
			t.Errorf("VimKeyMap.VimUp keys = %v, want to contain %q", m.keyMap.VimUp.Keys(), "k")
		}
		if !containsKey(m.keyMap.VimDown, "j") {
			t.Errorf("VimKeyMap.VimDown keys = %v, want to contain %q", m.keyMap.VimDown.Keys(), "j")
		}
		if !containsKey(m.keyMap.VimInsert, "i") {
			t.Errorf("VimKeyMap.VimInsert keys = %v, want to contain %q", m.keyMap.VimInsert.Keys(), "i")
		}
		if !containsKey(m.keyMap.VimEscape, "esc") {
			t.Errorf("VimKeyMap.VimEscape keys = %v, want to contain %q", m.keyMap.VimEscape.Keys(), "esc")
		}
	})

	t.Run("standard bindings still present", func(t *testing.T) {
		if !containsKey(m.keyMap.Quit, "ctrl+q") {
			t.Errorf("VimKeyMap.Quit should still contain ctrl+q")
		}
		if !containsKey(m.keyMap.ExecuteQuery, "ctrl+enter") {
			t.Errorf("VimKeyMap.ExecuteQuery should still contain ctrl+enter")
		}
	})

	t.Run("other defaults unchanged", func(t *testing.T) {
		if m.focusedPane != PaneEditor {
			t.Errorf("focusedPane = %d, want PaneEditor", m.focusedPane)
		}
		if !m.showSidebar {
			t.Error("showSidebar should be true by default")
		}
		if len(m.tabStates) != 1 {
			t.Errorf("tabStates length = %d, want 1", len(m.tabStates))
		}
	})
}

// ---------------------------------------------------------------------------
// TestNew_WithConnections
// ---------------------------------------------------------------------------

func TestNew_WithConnections(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Connections = []config.SavedConnection{
		{Name: "prod-pg", Adapter: "postgres", Host: "localhost", Port: 5432},
		{Name: "local-sqlite", Adapter: "sqlite", File: "/tmp/test.db"},
	}
	m := New(cfg, nil, nil)

	// Should still create normally without crashing
	if m.cfg != cfg {
		t.Error("cfg should be stored in model")
	}
	if len(m.cfg.Connections) != 2 {
		t.Errorf("cfg.Connections length = %d, want 2", len(m.cfg.Connections))
	}
}

// ---------------------------------------------------------------------------
// TestIsTypingKey
// ---------------------------------------------------------------------------

func TestIsTypingKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		// Regular printable characters
		{"letter a", "a", true},
		{"letter z", "z", true},
		{"letter A", "A", true},
		{"letter Z", "Z", true},
		{"digit 0", "0", true},
		{"digit 9", "9", true},
		{"space", " ", true},
		{"exclamation", "!", true},
		{"at sign", "@", true},
		{"hash", "#", true},
		{"dollar", "$", true},
		{"percent", "%", true},
		{"ampersand", "&", true},
		{"asterisk", "*", true},
		{"open paren", "(", true},
		{"close paren", ")", true},
		{"semicolon", ";", true},
		{"dot", ".", true},
		{"comma", ",", true},
		{"equals", "=", true},
		{"plus", "+", true},
		{"minus", "-", true},
		{"underscore", "_", true},
		{"tilde", "~", true},

		// Backspace and delete are typing keys
		{"backspace", "backspace", true},
		{"delete", "delete", true},

		// Non-typing keys
		{"ctrl+c", "ctrl+c", false},
		{"enter", "enter", false},
		{"tab", "tab", false},
		{"esc", "esc", false},
		{"up", "up", false},
		{"down", "down", false},
		{"left", "left", false},
		{"right", "right", false},
		{"ctrl+q", "ctrl+q", false},
		{"ctrl+t", "ctrl+t", false},
		{"f1", "f1", false},
		{"f5", "f5", false},
		{"shift+tab", "shift+tab", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := keyMsgFromString(tt.key)
			got := isTypingKey(msg)
			if got != tt.want {
				t.Errorf("isTypingKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestInit
// ---------------------------------------------------------------------------

func TestInit(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg, nil, nil)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil (no background tasks)")
	}
}

// ---------------------------------------------------------------------------
// TestView_BeforeWindowSize
// ---------------------------------------------------------------------------

func TestView_BeforeWindowSize(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg, nil, nil)
	// Before receiving a WindowSizeMsg, width and height are 0
	view := m.View()
	if view != "Loading..." {
		t.Errorf("View() before WindowSize = %q, want %q", view, "Loading...")
	}
}

// ---------------------------------------------------------------------------
// TestView_Quitting
// ---------------------------------------------------------------------------

func TestView_Quitting(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg, nil, nil)
	m.quitting = true
	view := m.View()
	if view != "Goodbye!\n" {
		t.Errorf("View() when quitting = %q, want %q", view, "Goodbye!\n")
	}
}

func TestUpdate_SwitchTabMsg_BlursInactiveTabs(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg, nil, nil)

	// Create a second tab.
	model, cmd := m.Update(NewTabMsg{})
	m = model.(Model)
	if cmd != nil {
		model, _ = m.Update(cmd())
		m = model.(Model)
	}

	// Switch to tab 0 and ensure its editor is focused.
	model, _ = m.Update(SwitchTabMsg{TabID: 0})
	m = model.(Model)
	if ts := m.tabStates[0]; ts == nil || !ts.Editor.Focused() {
		t.Fatal("expected tab 0 editor focused before switching")
	}

	// Simulate Ctrl+] path where tab model advances active tab before message delivery.
	nextCmd := m.tabs.NextTab()
	if nextCmd == nil {
		t.Fatal("expected NextTab command")
	}
	model, _ = m.Update(nextCmd())
	m = model.(Model)

	if m.tabs.ActiveID() != 1 {
		t.Fatalf("expected active tab ID 1, got %d", m.tabs.ActiveID())
	}
	if ts := m.tabStates[1]; ts == nil || !ts.Editor.Focused() {
		t.Fatal("expected tab 1 editor focused after switch")
	}
	if ts := m.tabStates[0]; ts != nil && ts.Editor.Focused() {
		t.Fatal("expected inactive tab 0 editor blurred after switch")
	}
}

type testConn struct {
	dbName      string
	cancelCalls int
	closed      bool
}

func (c *testConn) Databases(context.Context) ([]schema.Database, error) {
	return nil, nil
}
func (c *testConn) Tables(context.Context, string, string) ([]schema.Table, error) {
	return nil, nil
}
func (c *testConn) Columns(context.Context, string, string, string) ([]schema.Column, error) {
	return nil, nil
}
func (c *testConn) Indexes(context.Context, string, string, string) ([]schema.Index, error) {
	return nil, nil
}
func (c *testConn) ForeignKeys(context.Context, string, string, string) ([]schema.ForeignKey, error) {
	return nil, nil
}
func (c *testConn) Execute(context.Context, string) (*adapter.QueryResult, error) {
	return nil, nil
}
func (c *testConn) Cancel() error {
	c.cancelCalls++
	return nil
}
func (c *testConn) ExecuteStreaming(context.Context, string, int) (adapter.RowIterator, error) {
	return nil, nil
}
func (c *testConn) Completions(context.Context) ([]adapter.CompletionItem, error) {
	return nil, nil
}
func (c *testConn) Ping(context.Context) error { return nil }
func (c *testConn) Close() error {
	c.closed = true
	return nil
}
func (c *testConn) DatabaseName() string { return c.dbName }
func (c *testConn) AdapterName() string  { return "test" }

type testIter struct {
	closed bool
}

func (it *testIter) FetchNext(context.Context) ([][]string, error) { return nil, io.EOF }
func (it *testIter) FetchPrev(context.Context) ([][]string, error) { return nil, io.EOF }
func (it *testIter) Columns() []adapter.ColumnMeta                 { return []adapter.ColumnMeta{{Name: "col"}} }
func (it *testIter) TotalRows() int64                              { return -1 }
func (it *testIter) Close() error {
	it.closed = true
	return nil
}

func TestUpdate_ConnectMsg_CleansPreviousConnectionState(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg, nil, nil)

	oldConn := &testConn{dbName: "old"}
	newConn := &testConn{dbName: "new"}
	m.conn = oldConn
	m.executing = true

	cancelCalled := false
	m.cancelFunc = func() { cancelCalled = true }

	iter := &testIter{}
	ts := m.tabStates[0]
	ts.Results.SetIterator(iter)

	model, _ := m.Update(ConnectMsg{
		Conn:    newConn,
		Adapter: "test",
		DSN:     "test://new",
	})
	m = model.(Model)

	if !cancelCalled {
		t.Fatal("expected previous query context to be cancelled on reconnect")
	}
	if oldConn.cancelCalls != 1 {
		t.Fatalf("expected old connection cancel to be called once, got %d", oldConn.cancelCalls)
	}
	if !oldConn.closed {
		t.Fatal("expected old connection to be closed on reconnect")
	}
	if !iter.closed {
		t.Fatal("expected existing iterator to be closed on reconnect")
	}
	if m.conn != newConn {
		t.Fatal("expected new connection to be installed")
	}
	if m.executing {
		t.Fatal("expected executing=false after reconnect")
	}
}

// ---------------------------------------------------------------------------
// keyMsgFromString creates a tea.KeyMsg from a string representation.
// This handles common key names by mapping to the appropriate KeyType.
// ---------------------------------------------------------------------------

func keyMsgFromString(s string) tea.KeyMsg {
	// For single printable characters (length 1, ASCII 32-126)
	if len(s) == 1 && s[0] >= 32 && s[0] <= 126 {
		return tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune(s),
		}
	}

	// Map named keys to their bubbletea KeyType
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+q":
		return tea.KeyMsg{Type: tea.KeyCtrlQ}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	case "f1":
		return tea.KeyMsg{Type: tea.KeyF1}
	case "f5":
		return tea.KeyMsg{Type: tea.KeyF5}
	default:
		// Fallback: treat as runes (multi-byte string)
		return tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune(s),
		}
	}
}
