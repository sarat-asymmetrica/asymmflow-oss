package hospitality

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/documents/numbering"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/money"
)

// CloseSession settles the kitchen side of a session and issues its simplified
// (B2C) tax invoice: number, ICV/PIH chain link, signed UBL 2.1 XML and QR.
//
// Authority: issuing an invoice is a persist/post action, so the closing actor
// must satisfy kernel actor.CanApprove — which an AI agent never can. The till
// operator is an operator-type actor with approve authority; a Butler-style
// agent can draft the bill preview but not issue it.
//
// Gates: no live kitchen tickets (queued/preparing/ready) and at least one
// non-voided line.
func (s *Service) CloseSession(sessionID uint, by actor.Actor) (*Invoice, error) {
	if !by.CanApprove() {
		return nil, fmt.Errorf("hospitality: actor %q (type %s) lacks authority to issue an invoice", by.ID, by.Type)
	}
	session, err := s.openSession(sessionID)
	if err != nil {
		return nil, err
	}

	var liveTickets int64
	if err := s.db.Model(&Ticket{}).
		Where("session_id = ? AND status IN ?", sessionID, []string{TicketQueued, TicketPreparing, TicketReady}).
		Count(&liveTickets).Error; err != nil {
		return nil, err
	}
	if liveTickets > 0 {
		return nil, fmt.Errorf("hospitality: session %d has %d kitchen ticket(s) still live", sessionID, liveTickets)
	}

	var lines []OrderLine
	if err := s.db.Where("session_id = ? AND status <> ?", sessionID, LineVoided).Find(&lines).Error; err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, errors.New("hospitality: nothing to invoice (all lines voided or none added)")
	}

	profile := s.ov.Profile(s.ov.DefaultDivision())
	seller := saudi.Party{
		RegistrationName: profile.LegalName,
		VATNumber:        profile.VATNumber,
		City:             profile.City,
		CountryCode:      s.ov.JurisdictionCode(),
	}
	if len(profile.AddressLines) > 0 {
		seller.Street = profile.AddressLines[0]
	}
	if len(profile.AddressLines) > 1 {
		seller.District = profile.AddressLines[1]
	}

	issuedAt := s.now()
	var stored Invoice
	err = s.db.Transaction(func(tx *gorm.DB) error {
		number, err := numbering.NextInTx(tx, invoiceNumberSpec, issuedAt)
		if err != nil {
			return err
		}

		// ICV/PIH chain: monotonic counter, each document carries the previous
		// document's hash (GenesisPIH for the very first). Read inside the
		// writer transaction so two closes can't fork it; the head spans
		// invoices AND credit notes (one chain per EGS unit — see chainHead).
		icv, pih, err := chainHead(tx)
		if err != nil {
			return err
		}

		einv := &saudi.EInvoice{
			ID:       number,
			UUID:     uuid.NewString(),
			IssuedAt: issuedAt,
			TypeCode: saudi.TypeTaxInvoice,
			Subtype:  saudi.Subtype{Simplified: true},
			Currency: s.ov.Currency,
			ICV:      icv,
			PIH:      pih,
			Seller:   seller,
		}
		for _, l := range lines {
			einv.Lines = append(einv.Lines, saudi.Line{
				Name:      l.Name,
				Quantity:  l.Qty,
				UnitPrice: float64(l.UnitPriceHalalas) / 100,
				TaxRate:   l.TaxRate,
				Category:  "services",
			})
		}

		signed, err := einv.Sign(s.signKey, s.cert, issuedAt)
		if err != nil {
			return fmt.Errorf("hospitality: ZATCA signing failed: %w", err)
		}
		totals := einv.ComputeTotals()

		stored = Invoice{
			Number:          number,
			SessionID:       session.ID,
			UUID:            einv.UUID,
			IssuedAt:        issuedAt,
			SubtotalHalalas: halalas(totals.LineExtension),
			VATHalalas:      halalas(totals.TaxAmount),
			TotalHalalas:    halalas(totals.TaxInclusive),
			Currency:        einv.Currency,
			ICV:             icv,
			PIH:             pih,
			HashB64:         signed.InvoiceHashB64,
			QRBase64:        signed.QRBase64,
			XML:             signed.XML,
			Status:          InvoiceIssued,
		}
		if err := tx.Create(&stored).Error; err != nil {
			return err
		}

		// Stamp the billed lines with their invoice (Wave 5 C.1): refunds
		// scope their ledger by this, and split sessions depend on it.
		lineIDs := make([]uint, 0, len(lines))
		for _, l := range lines {
			lineIDs = append(lineIDs, l.ID)
		}
		if err := tx.Model(&OrderLine{}).Where("id IN ?", lineIDs).
			Update("invoice_id", stored.ID).Error; err != nil {
			return err
		}

		// Spool the guest copy (W5 C.2): job and invoice commit together.
		if err := s.enqueuePrintTx(tx, PrintKindInvoice, "counter", stored.Number); err != nil {
			return err
		}

		closedAt := issuedAt
		session.Status = SessionClosed
		session.ClosedAt = &closedAt
		session.InvoiceID = &stored.ID
		return tx.Save(session).Error
	})
	if err != nil {
		return nil, err
	}

	// Publish after commit: the compliance hook (pkg/compliance) picks this up
	// via the bus, routes on Jurisdiction=SA, and validates with the Saudi VAT
	// engine — the vertical never imports a validator, it just emits the event.
	if s.bus != nil {
		_ = s.bus.Publish(context.Background(), events.InvoiceCreated{
			BaseEvent:     events.BaseEvent{Timestamp: issuedAt, CorrelationID: stored.UUID},
			InvoiceID:     fmt.Sprintf("%d", stored.ID),
			InvoiceNumber: stored.Number,
			InvoiceDate:   stored.IssuedAt,
			SellerTaxID:   seller.VATNumber,
			// Amount is the NET taxable base — the convention every
			// pkg/compliance engine validates TaxAmount against.
			Amount:       float64(stored.SubtotalHalalas) / 100,
			TaxAmount:    float64(stored.VATHalalas) / 100,
			Currency:     stored.Currency,
			Jurisdiction: s.ov.JurisdictionCode(),
		})
	}
	return &stored, nil
}

