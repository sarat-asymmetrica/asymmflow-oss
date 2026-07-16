package main

// INTEG residue campaign — Wave R3 (non-financial + untyped-patch tail).
// The R3 agents wired ~40 mutations, type-gated against the generated bindings.
// Most rely on that type-gate + existing service-test coverage; this file adds
// focused NEW coverage for the highest-risk FINANCIAL untyped-patch wiring the
// agents confirmed by reading the server whitelist: UpdateAccount. The bridge
// sends a snake_case patch of ONLY whitelisted columns — this proves the Go
// side enforces that whitelist (balance is posting-owned and must be dropped).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIntegR3_UpdateAccountWhitelist(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&ChartOfAccount{}), "migrate chart of accounts")

	acct := ChartOfAccount{
		Base:        Base{ID: uuid.New().String()},
		AccountCode: "R3-4000",
		AccountName: "Synthetic Account",
		AccountType: "Expense",
		Balance:     1234.567,
		IsActive:    true,
	}
	require.NoError(t, app.db.Create(&acct).Error)

	// --- A whitelisted field applies; a posting-owned field (balance) is dropped. ---
	require.NoError(t, app.UpdateAccount(acct.ID, map[string]any{
		"account_name": "Renamed Account",
		"is_active":    false,
		"balance":      999999.0, // NOT whitelisted — must be ignored
	}))

	var stored ChartOfAccount
	require.NoError(t, app.db.First(&stored, "id = ?", acct.ID).Error)
	require.Equal(t, "Renamed Account", stored.AccountName, "whitelisted account_name applies")
	require.False(t, stored.IsActive, "whitelisted is_active applies")
	require.InDelta(t, 1234.567, stored.Balance, 1e-6, "posting-owned balance must be dropped, not overwritten")

	// --- A patch of ONLY non-whitelisted keys is refused (nothing to apply). ---
	err := app.UpdateAccount(acct.ID, map[string]any{"balance": 5.0})
	require.Error(t, err, "a patch with no editable fields is refused")

	var reread ChartOfAccount
	require.NoError(t, app.db.First(&reread, "id = ?", acct.ID).Error)
	require.InDelta(t, 1234.567, reread.Balance, 1e-6, "balance still untouched after the refused patch")
}
