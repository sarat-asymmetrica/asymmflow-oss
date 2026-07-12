// Package crm contains the CRM domain model.
package crm

import (
	"time"

	shareddomain "ph_holdings_app/pkg/domain"

	"gorm.io/gorm"
)

type Base = shareddomain.Base

type CustomerMaster struct {
	Base
	CustomerID   string `gorm:"uniqueIndex;size:50" json:"customer_id"`
	CustomerCode string `gorm:"uniqueIndex;size:50" json:"customer_code"` // Alternative business identifier
	CustomerType string `gorm:"index;size:50" json:"customer_type"`
	BusinessName string `gorm:"index;size:255" json:"business_name"`
	ShortCode    string `gorm:"size:10" json:"short_code"`

	// Identity & Registration
	TradingName string `gorm:"size:255" json:"trading_name"`
	CRNumber    string `gorm:"size:100" json:"cr_number"`
	Status      string `gorm:"size:50;default:'Active'" json:"status"`

	// Contact & Regional
	PrimaryPhone string `gorm:"size:50" json:"primary_phone"`
	PrimaryEmail string `gorm:"size:255" json:"primary_email"`
	Website      string `gorm:"size:255" json:"website"`
	AddressLine1 string `json:"address_line1"`
	City         string `gorm:"index;size:100" json:"city"`
	Country      string `gorm:"index;size:100" json:"country"`
	TRN          string `gorm:"size:100" json:"trn"`
	MobileNumber string `gorm:"size:50" json:"mobile_number"`

	// PH legacy contact/registration columns retained verbatim (PC-D22, Mission
	// I D-I-5). PH kept these alongside the OSS-spelled columns above
	// (address_line1 / primary_phone / primary_email), so they are distinct
	// stored data, not renames — the importer carries both.
	TaxCode string `gorm:"size:100" json:"tax_code"`
	Address string `json:"address"`
	Phone   string `gorm:"size:50" json:"phone"`
	Email   string `gorm:"size:255" json:"email"`

	// Business Details
	Industry      string `gorm:"index;size:100" json:"industry"`
	RelationYears int    `json:"relation_years"`

	// Performance Profile
	// PaymentGrade has standalone index for grade-only queries: WHERE payment_grade = ?
	// Used by: GetCustomersByGrade() - retrieves all customers of a specific grade (A/B/C/D)
	// No composite index needed here - CustomerID is uniqueIndex (1:1 lookup), PaymentGrade is low cardinality
	PaymentGrade     string  `gorm:"index;size:10;default:'C'" json:"payment_grade"`  // A/B/C/D grade (P1 Fix: default)
	CustomerGrade    string  `gorm:"index;size:10;default:'C'" json:"customer_grade"` // Overall customer grade (P1 Fix: default)
	PaymentTermsDays int     `gorm:"index" json:"payment_terms_days"`                 // P1 Fix: indexed for payment terms queries
	AvgPaymentDays   float64 `json:"avg_payment_days"`
	DisputeCount     int     `json:"dispute_count"`

	// Financial Metrics
	TotalOrdersValue float64    `json:"total_orders_value"`
	TotalOrdersCount int        `json:"total_orders_count"`
	AvgOrderValue    float64    `json:"avg_order_value"`
	LastOrderDate    *time.Time `json:"last_order_date"`

	// AR Tracking (Accounts Receivable)
	ARRiskTier     string  `gorm:"index;size:20" json:"ar_risk_tier"`     // Low/Medium/High/Critical
	OutstandingBHD float64 `json:"outstanding_bhd"`                       // Total outstanding amount
	OverdueDays    int     `gorm:"index" json:"overdue_days"`             // Days overdue for oldest invoice
	CreditLimitBHD float64 `gorm:"default:50000" json:"credit_limit_bhd"` // P1 FIX: Credit limit in BHD (default 50,000)

	// Flags (PostgreSQL boolean types)
	IsCreditBlocked    bool `gorm:"index;default:false" json:"is_credit_blocked"`
	RequiresPrepayment bool `gorm:"default:false" json:"requires_prepayment"`
	HasABBCompetition  bool `gorm:"default:false" json:"has_abb_competition"`
	IsEmergencyOnly    bool `gorm:"default:false" json:"is_emergency_only"`
}

func (CustomerMaster) TableName() string { return "customers" }

type CustomerContact struct {
	Base
	CustomerID       string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	ContactName      string `gorm:"size:255" json:"contact_name"`
	JobTitle         string `gorm:"size:100" json:"job_title"`
	Email            string `gorm:"size:255" json:"email"`
	Phone            string `gorm:"size:50" json:"phone"`
	Address          string `gorm:"type:varchar(1000)" json:"address"`
	IsPrimaryContact bool   `json:"is_primary_contact"`

	// PH legacy columns retained verbatim (PC-D22, Mission I D-I-5). PH kept
	// is_primary alongside is_primary_contact, so both are carried distinctly.
	IsPrimary  bool   `json:"is_primary"`
	Salutation string `gorm:"size:50" json:"salutation"`
}

type SupplierContact struct {
	Base
	SupplierID       string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:SupplierID;references:ID" json:"supplier_id"`
	ContactName      string `gorm:"size:255" json:"contact_name"`
	JobTitle         string `gorm:"size:100" json:"job_title"`
	Email            string `gorm:"size:255" json:"email"`
	Phone            string `gorm:"size:50" json:"phone"`
	Address          string `gorm:"type:varchar(1000)" json:"address"`
	IsPrimaryContact bool   `json:"is_primary_contact"`
}

