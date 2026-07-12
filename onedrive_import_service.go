// ============================================================================
// ONEDRIVE IMPORT SERVICE - Scan local OneDrive folders and import deal folders
//
// MISSION: Walk locally-synced OneDrive deal folders, detect either legacy
// FINAL subfolders or the standard RFQ/OFFER/EXECUTION layout, classify
// documents, fuzzy-match customers, and import into opportunities/offers/orders.
//
// Typical folder structure:
//   <OneDrive root>/
//     NORTHGRID LIT Q3 2025/
//       FINAL/
//         RFQ from NORTHGRID.pdf
//         Costing Sheet v2.xlsx
//         Quotation QT-2025-045.pdf
//     RI-37-26 NATIONALPETROLEUM UPSTREAM TIT/
//       RFQ/
//         enquiry.msg
//       OFFER/
//         Costing 37-26-R0.xlsx
//         Techno-Commercial Offer 37-26-R1.pdf
//       EXECUTION/
//         PO_BU2650000335_0.pdf
// ============================================================================

package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// DiscoveredDeal represents one deal folder found during the OneDrive scan.
type DiscoveredDeal struct {
	LocalID             string                `json:"local_id"`    // client-side UUID for UI tracking
	FolderPath          string                `json:"folder_path"` // full path to the parent folder (not FINAL)
	FolderName          string                `json:"folder_name"` // just the folder name e.g. "NORTHGRID LIT Q3 2025"
	FinalPath           string                `json:"final_path"`  // full path to the FINAL subfolder
	RootPath            string                `json:"root_path"`   // which root path this came from
	CustomerMatches     []CustomerMatchResult `json:"customer_matches"`
	Files               []DiscoveredFile      `json:"files"`
	InstrumentType      string                `json:"instrument_type"` // e.g. "Level (LIT)", "Pressure (PIT)"
	YearHint            string                `json:"year_hint"`       // e.g. "2025"
	Status              string                `json:"status"`          // "pending", "confirmed", "importing", "imported", "skipped", "error"
	ErrorMsg            string                `json:"error_msg,omitempty"`
	ConfirmedCustomerID string                `json:"confirmed_customer_id,omitempty"`
	ImportedOfferID     string                `json:"imported_offer_id,omitempty"`
}

// CustomerMatchResult is a single fuzzy-match candidate for a deal folder.
type CustomerMatchResult struct {
	CustomerID   string  `json:"customer_id"`
	BusinessName string  `json:"business_name"`
	ShortCode    string  `json:"short_code"`
	Score        float64 `json:"score"`        // 0.0-1.0
	MatchReason  string  `json:"match_reason"` // "acronym", "shortcode", "token_overlap", "prefix"
}

