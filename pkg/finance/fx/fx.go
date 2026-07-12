// Package fx owns foreign-exchange rate management and multi-currency
// revaluation: bounded rate entry, unrealized gain/loss calculation,
// posting/reversal, and exposure reporting.
//
// Wave 5 A.1: a W4-D1 peel — the logic moved inward from the root
// fx_revaluation_service.go, pinned first by root-level golden tests
// (fx_revaluation_golden_test.go) because revaluation is financial
// arithmetic (invariant 5). The models already live in pkg/finance, so
// the service needs only the database; RBAC guards stay with the host's
// thin delegates.
package fx

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/finance"
)

// Service is the FX rate + revaluation service.
type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

// CreateRate adds (or same-date-updates) an exchange rate, enforcing
// absolute bounds and a ±20% relative bound against the last known rate.
func (s *Service) CreateRate(fromCurrency, toCurrency string, rate float64, rateDate time.Time, source string) (*finance.FXRate, error) {
	// Absolute bounds: realistic FX rate range regardless of history
	const minFXRate = 0.001  // e.g. very weak currencies vs BHD
	const maxFXRate = 1000.0 // hard ceiling to catch typos / injected values
	if rate <= 0 {
		return nil, fmt.Errorf("rate must be positive")
	}
	if rate < minFXRate || rate > maxFXRate {
		return nil, fmt.Errorf("rate %.6f is outside absolute bounds [%.3f, %.1f]", rate, minFXRate, maxFXRate)
	}

	// Relative bounds: rate must be within ±20% of last known rate (prevents large sudden jumps)
	var lastRate finance.FXRate
	lastErr := s.db.Where("from_currency = ? AND to_currency = ?", fromCurrency, toCurrency).
		Order("rate_date DESC").First(&lastRate).Error
	if lastErr == nil && lastRate.Rate > 0 {
		ratio := rate / lastRate.Rate
		if ratio < 0.80 || ratio > 1.20 {
			return nil, fmt.Errorf("rate %.6f is outside ±20%% of last known rate %.6f (bounds: %.6f - %.6f)",
				rate, lastRate.Rate, lastRate.Rate*0.80, lastRate.Rate*1.20)
		}
	}

	// Check for existing rate on same date
	var existing finance.FXRate
	err := s.db.Where("from_currency = ? AND to_currency = ? AND rate_date = ?",
		fromCurrency, toCurrency, rateDate).First(&existing).Error
	if err == nil {
		existing.Rate = rate
		existing.Source = source
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, fmt.Errorf("failed to update rate: %w", err)
		}
		log.Printf("📊 FX rate updated: %s/%s = %.6f on %s", fromCurrency, toCurrency, rate, rateDate.Format("2006-01-02"))
		return &existing, nil
	}

	fxRate := &finance.FXRate{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		RateDate:     rateDate,
		Rate:         rate,
		Source:       source,
	}

	if err := s.db.Create(fxRate).Error; err != nil {
		return nil, fmt.Errorf("failed to create rate: %w", err)
	}

	log.Printf("📊 FX rate created: %s/%s = %.6f on %s", fromCurrency, toCurrency, rate, rateDate.Format("2006-01-02"))
	return fxRate, nil
}

// RateOn retrieves the rate for a pair on the given date, or the most
// recent prior date.
func (s *Service) RateOn(fromCurrency, toCurrency string, date time.Time) (*finance.FXRate, error) {
	var rate finance.FXRate
	err := s.db.Where("from_currency = ? AND to_currency = ? AND rate_date <= ?",
		fromCurrency, toCurrency, date).
		Order("rate_date DESC").
		First(&rate).Error
	if err != nil {
		return nil, fmt.Errorf("no rate found for %s/%s on or before %s", fromCurrency, toCurrency, date.Format("2006-01-02"))
	}
	return &rate, nil
}

// LatestRate retrieves the most recent rate for a currency pair.
func (s *Service) LatestRate(fromCurrency, toCurrency string) (*finance.FXRate, error) {
	var rate finance.FXRate
	err := s.db.Where("from_currency = ? AND to_currency = ?", fromCurrency, toCurrency).
		Order("rate_date DESC").
		First(&rate).Error
	if err != nil {
		return nil, fmt.Errorf("no rate found for %s/%s", fromCurrency, toCurrency)
	}
	return &rate, nil
}

// RateHistory retrieves rate history for a currency pair.
func (s *Service) RateHistory(fromCurrency, toCurrency string, startDate, endDate time.Time) ([]finance.FXRate, error) {
	var rates []finance.FXRate
	err := s.db.Where("from_currency = ? AND to_currency = ? AND rate_date BETWEEN ? AND ?",
		fromCurrency, toCurrency, startDate, endDate).
		Order("rate_date ASC").
		Find(&rates).Error
	return rates, err
}

