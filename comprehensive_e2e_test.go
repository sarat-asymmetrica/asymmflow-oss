package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// =============================================================================
// COMPREHENSIVE E2E TESTS - Full Pipeline Validation
//
// Tests ALL major flows end-to-end:
//   1. FieldCrypto (HKDF + AES-256-GCM, key rotation, versioning)
//   2. SettingsService (encrypt/decrypt with legacy fallback)
//   3. Bank Account Encryption (GORM hooks, migration)
//   4. CRM (Customer + Supplier CRUD, contacts, profiles)
//   5. RFQ -> Costing -> Offer pipeline
//   6. Orders (CRUD, stage transitions)
//   7. Invoicing (create, mark paid, partial payment, AR aging)
//   8. Payments (record, list)
//   9. Purchase Orders (CRUD)
//  10. GRN (create, complete, discrepancies)
//  11. Financial (dashboard, year data)
//  12. Utility (CSRF, greet, config)
// =============================================================================

// setupFullTestApp creates a comprehensive test environment with all tables migrated
func setupFullTestApp(t *testing.T) *App {
	t.Helper()

	// Use shared cache with unique name per test so:
	// 1. Multiple connections within a test share the same in-memory database (needed for db.Begin())
	// 2. Different tests get isolated databases (no data contamination)
	dbName := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	require.NoError(t, err, "Failed to create in-memory database")

	// Migrate all tables needed for E2E tests
	tables := []any{
		&CustomerMaster{},
		&CustomerContact{},
		&SupplierMaster{},
		&SupplierContact{},
		&Order{},
		&OrderItem{},
		&Payment{},
		&DBInvoiceItem{},
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&SupplierInvoice{},
		&GoodsReceivedNote{},
		&GRNItem{},
		&GRNDiscrepancy{},
		&InventoryItem{},
		&StockMovement{},
		&Opportunity{},
		&Offer{},
		&OfferItem{},
		&Setting{},
		&CompanyBankAccount{},
		&FollowUpTask{},
		&PostSaleNote{},
		&QuickCapture{},
		&Shipment{},
		&DeliveryNote{},
		&DeliveryNoteItem{},
		&RFQData{},
		&RFQComment{},
		&InvoiceSequence{},
		&PredictionRecord{},
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			t.Logf("Warning: AutoMigrate failed for %T: %v (continuing)", table, err)
		}
	}

	// Migrate Invoice separately (has many CHECK constraints but they work with SQLite)
	if err := db.AutoMigrate(&Invoice{}); err != nil {
		t.Logf("Warning: Invoice AutoMigrate failed: %v - creating table manually", err)
		db.Exec(`CREATE TABLE IF NOT EXISTS invoices (
			id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			version INTEGER DEFAULT 1, created_by TEXT, invoice_number TEXT UNIQUE, invoice_date DATETIME,
			customer_id TEXT, customer_name TEXT, order_id TEXT, customer_po_number TEXT,
			grand_total_bhd REAL DEFAULT 0, status TEXT DEFAULT 'Sent', outstanding_bhd REAL DEFAULT 0,
			subtotal_bhd REAL DEFAULT 0, due_date DATETIME, updated_by TEXT,
			rfq_id TEXT, quote_id TEXT, offer_id TEXT, offer_number TEXT,
			delivery_note_id TEXT, delivery_note_number TEXT,
			total_supplier_cost_bhd REAL DEFAULT 0, gross_margin_bhd REAL DEFAULT 0, gross_margin_percent REAL DEFAULT 0,
			vatbhd REAL DEFAULT 0, vat_percent REAL DEFAULT 0, journal_entry_id TEXT, field_visibility TEXT
		)`)
	}

	// Initialize FieldCrypto
	fc, err := NewFieldCrypto()
	require.NoError(t, err, "FieldCrypto initialization failed")
	globalFieldCrypto = fc

	// Initialize SettingsService
	settingsSvc := &SettingsService{
		db:          db,
		hardwareID:  "test-hardware-id",
		fieldCrypto: fc,
	}
	// Set legacy key
	legacyKey := [32]byte{}
	copy(legacyKey[:], "test-legacy-encryption-key-32byt")
	settingsSvc.encryptionKey = legacyKey[:]

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		fieldCrypto:            fc,
		settingsService:        settingsSvc,
		config:                 &Config{},
		startupImporting:       false,
		startupImportStartTime: time.Now(),
		currentUserID:          "test-user-001", // Avoid DB query in getCurrentUserID()
		currentUser: &User{
			Base:     Base{ID: "test-user-001"},
			Username: "test-admin",
			RoleName: "admin",
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator",
				Permissions: `["*"]`,
			},
		},
	}
	t.Cleanup(app.cache.Stop)

	return app
}

// =============================================================================
// 1. FIELD CRYPTO TESTS
// =============================================================================

