// ═══════════════════════════════════════════════════════════════════════════
// OFFER SCANNER - Scan Offer Folders and Extract Structured Metadata
//
// FEATURES:
//   - Recursive folder scanning
//   - Customer/product type extraction from folder names
//   - Workflow stage detection (RFQ → OFFER → EXECUTION)
//   - Revision tracking
//   - File inventory by type (XLSX, XML, PDF, MSG, DOCX, JPG)
//   - Summary statistics
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OfferMetadata represents extracted data from one offer folder
type OfferMetadata struct {
	// Core identification
	OfferID      string `json:"offer_id"`      // "101", "102", etc.
	CustomerName string `json:"customer_name"` // "VERTEX", "AQUAPURE", etc.
	ProductType  string `json:"product_type"`  // "AIT", "FIT", "LIT", "SP", etc.
	FullPath     string `json:"full_path"`     // Absolute path to offer folder

	// Workflow stages
	HasRFQ       bool `json:"has_rfq"`       // RFQ folder exists
	HasOffer     bool `json:"has_offer"`     // OFFER folder exists
	HasExecution bool `json:"has_execution"` // EXECUTION folder exists (PO received!)

	// Revision tracking
	RevisionCount int      `json:"revision_count"` // Number of REV-* folders
	Revisions     []string `json:"revisions"`      // ["REV-1", "REV-2", ...]
	HasSiteVisit  bool     `json:"has_site_visit"` // Site visit photos exist

	// File inventory
	XLSXFiles []string `json:"xlsx_files"` // Costing files (*.xlsx)
	XMLFiles  []string `json:"xml_files"`  // Rhine Instruments pricing (*.xml)
	PDFFiles  []string `json:"pdf_files"`  // Offers, invoices (*.pdf)
	MSGFiles  []string `json:"msg_files"`  // RFQ emails (*.msg)
	DOCXFiles []string `json:"docx_files"` // Offer documents (*.docx)
	JPGFiles  []string `json:"jpg_files"`  // Site photos (*.jpg)

	// Timestamps
	FirstRevisionDate time.Time `json:"first_revision_date"`
	LastModifiedDate  time.Time `json:"last_modified_date"`
	ExecutionDate     time.Time `json:"execution_date"`

	// Computed metrics
	CycleDays        int     `json:"cycle_days"`         // Days from RFQ to execution
	RevisionsPerWeek float64 `json:"revisions_per_week"` // Revision intensity
	HasCosting       bool    `json:"has_costing"`        // Costing file found
	HasEHPricing     bool    `json:"has_eh_pricing"`     // Rhine XML found
	EstimatedValue   float64 `json:"estimated_value"`    // Estimated from filename patterns
}

// OfferScanner scans offer folders and extracts structured metadata
type OfferScanner struct {
	BasePath       string            // Root path to scan (e.g., "PH_TEST_DATA/Offers No 101 -150 (2025)")
	Offers         []OfferMetadata   // All discovered offers
	KnownCustomers map[string]bool   // Known customer names
	ProductTypes   map[string]string // Product type descriptions
}

// NewOfferScanner creates scanner with known customer/product mappings
func NewOfferScanner(basePath string) *OfferScanner {
	return &OfferScanner{
		BasePath: basePath,
		Offers:   make([]OfferMetadata, 0, 50),
		KnownCustomers: map[string]bool{
			"VERTEX":   true,
			"AQUAPURE": true,
			"PNM":      true,
			"MERIDIAN": true,
			"GSC":      true,
			"NPC":      true,
			"PLC":      true,
			"DELTA":    true,
			"BLUEWAVE": true,
			"CJV":      true,
			"MDY":      true,
			"DPC":      true,
			"SLM":      true,
			"OMNI":     true,
			"MWS":      true,
			"NGA":      true,
			"ESW":      true,
			"LGS":      true,
			"AMS":      true,
			"HPC":      true,
			"CSW":      true,
			"SMT":      true,
			"K&S":      true,
			"LMN":      true,
		},
		ProductTypes: map[string]string{
			"AIT":  "Analytical Instruments",  // pH, conductivity, turbidity
			"FIT":  "Flow Instruments",        // Flow meters
			"LIT":  "Level Instruments",       // Level meters
			"TIT":  "Temperature Instruments", // Temperature sensors
			"PIT":  "Pressure Instruments",    // Pressure transmitters
			"SP":   "Spare Parts",             // Replacement parts
			"FEED": "Feed/Project",            // Large projects
		},
	}
}

