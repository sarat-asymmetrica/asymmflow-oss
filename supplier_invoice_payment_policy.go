package main

// Wave 8 P1: supplier-invoice payment-state policy, ported from deployed PH so a
// supplier invoice's paid/outstanding state is DERIVED from the SupplierPayment
// ledger (source of truth), not a bare status flag. MarkSupplierInvoicePaid uses
// createRemainingSupplierInvoicePayment + applySupplierInvoicePaymentState so that
// SUM(payments) == TotalBHD after marking paid. The hydrate* helpers are available
// for list/read parity (wired in a later pass).

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

type supplierInvoicePaymentState struct {
	InvoiceID         string
	TotalPaidBHD      float64
	OutstandingBHD    float64
	PaymentStatus     string
	InvoiceStatus     string
	LatestPaymentDate *time.Time
	LatestPaymentRef  string
	LatestPaymentType string
}

type supplierInvoicePaymentAggregate struct {
	SupplierInvoiceID string
	TotalPaidBHD      float64
}

type supplierInvoicePaymentTotals struct {
	AmountBHD     float64
	AmountForeign float64
}

func supplierInvoiceOutstandingBHD(invoice SupplierInvoice, totalPaidBHD float64) float64 {
	outstanding := invoice.TotalBHD - totalPaidBHD
	if math.Abs(outstanding) <= FloatingPointTolerance {
		return 0
	}
	if outstanding < 0 {
		return 0
	}
	return outstanding
}

func supplierInvoiceNonPaymentStatus(invoice SupplierInvoice) string {
	switch invoice.Status {
	case "Rejected", "Dispute", "Disputed":
		// Exceptional workflow states are preserved (OSS spells it "Disputed";
		// "Dispute" survives for legacy/synced PH rows).
		return invoice.Status
	case "Approved", "Paid":
		return "Approved"
	case "Verified":
		// Wave 8 P5-1: "Verified" (clean 3-way match, awaiting the explicit
		// approval step) is a workflow state of its own in the OSS vocabulary —
		// the PH-ported policy used to collapse it to "Pending" on hydrate,
		// silently undoing the match result.
		if invoice.ApprovedAt != nil || invoice.ApprovedBy != "" {
			return "Approved"
		}
		return "Verified"
	default:
		if invoice.ApprovedAt != nil || invoice.ApprovedBy != "" {
			return "Approved"
		}
		return "Pending"
	}
}

func supplierInvoicePaymentStateFromLedger(invoice SupplierInvoice, totalPaidBHD float64, latestPayment *SupplierPayment) supplierInvoicePaymentState {
	outstanding := supplierInvoiceOutstandingBHD(invoice, totalPaidBHD)

	state := supplierInvoicePaymentState{
		InvoiceID:      invoice.ID,
		TotalPaidBHD:   totalPaidBHD,
		OutstandingBHD: outstanding,
		InvoiceStatus:  supplierInvoiceNonPaymentStatus(invoice),
		PaymentStatus:  "Unpaid",
	}

	if totalPaidBHD > FloatingPointTolerance {
		state.PaymentStatus = "Partial"
	}
	if outstanding == 0 {
		state.PaymentStatus = "Paid"
	}

	switch state.InvoiceStatus {
	case "Rejected", "Dispute":
		// Preserve exceptional workflow states even if legacy data contains payments.
	default:
		if state.PaymentStatus == "Paid" {
			state.InvoiceStatus = "Paid"
		}
	}

	if latestPayment != nil {
		paymentDate := latestPayment.PaymentDate
		state.LatestPaymentDate = &paymentDate
		state.LatestPaymentRef = latestPayment.Reference
		state.LatestPaymentType = latestPayment.PaymentMethod
	}

	return state
}

func (a *App) getSupplierInvoicePaymentTotals(tx *gorm.DB, invoiceID string) (supplierInvoicePaymentTotals, error) {
	var totals supplierInvoicePaymentTotals
	err := tx.Model(&SupplierPayment{}).
		Where("supplier_invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount_bhd), 0) AS amount_bhd, COALESCE(SUM(amount_foreign), 0) AS amount_foreign").
		Scan(&totals).Error
	return totals, err
}

func (a *App) loadSupplierInvoicePaymentState(tx *gorm.DB, invoice SupplierInvoice) (supplierInvoicePaymentState, error) {
	totals, err := a.getSupplierInvoicePaymentTotals(tx, invoice.ID)
	if err != nil {
		return supplierInvoicePaymentState{}, err
	}

	var latestPayment SupplierPayment
	var latest *SupplierPayment
	if err := tx.Where("supplier_invoice_id = ?", invoice.ID).
		Order("payment_date DESC").
		Order("updated_at DESC").
		First(&latestPayment).Error; err == nil {
		latest = &latestPayment
	} else if err != gorm.ErrRecordNotFound {
		return supplierInvoicePaymentState{}, err
	}

	return supplierInvoicePaymentStateFromLedger(invoice, totals.AmountBHD, latest), nil
}

