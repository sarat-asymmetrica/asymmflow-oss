package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"ph_holdings_app/pkg/documents/excel"
)

// ETLService handles the Extract-Transform-Load pipeline for business documents
type ETLService struct {
	ocrService    *SimpleOCRService
	db            any // Will be *gorm.DB when wired
	basePath      string
	opportunities map[string]*OpportunityRecord // Folder number -> opportunity
}

// OpportunityRecord represents a business opportunity from the master Excel
type OpportunityRecord struct {
	Year          string  `json:"year"`
	OppNo         string  `json:"opp_no"`
	FolderNo      string  `json:"folder_no"`
	FolderName    string  `json:"folder_name"`
	Title         string  `json:"title"`
	EndUser       string  `json:"end_user"` // Customer
	LatestComment string  `json:"latest_comment"`
	EHReference   string  `json:"eh_reference"`
	ValueBHD      float64 `json:"value_bhd"`
	Status        string  `json:"status"` // "Closed (lost)", "Closed (Payment Received)", etc.
	ReasonForLoss string  `json:"reason_for_loss"`
	QuoteDate     string  `json:"quote_date"`
	OrderDate     string  `json:"order_date"`
	DeliveryDate  string  `json:"delivery_date"`
	PaymentTerms  string  `json:"payment_terms"`
	Owner         string  `json:"owner"`
}

// OCRBatchResult stores results of batch OCR processing
type OCRBatchResult struct {
	FolderPath   string             `json:"folder_path"`
	FolderNo     string             `json:"folder_no"`
	FolderName   string             `json:"folder_name"`
	DocType      string             `json:"doc_type"` // "rfq", "offer", "execution"
	Files        []OCRFileResult    `json:"files"`
	Opportunity  *OpportunityRecord `json:"opportunity,omitempty"`
	ProcessedAt  time.Time          `json:"processed_at"`
	TotalFiles   int                `json:"total_files"`
	SuccessCount int                `json:"success_count"`
	ErrorCount   int                `json:"error_count"`
}

// OCRFileResult stores result for a single file
type OCRFileResult struct {
	FilePath      string         `json:"file_path"`
	FileName      string         `json:"file_name"`
	Success       bool           `json:"success"`
	Engine        string         `json:"engine"`
	Text          string         `json:"text,omitempty"`
	ExtractedData map[string]any `json:"extracted_data,omitempty"`
	Error         string         `json:"error,omitempty"`
	ProcessingMS  int64          `json:"processing_ms"`
}

// ExtractedCustomer represents a customer extracted from documents
type ExtractedCustomer struct {
	Name           string   `json:"name"`
	NormalizedName string   `json:"normalized_name"`
	Sources        []string `json:"sources"` // Files where this customer was found
	FolderNumbers  []string `json:"folder_numbers"`
	EmailAddresses []string `json:"email_addresses,omitempty"`
	PhoneNumbers   []string `json:"phone_numbers,omitempty"`
}

// ExtractedSupplier represents a supplier extracted from documents
type ExtractedSupplier struct {
	Name           string   `json:"name"`
	NormalizedName string   `json:"normalized_name"`
	Sources        []string `json:"sources"`
	Products       []string `json:"products,omitempty"`
}

// NewETLService creates a new ETL service
func NewETLService(ocrService *SimpleOCRService, basePath string) *ETLService {
	return &ETLService{
		ocrService:    ocrService,
		basePath:      basePath,
		opportunities: make(map[string]*OpportunityRecord),
	}
}

