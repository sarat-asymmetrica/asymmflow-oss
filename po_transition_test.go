package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Wave 9.5 B5: UpdatePOStatus's validTransitions map uses the CANONICAL
// SPACED strings ("Partially Received", "Pending Approval") as both keys and
// allowed-value entries. These tests exercise legal/illegal transitions and
// specifically confirm that an unspaced variant ("PartiallyReceived") or a
// jump past the sanctioned chain is rejected, not silently accepted.
func TestUpdatePOStatus_TransitionValidation(t *testing.T) {
	cases := []struct {
		name     string
		from     string
		to       string
		totalBHD float64
		wantErr  bool
	}{
		// --- legal transitions ---
		{name: "draft to pending approval", from: "Draft", to: "Pending Approval", totalBHD: 100, wantErr: false},
		{name: "draft to sent under threshold", from: "Draft", to: "Sent", totalBHD: 100, wantErr: false},
		{name: "draft to cancelled", from: "Draft", to: "Cancelled", totalBHD: 100, wantErr: false},
		{name: "pending approval to approved", from: "Pending Approval", to: "Approved", totalBHD: 100, wantErr: false},
		{name: "approved to sent", from: "Approved", to: "Sent", totalBHD: 100, wantErr: false},
		{name: "sent to acknowledged", from: "Sent", to: "Acknowledged", totalBHD: 100, wantErr: false},
		{name: "sent to partially received (canonical spaced)", from: "Sent", to: "Partially Received", totalBHD: 100, wantErr: false},
		{name: "partially received to received", from: "Partially Received", to: "Received", totalBHD: 100, wantErr: false},

		// --- illegal transitions ---
		{name: "received is terminal", from: "Received", to: "Cancelled", totalBHD: 100, wantErr: true},
		{name: "cancelled is terminal", from: "Cancelled", to: "Draft", totalBHD: 100, wantErr: true},
		{name: "closed is terminal", from: "Closed", to: "Sent", totalBHD: 100, wantErr: true},
		{name: "unspaced canonical string rejected", from: "Sent", to: "PartiallyReceived", totalBHD: 100, wantErr: true},
		{name: "draft cannot jump straight to received", from: "Draft", to: "Received", totalBHD: 100, wantErr: true},
		{name: "draft cannot jump straight to acknowledged", from: "Draft", to: "Acknowledged", totalBHD: 100, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := setupTestApp(t)
			require.NoError(t, a.db.AutoMigrate(&PurchaseOrder{}, &PurchaseOrderItem{}))

			supplier := SupplierMaster{SupplierCode: "SUP-" + uuid.New().String()[:8], SupplierName: "Torrance Metering Co"}
			require.NoError(t, a.db.Create(&supplier).Error)

			po := PurchaseOrder{
				PONumber: "PO-TRANS-" + uuid.New().String()[:8], SupplierID: supplier.ID, SupplierName: supplier.SupplierName,
				Currency: "BHD", ExchangeRate: 1, Status: tc.from, PODate: time.Now(), TotalBHD: tc.totalBHD,
			}
			require.NoError(t, a.db.Create(&po).Error)

			err := a.UpdatePOStatus(po.ID, tc.to)

			var reloaded PurchaseOrder
			require.NoError(t, a.db.First(&reloaded, "id = ?", po.ID).Error)

			if tc.wantErr {
				require.Error(t, err, "expected transition %s -> %s to be rejected", tc.from, tc.to)
				require.Equal(t, tc.from, reloaded.Status, "status must not change on a rejected transition")
			} else {
				require.NoError(t, err, "expected transition %s -> %s to be accepted", tc.from, tc.to)
				require.Equal(t, tc.to, reloaded.Status)
			}
		})
	}
}

// The approval-threshold guards (P1 FIX) sit alongside the transition-map
// check inside UpdatePOStatus: a structurally legal Draft -> Sent/Approved
// transition is still blocked when the PO total exceeds the threshold.
func TestUpdatePOStatus_ApprovalThresholdBlocksLegalTransitionAboveLimit(t *testing.T) {
	a := setupTestApp(t)
	require.NoError(t, a.db.AutoMigrate(&PurchaseOrder{}, &PurchaseOrderItem{}))

	supplier := SupplierMaster{SupplierCode: "SUP-THRESH", SupplierName: "Torrance Metering Co"}
	require.NoError(t, a.db.Create(&supplier).Error)

	po := PurchaseOrder{
		PONumber: "PO-TRANS-THRESH", SupplierID: supplier.ID, SupplierName: supplier.SupplierName,
		Currency: "BHD", ExchangeRate: 1, Status: "Draft", PODate: time.Now(), TotalBHD: 9000,
	}
	require.NoError(t, a.db.Create(&po).Error)

	err := a.UpdatePOStatus(po.ID, "Sent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "approval threshold")

	var reloaded PurchaseOrder
	require.NoError(t, a.db.First(&reloaded, "id = ?", po.ID).Error)
	require.Equal(t, "Draft", reloaded.Status, "status must not change when threshold guard blocks the send")
}
