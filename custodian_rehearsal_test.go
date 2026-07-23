package main

// ============================================================================
// custodian_rehearsal_test.go — CW1-A recovery rehearsal harness.
//
// Mission: FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md, CW1-A (C2 — the
// key custody map + rehearsed recovery). This file is ADDITIVE ONLY: it adds
// no new runtime behavior to field_crypto.go / hardware_id_keystore_windows.go
// / settings_service.go, it only drives their real exported/package-level
// functions from a test.
//
// Why this lives in package main (not scripts/custodian/*.go): FieldCrypto,
// ImportKeyMaterial, keystoreProtect/keystoreUnprotect, and keystoreAvailable
// are unexported symbols of package main. A separate `go run` program cannot
// import package main, so the only way to drive the REAL production code
// (not a reimplementation) is a _test.go file in this package, per the spec's
// documented fallback ("a *_test.go style harness run via a script is
// acceptable"). Run with:
//
//	go test -run TestCustodianRehearsal -v .
//
// or via the thin wrapper: go run ./scripts/custodian/rehearse_recovery
//
// Doctrine followed (spec §1):
//   - RED FIRST: every recovery check first proves loss (wrong key / wrong
//     salt) fails loudly, content-asserted — never on exit code alone.
//   - GREEN SECOND: only then does the documented recovery path run and get
//     asserted byte-identical.
//   - Copies only: every artifact this test writes lives under
//     %TEMP%\custodian-rehearsal\<run-ts>\... A guard (scratchGuard) refuses
//     to write anywhere else, and refuses any path containing "ph_holdings.db"
//     or the "#" character, and is itself negative-tested.
//   - No real secrets: every key/salt/hardware-id used here is generated
//     fresh by the test (crypto/rand) — never a value from a real deployment.
// ============================================================================

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Scratch-dir guard — negative-tested below (TestScratchGuardRefusesUnsafePaths)
// ---------------------------------------------------------------------------

