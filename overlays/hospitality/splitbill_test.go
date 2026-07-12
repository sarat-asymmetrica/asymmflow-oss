package hospitality

// Wave 5 C.1 — bill split. Whole-line assignment only: the split
// documents' totals must sum EXACTLY to what one invoice would carry, each
// split invoice is its own ZATCA document on the shared ICV/PIH chain, and
// payments/refunds compose per invoice unchanged.

import (
	"testing"

	"ph_holdings_app/pkg/kernel/money"
)

// openSessionWithLines drives a session through order → KOT → served and
// returns it still OPEN with its line IDs by name.
func openSessionWithLines(t *testing.T, h *harness, tableCode string, items map[string]float64) (uint, map[string]OrderLine) {
	t.Helper()
	session, err := h.svc.OpenSession(tableCode)
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	for name, qty := range items {
		if _, err := h.svc.AddLine(session.ID, name, qty); err != nil {
			t.Fatalf("AddLine %s: %v", name, err)
		}
	}
	ticket, err := h.svc.SendKOT(session.ID)
	if err != nil {
		t.Fatalf("SendKOT: %v", err)
	}
	for _, state := range []string{TicketPreparing, TicketReady, TicketServed} {
		if ticket, err = h.svc.AdvanceTicket(ticket.ID, state); err != nil {
			t.Fatalf("AdvanceTicket → %s: %v", state, err)
		}
	}
	var lines []OrderLine
	if err := h.db.Where("session_id = ?", session.ID).Find(&lines).Error; err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]OrderLine, len(lines))
	for _, l := range lines {
		byName[l.Name] = l
	}
	return session.ID, byName
}

func TestSplitSession_TotalsSumExactlyAndChainHolds(t *testing.T) {
	h := newHarness(t)

	// Reference: the SAME order closed as one invoice on another table.
	reference := h.runSession(t, "T2", map[string]float64{"Karak Chai": 2, "Kunafa": 1, "Saudi Coffee (Dallah)": 1})

	sessionID, lines := openSessionWithLines(t, h, "T1", map[string]float64{"Karak Chai": 2, "Kunafa": 1, "Saudi Coffee (Dallah)": 1})
	invoices, err := h.svc.SplitSession(sessionID, [][]uint{
		{lines["Karak Chai"].ID},
		{lines["Kunafa"].ID, lines["Saudi Coffee (Dallah)"].ID},
	}, h.manager)
	if err != nil {
		t.Fatalf("SplitSession: %v", err)
	}
	if len(invoices) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(invoices))
	}

	// Whole-line assignment: the split totals sum exactly to the
	// single-invoice reference.
	var sum, vatSum int64
	for _, inv := range invoices {
		sum += inv.TotalHalalas
		vatSum += inv.VATHalalas
	}
	if sum != reference.TotalHalalas {
		t.Fatalf("split totals %d != single-invoice total %d halalas", sum, reference.TotalHalalas)
	}
	if vatSum != reference.VATHalalas {
		t.Fatalf("split VAT %d != single-invoice VAT %d halalas", vatSum, reference.VATHalalas)
	}

	// Chain: consecutive ICVs, each PIH linking the previous document.
	if invoices[1].ICV != invoices[0].ICV+1 {
		t.Fatalf("split invoices must have consecutive ICVs: %d, %d", invoices[0].ICV, invoices[1].ICV)
	}
	if invoices[1].PIH != invoices[0].HashB64 {
		t.Error("second split invoice does not chain onto the first")
	}

	// The session is closed and every line is stamped with its invoice.
	var session OrderSession
	if err := h.db.First(&session, sessionID).Error; err != nil {
		t.Fatal(err)
	}
	if session.Status != SessionClosed || session.InvoiceID == nil {
		t.Fatalf("session must be closed with an invoice reference: %+v", session)
	}
	var stamped []OrderLine
	if err := h.db.Where("session_id = ?", sessionID).Find(&stamped).Error; err != nil {
		t.Fatal(err)
	}
	for _, l := range stamped {
		if l.InvoiceID == nil {
			t.Fatalf("line %s not stamped with an invoice", l.Name)
		}
	}

	// A subsequent document still chains onto the split's tail.
	next := h.runSession(t, "T3", map[string]float64{"Kunafa": 1})
	if next.ICV != invoices[1].ICV+1 || next.PIH != invoices[1].HashB64 {
		t.Error("next invoice does not chain onto the last split invoice")
	}
}

