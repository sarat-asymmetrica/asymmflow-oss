// Package auth provides authentication infrastructure. This file implements
// the PIN lock engine: PBKDF2-hashed PINs with constant-time verification
// and failed-attempt lockout.
//
// Greenfield Wave 2 engine (the trading app has session auth but no
// PIN/app-lock); the design re-implements a pattern proven in two reference
// desktop apps — PBKDF2-SHA256 at 200k iterations, per-PIN random salt,
// crypto/subtle comparison, N-fails → timed lockout — the AsymmFlow way:
// the engine is PURE (no database). State lives in a small LockState value
// the caller persists wherever it stores settings; every mutation returns
// the new state.
//
// Intended consumers: manager-PIN approval for sensitive actions (void,
// refund, settlement close) in the hospitality vertical, and an optional
// app lock for any deployment.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const (
	pbkdf2Iterations = 200_000
	saltBytes        = 16
	keyBytes         = 32

	// MaxAttempts failed verifications trigger a lockout.
	MaxAttempts = 5
	// LockoutDuration is how long verification is refused after MaxAttempts.
	LockoutDuration = 2 * time.Minute

	// MinPINLength guards against trivially guessable PINs.
	MinPINLength = 4
)

// ErrLockedOut is returned while the lockout window is active.
var ErrLockedOut = errors.New("auth: locked out after repeated failed attempts")

// ErrWrongPIN is returned for a well-formed but incorrect PIN.
var ErrWrongPIN = errors.New("auth: incorrect PIN")

// PINHash is a serializable PBKDF2 hash of a PIN.
// Encode/Parse round-trip it through a single settings-friendly string:
// "pbkdf2-sha256$<iterations>$<salt b64>$<key b64>".
type PINHash struct {
	Iterations int
	Salt       []byte
	Key        []byte
}

// HashPIN derives a PINHash from a PIN with a fresh random salt.
func HashPIN(pin string) (PINHash, error) {
	pin = strings.TrimSpace(pin)
	if len(pin) < MinPINLength {
		return PINHash{}, fmt.Errorf("auth: PIN must be at least %d characters", MinPINLength)
	}
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return PINHash{}, fmt.Errorf("auth: cannot generate salt: %w", err)
	}
	key := pbkdf2.Key([]byte(pin), salt, pbkdf2Iterations, keyBytes, sha256.New)
	return PINHash{Iterations: pbkdf2Iterations, Salt: salt, Key: key}, nil
}

// Encode serializes the hash for storage.
func (h PINHash) Encode() string {
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s",
		h.Iterations,
		base64.StdEncoding.EncodeToString(h.Salt),
		base64.StdEncoding.EncodeToString(h.Key))
}

// ParsePINHash deserializes an encoded hash.
func ParsePINHash(encoded string) (PINHash, error) {
	parts := strings.Split(strings.TrimSpace(encoded), "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return PINHash{}, errors.New("auth: malformed PIN hash")
	}
	var iterations int
	if _, err := fmt.Sscanf(parts[1], "%d", &iterations); err != nil || iterations < 1 {
		return PINHash{}, errors.New("auth: malformed PIN hash iterations")
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return PINHash{}, errors.New("auth: malformed PIN hash salt")
	}
	key, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return PINHash{}, errors.New("auth: malformed PIN hash key")
	}
	return PINHash{Iterations: iterations, Salt: salt, Key: key}, nil
}

// matches reports whether pin derives to the stored key, in constant time.
func (h PINHash) matches(pin string) bool {
	derived := pbkdf2.Key([]byte(strings.TrimSpace(pin)), h.Salt, h.Iterations, len(h.Key), sha256.New)
	return subtle.ConstantTimeCompare(derived, h.Key) == 1
}

// LockState tracks failed attempts and any active lockout. The zero value is
// "no failures". Callers persist it alongside the hash.
type LockState struct {
	FailedAttempts int       `json:"failed_attempts"`
	LockedUntil    time.Time `json:"locked_until"`
}

// LockedOut reports whether the lockout window is active at time now.
func (s LockState) LockedOut(now time.Time) bool {
	return !s.LockedUntil.IsZero() && now.Before(s.LockedUntil)
}

// Verify checks pin against the hash, enforcing lockout. It returns the new
// LockState in every case — the caller must persist it, or lockout does not
// actually protect anything across restarts.
//
//   - correct PIN, not locked out → (reset state, nil)
//   - wrong PIN                   → (incremented state, ErrWrongPIN);
//     the MaxAttempts-th failure sets LockedUntil = now + LockoutDuration
//   - locked out                  → (unchanged state, ErrLockedOut)
func Verify(h PINHash, pin string, state LockState, now time.Time) (LockState, error) {
	if state.LockedOut(now) {
		return state, ErrLockedOut
	}
	if h.matches(pin) {
		return LockState{}, nil
	}
	state.FailedAttempts++
	if state.FailedAttempts >= MaxAttempts {
		state.LockedUntil = now.Add(LockoutDuration)
		state.FailedAttempts = 0
	}
	return state, ErrWrongPIN
}
