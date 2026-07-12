package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"github.com/xuri/excelize/v2"
	"ph_holdings_app/pkg/approvals"
)

func (a *App) CalculateCosting(req CostingRequest) (*CostingResult, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if len(req.Items) == 0 {
		return nil, fmt.Errorf("no line items provided")
	}

	// Get customer info
	var customer Customer
	if err := a.db.First(&customer, req.CustomerID).Error; err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Get customer grade from payment prediction
	predictor := NewPaymentPredictor(&customer)
	prediction := predictor.Predict(&customer)
	grade := CustomerGrade(prediction.Grade)

	// Initialize costing engine
	engine := NewCostingEngine()

	// Determine customer discount based on grade
	customerDiscount := engine.GetCustomerDiscount(grade)
	if req.ApplyDiscount && req.RequestedDiscount > 0 {
		// Use requested discount if lower than max allowed
		if req.RequestedDiscount <= customerDiscount {
			customerDiscount = req.RequestedDiscount
		}
	} else if !req.ApplyDiscount {
		customerDiscount = 0
	}

	// Get payment terms
	paymentTerms, advanceRequired := engine.GetPaymentTerms(grade)

	// Calculate each line item
	result := &CostingResult{
		CustomerID:      req.CustomerID,
		CustomerName:    customer.BusinessName,
		CustomerGrade:   string(grade),
		OpportunityID:   req.OpportunityID,
		Items:           make([]CostingLineResult, 0, len(req.Items)),
		RiskWarnings:    make([]string, 0),
		PaymentTerms:    paymentTerms,
		AdvanceRequired: advanceRequired,
		ValidUntil:      time.Now().AddDate(0, 0, 30).Format("2006-01-02"),
		CalculatedAt:    time.Now(),
	}

	for _, item := range req.Items {
		// Get margin for product type (or use provided margin)
		margin := item.MarginPercent
		if margin <= 0 {
			margin = engine.GetProductMargin(item.ProductType)
		}

		lineResult := CostingLineResult{
			CostingLineItem: item,
		}

		// Calculate values
		lineResult.TotalCostBHD = item.UnitCostBHD * float64(item.Quantity)
		lineResult.UnitSellBHD = item.UnitCostBHD * (1.0 + margin)
		lineResult.TotalSellBHD = lineResult.UnitSellBHD * float64(item.Quantity)

		// Apply discount
		finalUnit := lineResult.UnitSellBHD * (1.0 - customerDiscount)
		lineResult.UnitProfitBHD = finalUnit - item.UnitCostBHD
		lineResult.TotalProfitBHD = lineResult.UnitProfitBHD * float64(item.Quantity)

		if item.UnitCostBHD > 0 {
			lineResult.ActualMarginPct = lineResult.UnitProfitBHD / item.UnitCostBHD
		}

		result.Items = append(result.Items, lineResult)

		// Accumulate totals
		result.TotalCostBHD += lineResult.TotalCostBHD
		result.TotalSellBHD += lineResult.TotalSellBHD
		result.TotalProfitBHD += lineResult.TotalProfitBHD
	}

	// Calculate discount and final amounts
	result.TotalDiscountBHD = result.TotalSellBHD * customerDiscount
	result.TotalFinalBHD = result.TotalSellBHD - result.TotalDiscountBHD

	// Calculate margins
	if result.TotalCostBHD > 0 {
		result.StandardMarginPct = (result.TotalSellBHD - result.TotalCostBHD) / result.TotalCostBHD
		result.ActualMarginPct = result.TotalProfitBHD / result.TotalCostBHD
	}

	// Assess risk and set approval status
	a.assessCostingRisk(result, &customer, grade)

	log.Printf("✓ Costing calculated for customer %s: Total %.2f BHD, Margin %.1f%%",
		customer.BusinessName, result.TotalFinalBHD, result.ActualMarginPct*100)

	return result, nil
}

// assessCostingRisk performs risk analysis and determines approval status.
// The routing rules stay here (they are trading policy, driven by the
// overlay's BusinessRules); the fold into an approval decision is the
// pkg/approvals engine on kernel vocabulary (Wave 3 B.1). Output fields are
// byte-identical to the pre-engine version — including the historical
// behavior that the final "Requires manager approval" recommendation
// overrides any rule-specific one.
func (a *App) assessCostingRisk(result *CostingResult, customer *Customer, grade CustomerGrade) {
	br := activeOverlay.BusinessRules
	assessment := approvals.NewAssessment(fmt.Sprintf("customer-%d", result.CustomerID), "costing")

	// Margin below the approval threshold (default 20%) needs approval
	if result.ActualMarginPct < br.ApprovalThresholdMargin {
		assessment.Add(approvals.Finding{
			Code:             "margin_below_threshold",
			Message:          fmt.Sprintf("Margin (%.1f%%) below 20%% threshold", result.ActualMarginPct*100),
			RequiresApproval: true,
		})
	}

	// Customer grade
	switch grade {
	case GradeC:
		assessment.Add(approvals.Finding{Code: "grade_c", Message: "Grade C customer - 50% advance required"})
		if !assessment.NeedsApproval() && result.ActualMarginPct < br.ABBCompetitionMinMargin {
			assessment.Add(approvals.Finding{Code: "grade_c_low_margin", RequiresApproval: true})
		}
	case GradeD:
		assessment.Add(approvals.Finding{
			Code:             "grade_d",
			Message:          "HIGH RISK Grade D customer - 100% advance or DECLINE",
			RequiresApproval: true,
		})
	}

	// Very low margin
	if result.ActualMarginPct < br.MinMarginPct {
		assessment.Add(approvals.Finding{
			Code:             "margin_below_minimum",
			Message:          fmt.Sprintf("CRITICAL: Margin (%.1f%%) below minimum 8%%", result.ActualMarginPct*100),
			RequiresApproval: true,
		})
	}

	// Named-competitor competition (default competitor: ABB) — warnings only
	if customer.HasABB == 1 {
		competitor := activeOverlay.CompetitorName()
		assessment.Add(approvals.Finding{Code: "competitor", Message: fmt.Sprintf("%s COMPETING - only proceed if strategically important", competitor)})
		if result.ActualMarginPct < br.ABBCompetitionMinMargin {
			assessment.Add(approvals.Finding{Code: "competitor_low_margin", Message: fmt.Sprintf("⚠ Low margin + %s competition - consider declining", competitor)})
		}
	}

	// Large order (above the configured threshold, default 10K BHD) — warning only
	if result.TotalFinalBHD > br.LargeOrderThresholdBHD {
		assessment.Add(approvals.Finding{Code: "large_order", Message: fmt.Sprintf("Large order (%.2f BHD) - verify customer credit", result.TotalFinalBHD)})
	}

	// Fold into the kernel routing decision and map back to the ViewModel's
	// historical status strings.
	result.RiskWarnings = append(result.RiskWarnings, assessment.Warnings()...)
	result.NeedsApproval = assessment.NeedsApproval()
	if result.NeedsApproval {
		result.ApprovalStatus = "NEEDS_APPROVAL"
	} else {
		result.ApprovalStatus = "AUTO_APPROVED"
	}
	result.RecommendedAction = assessment.Recommendation("Proceed with quotation", "Requires manager approval before proceeding")
}

// ============================================================================
// COSTING SHEET EXPORT API
// ============================================================================

// CostingExportData represents data for exporting costing sheet
type CostingExportData struct {
	// Header
	Division        string `json:"division"`
	Source          string `json:"source"`
	OfferID         string `json:"offerId"`
	OfferNumber     string `json:"offerNumber"`
	Date            string `json:"date"`
	PreparedBy      string `json:"preparedBy"`
	CustomerID      string `json:"customerId"`
	CustomerName    string `json:"customerName"`
	ContactPerson   string `json:"contactPerson"`
	RfqReference    string `json:"rfqReference"`
	FolderNumber    string `json:"folderNumber"`
	CostingId       string `json:"costingId"`
	Subject         string `json:"subject"`
	EstDelivery     string `json:"estDelivery"`
	DeliveryTerms   string `json:"deliveryTerms"`
	PaymentTerms    string `json:"paymentTerms"`
	OrderType       string `json:"orderType"`
	CountryOfOrigin string `json:"countryOfOrigin"`
	CocCoo          string `json:"cocCoo"`          // Certificate of Conformity/Country of Origin
	TestCertificate string `json:"testCertificate"` // Test Certificate requirements
	Installation    string `json:"installation"`    // Installation included?
	Commissioning   string `json:"commissioning"`   // Commissioning included?
	Testing         string `json:"testing"`         // Testing included?
	// Quote type: "Quotation" or "Budgetary Quote"
	QuoteType string `json:"quoteType"`
	// Dynamic VAT rate (percentage, e.g., 10.0)
	VatRate float64 `json:"vatRate"`
	// Hidden charges (BHD amount added to cost)
	HiddenCharges float64 `json:"hiddenCharges"`

	// VAT/Tax compliance (Bahrain NBR)
	PlaceOfSupply string `json:"placeOfSupply"`
	TaxCategory   string `json:"taxCategory"`
	CustomerTRN   string `json:"customerTRN"`
	Body          string `json:"body"`

	// Line Items
	LineItems []CostingExportLineItem `json:"lineItems"`

	// Summary
	Subtotal      float64 `json:"subtotal"`
	Discount      float64 `json:"discount"`
	NetAmount     float64 `json:"netAmount"`
	VAT           float64 `json:"vat"`
	GrandTotal    float64 `json:"grandTotal"`
	TotalCost     float64 `json:"totalCost"`
	Profit        float64 `json:"profit"`
	ProfitPercent float64 `json:"profitPercent"`

	// Linked Opportunity
	OpportunityId       uint   `json:"opportunityId"`
	OpportunityRecordID string `json:"opportunityRecordId"`
	ProjectName         string `json:"projectName"`

	// Terms and Conditions (printed on separate page)
	TermsAndConditions string `json:"termsAndConditions"`

	// Technical datasheets (I-25). AttachmentScopeID binds the exported
	// document to its costing_sheet_attachments rows; Attachments carries the
	// resolved metadata, and any PDF among them is merged onto the end of the
	// exported quotation by appendCostingPDFDatasheets.
	AttachmentScopeID string                          `json:"attachmentScopeId"`
	Attachments       []CostingSheetAttachmentSummary `json:"attachments"`
}

