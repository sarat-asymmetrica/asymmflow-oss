# FABLE WAVE 13 — PERCEPTION & PRINT — WAVE REPORT

Report is built incrementally as missions land. This entry covers **P0 (A0 API ground truth)**
and **P2 (pkg/ocr/mistralocr client package)**.

## A0 — Mistral OCR API ground truth

Re-verified 2026-07-22 against live docs.mistral.ai (not the marketing announcement). Sources:

- https://docs.mistral.ai/api/endpoint/ocr (canonical endpoint reference)
- https://docs.mistral.ai/capabilities/OCR/basic_ocr/ (usage guide + code samples)
- https://docs.mistral.ai/capabilities/document_ai/annotations/ (Document AI / schema-annotation guide)
- https://docs.mistral.ai/models/model-cards/ocr-4-0 (OCR 4 model card)
- https://docs.mistral.ai/resources/known-limitations (limits)
- https://docs.mistral.ai/resources/error-glossary (error shape)
- https://mistral.ai/news/ocr-4/ (pricing confirmation)

### Endpoint

`POST /v1/ocr` against `https://api.mistral.ai`. Confirms the spec's assumption.

### Model ID

`mistral-ocr-4-0` — confirmed as the literal model string in current docs/code samples. (One
fetch pass rendered a garbled `mistral-ocr-4-0+2`; treated as a scraping artifact and discarded —
every other independent fetch/search converged cleanly on `mistral-ocr-4-0`.)

### Request body (OCRRequest)

| Field | Type | Notes |
|---|---|---|
| `model` | string \| null | e.g. `mistral-ocr-4-0` |
| `document` | `DocumentURLChunk` \| `ImageURLChunk` \| `FileChunk` | discriminated by `type` |
| `pages` | string \| int[] \| null | subset of pages, comma/range syntax |
| `include_image_base64` | bool \| null | inline extracted images in response |
| `image_limit` | int \| null | cap on extracted images |
| `image_min_size` | int \| null | min px to bother extracting an image |
| `include_blocks` | bool | paragraph-level bounding boxes / block list |
| `extract_header` / `extract_footer` | bool | default false |
| `confidence_scores_granularity` | `"word"` \| `"page"` | confidence is **opt-in**, not automatic |
| `table_format` | `"markdown"` \| `"html"` | |
| `bbox_annotation_format` | ResponseFormat (json_schema) \| null | per-image structured extraction |
| `document_annotation_format` | ResponseFormat (json_schema) \| null | whole-document structured extraction |
| `document_annotation_prompt` | string \| null | steering prompt for document annotation |

**Document input variants:**
- `DocumentURLChunk`: `{"type": "document_url", "document_url": "<url or data: URI>"}`
- `ImageURLChunk`: `{"type": "image_url", "image_url": "<url or data: URI>"}`
- `FileChunk`: references a file already uploaded via the Files API (`file_id`).

**Discrepancy vs spec assumption:** the spec assumed a bare "base64 data-URI" document field.
The docs confirm the *shape* is `document_url`/`image_url` with a URL string — but per Mistral's
established convention elsewhere in the API (chat `image_url` also accepts `data:` URIs), local
files are sent as `data:application/pdf;base64,<...>` inside the same `document_url`/`image_url`
string field, not a separate base64 field. A live "OCR with a Base64 Encoded PDF" example exists
in the docs (tab present) but its body did not render through the fetcher; the client below
implements the `data:` URI convention since it is consistent across every other Mistral endpoint
and is the only mechanism documented for local-file OCR (there is no separate raw-base64 field in
the OCRRequest table above). **Flag for re-verification** against a live test call once a Mistral
key is available in this environment (none was — the resolver is untouched per spec, and tests
here are mock-server only).

### Response body (OCRResponse)

```
{
  "pages": [ OCRPageObject, ... ],
  "model": "mistral-ocr-4-0",
  "document_annotation": { ... } | null,   // present iff document_annotation_format was set
  "usage_info": { "pages_processed": int, "doc_size_bytes": int|null }
}
```

`OCRPageObject`: `index`, `markdown`, `images[]`, `tables[]` (per `table_format`), `hyperlinks[]`,
`header`/`footer` (iff extracted), `dimensions {dpi,height,width}`, `blocks[]` (iff
`include_blocks`), and confidence carried as:
- `average_page_confidence_score`, `minimum_page_confidence_score` (page granularity)
- `word_confidence_scores[]` (word granularity, per extracted word)