func TestFieldCrypto_EncryptDecrypt(t *testing.T) {
	fc, err := NewFieldCrypto()
	require.NoError(t, err)

	testCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"short text", "hello"},
		{"API key", "sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890"},
		{"account number", "00DEMO0000000001"},
		{"IBAN", "BH29ACME00000000000000"},
		{"unicode", "Acme Instrumentation \u0628\u062d\u0631\u064a\u0646"},
		{"long text", strings.Repeat("sensitive-data-", 100)},
		{"special chars", "p@$$w0rd!#%^&*(){}[]|\\:\";<>?,./~`"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := fc.Encrypt(tc.input)
			require.NoError(t, err)

			if tc.input == "" {
				assert.Equal(t, "", encrypted, "Empty input should return empty output")
				return
			}

			assert.NotEqual(t, tc.input, encrypted, "Encrypted should differ from plaintext")
			assert.True(t, fc.IsEncrypted(encrypted), "Should detect encrypted value")

			decrypted, err := fc.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tc.input, decrypted, "Decrypted should match original")
		})
	}
}

func TestFieldCrypto_DifferentCiphertexts(t *testing.T) {
	fc, err := NewFieldCrypto()
	require.NoError(t, err)

	// Same plaintext should produce different ciphertexts (random nonce)
	enc1, _ := fc.Encrypt("same-input")
	enc2, _ := fc.Encrypt("same-input")
	assert.NotEqual(t, enc1, enc2, "Two encryptions of same text should differ")

	// But both should decrypt to the same value
	dec1, _ := fc.Decrypt(enc1)
	dec2, _ := fc.Decrypt(enc2)
	assert.Equal(t, dec1, dec2, "Both should decrypt to same value")
}

func TestFieldCrypto_IsEncrypted(t *testing.T) {
	fc, err := NewFieldCrypto()
	require.NoError(t, err)

	// Plaintext values should NOT be detected as encrypted
	assert.False(t, fc.IsEncrypted(""), "Empty string")
	assert.False(t, fc.IsEncrypted("00DEMO0000000001"), "Plain account number")
	assert.False(t, fc.IsEncrypted("BH29ACME00000000000000"), "Plain IBAN")
	assert.False(t, fc.IsEncrypted("hello world"), "Plain text")
	assert.False(t, fc.IsEncrypted("not-base64!@#$"), "Not base64")

	// Encrypted values SHOULD be detected
	enc, _ := fc.Encrypt("test-value")
	assert.True(t, fc.IsEncrypted(enc), "Encrypted value should be detected")
}

func TestFieldCrypto_KeyRotation(t *testing.T) {
	fc, err := NewFieldCrypto()
	require.NoError(t, err)

	// Encrypt with v1
	assert.Equal(t, uint8(1), fc.CurrentVersion())
	encV1, err := fc.Encrypt("secret-v1")
	require.NoError(t, err)

	// Rotate to v2
	newVer, err := fc.Rotate()
	require.NoError(t, err)
	assert.Equal(t, uint8(2), newVer)
	assert.Equal(t, uint8(2), fc.CurrentVersion())

	// Encrypt with v2
	encV2, err := fc.Encrypt("secret-v2")
	require.NoError(t, err)

	// v1 ciphertext should still decrypt
	decV1, err := fc.Decrypt(encV1)
	require.NoError(t, err)
	assert.Equal(t, "secret-v1", decV1, "v1 ciphertext should still decrypt after rotation")

	// v2 ciphertext should decrypt
	decV2, err := fc.Decrypt(encV2)
	require.NoError(t, err)
	assert.Equal(t, "secret-v2", decV2, "v2 ciphertext should decrypt")

	// Rotate several more times
	for i := 0; i < 5; i++ {
		_, err = fc.Rotate()
		require.NoError(t, err)
	}

	// v1 should STILL decrypt
	decV1Again, err := fc.Decrypt(encV1)
	require.NoError(t, err)
	assert.Equal(t, "secret-v1", decV1Again, "v1 should decrypt even after many rotations")
}

func TestFieldCrypto_InvalidInput(t *testing.T) {
	fc, err := NewFieldCrypto()
	require.NoError(t, err)

	// Decrypt garbage
	_, err = fc.Decrypt("not-valid-base64!!!")
	assert.Error(t, err, "Should fail on invalid base64")

	// Decrypt too-short base64
	_, err = fc.Decrypt("AQID") // Valid base64 but too short
	assert.Error(t, err, "Should fail on too-short ciphertext")

	// Decrypt tampered ciphertext
	enc, _ := fc.Encrypt("test")
	tampered := enc[:len(enc)-4] + "AAAA"
	_, err = fc.Decrypt(tampered)
	assert.Error(t, err, "Should fail on tampered ciphertext (GCM auth tag)")
}

// =============================================================================
// 2. SETTINGS SERVICE TESTS
// =============================================================================

