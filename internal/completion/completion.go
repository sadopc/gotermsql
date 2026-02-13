package completion

import (
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/schema"
	"github.com/sahilm/fuzzy"
)

// Engine provides SQL autocomplete suggestions based on schema and dialect.
type Engine struct {
	mu        sync.RWMutex
	tables    map[string][]schema.Column // "schema.table" -> columns
	schemas   []string
	databases []string
	dialect   string
	keywords  []string
	functions []string
}

// NewEngine creates a completion engine with keyword/function lists for the given dialect.
func NewEngine(dialect string) *Engine {
	return &Engine{
		tables:    make(map[string][]schema.Column),
		dialect:   dialect,
		keywords:  KeywordsForDialect(dialect),
		functions: FunctionsForDialect(dialect),
	}
}

// UpdateSchema refreshes the schema cache from introspection results.
func (e *Engine) UpdateSchema(databases []schema.Database) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tables = make(map[string][]schema.Column)
	e.schemas = nil
	e.databases = nil

	for _, db := range databases {
		e.databases = append(e.databases, db.Name)
		for _, s := range db.Schemas {
			e.schemas = append(e.schemas, s.Name)
			for _, t := range s.Tables {
				// Store with schema-qualified key.
				key := s.Name + "." + t.Name
				e.tables[key] = t.Columns
				// Also store with just the table name for unqualified lookups.
				e.tables[t.Name] = t.Columns
			}
			for _, v := range s.Views {
				key := s.Name + "." + v.Name
				e.tables[key] = v.Columns
				e.tables[v.Name] = v.Columns
			}
		}
	}
}

// Complete returns completion candidates for the given text and cursor position.
func (e *Engine) Complete(text string, cursorPos int) []adapter.CompletionItem {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}
	if cursorPos < 0 {
		cursorPos = 0
	}

	before := text[:cursorPos]

	// No completions inside string literals.
	if insideStringLiteral(before) {
		return nil
	}

	// Find the current word being typed.
	prefix, dotContext := extractPrefix(before)

	// If we have a dot context (e.g., "t." or "users."), complete columns for that table.
	if dotContext != "" {
		return e.completeDotAccess(dotContext, prefix)
	}

	// Determine the context keyword preceding the current word.
	ctx := detectContext(before, prefix)

	var items []adapter.CompletionItem

	switch ctx {
	case contextFrom:
		// After FROM, JOIN, INTO, UPDATE, TABLE: suggest table names.
		items = e.tableCompletions()
	case contextColumn:
		// After SELECT, WHERE, etc.: suggest columns from FROM tables + table names + functions.
		fromTables := e.parseFromTables(text)
		items = append(items, e.columnsFromTables(fromTables)...)
		items = append(items, e.tableCompletions()...)
		items = append(items, e.functionCompletions()...)
	default:
		// Default: suggest keywords + table names + functions.
		items = append(items, e.keywordCompletions()...)
		items = append(items, e.tableCompletions()...)
		items = append(items, e.functionCompletions()...)
	}

	if prefix == "" {
		// No prefix: return all candidates (limited to a reasonable number).
		if len(items) > 50 {
			items = items[:50]
		}
		return items
	}

	return fuzzyMatch(prefix, items)
}

// contextKind indicates the kind of SQL context before the cursor.
type contextKind int

const (
	contextGeneral contextKind = iota
	contextFrom
	contextColumn
)

// fromKeywords trigger table name completions.
var fromKeywords = map[string]bool{
	"FROM": true, "JOIN": true, "INTO": true, "UPDATE": true, "TABLE": true,
	"LEFT":  true, // LEFT JOIN
	"RIGHT": true, // RIGHT JOIN
	"INNER": true, // INNER JOIN
	"OUTER": true, // OUTER JOIN
	"FULL":  true, // FULL JOIN
	"CROSS": true, // CROSS JOIN
}

// columnKeywords trigger column name completions.
var columnKeywords = map[string]bool{
	"SELECT": true, "WHERE": true, "SET": true, "ON": true,
	"AND": true, "OR": true, "HAVING": true, "BY": true,
}

