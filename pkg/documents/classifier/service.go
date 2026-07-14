package classifier

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"ph_holdings_app/pkg/overlay"
)

// DOCUMENT TYPE ROUTING CLASSIFIER
// ========================================================================
//
// Wave 2 Agent 3 - Document Classification & Routing
//
// Two-stage classification:
// 1. Keyword detection (fast) - looks for explicit document type markers
// 2. Fallback heuristic - uses text patterns when no keywords found
//
// SSOT Mapping (from ASYMMETRICA_MATHEMATICAL_STANDARD.md):
// - Invoice       → Oblate Spheroid (flattened, stable)
// - RFQ/Tender    → Icosahedron (20 faces = complex requirements)
// - Contract      → S³ (3-sphere = binding agreement in 4D space)
// - OCR Document  → Banach Ball (unit ball in functional space)
// - Quotation     → Torus (circular exchange proposal)
// - PO            → Cube (6 faces = structured order)
//
// Built: December 22, 2025 - Wave 2 Execution
// ========================================================================

// ========================================================================
// FILESYSTEM-BASED DOCUMENT CLASSIFIER
// ========================================================================
//
// Classifies documents based on file paths, folder structure, and filenames
// Used for organizing documents in OneDrive/local folders
//
// Document Types:
// - INVOICE, DELIVERY_NOTE, CUSTOMER_PO, SUPPLIER_PO_ACK, INTERNAL_PO
// - RFQ_EMAIL, RFQ_DOCUMENT, COSTING_SHEET, COMMERCIAL_OFFER
// - SHIPPING_DOC, TECHNICAL_DOC, OTHER
//
// Added: 2026-02-05
// ========================================================================

// FilesystemDocClassification represents a classified document with extracted metadata
type FilesystemDocClassification struct {
	FilePath     string    `json:"file_path"`
	FileName     string    `json:"file_name"`
	DocumentType string    `json:"document_type"`
	OfferNumber  string    `json:"offer_number"`
	CustomerName string    `json:"customer_name"`
	ProductType  string    `json:"product_type"`
	Stage        string    `json:"stage"`
	ParentFolder string    `json:"parent_folder"`
	FileSize     int64     `json:"file_size"`
	ModTime      time.Time `json:"mod_time"`
	Extension    string    `json:"extension"`
}

// FilesystemClassificationSummary provides aggregate statistics
type FilesystemClassificationSummary struct {
	TotalFiles    int                           `json:"total_files"`
	ByType        map[string]int                `json:"by_type"`
	ByCustomer    map[string]int                `json:"by_customer"`
	ByOfferNumber map[string]int                `json:"by_offer_number"`
	ByStage       map[string]int                `json:"by_stage"`
	ByProductType map[string]int                `json:"by_product_type"`
	Documents     []FilesystemDocClassification `json:"documents"`
	ScanDuration  time.Duration                 `json:"scan_duration"`
}

// FilesystemDocumentClassifierService handles filesystem-based document classification
type FilesystemDocumentClassifierService struct {
	// Patterns for document type detection
	invoicePatterns       []*regexp.Regexp
	deliveryNotePatterns  []*regexp.Regexp
	customerPOPatterns    []*regexp.Regexp
	supplierPOAckPatterns []*regexp.Regexp
	internalPOPatterns    []*regexp.Regexp
	rfqPatterns           []*regexp.Regexp
	costingPatterns       []*regexp.Regexp
	offerPatterns         []*regexp.Regexp
	shippingPatterns      []*regexp.Regexp
	technicalPatterns     []*regexp.Regexp

	// Patterns for metadata extraction
	offerNumberPattern   *regexp.Regexp
	productTypePattern   *regexp.Regexp
	customerPoPatternA   *regexp.Regexp
	customerPoPatternB   *regexp.Regexp
	supplierOrderPattern *regexp.Regexp
	internalPoPattern    *regexp.Regexp
}

