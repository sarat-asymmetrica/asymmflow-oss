package main

// ============================================================================
// hardware_id_keystore_test.go — guards the C1 hardware-ID → OS keystore
// (DPAPI) migration. These tests exercise resolveHardwareID()/persistHardwareID()
// through the existing hardwareIDSidecarPathOverride seam (settings_service.go)
// so they never touch the real DB directory or the process-global
// getHardwareID() memoization.
//
// Windows-specific assertions (DPAPI file presence/format, migration retiring
// the plaintext file) skip honestly on non-Windows platforms, where
// hardware_id_keystore_other.go is an intentional plaintext passthrough — see
// that file's doc comment. The platform-neutral parts of the resolve/migrate
// logic (e.g. that a value keeps resolving, that plaintext-when-no-keystore
// still works) run everywhere.
// ============================================================================

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withSidecarOverride points hardwareIDSidecarPathOverride at a fresh temp
// file path for the duration of the test and restores it afterward.
func withSidecarOverride(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sidecarPath := filepath.Join(dir, ".hardware_id")

	orig := hardwareIDSidecarPathOverride
	hardwareIDSidecarPathOverride = sidecarPath
	t.Cleanup(func() { hardwareIDSidecarPathOverride = orig })

	return sidecarPath
}

// TestHardwareIDKeystore_FreshInstall_WindowsIsDPAPIOnly proves that on a
// fresh install (no sidecar of any kind yet) on a platform with a keystore,
// persistHardwareID() writes ONLY the keystore-protected sidecar and never
// creates a plaintext .hardware_id file.
func TestHardwareIDKeystore_FreshInstall_WindowsIsDPAPIOnly(t *testing.T) {
	if !keystoreAvailable() {
		t.Skip("no OS keystore on this platform; plaintext passthrough is intentional, see hardware_id_keystore_other.go")
	}

	sidecar := withSidecarOverride(t)

	id1, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() first call returned error: %v", err)
	}
	if strings.TrimSpace(id1) == "" {
		t.Fatal("resolveHardwareID() returned an empty id")
	}

	if _, err := os.Stat(sidecar); err == nil {
		t.Fatalf("plaintext sidecar %s was created on a fresh install with a keystore available; want DPAPI-only", sidecar)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error stat-ing plaintext sidecar: %v", err)
	}

	kpath := keystoreSidecarPath(sidecar)
	protected, err := os.ReadFile(kpath)
	if err != nil {
		t.Fatalf("expected keystore sidecar %s to exist after fresh install, read failed: %v", kpath, err)
	}
	if len(protected) == 0 {
		t.Fatalf("keystore sidecar %s is empty", kpath)
	}
	if strings.Contains(string(protected), id1) {
		t.Fatalf("keystore sidecar %s appears to contain the plaintext hardware ID unprotected", kpath)
	}

	decrypted, err := keystoreUnprotect(protected)
	if err != nil {
		t.Fatalf("keystoreUnprotect() failed on the persisted blob: %v", err)
	}
	if strings.TrimSpace(string(decrypted)) != id1 {
		t.Fatalf("keystoreUnprotect(persisted blob) = %q, want %q", decrypted, id1)
	}

	// A second resolution must return the identical value, sourced from the
	// keystore sidecar (readKeystoreSidecar), proving steady-state reuse.
	id2, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() second call returned error: %v", err)
	}
	if id2 != id1 {
		t.Fatalf("resolveHardwareID() second call = %q, want stable value %q", id2, id1)
	}
}

// TestHardwareIDKeystore_UpgradedInstall_MigratesAndVerifiesRoundTrip proves
// the migration path for pre-existing installs: a plaintext sidecar left by
// an older build is protected into the keystore sidecar, the round-trip is
// verified, and only then is the plaintext file retired (renamed to a
// ".migrated" backup, never hard-deleted).
func TestHardwareIDKeystore_UpgradedInstall_MigratesAndVerifiesRoundTrip(t *testing.T) {
	if !keystoreAvailable() {
		t.Skip("no OS keystore on this platform; plaintext passthrough is intentional, see hardware_id_keystore_other.go")
	}

	sidecar := withSidecarOverride(t)

	const preExisting = "legacy-plaintext-hardware-id-value"
	if err := os.WriteFile(sidecar, []byte(preExisting), 0o600); err != nil {
		t.Fatalf("failed to seed pre-existing plaintext sidecar: %v", err)
	}

	id, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() returned error: %v", err)
	}
	if id != preExisting {
		t.Fatalf("resolveHardwareID() = %q, want migrated value %q (the resolved VALUE must not change across migration)", id, preExisting)
	}

	kpath := keystoreSidecarPath(sidecar)
	protected, err := os.ReadFile(kpath)
	if err != nil {
		t.Fatalf("expected keystore sidecar %s after migration, read failed: %v", kpath, err)
	}
	decrypted, err := keystoreUnprotect(protected)
	if err != nil {
		t.Fatalf("keystoreUnprotect() failed on migrated blob: %v", err)
	}
	if strings.TrimSpace(string(decrypted)) != preExisting {
		t.Fatalf("migrated keystore sidecar round-trips to %q, want %q", decrypted, preExisting)
	}

	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Fatalf("expected plaintext sidecar %s to be retired (renamed away) after verified migration, stat err = %v", sidecar, err)
	}
	backup := sidecar + ".migrated"
	backupData, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("expected plaintext backup %s to exist after migration, read failed: %v", backup, err)
	}
	if strings.TrimSpace(string(backupData)) != preExisting {
		t.Fatalf("plaintext backup %s content = %q, want %q", backup, backupData, preExisting)
	}

	// A subsequent resolution must now be served from the keystore sidecar,
	// not by re-reading (now-absent) plaintext or re-deriving.
	id2, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() post-migration call returned error: %v", err)
	}
	if id2 != preExisting {
		t.Fatalf("resolveHardwareID() post-migration = %q, want %q", id2, preExisting)
	}
}

