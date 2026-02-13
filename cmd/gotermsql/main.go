package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/sadopc/gotermsql/internal/adapter"
	"github.com/sadopc/gotermsql/internal/app"
	"github.com/sadopc/gotermsql/internal/audit"
	"github.com/sadopc/gotermsql/internal/config"
	"github.com/sadopc/gotermsql/internal/history"

	// Register database adapters
	_ "github.com/sadopc/gotermsql/internal/adapter/duckdb"
	_ "github.com/sadopc/gotermsql/internal/adapter/mysql"
	_ "github.com/sadopc/gotermsql/internal/adapter/postgres"
	_ "github.com/sadopc/gotermsql/internal/adapter/sqlite"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		adapterFlag  string
		hostFlag     string
		portFlag     int
		userFlag     string
		passwordFlag string
		databaseFlag string
		fileFlag     string
		configFlag   string
	)

	rootCmd := &cobra.Command{
		Use:   "gotermsql [dsn]",
		Short: "A terminal SQL IDE",
		Long: `gotermsql is a full-featured terminal SQL IDE supporting
PostgreSQL, MySQL, SQLite, and DuckDB.

Examples:
  gotermsql                                    # Launch connection manager
  gotermsql postgres://user:pass@host/db       # Connect via DSN
  gotermsql --adapter sqlite --file ./data.db  # SQLite file
  gotermsql --adapter mysql -h localhost -u root -d mydb`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config
			var cfg *config.Config
			var err error
			if configFlag != "" {
				cfg, err = config.Load(configFlag)
			} else {
				cfg, err = config.LoadDefault()
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
				cfg = config.DefaultConfig()
			}

			// Open history
			hist, err := history.New()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open history: %v\n", err)
			}
			if hist != nil {
				defer hist.Close()
			}

			// Open audit log
			var auditLog *audit.Logger
			if cfg.Audit.Enabled {
				auditPath := cfg.Audit.Path
				if auditPath == "" {
					if dir, err := config.ConfigDir(); err == nil {
						auditPath = dir + "/audit.jsonl"
					}
				}
				if auditPath != "" {
					auditLog, err = audit.New(auditPath, cfg.Audit.MaxSizeMB)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not open audit log: %v\n", err)
					}
				}
			}
			if auditLog != nil {
				defer auditLog.Close()
			}

			// Create app model
			model := app.New(cfg, hist, auditLog)

			// Determine connection method
			var dsn string
			var adapterName string

			if len(args) > 0 {
				dsn = args[0]
				adapterName = detectAdapter(dsn)
			}

			if adapterFlag != "" {
				adapterName = adapterFlag
			}

			// Build DSN from individual flags if no DSN provided
			if dsn == "" && adapterName != "" {
				dsn = buildDSN(adapterName, hostFlag, portFlag, userFlag, passwordFlag, databaseFlag, fileFlag)
			}

			// If we have connection info, connect; otherwise show connection manager
			var initCmd tea.Cmd
			if adapterName != "" && dsn != "" {
				// Validate adapter exists
				if _, ok := adapter.Registry[adapterName]; !ok {
					return fmt.Errorf("unknown adapter: %s (available: %s)", adapterName, availableAdapters())
				}
				initCmd = model.InitialConnect(adapterName, dsn)
			} else {
				model.ShowConnManager()
			}

			// Run the TUI
			p := tea.NewProgram(
				model,
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)

			if initCmd != nil {
				go func() {
					p.Send(initCmd())
				}()
			}

			finalModel, err := p.Run()
			if err != nil {
				return fmt.Errorf("error running application: %w", err)
			}

			// Close database connection if open
			if m, ok := finalModel.(app.Model); ok {
				if conn := m.Connection(); conn != nil {
					_ = conn.Close()
				}
			}

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&adapterFlag, "adapter", "a", "", "Database adapter (postgres, mysql, sqlite, duckdb)")
	rootCmd.Flags().StringVarP(&hostFlag, "host", "H", "localhost", "Database host")
	rootCmd.Flags().IntVarP(&portFlag, "port", "p", 0, "Database port")
	rootCmd.Flags().StringVarP(&userFlag, "user", "u", "", "Database user")
	rootCmd.Flags().StringVarP(&passwordFlag, "password", "P", "", "Database password")
	rootCmd.Flags().StringVarP(&databaseFlag, "database", "d", "", "Database name")
	rootCmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Database file (for SQLite/DuckDB)")
	rootCmd.Flags().StringVarP(&configFlag, "config", "c", "", "Config file path")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gotermsql %s (commit: %s, built: %s)\n", version, commit, date)
			fmt.Println("\nSupported adapters:")
			for name := range adapter.Registry {
				fmt.Printf("  - %s\n", name)
			}
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func detectAdapter(dsn string) string {
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://"):
		return "postgres"
	case strings.HasPrefix(lower, "mysql://"):
		return "mysql"
	case strings.HasPrefix(lower, "sqlite://") || strings.HasPrefix(lower, "file:"):
		return "sqlite"
	case strings.HasPrefix(lower, "duckdb://"):
		return "duckdb"
	case strings.HasSuffix(lower, ".db") || strings.HasSuffix(lower, ".sqlite") || strings.HasSuffix(lower, ".sqlite3"):
		return "sqlite"
	case strings.HasSuffix(lower, ".duckdb"):
		return "duckdb"
	case strings.Contains(lower, "@tcp("):
		return "mysql"
	}
	// Default: try as PostgreSQL DSN
	if strings.Contains(dsn, "@") {
		return "postgres"
	}
	return ""
}

func buildDSN(adapterName, host string, port int, user, password, database, file string) string {
	switch adapterName {
	case "postgres":
		u := &url.URL{
			Scheme: "postgres",
			Host:   host,
		}
		if user != "" {
			if password != "" {
				u.User = url.UserPassword(user, password)
			} else {
				u.User = url.User(user)
			}
		}
		if port > 0 {
			u.Host = fmt.Sprintf("%s:%d", host, port)
		}
		if database != "" {
			u.Path = "/" + database
		}
		return u.String()

	case "mysql":
		// go-sql-driver format: user:pass@tcp(host:port)/db
		dsn := ""
		if user != "" {
			dsn += user
			if password != "" {
				dsn += ":" + url.PathEscape(password)
			}
			dsn += "@"
		}
		p := port
		if p == 0 {
			p = 3306
		}
		dsn += fmt.Sprintf("tcp(%s:%d)", host, p)
		if database != "" {
			dsn += "/" + database
		}
		return dsn

	case "sqlite":
		if file != "" {
			return file
		}
		if database != "" {
			return database
		}
		return ":memory:"

	case "duckdb":
		if file != "" {
			return file
		}
		if database != "" {
			return database
		}
		return ":memory:"
	}
	return ""
}

func availableAdapters() string {
	var names []string
	for name := range adapter.Registry {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
