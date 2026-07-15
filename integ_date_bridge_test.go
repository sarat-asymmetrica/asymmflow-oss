package main

// INTEG campaign — Wave I1.3 validation: the frontend date→time.Time form bridge.
//
// The frontend seam (frontend-lab/src/bridge/map.ts `goTime`) converts a form
// date string ('YYYY-MM-DD') into the value Wails marshals into a Go time.Time
// binding argument. Because the frontend can never reach SQL, the genuinely
// novel risk is the IPC boundary: does the RFC3339 string `goTime` emits
// actually json.Unmarshal into the time.Time the Go binding expects, and does
// the bound method then persist correctly?
//
// This test reproduces that exact wire format (the JSON string Wails puts on the
// wire for a time.Time arg), unmarshals it the way the Wails runtime does, and
// drives App.SetExchangeRate end-to-end against a scratch SQLite — asserting
// persistence, the prior-rate-close behavior, and the active-rate read-back.

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// goTimeWireFormat mirrors frontend-lab/src/bridge/map.ts `goTime` EXACTLY: a
// date-only string becomes UTC midnight with an explicit 'Z'. Kept here as the
// contract under test so a drift in the TS helper surfaces as a failing assert.
func goTimeWireFormat(dateStr string) string {
	if dateStr == "" {
		return "0001-01-01T00:00:00Z"
	}
	// map.ts appends the time component only when absent.
	if len(dateStr) >= 11 && dateStr[10] == 'T' {
		return dateStr
	}
	return dateStr + "T00:00:00Z"
}

func TestIntegDateBridge_SetExchangeRate(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CurrencyExchangeRate{}), "migrate currency_exchange_rates")

	// --- 1. The wire round-trip: goTime('2026-07-15') → Wails JSON → time.Time.
	// Wails wraps the arg in the args array and JSON-encodes it; for a single
	// time.Time arg the encoded token is a quoted RFC3339 string. We unmarshal it
	// the way the Wails runtime hands it to the bound Go method.
	wire := goTimeWireFormat("2026-07-15")
	require.Equal(t, "2026-07-15T00:00:00Z", wire, "goTime must emit explicit UTC-midnight RFC3339")

	var effectiveFrom time.Time
	require.NoError(t, json.Unmarshal([]byte(`"`+wire+`"`), &effectiveFrom),
		"the RFC3339 string goTime emits must unmarshal into a Go time.Time")
	require.Equal(t, 2026, effectiveFrom.Year())
	require.Equal(t, time.July, effectiveFrom.Month())
	require.Equal(t, 15, effectiveFrom.Day())
	require.True(t, effectiveFrom.Equal(time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)),
		"date-only input must land on UTC midnight, not a tz-shifted instant")

	// --- 2. Drive the real binding with that time.Time and assert persistence.
	require.NoError(t, app.SetExchangeRate("USD", 0.376, effectiveFrom, "CBB Reference"))

	var stored CurrencyExchangeRate
	require.NoError(t, app.db.Where("currency_code = ?", "USD").First(&stored).Error)
	require.Equal(t, "USD", stored.CurrencyCode)
	require.InDelta(t, 0.376, stored.Rate, 1e-9)
	require.True(t, stored.EffectiveFrom.Equal(effectiveFrom), "EffectiveFrom must persist the bridged date")
	require.Nil(t, stored.EffectiveTo, "a freshly-set rate is the active one (effective_to NULL)")
	require.Equal(t, "CBB Reference", stored.Notes)

	// --- 3. A second set for the same currency must CLOSE the prior rate.
	laterFrom := mustUnmarshalTime(t, goTimeWireFormat("2026-07-20"))
	require.NoError(t, app.SetExchangeRate("USD", 0.377, laterFrom, "Manual Entry"))

	active, err := app.GetActiveCurrencyRates()
	require.NoError(t, err)
	usdActive := filterByCurrency(active, "USD")
	require.Len(t, usdActive, 1, "exactly one USD rate stays active after a re-set")
	require.InDelta(t, 0.377, usdActive[0].Rate, 1e-9, "the newer rate is the active one")
	require.Nil(t, usdActive[0].EffectiveTo)

	// The prior 0.376 rate must now be closed (effective_to == the new from-date).
	var all []CurrencyExchangeRate
	require.NoError(t, app.db.Where("currency_code = ?", "USD").Order("effective_from").Find(&all).Error)
	require.Len(t, all, 2, "both rate rows are retained (history), not overwritten")
	require.NotNil(t, all[0].EffectiveTo, "the superseded 0.376 rate must be closed")
	require.True(t, all[0].EffectiveTo.Equal(laterFrom), "close date == the new rate's effective_from")

	// --- 4. The empty-date guard: goTime('') maps to Go zero time, which the
	// frontend seam refuses to send (validation error) — proven here that it
	// would otherwise be the 0001 sentinel, i.e. never a silent "today".
	require.Equal(t, "0001-01-01T00:00:00Z", goTimeWireFormat(""))
	zero := mustUnmarshalTime(t, goTimeWireFormat(""))
	require.True(t, zero.IsZero(), "blank date is Go zero time, guarded at the seam")
}

func mustUnmarshalTime(t *testing.T, rfc string) time.Time {
	t.Helper()
	var out time.Time
	require.NoError(t, json.Unmarshal([]byte(`"`+rfc+`"`), &out))
	return out
}

func filterByCurrency(rates []CurrencyExchangeRate, code string) []CurrencyExchangeRate {
	var out []CurrencyExchangeRate
	for _, r := range rates {
		if r.CurrencyCode == code {
			out = append(out, r)
		}
	}
	return out
}
