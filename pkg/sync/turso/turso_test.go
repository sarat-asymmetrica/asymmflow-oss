package turso

import (
	"path/filepath"
	"testing"
)

func TestNewLocalOnly(t *testing.T) {
	client := newLocalClient(t)
	if client.Mode() != "local" {
		t.Fatalf("mode = %q, want local", client.Mode())
	}
	if client.DB() == nil {
		t.Fatalf("DB() returned nil")
	}
}

func TestLocalDBReadWrite(t *testing.T) {
	client := newLocalClient(t)

	if _, err := client.DB().Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := client.DB().Exec("INSERT INTO items (name) VALUES (?)", "alpha"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var name string
	if err := client.DB().QueryRow("SELECT name FROM items WHERE id = 1").Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "alpha" {
		t.Fatalf("name = %q, want alpha", name)
	}
}

func TestModeDetection(t *testing.T) {
	tests := []struct {
		cfg  Config
		want string
	}{
		{Config{LocalPath: "local.db", RemoteURL: "libsql://example.turso.io"}, "remote"},
		{Config{RemoteURL: "libsql://example.turso.io"}, "remote"},
		{Config{LocalPath: "local.db"}, "local"},
		{Config{}, ""},
	}

	for _, tt := range tests {
		if got := ModeForConfig(tt.cfg); got != tt.want {
			t.Fatalf("ModeForConfig(%+v) = %q, want %q", tt.cfg, got, tt.want)
		}
	}
}

func newLocalClient(t *testing.T) *Client {
	t.Helper()

	path := filepath.Join(t.TempDir(), "local.db")
	client, err := New(Config{LocalPath: path})
	if err != nil {
		t.Fatalf("New(local): %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("Close(): %v", err)
		}
	})
	return client
}
