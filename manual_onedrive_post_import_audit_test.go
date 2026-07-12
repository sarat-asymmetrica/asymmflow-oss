//go:build manual

package main

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type postImportAuditSummary struct {
	GeneratedAt                   time.Time `json:"generated_at"`
	DatabasePath                  string    `json:"database_path"`
	SourceExportDir               string    `json:"source_export_dir"`
	StageRows                     int       `json:"stage_rows"`
	ActiveDBRows                  int       `json:"active_db_rows"`
	MatchedRows                   int       `json:"matched_rows"`
	AnomalyRows                   int       `json:"anomaly_rows"`
	MissingDBMatches              int       `json:"missing_db_matches"`
	RevenueMismatches             int       `json:"revenue_mismatches"`
	OfferTotalMismatches          int       `json:"offer_total_mismatches"`
	OfferItemTotalMismatches      int       `json:"offer_item_total_mismatches"`
	StageSubtotalMismatches       int       `json:"stage_subtotal_mismatches"`
	StageGrandTotalMathMismatches int       `json:"stage_grand_total_math_mismatches"`
	OfferTotalBelowItemSum        int       `json:"offer_total_below_item_sum"`
	DBRevenueBelowItemSum         int       `json:"db_revenue_below_item_sum"`
	StageZeroItemsNonzeroTotal    int       `json:"stage_zero_items_nonzero_total"`
	StageMissingCommercialData    int       `json:"stage_missing_commercial_data"`
	DBMissingProductDetails       int       `json:"db_missing_product_details"`
	DBMissingOfferItems           int       `json:"db_missing_offer_items"`
	SampleCount                   int       `json:"sample_count"`
	OutputDir                     string    `json:"output_dir"`
}

type postImportStageOpportunity struct {
	FolderName           string
	RawFolderNumber      string
	DerivedPipelineKey   string
	Stage                string
	CustomerMatchName    string
	PrimaryCostingPath   string
	PrimaryLineItemCount int
	PrimarySubtotalBHD   float64
	PrimaryVATBHD        float64
	PrimaryGrandTotalBHD float64
}

type postImportStageLineItem struct {
	DerivedPipelineKey string
	CostingPath        string
	LineNumber         int
	Model              string
	Quantity           float64
	SuggestedPriceBHD  float64
	LineTotalBHD       float64
}

type postImportAuditAnomalyRow struct {
	DerivedPipelineKey   string
	FolderName           string
	RawFolderNumber      string
	Stage                string
	CustomerName         string
	PrimaryCostingPath   string
	StageLineItemCount   int
	StageGrandTotalBHD   float64
	DBFolderNumber       string
	DBStage              string
	DBRevenueBHD         float64
	DBProductDetailCount int
	OfferID              string
	OfferTotalBHD        float64
	OfferItemCount       int
	OfferItemsTotalBHD   float64
	Reasons              string
}

type postImportSpotCheckRow struct {
	DerivedPipelineKey   string
	FolderName           string
	RawFolderNumber      string
	CustomerName         string
	PrimaryCostingPath   string
	StageLineItemCount   int
	StageGrandTotalBHD   float64
	DBFolderNumber       string
	DBStage              string
	DBRevenueBHD         float64
	DBProductDetailCount int
	OfferID              string
	OfferTotalBHD        float64
	OfferItemCount       int
	OfferItemsTotalBHD   float64
	PrimaryModels        string
	ComparisonStatus     string
}

type opportunityProductDetail struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
	PartNumber  string  `json:"part_number"`
}

