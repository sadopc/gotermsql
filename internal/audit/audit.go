package audit

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Entry is a single audit log record.
type Entry struct {
	Timestamp    time.Time `json:"timestamp"`
	Query        string    `json:"query"`
	Adapter      string    `json:"adapter"`
	DatabaseName string    `json:"database_name"`
	DurationMS   int64     `json:"duration_ms"`
	RowCount     int64     `json:"row_count"`
	IsError      bool      `json:"is_error"`
	DSN          string    `json:"dsn"`
}

// Logger writes JSON Lines audit entries to a file.
type Logger struct {
	mu        sync.Mutex
	f         *os.File
	enc       *json.Encoder
	path      string
	maxSizeMB int
}

// New creates an audit Logger. It creates parent directories (0o700) and opens
// the file in append mode (0o600). If maxSizeMB > 0, the file is rotated when
// it exceeds that size.
func New(path string, maxSizeMB int) (*Logger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("audit: create dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("audit: open file: %w", err)
	}

	return &Logger{
		f:         f,
		enc:       json.NewEncoder(f),
		path:      path,
		maxSizeMB: maxSizeMB,
	}, nil
}

// Log writes an entry as a JSON line. It is safe for concurrent use.
// Calling Log on a nil Logger is a no-op.
func (l *Logger) Log(e Entry) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	_ = l.enc.Encode(e)

	if l.maxSizeMB > 0 {
		l.rotateIfNeeded()
	}
}

// Close closes the underlying file. Calling Close on a nil Logger is a no-op.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.f.Close()
}

func (l *Logger) rotateIfNeeded() {
	info, err := l.f.Stat()
	if err != nil {
		return
	}
	if info.Size() < int64(l.maxSizeMB)*1024*1024 {
		return
	}
	l.rotate()
}

func (l *Logger) rotate() {
	_ = l.f.Close()
	_ = os.Rename(l.path, l.path+".1")

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	l.f = f
	l.enc = json.NewEncoder(f)
}

// SanitizeDSN strips credentials from a DSN string.
func SanitizeDSN(dsn string) string {
	// URL-style DSNs: postgres://user:pass@host → postgres://***@host
	for _, prefix := range []string{"postgres://", "postgresql://", "mysql://", "duckdb://"} {
		if strings.HasPrefix(strings.ToLower(dsn), prefix) {
			u, err := url.Parse(dsn)
			if err != nil {
				return dsn
			}
			if u.User != nil {
				u.User = url.User("***")
			}
			return u.String()
		}
	}
	// MySQL driver format: user:pass@tcp( → ***@tcp(
	dsn = reMySQLCreds.ReplaceAllString(dsn, "***@tcp(")
	// PostgreSQL keyword format: password=xxx
	dsn = rePGPassword.ReplaceAllString(dsn, "password=***")
	return dsn
}

var (
	reMySQLCreds = regexp.MustCompile(`[^@]+@tcp\(`)
	rePGPassword = regexp.MustCompile(`password=[^\s]+`)
)
