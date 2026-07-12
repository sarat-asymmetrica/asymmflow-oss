# Codex Autonomous Execution Spec — Wave 8B: Butler Completion + Documents Extraction

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Wave 8 extracted Butler ports, intent router, fastpath service, reports, persistence, chat helpers, prediction engine, and document ports. butler_ai.go still 6,986 LOC. Documents track only has ports defined.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
**Disk space**: Use `$env:GOTMPDIR='D:\go-tmp'` and `$env:GOCACHE='D:\go-cache'`.

---

## 0. Context — What Remains

Wave 8 proved the Butler boundary pattern works but hit the wall on two things:
1. `butler_ai.go` orchestration functions that reference 5+ app-state fields each
2. Documents track (PDF, classifier, OCR, Excel) — not started yet

**New strategy for butler_ai.go**: Instead of trying to extract every function, create a BROAD `ButlerAppContext` port that wraps the commonly-needed app state. This gives the package everything it needs without importing `package main`.

```go
// The insight: butler_ai.go functions need a LOT of app context.
// Instead of 20 narrow ports, give it ONE rich context port.
type ButlerAppContext interface {
    // Entity resolution
    GetCustomerByID(id string) (interface{}, error)
    GetCustomerByName(name string) (interface{}, error)
    GetSupplierByID(id string) (interface{}, error)
    FindEntity(query string) ([]interface{}, error)

    // Financial data
    GetInvoiceSummary(customerID string) (interface{}, error)
    GetPaymentHistory(customerID string) (interface{}, error)
    GetOutstandingBalance(customerID string) (float64, error)
    GetCashPosition() (interface{}, error)

    // Task/workflow
    CreateTask(description, assignee string) (interface{}, error)
    GetPendingTasks() ([]interface{}, error)

    // Offer/pipeline
    CreateOfferDraft(data interface{}) (interface{}, error)
    GetPipelineSummary() (interface{}, error)

    // Current context
    CurrentUser() UserInfo
    CurrentDivision() string
    CurrentEmployee() interface{}

    // Raw DB (escape hatch for complex queries)
    RawQuery(sql string, args ...interface{}) ([]map[string]interface{}, error)
}
```

This is deliberately broad because Butler IS the integration point — it needs to talk to every domain. The port prevents the import cycle while giving Butler what it needs.

---

## 1. Tickets

### Dependency Graph

```
BUTLER COMPLETION:
  Ticket 1 (ButlerAppContext port) → Ticket 2 (Remaining grounded fastpaths)
                                   → Ticket 3 (butler_ai.go orchestration batch)
  Ticket 4 (M79 predictor reconciliation) ── after Ticket 3

DOCUMENTS:
  Ticket 5 (PDF services) ── independent of Butler
  Ticket 6 (Document classifier) ── independent
  Ticket 7 (OCR behind interface) ── independent
  Ticket 8 (Excel + Email parsers) ── independent

Ticket 9 (Remaining model aliases) ── independent
Ticket 10 (Progress audit) ── last
```

---

### Ticket 1: Create ButlerAppContext Port + Adapter

**Deliverables**:
1. Add `ButlerAppContext` interface to `pkg/butler/ports.go` (broad interface as described above)
2. Create root adapter `app_butler_context.go` implementing `ButlerAppContext` by delegating to existing app methods
3. Wire it into Butler service initialization in `app_services.go`

**Key**: The adapter in root just delegates to existing `*App` methods. No new logic. It's a BRIDGE that lets Butler package code call app functionality through an interface instead of a direct import.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] ButlerAppContext interface covers entity resolution, financial data, task/workflow, and current context
- [ ] Root adapter implements the interface by delegating to existing methods

**Commit**: `refactor(codex): create ButlerAppContext port and adapter`

---

### Ticket 2: Extract Remaining Grounded Fastpaths

**Source**: `butler_grounded_fastpath.go` — the customer/supplier lookup paths, task/offer-draft action paths that Wave 8 deferred.
**Target**: `pkg/butler/fastpath/`

