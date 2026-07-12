package hospitality

// Wave 5 C.1 — bill split. One open session's lines split across N
// invoices by WHOLE-LINE assignment: a line's quantity is never divided,
// so per-line VAT rounding is identical whichever invoice a line lands
// on and the split documents' totals sum exactly to what one invoice
// would have carried. That invariant is enforced at issuance (W4-D6:
// refuse, never adjust) — a request that would need quantity splitting
// is refused, and a computed-sum mismatch aborts the whole transaction.

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/documents/numbering"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/kernel/actor"
)

// SplitSession closes an open session by issuing one invoice per
// assignment group. Each group is a set of the session's order-line IDs;
// together the groups must cover every non-voided line exactly once
// (whole-line assignment — quantities are never divided). Each split
// invoice is its own ZATCA document on the shared ICV/PIH chain; payments
// and refunds compose against each invoice unchanged.
//
// Authority: issuing invoices is a persist/post action — the closing
// actor must satisfy kernel CanApprove; agents never issue.
func (s *Service) SplitSession(sessionID uint, assignments [][]uint, by actor.Actor) ([]Invoice, error) {
	if !by.CanApprove() {
		return nil, fmt.Errorf("hospitality: actor %q (type %s) lacks authority to issue an invoice", by.ID, by.Type)
	}
	if len(assignments) < 2 {
		return nil, errors.New("hospitality: a bill split needs at least two groups (use CloseSession for one invoice)")
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

	// Whole-line coverage: every non-voided line assigned to exactly one
	// group; unknown, voided, or duplicated IDs are refused.
	byID := make(map[uint]OrderLine, len(lines))
	for _, l := range lines {
		byID[l.ID] = l
	}
	assigned := make(map[uint]bool, len(lines))
	groups := make([][]OrderLine, len(assignments))
	for gi, group := range assignments {
		if len(group) == 0 {
			return nil, fmt.Errorf("hospitality: split group %d is empty", gi+1)
		}
		for _, lineID := range group {
			line, ok := byID[lineID]
			if !ok {
				return nil, fmt.Errorf("hospitality: session %d has no billable line %d (unknown or voided)", sessionID, lineID)
			}
			if assigned[lineID] {
				return nil, fmt.Errorf("hospitality: line %d is assigned to more than one split group", lineID)
			}
			assigned[lineID] = true
			groups[gi] = append(groups[gi], line)
		}
	}
	if len(assigned) != len(lines) {
		missing := make([]uint, 0)
		for _, l := range lines {
			if !assigned[l.ID] {
				missing = append(missing, l.ID)
			}
		}
		return nil, fmt.Errorf("hospitality: split must assign every billable line; unassigned line(s): %v", missing)
	}

	// The invariant reference: what ONE invoice over all lines would carry.
	// Whole-line assignment makes the split total equal by construction
	// (the ZATCA arithmetic rounds per line); the check pins it.
	reference := &saudi.EInvoice{Currency: s.ov.Currency}
	for _, l := range lines {
		reference.Lines = append(reference.Lines, saudi.Line{
			Name:      l.Name,
			Quantity:  l.Qty,
			UnitPrice: float64(l.UnitPriceHalalas) / 100,
			TaxRate:   l.TaxRate,
			Category:  "services",
		})
	}
	referenceTotal := halalas(reference.ComputeTotals().TaxInclusive)

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
	invoices := make([]Invoice, 0, len(groups))
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var splitTotal int64
		for _, group := range groups {
			number, err := numbering.NextInTx(tx, invoiceNumberSpec, issuedAt)
			if err != nil {
				return err
			}
			// chainHead reads inside the writer transaction, so each split
			// invoice chains onto the one persisted just before it.
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
			for _, l := range group {
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

			stored := Invoice{
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

			lineIDs := make([]uint, 0, len(group))
			for _, l := range group {
				lineIDs = append(lineIDs, l.ID)
			}
			if err := tx.Model(&OrderLine{}).Where("id IN ?", lineIDs).
				Update("invoice_id", stored.ID).Error; err != nil {
				return err
			}

			// Spool each split invoice's guest copy (W5 C.2).
			if err := s.enqueuePrintTx(tx, PrintKindInvoice, "counter", stored.Number); err != nil {
				return err
			}

			splitTotal += stored.TotalHalalas
			invoices = append(invoices, stored)
		}

		// The W4-D6 discipline: the invariant is on the SUM, enforced at
		// issuance. Whole-line assignment makes drift impossible; if it
		// ever appears, refuse the whole split rather than adjust any
		// signed document's numbers.
		if splitTotal != referenceTotal {
			return fmt.Errorf("hospitality: split invoices total %d halalas but the session bills %d — rounding drift; split refused",
				splitTotal, referenceTotal)
		}

		closedAt := issuedAt
		session.Status = SessionClosed
		session.ClosedAt = &closedAt
		// The session points at its first split invoice; the full set is
		// recoverable via hosp_invoices.session_id (and each line's stamp).
		session.InvoiceID = &invoices[0].ID
		return tx.Save(session).Error
	})
	if err != nil {
		return nil, err
	}

	// Publish after commit, one event per issued document — the compliance
	// hook validates each split invoice independently.
	if s.bus != nil {
		for i := range invoices {
			inv := invoices[i]
			_ = s.bus.Publish(context.Background(), events.InvoiceCreated{
				BaseEvent:     events.BaseEvent{Timestamp: issuedAt, CorrelationID: inv.UUID},
				InvoiceID:     fmt.Sprintf("%d", inv.ID),
				InvoiceNumber: inv.Number,
				InvoiceDate:   inv.IssuedAt,
				SellerTaxID:   seller.VATNumber,
				Amount:        float64(inv.SubtotalHalalas) / 100,
				TaxAmount:     float64(inv.VATHalalas) / 100,
				Currency:      inv.Currency,
				Jurisdiction:  s.ov.JurisdictionCode(),
			})
		}
	}
	return invoices, nil
}