type SupplierMaster struct {
	Base
	SupplierCode string `gorm:"uniqueIndex;size:50" json:"supplier_code"`
	SupplierName string `gorm:"index;size:255" json:"supplier_name"`
	Country      string `gorm:"index;size:100" json:"country"`
	LeadTimeDays int    `json:"lead_time_days"`

	// Identity Section
	TaxID        string `gorm:"size:100" json:"tax_id"`
	SupplierType string `gorm:"index;size:50" json:"supplier_type"` // Manufacturer, Distributor, Agent

	// Offerings Section
	BrandsHandled string `gorm:"type:varchar(2000)" json:"brands_handled"` // JSON array of brand names
	ProductTypes  string `gorm:"type:varchar(2000)" json:"product_types"`  // JSON array

	// Contact Section
	PrimaryContact string `gorm:"size:255" json:"primary_contact"`
	Email          string `gorm:"size:255" json:"email"`
	Phone          string `gorm:"size:50" json:"phone"`
	Address        string `gorm:"type:varchar(1000)" json:"address"`

	// Bank Details Section (for payments)
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	IBAN          string `gorm:"size:50" json:"iban"`
	SwiftCode     string `gorm:"size:20" json:"swift_code"`

	// Commercial Terms
	PaymentTerms string `gorm:"index;size:100;default:'Net 30'" json:"payment_terms"` // P1 Fix: indexed for payment terms queries

	// Rating Section
	Rating int    `gorm:"index;default:3" json:"rating"` // P1 Fix: indexed for star rating queries
	Notes  string `gorm:"type:varchar(5000)" json:"notes"`

	// PH legacy column retained verbatim (PC-D22, Mission I D-I-5): active flag.
	IsActive bool `gorm:"index" json:"is_active"`
}

func (SupplierMaster) TableName() string { return "suppliers" }

type EntityNote struct {
	Base
	EntityType string `gorm:"index;size:20;check:entity_type IN ('customer','supplier')" json:"entity_type"` // "customer" or "supplier"
	EntityID   string `gorm:"index;size:36" json:"entity_id"`
	NoteType   string `gorm:"index;size:50;check:note_type IN ('delivery','issue','general','payment','commercial','technical')" json:"note_type"` // "delivery", "issue", "general", "payment"
	Content    string `gorm:"type:varchar(5000)" json:"content"`
}

type SupplierIssue struct {
	Base
	SupplierID  string     `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:SupplierID;references:ID" json:"supplier_id"`
	OrderRef    string     `gorm:"size:100" json:"order_ref"`
	Description string     `gorm:"type:varchar(5000)" json:"description"`
	Status      string     `gorm:"index;size:50;check:status IN ('open','pending','resolved','escalated')" json:"status"` // "open", "pending", "resolved"
	Resolution  string     `gorm:"type:varchar(5000)" json:"resolution"`
	CostBHD     float64    `gorm:"check:cost_bhd >= 0" json:"cost_bhd"`
	ResolvedAt  *time.Time `json:"resolved_at"`
}

type ProductMaster struct {
	Base
	ProductCode      string  `gorm:"uniqueIndex;size:100" json:"product_code"`
	ProductName      string  `gorm:"index;size:255" json:"product_name"`
	ProductCategory  string  `gorm:"index;size:100" json:"product_category"`
	SupplierID       string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT;" json:"supplier_id"`
	SupplierCode     string  `gorm:"index;size:50" json:"supplier_code"`
	StandardCostBHD  float64 `json:"standard_cost_bhd"`
	StandardPriceBHD float64 `json:"standard_price_bhd"`
	Description      string  `gorm:"type:text" json:"description"`
	IsActive         bool    `gorm:"index" json:"is_active"`
	StockQuantity    int     `json:"stock_quantity"`

	// Identity Section (NEW)
	SKU        string `gorm:"index;size:100" json:"sku"`
	PartNumber string `gorm:"size:100" json:"part_number"`

	// Commercial Section (NEW)
	HSCode        string `gorm:"size:20" json:"hs_code"` // Harmonized System Code
	UnitOfMeasure string `gorm:"size:20" json:"unit_of_measure"`

	// Technical Section (NEW)
	DatasheetURL   string `gorm:"type:varchar(2000)" json:"datasheet_url"`
	Specifications string `gorm:"type:varchar(5000)" json:"specifications"` // JSON

	// Serial Tracking (Phase 23)
	RequiresSerialTracking bool `gorm:"default:false" json:"requires_serial_tracking"`
}

