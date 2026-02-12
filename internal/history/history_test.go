package history

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newTestHistory creates a History backed by a SQLite database in the given
// directory. It avoids the real ConfigDir() by opening the DB directly.
func newTestHistory(t *testing.T, dir string) *History {
	t.Helper()

	dbPath := filepath.Join(dir, "history.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		t.Fatalf("create table: %v", err)
	}

	return &History{db: db}
}

func TestNew(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	h, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer h.Close()

	// Verify that the history.db file was created inside the config dir.
	// On macOS: ~/Library/Application Support/gotermsql/history.db
	// On Linux (XDG): ~/.config/gotermsql/history.db
	// Either way, the DB connection should work.
	entries, err := h.Recent(10)
	if err != nil {
		t.Fatalf("Recent() on new DB error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Recent() on new DB = %d entries, want 0", len(entries))
	}
}

func TestAddAndRecent(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := range 5 {
		err := h.Add(HistoryEntry{
			Query:        "SELECT " + string(rune('A'+i)),
			Adapter:      "postgres",
			DatabaseName: "testdb",
			ExecutedAt:   base.Add(time.Duration(i) * time.Minute),
			DurationMS:   int64(10 * (i + 1)),
			RowCount:     int64(i + 1),
			IsError:      false,
		})
		if err != nil {
			t.Fatalf("Add() entry %d error = %v", i, err)
		}
	}

	entries, err := h.Recent(3)
	if err != nil {
		t.Fatalf("Recent(3) error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Recent(3) returned %d entries, want 3", len(entries))
	}

	// Most recent first: E, D, C
	wantQueries := []string{"SELECT E", "SELECT D", "SELECT C"}
	for i, want := range wantQueries {
		if entries[i].Query != want {
			t.Errorf("entries[%d].Query = %q, want %q", i, entries[i].Query, want)
		}
	}
}

