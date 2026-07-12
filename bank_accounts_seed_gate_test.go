package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ph_holdings_app/pkg/overlay"
)

// TestDemoBankSeedRespectsSeedSets pins the PC-D18 fix (Mission H rehearsal
// finding): the demo bank fixtures were re-created on EVERY boot regardless of
// the overlay, so a sovereign deployment's first startup after a data import
// injected five synthetic-IBAN accounts next to the company's real ones. The
// demo-bank seed-set gate now lives at the seam every caller funnels through.
func TestDemoBankSeedRespectsSeedSets(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CompanyBankAccount{}))

	saved := activeOverlay
	t.Cleanup(func() { activeOverlay = saved })

	// setupTestApp's foundations may already have seeded fixtures — start
	// from the imported-deployment state: a table holding only company rows
	// (none, for this test's purposes).
	require.NoError(t, app.db.Unscoped().Where("1 = 1").Delete(&CompanyBankAccount{}).Error)

	// Sovereign-style overlay: demo-bank not listed → seed is a no-op.
	sovereign := *overlay.BuiltinDefaults()
	sovereign.SeedSets = []string{"default-assets"}
	activeOverlay = &sovereign
	require.NoError(t, app.seedCompanyBankAccountsInternal())
	var n int64
	require.NoError(t, app.db.Model(&CompanyBankAccount{}).Count(&n).Error)
	require.Zero(t, n, "sovereign overlay must not receive demo bank fixtures")

	// Demo/default overlay (nil SeedSets → every bundle): fixtures seed.
	activeOverlay = overlay.BuiltinDefaults()
	require.NoError(t, app.seedCompanyBankAccountsInternal())
	require.NoError(t, app.db.Model(&CompanyBankAccount{}).Count(&n).Error)
	require.NotZero(t, n, "default overlay must keep seeding the demo fixtures")
}
