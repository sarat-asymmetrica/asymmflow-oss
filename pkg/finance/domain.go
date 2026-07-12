// Package finance contains the finance domain model.
package finance

import (
	"context"
	"time"

	shareddomain "ph_holdings_app/pkg/domain"
	"ph_holdings_app/pkg/infra/events"
	"ph_holdings_app/pkg/overlay"

	"gorm.io/gorm"
)

type Base = shareddomain.Base

type ARAgingBucket struct {
	Base
	CustomerID   string    `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	CustomerName string    `gorm:"index;size:255" json:"customer_name"`
	SnapshotDate time.Time `gorm:"index" json:"snapshot_date"` // Date of aging calculation

	// AR Aging Buckets (in BHD)
	Less15Days float64 `json:"less_15_days"` // 0-15 days (current)
	Days16_30  float64 `json:"days_16_30"`   // 16-30 days
	Days31_60  float64 `json:"days_31_60"`   // 31-60 days
	Days61_90  float64 `json:"days_61_90"`   // 61-90 days
	Over90Days float64 `json:"over_90_days"` // 90+ days (high risk)

	// Totals
	TotalOutstanding float64 `json:"total_outstanding"`
	TotalOverdue     float64 `json:"total_overdue"` // Sum of all overdue buckets

	// Risk Classification
	RiskTier    string  `gorm:"index;size:20;check:risk_tier IN ('Low','Medium','High','Critical')" json:"risk_tier"` // Low/Medium/High/Critical
	RiskScore   float64 `gorm:"check:risk_score >= 0 AND risk_score <= 1" json:"risk_score"`                          // 0.0-1.0 calculated risk
	OverdueDays int     `gorm:"check:overdue_days >= 0" json:"overdue_days"`                                          // Days for oldest overdue invoice
}

type Invoice struct {
	Base
	InvoiceNumber string `gorm:"uniqueIndex;size:50" json:"invoice_number"`
	// InvoiceDate indexed for revenue reports: WHERE invoice_date BETWEEN ? AND ?
	// P1 Fix: Part of composite index idx_invoice_customer_date for customer invoice history
	InvoiceDate time.Time `gorm:"index:idx_invoice_customer_date,priority:2;autoCreateTime:false" json:"invoice_date"`

	// P1 Fix: Added composite index for customer invoice queries: WHERE customer_id = ? ORDER BY invoice_date
	// P2 Fix: Added covering index for AR aging queries: WHERE customer_id = ? AND status = ? ORDER BY due_date
	CustomerID   string `gorm:"index:idx_invoice_customer_date,priority:1;index:idx_invoice_ar_aging,priority:1;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	CustomerName string `gorm:"index;size:255" json:"customer_name"`

	// OrderID indexed for order-invoice relationship lookups
	// NOTE: NOT unique - multiple invoices can exist per order (partial, corrections, etc.)
	OrderID          string `gorm:"index;size:36" json:"order_id,omitempty"`
	CustomerPONumber string `gorm:"index;size:100" json:"customer_po_number"`

	GrandTotalBHD float64 `gorm:"type:decimal(15,3);not null;default:0;check:grand_total_bhd >= 0" json:"grand_total_bhd"` // P1 Fix: BHD precision
	// Status indexed for invoice filtering: WHERE status = 'Paid', status IN ('Sent', 'Overdue')
	// Used by: Dashboard KPIs, AR tracking, collection workflows
	// P2 Fix: Part of covering index idx_invoice_ar_aging
	Status         string  `gorm:"index:idx_invoice_ar_aging,priority:2;size:50;default:'Sent';check:status IN ('Draft','Sent','Paid','PartiallyPaid','Overdue','Cancelled','Void','Proforma')" json:"status"` // P1 Fix: added PartiallyPaid, Proforma
	OutstandingBHD float64 `gorm:"type:decimal(15,3);not null;default:0;check:outstanding_bhd >= 0" json:"outstanding_bhd"`                                                                                    // P1 Fix: BHD precision
	SubtotalBHD    float64 `gorm:"type:decimal(15,3);not null;default:0;check:subtotal_bhd >= 0" json:"subtotal_bhd"`                                                                                          // P1 Fix: BHD precision
	// DueDate indexed for overdue tracking: WHERE due_date < NOW()
	// Used by: Collection alerts, AR aging, payment prediction
	// P2 Fix: Part of covering index idx_invoice_ar_aging
	DueDate time.Time `gorm:"index:idx_invoice_ar_aging,priority:3" json:"due_date"`

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`

	// Traceability Links (Operations Pipeline)
	RfqID              string `gorm:"index;size:36" json:"rfq_id"`
	QuoteID            string `gorm:"index;size:36" json:"quote_id"`
	OfferID            string `gorm:"index;size:36" json:"offer_id"`
	OfferNumber        string `gorm:"size:50" json:"offer_number"`
	DeliveryNoteID     string `gorm:"index;size:36" json:"delivery_note_id"` // Links invoice to delivery note
	DeliveryNoteNumber string `gorm:"size:50" json:"delivery_note_number"`   // DN number for display

	// Margin & Cost Analysis
	TotalSupplierCostBHD float64 `gorm:"not null;default:0;check:total_supplier_cost_bhd >= 0" json:"total_supplier_cost_bhd"`
	GrossMarginBHD       float64 `gorm:"not null;default:0" json:"gross_margin_bhd"`
	GrossMarginPercent   float64 `gorm:"not null;default:0" json:"gross_margin_percent"`

	// Contact & RFQ Details (from Order/Offer - costing sheet data for invoice)
	CustomerReference string  `gorm:"size:255" json:"customer_reference"` // RFQ/enquiry reference
	AttentionPerson   string  `gorm:"size:255" json:"attention_person"`   // Contact name
	AttentionCompany  string  `gorm:"size:255" json:"attention_company"`
	AttentionPhone    string  `gorm:"size:50" json:"attention_phone"`
	AttentionAddress  string  `gorm:"type:varchar(1000)" json:"attention_address"`
	DeliveryWeeks     string  `gorm:"size:50" json:"delivery_weeks"`
	CountryOfOrigin   string  `gorm:"size:100" json:"country_of_origin"`
	IssuedBy          string  `gorm:"size:100" json:"issued_by"`
	ContactPhone      string  `gorm:"size:50" json:"contact_phone"`
	DiscountPercent   float64 `json:"discount_percent"`
	PaymentTerms      string  `gorm:"type:varchar(500)" json:"payment_terms"`
	DeliveryTerms     string  `gorm:"type:varchar(500)" json:"delivery_terms"`

	// E2: Division field - 'Acme Instrumentation' or 'Beacon Controls' (sister companies)
	Division string `gorm:"size:100" json:"division"`

	// Field Visibility Settings (JSON - user can toggle which fields appear on invoice PDF)
	// Example: {"show_fob": false, "show_freight": false, "show_margin": false, "show_cost": false, "show_contact": true}
	FieldVisibility string `gorm:"type:varchar(1000)" json:"field_visibility"`

	// Extended Invoice Header Fields (Issue #19 - matches client's Tally format)
	// These fields enable the full 11-field invoice header as per client reference PDF
	DeliveryNoteRef    string     `gorm:"size:100" json:"delivery_note_ref"`                                // e.g., "EH/253/25"
	ModeOfPayment      string     `gorm:"size:100" json:"mode_of_payment"`                                  // e.g., "30 Days"
	SuppliersRef       string     `gorm:"size:100" json:"suppliers_ref"`                                    // Supplier's reference number
	OtherReferences    string     `gorm:"size:200" json:"other_references"`                                 // e.g., "Rhine Instruments"
	BuyersOrderNumber  string     `gorm:"size:100" json:"buyers_order_number"`                              // e.g., "LPS -11347"
	BuyersOrderDate    *time.Time `json:"buyers_order_date"`                                                // Buyer's PO date
	DespatchDocumentNo string     `gorm:"size:100" json:"despatch_document_no"`                             // Despatch/delivery document number
	DeliveryNoteDate   *time.Time `json:"delivery_note_date"`                                               // Delivery note date
	DespatchedThrough  string     `gorm:"size:100;default:'Direct'" json:"despatched_through"`              // Mode of despatch
	Destination        string     `gorm:"size:100;default:'Bahrain'" json:"destination"`                    // Delivery destination
	PlaceOfSupply      string     `gorm:"size:100;default:'Kingdom of Bahrain'" json:"place_of_supply"`     // VAT place of supply
	TermsOfDelivery    string     `gorm:"size:200;default:'Direct Bank Transfer'" json:"terms_of_delivery"` // Payment method

	// VAT & Accounting (Tally Killer feature)
	VATBHD         float64 `gorm:"column:vatbhd;type:decimal(15,3);not null;default:0;check:vatbhd >= 0" json:"vat_bhd"` // P1 Fix: BHD precision
	VATPercent     float64 `gorm:"not null;default:0;check:vat_percent >= 0" json:"vat_percent"`                         // VAT rate applied (default 10)
	JournalEntryID string  `gorm:"index;size:36" json:"journal_entry_id"`                                                // Link to auto-generated GL entry

	// E-Invoicing (Phase 23)
	InvoiceHash string `gorm:"size:64" json:"invoice_hash"` // SHA-256 hash for integrity verification

	// PH free-text notes column retained verbatim (PC-D22, Mission I D-I-5).
	Notes string `gorm:"type:text" json:"notes"`

	Items []DBInvoiceItem `gorm:"foreignKey:InvoiceID" json:"items"`
}

func (Invoice) TableName() string { return "invoices" }

// AfterCreate publishes a finance.invoice.created domain event on the process
// default bus once an invoice row is inserted (covering every create path
// uniformly). It is best-effort: events.PublishDefault is nil-safe and never
// returns an error, so the insert always commits even with no bus or a failing
// subscriber. The seller tax ID and currency are resolved from the active
// company overlay by division, so the event carries the correct per-division
// TRN and the jurisdiction can be inferred downstream.
func (i *Invoice) AfterCreate(tx *gorm.DB) error {
	ov := overlay.Active()
	seller := ov.Profile(ov.NormalizeDivisionName(i.Division)).VATNumber
	events.PublishDefault(context.Background(), events.InvoiceCreated{
		BaseEvent:     events.BaseEvent{Timestamp: time.Now(), CorrelationID: "inv:" + i.ID},
		InvoiceID:     i.ID,
		InvoiceNumber: i.InvoiceNumber,
		InvoiceDate:   i.InvoiceDate,
		SellerTaxID:   seller,
		Amount:        i.SubtotalBHD,
		TaxAmount:     i.VATBHD,
		Currency:      ov.Currency,
		Jurisdiction:  ov.JurisdictionCode(),
	})
	return nil
}

type DBInvoiceItem struct {
	Base
	// NOTE: Removed constraint tag - SQLite can't modify FK constraints on existing tables
	InvoiceID   string  `gorm:"index;size:36" json:"invoice_id"`
	LineNumber  int     `json:"line_number"`
	Description string  `gorm:"type:varchar(2000)" json:"description"`
	Quantity    float64 `gorm:"not null;default:0;check:quantity >= 0" json:"quantity"`
	Rate        float64 `gorm:"not null;default:0;check:rate >= 0" json:"rate"`
	TotalBHD    float64 `gorm:"not null;default:0;check:total_bhd >= 0" json:"total_bhd"`

	// Full costing data (from OrderItem/OfferItem - the costing sheet becomes the invoice)
	ProductID           string  `gorm:"index;size:36" json:"product_id"`
	ProductCode         string  `gorm:"size:100" json:"product_code"`                                 // Model number
	Equipment           string  `gorm:"size:255" json:"equipment"`                                    // Equipment/Product name
	Model               string  `gorm:"size:255" json:"model"`                                        // Model number
	Specification       string  `gorm:"type:varchar(2000)" json:"specification"`                      // Short specification
	DetailedDescription string  `gorm:"type:varchar(5000)" json:"detailed_description"`               // Full instrumentation specs
	Currency            string  `gorm:"size:10" json:"currency"`                                      // Source currency (USD, EUR, etc.)
	FOB                 float64 `gorm:"not null;default:0;check:fob >= 0" json:"fob"`                 // FOB cost
	Freight             float64 `gorm:"not null;default:0;check:freight >= 0" json:"freight"`         // Freight cost
	TotalCost           float64 `gorm:"not null;default:0;check:total_cost >= 0" json:"total_cost"`   // Total landed cost
	MarginPercent       float64 `gorm:"not null;default:0" json:"margin_percent"`                     // Margin % applied
	TotalPrice          float64 `gorm:"not null;default:0;check:total_price >= 0" json:"total_price"` // Line total (qty × unit price)
}

func (DBInvoiceItem) TableName() string { return "invoice_items" }

type InvoiceSequence struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Prefix     string    `gorm:"size:10;uniqueIndex:idx_invoice_sequence_prefix_year" json:"prefix"`
	Year       int       `gorm:"uniqueIndex:idx_invoice_sequence_prefix_year" json:"year"`
	LastNumber int       `gorm:"not null;default:0" json:"last_number"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (InvoiceSequence) TableName() string { return "invoice_sequences" }

type CreditNote struct {
	Base
	CreditNoteNumber string           `gorm:"uniqueIndex;size:50" json:"credit_note_number"`
	CreditNoteDate   time.Time        `gorm:"index" json:"credit_note_date"`
	InvoiceID        string           `gorm:"index;size:36" json:"invoice_id"`
	InvoiceNumber    string           `gorm:"size:50" json:"invoice_number"`
	CustomerID       string           `gorm:"index;size:36" json:"customer_id"`
	CustomerName     string           `gorm:"size:255" json:"customer_name"`
	Reason           string           `gorm:"type:varchar(2000)" json:"reason"`
	SubtotalBHD      float64          `gorm:"type:decimal(15,3)" json:"subtotal_bhd"`
	VATBHD           float64          `gorm:"type:decimal(15,3)" json:"vat_bhd"`
	VATPercent       float64          `json:"vat_percent"`
	GrandTotalBHD    float64          `gorm:"type:decimal(15,3)" json:"grand_total_bhd"`
	Status           string           `gorm:"index;size:50;default:'Draft'" json:"status"` // Draft, Issued, Applied
	Division         string           `gorm:"size:100" json:"division"`
	AppliedAt        *time.Time       `json:"applied_at"`
	CreditNoteHash   string           `gorm:"size:64" json:"credit_note_hash"`
	Items            []CreditNoteItem `gorm:"foreignKey:CreditNoteID" json:"items"`
}

func (CreditNote) TableName() string { return "credit_notes" }

type CreditNoteItem struct {
	Base
	CreditNoteID string  `gorm:"index;size:36" json:"credit_note_id"`
	LineNumber   int     `json:"line_number"`
	Description  string  `gorm:"type:varchar(2000)" json:"description"`
	Quantity     float64 `json:"quantity"`
	Rate         float64 `json:"rate"`
	TotalBHD     float64 `gorm:"type:decimal(15,3)" json:"total_bhd"`
}

func (CreditNoteItem) TableName() string { return "credit_note_items" }

type ChartOfAccount struct {
	Base
	AccountCode string  `gorm:"uniqueIndex;size:20" json:"account_code"`
	AccountName string  `json:"account_name"`
	AccountType string  `gorm:"index;size:50" json:"account_type"`
	Balance     float64 `json:"balance"`
	IsActive    bool    `gorm:"default:true" json:"is_active"`

	// VAT Tracking (NEW)
	IsVATAccount bool   `gorm:"index" json:"is_vat_account"`
	VATDirection string `gorm:"size:20" json:"vat_direction"` // "input", "output", or null

	// Hierarchy (NEW)
	ParentAccountID string `gorm:"index;size:36;constraint:OnDelete:RESTRICT;" json:"parent_account_id"`
	AccountGroup    string `gorm:"index;size:10" json:"account_group"` // "BS" or "PL"
}

type JournalEntry struct {
	Base
	EntryNumber string    `gorm:"uniqueIndex;size:50" json:"entry_number"`
	EntryDate   time.Time `gorm:"index" json:"entry_date"`
	Description string    `json:"description"`

	DebitTotal   float64    `gorm:"type:decimal(15,3)" json:"debit_total"`  // P1 Fix: BHD precision
	CreditTotal  float64    `gorm:"type:decimal(15,3)" json:"credit_total"` // P1 Fix: BHD precision
	IsPosted     bool       `gorm:"index" json:"is_posted"`
	PostedAt     *time.Time `json:"posted_at"`
	PostedBy     string     `json:"posted_by"`
	FiscalYear   int        `gorm:"index" json:"fiscal_year"`
	FiscalPeriod int        `json:"fiscal_period"`

	// Auto-posting source tracking (Tally Killer feature)
	SourceType      string `gorm:"index;size:50" json:"source_type"` // invoice, payment, supplier_invoice, supplier_payment, manual
	SourceID        string `gorm:"index;size:36" json:"source_id"`   // ID of source document
	IsAutoGenerated bool   `gorm:"index;default:false" json:"is_auto_generated"`
	ReversedByID    string `gorm:"index;size:36" json:"reversed_by_id"` // If this entry was reversed, link to reversal
	ReversesID      string `gorm:"index;size:36" json:"reverses_id"`    // If this is a reversal, link to original

	// P1 Fix: Audit field (DeletedAt already in Base for soft delete)
	UpdatedBy string `json:"updated_by"`

	Lines []JournalLine `gorm:"foreignKey:EntryID" json:"lines,omitempty"`
}

type JournalLine struct {
	Base
	// P0-4: Changed CASCADE to RESTRICT to preserve audit trail
	EntryID     string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT;" json:"entry_id"`
	AccountID   string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT;" json:"account_id"`
	AccountName string  `json:"account_name"`
	Debit       float64 `gorm:"type:decimal(15,3);not null;default:0;check:debit >= 0" json:"debit"`   // P1 Fix: BHD precision
	Credit      float64 `gorm:"type:decimal(15,3);not null;default:0;check:credit >= 0" json:"credit"` // P1 Fix: BHD precision
	Description string  `json:"description"`

	// P1 Fix: Audit field (DeletedAt already in Base for soft delete)
	UpdatedBy string `json:"updated_by"`
}

type VATReturn struct {
	Base
	ReturnNumber string    `gorm:"uniqueIndex;size:50" json:"return_number"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	FiscalYear   int       `json:"fiscal_year"`
	Quarter      int       `json:"quarter"`
	NetVAT       float64   `json:"net_vat"`
	Status       string    `json:"status"`
}

type Payment struct {
	Base
	// P2 Fix: Added covering index for payment history queries: WHERE invoice_id = ? ORDER BY payment_date
	// P2.4 Fix: Changed CASCADE to RESTRICT - payments are financial records, must never auto-delete
	InvoiceID     string  `gorm:"index:idx_payment_invoice_date,priority:1;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:InvoiceID;references:ID" json:"invoice_id"`
	InvoiceNumber string  `gorm:"index;size:50" json:"invoice_number"`
	AmountBHD     float64 `gorm:"type:decimal(15,3);not null;default:0;check:amount_bhd >= 0" json:"amount_bhd"` // P1 Fix: BHD precision
	// P2 Fix: Part of covering index idx_payment_invoice_date
	PaymentDate   time.Time `gorm:"index:idx_payment_invoice_date,priority:2;autoCreateTime:false" json:"payment_date"`
	PaymentMethod string    `gorm:"size:50;check:payment_method IN ('Cash','Cheque','Bank Transfer','Credit Card','LC','PDC','Other')" json:"payment_method"`
	// DaysToPayment indexed for payment analytics: AVG(days_to_payment), payment speed reports
	// Used by: Payment prediction accuracy, customer payment behavior analysis, survival curves
	DaysToPayment int `gorm:"index;check:days_to_payment >= 0" json:"days_to_payment"`

	// BE-8 FIX: Idempotency key to prevent duplicate payments from network retries
	// Format: sha256(invoice_id + amount + date + ref) - unique per payment attempt
	IdempotencyKey string `gorm:"uniqueIndex;size:64" json:"idempotency_key"`

	// Accounting Integration (Tally Killer feature)
	JournalEntryID string  `gorm:"index;size:36" json:"journal_entry_id"`     // Link to auto-generated GL entry
	BankAccountID  string  `gorm:"index;size:36" json:"bank_account_id"`      // Bank account for reconciliation
	ReceiptID      *string `gorm:"index;size:36" json:"receipt_id,omitempty"` // Customer receipt header (incl. on-account receipts)
	Reference      string  `gorm:"size:100" json:"reference"`                 // Payment reference (cheque no, transfer ref)
	Division       string  `gorm:"size:100" json:"division"`

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`
}

func (Payment) TableName() string { return "payments" }

type ExpenseCategory struct {
	Base
	Name           string  `gorm:"uniqueIndex;size:120" json:"name"`
	Code           string  `gorm:"uniqueIndex;size:40" json:"code"`
	Description    string  `gorm:"size:500" json:"description"`
	GLAccountID    *string `gorm:"size:36" json:"gl_account_id"`
	DefaultTaxRate float64 `gorm:"default:0" json:"default_tax_rate"`
	IsActive       bool    `gorm:"default:true;index" json:"is_active"`
	SortOrder      int     `gorm:"default:0" json:"sort_order"`
	GLAccountName  string  `gorm:"-" json:"gl_account_name,omitempty"`
}

func (ExpenseCategory) TableName() string { return "expense_categories" }

type ExpenseVendor struct {
	Base
	Name         string `gorm:"uniqueIndex;size:255" json:"name"`
	ContactName  string `gorm:"size:255" json:"contact_name"`
	Email        string `gorm:"size:255" json:"email"`
	Phone        string `gorm:"size:50" json:"phone"`
	PaymentTerms string `gorm:"size:100" json:"payment_terms"`
	TaxNumber    string `gorm:"size:100" json:"tax_number"`
	Notes        string `gorm:"type:text" json:"notes"`
	IsActive     bool   `gorm:"default:true;index" json:"is_active"`
}

func (ExpenseVendor) TableName() string { return "expense_vendors" }

type ExpenseEntry struct {
	Base
	EntryNumber        string     `gorm:"uniqueIndex;size:50" json:"entry_number"`
	Division           string     `gorm:"size:100" json:"division"`
	ExpenseDate        time.Time  `gorm:"index" json:"expense_date"`
	DueDate            *time.Time `gorm:"index" json:"due_date"`
	Description        string     `gorm:"type:text" json:"description"`
	CategoryID         string     `gorm:"index;size:36" json:"category_id"`
	VendorID           *string    `gorm:"index;size:36" json:"vendor_id"`
	SourceType         string     `gorm:"index;size:40;default:'manual'" json:"source_type"`
	SourceRefID        *string    `gorm:"index;size:36" json:"source_ref_id"`
	BankExpenseEntryID *string    `gorm:"index;size:36" json:"bank_expense_entry_id"`
	ProjectID          *string    `gorm:"index;size:36" json:"project_id"`
	CustomerID         *string    `gorm:"index;size:36" json:"customer_id"`
	OpportunityID      *string    `gorm:"index;size:36" json:"opportunity_id"`
	OrderID            *string    `gorm:"index;size:36" json:"order_id"`
	CostCenter         string     `gorm:"size:100" json:"cost_center"`
	Currency           string     `gorm:"size:3;default:'BHD'" json:"currency"`
	Amount             float64    `gorm:"type:decimal(15,3)" json:"amount"`
	VATAmount          float64    `gorm:"type:decimal(15,3)" json:"vat_amount"`
	TotalAmount        float64    `gorm:"type:decimal(15,3)" json:"total_amount"`
	Status             string     `gorm:"index;size:20;default:'draft'" json:"status"`
	PaymentStatus      string     `gorm:"index;size:20;default:'unpaid'" json:"payment_status"`
	SubmittedAt        *time.Time `json:"submitted_at"`
	SubmittedBy        string     `gorm:"size:36" json:"submitted_by"`
	ApprovedAt         *time.Time `json:"approved_at"`
	ApprovedBy         string     `gorm:"size:36" json:"approved_by"`
	RejectedAt         *time.Time `json:"rejected_at"`
	RejectedBy         string     `gorm:"size:36" json:"rejected_by"`
	RejectionReason    string     `gorm:"type:text" json:"rejection_reason"`
	PostedAt           *time.Time `json:"posted_at"`
	PostedBy           string     `gorm:"size:36" json:"posted_by"`
	PaidAt             *time.Time `json:"paid_at"`
	PaymentMethod      string     `gorm:"size:50" json:"payment_method"`
	PaymentReference   string     `gorm:"size:120" json:"payment_reference"`
	BankAccountID      *string    `gorm:"size:36" json:"bank_account_id"`
	JournalEntryID     *string    `gorm:"size:36" json:"journal_entry_id"`
	Notes              string     `gorm:"type:text" json:"notes"`

	CategoryName string `gorm:"-" json:"category_name,omitempty"`
	VendorName   string `gorm:"-" json:"vendor_name,omitempty"`
}

func (ExpenseEntry) TableName() string { return "expense_entries" }

type RecurringExpense struct {
	Base
	Name             string     `gorm:"index;size:255" json:"name"`
	Division         string     `gorm:"size:100" json:"division"`
	Description      string     `gorm:"type:text" json:"description"`
	CategoryID       string     `gorm:"index;size:36" json:"category_id"`
	VendorID         *string    `gorm:"size:36" json:"vendor_id"`
	Frequency        string     `gorm:"size:20;default:'monthly'" json:"frequency"`
	IntervalValue    int        `gorm:"default:1" json:"interval_value"`
	NextRunDate      time.Time  `gorm:"index" json:"next_run_date"`
	LastGeneratedAt  *time.Time `json:"last_generated_at"`
	DefaultAmount    float64    `gorm:"type:decimal(15,3)" json:"default_amount"`
	DefaultVATAmount float64    `gorm:"type:decimal(15,3)" json:"default_vat_amount"`
	Currency         string     `gorm:"size:3;default:'BHD'" json:"currency"`
	CostCenter       string     `gorm:"size:100" json:"cost_center"`
	ProjectID        *string    `gorm:"size:36" json:"project_id"`
	IsActive         bool       `gorm:"default:true;index" json:"is_active"`
	AutoSubmit       bool       `gorm:"default:false" json:"auto_submit"`

	CategoryName string `gorm:"-" json:"category_name,omitempty"`
	VendorName   string `gorm:"-" json:"vendor_name,omitempty"`
}

func (RecurringExpense) TableName() string { return "recurring_expenses" }

type ExpenseDashboardSummary struct {
	TotalDrafts         int     `json:"total_drafts"`
	TotalSubmitted      int     `json:"total_submitted"`
	TotalApprovedUnpaid int     `json:"total_approved_unpaid"`
	TotalRecurring      int     `json:"total_recurring"`
	MonthToDateSpend    float64 `json:"month_to_date_spend"`
	UpcomingCommitments float64 `json:"upcoming_commitments"`
}

type CurrencyExchangeRate struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	CurrencyCode  string     `gorm:"type:varchar(3);not null;index" json:"currency_code"` // USD, EUR, CHF, SAR, AED
	Rate          float64    `gorm:"type:real;not null" json:"rate"`                      // e.g., 0.376 for 1 USD = 0.376 BHD
	EffectiveFrom time.Time  `gorm:"not null;index" json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to"` // NULL = currently active
	SetBy         string     `gorm:"size:100" json:"set_by"`
	Notes         string     `gorm:"size:500" json:"notes"`
}

