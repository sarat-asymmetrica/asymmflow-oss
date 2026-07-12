package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// PH convergence B4 (PH 0a96926): editing a Draft invoice's line items
// recomputes Subtotal/VAT/Grand/Outstanding server-side — header totals can
// no longer go stale (and the client's Amount field is ignored).
func TestUpdateCustomerInvoice_RecomputesTotalsFromLines(t *testing.T) {
	a := setupTestApp(t)

	inv := Invoice{
		InvoiceNumber: "INV-26-8001", Status: "Draft",
		InvoiceDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		VATPercent:  10, SubtotalBHD: 100, VATBHD: 10, GrandTotalBHD: 110, OutstandingBHD: 110,
	}
	require.NoError(t, a.db.Create(&inv).Error)

	edit := Invoice{Status: "Draft", GrandTotalBHD: 99999} // client Amount must be ignored
	edit.ID = inv.ID
	edit.Items = []DBInvoiceItem{
		{Description: "Flow transmitter", Quantity: 2, Rate: 150},
		{Description: "Commissioning", Quantity: 1, Rate: 75.5},
	}

	updated, err := a.UpdateCustomerInvoice(edit)
	require.NoError(t, err)
	require.Equal(t, 375.5, updated.SubtotalBHD)
	require.Equal(t, 37.55, updated.VATBHD)
	require.Equal(t, 413.05, updated.GrandTotalBHD, "grand total derived from lines, not the client payload")
	require.Equal(t, 413.05, updated.OutstandingBHD, "Draft = fully outstanding")
	require.Len(t, updated.Items, 2)
}

// PH convergence B4a (PH 70b05d2): a genuinely zero-rated Draft (VATBHD=0,
// grand==subtotal) keeps 0% on edit instead of being forced to the 10% default.
func TestUpdateCustomerInvoice_PreservesZeroRatedVAT(t *testing.T) {
	a := setupTestApp(t)

	inv := Invoice{
		InvoiceNumber: "INV-26-8002", Status: "Draft",
		InvoiceDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		VATPercent:  0, SubtotalBHD: 200, VATBHD: 0, GrandTotalBHD: 200, OutstandingBHD: 200,
	}
	require.NoError(t, a.db.Create(&inv).Error)

	edit := Invoice{Status: "Draft"}
	edit.ID = inv.ID
	edit.Items = []DBInvoiceItem{{Description: "Export shipment", Quantity: 4, Rate: 60}}

	updated, err := a.UpdateCustomerInvoice(edit)
	require.NoError(t, err)
	require.Equal(t, 240.0, updated.SubtotalBHD)
	require.Equal(t, 0.0, updated.VATBHD, "zero-rated invoice must stay zero-rated on edit")
	require.Equal(t, 240.0, updated.GrandTotalBHD)
}
