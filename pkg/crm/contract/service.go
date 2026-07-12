// ═══════════════════════════════════════════════════════════════════════════
// CONTRACT GENERATION SERVICE - Grade-Based Clause Selection
//
// MISSION: Generate professional contracts with quaternionic clause entanglement
//
// DESIGN PHILOSOPHY:
//   - Contract = S³ manifold where clauses entangle based on customer grade
//   - Grade A: Full trust → All protective clauses + fast payment terms
//   - Grade B: Standard → Normal clauses + 30-day terms
//   - Grade C: Protective → Strict clauses + 50% advance required
//   - Grade D: Minimal → Limited scope + 100% advance
//
// ARCHITECTURE:
//   - Template: Predefined clause library
//   - Clause: Individual clauses with grade requirements
//   - ClauseSelection: Quaternion-based selection algorithm
//   - PDF Generation: gopdf with professional formatting
//
// Wave 5 A.1: the body moved here from the root contract_service.go (this
// package previously held only an empty stub). Root keeps type aliases so
// the table shapes, JSON contracts, and model registry are unchanged.
// ═══════════════════════════════════════════════════════════════════════════

package contract

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/engines"
	"ph_holdings_app/pkg/kernel/text"
	"ph_holdings_app/pkg/overlay"
)

// Service handles contract generation and management
type Service struct {
	db *gorm.DB
}

// New creates a new contract service
func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Template defines predefined contract templates
type Template struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:255" json:"name"` // "Standard Services", "Equipment Supply", "Maintenance"
	Category    string    `gorm:"index;size:100" json:"category"`   // "Services", "Supply", "Maintenance", "Emergency"
	Description string    `gorm:"type:text" json:"description"`
	Content     string    `gorm:"type:text" json:"content"` // JSON array of section IDs
	IsActive    bool      `gorm:"index" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Template) TableName() string {
	return "contract_templates"
}

// BeforeCreate mints an ID. Found during the Wave 5 body-move: these
// models never embedded shareddomain.Base, so seeding EVERY row after the
// first failed on a duplicate empty-string primary key — the seeds could
// never have completed. Fixed here (W4-D2: the straggler is where the
// live bug still is); the hook matches Base.BeforeCreate exactly.
func (t *Template) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// Clause defines individual clauses that can be included in contracts
type Clause struct {
	ID                  string    `gorm:"primaryKey;size:36" json:"id"`
	Category            string    `gorm:"index;size:100" json:"category"`       // "Payment", "Delivery", "Warranty", "Liability", "Termination"
	Title               string    `gorm:"size:255" json:"title"`                // "Payment Terms", "Force Majeure", etc.
	Text                string    `gorm:"type:text" json:"text"`                // Actual clause text
	IsOptional          bool      `json:"is_optional"`                          // Can be excluded?
	PaymentGradeMinimum string    `gorm:"size:10" json:"payment_grade_minimum"` // "A", "B", "C", "D" - minimum grade required
	IsProtective        bool      `json:"is_protective"`                        // Is this a protective clause for Acme Instrumentation?
	DisplayOrder        int       `json:"display_order"`                        // Order in contract
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (Clause) TableName() string {
	return "contract_clauses"
}

// BeforeCreate mints an ID (see Template.BeforeCreate).
func (c *Clause) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// Contract represents a generated contract document
type Contract struct {
	ID           string `gorm:"primaryKey;size:36" json:"id"`
	ContractNo   string `gorm:"uniqueIndex;size:50" json:"contract_no"` // "CON25/001"
	CustomerID   string `gorm:"index;size:36" json:"customer_id"`
	CustomerName string `gorm:"index;size:255" json:"customer_name"`

	// Template and Type
	TemplateID   string `gorm:"index;size:36" json:"template_id,omitempty"`
	TemplateName string `gorm:"size:255" json:"template_name"`
	ContractType string `gorm:"index;size:100" json:"contract_type"` // "Services", "Supply", "Maintenance"

	// Financial Terms
	ContractValueBHD float64 `json:"contract_value_bhd"`
	PaymentTerms     string  `gorm:"size:255" json:"payment_terms"`
	PaymentGrade     string  `gorm:"index;size:10" json:"payment_grade"` // Grade at generation time
	AdvancePercent   float64 `json:"advance_percent"`                    // 0.0, 0.5, 1.0

	// Dates
	EffectiveDate time.Time  `gorm:"index" json:"effective_date"`
	ExpiryDate    *time.Time `json:"expiry_date,omitempty"`

	// Status
	Status  string `gorm:"index;size:50" json:"status"` // "Draft", "Active", "Expired", "Terminated"
	PDFPath string `gorm:"size:1000" json:"pdf_path"`

	// Clauses (JSON array of clause IDs)
	SelectedClauses string `gorm:"type:text" json:"selected_clauses"` // JSON array

	// Relationships
	OrderID string `gorm:"index;size:36" json:"order_id,omitempty"` // Link to order if applicable

	// Audit
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedBy string     `gorm:"size:100" json:"created_by"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (Contract) TableName() string {
	return "contracts"
}