There is **no per-field confidence** in the base OCR response — confidence is page/word only.
Per-field confidence (as the spec's `FieldValue.Confidence` implies) only exists when
`document_annotation_format`/`bbox_annotation_format` is used; the model does not natively emit
one, so the client derives a field's confidence from the minimum word/page confidence overlapping
that field's source region when available, falling back to the page-average confidence, and
finally to a neutral default that is always marked `NeedsReview` if no signal exists at all
(see "Confidence derivation" below in the package section) — **this is a client-side heuristic**,
not an API-native guarantee, and must be stated as such wherever displayed.

### Document AI / annotations (schema-shaped extraction)

Same `/v1/ocr` endpoint. Caller supplies `bbox_annotation_format` and/or
`document_annotation_format` as a `ResponseFormat` of type `json_schema` (standard Mistral
structured-output envelope: `{"type": "json_schema", "json_schema": {"name": ..., "schema": {...},
"strict": true}}`). `document_annotation_format` reshapes output for the **whole document**;
`bbox_annotation_format` reshapes **per-image** annotations. The client package below only wires
`document_annotation_format` (whole-document schema extraction), matching the spec's "caller
supplies a JSON schema per document type" requirement; `bbox_annotation_format` is left as a
documented gap (P3's dispatch rewire doesn't need per-image annotation).

### Limits

- Max file size: **50 MB** (per multiple independent sources) — one fetch of
  `known-limitations` returned "512 MB"; treated as unreliable/possibly conflating a different
  endpoint (Files API upload ceiling) and **not used**. The client enforces a configurable page
  cap only (per spec) and leaves file-size enforcement to the caller/UI, but logs the 50 MB
  figure as the documented ceiling in a doc comment.
- Max pages: **1000** per request (per search-aggregated FAQ; not independently confirmed via
  direct fetch of the FAQ answer body — flagged as re-verify-at-runtime, non-blocking since the
  client's own `PageCap` config defaults far below this).
- Supported OCR formats: PDF, PNG, JPG/JPEG, TIFF, BMP, GIF, WEBP (per known-limitations page).
  DOCX/PPTX handling for OCR 4 specifically is claimed by third-party summaries but not
  independently confirmed in first-party docs during this pass — the client accepts them as
  generic "document" input (same `document_url` chunk) since the API does not require a
  client-side format allowlist; the server will error if unsupported.

### Errors

Standard Mistral error envelope, uniform across all endpoints (not OCR-specific):

```json
{
  "object": "error",
  "message": "human-readable description",
  "type": "invalid_request_error",
  "param": "model",
  "code": "unknown_model"
}
```

HTTP status carries the primary signal: 401/403 → auth, 429 → quota/rate-limit, 413/422 (`type:
invalid_request_error` with size-related `code`) → too-large, 400/422 with schema-related `code`
→ schema-mismatch, 5xx → transient/server. The docs do not enumerate OCR-specific error `code`
values (e.g. no confirmed literal like `document_too_large`), so the client classifies by **HTTP
status + `type`** rather than pattern-matching undocumented `code` strings, and falls back to a
generic API error carrying the raw envelope for anything it can't classify.

### Pricing (confirmed, informational only — not wired into the client)

$4/1k pages OCR, $5/1k pages with Document AI annotations, $2/1k pages batch.

---

## P2 — `pkg/ocr/mistralocr` client package

New package, no existing files touched. Files:
- `pkg/ocr/mistralocr/client.go` — `Client`, config, HTTP plumbing, `Process`.
- `pkg/ocr/mistralocr/types.go` — request/response wire types, `Result`, `FieldValue`.
- `pkg/ocr/mistralocr/errors.go` — typed error classes.
- `pkg/ocr/mistralocr/client_test.go` — table-driven httptest tests.

### Exported API surface

```go
package mistralocr

type Config struct {
    APIKey             string        // caller-injected; client never reads env/DB
    BaseURL            string        // default "https://api.mistral.ai"
    Model              string        // default "mistral-ocr-4-0"
    PageCap            int           // default 200 (well under documented 1000-page ceiling)
    Timeout            time.Duration // default 60s
    ConfidenceThreshold float64      // default 0.85
}

func NewClient(cfg Config) *Client

type DocumentInput struct {
    // exactly one of:
    URL      string // remote document_url / image_url
    Data     []byte // local bytes, base64-encoded as a data: URI by the client
    MIMEType string // required when Data is set
    IsImage  bool   // routes to image_url vs document_url chunk type
}

type FieldValue struct {
    Value       any
    Confidence  float64
    NeedsReview bool
}

type Block struct {
    PageIndex int
    Type      string
    Text      string
    BBox      *BoundingBox // nil if not requested/available
    Confidence float64
}

type BoundingBox struct{ X0, Y0, X1, Y1 float64 }

type Result struct {
    Text    string                 // concatenated markdown across pages
    Pages   []string               // per-page markdown
    Blocks  []Block
    Fields  map[string]FieldValue  // populated only when a Schema was supplied
    ModelID string
}

func (c *Client) Process(ctx context.Context, doc DocumentInput, opts ProcessOptions) (*Result, error)

type ProcessOptions struct {
    Schema          *DocumentSchema // nil = plain OCR, no annotation request
    IncludeBlocks   bool
}

type DocumentSchema struct {
    Name   string
    Schema map[string]any // caller-authored JSON Schema
    Strict bool
}

// Typed errors — callers use errors.As to branch fallback behavior.
type AuthError struct{ *APIError }
type QuotaError struct{ *APIError }
type TooLargeError struct{ *APIError }
type SchemaMismatchError struct{ *APIError }
type APIError struct {
    StatusCode int
    Type       string
    Code       string
    Message    string
}
func (e *APIError) Error() string
```

### Confidence derivation

The API gives page/word confidence, not per-field confidence. When a `Schema` is supplied and the
model returns `document_annotation`, the client assigns each decoded field the **minimum page
confidence across the document** (conservative choice — a field could originate from any page)
when `confidence_scores_granularity: "page"` is requested, which the client always requests
whenever a schema is set. Any field whose assigned confidence is below `ConfidenceThreshold`
(default 0.85) is marked `NeedsReview: true`; if the API returned no confidence signal at all
(older/degraded response), the field defaults to confidence `0` and `NeedsReview: true` —
refuse-to-guess, never silently accepted.

### Typed errors

Classified by HTTP status + envelope `type`, per the A0 findings (no reliance on undocumented
`code` strings): 401/403 → `AuthError`; 429 → `QuotaError`; 413 or 422 with a size-shaped message
→ `TooLargeError`; 400/422 otherwise when a `Schema` was in the request → `SchemaMismatchError`;
anything else → plain `*APIError`. All wrap the raw envelope so callers can inspect `Code`/`Type`
directly if they need finer branching than the four classes.

### Tests (`client_test.go`, table-driven, httptest mock server, no real network)

1. Happy path — plain OCR, no schema: asserts request body (`model`, `document.type`,
   `document.document_url` data-URI prefix + payload), asserts decoded `Result.Text`/`Pages`.
2. Happy path — with schema annotation: asserts `document_annotation_format` sent with the exact
   schema payload, asserts decoded `Fields` map values.
3. Low-confidence flagging: mock page confidence below threshold → asserts `NeedsReview: true`
   and the numeric confidence carried through unmodified (not silently dropped).
4. Missing confidence signal: mock response omitting confidence entirely → asserts
   `NeedsReview: true` with confidence `0` (refuse-to-guess path).
5. Auth error (401 envelope) → asserts `errors.As` to `*AuthError`.
6. Quota error (429 envelope) → asserts `errors.As` to `*QuotaError`.
7. Too-large error (413 envelope) → asserts `errors.As` to `*TooLargeError`.
8. Schema-mismatch error (422 envelope, schema was set) → asserts `errors.As` to
   `*SchemaMismatchError`.
9. Image input request-shape: asserts `image_url` chunk type used and base64 payload correct for
   `IsImage: true`.
10. Page cap enforcement: request with page count/`Pages` exceeding `PageCap` is rejected
    client-side before any HTTP call (asserts mock server received zero requests).
11. Timeout: context cancelled mid-request → asserts error returned, not a panic/hang.

All tests assert on concrete decoded values (never `if err != nil { t.Fail() }`-only tautologies)
so a broken implementation fails visibly, per the "verify the probe" lesson.

### Build/test results

(filled in after implementation — see below)

### Deviations from spec

None structural. The per-field confidence derivation is a necessary client-side design decision
(the API doesn't provide it natively) — flagged above, not hidden.

### What the next wave (P1/P3) can assume

- `mistralocr.NewClient(Config{APIKey: <from existing resolver>})` is the only integration point;
  P1/P3 must NOT read env/DB for the Mistral key themselves — inject via existing
  `getMistralAPIKey`.
- `Process` handles both PDF (native, `IsImage: false`) and image (`IsImage: true`) inputs — no
  page-render-to-PNG loop needed anywhere upstream.
- Errors are typed; P3's dispatch should use `errors.As` to decide offline-fallback vs surfacing.
- Per-field confidence is a heuristic (page-level minimum), documented above — do not present it
  to users as word-exact.
