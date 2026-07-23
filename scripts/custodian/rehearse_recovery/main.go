// Command rehearse_recovery is a thin, repo-root-relative wrapper around the
// real CW1-A recovery rehearsal, which lives in
// ../../../custodian_rehearsal_test.go (package main, at the module root).
//
// Why a wrapper instead of the rehearsal itself: FieldCrypto,
// ImportKeyMaterial, and the DPAPI keystoreProtect/keystoreUnprotect
// functions the rehearsal drives are unexported symbols of the module-root
// `package main` (the Wails app). A separate `go run` program cannot import
// package main, so the rehearsal logic itself must be a _test.go file in
// that package — see the doc comment at the top of custodian_rehearsal_test.go.
// This command exists only so "go run ./scripts/custodian/rehearse_recovery"
// (the invocation named in the mission brief) does something real: it shells
// out to `go test -run <the rehearsal tests> -v .` from the module root and
// streams the output, so a human never has to know the go-test invocation.
//
// Usage:
//
//	go run ./scripts/custodian/rehearse_recovery
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "rehearse_recovery: could not locate module root (go.mod):", err)
		os.Exit(1)
	}

	fmt.Println("== CW1-A Recovery Rehearsal ==")
	fmt.Println("Module root:", moduleRoot)
	fmt.Println("Running: go test -run 'TestCustodianRehearsal|TestScratchGuardRefusesUnsafePaths' -v .")
	fmt.Println("(everything this writes lives under", filepath.Join(os.TempDir(), "custodian-rehearsal"), "and is removed at test cleanup)")
	fmt.Println()

	cmd := exec.Command("go", "test", "-run", "TestCustodianRehearsal|TestScratchGuardRefusesUnsafePaths", "-v", ".")
	cmd.Dir = moduleRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "\nrehearse_recovery: rehearsal FAILED —", err)
		os.Exit(1)
	}

	fmt.Println("\nrehearse_recovery: rehearsal PASSED (red-then-green, see transcript above).")
	if runtime.GOOS != "windows" {
		fmt.Println("NOTE: TestCustodianRehearsal_DPAPIKeystore honest-skips on this platform (DPAPI is Windows-only).")
	}
}

// findModuleRoot walks upward from the working directory (which `go run`
// sets to this package's own directory) looking for go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}