type Offer struct {
	Base
	OfferNumber    string `gorm:"uniqueIndex;size:50" json:"offer_number"`
	RevisionNumber int    `gorm:"index" json:"revision_number"`

	// Revision/renewal lineage (FLOW-006). When an offer is re-quoted or an
	// expired offer is renewed, a new Offer is cloned and these fields link the
	// clone back to its source and to the original root, while the source is
	// stamped as superseded. Column names match the deployed PH schema.
	RevisionOfOfferID   string     `gorm:"index;size:36" json:"revision_of_offer_id"`
	RevisionRootOfferID string     `gorm:"index;size:36" json:"revision_root_offer_id"`
	SupersededByOfferID string     `gorm:"index;size:36" json:"superseded_by_offer_id"`
	SupersededAt        *time.Time `json:"superseded_at"`

	// Relationships
	RFQID        string `gorm:"index;size:36" json:"rfq_id"` // Fixed: was uint, now string for consistency with Order.RFQID
	CustomerID   string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	CustomerName string `gorm:"index;size:255" json:"customer_name"`

	// Dates
	// QuotationDate indexed for dashboard date range filters: WHERE quotation_date BETWEEN ? AND ?
	// Used by: Dashboard timeline, monthly/quarterly reports, offer aging analysis
	QuotationDate time.Time `gorm:"index" json:"quotation_date"`
	ValidityDate  time.Time `json:"validity_date"`

	// Metrics
	TotalValueBHD   float64 `gorm:"not null;default:0;check:total_value_bhd >= 0" json:"total_value_bhd"`
	EstimatedMargin float64 `gorm:"not null;default:0" json:"estimated_margin"`
	Stage           string  `gorm:"index;size:50;check:stage IN ('RFQ','Quoted','Won','Lost','Expired')" json:"stage"` // RFQ, Quoted, Won, Lost

	// Competition
	HasABBCompetition bool   `json:"has_abb_competition"`
	LostReason        string `gorm:"type:varchar(2000)" json:"lost_reason"`

	// PDF Generation Fields (Terms & Conditions)
	PaymentTerms       string  `gorm:"type:varchar(500);default:'30 days from Date of Delivery'" json:"payment_terms"`
	DeliveryTerms      string  `gorm:"type:varchar(500);default:'DAP Bahrain at your store or Acme Instrumentation'" json:"delivery_terms"`
	DeliveryWeeks      string  `gorm:"size:50;default:'4-6 weeks'" json:"delivery_weeks"`
	CountryOfOrigin    string  `gorm:"size:100;default:'Germany / USA'" json:"country_of_origin"`
	IssuedBy           string  `gorm:"size:100" json:"issued_by"`
	ContactPhone       string  `gorm:"size:50" json:"contact_phone"`
	CustomerReference  string  `gorm:"size:255" json:"customer_reference"` // RFQ/enquiry reference
	AttentionPerson    string  `gorm:"size:255" json:"attention_person"`   // Contact name
	AttentionCompany   string  `gorm:"size:255" json:"attention_company"`
	AttentionPhone     string  `gorm:"size:50" json:"attention_phone"`
	AttentionAddress   string  `gorm:"type:varchar(1000)" json:"attention_address"`
	DiscountPercent    float64 `json:"discount_percent"`
	QuoteType          string  `gorm:"size:50;default:'Quotation'" json:"quote_type"` // "Quotation" or "Budgetary Quote"
	VatRate            float64 `gorm:"default:10" json:"vat_rate"`                    // VAT percentage (default 10%)
	AttachmentScopeID  string  `gorm:"size:160" json:"attachment_scope_id"`           // I-25: binds this offer to its costing_sheet_attachments datasheets
	Division           string  `gorm:"size:100" json:"division"`
	TermsAndConditions string  `gorm:"type:text" json:"terms_and_conditions"`
	Subject            string  `gorm:"type:varchar(1000)" json:"subject"`
	Body               string  `gorm:"type:text" json:"body"`

	// Additional costing header fields
	CocCoo          string `gorm:"size:50" json:"coc_coo"`                    // Certificate of Conformity/Country of Origin
	TestCertificate string `gorm:"type:varchar(500)" json:"test_certificate"` // Test Certificate requirements
	Installation    string `gorm:"size:50" json:"installation"`               // Installation included?
	Commissioning   string `gorm:"size:50" json:"commissioning"`              // Commissioning included?
	Testing         string `gorm:"type:varchar(500)" json:"testing"`          // Testing requirements

	// Derived UI fields from linked opportunity metadata.
	FolderNumber string `gorm:"-" json:"folder_number"`
	ProjectName  string `gorm:"-" json:"project_name"`

	Items []OfferItem `gorm:"foreignKey:OfferID" json:"items,omitempty"`
}

type OfferItem struct {
	Base
	OfferID    string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:OfferID;references:ID" json:"offer_id"`
	LineNumber int    `json:"line_number"`

	ProductID   string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:ProductID;references:ID" json:"product_id"`
	ProductCode string  `gorm:"size:100" json:"product_code"`          // Model number (legacy)
	Model       string  `gorm:"size:255" json:"model"`                 // Model number (explicit)
	Description string  `gorm:"type:varchar(2000)" json:"description"` // Equipment name - Model
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price_bhd"`

	// Extended fields for full costing data visibility (Fix #5)
	LongCode            string  `gorm:"type:varchar(500)" json:"long_code"`             // Long supplier code (e.g. Rhine Instruments instrumentation codes)
	Equipment           string  `gorm:"size:255" json:"equipment"`                      // Equipment/Product name
	Specification       string  `gorm:"type:varchar(2000)" json:"specification"`        // Short specification
	DetailedDescription string  `gorm:"type:varchar(5000)" json:"detailed_description"` // Full instrumentation specs
	Currency            string  `gorm:"size:10" json:"currency"`                        // Source currency
	FOB                 float64 `json:"fob"`                                            // FOB cost
	Freight             float64 `json:"freight"`                                        // Freight cost
	TotalCost           float64 `json:"total_cost"`                                     // Total cost (landed + handling + finance)
	MarginPercent       float64 `json:"margin_percent"`                                 // Margin % used
	TotalPrice          float64 `json:"total_price"`                                    // Line total (qty × unit price)

	// Full cost breakdown (for detailed costing persistence - Fix 2026-02-05)
	ExchangeRate    float64 `json:"exchange_rate"`    // Currency conversion rate
	FobBHD          float64 `json:"fob_bhd"`          // FOB cost in BHD
	FreightBHD      float64 `json:"freight_bhd"`      // Freight cost in BHD
	Insurance       float64 `json:"insurance"`        // Insurance charges
	CustomsPercent  float64 `json:"customs_percent"`  // Customs duty percentage
	CustomsBHD      float64 `json:"customs_bhd"`      // Customs duty in BHD
	HandlingPercent float64 `json:"handling_percent"` // Handling charges percentage
	HandlingBHD     float64 `json:"handling_bhd"`     // Handling charges in BHD
	FinancePercent  float64 `json:"finance_percent"`  // Finance cost percentage
	FinanceBHD      float64 `json:"finance_bhd"`      // Finance cost in BHD
	OtherCosts      float64 `json:"other_costs"`      // Other miscellaneous costs
	UserPrice       float64 `json:"user_price"`       // User-overridden price (if manually set)
	UserPriceSet    bool    `json:"user_price_set"`   // Boolean flag for manual price override
}

