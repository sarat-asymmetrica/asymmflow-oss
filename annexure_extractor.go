package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type annexureSpecBlock struct {
	LineNumber          int
	Quantity            float64
	Equipment           string
	Model               string
	LongCode            string
	Specification       string
	DetailedDescription string
	SourcePath          string
}

type annexureBackfillSummary struct {
	DealsScanned  int
	OffersSeen    int
	OffersUpdated int
	ItemsUpdated  int
	SpecsFound    int
}

var (
	annexureItemLineRe  = regexp.MustCompile(`^\s*(\d{1,3})\s+(\d+(?:[.,]\d+)?)\s*(?:PC|PCS|NOS?|EA|SET|LOT|NO)?\s+(.+?)\s*$`)
	annexureModelRe     = regexp.MustCompile(`(?i)\bModel\s*(?:no\.?|number)?\s*[:\-]\s*([^\n\r]+)`)
	annexureOrderCodeRe = regexp.MustCompile(`(?i)\bOrder\s*Code\s*[:\-]\s*([A-Z0-9][A-Z0-9+/_.,\-]{4,})`)
	annexureParenCodeRe = regexp.MustCompile(`\(([A-Z0-9][A-Z0-9+/_.,\-]{4,})\)`)
	annexureCodeLineRe  = regexp.MustCompile(`^\s*([A-Z0-9]{1,4})\s+([A-Za-z].*[:;].*)$`)
)

func extractAnnexureSpecsFromDealFiles(files []DiscoveredFile) []annexureSpecBlock {
	candidates := annexureSourceCandidates(files)
	var specs []annexureSpecBlock
	seen := map[string]bool{}

	for _, file := range candidates {
		text, err := extractAnnexureSourceText(file)
		if err != nil {
			log.Printf("⚠️ Annexure extraction skipped %s: %v", file.FilePath, err)
			continue
		}
		for _, spec := range parseAnnexureSpecBlocks(text, file.FilePath) {
			key := fmt.Sprintf("%d|%s|%s|%s", spec.LineNumber, normalizeAnnexureMatchKey(spec.Model), normalizeAnnexureMatchKey(spec.LongCode), normalizeAnnexureMatchKey(spec.Equipment))
			if seen[key] {
				continue
			}
			seen[key] = true
			specs = append(specs, spec)
		}
	}

	sort.SliceStable(specs, func(i, j int) bool {
		if specs[i].LineNumber == specs[j].LineNumber {
			return specs[i].SourcePath < specs[j].SourcePath
		}
		return specs[i].LineNumber < specs[j].LineNumber
	})
	return specs
}

func annexureSourceCandidates(files []DiscoveredFile) []DiscoveredFile {
	var scored []struct {
		file  DiscoveredFile
		score int
	}
	for _, file := range files {
		ext := strings.ToLower(file.Extension)
		if ext != ".pdf" && ext != ".rtf" && ext != ".xml" && ext != ".zip" {
			continue
		}
		name := strings.ToLower(file.FileName)
		path := strings.ToLower(file.FilePath)
		score := 0
		if strings.Contains(name, "techno") || strings.Contains(name, "technical") {
			score += 80
		}
		if strings.Contains(name, "commercial") || strings.Contains(name, "offer") || strings.Contains(name, "quotation") {
			score += 45
		}
		if strings.Contains(name, "ehonline-shop") || strings.Contains(name, "product_document") {
			score += 70
		}
		if strings.Contains(name, "2ddrawing") || strings.Contains(name, "drawing") || strings.Contains(name, "datasheet") ||
			strings.Contains(name, "technical information") || strings.Contains(path, "/rfq/") {
			score -= 40
		}
		if ext == ".pdf" {
			score += 10
		}
		if score <= 0 {
			continue
		}
		scored = append(scored, struct {
			file  DiscoveredFile
			score int
		}{file: file, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].file.FilePath < scored[j].file.FilePath
		}
		return scored[i].score > scored[j].score
	})

	const maxCandidates = 4
	out := make([]DiscoveredFile, 0, len(scored))
	for i, candidate := range scored {
		if i >= maxCandidates {
			break
		}
		out = append(out, candidate.file)
	}
	return out
}

