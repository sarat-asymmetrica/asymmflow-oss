# FABLE WAVE 13 — PERCEPTION & PRINT

**One provider in, one engine out.** Consolidate all AI perception (OCR + Butler) onto Mistral's
direct endpoints under a single `MISTRAL_API_KEY`, migrate OCR from vision-chat-plus-regex to the
dedicated Mistral OCR 4 document endpoint with schema-shaped extraction and honest confidence,
retire the Fly.io OCR runtime and the AIMLAPI/Grok secondary provider entirely, delete the dead
OCR research tree, and close the two real gaps in the print layer: host-font nondeterminism and
the missing payslip PDF.

---

## Sequencing & operating model

- **Repo:** asymmflow-oss (PUBLIC — synthetic law applies).
- **Branch:** `feat/fable-wave13-perception-print`, worktree `C:\Projects\asymmflow\asymmflow-oss-wave13`.
  The main checkout at `C:\Projects\asymmflow\asymmflow-oss` is OCCUPIED by the India IN-W1
  orchestrator on another branch. **Nothing in this wave touches that directory.**
- **Operating model:** Fable (this session) orchestrates and gates; Sonnet 5 agents code.
  Two phases; file ownership per mission is exclusive (listed per mission) — no two agents edit
  the same file in the same phase.
- **No merge, no push, no tag.** Owner final-gates from the wave report.
- **Owner rulings already made (2026-07-22):** Grok/AIMLAPI removed; Mistral-direct for both
  Butler (`mistral-large-latest`) and OCR (`mistral-ocr-4-0`); Fly.io retired entirely;
  go-fitz + tesseract remain as the offline fallback; cloud dependency on Mistral is accepted
  product posture (EU/GDPR story, one key on one console).

## Lessons inherited (binding)

- **Config-not-constant:** every model ID, endpoint base URL, confidence threshold, page cap
  is data (settings/env with sane defaults), never a literal sprinkled at call sites.
- **Refuse-to-guess:** when extraction confidence is below threshold, the field is surfaced as
  needs-verification — never silently filled.
- **Severity honesty** in the report; **stop-and-report** on anything touching posting logic
  (this wave should touch none).
- **Verify the probe:** the new OCR client's tests must be able to report failure — mock-server
  tests must assert on request shape AND exercise error/low-confidence paths.
- **Synthetic law:** public repo — synthetic names only in fixtures/tests.

## §0 Ground truth (from 2026-07-22 recon — trust but re-verify at file level)

- Production OCR path = `SimpleOCRService` (`ocr_service_simple.go`, wired `app.go:905`):
  PDF → go-fitz text layer → if <50 chars → `pixtral-large-latest` vision-chat (pages rendered
  to PNG, ≤5 pages) → Fly.io runtime fallback. Images → pixtral → Fly.io. Office formats parsed
  locally. Fields scraped from free text by ~30 regexes (`extractFieldsFromText` :1314) with
  synthetic-company literals and BHD assumptions — the brittle layer.
- Butler chat primary = `x-ai/grok-4-fast-reasoning` via AIMLAPI; Mistral fallback
  (`butler_ai.go:56-57,65,80-81`). Vision OCR wrapper `callMistralVision` (:711).
- `ACEEngine` (`pkg/ocr/engine.go`) = tesseract + pandoc local, escalates to AIMLAPI
  `gpt-4o-mini` <0.85 confidence; used by Archaeologist + batch offers.
- `pkg/ocr/orchestrator` tree (florence/modal/predator/octonion/etc.) = research code, NOT wired;
  Wails bindings `ProcessWithFlorence2/Tesseract/GPU` (`app_setup_documents_surface.go:4436+`)
  all silently redirect to `callFlyOCR`.
- Fly.io client: `callFlyOCR`, retry/backoff, endpoints under `FLY_OCR_URL` default
  `asymmetrica-runtime.fly.dev`, bearer `ASYMM_API_KEY`.
- Key resolution: `getMistralAPIKey` (`butler_ai.go:1615`) = encrypted DB → settings.json → env.
  Keep this resolver; all new Mistral calls go through it.
- PDF layer = pure Go (gofpdf primary, gopdf for offers/costing + Arabic, pdfcpu merge). Fonts
  NOT embedded — host-font probing (`app_costing_exports_surface.go:639`,
  `pkg/engines/pdf_generator.go:273`); Arabic depends on OS fonts. No payslip PDF exists.
- OSS and PH forks byte-identical across OCR and PDF layers — substrate changes flow to both.