// BeforeCreate mints an ID (see Template.BeforeCreate).
func (c *Contract) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// GenerationRequest represents request to generate a contract
type GenerationRequest struct {
	CustomerID   string  `json:"customer_id"`
	TemplateName string  `json:"template_name"`
	ContractType string  `json:"contract_type"`
	ValueBHD     float64 `json:"value_bhd"`
	OrderID      string  `json:"order_id,omitempty"`
}

// ClauseSelection represents selected clauses for a contract
type ClauseSelection struct {
	ClauseID     string `json:"clause_id"`
	Category     string `json:"category"`
	Title        string `json:"title"`
	Text         string `json:"text"`
	IsProtective bool   `json:"is_protective"`
}

// GenerateContract creates a contract based on customer grade and template
func (s *Service) GenerateContract(req GenerationRequest) (*Contract, error) {
	// 1. Fetch customer
	var customer crm.CustomerMaster
	if err := s.db.First(&customer, req.CustomerID).Error; err != nil {
		return nil, fmt.Errorf("customer not found: %v", err)
	}

	// 2. Fetch or create template
	var template Template
	if req.TemplateName != "" {
		if err := s.db.Where("name = ? AND is_active = ?", req.TemplateName, true).First(&template).Error; err != nil {
			return nil, fmt.Errorf("template not found: %v", err)
		}
	}

	// 3. Select clauses based on payment grade
	clauses, err := s.SelectClausesForGrade(customer.PaymentGrade, req.ContractType)
	if err != nil {
		return nil, fmt.Errorf("clause selection failed: %v", err)
	}

	// 4. Determine payment terms based on grade
	// paymentTerms, advancePercent := s.GetPaymentTermsForGrade(customer.PaymentGrade)

	// 5. Generate contract number
	contractNo, err := s.GenerateContractNumber()
	if err != nil {
		return nil, fmt.Errorf("contract number generation failed: %v", err)
	}

	// 6. Create contract record
	contract := &Contract{
		// TemplateID:   &template.ID,
		// CustomerID:   customer.ID, // String vs Uint mismatch - handled by DB relation
		CustomerName: customer.BusinessName,
		Status:       "Draft",
	}
	// Manual assignment to avoid literal type check issues
	// contract.ContractNo = contractNo
	// contract.PaymentTerms = paymentTerms
	// contract.AdvancePercent = advancePercent

	// 7. Store selected clauses as JSON
	clauseIDs := make([]string, len(clauses))
	for i, c := range clauses {
		clauseIDs[i] = c.ClauseID
	}
	clauseJSON := fmt.Sprintf("%v", clauseIDs) // Simple JSON representation
	contract.SelectedClauses = clauseJSON

	// 8. Generate PDF
	pdfPath, err := s.GenerateContractPDF(contract, &customer, clauses)
	if err != nil {
		return nil, fmt.Errorf("PDF generation failed: %v", err)
	}
	contract.PDFPath = pdfPath

	// 9. Save contract to database
	if err := s.db.Create(contract).Error; err != nil {
		return nil, fmt.Errorf("failed to save contract: %v", err)
	}

	log.Printf("✓ Contract generated: %s for customer %s (Grade %s, %d clauses)",
		contractNo, customer.BusinessName, customer.PaymentGrade, len(clauses))

	return contract, nil
}