// ScanAll recursively scans all offer folders
func (os *OfferScanner) ScanAll() error {
	// Walk all directories under base path
	err := filepath.WalkDir(os.BasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only process directories at level 1 (offer folders)
		if !d.IsDir() {
			return nil
		}

		// Skip base directory itself
		if path == os.BasePath {
			return nil
		}

		// Check if this looks like an offer folder (starts with digits)
		folderName := filepath.Base(path)
		if matched, _ := regexp.MatchString(`^\d+`, folderName); matched {
			// Extract metadata from this offer
			metadata, err := os.ExtractMetadata(path)
			if err != nil {
				// Log but continue (don't break entire scan)
				return nil
			}

			os.Offers = append(os.Offers, metadata)
		}

		return nil
	})

	return err
}

// ExtractMetadata extracts all data from one offer folder
func (os *OfferScanner) ExtractMetadata(offerPath string) (OfferMetadata, error) {
	metadata := OfferMetadata{
		FullPath:  offerPath,
		Revisions: make([]string, 0, 5),
		XLSXFiles: make([]string, 0, 10),
		XMLFiles:  make([]string, 0, 5),
		PDFFiles:  make([]string, 0, 20),
		MSGFiles:  make([]string, 0, 5),
		DOCXFiles: make([]string, 0, 10),
		JPGFiles:  make([]string, 0, 10),
	}

	// Extract offer ID and customer from folder name
	folderName := filepath.Base(offerPath)
	metadata.OfferID, metadata.CustomerName, metadata.ProductType = os.ParseFolderName(folderName)

	// Scan folder structure
	err := filepath.WalkDir(offerPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue scanning
		}

		relPath, _ := filepath.Rel(offerPath, path)
		upperPath := strings.ToUpper(relPath)

		// Detect workflow stages
		if strings.Contains(upperPath, "RFQ") && d.IsDir() {
			metadata.HasRFQ = true
		}
		if strings.Contains(upperPath, "OFFER") && d.IsDir() {
			metadata.HasOffer = true
		}
		if strings.Contains(upperPath, "EXECUTION") && d.IsDir() {
			metadata.HasExecution = true
		}
		if strings.Contains(upperPath, "SITE VISIT") && d.IsDir() {
			metadata.HasSiteVisit = true
		}

		// Detect revisions
		if matched, _ := regexp.MatchString(`REV-\d+`, upperPath); matched && d.IsDir() {
			revName := filepath.Base(path)
			metadata.Revisions = append(metadata.Revisions, revName)
			metadata.RevisionCount++
		}

		// Categorize files by extension
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))

			switch ext {
			case ".xlsx", ".xls":
				metadata.XLSXFiles = append(metadata.XLSXFiles, relPath)
				if strings.Contains(strings.ToUpper(filepath.Base(path)), "COSTING") {
					metadata.HasCosting = true
				}
			case ".xml":
				metadata.XMLFiles = append(metadata.XMLFiles, relPath)
				if strings.Contains(strings.ToUpper(filepath.Base(path)), "EHONLINE") ||
					strings.Contains(strings.ToUpper(filepath.Base(path)), "Rhine Instruments") {
					metadata.HasEHPricing = true
				}
			case ".pdf":
				metadata.PDFFiles = append(metadata.PDFFiles, relPath)

				// Extract execution date from invoice filenames
				if strings.Contains(upperPath, "EXECUTION") &&
					(strings.Contains(strings.ToUpper(filepath.Base(path)), "INVOICE") ||
						strings.Contains(strings.ToUpper(filepath.Base(path)), "INV")) {
					info, _ := d.Info()
					if info != nil && metadata.ExecutionDate.IsZero() {
						metadata.ExecutionDate = info.ModTime()
					}
				}
			case ".msg":
				metadata.MSGFiles = append(metadata.MSGFiles, relPath)
			case ".docx", ".doc":
				metadata.DOCXFiles = append(metadata.DOCXFiles, relPath)
			case ".jpg", ".jpeg", ".png":
				metadata.JPGFiles = append(metadata.JPGFiles, relPath)
			}

			// Track modification dates
			info, _ := d.Info()
			if info != nil {
				modTime := info.ModTime()
				if metadata.LastModifiedDate.IsZero() || modTime.After(metadata.LastModifiedDate) {
					metadata.LastModifiedDate = modTime
				}
			}
		}

		return nil
	})

	// Compute metrics
	if metadata.RevisionCount > 0 && !metadata.FirstRevisionDate.IsZero() {
		cycleDuration := metadata.LastModifiedDate.Sub(metadata.FirstRevisionDate)
		metadata.CycleDays = int(cycleDuration.Hours() / 24)

		if metadata.CycleDays > 0 {
			metadata.RevisionsPerWeek = float64(metadata.RevisionCount) * 7.0 / float64(metadata.CycleDays)
		}
	}

	return metadata, err
}

