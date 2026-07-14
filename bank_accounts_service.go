// ═══════════════════════════════════════════════════════════════════════════
// COMPANY BANK ACCOUNTS SERVICE
//
// MISSION: Manage Acme Instrumentation and Beacon Controls bank account details for invoices and reconciliation
//          Provides the official bank accounts used across finance workflows
//
// USED BY: Invoice PDF generation (bank details section)
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	finance "ph_holdings_app/pkg/finance"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompanyBankAccount stores division-aware bank account details for invoices and reconciliation.
type CompanyBankAccount = finance.CompanyBankAccount

// SeedCompanyBankAccounts creates the official PH/AHS bank accounts.
// Called on app startup to ensure bank details are always available.
// SeedCompanyBankAccounts is the RBAC-guarded entry point (RBAC-003): the
// seed overwrites company IBANs, so only admin sessions may invoke it from
// the frontend. Startup and internal callers use the unexported variant.
func (a *App) SeedCompanyBankAccounts() error {
	if err := a.requirePermission("*"); err != nil {
		return err
	}
	return a.seedCompanyBankAccountsInternal()
}

// nonDefaultDivisionKey returns the Key of the first configured division that
// is NOT the overlay's default division (e.g. "Beacon Controls" for the
// synthetic pair). Used by seed rows that are deliberately scoped to "the
// other" division rather than the default, so the registry — not a frozen
// literal — decides which division that is. Falls back to the default
// division key itself if the overlay declares only one division.
func nonDefaultDivisionKey() string {
	def := activeOverlay.DefaultDivision()
	for _, d := range activeOverlay.Divisions {
		if d.Key != def {
			return d.Key
		}
	}
	return def
}

func (a *App) seedCompanyBankAccountsInternal() error {
	if a.db == nil {
		return nil
	}
	// PC-D18 (Mission H rehearsal finding): these rows are the DEMO bank
	// fixtures (synthetic IBANs). Every caller — startup migration, expense
	// foundation, empty-list fallback, the RBAC-guarded reseed — funnels
	// through here, and before this gate the fixtures were re-created (and
	// demo-id rows refreshed) on EVERY boot regardless of the overlay. On a
	// sovereign deployment the company's real accounts arrive via import;
	// injecting demo fixtures next to them puts demo bank details one
	// dropdown away from a real invoice. The seed-set gate the flag register
	// always claimed is now enforced at the seam.
	if !activeOverlay.SeedEnabled("demo-bank") {
		return nil
	}

	banks := []CompanyBankAccount{
		{
			ID:            "bank-alpha",
			Division:      activeOverlay.DefaultDivision(),
			BankName:      "Demo Bank A",
			AccountName:   "ACME INSTRUMENTATION WLL",
			AccountNumber: "10000000001",
			IBAN:          "BH29DMOA10000000000001",
			SwiftBIC:      "DMOABHBM",
			Currency:      "BHD",
			IsActive:      true,
			DisplayOrder:  1,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            "bank-beta",
			Division:      activeOverlay.DefaultDivision(),
			BankName:      "Demo Bank B",
			AccountName:   "ACME INSTRUMENTATION WLL",
			AccountNumber: "10000000002",
			IBAN:          "BH29DMOB10000000000002",
			SwiftBIC:      "DMOBBHBM",
			Currency:      "BHD",
			IsActive:      true,
			DisplayOrder:  2,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            "bank-gamma",
			Division:      activeOverlay.DefaultDivision(),
			BankName:      "Demo Bank C",
			AccountName:   "ACME INSTRUMENTATION WLL",
			AccountNumber: "10000000003",
			IBAN:          "BH29DMOC10000000000003",
			SwiftBIC:      "DMOCBHBM",
			Currency:      "BHD",
			IsActive:      true,
			DisplayOrder:  3,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            "bank-delta",
			Division:      activeOverlay.DefaultDivision(),
			BankName:      "Demo Bank D",
			AccountName:   "ACME INSTRUMENTATION WLL",
			AccountNumber: "10000000004",
			IBAN:          "BH29DMOD10000000000004",
			SwiftBIC:      "DMODBHBM",
			Currency:      "BHD",
			IsActive:      true,
			DisplayOrder:  4,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            "bank-ahs-gamma",
			Division:      nonDefaultDivisionKey(),
			BankName:      "Demo Bank C",
			AccountName:   "BEACON CONTROLS W.L.L.",
			AccountNumber: "20000000001",
			IBAN:          "BH29DMOC20000000000001",
			SwiftBIC:      "DMOCBHBM",
			Currency:      "BHD",
			IsActive:      true,
			DisplayOrder:  10,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	for _, bank := range banks {
		var existing CompanyBankAccount
		result := a.db.Where("id = ?", bank.ID).First(&existing)
		if result.Error == nil {
			updates := map[string]any{
				"division":       normalizeDivisionName(firstNonEmptyString(bank.Division, existing.Division)),
				"bank_name":      bank.BankName,
				"account_name":   bank.AccountName,
				"account_number": bank.AccountNumber,
				"iban":           bank.IBAN,
				"swift_bic":      bank.SwiftBIC,
				"currency":       bank.Currency,
				"is_active":      bank.IsActive,
				"display_order":  bank.DisplayOrder,
				"updated_at":     time.Now(),
			}
			if err := a.db.Model(&existing).Updates(updates).Error; err != nil {
				log.Printf("⚠️ Failed to refresh bank account %s: %v", bank.ID, err)
				return err
			}
			continue
		}
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Printf("⚠️ Failed to inspect bank account %s: %v", bank.ID, result.Error)
			return result.Error
		}
		if err := a.db.Create(&bank).Error; err != nil {
			log.Printf("⚠️ Failed to seed bank account %s: %v", bank.ID, err)
			return err
		}
	}

	log.Println("✅ Company bank accounts seeded/refreshed")
	return nil
}

// GetActiveBankAccounts returns all active bank accounts for invoice PDF
// Ordered by display_order for consistent presentation
func (a *App) GetActiveBankAccounts() ([]CompanyBankAccount, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}

	return a.getActiveBankAccountsForDocuments()
}

