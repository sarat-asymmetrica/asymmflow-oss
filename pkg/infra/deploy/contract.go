package deploy

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver" // register database/sql "sqlite3" driver (pure-Go, CGO banned)

	"ph_holdings_app/pkg/infra/release"
)

// BaselineSchema is the schema version assumed for a database that carries no
// schema stamp (a deployment that predates stamping). It equals the first
// shipped manifest schema_version, so an unstamped DB is treated as the
// baseline: current binaries open it untouched, and only a genuinely newer
// binary triggers a migration.
const BaselineSchema = 1

// schemaVersionKey is the settings-table key that stamps the data plane with
// the schema version its rows were written under.
const schemaVersionKey = "schema_version"

// forceReseedEnv is the sole, explicit escape hatch that re-seeds a PRESENT
// database from the packaged canon — for dev/demo only. It backs up first and
// is one-shot within a process (cleared via os.Unsetenv after it fires).
//
// One-shot holds for the intended usage — prefixing a single launch
// (PH_FORCE_RESEED=1 ./app). A value set PERSISTENTLY in the machine/user
// environment (setx) cannot be cleared from within the process, so every launch
// will reseed again — each time backing up first, so no data is silently lost,
// but it is a surprising repeated full reseed. Set it per-invocation, not
// persistently.
const forceReseedEnv = "PH_FORCE_RESEED"

// preMigratePrefix names the backup taken immediately before a migration or a
// forced reseed, so rollback is "restore this file" not an improvisation.
const preMigratePrefix = "pre-migrate-"

// DefaultBackupRetention is how many pre-migrate backups to keep (§4: N=5).
const DefaultBackupRetention = 5

// Action is what EnsureDatabase did, for logging and tests.
type Action string

const (
	// ActionSeededFresh: DB was absent and a packaged seed was copied in + stamped.
	ActionSeededFresh Action = "seeded_fresh"
	// ActionCreatedEmpty: DB was absent and no packaged seed was available — the
	// path is left for the app to create + migrate from scratch (dev / wails dev).
	ActionCreatedEmpty Action = "created_empty"
	// ActionOpenedCurrent: DB present and schema equal — opened untouched. This
	// is the anti-reseed invariant: a present DB is never replaced automatically.
	ActionOpenedCurrent Action = "opened_current"
	// ActionMigrated: DB present and binary schema newer — backed up, migrated on
	// a copy, atomically swapped in, and stamped.
	ActionMigrated Action = "migrated"
	// ActionForceReseeded: PH_FORCE_RESEED re-seeded a present DB (backed up first).
	ActionForceReseeded Action = "force_reseeded"
)

// Migrator runs schema migrations against the database file at path. The
// contract invokes it on a COPY of the live database; only on success is the
// copy atomically swapped in. A returned error leaves the original untouched.
type Migrator func(path string) error

// ContractConfig parameterizes one application of the boot-time update contract.
type ContractConfig struct {
	// DBPath is the resolved live-database location (see ResolveDatabasePath).
	DBPath string
	// SeedPath is the packaged synthetic-canon DB (see PackagedSeedPath); "" if
	// none is available (the app will create an empty DB instead).
	SeedPath string
	// BinarySchema is this binary's schema version (see BinarySchemaVersion).
	BinarySchema int
	// Migrate runs the real migrations on a copy during an upgrade. Required for
	// the upgrade path; a nil Migrator makes an upgrade an error rather than a
	// silent no-op (never migrate in place, never skip a needed migration).
	Migrate Migrator
	// BackupDir overrides where pre-migrate backups are written. Blank →
	// <DBdir>\backups.
	BackupDir string
	// Retention is how many pre-migrate backups to keep. <=0 → DefaultBackupRetention.
	Retention int
	// ForceReseed requests the PH_FORCE_RESEED escape hatch for this call.
	ForceReseed bool
	// Now stamps backup filenames deterministically (tests inject a fixed time).
	Now time.Time
	// Logf receives human-readable progress; nil silences it.
	Logf func(format string, args ...any)
}