type Opportunity struct {
	Base
	FolderNumber string `gorm:"uniqueIndex;size:50" json:"folder_number"`                                                                     // Physical folder tracking number
	OfferID      string `gorm:"index;size:36;constraint:OnDelete:SET NULL,OnUpdate:CASCADE;foreignKey:OfferID;references:ID" json:"offer_id"` // Link to Offer if quoted

	// Customer & Sales
	CustomerID    string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	CustomerName  string `gorm:"index;size:255" json:"customer_name"`
	CustomerGrade string `gorm:"size:10" json:"customer_grade"` // Grade at time of opportunity (A/B/C)
	Salesperson   string `gorm:"index;size:100" json:"salesperson"`
	Division      string `gorm:"size:100" json:"division"`

	// Pipeline Identification (from canonical seed)
	Year           int    `gorm:"index" json:"year"`                     // Pipeline year (2024, 2025, 2026)
	OppNumber      int    `json:"opp_number"`                            // Opportunity sequence number within year
	FolderName     string `gorm:"size:500" json:"folder_name"`           // Full descriptive folder name
	Title          string `gorm:"size:500" json:"title"`                 // SFDC opportunity title
	EHRef          string `gorm:"size:100" json:"eh_ref"`                // Rhine Instruments reference number
	Source         string `gorm:"index;size:50" json:"source"`           // Data source: old_db, 2025_excel, 2026_ocr
	Comment        string `gorm:"type:varchar(2000)" json:"comment"`     // Latest salesperson comment
	OwnerNotes     string `gorm:"type:varchar(2000)" json:"owner_notes"` // Management notes (Abie's notes)
	ProductDetails string `gorm:"type:text" json:"product_details"`      // JSON array of extracted opportunity line items

	// Dates
	OfferDate    time.Time  `gorm:"index" json:"offer_date"`
	OrderDate    *time.Time `json:"order_date"`    // Date PO/order received
	ExpectedDate *time.Time `json:"expected_date"` // Expected close date
	ClosedDate   *time.Time `json:"closed_date"`   // Actual close date

	// Delivery & Terms
	DeliveryTerms string `gorm:"size:200" json:"delivery_terms"` // Delivery weeks/terms
	PaymentTerms  string `gorm:"size:500" json:"payment_terms"`  // Payment terms from pipeline

	// Financial
	RevenueBHD float64 `json:"revenue_bhd"` // Expected revenue
	CostBHD    float64 `json:"cost_bhd"`    // Expected cost
	ProfitBHD  float64 `json:"profit_bhd"`  // Expected profit

	// Status Tracking
	SPOCStatus string `gorm:"column:spoc_status;index;size:50" json:"spoc_status"` // Single Point of Contact status
	WIPStatus  string `gorm:"column:wip_status;index;size:50" json:"wip_status"`   // Work In Progress status
	Stage      string `gorm:"index;size:50" json:"stage"`                          // Lead/Qualified/Proposal/Negotiation/Won/Lost/Quoted

	// Three-Regime Dynamics (Asymmetrica Mathematical Framework)
	Regime     int     `gorm:"index" json:"regime"` // 1=Exploration(0-30%), 2=Optimization(30-50%), 3=Stabilization(50%+)
	Confidence float64 `json:"confidence"`          // 0.0-1.0 confidence score
	R1         float64 `json:"r1"`                  // Regime 1 weight (Exploration)
	R2         float64 `json:"r2"`                  // Regime 2 weight (Optimization)
	R3         float64 `json:"r3"`                  // Regime 3 weight (Stabilization)

	// Competition & Product
	HasABBCompetition bool   `json:"has_abb_competition"`
	ProductType       string `gorm:"index;size:100" json:"product_type"` // Oxan Analytics, Rhine Instruments_flow, etc.

	// Outcome
	WonReason  string `gorm:"type:varchar(2000)" json:"won_reason"`
	LostReason string `gorm:"type:varchar(2000)" json:"lost_reason"`
}

