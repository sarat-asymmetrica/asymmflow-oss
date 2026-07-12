package auth

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := HashPIN("2468")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	state, err := Verify(h, "2468", LockState{}, now)
	if err != nil {
		t.Fatalf("Verify correct: %v", err)
	}
	if state.FailedAttempts != 0 || !state.LockedUntil.IsZero() {
		t.Errorf("state not reset: %+v", state)
	}

	if _, err := Verify(h, "0000", LockState{}, now); !errors.Is(err, ErrWrongPIN) {
		t.Errorf("wrong pin: got %v, want ErrWrongPIN", err)
	}
}

func TestEncodeParseRoundTrip(t *testing.T) {
	h, err := HashPIN("13579")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParsePINHash(h.Encode())
	if err != nil {
		t.Fatalf("ParsePINHash: %v", err)
	}
	if _, err := Verify(parsed, "13579", LockState{}, now); err != nil {
		t.Errorf("verify after round-trip: %v", err)
	}
	if _, err := Verify(parsed, "97531", LockState{}, now); !errors.Is(err, ErrWrongPIN) {
		t.Errorf("wrong pin after round-trip: %v", err)
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "nonsense", "pbkdf2-sha256$x$y$z", "md5$1$a$b", "pbkdf2-sha256$0$YQ==$YQ=="} {
		if _, err := ParsePINHash(bad); err == nil {
			t.Errorf("ParsePINHash(%q) should fail", bad)
		}
	}
}

func TestShortPINRejected(t *testing.T) {
	if _, err := HashPIN("123"); err == nil {
		t.Error("3-char PIN should be rejected")
	}
	if _, err := HashPIN("  12  "); err == nil {
		t.Error("whitespace-padded short PIN should be rejected")
	}
}

func TestLockoutAfterMaxAttempts(t *testing.T) {
	h, err := HashPIN("2468")
	if err != nil {
		t.Fatal(err)
	}
	state := LockState{}
	for i := 0; i < MaxAttempts; i++ {
		state, err = Verify(h, "9999", state, now)
		if !errors.Is(err, ErrWrongPIN) {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	if !state.LockedOut(now) {
		t.Fatal("should be locked out after MaxAttempts failures")
	}

	// Correct PIN during lockout is still refused.
	if _, err := Verify(h, "2468", state, now.Add(time.Second)); !errors.Is(err, ErrLockedOut) {
		t.Errorf("during lockout: got %v, want ErrLockedOut", err)
	}

	// After the window, the correct PIN succeeds and resets state.
	after := now.Add(LockoutDuration + time.Second)
	state, err = Verify(h, "2468", state, after)
	if err != nil {
		t.Fatalf("after lockout: %v", err)
	}
	if state.LockedOut(after) || state.FailedAttempts != 0 {
		t.Errorf("state not reset after lockout: %+v", state)
	}
}

func TestSaltsDiffer(t *testing.T) {
	h1, _ := HashPIN("2468")
	h2, _ := HashPIN("2468")
	if h1.Encode() == h2.Encode() {
		t.Error("two hashes of the same PIN must differ (random salt)")
	}
}
