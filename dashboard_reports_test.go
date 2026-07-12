package main

// Wave 8 Bucket C — dashboard/report methods: AR-aging YTD (collectibility-
// normalized), pipeline-by-stage YTD, inventory pending-fulfillment report,
// inventory movements workspace.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetDashboardARAgingReportYTD_NormalizesCollectibility(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Invoice{}))

	now := time.Now()
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "ar-cur"}, InvoiceNumber: "AR-1", CustomerID: "c1", CustomerName: "Alpha",
		InvoiceDate: now.AddDate(0, 0, -5), DueDate: now.AddDate(0, 0, 10),
		Status: "Sent", GrandTotalBHD: 10, OutstandingBHD: 10,
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "ar-45"}, InvoiceNumber: "AR-2", CustomerID: "c1", CustomerName: "Alpha",
		InvoiceDate: now.AddDate(0, 0, -75), DueDate: now.AddDate(0, 0, -45),
		Status: "Sent", GrandTotalBHD: 20, OutstandingBHD: 20,
	}).Error)
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "ar-part"}, InvoiceNumber: "AR-3", CustomerID: "c2", CustomerName: "Beta",
		InvoiceDate: now.AddDate(0, 0, -35), DueDate: now.AddDate(0, 0, -5),
		Status: "PartiallyPaid", GrandTotalBHD: 50, OutstandingBHD: 15,
	}).Error)
	// Draft with stale outstanding — must be EXCLUDED by the payment-state
	// normalization (the behavioral upgrade over the raw status filter).
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "ar-draft"}, InvoiceNumber: "AR-4", CustomerID: "c3", CustomerName: "Gamma",
		InvoiceDate: now.AddDate(0, 0, -200), DueDate: now.AddDate(0, 0, -170),
		Status: "Draft", GrandTotalBHD: 99, OutstandingBHD: 99,
	}).Error)

	report, err := app.GetDashboardARAgingReportYTD()
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Nil(t, report.Details, "dashboard variant strips per-invoice details")
	require.InDelta(t, 10.0, report.Current, 0.001)
	require.InDelta(t, 15.0, report.Days30, 0.001, "5-day overdue partial payment lands in 30+")
	require.InDelta(t, 20.0, report.Days60, 0.001, "45-day overdue lands in 60+")
	require.InDelta(t, 45.0, report.Total, 0.001, "Draft's stale 99 BHD must not inflate the total")

	// The finance report keeps details and shares the same engine.
	full, err := app.GetARAgingReport()
	require.NoError(t, err)
	require.Len(t, full.Details, 3)
	require.InDelta(t, report.Total, full.Total, 0.001)
}

func TestGetDashboardPipelineByStageYTD_GroupsActivityYear(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Opportunity{}))

	year := time.Now().Year()
	rows := []Opportunity{
		{Base: Base{ID: "opp-1"}, FolderNumber: "PIPE-001", Title: "Plant retrofit", Source: "manual", Year: year, Stage: "Qualified", RevenueBHD: 200},
		{Base: Base{ID: "opp-2"}, FolderNumber: "PIPE-002", Title: "Meter swap", Source: "manual", Year: year, Stage: "qualified", RevenueBHD: 300},
		{Base: Base{ID: "opp-3"}, FolderNumber: "PIPE-003", Title: "Analyzer deal", Source: "manual", Year: year, Stage: "Won", RevenueBHD: 1000},
		// Prior-year row is outside the activity-year window.
		{Base: Base{ID: "opp-4"}, FolderNumber: "PIPE-004", Title: "Old tender", Source: "manual", Year: year - 1, Stage: "Lost", RevenueBHD: 400},
	}
	require.NoError(t, app.db.Create(&rows).Error)

	pipeline, err := app.GetDashboardPipelineByStageYTD()
	require.NoError(t, err)
	require.Len(t, pipeline, 2, "prior-year stage must not appear")

	// Sorted by count desc: Qualified (2, case-insensitively grouped) then Won (1).
	require.Equal(t, "Qualified", pipeline[0].Stage)
	require.Equal(t, 2, pipeline[0].Count)
	require.InDelta(t, 500.0, pipeline[0].Value, 0.001)
	require.Equal(t, "#0EA5E9", pipeline[0].Color)
	require.Equal(t, "Won", pipeline[1].Stage)
	require.Equal(t, 1, pipeline[1].Count)
	require.InDelta(t, 1000.0, pipeline[1].Value, 0.001)
	require.Equal(t, "#10B981", pipeline[1].Color)
}