func TestManualAuditOneDriveImport(t *testing.T) {
	dbPath := resolveOneDriveImportDBPath()

	sourceExportDir := os.Getenv("ONEDRIVE_AUDIT_SOURCE_DIR")
	if sourceExportDir == "" {
		var err error
		sourceExportDir, err = findLatestOneDriveReportDir("reports", "onedrive_seed_")
		if err != nil {
			t.Skipf("Skipping manual audit: latest OneDrive seed report is not available: %v", err)
		}
	}

	stamp := time.Now().Format("20060102_150405")
	outputDir := os.Getenv("ONEDRIVE_AUDIT_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = filepath.Join("reports", "onedrive_post_import_audit_"+stamp)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir %s: %v", outputDir, err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database %s: %v", dbPath, err)
	}

	stageOpps, err := readStageOpportunitiesCSV(filepath.Join(sourceExportDir, "opportunities.csv"))
	if err != nil {
		t.Fatalf("failed to read staged opportunities: %v", err)
	}
	importYear := inferStageOpportunityYear(stageOpps)
	if importYear == 0 {
		importYear = resolveOneDriveImportYear(sourceExportDir)
	}
	importSource := oneDriveOpportunitySource(importYear)
	stageLines, err := readStageLineItemsCSV(filepath.Join(sourceExportDir, "line_items.csv"))
	if err != nil {
		t.Fatalf("failed to read staged line items: %v", err)
	}
	stagePrimaryLines := buildPrimaryStageLineMap(stageOpps, stageLines)

	var dbOpps []Opportunity
	if err := db.Where("year = ? AND source = ? AND deleted_at IS NULL", importYear, importSource).
		Find(&dbOpps).Error; err != nil {
		t.Fatalf("failed to load imported opportunities: %v", err)
	}

	dbByDerived := map[string]*Opportunity{}
	dbByFolderName := map[string]*Opportunity{}
	for i := range dbOpps {
		opp := &dbOpps[i]
		if strings.TrimSpace(opp.FolderNumber) != "" {
			dbByDerived[strings.ToUpper(strings.TrimSpace(opp.FolderNumber))] = opp
		}
		dbByFolderName[strings.ToUpper(normalizeSeedSpace(opp.FolderName))] = opp
	}

	offerIDs := make([]string, 0, len(dbOpps))
	offerIDSeen := map[string]bool{}
	for _, opp := range dbOpps {
		if strings.TrimSpace(opp.OfferID) == "" || offerIDSeen[opp.OfferID] {
			continue
		}
		offerIDSeen[opp.OfferID] = true
		offerIDs = append(offerIDs, opp.OfferID)
	}

	offersByID := map[string]Offer{}
	if len(offerIDs) > 0 {
		var offers []Offer
		if err := db.Where("id IN ? AND deleted_at IS NULL", offerIDs).Find(&offers).Error; err != nil {
			t.Fatalf("failed to load offers: %v", err)
		}
		for _, offer := range offers {
			offersByID[offer.ID] = offer
		}
	}

	offerItemsByOfferID := map[string][]OfferItem{}
	if len(offerIDs) > 0 {
		var offerItems []OfferItem
		if err := db.Where("offer_id IN ? AND deleted_at IS NULL", offerIDs).
			Order("offer_id asc, line_number asc, created_at asc").
			Find(&offerItems).Error; err != nil {
			t.Fatalf("failed to load offer items: %v", err)
		}
		for _, item := range offerItems {
			offerItemsByOfferID[item.OfferID] = append(offerItemsByOfferID[item.OfferID], item)
		}
	}

	stageKeys := make([]string, 0, len(stageOpps))
	for key := range stageOpps {
		stageKeys = append(stageKeys, key)
	}
	sort.Strings(stageKeys)

	var anomalyRows []postImportAuditAnomalyRow
	var spotCheckRows []postImportSpotCheckRow
	summary := postImportAuditSummary{
		GeneratedAt:     time.Now(),
		DatabasePath:    dbPath,
		SourceExportDir: sourceExportDir,
		StageRows:       len(stageOpps),
		ActiveDBRows:    len(dbOpps),
		OutputDir:       outputDir,
	}

	for _, key := range stageKeys {
		stage := stageOpps[key]
		dbOpp := dbByDerived[strings.ToUpper(strings.TrimSpace(stage.DerivedPipelineKey))]
		if dbOpp == nil {
			dbOpp = dbByFolderName[strings.ToUpper(normalizeSeedSpace(stage.FolderName))]
		}

		if dbOpp == nil {
			summary.MissingDBMatches++
			summary.AnomalyRows++
			anomalyRows = append(anomalyRows, postImportAuditAnomalyRow{
				DerivedPipelineKey: stage.DerivedPipelineKey,
				FolderName:         stage.FolderName,
				RawFolderNumber:    stage.RawFolderNumber,
				Stage:              stage.Stage,
				CustomerName:       stage.CustomerMatchName,
				PrimaryCostingPath: stage.PrimaryCostingPath,
				StageLineItemCount: stage.PrimaryLineItemCount,
				StageGrandTotalBHD: stage.PrimaryGrandTotalBHD,
				Reasons:            "missing_active_db_match",
			})
			continue
		}

		summary.MatchedRows++

		productDetails := parseAuditOpportunityProductDetails(dbOpp.ProductDetails)
		productDetailCount := len(productDetails)
		productDetailTotal := 0.0
		for _, item := range productDetails {
			productDetailTotal += item.TotalPrice
		}

		offer := offersByID[dbOpp.OfferID]
		offerItems := offerItemsByOfferID[dbOpp.OfferID]
		offerItemsTotal := 0.0
		for _, item := range offerItems {
			offerItemsTotal += item.TotalPrice
		}

		stageLineItems := stagePrimaryLines[stage.DerivedPipelineKey]
		stageLineTotal := 0.0
		var models []string
		for _, item := range stageLineItems {
			stageLineTotal += item.LineTotalBHD
			model := strings.TrimSpace(item.Model)
			if model != "" {
				models = append(models, model)
			}
		}
		sort.Strings(models)
		models = compactSortedStrings(models)

		var reasons []string
		if (stage.Stage == "Quoted" || stage.Stage == "Won") && stage.PrimaryLineItemCount == 0 && stage.PrimaryGrandTotalBHD <= 0 {
			summary.StageMissingCommercialData++
			reasons = append(reasons, "stage_missing_commercial_data")
		}
		if stage.PrimaryLineItemCount == 0 && stage.PrimaryGrandTotalBHD > 0 {
			summary.StageZeroItemsNonzeroTotal++
			reasons = append(reasons, "stage_zero_items_nonzero_total")
		}
		if stage.PrimaryLineItemCount > 0 && deltaAbs(stageLineTotal, stage.PrimarySubtotalBHD) > 0.01 {
			summary.StageSubtotalMismatches++
			reasons = append(reasons, "stage_subtotal_mismatch")
		}
		if deltaAbs(stage.PrimarySubtotalBHD+stage.PrimaryVATBHD, stage.PrimaryGrandTotalBHD) > 0.01 {
			summary.StageGrandTotalMathMismatches++
			reasons = append(reasons, "stage_grand_total_mismatch")
		}
		if deltaAbs(dbOpp.RevenueBHD, stage.PrimaryGrandTotalBHD) > 0.01 {
			summary.RevenueMismatches++
			reasons = append(reasons, "db_revenue_mismatch")
		}
		if stage.PrimaryLineItemCount > 0 && productDetailCount == 0 {
			summary.DBMissingProductDetails++
			reasons = append(reasons, "missing_db_product_details")
		}
		if strings.TrimSpace(dbOpp.OfferID) != "" && stage.PrimaryLineItemCount > 0 && len(offerItems) == 0 {
			summary.DBMissingOfferItems++
			reasons = append(reasons, "missing_offer_items")
		}
		if strings.TrimSpace(dbOpp.OfferID) != "" && len(offerItems) > 0 && offer.TotalValueBHD+0.01 < offerItemsTotal {
			summary.OfferTotalBelowItemSum++
			reasons = append(reasons, "offer_total_below_item_sum")
		}
		if len(offerItems) > 0 && dbOpp.RevenueBHD+0.01 < offerItemsTotal {
			summary.DBRevenueBelowItemSum++
			reasons = append(reasons, "db_revenue_below_item_sum")
		}
		if strings.TrimSpace(dbOpp.OfferID) != "" && deltaAbs(offer.TotalValueBHD, stage.PrimaryGrandTotalBHD) > 0.01 {
			summary.OfferTotalMismatches++
			reasons = append(reasons, "offer_total_mismatch")
		}
		if stage.PrimaryLineItemCount > 0 && len(offerItems) > 0 && deltaAbs(offerItemsTotal, stageLineTotal) > 0.01 {
			summary.OfferItemTotalMismatches++
			reasons = append(reasons, "offer_items_total_mismatch")
		}

		status := "pass"
		if len(reasons) > 0 {
			status = "warn"
			summary.AnomalyRows++
			anomalyRows = append(anomalyRows, postImportAuditAnomalyRow{
				DerivedPipelineKey:   stage.DerivedPipelineKey,
				FolderName:           stage.FolderName,
				RawFolderNumber:      stage.RawFolderNumber,
				Stage:                stage.Stage,
				CustomerName:         stage.CustomerMatchName,
				PrimaryCostingPath:   stage.PrimaryCostingPath,
				StageLineItemCount:   stage.PrimaryLineItemCount,
				StageGrandTotalBHD:   stage.PrimaryGrandTotalBHD,
				DBFolderNumber:       dbOpp.FolderNumber,
				DBStage:              dbOpp.Stage,
				DBRevenueBHD:         dbOpp.RevenueBHD,
				DBProductDetailCount: productDetailCount,
				OfferID:              dbOpp.OfferID,
				OfferTotalBHD:        offer.TotalValueBHD,
				OfferItemCount:       len(offerItems),
				OfferItemsTotalBHD:   offerItemsTotal,
				Reasons:              strings.Join(reasons, " | "),
			})
		}

		spotCheckRows = append(spotCheckRows, postImportSpotCheckRow{
			DerivedPipelineKey:   stage.DerivedPipelineKey,
			FolderName:           stage.FolderName,
			RawFolderNumber:      stage.RawFolderNumber,
			CustomerName:         stage.CustomerMatchName,
			PrimaryCostingPath:   stage.PrimaryCostingPath,
			StageLineItemCount:   stage.PrimaryLineItemCount,
			StageGrandTotalBHD:   stage.PrimaryGrandTotalBHD,
			DBFolderNumber:       dbOpp.FolderNumber,
			DBStage:              dbOpp.Stage,
			DBRevenueBHD:         dbOpp.RevenueBHD,
			DBProductDetailCount: productDetailCount,
			OfferID:              dbOpp.OfferID,
			OfferTotalBHD:        offer.TotalValueBHD,
			OfferItemCount:       len(offerItems),
			OfferItemsTotalBHD:   offerItemsTotal,
			PrimaryModels:        strings.Join(models, " | "),
			ComparisonStatus:     status,
		})

		_ = productDetailTotal
	}

	sort.Slice(anomalyRows, func(i, j int) bool {
		if anomalyRows[i].Reasons == anomalyRows[j].Reasons {
			return anomalyRows[i].DerivedPipelineKey < anomalyRows[j].DerivedPipelineKey
		}
		return anomalyRows[i].Reasons < anomalyRows[j].Reasons
	})

	samples := selectDeterministicSpotChecks(spotCheckRows, 10, 20260406)
	summary.SampleCount = len(samples)

	if err := writeSeedCSV(filepath.Join(outputDir, "anomalies.csv"), anomalyRows, []string{
		"derived_pipeline_key", "folder_name", "raw_folder_number", "stage", "customer_name", "primary_costing_path",
		"stage_line_item_count", "stage_grand_total_bhd", "db_folder_number", "db_stage", "db_revenue_bhd",
		"db_product_detail_count", "offer_id", "offer_total_bhd", "offer_item_count", "offer_items_total_bhd", "reasons",
	}, func(row postImportAuditAnomalyRow) []string {
		return []string{
			row.DerivedPipelineKey,
			row.FolderName,
			row.RawFolderNumber,
			row.Stage,
			row.CustomerName,
			row.PrimaryCostingPath,
			intString(row.StageLineItemCount),
			floatString(row.StageGrandTotalBHD),
			row.DBFolderNumber,
			row.DBStage,
			floatString(row.DBRevenueBHD),
			intString(row.DBProductDetailCount),
			row.OfferID,
			floatString(row.OfferTotalBHD),
			intString(row.OfferItemCount),
			floatString(row.OfferItemsTotalBHD),
			row.Reasons,
		}
	}); err != nil {
		t.Fatalf("failed to write anomalies csv: %v", err)
	}

	if err := writeSeedCSV(filepath.Join(outputDir, "spot_check_samples.csv"), samples, []string{
		"derived_pipeline_key", "folder_name", "raw_folder_number", "customer_name", "primary_costing_path",
		"stage_line_item_count", "stage_grand_total_bhd", "db_folder_number", "db_stage", "db_revenue_bhd",
		"db_product_detail_count", "offer_id", "offer_total_bhd", "offer_item_count", "offer_items_total_bhd",
		"primary_models", "comparison_status",
	}, func(row postImportSpotCheckRow) []string {
		return []string{
			row.DerivedPipelineKey,
			row.FolderName,
			row.RawFolderNumber,
			row.CustomerName,
			row.PrimaryCostingPath,
			intString(row.StageLineItemCount),
			floatString(row.StageGrandTotalBHD),
			row.DBFolderNumber,
			row.DBStage,
			floatString(row.DBRevenueBHD),
			intString(row.DBProductDetailCount),
			row.OfferID,
			floatString(row.OfferTotalBHD),
			intString(row.OfferItemCount),
			floatString(row.OfferItemsTotalBHD),
			row.PrimaryModels,
			row.ComparisonStatus,
		}
	}); err != nil {
		t.Fatalf("failed to write sample csv: %v", err)
	}

	if err := writeSeedJSON(filepath.Join(outputDir, "summary.json"), summary); err != nil {
		t.Fatalf("failed to write summary json: %v", err)
	}

	t.Logf("post_import_audit output_dir=%s stage_rows=%d db_rows=%d matched=%d anomalies=%d samples=%d",
		outputDir, summary.StageRows, summary.ActiveDBRows, summary.MatchedRows, summary.AnomalyRows, summary.SampleCount)
}

