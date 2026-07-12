//go:build manual

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/nguyenthenguyen/docx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type oneDriveSeedOpportunityRow struct {
	FolderName           string
	FolderPath           string
	RawFolderNumber      string
	FolderPrefix         string
	SequenceToken        string
	Year                 int
	DerivedPipelineKey   string
	Title                string
	InstrumentType       string
	Stage                string
	CustomerMatchName    string
	CustomerMatchID      string
	CustomerMatchScore   float64
	CustomerMatchReason  string
	FileCount            int
	CostingFileCount     int
	PDFCount             int
	PrimaryCostingPath   string
	PrimaryRevision      int
	PrimaryOptionLabel   string
	PrimaryLineItemCount int
	PrimarySubtotalBHD   float64
	PrimaryVATBHD        float64
	PrimaryGrandTotalBHD float64
	PONumber             string
	InvoiceNumber        string
	ScanWarnings         string
}

type oneDriveSeedCostingRow struct {
	FolderName          string
	RawFolderNumber     string
	DerivedPipelineKey  string
	CostingPath         string
	RelativeCostingPath string
	Revision            int
	OptionLabel         string
	IsPrimary           bool
	Customer            string
	Reference           string
	Date                string
	PaymentTerms        string
	DeliveryTerms       string
	CountryOfOrigin     string
	LineItemCount       int
	SubtotalBHD         float64
	VATBHD              float64
	GrandTotalBHD       float64
	Warnings            string
}

type oneDriveSeedLineItemRow struct {
	FolderName          string
	RawFolderNumber     string
	DerivedPipelineKey  string
	CostingPath         string
	RelativeCostingPath string
	Revision            int
	OptionLabel         string
	LineNumber          int
	Supplier            string
	Equipment           string
	Model               string
	Specification       string
	Quantity            float64
	FobBHD              float64
	FreightBHD          float64
	TotalCostBHD        float64
	MarkupPercent       float64
	SuggestedPriceBHD   float64
	LineTotalBHD        float64
}

type oneDriveSupplierDocumentRow struct {
	FolderName         string
	RawFolderNumber    string
	DerivedPipelineKey string
	FilePath           string
	RelativePath       string
	FileName           string
	Extension          string
	DocumentType       string
	SupplierName       string
	InvoiceNumber      string
	InvoiceDate        string
	SalesOrderNumber   string
	SalesOrderDate     string
	CustomerReference  string
	OrderNumber        string
	DeliveryNote       string
	Currency           string
	Subtotal           float64
	Freight            float64
	VATRate            float64
	VATAmount          float64
	GrandTotal         float64
	PaymentTerms       string
	DeliveryTerms      string
	LineItemCount      int
	FirstItemCode      string
	FirstItemDesc      string
	Warnings           string
}

type oneDriveSeedAuditRow struct {
	FolderName            string
	RawFolderNumber       string
	DerivedPipelineKey    string
	CustomerMatchName     string
	Title                 string
	MatchedDBByFolderName int
	MatchedDBByRawKey     int
	MatchedDBByDerivedKey int
	DBFolderNumbers       string
	DBSources             string
	DBStages              string
	DBCustomerNames       string
	AuditStatus           string
}

type oneDriveSeedDBOpportunity struct {
	ID           string
	FolderNumber string
	FolderName   string
	CustomerName string
	Title        string
	Stage        string
	Source       string
	Year         int
}

type oneDriveSeedSummary struct {
	GeneratedAt               time.Time `json:"generated_at"`
	RootPath                  string    `json:"root_path"`
	DatabasePath              string    `json:"database_path"`
	ImportYear                int       `json:"import_year"`
	ImportSource              string    `json:"import_source"`
	DealsScanned              int       `json:"deals_scanned"`
	DealsWithCostings         int       `json:"deals_with_costings"`
	CostingFilesParsed        int       `json:"costing_files_parsed"`
	LineItemsExtracted        int       `json:"line_items_extracted"`
	SupplierDocumentsScanned  int       `json:"supplier_documents_scanned"`
	SupplierDocumentsParsed   int       `json:"supplier_documents_parsed"`
	UnresolvedCustomerMatches int       `json:"unresolved_customer_matches"`
	CurrentDBOpportunityRows  int       `json:"current_db_opportunity_rows"`
	CurrentDBOneDriveRows     int       `json:"current_db_onedrive_rows"`
	MissingInDB               int       `json:"missing_in_db"`
	MultipleDBMatches         int       `json:"multiple_db_matches"`
	OutputDir                 string    `json:"output_dir"`
}

type oneDriveFolderIdentity struct {
	RawFolderNumber    string
	FolderPrefix       string
	SequenceToken      string
	Year               int
	DerivedPipelineKey string
	Title              string
}

