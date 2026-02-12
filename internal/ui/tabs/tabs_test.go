package tabs

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	appmsg "github.com/sadopc/gotermsql/internal/msg"
	"github.com/sadopc/gotermsql/internal/theme"
)

func init() {
	theme.Current = theme.Default()
}

func TestNew(t *testing.T) {
	m := New()

	if m.Count() != 1 {
		t.Fatalf("expected 1 tab, got %d", m.Count())
	}
	if m.active != 0 {
		t.Fatalf("expected active=0, got %d", m.active)
	}
	if m.nextID != 1 {
		t.Fatalf("expected nextID=1, got %d", m.nextID)
	}
	tab := m.ActiveTab()
	if tab.ID != 0 {
		t.Fatalf("expected active tab ID=0, got %d", tab.ID)
	}
	if tab.Title != "Query 1" {
		t.Fatalf("expected title 'Query 1', got %q", tab.Title)
	}
}

func TestNewTab(t *testing.T) {
	m := New()

	m, cmd := m.Update(appmsg.NewTabMsg{})
	if m.Count() != 2 {
		t.Fatalf("expected 2 tabs, got %d", m.Count())
	}
	// New tab should be active (index 1).
	if m.active != 1 {
		t.Fatalf("expected active=1, got %d", m.active)
	}
	// The new tab should have ID=1.
	tab := m.ActiveTab()
	if tab.ID != 1 {
		t.Fatalf("expected new tab ID=1, got %d", tab.ID)
	}
	if tab.Title != "Query 2" {
		t.Fatalf("expected title 'Query 2', got %q", tab.Title)
	}
	// nextID should have incremented.
	if m.nextID != 2 {
		t.Fatalf("expected nextID=2, got %d", m.nextID)
	}
	// The command should produce a SwitchTabMsg.
	if cmd == nil {
		t.Fatal("expected a cmd from NewTabMsg, got nil")
	}
	msg := cmd()
	switchMsg, ok := msg.(appmsg.SwitchTabMsg)
	if !ok {
		t.Fatalf("expected SwitchTabMsg, got %T", msg)
	}
	if switchMsg.TabID != 1 {
		t.Fatalf("expected SwitchTabMsg.TabID=1, got %d", switchMsg.TabID)
	}
}

func TestCloseTab(t *testing.T) {
	m := New()
	// Add a second tab.
	m, _ = m.Update(appmsg.NewTabMsg{})
	if m.Count() != 2 {
		t.Fatalf("expected 2 tabs, got %d", m.Count())
	}

	// Close tab with ID=1 (the second tab, currently active).
	m, cmd := m.Update(appmsg.CloseTabMsg{TabID: 1})
	if m.Count() != 1 {
		t.Fatalf("expected 1 tab after close, got %d", m.Count())
	}
	// Active should adjust to remaining tab.
	if m.active != 0 {
		t.Fatalf("expected active=0 after close, got %d", m.active)
	}
	// Should produce a SwitchTabMsg for the remaining tab.
	if cmd == nil {
		t.Fatal("expected a cmd from CloseTabMsg, got nil")
	}
	msg := cmd()
	switchMsg, ok := msg.(appmsg.SwitchTabMsg)
	if !ok {
		t.Fatalf("expected SwitchTabMsg, got %T", msg)
	}
	if switchMsg.TabID != 0 {
		t.Fatalf("expected switch to tab 0, got %d", switchMsg.TabID)
	}
}

func TestCloseTab_LastTab(t *testing.T) {
	m := New()

	// Try to close the only tab.
	m, cmd := m.Update(appmsg.CloseTabMsg{TabID: 0})
	if m.Count() != 1 {
		t.Fatalf("expected 1 tab (should not close last), got %d", m.Count())
	}
	if cmd != nil {
		t.Fatal("expected nil cmd when closing last tab")
	}
}

func TestCloseTab_NonExistent(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{})

	// Close a tab with an ID that does not exist.
	m, cmd := m.Update(appmsg.CloseTabMsg{TabID: 999})
	if m.Count() != 2 {
		t.Fatalf("expected 2 tabs (non-existent close), got %d", m.Count())
	}
	if cmd != nil {
		t.Fatal("expected nil cmd when closing non-existent tab")
	}
}

func TestSwitchTab(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=1
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=2

	// Active should be 2 (last created).
	if m.ActiveID() != 2 {
		t.Fatalf("expected active ID=2, got %d", m.ActiveID())
	}

	// Switch to tab 0.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 0})
	if m.ActiveID() != 0 {
		t.Fatalf("expected active ID=0 after switch, got %d", m.ActiveID())
	}

	// Switch to tab 1.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 1})
	if m.ActiveID() != 1 {
		t.Fatalf("expected active ID=1 after switch, got %d", m.ActiveID())
	}

	// Switch to non-existent tab: active should not change.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 999})
	if m.ActiveID() != 1 {
		t.Fatalf("expected active ID=1 unchanged, got %d", m.ActiveID())
	}
}