func (CurrencyExchangeRate) TableName() string { return "currency_exchange_rates" }

type PurchaseOrder struct {
	Base
	// Links (One Order → Many POs)
	OrderID string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:OrderID;references:ID" json:"order_id"`
	RfqID   string `gorm:"index;size:36" json:"rfq_id"`
	// P1 Fix: Composite index for supplier PO queries: WHERE supplier_id = ? ORDER BY po_date
	SupplierID   string `gorm:"index:idx_po_supplier_date,priority:1;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:SupplierID;references:ID" json:"supplier_id"`
	SupplierName string `gorm:"index;size:255" json:"supplier_name"` // Denormalized for display

	// Document Info
	PONumber string `gorm:"uniqueIndex;size:50" json:"po_number"`
	// P1 Fix: Part of composite index idx_po_supplier_date
	PODate           time.Time `gorm:"index:idx_po_supplier_date,priority:2;autoCreateTime:false" json:"po_date"`
	ExpectedDelivery time.Time `json:"expected_delivery"`

	// Multi-Currency Support
	Currency        string  `gorm:"size:3" json:"currency"`
	ExchangeRate    float64 `gorm:"not null;default:1;check:exchange_rate > 0" json:"exchange_rate"`
	SubtotalForeign float64 `gorm:"not null;default:0;check:subtotal_foreign >= 0" json:"subtotal_foreign"`
	SubtotalBHD     float64 `gorm:"type:decimal(15,3);not null;default:0;check:subtotal_bhd >= 0" json:"subtotal_bhd"` // P1 Fix: BHD precision
	VATAmount       float64 `gorm:"type:decimal(15,3);not null;default:0;check:vat_amount >= 0" json:"vat_amount"`     // P1 Fix: BHD precision
	TotalForeign    float64 `gorm:"not null;default:0;check:total_foreign >= 0" json:"total_foreign"`
	TotalBHD        float64 `gorm:"type:decimal(15,3);not null;default:0;check:total_bhd >= 0" json:"total_bhd"` // P1 Fix: BHD precision

	// Payment
	PaymentTerms   string    `gorm:"type:varchar(500)" json:"payment_terms"`
	PaymentDueDate time.Time `json:"payment_due_date"`

	// Status Flow: Draft → Pending Approval → Approved → Sent → Acknowledged → Partially Received → Received → Closed
	Status string `gorm:"index;size:50;default:'Draft';check:status IN ('Draft','Pending Approval','Approved','Sent','Acknowledged','Partially Received','Received','Cancelled','Closed')" json:"status"` // P1 Fix: default status

	// P1 Fix: Approval workflow fields
	ApprovedBy string     `json:"approved_by"` // User who approved the PO
	ApprovedAt *time.Time `json:"approved_at"` // When PO was approved

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`

	Division string `gorm:"size:100" json:"division"`

	// Relationships
	Items []PurchaseOrderItem `gorm:"foreignKey:PurchaseOrderID" json:"items,omitempty"`
}

type PurchaseOrderItem struct {
	Base
	PurchaseOrderID    string  `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:PurchaseOrderID;references:ID" json:"purchase_order_id"`
	OrderItemID        string  `gorm:"index;size:36" json:"order_item_id"`
	ProductID          string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:ProductID;references:ID" json:"product_id"`
	ProductCode        string  `gorm:"size:100" json:"product_code"`
	SupplierPartNumber string  `gorm:"size:100" json:"supplier_part_number"` // P2 Fix: Supplier's internal part number
	Description        string  `gorm:"type:varchar(2000)" json:"description"`
	Quantity           float64 `json:"quantity"`
	UnitPriceForeign   float64 `json:"unit_price_foreign"`
	UnitPriceBHD       float64 `json:"unit_price_bhd"`
	TotalForeign       float64 `json:"total_foreign"`
	TotalBHD           float64 `json:"total_bhd"`
	QuantityReceived   float64 `json:"quantity_received"`

	// Wave 9.8 B1: query-time overlay — NOT persisted (gorm:"-"). Mirrors the
	// same field on crm.PurchaseOrderItem so both duplicate definitions stay
	// in sync; not populated by any finance-package query today.
	RequiresSerialTracking bool `gorm:"-" json:"requires_serial_tracking"`
}

