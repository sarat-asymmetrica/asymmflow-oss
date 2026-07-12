package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	crmpipeline "ph_holdings_app/pkg/crm/pipeline"
)

func (a *App) CreateRFQ(client, project string, value float64, notes string, productDetails string) (*RFQData, error) {
	return a.createRFQ(client, project, "", value, notes, productDetails)
}

// CreateRFQWithReference creates a new RFQ/opportunity while preserving the user-entered reference.
func (a *App) CreateRFQWithReference(client, project, reference string, value float64, notes string) (*RFQData, error) {
	return a.createRFQ(client, project, reference, value, notes, "")
}

func (a *App) createRFQ(client, project, reference string, value float64, notes string, productDetails string) (*RFQData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:create (RFQ is part of sales pipeline)
	if err := a.requirePermission("offers:create"); err != nil {
		log.Printf("🔒 CreateRFQ blocked: %v", err)
		return nil, err
	}

	userReference := limitReferenceRunes(reference, 100)
	rfqNumber := a.generateRFQNumber()
	if userReference != "" {
		rfqNumber = limitReferenceRunes(userReference, 50)
	}

	cleanedProductDetails := strings.TrimSpace(productDetails)
	if len(cleanedProductDetails) > 5000 {
		cleanedProductDetails = cleanedProductDetails[:5000]
	}

	rfq := &RFQData{
		RFQNumber:      rfqNumber,
		RFQRef:         userReference,
		Client:         client,
		Project:        project,
		Value:          value,
		Notes:          notes,
		Status:         "pending",
		Stage:          "RFQ Received",
		ProductDetails: cleanedProductDetails,
	}

	result := a.db.Create(rfq)
	if result.Error != nil {
		log.Printf("❌ Failed to create RFQ: %v", result.Error)
		return nil, result.Error
	}

	log.Printf("✅ Created RFQ #%d: %s - %s (%.2f BHD)", rfq.ID, client, project, value)
	return rfq, nil
}

func limitReferenceRunes(value string, maxRunes int) string {
	trimmed := strings.TrimSpace(value)
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes])
}

// GetRFQs retrieves all RFQs with optional pagination
func (a *App) GetRFQs(limit int, offset int) ([]RFQData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	// Default limit: 100, max limit: 1000
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	var rfqs []RFQData
	result := a.db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rfqs)

	if result.Error != nil {
		log.Printf("❌ Error retrieving RFQs: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve RFQs: %w", result.Error)
	}

	log.Printf("✅ Retrieved %d RFQs (limit=%d, offset=%d)", len(rfqs), limit, offset)
	return rfqs, nil
}

// GetPipelineOpportunities returns opportunities from the canonical pipeline (Phase 33 seed import)
func (a *App) GetPipelineOpportunities(limit int, offset int) ([]Opportunity, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 200
	} else if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var raw []Opportunity
	result := a.db.Find(&raw)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to retrieve opportunities: %w", result.Error)
	}

	opps := dedupePipelineOpportunities(raw)
	if offset >= len(opps) {
		return []Opportunity{}, nil
	}

	end := offset + limit
	if end > len(opps) {
		end = len(opps)
	}

	return opps[offset:end], nil
}

func dedupePipelineOpportunities(raw []Opportunity) []Opportunity {
	filtered := make([]Opportunity, 0, len(raw))
	for _, opp := range raw {
		normalized := normalizeOpportunityForList(opp)
		if shouldSuppressSyntheticOCR(normalized) {
			continue
		}
		filtered = append(filtered, normalized)
	}

	byKey := make(map[string]Opportunity, len(raw))
	for _, normalized := range filtered {
		key := canonicalOpportunityKey(normalized)
		if key == "" {
			key = normalized.ID
		}
		existing, exists := byKey[key]
		if !exists || shouldPreferOpportunity(normalized, existing) {
			byKey[key] = normalized
		}
	}

	deduped := make([]Opportunity, 0, len(byKey))
	for _, opp := range byKey {
		deduped = append(deduped, opp)
	}

	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].Year != deduped[j].Year {
			return deduped[i].Year > deduped[j].Year
		}
		if deduped[i].OppNumber != deduped[j].OppNumber {
			return deduped[i].OppNumber > deduped[j].OppNumber
		}
		return deduped[i].CreatedAt.After(deduped[j].CreatedAt)
	})

	return deduped
}

func normalizeCommercialReferenceNumber(value string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "-")
	if len(parts) != 3 {
		return trimmed
	}
	middle, errMiddle := strconv.Atoi(parts[1])
	last, errLast := strconv.Atoi(parts[2])
	if errMiddle != nil || errLast != nil {
		return trimmed
	}
	return fmt.Sprintf("%s-%d-%d", parts[0], middle, last)
}

func shouldPreferOrderListRow(candidate Order, existing Order) bool {
	candidateZeroPadded := normalizeCommercialReferenceNumber(candidate.OrderNumber) != strings.ToUpper(strings.TrimSpace(candidate.OrderNumber))
	existingZeroPadded := normalizeCommercialReferenceNumber(existing.OrderNumber) != strings.ToUpper(strings.TrimSpace(existing.OrderNumber))
	if candidateZeroPadded != existingZeroPadded {
		return candidateZeroPadded
	}
	if candidate.GrandTotalBHD != existing.GrandTotalBHD {
		return candidate.GrandTotalBHD > existing.GrandTotalBHD
	}
	if candidate.TotalValueBHD != existing.TotalValueBHD {
		return candidate.TotalValueBHD > existing.TotalValueBHD
	}
	if !candidate.CreatedAt.Equal(existing.CreatedAt) {
		return candidate.CreatedAt.Before(existing.CreatedAt)
	}
	return strings.Compare(candidate.ID, existing.ID) < 0
}

func dedupeOrdersForList(raw []Order) []Order {
	byKey := make(map[string]Order, len(raw))
	orderKeys := make([]string, 0, len(raw))
	for _, order := range raw {
		key := normalizeCommercialReferenceNumber(order.OrderNumber)
		if key == "" {
			key = order.ID
		}
		existing, ok := byKey[key]
		if !ok {
			byKey[key] = order
			orderKeys = append(orderKeys, key)
			continue
		}
		if shouldPreferOrderListRow(order, existing) {
			byKey[key] = order
		}
	}
	deduped := make([]Order, 0, len(byKey))
	for _, key := range orderKeys {
		if order, ok := byKey[key]; ok {
			deduped = append(deduped, order)
		}
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		return deduped[i].OrderDate.After(deduped[j].OrderDate)
	})
	return deduped
}

func shouldSuppressSyntheticOCR(opp Opportunity) bool {
	if !strings.EqualFold(strings.TrimSpace(opp.Source), "2026_ocr") {
		return false
	}
	if opp.Year != 2026 {
		return false
	}
	if opp.OppNumber < 200 {
		return false
	}

	folder := strings.TrimSpace(opp.FolderNumber)
	return strings.HasPrefix(folder, "2026-")
}

func normalizeOpportunityForList(opp Opportunity) Opportunity {
	metaCandidates := []string{
		strings.TrimSpace(opp.FolderName),
		strings.TrimSpace(opp.FolderNumber + " " + opp.Title),
		strings.TrimSpace(opp.FolderNumber),
		strings.TrimSpace(opp.Title),
	}

	for _, candidate := range metaCandidates {
		if candidate == "" {
			continue
		}
		meta := parseOneDriveFolderMeta(candidate)
		// FolderNumber drives the canonical dedup key, so guard it carefully. A real
		// folder number always contains a digit; never let a digit-less candidate
		// (e.g. a bare customer word like "BAPCO" produced by the loose fallback)
		// overwrite an already-good folder number. That collapse hid ~83 pipeline
		// opportunities and cross-linked costings to the wrong opportunity.
		if meta.FolderNumber != "" && folderNumberHasDigit(meta.FolderNumber) &&
			(opp.FolderNumber == "" || isOneDriveOpportunitySource(opp.Source)) {
			opp.FolderNumber = meta.FolderNumber
		}
		if meta.Title != "" && (opp.Title == "" || opp.Title == opp.FolderName) {
			opp.Title = meta.Title
		}
		if meta.Year != 0 && (opp.Year == 0 || isOneDriveOpportunitySource(opp.Source)) {
			opp.Year = meta.Year
		}
		if meta.OppNumber != 0 && (opp.OppNumber == 0 || isOneDriveOpportunitySource(opp.Source)) {
			opp.OppNumber = meta.OppNumber
		}
	}

	if opp.FolderName == "" {
		parts := []string{strings.TrimSpace(opp.FolderNumber), strings.TrimSpace(opp.Title)}
		opp.FolderName = strings.TrimSpace(strings.Join(parts, " "))
	}

	if opp.Year < 2000 || opp.Year > 2100 {
		opp.Year = 0
	}
	if opp.OppNumber < 0 || opp.OppNumber > 9999 {
		opp.OppNumber = 0
	}

	return opp
}

func canonicalOpportunityKey(opp Opportunity) string {
	if opp.Year >= 2000 && opp.Year <= 2100 && opp.OppNumber > 0 {
		return fmt.Sprintf("%d-%03d", opp.Year, opp.OppNumber)
	}

	folder := strings.ToUpper(strings.TrimSpace(opp.FolderNumber))
	if folder != "" {
		return folder
	}

	title := strings.ToUpper(strings.TrimSpace(opp.Title))
	if title != "" {
		return fmt.Sprintf("%d:%s", opp.Year, title)
	}

	return strings.TrimSpace(opp.ID)
}

func shouldPreferOpportunity(candidate, existing Opportunity) bool {
	if opportunitySourcePriority(candidate.Source) != opportunitySourcePriority(existing.Source) {
		return opportunitySourcePriority(candidate.Source) > opportunitySourcePriority(existing.Source)
	}

	if opportunityRichnessScore(candidate) != opportunityRichnessScore(existing) {
		return opportunityRichnessScore(candidate) > opportunityRichnessScore(existing)
	}

	return candidate.UpdatedAt.After(existing.UpdatedAt)
}

func opportunitySourcePriority(source string) int {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "onedrive_import":
		return 4
	default:
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(source)), "_onedrive") {
			return 4
		}
	}
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "2026_ocr":
		return 3
	case "2025_excel":
		return 2
	default:
		return 1
	}
}

func opportunityRichnessScore(opp Opportunity) int {
	score := 0
	for _, value := range []string{
		opp.CustomerName,
		opp.Title,
		opp.Comment,
		opp.OwnerNotes,
		opp.ProductDetails,
		opp.PaymentTerms,
		opp.DeliveryTerms,
		opp.FolderName,
	} {
		if strings.TrimSpace(value) != "" {
			score++
		}
	}
	if opp.RevenueBHD > 0 {
		score++
	}
	if opp.ExpectedDate != nil {
		score++
	}
	return score
}

func serializeOpportunityProductDetailsFromOfferItems(items []OfferItem) string {
	if len(items) == 0 {
		return ""
	}

	type opportunityLineItem struct {
		Description         string  `json:"description,omitempty"`
		Quantity            float64 `json:"quantity,omitempty"`
		UnitPrice           float64 `json:"unit_price,omitempty"`
		TotalPrice          float64 `json:"total_price,omitempty"`
		PartNumber          string  `json:"part_number,omitempty"`
		LongCode            string  `json:"long_code,omitempty"`
		Unit                string  `json:"unit,omitempty"`
		Currency            string  `json:"currency,omitempty"`
		Specification       string  `json:"specification,omitempty"`
		DetailedDescription string  `json:"detailed_description,omitempty"`
	}

	payload := make([]opportunityLineItem, 0, len(items))
	for _, item := range items {
		if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = strings.TrimSpace(item.Equipment)
		}
		partNumber := strings.TrimSpace(item.Model)
		if partNumber == "" {
			partNumber = strings.TrimSpace(item.ProductCode)
		}
		payload = append(payload, opportunityLineItem{
			Description:         description,
			Quantity:            item.Quantity,
			UnitPrice:           item.UnitPrice,
			TotalPrice:          item.TotalPrice,
			PartNumber:          partNumber,
			LongCode:            strings.TrimSpace(item.LongCode),
			Currency:            normalizeCurrencyCode(item.Currency),
			Specification:       strings.TrimSpace(item.Specification),
			DetailedDescription: strings.TrimSpace(item.DetailedDescription),
		})
	}
	if len(payload) == 0 {
		return ""
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}

func parseOpportunityProductDetails(details string) []OfferItem {
	details = strings.TrimSpace(details)
	if details == "" {
		return nil
	}

	var parsed []map[string]any
	if err := json.Unmarshal([]byte(details), &parsed); err != nil {
		var single map[string]any
		if singleErr := json.Unmarshal([]byte(details), &single); singleErr != nil {
			return nil
		}
		parsed = []map[string]any{single}
	}

	items := make([]OfferItem, 0, len(parsed))
	for idx, raw := range parsed {
		description := strings.TrimSpace(getStringFromMap(raw, "description"))
		if description == "" {
			description = strings.TrimSpace(getStringFromMap(raw, "name"))
		}
		if description == "" {
			description = strings.TrimSpace(getStringFromMap(raw, "equipment"))
		}
		partNumber := strings.TrimSpace(getStringFromMap(raw, "part_number"))
		if partNumber == "" {
			partNumber = strings.TrimSpace(getStringFromMap(raw, "model"))
		}
		if partNumber == "" {
			partNumber = strings.TrimSpace(getStringFromMap(raw, "product_code"))
		}

		quantity := getFloatFromMap(raw, "quantity")
		if quantity <= 0 {
			quantity = 1
		}

		unitPrice := getFloatFromMap(raw, "unit_price_bhd")
		if unitPrice <= 0 {
			unitPrice = getFloatFromMap(raw, "unit_price")
		}
		totalPrice := getFloatFromMap(raw, "total_price")
		if totalPrice <= 0 && unitPrice > 0 {
			totalPrice = quantity * unitPrice
		}

		if isSyntheticCommercialSummary(description, partNumber, partNumber, description) {
			continue
		}

		if description == "" && partNumber == "" && totalPrice == 0 && unitPrice == 0 {
			continue
		}

		unit := strings.TrimSpace(getStringFromMap(raw, "unit"))
		currency := normalizeCurrencyCode(getStringFromMap(raw, "currency"))
		if currency == "" && isCurrencyCode(unit) {
			currency = normalizeCurrencyCode(unit)
			unit = ""
		}

		items = append(items, OfferItem{
			Base:                Base{ID: uuid.New().String()},
			LineNumber:          idx + 1,
			Description:         description,
			Equipment:           description,
			Model:               partNumber,
			ProductCode:         partNumber,
			LongCode:            strings.TrimSpace(getStringFromMap(raw, "long_code")),
			Quantity:            quantity,
			UnitPrice:           unitPrice,
			TotalPrice:          totalPrice,
			Currency:            currency,
			Specification:       strings.TrimSpace(getStringFromMap(raw, "specification")),
			DetailedDescription: strings.TrimSpace(getStringFromMap(raw, "detailed_description")),
		})
	}

	return items
}

func normalizeOpportunityLineItemsJSON(rawItems any) (string, int) {
	if rawItems == nil {
		return "", 0
	}

	var rows []map[string]any
	switch typed := rawItems.(type) {
	case []any:
		for _, item := range typed {
			if mapped, ok := item.(map[string]any); ok {
				rows = append(rows, mapped)
			}
		}
	case []map[string]any:
		rows = typed
	case map[string]any:
		rows = []map[string]any{typed}
	default:
		return "", 0
	}

	type storedLineItem struct {
		Description   string  `json:"description,omitempty"`
		Quantity      float64 `json:"quantity,omitempty"`
		Unit          string  `json:"unit,omitempty"`
		PartNumber    string  `json:"part_number,omitempty"`
		ProductCode   string  `json:"product_code,omitempty"`
		Model         string  `json:"model,omitempty"`
		LongCode      string  `json:"long_code,omitempty"`
		UnitPrice     float64 `json:"unit_price,omitempty"`
		TotalPrice    float64 `json:"total_price,omitempty"`
		Currency      string  `json:"currency,omitempty"`
		Specification string  `json:"specification,omitempty"`
		RawText       string  `json:"raw_text,omitempty"`
	}

	payload := make([]storedLineItem, 0, len(rows))
	for _, raw := range rows {
		description := firstStringFromMap(raw, "description", "item_description", "item", "name", "equipment", "product", "product_name", "model", "part_number")
		partNumber := firstStringFromMap(raw, "part_number", "part_no", "item_code", "product_code", "model", "long_code")
		productCode := firstStringFromMap(raw, "product_code", "item_code", "part_number", "model")
		model := firstStringFromMap(raw, "model", "part_number", "product_code")
		longCode := firstStringFromMap(raw, "long_code", "eh_code", "ordering_code")

		quantity := firstFloatFromMap(raw, "quantity", "qty")
		unitPrice := firstFloatFromMap(raw, "unit_price_bhd", "unit_price", "price", "rate")
		totalPrice := firstFloatFromMap(raw, "total_price", "total", "line_total", "amount")
		if quantity <= 0 {
			if totalPrice > 0 && unitPrice > 0 {
				quantity = totalPrice / unitPrice
			} else {
				quantity = 1
			}
		}
		if unitPrice <= 0 && totalPrice > 0 && quantity > 0 {
			unitPrice = totalPrice / quantity
		}
		if totalPrice <= 0 && unitPrice > 0 && quantity > 0 {
			totalPrice = unitPrice * quantity
		}

		unit := firstStringFromMap(raw, "unit", "uom", "unit_of_measure")
		currency := normalizeCurrencyCode(firstStringFromMap(raw, "currency"))
		if currency == "" && isCurrencyCode(unit) {
			currency = normalizeCurrencyCode(unit)
			unit = ""
		}
		if isCurrencyCode(unit) {
			unit = ""
		}

		if description == "" && partNumber == "" && productCode == "" && model == "" && longCode == "" {
			continue
		}

		payload = append(payload, storedLineItem{
			Description:   description,
			Quantity:      quantity,
			Unit:          unit,
			PartNumber:    partNumber,
			ProductCode:   productCode,
			Model:         model,
			LongCode:      longCode,
			UnitPrice:     roundTo3(unitPrice),
			TotalPrice:    roundTo3(totalPrice),
			Currency:      currency,
			Specification: firstStringFromMap(raw, "specification", "notes"),
			RawText:       firstStringFromMap(raw, "raw_text", "raw_row"),
		})
	}
	if len(payload) == 0 {
		return "", 0
	}
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", 0
	}
	return string(jsonBytes), len(payload)
}

func firstStringFromMap(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(getStringFromMap(raw, key))
		if value != "" {
			return value
		}
	}
	return ""
}

func firstFloatFromMap(raw map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value := getFloatFromMap(raw, key)
		if value > 0 {
			return value
		}
	}
	return 0
}

func isCurrencyCode(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "BHD", "USD", "EUR", "GBP", "SAR", "AED", "CHF", "KWD", "OMR", "QAR":
		return true
	default:
		return false
	}
}

func normalizeCurrencyCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if isCurrencyCode(value) {
		return value
	}
	return ""
}

