package completion

// CommonKeywords are SQL keywords shared across all dialects.
var CommonKeywords = []string{
	"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
	"FULL", "CROSS", "ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN",
	"LIKE", "ILIKE", "IS", "NULL", "AS", "CASE", "WHEN", "THEN", "ELSE",
	"END", "INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE", "CREATE",
	"ALTER", "DROP", "TABLE", "VIEW", "INDEX", "UNIQUE", "PRIMARY", "KEY",
	"FOREIGN", "REFERENCES", "CONSTRAINT", "DEFAULT", "CHECK", "CASCADE",
	"RESTRICT", "GROUP", "BY", "ORDER", "ASC", "DESC", "HAVING", "LIMIT",
	"OFFSET", "DISTINCT", "ALL", "ANY", "SOME", "UNION", "INTERSECT",
	"EXCEPT", "WITH", "RECURSIVE", "RETURNING", "BEGIN", "COMMIT",
	"ROLLBACK", "TRANSACTION", "GRANT", "REVOKE", "EXPLAIN", "ANALYZE",
	"VACUUM", "TRUNCATE", "IF", "REPLACE", "TEMPORARY", "TEMP",
}

// CommonFunctions are SQL functions shared across all dialects.
var CommonFunctions = []string{
	"COUNT", "SUM", "AVG", "MIN", "MAX", "COALESCE", "NULLIF", "CAST",
	"CASE", "LOWER", "UPPER", "TRIM", "LTRIM", "RTRIM", "LENGTH",
	"SUBSTRING", "REPLACE", "CONCAT", "ABS", "CEIL", "FLOOR", "ROUND",
	"NOW", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME", "EXTRACT",
	"DATE_TRUNC", "TO_CHAR", "TO_DATE", "TO_NUMBER", "ROW_NUMBER", "RANK",
	"DENSE_RANK", "LAG", "LEAD", "FIRST_VALUE", "LAST_VALUE", "NTILE",
	"STRING_AGG", "ARRAY_AGG", "JSON_AGG", "BOOL_AND", "BOOL_OR", "EVERY",
}

// PostgresKeywords are additional keywords specific to PostgreSQL.
var PostgresKeywords = []string{
	"SERIAL", "BIGSERIAL", "RETURNING", "ILIKE", "SIMILAR", "LATERAL",
	"MATERIALIZED", "CONCURRENTLY", "TABLESPACE", "SCHEMA", "EXTENSION",
	"SEQUENCE", "OWNED", "NOTIFY", "LISTEN", "PERFORM", "RAISE", "COPY",
}

// MySQLKeywords are additional keywords specific to MySQL.
var MySQLKeywords = []string{
	"AUTO_INCREMENT", "ENGINE", "CHARSET", "COLLATE", "SHOW", "DESCRIBE",
	"USE", "DATABASES", "TABLES", "COLUMNS", "STATUS", "VARIABLES",
	"PROCESSLIST", "BINARY", "UNSIGNED", "ZEROFILL", "ENUM", "MEDIUMTEXT",
	"LONGTEXT", "TINYINT", "MEDIUMINT",
}

// SQLiteKeywords are additional keywords specific to SQLite.
var SQLiteKeywords = []string{
	"PRAGMA", "AUTOINCREMENT", "GLOB", "ATTACH", "DETACH", "REINDEX",
	"INDEXED", "WITHOUT", "ROWID", "STRICT",
}

// DuckDBKeywords are additional keywords specific to DuckDB.
var DuckDBKeywords = []string{
	"PIVOT", "UNPIVOT", "SAMPLE", "USING", "QUALIFY", "COLUMNS", "STRUCT",
	"LIST", "MAP", "HUGEINT", "UBIGINT", "UINTEGER",
}

// KeywordsForDialect returns CommonKeywords combined with dialect-specific keywords.
func KeywordsForDialect(dialect string) []string {
	result := make([]string, len(CommonKeywords))
	copy(result, CommonKeywords)

	switch dialect {
	case "postgres", "postgresql":
		result = append(result, PostgresKeywords...)
	case "mysql":
		result = append(result, MySQLKeywords...)
	case "sqlite":
		result = append(result, SQLiteKeywords...)
	case "duckdb":
		result = append(result, DuckDBKeywords...)
	}

	return result
}

// FunctionsForDialect returns the function list for the given dialect.
// For now, all dialects share the same function list.
func FunctionsForDialect(dialect string) []string {
	result := make([]string, len(CommonFunctions))
	copy(result, CommonFunctions)
	return result
}
