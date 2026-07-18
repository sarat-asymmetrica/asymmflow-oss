package main

// INTEG residue campaign — Wave R1.4 (bank-account CRUD) validation.
//
// CONTRACT NOTE: CompanyBankAccount IBAN/SWIFT/account_number are stored
// PLAINTEXT by deliberate design. Field encryption was removed —
// migrateBankAccountEncryption strips leftover ciphertext back to plaintext.
// So the correct assertion is a plaintext ROUNDTRIP (stored == entered), NOT
// "ciphertext != plaintext". This drives the bound Create/Update methods
// against a scratch SQLite and proves the plaintext contract end-to-end.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegBankAccount_CreateUpdateRoundtrip(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CompanyBankAccount{}), "migrate bank accounts")

	const iban = "BH29SYNZ00000000000042"
	const swift = "SYNZBHBM"
	const acctNo = "90000000042"

	// --- Create: operator fields go plaintext; server fills id/timestamps. ---
	created, err := app.CreateBankAccount(CompanyBankAccount{
		BankName:      "Synthetic Test Bank",
		AccountName:   "R1.4 Operating Account",
		AccountNumber: acctNo,
		IBAN:          iban,
		SwiftBIC:      swift,
		Currency:      "BHD",
	})
	require.NoError(t, err, "a valid account must create")
	require.NotNil(t, created)
	require.NotEmpty(t, created.ID, "server assigns an id")
	require.True(t, created.IsActive, "create forces is_active=true")

	// --- Read back from the DB: fields are stored plaintext (roundtrip). ---
	var stored CompanyBankAccount
	require.NoError(t, app.db.First(&stored, "id = ?", created.ID).Error)
	require.Equal(t, iban, stored.IBAN, "IBAN stored plaintext")
	require.Equal(t, swift, stored.SwiftBIC, "SWIFT stored plaintext")
	require.Equal(t, acctNo, stored.AccountNumber, "account number stored plaintext")
	// Explicit proof it is NOT ciphertext (only when the crypto helper exists).
	if globalFieldCrypto != nil {
		require.False(t, globalFieldCrypto.IsEncrypted(stored.IBAN), "IBAN must not be encrypted")
	}

	// --- Update: a plaintext patch of whitelisted columns roundtrips. ---
	const iban2 = "BH29SYNZ00000000000099"
	updated, err := app.UpdateBankAccount(created.ID, map[string]any{
		"iban":         iban2,
		"account_name": "R1.4 Operating Account (renamed)",
		"is_active":    false,
	})
	require.NoError(t, err, "a whitelisted patch must apply")
	require.Equal(t, iban2, updated.IBAN, "updated IBAN stored plaintext")
	require.Equal(t, "R1.4 Operating Account (renamed)", updated.AccountName)
	require.False(t, updated.IsActive, "is_active patch applies")

	// Re-read confirms persistence.
	var reread CompanyBankAccount
	require.NoError(t, app.db.First(&reread, "id = ?", created.ID).Error)
	require.Equal(t, iban2, reread.IBAN)
	require.False(t, reread.IsActive)
}