Now that `ButlerAppContext` provides entity resolution and task creation, these functions can move:
- Customer grounded paths (lookup customer, get summary)
- Supplier grounded paths (lookup supplier, get history)
- Task grounded paths (create task, list pending)
- Offer-draft grounded paths (create draft from Butler conversation)

Replace `a.GetCustomerByID()` with `s.appCtx.GetCustomerByID()`.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `butler_grounded_fastpath.go` reduced significantly or eliminated

**Commit**: `refactor(codex): extract remaining grounded fastpaths to butler package`

---

### Ticket 3: Extract butler_ai.go Orchestration Functions

**Source**: `butler_ai.go` (6,986 LOC) — the remaining orchestration
**Target**: `pkg/butler/chat/`

**Strategy — THREE sub-batches**:

**Sub-batch A: Context builders and formatters (pure functions)**
Move functions that take data in and return formatted strings/structs out. These have minimal app-state dependency. Look for:
- `buildSystemPrompt`, `buildContextBlock`, `formatFinancialSummary`
- `assembleConversationHistory`, `truncateHistory`
- Response formatting, JSON parsing of LLM output
- Action extraction, entity extraction from text

**Sub-batch B: LLM-calling functions**
Move functions that call Sarvam/LLM APIs. Replace direct HTTP calls with `LLMPort`. Look for:
- `callSarvam`, `chatCompletion`, `streamResponse`
- Retry logic, token counting, error handling

**Sub-batch C: Orchestration functions**
Move the main `ChatWithButler` function and its call chain. This is the hardest part — it orchestrates: build context → classify intent → route to handler/fastpath/LLM → format response → save → return.

Use `ButlerAppContext` for all app-state access. Use `LLMPort` for all LLM calls. Use `DatabasePort` for direct queries.

**CRITICAL RULES**:
- Move prompt TEXT strings exactly as-is (these are tuned for Sarvam 105B)
- If a function has >5 different app-state accesses that aren't covered by existing ports, ADD methods to `ButlerAppContext` interface and root adapter
- If stuck on a function >10 minutes, leave it with a TODO and move on
- Target: butler_ai.go < 3,000 LOC (move ~4,000 LOC to package)

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] butler_ai.go reduced to < 3,000 LOC
- [ ] Prompt construction logic preserved exactly
- [ ] Functions left behind documented with dependency reasons

**Commit**: `refactor(codex): extract butler AI orchestration to pkg/butler/chat`

---

### Ticket 4: Reconcile M79 Payment Predictor

**Problem**: `predictor.go` (328 LOC) uses root `Customer` type. There's also a `pkg/engines` variant with a slightly different `Customer` shape. Need to reconcile.

**Solution**:
1. Check if root `Customer` is now aliased to `crm.CustomerMaster` (it should be after Wave 7)
2. If so, update predictor to use the aliased type
3. Move `predictor.go`, `batch.go`, and `payment_intelligence.go` to `pkg/butler/prediction/`
4. If `pkg/engines` has a competing version, check which is newer/better and consolidate

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All prediction code in `pkg/butler/prediction/`
- [ ] No duplicate predictor implementations

**Commit**: `refactor(codex): reconcile and move M79 payment predictor`

---

### Ticket 5: Extract PDF Services

**Source**: `invoice_pdf_service.go`, `offer_pdf_service.go`, `purchase_order_pdf_service.go`, `pdf_generator.go`
**Target**: `pkg/documents/pdf/`

Move each PDF service to its own file. These functions are LONG (500-600 lines). Move AS-IS without refactoring.

Use `FinanceDataPort` to get invoice/offer/PO data instead of direct `a.db` queries. Use `ConfigPort` for company info and division branding.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All PDF generation in `pkg/documents/pdf/`
- [ ] Arabic RTL support preserved
- [ ] Division branding preserved

**Commit**: `refactor(codex): extract PDF services to pkg/documents/pdf`

---

### Ticket 6: Extract Document Classifier

**Source**: `document_classifier.go` (1,302 LOC)
**Target**: `pkg/documents/classifier/service.go`