// DiscoveredFile is a single file found inside a FINAL folder.
type DiscoveredFile struct {
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	FileType  string    `json:"file_type"` // "costing_sheet", "rfq", "quotation", "order", "document", "email", "unknown"
	Extension string    `json:"extension"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
}

// OneDriveScanResult is the full result of a scan across one or more root paths.
type OneDriveScanResult struct {
	Deals        []DiscoveredDeal `json:"deals"`
	TotalFolders int              `json:"total_folders"`
	TotalFiles   int              `json:"total_files"`
	ScanPaths    []string         `json:"scan_paths"`
	ScannedAt    time.Time        `json:"scanned_at"`
	Errors       []string         `json:"errors"`
}

// OneDriveImportResult is the per-deal result after ImportOneDriveDeals runs.
type OneDriveImportResult struct {
	DealLocalID           string `json:"deal_local_id"`
	Success               bool   `json:"success"`
	OfferID               string `json:"offer_id,omitempty"`
	Message               string `json:"message"`
	CostingSheetsImported int    `json:"costing_sheets_imported"`
	PDFsQueued            int    `json:"pdfs_queued"`
}

type oneDriveFolderMeta struct {
	FolderNumber string
	OppNumber    int
	Year         int
	Title        string
}

type oneDriveImportIdentity struct {
	RawFolderNumber       string
	CanonicalFolderNumber string
	Prefix                string
	SequenceToken         string
	Year                  int
	Title                 string
}

type parsedOneDriveCosting struct {
	File       DiscoveredFile
	Parsed     *ExcelCostingData
	Revision   int
	OptionName string
}

func oneDriveOpportunitySource(year int) string {
	if year >= 2000 && year <= 2100 {
		return fmt.Sprintf("%d_onedrive", year)
	}
	return "onedrive_import"
}

func isOneDriveOpportunitySource(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	return source == "onedrive_import" || strings.HasSuffix(source, "_onedrive")
}

// ============================================================================
// STOP-TOKEN SET — tokens stripped from folder names before customer matching
// ============================================================================

var oneDriveStopTokens = map[string]bool{
	// Instrument tags
	"LIT": true, "PIT": true, "TIT": true, "FIT": true, "GIT": true, "AIT": true,
	"AT": true, "FT": true, "PT": true, "LT": true, "TT": true, "UT": true,
	// Product categories
	"VALVES": true, "VALVE": true, "SPARES": true, "SPARE": true, "PARTS": true,
	"SERVICE": true, "SERVICES": true, "MAINTENANCE": true,
	"SUPPLY": true, "SUPPLIES": true,
	"INSTRUMENTS": true, "INSTRUMENT": true,
	"TRANSMITTERS": true, "TRANSMITTER": true,
	"SENSORS": true, "SENSOR": true,
	"ANALYZERS": true, "ANALYZER": true,
	// Instrument types (spelled out)
	"LEVEL": true, "PRESSURE": true, "TEMPERATURE": true, "FLOW": true,
	"GAS": true, "ANALYTICAL": true,
	// Misc
	"GENERAL": true, "MISC": true, "MISCELLANEOUS": true, "INSTALLATION": true,
	"AND": true, "FOR": true, "THE": true, "OF": true, "IN": true,
	"NO": true, "REF": true,
	// Document types
	"RFQ": true, "OFFER": true, "QUOTE": true, "QT": true,
	// Years & quarters
	"2023": true, "2024": true, "2025": true, "2026": true,
	"Q1": true, "Q2": true, "Q3": true, "Q4": true,
	// Months
	"JAN": true, "FEB": true, "MAR": true, "APR": true, "MAY": true, "JUN": true,
	"JUL": true, "AUG": true, "SEP": true, "OCT": true, "NOV": true, "DEC": true,
	// Version markers
	"FINAL": true, "DRAFT": true, "REV": true, "V1": true, "V2": true, "V3": true, "REVISED": true,
}

// signedWordSkipForAcronym are words that shouldn't contribute a letter to a
// first-letters acronym when building the customer's initials.
var acronymSkipWords = map[string]bool{
	"AND": true, "OF": true, "THE": true, "&": true, "FOR": true,
	"W.L.L": true, "B.S.C": true, "WLL": true, "BSC": true,
	"W.L.L.": true, "B.S.C.": true, "CO": true, "LLC": true,
}

var oneDriveCustomerAliases = map[string][]string{
	"GULF SMELTING CO":                {"GSC"},
	"MERIDIAN INDUSTRIAL CONTRACTING": {"MERIDIAN", "MERID"},
	"ALW DEMO":                        {"ALW"},
	"AQUAPURE TECHNOLOGIES":           {"APT"},
	"EASTSIDE WASTEWATER SERVICES":    {"EWS"},
	"JETHRO DEMO TRADING":             {"JETHROW"},
	"KESTREL TECHNICAL SERVICES":      {"KTS"},
}

// ============================================================================
// PURE HELPER FUNCTIONS
// ============================================================================

// classifyDiscoveredFile infers the role of a file from its name and extension.
// This is a pure function with no DB access.
func classifyDiscoveredFile(fileName string) string {
	lower := strings.ToLower(fileName)
	ext := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(filepath.Base(lower), ext)

	switch ext {
	case ".xlsx", ".xls":
		if strings.Contains(lower, "costing") ||
			strings.Contains(lower, "masterfile") ||
			strings.Contains(base, "cost ") ||
			strings.Contains(base, "cost-") ||
			strings.Contains(base, "cost_") ||
			strings.HasPrefix(base, "cost") {
			return "costing_sheet"
		}
		return "document"

	case ".pdf":
		switch {
		case strings.Contains(lower, "invoice"):
			return "invoice"
		case strings.Contains(lower, "rfq") ||
			strings.Contains(lower, "enquiry") ||
			strings.Contains(lower, "inquiry"):
			return "rfq"
		case strings.Contains(lower, "quotation") ||
			strings.Contains(lower, "quote") ||
			strings.Contains(lower, "offer") ||
			strings.Contains(lower, "qt-") ||
			strings.Contains(lower, "qt_"):
			return "quotation"
		case strings.Contains(lower, "order") ||
			strings.Contains(lower, "po ") ||
			strings.Contains(lower, "po-") ||
			strings.Contains(lower, "po_") ||
			strings.Contains(lower, "purchase"):
			return "order"
		case regexp.MustCompile(`^ph\d{2,6}[-_/]\d{2,4}$`).MatchString(base):
			return "invoice"
		default:
			return "document"
		}

	case ".docx", ".doc":
		return "document"

	case ".msg":
		return "email"

	default:
		return "unknown"
	}
}

// tokenizeName uppercases and splits a name into individual tokens after
// replacing common separators with spaces.
func tokenizeName(name string) []string {
	upper := strings.ToUpper(name)
	for _, ch := range []string{"-", "_", "/", "(", ")", ".", ","} {
		upper = strings.ReplaceAll(upper, ch, " ")
	}
	raw := strings.Fields(upper)
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// extractInstrumentType returns a human-readable label for the first
// instrument-type keyword found in the folder name.
func extractInstrumentType(folderName string) string {
	upper := strings.ToUpper(folderName)
	typeMap := map[string]string{
		"LIT":         "Level (LIT)",
		"PIT":         "Pressure (PIT)",
		"TIT":         "Temperature (TIT)",
		"FIT":         "Flow (FIT)",
		"GIT":         "Gas (GIT)",
		"AIT":         "Analytical (AIT)",
		"VALVE":       "Valves",
		"VALVES":      "Valves",
		"SPARE":       "Spares",
		"SPARES":      "Spares",
		"PARTS":       "Spare Parts",
		"SERVICE":     "Service",
		"MAINTENANCE": "Maintenance",
	}
	// Prioritise exact instrument tags first (order matters for display quality)
	priority := []string{"LIT", "PIT", "TIT", "FIT", "GIT", "AIT"}
	for _, k := range priority {
		// Require word boundary so "SPLIT" doesn't match "LIT"
		re := regexp.MustCompile(`\b` + k + `\b`)
		if re.MatchString(upper) {
			return typeMap[k]
		}
	}
	for k, label := range typeMap {
		if strings.Contains(upper, k) {
			return label
		}
	}
	return ""
}

// extractYearHint returns the first 20xx year string found in the folder name.
func extractYearHint(folderName string) string {
	re := regexp.MustCompile(`20(2[3-9]|3[0-9])`)
	if m := re.FindString(folderName); m != "" {
		return m
	}
	return ""
}

// buildFirstLettersAcronym builds an acronym from the first letter of each
// significant word in the customer's business name.
// e.g. "North Grid Authority" → "NGA"
func buildFirstLettersAcronym(businessName string) string {
	tokens := tokenizeName(businessName)
	var letters []byte
	for _, t := range tokens {
		if acronymSkipWords[t] {
			continue
		}
		if len(t) > 0 {
			letters = append(letters, t[0])
		}
	}
	if len(letters) < 2 {
		return ""
	}
	return string(letters)
}

// maxFloat64 returns the larger of two float64 values.
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// sliceContains reports whether a string slice contains a given value.
func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// matchCustomerToFolder is the core fuzzy-match algorithm.
// It returns up to 3 CustomerMatchResult entries with score > 0.1, sorted
// descending by score.  It is a pure function with no DB access.
func matchCustomerToFolder(folderName string, customers []CustomerMaster) []CustomerMatchResult {
	if len(customers) == 0 {
		return nil
	}

	// --- Step A: build cleaned folder tokens (stop-words removed) ---
	allFolderTokens := tokenizeName(folderName)
	var folderTokens []string
	for _, t := range allFolderTokens {
		if !oneDriveStopTokens[t] {
			folderTokens = append(folderTokens, t)
		}
	}

	// Pre-compile the parenthetical acronym extractor once
	parenAcronymRe := regexp.MustCompile(`\(([A-Z]{2,6})\)`)

	var results []CustomerMatchResult

	for _, c := range customers {
		score := 0.0
		reason := ""
		customerNameUpper := strings.ToUpper(c.BusinessName)

		// Rule 1: ShortCode exact match (highest priority)
		if c.ShortCode != "" {
			sc := strings.ToUpper(c.ShortCode)
			if sliceContains(folderTokens, sc) {
				score = 1.0
				reason = "shortcode"
			}
		}

		// Rule 2: Acronym from parentheses in BusinessName e.g. "(NGA)" "(DPC)"
		if score < 1.0 {
			parenMatches := parenAcronymRe.FindStringSubmatch(customerNameUpper)
			if len(parenMatches) >= 2 {
				acronym := parenMatches[1]
				if sliceContains(folderTokens, acronym) {
					if score < 0.95 {
						score = 0.95
						reason = "acronym"
					}
				}
			}
		}

		// Rule 3: First-letters acronym of customer name
		if score < 0.88 {
			initials := buildFirstLettersAcronym(c.BusinessName)
			if len(initials) >= 2 && sliceContains(folderTokens, initials) {
				score = maxFloat64(score, 0.88)
				reason = "first_letters_acronym"
			}
		}

		// Rule 3b: Known business aliases used in deal folder naming
		if score < 0.92 {
			customerNameUpper = strings.ToUpper(c.BusinessName)
			for pattern, aliases := range oneDriveCustomerAliases {
				if !strings.Contains(customerNameUpper, pattern) {
					continue
				}
				for _, alias := range aliases {
					if sliceContains(folderTokens, alias) {
						score = maxFloat64(score, 0.92)
						reason = "known_alias"
					}
				}
			}
		}

		// Rule 4: Customer name token is in folder tokens (length >= 3)
		if score < 1.0 {
			customerTokens := tokenizeName(c.BusinessName)
			for _, ct := range customerTokens {
				if len(ct) < 3 {
					continue
				}
				if sliceContains(folderTokens, ct) {
					// Longer token match → higher confidence
					lengthBonus := math.Min(float64(len(ct))/10.0, 0.1)
					candidate := 0.75 + lengthBonus
					if candidate > score {
						score = candidate
						reason = "token_overlap"
					}
				}
			}
		}

		// Rule 5: Partial prefix match (first 4+ chars)
		if score < 0.75 {
			customerTokens := tokenizeName(c.BusinessName)
			for _, ft := range folderTokens {
				if len(ft) < 4 {
					continue
				}
				for _, ct := range customerTokens {
					if len(ct) < 4 {
						continue
					}
					if strings.HasPrefix(ct, ft) || strings.HasPrefix(ft, ct) {
						if score < 0.60 {
							score = 0.60
							reason = "prefix"
						}
					}
				}
			}
		}

		if score > 0.1 {
			results = append(results, CustomerMatchResult{
				CustomerID:   c.ID,
				BusinessName: c.BusinessName,
				ShortCode:    c.ShortCode,
				Score:        score,
				MatchReason:  reason,
			})
		}
	}

	// Sort descending by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top 3
	if len(results) > 3 {
		results = results[:3]
	}
	return results
}

// generateOneDriveOfferRef produces a unique offer reference for an imported deal.
func generateOneDriveOfferRef(folderName string) string {
	// Sanitise: keep only alphanumeric + space, then replace spaces with dashes
	var sb strings.Builder
	for _, r := range folderName {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	sanitised := strings.Trim(sb.String(), "-")
	// Collapse repeated dashes
	dashRe := regexp.MustCompile(`-{2,}`)
	sanitised = dashRe.ReplaceAllString(sanitised, "-")

	const maxLen = 20
	if len(sanitised) > maxLen {
		sanitised = sanitised[:maxLen]
	}
	ts := time.Now().Format("060102150405") // YYMMDDHHMMSS
	return fmt.Sprintf("OD-%s-%s", sanitised, ts)
}

func isOneDriveSectionDir(name string) bool {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "RFQ", "OFFER", "WORKING", "EXECUTION", "FINAL":
		return true
	default:
		return false
	}
}

func extractDealSectionPaths(folderPath string) (map[string]string, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	sections := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToUpper(strings.TrimSpace(entry.Name()))
		if !isOneDriveSectionDir(name) {
			continue
		}
		sections[name] = filepath.Join(folderPath, entry.Name())
	}
	return sections, nil
}

func inferYearFromPath(paths ...string) int {
	re := regexp.MustCompile(`\b(20\d{2})\b`)
	for _, path := range paths {
		if m := re.FindStringSubmatch(path); len(m) == 2 {
			return parseMetaYear(m[1])
		}
	}
	return 0
}

func hasStructuredOneDriveDealIdentity(folderName string) bool {
	meta := parseOneDriveFolderMeta(folderName)
	return meta.OppNumber > 0
}

func collectDealFiles(sectionPath string) []DiscoveredFile {
	var files []DiscoveredFile

	_ = filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
			return nil
		}
		files = append(files, DiscoveredFile{
			FileName:  name,
			FilePath:  path,
			FileType:  classifyDiscoveredFile(name),
			Extension: strings.ToLower(filepath.Ext(name)),
			SizeBytes: info.Size(),
			ModTime:   info.ModTime(),
		})
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i].FilePath < files[j].FilePath
	})
	return files
}

func parseOneDriveFolderMeta(folderName string) oneDriveFolderMeta {
	trimmed := strings.TrimSpace(folderName)
	meta := oneDriveFolderMeta{Title: trimmed}
	if trimmed == "" {
		return meta
	}

	patterns := []struct {
		re     *regexp.Regexp
		build  func([]string) string
		yearAt int
		oppAt  int
	}{
		{
			re: regexp.MustCompile(`(?i)\b([A-Z]{1,8})(?:[-\s]+)(\d{1,3}[A-Z]?)[-\s/]+(\d{2})(?:\b|[^A-Z0-9])`),
			build: func(m []string) string {
				return strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%s-%d-%s", m[1], mustAtoi(m[2]), m[3])))
			},
			yearAt: 3,
			oppAt:  2,
		},
		{
			re: regexp.MustCompile(`\b(20\d{2})[-_/](\d{1,3})(?:\b|[^A-Z0-9])`),
			build: func(m []string) string {
				return fmt.Sprintf("%s-%d", m[1], mustAtoi(m[2]))
			},
			yearAt: 1,
			oppAt:  2,
		},
	}

	for _, pattern := range patterns {
		if loc := pattern.re.FindStringSubmatchIndex(trimmed); loc != nil {
			matches := pattern.re.FindStringSubmatch(trimmed)
			meta.FolderNumber = pattern.build(matches)
			meta.Year = parseMetaYear(matches[pattern.yearAt])
			meta.OppNumber = mustAtoi(matches[pattern.oppAt])

			title := strings.TrimSpace(trimmed[loc[1]:])
			title = strings.TrimLeft(title, "-_/ ")
			if title != "" {
				meta.Title = title
			}
			return meta
		}
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return meta
	}

	// Some 2026 folders omit the trailing year and instead look like
	// "RI-07-NORTHGRID- RIVERSIDE-RI". Capture the sequence so callers can still infer a
	// canonical key from the parent root path.
	if m := regexp.MustCompile(`(?i)^([A-Z]{1,8})(?:\s*-\s*|\s+)(\d{1,3}[A-Z]?)(?:\s*-\s*([A-Z]{2,8}))?(.*)$`).FindStringSubmatch(trimmed); len(m) == 5 {
		prefix := strings.ToUpper(strings.TrimSpace(m[1]))
		seqToken := strings.ToUpper(strings.TrimSpace(m[2]))
		third := strings.ToUpper(strings.TrimSpace(m[3]))
		remainder := strings.TrimSpace(strings.TrimLeft(m[4], "-_/ "))
		if !regexp.MustCompile(`^\d{2}$`).MatchString(third) {
			meta.FolderNumber = fmt.Sprintf("%s-%s", prefix, seqToken)
			if third != "" {
				meta.FolderNumber = fmt.Sprintf("%s-%s-%s", prefix, seqToken, third)
			}
			meta.OppNumber = mustAtoi(regexp.MustCompile(`\d+`).FindString(seqToken))
			if remainder != "" && third != "" {
				meta.Title = strings.TrimSpace(third + " " + remainder)
			} else if remainder != "" {
				meta.Title = remainder
			}
			return meta
		}
	}

	// D1 (PH 10f96a7): a real folder NUMBER always contains a digit. A
	// digit-less first token is a customer name, not a folder number —
	// accepting it as the loose fallback collapsed every OneDrive opportunity
	// for that customer onto one canonical key. The paired helpers reject a
	// digit-less token (returning an empty folder number) and split a real
	// numeric loose token (e.g. "150-AIRMENCH") into folder "150" + title
	// "AIRMENCH", exactly as PH's onedrive_import_service.go does.
	var looseTitle string
	meta.FolderNumber, looseTitle = splitLooseOneDriveFolderNumberToken(fields[0])
	var titleParts []string
	if meta.FolderNumber == "" {
		// Digit-less first token: it is a customer word, not a folder number —
		// keep the whole original token as the title so the folder is still
		// identifiable (matches the pre-D2 behavior the digit-guard preserves).
		titleParts = append(titleParts, fields[0])
	} else if looseTitle != "" {
		// A numeric loose token like "150-AIRMENCH" split into folder "150" +
		// title "AIRMENCH": carry the split-off remainder.
		titleParts = append(titleParts, looseTitle)
	}
	if len(fields) > 1 {
		titleParts = append(titleParts, fields[1:]...)
	}
	if len(titleParts) > 0 {
		meta.Title = strings.TrimSpace(strings.Join(titleParts, " "))
	}
	if m := regexp.MustCompile(`^\d{1,3}`).FindString(meta.FolderNumber); m != "" {
		meta.OppNumber = mustAtoi(m)
	}

	if m := regexp.MustCompile(`\b(20\d{2})[-_/](\d{1,3})\b`).FindStringSubmatch(meta.FolderNumber); len(m) == 3 {
		meta.Year = parseMetaYear(m[1])
		meta.OppNumber = mustAtoi(m[2])
		return meta
	}

	if m := regexp.MustCompile(`(?i)^([A-Z]{1,8})[-_]?(\d{1,3})[-_/](\d{2})$`).FindStringSubmatch(strings.ToUpper(strings.ReplaceAll(meta.FolderNumber, " ", "-"))); len(m) == 4 {
		meta.FolderNumber = fmt.Sprintf("%s-%d-%s", m[1], mustAtoi(m[2]), m[3])
		meta.Year = parseMetaYear(m[3])
		meta.OppNumber = mustAtoi(m[2])
	}

	return meta
}

func deriveOneDriveImportIdentity(folderName string) oneDriveImportIdentity {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(folderName)), " ")
	identity := oneDriveImportIdentity{Title: trimmed}

	re := regexp.MustCompile(`(?i)^([A-Z]{1,8})(?:\s*-\s*|\s+)(\d{1,3}[A-Z]?)\s*-\s*(\d{2})(.*)$`)
	if m := re.FindStringSubmatch(trimmed); len(m) == 5 {
		prefix := strings.ToUpper(strings.TrimSpace(m[1]))
		seqToken := strings.ToUpper(strings.TrimSpace(m[2]))
		yy := strings.TrimSpace(m[3])
		title := strings.TrimSpace(strings.TrimLeft(m[4], "-_/ "))
		if title == "" {
			title = trimmed
		}
		year := parseMetaYear(yy)
		identity.RawFolderNumber = fmt.Sprintf("%s-%s-%s", prefix, seqToken, yy)
		identity.Prefix = prefix
		identity.SequenceToken = seqToken
		identity.Year = year
		identity.Title = title
		identity.CanonicalFolderNumber = deriveCanonicalOneDriveFolderNumber(prefix, seqToken, year)
		return identity
	}

	meta := parseOneDriveFolderMeta(folderName)
	identity.RawFolderNumber = strings.TrimSpace(meta.FolderNumber)
	identity.Year = meta.Year
	if strings.TrimSpace(meta.Title) != "" {
		identity.Title = strings.TrimSpace(meta.Title)
	}
	if m := regexp.MustCompile(`(?i)^([A-Z]{1,8})-(\d{1,3}[A-Z]?)-(\d{2})$`).FindStringSubmatch(strings.ToUpper(strings.ReplaceAll(identity.RawFolderNumber, " ", "-"))); len(m) == 4 {
		identity.Prefix = strings.ToUpper(strings.TrimSpace(m[1]))
		identity.SequenceToken = strings.ToUpper(strings.TrimSpace(m[2]))
		if identity.Year == 0 {
			identity.Year = parseMetaYear(m[3])
		}
		identity.CanonicalFolderNumber = deriveCanonicalOneDriveFolderNumber(identity.Prefix, identity.SequenceToken, identity.Year)
	} else if m := regexp.MustCompile(`(?i)^([A-Z]{1,8})-(\d{1,3}[A-Z]?)(?:-([A-Z]{2,8}))?$`).FindStringSubmatch(strings.ToUpper(strings.ReplaceAll(identity.RawFolderNumber, " ", "-"))); len(m) >= 3 {
		identity.Prefix = strings.ToUpper(strings.TrimSpace(m[1]))
		identity.SequenceToken = strings.ToUpper(strings.TrimSpace(m[2]))
	}
	return identity
}

func deriveCanonicalOneDriveFolderNumber(prefix string, sequenceToken string, year int) string {
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	sequenceToken = strings.ToUpper(strings.TrimSpace(sequenceToken))
	if year == 0 || sequenceToken == "" {
		return ""
	}

	numberPart := regexp.MustCompile(`\d+`).FindString(sequenceToken)
	if numberPart == "" {
		return ""
	}
	suffix := strings.TrimPrefix(sequenceToken, numberPart)
	seq := mustAtoi(numberPart)

	switch prefix {
	case "EH":
		return fmt.Sprintf("%d-%d%s", year, seq, suffix)
	case "OTH":
		return fmt.Sprintf("%d-%d%s", year, 300+seq, suffix)
	case "SA":
		return fmt.Sprintf("%d-%d%s", year, 250+seq, suffix)
	default:
		return fmt.Sprintf("%d-%d%s", year, seq, suffix)
	}
}

func extractPathRevisionNumber(path string) int {
	re := regexp.MustCompile(`(?i)(?:^|[^A-Z0-9])REV(?:ISION)?[-_\s]?(\d{1,2})(?:[^A-Z0-9]|$)|(?:^|[^A-Z0-9])R[-_\s]?(\d{1,2})(?:[^A-Z0-9]|$)`)
	matches := re.FindStringSubmatch(strings.ToUpper(path))
	if len(matches) < 2 {
		return 0
	}
	for _, match := range matches[1:] {
		if strings.TrimSpace(match) == "" {
			continue
		}
		return mustAtoi(match)
	}
	return 0
}

func extractPathOptionName(path string) string {
	re := regexp.MustCompile(`(?i)OPTION[-_\s]?(\d+)`)
	if m := re.FindStringSubmatch(path); len(m) == 2 {
		return fmt.Sprintf("option-%d", mustAtoi(m[1]))
	}
	return ""
}

func scoreOneDriveCostingCandidate(candidate parsedOneDriveCosting) int {
	score := candidate.Revision * 1000
	switch strings.ToLower(strings.TrimSpace(candidate.OptionName)) {
	case "option-1":
		score += 40
	case "option-2":
		score += 30
	case "":
		score += 20
	default:
		score += 10
	}
	score += len(candidate.Parsed.LineItems)
	if candidate.Parsed.Totals.GrandTotal > 0 {
		score += 5
	}
	return score
}

func selectPrimaryOneDriveCosting(costings []parsedOneDriveCosting) *parsedOneDriveCosting {
	if len(costings) == 0 {
		return nil
	}
	best := costings[0]
	bestScore := scoreOneDriveCostingCandidate(best)
	for _, candidate := range costings[1:] {
		score := scoreOneDriveCostingCandidate(candidate)
		if score > bestScore || (score == bestScore && strings.ToUpper(candidate.File.FilePath) < strings.ToUpper(best.File.FilePath)) {
			best = candidate
			bestScore = score
		}
	}
	return &best
}

func rankExistingOpportunityForImport(opp Opportunity, canonicalKey string, rawKey string, dealFolderName string) int {
	score := 0
	switch {
	case canonicalKey != "" && strings.EqualFold(strings.TrimSpace(opp.FolderNumber), canonicalKey):
		score += 1000
	case rawKey != "" && strings.EqualFold(strings.TrimSpace(opp.FolderNumber), rawKey):
		score += 700
	case strings.EqualFold(strings.TrimSpace(opp.FolderName), strings.TrimSpace(dealFolderName)):
		score += 500
	}
	score += opportunitySourcePriority(opp.Source) * 100
	score += opportunityRichnessScore(opp) * 10
	if strings.TrimSpace(opp.OfferID) != "" {
		score += 25
	}
	if opp.RevenueBHD > 0 {
		score += 5
	}
	return score
}

func (a *App) findOneDriveOpportunityCandidates(identity oneDriveImportIdentity, dealFolderName string) ([]Opportunity, error) {
	seen := map[string]bool{}
	var candidates []Opportunity

	tryLookup := func(field string, value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		var rows []Opportunity
		if err := a.db.Where(field+" = ?", value).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
			candidates = append(candidates, row)
		}
		return nil
	}

	if err := tryLookup("folder_number", identity.CanonicalFolderNumber); err != nil {
		return nil, err
	}
	if err := tryLookup("folder_number", identity.RawFolderNumber); err != nil {
		return nil, err
	}
	if err := tryLookup("folder_name", dealFolderName); err != nil {
		return nil, err
	}

	sort.Slice(candidates, func(i, j int) bool {
		return rankExistingOpportunityForImport(candidates[i], identity.CanonicalFolderNumber, identity.RawFolderNumber, dealFolderName) >
			rankExistingOpportunityForImport(candidates[j], identity.CanonicalFolderNumber, identity.RawFolderNumber, dealFolderName)
	})
	return candidates, nil
}

func softDeleteDuplicateOpportunity(tx *gorm.DB, id string) error {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	return tx.Table("opportunities").
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now()).
		Error
}

func parseMetaYear(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	year, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	if year < 100 {
		return 2000 + year
	}
	return year
}

func mustAtoi(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return n
}

func detectOneDriveStage(files []DiscoveredFile) string {
	hasCosting := false
	hasQuotation := false
	hasExecution := false

	for _, f := range files {
		switch f.FileType {
		case "costing_sheet":
			hasCosting = true
		case "quotation":
			hasQuotation = true
		case "order", "invoice":
			hasExecution = true
		default:
			if strings.Contains(strings.ToUpper(f.FilePath), string(filepath.Separator)+"EXECUTION"+string(filepath.Separator)) {
				hasExecution = true
			}
		}
	}

	switch {
	case hasExecution:
		return "Won"
	case hasCosting || hasQuotation:
		return "Quoted"
	default:
		return "Qualified"
	}
}

func parseTimeWithLayouts(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{
		"02/01/2006", "2006-01-02", "01/02/2006", "02-Jan-06", "02-Jan-2006", "2-Jan-2006",
		"2/1/2006", "02.01.2006", "2 Jan 2006", "02 Jan 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func maxModTime(files []DiscoveredFile, fileType string, mustContainDir string) time.Time {
	var latest time.Time
	dirNeedle := strings.ToUpper(mustContainDir)
	for _, f := range files {
		if fileType != "" && f.FileType != fileType {
			continue
		}
		if dirNeedle != "" && !strings.Contains(strings.ToUpper(f.FilePath), dirNeedle) {
			continue
		}
		if f.ModTime.After(latest) {
			latest = f.ModTime
		}
	}
	return latest
}

func deriveOfferNumber(deal DiscoveredDeal) string {
	meta := parseOneDriveFolderMeta(deal.FolderName)
	if meta.FolderNumber != "" {
		return meta.FolderNumber
	}
	return generateOneDriveOfferRef(deal.FolderName)
}

func normalizeImportedOfferNumber(raw string) string {
	raw = strings.ToUpper(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, " ", "-")
	raw = regexp.MustCompile(`-+`).ReplaceAllString(raw, "-")
	return strings.Trim(raw, "-")
}

func deriveImportedOfferNumber(identity oneDriveImportIdentity, deal DiscoveredDeal) string {
	if normalized := normalizeImportedOfferNumber(identity.RawFolderNumber); normalized != "" {
		return normalized
	}
	if normalized := normalizeImportedOfferNumber(identity.CanonicalFolderNumber); normalized != "" {
		return normalized
	}
	if normalized := normalizeImportedOfferNumber(deriveOfferNumber(deal)); normalized != "" {
		return normalized
	}
	return generateOneDriveOfferRef(deal.FolderName)
}

func importedOfferMatchesDeal(offer Offer, desiredOfferNumber string, customerID string) bool {
	if strings.TrimSpace(offer.ID) == "" {
		return false
	}
	if desiredOfferNumber != "" && normalizeImportedOfferNumber(offer.OfferNumber) != normalizeImportedOfferNumber(desiredOfferNumber) {
		return false
	}
	if strings.TrimSpace(customerID) != "" && strings.TrimSpace(offer.CustomerID) != "" && strings.TrimSpace(offer.CustomerID) != strings.TrimSpace(customerID) {
		return false
	}
	return true
}

func (a *App) softDeleteOfferIfUnlinked(offerID string) error {
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return nil
	}

	var refCount int64
	if err := a.db.Model(&Opportunity{}).
		Where("offer_id = ? AND deleted_at IS NULL", offerID).
		Count(&refCount).Error; err != nil {
		return err
	}
	if refCount > 0 {
		return nil
	}

	now := time.Now()
	if err := a.db.Model(&Offer{}).
		Where("id = ? AND deleted_at IS NULL", offerID).
		Update("deleted_at", now).Error; err != nil {
		return err
	}
	return a.db.Model(&OfferItem{}).
		Where("offer_id = ? AND deleted_at IS NULL", offerID).
		Update("deleted_at", now).Error
}

func (a *App) clearPlaceholderCommercialChain(opportunityID string, offerID string) error {
	offerID = strings.TrimSpace(offerID)
	if offerID == "" {
		return nil
	}

	var offer Offer
	if err := a.db.Unscoped().Where("id = ?", offerID).First(&offer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if opportunityID != "" {
				return a.db.Model(&Opportunity{}).
					Where("id = ?", opportunityID).
					Update("offer_id", "").Error
			}
			return nil
		}
		return err
	}

	var offerItemCount int64
	if err := a.db.Model(&OfferItem{}).
		Where("offer_id = ? AND deleted_at IS NULL", offerID).
		Count(&offerItemCount).Error; err != nil {
		return err
	}
	if offer.TotalValueBHD > 0 || offerItemCount > 0 {
		return nil
	}

	var order Order
	err := a.db.Unscoped().Where("offer_id = ?", offerID).First(&order).Error
	hasOrder := err == nil
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if hasOrder {
		var orderItemCount int64
		if err := a.db.Model(&OrderItem{}).
			Where("order_id = ? AND deleted_at IS NULL", order.ID).
			Count(&orderItemCount).Error; err != nil {
			return err
		}
		if order.TotalValueBHD > 0 || order.GrandTotalBHD > 0 || orderItemCount > 0 {
			return nil
		}
		var invoiceCount int64
		if err := a.db.Model(&Invoice{}).
			Where("order_id = ? AND deleted_at IS NULL", order.ID).
			Count(&invoiceCount).Error; err != nil {
			return err
		}
		if invoiceCount > 0 {
			return nil
		}
	}

	now := time.Now()
	return a.db.Transaction(func(tx *gorm.DB) error {
		if opportunityID != "" {
			if err := tx.Model(&Opportunity{}).
				Where("id = ?", opportunityID).
				Update("offer_id", "").Error; err != nil {
				return err
			}
		}
		if hasOrder {
			if err := tx.Model(&OrderItem{}).
				Where("order_id = ? AND deleted_at IS NULL", order.ID).
				Update("deleted_at", now).Error; err != nil {
				return err
			}
			if err := tx.Model(&Order{}).
				Where("id = ? AND deleted_at IS NULL", order.ID).
				Update("deleted_at", now).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&OfferItem{}).
			Where("offer_id = ? AND deleted_at IS NULL", offerID).
			Update("deleted_at", now).Error; err != nil {
			return err
		}
		return tx.Model(&Offer{}).
			Where("id = ? AND deleted_at IS NULL", offerID).
			Update("deleted_at", now).Error
	})
}

func extractRevisionNumber(files []DiscoveredFile) int {
	maxRev := 1
	re := regexp.MustCompile(`(?i)(?:^|[^A-Z0-9])R(?:EV)?[-_\s]?(\d{1,2})(?:[^A-Z0-9]|$)`)
	for _, f := range files {
		for _, text := range []string{f.FileName, f.FilePath} {
			if m := re.FindStringSubmatch(text); len(m) == 2 {
				if n, err := strconv.Atoi(m[1]); err == nil && n > maxRev {
					maxRev = n
				}
			}
		}
	}
	return maxRev
}

func extractPONumber(files []DiscoveredFile) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bPO[-_ ]?([A-Z0-9][A-Z0-9/_-]{4,})`),
		regexp.MustCompile(`(?i)\bPurchase\s+Order[-_ ]?([A-Z0-9][A-Z0-9/_-]{4,})`),
		regexp.MustCompile(`(?i)\b([A-Z]{2,6}\d{6,})\b`),
	}
	for _, f := range files {
		if f.FileType != "order" && !strings.Contains(strings.ToUpper(f.FilePath), string(filepath.Separator)+"EXECUTION"+string(filepath.Separator)) {
			continue
		}
		base := strings.TrimSuffix(filepath.Base(f.FileName), filepath.Ext(f.FileName))
		for _, re := range patterns {
			if m := re.FindStringSubmatch(base); len(m) == 2 {
				return strings.Trim(m[1], "_- ")
			}
		}
	}
	return ""
}

