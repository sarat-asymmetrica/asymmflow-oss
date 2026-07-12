package intake

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ph_holdings_app/pkg/kernel/text"
)

// InboxProcessResultInput mirrors the existing main.InboxProcessResult shape
// without importing the runtime-facing main package into the pure kernel.
type InboxProcessResultInput struct {
	DocumentID               string            `json:"document_id"`
	DetectedType             string            `json:"detected_type"`
	ClassificationConfidence float64           `json:"classification_confidence"`
	ExtractedText            string            `json:"extracted_text"`
	Entities                 map[string]string `json:"entities"`
	SuggestedActions         []string          `json:"suggested_actions"`
	ProcessedAt              time.Time         `json:"processed_at"`
	NeedsReview              bool              `json:"needs_review"`
	OcrConfidence            float64           `json:"ocr_confidence"`
	Error                    string            `json:"error,omitempty"`
}

// InboxDocumentInput mirrors the existing main.InboxDocument persistence shape.
type InboxDocumentInput struct {
	ID               uint              `json:"id"`
	DocumentID       string            `json:"document_id"`
	FileName         string            `json:"file_name"`
	FilePath         string            `json:"file_path"`
	DocumentType     string            `json:"document_type"`
	Status           string            `json:"status"`
	Confidence       float64           `json:"confidence"`
	ExtractedData    map[string]string `json:"extracted_data"`
	SuggestedActions []string          `json:"suggested_actions"`
	ProcessedAt      time.Time         `json:"processed_at"`
	CreatedAt        time.Time         `json:"created_at"`
}

type OCRExtractionInput struct {
	SourceID        string         `json:"source_id,omitempty"`
	FileName        string         `json:"file_name,omitempty"`
	FilePath        string         `json:"file_path,omitempty"`
	DocumentType    string         `json:"document_type,omitempty"`
	Text            string         `json:"text,omitempty"`
	Confidence      float64        `json:"confidence,omitempty"`
	ExtractedData   map[string]any `json:"extracted_data,omitempty"`
	ExtractedFields map[string]any `json:"extracted_fields,omitempty"`
	Engine          string         `json:"engine,omitempty"`
	Success         bool           `json:"success,omitempty"`
	Error           string         `json:"error,omitempty"`
	ProcessedAt     time.Time      `json:"processed_at,omitempty"`
}

func FromInboxProcessResult(input InboxProcessResultInput, opts ...Options) Candidate {
	return normalizeCandidate(NormalizeInboxProcessResult(input), mergeOptions(opts...))
}

func FromInboxDocument(input InboxDocumentInput, opts ...Options) Candidate {
	return normalizeCandidate(NormalizeInboxDocument(input), mergeOptions(opts...))
}

func FromOCRExtraction(input OCRExtractionInput, opts ...Options) Candidate {
	fields := input.ExtractedFields
	if len(fields) == 0 {
		fields = input.ExtractedData
	}
	payload := map[string]any{
		"document_id":      input.SourceID,
		"file_name":        input.FileName,
		"file_path":        input.FilePath,
		"document_type":    input.DocumentType,
		"text":             input.Text,
		"confidence":       input.Confidence,
		"extracted_fields": fields,
		"engine":           input.Engine,
		"success":          input.Success,
		"error":            input.Error,
		"processed_at":     input.ProcessedAt,
	}
	return normalizeCandidate(NormalizeExtractionMap(payload), mergeOptions(opts...))
}

func ReviewStatusFromInboxStatus(status string, confidence float64) ReviewStatus {
	return normalizeInboxReviewStatus(status, confidence)
}

func NormalizeReviewStatus(status string, confidence float64) ReviewStatus {
	return normalizeInboxReviewStatus(status, confidence)
}

