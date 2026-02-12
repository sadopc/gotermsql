package editor

import (
	"strings"
	"testing"

	"github.com/sadopc/gotermsql/internal/theme"
)

// NOTE: lipgloss renders styles as no-ops when there is no TTY (such as in a
// test environment), so we cannot verify ANSI escape codes in the output.
// Instead, these tests verify:
// - The highlighter does not panic on various inputs
// - Content (identifiers, keywords, values) is preserved in the output
// - Structural properties (newlines, emptiness) are maintained
// - Nil theme handling works correctly

// ---------------------------------------------------------------------------
// TestNewHighlighter
// ---------------------------------------------------------------------------

func TestNewHighlighter(t *testing.T) {
	h := NewHighlighter()
	if h == nil {
		t.Fatal("NewHighlighter() returned nil")
	}
	if h.lexer == nil {
		t.Fatal("NewHighlighter() lexer is nil")
	}
}

// ---------------------------------------------------------------------------
// TestHighlight
// ---------------------------------------------------------------------------

func TestHighlight(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	sql := "SELECT id, name FROM users WHERE id = 1"
	result := h.Highlight(sql, th)

	// The highlighted output should not be empty.
	if result == "" {
		t.Fatal("Highlight() returned empty string")
	}

	// The output should contain the semantic content (keywords, identifiers, etc.).
	if !strings.Contains(result, "SELECT") {
		t.Error("highlighted output missing 'SELECT' keyword")
	}
	if !strings.Contains(result, "FROM") {
		t.Error("highlighted output missing 'FROM' keyword")
	}
	if !strings.Contains(result, "users") {
		t.Error("highlighted output missing 'users' identifier")
	}
	if !strings.Contains(result, "id") {
		t.Error("highlighted output missing 'id' identifier")
	}
	if !strings.Contains(result, "1") {
		t.Error("highlighted output missing '1' literal")
	}
}

func TestHighlight_NilTheme(t *testing.T) {
	h := NewHighlighter()

	sql := "SELECT 1"
	result := h.Highlight(sql, nil)

	// With nil theme, the function should return the raw SQL unchanged.
	if result != sql {
		t.Errorf("Highlight(sql, nil) = %q, want %q", result, sql)
	}
}

func TestHighlight_NonNilResult(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	inputs := []string{
		"SELECT 1",
		"SELECT * FROM users",
		"INSERT INTO t VALUES (1, 'test')",
		"UPDATE t SET x = 1",
		"DELETE FROM t WHERE id = 1",
		"CREATE TABLE t (id INT)",
	}

	for _, sql := range inputs {
		result := h.Highlight(sql, th)
		if result == "" {
			t.Errorf("Highlight(%q) returned empty string", sql)
		}
	}
}

// ---------------------------------------------------------------------------
// TestHighlight_EmptyString
// ---------------------------------------------------------------------------

func TestHighlight_EmptyString(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	result := h.Highlight("", th)

	// Empty input should produce an empty (or near-empty) result.
	if strings.TrimSpace(result) != "" {
		t.Errorf("Highlight(\"\") = %q, want empty or whitespace-only", result)
	}
}

// ---------------------------------------------------------------------------
// TestHighlight_MultiLine
// ---------------------------------------------------------------------------

func TestHighlight_MultiLine(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	sql := "SELECT id,\n       name\nFROM users\nWHERE active = true"
	result := h.Highlight(sql, th)

	if result == "" {
		t.Fatal("Highlight() returned empty string for multi-line SQL")
	}

	// The output should preserve newlines.
	if !strings.Contains(result, "\n") {
		t.Error("Highlight() multi-line output should contain newlines")
	}

	// Count newlines: original has 3 newlines, output should have at least 3.
	inputNewlines := strings.Count(sql, "\n")
	outputNewlines := strings.Count(result, "\n")
	if outputNewlines < inputNewlines {
		t.Errorf("output has %d newlines, want at least %d", outputNewlines, inputNewlines)
	}

	// Verify key content is present.
	if !strings.Contains(result, "SELECT") {
		t.Error("multi-line output missing SELECT")
	}
	if !strings.Contains(result, "FROM") {
		t.Error("multi-line output missing FROM")
	}
	if !strings.Contains(result, "WHERE") {
		t.Error("multi-line output missing WHERE")
	}
}

// ---------------------------------------------------------------------------
// TestHighlight_Comments
// ---------------------------------------------------------------------------

func TestHighlight_Comments_SingleLine(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	sql := "-- This is a comment\nSELECT 1"
	result := h.Highlight(sql, th)

	if result == "" {
		t.Fatal("Highlight() returned empty string for SQL with single-line comment")
	}

	// Content should still be present.
	if !strings.Contains(result, "This is a comment") {
		t.Error("highlighted output missing comment text")
	}
	if !strings.Contains(result, "SELECT") {
		t.Error("highlighted output missing SELECT keyword")
	}

	// Newline should be preserved.
	if !strings.Contains(result, "\n") {
		t.Error("highlighted output should contain newline separating comment from query")
	}
}

