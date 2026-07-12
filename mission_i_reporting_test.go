package main

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// MISSION I — BAND 2 REPORTING & PDF HARDENING (I-23, I-26, I-27, I-28)
// =============================================================================
// I-23  GetInvoicesByAgingBucket — due-date bucket drill-through, derived state.
// I-26  reporting RBAC gaps — GetReportData + PDF report generators gated.
// I-27  purchase-order DRAFT watermark for unapproved POs.
// I-28  invoice PDF bank-details block sourced from companyDocumentProfile.
// =============================================================================

// restrictedReportApp returns an App whose current user holds a role WITHOUT
// finance:view / reports:view — used to prove the report gates reject an
// unauthorised caller. db is nil deliberately: every gate under test runs its
// requirePermission BEFORE touching the database.
func restrictedReportApp() *App {
	return &App{
		currentUserID: "restricted-user",
		currentUser: &User{
			Base:     Base{ID: "restricted-user"},
			Username: "restricted",
			RoleName: "sales",
			Role: Role{
				Name:        "sales",
				DisplayName: "Sales",
				Permissions: `["dashboard:view"]`,
			},
		},
	}
}

// -----------------------------------------------------------------------------
// I-23 — GetInvoicesByAgingBucket
// -----------------------------------------------------------------------------

// TestGetInvoicesByAgingBucket_BoundariesAndDerivedState seeds invoices whose
// due dates straddle every bucket edge, plus a stored-"Sent" past-due invoice
// (which must surface as derived Overdue and still be collectible), plus
// terminal invoices that must be excluded. It then asserts the drill-through
// buckets and confirms they reconcile to the aggregate GetPaymentAgingReport.
func TestGetInvoicesByAgingBucket_BoundariesAndDerivedState(t *testing.T) {
	app := setupPaymentTestApp(t)
	now := time.Now()

	// Bucket edges (days overdue relative to now). grand == outstanding == open.
	makeMissionIInvoice(t, app, "AG-CUR", "Sent", 100, 100, now.AddDate(0, 0, 10))     // future → current
	makeMissionIInvoice(t, app, "AG-1_30a", "Sent", 200, 200, now.AddDate(0, 0, -1))   // 1 → 1_30 (derived Overdue)
	makeMissionIInvoice(t, app, "AG-1_30b", "Sent", 300, 300, now.AddDate(0, 0, -30))  // 30 → 1_30 (edge)
	makeMissionIInvoice(t, app, "AG-31_60a", "Sent", 400, 400, now.AddDate(0, 0, -31)) // 31 → 31_60
	makeMissionIInvoice(t, app, "AG-31_60b", "Sent", 500, 500, now.AddDate(0, 0, -60)) // 60 → 31_60 (edge)
	makeMissionIInvoice(t, app, "AG-61_90a", "Sent", 600, 600, now.AddDate(0, 0, -61)) // 61 → 61_90
	makeMissionIInvoice(t, app, "AG-61_90b", "Sent", 700, 700, now.AddDate(0, 0, -90)) // 90 → 61_90 (edge)
	makeMissionIInvoice(t, app, "AG-OVER", "Sent", 800, 800, now.AddDate(0, 0, -91))   // 91 → over_90

	// Excluded: Draft is a closed workflow status (not collectible) even with an
	// open balance; Paid has a zero balance so the open-balance query drops it.
	makeMissionIInvoice(t, app, "AG-DRAFT", "Draft", 900, 900, now.AddDate(0, 0, -45))
	makeMissionIInvoice(t, app, "AG-PAID", "Paid", 1000, 0, now.AddDate(0, 0, -45))

	type exp struct {
		total int
		bhd   float64
	}
	cases := map[string]exp{
		"current": {1, 100},
		"1_30":    {2, 500},
		"31_60":   {2, 900},
		"61_90":   {2, 1300},
		"over_90": {1, 800},
		"0_30":    {3, 600},  // composite current (1) + 1_30 (2)
		"all":     {8, 3600}, // every collectible invoice, terminal excluded
	}
	for bucket, want := range cases {
		res, err := app.GetInvoicesByAgingBucket(bucket, 50, 0)
		require.NoError(t, err, "bucket %s", bucket)
		assert.Equal(t, want.total, res.Total, "bucket %s count", bucket)
		assert.InDelta(t, want.bhd, res.TotalBHD, 0.0005, "bucket %s total BHD", bucket)
	}

	// Derived state: the two past-due "Sent" invoices in 1_30 must display as
	// Overdue (recomputed on read, not the stored column).
	oneThirty, err := app.GetInvoicesByAgingBucket("1_30", 50, 0)
	require.NoError(t, err)
	overdueCount := 0
	for _, inv := range oneThirty.Invoices {
		if inv.Status == "Overdue" {
			overdueCount++
		}
	}
	assert.Equal(t, 2, overdueCount, "past-due open invoices must surface as derived Overdue")

	// Reconciliation: the drill-through buckets must sum to the aggregate report.
	report, err := app.GetPaymentAgingReport()
	require.NoError(t, err)
	assert.InDelta(t, 3600, report.GrandTotal, 0.0005, "aggregate grand total")
	assert.InDelta(t, cases["current"].bhd, report.TotalCurrent, 0.0005)
	assert.InDelta(t, cases["1_30"].bhd, report.TotalDays1To30, 0.0005)
	assert.InDelta(t, cases["31_60"].bhd, report.TotalDays31To60, 0.0005)
	assert.InDelta(t, cases["61_90"].bhd, report.TotalDays61To90, 0.0005)
	assert.InDelta(t, cases["over_90"].bhd, report.TotalOver90Days, 0.0005)
}