// NewFilesystemDocumentClassifierService initializes the classifier with all patterns
func NewFilesystemDocumentClassifierService() *FilesystemDocumentClassifierService {
	return &FilesystemDocumentClassifierService{
		// Invoice patterns
		invoicePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)invoice`),
			regexp.MustCompile(`(?i)\binv\b`),
			regexp.MustCompile(`(?i)^\w{3}-\d{2}\s+INV`), // XXX-25 INV format
			regexp.MustCompile(`(?i)INV-\d+`),
		},

		// Delivery note patterns
		deliveryNotePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)delivery`),
			regexp.MustCompile(`(?i)\bDN\b`),
			regexp.MustCompile(`(?i)^DN[_-]?\d+`),
			regexp.MustCompile(`(?i)delivery\s*note`),
		},

		// Customer PO patterns (GSC: 453XXXXX, NPC: PO_81_XXX)
		customerPOPatterns: []*regexp.Regexp{
			regexp.MustCompile(`453\d{5}`),  // GSC format
			regexp.MustCompile(`PO_81_\d+`), // NPC format
			regexp.MustCompile(`(?i)purchase\s*order`),
			regexp.MustCompile(`(?i)customer.*po`),
		},

		// Supplier PO acknowledgment (Rhine Instruments: 601XXXXXXX)
		supplierPOAckPatterns: []*regexp.Regexp{
			regexp.MustCompile(`601\d{7}`), // Rhine Instruments order confirmation
			regexp.MustCompile(`(?i)order.*confirmation`),
			regexp.MustCompile(`(?i)po.*ack`),
		},

		// Internal PO (PH25-XXX.pdf)
		internalPOPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^PH\d{2}-\d+`), // PH25-XXX format
			regexp.MustCompile(`(?i)PH.*purchase.*order`),
		},

		// RFQ patterns
		rfqPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\brfq\b`),
			regexp.MustCompile(`(?i)^rfq[_-]`),
			regexp.MustCompile(`(?i)request.*quote`),
			regexp.MustCompile(`(?i)quotation.*request`),
		},

		// Costing sheet patterns
		costingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)costing`),
			regexp.MustCompile(`(?i)cost.*sheet`),
			regexp.MustCompile(`(?i)pricing.*sheet`),
		},

		// Commercial offer patterns
		offerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)offer`),
			regexp.MustCompile(`(?i)quotation`),
			regexp.MustCompile(`(?i)proposal`),
			regexp.MustCompile(`(?i)commercial.*offer`),
		},

		// Shipping document patterns
		shippingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)shipping`),
			regexp.MustCompile(`(?i)shipment`),
			regexp.MustCompile(`(?i)packing.*list`),
			regexp.MustCompile(`(?i)bill.*lading`),
			regexp.MustCompile(`(?i)awb`), // Air Waybill
		},

		// Technical document patterns
		technicalPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)technical`),
			regexp.MustCompile(`(?i)specification`),
			regexp.MustCompile(`(?i)datasheet`),
			regexp.MustCompile(`(?i)manual`),
			regexp.MustCompile(`(?i)msds`),
			regexp.MustCompile(`(?i)certificate`),
		},

		// Metadata extraction patterns
		offerNumberPattern:   regexp.MustCompile(`^(\d{3})\s+`), // "101 VERTEX AIT" -> "101"
		productTypePattern:   regexp.MustCompile(`(?i)(AIT|FIT|LIT|PIT|TIT|SP|FT|LT|PT|TT|VALVE|TRANSMITTER|ANALYZER|FLOWMETER)`),
		customerPoPatternA:   regexp.MustCompile(`453\d{5}`),
		customerPoPatternB:   regexp.MustCompile(`PO_81_\d+`),
		supplierOrderPattern: regexp.MustCompile(`601\d{7}`),
		internalPoPattern:    regexp.MustCompile(`PH\d{2}-\d+`),
	}
}

// DocumentClassifier classifies documents based on OCR text
type DocumentClassifier struct {
	// Keyword patterns for each document type
	invoicePatterns         []*regexp.Regexp
	supplierInvoicePatterns []*regexp.Regexp
	rfqPatterns             []*regexp.Regexp
	quotationPatterns       []*regexp.Regexp
	contractPatterns        []*regexp.Regexp
	poPatterns              []*regexp.Regexp
	deliveryNotePatterns    []*regexp.Regexp
	bankStatementPatterns   []*regexp.Regexp
	reportPatterns          []*regexp.Regexp

	// Confidence thresholds
	highConfidenceKeywords int // Number of keywords for high confidence
	lowConfidenceKeywords  int // Number of keywords for low confidence
}

// ClassificationResult represents the classification output
type ClassificationResult struct {
	DocumentType string  `json:"document_type"` // Invoice, RFQ, Quotation, Contract, PO, Other
	Confidence   float64 `json:"confidence"`    // 0.0 - 1.0
	Method       string  `json:"method"`        // "keyword", "heuristic", "unknown"

	// Routing information
	RouteTo         string `json:"route_to"`         // Which screen to route to
	SuggestedAction string `json:"suggested_action"` // What action to suggest

	// Evidence
	KeywordsFound []string `json:"keywords_found"` // Which keywords matched
	Explanation   string   `json:"explanation"`    // Human-readable explanation
}

// NewDocumentClassifier creates a new document classifier
func NewDocumentClassifier() *DocumentClassifier {
	c := &DocumentClassifier{
		highConfidenceKeywords: 3,
		lowConfidenceKeywords:  1,
	}

	// Compile regex patterns for each document type
	c.compilePatterns()

	return c
}