// LoadOpportunities loads the opportunity master data from Excel
func (e *ETLService) LoadOpportunities(filePath string) error {
	log.Printf("📊 Loading opportunities from: %s", filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open opportunities file: %w", err)
	}
	defer f.Close()

	// Read the first sheet (2025 data)
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("no sheets found in opportunities file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return fmt.Errorf("failed to read rows: %w", err)
	}

	// Find the header row (contains "Folder No" or similar), then index it
	// through the pkg/documents/excel engine (Wave 3 B.3).
	headerIdx := -1
	var colMap excel.HeaderIndex
	for i, row := range rows {
		for _, cell := range row {
			cellLower := excel.Normalize(cell)
			if strings.Contains(cellLower, "folder no") || strings.Contains(cellLower, "folder name") {
				headerIdx = i
				break
			}
		}
		if headerIdx >= 0 {
			colMap = excel.IndexHeader(row)
			break
		}
	}

	if headerIdx < 0 {
		// Try row 0 as header
		headerIdx = 0
		colMap = excel.IndexHeader(rows[0])
	}

	log.Printf("📊 Found %d columns, header at row %d", len(colMap), headerIdx+1)

	// Helper function to get cell value safely
	getCell := func(row []string, key string) string {
		return colMap.Cell(row, key)
	}

	// Parse data rows
	count := 0
	for i := headerIdx + 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		folderNo := getCell(row, "folder no.")
		if folderNo == "" {
			folderNo = getCell(row, "folder no")
		}
		if folderNo == "" {
			continue
		}

		// Normalize folder number (remove leading zeros, etc.)
		folderNo = strings.TrimLeft(folderNo, "0")
		if folderNo == "" {
			folderNo = "0"
		}

		opp := &OpportunityRecord{
			Year:          getCell(row, "year"),
			OppNo:         getCell(row, "opp. no."),
			FolderNo:      folderNo,
			FolderName:    getCell(row, "folder name"),
			Title:         getCell(row, "sfdc opportunity title"),
			EndUser:       getCell(row, "end user"),
			LatestComment: getCell(row, "latest comment"),
			EHReference:   getCell(row, "e+h refference no"),
			Status:        getCell(row, "status"),
			ReasonForLoss: getCell(row, "reason for loss"),
			QuoteDate:     getCell(row, "quote date"),
			OrderDate:     getCell(row, "order date"),
			DeliveryDate:  getCell(row, "approx delivery date"),
			PaymentTerms:  getCell(row, "payment terms"),
			Owner:         getCell(row, "owner"),
		}

		// Parse value
		valueStr := getCell(row, "value in bhd")
		if valueStr != "" {
			valueStr = strings.ReplaceAll(valueStr, ",", "")
			if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
				opp.ValueBHD = v
			}
		}

		e.opportunities[folderNo] = opp
		count++
	}

	log.Printf("✅ Loaded %d opportunities", count)
	return nil
}

// ScanFolder scans a folder for documents and returns file paths by type
func (e *ETLService) ScanFolder(folderPath string) (map[string][]string, error) {
	result := map[string][]string{
		"rfq":       {},
		"offer":     {},
		"execution": {},
		"other":     {},
	}

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		supportedExts := map[string]bool{
			".pdf": true, ".xlsx": true, ".xls": true, ".docx": true,
			".msg": true, ".eml": true, ".rtf": true,
			".png": true, ".jpg": true, ".jpeg": true, ".tiff": true, ".tif": true,
		}

		if !supportedExts[ext] {
			return nil
		}

		// Determine document type from path
		pathLower := strings.ToLower(path)
		switch {
		case strings.Contains(pathLower, "/rfq/") || strings.Contains(pathLower, "\\rfq\\"):
			result["rfq"] = append(result["rfq"], path)
		case strings.Contains(pathLower, "/offer/") || strings.Contains(pathLower, "\\offer\\"):
			result["offer"] = append(result["offer"], path)
		case strings.Contains(pathLower, "/execution/") || strings.Contains(pathLower, "\\execution\\") ||
			strings.Contains(pathLower, "/project execution/") || strings.Contains(pathLower, "\\project execution\\"):
			result["execution"] = append(result["execution"], path)
		default:
			result["other"] = append(result["other"], path)
		}

		return nil
	})

	return result, err
}

// ProcessProjectFolder processes a single project folder (e.g., "01 NORTHGRID-FIT(FEED)")
func (e *ETLService) ProcessProjectFolder(folderPath string) (*OCRBatchResult, error) {
	folderName := filepath.Base(folderPath)
	log.Printf("📁 Processing project folder: %s", folderName)

	// Extract folder number from name (e.g., "01 NORTHGRID-FIT(FEED)" -> "1")
	folderNo := extractFolderNumber(folderName)

	// Get opportunity data if available
	var opp *OpportunityRecord
	if e.opportunities != nil {
		opp = e.opportunities[folderNo]
	}

	// Scan for documents
	filesByType, err := e.ScanFolder(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan folder: %w", err)
	}

	result := &OCRBatchResult{
		FolderPath:  folderPath,
		FolderNo:    folderNo,
		FolderName:  folderName,
		Opportunity: opp,
		ProcessedAt: time.Now(),
		Files:       []OCRFileResult{},
	}

	// Process each document type in order: RFQ first, then Offer, then Execution
	for _, docType := range []string{"rfq", "offer", "execution", "other"} {
		files := filesByType[docType]
		for _, filePath := range files {
			result.TotalFiles++

			fileResult := OCRFileResult{
				FilePath: filePath,
				FileName: filepath.Base(filePath),
			}

			startTime := time.Now()
			ocrResult, err := e.ocrService.ProcessDocument(filePath, docType)
			fileResult.ProcessingMS = time.Since(startTime).Milliseconds()

			if err != nil {
				fileResult.Success = false
				fileResult.Error = err.Error()
				result.ErrorCount++
			} else if ocrResult != nil {
				fileResult.Success = ocrResult.Success
				fileResult.Engine = ocrResult.Engine
				if ocrResult.Success {
					fileResult.Text = ocrResult.Text
					fileResult.ExtractedData = ocrResult.ExtractedData
					result.SuccessCount++
				} else {
					fileResult.Error = ocrResult.Error
					result.ErrorCount++
				}
			}

			result.Files = append(result.Files, fileResult)
		}
	}

	log.Printf("✅ Folder %s: %d/%d files processed successfully",
		folderNo, result.SuccessCount, result.TotalFiles)

	return result, nil
}