// ParseFolderName extracts offer ID, customer name, and product type from folder name
func (os *OfferScanner) ParseFolderName(folderName string) (offerID, customer, productType string) {
	// Extract offer ID (leading digits)
	re := regexp.MustCompile(`^(\d+)[-\s]*(.*)$`)
	matches := re.FindStringSubmatch(folderName)

	if len(matches) < 3 {
		return "", "", ""
	}

	offerID = matches[1]
	remainder := strings.ToUpper(strings.TrimSpace(matches[2]))

	// Normalize separators: replace dashes with spaces, then split on whitespace
	normalized := strings.ReplaceAll(remainder, "-", " ")

	// Try to extract product type (last uppercase word)
	words := strings.Fields(normalized)
	if len(words) > 0 {
		lastWord := words[len(words)-1]

		// Check if last word is a known product type
		if _, exists := os.ProductTypes[lastWord]; exists {
			productType = lastWord
			// Remove product type from customer name search
			remainder = strings.TrimSuffix(remainder, lastWord)
			remainder = strings.TrimSpace(remainder)
		}
	}

	// Extract customer name (match against known customers)
	for knownCustomer := range os.KnownCustomers {
		if strings.Contains(remainder, knownCustomer) {
			customer = knownCustomer
			break
		}
	}

	// If no known customer matched, take first substantial word
	if customer == "" && len(words) > 0 {
		customer = words[0]
	}

	return offerID, customer, productType
}

// GetByCustomer returns all offers for a specific customer
func (os *OfferScanner) GetByCustomer(customerName string) []OfferMetadata {
	results := make([]OfferMetadata, 0, 10)
	upperCustomer := strings.ToUpper(customerName)

	for _, offer := range os.Offers {
		if strings.Contains(strings.ToUpper(offer.CustomerName), upperCustomer) {
			results = append(results, offer)
		}
	}

	return results
}

// GetByProductType returns all offers for a specific product type
func (os *OfferScanner) GetByProductType(productType string) []OfferMetadata {
	results := make([]OfferMetadata, 0, 10)
	upperType := strings.ToUpper(productType)

	for _, offer := range os.Offers {
		if strings.ToUpper(offer.ProductType) == upperType {
			results = append(results, offer)
		}
	}

	return results
}