// compilePatterns initializes all regex patterns for document classification
func (c *DocumentClassifier) compilePatterns() {
	// Invoice patterns (Oblate Spheroid - stable, flattened)
	c.invoicePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\binvoice\b`),
		regexp.MustCompile(`(?i)\binvoice\s*#?\s*\d+`),
		regexp.MustCompile(`(?i)\btax\s*invoice\b`),
		regexp.MustCompile(`(?i)\bproforma\s*invoice\b`),
		regexp.MustCompile(`(?i)\bcommercial\s*invoice\b`),
		regexp.MustCompile(`(?i)\bamount\s*due\b`),
		regexp.MustCompile(`(?i)\bpayment\s*terms?\b`),
		regexp.MustCompile(`(?i)\btotal\s*amount\b`),
		regexp.MustCompile(`(?i)\bdue\s*date\b`),
		regexp.MustCompile(`(?i)\bnet\s*\d+\s*days\b`),
	}

	c.supplierInvoicePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bsupplier\s*invoice\b`),
		regexp.MustCompile(`(?i)\bvendor\s*invoice\b`),
		regexp.MustCompile(`(?i)\bbill\s*to\s*:?\s*(?:p\.?\s*h\.?\s*trading|ph\s*trading)`),
		regexp.MustCompile(`(?i)\bship\s*to\s*:?\s*(?:p\.?\s*h\.?\s*trading|ph\s*trading)`),
		regexp.MustCompile(`(?i)\bissued\s*by\b`),
		regexp.MustCompile(`(?i)\bseller\b`),
		regexp.MustCompile(`(?i)\bvendor\b`),
	}

	// RFQ/Tender patterns (Icosahedron - complex, multi-faceted)
	c.rfqPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\brequest\s*for\s*quotation\b`),
		regexp.MustCompile(`(?i)\bRFQ\b`),
		regexp.MustCompile(`(?i)^rfq[_-]`),
		regexp.MustCompile(`(?i)\brequest\s*for\s*proposal\b`),
		regexp.MustCompile(`(?i)\bRFP\b`),
		regexp.MustCompile(`(?i)\brequest\s*for\s*tender\b`),
		regexp.MustCompile(`(?i)\btender\s*document\b`),
		regexp.MustCompile(`(?i)\binvitation\s*to\s*bid\b`),
		regexp.MustCompile(`(?i)\bITB\b`),
		regexp.MustCompile(`(?i)\bplease\s*quote\b`),
		regexp.MustCompile(`(?i)\brequesting\s*quotation\b`),
		regexp.MustCompile(`(?i)\bsubmit\s*your\s*(best\s*)?price\b`),
		regexp.MustCompile(`(?i)\bquotation\s*required\b`),
	}

	// Quotation patterns (Torus - circular exchange)
	c.quotationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bquotation\b`),
		regexp.MustCompile(`(?i)\bquote\s*#?\s*\d+`),
		regexp.MustCompile(`(?i)\bprice\s*quotation\b`),
		regexp.MustCompile(`(?i)\boffer\b`),
		regexp.MustCompile(`(?i)\bvalid\s*until\b`),
		regexp.MustCompile(`(?i)\bvalidity\s*period\b`),
		regexp.MustCompile(`(?i)\bquoted\s*price\b`),
		regexp.MustCompile(`(?i)\bunit\s*price\b`),
		regexp.MustCompile(`(?i)\btotal\s*quoted\b`),
	}

	// Contract patterns (S³ - binding agreement in 4D)
	c.contractPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bcontract\b`),
		regexp.MustCompile(`(?i)\bagreement\b`),
		regexp.MustCompile(`(?i)\bterms\s*and\s*conditions\b`),
		regexp.MustCompile(`(?i)\bparty\s*of\s*the\s*(first|second)\s*part\b`),
		regexp.MustCompile(`(?i)\bwhereas\b`),
		regexp.MustCompile(`(?i)\bhereby\s*agree\b`),
		regexp.MustCompile(`(?i)\beffective\s*date\b`),
		regexp.MustCompile(`(?i)\bsignature\b`),
		regexp.MustCompile(`(?i)\bwitness\b`),
	}

	// Purchase Order patterns (Cube - structured, 6-faced)
	c.poPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpurchase\s*order\b`),
		regexp.MustCompile(`(?i)\bP\.?O\.?\s*#?\s*\d+`),
		regexp.MustCompile(`(?i)\bPO\s*number\b`),
		regexp.MustCompile(`(?i)\border\s*number\b`),
		regexp.MustCompile(`(?i)\bshipping\s*address\b`),
		regexp.MustCompile(`(?i)\bdelivery\s*date\b`),
		regexp.MustCompile(`(?i)\bitem\s*description\b`),
		regexp.MustCompile(`(?i)\bquantity\s*ordered\b`),
	}

	c.deliveryNotePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bdelivery\s*note\b`),
		regexp.MustCompile(`(?i)\bdispatch\s*note\b`),
		regexp.MustCompile(`(?i)\bpacking\s*list\b`),
		regexp.MustCompile(`(?i)\bconsignment\s*note\b`),
		regexp.MustCompile(`(?i)\bdn\s*(?:no|number|#)\b`),
	}

	c.bankStatementPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bbank\s*statement\b`),
		regexp.MustCompile(`(?i)\baccount\s*statement\b`),
		regexp.MustCompile(`(?i)\bopening\s*balance\b`),
		regexp.MustCompile(`(?i)\bclosing\s*balance\b`),
		regexp.MustCompile(`(?i)\brunning\s*balance\b`),
		regexp.MustCompile(`(?i)\baccount\s*number\b`),
		regexp.MustCompile(`(?i)\bdebit\b`),
		regexp.MustCompile(`(?i)\bcredit\b`),
	}

	c.reportPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bmonthly\s*report\b`),
		regexp.MustCompile(`(?i)\bweekly\s*report\b`),
		regexp.MustCompile(`(?i)\bmanagement\s*report\b`),
		regexp.MustCompile(`(?i)\bexecutive\s*summary\b`),
		regexp.MustCompile(`(?i)\banalysis\s*report\b`),
		regexp.MustCompile(`(?i)\bdashboard\s*report\b`),
		regexp.MustCompile(`(?i)\bsummary\s*report\b`),
		regexp.MustCompile(`(?i)\bperformance\s*report\b`),
	}
}

