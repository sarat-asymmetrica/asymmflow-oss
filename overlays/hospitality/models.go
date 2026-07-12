// Package hospitality is the SECOND vertical built on the AsymmFlow substrate
// — the Wave 2 composition proof. It is a Saudi café/restaurant point-of-sale
// domain (order sessions, kitchen tickets, simplified ZATCA e-invoices,
// tender-reconciled day close) that contains NO trading vocabulary and writes
// NO new engine code: everything generic comes from pkg/.
//
// What it composes:
//
//	pkg/kernel/{money,actor}        exact arithmetic + the AI-authority boundary
//	pkg/overlay                     deployment identity (seller, currency, jurisdiction)
//	pkg/documents/numbering         invoice + kitchen-ticket numbers
//	pkg/finance/settlement          day-close tender reconciliation
//	pkg/infra/auth                  manager-PIN gate for voids and day close
//	pkg/infra/events                domain events out, compliance in
//	pkg/compliance(+saudi)          15% VAT validation + ZATCA Phase 2 signing
//
// The models below are the vertical's OWN vocabulary — the substrate never
// sees them. All table names carry a hosp_ prefix so the vertical can share a
// database file with another vertical without collision (the one deliberate
// exception is the numbering engine's invoice_sequences table, which is shared
// by design and keyed by prefix).
package hospitality

import "time"

// MenuItem is a sellable item. UnitPriceHalalas is the NET price (before VAT)
// in SAR minor units; menus that display VAT-inclusive prices derive the
// display price, the books keep net + rate.
type MenuItem struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	Name             string  `gorm:"size:120;uniqueIndex" json:"name"`
	Category         string  `gorm:"size:40" json:"category"` // menu grouping: "beverage", "food", ...
	UnitPriceHalalas int64   `gorm:"not null" json:"unit_price_halalas"`
	TaxRate          float64 `gorm:"not null" json:"tax_rate"` // 0.15 standard; 0 only with an exemption reason
	Active           bool    `gorm:"not null;default:true" json:"active"`
}

func (MenuItem) TableName() string { return "hosp_menu_items" }

// DiningTable is a physical table an order session attaches to.
type DiningTable struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Code  string `gorm:"size:20;uniqueIndex" json:"code"`
	Seats int    `json:"seats"`
}

func (DiningTable) TableName() string { return "hosp_tables" }

// Session states.
const (
	SessionOpen   = "open"
	SessionClosed = "closed"
)

// OrderSession is one visit at one table: lines accumulate, kitchen tickets
// dispatch, and closing the session issues the invoice.
type OrderSession struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	TableID   uint       `gorm:"index;not null" json:"table_id"`
	Status    string     `gorm:"size:12;not null;index" json:"status"`
	OpenedAt  time.Time  `json:"opened_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	InvoiceID *uint      `json:"invoice_id"`
}

func (OrderSession) TableName() string { return "hosp_order_sessions" }

// Order-line states.
const (
	LinePending = "pending" // not yet on a kitchen ticket
	LineSent    = "sent"    // dispatched on a ticket
	LineVoided  = "voided"  // manager-voided; excluded from the invoice
)

// OrderLine is one item on a session. Name/price/rate are snapshotted from the
// menu item at order time so later menu edits never rewrite an open bill.
//
// InvoiceID (Wave 5 C.1) records which invoice billed this line, stamped
// at issuance: a split session issues several invoices, and refunds must
// scope their line ledger to ONE of them. Lines billed before Wave 5
// carry NULL — those invoices predate bill split, so their session has
// exactly one invoice and the legacy session-scoped lookup stays correct.
type OrderLine struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	SessionID        uint    `gorm:"index;not null" json:"session_id"`
	MenuItemID       uint    `gorm:"not null" json:"menu_item_id"`
	InvoiceID        *uint   `gorm:"index" json:"invoice_id"`
	Name             string  `gorm:"size:120;not null" json:"name"`
	Qty              float64 `gorm:"not null" json:"qty"`
	UnitPriceHalalas int64   `gorm:"not null" json:"unit_price_halalas"`
	TaxRate          float64 `gorm:"not null" json:"tax_rate"`
	Status           string  `gorm:"size:12;not null" json:"status"`
	VoidReason       string  `gorm:"size:200" json:"void_reason"`
	VoidedByID       string  `gorm:"size:60" json:"voided_by_id"`
}

func (OrderLine) TableName() string { return "hosp_order_lines" }

// Kitchen-ticket states.
const (
	TicketQueued    = "queued"
	TicketPreparing = "preparing"
	TicketReady     = "ready"
	TicketServed    = "served"
	TicketCancelled = "cancelled"
)

// Ticket is a kitchen order ticket (KOT): the batch of lines dispatched to the
// kitchen in one send. Numbered by the shared numbering engine (prefix KOT).
type Ticket struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SessionID uint      `gorm:"index;not null" json:"session_id"`
	Number    string    `gorm:"size:30;uniqueIndex" json:"number"`
	Status    string    `gorm:"size:12;not null;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Ticket) TableName() string { return "hosp_tickets" }

