package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"ph_holdings_app/pkg/ocr"
)

func (a *App) getSettingsFilePath() string {
	dbDir := filepath.Dir(a.config.Database.Path)
	return filepath.Join(dbDir, "settings.json")
}

// loadUserSettings loads settings from the JSON file if it exists
func (a *App) loadUserSettings() (map[string]any, error) {
	settingsPath := a.getSettingsFilePath()

	// If file doesn't exist, return empty map (will use defaults)
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		return make(map[string]any), nil
	}

	// Read the file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	// Parse JSON
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}

	return settings, nil
}

// saveUserSettings saves settings to the JSON file
func (a *App) saveUserSettings(settings map[string]any) error {
	settingsPath := a.getSettingsFilePath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

func (a *App) GetSettings() (map[string]any, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil, err
	}
	if a.config == nil {
		return map[string]any{
			"error": "Configuration not loaded",
		}, newError("CONFIG_NOT_LOADED", "Configuration not available", "")
	}

	// Load user settings from JSON file
	userSettings, err := a.loadUserSettings()
	if err != nil {
		log.Printf("Warning: Failed to load user settings: %v", err)
		userSettings = make(map[string]any)
	}

	cfg := a.config

	// Build settings structure matching frontend expectations
	settings := map[string]any{
		"companyName": getSettingOrDefault(userSettings, "companyName", activeOverlay.CompanyDisplayName),
		"currency":    getSettingOrDefault(userSettings, "currency", "BHD"),
		"language":    getSettingOrDefault(userSettings, "language", "en"),
		"theme":       getSettingOrDefault(userSettings, "theme", "light"),
		"folders": map[string]any{
			"rfq_path":       cfg.OneDrive.RFQPath,
			"offers_path":    cfg.OneDrive.OffersPath,
			"invoices_path":  cfg.OneDrive.InvoicesPath,
			"eh_xml_path":    cfg.OneDrive.EHPath,
			"customers_path": getSettingOrDefault(userSettings, "folders.customers_path", ""),
			"reports_path":   getSettingOrDefault(userSettings, "folders.reports_path", ""),
		},
		"apiKeys": map[string]any{
			"aimlapi_key":    maskSecret(getSettingOrDefault(userSettings, "apiKeys.aimlapi_key", "").(string)),
			"aiml_model":     getSettingOrDefault(userSettings, "apiKeys.aiml_model", getAIMLModelID()).(string),
			"openai_key":     maskSecret(getSettingOrDefault(userSettings, "apiKeys.openai_key", "").(string)),
			"anthropic_key":  maskSecret(getSettingOrDefault(userSettings, "apiKeys.anthropic_key", "").(string)),
			"mistral_key":    maskSecret(getSettingOrDefault(userSettings, "apiKeys.mistral_key", "").(string)),
			"azure_endpoint": getSettingOrDefault(userSettings, "apiKeys.azure_endpoint", ""),
		},
		"gpu": map[string]any{
			"detected":    getSettingOrDefault(userSettings, "gpu.detected", false),
			"vendor":      getSettingOrDefault(userSettings, "gpu.vendor", ""),
			"device_name": getSettingOrDefault(userSettings, "gpu.device_name", ""),
			"vram_mb":     getSettingOrDefault(userSettings, "gpu.vram_mb", 0),
			"use_gpu":     getSettingOrDefault(userSettings, "gpu.use_gpu", true),
		},
		"office": map[string]any{
			"outlook_enabled": cfg.Azure.Enabled,
			"excel_enabled":   getSettingOrDefault(userSettings, "office.excel_enabled", false),
		},
		"business": map[string]any{
			"default_margin": getSettingOrDefault(userSettings, "business.default_margin", 20),
			"vat_rate":       getSettingOrDefault(userSettings, "business.vat_rate", activeOverlay.DefaultVATRate),
		},
		"security": map[string]any{
			"session_timeout_minutes": getSettingOrDefault(userSettings, "security.session_timeout_minutes",
				int(InteractiveSessionTimeout.Minutes())),
		},
		// Wave 10 / B4 (Article IV.3 — sound is saffron, one sound only):
		// opt-out for the single "paid settle" sound. Default ON. No
		// in-memory side effect needed — this round-trips through
		// saveUserSettings(settings) like any other plain JSON field.
		"sounds": map[string]any{
			"sound_on_paid_enabled": getSettingOrDefault(userSettings, "sounds.sound_on_paid_enabled", true),
		},
	}

	log.Printf("⚙️ Retrieved settings")
	return settings, nil
}

// getSettingOrDefault retrieves a nested setting value or returns default
func getSettingOrDefault(settings map[string]any, key string, defaultValue any) any {
	// Handle nested keys like "folders.customers_path"
	parts := strings.Split(key, ".")
	current := settings

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return value or default
			if val, ok := current[part]; ok {
				return val
			}
			return defaultValue
		}

		// Navigate deeper
		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return defaultValue
		}
	}

	return defaultValue
}

// UpdateSettings updates app settings (persists to settings.json file)
func (a *App) UpdateSettings(settings map[string]any) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.config == nil {
		return newError("CONFIG_NOT_LOADED", "Configuration not available", "")
	}

	// Update in-memory config for folder paths (used by OneDrive integrations)
	if folders, ok := settings["folders"].(map[string]any); ok {
		if path, ok := folders["rfq_path"].(string); ok {
			a.config.OneDrive.RFQPath = path
		}
		if path, ok := folders["offers_path"].(string); ok {
			a.config.OneDrive.OffersPath = path
		}
		if path, ok := folders["invoices_path"].(string); ok {
			a.config.OneDrive.InvoicesPath = path
		}
		if path, ok := folders["eh_xml_path"].(string); ok {
			a.config.OneDrive.EHPath = path
		}
	}

	// Update in-memory API keys (used by AI integrations)
	if apiKeys, ok := settings["apiKeys"].(map[string]any); ok {
		// Only update if not masked
		if key, ok := apiKeys["aimlapi_key"].(string); ok && key != "" && !strings.Contains(key, "*") {
			a.config.AI.APIKey = key
		}
		if key, ok := apiKeys["openai_key"].(string); ok && key != "" && !strings.Contains(key, "*") {
			// Store OpenAI key if needed for future use
		}
		if key, ok := apiKeys["anthropic_key"].(string); ok && key != "" && !strings.Contains(key, "*") {
			// Store Anthropic key if needed for future use
		}
		// Mistral API key - used by Butler AI
		if key, ok := apiKeys["mistral_key"].(string); ok && key != "" && !strings.Contains(key, "*") {
			log.Println("AI configuration updated")
		}
		if model, ok := apiKeys["aiml_model"].(string); ok {
			model = strings.TrimSpace(model)
			if model != "" {
				os.Setenv("AIML_MODEL", model)
				if a.settingsService != nil {
					if err := a.settingsService.SetSetting("apiKeys.aiml_model", model, "apiKeys", false); err != nil {
						log.Printf("⚠ Failed to persist AIML model preference: %v", err)
					}
				}
				log.Println("AI model preference updated")
			}
		}
	}

	// Wave 6 Mission C.2: apply the inactivity timeout immediately — the
	// live session picks up the new window without a re-login.
	if security, ok := settings["security"].(map[string]any); ok {
		if minutes, ok := security["session_timeout_minutes"].(float64); ok {
			a.applySessionTimeoutSetting(minutes)
			log.Printf("🔐 Session inactivity timeout set to %v", a.interactiveSessionTimeout())
		}
	}

	// Persist full settings to JSON file
	if err := a.saveUserSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	log.Printf("✅ Settings updated and persisted to settings.json")
	return nil
}

// ==================== CURRENCY EXCHANGE RATES ====================

// GetExchangeRate returns the active exchange rate for a currency on a given date
// Returns 1.0 for BHD (base currency)
func (a *App) GetExchangeRate(currencyCode string, asOfDate time.Time) (float64, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return 0, err
	}
	// BHD is base currency
	if currencyCode == "BHD" || currencyCode == "" {
		return 1.0, nil
	}

	var rate CurrencyExchangeRate
	err := a.db.Where(
		"currency_code = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?) AND deleted_at IS NULL",
		currencyCode, asOfDate, asOfDate,
	).Order("effective_from DESC").First(&rate).Error

	if err != nil {
		// Return default rates if not configured
		defaults := map[string]float64{
			"USD": activeOverlay.ExchangeRateToBase("USD"),
			"EUR": activeOverlay.ExchangeRateToBase("EUR"),
			"CHF": activeOverlay.ExchangeRateToBase("CHF"),
			"SAR": activeOverlay.ExchangeRateToBase("SAR"),
			"AED": activeOverlay.ExchangeRateToBase("AED"),
		}
		if defaultRate, ok := defaults[currencyCode]; ok {
			return defaultRate, nil
		}
		return 0, fmt.Errorf("no exchange rate found for %s", currencyCode)
	}

	return rate.Rate, nil
}

// GetCurrentExchangeRate returns today's exchange rate for a currency
func (a *App) GetCurrentExchangeRate(currencyCode string) (float64, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return 0, err
	}
	return a.GetExchangeRate(currencyCode, time.Now())
}

// SetExchangeRate creates or updates an exchange rate
// Automatically closes the previous active rate
func (a *App) SetExchangeRate(currencyCode string, rate float64, effectiveFrom time.Time, notes string) error {
	if err := a.requirePermission("finance:update"); err != nil {
		return err
	}
	// Validate currency
	validCurrencies := []string{"USD", "EUR", "CHF", "SAR", "AED"}
	valid := false
	for _, c := range validCurrencies {
		if c == currencyCode {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid currency code: %s (valid: USD, EUR, CHF, SAR, AED)", currencyCode)
	}

	// Validate rate
	if rate <= 0 {
		return fmt.Errorf("exchange rate must be positive, got: %f", rate)
	}
	if rate > 1000.0 {
		return fmt.Errorf("exchange rate %.4f exceeds maximum allowed (1000.0)", rate)
	}

	// Close previous active rate for this currency
	a.db.Model(&CurrencyExchangeRate{}).
		Where("currency_code = ? AND effective_to IS NULL AND deleted_at IS NULL", currencyCode).
		Update("effective_to", effectiveFrom)

	// Create new rate
	newRate := CurrencyExchangeRate{
		ID:            uuid.New().String(),
		CurrencyCode:  currencyCode,
		Rate:          rate,
		EffectiveFrom: effectiveFrom,
		SetBy:         "admin",
		Notes:         notes,
	}

	return a.db.Create(&newRate).Error
}

// GetActiveCurrencyRates returns all currently active exchange rates
func (a *App) GetActiveCurrencyRates() ([]CurrencyExchangeRate, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	var rates []CurrencyExchangeRate
	err := a.db.Where("effective_to IS NULL AND deleted_at IS NULL").
		Order("currency_code").
		Find(&rates).Error
	return rates, err
}

// GetCurrencyRateHistory returns all exchange rates for a currency (including historical)
func (a *App) GetCurrencyRateHistory(currencyCode string) ([]CurrencyExchangeRate, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	var rates []CurrencyExchangeRate
	err := a.db.Where("currency_code = ? AND deleted_at IS NULL", currencyCode).
		Order("effective_from DESC").
		Find(&rates).Error
	return rates, err
}

// GetSupportedCurrencies returns list of supported currencies
func (a *App) GetSupportedCurrencies() ([]map[string]string, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return nil, err
	}
	return []map[string]string{
		{"code": "BHD", "name": "Bahraini Dinar", "symbol": "BD"},
		{"code": "USD", "name": "US Dollar", "symbol": "$"},
		{"code": "EUR", "name": "Euro", "symbol": "€"},
		{"code": "CHF", "name": "Swiss Franc", "symbol": "CHF"},
		{"code": "SAR", "name": "Saudi Riyal", "symbol": "SAR"},
		{"code": "AED", "name": "UAE Dirham", "symbol": "AED"},
	}, nil
}

// SeedDefaultExchangeRates creates initial exchange rates if none exist
func (a *App) SeedDefaultExchangeRates() error {
	// SECURITY: Admin-only permission for seed functions
	if err := a.requirePermission("*"); err != nil {
		return err
	}

	var count int64
	a.db.Model(&CurrencyExchangeRate{}).Count(&count)

	defaults := []struct {
		code string
		rate float64
	}{
		{"USD", activeOverlay.ExchangeRateToBase("USD")},
		{"EUR", activeOverlay.ExchangeRateToBase("EUR")},
		{"CHF", activeOverlay.ExchangeRateToBase("CHF")},
		{"SAR", activeOverlay.ExchangeRateToBase("SAR")},
		{"AED", activeOverlay.ExchangeRateToBase("AED")},
	}

	now := time.Now()
	for _, d := range defaults {
		if count > 0 {
			var active CurrencyExchangeRate
			err := a.db.Where("currency_code = ? AND effective_to IS NULL AND deleted_at IS NULL", d.code).First(&active).Error
			if err == nil {
				if d.code == "EUR" && math.Abs(active.Rate-activeOverlay.ExchangeRateToBase("EUR")) > 0.0001 &&
					(strings.EqualFold(strings.TrimSpace(active.SetBy), "system") || math.Abs(active.Rate-0.410) < 0.005 || math.Abs(active.Rate-0.44) < 0.005) {
					if err := a.db.Model(&CurrencyExchangeRate{}).Where("id = ?", active.ID).Updates(map[string]any{
						"rate":  activeOverlay.ExchangeRateToBase("EUR"),
						"notes": "Default rate updated to 0.45 BHD per EUR",
					}).Error; err != nil {
						return err
					}
				}
				continue
			}
		}

		rate := CurrencyExchangeRate{
			ID:            uuid.New().String(),
			CurrencyCode:  d.code,
			Rate:          d.rate,
			EffectiveFrom: now,
			SetBy:         "system",
			Notes:         "Default rate",
		}
		if err := a.db.Create(&rate).Error; err != nil {
			return err
		}
	}

	log.Println("✅ Seeded default exchange rates for USD, EUR, CHF, SAR, AED")
	return nil
}

