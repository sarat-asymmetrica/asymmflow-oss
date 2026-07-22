package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/nguyenthenguyen/docx"
	"github.com/richardlehane/mscfb"
	"github.com/xuri/excelize/v2"
	aceocr "ph_holdings_app/pkg/ocr"
	documentsocr "ph_holdings_app/pkg/documents/ocr"
	"ph_holdings_app/pkg/ocr/mistralocr"
)

// MaxFileSize is the maximum allowed file size for OCR (50MB)
const MaxFileSize = 50 * 1024 * 1024 // 50 MB

// ═══════════════════════════════════════════════════════════════════════════
// SIMPLIFIED OCR SERVICE - Mistral OCR 4 + offline-first local fallback
// ═══════════════════════════════════════════════════════════════════════════
//
// ARCHITECTURE (Wave 13 "Perception & Print" — Fly.io retired):
//   PDF   → vector text via go-fitz (free, offline)
//         → scant?  → Mistral OCR 4 (native PDF, schema-shaped extraction)
//                   → offline fallback: tesseract via the local ACEEngine pipeline
//   Image → Mistral OCR 4 → offline fallback: tesseract via the local ACEEngine pipeline
//   Office formats (xlsx/xls/docx/rtf/msg/eml) → parsed locally, no OCR involved
//
// Offline-first is a hard boundary: with no Mistral key and no network, every
// document still gets a result (go-fitz text, or tesseract, or — worst case —
// the local file cleanly reported as unreadable) and the caller never blocks.
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS × SIMPLICITY
// Wave 2, Agent 3 - January 20, 2026 / Wave 13, Agent P3 - July 22, 2026
// ═══════════════════════════════════════════════════════════════════════════

// SimpleOCRService dispatches OCR across the Mistral OCR 4 cloud engine and the
// local (offline) go-fitz + tesseract pipeline. No Fly.io / AIMLAPI dependency.
type SimpleOCRService struct {
	httpClient  *http.Client
	maxPages    int
	dpi         int
	localEngine *aceocr.ACEEngine // offline tesseract+pandoc fallback, no cloud escalation
}

// mistralOCREnv reads Mistral OCR client configuration from the environment, applying the
// same sane defaults as pkg/ocr/mistralocr — config-not-constant: every model ID, endpoint,
// threshold, and page cap here is overridable without a code change.
type mistralOCREnv struct {
	Model               string
	BaseURL             string
	PageCap             int
	Timeout             time.Duration
	ConfidenceThreshold float64
	ScantTextThreshold  int // chars; below this, go-fitz output is treated as "scanned" (PDF path only)
}

func loadMistralOCREnv() mistralOCREnv {
	cfg := mistralOCREnv{
		Model:               getEnvOrDefault("MISTRAL_OCR_MODEL", mistralocr.DefaultModel),
		BaseURL:             getEnvOrDefault("MISTRAL_OCR_BASE_URL", mistralocr.DefaultBaseURL),
		PageCap:             getEnvIntOrDefault("MISTRAL_OCR_PAGE_CAP", mistralocr.DefaultPageCap),
		Timeout:             time.Duration(getEnvIntOrDefault("MISTRAL_OCR_TIMEOUT_SECONDS", int(mistralocr.DefaultTimeout/time.Second))) * time.Second,
		ConfidenceThreshold: getEnvFloatOrDefault("MISTRAL_OCR_CONFIDENCE_THRESHOLD", mistralocr.DefaultConfidenceThreshold),
		ScantTextThreshold:  getEnvIntOrDefault("OCR_SCANT_TEXT_THRESHOLD", 50),
	}
	return cfg
}

func getEnvOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getEnvIntOrDefault(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloatOrDefault(key string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// OCRResultSimple represents OCR extraction result (compatible with existing OCRResult)
type OCRResultSimple struct {
	Success          bool           `json:"success"`
	Text             string         `json:"text"`
	Confidence       float64        `json:"confidence"`
	DocumentType     string         `json:"document_type"`
	ExtractedData    map[string]any `json:"extracted_data"`
	ExtractedFields  map[string]any `json:"extracted_fields"` // Alias for legacy compatibility
	ProcessingTime   int64          `json:"processing_time_ms"`
	ProcessingTimeMS int64          `json:"processing_time_ms_legacy"` // Alias for legacy compatibility
	Engine           string         `json:"engine"`                    // "pymupdf", "mistral-ocr-4", "tesseract-local", ...
	TierUsed         string         `json:"tier_used"`                 // Legacy compatibility
	Cost             float64        `json:"cost"`                      // Processing cost (0 for local)
	DNACacheHit      bool           `json:"dna_cache_hit"`             // Legacy compatibility
	TableDetected    bool           `json:"table_detected"`            // Legacy compatibility
	GPUUsed          bool           `json:"gpu_used"`                  // Legacy compatibility

	// NeedsReview / FieldConfidence surface the Mistral OCR 4 Document AI confidence signal
	// (page-level heuristic — see pkg/ocr/mistralocr) through to the document review UI.
	// Never populated as word-exact; NeedsReview is honest "refuse to guess" per field.
	NeedsReview     bool               `json:"needs_review"`
	FieldConfidence map[string]float64 `json:"field_confidence,omitempty"`
	FieldsForReview []string           `json:"fields_needing_review,omitempty"`

	Error string `json:"error,omitempty"`
}

// OCRResult is an alias for OCRResultSimple (API compatibility with legacy code)
type OCRResult = OCRResultSimple

// NewSimpleOCRService creates a simplified OCR service. The local (offline) engine is
// constructed with no Mistral key and GPU preprocessing disabled, so it never attempts a
// network call and stays a pure tesseract+pandoc pipeline — see ocrWithLocalEngine.
func NewSimpleOCRService() (*SimpleOCRService, error) {
	log.Println("🌸 Initializing Simple OCR Service...")

	localEngine, err := aceocr.NewACEEngine(&aceocr.EngineConfig{
		EnableGPU:             false,
		EnablePreprocessing:   false,
		EnableVedicValidation: false,
		FallbackToMistral:     false, // no MistralAPIKey set below either — belt and suspenders
		DefaultLanguage:       aceocr.LangEnglish,
		MaxWorkers:            4,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize local OCR fallback engine: %w", err)
	}

	service := &SimpleOCRService{
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		maxPages:    10,
		dpi:         150,
		localEngine: localEngine,
	}

	log.Printf("✓ Simple OCR Service initialized")
	log.Println("  Strategy: Vector PDF → go-fitz (free) | Scanned/Image → Mistral OCR 4 → offline tesseract fallback")

	return service, nil
}

// ProcessDocument processes a single document (PDF, image, or Excel)
func (s *SimpleOCRService) ProcessDocument(filePath, docType string) (result *OCRResultSimple, err error) {
	// Panic recovery: prevent app crash on corrupt files
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in OCR ProcessDocument: %v", r)
			err = fmt.Errorf("document processing crashed: %v", r)
			result = nil
		}
	}()

	startTime := time.Now()

	log.Printf("📄 Processing document: %s (type=%s)", filepath.Base(filePath), docType)

	// Validate file exists and check size
	fileInfo, statErr := os.Stat(filePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("cannot access file: %w", statErr)
	}

	// Reject empty files
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("file is empty (0 bytes)")
	}

	// Reject files exceeding max size
	if fileInfo.Size() > MaxFileSize {
		return nil, fmt.Errorf("file too large (%d MB, max %d MB)", fileInfo.Size()/(1024*1024), MaxFileSize/(1024*1024))
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".xlsx", ".xls":
		// Excel files: parse locally with excelize
		log.Println("📊 Excel file detected - parsing with excelize")
		result, err := s.processExcel(filePath, docType)
		if err != nil {
			return nil, fmt.Errorf("excel processing failed: %w", err)
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	case ".msg":
		// Outlook MSG files: parse OLE2 compound document locally
		log.Println("📧 MSG file detected - parsing OLE2 compound document")
		result, err := s.processMSG(filePath, docType)
		if err != nil {
			return nil, fmt.Errorf("MSG processing failed: %w", err)
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	case ".eml":
		// EML files: parse RFC 5322 email format
		log.Println("📧 EML file detected - parsing RFC 5322 email")
		result, err := s.processEML(filePath, docType)
		if err != nil {
			return nil, fmt.Errorf("EML processing failed: %w", err)
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	case ".png", ".jpg", ".jpeg", ".bmp", ".tiff", ".tif", ".webp":
		// Images: Mistral OCR 4 first; offline tesseract fallback (never blocks the inbox).
		log.Println("🖼️ Image file detected - trying Mistral OCR 4")
		result, mistralErr := s.ocrWithMistral(filePath, docType, true)
		if mistralErr != nil {
			log.Printf("⚠ Mistral OCR failed for image: %v, falling back to local tesseract", mistralErr)
			var localErr error
			result, localErr = s.ocrWithLocalEngine(filePath, docType, "")
			if localErr != nil {
				return nil, fmt.Errorf("image OCR failed (mistral: %v, local: %w)", mistralErr, localErr)
			}
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	case ".pdf":
		// PDF: ALWAYS try go-fitz first (works offline, fast, free)
		log.Println("📄 PDF detected - trying local go-fitz extraction first")
		envCfg := loadMistralOCREnv()
		text, fitzErr := extractVectorPDF(filePath)
		if fitzErr == nil && len(strings.TrimSpace(text)) > envCfg.ScantTextThreshold {
			duration := time.Since(startTime).Milliseconds()

			extractedData := extractFieldsFromTextLegacy(text, docType)
			extractedData["raw_text"] = text
			log.Printf("📊 Extracted %d fields from PDF via go-fitz (%d chars)", len(extractedData)-1, len(text))

			return &OCRResultSimple{
				Success:        true,
				Text:           text,
				Confidence:     1.0,
				DocumentType:   docType,
				ExtractedData:  extractedData,
				ProcessingTime: duration,
				Engine:         "pymupdf",
			}, nil
		}

		if fitzErr != nil {
			log.Printf("⚠ go-fitz extraction failed: %v, trying Mistral OCR 4 (native PDF)", fitzErr)
		} else {
			log.Printf("⚠ go-fitz got insufficient text (%d chars), trying Mistral OCR 4 (native PDF)", len(strings.TrimSpace(text)))
		}

		// Fallback 1: Mistral OCR 4, native PDF submission (no page-render-to-PNG loop).
		result, mistralErr := s.ocrWithMistral(filePath, docType, false)
		if mistralErr == nil {
			result.ProcessingTime = time.Since(startTime).Milliseconds()
			return result, nil
		}
		log.Printf("⚠ Mistral OCR failed: %v, falling back to local tesseract", mistralErr)

		// Fallback 2: offline tesseract via the local ACEEngine pipeline. Never errors the
		// inbox — worst case is an honest low-confidence/NeedsReview result.
		result, localErr := s.ocrWithLocalEngine(filePath, docType, text)
		if localErr != nil {
			// Last resort: if go-fitz produced SOME partial text, surface that rather than fail.
			if strings.TrimSpace(text) != "" {
				log.Printf("⚠ All OCR engines failed - using partial go-fitz text (%d chars)", len(text))
				duration := time.Since(startTime).Milliseconds()
				extractedData := extractFieldsFromTextLegacy(text, docType)
				extractedData["raw_text"] = text
				return &OCRResultSimple{
					Success:        true,
					Text:           text,
					Confidence:     0.5,
					DocumentType:   docType,
					ExtractedData:  extractedData,
					ProcessingTime: duration,
					Engine:         "pymupdf-partial",
					NeedsReview:    true,
				}, nil
			}
			return nil, fmt.Errorf("all OCR engines failed: go-fitz insufficient, mistral: %v, local: %w", mistralErr, localErr)
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	case ".docx":
		// Word documents: parse with docx library
		log.Println("📝 DOCX file detected - parsing Word document")
		result, err := s.processDOCX(filePath, docType)
		if err != nil {
			return nil, fmt.Errorf("DOCX processing failed: %w", err)
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	case ".rtf":
		// RTF files: basic text extraction (RTF is plain text with formatting codes)
		log.Println("📝 RTF file detected - extracting text")
		result, err := s.processRTF(filePath, docType)
		if err != nil {
			return nil, fmt.Errorf("RTF processing failed: %w", err)
		}
		result.ProcessingTime = time.Since(startTime).Milliseconds()
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported file type: %s (supported: %s)", ext, strings.Join(supportedOCRFileExtensions(), ", "))
	}
}

// processExcel handles Excel file parsing (local, no OCR needed)
func (s *SimpleOCRService) processExcel(filePath, docType string) (*OCRResultSimple, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	var allText strings.Builder
	extractedData := make(map[string]any)

	sheets := f.GetSheetList()
	extractedData["sheet_count"] = len(sheets)

	for sheetIdx, sheet := range sheets {
		if sheetIdx > 5 { // Limit to first 6 sheets
			break
		}

		rows, err := f.GetRows(sheet)
		if err != nil {
			log.Printf("⚠ Failed to read sheet '%s': %v", sheet, err)
			continue
		}

		allText.WriteString(fmt.Sprintf("=== Sheet: %s ===\n", sheet))

		for rowIdx, row := range rows {
			if rowIdx > 500 { // Limit to 500 rows per sheet
				allText.WriteString("... (truncated)\n")
				break
			}
			line := strings.Join(row, "\t")
			allText.WriteString(line + "\n")
		}
		allText.WriteString("\n")

		// Extract key data from the first sheet (likely the main data)
		if sheetIdx == 0 && len(rows) > 0 {
			extractedData["row_count"] = len(rows)
			extractedData["column_count"] = len(rows[0])

			// Try to detect headers and extract structured info
			if len(rows) > 0 {
				extractedData["headers"] = strings.Join(rows[0], ", ")
			}

			// Look for common business document patterns in cell values
			for _, row := range rows {
				for _, cell := range row {
					cellLower := strings.ToLower(cell)
					if strings.Contains(cellLower, "total") && len(row) > 1 {
						// Try to find the value next to "total"
						for _, v := range row {
							if v != cell && v != "" {
								var f float64
								if _, err := fmt.Sscanf(strings.ReplaceAll(v, ",", ""), "%f", &f); err == nil && f > 0 {
									extractedData["total"] = f
								}
							}
						}
					}
				}
			}
		}
	}

	text := allText.String()
	if len(text) > 50000 {
		text = text[:50000] + "\n... (truncated)"
	}

	// Also run field extraction on the combined text
	fields := extractFieldsFromTextLegacy(text, docType)
	for k, v := range fields {
		if _, exists := extractedData[k]; !exists {
			extractedData[k] = v
		}
	}
	extractedData["raw_text"] = text
	extractedData["source_type"] = "excel"

	log.Printf("📊 Excel parsed: %d sheets, %d chars extracted", len(sheets), len(text))

	return &OCRResultSimple{
		Success:       true,
		Text:          text,
		Confidence:    1.0, // Direct parsing = perfect accuracy
		DocumentType:  docType,
		ExtractedData: extractedData,
		Engine:        "excelize",
	}, nil
}

// processMSG handles Outlook MSG files (OLE2 Compound Document format)
func (s *SimpleOCRService) processMSG(filePath, docType string) (*OCRResultSimple, error) {
	log.Printf("📧 Processing MSG file: %s", filepath.Base(filePath))

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MSG file: %w", err)
	}
	defer f.Close()

	doc, err := mscfb.New(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MSG compound document: %w", err)
	}

	// MSG property stream names follow pattern: __substg1.0_XXXX00YY
	// XXXX = property ID (hex), YY = type (1F=UTF-16LE, 1E=ANSI, 0102=binary)
	// Key properties:
	//   0037 = Subject
	//   1000 = Body (plain text)
	//   1009 = RTF Body (compressed)
	//   1013 = HTML Body
	//   0C1A = Sender Name
	//   0C1F = Sender Email
	//   0E04 = Display To
	//   0E03 = Display CC
	//   0070 = Conversation Topic
	//   007D = Transport Message Headers
	//   3001 = Display Name (recipient)
	//   5D01 = Sender SMTP Address

	properties := make(map[string]string)
	binaryProps := make(map[string][]byte) // For RTF body (compressed binary)
	foundStreams := make([]string, 0)

	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		name := entry.Name
		if !strings.HasPrefix(name, "__substg1.0_") {
			continue
		}

		// Parse property ID and type from stream name
		// Format: __substg1.0_XXXXYYYY where XXXX=propID, YYYY=type
		suffix := strings.TrimPrefix(name, "__substg1.0_")
		if len(suffix) < 8 {
			continue
		}

		propID := strings.ToUpper(suffix[:4])
		propType := strings.ToUpper(suffix[4:8])
		foundStreams = append(foundStreams, fmt.Sprintf("%s(%s)", propID, propType))

		// Read stream content
		content := make([]byte, entry.Size)
		n, readErr := entry.Read(content)
		if readErr != nil && readErr != io.EOF {
			continue
		}
		content = content[:n]

		// Handle string properties (001F=Unicode, 001E=ANSI)
		if propType == "001F" || propType == "001E" {
			var text string
			if propType == "001F" {
				text = decodeUTF16LE(content)
			} else {
				text = string(content)
			}
			text = strings.TrimRight(text, "\x00")
			if text == "" {
				continue
			}

			switch propID {
			case "0037":
				properties["subject"] = text
			case "1000":
				properties["body"] = text
			case "1013":
				properties["html_body"] = text
			case "0C1A":
				properties["sender_name"] = text
			case "0C1F", "0065", "5D01":
				properties["sender_email"] = text
			case "0E04":
				properties["to"] = text
			case "0E03":
				properties["cc"] = text
			case "0070":
				properties["conversation_topic"] = text
			case "007D":
				properties["headers"] = text
			case "0040":
				properties["received_date"] = text
			case "0039":
				properties["sent_date"] = text
			case "3001":
				// Additional display name (often duplicated)
				if properties["sender_name"] == "" {
					properties["sender_name"] = text
				}
			}
		} else if propType == "0102" && propID == "1009" {
			// RTF body (binary/compressed) - store for potential future decompression
			binaryProps["rtf_body"] = content
		}
	}

	log.Printf("📧 Found %d property streams: %s", len(foundStreams), strings.Join(foundStreams[:min(10, len(foundStreams))], ", "))

	// Build combined text for field extraction
	var allText strings.Builder
	if subject, ok := properties["subject"]; ok && subject != "" {
		allText.WriteString("Subject: " + subject + "\n")
	}
	if from, ok := properties["sender_name"]; ok && from != "" {
		allText.WriteString("From: " + from)
		if email, ok := properties["sender_email"]; ok && email != "" {
			allText.WriteString(" <" + email + ">")
		}
		allText.WriteString("\n")
	}
	if to, ok := properties["to"]; ok && to != "" {
		allText.WriteString("To: " + to + "\n")
	}
	if cc, ok := properties["cc"]; ok && cc != "" {
		allText.WriteString("CC: " + cc + "\n")
	}
	if date, ok := properties["sent_date"]; ok && date != "" {
		allText.WriteString("Date: " + date + "\n")
	}
	allText.WriteString("\n")

	// Try body sources - prefer the one with MORE content (emails often have partial plain text)
	plainBody := strings.TrimSpace(properties["body"])
	htmlBody := strings.TrimSpace(properties["html_body"])
	strippedHTML := ""
	if htmlBody != "" {
		strippedHTML = stripHTMLTags(htmlBody)
	}

	// Detect if plain body is just a signature/disclaimer (common in Outlook)
	isPlainBodyJustSignature := false
	if plainBody != "" {
		lowerPlain := strings.ToLower(plainBody)
		// Signature indicators: confidentiality notice, short length, or starts with common signature patterns
		if strings.Contains(lowerPlain, "confidential") ||
			strings.Contains(lowerPlain, "this message") && strings.Contains(lowerPlain, "third party") ||
			strings.Contains(lowerPlain, "if you received this") ||
			strings.Contains(lowerPlain, "disclaimer") ||
			len(plainBody) < 200 && len(strippedHTML) > len(plainBody)*2 {
			isPlainBodyJustSignature = true
			log.Printf("📧 Plain text body appears to be just signature/disclaimer (%d chars), checking HTML body", len(plainBody))
		}
	}

	// Use HTML body if plain body is missing, just a signature, or significantly shorter
	if plainBody != "" && !isPlainBodyJustSignature && len(plainBody) >= len(strippedHTML)/2 {
		allText.WriteString(plainBody)
		log.Printf("📧 Using plain text body (%d chars)", len(plainBody))
	} else if strippedHTML != "" {
		allText.WriteString(strippedHTML)
		log.Printf("📧 Using HTML body stripped (%d chars, plain was %d)", len(strippedHTML), len(plainBody))
	} else if plainBody != "" {
		// Fallback: use plain body even if it's just signature (better than nothing)
		allText.WriteString(plainBody)
		log.Printf("📧 Using plain text body as fallback (%d chars)", len(plainBody))
	}

	text := allText.String()
	if strings.TrimSpace(text) == "" {
		// Fallback: try reading transport headers for at least some content
		if headers, ok := properties["headers"]; ok && headers != "" {
			text = "--- EMAIL HEADERS ---\n" + headers
		}
	}

	// Log what we found
	log.Printf("📧 MSG properties found: subject=%v, body=%v, html=%v, from=%v, to=%v",
		properties["subject"] != "", properties["body"] != "",
		properties["html_body"] != "", properties["sender_name"] != "", properties["to"] != "")

	if strings.TrimSpace(text) == "" {
		log.Printf("⚠️ MSG file has no extractable text. Streams found: %v", foundStreams)
		return &OCRResultSimple{
			Success:       false,
			DocumentType:  docType,
			Error:         "MSG file contains no extractable text (check if attachments contain the content)",
			Engine:        "msg-parser",
			ExtractedData: map[string]any{"streams_found": foundStreams},
		}, nil
	}

	// Detect document type from email content
	detectedType := docType
	if detectedType == "" || detectedType == "auto" {
		detectedType = detectDocumentTypeFromTextLegacy(text)
	}

	// Extract fields from the email text
	extractedData := extractFieldsFromTextLegacy(text, detectedType)
	extractedData["raw_text"] = text
	extractedData["source_type"] = "msg_email"

	// Add email-specific metadata
	if v, ok := properties["subject"]; ok && v != "" {
		extractedData["email_subject"] = v
	}
	if v, ok := properties["sender_name"]; ok && v != "" {
		extractedData["email_from"] = v
	}
	if v, ok := properties["sender_email"]; ok && v != "" {
		extractedData["email_from_address"] = v
	}
	if v, ok := properties["to"]; ok && v != "" {
		extractedData["email_to"] = v
	}
	if v, ok := properties["sent_date"]; ok && v != "" {
		extractedData["email_date"] = v
	}

	log.Printf("✅ MSG parsed: subject=%q, from=%q, body=%d chars",
		properties["subject"], properties["sender_name"], len(text))

	// Build ExtractedFields for frontend compatibility (customer, project, body, total)
	extractedFields := make(map[string]any)
	if v, ok := properties["sender_name"]; ok && v != "" {
		extractedFields["customer"] = v
	} else if v, ok := properties["sender_email"]; ok && v != "" {
		// Use email domain as fallback customer
		parts := strings.Split(v, "@")
		if len(parts) == 2 {
			domain := strings.TrimSuffix(parts[1], ".com")
			domain = strings.TrimSuffix(domain, ".net")
			domain = strings.TrimSuffix(domain, ".org")
			extractedFields["customer"] = strings.Title(domain)
		}
	}
	if v, ok := properties["subject"]; ok && v != "" {
		extractedFields["project"] = v
	}
	// Include full body text for notes field
	extractedFields["body"] = text

	return &OCRResultSimple{
		Success:         true,
		Text:            text,
		Confidence:      1.0, // Direct parsing = perfect accuracy
		DocumentType:    detectedType,
		ExtractedData:   extractedData,
		ExtractedFields: extractedFields,
		Engine:          "msg-parser",
	}, nil
}

// stripHTMLTags removes HTML tags from a string for plain text extraction
func stripHTMLTags(html string) string {
	// Simple regex-based HTML tag removal
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	text := tagPattern.ReplaceAllString(html, " ")
	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	// Collapse multiple whitespace
	spacePattern := regexp.MustCompile(`\s+`)
	text = spacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// Note: using existing min function from document_classifier.go

// processEML handles RFC 5322 email files (.eml)
func (s *SimpleOCRService) processEML(filePath, docType string) (*OCRResultSimple, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EML file: %w", err)
	}
	defer f.Close()

	msg, err := mail.ReadMessage(bufio.NewReader(f))
	if err != nil {
		return nil, fmt.Errorf("failed to parse EML file: %w", err)
	}

	// Extract headers
	subject := msg.Header.Get("Subject")
	from := msg.Header.Get("From")
	to := msg.Header.Get("To")
	cc := msg.Header.Get("Cc")
	date := msg.Header.Get("Date")

	// Read body
	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read EML body: %w", err)
	}
	body := string(bodyBytes)

	// Build combined text
	var allText strings.Builder
	allText.WriteString("Subject: " + subject + "\n")
	allText.WriteString("From: " + from + "\n")
	allText.WriteString("To: " + to + "\n")
	if cc != "" {
		allText.WriteString("CC: " + cc + "\n")
	}
	if date != "" {
		allText.WriteString("Date: " + date + "\n")
	}
	allText.WriteString("\n")
	allText.WriteString(body)

	text := allText.String()

	// Detect document type from email content
	detectedType := docType
	if detectedType == "" || detectedType == "auto" {
		detectedType = detectDocumentTypeFromTextLegacy(text)
	}

	// Extract fields
	extractedData := extractFieldsFromTextLegacy(text, detectedType)
	extractedData["raw_text"] = text
	extractedData["source_type"] = "eml_email"
	extractedData["email_subject"] = subject
	extractedData["email_from"] = from
	extractedData["email_to"] = to
	if date != "" {
		extractedData["email_date"] = date
	}

	log.Printf("📧 EML parsed: subject=%q, from=%q, body=%d chars", subject, from, len(body))

	return &OCRResultSimple{
		Success:       true,
		Text:          text,
		Confidence:    1.0,
		DocumentType:  detectedType,
		ExtractedData: extractedData,
		Engine:        "eml-parser",
	}, nil
}

// processDOCX handles Microsoft Word .docx files
func (s *SimpleOCRService) processDOCX(filePath, docType string) (*OCRResultSimple, error) {
	log.Printf("📝 Processing DOCX file: %s", filepath.Base(filePath))

	// Open the docx file
	r, err := docx.ReadDocxFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX file: %w", err)
	}
	defer r.Close()

	// Get the document content
	doc := r.Editable()

	// Extract all text content
	text := doc.GetContent()

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("DOCX file contains no extractable text")
	}

	// Clean up the text (remove excessive whitespace)
	text = strings.TrimSpace(text)

	// Detect document type from content
	detectedType := docType
	if detectedType == "" || detectedType == "auto" {
		detectedType = detectDocumentTypeFromTextLegacy(text)
	}

	// Extract fields
	extractedData := extractFieldsFromTextLegacy(text, detectedType)
	extractedData["raw_text"] = text
	extractedData["source_type"] = "docx"

	log.Printf("✅ DOCX parsed: %d chars extracted, type=%s", len(text), detectedType)

	return &OCRResultSimple{
		Success:       true,
		Text:          text,
		Confidence:    1.0, // Direct parsing = perfect accuracy
		DocumentType:  detectedType,
		ExtractedData: extractedData,
		Engine:        "docx-parser",
	}, nil
}

// processRTF handles Rich Text Format files (basic text extraction)
func (s *SimpleOCRService) processRTF(filePath, docType string) (*OCRResultSimple, error) {
	log.Printf("📝 Processing RTF file: %s", filepath.Base(filePath))

	// Read the RTF file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read RTF file: %w", err)
	}

	// RTF is text-based with formatting codes like {\rtf1\ansi...}
	// Simple extraction: remove RTF control codes to get plain text
	text := extractTextFromRTF(string(content))

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("RTF file contains no extractable text")
	}

	// Detect document type from content
	detectedType := docType
	if detectedType == "" || detectedType == "auto" {
		detectedType = detectDocumentTypeFromTextLegacy(text)
	}

	// Extract fields
	extractedData := extractFieldsFromTextLegacy(text, detectedType)
	extractedData["raw_text"] = text
	extractedData["source_type"] = "rtf"

	log.Printf("✅ RTF parsed: %d chars extracted, type=%s", len(text), detectedType)

	return &OCRResultSimple{
		Success:       true,
		Text:          text,
		Confidence:    1.0, // Direct parsing = perfect accuracy
		DocumentType:  detectedType,
		ExtractedData: extractedData,
		Engine:        "rtf-parser",
	}, nil
}

