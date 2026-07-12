package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SUPPLIER PAYMENT FX PERSISTENCE — Wave 9.3 C1 (authorized posting change)
// =============================================================================
// RecordSupplierPayment used to hardcode exchangeRate := 1.0, so a non-BHD
// payment posted amount_bhd at a 1:1 rate and the overpay guard evaluated
// against that same wrong figure. It now takes the confirmed exchangeRate and
// posts amount_bhd = amount * exchangeRate in one write. These tests pin: a
// non-BHD payment posts the real converted amount, the overpay guard respects
// the real rate (not 1:1), and an omitted/zero rate still falls back to BHD
// 1:1 (unchanged behavior for existing callers).
// =============================================================================

func TestRecordSupplierPayment_NonBHDPostsConvertedAmount(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-FX-1", "Approved", "Matched", 1000.000)

	pay, err := app.RecordSupplierPayment(inv.ID, 100.000, "USD", "Bank Transfer", "2026-01-15", "FX-1", 0.376)
	require.NoError(t, err)
	require.NotNil(t, pay)

	assert.Equal(t, 100.000, pay.AmountForeign)
	assert.Equal(t, "USD", pay.Currency)
	assert.Equal(t, 0.376, pay.ExchangeRate)
	assert.Equal(t, 37.600, pay.AmountBHD, "amount_bhd must be amount * exchangeRate, not a 1:1 posting")

	var reloaded SupplierPayment
	require.NoError(t, app.db.First(&reloaded, "id = ?", pay.ID).Error)
	assert.Equal(t, 37.600, reloaded.AmountBHD)
}

func TestRecordSupplierPayment_OverpayGuardRespectsRate(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-FX-2", "Approved", "Matched", 100.000)

	// 250 USD at 0.376 posts 94.000 BHD — within the 100.000 BHD outstanding,
	// so it must be accepted even though 250 units is far more than 100 at a
	// naive 1:1 read.
	pay, err := app.RecordSupplierPayment(inv.ID, 250.000, "USD", "Bank Transfer", "2026-01-15", "FX-2A", 0.376)
	require.NoError(t, err)
	require.NotNil(t, pay)
	assert.Equal(t, 94.000, pay.AmountBHD)

	// Remaining outstanding is 6.000 BHD. 20 USD at 0.376 posts 7.520 BHD,
	// which exceeds it — the guard must reject using the real rate.
	_, err = app.RecordSupplierPayment(inv.ID, 20.000, "USD", "Bank Transfer", "2026-01-16", "FX-2B", 0.376)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds outstanding balance")
}

func TestRecordSupplierPayment_OmittedRateFallsBackToBHDOneToOne(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-FX-3", "Approved", "Matched", 500.000)

	// Zero/omitted exchangeRate must preserve the historical BHD 1:1 behavior.
	pay, err := app.RecordSupplierPayment(inv.ID, 50.000, "BHD", "Bank Transfer", "2026-01-15", "FX-3", 0)
	require.NoError(t, err)
	require.NotNil(t, pay)

	assert.Equal(t, 1.0, pay.ExchangeRate)
	assert.Equal(t, 50.000, pay.AmountBHD)
}
