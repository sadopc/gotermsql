package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Theme != "default" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "default")
	}
	if cfg.KeyMode != "standard" {
		t.Errorf("KeyMode = %q, want %q", cfg.KeyMode, "standard")
	}
	if cfg.Editor.TabSize != 4 {
		t.Errorf("Editor.TabSize = %d, want %d", cfg.Editor.TabSize, 4)
	}
	if cfg.Editor.ShowLineNumbers != true {
		t.Errorf("Editor.ShowLineNumbers = %v, want %v", cfg.Editor.ShowLineNumbers, true)
	}
	if cfg.Results.PageSize != 1000 {
		t.Errorf("Results.PageSize = %d, want %d", cfg.Results.PageSize, 1000)
	}
	if cfg.Results.MaxColumnWidth != 50 {
		t.Errorf("Results.MaxColumnWidth = %d, want %d", cfg.Results.MaxColumnWidth, 50)
	}
	if len(cfg.Connections) != 0 {
		t.Errorf("Connections length = %d, want 0", len(cfg.Connections))
	}
}

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `theme: monokai
keymode: vim
editor:
  tab_size: 2
  show_line_numbers: false
results:
  page_size: 500
  max_column_width: 80
connections:
  - name: mydb
    adapter: postgres
    host: db.example.com
    port: 5432
    user: admin
    password: secret
    database: production
  - name: localfile
    adapter: sqlite
    file: /tmp/test.db
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Theme != "monokai" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "monokai")
	}
	if cfg.KeyMode != "vim" {
		t.Errorf("KeyMode = %q, want %q", cfg.KeyMode, "vim")
	}
	if cfg.Editor.TabSize != 2 {
		t.Errorf("Editor.TabSize = %d, want %d", cfg.Editor.TabSize, 2)
	}
	if cfg.Editor.ShowLineNumbers != false {
		t.Errorf("Editor.ShowLineNumbers = %v, want false", cfg.Editor.ShowLineNumbers)
	}
	if cfg.Results.PageSize != 500 {
		t.Errorf("Results.PageSize = %d, want %d", cfg.Results.PageSize, 500)
	}
	if cfg.Results.MaxColumnWidth != 80 {
		t.Errorf("Results.MaxColumnWidth = %d, want %d", cfg.Results.MaxColumnWidth, 80)
	}
	if len(cfg.Connections) != 2 {
		t.Fatalf("Connections length = %d, want 2", len(cfg.Connections))
	}

	c := cfg.Connections[0]
	if c.Name != "mydb" || c.Adapter != "postgres" || c.Host != "db.example.com" ||
		c.Port != 5432 || c.User != "admin" || c.Password != "secret" || c.Database != "production" {
		t.Errorf("Connection[0] fields mismatch: %+v", c)
	}

	c2 := cfg.Connections[1]
	if c2.Name != "localfile" || c2.Adapter != "sqlite" || c2.File != "/tmp/test.db" {
		t.Errorf("Connection[1] fields mismatch: %+v", c2)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for missing file", err)
	}

	def := DefaultConfig()
	if !reflect.DeepEqual(cfg, def) {
		t.Errorf("Load(missing) = %+v, want DefaultConfig %+v", cfg, def)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	// Invalid YAML: tab characters in indentation and broken structure
	content := "theme: [\ninvalid:\n  - {broken\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load(invalid YAML) error = nil, want error")
	}
}

func TestLoadPartialYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yaml")

	// Only set theme and tab_size, everything else should default
	yaml := `theme: dracula