// ContractResult reports what happened.
type ContractResult struct {
	Action     Action
	DBPath     string
	BackupPath string // set when a backup was taken
	FromSchema int    // DB schema before (0 when absent)
	ToSchema   int    // DB schema after
}

// BinarySchemaVersion parses this binary's declared schema version from the
// embedded release manifest. A blank or unparseable value falls back to
// BaselineSchema so a malformed manifest can never present as a downgrade.
func BinarySchemaVersion() int {
	raw := strings.TrimSpace(release.Current().SchemaVersion)
	if raw == "" {
		return BaselineSchema
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return BaselineSchema
	}
	return v
}

// ForceReseedRequested reports whether the PH_FORCE_RESEED escape hatch is set.
func ForceReseedRequested() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(forceReseedEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (c *ContractConfig) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

func (c *ContractConfig) now() time.Time {
	if c.Now.IsZero() {
		return time.Now()
	}
	return c.Now
}

func (c *ContractConfig) retention() int {
	if c.Retention <= 0 {
		return DefaultBackupRetention
	}
	return c.Retention
}

func (c *ContractConfig) backupDir() string {
	if strings.TrimSpace(c.BackupDir) != "" {
		return c.BackupDir
	}
	return filepath.Join(filepath.Dir(c.DBPath), "backups")
}

// EnsureDatabase applies the update contract at cfg.DBPath and returns what it
// did. It performs its own short-lived connections for reading/stamping the
// schema version and closes them before returning, so the caller may open its
// pool afterwards without contention.
//
// Contract (§6.2), in order:
//   - ForceReseed set → back up any present DB, reseed from the packaged canon,
//     stamp, clear the one-shot flag.
//   - DB absent → seed from the packaged canon (+stamp) if one exists; else
//     leave the path for the app to create empty.
//   - DB present, binary schema < DB schema → REFUSE (downgrade), naming both
//     versions and the rollback procedure. The original is never touched.
//   - DB present, binary schema == DB schema → open untouched (anti-reseed).
//   - DB present, binary schema > DB schema → back up, migrate a COPY, stamp the
//     copy, atomically swap it in. A failed migration leaves the original intact.
func EnsureDatabase(cfg ContractConfig) (ContractResult, error) {
	if strings.TrimSpace(cfg.DBPath) == "" {
		return ContractResult{}, fmt.Errorf("deploy: EnsureDatabase requires a database path")
	}
	res := ContractResult{DBPath: cfg.DBPath, ToSchema: cfg.BinarySchema}
	present := fileExists(cfg.DBPath)

	// --- Escape hatch: PH_FORCE_RESEED (dev/demo only, backs up first) --------
	if cfg.ForceReseed {
		if strings.TrimSpace(cfg.SeedPath) == "" {
			return res, fmt.Errorf("deploy: %s requested but no packaged seed is available", forceReseedEnv)
		}
		if present {
			backup, err := cfg.backupBeforeMigrate()
			if err != nil {
				return res, err
			}
			res.BackupPath = backup
			res.FromSchema = readSchemaVersion(cfg.DBPath)
			cfg.logf("🧷 %s: backed up present database before reseed → %s", forceReseedEnv, backup)
		}
		if err := copySeedAtomically(cfg.SeedPath, cfg.DBPath); err != nil {
			return res, fmt.Errorf("deploy: force reseed copy failed: %w", err)
		}
		stampSchemaVersion(cfg.DBPath, cfg.BinarySchema, cfg.logf)
		clearForceReseedFlag()
		res.Action = ActionForceReseeded
		cfg.logf("♻️ %s: reseeded data plane from packaged canon (schema %d) → %s", forceReseedEnv, cfg.BinarySchema, cfg.DBPath)
		return res, nil
	}

	// --- Absent DB: seed into absence (never a replacement) -------------------
	if !present {
		if strings.TrimSpace(cfg.SeedPath) == "" {
			res.Action = ActionCreatedEmpty
			res.FromSchema = 0
			cfg.logf("📂 No database and no packaged seed; app will create a fresh database at %s", cfg.DBPath)
			return res, nil
		}
		if err := copySeedAtomically(cfg.SeedPath, cfg.DBPath); err != nil {
			return res, fmt.Errorf("deploy: seed copy failed: %w", err)
		}
		stampSchemaVersion(cfg.DBPath, cfg.BinarySchema, cfg.logf)
		res.Action = ActionSeededFresh
		res.FromSchema = 0
		cfg.logf("🌱 Seeded data plane from packaged canon (schema %d) → %s", cfg.BinarySchema, cfg.DBPath)
		return res, nil
	}

	// --- Present DB: compare schema versions ----------------------------------
	dbSchema := readSchemaVersion(cfg.DBPath)
	res.FromSchema = dbSchema

	switch {
	case cfg.BinarySchema < dbSchema:
		// Downgrade refusal. Quiet "compatibility" is how databases corrupt.
		return res, &DowngradeError{
			DBPath:        cfg.DBPath,
			BinarySchema:  cfg.BinarySchema,
			DBSchema:      dbSchema,
			BinaryVersion: strings.TrimSpace(release.Current().Version),
		}

	case cfg.BinarySchema == dbSchema:
		// Anti-reseed invariant: a present, current DB is opened untouched even
		// when a richer packaged seed is available.
		res.Action = ActionOpenedCurrent
		cfg.logf("📂 Database present at schema %d (current); opening untouched: %s", dbSchema, cfg.DBPath)
		return res, nil

	default: // cfg.BinarySchema > dbSchema → migrate
		if cfg.Migrate == nil {
			return res, fmt.Errorf("deploy: schema upgrade %d→%d required but no migrator provided", dbSchema, cfg.BinarySchema)
		}

		backup, err := cfg.backupBeforeMigrate()
		if err != nil {
			return res, err
		}
		res.BackupPath = backup
		cfg.logf("🧷 Backed up before migrate (schema %d→%d) → %s", dbSchema, cfg.BinarySchema, backup)

		// Migrate a COPY, then atomically swap on success — never migrate in
		// place. A failure here leaves the live DB and the backup intact.
		tmp := cfg.DBPath + ".migrate-tmp"
		_ = os.Remove(tmp)
		if err := copyFile(cfg.DBPath, tmp); err != nil {
			return res, fmt.Errorf("deploy: could not stage migration copy: %w", err)
		}
		if err := cfg.Migrate(tmp); err != nil {
			_ = os.Remove(tmp)
			_ = os.Remove(tmp + "-wal")
			_ = os.Remove(tmp + "-shm")
			return res, &MigrationError{
				DBPath:     cfg.DBPath,
				BackupPath: backup,
				FromSchema: dbSchema,
				ToSchema:   cfg.BinarySchema,
				Cause:      err,
			}
		}
		stampSchemaVersion(tmp, cfg.BinarySchema, cfg.logf)
		if err := swapInPlace(tmp, cfg.DBPath); err != nil {
			_ = os.Remove(tmp)
			return res, &MigrationError{
				DBPath:     cfg.DBPath,
				BackupPath: backup,
				FromSchema: dbSchema,
				ToSchema:   cfg.BinarySchema,
				Cause:      fmt.Errorf("atomic swap failed: %w", err),
			}
		}
		res.Action = ActionMigrated
		cfg.logf("⬆️ Migrated data plane %d→%d and swapped in: %s", dbSchema, cfg.BinarySchema, cfg.DBPath)
		return res, nil
	}
}

// DowngradeError is returned when the binary's schema is older than the DB's.
type DowngradeError struct {
	DBPath        string
	BinarySchema  int
	DBSchema      int
	BinaryVersion string
}

func (e *DowngradeError) Error() string {
	version := e.BinaryVersion
	if version == "" {
		version = "this build"
	}
	return fmt.Sprintf(
		"database schema v%d is newer than this application (%s, schema v%d). "+
			"Refusing to open to avoid corruption. To roll forward, install the newer application; "+
			"to roll back the data, restore a pre-migration backup from the backups\\ folder next to %s.",
		e.DBSchema, version, e.BinarySchema, e.DBPath,
	)
}

// MigrationError is returned when a migration fails; the original DB is intact
// and BackupPath names the pre-migration backup to restore from.
type MigrationError struct {
	DBPath     string
	BackupPath string
	FromSchema int
	ToSchema   int
	Cause      error
}

func (e *MigrationError) Error() string {
	return fmt.Sprintf(
		"schema migration %d→%d failed: %v. The original database was left untouched at %s; "+
			"a pre-migration backup is at %s.",
		e.FromSchema, e.ToSchema, e.Cause, e.DBPath, e.BackupPath,
	)
}

func (e *MigrationError) Unwrap() error { return e.Cause }

// backupBeforeMigrate copies the live DB to <backupDir>\pre-migrate-<ts>.db and
// prunes to the retention limit. It copies the file (rather than VACUUM INTO)
// because the contract runs before any connection pool is opened.
func (c *ContractConfig) backupBeforeMigrate() (string, error) {
	dir := c.backupDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("deploy: cannot create backup directory: %w", err)
	}
	backupPath := filepath.Join(dir, preMigratePrefix+c.now().Format("20060102_150405")+".db")
	// Avoid clobbering a same-second backup (deterministic test clocks).
	for i := 1; fileExists(backupPath); i++ {
		backupPath = filepath.Join(dir, fmt.Sprintf("%s%s_%d.db", preMigratePrefix, c.now().Format("20060102_150405"), i))
	}
	if err := copyFile(c.DBPath, backupPath); err != nil {
		return "", fmt.Errorf("deploy: backup copy failed: %w", err)
	}
	_ = os.Chmod(backupPath, 0o600)
	prunePreMigrateBackups(dir, c.retention())
	return backupPath, nil
}

