package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all application keybindings.
type KeyMap struct {
	// Navigation
	FocusNext    key.Binding
	FocusPrev    key.Binding
	FocusSidebar key.Binding
	FocusEditor  key.Binding
	FocusResults key.Binding

	// Tabs
	NewTab   key.Binding
	CloseTab key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding

	// Editor
	ExecuteQuery key.Binding
	CancelQuery  key.Binding

	// App
	Quit          key.Binding
	Help          key.Binding
	ToggleKeyMode key.Binding
	ToggleSidebar key.Binding
	RefreshSchema key.Binding
	OpenConnMgr   key.Binding
	History       key.Binding
	Export        key.Binding

	// Pane resizing
	ResizeLeft  key.Binding
	ResizeRight key.Binding
	ResizeUp    key.Binding
	ResizeDown  key.Binding

	// Vim normal mode
	VimUp     key.Binding
	VimDown   key.Binding
	VimLeft   key.Binding
	VimRight  key.Binding
	VimInsert key.Binding
	VimAppend key.Binding
	VimEscape key.Binding
	VimTop    key.Binding
	VimBottom key.Binding
	VimSearch key.Binding
	VimVisual key.Binding
	VimYank   key.Binding
}

// StandardKeyMap returns keybindings for standard mode.
func StandardKeyMap() KeyMap {
	return KeyMap{
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab", "ctrl+j"),
			key.WithHelp("shift+tab/ctrl+j", "prev pane"),
		),
		FocusSidebar: key.NewBinding(
			key.WithKeys("alt+1"),
			key.WithHelp("alt+1", "sidebar"),
		),
		FocusEditor: key.NewBinding(
			key.WithKeys("alt+2"),
			key.WithHelp("alt+2", "editor"),
		),
		FocusResults: key.NewBinding(
			key.WithKeys("alt+3"),
			key.WithHelp("alt+3", "results"),
		),
		NewTab: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "new tab"),
		),
		CloseTab: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "close tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("ctrl+pgdown", "ctrl+]"),
			key.WithHelp("ctrl+pgdn", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("ctrl+pgup", "ctrl+["),
			key.WithHelp("ctrl+pgup", "prev tab"),
		),
		ExecuteQuery: key.NewBinding(
			key.WithKeys("ctrl+enter", "f5", "ctrl+g"),
			key.WithHelp("ctrl+enter", "run query"),
		),
		CancelQuery: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "cancel query"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl+q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("f1", "help"),
		),
		ToggleKeyMode: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("f2", "vim/standard"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "toggle sidebar"),
		),
		RefreshSchema: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh schema"),
		),
		OpenConnMgr: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "connections"),
		),
		History: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "history"),
		),
		Export: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "export"),
		),
		ResizeLeft: key.NewBinding(
			key.WithKeys("ctrl+left"),
			key.WithHelp("ctrl+←", "shrink left"),
		),
		ResizeRight: key.NewBinding(
			key.WithKeys("ctrl+right"),
			key.WithHelp("ctrl+→", "grow right"),
		),
		ResizeUp: key.NewBinding(
			key.WithKeys("ctrl+up"),
			key.WithHelp("ctrl+↑", "shrink up"),
		),
		ResizeDown: key.NewBinding(
			key.WithKeys("ctrl+down"),
			key.WithHelp("ctrl+↓", "grow down"),
		),
	}
}

// VimKeyMap returns keybindings for vim mode.
func VimKeyMap() KeyMap {
	km := StandardKeyMap()

	km.VimUp = key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "up"),
	)
	km.VimDown = key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "down"),
	)
	km.VimLeft = key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "left"),
	)
	km.VimRight = key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "right"),
	)
	km.VimInsert = key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "insert"),
	)
	km.VimAppend = key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "append"),
	)
	km.VimEscape = key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "normal mode"),
	)
	km.VimTop = key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("gg", "top"),
	)
	km.VimBottom = key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom"),
	)
	km.VimSearch = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	)
	km.VimVisual = key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "visual"),
	)
	km.VimYank = key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yank"),
	)

	return km
}

// ShortHelp returns a subset of keybindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.ExecuteQuery, k.FocusNext, k.NewTab, k.Quit, k.Help,
	}
}

// FullHelp returns all keybindings grouped for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.ExecuteQuery, k.CancelQuery, k.Export},
		{k.FocusNext, k.FocusPrev, k.FocusSidebar, k.FocusEditor, k.FocusResults},
		{k.NewTab, k.CloseTab, k.NextTab, k.PrevTab},
		{k.ToggleKeyMode, k.ToggleSidebar, k.RefreshSchema, k.OpenConnMgr, k.History},
		{k.ResizeLeft, k.ResizeRight, k.ResizeUp, k.ResizeDown},
		{k.Quit, k.Help},
	}
}
