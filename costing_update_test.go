package main

// G2 standing default: UpdateCostingSheet wired from the costing VM with a full
// CostingSheetData assembled from the VM's authoritative totals (R1 technique).
// This pins the App-binding contract the kernel bridge (realUpdateCostingSheet)
// drives: the refreshed items JSON + the 4 totals persist, while the server-owned
// approval fields (Status/ApprovedBy/RevisionNumber/CreatedBy) are preserved and
// no duplicate row is created.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateCostingSheet_RefreshesItemsAndTotals_PreservesApproval(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CostingSheetData{}))

	// Seed an existing, already-approved costing sheet directly.
	initial := &CostingSheetData{
		RFQID:          7,
		RFQName:        "Gulf Fabrication - Skid Package",
		RevisionNumber: 3,
		IsActive:       true,
		Items:          `{"lineItems":[{"equipment":"Old Meter"}],"totalCost":500,"grandTotal":700,"profit":200,"profitPercent":28.57}`,
		Subtotal:       500,
		FinalPrice:     700,
		TotalMarkup:    200,
		MarginPercent:  28.57,
		Status:         "approved",
		ApprovedBy:     "manager-1",
		CreatedBy:      "prep-1",
		CustomerName:   "Gulf Fabrication W.L.L.",
	}
	require.NoError(t, app.db.Create(initial).Error)
	id := initial.ID

	// Refresh: new items + new totals (what the VM assembles on a re-costing).
	updated, err := app.UpdateCostingSheet(id, CostingSheetData{
		RFQID:         7,
		Items:         `{"lineItems":[{"equipment":"New Meter"},{"equipment":"Transmitter"}],"totalCost":800,"grandTotal":1100,"profit":300,"profitPercent":27.27}`,
		Subtotal:      800,
		FinalPrice:    1100,
		TotalMarkup:   300,
		MarginPercent: 27.27,
		CustomerName:  "Gulf Fabrication W.L.L.",
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	var row CostingSheetData
	require.NoError(t, app.db.First(&row, "id = ?", id).Error)
	// Items + totals refreshed.
	require.Contains(t, row.Items, "New Meter")
	require.Equal(t, 800.0, row.Subtotal)
	require.Equal(t, 1100.0, row.FinalPrice)
	require.Equal(t, 300.0, row.TotalMarkup)
	require.InDelta(t, 27.27, row.MarginPercent, 0.01)
	// Server-owned / approval fields preserved — a client refresh can NEVER
	// mass-assign approval state or overwrite the creator.
	require.Equal(t, "approved", row.Status)
	require.Equal(t, "manager-1", row.ApprovedBy)
	require.Equal(t, "prep-1", row.CreatedBy)
	require.Equal(t, 3, row.RevisionNumber)

	// No duplicate — update is in place.
	var count int64
	require.NoError(t, app.db.Model(&CostingSheetData{}).Where("rfq_id = ?", 7).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
