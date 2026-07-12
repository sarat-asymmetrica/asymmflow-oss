package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func openSourceDB(t *testing.T, dir string) (*sql.DB, string) {
	t.Helper()
	dbPath := filepath.Join(dir, "sample.db")
	conn, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO items (name) VALUES ('alpha'), ('beta')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	return conn, dbPath
}

func TestBackupCreatesVerifiableSnapshot(t *testing.T) {
	dir := t.TempDir()
	conn, dbPath := openSourceDB(t, dir)

	b := &Backuper{DB: conn}
	path, err := b.Backup(time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	wantDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if filepath.Dir(path) != wantDir {
		t.Errorf("backup dir = %s, want %s", filepath.Dir(path), wantDir)
	}
	if filepath.Base(path) != "sample_20260703_120000.db" {
		t.Errorf("backup name = %s", filepath.Base(path))
	}
	if err := VerifyBackup("sqlite3", path); err != nil {
		t.Errorf("VerifyBackup: %v", err)
	}
}

func TestBackupRotationPrunesOldest(t *testing.T) {
	dir := t.TempDir()
	conn, _ := openSourceDB(t, dir)

	b := &Backuper{DB: conn, Keep: 3}
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		if _, err := b.Backup(base.Add(time.Duration(i) * time.Hour)); err != nil {
			t.Fatalf("Backup #%d: %v", i, err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("kept %d backups, want 3", len(entries))
	}
	// Oldest two (00:00, 01:00) pruned; earliest kept is 02:00.
	if entries[0].Name() != "sample_20260701_020000.db" {
		t.Errorf("earliest kept = %s", entries[0].Name())
	}
}

func TestRestoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	conn, dbPath := openSourceDB(t, dir)

	b := &Backuper{DB: conn}
	backupPath, err := b.Backup(time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	// Mutate the live DB after the backup, then close it (restore precondition).
	if _, err := conn.Exec(`INSERT INTO items (name) VALUES ('gamma')`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	pre, err := Restore("sqlite3", dbPath, backupPath, time.Date(2026, 7, 3, 13, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if pre == "" {
		t.Fatal("expected a pre-restore snapshot path")
	}
	if _, err := os.Stat(pre); err != nil {
		t.Fatalf("pre-restore snapshot missing: %v", err)
	}

	// Restored DB has 2 rows (backup state), pre-restore snapshot has 3.
	assertCount := func(path string, want int) {
		t.Helper()
		c, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?mode=ro")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		var n int
		if err := c.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("%s: %d rows, want %d", filepath.Base(path), n, want)
		}
	}
	assertCount(dbPath, 2)
	assertCount(pre, 3)
}

func TestRestoreRefusesCorruptBackup(t *testing.T) {
	dir := t.TempDir()
	_, dbPath := openSourceDB(t, dir)

	corrupt := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(corrupt, []byte("this is not a sqlite database at all"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore("sqlite3", dbPath, corrupt, time.Now()); err == nil {
		t.Fatal("corrupt backup must be refused")
	}
}