type SupplierInvoice struct {
	Base
	// Links - P1 Fix: Composite index for supplier invoice queries: WHERE supplier_id = ? ORDER BY invoice_date
	// P2 Fix: Added covering index for supplier ledger: WHERE supplier_id = ? AND status = ?
	SupplierID      string `gorm:"index:idx_supplier_inv_date,priority:1;index:idx_supplier_inv_status,priority:1;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:SupplierID;references:ID" json:"supplier_id"`
	SupplierName    string `gorm:"index;size:255" json:"supplier_name"` // Denormalized for display
	PurchaseOrderID string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:PurchaseOrderID;references:ID" json:"purchase_order_id"`
	PONumber        string `gorm:"size:50" json:"po_number"` // Denormalized for display
	GRNID           string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:GRNID;references:ID" json:"grn_id"`
	OrderID         string `gorm:"index;size:36" json:"order_id"` // Internal customer order reference

	// Document Info (can be OCR'd)
	InvoiceNumber string `gorm:"index;size:100" json:"invoice_number"` // Supplier's number
	// P1 Fix: Part of composite index idx_supplier_inv_date
	InvoiceDate time.Time `gorm:"index:idx_supplier_inv_date,priority:2;autoCreateTime:false" json:"invoice_date"`
	DueDate     time.Time `gorm:"index" json:"due_date"`

	// Multi-Currency
	Currency        string  `gorm:"size:3" json:"currency"`
	ExchangeRate    float64 `gorm:"not null;default:1;check:exchange_rate > 0" json:"exchange_rate"`
	SubtotalForeign float64 `gorm:"not null;default:0;check:subtotal_foreign >= 0" json:"subtotal_foreign"`
	SubtotalBHD     float64 `gorm:"type:decimal(15,3);not null;default:0;check:subtotal_bhd >= 0" json:"subtotal_bhd"` // P1 Fix: BHD precision
	VATForeign      float64 `gorm:"not null;default:0;check:vat_foreign >= 0" json:"vat_foreign"`
	VATBHD          float64 `gorm:"column:vatbhd;type:decimal(15,3);not null;default:0;check:vatbhd >= 0" json:"vat_bhd"` // P1 Fix: BHD precision
	TotalForeign    float64 `gorm:"not null;default:0;check:total_foreign >= 0" json:"total_foreign"`
	TotalBHD        float64 `gorm:"type:decimal(15,3);not null;default:0;check:total_bhd >= 0" json:"total_bhd"` // P1 Fix: BHD precision

	// 3-Way Match Status
	// Mission G (Wave 4): CHECK widened to include 'Review Required' — the within-
	// tolerance price-variance state PerformThreeWayMatch actually writes. The old
	// constraint rejected it on a fresh AutoMigrated DB, so a variance-flagged
	// match could not persist. ('Dispute' retained for imported-data compatibility.)
	MatchStatus string `gorm:"index;default:'Pending';check:match_status IN ('Pending','Matched','Discrepancy','Review Required','Dispute')" json:"match_status"` // P1 Fix: indexed + default
	POMatchOK   bool   `json:"po_match_ok"`                                                                                                                       // Matches PO amounts?
	GRNMatchOK  bool   `json:"grn_match_ok"`                                                                                                                      // Matches GRN quantities?

	// Approval Workflow
	// P2 Fix: Part of covering index idx_supplier_inv_status
	// Mission G (Wave 4): CHECK widened to include 'Verified' (clean-match state,
	// pay-eligible) and 'Disputed' — both are values the supplier-invoice service
	// actually writes, but the old constraint rejected them on a fresh AutoMigrated
	// DB, making a clean 3-way match unpersistable. ('Dispute' retained so rows
	// imported from PH's vocabulary still satisfy the constraint.)
	Status     string     `gorm:"index:idx_supplier_inv_status,priority:2;size:50;default:'Pending';check:status IN ('Pending','Approved','Rejected','Paid','Verified','Disputed','Dispute')" json:"status"` // P1 Fix: default status
	ApprovedBy string     `json:"approved_by"`
	ApprovedAt *time.Time `json:"approved_at"`

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`

	// Payment
	PaymentStatus string     `json:"payment_status"` // Unpaid, Scheduled, Paid
	PaymentDate   *time.Time `json:"payment_date"`
	PaymentRef    string     `json:"payment_ref"`
	PaymentMethod string     `json:"payment_method"` // Bank Transfer, Cheque, etc.
	// Wave 8 P1: transient (gorm:"-", no column) outstanding balance, hydrated
	// from the SupplierPayment ledger by the payment-state policy — PH parity.
	OutstandingBHD float64 `gorm:"-" json:"outstanding_bhd"`

	// OCR Metadata
	OCRDocumentID string  `json:"ocr_document_id"` // Link to OCR scan
	OCRConfidence float64 `json:"ocr_confidence"`
	Division      string  `gorm:"size:100" json:"division"`

	// Discrepancy tracking
	DiscrepancyReason string `gorm:"type:varchar(2000)" json:"discrepancy_reason"`
	DisputeReason     string `gorm:"type:varchar(2000)" json:"dispute_reason"`

	// Accounting Integration (Tally Killer feature)
	JournalEntryID string `gorm:"index;size:36" json:"journal_entry_id"` // Link to auto-generated GL entry

	// Line Items
	Items []SupplierInvoiceItem `gorm:"foreignKey:SupplierInvoiceID" json:"items"`
}

func (SupplierInvoice) TableName() string { return "supplier_invoices" }

type SupplierInvoiceItem struct {
	Base
	SupplierInvoiceID string  `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:SupplierInvoiceID;references:ID" json:"supplier_invoice_id"`
	LineNumber        int     `json:"line_number"`
	Description       string  `gorm:"type:varchar(2000)" json:"description"`
	Quantity          float64 `json:"quantity"`
	UnitPrice         float64 `json:"unit_price"`
	TotalPrice        float64 `json:"total_price"`
	Currency          string  `gorm:"size:3" json:"currency"`
}