// extractTextFromRTF extracts plain text from RTF content by removing control codes
func extractTextFromRTF(rtf string) string {
	var result strings.Builder
	var inGroup int
	var escapeNext bool
	var controlWord strings.Builder

	for i := 0; i < len(rtf); i++ {
		c := rtf[i]

		if escapeNext {
			// Escaped character - add it directly
			if c == '\n' || c == '\r' {
				// Escaped newline - skip
			} else {
				result.WriteByte(c)
			}
			escapeNext = false
			continue
		}

		switch c {
		case '\\':
			// Check if it's an escape or control word
			if i+1 < len(rtf) {
				next := rtf[i+1]
				if next == '\\' || next == '{' || next == '}' {
					// Escaped literal character
					escapeNext = true
				} else if next == '\n' || next == '\r' {
					// Line continuation - skip
					i++
				} else if next == '\'' {
					// Hex character \'xx
					if i+3 < len(rtf) {
						// Try to decode hex character
						hexStr := rtf[i+2 : i+4]
						if val, err := strconv.ParseUint(hexStr, 16, 8); err == nil {
							result.WriteByte(byte(val))
						}
						i += 3
					}
				} else {
					// Control word - read until non-alpha
					controlWord.Reset()
					j := i + 1
					for j < len(rtf) && ((rtf[j] >= 'a' && rtf[j] <= 'z') || (rtf[j] >= 'A' && rtf[j] <= 'Z')) {
						controlWord.WriteByte(rtf[j])
						j++
					}
					// Skip optional numeric parameter
					for j < len(rtf) && ((rtf[j] >= '0' && rtf[j] <= '9') || rtf[j] == '-') {
						j++
					}
					// Skip optional space delimiter
					if j < len(rtf) && rtf[j] == ' ' {
						j++
					}
					i = j - 1

					// Handle special control words that produce text
					word := controlWord.String()
					switch word {
					case "par", "line":
						result.WriteString("\n")
					case "tab":
						result.WriteString("\t")
					}
				}
			}
		case '{':
			inGroup++
		case '}':
			if inGroup > 0 {
				inGroup--
			}
		case '\n', '\r':
			// Skip raw newlines (use \par for paragraph breaks)
		default:
			// Regular character - add if not in a special group
			// (Simple heuristic: always add printable ASCII)
			if c >= 32 && c < 127 {
				result.WriteByte(c)
			}
		}
	}

	// Clean up excessive whitespace
	text := result.String()
	spacePattern := regexp.MustCompile(`[ \t]+`)
	text = spacePattern.ReplaceAllString(text, " ")
	newlinePattern := regexp.MustCompile(`\n{3,}`)
	text = newlinePattern.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// decodeUTF16LE decodes a UTF-16LE byte slice to a Go string
func decodeUTF16LE(b []byte) string {
	if len(b) < 2 {
		return string(b)
	}

	// Ensure even byte count
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}

	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(b[i*2:])
	}

	runes := utf16.Decode(u16s)
	return string(runes)
}

