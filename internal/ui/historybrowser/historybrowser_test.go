package historybrowser

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gotermsql/internal/history"
)

func TestNilHistory(t *testing.T) {
	m := New(nil)
	m.Show()

	if !m.Visible() {
		t.Fatal("expected visible after Show()")
	}
	if len(m.entries) != 0 {
		t.Fatalf("expected 0 entries with nil history, got %d", len(m.entries))
	}

	// Should not panic on Update
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m.View()
}

func TestSelectQueryMsg(t *testing.T) {
	m := New(nil)
	m.visible = true
	m.entries = []histEntry{
		{Query: "SELECT 1"},
		{Query: "SELECT 2"},
	}
	m.cursor = 1

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.Visible() {
		t.Fatal("expected hidden after enter")
	}
	if cmd == nil {
		t.Fatal("expected cmd from enter")
	}
	msg := cmd()
	sel, ok := msg.(SelectQueryMsg)
	if !ok {
		t.Fatalf("expected SelectQueryMsg, got %T", msg)
	}
	if sel.Query != "SELECT 2" {
		t.Fatalf("expected 'SELECT 2', got %q", sel.Query)
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		offset time.Duration
		want   string
	}{
		{5 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{2 * time.Hour, "2h ago"},
		{36 * time.Hour, "yesterday"},
		{72 * time.Hour, "3d ago"},
	}
	for _, tt := range tests {
		got := RelativeTime(time.Now().Add(-tt.offset))
		if got != tt.want {
			t.Errorf("RelativeTime(-%v) = %q, want %q", tt.offset, got, tt.want)
		}
	}
}

func TestFormatEntryTruncation(t *testing.T) {
	m := New(nil)
	long := "SELECT very_long_column_name_one, very_long_column_name_two, very_long_column_name_three FROM some_extremely_long_table_name WHERE condition = true"
	e := histEntry{
		Query:      long,
		Adapter:    "sqlite",
		DurationMS: 42,
		ExecutedAt: time.Now().Add(-5 * time.Minute),
	}

	result := m.formatEntry(e, 60)
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	// The query part should be truncated
	if !contains(result, "...") && len(long) > 30 {
		t.Error("expected truncation with '...' for long query")
	}
}

func TestHideShow(t *testing.T) {
	m := New(nil)

	if m.Visible() {
		t.Fatal("should not be visible initially")
	}

	m.Show()
	if !m.Visible() {
		t.Fatal("should be visible after Show()")
	}

	m.Hide()
	if m.Visible() {
		t.Fatal("should not be visible after Hide()")
	}
}

func TestEscHides(t *testing.T) {
	m := New(nil)
	m.Show()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.Visible() {
		t.Fatal("esc should hide")
	}
}

// histEntry is a shorthand alias used only in tests to reduce verbosity.
type histEntry = history.HistoryEntry

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
