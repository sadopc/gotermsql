package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// containsKey checks whether the binding's keys contain the given key string.
func containsKey(b key.Binding, target string) bool {
	for _, k := range b.Keys() {
		if k == target {
			return true
		}
	}
	return false
}

// requireNonEmpty fails the test if the binding has no keys.
func requireNonEmpty(t *testing.T, name string, b key.Binding) {
	t.Helper()
	if len(b.Keys()) == 0 {
		t.Errorf("%s binding has no keys", name)
	}
}

// ---------------------------------------------------------------------------
// StandardKeyMap
// ---------------------------------------------------------------------------

func TestStandardKeyMap(t *testing.T) {
	km := StandardKeyMap()

	t.Run("ExecuteQuery has keys", func(t *testing.T) {
		requireNonEmpty(t, "ExecuteQuery", km.ExecuteQuery)
		if !containsKey(km.ExecuteQuery, "ctrl+enter") && !containsKey(km.ExecuteQuery, "f5") {
			t.Error("ExecuteQuery should contain ctrl+enter or f5")
		}
	})

	t.Run("Quit has ctrl+q", func(t *testing.T) {
		requireNonEmpty(t, "Quit", km.Quit)
		if !containsKey(km.Quit, "ctrl+q") {
			t.Errorf("Quit keys = %v, want to contain %q", km.Quit.Keys(), "ctrl+q")
		}
	})

	t.Run("FocusNext has tab", func(t *testing.T) {
		requireNonEmpty(t, "FocusNext", km.FocusNext)
		if !containsKey(km.FocusNext, "tab") {
			t.Errorf("FocusNext keys = %v, want to contain %q", km.FocusNext.Keys(), "tab")
		}
	})

	t.Run("FocusPrev has shift+tab", func(t *testing.T) {
		requireNonEmpty(t, "FocusPrev", km.FocusPrev)
		if !containsKey(km.FocusPrev, "shift+tab") {
			t.Errorf("FocusPrev keys = %v, want to contain %q", km.FocusPrev.Keys(), "shift+tab")
		}
	})

	t.Run("NewTab has ctrl+t", func(t *testing.T) {
		requireNonEmpty(t, "NewTab", km.NewTab)
		if !containsKey(km.NewTab, "ctrl+t") {
			t.Errorf("NewTab keys = %v, want to contain %q", km.NewTab.Keys(), "ctrl+t")
		}
	})

	t.Run("CloseTab has ctrl+w", func(t *testing.T) {
		requireNonEmpty(t, "CloseTab", km.CloseTab)
		if !containsKey(km.CloseTab, "ctrl+w") {
			t.Errorf("CloseTab keys = %v, want to contain %q", km.CloseTab.Keys(), "ctrl+w")
		}
	})

	t.Run("CancelQuery has ctrl+c", func(t *testing.T) {
		requireNonEmpty(t, "CancelQuery", km.CancelQuery)
		if !containsKey(km.CancelQuery, "ctrl+c") {
			t.Errorf("CancelQuery keys = %v, want to contain %q", km.CancelQuery.Keys(), "ctrl+c")
		}
	})

	t.Run("Help has f1", func(t *testing.T) {
		requireNonEmpty(t, "Help", km.Help)
		if !containsKey(km.Help, "f1") {
			t.Errorf("Help keys = %v, want to contain %q", km.Help.Keys(), "f1")
		}
	})

	t.Run("ToggleKeyMode has f2", func(t *testing.T) {
		requireNonEmpty(t, "ToggleKeyMode", km.ToggleKeyMode)
		if !containsKey(km.ToggleKeyMode, "f2") {
			t.Errorf("ToggleKeyMode keys = %v, want to contain %q", km.ToggleKeyMode.Keys(), "f2")
		}
	})

	t.Run("ToggleSidebar has ctrl+b", func(t *testing.T) {
		requireNonEmpty(t, "ToggleSidebar", km.ToggleSidebar)
		if !containsKey(km.ToggleSidebar, "ctrl+b") {
			t.Errorf("ToggleSidebar keys = %v, want to contain %q", km.ToggleSidebar.Keys(), "ctrl+b")
		}
	})

	t.Run("RefreshSchema has ctrl+r", func(t *testing.T) {
		requireNonEmpty(t, "RefreshSchema", km.RefreshSchema)
		if !containsKey(km.RefreshSchema, "ctrl+r") {
			t.Errorf("RefreshSchema keys = %v, want to contain %q", km.RefreshSchema.Keys(), "ctrl+r")
		}
	})

	t.Run("Export has ctrl+e", func(t *testing.T) {
		requireNonEmpty(t, "Export", km.Export)
		if !containsKey(km.Export, "ctrl+e") {
			t.Errorf("Export keys = %v, want to contain %q", km.Export.Keys(), "ctrl+e")
		}
	})

	t.Run("OpenConnMgr has ctrl+o", func(t *testing.T) {
		requireNonEmpty(t, "OpenConnMgr", km.OpenConnMgr)
		if !containsKey(km.OpenConnMgr, "ctrl+o") {
			t.Errorf("OpenConnMgr keys = %v, want to contain %q", km.OpenConnMgr.Keys(), "ctrl+o")
		}
	})

}