// NormalizeInboxProcessResult converts the runtime inbox process response shape
// into a canonical intake candidate without touching runtime authority.
func NormalizeInboxProcessResult(input InboxProcessResultInput) Candidate {
	confidence := NormalizeConfidence(input.ClassificationConfidence)
	sourceID := text.FirstNonEmpty(input.DocumentID, "inbox-process-result")
	sourceKind := SourceKindFromPath(sourceID)
	businessType := businessObjectType(input.DetectedType)

	candidate := Candidate{
		ID:                 sourceID,
		Source:             SourceRef{ID: sourceID, Label: sourceLabel(sourceID), Path: input.DocumentID, Kind: sourceKind, ProcessedAt: timePtr(input.ProcessedAt)},
		SourceKind:         sourceKind,
		BusinessObjectType: businessType,
		Classification: Classification{
			Type:       input.DetectedType,
			Confidence: confidence,
		},
		ExtractedFields: fieldsFromStringMap(input.Entities, "entities", confidence),
		SuggestedLinks:  suggestedLinksFromActions(input.SuggestedActions, businessType),
		ReviewStatus:    ReviewStatusForConfidence(confidence, input.NeedsReview || strings.TrimSpace(input.Error) != ""),
		AuditRefs:       []AuditRef{{Type: "runtime_inbox_process", SourceID: sourceID, Summary: "Normalized from InboxProcessResult."}},
		Confidence:      confidence,
		Warnings:        normalizeWarnings(input.Error, input.OcrConfidence),
	}
	if strings.TrimSpace(input.ExtractedText) != "" {
		candidate.ExtractedFields = append(candidate.ExtractedFields, ExtractedField{
			Name:       "raw_text",
			Label:      "Raw Text",
			Value:      input.ExtractedText,
			Status:     FieldStatusExtracted,
			Confidence: NormalizeConfidence(input.OcrConfidence),
			Source:     "extracted_text",
		})
	}
	return normalizeCandidate(candidate, Options{})
}

// NormalizeInboxDocument converts the persisted inbox record shape into a candidate.
func NormalizeInboxDocument(input InboxDocumentInput) Candidate {
	confidence := NormalizeConfidence(input.Confidence)
	sourceID := text.FirstNonEmpty(input.DocumentID, fmt.Sprint(input.ID))
	sourceKind := SourceKindFromPath(text.FirstNonEmpty(input.FilePath, input.FileName))
	if sourceKind == SourceKindOther {
		sourceKind = SourceKindInboxRecord
	}
	businessType := businessObjectType(input.DocumentType)

	return normalizeCandidate(Candidate{
		ID:                 sourceID,
		Source:             SourceRef{ID: sourceID, Label: sourceLabel(text.FirstNonEmpty(input.FileName, input.FilePath, sourceID)), Path: input.FilePath, Kind: sourceKind, ProcessedAt: timePtr(input.ProcessedAt)},
		SourceKind:         sourceKind,
		BusinessObjectType: businessType,
		Classification: Classification{
			Type:       input.DocumentType,
			Confidence: confidence,
		},
		ExtractedFields: fieldsFromStringMap(input.ExtractedData, "inbox_document", confidence),
		SuggestedLinks:  suggestedLinksFromActions(input.SuggestedActions, businessType),
		ReviewStatus:    normalizeInboxReviewStatus(input.Status, confidence),
		AuditRefs:       []AuditRef{{Type: "inbox_document", SourceID: sourceID, Summary: "Normalized from stored InboxDocument."}},
		Confidence:      confidence,
		Warnings:        normalizeWarnings("", confidence),
	}, Options{})
}