func TestManualExportOneDriveSeed(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ONEDRIVE_IMPORT_ROOT")) == "" {
		t.Skip("set ONEDRIVE_IMPORT_ROOT to stage a OneDrive folder")
	}

	rootPath := defaultOneDriveImportRoot()
	dbPath := resolveOneDriveImportDBPath()
	importYear := resolveOneDriveImportYear(rootPath)
	importSource := oneDriveOpportunitySource(importYear)

	stamp := time.Now().Format("20060102_150405")
	outputDir := os.Getenv("ONEDRIVE_SEED_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = filepath.Join("reports", fmt.Sprintf("onedrive_seed_%d_%s", importYear, stamp))
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database %s: %v", dbPath, err)
	}

	app := &App{
		db:                     db,
		cache:                  NewCache(),
		startupImporting:       true,
		startupImportStartTime: time.Now(),
		currentUserID:          "manual-onedrive-seed-export",
		currentUser:            &User{Base: Base{ID: "manual-onedrive-seed-export"}, Username: "manual-onedrive-seed-export"},
	}
	t.Cleanup(app.cache.Stop)

	scan, err := app.ScanOneDrivePaths([]string{rootPath})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	var dbRows []oneDriveSeedDBOpportunity
	if err := db.Raw(`
		SELECT id, folder_number, folder_name, customer_name, title, stage, source, year
		FROM opportunities
		WHERE year = ?
	`, importYear).Scan(&dbRows).Error; err != nil {
		t.Fatalf("failed to load db opportunities: %v", err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir %s: %v", outputDir, err)
	}

	sort.Slice(scan.Deals, func(i, j int) bool {
		return strings.ToUpper(scan.Deals[i].FolderName) < strings.ToUpper(scan.Deals[j].FolderName)
	})

	var oppRows []oneDriveSeedOpportunityRow
	var costingRows []oneDriveSeedCostingRow
	var lineItemRows []oneDriveSeedLineItemRow
	var supplierDocumentRows []oneDriveSupplierDocumentRow
	var auditRows []oneDriveSeedAuditRow

	dealsWithCostings := 0
	totalParsedCostings := 0
	totalLineItems := 0
	totalSupplierDocuments := 0
	totalParsedSupplierDocuments := 0
	unresolvedMatches := 0
	missingInDB := 0
	multipleDBMatches := 0

	for _, deal := range scan.Deals {
		identity := deriveOneDriveFolderIdentity(deal.FolderName)
		if identity.Year == 0 {
			identity.Year = inferYearFromPath(deal.RootPath, deal.FolderPath)
			if identity.DerivedPipelineKey == "" && identity.FolderPrefix != "" && identity.SequenceToken != "" && identity.Year != 0 {
				identity.DerivedPipelineKey = derivePipelineKey(identity.FolderPrefix, identity.SequenceToken, identity.Year)
			}
		}
		bestCustomerName := ""
		bestCustomerID := ""
		bestCustomerScore := 0.0
		bestCustomerReason := ""
		if len(deal.CustomerMatches) > 0 {
			bestCustomerName = deal.CustomerMatches[0].BusinessName
			bestCustomerID = deal.CustomerMatches[0].CustomerID
			bestCustomerScore = deal.CustomerMatches[0].Score
			bestCustomerReason = deal.CustomerMatches[0].MatchReason
		} else {
			unresolvedMatches++
		}

		pdfCount := 0
		var costingFiles []DiscoveredFile
		for _, f := range deal.Files {
			if f.Extension == ".pdf" {
				pdfCount++
			}
			if f.FileType == "costing_sheet" {
				costingFiles = append(costingFiles, f)
			}
			if isSupplierDocumentCandidate(f) {
				totalSupplierDocuments++
				row := parseOneDriveSupplierDocument(deal.FolderPath, deal.FolderName, identity, f)
				if strings.TrimSpace(row.Warnings) == "" {
					totalParsedSupplierDocuments++
				}
				supplierDocumentRows = append(supplierDocumentRows, row)
			}
		}
		if len(costingFiles) > 0 {
			dealsWithCostings++
		}

		parsedCostings := make([]parsedCostingBundle, 0, len(costingFiles))
		var costingWarnings []string
		for _, f := range costingFiles {
			parsed, parseErr := ParseCostingSheet(f.FilePath)
			if parseErr != nil {
				costingWarnings = append(costingWarnings, fmt.Sprintf("%s: %v", filepath.Base(f.FilePath), parseErr))
				continue
			}
			totalParsedCostings++
			totalLineItems += len(parsed.LineItems)
			bundle := buildParsedCostingBundle(deal.FolderPath, identity, f, parsed)
			parsedCostings = append(parsedCostings, bundle)

			costingRows = append(costingRows, oneDriveSeedCostingRow{
				FolderName:          deal.FolderName,
				RawFolderNumber:     identity.RawFolderNumber,
				DerivedPipelineKey:  identity.DerivedPipelineKey,
				CostingPath:         f.FilePath,
				RelativeCostingPath: bundle.RelativeCostingPath,
				Revision:            bundle.Revision,
				OptionLabel:         bundle.OptionLabel,
				Customer:            strings.TrimSpace(parsed.Metadata.Customer),
				Reference:           strings.TrimSpace(parsed.Metadata.Reference),
				Date:                strings.TrimSpace(parsed.Metadata.Date),
				PaymentTerms:        strings.TrimSpace(parsed.Metadata.PaymentTerms),
				DeliveryTerms:       strings.TrimSpace(parsed.Metadata.DeliveryTerms),
				CountryOfOrigin:     strings.TrimSpace(parsed.Metadata.CountryOfOrigin),
				LineItemCount:       len(parsed.LineItems),
				SubtotalBHD:         parsed.Totals.Subtotal,
				VATBHD:              parsed.Totals.VatAmount,
				GrandTotalBHD:       parsed.Totals.GrandTotal,
				Warnings:            strings.Join(parsed.Warnings, " | "),
			})

			for idx, item := range parsed.LineItems {
				lineItemRows = append(lineItemRows, oneDriveSeedLineItemRow{
					FolderName:          deal.FolderName,
					RawFolderNumber:     identity.RawFolderNumber,
					DerivedPipelineKey:  identity.DerivedPipelineKey,
					CostingPath:         f.FilePath,
					RelativeCostingPath: bundle.RelativeCostingPath,
					Revision:            bundle.Revision,
					OptionLabel:         bundle.OptionLabel,
					LineNumber:          idx + 1,
					Supplier:            strings.TrimSpace(item.Supplier),
					Equipment:           strings.TrimSpace(item.Equipment),
					Model:               strings.TrimSpace(item.Model),
					Specification:       strings.TrimSpace(item.Specification),
					Quantity:            item.Quantity,
					FobBHD:              item.FobBHD,
					FreightBHD:          item.FreightBHD,
					TotalCostBHD:        item.TotalCost,
					MarkupPercent:       item.MarkupPercent,
					SuggestedPriceBHD:   item.SuggestedPriceBHD,
					LineTotalBHD:        item.TotalSuggestedBHD,
				})
			}
		}

		primary := selectPrimaryCosting(parsedCostings)
		for i := range costingRows {
			if costingRows[i].FolderName == deal.FolderName && costingRows[i].CostingPath == primary.CostingPath && primary.CostingPath != "" {
				costingRows[i].IsPrimary = true
			}
		}

		oppRows = append(oppRows, oneDriveSeedOpportunityRow{
			FolderName:           deal.FolderName,
			FolderPath:           deal.FolderPath,
			RawFolderNumber:      identity.RawFolderNumber,
			FolderPrefix:         identity.FolderPrefix,
			SequenceToken:        identity.SequenceToken,
			Year:                 identity.Year,
			DerivedPipelineKey:   identity.DerivedPipelineKey,
			Title:                identity.Title,
			InstrumentType:       deal.InstrumentType,
			Stage:                detectOneDriveStage(deal.Files),
			CustomerMatchName:    bestCustomerName,
			CustomerMatchID:      bestCustomerID,
			CustomerMatchScore:   bestCustomerScore,
			CustomerMatchReason:  bestCustomerReason,
			FileCount:            len(deal.Files),
			CostingFileCount:     len(costingFiles),
			PDFCount:             pdfCount,
			PrimaryCostingPath:   primary.CostingPath,
			PrimaryRevision:      primary.Revision,
			PrimaryOptionLabel:   primary.OptionLabel,
			PrimaryLineItemCount: len(primary.LineItems),
			PrimarySubtotalBHD:   primary.SubtotalBHD,
			PrimaryVATBHD:        primary.VATBHD,
			PrimaryGrandTotalBHD: primary.GrandTotalBHD,
			PONumber:             extractPONumber(deal.Files),
			InvoiceNumber:        extractInvoiceNumber(deal.Files),
			ScanWarnings:         strings.Join(costingWarnings, " | "),
		})

		audit := buildOneDriveAuditRow(identity, deal, bestCustomerName, dbRows)
		auditRows = append(auditRows, audit)
		switch audit.AuditStatus {
		case "missing_in_db":
			missingInDB++
		case "multiple_db_matches":
			multipleDBMatches++
		}
	}

	if err := writeSeedCSV(filepath.Join(outputDir, "opportunities.csv"), oppRows, []string{
		"folder_name", "folder_path", "raw_folder_number", "folder_prefix", "sequence_token", "year",
		"derived_pipeline_key", "title", "instrument_type", "stage", "customer_match_name", "customer_match_id",
		"customer_match_score", "customer_match_reason", "file_count", "costing_file_count", "pdf_count",
		"primary_costing_path", "primary_revision", "primary_option_label", "primary_line_item_count",
		"primary_subtotal_bhd", "primary_vat_bhd", "primary_grand_total_bhd", "po_number", "invoice_number",
		"scan_warnings",
	}, func(row oneDriveSeedOpportunityRow) []string {
		return []string{
			row.FolderName,
			row.FolderPath,
			row.RawFolderNumber,
			row.FolderPrefix,
			row.SequenceToken,
			intString(row.Year),
			row.DerivedPipelineKey,
			row.Title,
			row.InstrumentType,
			row.Stage,
			row.CustomerMatchName,
			row.CustomerMatchID,
			floatString(row.CustomerMatchScore),
			row.CustomerMatchReason,
			intString(row.FileCount),
			intString(row.CostingFileCount),
			intString(row.PDFCount),
			row.PrimaryCostingPath,
			intString(row.PrimaryRevision),
			row.PrimaryOptionLabel,
			intString(row.PrimaryLineItemCount),
			floatString(row.PrimarySubtotalBHD),
			floatString(row.PrimaryVATBHD),
			floatString(row.PrimaryGrandTotalBHD),
			row.PONumber,
			row.InvoiceNumber,
			row.ScanWarnings,
		}
	}); err != nil {
		t.Fatalf("failed to write opportunities csv: %v", err)
	}

	if err := writeSeedCSV(filepath.Join(outputDir, "costings.csv"), costingRows, []string{
		"folder_name", "raw_folder_number", "derived_pipeline_key", "costing_path", "relative_costing_path",
		"revision", "option_label", "is_primary", "customer", "reference", "date", "payment_terms",
		"delivery_terms", "country_of_origin", "line_item_count", "subtotal_bhd", "vat_bhd", "grand_total_bhd",
		"warnings",
	}, func(row oneDriveSeedCostingRow) []string {
		return []string{
			row.FolderName,
			row.RawFolderNumber,
			row.DerivedPipelineKey,
			row.CostingPath,
			row.RelativeCostingPath,
			intString(row.Revision),
			row.OptionLabel,
			boolString(row.IsPrimary),
			row.Customer,
			row.Reference,
			row.Date,
			row.PaymentTerms,
			row.DeliveryTerms,
			row.CountryOfOrigin,
			intString(row.LineItemCount),
			floatString(row.SubtotalBHD),
			floatString(row.VATBHD),
			floatString(row.GrandTotalBHD),
			row.Warnings,
		}
	}); err != nil {
		t.Fatalf("failed to write costings csv: %v", err)
	}

	if err := writeSeedCSV(filepath.Join(outputDir, "line_items.csv"), lineItemRows, []string{
		"folder_name", "raw_folder_number", "derived_pipeline_key", "costing_path", "relative_costing_path",
		"revision", "option_label", "line_number", "supplier", "equipment", "model", "specification",
		"quantity", "fob_bhd", "freight_bhd", "total_cost_bhd", "markup_percent", "suggested_price_bhd",
		"line_total_bhd",
	}, func(row oneDriveSeedLineItemRow) []string {
		return []string{
			row.FolderName,
			row.RawFolderNumber,
			row.DerivedPipelineKey,
			row.CostingPath,
			row.RelativeCostingPath,
			intString(row.Revision),
			row.OptionLabel,
			intString(row.LineNumber),
			row.Supplier,
			row.Equipment,
			row.Model,
			row.Specification,
			floatString(row.Quantity),
			floatString(row.FobBHD),
			floatString(row.FreightBHD),
			floatString(row.TotalCostBHD),
			floatString(row.MarkupPercent),
			floatString(row.SuggestedPriceBHD),
			floatString(row.LineTotalBHD),
		}
	}); err != nil {
		t.Fatalf("failed to write line items csv: %v", err)
	}

	if err := writeSeedCSV(filepath.Join(outputDir, "supplier_documents.csv"), supplierDocumentRows, []string{
		"folder_name", "raw_folder_number", "derived_pipeline_key", "file_path", "relative_path", "file_name",
		"extension", "document_type", "supplier_name", "invoice_number", "invoice_date", "sales_order_number",
		"sales_order_date", "customer_reference", "order_number", "delivery_note", "currency", "subtotal",
		"freight", "vat_rate", "vat_amount", "grand_total", "payment_terms", "delivery_terms", "line_item_count",
		"first_item_code", "first_item_description", "warnings",
	}, func(row oneDriveSupplierDocumentRow) []string {
		return []string{
			row.FolderName,
			row.RawFolderNumber,
			row.DerivedPipelineKey,
			row.FilePath,
			row.RelativePath,
			row.FileName,
			row.Extension,
			row.DocumentType,
			row.SupplierName,
			row.InvoiceNumber,
			row.InvoiceDate,
			row.SalesOrderNumber,
			row.SalesOrderDate,
			row.CustomerReference,
			row.OrderNumber,
			row.DeliveryNote,
			row.Currency,
			floatString(row.Subtotal),
			floatString(row.Freight),
			floatString(row.VATRate),
			floatString(row.VATAmount),
			floatString(row.GrandTotal),
			row.PaymentTerms,
			row.DeliveryTerms,
			intString(row.LineItemCount),
			row.FirstItemCode,
			row.FirstItemDesc,
			row.Warnings,
		}
	}); err != nil {
		t.Fatalf("failed to write supplier documents csv: %v", err)
	}

	if err := writeSeedCSV(filepath.Join(outputDir, "db_audit.csv"), auditRows, []string{
		"folder_name", "raw_folder_number", "derived_pipeline_key", "customer_match_name", "title",
		"matched_db_by_folder_name", "matched_db_by_raw_key", "matched_db_by_derived_key", "db_folder_numbers",
		"db_sources", "db_stages", "db_customer_names", "audit_status",
	}, func(row oneDriveSeedAuditRow) []string {
		return []string{
			row.FolderName,
			row.RawFolderNumber,
			row.DerivedPipelineKey,
			row.CustomerMatchName,
			row.Title,
			intString(row.MatchedDBByFolderName),
			intString(row.MatchedDBByRawKey),
			intString(row.MatchedDBByDerivedKey),
			row.DBFolderNumbers,
			row.DBSources,
			row.DBStages,
			row.DBCustomerNames,
			row.AuditStatus,
		}
	}); err != nil {
		t.Fatalf("failed to write audit csv: %v", err)
	}

	summary := oneDriveSeedSummary{
		GeneratedAt:               time.Now(),
		RootPath:                  rootPath,
		DatabasePath:              dbPath,
		ImportYear:                importYear,
		ImportSource:              importSource,
		DealsScanned:              len(scan.Deals),
		DealsWithCostings:         dealsWithCostings,
		CostingFilesParsed:        totalParsedCostings,
		LineItemsExtracted:        totalLineItems,
		SupplierDocumentsScanned:  totalSupplierDocuments,
		SupplierDocumentsParsed:   totalParsedSupplierDocuments,
		UnresolvedCustomerMatches: unresolvedMatches,
		CurrentDBOpportunityRows:  len(dbRows),
		CurrentDBOneDriveRows:     countDBRowsBySource(dbRows, importSource),
		MissingInDB:               missingInDB,
		MultipleDBMatches:         multipleDBMatches,
		OutputDir:                 outputDir,
	}
	if err := writeSeedJSON(filepath.Join(outputDir, "summary.json"), summary); err != nil {
		t.Fatalf("failed to write summary json: %v", err)
	}

	t.Logf("seed_export output_dir=%s deals=%d costings=%d line_items=%d supplier_docs=%d supplier_docs_parsed=%d missing_in_db=%d multiple_db_matches=%d unresolved_matches=%d",
		outputDir, len(scan.Deals), totalParsedCostings, totalLineItems, totalSupplierDocuments, totalParsedSupplierDocuments, missingInDB, multipleDBMatches, unresolvedMatches)
}

