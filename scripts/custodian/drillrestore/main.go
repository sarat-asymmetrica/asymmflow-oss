// drill_restore.go (package drillrestore) — Custodian Wave 1, Mission CW1-B.
//
// Standalone, go-run-able restore drill. It exercises the REAL backup/restore
// engine the product ships (pkg/infra/db: Backuper.Backup via VACUUM INTO,
// VerifyBackup, Restore) against a synthetic SQLite database, entirely inside
// a scratch directory under the OS temp root. It never opens, reads, or
// writes anything under a real deployment's data plane.
//
// Scenario-build choice (documented per the mission brief): the app's real
// schema is produced by GORM AutoMigrate from the compiled binary (~90
// tables, database.go:27-443), not from SQL files. Importing that whole
// migration surface into a throwaway drill binary would be heavy and would
// couple this drill to unrelated schema churn. Instead this drill opens a
// real SQLite database through the SAME driver the product uses
// (github.com/ncruces/go-sqlite3/driver, "sqlite3"), and creates a
// REPRESENTATIVE SUBSET of the tables CW10_INVENTORY.md names as the restore
// sentinels (customers, invoices, payments, settings incl. schema_version),
// seeded with synthetic rows carrying known checksums. This proves the exact
// backup/restore code path PH runs (VACUUM INTO -> integrity_check ->
// snapshot-then-swap) without depending on the full production schema.
//
// Run:  go run ./scripts/custodian/drillrestore
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	dbengine "ph_holdings_app/pkg/infra/db"

	_ "github.com/ncruces/go-sqlite3/driver"
)

const driverName = "sqlite3"

var (
	failures    int
	scratchRoot string
)

func check(name string, cond bool, detail string) {
	if cond {
		fmt.Printf("  [PASS] %s\n", name)
		return
	}
	failures++
	if detail != "" {
		fmt.Printf("  [FAIL] %s — %s\n", name, detail)
	} else {
		fmt.Printf("  [FAIL] %s\n", name)
	}
}

// checkExpectError is for negative controls: cond is true when the expected
// error actually occurred (i.e. the drill went red the way it should).
func checkExpectError(name string, err error, wantErr bool) {
	got := err != nil
	if got == wantErr {
		if wantErr {
			fmt.Printf("  [PASS-RED] %s (correctly refused: %v)\n", name, err)
		} else {
			fmt.Printf("  [PASS] %s\n", name)
		}
		return
	}
	failures++
	if wantErr {
		fmt.Printf("  [FAIL] %s — expected refusal, got success\n", name)
	} else {
		fmt.Printf("  [FAIL] %s — unexpected error: %v\n", name, err)
	}
}

func section(title string) {
	fmt.Printf("\n== %s ==\n", title)
}

// guardTarget is the liveness guard: refuses to operate on any path that is
// not inside this run's own scratch root, with two named checks the mission
// brief calls out explicitly (live Asymmetrica data plane, live-looking
// ph_holdings.db) plus the general scratch-root containment check.
func guardTarget(p string) error {
	abs, err := filepath.Abs(p)
	if err != nil {
		return fmt.Errorf("liveness guard: cannot resolve %q: %w", p, err)
	}
	lower := strings.ToLower(abs)

	if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
		liveRoot, err := filepath.Abs(filepath.Join(appData, "Asymmetrica"))
		if err == nil {
			liveRootLower := strings.ToLower(liveRoot)
			if lower == liveRootLower || strings.HasPrefix(lower, liveRootLower+string(filepath.Separator)) {
				return fmt.Errorf("refused — %s is under the live Asymmetrica data plane (%s)", abs, liveRoot)
			}
		}
	}

	absScratchRoot, err := filepath.Abs(scratchRoot)
	if err != nil {
		return fmt.Errorf("liveness guard: cannot resolve scratch root: %w", err)
	}
	rel, err := filepath.Rel(absScratchRoot, abs)
	outsideScratch := err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))

	if strings.EqualFold(filepath.Base(abs), "ph_holdings.db") && outsideScratch {
		return fmt.Errorf("refused — %s is named ph_holdings.db but sits outside this drill's scratch root (%s)", abs, absScratchRoot)
	}
	if outsideScratch {
		return fmt.Errorf("refused — %s is outside this drill's scratch root (%s)", abs, absScratchRoot)
	}
	return nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func mustExec(conn *sql.DB, query string, args ...any) {
	if _, err := conn.Exec(query, args...); err != nil {
		panic(fmt.Sprintf("exec failed: %s: %v", query, err))
	}
}