// ProcessAllProjects processes all project folders in the base path
func (e *ETLService) ProcessAllProjects(outputDir string) error {
	log.Printf("🚀 Starting batch OCR processing: %s", e.basePath)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find all project folders (numbered folders like "01 ...", "02 ...", etc.)
	entries, err := os.ReadDir(e.basePath)
	if err != nil {
		return fmt.Errorf("failed to read base path: %w", err)
	}

	var projectFolders []string
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			// Check if folder name starts with a number
			if len(name) > 0 && (name[0] >= '0' && name[0] <= '9') {
				projectFolders = append(projectFolders, filepath.Join(e.basePath, name))
			}
		}
	}

	log.Printf("📁 Found %d project folders to process", len(projectFolders))

	allResults := make([]*OCRBatchResult, 0, len(projectFolders))
	totalFiles := 0
	totalSuccess := 0
	totalErrors := 0

	for i, folder := range projectFolders {
		log.Printf("\n[%d/%d] Processing: %s", i+1, len(projectFolders), filepath.Base(folder))

		result, err := e.ProcessProjectFolder(folder)
		if err != nil {
			log.Printf("❌ Error processing folder: %v", err)
			continue
		}

		allResults = append(allResults, result)
		totalFiles += result.TotalFiles
		totalSuccess += result.SuccessCount
		totalErrors += result.ErrorCount

		// Save individual folder result as JSON
		folderOutputPath := filepath.Join(outputDir, fmt.Sprintf("project_%s.json", result.FolderNo))
		if err := saveJSONFile(folderOutputPath, result); err != nil {
			log.Printf("⚠️ Failed to save result JSON: %v", err)
		}
	}

	// Save combined results
	summaryPath := filepath.Join(outputDir, "batch_summary.json")
	summary := map[string]any{
		"processed_at":   time.Now(),
		"total_projects": len(allResults),
		"total_files":    totalFiles,
		"success_count":  totalSuccess,
		"error_count":    totalErrors,
		"success_rate":   float64(totalSuccess) / float64(totalFiles) * 100,
		"projects":       allResults,
	}
	if err := saveJSONFile(summaryPath, summary); err != nil {
		log.Printf("⚠️ Failed to save summary JSON: %v", err)
	}

	log.Printf("\n"+
		"═══════════════════════════════════════════════════════════════════\n"+
		"📊 BATCH OCR COMPLETE\n"+
		"═══════════════════════════════════════════════════════════════════\n"+
		"  Projects: %d\n"+
		"  Total Files: %d\n"+
		"  Successful: %d (%.1f%%)\n"+
		"  Errors: %d\n"+
		"  Output: %s\n"+
		"═══════════════════════════════════════════════════════════════════\n",
		len(allResults), totalFiles, totalSuccess,
		float64(totalSuccess)/float64(totalFiles)*100,
		totalErrors, outputDir)

	return nil
}