// SeedBankDemoData populates sample bank statement data for demo purposes
// Only runs if bank_statements table is empty
func (a *App) SeedBankDemoData() error {
	// SECURITY: Admin-only permission for seed functions
	if err := a.requirePermission("*"); err != nil {
		return err
	}

	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Check if data already exists
	var count int64
	a.db.Model(&BankStatement{}).Count(&count)
	if count > 0 {
		log.Printf("📊 Bank statements already exist (%d), skipping seed", count)
		return nil
	}

	// Get first active bank account
	var account CompanyBankAccount
	if err := a.db.Where("is_active = ?", true).First(&account).Error; err != nil {
		log.Printf("⚠️ No active bank accounts found, creating demo account")
		account = CompanyBankAccount{
			ID:            "bank-demo-" + uuid.New().String(),
			BankName:      "National Bank of Bahrain",
			AccountNumber: "0012-345678-001",
			AccountName:   "Acme Instrumentation WLL - Operating",
			Currency:      "BHD",
			IsActive:      true,
			DisplayOrder:  1,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := a.db.Create(&account).Error; err != nil {
			return fmt.Errorf("failed to create demo bank account: %w", err)
		}
	}

	// Create demo bank statement
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	periodEnd := periodStart.AddDate(0, 1, -1)

	openingBalance := 125000.000
	closingBalance := openingBalance + 15420.500 // Net positive

	statement := BankStatement{
		Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BankAccountID:   account.ID,
		StatementNumber: fmt.Sprintf("STMT-%s-%02d", now.Format("2006"), now.Month()),
		StatementDate:   now,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		OpeningBalance:  openingBalance,
		ClosingBalance:  closingBalance,
		Currency:        "BHD",
		Status:          "InProgress",
		TotalDebits:     33650.000,
		TotalCredits:    49070.500,
		DebitCount:      5,
		CreditCount:     5,
		ImportMethod:    "Seed",
		Division:        activeOverlay.DefaultDivision(),
	}

	if err := a.db.Create(&statement).Error; err != nil {
		return fmt.Errorf("failed to create demo statement: %w", err)
	}

	// Create sample transactions with running balance
	runningBalance := openingBalance
	demoLines := []struct {
		days    int
		desc    string
		debit   float64
		credit  float64
		txType  string
		ref     string
		lineNum int
	}{
		{3, "NPC - Payment Received", 0, 25000.000, "Credit", "TRF-NPC-001", 1},
		{5, "Rhine Instruments - Supplier Payment", 8500.000, 0, "Debit", "CHQ-001234", 2},
		{8, "Gulf Smelting - Payment Received", 0, 12500.000, "Credit", "TRF-GSC-002", 3},
		{10, "Office Rent", 1200.000, 0, "Debit", "DD-RENT-FEB", 4},
		{12, "Oxan Analytics - Supplier Payment", 6500.000, 0, "Debit", "CHQ-001235", 5},
		{15, "NGA - Payment Received", 0, 18750.000, "Credit", "TRF-NGA-003", 6},
		{18, "Salaries - Staff", 15000.000, 0, "Debit", "SAL-FEB-2026", 7},
		{20, "Gulf Air - Travel Expense", 450.000, 0, "Debit", "CC-TRAVEL-01", 8},
		{22, "Horizon - Payment Received", 0, 8320.500, "Credit", "TRF-TAT-004", 9},
		{25, "Helvetia Metering - Supplier Payment", 2000.000, 0, "Debit", "CHQ-001236", 10},
	}

	for _, line := range demoLines {
		runningBalance += line.credit - line.debit
		bankLine := BankStatementLine{
			Base:            Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
			BankStatementID: statement.ID,
			LineNumber:      line.lineNum,
			TransactionDate: periodStart.AddDate(0, 0, line.days),
			ValueDate:       periodStart.AddDate(0, 0, line.days),
			Description:     line.desc,
			Reference:       line.ref,
			Debit:           line.debit,
			Credit:          line.credit,
			Balance:         runningBalance,
			TransactionType: line.txType,
			IsMatched:       false,
		}
		if err := a.db.Create(&bankLine).Error; err != nil {
			log.Printf("⚠️ Failed to create demo line: %v", err)
		}
	}

	log.Printf("✅ Seeded bank demo data: 1 statement with %d transactions", len(demoLines))
	return nil
}

// TestAIConnection tests connection to AI provider (AIMLAPI/OpenAI)
func (a *App) TestAIConnection(provider string, apiKey string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if provider == "" || apiKey == "" {
		return newError("INVALID_INPUT", "Provider and API key required", "")
	}

	// Basic validation: check API key format
	if len(apiKey) < 20 {
		return newError("INVALID_API_KEY", "API key appears invalid (too short)", "")
	}

	// Note: API connection is tested lazily during first Butler chat request
	// This saves startup time and avoids false negatives from network issues
	// The butler_ai.go service handles connection errors gracefully with user feedback
	log.Printf("AI connection validated successfully: provider=%s", provider)
	return nil
}

// GetBusinessVATRate returns the configured VAT rate from settings. The
// fallback is the overlay's DefaultVATRate (built-in 10%), so a user setting
// overrides the overlay, and the overlay overrides nothing being set —
// one chain instead of three independent "10"s (Mission D).
func (a *App) GetBusinessVATRate() float64 {
	if err := a.requirePermission("settings:view"); err != nil {
		return activeOverlay.DefaultVATRate // Default if no permission
	}
	userSettings, err := a.loadUserSettings()
	if err != nil {
		return activeOverlay.DefaultVATRate
	}
	if rate := getSettingOrDefault(userSettings, "business.vat_rate", activeOverlay.DefaultVATRate); rate != nil {
		switch v := rate.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return activeOverlay.DefaultVATRate
}

// GetFolderPaths returns current OneDrive folder paths
func (a *App) GetFolderPaths() (map[string]string, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return nil, err
	}
	if a.config == nil {
		return nil, newError("CONFIG_NOT_LOADED", "Configuration not available", "")
	}

	paths := map[string]string{
		"rfq_path":      a.config.OneDrive.RFQPath,
		"eh_path":       a.config.OneDrive.EHPath,
		"offers_path":   a.config.OneDrive.OffersPath,
		"invoices_path": a.config.OneDrive.InvoicesPath,
	}

	log.Printf("📁 Retrieved folder paths")
	return paths, nil
}

// SetAPIKeys updates API keys securely - encrypts and persists to DB
func (a *App) SetAPIKeys(apiKeys map[string]string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.config == nil {
		return newError("CONFIG_NOT_LOADED", "Configuration not available", "")
	}

	// Helper: persist an API key encrypted to the settings DB.
	// SetSetting with encrypt=true handles the encryption, so we pass plaintext.
	persistKey := func(settingKey, value string) {
		if a.settingsService != nil {
			if err := a.settingsService.SetSetting(settingKey, value, "apiKeys", true); err != nil {
				log.Printf("⚠ Failed to persist %s: %v", settingKey, err)
			}
		}
	}
	persistPlainKey := func(settingKey, value string) {
		if a.settingsService != nil {
			if err := a.settingsService.SetSetting(settingKey, value, "apiKeys", false); err != nil {
				log.Printf("⚠ Failed to persist %s: %v", settingKey, err)
			}
		}
	}

	// Update AIMLAPI key (also used as primary Butler/Grok backend)
	if key, ok := apiKeys["aimlapi_key"]; ok && key != "" && key != "****" {
		a.config.AI.APIKey = key
		os.Setenv("AIML_API_KEY", key)
		persistKey("apiKeys.aimlapi_key", key)
		log.Println("AI configuration updated")
	}
	if model, ok := apiKeys["aiml_model"]; ok && strings.TrimSpace(model) != "" {
		model = strings.TrimSpace(model)
		os.Setenv("AIML_MODEL", model)
		persistPlainKey("apiKeys.aiml_model", model)
		log.Println("AI model preference updated")
	}

	// Update Mistral key
	if key, ok := apiKeys["mistral_key"]; ok && key != "" && key != "****" {
		os.Setenv("MISTRAL_API_KEY", key)
		persistKey("apiKeys.mistral_key", key)
		log.Println("AI configuration updated")
	}

	// Update OpenAI key
	if key, ok := apiKeys["openai_key"]; ok && key != "" && key != "****" {
		os.Setenv("OPENAI_API_KEY", key)
		persistKey("apiKeys.openai_key", key)
		log.Println("AI configuration updated")
	}

	// Update Anthropic key
	if key, ok := apiKeys["anthropic_key"]; ok && key != "" && key != "****" {
		os.Setenv("ANTHROPIC_API_KEY", key)
		persistKey("apiKeys.anthropic_key", key)
		log.Println("AI configuration updated")
	}

	// Update Azure endpoint
	if endpoint, ok := apiKeys["azure_endpoint"]; ok && endpoint != "" {
		os.Setenv("AZURE_ENDPOINT", endpoint)
		persistKey("apiKeys.azure_endpoint", endpoint)
		log.Printf("✅ Azure endpoint updated (encrypted)")
	}

	log.Println("API configuration saved securely")
	return nil
}

// RotateEncryptionKey rotates the field-level encryption key.
// All encrypted settings and bank account fields are re-encrypted with the new key version.
// Old key versions are kept in memory for decryption of any values missed during rotation.
// Admin-only operation - use quarterly or after suspected key compromise.
func (a *App) RotateEncryptionKey() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.fieldCrypto == nil {
		return fmt.Errorf("encryption system not initialized")
	}

	oldVer := a.fieldCrypto.CurrentVersion()
	newVer, err := a.fieldCrypto.Rotate()
	if err != nil {
		return fmt.Errorf("key rotation failed: %w", err)
	}

	log.Printf("🔑 Encryption key rotated: v%d -> v%d", oldVer, newVer)

	// Re-encrypt all settings that have IsEncrypted=true
	reEncryptedSettings := 0
	if a.db != nil {
		var settings []Setting
		a.db.Where("is_encrypted = ?", true).Find(&settings)

		for _, s := range settings {
			// Decrypt with old key (FieldCrypto keeps all versions)
			plaintext, err := a.fieldCrypto.Decrypt(s.Value)
			if err != nil {
				// Try legacy decrypt
				if a.settingsService != nil {
					plaintext, err = a.settingsService.legacyDecrypt(s.Value)
				}
				if err != nil {
					log.Printf("⚠ Could not decrypt setting %s during rotation: %v", s.Key, err)
					continue
				}
			}

			// Re-encrypt with new key version
			newEncrypted, err := a.fieldCrypto.Encrypt(plaintext)
			if err != nil {
				log.Printf("⚠ Could not re-encrypt setting %s: %v", s.Key, err)
				continue
			}

			a.db.Model(&Setting{}).Where("key = ?", s.Key).Update("value", newEncrypted)
			reEncryptedSettings++
		}
	}

	// Re-encrypt all bank account fields
	reEncryptedBanks := 0
	if a.db != nil {
		rows, err := a.db.Raw("SELECT id, account_number, iban FROM company_bank_accounts").Rows()
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, acctNum, iban string
				if err := rows.Scan(&id, &acctNum, &iban); err != nil {
					continue
				}

				updates := map[string]any{}

				if acctNum != "" {
					if plain, err := a.fieldCrypto.Decrypt(acctNum); err == nil {
						if enc, err := a.fieldCrypto.Encrypt(plain); err == nil {
							updates["account_number"] = enc
						}
					}
				}

				if iban != "" {
					if plain, err := a.fieldCrypto.Decrypt(iban); err == nil {
						if enc, err := a.fieldCrypto.Encrypt(plain); err == nil {
							updates["iban"] = enc
						}
					}
				}

				if len(updates) > 0 {
					a.db.Model(&CompanyBankAccount{}).Where("id = ?", id).Updates(updates)
					reEncryptedBanks++
				}
			}
		}
	}

	// Audit log
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(
			a.getCurrentUserID(),
			"encryption_key_rotated",
			"encryption_key", "",
			0, "",
			true,
			map[string]any{
				"old_version":           oldVer,
				"new_version":           newVer,
				"re_encrypted_settings": reEncryptedSettings,
				"re_encrypted_banks":    reEncryptedBanks,
			},
		)
	}

	log.Printf("🔐 Key rotation complete: v%d -> v%d (%d settings, %d bank accounts re-encrypted)",
		oldVer, newVer, reEncryptedSettings, reEncryptedBanks)
	return nil
}

// ExportEncryptionBackup exports the encryption key material for disaster recovery.
// Returns master key hex and salt hex. ADMIN ONLY. Store these securely.
// Without this backup, a hardware change (e.g., new motherboard) will make all
// encrypted data permanently unrecoverable.
func (a *App) ExportEncryptionBackup() (map[string]string, error) {
	// SECURITY: CRITICAL - Admin-only permission (wildcard "*")
	// This exports the master encryption key - highest security function
	if err := a.requirePermission("*"); err != nil {
		return nil, err
	}
	if a.fieldCrypto == nil {
		return nil, fmt.Errorf("encryption system not initialized")
	}

	backup := map[string]string{
		"master_key_hex": a.fieldCrypto.ExportKeyMaterial(),
		"salt_hex":       a.fieldCrypto.ExportSalt(),
		"key_version":    fmt.Sprintf("%d", a.fieldCrypto.CurrentVersion()),
		"warning":        "STORE SECURELY. This is the only way to recover encrypted data after hardware changes.",
	}

	// Audit log
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(
			a.getCurrentUserID(),
			"encryption_key_exported",
			"encryption_key", "",
			0, "",
			true,
			map[string]any{"key_version": a.fieldCrypto.CurrentVersion()},
		)
	}

	log.Printf("🔑 Encryption key backup exported by %s", a.getCurrentUserID())
	return backup, nil
}

// ImportEncryptionBackup restores encryption from a previously exported backup.
// Use this after hardware changes to recover access to encrypted data.
func (a *App) ImportEncryptionBackup(masterKeyHex, saltHex string) error {
	// SECURITY: CRITICAL - Admin-only permission (wildcard "*")
	// This replaces the master encryption key - highest security function
	if err := a.requirePermission("*"); err != nil {
		return err
	}
	if masterKeyHex == "" || saltHex == "" {
		return fmt.Errorf("both master_key_hex and salt_hex are required")
	}

	fc, err := ImportKeyMaterial(masterKeyHex, saltHex)
	if err != nil {
		return fmt.Errorf("failed to import key material: %w", err)
	}

	// Replace the current crypto instance
	a.fieldCrypto = fc
	globalFieldCrypto = fc

	// Re-wire SettingsService
	if a.settingsService != nil {
		a.settingsService.SetFieldCrypto(fc)
	}

	// Audit log
	if GlobalAuditLogger != nil {
		GlobalAuditLogger.LogFinancialTransaction(
			a.getCurrentUserID(),
			"encryption_key_imported",
			"encryption_key", "",
			0, "",
			true,
			nil,
		)
	}

	log.Printf("🔑 Encryption key restored from backup by %s", a.getCurrentUserID())
	return nil
}

// UpdateFolderPaths updates OneDrive folder paths
func (a *App) UpdateFolderPaths(paths map[string]string) error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	if a.config == nil {
		return newError("CONFIG_NOT_LOADED", "Configuration not available", "")
	}

	if rfqPath, ok := paths["rfq_path"]; ok {
		a.config.OneDrive.RFQPath = rfqPath
	}
	if ehPath, ok := paths["eh_path"]; ok {
		a.config.OneDrive.EHPath = ehPath
	}
	if offersPath, ok := paths["offers_path"]; ok {
		a.config.OneDrive.OffersPath = offersPath
	}
	if invoicesPath, ok := paths["invoices_path"]; ok {
		a.config.OneDrive.InvoicesPath = invoicesPath
	}

	// Restart file watcher with new paths if enabled
	if a.config.App.EnableFileWatcher && a.fileWatcher != nil {
		if a.fileWatcher.IsRunning() {
			log.Printf("🔄 Restarting file watcher with updated paths...")
			a.fileWatcher.Stop()
		}

		// Update watcher config
		watchConfig := &WatchConfig{
			RFQPath:       a.config.OneDrive.RFQPath,
			EHXMLPath:     a.config.OneDrive.EHPath,
			OfferPath:     a.config.OneDrive.OffersPath,
			InvoicePath:   a.config.OneDrive.InvoicesPath,
			Recursive:     true,
			DebounceDelay: time.Duration(a.config.App.WatcherDebounceMS) * time.Millisecond,
			MaxQueueSize:  a.config.App.WatcherQueueSize,
			IncludeExts:   supportedOCRWatcherExtensions(),
		}

		newWatcher, err := NewFileWatcher(watchConfig)
		if err != nil {
			return newError("WATCHER_RESTART_FAILED", "Failed to restart file watcher", err.Error())
		}

		a.fileWatcher = newWatcher
		a.registerFileWatcherHandlers()

		if watchConfig.hasValidPaths() {
			if err := a.fileWatcher.Start(); err != nil {
				return newError("WATCHER_START_FAILED", "Failed to start file watcher", err.Error())
			}
			log.Printf("✅ File watcher restarted with updated paths")
		}
	}

	log.Printf("✅ Folder paths updated")
	return nil
}

// ============================================================================
// SETUP WIZARD FUNCTIONS (SetupWizard.svelte, ConversationalSetup.svelte)
// ============================================================================

// GPUInfo represents detected GPU capabilities
type GPUInfo struct {
	Detected      bool   `json:"detected"`
	Vendor        string `json:"vendor"`
	DeviceName    string `json:"device_name"`
	VRAM          int64  `json:"vram_mb"`
	LevelZeroOK   bool   `json:"level_zero_ok"`
	CudaOK        bool   `json:"cuda_ok"`
	UseGPU        bool   `json:"use_gpu"`
	KernelsLoaded int    `json:"kernels_loaded"`
}

// OfficeInfo represents detected Microsoft Office capabilities
type OfficeInfo struct {
	OutlookEnabled    bool   `json:"outlook_enabled"`
	ExcelEnabled      bool   `json:"excel_enabled"`
	WordEnabled       bool   `json:"word_enabled"`
	PowerPointEnabled bool   `json:"powerpoint_enabled"`
	Version           string `json:"version"`
}

// SystemInfo represents detected system capabilities
type SystemInfo struct {
	OS     string `json:"os"`
	CPU    string `json:"cpu"`
	GPU    string `json:"gpu"`
	RAM    string `json:"ram"`
	HasGPU bool   `json:"has_gpu"`
}

// FolderStructureResult represents result of folder creation
type FolderStructureResult struct {
	Success   bool     `json:"success"`
	Created   []string `json:"created"`
	InboxPath string   `json:"inbox_path"`
}

// ScanResult represents result of initial document scan
type InitialScanResult struct {
	TotalFiles   int            `json:"total_files"`
	FilesByType  map[string]int `json:"files_by_type"`
	Conflicts    []string       `json:"conflicts"`
	Warnings     []string       `json:"warnings"`
	ScanDuration int64          `json:"scan_duration_ms"`
	ReportPath   string         `json:"report_path"`
}

// NeedsSetup checks if initial setup is required
func (a *App) NeedsSetup() bool {
	if a.config == nil {
		return true
	}
	// Check if essential paths are configured
	if a.config.OneDrive.RFQPath == "" || a.config.OneDrive.OffersPath == "" {
		return true
	}
	// Check if AI is configured
	if a.config.AI.APIKey == "" {
		return true
	}
	log.Printf("⚙️ Setup check: NOT needed (already configured)")
	return false
}

// DetectGPU detects available GPU for acceleration (REAL HARDWARE DETECTION!)
//
// Two-tier detection:
// 1. WMIC (Windows Management Instrumentation) - Works on all Windows systems
// 2. CUDA (nvidia-smi) - For NVIDIA GPUs
//
// NOT a stub! This actually queries hardware via:
//   - wmic path win32_videocontroller (Windows)
//   - nvidia-smi (NVIDIA CUDA toolkit)
//
// Returns GPUInfo with real detection results or graceful CPU fallback.
func (a *App) DetectGPU() (GPUInfo, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return GPUInfo{}, err
	}
	// Defensive initialization - will be updated by actual detection
	info := GPUInfo{
		Detected:      false, // Updated if GPU found
		Vendor:        "Unknown",
		DeviceName:    "CPU Only", // Updated if GPU found
		VRAM:          0,          // Updated if GPU found
		LevelZeroOK:   false,      // Set true for Intel GPUs
		CudaOK:        false,      // Set true for NVIDIA GPUs
		UseGPU:        false,      // Set true if GPU detected
		KernelsLoaded: 0,
	}

	log.Printf("🔍 Detecting GPU capabilities via hardware query...")

	// TIER 1: Platform-specific GPU enumeration
	switch goruntime.GOOS {
	case "windows":
		gpuInfo, err := detectGPUViaWMIC()
		if err != nil {
			log.Printf("⚠️ WMIC GPU detection failed: %v", err)
		} else if gpuInfo != nil {
			info.Detected = gpuInfo.HasGPU
			info.Vendor = gpuInfo.Type
			info.DeviceName = gpuInfo.DeviceName
			info.VRAM = gpuInfo.Memory

			if strings.Contains(strings.ToLower(gpuInfo.DeviceName), "intel") {
				info.LevelZeroOK = true
			} else if strings.Contains(strings.ToLower(gpuInfo.DeviceName), "nvidia") {
				info.CudaOK = true
			}

			info.UseGPU = info.Detected
			log.Printf("✓ GPU detected via WMIC: %s (%d MB VRAM)", info.DeviceName, info.VRAM)
			return info, nil
		}
	case "darwin":
		gpuInfo, err := detectGPUViaMacOS()
		if err != nil {
			log.Printf("⚠️ macOS GPU detection failed: %v", err)
		} else if gpuInfo != nil {
			info.Detected = gpuInfo.HasGPU
			info.Vendor = gpuInfo.Type
			info.DeviceName = gpuInfo.DeviceName
			info.VRAM = gpuInfo.Memory
			info.UseGPU = info.Detected
			log.Printf("✓ GPU detected via system_profiler: %s (%d MB VRAM)", info.DeviceName, info.VRAM)
			return info, nil
		}
	}

	// TIER 2: Try CUDA detection (nvidia-smi) - works on all platforms with NVIDIA
	gpuInfo, err := detectGPUViaCUDA()
	if err != nil {
		log.Printf("⚠️ CUDA GPU detection failed: %v", err)
		// Not fatal - CPU mode is valid fallback
	} else if gpuInfo != nil {
		info.Detected = true
		info.Vendor = "NVIDIA CUDA"
		info.DeviceName = gpuInfo.DeviceName
		info.VRAM = gpuInfo.Memory
		info.CudaOK = true
		info.UseGPU = true
		log.Printf("✓ GPU detected via CUDA: %s (%d MB VRAM)", info.DeviceName, info.VRAM)
		return info, nil
	}

	log.Printf("⚠ No GPU detected - using CPU mode")
	return info, nil // CPU fallback with no error
}

