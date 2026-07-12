package hospitality

// Wave 3 C.3 — the refund/credit-note flow. Retires the "proof vertical is
// happy-path-only" critique: money now flows BOTH ways through the same
// substrate (kernel authority, PIN engine, shared numbering, one ICV/PIH
// chain, settlement-reconciled day close).

import (
	"strings"
	"testing"

	"ph_holdings_app/pkg/finance/settlement"
	"ph_holdings_app/pkg/kernel/money"
)

func TestRefundInvoice_FullFlow(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Karak Chai": 2, "Chicken Kabsa": 1})

	cn, err := h.svc.RefundInvoice(inv.ID, "card", h.manager, testPIN, "guest complaint — meal comped")
	if err != nil {
		t.Fatalf("RefundInvoice: %v", err)
	}

	// Chain: the credit note is the next link after the invoice.
	if cn.ICV != inv.ICV+1 {
		t.Errorf("credit note ICV = %d, want %d", cn.ICV, inv.ICV+1)
	}
	if cn.PIH != inv.HashB64 {
		t.Error("credit note PIH is not the invoice's hash — chain forked")
	}
	if cn.QRBase64 == "" {
		t.Error("simplified credit note must carry a QR")
	}
	if !strings.HasPrefix(cn.Number, "SCRN-") {
		t.Errorf("credit note number = %q", cn.Number)
	}

	// Totals mirror the invoice exactly (full refund).
	if cn.TotalHalalas != inv.TotalHalalas || cn.VATHalalas != inv.VATHalalas || cn.SubtotalHalalas != inv.SubtotalHalalas {
		t.Errorf("credit note totals %d/%d/%d != invoice %d/%d/%d",
			cn.SubtotalHalalas, cn.VATHalalas, cn.TotalHalalas,
			inv.SubtotalHalalas, inv.VATHalalas, inv.TotalHalalas)
	}

	// The UBL document is a 381 with billing reference + reason.
	xml := string(cn.XML)
	for _, want := range []string{
		">381<", inv.Number, "guest complaint — meal comped",
		"BillingReference", "InstructionNote",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("credit-note XML missing %q", want)
		}
	}

	// Original invoice is now refunded; a second refund is refused.
	var reloaded Invoice
	if err := h.db.First(&reloaded, inv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != InvoiceRefunded {
		t.Errorf("invoice status = %s, want refunded", reloaded.Status)
	}
	if _, err := h.svc.RefundInvoice(inv.ID, "card", h.manager, testPIN, "again"); err == nil {
		t.Fatal("double refund allowed")
	}

	// The refund landed as a negative tender on the credit note.
	var refund Payment
	if err := h.db.Where("credit_note_id = ?", cn.ID).First(&refund).Error; err != nil {
		t.Fatalf("no refund tender row: %v", err)
	}
	if refund.AmountHalalas != -inv.TotalHalalas || refund.Method != "card" {
		t.Errorf("refund tender = %+v", refund)
	}

	// Next document (an ordinary invoice) chains onto the credit note.
	inv2 := h.runSession(t, "T2", map[string]float64{"Kunafa": 1})
	if inv2.ICV != cn.ICV+1 || inv2.PIH != cn.HashB64 {
		t.Errorf("next invoice (ICV %d, PIH %.12s…) does not chain onto the credit note (ICV %d, hash %.12s…)",
			inv2.ICV, inv2.PIH, cn.ICV, cn.HashB64)
	}
}

func TestRefundInvoice_Gates(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Karak Chai": 1})

	// The AI agent can never issue a credit note (kernel boundary).
	if _, err := h.svc.RefundInvoice(inv.ID, "cash", h.agent, testPIN, "please refund"); err == nil {
		t.Fatal("agent issued a credit note")
	}
	// Wrong PIN refused even for the manager.
	if _, err := h.svc.RefundInvoice(inv.ID, "cash", h.manager, "0000", "reason"); err == nil {
		t.Fatal("wrong PIN accepted")
	}
	// A reason (InstructionNote) is mandatory.
	if _, err := h.svc.RefundInvoice(inv.ID, "cash", h.manager, testPIN, "  "); err == nil {
		t.Fatal("blank reason accepted")
	}
	// Unpaid invoices cannot be refunded.
	session, err := h.svc.OpenSession("T2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.AddLine(session.ID, "Kunafa", 1); err != nil {
		t.Fatal(err)
	}
	unpaid, err := h.svc.CloseSession(session.ID, h.manager)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.RefundInvoice(unpaid.ID, "cash", h.manager, testPIN, "reason"); err == nil {
		t.Fatal("refunded an unpaid invoice")
	}
}

// A same-day sale + full refund nets the drawer to zero, and the day close
// reconciles that NET movement through the settlement engine.
func TestRefundInvoice_DayCloseNetsToZero(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Karak Chai": 2, "Kunafa": 1})
	if _, err := h.svc.RefundInvoice(inv.ID, "card", h.manager, testPIN, "order cancelled after payment"); err != nil {
		t.Fatal(err)
	}

	businessDate := h.svc.now().Format("2006-01-02")
	expected, err := h.svc.ExpectedTenders(businessDate)
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != 1 || expected[0].Method != "card" || expected[0].Expected.Minor() != 0 {
		t.Fatalf("expected tenders after net-zero day = %+v", expected)
	}

	dc, err := h.svc.CloseDay(businessDate,
		[]settlement.Declaration{{Method: "card", Counted: money.FromMinor(0, "SAR", 2)}},
		h.manager, testPIN, "")
	if err != nil {
		t.Fatalf("CloseDay: %v", err)
	}
	if dc.ExpectedHalalas != 0 || dc.CountedHalalas != 0 || dc.VarianceHalalas != 0 {
		t.Errorf("day close = %+v, want all zero", dc)
	}
}
