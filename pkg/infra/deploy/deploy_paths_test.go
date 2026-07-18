package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withExeDir overrides the executable-directory seam for the duration of the
// test and restores it after. It also writes a deployment.json declaring slug
// into that dir when slug != "", so DeploymentSlug resolves deterministically
// without mutating the process-wide overlay.
func withExeDir(t *testing.T, slug string) string {
	t.Helper()
	dir := t.TempDir()
	if slug != "" {
		if err := os.WriteFile(filepath.Join(dir, DeploymentJSONName), []byte(`{"slug":"`+slug+`"}`), 0o600); err != nil {
			t.Fatalf("write deployment.json: %v", err)
		}
	}
	prev := exeDirFn
	exeDirFn = func() string { return dir }
	t.Cleanup(func() { exeDirFn = prev })
	return dir
}

// setPlatformDataRoot points the per-user data root at a temp dir on whatever
// OS the test runs on (Windows reads APPDATA; elsewhere the home dir), and
// returns that root. slugRoot(slug) then lives beneath it.
func setPlatformDataRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", root)
	} else {
		t.Setenv("HOME", root)
	}
	return root
}

func TestResolveDatabasePath_TotalOrder(t *testing.T) {
	dataDir := filepath.Join("X:", "data-plane")

	// 1. PH_DB_PATH (absolute) wins over everything.
	exeWithFlag := t.TempDir()
	if err := os.WriteFile(filepath.Join(exeWithFlag, PortableFlagName), []byte("x"), 0o600); err != nil {
		t.Fatalf("write portable.flag: %v", err)
	}
	abs := filepath.Join(t.TempDir(), "custom.db")
	got, src := resolveDatabasePath(resolveInputs{phDBPath: abs, exeDir: exeWithFlag, dataDir: dataDir})
	if src != "PH_DB_PATH" || got != filepath.Clean(abs) {
		t.Fatalf("PH_DB_PATH must win: got (%s, %s) want (%s, PH_DB_PATH)", got, src, abs)
	}

	// 1b. PH_DB_PATH relative → resolved against CWD (wails dev CWD dev-DB flow).
	got, src = resolveDatabasePath(resolveInputs{phDBPath: "ph_holdings.db", cwd: filepath.Join("Y:", "proj"), exeDir: exeWithFlag, dataDir: dataDir})
	if src != "PH_DB_PATH" || got != filepath.Clean(filepath.Join("Y:", "proj", "ph_holdings.db")) {
		t.Fatalf("relative PH_DB_PATH must resolve against CWD: got (%s, %s)", got, src)
	}

	// 2. No env, portable.flag present → exe-dir data\.
	got, src = resolveDatabasePath(resolveInputs{exeDir: exeWithFlag, dataDir: dataDir})
	if src != "portable.flag" || got != filepath.Join(exeWithFlag, "data", DBFileName) {
		t.Fatalf("portable.flag must route to exe data dir: got (%s, %s)", got, src)
	}

	// 3. No env, no portable.flag → DataDir.
	exeNoFlag := t.TempDir()
	got, src = resolveDatabasePath(resolveInputs{exeDir: exeNoFlag, dataDir: dataDir})
	if src != "DataDir" || got != filepath.Join(dataDir, DBFileName) {
		t.Fatalf("default must route to DataDir: got (%s, %s)", got, src)
	}
}

func TestDeploymentSlug_DeploymentJSONWins(t *testing.T) {
	withExeDir(t, "AsymmFlow-PH")
	if got := DeploymentSlug(); got != "AsymmFlow-PH" {
		t.Fatalf("deployment.json slug must win: got %q", got)
	}
}

func TestDeploymentSlug_DefaultsWhenAbsent(t *testing.T) {
	withExeDir(t, "") // no deployment.json; overlay not configured → default
	if got := DeploymentSlug(); got != DefaultSlug {
		t.Fatalf("absent slug sources must default to %q: got %q", DefaultSlug, got)
	}
}

