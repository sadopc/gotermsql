package results

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadopc/gotermsql/internal/adapter"
)

func columns(names ...string) []adapter.ColumnMeta {
	cols := make([]adapter.ColumnMeta, len(names))
	for i, name := range names {
		cols[i] = adapter.ColumnMeta{Name: name}
	}
	return cols
}

// --- CSV Tests ---

func TestExportCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	cols := columns("id", "name", "email")
	rows := [][]string{
		{"1", "Alice", "alice@example.com"},
		{"2", "Bob", "bob@example.com"},
		{"3", "Charlie", "charlie@example.com"},
	}

	err := ExportCSV(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	// Read back and verify.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	// 1 header + 3 data rows.
	if len(records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(records))
	}

	// Verify header.
	if records[0][0] != "id" || records[0][1] != "name" || records[0][2] != "email" {
		t.Fatalf("unexpected header: %v", records[0])
	}

	// Verify data rows.
	if records[1][0] != "1" || records[1][1] != "Alice" || records[1][2] != "alice@example.com" {
		t.Fatalf("unexpected row 1: %v", records[1])
	}
	if records[2][0] != "2" || records[2][1] != "Bob" {
		t.Fatalf("unexpected row 2: %v", records[2])
	}
	if records[3][0] != "3" || records[3][1] != "Charlie" {
		t.Fatalf("unexpected row 3: %v", records[3])
	}
}

func TestExportCSV_EmptyRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")

	cols := columns("id", "name")
	rows := [][]string{}

	err := ExportCSV(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	// Read back.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	// Header only.
	if len(records) != 1 {
		t.Fatalf("expected 1 record (header only), got %d", len(records))
	}
	if records[0][0] != "id" || records[0][1] != "name" {
		t.Fatalf("unexpected header: %v", records[0])
	}
}

func TestExportCSV_SpecialChars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "special.csv")

	cols := columns("description", "notes")
	rows := [][]string{
		{"has, commas", "has \"quotes\""},
		{"has\nnewlines", "normal text"},
		{"", "empty first column"},
	}

	err := ExportCSV(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	// Read back and verify special characters are preserved.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	// 1 header + 3 data rows.
	if len(records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(records))
	}

	// Verify special characters are preserved through CSV encoding/decoding.
	if records[1][0] != "has, commas" {
		t.Fatalf("comma in value not preserved: got %q", records[1][0])
	}
	if records[1][1] != "has \"quotes\"" {
		t.Fatalf("quotes in value not preserved: got %q", records[1][1])
	}
	if records[2][0] != "has\nnewlines" {
		t.Fatalf("newline in value not preserved: got %q", records[2][0])
	}
	if records[3][0] != "" {
		t.Fatalf("empty value not preserved: got %q", records[3][0])
	}
}

func TestExportCSV_InvalidPath(t *testing.T) {
	err := ExportCSV("/nonexistent/dir/file.csv", columns("id"), nil)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// --- JSON Tests ---

func TestExportJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	cols := columns("id", "name", "email")
	rows := [][]string{
		{"1", "Alice", "alice@example.com"},
		{"2", "Bob", "bob@example.com"},
	}

	err := ExportJSON(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	// Read back and parse.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}

	// Verify first object.
	if objects[0]["id"] != "1" {
		t.Fatalf("expected id=1, got %q", objects[0]["id"])
	}
	if objects[0]["name"] != "Alice" {
		t.Fatalf("expected name=Alice, got %q", objects[0]["name"])
	}
	if objects[0]["email"] != "alice@example.com" {
		t.Fatalf("expected email=alice@example.com, got %q", objects[0]["email"])
	}

	// Verify second object.
	if objects[1]["id"] != "2" {
		t.Fatalf("expected id=2, got %q", objects[1]["id"])
	}
	if objects[1]["name"] != "Bob" {
		t.Fatalf("expected name=Bob, got %q", objects[1]["name"])
	}
}

func TestExportJSON_EmptyRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	cols := columns("id", "name")
	rows := [][]string{}

	err := ExportJSON(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	// Read back and parse.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if len(objects) != 0 {
		t.Fatalf("expected 0 objects (empty array), got %d", len(objects))
	}
}

func TestExportJSON_NullValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nulls.json")

	cols := columns("id", "name", "bio")
	rows := [][]string{
		{"1", "Alice", ""},
		{"2", "", ""},
	}

	err := ExportJSON(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}

	// Empty strings should be preserved as empty strings in JSON.
	if objects[0]["bio"] != "" {
		t.Fatalf("expected empty bio, got %q", objects[0]["bio"])
	}
	if objects[1]["name"] != "" {
		t.Fatalf("expected empty name, got %q", objects[1]["name"])
	}
}

func TestExportJSON_RowShorterThanColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short_row.json")

	cols := columns("id", "name", "email")
	rows := [][]string{
		{"1"}, // Only 1 value but 3 columns
	}

	err := ExportJSON(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}

	// Missing columns should default to empty string.
	if objects[0]["id"] != "1" {
		t.Fatalf("expected id=1, got %q", objects[0]["id"])
	}
	if objects[0]["name"] != "" {
		t.Fatalf("expected empty name for short row, got %q", objects[0]["name"])
	}
	if objects[0]["email"] != "" {
		t.Fatalf("expected empty email for short row, got %q", objects[0]["email"])
	}
}

func TestExportJSON_SpecialChars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "special.json")

	cols := columns("data")
	rows := [][]string{
		{`has "quotes" and \backslash`},
		{"has\nnewlines\tand\ttabs"},
		{"unicode: \u00e9\u00e8\u00ea"},
	}

	err := ExportJSON(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if len(objects) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(objects))
	}

	if objects[0]["data"] != `has "quotes" and \backslash` {
		t.Fatalf("quotes not preserved: got %q", objects[0]["data"])
	}
	if objects[1]["data"] != "has\nnewlines\tand\ttabs" {
		t.Fatalf("newlines/tabs not preserved: got %q", objects[1]["data"])
	}
}

func TestExportJSON_InvalidPath(t *testing.T) {
	err := ExportJSON("/nonexistent/dir/file.json", columns("id"), nil)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// --- Edge cases ---

func TestExportCSV_SingleColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.csv")

	cols := columns("value")
	rows := [][]string{{"hello"}, {"world"}}

	err := ExportCSV(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if len(records[0]) != 1 || records[0][0] != "value" {
		t.Fatalf("unexpected header: %v", records[0])
	}
	if records[1][0] != "hello" {
		t.Fatalf("expected 'hello', got %q", records[1][0])
	}
}

func TestExportJSON_SingleColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.json")

	cols := columns("value")
	rows := [][]string{{"hello"}, {"world"}}

	err := ExportJSON(path, cols, rows)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}
	if objects[0]["value"] != "hello" {
		t.Fatalf("expected 'hello', got %q", objects[0]["value"])
	}
	if objects[1]["value"] != "world" {
		t.Fatalf("expected 'world', got %q", objects[1]["value"])
	}
}

func TestExportCSV_NoColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nocols.csv")

	err := ExportCSV(path, nil, nil)
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	// Writing an empty slice header produces no records (csv.Writer skips empty slices).
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("expected 0 records for nil columns, got %d", len(records))
	}
}

func TestExportJSON_NoColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nocols.json")

	err := ExportJSON(path, nil, nil)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var objects []map[string]string
	err = json.Unmarshal(data, &objects)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(objects) != 0 {
		t.Fatalf("expected 0 objects, got %d", len(objects))
	}
}