type Order struct {
	Base
	OrderNumber      string `gorm:"uniqueIndex;size:50" json:"order_number"`
	CustomerPONumber string `gorm:"index;size:100" json:"customer_po_number"`

	// P1 Fix: Added composite index for customer order queries: WHERE customer_id = ? AND status = ?
	CustomerID   string `gorm:"index:idx_order_customer_status,priority:1;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	CustomerName string `gorm:"index;size:255" json:"customer_name"`

	// OrderDate indexed for date range queries: WHERE order_date BETWEEN ? AND ?
	// Used by: Order history, timeline filters, monthly reports, customer 360 view
	OrderDate     time.Time `gorm:"index;autoCreateTime:false" json:"order_date"`
	RequiredDate  time.Time `json:"required_date"`
	TotalValueBHD float64   `gorm:"type:decimal(15,3);not null;default:0;check:total_value_bhd >= 0" json:"total_value_bhd"` // P1 Fix: BHD precision
	GrandTotalBHD float64   `gorm:"type:decimal(15,3);not null;default:0;check:grand_total_bhd >= 0" json:"grand_total_bhd"` // P1 Fix: BHD precision
	// Status indexed for order pipeline: WHERE status NOT IN ('Delivered', 'Cancelled')
	// P1 Fix: Part of composite index idx_order_customer_status
	Status string `gorm:"index:idx_order_customer_status,priority:2;size:50;default:'Processing'" json:"status"` // Removed CHECK constraint - data has various statuses

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`

	PaymentTerms  string `gorm:"type:varchar(500)" json:"payment_terms"`
	DeliveryTerms string `gorm:"type:varchar(500)" json:"delivery_terms"`

	// Traceability - link to source Offer
	OfferID     string `gorm:"index;size:36" json:"offer_id"`
	OfferNumber string `gorm:"size:50" json:"offer_number"`
	RFQID       string `gorm:"index;size:36" json:"rfq_id"` // Link to RFQ/Opportunity

	// Contact & RFQ Details (from Offer - for invoice generation)
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

	// E2: Division field - 'Acme Instrumentation' or 'Beacon Controls' (sister companies)
	Division string `gorm:"size:100" json:"division"`

	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

type OrderItem struct {
	Base
	OrderID    string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:OrderID;references:ID" json:"order_id"`
	LineNumber int    `json:"line_number"`

	ProductID   string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:ProductID;references:ID" json:"product_id"`
	ProductCode string  `gorm:"size:100" json:"product_code"`
	Description string  `gorm:"type:varchar(2000)" json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price_bhd"`

	// Fulfillment tracking
	QuantityShipped  float64 `json:"quantity_shipped"`
	QuantityInvoiced float64 `json:"quantity_invoiced"`

	// Extended costing fields (from OfferItem - costing sheet data)
	Equipment           string  `gorm:"size:255" json:"equipment"`                      // Equipment/Product name
	Model               string  `gorm:"size:255" json:"model"`                          // Model number
	Specification       string  `gorm:"type:varchar(2000)" json:"specification"`        // Short specification
	DetailedDescription string  `gorm:"type:varchar(5000)" json:"detailed_description"` // Full instrumentation specs
	Currency            string  `gorm:"size:10" json:"currency"`                        // Source currency (USD, EUR, etc.)
	FOB                 float64 `json:"fob"`                                            // FOB cost
	Freight             float64 `json:"freight"`                                        // Freight cost
	TotalCost           float64 `json:"total_cost"`                                     // Total landed cost
	MarginPercent       float64 `json:"margin_percent"`                                 // Margin % applied
	TotalPrice          float64 `json:"total_price"`                                    // Line total (qty × unit price)

	// PH legacy/enrichment columns retained verbatim (PC-D22, Mission I D-I-5).
	// unit_price_bhd is a column PH kept distinct from the unit_price above (the
	// OSS UnitPrice field owns column unit_price but serialises as unit_price_bhd
	// for the frontend, so this field takes json:"-" to avoid a duplicate JSON
	// key while still carrying the source column). brand/token identify the
	// instrument — PH has no SKU catalog, it keys instruments by brand × token.
	UnitPriceBHD float64 `gorm:"column:unit_price_bhd;type:decimal(15,3)" json:"-"`
	Brand        string  `gorm:"size:255" json:"brand"`
	Token        string  `gorm:"size:255" json:"token"`
}

type Shipment struct {
	Base
	OrderID        string     `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:OrderID;references:ID" json:"order_id"`
	OrderNumber    string     `gorm:"index;size:50" json:"order_number"`
	Status         string     `gorm:"index;size:50;check:status IN ('Pending','In Transit','Delivered','Failed','Cancelled')" json:"status"`
	ShipmentDate   time.Time  `json:"shipment_date"`
	DeliveredDate  *time.Time `json:"delivered_date,omitempty"`
	CourierName    string     `json:"courier_name"`
	TrackingNumber string     `json:"tracking_number"`
}

type PostSaleNote struct {
	Base
	OrderID     string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:OrderID;references:ID" json:"order_id"`
	OrderNumber string `gorm:"index;size:50" json:"order_number"`

	NoteType    string    `gorm:"index;size:50;check:note_type IN ('repair','replace','reinstall','warranty','refund','calibration','other')" json:"note_type"` // repair, replace, reinstall, warranty, refund, other
	Description string    `gorm:"type:varchar(5000)" json:"description"`
	CostBHD     float64   `gorm:"check:cost_bhd >= 0" json:"cost_bhd"`
	NoteDate    time.Time `gorm:"index" json:"note_date"`

	// Resolution tracking
	ResolvedAt *time.Time `json:"resolved_at"`
	Resolution string     `gorm:"type:varchar(5000)" json:"resolution"`
}

type DeliveryNote struct {
	Base
	// Links - P1 Fix: Composite index for order delivery lookups: WHERE order_id = ? AND status = ?
	OrderID    string `gorm:"index:idx_dn_order_status,priority:1;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:OrderID;references:ID" json:"order_id"`
	CustomerID string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`

	// Document Info
	DNNumber     string    `gorm:"uniqueIndex;size:50" json:"dn_number"` // DN-2026-0001
	DeliveryDate time.Time `gorm:"index;autoCreateTime:false" json:"delivery_date"`

	// Delivery Details
	DeliveryAddress string `gorm:"type:varchar(1000)" json:"delivery_address"`
	ContactPerson   string `json:"contact_person"`
	ContactPhone    string `json:"contact_phone"`

	// Transport
	DriverName      string `json:"driver_name"`
	VehicleNumber   string `json:"vehicle_number"`
	TransportMethod string `json:"transport_method"` // Own Vehicle, Courier, Customer Pickup

	// Status Flow: Prepared → Dispatched → InTransit → Delivered → Signed
	// P1 Fix: Part of composite index idx_dn_order_status
	Status string `gorm:"index:idx_dn_order_status,priority:2;size:50;default:'Prepared';check:status IN ('Prepared','Dispatched','InTransit','Delivered','Signed','Cancelled')" json:"status"` // P1 Fix: default status

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`

	// Proof of Delivery
	SignedBy       string     `json:"signed_by"`
	SignedAt       *time.Time `json:"signed_at"`
	SignatureImage string     `json:"signature_image"` // Base64 or file path (future)

	// Partial Delivery Tracking
	IsPartialDelivery bool `json:"is_partial_delivery"`
	DeliverySequence  int  `json:"delivery_sequence"` // 1 of 3, 2 of 3, etc.
	TotalDeliveries   int  `json:"total_deliveries"`

	Items []DeliveryNoteItem `gorm:"foreignKey:DeliveryNoteID" json:"items,omitempty"`
}

func (DeliveryNote) TableName() string { return "delivery_notes" }

type DeliveryNoteItem struct {
	Base
	DeliveryNoteID    string  `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:DeliveryNoteID;references:ID" json:"delivery_note_id"`
	OrderItemID       string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:OrderItemID;references:ID" json:"order_item_id"`
	ProductID         string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:ProductID;references:ID" json:"product_id"`
	ProductCode       string  `gorm:"size:100" json:"product_code"`
	Description       string  `gorm:"type:varchar(2000)" json:"description"`
	QuantityOrdered   float64 `json:"quantity_ordered"`   // Original order qty
	QuantityDelivered float64 `json:"quantity_delivered"` // This delivery
	QuantityRemaining float64 `json:"quantity_remaining"` // Left to deliver
}

func (DeliveryNoteItem) TableName() string { return "delivery_note_items" }

type DBCostingSheet struct {
	Base
	CostingNumber string `gorm:"uniqueIndex;size:50" json:"costing_number"`
	CustomerID    string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	CustomerName  string `json:"customer_name"`

	// Dates
	CostingDate time.Time `json:"costing_date"`
	ValidUntil  time.Time `json:"valid_until"`

	// Line Item Totals
	SubtotalBHD    float64 `gorm:"not null;default:0;check:subtotal_bhd >= 0" json:"subtotal_bhd"`
	TotalMarginBHD float64 `gorm:"not null;default:0" json:"total_margin_bhd"`

	// Bahrain Logistics Costs
	ShippingCostBHD  float64 `gorm:"not null;default:0;check:shipping_cost_bhd >= 0" json:"shipping_cost_bhd"`
	CustomsDutyBHD   float64 `gorm:"not null;default:0;check:customs_duty_bhd >= 0" json:"customs_duty_bhd"`
	ClearanceCostBHD float64 `gorm:"not null;default:0;check:clearance_cost_bhd >= 0" json:"clearance_cost_bhd"`
	HandlingCostBHD  float64 `gorm:"not null;default:0;check:handling_cost_bhd >= 0" json:"handling_cost_bhd"`

	// Additional Costs Total
	AdditionalCostsBHD float64 `gorm:"not null;default:0;check:additional_costs_bhd >= 0" json:"additional_costs_bhd"`

	// Grand Total
	GrandTotalBHD float64 `gorm:"not null;default:0;check:grand_total_bhd >= 0" json:"grand_total_bhd"`

	// Status
	Status             string `gorm:"index;size:50;default:'Draft';check:status IN ('Draft','Approved','Converted','Rejected')" json:"status"` // Draft, Approved, Converted
	ConvertedToOfferID string `gorm:"size:36" json:"converted_to_offer_id"`

	// Relationships
	Items           []DBCostingItem           `gorm:"foreignKey:CostingSheetID" json:"items,omitempty"`
	AdditionalCosts []DBCostingAdditionalCost `gorm:"foreignKey:CostingSheetID" json:"additional_costs,omitempty"`
}

type DBCostingItem struct {
	Base
	CostingSheetID string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:CostingSheetID;references:ID" json:"costing_sheet_id"`
	LineNumber     int    `json:"line_number"`

	ProductID   string `gorm:"size:36" json:"product_id"`
	ProductType string `gorm:"size:100" json:"product_type"`
	Description string `gorm:"type:varchar(2000)" json:"description"`

	Quantity      float64 `gorm:"not null;default:0;check:quantity >= 0" json:"quantity"`
	UnitCostBHD   float64 `gorm:"not null;default:0;check:unit_cost_bhd >= 0" json:"unit_cost_bhd"`
	MarginPercent float64 `gorm:"not null;default:0" json:"margin_percent"`
	UnitPriceBHD  float64 `gorm:"not null;default:0;check:unit_price_bhd >= 0" json:"unit_price_bhd"`
	LineTotalBHD  float64 `gorm:"not null;default:0;check:line_total_bhd >= 0" json:"line_total_bhd"`
}

type DBCostingAdditionalCost struct {
	Base
	CostingSheetID string  `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:CostingSheetID;references:ID" json:"costing_sheet_id"`
	Description    string  `gorm:"size:500" json:"description"`
	AmountBHD      float64 `gorm:"not null;default:0;check:amount_bhd >= 0" json:"amount_bhd"`
}

type SerialNumber struct {
	Base
	ProductID   string `gorm:"index;size:36" json:"product_id"`
	ProductCode string `gorm:"index;size:100" json:"product_code"`
	SerialNo    string `gorm:"uniqueIndex;size:255" json:"serial_no"`
	LotNumber   string `gorm:"index;size:100" json:"lot_number"`
	Status      string `gorm:"index;size:50;default:'Available'" json:"status"` // Available, Reserved, Shipped, Delivered

	// Traceability chain
	POID          string `gorm:"index;size:36" json:"po_id"`
	PONumber      string `gorm:"size:50" json:"po_number"`
	GRNItemID     string `gorm:"index;size:36" json:"grn_item_id"`
	GRNNumber     string `gorm:"size:50" json:"grn_number"`
	DNItemID      string `gorm:"index;size:36" json:"dn_item_id"`
	DNNumber      string `gorm:"size:50" json:"dn_number"`
	InvoiceID     string `gorm:"index;size:36" json:"invoice_id"`
	InvoiceNumber string `gorm:"size:50" json:"invoice_number"`
	CustomerID    string `gorm:"index;size:36" json:"customer_id"`
	CustomerName  string `gorm:"size:255" json:"customer_name"`

	// Dates
	ReceivedDate *time.Time `json:"received_date"`
	ShippedDate  *time.Time `json:"shipped_date"`

	// Warranty
	WarrantyStartDate *time.Time `json:"warranty_start_date"`
	WarrantyEndDate   *time.Time `json:"warranty_end_date"`
	WarrantyMonths    int        `json:"warranty_months"`

	// Calibration
	CalibrationDate     *time.Time `json:"calibration_date"`
	CalibrationDueDate  *time.Time `json:"calibration_due_date"`
	CalibrationCertPath string     `gorm:"size:500" json:"calibration_cert_path"`

	Notes string `gorm:"type:varchar(2000)" json:"notes"`
}

func (SerialNumber) TableName() string { return "serial_numbers" }

type InventoryItem struct {
	Base
	ProductID         string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT;" json:"product_id"`
	ProductCode       string  `gorm:"index;size:100" json:"product_code"`
	WarehouseID       string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT;" json:"warehouse_id"`
	QuantityOnHand    float64 `gorm:"type:decimal(15,3);default:0" json:"quantity_on_hand"`
	QuantityReserved  float64 `gorm:"type:decimal(15,3);default:0" json:"quantity_reserved"`
	QuantityAvailable float64 `gorm:"type:decimal(15,3);default:0" json:"quantity_available"`
	UnitCost          float64 `gorm:"type:decimal(15,2)" json:"unit_cost"`
	StockStatus       string  `gorm:"index;size:50" json:"stock_status"`
	IsActive          bool    `gorm:"index" json:"is_active"`

	ReorderPoint     float64    `json:"reorder_point"`
	MinimumStock     float64    `json:"minimum_stock"`
	MaximumStock     float64    `json:"maximum_stock"`
	TotalValue       float64    `json:"total_value"`
	LastPurchaseCost float64    `json:"last_purchase_cost"`
	LastMovementAt   *time.Time `json:"last_movement_at"`
}

func (InventoryItem) TableName() string { return "inventory_items" }

type StockMovement struct {
	Base
	InventoryItemID string    `gorm:"index;size:36;constraint:OnDelete:CASCADE;" json:"inventory_item_id"`
	MovementType    string    `json:"movement_type"`
	MovementNumber  string    `json:"movement_number"`
	Quantity        float64   `json:"quantity"`
	Direction       string    `json:"direction"`
	BalanceBefore   float64   `json:"balance_before"`
	BalanceAfter    float64   `json:"balance_after"`
	MovementDate    time.Time `json:"movement_date"`

	// Provenance of the movement (PH parity): what document caused it and,
	// where relevant, who the counterparty was.
	ReferenceType    string `gorm:"index;size:50" json:"reference_type"`
	ReferenceID      string `gorm:"index;size:36" json:"reference_id"`
	ReferenceNumber  string `gorm:"size:100" json:"reference_number"`
	CounterpartyID   string `gorm:"index;size:36" json:"counterparty_id"`
	CounterpartyName string `gorm:"size:255" json:"counterparty_name"`
	Notes            string `gorm:"type:varchar(2000)" json:"notes"`

	UnitCost   float64 `json:"unit_cost"`
	TotalValue float64 `json:"total_value"`
	// CreatedBy is inherited from Base - no need to duplicate
}

func (StockMovement) TableName() string { return "stock_movements" }

type StockAdjustment struct {
	Base
	InventoryItemID string    `gorm:"index;size:36;constraint:OnDelete:CASCADE;" json:"inventory_item_id"`
	AdjustmentDate  time.Time `json:"adjustment_date"`
	AdjustmentType  string    `json:"adjustment_type"`
	Reason          string    `json:"reason"`
	Variance        float64   `json:"variance"`

	SystemQuantity   float64    `json:"system_quantity"`
	PhysicalQuantity float64    `json:"physical_quantity"`
	UnitCost         float64    `json:"unit_cost"`
	ValueImpact      float64    `json:"value_impact"`
	Notes            string     `json:"notes"`
	Status           string     `json:"status"`
	AdjustmentNumber string     `gorm:"uniqueIndex;size:50" json:"adjustment_number"`
	ApprovedBy       string     `json:"approved_by"`
	ApprovedAt       *time.Time `json:"approved_at"`
}

func (StockAdjustment) TableName() string { return "stock_adjustments" }

type Warehouse struct {
	Base
	Code     string `gorm:"uniqueIndex;size:50" json:"code"`
	Name     string `json:"name"`
	IsActive bool   `gorm:"default:true" json:"is_active"`
}

func (Warehouse) TableName() string { return "warehouses" }

type CostingHistory struct {
	Base
	ProductID string  `gorm:"index;size:36" json:"product_id"`
	CostBHD   float64 `json:"cost_bhd"`
}

type CostingLineItemData struct {
	ID                string    `gorm:"primaryKey;type:text" json:"id"`
	CostingSheetID    int       `gorm:"index" json:"costing_sheet_id"`
	ProductNumber     int       `json:"product_number"`
	Equipment         string    `gorm:"type:text" json:"equipment"`
	Model             string    `gorm:"type:text" json:"model"`
	Specification     string    `gorm:"type:text" json:"specification"`
	Supplier          string    `gorm:"type:text" json:"supplier"`
	Quantity          float64   `json:"quantity"`
	FobEUR            float64   `json:"fob_eur"`
	ExchangeRate      float64   `json:"exchange_rate"`
	TotalCostBHD      float64   `json:"total_cost_bhd"`
	MarkupPercent     float64   `json:"markup_percent"`
	SellingPriceBHD   float64   `json:"selling_price_bhd"`
	TotalSuggestedBHD float64   `json:"total_suggested_bhd"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type GradeChange struct {
	Base
	// Composite index for grade change history: WHERE customer_id = ? AND new_grade = ?
	// Used by: Customer grade history, survival intelligence, grade trend analysis
	// High cardinality on CustomerID, low cardinality on NewGrade (A/B/C/D)
	CustomerID string `gorm:"index:idx_grade_change_customer_grade,priority:1;size:36" json:"customer_id"`
	NewGrade   string `gorm:"index:idx_grade_change_customer_grade,priority:2;size:10" json:"new_grade"`
}

type FollowUpTask struct {
	Base
	CustomerID  string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:CustomerID;references:ID" json:"customer_id"`
	Title       string `gorm:"size:255" json:"title"`
	Description string `gorm:"type:varchar(2000)" json:"description"`
	// DueDate indexed for task queue queries: WHERE due_date < NOW(), ORDER BY due_date
	// Used by: Overdue task alerts, calendar view, predictive Butler task scheduling
	DueDate     time.Time  `gorm:"index;autoCreateTime:false" json:"due_date"`
	Status      string     `gorm:"index;size:50;check:status IN ('pending','in_progress','completed','cancelled','overdue')" json:"status"`
	Priority    string     `gorm:"index;size:50;check:priority IN ('low','medium','high','urgent')" json:"priority"`
	Type        string     `gorm:"index;size:50" json:"type"`
	Amount      float64    `gorm:"check:amount >= 0" json:"amount"`
	Contact     string     `gorm:"size:255" json:"contact"`
	Notes       string     `gorm:"type:varchar(2000)" json:"notes"`
	CompletedAt *time.Time `json:"completed_at"`
}

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

func (PurchaseOrder) TableName() string { return "purchase_orders" }

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

	// Wave 9.8 B1: query-time overlay — NOT persisted (gorm:"-"). Populated
	// from ProductMaster.RequiresSerialTracking by getPurchaseOrders /
	// getPurchaseOrderByID (purchase_order_service.go) so the PO receive
	// panel can enforce per-line serial capture without a schema change.
	RequiresSerialTracking bool `gorm:"-" json:"requires_serial_tracking"`
}

