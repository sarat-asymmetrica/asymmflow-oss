package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =============================================================================
// SUPPLIER AP CASH-CONTROL GATE TESTS — Mission G (Wave 4) + Wave 8 P5-1
// =============================================================================
// The parity audit found OSS let a supplier invoice be PAID while still
// "Pending" (never 3-way-matched, never approved) and APPROVED unless flagged
// "Discrepancy". Mission G tightened to: pay requires Approved or Verified;
// approve requires MatchStatus == Matched. Wave 8 P5-1 (user-ratified hybrid)
// tightened payment further: ONLY Approved is payable — "Verified" (clean
// match) must pass through the segregated ApproveSupplierInvoice step before
// cash leaves. These tests pin the current control.
// =============================================================================

func setupSupplierGateTestApp(t *testing.T) *App {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&SupplierInvoice{},
		&SupplierPayment{},
		&SupplierMaster{},
		&PurchaseOrder{},
		&PurchaseOrderItem{},
		&SupplierInvoiceItem{},
		&GoodsReceivedNote{},
		&GRNItem{},
		&ProductMaster{},
		&AuditLog{},
	))

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       false,
		startupImportStartTime: time.Now(),
		currentUserID:          "test-user",
		currentUser: &User{
			Base:     Base{ID: "test-user"},
			Username: "test-admin",
			RoleName: "admin",
			Role:     Role{Name: "admin", DisplayName: "Administrator", Permissions: `["*"]`},
		},
	}
	t.Cleanup(app.cache.Stop)
	return app
}

func makeSupplierInvoice(t *testing.T, app *App, number, status, matchStatus string, total float64) *SupplierInvoice {
	t.Helper()
	inv := &SupplierInvoice{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now(), CreatedBy: "creator-user"},
		SupplierID:    "SUP-1",
		SupplierName:  "Acme Supply WLL",
		InvoiceNumber: number,
		InvoiceDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		SubtotalBHD:   total,
		TotalBHD:      total,
		Status:        status,
		MatchStatus:   matchStatus,
	}
	require.NoError(t, app.db.Create(inv).Error)
	return inv
}

// TestSupplierInvoiceVerifiedStatusIsPersistable is a guard probe: a clean
// 3-way match writes Status="Verified", so that value MUST survive the schema's
// CHECK constraint on a from-zero AutoMigrated DB — otherwise the pay-gate that
// keys on "Verified" is unreachable. Regression pin for the fresh-provision
// constraint parity.
func TestSupplierInvoiceVerifiedStatusIsPersistable(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-VER-1", "Pending", "Pending", 100.000)

	inv.Status = "Verified"
	inv.MatchStatus = "Matched"
	err := app.db.Save(inv).Error
	require.NoError(t, err, "clean-match Status=Verified must persist (fresh-DB CHECK constraint must allow it)")

	var reloaded SupplierInvoice
	require.NoError(t, app.db.First(&reloaded, "id = ?", inv.ID).Error)
	assert.Equal(t, "Verified", reloaded.Status)
}

func TestRecordSupplierPayment_GateBlocksUnmatchedPending(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-PAY-1", "Pending", "Pending", 500.000)

	pay, err := app.RecordSupplierPayment(inv.ID, 100.000, "BHD", "Bank Transfer", "2026-01-15", "REF1", 1.0)
	require.Error(t, err)
	assert.Nil(t, pay)
	assert.Contains(t, err.Error(), "must be Approved")
}

func TestRecordSupplierPayment_RequiresExplicitApproval(t *testing.T) {
	// Wave 8 P5-1: an Approved invoice pays; a merely Verified one (clean match,
	// approval step skipped) is blocked with guidance to ApproveSupplierInvoice.
	app := setupSupplierGateTestApp(t)
	approved := makeSupplierInvoice(t, app, "SI-OK-Approved", "Approved", "Matched", 500.000)

	pay, err := app.RecordSupplierPayment(approved.ID, 100.000, "BHD", "Bank Transfer", "2026-01-15", "REF-A", 1.0)
	require.NoError(t, err)
	require.NotNil(t, pay)
	assert.Equal(t, 100.000, pay.AmountBHD)

	verified := makeSupplierInvoice(t, app, "SI-VER-ONLY", "Verified", "Matched", 500.000)
	pay, err = app.RecordSupplierPayment(verified.ID, 100.000, "BHD", "Bank Transfer", "2026-01-15", "REF-V", 1.0)
	require.Error(t, err)
	assert.Nil(t, pay)
	assert.Contains(t, err.Error(), "ApproveSupplierInvoice")
}