// TestGetInvoicesByAgingBucket_Pagination checks that Total/TotalBHD reflect the
// whole bucket while Invoices is only the requested page.
func TestGetInvoicesByAgingBucket_Pagination(t *testing.T) {
	app := setupPaymentTestApp(t)
	now := time.Now()
	for i := 0; i < 8; i++ {
		makeMissionIInvoice(t, app, uuid.New().String(), "Sent", 100, 100, now.AddDate(0, 0, -95))
	}

	page, err := app.GetInvoicesByAgingBucket("all", 3, 0)
	require.NoError(t, err)
	assert.Equal(t, 8, page.Total, "Total is the full bucket count, pre-pagination")
	assert.InDelta(t, 800, page.TotalBHD, 0.0005, "TotalBHD spans the whole bucket")
	assert.Len(t, page.Invoices, 3, "page respects the limit")

	tail, err := app.GetInvoicesByAgingBucket("all", 3, 6)
	require.NoError(t, err)
	assert.Len(t, tail.Invoices, 2, "final page returns the remainder")

	past, err := app.GetInvoicesByAgingBucket("all", 3, 100)
	require.NoError(t, err)
	assert.Empty(t, past.Invoices, "offset beyond the set returns no rows")
	assert.Equal(t, 8, past.Total)
}