func findLatestOneDriveReportDir(rootDir, prefix string) (string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return "", err
	}

	type candidate struct {
		path    string
		modTime time.Time
	}
	var matches []candidate
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		fullPath := filepath.Join(rootDir, entry.Name())
		info, statErr := os.Stat(fullPath)
		if statErr != nil {
			continue
		}
		matches = append(matches, candidate{path: fullPath, modTime: info.ModTime()})
	}
	if len(matches) == 0 {
		return "", os.ErrNotExist
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})
	return matches[0].path, nil
}

func inferStageOpportunityYear(stageOpps map[string]postImportStageOpportunity) int {
	for _, stage := range stageOpps {
		derivedParts := strings.Split(strings.TrimSpace(stage.DerivedPipelineKey), "-")
		if len(derivedParts) > 0 {
			if year := parseMetaYear(strings.TrimSpace(derivedParts[0])); year != 0 {
				return year
			}
		}
		rawParts := strings.Split(strings.TrimSpace(stage.RawFolderNumber), "-")
		if len(rawParts) >= 3 {
			if year := parseMetaYear(strings.TrimSpace(rawParts[len(rawParts)-1])); year != 0 {
				return year
			}
		}
	}
	return 0
}

func readStageOpportunitiesCSV(path string) (map[string]postImportStageOpportunity, error) {
	rows, err := readCSVRows(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string]postImportStageOpportunity, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row["derived_pipeline_key"])
		if key == "" {
			key = strings.TrimSpace(row["raw_folder_number"])
		}
		if key == "" {
			key = strings.TrimSpace(row["folder_name"])
		}
		out[key] = postImportStageOpportunity{
			FolderName:           strings.TrimSpace(row["folder_name"]),
			RawFolderNumber:      strings.TrimSpace(row["raw_folder_number"]),
			DerivedPipelineKey:   strings.TrimSpace(row["derived_pipeline_key"]),
			Stage:                strings.TrimSpace(row["stage"]),
			CustomerMatchName:    strings.TrimSpace(row["customer_match_name"]),
			PrimaryCostingPath:   strings.TrimSpace(row["primary_costing_path"]),
			PrimaryLineItemCount: parseCSVInt(row["primary_line_item_count"]),
			PrimarySubtotalBHD:   parseCSVFloat(row["primary_subtotal_bhd"]),
			PrimaryVATBHD:        parseCSVFloat(row["primary_vat_bhd"]),
			PrimaryGrandTotalBHD: parseCSVFloat(row["primary_grand_total_bhd"]),
		}
	}
	return out, nil
}