// AllRates retrieves the latest rates for each pair involving a base currency.
func (s *Service) AllRates(baseCurrency string, date time.Time) ([]finance.FXRate, error) {
	var rates []finance.FXRate
	subquery := s.db.Model(&finance.FXRate{}).
		Select("from_currency, to_currency, MAX(rate_date) as max_date").
		Where("(from_currency = ? OR to_currency = ?) AND rate_date <= ?", baseCurrency, baseCurrency, date).
		Group("from_currency, to_currency")

	err := s.db.Where("(from_currency, to_currency, rate_date) IN (?)", subquery).
		Find(&rates).Error
	return rates, err
}

// DeleteRate removes a rate entry.
func (s *Service) DeleteRate(rateID string) error {
	return s.db.Delete(&finance.FXRate{}, "id = ?", rateID).Error
}

// Revalue computes unrealized gain/loss for a foreign currency account.
func (s *Service) Revalue(bankAccountID string, revaluationDate time.Time) (*finance.FXRevaluation, error) {
	// Wrap the entire calculation in a transaction so the account's BookingRate cannot
	// be modified between the account read and the revaluation insert.
	var reval *finance.FXRevaluation
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var account finance.CompanyBankAccount
		if err := tx.First(&account, "id = ?", bankAccountID).Error; err != nil {
			return fmt.Errorf("bank account not found: %w", err)
		}

		if account.Currency == "BHD" {
			return fmt.Errorf("BHD accounts do not require FX revaluation")
		}

		// Get current balance from latest statement
		var latestStatement finance.BankStatement
		if err := tx.Where("bank_account_id = ?", bankAccountID).
			Order("period_end DESC").
			First(&latestStatement).Error; err != nil {
			return fmt.Errorf("no bank statement found for account")
		}

		foreignBalance := latestStatement.ClosingBalance

		// Get previous revaluation (if any)
		var prevReval finance.FXRevaluation
		var previousRate float64 = 0.0
		var previousBHD float64 = 0.0

		if err := tx.Where("bank_account_id = ?", bankAccountID).
			Order("revaluation_date DESC").
			First(&prevReval).Error; err == nil {
			previousRate = prevReval.CurrentRate
			previousBHD = prevReval.CurrentBHD
		} else {
			// First revaluation: use the account's booking rate (set when account was opened).
			// Assuming 1.0 produced wildly incorrect gain/loss figures for foreign accounts.
			if account.BookingRate > 0 {
				previousRate = account.BookingRate
				previousBHD = foreignBalance * account.BookingRate
			} else {
				// No prior revaluation and booking rate not recorded.
				// Fall back to current rate as baseline so gain/loss = 0 on first run (safe).
				// Users should set BookingRate on the account for accurate historical comparison.
				log.Printf("⚠️ WARNING: No booking_rate set for %s account %s — using current rate as baseline (first revaluation will show zero gain/loss)",
					account.Currency, account.AccountNumber)
				currentFXRateForBaseline, baseErr := s.RateOn(account.Currency, "BHD", revaluationDate)
				if baseErr != nil {
					return fmt.Errorf("cannot revalue %s account %s: no prior revaluation, no booking rate, and no current FX rate: %w",
						account.Currency, account.AccountNumber, baseErr)
				}
				previousRate = currentFXRateForBaseline.Rate
				previousBHD = foreignBalance * previousRate
			}
		}

		// Get current FX rate
		currentFXRate, err := s.RateOn(account.Currency, "BHD", revaluationDate)
		if err != nil {
			return fmt.Errorf("no FX rate available for %s/BHD: %w", account.Currency, err)
		}

		// Calculate current BHD value and gain/loss
		currentBHD := foreignBalance * currentFXRate.Rate
		gainLoss := currentBHD - previousBHD

		r := &finance.FXRevaluation{
			BankAccountID:   bankAccountID,
			RevaluationDate: revaluationDate,
			ForeignCurrency: account.Currency,
			ForeignBalance:  foreignBalance,
			PreviousRate:    previousRate,
			PreviousBHD:     previousBHD,
			CurrentRate:     currentFXRate.Rate,
			CurrentBHD:      currentBHD,
			GainLossBHD:     gainLoss,
			IsPosted:        false,
		}

		if err := tx.Create(r).Error; err != nil {
			return fmt.Errorf("failed to create revaluation: %w", err)
		}

		gainLossType := "GAIN"
		if gainLoss < 0 {
			gainLossType = "LOSS"
		}
		log.Printf("💱 FX revaluation: %s %.3f %s → %.3f BHD (%s: %.3f BHD)",
			account.AccountNumber, foreignBalance, account.Currency, currentBHD, gainLossType, gainLoss)

		reval = r
		return nil
	})

	return reval, txErr
}

