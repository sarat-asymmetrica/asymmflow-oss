package main

// INTEG residue campaign — Wave R2: deferred Go persistence tests for bindings
// that were WIRED + type-verified in earlier waves but lacked a focused Go
// persistence test. House style mirrors the integ_*_hotzone_test.go files.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// R2 — FinalizeBookBankReconciliation (adjacent to the tested FinalizeReconciliation).
func TestIntegR2_FinalizeBookBankReconciliation(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BookBankReconciliation{}), "migrate book-bank recon")

	// A balanced (zero-difference) recon can be finalized.
	recon := BookBankReconciliation{
		Base:          Base{ID: uuid.New().String()},
		BankAccountID: "bank-r2",
		Difference:    0,
		IsReconciled:  false,
	}
	require.NoError(t, app.db.Create(&recon).Error)

	// The client-passed user is IGNORED; the server records the session identity.
	require.NoError(t, app.FinalizeBookBankReconciliation(recon.ID, "client-supplied-ignored"))

	var stored BookBankReconciliation
	require.NoError(t, app.db.First(&stored, "id = ?", recon.ID).Error)
	require.True(t, stored.IsReconciled, "finalize sets is_reconciled")
	require.Equal(t, "test-user", stored.ReconciledBy, "reviewer is the session identity, not the client value")
	require.NotNil(t, stored.ReconciledAt, "reconciled_at stamped")

	// Idempotency guard: a second finalize is refused.
	require.Error(t, app.FinalizeBookBankReconciliation(recon.ID, ""), "already-reconciled is refused")

	// A non-zero difference cannot be finalized.
	skewed := BookBankReconciliation{Base: Base{ID: uuid.New().String()}, BankAccountID: "bank-r2", Difference: 5.000}
	require.NoError(t, app.db.Create(&skewed).Error)
	require.Error(t, app.FinalizeBookBankReconciliation(skewed.ID, ""), "non-zero difference is refused")
	var skewedStored BookBankReconciliation
	require.NoError(t, app.db.First(&skewedStored, "id = ?", skewed.ID).Error)
	require.False(t, skewedStored.IsReconciled, "a refused finalize writes nothing")
}

// R2 — DeleteRFQWithCascade: cascade=false errors when links exist (nothing
// deleted); cascade=true removes the RFQ + its linked costings/offers/items.
func TestIntegR2_DeleteRFQWithCascade(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&RFQData{}, &CostingSheetData{}, &Offer{}, &OfferItem{}, &RFQComment{}), "migrate rfq + links")

	rfq := RFQData{ID: 4201, RFQNumber: "R2-4201", Client: "Synthetic Client"}
	require.NoError(t, app.db.Create(&rfq).Error)
	require.NoError(t, app.db.Create(&CostingSheetData{RFQID: rfq.ID, RFQName: "R2 costing"}).Error)
	offer := Offer{Base: Base{ID: uuid.New().String()}, RFQID: "4201", OfferNumber: "OFR-R2-4201", Stage: "Quoted"}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{Base: Base{ID: uuid.New().String()}, OfferID: offer.ID, Description: "line"}).Error)

	// --- cascade=false with links → error, nothing deleted. ---
	res, err := app.DeleteRFQWithCascade(rfq.ID, false)
	require.Error(t, err, "refuses to orphan linked records without cascade")
	require.NotNil(t, res)
	require.Equal(t, 1, res.LinkedCostingSheets)
	require.Equal(t, 1, res.LinkedOffers)
	var rfqCount int64
	require.NoError(t, app.db.Model(&RFQData{}).Where("id = ?", rfq.ID).Count(&rfqCount).Error)
	require.EqualValues(t, 1, rfqCount, "the RFQ still exists after a refused delete")

	// --- cascade=true → RFQ + linked costings/offers/items all removed. ---
	res2, err := app.DeleteRFQWithCascade(rfq.ID, true)
	require.NoError(t, err)
	require.Equal(t, 1, res2.DeletedCostingSheets)
	require.Equal(t, 1, res2.DeletedOffers)

	require.NoError(t, app.db.Model(&RFQData{}).Where("id = ?", rfq.ID).Count(&rfqCount).Error)
	require.EqualValues(t, 0, rfqCount, "the RFQ is gone")
	var costingCount, offerCount, itemCount int64
	require.NoError(t, app.db.Model(&CostingSheetData{}).Where("rfq_id = ?", rfq.ID).Count(&costingCount).Error)
	require.NoError(t, app.db.Model(&Offer{}).Where("rfq_id = ?", "4201").Count(&offerCount).Error)
	require.NoError(t, app.db.Model(&OfferItem{}).Where("offer_id = ?", offer.ID).Count(&itemCount).Error)
	require.EqualValues(t, 0, costingCount, "linked costings cascade-deleted")
	require.EqualValues(t, 0, offerCount, "linked offers cascade-deleted")
	require.EqualValues(t, 0, itemCount, "linked offer items cascade-deleted")
}