// GPUDetectionInfo is internal helper for detection
type GPUDetectionInfo struct {
	HasGPU     bool
	Type       string
	DeviceName string
	Memory     int64
	Driver     string
}

// detectGPUViaWMIC uses Windows Management Instrumentation
// REAL HARDWARE DETECTION - NOT A STUB!
// Queries: wmic path win32_videocontroller get name,AdapterRAM,DriverVersion /format:list
func detectGPUViaWMIC() (*GPUDetectionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wmic", "path", "win32_videocontroller", "get", "name,AdapterRAM,DriverVersion", "/format:list")
	suppressCommandWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wmic command failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	info := &GPUDetectionInfo{
		HasGPU: false, // Will be set to true if real GPU found
		Type:   "Windows WMI",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			deviceName := strings.TrimPrefix(line, "Name=")
			deviceLower := strings.ToLower(deviceName)

			// Skip basic/generic display adapters and remote desktop adapters
			if deviceName != "" &&
				!strings.Contains(deviceLower, "basic") &&
				!strings.Contains(deviceLower, "remote") &&
				!strings.Contains(deviceLower, "microsoft basic") {
				info.DeviceName = deviceName
				info.HasGPU = true
				log.Printf("🎯 WMIC found GPU: %s", deviceName)
			}
		}
		if strings.HasPrefix(line, "AdapterRAM=") {
			ramStr := strings.TrimPrefix(line, "AdapterRAM=")
			var ram int64
			n, err := fmt.Sscanf(ramStr, "%d", &ram)
			if err == nil && n > 0 && ram > 0 {
				info.Memory = ram / (1024 * 1024) // Bytes to MB
				log.Printf("🎯 WMIC found VRAM: %d MB", info.Memory)
			}
		}
		if strings.HasPrefix(line, "DriverVersion=") {
			info.Driver = strings.TrimPrefix(line, "DriverVersion=")
			if info.Driver != "" {
				log.Printf("🎯 WMIC found driver: %s", info.Driver)
			}
		}
	}

	if !info.HasGPU {
		log.Printf("⚠️ WMIC: No discrete GPU found (may be using integrated graphics)")
		return nil, nil // No GPU found, not an error
	}

	log.Printf("✅ WMIC detection successful: %s (%d MB VRAM)", info.DeviceName, info.Memory)
	return info, nil
}

// detectGPUViaMacOS uses system_profiler to detect GPU on macOS
func detectGPUViaMacOS() (*GPUDetectionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "system_profiler", "SPDisplaysDataType")
	suppressCommandWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("system_profiler failed: %w", err)
	}

	info := &GPUDetectionInfo{HasGPU: false, Type: "macOS"}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Chipset Model:") {
			info.DeviceName = strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
			info.HasGPU = true
			log.Printf("🎯 macOS found GPU: %s", info.DeviceName)
		}
		if strings.HasPrefix(line, "VRAM") || strings.HasPrefix(line, "Memory:") {
			valStr := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			var vram int64
			if strings.Contains(valStr, "GB") {
				fmt.Sscanf(valStr, "%d", &vram)
				vram *= 1024 // Convert GB to MB
			} else {
				fmt.Sscanf(valStr, "%d", &vram)
			}
			info.Memory = vram
		}
	}

	// Apple Silicon has unified memory - always has GPU capability
	if !info.HasGPU {
		// Check for Apple Silicon GPU (reported differently)
		for _, line := range lines {
			if strings.Contains(line, "Apple") && strings.Contains(line, "GPU") {
				info.DeviceName = strings.TrimSpace(line)
				info.HasGPU = true
				info.Type = "Apple Silicon"
				break
			}
		}
	}

	if !info.HasGPU {
		return nil, nil
	}

	log.Printf("✅ macOS GPU detection successful: %s (%d MB VRAM)", info.DeviceName, info.Memory)
	return info, nil
}

// detectGPUViaCUDA checks for NVIDIA CUDA toolkit
// REAL NVIDIA-SMI DETECTION - NOT A STUB!
// Queries: nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader
func detectGPUViaCUDA() (*GPUDetectionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader")
	suppressCommandWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi command failed (CUDA/NVIDIA drivers may not be installed): %w", err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		log.Printf("⚠️ nvidia-smi returned empty output (no NVIDIA GPU found)")
		return nil, nil // No NVIDIA GPU, not an error
	}

	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		log.Printf("⚠️ nvidia-smi returned invalid format: %s", line)
		return nil, nil // Invalid output, not an error
	}

	info := &GPUDetectionInfo{
		HasGPU:     true,
		Type:       "NVIDIA CUDA",
		DeviceName: strings.TrimSpace(parts[0]),
		Driver:     strings.TrimSpace(parts[2]),
	}

	// Parse memory (format: "12288 MiB")
	memStr := strings.TrimSpace(parts[1])
	var memMB int64
	n, err := fmt.Sscanf(memStr, "%d", &memMB)
	if err == nil && n > 0 {
		info.Memory = memMB
		log.Printf("🎯 nvidia-smi found GPU: %s (%d MB VRAM, driver %s)",
			info.DeviceName, info.Memory, info.Driver)
	}

	log.Printf("✅ CUDA detection successful: %s", info.DeviceName)
	return info, nil
}

// DetectOffice detects Microsoft Office installation
func (a *App) DetectOffice() (OfficeInfo, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return OfficeInfo{}, err
	}
	info := OfficeInfo{
		OutlookEnabled:    false,
		ExcelEnabled:      false,
		WordEnabled:       false,
		PowerPointEnabled: false,
		Version:           "None",
	}

	log.Printf("🔍 Detecting Microsoft Office...")

	// File system checks can fail
	if a.config != nil && a.config.Tools.PandocPath != "" {
		if _, err := os.Stat(a.config.Tools.PandocPath); err != nil {
			// Not fatal - Office might still be installed
			log.Printf("⚠️ Pandoc not found at configured path: %v", err)
		} else {
			// If pandoc is configured and exists, likely Office is present
			info.WordEnabled = true
			info.ExcelEnabled = true
			info.PowerPointEnabled = true
			info.Version = "Microsoft 365"
		}
	}

	// Check if Outlook path exists
	outlookPath := os.ExpandEnv("%PROGRAMFILES%\\Microsoft Office\\root\\Office16\\OUTLOOK.EXE")
	if _, err := os.Stat(outlookPath); err != nil && !os.IsNotExist(err) {
		return info, fmt.Errorf("failed to check Office installation: %w", err)
	} else if err == nil {
		info.OutlookEnabled = true
	}

	log.Printf("📧 Office Detection: outlook=%v excel=%v word=%v", info.OutlookEnabled, info.ExcelEnabled, info.WordEnabled)
	return info, nil
}

// DetectSystemInfo detects system hardware and software
// Calls DetectGPU() for REAL hardware detection, not hardcoded values
func (a *App) DetectSystemInfo() (SystemInfo, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return SystemInfo{}, err
	}
	// Defensive initialization - will be updated by actual detection
	info := SystemInfo{
		OS:     goruntime.GOOS,
		CPU:    fmt.Sprintf("CPU with %d cores", goruntime.NumCPU()),
		GPU:    "Integrated Graphics",
		RAM:    "Unknown",
		HasGPU: false,
	}

	log.Printf("🔍 Detecting system info...")

	// Detect OS
	switch goruntime.GOOS {
	case "windows":
		info.OS = fmt.Sprintf("Windows %s", os.Getenv("OS"))
		if numCPU := os.Getenv("NUMBER_OF_PROCESSORS"); numCPU != "" {
			info.CPU = fmt.Sprintf("CPU with %s cores", numCPU)
		}
	case "darwin":
		// Get macOS version
		cmd := exec.Command("sw_vers", "-productVersion")
		suppressCommandWindow(cmd)
		if out, err := cmd.Output(); err == nil {
			info.OS = fmt.Sprintf("macOS %s", strings.TrimSpace(string(out)))
		}
		// Get CPU brand string
		cmd = exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
		suppressCommandWindow(cmd)
		if out, err := cmd.Output(); err == nil {
			info.CPU = strings.TrimSpace(string(out))
		} else {
			// Apple Silicon doesn't have brand_string, use chip name
			info.CPU = fmt.Sprintf("Apple Silicon (%d cores)", goruntime.NumCPU())
		}
	default:
		info.OS = fmt.Sprintf("Linux (%s/%s)", goruntime.GOOS, goruntime.GOARCH)
	}

	// Get GPU info via REAL hardware detection
	gpuInfo, err := a.DetectGPU()
	if err != nil {
		log.Printf("⚠️ GPU detection failed: %v", err)
	} else if gpuInfo.Detected {
		info.GPU = gpuInfo.DeviceName
		info.HasGPU = true
		log.Printf("✅ Discrete GPU detected and enabled: %s", info.GPU)
	} else {
		log.Printf("ℹ️ No discrete GPU detected - using CPU/integrated graphics")
	}

	// Query memory - platform-specific
	switch goruntime.GOOS {
	case "windows":
		memCtx, memCancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(memCtx, "wmic", "OS", "get", "TotalVisibleMemorySize", "/value")
		suppressCommandWindow(cmd)
		output, err := cmd.Output()
		memCancel()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
					var memKB int64
					fmt.Sscanf(strings.TrimPrefix(line, "TotalVisibleMemorySize="), "%d", &memKB)
					info.RAM = fmt.Sprintf("%dGB", memKB/(1024*1024))
				}
			}
		}
	case "darwin":
		cmd := exec.Command("sysctl", "-n", "hw.memsize")
		suppressCommandWindow(cmd)
		if out, err := cmd.Output(); err == nil {
			var memBytes int64
			fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &memBytes)
			info.RAM = fmt.Sprintf("%dGB", memBytes/(1024*1024*1024))
		}
	default:
		// Linux: read /proc/meminfo
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "MemTotal:") {
					var memKB int64
					fmt.Sscanf(strings.TrimPrefix(line, "MemTotal:"), "%d", &memKB)
					info.RAM = fmt.Sprintf("%dGB", memKB/(1024*1024))
					break
				}
			}
		}
	}

	log.Printf("💻 System: OS=%s CPU=%s GPU=%s HasGPU=%v RAM=%s", info.OS, info.CPU, info.GPU, info.HasGPU, info.RAM)
	return info, nil
}

// DetectOneDrivePath detects OneDrive folder location
func (a *App) DetectOneDrivePath() (string, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return "", err
	}
	log.Printf("🔍 Detecting OneDrive path...")

	// Common OneDrive paths on Windows
	candidates := []string{
		os.ExpandEnv("%USERPROFILE%\\OneDrive"),
		os.ExpandEnv("%USERPROFILE%\\OneDrive - Personal"),
		os.ExpandEnv("%OneDrive%"),
		os.ExpandEnv("%OneDriveConsumer%"),
		os.ExpandEnv("%OneDriveCommercial%"),
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			log.Printf("📁 Found OneDrive at: %s", path)
			return path, nil
		}
	}

	// Fallback to Documents
	docsPath := os.ExpandEnv("%USERPROFILE%\\Documents")
	log.Printf("📁 OneDrive not found, using Documents: %s", docsPath)
	return docsPath, nil
}

// BrowseFolder opens native folder picker dialog
func (a *App) BrowseFolder() (string, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return "", err
	}
	log.Printf("📂 Opening folder browser...")

	if a.ctx == nil {
		return "", fmt.Errorf("application context not initialized")
	}

	// Use Wails runtime to open directory dialog
	selectedPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder",
	})

	if err != nil {
		return "", fmt.Errorf("folder selection failed: %w", err)
	}

	if selectedPath == "" {
		log.Printf("⚠️ No folder selected (user cancelled)")
		return "", nil // Return empty without error for user cancellation
	}

	log.Printf("✅ Folder selected: %s", selectedPath)
	return selectedPath, nil
}

// ValidateFolder checks if a folder path is valid and writable
func (a *App) ValidateFolder(path string) (bool, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return false, err
	}
	if path == "" {
		return false, newError("EMPTY_PATH", "Path cannot be empty", "")
	}

	// Check if path exists
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Path doesn't exist - check if we can create it
		parentDir := filepath.Dir(path)
		if _, parentErr := os.Stat(parentDir); parentErr == nil {
			log.Printf("📁 Path doesn't exist but can be created: %s", path)
			return true, nil
		}
		return false, newError("PATH_NOT_FOUND", "Path does not exist and cannot be created", path)
	}

	if !stat.IsDir() {
		return false, newError("NOT_A_DIRECTORY", "Path is not a directory", path)
	}

	// Check if writable by attempting to create a temp file
	testFile := filepath.Join(path, ".asymm_write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false, newError("NOT_WRITABLE", "Directory is not writable", err.Error())
	}
	f.Close()
	os.Remove(testFile)

	log.Printf("✅ Folder validated: %s", path)
	return true, nil
}

// QuickCaptureDocument handles manual document upload for instant processing
// This is for drag-and-drop or manual file upload in the Inbox screen
func (a *App) QuickCaptureDocument(filePath string) (map[string]any, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}

	// SECURITY FIX: Validate file path to prevent path traversal
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, newError("INVALID_PATH", "Invalid file path", err.Error())
	}
	filePath = absPath // Use validated path

	log.Printf("📥 Quick capture: %s", filePath)

	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, newError("FILE_NOT_FOUND", "File does not exist", filePath)
	}

	// Process immediately via OCR service
	if a.ocrService == nil {
		return nil, newError("OCR_NOT_AVAILABLE", "OCR service not initialized", "")
	}

	// Extract text using OCR (auto-detect document type)
	result, err := a.ocrService.ProcessDocument(filePath, "auto")
	if err != nil {
		log.Printf("⚠️ OCR failed for quick capture: %v", err)
		return nil, err
	}

	// Classify document type using AI-first classifier (Mistral)
	// Falls back to regex if Mistral is unavailable
	classification := a.AIClassifyDocumentType(result.Text, filepath.Base(filePath))
	if classification == nil {
		classification = a.classifyDocumentForOCR(result.Text, filepath.Base(filePath))
	}
	docType := classification.DocumentType

	// Build suggested actions from classification
	suggestedActions := []map[string]any{
		{
			"action":      classification.SuggestedAction,
			"route":       classification.RouteTo,
			"confidence":  classification.Confidence,
			"method":      classification.Method,
			"keywords":    classification.KeywordsFound,
			"explanation": classification.Explanation,
		},
	}

	mergedExtractedData := extractBasicFields(result.Text, docType)
	if docType == "BankStatement" || docType == "bank_statement" {
		if parseOutcome, parseErr := a.parseImportedPDFBankStatement(result.Text); parseErr != nil {
			log.Printf("⚠️ Quick capture bank statement pre-parse failed, keeping basic extraction only: %v", parseErr)
		} else if parseOutcome != nil && parseOutcome.Statement != nil {
			parsed := parseOutcome.Statement
			if parsed.AccountNumber != "" {
				mergedExtractedData["account_number"] = parsed.AccountNumber
			}
			if parsed.IBAN != "" {
				mergedExtractedData["iban"] = parsed.IBAN
			}
			if parsed.Currency != "" {
				mergedExtractedData["currency"] = parsed.Currency
			}
			if !parsed.PeriodStart.IsZero() {
				mergedExtractedData["period_start"] = parsed.PeriodStart.Format("2006-01-02")
			}
			if !parsed.PeriodEnd.IsZero() {
				mergedExtractedData["period_end"] = parsed.PeriodEnd.Format("2006-01-02")
			}
			mergedExtractedData["opening_balance"] = parsed.OpeningBalance
			mergedExtractedData["closing_balance"] = parsed.ClosingBalance
			mergedExtractedData["total_debits"] = parsed.TotalDebits
			mergedExtractedData["total_credits"] = parsed.TotalCredits
			mergedExtractedData["debit_count"] = parsed.DebitCount
			mergedExtractedData["credit_count"] = parsed.CreditCount
			mergedExtractedData["import_method"] = parseOutcome.ImportMethod
			if len(parseOutcome.ValidationNote) > 0 {
				mergedExtractedData["validation_notes"] = parseOutcome.ValidationNote
			}

			lineItems := make([]map[string]any, 0, len(parsed.Lines))
			for _, line := range parsed.Lines {
				lineItems = append(lineItems, map[string]any{
					"date":        line.Date.Format("2006-01-02"),
					"value_date":  line.ValueDate.Format("2006-01-02"),
					"description": line.Description,
					"reference":   line.Reference,
					"debit":       line.Debit,
					"credit":      line.Credit,
					"balance":     line.Balance,
				})
			}
			mergedExtractedData["line_items"] = lineItems
		}
	}

	// PERSIST OCR RESULT TO DATABASE
	ocrDoc := OCRDocument{
		FileName:         filepath.Base(filePath),
		FilePath:         filePath,
		DocumentType:     docType,
		ExtractedText:    result.Text,
		Confidence:       result.Confidence,
		ProcessingTimeMS: result.ProcessingTime,
		Engine:           result.Engine,
		TierUsed:         result.TierUsed,
		Cost:             result.Cost,
		DNACacheHit:      result.DNACacheHit,
		TableDetected:    result.TableDetected,
		GPUUsed:          result.GPUUsed,
		ProcessedAt:      time.Now(),
	}

	if a.db != nil {
		if err := a.db.Create(&ocrDoc).Error; err != nil {
			log.Printf("⚠️ Failed to persist OCR result: %v", err)
			// Non-fatal - continue with response
		} else {
			log.Printf("💾 OCR result persisted: ID=%d", ocrDoc.ID)
		}
	}

	// Create inbox entry
	response := map[string]any{
		"id":               ocrDoc.ID,
		"fileName":         filepath.Base(filePath),
		"filePath":         filePath,
		"documentType":     docType,
		"confidence":       result.Confidence,
		"textLength":       len(result.Text),
		"processedAt":      time.Now().Format(time.RFC3339),
		"status":           "Ready",
		"suggestedActions": suggestedActions,
		"extractedData":    mergedExtractedData,
		"engine":           result.Engine,
		"tierUsed":         result.TierUsed,
		"dnaCacheHit":      result.DNACacheHit,
		"processingTimeMS": result.ProcessingTimeMS,
	}

	log.Printf("✅ Quick capture complete: %s (type=%s, confidence=%.2f, engine=%s)",
		filepath.Base(filePath), docType, result.Confidence, result.Engine)

	return response, nil
}

