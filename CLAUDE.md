# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build              # CGO_ENABLED=0, pure Go (no DuckDB)
make build-full         # CGO_ENABLED=1 -tags duckdb (requires C compiler)
make build-all          # Cross-compile all release targets (linux/darwin/windows)
make test               # go test ./...
make test-race          # go test -race ./...
make vet                # go vet ./...
make fmt                # gofmt + goimports
make lint               # golangci-lint run ./...
make tidy               # go mod tidy
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

Version info is injected via LDFLAGS from git tags/commit/date. Releases via `goreleaser` (targets: linux/darwin amd64+arm64, windows amd64). Homebrew tap at `sadopc/homebrew-tap`.

## Architecture

Bubble Tea (Elm Architecture) TUI. Root model in `internal/app/app.go`.

**Message routing priority in `Update()`:**
1. `tea.WindowSizeMsg` → recalculate layout
2. `tea.KeyMsg` → connection manager (if visible, blocks all input) → help overlay (if visible, blocks all input) → autocomplete (if visible) → global keys → focused pane handler
3. Application messages: Connect, SchemaLoaded, QueryResult, NewTab, etc.

**Focus system:** Three panes (`PaneSidebar`, `PaneEditor`, `PaneResults`). Tab/Shift+Tab cycles. Alt+1/2/3 jumps directly. Each pane's component has `Focus()`/`Blur()` methods that control whether it processes input.

**Multi-tab state:** `tabStates map[int]*TabState` — each tab owns its own `editor.Model` and `results.Model`. Always nil-check `activeTabState()`.

**Layout system:** Tab bar (top) + status bar (bottom) + main area. Main area splits into sidebar (left, fixed width) + editor/results (right, percentage-based vertical split). Resizable via Ctrl+Arrow keys. Sidebar: 15–50% width. Editor height: 20–80%.

**Border width accounting:** lipgloss `.Width(w)` sets *content* width; borders add 2 chars on top. All components must use `.Width(m.width - 2)` to fit within their allocated space.

**Message re-export layer:** All message types live in `internal/msg/msg.go`. The file `internal/app/messages.go` re-exports them as type aliases for convenience within the app package. When adding new messages, update both files.

## Async Message Flow & Staleness Guards

Query execution and schema loading are async (goroutines returning `tea.Cmd`). Two generation counters prevent stale results from overwriting newer data:

**`TabState.RunID uint64`** — per-tab query execution counter. Incremented in `executeQuery()` before dispatching. `QueryStartedMsg`, `QueryResultMsg`, and `QueryErrMsg` all carry `RunID`. Handlers discard messages where `msg.RunID != ts.RunID`. Prevents slow query A from overwriting fast query B's results on the same tab.

**`Model.connGen uint64`** — connection generation counter. Incremented in `ConnectMsg` handler. `SchemaLoadedMsg` and `SchemaErrMsg` carry `ConnGen`. Handlers discard messages where `msg.ConnGen != m.connGen`. Prevents old connection's schema from overwriting after reconnect.

**When adding new async messages:** Always capture the relevant generation counter at dispatch time and check it in the handler. The closure in `tea.Cmd` functions must capture the counter value, not a pointer to the model.

## Connection Lifecycle

- **Connect:** `connect()` returns a `tea.Cmd` that opens connection + pings. On success, sends `ConnectMsg`.
- **Reconnect:** `ConnectMsg` handler closes old `m.conn` before assigning new one, increments `connGen`.
- **Shutdown:** `main.go` calls `m.Connection()` on the final model and closes it. History DB is also closed.
- **Query cancellation:** `executeQuery()` creates a context with 5-minute timeout and stores cancel in `m.cancelFunc`. Ctrl+C calls both `m.cancelFunc()` (cancels context) and `m.conn.Cancel()` (database-level cancellation).

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

**Accepting completions:** The app calls `editor.ReplaceWord(text, prefixLen)` which removes the typed prefix from the end and appends the full completion.

**Suppression:** Autocomplete dismisses after `;` to prevent suggestions on completed statements.

## Results Table & Export

**Column sizing (`autoSizeColumns`):** Samples up to 100 rows to estimate content widths, caps at 50 chars per column, scales proportionally when total exceeds terminal width.

**bubbles/table has zero gap between columns.** All spacing comes from the Cell style's `Padding(0, 1)` (1 char left + 1 right). The width calculation accounts for `numCols * 2` padding overhead. When modifying theme `ResultsCell`, always include `Padding(0, 1)` or columns will run together.

**Iterator lifecycle:** `SetResults()` and `SetIterator()` both close the previous iterator before replacing. Never set `m.iterator = nil` without closing first.

**Export (`internal/ui/results/exporter.go`):** Four functions — `ExportCSV`/`ExportJSON` for in-memory rows, `ExportCSVFromIterator`/`ExportJSONFromIterator` for streaming large result sets. Ctrl+E triggers in-memory CSV export to `export_<timestamp>.csv` in the working directory.

## Status Bar

**Auto-clear timer:** After query results, errors, or status messages appear, the status bar reverts to key hints after 5 seconds via `ClearStatusMsg` + `tea.Tick`.

**Command propagation:** When calling `m.statusbar.Update(msg)`, always capture and append the returned `tea.Cmd` — the statusbar returns timer commands that must reach the Bubble Tea runtime.

## Theme System

Three themes in `internal/theme/theme.go`: `"default"` (dark), `"light"`, `"monokai"`. `theme.Current` is a global pointer used by all components. When adding styles to themes, add to all three variants.

## Key Patterns & Gotchas

- **Query execution is async:** `tea.Batch()` sends `QueryStartedMsg` immediately, then `QueryResultMsg` when the goroutine completes. 5-minute context timeout on all queries.
- **Nil guards on async handlers:** Always check both `ts != nil` (tab may be closed) and `m.conn != nil` (may be disconnected) before accessing tab state or connection in async message handlers.
- **Ctrl+Enter not portable:** Most terminals cannot distinguish Ctrl+Enter from Enter. Use F5 or Ctrl+G as reliable alternatives.
- **Editor Focus():** Must be called explicitly after creating a new editor — `textarea` defaults to blurred state and silently drops all input when blurred.
- **Editor InsertText():** Appends at end, not at cursor position (textarea library limitation). `ReplaceWord()` handles autocomplete replacement.
- **Syntax highlighting:** Chroma tokenization runs on every `View()` call in blurred mode. No caching.
- **DSN auto-detection:** `detectAdapter()` in main.go uses protocol prefixes and file extensions. Ambiguous DSNs default to PostgreSQL. DSN building uses `net/url` for proper escaping of special characters in credentials.
- **History:** SQLite-backed (`~/.config/gotermsql/history.db`). `hist.Close()` must be called on shutdown.
- **Config/history permissions:** Directories created with `0o700`, files with `0o600` (config may contain passwords).
- **pgtype.Numeric:** pgx v5 returns `pgtype.Numeric` for PostgreSQL numeric/decimal columns. The `valueToString()` function handles this via `val.Value()` — if adding new pgx type conversions, add cases before the `default` fallback.
- **Help overlay:** Full-screen, blocks all key input when visible. Closed by `?`, `F1`, `Esc`, or `q`.