// getActiveBankAccountsForDocuments returns active bank accounts for
// document-context callers (e.g. invoice PDF generation) that have already
// passed their own document gate. It is deliberately unguarded by
// finance:view so an operator holding only invoices:view can still produce a
// payable invoice, and it auto-seeds the demo accounts if the table is empty.
func (a *App) getActiveBankAccountsForDocuments() ([]CompanyBankAccount, error) {
	if a.db == nil {
		return nil, nil
	}

	var banks []CompanyBankAccount
	if err := a.db.Where("is_active = ?", true).Order("display_order").Find(&banks).Error; err != nil {
		return nil, err
	}
	if len(banks) > 0 {
		return banks, nil
	}

	if err := a.seedCompanyBankAccountsInternal(); err != nil {
		return nil, err
	}
	err := a.db.Where("is_active = ?", true).Order("display_order").Find(&banks).Error
	return banks, err
}

// GetBankAccountByID retrieves a specific bank account
func (a *App) GetBankAccountByID(id string) (*CompanyBankAccount, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, nil
	}

	var bank CompanyBankAccount
	err := a.db.First(&bank, "id = ?", id).Error
	if err != nil {
		// Log detailed error for debugging
		log.Printf("GetBankAccountByID error: %v", err)
		// Return sanitized error to frontend
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}
	return &bank, nil
}

