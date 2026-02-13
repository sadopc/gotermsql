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

Version info is injected via LDFLAGS from git tags/commit/date. Releases use `make build-all` + `gh release create` (targets: linux/darwin amd64+arm64, windows amd64). Homebrew tap at `sadopc/homebrew-tap`. Release archives follow naming `gotermsql_X.Y.Z_{os}_{arch}.tar.gz`.

## Architecture

Bubble Tea (Elm Architecture) TUI. Root model in `internal/app/app.go`.

**Message routing priority in `Update()`:**
1. `tea.WindowSizeMsg` → recalculate layout
2. `tea.KeyMsg` → connection manager (if visible, blocks all input) → help overlay (if visible, blocks all input) → autocomplete (if visible) → global keys → focused pane handler
3. Application messages: Connect, SchemaLoaded, QueryResult, NewTab, etc.

**Focus system:** Three panes (`PaneSidebar`, `PaneEditor`, `PaneResults`). Tab/Shift+Tab cycles. Alt+1/2/3 jumps directly. Each pane's component has `Focus()`/`Blur()` methods that control whether it processes input.

**Multi-tab state:** `tabStates map[int]*TabState` — each tab owns its own `editor.Model` and `results.Model`. Always nil-check `activeTabState()`. `results.New(tabID)` takes the tab ID for message routing.

**Layout system:** Tab bar (top) + status bar (bottom) + main area. Main area splits into sidebar (left, fixed width) + editor/results (right, percentage-based vertical split). Resizable via Ctrl+Arrow keys. Sidebar: 15–50% width. Editor height: 20–80%.

**Border width accounting:** lipgloss `.Width(w)` sets *content* width; borders add 2 chars on top. All components must use `.Width(m.width - 2)` to fit within their allocated space.

**Message re-export layer:** All message types live in `internal/msg/msg.go`. The file `internal/app/messages.go` re-exports them as type aliases for convenience within the app package. When adding new messages, update both files.

## Async Message Flow & Staleness Guards

Query execution and schema loading are async (goroutines returning `tea.Cmd`). Three generation/ID counters prevent stale results:

**`TabState.RunID uint64`** — per-tab query execution counter. Incremented in `executeQuery()` before dispatching. `QueryStartedMsg`, `QueryResultMsg`, and `QueryErrMsg` all carry `RunID`. Handlers discard messages where `msg.RunID != ts.RunID`.

**`Model.connGen uint64`** — connection generation counter. Incremented in `ConnectMsg` handler. All async messages carry `ConnGen`: `SchemaLoadedMsg`, `SchemaErrMsg`, `QueryStartedMsg`, `QueryResultMsg`, `QueryErrMsg`. Handlers discard messages where `msg.ConnGen != m.connGen`.

**`Model.executingTabID int`** — tracks which tab has the in-flight query. When a tab is closed while executing (`CloseTabMsg`), the query is cancelled and `m.executing` is cleared. When `QueryResultMsg`/`QueryErrMsg` arrives for a closed tab (`ts == nil`), `m.executing` is still cleared if `msg.TabID == m.executingTabID`.

**`QueryStreamingMsg`** also carries `ConnGen` and `RunID`. Its handler must close the iterator on stale/mismatched messages to avoid resource leaks.

**When adding new async messages:** Always capture the relevant generation counter at dispatch time and check it in the handler. The closure in `tea.Cmd` functions must capture the counter value, not a pointer to the model.

## Connection Lifecycle

- **Connect:** `connect()` returns a `tea.Cmd` that opens connection + pings. On success, sends `ConnectMsg`.
- **Reconnect:** `ConnectMsg` handler closes old `m.conn`, cancels in-flight schema load (`m.schemaCancel()`), assigns new connection, increments `connGen`.
- **Shutdown:** `main.go` calls `m.Connection()` on the final model and closes it. History DB is closed via `defer hist.Close()` (panic-safe).
- **Query cancellation:** `executeQuery()` creates a cancellable context and stores cancel in `m.cancelFunc`. For streaming SELECTs, the context has no timeout (iterator may be browsed for hours); for non-streaming queries, a 5-minute timeout is applied. Ctrl+C calls both `m.cancelFunc()` (cancels context) and `m.conn.Cancel()` (database-level cancellation).
- **Schema loading:** `loadSchema()` uses `context.WithTimeout(30s)`. Cancel func stored in `m.schemaCancel`; previous load cancelled on reconnect or quit.

