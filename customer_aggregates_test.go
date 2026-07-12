package main

// Wave 8 Bucket G tail — CRM recompute methods: RecomputeAllCustomerAggregates
// (computed CustomerMaster columns from live order/invoice/payment data) and
// RecomputeCustomerPrediction (payment-history-gated prediction generation).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecomputeAllCustomerAggregates_ComputesColumns(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CustomerMaster{}, &Order{}, &Invoice{}, &Payment{}))

	require.NoError(t, app.db.Create(&CustomerMaster{
		Base: Base{ID: "agg-c1"}, CustomerID: "AGG-C1", CustomerCode: "AGGC1",
		BusinessName: "Alpha Trading Co", Status: "Active",
	}).Error)
	require.NoError(t, app.db.Create(&CustomerMaster{
		Base: Base{ID: "agg-c2"}, CustomerID: "AGG-C2", CustomerCode: "AGGC2",
		BusinessName: "Beta Industrial", Status: "Active",
	}).Error)

	// Two orders for c1, linked via two DIFFERENT identifier aliases — the
	// rollup must union them through uniqueNonEmptyStrings(ID, CustomerID, Code).
	require.NoError(t, app.db.Create(&Order{
		Base: Base{ID: "agg-o1"}, OrderNumber: "AGG-ORD-1", CustomerID: "agg-c1",
		OrderDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Status: "Confirmed", GrandTotalBHD: 150.5,
	}).Error)
	require.NoError(t, app.db.Create(&Order{
		Base: Base{ID: "agg-o2"}, OrderNumber: "AGG-ORD-2", CustomerID: "AGGC1",
		OrderDate: time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), Status: "Confirmed", TotalValueBHD: 49.5,
	}).Error)

	// Overdue collectible invoice (100 days past due) + a payment 45 days-to-pay.
	now := time.Now()
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "agg-i1"}, InvoiceNumber: "AGG-INV-1", CustomerID: "agg-c1",
		InvoiceDate: now.AddDate(0, 0, -130), DueDate: now.AddDate(0, 0, -100),
		Status: "Sent", GrandTotalBHD: 30, OutstandingBHD: 30,
	}).Error)
	require.NoError(t, app.db.Create(&Payment{
		Base: Base{ID: "agg-p1"}, InvoiceID: "agg-i1", InvoiceNumber: "AGG-INV-1",
		AmountBHD: 10, PaymentDate: now.AddDate(0, 0, -85), PaymentMethod: "Cash",
		DaysToPayment: 45, IdempotencyKey: "agg-pay-1",
	}).Error)

	// c2 has a fully-paid invoice only: no outstanding, no orders. (PH's
	// dispute-count branch is retained in the port but dormant here — OSS's
	// customer-invoice CHECK constraint has no Dispute/Disputed status.)
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "agg-i2"}, InvoiceNumber: "AGG-INV-2", CustomerID: "agg-c2",
		InvoiceDate: now.AddDate(0, 0, -10), Status: "Paid", GrandTotalBHD: 20, OutstandingBHD: 0,
	}).Error)

	res, err := app.RecomputeAllCustomerAggregates()
	require.NoError(t, err)
	require.Equal(t, 2, res["customers_processed"])
	require.Equal(t, 2, res["customers_updated"])
	require.Equal(t, 1, res["with_orders"])
	require.InDelta(t, 200.0, res["total_orders_value"].(float64), 0.001)
	require.InDelta(t, 30.0, res["total_outstanding"].(float64), 0.001)
	require.Equal(t, 2, res["active_raw"])
	require.Equal(t, 2, res["active_canonical"])

	var c1 CustomerMaster
	require.NoError(t, app.db.First(&c1, "id = ?", "agg-c1").Error)
	require.Equal(t, 2, c1.TotalOrdersCount)
	require.InDelta(t, 200.0, c1.TotalOrdersValue, 0.001)
	require.InDelta(t, 100.0, c1.AvgOrderValue, 0.001)
	require.InDelta(t, 30.0, c1.OutstandingBHD, 0.001)
	require.GreaterOrEqual(t, c1.OverdueDays, 99)
	require.Equal(t, "High", c1.ARRiskTier, "100 days overdue with outstanding must tier High")
	require.InDelta(t, 45.0, c1.AvgPaymentDays, 0.01)
	require.NotNil(t, c1.LastOrderDate)

	var c2 CustomerMaster
	require.NoError(t, app.db.First(&c2, "id = ?", "agg-c2").Error)
	require.Equal(t, 0, c2.DisputeCount)
	require.InDelta(t, 0.0, c2.OutstandingBHD, 0.001)
	require.Equal(t, "Low", c2.ARRiskTier)
	require.Equal(t, 0, c2.TotalOrdersCount)

	// Idempotent: a second run must land on identical numbers.
	_, err = app.RecomputeAllCustomerAggregates()
	require.NoError(t, err)
	var again CustomerMaster
	require.NoError(t, app.db.First(&again, "id = ?", "agg-c1").Error)
	require.Equal(t, c1.TotalOrdersCount, again.TotalOrdersCount)
	require.InDelta(t, c1.TotalOrdersValue, again.TotalOrdersValue, 0.001)
}

