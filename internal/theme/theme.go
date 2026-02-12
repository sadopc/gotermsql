// Package theme provides a centralized styling system for the gotermsql
// terminal UI. Every visual element references a lipgloss.Style held in a
// Theme struct so that the entire look-and-feel can be swapped at runtime.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds lipgloss.Style values for every UI element in the application.
type Theme struct {
	Name string

	// App-level
	AppBackground lipgloss.Style

	// Sidebar / Schema browser
	SidebarBorder     lipgloss.Style
	SidebarTitle      lipgloss.Style
	SidebarDatabase   lipgloss.Style
	SidebarSchema     lipgloss.Style
	SidebarTable      lipgloss.Style
	SidebarView       lipgloss.Style
	SidebarColumn     lipgloss.Style
	SidebarColumnType lipgloss.Style
	SidebarSelected   lipgloss.Style

	// Editor
	EditorBorder     lipgloss.Style
	EditorLineNumber lipgloss.Style
	EditorCursor     lipgloss.Style

	// SQL Syntax highlighting
	SQLKeyword    lipgloss.Style
	SQLString     lipgloss.Style
	SQLNumber     lipgloss.Style
	SQLComment    lipgloss.Style
	SQLOperator   lipgloss.Style
	SQLFunction   lipgloss.Style
	SQLType       lipgloss.Style
	SQLIdentifier lipgloss.Style

	// Results table
	ResultsBorder      lipgloss.Style
	ResultsHeader      lipgloss.Style
	ResultsCell        lipgloss.Style
	ResultsSelectedRow lipgloss.Style
	ResultsNull        lipgloss.Style

	// Tab bar
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	TabBar      lipgloss.Style

	// Status bar
	StatusBar        lipgloss.Style
	StatusBarKey     lipgloss.Style
	StatusBarValue   lipgloss.Style
	StatusBarError   lipgloss.Style
	StatusBarSuccess lipgloss.Style

	// Autocomplete
	AutocompleteItem     lipgloss.Style
	AutocompleteSelected lipgloss.Style
	AutocompleteBorder   lipgloss.Style

	// Dialog/Modal
	DialogBorder       lipgloss.Style
	DialogTitle        lipgloss.Style
	DialogButton       lipgloss.Style
	DialogButtonActive lipgloss.Style

	// General
	FocusedBorder   lipgloss.Style
	UnfocusedBorder lipgloss.Style
	ErrorText       lipgloss.Style
	SuccessText     lipgloss.Style
	WarningText     lipgloss.Style
	MutedText       lipgloss.Style
}

// ---------------------------------------------------------------------------
// Theme definitions
// ---------------------------------------------------------------------------