func TestStandardKeyMap_AllNavigationBindingsHaveKeys(t *testing.T) {
	km := StandardKeyMap()
	bindings := map[string]key.Binding{
		"FocusNext":    km.FocusNext,
		"FocusPrev":    km.FocusPrev,
		"FocusSidebar": km.FocusSidebar,
		"FocusEditor":  km.FocusEditor,
		"FocusResults": km.FocusResults,
	}
	for name, b := range bindings {
		requireNonEmpty(t, name, b)
	}
}

func TestStandardKeyMap_AllTabBindingsHaveKeys(t *testing.T) {
	km := StandardKeyMap()
	bindings := map[string]key.Binding{
		"NewTab":   km.NewTab,
		"CloseTab": km.CloseTab,
		"NextTab":  km.NextTab,
		"PrevTab":  km.PrevTab,
	}
	for name, b := range bindings {
		requireNonEmpty(t, name, b)
	}
}

func TestStandardKeyMap_AllResizeBindingsHaveKeys(t *testing.T) {
	km := StandardKeyMap()
	bindings := map[string]key.Binding{
		"ResizeLeft":  km.ResizeLeft,
		"ResizeRight": km.ResizeRight,
		"ResizeUp":    km.ResizeUp,
		"ResizeDown":  km.ResizeDown,
	}
	for name, b := range bindings {
		requireNonEmpty(t, name, b)
	}
}

func TestStandardKeyMap_VimBindingsAreEmpty(t *testing.T) {
	km := StandardKeyMap()
	// Standard keymap should NOT have vim-specific bindings set
	vimBindings := map[string]key.Binding{
		"VimUp":     km.VimUp,
		"VimDown":   km.VimDown,
		"VimLeft":   km.VimLeft,
		"VimRight":  km.VimRight,
		"VimInsert": km.VimInsert,
		"VimAppend": km.VimAppend,
		"VimEscape": km.VimEscape,
		"VimTop":    km.VimTop,
		"VimBottom": km.VimBottom,
		"VimSearch": km.VimSearch,
		"VimVisual": km.VimVisual,
		"VimYank":   km.VimYank,
	}
	for name, b := range vimBindings {
		if len(b.Keys()) != 0 {
			t.Errorf("StandardKeyMap.%s should have no keys, got %v", name, b.Keys())
		}
	}
}

// ---------------------------------------------------------------------------
// VimKeyMap
// ---------------------------------------------------------------------------