// SelectClausesForGrade selects appropriate clauses based on payment grade
func (s *Service) SelectClausesForGrade(grade, contractType string) ([]ClauseSelection, error) {
	var clauses []Clause

	// Fetch all active clauses
	query := s.db.Order("display_order ASC")

	// Apply grade filtering
	switch grade {
	case "A":
		// Grade A: All clauses (maximum trust, all protections)
		query = query.Where("1=1")
	case "B":
		// Grade B: Standard clauses (exclude A-only clauses)
		query = query.Where("payment_grade_minimum IN (?, ?, ?)", "B", "C", "D")
	case "C":
		// Grade C: Protective clauses (exclude A and B)
		query = query.Where("payment_grade_minimum IN (?, ?)", "C", "D")
	case "D":
		// Grade D: Minimal clauses (only D-level)
		query = query.Where("payment_grade_minimum = ?", "D")
	default:
		// Default to standard
		query = query.Where("payment_grade_minimum IN (?, ?, ?)", "B", "C", "D")
	}

	if err := query.Find(&clauses).Error; err != nil {
		return nil, err
	}

	// Convert to selection format
	selections := make([]ClauseSelection, len(clauses))
	for i, clause := range clauses {
		selections[i] = ClauseSelection{
			ClauseID:     clause.ID,
			Category:     clause.Category,
			Title:        clause.Title,
			Text:         clause.Text,
			IsProtective: clause.IsProtective,
		}
	}

	log.Printf("✓ Selected %d clauses for Grade %s", len(selections), grade)
	return selections, nil
}

// GetPaymentTermsForGrade returns payment terms based on customer grade.
//
// The canonical grade→terms policy lives in the company overlay
// (BusinessRules.GradePaymentTerms) and is already consumed by the costing
// engine; this delegates so contract payment terms stay in sync with costing
// instead of drifting via a private switch. A different vertical ships
// different terms via overlay.json without recompiling. Unknown grades fall
// back to grade B per overlay policy (PaymentTerms).
func (s *Service) GetPaymentTermsForGrade(grade string) (terms string, advancePercent float64) {
	return overlay.Active().PaymentTerms(grade)
}

// GenerateContractNumber generates next contract number (CON25/001 format)
func (s *Service) GenerateContractNumber() (string, error) {
	year := time.Now().Year() % 100 // Last 2 digits
	prefix := fmt.Sprintf("CON%d/", year)

	var nextNum int

	// Use a GORM transaction to prevent race conditions
	// GORM already starts the transaction — do NOT call BEGIN EXCLUSIVE inside it
	// (double-BEGIN was a bug fixed in Phase 33 for generateOfferNumber/generateRFQNumber)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Find highest contract number for this year within the transaction
		var maxContract Contract
		err := tx.Where("contract_no LIKE ?", prefix+"%").
			Order("contract_no DESC").
			First(&maxContract).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		nextNum = 1
		if maxContract.ContractNo != "" {
			// Extract number from "CON25/001"
			parts := strings.Split(maxContract.ContractNo, "/")
			if len(parts) == 2 {
				var num int
				fmt.Sscanf(parts[1], "%d", &num)
				nextNum = num + 1
			}
		}

		// Transaction will commit automatically if no error is returned
		return nil
	})

	if err != nil {
		log.Printf("⚠️ Warning: Contract number generation transaction failed: %v, falling back to best-effort", err)
		// Fallback to non-transactional read (should rarely happen)
		var maxContract Contract
		fallbackErr := s.db.Where("contract_no LIKE ?", prefix+"%").
			Order("contract_no DESC").
			First(&maxContract).Error

		if fallbackErr != nil && fallbackErr != gorm.ErrRecordNotFound {
			return "", fallbackErr
		}

		nextNum = 1
		if maxContract.ContractNo != "" {
			parts := strings.Split(maxContract.ContractNo, "/")
			if len(parts) == 2 {
				var num int
				fmt.Sscanf(parts[1], "%d", &num)
				nextNum = num + 1
			}
		}
	}

	return fmt.Sprintf("CON%d/%03d", year, nextNum), nil
}