func TestRecordSupplierPayment_StillBlocksDiscrepancy(t *testing.T) {
	app := setupSupplierGateTestApp(t)
	inv := makeSupplierInvoice(t, app, "SI-DISC-1", "Verified", "Discrepancy", 500.000)

	pay, err := app.RecordSupplierPayment(inv.ID, 100.000, "BHD", "Bank Transfer", "2026-01-15", "REF1", 1.0)
	require.Error(t, err)
	assert.Nil(t, pay)
	assert.Contains(t, err.Error(), "discrepanc")
}

func TestApproveSupplierInvoice_RequiresMatched(t *testing.T) {
	app := setupSupplierGateTestApp(t)

	// Unmatched (Pending) invoice must not be approvable.
	pending := makeSupplierInvoice(t, app, "SI-APP-1", "Pending", "Pending", 500.000)
	err := app.ApproveSupplierInvoice(pending.ID, "approver-user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "3-way match")

	// A cleanly matched invoice with a distinct approver (SoD) approves.
	matched := makeSupplierInvoice(t, app, "SI-APP-2", "Verified", "Matched", 500.000)
	require.NoError(t, app.ApproveSupplierInvoice(matched.ID, "approver-user"))

	var reloaded SupplierInvoice
	require.NoError(t, app.db.First(&reloaded, "id = ?", matched.ID).Error)
	assert.Equal(t, "Approved", reloaded.Status)
}

// TestPerformThreeWayMatch_ZeroPricedPOLineFallsBackToStandardCost pins the
// Mission G wiring of the pkg/inventory reference-cost resolvers into
// PerformThreeWayMatch. A PO line with a 0 unit price previously escaped price
// validation entirely (the variance guard required poItem.UnitPriceBHD > 0);
// now it falls back to the product standard cost, so a real price gap is caught.
func TestPerformThreeWayMatch_ZeroPricedPOLineFallsBackToStandardCost(t *testing.T) {
	app := setupSupplierGateTestApp(t)

	// Product carries a standard cost of 50 BHD.
	require.NoError(t, app.db.Create(&ProductMaster{
		Base:            Base{ID: "PROD-1"},
		StandardCostBHD: 50.000,
	}).Error)

	// PO line has NO unit price (0) — the exact case that used to escape.
	po := &PurchaseOrder{Base: Base{ID: "PO-1"}, TotalBHD: 1000.000}
	require.NoError(t, app.db.Create(po).Error)
	require.NoError(t, app.db.Create(&PurchaseOrderItem{
		Base:            Base{ID: "POI-1"},
		PurchaseOrderID: "PO-1",
		ProductID:       "PROD-1",
		Quantity:        10,
		UnitPriceBHD:    0, // missing price
	}).Error)

	// Supplier invoice, BHD, one line at 100 BHD/unit — a 2x gap vs standard cost.
	inv := &SupplierInvoice{
		Base:            Base{ID: "SI-3WM-1", CreatedBy: "creator-user"},
		SupplierID:      "SUP-1",
		InvoiceNumber:   "SI-3WM-1",
		PurchaseOrderID: "PO-1",
		Currency:        "BHD",
		ExchangeRate:    1,
		TotalBHD:        1000.000,
		Status:          "Pending",
		MatchStatus:     "Pending",
	}
	require.NoError(t, app.db.Create(inv).Error)
	require.NoError(t, app.db.Create(&SupplierInvoiceItem{
		Base:              Base{ID: "SII-1"},
		SupplierInvoiceID: "SI-3WM-1",
		LineNumber:        1,
		UnitPrice:         100.000,
	}).Error)

	res, err := app.PerformThreeWayMatch("SI-3WM-1")
	require.NoError(t, err)
	assert.False(t, res.Matched)
	// The variance is measured against the resolved standard cost (50), not 0 —
	// proving the fallback is wired. Before the fix, a 0-priced PO line produced
	// no unit-price variance at all.
	assert.Contains(t, res.Reason, "unit price variance")
	assert.Contains(t, res.Reason, "PO=50.000 BHD")
}