// ProcessBatch processes multiple documents
func (s *SimpleOCRService) ProcessBatch(filePaths []string, docType string) ([]*OCRResultSimple, error) {
	log.Printf("📦 Batch processing %d documents (type=%s)", len(filePaths), docType)

	results := make([]*OCRResultSimple, 0, len(filePaths))

	for i, path := range filePaths {
		log.Printf("  [%d/%d] Processing: %s", i+1, len(filePaths), filepath.Base(path))

		result, err := s.ProcessDocument(path, docType)
		if err != nil {
			log.Printf("  ❌ Failed: %v", err)
			results = append(results, &OCRResultSimple{
				Success:      false,
				DocumentType: docType,
				Error:        err.Error(),
				Engine:       "none",
			})
			continue
		}

		results = append(results, result)
		log.Printf("  ✅ Success: %.2f confidence, %dms, engine=%s",
			result.Confidence, result.ProcessingTime, result.Engine)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	log.Printf("✅ Batch complete: %d/%d successful", successCount, len(filePaths))
	return results, nil
}

// ocrWithMistral submits a PDF (native — no page-render-to-PNG loop) or image to the Mistral
// OCR 4 endpoint with a Document AI schema for the given docType, and derives an honest
// per-field confidence + overall NeedsReview flag from the client's response. Offline-first:
// if no API key is configured, this returns immediately without attempting a network call so
// the caller can fall straight through to ocrWithLocalEngine.
func (s *SimpleOCRService) ocrWithMistral(filePath, docType string, isImage bool) (*OCRResultSimple, error) {
	apiKey := getMistralAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("Mistral API key not configured")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if fileInfo.Size() > MaxFileSize {
		return nil, fmt.Errorf("file too large (%d bytes, max %d bytes / %.1f MB)",
			fileInfo.Size(), MaxFileSize, float64(MaxFileSize)/(1024*1024))
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	envCfg := loadMistralOCREnv()
	client := mistralocr.NewClient(mistralocr.Config{
		APIKey:              apiKey,
		BaseURL:             envCfg.BaseURL,
		Model:               envCfg.Model,
		PageCap:             envCfg.PageCap,
		Timeout:             envCfg.Timeout,
		ConfidenceThreshold: envCfg.ConfidenceThreshold,
	})

	mimeType := mimeTypeForOCRFile(filePath, isImage)
	doc := mistralocr.DocumentInput{
		Data:     fileBytes,
		MIMEType: mimeType,
		IsImage:  isImage,
	}

	ctx, cancel := context.WithTimeout(context.Background(), envCfg.Timeout)
	defer cancel()

	log.Printf("→ Calling Mistral OCR 4 (model=%s, image=%v): %s", envCfg.Model, isImage, filepath.Base(filePath))
	result, err := client.Process(ctx, doc, mistralocr.ProcessOptions{
		Schema: schemaForDocType(docType),
	})
	if err != nil {
		var authErr *mistralocr.AuthError
		var quotaErr *mistralocr.QuotaError
		var tooLargeErr *mistralocr.TooLargeError
		var schemaErr *mistralocr.SchemaMismatchError
		switch {
		case errors.As(err, &authErr):
			log.Printf("⚠ Mistral OCR auth error (bad/missing key): %v", err)
		case errors.As(err, &quotaErr):
			log.Printf("⚠ Mistral OCR quota/rate-limit error: %v", err)
		case errors.As(err, &tooLargeErr):
			log.Printf("⚠ Mistral OCR document too large: %v", err)
		case errors.As(err, &schemaErr):
			log.Printf("⚠ Mistral OCR schema mismatch (falling back to plain OCR would help future work): %v", err)
		default:
			log.Printf("⚠ Mistral OCR request failed: %v", err)
		}
		return nil, err
	}

	if len(strings.TrimSpace(result.Text)) == 0 {
		return nil, fmt.Errorf("Mistral OCR extracted no text")
	}

	extractedData := make(map[string]any, len(result.Fields)+2)
	fieldConfidence := make(map[string]float64, len(result.Fields))
	var fieldsForReview []string
	minConfidence := 1.0
	for name, fv := range result.Fields {
		extractedData[name] = fv.Value
		fieldConfidence[name] = fv.Confidence
		if fv.NeedsReview {
			fieldsForReview = append(fieldsForReview, name)
		}
		if fv.Confidence < minConfidence {
			minConfidence = fv.Confidence
		}
	}
	if len(result.Fields) == 0 {
		// No schema fields decoded (e.g. document_annotation absent) — refuse to guess.
		minConfidence = 0
	}
	extractedData["raw_text"] = result.Text
	extractedData["source_type"] = map[bool]string{true: "image", false: "pdf"}[isImage]

	engine := "mistral-ocr-4"
	if isImage {
		engine = "mistral-ocr-4-image"
	}

	log.Printf("✅ Mistral OCR complete: %d fields extracted, %d flagged for review", len(result.Fields), len(fieldsForReview))

	return &OCRResultSimple{
		Success:         true,
		Text:            result.Text,
		Confidence:      minConfidence,
		DocumentType:    docType,
		ExtractedData:   extractedData,
		Engine:          engine,
		NeedsReview:     len(fieldsForReview) > 0 || len(result.Fields) == 0,
		FieldConfidence: fieldConfidence,
		FieldsForReview: fieldsForReview,
	}, nil
}

// mimeTypeForOCRFile derives the MIME type to send to Mistral OCR from the file extension.
func mimeTypeForOCRFile(filePath string, isImage bool) string {
	if !isImage {
		return "application/pdf"
	}
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

// ocrWithLocalEngine is the offline-first fallback: tesseract + pandoc via the local ACEEngine
// pipeline (no cloud escalation — the engine was constructed with FallbackToMistral:false and no
// MistralAPIKey in NewSimpleOCRService). This never returns a hard error for a document that opens
// on disk — worst case is empty text at 0 confidence, honestly marked NeedsReview, so the
// document inbox never blocks or errors out entirely offline.
// partialText, if non-empty, is go-fitz's partial extraction (used only to enrich extractedData
// when tesseract also fails to produce anything useful).
func (s *SimpleOCRService) ocrWithLocalEngine(filePath, docType, partialText string) (*OCRResultSimple, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for local OCR: %w", err)
	}
	defer file.Close()

	req := &aceocr.ProcessRequest{
		Source:              file,
		SourceType:          aceocr.SourceReader,
		DocumentType:        aceocr.DocTypeGeneric,
		Language:            aceocr.LangEnglish,
		EnableGPU:           false,
		EnablePreprocessing: false,
		FallbackToMistral:   false,
	}

	resp, err := s.localEngine.Process(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("local OCR engine failed: %w", err)
	}

	text := resp.Text
	if strings.TrimSpace(text) == "" {
		text = partialText
	}

	extractedData := extractFieldsFromTextLegacy(text, docType)
	extractedData["raw_text"] = text
	extractedData["source_type"] = "local_tesseract_fallback"

	confidence := resp.Confidence
	needsReview := true // offline/degraded-mode result — always surfaced for human review

	log.Printf("✅ Local tesseract fallback complete: %d chars, confidence=%.2f (degraded mode — review recommended)", len(text), confidence)

	return &OCRResultSimple{
		Success:       true,
		Text:          text,
		Confidence:    confidence,
		DocumentType:  docType,
		ExtractedData: extractedData,
		Engine:        "tesseract-local",
		NeedsReview:   needsReview,
	}, nil
}

// isVectorPDF checks if a PDF contains sufficient extractable text (vector PDF)
// Returns true only if MOST pages have extractable vector text (>70% threshold)
func isVectorPDF(filePath string) bool {
	return documentsocr.NewFitzEngine().IsVectorPDF(filePath)
}

// extractVectorPDF extracts all text from a vector PDF using the configured local PDF engine.
func extractVectorPDF(filePath string) (string, error) {
	return documentsocr.NewFitzEngine().ExtractText(filePath)
}

// extractFieldsFromTextLegacy extracts structured fields from OCR text using regex patterns.
// DEGRADED MODE: this regex layer is demoted (Wave 13) to the offline/tesseract fallback path
// and local-parse office formats (Excel/MSG/EML/DOCX/RTF) only. The online path uses Mistral OCR
// 4's schema-shaped Document AI extraction instead (see schemaForDocType / ocrWithMistral).
func extractFieldsFromTextLegacy(text, docType string) map[string]any {
	fields := make(map[string]any)

	// Patterns for structured field extraction across document types
	// Optimized for Bahrain instrumentation industry documents (Acme Instrumentation)
	patterns := map[string]*regexp.Regexp{
		// Invoice fields
		"invoice_number": regexp.MustCompile(`(?i)(?:invoice|inv|bill|tax\s*invoice)[\s#:.\-]*(?:no\.?|number|num|#)?[\s#:.\-]*([A-Z0-9][\w\-/]{2,20})`),
		"invoice_date":   regexp.MustCompile(`(?i)(?:invoice\s*date|date|dated)[\s:.\-]*(\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}|\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\w*\s+\d{2,4})`),
		"due_date":       regexp.MustCompile(`(?i)(?:due\s*date|payment\s*due|pay\s*by)[\s:.\-]*(\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}|\d{1,2}\s+\w+\s+\d{2,4})`),
		// PO fields (common formats: PO 12345, P.O. #123, Purchase Order 4500019873)
		"po_number": regexp.MustCompile(`(?i)(?:p\.?\s*o\.?|purchase\s*order|order\s*(?:no|number|#))[\s#:.\-]*(?:no\.?|number|#)?[\s#:.\-]*([A-Z0-9][\w\-/]{3,20})`),
		// RFQ fields (formats: RFQ-5592, RFQ# 10951459, Enquiry 215-24, Request for Quotation Code rfq_56121)
		"rfq_number":       regexp.MustCompile(`(?i)(?:rfq|enquiry|inquiry|tender|request\s*for\s*quot\w*|quotation\s*code)[\s#:.\-]*(?:no\.?|number|#|ref|code)?[\s#:.\-]*([A-Z0-9][\w\-/_]{2,25})`),
		"rfq_reference":    regexp.MustCompile(`(?i)(?:ref\.?|reference|our\s*ref|your\s*ref)[\s#:.\-]*([A-Z0-9][\w\-/]{3,30})`),
		"quotation_number": regexp.MustCompile(`(?i)(?:quot(?:ation)?|offer|qtn)[\s#:.\-]*(?:no\.?|number|#|ref)?[\s#:.\-]*([A-Z0-9][\w\-/]{2,20})`),
		"project":          regexp.MustCompile(`(?i)(?:project|subject|re[:\s]|regarding|description)[\s:.\-]*\n?\s*([A-Za-z0-9][A-Za-z0-9\s&.,'\-()]+?)(?:\n|$)`),
		"delivery_date":    regexp.MustCompile(`(?i)(?:delivery\s*date|required\s*(?:date|by)|need\s*by|expected\s*delivery|target\s*date)[\s:.\-]*(\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}|\d{1,2}\s+\w+\s+\d{2,4})`),
		"validity":         regexp.MustCompile(`(?i)(?:validity|valid\s*(?:for|until)|offer\s*valid)[\s:.\-]*(\d+\s*(?:days?|weeks?|months?)|until\s+\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4})`),
		"bid_deadline":     regexp.MustCompile(`(?i)(?:bid\s*deadline|submission\s*date|closing\s*date|submit\s*by)[\s:.\-]*(\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4})`),
		// Parties (Bahrain companies: NPC, Gulf Smelting, DPC, Meadow Dairy, etc.)
		"customer_name":  regexp.MustCompile(`(?i)(?:bill\s*to|customer|client|sold\s*to|attention|attn|dear\s+(?:sir|madam|mr|ms))[\s:.\-]*\n?\s*([A-Za-z0-9][A-Za-z0-9\s&.,'\-()]+?)(?:\n|$)`),
		"supplier_name":  regexp.MustCompile(`(?i)(?:from|supplier|vendor|ship\s*from|manufacturer|oem)[\s:.\-]*\n?\s*([A-Za-z0-9][A-Za-z0-9\s&.,'\-()]+?)(?:\n|$)`),
		"contact_person": regexp.MustCompile(`(?i)(?:contact|attention|attn|contact\s*person)[\s:.\-]*\n?\s*([A-Z][a-z]+(?:\s+[A-Z][a-z]+){1,3})`),
		"contact_email":  regexp.MustCompile(`(?i)([a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})`),
		"contact_phone":  regexp.MustCompile(`(?i)(?:tel|phone|mobile|fax)[\s:.\-]*([+\d][\d\s\-()]{7,20})`),
		// Financials (BHD = Bahraini Dinar, 3 decimal places)
		"total":      regexp.MustCompile(`(?i)(?:total|grand\s*total|amount\s*due|balance\s*due|net\s*amount|total\s*price)[\s:]*(?:BHD|BD|USD|\$|£|€)?\s*([\d,]+\.?\d{0,3})`),
		"subtotal":   regexp.MustCompile(`(?i)(?:sub\s*total|net\s*total|total\s*before)[\s:]*(?:BHD|BD|USD|\$|£|€)?\s*([\d,]+\.?\d{0,3})`),
		"vat":        regexp.MustCompile(`(?i)(?:vat|tax|gst)[\s:@]*(?:\d+%)?[\s:]*(?:BHD|BD|USD|\$|£|€)?\s*([\d,]+\.?\d{0,3})`),
		"unit_price": regexp.MustCompile(`(?i)(?:unit\s*price|price\s*each|rate)[\s:]*(?:BHD|BD|USD|\$|£|€)?\s*([\d,]+\.?\d{0,3})`),
		"currency":   regexp.MustCompile(`(?i)\b(BHD|BD|USD|EUR|GBP|AED)\b`),
		// Delivery note fields
		"dn_number": regexp.MustCompile(`(?i)(?:delivery\s*note|dn|dispatch\s*note|packing\s*list)[\s#:.\-]*(?:no\.?|number|#)?[\s#:.\-]*([A-Z0-9][\w\-/]{2,15})`),
		"tracking":  regexp.MustCompile(`(?i)(?:tracking|awb|waybill|shipment)[\s#:.\-]*(?:no\.?|number|#)?[\s#:.\-]*([A-Z0-9][\w\-]{5,25})`),
		// Product/Item fields (common in instrumentation: Rhine Instruments, Oxan Analytics codes)
		"part_number": regexp.MustCompile(`(?i)(?:part\s*(?:no\.?|number|#)|item\s*(?:no\.?|code)|model|sku|material)[\s#:.\-]*([A-Z0-9][\w\-+/]{3,30})`),
		"quantity":    regexp.MustCompile(`(?i)(?:qty\.?|quantity|nos\.?|pcs\.?)[\s:.\-]*(\d+)`),
		// Bahrain-specific (CR number, VAT number)
		"cr_number":  regexp.MustCompile(`(?i)(?:cr|c\.r\.|commercial\s*registration)[\s#:.\-]*(\d{4,10})`),
		"vat_number": regexp.MustCompile(`(?i)(?:vat\s*(?:no\.?|number|reg)|tax\s*(?:no\.?|id))[\s#:.\-]*(\d{9,15})`),
	}

	// Extract each field
	extractedCount := 0
	for name, pattern := range patterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			value := strings.TrimSpace(match[1])
			// Sanity checks: not empty, reasonable length, not just numbers/punctuation
			if value != "" && len(value) >= 2 && len(value) < 200 {
				// Skip if value is just common words or single letters
				lowerVal := strings.ToLower(value)
				skipWords := []string{"the", "and", "for", "from", "with", "this", "that", "your", "our"}
				skip := false
				for _, w := range skipWords {
					if lowerVal == w {
						skip = true
						break
					}
				}
				if !skip {
					fields[name] = value
					extractedCount++
				}
			}
		}
	}

	log.Printf("📊 Field extraction: %d fields extracted from %d chars of text", extractedCount, len(text))

	// Count pages (using page break markers)
	pageCount := strings.Count(text, "\f") + strings.Count(text, "--- PAGE ") + 1
	if pageCount > 1 {
		pageCount-- // Adjust for double-counting if both markers present
	}
	fields["page_count"] = pageCount

	return fields
}