// Revaluations retrieves revaluation history for an account.
func (s *Service) Revaluations(bankAccountID string) ([]finance.FXRevaluation, error) {
	var revaluations []finance.FXRevaluation
	err := s.db.Where("bank_account_id = ?", bankAccountID).
		Order("revaluation_date DESC").
		Find(&revaluations).Error
	return revaluations, err
}

// RevaluationBatchResult bundles a set of FX revaluations with their
// aggregate gain/loss total. Used by both Unposted() and RevalueAll().
// Wails v2's bound-method marshaling only handles OutputCount 1 or 2 (see
// internal/binding/boundMethod.go) — a 3-value Go return silently marshals
// to null on the JS side. Bundling into a struct + error keeps the binding
// a clean 2-value return.
type RevaluationBatchResult struct {
	Revaluations []finance.FXRevaluation `json:"revaluations"`
	Total        float64                 `json:"total"`
}

// Unposted retrieves all unposted revaluations and their total gain/loss.
func (s *Service) Unposted() (*RevaluationBatchResult, error) {
	var revaluations []finance.FXRevaluation
	err := s.db.Where("is_posted = ?", false).
		Order("revaluation_date ASC").
		Find(&revaluations).Error
	if err != nil {
		return nil, err
	}

	var totalGainLoss float64
	for _, r := range revaluations {
		totalGainLoss += r.GainLossBHD
	}
	return &RevaluationBatchResult{Revaluations: revaluations, Total: totalGainLoss}, nil
}

// Post posts a revaluation to the general ledger.
func (s *Service) Post(revaluationID, user string) error {
	var reval finance.FXRevaluation
	if err := s.db.First(&reval, "id = ?", revaluationID).Error; err != nil {
		return fmt.Errorf("revaluation not found: %w", err)
	}

	if reval.IsPosted {
		return fmt.Errorf("revaluation already posted")
	}

	// In a full system, this would create a journal entry
	// For now, we just mark it as posted
	now := time.Now()
	result := s.db.Model(&reval).Updates(map[string]any{
		"is_posted": true,
		"posted_by": user,
		"posted_at": now,
	})
	if result.Error != nil {
		return result.Error
	}

	log.Printf("📝 FX revaluation posted: %.3f BHD gain/loss by %s", reval.GainLossBHD, user)
	return nil
}

// Reverse reverses a posted revaluation (deletes an unposted one).
func (s *Service) Reverse(revaluationID, user, reason string) error {
	var reval finance.FXRevaluation
	if err := s.db.First(&reval, "id = ?", revaluationID).Error; err != nil {
		return fmt.Errorf("revaluation not found: %w", err)
	}

	if !reval.IsPosted {
		// Just delete if not posted
		return s.db.Delete(&reval).Error
	}

	// Create reversing entry
	reversing := &finance.FXRevaluation{
		BankAccountID:   reval.BankAccountID,
		RevaluationDate: time.Now(),
		ForeignCurrency: reval.ForeignCurrency,
		ForeignBalance:  reval.ForeignBalance,
		PreviousRate:    reval.CurrentRate,
		PreviousBHD:     reval.CurrentBHD,
		CurrentRate:     reval.PreviousRate,
		CurrentBHD:      reval.PreviousBHD,
		GainLossBHD:     -reval.GainLossBHD, // Reverse the gain/loss
		IsPosted:        true,
		PostedBy:        user,
	}
	now := time.Now()
	reversing.PostedAt = &now

	if err := s.db.Create(reversing).Error; err != nil {
		return fmt.Errorf("failed to create reversing entry: %w", err)
	}

	log.Printf("🔄 FX revaluation reversed: %.3f BHD by %s - %s", reval.GainLossBHD, user, reason)
	return nil
}

