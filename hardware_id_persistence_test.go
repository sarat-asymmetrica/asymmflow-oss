package main

// ============================================================================
// TestHardwareID_PersistsAndPrefersSidecar guards the item (g) fix: the
// hardware ID that derives the field-crypto encryption key must be stable
// across boots. resolveHardwareID() now persists its first successful
// resolution to a plaintext sidecar file next to the DB and prefers that
// persisted value on every subsequent call, so timing-dependent differences
// between the CIM/WMIC/hostname resolvers can never silently change the key.
//
// The seam: hardwareIDSidecarPathOverride (settings_service.go) lets this
// test redirect persistence to a t.TempDir() instead of the real DB
// directory. It is unexported, defaults to "" (real path) in production, and
// is the minimal change needed to make resolveHardwareID() independently
// testable without touching key-derivation logic.
//
// This test calls resolveHardwareID() directly rather than the memoized
// getHardwareID() — getHardwareID()'s sync.Once is process-global and may
// already be warmed by other tests in this package (e.g. hardware_id_test.go),
// which would make a second call a no-op regardless of this fix.
//
// NOTE (C1 update): where a native OS keystore is available (Windows/DPAPI,
// see hardware_id_keystore_windows.go), persistHardwareID() now writes ONLY
// the keystore-protected sidecar for fresh installs — no plaintext file is
// created at all (see hardware_id_keystore_test.go for that contract). This
// test is therefore keystore-aware: it asserts persistence and preference
// against whichever store is actually authoritative on this platform, rather
// than hardcoding the plaintext path.
// ============================================================================

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHardwareID_PersistsAndPrefersSidecar(t *testing.T) {
	dir := t.TempDir()
	sidecarPath := filepath.Join(dir, ".hardware_id")

	orig := hardwareIDSidecarPathOverride
	hardwareIDSidecarPathOverride = sidecarPath
	defer func() { hardwareIDSidecarPathOverride = orig }()

	// 1. No sidecar yet: resolveHardwareID() must fall through to the real
	// CIM/WMIC/hostname resolvers, succeed, and persist the result — to the
	// OS keystore sidecar when one is available, else to the plaintext file.
	id1, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() first call returned error: %v", err)
	}
	if strings.TrimSpace(id1) == "" {
		t.Fatal("resolveHardwareID() first call returned an empty id")
	}

	if keystoreAvailable() {
		kpath := keystoreSidecarPath(sidecarPath)
		protected, readErr := os.ReadFile(kpath)
		if readErr != nil {
			t.Fatalf("expected keystore sidecar to be written after first resolution, read failed: %v", readErr)
		}
		decrypted, unprotectErr := keystoreUnprotect(protected)
		if unprotectErr != nil {
			t.Fatalf("keystoreUnprotect() failed on persisted blob: %v", unprotectErr)
		}
		if strings.TrimSpace(string(decrypted)) != id1 {
			t.Fatalf("keystore sidecar content = %q, want resolved id %q", decrypted, id1)
		}
		if _, statErr := os.Stat(sidecarPath); !os.IsNotExist(statErr) {
			t.Fatalf("expected no plaintext sidecar when a keystore is available, stat err = %v", statErr)
		}
	} else {
		data, readErr := os.ReadFile(sidecarPath)
		if readErr != nil {
			t.Fatalf("expected sidecar file to be written after first resolution, read failed: %v", readErr)
		}
		if strings.TrimSpace(string(data)) != id1 {
			t.Fatalf("sidecar content = %q, want resolved id %q", string(data), id1)
		}
	}

	// 2. Overwrite whichever store is authoritative with a sentinel value
	// that could never come from a real resolver. A second call must return
	// this sentinel verbatim (trimmed) — proving the persisted value is
	// PREFERRED over re-resolving, not merely that persistence happened once.
	sentinel := "test-sentinel-hardware-id-persisted-value"
	if keystoreAvailable() {
		protected, protectErr := keystoreProtect([]byte(sentinel))
		if protectErr != nil {
			t.Fatalf("keystoreProtect(sentinel) failed: %v", protectErr)
		}
		if err := os.WriteFile(keystoreSidecarPath(sidecarPath), protected, 0o600); err != nil {
			t.Fatalf("failed to seed keystore sidecar with sentinel value: %v", err)
		}
	} else {
		if err := os.WriteFile(sidecarPath, []byte(sentinel+"\n"), 0o600); err != nil {
			t.Fatalf("failed to seed sidecar with sentinel value: %v", err)
		}
	}

	id2, err := resolveHardwareID()
	if err != nil {
		t.Fatalf("resolveHardwareID() second call returned error: %v", err)
	}
	if id2 != sentinel {
		t.Fatalf("resolveHardwareID() = %q, want persisted sidecar value %q (persisted value was not preferred over re-resolution)", id2, sentinel)
	}
}

// TestHardwareID_SidecarPathDerivesFromDBDir sanity-checks that, absent a
// test override, the sidecar path sits next to the resolved database file
// rather than at some unrelated location — a regression here would silently
// break persistence in production (e.g. writing to the CWD instead of the
// app's data directory).
func TestHardwareID_SidecarPathDerivesFromDBDir(t *testing.T) {
	orig := hardwareIDSidecarPathOverride
	hardwareIDSidecarPathOverride = ""
	defer func() { hardwareIDSidecarPathOverride = orig }()

	sidecar := hardwareIDSidecarPath()
	dbPath := getDatabasePath()

	if strings.TrimSpace(dbPath) == "" {
		if sidecar != "" {
			t.Fatalf("hardwareIDSidecarPath() = %q, want \"\" when getDatabasePath() is empty", sidecar)
		}
		return
	}

	wantDir := filepath.Dir(dbPath)
	if filepath.Dir(sidecar) != wantDir {
		t.Fatalf("hardwareIDSidecarPath() dir = %q, want %q (next to DB)", filepath.Dir(sidecar), wantDir)
	}
	if filepath.Base(sidecar) != ".hardware_id" {
		t.Fatalf("hardwareIDSidecarPath() base = %q, want %q", filepath.Base(sidecar), ".hardware_id")
	}
}