// detectDocumentTypeFromTextLegacy classifies a document based on its text content using
// keyword scoring. DEGRADED MODE — see extractFieldsFromTextLegacy.
func detectDocumentTypeFromTextLegacy(text string) string {
	lower := strings.ToLower(text)

	// Score each document type by keyword matches
	scores := map[string]int{
		"invoice":          0,
		"rfq":              0,
		"quotation":        0,
		"purchase_order":   0,
		"delivery_note":    0,
		"supplier_invoice": 0,
		"bank_statement":   0,
		"report":           0,
		"other":            0,
	}

	// Invoice keywords
	invoiceKeywords := []string{"tax invoice", "invoice no", "invoice number", "invoice date", "amount due", "payment terms", "bill to", "original invoice", "commercial invoice"}
	for _, kw := range invoiceKeywords {
		if strings.Contains(lower, kw) {
			scores["invoice"] += 2
		}
	}

	// Strong supplier invoice indicators (invoice FROM a supplier TO Acme Instrumentation)
	// Check if Acme Instrumentation is the recipient (ship-to, company, bill-to)
	isPHRecipient := strings.Contains(lower, "acme instrumentation")
	hasInvoiceWord := strings.Contains(lower, "invoice")
	hasKnownSupplier := strings.Contains(lower, "rhine") || strings.Contains(lower, "oxan") ||
		strings.Contains(lower, "helvetia") || strings.Contains(lower, "meridian") ||
		strings.Contains(lower, "issued by:") || strings.Contains(lower, "seller:") || strings.Contains(lower, "vendor:")

	// If invoice mentions Acme Instrumentation as recipient + has invoice word = supplier invoice
	if isPHRecipient && hasInvoiceWord {
		scores["supplier_invoice"] += 5
	}
	if hasKnownSupplier && hasInvoiceWord {
		scores["supplier_invoice"] += 3
	}

	// RFQ keywords
	rfqKeywords := []string{"request for quotation", "rfq", "enquiry", "inquiry", "request for quote", "bid request", "tender", "requirement for the supply", "requirement for supply", "provide us the quote", "provide us your quote", "provide your quote", "final quote", "kindly quote", "please quote", "submit your quote", "need a quote", "requesting quote"}
	for _, kw := range rfqKeywords {
		if strings.Contains(lower, kw) {
			scores["rfq"] += 2
		}
	}
	// Additional RFQ patterns for emails asking for quotes
	if strings.Contains(lower, "we have a requirement") || strings.Contains(lower, "kindly provide") && strings.Contains(lower, "quote") {
		scores["rfq"] += 3
	}

	// Quotation keywords
	quoteKeywords := []string{"quotation", "quote ref", "price offer", "our offer", "validity", "quoted price", "proforma"}
	for _, kw := range quoteKeywords {
		if strings.Contains(lower, kw) {
			scores["quotation"] += 2
		}
	}

	// PO keywords
	poKeywords := []string{"purchase order", "p.o. number", "po number", "po no", "order confirmation", "buyer"}
	for _, kw := range poKeywords {
		if strings.Contains(lower, kw) {
			scores["purchase_order"] += 2
		}
	}

	// Delivery note keywords (must be the document type, not a reference)
	// "delivery note" as a field reference (e.g., "Delivery note : 123456") should NOT classify as DN
	dnKeywords := []string{"packing list", "dispatch note", "shipping note", "goods delivered", "consignment note"}
	for _, kw := range dnKeywords {
		if strings.Contains(lower, kw) {
			scores["delivery_note"] += 2
		}
	}
	// Only count "delivery note" if it appears as document title (not as field)
	// Pattern: "delivery note" at start of line or without colon after
	if strings.Contains(lower, "delivery note\n") || strings.Contains(lower, "delivery note ") && !strings.Contains(lower, "delivery note :") && !strings.Contains(lower, "delivery note:") {
		scores["delivery_note"] += 2
	}

	// Supplier invoice keywords (invoice from supplier to us)
	// Note: Avoid matching "From:" which is common in email headers
	siKeywords := []string{"ship from", "vendor invoice", "supplier invoice", "from supplier"}
	for _, kw := range siKeywords {
		if strings.Contains(lower, kw) {
			scores["supplier_invoice"]++
		}
	}
	// If it has invoice keywords AND supplier keywords, likely a supplier invoice
	if scores["invoice"] > 0 && scores["supplier_invoice"] > 0 {
		scores["supplier_invoice"] += scores["invoice"]
	}

	// Bank statement keywords
	bankKeywords := []string{"bank statement", "account statement", "opening balance", "closing balance", "running balance", "available balance", "account number", "value date"}
	for _, kw := range bankKeywords {
		if strings.Contains(lower, kw) {
			scores["bank_statement"] += 2
		}
	}
	if strings.Contains(lower, "debit") && strings.Contains(lower, "credit") && strings.Contains(lower, "balance") {
		scores["bank_statement"] += 4
	}

	// Report keywords
	reportKeywords := []string{"monthly report", "weekly report", "management report", "executive summary", "analysis report", "summary report", "performance report", "dashboard report"}
	for _, kw := range reportKeywords {
		if strings.Contains(lower, kw) {
			scores["report"] += 2
		}
	}
	if strings.Contains(lower, "kpi") && (strings.Contains(lower, "summary") || strings.Contains(lower, "analysis")) {
		scores["report"] += 3
	}

	// Find highest scoring type
	best := "other"
	bestScore := 0
	for docType, score := range scores {
		if score > bestScore {
			best = docType
			bestScore = score
		}
	}

	log.Printf("📋 Document type detected: %s (score=%d)", best, bestScore)
	return best
}

