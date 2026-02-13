package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogWritesJSONLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	l, err := New(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.Log(Entry{
		Timestamp:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Query:        "SELECT 1",
		Adapter:      "sqlite",
		DatabaseName: "test.db",
		DurationMS:   5,
		RowCount:     1,
		IsError:      false,
		DSN:          "test.db",
	})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatalf("invalid JSON line: %v\ndata: %s", err, data)
	}
	if e.Query != "SELECT 1" {
		t.Errorf("query = %q, want %q", e.Query, "SELECT 1")
	}
	if e.Adapter != "sqlite" {
		t.Errorf("adapter = %q, want %q", e.Adapter, "sqlite")
	}
	if e.RowCount != 1 {
		t.Errorf("row_count = %d, want 1", e.RowCount)
	}
}

func TestMultipleEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	l, err := New(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	for i := range 5 {
		l.Log(Entry{
			Timestamp: time.Now(),
			Query:     "SELECT " + string(rune('a'+i)),
			Adapter:   "sqlite",
		})
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("got %d lines, want 5", len(lines))
	}
}

func TestNilReceiver(t *testing.T) {
	var l *Logger
	// Should not panic
	l.Log(Entry{Query: "SELECT 1"})
	if err := l.Close(); err != nil {
		t.Errorf("Close on nil logger returned error: %v", err)
	}
}

func TestRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	// maxSizeMB=0 means we'll set a very small value to trigger rotation
	// Use a 1-byte fake to force rotation (we'll use the internal rotate)
	l, err := New(path, 1) // 1 MB
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Write enough data to exceed 1 MB
	bigQuery := strings.Repeat("x", 10000)
	for range 120 {
		l.Log(Entry{Query: bigQuery, Adapter: "test"})
	}

	// Check that backup file exists
	if _, err := os.Stat(path + ".1"); os.IsNotExist(err) {
		t.Error("rotation backup file does not exist")
	}

	// Current file should be smaller than max
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 1024*1024 {
		t.Errorf("current file size %d exceeds 1 MB after rotation", info.Size())
	}
}

func TestFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")
	l, err := New(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	l.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permissions = %o, want 600", perm)
	}
}

func TestDirectoryCreation(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	path := filepath.Join(nested, "audit.jsonl")
	l, err := New(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	l.Close()

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("nested directory was not created")
	}
}

func TestSanitizeDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "postgres with credentials",
			dsn:  "postgres://admin:s3cret@host:5432/mydb",
			want: "postgres://%2A%2A%2A@host:5432/mydb",
		},
		{
			name: "postgresql scheme",
			dsn:  "postgresql://user:pass@host/db",
			want: "postgresql://%2A%2A%2A@host/db",
		},
		{
			name: "postgres no password",
			dsn:  "postgres://user@host/db",
			want: "postgres://%2A%2A%2A@host/db",
		},
		{
			name: "mysql driver format",
			dsn:  "root:password@tcp(localhost:3306)/mydb",
			want: "***@tcp(localhost:3306)/mydb",
		},
		{
			name: "sqlite file",
			dsn:  "/path/to/data.db",
			want: "/path/to/data.db",
		},
		{
			name: "postgres keyword password",
			dsn:  "host=localhost password=secret dbname=test",
			want: "host=localhost password=*** dbname=test",
		},
		{
			name: "empty dsn",
			dsn:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeDSN(tt.dsn)
			if got != tt.want {
				t.Errorf("SanitizeDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