func getStringFromMap(raw map[string]any, key string) string {
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getFloatFromMap(raw map[string]any, key string) float64 {
	value, ok := raw[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, err := parseFlexibleBusinessNumber(v)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func parseFlexibleBusinessNumber(value string) (float64, error) {
	cleaned := regexp.MustCompile(`(?i)\b(BHD|USD|EUR|GBP|SAR|AED|CHF|KWD|OMR|QAR)\b`).ReplaceAllString(value, "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	match := regexp.MustCompile(`[-+]?\d*\.?\d+`).FindString(cleaned)
	if match == "" {
		return 0, fmt.Errorf("no numeric value in %q", value)
	}
	return strconv.ParseFloat(match, 64)
}

func backfillOpportunityProductDetailsFromOffers(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	var opportunities []Opportunity
	if err := db.Where("COALESCE(product_details, '') = '' AND COALESCE(offer_id, '') <> ''").Find(&opportunities).Error; err != nil {
		return err
	}

	for _, opp := range opportunities {
		var items []OfferItem
		if err := db.Where("offer_id = ?", opp.OfferID).Order("line_number ASC").Find(&items).Error; err != nil {
			continue
		}
		productDetails := serializeOpportunityProductDetailsFromOfferItems(items)
		if productDetails == "" {
			continue
		}
		_ = db.Model(&Opportunity{}).Where("id = ?", opp.ID).Update("product_details", productDetails).Error
	}

	return nil
}

func repairImportedOpportunityMetadata(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	var imported []Opportunity
	if err := db.Where("source = ? OR source LIKE ?", "onedrive_import", "%_onedrive").Find(&imported).Error; err != nil {
		return err
	}

	updated := 0
	for _, opp := range imported {
		normalized := normalizeOpportunityForList(opp)
		updates := map[string]any{}

		if normalized.FolderNumber != "" && normalized.FolderNumber != opp.FolderNumber {
			updates["folder_number"] = normalized.FolderNumber
		}
		if normalized.Title != "" && normalized.Title != opp.Title {
			updates["title"] = normalized.Title
		}
		if normalized.FolderName != "" && normalized.FolderName != opp.FolderName {
			updates["folder_name"] = normalized.FolderName
		}
		if normalized.Year != 0 && normalized.Year != opp.Year {
			updates["year"] = normalized.Year
		}
		if normalized.OppNumber != 0 && normalized.OppNumber != opp.OppNumber {
			updates["opp_number"] = normalized.OppNumber
		}

		if len(updates) == 0 {
			continue
		}

		if err := db.Model(&Opportunity{}).Where("id = ?", opp.ID).Updates(updates).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed: opportunities.folder_number") {
				safeUpdates := make(map[string]any, len(updates))
				for key, value := range updates {
					if key == "folder_number" {
						continue
					}
					safeUpdates[key] = value
				}
				if len(safeUpdates) > 0 {
					if retryErr := db.Model(&Opportunity{}).Where("id = ?", opp.ID).Updates(safeUpdates).Error; retryErr == nil {
						updated++
						continue
					}
				}
			}
			log.Printf("⚠️ Failed to repair opportunity %s (%s): %v", opp.ID, opp.FolderNumber, err)
			continue
		}
		updated++
	}

	if updated > 0 {
		log.Printf("✅ Repaired metadata for %d imported opportunities", updated)
	}

	return nil
}

type convertedOfferItemRestore struct {
	ActiveID      string  `gorm:"column:active_id"`
	RawUnitPrice  float64 `gorm:"column:raw_unit_price"`
	RawTotalPrice float64 `gorm:"column:raw_total_price"`
}

func restoreConvertedOfferPricesFromDeletedSiblings(db *gorm.DB) (int, error) {
	if db == nil {
		return 0, nil
	}

	var candidates []convertedOfferItemRestore
	if err := db.Raw(`
		SELECT
			active.id AS active_id,
			deleted.unit_price AS raw_unit_price,
			deleted.total_price AS raw_total_price
		FROM offer_items active
		JOIN offer_items deleted
			ON deleted.offer_id = active.offer_id
			AND COALESCE(deleted.line_number, 0) = COALESCE(active.line_number, 0)
			AND deleted.deleted_at IS NOT NULL
		WHERE active.deleted_at IS NULL
			AND UPPER(TRIM(COALESCE(active.currency, ''))) NOT IN ('', 'BHD')
			AND COALESCE(active.exchange_rate, 0) > 0
			AND COALESCE(deleted.total_price, 0) > COALESCE(active.total_price, 0)
			AND ABS(COALESCE(active.total_price, 0) - (COALESCE(deleted.total_price, 0) * COALESCE(active.exchange_rate, 0))) <= MAX(0.05, COALESCE(deleted.total_price, 0) * 0.01)
	`).Scan(&candidates).Error; err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	seen := map[string]bool{}
	restored := 0
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, candidate := range candidates {
			if candidate.ActiveID == "" || seen[candidate.ActiveID] || candidate.RawTotalPrice <= 0 {
				continue
			}
			seen[candidate.ActiveID] = true
			if err := tx.Model(&OfferItem{}).
				Where("id = ?", candidate.ActiveID).
				Updates(map[string]any{
					"unit_price":  candidate.RawUnitPrice,
					"total_price": candidate.RawTotalPrice,
				}).Error; err != nil {
				return err
			}
			restored++
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return restored, nil
}

func repairImportedCommercialDocuments(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	if restored, err := restoreConvertedOfferPricesFromDeletedSiblings(db); err != nil {
		log.Printf("⚠️ Converted offer price restore skipped: %v", err)
	} else if restored > 0 {
		log.Printf("✅ Restored %d offer item sale prices that had been multiplied by FX", restored)
	}

	var offers []Offer
	if err := db.Preload("Items").Find(&offers).Error; err != nil {
		return err
	}

	repairedOffers := 0
	for _, offer := range offers {
		validItems := make([]OfferItem, 0, len(offer.Items))
		invalidIDs := make([]string, 0)
		rawSubtotal := 0.0
		convertedSubtotal := 0.0
		totalCost := 0.0
		itemRateUpdates := map[string]float64{}
		type priceUpdate struct {
			UnitPrice  float64
			TotalPrice float64
		}
		itemPriceUpdates := map[string]priceUpdate{}

		for _, item := range offer.Items {
			if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
				invalidIDs = append(invalidIDs, item.ID)
				continue
			}
			validItems = append(validItems, item)
			lineTotal := commercialLineTotal(item.Quantity, item.UnitPrice, item.TotalPrice)
			rawSubtotal += lineTotal
			totalCost += item.TotalCost

			rate := normalizeExchangeRateToBHD(item.Currency, item.ExchangeRate)
			if strings.ToUpper(strings.TrimSpace(item.Currency)) != "" &&
				strings.ToUpper(strings.TrimSpace(item.Currency)) != "BHD" &&
				math.Abs(rate-item.ExchangeRate) > 0.0001 {
				itemRateUpdates[item.ID] = rate
			}

			if strings.ToUpper(strings.TrimSpace(item.Currency)) == "" || strings.ToUpper(strings.TrimSpace(item.Currency)) == "BHD" {
				convertedSubtotal += lineTotal
			} else {
				convertedSubtotal += lineTotal * rate
			}
		}

		_ = convertedSubtotal // Historical diagnostic only; sale totals are already stored in BHD.
		convertForeignPrices := false
		commercialSubtotal := rawSubtotal
		if convertForeignPrices {
			commercialSubtotal = convertedSubtotal
			for _, item := range validItems {
				currency := strings.ToUpper(strings.TrimSpace(item.Currency))
				if currency == "" || currency == "BHD" {
					continue
				}
				lineTotal := commercialLineTotal(item.Quantity, item.UnitPrice, item.TotalPrice)
				if lineTotal <= 0 {
					continue
				}
				rate := normalizeExchangeRateToBHD(item.Currency, item.ExchangeRate)
				convertedLineTotal := roundTo3(lineTotal * rate)
				convertedUnitPrice := convertedLineTotal
				if item.Quantity > 0 {
					convertedUnitPrice = roundTo3(convertedLineTotal / item.Quantity)
				}
				if math.Abs(convertedLineTotal-item.TotalPrice) > 0.01 || math.Abs(convertedUnitPrice-item.UnitPrice) > 0.01 {
					itemPriceUpdates[item.ID] = priceUpdate{
						UnitPrice:  convertedUnitPrice,
						TotalPrice: convertedLineTotal,
					}
				}
				itemRateUpdates[item.ID] = rate
			}
		}

		updates := map[string]any{}
		if len(validItems) > 0 && commercialSubtotal > 0 {
			discount := commercialSubtotal * (offer.DiscountPercent / 100.0)
			netAmount := commercialSubtotal - discount
			vatRate := offer.VatRate
			totalValueBHD := roundTo3(netAmount + (netAmount * vatRate / 100.0))
			if math.Abs(offer.TotalValueBHD-totalValueBHD) > 0.01 {
				updates["total_value_bhd"] = totalValueBHD
			}
			if netAmount > 0 && totalCost > 0 {
				margin := roundTo3(((netAmount - totalCost) / netAmount) * 100)
				if math.Abs(offer.EstimatedMargin-margin) > 0.01 {
					updates["estimated_margin"] = margin
				}
			}
		}

		if len(invalidIDs) == 0 && len(updates) == 0 && len(itemRateUpdates) == 0 && len(itemPriceUpdates) == 0 {
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if len(invalidIDs) > 0 {
				if err := tx.Where("id IN ?", invalidIDs).Delete(&OfferItem{}).Error; err != nil {
					return err
				}
			}
			for index, item := range validItems {
				if err := tx.Model(&OfferItem{}).Where("id = ?", item.ID).Update("line_number", index+1).Error; err != nil {
					return err
				}
			}
			for itemID, rate := range itemRateUpdates {
				if err := tx.Model(&OfferItem{}).Where("id = ?", itemID).Update("exchange_rate", rate).Error; err != nil {
					return err
				}
			}
			for itemID, update := range itemPriceUpdates {
				if err := tx.Model(&OfferItem{}).Where("id = ?", itemID).Updates(map[string]any{
					"unit_price":  update.UnitPrice,
					"total_price": update.TotalPrice,
				}).Error; err != nil {
					return err
				}
			}
			if len(updates) > 0 {
				if err := tx.Model(&Offer{}).Where("id = ?", offer.ID).Updates(updates).Error; err != nil {
					return err
				}
				if totalValue, ok := updates["total_value_bhd"]; ok {
					if err := tx.Model(&Opportunity{}).Where("offer_id = ?", offer.ID).Update("revenue_bhd", totalValue).Error; err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			log.Printf("⚠️ Failed to repair offer %s: %v", offer.OfferNumber, err)
			continue
		}

		if convertForeignPrices {
			log.Printf("✅ Repriced foreign-currency offer %s from %.3f to %.3f BHD net", offer.OfferNumber, rawSubtotal, commercialSubtotal)
		}
		repairedOffers++
	}

	var orders []Order
	if err := db.Preload("Items").Find(&orders).Error; err != nil {
		return err
	}

	repairedOrders := 0
	for _, order := range orders {
		validItems := make([]OrderItem, 0, len(order.Items))
		invalidIDs := make([]string, 0)
		computedTotal := 0.0
		itemPriceUpdates := map[string]float64{}

		for _, item := range order.Items {
			if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
				invalidIDs = append(invalidIDs, item.ID)
				continue
			}
			validItems = append(validItems, item)
			lineTotal := item.TotalPrice
			if lineTotal <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
				lineTotal = item.Quantity * item.UnitPrice
			}
			if item.Quantity > 0 && lineTotal > 0 {
				derivedUnitPrice := roundTo3(lineTotal / item.Quantity)
				if derivedUnitPrice > 0 && math.Abs(item.UnitPrice-derivedUnitPrice) > 0.01 {
					itemPriceUpdates[item.ID] = derivedUnitPrice
				}
			}
			computedTotal += lineTotal
		}

		updates := map[string]any{}
		if len(invalidIDs) > 0 && computedTotal > 0 {
			updates["total_value_bhd"] = roundTo3(computedTotal)
			updates["grand_total_bhd"] = roundTo3(computedTotal)
		}

		if strings.HasPrefix(strings.TrimSpace(order.OrderNumber), "IMP-") {
			desiredOrderNumber := strings.TrimSpace(order.OfferNumber)
			if desiredOrderNumber == "" && strings.TrimSpace(order.OfferID) != "" {
				var offer Offer
				if err := db.Select("offer_number").Where("id = ?", order.OfferID).First(&offer).Error; err == nil {
					desiredOrderNumber = strings.TrimSpace(offer.OfferNumber)
				}
			}
			if desiredOrderNumber != "" && desiredOrderNumber != order.OrderNumber {
				var conflictCount int64
				db.Model(&Order{}).Where("order_number = ? AND id != ?", desiredOrderNumber, order.ID).Count(&conflictCount)
				if conflictCount == 0 {
					updates["order_number"] = desiredOrderNumber
				}
			}
		}

		if len(invalidIDs) == 0 && len(updates) == 0 && len(itemPriceUpdates) == 0 {
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if len(invalidIDs) > 0 {
				if err := tx.Where("id IN ?", invalidIDs).Delete(&OrderItem{}).Error; err != nil {
					return err
				}
				for index, item := range validItems {
					if err := tx.Model(&OrderItem{}).Where("id = ?", item.ID).Update("line_number", index+1).Error; err != nil {
						return err
					}
				}
			}
			for itemID, unitPrice := range itemPriceUpdates {
				if err := tx.Model(&OrderItem{}).Where("id = ?", itemID).Update("unit_price", unitPrice).Error; err != nil {
					return err
				}
			}
			if len(updates) == 0 {
				return nil
			}
			return tx.Model(&Order{}).Where("id = ?", order.ID).Updates(updates).Error
		}); err != nil {
			log.Printf("⚠️ Failed to repair order %s: %v", order.OrderNumber, err)
			continue
		}

		repairedOrders++
	}

	if repairedOffers > 0 || repairedOrders > 0 {
		log.Printf("✅ Repaired imported commercial data: %d offers, %d orders", repairedOffers, repairedOrders)
	}

	residualOfferRows, residualOrderRows, err := softDeleteResidualSyntheticCommercialRows(db)
	if err != nil {
		return err
	}
	if residualOfferRows > 0 || residualOrderRows > 0 {
		log.Printf("✅ Removed residual synthetic commercial rows: %d offer_items, %d order_items", residualOfferRows, residualOrderRows)
	}

	if err := repairImportedOpportunityProductDetails(db); err != nil {
		return err
	}

	return nil
}

func softDeleteResidualSyntheticCommercialRows(db *gorm.DB) (int64, int64, error) {
	if db == nil {
		return 0, 0, nil
	}

	const syntheticRowCondition = `
		(
			LOWER(TRIM(COALESCE(description, ''), ' -')) GLOB 'line item [0-9]*'
			OR LOWER(TRIM(COALESCE(product_code, ''), ' -')) GLOB 'line item [0-9]*'
			OR LOWER(TRIM(COALESCE(model, ''), ' -')) GLOB 'line item [0-9]*'
			OR LOWER(TRIM(COALESCE(equipment, ''), ' -')) GLOB 'line item [0-9]*'
			OR LOWER(TRIM(COALESCE(description, ''), ' -')) = 'total for order'
			OR LOWER(TRIM(COALESCE(product_code, ''), ' -')) = 'total for order'
			OR LOWER(TRIM(COALESCE(model, ''), ' -')) = 'total for order'
			OR LOWER(TRIM(COALESCE(equipment, ''), ' -')) = 'total for order'
		)
	`
	deletedAt := time.Now()

	offerResult := db.Model(&OfferItem{}).
		Where(syntheticRowCondition).
		Update("deleted_at", deletedAt)
	if offerResult.Error != nil {
		return 0, 0, fmt.Errorf("failed to clean residual synthetic offer rows: %w", offerResult.Error)
	}

	orderResult := db.Model(&OrderItem{}).
		Where(syntheticRowCondition).
		Update("deleted_at", deletedAt)
	if orderResult.Error != nil {
		return 0, 0, fmt.Errorf("failed to clean residual synthetic order rows: %w", orderResult.Error)
	}

	return offerResult.RowsAffected, orderResult.RowsAffected, nil
}

func repairImportedOpportunityProductDetails(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	var opportunities []Opportunity
	if err := db.
		Where("deleted_at IS NULL AND COALESCE(product_details, '') <> '' AND LOWER(product_details) LIKE ?", "%line item%").
		Find(&opportunities).Error; err != nil {
		return err
	}

	repaired := 0
	for _, opportunity := range opportunities {
		cleaned := serializeOpportunityProductDetailsFromOfferItems(parseOpportunityProductDetails(opportunity.ProductDetails))
		if cleaned == strings.TrimSpace(opportunity.ProductDetails) {
			continue
		}
		if err := db.Model(&Opportunity{}).
			Where("id = ?", opportunity.ID).
			Update("product_details", cleaned).Error; err != nil {
			log.Printf("⚠️ Failed to clean opportunity product details for %s: %v", opportunity.FolderNumber, err)
			continue
		}
		repaired++
	}

	if repaired > 0 {
		log.Printf("✅ Cleaned placeholder line items from %d opportunity records", repaired)
	}

	return nil
}

// GetRFQ retrieves a single RFQ by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetRFQ(id uint) (RFQData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return RFQData{}, err
	}
	if a.db == nil {
		return RFQData{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var rfq RFQData
	if err := a.db.First(&rfq, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return RFQData{}, newError("NOT_FOUND", "RFQ not found", fmt.Sprintf("ID: %d", id))
		}
		return RFQData{}, newError("DB_QUERY_FAILED", "Failed to retrieve RFQ", err.Error())
	}

	return rfq, nil
}

// UpdateRFQStatus updates the status of an RFQ
func (a *App) UpdateRFQStatus(id uint, status string) error {
	if err := a.requirePermission("offers:edit"); err != nil {
		return err
	}
	result := a.db.Model(&RFQData{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	log.Printf("✅ Updated RFQ #%d status to: %s", id, status)
	return nil
}

// generateRFQNumber creates a new RFQ number in X-YY format (matching offer numbering convention)
// Example: 1-26, 2-26, etc.
// This implementation is atomic and race-condition-safe using database transactions.
func (a *App) generateRFQNumber() string {
	year := time.Now().Year() % 100 // 26 for 2026
	yearSuffix := fmt.Sprintf("-%02d", year)

	var maxNum int

	// Use GORM transaction (SQLite single-writer ensures serialization)
	err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&RFQData{}).
			Where("rfq_number LIKE ?", "%"+yearSuffix).
			Select("COALESCE(MAX(CAST(SUBSTR(rfq_number, 1, INSTR(rfq_number, '-')-1) AS INTEGER)), 0)").
			Scan(&maxNum).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Warning: RFQ number generation failed: %v, falling back to best-effort", err)
		a.db.Model(&RFQData{}).
			Where("rfq_number LIKE ?", "%"+yearSuffix).
			Select("COALESCE(MAX(CAST(SUBSTR(rfq_number, 1, INSTR(rfq_number, '-')-1) AS INTEGER)), 0)").
			Scan(&maxNum)
	}

	return fmt.Sprintf("%d-%02d", maxNum+1, year)
}

