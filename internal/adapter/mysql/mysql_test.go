package mysql

import (
	"testing"

	"github.com/sadopc/gotermsql/internal/adapter"
)

func TestMySQLAdapter_Name(t *testing.T) {
	a := &mysqlAdapter{}
	if got := a.Name(); got != "mysql" {
		t.Errorf("Name() = %q, want %q", got, "mysql")
	}
}

func TestMySQLAdapter_DefaultPort(t *testing.T) {
	a := &mysqlAdapter{}
	if got := a.DefaultPort(); got != 3306 {
		t.Errorf("DefaultPort() = %d, want %d", got, 3306)
	}
}

func TestMySQLAdapter_Registration(t *testing.T) {
	a, ok := adapter.Registry["mysql"]
	if !ok {
		t.Fatal("mysql adapter not found in registry")
	}
	if a.Name() != "mysql" {
		t.Errorf("registered adapter Name() = %q, want %q", a.Name(), "mysql")
	}
	if a.DefaultPort() != 3306 {
		t.Errorf("registered adapter DefaultPort() = %d, want %d", a.DefaultPort(), 3306)
	}
}

func TestNormalizeDSN(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantDSN    string
		wantDBName string
		wantErr    bool
	}{
		{
			name:       "mysql URL with user and pass",
			input:      "mysql://user:pass@localhost:3306/mydb",
			wantDSN:    "user:pass@tcp(localhost:3306)/mydb?parseTime=true",
			wantDBName: "mydb",
		},
		{
			name:       "mysql URL user only, no port",
			input:      "mysql://user@localhost/db",
			wantDSN:    "user@tcp(localhost:3306)/db?parseTime=true",
			wantDBName: "db",
		},
		{
			name:       "mysql URL with existing params",
			input:      "mysql://user:pass@host:3307/testdb?charset=utf8",
			wantDSN:    "user:pass@tcp(host:3307)/testdb?charset=utf8&parseTime=true",
			wantDBName: "testdb",
		},
		{
			name:       "mysql URL with parseTime already set",
			input:      "mysql://user:pass@host:3306/db?parseTime=true",
			wantDSN:    "user:pass@tcp(host:3306)/db?parseTime=true",
			wantDBName: "db",
		},
		{
			name:       "go-sql-driver format passthrough",
			input:      "user:pass@tcp(host:3306)/db",
			wantDSN:    "user:pass@tcp(host:3306)/db?parseTime=true",
			wantDBName: "db",
		},
		{
			name:       "go-sql-driver format with existing params",
			input:      "user:pass@tcp(host:3306)/db?charset=utf8",
			wantDSN:    "user:pass@tcp(host:3306)/db?charset=utf8&parseTime=true",
			wantDBName: "db",
		},
		{
			name:       "go-sql-driver format with parseTime",
			input:      "user:pass@tcp(host:3306)/db?parseTime=true",
			wantDSN:    "user:pass@tcp(host:3306)/db?parseTime=true",
			wantDBName: "db",
		},
		{
			name:       "mysql URL no user",
			input:      "mysql://localhost/mydb",
			wantDSN:    "@tcp(localhost:3306)/mydb?parseTime=true",
			wantDBName: "mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDSN, gotDBName, err := normalizeDSN(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeDSN(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotDSN != tt.wantDSN {
				t.Errorf("normalizeDSN(%q) DSN = %q, want %q", tt.input, gotDSN, tt.wantDSN)
			}
			if gotDBName != tt.wantDBName {
				t.Errorf("normalizeDSN(%q) dbName = %q, want %q", tt.input, gotDBName, tt.wantDBName)
			}
		})
	}
}

func TestExtractDBName_GoDriverFormat(t *testing.T) {
	// Test that database name extraction works for go-sql-driver DSN format.
	tests := []struct {
		name       string
		input      string
		wantDBName string
	}{
		{
			name:       "standard go-sql-driver DSN",
			input:      "user:pass@tcp(host:3306)/mydb",
			wantDBName: "mydb",
		},
		{
			name:       "go-sql-driver DSN with params",
			input:      "user:pass@tcp(host:3306)/testdb?charset=utf8mb4",
			wantDBName: "testdb",
		},
		{
			name:       "simple DSN with just database",
			input:      "/mydb",
			wantDBName: "mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotDBName, err := normalizeDSN(tt.input)
			if err != nil {
				t.Fatalf("normalizeDSN(%q) unexpected error: %v", tt.input, err)
			}
			if gotDBName != tt.wantDBName {
				t.Errorf("normalizeDSN(%q) dbName = %q, want %q", tt.input, gotDBName, tt.wantDBName)
			}
		})
	}
}

func TestIsSelectQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"simple SELECT", "SELECT * FROM users", true},
		{"lowercase select", "select 1", true},
		{"SELECT with leading spaces", "  SELECT * FROM t", true},
		{"SHOW", "SHOW DATABASES", true},
		{"DESCRIBE", "DESCRIBE users", true},
		{"DESC", "DESC users", true},
		{"EXPLAIN", "EXPLAIN SELECT * FROM users", true},
		{"WITH CTE", "WITH cte AS (SELECT 1) SELECT * FROM cte", true},
		{"INSERT", "INSERT INTO users VALUES (1)", false},
		{"UPDATE", "UPDATE users SET name = 'bob'", false},
		{"DELETE", "DELETE FROM users WHERE id = 1", false},
		{"CREATE TABLE", "CREATE TABLE foo (id INT)", false},
		{"DROP TABLE", "DROP TABLE foo", false},
		{"ALTER TABLE", "ALTER TABLE foo ADD COLUMN bar INT", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSelectQuery(tt.query)
			if got != tt.want {
				t.Errorf("isSelectQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestNormalizeDSN_ParseTimeInjection(t *testing.T) {
	// Verify that parseTime=true is always present in the output DSN.
	tests := []struct {
		name  string
		input string
	}{
		{"mysql URL no params", "mysql://user:pass@localhost:3306/db"},
		{"mysql URL with other params", "mysql://user:pass@localhost:3306/db?charset=utf8"},
		{"go-driver no params", "user:pass@tcp(localhost:3306)/db"},
		{"go-driver with other params", "user:pass@tcp(localhost:3306)/db?charset=utf8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDSN, _, err := normalizeDSN(tt.input)
			if err != nil {
				t.Fatalf("normalizeDSN(%q) error = %v", tt.input, err)
			}
			if !containsParseTime(gotDSN) {
				t.Errorf("normalizeDSN(%q) = %q, expected parseTime to be present", tt.input, gotDSN)
			}
		})
	}
}

// containsParseTime checks if the DSN contains parseTime parameter.
func containsParseTime(dsn string) bool {
	for i := 0; i <= len(dsn)-9; i++ {
		if dsn[i:i+9] == "parseTime" {
			return true
		}
	}
	return false
}
