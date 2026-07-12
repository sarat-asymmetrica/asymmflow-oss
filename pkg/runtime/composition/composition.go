// Package composition is the shared composition seam every AsymmFlow vertical
// boots through: overlay (deployment identity) → SQLite database → event bus →
// compliance registry + hook. The trading app (app.go startup) and the
// hospitality vertical (cmd/hospitality) wire the SAME seam; what differs per
// vertical is configuration — overlay search dirs, DSN pragmas, GORM options,
// and which tax engines are registered — never the wiring itself.
//
// The seam deliberately exposes STAGES rather than one monolithic Build():
// the trading app wires compliance late (after bulk bootstrap imports, so
// boot-time backfills don't flood the hook) while hospitality wires it before
// its domain service. Each stage is idempotent on the fields it populates.
package composition

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/compliance"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/overlay"
)

// Root holds everything the composition stages produce. Fields are populated
// by the stage methods; callers read them directly after each stage.
type Root struct {
	Overlay  *overlay.CompanyOverlay
	DB       *gorm.DB
	Bus      events.Bus
	Registry *compliance.Registry
	Hook     *compliance.ComplianceHook
}

// NewRoot returns an empty composition root. Stages are called explicitly so
// each vertical controls WHEN in its boot sequence a dependency comes up.
func NewRoot() *Root { return &Root{} }

// LoadOverlay loads the deployment identity from the first overlay.json found
// in dirs (see overlay.LoadOverlay for the cascade), stores it on the root and
// returns it. LoadOverlay never returns nil — missing files fall back to the
// built-in synthetic defaults.
func (r *Root) LoadOverlay(dirs []string) *overlay.CompanyOverlay {
	r.Overlay = overlay.LoadOverlay(dirs)
	return r.Overlay
}

// OpenSQLite opens the vertical's database through the shared driver
// discipline (pure-Go ncruces SQLite; CGO stays banned). The DSN should come
// from SQLiteDSN so pragmas use the ncruces form the driver actually honors.
func (r *Root) OpenSQLite(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
	if cfg == nil {
		cfg = &gorm.Config{}
	}
	db, err := gorm.Open(gormlite.Open(dsn), cfg)
	if err != nil {
		return nil, fmt.Errorf("composition: open sqlite: %w", err)
	}
	r.DB = db
	return db, nil
}

// MigrateModels runs GORM AutoMigrate over a vertical's registered model-set,
// model by model, so one un-migratable table (SQLite cannot alter constraints
// on existing tables) skips just that model instead of aborting the set.
// report, if non-nil, is called after each model with its 1-based index and
// the AutoMigrate error (nil on success) — verticals hook their own
// diagnostics in. Returns (migrated, skipped) counts.
func (r *Root) MigrateModels(models []any, report func(index, total int, model string, err error)) (migrated, skipped int) {
	for i, model := range models {
		name := fmt.Sprintf("%T", model)
		err := r.DB.AutoMigrate(model)
		if report != nil {
			report(i+1, len(models), name, err)
		}
		if err != nil {
			skipped++
		} else {
			migrated++
		}
	}
	return migrated, skipped
}

// WireCompliance creates the event bus (if the root doesn't already carry
// one), a compliance registry with the given engines, and the compliance hook
// subscribed to the bus. Engine registration happens HERE and nowhere else —
// one seam, one registration site per process.
func (r *Root) WireCompliance(engines ...compliance.TaxEngine) *compliance.ComplianceHook {
	if r.Bus == nil {
		r.Bus = events.NewInMemoryBus()
	}
	r.Registry = compliance.NewRegistry()
	for _, e := range engines {
		r.Registry.Register(e)
	}
	r.Hook = compliance.NewComplianceHook(r.Registry, r.Bus)
	return r.Hook
}