func (SupplierInvoiceItem) TableName() string { return "supplier_invoice_items" }

type SupplierPayment struct {
	Base
	// P2.4 Fix: Changed CASCADE to RESTRICT - payment records must never auto-delete
	SupplierInvoiceID string    `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:SupplierInvoiceID;references:ID" json:"supplier_invoice_id"`
	SupplierID        string    `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:SupplierID;references:ID" json:"supplier_id"`
	AmountForeign     float64   `gorm:"not null;default:0;check:amount_foreign >= 0" json:"amount_foreign"`
	Currency          string    `gorm:"size:3" json:"currency"`
	ExchangeRate      float64   `gorm:"not null;default:1;check:exchange_rate > 0" json:"exchange_rate"`
	AmountBHD         float64   `gorm:"type:decimal(15,3);not null;default:0;check:amount_bhd >= 0" json:"amount_bhd"` // P1 Fix: BHD precision
	PaymentDate       time.Time `gorm:"index;autoCreateTime:false" json:"payment_date"`
	PaymentMethod     string    `gorm:"size:50;check:payment_method IN ('Bank Transfer','Cheque','LC','Cash','Wire Transfer','PDC','Other')" json:"payment_method"` // Bank Transfer, Cheque, LC
	Reference         string    `gorm:"size:100" json:"reference"`                                                                                                  // Cheque/transfer reference
	Notes             string    `gorm:"type:varchar(2000)" json:"notes"`
	// PH human-facing payment number retained verbatim (PC-D22, Mission I D-I-5).
	PaymentNumber string `gorm:"size:50" json:"payment_number"`

	// Accounting Integration (Tally Killer feature)
	JournalEntryID string `gorm:"index;size:36" json:"journal_entry_id"` // Link to auto-generated GL entry
	BankAccountID  string `gorm:"index;size:36" json:"bank_account_id"`  // Bank account for reconciliation

	// P1 Fix: Audit field (DeletedAt already in Base for soft delete)
	UpdatedBy string `json:"updated_by"`
	Division  string `gorm:"size:100" json:"division"`

	// Display fields (populated from linked invoice for UI display)
	SupplierName  string `gorm:"column:supplier_name;size:255" json:"supplier_name"`
	InvoiceNumber string `gorm:"column:invoice_number;size:100" json:"invoice_number"`
}