// CostingExportLineItem represents a line item for export
type CostingExportLineItem struct {
	SlNo                int     `json:"slNo"`
	Supplier            string  `json:"supplier"` // Kept for backwards compatibility but not used in UI
	Equipment           string  `json:"equipment"`
	Model               string  `json:"model"`
	SerialNumber        string  `json:"serialNumber"` // Serial number for traceability
	LongCode            string  `json:"longCode"`
	Specification       string  `json:"specification"`
	DetailedDescription string  `json:"detailedDescription"` // NEW: Wide field for instrumentation specs
	Currency            string  `json:"currency"`
	Quantity            int     `json:"quantity"`
	FOB                 float64 `json:"fob"`
	Freight             float64 `json:"freight"`
	FreightPercent      float64 `json:"freightPercent"`
	TotalCost           float64 `json:"totalCost"`
	MarginPercent       float64 `json:"marginPercent"` // NEW: Margin % (not markup)
	MarkupPercent       float64 `json:"markupPercent"`
	SuggestedPrice      float64 `json:"suggestedPrice"`
	TotalPrice          float64 `json:"totalPrice"`
	// Full cost breakdown (for detailed costing persistence)
	ExchangeRate    float64 `json:"exchangeRate"`
	FobBHD          float64 `json:"fobBHD"`
	FreightBHD      float64 `json:"freightBHD"`
	Insurance       float64 `json:"insurance"`
	CustomsPercent  float64 `json:"customsPercent"`
	CustomsBHD      float64 `json:"customsBHD"`
	HandlingPercent float64 `json:"handlingPercent"`
	HandlingBHD     float64 `json:"handlingBHD"`
	FinancePercent  float64 `json:"financePercent"`
	FinanceBHD      float64 `json:"financeBHD"`
	OtherCosts      float64 `json:"otherCosts"`
	UserPrice       float64 `json:"userPrice"`    // User override price
	UserPriceSet    bool    `json:"userPriceSet"` // Was price manually set?
}

type ButlerOfferDraftLineItem struct {
	Description   string  `json:"description"`
	Equipment     string  `json:"equipment"`
	Model         string  `json:"model"`
	Specification string  `json:"specification"`
	Quantity      int     `json:"quantity"`
	UnitPriceBHD  float64 `json:"unit_price_bhd"`
	Optional      bool    `json:"optional"`
}

type ButlerOfferDraftRequest struct {
	Division        string                     `json:"division"`
	PreparedBy      string                     `json:"prepared_by"`
	CustomerID      string                     `json:"customer_id"`
	CustomerName    string                     `json:"customer_name"`
	ContactPerson   string                     `json:"contact_person"`
	RfqReference    string                     `json:"rfq_reference"`
	DeliveryTerms   string                     `json:"delivery_terms"`
	PaymentTerms    string                     `json:"payment_terms"`
	EstDelivery     string                     `json:"est_delivery"`
	CountryOfOrigin string                     `json:"country_of_origin"`
	QuoteType       string                     `json:"quote_type"`
	VatRate         float64                    `json:"vat_rate"`
	LineItems       []ButlerOfferDraftLineItem `json:"line_items"`
}

func (a *App) CreateOfferDraftFromButler(req ButlerOfferDraftRequest) (*Offer, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if strings.TrimSpace(req.CustomerName) == "" && strings.TrimSpace(req.CustomerID) == "" {
		return nil, fmt.Errorf("customer name or customer id is required")
	}
	if len(req.LineItems) == 0 {
		return nil, fmt.Errorf("at least one line item is required")
	}

	// Resolve customer name from ID if provided (avoids "NPC group" LIKE mismatch)
	customerName := strings.TrimSpace(req.CustomerName)
	if strings.TrimSpace(req.CustomerID) != "" {
		var cust CustomerMaster
		if err := a.db.Where("customer_id = ?", strings.TrimSpace(req.CustomerID)).First(&cust).Error; err == nil {
			customerName = cust.BusinessName
		} else if err := a.db.Where("id = ?", strings.TrimSpace(req.CustomerID)).First(&cust).Error; err == nil {
			customerName = cust.BusinessName
		}
	}
	if strings.TrimSpace(customerName) == "" {
		return nil, fmt.Errorf("could not resolve customer")
	}

	vatRate := req.VatRate
	if vatRate == 0 {
		vatRate = 10
	}

	exportData := CostingExportData{
		Division:        firstNonEmptyString(req.Division, activeOverlay.DefaultDivision()),
		Date:            time.Now().Format("2006-01-02"),
		PreparedBy:      firstNonEmptyString(req.PreparedBy, "Butler"),
		CustomerName:    customerName,
		ContactPerson:   req.ContactPerson,
		RfqReference:    req.RfqReference,
		EstDelivery:     req.EstDelivery,
		DeliveryTerms:   req.DeliveryTerms,
		PaymentTerms:    req.PaymentTerms,
		CountryOfOrigin: req.CountryOfOrigin,
		QuoteType:       firstNonEmptyString(req.QuoteType, "Quotation"),
		VatRate:         vatRate,
		OpportunityId:   0,
		LineItems:       make([]CostingExportLineItem, 0, len(req.LineItems)),
	}

	subtotal := 0.0
	for idx, item := range req.LineItems {
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		unitPrice := roundTo3(item.UnitPriceBHD)
		totalPrice := roundTo3(float64(qty) * unitPrice)
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = firstNonEmptyString(item.Equipment, item.Model, "Service line item")
		}

		exportData.LineItems = append(exportData.LineItems, CostingExportLineItem{
			SlNo:           idx + 1,
			Equipment:      description,
			Model:          item.Model,
			Specification:  item.Specification,
			Quantity:       qty,
			SuggestedPrice: unitPrice,
			TotalPrice:     totalPrice,
			TotalCost:      totalPrice,
			MarginPercent:  0,
			Currency:       "BHD",
			UserPrice:      unitPrice,
			UserPriceSet:   true,
		})
		subtotal += totalPrice
	}

	exportData.Subtotal = roundTo3(subtotal)
	exportData.NetAmount = exportData.Subtotal
	exportData.VAT = roundTo3(exportData.Subtotal * exportData.VatRate / 100.0)
	exportData.GrandTotal = roundTo3(exportData.NetAmount + exportData.VAT)
	exportData.TotalCost = exportData.Subtotal
	exportData.Profit = 0
	exportData.ProfitPercent = 0

	return a.SaveCostingAsOffer(exportData)
}

// ButlerCustomerRequest holds parameters for creating a customer via Butler AI
type ButlerCustomerRequest struct {
	BusinessName   string `json:"business_name"`
	CustomerType   string `json:"customer_type"`
	PaymentGrade   string `json:"payment_grade"`
	City           string `json:"city"`
	Country        string `json:"country"`
	PrimaryContact string `json:"primary_contact"`
	PrimaryEmail   string `json:"primary_email"`
	PrimaryPhone   string `json:"primary_phone"`
	MobileNumber   string `json:"mobile_number"`
	Industry       string `json:"industry"`
	AddressLine1   string `json:"address_line1"`
	TRN            string `json:"trn"`
}

