package excel

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	// Metadata rows
	ROW_DATE        = 2
	ROW_PREPARED_BY = 3
	ROW_CUSTOMER    = 4
	ROW_REFERENCE   = 5

	// Product data rows
	ROW_SUPPLIER        = 7
	ROW_EQUIPMENT       = 8
	ROW_MODEL           = 9
	ROW_SPECIFICATION   = 10
	ROW_QUANTITY        = 11
	ROW_FOB_EUR         = 12
	ROW_FREIGHT_EUR     = 13
	ROW_EXCHANGE_RATE   = 14
	ROW_FOB_BHD         = 15
	ROW_FREIGHT_BHD     = 16
	ROW_CNF             = 17
	ROW_INSURANCE       = 18
	ROW_CUSTOMS         = 19
	ROW_LANDED_COST     = 20
	ROW_HANDLING        = 21
	ROW_FINANCE_CHARGES = 22
	ROW_OTHER_COSTS     = 23
	ROW_TOTAL_COST      = 24
	ROW_MARKUP_PERCENT  = 25
	ROW_SELLING_PRICE   = 26
	ROW_SUGGESTED_PRICE = 27
	ROW_TOTAL_SUGGESTED = 28
	ROW_VAT_AMOUNT      = 29
	ROW_GRAND_TOTAL     = 30

	// Column constants
	COL_LABELS        = 1  // Column A - Row labels
	COL_VALUES        = 2  // Column B - Metadata values
	COL_FIRST_PRODUCT = 3  // Column C - First product
	COL_MAX_PRODUCTS  = 50 // Scan up to column 50 for products
	COL_TOTALS        = 13 // Column M - Totals (in standard template)

	// Sheet names
	SHEET_COSTING = "Costing Sheet"
)

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// ExcelCostingData represents the complete parsed data from a costing sheet
type ExcelCostingData struct {
	FilePath  string                 `json:"file_path"`
	FileName  string                 `json:"file_name"`
	Metadata  ExcelCostingMetadata   `json:"metadata"`
	LineItems []ExcelCostingLineItem `json:"line_items"`
	Totals    ExcelCostingTotals     `json:"totals"`
	ParsedAt  time.Time              `json:"parsed_at"`
	Warnings  []string               `json:"warnings"`
}

// ExcelCostingMetadata contains header information from the costing sheet
type ExcelCostingMetadata struct {
	Date            string `json:"date"`
	FolderNumber    string `json:"folder_number"`
	EstDelivery     string `json:"est_delivery"`
	PreparedBy      string `json:"prepared_by"`
	CostingID       string `json:"costing_id"`
	DeliveryTerms   string `json:"delivery_terms"`
	Customer        string `json:"customer"`
	ContactPerson   string `json:"contact_person"`
	OrderType       string `json:"order_type"`
	Reference       string `json:"reference"`
	PaymentTerms    string `json:"payment_terms"`
	CountryOfOrigin string `json:"country_of_origin"`
}

// ExcelCostingLineItem represents a single product/line in the costing
type ExcelCostingLineItem struct {
	ProductNumber     int     `json:"product_number"`
	ColumnIndex       int     `json:"column_index"`
	Supplier          string  `json:"supplier"`
	Equipment         string  `json:"equipment"`
	Model             string  `json:"model"`
	Specification     string  `json:"specification"`
	Quantity          float64 `json:"quantity"`
	FobEUR            float64 `json:"fob_eur"`
	FreightEUR        float64 `json:"freight_eur"`
	ExchangeRate      float64 `json:"exchange_rate"`
	FobBHD            float64 `json:"fob_bhd"`
	FreightBHD        float64 `json:"freight_bhd"`
	CnfBHD            float64 `json:"cnf_bhd"`
	Insurance         float64 `json:"insurance"`
	Customs           float64 `json:"customs"`
	LandedCost        float64 `json:"landed_cost"`
	Handling          float64 `json:"handling"`
	FinanceCharges    float64 `json:"finance_charges"`
	OtherCosts        float64 `json:"other_costs"`
	TotalCost         float64 `json:"total_cost"`
	MarkupPercent     float64 `json:"markup_percent"`
	SellingPriceBHD   float64 `json:"selling_price_bhd"`
	SuggestedPriceBHD float64 `json:"suggested_price_bhd"`
	TotalSuggestedBHD float64 `json:"total_suggested_bhd"`
}

