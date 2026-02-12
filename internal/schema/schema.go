package schema

// Database represents a database with its schemas.
type Database struct {
	Name    string
	Schemas []Schema
}

// Schema represents a database schema (e.g., "public" in PostgreSQL).
type Schema struct {
	Name   string
	Tables []Table
	Views  []View
}

// Table represents a database table.
type Table struct {
	Name    string
	Columns []Column
	Indexes []Index
	FKs     []ForeignKey
}

// Column represents a table column.
type Column struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
	IsPK     bool
}

// Index represents a table index.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKey represents a foreign key constraint.
type ForeignKey struct {
	Name       string
	Columns    []string
	RefTable   string
	RefColumns []string
}

// View represents a database view.
type View struct {
	Name       string
	Columns    []Column
	Definition string
}