editor:
  tab_size: 8
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Theme != "dracula" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dracula")
	}
	if cfg.Editor.TabSize != 8 {
		t.Errorf("Editor.TabSize = %d, want %d", cfg.Editor.TabSize, 8)
	}
	// These should remain at default values
	if cfg.KeyMode != "standard" {
		t.Errorf("KeyMode = %q, want default %q", cfg.KeyMode, "standard")
	}
	if cfg.Editor.ShowLineNumbers != true {
		t.Errorf("Editor.ShowLineNumbers = %v, want default true", cfg.Editor.ShowLineNumbers)
	}
	if cfg.Results.PageSize != 1000 {
		t.Errorf("Results.PageSize = %d, want default %d", cfg.Results.PageSize, 1000)
	}
	if cfg.Results.MaxColumnWidth != 50 {
		t.Errorf("Results.MaxColumnWidth = %d, want default %d", cfg.Results.MaxColumnWidth, 50)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")

	original := &Config{
		Theme:   "nord",
		KeyMode: "vim",
		Editor: EditorConfig{
			TabSize:         3,
			ShowLineNumbers: false,
		},
		Results: ResultsConfig{
			PageSize:       200,
			MaxColumnWidth: 100,
		},
		Connections: []SavedConnection{
			{
				Name:     "prod-pg",
				Adapter:  "postgres",
				Host:     "db.prod.internal",
				Port:     5433,
				User:     "appuser",
				Password: "p@ss!",
				Database: "maindb",
			},
			{
				Name:    "local-duck",
				Adapter: "duckdb",
				File:    "/data/analytics.duckdb",
			},
		},
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("roundtrip mismatch:\n  saved:  %+v\n  loaded: %+v", original, loaded)
	}
}