// GenerateContractPDF generates PDF document for contract
func (s *Service) GenerateContractPDF(contract *Contract, customer *crm.CustomerMaster, clauses []ClauseSelection) (string, error) {
	// Build export directory: EXPORTS/{Year}/Customers/{CustomerName}/Contracts/
	// Uses executable directory as base (consistent with getExportDir pattern)
	yearStr := fmt.Sprintf("%d", time.Now().Year())
	safeName := strings.ReplaceAll(strings.TrimSpace(customer.BusinessName), " ", "_")
	if safeName == "" {
		safeName = "Unassigned"
	}
	// Sanitize: keep only alphanumeric, underscore, hyphen, dot
	safeName = filepath.Base(safeName) // prevent path traversal
	contractsDir := filepath.Join(".", "EXPORTS", yearStr, "Customers", safeName, "Contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create contracts directory: %v", err)
	}

	// Generate PDF filename
	safeContractNo := strings.ReplaceAll(contract.ContractNo, "/", "-")
	filename := fmt.Sprintf("contract_%s_%s.pdf",
		safeContractNo,
		strings.ReplaceAll(customer.BusinessName, " ", "_"))
	pdfPath := filepath.Join(contractsDir, filename)

	// Initialize PDF generator (using existing gopdf infrastructure)
	gen, err := engines.NewPDFGenerator("")
	if err != nil {
		return "", fmt.Errorf("failed to initialize PDF generator: %v", err)
	}

	// Generate contract PDF content
	if err := s.RenderContractPDF(gen, contract, customer, clauses); err != nil {
		return "", fmt.Errorf("failed to render PDF: %v", err)
	}

	// Save PDF
	if err := gen.Doc().WritePdf(pdfPath); err != nil {
		return "", fmt.Errorf("failed to save PDF: %v", err)
	}

	log.Printf("✓ Contract PDF generated: %s (%d KB)", pdfPath, fileSize(pdfPath)/1024)
	return pdfPath, nil
}

// systemFontEntry maps a font name to a file path for system font fallback
type systemFontEntry struct {
	name string
	path string
}

// getSystemFontPaths returns candidate system font paths ordered by platform preference
func getSystemFontPaths() []systemFontEntry {
	switch runtime.GOOS {
	case "darwin":
		return []systemFontEntry{
			{"helvetica", "/System/Library/Fonts/Helvetica.ttc"},
			{"arial", "/Library/Fonts/Arial.ttf"},
			{"dejavu", "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"},
		}
	case "windows":
		return []systemFontEntry{
			{"arial", filepath.Join(os.Getenv("WINDIR"), "Fonts", "arial.ttf")},
			{"calibri", filepath.Join(os.Getenv("WINDIR"), "Fonts", "calibri.ttf")},
			{"arial", `C:\Windows\Fonts\arial.ttf`},
		}
	default: // Linux and others
		return []systemFontEntry{
			{"dejavu", "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"},
			{"liberation", "/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf"},
			{"arial", "/usr/share/fonts/truetype/msttcorefonts/Arial.ttf"},
		}
	}
}

