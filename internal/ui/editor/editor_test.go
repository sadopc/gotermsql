package editor

import (
	"strings"
	"testing"

	"github.com/sadopc/gotermsql/internal/theme"
)

func init() {
	theme.Current = theme.Default()
}

// ---------------------------------------------------------------------------
// TestNew
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	m := New(42)

	if m.ID() != 42 {
		t.Errorf("ID() = %d, want 42", m.ID())
	}
	if m.Value() != "" {
		t.Errorf("Value() = %q, want empty string", m.Value())
	}
	if m.Modified() {
		t.Error("Modified() should be false for a new editor")
	}
	if m.Focused() {
		t.Error("Focused() should be false for a new editor")
	}
}

func TestNew_DifferentIDs(t *testing.T) {
	m0 := New(0)
	m1 := New(1)
	m99 := New(99)

	if m0.ID() != 0 {
		t.Errorf("ID() = %d, want 0", m0.ID())
	}
	if m1.ID() != 1 {
		t.Errorf("ID() = %d, want 1", m1.ID())
	}
	if m99.ID() != 99 {
		t.Errorf("ID() = %d, want 99", m99.ID())
	}
}

// ---------------------------------------------------------------------------
// TestValue_SetValue
// ---------------------------------------------------------------------------

func TestValue_SetValue(t *testing.T) {
	m := New(0)

	m.SetValue("SELECT * FROM users")
	if got := m.Value(); got != "SELECT * FROM users" {
		t.Errorf("Value() = %q, want %q", got, "SELECT * FROM users")
	}
}

func TestValue_SetValue_Empty(t *testing.T) {
	m := New(0)
	m.SetValue("something")
	m.SetValue("")
	if got := m.Value(); got != "" {
		t.Errorf("Value() = %q, want empty string", got)
	}
}

func TestValue_SetValue_MultiLine(t *testing.T) {
	m := New(0)
	query := "SELECT *\nFROM users\nWHERE id > 5"
	m.SetValue(query)
	if got := m.Value(); got != query {
		t.Errorf("Value() = %q, want %q", got, query)
	}
}

func TestValue_SetValue_Overwrite(t *testing.T) {
	m := New(0)
	m.SetValue("first value")
	m.SetValue("second value")
	if got := m.Value(); got != "second value" {
		t.Errorf("Value() = %q, want %q", got, "second value")
	}
}

// ---------------------------------------------------------------------------
// TestModified
// ---------------------------------------------------------------------------

func TestModified_InitiallyFalse(t *testing.T) {
	m := New(0)
	if m.Modified() {
		t.Error("Modified() should be false initially")
	}
}

func TestModified_AfterInsertText(t *testing.T) {
	m := New(0)
	m.InsertText("hello")
	if !m.Modified() {
		t.Error("Modified() should be true after InsertText")
	}
}

func TestModified_InsertTextOnEmpty(t *testing.T) {
	m := New(0)
	m.InsertText("SELECT 1")
	if !m.Modified() {
		t.Error("Modified() should be true after InsertText on empty editor")
	}
	if got := m.Value(); got != "SELECT 1" {
		t.Errorf("Value() = %q, want %q", got, "SELECT 1")
	}
}

func TestModified_InsertTextAppends(t *testing.T) {
	m := New(0)
	m.SetValue("SELECT")
	m.ResetModified()
	m.InsertText("* FROM users")
	if !m.Modified() {
		t.Error("Modified() should be true after InsertText")
	}
	// InsertText adds a space before the text since the last char is not whitespace.
	got := m.Value()
	if !strings.Contains(got, "* FROM users") {
		t.Errorf("Value() = %q, expected it to contain %q", got, "* FROM users")
	}
}

// ---------------------------------------------------------------------------
// TestResetModified
// ---------------------------------------------------------------------------

func TestResetModified(t *testing.T) {
	m := New(0)
	m.InsertText("test")
	if !m.Modified() {
		t.Fatal("expected Modified() = true after InsertText")
	}

	m.ResetModified()
	if m.Modified() {
		t.Error("Modified() should be false after ResetModified()")
	}
}

func TestResetModified_WhenAlreadyFalse(t *testing.T) {
	m := New(0)
	// Should not panic.
	m.ResetModified()
	if m.Modified() {
		t.Error("Modified() should remain false")
	}
}

// ---------------------------------------------------------------------------
// TestID
// ---------------------------------------------------------------------------

func TestID(t *testing.T) {
	tests := []int{0, 1, 5, 100, 9999}
	for _, id := range tests {
		m := New(id)
		if m.ID() != id {
			t.Errorf("New(%d).ID() = %d, want %d", id, m.ID(), id)
		}
	}
}

