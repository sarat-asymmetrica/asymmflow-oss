package india

import (
	"strings"
	"testing"
)

func TestIncomeFiveLakhRebateZeroTaxBothRegimes(t *testing.T) {
	engine := NewIncomeTax()

	oldResult := engine.CalculateOldRegime(500000, Deductions{})
	newResult := engine.CalculateNewRegime(500000)

	if oldResult.TotalTax != 0 {
		t.Fatalf("old regime tax = %v, want 0", oldResult.TotalTax)
	}
	if newResult.TotalTax != 0 {
		t.Fatalf("new regime tax = %v, want 0", newResult.TotalTax)
	}
}

func TestIncomeTenLakhComparison(t *testing.T) {
	engine := NewIncomeTax()

	comparison := engine.Compare(1000000, Deductions{Section80C: 150000, Section80D: 50000})

	if comparison.OldRegime == nil || comparison.NewRegime == nil {
		t.Fatal("comparison should include both regimes")
	}
	if comparison.Recommendation == "" {
		t.Fatal("comparison should include a recommendation")
	}
}

func TestIncomeTwentyLakhFullDeductionsOldRegimeBetter(t *testing.T) {
	engine := NewIncomeTax()

	comparison := engine.Compare(2000000, Deductions{
		Section80C:   150000,
		Section80CCD: 50000,
		Section80D:   25000,
		Section24:    200000,
		HRAExemption: 400000,
		Section80E:   100000,
		Section80G:   100000,
	})

	if comparison.Savings <= 0 {
		t.Fatalf("old regime should save tax with full deductions: %+v", comparison)
	}
	if !strings.Contains(comparison.Recommendation, "Old regime saves") {
		t.Fatalf("recommendation = %q", comparison.Recommendation)
	}
}

func TestIncomeEightLakhNoDeductionsNewRegimeBetter(t *testing.T) {
	engine := NewIncomeTax()

	comparison := engine.Compare(800000, Deductions{})

	if comparison.Savings >= 0 {
		t.Fatalf("new regime should save tax: %+v", comparison)
	}
	if !strings.Contains(comparison.Recommendation, "New regime saves") {
		t.Fatalf("recommendation = %q", comparison.Recommendation)
	}
}

func TestSurchargeAboveFiftyLakh(t *testing.T) {
	engine := NewIncomeTax()

	result := engine.CalculateNewRegime(5100000)

	if result.Surcharge <= 0 {
		t.Fatalf("surcharge should apply above 50L: %+v", result)
	}
}

func TestCessCalculatedAtFourPercent(t *testing.T) {
	engine := NewIncomeTax()

	result := engine.CalculateNewRegime(2000000)
	wantCess := roundRupee((result.TaxAmount + result.Surcharge) * 0.04)

	if result.Cess != wantCess {
		t.Fatalf("cess = %v, want %v", result.Cess, wantCess)
	}
}