func TestSplitSession_RefusesAgentsAndBadAssignments(t *testing.T) {
	h := newHarness(t)
	sessionID, lines := openSessionWithLines(t, h, "T1", map[string]float64{"Karak Chai": 2, "Kunafa": 1})
	chai, kunafa := lines["Karak Chai"].ID, lines["Kunafa"].ID

	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai}, {kunafa}}, h.agent); err == nil {
		t.Fatal("an agent must never issue split invoices")
	}
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai, kunafa}}, h.manager); err == nil {
		t.Fatal("a single group is not a split")
	}
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai}, {}}, h.manager); err == nil {
		t.Fatal("an empty group must be refused")
	}
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai}, {chai}}, h.manager); err == nil {
		t.Fatal("a line assigned twice must be refused")
	}
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai}, {99999}}, h.manager); err == nil {
		t.Fatal("an unknown line must be refused")
	}
	// Unassigned line (kunafa missing) — the split must cover everything.
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai}, {chai}}, h.manager); err == nil {
		t.Fatal("full coverage is mandatory")
	}
	if _, err := h.svc.SplitSession(sessionID, [][]uint{{chai}}, h.manager); err == nil {
		t.Fatal("partial coverage must be refused")
	}

	// Every refusal leaves the session OPEN and unbilled.
	var session OrderSession
	if err := h.db.First(&session, sessionID).Error; err != nil {
		t.Fatal(err)
	}
	if session.Status != SessionOpen {
		t.Fatalf("refused splits must leave the session open, got %s", session.Status)
	}
	var invoiceCount int64
	if err := h.db.Model(&Invoice{}).Where("session_id = ?", sessionID).Count(&invoiceCount).Error; err != nil {
		t.Fatal(err)
	}
	if invoiceCount != 0 {
		t.Fatalf("refused splits must leave no invoices, found %d", invoiceCount)
	}
}

func TestSplitSession_PaymentsAndRefundsComposePerInvoice(t *testing.T) {
	h := newHarness(t)
	sessionID, lines := openSessionWithLines(t, h, "T1", map[string]float64{"Karak Chai": 2, "Kunafa": 1})

	invoices, err := h.svc.SplitSession(sessionID, [][]uint{
		{lines["Karak Chai"].ID},
		{lines["Kunafa"].ID},
	}, h.manager)
	if err != nil {
		t.Fatalf("SplitSession: %v", err)
	}
	first, second := invoices[0], invoices[1]

	// Each split invoice pays independently.
	for _, inv := range invoices {
		if _, err := h.svc.RecordPayment(inv.ID, "card", money.FromMinor(inv.TotalHalalas, "SAR", 2)); err != nil {
			t.Fatalf("RecordPayment %s: %v", inv.Number, err)
		}
	}

	// Refunding a line that belongs to the OTHER split invoice is refused —
	// the refund ledger scopes to the invoice's own stamped lines.
	if _, err := h.svc.RefundInvoiceLines(first.ID,
		[]LineRefund{{OrderLineID: lines["Kunafa"].ID, Qty: 1}},
		"cash", h.manager, testPIN, "wrong invoice"); err == nil {
		t.Fatal("refunding another split invoice's line must be refused")
	}

	// A full refund of one split invoice credits exactly its own total and
	// leaves the sibling untouched.
	cn, err := h.svc.RefundInvoice(first.ID, "cash", h.manager, testPIN, "guest complaint")
	if err != nil {
		t.Fatalf("RefundInvoice: %v", err)
	}
	if cn.TotalHalalas != first.TotalHalalas {
		t.Fatalf("credit note %d halalas != split invoice %d halalas", cn.TotalHalalas, first.TotalHalalas)
	}

	var reloadedFirst, reloadedSecond Invoice
	if err := h.db.First(&reloadedFirst, first.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := h.db.First(&reloadedSecond, second.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloadedFirst.Status != InvoiceRefunded {
		t.Fatalf("refunded split invoice should be refunded, got %s", reloadedFirst.Status)
	}
	if reloadedSecond.Status != InvoicePaid {
		t.Fatalf("sibling split invoice must stay paid, got %s", reloadedSecond.Status)
	}
}

func TestSplitSession_LiveTicketsBlock(t *testing.T) {
	h := newHarness(t)
	session, err := h.svc.OpenSession("T1")
	if err != nil {
		t.Fatal(err)
	}
	line, err := h.svc.AddLine(session.ID, "Kunafa", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SendKOT(session.ID); err != nil {
		t.Fatal(err)
	}
	// Ticket still queued — the split must refuse.
	if _, err := h.svc.SplitSession(session.ID, [][]uint{{line.ID}, {line.ID}}, h.manager); err == nil {
		t.Fatal("a session with live kitchen tickets must refuse splitting")
	}
}