func TestVimKeyMap(t *testing.T) {
	km := VimKeyMap()

	t.Run("VimUp has k", func(t *testing.T) {
		requireNonEmpty(t, "VimUp", km.VimUp)
		if !containsKey(km.VimUp, "k") {
			t.Errorf("VimUp keys = %v, want to contain %q", km.VimUp.Keys(), "k")
		}
	})

	t.Run("VimDown has j", func(t *testing.T) {
		requireNonEmpty(t, "VimDown", km.VimDown)
		if !containsKey(km.VimDown, "j") {
			t.Errorf("VimDown keys = %v, want to contain %q", km.VimDown.Keys(), "j")
		}
	})

	t.Run("VimLeft has h", func(t *testing.T) {
		requireNonEmpty(t, "VimLeft", km.VimLeft)
		if !containsKey(km.VimLeft, "h") {
			t.Errorf("VimLeft keys = %v, want to contain %q", km.VimLeft.Keys(), "h")
		}
	})

	t.Run("VimRight has l", func(t *testing.T) {
		requireNonEmpty(t, "VimRight", km.VimRight)
		if !containsKey(km.VimRight, "l") {
			t.Errorf("VimRight keys = %v, want to contain %q", km.VimRight.Keys(), "l")
		}
	})

	t.Run("VimInsert has i", func(t *testing.T) {
		requireNonEmpty(t, "VimInsert", km.VimInsert)
		if !containsKey(km.VimInsert, "i") {
			t.Errorf("VimInsert keys = %v, want to contain %q", km.VimInsert.Keys(), "i")
		}
	})

	t.Run("VimAppend has a", func(t *testing.T) {
		requireNonEmpty(t, "VimAppend", km.VimAppend)
		if !containsKey(km.VimAppend, "a") {
			t.Errorf("VimAppend keys = %v, want to contain %q", km.VimAppend.Keys(), "a")
		}
	})

	t.Run("VimEscape has esc", func(t *testing.T) {
		requireNonEmpty(t, "VimEscape", km.VimEscape)
		if !containsKey(km.VimEscape, "esc") {
			t.Errorf("VimEscape keys = %v, want to contain %q", km.VimEscape.Keys(), "esc")
		}
	})

	t.Run("VimTop has g", func(t *testing.T) {
		requireNonEmpty(t, "VimTop", km.VimTop)
		if !containsKey(km.VimTop, "g") {
			t.Errorf("VimTop keys = %v, want to contain %q", km.VimTop.Keys(), "g")
		}
	})

	t.Run("VimBottom has G", func(t *testing.T) {
		requireNonEmpty(t, "VimBottom", km.VimBottom)
		if !containsKey(km.VimBottom, "G") {
			t.Errorf("VimBottom keys = %v, want to contain %q", km.VimBottom.Keys(), "G")
		}
	})

	t.Run("VimSearch has /", func(t *testing.T) {
		requireNonEmpty(t, "VimSearch", km.VimSearch)
		if !containsKey(km.VimSearch, "/") {
			t.Errorf("VimSearch keys = %v, want to contain %q", km.VimSearch.Keys(), "/")
		}
	})

	t.Run("VimVisual has v", func(t *testing.T) {
		requireNonEmpty(t, "VimVisual", km.VimVisual)
		if !containsKey(km.VimVisual, "v") {
			t.Errorf("VimVisual keys = %v, want to contain %q", km.VimVisual.Keys(), "v")
		}
	})

	t.Run("VimYank has y", func(t *testing.T) {
		requireNonEmpty(t, "VimYank", km.VimYank)
		if !containsKey(km.VimYank, "y") {
			t.Errorf("VimYank keys = %v, want to contain %q", km.VimYank.Keys(), "y")
		}
	})
}

func TestVimKeyMap_InheritsStandardBindings(t *testing.T) {
	km := VimKeyMap()

	// VimKeyMap should still have all the standard bindings
	t.Run("Quit still has ctrl+q", func(t *testing.T) {
		if !containsKey(km.Quit, "ctrl+q") {
			t.Errorf("VimKeyMap.Quit keys = %v, want to contain %q", km.Quit.Keys(), "ctrl+q")
		}
	})

	t.Run("ExecuteQuery still has keys", func(t *testing.T) {
		requireNonEmpty(t, "ExecuteQuery", km.ExecuteQuery)
		if !containsKey(km.ExecuteQuery, "ctrl+enter") && !containsKey(km.ExecuteQuery, "f5") {
			t.Error("VimKeyMap.ExecuteQuery should contain ctrl+enter or f5")
		}
	})

	t.Run("FocusNext still has tab", func(t *testing.T) {
		if !containsKey(km.FocusNext, "tab") {
			t.Errorf("VimKeyMap.FocusNext keys = %v, want to contain %q", km.FocusNext.Keys(), "tab")
		}
	})

	t.Run("NewTab still has ctrl+t", func(t *testing.T) {
		if !containsKey(km.NewTab, "ctrl+t") {
			t.Errorf("VimKeyMap.NewTab keys = %v, want to contain %q", km.NewTab.Keys(), "ctrl+t")
		}
	})

	t.Run("CloseTab still has ctrl+w", func(t *testing.T) {
		if !containsKey(km.CloseTab, "ctrl+w") {
			t.Errorf("VimKeyMap.CloseTab keys = %v, want to contain %q", km.CloseTab.Keys(), "ctrl+w")
		}
	})

	t.Run("Help still has f1", func(t *testing.T) {
		if !containsKey(km.Help, "f1") {
			t.Errorf("VimKeyMap.Help keys = %v, want to contain %q", km.Help.Keys(), "f1")
		}
	})

	t.Run("ToggleKeyMode still has f2", func(t *testing.T) {
		if !containsKey(km.ToggleKeyMode, "f2") {
			t.Errorf("VimKeyMap.ToggleKeyMode keys = %v, want to contain %q", km.ToggleKeyMode.Keys(), "f2")
		}
	})
}

