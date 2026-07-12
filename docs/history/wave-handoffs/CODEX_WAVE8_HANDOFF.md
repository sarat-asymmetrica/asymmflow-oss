# Codex Autonomous Execution Spec — Wave 8: Butler AI + Documents Extraction

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Waves 0-7 complete. 54 commits. Banking FULLY COMPLETE. CGO eliminated (ncruces). 55 aliases, 35 remaining structs. Tests GREEN.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
**Disk space**: Use `$env:GOTMPDIR='D:\go-tmp'` and `$env:GOCACHE='D:\go-cache'` if C: is tight.

---

## 0. Context — The Tangled Domains

Waves 2-7 extracted the CLEAN domains (finance, CRM, procurement, fulfillment). This wave tackles the two TANGLED domains:

1. **Butler AI** — `butler_ai.go` (7,049 LOC), the largest service file. Deeply interleaved with app state: database queries, session management, permission checks, entity resolution. But it's also unique IP (prompt routing, grounded fastpath, payment prediction).

2. **Documents** — PDF generation (500-600 line functions), OCR (CGO via go-fitz), document classification, Excel parsing. Large but more mechanical than Butler.

**Strategy**: Same pattern as banking — define ports for external dependencies, move logic into packages, leave thin Wails wrappers. But with TWO important rules:
- **DO NOT refactor Butler internals** — move the code as-is, preserve exact behavior
- **DO NOT eliminate go-fitz CGO** — put OCR behind an interface, keep the CGO implementation for now

Read `docs/MASTER_PLAN.md` Wave 8 section for full source→target file mappings.

---

## 1. Governance

You are governed by:
- `.codex/AGENTS.md`
- `docs/OPERATING_PRINCIPLES.md`
- `docs/MASTER_PLAN.md` (Wave 8 section)

---

## 2. Tickets

### Dependency Graph

```
BUTLER TRACK:
  Ticket 1 (Butler ports) → Ticket 2 (Intent router) → Ticket 3 (Grounded fastpath)
                          → Ticket 4 (Reports)
                          → Ticket 5 (Chat persistence)
                          → Ticket 6 (Butler AI core — THE BIG ONE)
  Ticket 7 (Payment prediction) ── independent

DOCUMENTS TRACK:
  Ticket 8 (Document ports) → Ticket 9 (PDF services)
                            → Ticket 10 (Document classifier)
                            → Ticket 11 (OCR behind interface)
                            → Ticket 12 (Excel + Email parsers)

Ticket 13 (Remaining model aliases) ── independent
Ticket 14 (Progress audit) ── last
```

Butler and Documents tracks are INDEPENDENT. Execute Butler first (higher value), then Documents.

---

### Ticket 1: Define Butler Dependency Ports

**Deliverables**: Create `pkg/butler/ports.go` with interfaces for external dependencies:

```go
package butler

type DatabasePort interface {
    // Butler needs to query across domains for grounded responses
    QueryInvoices(filter map[string]interface{}) ([]map[string]interface{}, error)
    QueryCustomers(filter map[string]interface{}) ([]map[string]interface{}, error)
    QueryOrders(filter map[string]interface{}) ([]map[string]interface{}, error)
    QueryPayments(filter map[string]interface{}) ([]map[string]interface{}, error)
    QueryOffers(filter map[string]interface{}) ([]map[string]interface{}, error)
    RawQuery(sql string, args ...interface{}) ([]map[string]interface{}, error)
}

type UserContextPort interface {
    CurrentUserID() string
    CurrentUserName() string
    CurrentDivision() string
    HasPermission(action string) bool
}

type LLMPort interface {
    ChatCompletion(systemPrompt, userMessage string, maxTokens int) (string, error)
    ChatCompletionWithHistory(messages []ChatMessage, maxTokens int) (string, error)
}

type AuditPort interface {
    LogAction(entityType, entityID, action, detail, userID string) error
}
```

Adapt these based on what `butler_ai.go` actually references from `*App`. Read the file first.