func (SupplierPayment) TableName() string { return "supplier_payments" }

type AccountMapping struct {
	Base
	TransactionType string `gorm:"uniqueIndex;size:50" json:"transaction_type"` // AR, AP, Revenue, COGS, Cash, Bank, VAT_Output, VAT_Input
	AccountID       string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:AccountID;references:ID" json:"account_id"`
	AccountCode     string `gorm:"size:20" json:"account_code"`
	AccountName     string `gorm:"size:255" json:"account_name"`
	Description     string `gorm:"type:varchar(1000)" json:"description"`
	IsActive        bool   `gorm:"default:true" json:"is_active"`
}

type FiscalPeriod struct {
	Base
	FiscalYear  int        `gorm:"index;check:fiscal_year >= 2000 AND fiscal_year <= 2100" json:"fiscal_year"`
	Period      int        `gorm:"index;check:period >= 1 AND period <= 12" json:"period"` // 1-12 for months
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	Status      string     `gorm:"index;size:20;default:'Open';check:status IN ('Open','Closed','Locked')" json:"status"` // Open, Closed, Locked
	ClosedAt    *time.Time `json:"closed_at"`
	ClosedBy    string     `gorm:"size:255" json:"closed_by"`
}

type BankAccount struct {
	Base
	AccountID      string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:AccountID;references:ID" json:"account_id"` // Link to ChartOfAccount
	BankName       string  `gorm:"size:255" json:"bank_name"`
	AccountNumber  string  `gorm:"size:100" json:"account_number"`
	AccountName    string  `gorm:"size:255" json:"account_name"`
	Currency       string  `gorm:"size:3;default:'BHD'" json:"currency"`
	CurrentBalance float64 `gorm:"type:decimal(15,3)" json:"current_balance"`
	IsActive       bool    `gorm:"index;default:true" json:"is_active"`
}

