# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build              # CGO_ENABLED=0, pure Go (no DuckDB)
make build-full         # CGO_ENABLED=1 -tags duckdb (requires C compiler)
make test               # go test ./...
make test-race          # go test -race ./...
make vet                # go vet ./...
make fmt                # gofmt + goimports
make run ARGS="--adapter sqlite --file demo.db"
```

Run a single test:
```bash
go test ./internal/completion/ -run TestFuzzyMatch
```

PostgreSQL integration tests require a running instance and `gotermsql_test` database:
```bash
go test ./internal/adapter/postgres/ -run TestIntegration -v
GOTERMSQL_PG_DSN="postgres://user:pass@host/db" go test ./internal/adapter/postgres/ -run TestIntegration
```
Integration tests auto-skip if PostgreSQL is unavailable.

Version info is injected via LDFLAGS from git tags/commit/date.

## Architecture

Bubble Tea (Elm Architecture) TUI. Root model in `internal/app/app.go`.

**Message routing priority in `Update()`:**
1. `tea.WindowSizeMsg` → recalculate layout
2. `tea.KeyMsg` → help overlay (if visible, blocks all input) → connection manager (if visible) → autocomplete (if visible) → global keys → focused pane handler
3. Application messages: Connect, SchemaLoaded, QueryResult, NewTab, etc.

**Focus system:** Three panes (`PaneSidebar`, `PaneEditor`, `PaneResults`). Tab/Shift+Tab cycles. Alt+1/2/3 jumps directly. Each pane's component has `Focus()`/`Blur()` methods that control whether it processes input.

**Multi-tab state:** `tabStates map[int]*TabState` — each tab owns its own `editor.Model` and `results.Model`. Always nil-check `activeTabState()`.

**Layout system:** Tab bar (top) + status bar (bottom) + main area. Main area splits into sidebar (left, fixed width) + editor/results (right, percentage-based vertical split). Resizable via Ctrl+Arrow keys. Sidebar: 15–50% width. Editor height: 20–80%.

**Border width accounting:** lipgloss `.Width(w)` sets *content* width; borders add 2 chars on top. All components must use `.Width(m.width - 2)` to fit within their allocated space (editor and results do this; sidebar was fixed to match).

## Adapter Pattern

`internal/adapter/adapter.go` defines the `Adapter` and `Connection` interfaces. Each database package registers itself via `init()`:

```go
func init() { adapter.Register(&Adapter{}) }
```

Imported as blank imports in `cmd/gotermsql/main.go` to trigger registration.

**`Connection.Databases()` contract:** Must return `[]schema.Database` with `Schemas` and `Tables` populated for the connected database. PostgreSQL can only introspect the current database via `information_schema`; other databases appear as names only. SQLite returns a single database with `"main"` schema.

**DuckDB conditional compilation:** `duckdb_enabled.go` (`//go:build duckdb`) has the real implementation; `duckdb_disabled.go` (`//go:build !duckdb`) registers a stub that returns "not compiled in" errors. Both files exist so the code compiles with or without the tag.

## Autocomplete System

Two layers with different word-break rules:

- **`internal/completion/completion.go`** (Engine): Determines context from SQL text (FROM → tables, SELECT → columns+functions, dot → qualified columns). Thread-safe with `sync.RWMutex`. Dot is NOT a word break here (enables `table.column` lookup). Fuzzy matching ranks candidates.
- **`internal/ui/autocomplete/autocomplete.go`** (UI Model): Manages the visible dropdown. Dot IS a word break here (for prefix extraction). Sends `SelectedMsg{Text, PrefixLen}` — the full label plus how many chars to replace.

**Accepting completions:** The app calls `editor.ReplaceWord(text, prefixLen)` which removes the typed prefix from the end and appends the full completion. This avoids the old bug where prefix-stripped text was appended (e.g., "fr" + "OM" → "fr OM").

**Suppression:** Autocomplete dismisses after `;` to prevent suggestions on completed statements.

## Key Patterns & Gotchas

- **Query execution is async:** `tea.Batch()` sends `QueryStartedMsg` immediately, then `QueryResultMsg` when the goroutine completes. 5-minute context timeout on all queries.
- **Ctrl+Enter not portable:** Most terminals (macOS Terminal.app, etc.) cannot distinguish Ctrl+Enter from Enter. Use F5 or Ctrl+G as reliable alternatives for executing queries.
- **Editor Focus():** Must be called explicitly after creating a new editor — `textarea` defaults to blurred state and silently drops all input when blurred.
- **Editor InsertText():** Appends at end, not at cursor position (textarea library limitation). `ReplaceWord()` handles autocomplete replacement.
- **Syntax highlighting:** Chroma tokenization runs on every `View()` call in blurred mode. No caching.
- **DSN auto-detection:** `detectAdapter()` in main.go uses protocol prefixes and file extensions. Ambiguous DSNs default to PostgreSQL.
- **History:** SQLite-backed (`~/.config/gotermsql/history.db`). `hist.Close()` must be called on shutdown.
- **pgtype.Numeric:** pgx v5 returns `pgtype.Numeric` for PostgreSQL numeric/decimal columns. The `valueToString()` function handles this via `val.Value()` — if adding new pgx type conversions, add cases before the `default` fallback.
- **Help overlay:** Full-screen, blocks all key input when visible. Closed by `?`, `F1`, `Esc`, or `q`.