func TestVimKeyMap_DirectFocusBindingsPreserved(t *testing.T) {
	km := VimKeyMap()

	if !containsKey(km.FocusSidebar, "alt+1") {
		t.Errorf("VimKeyMap.FocusSidebar keys = %v, want to contain %q", km.FocusSidebar.Keys(), "alt+1")
	}
	if !containsKey(km.FocusEditor, "alt+2") {
		t.Errorf("VimKeyMap.FocusEditor keys = %v, want to contain %q", km.FocusEditor.Keys(), "alt+2")
	}
	if !containsKey(km.FocusResults, "alt+3") {
		t.Errorf("VimKeyMap.FocusResults keys = %v, want to contain %q", km.FocusResults.Keys(), "alt+3")
	}
}

// ---------------------------------------------------------------------------
// ShortHelp
// ---------------------------------------------------------------------------

func TestShortHelp(t *testing.T) {
	km := StandardKeyMap()
	short := km.ShortHelp()
	if len(short) == 0 {
		t.Fatal("ShortHelp() returned empty slice")
	}
	// ShortHelp should return exactly 5 bindings: ExecuteQuery, FocusNext, NewTab, Quit, Help
	if len(short) != 5 {
		t.Errorf("ShortHelp() length = %d, want 5", len(short))
	}
	// Verify each binding has keys
	for i, b := range short {
		if len(b.Keys()) == 0 {
			t.Errorf("ShortHelp()[%d] has no keys", i)
		}
	}
}

func TestShortHelp_VimKeyMap(t *testing.T) {
	km := VimKeyMap()
	short := km.ShortHelp()
	if len(short) == 0 {
		t.Fatal("VimKeyMap.ShortHelp() returned empty slice")
	}
	// VimKeyMap inherits ShortHelp from the same method, should still work
	for i, b := range short {
		if len(b.Keys()) == 0 {
			t.Errorf("VimKeyMap.ShortHelp()[%d] has no keys", i)
		}
	}
}

// ---------------------------------------------------------------------------
// FullHelp
// ---------------------------------------------------------------------------

func TestFullHelp(t *testing.T) {
	km := StandardKeyMap()
	full := km.FullHelp()
	if len(full) == 0 {
		t.Fatal("FullHelp() returned empty slice")
	}
	// FullHelp returns 6 groups of bindings
	if len(full) != 6 {
		t.Errorf("FullHelp() groups = %d, want 6", len(full))
	}
	// Each group should be non-empty
	for i, group := range full {
		if len(group) == 0 {
			t.Errorf("FullHelp()[%d] is empty", i)
		}
		// Each binding in the group should have keys
		for j, b := range group {
			if len(b.Keys()) == 0 {
				t.Errorf("FullHelp()[%d][%d] has no keys", i, j)
			}
		}
	}
}

func TestFullHelp_VimKeyMap(t *testing.T) {
	km := VimKeyMap()
	full := km.FullHelp()
	if len(full) == 0 {
		t.Fatal("VimKeyMap.FullHelp() returned empty slice")
	}
	for i, group := range full {
		if len(group) == 0 {
			t.Errorf("VimKeyMap.FullHelp()[%d] is empty", i)
		}
	}
}

