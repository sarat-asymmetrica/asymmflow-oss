package main

// Mission I Band-3 follow-through: the startup "wrong-date invoice" audit
// must flag FUTURE-DATED invoices relative to today, not a hardcoded year
// cutoff. With FY2026 live, the old `>= '2026'` rule flagged every valid
// current-year invoice forever (surfaced by the Band-3 review pack).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountFutureDatedInvoices_FlagsFutureNotCurrentYear(t *testing.T) {
	app := setupPaymentTestApp(t)

	// A valid invoice dated earlier this year (or last month) — the old
	// year-cutoff rule would have flagged this forever once 2026 went live.
	past := makeMissionIInvoice(t, app, "INV-AUDIT-PAST", "Sent",
		500.000, 500.000, time.Now().AddDate(0, 1, 0))
	require.NoError(t, app.db.Model(&Invoice{}).Where("id = ?", past.ID).
		Update("invoice_date", time.Now().AddDate(0, -1, 0)).Error)

	assert.EqualValues(t, 0, app.countFutureDatedInvoices(),
		"a current-FY invoice dated in the past must not be flagged")

	// A genuinely future-dated invoice must be flagged.
	future := makeMissionIInvoice(t, app, "INV-AUDIT-FUTURE", "Sent",
		500.000, 500.000, time.Now().AddDate(0, 2, 0))
	require.NoError(t, app.db.Model(&Invoice{}).Where("id = ?", future.ID).
		Update("invoice_date", time.Now().AddDate(0, 1, 0)).Error)

	assert.EqualValues(t, 1, app.countFutureDatedInvoices(),
		"an invoice dated after today must be flagged")

	// Today itself is not "future".
	require.NoError(t, app.db.Model(&Invoice{}).Where("id = ?", future.ID).
		Update("invoice_date", time.Now()).Error)
	assert.EqualValues(t, 0, app.countFutureDatedInvoices(),
		"an invoice dated today must not be flagged")
}
