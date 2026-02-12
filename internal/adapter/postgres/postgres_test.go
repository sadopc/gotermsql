package postgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/sadopc/gotermsql/internal/adapter"
)

func TestPostgresAdapter_Name(t *testing.T) {
	a := &postgresAdapter{}
	if got := a.Name(); got != "postgres" {
		t.Errorf("Name() = %q, want %q", got, "postgres")
	}
}

func TestPostgresAdapter_DefaultPort(t *testing.T) {
	a := &postgresAdapter{}
	if got := a.DefaultPort(); got != 5432 {
		t.Errorf("DefaultPort() = %d, want %d", got, 5432)
	}
}

func TestPostgresAdapter_Registration(t *testing.T) {
	// The init() function should have registered the adapter.
	a, ok := adapter.Registry["postgres"]
	if !ok {
		t.Fatal("postgres adapter not found in registry")
	}
	if a.Name() != "postgres" {
		t.Errorf("registered adapter Name() = %q, want %q", a.Name(), "postgres")
	}
	if a.DefaultPort() != 5432 {
		t.Errorf("registered adapter DefaultPort() = %d, want %d", a.DefaultPort(), 5432)
	}
}

func TestExtractDBName(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "standard postgres URL",
			dsn:  "postgres://user:pass@localhost:5432/mydb",
			want: "mydb",
		},
		{
			name: "postgres URL without port",
			dsn:  "postgres://localhost/testdb",
			want: "testdb",
		},
		{
			name: "postgres URL without database",
			dsn:  "postgres://localhost",
			want: "",
		},
		{
			name: "postgresql scheme with params",
			dsn:  "postgresql://user@host:5432/dbname?sslmode=disable",
			want: "dbname",
		},
		{
			name: "postgres URL with complex password",
			dsn:  "postgres://user:p%40ss@localhost:5432/production",
			want: "production",
		},
		{
			name: "keyword=value format with dbname",
			dsn:  "host=localhost port=5432 dbname=myapp user=admin",
			want: "myapp",
		},
		{
			name: "empty string",
			dsn:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDBName(tt.dsn)
			if got != tt.want {
				t.Errorf("extractDBName(%q) = %q, want %q", tt.dsn, got, tt.want)
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
		{"INSERT", "INSERT INTO users (name) VALUES ('alice')", false},
		{"UPDATE", "UPDATE users SET name = 'bob'", false},
		{"DELETE", "DELETE FROM users WHERE id = 1", false},
		{"lowercase select with leading space", "  select * from t", true},
		{"WITH CTE", "WITH cte AS (SELECT 1) SELECT * FROM cte", true},
		{"EXPLAIN", "EXPLAIN SELECT * FROM users", true},
		{"SHOW", "SHOW search_path", true},
		{"VALUES", "VALUES (1, 'a'), (2, 'b')", true},
		{"TABLE", "TABLE users", true},
		{"CREATE TABLE", "CREATE TABLE foo (id int)", false},
		{"DROP TABLE", "DROP TABLE foo", false},
		{"ALTER TABLE", "ALTER TABLE foo ADD COLUMN bar int", false},
		{"mixed case SELECT", "SeLeCt 1", true},
		{"line comment before SELECT", "-- comment\nSELECT 1", true},
		{"block comment before SELECT", "/* comment */ SELECT 1", true},
		{"line comment before INSERT", "-- comment\nINSERT INTO t VALUES (1)", false},
		{"empty string", "", false},
		{"only whitespace", "   ", false},
		{"GRANT", "GRANT ALL ON users TO admin", false},
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

func TestValueToString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"bytes", []byte("world"), "world"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int8", int8(42), "42"},
		{"int16", int16(1000), "1000"},
		{"int32", int32(100000), "100000"},
		{"int64", int64(9999999999), "9999999999"},
		{"float32", float32(3.14), "3.14"},
		{"float64", float64(2.718281828), "2.718281828"},
		{"time date only", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), "2024-06-15"},
		{"time with time", time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC), "2024-06-15 14:30:45"},
		{"string slice", []string{"a", "b", "c"}, "{a,b,c}"},
		{"empty string slice", []string{}, "{}"},
		{"int32 slice", []int32{1, 2, 3}, "{1,2,3}"},
		{"int64 slice", []int64{10, 20, 30}, "{10,20,30}"},
		{"float64 slice", []float64{1.1, 2.2}, "{1.1,2.2}"},
		{"bool slice", []bool{true, false, true}, "{true,false,true}"},
		{"UUID [16]byte", [16]byte{
			0x12, 0x34, 0x56, 0x78,
			0x9a, 0xbc,
			0xde, 0xf0,
			0x12, 0x34,
			0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
		}, "12345678-9abc-def0-1234-56789abcdef0"},
		{"unknown type (int)", 42, "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueToString(tt.value)
			if got != tt.want {
				t.Errorf("valueToString(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestValuesToStrings(t *testing.T) {
	input := []any{"hello", int32(42), nil, true}
	got := valuesToStrings(input)
	want := []string{"hello", "42", "", "true"}

	if len(got) != len(want) {
		t.Fatalf("valuesToStrings() returned %d elements, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("valuesToStrings()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPgTypeOIDToName(t *testing.T) {
	tests := []struct {
		oid  uint32
		want string
	}{
		{16, "bool"},
		{17, "bytea"},
		{18, "char"},
		{20, "int8"},
		{21, "int2"},
		{23, "int4"},
		{25, "text"},
		{26, "oid"},
		{114, "json"},
		{142, "xml"},
		{700, "float4"},
		{701, "float8"},
		{790, "money"},
		{1000, "bool[]"},
		{1005, "int2[]"},
		{1007, "int4[]"},
		{1009, "text[]"},
		{1016, "int8[]"},
		{1021, "float4[]"},
		{1022, "float8[]"},
		{1042, "bpchar"},
		{1043, "varchar"},
		{1082, "date"},
		{1083, "time"},
		{1114, "timestamp"},
		{1184, "timestamptz"},
		{1186, "interval"},
		{1266, "timetz"},
		{1700, "numeric"},
		{2249, "record"},
		{2278, "void"},
		{2950, "uuid"},
		{3802, "jsonb"},
		{3807, "jsonb[]"},
		{99999, fmt.Sprintf("oid:%d", 99999)},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := pgTypeOIDToName(tt.oid)
			if got != tt.want {
				t.Errorf("pgTypeOIDToName(%d) = %q, want %q", tt.oid, got, tt.want)
			}
		})
	}
}
