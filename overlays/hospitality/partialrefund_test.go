package hospitality

// Wave 4 C.1/C.2 — partial (line-level) refunds and the CreditNoteIssued
// domain event. The refund ledger (hosp_credit_note_lines) is quantity-truth:
// a billed quantity can never be credited twice, several credit notes may
// share one original invoice, and the invoice flips to refunded exactly when
// the last billed quantity is credited.

import (
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/infra/events"
)

func sessionLines(t *testing.T, h *harness, inv *Invoice) map[string]OrderLine {
	t.Helper()
	var lines []OrderLine
	if err := h.db.Where("session_id = ? AND status <> ?", inv.SessionID, LineVoided).Find(&lines).Error; err != nil {
		t.Fatal(err)
	}
	out := make(map[string]OrderLine, len(lines))
	for _, l := range lines {
		out[l.Name] = l
	}
	return out
}

func TestRefundInvoiceLines_PartialThenComplete(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Karak Chai": 2, "Kunafa": 1})
	lines := sessionLines(t, h, inv)

	// First partial: the whole Karak Chai line.
	cn1, err := h.svc.RefundInvoiceLines(inv.ID,
		[]LineRefund{{OrderLineID: lines["Karak Chai"].ID, Qty: 2}},
		"cash", h.manager, testPIN, "wrong order — chai remade")
	if err != nil {
		t.Fatalf("first partial refund: %v", err)
	}
	if cn1.TotalHalalas >= inv.TotalHalalas {
		t.Fatalf("partial credit note total %d should be below invoice total %d", cn1.TotalHalalas, inv.TotalHalalas)
	}
	if cn1.ICV != inv.ICV+1 || cn1.PIH != inv.HashB64 {
		t.Error("first partial credit note does not chain onto the invoice")
	}
	xml := string(cn1.XML)
	for _, want := range []string{">381<", inv.Number, "InstructionNote"} {
		if !strings.Contains(xml, want) {
			t.Errorf("partial credit-note XML missing %q", want)
		}
	}

	// Invoice stays PAID while partially refunded (no invented status).
	var reloaded Invoice
	if err := h.db.First(&reloaded, inv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != InvoicePaid {
		t.Fatalf("partially refunded invoice status = %s, want paid", reloaded.Status)
	}

	// A quantity cannot be credited twice.
	if _, err := h.svc.RefundInvoiceLines(inv.ID,
		[]LineRefund{{OrderLineID: lines["Karak Chai"].ID, Qty: 1}},
		"cash", h.manager, testPIN, "again"); err == nil {
		t.Fatal("over-refund of an exhausted line was allowed")
	}
	// And a full refund is refused once partials exist.
	if _, err := h.svc.RefundInvoice(inv.ID, "cash", h.manager, testPIN, "full"); err == nil {
		t.Fatal("full refund allowed on a partially refunded invoice")
	}

	// Second partial completes the refund: totals add up exactly (whole-line
	// splits share the document arithmetic's per-line rounding) and the
	// invoice flips to refunded.
	cn2, err := h.svc.RefundInvoiceLines(inv.ID,
		[]LineRefund{{OrderLineID: lines["Kunafa"].ID, Qty: 1}},
		"cash", h.manager, testPIN, "guest left before dessert")
	if err != nil {
		t.Fatalf("completing partial refund: %v", err)
	}
	if cn2.ICV != cn1.ICV+1 || cn2.PIH != cn1.HashB64 {
		t.Error("second partial credit note does not chain onto the first")
	}
	if cn1.TotalHalalas+cn2.TotalHalalas != inv.TotalHalalas {
		t.Errorf("credit notes sum to %d halalas, invoice total is %d",
			cn1.TotalHalalas+cn2.TotalHalalas, inv.TotalHalalas)
	}
	if err := h.db.First(&reloaded, inv.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != InvoiceRefunded {
		t.Fatalf("fully credited invoice status = %s, want refunded", reloaded.Status)
	}

	// The refund ledger carries one row per credited line.
	var ledger []CreditNoteLine
	if err := h.db.Order("id").Find(&ledger).Error; err != nil {
		t.Fatal(err)
	}
	if len(ledger) != 2 {
		t.Fatalf("ledger rows = %d, want 2", len(ledger))
	}

	// Both refunds landed as negative tenders; drawer nets to zero.
	var net int64
	if err := h.db.Model(&Payment{}).Where("invoice_id = ?", inv.ID).
		Select("COALESCE(SUM(amount_halalas), 0)").Scan(&net).Error; err != nil {
		t.Fatal(err)
	}
	if net != 0 {
		t.Errorf("net tender movement = %d halalas, want 0", net)
	}
}

func TestRefundInvoiceLines_Gates(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Karak Chai": 2})
	lines := sessionLines(t, h, inv)
	chai := lines["Karak Chai"].ID

	// Agent actors are refused before anything happens (kernel boundary).
	if _, err := h.svc.RefundInvoiceLines(inv.ID, []LineRefund{{OrderLineID: chai, Qty: 1}},
		"cash", h.agent, testPIN, "refund"); err == nil {
		t.Fatal("agent issued a partial credit note")
	}
	// Empty request.
	if _, err := h.svc.RefundInvoiceLines(inv.ID, nil, "cash", h.manager, testPIN, "r"); err == nil {
		t.Fatal("empty line-refund request accepted")
	}
	// Unknown line.
	if _, err := h.svc.RefundInvoiceLines(inv.ID, []LineRefund{{OrderLineID: 99999, Qty: 1}},
		"cash", h.manager, testPIN, "r"); err == nil {
		t.Fatal("unknown line accepted")
	}
	// Duplicate line in one request.
	if _, err := h.svc.RefundInvoiceLines(inv.ID,
		[]LineRefund{{OrderLineID: chai, Qty: 1}, {OrderLineID: chai, Qty: 1}},
		"cash", h.manager, testPIN, "r"); err == nil {
		t.Fatal("duplicate line accepted")
	}
	// Non-positive and over-quantity.
	if _, err := h.svc.RefundInvoiceLines(inv.ID, []LineRefund{{OrderLineID: chai, Qty: 0}},
		"cash", h.manager, testPIN, "r"); err == nil {
		t.Fatal("zero quantity accepted")
	}
	if _, err := h.svc.RefundInvoiceLines(inv.ID, []LineRefund{{OrderLineID: chai, Qty: 3}},
		"cash", h.manager, testPIN, "r"); err == nil {
		t.Fatal("over-quantity accepted")
	}
}

// Splitting a single quantity can round each partial document UP a halala;
// the cumulative guard must refuse the split that would hand back more than
// the guest paid, rather than silently over-refund.
func TestRefundInvoiceLines_RoundingGuardNeverOverRefunds(t *testing.T) {
	h := newHarness(t)
	// Karak Chai: 848 halalas net. Full line (qty 1): VAT = round2(1.272) =
	// 1.27. Each half: VAT = round2(0.636) = 0.64 → two halves credit 976
	// halalas against a 975-halala invoice.
	inv := h.runSession(t, "T1", map[string]float64{"Karak Chai": 1})
	lines := sessionLines(t, h, inv)
	chai := lines["Karak Chai"].ID

	if _, err := h.svc.RefundInvoiceLines(inv.ID, []LineRefund{{OrderLineID: chai, Qty: 0.5}},
		"cash", h.manager, testPIN, "half comped"); err != nil {
		t.Fatalf("first half refund: %v", err)
	}
	_, err := h.svc.RefundInvoiceLines(inv.ID, []LineRefund{{OrderLineID: chai, Qty: 0.5}},
		"cash", h.manager, testPIN, "other half")
	if err == nil {
		t.Fatal("second half refund over-credited the invoice and was allowed")
	}
	if !strings.Contains(err.Error(), "exceed") {
		t.Fatalf("expected the cumulative guard, got: %v", err)
	}

	// Nothing from the refused document persisted: no second credit note,
	// no second negative tender.
	var cnCount, ledgerCount int64
	h.db.Model(&CreditNote{}).Count(&cnCount)
	h.db.Model(&CreditNoteLine{}).Count(&ledgerCount)
	if cnCount != 1 || ledgerCount != 1 {
		t.Fatalf("refused refund leaked rows: credit notes = %d, ledger = %d", cnCount, ledgerCount)
	}
}

// W4 C.2: a credit note publishes its OWN domain event, and the compliance
// hook validates it under the credit-note event name — not smuggled through
// InvoiceCreated.
func TestCreditNoteIssued_EventReachesComplianceHook(t *testing.T) {
	h := newHarness(t)
	inv := h.runSession(t, "T1", map[string]float64{"Kunafa": 1})

	if _, err := h.svc.RefundInvoice(inv.ID, "card", h.manager, testPIN, "guest complaint"); err != nil {
		t.Fatal(err)
	}

	// The hook validates asynchronously — poll.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, v := range h.hook.RecentValidations(20) {
			if v.EventName == events.EventCreditNoteIssued {
				if !v.Valid {
					t.Fatalf("credit-note compliance validation failed: %+v", v)
				}
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("compliance hook never recorded a credit-note validation")
}