// GetAllBankAccounts returns ALL accounts including inactive (for management UI)
func (a *App) GetAllBankAccounts() ([]CompanyBankAccount, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, nil
	}

	var banks []CompanyBankAccount
	err := a.db.Order("display_order").Find(&banks).Error
	if err != nil {
		log.Printf("GetAllBankAccounts error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}
	return banks, nil
}

// CreateBankAccount creates a new bank account (user-entered or from OCR)
func (a *App) CreateBankAccount(account CompanyBankAccount) (*CompanyBankAccount, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Validation
	if account.BankName == "" {
		return nil, fmt.Errorf("bank name is required")
	}
	if account.AccountNumber == "" {
		return nil, fmt.Errorf("account number is required")
	}
	// BookingRate is the historic rate when the account was opened (for FX revaluation).
	// 0 means "not set"; anything else must be a realistic FX rate (< 1000 BHD per foreign unit).
	if account.BookingRate != 0 && (account.BookingRate < 0.0001 || account.BookingRate > 1000) {
		return nil, fmt.Errorf("booking rate %.6f is outside valid FX range [0.0001, 1000]", account.BookingRate)
	}

	// Generate UUID if not provided
	if account.ID == "" {
		account.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	// Default currency to BHD
	if account.Currency == "" {
		account.Currency = "BHD"
	}
	account.Division = normalizeDivisionName(account.Division)

	// Default is_active to true
	account.IsActive = true

	// Set display_order to max+1 if not provided
	if account.DisplayOrder == 0 {
		var maxOrder int
		a.db.Model(&CompanyBankAccount{}).Select("COALESCE(MAX(display_order), 0)").Scan(&maxOrder)
		account.DisplayOrder = maxOrder + 1
	}

	// Create in database
	err := a.db.Create(&account).Error
	if err != nil {
		log.Printf("CreateBankAccount error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	log.Printf("✅ Created bank account: %s - %s", account.BankName, account.AccountNumber)
	return &account, nil
}

// UpdateBankAccount updates an existing bank account
func (a *App) UpdateBankAccount(id string, updates map[string]any) (*CompanyBankAccount, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Find existing account
	var account CompanyBankAccount
	err := a.db.First(&account, "id = ?", id).Error
	if err != nil {
		log.Printf("UpdateBankAccount find error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	// Mission I (I-12): whitelist — this map lands on the account details
	// printed on customer invoices; only the editable profile fields pass.
	allowedColumns := map[string]bool{
		"division": true, "bank_name": true, "account_name": true,
		"account_number": true, "iban": true, "swift_bic": true,
		"currency": true, "is_active": true, "display_order": true,
		"booking_rate": true,
	}
	filtered := make(map[string]any, len(updates))
	for key, value := range updates {
		if allowedColumns[key] {
			filtered[key] = value
		} else {
			log.Printf("⚠️ UpdateBankAccount: dropped non-editable column %q", key)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no editable fields in update payload")
	}

	// Set updated_at
	filtered["updated_at"] = time.Now()
	if division, ok := filtered["division"].(string); ok {
		filtered["division"] = normalizeDivisionName(division)
	}

	// Apply updates
	err = a.db.Model(&account).Updates(filtered).Error
	if err != nil {
		log.Printf("UpdateBankAccount update error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	// Fetch updated record
	err = a.db.First(&account, "id = ?", id).Error
	if err != nil {
		log.Printf("UpdateBankAccount refetch error: %v", err)
		return nil, fmt.Errorf("operation failed. Please try again or contact support")
	}

	log.Printf("✅ Updated bank account: %s", id)
	return &account, nil
}

// DeleteBankAccount soft deletes a bank account (sets is_active=false)
func (a *App) DeleteBankAccount(id string) error {
	if err := a.requirePermission("finance:delete"); err != nil {
		return err
	}

	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Check if account has linked bank statements
	var statementCount int64
	err := a.db.Model(&BankStatement{}).Where("bank_account_id = ?", id).Count(&statementCount).Error
	if err != nil {
		log.Printf("DeleteBankAccount count error: %v", err)
		return fmt.Errorf("operation failed. Please try again or contact support")
	}

	if statementCount > 0 {
		return fmt.Errorf("cannot delete account with linked bank statements - please deactivate instead")
	}

	// Soft delete - set is_active to false
	err = a.db.Model(&CompanyBankAccount{}).Where("id = ?", id).Update("is_active", false).Error
	if err != nil {
		log.Printf("DeleteBankAccount deactivate error: %v", err)
		return fmt.Errorf("operation failed. Please try again or contact support")
	}

	log.Printf("✅ Deactivated bank account: %s", id)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// NOTE: Bank account encryption was REMOVED.
// Account numbers and IBANs are printed on every customer invoice PDF —
// they are not secrets. The old encryption was hardware-bound (derived from
// Mac serial number), making the database non-portable across machines.
// ═══════════════════════════════════════════════════════════════════════════

// MigrateBankAccountEncryption strips any leftover encrypted values from
// the 4 known seeded accounts and replaces them with plaintext.
// For any user-added accounts with unrecoverable encrypted values, it
// clears the field so the user can re-enter it via the Manage Accounts UI.
// MigrateBankAccountEncryption is the RBAC-guarded entry point (RBAC-003);
// startup uses the unexported variant.
func (a *App) MigrateBankAccountEncryption() {
	if err := a.requirePermission("*"); err != nil {
		return
	}
	a.migrateBankAccountEncryptionInternal()
}

func (a *App) migrateBankAccountEncryptionInternal() {
	if a.db == nil {
		return
	}

	// Known seeded accounts — restore their plaintext values
	knownAccounts := map[string]struct{ acct, iban string }{
		"bank-alpha": {"10000000001", "BH29DMOA10000000000001"},
		"bank-beta":  {"10000000002", "BH29DMOB10000000000002"},
		"bank-gamma": {"10000000003", "BH29DMOC10000000000003"},
		"bank-delta": {"10000000004", "BH29DMOD10000000000004"},
	}

	rows, err := a.db.Raw("SELECT id, account_number, iban FROM company_bank_accounts").Rows()
	if err != nil {
		log.Printf("⚠ Bank decryption migration: query failed: %v", err)
		return
	}
	defer rows.Close()

	fixed := 0
	for rows.Next() {
		var id, acctNum, iban string
		if err := rows.Scan(&id, &acctNum, &iban); err != nil {
			continue
		}

		// Check if value looks encrypted (base64 with length > 20)
		acctEncrypted := len(acctNum) > 20 && globalFieldCrypto != nil && globalFieldCrypto.IsEncrypted(acctNum)
		ibanEncrypted := len(iban) > 30 && globalFieldCrypto != nil && globalFieldCrypto.IsEncrypted(iban)

		if !acctEncrypted && !ibanEncrypted {
			continue // Already plaintext
		}

		updates := map[string]any{}
		if known, ok := knownAccounts[id]; ok {
			// Restore known plaintext values
			updates["account_number"] = known.acct
			updates["iban"] = known.iban
		} else {
			// Unknown account — try to decrypt, clear if impossible
			if acctEncrypted {
				if decrypted, err := globalFieldCrypto.Decrypt(acctNum); err == nil {
					updates["account_number"] = decrypted
				} else {
					updates["account_number"] = ""
					log.Printf("⚠ Cleared unrecoverable encrypted account_number for %s — re-enter via Manage Accounts", id)
				}
			}
			if ibanEncrypted {
				if decrypted, err := globalFieldCrypto.Decrypt(iban); err == nil {
					updates["iban"] = decrypted
				} else {
					updates["iban"] = ""
					log.Printf("⚠ Cleared unrecoverable encrypted iban for %s — re-enter via Manage Accounts", id)
				}
			}
		}

		if len(updates) > 0 {
			for col, val := range updates {
				a.db.Exec(fmt.Sprintf("UPDATE company_bank_accounts SET %s = ? WHERE id = ?", col), val, id)
			}
			fixed++
		}
	}

	if fixed > 0 {
		log.Printf("🔓 Decrypted %d bank account(s) — account numbers now stored as plaintext", fixed)
	} else {
		log.Printf("✅ Bank accounts: all fields already plaintext")
	}
}
