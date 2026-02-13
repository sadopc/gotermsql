package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Theme       string            `yaml:"theme"`
	KeyMode     string            `yaml:"keymode"` // "vim" or "standard"
	Editor      EditorConfig      `yaml:"editor"`
	Results     ResultsConfig     `yaml:"results"`
	Connections []SavedConnection `yaml:"connections"`
}

// EditorConfig holds editor-related settings.
type EditorConfig struct {
	TabSize         int  `yaml:"tab_size"`
	ShowLineNumbers bool `yaml:"show_line_numbers"`
}

// ResultsConfig holds result display settings.
type ResultsConfig struct {
	PageSize       int `yaml:"page_size"`
	MaxColumnWidth int `yaml:"max_column_width"`
}

// SavedConnection holds parameters for a saved database connection.
type SavedConnection struct {
	Name     string `yaml:"name"`
	Adapter  string `yaml:"adapter"`
	DSN      string `yaml:"dsn,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	File     string `yaml:"file,omitempty"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Theme:   "default",
		KeyMode: "standard",
		Editor: EditorConfig{
			TabSize:         4,
			ShowLineNumbers: true,
		},
		Results: ResultsConfig{
			PageSize:       1000,
			MaxColumnWidth: 50,
		},
	}
}

// ConfigDir returns the gotermsql configuration directory path.
// It uses os.UserConfigDir to locate the base config directory and
// appends "gotermsql" to it, typically resulting in ~/.config/gotermsql/.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(base, "gotermsql"), nil
}

// Load reads a Config from the YAML file at path. If the file does not exist,
// it returns DefaultConfig without error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// LoadDefault loads configuration from the default path
// (ConfigDir()/config.yaml).
func LoadDefault() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	return Load(filepath.Join(dir, "config.yaml"))
}

// Save writes the Config to the YAML file at path atomically, creating any
// necessary parent directories. It writes to a temp file first and renames
// to avoid corruption on crash.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Write to a temp file in the same directory, then rename for atomicity.
	tmp, err := os.CreateTemp(dir, ".config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// SaveDefault writes the Config to the default path
// (ConfigDir()/config.yaml).
func (c *Config) SaveDefault() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return c.Save(filepath.Join(dir, "config.yaml"))
}

// BuildDSN constructs a connection string from the individual fields of a
// SavedConnection. If DSN is already set, it is returned as-is. For
// file-based adapters (sqlite, duckdb) it returns the File field. For
// postgres it builds a proper postgres:// URL with escaped credentials. For
// mysql it builds the go-sql-driver format with escaped password.
func (sc *SavedConnection) BuildDSN() string {
	if sc.DSN != "" {
		return sc.DSN
	}

	adapterName := strings.ToLower(sc.Adapter)
	if adapterName == "sqlite" || adapterName == "duckdb" {
		return sc.File
	}

	host := sc.Host
	if host == "" {
		host = "localhost"
	}

	switch adapterName {
	case "postgres":
		u := &url.URL{Scheme: "postgres", Host: host}
		if sc.Port > 0 {
			u.Host = fmt.Sprintf("%s:%d", host, sc.Port)
		}
		if sc.User != "" {
			if sc.Password != "" {
				u.User = url.UserPassword(sc.User, sc.Password)
			} else {
				u.User = url.User(sc.User)
			}
		}
		if sc.Database != "" {
			u.Path = "/" + sc.Database
		}
		return u.String()

	case "mysql":
		var b strings.Builder
		if sc.User != "" {
			b.WriteString(sc.User)
			if sc.Password != "" {
				b.WriteByte(':')
				b.WriteString(url.QueryEscape(sc.Password))
			}
			b.WriteByte('@')
		}
		port := sc.Port
		if port == 0 {
			port = 3306
		}
		fmt.Fprintf(&b, "tcp(%s:%d)", host, port)
		if sc.Database != "" {
			b.WriteByte('/')
			b.WriteString(sc.Database)
		}
		return b.String()

	default:
		// Generic fallback
		var b strings.Builder
		if sc.User != "" {
			b.WriteString(sc.User)
			if sc.Password != "" {
				b.WriteByte(':')
				b.WriteString(sc.Password)
			}
			b.WriteByte('@')
		}
		b.WriteString(host)
		if sc.Port > 0 {
			fmt.Fprintf(&b, ":%d", sc.Port)
		}
		if sc.Database != "" {
			b.WriteByte('/')
			b.WriteString(sc.Database)
		}
		return b.String()
	}
}

// DisplayString returns a human-readable representation of the connection,
// formatted as "adapter://host:port/database" for network adapters or
// "adapter://file" for file-based adapters.
func (sc *SavedConnection) DisplayString() string {
	adapter := strings.ToLower(sc.Adapter)
	if adapter == "sqlite" || adapter == "duckdb" {
		file := sc.File
		if file == "" {
			file = sc.DSN
		}
		return fmt.Sprintf("%s://%s", sc.Adapter, file)
	}

	host := sc.Host
	if host == "" {
		host = "localhost"
	}

	var location string
	if sc.Port > 0 {
		location = fmt.Sprintf("%s:%d", host, sc.Port)
	} else {
		location = host
	}

	db := sc.Database
	if db != "" {
		return fmt.Sprintf("%s://%s/%s", sc.Adapter, location, db)
	}
	return fmt.Sprintf("%s://%s", sc.Adapter, location)
}
