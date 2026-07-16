package main

// INTEG residue campaign — Wave R1.1 (costing hot-zone) persistence validation.
// SaveCostingAsOffer is the create-an-Offer-from-a-costing-sheet path. The
// frontend now assembles the FLAT main.CostingExportData (header + calcLine
// computed line items) that this binding takes. This drives the bound App
// method against a scratch SQLite: a costing export creates an Offer whose
// header totals (value / margin / discount / VAT) and line items are derived
// exactly as the Go impl specifies, and the offer + its items persist.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegCosting_SaveCostingAsOffer(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Offer{}, &OfferItem{}), "migrate offers")

	// A flat export payload matching what the VM's buildCostingExportData()
	// produces: two priced line items, healthy margin (30% ⇒ no margin alert),
	// a discount, and a folder number that seeds a deterministic offer number.
	data := CostingExportData{
		Division:      "Acme Instrumentation",
		Date:          "2026-07-15",
		PreparedBy:    "A. Yusuf",
		CustomerName:  "Gulf Fabrication W.L.L.",
		FolderNumber:  "TESTCS0001",
		QuoteType:     "Quotation",
		VatRate:       10,
		Subtotal:      1100.000,
		Discount:      100.000,
		NetAmount:     1000.000,
		VAT:           100.000,
		GrandTotal:    1000.000,
		TotalCost:     700.000,
		Profit:        300.000,
		ProfitPercent: 30.0,
		OpportunityId: 0, // no linked RFQ — exercises the pure create path
		LineItems: []CostingExportLineItem{
			{
				SlNo:           1,
				Equipment:      "Coriolis Flow Meter",
				Model:          "CFM-2200",
				Currency:       "EUR",
				Quantity:       2,
				MarginPercent:  25,
				SuggestedPrice: 300.000, // per-unit price used
				TotalPrice:     600.000, // line total = 300 * 2
				TotalCost:      210.000,
				ExchangeRate:   0.45,
			},
			{
				SlNo:           2,
				Equipment:      "Pressure Transmitter",
				Model:          "PT-100",
				Currency:       "BHD",
				Quantity:       1,
				MarginPercent:  20,
				SuggestedPrice: 400.000,
				TotalPrice:     400.000,
				TotalCost:      280.000,
				ExchangeRate:   1.0,
			},
		},
	}

	offer, err := app.SaveCostingAsOffer(data)
	require.NoError(t, err, "a well-formed costing export must create an offer")
	require.NotNil(t, offer)

	// --- Header totals derive exactly as the Go impl specifies. ---
	require.InDelta(t, 1000.000, offer.TotalValueBHD, 1e-6, "TotalValueBHD == GrandTotal")
	require.InDelta(t, 30.0, offer.EstimatedMargin, 1e-6, "margin == Profit/GrandTotal*100")
	require.InDelta(t, (100.0/1100.0)*100.0, offer.DiscountPercent, 1e-6, "discount%% == Discount/Subtotal*100")
	require.InDelta(t, 10.0, offer.VatRate, 1e-6, "VatRate carried through")
	require.Equal(t, "Quoted", offer.Stage, "a fresh offer is Quoted")
	require.Equal(t, "Quotation", offer.QuoteType)
	require.NotEmpty(t, offer.OfferNumber, "offer number derived from the folder number")

	// --- Line items build from the costing lines (SuggestedPrice/TotalPrice). ---
	require.Len(t, offer.Items, 2, "both priced lines become offer items")

	// --- The offer + its items persist to the DB. ---
	var stored Offer
	require.NoError(t, app.db.Preload("Items").Where("id = ?", offer.ID).First(&stored).Error)
	require.Len(t, stored.Items, 2, "offer items persist")
	require.InDelta(t, 1000.000, stored.TotalValueBHD, 1e-6)

	// --- Exactly one offer persisted. ---
	var count int64
	require.NoError(t, app.db.Model(&Offer{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
