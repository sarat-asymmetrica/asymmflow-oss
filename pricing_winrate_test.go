package main

// Owner ruling G1.4: per-customer win-rate is a REAL aggregation over decided
// offers, replacing the legacy screen's hardcoded sidebar list. These tests pin
// the aggregation math, the decided-only filter, the name-keyed fallback, and
// the revenue/order.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func seedWinRateOffer(t *testing.T, app *App, number, custID, custName, stage string, value float64) {
	t.Helper()
	require.NoError(t, app.db.Create(&Offer{
		OfferNumber:   number,
		CustomerID:    custID,
		CustomerName:  custName,
		Stage:         stage,
		TotalValueBHD: value,
	}).Error)
}

func TestGetCustomerWinRates_Aggregates(t *testing.T) {
	app := setupTestApp(t)

	// Customer A: 3 Won + 1 Lost → 0.75, won revenue 60.000.
	seedWinRateOffer(t, app, "OF-A1", "cust-a", "Alpha Controls", "Won", 10.000)
	seedWinRateOffer(t, app, "OF-A2", "cust-a", "Alpha Controls", "Won", 20.000)
	seedWinRateOffer(t, app, "OF-A3", "cust-a", "Alpha Controls", "Won", 30.000)
	seedWinRateOffer(t, app, "OF-A4", "cust-a", "Alpha Controls", "Lost", 99.000)
	// Customer B: 1 Won + 3 Lost → 0.25, won revenue 5.000.
	seedWinRateOffer(t, app, "OF-B1", "cust-b", "Beta Instruments", "Won", 5.000)
	seedWinRateOffer(t, app, "OF-B2", "cust-b", "Beta Instruments", "Lost", 1.000)
	seedWinRateOffer(t, app, "OF-B3", "cust-b", "Beta Instruments", "Lost", 1.000)
	seedWinRateOffer(t, app, "OF-B4", "cust-b", "Beta Instruments", "Lost", 1.000)
	// In-flight offers are NOT decided — must be excluded from the ratio.
	seedWinRateOffer(t, app, "OF-A5", "cust-a", "Alpha Controls", "Quoted", 500.000)
	seedWinRateOffer(t, app, "OF-A6", "cust-a", "Alpha Controls", "RFQ", 500.000)
	seedWinRateOffer(t, app, "OF-A7", "cust-a", "Alpha Controls", "Expired", 500.000)

	rows, err := app.GetCustomerWinRates()
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byID := map[string]CustomerWinRate{}
	for _, r := range rows {
		byID[r.CustomerID] = r
	}

	a := byID["cust-a"]
	require.Equal(t, "Alpha Controls", a.CustomerName)
	require.Equal(t, 3, a.OffersWon)
	require.Equal(t, 1, a.OffersLost)
	require.Equal(t, 4, a.OffersTotal, "in-flight offers excluded")
	require.InDelta(t, 0.75, a.WinRate, 1e-9)
	require.InDelta(t, 60.000, a.WonValueBHD, 1e-9, "revenue = won offers only")

	b := byID["cust-b"]
	require.Equal(t, 1, b.OffersWon)
	require.Equal(t, 3, b.OffersLost)
	require.InDelta(t, 0.25, b.WinRate, 1e-9)
	require.InDelta(t, 5.000, b.WonValueBHD, 1e-9)

	// Ordered by won revenue descending (A's 60 before B's 5).
	require.Equal(t, "cust-a", rows[0].CustomerID)
}

func TestGetCustomerWinRates_NameKeyedFallback(t *testing.T) {
	app := setupTestApp(t)

	// Offers with NO customer id but a name still aggregate honestly under the
	// name, rather than collapsing into a single empty-id bucket.
	seedWinRateOffer(t, app, "OF-N1", "", "Gamma Trading", "Won", 12.000)
	seedWinRateOffer(t, app, "OF-N2", "", "Gamma Trading", "Lost", 3.000)
	// An offer with neither id nor name cannot be attributed — dropped, not crashed.
	seedWinRateOffer(t, app, "OF-N3", "", "", "Won", 100.000)

	rows, err := app.GetCustomerWinRates()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "Gamma Trading", rows[0].CustomerName)
	require.Equal(t, 2, rows[0].OffersTotal)
	require.InDelta(t, 0.5, rows[0].WinRate, 1e-9)
}

func TestGetCustomerWinRates_Empty(t *testing.T) {
	app := setupTestApp(t)
	rows, err := app.GetCustomerWinRates()
	require.NoError(t, err)
	require.Empty(t, rows)
}