// InstallDefaultBus installs the root's bus as the process-wide default so
// domain publishers that reach for events.Default() (e.g. GORM AfterCreate
// hooks in pkg/finance) publish onto it. Verticals that inject the bus
// explicitly (hospitality) don't need this.
func (r *Root) InstallDefaultBus() {
	if r.Bus != nil {
		events.SetDefault(r.Bus)
	}
}

// ---------------------------------------------------------------------------
// DSN construction
// ---------------------------------------------------------------------------

// DefaultPragmas is the standard per-connection pragma set for a production
// vertical: WAL journaling (concurrent readers alongside one writer), a 5s
// busy timeout, NORMAL fsync discipline (safe under WAL), enforced foreign
// keys and a 20MB page cache.
//
// History (Wave 3 finding): the trading app's original DSN used mattn-style
// params (?_journal_mode=WAL&_busy_timeout=5000&…) which the ncruces driver
// silently IGNORES — the pilot had been running journal_mode=DELETE with the
// driver-default 60s busy timeout. Only the ?_pragma=name(value) form below
// is honored; TestSQLiteDSN_PragmasAreHonored pins that these actually apply.
var DefaultPragmas = []string{
	"busy_timeout(5000)",
	"journal_mode(WAL)",
	"synchronous(NORMAL)",
	"foreign_keys(ON)",
	"cache_size(-20000)",
}

// SQLiteDSN builds a file DSN with ncruces-style pragma parameters, e.g.
//
//	SQLiteDSN("pos.db", "busy_timeout(5000)", "journal_mode(WAL)")
//	→ file:pos.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)
//
// Every pooled connection gets the pragmas — never rely on a one-off
// sqlDB.Exec("PRAGMA …"), which only reaches a single pooled connection.
func SQLiteDSN(path string, pragmas ...string) string {
	var b strings.Builder
	b.WriteString("file:")
	b.WriteString(filepath.ToSlash(path))
	for i, p := range pragmas {
		if i == 0 {
			b.WriteString("?_pragma=")
		} else {
			b.WriteString("&_pragma=")
		}
		b.WriteString(p)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Standard deployment search directories
// ---------------------------------------------------------------------------

// ExecutableSearchDirs returns the executable-adjacent directories a portable
// deployment keeps its config in (the exe's dir, plus macOS bundle locations).
func ExecutableSearchDirs() []string {
	exePath, err := os.Executable()
	if err != nil || strings.TrimSpace(exePath) == "" {
		return nil
	}

	exeDir := filepath.Dir(exePath)
	seen := make(map[string]struct{})
	dirs := make([]string, 0, 5)
	dirs = appendUniquePath(dirs, seen, exeDir)

	// macOS app bundles: Contents/Resources for files copied inside the
	// bundle; the folder containing the .app for sidecar deployment packages.
	if runtime.GOOS == "darwin" {
		dirs = appendUniquePath(dirs, seen, filepath.Join(exeDir, "..", "Resources"))
		dirs = appendUniquePath(dirs, seen, filepath.Join(exeDir, "..", "..", ".."))
	}

	return dirs
}

// StandardOverlayDirs is the overlay.json search cascade shared by packaged
// verticals, in precedence order:
//
//  1. executable-adjacent dirs (portable deployment, macOS bundles)
//  2. data/ sub-dir of CWD, then CWD itself (development / wails dev mode)
//  3. the platform user app-data dir (Windows %APPDATA%\<appDirName>,
//     Unix ~/.local/share/<appDirName>)
func StandardOverlayDirs(appDirName string) []string {
	dirs := make([]string, 0, 6)
	dirs = append(dirs, ExecutableSearchDirs()...)
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "data"), cwd)
	}
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			dirs = append(dirs, filepath.Join(appData, appDirName))
		}
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(homeDir, ".local", "share", appDirName))
	}
	return dirs
}

func appendUniquePath(paths []string, seen map[string]struct{}, path string) []string {
	if path == "" {
		return paths
	}
	clean := filepath.Clean(path)
	if _, exists := seen[clean]; exists {
		return paths
	}
	seen[clean] = struct{}{}
	return append(paths, clean)
}