// CompanyBankAccount stores division-aware bank account details for invoices
// and reconciliation.
type CompanyBankAccount struct {
	ID            string `gorm:"primaryKey;size:36" json:"id"`
	Division      string `gorm:"size:100" json:"division"`
	BankName      string `gorm:"not null;size:100" json:"bank_name"`
	AccountName   string `gorm:"not null;size:100" json:"account_name"`
	AccountNumber string `gorm:"not null;size:50" json:"account_number"`
	IBAN          string `gorm:"size:50" json:"iban"`
	SwiftBIC      string `gorm:"size:20" json:"swift_bic"`
	Currency      string `gorm:"size:3;default:'BHD'" json:"currency"`
	IsActive      bool   `gorm:"default:true" json:"is_active"`
	DisplayOrder  int    `json:"display_order"`
	// BookingRate is the exchange rate at which this foreign-currency account
	// was opened. Zero means first revaluation uses current rate as baseline.
	BookingRate float64   `gorm:"default:0" json:"booking_rate"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CompanyBankAccount) TableName() string { return "company_bank_accounts" }

type BankStatement struct {
	Base
	BankAccountID   string    `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:BankAccountID;references:ID" json:"bank_account_id"`
	StatementNumber string    `gorm:"uniqueIndex;size:50" json:"statement_number"`
	StatementDate   time.Time `gorm:"index;autoCreateTime:false" json:"statement_date"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	OpeningBalance  float64   `gorm:"type:decimal(15,3)" json:"opening_balance"`
	ClosingBalance  float64   `gorm:"type:decimal(15,3)" json:"closing_balance"`
	Currency        string    `gorm:"size:3;default:'BHD'" json:"currency"`

	// Totals
	TotalDebits  float64 `gorm:"type:decimal(15,3)" json:"total_debits"`
	TotalCredits float64 `gorm:"type:decimal(15,3)" json:"total_credits"`
	DebitCount   int     `json:"debit_count"`
	CreditCount  int     `json:"credit_count"`

	// Status
	Status       string     `gorm:"index;size:20;default:'Imported';check:status IN ('Imported','InProgress','Reconciled','Verified','Cancelled')" json:"status"`
	ReconciledAt *time.Time `json:"reconciled_at"`
	ReconciledBy string     `gorm:"size:255" json:"reconciled_by"`

	// Import metadata
	ImportedFrom  string  `gorm:"size:500" json:"imported_from"`
	ImportMethod  string  `gorm:"size:20" json:"import_method"` // PDF_OCR, CSV, Manual
	OCRConfidence float64 `json:"ocr_confidence"`
	Notes         string  `gorm:"type:text" json:"notes"`
	Division      string  `gorm:"size:100" json:"division"`

	// Verification
	BalanceVerified   bool    `gorm:"default:false" json:"balance_verified"`
	DiscrepancyAmount float64 `gorm:"type:decimal(15,3)" json:"discrepancy_amount"`

	Lines []BankStatementLine `gorm:"foreignKey:BankStatementID" json:"lines,omitempty"`
}

func (BankStatement) TableName() string { return "bank_statements" }

// BalanceGap represents a gap in statement balance continuity.
type BalanceGap struct {
	FromStatementID string    `json:"from_statement_id"`
	ToStatementID   string    `json:"to_statement_id"`
	FromDate        time.Time `json:"from_date"`
	ToDate          time.Time `json:"to_date"`
	ClosingBalance  float64   `json:"closing_balance"`
	OpeningBalance  float64   `json:"opening_balance"`
	GapAmount       float64   `json:"gap_amount"`
}

// BalanceContinuityReportData provides a full continuity audit for an account.
type BalanceContinuityReportData struct {
	BankAccountID     string       `json:"bank_account_id"`
	BankName          string       `json:"bank_name"`
	Gaps              []BalanceGap `json:"gaps"`
	TotalGapAmount    float64      `json:"total_gap_amount"`
	IsContinuous      bool         `json:"is_continuous"`
	StatementsCovered int          `json:"statements_covered"`
}

type BankStatementLine struct {
	Base
	BankStatementID string    `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:BankStatementID;references:ID" json:"bank_statement_id"`
	LineNumber      int       `json:"line_number"`
	TransactionDate time.Time `gorm:"index;autoCreateTime:false" json:"transaction_date"`
	ValueDate       time.Time `json:"value_date"`
	Description     string    `gorm:"type:varchar(2000)" json:"description"`
	Reference       string    `gorm:"index;size:100" json:"reference"`
	Debit           float64   `gorm:"type:decimal(15,3);check:debit >= 0" json:"debit"`
	Credit          float64   `gorm:"type:decimal(15,3);check:credit >= 0" json:"credit"`
	Balance         float64   `gorm:"type:decimal(15,3)" json:"balance"`

	// Categorization (Enhanced)
	TransactionType    string `gorm:"size:50" json:"transaction_type"`
	Category           string `gorm:"size:50" json:"category"`
	SubCategory        string `gorm:"size:50" json:"sub_category"`
	ExtractedCustomer  string `gorm:"size:100" json:"extracted_customer"`
	ExtractedSupplier  string `gorm:"size:100" json:"extracted_supplier"`
	ExtractedInvoices  string `gorm:"type:text" json:"extracted_invoices"`   // JSON array
	ExtractedPONumbers string `gorm:"type:text" json:"extracted_po_numbers"` // JSON array

	// Matching
	IsMatched         bool    `gorm:"index;default:false" json:"is_matched"`
	MatchedPaymentID  string  `gorm:"index;size:36" json:"matched_payment_id"`
	MatchedJournalID  string  `gorm:"index;size:36" json:"matched_journal_id"`
	MatchedInvoiceIDs string  `gorm:"type:text" json:"matched_invoice_ids"` // JSON array
	MatchedExpenseID  *string `gorm:"size:36" json:"matched_expense_id"`
	MatchType         string  `gorm:"size:20" json:"match_type"` // Auto, Manual, Split, Unmatched
	MatchConfidence   float64 `json:"match_confidence"`

	// Verification
	VerifiedBy string     `gorm:"size:100" json:"verified_by"`
	VerifiedAt *time.Time `json:"verified_at"`
	Notes      string     `gorm:"type:text" json:"notes"`
}