func isSupplierDocumentCandidate(file DiscoveredFile) bool {
	ext := strings.ToLower(strings.TrimSpace(file.Extension))
	if ext != ".pdf" && ext != ".rtf" && ext != ".docx" {
		return false
	}
	haystack := strings.ToLower(file.FilePath + " " + file.FileName + " " + file.FileType)
	signals := []string{
		"supplier", "invoice", "inv", "quote", "quotation", "sales order", "execution", "rhine instruments",
	}
	for _, signal := range signals {
		if strings.Contains(haystack, signal) {
			return true
		}
	}
	return false
}

func parseOneDriveSupplierDocument(folderPath, folderName string, identity oneDriveFolderIdentity, file DiscoveredFile) oneDriveSupplierDocumentRow {
	relative := strings.TrimPrefix(file.FilePath, folderPath)
	relative = strings.TrimPrefix(relative, string(filepath.Separator))
	row := oneDriveSupplierDocumentRow{
		FolderName:         folderName,
		RawFolderNumber:    identity.RawFolderNumber,
		DerivedPipelineKey: identity.DerivedPipelineKey,
		FilePath:           file.FilePath,
		RelativePath:       relative,
		FileName:           filepath.Base(file.FilePath),
		Extension:          strings.ToLower(file.Extension),
	}

	text, err := extractSupplierDocumentText(file.FilePath, row.Extension)
	if err != nil {
		row.Warnings = err.Error()
		return row
	}
	text = normalizeSeedSpace(text)
	if text == "" {
		row.Warnings = "no extractable text"
		return row
	}

	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "sales order") || strings.Contains(lower, "quotation") || strings.Contains(lower, "quote"):
		row.DocumentType = "SupplierQuote"
	case strings.Contains(lower, "invoice"):
		row.DocumentType = "SupplierInvoice"
	default:
		row.DocumentType = "SupplierDocument"
	}

	row.SupplierName = firstSupplierRegex(text,
		`(?i)(Rhine Instruments[^
,]*)`,
		`(?i)(Oxan Analytics[^\n\r,]*)`,
		`(?i)(Meridian Systems[^\n\r,]*)`,
	)
	row.InvoiceNumber = firstSupplierRegex(text, `(?i)(?:invoice\s*(?:no\.?|number|#)?|inv\.?\s*no\.?)\s*[:#-]?\s*([A-Z0-9][A-Z0-9\/-]{3,})`)
	row.InvoiceDate = firstSupplierRegex(text, `(?i)(?:invoice\s*date|date)\s*[:#-]?\s*(\d{1,2}[./-]\d{1,2}[./-]\d{2,4})`)
	row.SalesOrderNumber = firstSupplierRegex(text, `(?i)sales\s*order\s*(?:no\.?|number|#)?\s*[:#-]?\s*([A-Z0-9][A-Z0-9\/-]{3,})`)
	row.SalesOrderDate = firstSupplierRegex(text, `(?i)sales\s*order\s*date\s*[:#-]?\s*(\d{1,2}[./-]\d{1,2}[./-]\d{2,4})`)
	row.CustomerReference = firstSupplierRegex(text, `(?i)customer\s*(?:reference|ref\.?)\s*[:#-]?\s*([A-Z0-9][A-Z0-9\/-]{2,})`)
	row.OrderNumber = firstSupplierRegex(text, `(?i)(?:your\s*)?order\s*(?:no\.?|number|#)?\s*[:#-]?\s*([A-Z0-9][A-Z0-9\/-]{3,})`)
	row.DeliveryNote = firstSupplierRegex(text, `(?i)delivery\s*note\s*(?:no\.?|number|#)?\s*[:#-]?\s*([A-Z0-9][A-Z0-9\/-]{3,})`)
	row.Currency = firstSupplierRegex(text, `\b(EUR|BHD|USD|GBP|AED|SAR)\b`)
	row.Subtotal = firstSupplierAmount(text, `(?i)(?:subtotal|sub\s*total|net\s*amount|total\s*net)\D{0,30}([0-9][0-9,]*\.?[0-9]*)`)
	row.Freight = firstSupplierAmount(text, `(?i)(?:freight|shipping|transport)\D{0,30}([0-9][0-9,]*\.?[0-9]*)`)
	row.VATRate = firstSupplierAmount(text, `(?i)(?:vat|tax)\D{0,15}([0-9]{1,2}(?:\.[0-9]+)?)\s*%`)
	row.VATAmount = firstSupplierAmount(text, `(?i)(?:vat|tax)\D{0,30}([0-9][0-9,]*\.?[0-9]*)`)
	row.GrandTotal = firstSupplierAmount(text, `(?i)(?:grand\s*total|total\s*amount|amount\s*due|invoice\s*total)\D{0,40}([0-9][0-9,]*\.?[0-9]*)`)
	row.PaymentTerms = firstSupplierRegex(text, `(?i)payment\s*terms?\s*[:#-]?\s*([^.;]{3,80})`)
	row.DeliveryTerms = firstSupplierRegex(text, `(?i)delivery\s*terms?\s*[:#-]?\s*([^.;]{3,80})`)
	row.LineItemCount = countSupplierLineItems(text)
	row.FirstItemCode, row.FirstItemDesc = firstSupplierItem(text)

	var warnings []string
	if row.DocumentType == "SupplierInvoice" && row.InvoiceNumber == "" {
		warnings = append(warnings, "invoice number not found")
	}
	if row.GrandTotal == 0 {
		warnings = append(warnings, "grand total not found")
	}
	if row.LineItemCount == 0 {
		warnings = append(warnings, "line items not confidently detected")
	}
	row.Warnings = strings.Join(warnings, " | ")
	return row
}