func extractAnnexureSourceText(file DiscoveredFile) (string, error) {
	switch strings.ToLower(file.Extension) {
	case ".pdf":
		return extractVectorPDF(file.FilePath)
	case ".rtf":
		raw, err := os.ReadFile(file.FilePath)
		if err != nil {
			return "", err
		}
		return rtfToPlainText(string(raw)), nil
	case ".xml":
		raw, err := os.ReadFile(file.FilePath)
		if err != nil {
			return "", err
		}
		return xmlToPlainText(raw), nil
	case ".zip":
		return extractAnnexureTextFromZip(file.FilePath)
	default:
		return "", fmt.Errorf("unsupported annexure source type: %s", file.Extension)
	}
}

func extractAnnexureTextFromZip(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var builder strings.Builder
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name))
		if ext != ".html" && ext != ".htm" && ext != ".xml" && ext != ".rtf" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			continue
		}
		contents, readErr := io.ReadAll(io.LimitReader(rc, 2*1024*1024))
		_ = rc.Close()
		if readErr != nil {
			continue
		}
		switch ext {
		case ".rtf":
			builder.WriteString(rtfToPlainText(string(contents)))
		case ".xml":
			builder.WriteString(xmlToPlainText(contents))
		default:
			builder.WriteString(htmlToPlainText(string(contents)))
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String()), nil
}

func parseAnnexureSpecBlocks(text string, sourcePath string) []annexureSpecBlock {
	text = normalizeAnnexureText(text)
	if text == "" {
		return nil
	}

	parseText := text
	if idx := strings.Index(strings.ToUpper(parseText), "ANNEXURE"); idx >= 0 {
		parseText = parseText[idx:]
	}

	var blocks []annexureSpecBlock
	var current *annexureSpecBlock
	var body []string
	flush := func() {
		if current == nil {
			return
		}
		populateAnnexureSpecBlock(current, body)
		if hasUsefulAnnexureSpec(*current) {
			blocks = append(blocks, *current)
		}
		current = nil
		body = nil
	}

	lines := strings.Split(parseText, "\n")
	for idx := 0; idx < len(lines); idx++ {
		line := strings.TrimSpace(lines[idx])
		if line == "" {
			continue
		}
		if isAnnexureNoiseLine(line) {
			continue
		}
		if vertical, ok, consumed := parseVerticalAnnexureItem(lines, idx, sourcePath); ok {
			flush()
			current = &vertical
			idx += consumed
			continue
		}
		if match := annexureItemLineRe.FindStringSubmatch(line); len(match) == 4 {
			lineNo, _ := strconv.Atoi(match[1])
			qty, _ := strconv.ParseFloat(strings.ReplaceAll(match[2], ",", "."), 64)
			if lineNo > 0 && qty >= 0 {
				flush()
				current = &annexureSpecBlock{
					LineNumber: lineNo,
					Quantity:   qty,
					Equipment:  cleanAnnexureLine(match[3]),
					SourcePath: sourcePath,
				}
				continue
			}
		}
		if current == nil {
			continue
		}
		body = append(body, line)
	}
	flush()

	return blocks
}

func parseVerticalAnnexureItem(lines []string, idx int, sourcePath string) (annexureSpecBlock, bool, int) {
	if idx+3 >= len(lines) {
		return annexureSpecBlock{}, false, 0
	}
	lineNoRaw := strings.TrimSpace(lines[idx])
	qtyRaw := strings.TrimSpace(lines[idx+1])
	unitRaw := strings.TrimSpace(lines[idx+2])
	equipmentRaw := strings.TrimSpace(lines[idx+3])
	if !regexp.MustCompile(`^\d{1,3}$`).MatchString(lineNoRaw) {
		return annexureSpecBlock{}, false, 0
	}
	if !regexp.MustCompile(`^\d+(?:[.,]\d+)?$`).MatchString(qtyRaw) {
		return annexureSpecBlock{}, false, 0
	}
	if !regexp.MustCompile(`(?i)^(PC|PCS|NO|NOS|EA|SET|LOT)$`).MatchString(unitRaw) {
		return annexureSpecBlock{}, false, 0
	}
	if equipmentRaw == "" || regexp.MustCompile(`^\d`).MatchString(equipmentRaw) || isAnnexureNoiseLine(equipmentRaw) {
		return annexureSpecBlock{}, false, 0
	}
	lineNo, _ := strconv.Atoi(lineNoRaw)
	qty, _ := strconv.ParseFloat(strings.ReplaceAll(qtyRaw, ",", "."), 64)
	return annexureSpecBlock{
		LineNumber: lineNo,
		Quantity:   qty,
		Equipment:  cleanAnnexureLine(equipmentRaw),
		SourcePath: sourcePath,
	}, true, 3
}