// ExcelCostingTotals contains the summary totals
type ExcelCostingTotals struct {
	Subtotal   float64 `json:"subtotal"`
	VatPercent float64 `json:"vat_percent"`
	VatAmount  float64 `json:"vat_amount"`
	GrandTotal float64 `json:"grand_total"`
}

// ExcelImportResult represents the result of importing a single file
type ExcelImportResult struct {
	FilePath    string  `json:"file_path"`
	FileName    string  `json:"file_name"`
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
	ItemCount   int     `json:"item_count"`
	GrandTotal  float64 `json:"grand_total"`
	Customer    string  `json:"customer"`
	OfferNumber string  `json:"offer_number"`
}

// ExcelBatchImportResult represents the result of batch importing
type ExcelBatchImportResult struct {
	TotalFiles     int                 `json:"total_files"`
	Successful     int                 `json:"successful"`
	Failed         int                 `json:"failed"`
	Results        []ExcelImportResult `json:"results"`
	TotalLineItems int                 `json:"total_line_items"`
	TotalValue     float64             `json:"total_value"`
}

type costingRowMap struct {
	Supplier       int
	Equipment      int
	Model          int
	Specification  int
	Quantity       int
	FobEUR         int
	FreightEUR     int
	ExchangeRate   int
	FobBHD         int
	FreightBHD     int
	Cnf            int
	Insurance      int
	Customs        int
	LandedCost     int
	Handling       int
	FinanceCharges int
	OtherCosts     int
	TotalCost      int
	Markup         int
	SellingPrice   int
	SuggestedPrice int
	TotalSuggested int
	SummaryTotal   int
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// isPlaceholder checks if a cell value is a template placeholder like "[Product 1]"
func isPlaceholder(s string) bool {
	trimmed := strings.TrimSpace(s)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

// isEmpty checks if a string is empty or whitespace
func isEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

func cleanProductCell(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || isPlaceholder(trimmed) || isCostingSummaryLabel(trimmed) || isCostingTemplatePlaceholder(trimmed) {
		return ""
	}
	return trimmed
}

func isCostingTemplatePlaceholder(s string) bool {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
	normalized = strings.TrimSuffix(normalized, " -")
	normalized = strings.TrimSuffix(normalized, "-")
	switch normalized {
	case "principle name", "principal name", "supplier name", "supplier", "equipment", "model", "specification":
		return true
	}
	return regexp.MustCompile(`^line item\s+\d+$`).MatchString(normalized)
}

func firstNonEmptyCostingValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// isCostingSummaryLabel skips summary columns that appear to the right of real
// line items in several PH master files.
func isCostingSummaryLabel(s string) bool {
	normalized := strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
	switch normalized {
	case "TOTAL", "TOTAL FOR ORDER", "SUBTOTAL", "GRAND TOTAL", "TOTAL ORDER":
		return true
	default:
		return false
	}
}

// parseFloat safely parses a float from a cell value
func parseExcelFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || isPlaceholder(s) {
		return 0
	}
	// Remove currency symbols and thousands separators
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "BHD", "")
	s = strings.ReplaceAll(s, "EUR", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.TrimSpace(s)

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func normalizeCostingLabel(s string) string {
	replacer := strings.NewReplacer(
		"\n", " ",
		"\r", " ",
		"\t", " ",
		"-", " ",
		"_", " ",
		"/", " ",
		"(", " ",
		")", " ",
		".", " ",
		":", " ",
		"@", " ",
		"&", " ",
	)
	cleaned := strings.ToUpper(replacer.Replace(strings.TrimSpace(s)))
	return strings.Join(strings.Fields(cleaned), "")
}

func metadataLabelLike(s string) bool {
	switch normalizeCostingLabel(s) {
	case "", "REFERENCE", "EMAIL", "DATE", "PREPAREDBY", "CUSTOMER", "CONTACTPERSON", "CHOOSECONTACTPERSON",
		"ORDERTYPE", "PAYMENTTERMS", "COUNTRYOFORIGIN", "ESTDELIVERY", "DELIVERYTERMS", "FOLDERNUMBER":
		return true
	default:
		return false
	}
}

func normalizeExcelDateValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if value, err := strconv.ParseFloat(raw, 64); err == nil && value > 20000 && value < 60000 {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		return base.AddDate(0, 0, int(value)).Format("2006-01-02")
	}

	return raw
}

func getCellValue(f *excelize.File, sheetName string, col int, row int) string {
	if row <= 0 || col <= 0 {
		return ""
	}
	ref, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}
	value, _ := f.GetCellValue(sheetName, ref)
	return strings.TrimSpace(value)
}