// Classify classifies a document based on OCR text and filename
func (c *DocumentClassifier) Classify(text string, filename string) *ClassificationResult {
	log.Printf("🔍 Classifying document: %s (%d chars)", filename, len(text))

	// Try keyword-based classification first (fast path)
	if result := c.classifyByKeywords(text, filename); result != nil {
		log.Printf("✅ Classified by keywords: %s (confidence: %.2f)", result.DocumentType, result.Confidence)
		return result
	}

	// Fallback to heuristic classification
	result := c.classifyByHeuristics(text, filename)
	log.Printf("✅ Classified by heuristics: %s (confidence: %.2f)", result.DocumentType, result.Confidence)

	return result
}

// classifyByKeywords performs keyword-based classification
func (c *DocumentClassifier) classifyByKeywords(text string, filename string) *ClassificationResult {
	// Convert text to lowercase for case-insensitive matching
	_ = strings.ToLower(text) // Reserved for future keyword matching
	lowerFilename := strings.ToLower(filename)

	// Check each document type
	candidates := []struct {
		docType  string
		patterns []*regexp.Regexp
	}{
		{"BankStatement", c.bankStatementPatterns},
		{"SupplierInvoice", c.supplierInvoicePatterns},
		{"Invoice", c.invoicePatterns},
		{"RFQ", c.rfqPatterns},
		{"Quotation", c.quotationPatterns},
		{"Contract", c.contractPatterns},
		{"PurchaseOrder", c.poPatterns},
		{"DeliveryNote", c.deliveryNotePatterns},
		{"Report", c.reportPatterns},
	}

	var bestMatch *ClassificationResult
	maxKeywords := 0

	for _, candidate := range candidates {
		keywordsFound := []string{}

		// Check text
		for _, pattern := range candidate.patterns {
			if matches := pattern.FindAllString(text, -1); len(matches) > 0 {
				keywordsFound = append(keywordsFound, matches...)
			}
		}

		// Check filename
		for _, pattern := range candidate.patterns {
			if pattern.MatchString(lowerFilename) {
				keywordsFound = append(keywordsFound, "filename:"+candidate.docType)
			}
		}

		if candidate.docType == "SupplierInvoice" {
			if strings.Contains(strings.ToLower(text), "acme instrumentation") &&
				(strings.Contains(strings.ToLower(text), "bill to") || strings.Contains(strings.ToLower(text), "ship to")) {
				keywordsFound = append(keywordsFound, "recipient:ph_trading")
			}
			if strings.Contains(strings.ToLower(text), "vendor") || strings.Contains(strings.ToLower(text), "seller") {
				keywordsFound = append(keywordsFound, "issuer:supplier")
			}
		}

		// If we found keywords, this might be our answer
		if len(keywordsFound) > maxKeywords {
			maxKeywords = len(keywordsFound)

			// Calculate confidence based on number of keywords
			confidence := 0.0
			if maxKeywords >= c.highConfidenceKeywords {
				confidence = 0.95
			} else if maxKeywords >= c.lowConfidenceKeywords {
				confidence = 0.75
			} else {
				confidence = 0.5
			}

			// Keep max 5 keywords for display
			displayKeywords := keywordsFound
			if len(keywordsFound) > 5 {
				displayKeywords = keywordsFound[:5]
			}

			bestMatch = &ClassificationResult{
				DocumentType:  candidate.docType,
				Confidence:    confidence,
				Method:        "keyword",
				KeywordsFound: displayKeywords,
				Explanation:   fmt.Sprintf("Found %d keywords matching %s", len(keywordsFound), candidate.docType),
			}
		}
	}

	// Only return if we have sufficient confidence
	if bestMatch != nil && maxKeywords >= c.lowConfidenceKeywords {
		// Add routing information
		c.addRoutingInfo(bestMatch)
		return bestMatch
	}

	return nil
}

