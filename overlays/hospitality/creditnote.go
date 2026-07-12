package hospitality

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/documents/numbering"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/kernel/actor"
)

// qtyEpsilon absorbs float64 noise when comparing refunded quantities; menu
// quantities are human-entered (1, 2, 0.5), so anything below this is zero.
const qtyEpsilon = 1e-9

// chainHead returns the next ICV and the PIH to chain onto, reading the
// current head across EVERY document type this EGS unit issues (invoices AND
// credit notes share one ZATCA counter/hash chain). Must be called inside
// the writer transaction that persists the new document, so two concurrent
// issuances cannot fork the chain.
func chainHead(tx *gorm.DB) (icv int64, pih string, err error) {
	icv, pih = 1, saudi.GenesisPIH

	var lastInv Invoice
	switch err := tx.Order("icv DESC").First(&lastInv).Error; {
	case err == nil:
		icv, pih = lastInv.ICV+1, lastInv.HashB64
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return 0, "", err
	}

	var lastCN CreditNote
	switch err := tx.Order("icv DESC").First(&lastCN).Error; {
	case err == nil:
		if lastCN.ICV >= icv {
			icv, pih = lastCN.ICV+1, lastCN.HashB64
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return 0, "", err
	}
	return icv, pih, nil
}

// LineRefund names one order line and the quantity of it to credit.
type LineRefund struct {
	OrderLineID uint
	Qty         float64
}

// RefundInvoice fully refunds a PAID invoice with a ZATCA credit note
// (TypeCode 381): manager-PIN + kernel-authority gated, BillingReference to
// the original, mandatory InstructionNote (the reason), chained on the same
// ICV/PIH sequence as invoices, and a NEGATIVE tender row so the refund
// lands in the business date's day-close reconciliation.
//
// refundMethod names the tender the money leaves by ("cash", "card", …).
// One full refund per invoice; an invoice that already carries partial
// credit notes must be finished line-by-line (RefundInvoiceLines).
func (s *Service) RefundInvoice(invoiceID uint, refundMethod string, by actor.Actor, pin, reason string) (*CreditNote, error) {
	if err := s.gateRefund(by, pin, &reason, &refundMethod); err != nil {
		return nil, err
	}

	issuedAt := s.now()
	var stored CreditNote
	err := s.db.Transaction(func(tx *gorm.DB) error {
		inv, err := refundableInvoice(tx, invoiceID)
		if err != nil {
			return err
		}

		refunded, err := refundedQuantities(tx, inv.ID)
		if err != nil {
			return err
		}
		if len(refunded) > 0 {
			return fmt.Errorf("hospitality: invoice %s already has partial credit notes; refund the remaining lines with RefundInvoiceLines", inv.Number)
		}

		// Re-bill the ORIGINAL lines onto the credit note: same snapshotted
		// names, quantities, prices and rates the invoice carried.
		lines, err := billableLines(tx, inv)
		if err != nil {
			return err
		}

		requireTotal := inv.TotalHalalas
		stored, err = s.issueCreditNoteTx(tx, inv, lines, refundMethod, by, reason, issuedAt, &requireTotal)
		if err != nil {
			return err
		}

		inv.Status = InvoiceRefunded
		return tx.Save(inv).Error
	})
	if err != nil {
		return nil, err
	}
	s.publishCreditNoteIssued(stored, issuedAt)
	return &stored, nil
}

// RefundInvoiceLines partially refunds a PAID invoice: a ZATCA credit note
// covering the named quantities of the named lines, under the same gates,
// chain and negative-tender discipline as a full refund. Several partial
// credit notes may be issued against one invoice; the per-line refund
// ledger (hosp_credit_note_lines) caps each line at its billed quantity.
// When the last billed quantity is credited, the invoice flips to refunded.
func (s *Service) RefundInvoiceLines(invoiceID uint, refunds []LineRefund, refundMethod string, by actor.Actor, pin, reason string) (*CreditNote, error) {
	if err := s.gateRefund(by, pin, &reason, &refundMethod); err != nil {
		return nil, err
	}
	if len(refunds) == 0 {
		return nil, errors.New("hospitality: a line refund needs at least one line")
	}

	issuedAt := s.now()
	var stored CreditNote
	err := s.db.Transaction(func(tx *gorm.DB) error {
		inv, err := refundableInvoice(tx, invoiceID)
		if err != nil {
			return err
		}
		billed, err := billableLines(tx, inv)
		if err != nil {
			return err
		}
		byID := make(map[uint]OrderLine, len(billed))
		for _, l := range billed {
			byID[l.ID] = l
		}
		refunded, err := refundedQuantities(tx, inv.ID)
		if err != nil {
			return err
		}

		seen := make(map[uint]bool, len(refunds))
		creditLines := make([]OrderLine, 0, len(refunds))
		for _, r := range refunds {
			line, ok := byID[r.OrderLineID]
			if !ok {
				return fmt.Errorf("hospitality: invoice %s has no billable line %d", inv.Number, r.OrderLineID)
			}
			if seen[r.OrderLineID] {
				return fmt.Errorf("hospitality: line %d appears twice in the refund request", r.OrderLineID)
			}
			seen[r.OrderLineID] = true
			if r.Qty <= 0 {
				return fmt.Errorf("hospitality: refund quantity for line %d must be positive", r.OrderLineID)
			}
			remaining := line.Qty - refunded[line.ID]
			if r.Qty > remaining+qtyEpsilon {
				return fmt.Errorf("hospitality: line %d has only %g of %g left to refund (requested %g)",
					line.ID, remaining, line.Qty, r.Qty)
			}
			partial := line // snapshot copy, then override the quantity
			partial.Qty = r.Qty
			creditLines = append(creditLines, partial)
		}

		stored, err = s.issueCreditNoteTx(tx, inv, creditLines, refundMethod, by, reason, issuedAt, nil)
		if err != nil {
			return err
		}

		// The refund ledger is quantity-truth; amounts are per-document ZATCA
		// arithmetic. Cumulative credited amounts may therefore round a
		// halala differently than one big document would — but they must
		// NEVER exceed what the guest paid.
		var creditedTotal int64
		if err := tx.Model(&CreditNote{}).Where("original_invoice_id = ?", inv.ID).
			Select("COALESCE(SUM(total_halalas), 0)").Scan(&creditedTotal).Error; err != nil {
			return err
		}
		if creditedTotal > inv.TotalHalalas {
			return fmt.Errorf("hospitality: cumulative credit notes (%d halalas) would exceed invoice total (%d halalas) — rounding drift; refund the remainder as a manual document",
				creditedTotal, inv.TotalHalalas)
		}

		// All billed quantity credited → the invoice is fully refunded.
		for _, r := range creditLines {
			refunded[r.ID] += r.Qty
		}
		fullyRefunded := true
		for _, l := range billed {
			if l.Qty-refunded[l.ID] > qtyEpsilon {
				fullyRefunded = false
				break
			}
		}
		if fullyRefunded {
			inv.Status = InvoiceRefunded
			return tx.Save(inv).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishCreditNoteIssued(stored, issuedAt)
	return &stored, nil
}

// gateRefund runs the shared refund gates: kernel authority (an AI agent can
// NEVER issue a credit note), manager PIN, mandatory reason and tender.
func (s *Service) gateRefund(by actor.Actor, pin string, reason, refundMethod *string) error {
	if !by.CanApprove() {
		return fmt.Errorf("hospitality: actor %q (type %s) lacks authority to issue a credit note", by.ID, by.Type)
	}
	if err := s.VerifyManagerPIN(pin); err != nil {
		return err
	}
	*reason = strings.TrimSpace(*reason)
	if *reason == "" {
		return errors.New("hospitality: a credit note requires a reason (ZATCA InstructionNote)")
	}
	*refundMethod = strings.ToLower(strings.TrimSpace(*refundMethod))
	if *refundMethod == "" {
		return errors.New("hospitality: refund method is required")
	}
	return nil
}

// refundableInvoice loads an invoice and checks it is in a refundable state.
func refundableInvoice(tx *gorm.DB, invoiceID uint) (*Invoice, error) {
	var inv Invoice
	if err := tx.First(&inv, invoiceID).Error; err != nil {
		return nil, fmt.Errorf("hospitality: invoice %d: %w", invoiceID, err)
	}
	switch inv.Status {
	case InvoicePaid:
		return &inv, nil
	case InvoiceRefunded:
		return nil, fmt.Errorf("hospitality: invoice %s is already refunded", inv.Number)
	default:
		return nil, fmt.Errorf("hospitality: only paid invoices can be refunded (invoice %s is %s)", inv.Number, inv.Status)
	}
}

// billableLines loads the non-voided lines the invoice billed. Lines are
// stamped with their invoice at issuance (Wave 5 C.1 — a split session
// issues several invoices over one session); invoices issued before the
// stamp existed fall back to the historical session-scoped lookup, which
// is exact for them because pre-split sessions have exactly one invoice
// (the fallback filters invoice_id IS NULL so a later split on the same
// database can never leak lines across the boundary).
func billableLines(tx *gorm.DB, inv *Invoice) ([]OrderLine, error) {
	var lines []OrderLine
	if err := tx.Where("invoice_id = ? AND status <> ?", inv.ID, LineVoided).Find(&lines).Error; err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		if err := tx.Where("session_id = ? AND invoice_id IS NULL AND status <> ?", inv.SessionID, LineVoided).Find(&lines).Error; err != nil {
			return nil, err
		}
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("hospitality: invoice %s has no lines to credit", inv.Number)
	}
	return lines, nil
}

// refundedQuantities sums, per order line, the quantity already credited by
// earlier credit notes against this invoice.
func refundedQuantities(tx *gorm.DB, invoiceID uint) (map[uint]float64, error) {
	type row struct {
		OrderLineID uint
		Qty         float64
	}
	var rows []row
	err := tx.Model(&CreditNoteLine{}).
		Select("hosp_credit_note_lines.order_line_id AS order_line_id, SUM(hosp_credit_note_lines.qty) AS qty").
		Joins("JOIN hosp_credit_notes ON hosp_credit_notes.id = hosp_credit_note_lines.credit_note_id").
		Where("hosp_credit_notes.original_invoice_id = ?", invoiceID).
		Group("hosp_credit_note_lines.order_line_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]float64, len(rows))
	for _, r := range rows {
		out[r.OrderLineID] = r.Qty
	}
	return out, nil
}

// issueCreditNoteTx builds, signs, and persists one credit note over the
// given (possibly partial-quantity) lines, plus its refund ledger rows and
// the NEGATIVE tender row that nets it into the day close. When
// requireTotalHalalas is non-nil the signed document's total must equal it
// exactly (the full-refund invariant).
func (s *Service) issueCreditNoteTx(tx *gorm.DB, inv *Invoice, lines []OrderLine, refundMethod string, by actor.Actor, reason string, issuedAt time.Time, requireTotalHalalas *int64) (CreditNote, error) {
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

	number, err := numbering.NextInTx(tx, creditNoteNumberSpec, issuedAt)
	if err != nil {
		return CreditNote{}, err
	}
	icv, pih, err := chainHead(tx)
	if err != nil {
		return CreditNote{}, err
	}

	einv := &saudi.EInvoice{
		ID:                 number,
		UUID:               uuid.NewString(),
		IssuedAt:           issuedAt,
		TypeCode:           saudi.TypeCreditNote,
		Subtype:            saudi.Subtype{Simplified: true},
		Currency:           s.ov.Currency,
		ICV:                icv,
		PIH:                pih,
		Seller:             seller,
		BillingReferenceID: inv.Number,
		InstructionNote:    reason,
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
		return CreditNote{}, fmt.Errorf("hospitality: ZATCA credit-note signing failed: %w", err)
	}
	totals := einv.ComputeTotals()
	if requireTotalHalalas != nil && halalas(totals.TaxInclusive) != *requireTotalHalalas {
		// The credit note re-bills the invoice's own lines, so totals must
		// agree to the halala — a mismatch means the data drifted.
		return CreditNote{}, fmt.Errorf("hospitality: credit note total %d does not match invoice total %d halalas",
			halalas(totals.TaxInclusive), *requireTotalHalalas)
	}

	stored := CreditNote{
		Number:            number,
		OriginalInvoiceID: inv.ID,
		OriginalNumber:    inv.Number,
		UUID:              einv.UUID,
		IssuedAt:          issuedAt,
		SubtotalHalalas:   halalas(totals.LineExtension),
		VATHalalas:        halalas(totals.TaxAmount),
		TotalHalalas:      halalas(totals.TaxInclusive),
		Currency:          einv.Currency,
		ICV:               icv,
		PIH:               pih,
		HashB64:           signed.InvoiceHashB64,
		QRBase64:          signed.QRBase64,
		XML:               signed.XML,
		Reason:            reason,
		IssuedByID:        by.ID,
		IssuedByName:      by.DisplayName,
	}
	if err := tx.Create(&stored).Error; err != nil {
		return CreditNote{}, err
	}
	for _, l := range lines {
		ledger := CreditNoteLine{
			CreditNoteID:     stored.ID,
			OrderLineID:      l.ID,
			Name:             l.Name,
			Qty:              l.Qty,
			UnitPriceHalalas: l.UnitPriceHalalas,
			TaxRate:          l.TaxRate,
		}
		if err := tx.Create(&ledger).Error; err != nil {
			return CreditNote{}, err
		}
	}

	// The refund leaves the drawer: a NEGATIVE tender row on today's
	// business date, so ExpectedTenders — and therefore the day close —
	// reconciles the net movement.
	refund := Payment{
		InvoiceID:     inv.ID,
		CreditNoteID:  &stored.ID,
		Method:        refundMethod,
		AmountHalalas: -stored.TotalHalalas,
		ReceivedAt:    issuedAt,
		BusinessDate:  issuedAt.Format("2006-01-02"),
	}
	if err := tx.Create(&refund).Error; err != nil {
		return CreditNote{}, err
	}
	return stored, nil
}

// publishCreditNoteIssued emits the credit note's OWN domain event after
// commit (W4 C.2 — deliberately not smuggled under InvoiceCreated: the
// compliance hook validates the same rate arithmetic, but subscribers can
// tell money-out from money-in).
func (s *Service) publishCreditNoteIssued(cn CreditNote, issuedAt time.Time) {
	if s.bus == nil {
		return
	}
	profile := s.ov.Profile(s.ov.DefaultDivision())
	_ = s.bus.Publish(context.Background(), events.CreditNoteIssued{
		BaseEvent:             events.BaseEvent{Timestamp: issuedAt, CorrelationID: cn.UUID},
		CreditNoteID:          fmt.Sprintf("%d", cn.ID),
		CreditNoteNumber:      cn.Number,
		OriginalInvoiceNumber: cn.OriginalNumber,
		IssueDate:             cn.IssuedAt,
		SellerTaxID:           profile.VATNumber,
		// Amount is the NET taxable base, positive magnitude — the direction
		// is the event type (same convention as the stored document).
		Amount:       float64(cn.SubtotalHalalas) / 100,
		TaxAmount:    float64(cn.VATHalalas) / 100,
		Currency:     cn.Currency,
		Jurisdiction: s.ov.JurisdictionCode(),
	})
}