func getFloatCellValue(f *excelize.File, sheetName string, col int, row int) float64 {
	return parseExcelFloat(getCellValue(f, sheetName, col, row))
}

func getPositiveNumericRowValues(f *excelize.File, sheetName string, row int) []float64 {
	if row <= 0 {
		return nil
	}

	values := make([]float64, 0, 4)
	for col := 1; col <= COL_MAX_PRODUCTS; col++ {
		value := parseExcelFloat(getCellValue(f, sheetName, col, row))
		if value > 0 {
			values = append(values, value)
		}
	}

	return values
}

func getRightmostPositiveNumericRowValue(f *excelize.File, sheetName string, row int) float64 {
	if row <= 0 {
		return 0
	}

	for col := COL_MAX_PRODUCTS; col >= 1; col-- {
		value := parseExcelFloat(getCellValue(f, sheetName, col, row))
		if value > 0 {
			return value
		}
	}

	return 0
}

func findCostingRows(f *excelize.File, sheetName string) costingRowMap {
	rows := costingRowMap{}
	for row := 1; row <= 80; row++ {
		labelA := normalizeCostingLabel(getCellValue(f, sheetName, 1, row))
		labelB := normalizeCostingLabel(getCellValue(f, sheetName, 2, row))

		switch {
		case labelA == "SUPPLIER":
			rows.Supplier = row
		case labelA == "EQUIPMENT":
			rows.Equipment = row
		case labelA == "MODEL":
			rows.Model = row
		case strings.Contains(labelA, "SPECIFICATION") || strings.Contains(labelA, "ORDERCODE"):
			rows.Specification = row
		case labelB == "QUANTITY" || labelA == "QUANTITY":
			rows.Quantity = row
		case labelA == "EUR" && labelB == "FOB":
			rows.FobEUR = row
		case labelA == "EUR" && strings.Contains(labelB, "FREIGHT"):
			rows.FreightEUR = row
		case strings.Contains(labelB, "EXCHANGERATE") && strings.Contains(labelB, "BHD"):
			rows.ExchangeRate = row
		case labelA == "BHD" && labelB == "FOB":
			rows.FobBHD = row
		case labelA == "BHD" && strings.Contains(labelB, "FREIGHT"):
			rows.FreightBHD = row
		case strings.Contains(labelB, "CF") || strings.Contains(labelB, "CNF"):
			rows.Cnf = row
		case strings.Contains(labelB, "INSURANCE"):
			rows.Insurance = row
		case strings.Contains(labelB, "CUSTOMS"):
			rows.Customs = row
		case strings.Contains(labelB, "LANDEDCOST"):
			rows.LandedCost = row
		case strings.Contains(labelB, "HANDLING"):
			rows.Handling = row
		case strings.Contains(labelB, "FINANCECHARGES"):
			rows.FinanceCharges = row
		case strings.Contains(labelB, "OTHERCOSTS"):
			rows.OtherCosts = row
		case strings.Contains(labelB, "TOTALCOST"):
			rows.TotalCost = row
		case strings.Contains(labelB, "MARKUP"):
			rows.Markup = row
		case strings.Contains(labelA, "SELLINGPRICE"):
			rows.SellingPrice = row
		case strings.Contains(labelA, "SUGGESTEDPRICEPERUNIT"):
			rows.SuggestedPrice = row
		case strings.Contains(labelA, "TOTALSUGGESTEDPRICE"):
			rows.TotalSuggested = row
		case strings.Contains(labelA, "TOTALPOVALUEEXPECTEDFROMCLIENT"):
			rows.SummaryTotal = row
		}
	}

	if rows.Supplier == 0 {
		rows.Supplier = ROW_SUPPLIER
	}
	if rows.Equipment == 0 {
		rows.Equipment = ROW_EQUIPMENT
	}
	if rows.Model == 0 {
		rows.Model = ROW_MODEL
	}
	if rows.Specification == 0 {
		rows.Specification = ROW_SPECIFICATION
	}
	if rows.Quantity == 0 {
		rows.Quantity = ROW_QUANTITY
	}
	if rows.FobEUR == 0 {
		rows.FobEUR = ROW_FOB_EUR
	}
	if rows.FreightEUR == 0 {
		rows.FreightEUR = ROW_FREIGHT_EUR
	}
	if rows.ExchangeRate == 0 {
		rows.ExchangeRate = ROW_EXCHANGE_RATE
	}
	if rows.FobBHD == 0 {
		rows.FobBHD = ROW_FOB_BHD
	}
	if rows.FreightBHD == 0 {
		rows.FreightBHD = ROW_FREIGHT_BHD
	}
	if rows.Cnf == 0 {
		rows.Cnf = ROW_CNF
	}
	if rows.Insurance == 0 {
		rows.Insurance = ROW_INSURANCE
	}
	if rows.Customs == 0 {
		rows.Customs = ROW_CUSTOMS
	}
	if rows.LandedCost == 0 {
		rows.LandedCost = ROW_LANDED_COST
	}
	if rows.Handling == 0 {
		rows.Handling = ROW_HANDLING
	}
	if rows.FinanceCharges == 0 {
		rows.FinanceCharges = ROW_FINANCE_CHARGES
	}
	if rows.OtherCosts == 0 {
		rows.OtherCosts = ROW_OTHER_COSTS
	}
	if rows.TotalCost == 0 {
		rows.TotalCost = ROW_TOTAL_COST
	}
	if rows.SellingPrice == 0 {
		rows.SellingPrice = ROW_SELLING_PRICE
	}
	if rows.SuggestedPrice == 0 {
		rows.SuggestedPrice = ROW_SUGGESTED_PRICE
	}
	if rows.TotalSuggested == 0 {
		rows.TotalSuggested = ROW_TOTAL_SUGGESTED
	}

	return rows
}

