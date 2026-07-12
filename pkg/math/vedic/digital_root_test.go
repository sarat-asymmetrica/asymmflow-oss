package vedic

import "testing"

func TestDigitalRoot(t *testing.T) {
	tests := []struct {
		n    int64
		want int64
	}{
		{0, 0},
		{1, 1},
		{9, 9},
		{10, 1},
		{18, 9},
		{12345, 6},
		{-12345, 6},
	}

	for _, tt := range tests {
		if got := DigitalRoot(tt.n); got != tt.want {
			t.Fatalf("DigitalRoot(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestDigitalRootHomomorphism(t *testing.T) {
	pairs := [][2]int64{
		{12, 34},
		{999, 123456},
		{77, 52},
	}

	for _, pair := range pairs {
		if !VerifyDigitalRootHomomorphism(pair[0], pair[1]) {
			t.Fatalf("homomorphism failed for %d, %d", pair[0], pair[1])
		}
	}
}

func TestCanBeDivisibleBy9(t *testing.T) {
	if !CanBeDivisibleBy9(999) {
		t.Fatalf("999 should pass divisibility-by-9 filter")
	}
	if CanBeDivisibleBy9(998) {
		t.Fatalf("998 should not pass divisibility-by-9 filter")
	}
}

func TestDigitalRootChain(t *testing.T) {
	values := []int64{123, 456, 789}
	drs := BatchDigitalRoot(values)
	want := DigitalRoot(123 + 456 + 789)

	if got := DigitalRootChain(drs); got != want {
		t.Fatalf("DigitalRootChain(%v) = %d, want %d", drs, got, want)
	}
}

func TestNavaYoni(t *testing.T) {
	if got := NavaYoni(0); got != 9 {
		t.Fatalf("NavaYoni(0) = %d, want 9", got)
	}
	if got := NavaYoni(14); got != 5 {
		t.Fatalf("NavaYoni(14) = %d, want 5", got)
	}
}
