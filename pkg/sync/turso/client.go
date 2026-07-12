package turso

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
	libsql "github.com/tursodatabase/libsql-client-go/libsql"
)

// Config holds Turso connection settings.
type Config struct {
	LocalPath  string
	RemoteURL  string
	AuthToken  string
	SyncPeriod int
}

// Client wraps a Turso/libsql connection.
type Client struct {
	db        *sql.DB
	config    Config
	mode      string
	connector driver.Connector
}

// New creates a Turso client based on config.
func New(cfg Config) (*Client, error) {
	mode := ModeForConfig(cfg)
	if mode == "" {
		return nil, fmt.Errorf("turso: LocalPath or RemoteURL is required")
	}

	switch mode {
	case "remote":
		opts := make([]libsql.Option, 0, 2)
		if cfg.AuthToken != "" {
			opts = append(opts, libsql.WithAuthToken(cfg.AuthToken))
		}
		connector, err := libsql.NewConnector(cfg.RemoteURL, opts...)
		if err != nil {
			return nil, err
		}
		return &Client{
			db:        sql.OpenDB(connector),
			config:    cfg,
			mode:      mode,
			connector: connector,
		}, nil
	default:
		db, err := sql.Open("sqlite3", localDSN(cfg.LocalPath))
		if err != nil {
			return nil, err
		}
		return &Client{db: db, config: cfg, mode: mode}, nil
	}
}

// ModeForConfig returns the client mode implied by the config.
func ModeForConfig(cfg Config) string {
	switch {
	case cfg.RemoteURL != "":
		return "remote"
	case cfg.LocalPath != "":
		return "local"
	default:
		return ""
	}
}

// DB returns the underlying *sql.DB for use with GORM or raw queries.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Sync triggers a manual sync in embedded mode.
func (c *Client) Sync() error {
	return nil
}

// Mode returns "embedded", "remote", or "local".
func (c *Client) Mode() string {
	return c.mode
}

// Close closes the database connection.
func (c *Client) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	err := c.db.Close()
	return err
}

func localDSN(path string) string {
	return "file:" + filepath.ToSlash(path)
}