func (PurchaseOrderItem) TableName() string { return "purchase_order_items" }

type GoodsReceivedNote struct {
	Base
	// P0 Fix: Changed from CASCADE to RESTRICT - deleting PO shouldn't delete GRN history
	PurchaseOrderID string    `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:PurchaseOrderID;references:ID" json:"purchase_order_id"`
	GRNNumber       string    `gorm:"uniqueIndex;size:50" json:"grn_number"`
	ReceivedDate    time.Time `gorm:"index;autoCreateTime:false" json:"received_date"`
	ReceivedBy      string    `gorm:"size:255" json:"received_by"`

	// Location
	WarehouseID string `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:WarehouseID;references:ID" json:"warehouse_id"`

	// Supplier Delivery Note Reference (P1 Fix)
	SupplierDNNumber string `gorm:"size:100" json:"supplier_dn_number"` // Supplier's delivery note reference

	// Quality Control
	QCStatus string     `gorm:"index;default:'Pending';check:qc_status IN ('Pending','Passed','Failed','Partial')" json:"qc_status"` // P1 Fix: indexed + default
	QCNotes  string     `gorm:"type:varchar(5000)" json:"qc_notes"`
	QCDate   *time.Time `json:"qc_date"`
	QCBy     string     `json:"qc_by"`

	// B3: dedicated persisted completion marker. Set once, inside the same
	// row-locked transaction as CompleteGRN's PO-quantity update, so an
	// all-rejected GRN (which posts no StockMovement — see
	// grnHasPostedMovement in grn_service.go) still has an authoritative,
	// unambiguous "already completed" signal. nil = not completed.
	CompletedAt *time.Time `json:"completed_at"`

	// P1 Fix: Audit fields
	UpdatedBy string `json:"updated_by"`

	Items []GRNItem `gorm:"foreignKey:GRNID" json:"items,omitempty"`
}