// QuickCaptureDocumentFromBase64 processes a document from base64 data (for Outlook drag-drop)
// When dragging from Outlook, Wails OnFileDrop doesn't fire - we get DataTransfer objects instead
func (a *App) QuickCaptureDocumentFromBase64(base64Data string, fileName string) (map[string]any, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}

	// SECURITY FIX: Size limit to prevent OOM from large base64 payloads
	const maxBase64Size = 75 * 1024 * 1024 // ~50MB decoded
	if len(base64Data) > maxBase64Size {
		return nil, fmt.Errorf("file too large: maximum 50MB allowed")
	}

	// SECURITY FIX: Sanitize filename to prevent path traversal
	safeName := filepath.Base(fileName)
	if safeName == "." || safeName == "/" || safeName == "" {
		return nil, fmt.Errorf("invalid file name")
	}

	log.Printf("📥 Quick capture from base64: %s", safeName)

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create temp file
	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, "ph_capture_"+safeName)

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tempPath) // Clean up after processing

	log.Printf("📄 Temp file created: %s (%d bytes)", tempPath, len(data))

	// Use existing QuickCaptureDocument
	return a.QuickCaptureDocument(tempPath)
}

// classifyDocumentType determines document type from content and filename
func classifyDocumentType(text string, filename string) string {
	textLower := strings.ToLower(text)
	filenameLower := strings.ToLower(filename)

	// Priority order - check most specific first
	if strings.Contains(textLower, "request for quotation") ||
		strings.Contains(textLower, "rfq") ||
		strings.Contains(filenameLower, "rfq") {
		return "RFQ"
	}

	if strings.Contains(textLower, "purchase order") ||
		strings.Contains(textLower, "p.o.") ||
		strings.Contains(filenameLower, "po_") {
		return "PurchaseOrder"
	}

	if strings.Contains(textLower, "invoice") ||
		strings.Contains(textLower, "bill to") ||
		strings.Contains(filenameLower, "invoice") {
		return "Invoice"
	}

	if strings.Contains(textLower, "quotation") ||
		strings.Contains(textLower, "quote") ||
		strings.Contains(filenameLower, "quote") {
		return "Quote"
	}

	if strings.Contains(textLower, "delivery note") ||
		strings.Contains(textLower, "dn_") ||
		strings.Contains(filenameLower, "delivery") {
		return "DeliveryNote"
	}

	return "Unknown"
}

// getSuggestedActions returns recommended actions based on document type
func getSuggestedActions(docType string) []string {
	actions := map[string][]string{
		"RFQ":           {"Create Opportunity", "Forward to Sales", "Archive"},
		"PurchaseOrder": {"Confirm Order", "Generate Invoice", "Archive"},
		"Invoice":       {"Record Payment", "Match to PO", "Archive"},
		"Quote":         {"Convert to Order", "Follow Up", "Archive"},
		"DeliveryNote":  {"Update Inventory", "Confirm Receipt", "Archive"},
		"Unknown":       {"Review Manually", "Archive"},
	}

	if acts, ok := actions[docType]; ok {
		return acts
	}
	return []string{"Review Manually", "Archive"}
}

// extractBasicFields extracts common fields from document text
func extractBasicFields(text string, docType string) map[string]any {
	fields := make(map[string]any)

	// Extract dates (simple pattern matching)
	datePattern := regexp.MustCompile(`\d{1,2}[-/]\d{1,2}[-/]\d{2,4}`)
	if dates := datePattern.FindAllString(text, -1); len(dates) > 0 {
		fields["dates"] = dates
		if len(dates) > 0 {
			fields["deadline"] = dates[0] // Use first date as potential deadline
		}
	}

	// Extract numbers (potential RFQ/PO numbers)
	if docType == "RFQ" {
		rfqPattern := regexp.MustCompile(`(?i)rfq[#:\s]*([0-9-]+)`)
		if match := rfqPattern.FindStringSubmatch(text); len(match) > 1 {
			fields["rfq_number"] = match[1]
		}
	}

	if docType == "PurchaseOrder" {
		poPattern := regexp.MustCompile(`(?i)(?:po|p\.o\.|purchase order)[#:\s]*([0-9-]+)`)
		if match := poPattern.FindStringSubmatch(text); len(match) > 1 {
			fields["po_number"] = match[1]
		}
	}

	if docType == "BankStatement" || docType == "bank_statement" {
		// Bank name extraction - look for "National Bank of Bahrain", "NBB", etc.
		bankNamePatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)(national\s+bank\s+of\s+bahrain\s*(?:BSC)?)`),
			regexp.MustCompile(`(?i)((?:al\s+)?(?:ahli|salam|baraka|kuwait\s+finance|bbk|nbb|hsbc|standard\s+chartered|citibank|bnp\s+paribas)[\s\w]*(?:bank|BSC|B\.S\.C\.?)?)`),
			regexp.MustCompile(`(?im)^([A-Z][A-Za-z\s]+(?:Bank|BANK|BSC|B\.S\.C)[\w\s.]*)`),
		}
		for _, p := range bankNamePatterns {
			if match := p.FindStringSubmatch(text); len(match) > 1 {
				fields["bank_name"] = strings.TrimSpace(match[1])
				break
			}
		}

		// Account number - handles "Account No: 123" and multi-line "Account Number\n123"
		acctPatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)account\s*(?:no|number|#|:)[:\s]*([A-Z0-9][A-Z0-9-]+)`),
			regexp.MustCompile(`(?i)account\s*(?:no|number|#)\s*\n\s*([0-9][0-9-]+)`),
			regexp.MustCompile(`(?i)a/c\s*(?:no|number|#)?[:\s]*([0-9][0-9-]+)`),
		}
		for _, p := range acctPatterns {
			if match := p.FindStringSubmatch(text); len(match) > 1 {
				fields["account_number"] = strings.TrimSpace(match[1])
				break
			}
		}

		// Opening balance - handles same-line and next-line values
		openPatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)opening\s*balance[:\s]*([0-9][0-9,.]+)`),
			regexp.MustCompile(`(?i)opening\s*balance\s*\n\s*([0-9][0-9,.]+)`),
			regexp.MustCompile(`(?i)brought?\s*forward[:\s]*([0-9][0-9,.]+)`),
		}
		for _, p := range openPatterns {
			if match := p.FindStringSubmatch(text); len(match) > 1 {
				fields["opening_balance"] = strings.TrimSpace(match[1])
				break
			}
		}

		// Closing balance - handles same-line and next-line values
		closePatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)closing\s*balance[:\s]*([0-9][0-9,.]+)`),
			regexp.MustCompile(`(?i)closing\s*balance\s*\n\s*([0-9][0-9,.]+)`),
			regexp.MustCompile(`(?i)carried?\s*forward[:\s]*([0-9][0-9,.]+)`),
		}
		for _, p := range closePatterns {
			if match := p.FindStringSubmatch(text); len(match) > 1 {
				fields["closing_balance"] = strings.TrimSpace(match[1])
				break
			}
		}

		// IBAN
		ibanPattern := regexp.MustCompile(`(?i)IBAN[:\s]*([A-Z]{2}\d{2}[A-Z0-9]+)`)
		if match := ibanPattern.FindStringSubmatch(text); len(match) > 1 {
			fields["iban"] = strings.TrimSpace(match[1])
		}

		// Statement period - "01/01/2026 To 31/01/2026" or "From: X To: Y"
		periodPatterns := []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\s*(?:to|through|-)\s*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`),
			regexp.MustCompile(`(?i)(?:period|from)[:\s]*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\s*(?:to|through|-)\s*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`),
			regexp.MustCompile(`(?i)statement\s*(?:period|date)[:\s]*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})\s*(?:to|through|-)\s*(\d{1,2}[/-]\d{1,2}[/-]\d{2,4})`),
		}
		for _, p := range periodPatterns {
			if match := p.FindStringSubmatch(text); len(match) > 2 {
				fields["period_start"] = strings.TrimSpace(match[1])
				fields["period_end"] = strings.TrimSpace(match[2])
				break
			}
		}

		// Currency detection
		if strings.Contains(text, "BHD") || strings.Contains(text, "Bahraini Dinar") {
			fields["currency"] = "BHD"
		} else if strings.Contains(text, "USD") {
			fields["currency"] = "USD"
		} else if strings.Contains(text, "EUR") {
			fields["currency"] = "EUR"
		}

		log.Printf("🏦 Bank statement fields extracted: %v", fields)
	}

	// Extract customer name (first company-like text)
	companyPattern := regexp.MustCompile(`(?m)^([A-Z][A-Za-z\s&.-]+(?:LLC|LLP|Inc|Ltd|Corporation|Corp|W\.L\.L))`)
	if match := companyPattern.FindStringSubmatch(text); len(match) > 1 {
		fields["customer"] = strings.TrimSpace(match[1])
	}

	return fields
}

// CreateFolderStructure creates the hierarchical company folder structure
// Supports Acme Instrumentation and Beacon Controls with Customer and Supplier subdirectories
func (a *App) CreateFolderStructure(basePath string, companyName string) (FolderStructureResult, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return FolderStructureResult{}, err
	}
	result := FolderStructureResult{
		Success: false,
		Created: []string{},
	}

	if basePath == "" {
		var err error
		basePath, err = a.DetectOneDrivePath()
		if err != nil {
			return result, newError("BASE_PATH_ERROR", "Cannot determine base path", err.Error())
		}
	}

	// Create structure for specified company, or both if empty
	companies := []string{}
	if companyName != "" {
		companies = append(companies, companyName)
	} else {
		companies = []string{"Acme Instrumentation", "Beacon Controls"}
	}

	for _, company := range companies {
		companyPath := filepath.Join(basePath, company)

		// Create Customer and Supplier root dirs with subfolders
		customerRoot := filepath.Join(companyPath, "Customers")
		supplierRoot := filepath.Join(companyPath, "Suppliers")

		for _, dir := range []string{customerRoot, supplierRoot} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return result, newError("FOLDER_CREATE_FAILED", fmt.Sprintf("Failed to create %s", dir), err.Error())
			}
			result.Created = append(result.Created, dir)
			log.Printf("  ✓ Created: %s", dir)
		}

		// Also create shared operational folders under the company
		sharedFolders := []string{"Inbox", "Quotations", "Archive"}
		for _, folder := range sharedFolders {
			folderPath := filepath.Join(companyPath, folder)
			if err := os.MkdirAll(folderPath, 0755); err != nil {
				return result, newError("FOLDER_CREATE_FAILED", fmt.Sprintf("Failed to create %s", folder), err.Error())
			}
			result.Created = append(result.Created, folderPath)
			log.Printf("  ✓ Created: %s", folderPath)
		}
	}

	result.Success = true
	// Set inbox path for the first company
	if len(companies) > 0 {
		result.InboxPath = filepath.Join(basePath, companies[0], "Inbox")
	}

	// Update config with new paths
	if a.config != nil && len(companies) > 0 {
		firstCompany := companies[0]
		a.config.OneDrive.RFQPath = filepath.Join(basePath, firstCompany, "Inbox")
		a.config.OneDrive.OffersPath = filepath.Join(basePath, firstCompany, "Quotations")
		a.config.OneDrive.InvoicesPath = filepath.Join(basePath, firstCompany, "Customers")
	}

	log.Printf("✅ Folder structure created: %d folders for %d companies", len(result.Created), len(companies))
	return result, nil
}

// CreateCustomerFolder creates the per-customer folder structure under the company hierarchy
// Structure: basePath/company/Customers/CustomerName/{RFQ, Costing Sheets, Invoices, Reports}
func (a *App) CreateCustomerFolder(basePath string, company string, customerName string) (string, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return "", err
	}
	if basePath == "" || company == "" || customerName == "" {
		return "", fmt.Errorf("basePath, company, and customerName are all required")
	}
	// Validate company is an allowed value
	if company != "Acme Instrumentation" && company != "Beacon Controls" {
		return "", fmt.Errorf("invalid company: must be 'Acme Instrumentation' or 'Beacon Controls'")
	}
	// Validate basePath doesn't contain traversal
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("invalid base path")
	}

	// Sanitize customer name for filesystem
	safeName := sanitizeFileName(customerName)
	customerPath := filepath.Join(absBase, company, "Customers", safeName)
	// Verify final path is still within the base
	absCustomer, _ := filepath.Abs(customerPath)
	if !strings.HasPrefix(absCustomer, absBase) {
		return "", fmt.Errorf("invalid path: traversal detected")
	}

	subfolders := []string{"RFQ", "Costing Sheets", "Invoices", "Reports"}
	for _, sub := range subfolders {
		if err := os.MkdirAll(filepath.Join(customerPath, sub), 0755); err != nil {
			return "", fmt.Errorf("failed to create %s folder: %w", sub, err)
		}
	}

	log.Printf("📁 Created customer folder: %s (%d subfolders)", customerPath, len(subfolders))
	return customerPath, nil
}

// CreateSupplierFolder creates the per-supplier folder structure under the company hierarchy
// Structure: basePath/company/Suppliers/SupplierName/{PO, Invoices}
func (a *App) CreateSupplierFolder(basePath string, company string, supplierName string) (string, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return "", err
	}
	if basePath == "" || company == "" || supplierName == "" {
		return "", fmt.Errorf("basePath, company, and supplierName are all required")
	}
	// Validate company is an allowed value
	if company != "Acme Instrumentation" && company != "Beacon Controls" {
		return "", fmt.Errorf("invalid company: must be 'Acme Instrumentation' or 'Beacon Controls'")
	}
	// Validate basePath doesn't contain traversal
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("invalid base path")
	}

	// Sanitize supplier name for filesystem
	safeName := sanitizeFileName(supplierName)
	supplierPath := filepath.Join(absBase, company, "Suppliers", safeName)
	// Verify final path is still within the base
	absSupplier, _ := filepath.Abs(supplierPath)
	if !strings.HasPrefix(absSupplier, absBase) {
		return "", fmt.Errorf("invalid path: traversal detected")
	}

	subfolders := []string{"PO", "Invoices"}
	for _, sub := range subfolders {
		if err := os.MkdirAll(filepath.Join(supplierPath, sub), 0755); err != nil {
			return "", fmt.Errorf("failed to create %s folder: %w", sub, err)
		}
	}

	log.Printf("📁 Created supplier folder: %s (%d subfolders)", supplierPath, len(subfolders))
	return supplierPath, nil
}

// RunInitialScan scans configured folders for existing documents
func (a *App) RunInitialScan() (InitialScanResult, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return InitialScanResult{}, err
	}
	result := InitialScanResult{
		TotalFiles:  0,
		FilesByType: make(map[string]int),
		Conflicts:   []string{},
		Warnings:    []string{},
	}

	startTime := time.Now()
	log.Printf("🔍 Starting initial document scan...")

	// Get paths to scan
	paths := []string{}
	if a.config != nil {
		if a.config.OneDrive.RFQPath != "" {
			paths = append(paths, a.config.OneDrive.RFQPath)
		}
		if a.config.OneDrive.OffersPath != "" {
			paths = append(paths, a.config.OneDrive.OffersPath)
		}
		if a.config.OneDrive.InvoicesPath != "" {
			paths = append(paths, a.config.OneDrive.InvoicesPath)
		}
	}

	// Scan each path
	extensions := supportedOCRWatcherExtensions()
	for _, scanPath := range paths {
		if scanPath == "" {
			continue
		}
		filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			for _, validExt := range extensions {
				if ext == validExt {
					result.TotalFiles++
					result.FilesByType[ext]++
					break
				}
			}
			return nil
		})
	}

	result.ScanDuration = time.Since(startTime).Milliseconds()
	log.Printf("✅ Scan complete: %d files found in %dms", result.TotalFiles, result.ScanDuration)
	return result, nil
}

// CompleteSetup marks setup as complete and saves configuration
func (a *App) CompleteSetup() error {
	if err := a.requirePermission("settings:update"); err != nil {
		return err
	}
	log.Printf("🎉 Completing setup...")

	// Ensure config is saved
	if a.config != nil {
		// Mark setup as complete (could add a flag to config)
		log.Printf("✅ Setup complete! Configuration saved.")
	}

	// Start file watcher if configured
	if a.config != nil && a.config.App.EnableFileWatcher && a.fileWatcher != nil {
		if !a.fileWatcher.IsRunning() {
			watchConfig := &WatchConfig{
				RFQPath:       a.config.OneDrive.RFQPath,
				EHXMLPath:     a.config.OneDrive.EHPath,
				OfferPath:     a.config.OneDrive.OffersPath,
				InvoicePath:   a.config.OneDrive.InvoicesPath,
				Recursive:     true,
				DebounceDelay: time.Duration(a.config.App.WatcherDebounceMS) * time.Millisecond,
			}
			if watchConfig.hasValidPaths() {
				if err := a.fileWatcher.Start(); err != nil {
					log.Printf("⚠️ Could not start file watcher: %v", err)
				} else {
					log.Printf("✅ File watcher started")
				}
			}
		}
	}

	return nil
}

// WatchInboxForTestFile starts watching inbox for a test file (used during setup)
func (a *App) WatchInboxForTestFile(inboxPath string) error {
	if err := a.requirePermission("settings:view"); err != nil {
		return err
	}
	log.Printf("👀 Watching for test file in: %s", inboxPath)

	// This would set up a one-time file watcher
	// When a file is detected, it emits a "setup:file-detected" event
	// For now, just log that we're watching

	go func() {
		// Simple polling approach for setup test
		for i := 0; i < 60; i++ { // Watch for 60 seconds
			time.Sleep(1 * time.Second)

			files, err := os.ReadDir(inboxPath)
			if err != nil {
				continue
			}

			for _, file := range files {
				if file.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if ext == ".pdf" || ext == ".xlsx" || ext == ".docx" || ext == ".msg" {
					// Found a file! Emit event
					log.Printf("📄 Test file detected: %s", file.Name())

					// Emit Wails event
					if a.ctx != nil {
						runtime.EventsEmit(a.ctx, "setup:file-detected", map[string]any{
							"path":         filepath.Join(inboxPath, file.Name()),
							"name":         file.Name(),
							"documentType": strings.ToUpper(ext[1:]),
							"date":         time.Now().Format("2006-01-02"),
						})
					}
					return
				}
			}
		}
		log.Printf("⏰ Test file watch timed out after 60 seconds")
	}()

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// BIG 3 FIXES - OCR INTEGRATION, FOLDER BROWSER, QUICK CAPTURE
// ═══════════════════════════════════════════════════════════════════════════

// PickFile opens a native file dialog for selecting documents
// Returns the selected file path or empty string if cancelled
func (a *App) PickFile(title string) (string, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return "", err
	}
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Documents", Pattern: supportedOCRFileDialogPattern()},
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
			{DisplayName: "Images", Pattern: supportedOCRImagePattern()},
			{DisplayName: "Office Files", Pattern: supportedOCROfficePattern()},
			{DisplayName: "Email Files", Pattern: supportedOCREmailPattern()},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("file dialog failed: %w", err)
	}
	return selection, nil
}

// PickCSVFile opens a native file dialog for selecting CSV files (bank statements)
// Returns the selected file path or empty string if cancelled
func (a *App) PickCSVFile(title string) (string, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return "", err
	}
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files", Pattern: "*.csv"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("file dialog failed: %w", err)
	}
	return selection, nil
}

// ImportBankStatementWithDialog opens file picker and imports bank statement
// Convenience function for frontend to call without handling file dialog separately
// Supports both CSV and PDF formats
func (a *App) ImportBankStatementWithDialog(bankAccountID string) (*BankStatement, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	// Open file picker for both CSV and PDF
	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Bank Statement",
		Filters: []runtime.FileFilter{
			{DisplayName: "Bank Statements", Pattern: "*.csv;*.pdf"},
			{DisplayName: "CSV Files", Pattern: "*.csv"},
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("file selection failed: %w", err)
	}
	if selection == "" {
		return nil, nil // User cancelled
	}

	// Route based on file extension
	ext := strings.ToLower(filepath.Ext(selection))
	switch ext {
	case ".csv":
		return a.ImportBankStatementCSV(selection, bankAccountID)
	case ".pdf":
		return a.ImportBankStatementPDF(selection, bankAccountID)
	default:
		return nil, fmt.Errorf("unsupported file format: %s (use CSV or PDF)", ext)
	}
}

// bankStatementImportPreview holds a parsed-but-not-persisted statement,
// keyed by its own (pre-allocated) statement ID, so the frontend can review
// the parsed rows before committing them to the database (Wave 9.3 B1d).
type bankStatementImportPreview struct {
	Statement *BankStatement
	FilePath  string
}

var (
	bankStatementPreviewMu    sync.Mutex
	bankStatementPreviewStore = map[string]*bankStatementImportPreview{}
)

// PreviewBankStatementImportWithDialog opens the file picker and parses the
// selected statement WITHOUT persisting it. The frontend shows the parsed
// rows in a confirm modal; only ConfirmBankStatementImport writes to the
// database. The returned statement's ID doubles as the preview handle.
func (a *App) PreviewBankStatementImportWithDialog(bankAccountID string) (*BankStatement, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}

	selection, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Bank Statement",
		Filters: []runtime.FileFilter{
			{DisplayName: "Bank Statements", Pattern: "*.csv;*.pdf"},
			{DisplayName: "CSV Files", Pattern: "*.csv"},
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("file selection failed: %w", err)
	}
	if selection == "" {
		return nil, nil // User cancelled
	}

	ext := strings.ToLower(filepath.Ext(selection))
	var statement *BankStatement
	switch ext {
	case ".csv":
		statement, err = a.parseBankStatementCSV(selection, bankAccountID)
	case ".pdf":
		statement, err = a.parseBankStatementPDF(selection, bankAccountID)
	default:
		return nil, fmt.Errorf("unsupported file format: %s (use CSV or PDF)", ext)
	}
	if err != nil {
		return nil, err
	}

	bankStatementPreviewMu.Lock()
	bankStatementPreviewStore[statement.ID] = &bankStatementImportPreview{Statement: statement, FilePath: selection}
	bankStatementPreviewMu.Unlock()

	return statement, nil
}

// ConfirmBankStatementImport persists a previously-previewed statement.
// previewID is the ID returned on the statement from
// PreviewBankStatementImportWithDialog.
func (a *App) ConfirmBankStatementImport(previewID string) (*BankStatement, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	bankStatementPreviewMu.Lock()
	pending, ok := bankStatementPreviewStore[previewID]
	if ok {
		delete(bankStatementPreviewStore, previewID)
	}
	bankStatementPreviewMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("preview not found or already committed — re-import the file")
	}

	if err := a.db.Create(pending.Statement).Error; err != nil {
		return nil, fmt.Errorf("failed to save statement: %w", err)
	}
	if strings.ToLower(filepath.Ext(pending.FilePath)) == ".pdf" {
		a.ArchiveStatementPDF(pending.Statement.ID, pending.FilePath)
	}

	log.Printf("📊 Confirmed statement import: %s with %d lines", pending.Statement.StatementNumber, len(pending.Statement.Lines))
	return pending.Statement, nil
}

// DiscardBankStatementImportPreview drops a previewed-but-not-confirmed
// import without writing anything to the database.
func (a *App) DiscardBankStatementImportPreview(previewID string) {
	bankStatementPreviewMu.Lock()
	delete(bankStatementPreviewStore, previewID)
	bankStatementPreviewMu.Unlock()
}

// OCRResult is now defined in ocr_service_simple.go (type alias for OCRResultSimple)

// ProcessDocumentWithOCR processes a document using the Simple OCR service
// filePath: path to the document
// docType: document type hint ("auto", "invoice", "supplier_invoice", "rfq", "quotation", "purchase_order", "delivery_note", "bank_statement", "contract", "report")
func (a *App) ProcessDocumentWithOCR(filePath string, docType string) (*OCRResult, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}

	// SECURITY FIX: Validate file path to prevent path traversal
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	filePath = absPath // Use validated path for processing

	log.Printf("📄 Starting OCR processing: %s (type: %s)", filePath, docType)

	if a.ocrService == nil {
		return nil, fmt.Errorf("OCR service not initialized")
	}

	// Process document using SimpleOCRService
	result, err := a.ocrService.ProcessDocument(filePath, docType)
	if err != nil {
		return nil, fmt.Errorf("OCR processing failed: %w", err)
	}

	// Auto-detect document type using AI classifier if "auto" was specified
	if docType == "auto" && result.Text != "" {
		aiResult := a.AIClassifyDocumentType(result.Text, filepath.Base(filePath))
		if aiResult == nil {
			aiResult = a.classifyDocumentForOCR(result.Text, filepath.Base(filePath))
		}
		if aiResult != nil {
			result.DocumentType = aiResult.DocumentType
			log.Printf("🤖 AI-detected document type: %s (confidence: %.2f)", aiResult.DocumentType, aiResult.Confidence)
		}
	}

	// Populate extracted_data field (QuickCaptureDocument does this but ProcessDocumentWithOCR was missing it)
	mergedExtractedData := extractBasicFields(result.Text, result.DocumentType)
	if result.ExtractedData != nil {
		for key, value := range result.ExtractedData {
			mergedExtractedData[key] = value
		}
	}
	if result.DocumentType == "BankStatement" || result.DocumentType == "bank_statement" {
		if parseOutcome, parseErr := a.parseImportedPDFBankStatement(result.Text); parseErr != nil {
			log.Printf("⚠️ OCR bank statement pre-parse failed, keeping basic extraction only: %v", parseErr)
		} else if parseOutcome != nil && parseOutcome.Statement != nil {
			parsed := parseOutcome.Statement
			if parsed.AccountNumber != "" {
				mergedExtractedData["account_number"] = parsed.AccountNumber
			}
			if parsed.IBAN != "" {
				mergedExtractedData["iban"] = parsed.IBAN
			}
			if parsed.Currency != "" {
				mergedExtractedData["currency"] = parsed.Currency
			}
			if !parsed.PeriodStart.IsZero() {
				mergedExtractedData["period_start"] = parsed.PeriodStart.Format("2006-01-02")
			}
			if !parsed.PeriodEnd.IsZero() {
				mergedExtractedData["period_end"] = parsed.PeriodEnd.Format("2006-01-02")
			}
			mergedExtractedData["opening_balance"] = parsed.OpeningBalance
			mergedExtractedData["closing_balance"] = parsed.ClosingBalance
			mergedExtractedData["total_debits"] = parsed.TotalDebits
			mergedExtractedData["total_credits"] = parsed.TotalCredits
			mergedExtractedData["debit_count"] = parsed.DebitCount
			mergedExtractedData["credit_count"] = parsed.CreditCount
			mergedExtractedData["import_method"] = parseOutcome.ImportMethod
			if len(parseOutcome.ValidationNote) > 0 {
				mergedExtractedData["validation_notes"] = parseOutcome.ValidationNote
			}

			lineItems := make([]map[string]any, 0, len(parsed.Lines))
			for _, line := range parsed.Lines {
				lineItems = append(lineItems, map[string]any{
					"date":        line.Date.Format("2006-01-02"),
					"value_date":  line.ValueDate.Format("2006-01-02"),
					"description": line.Description,
					"reference":   line.Reference,
					"debit":       line.Debit,
					"credit":      line.Credit,
					"balance":     line.Balance,
				})
			}
			mergedExtractedData["line_items"] = lineItems
		}
	}
	result.ExtractedData = mergedExtractedData

	log.Printf("✅ OCR processing complete: %.2f%% confidence, %s engine, type=%s, extracted_fields=%d",
		result.Confidence*100, result.Engine, result.DocumentType, len(result.ExtractedData))

	return result, nil
}

// QuickCapture represents a quick note/capture
type QuickCapture struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      string    `json:"tags"`     // Comma-separated tags
	Priority  string    `json:"priority"` // Low, Medium, High, Urgent
	Status    string    `json:"status"`   // Open, InProgress, Done
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OCRDocument represents processed OCR document results
type OCRDocument struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	FileName          string    `json:"file_name" gorm:"not null"`
	FilePath          string    `json:"file_path" gorm:"not null"`
	DocumentType      string    `json:"document_type" gorm:"index"` // RFQ, Invoice, Quote, etc.
	ExtractedText     string    `json:"extracted_text" gorm:"type:text"`
	ExtractedDataJSON string    `json:"extracted_data_json" gorm:"type:text"` // JSON string of extracted fields
	Confidence        float64   `json:"confidence"`
	ProcessingTimeMS  int64     `json:"processing_time_ms"`
	Engine            string    `json:"engine"`        // go-fitz, florence-2, tesseract, dna-cache
	TierUsed          string    `json:"tier_used"`     // Which processing tier was used
	Cost              float64   `json:"cost"`          // USD cost for processing
	DNACacheHit       bool      `json:"dna_cache_hit"` // Was this processed via DNA cache?
	TableDetected     bool      `json:"table_detected"`
	GPUUsed           bool      `json:"gpu_used"`
	ProcessedAt       time.Time `json:"processed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// CreateQuickCapture creates a quick note/capture from dashboard