## Adapter Pattern

`internal/adapter/adapter.go` defines the `Adapter` and `Connection` interfaces. Each database package registers itself via `init()`:

```go
func init() { adapter.Register(&Adapter{}) }
```

Imported as blank imports in `cmd/gotermsql/main.go` to trigger registration.

**`Connection.Databases()` contract:** Must return `[]schema.Database` with `Schemas` and `Tables` populated for the connected database. PostgreSQL can only introspect the current database via `information_schema`; other databases appear as names only. SQLite returns a single database with `"main"` schema.

**`BatchIntrospector` interface (optional):** Connections can implement `AllColumns()`, `AllIndexes()`, `AllForeignKeys()` methods that return `map[tableName][]T` for an entire schema in a single query each. `loadSchema()` type-asserts for this interface and uses batch methods when available (3 queries per schema vs 3×N per table). PostgreSQL and MySQL both implement it.

**DuckDB conditional compilation:** `duckdb_enabled.go` (`//go:build duckdb`) has the real implementation; `duckdb_disabled.go` (`//go:build !duckdb`) registers a stub that returns "not compiled in" errors. Both files exist so the code compiles with or without the tag.

## Autocomplete System

Two layers with different word-break rules:

- **`internal/completion/completion.go`** (Engine): Determines context from SQL text (FROM → tables, SELECT → columns+functions, dot → qualified columns). Thread-safe with `sync.RWMutex`. Dot is NOT a word break here (enables `table.column` lookup). Fuzzy matching ranks candidates.
- **`internal/ui/autocomplete/autocomplete.go`** (UI Model): Manages the visible dropdown. Dot IS a word break here (for prefix extraction). Sends `SelectedMsg{Text, PrefixLen}` — the full label plus how many chars to replace.

**Accepting completions:** The app calls `editor.ReplaceWord(text, prefixLen)` which removes the typed prefix from the end and appends the full completion.

**Suppression:** Autocomplete dismisses after `;` to prevent suggestions on completed statements.

## Results Table & Export

**Column sizing (`autoSizeColumns`):** Samples up to 100 rows to estimate content widths, caps at 50 chars per column, scales proportionally when total exceeds terminal width. `SetSize()` caches dimensions and early-returns when unchanged to avoid recalculating every render frame.

**bubbles/table has zero gap between columns.** All spacing comes from the Cell style's `Padding(0, 1)` (1 char left + 1 right). The width calculation accounts for `numCols * 2` padding overhead. When modifying theme `ResultsCell`, always include `Padding(0, 1)` or columns will run together.

**Iterator lifecycle:** `SetResults()` and `SetIterator()` both close the previous iterator before replacing. Never set `m.iterator = nil` without closing first.

**Pagination routing:** `FetchedPageMsg` (exported) carries `TabID`. The `fetchNextPage()`/`fetchPrevPage()` functions embed the tab's ID. The app routes `FetchedPageMsg` to the correct tab's `Results.Update(msg)` in its main Update switch.

**Streaming SELECT queries:** `executeQuery()` uses `adapter.IsSelectQuery()` to detect row-returning statements (SELECT, WITH, EXPLAIN, SHOW, DESCRIBE, PRAGMA, etc.). For these, it calls `conn.ExecuteStreaming()` first, returning a `QueryStreamingMsg` with a `RowIterator`. If streaming fails, it falls back to `conn.Execute()`. Non-SELECT statements always use `Execute()`. The `QueryStreamingMsg` handler wires the iterator into `results.Model` via `SetIterator()` + `FetchFirstPage()`.

