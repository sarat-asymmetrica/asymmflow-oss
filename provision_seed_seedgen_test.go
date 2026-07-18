//go:build seedgen

package main

// Provision a fresh SYNTHETIC seed database for the DP2 installer.
//
// The installer ships a packaged seed inside the CODE plane
// (build/bin/data/ph_holdings.db, where deploy.PackagedSeedPath() looks); on
// first boot the app's update contract copies it into an ABSENT data plane and
// reports `seeded_fresh` (DP2 gate G2). This harness produces that seed from
// the app's own provisioning path — schema migrations + critical deployment
// foundations — so it is:
//
//   - SYNTHETIC by construction: it opens an empty database and runs only
//     schema/foundation setup. It never reads any existing business database,
//     so it can carry ZERO client data (campaign invariant §4.6).
//   - REPRODUCIBLE: same code in → same schema-complete seed out, in-repo, no
//     external artifact. "Fresh data plane" per the owner's DP2 clarification.
//
// Run (from the repo root, after `wails build` has produced build/bin/):
//
//	go test -tags manual -run TestProvisionFreshInstallerSeed -count=1 .
//
// It writes build/bin/data/ph_holdings.db. The release/staging step then leaves
// it in place for `wails build -nsis` to File into the code plane.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProvisionFreshInstallerSeed(t *testing.T) {
	// The installer payload dir is a sibling of project.nsi that the wails exe
	// build never touches, so the seed staged here survives the build→package
	// sequence. NSIS Files it into the code plane ($INSTDIR\data). Overridable
	// via INSTALLER_SEED_PATH for ad-hoc runs.
	seedPath := os.Getenv("INSTALLER_SEED_PATH")
	if strings.TrimSpace(seedPath) == "" {
		seedPath = filepath.Join("build", "windows", "installer", "payload", "ph_holdings.db")
	}
	if err := os.MkdirAll(filepath.Dir(seedPath), 0755); err != nil {
		t.Fatalf("create seed staging dir: %v", err)
	}

	// Start from a genuinely empty file so the seed can only contain what the
	// provisioning path creates — never any pre-existing business rows.
	for _, p := range []string{seedPath, seedPath + "-wal", seedPath + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			t.Fatalf("reset seed file %s: %v", p, err)
		}
	}

	db, err := gorm.Open(sqlite.Open(seedPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open fresh seed database: %v", err)
	}

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       true,
		startupImportStartTime: time.Now(),
		currentUserID:          "installer-seed-provision",
		currentUser:            &User{Base: Base{ID: "installer-seed-provision"}, Username: "installer-seed-provision"},
	}
	t.Cleanup(app.cache.Stop)

	// Mirror the real boot order: critical deployment foundations first (schema
	// subset + collaborative/expense/payroll/activity structures the app seeds
	// on startup), then the FULL trading-model migration so the packaged seed is
	// schema-complete rather than a subset the first boot would have to finish.
	if err := app.ensureCriticalDeploymentFoundations(); err != nil {
		t.Fatalf("provision foundations: %v", err)
	}
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		t.Fatalf("checkpoint WAL: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}

	// migrateDatabaseFileForContract opens its own connection, migrates the full
	// tradingModels() set, checkpoints (TRUNCATE) and closes — so it must run
	// after our connection is closed to avoid write contention on the same file.
	if err := migrateDatabaseFileForContract(seedPath); err != nil {
		t.Fatalf("full schema migration: %v", err)
	}
	for _, p := range []string{seedPath + "-wal", seedPath + "-shm"} {
		_ = os.Remove(p)
	}

	info, err := os.Stat(seedPath)
	if err != nil {
		t.Fatalf("seed not written: %v", err)
	}
	t.Logf("provisioned fresh synthetic installer seed: %s (%d bytes)", seedPath, info.Size())
}