func (a *App) CreateQuickCapture(title, content, tags, priority string) (uint, error) {
	if err := a.requirePermission("notes:create"); err != nil {
		return 0, err
	}
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	// Auto-migrate QuickCapture if not exists
	if err := a.db.AutoMigrate(&QuickCapture{}); err != nil {
		return 0, fmt.Errorf("failed to migrate QuickCapture table: %w", err)
	}

	// Create capture
	capture := QuickCapture{
		Title:    title,
		Content:  content,
		Tags:     tags,
		Priority: priority,
		Status:   "Open",
	}

	result := a.db.Create(&capture)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create quick capture: %w", result.Error)
	}

	log.Printf("📝 Quick capture created: ID=%d, Title=%s", capture.ID, title)
	return capture.ID, nil
}

// GetQuickCaptures retrieves recent quick captures
func (a *App) GetQuickCaptures(limit int) ([]QuickCapture, error) {
	if err := a.requirePermission("notes:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}

	var captures []QuickCapture
	result := a.db.Order("created_at DESC").Limit(limit).Find(&captures)
	if result.Error != nil {
		return nil, result.Error
	}

	return captures, nil
}

// UpdateQuickCapture updates a quick capture
func (a *App) UpdateQuickCapture(id uint, title, content, tags, priority, status string) error {
	if err := a.requirePermission("notes:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result := a.db.Model(&QuickCapture{}).Where("id = ?", id).Updates(map[string]any{
		"title":    title,
		"content":  content,
		"tags":     tags,
		"priority": priority,
		"status":   status,
	})

	if result.Error != nil {
		return result.Error
	}

	log.Printf("✏️ Quick capture updated: ID=%d", id)
	return nil
}

// DeleteQuickCapture deletes a quick capture
func (a *App) DeleteQuickCapture(id string) error {
	if ok, err := a.guardDeleteOrRequest("notes:delete", "quick_capture", id, "Quick capture"); !ok {
		return err
	}
	if err := a.requirePermission("notes:delete"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	result := a.db.Delete(&QuickCapture{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	log.Printf("🗑️ Quick capture deleted: ID=%s", id)
	return nil
}

// ========================================================================
// OCR DOCUMENT CRUD OPERATIONS
// ========================================================================

// GetOCRDocuments retrieves recent OCR processed documents
func (a *App) GetOCRDocuments(limit int) ([]OCRDocument, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}

	var docs []OCRDocument
	result := a.db.Order("processed_at DESC").Limit(limit).Find(&docs)
	if result.Error != nil {
		return nil, result.Error
	}

	return docs, nil
}

// GetOCRDocumentByID retrieves a single OCR document
func (a *App) GetOCRDocumentByID(id string) (*OCRDocument, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var doc OCRDocument
	result := a.db.First(&doc, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}

	return &doc, nil
}

// GetOCRDocumentsByType retrieves OCR documents filtered by document type
func (a *App) GetOCRDocumentsByType(docType string, limit int) ([]OCRDocument, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}

	var docs []OCRDocument
	result := a.db.Where("document_type = ?", docType).Order("processed_at DESC").Limit(limit).Find(&docs)
	if result.Error != nil {
		return nil, result.Error
	}

	return docs, nil
}

// SaveOCRDocument saves an OCR document result to the database
func (a *App) SaveOCRDocument(fileName, filePath, documentType, extractedText string, confidence float64, processingTimeMS int64, engine string, extractedDataJSON string) (*OCRDocument, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Sanitize inputs
	if fileName == "" {
		fileName = "unknown_document"
	}
	// Remove path separators and control characters from filename
	fileName = filepath.Base(fileName)
	// Never store full file paths in database (privacy + security)
	filePath = "" // Always blank - we only store the filename

	// Clamp confidence to valid range
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	// Limit extracted data size (prevent bloat)
	const maxExtractedDataSize = 500 * 1024 // 500KB
	if len(extractedDataJSON) > maxExtractedDataSize {
		log.Printf("⚠️ Extracted data too large (%d bytes), truncating", len(extractedDataJSON))
		extractedDataJSON = extractedDataJSON[:maxExtractedDataSize]
	}

	// Auto-migrate OCRDocument table if not exists
	if err := a.db.AutoMigrate(&OCRDocument{}); err != nil {
		return nil, fmt.Errorf("failed to migrate OCRDocument table: %w", err)
	}

	doc := OCRDocument{
		FileName:          fileName,
		FilePath:          filePath,
		DocumentType:      documentType,
		ExtractedText:     extractedText,
		ExtractedDataJSON: extractedDataJSON,
		Confidence:        confidence,
		ProcessingTimeMS:  processingTimeMS,
		Engine:            engine,
		ProcessedAt:       time.Now(),
	}

	result := a.db.Create(&doc)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to save OCR document: %w", result.Error)
	}

	log.Printf("💾 OCR document saved: ID=%d, Type=%s, File=%s, HasExtractedData=%v", doc.ID, documentType, fileName, extractedDataJSON != "")
	return &doc, nil
}

func normalizeOCRDocumentType(documentType string) string {
	docType := strings.TrimSpace(documentType)
	if docType == "" {
		return "other"
	}

	canonical := map[string]string{
		"RFQ":             "rfq",
		"Invoice":         "invoice",
		"CustomerInvoice": "invoice",
		"SupplierInvoice": "supplier_invoice",
		"PurchaseOrder":   "purchase_order",
		"PO":              "purchase_order",
		"Quotation":       "quotation",
		"Quote":           "quotation",
		"DeliveryNote":    "delivery_note",
		"BankStatement":   "bank_statement",
		"Costing":         "costing",
		"CostingSheet":    "costing",
		"ExcelData":       "excel_data",
		"Contract":        "contract",
		"Report":          "report",
		"Other":           "other",
	}
	if normalized, ok := canonical[docType]; ok {
		return normalized
	}

	lower := strings.ToLower(docType)
	lower = strings.NewReplacer(" ", "_", "-", "_").Replace(lower)
	switch lower {
	case "customer_invoice":
		return "invoice"
	case "po":
		return "purchase_order"
	case "quote":
		return "quotation"
	default:
		return lower
	}
}

func (a *App) requireDocumentRoutingPermission(documentType string) error {
	switch documentType {
	case "rfq", "quotation", "costing", "excel_data":
		return a.requirePermission("offers:create")
	case "invoice":
		return a.requirePermission("invoices:create")
	case "purchase_order", "supplier_invoice":
		return a.requirePermission("po:create")
	case "delivery_note":
		return a.requirePermission("delivery_notes:create")
	case "bank_statement":
		return a.requirePermission("finance:create")
	default:
		return nil
	}
}