// TestHardwareIDKeystore_ProtectFailure_KeepsPlaintextIntact simulates a
// DPAPI failure during migration (protected input rejected) and proves the
// plaintext sidecar is left completely untouched and the value still
// resolves — i.e. key material is never stranded between stores.
func TestHardwareIDKeystore_ProtectFailure_KeepsPlaintextIntact(t *testing.T) {
	sidecar := withSidecarOverride(t)

	const preExisting = "legacy-plaintext-hardware-id-value-2"
	if err := os.WriteFile(sidecar, []byte(preExisting), 0o600); err != nil {
		t.Fatalf("failed to seed pre-existing plaintext sidecar: %v", err)
	}

	// Force the keystore sidecar WRITE step to fail deterministically —
	// independent of whether DPAPI itself would succeed on this machine — by
	// making keystoreSidecarPath(sidecar) collide with an existing directory
	// instead of a writable file path. os.WriteFile to a directory always
	// fails, so this exercises the same "protect succeeded but persisting it
	// failed" contract that a real DPAPI outage would hit, without needing to
	// mock Windows APIs.
	kpath := keystoreSidecarPath(sidecar)
	if err := os.MkdirAll(kpath, 0o755); err != nil {
		t.Fatalf("failed to seed directory collision at %s: %v", kpath, err)
	}

	if err := protectAndPersistKeystoreSidecar(sidecar, preExisting); err == nil {
		t.Fatal("protectAndPersistKeystoreSidecar() unexpectedly succeeded despite the keystore path colliding with a directory")
	}

	// migrateHardwareIDToKeystore must leave the plaintext sidecar alone when
	// the write step fails, and resolveHardwareID() must still resolve the
	// plaintext-sourced value — no key material is stranded between stores.
	id, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() returned error: %v", err)
	}
	if id != preExisting {
		t.Fatalf("resolveHardwareID() = %q, want plaintext-sourced %q", id, preExisting)
	}

	data, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("plaintext sidecar %s missing/unreadable after failed keystore write: %v", sidecar, err)
	}
	if strings.TrimSpace(string(data)) != preExisting {
		t.Fatalf("plaintext sidecar content = %q, want untouched %q", data, preExisting)
	}

	info, statErr := os.Stat(kpath)
	if statErr != nil || !info.IsDir() {
		t.Fatalf("expected the directory collision at %s to remain untouched (a successful write would have failed to overwrite a directory), stat err = %v", kpath, statErr)
	}
}

// TestHardwareIDKeystore_NonWindowsIsHonestPlaintextPassthrough documents and
// guards the non-Windows contract: keystoreAvailable() is false, and
// keystoreProtect/keystoreUnprotect both fail with errKeystoreUnavailable
// rather than silently no-op-ing or fabricating a fake "encrypted" format.
func TestHardwareIDKeystore_NonWindowsIsHonestPlaintextPassthrough(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has a real DPAPI keystore; this guards the non-Windows passthrough contract")
	}

	if keystoreAvailable() {
		t.Fatal("keystoreAvailable() = true on a non-Windows platform; expected honest plaintext passthrough")
	}
	if _, err := keystoreProtect([]byte("x")); err == nil {
		t.Fatal("keystoreProtect() succeeded on non-Windows platform; expected errKeystoreUnavailable")
	}
	if _, err := keystoreUnprotect([]byte("x")); err == nil {
		t.Fatal("keystoreUnprotect() succeeded on non-Windows platform; expected errKeystoreUnavailable")
	}
}

// TestHardwareIDKeystore_KeystoreSidecarPathHonorsOverride sanity-checks that
// keystoreSidecarPath is derived from (and only from) its input, so it
// automatically follows hardwareIDSidecarPathOverride without needing its
// own override seam.
func TestHardwareIDKeystore_KeystoreSidecarPathHonorsOverride(t *testing.T) {
	if got := keystoreSidecarPath(""); got != "" {
		t.Fatalf("keystoreSidecarPath(\"\") = %q, want \"\"", got)
	}

	sidecar := withSidecarOverride(t)
	got := keystoreSidecarPath(hardwareIDSidecarPath())
	want := sidecar + ".dpapi"
	if got != want {
		t.Fatalf("keystoreSidecarPath(hardwareIDSidecarPath()) = %q, want %q", got, want)
	}
}
