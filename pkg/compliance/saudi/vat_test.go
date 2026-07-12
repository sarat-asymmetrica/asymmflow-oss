package saudi

import (
	"math"
	"strings"
	"testing"

	"ph_holdings_app/pkg/compliance"
)

func TestStandardRateCalculation(t *testing.T) {
	engine := New()
	// Realistic scenario: restaurant bill of SAR 230.00 → VAT 34.50.
	result, err := engine.CalculateTax(compliance.TaxableTransaction{
		Amount: 230.00, Currency: "SAR", Category: "services",
	})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 34.50 {
		t.Errorf("tax = %v, want 34.50", result.TaxAmount)
	}
	if result.TotalAmount != 264.50 {
		t.Errorf("total = %v, want 264.50", result.TotalAmount)
	}
	if result.Jurisdiction != compliance.JurisdictionSaudi {
		t.Errorf("jurisdiction = %v, want SA", result.Jurisdiction)
	}
	if len(result.TaxBreakdown) != 1 || result.TaxBreakdown[0].Rate != StandardVATRate {
		t.Errorf("breakdown = %+v, want single 15%% VAT component", result.TaxBreakdown)
	}
}

func TestHalalaRounding(t *testing.T) {
	engine := New()
	// 33.33 × 15% = 4.9995 → 5.00 (half away from zero at 2 decimals).
	result, err := engine.CalculateTax(compliance.TaxableTransaction{Amount: 33.33})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TaxAmount != 5.00 {
		t.Errorf("tax = %v, want 5.00", result.TaxAmount)
	}
	if result.TotalAmount != 38.33 {
		t.Errorf("total = %v, want 38.33", result.TotalAmount)
	}
}

func TestZeroRatedExemptOutOfScope(t *testing.T) {
	engine := New()
	cases := []struct {
		category string
		wantCode string
	}{
		{"exports", CategoryZeroRated},
		{"international_transport", CategoryZeroRated},
		{"qualifying_medicines", CategoryZeroRated},
		{"investment_metals", CategoryZeroRated},
		{"financial_services", CategoryExempt},
		{"life_insurance", CategoryExempt},
		{"residential_real_estate", CategoryExempt},
		{"out_of_scope", CategoryOutOfScope},
	}
	for _, tc := range cases {
		result, err := engine.CalculateTax(compliance.TaxableTransaction{Amount: 1000, Category: tc.category})
		if err != nil {
			t.Fatalf("CalculateTax(%s): %v", tc.category, err)
		}
		if result.TaxAmount != 0 {
			t.Errorf("category %s: tax = %v, want 0", tc.category, result.TaxAmount)
		}
		if result.TotalAmount != 1000 {
			t.Errorf("category %s: total = %v, want 1000", tc.category, result.TotalAmount)
		}
		if got := CategoryCode(tc.category); got != tc.wantCode {
			t.Errorf("CategoryCode(%s) = %s, want %s", tc.category, got, tc.wantCode)
		}
	}
	if got := CategoryCode("services"); got != CategoryStandard {
		t.Errorf("CategoryCode(services) = %s, want S", got)
	}
}

func TestReverseChargeImportedServices(t *testing.T) {
	engine := New()
	// Realistic scenario: KSA business buys SAR 10,000 of consulting from a
	// foreign supplier. Buyer self-accounts 1,500 VAT; supplier is paid 10,000.
	for _, tx := range []compliance.TaxableTransaction{
		{Amount: 10000, Category: "imported_services"},
		{Amount: 10000, Category: "services", SupplierType: "non_resident"},
		{Amount: 10000, Category: "services", SupplierType: "foreign"},
	} {
		result, err := engine.CalculateTax(tx)
		if err != nil {
			t.Fatalf("CalculateTax(%+v): %v", tx, err)
		}
		if result.TaxAmount != 1500 {
			t.Errorf("%+v: tax = %v, want 1500", tx, result.TaxAmount)
		}
		if result.TotalAmount != 10000 {
			t.Errorf("%+v: total payable = %v, want 10000 (VAT self-accounted, not paid to supplier)", tx, result.TotalAmount)
		}
		if len(result.TaxBreakdown) != 1 || !strings.Contains(result.TaxBreakdown[0].Name, "reverse charge") {
			t.Errorf("%+v: breakdown = %+v, want reverse-charge component", tx, result.TaxBreakdown)
		}
	}
	// A domestic service supplier is NOT reverse charge.
	result, err := engine.CalculateTax(compliance.TaxableTransaction{Amount: 10000, Category: "services", SupplierType: "resident"})
	if err != nil {
		t.Fatalf("CalculateTax: %v", err)
	}
	if result.TotalAmount != 11500 {
		t.Errorf("domestic services: total = %v, want 11500", result.TotalAmount)
	}
}