// UpdateRFQStage updates the pipeline stage of an RFQ with state machine validation.
// Valid stages: the canonical enum (see stage_vocabulary.go) — New, Qualified,
// Proposal, Quoted, Won, Lost, Expired, On Hold. Legacy/ad-hoc vocabulary (e.g.
// "RFQ Received", "Order Placed", "Closed (Payment)") is transparently
// canonicalized via the owner-ratified migration map before validation.
// State Machine Invariants:
//   - Lost is TERMINAL: cannot transition to any other stage (revenue inflation guard)
//   - Won is FINAL: can only transition to Lost (payment failure) or stay same
//   - Any stage → Lost allowed (one-way terminal)
func (a *App) UpdateRFQStage(rfqID uint, stage string) error {
	if err := a.requirePermission("offers:edit"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	canonicalStage, _ := canonicalizeOpportunityStage(stage)
	if !isCanonicalOpportunityStage(canonicalStage) {
		return fmt.Errorf("invalid stage %q: must be one of %v", stage, canonicalOpportunityStages)
	}
	stage = canonicalStage

	// STATE MACHINE VALIDATION: Fetch current stage before updating
	var currentRFQ RFQData
	if err := a.db.First(&currentRFQ, "id = ?", rfqID).Error; err != nil {
		return fmt.Errorf("RFQ not found: %v", err)
	}

	// The stored stage may still be legacy vocabulary on a not-yet-migrated
	// row; canonicalize it before enforcing the terminal invariants so the
	// guards fire regardless of migration timing. (`stage` is already
	// canonical from the validation above.)
	currentStage, _ := canonicalizeOpportunityStage(currentRFQ.Stage)

	// INVARIANT: Lost opportunities cannot transition to any other stage
	// (one-way terminal — revenue inflation guard).
	if currentStage == "Lost" {
		return fmt.Errorf("invalid transition: Lost opportunities cannot be changed to '%s' (revenue inflation guard)", stage)
	}

	// INVARIANT: Won (paid/closed) can only transition to Lost (if payment
	// failed) or stay the same.
	if currentStage == "Won" && stage != "Won" && stage != "Lost" {
		return fmt.Errorf("invalid transition: Paid opportunities cannot revert to '%s' (data integrity guard)", stage)
	}

	// VALID TRANSITIONS (enforcing forward progress + Lost/Won as terminal):
	// New → Qualified → Proposal → Quoted → Won; any stage → Lost (one-way).

	if err := a.db.Model(&RFQData{}).Where("id = ?", rfqID).Update("stage", stage).Error; err != nil {
		return fmt.Errorf("failed to update stage: %v", err)
	}

	log.Printf("✅ Updated RFQ #%d stage: %s → %s", rfqID, currentRFQ.Stage, stage)
	return nil
}

func (a *App) UpdateOpportunityStage(opportunityID, stage string) error {
	updated, err := a.UpdateOpportunityStageWithVersion(opportunityID, stage, 0)
	if err != nil {
		return err
	}
	log.Printf("✅ Updated opportunity %s stage to %s (version %d)", opportunityID, updated.Stage, updated.Version)
	return nil
}

// DeleteRFQ deletes an RFQ by ID (only if no linked offers exist)
func (a *App) DeleteRFQ(id uint) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if ok, err := a.guardDeleteOrRequest("offers:edit", "rfq", strconv.FormatUint(uint64(id), 10), fmt.Sprintf("RFQ #%d", id)); !ok {
		return err
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:edit for RFQ deletion
	if err := a.requirePermission("offers:edit"); err != nil {
		log.Printf("🔒 DeleteRFQ blocked: %v", err)
		return err
	}

	// Check for linked offers (via costing sheets or direct reference)
	// Only block if there are active costings with non-rejected/cancelled status
	var activeCostingCount int64
	a.db.Model(&CostingSheetData{}).
		Where("rfq_id = ? AND is_active = ? AND status NOT IN (?, ?)", id, true, "rejected", "cancelled").
		Count(&activeCostingCount)
	if activeCostingCount > 0 {
		return fmt.Errorf("cannot delete RFQ #%d: has active costing sheet(s)", id)
	}

	// Optionally: cascade delete archived/rejected costings
	a.db.Where("rfq_id = ? AND is_active = ?", id, false).Delete(&CostingSheetData{})

	result := a.db.Delete(&RFQData{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete RFQ: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("RFQ #%d not found", id)
	}

	log.Printf("✅ Deleted RFQ #%d", id)
	return nil
}

// UpdateRFQ updates an RFQ's editable fields
func (a *App) UpdateRFQ(id uint, updates RFQUpdateRequest) (*RFQData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:edit for RFQ updates
	if err := a.requirePermission("offers:edit"); err != nil {
		log.Printf("🔒 UpdateRFQ blocked: %v", err)
		return nil, err
	}

	var rfq RFQData
	if err := a.db.First(&rfq, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("RFQ not found: %v", err)
	}

	// SECURITY: Validate status values to prevent arbitrary state changes.
	// The pipeline-stage names are accepted here too so UpdateRFQ can drive the
	// status↔stage sync below (parity with PH).
	validStatuses := map[string]bool{
		"New": true, "In Progress": true, "Quoted": true, "Proposal": true,
		"Won": true, "Lost": true, "Closed": true, "On Hold": true, "": true,
		"RFQ Received": true, "Qualified": true, "Offer Sent": true,
		"Follow-up/Eval": true, "PO/LOI Received": true, "Order Placed": true,
		"In Process": true, "Delivered": true, "Closed (Payment)": true, "Closed (Lost)": true,
	}

	// Update only provided fields with validation
	if updates.Client != "" {
		// SECURITY: Limit client name length
		if len(updates.Client) > 255 {
			updates.Client = updates.Client[:255]
		}
		rfq.Client = updates.Client
	}
	if updates.Project != "" {
		// SECURITY: Limit project name length
		if len(updates.Project) > 500 {
			updates.Project = updates.Project[:500]
		}
		rfq.Project = updates.Project
	}
	if strings.TrimSpace(updates.RFQRef) != "" {
		ref := limitReferenceRunes(updates.RFQRef, 100)
		rfq.RFQRef = ref
		rfq.RFQNumber = limitReferenceRunes(ref, 50)
	}
	if updates.Value > 0 {
		rfq.Value = updates.Value
	}
	if updates.Notes != "" {
		// SECURITY: Limit notes length (10KB max)
		if len(updates.Notes) > 10000 {
			updates.Notes = updates.Notes[:10000]
		}
		rfq.Notes = updates.Notes
	}
	if updates.Status != "" {
		// SECURITY: Only allow valid status transitions
		if !validStatuses[updates.Status] {
			return nil, fmt.Errorf("invalid status: %s", updates.Status)
		}
		rfq.Status = updates.Status
		// Keep Stage synced to Status (parity with PH), but Stage must always
		// land on the canonical enum — canonicalize the derived value and
		// reject rather than persist a non-canonical stage.
		canonicalStage, _ := canonicalizeOpportunityStage(updates.Status)
		if !isCanonicalOpportunityStage(canonicalStage) {
			return nil, fmt.Errorf("invalid stage %q derived from status %q: must be one of %v", canonicalStage, updates.Status, canonicalOpportunityStages)
		}
		rfq.Stage = canonicalStage
	}
	if updates.VisitLocations != "" {
		if len(updates.VisitLocations) > 2000 {
			updates.VisitLocations = updates.VisitLocations[:2000]
		}
		rfq.VisitLocations = updates.VisitLocations
	}
	if updates.ProductDetails != "" {
		if len(updates.ProductDetails) > 5000 {
			updates.ProductDetails = updates.ProductDetails[:5000]
		}
		rfq.ProductDetails = updates.ProductDetails
	}
	if updates.DocumentHash != "" {
		rfq.DocumentHash = updates.DocumentHash
	}
	if updates.SourceDocPath != "" {
		// SECURITY: Limit path length
		if len(updates.SourceDocPath) > 500 {
			updates.SourceDocPath = updates.SourceDocPath[:500]
		}
		rfq.SourceDocPath = updates.SourceDocPath
	}

	if err := a.db.Save(&rfq).Error; err != nil {
		return nil, fmt.Errorf("failed to update RFQ: %v", err)
	}

	log.Printf("✅ Updated RFQ #%d", id)
	return &rfq, nil
}

func (a *App) UpdateRFQNotes(id uint, notes string) (*RFQData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if err := a.requirePermission("offers:edit"); err != nil {
		return nil, err
	}

	var rfq RFQData
	if err := a.db.First(&rfq, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("RFQ not found: %v", err)
	}

	notes = strings.TrimSpace(notes)
	if len(notes) > 10000 {
		notes = notes[:10000]
	}
	rfq.Notes = notes

	if err := a.db.Save(&rfq).Error; err != nil {
		return nil, fmt.Errorf("failed to update RFQ notes: %v", err)
	}

	return &rfq, nil
}

// CheckDuplicateRFQ checks for potential duplicate RFQs by customer, project, or document hash
// Returns: existing RFQ (if found), isDuplicate flag, error
func (a *App) CheckDuplicateRFQ(customer, project, documentHash string) (*RFQData, bool, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, false, err
	}
	if a.db == nil {
		return nil, false, fmt.Errorf("database not initialized")
	}

	var existing RFQData

	// Check 1: Exact document hash match (same file uploaded twice)
	if documentHash != "" {
		if err := a.db.Where("document_hash = ?", documentHash).First(&existing).Error; err == nil {
			log.Printf("⚠️ Duplicate RFQ detected by document hash: RFQ #%d", existing.ID)
			return &existing, true, nil
		}
	}

	// Check 2: Same customer + similar project name (fuzzy match)
	if customer != "" && project != "" {
		// SECURITY FIX: Escape SQL LIKE wildcards to prevent injection
		escapedProject := escapeLikeWildcards(project)
		if err := a.db.Where("client = ? AND project LIKE ? ESCAPE '\\\\'", customer, "%"+escapedProject+"%").First(&existing).Error; err == nil {
			log.Printf("⚠️ Similar RFQ detected: RFQ #%d for %s - %s", existing.ID, existing.Client, existing.Project)
			return &existing, true, nil
		}
	}

	return nil, false, nil
}

// CheckDuplicateOpportunity checks for potential duplicate opportunities by
// folder/reference number first, then by customer + project/title.
func (a *App) CheckDuplicateOpportunity(reference, customer, project string) (*Opportunity, bool, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, false, err
	}
	if a.db == nil {
		return nil, false, fmt.Errorf("database not initialized")
	}

	var existing Opportunity

	ref := strings.TrimSpace(reference)
	if ref != "" {
		if err := a.db.Where("folder_number = ?", ref).First(&existing).Error; err == nil {
			return &existing, true, nil
		}
	}

	customer = strings.TrimSpace(customer)
	project = strings.TrimSpace(project)
	if customer != "" && project != "" {
		escapedProject := escapeLikeWildcards(project)
		if err := a.db.Where("customer_name = ? AND (title LIKE ? ESCAPE '\\\\' OR folder_name LIKE ? ESCAPE '\\\\')",
			customer, "%"+escapedProject+"%", "%"+escapedProject+"%").First(&existing).Error; err == nil {
			return &existing, true, nil
		}
	}

	return nil, false, nil
}

// DeleteRFQWithCascade deletes an RFQ and optionally its linked costing sheets/offers
func (a *App) DeleteRFQWithCascade(id uint, cascade bool) (*DeleteCascadeResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	entityType := "rfq"
	if cascade {
		entityType = "rfq_cascade"
	}
	if ok, err := a.guardDeleteOrRequest("offers:edit", entityType, strconv.FormatUint(uint64(id), 10), fmt.Sprintf("RFQ #%d", id)); !ok {
		return nil, err
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:edit for RFQ cascade delete
	if err := a.requirePermission("offers:edit"); err != nil {
		log.Printf("🔒 DeleteRFQWithCascade blocked: %v", err)
		return nil, err
	}

	result := &DeleteCascadeResult{
		RFQID: id,
	}

	// Count linked records
	var costingCount, offerCount int64
	a.db.Model(&CostingSheetData{}).Where("rfq_id = ?", id).Count(&costingCount)
	a.db.Model(&Offer{}).Where("rfq_id = ?", fmt.Sprintf("%d", id)).Count(&offerCount)

	result.LinkedCostingSheets = int(costingCount)
	result.LinkedOffers = int(offerCount)

	if !cascade && (costingCount > 0 || offerCount > 0) {
		return result, fmt.Errorf("RFQ #%d has %d costing sheet(s) and %d offer(s). Use cascade=true to delete all", id, costingCount, offerCount)
	}

	// Start transaction
	tx := a.db.Begin()

	if cascade {
		// Delete linked costing sheets
		if err := tx.Where("rfq_id = ?", id).Delete(&CostingSheetData{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to delete costing sheets: %v", err)
		}
		result.DeletedCostingSheets = int(costingCount)

		// Delete linked offers and their items
		var offers []Offer
		tx.Where("rfq_id = ?", fmt.Sprintf("%d", id)).Find(&offers)
		for _, offer := range offers {
			tx.Where("offer_id = ?", offer.ID).Delete(&OfferItem{})
		}
		if err := tx.Where("rfq_id = ?", fmt.Sprintf("%d", id)).Delete(&Offer{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to delete offers: %v", err)
		}
		result.DeletedOffers = int(offerCount)

		// Delete RFQ comments
		tx.Where("rfq_id = ?", id).Delete(&RFQComment{})
	}

	// Delete the RFQ
	if err := tx.Delete(&RFQData{}, id).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to delete RFQ: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("transaction failed: %v", err)
	}

	log.Printf("✅ Deleted RFQ #%d (cascade=%v, costings=%d, offers=%d)", id, cascade, result.DeletedCostingSheets, result.DeletedOffers)
	return result, nil
}

// DeleteCascadeResult contains info about what was deleted
type DeleteCascadeResult struct {
	RFQID                uint `json:"rfq_id"`
	LinkedCostingSheets  int  `json:"linked_costing_sheets"`
	LinkedOffers         int  `json:"linked_offers"`
	DeletedCostingSheets int  `json:"deleted_costing_sheets"`
	DeletedOffers        int  `json:"deleted_offers"`
}

// AddRFQComment adds a comment to an RFQ's history log
func (a *App) AddRFQComment(rfqID uint, comment, createdBy string) (*RFQComment, error) {
	if err := a.requirePermission("offers:edit"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SECURITY: Input validation
	if strings.TrimSpace(comment) == "" {
		return nil, fmt.Errorf("comment cannot be empty")
	}
	if len(comment) > 5000 {
		comment = comment[:5000] // Limit to 5KB
	}
	if strings.TrimSpace(createdBy) == "" || createdBy == "User" {
		createdBy = a.getCurrentUserDisplayName()
	}
	if len(createdBy) > 100 {
		createdBy = createdBy[:100]
	}

	// Verify RFQ exists
	var rfq RFQData
	if err := a.db.First(&rfq, "id = ?", rfqID).Error; err != nil {
		return nil, fmt.Errorf("RFQ not found: %v", err)
	}

	rfqComment := &RFQComment{
		RFQID:     rfqID,
		Comment:   comment,
		CreatedBy: createdBy,
	}

	if err := a.db.Create(rfqComment).Error; err != nil {
		return nil, fmt.Errorf("failed to add comment: %v", err)
	}

	log.Printf("✅ Added comment to RFQ #%d by %s", rfqID, createdBy)
	return rfqComment, nil
}

func (a *App) AddOpportunityComment(opportunityID, comment string) (*OpportunityComment, error) {
	if !a.currentSessionIsManagementOrAbove() {
		return nil, fmt.Errorf("opportunity comments are limited to authorized roles")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	opportunityID = strings.TrimSpace(opportunityID)
	comment = strings.TrimSpace(comment)
	if opportunityID == "" {
		return nil, fmt.Errorf("opportunity ID is required")
	}
	if comment == "" {
		return nil, fmt.Errorf("comment cannot be empty")
	}
	if len(comment) > 5000 {
		comment = comment[:5000]
	}

	var opp Opportunity
	if err := a.db.First(&opp, "id = ?", opportunityID).Error; err != nil {
		return nil, fmt.Errorf("opportunity not found: %v", err)
	}

	entry := &OpportunityComment{
		OpportunityID: opportunityID,
		Comment:       comment,
		CreatedBy:     a.getCurrentUserDisplayName(),
	}

	if err := a.db.Create(entry).Error; err != nil {
		return nil, fmt.Errorf("failed to add opportunity comment: %v", err)
	}

	log.Printf("✅ Added comment to opportunity %s by %s", opportunityID, entry.CreatedBy)
	return entry, nil
}

// DeleteOpportunity removes an opportunity through the controlled admin path
// (ported from ph_holdings/user_feedback_hardening_service.go DeleteOpportunity).
//
// PH gated this behind requireAdminDelete + the offers:delete permission. OSS
// has no requireAdminDelete; its equivalent admin gate is guardDeleteOrRequest,
// which lets an admin session delete directly (after the offers:delete permission
// check) and routes any non-admin session into the delete-approval workflow. That
// composes with, and is strictly stronger than, PH's admin-only gate, so it is the
// preferred adaptation here.
func (a *App) DeleteOpportunity(opportunityID string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	opportunityID = strings.TrimSpace(opportunityID)
	if opportunityID == "" {
		return fmt.Errorf("opportunity id is required")
	}

	var opportunity Opportunity
	if err := a.db.First(&opportunity, "id = ?", opportunityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("opportunity not found")
		}
		return fmt.Errorf("failed to load opportunity: %w", err)
	}

	label := firstNonEmptyString(opportunity.FolderNumber, opportunity.Title, opportunity.CustomerName, opportunityID)
	if ok, err := a.guardDeleteOrRequest("offers:delete", "opportunity", opportunity.ID, fmt.Sprintf("Opportunity %s", label)); !ok {
		return err
	}
	if err := a.requirePermission("offers:delete"); err != nil {
		return err
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&Opportunity{}, "id = ?", opportunityID)
		if result.Error != nil {
			return fmt.Errorf("failed to delete opportunity: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("opportunity not found")
		}
		return nil
	}); err != nil {
		return err
	}

	log.Printf("✅ Admin deleted opportunity %s", label)
	return nil
}

func (a *App) DeleteOpportunityComment(commentID string) error {
	if !a.currentSessionIsAdministrator() {
		return fmt.Errorf("only administrators can delete opportunity comments")
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return fmt.Errorf("comment ID is required")
	}

	parsedID, err := strconv.ParseUint(commentID, 10, 64)
	if err != nil || parsedID == 0 {
		return fmt.Errorf("invalid comment ID")
	}

	var comment OpportunityComment
	if err := a.db.First(&comment, "id = ?", uint(parsedID)).Error; err != nil {
		return fmt.Errorf("opportunity comment not found: %v", err)
	}

	if err := a.db.Delete(&comment).Error; err != nil {
		return fmt.Errorf("failed to delete opportunity comment: %v", err)
	}

	log.Printf("🗑️ Deleted comment %d for opportunity %s", comment.ID, comment.OpportunityID)
	return nil
}

// GetRFQComments retrieves all comments for an RFQ in chronological order
func (a *App) GetRFQComments(rfqID uint) ([]RFQComment, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var comments []RFQComment
	if err := a.db.Where("rfq_id = ?", rfqID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve comments: %v", err)
	}

	return comments, nil
}

func (a *App) GetOpportunityComments(opportunityID string) ([]OpportunityComment, error) {
	if !a.currentSessionIsManagementOrAbove() {
		return nil, fmt.Errorf("opportunity comments are limited to authorized roles")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var comments []OpportunityComment
	if err := a.db.Where("opportunity_id = ?", strings.TrimSpace(opportunityID)).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve opportunity comments: %v", err)
	}

	return comments, nil
}

// FindOfferByReference searches for an offer by PO number or other reference
// Used for smart OCR routing when a customer PO references our offer
func (a *App) FindOfferByReference(reference string) (*Offer, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var offer Offer
	// Search by offer number, customer reference, or customer PO number
	// SECURITY FIX: Escape LIKE wildcards to prevent LIKE injection
	escapedRef := escapeLikeWildcards(reference)
	if err := a.db.Where(
		"offer_number = ? OR customer_reference LIKE ? ESCAPE '\\'",
		reference, "%"+escapedRef+"%",
	).First(&offer).Error; err != nil {
		return nil, nil // Not found is not an error
	}

	return &offer, nil
}

// ============================================================================
// COSTING MANAGEMENT
// ============================================================================

// CostingSheetData represents a costing sheet in database (simplified from costing_engine.go)
type CostingSheetData struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	RFQID   uint   `json:"rfq_id" gorm:"index"`
	RFQName string `json:"rfq_name"`

	// FLOW-002: pipeline costings carry RFQID==0 and are instead scoped to a
	// canonical opportunity UUID. Column name matches deployed PH.
	OpportunityRecordID string `json:"opportunity_id" gorm:"column:opportunity_id;index;size:64"`

	// Revision tracking (Feature D)
	RevisionNumber  int   `json:"revision_number" gorm:"default:1"`
	ParentCostingID *uint `json:"parent_costing_id" gorm:"index"`
	IsActive        bool  `json:"is_active" gorm:"default:true;index"`

	Items            string    `json:"items" gorm:"type:text"` // JSON array of CostingItem
	Subtotal         float64   `json:"subtotal"`
	TotalMarkup      float64   `json:"total_markup"`
	FinalPrice       float64   `json:"final_price"`
	MarginPercent    float64   `json:"margin_percent"`
	Status           string    `json:"status" gorm:"default:'draft'"` // draft, pending_approval, approved, rejected
	CreatedBy        string    `json:"created_by"`
	ApprovedBy       string    `json:"approved_by"`
	ApprovalRequired bool      `json:"approval_required"`              // True if margin < 20%
	RiskWarnings     string    `json:"risk_warnings" gorm:"type:text"` // JSON array
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// PH extraction/import provenance columns retained verbatim (PC-D22, Mission
	// I D-I-5): PH's OCR/offer-import populates these on costing_sheet_data.
	OfferNumber    string     `json:"offer_number"`
	CustomerName   string     `json:"customer_name"`
	ProductType    string     `json:"product_type"`
	TotalValueBHD  float64    `json:"total_value_bhd" gorm:"column:total_value_bhd"`
	LineItemCount  int        `json:"line_item_count"`
	SourceFilePath string     `json:"source_file_path"`
	ExtractedAt    *time.Time `json:"extracted_at"`
}

type persistedCostingPayload struct {
	LineItems          []CostingExportLineItem `json:"lineItems"`
	Division           string                  `json:"division"`
	Date               string                  `json:"date"`
	PreparedBy         string                  `json:"preparedBy"`
	CustomerName       string                  `json:"customerName"`
	ContactPerson      string                  `json:"contactPerson"`
	RfqReference       string                  `json:"rfqReference"`
	FolderNumber       string                  `json:"folderNumber"`
	CostingID          string                  `json:"costingId"`
	Subject            string                  `json:"subject"`
	EstDelivery        string                  `json:"estDelivery"`
	DeliveryTerms      string                  `json:"deliveryTerms"`
	PaymentTerms       string                  `json:"paymentTerms"`
	OrderType          string                  `json:"orderType"`
	CountryOfOrigin    string                  `json:"countryOfOrigin"`
	CocCoo             string                  `json:"cocCoo"`
	TestCertificate    string                  `json:"testCertificate"`
	Installation       string                  `json:"installation"`
	Commissioning      string                  `json:"commissioning"`
	Testing            string                  `json:"testing"`
	Subtotal           float64                 `json:"subtotal"`
	Discount           float64                 `json:"discount"`
	NetAmount          float64                 `json:"netAmount"`
	VAT                float64                 `json:"vat"`
	GrandTotal         float64                 `json:"grandTotal"`
	TotalCost          float64                 `json:"totalCost"`
	Profit             float64                 `json:"profit"`
	ProfitPercent      float64                 `json:"profitPercent"`
	QuoteType          string                  `json:"quoteType"`
	VatRate            float64                 `json:"vatRate"`
	HiddenCharges      float64                 `json:"hiddenCharges"`
	PlaceOfSupply      string                  `json:"placeOfSupply"`
	TaxCategory        string                  `json:"taxCategory"`
	CustomerTRN        string                  `json:"customerTRN"`
	Body               string                  `json:"body"`
	TermsAndConditions string                  `json:"termsAndConditions"`
	OpportunityId      uint                    `json:"opportunityId"`
	ProjectName        string                  `json:"projectName"`
}

func parsePersistedCosting(itemsJSON string) (*persistedCostingPayload, error) {
	var payload persistedCostingPayload
	if err := json.Unmarshal([]byte(itemsJSON), &payload); err == nil && (len(payload.LineItems) > 0 || payload.CustomerName != "" || payload.PreparedBy != "" || payload.CostingID != "") {
		return &payload, nil
	}

	var legacyItems []map[string]any
	if err := json.Unmarshal([]byte(itemsJSON), &legacyItems); err != nil {
		return nil, fmt.Errorf("invalid items JSON: %v", err)
	}

	for idx, item := range legacyItems {
		line := CostingExportLineItem{
			SlNo:                idx + 1,
			Equipment:           getStringFromInterface(item["equipment"]),
			Model:               getStringFromInterface(item["model"]),
			LongCode:            getStringFromInterface(item["long_code"]),
			Specification:       getStringFromInterface(item["specification"]),
			DetailedDescription: getStringFromInterface(item["detailed_description"]),
			Quantity:            int(getFloatFromAny(item["quantity"])),
			TotalCost:           getFloatFromAny(item["total_cost"]),
			SuggestedPrice:      getFloatFromAny(item["selling_price"]),
			FOB:                 getFloatFromAny(item["unit_cost"]),
		}
		marginPercent := getFloatFromAny(item["margin_percent"])
		if marginPercent > 0 && marginPercent <= 1 {
			marginPercent *= 100
		}
		line.MarginPercent = marginPercent
		if line.Quantity == 0 {
			line.Quantity = 1
		}
		line.TotalPrice = line.SuggestedPrice * float64(line.Quantity)
		payload.LineItems = append(payload.LineItems, line)
	}

	return &payload, nil
}

func getFloatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}