// RevalueAll performs revaluation for all foreign currency accounts.
func (s *Service) RevalueAll(revaluationDate time.Time) (*RevaluationBatchResult, error) {
	var accounts []finance.CompanyBankAccount
	if err := s.db.Where("currency != ?", "BHD").Find(&accounts).Error; err != nil {
		return nil, err
	}

	var revaluations []finance.FXRevaluation
	var totalGainLoss float64

	for _, account := range accounts {
		reval, err := s.Revalue(account.ID, revaluationDate)
		if err != nil {
			log.Printf("⚠️ Failed to revalue %s: %v", account.AccountNumber, err)
			continue
		}
		revaluations = append(revaluations, *reval)
		totalGainLoss += reval.GainLossBHD
	}

	gainLossType := "NET GAIN"
	if totalGainLoss < 0 {
		gainLossType = "NET LOSS"
	}
	log.Printf("💱 Batch revaluation complete: %d accounts, %s: %.3f BHD", len(revaluations), gainLossType, totalGainLoss)

	return &RevaluationBatchResult{Revaluations: revaluations, Total: totalGainLoss}, nil
}

// ExposureReport summarizes foreign currency exposure for one currency.
type ExposureReport struct {
	Currency        string  `json:"currency"`
	AccountCount    int     `json:"account_count"`
	TotalForeign    float64 `json:"total_foreign"`
	CurrentRate     float64 `json:"current_rate"`
	TotalBHD        float64 `json:"total_bhd"`
	UnrealizedGain  float64 `json:"unrealized_gain"`
	PercentExposure float64 `json:"percent_exposure"`
}

// ExposureResult bundles per-currency exposure reports with the total BHD
// exposure across all foreign currencies. Wails v2's bound-method
// marshaling only handles OutputCount 1 or 2 (see
// internal/binding/boundMethod.go) — a 3-value Go return silently marshals
// to null on the JS side. Bundling into a struct + error keeps the binding
// a clean 2-value return.
type ExposureResult struct {
	Reports []ExposureReport `json:"reports"`
	Total   float64          `json:"total"`
}

// Exposure generates a foreign currency exposure report.
func (s *Service) Exposure() (*ExposureResult, error) {
	type currencyGroup struct {
		Currency     string
		TotalBalance float64
		AccountCount int
	}

	var groups []currencyGroup
	s.db.Model(&finance.CompanyBankAccount{}).
		Select("currency, SUM(balance) as total_balance, COUNT(*) as account_count").
		Where("currency != ?", "BHD").
		Group("currency").
		Scan(&groups)

	var reports []ExposureReport
	var totalBHDExposure float64

	for _, g := range groups {
		rate, err := s.LatestRate(g.Currency, "BHD")
		rateValue := 0.0
		if err == nil {
			rateValue = rate.Rate
		}

		bhdValue := g.TotalBalance * rateValue

		// Get unrealized gain from latest revaluation
		var latestReval finance.FXRevaluation
		var unrealizedGain float64
		s.db.Where("foreign_currency = ? AND is_posted = ?", g.Currency, false).
			Order("revaluation_date DESC").
			First(&latestReval)
		if latestReval.ID != "" {
			unrealizedGain = latestReval.GainLossBHD
		}

		reports = append(reports, ExposureReport{
			Currency:       g.Currency,
			AccountCount:   g.AccountCount,
			TotalForeign:   g.TotalBalance,
			CurrentRate:    rateValue,
			TotalBHD:       bhdValue,
			UnrealizedGain: unrealizedGain,
		})

		totalBHDExposure += bhdValue
	}

	// Calculate percentages
	for i := range reports {
		if totalBHDExposure > 0 {
			reports[i].PercentExposure = (reports[i].TotalBHD / totalBHDExposure) * 100
		}
	}

	return &ExposureResult{Reports: reports, Total: totalBHDExposure}, nil
}

// GainLossSummary provides a YTD gain/loss summary for a year.
func (s *Service) GainLossSummary(year int) (map[string]any, error) {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	var revaluations []finance.FXRevaluation
	s.db.Where("revaluation_date BETWEEN ? AND ?", startDate, endDate).
		Order("revaluation_date ASC").
		Find(&revaluations)

	var totalGain, totalLoss, netGainLoss float64
	var postedGain, postedLoss float64

	for _, r := range revaluations {
		if r.GainLossBHD > 0 {
			totalGain += r.GainLossBHD
			if r.IsPosted {
				postedGain += r.GainLossBHD
			}
		} else {
			totalLoss += r.GainLossBHD
			if r.IsPosted {
				postedLoss += r.GainLossBHD
			}
		}
		netGainLoss += r.GainLossBHD
	}

	return map[string]any{
		"year":              year,
		"total_gain":        totalGain,
		"total_loss":        totalLoss,
		"net_gain_loss":     netGainLoss,
		"posted_gain":       postedGain,
		"posted_loss":       postedLoss,
		"posted_net":        postedGain + postedLoss,
		"unposted_net":      netGainLoss - (postedGain + postedLoss),
		"revaluation_count": len(revaluations),
	}, nil
}