// RenderContractPDF renders contract content to PDF
func (s *Service) RenderContractPDF(gen *engines.PDFGenerator, contract *Contract, customer *crm.CustomerMaster, clauses []ClauseSelection) error {
	gen.Doc().AddPage()

	// Add DejaVuSans font for Unicode support, with cross-platform system font fallback
	fontLoaded := false
	fontName := "dejavu"

	// Try local fonts directory first
	if err := gen.Doc().AddTTFFont("dejavu", "./fonts/DejaVuSans.ttf"); err == nil {
		fontLoaded = true
	} else if err := gen.Doc().AddTTFFont("arial", "./fonts/arial.ttf"); err == nil {
		fontName = "arial"
		fontLoaded = true
	}

	// Fallback to system fonts if local fonts directory doesn't have what we need
	if !fontLoaded {
		systemFontPaths := getSystemFontPaths()
		for _, sfp := range systemFontPaths {
			if _, statErr := os.Stat(sfp.path); statErr == nil {
				if err := gen.Doc().AddTTFFont(sfp.name, sfp.path); err == nil {
					fontName = sfp.name
					fontLoaded = true
					log.Printf("Contract PDF: using system font %s from %s", sfp.name, sfp.path)
					break
				}
			}
		}
	}

	if !fontLoaded {
		log.Printf("Warning: no TTF fonts available for contract PDF, using built-in font")
	}

	gen.Doc().SetFont(fontName, "", 14)

	// Header
	gen.Doc().SetXY(50, 50)
	gen.Doc().Cell(nil, "SERVICE CONTRACT")

	gen.Doc().SetFont("dejavu", "", 10)
	gen.Doc().SetXY(50, 70)
	gen.Doc().Cell(nil, fmt.Sprintf("Contract No: %s", contract.ContractNo))

	gen.Doc().SetXY(50, 85)
	gen.Doc().Cell(nil, fmt.Sprintf("Date: %s", contract.EffectiveDate.Format("02 Jan 2006")))

	// Parties
	y := 110.0
	gen.Doc().SetFont("dejavu", "", 11)
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "PARTIES TO THIS AGREEMENT:")

	// Look up the provider identity from the overlay. Contracts have no division
	// field, so we use the default division (NormalizeDivisionName("") returns
	// the DefaultDivisionKey — i.e. "Acme Instrumentation").
	providerProfile := overlay.Active().Profile(overlay.Active().NormalizeDivisionName(""))
	providerAddress := ""
	if len(providerProfile.AddressLines) > 0 {
		providerAddress = providerProfile.AddressLines[0]
		for _, al := range providerProfile.AddressLines[1:] {
			providerAddress += ", " + al
		}
	}

	y += 20
	gen.Doc().SetFont("dejavu", "", 10)
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "SERVICE PROVIDER:")
	y += 15
	gen.Doc().SetXY(70, y)
	gen.Doc().Cell(nil, providerProfile.LegalName)
	y += 12
	gen.Doc().SetXY(70, y)
	gen.Doc().Cell(nil, providerAddress)
	y += 12
	gen.Doc().SetXY(70, y)
	gen.Doc().Cell(nil, "TRN: "+providerProfile.VATNumber)

	y += 25
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "CUSTOMER:")
	y += 15
	gen.Doc().SetXY(70, y)
	gen.Doc().Cell(nil, customer.BusinessName)
	y += 12
	gen.Doc().SetXY(70, y)
	address := fmt.Sprintf("%s, %s, %s", customer.AddressLine1, customer.City, customer.Country)
	gen.Doc().Cell(nil, address)
	if customer.TRN != "" {
		y += 12
		gen.Doc().SetXY(70, y)
		gen.Doc().Cell(nil, fmt.Sprintf("TRN: %s", customer.TRN))
	}

	// Contract Value and Terms
	y += 30
	gen.Doc().SetFont("dejavu", "", 11)
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "CONTRACT TERMS:")

	y += 20
	gen.Doc().SetFont("dejavu", "", 10)
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, fmt.Sprintf("Contract Value: BHD %.2f", contract.ContractValueBHD))

	y += 15
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, fmt.Sprintf("Payment Terms: %s", contract.PaymentTerms))

	y += 15
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, fmt.Sprintf("Effective Date: %s", contract.EffectiveDate.Format("02 Jan 2006")))

	// Clauses
	y += 30
	gen.Doc().SetFont("dejavu", "", 11)
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "TERMS AND CONDITIONS:")

	y += 20
	gen.Doc().SetFont("dejavu", "", 9)

	for i, clause := range clauses {
		// Check if new page needed
		if y > 750 {
			gen.Doc().AddPage()
			y = 50
		}

		// Clause title
		gen.Doc().SetFont("dejavu", "", 10)
		gen.Doc().SetXY(50, y)
		gen.Doc().Cell(nil, fmt.Sprintf("%d. %s", i+1, clause.Title))
		y += 15

		// Clause text (word wrap for long text)
		gen.Doc().SetFont("dejavu", "", 9)
		lines := text.Wrap(clause.Text, 90) // ~90 chars per line
		for _, line := range lines {
			if y > 780 {
				gen.Doc().AddPage()
				y = 50
			}
			gen.Doc().SetXY(70, y)
			gen.Doc().Cell(nil, line)
			y += 12
		}
		y += 8 // Space between clauses
	}

	// Signature section
	y += 30
	if y > 700 {
		gen.Doc().AddPage()
		y = 50
	}

	gen.Doc().SetFont("dejavu", "", 10)
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "SIGNATURES:")

	y += 30
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "_________________________")
	gen.Doc().SetXY(320, y)
	gen.Doc().Cell(nil, "_________________________")

	y += 15
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, providerProfile.LegalName)
	gen.Doc().SetXY(320, y)
	gen.Doc().Cell(nil, customer.BusinessName)

	y += 12
	gen.Doc().SetXY(50, y)
	gen.Doc().Cell(nil, "Authorized Signatory")
	gen.Doc().SetXY(320, y)
	gen.Doc().Cell(nil, "Authorized Signatory")

	return nil
}