func summarisePersistedCosting(payload *persistedCostingPayload) (subtotal, totalMarkup, finalPrice, marginPercent float64) {
	if payload == nil {
		return 0, 0, 0, 0
	}

	if payload.Subtotal > 0 || payload.GrandTotal > 0 || payload.TotalCost > 0 {
		subtotal = payload.TotalCost
		finalPrice = payload.GrandTotal
		if finalPrice == 0 {
			finalPrice = payload.NetAmount + payload.VAT
		}
		if subtotal == 0 {
			subtotal = payload.Subtotal - payload.Discount
		}
		if payload.Profit != 0 {
			totalMarkup = payload.Profit
		} else if finalPrice > 0 && subtotal > 0 {
			totalMarkup = finalPrice - subtotal
		}
		if payload.ProfitPercent != 0 {
			marginPercent = payload.ProfitPercent
		} else if finalPrice > 0 && totalMarkup > 0 {
			marginPercent = (totalMarkup / finalPrice) * 100
		}
		return
	}

	for _, item := range payload.LineItems {
		lineSubtotal := item.TotalCost * float64(item.Quantity)
		lineFinal := item.TotalPrice
		if lineFinal == 0 && item.SuggestedPrice > 0 {
			lineFinal = item.SuggestedPrice * float64(item.Quantity)
		}
		subtotal += lineSubtotal
		finalPrice += lineFinal
		totalMarkup += (lineFinal - lineSubtotal)
	}

	if finalPrice > 0 && totalMarkup > 0 {
		marginPercent = (totalMarkup / finalPrice) * 100
	}
	return
}

// CreateCostingSheet creates a new costing sheet linked to an RFQ
func (a *App) CreateCostingSheet(rfqID uint, itemsJSON string, createdBy string) (*CostingSheetData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:create for costing sheets
	if err := a.requirePermission("offers:create"); err != nil {
		log.Printf("🔒 CreateCostingSheet blocked: %v", err)
		return nil, err
	}

	// Fetch RFQ to get name
	var rfq RFQData
	if err := a.db.First(&rfq, rfqID).Error; err != nil {
		return nil, fmt.Errorf("RFQ not found: %v", err)
	}

	// Find the highest revision number for this RFQ
	var maxRevision int
	a.db.Model(&CostingSheetData{}).
		Where("rfq_id = ?", rfqID).
		Select("COALESCE(MAX(revision_number), 0)").
		Scan(&maxRevision)

	newRevision := maxRevision + 1

	// If there are existing revisions, mark them inactive
	if maxRevision > 0 {
		a.db.Model(&CostingSheetData{}).
			Where("rfq_id = ? AND is_active = ?", rfqID, true).
			Update("is_active", false)
	}

	// Find parent costing (the previous active one)
	var parentID *uint
	if maxRevision > 0 {
		var parentCosting CostingSheetData
		if err := a.db.Where("rfq_id = ? AND revision_number = ?", rfqID, maxRevision).
			First(&parentCosting).Error; err == nil {
			parentID = &parentCosting.ID
		}
	}

	payload, err := parsePersistedCosting(itemsJSON)
	if err != nil {
		return nil, err
	}

	subtotal, totalMarkup, finalPrice, marginPercent := summarisePersistedCosting(payload)

	// Determine if approval required (margin < 20%)
	approvalRequired := marginPercent < 20.0

	status := "draft"
	if approvalRequired {
		status = "pending_approval"
	}

	// Generate risk warnings based on margin
	riskWarnings := []string{}
	if marginPercent < 8.0 {
		riskWarnings = append(riskWarnings, fmt.Sprintf("Very low margin (%.1f%%) - consider rejecting", marginPercent))
	} else if marginPercent < 20.0 {
		riskWarnings = append(riskWarnings, fmt.Sprintf("Low margin (%.1f%%) - requires approval", marginPercent))
	}
	riskWarningsJSON, _ := json.Marshal(riskWarnings)

	costing := &CostingSheetData{
		RFQID:            rfqID,
		RFQName:          fmt.Sprintf("%s - %s", rfq.Client, rfq.Project),
		RevisionNumber:   newRevision,
		ParentCostingID:  parentID,
		IsActive:         true,
		Items:            itemsJSON,
		Subtotal:         subtotal,
		TotalMarkup:      totalMarkup,
		FinalPrice:       finalPrice,
		MarginPercent:    marginPercent,
		Status:           status,
		CreatedBy:        createdBy,
		ApprovalRequired: approvalRequired,
		RiskWarnings:     string(riskWarningsJSON),
	}

	result := a.db.Create(costing)
	if result.Error != nil {
		log.Printf("❌ Failed to create costing sheet: %v", result.Error)
		return nil, result.Error
	}

	log.Printf("✅ Created Costing Sheet #%d Rev %d for RFQ #%d (Margin: %.1f%%)", costing.ID, costing.RevisionNumber, rfqID, marginPercent)
	return costing, nil
}

// GetCostingsByRFQ returns all costing sheet revisions for an RFQ
func (a *App) GetCostingsByRFQ(rfqID uint) ([]CostingSheetData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var costings []CostingSheetData
	err := a.db.Where("rfq_id = ?", rfqID).
		Order("revision_number DESC").
		Find(&costings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get costings: %w", err)
	}

	log.Printf("✅ Retrieved %d costing revisions for RFQ #%d", len(costings), rfqID)
	return costings, nil
}

// GetActiveCostingByRFQ returns the current active costing for an RFQ
func (a *App) GetActiveCostingByRFQ(rfqID uint) (*CostingSheetData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var costing CostingSheetData
	err := a.db.Where("rfq_id = ? AND is_active = ?", rfqID, true).
		First(&costing).Error

	if err != nil {
		return nil, fmt.Errorf("no active costing found: %w", err)
	}

	log.Printf("✅ Retrieved active costing #%d (Rev %d) for RFQ #%d", costing.ID, costing.RevisionNumber, rfqID)
	return &costing, nil
}

// SetActiveCostingRevision marks a specific revision as the active one
func (a *App) SetActiveCostingRevision(costingID uint) error {
	if err := a.requirePermission("offers:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Get the costing to find its RFQ
	var costing CostingSheetData
	if err := a.db.First(&costing, costingID).Error; err != nil {
		return fmt.Errorf("costing not found: %w", err)
	}

	// Deactivate all revisions for this RFQ
	a.db.Model(&CostingSheetData{}).
		Where("rfq_id = ?", costing.RFQID).
		Update("is_active", false)

	// Activate the selected one
	err := a.db.Model(&CostingSheetData{}).
		Where("id = ?", costingID).
		Update("is_active", true).Error

	if err != nil {
		return fmt.Errorf("failed to set active revision: %w", err)
	}

	log.Printf("✅ Set Costing #%d (Rev %d) as active for RFQ #%d", costingID, costing.RevisionNumber, costing.RFQID)
	return nil
}

// CloneCostingAsNewRevision creates a new revision by cloning an existing costing
func (a *App) CloneCostingAsNewRevision(sourceCostingID uint, preparedBy string) (*CostingSheetData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:create for costing sheets
	if err := a.requirePermission("offers:create"); err != nil {
		log.Printf("🔒 CloneCostingAsNewRevision blocked: %v", err)
		return nil, err
	}

	// Get source costing
	var source CostingSheetData
	if err := a.db.First(&source, sourceCostingID).Error; err != nil {
		return nil, fmt.Errorf("source costing not found: %w", err)
	}

	// Create new revision using the source items
	log.Printf("🔄 Cloning Costing #%d (Rev %d) as new revision for RFQ #%d", sourceCostingID, source.RevisionNumber, source.RFQID)
	return a.CreateCostingSheet(source.RFQID, source.Items, preparedBy)
}

// GetCostingSheets retrieves all costing sheets
func (a *App) GetCostingSheets(limit int) ([]CostingSheetData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	var costings []CostingSheetData
	result := a.db.Order("created_at DESC")
	if limit > 0 {
		result = result.Limit(limit)
	}
	result = result.Find(&costings)

	if result.Error != nil {
		log.Printf("❌ Error retrieving costing sheets: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve costing sheets: %w", result.Error)
	}

	log.Printf("✅ Retrieved %d costing sheets", len(costings))
	return costings, nil
}

// GetCostingSheet retrieves a single costing sheet by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetCostingSheet(id uint) (CostingSheetData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return CostingSheetData{}, err
	}
	if a.db == nil {
		return CostingSheetData{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var costing CostingSheetData
	if err := a.db.First(&costing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return CostingSheetData{}, newError("NOT_FOUND", "Costing sheet not found", fmt.Sprintf("ID: %d", id))
		}
		return CostingSheetData{}, newError("DB_QUERY_FAILED", "Failed to retrieve costing sheet", err.Error())
	}

	return costing, nil
}

// UpdateCostingSheet updates an existing costing sheet
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) UpdateCostingSheet(id uint, data CostingSheetData) (*CostingSheetData, error) {
	if err := a.requirePermission("offers:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if costing sheet exists
	var existing CostingSheetData
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newError("NOT_FOUND", "Costing sheet not found", fmt.Sprintf("ID: %d", id))
		}
		return nil, newError("DB_QUERY_FAILED", "Failed to retrieve costing sheet", err.Error())
	}

	// Preserve ID and timestamps
	data.ID = existing.ID
	data.CreatedAt = existing.CreatedAt

	// Mission I (I-12): approval-workflow and revision-lineage fields are
	// server-owned — Status/ApprovedBy/ApprovalRequired change only through
	// ApproveCostingSheet (which enforces margin rules), and revision fields
	// only through CreateCostingRevision. A client payload must not be able to
	// mass-assign a costing to "approved".
	if err := a.db.Model(&existing).
		Omit("Status", "ApprovedBy", "ApprovalRequired", "RevisionNumber", "ParentCostingID", "IsActive").
		Updates(data).Error; err != nil {
		return nil, newError("DB_UPDATE_FAILED", "Failed to update costing sheet", err.Error())
	}

	// Reload
	if err := a.db.First(&existing, "id = ?", id).Error; err != nil {
		return nil, newError("DB_QUERY_FAILED", "Failed to reload costing sheet", err.Error())
	}

	log.Printf("✅ Updated Costing Sheet #%d", id)
	return &existing, nil
}

// DeleteCostingSheet deletes a costing sheet by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) DeleteCostingSheet(id uint) error {
	if ok, err := a.guardDeleteOrRequest("offers:delete", "costing_sheet", strconv.FormatUint(uint64(id), 10), fmt.Sprintf("Costing Sheet #%d", id)); !ok {
		return err
	}
	if err := a.requirePermission("offers:delete"); err != nil {
		return err
	}
	if a.db == nil {
		return newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	// Check if costing sheet exists
	var costing CostingSheetData
	if err := a.db.First(&costing, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return newError("NOT_FOUND", "Costing sheet not found", fmt.Sprintf("ID: %d", id))
		}
		return newError("DB_QUERY_FAILED", "Failed to retrieve costing sheet", err.Error())
	}

	// Delete costing sheet
	if err := a.db.Delete(&costing).Error; err != nil {
		return newError("DB_DELETE_FAILED", "Failed to delete costing sheet", err.Error())
	}

	log.Printf("🗑️ Deleted Costing Sheet #%d", id)
	return nil
}

// ApproveCostingSheet approves a costing sheet (manager action)
func (a *App) ApproveCostingSheet(id uint, approvedBy string) error {
	if err := a.requirePermission("offers:update"); err != nil {
		return err
	}

	// Load costing sheet to validate business invariants before approval
	var costing CostingSheetData
	if err := a.db.First(&costing, id).Error; err != nil {
		return fmt.Errorf("costing sheet #%d not found: %w", id, err)
	}

	// Look up customer grade via RFQ linkage for business rule validation
	var customerGrade string
	var hasABB bool
	if costing.RFQID > 0 {
		var rfq RFQData
		if err := a.db.First(&rfq, costing.RFQID).Error; err != nil {
			log.Printf("WARNING: ApproveCostingSheet — RFQ #%d not found: %v (grade checks will use empty grade)", costing.RFQID, err)
		} else if rfq.Client != "" {
			// Try exact match first, then case-insensitive LIKE fallback
			var customer CustomerMaster
			if err := a.db.Where("business_name = ?", rfq.Client).First(&customer).Error; err != nil {
				// Case-insensitive fallback for name mismatches
				if err2 := a.db.Where("UPPER(business_name) = UPPER(?)", rfq.Client).First(&customer).Error; err2 != nil {
					log.Printf("WARNING: ApproveCostingSheet — customer '%s' not found for RFQ #%d (grade rules will be skipped for this customer)", rfq.Client, costing.RFQID)
				} else {
					customerGrade = customer.PaymentGrade
					hasABB = customer.HasABBCompetition
				}
			} else {
				customerGrade = customer.PaymentGrade
				hasABB = customer.HasABBCompetition
			}
		}
	}

	// Validate business rules: minimum margin, grade-based advance requirements, ABB competition
	// Note: advance is not tracked on costing sheets; Grade C/D advance rules are enforced
	// at order/payment stage. The margin check (8% minimum, 15% for ABB) is always enforced here.
	if err := ValidateCostingApproval(
		customerGrade,
		costing.MarginPercent/100.0, // stored as percentage, function expects decimal
		0,                           // advance not tracked on costing sheet — enforced at order/payment stage
		hasABB,
	); err != nil {
		return fmt.Errorf("business rule violation — cannot approve: %w", err)
	}

	result := a.db.Model(&CostingSheetData{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      "approved",
			"approved_by": approvedBy,
		})
	if result.Error != nil {
		return result.Error
	}
	log.Printf("✅ Approved Costing Sheet #%d by %s", id, approvedBy)
	return nil
}

// RejectCostingSheet rejects a costing sheet
func (a *App) RejectCostingSheet(id uint, rejectedBy string) error {
	if err := a.requirePermission("offers:update"); err != nil {
		return err
	}
	result := a.db.Model(&CostingSheetData{}).
		Where("id = ?", id).
		Update("status", "rejected")
	if result.Error != nil {
		return result.Error
	}
	log.Printf("❌ Rejected Costing Sheet #%d by %s", id, rejectedBy)
	return nil
}

// ============================================================================
// OFFER MANAGEMENT
// ============================================================================

// OfferData represents a simplified customer offer
type OfferData struct {
	ID           string    `json:"id" gorm:"primaryKey;size:36"`
	CostingID    string    `json:"costing_id" gorm:"index;size:36"`
	CustomerName string    `json:"customer_name"`
	ProjectName  string    `json:"project_name"`
	Amount       float64   `json:"amount"`
	Status       string    `json:"status" gorm:"default:'draft'"` // draft, sent, accepted, rejected
	PDFPath      string    `json:"pdf_path"`
	SentAt       time.Time `json:"sent_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateOffer generates an offer from an approved costing sheet
func (a *App) CreateOffer(costingID string) (*OfferData, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// SERVER-SIDE PERMISSION CHECK: Require offers:create or admin (wildcard)
	if err := a.requirePermission("offers:create"); err != nil {
		log.Printf("🔒 CreateOffer blocked: %v", err)
		return nil, err
	}

	// Fetch costing sheet
	var costing CostingSheetData
	if err := a.db.First(&costing, costingID).Error; err != nil {
		return nil, fmt.Errorf("costing sheet not found: %v", err)
	}

	// Verify it's approved
	if costing.Status != "approved" {
		return nil, fmt.Errorf("costing sheet must be approved before creating offer")
	}

	// Validate amount
	if costing.FinalPrice <= 0 || math.IsNaN(costing.FinalPrice) || math.IsInf(costing.FinalPrice, 0) {
		return nil, fmt.Errorf("offer amount must be a positive number (got %.3f BHD)", costing.FinalPrice)
	}

	offer := &OfferData{
		CostingID:    costingID,
		CustomerName: costing.RFQName,
		ProjectName:  costing.RFQName,
		Amount:       math.Round(costing.FinalPrice*1000) / 1000,
		Status:       "draft",
	}

	result := a.db.Create(offer)
	if result.Error != nil {
		log.Printf("❌ Failed to create offer: %v", result.Error)
		return nil, result.Error
	}

	log.Printf("✅ Created Offer #%s from Costing #%s (%.2f BHD)", offer.ID, costingID, offer.Amount)
	return offer, nil
}

// GetOffers retrieves all offers
func (a *App) GetOffers(limit int) ([]OfferData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	var offers []OfferData
	result := a.db.Order("created_at DESC")
	if limit > 0 {
		result = result.Limit(limit)
	}
	result = result.Find(&offers)

	if result.Error != nil {
		log.Printf("❌ Error retrieving offers: %v", result.Error)
		return nil, fmt.Errorf("failed to retrieve offers: %w", result.Error)
	}

	log.Printf("✅ Retrieved %d offers", len(offers))
	return offers, nil
}

// GetOffer retrieves a single offer by ID
// P3 FIX: Added missing CRUD operation for consistency
func (a *App) GetOffer(id uint) (OfferData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return OfferData{}, err
	}
	if a.db == nil {
		return OfferData{}, newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var offer OfferData
	if err := a.db.First(&offer, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return OfferData{}, newError("NOT_FOUND", "Offer not found", fmt.Sprintf("ID: %d", id))
		}
		return OfferData{}, newError("DB_QUERY_FAILED", "Failed to retrieve offer", err.Error())
	}

	return offer, nil
}

func (a *App) GetOpportunityLineItems(opportunityID string) ([]OfferItem, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	opportunityID = strings.TrimSpace(opportunityID)
	if opportunityID == "" {
		return nil, fmt.Errorf("opportunity ID is required")
	}

	var opportunity Opportunity
	if err := a.db.First(&opportunity, "id = ?", opportunityID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("opportunity not found")
		}
		return nil, fmt.Errorf("failed to load opportunity: %w", err)
	}

	fallbackItems := parseOpportunityProductDetails(opportunity.ProductDetails)

	if strings.TrimSpace(opportunity.OfferID) == "" {
		return fallbackItems, nil
	}

	optimizer := NewQueryOptimizer(a.db, a.cache)
	offer, err := optimizer.GetOfferWithItems(opportunity.OfferID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fallbackItems, nil
		}
		return nil, fmt.Errorf("failed to load linked offer: %w", err)
	}

	items := offer.Items
	if len(items) == 0 && len(fallbackItems) > 0 {
		return fallbackItems, nil
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LineNumber != items[j].LineNumber {
			return items[i].LineNumber < items[j].LineNumber
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}

// UpdateOfferStatus updates offer status (sent, accepted, rejected)
func (a *App) UpdateOfferStatus(id uint, status string) error {
	if err := a.requirePermission("offers:update"); err != nil {
		return err
	}
	updates := map[string]any{
		"status": status,
	}

	// If sending, record timestamp
	if status == "sent" {
		updates["sent_at"] = time.Now()
	}

	result := a.db.Model(&OfferData{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	log.Printf("✅ Updated Offer #%d status to: %s", id, status)
	return nil
}

// ConvertOfferToOrder converts an accepted offer into an order
// FIXED: Now copies OfferItems to OrderItems for proper invoice generation
func (a *App) ConvertOfferToOrder(offerID uint) error {
	if err := a.requirePermission("orders:create"); err != nil {
		return err
	}
	// Fetch offer with items preloaded
	var offer Offer
	if err := a.db.Preload("Items").First(&offer, "id = ?", fmt.Sprintf("%d", offerID)).Error; err != nil {
		// Fallback: try OfferData for legacy compatibility
		var offerData OfferData
		if err := a.db.First(&offerData, offerID).Error; err != nil {
			return fmt.Errorf("offer not found: %v", err)
		}
		// Convert legacy OfferData path
		return a.convertLegacyOfferToOrder(offerData)
	}

	// Verify it's accepted/won
	if offer.Stage != "Won" && offer.Stage != "accepted" {
		return fmt.Errorf("offer must be Won/accepted before converting to order (current: %s)", offer.Stage)
	}

	// Lookup customer ID from offer or by name
	customerID := offer.CustomerID
	if customerID == "" {
		var customer CustomerMaster
		if err := a.db.Where("business_name = ?", offer.CustomerName).First(&customer).Error; err == nil {
			customerID = customer.ID
		}
	}

	// Generate order number (format: ORD-PHYY/NNN)
	now := time.Now()
	var orderCount int64
	a.db.Model(&Order{}).Where("order_number LIKE ?", fmt.Sprintf("ORD-PH%s/%%", now.Format("06"))).Count(&orderCount)
	orderNumber := fmt.Sprintf("ORD-PH%s/%03d", now.Format("06"), orderCount+1)

	// Create Order record with full offer data
	orderID := uuid.New().String()
	order := &Order{
		Base:              Base{ID: orderID, CreatedAt: now, UpdatedAt: now},
		OrderNumber:       orderNumber,
		CustomerPONumber:  offer.CustomerReference,
		CustomerID:        customerID,
		CustomerName:      offer.CustomerName,
		OrderDate:         now,
		RequiredDate:      now.AddDate(0, 0, 30),
		TotalValueBHD:     offer.TotalValueBHD,
		GrandTotalBHD:     offer.TotalValueBHD,
		Status:            "Confirmed",
		PaymentTerms:      "Net 30",
		DeliveryTerms:     "DDP Bahrain",
		OfferID:           offer.ID,
		OfferNumber:       offer.OfferNumber,
		RFQID:             offer.RFQID,
		CustomerReference: offer.CustomerReference,
		AttentionPerson:   offer.AttentionPerson,
		AttentionCompany:  offer.AttentionCompany,
		AttentionPhone:    offer.AttentionPhone,
		AttentionAddress:  offer.AttentionAddress,
		DeliveryWeeks:     offer.DeliveryWeeks,
		CountryOfOrigin:   offer.CountryOfOrigin,
		IssuedBy:          offer.IssuedBy,
		DiscountPercent:   offer.DiscountPercent,
	}

	// Create order, items, and update offer stage atomically in a transaction
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		// Create order in database
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("failed to create order: %v", err)
		}

		// Copy OfferItems to OrderItems
		for i, offerItem := range offer.Items {
			orderItem := OrderItem{
				Base:                Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
				OrderID:             orderID,
				LineNumber:          i + 1,
				ProductID:           offerItem.ProductID,
				ProductCode:         offerItem.ProductCode,
				Description:         offerItem.Description,
				Quantity:            offerItem.Quantity,
				UnitPrice:           offerItem.UnitPrice,
				QuantityShipped:     0,
				QuantityInvoiced:    0,
				Equipment:           offerItem.Equipment,
				Model:               offerItem.Model,
				Specification:       offerItem.Specification,
				DetailedDescription: offerItem.DetailedDescription,
				Currency:            offerItem.Currency,
				FOB:                 offerItem.FOB,
				Freight:             offerItem.Freight,
				TotalCost:           offerItem.TotalCost,
				MarginPercent:       offerItem.MarginPercent,
				TotalPrice:          offerItem.TotalPrice,
			}
			if err := tx.Create(&orderItem).Error; err != nil {
				return fmt.Errorf("failed to create order item %d: %v", i+1, err)
			}
		}

		// Update offer stage to indicate it's been converted
		if err := tx.Model(&offer).Update("stage", "Converted").Error; err != nil {
			return fmt.Errorf("failed to update offer stage to 'Converted': %v", err)
		}

		return nil
	}); err != nil {
		return err
	}

	log.Printf("✅ Converted Offer %s to Order %s (%d items, %.3f BHD)",
		offer.OfferNumber, orderNumber, len(offer.Items), offer.TotalValueBHD)
	return nil
}

// convertLegacyOfferToOrder handles conversion for legacy OfferData records
func (a *App) convertLegacyOfferToOrder(offer OfferData) error {
	now := time.Now()

	// Lookup customer ID
	var customerID string
	var customer CustomerMaster
	if err := a.db.Where("business_name = ?", offer.CustomerName).First(&customer).Error; err == nil {
		customerID = customer.ID
	}

	// Generate order number
	var orderCount int64
	a.db.Model(&Order{}).Where("order_number LIKE ?", fmt.Sprintf("ORD-PH%s/%%", now.Format("06"))).Count(&orderCount)
	orderNumber := fmt.Sprintf("ORD-PH%s/%03d", now.Format("06"), orderCount+1)

	orderID := uuid.New().String()
	order := &Order{
		Base:          Base{ID: orderID, CreatedAt: now, UpdatedAt: now},
		OrderNumber:   orderNumber,
		CustomerID:    customerID,
		CustomerName:  offer.CustomerName,
		OrderDate:     now,
		RequiredDate:  now.AddDate(0, 0, 30),
		TotalValueBHD: offer.Amount,
		GrandTotalBHD: offer.Amount,
		Status:        "Confirmed",
		PaymentTerms:  "Net 30",
		DeliveryTerms: "DDP Bahrain",
	}

	if err := a.db.Create(order).Error; err != nil {
		return fmt.Errorf("failed to create order: %v", err)
	}

	// Create single line item for legacy offer
	orderItem := OrderItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OrderID:     orderID,
		LineNumber:  1,
		Description: fmt.Sprintf("Per Offer ID %s - %s", offer.ID, offer.ProjectName),
		Quantity:    1,
		UnitPrice:   offer.Amount,
		TotalPrice:  offer.Amount,
	}
	a.db.Create(&orderItem)

	// Update offer status
	a.db.Model(&offer).Update("status", "converted")

	log.Printf("✅ Converted Legacy Offer ID %s to Order %s (%.2f BHD)", offer.ID, orderNumber, offer.Amount)
	return nil
}

// ============================================================================
// OFFER PIPELINE - Costing → Offer → Order Flow
// ============================================================================

// OfferUpdateItem represents a line item in an offer update
type OfferUpdateItem struct {
	Description string  `json:"description"`
	Model       string  `json:"model"`
	Supplier    string  `json:"supplier"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	// Extended costing fields
	Equipment           string  `json:"equipment"`
	ProductCode         string  `json:"product_code"`
	Specification       string  `json:"specification"`
	DetailedDescription string  `json:"detailed_description"`
	Currency            string  `json:"currency"`
	FOB                 float64 `json:"fob"`
	Freight             float64 `json:"freight"`
	TotalCost           float64 `json:"total_cost"`
	MarginPercent       float64 `json:"margin_percent"`
	TotalPrice          float64 `json:"total_price"`
	ExchangeRate        float64 `json:"exchange_rate"`
}