// TicketLine snapshots one order line onto a ticket.
type TicketLine struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	TicketID    uint    `gorm:"index;not null" json:"ticket_id"`
	OrderLineID uint    `gorm:"not null" json:"order_line_id"`
	Name        string  `gorm:"size:120;not null" json:"name"`
	Qty         float64 `gorm:"not null" json:"qty"`
}

func (TicketLine) TableName() string { return "hosp_ticket_lines" }

// Invoice states.
const (
	InvoiceIssued   = "issued"
	InvoicePaid     = "paid"
	InvoiceRefunded = "refunded" // fully credited by a credit note
)

// Invoice is an issued simplified (B2C) tax invoice. The signed ZATCA XML,
// invoice hash, QR and the ICV/PIH chain live on the row: the hash chain makes
// the invoice history tamper-evident, so these are data, not derivable views.
type Invoice struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Number    string    `gorm:"size:30;uniqueIndex" json:"number"`
	SessionID uint      `gorm:"index;not null" json:"session_id"`
	UUID      string    `gorm:"size:40;uniqueIndex" json:"uuid"`
	IssuedAt  time.Time `json:"issued_at"`

	SubtotalHalalas int64  `gorm:"not null" json:"subtotal_halalas"` // net of VAT
	VATHalalas      int64  `gorm:"not null" json:"vat_halalas"`
	TotalHalalas    int64  `gorm:"not null" json:"total_halalas"`
	Currency        string `gorm:"size:3;not null" json:"currency"`

	ICV      int64  `gorm:"uniqueIndex;not null" json:"icv"` // ZATCA invoice counter, never resets
	PIH      string `gorm:"size:64;not null" json:"pih"`     // previous invoice hash
	HashB64  string `gorm:"size:64;not null" json:"hash_b64"`
	QRBase64 string `json:"qr_base64"`
	XML      []byte `json:"-"` // the signed UBL 2.1 document

	Status string `gorm:"size:12;not null;index" json:"status"`
}

func (Invoice) TableName() string { return "hosp_invoices" }

// Payment is one tender movement against an invoice. Sales tenders are
// positive; a refund paid out against a credit note is a NEGATIVE row
// (CreditNoteID set), so the day close reconciles the NET drawer movement.
// BusinessDate (YYYY-MM-DD) is stamped at capture time and is what the day
// close reconciles over.
type Payment struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	InvoiceID     uint      `gorm:"index;not null" json:"invoice_id"`
	CreditNoteID  *uint     `gorm:"index" json:"credit_note_id"`    // set on refund pay-outs
	Method        string    `gorm:"size:20;not null" json:"method"` // "cash", "card", ...
	AmountHalalas int64     `gorm:"not null" json:"amount_halalas"`
	ReceivedAt    time.Time `json:"received_at"`
	BusinessDate  string    `gorm:"size:10;index;not null" json:"business_date"`
}

func (Payment) TableName() string { return "hosp_payments" }

