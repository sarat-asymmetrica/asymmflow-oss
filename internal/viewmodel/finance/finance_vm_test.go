package finance

import (
	"testing"
	"time"

	domain "ph_holdings_app/pkg/finance"
)

func TestBuildInvoiceListVMFormatsDisplayValues(t *testing.T) {
	now := time.Now()
	invoices := []domain.Invoice{
		{
			Base:           domain.Base{ID: "inv-1"},
			InvoiceNumber:  "INV-001",
			CustomerName:   "Gulf Smelting",
			InvoiceDate:    now,
			DueDate:        now.AddDate(0, 0, 30), // not yet due
			Status:         "Sent",
			GrandTotalBHD:  1234.5,
			OutstandingBHD: 234.5,
		},
		{
			Base:           domain.Base{ID: "inv-2"},
			InvoiceNumber:  "INV-002",
			CustomerName:   "NGA",
			InvoiceDate:    now.AddDate(0, 0, -60),
			DueDate:        now.AddDate(0, 0, -30), // past due
			Status:         "Overdue",
			GrandTotalBHD:  500,
			OutstandingBHD: 500,
		},
	}

	got := BuildInvoiceListVM(invoices, 1, 20)
	if got.Summary.TotalOutstanding != "BHD 734.50" {
		t.Fatalf("unexpected outstanding total: %s", got.Summary.TotalOutstanding)
	}
	if got.Summary.OverdueCount != 1 || got.Summary.OverdueAmount != "BHD 500.00" {
		t.Fatalf("unexpected overdue summary: %#v", got.Summary)
	}
	firstTotal := got.Table.Rows[0].Fields["total"]
	if firstTotal != "BHD 1,234.50" {
		t.Fatalf("expected formatted total, got %#v", firstTotal)
	}
}

func TestBuildInvoiceDetailVMFormatsItemsAndActions(t *testing.T) {
	invoice := domain.Invoice{
		Base:           domain.Base{ID: "inv-1"},
		InvoiceNumber:  "INV-001",
		CustomerName:   "Gulf Smelting",
		InvoiceDate:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		Status:         "Sent",
		SubtotalBHD:    100,
		VATBHD:         10,
		GrandTotalBHD:  110,
		OutstandingBHD: 110,
		Items: []domain.DBInvoiceItem{{
			Base:        domain.Base{ID: "line-1"},
			LineNumber:  1,
			Description: "Pressure transmitter",
			Quantity:    2,
			Rate:        50,
			TotalBHD:    100,
			ProductCode: "PT-100",
		}},
	}

	got := BuildInvoiceDetailVM(invoice, nil)
	if got.InvoiceDate != "1 May 2026" || got.TotalDisplay != "BHD 110.00" {
		t.Fatalf("unexpected header formatting: %#v", got)
	}
	if got.Items[0].Quantity != "2" || got.Items[0].RateDisplay != "BHD 50.00" {
		t.Fatalf("unexpected item formatting: %#v", got.Items[0])
	}
	if !got.Actions[1].Enabled {
		t.Fatalf("record payment action should be enabled")
	}
}