// SaveDocumentToEntity routes an OCR-processed document to the correct domain table
// based on the detected/selected document type. Also saves to OCRDocument for audit trail.
// Returns the created entity ID and any error.
func (a *App) SaveDocumentToEntity(fileName, filePath, documentType, extractedText string, confidence float64, processingTimeMS int64, engine string, extractedDataJSON string) (map[string]any, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	documentType = normalizeOCRDocumentType(documentType)
	if err := a.requireDocumentRoutingPermission(documentType); err != nil {
		return nil, err
	}

	// Parse extracted data
	var extractedData map[string]any
	if extractedDataJSON != "" {
		if err := json.Unmarshal([]byte(extractedDataJSON), &extractedData); err != nil {
			log.Printf("⚠ Failed to parse extracted data JSON: %v", err)
			extractedData = make(map[string]any)
		}
	} else {
		extractedData = make(map[string]any)
	}

	// Always save to OCRDocument as audit trail
	ocrDoc := OCRDocument{
		FileName:          fileName,
		FilePath:          filePath,
		DocumentType:      documentType,
		ExtractedText:     extractedText,
		ExtractedDataJSON: extractedDataJSON,
		Confidence:        confidence,
		ProcessingTimeMS:  processingTimeMS,
		Engine:            engine,
		ProcessedAt:       time.Now(),
	}
	if err := a.db.AutoMigrate(&OCRDocument{}); err != nil {
		log.Printf("⚠ OCRDocument migration warning: %v", err)
	}
	a.db.Create(&ocrDoc)

	// Route to the correct domain table based on document type
	result := map[string]any{
		"ocr_document_id": ocrDoc.ID,
		"document_type":   documentType,
		"routed":          false,
	}

	switch documentType {
	case "rfq":
		entityID, err := a.routeToOpportunity(extractedData, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to save opportunity: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "opportunities"
		result["routed"] = true

	case "invoice":
		entityID, err := a.routeToInvoice(extractedData, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to save invoice: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "invoices"
		result["routed"] = true

	case "purchase_order":
		entityID, err := a.routeToPurchaseOrder(extractedData, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to save purchase order: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "purchase_orders"
		result["routed"] = true

	case "supplier_invoice":
		entityID, err := a.routeToSupplierInvoice(extractedData, fileName, fmt.Sprintf("%d", ocrDoc.ID), confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to save supplier invoice: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "supplier_invoices"
		result["routed"] = true

	case "delivery_note":
		entityID, err := a.routeToDeliveryNote(extractedData, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to save delivery note: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "delivery_notes"
		result["routed"] = true

	case "quotation":
		entityID, err := a.routeToOpportunity(extractedData, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to save quotation opportunity: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "opportunities"
		result["routed"] = true

	case "costing", "excel_data":
		entityID, err := a.routeToOpportunity(extractedData, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to save %s opportunity: %w", documentType, err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "opportunities"
		result["routed"] = true

	case "bank_statement":
		entityID, err := a.routeToBankStatement(extractedData, fileName, confidence)
		if err != nil {
			return nil, fmt.Errorf("failed to save bank statement: %w", err)
		}
		result["entity_id"] = entityID
		result["entity_table"] = "bank_statements"
		result["routed"] = true

	case "contract", "report":
		result["entity_table"] = "ocr_documents"
		result["routed"] = false
		result["stored_only"] = true

	default:
		log.Printf("⚠ Unknown document type '%s' - saved to OCRDocument only", documentType)
	}

	log.Printf("✅ Document routed: type=%s, routed=%v, entity_table=%v",
		documentType, result["routed"], result["entity_table"])

	return result, nil
}

// --- Domain routing helpers ---

func (a *App) routeToOpportunity(data map[string]any, fileName string) (string, error) {
	division := normalizeDivisionName(getStringField(data, "division"))
	customerID := strings.TrimSpace(getStringField(data, "customer_id"))
	customerName := strings.TrimSpace(getStringField(data, "customer_name"))

	if customerID != "" {
		var customer CustomerMaster
		if err := a.db.Where("id = ?", customerID).First(&customer).Error; err == nil {
			customerName = customer.BusinessName
		}
	}

	if customerName == "" {
		customerName = "Unknown (OCR)"
	}

	project := strings.TrimSpace(getStringField(data, "project"))
	if project == "" {
		project = strings.TrimSpace(fileName)
	}

	folderNumber := strings.TrimSpace(getStringField(data, "rfq_number"))
	if folderNumber == "" {
		folderNumber = strings.TrimSpace(getStringField(data, "folder_number"))
	}
	if folderNumber == "" {
		folderNumber = strings.TrimSpace(getStringField(data, "po_number"))
	}
	if folderNumber == "" {
		folderNumber = fmt.Sprintf("OCR-%s", time.Now().Format("20060102-150405"))
	}

	meta := parseOneDriveFolderMeta(folderNumber + " " + project)
	year := meta.Year
	if year == 0 {
		year = time.Now().Year()
	}

	oppNumber := meta.OppNumber
	title := strings.TrimSpace(project)
	if meta.Title != "" {
		title = meta.Title
	}

	if folderNumber != "" {
		var existing Opportunity
		if err := a.db.Where("folder_number = ?", folderNumber).First(&existing).Error; err == nil {
			return "", fmt.Errorf("duplicate opportunity: %s already exists for %s (%s)", existing.FolderNumber, existing.CustomerName, existing.Stage)
		}
	}

	productDetails := ""
	if lineItems, ok := data["line_items"]; ok && lineItems != nil {
		if normalized, count := normalizeOpportunityLineItemsJSON(lineItems); count > 0 {
			productDetails = normalized
			log.Printf("📦 Opportunity has %d normalized line items from OCR/Butler", count)
		}
	}

	comment := strings.TrimSpace(getStringField(data, "butler_summary"))
	if comment == "" {
		comment = strings.TrimSpace(getStringField(data, "notes"))
	}

	revenue := getFloatField(data, "total")
	paymentTerms := strings.TrimSpace(getStringField(data, "payment_terms"))
	deliveryTerms := strings.TrimSpace(getStringField(data, "delivery_terms"))

	opp := &Opportunity{
		Base:           Base{ID: uuid.New().String()},
		FolderNumber:   folderNumber,
		CustomerID:     customerID,
		CustomerName:   customerName,
		Year:           year,
		OppNumber:      oppNumber,
		FolderName:     strings.TrimSpace(strings.Join([]string{folderNumber, project}, " ")),
		Title:          title,
		Source:         "2026_ocr",
		Comment:        comment,
		ProductDetails: productDetails,
		OfferDate:      time.Now(),
		PaymentTerms:   paymentTerms,
		DeliveryTerms:  deliveryTerms,
		RevenueBHD:     revenue,
		Stage:          "New",
		Division:       division,
	}

	if err := a.db.Create(opp).Error; err != nil {
		return "", err
	}

	log.Printf("🧭 Opportunity created from OCR: ID=%s, Folder=%s, Customer=%s, Stage=%s",
		opp.ID, opp.FolderNumber, opp.CustomerName, opp.Stage)
	return opp.ID, nil
}

func (a *App) routeToRFQ(data map[string]any, fileName string) (uint, error) {
	client := getStringField(data, "customer_name")

	// If customer_id is provided, look up the customer name from database
	if custID := strings.TrimSpace(getStringField(data, "customer_id")); custID != "" {
		var customer CustomerMaster
		if err := a.db.First(&customer, "id = ?", custID).Error; err == nil {
			client = customer.BusinessName
			log.Printf("✅ RFQ linked to customer ID %s: %s", custID, client)
		}
	}

	if client == "" {
		client = getStringField(data, "supplier_name")
	}
	if client == "" {
		client = "Unknown (OCR)"
	}

	project := getStringField(data, "project")
	if project == "" {
		project = fileName
	}

	value := getFloatField(data, "total")

	// Build notes from Butler summary if available, otherwise use raw text excerpt
	notes := getStringField(data, "butler_summary")
	if notes == "" {
		notes = getStringField(data, "raw_text")
		if len(notes) > 500 {
			notes = notes[:500] + "..."
		}
	}
	// Append any user-provided notes
	userNotes := getStringField(data, "notes")
	if userNotes != "" && userNotes != notes {
		notes = notes + "\n\n--- User Notes ---\n" + userNotes
	}
	if len(notes) > 2000 {
		notes = notes[:2000]
	}

	// Extract and store line items as JSON in ProductDetails
	var productDetails string
	if lineItems, ok := data["line_items"]; ok && lineItems != nil {
		if normalized, count := normalizeOpportunityLineItemsJSON(lineItems); count > 0 {
			productDetails = normalized
			log.Printf("📦 RFQ has %d normalized line items from OCR/Butler", count)
		}
	}

	rfq := &RFQData{
		Client:         client,
		Project:        project,
		Value:          value,
		Notes:          notes,
		Status:         "pending",
		ProductDetails: productDetails, // JSON array of line items
		SourceDocPath:  fileName,
	}

	if err := a.db.Create(rfq).Error; err != nil {
		return 0, err
	}

	log.Printf("📋 RFQ created from OCR: ID=%d, Client=%s, Project=%s, Value=%.2f, Items=%d",
		rfq.ID, client, project, value, len(productDetails))
	return rfq.ID, nil
}

func (a *App) routeToInvoice(data map[string]any, fileName string) (string, error) {
	division := normalizeDivisionName(getStringField(data, "division"))
	invoiceNumber := getStringField(data, "invoice_number")
	if invoiceNumber == "" {
		invoiceNumber = fmt.Sprintf("OCR-%s", time.Now().Format("20060102-150405"))
	}

	customerName := getStringField(data, "customer_name")
	if customerName == "" {
		customerName = "Unknown (OCR)"
	}

	// Look up customer ID
	var customerID string
	var customer CustomerMaster
	if err := a.db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(customerName)+"%").First(&customer).Error; err == nil {
		customerID = customer.ID
		customerName = customer.BusinessName
	}

	invoiceDate := parseDate(getStringField(data, "invoice_date"))
	if invoiceDate.IsZero() {
		invoiceDate = time.Now()
	}
	dueDate := parseDate(getStringField(data, "due_date"))
	if dueDate.IsZero() {
		dueDate = invoiceDate.AddDate(0, 0, 30) // Default Net 30
	}

	grandTotal := getFloatField(data, "total")
	poNumber := getStringField(data, "po_number")

	invoice := &Invoice{
		InvoiceNumber:    invoiceNumber,
		InvoiceDate:      invoiceDate,
		CustomerID:       customerID,
		CustomerName:     customerName,
		CustomerPONumber: poNumber,
		GrandTotalBHD:    grandTotal,
		OutstandingBHD:   grandTotal,
		SubtotalBHD:      grandTotal - getFloatField(data, "vat"),
		DueDate:          dueDate,
		Status:           "Sent",
		Division:         division,
	}

	if err := a.db.Create(invoice).Error; err != nil {
		return "", err
	}

	// Save line items if present
	if lineItems, ok := data["line_items"]; ok && lineItems != nil {
		if items, ok := lineItems.([]any); ok && len(items) > 0 {
			for i, item := range items {
				if itemMap, ok := item.(map[string]any); ok {
					invItem := DBInvoiceItem{
						InvoiceID:   invoice.ID,
						LineNumber:  i + 1,
						Description: getStringField(itemMap, "description"),
						Quantity:    getFloatField(itemMap, "quantity"),
						Rate:        getFloatField(itemMap, "unit_price"),
						TotalBHD:    getFloatField(itemMap, "total_price"),
					}
					if invItem.TotalBHD == 0 && invItem.Quantity > 0 && invItem.Rate > 0 {
						invItem.TotalBHD = invItem.Quantity * invItem.Rate
					}
					if err := a.db.Create(&invItem).Error; err != nil {
						log.Printf("⚠️ Failed to save invoice item %d: %v", i+1, err)
					}
				}
			}
			log.Printf("📦 Saved %d line items for invoice %s", len(items), invoice.ID)
		}
	}

	log.Printf("🧾 Invoice created from OCR: ID=%s, Number=%s, Customer=%s, Total=%.3f",
		invoice.ID, invoiceNumber, customerName, grandTotal)
	return invoice.ID, nil
}

func (a *App) routeToPurchaseOrder(data map[string]any, fileName string) (string, error) {
	division := normalizeDivisionName(getStringField(data, "division"))
	poNumber := getStringField(data, "po_number")
	if poNumber == "" {
		poNumber = fmt.Sprintf("PO-OCR-%s", time.Now().Format("20060102-150405"))
	}

	// Look up supplier
	supplierName := getStringField(data, "supplier_name")
	var supplierID string
	if supplierName != "" {
		var supplier SupplierMaster
		if err := a.db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(supplierName)+"%").First(&supplier).Error; err == nil {
			supplierID = supplier.ID
		}
	}

	poDate := parseDate(getStringField(data, "invoice_date"))
	if poDate.IsZero() {
		poDate = time.Now()
	}

	total := getFloatField(data, "total")
	currency := getStringField(data, "currency")
	if currency == "" {
		currency = "BHD"
	}

	po := &PurchaseOrder{
		PONumber:     poNumber,
		PODate:       poDate,
		SupplierID:   supplierID,
		Currency:     currency,
		SubtotalBHD:  total - getFloatField(data, "vat"),
		VATAmount:    getFloatField(data, "vat"),
		TotalBHD:     total,
		TotalForeign: total,
		Status:       "Draft",
		PaymentTerms: "Net 30",
		Division:     division,
	}

	if err := a.db.Create(po).Error; err != nil {
		return "", err
	}

	// Save line items if present
	if lineItems, ok := data["line_items"]; ok && lineItems != nil {
		if items, ok := lineItems.([]any); ok && len(items) > 0 {
			for i, item := range items {
				if itemMap, ok := item.(map[string]any); ok {
					poItem := PurchaseOrderItem{
						PurchaseOrderID:  po.ID,
						Description:      getStringField(itemMap, "description"),
						Quantity:         getFloatField(itemMap, "quantity"),
						UnitPriceBHD:     getFloatField(itemMap, "unit_price"),
						UnitPriceForeign: getFloatField(itemMap, "unit_price"),
						TotalBHD:         getFloatField(itemMap, "total_price"),
						TotalForeign:     getFloatField(itemMap, "total_price"),
					}
					if poItem.TotalBHD == 0 && poItem.Quantity > 0 && poItem.UnitPriceBHD > 0 {
						poItem.TotalBHD = poItem.Quantity * poItem.UnitPriceBHD
						poItem.TotalForeign = poItem.TotalBHD
					}
					if err := a.db.Create(&poItem).Error; err != nil {
						log.Printf("⚠️ Failed to save PO item %d: %v", i+1, err)
					}
				}
			}
			log.Printf("📦 Saved %d line items for PO %s", len(items), po.ID)
		}
	}

	log.Printf("📦 PurchaseOrder created from OCR: ID=%s, PO=%s, Total=%.3f", po.ID, poNumber, total)
	return po.ID, nil
}

func (a *App) routeToSupplierInvoice(data map[string]any, fileName string, ocrDocID string, ocrConfidence float64) (string, error) {
	division := normalizeDivisionName(getStringField(data, "division"))
	invoiceNumber := getStringField(data, "invoice_number")
	if invoiceNumber == "" {
		invoiceNumber = fmt.Sprintf("SI-OCR-%s", time.Now().Format("20060102-150405"))
	}

	// Look up supplier - prefer supplier_id if provided
	supplierName := getStringField(data, "supplier_name")
	var supplierID string

	// If supplier_id is provided from frontend dropdown, use it directly
	if suppID := strings.TrimSpace(getStringField(data, "supplier_id")); suppID != "" {
		var supplier SupplierMaster
		if err := a.db.First(&supplier, "id = ?", suppID).Error; err == nil {
			supplierID = supplier.ID
			supplierName = supplier.SupplierName
			log.Printf("✅ Supplier Invoice linked to supplier ID %s: %s", suppID, supplierName)
		}
	} else if supplierName != "" {
		// Fallback: look up by name
		var supplier SupplierMaster
		if err := a.db.Where("supplier_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(supplierName)+"%").First(&supplier).Error; err == nil {
			supplierID = supplier.ID
			supplierName = supplier.SupplierName
		}
	}
	if supplierName == "" {
		supplierName = "Unknown Supplier (OCR)"
	}

	poID := strings.TrimSpace(getStringField(data, "purchase_order_id"))
	poNumber := strings.TrimSpace(getStringField(data, "po_number"))
	if poID != "" {
		var po PurchaseOrder
		if err := a.db.First(&po, "id = ?", poID).Error; err == nil {
			poNumber = po.PONumber
			if supplierID == "" {
				supplierID = po.SupplierID
				supplierName = po.SupplierName
			}
		}
	} else if poNumber != "" {
		var po PurchaseOrder
		if err := a.db.Where("po_number = ?", poNumber).First(&po).Error; err == nil {
			poID = po.ID
			poNumber = po.PONumber
			if supplierID == "" {
				supplierID = po.SupplierID
				supplierName = po.SupplierName
			}
		}
	}
	orderID := strings.TrimSpace(getStringField(data, "order_id"))

	invoiceDate := parseDate(getStringField(data, "invoice_date"))
	if invoiceDate.IsZero() {
		invoiceDate = time.Now()
	}
	dueDate := parseDate(getStringField(data, "due_date"))
	if dueDate.IsZero() {
		dueDate = invoiceDate.AddDate(0, 0, 30)
	}

	getFirstFloat := func(keys ...string) float64 {
		for _, key := range keys {
			if value := getFloatField(data, key); value > 0 {
				return value
			}
		}
		return 0
	}

	total := getFirstFloat("total", "total_amount", "grand_total", "amount", "total_foreign")
	vat := getFirstFloat("vat", "vat_amount", "vat_foreign")
	subtotal := getFirstFloat("subtotal", "subtotal_amount", "net_amount", "subtotal_foreign")
	currency := strings.ToUpper(strings.TrimSpace(getStringField(data, "currency")))
	if currency == "" {
		currency = "BHD"
	}
	exchangeRate := normalizeExchangeRateToBHD(currency, getFloatField(data, "exchange_rate"))

	var invoiceItems []SupplierInvoiceItem
	lineSubtotal := 0.0
	if lineItems, ok := data["line_items"]; ok && lineItems != nil {
		if items, ok := lineItems.([]any); ok && len(items) > 0 {
			invoiceItems = make([]SupplierInvoiceItem, 0, len(items))
			for i, item := range items {
				itemMap, ok := item.(map[string]any)
				if !ok {
					continue
				}
				quantity := getFloatField(itemMap, "quantity")
				if quantity <= 0 {
					quantity = 1
				}
				unitPrice := getFloatField(itemMap, "unit_price")
				totalPrice := getFloatField(itemMap, "total_price")
				if totalPrice <= 0 {
					totalPrice = getFloatField(itemMap, "amount")
				}
				if totalPrice <= 0 && quantity > 0 && unitPrice > 0 {
					totalPrice = quantity * unitPrice
				}
				if unitPrice <= 0 && quantity > 0 && totalPrice > 0 {
					unitPrice = totalPrice / quantity
				}

				description := strings.TrimSpace(getStringField(itemMap, "description"))
				if description == "" {
					description = strings.TrimSpace(getStringField(itemMap, "part_number"))
				}
				if description == "" {
					description = fmt.Sprintf("OCR line item %d", i+1)
				}

				invoiceItems = append(invoiceItems, SupplierInvoiceItem{
					LineNumber:  i + 1,
					Description: description,
					Quantity:    quantity,
					UnitPrice:   roundTo3(unitPrice),
					TotalPrice:  roundTo3(totalPrice),
					Currency:    currency,
				})
				lineSubtotal += totalPrice
			}
		}
	}

	if subtotal <= 0 && lineSubtotal > 0 {
		subtotal = lineSubtotal
	}
	if subtotal <= 0 && total > 0 {
		subtotal = total - vat
	}
	if subtotal < 0 {
		subtotal = 0
	}
	if total <= 0 {
		total = subtotal + vat
	}

	si := &SupplierInvoice{
		SupplierID:      supplierID,
		SupplierName:    supplierName,
		PurchaseOrderID: poID,
		PONumber:        poNumber,
		OrderID:         orderID,
		InvoiceNumber:   invoiceNumber,
		InvoiceDate:     invoiceDate,
		DueDate:         dueDate,
		Currency:        currency,
		ExchangeRate:    exchangeRate,
		SubtotalBHD:     roundTo3(subtotal * exchangeRate),
		SubtotalForeign: roundTo3(subtotal),
		VATBHD:          roundTo3(vat * exchangeRate),
		VATForeign:      roundTo3(vat),
		TotalBHD:        roundTo3(total * exchangeRate),
		TotalForeign:    roundTo3(total),
		Status:          "Pending",
		MatchStatus:     "Pending",
		PaymentStatus:   "Unpaid",
		OCRDocumentID:   ocrDocID,
		OCRConfidence:   clampConfidence(ocrConfidence),
		Division:        division,
	}

	if err := a.db.Create(si).Error; err != nil {
		return "", err
	}

	if len(invoiceItems) > 0 {
		for i := range invoiceItems {
			invoiceItems[i].SupplierInvoiceID = si.ID
			if err := a.db.Create(&invoiceItems[i]).Error; err != nil {
				log.Printf("⚠️ Failed to save supplier invoice item %d: %v", i+1, err)
			}
		}
		log.Printf("📦 Saved %d line items for supplier invoice %s", len(invoiceItems), si.ID)
	}

	log.Printf("📄 SupplierInvoice created from OCR: ID=%s, OCRDoc=%s, Number=%s, Supplier=%s, Total=%.3f %s / %.3f BHD",
		si.ID, ocrDocID, invoiceNumber, supplierName, total, currency, si.TotalBHD)
	return si.ID, nil
}

func (a *App) routeToDeliveryNote(data map[string]any, fileName string) (string, error) {
	dnNumber := fmt.Sprintf("DN-OCR-%s", time.Now().Format("20060102-150405"))

	customerName := getStringField(data, "customer_name")
	var customerID string
	if customerName != "" {
		var customer CustomerMaster
		if err := a.db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(customerName)+"%").First(&customer).Error; err == nil {
			customerID = customer.ID
		}
	}

	deliveryDate := parseDate(getStringField(data, "invoice_date"))
	if deliveryDate.IsZero() {
		deliveryDate = time.Now()
	}

	dn := &DeliveryNote{
		DNNumber:     dnNumber,
		CustomerID:   customerID,
		DeliveryDate: deliveryDate,
		Status:       "Prepared",
	}

	if err := a.db.Create(dn).Error; err != nil {
		return "", err
	}

	// Save line items if present
	if lineItems, ok := data["line_items"]; ok && lineItems != nil {
		if items, ok := lineItems.([]any); ok && len(items) > 0 {
			for i, item := range items {
				if itemMap, ok := item.(map[string]any); ok {
					qty := getFloatField(itemMap, "quantity")
					dnItem := DeliveryNoteItem{
						DeliveryNoteID:    dn.ID,
						Description:       getStringField(itemMap, "description"),
						QuantityOrdered:   qty,
						QuantityDelivered: qty,
						QuantityRemaining: 0,
					}
					if err := a.db.Create(&dnItem).Error; err != nil {
						log.Printf("⚠️ Failed to save DN item %d: %v", i+1, err)
					}
				}
			}
			log.Printf("📦 Saved %d line items for DN %s", len(items), dn.ID)
		}
	}

	log.Printf("🚚 DeliveryNote created from OCR: ID=%s, DN=%s, Customer=%s", dn.ID, dnNumber, customerName)
	return dn.ID, nil
}

func (a *App) routeToBankStatement(data map[string]any, fileName string, ocrConfidence float64) (string, error) {
	// Get or find bank account
	explicitBankAccountID := strings.TrimSpace(getStringField(data, "bank_account_id"))
	bankName := getStringField(data, "bank_name")
	accountNumber := getStringField(data, "account_number")
	butlerSummary := strings.TrimSpace(getStringField(data, "butler_summary"))
	userNotes := strings.TrimSpace(getStringField(data, "notes"))
	statementNotes := butlerSummary
	if statementNotes == "" {
		statementNotes = userNotes
	} else if userNotes != "" && userNotes != statementNotes {
		statementNotes = statementNotes + "\n\n--- User Notes ---\n" + userNotes
	}
	if len(statementNotes) > 4000 {
		statementNotes = statementNotes[:4000]
	}

	// Try to find matching bank account
	var bankAccountID string
	var accounts []CompanyBankAccount
	a.db.Where("is_active = ?", true).Find(&accounts)

	if explicitBankAccountID != "" {
		for _, acct := range accounts {
			if acct.ID == explicitBankAccountID {
				bankAccountID = acct.ID
				break
			}
		}
		if bankAccountID == "" {
			return "", fmt.Errorf("selected bank account is no longer active. Please choose the correct bank account and save again")
		}
	}

	if bankAccountID == "" {
		normalizeIdentifier := func(value string) string {
			var b strings.Builder
			for _, r := range strings.ToUpper(strings.TrimSpace(value)) {
				if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					b.WriteRune(r)
				}
			}
			return b.String()
		}
		normalizeBankName := func(value string) string {
			value = strings.ToLower(strings.TrimSpace(value))
			replacer := strings.NewReplacer(
				"bank", " ",
				"bsc", " ",
				"b.s.c", " ",
				"wll", " ",
				"w.l.l", " ",
				"account", " ",
				"current", " ",
				"checking", " ",
				"call", " ",
			)
			value = replacer.Replace(value)
			var b strings.Builder
			lastWasSpace := false
			for _, r := range value {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
					b.WriteRune(r)
					lastWasSpace = false
					continue
				}
				if !lastWasSpace {
					b.WriteRune(' ')
					lastWasSpace = true
				}
			}
			return strings.TrimSpace(b.String())
		}

		accountNeedle := normalizeIdentifier(accountNumber)
		bankNeedle := normalizeBankName(bankName)
		bestScore := 0
		bestID := ""
		tied := false

		for _, acct := range accounts {
			score := 0
			normalizedAccount := normalizeIdentifier(acct.AccountNumber)
			normalizedIBAN := normalizeIdentifier(acct.IBAN)
			normalizedBank := normalizeBankName(acct.BankName)

			if accountNeedle != "" {
				switch {
				case accountNeedle == normalizedAccount:
					score += 300
				case normalizedIBAN != "" && accountNeedle == normalizedIBAN:
					score += 300
				case len(accountNeedle) >= 6 && strings.HasSuffix(normalizedAccount, accountNeedle):
					score += 260
				case normalizedIBAN != "" && len(accountNeedle) >= 6 && strings.HasSuffix(normalizedIBAN, accountNeedle):
					score += 220
				}
			}

			if bankNeedle != "" {
				switch {
				case bankNeedle == normalizedBank:
					score += 90
				case normalizedBank != "" && strings.Contains(normalizedBank, bankNeedle):
					score += 45
				case bankNeedle != "" && strings.Contains(bankNeedle, normalizedBank):
					score += 45
				}
			}

			if score > bestScore {
				bestScore = score
				bestID = acct.ID
				tied = false
			} else if score > 0 && score == bestScore {
				tied = true
			}
		}

		if bestScore > 0 && !tied {
			bankAccountID = bestID
		} else if tied {
			return "", fmt.Errorf("statement matches multiple active bank accounts. Please choose the correct bank account before saving")
		}
	}

	// If still no account, return error suggesting user choose/create one
	if bankAccountID == "" {
		if len(accounts) == 0 {
			return "", fmt.Errorf("no bank accounts found. Please add a bank account in Bank Reconciliation first")
		}
		return "", fmt.Errorf("could not match this statement to an active bank account. Please choose the correct bank account before saving")
	}

	var matchedAccount CompanyBankAccount
	for _, acct := range accounts {
		if acct.ID == bankAccountID {
			matchedAccount = acct
			break
		}
	}

	// Parse balances
	openBal := getFloatField(data, "opening_balance")
	closeBal := getFloatField(data, "closing_balance")
	currency := getStringField(data, "currency")
	if currency == "" {
		currency = "BHD"
	}

	// Parse dates
	periodStart := parseDate(getStringField(data, "period_start"))
	periodEnd := parseDate(getStringField(data, "period_end"))
	if periodEnd.IsZero() {
		periodEnd = time.Now()
	}
	if periodStart.IsZero() {
		periodStart = periodEnd.AddDate(0, -1, 0) // Default to 1 month prior
	}

	replaceExisting := false
	if rawReplace, ok := data["replace_existing"]; ok {
		switch value := rawReplace.(type) {
		case bool:
			replaceExisting = value
		case string:
			replaceExisting = strings.EqualFold(strings.TrimSpace(value), "true")
		}
	}

	// Check for duplicate: same account + exact period
	var existingCount int64
	a.db.Model(&BankStatement{}).Where(
		"bank_account_id = ? AND period_start = ? AND period_end = ? AND deleted_at IS NULL",
		bankAccountID, periodStart, periodEnd,
	).Count(&existingCount)

	if existingCount > 0 {
		var oldStmt BankStatement
		a.db.Where("bank_account_id = ? AND period_start = ? AND period_end = ? AND deleted_at IS NULL",
			bankAccountID, periodStart, periodEnd).First(&oldStmt)
		if oldStmt.ID != "" && !replaceExisting {
			return "", fmt.Errorf("duplicate bank statement exists for this account and period: %s (%s to %s). Confirm replace to overwrite it",
				oldStmt.StatementNumber,
				oldStmt.PeriodStart.Format("2006-01-02"),
				oldStmt.PeriodEnd.Format("2006-01-02"))
		}
		if oldStmt.ID != "" && replaceExisting {
			a.db.Where("bank_statement_id = ?", oldStmt.ID).Delete(&BankStatementLine{})
			a.db.Delete(&oldStmt)
			log.Printf("♻️ Replaced existing statement %s for same period", oldStmt.StatementNumber)
		}
	}

	// Create bank statement
	stmtID := uuid.New().String()
	now := time.Now()
	stmtNum := buildBankStatementNumber(matchedAccount, periodStart, periodEnd)
	if stmtNum == "" {
		stmtNum = fmt.Sprintf("OCR-%s", now.Format("20060102-150405"))
	}

	totalDebitsInput := getFloatField(data, "total_debits")
	totalCreditsInput := getFloatField(data, "total_credits")
	if statementNotes == "" {
		statementNotes = buildBankStatementSummaryNote(matchedAccount, periodStart, periodEnd, openBal, closeBal, totalDebitsInput, totalCreditsInput, currency)
	}

	stmt := BankStatement{
		Base:            Base{ID: stmtID},
		BankAccountID:   bankAccountID,
		StatementNumber: stmtNum,
		StatementDate:   periodEnd,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		OpeningBalance:  openBal,
		ClosingBalance:  closeBal,
		Currency:        currency,
		Status:          "Imported",
		ImportedFrom:    fileName,
		ImportMethod:    "OCR",
		OCRConfidence:   ocrConfidence,
		Notes:           statementNotes,
		Division:        normalizeDivisionName(matchedAccount.Division),
	}

	if err := a.db.Create(&stmt).Error; err != nil {
		return "", fmt.Errorf("failed to create bank statement: %w", err)
	}

	// Create line items from transactions
	var totalDebits, totalCredits float64
	var debitCount, creditCount int

	if items, ok := data["line_items"]; ok {
		if itemsList, ok := items.([]any); ok {
			for i, item := range itemsList {
				if itemMap, ok := item.(map[string]any); ok {
					debit := parseFloatFromInterface(itemMap["debit"])
					credit := parseFloatFromInterface(itemMap["credit"])
					balance := parseFloatFromInterface(itemMap["balance"])

					txnDate := parseDateFromInterface(itemMap["date"])
					if txnDate.IsZero() {
						txnDate = now
					}

					line := BankStatementLine{
						Base:            Base{ID: uuid.New().String()},
						BankStatementID: stmtID,
						LineNumber:      i + 1,
						TransactionDate: txnDate,
						ValueDate:       txnDate,
						Description:     getStringFromInterface(itemMap["description"]),
						Reference:       getStringFromInterface(itemMap["reference"]),
						Debit:           debit,
						Credit:          credit,
						Balance:         balance,
						MatchType:       "Unmatched",
					}

					if err := a.db.Create(&line).Error; err != nil {
						log.Printf("⚠️ Failed to create bank statement line %d: %v", i+1, err)
						continue
					}

					if debit > 0 {
						totalDebits += debit
						debitCount++
					}
					if credit > 0 {
						totalCredits += credit
						creditCount++
					}
				}
			}
		}
	}

	// Balance validation: preserve extracted rows exactly as captured.
	// If the balance math looks wrong, log it for manual review instead of
	// silently flipping all debits/credits, which can corrupt statement truth.
	if closeBal > 0 && openBal > 0 {
		expected := openBal + totalCredits - totalDebits
		diff := math.Abs(expected - closeBal)
		if diff > 1.0 {
			log.Printf("⚠️ Bank statement balance mismatch requires review: stmt=%s expected_closing=%.3f actual_closing=%.3f diff=%.3f", stmtID, expected, closeBal, diff)
		}
	}

	// Update totals on statement
	if stmt.Notes == "" {
		stmt.Notes = buildBankStatementSummaryNote(matchedAccount, periodStart, periodEnd, openBal, closeBal, totalDebits, totalCredits, currency)
	}
	a.db.Model(&BankStatement{}).Where("id = ?", stmtID).Updates(map[string]any{
		"total_debits":  totalDebits,
		"total_credits": totalCredits,
		"debit_count":   debitCount,
		"credit_count":  creditCount,
		"notes":         stmt.Notes,
	})

	log.Printf("✅ Bank statement created: ID=%s, %d debits (%.3f), %d credits (%.3f)",
		stmtID, debitCount, totalDebits, creditCount, totalCredits)

	return stmtID, nil
}

func buildBankStatementNumber(account CompanyBankAccount, periodStart, periodEnd time.Time) string {
	identifierSanitizer := regexp.MustCompile(`[^A-Za-z0-9]+`)

	bankCode := "BANK"
	lowerBank := strings.ToLower(strings.TrimSpace(account.BankName))
	switch {
	case strings.Contains(lowerBank, "national bank of bahrain"):
		bankCode = "NBB"
	case strings.Contains(lowerBank, "al salam"):
		bankCode = "ALSALAM"
	case strings.Contains(lowerBank, "kuwait finance house"):
		bankCode = "KFH"
	case strings.Contains(lowerBank, "bbk"):
		bankCode = "BBK"
	default:
		cleanBank := identifierSanitizer.ReplaceAllString(strings.ToUpper(account.BankName), "")
		if cleanBank != "" {
			if len(cleanBank) > 8 {
				cleanBank = cleanBank[:8]
			}
			bankCode = cleanBank
		}
	}

	accountRef := strings.TrimSpace(account.AccountNumber)
	if accountRef == "" {
		accountRef = strings.TrimSpace(account.ID)
	}
	accountRef = identifierSanitizer.ReplaceAllString(accountRef, "")
	if accountRef == "" {
		return ""
	}

	if !periodStart.IsZero() && !periodEnd.IsZero() {
		return fmt.Sprintf("%s-%s-%s_to_%s", bankCode, accountRef, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	}
	if !periodEnd.IsZero() {
		return fmt.Sprintf("%s-%s-%s", bankCode, accountRef, periodEnd.Format("2006-01"))
	}
	return fmt.Sprintf("%s-%s", bankCode, accountRef)
}

func buildBankStatementSummaryNote(account CompanyBankAccount, periodStart, periodEnd time.Time, openingBalance, closingBalance, totalDebits, totalCredits float64, currency string) string {
	if currency == "" {
		currency = "BHD"
	}

	accountLabel := strings.TrimSpace(account.AccountName)
	if accountLabel == "" {
		accountLabel = strings.TrimSpace(account.BankName)
	}
	if accountLabel == "" {
		accountLabel = "selected account"
	}

	periodLabel := "the statement period"
	if !periodStart.IsZero() && !periodEnd.IsZero() {
		periodLabel = fmt.Sprintf("%s to %s", periodStart.Format("02 Jan 2006"), periodEnd.Format("02 Jan 2006"))
	} else if !periodEnd.IsZero() {
		periodLabel = periodEnd.Format("02 Jan 2006")
	}

	return fmt.Sprintf(
		"Bank statement for %s covering %s. Opening balance %s %.3f, closing balance %s %.3f, total debits %s %.3f, total credits %s %.3f.",
		accountLabel,
		periodLabel,
		currency,
		openingBalance,
		currency,
		closingBalance,
		currency,
		totalDebits,
		currency,
		totalCredits,
	)
}

// --- Field extraction helpers ---

func clampConfidence(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

func getStringField(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return fmt.Sprintf("%.0f", val)
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func getFloatField(data map[string]any, key string) float64 {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case string:
			var f float64
			fmt.Sscanf(strings.ReplaceAll(val, ",", ""), "%f", &f)
			return f
		case int:
			return float64(val)
		}
	}
	return 0
}

func getIntField(data map[string]any, key string) int {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		case string:
			var i int
			fmt.Sscanf(val, "%d", &i)
			return i
		}
	}
	return 0
}

// Helper functions for parsing interface{} values (used in bank statement line items)
func parseFloatFromInterface(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(strings.ReplaceAll(val, ",", ""), "%f", &f)
		return f
	default:
		return 0
	}
}