func parseSingleAnnexureBlock(text string, sourcePath string) annexureSpecBlock {
	lines := make([]string, 0)
	for _, rawLine := range strings.Split(normalizeAnnexureText(text), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || isAnnexureNoiseLine(line) {
			continue
		}
		lines = append(lines, line)
	}
	block := annexureSpecBlock{LineNumber: 1, Quantity: 1, SourcePath: sourcePath}
	if len(lines) > 0 {
		block.Equipment = cleanAnnexureLine(lines[0])
	}
	populateAnnexureSpecBlock(&block, lines)
	return block
}

func populateAnnexureSpecBlock(block *annexureSpecBlock, body []string) {
	bodyText := strings.Join(body, "\n")
	if match := annexureModelRe.FindStringSubmatch(bodyText); len(match) == 2 {
		block.Model = cleanAnnexureLine(match[1])
	}
	if match := annexureOrderCodeRe.FindStringSubmatch(bodyText); len(match) == 2 {
		block.LongCode = cleanAnnexureCode(match[1])
	}
	if block.LongCode == "" {
		for _, match := range annexureParenCodeRe.FindAllStringSubmatch(bodyText, -1) {
			if len(match) == 2 && looksLikeLongOrderCode(match[1]) {
				block.LongCode = cleanAnnexureCode(match[1])
				break
			}
		}
	}

	descriptionLines := make([]string, 0, len(body))
	specLines := make([]string, 0, 6)
	for _, line := range body {
		line = cleanAnnexureLine(line)
		if line == "" {
			continue
		}
		if isAnnexureDetailStopLine(line) {
			break
		}
		if strings.EqualFold(line, block.Equipment) {
			continue
		}
		if strings.Contains(strings.ToLower(line), "model no") {
			continue
		}
		if block.LongCode != "" && strings.Contains(line, block.LongCode) {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "hs-code") ||
			strings.HasPrefix(strings.ToLower(line), "country of origin") ||
			strings.HasPrefix(strings.ToLower(line), "country of dispatch") {
			continue
		}
		if regexp.MustCompile(`^\d{1,3}(?:[.,]\d{2,3})?$`).MatchString(line) || regexp.MustCompile(`^\d{1,3}(?:[.,]\d{3})+[.,]\d{2,3}$`).MatchString(line) {
			continue
		}
		descriptionLines = append(descriptionLines, line)
		if len(specLines) < 6 && !annexureCodeLineRe.MatchString(line) {
			specLines = append(specLines, line)
		}
	}
	block.Specification = strings.TrimSpace(strings.Join(specLines, "\n"))
	block.DetailedDescription = strings.TrimSpace(strings.Join(descriptionLines, "\n"))
}

func enrichOfferItemsWithAnnexureSpecs(items []OfferItem, specs []annexureSpecBlock) int {
	if len(items) == 0 || len(specs) == 0 {
		return 0
	}
	updated := 0
	used := map[int]bool{}
	for i := range items {
		specIndex := bestAnnexureSpecMatch(items[i], specs, used)
		if specIndex < 0 {
			continue
		}
		spec := specs[specIndex]
		changed := false
		if strings.TrimSpace(items[i].LongCode) == "" && strings.TrimSpace(spec.LongCode) != "" {
			items[i].LongCode = spec.LongCode
			changed = true
		}
		if shouldReplaceAnnexureText(items[i].Specification) && strings.TrimSpace(spec.Specification) != "" {
			items[i].Specification = truncateAnnexureString(spec.Specification, 1900)
			changed = true
		}
		if shouldReplaceAnnexureText(items[i].DetailedDescription) && strings.TrimSpace(spec.DetailedDescription) != "" {
			items[i].DetailedDescription = truncateAnnexureString(spec.DetailedDescription, 4800)
			changed = true
		}
		if strings.TrimSpace(items[i].Model) == "" && strings.TrimSpace(spec.Model) != "" {
			items[i].Model = spec.Model
			items[i].ProductCode = spec.Model
			changed = true
		}
		if strings.TrimSpace(items[i].Equipment) == "" && strings.TrimSpace(spec.Equipment) != "" {
			items[i].Equipment = truncateAnnexureString(spec.Equipment, 240)
			changed = true
		}
		if changed {
			updated++
			used[specIndex] = true
		}
	}
	return updated
}