// OfferUpdateData represents data for creating or updating an offer
type OfferUpdateData struct {
	OfferNumber       string            `json:"offer_number"`
	CustomerID        string            `json:"customer_id"`
	CustomerName      string            `json:"customer_name"`
	Division          string            `json:"division"`
	ProjectName       string            `json:"project_name"`
	FolderNumber      string            `json:"folder_number"`
	QuotationDate     string            `json:"quotation_date"`
	ValidityDate      string            `json:"validity_date"`
	PaymentTerms      string            `json:"payment_terms"`
	DeliveryTerms     string            `json:"delivery_terms"`
	DeliveryWeeks     string            `json:"delivery_weeks"`
	CountryOfOrigin   string            `json:"country_of_origin"`
	IssuedBy          string            `json:"issued_by"`
	ContactPhone      string            `json:"contact_phone"`
	CustomerReference string            `json:"customer_reference"`
	AttentionPerson   string            `json:"attention_person"`
	AttentionCompany  string            `json:"attention_company"`
	AttentionPhone    string            `json:"attention_phone"`
	AttentionAddress  string            `json:"attention_address"`
	Subject           string            `json:"subject"`
	Body              string            `json:"body"`
	QuoteType         string            `json:"quote_type"`
	VatRate           float64           `json:"vat_rate"`
	Discount          float64           `json:"discount"`
	Stage             string            `json:"stage"` // Won, Lost, Sent, Draft
	Items             []OfferUpdateItem `json:"items"`
}

// defaultExchangeRateToBHD returns the configured rate that converts one unit
// of currency into BHD. Rates are the single source of truth in the active
// overlay (overlay.json), shared with the import-time Rhine parser so the two
// paths can never disagree.
func defaultExchangeRateToBHD(currency string) float64 {
	return activeOverlay.ExchangeRateToBase(currency)
}

func normalizeExchangeRateToBHD(currency string, rate float64) float64 {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" || currency == "BHD" {
		return 1
	}

	defaultRate := defaultExchangeRateToBHD(currency)
	if currency == "EUR" {
		if rate <= 0 || rate > 2 || math.Abs(rate-1) < 0.0001 || math.Abs(rate-0.44) < 0.005 || math.Abs(rate-0.410) < 0.005 {
			return defaultRate
		}
	}
	if rate <= 0 || rate > 2 || math.Abs(rate-1) < 0.0001 {
		return defaultRate
	}
	return rate
}

func commercialLineTotal(quantity, unitPrice, totalPrice float64) float64 {
	if totalPrice > 0 {
		return totalPrice
	}
	if quantity > 0 && unitPrice > 0 {
		return quantity * unitPrice
	}
	return 0
}

func cleanOfferNumber(value string) string {
	return limitReferenceRunes(value, 50)
}

func desiredOfferNumberFromCosting(data CostingExportData, fallback string) string {
	return cleanOfferNumber(firstNonEmptyString(data.FolderNumber, data.CostingId, data.RfqReference, data.OfferNumber, fallback))
}

func offerNumberWithRevision(offerNumber string, revisionNumber int) string {
	base := strings.TrimSpace(offerNumber)
	if base == "" || revisionNumber <= 0 {
		return base
	}
	hasRevisionSuffix := regexp.MustCompile(`(?i)(?:[-_\s]*(?:rev\.?|r)\s*\d+)$`).MatchString(base)
	if hasRevisionSuffix {
		return base
	}
	return fmt.Sprintf("%s-R%d", base, revisionNumber)
}

func (a *App) ensureOfferNumberAvailable(tx *gorm.DB, offerNumber, excludeOfferID string) error {
	if strings.TrimSpace(offerNumber) == "" {
		return fmt.Errorf("offer number is required")
	}

	query := tx.Model(&Offer{}).Where("offer_number = ?", strings.TrimSpace(offerNumber))
	if strings.TrimSpace(excludeOfferID) != "" {
		query = query.Where("id <> ?", strings.TrimSpace(excludeOfferID))
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("offer number %q already exists; use the correct unique folder/reference number", offerNumber)
	}
	return nil
}

func shouldConvertForeignCommercialSubtotal(rawSubtotal, convertedSubtotal, totalCost float64) bool {
	if rawSubtotal <= 0 || convertedSubtotal <= 0 || convertedSubtotal >= rawSubtotal {
		return false
	}
	if totalCost <= 0 {
		return rawSubtotal >= 1000
	}
	rawCostRatio := rawSubtotal / totalCost
	convertedCostRatio := convertedSubtotal / totalCost
	return rawCostRatio >= 2.0 && convertedCostRatio >= 0.60
}

func isSyntheticCommercialSummary(description, productCode, model, equipment string) bool {
	candidates := []string{description, productCode, model, equipment}
	for _, candidate := range candidates {
		normalized := strings.ToLower(strings.TrimSpace(candidate))
		for strings.Contains(normalized, "  ") {
			normalized = strings.ReplaceAll(normalized, "  ", " ")
		}
		normalized = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(normalized, " -"), "-"))
		if normalized == "total for order" || strings.HasPrefix(normalized, "total for order") {
			return true
		}
		if regexp.MustCompile(`^line item\s+\d+$`).MatchString(normalized) {
			return true
		}
	}
	return false
}

func buildOfferItemsFromCostingData(offerID string, data CostingExportData) []OfferItem {
	items := make([]OfferItem, 0, len(data.LineItems))
	for _, item := range data.LineItems {
		if isSyntheticCommercialSummary(item.Equipment, item.Model, item.Model, item.Equipment) {
			continue
		}
		quantity := float64(item.Quantity)
		if quantity <= 0 {
			quantity = 1
		}
		unitPrice := item.SuggestedPrice
		lineTotal := item.TotalPrice
		if lineTotal <= 0 && unitPrice > 0 {
			lineTotal = unitPrice * quantity
		}
		if unitPrice <= 0 && lineTotal > 0 {
			unitPrice = lineTotal / quantity
		}
		items = append(items, OfferItem{
			Base:        Base{ID: uuid.New().String()},
			OfferID:     offerID,
			LineNumber:  len(items) + 1,
			ProductCode: item.Model,
			Model:       item.Model,
			LongCode:    item.LongCode,
			Description: fmt.Sprintf("%s - %s", item.Equipment, item.Model),
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			// Currency/exchange rate describe the source costing currency. UnitPrice and
			// TotalPrice are already customer-facing BHD sale values and must not be
			// multiplied by this rate again.
			Equipment:           item.Equipment,
			Specification:       item.Specification,
			DetailedDescription: item.DetailedDescription,
			Currency:            firstNonEmptyString(item.Currency, "BHD"),
			FOB:                 item.FOB,
			Freight:             item.Freight,
			TotalCost:           item.TotalCost,
			MarginPercent: func() float64 {
				if item.MarkupPercent != 0 {
					return item.MarkupPercent
				}
				return item.MarginPercent
			}(),
			TotalPrice:      lineTotal,
			ExchangeRate:    normalizeExchangeRateToBHD(item.Currency, item.ExchangeRate),
			FobBHD:          item.FobBHD,
			FreightBHD:      item.FreightBHD,
			Insurance:       item.Insurance,
			CustomsPercent:  item.CustomsPercent,
			CustomsBHD:      item.CustomsBHD,
			HandlingPercent: item.HandlingPercent,
			HandlingBHD:     item.HandlingBHD,
			FinancePercent:  item.FinancePercent,
			FinanceBHD:      item.FinanceBHD,
			OtherCosts:      item.OtherCosts,
			UserPrice:       item.UserPrice,
			UserPriceSet:    item.UserPriceSet,
		})
	}
	return items
}

// SaveCostingAsOffer creates a proper Offer with OfferItems from costing sheet data
func (a *App) SaveCostingAsOffer(data CostingExportData) (*Offer, error) {
	if strings.TrimSpace(data.OfferID) != "" {
		if err := a.requirePermission("offers:edit"); err != nil {
			return nil, err
		}
		return a.updateOfferFromCostingData(data)
	}

	if err := a.requirePermission("offers:create"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Validate VatRate bounds (0-100)
	if data.VatRate < 0 || data.VatRate > 100 {
		return nil, fmt.Errorf("VAT rate must be between 0%% and 100%%, got %.1f%%", data.VatRate)
	}

	// Validate QuoteType whitelist
	validQuoteTypes := map[string]bool{
		"":                   true, // empty = default Quotation
		"Quotation":          true,
		"Budgetary Quote":    true,
		"Budgetary Estimate": true,
		"Technical Offer":    true,
		"Commercial Offer":   true,
	}
	if !validQuoteTypes[data.QuoteType] {
		return nil, fmt.Errorf("invalid quote type: %q", data.QuoteType)
	}

	// Prevent duplicate offers for the same RFQ
	if data.OpportunityId > 0 {
		var existingCount int64
		a.db.Model(&Offer{}).Where("rfq_id = ? AND stage != ?", data.OpportunityId, "Lost").Count(&existingCount)
		if existingCount > 0 {
			return nil, fmt.Errorf("an active offer already exists for this RFQ (RFQ #%d). Mark it as Lost before creating a new one", data.OpportunityId)
		}
	}

	// Parse dates
	quotationDate := time.Now()
	if data.Date != "" {
		if t, err := time.Parse("2006-01-02", data.Date); err == nil {
			quotationDate = t
		}
	}
	validityDate := quotationDate.AddDate(0, 0, 30) // 30 days default

	// P1 FIX #1: Validate validity date is in the future
	if err := ValidateOfferExpiry(validityDate); err != nil {
		log.Printf("⚠️ Offer validity date validation failed: %v", err)
		return nil, err
	}

	// Find customer ID from name
	customerID := ""
	if data.CustomerName != "" {
		var customer CustomerMaster
		if err := a.db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(data.CustomerName)+"%").First(&customer).Error; err == nil {
			customerID = customer.ID
		}
	}

	// Calculate margin
	margin := float64(0)
	if data.GrandTotal > 0 {
		margin = (data.Profit / data.GrandTotal) * 100
	}

	// P1 FIX #4: Check margin threshold and generate alerts
	marginAlert := CheckMarginThreshold(margin, data.GrandTotal)
	if marginAlert != nil {
		log.Printf("⚠️ %s - Margin: %.2f%% on offer for %s (%.3f BHD)",
			marginAlert.Message, margin, data.CustomerName, data.GrandTotal)
	}

	// Convert uint OpportunityId to string RFQID for type consistency
	rfqIDStr := ""
	if data.OpportunityId > 0 {
		rfqIDStr = fmt.Sprintf("%d", data.OpportunityId)
	}
	customerReference := costingUserReference(data)
	offerNumber := desiredOfferNumberFromCosting(data, "")
	if offerNumber == "" {
		offerNumber = a.generateOfferNumber()
	}

	offer := &Offer{
		Base:            Base{ID: uuid.New().String()},
		OfferNumber:     offerNumber,
		RevisionNumber:  0,
		RFQID:           rfqIDStr, // Link to RFQ (P0 Fix: string type)
		CustomerID:      customerID,
		CustomerName:    data.CustomerName,
		QuotationDate:   quotationDate,
		ValidityDate:    validityDate,
		TotalValueBHD:   data.GrandTotal,
		EstimatedMargin: margin,
		Stage:           "Quoted",
		QuoteType:       data.QuoteType,
		// Commercial terms from costing header
		PaymentTerms:       data.PaymentTerms,
		DeliveryTerms:      data.DeliveryTerms,
		DeliveryWeeks:      data.EstDelivery,
		CountryOfOrigin:    data.CountryOfOrigin,
		IssuedBy:           data.PreparedBy,
		CustomerReference:  customerReference,
		AttentionPerson:    data.ContactPerson,
		AttentionCompany:   data.CustomerName,
		TermsAndConditions: data.TermsAndConditions,
		Subject:            strings.TrimSpace(data.Subject),
		Body:               strings.TrimSpace(data.Body),
		VatRate:            data.VatRate,
		Division:           normalizeDivisionName(data.Division),
		DiscountPercent: func() float64 {
			if data.Subtotal > 0 {
				return (data.Discount / data.Subtotal) * 100
			}
			return 0
		}(),
		// Additional costing header fields (Fix 2026-02-05)
		CocCoo:          data.CocCoo,
		TestCertificate: data.TestCertificate,
		Installation:    data.Installation,
		Commissioning:   data.Commissioning,
		Testing:         data.Testing,
		// I-25: carry the datasheet scope onto the offer so the offer PDF
		// bundles the same technical datasheets as the costing export.
		AttachmentScopeID: normaliseCostingAttachmentScope(data.AttachmentScopeID),
	}

	// Create offer items from costing line items (with full costing data for viewing).
	// Line totals are already BHD sale prices.
	offer.Items = buildOfferItemsFromCostingData(offer.ID, data)
	if len(offer.Items) == 0 {
		return nil, fmt.Errorf("cannot create offer: no valid line items after filtering")
	}

	// Save offer with items in transaction (re-check duplicate to prevent TOCTOU)
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if data.OpportunityId > 0 {
			var dupCount int64
			tx.Model(&Offer{}).Where("rfq_id = ? AND stage != ?", data.OpportunityId, "Lost").Count(&dupCount)
			if dupCount > 0 {
				return fmt.Errorf("an active offer already exists for this RFQ (RFQ #%d)", data.OpportunityId)
			}
		}
		if err := a.ensureOfferNumberAvailable(tx, offer.OfferNumber, ""); err != nil {
			return err
		}
		if err := tx.Create(offer).Error; err != nil {
			return err
		}
		if err := a.syncRFQReferenceFromCosting(tx, data.OpportunityId, customerReference); err != nil {
			log.Printf("⚠️ Warning: Failed to sync RFQ #%d reference from costing: %v", data.OpportunityId, err)
		}
		return nil
	}); err != nil {
		log.Printf("❌ Failed to save offer: %v", err)
		return nil, fmt.Errorf("failed to save offer: %v", err)
	}

	// P1 FIX #4: Log margin alert to database for management review
	if marginAlert != nil {
		if err := a.LogMarginAlert("offer", offer.ID, marginAlert); err != nil {
			log.Printf("⚠️ Failed to log margin alert for offer %s: %v", offer.OfferNumber, err)
		}
	}

	// Update RFQ status and stage if linked to an RFQ
	if data.OpportunityId > 0 {
		// Use UpdateRFQStage to enforce state machine validation
		if err := a.UpdateRFQStage(data.OpportunityId, "Offer Sent"); err != nil {
			log.Printf("⚠️ Warning: Failed to update RFQ #%d stage: %v", data.OpportunityId, err)
		}
		// Update status separately (no validation needed for status)
		if err := a.db.Model(&RFQData{}).Where("id = ?", data.OpportunityId).Update("status", "Proposal").Error; err != nil {
			log.Printf("⚠️ Warning: Failed to update RFQ #%d status: %v", data.OpportunityId, err)
		}
	}

	// Link the opportunity record to this offer (removes it from pipeline view)
	a.linkOfferToOpportunity(offer, data.OpportunityRecordID, data.FolderNumber, customerReference, data.CustomerName, data.ProjectName, "Quoted")

	log.Printf("✅ Created Offer %s for %s (%.3f BHD, %d items)", offerNumber, data.CustomerName, data.GrandTotal, len(data.LineItems))
	return offer, nil
}

func (a *App) syncRFQReferenceFromCosting(tx *gorm.DB, rfqID uint, reference string) error {
	if rfqID == 0 || strings.TrimSpace(reference) == "" {
		return nil
	}

	ref := limitReferenceRunes(reference, 100)
	updates := map[string]any{
		"rfq_ref": ref,
	}

	rfqNumber := limitReferenceRunes(ref, 50)
	if rfqNumber != "" {
		var duplicateCount int64
		if err := tx.Model(&RFQData{}).
			Where("rfq_number = ? AND id <> ?", rfqNumber, rfqID).
			Count(&duplicateCount).Error; err != nil {
			return err
		}
		if duplicateCount == 0 {
			updates["rfq_number"] = rfqNumber
		}
	}

	return tx.Model(&RFQData{}).Where("id = ?", rfqID).Updates(updates).Error
}

