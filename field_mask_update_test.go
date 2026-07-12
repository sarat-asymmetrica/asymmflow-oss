package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// PH convergence 1-FM (PH INT-001): partial update payloads must not wipe
// server-owned / workflow fields. GORM Save writes zero-values too, so each
// update method load-then-overlays the protected fields from the stored row.
func TestUpdateSupplierInvoice_PreservesServerOwnedFields(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&SupplierInvoice{}))

	approvedAt := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	inv := SupplierInvoice{
		InvoiceNumber: "SINV-26-1001",
		SupplierName:  "Meridian Instruments GmbH",
		Status:        "Approved",
		Currency:      "BHD",
		ExchangeRate:  1,
		TotalForeign:  500,
		ApprovedBy:    "ops-manager",
		ApprovedAt:    &approvedAt,
		POMatchOK:     true,
		GRNMatchOK:    true,
		OCRDocumentID: "ocr-doc-77",
		OCRConfidence: 0.93,
	}
	require.NoError(t, a.db.Create(&inv).Error)

	// A partial payload: only ID + a note-level edit, everything else zero.
	partial := SupplierInvoice{Status: "Approved", Currency: "BHD", ExchangeRate: 1, TotalForeign: 500, InvoiceNumber: "SINV-26-1001", SupplierName: "Meridian Instruments GmbH"}
	partial.ID = inv.ID

	updated, err := a.UpdateSupplierInvoice(partial)
	require.NoError(t, err)
	require.Equal(t, "ops-manager", updated.ApprovedBy, "approval trail must survive a partial payload")
	require.NotNil(t, updated.ApprovedAt)
	require.True(t, updated.POMatchOK)
	require.True(t, updated.GRNMatchOK)
	require.Equal(t, "ocr-doc-77", updated.OCRDocumentID)
	require.Equal(t, 0.93, updated.OCRConfidence)
}

func TestUpdateGRN_PreservesQCFields(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&GoodsReceivedNote{}))

	qcDate := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	grn := GoodsReceivedNote{GRNNumber: "GRN-26-0042", QCStatus: "Passed", QCNotes: "All items within tolerance", QCDate: &qcDate, QCBy: "qc-inspector"}
	require.NoError(t, a.db.Create(&grn).Error)

	partial := GoodsReceivedNote{GRNNumber: "GRN-26-0042"}
	partial.ID = grn.ID

	updated, err := a.UpdateGRN(partial)
	require.NoError(t, err)
	require.Equal(t, "Passed", updated.QCStatus, "QC workflow fields must survive a header edit")
	require.Equal(t, "All items within tolerance", updated.QCNotes)
	require.NotNil(t, updated.QCDate)
	require.Equal(t, "qc-inspector", updated.QCBy)
}

func TestUpdateCustomerContact_PreservesCustomerLink(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&CustomerContact{}))

	customer := CustomerMaster{BusinessName: "Wasela Trading"}
	require.NoError(t, a.db.Create(&customer).Error)
	contact := CustomerContact{CustomerID: customer.ID, ContactName: "Fatima K.", Email: "fatima@example.test"}
	require.NoError(t, a.db.Create(&contact).Error)

	partial := CustomerContact{ContactName: "Fatima Khalid", Email: "fatima@example.test"}
	partial.ID = contact.ID

	updated, err := a.UpdateCustomerContact(partial)
	require.NoError(t, err)
	require.Equal(t, customer.ID, updated.CustomerID, "blank CustomerID in a partial payload must not orphan the contact")
	require.Equal(t, "Fatima Khalid", updated.ContactName)
}