func (GoodsReceivedNote) TableName() string { return "goods_received_notes" }

type GRNItem struct {
	Base
	GRNID string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:GRNID;references:ID" json:"grn_id"`
	// P0-5: Added FK constraint to prevent orphaned GRN items
	POItemID         string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:POItemID;references:ID" json:"po_item_id"`
	ProductID        string  `gorm:"index;size:36;constraint:OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:ProductID;references:ID" json:"product_id"`
	QuantityOrdered  float64 `gorm:"not null;default:0;check:quantity_ordered >= 0" json:"quantity_ordered"`
	QuantityReceived float64 `gorm:"not null;default:0;check:quantity_received >= 0" json:"quantity_received"`
	QuantityAccepted float64 `gorm:"not null;default:0;check:quantity_accepted >= 0" json:"quantity_accepted"` // P2 Fix: QuantityReceived - QuantityRejected
	QuantityRejected float64 `gorm:"not null;default:0;check:quantity_rejected >= 0" json:"quantity_rejected"`
	RejectionReason  string  `gorm:"type:varchar(2000)" json:"rejection_reason"`
}

func (GRNItem) TableName() string { return "grn_items" }

func (g *GRNItem) BeforeSave(tx *gorm.DB) error {
	g.QuantityAccepted = g.QuantityReceived - g.QuantityRejected
	return nil
}

