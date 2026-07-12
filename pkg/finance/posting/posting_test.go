package posting

import (
	"testing"
	"time"
)

func TestCustomerInvoicePreviewBalancesVATInvoice(t *testing.T) {
	entry, err := CustomerInvoicePreview(SourceDocument{
		ID:          "inv-1",
		Number:      "INV-001",
		PartyName:   "Acme",
		Date:        time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
		SubtotalBHD: 100,
		VATBHD:      10,
		TotalBHD:    110,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if !entry.IsBalanced || entry.DebitTotal != 110 || entry.CreditTotal != 110 {
		t.Fatalf("unexpected balance: %+v", entry)
	}
	if len(entry.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(entry.Lines))
	}
	if entry.Lines[0].Account.Role != RoleAccountsReceivable || entry.Lines[0].Debit != 110 {
		t.Fatalf("unexpected AR line: %+v", entry.Lines[0])
	}
	if entry.Lines[2].Account.Role != RoleVATOutput || entry.Lines[2].Credit != 10 {
		t.Fatalf("unexpected VAT output line: %+v", entry.Lines[2])
	}
}

func TestCustomerInvoicePreviewAllowsZeroVAT(t *testing.T) {
	entry, err := CustomerInvoicePreview(SourceDocument{
		ID:          "inv-2",
		Number:      "INV-002",
		PartyName:   "Export Customer",
		Date:        time.Now(),
		SubtotalBHD: 100,
		TotalBHD:    100,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(entry.Lines) != 2 {
		t.Fatalf("expected no VAT line for zero VAT, got %d lines", len(entry.Lines))
	}
}

func TestSupplierInvoicePreviewBalancesVATInvoice(t *testing.T) {
	entry, err := SupplierInvoicePreview(SourceDocument{
		ID:          "sinv-1",
		Number:      "SUP-001",
		PartyName:   "Supplier",
		Date:        time.Now(),
		SubtotalBHD: 50,
		VATBHD:      5,
		TotalBHD:    55,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if entry.DebitTotal != 55 || entry.CreditTotal != 55 {
		t.Fatalf("unexpected totals: debit %.3f credit %.3f", entry.DebitTotal, entry.CreditTotal)
	}
	if entry.Lines[0].Account.Role != RolePurchases || entry.Lines[1].Account.Role != RoleVATInput || entry.Lines[2].Account.Role != RoleAccountsPayable {
		t.Fatalf("unexpected supplier invoice lines: %+v", entry.Lines)
	}
}

func TestPaymentPreviewsBalance(t *testing.T) {
	customer, err := CustomerPaymentPreview(SourceDocument{ID: "pay-1", Number: "PAY-1", PartyName: "Customer", Date: time.Now(), TotalBHD: 25})
	if err != nil {
		t.Fatalf("customer payment preview failed: %v", err)
	}
	supplier, err := SupplierPaymentPreview(SourceDocument{ID: "spay-1", Number: "SPAY-1", PartyName: "Supplier", Date: time.Now(), TotalBHD: 25})
	if err != nil {
		t.Fatalf("supplier payment preview failed: %v", err)
	}
	if !customer.IsBalanced || !supplier.IsBalanced {
		t.Fatalf("payment previews should be balanced: customer=%+v supplier=%+v", customer, supplier)
	}
}

func TestPreviewRejectsMismatchedTotals(t *testing.T) {
	_, err := CustomerInvoicePreview(SourceDocument{
		ID:          "inv-bad",
		Number:      "INV-BAD",
		Date:        time.Now(),
		SubtotalBHD: 100,
		VATBHD:      10,
		TotalBHD:    100,
	})
	if err == nil {
		t.Fatal("expected mismatched total error")
	}
}
