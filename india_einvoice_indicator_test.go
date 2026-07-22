package main

import (
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/overlay"
)

// mountedIndiaOverlayForTest builds a synthetic India-plane overlay (never a
// real GSTIN/PAN, per India Spec-01 §3 public-repo synthetic law) with the
// given threshold/override, so each subtest proves the indicator's wording
// tracks configured values rather than a hardcoded constant.
func mountedIndiaOverlayForTest(thresholdINR, overrideINR float64) *overlay.CompanyOverlay {
	return &overlay.CompanyOverlay{
		DefaultDivisionKey: "Synthetic Instruments",
		India: &overlay.IndiaCompanyConfig{
			// Meridian canon PAN/GSTIN (SYNTHETIC_IDENTITY.md) — checksum-valid
			// synthetic values, per the India-canon law (gate fix: the original
			// fixture carried a checksum-invalid GSTIN).
			PAN:                   "AABCM0472E",
			AATOOverrideINR:       overrideINR,
			EInvoiceThresholdAATO: thresholdINR,
		},
		Divisions: []overlay.DivisionProfile{
			{
				Key: "Synthetic Instruments",
				India: &overlay.IndiaDivisionProfile{
					GSTIN:     "27AABCM0472E1ZT",
					StateCode: "27",
				},
			},
		},
	}
}

func TestEInvoiceApplicability(t *testing.T) {
	saved := activeOverlay
	defer func() { activeOverlay = saved }()

	// Bare App with startupImporting bypasses RBAC for "settings:view" (the
	// very first check in requirePermission, app_auth_rbac.go) — the same
	// technique app_test.go uses for permission-gated getters, without
	// needing a full setupTestApp(t) database.
	app := &App{startupImporting: true, startupImportStartTime: time.Now()}

	t.Run("unmounted overlay", func(t *testing.T) {
		activeOverlay = &overlay.CompanyOverlay{DefaultDivisionKey: "GCC Division"}

		got := app.GetEInvoiceApplicability()

		if got.Mounted {
			t.Fatalf("expected Mounted=false for an overlay with no India plane, got %+v", got)
		}
		if !strings.Contains(got.Message, "not mounted") {
			t.Fatalf("expected message to say the plane is not mounted, got %q", got.Message)
		}
	})

	t.Run("mounted, no override (unknown AATO)", func(t *testing.T) {
		// ₹10,00,00,000 threshold — deliberately NOT the ₹5cr default, to
		// prove the message renders configured values, not a constant.
		activeOverlay = mountedIndiaOverlayForTest(100000000, 0)

		got := app.GetEInvoiceApplicability()

		if !got.Mounted {
			t.Fatalf("expected Mounted=true, got %+v", got)
		}
		if got.AATOSource != "unknown" {
			t.Fatalf("expected AATOSource=unknown with no override, got %q", got.AATOSource)
		}
		if !strings.Contains(got.Message, "10,00,00,000") {
			t.Fatalf("expected message to mention the configured threshold 10,00,00,000, got %q", got.Message)
		}
	})

	t.Run("mounted, override below threshold", func(t *testing.T) {
		activeOverlay = mountedIndiaOverlayForTest(100000000, 50000000) // 10cr threshold, 5cr AATO

		got := app.GetEInvoiceApplicability()

		if got.Applicable {
			t.Fatalf("expected Applicable=false when AATO is below threshold, got %+v", got)
		}
		if got.AATOSource != "override" {
			t.Fatalf("expected AATOSource=override, got %q", got.AATOSource)
		}
		if !strings.Contains(got.Message, "not applicable") {
			t.Fatalf("expected 'not applicable' message, got %q", got.Message)
		}
	})

	t.Run("mounted, override above threshold", func(t *testing.T) {
		activeOverlay = mountedIndiaOverlayForTest(100000000, 150000000) // 10cr threshold, 15cr AATO

		got := app.GetEInvoiceApplicability()

		if !got.Applicable {
			t.Fatalf("expected Applicable=true when AATO exceeds threshold, got %+v", got)
		}
		if got.AATOSource != "override" {
			t.Fatalf("expected AATOSource=override, got %q", got.AATOSource)
		}
		if !strings.Contains(got.Message, "APPLICABLE") {
			t.Fatalf("expected 'APPLICABLE' message, got %q", got.Message)
		}
	})
}
