package main

// C4: proforma invoice path — orderless creation on a dedicated PF- sequence,
// exclusion from AR aging/VAT until converted, and guarded conversion into a
// real, fiscally-numbered invoice (mirrors MarkOfferWon's guard pattern,
// app_sales_pipeline.go).

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// invoiceSeqFromNumber extracts the trailing sequence digits from a
// "PREFIX-YYYYMMDD-NNNN" style document number.
func invoiceSeqFromNumber(t *testing.T, number string) int {
	t.Helper()
	parts := strings.Split(number, "-")
	require.Len(t, parts, 3, "expected PREFIX-YYYYMMDD-NNNN format: %s", number)
	seq, err := strconv.Atoi(parts[2])
	require.NoError(t, err)
	return seq
}

func TestCreateProformaInvoiceManual_DoesNotConsumeInvoiceNumber(t *testing.T) {
	app := setupTestApp(t)
	customerID := seedTestCustomer(t, app.db, "Proforma Customer")

	before, err := app.GenerateInvoiceNumber()
	require.NoError(t, err)

	proforma, err := app.CreateProformaInvoiceManual(customerID, "Proforma Customer", []ProformaInvoiceItemInput{
		{Description: "Pressure gauge", Quantity: 2, Rate: 50},
	}, "")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(proforma.InvoiceNumber, "PF-"), "expected PF- prefix, got %s", proforma.InvoiceNumber)
	require.Equal(t, "Proforma", proforma.Status)
	require.Equal(t, 0.0, proforma.OutstandingBHD)
	require.InDelta(t, 100, proforma.SubtotalBHD, 0.001)
	require.InDelta(t, 10, proforma.VATBHD, 0.001)
	require.InDelta(t, 110, proforma.GrandTotalBHD, 0.001)

	after, err := app.GenerateInvoiceNumber()
	require.NoError(t, err)

	require.Equal(t, invoiceSeqFromNumber(t, before)+1, invoiceSeqFromNumber(t, after),
		"proforma creation must not consume an INV- sequence number")
}

func TestProformaInvoice_ExcludedFromAgingAndVAT(t *testing.T) {
	app := setupTestApp(t)
	customerID := seedTestCustomer(t, app.db, "Proforma Aging Customer")

	proforma, err := app.CreateProformaInvoiceManual(customerID, "Proforma Aging Customer", []ProformaInvoiceItemInput{
		{Description: "Flow meter", Quantity: 1, Rate: 500},
	}, "")
	require.NoError(t, err)

	// Backdate the due date so it WOULD read as badly overdue if aging didn't
	// exclude it outright.
	require.NoError(t, app.db.Model(&Invoice{}).Where("id = ?", proforma.ID).
		Update("due_date", time.Now().AddDate(0, 0, -60)).Error)

	aging, err := app.GetARAgingReport()
	require.NoError(t, err)
	for _, d := range aging.Details {
		require.NotEqual(t, proforma.ID, d.InvoiceID, "proforma must not appear in AR aging")
	}
	require.Equal(t, 0.0, aging.Total)

	start := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	end := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	vat, err := app.GetVATReconciliation(start, end)
	require.NoError(t, err)
	require.Equal(t, 0, vat.CustomerInvoices, "proforma must not be counted toward output VAT")
	require.Equal(t, 0.0, vat.OutputVAT)
}

func TestConvertProformaToInvoice(t *testing.T) {
	app := setupTestApp(t)
	customerID := seedTestCustomer(t, app.db, "Proforma Convert Customer")

	proforma, err := app.CreateProformaInvoiceManual(customerID, "Proforma Convert Customer", []ProformaInvoiceItemInput{
		{Description: "Level transmitter", Quantity: 1, Rate: 1000},
	}, "")
	require.NoError(t, err)
	require.Equal(t, 0.0, proforma.OutstandingBHD)

	converted, err := app.ConvertProformaToInvoice(proforma.ID, "PO-9001")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(converted.InvoiceNumber, "INV-"), "expected INV- prefix, got %s", converted.InvoiceNumber)
	require.NotEqual(t, proforma.InvoiceNumber, converted.InvoiceNumber)
	require.Equal(t, "Sent", converted.Status)
	require.InDelta(t, converted.GrandTotalBHD, converted.OutstandingBHD, 0.001)
	require.Equal(t, "PO-9001", converted.CustomerPONumber)

	var persisted Invoice
	require.NoError(t, app.db.First(&persisted, "id = ?", proforma.ID).Error)
	require.Equal(t, converted.InvoiceNumber, persisted.InvoiceNumber)
	require.Equal(t, "Sent", persisted.Status)
	require.InDelta(t, persisted.GrandTotalBHD, persisted.OutstandingBHD, 0.001)

	aging, err := app.GetARAgingReport()
	require.NoError(t, err)
	found := false
	for _, d := range aging.Details {
		if d.InvoiceID == proforma.ID {
			found = true
		}
	}
	require.True(t, found, "converted invoice must now appear in AR aging")
}

func TestConvertProformaToInvoice_RejectsNonProforma(t *testing.T) {
	app := setupTestApp(t)

	inv := Invoice{InvoiceNumber: "INV-26-9999", Status: "Draft", CustomerName: "Not A Proforma", GrandTotalBHD: 100}
	require.NoError(t, app.db.Create(&inv).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{InvoiceID: inv.ID, Description: "Item", Quantity: 1, Rate: 100, TotalBHD: 100}).Error)

	_, err := app.ConvertProformaToInvoice(inv.ID, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "only Proforma invoices can be converted")

	var after Invoice
	require.NoError(t, app.db.First(&after, "id = ?", inv.ID).Error)
	require.Equal(t, "Draft", after.Status, "rejected conversion must not mutate the invoice")
	require.Equal(t, "INV-26-9999", after.InvoiceNumber)
}