// detectContext looks at the text before the prefix to determine what completions to offer.
func detectContext(before, prefix string) contextKind {
	// Strip the prefix from the end to get the context text.
	ctxText := strings.TrimSpace(before[:len(before)-len(prefix)])
	if ctxText == "" {
		return contextGeneral
	}

	// Find the last significant token.
	tokens := tokenize(ctxText)
	if len(tokens) == 0 {
		return contextGeneral
	}

	lastToken := strings.ToUpper(tokens[len(tokens)-1])

	if fromKeywords[lastToken] {
		return contextFrom
	}
	if columnKeywords[lastToken] {
		return contextColumn
	}

	// Check if the last token ends with a comma (continuation of a list).
	if strings.HasSuffix(lastToken, ",") {
		// Walk backward to find what list we are in.
		for i := len(tokens) - 1; i >= 0; i-- {
			tok := strings.ToUpper(strings.TrimRight(tokens[i], ","))
			if fromKeywords[tok] {
				return contextFrom
			}
			if columnKeywords[tok] {
				return contextColumn
			}
		}
	}

	return contextGeneral
}

// extractPrefix returns the current word being typed and any dot-context.
// For "users.na", it returns prefix="na", dotContext="users".
// For "sel", it returns prefix="sel", dotContext="".
func extractPrefix(before string) (prefix, dotContext string) {
	// Walk backward to find the start of the current word.
	i := len(before) - 1
	for i >= 0 && !isWordBreak(rune(before[i])) {
		i--
	}
	word := before[i+1:]

	// Check for dot access.
	if dotIdx := strings.LastIndex(word, "."); dotIdx >= 0 {
		return word[dotIdx+1:], word[:dotIdx]
	}

	return word, ""
}

// isWordBreak returns true if the rune is a word boundary for SQL identifiers.
func isWordBreak(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' {
		return false
	}
	return true
}

// insideStringLiteral checks if the cursor is inside an unmatched string literal.
func insideStringLiteral(before string) bool {
	singleCount := 0
	for _, ch := range before {
		if ch == '\'' {
			singleCount++
		}
	}
	return singleCount%2 != 0
}

// tokenize splits text into rough SQL tokens (whitespace-separated).
func tokenize(text string) []string {
	return strings.Fields(text)
}

// fromClauseRe matches FROM clauses and extracts table references.
var fromClauseRe = regexp.MustCompile(`(?i)\bFROM\s+([\w."]+(?:\s+(?:AS\s+)?[\w]+)?(?:\s*,\s*[\w."]+(?:\s+(?:AS\s+)?[\w]+)?)*)`)

// joinClauseRe matches JOIN clauses and extracts the table name.
var joinClauseRe = regexp.MustCompile(`(?i)\bJOIN\s+([\w."]+)`)

// parseFromTables extracts table names from FROM and JOIN clauses in the SQL text.
func (e *Engine) parseFromTables(text string) []string {
	var tables []string
	seen := map[string]bool{}

	// Extract FROM clause tables.
	for _, match := range fromClauseRe.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		// Split by comma for multi-table FROM.
		parts := strings.Split(match[1], ",")
		for _, part := range parts {
			tokens := strings.Fields(strings.TrimSpace(part))
			if len(tokens) > 0 {
				name := strings.Trim(tokens[0], `"`)
				if !seen[name] {
					seen[name] = true
					tables = append(tables, name)
				}
			}
		}
	}

	// Extract JOIN clause tables.
	for _, match := range joinClauseRe.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		name := strings.Trim(match[1], `"`)
		if !seen[name] {
			seen[name] = true
			tables = append(tables, name)
		}
	}

	return tables
}

// completeDotAccess returns column completions for a dot-accessed table or alias.
func (e *Engine) completeDotAccess(tableName, prefix string) []adapter.CompletionItem {
	items := e.columnsForTable(tableName)
	if prefix == "" {
		return items
	}
	return fuzzyMatch(prefix, items)
}

// columnsForTable looks up columns for a table name, trying with and without schema prefix.
func (e *Engine) columnsForTable(tableName string) []adapter.CompletionItem {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Direct lookup.
	if cols, ok := e.tables[tableName]; ok {
		return columnsToItems(tableName, cols)
	}

	// Try with each known schema prefix.
	for _, s := range e.schemas {
		key := s + "." + tableName
		if cols, ok := e.tables[key]; ok {
			return columnsToItems(tableName, cols)
		}
	}

	return nil
}

