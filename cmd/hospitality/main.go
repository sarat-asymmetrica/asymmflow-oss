// Command hospitality is the Wave 2 composition proof: a SECOND vertical
// (Saudi café point-of-sale) booted purely from the AsymmFlow substrate —
// pkg/kernel + pkg/overlay + pkg/documents/numbering + pkg/finance/settlement
// + pkg/infra/{auth,events} + pkg/compliance(+saudi) — with its own domain
// package (overlays/hospitality) and its own overlay.json. No trading code is
// imported; no engine code is duplicated.
//
// Run it and it executes one full end-to-end business day against synthetic
// seed data: open table → order → kitchen tickets → manager-PIN void →
// signed ZATCA simplified invoice → payment → compliance validation via the
// event bus → tender-reconciled day close. Exit code 0 = the thesis holds.
//
//	go run ./cmd/hospitality              # in-memory demo day
//	go run ./cmd/hospitality -db pos.db   # persistent database
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/overlays/hospitality"
	"ph_holdings_app/pkg/compliance"
	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/finance/settlement"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/money"
	"ph_holdings_app/pkg/runtime/composition"
)

const demoManagerPIN = "4321" // synthetic demo PIN, set at boot

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hospitality: FAILED: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dbPath := flag.String("db", "", "SQLite database path (default: temporary file, deleted afterwards)")
	overlayDir := flag.String("overlay", "overlays/hospitality", "directory containing overlay.json")
	flag.Parse()

	// ---- Composition root: every dependency is wired HERE, through the
	// SAME seam the trading app boots through (pkg/runtime/composition) ----
	root := composition.NewRoot()

	// 1. Deployment identity from overlay.json (falls back to trading defaults
	//    if missing — so we hard-require the hospitality overlay actually loaded).
	ov := root.LoadOverlay([]string{*overlayDir})
	if ov.JurisdictionCode() != "SA" {
		return fmt.Errorf("expected the KSA hospitality overlay from %s (got jurisdiction %q) — run from the repo root or pass -overlay", *overlayDir, ov.JurisdictionCode())
	}

	// 2. Database (pure-Go SQLite; same driver discipline as the trading app).
	path := *dbPath
	if path == "" {
		dir, err := os.MkdirTemp("", "hospitality-demo-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		path = filepath.Join(dir, "pos.db")
	}
	db, err := root.OpenSQLite(
		composition.SQLiteDSN(path, "busy_timeout(5000)", "journal_mode(WAL)"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	// 3. Event bus + compliance: the hook subscribes to invoice events and
	//    routes them to the registered engine for the invoice's jurisdiction.
	hook := root.WireCompliance(saudi.New())
	bus := root.Bus

	// 4. EGS signing identity: ephemeral key + self-signed certificate. A real
	//    deployment onboards via the Fatoora CSID flow and loads those instead.
	key, err := saudi.GenerateKeyPair()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	cert, err := saudi.NewSelfSignedCertificate(key, ov.CompanyDisplayName, "SA", now.AddDate(0, 0, -1), now.AddDate(5, 0, 0))
	if err != nil {
		return err
	}

	// 5. The vertical's domain service.
	svc, err := hospitality.NewService(db, ov, bus, key, cert)
	if err != nil {
		return err
	}
	if err := svc.SeedDemo(); err != nil {
		return err
	}
	if err := svc.SetManagerPIN(demoManagerPIN); err != nil {
		return err
	}

	// ---- One end-to-end business day ----

	fmt.Printf("═══ %s — %s (%s, VAT %s) ═══\n\n",
		ov.CompanyDisplayName, ov.Industry, ov.Currency, ov.Profile(ov.DefaultDivision()).VATNumber)

	manager, err := actor.New(actor.Input{ID: "mgr-01", DisplayName: "Huda (shift manager)", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	if err != nil {
		return err
	}
	agent, err := actor.New(actor.Input{ID: "butler-01", DisplayName: "Butler (AI)", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})
	if err != nil {
		return err
	}

	session, err := svc.OpenSession("T3")
	if err != nil {
		return err
	}
	fmt.Printf("· Session %d opened on table T3\n", session.ID)

	for _, order := range []struct {
		item string
		qty  float64
	}{
		{"Karak Chai", 2},
		{"Chicken Kabsa", 1},
		{"Kunafa", 1},
		{"Mint Lemonade", 1},
	} {
		if _, err := svc.AddLine(session.ID, order.item, order.qty); err != nil {
			return err
		}
		fmt.Printf("· Ordered %.0f× %s\n", order.qty, order.item)
	}

	ticket, err := svc.SendKOT(session.ID)
	if err != nil {
		return err
	}
	fmt.Printf("· Kitchen ticket %s dispatched (%s)\n", ticket.Number, ticket.Status)
	for _, state := range []string{hospitality.TicketPreparing, hospitality.TicketReady, hospitality.TicketServed} {
		if ticket, err = svc.AdvanceTicket(ticket.ID, state); err != nil {
			return err
		}
	}
	fmt.Printf("· Kitchen ticket %s served\n", ticket.Number)

	// The customer sends the lemonade back — a manager void, PIN-gated.
	var lemonade hospitality.OrderLine
	if err := db.First(&lemonade, "session_id = ? AND name = ?", session.ID, "Mint Lemonade").Error; err != nil {
		return err
	}
	// The AI agent cannot void, whatever it asks for (kernel boundary):
	if err := svc.VoidLine(session.ID, lemonade.ID, agent, demoManagerPIN, "customer returned it"); err == nil {
		return fmt.Errorf("AI-authority boundary FAILED: agent voided a line")
	} else {
		fmt.Printf("· Agent void refused ✔ (%v)\n", err)
	}
	if err := svc.VoidLine(session.ID, lemonade.ID, manager, demoManagerPIN, "customer returned it — too sweet"); err != nil {
		return err
	}
	fmt.Println("· Mint Lemonade voided by manager (PIN verified)")

	// Close the session → signed simplified ZATCA invoice. Agents can't:
	if _, err := svc.CloseSession(session.ID, agent); err == nil {
		return fmt.Errorf("AI-authority boundary FAILED: agent issued an invoice")
	}
	invoice, err := svc.CloseSession(session.ID, manager)
	if err != nil {
		return err
	}
	fmt.Printf("\n· Invoice %s issued (ICV %d)\n", invoice.Number, invoice.ICV)
	fmt.Printf("    net %s  VAT %s  total %s\n",
		money.FromMinor(invoice.SubtotalHalalas, invoice.Currency, 2).Format(),
		money.FromMinor(invoice.VATHalalas, invoice.Currency, 2).Format(),
		money.FromMinor(invoice.TotalHalalas, invoice.Currency, 2).Format())
	fmt.Printf("    hash %s…\n", invoice.HashB64[:24])
	if qr, err := base64.StdEncoding.DecodeString(invoice.QRBase64); err == nil {
		fmt.Printf("    QR   %d TLV bytes (tags 1–9, phone-scannable)\n", len(qr))
	}

	if _, err := svc.RecordPayment(invoice.ID, "card", money.FromMinor(invoice.TotalHalalas, "SAR", 2)); err != nil {
		return err
	}
	fmt.Println("· Paid by card")

	// Give the async compliance hook a beat, then show what it recorded.
	var validations []compliance.ValidationEntry
	for range 50 {
		if validations = hook.RecentValidations(5); len(validations) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(validations) == 0 {
		return fmt.Errorf("compliance hook recorded no validation — event wiring broken")
	}
	v := validations[len(validations)-1]
	fmt.Printf("· Compliance hook validated invoice: jurisdiction=%s valid=%v warnings=%v\n", v.Jurisdiction, v.Valid, v.Warnings)

	// A second table pays and then cancels — the refund flows back out as a
	// ZATCA credit note (TypeCode 381) chained on the SAME ICV/PIH sequence,
	// with the drawer movement netted into the day close. Agents can't:
	session2, err := svc.OpenSession("T5")
	if err != nil {
		return err
	}
	if _, err := svc.AddLine(session2.ID, "Kunafa", 1); err != nil {
		return err
	}
	invoice2, err := svc.CloseSession(session2.ID, manager)
	if err != nil {
		return err
	}
	if _, err := svc.RecordPayment(invoice2.ID, "card", money.FromMinor(invoice2.TotalHalalas, "SAR", 2)); err != nil {
		return err
	}
	fmt.Printf("\n· Invoice %s issued and paid (ICV %d)\n", invoice2.Number, invoice2.ICV)
	if _, err := svc.RefundInvoice(invoice2.ID, "card", agent, demoManagerPIN, "guest cancelled"); err == nil {
		return fmt.Errorf("AI-authority boundary FAILED: agent issued a credit note")
	} else {
		fmt.Printf("· Agent refund refused ✔ (%v)\n", err)
	}
	creditNote, err := svc.RefundInvoice(invoice2.ID, "card", manager, demoManagerPIN, "guest cancelled — full refund")
	if err != nil {
		return err
	}
	fmt.Printf("· Credit note %s issued (ICV %d, refunds %s, chained on %.12s…)\n",
		creditNote.Number, creditNote.ICV,
		money.FromMinor(creditNote.TotalHalalas, "SAR", 2).Format(), creditNote.PIH)

	// A third table orders two items and returns ONE — a PARTIAL refund
	// (W4 C.1): its own credit note on the same chain, per-line refund
	// ledger, invoice stays paid until the last quantity is credited.
	session3, err := svc.OpenSession("T4")
	if err != nil {
		return err
	}
	chaiLine, err := svc.AddLine(session3.ID, "Karak Chai", 2)
	if err != nil {
		return err
	}
	if _, err := svc.AddLine(session3.ID, "Luqaimat", 1); err != nil {
		return err
	}
	invoice3, err := svc.CloseSession(session3.ID, manager)
	if err != nil {
		return err
	}
	if _, err := svc.RecordPayment(invoice3.ID, "card", money.FromMinor(invoice3.TotalHalalas, "SAR", 2)); err != nil {
		return err
	}
	partialCN, err := svc.RefundInvoiceLines(invoice3.ID,
		[]hospitality.LineRefund{{OrderLineID: chaiLine.ID, Qty: 2}},
		"card", manager, demoManagerPIN, "chai returned — brewed wrong")
	if err != nil {
		return err
	}
	fmt.Printf("· Partial credit note %s issued (ICV %d, refunds %s of invoice %s — invoice stays paid)\n",
		partialCN.Number, partialCN.ICV,
		money.FromMinor(partialCN.TotalHalalas, "SAR", 2).Format(), invoice3.Number)

	// A fourth table splits the bill (W5 C.1): two friends, one session,
	// TWO invoices by whole-line assignment — each its own ZATCA document
	// on the same ICV/PIH chain, each paid separately. Agents can't split
	// either (issuing invoices is a persist action).
	session4, err := svc.OpenSession("T6")
	if err != nil {
		return err
	}
	kabsaLine, err := svc.AddLine(session4.ID, "Chicken Kabsa", 1)
	if err != nil {
		return err
	}
	mandiLine, err := svc.AddLine(session4.ID, "Lamb Mandi", 1)
	if err != nil {
		return err
	}
	coffeeLine, err := svc.AddLine(session4.ID, "Saudi Coffee (Dallah)", 2)
	if err != nil {
		return err
	}
	if _, err := svc.SplitSession(session4.ID, [][]uint{{kabsaLine.ID}, {mandiLine.ID, coffeeLine.ID}}, agent); err == nil {
		return fmt.Errorf("AI-authority boundary FAILED: agent split a bill into invoices")
	}
	splitInvoices, err := svc.SplitSession(session4.ID, [][]uint{{kabsaLine.ID}, {mandiLine.ID, coffeeLine.ID}}, manager)
	if err != nil {
		return err
	}
	var splitTotal int64
	fmt.Println()
	for i, inv := range splitInvoices {
		if _, err := svc.RecordPayment(inv.ID, "card", money.FromMinor(inv.TotalHalalas, "SAR", 2)); err != nil {
			return err
		}
		splitTotal += inv.TotalHalalas
		fmt.Printf("· Split invoice %d/%d %s issued and paid (ICV %d, total %s)\n",
			i+1, len(splitInvoices), inv.Number, inv.ICV,
			money.FromMinor(inv.TotalHalalas, "SAR", 2).Format())
	}
	fmt.Printf("· Bill split: one session, %d invoices, totals sum to %s exactly\n",
		len(splitInvoices), money.FromMinor(splitTotal, "SAR", 2).Format())

	// Day close: reconcile the drawer with the settlement engine. The card
	// drawer expects invoice 1, plus invoice 2 minus its full refund, plus
	// invoice 3 minus its partial refund, plus both split invoices — the net.
	businessDate := time.Now().UTC().Format("2006-01-02")
	expectedCard := invoice.TotalHalalas + invoice3.TotalHalalas - partialCN.TotalHalalas + splitTotal
	dayClose, err := svc.CloseDay(businessDate,
		[]settlement.Declaration{{Method: "card", Counted: money.FromMinor(expectedCard, "SAR", 2)}},
		manager, demoManagerPIN, "")
	if err != nil {
		return err
	}
	fmt.Printf("\n· Day %s closed by %s: expected %s counted %s variance %s\n",
		dayClose.BusinessDate, dayClose.ClosedByName,
		money.FromMinor(dayClose.ExpectedHalalas, "SAR", 2).Format(),
		money.FromMinor(dayClose.CountedHalalas, "SAR", 2).Format(),
		money.FromMinor(dayClose.VarianceHalalas, "SAR", 2).Format())

	fmt.Println("\n═══ Composition proof: BOOTED, end-to-end workflow complete ═══")
	return nil
}