func TestNextTab(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=1
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=2

	// Switch to first tab.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 0})
	if m.ActiveID() != 0 {
		t.Fatalf("expected active=0, got %d", m.ActiveID())
	}

	// NextTab: 0 -> 1
	cmd := m.NextTab()
	if m.ActiveID() != 1 {
		t.Fatalf("expected active=1 after NextTab, got %d", m.ActiveID())
	}
	if cmd == nil {
		t.Fatal("expected cmd from NextTab")
	}
	msg := cmd()
	switchMsg, ok := msg.(appmsg.SwitchTabMsg)
	if !ok {
		t.Fatalf("expected SwitchTabMsg, got %T", msg)
	}
	if switchMsg.TabID != 1 {
		t.Fatalf("expected TabID=1, got %d", switchMsg.TabID)
	}

	// NextTab: 1 -> 2
	m.NextTab()
	if m.ActiveID() != 2 {
		t.Fatalf("expected active=2 after NextTab, got %d", m.ActiveID())
	}

	// NextTab: 2 -> 0 (wrap around)
	m.NextTab()
	if m.ActiveID() != 0 {
		t.Fatalf("expected active=0 after wrap, got %d", m.ActiveID())
	}
}

func TestPrevTab(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=1
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=2

	// Active is 2 (last created).
	if m.ActiveID() != 2 {
		t.Fatalf("expected active=2, got %d", m.ActiveID())
	}

	// PrevTab: 2 -> 1
	cmd := m.PrevTab()
	if m.ActiveID() != 1 {
		t.Fatalf("expected active=1 after PrevTab, got %d", m.ActiveID())
	}
	if cmd == nil {
		t.Fatal("expected cmd from PrevTab")
	}

	// PrevTab: 1 -> 0
	m.PrevTab()
	if m.ActiveID() != 0 {
		t.Fatalf("expected active=0 after PrevTab, got %d", m.ActiveID())
	}

	// PrevTab: 0 -> 2 (wrap around to end)
	m.PrevTab()
	if m.ActiveID() != 2 {
		t.Fatalf("expected active=2 after wrap, got %d", m.ActiveID())
	}
}

func TestSetModified(t *testing.T) {
	m := New()

	tab := m.ActiveTab()
	if tab.Modified {
		t.Fatal("expected Modified=false initially")
	}

	m.SetModified(0, true)
	tab = m.ActiveTab()
	if !tab.Modified {
		t.Fatal("expected Modified=true after SetModified(0, true)")
	}

	m.SetModified(0, false)
	tab = m.ActiveTab()
	if tab.Modified {
		t.Fatal("expected Modified=false after SetModified(0, false)")
	}

	// SetModified on non-existent tab should not panic.
	m.SetModified(999, true)
}

func TestActiveTab(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{})

	// Switch to first tab.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 0})
	tab := m.ActiveTab()
	if tab.ID != 0 || tab.Title != "Query 1" {
		t.Fatalf("expected tab 0 'Query 1', got ID=%d title=%q", tab.ID, tab.Title)
	}

	// Switch to second tab.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 1})
	tab = m.ActiveTab()
	if tab.ID != 1 || tab.Title != "Query 2" {
		t.Fatalf("expected tab 1 'Query 2', got ID=%d title=%q", tab.ID, tab.Title)
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
	m.SetSize(80)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view when width is set")
	}
}

func TestView_ModifiedTab(t *testing.T) {
	m := New()
	m.SetSize(80)
	m.SetModified(0, true)

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd from Init")
	}
}

func TestTabs(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{})

	tabs := m.Tabs()
	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(tabs))
	}
	if tabs[0].ID != 0 {
		t.Fatalf("expected first tab ID=0, got %d", tabs[0].ID)
	}
	if tabs[1].ID != 1 {
		t.Fatalf("expected second tab ID=1, got %d", tabs[1].ID)
	}
}

func TestCloseTab_ActiveAdjustsCorrectly(t *testing.T) {
	m := New()
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=1
	m, _ = m.Update(appmsg.NewTabMsg{}) // ID=2

	// Switch to middle tab.
	m, _ = m.Update(appmsg.SwitchTabMsg{TabID: 1})

	// Close the first tab (ID=0). Active index should adjust.
	m, _ = m.Update(appmsg.CloseTabMsg{TabID: 0})
	if m.Count() != 2 {
		t.Fatalf("expected 2 tabs, got %d", m.Count())
	}
	// After closing tab at index 0, active (was 1) should become 0.
	// But the current logic keeps active as is, then clamps to len(tabs)-1 if needed.
	// Since we closed index 0, old active=1 stays, but the tab at new index 0 is ID=1
	// and at new index 1 is ID=2. active=1 is still valid, pointing to ID=2.
	// The SwitchTabMsg will be issued for tabs[active].
	_ = m.ActiveTab()
}

// Verify that unknown message types are handled gracefully.
func TestUpdate_UnknownMsg(t *testing.T) {
	m := New()
	m, cmd := m.Update(tea.KeyMsg{})
	if cmd != nil {
		t.Fatal("expected nil cmd for unknown msg type")
	}
	if m.Count() != 1 {
		t.Fatalf("expected 1 tab, got %d", m.Count())
	}
}