// columnsFromTables returns column completions for a list of table names.
func (e *Engine) columnsFromTables(tableNames []string) []adapter.CompletionItem {
	var items []adapter.CompletionItem
	for _, t := range tableNames {
		items = append(items, e.columnsForTable(t)...)
	}
	return items
}

// columnsToItems converts schema columns to completion items.
func columnsToItems(tableName string, cols []schema.Column) []adapter.CompletionItem {
	items := make([]adapter.CompletionItem, 0, len(cols))
	for _, c := range cols {
		detail := c.Type
		if c.IsPK {
			detail += " PK"
		}
		if !c.Nullable {
			detail += " NOT NULL"
		}
		items = append(items, adapter.CompletionItem{
			Label:  c.Name,
			Kind:   adapter.CompletionColumn,
			Detail: tableName + " - " + detail,
		})
	}
	return items
}

// tableCompletions returns completion items for all known tables.
func (e *Engine) tableCompletions() []adapter.CompletionItem {
	e.mu.RLock()
	defer e.mu.RUnlock()

	seen := map[string]bool{}
	var items []adapter.CompletionItem

	for name := range e.tables {
		// Skip schema-qualified names if the unqualified name is also present
		// to avoid duplicates in the suggestion list.
		if strings.Contains(name, ".") {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		items = append(items, adapter.CompletionItem{
			Label:  name,
			Kind:   adapter.CompletionTable,
			Detail: "table",
		})
	}

	// Also add schema-qualified names for tables that only exist qualified.
	for name := range e.tables {
		if !strings.Contains(name, ".") {
			continue
		}
		parts := strings.SplitN(name, ".", 2)
		if len(parts) == 2 && !seen[parts[1]] {
			// The unqualified name was not added, so add the qualified one.
			if !seen[name] {
				seen[name] = true
				items = append(items, adapter.CompletionItem{
					Label:  name,
					Kind:   adapter.CompletionTable,
					Detail: "table",
				})
			}
		}
	}

	return items
}

// keywordCompletions returns completion items for all keywords in the dialect.
func (e *Engine) keywordCompletions() []adapter.CompletionItem {
	items := make([]adapter.CompletionItem, 0, len(e.keywords))
	for _, kw := range e.keywords {
		items = append(items, adapter.CompletionItem{
			Label:  kw,
			Kind:   adapter.CompletionKeyword,
			Detail: "keyword",
		})
	}
	return items
}

// functionCompletions returns completion items for all functions in the dialect.
func (e *Engine) functionCompletions() []adapter.CompletionItem {
	items := make([]adapter.CompletionItem, 0, len(e.functions))
	for _, fn := range e.functions {
		items = append(items, adapter.CompletionItem{
			Label:  fn,
			Kind:   adapter.CompletionFunction,
			Detail: "function",
		})
	}
	return items
}

// candidateLabels implements fuzzy.Source for a slice of CompletionItems.
type candidateLabels []adapter.CompletionItem

func (c candidateLabels) String(i int) string { return c[i].Label }
func (c candidateLabels) Len() int            { return len(c) }

// fuzzyMatch filters and ranks completion items by fuzzy matching against the prefix.
func fuzzyMatch(prefix string, items []adapter.CompletionItem) []adapter.CompletionItem {
	if len(items) == 0 {
		return nil
	}

	// Fuzzy match is case-insensitive: we lowercase both the prefix and the
	// labels for matching, but return the original labels.
	lowerPrefix := strings.ToLower(prefix)
	lowerItems := make(candidateLabels, len(items))
	for i, item := range items {
		lowerItems[i] = adapter.CompletionItem{
			Label:  strings.ToLower(item.Label),
			Kind:   item.Kind,
			Detail: item.Detail,
		}
	}

	matches := fuzzy.FindFrom(lowerPrefix, lowerItems)

	// Sort by score descending.
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	result := make([]adapter.CompletionItem, 0, len(matches))
	for _, m := range matches {
		result = append(result, items[m.Index])
	}

	// Cap at a reasonable maximum.
	if len(result) > 50 {
		result = result[:50]
	}

	return result
}