// GetExecuted returns all offers that reached execution stage
func (os *OfferScanner) GetExecuted() []OfferMetadata {
	results := make([]OfferMetadata, 0, 20)

	for _, offer := range os.Offers {
		if offer.HasExecution {
			results = append(results, offer)
		}
	}

	return results
}

// GetPending returns all offers still in offer stage (no execution)
func (os *OfferScanner) GetPending() []OfferMetadata {
	results := make([]OfferMetadata, 0, 20)

	for _, offer := range os.Offers {
		if offer.HasOffer && !offer.HasExecution {
			results = append(results, offer)
		}
	}

	return results
}

// ScanSummary returns statistics about scanned offers
type ScanSummary struct {
	TotalOffers      int               `json:"total_offers"`
	UniqueCustomers  int               `json:"unique_customers"`
	ProductTypeCount map[string]int    `json:"product_type_count"`
	ExecutionRate    float64           `json:"execution_rate"` // % that got PO
	AvgRevisions     float64           `json:"avg_revisions"`
	AvgCycleDays     float64           `json:"avg_cycle_days"`
	CustomersRanked  []CustomerRanking `json:"customers_ranked"` // By order volume
}

// CustomerRanking tracks customer engagement
type CustomerRanking struct {
	Name           string  `json:"name"`
	OfferCount     int     `json:"offer_count"`
	ExecutionCount int     `json:"execution_count"`
	WinRate        float64 `json:"win_rate"` // % executed
}

// GetSummary generates statistics from scanned offers
func (os *OfferScanner) GetSummary() ScanSummary {
	summary := ScanSummary{
		TotalOffers:      len(os.Offers),
		ProductTypeCount: make(map[string]int),
		CustomersRanked:  make([]CustomerRanking, 0, 20),
	}

	// Count unique customers and product types
	customerMap := make(map[string]*CustomerRanking)

	totalRevisions := 0
	totalCycleDays := 0
	executedCount := 0

	for _, offer := range os.Offers {
		// Product type counts
		if offer.ProductType != "" {
			summary.ProductTypeCount[offer.ProductType]++
		}

		// Customer tracking
		if offer.CustomerName != "" {
			if _, exists := customerMap[offer.CustomerName]; !exists {
				customerMap[offer.CustomerName] = &CustomerRanking{
					Name: offer.CustomerName,
				}
			}

			ranking := customerMap[offer.CustomerName]
			ranking.OfferCount++

			if offer.HasExecution {
				ranking.ExecutionCount++
				executedCount++
			}
		}

		// Metrics
		totalRevisions += offer.RevisionCount
		if offer.CycleDays > 0 {
			totalCycleDays += offer.CycleDays
		}
	}

	// Calculate averages
	if summary.TotalOffers > 0 {
		summary.AvgRevisions = float64(totalRevisions) / float64(summary.TotalOffers)
		summary.ExecutionRate = float64(executedCount) * 100.0 / float64(summary.TotalOffers)

		if totalCycleDays > 0 {
			summary.AvgCycleDays = float64(totalCycleDays) / float64(summary.TotalOffers)
		}
	}

	// Build customer rankings
	summary.UniqueCustomers = len(customerMap)

	for _, ranking := range customerMap {
		if ranking.OfferCount > 0 {
			ranking.WinRate = float64(ranking.ExecutionCount) * 100.0 / float64(ranking.OfferCount)
		}
		summary.CustomersRanked = append(summary.CustomersRanked, *ranking)
	}

	// Sort by offer count (descending)
	// Simple bubble sort (fine for ~20 customers)
	for i := 0; i < len(summary.CustomersRanked); i++ {
		for j := i + 1; j < len(summary.CustomersRanked); j++ {
			if summary.CustomersRanked[j].OfferCount > summary.CustomersRanked[i].OfferCount {
				summary.CustomersRanked[i], summary.CustomersRanked[j] =
					summary.CustomersRanked[j], summary.CustomersRanked[i]
			}
		}
	}

	return summary
}