// ---------------------------------------------------------------------------
// TestFocusBlur
// ---------------------------------------------------------------------------

func TestFocusBlur(t *testing.T) {
	m := New(0)

	// Initially not focused.
	if m.Focused() {
		t.Error("expected Focused() = false initially")
	}

	// Focus.
	m.Focus()
	if !m.Focused() {
		t.Error("expected Focused() = true after Focus()")
	}

	// Blur.
	m.Blur()
	if m.Focused() {
		t.Error("expected Focused() = false after Blur()")
	}
}

func TestFocus_DoubleFocus(t *testing.T) {
	m := New(0)
	m.Focus()
	m.Focus() // Should not panic.
	if !m.Focused() {
		t.Error("expected Focused() = true after double Focus()")
	}
}

func TestBlur_DoubleBlur(t *testing.T) {
	m := New(0)
	m.Blur()
	m.Blur() // Should not panic.
	if m.Focused() {
		t.Error("expected Focused() = false after double Blur()")
	}
}

// ---------------------------------------------------------------------------
// TestSetSize
// ---------------------------------------------------------------------------

func TestSetSize(t *testing.T) {
	m := New(0)

	// Should not panic with various sizes.
	m.SetSize(80, 24)
	m.SetSize(120, 40)
	m.SetSize(1, 1)
	m.SetSize(0, 0) // Edge case: zero dimensions.
}

func TestSetSize_SmallValues(t *testing.T) {
	m := New(0)
	// SetSize with very small values should not panic (internal clamping).
	m.SetSize(2, 2)
	m.SetSize(3, 3)
}

// ---------------------------------------------------------------------------
// TestInit
// ---------------------------------------------------------------------------

func TestInit(t *testing.T) {
	m := New(0)
	cmd := m.Init()
	// Init returns textarea.Blink which is non-nil.
	if cmd == nil {
		t.Error("Init() should return a non-nil command (textarea blink)")
	}
}

// ---------------------------------------------------------------------------
// TestUpdate_NotFocused
// ---------------------------------------------------------------------------

func TestUpdate_NotFocused(t *testing.T) {
	m := New(0)
	// When not focused, Update should return nil cmd.
	m2, cmd := m.Update(nil)
	if cmd != nil {
		t.Error("expected nil cmd when not focused")
	}
	_ = m2
}

// ---------------------------------------------------------------------------
// TestView
// ---------------------------------------------------------------------------

func TestView_NotFocused(t *testing.T) {
	m := New(0)
	m.SetSize(80, 24)

	view := m.View()
	// Should render something even when not focused (placeholder or content).
	if view == "" {
		t.Error("View() should return non-empty string with size set")
	}
}

func TestView_Focused(t *testing.T) {
	m := New(0)
	m.SetSize(80, 24)
	m.Focus()

	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty string when focused")
	}
}

func TestView_WithContent(t *testing.T) {
	m := New(0)
	m.SetSize(80, 24)
	m.SetValue("SELECT * FROM users")

	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty string with content")
	}
}

func TestView_ZeroSize(t *testing.T) {
	m := New(0)
	// With zero size, View should still not panic.
	_ = m.View()
}

// ---------------------------------------------------------------------------
// TestInsertText
// ---------------------------------------------------------------------------

func TestInsertText_OnEmpty(t *testing.T) {
	m := New(0)
	m.InsertText("users")
	if got := m.Value(); got != "users" {
		t.Errorf("Value() = %q, want %q", got, "users")
	}
}

func TestInsertText_Appends(t *testing.T) {
	m := New(0)
	m.SetValue("SELECT * FROM")
	m.InsertText("users")
	got := m.Value()
	// InsertText prepends a space when last char is not whitespace.
	if got != "SELECT * FROM users" {
		t.Errorf("Value() = %q, want %q", got, "SELECT * FROM users")
	}
}

func TestInsertText_AfterNewline(t *testing.T) {
	m := New(0)
	m.SetValue("SELECT *\n")
	m.InsertText("FROM users")
	got := m.Value()
	// Last char is \n which is whitespace, so no extra space added.
	if got != "SELECT *\nFROM users" {
		t.Errorf("Value() = %q, want %q", got, "SELECT *\nFROM users")
	}
}

func TestInsertText_SetsModified(t *testing.T) {
	m := New(0)
	m.InsertText("test")
	if !m.Modified() {
		t.Error("InsertText should set Modified() = true")
	}
}