func TestRecomputeAllCustomerAggregates_RequiresPermission(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CustomerMaster{}))

	setDataQualityRole(app, "user-1", "sales", `["customers:view"]`, "Sales")
	_, err := app.RecomputeAllCustomerAggregates()
	require.Error(t, err, "customers:edit must gate the recompute")

	setDataQualityRole(app, "admin-1", "admin", `["*"]`, "Admin")
	_, err = app.RecomputeAllCustomerAggregates()
	require.NoError(t, err)
}

func TestRecomputeCustomerPrediction_QualityGateAndPersist(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&CustomerMaster{}, &Order{}, &Invoice{}, &Payment{}, &PredictionRecord{}))

	require.NoError(t, app.db.Create(&CustomerMaster{
		Base: Base{ID: "pred-c1"}, CustomerID: "PRED-C1", CustomerCode: "PREDC1",
		BusinessName: "Gamma Utilities", Status: "Active", RelationYears: 4,
	}).Error)

	// QUALITY GATE: no payment history → (nil, nil), never a guessed grade.
	record, err := app.RecomputeCustomerPrediction("pred-c1")
	require.NoError(t, err)
	require.Nil(t, record, "no payment signal must yield no prediction")

	// Real history: invoice + payment with positive days-to-pay, plus an order.
	now := time.Now()
	require.NoError(t, app.db.Create(&Invoice{
		Base: Base{ID: "pred-i1"}, InvoiceNumber: "PRED-INV-1", CustomerID: "pred-c1",
		InvoiceDate: now.AddDate(0, 0, -60), Status: "Paid", GrandTotalBHD: 500,
	}).Error)
	require.NoError(t, app.db.Create(&Payment{
		Base: Base{ID: "pred-p1"}, InvoiceID: "pred-i1", InvoiceNumber: "PRED-INV-1",
		AmountBHD: 500, PaymentDate: now.AddDate(0, 0, -30), PaymentMethod: "Bank Transfer",
		DaysToPayment: 30, IdempotencyKey: "pred-pay-1",
	}).Error)
	require.NoError(t, app.db.Create(&Order{
		Base: Base{ID: "pred-o1"}, OrderNumber: "PRED-ORD-1", CustomerID: "pred-c1",
		OrderDate: now.AddDate(0, 0, -70), Status: "Delivered", GrandTotalBHD: 500,
	}).Error)

	record, err = app.RecomputeCustomerPrediction("pred-c1")
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, "pred-c1", record.CustomerID)
	require.Equal(t, "Gamma Utilities", record.CustomerName)
	require.NotEmpty(t, record.Grade)

	var persisted []PredictionRecord
	require.NoError(t, app.db.Where("customer_id = ?", "pred-c1").Find(&persisted).Error)
	require.Len(t, persisted, 1, "the prediction must be persisted")

	// Resolution accepts the alternate business identifier too.
	record, err = app.RecomputeCustomerPrediction("PRED-C1")
	require.NoError(t, err)
	require.NotNil(t, record)

	// Unknown customer errors; blank id is invalid input.
	_, err = app.RecomputeCustomerPrediction("missing")
	require.ErrorContains(t, err, "customer not found")
	_, err = app.RecomputeCustomerPrediction("")
	require.Error(t, err)

	// predictions:create gates the explicit recompute path.
	setDataQualityRole(app, "user-1", "sales", `["customers:view"]`, "Sales")
	_, err = app.RecomputeCustomerPrediction("pred-c1")
	require.Error(t, err)
}