func TestSaveDefaultAndLoadDefault(t *testing.T) {
	// Override HOME (and XDG_CONFIG_HOME on Linux) to use a temp dir so
	// ConfigDir() resolves inside the test directory.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// On macOS, UserConfigDir returns ~/Library/Application Support, which
	// uses HOME. On Linux it checks XDG_CONFIG_HOME first.
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	cfg := &Config{
		Theme:   "solarized",
		KeyMode: "vim",
		Editor: EditorConfig{
			TabSize:         2,
			ShowLineNumbers: true,
		},
		Results: ResultsConfig{
			PageSize:       100,
			MaxColumnWidth: 40,
		},
	}

	if err := cfg.SaveDefault(); err != nil {
		t.Fatalf("SaveDefault() error = %v", err)
	}

	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	if loaded.Theme != cfg.Theme {
		t.Errorf("Theme = %q, want %q", loaded.Theme, cfg.Theme)
	}
	if loaded.KeyMode != cfg.KeyMode {
		t.Errorf("KeyMode = %q, want %q", loaded.KeyMode, cfg.KeyMode)
	}
	if loaded.Editor != cfg.Editor {
		t.Errorf("Editor = %+v, want %+v", loaded.Editor, cfg.Editor)
	}
	if loaded.Results != cfg.Results {
		t.Errorf("Results = %+v, want %+v", loaded.Results, cfg.Results)
	}
	if len(loaded.Connections) != len(cfg.Connections) {
		t.Errorf("Connections length = %d, want %d", len(loaded.Connections), len(cfg.Connections))
	}
}

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name string
		conn SavedConnection
		want string
	}{
		{
			name: "postgres all fields",
			conn: SavedConnection{
				Adapter:  "postgres",
				User:     "admin",
				Password: "secret",
				Host:     "db.example.com",
				Port:     5432,
				Database: "mydb",
			},
			want: "admin:secret@db.example.com:5432/mydb",
		},
		{
			name: "postgres host and database only",
			conn: SavedConnection{
				Adapter:  "postgres",
				Host:     "db.example.com",
				Database: "mydb",
			},
			want: "db.example.com/mydb",
		},
		{
			name: "postgres user without password",
			conn: SavedConnection{
				Adapter:  "postgres",
				User:     "readonly",
				Host:     "db.example.com",
				Port:     5432,
				Database: "mydb",
			},
			want: "readonly@db.example.com:5432/mydb",
		},
		{
			name: "postgres with DSN field set",
			conn: SavedConnection{
				Adapter:  "postgres",
				DSN:      "postgres://user:pass@host:5432/db?sslmode=disable",
				Host:     "ignored",
				Database: "ignored",
			},
			want: "postgres://user:pass@host:5432/db?sslmode=disable",
		},
		{
			name: "postgres defaults host to localhost",
			conn: SavedConnection{
				Adapter:  "postgres",
				User:     "dev",
				Password: "dev",
				Port:     5432,
				Database: "devdb",
			},
			want: "dev:dev@localhost:5432/devdb",
		},
		{
			name: "mysql all fields",
			conn: SavedConnection{
				Adapter:  "mysql",
				User:     "root",
				Password: "toor",
				Host:     "mysql.local",
				Port:     3306,
				Database: "app",
			},
			want: "root:toor@mysql.local:3306/app",
		},
		{
			name: "mysql with DSN field set",
			conn: SavedConnection{
				Adapter: "mysql",
				DSN:     "root:pass@tcp(localhost:3306)/db",
			},
			want: "root:pass@tcp(localhost:3306)/db",
		},
		{
			name: "sqlite file path",
			conn: SavedConnection{
				Adapter: "sqlite",
				File:    "/home/user/data.db",
			},
			want: "/home/user/data.db",
		},
		{
			name: "sqlite uppercase adapter",
			conn: SavedConnection{
				Adapter: "SQLite",
				File:    "/tmp/test.db",
			},
			want: "/tmp/test.db",
		},
		{
			name: "duckdb file path",
			conn: SavedConnection{
				Adapter: "duckdb",
				File:    "/data/analytics.duckdb",
			},
			want: "/data/analytics.duckdb",
		},
		{
			name: "duckdb uppercase adapter",
			conn: SavedConnection{
				Adapter: "DuckDB",
				File:    "/data/test.duckdb",
			},
			want: "/data/test.duckdb",
		},
		{
			name: "network adapter no port no database",
			conn: SavedConnection{
				Adapter: "postgres",
				Host:    "myhost",
			},
			want: "myhost",
		},
		{
			name: "network adapter empty fields defaults to localhost",
			conn: SavedConnection{
				Adapter: "postgres",
			},
			want: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.conn.BuildDSN()
			if got != tt.want {
				t.Errorf("BuildDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDisplayString(t *testing.T) {
	tests := []struct {
		name string
		conn SavedConnection
		want string
	}{
		{
			name: "postgres full",
			conn: SavedConnection{
				Adapter:  "postgres",
				Host:     "db.example.com",
				Port:     5432,
				Database: "mydb",
			},
			want: "postgres://db.example.com:5432/mydb",
		},
		{
			name: "postgres no port",
			conn: SavedConnection{
				Adapter:  "postgres",
				Host:     "db.example.com",
				Database: "mydb",
			},
			want: "postgres://db.example.com/mydb",
		},
		{
			name: "postgres no database",
			conn: SavedConnection{
				Adapter: "postgres",
				Host:    "db.example.com",
				Port:    5432,
			},
			want: "postgres://db.example.com:5432",
		},
		{
			name: "postgres host only (defaults to localhost)",
			conn: SavedConnection{
				Adapter: "postgres",
			},
			want: "postgres://localhost",
		},
		{
			name: "mysql full",
			conn: SavedConnection{
				Adapter:  "mysql",
				Host:     "mysql.local",
				Port:     3306,
				Database: "app",
			},
			want: "mysql://mysql.local:3306/app",
		},
		{
			name: "sqlite with file",
			conn: SavedConnection{
				Adapter: "sqlite",
				File:    "/home/user/data.db",
			},
			want: "sqlite:///home/user/data.db",
		},
		{
			name: "sqlite with DSN fallback",
			conn: SavedConnection{
				Adapter: "sqlite",
				DSN:     "/tmp/fallback.db",
			},
			want: "sqlite:///tmp/fallback.db",
		},
		{
			name: "duckdb with file",
			conn: SavedConnection{
				Adapter: "duckdb",
				File:    "/data/analytics.duckdb",
			},
			want: "duckdb:///data/analytics.duckdb",
		},
		{
			name: "sqlite empty file and DSN",
			conn: SavedConnection{
				Adapter: "sqlite",
			},
			want: "sqlite://",
		},
		{
			name: "DisplayString preserves adapter casing",
			conn: SavedConnection{
				Adapter:  "PostgreSQL",
				Host:     "myhost",
				Port:     5432,
				Database: "db",
			},
			// Adapter is not lowered for the display prefix
			want: "PostgreSQL://myhost:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.conn.DisplayString()
			if got != tt.want {
				t.Errorf("DisplayString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	if dir == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	if filepath.Base(dir) != "gotermsql" {
		t.Errorf("ConfigDir() base = %q, want %q", filepath.Base(dir), "gotermsql")
	}
}