// prunePreMigrateBackups keeps the newest keep pre-migrate backups. Timestamped
// names sort chronologically, so a lexical sort suffices.
func prunePreMigrateBackups(dir string, keep int) {
	if keep <= 0 {
		keep = DefaultBackupRetention
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var backups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), preMigratePrefix) && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, e.Name())
		}
	}
	if len(backups) <= keep {
		return
	}
	sort.Strings(backups)
	for _, name := range backups[:len(backups)-keep] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// ---------------------------------------------------------------------------
// Schema stamping (settings table)
// ---------------------------------------------------------------------------

func readonlyDSN(path string) string {
	// mode=ro guarantees no writes to the main database file, so reading the
	// schema stamp never mutates a byte (the anti-reseed byte-compare depends on
	// this). immutable is deliberately omitted to avoid driver-specific URI
	// param handling — read-only is sufficient and portable.
	return "file:" + filepath.ToSlash(filepath.Clean(path)) + "?mode=ro&_pragma=busy_timeout(5000)"
}

func readwriteDSN(path string) string {
	return "file:" + filepath.ToSlash(filepath.Clean(path)) + "?_pragma=busy_timeout(5000)"
}

// readSchemaVersion returns the stamped schema version, or BaselineSchema when
// the DB carries no stamp (predates stamping) or cannot be read. It opens the
// file read-only and immutable, so it never mutates a byte of the database.
func readSchemaVersion(path string) int {
	if !isSQLiteFile(path) {
		return BaselineSchema
	}
	db, err := sql.Open("sqlite3", readonlyDSN(path))
	if err != nil {
		return BaselineSchema
	}
	defer db.Close()

	var hasSettings int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='settings'`).Scan(&hasSettings); err != nil || hasSettings == 0 {
		return BaselineSchema
	}
	var value string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = ? AND deleted_at IS NULL`, schemaVersionKey).Scan(&value); err != nil {
		return BaselineSchema
	}
	v, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || v <= 0 {
		return BaselineSchema
	}
	return v
}