func bestAnnexureSpecMatch(item OfferItem, specs []annexureSpecBlock, used map[int]bool) int {
	bestIndex := -1
	bestScore := 0
	for i, spec := range specs {
		if used[i] {
			continue
		}
		score := 0
		if item.LineNumber > 0 && spec.LineNumber == item.LineNumber {
			score += 80
		}
		if item.LineNumber > 0 && spec.LineNumber >= 10 && spec.LineNumber%10 == 0 && spec.LineNumber/10 == item.LineNumber {
			score += 65
		}
		itemModel := normalizeAnnexureMatchKey(firstNonEmptyString(item.Model, item.ProductCode))
		specModel := normalizeAnnexureMatchKey(spec.Model)
		if itemModel != "" && specModel != "" {
			if itemModel == specModel {
				score += 70
			} else if strings.Contains(specModel, itemModel) || strings.Contains(itemModel, specModel) {
				score += 35
			}
		}
		itemDesc := normalizeAnnexureMatchKey(firstNonEmptyString(item.Equipment, item.Description))
		specDesc := normalizeAnnexureMatchKey(spec.Equipment)
		if itemDesc != "" && specDesc != "" {
			if strings.Contains(specDesc, itemDesc) || strings.Contains(itemDesc, specDesc) {
				score += 30
			}
		}
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	if bestScore < 50 {
		return -1
	}
	return bestIndex
}

func isAnnexureDetailStopLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	stops := []string{
		"total price net",
		"total freight",
		"total tax",
		"total price gross",
		"terms",
		"payment terms",
		"delivery time",
		"delivery",
		"prices valid until",
	}
	for _, stop := range stops {
		if lower == stop || strings.HasPrefix(lower, stop+":") {
			return true
		}
	}
	return false
}

func shouldReplaceAnnexureText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	normalized := strings.ToLower(value)
	return strings.HasPrefix(normalized, "line item") || normalized == "-" || normalized == "technical specification"
}

func normalizeAnnexureText(text string) string {
	replacer := strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"\u00a0", " ",
		"\u2028", "\n",
		"\u2013", "-",
		"\u2014", "-",
		"\ufeff", "",
	)
	text = replacer.Replace(text)
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		clean := cleanAnnexureLine(line)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return strings.Join(out, "\n")
}

func cleanAnnexureLine(line string) string {
	line = strings.TrimSpace(html.UnescapeString(line))
	line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
	return strings.Trim(line, " \t|")
}

func cleanAnnexureCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.Trim(code, ".,;:()[]{} ")
	return code
}

func looksLikeLongOrderCode(value string) bool {
	value = cleanAnnexureCode(value)
	if len(value) < 6 {
		return false
	}
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(value)
	hasLetter := regexp.MustCompile(`[A-Z]`).MatchString(value)
	return hasDigit && hasLetter
}

func isAnnexureNoiseLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return true
	}
	noise := []string{
		"annexure", "order code", "description", "item", "qty", "thank you for your business",
		"should you have any enquiries", "terms and conditions", "best regards",
	}
	for _, token := range noise {
		if lower == token || strings.HasPrefix(lower, token+" ") {
			return true
		}
	}
	return false
}