// CreditNote is an issued ZATCA credit note (TypeCode 381) refunding one
// invoice in full or in part. It participates in the SAME ICV/PIH hash chain
// as invoices — ZATCA chains every e-document an EGS unit issues, whatever
// its type — so the chain head must be read across both tables (see
// chainHead). Amount fields carry positive magnitudes; the document type
// conveys direction.
//
// W4 C.1: OriginalInvoiceID was uniqueIndex ("one full refund per invoice");
// partial line-level refunds mean several credit notes may reference one
// invoice, so the index is plain now (named idx_hosp_cn_original_invoice;
// the legacy unique index is dropped explicitly during migrate).
type CreditNote struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Number            string    `gorm:"size:30;uniqueIndex" json:"number"`
	OriginalInvoiceID uint      `gorm:"index:idx_hosp_cn_original_invoice;not null" json:"original_invoice_id"`
	OriginalNumber    string    `gorm:"size:30;not null" json:"original_number"`
	UUID              string    `gorm:"size:40;uniqueIndex" json:"uuid"`
	IssuedAt          time.Time `json:"issued_at"`

	SubtotalHalalas int64  `gorm:"not null" json:"subtotal_halalas"` // net of VAT
	VATHalalas      int64  `gorm:"not null" json:"vat_halalas"`
	TotalHalalas    int64  `gorm:"not null" json:"total_halalas"`
	Currency        string `gorm:"size:3;not null" json:"currency"`

	ICV      int64  `gorm:"uniqueIndex;not null" json:"icv"` // shared ZATCA counter with hosp_invoices
	PIH      string `gorm:"size:64;not null" json:"pih"`
	HashB64  string `gorm:"size:64;not null" json:"hash_b64"`
	QRBase64 string `json:"qr_base64"`
	XML      []byte `json:"-"`

	Reason       string `gorm:"size:200;not null" json:"reason"` // ZATCA InstructionNote
	IssuedByID   string `gorm:"size:60;not null" json:"issued_by_id"`
	IssuedByName string `gorm:"size:120" json:"issued_by_name"`
}

func (CreditNote) TableName() string { return "hosp_credit_notes" }

// CreditNoteLine records which order line, and how much of it, one credit
// note refunds. Name/price/rate are snapshotted from the order line (which
// itself snapshotted the menu item), so the refund ledger stays stable under
// later edits. The per-line ledger is what caps cumulative refunds: a
// quantity can never be credited twice.
type CreditNoteLine struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	CreditNoteID     uint    `gorm:"index;not null" json:"credit_note_id"`
	OrderLineID      uint    `gorm:"index;not null" json:"order_line_id"`
	Name             string  `gorm:"size:120;not null" json:"name"`
	Qty              float64 `gorm:"not null" json:"qty"`
	UnitPriceHalalas int64   `gorm:"not null" json:"unit_price_halalas"`
	TaxRate          float64 `gorm:"not null" json:"tax_rate"`
}

func (CreditNoteLine) TableName() string { return "hosp_credit_note_lines" }

// DayClose is the persisted settlement record for one business date —
// the storage shape of a settlement.Record.
type DayClose struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	BusinessDate    string    `gorm:"size:10;uniqueIndex;not null" json:"business_date"`
	ExpectedHalalas int64     `json:"expected_halalas"`
	CountedHalalas  int64     `json:"counted_halalas"`
	VarianceHalalas int64     `json:"variance_halalas"`
	TendersJSON     string    `json:"tenders_json"` // per-tender breakdown, serialized
	ClosedByID      string    `gorm:"size:60;not null" json:"closed_by_id"`
	ClosedByName    string    `gorm:"size:120" json:"closed_by_name"`
	Note            string    `gorm:"size:400" json:"note"`
	ClosedAt        time.Time `json:"closed_at"`
}

func (DayClose) TableName() string { return "hosp_day_closes" }

// Setting is a key/value row for deployment state the engines treat as
// caller-persisted values: the manager PIN hash and its lockout state.
type Setting struct {
	Key   string `gorm:"primaryKey;size:60" json:"key"`
	Value string `gorm:"size:2000" json:"value"`
}

func (Setting) TableName() string { return "hosp_settings" }