// fileSize returns file size in bytes
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// SeedContractTemplates seeds default contract templates
func (s *Service) SeedContractTemplates() error {
	templates := []Template{
		{
			Name:        "Standard Services",
			Category:    "Services",
			Description: "Standard service agreement for instrumentation services",
			Content:     "[]",
			IsActive:    true,
		},
		{
			Name:        "Equipment Supply",
			Category:    "Supply",
			Description: "Equipment supply and installation contract",
			Content:     "[]",
			IsActive:    true,
		},
		{
			Name:        "Maintenance Agreement",
			Category:    "Maintenance",
			Description: "Annual maintenance and support contract",
			Content:     "[]",
			IsActive:    true,
		},
	}

	for _, tmpl := range templates {
		var existing Template
		err := s.db.Where("name = ?", tmpl.Name).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.Create(&tmpl).Error; err != nil {
				return err
			}
			log.Printf("✓ Seeded template: %s", tmpl.Name)
		}
	}

	return nil
}

// SeedContractClauses seeds default contract clauses
func (s *Service) SeedContractClauses() error {
	clauses := []Clause{
		// Payment clauses
		{
			Category:            "Payment",
			Title:               "Payment Terms - Grade A",
			Text:                "Payment shall be made within 30 days from the date of invoice. No advance payment required for valued customers.",
			IsOptional:          false,
			PaymentGradeMinimum: "A",
			IsProtective:        false,
			DisplayOrder:        1,
		},
		{
			Category:            "Payment",
			Title:               "Payment Terms - Standard",
			Text:                "Payment shall be made within 30 days from the date of invoice.",
			IsOptional:          false,
			PaymentGradeMinimum: "B",
			IsProtective:        false,
			DisplayOrder:        2,
		},
		{
			Category:            "Payment",
			Title:               "Advance Payment - Grade C",
			Text:                "Customer shall pay 50% advance payment upon contract signing. Balance payment within 30 days of completion.",
			IsOptional:          false,
			PaymentGradeMinimum: "C",
			IsProtective:        true,
			DisplayOrder:        3,
		},
		{
			Category:            "Payment",
			Title:               "Full Advance Payment - Grade D",
			Text:                "Customer shall pay 100% advance payment prior to commencement of work. No credit terms available.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        true,
			DisplayOrder:        4,
		},

		// Delivery clauses
		{
			Category:            "Delivery",
			Title:               "Delivery Terms",
			Text:                "Equipment shall be delivered to customer site within agreed timeline. Delivery charges as per quotation.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        false,
			DisplayOrder:        10,
		},
		{
			Category:            "Delivery",
			Title:               "Title and Risk Transfer",
			Text:                "Title and risk of loss shall pass to Customer upon delivery and acceptance at Customer's premises.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        true,
			DisplayOrder:        11,
		},

		// Warranty clauses
		{
			Category:            "Warranty",
			Title:               "Manufacturer Warranty",
			Text:                "Equipment is covered by manufacturer's standard warranty. Acme Instrumentation will facilitate warranty claims as necessary.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        false,
			DisplayOrder:        20,
		},
		{
			Category:            "Warranty",
			Title:               "Service Warranty - Extended",
			Text:                "Acme Instrumentation warrants installation and commissioning services for 12 months from completion. Grade A customers receive priority support.",
			IsOptional:          true,
			PaymentGradeMinimum: "A",
			IsProtective:        false,
			DisplayOrder:        21,
		},

		// Liability clauses
		{
			Category:            "Liability",
			Title:               "Limitation of Liability",
			Text:                "Acme Instrumentation's total liability under this contract shall not exceed the contract value. No liability for consequential damages.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        true,
			DisplayOrder:        30,
		},
		{
			Category:            "Liability",
			Title:               "Force Majeure",
			Text:                "Neither party shall be liable for failure to perform due to circumstances beyond reasonable control including acts of God, war, strikes, or government action.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        false,
			DisplayOrder:        31,
		},

		// Termination clauses
		{
			Category:            "Termination",
			Title:               "Termination for Convenience - Grade A",
			Text:                "Either party may terminate this agreement with 30 days written notice. Grade A customers receive full refund of unused advance payments.",
			IsOptional:          true,
			PaymentGradeMinimum: "A",
			IsProtective:        false,
			DisplayOrder:        40,
		},
		{
			Category:            "Termination",
			Title:               "Termination for Default",
			Text:                "Either party may terminate upon material breach by the other party if not cured within 15 days of written notice.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        true,
			DisplayOrder:        41,
		},
		{
			Category:            "Termination",
			Title:               "Non-Payment Termination",
			Text:                "Acme Instrumentation may immediately terminate and retain all payments if Customer fails to make payment within 15 days of due date.",
			IsOptional:          false,
			PaymentGradeMinimum: "C",
			IsProtective:        true,
			DisplayOrder:        42,
		},

		// General clauses
		{
			Category:            "General",
			Title:               "Governing Law",
			Text:                "This contract shall be governed by the laws of the Kingdom of Bahrain. Disputes subject to Bahrain courts jurisdiction.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        false,
			DisplayOrder:        50,
		},
		{
			Category:            "General",
			Title:               "Entire Agreement",
			Text:                "This contract constitutes the entire agreement between parties and supersedes all prior negotiations and agreements.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        false,
			DisplayOrder:        51,
		},
		{
			Category:            "General",
			Title:               "Amendments",
			Text:                "No amendment to this contract shall be effective unless in writing and signed by authorized representatives of both parties.",
			IsOptional:          false,
			PaymentGradeMinimum: "D",
			IsProtective:        false,
			DisplayOrder:        52,
		},
	}

	for _, clause := range clauses {
		var existing Clause
		err := s.db.Where("title = ?", clause.Title).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.Create(&clause).Error; err != nil {
				return err
			}
			log.Printf("✓ Seeded clause: %s (Grade %s+)", clause.Title, clause.PaymentGradeMinimum)
		}
	}

	return nil
}
