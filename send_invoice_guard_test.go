package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// PH convergence 1-HOLLOW (PH MON-007): a Draft invoice with no line items
// must be refused at send — sending it renders a blank line-item table
// against a non-zero total.
func TestSendCustomerInvoice_RefusesHollowInvoice(t *testing.T) {
	a := setupTestApp(t)

	hollow := Invoice{InvoiceNumber: "INV-26-9001", Status: "Draft", CustomerName: "Wasela Trading", GrandTotalBHD: 525}
	require.NoError(t, a.db.Create(&hollow).Error)

	err := a.SendCustomerInvoice(hollow.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no line items")

	var after Invoice
	require.NoError(t, a.db.First(&after, "id = ?", hollow.ID).Error)
	require.Equal(t, "Draft", after.Status, "hollow invoice must stay Draft")

	full := Invoice{InvoiceNumber: "INV-26-9002", Status: "Draft", CustomerName: "Wasela Trading", GrandTotalBHD: 105}
	require.NoError(t, a.db.Create(&full).Error)
	require.NoError(t, a.db.Create(&DBInvoiceItem{InvoiceID: full.ID, Description: "Level switch", Quantity: 1, Rate: 100, TotalBHD: 100}).Error)

	require.NoError(t, a.SendCustomerInvoice(full.ID))
	after = Invoice{}
	require.NoError(t, a.db.First(&after, "id = ?", full.ID).Error)
	require.Equal(t, "Sent", after.Status)
}