// NormalizeExtractionMap accepts OCRResultSimple-like maps, OCRDocument-like maps,
// and flat extraction maps produced by existing services.
func NormalizeExtractionMap(input map[string]any) Candidate {
	confidence := NormalizeConfidence(firstFloat(input, "confidence", "classification_confidence", "ocr_confidence"))
	docType := firstString(input, "document_type", "documentType", "detected_type", "type")
	id := firstString(input, "id", "document_id", "documentID")
	fileName := firstString(input, "file_name", "fileName", "name")
	filePath := firstString(input, "file_path", "filePath", "path")
	sourceKind := NormalizeSourceKind(firstString(input, "source_kind", "sourceKind", "source_type", "sourceType"))
	if sourceKind == SourceKindOther {
		sourceKind = SourceKindFromPath(text.FirstNonEmpty(filePath, fileName))
	}

	fieldsMap := firstMap(input, "extracted_data", "extractedData", "extracted_fields", "extractedFields", "entities")
	if len(fieldsMap) == 0 {
		fieldsMap = copyExtractionFields(input)
	}
	needsReview := firstBool(input, "needs_review", "needsReview")
	errorText := firstString(input, "error")
	reviewStatus := normalizeInboxReviewStatus(firstString(input, "review_status", "reviewStatus", "status"), confidence)
	if needsReview || strings.TrimSpace(errorText) != "" {
		reviewStatus = ReviewStatusNeedsReview
	}

	sourceID := text.FirstNonEmpty(id, filePath, fileName, "ocr-extraction")
	businessType := businessObjectType(docType)
	return normalizeCandidate(Candidate{
		ID:                 sourceID,
		Source:             SourceRef{ID: sourceID, Label: sourceLabel(text.FirstNonEmpty(fileName, filePath, sourceID)), Path: filePath, Kind: sourceKind, ProcessedAt: firstTimePtr(input, "processed_at", "processedAt", "created_at", "createdAt")},
		SourceKind:         sourceKind,
		BusinessObjectType: businessType,
		Classification: Classification{
			Type:       docType,
			Method:     firstString(input, "method", "engine"),
			Confidence: confidence,
		},
		ExtractedFields: fieldsFromExtractionMap(fieldsMap, "extracted_data", confidence),
		SuggestedLinks:  suggestedLinksFromActions(firstStringSlice(input, "suggested_links", "suggestedLinks", "suggested_actions", "suggestedActions"), businessType),
		ReviewStatus:    reviewStatus,
		AuditRefs:       []AuditRef{{Type: string(sourceKind), SourceID: sourceID, Summary: "Normalized from OCR/extraction map."}},
		Confidence:      confidence,
		Warnings:        normalizeWarnings(errorText, 0),
	}, Options{})
}

func NormalizeConfidence(value float64) float64 {
	if value > 1 && value <= 100 {
		value = value / 100
	}
	return clampConfidence(value)
}

func NormalizeFieldStatus(status string, confidence float64, value any) FieldStatus {
	switch normalizeKey(status) {
	case "extracted", "present":
		return FieldStatusExtracted
	case "missing", "empty":
		return FieldStatusMissing
	case "inferred", "derived":
		return FieldStatusInferred
	case "needs_confirmation", "needs_review", "uncertain":
		return FieldStatusNeedsConfirmation
	case "corrected", "edited":
		return FieldStatusCorrected
	}
	return FieldStatusForValue(strings.TrimSpace(fmt.Sprint(value)), confidence)
}

func NormalizeSourceKind(kind string) SourceKind {
	switch normalizeKey(kind) {
	case "message", "chat", "whatsapp":
		return SourceKindMessage
	case "email", "mail", "eml", "msg":
		return SourceKindEmail
	case "pdf":
		return SourceKindPDF
	case "scan", "scanned_pdf":
		return SourceKindScan
	case "screenshot", "image", "png", "jpg", "jpeg":
		return SourceKindScreenshot
	case "excel", "spreadsheet", "xlsx", "xls", "csv":
		return SourceKindExcel
	case "folder", "directory":
		return SourceKindFolder
	case "inbox_record", "inbox":
		return SourceKindInboxRecord
	default:
		return SourceKindOther
	}
}