// CreateCustomerFromButler creates a new customer record from Butler AI action data
func (a *App) CreateCustomerFromButler(req ButlerCustomerRequest) (*CustomerMaster, error) {
	if err := a.requirePermission("customers:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	businessName := strings.TrimSpace(req.BusinessName)
	if businessName == "" {
		return nil, fmt.Errorf("business name is required")
	}

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = "Bahrain"
	}

	paymentGrade := strings.TrimSpace(req.PaymentGrade)
	if paymentGrade == "" {
		paymentGrade = "B"
	}
	// Validate grade is one of A/B/C/D
	validGrades := map[string]bool{"A": true, "B": true, "C": true, "D": true}
	if !validGrades[strings.ToUpper(paymentGrade)] {
		paymentGrade = "B"
	}
	paymentGrade = strings.ToUpper(paymentGrade)

	customerType := strings.TrimSpace(req.CustomerType)
	if customerType == "" {
		customerType = "Corporate"
	}

	customer := CustomerMaster{
		BusinessName: businessName,
		CustomerType: customerType,
		PaymentGrade: paymentGrade,
		City:         strings.TrimSpace(req.City),
		Country:      country,
		PrimaryEmail: strings.TrimSpace(req.PrimaryEmail),
		PrimaryPhone: strings.TrimSpace(req.PrimaryPhone),
		MobileNumber: strings.TrimSpace(req.MobileNumber),
		Industry:     strings.TrimSpace(req.Industry),
		AddressLine1: strings.TrimSpace(req.AddressLine1),
		TRN:          strings.TrimSpace(req.TRN),
		Status:       "Active",
	}

	// CreateCustomer handles CustomerCode/CustomerID auto-generation and validation
	return a.CreateCustomer(customer)
}