func readStageLineItemsCSV(path string) ([]postImportStageLineItem, error) {
	rows, err := readCSVRows(path)
	if err != nil {
		return nil, err
	}
	out := make([]postImportStageLineItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, postImportStageLineItem{
			DerivedPipelineKey: strings.TrimSpace(row["derived_pipeline_key"]),
			CostingPath:        strings.TrimSpace(row["costing_path"]),
			LineNumber:         parseCSVInt(row["line_number"]),
			Model:              strings.TrimSpace(row["model"]),
			Quantity:           parseCSVFloat(row["quantity"]),
			SuggestedPriceBHD:  parseCSVFloat(row["suggested_price_bhd"]),
			LineTotalBHD:       parseCSVFloat(row["line_total_bhd"]),
		})
	}
	return out, nil
}

func readCSVRows(path string) ([]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	header := records[0]
	out := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		row := map[string]string{}
		for i, column := range header {
			if i < len(record) {
				row[column] = record[i]
			} else {
				row[column] = ""
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func buildPrimaryStageLineMap(stageOpps map[string]postImportStageOpportunity, stageLines []postImportStageLineItem) map[string][]postImportStageLineItem {
	out := map[string][]postImportStageLineItem{}
	for _, line := range stageLines {
		stage := stageOpps[line.DerivedPipelineKey]
		if stage.PrimaryCostingPath == "" {
			continue
		}
		if strings.TrimSpace(line.CostingPath) != stage.PrimaryCostingPath {
			continue
		}
		out[line.DerivedPipelineKey] = append(out[line.DerivedPipelineKey], line)
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool {
			return out[key][i].LineNumber < out[key][j].LineNumber
		})
	}
	return out
}

func parseAuditOpportunityProductDetails(raw string) []opportunityProductDetail {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var details []opportunityProductDetail
	if err := json.Unmarshal([]byte(raw), &details); err != nil {
		return nil
	}
	return details
}

func selectDeterministicSpotChecks(rows []postImportSpotCheckRow, count int, seed int64) []postImportSpotCheckRow {
	if len(rows) == 0 || count <= 0 {
		return nil
	}
	copyRows := append([]postImportSpotCheckRow(nil), rows...)
	sort.Slice(copyRows, func(i, j int) bool {
		return copyRows[i].DerivedPipelineKey < copyRows[j].DerivedPipelineKey
	})
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(copyRows), func(i, j int) {
		copyRows[i], copyRows[j] = copyRows[j], copyRows[i]
	})
	if len(copyRows) > count {
		copyRows = copyRows[:count]
	}
	sort.Slice(copyRows, func(i, j int) bool {
		return copyRows[i].DerivedPipelineKey < copyRows[j].DerivedPipelineKey
	})
	return copyRows
}

func compactSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	var out []string
	last := ""
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == last {
			continue
		}
		out = append(out, value)
		last = value
	}
	return out
}

func deltaAbs(a, b float64) float64 {
	return math.Abs(a - b)
}

func parseCSVFloat(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseCSVInt(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return v
}