// ExtractCustomersAndSuppliers extracts unique customers and suppliers from OCR results
func (e *ETLService) ExtractCustomersAndSuppliers(resultsDir string) ([]ExtractedCustomer, []ExtractedSupplier, error) {
	customers := make(map[string]*ExtractedCustomer)
	suppliers := make(map[string]*ExtractedSupplier)

	// Known (synthetic) customers used for demo-data normalization.
	knownCustomers := map[string]string{
		"gulf smelting":      "Gulf Smelting Co.",
		"national petroleum": "National Petroleum Co.",
		"delta petro":        "Delta Petrochemicals",
		"meadow dairy":       "Meadow Dairy",
		"summit light":       "Summit Light Metals",
		"vertex":             "Vertex Energy",
		"bluewave":           "BlueWave Marine",
		"aquapure":           "AquaPure Technologies",
		"coastal jv":         "Coastal JV W.L.L.",
		"eastside":           "Eastside Wastewater",
		"crescent":           "Crescent Trading",
		"north grid":         "North Grid Authority",
		"pinnacle":           "Pinnacle O&M",
		"intercon":           "Intercon Group",
		"metalworks":         "Metalworks Services",
		"meridian":           "Meridian Systems",
		"stonewell":          "Stonewell Systems",
	}

	// Known (synthetic) suppliers used for demo-data normalization.
	_ = map[string]string{
		"rhine":     "Rhine Instruments",
		"oxan":      "Oxan Analytics",
		"helvetia":  "Helvetia Metering",
		"helix":     "Helix Automation",
		"northwind": "Northwind Controls",
		"apex":      "Apex Process",
		"meridian":  "Meridian Systems",
		"volta":     "Volta Electric",
	}

	// Process each opportunity
	for _, opp := range e.opportunities {
		if opp.EndUser != "" {
			normalized := normalizeCompanyName(opp.EndUser, knownCustomers)
			if c, exists := customers[normalized]; exists {
				c.FolderNumbers = append(c.FolderNumbers, opp.FolderNo)
				c.Sources = append(c.Sources, "opportunities_excel")
			} else {
				customers[normalized] = &ExtractedCustomer{
					Name:           opp.EndUser,
					NormalizedName: normalized,
					FolderNumbers:  []string{opp.FolderNo},
					Sources:        []string{"opportunities_excel"},
				}
			}
		}
	}

	// Add Rhine Instruments as primary supplier (based on project context)
	suppliers["Rhine Instruments"] = &ExtractedSupplier{
		Name:           "Rhine Instruments",
		NormalizedName: "Rhine Instruments",
		Sources:        []string{"known_supplier"},
		Products:       []string{"Flow meters", "Level transmitters", "Temperature transmitters", "Pressure transmitters", "Analytical instruments"},
	}

	// Convert maps to slices
	customerList := make([]ExtractedCustomer, 0, len(customers))
	for _, c := range customers {
		customerList = append(customerList, *c)
	}

	supplierList := make([]ExtractedSupplier, 0, len(suppliers))
	for _, s := range suppliers {
		supplierList = append(supplierList, *s)
	}

	return customerList, supplierList, nil
}

// Helper functions

func extractFolderNumber(folderName string) string {
	// Extract leading numbers from folder name
	re := regexp.MustCompile(`^(\d+)`)
	match := re.FindStringSubmatch(folderName)
	if len(match) > 1 {
		// Remove leading zeros
		num, err := strconv.Atoi(match[1])
		if err == nil {
			return strconv.Itoa(num)
		}
		return match[1]
	}
	return "0"
}

func normalizeCompanyName(name string, known map[string]string) string {
	nameLower := strings.ToLower(name)
	for key, normalized := range known {
		if strings.Contains(nameLower, key) {
			return normalized
		}
	}
	// Default: title case the name
	return strings.TrimSpace(name)
}

// saveJSONFile saves data to a JSON file (using unique name to avoid conflict with archaeologist.go)
func saveJSONFile(path string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// GenerateSeedData generates database seed data from extracted information
func (e *ETLService) GenerateSeedData() (map[string]any, error) {
	customers, suppliers, err := e.ExtractCustomersAndSuppliers("")
	if err != nil {
		return nil, err
	}

	// Convert opportunities to RFQs
	rfqs := make([]map[string]any, 0)
	for _, opp := range e.opportunities {
		rfq := map[string]any{
			"id":            uuid.New().String(),
			"rfq_number":    fmt.Sprintf("RFQ-%s-25", opp.FolderNo),
			"customer_name": opp.EndUser,
			"project_name":  opp.FolderName,
			"description":   opp.Title,
			"status":        mapStatusToRFQStatus(opp.Status),
			"received_date": opp.QuoteDate,
			"value_bhd":     opp.ValueBHD,
			"owner":         opp.Owner,
			"folder_no":     opp.FolderNo,
		}
		rfqs = append(rfqs, rfq)
	}

	return map[string]any{
		"customers":     customers,
		"suppliers":     suppliers,
		"rfqs":          rfqs,
		"opportunities": e.opportunities,
		"generated_at":  time.Now(),
	}, nil
}

func mapStatusToRFQStatus(tallyStatus string) string {
	statusLower := strings.ToLower(tallyStatus)
	switch {
	case strings.Contains(statusLower, "payment received") || strings.Contains(statusLower, "completed"):
		return "Won"
	case strings.Contains(statusLower, "lost"):
		return "Lost"
	case strings.Contains(statusLower, "pending") || strings.Contains(statusLower, "submitted"):
		return "Pending"
	default:
		return "Pending"
	}
}