// extractOfferNumber extracts offer number from folder name like "101 VERTEX AIT"
func extractOfferNumber(folderName string) string {
	re := regexp.MustCompile(`^(\d+)`)
	matches := re.FindStringSubmatch(folderName)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractCustomerFromFolder extracts customer name from folder name
func extractCustomerFromFolder(folderName string) string {
	re := regexp.MustCompile(`^\d+[-\s]*(.+)$`)
	matches := re.FindStringSubmatch(folderName)
	if len(matches) > 1 {
		// Clean up the customer name
		name := strings.TrimSpace(matches[1])
		// Remove common suffixes like "AIT", "FIT", "LIT", "SP"
		suffixes := []string{"-AIT", " AIT", "-FIT", " FIT", "-LIT", " LIT", "-SP", " SP", "-FEED", " FEED"}
		for _, suffix := range suffixes {
			if strings.HasSuffix(strings.ToUpper(name), strings.ToUpper(suffix)) {
				name = strings.TrimSuffix(name, suffix)
				name = strings.TrimSuffix(name, strings.ToLower(suffix))
				break
			}
		}
		return strings.TrimSpace(name)
	}
	return folderName
}

// ============================================================================
// MAIN PARSING FUNCTIONS
// ============================================================================

// ParseCostingSheet parses an Excel costing sheet and extracts all data
func ParseCostingSheet(filePath string) (*ExcelCostingData, error) {
	log.Printf("📊 Parsing costing sheet: %s", filePath)

	// Open the Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	result := &ExcelCostingData{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		ParsedAt: time.Now(),
		Warnings: []string{},
	}

	// Find the costing sheet
	sheetName := SHEET_COSTING
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if strings.EqualFold(s, SHEET_COSTING) || strings.Contains(strings.ToLower(s), "costing") {
			sheetName = s
			found = true
			break
		}
	}
	if !found && len(sheets) > 1 {
		// Try second sheet (common pattern)
		sheetName = sheets[1]
		result.Warnings = append(result.Warnings, fmt.Sprintf("Costing sheet not found by name, using sheet: %s", sheetName))
	}

	log.Printf("📊 Using sheet: %s", sheetName)

	// Extract metadata
	result.Metadata = extractMetadata(f, sheetName)

	// Extract line items (HORIZONTAL - iterate columns!)
	result.LineItems = extractLineItems(f, sheetName)

	// Extract totals
	result.Totals = extractTotals(f, sheetName, result.LineItems)

	log.Printf("✅ Parsed %d line items, Grand Total: %.3f BHD", len(result.LineItems), result.Totals.GrandTotal)

	return result, nil
}