func TestNegativeAmountRejected(t *testing.T) {
	if _, err := New().CalculateTax(compliance.TaxableTransaction{Amount: -1}); err == nil {
		t.Error("negative amount should error")
	}
}

func TestValidVATNumber(t *testing.T) {
	valid := []string{"310122393500003", "399999999999993", " 310122393500003 "}
	invalid := []string{"", "12345", "310122393500001", "110122393500003", "31012239350000", "3101223935000031", "31012239350000a"}
	for _, v := range valid {
		if !ValidVATNumber(v) {
			t.Errorf("ValidVATNumber(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if ValidVATNumber(v) {
			t.Errorf("ValidVATNumber(%q) = true, want false", v)
		}
	}
}

func TestValidateInvoiceRealistic(t *testing.T) {
	engine := New()
	// Realistic simplified invoice: café bill, 2 lines, standard-rated.
	inv := compliance.InvoiceData{
		InvoiceNumber: "SME00062",
		SellerTaxID:   "310122393500003",
		Amount:        86.96,
		TaxAmount:     13.04,
		Currency:      "SAR",
		LineItems: []compliance.LineItemData{
			{Description: "Karak chai", Quantity: 2, UnitPrice: 8.48, TaxRate: 0.15},
			{Description: "Shakshuka plate", Quantity: 1, UnitPrice: 70.00, TaxRate: 0.15},
		},
	}
	result, err := engine.ValidateInvoice(inv)
	if err != nil {
		t.Fatalf("ValidateInvoice: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, errors = %v", result.Errors)
	}
}

func TestValidateInvoiceErrors(t *testing.T) {
	engine := New()
	inv := compliance.InvoiceData{
		InvoiceNumber: "",
		SellerTaxID:   "12345",
		BuyerTaxID:    "not-a-vat",
		Amount:        -5,
		TaxAmount:     -1,
		Currency:      "USD",
		LineItems: []compliance.LineItemData{
			{Quantity: 0, UnitPrice: -2, TaxRate: 0.10},
		},
	}
	result, err := engine.ValidateInvoice(inv)
	if err != nil {
		t.Fatalf("ValidateInvoice: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid")
	}
	wantFragments := []string{
		"invoice number is required",
		"seller VAT registration number",
		"buyer VAT registration number",
		"invoice amount cannot be negative",
		"tax amount cannot be negative",
		"quantity must be greater than zero",
		"unit price cannot be negative",
		"tax rate must be 0% or 15%",
	}
	joined := strings.Join(result.Errors, " | ")
	for _, want := range wantFragments {
		if !strings.Contains(joined, want) {
			t.Errorf("errors missing %q; got %v", want, result.Errors)
		}
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, " "), "SAR") {
		t.Errorf("expected SAR currency warning, got %v", result.Warnings)
	}
}

func TestValidateInvoiceArithmeticWarning(t *testing.T) {
	engine := New()
	inv := compliance.InvoiceData{
		InvoiceNumber: "INV-1",
		SellerTaxID:   "310122393500003",
		Amount:        100.00,
		TaxAmount:     12.00, // should be 15.00 for all-standard lines
		Currency:      "SAR",
	}
	result, err := engine.ValidateInvoice(inv)
	if err != nil {
		t.Fatalf("ValidateInvoice: %v", err)
	}
	if !result.Valid {
		t.Errorf("arithmetic mismatch should warn, not error: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected arithmetic warning")
	}
}

func TestTaxRatesCatalog(t *testing.T) {
	rates := New().TaxRates()
	if len(rates) != 5 {
		t.Fatalf("want 5 rate entries, got %d", len(rates))
	}
	var std, rc bool
	for _, r := range rates {
		if r.Name == "Standard VAT" && math.Abs(r.Rate-0.15) < 1e-9 {
			std = true
		}
		if r.Name == "Reverse charge" {
			rc = true
		}
	}
	if !std || !rc {
		t.Errorf("rate catalog missing standard/reverse-charge entries: %+v", rates)
	}
}

func TestRegistryIntegration(t *testing.T) {
	registry := compliance.NewRegistry()
	registry.Register(New())
	engine, ok := registry.Get(compliance.JurisdictionSaudi)
	if !ok {
		t.Fatal("Saudi engine not found in registry")
	}
	if engine.Name() != "Saudi Arabia VAT (ZATCA)" {
		t.Errorf("Name = %q", engine.Name())
	}
}
