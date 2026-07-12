package main

// FX revaluation golden tests (Wave 5 A.1c). Written and committed green
// against the UNTOUCHED root implementation BEFORE the peel to
// pkg/finance/fx — revaluation is financial arithmetic (invariant 5), so
// the numbers are pinned first and the peel must reproduce them exactly.
//
// Fixture values are chosen to be exact in binary floating point
// (1024.0 balance; rates 0.375, 0.4375, 0.5) so every expected number is
// an EXACT float64 equality, not a tolerance.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupFXApp(t *testing.T) *App {
	t.Helper()
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CompanyBankAccount{}, &BankStatement{}, &FXRate{}, &FXRevaluation{}))
	return app
}

func seedFXAccount(t *testing.T, app *App, currency string, bookingRate, closingBalance float64) CompanyBankAccount {
	t.Helper()
	account := CompanyBankAccount{
		AccountNumber: "ACC-" + currency,
		Currency:      currency,
		BookingRate:   bookingRate,
	}
	require.NoError(t, app.db.Create(&account).Error)
	statement := BankStatement{
		BankAccountID:  account.ID,
		PeriodEnd:      time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		ClosingBalance: closingBalance,
	}
	require.NoError(t, app.db.Create(&statement).Error)
	return account
}

func TestFXRevaluation_GoldenNumbers(t *testing.T) {
	app := setupFXApp(t)
	account := seedFXAccount(t, app, "USD", 0.375, 1024.0)

	jan := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)

	_, err := app.CreateFXRate("USD", "BHD", 0.4375, jan, "TEST")
	require.NoError(t, err)

	// First revaluation: baseline is the account's booking rate.
	first, err := app.CalculateFXRevaluation(account.ID, jan)
	require.NoError(t, err)
	require.Equal(t, 1024.0, first.ForeignBalance)
	require.Equal(t, 0.375, first.PreviousRate)
	require.Equal(t, 384.0, first.PreviousBHD) // 1024 × 0.375
	require.Equal(t, 0.4375, first.CurrentRate)
	require.Equal(t, 448.0, first.CurrentBHD) // 1024 × 0.4375
	require.Equal(t, 64.0, first.GainLossBHD) // 448 − 384
	require.False(t, first.IsPosted)

	// Second revaluation: baseline is the previous revaluation.
	_, err = app.CreateFXRate("USD", "BHD", 0.5, feb, "TEST")
	require.NoError(t, err)
	second, err := app.CalculateFXRevaluation(account.ID, feb)
	require.NoError(t, err)
	require.Equal(t, 0.4375, second.PreviousRate)
	require.Equal(t, 448.0, second.PreviousBHD)
	require.Equal(t, 0.5, second.CurrentRate)
	require.Equal(t, 512.0, second.CurrentBHD) // 1024 × 0.5
	require.Equal(t, 64.0, second.GainLossBHD) // 512 − 448

	// Unposted total folds both revaluations.
	unposted, err := app.GetUnpostedRevaluations()
	require.NoError(t, err)
	require.Equal(t, 128.0, unposted.Total)

	// Posting then reversing produces an exact negation on the same balance.
	require.NoError(t, app.PostFXRevaluation(second.ID, "test-admin"))
	require.NoError(t, app.ReverseRevaluation(second.ID, "test-admin", "golden test"))

	revals, err := app.GetFXRevaluations(account.ID)
	require.NoError(t, err)
	require.Len(t, revals, 3)
	reversal := revals[0] // newest first
	require.Equal(t, -64.0, reversal.GainLossBHD)
	require.Equal(t, 0.5, reversal.PreviousRate)
	require.Equal(t, 512.0, reversal.PreviousBHD)
	require.Equal(t, 0.4375, reversal.CurrentRate)
	require.Equal(t, 448.0, reversal.CurrentBHD)
	require.True(t, reversal.IsPosted)
}

func TestFXRevaluation_NoBookingRateBaselinesToZeroGain(t *testing.T) {
	app := setupFXApp(t)
	account := seedFXAccount(t, app, "EUR", 0, 1024.0)

	jan := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	_, err := app.CreateFXRate("EUR", "BHD", 0.4375, jan, "TEST")
	require.NoError(t, err)

	reval, err := app.CalculateFXRevaluation(account.ID, jan)
	require.NoError(t, err)
	require.Equal(t, 0.4375, reval.PreviousRate) // falls back to current rate
	require.Equal(t, 448.0, reval.PreviousBHD)
	require.Equal(t, 448.0, reval.CurrentBHD)
	require.Equal(t, 0.0, reval.GainLossBHD) // first run must show zero, never a fabricated gain
}

func TestFXRevaluation_RefusesBHDAccounts(t *testing.T) {
	app := setupFXApp(t)
	account := seedFXAccount(t, app, "BHD", 1.0, 1024.0)

	_, err := app.CalculateFXRevaluation(account.ID, time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
}

func TestCreateFXRate_GuardsBounds(t *testing.T) {
	app := setupFXApp(t)
	jan := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)

	_, err := app.CreateFXRate("USD", "BHD", 0, jan, "TEST")
	require.Error(t, err, "zero rate must be refused")
	_, err = app.CreateFXRate("USD", "BHD", 2000, jan, "TEST")
	require.Error(t, err, "absurd rate must be refused")

	_, err = app.CreateFXRate("USD", "BHD", 0.4, jan, "TEST")
	require.NoError(t, err)
	// ±20% relative bound against the last known rate.
	_, err = app.CreateFXRate("USD", "BHD", 0.5, feb, "TEST")
	require.Error(t, err, "0.4 → 0.5 is +25%% and must be refused")
	_, err = app.CreateFXRate("USD", "BHD", 0.44, feb, "TEST")
	require.NoError(t, err, "0.4 → 0.44 is +10%% and must pass")

	// Same-date rate is an update, not a duplicate row.
	updated, err := app.CreateFXRate("USD", "BHD", 0.45, feb, "TEST2")
	require.NoError(t, err)
	history, err := app.GetFXRateHistory("USD", "BHD", jan, feb)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, 0.45, updated.Rate)
	require.Equal(t, "TEST2", updated.Source)
}
