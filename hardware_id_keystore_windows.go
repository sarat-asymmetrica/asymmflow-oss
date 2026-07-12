//go:build windows

package main

// ============================================================================
// hardware_id_keystore_windows.go — Windows DPAPI-backed at-rest wrapping for
// the hardware-ID sidecar.
//
// IMPORTANT: this file changes ONLY how the sidecar BYTES are stored on disk.
// It never changes the resolved hardware-ID VALUE that getHardwareID() /
// resolveHardwareID() return, and therefore never changes the field-crypto
// key-derivation formula (settings_service.go / field_crypto.go). DPAPI here
// is machine-scoped encryption-at-rest for the plaintext sidecar file, not a
// new source of the identifier.
//
// CryptProtectData/CryptUnprotectData are used with CRYPTPROTECT_LOCAL_MACHINE
// (so any local process/user context on this machine can unprotect it — this
// mirrors the sidecar's previous plaintext-on-local-disk trust model, just
// removing "copy the file off the machine and read it in a text editor" as an
// attack) and CRYPTPROTECT_UI_FORBIDDEN (never blocks boot on a UI prompt).
// ============================================================================

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// keystoreAvailable reports whether a native OS keystore (DPAPI) is usable on
// this platform. Always true on Windows.
func keystoreAvailable() bool {
	return true
}

// keystoreProtect encrypts plaintext via DPAPI (machine scope, no UI prompt)
// and returns the opaque protected blob to persist at rest.
func keystoreProtect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("keystoreProtect: empty plaintext")
	}

	in := windows.DataBlob{
		Size: uint32(len(plaintext)),
		Data: &plaintext[0],
	}
	var out windows.DataBlob

	flags := uint32(windows.CRYPTPROTECT_LOCAL_MACHINE | windows.CRYPTPROTECT_UI_FORBIDDEN)
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, flags, &out); err != nil {
		return nil, fmt.Errorf("keystoreProtect: CryptProtectData failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))

	return dataBlobBytes(&out), nil
}

// keystoreUnprotect reverses keystoreProtect, returning the original
// plaintext bytes.
func keystoreUnprotect(protected []byte) ([]byte, error) {
	if len(protected) == 0 {
		return nil, fmt.Errorf("keystoreUnprotect: empty input")
	}

	in := windows.DataBlob{
		Size: uint32(len(protected)),
		Data: &protected[0],
	}
	var out windows.DataBlob

	flags := uint32(windows.CRYPTPROTECT_UI_FORBIDDEN)
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, flags, &out); err != nil {
		return nil, fmt.Errorf("keystoreUnprotect: CryptUnprotectData failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))

	return dataBlobBytes(&out), nil
}

// dataBlobBytes copies a DPAPI-allocated DataBlob's contents into a
// Go-managed byte slice before the underlying LocalAlloc buffer is freed.
func dataBlobBytes(blob *windows.DataBlob) []byte {
	if blob == nil || blob.Data == nil || blob.Size == 0 {
		return nil
	}
	src := unsafe.Slice(blob.Data, int(blob.Size))
	out := make([]byte, len(src))
	copy(out, src)
	return out
}
