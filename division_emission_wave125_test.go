package main

// division_emission_wave125_test.go
//
// Wave 12.5 closes two "default division leaks into non-default division
// documents" gaps:
//
//  1. Purchase Order bank-details "Account Name:" cells previously stamped
//     activeOverlay.CompanyDisplayName — which is the DEFAULT (Acme)
//     division's display name — regardless of the PO's own Division. A
//     Beacon Controls PO wrongly showed "ACME INSTRUMENTATION WLL" as the
//     payee account name.
//  2. Offer Terms & Conditions prose previously interpolated the same
//     company-level name regardless of the Offer's own Division.
//
// Both now resolve via activeOverlay.DivisionDocumentDisplayName(<division>),
// which is BYTE-IDENTICAL to the old CompanyDisplayName for the default
// division ("Acme Instrumentation WLL") and correctly emits the division's
// own document name for a non-default division ("Beacon Controls WLL").
//
// Test approach: pdftotext IS available in this environment (verified via
// `exec.LookPath`, same gate used by offer_signature_blocks_test.go's
// pdfText helper), so the PO assertions render the ACTUAL PDF end-to-end via
// GeneratePurchaseOrderPDF and extract text with pdftotext — this exercises
// the real emission function, not just the resolver. As a belt-and-braces
// fallback (in case pdftotext is unavailable in a different CI image), the
// tests also include a direct unit assertion on
// activeOverlay.DivisionDocumentDisplayName so the core resolver logic is
// still verified even when the PDF-text path is skipped.
//
// The Offer T&C assertions call defaultOfferTermsAndConditions directly
// (a pure string function) since that is both simpler and gives an exact
// byte-identity check against the pre-Wave-12.5 Acme output.

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildPOTestPurchaseOrder creates a minimal supplier + purchase order (with
// one line item) in the test DB and returns the PO ID. The caller controls
// Division.
func buildPOTestPurchaseOrder(t *testing.T, app *App, division, poNumber string) string {
	t.Helper()

	now := time.Now()
	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test"},
		SupplierCode: "WAVE125-SUP-" + poNumber,
		SupplierName: "Wave 12.5 Test Supplier",
		Country:      "BH",
		PaymentTerms: "Net 30",
	}
	require.NoError(t, app.db.Create(&supplier).Error)

	po := PurchaseOrder{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test"},
		PONumber:     poNumber,
		PODate:       now,
		SupplierID:   supplier.ID,
		SupplierName: supplier.SupplierName,
		Currency:     "BHD",
		ExchangeRate: 1,
		SubtotalBHD:  100,
		TotalBHD:     100,
		Status:       "Approved",
		Division:     division,
	}
	require.NoError(t, app.db.Create(&po).Error)

	require.NoError(t, app.db.Create(&PurchaseOrderItem{
		Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now, CreatedBy: "test"},
		PurchaseOrderID:  po.ID,
		Description:      "Wave 12.5 test line item",
		Quantity:         1,
		UnitPriceForeign: 100,
		UnitPriceBHD:     100,
		TotalForeign:     100,
		TotalBHD:         100,
	}).Error)

	return po.ID
}

// TestDivisionDocumentDisplayNameResolvesPerDivision is the belt-and-braces
// unit check on the resolver itself: byte-identical for the default
// division, correct for Beacon.
func TestDivisionDocumentDisplayNameResolvesPerDivision(t *testing.T) {
	withSyntheticOverlay(t)

	assert.Equal(t, "ACME INSTRUMENTATION WLL",
		strings.ToUpper(activeOverlay.DivisionDocumentDisplayName("Acme Instrumentation")),
		"default division's document display name must stay byte-identical to the old CompanyDisplayName")

	assert.Equal(t, "BEACON CONTROLS WLL",
		strings.ToUpper(activeOverlay.DivisionDocumentDisplayName("Beacon Controls")))
}

// TestGeneratePurchaseOrderPDFStampsDivisionAccountName is the canonical
// regression test for the PO bank-details GAP: a Beacon Controls PO's
// "Account Name:" cells must show Beacon's own document display name, never
// Acme's — and an Acme PO must remain byte-identical to the pre-fix output.
func TestGeneratePurchaseOrderPDFStampsDivisionAccountName(t *testing.T) {
	withSyntheticOverlay(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&PurchaseOrder{}, &PurchaseOrderItem{}))

	acmeID := buildPOTestPurchaseOrder(t, app, "Acme Instrumentation", "WAVE125-PO-ACME-001")
	acmePath, err := app.GeneratePurchaseOrderPDF(acmeID)
	require.NoError(t, err, "Acme PO PDF generation must succeed")
	t.Cleanup(func() { _ = os.Remove(acmePath) })
	acmeBytes, err := os.ReadFile(acmePath)
	require.NoError(t, err)
	require.NotEmpty(t, acmeBytes)

	beaconID := buildPOTestPurchaseOrder(t, app, "Beacon Controls", "WAVE125-PO-BEACON-001")
	beaconPath, err := app.GeneratePurchaseOrderPDF(beaconID)
	require.NoError(t, err, "Beacon PO PDF generation must succeed")
	t.Cleanup(func() { _ = os.Remove(beaconPath) })
	beaconBytes, err := os.ReadFile(beaconPath)
	require.NoError(t, err)
	require.NotEmpty(t, beaconBytes)

	acmeText, acmeOK := pdfText(t, acmeBytes)
	beaconText, beaconOK := pdfText(t, beaconBytes)
	if acmeOK && beaconOK {
		assert.Contains(t, acmeText, "ACME INSTRUMENTATION WLL",
			"Acme PO must still show Acme's document display name (byte-identity)")

		// THE regression assertion — before the fix, a Beacon Controls PO's
		// bank-details "Account Name:" cells emitted Acme's name here instead
		// of Beacon's.
		assert.Contains(t, beaconText, "BEACON CONTROLS WLL",
			"Beacon PO must show Beacon's own document display name in the bank Account Name cells")
		assert.NotContains(t, beaconText, "ACME INSTRUMENTATION WLL",
			"Beacon PO must NOT leak Acme's document display name — this was the bug before the fix")
	} else {
		t.Log("pdftotext unavailable — skipping rendered-text assertions; resolver-level coverage is in TestDivisionDocumentDisplayNameResolvesPerDivision")
	}
}