func openDB(path string) (*sql.DB, error) {
	return sql.Open(driverName, "file:"+filepath.ToSlash(path))
}

func openReadOnly(path string) (*sql.DB, error) {
	return sql.Open(driverName, "file:"+filepath.ToSlash(path)+"?mode=ro")
}

// scenario holds the synthetic sentinel values seeded into the source DB, so
// every later assertion (restore, corrupt-negative, etc.) checks against a
// single source of truth rather than re-deriving expected values.
type scenario struct {
	runTS             string
	customerSentinel  string
	customerChecksum  string
	invoiceCount      int
	paymentCount      int
	customerCount     int
	schemaVersion     string
}

func buildScenario(conn *sql.DB, runTS string) scenario {
	mustExec(conn, `CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL)`)
	mustExec(conn, `CREATE TABLE invoices (id INTEGER PRIMARY KEY, customer_id INTEGER NOT NULL, total_cents INTEGER NOT NULL)`)
	mustExec(conn, `CREATE TABLE payments (id INTEGER PRIMARY KEY, invoice_id INTEGER NOT NULL, amount_cents INTEGER NOT NULL)`)
	mustExec(conn, `CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)

	sentinel := fmt.Sprintf("CUSTODIAN-DRILL-SENTINEL-%s", runTS)
	checksum := sha256Hex(sentinel)

	mustExec(conn, `INSERT INTO customers (id, name, checksum) VALUES (1, ?, ?)`, sentinel, checksum)
	mustExec(conn, `INSERT INTO customers (id, name, checksum) VALUES (2, 'Synthetic Customer B', ?)`, sha256Hex("Synthetic Customer B"))
	mustExec(conn, `INSERT INTO customers (id, name, checksum) VALUES (3, 'Synthetic Customer C', ?)`, sha256Hex("Synthetic Customer C"))

	mustExec(conn, `INSERT INTO invoices (id, customer_id, total_cents) VALUES (1, 1, 150000)`)
	mustExec(conn, `INSERT INTO invoices (id, customer_id, total_cents) VALUES (2, 2, 275000)`)

	mustExec(conn, `INSERT INTO payments (id, invoice_id, amount_cents) VALUES (1, 1, 150000)`)
	mustExec(conn, `INSERT INTO payments (id, invoice_id, amount_cents) VALUES (2, 2, 100000)`)

	schemaVersion := "custodian-drill-schema-v1"
	mustExec(conn, `INSERT INTO settings (key, value) VALUES ('schema_version', ?)`, schemaVersion)
	mustExec(conn, `INSERT INTO settings (key, value) VALUES ('backup_auto_enabled', 'true')`)

	return scenario{
		runTS:            runTS,
		customerSentinel: sentinel,
		customerChecksum: checksum,
		invoiceCount:     2,
		paymentCount:     2,
		customerCount:    3,
		schemaVersion:    schemaVersion,
	}
}

func rowCount(conn *sql.DB, table string) (int, error) {
	var n int
	err := conn.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)).Scan(&n)
	return n, err
}

func assertContent(label string, conn *sql.DB, sc scenario) {
	var integrity string
	if err := conn.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		check(label+": integrity_check runs", false, err.Error())
	} else {
		check(label+": PRAGMA integrity_check == ok", strings.EqualFold(strings.TrimSpace(integrity), "ok"), integrity)
	}

	if n, err := rowCount(conn, "customers"); err != nil {
		check(label+": customers row count query", false, err.Error())
	} else {
		check(label+": customers row count matches source", n == sc.customerCount, fmt.Sprintf("got %d want %d", n, sc.customerCount))
	}
	if n, err := rowCount(conn, "invoices"); err != nil {
		check(label+": invoices row count query", false, err.Error())
	} else {
		check(label+": invoices row count matches source", n == sc.invoiceCount, fmt.Sprintf("got %d want %d", n, sc.invoiceCount))
	}
	if n, err := rowCount(conn, "payments"); err != nil {
		check(label+": payments row count query", false, err.Error())
	} else {
		check(label+": payments row count matches source", n == sc.paymentCount, fmt.Sprintf("got %d want %d", n, sc.paymentCount))
	}

	var gotName, gotChecksum string
	err := conn.QueryRow(`SELECT name, checksum FROM customers WHERE id = 1`).Scan(&gotName, &gotChecksum)
	if err != nil {
		check(label+": sentinel record readback", false, err.Error())
	} else {
		check(label+": sentinel record readback byte-for-byte", gotName == sc.customerSentinel && gotChecksum == sc.customerChecksum,
			fmt.Sprintf("got name=%q checksum=%q", gotName, gotChecksum))
	}

	var gotSchemaVersion string
	err = conn.QueryRow(`SELECT value FROM settings WHERE key = 'schema_version'`).Scan(&gotSchemaVersion)
	if err != nil {
		check(label+": settings.schema_version survives", false, err.Error())
	} else {
		check(label+": settings.schema_version survives", gotSchemaVersion == sc.schemaVersion,
			fmt.Sprintf("got %q want %q", gotSchemaVersion, sc.schemaVersion))
	}
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
	return out.Close()
}

func main() {
	runTS := time.Now().Format("20060102_150405")
	scratchRoot = filepath.Join(os.TempDir(), "custodian-drill", runTS)
	if strings.Contains(scratchRoot, "#") {
		fmt.Println("FATAL: scratch root contains '#', refusing to run:", scratchRoot)
		os.Exit(2)
	}
	if err := os.MkdirAll(scratchRoot, 0700); err != nil {
		fmt.Println("FATAL: cannot create scratch root:", err)
		os.Exit(2)
	}
	fmt.Println("Custodian Wave 1 — CW1-B restore drill")
	fmt.Println("Scratch root:", scratchRoot)

	// ── Liveness guard negative tests (run FIRST, before any real work,
	// so a guard defect cannot be masked by later scratch-only success) ──
	section("Liveness guard — negative controls")
	{
		fakeLiveDir := filepath.Join(os.TempDir(), "custodian-drill-livecheck-fake-"+runTS)
		if err := os.MkdirAll(fakeLiveDir, 0700); err != nil {
			fmt.Println("FATAL: cannot create fake-live scratch dir:", err)
			os.Exit(2)
		}
		fakeLivePath := filepath.Join(fakeLiveDir, "ph_holdings.db")
		if err := os.WriteFile(fakeLivePath, []byte("not a real db, just a name collision"), 0600); err != nil {
			fmt.Println("FATAL: cannot write fake-live scratch file:", err)
			os.Exit(2)
		}
		err := guardTarget(fakeLivePath)
		checkExpectError("guard refuses a live-looking ph_holdings.db outside the scratch root", err, true)

		// Path-string-only check (no I/O against real APPDATA): construct a
		// hypothetical path under %APPDATA%\Asymmetrica and confirm the guard
		// refuses it purely from the string, without ever touching it.
		if appData := os.Getenv("APPDATA"); appData != "" {
			hypotheticalLive := filepath.Join(appData, "Asymmetrica", "AsymmFlow-Dev", "data", "ph_holdings.db")
			err := guardTarget(hypotheticalLive)
			checkExpectError("guard refuses a path under %APPDATA%\\Asymmetrica (string-only, never touched)", err, true)
		} else {
			fmt.Println("  [SKIP] APPDATA not set in this environment — cannot exercise the Asymmetrica-root check")
		}

		// Positive control: a path INSIDE the scratch root must be allowed.
		okPath := filepath.Join(scratchRoot, "source", "ph_holdings.db")
		err = guardTarget(okPath)
		checkExpectError("guard allows a path inside the drill's own scratch root", err, false)

		_ = os.RemoveAll(fakeLiveDir)
	}

	// ── Scenario build ──
	section("Scenario build (synthetic DB, real driver + schema-subset)")
	sourceDir := filepath.Join(scratchRoot, "source")
	if err := os.MkdirAll(sourceDir, 0700); err != nil {
		fmt.Println("FATAL:", err)
		os.Exit(2)
	}
	sourceDBPath := filepath.Join(sourceDir, "ph_holdings.db")
	if err := guardTarget(sourceDBPath); err != nil {
		fmt.Println("FATAL: liveness guard refused the drill's own source path (bug):", err)
		os.Exit(2)
	}
	sourceConn, err := openDB(sourceDBPath)
	if err != nil {
		fmt.Println("FATAL: cannot open source DB:", err)
		os.Exit(2)
	}
	sc := buildScenario(sourceConn, runTS)
	check("scenario: source DB seeded with customers/invoices/payments/settings", true, "")
	fmt.Printf("  sentinel customer id=1 name=%q checksum=%s\n", sc.customerSentinel, sc.customerChecksum)

	// ── Backup using the REAL engine (VACUUM INTO via pkg/infra/db) ──
	section("Backup (real engine: pkg/infra/db.Backuper.Backup)")
	backuper := &dbengine.Backuper{DB: sourceConn, Dir: filepath.Join(sourceDir, "backups")}
	backupStart := time.Now()
	backupPath, err := backuper.Backup(time.Now())
	backupDuration := time.Since(backupStart)
	if err != nil {
		fmt.Println("FATAL: Backup failed:", err)
		os.Exit(2)
	}
	if err := guardTarget(backupPath); err != nil {
		fmt.Println("FATAL: liveness guard refused the drill's own backup path (bug):", err)
		os.Exit(2)
	}
	check("backup: artifact created", fileExists(backupPath), backupPath)
	fmt.Printf("  backup artifact: %s\n", backupPath)
	fmt.Printf("  STOPWATCH backup leg: %s\n", backupDuration)

	// Source connection must close before Restore ever touches a dbPath
	// (Restore's own precondition — see pkg/infra/db/backup.go doc comment).
	if err := sourceConn.Close(); err != nil {
		fmt.Println("FATAL: cannot close source connection:", err)
		os.Exit(2)
	}

	// ── Restore to a SECOND scratch location using the real Restore()/VerifyBackup() ──
	section("Restore (real engine: pkg/infra/db.Restore + VerifyBackup) — GREEN path")
	restoreDir := filepath.Join(scratchRoot, "restored")
	if err := os.MkdirAll(restoreDir, 0700); err != nil {
		fmt.Println("FATAL:", err)
		os.Exit(2)
	}
	restoreDBPath := filepath.Join(restoreDir, "ph_holdings.db")
	if err := guardTarget(restoreDBPath); err != nil {
		fmt.Println("FATAL: liveness guard refused the drill's own restore target (bug):", err)
		os.Exit(2)
	}

	verifyStart := time.Now()
	verifyErr := dbengine.VerifyBackup(driverName, backupPath)
	check("VerifyBackup: real (uncorrupted) artifact passes integrity_check", verifyErr == nil, fmt.Sprintf("%v", verifyErr))

	restoreStart := time.Now()
	preRestorePath, restoreErr := dbengine.Restore(driverName, restoreDBPath, backupPath, time.Now())
	restoreDuration := time.Since(restoreStart)
	verifyPlusRestoreDuration := time.Since(verifyStart)
	if restoreErr != nil {
		check("Restore: succeeds against a fresh scratch target", false, restoreErr.Error())
	} else {
		check("Restore: succeeds against a fresh scratch target", true, "")
	}
	// A fresh target has nothing to snapshot, so no pre-restore file is expected.
	check("Restore: no pre-restore snapshot needed for an empty target", preRestorePath == "", preRestorePath)
	fmt.Printf("  STOPWATCH restore+verify leg: %s (restore-only: %s)\n", verifyPlusRestoreDuration, restoreDuration)

	if restoreErr == nil {
		restoredConn, err := openReadOnly(restoreDBPath)
		if err != nil {
			check("restored DB: opens read-only", false, err.Error())
		} else {
			assertContent("restored DB", restoredConn, sc)
			restoredConn.Close()
		}
	}

	// ── NEGATIVE CONTROL: corrupt a COPY of the backup artifact ──
	section("Corrupt-backup negative control — RED path (mandatory)")
	corruptPath := filepath.Join(scratchRoot, "corrupt_backup.db")
	if err := copyFile(backupPath, corruptPath); err != nil {
		fmt.Println("FATAL: cannot copy backup for corruption test:", err)
		os.Exit(2)
	}
	corruptBytesMidFile(corruptPath)
	check("corrupt artifact: file still exists post-corruption (bytes flipped, not truncated)", fileExists(corruptPath), "")

	corruptVerifyErr := dbengine.VerifyBackup(driverName, corruptPath)
	checkExpectError("VerifyBackup: REFUSES the corrupted artifact", corruptVerifyErr, true)

	corruptRestoreTarget := filepath.Join(scratchRoot, "restored-from-corrupt", "ph_holdings.db")
	_ = os.MkdirAll(filepath.Dir(corruptRestoreTarget), 0700)
	if err := guardTarget(corruptRestoreTarget); err != nil {
		fmt.Println("FATAL: liveness guard refused the drill's own corrupt-restore target (bug):", err)
		os.Exit(2)
	}
	_, corruptRestoreErr := dbengine.Restore(driverName, corruptRestoreTarget, corruptPath, time.Now())
	checkExpectError("Restore: REFUSES to restore from the corrupted artifact", corruptRestoreErr, true)
	check("corrupt-restore: target file was NOT created (Restore refused before copying)", !fileExists(corruptRestoreTarget), "")

	if corruptVerifyErr == nil || corruptRestoreErr == nil {
		fmt.Println("  *** REAL FINDING: corruption was not detected by VerifyBackup/Restore. ***")
		fmt.Println("  *** This is a genuine gap in the machinery, reported honestly, not papered over. ***")
	}

	// ── Summary ──
	section("Summary")
	fmt.Printf("Scratch root (left in place for inspection): %s\n", scratchRoot)
	fmt.Printf("Backup leg:        %s\n", backupDuration)
	fmt.Printf("Restore+verify leg: %s\n", verifyPlusRestoreDuration)
	if failures == 0 {
		fmt.Println("\nRESTORE DRILL GREEN — all assertions passed, both negative controls fired red as required.")
	} else {
		fmt.Printf("\nRESTORE DRILL RED — %d assertion(s) failed. See [FAIL] lines above.\n", failures)
	}
	if failures > 0 {
		os.Exit(1)
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// corruptBytesMidFile flips a run of bytes in the middle of the file —
// a bit-rot / partial-write style corruption, not a truncation, so the file
// size and header remain plausible and only the content is wrong.
func corruptBytesMidFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("cannot read %s for corruption: %v", path, err))
	}
	mid := len(data) / 2
	end := mid + 256
	if end > len(data) {
		end = len(data)
	}
	for i := mid; i < end; i++ {
		data[i] ^= 0xFF
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		panic(fmt.Sprintf("cannot write corrupted %s: %v", path, err))
	}
}
