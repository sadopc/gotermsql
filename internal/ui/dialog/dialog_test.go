package dialog

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gotermsql/internal/theme"
)

func init() {
	theme.Current = theme.Default()
}

type testActionMsg struct {
	label string
}

func TestNew(t *testing.T) {
	d := New("Title", "Body text",
		Button{Label: "OK"},
		Button{Label: "Cancel"},
	)

	if d.Visible() {
		t.Fatal("expected dialog to be not visible initially")
	}
	if d.title != "Title" {
		t.Fatalf("expected title 'Title', got %q", d.title)
	}
	if d.body != "Body text" {
		t.Fatalf("expected body 'Body text', got %q", d.body)
	}
	if len(d.buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(d.buttons))
	}
	if d.maxWidth != 60 {
		t.Fatalf("expected maxWidth=60, got %d", d.maxWidth)
	}
}

func TestShow(t *testing.T) {
	d := New("Test", "test body", Button{Label: "OK"})

	d.Show()
	if !d.Visible() {
		t.Fatal("expected visible after Show()")
	}
	if d.active != 0 {
		t.Fatalf("expected active=0 after Show(), got %d", d.active)
	}
}

func TestHide(t *testing.T) {
	d := New("Test", "test body", Button{Label: "OK"})
	d.Show()

	d.Hide()
	if d.Visible() {
		t.Fatal("expected not visible after Hide()")
	}
}

func TestUpdate_NotVisible(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})

	// When not visible, key messages should be ignored.
	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd when dialog not visible")
	}
}

func TestUpdate_Navigation(t *testing.T) {
	d := New("Test", "body",
		Button{Label: "Yes"},
		Button{Label: "No"},
		Button{Label: "Cancel"},
	)
	d.Show()

	// Initially active=0.
	if d.active != 0 {
		t.Fatalf("expected active=0, got %d", d.active)
	}

	// Press right to move to next button.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	if d.active != 1 {
		t.Fatalf("expected active=1 after right, got %d", d.active)
	}

	// Press right again.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	if d.active != 2 {
		t.Fatalf("expected active=2 after second right, got %d", d.active)
	}

	// Press right at the end: should stay at 2.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	if d.active != 2 {
		t.Fatalf("expected active=2 at boundary, got %d", d.active)
	}

	// Press left.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if d.active != 1 {
		t.Fatalf("expected active=1 after left, got %d", d.active)
	}

	// Press left again.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if d.active != 0 {
		t.Fatalf("expected active=0 after left, got %d", d.active)
	}

	// Press left at the start: should stay at 0.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if d.active != 0 {
		t.Fatalf("expected active=0 at boundary, got %d", d.active)
	}
}

func TestUpdate_Tab(t *testing.T) {
	d := New("Test", "body",
		Button{Label: "A"},
		Button{Label: "B"},
	)
	d.Show()

	// Tab should move right.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})
	if d.active != 1 {
		t.Fatalf("expected active=1 after tab, got %d", d.active)
	}

	// Shift+tab should move left.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if d.active != 0 {
		t.Fatalf("expected active=0 after shift+tab, got %d", d.active)
	}
}

func TestUpdate_Enter(t *testing.T) {
	var actionCalled bool

	d := New("Confirm", "Are you sure?",
		Button{Label: "Yes", Action: func() tea.Msg {
			actionCalled = true
			return testActionMsg{label: "yes"}
		}},
		Button{Label: "No", Action: func() tea.Msg {
			return testActionMsg{label: "no"}
		}},
	)
	d.Show()

	// Press enter on the first button (active=0).
	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if d.Visible() {
		t.Fatal("expected dialog hidden after enter")
	}
	if cmd == nil {
		t.Fatal("expected cmd from enter")
	}
	msg := cmd()
	actionMsg, ok := msg.(testActionMsg)
	if !ok {
		t.Fatalf("expected testActionMsg, got %T", msg)
	}
	if actionMsg.label != "yes" {
		t.Fatalf("expected action 'yes', got %q", actionMsg.label)
	}
	_ = actionCalled
}

func TestUpdate_Enter_NilAction(t *testing.T) {
	d := New("Test", "body",
		Button{Label: "OK", Action: nil},
	)
	d.Show()

	// Pressing enter with nil action should not panic.
	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd when action is nil")
	}
}

func TestUpdate_Enter_SecondButton(t *testing.T) {
	d := New("Confirm", "body",
		Button{Label: "Yes", Action: func() tea.Msg { return testActionMsg{label: "yes"} }},
		Button{Label: "No", Action: func() tea.Msg { return testActionMsg{label: "no"} }},
	)
	d.Show()

	// Move to second button and press enter.
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd from second button enter")
	}
	msg := cmd()
	actionMsg, ok := msg.(testActionMsg)
	if !ok {
		t.Fatalf("expected testActionMsg, got %T", msg)
	}
	if actionMsg.label != "no" {
		t.Fatalf("expected action 'no', got %q", actionMsg.label)
	}
}

func TestUpdate_Escape(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})
	d.Show()

	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if d.Visible() {
		t.Fatal("expected dialog hidden after escape")
	}
}

func TestView_Hidden(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})

	view := d.View()
	if view != "" {
		t.Fatalf("expected empty view when hidden, got %q", view)
	}
}

func TestView_Visible(t *testing.T) {
	d := New("Confirm Delete", "Are you sure you want to delete this?",
		Button{Label: "Yes"},
		Button{Label: "No"},
	)
	d.Show()

	view := d.View()
	if view == "" {
		t.Fatal("expected non-empty view when visible")
	}
}

func TestOverlay_Hidden(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})

	background := "line1\nline2\nline3"
	result := d.Overlay(background)
	if result != background {
		t.Fatalf("expected overlay to return background unchanged when hidden")
	}
}

func TestOverlay_Visible(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})
	d.SetSize(80, 24)
	d.Show()

	// Create a background with enough lines.
	bg := ""
	for i := 0; i < 24; i++ {
		if i > 0 {
			bg += "\n"
		}
		bg += "                                                                                "
	}

	result := d.Overlay(bg)
	if result == "" {
		t.Fatal("expected non-empty overlay result")
	}
	// The result should differ from the background since dialog is overlaid.
	if result == bg {
		t.Fatal("expected overlay to modify background content")
	}
}

func TestSetSize(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})
	d.SetSize(40, 20)

	if d.width != 40 {
		t.Fatalf("expected width=40, got %d", d.width)
	}
	if d.maxHeight != 20 {
		t.Fatalf("expected maxHeight=20, got %d", d.maxHeight)
	}
	// maxWidth should be clamped if bigger than width-4.
	if d.maxWidth > 40-4 {
		t.Fatalf("expected maxWidth <= 36, got %d", d.maxWidth)
	}
}

func TestSetSize_LargeWidth(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})
	d.SetSize(200, 50)

	// maxWidth default is 60, which is less than 200-4=196, so it stays at 60.
	if d.maxWidth != 60 {
		t.Fatalf("expected maxWidth=60 (unchanged), got %d", d.maxWidth)
	}
}

func TestInit(t *testing.T) {
	d := New("Test", "body", Button{Label: "OK"})
	cmd := d.Init()
	if cmd != nil {
		t.Fatal("expected nil cmd from Init")
	}
}
