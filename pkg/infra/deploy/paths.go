// Package deploy owns the deployment layout: the three-plane directory
// namespace (code / identity / data), deterministic database-path resolution,
// and the boot-time update contract (seed-into-absence, backup-before-migrate,
// downgrade refusal, schema stamping).
//
// It replaces the six-priority path archaeology that used to live in
// config.go (getDatabasePath). Resolution is now configuration, not inference:
// a total three-step order with one dev escape hatch and one explicit portable
// marker. Nothing here ever reads or writes the legacy %APPDATA%\AsymmFlow
// directory — the new namespace makes that collision structurally impossible.
//
// The plane doctrine (see FABLE_CAMPAIGN_DEPLOYMENT.md §0):
//
//	Code     — replaced wholesale every update; the installer, and only it.
//	Identity — installed once, edited rarely; installer-if-absent, then human.
//	Data     — never touched by any installer; the app, after a backup.
package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"ph_holdings_app/pkg/overlay"
)

const (
	// DefaultSlug keys the three-plane layout for an un-branded substrate build.
	// A sovereign deployment overrides it via deployment.json (installer) or the
	// overlay's deployment.slug.
	DefaultSlug = "AsymmFlow-Dev"

	// VendorDir is the top-level namespace under the platform data directory.
	// Every deployment's slug tree lives beneath it, so two deployments on one
	// machine can never collide and neither can ever be the legacy layout.
	VendorDir = "Asymmetrica"

	// DBFileName is the canonical database filename inside a data plane.
	DBFileName = "ph_holdings.db"

	// PortableFlagName is the explicit marker file that opts a deployment into
	// portable mode: the data plane lives in data\ next to the executable
	// instead of the per-user platform directory. Portable mode is a decision,
	// not an inference from what happens to exist where.
	PortableFlagName = "portable.flag"

	// DeploymentJSONName is the exe-adjacent slug declaration the installer
	// writes. It is the authoritative slug source (order-independent), so path
	// resolution never depends on the overlay having loaded first.
	DeploymentJSONName = "deployment.json"
)

// deploymentJSON is the schema of the exe-adjacent deployment.json.
type deploymentJSON struct {
	Slug string `json:"slug"`
}

// exeDir returns the directory the running executable lives in, or "" if it
// cannot be resolved. It is the single source of truth for "where the code
// plane is". Overridable in tests via exeDirFn.
var exeDirFn = defaultExeDir

func defaultExeDir() string {
	exePath, err := os.Executable()
	if err != nil || strings.TrimSpace(exePath) == "" {
		return ""
	}
	return filepath.Dir(exePath)
}

func exeDir() string { return exeDirFn() }

// slugFromDeploymentJSON reads the slug from an exe-adjacent deployment.json.
// Returns "" when the file is absent, unreadable, unparseable, or blank.
func slugFromDeploymentJSON(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, DeploymentJSONName))
	if err != nil {
		return ""
	}
	var dj deploymentJSON
	if err := json.Unmarshal(data, &dj); err != nil {
		return ""
	}
	return strings.TrimSpace(dj.Slug)
}

// DeploymentSlug resolves the slug that keys the three-plane directory layout.
//
// Precedence (§8 bootstrap-order resolution): an exe-adjacent deployment.json
// wins — installers know their slug, and resolving it from a standalone file
// means path resolution never waits on the overlay to load. Failing that, the
// active overlay's deployment.slug is consulted. Failing both, DefaultSlug.
//
// The overlay fallback is best-effort: because the earliest data-path read
// (config load) happens before the overlay is active, relocating the data
// plane should be done via deployment.json (or PH_DB_PATH), not overlay.json.
func DeploymentSlug() string {
	if slug := slugFromDeploymentJSON(exeDir()); slug != "" {
		return slug
	}
	// overlay.Active() never returns nil; DeploymentSlug() defaults internally.
	return overlay.Active().DeploymentSlug()
}

// slugRoot is the per-user root of a deployment's identity+data planes:
// %APPDATA%\Asymmetrica\<slug> on Windows, ~/.local/share/asymmetrica/<slug>
// elsewhere. Returns "" when the platform data directory cannot be resolved.
func slugRoot(slug string) string {
	if runtime.GOOS == "windows" {
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); appData != "" {
			return filepath.Join(appData, VendorDir, slug)
		}
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".local", "share", strings.ToLower(VendorDir), slug)
	}
	return ""
}