// classifyByHeuristics performs heuristic-based classification
func (c *DocumentClassifier) classifyByHeuristics(text string, filename string) *ClassificationResult {
	lowerText := strings.ToLower(text)
	lowerFilename := strings.ToLower(filename)

	// Heuristic 1: Filename contains document type
	if strings.Contains(lowerFilename, "invoice") {
		if strings.Contains(lowerText, "acme instrumentation") && (strings.Contains(lowerText, "bill to") || strings.Contains(lowerText, "ship to") || strings.Contains(lowerText, "vendor")) {
			return c.createHeuristicResult("SupplierInvoice", 0.65, "Invoice appears addressed to "+overlay.Active().CompanyDisplayName+" by a supplier")
		}
		return c.createHeuristicResult("Invoice", 0.6, "Filename contains 'invoice'")
	}
	if strings.Contains(lowerFilename, "rfq") || strings.Contains(lowerFilename, "tender") {
		return c.createHeuristicResult("RFQ", 0.6, "Filename contains 'rfq' or 'tender'")
	}
	if strings.Contains(lowerFilename, "quote") || strings.Contains(lowerFilename, "quotation") {
		return c.createHeuristicResult("Quotation", 0.6, "Filename contains 'quote' or 'quotation'")
	}
	if strings.Contains(lowerFilename, "contract") || strings.Contains(lowerFilename, "agreement") {
		return c.createHeuristicResult("Contract", 0.6, "Filename contains 'contract' or 'agreement'")
	}
	if strings.Contains(lowerFilename, "statement") && strings.Contains(lowerFilename, "bank") {
		return c.createHeuristicResult("BankStatement", 0.7, "Filename contains 'bank statement'")
	}
	if strings.Contains(lowerFilename, "report") || strings.Contains(lowerFilename, "summary") {
		return c.createHeuristicResult("Report", 0.6, "Filename contains 'report' or 'summary'")
	}
	if strings.Contains(lowerFilename, "delivery") || strings.Contains(lowerFilename, "packing") {
		return c.createHeuristicResult("DeliveryNote", 0.6, "Filename contains delivery/shipping terms")
	}
	if strings.Contains(lowerFilename, "po") || strings.Contains(lowerFilename, "purchase") {
		return c.createHeuristicResult("PurchaseOrder", 0.6, "Filename contains 'po' or 'purchase'")
	}

	// Heuristic 2: Text patterns

	if regexp.MustCompile(`(?i)(bank statement|account statement|opening balance|closing balance|running balance)`).MatchString(lowerText) &&
		regexp.MustCompile(`(?i)(debit|credit|balance)`).MatchString(lowerText) {
		return c.createHeuristicResult("BankStatement", 0.85, "Contains bank statement balances and transaction columns")
	}

	if regexp.MustCompile(`(?i)(delivery note|dispatch note|packing list|consignment note)`).MatchString(lowerText) {
		return c.createHeuristicResult("DeliveryNote", 0.75, "Contains delivery/shipping document markers")
	}

	if regexp.MustCompile(`(?i)(monthly report|weekly report|management report|executive summary|performance report|analysis report)`).MatchString(lowerText) {
		return c.createHeuristicResult("Report", 0.7, "Contains report-style summary terminology")
	}

	if strings.Contains(lowerText, "invoice") && strings.Contains(lowerText, "acme instrumentation") &&
		regexp.MustCompile(`(?i)(bill to|ship to|vendor|seller|issued by)`).MatchString(lowerText) {
		return c.createHeuristicResult("SupplierInvoice", 0.72, "Invoice appears to be received from a supplier")
	}

	// Look for currency symbols and amounts (likely Invoice or Quotation)
	currencyPattern := regexp.MustCompile(`[$£€¥]\s*\d+|(?i)(amount|total|price):\s*\d+`)
	if currencyPattern.MatchString(lowerText) {
		// Check if it has "due" or "payment" (more likely Invoice)
		if regexp.MustCompile(`(?i)(due|payment|pay|amount due)`).MatchString(lowerText) {
			return c.createHeuristicResult("Invoice", 0.5, "Contains currency and payment terms")
		}
		// Otherwise, could be Quotation
		return c.createHeuristicResult("Quotation", 0.45, "Contains pricing information")
	}

	// Look for table-like structures (could be RFQ or Quotation)
	if strings.Contains(lowerText, "item") && strings.Contains(lowerText, "quantity") {
		// Check if it asks for pricing (RFQ)
		if regexp.MustCompile(`(?i)(please (quote|submit|provide)|requesting)`).MatchString(lowerText) {
			return c.createHeuristicResult("RFQ", 0.5, "Contains item list and price request")
		}
		// Otherwise, could be Quotation or PO
		return c.createHeuristicResult("Quotation", 0.45, "Contains structured item list")
	}

	// Look for legal language (Contract)
	if regexp.MustCompile(`(?i)(whereas|hereby|party of the|in witness whereof)`).MatchString(lowerText) {
		return c.createHeuristicResult("Contract", 0.5, "Contains legal language")
	}

	// Heuristic 3: Default to "Other" with low confidence
	return c.createHeuristicResult("Other", 0.3, "No clear document type patterns found")
}