// extractMetadata extracts header/metadata from the costing sheet
func extractMetadata(f *excelize.File, sheetName string) ExcelCostingMetadata {
	meta := ExcelCostingMetadata{}

	// Row 2: Date (B2), Folder Number (D2), Est Delivery (F2)
	meta.Date = normalizeExcelDateValue(getCellValue(f, sheetName, 2, 2))
	meta.FolderNumber = getCellValue(f, sheetName, 4, 2)
	meta.EstDelivery = getCellValue(f, sheetName, 6, 2)

	// Row 3: Prepared By (B3), Costing ID (D3), Delivery Terms (F3)
	meta.PreparedBy = getCellValue(f, sheetName, 2, 3)
	meta.CostingID = getCellValue(f, sheetName, 4, 3)
	meta.DeliveryTerms = getCellValue(f, sheetName, 6, 3)

	// Row 4: Customer (B4), Contact Person (D4), Order Type (F4)
	meta.Customer = getCellValue(f, sheetName, 2, 4)
	meta.ContactPerson = getCellValue(f, sheetName, 4, 4)
	meta.OrderType = getCellValue(f, sheetName, 6, 4)

	// Row 5: Reference (B5), Payment Terms (D5), Country (F5)
	refA := getCellValue(f, sheetName, 1, 5)
	refB := getCellValue(f, sheetName, 2, 5)
	if !metadataLabelLike(refA) {
		meta.Reference = refA
	} else if !metadataLabelLike(refB) {
		meta.Reference = refB
	}
	meta.PaymentTerms = getCellValue(f, sheetName, 4, 5)
	meta.CountryOfOrigin = getCellValue(f, sheetName, 6, 5)

	return meta
}