func extractSupplierDocumentText(path, ext string) (string, error) {
	switch ext {
	case ".pdf":
		return extractVectorPDF(path)
	case ".rtf":
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return extractTextFromRTF(string(raw)), nil
	case ".docx":
		reader, err := docx.ReadDocxFile(path)
		if err != nil {
			return "", err
		}
		defer reader.Close()
		return reader.Editable().GetContent(), nil
	default:
		return "", fmt.Errorf("unsupported supplier document type %s", ext)
	}
}

func firstSupplierRegex(text string, patterns ...string) string {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(text); len(match) > 1 {
			return normalizeSeedSpace(match[1])
		}
	}
	return ""
}

func firstSupplierAmount(text, pattern string) float64 {
	raw := firstSupplierRegex(text, pattern)
	raw = strings.ReplaceAll(raw, ",", "")
	value, _ := strconv.ParseFloat(raw, 64)
	return value
}

func countSupplierLineItems(text string) int {
	re := regexp.MustCompile(`(?m)(?:^|\s)(?:\d{1,3}|[A-Z]{1,5}\d{3,})\s+[A-Z0-9][A-Z0-9/-]*\s+.{8,}?`)
	matches := re.FindAllString(text, -1)
	return len(matches)
}

func firstSupplierItem(text string) (string, string) {
	re := regexp.MustCompile(`(?m)(?:^|\s)(?:\d{1,3}\s+)?([A-Z]{1,8}[A-Z0-9/-]{3,})\s+([^0-9]{8,120})`)
	if match := re.FindStringSubmatch(text); len(match) > 2 {
		return normalizeSeedSpace(match[1]), normalizeSeedSpace(match[2])
	}
	return "", ""
}