// createHeuristicResult creates a classification result from heuristics
func (c *DocumentClassifier) createHeuristicResult(docType string, confidence float64, explanation string) *ClassificationResult {
	result := &ClassificationResult{
		DocumentType:  docType,
		Confidence:    confidence,
		Method:        "heuristic",
		KeywordsFound: []string{},
		Explanation:   explanation,
	}

	c.addRoutingInfo(result)
	return result
}

// addRoutingInfo adds routing information based on document type
func (c *DocumentClassifier) addRoutingInfo(result *ClassificationResult) {
	switch result.DocumentType {
	case "Invoice":
		result.RouteTo = "dashboard"
		result.SuggestedAction = "Link to dashboard and flag for reconciliation"

	case "RFQ":
		result.RouteTo = "opportunities"
		result.SuggestedAction = "Create opportunity in Opportunities screen"

	case "Quotation":
		result.RouteTo = "pricing"
		result.SuggestedAction = "Link to Pricing screen for review"

	case "Contract":
		result.RouteTo = "inbox"
		result.SuggestedAction = "Flag for contract review (placeholder for future contract module)"

	case "PurchaseOrder":
		result.RouteTo = "orders"
		result.SuggestedAction = "Create order in Orders screen"

	case "SupplierInvoice":
		result.RouteTo = "finance"
		result.SuggestedAction = "Record supplier invoice and match to PO"

	case "DeliveryNote":
		result.RouteTo = "operations"
		result.SuggestedAction = "Record delivery note and update fulfillment"

	case "BankStatement":
		result.RouteTo = "finance"
		result.SuggestedAction = "Import bank statement for reconciliation"

	case "Report":
		result.RouteTo = "intelligence"
		result.SuggestedAction = "Store as reference report for manual review"

	default:
		result.RouteTo = "inbox"
		result.SuggestedAction = "Send to Inbox for manual review"
	}
}

// ========================================================================

func AddAIRoutingInfo(result *ClassificationResult) {
	switch result.DocumentType {
	case "Invoice":
		result.RouteTo = "finance"
		result.SuggestedAction = "Record customer invoice"
	case "SupplierInvoice":
		result.RouteTo = "finance"
		result.SuggestedAction = "Record supplier invoice and match to PO"
	case "RFQ":
		result.RouteTo = "opportunities"
		result.SuggestedAction = "Create opportunity from RFQ"
	case "Quotation":
		result.RouteTo = "pricing"
		result.SuggestedAction = "Review quotation and link to opportunity"
	case "PurchaseOrder":
		result.RouteTo = "orders"
		result.SuggestedAction = "Create order from purchase order"
	case "DeliveryNote":
		result.RouteTo = "operations"
		result.SuggestedAction = "Record delivery and update GRN"
	case "BankStatement":
		result.RouteTo = "finance"
		result.SuggestedAction = "Import bank statement for reconciliation"
	case "Contract":
		result.RouteTo = "inbox"
		result.SuggestedAction = "Flag for contract review"
	case "Report":
		result.RouteTo = "intelligence"
		result.SuggestedAction = "Store report for reference and manual review"
	default:
		result.RouteTo = "inbox"
		result.SuggestedAction = "Review document manually"
	}
}

// ========================================================================
// INTEGRATION WITH QUICK CAPTURE
// ========================================================================

// ClassifyDocument is the regex-based fallback classifier (used when AI is unavailable)