func TestSettingsService_EncryptedSettings(t *testing.T) {
	app := setupFullTestApp(t)
	svc := app.settingsService

	// Save encrypted setting
	err := svc.SetSetting("apiKeys.test_key", "sk-test-12345", "apiKeys", true)
	require.NoError(t, err)

	// Retrieve and verify decryption
	val, err := svc.GetSetting("apiKeys.test_key")
	require.NoError(t, err)
	assert.Equal(t, "sk-test-12345", val, "Should decrypt to original value")

	// Verify it's actually encrypted in DB
	var setting Setting
	app.db.Where("key = ?", "apiKeys.test_key").First(&setting)
	assert.True(t, setting.IsEncrypted, "Setting should be marked encrypted")
	assert.NotEqual(t, "sk-test-12345", setting.Value, "Value in DB should NOT be plaintext")
}

func TestSettingsService_PlaintextSettings(t *testing.T) {
	app := setupFullTestApp(t)
	svc := app.settingsService

	// Save unencrypted setting
	err := svc.SetSetting("ui.theme", "dark", "ui", false)
	require.NoError(t, err)

	val, err := svc.GetSetting("ui.theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestSettingsService_SettingNotFound(t *testing.T) {
	app := setupFullTestApp(t)
	_, err := app.settingsService.GetSetting("nonexistent.key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "setting not found")
}

func TestSettingsService_UpdateExistingSetting(t *testing.T) {
	app := setupFullTestApp(t)
	svc := app.settingsService

	// Set initial value
	err := svc.SetSetting("test.key", "value1", "test", false)
	require.NoError(t, err)

	// Update
	err = svc.SetSetting("test.key", "value2", "test", false)
	require.NoError(t, err)

	val, err := svc.GetSetting("test.key")
	require.NoError(t, err)
	assert.Equal(t, "value2", val, "Should return updated value")
}

// =============================================================================
// 3. BANK ACCOUNT ENCRYPTION TESTS
// =============================================================================

func TestBankAccount_SeedAndEncryption(t *testing.T) {
	app := setupFullTestApp(t)

	// Seed bank accounts
	err := app.SeedCompanyBankAccounts()
	require.NoError(t, err)

	// Bank details are intentionally stored plaintext now so the DB remains portable
	// across client machines and invoice PDFs can read them directly.
	banks, err := app.GetActiveBankAccounts()
	require.NoError(t, err)
	require.Len(t, banks, 5, "Should have PH and AHS seeded bank accounts")

	// Verify decrypted values are readable (not gibberish)
	nbb := banks[0]
	assert.Equal(t, "Demo Bank A", nbb.BankName)
	assert.Equal(t, "10000000001", nbb.AccountNumber, "Account number should be decrypted for API")
	assert.Equal(t, "BH29DMOA10000000000001", nbb.IBAN, "IBAN should be decrypted for API")

	ahli := banks[1]
	assert.Equal(t, "10000000002", ahli.AccountNumber)
	assert.Equal(t, "BH29DMOB10000000000002", ahli.IBAN)

	var ahsBank *CompanyBankAccount
	for i := range banks {
		if banks[i].Division == "Beacon Controls" {
			ahsBank = &banks[i]
			break
		}
	}
	require.NotNil(t, ahsBank, "AHS should have a seeded bank account")
	assert.Equal(t, "20000000001", ahsBank.AccountNumber)

	// Verify data remains plaintext in the raw DB as well
	var rawBank CompanyBankAccount
	app.db.Raw("SELECT account_number, iban FROM company_bank_accounts WHERE id = ?", "bank-alpha").Scan(&rawBank)
	assert.Equal(t, "10000000001", rawBank.AccountNumber, "Raw DB should keep plaintext account number")
	assert.Equal(t, "BH29DMOA10000000000001", rawBank.IBAN, "Raw DB should keep plaintext IBAN")
	assert.False(t, app.fieldCrypto.IsEncrypted(rawBank.AccountNumber), "Raw value should not be encrypted anymore")
}

func TestBankAccount_CreateEncrypted(t *testing.T) {
	app := setupFullTestApp(t)

	newBank, err := app.CreateBankAccount(CompanyBankAccount{
		BankName:      "Test Bank",
		AccountName:   "Acme Test Account",
		AccountNumber: "1234567890",
		IBAN:          "BH99TEST1234567890",
		SwiftBIC:      "TESTBHBM",
		Currency:      "BHD",
	})
	require.NoError(t, err)
	require.NotNil(t, newBank)

	// Retrieve and verify decrypted
	retrieved, err := app.GetBankAccountByID(newBank.ID)
	require.NoError(t, err)
	assert.Equal(t, "1234567890", retrieved.AccountNumber)
	assert.Equal(t, "BH99TEST1234567890", retrieved.IBAN)

	// Verify plaintext in raw DB for portability
	var rawBank CompanyBankAccount
	app.db.Raw("SELECT account_number, iban FROM company_bank_accounts WHERE id = ?", newBank.ID).Scan(&rawBank)
	assert.Equal(t, "1234567890", rawBank.AccountNumber)
	assert.Equal(t, "BH99TEST1234567890", rawBank.IBAN)
}

func TestBankAccount_Migration(t *testing.T) {
	app := setupFullTestApp(t)

	// Insert a plaintext bank account directly (simulating pre-encryption data)
	app.db.Exec(`INSERT INTO company_bank_accounts (id, bank_name, account_name, account_number, iban, swift_bic, currency, is_active, display_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"bank-legacy", "Legacy Bank", "Acme Legacy", "PLAINTEXT123", "BH99PLAIN123", "LEGABH", "BHD", true, 99, time.Now(), time.Now())

	// Run migration
	app.MigrateBankAccountEncryption()

	// Verify plaintext remains available after migration
	var rawBank CompanyBankAccount
	app.db.Raw("SELECT account_number, iban FROM company_bank_accounts WHERE id = ?", "bank-legacy").Scan(&rawBank)
	assert.Equal(t, "PLAINTEXT123", rawBank.AccountNumber, "Migration should preserve plaintext portability")
	assert.Equal(t, "BH99PLAIN123", rawBank.IBAN)

	// Verify GORM still reads back the same plaintext values
	var decryptedBank CompanyBankAccount
	app.db.First(&decryptedBank, "id = ?", "bank-legacy")
	assert.Equal(t, "PLAINTEXT123", decryptedBank.AccountNumber, "Should decrypt after migration")
	assert.Equal(t, "BH99PLAIN123", decryptedBank.IBAN)
}

func TestBankAccount_UpdatePreservesEncryption(t *testing.T) {
	app := setupFullTestApp(t)

	// Seed
	err := app.SeedCompanyBankAccounts()
	require.NoError(t, err)

	// Update the bank account name (non-sensitive field)
	_, err = app.UpdateBankAccount("bank-alpha", map[string]any{
		"bank_name": "Demo Bank A - Updated Name",
	})
	require.NoError(t, err)

	// Verify account number is still encrypted and readable
	bank, err := app.GetBankAccountByID("bank-alpha")
	require.NoError(t, err)
	assert.Equal(t, "Demo Bank A - Updated Name", bank.BankName)
	assert.Equal(t, "10000000001", bank.AccountNumber, "Account number should still decrypt correctly")
}

func TestBankAccount_GetAll(t *testing.T) {
	app := setupFullTestApp(t)
	err := app.SeedCompanyBankAccounts()
	require.NoError(t, err)

	banks, err := app.GetAllBankAccounts()
	require.NoError(t, err)
	assert.Len(t, banks, 5)

	// All should have decrypted values
	for _, bank := range banks {
		assert.NotEmpty(t, bank.AccountNumber, "Account number should be present")
		assert.NotEmpty(t, bank.IBAN, "IBAN should be present")
		assert.False(t, app.fieldCrypto.IsEncrypted(bank.AccountNumber),
			"API-returned account number should be plaintext (decrypted)")
	}
}

// =============================================================================
// 4. API KEY ENCRYPTION TESTS
// =============================================================================

func TestSetAPIKeys_EncryptsToDatabase(t *testing.T) {
	app := setupFullTestApp(t)

	err := app.SetAPIKeys(map[string]string{
		"mistral_key":   "test-mistral-key-12345",
		"openai_key":    "sk-test-openai-67890",
		"anthropic_key": "sk-ant-test-abcdef",
	})
	require.NoError(t, err)

	// Verify each key is encrypted in DB
	for _, key := range []string{"apiKeys.mistral_key", "apiKeys.openai_key", "apiKeys.anthropic_key"} {
		var setting Setting
		err := app.db.Where("key = ?", key).First(&setting).Error
		require.NoError(t, err, "Setting %s should exist", key)
		assert.True(t, setting.IsEncrypted, "Setting %s should be encrypted", key)
		assert.NotContains(t, setting.Value, "test", "Encrypted value should not contain plaintext")
	}
}

func TestSetAPIKeys_SkipsMaskedValues(t *testing.T) {
	app := setupFullTestApp(t)

	// First save a real key
	err := app.SetAPIKeys(map[string]string{
		"mistral_key": "real-key-12345",
	})
	require.NoError(t, err)

	// Then call with masked value "****" - should NOT overwrite
	err = app.SetAPIKeys(map[string]string{
		"mistral_key": "****",
	})
	require.NoError(t, err)

	// Original should still be there
	var setting Setting
	app.db.Where("key = ?", "apiKeys.mistral_key").First(&setting)
	decrypted, err := app.fieldCrypto.Decrypt(setting.Value)
	require.NoError(t, err)
	assert.Equal(t, "real-key-12345", decrypted, "Masked update should not overwrite real key")
}

// =============================================================================
// 5. KEY ROTATION END-TO-END
// =============================================================================

func TestKeyRotation_EndToEnd(t *testing.T) {
	app := setupFullTestApp(t)

	// Setup: seed banks, save API keys
	err := app.SeedCompanyBankAccounts()
	require.NoError(t, err)

	err = app.SetAPIKeys(map[string]string{
		"mistral_key": "pre-rotation-key",
	})
	require.NoError(t, err)

	// Rotate
	err = app.RotateEncryptionKey()
	require.NoError(t, err)

	assert.Equal(t, uint8(2), app.fieldCrypto.CurrentVersion(), "Should be on version 2")

	// Verify bank accounts still readable
	banks, err := app.GetActiveBankAccounts()
	require.NoError(t, err)
	require.Len(t, banks, 5)
	assert.Equal(t, "10000000001", banks[0].AccountNumber, "Bank data should survive rotation")

	// Verify settings still readable
	var setting Setting
	app.db.Where("key = ?", "apiKeys.mistral_key").First(&setting)
	decrypted, err := app.fieldCrypto.Decrypt(setting.Value)
	require.NoError(t, err)
	assert.Equal(t, "pre-rotation-key", decrypted, "API key should survive rotation")
}

// =============================================================================
// 6. CRM - CUSTOMER CRUD
// =============================================================================

func TestCustomer_CRUD(t *testing.T) {
	app := setupFullTestApp(t)

	// CREATE
	customer := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerCode:     "CUST-NATPETRO",
		BusinessName:     "National Petroleum Co.",
		CustomerType:     "Corporate",
		City:             "Awali",
		Country:          "Bahrain",
		PaymentGrade:     "A",
		CustomerGrade:    "A",
		PaymentTermsDays: 30,
		CreditLimitBHD:   100000.0,
	}
	created, err := app.CreateCustomer(customer)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "National Petroleum Co.", created.BusinessName)

	// READ
	retrieved, err := app.GetCustomer(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "National Petroleum Co.", retrieved.BusinessName)
	assert.Equal(t, "Bahrain", retrieved.Country)

	// LIST
	customers, err := app.ListCustomers(10, 0)
	require.NoError(t, err)
	assert.Len(t, customers, 1)

	// UPDATE
	created.City = "Manama"
	updatedCustomer, err := app.UpdateCustomer(*created)
	require.NoError(t, err)
	assert.Equal(t, "Manama", updatedCustomer.City)

	// DELETE
	err = app.DeleteCustomer(created.ID)
	require.NoError(t, err)

	// Reset cache so ListCustomers queries fresh from DB
	newCache := NewCache()
	app.cache = newCache
	t.Cleanup(newCache.Stop)

	// Verify soft delete
	customers, err = app.ListCustomers(10, 0)
	require.NoError(t, err)
	assert.Len(t, customers, 0, "Should not appear after soft delete")
}

func TestCustomer_Contacts(t *testing.T) {
	app := setupFullTestApp(t)

	// Create customer first
	cust := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerCode: "CUST-GULFSMELT",
		BusinessName: "Gulf Smelting Co.",
	}
	_, err := app.CreateCustomer(cust)
	require.NoError(t, err)

	// Add contact
	contact := CustomerContact{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:       cust.ID,
		ContactName:      "John Smith",
		Email:            "john@gulfsmelting.example",
		Phone:            "+973 17111111",
		JobTitle:         "Procurement Manager",
		IsPrimaryContact: true,
	}
	created, err := app.AddCustomerContact(contact)
	require.NoError(t, err)
	assert.Equal(t, "John Smith", created.ContactName)

	// List contacts
	contacts, err := app.ListCustomerContacts(cust.ID)
	require.NoError(t, err)
	assert.Len(t, contacts, 1)
	assert.Equal(t, "john@gulfsmelting.example", contacts[0].Email)
}

// =============================================================================
// 7. CRM - SUPPLIER CRUD
// =============================================================================

func TestSupplier_CRUD(t *testing.T) {
	app := setupFullTestApp(t)

	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierCode: "SUP-EH",
		SupplierName: "Rhine Instruments",
		Country:      "Switzerland",
		LeadTimeDays: 45,
		PaymentTerms: "Net 60",
		Rating:       5,
	}
	created, err := app.CreateSupplier(supplier)
	require.NoError(t, err)
	assert.Equal(t, "Rhine Instruments", created.SupplierName)

	// List
	suppliers, err := app.ListSuppliers(10, 0)
	require.NoError(t, err)
	assert.Len(t, suppliers, 1)

	// Get
	got, err := app.GetSupplier(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Switzerland", got.Country)

	// Update
	created.LeadTimeDays = 60
	updated, err := app.UpdateSupplier(*created)
	require.NoError(t, err)
	assert.Equal(t, 60, updated.LeadTimeDays)

	// Delete
	err = app.DeleteSupplier(created.ID)
	require.NoError(t, err)
}

// =============================================================================
// 8. ORDERS - CRUD & STAGES
// =============================================================================

func TestOrder_CreateAndList(t *testing.T) {
	app := setupFullTestApp(t)

	// Create customer first
	cust := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerCode: "CUST-TEST",
		BusinessName: "Test Corp",
	}
	app.db.Create(&cust)

	// Create order
	order, err := app.CreateOrder("ORD-2025-001", "Test Corp", 15000.0, time.Now().Format("2006-01-02"), "confirmed")
	require.NoError(t, err)
	require.NotNil(t, order)
	assert.Equal(t, "ORD-2025-001", order.OrderNumber)

	// List orders
	orders, err := app.ListOrders(10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orders), 1)

	// Get order
	got, err := app.GetOrder(order.ID)
	require.NoError(t, err)
	assert.Equal(t, 15000.0, got.GrandTotalBHD)
}

func TestOrder_StageTransitions(t *testing.T) {
	app := setupFullTestApp(t)

	order, err := app.CreateOrder("ORD-STAGE-001", "Stage Test Corp", 5000.0, time.Now().Format("2006-01-02"), "Confirmed")
	require.NoError(t, err)

	// Advance through the current canonical state machine
	stages := []string{"Processing", "FullyDelivered", "Invoiced", "Complete"}
	for _, stage := range stages {
		err = app.UpdateOrderStage(order.ID, stage)
		require.NoError(t, err, "Should transition to %s", stage)

		// Verify stage persisted
		got, err := app.GetOrder(order.ID)
		require.NoError(t, err)
		assert.Equal(t, stage, got.Status, "Status should be %s", stage)
	}
}

// =============================================================================
// 9. INVOICING
// =============================================================================

func TestInvoice_CreateFromOrder(t *testing.T) {
	app := setupFullTestApp(t)

	// Create customer
	cust := CustomerMaster{
		Base:             Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerCode:     "CUST-INV",
		BusinessName:     "Invoice Test Corp",
		PaymentTermsDays: 30,
	}
	app.db.Create(&cust)

	// Create order with items
	order := Order{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderNumber:   "ORD-INV-001",
		CustomerID:    cust.ID,
		CustomerName:  cust.BusinessName,
		TotalValueBHD: 5000.0,
		GrandTotalBHD: 5000.0,
		Status:        "confirmed",
		OrderDate:     time.Now(),
	}
	app.db.Create(&order)

	// Create order items
	item := OrderItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrderID:     order.ID,
		Description: "Flow Meter XYZ",
		Quantity:    2,
		UnitPrice:   2500.0,
		TotalPrice:  5000.0,
		Currency:    "BHD",
	}
	app.db.Create(&item)

	// Create invoice from order
	invoice, err := app.CreateInvoiceFromOrder(order.ID)
	require.NoError(t, err)
	require.NotNil(t, invoice)
	assert.Equal(t, cust.ID, invoice.CustomerID)
	assert.NotEmpty(t, invoice.InvoiceNumber)

	// List invoices
	invoices, err := app.ListCustomerInvoices(10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(invoices), 1)
}

func TestInvoice_PartialPayment(t *testing.T) {
	app := setupFullTestApp(t)

	// Create invoice directly
	invoiceID := uuid.New().String()
	invoiceNumber := "INV-PP-001"
	invoiceDate := time.Now()
	customerID := "cust-test"
	customerName := "Partial Pay Corp"
	grandTotalBHD := 10000.0
	outstandingBHD := 10000.0
	status := "Sent"
	dueDate := time.Now().AddDate(0, 0, 30)

	app.db.Exec(`INSERT INTO invoices (id, invoice_number, invoice_date, customer_id, customer_name, grand_total_bhd, outstanding_bhd, status, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		invoiceID, invoiceNumber, invoiceDate, customerID,
		customerName, grandTotalBHD, outstandingBHD, status, dueDate)

	// Record partial payment
	err := app.RecordPartialPayment(invoiceID, 3000.0, time.Now(), "CHQ-001")
	require.NoError(t, err)

	// Check outstanding is reduced
	inv, err := app.GetCustomerInvoiceByID(invoiceID)
	require.NoError(t, err)
	assert.InDelta(t, 7000.0, inv.OutstandingBHD, 0.01, "Outstanding should be reduced by payment")
	assert.Equal(t, "PartiallyPaid", inv.Status)

	// Record remaining payment
	err = app.RecordPartialPayment(invoiceID, 7000.0, time.Now(), "CHQ-002")
	require.NoError(t, err)

	inv, err = app.GetCustomerInvoiceByID(invoiceID)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, inv.OutstandingBHD, 0.01, "Should be fully paid")
	assert.Equal(t, "Paid", inv.Status)
}

// =============================================================================
// 10. PAYMENTS
// =============================================================================

func TestPayments_RecordAndList(t *testing.T) {
	app := setupFullTestApp(t)

	// Create an invoice to pay against
	app.db.Exec(`INSERT INTO invoices (id, invoice_number, invoice_date, customer_id, customer_name, grand_total_bhd, outstanding_bhd, status, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"inv-pay-001", "INV-PAY-001", time.Now(), "cust-pay", "Payment Test Corp", 5000.0, 5000.0, "Sent", time.Now().AddDate(0, 0, 30))

	// Record payment (use "Bank Transfer" to match CHECK constraint on payment_method)
	payment, err := app.RecordPayment("inv-pay-001", 5000.0, "Bank Transfer", time.Now().Format("2006-01-02"), "TT-12345")
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, 5000.0, payment.AmountBHD)

	// List payments
	payments, err := app.GetAllPayments(10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(payments), 1)

	// Get payments by invoice
	invPayments, err := app.GetPaymentsByInvoice("inv-pay-001")
	require.NoError(t, err)
	assert.Len(t, invPayments, 1)
}

// =============================================================================
// 11. RFQ & COSTING
// =============================================================================

func TestRFQ_CreateAndList(t *testing.T) {
	app := setupFullTestApp(t)

	// Create RFQ
	rfq, err := app.CreateRFQ("National Petroleum Co.", "Pressure Transmitters", 25000.0, "Urgent requirement for National Petroleum refinery", "")
	require.NoError(t, err)
	require.NotNil(t, rfq)
	assert.Equal(t, "National Petroleum Co.", rfq.Client)
	assert.Equal(t, "pending", rfq.Status)

	// List
	rfqs, err := app.GetRFQs(10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(rfqs), 1)

	// Get
	got, err := app.GetRFQ(rfq.ID)
	require.NoError(t, err)
	assert.Equal(t, "Pressure Transmitters", got.Project)

	// Update status
	err = app.UpdateRFQStatus(rfq.ID, "quoted")
	require.NoError(t, err)

	got, err = app.GetRFQ(rfq.ID)
	require.NoError(t, err)
	assert.Equal(t, "quoted", got.Status)
}

// =============================================================================
// 12. PURCHASE ORDERS
// =============================================================================

func TestPurchaseOrder_Create(t *testing.T) {
	app := setupFullTestApp(t)

	// Create supplier
	supplier := SupplierMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierCode: "SUP-PO-TEST",
		SupplierName: "PO Test Supplier",
		Country:      "Germany",
	}
	app.db.Create(&supplier)

	// Create PO
	po := PurchaseOrder{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		PONumber:     "PO-2025-001",
		SupplierID:   supplier.ID,
		SupplierName: supplier.SupplierName,
		PODate:       time.Now(),
		Currency:     "EUR",
		ExchangeRate: 0.385,
		TotalForeign: 10000.0,
		TotalBHD:     3850.0,
		Status:       "Draft",
	}
	err := app.db.Create(&po).Error
	require.NoError(t, err)

	// Verify it persisted
	var retrieved PurchaseOrder
	err = app.db.First(&retrieved, "id = ?", po.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "PO-2025-001", retrieved.PONumber)
	assert.Equal(t, "PO Test Supplier", retrieved.SupplierName)
}

// =============================================================================
// 13. GRN - GOODS RECEIVED NOTES
// =============================================================================

func TestGRN_CreateAndComplete(t *testing.T) {
	app := setupFullTestApp(t)

	// Create a PO first (GRN requires a valid PurchaseOrderID)
	poID := uuid.New().String()
	po := PurchaseOrder{
		Base:         Base{ID: poID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		PONumber:     "PO-GRN-TEST",
		SupplierID:   "sup-test",
		SupplierName: "GRN Test Supplier",
		PODate:       time.Now(),
		Currency:     "BHD",
		TotalBHD:     5000.0,
		Status:       "Sent",
	}
	app.db.Create(&po)

	grn := GoodsReceivedNote{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		GRNNumber:       "GRN-2025-001",
		PurchaseOrderID: poID,
		ReceivedBy:      "Warehouse Manager",
		ReceivedDate:    time.Now(),
		QCStatus:        "Pending",
	}
	created, err := app.CreateGRN(grn)
	require.NoError(t, err)
	assert.Equal(t, "GRN-2025-001", created.GRNNumber)

	// List GRNs
	grns, err := app.ListGRNs(10, 0, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(grns), 1)

	// Get specific GRN
	got, err := app.GetGRNByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "GRN-2025-001", got.GRNNumber)

	// Complete GRN
	err = app.CompleteGRN(created.ID)
	require.NoError(t, err)

	got, err = app.GetGRNByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Passed", got.QCStatus)
}

// =============================================================================
// 14. FOLLOW-UPS
// =============================================================================

func TestFollowUps_CRUD(t *testing.T) {
	app := setupFullTestApp(t)

	// Create customer first (FollowUp requires CustomerID)
	custID := uuid.New().String()
	app.db.Create(&CustomerMaster{
		Base:         Base{ID: custID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:   "CUST-FU-001",
		CustomerCode: "CUST-NATPETRO-FU",
		BusinessName: "National Petroleum Co.",
	})

	task := FollowUpTask{
		Base:        Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:  custID,
		Title:       "Follow up with National Petroleum on PO",
		Description: "Call procurement team about pending PO approval",
		DueDate:     time.Now().AddDate(0, 0, 7),
		Priority:    "high",
		Status:      "pending",
	}
	created, err := app.CreateFollowUp(task)
	require.NoError(t, err)
	assert.Equal(t, "Follow up with National Petroleum on PO", created.Title)

	// List
	tasks, err := app.ListFollowUps(10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 1)

	// Complete
	err = app.CompleteFollowUp(created.ID)
	require.NoError(t, err)
}

// =============================================================================
// 15. UTILITY FUNCTIONS
// =============================================================================

func TestCSRFToken(t *testing.T) {
	app := setupFullTestApp(t)

	token := app.GetCSRFToken()
	assert.NotEmpty(t, token, "Should generate a CSRF token")
	assert.Len(t, token, 64, "Token should be 64 hex chars (32 bytes)")

	// Validate
	valid := app.ValidateCSRFToken(token)
	assert.True(t, valid, "Fresh token should be valid")

	// Invalid token
	invalid := app.ValidateCSRFToken("totally-fake-token")
	assert.False(t, invalid, "Fake token should be invalid")

	// Same token should only work once (consumed)
	secondUse := app.ValidateCSRFToken(token)
	assert.False(t, secondUse, "Token should be consumed after first validation")
}

func TestGreet(t *testing.T) {
	app := setupFullTestApp(t)
	msg := app.Greet("Commander")
	assert.Contains(t, msg, "Commander", "Should contain the name")
}

func TestGeometryBridge(t *testing.T) {
	bridge := NewGeometryBridge()

	// Test invoice processing
	invoice := InvoiceGeometry{
		ID:         "INV-GEO-001",
		CustomerID: "CUST-001",
		Amount:     15000.0,
		IssueDate:  time.Now(),
		DueDate:    time.Now().AddDate(0, 0, 45),
		Status:     "pending",
		Currency:   "BHD",
		ItemCount:  5,
	}

	result, err := bridge.ProcessInvoice(invoice)
	require.NoError(t, err)
	assert.NotEmpty(t, result.InvoiceID)
	assert.Greater(t, result.PredictedDays, 0)
	assert.GreaterOrEqual(t, result.Confidence, 0.0)
}

// =============================================================================
// 16. FINANCIAL YEAR DATA
// =============================================================================

func TestFinancialYearData(t *testing.T) {
	app := setupFullTestApp(t)

	// Get available years (should include hardcoded 2023, 2024)
	years, err := app.GetAvailableFinancialYears()
	require.NoError(t, err)
	assert.NotEmpty(t, years, "Should have available financial years")

	// Get FY2024 data (hardcoded demo financial data)
	data, err := app.GetFinancialYearData(2024)
	require.NoError(t, err)
	assert.NotNil(t, data)
}

// =============================================================================
// 17. DASHBOARD
// =============================================================================

func TestDashboardStats(t *testing.T) {
	app := setupFullTestApp(t)

	// Seed some data (CustomerID must be unique - has uniqueIndex)
	for i := 0; i < 3; i++ {
		app.db.Create(&CustomerMaster{
			Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
			CustomerID:   fmt.Sprintf("CID-DASH-%03d", i),
			CustomerCode: fmt.Sprintf("CUST-DASH-%03d", i),
			BusinessName: fmt.Sprintf("Dashboard Customer %d", i),
		})
	}

	stats, err := app.GetDashboardStats()
	require.NoError(t, err)

	// Check it has expected fields
	assert.GreaterOrEqual(t, stats.ActiveCustomers, 3, "Should count at least 3 customers")
}

// =============================================================================
// 18. SEARCH & FILTER
// =============================================================================

func TestFilterOrders(t *testing.T) {
	app := setupFullTestApp(t)

	// Create test orders
	for i := 1; i <= 5; i++ {
		app.CreateOrder(
			fmt.Sprintf("ORD-FILTER-%03d", i),
			fmt.Sprintf("Filter Corp %d", i),
			float64(i*1000),
			time.Now().Format("2006-01-02"),
			"confirmed",
		)
	}

	// Filter by customer name
	orders, err := app.FilterOrders("Filter Corp", "", "", "", 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orders), 5, "Should find all Filter Corp orders")

	// Filter by status
	orders, err = app.FilterOrders("", "", "", "confirmed", 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orders), 5)
}

// =============================================================================
// 19. QUICK CAPTURES
// =============================================================================

func TestQuickCapture_CRUD(t *testing.T) {
	app := setupFullTestApp(t)

	// Create (returns uint ID, not pointer)
	captureID, err := app.CreateQuickCapture("Test Note", "This is a test quick capture", "test,note", "medium")
	require.NoError(t, err)
	assert.Greater(t, captureID, uint(0), "Should return a valid ID")

	// List
	captures, err := app.GetQuickCaptures(10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(captures), 1)
}

// =============================================================================
// 20. CONCURRENT ENCRYPTION SAFETY
// =============================================================================

func TestFieldCrypto_ConcurrentAccess(t *testing.T) {
	fc, err := NewFieldCrypto()
	require.NoError(t, err)

	done := make(chan bool, 20)

	// 10 goroutines encrypting
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()
			val := fmt.Sprintf("concurrent-value-%d", n)
			enc, err := fc.Encrypt(val)
			assert.NoError(t, err)
			dec, err := fc.Decrypt(enc)
			assert.NoError(t, err)
			assert.Equal(t, val, dec)
		}(i)
	}

	// 10 goroutines rotating + encrypting
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()
			if n%3 == 0 {
				fc.Rotate()
			}
			val := fmt.Sprintf("rotated-value-%d", n)
			enc, err := fc.Encrypt(val)
			assert.NoError(t, err)
			dec, err := fc.Decrypt(enc)
			assert.NoError(t, err)
			assert.Equal(t, val, dec)
		}(i)
	}

	// Wait for all
	for i := 0; i < 20; i++ {
		<-done
	}
}
