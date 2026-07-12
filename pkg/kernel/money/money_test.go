package money

import (
	"math"
	"testing"
)

// usd is a test-local helper for constructing USD amounts.
func usd(v float64) Amount { return FromMinor(int64(math.Round(v*100)), "USD", 2) }

// TestBHDConstructorPrecision: BHD(125.5556) must store 125556 minor units
// and export as 125.556 (truncated at 3dp, not 125.5556).
func TestBHDConstructorPrecision(t *testing.T) {
	a := BHD(125.5556)
	if a.Minor() != 125556 {
		t.Errorf("minor units: got %d, want 125556", a.Minor())
	}
	got := a.Float64()
	want := 125.556
	if got != want {
		t.Errorf("Float64: got %v, want %v", got, want)
	}
}

// TestBHDRoundTrip: for a set of representative values, BHD(x).Float64() must
// equal math.Round(x*1000)/1000 — the same behaviour as the legacy round() helpers.
func TestBHDRoundTrip(t *testing.T) {
	cases := []float64{0, -50.5, 125.5556, 77.1254, 1000.999}
	for _, v := range cases {
		got := BHD(v).Float64()
		want := math.Round(v*1000) / 1000
		if got != want {
			t.Errorf("BHD(%v).Float64() = %v, want %v", v, got, want)
		}
	}
}

// TestBHDZero checks IsZero boundary.
func TestBHDZero(t *testing.T) {
	if !BHD(0).IsZero() {
		t.Error("BHD(0).IsZero() should be true")
	}
	if BHD(1).IsZero() {
		t.Error("BHD(1).IsZero() should be false")
	}
}

// TestBHDPositiveNegative checks sign predicates.
func TestBHDPositiveNegative(t *testing.T) {
	pos := BHD(100.5)
	neg := BHD(-100.5)
	zero := BHD(0)

	if !pos.IsPositive() {
		t.Error("BHD(100.5).IsPositive() should be true")
	}
	if pos.IsNegative() {
		t.Error("BHD(100.5).IsNegative() should be false")
	}

	if !neg.IsNegative() {
		t.Error("BHD(-100.5).IsNegative() should be true")
	}
	if neg.IsPositive() {
		t.Error("BHD(-100.5).IsPositive() should be false")
	}

	if zero.IsPositive() {
		t.Error("BHD(0).IsPositive() should be false")
	}
	if zero.IsNegative() {
		t.Error("BHD(0).IsNegative() should be false")
	}
}

// TestAddSameCurrency: BHD(100.5) + BHD(200.25) == BHD(300.75).
func TestAddSameCurrency(t *testing.T) {
	a := BHD(100.5)
	b := BHD(200.25)
	result, err := a.Add(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := BHD(300.75)
	if result.Minor() != want.Minor() {
		t.Errorf("Add: got minor=%d, want minor=%d", result.Minor(), want.Minor())
	}
}

// TestAddCrossCurrencyRejects: BHD(100) + USD(200) must return an error.
func TestAddCrossCurrencyRejects(t *testing.T) {
	_, err := BHD(100).Add(usd(200))
	if err == nil {
		t.Error("expected error when adding BHD to USD, got nil")
	}
}

// TestSubSameCurrency: BHD(300) - BHD(100.5) == BHD(199.5).
func TestSubSameCurrency(t *testing.T) {
	result, err := BHD(300).Sub(BHD(100.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := BHD(199.5)
	if result.Minor() != want.Minor() {
		t.Errorf("Sub: got minor=%d, want minor=%d", result.Minor(), want.Minor())
	}
}

// TestNegate: BHD(100).Negate() == BHD(-100).
func TestNegate(t *testing.T) {
	got := BHD(100).Negate().Float64()
	if got != -100.0 {
		t.Errorf("Negate: got %v, want -100", got)
	}
}

// TestFormatBHD verifies the canonical Format() output.
func TestFormatBHD(t *testing.T) {
	cases := []struct {
		input Amount
		want  string
	}{
		{BHD(125.5), "BHD 125.500"},
		{BHD(0), "BHD 0.000"},
		{BHD(-50.123), "BHD -50.123"},
	}
	for _, c := range cases {
		got := c.input.Format()
		if got != c.want {
			t.Errorf("Format(%v): got %q, want %q", c.input.Float64(), got, c.want)
		}
	}
}

// TestRoundFloat64Equivalence: RoundFloat64(v, 3) must match the legacy
// math.Round(v*1000)/1000 expression for a set of edge values.
func TestRoundFloat64Equivalence(t *testing.T) {
	cases := []float64{0, 77.1254, 125.5556, -0.0005, 999.9999, 0.0004}
	for _, v := range cases {
		got := RoundFloat64(v, 3)
		want := math.Round(v*1000) / 1000
		if got != want {
			t.Errorf("RoundFloat64(%v, 3) = %v, want %v", v, got, want)
		}
	}
}

// TestRoundFloat64Scale2: USD-style 2dp rounding.
func TestRoundFloat64Scale2(t *testing.T) {
	cases := []struct {
		input float64
		want  float64
	}{
		{125.555, 125.56},
		{125.554, 125.55},
		{0, 0},
		{-1.005, -1.0}, // IEEE 754 edge: math.Round rounds half away from zero
	}
	for _, c := range cases {
		got := RoundFloat64(c.input, 2)
		want := math.Round(c.input*100) / 100
		// We verify against stdlib, not a hard-coded expected value, to stay
		// consistent with floating-point behaviour across platforms.
		if got != want {
			t.Errorf("RoundFloat64(%v, 2) = %v, want %v", c.input, got, want)
		}
	}
	// Additionally verify the spec example: 125.555 → 125.56
	if got := RoundFloat64(125.555, 2); got != 125.56 {
		t.Errorf("RoundFloat64(125.555, 2): got %v, want 125.56", got)
	}
}

// TestFromMinor: FromMinor(125500, "BHD", 3).Float64() == 125.5
func TestFromMinor(t *testing.T) {
	a := FromMinor(125500, "BHD", 3)
	got := a.Float64()
	want := 125.5
	if got != want {
		t.Errorf("FromMinor(125500, BHD, 3).Float64() = %v, want %v", got, want)
	}
	if a.Currency() != "BHD" {
		t.Errorf("Currency: got %q, want BHD", a.Currency())
	}
	if a.Scale() != 3 {
		t.Errorf("Scale: got %d, want 3", a.Scale())
	}
	if a.Minor() != 125500 {
		t.Errorf("Minor: got %d, want 125500", a.Minor())
	}
}