Create root-level port implementations in `app_butler_ports.go` that delegate to existing app state.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Ports cover all external dependencies Butler needs

**Commit**: `refactor(codex): define butler dependency ports`

---

### Ticket 2: Extract Intent Router

**Source**: `butler_intent_router.go` (563 LOC)
**Target**: `pkg/butler/intent/router.go`

This is a clean extraction — the intent router classifies user messages and routes them to appropriate handlers. It has the `calculateWeightedButlerPipeline` function which is unique IP.

**Preserve exactly**: The weighted pipeline logic, intent categories, and routing thresholds.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All intent routing logic in `pkg/butler/intent/`
- [ ] Pipeline weights preserved exactly

**Commit**: `refactor(codex): extract intent router to pkg/butler/intent`

---

### Ticket 3: Extract Grounded Fastpath

**Source**: `butler_grounded_fastpath.go` (1,796 LOC)
**Target**: `pkg/butler/fastpath/grounded.go`

Grounded fastpath provides DB-backed quick answers without calling the LLM. Uses DatabasePort to query domain data. This is high-value unique logic.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All grounded response logic in `pkg/butler/fastpath/`
- [ ] Database queries go through DatabasePort, not direct `a.db`

**Commit**: `refactor(codex): extract grounded fastpath to pkg/butler/fastpath`

---

### Ticket 4: Extract Butler Reports

**Source**: `butler_reports.go` (793 LOC)
**Target**: `pkg/butler/reports/generator.go`

Report generation is relatively clean — it queries data and formats it for Butler responses.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): extract butler reports to pkg/butler/reports`

---

### Ticket 5: Extract Chat Persistence

**Source**: `chat_service.go` (includes `ChatWithButlerPersistent` at 331 lines)
**Target**: `pkg/butler/persistence/service.go`

Chat persistence handles session management, message storage, conversation history. The `ChatWithButlerPersistent` function is the main entry point.

**Key**: This function orchestrates: load history → classify intent → route to handler/LLM → save response → return. Move the ORCHESTRATION into the package, with external calls going through ports.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Chat history CRUD in `pkg/butler/persistence/`
- [ ] Session management in package

**Commit**: `refactor(codex): extract chat persistence to pkg/butler/persistence`

---

### Ticket 6: Extract Butler AI Core

**Source**: `butler_ai.go` (7,049 LOC) — THE BIG ONE
**Target**: `pkg/butler/chat/service.go`

**Strategy**: This is the LARGEST single extraction. Break it into sub-batches:

1. **Sub-batch A**: Move helper functions (prompt builders, response formatters, context assemblers) that DON'T reference `*App` state. These are the easiest.

2. **Sub-batch B**: Move LLM-calling functions. Replace `a.db` with `DatabasePort`, `a.getCurrentUser()` with `UserContextPort`, Sarvam API calls with `LLMPort`.

3. **Sub-batch C**: Move the remaining orchestration functions. These are the ones that tie everything together.

Run `go build` after each sub-batch. If a function is TOO entangled to extract cleanly (>3 attempts), leave it in root with a `// TODO: extract after port enrichment` comment.

**CRITICAL**: Do NOT refactor the prompt construction logic. Move it AS-IS. The prompts are tuned for Sarvam 105B and Acme Instrumentation's specific context. Changing them risks degrading AI quality.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Majority of butler_ai.go logic moved to `pkg/butler/chat/`
- [ ] Prompt construction preserved exactly
- [ ] Entity resolution preserved exactly
- [ ] Functions left behind documented with reason
- [ ] `butler_ai.go` reduced to <2,000 LOC (ideally <1,000)

**Commit**: `refactor(codex): extract butler AI core to pkg/butler/chat`

---

### Ticket 7: Extract Payment Prediction

**Source**: `predictor.go` (328 LOC) + `payment_intelligence.go` (83 LOC) + `batch.go` (200 LOC)
**Target**: `pkg/butler/prediction/`