type parsedCostingBundle struct {
	CostingPath         string
	RelativeCostingPath string
	Revision            int
	OptionLabel         string
	LineItems           []ExcelCostingLineItem
	SubtotalBHD         float64
	VATBHD              float64
	GrandTotalBHD       float64
}

func buildParsedCostingBundle(folderPath string, identity oneDriveFolderIdentity, file DiscoveredFile, parsed *ExcelCostingData) parsedCostingBundle {
	relative := strings.TrimPrefix(file.FilePath, folderPath)
	relative = strings.TrimPrefix(relative, string(filepath.Separator))
	return parsedCostingBundle{
		CostingPath:         file.FilePath,
		RelativeCostingPath: relative,
		Revision:            extractPathRevision(file.FilePath),
		OptionLabel:         extractOptionLabel(file.FilePath),
		LineItems:           parsed.LineItems,
		SubtotalBHD:         parsed.Totals.Subtotal,
		VATBHD:              parsed.Totals.VatAmount,
		GrandTotalBHD:       parsed.Totals.GrandTotal,
	}
}

func selectPrimaryCosting(costings []parsedCostingBundle) parsedCostingBundle {
	if len(costings) == 0 {
		return parsedCostingBundle{}
	}
	best := costings[0]
	bestScore := primaryCostingScore(best)
	for _, candidate := range costings[1:] {
		score := primaryCostingScore(candidate)
		if score > bestScore || (score == bestScore && strings.ToUpper(candidate.RelativeCostingPath) < strings.ToUpper(best.RelativeCostingPath)) {
			best = candidate
			bestScore = score
		}
	}
	return best
}

