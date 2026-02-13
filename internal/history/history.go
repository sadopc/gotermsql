package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sadopc/gotermsql/internal/config"
)

const createTableSQL = `CREATE TABLE IF NOT EXISTS history (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	query        TEXT NOT NULL,
	adapter      TEXT,
	database_name TEXT,
	executed_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
	duration_ms  INTEGER,
	row_count    INTEGER,
	is_error     BOOLEAN DEFAULT FALSE
)`

// HistoryEntry represents a single executed query in the history log.
type HistoryEntry struct {
	ID           int64
	Query        string
	Adapter      string
	DatabaseName string
	ExecutedAt   time.Time
	DurationMS   int64
	RowCount     int64
	IsError      bool
}

// History provides SQLite-backed query history storage.
type History struct {
	db *sql.DB
}

// New opens (or creates) the history database at ConfigDir()/history.db and
// ensures the schema exists.
func New() (*History, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, fmt.Errorf("history: config dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("history: create dir: %w", err)
	}

	dbPath := filepath.Join(dir, "history.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("history: open db: %w", err)
	}

	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("history: create table: %w", err)
	}

	return &History{db: db}, nil
}

// Add inserts a new history entry.
func (h *History) Add(entry HistoryEntry) error {
	_, err := h.db.Exec(
		`INSERT INTO history (query, adapter, database_name, executed_at, duration_ms, row_count, is_error)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.Query,
		entry.Adapter,
		entry.DatabaseName,
		entry.ExecutedAt,
		entry.DurationMS,
		entry.RowCount,
		entry.IsError,
	)
	if err != nil {
		return fmt.Errorf("history add: %w", err)
	}
	return nil
}

// Search returns history entries whose query text matches the given pattern
// using SQL LIKE. Results are ordered by most recent first, limited to limit
// rows.
func (h *History) Search(pattern string, limit int) ([]HistoryEntry, error) {
	rows, err := h.db.Query(
		`SELECT id, query, adapter, database_name, executed_at, duration_ms, row_count, is_error
		 FROM history
		 WHERE query LIKE ?
		 ORDER BY executed_at DESC
		 LIMIT ?`,
		pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("history search: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Recent returns the most recent history entries, limited to limit rows.
func (h *History) Recent(limit int) ([]HistoryEntry, error) {
	rows, err := h.db.Query(
		`SELECT id, query, adapter, database_name, executed_at, duration_ms, row_count, is_error
		 FROM history
		 ORDER BY executed_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("history recent: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Clear deletes all history entries.
func (h *History) Clear() error {
	if _, err := h.db.Exec(`DELETE FROM history`); err != nil {
		return fmt.Errorf("history clear: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (h *History) Close() error {
	return h.db.Close()
}

// scanEntries reads all rows from the result set into a slice of HistoryEntry.
func scanEntries(rows *sql.Rows) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(
			&e.ID,
			&e.Query,
			&e.Adapter,
			&e.DatabaseName,
			&e.ExecutedAt,
			&e.DurationMS,
			&e.RowCount,
			&e.IsError,
		); err != nil {
			return nil, fmt.Errorf("history scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history rows: %w", err)
	}
	return entries, nil
}