### Mistral OCR 4 facts (announcement 2026-06-23; A0 re-verification required)

- Model `mistral-ocr-4-0`; dedicated OCR endpoint (documented as `/v1/ocr`); accepts PDF/DOC/PPT
  directly (no page-render loop needed); returns structured blocks with bounding boxes,
  block-type classification, per-page and per-word confidence.
- Document AI layer: same endpoint reshapes output via caller-supplied JSON schema
  (annotations) — structured field extraction without client-side parsing.
- Pricing: $4/1k pages OCR, $5/1k Document AI, $2/1k batch. B2C-friendly.
- **A0 (first task of Phase 1, blocking P2):** fetch current official API docs
  (docs.mistral.ai — OCR + annotations endpoints) and pin: exact request/response JSON shapes,
  document input encoding (URL vs base64 data-URI), schema-annotation request format, confidence
  field names, page limits, error codes. Write findings into the wave report. Do NOT build the
  client from the marketing summary above.

---

## Phase 1 (parallel, independent)

### P0 — API ground truth (A0 above)
Owner: the P2 agent, as its first deliverable before any client code.

### P2 — Mistral OCR 4 client package
**New package `pkg/ocr/mistralocr`** (exclusive ownership; no edits to existing files this phase):
- Client for the dedicated OCR endpoint: submit PDF (native, no PNG rendering) and images;
  configurable model ID (default `mistral-ocr-4-0`), base URL, page cap, timeout.
- Schema-annotation support: caller passes a JSON schema per document type; client returns
  decoded structured fields + per-field/page confidence + raw markdown/text for display.
- Typed result: `{Text, Pages, Blocks, Fields map[string]FieldValue{Value, Confidence}, ModelID}`.
- Confidence threshold config (default 0.85): fields below threshold marked `NeedsReview: true`.
- Errors typed (auth, quota, too-large, schema-mismatch) — caller decides fallback.
- Table-driven tests against `httptest` mock server: happy path, low-confidence flagging,
  each error class, request-shape assertions (model, encoding, schema payload).
- **No network in tests.** No other file in the repo touched.

### P5 — Print determinism: embedded fonts
Files: `app_costing_exports_surface.go` (font-probe function only), `pkg/engines/pdf_generator.go`
(font loading only), new `pkg/fonts/` package, `go.mod` if needed.
- Add `pkg/fonts` with `go:embed` of Noto fonts: NotoSans (Regular/Bold) + NotoNaskhArabic
  (Regular/Bold) — download from Google Fonts (OFL), commit the TTFs + OFL license file.
- gofpdf: register via `AddUTF8FontFromBytes`; gopdf: `LoadTTFData` (or nearest API) — replace
  host-font probing as the PRIMARY source; host probe remains only as fallback if embed fails.
- All document generators end up on embedded fonts → identical output across machines.
- Acceptance: generate invoice + offer/costing + Arabic-content doc on the branch; PDFs open,
  no missing-glyph boxes, truncation helper still behaves (long-reference test). PDF bytes WILL
  differ from main (different font) — this is expected and must be stated honestly in the report;
  the invariant is CONTENT parity (same fields, same values, no clipped/overflowed text), not
  byte parity.
- Check binary-size delta and report it (Noto subsets if >8MB total; note what was subset).

### P6 — Payslip PDF
Files: new `payslip_pdf_service.go`, Wails binding registration (its own file or `app.go` region
coordinated with gate), frontend payroll screen hook (minimal button wiring).
- `GeneratePayslipPDF(employeeID, period)` on gofpdf, following `invoice_pdf_service.go`
  patterns: letterhead via `applyLetterheadForDivision`, identity via `overlay.Active()`,
  export dir via `a.getExportDir(...)`, amount-in-words via the existing `amountInWords` impl
  (NOT the stub in pkg/engines).
- Content: employee identity (respecting existing field-crypto rules — display only what the
  payroll screen already shows), earnings/deductions table, net pay, period, division identity.
- **Zero payroll computation changes** — read existing payroll records only. If any payslip
  field would require computing something new, stop-and-report.
- Test: generator unit test with synthetic employee fixture.

## Phase 2 (parallel, after P2 lands and is gated)

### P1 — One provider: Butler → Mistral direct
Files (exclusive): `butler_ai.go`, `document_classifier.go`, `bank_statement_ai_assist.go`,
`archaeologist.go`, `pkg/ocr/engine.go` + `pkg/ocr/aimlapi.go`, config/docs references to AIMLAPI.
- Butler chat: primary = `mistral-large-latest` via `api.mistral.ai` direct; delete the
  AIMLAPI/Grok provider path. Model IDs configurable; keep `mistral-small-latest` for
  classification where already used.