// newDefaultTheme builds the Default dark theme.
func newDefaultTheme() *Theme {
	return &Theme{
		Name: "default",

		// App-level
		AppBackground: lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E1E")),

		// Sidebar
		SidebarBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3C3C3C")),
		SidebarTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#569CD6")).
			PaddingLeft(1),
		SidebarDatabase: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#DCDCAA")),
		SidebarSchema: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CDCFE")),
		SidebarTable: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4EC9B0")),
		SidebarView: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C586C0")),
		SidebarColumn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4D4D4")),
		SidebarColumnType: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		SidebarSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#264F78")),

		// Editor
		EditorBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3C3C3C")),
		EditorLineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#858585")),
		EditorCursor: lipgloss.NewStyle().
			Background(lipgloss.Color("#AEAFAD")),

		// SQL Syntax highlighting
		SQLKeyword: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#569CD6")),
		SQLString: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CE9178")),
		SQLNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B5CEA8")),
		SQLComment: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#6A9955")),
		SQLOperator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4D4D4")),
		SQLFunction: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DCDCAA")),
		SQLType: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4EC9B0")),
		SQLIdentifier: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CDCFE")),

		// Results table
		ResultsBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3C3C3C")),
		ResultsHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#569CD6")).
			Background(lipgloss.Color("#252526")),
		ResultsCell: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4D4D4")),
		ResultsSelectedRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#264F78")),
		ResultsNull: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#808080")),

		// Tab bar
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1E1E1E")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(false).
			BorderForeground(lipgloss.Color("#569CD6")).
			PaddingLeft(1).
			PaddingRight(1),
		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Background(lipgloss.Color("#2D2D2D")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#3C3C3C")).
			PaddingLeft(1).
			PaddingRight(1),
		TabBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#252526")),

		// Status bar
		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#007ACC")),
		StatusBarKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#007ACC")).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBarValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4D4D4")).
			Background(lipgloss.Color("#1E1E1E")).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBarError: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#F44747")),
		StatusBarSuccess: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#6A9955")),

		// Autocomplete
		AutocompleteItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4D4D4")).
			Background(lipgloss.Color("#252526")).
			PaddingLeft(1).
			PaddingRight(1),
		AutocompleteSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#264F78")).
			PaddingLeft(1).
			PaddingRight(1),
		AutocompleteBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#569CD6")),

		// Dialog/Modal
		DialogBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#569CD6")).
			Padding(1, 2),
		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#569CD6")),
		DialogButton: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D4D4D4")).
			Background(lipgloss.Color("#3C3C3C")).
			PaddingLeft(2).
			PaddingRight(2),
		DialogButtonActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#007ACC")).
			PaddingLeft(2).
			PaddingRight(2),

		// General
		FocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#569CD6")),
		UnfocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3C3C3C")),
		ErrorText: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F44747")),
		SuccessText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6A9955")),
		WarningText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCA700")),
		MutedText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
	}
}

// newLightTheme builds the Light theme suitable for light terminal backgrounds.
func newLightTheme() *Theme {
	return &Theme{
		Name: "light",

		// App-level
		AppBackground: lipgloss.NewStyle().
			Background(lipgloss.Color("#FFFFFF")),

		// Sidebar
		SidebarBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#D4D4D4")),
		SidebarTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0451A5")).
			PaddingLeft(1),
		SidebarDatabase: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#795E26")),
		SidebarSchema: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#001080")),
		SidebarTable: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#267F99")),
		SidebarView: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AF00DB")),
		SidebarColumn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E1E")),
		SidebarColumnType: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Italic(true),
		SidebarSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0060C0")),

		// Editor
		EditorBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#D4D4D4")),
		EditorLineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#237893")),
		EditorCursor: lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")),

		// SQL Syntax highlighting
		SQLKeyword: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0000FF")),
		SQLString: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A31515")),
		SQLNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#098658")),
		SQLComment: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#008000")),
		SQLOperator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E1E")),
		SQLFunction: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#795E26")),
		SQLType: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#267F99")),
		SQLIdentifier: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#001080")),

		// Results table
		ResultsBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#D4D4D4")),
		ResultsHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0451A5")).
			Background(lipgloss.Color("#F3F3F3")),
		ResultsCell: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E1E")),
		ResultsSelectedRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0060C0")),
		ResultsNull: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#A0A0A0")),

		// Tab bar
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1E1E1E")).
			Background(lipgloss.Color("#FFFFFF")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(false).
			BorderForeground(lipgloss.Color("#0451A5")).
			PaddingLeft(1).
			PaddingRight(1),
		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Background(lipgloss.Color("#ECECEC")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#D4D4D4")).
			PaddingLeft(1).
			PaddingRight(1),
		TabBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#F3F3F3")),

		// Status bar
		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0060C0")),
		StatusBarKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0060C0")).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBarValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E1E")).
			Background(lipgloss.Color("#F3F3F3")).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBarError: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#E51400")),
		StatusBarSuccess: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#16825D")),

		// Autocomplete
		AutocompleteItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E1E")).
			Background(lipgloss.Color("#F3F3F3")).
			PaddingLeft(1).
			PaddingRight(1),
		AutocompleteSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0060C0")).
			PaddingLeft(1).
			PaddingRight(1),
		AutocompleteBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0451A5")),

		// Dialog/Modal
		DialogBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0451A5")).
			Padding(1, 2),
		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0451A5")),
		DialogButton: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1E1E1E")).
			Background(lipgloss.Color("#D4D4D4")).
			PaddingLeft(2).
			PaddingRight(2),
		DialogButtonActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#0060C0")).
			PaddingLeft(2).
			PaddingRight(2),

		// General
		FocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0451A5")),
		UnfocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#D4D4D4")),
		ErrorText: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E51400")),
		SuccessText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#16825D")),
		WarningText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BF8803")),
		MutedText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")),
	}
}

// newMonokaiTheme builds a Monokai-inspired dark theme.
func newMonokaiTheme() *Theme {
	return &Theme{
		Name: "monokai",

		// App-level
		AppBackground: lipgloss.NewStyle().
			Background(lipgloss.Color("#272822")),

		// Sidebar
		SidebarBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#49483E")),
		SidebarTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F92672")).
			PaddingLeft(1),
		SidebarDatabase: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E6DB74")),
		SidebarSchema: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#66D9EF")),
		SidebarTable: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E22E")),
		SidebarView: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AE81FF")),
		SidebarColumn: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")),
		SidebarColumnType: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#75715E")).
			Italic(true),
		SidebarSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#49483E")),

		// Editor
		EditorBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#49483E")),
		EditorLineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#90908A")),
		EditorCursor: lipgloss.NewStyle().
			Background(lipgloss.Color("#F8F8F0")),

		// SQL Syntax highlighting
		SQLKeyword: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F92672")),
		SQLString: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E6DB74")),
		SQLNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AE81FF")),
		SQLComment: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#75715E")),
		SQLOperator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F92672")),
		SQLFunction: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E22E")),
		SQLType: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#66D9EF")).
			Italic(true),
		SQLIdentifier: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")),

		// Results table
		ResultsBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#49483E")),
		ResultsHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A6E22E")).
			Background(lipgloss.Color("#3E3D32")),
		ResultsCell: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")),
		ResultsSelectedRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#49483E")),
		ResultsNull: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#75715E")),

		// Tab bar
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#272822")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(false).
			BorderForeground(lipgloss.Color("#F92672")).
			PaddingLeft(1).
			PaddingRight(1),
		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#75715E")).
			Background(lipgloss.Color("#3E3D32")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#49483E")).
			PaddingLeft(1).
			PaddingRight(1),
		TabBar: lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1F1C")),

		// Status bar
		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#75715E")),
		StatusBarKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#272822")).
			Background(lipgloss.Color("#A6E22E")).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBarValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#3E3D32")).
			PaddingLeft(1).
			PaddingRight(1),
		StatusBarError: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#F92672")),
		StatusBarSuccess: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#272822")).
			Background(lipgloss.Color("#A6E22E")),

		// Autocomplete
		AutocompleteItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#3E3D32")).
			PaddingLeft(1).
			PaddingRight(1),
		AutocompleteSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#49483E")).
			PaddingLeft(1).
			PaddingRight(1),
		AutocompleteBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F92672")),

		// Dialog/Modal
		DialogBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F92672")).
			Padding(1, 2),
		DialogTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F92672")),
		DialogButton: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#49483E")).
			PaddingLeft(2).
			PaddingRight(2),
		DialogButtonActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#272822")).
			Background(lipgloss.Color("#A6E22E")).
			PaddingLeft(2).
			PaddingRight(2),

		// General
		FocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F92672")),
		UnfocusedBorder: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#49483E")),
		ErrorText: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F92672")),
		SuccessText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E22E")),
		WarningText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E6DB74")),
		MutedText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#75715E")),
	}
}

// ---------------------------------------------------------------------------
// Registry and accessors
// ---------------------------------------------------------------------------

// Themes maps theme names to their Theme definitions.
var Themes = map[string]*Theme{
	"default": newDefaultTheme(),
	"light":   newLightTheme(),
	"monokai": newMonokaiTheme(),
}

// Current is the currently active theme. It is initialized to Default.
var Current = Themes["default"]

// Default returns the default dark theme.
func Default() *Theme {
	return Themes["default"]
}

// Get returns the theme identified by name. If no theme with that name exists
// it falls back to the default theme.
func Get(name string) *Theme {
	if t, ok := Themes[name]; ok {
		return t
	}
	return Default()
}
