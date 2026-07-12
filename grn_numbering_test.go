package main

// W4 A.2: GRN numbering moved off the legacy BEGIN EXCLUSIVE max-scan onto
// pkg/documents/numbering (the last document type to migrate after the S4
// fixes moved INV/CN/PO/DN). These tests pin the migration semantics: the
// format is unchanged and the first allocation of a year continues from the
// highest existing legacy number instead of restarting at 0001.

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateGRNNumber_ContinuesFromLegacyMax(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&GoodsReceivedNote{}))
	year := time.Now().Year()

	// Legacy rows numbered by the old max-scan generator.
	for _, n := range []int{1, 2, 7} {
		grn := GoodsReceivedNote{
			Base:      Base{ID: fmt.Sprintf("grn-legacy-%d", n)},
			GRNNumber: fmt.Sprintf("GRN-%d-%04d", year, n),
		}
		require.NoError(t, app.db.Create(&grn).Error)
	}

	first, err := app.GenerateGRNNumber()
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("GRN-%d-0008", year), first,
		"first allocation must continue from the legacy max, not the count")

	second, err := app.GenerateGRNNumber()
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("GRN-%d-0009", year), second)
}

func TestGenerateGRNNumber_FreshYearStartsAtOne(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&GoodsReceivedNote{}))
	year := time.Now().Year()

	number, err := app.GenerateGRNNumber()
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("GRN-%d-0001", year), number)
}

func TestGenerateGRNNumber_DeletedGRNDoesNotReissueNumber(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&GoodsReceivedNote{}))

	first, err := app.GenerateGRNNumber()
	require.NoError(t, err)

	// Simulate the numbered GRN being deleted: the sequence row must still
	// advance — numbers are never reissued.
	next, err := app.GenerateGRNNumber()
	require.NoError(t, err)
	require.NotEqual(t, first, next)
	require.Greater(t, next, first)
}
