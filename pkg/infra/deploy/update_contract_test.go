package deploy

import (
	"bytes"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// makeDB creates a minimal but real SQLite database at path with a settings
// table (so schema stamping works), an optional schema_version stamp
// (schema<=0 leaves it unstamped), and a widgets table carrying `rows` rows so
// two databases can differ in "richness". Journal mode is DELETE so the file is
// a single self-contained artifact (no -wal sibling) — byte comparisons stay
// meaningful.
func makeDB(t *testing.T, path string, schema, rows int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=journal_mode(DELETE)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE settings (
			id TEXT PRIMARY KEY,
			key TEXT UNIQUE,
			value TEXT,
			category TEXT,
			description TEXT,
			is_encrypted INTEGER DEFAULT 0,
			created_at TEXT,
			updated_at TEXT,
			version INTEGER DEFAULT 1,
			created_by TEXT,
			deleted_at TEXT
		);`,
		`CREATE TABLE widgets (id INTEGER PRIMARY KEY);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
	for i := 0; i < rows; i++ {
		if _, err := db.Exec(`INSERT INTO widgets DEFAULT VALUES`); err != nil {
			t.Fatalf("insert widget: %v", err)
		}
	}
	if schema > 0 {
		if _, err := db.Exec(
			`INSERT INTO settings (id, key, value, category, created_by) VALUES (?, ?, ?, 'deployment', 'test')`,
			"deployment-schema-version", schemaVersionKey, strconv.Itoa(schema),
		); err != nil {
			t.Fatalf("stamp schema: %v", err)
		}
	}
}

func widgetCount(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		t.Fatalf("open ro %s: %v", path, err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM widgets`).Scan(&n); err != nil {
		t.Fatalf("count widgets: %v", err)
	}
	return n
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func fixedClock() time.Time { return time.Date(2026, 7, 18, 9, 0, 0, 0, time.UTC) }

// --- Gate 2: the update contract -------------------------------------------

func TestContract_FreshBootSeed(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	seed := filepath.Join(dir, "seed.db")
	makeDB(t, seed, 0, 42) // packaged canon: 42 rows, unstamped

	res, err := EnsureDatabase(ContractConfig{DBPath: dbPath, SeedPath: seed, BinarySchema: 1, Now: fixedClock()})
	if err != nil {
		t.Fatalf("EnsureDatabase: %v", err)
	}
	if res.Action != ActionSeededFresh {
		t.Fatalf("action = %s, want %s", res.Action, ActionSeededFresh)
	}
	if !fileExists(dbPath) {
		t.Fatalf("database not seeded at %s", dbPath)
	}
	if got := widgetCount(t, dbPath); got != 42 {
		t.Fatalf("seeded row count = %d, want 42", got)
	}
	if got := readSchemaVersion(dbPath); got != 1 {
		t.Fatalf("seeded DB schema = %d, want 1 (stamped)", got)
	}
}

func TestContract_CreatedEmptyWhenNoSeed(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)

	res, err := EnsureDatabase(ContractConfig{DBPath: dbPath, SeedPath: "", BinarySchema: 1})
	if err != nil {
		t.Fatalf("EnsureDatabase: %v", err)
	}
	if res.Action != ActionCreatedEmpty {
		t.Fatalf("action = %s, want %s", res.Action, ActionCreatedEmpty)
	}
	if fileExists(dbPath) {
		t.Fatalf("no seed was available; contract must not create a file, app owns that")
	}
}

// The anti-reseed byte-compare proof: a present DB at the current schema is
// opened untouched even when a RICHER packaged seed is available.
func TestContract_PresentDBUntouched_AntiReseed(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	makeDB(t, dbPath, 1, 3) // present, stamped v1, only 3 rows
	seed := filepath.Join(dir, "seed.db")
	makeDB(t, seed, 1, 999) // much richer seed, same schema

	before := readBytes(t, dbPath)

	res, err := EnsureDatabase(ContractConfig{DBPath: dbPath, SeedPath: seed, BinarySchema: 1, Now: fixedClock()})
	if err != nil {
		t.Fatalf("EnsureDatabase: %v", err)
	}
	if res.Action != ActionOpenedCurrent {
		t.Fatalf("action = %s, want %s", res.Action, ActionOpenedCurrent)
	}
	after := readBytes(t, dbPath)
	if !bytes.Equal(before, after) {
		t.Fatalf("present DB was modified (%d → %d bytes) — anti-reseed invariant violated", len(before), len(after))
	}
	if got := widgetCount(t, dbPath); got != 3 {
		t.Fatalf("present DB rows = %d, want 3 (seed must NOT have replaced it)", got)
	}
	if fileExists(filepath.Join(filepath.Dir(dbPath), "backups")) {
		t.Fatalf("no backup should be taken when opening an unchanged DB")
	}
}

func TestContract_UpgradeBackupMigrateStamp(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	makeDB(t, dbPath, 1, 5) // present at schema v1
	before := readBytes(t, dbPath)

	migrated := false
	migrator := func(copyPath string) error {
		migrated = true
		db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(copyPath)+"?_pragma=journal_mode(DELETE)")
		if err != nil {
			return err
		}
		defer db.Close()
		_, err = db.Exec(`CREATE TABLE gadgets (id INTEGER PRIMARY KEY)`)
		return err
	}

	res, err := EnsureDatabase(ContractConfig{
		DBPath: dbPath, BinarySchema: 2, Migrate: migrator, Now: fixedClock(),
	})
	if err != nil {
		t.Fatalf("EnsureDatabase: %v", err)
	}
	if res.Action != ActionMigrated {
		t.Fatalf("action = %s, want %s", res.Action, ActionMigrated)
	}
	if !migrated {
		t.Fatalf("migrator was never invoked")
	}
	// Backup exists and holds the pre-migration bytes.
	if res.BackupPath == "" || !fileExists(res.BackupPath) {
		t.Fatalf("backup was not created (path=%q)", res.BackupPath)
	}
	if !bytes.Equal(before, readBytes(t, res.BackupPath)) {
		t.Fatalf("backup does not match the pre-migration database")
	}
	// Live DB now carries the migration and the new stamp.
	if got := readSchemaVersion(dbPath); got != 2 {
		t.Fatalf("post-migrate schema = %d, want 2", got)
	}
	if !tableExists(t, dbPath, "gadgets") {
		t.Fatalf("migration change (gadgets table) not present in live DB")
	}
	// No temp artifact left behind.
	if fileExists(dbPath + ".migrate-tmp") {
		t.Fatalf("migrate temp file was not cleaned up")
	}
}

func TestContract_DowngradeRefused(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	makeDB(t, dbPath, 3, 5) // DB is schema v3
	before := readBytes(t, dbPath)

	res, err := EnsureDatabase(ContractConfig{DBPath: dbPath, BinarySchema: 2, Now: fixedClock()})
	if err == nil {
		t.Fatalf("expected downgrade refusal, got action %s", res.Action)
	}
	var de *DowngradeError
	if !errors.As(err, &de) {
		t.Fatalf("expected *DowngradeError, got %T: %v", err, err)
	}
	if de.DBSchema != 3 || de.BinarySchema != 2 {
		t.Fatalf("downgrade error versions wrong: %+v", de)
	}
	if !bytes.Equal(before, readBytes(t, dbPath)) {
		t.Fatalf("downgrade must leave the DB untouched")
	}
	if fileExists(filepath.Join(filepath.Dir(dbPath), "backups")) {
		t.Fatalf("downgrade refusal must not take a backup")
	}
}

func TestContract_FailedMigrationLeavesOriginalIntact(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	makeDB(t, dbPath, 1, 5)
	before := readBytes(t, dbPath)

	boom := errors.New("migration exploded")
	res, err := EnsureDatabase(ContractConfig{
		DBPath: dbPath, BinarySchema: 2, Now: fixedClock(),
		Migrate: func(string) error { return boom },
	})
	if err == nil {
		t.Fatalf("expected migration failure, got action %s", res.Action)
	}
	var me *MigrationError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MigrationError, got %T: %v", err, err)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("migration error should wrap the cause")
	}
	// Original DB byte-identical; backup was taken; no temp left.
	if !bytes.Equal(before, readBytes(t, dbPath)) {
		t.Fatalf("failed migration must leave the original DB untouched")
	}
	if me.BackupPath == "" || !fileExists(me.BackupPath) {
		t.Fatalf("a pre-migration backup should exist for rollback")
	}
	if fileExists(dbPath+".migrate-tmp") || fileExists(dbPath+".migrate-tmp-wal") {
		t.Fatalf("migration temp artifacts not cleaned up")
	}
	if got := readSchemaVersion(dbPath); got != 1 {
		t.Fatalf("schema stamp must be unchanged after failed migration, got %d", got)
	}
}

func TestContract_BackupRetentionPrunesToFive(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	makeDB(t, dbPath, 1, 1)
	seed := filepath.Join(dir, "seed.db")
	makeDB(t, seed, 1, 1)

	base := fixedClock()
	for i := 0; i < 7; i++ {
		_, err := EnsureDatabase(ContractConfig{
			DBPath: dbPath, SeedPath: seed, BinarySchema: 1,
			ForceReseed: true, Now: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("force reseed %d: %v", i, err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(dir, "data", "backups"))
	if err != nil {
		t.Fatalf("read backups dir: %v", err)
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	if count != DefaultBackupRetention {
		t.Fatalf("backup retention = %d, want %d", count, DefaultBackupRetention)
	}
}

// --- Gate 3: PH_FORCE_RESEED one-shot semantics -----------------------------

func TestForceReseed_OneShot(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", DBFileName)
	makeDB(t, dbPath, 1, 2) // present, 2 rows
	seed := filepath.Join(dir, "seed.db")
	makeDB(t, seed, 1, 88) // richer canon

	t.Setenv(forceReseedEnv, "1")
	if !ForceReseedRequested() {
		t.Fatalf("ForceReseedRequested should be true when %s=1", forceReseedEnv)
	}

	// First boot: backs up, reseeds, and clears the one-shot flag.
	res, err := EnsureDatabase(ContractConfig{
		DBPath: dbPath, SeedPath: seed, BinarySchema: 1,
		ForceReseed: ForceReseedRequested(), Now: fixedClock(),
	})
	if err != nil {
		t.Fatalf("force reseed: %v", err)
	}
	if res.Action != ActionForceReseeded {
		t.Fatalf("action = %s, want %s", res.Action, ActionForceReseeded)
	}
	if res.BackupPath == "" || !fileExists(res.BackupPath) {
		t.Fatalf("force reseed must back up the present DB first")
	}
	if got := widgetCount(t, dbPath); got != 88 {
		t.Fatalf("after reseed rows = %d, want 88 (from canon)", got)
	}
	// One-shot: the flag cleared itself.
	if os.Getenv(forceReseedEnv) != "" {
		t.Fatalf("%s should be cleared after firing (one-shot)", forceReseedEnv)
	}
	if ForceReseedRequested() {
		t.Fatalf("ForceReseedRequested should be false after the one-shot fired")
	}

	// Second boot rebuilt from the (now-cleared) env: no reseed, opened as-is.
	res2, err := EnsureDatabase(ContractConfig{
		DBPath: dbPath, SeedPath: seed, BinarySchema: 1,
		ForceReseed: ForceReseedRequested(), Now: fixedClock(),
	})
	if err != nil {
		t.Fatalf("second boot: %v", err)
	}
	if res2.Action != ActionOpenedCurrent {
		t.Fatalf("second boot action = %s, want %s (one-shot must not reseed again)", res2.Action, ActionOpenedCurrent)
	}
}

func TestBinarySchemaVersion_ParsesManifest(t *testing.T) {
	// The embedded manifest ships schema_version "1"; a valid parse is required.
	if got := BinarySchemaVersion(); got < 1 {
		t.Fatalf("BinarySchemaVersion = %d, want >= 1", got)
	}
}

func tableExists(t *testing.T, path, table string) bool {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		t.Fatalf("open ro: %v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil {
		t.Fatalf("query table: %v", err)
	}
	return n > 0
}