func (a *App) updateOfferFromCostingData(data CostingExportData) (*Offer, error) {
	offerID := strings.TrimSpace(data.OfferID)
	if offerID == "" {
		return nil, fmt.Errorf("offer id is required for update")
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if data.VatRate < 0 || data.VatRate > 100 {
		return nil, fmt.Errorf("VAT rate must be between 0%% and 100%%, got %.1f%%", data.VatRate)
	}
	validQuoteTypes := map[string]bool{
		"":                   true,
		"Quotation":          true,
		"Budgetary Quote":    true,
		"Budgetary Estimate": true,
		"Technical Offer":    true,
		"Commercial Offer":   true,
	}
	if !validQuoteTypes[data.QuoteType] {
		return nil, fmt.Errorf("invalid quote type: %q", data.QuoteType)
	}

	var offer Offer
	if err := a.db.Preload("Items").First(&offer, "id = ?", offerID).Error; err != nil {
		return nil, fmt.Errorf("offer not found: %w", err)
	}
	if offer.Stage == "Won" || offer.Stage == "Lost" {
		return nil, fmt.Errorf("cannot edit terminal offer stage '%s'", offer.Stage)
	}

	items := buildOfferItemsFromCostingData(offer.ID, data)
	if len(items) == 0 {
		return nil, fmt.Errorf("cannot update offer: no valid line items after filtering")
	}

	quotationDate := offer.QuotationDate
	if strings.TrimSpace(data.Date) != "" {
		if t, err := time.Parse("2006-01-02", data.Date); err == nil {
			quotationDate = t
		}
	}

	customerID := strings.TrimSpace(data.CustomerID)
	customerName := strings.TrimSpace(data.CustomerName)
	if customerID == "" && customerName != "" {
		var customer CustomerMaster
		if err := a.db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(customerName)+"%").First(&customer).Error; err == nil {
			customerID = customer.ID
			customerName = customer.BusinessName
		}
	}

	subtotal := 0.0
	for _, item := range items {
		subtotal += commercialLineTotal(item.Quantity, item.UnitPrice, item.TotalPrice)
	}
	if data.Subtotal > 0 {
		subtotal = data.Subtotal
	}
	discount := math.Max(0, data.Discount)
	netAmount := math.Max(0, subtotal-discount)
	grandTotal := data.GrandTotal
	if grandTotal <= 0 {
		grandTotal = netAmount + (netAmount * data.VatRate / 100.0)
	}
	totalCost := data.TotalCost
	if totalCost <= 0 {
		totalCost = sumOfferItemCost(items)
	}
	customerReference := costingUserReference(data)
	if desiredOfferNumber := desiredOfferNumberFromCosting(data, offer.OfferNumber); desiredOfferNumber != "" {
		offer.OfferNumber = desiredOfferNumber
	}
	margin := 0.0
	if netAmount > 0 {
		margin = ((netAmount - totalCost) / netAmount) * 100
	}

	offer.CustomerID = customerID
	if customerName != "" {
		offer.CustomerName = customerName
	}
	offer.QuotationDate = quotationDate
	offer.TotalValueBHD = roundTo3(grandTotal)
	offer.EstimatedMargin = roundTo3(margin)
	if strings.TrimSpace(data.QuoteType) != "" {
		offer.QuoteType = strings.TrimSpace(data.QuoteType)
	}
	offer.PaymentTerms = strings.TrimSpace(data.PaymentTerms)
	offer.DeliveryTerms = strings.TrimSpace(data.DeliveryTerms)
	offer.DeliveryWeeks = strings.TrimSpace(data.EstDelivery)
	offer.CountryOfOrigin = strings.TrimSpace(data.CountryOfOrigin)
	offer.IssuedBy = strings.TrimSpace(data.PreparedBy)
	offer.CustomerReference = customerReference
	offer.AttentionPerson = strings.TrimSpace(data.ContactPerson)
	offer.AttentionCompany = strings.TrimSpace(data.CustomerName)
	offer.TermsAndConditions = strings.TrimSpace(data.TermsAndConditions)
	if strings.TrimSpace(data.Subject) != "" {
		offer.Subject = strings.TrimSpace(data.Subject)
	}
	if strings.TrimSpace(data.Body) != "" {
		offer.Body = strings.TrimSpace(data.Body)
	}
	offer.VatRate = data.VatRate
	offer.Division = normalizeDivisionName(data.Division)
	if subtotal > 0 {
		offer.DiscountPercent = (discount / subtotal) * 100
	} else {
		offer.DiscountPercent = 0
	}
	offer.CocCoo = strings.TrimSpace(data.CocCoo)
	offer.TestCertificate = strings.TrimSpace(data.TestCertificate)
	offer.Installation = strings.TrimSpace(data.Installation)
	offer.Commissioning = strings.TrimSpace(data.Commissioning)
	offer.Testing = strings.TrimSpace(data.Testing)
	if scope := normaliseCostingAttachmentScope(data.AttachmentScopeID); scope != "" {
		offer.AttachmentScopeID = scope
	}
	offer.RevisionNumber++

	opportunityUpdates := map[string]any{}
	if strings.TrimSpace(data.FolderNumber) != "" {
		opportunityUpdates["folder_number"] = strings.TrimSpace(data.FolderNumber)
	}
	if customerReference != "" {
		opportunityUpdates["eh_ref"] = customerReference
	}
	if strings.TrimSpace(data.ProjectName) != "" {
		opportunityUpdates["title"] = strings.TrimSpace(data.ProjectName)
	} else if strings.TrimSpace(data.Subject) != "" {
		opportunityUpdates["title"] = strings.TrimPrefix(strings.TrimSpace(data.Subject), "Sub:")
	}
	if strings.TrimSpace(offer.CustomerName) != "" {
		opportunityUpdates["customer_name"] = strings.TrimSpace(offer.CustomerName)
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("offer_id = ?", offer.ID).Delete(&OfferItem{}).Error; err != nil {
			return fmt.Errorf("failed to replace offer items: %w", err)
		}
		if err := tx.Create(&items).Error; err != nil {
			return fmt.Errorf("failed to save offer items: %w", err)
		}
		if err := a.ensureOfferNumberAvailable(tx, offer.OfferNumber, offer.ID); err != nil {
			return fmt.Errorf("failed to update offer: %w", err)
		}
		if err := tx.Save(&offer).Error; err != nil {
			return fmt.Errorf("failed to update offer: %w", err)
		}
		if err := a.syncRFQReferenceFromCosting(tx, data.OpportunityId, customerReference); err != nil {
			log.Printf("⚠️ Warning: Failed to sync RFQ #%d reference from costing update: %v", data.OpportunityId, err)
		}
		if len(opportunityUpdates) > 0 {
			if err := tx.Model(&Opportunity{}).Where("offer_id = ?", offer.ID).Updates(opportunityUpdates).Error; err != nil {
				log.Printf("⚠️ Failed to sync linked opportunity metadata for offer %s: %v", offer.OfferNumber, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	offer.Items = items
	log.Printf("✅ Updated Offer %s from costing sheet (Rev %d, %.3f BHD)", offer.OfferNumber, offer.RevisionNumber, offer.TotalValueBHD)
	return &offer, nil
}

// linkOfferToOpportunity finds the matching opportunity record and updates its offer_id, stage, and visible reference.
// This removes the opportunity from the active pipeline view once an offer has been created.
func (a *App) linkOfferToOpportunity(offer *Offer, opportunityRecordID, folderNumber, reference, customerName, projectName, newStage string) {
	if offer == nil || a.db == nil {
		return
	}

	cleanFolder := strings.TrimSpace(folderNumber)
	cleanReference := strings.TrimSpace(reference)
	cleanCustomer := strings.TrimSpace(customerName)
	cleanProject := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(projectName), "Sub:"))
	buildUpdates := func(includeFolder bool) map[string]any {
		updates := map[string]any{
			"offer_id": offer.ID,
			"stage":    newStage,
		}
		if includeFolder && cleanFolder != "" {
			updates["folder_number"] = cleanFolder
		}
		if cleanReference != "" {
			updates["eh_ref"] = cleanReference
		}
		if cleanCustomer != "" {
			updates["customer_name"] = cleanCustomer
		}
		if cleanProject != "" {
			updates["title"] = cleanProject
		}
		return updates
	}

	// Strategy 0: Match by canonical opportunity UUID from the costing sheet launch.
	if opportunityID := strings.TrimSpace(opportunityRecordID); opportunityID != "" {
		result := a.db.Model(&Opportunity{}).
			Where("id = ? AND (offer_id IS NULL OR offer_id = '' OR offer_id = ?)", opportunityID, offer.ID).
			Updates(buildUpdates(cleanFolder != ""))
		if result.Error != nil {
			log.Printf("⚠️ Failed to link offer %s to opportunity id=%s: %v", offer.OfferNumber, opportunityID, result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("✅ Linked offer %s to opportunity (id=%s, stage→%s)", offer.OfferNumber, opportunityID, newStage)
			return
		}
	}

	// Strategy 1: Match by folder_number (most precise — from costing sheet context)
	if cleanFolder != "" {
		result := a.db.Model(&Opportunity{}).
			Where("folder_number = ? AND (offer_id IS NULL OR offer_id = '')", cleanFolder).
			Updates(buildUpdates(false))
		if result.RowsAffected > 0 {
			log.Printf("✅ Linked offer %s to opportunity (folder_number=%s, stage→%s)", offer.OfferNumber, cleanFolder, newStage)
			return
		}
	}

	// Strategy 2: Match by customer_name + no existing offer (least precise, single match only)
	if cleanCustomer != "" {
		var count int64
		a.db.Model(&Opportunity{}).
			Where("customer_name LIKE ? ESCAPE '\\' AND (offer_id IS NULL OR offer_id = '')", "%"+escapeLikeWildcards(cleanCustomer)+"%").
			Count(&count)
		if count == 1 {
			a.db.Model(&Opportunity{}).
				Where("customer_name LIKE ? ESCAPE '\\' AND (offer_id IS NULL OR offer_id = '')", "%"+escapeLikeWildcards(cleanCustomer)+"%").
				Updates(buildUpdates(false))
			log.Printf("✅ Linked offer %s to opportunity (customer_name=%s, stage→%s)", offer.OfferNumber, cleanCustomer, newStage)
		}
	}
}

// generateOfferNumber creates a new offer number in XX-YY format
// Example: 50-25, 51-25 (matching OneDrive folder convention)
func (a *App) generateOfferNumber() string {
	year := time.Now().Year() % 100 // 25 for 2025
	yearSuffix := fmt.Sprintf("-%02d", year)

	var maxNum int

	// Use GORM transaction (SQLite single-writer ensures serialization)
	err := a.db.Transaction(func(tx *gorm.DB) error {
		// Find highest number for this year within the transaction
		if err := tx.Model(&Offer{}).
			Where("offer_number LIKE ?", "%"+yearSuffix).
			Select("COALESCE(MAX(CAST(SUBSTR(offer_number, 1, INSTR(offer_number, '-')-1) AS INTEGER)), 0)").
			Scan(&maxNum).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Printf("Warning: Offer number generation failed: %v, falling back to best-effort", err)
		a.db.Model(&Offer{}).
			Where("offer_number LIKE ?", "%"+yearSuffix).
			Select("COALESCE(MAX(CAST(SUBSTR(offer_number, 1, INSTR(offer_number, '-')-1) AS INTEGER)), 0)").
			Scan(&maxNum)
	}

	return fmt.Sprintf("%d-%02d", maxNum+1, year)
}

// GetAllOffers retrieves all offers with their items using the proper Offer model
func (a *App) GetAllOffers() ([]Offer, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}

	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var offers []Offer
	query := a.db.Preload("Items").Order("created_at DESC")
	if a.db.Migrator().HasTable("offer_items") {
		query = query.Where("id NOT IN (?)", legacyOfferShellSubquery(a.db))
	}
	if err := query.Find(&offers).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve offers: %v", err)
	}

	for i := range offers {
		var opp Opportunity
		if err := a.db.Select("folder_number, title, eh_ref").Where("offer_id = ?", offers[i].ID).First(&opp).Error; err == nil {
			offers[i].FolderNumber = opp.FolderNumber
			offers[i].ProjectName = opp.Title
			if strings.TrimSpace(offers[i].CustomerReference) == "" {
				offers[i].CustomerReference = opp.EHRef
			}
		}
	}

	log.Printf("✅ Retrieved %d offers with items", len(offers))
	return offers, nil
}

// GetOffersWithNoItems returns offers that have zero line items in offer_items.
// Useful for warning users before PDF generation or invoicing.
func (a *App) GetOffersWithNoItems() ([]map[string]any, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	type offerRow struct {
		ID            string    `gorm:"column:id"`
		OfferNumber   string    `gorm:"column:offer_number"`
		CustomerName  string    `gorm:"column:customer_name"`
		Stage         string    `gorm:"column:stage"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		TotalValueBHD float64   `gorm:"column:total_value_bhd"`
	}

	var rows []offerRow
	err := a.db.Raw(`
		SELECT o.id, o.offer_number, o.customer_name, o.stage, o.created_at, COALESCE(o.total_value_bhd, 0) AS total_value_bhd
		FROM offers o
		LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
		WHERE o.deleted_at IS NULL
		GROUP BY o.id
		HAVING COUNT(oi.id) = 0
		ORDER BY o.created_at DESC
	`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query offers with no items: %w", err)
	}

	result := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		stageNormalized := strings.ToUpper(strings.TrimSpace(r.Stage))
		isLegacyShell := (stageNormalized == "QUOTED" || stageNormalized == "RFQ") && math.Abs(r.TotalValueBHD) < legacyOfferShellAmountTolerance
		result = append(result, map[string]any{
			"id":              r.ID,
			"offer_number":    r.OfferNumber,
			"customer_name":   r.CustomerName,
			"stage":           r.Stage,
			"created_at":      r.CreatedAt,
			"total_value_bhd": r.TotalValueBHD,
			"is_legacy_shell": isLegacyShell,
		})
	}

	log.Printf("📋 Found %d offers with no line items", len(result))
	return result, nil
}

func deriveCanonicalOpportunityFolderFromOfferNumber(offerNumber string) string {
	normalized := normalizeImportedOfferNumber(offerNumber)
	if normalized == "" {
		return ""
	}

	if m := regexp.MustCompile(`^([A-Z]{1,8})-(\d{1,3}[A-Z]?)-(\d{2})$`).FindStringSubmatch(normalized); len(m) == 4 {
		return deriveCanonicalOneDriveFolderNumber(m[1], m[2], parseMetaYear(m[3]))
	}
	if m := regexp.MustCompile(`^(\d{1,3}[A-Z]?)-(\d{2})$`).FindStringSubmatch(normalized); len(m) == 3 {
		return deriveCanonicalOneDriveFolderNumber("", m[1], parseMetaYear(m[2]))
	}
	return ""
}

// BackfillWonOfferItemsFromOpportunityProductDetails repairs won offer shells
// that still have zero offer_items but do have verified imported opportunity
// line items in opportunities.product_details.
// Bound entry point. Mission I (I-11): gated — startup uses the internal.
func (a *App) BackfillWonOfferItemsFromOpportunityProductDetails() (map[string]any, error) {
	if err := a.requirePermission("offers:update"); err != nil {
		return nil, err
	}
	return a.backfillWonOfferItemsFromOpportunityProductDetailsInternal()
}

func (a *App) backfillWonOfferItemsFromOpportunityProductDetailsInternal() (map[string]any, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	type offerShell struct {
		ID          string `gorm:"column:id"`
		OfferNumber string `gorm:"column:offer_number"`
	}

	var shells []offerShell
	if err := a.db.Raw(`
		SELECT o.id, o.offer_number
		FROM offers o
		LEFT JOIN offer_items oi ON oi.offer_id = o.id AND oi.deleted_at IS NULL
		WHERE o.deleted_at IS NULL
		  AND LOWER(COALESCE(o.stage, '')) = 'won'
		GROUP BY o.id
		HAVING COUNT(oi.id) = 0
		ORDER BY o.offer_number ASC
	`).Scan(&shells).Error; err != nil {
		return nil, fmt.Errorf("failed to query won offers with no items: %w", err)
	}

	results := map[string]any{
		"checked":  len(shells),
		"repaired": 0,
		"skipped":  0,
		"relinked": 0,
		"cleaned":  0,
		"examples": []string{},
		"skips":    []string{},
	}

	repairedExamples := make([]string, 0, len(shells))
	skipReasons := make([]string, 0)
	cleaned := 0
	relinked := 0
	repaired := 0

	for _, shell := range shells {
		folderNumber := deriveCanonicalOpportunityFolderFromOfferNumber(shell.OfferNumber)
		if folderNumber == "" {
			skipReasons = append(skipReasons, fmt.Sprintf("%s:no_folder_match", shell.OfferNumber))
			continue
		}

		var opportunity Opportunity
		if err := a.db.
			Where("deleted_at IS NULL AND (source = ? OR source LIKE ?) AND folder_number = ? AND stage = ?", "onedrive_import", "%_onedrive", folderNumber, "Won").
			First(&opportunity).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("failed to load opportunity for offer %s: %w", shell.OfferNumber, err)
			}
			skipReasons = append(skipReasons, fmt.Sprintf("%s:no_verified_opportunity", shell.OfferNumber))
			continue
		}

		items := parseOpportunityProductDetails(opportunity.ProductDetails)
		if len(items) == 0 {
			skipReasons = append(skipReasons, fmt.Sprintf("%s:no_product_details", shell.OfferNumber))
			continue
		}

		previousOfferID := strings.TrimSpace(opportunity.OfferID)
		needsRelink := previousOfferID != "" && previousOfferID != shell.ID
		now := time.Now()

		err := a.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("offer_id = ?", shell.ID).Delete(&OfferItem{}).Error; err != nil {
				return fmt.Errorf("failed to clear stale items for %s: %w", shell.OfferNumber, err)
			}

			dbItems := make([]OfferItem, 0, len(items))
			for idx, item := range items {
				lineTotal := item.TotalPrice
				if lineTotal <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
					lineTotal = item.Quantity * item.UnitPrice
				}
				dbItems = append(dbItems, OfferItem{
					Base:                Base{ID: uuid.New().String()},
					OfferID:             shell.ID,
					LineNumber:          idx + 1,
					ProductCode:         item.ProductCode,
					Model:               item.Model,
					Description:         item.Description,
					Quantity:            item.Quantity,
					UnitPrice:           item.UnitPrice,
					Equipment:           item.Equipment,
					Specification:       item.Specification,
					DetailedDescription: item.DetailedDescription,
					Currency:            item.Currency,
					TotalPrice:          lineTotal,
				})
			}

			if err := tx.Create(&dbItems).Error; err != nil {
				return fmt.Errorf("failed to create repaired offer items for %s: %w", shell.OfferNumber, err)
			}

			if err := tx.Model(&Offer{}).
				Where("id = ?", shell.ID).
				Updates(map[string]any{
					"total_value_bhd": opportunity.RevenueBHD,
					"updated_at":      now,
				}).Error; err != nil {
				return fmt.Errorf("failed to update offer total for %s: %w", shell.OfferNumber, err)
			}

			if err := tx.Model(&Opportunity{}).
				Where("id = ?", opportunity.ID).
				Updates(map[string]any{
					"offer_id":    shell.ID,
					"updated_at":  now,
					"revenue_bhd": opportunity.RevenueBHD,
				}).Error; err != nil {
				return fmt.Errorf("failed to relink opportunity for %s: %w", shell.OfferNumber, err)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		if needsRelink {
			relinked++
			if err := a.softDeleteOfferIfUnlinked(previousOfferID); err == nil {
				cleaned++
			}
		}
		repaired++
		repairedExamples = append(repairedExamples, fmt.Sprintf("%s→%s", shell.OfferNumber, folderNumber))
	}

	results["repaired"] = repaired
	results["relinked"] = relinked
	results["cleaned"] = cleaned
	results["skipped"] = len(skipReasons)
	results["examples"] = repairedExamples
	results["skips"] = skipReasons
	return results, nil
}

// GetOrdersWithNoItems returns orders that have zero line items in order_items.
// Useful for warning users before invoicing or delivery note creation.
func (a *App) GetOrdersWithNoItems() ([]map[string]any, error) {
	if err := a.requirePermission("orders:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	type orderRow struct {
		ID           string    `gorm:"column:id"`
		OrderNumber  string    `gorm:"column:order_number"`
		CustomerName string    `gorm:"column:customer_name"`
		Status       string    `gorm:"column:status"`
		CreatedAt    time.Time `gorm:"column:created_at"`
	}

	var rows []orderRow
	err := a.db.Raw(`
		SELECT o.id, o.order_number, o.customer_name, o.status, o.created_at
		FROM orders o
		LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL
		WHERE o.deleted_at IS NULL
		GROUP BY o.id
		HAVING COUNT(oi.id) = 0
		ORDER BY o.created_at DESC
	`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query orders with no items: %w", err)
	}

	result := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		result = append(result, map[string]any{
			"id":            r.ID,
			"order_number":  r.OrderNumber,
			"customer_name": r.CustomerName,
			"status":        r.Status,
			"created_at":    r.CreatedAt,
		})
	}

	log.Printf("📋 Found %d orders with no line items", len(result))
	return result, nil
}

// UpdateOfferFull updates an offer's details and line items
func (a *App) UpdateOfferFull(offerID string, data OfferUpdateData) (*Offer, error) {
	if err := a.requirePermission("offers:edit"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var offer Offer
	if err := a.db.Preload("Items").First(&offer, "id = ?", offerID).Error; err != nil {
		return nil, fmt.Errorf("offer not found: %v", err)
	}

	// SECURITY: Won and Lost stages are terminal - block regression (audit requirement)
	if offer.Stage == "Won" || offer.Stage == "Lost" {
		if data.Stage != "" && data.Stage != offer.Stage {
			return nil, fmt.Errorf("cannot change terminal offer stage '%s' (audit requirement)", offer.Stage)
		}
	}

	// Update header fields
	if strings.TrimSpace(data.OfferNumber) != "" {
		offer.OfferNumber = cleanOfferNumber(data.OfferNumber)
	}
	if data.CustomerName != "" {
		offer.CustomerName = data.CustomerName
	}
	if data.CustomerID != "" {
		offer.CustomerID = data.CustomerID
	} else if strings.TrimSpace(data.CustomerName) != "" {
		var customer CustomerMaster
		customerName := strings.TrimSpace(data.CustomerName)
		if err := a.db.Where("business_name = ?", customerName).First(&customer).Error; err == nil {
			offer.CustomerID = customer.ID
			offer.CustomerName = customer.BusinessName
		} else if err := a.db.Where("business_name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeWildcards(customerName)+"%").First(&customer).Error; err == nil {
			offer.CustomerID = customer.ID
			offer.CustomerName = customer.BusinessName
		}
	}
	if data.QuotationDate != "" {
		if t, err := time.Parse("2006-01-02", data.QuotationDate); err == nil {
			offer.QuotationDate = t
		}
	}
	if data.ValidityDate != "" {
		if t, err := time.Parse("2006-01-02", data.ValidityDate); err == nil {
			// P1 FIX #1: Validate validity date is in the future
			if err := ValidateOfferExpiry(t); err != nil {
				log.Printf("⚠️ Offer validity date validation failed during update: %v", err)
				return nil, err
			}
			offer.ValidityDate = t
		}
	}
	offer.PaymentTerms = strings.TrimSpace(data.PaymentTerms)
	offer.DeliveryTerms = strings.TrimSpace(data.DeliveryTerms)
	offer.DeliveryWeeks = strings.TrimSpace(data.DeliveryWeeks)
	offer.CountryOfOrigin = strings.TrimSpace(data.CountryOfOrigin)
	offer.IssuedBy = strings.TrimSpace(data.IssuedBy)
	offer.ContactPhone = strings.TrimSpace(data.ContactPhone)
	offer.CustomerReference = strings.TrimSpace(data.CustomerReference)
	offer.AttentionPerson = strings.TrimSpace(data.AttentionPerson)
	offer.AttentionCompany = strings.TrimSpace(data.AttentionCompany)
	offer.AttentionPhone = strings.TrimSpace(data.AttentionPhone)
	offer.AttentionAddress = strings.TrimSpace(data.AttentionAddress)
	offer.Subject = strings.TrimSpace(data.Subject)
	offer.Body = strings.TrimSpace(data.Body)
	if data.QuoteType != "" {
		offer.QuoteType = data.QuoteType
	}
	if data.VatRate >= 0 && data.VatRate <= 100 {
		offer.VatRate = data.VatRate
	}
	if strings.TrimSpace(data.Division) != "" {
		offer.Division = normalizeDivisionName(data.Division)
	}

	// Increment revision
	offer.RevisionNumber++

	opportunityUpdates := map[string]any{}
	if strings.TrimSpace(data.FolderNumber) != "" {
		opportunityUpdates["folder_number"] = strings.TrimSpace(data.FolderNumber)
	}
	if strings.TrimSpace(data.CustomerReference) != "" {
		opportunityUpdates["eh_ref"] = strings.TrimSpace(data.CustomerReference)
	}
	if strings.TrimSpace(data.ProjectName) != "" {
		opportunityUpdates["title"] = strings.TrimSpace(data.ProjectName)
	}
	if strings.TrimSpace(offer.CustomerName) != "" {
		opportunityUpdates["customer_name"] = strings.TrimSpace(offer.CustomerName)
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if len(data.Items) > 0 {
			if err := tx.Where("offer_id = ?", offerID).Delete(&OfferItem{}).Error; err != nil {
				return fmt.Errorf("failed to replace offer items: %w", err)
			}

			totalValue := float64(0)
			offer.Items = nil
			for _, item := range data.Items {
				if isSyntheticCommercialSummary(item.Description, item.ProductCode, item.Model, item.Equipment) {
					continue
				}
				lineTotal := item.TotalPrice
				if lineTotal <= 0 && item.Quantity > 0 && item.UnitPrice > 0 {
					lineTotal = item.Quantity * item.UnitPrice
				}
				unitPrice := item.UnitPrice
				if unitPrice <= 0 && item.Quantity > 0 && lineTotal > 0 {
					unitPrice = lineTotal / item.Quantity
				}
				totalValue += lineTotal
				exchangeRate := normalizeExchangeRateToBHD(item.Currency, item.ExchangeRate)

				offer.Items = append(offer.Items, OfferItem{
					Base:                Base{ID: uuid.New().String()},
					OfferID:             offerID,
					LineNumber:          len(offer.Items) + 1,
					ProductCode:         item.ProductCode,
					Model:               item.Model,
					Description:         item.Description,
					Quantity:            item.Quantity,
					UnitPrice:           unitPrice,
					Equipment:           item.Equipment,
					Specification:       item.Specification,
					DetailedDescription: item.DetailedDescription,
					Currency:            item.Currency,
					FOB:                 item.FOB,
					Freight:             item.Freight,
					TotalCost:           item.TotalCost,
					MarginPercent:       item.MarginPercent,
					TotalPrice:          lineTotal,
					ExchangeRate:        exchangeRate,
				})
			}
			discount := math.Max(0, data.Discount)
			netAmount := math.Max(0, totalValue-discount)
			offer.TotalValueBHD = roundTo3(netAmount + (netAmount * offer.VatRate / 100.0))

			if len(offer.Items) > 0 {
				if err := tx.Create(&offer.Items).Error; err != nil {
					return fmt.Errorf("failed to save offer items: %w", err)
				}
			}
		}

		if err := a.ensureOfferNumberAvailable(tx, offer.OfferNumber, offer.ID); err != nil {
			return fmt.Errorf("failed to update offer: %w", err)
		}
		if err := tx.Save(&offer).Error; err != nil {
			return fmt.Errorf("failed to update offer: %w", err)
		}

		if len(opportunityUpdates) > 0 {
			if err := tx.Model(&Opportunity{}).Where("offer_id = ?", offer.ID).Updates(opportunityUpdates).Error; err != nil {
				log.Printf("⚠️ Failed to sync linked opportunity metadata for offer %s: %v", offer.OfferNumber, err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	log.Printf("✅ Updated Offer %s (Rev %d, %.3f BHD)", offer.OfferNumber, offer.RevisionNumber, offer.TotalValueBHD)
	return &offer, nil
}

// =============================================================================
// OFFER NOTES CRUD
// =============================================================================

// AddOfferNote adds a freeform note to an offer
func (a *App) AddOfferNote(offerID string, content string) (OfferNote, error) {
	if err := a.requirePermission("offers:edit"); err != nil {
		return OfferNote{}, err
	}
	if a.db == nil {
		return OfferNote{}, fmt.Errorf("database not initialized")
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return OfferNote{}, fmt.Errorf("note content cannot be empty")
	}
	if len(trimmed) > 5000 {
		return OfferNote{}, fmt.Errorf("note content too long (max 5000 characters)")
	}
	// Sanitize HTML to prevent stored XSS
	trimmed = html.EscapeString(trimmed)
	note := OfferNote{
		Base:     Base{CreatedBy: "System"},
		OfferID:  offerID,
		NoteDate: time.Now(),
		Content:  trimmed,
	}
	if err := a.db.Create(&note).Error; err != nil {
		return OfferNote{}, fmt.Errorf("failed to create note: %w", err)
	}
	return note, nil
}

// GetOfferNotes returns all notes for an offer, newest first (capped at 200)
func (a *App) GetOfferNotes(offerID string) ([]OfferNote, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var notes []OfferNote
	if err := a.db.Where("offer_id = ?", offerID).Order("note_date desc").Limit(200).Find(&notes).Error; err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}
	return notes, nil
}

// DeleteOffer removes an unconverted offer from the live commercial pipeline
// (ported from ph_holdings/app.go DeleteOffer). Converted/won offers remain
// immutable because orders and invoices depend on them for traceability. The
// linked opportunity is unlinked (and reverted to Proposal for shell stages)
// inside the same transaction that deletes the offer and its children.
func (a *App) DeleteOffer(offerID string) error {
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return fmt.Errorf("offer id is required")
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var offer Offer
	if err := a.db.Where("id = ?", offerID).First(&offer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("offer not found")
		}
		return fmt.Errorf("failed to load offer: %w", err)
	}

	label := strings.TrimSpace(offer.OfferNumber)
	if label == "" {
		label = offer.ID
	}
	if ok, err := a.guardDeleteOrRequest("offers:delete", "offer", offer.ID, fmt.Sprintf("Offer %s", label)); !ok {
		return err
	}
	if err := a.requirePermission("offers:delete"); err != nil {
		return err
	}

	stage := strings.ToLower(strings.TrimSpace(offer.Stage))
	if stage == "won" {
		return fmt.Errorf("won offers cannot be deleted; reverse the linked order flow or mark a replacement through the controlled workflow")
	}

	var linkedOrders int64
	if err := a.db.Model(&Order{}).Where("offer_id = ?", offer.ID).Count(&linkedOrders).Error; err != nil {
		return fmt.Errorf("failed to inspect linked orders: %w", err)
	}
	if linkedOrders > 0 {
		return fmt.Errorf("cannot delete offer %s because %d linked order(s) exist", label, linkedOrders)
	}

	var linkedInvoices int64
	if err := a.db.Model(&Invoice{}).Where("offer_id = ? OR quote_id = ?", offer.ID, offer.ID).Count(&linkedInvoices).Error; err != nil {
		return fmt.Errorf("failed to inspect linked invoices: %w", err)
	}
	if linkedInvoices > 0 {
		return fmt.Errorf("cannot delete offer %s because %d linked invoice(s) exist", label, linkedInvoices)
	}

	return a.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		oppUpdates := map[string]any{
			"offer_id":   "",
			"updated_at": now,
		}
		if stage == "quoted" || stage == "rfq" || stage == "" {
			oppUpdates["stage"] = "Proposal"
		}
		if err := tx.Model(&Opportunity{}).Where("offer_id = ?", offer.ID).Updates(oppUpdates).Error; err != nil {
			return fmt.Errorf("failed to unlink offer from opportunity: %w", err)
		}
		if err := tx.Where("offer_id = ?", offer.ID).Delete(&OfferFollowUp{}).Error; err != nil {
			return fmt.Errorf("failed to delete offer follow-ups: %w", err)
		}
		if err := tx.Where("offer_id = ?", offer.ID).Delete(&OfferNote{}).Error; err != nil {
			return fmt.Errorf("failed to delete offer notes: %w", err)
		}
		if err := tx.Where("offer_id = ?", offer.ID).Delete(&OfferItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete offer items: %w", err)
		}
		if err := tx.Delete(&offer).Error; err != nil {
			return fmt.Errorf("failed to delete offer: %w", err)
		}
		return nil
	})
}

// ============================================================================
// OFFER REVISION / RENEWAL LINEAGE (Mission I — I-19/I-20)
// Ported from ph_holdings. Re-quoting or renewing an offer clones it into a
// fresh Quoted offer, links the clone to its source/root via lineage fields,
// and stamps the source as superseded — all inside one transaction so orders
// and invoices keep an intact audit trail.
// ============================================================================

func offerNumberAvailable(tx *gorm.DB, offerNumber, excludeOfferID string) (bool, error) {
	offerNumber = strings.TrimSpace(offerNumber)
	if offerNumber == "" {
		return false, nil
	}
	query := tx.Unscoped().Model(&Offer{}).Where("offer_number = ?", offerNumber)
	if strings.TrimSpace(excludeOfferID) != "" {
		query = query.Where("id <> ?", strings.TrimSpace(excludeOfferID))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func stripOfferRevisionSuffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	re := regexp.MustCompile(`(?i)[-_\s]*(?:rev\.?|r)\s*\d+$`)
	return strings.TrimSpace(re.ReplaceAllString(value, ""))
}

func offerRevisionSuffixNumber(value string) int {
	matches := regexp.MustCompile(`(?i)(?:[-_\s]*(?:rev\.?|r)\s*)(\d+)$`).FindStringSubmatch(strings.TrimSpace(value))
	if len(matches) != 2 {
		return 0
	}
	parsed, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return parsed
}

func friendlyOfferSaveError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed: offers.offer_number") ||
		strings.Contains(msg, "duplicate key value") && strings.Contains(msg, "offers") && strings.Contains(msg, "offer_number") {
		return fmt.Errorf("offer number already exists; use the existing offer edit flow or create a new revision")
	}
	return err
}

func (a *App) nextOfferRevisionIdentity(tx *gorm.DB, source Offer) (int, string, error) {
	if tx == nil {
		return 0, "", fmt.Errorf("database not initialized")
	}
	revisionRoot := stripOfferRevisionSuffix(firstNonEmptyString(source.OfferNumber, source.CustomerReference))
	if revisionRoot == "" {
		revisionRoot = a.generateOfferNumber()
	}

	start := source.RevisionNumber + 1
	if suffix := offerRevisionSuffixNumber(source.OfferNumber); suffix >= start {
		start = suffix + 1
	}
	if start < 1 {
		start = 1
	}

	for revision := start; revision <= 99; revision++ {
		candidate := cleanOfferNumber(fmt.Sprintf("%s-R%d", revisionRoot, revision))
		available, err := offerNumberAvailable(tx, candidate, "")
		if err != nil {
			return 0, "", err
		}
		if available {
			return revision, candidate, nil
		}
	}
	return 0, "", fmt.Errorf("no available revision number for offer %s", source.OfferNumber)
}

func cloneOfferItemsForRevision(sourceItems []OfferItem, newOfferID string) []OfferItem {
	items := make([]OfferItem, 0, len(sourceItems))
	for idx, sourceItem := range sourceItems {
		item := sourceItem
		item.Base = Base{ID: uuid.New().String()}
		item.OfferID = newOfferID
		if item.LineNumber <= 0 {
			item.LineNumber = idx + 1
		}
		items = append(items, item)
	}
	return items
}

// CreateOfferRevision creates a new quoted offer from an existing offer while
// preserving the source commercial document for audit and order traceability.
func (a *App) CreateOfferRevision(sourceOfferID string) (*Offer, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		if editErr := a.requirePermission("offers:edit"); editErr != nil {
			return nil, err
		}
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	sourceOfferID = strings.TrimSpace(sourceOfferID)
	if sourceOfferID == "" {
		return nil, fmt.Errorf("source offer id is required")
	}

	var created Offer
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var source Offer
		if err := tx.Preload("Items").First(&source, "id = ?", sourceOfferID).Error; err != nil {
			return fmt.Errorf("source offer not found: %w", err)
		}
		if strings.EqualFold(strings.TrimSpace(source.Stage), "Won") {
			return fmt.Errorf("won offers cannot be re-quoted; create an amendment/order workflow instead")
		}
		if len(source.Items) == 0 {
			return fmt.Errorf("cannot create revision for offer %s because it has no line items", source.OfferNumber)
		}

		revisionNumber, offerNumber, err := a.nextOfferRevisionIdentity(tx, source)
		if err != nil {
			return err
		}

		now := time.Now()
		rootOfferID := strings.TrimSpace(source.RevisionRootOfferID)
		if rootOfferID == "" {
			rootOfferID = strings.TrimSpace(source.RevisionOfOfferID)
		}
		if rootOfferID == "" {
			rootOfferID = source.ID
		}

		newOffer := source
		newOffer.Base = Base{ID: uuid.New().String(), CreatedBy: a.getCurrentUserID()}
		newOffer.OfferNumber = offerNumber
		newOffer.RevisionNumber = revisionNumber
		newOffer.RevisionOfOfferID = source.ID
		newOffer.RevisionRootOfferID = rootOfferID
		newOffer.SupersededByOfferID = ""
		newOffer.SupersededAt = nil
		newOffer.Stage = "Quoted"
		newOffer.LostReason = ""
		newOffer.QuotationDate = now
		newOffer.ValidityDate = now.AddDate(0, 0, 30)
		newOffer.Items = nil

		if err := tx.Create(&newOffer).Error; err != nil {
			return friendlyOfferSaveError(err)
		}

		items := cloneOfferItemsForRevision(source.Items, newOffer.ID)
		if err := tx.Create(&items).Error; err != nil {
			return fmt.Errorf("failed to create revision line items: %w", err)
		}

		sourceUpdates := map[string]any{
			"superseded_by_offer_id": newOffer.ID,
			"superseded_at":          now,
			"updated_at":             now,
		}
		if strings.EqualFold(source.Stage, "Quoted") || strings.EqualFold(source.Stage, "RFQ") {
			sourceUpdates["stage"] = "Expired"
		}
		if err := tx.Model(&Offer{}).Where("id = ?", source.ID).Updates(sourceUpdates).Error; err != nil {
			return fmt.Errorf("failed to mark source offer as superseded: %w", err)
		}

		if err := tx.Model(&Opportunity{}).Where("offer_id = ?", source.ID).Updates(map[string]any{
			"offer_id":    newOffer.ID,
			"stage":       "Quoted",
			"updated_at":  now,
			"closed_date": nil,
		}).Error; err != nil {
			log.Printf("⚠️ Failed to relink opportunity to offer revision %s: %v", newOffer.OfferNumber, err)
		}

		newOffer.Items = items
		created = newOffer
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Printf("✅ Created Offer revision %s from %s", created.OfferNumber, sourceOfferID)
	return &created, nil
}

// RenewOffer (FLOW-006) recovers an Expired offer by cloning it into a fresh
// Quoted offer with a new 30-day validity window, linked back to the original via
// the revision lineage. A 90-120 day sales cycle routinely expires offers before
// the customer PO lands, so Expired is NOT a terminal state — it is recoverable.
// Renewal reuses CreateOfferRevision so the new offer carries the line items,
// customer, terms, and lineage of the original.
func (a *App) RenewOffer(offerID string) (*Offer, error) {
	if err := a.requirePermission("offers:create"); err != nil {
		if editErr := a.requirePermission("offers:update"); editErr != nil {
			return nil, err
		}
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return nil, fmt.Errorf("offer id is required")
	}

	var source Offer
	if err := a.db.First(&source, "id = ?", offerID).Error; err != nil {
		return nil, fmt.Errorf("offer not found: %w", err)
	}

	// Renewal applies only to lapsed offers: those already marked Expired, or
	// still Quoted/RFQ but past their validity date. Won/Lost are terminal, and
	// an in-validity offer should be edited/revised rather than renewed.
	stage := strings.TrimSpace(source.Stage)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lapsed := (strings.EqualFold(stage, "Quoted") || strings.EqualFold(stage, "RFQ")) &&
		!source.ValidityDate.IsZero() && source.ValidityDate.Before(today)
	if !strings.EqualFold(stage, "Expired") && !lapsed {
		return nil, fmt.Errorf("only expired offers can be renewed (offer %s is %q)", source.OfferNumber, stage)
	}

	renewed, err := a.CreateOfferRevision(offerID)
	if err != nil {
		return nil, err
	}
	log.Printf("🔄 Renewed expired offer %s → %s (valid until %s)", source.OfferNumber, renewed.OfferNumber, renewed.ValidityDate.Format("2006-01-02"))
	return renewed, nil
}

// GetCostingsByOpportunity returns pipeline costing revisions (RFQID==0)
// scoped to a canonical opportunity UUID, newest revision first.
func (a *App) GetCostingsByOpportunity(opportunityRecordID string) ([]CostingSheetData, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	opportunityRecordID = strings.TrimSpace(opportunityRecordID)
	if opportunityRecordID == "" {
		return []CostingSheetData{}, nil
	}

	var costings []CostingSheetData
	err := a.db.Where("rfq_id = 0 AND opportunity_id = ?", opportunityRecordID).
		Order("revision_number DESC").
		Find(&costings).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get costings: %w", err)
	}

	log.Printf("✅ Retrieved %d costing revisions for Opportunity %s", len(costings), opportunityRecordID)
	return costings, nil
}

// ============================================================================
// OPPORTUNITY COMMERCIAL FIELDS + UNIFIED OFFER THREAD (Mission I — I-21)
// Ported from ph_holdings/user_feedback_hardening_service.go.
// ============================================================================

// UnifiedThreadEntry is a single comment/note in the merged offer conversation.
type UnifiedThreadEntry struct {
	ID          string    `json:"id"`
	SourceType  string    `json:"source_type"`
	SourceID    string    `json:"source_id"`
	WorkflowKey string    `json:"workflow_key"`
	Comment     string    `json:"comment"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateOpportunityCommercialFields applies a whitelisted set of commercial
// corrections to an opportunity. Only columns in the allow-list are written;
// everything else is silently ignored so the frontend cannot mass-assign
// protected columns. Every licensed employee may correct these fields;
// destructive actions remain admin-only.
func (a *App) UpdateOpportunityCommercialFields(opportunityID string, updates map[string]any) (*Opportunity, error) {
	if err := a.requirePermission("dashboard:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	opportunityID = strings.TrimSpace(opportunityID)
	if opportunityID == "" {
		return nil, fmt.Errorf("opportunity id is required")
	}

	allowed := map[string]bool{
		"folder_number": true, "folder_name": true, "title": true, "eh_ref": true,
		"customer_id": true, "customer_name": true, "salesperson": true, "division": true,
		"delivery_terms": true, "payment_terms": true, "revenue_bhd": true, "cost_bhd": true,
		"profit_bhd": true, "stage": true, "spoc_status": true, "wip_status": true,
		"comment": true, "owner_notes": true, "product_type": true, "product_details": true,
		"customer_grade": true, "source": true, "year": true, "expected_date": true,
		"offer_date": true, "won_reason": true, "lost_reason": true,
	}
	clean := make(map[string]any)
	for key, value := range updates {
		if !allowed[key] {
			continue
		}
		switch key {
		case "year":
			year := int(asFloat64(value))
			if year >= 2000 && year <= 2100 {
				clean[key] = year
			}
			continue
		case "expected_date", "offer_date":
			parsed, hasDate, err := parseOpportunityDateValue(value)
			if err != nil {
				return nil, err
			}
			if hasDate {
				clean[key] = parsed
			} else if key == "expected_date" {
				clean[key] = nil
			}
			continue
		case "stage":
			raw, _ := value.(string)
			canonicalStage, _ := canonicalizeOpportunityStage(raw)
			if !isCanonicalOpportunityStage(canonicalStage) {
				return nil, fmt.Errorf("invalid stage %q: must be one of %v", raw, canonicalOpportunityStages)
			}
			clean[key] = canonicalStage
			continue
		}
		if text, ok := value.(string); ok {
			clean[key] = strings.TrimSpace(text)
			continue
		}
		clean[key] = value
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("no supported opportunity updates supplied")
	}
	if title, ok := clean["title"].(string); ok && len(title) > 500 {
		clean["title"] = title[:500]
	}
	if folderNumber, ok := clean["folder_number"].(string); ok && len(folderNumber) > 50 {
		clean["folder_number"] = folderNumber[:50]
	}

	var updated Opportunity
	var changedFields []string
	if err := a.db.Transaction(func(tx *gorm.DB) error {
		var current Opportunity
		if err := tx.First(&current, "id = ?", opportunityID).Error; err != nil {
			return fmt.Errorf("opportunity not found: %w", err)
		}
		for key, next := range clean {
			if opportunityFieldChanged(current, key, next) {
				changedFields = append(changedFields, key)
			}
		}
		clean["version"] = gorm.Expr("version + ?", 1)
		clean["updated_at"] = time.Now()
		if err := tx.Model(&Opportunity{}).Where("id = ?", opportunityID).Updates(clean).Error; err != nil {
			return fmt.Errorf("failed to update opportunity: %w", err)
		}
		return tx.First(&updated, "id = ?", opportunityID).Error
	}); err != nil {
		return nil, err
	}

	if len(changedFields) > 0 {
		sort.Strings(changedFields)
		_ = a.db.Create(&OpportunityComment{
			OpportunityID: opportunityID,
			Comment:       "Updated fields: " + strings.Join(changedFields, ", "),
			CreatedBy:     firstNonEmpty(a.getCurrentUserDisplayName(), a.getCurrentUserID(), "System"),
		}).Error
	}
	return &updated, nil
}

func parseOpportunityDateValue(value any) (time.Time, bool, error) {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || strings.EqualFold(text, "<nil>") {
		return time.Time{}, false, nil
	}
	if typed, ok := value.(time.Time); ok {
		if typed.IsZero() {
			return time.Time{}, false, nil
		}
		return typed, true, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02", "02/01/2006", "2/1/2006", "01/02/2006"} {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			return parsed, true, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("invalid opportunity date %q", text)
}

func opportunityFieldChanged(current Opportunity, key string, next any) bool {
	nextText := strings.TrimSpace(fmt.Sprint(next))
	switch key {
	case "folder_number":
		return current.FolderNumber != nextText
	case "folder_name":
		return current.FolderName != nextText
	case "title":
		return current.Title != nextText
	case "eh_ref":
		return current.EHRef != nextText
	case "customer_id":
		return current.CustomerID != nextText
	case "customer_name":
		return current.CustomerName != nextText
	case "customer_grade":
		return current.CustomerGrade != nextText
	case "salesperson":
		return current.Salesperson != nextText
	case "division":
		return current.Division != nextText
	case "source":
		return current.Source != nextText
	case "delivery_terms":
		return current.DeliveryTerms != nextText
	case "payment_terms":
		return current.PaymentTerms != nextText
	case "product_details":
		return current.ProductDetails != nextText
	case "stage":
		return current.Stage != nextText
	case "spoc_status":
		return current.SPOCStatus != nextText
	case "wip_status":
		return current.WIPStatus != nextText
	case "comment":
		return current.Comment != nextText
	case "owner_notes":
		return current.OwnerNotes != nextText
	case "product_type":
		return current.ProductType != nextText
	case "won_reason":
		return current.WonReason != nextText
	case "lost_reason":
		return current.LostReason != nextText
	case "year":
		return current.Year != int(asFloat64(next))
	case "expected_date":
		if next == nil {
			return current.ExpectedDate != nil
		}
		parsed, hasDate, _ := parseOpportunityDateValue(next)
		if !hasDate {
			return current.ExpectedDate != nil
		}
		return current.ExpectedDate == nil || !current.ExpectedDate.Equal(parsed)
	case "offer_date":
		parsed, hasDate, _ := parseOpportunityDateValue(next)
		return hasDate && !current.OfferDate.Equal(parsed)
	case "revenue_bhd":
		return fmt.Sprintf("%.3f", current.RevenueBHD) != fmt.Sprintf("%.3f", asFloat64(next))
	case "cost_bhd":
		return fmt.Sprintf("%.3f", current.CostBHD) != fmt.Sprintf("%.3f", asFloat64(next))
	case "profit_bhd":
		return fmt.Sprintf("%.3f", current.ProfitBHD) != fmt.Sprintf("%.3f", asFloat64(next))
	default:
		return true
	}
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		parsed, _ := strconv.ParseFloat(fmt.Sprint(value), 64)
		return parsed
	}
}

// GetUnifiedOfferThread assembles a chronological, de-duplicated conversation
// spanning an offer's notes, its linked opportunities' comments and legacy
// notes, and any RFQ comments/notes reachable through shared workflow keys
// (offer number, customer reference, RFQ id, folder number, EH ref).
func (a *App) GetUnifiedOfferThread(offerID string) ([]UnifiedThreadEntry, error) {
	if err := a.requirePermission("offers:view"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return nil, fmt.Errorf("offer id is required")
	}

	var offer Offer
	if err := a.db.First(&offer, "id = ?", offerID).Error; err != nil {
		return nil, fmt.Errorf("offer not found: %w", err)
	}

	keys := map[string]bool{}
	addKey := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			keys[value] = true
		}
	}
	addKey(offer.OfferNumber)
	addKey(offer.CustomerReference)
	addKey(offer.RFQID)

	var linkedOpps []Opportunity
	query := a.db.Where("offer_id = ?", offer.ID)
	for key := range keys {
		query = query.Or("folder_number = ? OR eh_ref = ?", key, key)
		if !isPureNumericThreadKey(key) {
			query = query.Or("folder_name LIKE ? OR title LIKE ?", "%"+key+"%", "%"+key+"%")
		}
	}
	var candidateOpps []Opportunity
	_ = query.Find(&candidateOpps).Error
	for _, opp := range candidateOpps {
		if opportunityMatchesThreadKeys(opp, offer.ID, keys) {
			linkedOpps = append(linkedOpps, opp)
		}
	}
	for _, opp := range linkedOpps {
		addKey(opp.FolderNumber)
		addKey(opp.EHRef)
	}

	entries := make([]UnifiedThreadEntry, 0)

	var notes []OfferNote
	if err := a.db.Where("offer_id = ?", offer.ID).Order("note_date ASC, created_at ASC").Limit(200).Find(&notes).Error; err == nil {
		for _, note := range notes {
			entries = append(entries, UnifiedThreadEntry{
				ID:          note.ID,
				SourceType:  "offer",
				SourceID:    offer.ID,
				WorkflowKey: firstThreadKey(keys),
				Comment:     htmlUnescapeForThread(note.Content),
				CreatedBy:   firstNonEmpty(note.CreatedBy, "Legacy note"),
				CreatedAt:   firstNonZeroTime(note.NoteDate, note.CreatedAt),
			})
		}
	}

	for _, opp := range linkedOpps {
		var oppComments []OpportunityComment
		if err := a.db.Where("opportunity_id = ?", opp.ID).Order("created_at ASC").Find(&oppComments).Error; err == nil {
			for _, comment := range oppComments {
				entries = append(entries, UnifiedThreadEntry{
					ID:          fmt.Sprintf("opp-%d", comment.ID),
					SourceType:  "opportunity",
					SourceID:    opp.ID,
					WorkflowKey: firstNonEmpty(opp.FolderNumber, opp.EHRef, firstThreadKey(keys)),
					Comment:     comment.Comment,
					CreatedBy:   firstNonEmpty(comment.CreatedBy, "Legacy note"),
					CreatedAt:   comment.CreatedAt,
				})
			}
		}
		for _, legacy := range []string{opp.Comment, opp.OwnerNotes} {
			if strings.TrimSpace(legacy) == "" {
				continue
			}
			entries = append(entries, UnifiedThreadEntry{
				ID:          "opp-legacy-" + opp.ID + "-" + strconv.Itoa(len(entries)),
				SourceType:  "opportunity",
				SourceID:    opp.ID,
				WorkflowKey: firstNonEmpty(opp.FolderNumber, opp.EHRef, firstThreadKey(keys)),
				Comment:     strings.TrimSpace(legacy),
				CreatedBy:   "Legacy note",
				CreatedAt:   firstNonZeroTime(opp.UpdatedAt, opp.CreatedAt),
			})
		}
	}

	for key := range keys {
		if id, err := strconv.ParseUint(key, 10, 64); err == nil && id > 0 {
			appendRFQThreadEntries(a.db, &entries, uint(id), key)
		}
		var rfqs []RFQData
		if err := a.db.Where("rfq_number = ? OR rfq_ref = ?", key, key).Find(&rfqs).Error; err == nil {
			for _, rfq := range rfqs {
				appendRFQThreadEntries(a.db, &entries, rfq.ID, firstNonEmpty(rfq.RFQNumber, rfq.RFQRef, key))
				if strings.TrimSpace(rfq.Notes) != "" {
					entries = append(entries, UnifiedThreadEntry{
						ID:          fmt.Sprintf("rfq-legacy-%d", rfq.ID),
						SourceType:  "rfq",
						SourceID:    strconv.FormatUint(uint64(rfq.ID), 10),
						WorkflowKey: firstNonEmpty(rfq.RFQNumber, rfq.RFQRef, key),
						Comment:     strings.TrimSpace(rfq.Notes),
						CreatedBy:   "Legacy note",
						CreatedAt:   firstNonZeroTime(rfq.UpdatedAt, rfq.CreatedAt),
					})
				}
			}
		}
	}

	entries = dedupeThreadEntries(entries)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

func appendRFQThreadEntries(db *gorm.DB, entries *[]UnifiedThreadEntry, rfqID uint, workflowKey string) {
	var comments []RFQComment
	if err := db.Where("rfq_id = ?", rfqID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return
	}
	for _, comment := range comments {
		*entries = append(*entries, UnifiedThreadEntry{
			ID:          fmt.Sprintf("rfq-%d", comment.ID),
			SourceType:  "rfq",
			SourceID:    strconv.FormatUint(uint64(rfqID), 10),
			WorkflowKey: workflowKey,
			Comment:     comment.Comment,
			CreatedBy:   firstNonEmpty(comment.CreatedBy, "Legacy note"),
			CreatedAt:   comment.CreatedAt,
		})
	}
}

func opportunityMatchesThreadKeys(opp Opportunity, offerID string, keys map[string]bool) bool {
	if strings.TrimSpace(opp.OfferID) != "" && strings.TrimSpace(opp.OfferID) == offerID {
		return true
	}
	for key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(opp.FolderNumber), key) || strings.EqualFold(strings.TrimSpace(opp.EHRef), key) {
			return true
		}
		if !isPureNumericThreadKey(key) &&
			(threadKeyAppearsAsToken(opp.FolderName, key) || threadKeyAppearsAsToken(opp.Title, key)) {
			return true
		}
	}
	return false
}

func isPureNumericThreadKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	for _, r := range key {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func threadKeyAppearsAsToken(text, key string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	key = strings.ToLower(strings.TrimSpace(key))
	if text == "" || key == "" {
		return false
	}
	start := 0
	for {
		idx := strings.Index(text[start:], key)
		if idx < 0 {
			return false
		}
		absolute := start + idx
		beforeOK := absolute == 0 || !isThreadKeyTokenRune(rune(text[absolute-1]))
		afterIndex := absolute + len(key)
		afterOK := afterIndex >= len(text) || !isThreadKeyTokenRune(rune(text[afterIndex]))
		if beforeOK && afterOK {
			return true
		}
		start = absolute + 1
	}
}

func isThreadKeyTokenRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
}

func dedupeThreadEntries(entries []UnifiedThreadEntry) []UnifiedThreadEntry {
	seen := make(map[string]bool, len(entries))
	out := make([]UnifiedThreadEntry, 0, len(entries))
	for _, entry := range entries {
		key := strings.Join([]string{
			strings.ToLower(strings.TrimSpace(entry.SourceType)),
			strings.TrimSpace(entry.SourceID),
			strings.ToLower(strings.TrimSpace(entry.Comment)),
			entry.CreatedAt.Format(time.RFC3339Nano),
		}, "|")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, entry)
	}
	return out
}

func firstThreadKey(keys map[string]bool) string {
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	if len(ordered) == 0 {
		return ""
	}
	return ordered[0]
}

func htmlUnescapeForThread(value string) string {
	value = strings.ReplaceAll(value, "&lt;", "<")
	value = strings.ReplaceAll(value, "&gt;", ">")
	value = strings.ReplaceAll(value, "&amp;", "&")
	value = strings.ReplaceAll(value, "&#34;", "\"")
	value = strings.ReplaceAll(value, "&#39;", "'")
	return value
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Now()
}

// DeleteOfferNote deletes a specific note by ID, scoped to the given offer
func (a *App) DeleteOfferNote(noteID string) error {
	if ok, err := a.guardDeleteOrRequest("offers:edit", "offer_note", noteID, "Offer note"); !ok {
		return err
	}
	if err := a.requirePermission("offers:edit"); err != nil {
		return err
	}
	return crmpipeline.DeleteOfferNote(a.db, noteID)
}

// MarkOfferWon marks an offer as Won and creates an Order with line items
func (a *App) MarkOfferWon(offerID string, customerPO string) (*Order, error) {
	if err := a.requirePermission("offers:edit"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if offerID == "" {
		return nil, fmt.Errorf("offer ID is required")
	}
	customerPO = strings.TrimSpace(customerPO)
	if customerPO == "" {
		return nil, fmt.Errorf("customer PO number is required to mark an offer as won")
	}

	// Use transaction for atomicity: if order creation fails, stage reverts
	var order *Order
	err := a.db.Transaction(func(tx *gorm.DB) error {
		var offer Offer
		if err := tx.Preload("Items").First(&offer, "id = ?", offerID).Error; err != nil {
			return fmt.Errorf("offer not found: %v", err)
		}

		// P0-1: Prevent Lost offers from being re-won
		if offer.Stage == "Lost" {
			return fmt.Errorf("cannot mark lost offer as won - offer %s was previously lost", offer.OfferNumber)
		}

		// Only allow Won transition from Quoted stage
		if offer.Stage != "Quoted" && offer.Stage != "Won" {
			return fmt.Errorf("offer must be in 'Quoted' stage to be marked as won, current stage: %s", offer.Stage)
		}

		if offer.Stage == "Won" {
			// Idempotent: check if order already exists
			var existingOrder Order
			if err := tx.Where("customer_id = ? AND customer_po_number = ? AND status != ?", offer.CustomerID, customerPO, "Cancelled").First(&existingOrder).Error; err == nil {
				order = &existingOrder
				return nil // Already processed, return existing order
			}
			return fmt.Errorf("offer already marked as won")
		}

		// Atomic update: only update if stage is NOT already "Won" (prevents race)
		result := tx.Model(&Offer{}).Where("id = ? AND stage != ?", offerID, "Won").Update("stage", "Won")
		if result.Error != nil {
			return fmt.Errorf("failed to update offer stage: %v", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("offer was concurrently marked as won by another user")
		}

		// Update parent RFQ status AND stage to "Won" if linked (P0 Fix: string
		// comparison). Stage must be kept in sync with Status here — otherwise
		// the RFQ's stale non-terminal Stage keeps it counted as active
		// pipeline in GetDashboardStats (deflates WinRate, inflates
		// PipelineValueBHD).
		if offer.RFQID != "" {
			if err := tx.Model(&RFQData{}).Where("id = ?", offer.RFQID).Updates(map[string]any{"status": "Won", "stage": "Won"}).Error; err != nil {
				log.Printf("⚠️ Warning: Failed to update RFQ #%s status: %v", offer.RFQID, err)
			} else {
				log.Printf("✅ Updated RFQ #%s status+stage to: Won", offer.RFQID)
			}
		}

		// Update linked opportunity stage to Won with close date
		if err := tx.Model(&Opportunity{}).Where("offer_id = ?", offerID).Updates(map[string]any{
			"stage":       "Won",
			"closed_date": time.Now(),
		}).Error; err != nil {
			log.Printf("⚠️ Warning: Failed to update opportunity stage for offer %s: %v", offerID, err)
		}

		// Generate order number
		var orderCount int64
		tx.Model(&Order{}).Count(&orderCount)
		orderNumber := fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102"), orderCount+1)

		// Create Order from the offer - copy ALL costing sheet data
		newOrder := &Order{
			Base:             Base{ID: uuid.New().String()},
			OrderNumber:      orderNumber,
			CustomerPONumber: customerPO,
			CustomerID:       offer.CustomerID,
			CustomerName:     offer.CustomerName,
			OrderDate:        time.Now(),
			RequiredDate:     time.Now().AddDate(0, 0, 30),
			TotalValueBHD:    offer.TotalValueBHD,
			GrandTotalBHD:    offer.TotalValueBHD,
			Status:           "Confirmed",
			// Copy offer terms
			PaymentTerms:  offer.PaymentTerms,
			DeliveryTerms: offer.DeliveryTerms,
			// Traceability - link back to source offer
			OfferID:     offer.ID,
			OfferNumber: offer.OfferNumber,
			RFQID:       offer.RFQID, // P1 FIX #2: RFQID is already string type
			// Copy contact & RFQ details (for invoice generation)
			CustomerReference: offer.CustomerReference,
			AttentionPerson:   offer.AttentionPerson,
			AttentionCompany:  offer.AttentionCompany,
			AttentionPhone:    offer.AttentionPhone,
			AttentionAddress:  offer.AttentionAddress,
			DeliveryWeeks:     offer.DeliveryWeeks,
			CountryOfOrigin:   offer.CountryOfOrigin,
			IssuedBy:          offer.IssuedBy,
			ContactPhone:      offer.ContactPhone,
			DiscountPercent:   offer.DiscountPercent,
			Division:          normalizeDivisionName(offer.Division),
		}

		// Map offer items to order items - copy ALL costing data
		for _, offerItem := range offer.Items {
			// Use explicit Model field if set, otherwise fall back to ProductCode
			model := offerItem.Model
			if model == "" {
				model = offerItem.ProductCode
			}
			orderItem := OrderItem{
				Base:        Base{ID: uuid.New().String()},
				OrderID:     newOrder.ID,
				LineNumber:  offerItem.LineNumber,
				ProductID:   offerItem.ProductID,
				ProductCode: offerItem.ProductCode,
				Description: offerItem.Description,
				Quantity:    offerItem.Quantity,
				UnitPrice:   offerItem.UnitPrice,
				// Extended costing fields (the costing sheet data flows through)
				Equipment:           offerItem.Equipment,
				Model:               model,
				Specification:       offerItem.Specification,
				DetailedDescription: offerItem.DetailedDescription,
				Currency:            offerItem.Currency,
				FOB:                 offerItem.FOB,
				Freight:             offerItem.Freight,
				TotalCost:           offerItem.TotalCost,
				MarginPercent:       offerItem.MarginPercent,
				TotalPrice:          offerItem.TotalPrice,
			}
			newOrder.Items = append(newOrder.Items, orderItem)
		}

		// Save order with items (inside transaction - rolls back if fails)
		if err := tx.Create(newOrder).Error; err != nil {
			return fmt.Errorf("failed to create order: %v", err)
		}

		// P1 FIX #2: Validate RFQ-to-Order traceability
		if err := ValidateRFQTraceability(&offer, newOrder); err != nil {
			log.Printf("⚠️ Warning: RFQ traceability validation failed: %v", err)
		}

		log.Printf("✅ Offer %s WON → Created Order %s (%.3f BHD, %d items, RFQID=%s)", offer.OfferNumber, orderNumber, offer.TotalValueBHD, len(newOrder.Items), newOrder.RFQID)

		// Wave 8 P5-2 (user-ratified): the auto-created Draft PurchaseOrder was
		// removed. It hardcoded EUR with a 0 exchange rate and no supplier — a
		// junk row in the procurement ledger. Deployed PH deliberately defers PO
		// creation to the explicit CreatePOsFromOrder / CreatePOFromOrder flow,
		// where the supplier, currency, and costing are chosen deliberately.

		order = newOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

// MarkOfferLost marks an offer as Lost with a reason
func (a *App) MarkOfferLost(offerID string, reason string) error {
	if err := a.requirePermission("offers:edit"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// First, get the offer to retrieve RFQID
	var offer Offer
	if err := a.db.First(&offer, "id = ?", offerID).Error; err != nil {
		return fmt.Errorf("offer not found: %v", err)
	}

	// SECURITY: Won stage is terminal - block regression (audit requirement)
	if offer.Stage == "Won" {
		return fmt.Errorf("cannot change terminal offer stage 'Won' to 'Lost' (audit requirement)")
	}

	result := a.db.Model(&Offer{}).Where("id = ? AND stage != ?", offerID, "Won").Updates(map[string]any{
		"stage":       "Lost",
		"lost_reason": reason,
	})

	if result.Error != nil {
		return fmt.Errorf("failed to mark offer as lost: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("offer not found: %s", offerID)
	}

	log.Printf("✅ Offer %s marked as LOST (reason: %s)", offerID, reason)

	// Update parent RFQ status AND stage to "Lost" if linked (P0 Fix: string
	// comparison). See MarkOfferWon for why Stage must stay in sync.
	if offer.RFQID != "" {
		if err := a.db.Model(&RFQData{}).Where("id = ?", offer.RFQID).Updates(map[string]any{"status": "Lost", "stage": "Lost"}).Error; err != nil {
			log.Printf("⚠️ Warning: Failed to update RFQ #%s status: %v", offer.RFQID, err)
		} else {
			log.Printf("✅ Updated RFQ #%s status+stage to: Lost", offer.RFQID)
		}
	}

	// Also update linked Opportunity for intelligence pipeline
	now := time.Now()
	a.db.Model(&Opportunity{}).Where("offer_id = ?", offerID).Updates(map[string]any{
		"stage":       "Lost",
		"lost_reason": reason,
		"closed_date": &now,
	})

	return nil
}

func (a *App) UpdateOpportunityDetails(opportunityID, comment, ownerNotes string) (*Opportunity, error) {
	return a.UpdateOpportunityDetailsWithVersion(opportunityID, 0, comment, ownerNotes)
}

// ============================================================================
// SURVIVAL GARDEN - GPU-ACCELERATED CASH FLOW SIMULATION
// ============================================================================

// SimulateSurvivalGarden runs GPU-accelerated cash flow simulation
// Returns array of garden states (one per month) with 3-regime dynamics