// stampSchemaVersion upserts the schema-version row into the settings table.
// It is best-effort: a database without a settings table (a bare, not-yet-
// migrated file) is left unstamped — the app's migration creates the table and
// the next boot treats the file as BaselineSchema, which is correct.
func stampSchemaVersion(path string, version int, logf func(string, ...any)) {
	if !isSQLiteFile(path) {
		return
	}
	db, err := sql.Open("sqlite3", readwriteDSN(path))
	if err != nil {
		if logf != nil {
			logf("⚠️ Could not open database to stamp schema version: %v", err)
		}
		return
	}
	defer db.Close()

	var hasSettings int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='settings'`).Scan(&hasSettings); err != nil || hasSettings == 0 {
		return
	}

	// Mirror the settings row shape used elsewhere (Base columns + payload).
	_, err = db.Exec(`
		INSERT INTO settings (
			id, key, value, category, description, is_encrypted,
			created_at, updated_at, version, created_by, deleted_at
		)
		VALUES (
			?, ?, ?, 'deployment',
			'Schema version the data plane was last written/migrated under',
			0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1, 'update-contract', NULL
		)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			category = excluded.category,
			description = excluded.description,
			is_encrypted = 0,
			updated_at = CURRENT_TIMESTAMP,
			deleted_at = NULL
	`, "deployment-schema-version", schemaVersionKey, strconv.Itoa(version))
	if err != nil {
		if logf != nil {
			logf("⚠️ Could not stamp schema version %d: %v", version, err)
		}
		return
	}

	// Flush the stamp into the main file BEFORE this connection closes. The file
	// may already be in WAL mode (journal_mode is persisted in the SQLite header
	// — e.g. after migrateDatabaseFileForContract migrated the copy in WAL), so
	// the upsert could otherwise land only in a -wal sibling that the caller's
	// atomic swap then deletes, silently dropping the stamp and re-triggering the
	// migration on every boot. TRUNCATE is a no-op on non-WAL databases.
	if _, cpErr := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); cpErr != nil && logf != nil {
		logf("⚠️ Could not checkpoint after stamping schema version %d: %v", version, cpErr)
	}
}

func clearForceReseedFlag() { _ = os.Unsetenv(forceReseedEnv) }

// ---------------------------------------------------------------------------
// File primitives
// ---------------------------------------------------------------------------

// isSQLiteFile reports whether path begins with the SQLite file header.
func isSQLiteFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 16)
	if _, err := io.ReadFull(f, header); err != nil {
		return false
	}
	return string(header) == "SQLite format 3\x00"
}

// copyFile copies src to dst (creating parent dirs), syncing to disk.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		// Never leave a truncated file behind: a half-written pre-migrate backup
		// is indistinguishable by name from a good one and a human following the
		// restore instructions could pick it (it would be the newest).
		_ = os.Remove(dst)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

// copySeedAtomically copies a packaged seed to dst via a temp file + rename,
// validating the SQLite header and clearing stale WAL/SHM siblings. It refuses
// to install a file that is not a valid SQLite database.
func copySeedAtomically(src, dst string) error {
	if !isSQLiteFile(src) {
		return fmt.Errorf("packaged seed failed SQLite header validation: %s", src)
	}
	tmp := dst + ".seed-tmp"
	_ = os.Remove(tmp)
	if err := copyFile(src, tmp); err != nil {
		return err
	}
	if !isSQLiteFile(tmp) {
		_ = os.Remove(tmp)
		return fmt.Errorf("copied seed failed SQLite header validation")
	}
	return swapInPlace(tmp, dst)
}

// swapInPlace atomically replaces dst with tmp, clearing stale WAL/SHM siblings
// on both sides so the swapped-in file is not shadowed by an old journal.
func swapInPlace(tmp, dst string) error {
	_ = os.Remove(dst + "-wal")
	_ = os.Remove(dst + "-shm")
	_ = os.Remove(tmp + "-wal")
	_ = os.Remove(tmp + "-shm")
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	_ = os.Remove(dst + "-wal")
	_ = os.Remove(dst + "-shm")
	_ = os.Chmod(dst, 0o600)
	return nil
}