func TestFullHelp_ContainsAllGroups(t *testing.T) {
	km := StandardKeyMap()
	full := km.FullHelp()

	// Group 0: Editor actions (ExecuteQuery, CancelQuery, Export)
	if len(full[0]) != 3 {
		t.Errorf("FullHelp group 0 (editor) length = %d, want 3", len(full[0]))
	}
	// Group 1: Navigation (FocusNext, FocusPrev, FocusSidebar, FocusEditor, FocusResults)
	if len(full[1]) != 5 {
		t.Errorf("FullHelp group 1 (navigation) length = %d, want 5", len(full[1]))
	}
	// Group 2: Tabs (NewTab, CloseTab, NextTab, PrevTab)
	if len(full[2]) != 4 {
		t.Errorf("FullHelp group 2 (tabs) length = %d, want 4", len(full[2]))
	}
	// Group 3: App (ToggleKeyMode, ToggleSidebar, RefreshSchema, OpenConnMgr, History)
	if len(full[3]) != 5 {
		t.Errorf("FullHelp group 3 (app) length = %d, want 5", len(full[3]))
	}
	// Group 4: Resize (ResizeLeft, ResizeRight, ResizeUp, ResizeDown)
	if len(full[4]) != 4 {
		t.Errorf("FullHelp group 4 (resize) length = %d, want 4", len(full[4]))
	}
	// Group 5: Quit + Help
	if len(full[5]) != 2 {
		t.Errorf("FullHelp group 5 (quit/help) length = %d, want 2", len(full[5]))
	}
}

// ---------------------------------------------------------------------------
// Specific key values
// ---------------------------------------------------------------------------

func TestStandardKeyMap_SpecificKeyValues(t *testing.T) {
	km := StandardKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
		wantKey string
	}{
		{"FocusNext", km.FocusNext, "tab"},
		{"FocusPrev", km.FocusPrev, "shift+tab"},
		{"FocusSidebar", km.FocusSidebar, "alt+1"},
		{"FocusEditor", km.FocusEditor, "alt+2"},
		{"FocusResults", km.FocusResults, "alt+3"},
		{"NewTab", km.NewTab, "ctrl+t"},
		{"CloseTab", km.CloseTab, "ctrl+w"},
		{"Quit", km.Quit, "ctrl+q"},
		{"Help", km.Help, "f1"},
		{"ToggleKeyMode", km.ToggleKeyMode, "f2"},
		{"ToggleSidebar", km.ToggleSidebar, "ctrl+b"},
		{"RefreshSchema", km.RefreshSchema, "ctrl+r"},
		{"OpenConnMgr", km.OpenConnMgr, "ctrl+o"},
		{"Export", km.Export, "ctrl+e"},
		{"CancelQuery", km.CancelQuery, "ctrl+c"},
		{"ResizeLeft", km.ResizeLeft, "ctrl+left"},
		{"ResizeRight", km.ResizeRight, "ctrl+right"},
		{"ResizeUp", km.ResizeUp, "ctrl+up"},
		{"ResizeDown", km.ResizeDown, "ctrl+down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsKey(tt.binding, tt.wantKey) {
				t.Errorf("%s keys = %v, want to contain %q", tt.name, tt.binding.Keys(), tt.wantKey)
			}
		})
	}
}

func TestStandardKeyMap_ExecuteQueryMultipleKeys(t *testing.T) {
	km := StandardKeyMap()
	keys := km.ExecuteQuery.Keys()
	if len(keys) < 2 {
		t.Errorf("ExecuteQuery should have at least 2 keys (ctrl+enter and f5), got %v", keys)
	}
	if !containsKey(km.ExecuteQuery, "ctrl+enter") {
		t.Errorf("ExecuteQuery missing ctrl+enter, keys = %v", keys)
	}
	if !containsKey(km.ExecuteQuery, "f5") {
		t.Errorf("ExecuteQuery missing f5, keys = %v", keys)
	}
}

func TestStandardKeyMap_NextTabMultipleKeys(t *testing.T) {
	km := StandardKeyMap()
	keys := km.NextTab.Keys()
	if len(keys) < 2 {
		t.Errorf("NextTab should have at least 2 keys, got %v", keys)
	}
	if !containsKey(km.NextTab, "ctrl+pgdown") {
		t.Errorf("NextTab missing ctrl+pgdown, keys = %v", keys)
	}
	if !containsKey(km.NextTab, "ctrl+]") {
		t.Errorf("NextTab missing ctrl+], keys = %v", keys)
	}
}

func TestStandardKeyMap_PrevTabMultipleKeys(t *testing.T) {
	km := StandardKeyMap()
	keys := km.PrevTab.Keys()
	if len(keys) < 2 {
		t.Errorf("PrevTab should have at least 2 keys, got %v", keys)
	}
	if !containsKey(km.PrevTab, "ctrl+pgup") {
		t.Errorf("PrevTab missing ctrl+pgup, keys = %v", keys)
	}
	if !containsKey(km.PrevTab, "ctrl+[") {
		t.Errorf("PrevTab missing ctrl+[, keys = %v", keys)
	}
}