func (s *FilesystemDocumentClassifierService) classifyDocument(fullPath string, info os.FileInfo, basePath string) FilesystemDocClassification {
	fileName := info.Name()
	fileNameLower := strings.ToLower(fileName)
	ext := strings.ToLower(filepath.Ext(fileName))

	// Get relative path for folder analysis
	relPath, _ := filepath.Rel(basePath, fullPath)
	parentFolder := filepath.Base(filepath.Dir(fullPath))
	folderPath := filepath.Dir(relPath)

	doc := FilesystemDocClassification{
		FilePath:     fullPath,
		FileName:     fileName,
		ParentFolder: parentFolder,
		FileSize:     info.Size(),
		ModTime:      info.ModTime(),
		Extension:    ext,
	}

	// Extract metadata from folder path
	doc.OfferNumber = s.extractOfferNumber(folderPath)
	doc.CustomerName = s.extractCustomerName(folderPath, fileName)
	doc.ProductType = s.extractProductType(folderPath, fileName)
	doc.Stage = s.extractStage(folderPath)

	// Classify document type
	doc.DocumentType = s.determineDocumentType(fileName, fileNameLower, ext, folderPath, parentFolder)

	return doc
}

// determineDocumentType applies pattern matching to classify document
func (s *FilesystemDocumentClassifierService) determineDocumentType(fileName, fileNameLower, ext, folderPath, parentFolder string) string {
	// Strong filename/document-specific signals should win over generic stage folders.
	// RFQ/OFFER/EXECUTION folders are useful context, but they should not override an
	// invoice, PO acknowledgement, delivery note, or technical datasheet filename.

	// Invoice detection
	for _, pattern := range s.invoicePatterns {
		if pattern.MatchString(fileName) {
			return "INVOICE"
		}
	}

	// Delivery note detection
	for _, pattern := range s.deliveryNotePatterns {
		if pattern.MatchString(fileName) {
			return "DELIVERY_NOTE"
		}
	}

	// Supplier PO acknowledgment (Rhine Instruments order confirmations)
	for _, pattern := range s.supplierPOAckPatterns {
		if pattern.MatchString(fileName) {
			return "SUPPLIER_PO_ACK"
		}
	}

	// Internal PO (Acme Instrumentation to suppliers)
	for _, pattern := range s.internalPOPatterns {
		if pattern.MatchString(fileName) {
			return "INTERNAL_PO"
		}
	}

	// Customer PO (from GSC, NPC, etc.)
	for _, pattern := range s.customerPOPatterns {
		if pattern.MatchString(fileName) {
			return "CUSTOMER_PO"
		}
	}

	// Costing sheet (Excel files)
	if ext == ".xlsx" || ext == ".xls" {
		for _, pattern := range s.costingPatterns {
			if pattern.MatchString(fileName) {
				return "COSTING_SHEET"
			}
		}
	}

	// Check folder-based classification next
	folderPathLower := strings.ToLower(folderPath)
	parentFolderLower := strings.ToLower(parentFolder)

	// RFQ folder detection
	if strings.Contains(folderPathLower, "rfq") || strings.Contains(parentFolderLower, "rfq") {
		if ext == ".msg" || ext == ".eml" {
			return "RFQ_EMAIL"
		}
		for _, pattern := range s.rfqPatterns {
			if pattern.MatchString(fileName) {
				return "RFQ_DOCUMENT"
			}
		}
		for _, pattern := range s.technicalPatterns {
			if pattern.MatchString(fileName) {
				return "TECHNICAL_DOC"
			}
		}
		if ext == ".pdf" || ext == ".docx" || ext == ".doc" {
			return "RFQ_DOCUMENT"
		}
	}

	// Offer folder detection
	if strings.Contains(folderPathLower, "offer") || strings.Contains(parentFolderLower, "offer") {
		if ext == ".pdf" {
			return "COMMERCIAL_OFFER"
		}
	}

	// Execution/Shipment folder detection
	if strings.Contains(folderPathLower, "execution") || strings.Contains(folderPathLower, "shipment") ||
		strings.Contains(parentFolderLower, "execution") || strings.Contains(parentFolderLower, "shipment") {
		if ext == ".pdf" {
			return "SHIPPING_DOC"
		}
	}

	// RFQ documents
	for _, pattern := range s.rfqPatterns {
		if pattern.MatchString(fileName) {
			if ext == ".msg" || ext == ".eml" {
				return "RFQ_EMAIL"
			}
			return "RFQ_DOCUMENT"
		}
	}

	// Shipping documents
	for _, pattern := range s.shippingPatterns {
		if pattern.MatchString(fileName) {
			return "SHIPPING_DOC"
		}
	}

	// Technical documents
	for _, pattern := range s.technicalPatterns {
		if pattern.MatchString(fileName) {
			return "TECHNICAL_DOC"
		}
	}

	// Commercial offers
	for _, pattern := range s.offerPatterns {
		if pattern.MatchString(fileName) {
			return "COMMERCIAL_OFFER"
		}
	}

	// Default to OTHER
	return "OTHER"
}