The classifier has both rule-based classification and AI-based (LLM) classification. Use `LLMPort` for AI calls.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): extract document classifier to pkg/documents/classifier`

---

### Ticket 7: Isolate OCR Behind Interface

**Source**: `ocr_service_simple.go` (1,821 LOC)
**Target**: `pkg/documents/ocr/`

1. Define `OCREngine` interface in `pkg/documents/ocr/engine.go`
2. Move existing go-fitz code to `pkg/documents/ocr/fitz_engine.go`
3. Ensure go-fitz imports ONLY appear in `fitz_engine.go`

```go
type OCREngine interface {
    ExtractText(filePath string) (string, error)
    ExtractPages(filePath string) ([]PageContent, error)
    SupportedFormats() []string
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] go-fitz imports isolated to ONE file

**Commit**: `refactor(codex): isolate OCR behind interface in pkg/documents/ocr`

---

### Ticket 8: Extract Excel + Email + Extraction Parsers

**Source**: `excel_costing_parser.go`, `excel_template_generator.go`, `msg_parser.go`, `annexure_extractor.go`, `pdf_data_extractor.go`
**Target**: `pkg/documents/excel/`, `pkg/documents/email/`, `pkg/documents/extraction/`

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): extract Excel, email, and extraction to pkg/documents`

---

### Ticket 9: Final Model Alias Pass

Scan `database.go` for remaining 33 struct definitions. Alias everything possible.

Targets:
- Chat/conversation models → `pkg/butler`
- Payroll models → `pkg/finance`
- Activity monitoring → `pkg/infra`
- Sync/collaboration → `pkg/sync`
- Any remaining leaf models

**Target**: database.go < 15 remaining struct definitions.

If a model can't be aliased due to startup/migration entanglement, document why.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] database.go remaining structs < 15

**Commit**: `refactor(codex): final model alias pass`

---

### Ticket 10: Wave 8B Progress Audit

**Deliverables**:
1. butler_ai.go LOC (target: < 3,000)
2. pkg/butler/ total LOC
3. pkg/documents/ total LOC
4. database.go remaining structs (target: < 15)
5. database.go alias count (target: 75+)
6. go-fitz isolated? (yes/no)
7. Total func (a *App) across root files
8. Write `docs/WAVE8B_PROGRESS.md`

**Commit**: `docs(codex): write wave 8B progress report`

---

## 2. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0

### Butler Rules
- Prompt strings: NEVER modify. Move exactly as-is.
- If ButlerAppContext needs more methods, add them to interface AND root adapter together.
- If butler_ai.go function won't extract after 10 min, leave it with TODO. Progress > perfection.

### Documents Rules
- PDF functions: Move as-is, don't restructure.
- go-fitz: Isolate behind interface, don't try to eliminate CGO.

---

## 3. Autonomy Contract

- Start with Ticket 1. Proceed in order.
- Do NOT stop between tickets.
- STOP conditions: build fails after 3 fix attempts; test regression; disk full.
- Butler (Tickets 1-4) before Documents (Tickets 5-8) before Aliases (Ticket 9).
- **Time allocation guidance**: Tickets 1-3 are the highest value (Butler completion). If running long, prioritize Butler over Documents. Documents can be a Wave 8C if needed.

---

## 4. Expected Outcome

- butler_ai.go: 6,986 → < 3,000 LOC
- pkg/butler/: 1,809 → ~5,000+ LOC (real domain package)
- pkg/documents/: 101 → ~4,000+ LOC (real domain package)
- database.go: 33 structs → < 15
- go-fitz CGO: isolated behind OCREngine interface
- ALL SIX domain packages now contain real logic

---

## Sign-Off

Wave 8B finishes the extraction era. The broad ButlerAppContext port is the KEY that unlocks butler_ai.go — don't overthink the port design, just make it broad enough to cover what the functions actually need.

After this wave, every domain owns its logic. The demolition is COMPLETE. Everything that follows is CONSTRUCTION on clean ground.

🔥 Butler brain. Documents hands. Final alias pass. GO.