**Sliding window buffer:** `maxBufferedRows = 5000` in `results.go`. When streaming pages push past this limit, the oldest rows are trimmed from the front. This keeps memory constant regardless of result set size (verified: 2 MB overhead for 10M rows).

**Export (`internal/ui/results/exporter.go`):** Four functions — `ExportCSV`/`ExportJSON` for in-memory rows, `ExportCSVFromIterator`/`ExportJSONFromIterator` for streaming large result sets. Ctrl+E triggers in-memory CSV export to `export_<timestamp>.csv` in the working directory.

## Status Bar

**Auto-clear timer:** After query results, errors, or status messages appear, the status bar reverts to key hints after 5 seconds via `ClearStatusMsg` + `tea.Tick`.

**Command propagation:** When calling `m.statusbar.Update(msg)`, always capture and append the returned `tea.Cmd` — the statusbar returns timer commands that must reach the Bubble Tea runtime.

## Connection Manager & Config Persistence

**Persisting changes:** `ConnectionsUpdatedMsg` is sent by the connection manager on Ctrl+S (save) and `d` (delete). The app handles it by updating `m.cfg.Connections` and calling `m.cfg.SaveDefault()`.

**Atomic config writes:** `Config.Save()` writes to a temp file in the same directory, then `os.Rename()` for crash-safe atomicity. Temp file is cleaned up on any error.

**DSN credential escaping:** `SavedConnection.BuildDSN()` uses `url.UserPassword()` for postgres (handles all special chars) and `url.QueryEscape()` for mysql passwords. The `main.go` `buildDSN()` function mirrors this.

**Config/history permissions:** Directories created with `0o700`, files with `0o600` (config may contain passwords).

## Theme System

Three themes in `internal/theme/theme.go`: `"default"` (dark), `"light"`, `"monokai"`. `theme.Current` is a global pointer used by all components. When adding styles to themes, add to all three variants.

## Key Patterns & Gotchas

- **Query execution is async:** `tea.Batch()` sends `QueryStartedMsg` immediately, then `QueryResultMsg` or `QueryStreamingMsg` when the goroutine completes. Streaming SELECTs have no timeout; non-streaming queries have a 5-minute timeout.
- **Nil guards on async handlers:** Always check both `ts != nil` (tab may be closed) and `m.conn != nil` (may be disconnected) before accessing tab state or connection in async message handlers. When `ts == nil`, still clear `m.executing` if `msg.TabID == m.executingTabID`.
- **Error sanitization:** `sanitizeError()` strips credentials from DSN URLs in error messages (e.g., `postgres://user:pass@` → `postgres://***@`). Applied in `ConnectErrMsg` handler and connmgr test result display. Defined separately in both `internal/app/` and `internal/ui/connmgr/` packages.
- **Ctrl+Enter not portable:** Most terminals cannot distinguish Ctrl+Enter from Enter. Use F5 or Ctrl+G as reliable alternatives.
- **Editor Focus():** Must be called explicitly after creating a new editor — `textarea` defaults to blurred state and silently drops all input when blurred.
- **Editor InsertText():** Appends at end, not at cursor position (textarea library limitation). `ReplaceWord()` handles autocomplete replacement.
- **Syntax highlighting:** Chroma tokenization runs on every `View()` call in blurred mode. No caching.
- **DSN auto-detection:** `detectAdapter()` in main.go uses protocol prefixes and file extensions. Ambiguous DSNs default to PostgreSQL.
- **History:** SQLite-backed (`~/.config/gotermsql/history.db`). Closed via `defer` in main.
- **pgtype.Numeric:** pgx v5 returns `pgtype.Numeric` for PostgreSQL numeric/decimal columns. The `valueToString()` function handles this via `val.Value()` — if adding new pgx type conversions, add cases before the `default` fallback.
- **Help overlay:** Full-screen, blocks all key input when visible. Closed by `?`, `F1`, `Esc`, or `q`.
- **Schema load warnings:** Introspection errors (per-table or batch) are collected as warnings. If any exist, "Schema loaded with N warnings" appears in the status bar.