// CodeDir returns the executable's own directory (informational — it is where
// the code plane lives and where deployment.json / portable.flag are read).
func CodeDir() string { return exeDir() }

// IdentityDir returns the identity plane: <slugRoot>\identity. Installed once,
// edited rarely; carries overlay.json / branding.
func IdentityDir() string { return filepath.Join(slugRoot(DeploymentSlug()), "identity") }

// DataDir returns the data plane: <slugRoot>\data. Holds the database,
// attachments\, exports\, backups\, and future mesh\ + keys\. Never touched by
// any installer. This is the sole replacement for the retired appDataDirPath
// and it is structurally incapable of returning the legacy layout.
func DataDir() string { return filepath.Join(slugRoot(DeploymentSlug()), "data") }

// resolveInputs is the environment a database-path resolution reads. Isolating
// it keeps resolveDatabasePath a pure function the gate tests drive with no
// process-global mutation.
type resolveInputs struct {
	phDBPath string // PH_DB_PATH value (may be relative → resolved against cwd)
	cwd      string // working directory, for relative PH_DB_PATH (wails dev)
	exeDir   string // code-plane directory, for portable.flag detection
	dataDir  string // slug-resolved data plane (DataDir())
}

// resolveDatabasePath is the TOTAL resolution order (§6.1). There is no fourth
// step: DATABASE_PATH, CWD scanning, exe-dir search, and packaged-path pinning
// were all retired.
//
//  1. PH_DB_PATH — dev escape hatch. A relative value resolves against the
//     working directory, preserving the CWD dev-DB flow for `wails dev`.
//  2. portable.flag next to the exe → <exeDir>\data\ph_holdings.db.
//  3. DataDir()\ph_holdings.db — the per-user data plane. The default.
func resolveDatabasePath(in resolveInputs) (path, source string) {
	if v := strings.TrimSpace(in.phDBPath); v != "" {
		if filepath.IsAbs(v) {
			return filepath.Clean(v), "PH_DB_PATH"
		}
		if strings.TrimSpace(in.cwd) != "" {
			return filepath.Clean(filepath.Join(in.cwd, v)), "PH_DB_PATH"
		}
		return filepath.Clean(v), "PH_DB_PATH"
	}

	if dir := strings.TrimSpace(in.exeDir); dir != "" {
		if fileExists(filepath.Join(dir, PortableFlagName)) {
			return filepath.Join(dir, "data", DBFileName), "portable.flag"
		}
	}

	return filepath.Join(in.dataDir, DBFileName), "DataDir"
}

// ResolveDatabasePath returns the absolute path the live database should occupy,
// per the total three-step order. It performs no I/O beyond checking for the
// portable flag and never seeds, migrates, or replaces anything — that is the
// update contract's job (see EnsureDatabase).
func ResolveDatabasePath() string {
	cwd, _ := os.Getwd()
	path, _ := resolveDatabasePath(resolveInputs{
		phDBPath: os.Getenv("PH_DB_PATH"),
		cwd:      cwd,
		exeDir:   exeDir(),
		dataDir:  DataDir(),
	})
	return path
}

// ResolveDatabasePathVerbose is ResolveDatabasePath plus the source label, for
// boot-time logging ("loud" PH_DB_PATH logging per §6.1).
func ResolveDatabasePathVerbose() (path, source string) {
	cwd, _ := os.Getwd()
	return resolveDatabasePath(resolveInputs{
		phDBPath: os.Getenv("PH_DB_PATH"),
		cwd:      cwd,
		exeDir:   exeDir(),
		dataDir:  DataDir(),
	})
}

// PackagedSeedPath locates the packaged synthetic-canon database that ships in
// the code plane (data\ph_holdings.db or ph_holdings.db next to the exe). It is
// a SEED, copied into an absent data plane once — never pinned as the live DB.
// Returns "" when no packaged seed is present (e.g. `wails dev`).
func PackagedSeedPath() string {
	dir := exeDir()
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	for _, candidate := range []string{
		filepath.Join(dir, "data", DBFileName),
		filepath.Join(dir, DBFileName),
	} {
		if fileExists(candidate) {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
