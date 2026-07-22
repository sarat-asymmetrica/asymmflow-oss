// Wire and result types for the Mistral OCR 4 client.
// σ: MistralOCR-Types | ρ: pkg/ocr/mistralocr | γ: Production
//
// Wire shapes are pinned against docs.mistral.ai/api/endpoint/ocr as re-verified 2026-07-22
// (see FABLE_WAVE13_REPORT.md, "A0 — Mistral OCR API ground truth"). Confidence is native at
// page/word granularity only; per-field confidence is a client-side derivation, documented in
// the report and in Result.Fields below — never presented as an API-native guarantee.
package mistralocr

import "strings"

// DocumentInput describes exactly one document to send to /v1/ocr. Exactly one of URL or Data
// must be set.
type DocumentInput struct {
	URL      string // remote document_url / image_url
	Data     []byte // local bytes; encoded by the client as a data: URI
	MIMEType string // required when Data is set (e.g. "application/pdf", "image/png")
	IsImage  bool   // true routes to an image_url chunk instead of document_url
}

// DocumentSchema is a caller-authored JSON Schema used for whole-document structured
// extraction via the OCR endpoint's document_annotation_format field.
type DocumentSchema struct {
	Name   string
	Schema map[string]any
	Strict bool
}

// ProcessOptions controls a single Process call.
type ProcessOptions struct {
	// Schema, if set, requests document-level structured extraction (Document AI annotations).
	// Nil means plain OCR — Result.Fields will be empty.
	Schema *DocumentSchema

	// IncludeBlocks requests paragraph-level bounding boxes / block classification.
	IncludeBlocks bool

	// Pages restricts processing to a subset (comma/range syntax per the API, e.g. "1-3,5").
	// Empty means all pages, subject to the client's PageCap.
	Pages string
}

// FieldValue is one decoded structured-extraction field with an honest confidence signal.
// Confidence is derived from the page-level confidence scores the API returns (there is no
// native per-field confidence) — see the report for the derivation rule. NeedsReview is true
// whenever Confidence is below the client's ConfidenceThreshold, INCLUDING when no confidence
// signal was available at all (Confidence defaults to 0 in that case — refuse-to-guess).
type FieldValue struct {
	Value       any
	Confidence  float64
	NeedsReview bool
}

// BoundingBox is a page-relative box in the units the API reports (top-left origin).
type BoundingBox struct {
	X0, Y0, X1, Y1 float64
}

// Block is one paragraph/content block, present only when ProcessOptions.IncludeBlocks was set.
type Block struct {
	PageIndex  int
	Type       string
	Text       string
	BBox       *BoundingBox // nil if the API did not return coordinates for this block
	Confidence float64
}

// Result is the decoded, typed OCR outcome for one document.
type Result struct {
	Text    string // all pages' markdown, joined with blank lines
	Pages   []string
	Blocks  []Block
	Fields  map[string]FieldValue // populated only when ProcessOptions.Schema was set
	ModelID string
}

// ---- wire types (unexported — these mirror the OCRRequest/OCRResponse shapes from
// docs.mistral.ai/api/endpoint/ocr) ----

type documentURLChunk struct {
	Type        string `json:"type"` // "document_url"
	DocumentURL string `json:"document_url"`
}

type imageURLChunk struct {
	Type     string `json:"type"` // "image_url"
	ImageURL string `json:"image_url"`
}

type responseFormat struct {
	Type       string             `json:"type"` // "json_schema"
	JSONSchema jsonSchemaEnvelope `json:"json_schema"`
}

type jsonSchemaEnvelope struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type ocrRequest struct {
	Model                       string          `json:"model"`
	Document                    any             `json:"document"` // documentURLChunk or imageURLChunk
	Pages                       string          `json:"pages,omitempty"`
	IncludeBlocks               bool            `json:"include_blocks,omitempty"`
	ConfidenceScoresGranularity string          `json:"confidence_scores_granularity,omitempty"`
	DocumentAnnotationFormat    *responseFormat `json:"document_annotation_format,omitempty"`
}

type ocrPageObject struct {
	Index      int            `json:"index"`
	Markdown   string         `json:"markdown"`
	Dimensions map[string]any `json:"dimensions,omitempty"`
	Blocks     []ocrBlockWire `json:"blocks,omitempty"`

	AveragePageConfidenceScore float64   `json:"average_page_confidence_score,omitempty"`
	MinimumPageConfidenceScore float64   `json:"minimum_page_confidence_score,omitempty"`
	WordConfidenceScores       []float64 `json:"word_confidence_scores,omitempty"`
}

type ocrBlockWire struct {
	Type       string    `json:"type"`
	Text       string    `json:"text"`
	BBox       *bboxWire `json:"bbox,omitempty"`
	Confidence float64   `json:"confidence,omitempty"`
}

type bboxWire struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}

type ocrUsageInfo struct {
	PagesProcessed int  `json:"pages_processed"`
	DocSizeBytes   *int `json:"doc_size_bytes"`
}

type ocrResponse struct {
	Pages              []ocrPageObject `json:"pages"`
	Model              string          `json:"model"`
	DocumentAnnotation map[string]any  `json:"document_annotation"`
	UsageInfo          ocrUsageInfo    `json:"usage_info"`
}

type errorEnvelope struct {
	Object  string `json:"object"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
