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

### Build/test results (measured at the Phase-1 gate, 2026-07-22)

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./... -count=1` (full suite, run alone in the worktree): **87 packages ok, 0 failures**,
  exit code 0. Includes the 11 `pkg/ocr/mistralocr` table-driven tests (re-verified green after
  crash recovery) and the 3 new payslip tests (below).
- `go test -v ./pkg/engines/` — embedded-font path confirmed LIVE, not just compiled: test logs
  show `✓ Loaded embedded font for العربية: pkg/fonts (182848 bytes)` and
  `✓ Loaded embedded font for English: pkg/fonts (512588 bytes)` (+ Russian on Noto Sans), with
  **zero** "falling back to host probe" lines. The Arabic RTL and langpack root tests exercise
  real PDF generation through the embedded fonts.
- `TestAHSCostingExportEmbedsLetterhead` — the gopdf costing-export path renders green on
  embedded Noto Sans (no fallback warnings).

### Deviations from spec

None structural. The per-field confidence derivation is a necessary client-side design decision
(the API doesn't provide it natively) — flagged above, not hidden.

### What the next phase (P1/P3) can assume

- `mistralocr.NewClient(Config{APIKey: <from existing resolver>})` is the only integration point;
  P1/P3 must NOT read env/DB for the Mistral key themselves — inject via existing
  `getMistralAPIKey`.
- `Process` handles both PDF (native, `IsImage: false`) and image (`IsImage: true`) inputs — no
  page-render-to-PNG loop needed anywhere upstream.
- Errors are typed; P3's dispatch should use `errors.As` to decide offline-fallback vs surfacing.
- Per-field confidence is a heuristic (page-level minimum), documented above — do not present it
  to users as word-exact.

---

## P5 — Print determinism: embedded fonts

**Reconstructed at the gate from the recovered diff** (the coder's report entry was lost to the
crash); every claim below is re-verified against the code and measured test output, not inherited.

Files: new `pkg/fonts/` (fonts.go + `data/*.ttf` + OFL.txt), `pkg/engines/pdf_generator.go`
(`loadLanguageFonts` only), `app_costing_exports_surface.go` (font-probe block of
`exportCostingToPDF` only). No `go.mod` change needed (`go:embed` is stdlib).

- `pkg/fonts` embeds four TTFs via `go:embed` with accessor functions: Noto Sans Regular/Bold
  (Latin/Cyrillic/Greek) and Noto Naskh Arabic Regular/Bold. OFL 1.1 license text committed
  alongside (`pkg/fonts/OFL.txt`). Sources: github.com/notofonts hinted static instances.
- `exportCostingToPDF` (gopdf): embedded Noto Sans via `AddTTFFontData` is now PRIMARY; the old
  host-font candidate list (arial/calibri/DejaVu/...) survives ONLY as the fallback branch if the
  embed fails to parse, per spec.
- `pkg/engines` `loadLanguageFonts` (gopdf): embedded fonts are PRIMARY for the scripts we ship —
  `ar` → Noto Naskh Arabic, `en`/`ru` → Noto Sans. Host probing remains the source for scripts we
  do NOT embed (CJK, Thai, Devanagari, Korean, Hebrew) and the fallback if embedding fails.

### Binary-size delta (measured)

Embedded payload = 1,404,176 bytes (~1.34 MiB): NotoSans-Regular 512,588 + NotoSans-Bold
515,668 + NotoNaskhArabic-Regular 182,848 + NotoNaskhArabic-Bold 193,072. Well under the spec's
8 MB subsetting threshold — **no subsetting performed**, full glyph coverage retained.

### Acceptance evidence

- Arabic + langpack + generator tests in `pkg/engines` PASS and log the embedded path
  (`✓ Loaded embedded font ... pkg/fonts (...bytes)`) with zero host-probe fallbacks.
- `TestAHSCostingExportEmbedsLetterhead` PASS — costing export renders on embedded Noto Sans.
- Invoice PDF: `TestGenerateInvoicePDF_AttentionFallbackAndLongRef` PASS (truncation helper
  behaviour intact — `TestTruncatePDFTextToWidth` PASS).
- **Byte parity with main is intentionally broken** for gopdf documents (different font bytes
  embedded in the PDF). The invariant honoured is CONTENT parity: same fields, same values, no
  clipped/overflowed text — stated per spec, not hidden.

### Deviations / honesty notes

- The gofpdf-based generators (invoice, payslip, butler reports) use gofpdf's built-in core
  Helvetica, which never touched host fonts — they were already deterministic, so they were NOT
  moved to `AddUTF8FontFromBytes`. The spec's gofpdf mention is satisfied vacuously; the two
  host-probing sites named in the spec's §0 ground truth were exactly the two fixed here.
- Scripts other than Latin/Cyrillic/Arabic still depend on host fonts (unchanged from main) —
  documented in the code comment at the fallback branch.

---

## P6 — Payslip PDF

**Reconstructed at the gate from the recovered diff** — payslip_pdf_service.go survived the crash
only as an orphaned atomic-write tmp file (write complete, rename lost); it was gated as if no
coder report ever existed: every helper call re-verified against the real signature, scope
discipline re-checked against the spec.

Files: new `payslip_pdf_service.go` (278 lines), `service_finance.go` (one delegate),
`frontend-lab` (bridge + viewmodel + screen + parity ledger + hand-added wailsjs binding entries),
new `payslip_pdf_service_test.go`.

- `GeneratePayslipPDF(employeeID, payrollPeriodID)` on gofpdf, following the
  invoice_pdf_service.go pattern: letterhead via `applyLetterheadForDivision`, identity via the
  overlay-backed `companyDocumentProfile(run.Division)`, export dir via
  `a.getExportDir(...)` → `<Documents>\AsymmFlow Exports\Reports`, amount-in-words via the
  existing package-main `amountInWords` (NOT the pkg/engines stub). RBAC:
  `payrollGuarded(requirePayrollView)`.
- **Zero payroll computation**: earnings/deductions render from stored `PayrollComponent` rows
  (fallback for pre-component legacy items: the item's own stored totals); NET PAY displays the
  item's stored `NetPay`, never re-derived. Picks the MOST RECENT run for the employee/period
  pair (regenerate-after-correction is legal). Currency from the stored run, defaulting BHD only
  when the run carries none.
- Employee display fields fall back to the run item's name/title SNAPSHOTS when the live employee
  record is gone — a payslip never renders blank identity.
- Frontend (gated): `FinanceService.GeneratePayslipPDF` delegate + hand-added wailsjs entries;
  bridge `generatePayslipPdf` is a REAL pass-through (read-only export action — renders committed
  data, no state transition — documented in the parity ledger as row 15, unlike the file's
  financial mutations which stay honest INTEG-gap throws); minimal employee picker + button in
  the run detail, success/error surfaced via the screen's existing CalloutWidget idiom;
  `selectRun` resets payslip state so results never carry across runs.

### Tests (new, all green)

- `TestGeneratePayslipPDF_FromGoldenRun` — synthetic employee through the payroll golden
  harness, real run generated, payslip rendered; asserts file exists, >1000 bytes, `%PDF-` header.
- `TestGeneratePayslipPDF_NoRunItem_Errors` — refuses to render for an employee with no run item.
- `TestGeneratePayslipPDF_BlankInputs_Error` — blank IDs rejected before DB work.
- Frontend: svelte-check 357 files 0 errors/0 warnings; vite build green; vitest 238/238
  (INTEG-gap-count tripwire unaffected).

### Deviations / honesty notes

- `getExportDir("report", ...)` routes all payslips to the flat `Reports` folder (its default
  branch ignores the category/year args) — consistent with the service's header comment; noted
  in case a `Payslips` subfolder is wanted later.
- The tracked-but-gitignored `frontend-lab/dist/index.html` build artifact was reverted at the
  gate (hash churn referencing an untracked asset — build noise, not mission content).

**Phase 1 gate: COMPLETE.** P2 + P5 + P6 all gated; full suite alone = 87 packages ok, 0
failures; build + vet clean.

---

## P1 — One provider: Butler → Mistral direct (AIMLAPI/Grok deleted)

Coder report gate-verified: the sacred key resolver `getMistralAPIKey` is untouched (all 10
diff hunks in butler_ai.go end before it — verified hunk-by-hunk at the gate), and the
aiml/grok grep sweep was re-run independently.

- **Butler chat**: primary = `mistral-large-latest` via api.mistral.ai direct; the entire
  AIMLAPI/Grok client deleted from butler_ai.go (callAIML* family, key/model providers, ~290
  lines), the app.go provider-registration block removed, chat_service.go's two AIML branches
  now call Mistral directly (`usedBackend` always "Mistral"), app_butler_ports.go de-branched.
- **Config-not-constant**: `MISTRAL_CHAT_API_URL` (default api.mistral.ai/v1/chat/completions),
  `MISTRAL_MODEL_SMALL` (mistral-small-latest), `MISTRAL_MODEL_LARGE` (mistral-large-latest) —
  env-seeded package vars with the previous literals as defaults. `mistral-small-latest` kept
  where already used for classification.
- **callMistralVision / pixtral + page-render loop: DELETED.** Callers repointed to
  pkg/ocr/mistralocr (P3's dispatch).
- **ACEEngine (pkg/ocr/engine.go)**: tesseract+pandoc local pipeline kept exactly; cloud
  escalation is now pkg/ocr/mistralocr (native PDF/image, IncludeBlocks) behind
  `EngineConfig.MistralAPIKey/MistralBaseURL/MistralModel/FallbackToMistral` +
  `ConfidenceThreshold` (default 0.85, replaces the old AIMLAPI_THRESHOLD constant).
  `TierAIMLAPI` → `TierCloudOCR`. pkg/ocr/aimlapi.go deleted. Archaeologist key usage moved to
  the standard resolver.
- **Bank-statement AI assist** (LOW deviation, ratified at gate): stays on Mistral
  chat-completions rather than mistralocr because both call sites only ever pass already-
  extracted TEXT, never document bytes — mistralocr (document-native) does not fit. Renames:
  `ENABLE_MISTRAL_BANK_STATEMENT_ASSIST` (was ENABLE_AIML_...), import tag `PDF_OCR_MISTRAL:`.
- **Dead scaffolding modernized rather than deleted** (zero callers, verified): pkg/setup/wizard
  (Mistral key + api.mistral.ai validation), pkg/orchestrator/intent_processor
  (mistral-small-latest).
- **Docs**: SECRET_CONFIGURATION_BASELINE gains a Wave 13 addendum retiring the documented
  AIML key-naming-mismatch bug (strikethrough, not silent rewrite); admin guide settings table
  collapsed to one Mistral key row; BUTLER_PIPELINE_REBUILD_SPEC banner-marked superseded.
- **Grep-clean (gate-verified)**: remaining aiml/grok hits in live Go code are removal-
  documentation comments, one legacy-key MASKING rule in security_enhancements.go (kept
  deliberately — old DBs may still hold an aimlapi key that must never print), an arbitrary
  test env-var name in config_test.go, and the generated capnp schema (exempt).
- LOW deviations ratified: three one-line collateral compile fixes outside the exclusive list
  (pkg/ocr/example_test.go, pkg/ocr/batch_offers_test.go, manual_deployment_package_test.go);
  chat model config is env-level only (the settings-DB model provider was itself part of the
  deleted provider plumbing — single-provider system needs no per-user model override).

## P3 — Dispatch rewire · Fly.io retired · dead tree deleted

- **New dispatch** (ocr_service_simple.go): PDF → go-fitz → if scant (`OCR_SCANT_TEXT_THRESHOLD`,
  default 50, was a hardcoded literal) → Mistral OCR 4 NATIVE PDF with a Document AI schema →
  offline tesseract via a dedicated local-only ACEEngine (constructed with no key,
  FallbackToMistral:false — structurally incapable of a network call). Images → mistralocr →
  tesseract. Office formats unchanged (local parse). `ocrWithMistral` checks the key BEFORE any
  network attempt, so keyless boxes fall straight through — offline-first verified by a test
  asserting the local path returns Success=true + NeedsReview=true, never an error.
- **Structured extraction** (new ocr_document_schemas.go): 5 Document AI schemas (invoice
  [+supplier_invoice], rfq [+quotation], purchase_order, bank_statement, generic) mirroring the
  regex layer's field names so downstream consumers are agnostic to the engine. The regex layer
  is DEMOTED: `extractFieldsFromTextLegacy` / `detectDocumentTypeFromTextLegacy`, offline/
  local-parse paths only, doc-commented as degraded mode.
- **Confidence UX**: OCRResultSimple gains `NeedsReview`, `FieldConfidence`,
  `FieldsForReview` (wailsjs models/bindings hand-updated). HONEST GAP, ratified at gate: no
  frontend screen consumes OCR results today (verified — none of the OCR bindings are called
  from frontend-lab/src), so this is forward-wiring; building a review screen would have been a
  redesign outside scope. Flagged for the wave that builds the document inbox UI.
- **Mistral OCR config** (env, all defaulted): MISTRAL_OCR_MODEL, MISTRAL_OCR_BASE_URL,
  MISTRAL_OCR_PAGE_CAP, MISTRAL_OCR_TIMEOUT_SECONDS, MISTRAL_OCR_CONFIDENCE_THRESHOLD.
- **Fly.io retired**: callFlyOCR + retry client + pkg/ocr/fly_ocr_client.go + `flyEndpoint`
  field deleted; the misleading ProcessWithFlorence2/Tesseract/GPU bindings deleted outright
  (zero frontend callers — verified, so no repoint needed) with their service_documents.go and
  wailsjs mirrors. Gate-verified grep: `fly.dev` / `FLY_OCR_URL` / `ASYMM_API_KEY` (OCR) /
  `pixtral` = zero hits in live Go code.
- **Dead tree deleted**: pkg/ocr went from ~90 files / 8 subdirs to 12 files / 2 subdirs.
  orchestrator (~30 files incl. florence/modal/fly processors), ksum, octonion, organism,
  predator, sparse, and the research fitz all deleted (each verified import-free first;
  pkg/documents/ocr fitz engine kept). trinity.go kept (live: TrinityOptimizer/BabelMapper used
  by engine.go).
- **Error-policy ruling**: on ANY mistralocr error class the dispatch logs the specific class
  but always falls back to the offline path — offline-first ("never error the inbox") outranks
  fast-fail. Ratified at gate.
- Also fixed at coder level: a self-caught err-shadowing bug in the image branch
  (mistralErr/localErr separated); two AIMLAPI settings-surface remnants in
  app_setup_documents_surface.go.

## Gate rulings & gate-applied fixes (Phase 2)

1. **pkg/ocr/orchestrator rump → pkg/ocr/preprocess** (gate-applied): the sole live dependency
   of the deleted research tree (gpu_preprocessor.go, self-contained, stdlib-only) is a genuine
   direct dep of ACEEngine.preprocess() — live via Archaeologist/batch-offers `EnableGPU:true` —
   so per spec it survives; but a one-file package still named "orchestrator" is misleading
   residue, so it was renamed to `pkg/ocr/preprocess` and its header comment corrected for
   honesty: it is a pure-Go CPU implementation (SLERP quaternion denoising + contrast); the old
   "Level Zero GPU" / "22.68M ops/sec" claims are gone, and engine.go's call-site comment now
   states that EnableGPU merely selects the path.
2. **runtime_handlers.go needs no change** (verified at gate): its client targets a LOCAL
   .NET runtime at localhost:5263 — NOT the Fly deployment — and its offline fallback already
   routes through SimpleOCRService, so it inherits the new dispatch automatically. Its header
   comment describes the external C# runtime's endpoints, left as-is.
3. **.env.example**: verified clean at the gate — zero aiml/fly/grok/pixtral references existed;
   nothing to remove. MISTRAL_API_KEY remains the single AI credential.
4. **Generated capnp schema** (OCRTier_aimlapi): exempt as generated wire-schema; regeneration
   is out of wave scope. Residue, flagged.