func normalizeAnnexureMatchKey(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^A-Z0-9]+`).ReplaceAllString(value, "")
	return value
}

func hasUsefulAnnexureSpec(spec annexureSpecBlock) bool {
	return strings.TrimSpace(spec.LongCode) != "" ||
		strings.TrimSpace(spec.Specification) != "" ||
		strings.TrimSpace(spec.DetailedDescription) != ""
}

func rtfToPlainText(raw string) string {
	var out strings.Builder
	inControl := false
	braceDepth := 0
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		switch {
		case ch == '{':
			braceDepth++
			inControl = false
		case ch == '}':
			if braceDepth > 0 {
				braceDepth--
			}
			inControl = false
		case ch == '\\':
			if strings.HasPrefix(raw[i:], `\par`) || strings.HasPrefix(raw[i:], `\line`) {
				out.WriteByte('\n')
			}
			inControl = true
		case inControl:
			if ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
				inControl = false
			}
		default:
			out.WriteByte(ch)
		}
	}
	return normalizeAnnexureText(out.String())
}

func xmlToPlainText(raw []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	var out strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if chars, ok := token.(xml.CharData); ok {
			text := strings.TrimSpace(string(chars))
			if text != "" {
				out.WriteString(text)
				out.WriteByte('\n')
			}
		}
	}
	return normalizeAnnexureText(out.String())
}

func htmlToPlainText(raw string) string {
	raw = regexp.MustCompile(`(?is)<(br|p|tr|li|div|h[1-6])[^>]*>`).ReplaceAllString(raw, "\n")
	raw = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(raw, " ")
	return normalizeAnnexureText(raw)
}

func truncateAnnexureString(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return strings.TrimSpace(value[:maxLen])
}

func backfillOfferAnnexureDetailsFromFolder(db *gorm.DB, root string) (annexureBackfillSummary, error) {
	summary := annexureBackfillSummary{}
	if db == nil {
		return summary, fmt.Errorf("database is nil")
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return summary, fmt.Errorf("root folder is required")
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || !info.IsDir() {
			return nil
		}
		sections, err := extractDealSectionPaths(path)
		if err != nil {
			return nil
		}
		if sections["RFQ"] == "" && sections["OFFER"] == "" && sections["WORKING"] == "" && sections["EXECUTION"] == "" {
			return nil
		}

		deal := DiscoveredDeal{
			LocalID:    "manual-annexure-backfill",
			FolderPath: path,
			FolderName: filepath.Base(path),
			RootPath:   root,
		}
		if sections["OFFER"] != "" {
			deal.FinalPath = sections["OFFER"]
		} else {
			deal.FinalPath = sections["WORKING"]
		}
		for _, sectionName := range []string{"RFQ", "OFFER", "WORKING", "EXECUTION"} {
			sectionPath := sections[sectionName]
			if sectionPath == "" {
				continue
			}
			deal.Files = append(deal.Files, collectDealFiles(sectionPath)...)
		}
		summary.DealsScanned++

		specs := extractAnnexureSpecsFromDealFiles(deal.Files)
		summary.SpecsFound += len(specs)
		if len(specs) == 0 {
			return filepath.SkipDir
		}

		identity := deriveOneDriveImportIdentity(deal.FolderName)
		offerNumber := deriveImportedOfferNumber(identity, deal)
		if offerNumber == "" {
			return filepath.SkipDir
		}

		var offer Offer
		if err := db.Preload("Items").Where("offer_number = ?", offerNumber).First(&offer).Error; err != nil {
			return filepath.SkipDir
		}
		summary.OffersSeen++
		if len(offer.Items) == 0 {
			return filepath.SkipDir
		}

		items := append([]OfferItem(nil), offer.Items...)
		updated := enrichOfferItemsWithAnnexureSpecs(items, specs)
		if updated == 0 {
			return filepath.SkipDir
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, item := range items {
				if err := tx.Model(&OfferItem{}).Where("id = ?", item.ID).Updates(map[string]any{
					"long_code":            item.LongCode,
					"specification":        item.Specification,
					"detailed_description": item.DetailedDescription,
					"model":                item.Model,
					"product_code":         item.ProductCode,
					"equipment":            item.Equipment,
				}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&Opportunity{}).
				Where("offer_id = ?", offer.ID).
				Update("product_details", serializeOpportunityProductDetailsFromOfferItems(items)).Error; err != nil {
				return err
			}
			if err := backfillOrderItemsFromOfferItems(tx, offer.ID, items); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}

		summary.OffersUpdated++
		summary.ItemsUpdated += updated
		return filepath.SkipDir
	})
	if err != nil {
		return summary, err
	}
	return summary, nil
}

func backfillOrderItemsFromOfferItems(tx *gorm.DB, offerID string, offerItems []OfferItem) error {
	var orders []Order
	if err := tx.Where("offer_id = ?", offerID).Find(&orders).Error; err != nil {
		return err
	}
	for _, order := range orders {
		for _, item := range offerItems {
			if err := tx.Model(&OrderItem{}).
				Where("order_id = ? AND line_number = ?", order.ID, item.LineNumber).
				Updates(map[string]any{
					"specification":        item.Specification,
					"detailed_description": item.DetailedDescription,
					"model":                item.Model,
					"product_code":         item.ProductCode,
					"equipment":            item.Equipment,
				}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
