package india

// This file scaffolds the PAN-level AATO (Aggregate Annual Turnover) roll-up
// that feeds the G4 HSN-digit mandate and the G8 e-invoicing applicability
// indicator. AATO is computed PAN-level, across every sibling GSTIN sharing
// one PAN (§0 G1) — summing invoice history is a later mission's job (it
// needs a document store); this file defines the pure tier/digit logic and
// the seam (AATOSource) a later engine plugs a real summing implementation
// into.

// HSNTier classifies a PAN's Aggregate Annual Turnover against the N/N
// 78/2020-CT boundary (§0 G4).
type HSNTier int

const (
	// TierUpTo5Cr is AATO at or below the threshold: 4-digit HSN is
	// mandatory on B2B invoices, optional on B2C.
	TierUpTo5Cr HSNTier = iota
	// TierAbove5Cr is AATO strictly above the threshold ("AATO > ₹5cr",
	// §0 G4/G8 wording): 6-digit HSN is mandatory on ALL invoices.
	TierAbove5Cr
)

// AATOTier classifies aatoINR against thresholdINR. The boundary itself is
// exclusive on the lower tier — AATO exactly at the threshold is still
// TierUpTo5Cr, matching the "> ₹5cr" wording in §0 G4/G8.
func AATOTier(aatoINR, thresholdINR float64) HSNTier {
	if aatoINR > thresholdINR {
		return TierAbove5Cr
	}
	return TierUpTo5Cr
}

// HSNDigitsForExport is the HSN digit count mandated on export invoices
// regardless of tier (§0 G4: "8-digit on exports always").
func HSNDigitsForExport() int {
	return 8
}

// RequiredHSNDigits returns the mandatory HSN/SAC digit count for a
// non-export line, given the PAN's tier and whether the invoice is B2B.
// TierUpTo5Cr: 4 digits on B2B, 0 (optional) on B2C. TierAbove5Cr: 6 digits
// on both B2B and B2C.
func RequiredHSNDigits(tier HSNTier, b2b bool) int {
	switch tier {
	case TierAbove5Cr:
		return 6
	default: // TierUpTo5Cr
		if b2b {
			return 4
		}
		return 0
	}
}

// AATOSource resolves a PAN's Aggregate Annual Turnover for a fiscal year,
// summing across every sibling GSTIN registered under that PAN. The document
// store implements this in a later mission; this package only consumes the
// interface so the tier/digit logic stays independent of storage.
type AATOSource interface {
	AATOForPAN(pan string, fyStartYear int) (float64, error)
}

// ResolveAATO returns the AATO figure to use: overrideINR wins whenever it is
// positive (fresh deployments without in-app invoice history have no other
// way to know their turnover), otherwise the computed figure is used as-is.
func ResolveAATO(computed float64, overrideINR float64) float64 {
	if overrideINR > 0 {
		return overrideINR
	}
	return computed
}