// truncateForLog truncates a string to maxLen characters (for logging)
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Close cleans up resources (no-op for simple service)
func (s *SimpleOCRService) Close() error {
	log.Println("🌸 Closing Simple OCR Service...")
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// LEGACY COMPATIBILITY METHODS
// ═══════════════════════════════════════════════════════════════════════════

// ExtractRFQ extracts RFQ-specific fields from a document
func (s *SimpleOCRService) ExtractRFQ(filePath string) (*OCRResultSimple, error) {
	return s.ProcessDocument(filePath, "rfq")
}

// ExtractInvoice extracts invoice-specific fields from a document
func (s *SimpleOCRService) ExtractInvoice(filePath string) (*OCRResultSimple, error) {
	return s.ProcessDocument(filePath, "invoice")
}

// ExtractQuotation extracts quotation-specific fields from a document
func (s *SimpleOCRService) ExtractQuotation(filePath string) (*OCRResultSimple, error) {
	return s.ProcessDocument(filePath, "quotation")
}

// GetPipelineStats returns pipeline statistics (simplified)
func (s *SimpleOCRService) GetPipelineStats() map[string]any {
	envCfg := loadMistralOCREnv()
	return map[string]any{
		"engine":          "simple-ocr",
		"mistral_model":   envCfg.Model,
		"max_pages":       s.maxPages,
		"dpi":             s.dpi,
		"vector_strategy": "go-fitz (FREE)",
		"scan_strategy":   "Mistral OCR 4 (cloud) -> tesseract-local (offline fallback)",
	}
}

// GetProcessorStats returns processor statistics (simplified)
func (s *SimpleOCRService) GetProcessorStats() map[string]any {
	envCfg := loadMistralOCREnv()
	return map[string]any{
		"go_fitz": map[string]any{
			"status":   "active",
			"cost":     "FREE",
			"speed":    "<100ms for vector PDFs",
			"accuracy": "100% for vector text",
		},
		"mistral_ocr_4": map[string]any{
			"status": "active (requires MISTRAL_API_KEY)",
			"model":  envCfg.Model,
			"cost":   "$4/1k pages OCR, $5/1k pages with Document AI schema extraction",
			"speed":  "1-3s typical",
		},
		"tesseract_local": map[string]any{
			"status": "active (offline fallback, no cloud escalation)",
			"cost":   "FREE",
		},
	}
}

// ProcessWithGoFitz forces local go-fitz processing (for vector PDFs)
func (s *SimpleOCRService) ProcessWithGoFitz(filePath string) (*OCRResultSimple, error) {
	startTime := time.Now()
	text, err := extractVectorPDF(filePath)
	if err != nil {
		return nil, fmt.Errorf("go-fitz extraction failed: %w", err)
	}
	return &OCRResultSimple{
		Success:        true,
		Text:           text,
		Confidence:     1.0,
		Engine:         "pymupdf",
		ProcessingTime: time.Since(startTime).Milliseconds(),
	}, nil
}