- `callMistralVision` + `ocrWithMistralVision` (pixtral + page-render loop): DELETE — callers
  repointed to `pkg/ocr/mistralocr` (coordinate signature with P3 agent via the interfaces P2
  defined; butler_ai.go is owned by THIS mission).
- Bank-statement AI assist: AIML PDF-OCR path → `pkg/ocr/mistralocr`.
- ACEEngine: keep tesseract + pandoc local pipeline; its cloud escalation (AIMLAPI gpt-4o-mini)
  → `pkg/ocr/mistralocr`; delete `pkg/ocr/aimlapi.go`. Archaeologist key usage moves to the
  standard Mistral key resolver.
- Remove `AIMLAPI_KEY`/`AIML_API_KEY` from config surface, `.env.example`, settings UI if
  present, and update `docs/ops/SECRET_CONFIGURATION_BASELINE*` (this also retires the
  documented key-naming-mismatch bug).

### P3 — Rewire dispatch + retire Fly.io + delete dead tree
Files (exclusive): `ocr_service_simple.go`, `runtime_handlers.go` (OCR parts),
`app_setup_documents_surface.go` (OCR bindings), `pkg/ocr/orchestrator` + sibling research dirs,
`.env.example` (FLY vars), related docs.
- New dispatch: **PDF** → go-fitz text layer → if scant → `mistralocr` (native PDF) → offline
  fallback tesseract (via ACEEngine local path) with honest low-confidence marking. **Images** →
  `mistralocr` → tesseract fallback. Office formats unchanged (local parse).
- Structured extraction: define JSON schemas per document type currently served by
  `extractFieldsFromText` (RFQ, invoice, PO, bank statement, generic) and pass them as
  Document AI annotations. The regex layer (`extractFieldsFromText`,
  `detectDocumentTypeFromText`) is DEMOTED to the offline/tesseract path only — clearly renamed
  (`extractFieldsFromTextLegacy`) and documented as degraded mode.
- Confidence UX: OCR results carry per-field confidence + `NeedsReview`; the existing document
  inbox/review surface shows flagged fields (minimal frontend wiring; no redesign).
- Fly.io: delete `callFlyOCR`, retry client, `FLY_OCR_URL`, `ASYMM_API_KEY` usage for OCR, all
  `/api/ocr/*` endpoint references, and the misleading `ProcessWithFlorence2/Tesseract/GPU`
  bindings (remove; if the frontend calls them, repoint honestly to the real dispatch).
- Delete `pkg/ocr/orchestrator` and sibling research subtrees (florence/modal/predator/octonion/
  sparse/ksum/fly_ocr and mystical-constants files) — everything in `pkg/ocr` EXCEPT the live
  ACEEngine pipeline (engine.go + its direct deps) and the fitz engine under `pkg/documents/ocr`.
  Build must stay green after deletion; if anything live imports the tree, stop-and-report
  rather than keep it.

## Hard boundaries

1. **Zero posting-logic changes.** OCR/print only. Stop-and-report otherwise.
2. **Offline-first survives:** with no network and no key, every document still gets go-fitz/
   tesseract treatment and the app never blocks or errors the inbox.
3. **Config-not-constant** (model IDs, endpoints, thresholds, page caps).
4. **Synthetic law** — public repo.
5. **India worktree untouchable** (`C:\Projects\asymmflow\asymmflow-oss`).
6. **No merge, no push, no tag.** Key revocation/minting = owner console action (the code's
   key resolver is unchanged, so a new key drops in with zero code change).

## Definition of done

- `go build ./...` and `go test ./...` green in the worktree; frontend builds.
- P2 client tests prove request shape + error + low-confidence paths (verify-the-probe).
- Grep-clean: no `aimlapi`, `grok`, `fly.dev`, `FLY_OCR_URL`, `ASYMM_API_KEY` (OCR),
  `pixtral` references left in live code (docs/history exempt).
- Fonts: three sample PDFs generated (invoice, costing/offer, Arabic content), rendered
  correctly; binary-size delta reported.
- Payslip PDF generated from synthetic fixture.
- Wave report `FABLE_WAVE13_REPORT.md` in repo root: what shipped per mission, A0 API ground
  truth, deviations, severity-honest issue list, and "what the next wave can assume."