// TestGetInvoicesByAgingBucket_InvalidBucket rejects an unknown bucket key.
func TestGetInvoicesByAgingBucket_InvalidBucket(t *testing.T) {
	app := setupPaymentTestApp(t)
	_, err := app.GetInvoicesByAgingBucket("last_century", 50, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid aging bucket")
}

// TestAgingBucketForDueDate_Edges locks the shared helper's boundaries.
func TestAgingBucketForDueDate_Edges(t *testing.T) {
	now := time.Now()
	cases := []struct {
		days   int
		bucket string
	}{
		{-5, "current"}, {0, "current"},
		{1, "1_30"}, {30, "1_30"},
		{31, "31_60"}, {60, "31_60"},
		{61, "61_90"}, {90, "61_90"},
		{91, "over_90"}, {365, "over_90"},
	}
	for _, c := range cases {
		got := agingBucketForDueDate(now.AddDate(0, 0, -c.days), now)
		assert.Equalf(t, c.bucket, got, "%d days overdue", c.days)
	}
}

// -----------------------------------------------------------------------------
// I-26 — reporting RBAC gates
// -----------------------------------------------------------------------------

func TestGetInvoicesByAgingBucket_RejectsUnauthorized(t *testing.T) {
	app := restrictedReportApp()
	_, err := app.GetInvoicesByAgingBucket("all", 50, 0)
	require.Error(t, err, "caller without finance:view must be rejected")
}

func TestGetReportData_RejectsUnauthorized(t *testing.T) {
	app := restrictedReportApp()
	_, err := app.GetReportData("financial", "month")
	require.Error(t, err, "financial report data requires finance:view")
}

func TestGenerateReports_RejectUnauthorized(t *testing.T) {
	app := restrictedReportApp()

	_, err := app.GenerateDashboardReport()
	require.Error(t, err, "dashboard report requires reports:view")

	_, err = app.GeneratePredictionHistoryReport(10)
	require.Error(t, err, "prediction history report requires reports:view")

	_, err = app.GenerateCustomer360Report("CUST-1")
	require.Error(t, err, "customer 360 report requires reports:view")
}

// TestReportGates_AllowAdmin confirms the new gates do not block an authorised
// (wildcard) caller — GetInvoicesByAgingBucket runs to completion for admin.
func TestReportGates_AllowAdmin(t *testing.T) {
	app := setupPaymentTestApp(t) // admin ["*"]
	_, err := app.GetInvoicesByAgingBucket("all", 50, 0)
	require.NoError(t, err, "authorised caller must pass the finance gate")
}

// -----------------------------------------------------------------------------
// I-27 — purchase-order DRAFT watermark
// -----------------------------------------------------------------------------

// TestDraftWatermark_EmitsDraftText proves the watermark helper actually renders
// the "DRAFT" glyphs into the page content stream (compression disabled so the
// text is inspectable in the raw PDF bytes).
func TestDraftWatermark_EmitsDraftText(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.AddPage()
	drawDraftWatermark(pdf)

	var buf bytes.Buffer
	require.NoError(t, pdf.Output(&buf))
	assert.Contains(t, buf.String(), "DRAFT", "unapproved-PO watermark text must be emitted")
	require.NoError(t, pdf.Error(), "watermark transforms must leave the PDF in a valid state")
}

// TestGeneratePurchaseOrderPDF_DraftPath drives the real generator for an
// unapproved (Draft) PO and an approved one; both must render without error,
// exercising the approvalPending → drawDraftWatermark branch and its absence.
func TestGeneratePurchaseOrderPDF_DraftPath(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&PurchaseOrder{}, &PurchaseOrderItem{}))

	supplier := SupplierMaster{Base: Base{ID: uuid.New().String()}, SupplierName: "Rhine Instruments GmbH"}
	require.NoError(t, app.db.Create(&supplier).Error)

	makePO := func(number, status string) string {
		po := PurchaseOrder{
			Base:         Base{ID: uuid.New().String()},
			PONumber:     number,
			PODate:       time.Now(),
			SupplierID:   supplier.ID,
			SupplierName: supplier.SupplierName,
			Status:       status,
			Currency:     "BHD",
			ExchangeRate: 1,
			VATAmount:    100,
			TotalBHD:     1100,
			Items: []PurchaseOrderItem{
				{Description: "Pressure transmitter", Quantity: 4, UnitPriceBHD: 250, TotalBHD: 1000},
			},
		}
		require.NoError(t, app.db.Create(&po).Error)
		return po.ID
	}

	draftID := makePO("PO-I27-DRAFT", "Draft")
	path, err := app.GeneratePurchaseOrderPDF(draftID)
	require.NoError(t, err, "Draft PO PDF (with watermark) must render")
	require.FileExists(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })

	approvedID := makePO("PO-I27-APPROVED", "Approved")
	path2, err := app.GeneratePurchaseOrderPDF(approvedID)
	require.NoError(t, err, "approved PO PDF (no watermark) must render")
	require.FileExists(t, path2)
	t.Cleanup(func() { _ = os.Remove(path2) })
}

// -----------------------------------------------------------------------------
// I-28 — invoice bank-details block from companyDocumentProfile
// -----------------------------------------------------------------------------

// TestInvoiceBankBlock_SourcedFromProfile confirms the profile the invoice PDF
// reads its bank block from is populated for a branded division, and that an
// invoice tagged to that division renders through the profile bank-block branch.
func TestInvoiceBankBlock_SourcedFromProfile(t *testing.T) {
	profile := companyDocumentProfile("Beacon Controls")
	require.NotEmpty(t, profile.BankDetails, "branded division profile must carry bank details for the invoice block")
	assert.Contains(t, profile.BankDetails[0], "IBAN", "profile bank line must be a full bank-details string")

	app := setupTestApp(t)
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)

	customer := CustomerMaster{Base: Base{ID: uuid.New().String()}, BusinessName: "Nimbus Controls", CustomerCode: "NC-28", CustomerID: "NC-28", Status: "Active"}
	require.NoError(t, app.db.Create(&customer).Error)

	invoice := Invoice{
		Base:          Base{ID: uuid.New().String()},
		InvoiceNumber: "INV-I28-9001", InvoiceDate: now, DueDate: now.AddDate(0, 0, 30),
		CustomerID: customer.ID, CustomerName: customer.BusinessName,
		Division: "Beacon Controls",
		Status:   "Draft", SubtotalBHD: 1000, VATPercent: 10, VATBHD: 100,
		GrandTotalBHD: 1100, OutstandingBHD: 1100,
	}
	require.NoError(t, app.db.Create(&invoice).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		InvoiceID: invoice.ID, LineNumber: 1,
		Description: "Conductivity sensor", Quantity: 4, Rate: 250, TotalBHD: 1000,
	}).Error)

	path, err := app.GenerateInvoicePDF(invoice.ID)
	require.NoError(t, err, "invoice PDF must render the profile bank-details block without error")
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })
}