// RecordPayment captures one tender against an issued invoice. Overpayment is
// an error (change is given in cash, not recorded as tender). When cumulative
// payments reach the total, the invoice flips to paid.
func (s *Service) RecordPayment(invoiceID uint, method string, amount money.Amount) (*Payment, error) {
	method = strings.ToLower(strings.TrimSpace(method))
	if method == "" {
		return nil, errors.New("hospitality: payment method is required")
	}
	if !amount.IsPositive() {
		return nil, errors.New("hospitality: payment amount must be positive")
	}

	var payment Payment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var inv Invoice
		if err := tx.First(&inv, invoiceID).Error; err != nil {
			return fmt.Errorf("hospitality: invoice %d: %w", invoiceID, err)
		}
		if amount.Currency() != inv.Currency {
			return fmt.Errorf("hospitality: payment currency %s does not match invoice currency %s", amount.Currency(), inv.Currency)
		}
		if inv.Status == InvoicePaid {
			return fmt.Errorf("hospitality: invoice %s is already paid", inv.Number)
		}

		var paidSoFar int64
		if err := tx.Model(&Payment{}).Where("invoice_id = ?", inv.ID).
			Select("COALESCE(SUM(amount_halalas), 0)").Scan(&paidSoFar).Error; err != nil {
			return err
		}
		if paidSoFar+amount.Minor() > inv.TotalHalalas {
			return fmt.Errorf("hospitality: payment of %s would exceed invoice total %s",
				amount.Format(), money.FromMinor(inv.TotalHalalas, inv.Currency, 2).Format())
		}

		receivedAt := s.now()
		payment = Payment{
			InvoiceID:     inv.ID,
			Method:        method,
			AmountHalalas: amount.Minor(),
			ReceivedAt:    receivedAt,
			BusinessDate:  receivedAt.Format("2006-01-02"),
		}
		if err := tx.Create(&payment).Error; err != nil {
			return err
		}
		if paidSoFar+amount.Minor() == inv.TotalHalalas {
			inv.Status = InvoicePaid
			return tx.Save(&inv).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// halalas converts a SAR float (already rounded to 2dp by the ZATCA totals
// arithmetic) to integer minor units.
func halalas(sar float64) int64 { return int64(math.Round(sar * 100)) }

// sar wraps halalas as kernel money (SAR, scale 2).
func sar(minor int64) money.Amount { return money.FromMinor(minor, "SAR", 2) }