func (BankStatementLine) TableName() string { return "bank_statement_lines" }

type BankLinePaymentAllocation struct {
	Base
	BankStatementLineID string  `gorm:"index;size:36" json:"bank_statement_line_id"`
	AllocationType      string  `gorm:"size:30" json:"allocation_type"` // CUSTOMER_INVOICE, SUPPLIER_INVOICE, EXPENSE
	CustomerInvoiceID   *string `gorm:"size:36" json:"customer_invoice_id"`
	SupplierInvoiceID   *string `gorm:"size:36" json:"supplier_invoice_id"`
	ExpenseEntryID      *string `gorm:"size:36" json:"expense_entry_id"`
	AllocatedAmount     float64 `gorm:"type:decimal(15,3)" json:"allocated_amount"`
	Currency            string  `gorm:"size:3;default:'BHD'" json:"currency"`
	Status              string  `gorm:"size:20;default:'Allocated'" json:"status"` // Allocated, Verified, Disputed
}

func (BankLinePaymentAllocation) TableName() string { return "bank_line_payment_allocations" }

type BankCashBalance struct {
	Base
	BankAccountID    string    `gorm:"uniqueIndex:idx_bank_date;size:36" json:"bank_account_id"`
	BalanceDate      time.Time `gorm:"uniqueIndex:idx_bank_date" json:"balance_date"`
	StatementBalance float64   `gorm:"type:decimal(15,3)" json:"statement_balance"`
	ComputedBalance  float64   `gorm:"type:decimal(15,3)" json:"computed_balance"`
	Currency         string    `gorm:"size:3;default:'BHD'" json:"currency"`
	Discrepancy      float64   `gorm:"type:decimal(15,3)" json:"discrepancy"`
	IsReconciled     bool      `gorm:"default:false" json:"is_reconciled"`
	BankStatementID  *string   `gorm:"size:36" json:"bank_statement_id"`
}

func (BankCashBalance) TableName() string { return "bank_cash_balances" }

type BankExpenseEntry struct {
	Base
	BankStatementLineID string    `gorm:"index;size:36" json:"bank_statement_line_id"`
	Division            string    `gorm:"size:100" json:"division"`
	ExpenseDate         time.Time `json:"expense_date"`
	Description         string    `gorm:"size:500" json:"description"`
	Category            string    `gorm:"size:50" json:"category"` // BANK_FEE, SWIFT_CHARGE, VAT, BG_FEE, OTHER
	Amount              float64   `gorm:"type:decimal(15,3)" json:"amount"`
	Currency            string    `gorm:"size:3;default:'BHD'" json:"currency"`
	VATAmount           float64   `gorm:"type:decimal(15,3)" json:"vat_amount"`
	GLAccountID         *string   `gorm:"size:36" json:"gl_account_id"`
	IsPosted            bool      `gorm:"default:false" json:"is_posted"`
	JournalEntryID      *string   `gorm:"size:36" json:"journal_entry_id"`
}

func (BankExpenseEntry) TableName() string { return "bank_expense_entries" }

type StatementHash struct {
	Base
	BankAccountID    string    `gorm:"index;size:36" json:"bank_account_id"`
	StatementHash    string    `gorm:"uniqueIndex;size:64" json:"statement_hash"`
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	TransactionCount int       `json:"transaction_count"`
	ClosingBalance   float64   `gorm:"type:decimal(15,3)" json:"closing_balance"`
	ImportedAt       time.Time `json:"imported_at"`
	BankStatementID  string    `gorm:"size:36" json:"bank_statement_id"`
}

func (StatementHash) TableName() string { return "statement_hashes" }

// StatementBalanceValidation bundles a balance-check result with its
// discrepancy amount. Wails v2's bound-method marshaling only handles
// OutputCount 1 or 2 (see internal/binding/boundMethod.go) — a 3-value
// Go return silently marshals to null on the JS side. Bundling into a
// struct + error keeps the binding a clean 2-value return.
type StatementBalanceValidation struct {
	IsValid     bool    `json:"is_valid"`
	Discrepancy float64 `json:"discrepancy"`
}

// DuplicateStatementCheck bundles a duplicate-detection result with the
// matching hash record, if any. Same Wails 3-return marshaling bug as
// StatementBalanceValidation above.
type DuplicateStatementCheck struct {
	IsDuplicate bool           `json:"is_duplicate"`
	Existing    *StatementHash `json:"existing"`
}

type BookBankReconciliation struct {
	Base
	BankAccountID          string     `gorm:"index;size:36" json:"bank_account_id"`
	ReconciliationDate     time.Time  `json:"reconciliation_date"`
	Currency               string     `gorm:"size:3;default:'BHD'" json:"currency"`
	BankStatementBalance   float64    `gorm:"type:decimal(15,3)" json:"bank_statement_balance"`
	DepositsInTransit      float64    `gorm:"type:decimal(15,3)" json:"deposits_in_transit"`
	OutstandingCheques     float64    `gorm:"type:decimal(15,3)" json:"outstanding_cheques"`
	BankErrors             float64    `gorm:"type:decimal(15,3)" json:"bank_errors"`
	AdjustedBankBalance    float64    `gorm:"type:decimal(15,3)" json:"adjusted_bank_balance"`
	BookBalance            float64    `gorm:"type:decimal(15,3)" json:"book_balance"`
	BankChargesNotRecorded float64    `gorm:"type:decimal(15,3)" json:"bank_charges_not_recorded"`
	InterestNotRecorded    float64    `gorm:"type:decimal(15,3)" json:"interest_not_recorded"`
	NSFCheques             float64    `gorm:"type:decimal(15,3)" json:"nsf_cheques"`
	BookErrors             float64    `gorm:"type:decimal(15,3)" json:"book_errors"`
	AdjustedBookBalance    float64    `gorm:"type:decimal(15,3)" json:"adjusted_book_balance"`
	Difference             float64    `gorm:"type:decimal(15,3)" json:"difference"`
	IsReconciled           bool       `gorm:"default:false" json:"is_reconciled"`
	ReconciledBy           string     `gorm:"size:100" json:"reconciled_by"`
	ReconciledAt           *time.Time `json:"reconciled_at"`
	Notes                  string     `gorm:"type:text" json:"notes"`
}

