package theme

import (
	"testing"
)

func TestThemes_AllRegistered(t *testing.T) {
	expected := []string{"default", "light", "monokai"}
	for _, name := range expected {
		if _, ok := Themes[name]; !ok {
			t.Errorf("expected theme %q to be registered", name)
		}
	}
}

func TestThemes_NamesMatch(t *testing.T) {
	for name, th := range Themes {
		if th.Name != name {
			t.Errorf("theme registered as %q has Name=%q", name, th.Name)
		}
	}
}

func TestDefault(t *testing.T) {
	d := Default()
	if d == nil {
		t.Fatal("Default() returned nil")
	}
	if d.Name != "default" {
		t.Errorf("Default().Name = %q, want %q", d.Name, "default")
	}
}

func TestGet_ExistingTheme(t *testing.T) {
	tests := []string{"default", "light", "monokai"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			th := Get(name)
			if th == nil {
				t.Fatalf("Get(%q) returned nil", name)
			}
			if th.Name != name {
				t.Errorf("Get(%q).Name = %q", name, th.Name)
			}
		})
	}
}

func TestGet_UnknownTheme_FallsBackToDefault(t *testing.T) {
	th := Get("nonexistent")
	if th == nil {
		t.Fatal("Get(nonexistent) returned nil")
	}
	if th.Name != "default" {
		t.Errorf("Get(nonexistent).Name = %q, want %q", th.Name, "default")
	}
}

func TestGet_EmptyString_FallsBackToDefault(t *testing.T) {
	th := Get("")
	if th == nil {
		t.Fatal("Get(\"\") returned nil")
	}
	if th.Name != "default" {
		t.Errorf("Get(\"\").Name = %q, want %q", th.Name, "default")
	}
}

func TestCurrent_InitialValue(t *testing.T) {
	if Current == nil {
		t.Fatal("Current is nil at init")
	}
	if Current.Name != "default" {
		t.Errorf("Current.Name = %q, want %q", Current.Name, "default")
	}
}

func TestCurrent_CanBeSwapped(t *testing.T) {
	original := Current
	defer func() { Current = original }()

	Current = Themes["monokai"]
	if Current.Name != "monokai" {
		t.Errorf("Current.Name = %q after swap, want %q", Current.Name, "monokai")
	}
}

func TestDefaultTheme_SQLStyles(t *testing.T) {
	d := Default()
	// Verify that SQL styles render non-empty output (styles are properly initialised).
	tests := []struct {
		name  string
		style func() string
	}{
		{"SQLKeyword", func() string { return d.SQLKeyword.Render("SELECT") }},
		{"SQLString", func() string { return d.SQLString.Render("'hello'") }},
		{"SQLNumber", func() string { return d.SQLNumber.Render("42") }},
		{"SQLComment", func() string { return d.SQLComment.Render("-- note") }},
		{"SQLOperator", func() string { return d.SQLOperator.Render("=") }},
		{"SQLFunction", func() string { return d.SQLFunction.Render("COUNT") }},
		{"SQLType", func() string { return d.SQLType.Render("INTEGER") }},
		{"SQLIdentifier", func() string { return d.SQLIdentifier.Render("users") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.style()
			if out == "" {
				t.Errorf("%s rendered empty string", tt.name)
			}
		})
	}
}

func TestLightTheme_SQLStyles(t *testing.T) {
	th := Themes["light"]
	out := th.SQLKeyword.Render("SELECT")
	if out == "" {
		t.Error("light theme SQLKeyword rendered empty string")
	}
}

func TestMonokaiTheme_SQLStyles(t *testing.T) {
	th := Themes["monokai"]
	out := th.SQLKeyword.Render("SELECT")
	if out == "" {
		t.Error("monokai theme SQLKeyword rendered empty string")
	}
}

func TestTheme_UIStyles_NotZeroValue(t *testing.T) {
	for name, th := range Themes {
		t.Run(name, func(t *testing.T) {
			// Verify key styles produce non-empty output.
			pairs := []struct {
				label string
				out   string
			}{
				{"TabActive", th.TabActive.Render("tab")},
				{"TabInactive", th.TabInactive.Render("tab")},
				{"StatusBar", th.StatusBar.Render("status")},
				{"StatusBarError", th.StatusBarError.Render("err")},
				{"DialogBorder", th.DialogBorder.Render("dlg")},
				{"FocusedBorder", th.FocusedBorder.Render("focused")},
				{"UnfocusedBorder", th.UnfocusedBorder.Render("unfocused")},
				{"ErrorText", th.ErrorText.Render("error")},
				{"SuccessText", th.SuccessText.Render("ok")},
				{"MutedText", th.MutedText.Render("muted")},
				{"SidebarSelected", th.SidebarSelected.Render("sel")},
				{"AutocompleteItem", th.AutocompleteItem.Render("item")},
				{"AutocompleteSelected", th.AutocompleteSelected.Render("sel")},
				{"ResultsHeader", th.ResultsHeader.Render("hdr")},
			}
			for _, p := range pairs {
				if p.out == "" {
					t.Errorf("%s: %s rendered empty", name, p.label)
				}
			}
		})
	}
}

func TestThemes_AreDistinct(t *testing.T) {
	d := Themes["default"]
	l := Themes["light"]
	m := Themes["monokai"]

	// Themes should be different objects.
	if d == l {
		t.Error("default and light are the same pointer")
	}
	if d == m {
		t.Error("default and monokai are the same pointer")
	}
	if l == m {
		t.Error("light and monokai are the same pointer")
	}
}