// scratchGuard refuses to let the rehearsal touch anything that is not a
// path under the given scratch root, and refuses any path that mentions
// "ph_holdings.db" (the live database filename) or contains "#" (banned per
// the mission brief — some tooling on this machine mishandles it in paths).
func scratchGuard(scratchRoot, path string) error {
	absRoot, err := filepath.Abs(scratchRoot)
	if err != nil {
		return fmt.Errorf("scratchGuard: cannot resolve scratch root: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("scratchGuard: cannot resolve path: %w", err)
	}
	if strings.Contains(absPath, "#") {
		return fmt.Errorf("scratchGuard: refused — path contains '#': %s", absPath)
	}
	if strings.Contains(strings.ToLower(absPath), "ph_holdings.db") {
		return fmt.Errorf("scratchGuard: refused — path names the live database file: %s", absPath)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("scratchGuard: refused — path %s is outside scratch root %s", absPath, absRoot)
	}
	return nil
}

// TestScratchGuardRefusesUnsafePaths is the guard's own negative test — the
// mission brief requires the liveness/scope guard to have one.
func TestScratchGuardRefusesUnsafePaths(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name string
		path string
		ok   bool
	}{
		{"inside scratch root", filepath.Join(root, "sentinel.txt"), true},
		{"nested inside scratch root", filepath.Join(root, "sub", "dir", "salt.hex"), true},
		{"outside scratch root", filepath.Join(filepath.Dir(root), "escape.txt"), false},
		{"names ph_holdings.db", filepath.Join(root, "ph_holdings.db"), false},
		{"names ph_holdings.db mixed case", filepath.Join(root, "PH_Holdings.DB"), false},
		{"contains hash", filepath.Join(root, "bad#path", "x.txt"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := scratchGuard(root, tc.path)
			if tc.ok && err != nil {
				t.Fatalf("expected %s to be allowed, got error: %v", tc.path, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected %s to be REFUSED, but scratchGuard allowed it", tc.path)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scratch-dir scaffolding
// ---------------------------------------------------------------------------

// newRehearsalScratchRoot creates %TEMP%\custodian-rehearsal\<run-ts>\ and
// returns it. Using os.TempDir() directly (not t.TempDir()) because the
// mission brief pins the location explicitly for auditability of what a
// human re-running this by hand will find; it's still cleaned up at the end
// of the test via t.Cleanup.
func newRehearsalScratchRoot(t *testing.T) string {
	t.Helper()
	ts := time.Now().Format("20060102_150405.000000")
	ts = strings.ReplaceAll(ts, ".", "_") // keep it filesystem-friendly, no '#' ever introduced
	root := filepath.Join(os.TempDir(), "custodian-rehearsal", ts)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("failed to create scratch root %s: %v", root, err)
	}
	if strings.Contains(root, "#") {
		t.Fatalf("scratch root %s contains '#' — refusing per mission brief", root)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	t.Logf("scratch root: %s", root)
	return root
}

func writeGuarded(t *testing.T, scratchRoot, path string, data []byte) {
	t.Helper()
	if err := scratchGuard(scratchRoot, path); err != nil {
		t.Fatalf("writeGuarded: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("writeGuarded: mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("writeGuarded: write: %v", err)
	}
}

func randomHex(t *testing.T, nBytes int) string {
	t.Helper()
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("randomHex: %v", err)
	}
	return hex.EncodeToString(buf)
}

// ---------------------------------------------------------------------------
// TestCustodianRehearsal_FieldCrypto — the C2 rehearsal (red then green).
//
// This drives the REAL recovery API a steward would use in production:
//
//	(export side, on the original machine)  fc.ExportKeyMaterial() / fc.ExportSalt()
//	(import side, on the recovery machine)  ImportKeyMaterial(masterHex, saltHex)
//
// exactly as wired into the app bindings ExportEncryptionBackup() /
// ImportEncryptionBackup() in app_setup_documents_surface.go:815-888. The
// "original instance" here is built via ImportKeyMaterial with a freshly
// generated master key + salt (crypto/rand) rather than via NewFieldCrypto(),
// because NewFieldCrypto() resolves its salt file via loadOrCreateSalt(),
// which is NOT parameterized (tries exe-adjacent dir, then deploy.DataDir())
// and has no override seam — redirecting it would mean either mutating
// process-global state (os.Executable()'s directory cannot be faked; APPDATA
// env mutation would race every other test in this package that reads
// deploy.DataDir()) or editing field_crypto.go, which is out of scope for
// this wave (ADD-ONLY rule). ImportKeyMaterial/ExportKeyMaterial/ExportSalt
// are themselves the exact real, unedited recovery code path (field_crypto.go
// :267-309) and are what an owner's recovery ritual actually calls — so this
// rehearsal is faithful to the documented procedure. Recorded as a residue
// item in CW1A_REPORT.md: NewFieldCrypto()'s own file-path resolution
// (loadOrCreateSalt, the ENCRYPTION_MASTER_KEY-env branch) is exercised by
// the existing test suite's own coverage, not re-proven here.
func TestCustodianRehearsal_FieldCrypto(t *testing.T) {
	scratchRoot := newRehearsalScratchRoot(t)
	originalDir := filepath.Join(scratchRoot, "original-machine")
	recoveryDir := filepath.Join(scratchRoot, "recovery-machine") // simulates a DIFFERENT machine
	if err := os.MkdirAll(originalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(recoveryDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sentinel := fmt.Sprintf("CUSTODIAN-SENTINEL-%d", time.Now().UnixNano())
	t.Logf("sentinel plaintext: %q", sentinel)

	// --- Scenario build: the "original machine" instance -------------------
	masterHex := randomHex(t, 32) // 32 bytes -> 64 hex chars, matches the >=64-hex-char direct-use branch
	saltBytes := make([]byte, 32)
	if _, err := rand.Read(saltBytes); err != nil {
		t.Fatalf("salt generation: %v", err)
	}
	saltHex := hex.EncodeToString(saltBytes)

	original, err := ImportKeyMaterial(masterHex, saltHex)
	if err != nil {
		t.Fatalf("failed to construct the 'original machine' FieldCrypto instance: %v", err)
	}

	// Record what an owner's recovery envelope WOULD hold, as plaintext files
	// under scratch (this is the rehearsal's own bookkeeping, not production
	// storage — RECOVERY_ENVELOPE_TEMPLATE.md governs the real thing and
	// carries NO material).
	writeGuarded(t, scratchRoot, filepath.Join(originalDir, "master_key.hex"), []byte(masterHex))
	writeGuarded(t, scratchRoot, filepath.Join(originalDir, ".field_crypto_salt.hex"), []byte(saltHex))

	ciphertext, err := original.Encrypt(sentinel)
	if err != nil {
		t.Fatalf("Encrypt() on original instance failed: %v", err)
	}
	writeGuarded(t, scratchRoot, filepath.Join(originalDir, "sentinel.ciphertext"), []byte(ciphertext))
	t.Logf("sentinel ciphertext (base64): %s", ciphertext)

	// Round-trip sanity on the original instance itself, before touching loss/recovery.
	if plain, err := original.Decrypt(ciphertext); err != nil || plain != sentinel {
		t.Fatalf("sanity round-trip on original instance failed: plain=%q err=%v", plain, err)
	}

	// --- RED FIRST: prove loss -----------------------------------------------

	t.Run("RED_wrong_master_key_fails", func(t *testing.T) {
		wrongMasterHex := randomHex(t, 32)
		wrong, err := ImportKeyMaterial(wrongMasterHex, saltHex) // right salt, wrong master
		if err != nil {
			t.Fatalf("ImportKeyMaterial with wrong master key should still construct (bad key != bad hex), got: %v", err)
		}
		plain, decErr := wrong.Decrypt(ciphertext)
		if decErr == nil {
			t.Fatalf("SECURITY FAILURE: decrypt succeeded with the WRONG master key — got plaintext %q. AES-GCM authentication should have rejected this.", plain)
		}
		if plain == sentinel {
			t.Fatalf("SECURITY FAILURE: wrong-key decrypt returned the correct sentinel plaintext")
		}
		t.Logf("confirmed RED: wrong master key -> Decrypt() error = %v (plaintext NOT recovered)", decErr)
	})

	t.Run("RED_missing_salt_file_means_missing_salt_hex_fails", func(t *testing.T) {
		// "Missing salt file" in production means the owner has the master key
		// but not the salt — they cannot reconstruct saltHex at all. We model
		// the failure mode precisely: attempting recovery with a DIFFERENT
		// (e.g. re-generated/guessed) salt in place of the lost original,
		// which is the only thing a steward without the real salt could try.
		substituteSaltBytes := make([]byte, 32)
		if _, err := rand.Read(substituteSaltBytes); err != nil {
			t.Fatal(err)
		}
		substituteSaltHex := hex.EncodeToString(substituteSaltBytes)

		wrong, err := ImportKeyMaterial(masterHex, substituteSaltHex) // right master, wrong/"regenerated" salt
		if err != nil {
			t.Fatalf("ImportKeyMaterial with substitute salt should still construct, got: %v", err)
		}
		plain, decErr := wrong.Decrypt(ciphertext)
		if decErr == nil {
			t.Fatalf("SECURITY FAILURE: decrypt succeeded with the WRONG salt — got plaintext %q", plain)
		}
		if plain == sentinel {
			t.Fatalf("SECURITY FAILURE: wrong-salt decrypt returned the correct sentinel plaintext")
		}
		t.Logf("confirmed RED: missing/substitute salt -> Decrypt() error = %v (plaintext NOT recovered)", decErr)

		// And the literal "file missing" case: reading a salt file that was
		// never written must fail at the I/O layer, before any crypto happens.
		neverWritten := filepath.Join(recoveryDir, "salt-that-does-not-exist.hex")
		if _, statErr := os.Stat(neverWritten); !os.IsNotExist(statErr) {
			t.Fatalf("test setup error: %s unexpectedly exists", neverWritten)
		}
		if _, readErr := os.ReadFile(neverWritten); readErr == nil {
			t.Fatal("SECURITY FAILURE: reading a nonexistent salt file unexpectedly succeeded")
		} else {
			t.Logf("confirmed RED: reading missing salt file -> %v", readErr)
		}
	})

	// --- GREEN SECOND: the documented recovery path -------------------------

	t.Run("GREEN_documented_recovery_path_round_trips", func(t *testing.T) {
		// Simulate "new machine": read the envelope files back from disk exactly
		// as a steward would (they wrote master_key_hex/salt_hex down from
		// ExportEncryptionBackup(); here we read them back from the scratch
		// files written above), then call the same ImportKeyMaterial() that
		// ImportEncryptionBackup() calls in production
		// (app_setup_documents_surface.go:860).
		recoveredMasterHex, err := os.ReadFile(filepath.Join(originalDir, "master_key.hex"))
		if err != nil {
			t.Fatalf("failed to read back master key envelope file: %v", err)
		}
		recoveredSaltHex, err := os.ReadFile(filepath.Join(originalDir, ".field_crypto_salt.hex"))
		if err != nil {
			t.Fatalf("failed to read back salt envelope file: %v", err)
		}
		recoveredCiphertext, err := os.ReadFile(filepath.Join(originalDir, "sentinel.ciphertext"))
		if err != nil {
			t.Fatalf("failed to read back sentinel ciphertext: %v", err)
		}

		recovered, err := ImportKeyMaterial(string(recoveredMasterHex), string(recoveredSaltHex))
		if err != nil {
			t.Fatalf("ImportKeyMaterial (the documented recovery path) failed: %v", err)
		}

		plain, err := recovered.Decrypt(string(recoveredCiphertext))
		if err != nil {
			t.Fatalf("GREEN recovery FAILED: Decrypt() on the recovered instance errored: %v", err)
		}
		if plain != sentinel {
			t.Fatalf("GREEN recovery FAILED: recovered plaintext %q != original sentinel %q (not byte-identical)", plain, sentinel)
		}
		t.Logf("confirmed GREEN: recovered instance decrypted sentinel byte-identical: %q", plain)

		// Cross-check exported material matches what was written (proves
		// ExportKeyMaterial/ExportSalt themselves, not just the file bookkeeping).
		if got := original.ExportKeyMaterial(); got != masterHex {
			t.Fatalf("ExportKeyMaterial() = %s, want %s", got, masterHex)
		}
		if got := original.ExportSalt(); got != saltHex {
			t.Fatalf("ExportSalt() = %s, want %s", got, saltHex)
		}
	})
}

// ---------------------------------------------------------------------------
// TestCustodianRehearsal_DPAPIKeystore — reality check for the Windows DPAPI
// sidecar wrapping (hardware_id_keystore_windows.go). Uses the REAL
// keystoreProtect/keystoreUnprotect/keystoreAvailable functions.
// ---------------------------------------------------------------------------

func TestCustodianRehearsal_DPAPIKeystore(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("DPAPI is Windows-only; honest skip on this platform (matches hardware_id_keystore_test.go's own skip contract)")
	}
	if !keystoreAvailable() {
		t.Skip("keystoreAvailable() = false on this Windows machine; nothing to rehearse")
	}

	scratchRoot := newRehearsalScratchRoot(t)
	simulatedHardwareID := "REHEARSAL-SIMULATED-HW-ID-" + randomHex(t, 8) // never a real hardware ID

	t.Run("same_process_round_trip_succeeds", func(t *testing.T) {
		protected, err := keystoreProtect([]byte(simulatedHardwareID))
		if err != nil {
			t.Fatalf("keystoreProtect failed: %v", err)
		}
		blobPath := filepath.Join(scratchRoot, "dpapi-blob.bin")
		writeGuarded(t, scratchRoot, blobPath, protected)

		readBack, err := os.ReadFile(blobPath)
		if err != nil {
			t.Fatal(err)
		}
		unprotected, err := keystoreUnprotect(readBack)
		if err != nil {
			t.Fatalf("keystoreUnprotect failed on a blob written to disk and read back (same process/user/machine): %v", err)
		}
		if string(unprotected) != simulatedHardwareID {
			t.Fatalf("round-trip mismatch: got %q, want %q", unprotected, simulatedHardwareID)
		}
		t.Logf("confirmed: DPAPI (CRYPTPROTECT_LOCAL_MACHINE) round-trips within this machine/session")
	})

	t.Run("RED_corrupted_blob_fails", func(t *testing.T) {
		// Approximates the "wrong machine" / "DPAPI machine key differs" class
		// of failure: a DPAPI blob is opaque ciphertext bound to this
		// machine's DPAPI master key. We cannot literally move this process
		// to a second machine inside a test, so we flip bytes in a real
		// protected blob to prove CryptUnprotectData fails closed on
		// malformed/foreign-looking input rather than silently returning
		// something. This is a PROXY, not a true cross-machine test — see
		// residue.
		protected, err := keystoreProtect([]byte(simulatedHardwareID))
		if err != nil {
			t.Fatalf("keystoreProtect failed: %v", err)
		}
		corrupted := append([]byte(nil), protected...)
		for i := range corrupted {
			corrupted[i] ^= 0xFF // flip every bit
		}
		unprotected, err := keystoreUnprotect(corrupted)
		if err == nil {
			t.Fatalf("SECURITY FAILURE: keystoreUnprotect succeeded on a corrupted/foreign-looking blob, returned %q", unprotected)
		}
		t.Logf("confirmed RED: corrupted DPAPI blob (proxy for cross-machine/foreign-key) -> error = %v", err)
	})

	t.Log("RESIDUE: true cross-machine DPAPI failure (same blob, different physical machine's DPAPI master key) cannot be simulated on one dev machine and is NOT claimed here — see CW1A_REPORT.md residue list. Likewise 'profile loss' (new Windows user account, same machine): CRYPTPROTECT_LOCAL_MACHINE scope means, per the source comment in hardware_id_keystore_windows.go:16-20, any local user/process on this machine can unprotect — so a same-machine profile loss is EXPECTED to still succeed by design, but no second Windows user account was available on this dev machine to verify live.")
}

// ---------------------------------------------------------------------------
// TestCustodianRehearsal_NoRealSecretsInThisFile is a self-check: greps this
// file's own source for anything that looks like a 64-hex-char string
// (a plausible real 32-byte key) that isn't clearly test-generated. Belt and
// suspenders alongside the manual grep recorded in CW1A_REPORT.md.
// ---------------------------------------------------------------------------

func TestCustodianRehearsal_NoRealSecretsInThisFile(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("could not resolve this file's own path via runtime.Caller")
	}
	data, err := os.ReadFile(thisFile)
	if err != nil {
		t.Fatalf("failed to read own source for self-check: %v", err)
	}
	if hasLiteralHex64(string(data)) {
		t.Fatal("this file appears to contain a literal 64-hex-char string; all key/salt material in this harness must be generated at runtime via crypto/rand, never hardcoded")
	}
}

func hasLiteralHex64(src string) bool {
	const hexAlphabet = "0123456789abcdefABCDEF"
	run := 0
	for _, r := range src {
		if strings.ContainsRune(hexAlphabet, r) {
			run++
			if run >= 64 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}