// R2 — Import two-phase: nothing persists on Preview; Confirm persists exactly
// once (then the preview is consumed); Discard drops a preview so a later
// Confirm writes nothing. Exercised via the package-level preview store (the
// dialog half of Preview cannot run headlessly, but the persistence guarantee
// lives entirely in Confirm/Discard).
func TestIntegR2_BankStatementImportTwoPhase(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&BankStatement{}, &BankStatementLine{}), "migrate bank statements")

	newPreview := func(num string) *BankStatement {
		stmt := &BankStatement{
			Base:            Base{ID: uuid.New().String()},
			BankAccountID:   "bank-r2",
			StatementNumber: num,
			Status:          "Imported",
		}
		bankStatementPreviewMu.Lock()
		bankStatementPreviewStore[stmt.ID] = &bankStatementImportPreview{Statement: stmt, FilePath: "synthetic.csv"}
		bankStatementPreviewMu.Unlock()
		return stmt
	}

	// --- Preview stages nothing in the DB. ---
	confirmMe := newPreview("R2-STMT-CONFIRM")
	var count int64
	require.NoError(t, app.db.Model(&BankStatement{}).Where("id = ?", confirmMe.ID).Count(&count).Error)
	require.EqualValues(t, 0, count, "a previewed-only statement is not persisted")

	// --- Confirm persists it, once. ---
	out, err := app.ConfirmBankStatementImport(confirmMe.ID)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NoError(t, app.db.Model(&BankStatement{}).Where("id = ?", confirmMe.ID).Count(&count).Error)
	require.EqualValues(t, 1, count, "confirm persists the statement")
	// The preview is consumed — a second confirm cannot double-write.
	_, err = app.ConfirmBankStatementImport(confirmMe.ID)
	require.Error(t, err, "a consumed preview cannot be confirmed again")

	// --- Discard drops a preview so Confirm writes nothing. ---
	discardMe := newPreview("R2-STMT-DISCARD")
	app.DiscardBankStatementImportPreview(discardMe.ID)
	_, err = app.ConfirmBankStatementImport(discardMe.ID)
	require.Error(t, err, "a discarded preview cannot be confirmed")
	require.NoError(t, app.db.Model(&BankStatement{}).Where("id = ?", discardMe.ID).Count(&count).Error)
	require.EqualValues(t, 0, count, "a discarded preview persists nothing")
}

// R2 — focused REJECT round-trip for ReviewDeleteApprovalRequest. The approve
// path is covered in app_test.go; the employee-archive binding's approve AND
// reject round-trips are covered in employee_archive_service_test.go
// (TestReviewEmployeeArchiveRequest_ApproveArchives / _RejectDoesNotArchive),
// so this fills the one remaining gap: delete-approval reject flips a pending
// request to rejected without deleting the target.
func TestIntegR2_ReviewDeleteApprovalReject(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&DeleteApprovalRequest{}), "migrate delete-approval requests")

	delReq := DeleteApprovalRequest{
		Base:        Base{ID: uuid.New().String()},
		EntityType:  "customer",
		EntityID:    uuid.New().String(),
		EntityLabel: "Synthetic Customer",
		Status:      "pending",
	}
	require.NoError(t, app.db.Create(&delReq).Error)

	reviewedDel, err := app.ReviewDeleteApprovalRequest(delReq.ID, "reject", "not a duplicate after all")
	require.NoError(t, err)
	require.Equal(t, "rejected", reviewedDel.Status, "delete-approval reject flips status")

	// The rejected request persists as rejected (not deleted, not still pending).
	var stored DeleteApprovalRequest
	require.NoError(t, app.db.First(&stored, "id = ?", delReq.ID).Error)
	require.Equal(t, "rejected", stored.Status)
}