func extractInvoiceNumber(files []DiscoveredFile) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)invoice(?:\s+no\.?)?[-_\s]*([A-Z0-9][A-Z0-9/_-]{2,})`),
		regexp.MustCompile(`(?i)\b(PH\d{2,6}[-_/]\d{2,4})\b`),
	}
	for _, f := range files {
		if f.FileType != "invoice" {
			continue
		}
		base := strings.TrimSuffix(filepath.Base(f.FileName), filepath.Ext(f.FileName))
		for _, re := range patterns {
			if m := re.FindStringSubmatch(base); len(m) == 2 {
				return strings.Trim(m[1], "_- ")
			}
		}
	}
	return ""
}

func buildOrderItemsFromOfferItems(offerItems []OfferItem, orderID string) []OrderItem {
	items := make([]OrderItem, 0, len(offerItems))
	for _, item := range offerItems {
		items = append(items, OrderItem{
			Base:                Base{ID: uuid.New().String()},
			OrderID:             orderID,
			LineNumber:          item.LineNumber,
			ProductID:           item.ProductID,
			ProductCode:         item.ProductCode,
			Description:         item.Description,
			Quantity:            item.Quantity,
			UnitPrice:           item.UnitPrice,
			Equipment:           item.Equipment,
			Model:               item.Model,
			Specification:       item.Specification,
			DetailedDescription: item.DetailedDescription,
			Currency:            item.Currency,
			FOB:                 item.FOB,
			Freight:             item.Freight,
			TotalCost:           item.TotalCost,
			MarginPercent:       item.MarginPercent,
			TotalPrice:          item.TotalPrice,
		})
	}
	return items
}

// ============================================================================
// APP METHODS — Wails-bound
// ============================================================================

// ValidateOneDrivePath checks that a path exists, is a directory, and gives
// an estimate of how many deal folders it contains.
// No RBAC guard is required — this is a safe read-only stat call.
func (a *App) ValidateOneDrivePath(path string) (map[string]any, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if path == "" {
		return map[string]any{"valid": false, "error": "path is empty"}, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return map[string]any{"valid": false, "error": err.Error()}, nil
	}
	if !info.IsDir() {
		return map[string]any{"valid": false, "error": "path is not a directory"}, nil
	}

	estimatedDeals := 0
	rootDepth := strings.Count(filepath.ToSlash(path), "/")

	_ = filepath.Walk(path, func(p string, fi os.FileInfo, werr error) error {
		if werr != nil {
			return nil // skip unreadable dirs
		}
		if !fi.IsDir() {
			return nil
		}
		currentDepth := strings.Count(filepath.ToSlash(p), "/") - rootDepth
		if currentDepth > 3 {
			return filepath.SkipDir
		}
		if strings.EqualFold(filepath.Base(p), "final") {
			estimatedDeals++
			return filepath.SkipDir
		}
		sections, err := extractDealSectionPaths(p)
		if err == nil && (sections["RFQ"] != "" || sections["OFFER"] != "" || sections["WORKING"] != "" || sections["EXECUTION"] != "") {
			estimatedDeals++
			return filepath.SkipDir
		}
		return nil
	})

	return map[string]any{
		"valid":           true,
		"estimated_deals": estimatedDeals,
		"path":            path,
	}, nil
}

// ScanOneDrivePaths walks each supplied root path, discovers FINAL subfolders,
// classifies files, fuzzy-matches customers, and returns a structured result.
func (a *App) ScanOneDrivePaths(paths []string) (OneDriveScanResult, error) {
	if err := a.requirePermission("settings:view"); err != nil {
		return OneDriveScanResult{}, err
	}
	if a.db == nil {
		return OneDriveScanResult{}, fmt.Errorf("database not initialized")
	}

	const maxDeals = 500

	result := OneDriveScanResult{
		ScanPaths: paths,
		ScannedAt: time.Now(),
	}

	// Load all active customers once
	var customers []CustomerMaster
	if err := a.db.Where("deleted_at IS NULL").Find(&customers).Error; err != nil {
		return OneDriveScanResult{}, fmt.Errorf("failed to load customers: %w", err)
	}

	var scanErrors []string

	for _, rootPath := range paths {
		if rootPath == "" {
			continue
		}

		err := filepath.Walk(rootPath, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				scanErrors = append(scanErrors, fmt.Sprintf("walk error at %s: %v", path, walkErr))
				return nil // continue walking
			}
			if !info.IsDir() {
				return nil
			}

			var (
				finalPath  string
				folderPath string
				folderName string
				files      []DiscoveredFile
				isDeal     bool
			)

			switch {
			case strings.EqualFold(filepath.Base(path), "final"):
				finalPath = path
				folderPath = filepath.Dir(finalPath)
				folderName = filepath.Base(folderPath)
				files = collectDealFiles(finalPath)
				isDeal = true
			default:
				sections, err := extractDealSectionPaths(path)
				if err != nil {
					return nil
				}
				if sections["RFQ"] == "" && sections["OFFER"] == "" && sections["WORKING"] == "" && sections["EXECUTION"] == "" {
					return nil
				}
				parentFolderName := filepath.Base(filepath.Dir(path))
				if hasStructuredOneDriveDealIdentity(parentFolderName) && !hasStructuredOneDriveDealIdentity(filepath.Base(path)) {
					return filepath.SkipDir
				}
				folderPath = path
				folderName = filepath.Base(folderPath)
				finalPath = sections["OFFER"]
				if finalPath == "" {
					finalPath = sections["WORKING"]
				}
				for _, sectionName := range []string{"RFQ", "OFFER", "WORKING", "EXECUTION"} {
					sectionPath := sections[sectionName]
					if sectionPath == "" {
						continue
					}
					files = append(files, collectDealFiles(sectionPath)...)
				}
				isDeal = true
			}

			if !isDeal {
				return nil
			}
			result.TotalFiles += len(files)

			matches := matchCustomerToFolder(folderName, customers)
			deal := DiscoveredDeal{
				LocalID:         uuid.New().String(),
				FolderPath:      folderPath,
				FolderName:      folderName,
				FinalPath:       finalPath,
				RootPath:        rootPath,
				CustomerMatches: matches,
				Files:           files,
				InstrumentType:  extractInstrumentType(folderName),
				YearHint:        extractYearHint(folderName),
				Status:          "pending",
			}

			result.Deals = append(result.Deals, deal)
			result.TotalFolders++

			if len(result.Deals)%50 == 0 {
				log.Printf("✅ OneDrive scan: %d deals found so far...", len(result.Deals))
			}

			if len(result.Deals) >= maxDeals {
				log.Printf("⚠️ OneDrive scan capped at %d deals — stopping early", maxDeals)
				return filepath.SkipAll
			}

			if !strings.EqualFold(filepath.Base(path), "final") {
				return filepath.SkipDir
			}
			return nil
		})

		if err != nil && err != filepath.SkipAll {
			scanErrors = append(scanErrors, fmt.Sprintf("error scanning %s: %v", rootPath, err))
		}
	}

	// Sort deals alphabetically by folder name for a predictable UI order
	sort.Slice(result.Deals, func(i, j int) bool {
		return result.Deals[i].FolderName < result.Deals[j].FolderName
	})

	result.Errors = scanErrors
	log.Printf("✅ OneDrive scan complete: %d deals, %d files across %d root paths",
		len(result.Deals), result.TotalFiles, len(paths))
	return result, nil
}

// ConfirmOneDriveDeal validates that the supplied customerID exists in the DB
// and returns the customer's BusinessName.  The frontend holds all scan state;
// this is purely a server-side validation call.
func (a *App) ConfirmOneDriveDeal(localID string, customerID string) (map[string]any, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if customerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}

	var customer CustomerMaster
	if err := a.db.Where("id = ? AND deleted_at IS NULL", customerID).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to look up customer. Please try again or contact support")
	}

	return map[string]any{
		"local_id":      localID,
		"customer_id":   customer.ID,
		"business_name": customer.BusinessName,
		"short_code":    customer.ShortCode,
	}, nil
}

// ImportOneDriveDeals processes confirmed deals, upserting opportunities and
// then creating/updating offers plus downstream execution records when present.
func (a *App) ImportOneDriveDeals(deals []DiscoveredDeal) ([]OneDriveImportResult, error) {
	if err := a.requirePermission("finance:create"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	a.ensureOneDriveImportSchema()

	var results []OneDriveImportResult

	for _, deal := range deals {
		if deal.ConfirmedCustomerID == "" {
			results = append(results, OneDriveImportResult{
				DealLocalID: deal.LocalID,
				Success:     false,
				Message:     "skipped: no customer confirmed",
			})
			continue
		}

		res := a.importSingleDeal(deal)
		results = append(results, res)
	}

	log.Printf("✅ OneDrive import complete: %d deals processed", len(results))
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "data:refresh", map[string]any{
			"source": "onedrive_import",
			"count":  len(results),
		})
	}
	return results, nil
}

func (a *App) ensureOneDriveImportSchema() {
	// Older runtime databases may predate the canonical pipeline columns added
	// during startup migrations. Import needs these fields available even when it
	// is run from a test helper or maintenance path that bypasses full startup.
	a.addColumnIfNotExists("opportunities", "product_details", "TEXT")
}

// importSingleDeal handles the DB work for one confirmed DiscoveredDeal.
// It is an unexported helper so it can be called within a loop without
// duplicating the DB-nil guard.
func (a *App) importSingleDeal(deal DiscoveredDeal) OneDriveImportResult {
	res := OneDriveImportResult{DealLocalID: deal.LocalID}
	identity := deriveOneDriveImportIdentity(deal.FolderName)
	folderMeta := parseOneDriveFolderMeta(deal.FolderName)
	if identity.Year == 0 {
		identity.Year = inferYearFromPath(deal.RootPath, deal.FolderPath)
	}
	if folderMeta.Year == 0 {
		folderMeta.Year = identity.Year
	}
	if identity.CanonicalFolderNumber == "" && identity.Prefix != "" && identity.SequenceToken != "" && identity.Year != 0 {
		identity.CanonicalFolderNumber = deriveCanonicalOneDriveFolderNumber(identity.Prefix, identity.SequenceToken, identity.Year)
	}
	if identity.Year != 0 {
		folderMeta.Year = identity.Year
	}
	if identity.Title != "" {
		folderMeta.Title = identity.Title
	}
	if identity.RawFolderNumber != "" {
		folderMeta.FolderNumber = identity.RawFolderNumber
	}

	// Resolve the customer
	var customer CustomerMaster
	if err := a.db.Where("id = ? AND deleted_at IS NULL", deal.ConfirmedCustomerID).
		First(&customer).Error; err != nil {
		res.Success = false
		res.Message = fmt.Sprintf("customer lookup failed: %v", err)
		return res
	}

	// Collect costing sheets and parse them
	var costingData []parsedOneDriveCosting
	for _, f := range deal.Files {
		if f.FileType != "costing_sheet" {
			continue
		}
		parsed, err := ParseCostingSheet(f.FilePath)
		if err != nil {
			log.Printf("⚠️ OneDrive import: failed to parse costing sheet %s: %v", f.FilePath, err)
			continue
		}
		costingData = append(costingData, parsedOneDriveCosting{
			File:       f,
			Parsed:     parsed,
			Revision:   extractPathRevisionNumber(f.FilePath),
			OptionName: extractPathOptionName(f.FilePath),
		})
		res.CostingSheetsImported++
	}

	// Count PDFs that will be associated (queued for later document attachment)
	for _, f := range deal.Files {
		if f.Extension == ".pdf" {
			res.PDFsQueued++
		}
	}

	// Determine totals from first available costing sheet
	var totalValueBHD float64
	var estimatedMargin float64
	var quotationDate time.Time
	var offerItems []OfferItem
	var subtotalBHD float64
	var vatAmountBHD float64
	var vatPercent float64
	paymentTerms := ""
	deliveryTerms := ""
	customerReference := ""
	contactPerson := ""
	countryOfOrigin := ""
	deliveryWeeks := ""

	primaryCosting := selectPrimaryOneDriveCosting(costingData)
	if primaryCosting != nil {
		primary := primaryCosting.Parsed
		totalValueBHD = primary.Totals.GrandTotal
		subtotalBHD = primary.Totals.Subtotal
		vatAmountBHD = primary.Totals.VatAmount
		vatPercent = primary.Totals.VatPercent
		paymentTerms = strings.TrimSpace(primary.Metadata.PaymentTerms)
		deliveryTerms = strings.TrimSpace(primary.Metadata.DeliveryTerms)
		customerReference = strings.TrimSpace(primary.Metadata.Reference)
		contactPerson = strings.TrimSpace(primary.Metadata.ContactPerson)
		countryOfOrigin = strings.TrimSpace(primary.Metadata.CountryOfOrigin)
		deliveryWeeks = strings.TrimSpace(primary.Metadata.EstDelivery)

		// Compute margin using the same logic as calculateMargin in excel_costing_parser.go
		if primary.Totals.Subtotal > 0 {
			totalCost := 0.0
			for _, item := range primary.LineItems {
				totalCost += item.TotalCost * item.Quantity
			}
			if totalCost > 0 {
				estimatedMargin = (primary.Totals.Subtotal - totalCost) / primary.Totals.Subtotal * 100
			}
		}

		// Try to parse the date from the costing sheet metadata
		quotationDate = parseTimeWithLayouts(primary.Metadata.Date)

		// Build offer items
		for i, item := range primary.LineItems {
			oi := OfferItem{
				Base:          Base{ID: uuid.New().String()},
				LineNumber:    i + 1,
				ProductCode:   item.Model,
				Model:         item.Model,
				Description:   fmt.Sprintf("%s - %s", item.Equipment, item.Model),
				Quantity:      item.Quantity,
				UnitPrice:     item.SuggestedPriceBHD,
				Equipment:     item.Equipment,
				Specification: item.Specification,
				Currency:      "",
				FOB:           item.FobBHD,
				Freight:       item.FreightBHD,
				TotalCost:     item.TotalCost,
				MarginPercent: item.MarkupPercent,
				TotalPrice:    item.TotalSuggestedBHD,
				FobBHD:        item.FobBHD,
				FreightBHD:    item.FreightBHD,
				Insurance:     item.Insurance,
				CustomsBHD:    item.Customs,
				OtherCosts:    item.OtherCosts,
			}
			offerItems = append(offerItems, oi)
		}

		if specs := extractAnnexureSpecsFromDealFiles(deal.Files); len(specs) > 0 {
			if enriched := enrichOfferItemsWithAnnexureSpecs(offerItems, specs); enriched > 0 {
				log.Printf("✅ OneDrive import: enriched %d offer item(s) with annexure specs for %s", enriched, deal.FolderName)
			}
		}
	}

	if quotationDate.IsZero() {
		quotationDate = maxModTime(deal.Files, "quotation", "")
	}
	if quotationDate.IsZero() {
		quotationDate = maxModTime(deal.Files, "costing_sheet", "")
	}
	if quotationDate.IsZero() {
		quotationDate = time.Now()
	}

	stage := detectOneDriveStage(deal.Files)
	// Opportunity.Stage must always land on the canonical enum
	// (stage_vocabulary.go). Offer.Stage keeps its own separate DB-CHECK
	// vocabulary and is written straight from `stage` below — never from
	// oppStage. Importers are lenient: coerce + log, never abort the import.
	oppStage, _ := canonicalizeOpportunityStage(stage)
	if !isCanonicalOpportunityStage(oppStage) {
		log.Printf("⚠️ OneDrive import: unrecognized opportunity stage %q for %s, coercing to \"New\"", stage, deal.FolderName)
		oppStage = "New"
	}
	if paymentTerms == "" {
		if customer.PaymentTermsDays > 0 {
			paymentTerms = fmt.Sprintf("%d days", customer.PaymentTermsDays)
		}
	}

	opportunityFolderNumber := strings.TrimSpace(identity.CanonicalFolderNumber)
	if opportunityFolderNumber == "" {
		opportunityFolderNumber = strings.TrimSpace(identity.RawFolderNumber)
	}

	candidates, err := a.findOneDriveOpportunityCandidates(identity, deal.FolderName)
	if err != nil {
		res.Success = false
		res.Message = fmt.Sprintf("failed to load opportunity: %v", err)
		return res
	}

	var opportunity Opportunity
	var duplicateOpportunities []Opportunity
	isNewOpportunity := len(candidates) == 0
	if isNewOpportunity {
		opportunity = Opportunity{
			Base:         Base{ID: uuid.New().String()},
			FolderNumber: opportunityFolderNumber,
		}
	} else {
		opportunity = candidates[0]
		if len(candidates) > 1 {
			duplicateOpportunities = candidates[1:]
		}
	}

	opportunity.FolderNumber = opportunityFolderNumber
	opportunity.CustomerID = customer.ID
	opportunity.CustomerName = customer.BusinessName
	opportunity.CustomerGrade = customer.CustomerGrade
	opportunity.Year = folderMeta.Year
	opportunity.OppNumber = folderMeta.OppNumber
	opportunity.FolderName = deal.FolderName
	opportunity.Title = folderMeta.Title
	opportunity.Source = oneDriveOpportunitySource(opportunity.Year)
	opportunity.ProductType = deal.InstrumentType
	opportunity.OfferDate = quotationDate
	opportunity.PaymentTerms = paymentTerms
	opportunity.DeliveryTerms = deliveryTerms
	if strings.TrimSpace(identity.RawFolderNumber) != "" {
		opportunity.EHRef = identity.RawFolderNumber
	}
	opportunity.RevenueBHD = totalValueBHD
	opportunity.CostBHD = totalValueBHD - (estimatedMargin * totalValueBHD / 100.0)
	opportunity.ProfitBHD = totalValueBHD - opportunity.CostBHD
	opportunity.Stage = oppStage
	if stage == "Won" {
		closedAt := maxModTime(deal.Files, "", string(filepath.Separator)+"EXECUTION"+string(filepath.Separator))
		if closedAt.IsZero() {
			closedAt = quotationDate
		}
		opportunity.ClosedDate = &closedAt
		opportunity.OrderDate = &closedAt
	}
	productDetails := serializeOpportunityProductDetailsFromOfferItems(offerItems)

	opportunityValues := map[string]any{
		"folder_number":   opportunity.FolderNumber,
		"customer_id":     opportunity.CustomerID,
		"customer_name":   opportunity.CustomerName,
		"customer_grade":  opportunity.CustomerGrade,
		"year":            opportunity.Year,
		"opp_number":      opportunity.OppNumber,
		"folder_name":     opportunity.FolderName,
		"title":           opportunity.Title,
		"eh_ref":          opportunity.EHRef,
		"source":          opportunity.Source,
		"product_type":    opportunity.ProductType,
		"offer_date":      opportunity.OfferDate,
		"payment_terms":   opportunity.PaymentTerms,
		"delivery_terms":  opportunity.DeliveryTerms,
		"product_details": productDetails,
		"revenue_bhd":     opportunity.RevenueBHD,
		"cost_bhd":        opportunity.CostBHD,
		"profit_bhd":      opportunity.ProfitBHD,
		"stage":           opportunity.Stage,
		"closed_date":     opportunity.ClosedDate,
		"order_date":      opportunity.OrderDate,
		"updated_at":      time.Now(),
	}

	if isNewOpportunity {
		opportunityValues["id"] = opportunity.ID
		opportunityValues["created_at"] = time.Now()
		opportunityValues["version"] = 1
		opportunityValues["created_by"] = a.getCurrentUserID()
		opportunityValues["division"] = normalizeDivisionName(opportunity.Division)
		if err := a.db.Table("opportunities").Create(opportunityValues).Error; err != nil {
			res.Success = false
			res.Message = fmt.Sprintf("failed to create opportunity: %v", err)
			return res
		}
	} else {
		if err := a.db.Table("opportunities").Where("id = ?", opportunity.ID).Updates(opportunityValues).Error; err != nil {
			res.Success = false
			res.Message = fmt.Sprintf("failed to update opportunity: %v", err)
			return res
		}
	}

	var offer *Offer
	offerNumber := deriveImportedOfferNumber(identity, deal)
	if primaryCosting != nil {
		var existingOffer Offer
		staleOfferID := ""
		switch {
		case opportunity.OfferID != "":
			err = a.db.Unscoped().Preload("Items").Where("id = ?", opportunity.OfferID).First(&existingOffer).Error
			if err == nil && !importedOfferMatchesDeal(existingOffer, offerNumber, customer.ID) {
				staleOfferID = existingOffer.ID
				err = gorm.ErrRecordNotFound
			}
		default:
			err = gorm.ErrRecordNotFound
		}
		if err == gorm.ErrRecordNotFound {
			existingOffer = Offer{}
			err = a.db.Unscoped().Preload("Items").Where("offer_number = ?", offerNumber).First(&existingOffer).Error
			if err == nil && !importedOfferMatchesDeal(existingOffer, offerNumber, "") {
				staleOfferID = existingOffer.ID
				err = gorm.ErrRecordNotFound
			}
		}

		isNewOffer := err == gorm.ErrRecordNotFound
		if err != nil && err != gorm.ErrRecordNotFound {
			res.Success = false
			res.Message = fmt.Sprintf("failed to load offer: %v", err)
			return res
		}
		if isNewOffer {
			existingOffer = Offer{
				Base:        Base{ID: uuid.New().String()},
				OfferNumber: offerNumber,
			}
		}

		existingOffer.RevisionNumber = extractRevisionNumber(deal.Files)
		existingOffer.CustomerID = customer.ID
		existingOffer.CustomerName = customer.BusinessName
		existingOffer.QuotationDate = quotationDate
		existingOffer.ValidityDate = quotationDate.AddDate(0, 3, 0)
		existingOffer.TotalValueBHD = totalValueBHD
		existingOffer.EstimatedMargin = estimatedMargin
		existingOffer.Stage = stage
		existingOffer.PaymentTerms = paymentTerms
		existingOffer.DeliveryTerms = deliveryTerms
		existingOffer.DeliveryWeeks = deliveryWeeks
		existingOffer.CountryOfOrigin = countryOfOrigin
		existingOffer.CustomerReference = customerReference
		existingOffer.AttentionPerson = contactPerson
		if vatPercent > 0 {
			existingOffer.VatRate = vatPercent
		}
		existingOffer.DeletedAt = gorm.DeletedAt{}

		if isNewOffer {
			if err := a.db.Create(&existingOffer).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to create offer for %s (%s): %v", deal.FolderName, offerNumber, err)
				return res
			}
		} else {
			if err := a.db.Unscoped().Save(&existingOffer).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to update offer for %s (%s): %v", deal.FolderName, offerNumber, err)
				return res
			}
		}

		if len(offerItems) > 0 {
			if err := a.db.Where("offer_id = ?", existingOffer.ID).Delete(&OfferItem{}).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to refresh offer items: %v", err)
				return res
			}
			for i := range offerItems {
				offerItems[i].OfferID = existingOffer.ID
			}
			if err := a.db.Create(&offerItems).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to create offer items: %v", err)
				return res
			}
		}

		offer = &existingOffer
		opportunity.OfferID = existingOffer.ID
		opportunity.Stage = oppStage
		if err := a.db.Model(&Opportunity{}).Where("id = ?", opportunity.ID).Updates(map[string]any{
			"offer_id": offer.ID,
			"stage":    opportunity.Stage,
		}).Error; err != nil {
			res.Success = false
			res.Message = fmt.Sprintf("failed to link opportunity to offer: %v", err)
			return res
		}
		res.OfferID = offer.ID
		if staleOfferID != "" && staleOfferID != offer.ID {
			if err := a.softDeleteOfferIfUnlinked(staleOfferID); err != nil {
				log.Printf("⚠️ OneDrive import: failed to soft-delete stale offer %s: %v", staleOfferID, err)
			}
		}
	} else if strings.TrimSpace(opportunity.OfferID) != "" {
		if err := a.clearPlaceholderCommercialChain(opportunity.ID, opportunity.OfferID); err != nil {
			log.Printf("⚠️ OneDrive import: failed to clear placeholder commercial chain for %s: %v", deal.FolderName, err)
		}
		opportunity.OfferID = ""
	}

	if len(duplicateOpportunities) > 0 {
		for _, duplicate := range duplicateOpportunities {
			if duplicate.ID == opportunity.ID {
				continue
			}
			if err := softDeleteDuplicateOpportunity(a.db, duplicate.ID); err != nil {
				log.Printf("⚠️ OneDrive import: failed to soft-delete duplicate opportunity %s: %v", duplicate.ID, err)
			}
		}
	}

	if stage == "Won" && offer != nil {
		customerPONumber := extractPONumber(deal.Files)
		orderNumber := strings.TrimSpace(offer.OfferNumber)
		if orderNumber == "" {
			orderNumber = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(folderMeta.FolderNumber), " ", "-"))
		}
		if orderNumber == "" && len(deal.LocalID) >= 8 {
			orderNumber = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(deal.LocalID[:8]), " ", "-"))
		}
		if orderNumber == "" {
			orderNumber = fmt.Sprintf("ORD-%d", time.Now().Unix())
		}

		var order Order
		err := a.db.Preload("Items").Where("offer_id = ?", offer.ID).First(&order).Error
		isNewOrder := err == gorm.ErrRecordNotFound
		if err != nil && err != gorm.ErrRecordNotFound {
			res.Success = false
			res.Message = fmt.Sprintf("failed to load order: %v", err)
			return res
		}
		if isNewOrder {
			var reusedOrder Order
			reuseErr := a.db.Unscoped().
				Preload("Items").
				Where("order_number = ?", orderNumber).
				First(&reusedOrder).Error
			if reuseErr == nil {
				order = reusedOrder
				isNewOrder = false
			} else if reuseErr != nil && reuseErr != gorm.ErrRecordNotFound {
				res.Success = false
				res.Message = fmt.Sprintf("failed to load reusable order: %v", reuseErr)
				return res
			}
		}
		orderDate := maxModTime(deal.Files, "", string(filepath.Separator)+"EXECUTION"+string(filepath.Separator))
		if orderDate.IsZero() {
			orderDate = quotationDate
		}
		if isNewOrder {
			order = Order{
				Base:        Base{ID: uuid.New().String()},
				OrderNumber: orderNumber,
			}
		}
		order.CustomerPONumber = customerPONumber
		order.CustomerID = customer.ID
		order.CustomerName = customer.BusinessName
		order.OrderDate = orderDate
		order.RequiredDate = orderDate
		order.TotalValueBHD = subtotalBHD
		if order.TotalValueBHD == 0 {
			order.TotalValueBHD = totalValueBHD
		}
		order.GrandTotalBHD = totalValueBHD
		order.Status = "Confirmed"
		order.PaymentTerms = paymentTerms
		order.DeliveryTerms = deliveryTerms
		order.OfferID = offer.ID
		order.OfferNumber = offer.OfferNumber
		order.RFQID = opportunity.ID
		order.CustomerReference = customerReference
		order.AttentionPerson = contactPerson
		order.DeliveryWeeks = deliveryWeeks
		order.CountryOfOrigin = countryOfOrigin
		order.DeletedAt = gorm.DeletedAt{}

		if isNewOrder {
			if err := a.db.Create(&order).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to create order: %v", err)
				return res
			}
		} else {
			if err := a.db.Save(&order).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to update order: %v", err)
				return res
			}
		}

		if len(offerItems) > 0 {
			orderItems := buildOrderItemsFromOfferItems(offerItems, order.ID)
			if err := a.db.Where("order_id = ?", order.ID).Delete(&OrderItem{}).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to refresh order items: %v", err)
				return res
			}
			if err := a.db.Create(&orderItems).Error; err != nil {
				res.Success = false
				res.Message = fmt.Sprintf("failed to create order items: %v", err)
				return res
			}
		}

		invoiceNumber := extractInvoiceNumber(deal.Files)
		if invoiceNumber != "" {
			var invoice Invoice
			err := a.db.Where("invoice_number = ?", invoiceNumber).First(&invoice).Error
			isNewInvoice := err == gorm.ErrRecordNotFound
			if err != nil && err != gorm.ErrRecordNotFound {
				res.Success = false
				res.Message = fmt.Sprintf("failed to load invoice: %v", err)
				return res
			}
			if isNewInvoice {
				invoice = Invoice{
					Base:          Base{ID: uuid.New().String()},
					InvoiceNumber: invoiceNumber,
				}
			}
			invoiceDate := maxModTime(deal.Files, "invoice", "")
			if invoiceDate.IsZero() {
				invoiceDate = orderDate
			}
			invoice.InvoiceDate = invoiceDate
			invoice.CustomerID = customer.ID
			invoice.CustomerName = customer.BusinessName
			invoice.OrderID = order.ID
			invoice.Division = normalizeDivisionName(order.Division)
			invoice.CustomerPONumber = customerPONumber
			invoice.GrandTotalBHD = order.GrandTotalBHD
			invoice.OutstandingBHD = order.GrandTotalBHD
			invoice.SubtotalBHD = order.TotalValueBHD
			invoice.DueDate = invoiceDate.AddDate(0, 0, 30)
			invoice.Status = "Draft"
			invoice.RfqID = opportunity.ID
			invoice.OfferID = offer.ID
			invoice.OfferNumber = offer.OfferNumber
			invoice.PaymentTerms = paymentTerms
			invoice.DeliveryTerms = deliveryTerms
			invoice.CustomerReference = customerReference
			invoice.AttentionPerson = contactPerson
			invoice.CountryOfOrigin = countryOfOrigin
			invoice.DeliveryWeeks = deliveryWeeks
			invoice.VATBHD = vatAmountBHD
			invoice.VATPercent = vatPercent

			if isNewInvoice {
				if err := a.db.Create(&invoice).Error; err != nil {
					res.Success = false
					res.Message = fmt.Sprintf("failed to create invoice: %v", err)
					return res
				}
			} else {
				if err := a.db.Save(&invoice).Error; err != nil {
					res.Success = false
					res.Message = fmt.Sprintf("failed to update invoice: %v", err)
					return res
				}
			}

			// PH convergence B1 (PH 3c5127b): populate the invoice's line items
			// inline from the order, so a won-import never leaves a hollow
			// invoice sitting until the next restart's backfill. Delete-then-
			// create inside makes re-imports idempotent, matching PH. A failure
			// here is non-fatal: the header exists and the startup backfill
			// repairs it on the next launch.
			if _, _, err := a.repairInvoiceItemsFromOrder(invoice.ID); err != nil {
				log.Printf("⚠️ OneDrive import: could not populate items for invoice %s: %v", invoice.InvoiceNumber, err)
			}
		}
	}

	log.Printf("✅ OneDrive import: processed %s for %s (stage=%s, %d items, %.3f BHD)",
		deal.FolderName, customer.BusinessName, stage, len(offerItems), totalValueBHD)

	res.Success = true
	res.Message = fmt.Sprintf("imported %s as %s (%d costing items, %d PDFs noted)",
		deal.FolderName, stage, len(offerItems), res.PDFsQueued)
	return res
}

// folderNumberHasDigit reports whether s contains at least one digit. A real
// opportunity folder number always contains a digit; a purely alphabetic
// string (e.g. a customer word) is never a valid folder number.
func folderNumberHasDigit(s string) bool {
	return strings.ContainsAny(s, "0123456789")
}

// cleanLooseOneDriveFolderNumberToken (ported from PH onedrive_import_service.go)
// trims a loose folder-number token and rejects it entirely if it carries no
// digit. A real folder NUMBER always contains a digit; a digit-less token is a
// customer name, not a folder number — accepting it as the loose fallback
// collapsed every OneDrive opportunity for that customer onto one canonical key
// (e.g. all of Bapco's deals keyed as "BAPCO").
func cleanLooseOneDriveFolderNumberToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	token = strings.Trim(token, "-_/ ")
	if !folderNumberHasDigit(token) {
		return ""
	}
	return token
}

// splitLooseOneDriveFolderNumberToken (ported from PH onedrive_import_service.go)
// splits a cleaned loose token into (folderNumber, title). A leading numeric run
// followed by a separator and remainder (e.g. "150-AIRMENCH") yields folder
// "150" and title "AIRMENCH". Digit-less tokens are rejected up front.
func splitLooseOneDriveFolderNumberToken(token string) (string, string) {
	token = cleanLooseOneDriveFolderNumberToken(token)
	if token == "" {
		return "", ""
	}
	if m := regexp.MustCompile(`^(\d{1,3})(?:[-_/]+(.+))?$`).FindStringSubmatch(token); len(m) == 3 {
		return m[1], strings.TrimSpace(strings.Trim(m[2], "-_/ "))
	}
	// Defense in depth: cleanLooseOneDriveFolderNumberToken already strips
	// digit-less tokens, but never surface one as a folder number even if that
	// guard later changes — a folder number must contain a digit.
	if !folderNumberHasDigit(token) {
		return "", ""
	}
	return token, ""
}