func (a *App) applySupplierInvoicePaymentState(tx *gorm.DB, invoiceID string) (supplierInvoicePaymentState, error) {
	var invoice SupplierInvoice
	if err := tx.Where("id = ?", invoiceID).First(&invoice).Error; err != nil {
		return supplierInvoicePaymentState{}, err
	}

	state, err := a.loadSupplierInvoicePaymentState(tx, invoice)
	if err != nil {
		return supplierInvoicePaymentState{}, err
	}

	updates := map[string]any{
		"payment_status": state.PaymentStatus,
		"status":         state.InvoiceStatus,
		"updated_at":     time.Now(),
	}

	if state.LatestPaymentDate != nil {
		updates["payment_date"] = *state.LatestPaymentDate
		updates["payment_ref"] = state.LatestPaymentRef
		updates["payment_method"] = state.LatestPaymentType
	} else {
		updates["payment_date"] = nil
		updates["payment_ref"] = ""
		updates["payment_method"] = ""
	}

	if err := tx.Model(&SupplierInvoice{}).Where("id = ?", invoiceID).Updates(updates).Error; err != nil {
		return supplierInvoicePaymentState{}, err
	}

	return state, nil
}

func (a *App) buildSupplierInvoicePaymentStateMap(tx *gorm.DB, invoices []SupplierInvoice) (map[string]supplierInvoicePaymentState, error) {
	states := make(map[string]supplierInvoicePaymentState, len(invoices))
	if len(invoices) == 0 {
		return states, nil
	}

	ids := make([]string, 0, len(invoices))
	for _, invoice := range invoices {
		ids = append(ids, invoice.ID)
	}

	var aggregates []supplierInvoicePaymentAggregate
	if err := tx.Model(&SupplierPayment{}).
		Where("supplier_invoice_id IN ?", ids).
		Select("supplier_invoice_id, COALESCE(SUM(amount_bhd), 0) AS total_paid_bhd").
		Group("supplier_invoice_id").
		Scan(&aggregates).Error; err != nil {
		return nil, err
	}

	paidByInvoiceID := make(map[string]float64, len(aggregates))
	for _, aggregate := range aggregates {
		paidByInvoiceID[aggregate.SupplierInvoiceID] = aggregate.TotalPaidBHD
	}

	for _, invoice := range invoices {
		states[invoice.ID] = supplierInvoicePaymentStateFromLedger(invoice, paidByInvoiceID[invoice.ID], nil)
	}

	return states, nil
}

func (a *App) hydrateSupplierInvoicePaymentState(tx *gorm.DB, invoice *SupplierInvoice) error {
	if invoice == nil || strings.TrimSpace(invoice.ID) == "" {
		return nil
	}

	state, err := a.loadSupplierInvoicePaymentState(tx, *invoice)
	if err != nil {
		return err
	}

	invoice.PaymentStatus = state.PaymentStatus
	invoice.OutstandingBHD = state.OutstandingBHD
	if state.InvoiceStatus != "" {
		invoice.Status = state.InvoiceStatus
	}
	if state.LatestPaymentDate != nil {
		paymentDate := *state.LatestPaymentDate
		invoice.PaymentDate = &paymentDate
		invoice.PaymentRef = state.LatestPaymentRef
		invoice.PaymentMethod = state.LatestPaymentType
	} else {
		invoice.PaymentDate = nil
		invoice.PaymentRef = ""
		invoice.PaymentMethod = ""
	}

	return nil
}

func (a *App) hydrateSupplierInvoicesPaymentState(tx *gorm.DB, invoices []SupplierInvoice) error {
	if len(invoices) == 0 {
		return nil
	}

	states, err := a.buildSupplierInvoicePaymentStateMap(tx, invoices)
	if err != nil {
		return err
	}

	for i := range invoices {
		state := states[invoices[i].ID]
		invoices[i].PaymentStatus = state.PaymentStatus
		invoices[i].OutstandingBHD = state.OutstandingBHD
		if state.InvoiceStatus != "" {
			invoices[i].Status = state.InvoiceStatus
		}
	}

	return nil
}

func (a *App) createRemainingSupplierInvoicePayment(tx *gorm.DB, invoice SupplierInvoice, paymentDate time.Time, paymentMethod string, paymentRef string, notes string) (bool, error) {
	totals, err := a.getSupplierInvoicePaymentTotals(tx, invoice.ID)
	if err != nil {
		return false, err
	}

	remainingBHD := supplierInvoiceOutstandingBHD(invoice, totals.AmountBHD)
	if remainingBHD <= FloatingPointTolerance {
		return false, nil
	}

	currency := invoice.Currency
	if currency == "" {
		currency = "BHD"
	}

	exchangeRate := invoice.ExchangeRate
	if exchangeRate <= 0 {
		exchangeRate = 1
	}

	amountForeign := remainingBHD
	if currency != "BHD" {
		remainingForeign := invoice.TotalForeign - totals.AmountForeign
		if remainingForeign > FloatingPointTolerance {
			amountForeign = remainingForeign
		} else {
			amountForeign = remainingBHD / exchangeRate
		}
	}

	if paymentMethod == "" {
		paymentMethod = "Other"
	}

	payment := SupplierPayment{
		SupplierInvoiceID: invoice.ID,
		SupplierID:        invoice.SupplierID,
		AmountForeign:     amountForeign,
		Currency:          currency,
		ExchangeRate:      exchangeRate,
		AmountBHD:         remainingBHD,
		PaymentDate:       paymentDate,
		PaymentMethod:     paymentMethod,
		Reference:         paymentRef,
		Notes:             notes,
		SupplierName:      invoice.SupplierName,
		InvoiceNumber:     invoice.InvoiceNumber,
		Division:          a.resolveSupplierInvoiceDivision(invoice),
	}

	if err := tx.Create(&payment).Error; err != nil {
		return false, err
	}

	return true, nil
}
