package main

// Customer-invoice payment/settlement policy — ported from deployed PH in
// Mission G (Wave 4) to close two parity gaps the G.1 audit surfaced:
//  1. OSS's inline non-payable map omitted "Draft", so a payment could be
//     recorded against an unsent Draft invoice. canRecordCustomerInvoicePayment
//     closes it (Draft is a closed workflow status → not open → not payable).
//  2. OSS never recomputed settlement status on read, so a past-due invoice
//     never surfaced as "Overdue" until a mutator ran. hydrate* recomputes
//     Status + Outstanding on read, matching PH exactly.
// The math (BHD rounding, tolerance) is identical to PH; behavior is parity.

import (
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

type customerInvoicePaymentState struct {
	InvoiceID      string
	OutstandingBHD float64
	Status         string
	IsOpen         bool
	IsCollectible  bool
	IsOverdue      bool
}

var customerInvoiceClosedWorkflowStatuses = map[string]bool{
	"Draft":     true,
	"Cancelled": true,
	"Void":      true,
	"Proforma":  true,
}

func customerInvoiceOutstandingBHD(outstanding float64) float64 {
	rounded := math.Round(outstanding*BHDPrecisionMultiplier) / BHDPrecisionMultiplier
	if math.Abs(rounded) <= FloatingPointTolerance {
		return 0
	}
	if rounded < 0 {
		return 0
	}
	return rounded
}

func customerInvoiceSettlementStatus(invoice Invoice, outstanding float64, asOf time.Time) string {
	currentStatus := strings.TrimSpace(invoice.Status)
	if customerInvoiceClosedWorkflowStatuses[currentStatus] {
		return currentStatus
	}

	if outstanding <= FloatingPointTolerance {
		return "Paid"
	}

	if invoice.GrandTotalBHD-outstanding > FloatingPointTolerance {
		return "PartiallyPaid"
	}

	if !invoice.DueDate.IsZero() && asOf.After(invoice.DueDate) {
		return "Overdue"
	}

	return "Sent"
}

func customerInvoicePaymentStateFromInvoice(invoice Invoice, asOf time.Time) customerInvoicePaymentState {
	outstanding := customerInvoiceOutstandingBHD(invoice.OutstandingBHD)
	status := customerInvoiceSettlementStatus(invoice, outstanding, asOf)
	isOpen := outstanding > FloatingPointTolerance && !customerInvoiceClosedWorkflowStatuses[status]

	return customerInvoicePaymentState{
		InvoiceID:      invoice.ID,
		OutstandingBHD: outstanding,
		Status:         status,
		IsOpen:         isOpen,
		IsCollectible:  isOpen,
		IsOverdue:      isOpen && !invoice.DueDate.IsZero() && asOf.After(invoice.DueDate),
	}
}

func canRecordCustomerInvoicePayment(invoice Invoice, asOf time.Time) bool {
	state := customerInvoicePaymentStateFromInvoice(invoice, asOf)
	return state.IsOpen
}

func hydrateCustomerInvoicePaymentState(invoice *Invoice) {
	if invoice == nil {
		return
	}

	state := customerInvoicePaymentStateFromInvoice(*invoice, time.Now())
	invoice.OutstandingBHD = state.OutstandingBHD
	invoice.Status = state.Status
}

func hydrateCustomerInvoicesPaymentState(invoices []Invoice) {
	for i := range invoices {
		hydrateCustomerInvoicePaymentState(&invoices[i])
	}
}

func isCustomerInvoiceSettlementStatus(status string) bool {
	switch status {
	case "Paid", "PartiallyPaid", "Overdue":
		return true
	default:
		return false
	}
}

func isCustomerInvoicePostedStatus(status string) bool {
	switch status {
	case "Sent", "Paid", "PartiallyPaid", "Overdue":
		return true
	default:
		return false
	}
}

func (a *App) applyCustomerInvoicePaymentState(tx *gorm.DB, invoice *Invoice) (customerInvoicePaymentState, error) {
	if invoice == nil {
		return customerInvoicePaymentState{}, nil
	}

	state := customerInvoicePaymentStateFromInvoice(*invoice, time.Now())
	now := time.Now()
	updates := map[string]any{
		"outstanding_bhd": state.OutstandingBHD,
		"status":          state.Status,
		"updated_at":      now,
	}

	if err := tx.Model(invoice).Updates(updates).Error; err != nil {
		return customerInvoicePaymentState{}, err
	}

	invoice.OutstandingBHD = state.OutstandingBHD
	invoice.Status = state.Status
	invoice.UpdatedAt = now

	return state, nil
}
