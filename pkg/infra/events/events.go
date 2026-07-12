package events

import "time"

const (
	EventInvoiceCreated   = "finance.invoice.created"
	EventInvoiceUpdated   = "finance.invoice.updated"
	EventPaymentRecorded  = "finance.payment.recorded"
	EventExpenseCreated   = "finance.expense.created"
	EventCreditNoteIssued = "finance.credit_note.issued"
)

// BaseEvent carries metadata common to all domain events.
type BaseEvent struct {
	Timestamp     time.Time
	CorrelationID string
}

type InvoiceCreated struct {
	BaseEvent
	InvoiceID string

	// Compliance-relevant fields, populated by the publisher so subscribers
	// (e.g. pkg/compliance) can validate without a database round-trip.
	InvoiceNumber string
	InvoiceDate   time.Time
	SellerTaxID   string // seller TRN/VAT number (overlay division identity)
	BuyerTaxID    string // buyer TRN, when known
	Amount        float64
	TaxAmount     float64
	Currency      string
	Jurisdiction  string // optional explicit jurisdiction tag; blank defers to currency
}

func (InvoiceCreated) Name() string { return EventInvoiceCreated }

// ComplianceData implements the carrier interface the compliance hook looks for.
// It exposes the invoice's tax-relevant fields as the map shape the hook expects.
func (e InvoiceCreated) ComplianceData() map[string]any {
	return map[string]any{
		"invoice_number": e.InvoiceNumber,
		"invoice_date":   e.InvoiceDate,
		"seller_tax_id":  e.SellerTaxID,
		"buyer_tax_id":   e.BuyerTaxID,
		"amount":         e.Amount,
		"tax_amount":     e.TaxAmount,
		"currency":       e.Currency,
		"jurisdiction":   e.Jurisdiction,
	}
}

// CreditNoteIssued is published when a credit note (a negative-direction tax
// document referencing an original invoice) is issued. It is deliberately its
// OWN event — a credit note smuggled under InvoiceCreated would validate, but
// subscribers could no longer tell money-out from money-in. Amounts carry
// positive magnitudes; the event type conveys direction.
type CreditNoteIssued struct {
	BaseEvent
	CreditNoteID string

	CreditNoteNumber      string
	OriginalInvoiceNumber string
	IssueDate             time.Time
	SellerTaxID           string
	Amount                float64 // NET taxable base, positive magnitude
	TaxAmount             float64
	Currency              string
	Jurisdiction          string
}

func (CreditNoteIssued) Name() string { return EventCreditNoteIssued }

// ComplianceData implements the carrier interface the compliance hook looks
// for. Credit notes validate under the same rate arithmetic as invoices, so
// the tax-relevant fields map onto the same keys; the referenced original is
// carried alongside.
func (e CreditNoteIssued) ComplianceData() map[string]any {
	return map[string]any{
		"invoice_number":          e.CreditNoteNumber,
		"invoice_date":            e.IssueDate,
		"seller_tax_id":           e.SellerTaxID,
		"amount":                  e.Amount,
		"tax_amount":              e.TaxAmount,
		"currency":                e.Currency,
		"jurisdiction":            e.Jurisdiction,
		"original_invoice_number": e.OriginalInvoiceNumber,
	}
}

type PaymentRecorded struct {
	BaseEvent
	PaymentID string
	InvoiceID string
}

func (PaymentRecorded) Name() string { return EventPaymentRecorded }

type OfferWon struct {
	BaseEvent
	OfferID string
}

func (OfferWon) Name() string { return "crm.offer.won" }

type OfferLost struct {
	BaseEvent
	OfferID string
	Reason  string
}

func (OfferLost) Name() string { return "crm.offer.lost" }

type DocumentClassified struct {
	BaseEvent
	DocumentID   string
	DocumentType string
}

func (DocumentClassified) Name() string { return "documents.document.classified" }

type BankStatementImported struct {
	BaseEvent
	BankStatementID string
}

func (BankStatementImported) Name() string { return "finance.bank_statement.imported" }

type DeliveryNoteCreated struct {
	BaseEvent
	DeliveryNoteID string
}

func (DeliveryNoteCreated) Name() string { return "crm.delivery_note.created" }

type SerialNumberRegistered struct {
	BaseEvent
	SerialNumberID string
	SerialNo       string
}

func (SerialNumberRegistered) Name() string { return "crm.serial_number.registered" }
