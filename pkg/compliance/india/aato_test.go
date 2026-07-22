package india

import "testing"

func TestAATOTierBoundary(t *testing.T) {
	const threshold = 50000000 // ₹5cr
	cases := []struct {
		aato float64
		want HSNTier
	}{
		{0, TierUpTo5Cr},
		{threshold - 1, TierUpTo5Cr},
		{threshold, TierUpTo5Cr},      // exactly ₹5cr: "> ₹5cr" wording keeps this in the lower tier
		{threshold + 1, TierAbove5Cr}, // strictly above
		{threshold * 2, TierAbove5Cr},
	}
	for _, tc := range cases {
		if got := AATOTier(tc.aato, threshold); got != tc.want {
			t.Errorf("AATOTier(%v, %v) = %v, want %v", tc.aato, threshold, got, tc.want)
		}
	}
}

func TestRequiredHSNDigits(t *testing.T) {
	cases := []struct {
		tier HSNTier
		b2b  bool
		want int
	}{
		{TierUpTo5Cr, true, 4},
		{TierUpTo5Cr, false, 0},
		{TierAbove5Cr, true, 6},
		{TierAbove5Cr, false, 6},
	}
	for _, tc := range cases {
		if got := RequiredHSNDigits(tc.tier, tc.b2b); got != tc.want {
			t.Errorf("RequiredHSNDigits(%v, b2b=%v) = %d, want %d", tc.tier, tc.b2b, got, tc.want)
		}
	}
}

func TestHSNDigitsForExportAlwaysEight(t *testing.T) {
	if got := HSNDigitsForExport(); got != 8 {
		t.Errorf("HSNDigitsForExport() = %d, want 8", got)
	}
}

func TestResolveAATO(t *testing.T) {
	cases := []struct {
		computed, override, want float64
	}{
		{1000, 0, 1000},    // no override: use computed
		{1000, -5, 1000},   // negative override treated as absent
		{1000, 2000, 2000}, // positive override wins (fresh deployment, no history)
		{0, 3000, 3000},    // computed zero, override present
	}
	for _, tc := range cases {
		if got := ResolveAATO(tc.computed, tc.override); got != tc.want {
			t.Errorf("ResolveAATO(%v, %v) = %v, want %v", tc.computed, tc.override, got, tc.want)
		}
	}
}