// ButlerSupplierRequest holds parameters for creating a supplier via Butler AI
type ButlerSupplierRequest struct {
	SupplierName   string `json:"supplier_name"`
	SupplierType   string `json:"supplier_type"`
	Country        string `json:"country"`
	PrimaryContact string `json:"primary_contact"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	TaxID          string `json:"tax_id"`
	BrandsHandled  string `json:"brands_handled"`
	LeadTimeDays   int    `json:"lead_time_days"`
}

// CreateSupplierFromButler creates a new supplier record from Butler AI action data
func (a *App) CreateSupplierFromButler(req ButlerSupplierRequest) (*SupplierMaster, error) {
	if err := a.requirePermission("suppliers:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	supplierName := strings.TrimSpace(req.SupplierName)
	if supplierName == "" {
		return nil, fmt.Errorf("supplier name is required")
	}

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = "Bahrain"
	}

	supplierType := strings.TrimSpace(req.SupplierType)
	if supplierType == "" {
		supplierType = "Manufacturer"
	}

	supplier := SupplierMaster{
		SupplierName:   supplierName,
		SupplierType:   supplierType,
		Country:        country,
		PrimaryContact: strings.TrimSpace(req.PrimaryContact),
		Email:          strings.TrimSpace(req.Email),
		Phone:          strings.TrimSpace(req.Phone),
		Address:        strings.TrimSpace(req.Address),
		TaxID:          strings.TrimSpace(req.TaxID),
		BrandsHandled:  strings.TrimSpace(req.BrandsHandled),
		LeadTimeDays:   req.LeadTimeDays,
	}

	// CreateSupplier handles SupplierCode auto-generation and validation
	return a.CreateSupplier(supplier)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func costingUserReference(data CostingExportData) string {
	return limitReferenceRunes(firstNonEmptyString(data.RfqReference, data.FolderNumber, data.CostingId), 100)
}

func roundTo3(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func costingAnnexureSpecForPDF(item CostingExportLineItem) string {
	spec := strings.TrimSpace(item.Specification)
	if spec == "" {
		return ""
	}
	normalizedSpec := normalizeAnnexureMatchKey(strings.Trim(spec, "()| "))
	normalizedLongCode := normalizeAnnexureMatchKey(item.LongCode)
	if normalizedLongCode != "" && normalizedSpec == normalizedLongCode {
		return ""
	}
	if len(spec) < 120 && looksLikeLongOrderCode(strings.Trim(spec, "()| ")) {
		return ""
	}
	return spec
}

// ExportCostingToPDF generates a PDF quotation on Acme Instrumentation letterhead
func (a *App) ExportCostingToPDF(data CostingExportData) (string, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return "", err
	}
	return a.exportCostingToPDF(data, "RFQ")
}

func (a *App) exportCostingToPDF(data CostingExportData, category string) (string, error) {
	if strings.TrimSpace(category) == "" {
		category = "RFQ"
	}
	division := normalizeDivisionName(data.Division)
	// Get letterhead path (shared helper finds it in DB cache, project root, or exe dir)
	letterheadPath := a.gopdfLetterheadPathForDivision(division)

	// Combined filename: {CostingId}_{CustomerName}.pdf
	cleanCostingId := sanitizeFileName(data.CostingId)
	cleanCustomer := sanitizeFileName(data.CustomerName)
	userRef := ""
	if data.RfqReference != "" {
		userRef = sanitizeFileName(data.RfqReference)
	}
	fileName := cleanCostingId
	if fileName == "" {
		fileName = cleanCustomer
	}
	if userRef != "" {
		fileName = fmt.Sprintf("%s_%s", fileName, userRef)
	}
	filePrefix := "Quotation"
	if data.QuoteType != "" && data.QuoteType != "Quotation" {
		filePrefix = sanitizeFileName(data.QuoteType)
	}
	fileName = fmt.Sprintf("%s_%s.pdf", filePrefix, fileName)

	// Save to structured folder: <exports>/<division>/Customers/<CustomerName>/<category>/
	docYear := time.Now().Year()
	outputDir := a.getExportDir("customer", data.CustomerName, category, docYear)
	outputPath := filepath.Join(outputDir, fileName)

	// Create PDF directly with gopdf for clean quotation layout
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4}) // 595 x 842 points
	pdf.AddPage()

	drawCostingLetterhead := func() {
		if letterheadPath == "" {
			return
		}
		if _, err := os.Stat(letterheadPath); err != nil {
			log.Printf("⚠️ Costing letterhead path missing for %s: %v", division, err)
			return
		}
		if err := pdf.Image(letterheadPath, 0, 0, &gopdf.Rect{W: 595, H: 842}); err != nil {
			log.Printf("⚠️ Failed to draw costing letterhead for %s: %v", division, err)
		}
	}

	// Load font - cross-platform font search
	fontCandidates := []string{
		"C:/Windows/Fonts/arial.ttf",
		"C:/Windows/Fonts/calibri.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/Library/Fonts/Arial.ttf",
		"/System/Library/Fonts/Helvetica.ttc",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	}
	fontPath := ""
	for _, candidate := range fontCandidates {
		if _, err := os.Stat(candidate); err == nil {
			fontPath = candidate
			break
		}
	}
	if fontPath == "" {
		return "", fmt.Errorf("no suitable font found for PDF generation")
	}
	if err := pdf.AddTTFFont("main", fontPath); err != nil {
		return "", fmt.Errorf("failed to load font: %v", err)
	}
	// Load bold font - cross-platform
	boldCandidates := []string{
		"C:/Windows/Fonts/arialbd.ttf",
		"C:/Windows/Fonts/calibrib.ttf",
		"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
		"/Library/Fonts/Arial Bold.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	}
	hasBold := false
	for _, candidate := range boldCandidates {
		if _, err := os.Stat(candidate); err == nil {
			if err := pdf.AddTTFFont("mainBold", candidate); err == nil {
				hasBold = true
				break
			}
		}
	}

	// Draw letterhead background
	drawCostingLetterhead()

	// === LAYOUT CONSTANTS (all in points, A4 = 595 x 842) ===
	leftMargin := 50.0
	rightMargin := 545.0
	contentWidth := rightMargin - leftMargin

	// === QUOTATION TITLE ===
	currentY := 90.0
	if hasBold {
		pdf.SetFont("mainBold", "", 16)
	} else {
		pdf.SetFont("main", "", 16)
	}
	quoteTitle := "QUOTATION"
	if data.QuoteType != "" {
		quoteTitle = strings.ToUpper(data.QuoteType)
	}
	pdf.SetXY(220, currentY)
	pdf.Cell(nil, quoteTitle)

	// Quotation number on right
	pdf.SetFont("main", "", 10)
	pdf.SetXY(rightMargin-120, currentY)
	pdf.Cell(nil, "Ref: "+data.CostingId)
	pdf.SetXY(rightMargin-120, currentY+14)
	pdf.Cell(nil, "Date: "+data.Date)

	// === CUSTOMER & PROJECT INFO ===
	currentY = 130.0
	pdf.SetFont("main", "", 9)

	subjectLine := strings.TrimSpace(data.Subject)
	if subjectLine == "" {
		subjectLine = firstNonEmptyString(data.ProjectName, data.RfqReference, data.CustomerName)
	}
	subjectLine = strings.TrimSpace(strings.TrimPrefix(subjectLine, "Sub:"))

	drawColumn := func(labelX, valueX, y float64, label, value string, width int) float64 {
		text := strings.TrimSpace(value)
		if text == "" {
			text = "-"
		}
		lines := wrapText(text, width)
		if len(lines) == 0 {
			lines = []string{"-"}
		}
		if hasBold {
			pdf.SetFont("mainBold", "", 9)
		}
		pdf.SetXY(labelX, y)
		pdf.Cell(nil, label+":")
		pdf.SetFont("main", "", 9)
		lineY := y
		for _, line := range lines {
			pdf.SetXY(valueX, lineY)
			pdf.Cell(nil, line)
			lineY += 10
		}
		return float64(len(lines)) * 10
	}

	infoRows := [][4]string{
		{"Company", data.CustomerName, "Prepared By", strings.TrimSpace(data.PreparedBy)},
		{"Attention", data.ContactPerson, "Division", division},
		{"Subject", subjectLine, "Payment", strings.TrimSpace(data.PaymentTerms)},
		{"RFQ Ref", data.RfqReference, "Delivery", strings.TrimSpace(data.DeliveryTerms)},
		{"Folder No", data.FolderNumber, "Est. Delivery", strings.TrimSpace(data.EstDelivery)},
	}
	infoY := currentY
	for _, row := range infoRows {
		leftHeight := drawColumn(leftMargin, leftMargin+70, infoY, row[0], row[1], 34)
		rightHeight := drawColumn(330, 420, infoY, row[2], row[3], 24)
		rowHeight := leftHeight
		if rightHeight > rowHeight {
			rowHeight = rightHeight
		}
		infoY += rowHeight + 4
	}
	termsY := infoY

	// Body paragraph before the table
	currentY = math.Max(infoY, termsY) + 8
	bodyText := strings.TrimSpace(data.Body)
	if bodyText == "" {
		bodyText = "We thank you for the opportunity and are pleased to submit our techno-commercial offer for your review. Please find our pricing and scope below."
	}
	pdf.SetFont("main", "", 9)
	bodyLines := wrapText(bodyText, 95)
	for _, line := range bodyLines {
		pdf.SetXY(leftMargin, currentY)
		pdf.Cell(nil, line)
		currentY += 12
	}

	// === LINE ITEMS TABLE ===
	currentY += 10

	// Table columns: Sl No | Description | Qty | Unit Price (BHD) | Total (BHD)
	colX := []float64{leftMargin, leftMargin + 30, leftMargin + 280, leftMargin + 345, leftMargin + 420}
	colW := []float64{30, 250, 65, 75, 75}
	headers := []string{"No.", "Description", "Qty", "Unit Price", "Total (BHD)"}

	// Draw table header background
	pdf.SetLineWidth(0.5)
	pdf.SetFillColor(240, 240, 240)
	pdf.RectFromUpperLeftWithStyle(leftMargin, currentY-2, contentWidth, 16, "F")
	pdf.SetFillColor(0, 0, 0) // Reset fill

	// Draw header text
	if hasBold {
		pdf.SetFont("mainBold", "", 8)
	} else {
		pdf.SetFont("main", "", 8)
	}
	for i, header := range headers {
		pdf.SetXY(colX[i]+2, currentY)
		pdf.Cell(nil, header)
	}
	_ = colW // Used for reference

	// Draw header bottom line
	currentY += 16
	pdf.SetLineWidth(0.8)
	pdf.Line(leftMargin, currentY, rightMargin, currentY)

	// Draw line items
	pdf.SetFont("main", "", 8)
	for _, item := range data.LineItems {
		description := strings.TrimSpace(item.Equipment)
		if description == "" {
			description = firstNonEmptyString(item.Model, item.LongCode, "Line item")
		}
		descriptionLines := wrapText(description, 44)
		if len(descriptionLines) == 0 {
			descriptionLines = []string{"Line item"}
		}
		modelLines := []string{}
		if strings.TrimSpace(item.Model) != "" {
			modelLines = wrapText("Model: "+strings.TrimSpace(item.Model), 52)
		}
		rowHeight := math.Max(24, 8+float64(len(descriptionLines))*9+float64(len(modelLines))*8)

		if currentY+rowHeight > 750 {
			// Add new page with letterhead
			pdf.AddPage()
			drawCostingLetterhead()
			currentY = 90.0
		}

		rowY := currentY + 4
		// Sl No
		pdf.SetXY(colX[0]+2, rowY)
		pdf.Cell(nil, fmt.Sprintf("%d", item.SlNo))

		// Description and model, wrapped inside the description column.
		if hasBold {
			pdf.SetFont("mainBold", "", 8)
		}
		descY := rowY
		for _, line := range descriptionLines {
			pdf.SetXY(colX[1]+2, descY)
			pdf.Cell(nil, sanitizeForPDF(line))
			descY += 9
		}
		if len(modelLines) > 0 {
			pdf.SetFont("main", "", 7)
			for _, line := range modelLines {
				pdf.SetXY(colX[1]+2, descY)
				pdf.Cell(nil, sanitizeForPDF(line))
				descY += 8
			}
		}

		// Long Code and Detailed Description are shown ONLY on Annexure page (not page 1)
		pdf.SetFont("main", "", 8)

		// Quantity
		pdf.SetXY(colX[2]+2, rowY)
		pdf.Cell(nil, fmt.Sprintf("%d PC", item.Quantity))

		// Unit Price
		pdf.SetXY(colX[3]+2, rowY)
		pdf.Cell(nil, fmt.Sprintf("%.3f", item.SuggestedPrice))

		// Total
		pdf.SetXY(colX[4]+2, rowY)
		pdf.Cell(nil, fmt.Sprintf("%.3f", item.TotalPrice))

		currentY += rowHeight

		// Detailed description is shown only on Annexure page (not here on page 1)

		// Light separator line between items
		currentY += 4
		pdf.SetLineWidth(0.2)
		pdf.Line(leftMargin, currentY, rightMargin, currentY)
	}

	// Table bottom border
	currentY += 4
	pdf.SetLineWidth(0.8)
	pdf.Line(leftMargin, currentY, rightMargin, currentY)

	// === TOTALS SECTION ===
	currentY += 12
	totalsX := 380.0
	valuesX := 470.0

	pdf.SetFont("main", "", 9)

	// Subtotal
	pdf.SetXY(totalsX, currentY)
	pdf.Cell(nil, "Subtotal:")
	pdf.SetXY(valuesX, currentY)
	pdf.Cell(nil, fmt.Sprintf("%.3f BHD", data.Subtotal))

	// Discount (if any)
	if data.Discount > 0 {
		currentY += 14
		pdf.SetXY(totalsX, currentY)
		pdf.Cell(nil, "Discount:")
		pdf.SetXY(valuesX, currentY)
		pdf.Cell(nil, fmt.Sprintf("-%.3f BHD", data.Discount))
	}

	// Net Amount
	if data.Discount > 0 {
		currentY += 14
		pdf.SetXY(totalsX, currentY)
		pdf.Cell(nil, "Net Amount:")
		pdf.SetXY(valuesX, currentY)
		pdf.Cell(nil, fmt.Sprintf("%.3f BHD", data.NetAmount))
	}

	// VAT
	currentY += 14
	pdf.SetXY(totalsX, currentY)
	displayVatRate := data.VatRate
	if displayVatRate == 0 && data.VAT > 0 {
		displayVatRate = 10.0 // Fallback for old data without VatRate
	}
	vatLabel := fmt.Sprintf("VAT (%.0f%%)", displayVatRate)
	pdf.Cell(nil, vatLabel+":")
	pdf.SetXY(valuesX, currentY)
	pdf.Cell(nil, fmt.Sprintf("%.3f BHD", data.VAT))

	// Grand Total (bold)
	currentY += 18
	if hasBold {
		pdf.SetFont("mainBold", "", 11)
	} else {
		pdf.SetFont("main", "", 11)
	}
	pdf.SetXY(totalsX, currentY)
	pdf.Cell(nil, "Grand Total:")
	pdf.SetXY(valuesX, currentY)
	pdf.Cell(nil, fmt.Sprintf("%.3f BHD", data.GrandTotal))

	// === VALIDITY NOTE ===
	currentY += 30
	pdf.SetFont("main", "", 8)
	pdf.SetXY(leftMargin, currentY)
	pdf.Cell(nil, "This quotation is valid for 30 days from the date of issue.")

	currentY += 12
	pdf.SetXY(leftMargin, currentY)
	pdf.Cell(nil, "Prices are exclusive of VAT unless otherwise stated. Delivery subject to availability.")

	// === TERMS AND CONDITIONS PAGE ===
	// T&C must immediately follow the commercial line items; annexure stays last.
	if data.TermsAndConditions != "" {
		pdf.AddPage()
		drawCostingLetterhead()

		// T&C Title
		tcY := 90.0
		ensureTermsSpace := func(required float64) {
			if tcY+required <= 760 {
				return
			}
			pdf.AddPage()
			drawCostingLetterhead()
			tcY = 90.0
		}
		if hasBold {
			pdf.SetFont("mainBold", "", 14)
		} else {
			pdf.SetFont("main", "", 14)
		}
		pdf.SetXY(leftMargin, tcY)
		pdf.Cell(nil, "TERMS AND CONDITIONS")

		// Reference back to quotation
		pdf.SetFont("main", "", 9)
		pdf.SetXY(rightMargin-120, tcY)
		pdf.Cell(nil, "Ref: "+data.CostingId)

		// T&C Content - Split by newlines first to preserve paragraph structure
		tcY += 30
		pdf.SetFont("main", "", 9)

		// Split T&C by newlines to preserve paragraph breaks
		paragraphs := strings.Split(data.TermsAndConditions, "\n")
		for _, para := range paragraphs {
			para = strings.TrimSpace(para)
			if para == "" {
				ensureTermsSpace(6)
				tcY += 6 // Extra space for blank lines between sections
				continue
			}

			// Check if this is a numbered point header (e.g., "1. QUOTATION VALIDITY")
			isHeader := len(para) > 2 && para[0] >= '1' && para[0] <= '9' && para[1] == '.'
			if isHeader {
				ensureTermsSpace(20)
				tcY += 8 // Extra space before numbered sections
				if hasBold {
					pdf.SetFont("mainBold", "", 9)
				}
			} else {
				pdf.SetFont("main", "", 9)
			}

			// Wrap long paragraphs to fit page width
			wrappedLines := wrapText(para, 95)
			for _, line := range wrappedLines {
				ensureTermsSpace(12)
				pdf.SetXY(leftMargin, tcY)
				pdf.Cell(nil, line)
				tcY += 12
			}

			// Reset to normal font after header
			if isHeader {
				pdf.SetFont("main", "", 9)
			}
		}

		preparedBy := strings.TrimSpace(data.PreparedBy)
		if preparedBy != "" {
			// Full "Best Regards" signature block, resolved from the active
			// overlay's signature list (matched by name/alias, else the
			// company-level fallback stamped with the signer's name). Ordering
			// and spacing mirror the deployed PH quotation footer.
			block := a.resolvePreparedBySignatureBlock(preparedBy)
			signatureLines := []signaturePDFLine{
				{Text: "Best Regards,"},
				{},
				{Text: block.DisplayName, Bold: true},
			}
			for _, value := range []string{block.Title, block.Company} {
				if strings.TrimSpace(value) != "" {
					signatureLines = append(signatureLines, signaturePDFLine{Text: value})
				}
			}
			for _, line := range block.AddressLines {
				if strings.TrimSpace(line) != "" {
					signatureLines = append(signatureLines, signaturePDFLine{Text: line})
				}
			}
			if strings.TrimSpace(block.Mobile) != "" {
				signatureLines = append(signatureLines, signaturePDFLine{Text: "Mob: " + block.Mobile})
			}
			if strings.TrimSpace(block.Office) != "" {
				signatureLines = append(signatureLines, signaturePDFLine{Text: "Office: " + block.Office})
			}
			if strings.TrimSpace(block.Fax) != "" {
				signatureLines = append(signatureLines, signaturePDFLine{Text: "Fax: " + block.Fax})
			}
			if strings.TrimSpace(block.Email) != "" {
				signatureLines = append(signatureLines, signaturePDFLine{Text: block.Email})
			}

			ensureTermsSpace(float64(len(signatureLines))*10 + 22)
			tcY += 18
			for _, line := range signatureLines {
				text := strings.TrimSpace(line.Text)
				if text == "" {
					tcY += 6
					continue
				}
				if line.Bold && hasBold {
					pdf.SetFont("mainBold", "", 9)
				} else {
					pdf.SetFont("main", "", 9)
				}
				pdf.SetXY(leftMargin, tcY)
				pdf.Cell(nil, sanitizeForPDF(text))
				tcY += 10
			}
		}
	}

	// === ANNEXURE PAGE (Detailed Specifications) ===
	hasDetailedSpecs := false
	for _, item := range data.LineItems {
		if item.DetailedDescription != "" || item.LongCode != "" || costingAnnexureSpecForPDF(item) != "" {
			hasDetailedSpecs = true
			break
		}
	}
	if hasDetailedSpecs {
		pdf.AddPage()
		drawCostingLetterhead()
		annexY := 90.0
		if hasBold {
			pdf.SetFont("mainBold", "", 14)
		} else {
			pdf.SetFont("main", "", 14)
		}
		pdf.SetXY(leftMargin, annexY)
		pdf.Cell(nil, "ANNEXURE - 1")
		annexY += 20
		pdf.SetFont("main", "", 9)
		pdf.SetXY(leftMargin, annexY)
		pdf.Cell(nil, "Detailed Order Codes, Specifications and Code Breakdown")
		annexY += 16

		for _, item := range data.LineItems {
			displaySpecification := costingAnnexureSpecForPDF(item)
			if item.DetailedDescription == "" && item.LongCode == "" && displaySpecification == "" {
				continue
			}
			if annexY > 740 {
				pdf.AddPage()
				drawCostingLetterhead()
				annexY = 90.0
			}

			if hasBold {
				pdf.SetFont("mainBold", "", 9)
			}
			pdf.SetXY(leftMargin, annexY)
			pdf.Cell(nil, fmt.Sprintf("%d. %s", item.SlNo, item.Equipment))
			annexY += 12

			pdf.SetFont("main", "", 8)
			if item.Model != "" {
				pdf.SetXY(leftMargin+10, annexY)
				pdf.Cell(nil, "Model: "+item.Model)
				annexY += 10
			}
			if item.LongCode != "" {
				codeLines := wrapText("Long Code: "+item.LongCode, 90)
				for _, cl := range codeLines {
					pdf.SetXY(leftMargin+10, annexY)
					pdf.Cell(nil, cl)
					annexY += 10
				}
			}
			if displaySpecification != "" {
				if hasBold {
					pdf.SetFont("mainBold", "", 8)
					pdf.SetXY(leftMargin+10, annexY)
					pdf.Cell(nil, "Specification:")
					annexY += 10
				}
				pdf.SetFont("main", "", 8)
				specLines := wrapText(displaySpecification, 90)
				for _, sl := range specLines {
					if annexY > 740 {
						pdf.AddPage()
						drawCostingLetterhead()
						annexY = 90.0
					}
					pdf.SetXY(leftMargin+10, annexY)
					pdf.Cell(nil, sl)
					annexY += 10
				}
			}
			if item.DetailedDescription != "" {
				if hasBold {
					pdf.SetFont("mainBold", "", 8)
					pdf.SetXY(leftMargin+10, annexY)
					pdf.Cell(nil, "Code Breakdown / Detailed Description:")
					annexY += 10
				}
				pdf.SetFont("main", "", 8)
				descLines := wrapText(item.DetailedDescription, 90)
				for _, dl := range descLines {
					if annexY > 740 {
						pdf.AddPage()
						drawCostingLetterhead()
						annexY = 90.0
					}
					pdf.SetXY(leftMargin+10, annexY)
					pdf.Cell(nil, dl)
					annexY += 10
				}
			}
			annexY += 8
		}
	}

	// Save PDF
	if err := pdf.WritePdf(outputPath); err != nil {
		return "", fmt.Errorf("failed to write PDF: %v", err)
	}

	// I-25: append any technical PDF datasheets bound to this costing/offer so
	// the delivered quotation is a single self-contained document. Non-fatal —
	// a merge failure preserves the datasheets in a sidecar folder instead.
	if err := a.appendCostingPDFDatasheets(outputPath, data.Attachments); err != nil {
		return "", err
	}

	log.Printf("✅ Generated PDF quotation: %s", outputPath)
	return outputPath, nil
}

// ExportCostingToExcel generates a real XLSX workbook using the PH master template.
func (a *App) ExportCostingToExcel(data CostingExportData) (string, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		return "", err
	}

	cleanCostingId := sanitizeFileName(data.CostingId)
	cleanCustomer := sanitizeFileName(data.CustomerName)
	userRef := ""
	if data.RfqReference != "" {
		userRef = sanitizeFileName(data.RfqReference)
	}

	xlsxFileName := cleanCostingId
	if xlsxFileName == "" {
		xlsxFileName = cleanCustomer
	}
	if userRef != "" {
		xlsxFileName = fmt.Sprintf("%s_%s", xlsxFileName, userRef)
	}
	xlsxFileName = fmt.Sprintf("Costing_%s.xlsx", xlsxFileName)

	paths := a.getAppPaths()
	division := data.Division
	if division == "" {
		division = activeOverlay.DefaultDivision()
	}
	outputDir := a.getExportDir("customer", data.CustomerName, "RFQ", time.Now().Year())
	outputPath := filepath.Join(outputDir, xlsxFileName)

	templatePath := findCostingTemplatePath(paths.ProjectRoot)
	if templatePath == "" {
		log.Printf("⚠️ Costing template not found; generating standalone workbook: %s", outputPath)
		if err := writeStandaloneCostingWorkbook(data, outputPath); err != nil {
			return "", err
		}
		return outputPath, nil
	}

	workbook, err := excelize.OpenFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to open costing template: %v", err)
	}
	defer workbook.Close()

	costingSheet := "Customer Costing"
	customerSheet := "Customer Database"

	contactName := strings.TrimSpace(data.ContactPerson)
	if contactName == "" {
		contactName = strings.TrimSpace(data.CustomerName)
	}

	divisionValue := strings.TrimSpace(data.Division)
	switch divisionValue {
	case "Beacon Controls":
		divisionValue = "Beacon Controls WLL"
	case "", "Acme Instrumentation":
		divisionValue = "Acme Instrumentation WLL"
	}

	supplierCode := "EH"
	if len(data.LineItems) > 0 && strings.TrimSpace(data.LineItems[0].Supplier) != "" {
		supplierCode = strings.TrimSpace(data.LineItems[0].Supplier)
	}

	dateValue := data.Date
	if dateValue == "" {
		dateValue = time.Now().Format("2006-01-02")
	}

	headerValues := map[string]any{
		"B1": divisionValue,
		"B2": dateValue,
		"B3": data.PreparedBy,
		"B4": supplierCode,
		"D1": data.RfqReference,
		"D4": contactName,
		"D5": data.PaymentTerms,
		"F2": data.EstDelivery,
		"F3": data.DeliveryTerms,
		"F4": data.OrderType,
		"F5": data.CountryOfOrigin,
		"H1": data.CocCoo,
		"H2": data.TestCertificate,
		"H3": data.Installation,
		"H4": data.Commissioning,
		"H5": data.Testing,
	}
	for cell, value := range headerValues {
		if err := workbook.SetCellValue(costingSheet, cell, value); err != nil {
			return "", fmt.Errorf("failed to write template header %s: %v", cell, err)
		}
	}

	customerValues := map[string]any{
		"A2": contactName,
		"B2": "Mr.",
		"C2": "",
		"D2": "",
		"E2": "",
		"F2": data.CustomerName,
	}
	for cell, value := range customerValues {
		if err := workbook.SetCellValue(customerSheet, cell, value); err != nil {
			return "", fmt.Errorf("failed to write customer lookup %s: %v", cell, err)
		}
	}

	for i := 0; i < 12; i++ {
		colName, err := excelize.ColumnNumberToName(3 + i) // C:N
		if err != nil {
			return "", fmt.Errorf("failed to resolve template column: %v", err)
		}
		for _, cell := range []string{
			colName + "7",
			colName + "8",
			colName + "9",
			colName + "10",
			colName + "11",
			colName + "12",
			colName + "14",
			colName + "17",
			colName + "30",
			colName + "31",
		} {
			if err := workbook.SetCellValue(costingSheet, cell, ""); err != nil {
				return "", fmt.Errorf("failed to clear template cell %s: %v", cell, err)
			}
		}
	}

	for i, item := range data.LineItems {
		if i >= 12 {
			break
		}

		colName, err := excelize.ColumnNumberToName(3 + i) // C:N
		if err != nil {
			return "", fmt.Errorf("failed to resolve template item column: %v", err)
		}

		modelText := strings.TrimSpace(item.Model)
		if modelText != "" {
			modelText = "Model no.: " + modelText
		}

		itemSupplier := strings.TrimSpace(item.Supplier)
		if itemSupplier == "" {
			itemSupplier = supplierCode
		}

		itemValues := map[string]any{
			colName + "7":  itemSupplier,
			colName + "8":  strings.TrimSpace(item.Equipment),
			colName + "9":  modelText,
			colName + "10": buildCostingSpecification(item),
			colName + "11": item.Quantity,
			colName + "12": item.FOB,
			colName + "14": item.Freight,
			colName + "17": item.ExchangeRate,
			colName + "30": item.SuggestedPrice,
			colName + "31": item.TotalPrice,
		}
		for cell, value := range itemValues {
			if err := workbook.SetCellValue(costingSheet, cell, value); err != nil {
				return "", fmt.Errorf("failed to write template item %s: %v", cell, err)
			}
		}

		if i == 0 {
			currency := strings.TrimSpace(item.Currency)
			if currency == "" {
				currency = "USD"
			}
			if err := workbook.SetCellValue(costingSheet, "A12", currency); err != nil {
				return "", fmt.Errorf("failed to write template currency: %v", err)
			}
		}
	}

	summaryValues := map[string]any{
		"C35": data.Subtotal,
		"D35": data.VAT,
		"E35": data.GrandTotal,
		"C36": data.TotalCost,
		"C37": data.Profit,
		"C38": data.ProfitPercent / 100,
	}
	for cell, value := range summaryValues {
		if err := workbook.SetCellValue(costingSheet, cell, value); err != nil {
			return "", fmt.Errorf("failed to write template summary %s: %v", cell, err)
		}
	}

	if err := appendDetailedCostingWorkbookSheet(workbook, data); err != nil {
		return "", err
	}

	if err := workbook.SaveAs(outputPath); err != nil {
		return "", fmt.Errorf("failed to write XLSX file: %v", err)
	}

	log.Printf("✅ Generated Excel costing sheet from template: %s", outputPath)
	return outputPath, nil
}

func appendDetailedCostingWorkbookSheet(workbook *excelize.File, data CostingExportData) error {
	sheet := "Detailed Costing"
	if sheetIndex, err := workbook.GetSheetIndex(sheet); err == nil && sheetIndex >= 0 {
		_ = workbook.DeleteSheet(sheet)
	}

	index, err := workbook.NewSheet(sheet)
	if err != nil {
		return fmt.Errorf("failed to create detailed costing sheet: %v", err)
	}
	workbook.SetActiveSheet(index)

	titleStyle, _ := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "1D1D1F"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	sectionStyle, _ := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"334155"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	labelStyle, _ := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "475569"},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "E2E8F0", Style: 1}, {Type: "right", Color: "E2E8F0", Style: 1}, {Type: "top", Color: "E2E8F0", Style: 1}, {Type: "bottom", Color: "E2E8F0", Style: 1}},
	})
	cellStyle, _ := workbook.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "E2E8F0", Style: 1}, {Type: "right", Color: "E2E8F0", Style: 1}, {Type: "top", Color: "E2E8F0", Style: 1}, {Type: "bottom", Color: "E2E8F0", Style: 1}},
	})
	headerStyle, _ := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1D1D1F"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "CBD5E1", Style: 1}, {Type: "right", Color: "CBD5E1", Style: 1}, {Type: "top", Color: "CBD5E1", Style: 1}, {Type: "bottom", Color: "CBD5E1", Style: 1}},
	})
	moneyStyle, _ := workbook.NewStyle(&excelize.Style{
		NumFmt:    40,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "top"},
		Border:    []excelize.Border{{Type: "left", Color: "E2E8F0", Style: 1}, {Type: "right", Color: "E2E8F0", Style: 1}, {Type: "top", Color: "E2E8F0", Style: 1}, {Type: "bottom", Color: "E2E8F0", Style: 1}},
	})
	numberStyle, _ := workbook.NewStyle(&excelize.Style{
		NumFmt:    4,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "top"},
		Border:    []excelize.Border{{Type: "left", Color: "E2E8F0", Style: 1}, {Type: "right", Color: "E2E8F0", Style: 1}, {Type: "top", Color: "E2E8F0", Style: 1}, {Type: "bottom", Color: "E2E8F0", Style: 1}},
	})

	lastColumn := "AD"
	_ = workbook.MergeCell(sheet, "A1", lastColumn+"1")
	_ = workbook.SetCellValue(sheet, "A1", firstNonEmptyString(data.QuoteType, "Quotation")+" - Complete Costing Export")
	_ = workbook.SetCellStyle(sheet, "A1", lastColumn+"1", titleStyle)
	_ = workbook.SetRowHeight(sheet, 1, 26)

	_ = workbook.MergeCell(sheet, "A3", lastColumn+"3")
	_ = workbook.SetCellValue(sheet, "A3", "Header and Commercial Terms")
	_ = workbook.SetCellStyle(sheet, "A3", lastColumn+"3", sectionStyle)

	headerRows := [][]any{
		{"Division", data.Division, "Date", data.Date, "Prepared By", data.PreparedBy, "Quote Type", data.QuoteType},
		{"Customer", data.CustomerName, "Customer ID", data.CustomerID, "Contact Person", data.ContactPerson, "Customer TRN", data.CustomerTRN},
		{"RFQ Reference", data.RfqReference, "Folder No.", data.FolderNumber, "Costing ID", data.CostingId, "Subject", data.Subject},
		{"Payment Terms", data.PaymentTerms, "Delivery Terms", data.DeliveryTerms, "Estimated Delivery", data.EstDelivery, "Order Type", data.OrderType},
		{"Country of Origin", data.CountryOfOrigin, "COC/COO", data.CocCoo, "Test Certificate", data.TestCertificate, "Place of Supply", data.PlaceOfSupply},
		{"Installation", data.Installation, "Commissioning", data.Commissioning, "Testing", data.Testing, "Tax Category", data.TaxCategory},
		{"Source Offer", data.OfferNumber, "Source Offer ID", data.OfferID, "Linked Opportunity ID", data.OpportunityId, "Project", data.ProjectName},
	}

	for i, row := range headerRows {
		excelRow := i + 4
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, excelRow)
			_ = workbook.SetCellValue(sheet, cell, value)
			if j%2 == 0 {
				_ = workbook.SetCellStyle(sheet, cell, cell, labelStyle)
			} else {
				_ = workbook.SetCellStyle(sheet, cell, cell, cellStyle)
			}
		}
	}

	bodyRow := 12
	_ = workbook.MergeCell(sheet, fmt.Sprintf("A%d", bodyRow), fmt.Sprintf("%s%d", lastColumn, bodyRow))
	_ = workbook.SetCellValue(sheet, fmt.Sprintf("A%d", bodyRow), "PDF Body")
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", bodyRow), fmt.Sprintf("%s%d", lastColumn, bodyRow), sectionStyle)
	_ = workbook.MergeCell(sheet, fmt.Sprintf("A%d", bodyRow+1), fmt.Sprintf("%s%d", lastColumn, bodyRow+2))
	_ = workbook.SetCellValue(sheet, fmt.Sprintf("A%d", bodyRow+1), data.Body)
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", bodyRow+1), fmt.Sprintf("%s%d", lastColumn, bodyRow+2), cellStyle)
	_ = workbook.SetRowHeight(sheet, bodyRow+1, 48)

	tableStart := 16
	headers := []string{
		"Sl No", "Supplier", "Equipment", "Model", "Long Code", "Specification", "Detailed Description",
		"Currency", "Qty", "Exchange Rate", "Unit Price FOB", "FOB BHD", "Freight %", "Freight Foreign",
		"Freight BHD", "Insurance", "Customs %", "Customs BHD", "Handling %", "Handling BHD",
		"Finance %", "Finance BHD", "Extra Cost", "Unit PH Cost BHD", "Total PH Cost BHD",
		"Markup %", "Suggested Unit Price", "Manual Unit Price", "Manual Price Used", "Line Total BHD",
	}
	for i, value := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, tableStart)
		_ = workbook.SetCellValue(sheet, cell, value)
	}
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", tableStart), fmt.Sprintf("%s%d", lastColumn, tableStart), headerStyle)
	_ = workbook.SetRowHeight(sheet, tableStart, 34)

	for i, item := range data.LineItems {
		row := tableStart + 1 + i
		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		manualUnitPrice := any("")
		if item.UserPriceSet {
			manualUnitPrice = item.UserPrice
		}
		manualUsed := "No"
		if item.UserPriceSet {
			manualUsed = "Yes"
		}
		values := []any{
			item.SlNo,
			item.Supplier,
			item.Equipment,
			item.Model,
			item.LongCode,
			item.Specification,
			item.DetailedDescription,
			item.Currency,
			quantity,
			item.ExchangeRate,
			item.FOB,
			item.FobBHD,
			item.FreightPercent,
			item.Freight,
			item.FreightBHD,
			item.Insurance,
			item.CustomsPercent,
			item.CustomsBHD,
			item.HandlingPercent,
			item.HandlingBHD,
			item.FinancePercent,
			item.FinanceBHD,
			item.OtherCosts,
			item.TotalCost,
			item.TotalCost * float64(quantity),
			item.MarkupPercent,
			item.SuggestedPrice,
			manualUnitPrice,
			manualUsed,
			item.TotalPrice,
		}
		for j, value := range values {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			_ = workbook.SetCellValue(sheet, cell, value)
		}
		_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastColumn, row), cellStyle)
		for _, col := range []string{"J", "M", "Q", "S", "U", "Z"} {
			_ = workbook.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), numberStyle)
		}
		for _, col := range []string{"K", "L", "N", "O", "P", "R", "T", "V", "W", "X", "Y", "AA", "AB", "AD"} {
			_ = workbook.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), moneyStyle)
		}
		if strings.TrimSpace(item.DetailedDescription) != "" || strings.TrimSpace(item.Specification) != "" {
			_ = workbook.SetRowHeight(sheet, row, 54)
		}
	}

	lastItemRow := tableStart + len(data.LineItems)
	if len(data.LineItems) == 0 {
		lastItemRow = tableStart
	}
	summaryStart := lastItemRow + 3
	_ = workbook.MergeCell(sheet, fmt.Sprintf("A%d", summaryStart), fmt.Sprintf("H%d", summaryStart))
	_ = workbook.SetCellValue(sheet, fmt.Sprintf("A%d", summaryStart), "Summary")
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", summaryStart), fmt.Sprintf("H%d", summaryStart), sectionStyle)

	summaryRows := [][]any{
		{"Subtotal", data.Subtotal, "Discount", data.Discount, "Net Amount", data.NetAmount, "Hidden Charges", data.HiddenCharges},
		{"VAT Rate %", data.VatRate, "VAT Amount", data.VAT, "Grand Total", data.GrandTotal, "Total PH Cost", data.TotalCost},
		{"Profit", data.Profit, "Profit %", data.ProfitPercent, "Line Items", len(data.LineItems), "Currency", "BHD"},
	}
	for i, row := range summaryRows {
		excelRow := summaryStart + 1 + i
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, excelRow)
			_ = workbook.SetCellValue(sheet, cell, value)
			if j%2 == 0 {
				_ = workbook.SetCellStyle(sheet, cell, cell, labelStyle)
			} else {
				_ = workbook.SetCellStyle(sheet, cell, cell, cellStyle)
			}
		}
	}
	for _, col := range []string{"B", "D", "F", "H"} {
		_ = workbook.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, summaryStart+1), fmt.Sprintf("%s%d", col, summaryStart+3), moneyStyle)
	}
	for _, cell := range []string{
		fmt.Sprintf("B%d", summaryStart+2),
		fmt.Sprintf("D%d", summaryStart+3),
		fmt.Sprintf("F%d", summaryStart+3),
		fmt.Sprintf("H%d", summaryStart+3),
	} {
		_ = workbook.SetCellStyle(sheet, cell, cell, cellStyle)
	}

	termsStart := summaryStart + 6
	_ = workbook.MergeCell(sheet, fmt.Sprintf("A%d", termsStart), fmt.Sprintf("%s%d", lastColumn, termsStart))
	_ = workbook.SetCellValue(sheet, fmt.Sprintf("A%d", termsStart), "Terms and Conditions")
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", termsStart), fmt.Sprintf("%s%d", lastColumn, termsStart), sectionStyle)
	_ = workbook.MergeCell(sheet, fmt.Sprintf("A%d", termsStart+1), fmt.Sprintf("%s%d", lastColumn, termsStart+4))
	_ = workbook.SetCellValue(sheet, fmt.Sprintf("A%d", termsStart+1), data.TermsAndConditions)
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", termsStart+1), fmt.Sprintf("%s%d", lastColumn, termsStart+4), cellStyle)
	_ = workbook.SetRowHeight(sheet, termsStart+1, 120)

	_ = workbook.SetColWidth(sheet, "A", "B", 10)
	_ = workbook.SetColWidth(sheet, "C", "C", 28)
	_ = workbook.SetColWidth(sheet, "D", "E", 20)
	_ = workbook.SetColWidth(sheet, "F", "G", 38)
	_ = workbook.SetColWidth(sheet, "H", "I", 12)
	_ = workbook.SetColWidth(sheet, "J", "AD", 15)
	_ = workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: tableStart, TopLeftCell: "A17", ActivePane: "bottomLeft"})

	return nil
}

func writeStandaloneCostingWorkbook(data CostingExportData, outputPath string) error {
	workbook := excelize.NewFile()
	defer workbook.Close()

	sheet := "Costing"
	index, err := workbook.NewSheet(sheet)
	if err != nil {
		return fmt.Errorf("failed to create costing workbook: %v", err)
	}
	workbook.SetActiveSheet(index)
	_ = workbook.DeleteSheet("Sheet1")

	titleStyle, _ := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "1D1D1F"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	headerStyle, _ := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1D1D1F"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "D9D9D9", Style: 1}, {Type: "right", Color: "D9D9D9", Style: 1}, {Type: "top", Color: "D9D9D9", Style: 1}, {Type: "bottom", Color: "D9D9D9", Style: 1}},
	})
	cellStyle, _ := workbook.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "E5E5E5", Style: 1}, {Type: "right", Color: "E5E5E5", Style: 1}, {Type: "top", Color: "E5E5E5", Style: 1}, {Type: "bottom", Color: "E5E5E5", Style: 1}},
	})
	moneyStyle, _ := workbook.NewStyle(&excelize.Style{
		NumFmt:    40,
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "top"},
		Border:    []excelize.Border{{Type: "left", Color: "E5E5E5", Style: 1}, {Type: "right", Color: "E5E5E5", Style: 1}, {Type: "top", Color: "E5E5E5", Style: 1}, {Type: "bottom", Color: "E5E5E5", Style: 1}},
	})

	_ = workbook.MergeCell(sheet, "A1", "J1")
	_ = workbook.SetCellValue(sheet, "A1", firstNonEmptyString(data.QuoteType, "Quotation")+" Costing Sheet")
	_ = workbook.SetCellStyle(sheet, "A1", "J1", titleStyle)

	headerRows := [][]any{
		{"Division", data.Division, "Date", data.Date, "Prepared By", data.PreparedBy},
		{"Customer", data.CustomerName, "Contact", data.ContactPerson, "RFQ Ref", data.RfqReference},
		{"Delivery", data.DeliveryTerms, "Payment", data.PaymentTerms, "Est. Delivery", data.EstDelivery},
		{"Country", data.CountryOfOrigin, "VAT Rate", data.VatRate, "Subject", data.Subject},
	}
	for i, row := range headerRows {
		excelRow := i + 3
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, excelRow)
			_ = workbook.SetCellValue(sheet, cell, value)
		}
	}

	tableStart := 9
	headers := []any{"Sl No", "Equipment", "Model", "Specification", "Currency", "Qty", "FOB", "Freight", "Unit Price", "Total"}
	for i, value := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, tableStart)
		_ = workbook.SetCellValue(sheet, cell, value)
	}
	_ = workbook.SetCellStyle(sheet, "A9", "J9", headerStyle)

	for i, item := range data.LineItems {
		row := tableStart + 1 + i
		values := []any{
			item.SlNo,
			item.Equipment,
			item.Model,
			buildCostingSpecification(item),
			item.Currency,
			item.Quantity,
			item.FOB,
			item.Freight,
			item.SuggestedPrice,
			item.TotalPrice,
		}
		for j, value := range values {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			_ = workbook.SetCellValue(sheet, cell, value)
		}
	}

	lastRow := tableStart + len(data.LineItems)
	if lastRow >= tableStart+1 {
		_ = workbook.SetCellStyle(sheet, fmt.Sprintf("A%d", tableStart+1), fmt.Sprintf("J%d", lastRow), cellStyle)
		_ = workbook.SetCellStyle(sheet, fmt.Sprintf("G%d", tableStart+1), fmt.Sprintf("J%d", lastRow), moneyStyle)
	}

	summaryRow := lastRow + 3
	summaryValues := [][]any{
		{"Subtotal", data.Subtotal},
		{"Discount", data.Discount},
		{"Net Amount", data.NetAmount},
		{fmt.Sprintf("VAT (%.0f%%)", data.VatRate), data.VAT},
		{"Grand Total", data.GrandTotal},
		{"Total Cost", data.TotalCost},
		{"Profit", data.Profit},
		{"Profit %", data.ProfitPercent},
	}
	for i, row := range summaryValues {
		_ = workbook.SetCellValue(sheet, fmt.Sprintf("I%d", summaryRow+i), row[0])
		_ = workbook.SetCellValue(sheet, fmt.Sprintf("J%d", summaryRow+i), row[1])
	}
	_ = workbook.SetCellStyle(sheet, fmt.Sprintf("J%d", summaryRow), fmt.Sprintf("J%d", summaryRow+len(summaryValues)-1), moneyStyle)

	_ = workbook.SetColWidth(sheet, "A", "A", 8)
	_ = workbook.SetColWidth(sheet, "B", "D", 28)
	_ = workbook.SetColWidth(sheet, "E", "F", 12)
	_ = workbook.SetColWidth(sheet, "G", "J", 15)
	_ = workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: 9, TopLeftCell: "A10", ActivePane: "bottomLeft"})

	if strings.TrimSpace(data.TermsAndConditions) != "" {
		termsSheet := "Terms"
		if _, err := workbook.NewSheet(termsSheet); err == nil {
			_ = workbook.SetCellValue(termsSheet, "A1", "Terms and Conditions")
			_ = workbook.SetCellStyle(termsSheet, "A1", "A1", titleStyle)
			_ = workbook.SetCellValue(termsSheet, "A3", data.TermsAndConditions)
			_ = workbook.SetColWidth(termsSheet, "A", "A", 100)
			_ = workbook.SetRowHeight(termsSheet, 3, 240)
		}
	}

	if err := appendDetailedCostingWorkbookSheet(workbook, data); err != nil {
		return err
	}

	if err := workbook.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to write standalone XLSX file: %v", err)
	}
	log.Printf("✅ Generated standalone Excel costing sheet: %s", outputPath)
	return nil
}

// sanitizeFileName removes invalid characters from filename
// getExportDir returns the canonical export directory for generated files.
// Client builds write to a shallow user-facing folder, while still grouping
// customer-facing documents by customer:
// Documents/AsymmFlow Exports/{Customer}/{Category}/
// Documents/AsymmFlow Exports/Suppliers/{Supplier}/{Category}/
//
// If the user's Documents folder cannot be resolved, it falls back to the
// application data root.
//
// Category mapping:
//
//	Customer: RFQ, Quotation, Order, MISC
//	Supplier: Offers, Orders, MISC
func (a *App) getExportDir(entityType, entityName, category string, docYear int) string {
	paths := a.getAppPaths()
	if paths == nil {
		return filepath.Join(".", "EXPORTS")
	}
	root := filepath.Join(paths.ProjectRoot, "EXPORTS")
	if homeDir, err := os.UserHomeDir(); err == nil && strings.TrimSpace(homeDir) != "" {
		root = filepath.Join(homeDir, "Documents", "AsymmFlow Exports")
	}
	safeCategory := sanitizeFileName(category)
	if safeCategory == "" || safeCategory == "unnamed" {
		safeCategory = "General"
	}
	safeEntity := sanitizeFileName(entityName)

	var dir string
	switch strings.ToLower(entityType) {
	case "customer":
		if safeEntity == "" || safeEntity == "unnamed" {
			safeEntity = "Unassigned_Customer"
		}
		dir = filepath.Join(root, safeEntity, safeCategory)
	case "supplier":
		if safeEntity == "" || safeEntity == "unnamed" {
			safeEntity = "Unassigned_Supplier"
		}
		dir = filepath.Join(root, "Suppliers", safeEntity, safeCategory)
	default:
		dir = filepath.Join(root, "Reports")
	}
	os.MkdirAll(dir, 0755)
	return dir
}

func sanitizeFileName(name string) string {
	// Strip path traversal sequences first
	result := strings.ReplaceAll(name, "..", "")
	// Replace invalid characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove leading/trailing dots and underscores
	result = strings.Trim(result, "._")
	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}
	if result == "" {
		result = "unnamed"
	}
	return result
}

// costingTemplateDownloadsCandidate resolves the user's Downloads copy of the
// costing template. os.UserHomeDir works on every platform; $HOME is empty on
// Windows (3-PLAT).
func costingTemplateDownloadsCandidate() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "Downloads", "Acme Instrumentation Costing MasterFile.xlsx")
}

func findCostingTemplatePath(projectRoot string) string {
	candidates := []string{
		filepath.Join(projectRoot, "Acme Instrumentation Costing MasterFile.xlsx"),
		filepath.Join(projectRoot, "assets", "Acme Instrumentation Costing MasterFile.xlsx"),
		filepath.Join(projectRoot, "templates", "Acme Instrumentation Costing MasterFile.xlsx"),
		costingTemplateDownloadsCandidate(),
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func normalizeMultilineText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func buildCostingSpecification(item CostingExportLineItem) string {
	var parts []string

	if item.LongCode != "" {
		parts = append(parts, "Order Code: "+normalizeMultilineText(item.LongCode))
	}
	if item.Specification != "" {
		parts = append(parts, normalizeMultilineText(item.Specification))
	}
	if item.DetailedDescription != "" {
		parts = append(parts, normalizeMultilineText(item.DetailedDescription))
	}

	return strings.Join(parts, "\n")
}

// OpenExportedFile opens the exported file in the default application
// SECURITY: Validates file path to prevent command injection
func (a *App) OpenExportedFile(filePath string) error {
	// Mission I (I-11): launches an OS handler on a caller-supplied path — gated.
	if err := a.requirePermission("reports:export"); err != nil {
		return err
	}
	// SECURITY FIX: Validate file path to prevent command injection
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Verify file exists before attempting to open
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Use validated absolute path with proper command syntax
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		// Use rundll32 instead of cmd /c start for better security
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", absPath)
	case "darwin":
		// Use -- to prevent filenames starting with - from being interpreted as flags
		cmd = exec.Command("open", "--", absPath)
	default:
		// Linux/Unix: use -- for xdg-open as well
		cmd = exec.Command("xdg-open", "--", absPath)
	}
	suppressCommandWindow(cmd)

	return cmd.Start()
}

// ============================================================================
// ACCOUNTING & FINANCE API (Phase 3)
// ============================================================================

// GetChartOfAccounts retrieves all accounts or filtered by type
