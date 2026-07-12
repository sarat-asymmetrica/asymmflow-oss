package posting

import "testing"

func TestBuildCoverageReport(t *testing.T) {
	report := BuildCoverageReport([]CoverageRow{
		{SourceType: SourceCustomerInvoice, Label: "Customer Invoices", Total: 10, Linked: 6, DraftEntries: 7},
		{SourceType: SourceCustomerPayment, Label: "Customer Payments", Total: 2, Linked: 2, DraftEntries: 2},
	})
	if report.Total != 12 || report.Linked != 8 || report.Missing != 4 || report.DraftEntries != 9 {
		t.Fatalf("unexpected totals: %+v", report)
	}
	if report.IsComplete {
		t.Fatal("report should not be complete")
	}
	if report.Rows[0].Missing != 4 || report.Rows[0].IsComplete {
		t.Fatalf("unexpected row: %+v", report.Rows[0])
	}
}
