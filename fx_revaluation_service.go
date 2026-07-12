// =============================================================================
// FX REVALUATION SERVICE
//
// MISSION: Manage foreign exchange rates and revaluation for multi-currency accounts
// FEATURES: FX rate management, unrealized gain/loss calculation, CBB rates import
//
// Wave 5 A.1: the rate + revaluation logic lives in pkg/finance/fx, pinned
// by fx_revaluation_golden_test.go BEFORE the move (invariant 5). These
// delegates keep the Wails binding surface and the RBAC guards.
// =============================================================================

package main

import (
	"fmt"
	"time"

	financefx "ph_holdings_app/pkg/finance/fx"
)

// FXExposureReport provides a summary of foreign currency exposure. The
// alias keeps the JSON contract and Wails binding shape at the root.
type FXExposureReport = financefx.ExposureReport

// FXExposureResult bundles per-currency exposure reports with the total
// BHD exposure (Wails 3-return marshaling workaround — see the type doc
// comment in pkg/finance/fx/fx.go).
type FXExposureResult = financefx.ExposureResult

// FXRevaluationBatchResult bundles a batch of FX revaluations with their
// aggregate gain/loss total. Shared shape for GetUnpostedRevaluations and
// RevalueAllForeignAccounts (Wails 3-return marshaling workaround).
type FXRevaluationBatchResult = financefx.RevaluationBatchResult

func (a *App) fxGuarded(permission string) error {
	if err := a.requirePermission(permission); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

// CreateFXRate adds a new exchange rate
func (a *App) CreateFXRate(fromCurrency, toCurrency string, rate float64, rateDate time.Time, source string) (*FXRate, error) {
	if err := a.fxGuarded("finance:create"); err != nil {
		return nil, err
	}
	return a.fxService().CreateRate(fromCurrency, toCurrency, rate, rateDate, source)
}

// GetFXRate retrieves the rate for a currency pair on a specific date
func (a *App) GetFXRate(fromCurrency, toCurrency string, date time.Time) (*FXRate, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().RateOn(fromCurrency, toCurrency, date)
}

// GetLatestFXRate retrieves the most recent rate for a currency pair
func (a *App) GetLatestFXRate(fromCurrency, toCurrency string) (*FXRate, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().LatestRate(fromCurrency, toCurrency)
}

// GetFXRateHistory retrieves rate history for a currency pair
func (a *App) GetFXRateHistory(fromCurrency, toCurrency string, startDate, endDate time.Time) ([]FXRate, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().RateHistory(fromCurrency, toCurrency, startDate, endDate)
}

// GetAllFXRates retrieves all rates for a base currency
func (a *App) GetAllFXRates(baseCurrency string, date time.Time) ([]FXRate, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().AllRates(baseCurrency, date)
}

// DeleteFXRate removes a rate entry
func (a *App) DeleteFXRate(rateID string) error {
	if err := a.fxGuarded("finance:create"); err != nil {
		return err
	}
	return a.fxService().DeleteRate(rateID)
}

// CalculateFXRevaluation computes unrealized gain/loss for a foreign currency account
func (a *App) CalculateFXRevaluation(bankAccountID string, revaluationDate time.Time) (*FXRevaluation, error) {
	if err := a.fxGuarded("finance:create"); err != nil {
		return nil, err
	}
	return a.fxService().Revalue(bankAccountID, revaluationDate)
}

// GetFXRevaluations retrieves revaluation history for an account
func (a *App) GetFXRevaluations(bankAccountID string) ([]FXRevaluation, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().Revaluations(bankAccountID)
}

// GetUnpostedRevaluations retrieves all unposted revaluations
func (a *App) GetUnpostedRevaluations() (*FXRevaluationBatchResult, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().Unposted()
}

// PostFXRevaluation posts a revaluation to the general ledger.
// Wave 9.3 B2: identity server-resolved (Article III.4) — the client-supplied
// user is ignored; the parameter stays for Wails binding stability.
func (a *App) PostFXRevaluation(revaluationID, user string) error {
	if err := a.fxGuarded("finance:create"); err != nil {
		return err
	}
	return a.fxService().Post(revaluationID, a.getCurrentUserID())
}

// ReverseRevaluation reverses a posted revaluation
func (a *App) ReverseRevaluation(revaluationID, user, reason string) error {
	if err := a.fxGuarded("finance:create"); err != nil {
		return err
	}
	return a.fxService().Reverse(revaluationID, user, reason)
}

// RevalueAllForeignAccounts performs revaluation for all foreign currency accounts
func (a *App) RevalueAllForeignAccounts(revaluationDate time.Time) (*FXRevaluationBatchResult, error) {
	if err := a.fxGuarded("finance:create"); err != nil {
		return nil, err
	}
	return a.fxService().RevalueAll(revaluationDate)
}

// GetFXExposureReport generates a foreign currency exposure report
func (a *App) GetFXExposureReport() (*FXExposureResult, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().Exposure()
}

// GetFXGainLossSummary provides YTD gain/loss summary
func (a *App) GetFXGainLossSummary(year int) (map[string]any, error) {
	if err := a.fxGuarded("finance:view"); err != nil {
		return nil, err
	}
	return a.fxService().GainLossSummary(year)
}
