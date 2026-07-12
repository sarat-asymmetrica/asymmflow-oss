//go:build !windows

package main

// ============================================================================
// hardware_id_keystore_other.go — non-Windows passthrough.
//
// There is no cross-platform, dependency-free OS keystore available here
// without adding CGO (Keychain on macOS, libsecret/DBus on Linux both
// require CGO or a running desktop session/daemon that a headless server
// process cannot assume). Rather than fake a keystore with a stub that
// looks encrypted but isn't, this is an HONEST plaintext passthrough: the
// sidecar stays exactly as it was before this change on non-Windows
// platforms. This is a documented, intentional platform limitation, not an
// oversight — do not silently "fix" it by bolting on CGO-based keychain
// bindings without discussing the CGO-ban tradeoff first.
// ============================================================================

// keystoreAvailable reports whether a native OS keystore is usable on this
// platform. Always false outside Windows — callers must fall back to the
// plaintext sidecar.
func keystoreAvailable() bool {
	return false
}

// keystoreProtect is unavailable on non-Windows platforms.
func keystoreProtect(plaintext []byte) ([]byte, error) {
	return nil, errKeystoreUnavailable
}

// keystoreUnprotect is unavailable on non-Windows platforms.
func keystoreUnprotect(protected []byte) ([]byte, error) {
	return nil, errKeystoreUnavailable
}