// extractLineItems extracts product line items - HORIZONTAL iteration!
func extractLineItems(f *excelize.File, sheetName string) []ExcelCostingLineItem {
	items := []ExcelCostingLineItem{}
	rows := findCostingRows(f, sheetName)

	productNum := 0

	// Iterate COLUMNS from C (col 3) onwards
	for col := COL_FIRST_PRODUCT; col <= COL_MAX_PRODUCTS; col++ {
		rawSupplier := getCellValue(f, sheetName, col, rows.Supplier)
		rawEquipment := getCellValue(f, sheetName, col, rows.Equipment)
		rawModel := getCellValue(f, sheetName, col, rows.Model)
		rawSpecification := getCellValue(f, sheetName, col, rows.Specification)

		supplier := cleanProductCell(rawSupplier)
		equipment := cleanProductCell(rawEquipment)
		model := cleanProductCell(rawModel)
		specification := cleanProductCell(rawSpecification)

		if isCostingSummaryLabel(rawSupplier) || isCostingSummaryLabel(rawEquipment) || isCostingSummaryLabel(rawModel) {
			continue
		}

		quantity := getFloatCellValue(f, sheetName, col, rows.Quantity)
		fobEUR := getFloatCellValue(f, sheetName, col, rows.FobEUR)
		freightEUR := getFloatCellValue(f, sheetName, col, rows.FreightEUR)
		exchangeRate := getFloatCellValue(f, sheetName, col, rows.ExchangeRate)
		fobBHD := getFloatCellValue(f, sheetName, col, rows.FobBHD)
		freightBHD := getFloatCellValue(f, sheetName, col, rows.FreightBHD)
		cnfBHD := getFloatCellValue(f, sheetName, col, rows.Cnf)
		insurance := getFloatCellValue(f, sheetName, col, rows.Insurance)
		customs := getFloatCellValue(f, sheetName, col, rows.Customs)
		landedCost := getFloatCellValue(f, sheetName, col, rows.LandedCost)
		handling := getFloatCellValue(f, sheetName, col, rows.Handling)
		financeCharges := getFloatCellValue(f, sheetName, col, rows.FinanceCharges)
		otherCosts := getFloatCellValue(f, sheetName, col, rows.OtherCosts)
		totalCost := getFloatCellValue(f, sheetName, col, rows.TotalCost)
		markupPercent := getFloatCellValue(f, sheetName, col, rows.Markup)
		sellingPrice := getFloatCellValue(f, sheetName, col, rows.SellingPrice)
		suggestedPrice := getFloatCellValue(f, sheetName, col, rows.SuggestedPrice)
		totalSuggested := getFloatCellValue(f, sheetName, col, rows.TotalSuggested)

		hasDescriptor := firstNonEmptyCostingValue(equipment, model, specification) != ""
		hasCommercialSignal := totalSuggested > 0 || fobEUR > 0 || freightEUR > 0 ||
			fobBHD > 0 || freightBHD > 0 || cnfBHD > 0 || customs > 0 ||
			landedCost > 0 || otherCosts > 0 || totalCost > 0 ||
			sellingPrice > 0 || suggestedPrice > 0
		hasStrongNumericSignal := hasDescriptor && (quantity > 0 || hasCommercialSignal)

		if !hasStrongNumericSignal {
			continue
		}

		productNum++
		item := ExcelCostingLineItem{
			ProductNumber: productNum,
			ColumnIndex:   col,
		}

		item.Supplier = supplier
		item.Equipment = firstNonEmptyCostingValue(equipment, model, specification)
		item.Model = model
		item.Specification = specification

		item.Quantity = quantity
		if item.Quantity == 0 {
			item.Quantity = 1
		}

		item.FobEUR = fobEUR
		item.FreightEUR = freightEUR
		item.ExchangeRate = exchangeRate
		item.FobBHD = fobBHD
		item.FreightBHD = freightBHD
		item.CnfBHD = cnfBHD
		item.Insurance = insurance
		item.Customs = customs
		item.LandedCost = landedCost
		item.Handling = handling
		item.FinanceCharges = financeCharges
		item.OtherCosts = otherCosts
		item.TotalCost = totalCost

		item.MarkupPercent = 0
		if item.TotalCost > 0 {
			item.MarkupPercent = markupPercent
		}

		item.SellingPriceBHD = sellingPrice

		item.SuggestedPriceBHD = suggestedPrice
		if item.SuggestedPriceBHD == 0 {
			item.SuggestedPriceBHD = item.SellingPriceBHD
		}

		item.TotalSuggestedBHD = totalSuggested
		if item.TotalSuggestedBHD == 0 && item.SuggestedPriceBHD > 0 {
			item.TotalSuggestedBHD = item.SuggestedPriceBHD * item.Quantity
		}
		if item.MarkupPercent == 0 && item.TotalCost > 0 && item.TotalSuggestedBHD > 0 {
			item.MarkupPercent = ((item.TotalSuggestedBHD - item.TotalCost) / item.TotalCost) * 100
		}

		items = append(items, item)
	}

	return items
}

