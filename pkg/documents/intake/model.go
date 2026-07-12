// Package intake defines the pure Business Memory Intake contract.
package intake

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type SourceKind string

const (
	SourceKindMessage     SourceKind = "message"
	SourceKindEmail       SourceKind = "email"
	SourceKindPDF         SourceKind = "pdf"
	SourceKindScan        SourceKind = "scan"
	SourceKindScreenshot  SourceKind = "screenshot"
	SourceKindExcel       SourceKind = "excel"
	SourceKindFolder      SourceKind = "folder"
	SourceKindInboxRecord SourceKind = "inbox_record"
	SourceKindOther       SourceKind = "other"
)

type ReviewStatus string

const (
	ReviewStatusNew         ReviewStatus = "new"
	ReviewStatusNeedsReview ReviewStatus = "needs_review"
	ReviewStatusCorrected   ReviewStatus = "corrected"
	ReviewStatusLinked      ReviewStatus = "linked"
	ReviewStatusRejected    ReviewStatus = "rejected"
	ReviewStatusArchived    ReviewStatus = "archived"
)

type FieldStatus string

const (
	FieldStatusExtracted         FieldStatus = "extracted"
	FieldStatusMissing           FieldStatus = "missing"
	FieldStatusInferred          FieldStatus = "inferred"
	FieldStatusNeedsConfirmation FieldStatus = "needs_confirmation"
	FieldStatusCorrected         FieldStatus = "corrected"
)

type Candidate struct {
	ID                 string           `json:"id"`
	Source             SourceRef        `json:"source"`
	SourceKind         SourceKind       `json:"source_kind"`
	BusinessObjectType string           `json:"business_object_type"`
	Classification     Classification   `json:"classification"`
	ExtractedFields    []ExtractedField `json:"extracted_fields"`
	SuggestedLinks     []SuggestedLink  `json:"suggested_links"`
	ReviewStatus       ReviewStatus     `json:"review_status"`
	AuditRefs          []AuditRef       `json:"audit_refs"`
	Confidence         float64          `json:"confidence"`
	Warnings           []string         `json:"warnings,omitempty"`
}

type SourceRef struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Path        string     `json:"path,omitempty"`
	Kind        SourceKind `json:"kind"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

type Classification struct {
	Type       string   `json:"type"`
	Method     string   `json:"method,omitempty"`
	RouteTo    string   `json:"route_to,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Confidence float64  `json:"confidence"`
}

type ExtractedField struct {
	Name       string      `json:"name"`
	Label      string      `json:"label"`
	Value      string      `json:"value,omitempty"`
	Status     FieldStatus `json:"status"`
	Confidence float64     `json:"confidence,omitempty"`
	Source     string      `json:"source,omitempty"`
}

type SuggestedLink struct {
	ID                           string `json:"id"`
	Label                        string `json:"label"`
	Reason                       string `json:"reason"`
	BusinessObjectType           string `json:"business_object_type"`
	RequiredDeterministicService string `json:"required_deterministic_service"`
}

type AuditRef struct {
	Type      string `json:"type"`
	SourceID  string `json:"source_id"`
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp,omitempty"`
}

type Options struct {
	Now func() time.Time
}

func normalizeCandidate(candidate Candidate, opts Options) Candidate {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if candidate.SourceKind == "" {
		candidate.SourceKind = candidate.Source.Kind
	}
	if candidate.SourceKind == "" {
		candidate.SourceKind = SourceKindOther
	}
	candidate.Source.Kind = candidate.SourceKind
	if candidate.ID == "" {
		candidate.ID = buildID(candidate.SourceKind, candidate.Source.ID, candidate.Source.Label)
	}
	if candidate.Source.ID == "" {
		candidate.Source.ID = candidate.ID
	}
	if candidate.Source.Label == "" {
		candidate.Source.Label = candidate.Source.ID
	}
	if candidate.BusinessObjectType == "" {
		candidate.BusinessObjectType = businessObjectType(candidate.Classification.Type)
	}
	if candidate.Confidence == 0 {
		candidate.Confidence = candidate.Classification.Confidence
	}
	candidate.Confidence = clampConfidence(candidate.Confidence)
	candidate.Classification.Confidence = clampConfidence(candidate.Classification.Confidence)
	if candidate.ReviewStatus == "" {
		candidate.ReviewStatus = ReviewStatusForConfidence(candidate.Confidence, false)
	}
	if len(candidate.ExtractedFields) == 0 {
		candidate.Warnings = append(candidate.Warnings, "no extracted fields available")
	}
	candidate.ExtractedFields = sortedFields(candidate.ExtractedFields)
	candidate.SuggestedLinks = sortedLinks(candidate.SuggestedLinks)
	candidate.AuditRefs = sortedAuditRefs(candidate.AuditRefs)
	candidate.Warnings = uniqueStrings(candidate.Warnings)
	return candidate
}

func ReviewStatusForConfidence(confidence float64, needsReview bool) ReviewStatus {
	confidence = clampConfidence(confidence)
	if needsReview || confidence < 0.85 {
		return ReviewStatusNeedsReview
	}
	return ReviewStatusNew
}