func primaryCostingScore(c parsedCostingBundle) int {
	score := c.Revision * 1000
	switch {
	case strings.EqualFold(c.OptionLabel, "option-1"):
		score += 40
	case strings.EqualFold(c.OptionLabel, "option-2"):
		score += 30
	case c.OptionLabel == "":
		score += 20
	default:
		score += 10
	}
	if len(c.LineItems) > 0 {
		score += len(c.LineItems)
	}
	return score
}

func deriveOneDriveFolderIdentity(folderName string) oneDriveFolderIdentity {
	trimmed := normalizeSeedSpace(folderName)
	identity := oneDriveFolderIdentity{Title: trimmed}
	re := regexp.MustCompile(`(?i)^([A-Z]{1,8})(?:[-\s]+)(\d{1,3}[A-Z]?)[-\s/]+(\d{2})(.*)$`)
	if m := re.FindStringSubmatch(trimmed); len(m) == 5 {
		prefix := strings.ToUpper(strings.TrimSpace(m[1]))
		seqToken := strings.ToUpper(strings.TrimSpace(m[2]))
		yy := strings.TrimSpace(m[3])
		title := normalizeSeedSpace(strings.TrimLeft(m[4], "-_/ "))
		if title == "" {
			title = trimmed
		}
		year := 2000 + mustSeedAtoi(yy)
		identity.RawFolderNumber = fmt.Sprintf("%s-%s-%s", prefix, seqToken, yy)
		identity.FolderPrefix = prefix
		identity.SequenceToken = seqToken
		identity.Year = year
		identity.DerivedPipelineKey = derivePipelineKey(prefix, seqToken, year)
		identity.Title = title
		return identity
	}

	fallback := parseOneDriveFolderMeta(folderName)
	identity.RawFolderNumber = strings.TrimSpace(fallback.FolderNumber)
	identity.Year = fallback.Year
	identity.Title = normalizeSeedSpace(fallback.Title)
	if m := regexp.MustCompile(`(?i)^([A-Z]{1,8})-(\d{1,3}[A-Z]?)(?:-([A-Z]{2,8}))?$`).FindStringSubmatch(strings.ToUpper(strings.ReplaceAll(identity.RawFolderNumber, " ", "-"))); len(m) >= 3 {
		identity.FolderPrefix = strings.ToUpper(strings.TrimSpace(m[1]))
		identity.SequenceToken = strings.ToUpper(strings.TrimSpace(m[2]))
		if identity.Year != 0 {
			identity.DerivedPipelineKey = derivePipelineKey(identity.FolderPrefix, identity.SequenceToken, identity.Year)
		}
	}
	return identity
}