// extractOfferNumber extracts offer number from folder path (e.g., "101" from "101 VERTEX AIT")
func (s *FilesystemDocumentClassifierService) extractOfferNumber(folderPath string) string {
	parts := splitDocumentPathParts(folderPath)

	for _, part := range parts {
		if matches := s.offerNumberPattern.FindStringSubmatch(part); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// extractCustomerName extracts customer name from folder path or filename
func (s *FilesystemDocumentClassifierService) extractCustomerName(folderPath, fileName string) string {
	if exportCustomer := extractAsymmFlowExportCustomerName(folderPath); exportCustomer != "" {
		return exportCustomer
	}

	// Try to extract from folder path first (e.g., "101 VERTEX AIT" -> "VERTEX")
	parts := splitDocumentPathParts(folderPath)

	for _, part := range parts {
		// Match pattern: "101 VERTEX AIT" -> extract "VERTEX"
		if matches := s.offerNumberPattern.FindStringSubmatch(part); len(matches) > 0 {
			// Remove offer number and product type to get customer name
			remainder := strings.TrimSpace(strings.TrimPrefix(part, matches[0]))

			// Remove product type suffix
			customerParts := strings.Fields(remainder)
			if len(customerParts) > 0 {
				// Check if last part is a product type
				lastPart := customerParts[len(customerParts)-1]
				if s.productTypePattern.MatchString(lastPart) && len(customerParts) > 1 {
					// Customer name is everything except last part
					return strings.Join(customerParts[:len(customerParts)-1], " ")
				}
				// Return full remainder as customer name
				return remainder
			}
		}

		// Also check for known customer names
		partUpper := strings.ToUpper(part)
		knownCustomers := []string{"GSC", "NPC", "NGA", "VERTEX", "DPC", "CGC", "HZP"}
		for _, customer := range knownCustomers {
			if strings.Contains(partUpper, customer) {
				return customer
			}
		}
	}

	// Try to extract from filename (e.g., GSC PO, NPC invoice)
	fileNameUpper := strings.ToUpper(fileName)
	knownCustomers := []string{"GSC", "NPC", "NGA", "VERTEX", "DPC", "CGC", "HZP"}
	for _, customer := range knownCustomers {
		if strings.Contains(fileNameUpper, customer) {
			return customer
		}
	}

	return ""
}

func splitDocumentPathParts(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\' || r == filepath.Separator
	})
}

func extractAsymmFlowExportCustomerName(folderPath string) string {
	parts := splitDocumentPathParts(folderPath)
	for i, part := range parts {
		if !strings.EqualFold(strings.TrimSpace(part), "AsymmFlow Exports") {
			continue
		}
		if i+1 >= len(parts) {
			return ""
		}
		next := strings.TrimSpace(parts[i+1])
		if strings.EqualFold(next, "Suppliers") || strings.EqualFold(next, "Reports") {
			return ""
		}
		return displayNameFromExportFolder(next)
	}
	return ""
}

func displayNameFromExportFolder(segment string) string {
	name := strings.TrimSpace(strings.ReplaceAll(segment, "_", " "))
	if name == "" || strings.EqualFold(name, "unnamed") || strings.EqualFold(name, "Unassigned Customer") {
		return ""
	}
	return name
}

// extractProductType extracts product type from folder path or filename
func (s *FilesystemDocumentClassifierService) extractProductType(folderPath, fileName string) string {
	// Check folder path first
	if matches := s.productTypePattern.FindStringSubmatch(folderPath); len(matches) > 0 {
		return strings.ToUpper(matches[1])
	}

	// Check filename
	if matches := s.productTypePattern.FindStringSubmatch(fileName); len(matches) > 0 {
		return strings.ToUpper(matches[1])
	}

	return ""
}

// extractStage determines the stage from folder path
func (s *FilesystemDocumentClassifierService) extractStage(folderPath string) string {
	folderPathUpper := strings.ToUpper(folderPath)

	if strings.Contains(folderPathUpper, "RFQ") {
		return "RFQ"
	}

	if strings.Contains(folderPathUpper, "OFFER") || strings.Contains(folderPathUpper, "QUOTATION") {
		return "OFFER"
	}

	if strings.Contains(folderPathUpper, "EXECUTION") || strings.Contains(folderPathUpper, "ORDER") ||
		strings.Contains(folderPathUpper, "SHIPMENT") || strings.Contains(folderPathUpper, "DELIVERY") {
		return "EXECUTION"
	}

	return ""
}