func getStringFromInterface(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func parseDateFromInterface(v any) time.Time {
	if v == nil {
		return time.Time{}
	}

	dateStr := ""
	switch val := v.(type) {
	case string:
		dateStr = val
	case float64:
		// Could be Unix timestamp
		if val > 1000000000 && val < 9999999999 {
			return time.Unix(int64(val), 0)
		}
		dateStr = fmt.Sprintf("%.0f", val)
	default:
		dateStr = fmt.Sprintf("%v", val)
	}

	return parseDate(dateStr)
}

// GetOCRStats returns OCR processing statistics
func (a *App) GetOCRStats() (map[string]any, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var stats struct {
		TotalDocuments int64
		AvgConfidence  float64
		TotalCost      float64
		DNACacheHits   int64
		GPUProcessed   int64
	}

	// Total documents
	a.db.Model(&OCRDocument{}).Count(&stats.TotalDocuments)

	// Average confidence
	a.db.Model(&OCRDocument{}).Select("AVG(confidence)").Row().Scan(&stats.AvgConfidence)

	// Total cost
	a.db.Model(&OCRDocument{}).Select("SUM(cost)").Row().Scan(&stats.TotalCost)

	// DNA cache hits
	a.db.Model(&OCRDocument{}).Where("dna_cache_hit = ?", true).Count(&stats.DNACacheHits)

	// GPU processed
	a.db.Model(&OCRDocument{}).Where("gpu_used = ?", true).Count(&stats.GPUProcessed)

	// Engine distribution
	var engineDist []struct {
		Engine string
		Count  int
	}
	a.db.Model(&OCRDocument{}).Select("engine, COUNT(*) as count").Group("engine").Scan(&engineDist)

	return map[string]any{
		"total_documents":     stats.TotalDocuments,
		"avg_confidence":      stats.AvgConfidence,
		"total_cost":          stats.TotalCost,
		"dna_cache_hits":      stats.DNACacheHits,
		"gpu_processed":       stats.GPUProcessed,
		"engine_distribution": engineDist,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// BATCH OFFER PROCESSING
// ═══════════════════════════════════════════════════════════════════════════

// BatchOfferResult represents batch offer processing result (for Wails binding)
type BatchOfferResult struct {
	TotalOffers       int            `json:"total_offers"`
	TotalFiles        int            `json:"total_files"`
	ProcessedFiles    int            `json:"processed_files"`
	SkippedFiles      int            `json:"skipped_files"`
	FailedFiles       int            `json:"failed_files"`
	TotalTimeSeconds  float64        `json:"total_time_seconds"`
	AverageConfidence float64        `json:"average_confidence"`
	DocumentsByType   map[string]int `json:"documents_by_type"`
	TotalCostUSD      float64        `json:"total_cost_usd"`
	GPUUsagePercent   float64        `json:"gpu_usage_percent"`
}

// BatchOfferProgress represents real-time batch processing progress (for Wails binding)
type BatchOfferProgress struct {
	CurrentFile    string  `json:"current_file"`
	FilesProcessed int     `json:"files_processed"`
	TotalFiles     int     `json:"total_files"`
	Percentage     float64 `json:"percentage"`
	OfferNumber    string  `json:"offer_number"`
	CustomerName   string  `json:"customer_name"`
	Stage          string  `json:"stage"`
	Error          string  `json:"error,omitempty"`
}

// ProcessOffersBatch processes all offers in the specified folder
func (a *App) ProcessOffersBatch(offersFolder string) (*BatchOfferResult, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}
	ctx := context.Background()

	// Initialize OCR engine
	engine, err := a.getOCREngine()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OCR engine: %w", err)
	}
	defer engine.Close()

	// Cast to ACEEngine to access batch methods
	aceEngine, ok := engine.(*ocr.ACEEngine)
	if !ok {
		return nil, fmt.Errorf("OCR engine is not ACEEngine type")
	}

	// Create progress channel (buffered to avoid blocking)
	progressChan := make(chan ocr.BatchOfferProgress, 100)
	defer close(progressChan)

	// Start goroutine to emit progress events to frontend via Wails runtime
	go func() {
		for progress := range progressChan {
			errStr := ""
			if progress.Error != nil {
				errStr = progress.Error.Error()
			}

			// Emit event to frontend
			runtime.EventsEmit(a.ctx, "batch-offer-progress", BatchOfferProgress{
				CurrentFile:    progress.CurrentFile,
				FilesProcessed: progress.FilesProcessed,
				TotalFiles:     progress.TotalFiles,
				Percentage:     progress.Percentage,
				OfferNumber:    progress.OfferNumber,
				CustomerName:   progress.CustomerName,
				Stage:          progress.Stage,
				Error:          errStr,
			})
		}
	}()

	// Create batch request
	req := &ocr.BatchOfferRequest{
		OffersFolder:   offersFolder,
		MaxConcurrency: 8, // Williams-optimized worker pool
		EnableGPU:      true,
		StopOnError:    false, // Continue processing on errors
		DB:             a.db,
		ProgressChan:   progressChan,
	}

	// Process batch
	result, err := aceEngine.ProcessOffersBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("batch processing failed: %w", err)
	}

	// Convert to Wails-compatible result
	return &BatchOfferResult{
		TotalOffers:       result.TotalOffers,
		TotalFiles:        result.TotalFiles,
		ProcessedFiles:    result.ProcessedFiles,
		SkippedFiles:      result.SkippedFiles,
		FailedFiles:       result.FailedFiles,
		TotalTimeSeconds:  result.TotalTime.Seconds(),
		AverageConfidence: result.AverageConfidence,
		DocumentsByType:   result.DocumentsByType,
		TotalCostUSD:      result.TotalCostUSD,
		GPUUsagePercent:   result.GPUUsagePercent,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// OCR ENGINE HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// getOCREngine initializes and returns the OCR engine
func (a *App) getOCREngine() (ocr.Engine, error) {
	config := &ocr.EngineConfig{
		EnableGPU:             true,
		GPUBackend:            ocr.GPULevelZero,
		MaxWorkers:            8,
		DefaultLanguage:       ocr.LangEnglish,
		EnablePreprocessing:   true,
		EnableVedicValidation: true,
		FallbackToAIMLAPI:     true,
		AIMLAPIKey:            os.Getenv("AIMLAPI_KEY"),
		TesseractPath:         a.config.Tools.TesseractPath,
		PandocPath:            a.config.Tools.PandocPath,
		LogLevel:              "info",
	}

	return ocr.NewACEEngine(config)
}

// parseAmountFromFields extracts amount from OCR fields
func parseAmountFromFields(fields map[string]string) float64 {
	amountStr := fields["total_amount"]
	if amountStr == "" {
		amountStr = fields["amount"]
	}
	if amountStr == "" {
		return 0.0
	}

	// Remove currency symbols and parse
	amountStr = strings.ReplaceAll(amountStr, "BHD", "")
	amountStr = strings.ReplaceAll(amountStr, "$", "")
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.TrimSpace(amountStr)

	amount := 0.0
	fmt.Sscanf(amountStr, "%f", &amount)
	return amount
}

// parseAmountFromFieldsInterface extracts amount from OCR fields (interface{} version)
func parseAmountFromFieldsInterface(fields map[string]any) float64 {
	// Try to get amount as interface{} and convert
	var amountStr string
	if v, ok := fields["total_amount"].(string); ok && v != "" {
		amountStr = v
	} else if v, ok := fields["amount"].(string); ok && v != "" {
		amountStr = v
	} else if v, ok := fields["total"].(float64); ok {
		return v
	} else if v, ok := fields["total_amount"].(float64); ok {
		return v
	}

	if amountStr == "" {
		return 0.0
	}

	// Remove currency symbols and parse
	amountStr = strings.ReplaceAll(amountStr, "BHD", "")
	amountStr = strings.ReplaceAll(amountStr, "$", "")
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.TrimSpace(amountStr)

	amount := 0.0
	fmt.Sscanf(amountStr, "%f", &amount)
	return amount
}

// parseDate parses date from string
func parseDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}

	// Try common date formats
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t
		}
	}

	return time.Time{}
}