These files are relatively independent. Move them as-is. The quaternion-based prediction algorithm is unique IP — preserve EXACTLY.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All prediction logic in `pkg/butler/prediction/`
- [ ] Williams batching logic preserved
- [ ] ThreeRegime struct usage preserved

**Commit**: `refactor(codex): extract payment prediction to pkg/butler/prediction`

---

### Ticket 8: Define Document Dependency Ports

**Deliverables**: Create `pkg/documents/ports.go`:

```go
package documents

type StoragePort interface {
    SaveFile(path string, data []byte) error
    ReadFile(path string) ([]byte, error)
    FileExists(path string) bool
}

type ConfigPort interface {
    GetExportDir() string
    GetCompanyInfo() CompanyInfo
    GetDivisionBranding(division string) BrandingConfig
}

type FinanceDataPort interface {
    GetInvoiceForPDF(id string) (interface{}, error)
    GetOfferForPDF(id string) (interface{}, error)
    GetPurchaseOrderForPDF(id string) (interface{}, error)
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): define document dependency ports`

---

### Ticket 9: Extract PDF Services

**Source**: `invoice_pdf_service.go` + `offer_pdf_service.go` + `purchase_order_pdf_service.go` + `pdf_generator.go`
**Target**: `pkg/documents/pdf/`

**IMPORTANT**: These functions are LONG (500-600 lines each). Move them AS-IS without refactoring. They work correctly for Acme Instrumentation and will be regenerated with Pretext in a later wave. The goal is to move ownership, not improve code quality.

Move each PDF service to its own file in `pkg/documents/pdf/`:
- `pkg/documents/pdf/invoice.go` — GenerateInvoicePDF (612 lines)
- `pkg/documents/pdf/offer.go` — Offer PDF functions
- `pkg/documents/pdf/purchase_order.go` — GeneratePurchaseOrderPDF (607 lines)
- `pkg/documents/pdf/generator.go` — Shared PDF helpers

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All PDF generation in `pkg/documents/pdf/`
- [ ] Arabic RTL support preserved
- [ ] Division-specific branding preserved

**Commit**: `refactor(codex): extract PDF services to pkg/documents/pdf`

---

### Ticket 10: Extract Document Classifier

**Source**: `document_classifier.go` (1,302 LOC)
**Target**: `pkg/documents/classifier/service.go`

The classifier uses both rule-based and AI-based classification. Move the logic, use `LLMPort` for AI classification calls.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Classification logic in `pkg/documents/classifier/`

**Commit**: `refactor(codex): extract document classifier to pkg/documents/classifier`

---

### Ticket 11: Isolate OCR Behind Interface

**Source**: `ocr_service_simple.go` (1,821 LOC)
**Target**: `pkg/documents/ocr/`

**IMPORTANT**: OCR depends on `go-fitz` which is CGO. Do NOT try to eliminate this CGO dependency. Instead:

1. Define an `OCREngine` interface in `pkg/documents/ocr/engine.go`
2. Create `pkg/documents/ocr/fitz.go` — the go-fitz implementation (wraps existing code)
3. The interface allows future swap to Wasm-based OCR or external OCR service

```go
type OCREngine interface {
    ExtractText(pdfPath string) (string, error)
    ExtractPages(pdfPath string) ([]PageContent, error)
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] OCR code in `pkg/documents/ocr/`
- [ ] go-fitz dependency isolated behind OCREngine interface
- [ ] No go-fitz imports outside of `pkg/documents/ocr/fitz.go`

**Commit**: `refactor(codex): isolate OCR behind interface in pkg/documents/ocr`

---

### Ticket 12: Extract Excel + Email Parsers

**Source**: `excel_costing_parser.go` + `excel_template_generator.go` + `msg_parser.go` + `annexure_extractor.go` + `pdf_data_extractor.go`
**Target**: `pkg/documents/excel/` + `pkg/documents/email/` + `pkg/documents/extraction/`

Smaller, cleaner extractions. Good finishing ticket for the Documents track.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): extract Excel, email, and extraction services to pkg/documents`