func fieldsFromExtractionMap(values map[string]any, source string, fallbackConfidence float64) []ExtractedField {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fields := make([]ExtractedField, 0, len(keys))
	for _, key := range keys {
		if strings.EqualFold(key, "raw_text") {
			continue
		}
		value := values[key]
		statusText := ""
		confidence := fallbackConfidence
		if nested, ok := value.(map[string]any); ok {
			statusText = firstString(nested, "status", "field_status", "fieldStatus")
			if nestedConfidence := firstFloat(nested, "confidence"); nestedConfidence > 0 {
				confidence = NormalizeConfidence(nestedConfidence)
			}
			if nestedValue, ok := firstExisting(nested, "value", "text", "raw"); ok {
				value = nestedValue
			}
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if value == nil {
			text = ""
		}
		fields = append(fields, ExtractedField{
			Name:       normalizeKey(key),
			Label:      titleFromKey(key),
			Value:      text,
			Status:     NormalizeFieldStatus(statusText, confidence, text),
			Confidence: NormalizeConfidence(confidence),
			Source:     source,
		})
	}
	return sortedFields(fields)
}

func normalizeInboxReviewStatus(status string, confidence float64) ReviewStatus {
	switch normalizeKey(status) {
	case "new", "ready", "open":
		return ReviewStatusForConfidence(confidence, false)
	case "needsreview", "needs_review", "review", "manual_review":
		return ReviewStatusNeedsReview
	case "corrected", "edited":
		return ReviewStatusCorrected
	case "linked", "processed", "complete", "completed":
		return ReviewStatusLinked
	case "rejected", "reject":
		return ReviewStatusRejected
	case "archived", "archive":
		return ReviewStatusArchived
	default:
		return ReviewStatusForConfidence(confidence, false)
	}
}

func normalizeWarnings(errorText string, ocrConfidence float64) []string {
	var warnings []string
	if strings.TrimSpace(errorText) != "" {
		warnings = append(warnings, strings.TrimSpace(errorText))
	}
	if ocrConfidence > 0 && NormalizeConfidence(ocrConfidence) < 0.75 {
		warnings = append(warnings, "ocr confidence below review threshold")
	}
	return uniqueStrings(warnings)
}

func sourceLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Intake source"
	}
	if base := filepath.Base(value); base != "." && base != string(filepath.Separator) && base != "" {
		return base
	}
	return value
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func firstExisting(input map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := input[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func firstString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case fmt.Stringer:
			if strings.TrimSpace(typed.String()) != "" {
				return strings.TrimSpace(typed.String())
			}
		case float64, float32, int, int64, uint, uint64:
			return fmt.Sprint(typed)
		}
	}
	return ""
}

func firstFloat(input map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case float32:
			return float64(typed)
		case int:
			return float64(typed)
		case int64:
			return float64(typed)
		case uint:
			return float64(typed)
		case uint64:
			return float64(typed)
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(typed), "%"), 64)
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func firstBool(input map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			parsed, err := strconv.ParseBool(typed)
			return err == nil && parsed
		}
	}
	return false
}

func firstTimePtr(input map[string]any, keys ...string) *time.Time {
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case time.Time:
			return timePtr(typed)
		case string:
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
				parsed, err := time.Parse(layout, typed)
				if err == nil {
					return &parsed
				}
			}
		}
	}
	return nil
}

func firstMap(input map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		value, ok := input[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			return typed
		case map[string]string:
			converted := make(map[string]any, len(typed))
			for k, v := range typed {
				converted[k] = v
			}
			return converted
		}
	}
	return nil
}

func firstStringSlice(input map[string]any, keys ...string) []string {
	raw, ok := firstExisting(input, keys...)
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			switch value := item.(type) {
			case string:
				values = append(values, value)
			case map[string]any:
				values = append(values, text.FirstNonEmpty(firstString(value, "label"), firstString(value, "action")))
			}
		}
		return values
	default:
		return nil
	}
}

func copyExtractionFields(input map[string]any) map[string]any {
	control := map[string]bool{
		"id": true, "document_id": true, "documentID": true, "file_name": true, "fileName": true,
		"file_path": true, "filePath": true, "path": true, "name": true, "document_type": true,
		"documentType": true, "detected_type": true, "type": true, "text": true, "extracted_text": true,
		"extractedText": true, "raw_text": true, "rawText": true, "confidence": true,
		"classification_confidence": true, "ocr_confidence": true, "source_kind": true, "sourceKind": true,
		"source_type": true, "sourceType": true, "review_status": true, "reviewStatus": true,
		"status": true, "needs_review": true, "needsReview": true, "suggested_links": true,
		"suggestedLinks": true, "suggested_actions": true, "suggestedActions": true, "error": true,
		"engine": true, "method": true, "processed_at": true, "processedAt": true, "created_at": true,
		"createdAt": true,
	}
	fields := make(map[string]any)
	for key, value := range input {
		if !control[key] {
			fields[key] = value
		}
	}
	return fields
}

func mergeOptions(opts ...Options) Options {
	if len(opts) == 0 {
		return Options{}
	}
	return opts[0]
}