// extractTotals extracts the totals from the sheet
func extractTotals(f *excelize.File, sheetName string, items []ExcelCostingLineItem) ExcelCostingTotals {
	rows := findCostingRows(f, sheetName)
	totals := ExcelCostingTotals{
		VatPercent: 10.0, // Bahrain VAT default
	}

	if rows.SummaryTotal > 0 {
		// The client summary block is consistently laid out in columns C/D/E.
		// Some templates also carry other numeric cells later in the same row
		// (for internal cost/profit notes), so we should not treat the last
		// numeric value on the row as the commercial grand total.
		totals.Subtotal = getFloatCellValue(f, sheetName, 3, rows.SummaryTotal)
		totals.VatAmount = getFloatCellValue(f, sheetName, 4, rows.SummaryTotal)
		totals.GrandTotal = getFloatCellValue(f, sheetName, 5, rows.SummaryTotal)
		if totals.Subtotal == 0 || totals.VatAmount == 0 || totals.GrandTotal == 0 {
			summaryValues := getPositiveNumericRowValues(f, sheetName, rows.SummaryTotal)
			if totals.Subtotal == 0 && len(summaryValues) > 0 {
				totals.Subtotal = summaryValues[0]
			}
			if totals.VatAmount == 0 && len(summaryValues) > 1 {
				totals.VatAmount = summaryValues[1]
			}
			if totals.GrandTotal == 0 && len(summaryValues) > 2 {
				totals.GrandTotal = summaryValues[2]
			}
		}
	}

	if totals.Subtotal == 0 && rows.TotalSuggested > 0 {
		totals.Subtotal = getRightmostPositiveNumericRowValue(f, sheetName, rows.TotalSuggested)
	}

	if totals.Subtotal == 0 {
		for _, item := range items {
			totals.Subtotal += item.TotalSuggestedBHD
		}
	}

	if totals.VatAmount == 0 && rows.TotalSuggested > 0 {
		totals.VatAmount = getRightmostPositiveNumericRowValue(f, sheetName, rows.TotalSuggested+1)
	}

	if totals.VatAmount == 0 {
		val, _ := f.GetCellValue(sheetName, "M29")
		totals.VatAmount = parseExcelFloat(val)
	}

	if totals.GrandTotal == 0 && rows.TotalSuggested > 0 {
		totals.GrandTotal = getRightmostPositiveNumericRowValue(f, sheetName, rows.TotalSuggested+2)
	}

	if totals.GrandTotal == 0 {
		val, _ := f.GetCellValue(sheetName, "M30")
		totals.GrandTotal = parseExcelFloat(val)
	}

	if totals.GrandTotal == 0 && totals.Subtotal > 0 {
		totals.GrandTotal = totals.Subtotal + totals.VatAmount
	}

	if totals.VatAmount == 0 && totals.GrandTotal > totals.Subtotal && totals.Subtotal > 0 {
		totals.VatAmount = totals.GrandTotal - totals.Subtotal
	}

	if totals.VatAmount == 0 && totals.Subtotal > 0 {
		totals.VatAmount = totals.Subtotal * (totals.VatPercent / 100)
		if totals.GrandTotal == 0 {
			totals.GrandTotal = totals.Subtotal + totals.VatAmount
		}
	}

	return totals
}

// ============================================================================
// WAILS BINDINGS
// ============================================================================

// ParseCostingSheetFile parses a single costing sheet file (Wails binding)