func TestHighlight_Comments_MultiLine(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	sql := "/* multi\n   line\n   comment */\nSELECT 1"
	result := h.Highlight(sql, th)

	if result == "" {
		t.Fatal("Highlight() returned empty string for SQL with multi-line comment")
	}

	// Verify content preservation.
	if !strings.Contains(result, "multi") {
		t.Error("highlighted output missing 'multi' from block comment")
	}
	if !strings.Contains(result, "comment") {
		t.Error("highlighted output missing 'comment' from block comment")
	}

	// Newlines should be preserved.
	inputNewlines := strings.Count(sql, "\n")
	outputNewlines := strings.Count(result, "\n")
	if outputNewlines < inputNewlines {
		t.Errorf("output has %d newlines, want at least %d", outputNewlines, inputNewlines)
	}
}

// ---------------------------------------------------------------------------
// TestHighlight_ContentPreservation
// ---------------------------------------------------------------------------

func TestHighlight_ContentPreservation(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	tests := []struct {
		name     string
		sql      string
		contains []string
	}{
		{
			name: "keywords",
			sql:  "SELECT FROM WHERE INSERT UPDATE DELETE",
			contains: []string{
				"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE",
			},
		},
		{
			name:     "string literal",
			sql:      "SELECT * FROM users WHERE name = 'Alice'",
			contains: []string{"Alice", "users", "name"},
		},
		{
			name:     "number literal",
			sql:      "SELECT * FROM users WHERE id = 42",
			contains: []string{"42", "users", "id"},
		},
		{
			name:     "operators",
			sql:      "SELECT a + b, c - d FROM t WHERE x > 0 AND y < 10",
			contains: []string{"a", "b", "c", "d", "t", "x", "y", "0", "10"},
		},
		{
			name:     "mixed case",
			sql:      "select ID from Users where Active = TRUE",
			contains: []string{"select", "ID", "from", "Users", "where", "Active", "TRUE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Highlight(tt.sql, th)
			if result == "" {
				t.Fatal("Highlight() returned empty string")
			}
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("output missing %q", expected)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHighlight_WhitespaceOnly
// ---------------------------------------------------------------------------

func TestHighlight_WhitespaceOnly(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	// Should not panic on whitespace-only input.
	result := h.Highlight("   \n\t  ", th)
	_ = result
}

// ---------------------------------------------------------------------------
// TestHighlight_Tokenization
// ---------------------------------------------------------------------------

func TestHighlight_Tokenization(t *testing.T) {
	h := NewHighlighter()

	// Verify the lexer can tokenise SQL without errors by checking the
	// Highlight method completes without panicking for various SQL patterns.
	th := theme.Default()

	sqls := []string{
		"SELECT 1",
		"SELECT 'hello world'",
		"SELECT 1 + 2",
		"SELECT * FROM t1 JOIN t2 ON t1.id = t2.fk",
		"INSERT INTO users (name) VALUES ('test')",
		"UPDATE users SET name = 'new' WHERE id = 1",
		"DELETE FROM users WHERE id = 1",
		"CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT NOT NULL)",
		"DROP TABLE IF EXISTS t",
		"ALTER TABLE t ADD COLUMN email TEXT",
		"-- comment only",
		"/* block comment only */",
		"SELECT /* inline */ 1",
		"EXPLAIN SELECT 1",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"SELECT COUNT(*), AVG(price) FROM products GROUP BY category HAVING COUNT(*) > 5",
	}

	for _, sql := range sqls {
		result := h.Highlight(sql, th)
		if result == "" && sql != "" {
			t.Errorf("Highlight(%q) returned empty string", sql)
		}
	}
}

// ---------------------------------------------------------------------------
// Test helper functions from highlight.go
// ---------------------------------------------------------------------------

func TestIsKeyword(t *testing.T) {
	// isKeyword is an unexported function that checks chroma token types.
	// We test it indirectly through Highlight by ensuring keyword content
	// is preserved.
	h := NewHighlighter()
	th := theme.Default()

	result := h.Highlight("SELECT FROM WHERE", th)
	if !strings.Contains(result, "SELECT") || !strings.Contains(result, "FROM") || !strings.Contains(result, "WHERE") {
		t.Error("keywords not preserved in output")
	}
}

func TestIsString(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	result := h.Highlight("'hello world'", th)
	if !strings.Contains(result, "hello world") {
		t.Error("string literal content not preserved")
	}
}

func TestIsNumber(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	result := h.Highlight("SELECT 123, 45.67", th)
	if !strings.Contains(result, "123") {
		t.Error("integer literal not preserved")
	}
	if !strings.Contains(result, "45") {
		t.Error("float literal not preserved")
	}
}

func TestIsComment(t *testing.T) {
	h := NewHighlighter()
	th := theme.Default()

	result := h.Highlight("-- a comment", th)
	if !strings.Contains(result, "a comment") {
		t.Error("single-line comment content not preserved")
	}

	result = h.Highlight("/* block */", th)
	if !strings.Contains(result, "block") {
		t.Error("block comment content not preserved")
	}
}