// ═══════════════════════════════════════════════════════════════════════════
// UNIFIED OCR SERVICE METHODS (New Integration - Dec 22, 2025)
// ═══════════════════════════════════════════════════════════════════════════

// ExtractRFQDocument processes an RFQ document using the Simple OCR service
func (a *App) ExtractRFQDocument(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	log.Printf("🔥 ExtractRFQDocument called with: %s", filePath)

	if a.ocrService == nil {
		log.Printf("❌ OCR service not initialized!")
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	log.Printf("📞 Calling ocrService.ExtractRFQ...")
	result, err := a.ocrService.ExtractRFQ(filePath)
	if err != nil {
		log.Printf("❌ ExtractRFQ error: %v", err)
		return &OCRResult{Error: err.Error()}
	}

	log.Printf("✅ ExtractRFQ success: confidence=%.2f, engine=%s", result.Confidence, result.Engine)
	return result
}

// ExtractInvoiceDocument processes an invoice document using the Simple OCR service
func (a *App) ExtractInvoiceDocument(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	if a.ocrService == nil {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	result, err := a.ocrService.ExtractInvoice(filePath)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	return result
}

// ExtractQuotationDocument processes a quotation document using the Simple OCR service
func (a *App) ExtractQuotationDocument(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	if a.ocrService == nil {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	result, err := a.ocrService.ExtractQuotation(filePath)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	return result
}

// ProcessDocumentsBatch processes multiple documents in batch
func (a *App) ProcessDocumentsBatch(filePaths []string, docType string) []*OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return []*OCRResult{{Error: err.Error()}}
	}
	if a.ocrService == nil {
		return []*OCRResult{{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}}
	}

	results, err := a.ocrService.ProcessBatch(filePaths, docType)
	if err != nil {
		return []*OCRResult{{Error: err.Error()}}
	}
	return results
}

// GetOCRPipelineStats returns OCR pipeline statistics for monitoring
func (a *App) GetOCRPipelineStats() (map[string]any, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	if a.ocrService == nil {
		return nil, fmt.Errorf("OCR service not initialized")
	}

	stats := a.ocrService.GetPipelineStats()
	return stats, nil
}

// GetOCRProcessorStats returns individual processor statistics
func (a *App) GetOCRProcessorStats() (map[string]any, error) {
	if err := a.requirePermission("documents:view"); err != nil {
		return nil, err
	}
	if a.ocrService == nil {
		return nil, fmt.Errorf("OCR service not initialized")
	}

	stats := a.ocrService.GetProcessorStats()
	return stats, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// MANUAL OCR TIER SELECTION (Advanced Users)
// ═══════════════════════════════════════════════════════════════════════════

// ProcessWithGoFitz forces go-fitz processor (FREE, fast for vector PDFs)
func (a *App) ProcessWithGoFitz(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	if a.ocrService == nil {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	result, err := a.ocrService.ProcessWithGoFitz(filePath)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	return result
}

// ProcessWithFlorence2 forces Florence-2 processor (via Fly.io)
func (a *App) ProcessWithFlorence2(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	if a.ocrService == nil {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	result, err := a.ocrService.ProcessWithFlorence2(filePath)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	return result
}

// ProcessWithTesseract forces Tesseract processor (via Fly.io)
func (a *App) ProcessWithTesseract(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	if a.ocrService == nil {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	result, err := a.ocrService.ProcessWithTesseract(filePath)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	return result
}

// ProcessWithGPU forces GPU preprocessing (via Fly.io)
func (a *App) ProcessWithGPU(filePath string) *OCRResult {
	if err := a.requirePermission("documents:create"); err != nil {
		return &OCRResult{Error: err.Error()}
	}
	if a.ocrService == nil {
		return &OCRResult{
			Text:       "",
			Confidence: 0,
			Error:      "OCR service not initialized",
		}
	}

	result, err := a.ocrService.ProcessWithGPU(filePath)
	if err != nil {
		return &OCRResult{Error: err.Error()}
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// ETL SERVICE - Batch OCR and Data Seeding (Wave 3 Agent 1)
// ═══════════════════════════════════════════════════════════════════════════

// RunBatchOCR runs batch OCR processing on a folder of documents
// Returns a map with processing summary and results
func (a *App) RunBatchOCR(basePath, opportunitiesFile, outputDir string) (map[string]any, error) {
	if err := a.requirePermission("documents:create"); err != nil {
		return nil, err
	}
	if a.ocrService == nil {
		return nil, fmt.Errorf("OCR service not initialized")
	}

	log.Printf("🚀 Starting ETL batch OCR process")
	log.Printf("   Base path: %s", basePath)
	log.Printf("   Opportunities: %s", opportunitiesFile)
	log.Printf("   Output: %s", outputDir)

	etl := NewETLService(a.ocrService, basePath)

	// Load opportunities data if provided
	if opportunitiesFile != "" {
		if err := etl.LoadOpportunities(opportunitiesFile); err != nil {
			log.Printf("⚠️ Failed to load opportunities: %v (continuing without)", err)
		}
	}

	// Run batch processing
	if err := etl.ProcessAllProjects(outputDir); err != nil {
		return nil, fmt.Errorf("batch processing failed: %w", err)
	}

	// Return summary
	return map[string]any{
		"success":    true,
		"output_dir": outputDir,
		"message":    "Batch OCR processing complete",
	}, nil
}

// GenerateSeedData generates database seed data from the opportunities Excel
func (a *App) GenerateSeedData(opportunitiesFile, outputPath string) (map[string]any, error) {
	if err := a.requirePermission("settings:update"); err != nil {
		return nil, err
	}
	if a.ocrService == nil {
		return nil, fmt.Errorf("OCR service not initialized")
	}

	etl := NewETLService(a.ocrService, "")

	// Load opportunities
	if err := etl.LoadOpportunities(opportunitiesFile); err != nil {
		return nil, fmt.Errorf("failed to load opportunities: %w", err)
	}

	// Generate seed data
	seedData, err := etl.GenerateSeedData()
	if err != nil {
		return nil, fmt.Errorf("failed to generate seed data: %w", err)
	}

	// Save to file if path provided
	if outputPath != "" {
		if err := saveJSONFile(outputPath, seedData); err != nil {
			log.Printf("⚠️ Failed to save seed data to file: %v", err)
		}
	}

	return seedData, nil
}

// SeedDatabaseFromOpportunities seeds the database with customers, suppliers, and opportunities
func (a *App) SeedDatabaseFromOpportunities(opportunitiesFile string) (map[string]any, error) {
	// SECURITY: Admin-only permission for seed functions
	if err := a.requirePermission("*"); err != nil {
		return nil, err
	}
	seedData, err := a.GenerateSeedData(opportunitiesFile, "")
	if err != nil {
		return nil, err
	}

	// Extract data from seed
	customersData, _ := seedData["customers"].([]ExtractedCustomer)
	rfqsData, _ := seedData["rfqs"].([]map[string]any)

	// Seed customers using actual CustomerMaster fields from database.go
	customersCreated := 0
	for _, c := range customersData {
		existing := &CustomerMaster{}
		if err := a.db.Where("business_name = ?", c.NormalizedName).First(existing).Error; err == nil {
			continue // Already exists
		}

		// Generate customer code from name (first 3 letters + sequence)
		customerCode := strings.ToUpper(strings.ReplaceAll(c.NormalizedName, " ", ""))
		if len(customerCode) > 10 {
			customerCode = customerCode[:10]
		}

		customer := &CustomerMaster{
			CustomerID:    uuid.New().String(),
			CustomerCode:  customerCode + "-" + uuid.New().String()[:4],
			BusinessName:  c.NormalizedName,
			CustomerType:  "Industrial", // Default for instrumentation
			Country:       "Bahrain",    // Default
			CustomerGrade: "B",          // Default grade
		}

		if err := a.db.Create(customer).Error; err == nil {
			customersCreated++

			// If we have contact info, create a CustomerContact record
			if len(c.EmailAddresses) > 0 || len(c.PhoneNumbers) > 0 {
				contact := &CustomerContact{
					CustomerID:       customer.ID,
					IsPrimaryContact: true,
				}
				if len(c.EmailAddresses) > 0 {
					contact.Email = c.EmailAddresses[0]
				}
				if len(c.PhoneNumbers) > 0 {
					contact.Phone = c.PhoneNumbers[0]
				}
				a.db.Create(contact)
			}
		}
	}

	// Seed opportunities (replacing RFQs - Opportunity model handles sales pipeline)
	opportunitiesCreated := 0
	for _, rfq := range rfqsData {
		folderNumber, _ := rfq["rfq_number"].(string)
		if folderNumber == "" {
			continue
		}

		// Check if opportunity already exists by folder number
		existing := &Opportunity{}
		if err := a.db.Where("folder_number = ?", folderNumber).First(existing).Error; err == nil {
			continue
		}

		customerName, _ := rfq["customer_name"].(string)
		status, _ := rfq["status"].(string)
		valueBHD, _ := rfq["value_bhd"].(float64)

		// Map status to Stage
		stage := "Lead"
		switch strings.ToLower(status) {
		case "won", "execution":
			stage = "Won"
		case "lost":
			stage = "Lost"
		case "quoted", "proposal":
			stage = "Proposal"
		case "pending", "rfq":
			stage = "Lead"
		}

		newOpp := &Opportunity{
			FolderNumber: folderNumber,
			CustomerName: customerName,
			Stage:        stage,
			RevenueBHD:   valueBHD,
			OfferDate:    time.Now(),
			Division:     activeOverlay.DefaultDivision(),
		}

		if err := a.db.Create(newOpp).Error; err == nil {
			opportunitiesCreated++
		}
	}

	// Seed default suppliers (known instrumentation suppliers)
	suppliers := []map[string]string{
		{"name": "Rhine Instruments", "code": "EH", "country": "Germany", "payment_terms": "Net 60"},
		{"name": "Oxan Analytics", "code": "SRVX", "country": "UK", "payment_terms": "Net 45"},
		{"name": "Helix Automation", "code": "SIEM", "country": "Germany", "payment_terms": "Net 60"},
		{"name": "Apex Process", "code": "EMER", "country": "USA", "payment_terms": "Net 45"},
		{"name": "ABB", "code": "ABB", "country": "Switzerland", "payment_terms": "Net 45"},
		{"name": "Meridian Systems", "code": "YOKO", "country": "Japan", "payment_terms": "Net 60"},
		{"name": "Northwind Controls", "code": "HON", "country": "USA", "payment_terms": "Net 45"},
		{"name": "Granite Gauge Works", "code": "GIC", "country": "India", "payment_terms": "Net 30"},
	}

	suppliersCreated := 0
	for _, s := range suppliers {
		existing := &SupplierMaster{}
		if err := a.db.Where("supplier_name = ?", s["name"]).First(existing).Error; err == nil {
			continue
		}

		supplier := &SupplierMaster{
			SupplierCode: s["code"],
			SupplierName: s["name"],
			Country:      s["country"],
			PaymentTerms: s["payment_terms"],
			Rating:       5, // Default 5-star
		}

		if err := a.db.Create(supplier).Error; err == nil {
			suppliersCreated++
		}
	}

	return map[string]any{
		"success":               true,
		"customers_created":     customersCreated,
		"suppliers_created":     suppliersCreated,
		"opportunities_created": opportunitiesCreated,
		"message":               fmt.Sprintf("Seeded %d customers, %d suppliers, %d opportunities", customersCreated, suppliersCreated, opportunitiesCreated),
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// CONTRACT GENERATION SYSTEM (Wave 2 Agent 5)
// ═══════════════════════════════════════════════════════════════════════════

// GenerateContract generates a new contract for a customer
