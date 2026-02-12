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

Version info is injected via LDFLAGS from git tags/commit/date.

## Architecture

Bubble Tea (Elm Architecture) TUI. Root model in `internal/app/app.go`.

**Message routing priority in `Update()`:**
1. `tea.WindowSizeMsg` → recalculate layout
2. `tea.KeyMsg` → connection manager (if visible) → autocomplete (if visible) → global keys → focused pane handler
3. Application messages: Connect, SchemaLoaded, QueryResult, NewTab, etc.

**Focus system:** Three panes (`PaneSidebar`, `PaneEditor`, `PaneResults`). Tab/Shift+Tab cycles. Alt+1/2/3 jumps directly. Each pane's component has `Focus()`/`Blur()` methods that control whether it processes input.

**Multi-tab state:** `tabStates map[int]*TabState` — each tab owns its own `editor.Model` and `results.Model`. Always nil-check `activeTabState()`.

## Adapter Pattern

`internal/adapter/adapter.go` defines the `Adapter` and `Connection` interfaces. Each database package registers itself via `init()`:

```go
func init() { adapter.Register(&Adapter{}) }
```

Imported as blank imports in `cmd/gotermsql/main.go` to trigger registration.

**DuckDB conditional compilation:** `duckdb_enabled.go` (`//go:build duckdb`) has the real implementation; `duckdb_disabled.go` (`//go:build !duckdb`) registers a stub that returns "not compiled in" errors. Both files exist so the code compiles with or without the tag.

## Key Patterns & Gotchas

- **Query execution is async:** `tea.Batch()` sends `QueryStartedMsg` immediately, then `QueryResultMsg` when the goroutine completes. 5-minute context timeout on all queries.
- **Autocomplete overlay rendering:** Overlays within the editor's allocated space by replacing bottom lines (not appending), to avoid pushing content off-screen. Max 5 visible items.
- **Completion context detection:** Engine parses the word before cursor to determine context (FROM → tables, SELECT → columns+functions, dot notation → columns of specific table). Word-break chars differ between `completion.go` and `autocomplete.go` — the dot (`.`) is a break char in autocomplete but triggers qualified lookup in completion.
- **Editor Focus():** Must be called explicitly after creating a new editor — `textarea` defaults to blurred state and silently drops all input when blurred.
- **Editor InsertText():** Appends at end, not at cursor position (textarea library limitation).
- **Syntax highlighting:** Chroma tokenization runs on every `View()` call in blurred mode. No caching.
- **DSN auto-detection:** `detectAdapter()` in main.go uses protocol prefixes and file extensions. Ambiguous DSNs default to PostgreSQL.
- **History:** SQLite-backed (`~/.config/gotermsql/history.db`). `hist.Close()` must be called on shutdown.