func derivePipelineKey(prefix, seqToken string, year int) string {
	if year == 0 || seqToken == "" {
		return ""
	}
	numberPart := regexp.MustCompile(`\d+`).FindString(seqToken)
	if numberPart == "" {
		return ""
	}
	n := mustSeedAtoi(numberPart)
	suffix := strings.TrimPrefix(seqToken, numberPart)
	switch prefix {
	case "EH":
		return fmt.Sprintf("%d-%d%s", year, n, suffix)
	case "OTH":
		return fmt.Sprintf("%d-%d%s", year, 300+n, suffix)
	case "SA":
		return fmt.Sprintf("%d-%d%s", year, 250+n, suffix)
	default:
		return fmt.Sprintf("%d-%d%s", year, n, suffix)
	}
}

func buildOneDriveAuditRow(identity oneDriveFolderIdentity, deal DiscoveredDeal, customerName string, dbRows []oneDriveSeedDBOpportunity) oneDriveSeedAuditRow {
	var folderNameMatches []oneDriveSeedDBOpportunity
	var rawKeyMatches []oneDriveSeedDBOpportunity
	var derivedKeyMatches []oneDriveSeedDBOpportunity

	dealFolderNameNorm := normalizeSeedSpace(strings.ToUpper(deal.FolderName))
	rawKeyNorm := normalizeSeedSpace(strings.ToUpper(identity.RawFolderNumber))
	derivedKeyNorm := normalizeSeedSpace(strings.ToUpper(identity.DerivedPipelineKey))

	for _, row := range dbRows {
		if normalizeSeedSpace(strings.ToUpper(row.FolderName)) == dealFolderNameNorm {
			folderNameMatches = append(folderNameMatches, row)
		}
		if rawKeyNorm != "" && normalizeSeedSpace(strings.ToUpper(row.FolderNumber)) == rawKeyNorm {
			rawKeyMatches = append(rawKeyMatches, row)
		}
		if derivedKeyNorm != "" && normalizeSeedSpace(strings.ToUpper(row.FolderNumber)) == derivedKeyNorm {
			derivedKeyMatches = append(derivedKeyMatches, row)
		}
	}

	combined := combineAuditMatches(folderNameMatches, rawKeyMatches, derivedKeyMatches)
	status := "matched"
	switch {
	case len(combined) == 0:
		status = "missing_in_db"
	case len(combined) > 1:
		status = "multiple_db_matches"
	case len(rawKeyMatches) == 0 && len(folderNameMatches) == 0 && len(derivedKeyMatches) == 1:
		status = "derived_key_only"
	case len(folderNameMatches) == 1 && len(derivedKeyMatches) == 1 && folderNameMatches[0].ID != derivedKeyMatches[0].ID:
		status = "split_identity"
	}

	return oneDriveSeedAuditRow{
		FolderName:            deal.FolderName,
		RawFolderNumber:       identity.RawFolderNumber,
		DerivedPipelineKey:    identity.DerivedPipelineKey,
		CustomerMatchName:     customerName,
		Title:                 identity.Title,
		MatchedDBByFolderName: len(folderNameMatches),
		MatchedDBByRawKey:     len(rawKeyMatches),
		MatchedDBByDerivedKey: len(derivedKeyMatches),
		DBFolderNumbers:       strings.Join(uniqueAuditField(combined, func(row oneDriveSeedDBOpportunity) string { return row.FolderNumber }), " | "),
		DBSources:             strings.Join(uniqueAuditField(combined, func(row oneDriveSeedDBOpportunity) string { return row.Source }), " | "),
		DBStages:              strings.Join(uniqueAuditField(combined, func(row oneDriveSeedDBOpportunity) string { return row.Stage }), " | "),
		DBCustomerNames:       strings.Join(uniqueAuditField(combined, func(row oneDriveSeedDBOpportunity) string { return row.CustomerName }), " | "),
		AuditStatus:           status,
	}
}