func (BookBankReconciliation) TableName() string { return "book_bank_reconciliations" }

type OutstandingCheque struct {
	Base
	BankAccountID string     `gorm:"index;size:36" json:"bank_account_id"`
	ChequeNumber  string     `gorm:"index;size:20" json:"cheque_number"`
	Amount        float64    `gorm:"type:decimal(15,3)" json:"amount"`
	Currency      string     `gorm:"size:3;default:'BHD'" json:"currency"`
	IssuedDate    time.Time  `json:"issued_date"`
	PayeeName     string     `gorm:"size:200" json:"payee_name"`
	PayeeType     string     `gorm:"size:20" json:"payee_type"` // SUPPLIER, EMPLOYEE, OTHER
	SupplierID    *string    `gorm:"size:36" json:"supplier_id"`
	Purpose       string     `gorm:"size:500" json:"purpose"`
	Status        string     `gorm:"index;size:20;default:'ISSUED'" json:"status"` // ISSUED, PRESENTED, CLEARED, STALE, CANCELLED, BOUNCED
	ClearedDate   *time.Time `json:"cleared_date"`
	MatchedLineID *string    `gorm:"size:36" json:"matched_line_id"`
	IsStale       bool       `gorm:"default:false" json:"is_stale"`
	StaleDate     *time.Time `json:"stale_date"`
	ReissuedAs    *string    `gorm:"size:20" json:"reissued_as"`
}

func (OutstandingCheque) TableName() string { return "outstanding_cheques" }

type DepositInTransit struct {
	Base
	BankAccountID string     `gorm:"index;size:36" json:"bank_account_id"`
	DepositDate   time.Time  `json:"deposit_date"`
	Amount        float64    `gorm:"type:decimal(15,3)" json:"amount"`
	Currency      string     `gorm:"size:3;default:'BHD'" json:"currency"`
	DepositSlipNo string     `gorm:"size:50" json:"deposit_slip_no"`
	Description   string     `gorm:"size:500" json:"description"`
	SourceType    string     `gorm:"size:30" json:"source_type"` // CUSTOMER_PAYMENT, CASH_SALE, OTHER
	CustomerID    *string    `gorm:"size:36" json:"customer_id"`
	InvoiceIDs    string     `gorm:"type:text" json:"invoice_ids"`                  // JSON array
	Status        string     `gorm:"index;size:20;default:'PENDING'" json:"status"` // PENDING, CLEARED, RETURNED
	ClearedDate   *time.Time `json:"cleared_date"`
	MatchedLineID *string    `gorm:"size:36" json:"matched_line_id"`
}

func (DepositInTransit) TableName() string { return "deposits_in_transit" }

type ChequeRegister struct {
	Base
	BankAccountID string     `gorm:"index;size:36" json:"bank_account_id"`
	ChequeBookNo  string     `gorm:"size:20" json:"cheque_book_no"`
	StartNumber   int        `json:"start_number"`
	EndNumber     int        `json:"end_number"`
	CurrentNumber int        `json:"current_number"`
	Status        string     `gorm:"index;size:20;default:'ACTIVE'" json:"status"` // ACTIVE, EXHAUSTED, CANCELLED
	IssuedDate    time.Time  `json:"issued_date"`
	ExhaustedDate *time.Time `json:"exhausted_date"`
}

func (ChequeRegister) TableName() string { return "cheque_registers" }

type FXRate struct {
	Base
	FromCurrency string    `gorm:"size:3;uniqueIndex:idx_fx_date,priority:1" json:"from_currency"`
	ToCurrency   string    `gorm:"size:3;uniqueIndex:idx_fx_date,priority:2" json:"to_currency"`
	RateDate     time.Time `gorm:"uniqueIndex:idx_fx_date,priority:3" json:"rate_date"`
	Rate         float64   `gorm:"type:decimal(12,6)" json:"rate"`
	Source       string    `gorm:"size:20" json:"source"` // CBB, MANUAL, API
}

func (FXRate) TableName() string { return "fx_rates" }

type FXRevaluation struct {
	Base
	BankAccountID   string     `gorm:"index;size:36" json:"bank_account_id"`
	RevaluationDate time.Time  `json:"revaluation_date"`
	ForeignCurrency string     `gorm:"size:3" json:"foreign_currency"`
	ForeignBalance  float64    `gorm:"type:decimal(15,3)" json:"foreign_balance"`
	PreviousRate    float64    `gorm:"type:decimal(12,6)" json:"previous_rate"`
	PreviousBHD     float64    `gorm:"type:decimal(15,3)" json:"previous_bhd"`
	CurrentRate     float64    `gorm:"type:decimal(12,6)" json:"current_rate"`
	CurrentBHD      float64    `gorm:"type:decimal(15,3)" json:"current_bhd"`
	GainLossBHD     float64    `gorm:"type:decimal(15,3)" json:"gain_loss_bhd"`
	IsPosted        bool       `gorm:"default:false" json:"is_posted"`
	JournalEntryID  *string    `gorm:"size:36" json:"journal_entry_id"`
	PostedBy        string     `gorm:"size:100" json:"posted_by"`
	PostedAt        *time.Time `json:"posted_at"`
}

func (FXRevaluation) TableName() string { return "fx_revaluations" }

type BankStatementFile struct {
	Base
	BankStatementID string     `gorm:"index;size:36" json:"bank_statement_id"`
	FileName        string     `gorm:"size:255" json:"file_name"`
	FileType        string     `gorm:"size:10" json:"file_type"` // PDF, CSV, XLS
	FileSize        int64      `json:"file_size"`
	FileHash        string     `gorm:"size:64" json:"file_hash"` // SHA-256
	StoragePath     string     `gorm:"size:500" json:"storage_path"`
	IsStored        bool       `gorm:"default:false" json:"is_stored"`
	OCREngine       string     `gorm:"size:30" json:"ocr_engine"`
	OCRConfidence   float64    `json:"ocr_confidence"`
	OCRProcessedAt  *time.Time `json:"ocr_processed_at"`
}

func (BankStatementFile) TableName() string { return "bank_statement_files" }

type BankReconciliationAuditLog struct {
	Base
	BankStatementID     string     `gorm:"index;size:36" json:"bank_statement_id"`
	BankStatementLineID *string    `gorm:"index;size:36" json:"bank_statement_line_id"`
	Action              string     `gorm:"size:30" json:"action"`          // IMPORT, MATCH, UNMATCH, SPLIT, CATEGORIZE, RECONCILE, VERIFY
	ActionDetail        string     `gorm:"type:text" json:"action_detail"` // JSON
	PerformedBy         string     `gorm:"size:100" json:"performed_by"`
	PerformedAt         time.Time  `gorm:"index" json:"performed_at"`
	IsAutomatic         bool       `gorm:"default:false" json:"is_automatic"`
	ConfidenceScore     float64    `json:"confidence_score"`
	Reason              string     `gorm:"type:text" json:"reason"`
	IsReversed          bool       `gorm:"default:false" json:"is_reversed"`
	ReversedBy          string     `gorm:"size:100" json:"reversed_by"`
	ReversedAt          *time.Time `json:"reversed_at"`
	ReversalReason      string     `gorm:"type:text" json:"reversal_reason"`
}

func (BankReconciliationAuditLog) TableName() string { return "bank_reconciliation_audit_logs" }