type OfferFollowUp struct {
	Base
	OfferID      string     `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:OfferID;references:ID" json:"offer_id"`
	FollowUpDate time.Time  `gorm:"index;autoCreateTime:false" json:"follow_up_date"`
	Notes        string     `gorm:"type:varchar(2000)" json:"notes"`
	Status       string     `gorm:"index;size:20;default:'pending';check:status IN ('pending','completed','cancelled','overdue')" json:"status"` // pending, completed, cancelled
	CompletedAt  *time.Time `json:"completed_at"`
	CompletedBy  string     `gorm:"size:255" json:"completed_by"`
}

type OfferNote struct {
	Base
	OfferID  string    `gorm:"index;size:36" json:"offer_id"`
	NoteDate time.Time `gorm:"index" json:"note_date"`
	Content  string    `gorm:"type:text" json:"content"`
}

func (Offer) TableName() string           { return "offers" }
func (OfferItem) TableName() string       { return "offer_items" }
func (Opportunity) TableName() string     { return "opportunities" }
func (FollowUpTask) TableName() string    { return "followup_tasks" }
func (OfferFollowUp) TableName() string   { return "offer_follow_ups" }
func (OfferNote) TableName() string       { return "offer_notes" }
func (ProductMaster) TableName() string   { return "products" }
func (Order) TableName() string           { return "orders" }
func (OrderItem) TableName() string       { return "order_items" }
func (GradeChange) TableName() string     { return "grade_changes" }
func (CustomerContact) TableName() string { return "customer_contacts" }
func (SupplierContact) TableName() string { return "supplier_contacts" }
func (EntityNote) TableName() string      { return "entity_notes" }
func (SupplierIssue) TableName() string   { return "supplier_issues" }