func combineAuditMatches(groups ...[]oneDriveSeedDBOpportunity) []oneDriveSeedDBOpportunity {
	seen := map[string]bool{}
	var combined []oneDriveSeedDBOpportunity
	for _, group := range groups {
		for _, row := range group {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
			combined = append(combined, row)
		}
	}
	sort.Slice(combined, func(i, j int) bool {
		return strings.ToUpper(combined[i].FolderNumber) < strings.ToUpper(combined[j].FolderNumber)
	})
	return combined
}

func uniqueAuditField(rows []oneDriveSeedDBOpportunity, get func(oneDriveSeedDBOpportunity) string) []string {
	seen := map[string]bool{}
	var out []string
	for _, row := range rows {
		val := strings.TrimSpace(get(row))
		if val == "" || seen[val] {
			continue
		}
		seen[val] = true
		out = append(out, val)
	}
	sort.Strings(out)
	return out
}

func writeSeedCSV[T any](path string, rows []T, header []string, toRecord func(T) []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write(toRecord(row)); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeSeedJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func extractPathRevision(path string) int {
	re := regexp.MustCompile(`(?i)(?:^|[^A-Z0-9])REV(?:ISION)?[-_\s]?(\d{1,2})(?:[^A-Z0-9]|$)|(?:^|[^A-Z0-9])R[-_\s]?(\d{1,2})(?:[^A-Z0-9]|$)`)
	matches := re.FindStringSubmatch(strings.ToUpper(path))
	if len(matches) < 2 {
		return 0
	}
	for _, match := range matches[1:] {
		if strings.TrimSpace(match) == "" {
			continue
		}
		return mustSeedAtoi(match)
	}
	return 0
}

func extractOptionLabel(path string) string {
	re := regexp.MustCompile(`(?i)OPTION[-_\s]?(\d+)`)
	if m := re.FindStringSubmatch(path); len(m) == 2 {
		return fmt.Sprintf("option-%d", mustSeedAtoi(m[1]))
	}
	return ""
}

func normalizeSeedSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func mustSeedAtoi(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return n
}

func countDBRowsBySource(rows []oneDriveSeedDBOpportunity, source string) int {
	count := 0
	for _, row := range rows {
		if row.Source == source {
			count++
		}
	}
	return count
}

func intString(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}

func floatString(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
