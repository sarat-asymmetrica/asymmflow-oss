package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplyCustomerReferenceSeed_UpdatesLegacyCustomerIDs(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CustomerMaster{}))

	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		CustomerID:   "CUST-OLD-001",
		CustomerCode: "CUST-OLD-001",
		BusinessName: "GULF SMELTING B.S.C",
		CustomerType: "Industrial",
		Status:       "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	seedPath := filepath.Join(t.TempDir(), "customer_reference_seed.tsv")
	seedText := "business_name\tcustomer_type\tshort_code\tserial_no\tcustomer_id\n" +
		"GULF SMELTING B.S.C.(C)\tEnd Customer\tEC\t14\tEC14\n"
	require.NoError(t, os.WriteFile(seedPath, []byte(seedText), 0644))

	result, err := applyCustomerReferenceSeed(app.db, seedPath)
	require.NoError(t, err)
	require.Equal(t, 1, result.SeedRows)
	require.Equal(t, 1, result.MatchedExisting)
	require.Equal(t, 1, result.UpdatedExisting)
	require.Equal(t, 0, result.InsertedNew)

	var stored CustomerMaster
	require.NoError(t, app.db.First(&stored, "id = ?", customer.ID).Error)
	require.Equal(t, "EC14", stored.CustomerID)
	require.Equal(t, "EC14", stored.CustomerCode)
	require.Equal(t, "EC", stored.ShortCode)
	require.Equal(t, "End Customer", stored.CustomerType)
}
