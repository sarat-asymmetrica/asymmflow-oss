// Package db provides database infrastructure capabilities: atomic SQLite
// backup with rotation, and verified restore.
//
// Promoted from package main (database.go backupDatabaseInternal /
// pruneOldBackups) as Wave 2 Mission A engine-promotion work, with two
// generalizations: the filename stem derives from the actual database file
// (no hardcoded product name), and Restore — which package main never had —
// verifies the backup's integrity before touching anything and snapshots the
// current database first.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultKeep is the default backup-rotation depth.
const DefaultKeep = 7

// Backuper takes atomic backups of a live SQLite database via VACUUM INTO.
type Backuper struct {
	// DB is the open database handle (any connection to the target database).
	DB *sql.DB

	// Dir is the backup directory. Blank means "backups/" next to the
	// database file.
	Dir string

	// Keep is how many backups to retain (oldest pruned). <=0 means DefaultKeep.
	Keep int
}

// DatabasePath resolves the file path of the main database via
// PRAGMA database_list. Returns an error for in-memory or unresolvable
// databases.
func DatabasePath(sqlDB *sql.DB) (string, error) {
	if sqlDB == nil {
		return "", errors.New("db: nil database handle")
	}
	var seq int
	var name, path string
	if err := sqlDB.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
		return "", fmt.Errorf("db: cannot resolve database path: %w", err)
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("db: database has no file path (in-memory?)")
	}
	return path, nil
}

// Backup creates an atomic, consistent snapshot using VACUUM INTO, restricts
// its permissions, prunes old backups beyond Keep, and returns the backup
// file path. The backup file is named "<stem>_<timestamp>.db" where stem is
// the database's own filename.
func (b *Backuper) Backup(now time.Time) (string, error) {
	if b == nil || b.DB == nil {
		return "", errors.New("db: backuper has no database")
	}
	dbPath, err := DatabasePath(b.DB)
	if err != nil {
		return "", err
	}
	stem := strings.TrimSuffix(filepath.Base(dbPath), filepath.Ext(dbPath))

	backupDir := b.Dir
	if strings.TrimSpace(backupDir) == "" {
		backupDir = filepath.Join(filepath.Dir(dbPath), "backups")
	}
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("db: cannot create backup directory: %w", err)
	}

	timestamp := now.Format("20060102_150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s_%s.db", stem, timestamp))

	// VACUUM INTO is atomic and consistent. Escape single quotes in the path
	// to prevent malformed SQL (e.g. /Users/O'Brien/...).
	escapedPath := strings.ReplaceAll(backupPath, "'", "''")
	if _, err := b.DB.Exec(fmt.Sprintf("VACUUM INTO '%s'", escapedPath)); err != nil {
		return "", fmt.Errorf("db: backup failed: %w", err)
	}
	// Owner read/write only; failure is non-fatal (e.g. filesystems without
	// POSIX permissions).
	_ = os.Chmod(backupPath, 0600)

	b.prune(backupDir, stem)
	return backupPath, nil
}

// prune keeps the newest Keep backups matching this stem and removes older
// ones. Timestamped names sort chronologically, so a name sort suffices.
func (b *Backuper) prune(backupDir, stem string) {
	keep := b.Keep
	if keep <= 0 {
		keep = DefaultKeep
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}
	var backups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if matched, _ := filepath.Match(stem+"_*_*.db", e.Name()); matched {
			backups = append(backups, e.Name())
		}
	}
	if len(backups) <= keep {
		return
	}
	sort.Strings(backups)
	for _, name := range backups[:len(backups)-keep] {
		_ = os.Remove(filepath.Join(backupDir, name))
	}
}

// VerifyBackup opens the backup file read-only and runs PRAGMA
// integrity_check against it. openDriver is the SQL driver name to use
// (e.g. "sqlite3" for ncruces/go-sqlite3's database/sql registration).
func VerifyBackup(openDriver, backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("db: backup not readable: %w", err)
	}
	check, err := sql.Open(openDriver, "file:"+filepath.ToSlash(backupPath)+"?mode=ro")
	if err != nil {
		return fmt.Errorf("db: cannot open backup: %w", err)
	}
	defer check.Close()
	var result string
	if err := check.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return fmt.Errorf("db: integrity check failed to run: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(result), "ok") {
		return fmt.Errorf("db: backup failed integrity check: %s", result)
	}
	return nil
}

// Restore replaces the database file at dbPath with the verified backup at
// backupPath.
//
// PRECONDITION: every connection to dbPath must be closed first — restoring
// under a live pool corrupts the WAL. This function cannot verify that from
// a file path, so the caller owns connection shutdown; that is why Restore
// takes paths, not *sql.DB.
//
// Safety order: (1) integrity-check the backup; (2) snapshot the current
// database to "<db>.pre-restore-<timestamp>"; (3) copy the backup over the
// database and remove stale -wal/-shm siblings. If step 3 fails the
// pre-restore snapshot still holds the previous state.
func Restore(openDriver, dbPath, backupPath string, now time.Time) (preRestorePath string, err error) {
	if err := VerifyBackup(openDriver, backupPath); err != nil {
		return "", err
	}

	if _, err := os.Stat(dbPath); err == nil {
		preRestorePath = dbPath + ".pre-restore-" + now.Format("20060102_150405")
		if err := copyFile(dbPath, preRestorePath); err != nil {
			return "", fmt.Errorf("db: cannot snapshot current database before restore: %w", err)
		}
	}

	if err := copyFile(backupPath, dbPath); err != nil {
		return preRestorePath, fmt.Errorf("db: restore copy failed (previous state preserved at %s): %w", preRestorePath, err)
	}
	// A restored snapshot must not inherit the old journal.
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	return preRestorePath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