func TestAddAndSearch(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	now := time.Now().UTC()
	queries := []string{
		"SELECT * FROM users",
		"INSERT INTO users (name) VALUES ('alice')",
		"SELECT * FROM orders",
		"UPDATE users SET name='bob'",
		"SELECT count(*) FROM users",
	}

	for i, q := range queries {
		err := h.Add(HistoryEntry{
			Query:        q,
			Adapter:      "postgres",
			DatabaseName: "testdb",
			ExecutedAt:   now.Add(time.Duration(i) * time.Second),
			DurationMS:   5,
			RowCount:     1,
		})
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Search for entries containing "users"
	entries, err := h.Search("%users%", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("Search(%%users%%) returned %d entries, want 4", len(entries))
	}

	// Results should be most recent first
	if entries[0].Query != "SELECT count(*) FROM users" {
		t.Errorf("entries[0].Query = %q, want %q", entries[0].Query, "SELECT count(*) FROM users")
	}
}

func TestSearchNoMatches(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	err := h.Add(HistoryEntry{
		Query:      "SELECT 1",
		Adapter:    "sqlite",
		ExecutedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	entries, err := h.Search("%nonexistent_pattern%", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Search(no match) returned %d entries, want 0", len(entries))
	}
}

func TestSearchLIKEPattern(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	now := time.Now().UTC()
	entries := []string{
		"CREATE TABLE products (id INT)",
		"DROP TABLE products",
		"ALTER TABLE users ADD COLUMN email TEXT",
		"SELECT * FROM products",
	}
	for i, q := range entries {
		err := h.Add(HistoryEntry{
			Query:      q,
			Adapter:    "postgres",
			ExecutedAt: now.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	tests := []struct {
		name    string
		pattern string
		want    int
	}{
		{"match TABLE", "%TABLE%", 3},
		{"match products", "%products%", 3},
		{"match SELECT", "SELECT%", 1},
		{"match CREATE", "CREATE%", 1},
		{"no match", "%TRUNCATE%", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := h.Search(tt.pattern, 100)
			if err != nil {
				t.Fatalf("Search(%q) error = %v", tt.pattern, err)
			}
			if len(results) != tt.want {
				t.Errorf("Search(%q) returned %d entries, want %d", tt.pattern, len(results), tt.want)
			}
		})
	}
}

func TestRecentEmptyHistory(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	entries, err := h.Recent(10)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if entries != nil {
		// The implementation returns nil when there are no rows, but we also
		// accept an empty slice.
		if len(entries) != 0 {
			t.Errorf("Recent() on empty DB returned %d entries, want 0", len(entries))
		}
	}
}

func TestRecentWithLimit(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	for i := range 10 {
		err := h.Add(HistoryEntry{
			Query:      "SELECT " + string(rune('0'+i)),
			Adapter:    "postgres",
			ExecutedAt: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("Add() entry %d error = %v", i, err)
		}
	}

	entries, err := h.Recent(5)
	if err != nil {
		t.Fatalf("Recent(5) error = %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("Recent(5) returned %d entries, want 5", len(entries))
	}

	// All 10 entries
	all, err := h.Recent(100)
	if err != nil {
		t.Fatalf("Recent(100) error = %v", err)
	}
	if len(all) != 10 {
		t.Errorf("Recent(100) returned %d entries, want 10", len(all))
	}
}

func TestClear(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	for i := range 3 {
		err := h.Add(HistoryEntry{
			Query:      "SELECT " + string(rune('A'+i)),
			Adapter:    "sqlite",
			ExecutedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Verify entries exist
	before, err := h.Recent(10)
	if err != nil {
		t.Fatalf("Recent() before clear error = %v", err)
	}
	if len(before) != 3 {
		t.Fatalf("Recent() before clear = %d, want 3", len(before))
	}

	if err := h.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	after, err := h.Recent(10)
	if err != nil {
		t.Fatalf("Recent() after clear error = %v", err)
	}
	if len(after) != 0 {
		t.Errorf("Recent() after clear = %d entries, want 0", len(after))
	}
}

func TestHistoryEntryFields(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	execAt := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	entry := HistoryEntry{
		Query:        "SELECT * FROM big_table WHERE id > 1000",
		Adapter:      "postgres",
		DatabaseName: "analytics",
		ExecutedAt:   execAt,
		DurationMS:   1234,
		RowCount:     5678,
		IsError:      false,
	}

	if err := h.Add(entry); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	entries, err := h.Recent(1)
	if err != nil {
		t.Fatalf("Recent(1) error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Recent(1) returned %d entries, want 1", len(entries))
	}

	got := entries[0]
	if got.ID == 0 {
		t.Error("ID should be non-zero after insert")
	}
	if got.Query != entry.Query {
		t.Errorf("Query = %q, want %q", got.Query, entry.Query)
	}
	if got.Adapter != entry.Adapter {
		t.Errorf("Adapter = %q, want %q", got.Adapter, entry.Adapter)
	}
	if got.DatabaseName != entry.DatabaseName {
		t.Errorf("DatabaseName = %q, want %q", got.DatabaseName, entry.DatabaseName)
	}
	if got.DurationMS != entry.DurationMS {
		t.Errorf("DurationMS = %d, want %d", got.DurationMS, entry.DurationMS)
	}
	if got.RowCount != entry.RowCount {
		t.Errorf("RowCount = %d, want %d", got.RowCount, entry.RowCount)
	}
	if got.IsError != entry.IsError {
		t.Errorf("IsError = %v, want %v", got.IsError, entry.IsError)
	}
	// Check that ExecutedAt is approximately correct (SQLite may lose sub-second precision)
	if got.ExecutedAt.Sub(execAt).Abs() > time.Second {
		t.Errorf("ExecutedAt = %v, want approximately %v", got.ExecutedAt, execAt)
	}
}

func TestCloseAndReopen(t *testing.T) {
	dir := t.TempDir()

	// First session: add entries
	h1 := newTestHistory(t, dir)
	for i := range 3 {
		err := h1.Add(HistoryEntry{
			Query:      "query_" + string(rune('A'+i)),
			Adapter:    "postgres",
			ExecutedAt: time.Now().UTC().Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}
	if err := h1.Close(); err != nil {
		t.Fatalf("Close() first session error = %v", err)
	}

	// Second session: reopen and verify entries persist
	h2 := newTestHistory(t, dir)
	defer h2.Close()

	entries, err := h2.Recent(10)
	if err != nil {
		t.Fatalf("Recent() after reopen error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Recent() after reopen = %d entries, want 3", len(entries))
	}

	// Verify most recent is first
	if entries[0].Query != "query_C" {
		t.Errorf("entries[0].Query = %q, want %q", entries[0].Query, "query_C")
	}
	if entries[2].Query != "query_A" {
		t.Errorf("entries[2].Query = %q, want %q", entries[2].Query, "query_A")
	}
}

func TestErrorEntries(t *testing.T) {
	h := newTestHistory(t, t.TempDir())
	defer h.Close()

	now := time.Now().UTC()

	// Add a mix of successful and error entries
	entries := []HistoryEntry{
		{Query: "SELECT 1", Adapter: "postgres", ExecutedAt: now, DurationMS: 5, RowCount: 1, IsError: false},
		{Query: "SELECT * FROM nonexistent", Adapter: "postgres", ExecutedAt: now.Add(time.Second), DurationMS: 2, RowCount: 0, IsError: true},
		{Query: "INSERT INTO t VALUES (1)", Adapter: "postgres", ExecutedAt: now.Add(2 * time.Second), DurationMS: 10, RowCount: 1, IsError: false},
		{Query: "DROP TABLE oops", Adapter: "postgres", ExecutedAt: now.Add(3 * time.Second), DurationMS: 1, RowCount: 0, IsError: true},
	}

	for i, e := range entries {
		if err := h.Add(e); err != nil {
			t.Fatalf("Add() entry %d error = %v", i, err)
		}
	}

	results, err := h.Recent(10)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("Recent() returned %d entries, want 4", len(results))
	}

	// Results are most recent first: DROP, INSERT, SELECT nonexistent, SELECT 1
	wantErrors := []bool{true, false, true, false}
	for i, want := range wantErrors {
		if results[i].IsError != want {
			t.Errorf("results[%d].IsError = %v, want %v (query: %s)", i, results[i].IsError, want, results[i].Query)
		}
	}

	// Verify error entries have correct row counts
	if results[0].RowCount != 0 {
		t.Errorf("error entry RowCount = %d, want 0", results[0].RowCount)
	}
}

func TestNewCreatesDBFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	h, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer h.Close()

	// Check that the DB file exists on disk. On macOS ConfigDir returns
	// ~/Library/Application Support/gotermsql, on Linux ~/.config/gotermsql.
	// We check both possible paths.
	candidates := []string{
		filepath.Join(tmpHome, "Library", "Application Support", "gotermsql", "history.db"),
		filepath.Join(tmpHome, ".config", "gotermsql", "history.db"),
	}

	found := false
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("history.db file was not created in any expected config dir location")
	}
}