func FieldStatusForValue(value string, confidence float64) FieldStatus {
	if strings.TrimSpace(value) == "" {
		return FieldStatusMissing
	}
	if confidence > 0 && confidence < 0.75 {
		return FieldStatusNeedsConfirmation
	}
	return FieldStatusExtracted
}

func fieldFromValue(name string, value any, source string, confidence float64) ExtractedField {
	text := strings.TrimSpace(fmt.Sprint(value))
	if value == nil {
		text = ""
	}
	return ExtractedField{
		Name:       normalizeKey(name),
		Label:      titleFromKey(name),
		Value:      text,
		Status:     FieldStatusForValue(text, confidence),
		Confidence: clampConfidence(confidence),
		Source:     source,
	}
}

func fieldsFromStringMap(values map[string]string, source string, confidence float64) []ExtractedField {
	fields := make([]ExtractedField, 0, len(values))
	for key, value := range values {
		fields = append(fields, fieldFromValue(key, value, source, confidence))
	}
	return sortedFields(fields)
}

func fieldsFromAnyMap(values map[string]any, source string, confidence float64) []ExtractedField {
	fields := make([]ExtractedField, 0, len(values))
	for key, value := range values {
		if strings.EqualFold(key, "raw_text") {
			continue
		}
		fields = append(fields, fieldFromValue(key, value, source, confidence))
	}
	return sortedFields(fields)
}

func suggestedLinksFromActions(actions []string, businessType string) []SuggestedLink {
	links := make([]SuggestedLink, 0, len(actions))
	for _, action := range actions {
		label := strings.TrimSpace(action)
		if label == "" {
			continue
		}
		links = append(links, SuggestedLink{
			ID:                           buildID("link", businessType, label),
			Label:                        label,
			Reason:                       "Runtime or classifier suggested this as the next deterministic review target.",
			BusinessObjectType:           businessType,
			RequiredDeterministicService: deterministicServiceForAction(label, businessType),
		})
	}
	return sortedLinks(links)
}

func businessObjectType(classification string) string {
	switch normalizeKey(classification) {
	case "invoice":
		return "customer_invoice"
	case "supplierinvoice", "supplier_invoice":
		return "supplier_invoice"
	case "rfq", "rfq_document", "rfq_email":
		return "opportunity"
	case "quotation", "commercial_offer":
		return "quotation"
	case "purchaseorder", "purchase_order", "customer_po", "internal_po", "supplier_po_ack":
		return "purchase_order"
	case "deliverynote", "delivery_note":
		return "delivery_note"
	case "bankstatement", "bank_statement":
		return "bank_statement"
	case "contract":
		return "contract"
	case "report":
		return "report"
	default:
		return "business_record"
	}
}

func deterministicServiceForAction(action string, businessType string) string {
	action = strings.ToLower(action)
	switch {
	case strings.Contains(action, "bank"):
		return "finance.banking.review"
	case strings.Contains(action, "supplier invoice"):
		return "finance.supplier_invoice.review"
	case strings.Contains(action, "invoice"), businessType == "customer_invoice":
		return "finance.invoice.review"
	case strings.Contains(action, "opportunity"), strings.Contains(action, "rfq"):
		return "crm.opportunity.review"
	case strings.Contains(action, "order"):
		return "crm.order.review"
	case strings.Contains(action, "delivery"), strings.Contains(action, "grn"):
		return "operations.delivery.review"
	case strings.Contains(action, "archive"):
		return "documents.inbox.review"
	default:
		return "documents.intake.review"
	}
}

func SourceKindFromPath(path string) SourceKind {
	ext := strings.ToLower(strings.TrimPrefix(lastExt(path), "."))
	switch ext {
	case "msg", "eml":
		return SourceKindEmail
	case "pdf":
		return SourceKindPDF
	case "png", "jpg", "jpeg", "bmp", "tiff", "tif", "webp":
		return SourceKindScreenshot
	case "xlsx", "xls", "csv":
		return SourceKindExcel
	default:
		if strings.TrimSpace(path) == "" {
			return SourceKindOther
		}
		return SourceKindOther
	}
}

func buildID(parts ...any) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		value := normalizeKey(fmt.Sprint(part))
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	if len(normalized) == 0 {
		return "intake-candidate"
	}
	return strings.Join(normalized, "-")
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_", "/", "_", "\\", "_", ":", "_")
	value = replacer.Replace(value)
	value = strings.Trim(value, "_")
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return value
}

func titleFromKey(value string) string {
	value = strings.ReplaceAll(normalizeKey(value), "_", " ")
	parts := strings.Fields(value)
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedFields(fields []ExtractedField) []ExtractedField {
	sort.SliceStable(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
	return fields
}

func sortedLinks(links []SuggestedLink) []SuggestedLink {
	sort.SliceStable(links, func(i, j int) bool { return links[i].ID < links[j].ID })
	return links
}

func sortedAuditRefs(refs []AuditRef) []AuditRef {
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Type == refs[j].Type {
			return refs[i].SourceID < refs[j].SourceID
		}
		return refs[i].Type < refs[j].Type
	})
	return refs
}

func lastExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		switch path[i] {
		case '.':
			return path[i:]
		case '/', '\\':
			return ""
		}
	}
	return ""
}