func TestGetInventoryPendingFulfillmentReport_ComputesShortage(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Order{}, &OrderItem{}, &DeliveryNoteItem{}, &InventoryItem{}))

	require.NoError(t, app.db.Create(&Order{
		Base: Base{ID: "ful-o1"}, OrderNumber: "FUL-ORD-1", CustomerName: "Alpha Trading Co",
		OrderDate: time.Now().AddDate(0, 0, -3), Status: "Confirmed",
	}).Error)
	require.NoError(t, app.db.Create(&OrderItem{
		Base: Base{ID: "ful-i1"}, OrderID: "ful-o1", ProductID: "prod-1", ProductCode: "SVX-100",
		Description: "Gas analyzer", Quantity: 10, QuantityShipped: 4, QuantityInvoiced: 3,
	}).Error)
	// Second line: shipped counter still zero — delivered must fall back to DN items.
	require.NoError(t, app.db.Create(&OrderItem{
		Base: Base{ID: "ful-i2"}, OrderID: "ful-o1", ProductID: "prod-2", ProductCode: "FLW-200",
		Description: "Flow meter", Quantity: 5, QuantityShipped: 0,
	}).Error)
	require.NoError(t, app.db.Create(&DeliveryNoteItem{
		Base: Base{ID: "ful-dn1"}, DeliveryNoteID: "dn-1", OrderItemID: "ful-i2",
		ProductCode: "FLW-200", QuantityDelivered: 5,
	}).Error)
	require.NoError(t, app.db.Create(&InventoryItem{
		Base: Base{ID: "ful-inv1"}, ProductID: "prod-1", ProductCode: "SVX-100",
		QuantityAvailable: 2, IsActive: true,
	}).Error)

	rows, err := app.GetInventoryPendingFulfillmentReport(0)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byCode := map[string]InventoryPendingFulfillmentRow{}
	for _, row := range rows {
		byCode[row.ProductCode] = row
	}

	analyzer := byCode["SVX-100"]
	require.InDelta(t, 6.0, analyzer.PendingQuantity, 0.001)
	require.InDelta(t, 4.0, analyzer.DeliveredQuantity, 0.001)
	require.InDelta(t, 3.0, analyzer.InvoicedQuantity, 0.001)
	require.InDelta(t, 2.0, analyzer.AvailableQuantity, 0.001)
	require.InDelta(t, 4.0, analyzer.ShortageQuantity, 0.001, "pending 6 - available 2")

	meter := byCode["FLW-200"]
	require.InDelta(t, 5.0, meter.DeliveredQuantity, 0.001, "DN fallback when shipped counter is zero")
	require.InDelta(t, 0.0, meter.PendingQuantity, 0.001)
	require.InDelta(t, 0.0, meter.ShortageQuantity, 0.001)
}

func TestGetInventoryMovementsWorkspace_ReturnsRecentFeed(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&StockMovement{}))

	now := time.Now()
	require.NoError(t, app.db.Create(&StockMovement{
		Base: Base{ID: "mov-1"}, InventoryItemID: "inv-1", MovementType: "Receipt",
		MovementNumber: "MOV-1", Quantity: 5, Direction: "IN", MovementDate: now.AddDate(0, 0, -1),
	}).Error)
	require.NoError(t, app.db.Create(&StockMovement{
		Base: Base{ID: "mov-2"}, InventoryItemID: "inv-1", MovementType: "Issue",
		MovementNumber: "MOV-2", Quantity: 2, Direction: "OUT", MovementDate: now,
	}).Error)

	movements, err := app.GetInventoryMovementsWorkspace(0)
	require.NoError(t, err)
	require.Len(t, movements, 2)
	require.Equal(t, "MOV-2", movements[0].MovementNumber, "newest movement first")

	// inventory:view gates the workspace via the delegate.
	setDataQualityRole(app, "user-1", "sales", `["customers:view"]`, "Sales")
	_, err = app.GetInventoryMovementsWorkspace(0)
	require.Error(t, err)
}