func TestSlugIsolation_TwoSlugsTwoTrees(t *testing.T) {
	root := setPlatformDataRoot(t)

	withExeDir(t, "AsymmFlow-PH")
	phData := DataDir()
	phIdentity := IdentityDir()

	withExeDir(t, "AcmeCorp-Prod")
	acmeData := DataDir()

	if phData == acmeData {
		t.Fatalf("two slugs must yield two data trees; both = %s", phData)
	}
	// Each slug tree lives under <root>\Asymmetrica\<slug>\...
	wantPHData := filepath.Join(root, VendorDir, "AsymmFlow-PH", "data")
	wantPHIdentity := filepath.Join(root, VendorDir, "AsymmFlow-PH", "identity")
	if !strings.EqualFold(phData, wantPHData) {
		t.Fatalf("PH data plane: got %s want %s", phData, wantPHData)
	}
	if !strings.EqualFold(phIdentity, wantPHIdentity) {
		t.Fatalf("PH identity plane: got %s want %s", phIdentity, wantPHIdentity)
	}
	wantAcmeData := filepath.Join(root, VendorDir, "AcmeCorp-Prod", "data")
	if !strings.EqualFold(acmeData, wantAcmeData) {
		t.Fatalf("Acme data plane: got %s want %s", acmeData, wantAcmeData)
	}
	// Identity and data planes are siblings, never the same directory.
	if phData == phIdentity {
		t.Fatalf("identity and data planes must be distinct: both = %s", phData)
	}
}

// TestLegacyAppDataNeverReturned is the explicit invariant proof: no resolver
// can produce the legacy %APPDATA%\AsymmFlow directory (or any child of it).
// The new namespace interposes Asymmetrica\<slug>, making the collision
// structurally impossible even for a slug literally named "AsymmFlow".
func TestLegacyAppDataNeverReturned(t *testing.T) {
	root := setPlatformDataRoot(t)
	legacy := filepath.Join(root, "AsymmFlow") // the forbidden directory

	assertNotUnderLegacy := func(label, path string) {
		t.Helper()
		clean := filepath.Clean(path)
		if clean == legacy || strings.HasPrefix(clean, legacy+string(os.PathSeparator)) {
			t.Fatalf("%s returned a path under the legacy dir %s: %s", label, legacy, clean)
		}
	}

	for _, slug := range []string{"AsymmFlow-PH", "AsymmFlow-Dev", "AsymmFlow", "anything"} {
		withExeDir(t, slug)
		assertNotUnderLegacy("DataDir", DataDir())
		assertNotUnderLegacy("IdentityDir", IdentityDir())
		assertNotUnderLegacy("ResolveDatabasePath", ResolveDatabasePath())
		// And the data plane must actually live under the Asymmetrica namespace.
		if !strings.HasPrefix(filepath.Clean(DataDir()), filepath.Join(root, VendorDir)+string(os.PathSeparator)) {
			t.Fatalf("slug %q: DataDir escaped the %s namespace: %s", slug, VendorDir, DataDir())
		}
	}

	// Even the portable and PH_DB_PATH branches never route to the legacy dir.
	exeWithFlag := t.TempDir()
	_ = os.WriteFile(filepath.Join(exeWithFlag, PortableFlagName), []byte("x"), 0o600)
	got, _ := resolveDatabasePath(resolveInputs{exeDir: exeWithFlag, dataDir: DataDir()})
	assertNotUnderLegacy("portable resolution", got)
}

func TestPackagedSeedPath_FindsAdjacentSeed(t *testing.T) {
	dir := withExeDir(t, "AsymmFlow-Dev")

	if got := PackagedSeedPath(); got != "" {
		t.Fatalf("no seed present should return empty, got %s", got)
	}

	// data\ph_holdings.db is preferred over a bare exe-adjacent copy.
	dataSeed := filepath.Join(dir, "data", DBFileName)
	if err := os.MkdirAll(filepath.Dir(dataSeed), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(dataSeed, []byte("x"), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	if got := PackagedSeedPath(); got != filepath.Clean(dataSeed) {
		t.Fatalf("packaged seed: got %s want %s", got, dataSeed)
	}
}