---

### Ticket 13: Alias Remaining Models + Cleanup

**Scan `database.go`** for the remaining 35 struct definitions. Alias any that haven't been moved yet:

Priority targets:
- Auth/session models → `pkg/infra`
- Sync/collaboration models → `pkg/sync`
- Any remaining finance/CRM models
- Chat/conversation models → `pkg/butler`
- Payroll models → `pkg/finance`
- Activity monitoring models → `pkg/infra`

Skip models that have deep entanglement with startup/migration code. Document skipped models and why.

**Target**: database.go < 20 remaining struct definitions.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] database.go structs reduced to <20

**Commit**: `refactor(codex): alias remaining domain models`

---

### Ticket 14: Wave 8 Progress Audit

**Deliverables**:
1. `butler_ai.go` LOC (target: <2,000, ideally <1,000)
2. `pkg/butler/` total LOC
3. `pkg/documents/` total LOC
4. database.go remaining structs count
5. database.go alias count (target: 70+)
6. Total `func (a *App)` across root files
7. go-fitz CGO isolated? (yes/no)
8. What remains for Wave 9+ (Cap'n Proto, Math Framework, MVVM, etc.)
9. Write `docs/WAVE8_PROGRESS.md`

**Commit**: `docs(codex): write wave 8 progress report`

---

## 3. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0

### Butler-Specific Rules

- **Ticket 6 is the hardest extraction in the entire refactor.** If `butler_ai.go` has functions that are deeply entangled with 5+ app-state fields:
  - First try: extract with ports for all dependencies
  - Second try: extract with a broader `AppContext` port that wraps multiple concerns
  - Third try: leave the function in root with TODO comment
  - **Do not spend >20 minutes on a single function extraction**

- **Prompt strings**: Move them exactly as-is. Do not reformat, reword, or "improve" any system prompts or prompt templates. These are tuned.

### Documents-Specific Rules

- **PDF functions**: Move as-is. Do NOT restructure 600-line functions. They're ugly but they work. Future waves will regenerate them with Pretext.
- **go-fitz**: Keep behind interface. Do not try alternative OCR implementations. Just isolate.

---

## 4. Autonomy Contract

- Start with Ticket 1. Proceed in order.
- Butler track (Tickets 1-7) before Documents track (Tickets 8-12).
- Do NOT stop between tickets unless a STOP condition hits.
- STOP conditions: build fails after 3 fix attempts; test regression; disk full.
- If `butler_ai.go` extraction (Ticket 6) takes >45 minutes with limited progress, commit what you have and move to Ticket 7.
- Ticket 13 (remaining aliases) is stretch — do it if time/budget allows.

---

## 5. What NOT To Touch

- `pkg/finance/banking/` — COMPLETE, do not modify
- Frontend files — no Svelte changes
- `app.go` — already lifecycle shell, leave as-is
- Wails binding signatures — ALL `func (a *App)` method names must stay the same
- System prompt content — move but never edit prompt text

---

## 6. Expected Outcome

By end of this run:
- `butler_ai.go` reduced from 7,049 LOC to <2,000 LOC
- `pkg/butler/` is a real domain package with 5+ sub-packages (chat, intent, fastpath, reports, persistence, prediction)
- `pkg/documents/` is a real domain package with 5 sub-packages (pdf, classifier, ocr, excel, email, extraction)
- go-fitz CGO isolated behind OCREngine interface
- database.go < 20 remaining structs
- 70+ type aliases
- Payment prediction (unique IP) safely in pkg/butler/prediction/

---

## Sign-Off

The tangled domains are the last frontier. Butler is the BRAIN of AsymmFlow — handle it with the same precision you brought to banking. Documents are the HANDS — mechanical but essential.

After this wave, ALL six domain packages have real logic. The package-boundary refactor is architecturally COMPLETE. Everything after is schemas, framework upgrades, and integration wiring.

🔥 Execute. Butler brain first. Documents hands second. The compiler guides you.