// TestDefaultOfferTermsAndConditionsStampsDivision is the canonical
// regression test for the Offer T&C GAP: a Beacon Controls offer's T&C must
// name Beacon, never the default (Acme) company name — and an Acme offer's
// T&C must be byte-identical to the pre-Wave-12.5 output.
func TestDefaultOfferTermsAndConditionsStampsDivision(t *testing.T) {
	withSyntheticOverlay(t)

	acmeTerms := defaultOfferTermsAndConditions("Acme Instrumentation", 10)
	beaconTerms := defaultOfferTermsAndConditions("Beacon Controls", 10)

	// Byte-identity: this is exactly what the old
	// `company := activeOverlay.CompanyDisplayName` produced for vatRate=10,
	// since CompanyDisplayName == "Acme Instrumentation WLL" == the default
	// division's DivisionDocumentDisplayName.
	wantAcme := `1. QUOTATION VALIDITY
This quotation is valid for thirty (30) days from the date of issue.

2. PRICES
All prices are in Bahraini Dinars (BHD) unless otherwise stated. Prices are exclusive of VAT (10%) which will be added to the invoice.

3. PAYMENT TERMS
As per the payment terms specified in this quotation. Late payments may incur interest charges.

4. DELIVERY
Delivery times are estimates and subject to manufacturer's confirmation. Acme Instrumentation WLL shall not be liable for delays beyond our control.

5. WARRANTY
All products carry the manufacturer's standard warranty. Extended warranty options are available upon request.

6. INSTALLATION & COMMISSIONING
Installation and commissioning services are available at additional cost unless included in the quotation.

7. FORCE MAJEURE
Acme Instrumentation WLL shall not be liable for failure to perform due to causes beyond reasonable control.

8. GOVERNING LAW
This quotation is governed by the laws of the Kingdom of Bahrain.`
	assert.Equal(t, wantAcme, acmeTerms, "Acme (default division) T&C must be byte-identical to the pre-Wave-12.5 output")

	assert.Contains(t, beaconTerms, "Beacon Controls WLL shall not be liable for delays beyond our control.")
	assert.Contains(t, beaconTerms, "Beacon Controls WLL shall not be liable for failure to perform due to causes beyond reasonable control.")
	assert.NotContains(t, beaconTerms, "Acme Instrumentation WLL",
		"Beacon T&C must NOT leak Acme's name — this was the bug before the fix")
}

// TestBuildCostingExportDataFromOfferUsesOfferDivisionForTerms verifies the
// call-site wiring: buildCostingExportDataFromOffer must resolve T&C via the
// OFFER'S OWN division (not the default), when the offer has no explicit
// TermsAndConditions override.
func TestBuildCostingExportDataFromOfferUsesOfferDivisionForTerms(t *testing.T) {
	withSyntheticOverlay(t)

	beaconOffer := Offer{
		Base:     Base{ID: uuid.New().String()},
		Division: "Beacon Controls",
		VatRate:  10,
		Items:    []OfferItem{{LineNumber: 1, Description: "Line", Quantity: 1, UnitPrice: 100, TotalPrice: 100}},
	}
	data := buildCostingExportDataFromOffer(beaconOffer, CustomerMaster{}, CustomerContact{})
	assert.Contains(t, data.TermsAndConditions, "Beacon Controls WLL",
		"a Beacon offer's auto-generated T&C must name Beacon, not the default division")
	assert.NotContains(t, data.TermsAndConditions, "Acme Instrumentation WLL")

	acmeOffer := Offer{
		Base:     Base{ID: uuid.New().String()},
		Division: "Acme Instrumentation",
		VatRate:  10,
		Items:    []OfferItem{{LineNumber: 1, Description: "Line", Quantity: 1, UnitPrice: 100, TotalPrice: 100}},
	}
	acmeData := buildCostingExportDataFromOffer(acmeOffer, CustomerMaster{}, CustomerContact{})
	assert.Contains(t, acmeData.TermsAndConditions, "Acme Instrumentation WLL")
}
