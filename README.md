# gotermsql

A full-featured terminal SQL IDE written in Go. Single binary, zero config, multi-database.

![demo](demo.gif)

## Features

- **Multi-database support** - PostgreSQL, MySQL, SQLite, DuckDB (optional build tag)
- **Schema browser** - Hierarchical tree view with databases, schemas, tables, columns
- **SQL editor** - Syntax highlighting, line numbers, multi-tab editing
- **Autocomplete** - Context-aware completions for tables, columns, keywords, functions
- **Results viewer** - Tabular display with row count, query timing, and export support
- **Streaming results** - SELECT queries stream via paginated iterator, keeping memory constant even for millions of rows
- **Vim keybindings** - Toggleable vim/standard mode (F2)
- **Connection manager** - Save, edit, and manage database connections
- **Query history** - SQLite-backed local history with search (Ctrl+H)
- **Export** - CSV and JSON export of query results (Ctrl+E)
- **Resizable panes** - Adjust sidebar width and editor/results split with Ctrl+Arrow keys
- **Single binary** - Pure Go, zero CGo by default, cross-platform

## Install

### Homebrew (macOS/Linux)

```bash
brew install sadopc/tap/gotermsql
```

### From source

```bash
go install github.com/sadopc/gotermsql/cmd/gotermsql@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/sadopc/gotermsql/releases) for Linux, macOS, and Windows.

### Build from repo

```bash
git clone https://github.com/sadopc/gotermsql.git
cd gotermsql
make build
# Binary is at bin/gotermsql
```

### With DuckDB support (requires CGo)

```bash
make build-full
```

## Usage

```bash
# Launch with connection manager
gotermsql

# Connect via DSN
gotermsql postgres://user:pass@localhost:5432/mydb

# SQLite file
gotermsql --adapter sqlite --file ./data.db

# MySQL with flags
gotermsql --adapter mysql -H localhost -u root -d mydb

# PostgreSQL with individual flags
gotermsql --adapter postgres -H localhost -p 5432 -u admin -d production
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus: Editor -> Results -> Sidebar |
| `Shift+Tab` | Cycle focus backwards |
| `Alt+1/2/3` | Jump to Sidebar/Editor/Results |
| `Ctrl+Arrow` | Resize panes |

### Editor

| Key | Action |
|-----|--------|
| `Ctrl+Enter` / `F5` / `Ctrl+G` | Execute query |
| `Ctrl+C` | Cancel running query |
| `Ctrl+Space` | Force autocomplete |
| `Esc` | Dismiss autocomplete |

### Tabs

| Key | Action |
|-----|--------|
| `Ctrl+T` | New tab |
| `Ctrl+W` | Close tab |
| `Ctrl+]` | Next tab |
| `Ctrl+[` | Previous tab |

### Application

| Key | Action |
|-----|--------|
| `Ctrl+Q` | Quit |
| `Ctrl+B` | Toggle sidebar |
| `Ctrl+R` | Refresh schema |
| `Ctrl+O` | Connection manager |
| `Ctrl+H` | Query history |
| `Ctrl+E` | Export results |
| `F1` | Help |
| `F2` | Toggle vim/standard mode |

## Configuration

Config file is stored at `~/.config/gotermsql/config.yaml`:

```yaml
theme: default
keymode: standard  # "vim" or "standard"
editor:
  tab_size: 4
  show_line_numbers: true
results:
  page_size: 1000
  max_column_width: 50
connections:
  - name: local-pg
    adapter: postgres
    dsn: postgres://user:pass@localhost:5432/mydb
```

## Supported Databases

| Database | Driver | CGo Required |
|----------|--------|-------------|
| PostgreSQL | [pgx](https://github.com/jackc/pgx) | No |
| MySQL | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | No |
| SQLite | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | No |
| DuckDB | [go-duckdb](https://github.com/marcboeker/go-duckdb) | Yes (build tag: `-tags duckdb`) |

Default builds are 100% pure Go with zero CGo dependencies.

## Architecture

```
gotermsql/
├── cmd/gotermsql/          # CLI entry point (cobra)
├── internal/
│   ├── adapter/            # Database adapter interface + implementations
│   │   ├── postgres/       # PostgreSQL (pgx)
│   │   ├── mysql/          # MySQL (go-sql-driver)
│   │   ├── sqlite/         # SQLite (modernc.org)
│   │   └── duckdb/         # DuckDB (optional, build tag)
│   ├── app/                # Root Bubble Tea model, keymaps, messages
│   ├── ui/                 # UI components
│   │   ├── sidebar/        # Schema tree browser
│   │   ├── editor/         # SQL editor + syntax highlighting
│   │   ├── results/        # Results table + exporter
│   │   ├── tabs/           # Tab bar
│   │   ├── statusbar/      # Status bar
│   │   ├── autocomplete/   # Autocomplete dropdown
│   │   ├── connmgr/        # Connection manager modal
│   │   └── dialog/         # Reusable dialog component
│   ├── completion/         # SQL completion engine
│   ├── schema/             # Unified schema types
│   ├── config/             # YAML config management
│   ├── history/            # Query history (SQLite-backed)
│   └── theme/              # Theme definitions (Lip Gloss)
├── Makefile
└── .goreleaser.yaml
```

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and [Bubbles](https://github.com/charmbracelet/bubbles).

## Development

```bash
# Run tests
make test

# Run with race detector
make test-race

# Build and run
make run ARGS="--adapter sqlite --file demo.db"

# Format code
make fmt

# Vet
make vet
```

## License

MIT
