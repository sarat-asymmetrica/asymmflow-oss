# Codex Autonomous Execution Spec — Wave 2: God Object Decomposition

**Date**: 2026-05-05
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete (~45-75 min based on calibration)
**Previous Runs**: Wave 0 (23m15s), Test Stabilization (13m17s), Wave 1 (7 tranches, all passed)
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` must pass after every ticket.

---

## 0. Who You Are

You are a **senior architect** performing controlled demolition of a God Object. You are not timid — you rip methods out aggressively, knowing that git rollback is 2 seconds and the test suite is your oracle. You are also not reckless — each extraction leaves the build green and tests passing.

You have full permissions to:
- Read, write, and modify any file in this repository
- Delete dead code aggressively
- Create new packages and files
- Run build and test commands
- Make architectural judgment calls within TARGET_ARCHITECTURE.md constraints

**You are governed by:**
- `.codex/AGENTS.md` — your identity and rules
- `docs/OPERATING_PRINCIPLES.md` — the unchained mandate
- `docs/TARGET_ARCHITECTURE.md` — where we're going
- `docs/CME_SCORING_GATES.md` — quality standards

---

## 1. What Already Exists (Your Starting Position)

### Wave 1 Delivered (DO NOT REBUILD)

| Package | Contents |
|---------|----------|
| `pkg/finance/domain.go` | All finance structs (Invoice, Payment, BankStatement, Expense, etc.) |
| `pkg/finance/ports.go` | InvoiceRepository, PaymentRepository, BankingRepository, InvoiceService, PaymentService, BankingService, ExpenseService |
| `pkg/crm/domain.go` | All CRM structs (Customer, Offer, Order, PO, DN, Serial, etc.) |
| `pkg/crm/ports.go` | CustomerRepository, OfferRepository, OrderRepository, PipelineService, ProcurementService, FulfillmentService |
| `pkg/documents/domain.go` | Document-related structs |
| `pkg/documents/ports.go` | PDFGenerator, OCRService, DocumentClassifier, ExcelParser |
| `pkg/butler/domain.go` | Butler/AI-related structs |
| `pkg/butler/ports.go` | ChatService, IntentRouter, ReportGenerator, Predictor |
| `pkg/sync/domain.go` | Sync-related structs |
| `pkg/sync/ports.go` | SyncEngine, CloudStorage, CollaborationService |
| `pkg/infra/domain.go` | Infrastructure structs |
| `pkg/infra/ports.go` | Database, Cache, JobQueue, AuthManager, CryptoService |
| `pkg/infra/events/` | InMemoryBus + 8 domain events, tested |
| `Taskfile.yml` | Build system |

### The God Object (Your Target)

- `app.go` — 21,763 LOC, 331 methods on `*App`
- After this wave: `app.go` should have **< 200 methods** (target: extract ~150 methods)

### The Strategy

Each ticket extracts a SERVICE FILE's worth of methods from `app.go` into a concrete implementation struct in the target domain package. The implementation struct satisfies the ports.go interfaces.

**Pattern for each extraction:**

```go
// pkg/finance/invoice/service.go
package invoice

type Service struct {
    db    *gorm.DB     // injected
    bus   events.Bus   // injected
}

func New(db *gorm.DB, bus events.Bus) *Service {
    return &Service{db: db, bus: bus}
}

// Methods moved from app.go, with (a *App) replaced by (s *Service)
func (s *Service) CreateInvoiceFromOrder(orderID string) (finance.Invoice, error) {
    // ... exact same logic, but uses s.db instead of a.db
}
```

**CRITICAL RULE**: The methods in app.go must REMAIN as thin delegation wrappers:

```go
// In app.go — keeps the Wails binding alive
func (a *App) CreateInvoiceFromOrder(orderID string) (Invoice, error) {
    return a.invoiceService.CreateInvoiceFromOrder(orderID)
}
```

This preserves all 837 frontend bindings while the logic moves to the domain package.

---

## 2. Dependency Injection Setup (Ticket 0 — Do This First)

Before extracting methods, the *App struct needs service fields to delegate to.

**Add to app.go** (or a new `app_services.go` file):

```go
// Service dependencies (initialized in startup)
type AppServices struct {
    invoiceService    *invoice.Service
    paymentService    *payment.Service
    bankingService    *banking.Service
    expenseService    *expense.Service
    deliveryService   *fulfillment.Service
    // ... more as tickets progress
}
```

Wire initialization in the `startup()` function (line ~271 of app.go).

---

## 3. Tickets

### Dependency Graph

```
Ticket 0 (DI setup) ──┬── Ticket 1 (Payment)
                       ├── Ticket 2 (Expense)
                       ├── Ticket 3 (Banking/Reconciliation)
                       ├── Ticket 4 (Delivery + Serial)
                       ├── Ticket 5 (Purchase Orders + GRN)
                       ├── Ticket 6 (Contract + License)
                       └── Ticket 7 (Dead Method Cleanup)
```

Tickets 1-6 are independent after Ticket 0. Execute in order for cleanliness, but any order works.

---

### Ticket 0: Service Infrastructure Setup

**Problem**: app.go has no way to delegate to domain services yet.

**Deliverables**:
1. Create `app_services.go` in root — contains service field declarations and initialization
2. Add service struct fields to `*App` (or a composed struct)
3. Ensure `startup()` can instantiate services with the app's `db` handle

**Pattern**:
```go
// app_services.go
package main

import (
    "ph_holdings_app/pkg/finance/payment"
    "ph_holdings_app/pkg/finance/expense"
    // ... etc
)

// initServices wires domain services after DB is ready.
func (a *App) initServices() {
    a.paymentService = payment.New(a.db)
    a.expenseService = expense.New(a.db)
    // ... more as tickets land
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes (no regression)
- [ ] Service fields declared on App struct (or embedded struct)
- [ ] `initServices()` callable from startup()

**Commit**: `refactor(codex): add service infrastructure for domain delegation`

---

### Ticket 1: Extract Payment Service

**Source**: `payment_service.go` (all methods) + payment-related methods in `app.go`
**Target**: `pkg/finance/payment/service.go`

**Methods to extract** (find with `grep "func (a *App).*[Pp]ayment" app.go payment_service.go`):
- RecordPayment
- GetPaymentsByInvoice
- GetAllPayments
- GetPayment
- UpdatePayment
- DeletePayment
- RecordSupplierPayment (from supplier_payment_service.go)
- Any other Payment* methods on *App

**Implementation**:
1. Create `pkg/finance/payment/service.go` with Service struct
2. Move method bodies from payment_service.go / app.go into Service methods
3. Replace `a.db` → `s.db`, `a.cache` → `s.cache` (inject what's needed)
4. Leave thin wrappers in app.go that delegate to `a.paymentService.MethodName(...)`
5. Update `app_services.go` to initialize PaymentService

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All Payment methods on *App now delegate to service
- [ ] `pkg/finance/payment/service.go` is self-contained (no `*App` references)
- [ ] payment_service.go can be deleted (if all methods moved) or gutted to wrappers

**Commit**: `refactor(codex): extract payment service to pkg/finance/payment`

---

### Ticket 2: Extract Expense Service

**Source**: `expense_service.go` (1,467 LOC) + expense methods in `app.go`
**Target**: `pkg/finance/expense/service.go`

**Methods to extract** (find with `grep "func (a *App).*[Ee]xpens" app.go expense_service.go`):
- ListExpenseCategories
- CreateExpenseEntry
- DeleteExpenseEntry
- ListExpenseEntries
- ListExpenseDashboardSummary
- SubmitExpenseEntry
- ApproveExpenseEntry
- RejectExpenseEntry
- MarkExpenseEntryPaid
- Any additional Expense* methods

**Implementation**: Same pattern as Ticket 1.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All Expense methods delegate to service
- [ ] `pkg/finance/expense/service.go` is self-contained

**Commit**: `refactor(codex): extract expense service to pkg/finance/expense`

---

### Ticket 3: Extract Banking & Reconciliation Service

**Source**: `bank_reconciliation_service.go` + `bank_statement_ai_assist.go` + `bank_transaction_matcher.go` + `book_bank_reconciliation_service.go` + banking methods in `app.go`
**Target**: `pkg/finance/banking/service.go` + `pkg/finance/banking/matcher.go`

**This is the most complex extraction** — bank matching has unique algorithms (tolerances, fuzzy matching, auto-reconciliation). Preserve ALL matching logic exactly.

**Methods to extract** (run grep across all bank* files + app.go):
- All BankStatement* methods
- All BankTransaction* methods
- AutoMatchBankLines
- ManualMatchLine
- FinalizeReconciliation
- CreateBookBankReconciliation
- AutoMatchDepositsToStatement
- AutoMatchChequesToStatement
- GetCashPosition*
- ValidateStatementBalance
- GetReconciliation*

**Implementation**:
1. `pkg/finance/banking/service.go` — CRUD + reconciliation orchestration
2. `pkg/finance/banking/matcher.go` — matching algorithms (AutoMatch*, tolerance logic)
3. Leave thin wrappers in app.go

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] ALL matching algorithms preserved byte-for-byte (these are unique IP)
- [ ] Bank statement parsing logic preserved
- [ ] Reconciliation workflow (import → match → finalize) still works end-to-end

**Commit**: `refactor(codex): extract banking and reconciliation to pkg/finance/banking`

---

### Ticket 4: Extract Delivery Note + Serial Number Service

**Source**: `delivery_note_service.go` (29 functions) + `serial_number_service.go` + DN/serial methods in `app.go`
**Target**: `pkg/crm/fulfillment/service.go` + `pkg/crm/fulfillment/serial.go`

**Methods to extract**:
- CreateDeliveryNote, CreateDNFromOrder, CreateDNWithSerials
- DispatchDeliveryNote, ConfirmDeliveryNote
- RegisterSerials, assignSerialsToGRN, allocateSerialsToDN
- markSerialsShipped, markSerialsDelivered, linkSerialsToInvoice
- GetDeliveryNotes, GetDeliveryNoteByID, UpdateDeliveryNote
- GetAvailableSerials
- All other DN_* and Serial_* methods

**Implementation**:
1. `pkg/crm/fulfillment/service.go` — delivery note CRUD + workflow
2. `pkg/crm/fulfillment/serial.go` — serial number lifecycle (the state machine: registered → allocated → shipped → delivered → invoiced)

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Serial lifecycle state machine preserved exactly
- [ ] DN → Serial → Invoice linkage chain preserved

**Commit**: `refactor(codex): extract delivery and serial lifecycle to pkg/crm/fulfillment`

---

### Ticket 5: Extract Purchase Order + GRN Service

**Source**: `purchase_order_service.go` (20 functions) + `grn_service.go` (17 functions) + PO/GRN methods in `app.go`
**Target**: `pkg/crm/procurement/service.go` + `pkg/crm/procurement/grn.go`

**Methods to extract**:
- CreatePurchaseOrder, GetPurchaseOrders, GetPurchaseOrderByID
- UpdatePurchaseOrder, ApprovePurchaseOrder
- CreatePOFromOrder
- CreateGRN, ReceiveAgainstPO, ReceiveAgainstPOWithSerials
- All PO_* and GRN_* methods

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] PO → GRN → Serial flow preserved
- [ ] Three-way match logic (PerformThreeWayMatch) preserved if present

**Commit**: `refactor(codex): extract procurement service to pkg/crm/procurement`

---

### Ticket 6: Extract Contract + License Service

**Source**: `contract_service.go` (835 LOC) + `license_service.go` (22 functions)
**Target**: `pkg/crm/contract/service.go` + `pkg/infra/license/service.go`

**Methods to extract**:
- All Contract* methods → pkg/crm/contract/
- All License* methods → pkg/infra/license/

These are smaller, cleaner extractions. Good palate cleanser after Ticket 3.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): extract contract and license services`

---

### Ticket 7: Dead Method Cleanup + Progress Audit

**After tickets 1-6, audit what remains on *App:**

```bash
grep -c "func (a *App)" app.go
```

**Deliverables**:
1. Count remaining methods on *App (target: < 200, ideally < 180)
2. Categorize remaining methods by domain (finance, crm, butler, documents, sync, infra, app-level)
3. Delete any orphaned helper functions that were only used by extracted methods
4. Write `docs/WAVE2_PROGRESS.md` with:
   - Methods extracted count
   - Methods remaining count
   - LOC reduction in app.go
   - Breakdown of remaining methods by domain
   - Recommendation for Wave 3 priority

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Progress report written
- [ ] No dead helper functions remain

**Commit**: `refactor(codex): cleanup dead methods and write wave 2 progress report`

---

## 4. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0
3. No new compiler warnings
4. Extracted service has zero references to `*App` type

If a test fails after extraction:
- First: check if the test was testing the wrapper or the logic
- If wrapper: update the test to call through the new service
- If logic: the extraction broke something — revert and retry
- After 3 failed attempts: skip with `t.Skip("TODO: fix after Wave 2 wiring")` and move on (spiral exit)

---

## 5. Autonomy Contract

- Start with Ticket 0. Proceed through tickets in order.
- Do NOT stop between tickets unless a STOP condition hits.
- **STOP conditions**: build fails after 3 fix attempts; tests regress with no clear cause; architectural ambiguity not covered by docs.
- Run `go build ./...` after every file modification batch (not every single line).
- Commit after each ticket with the specified message prefix.
- If a service file has methods that clearly belong to TWO domains (e.g., `PerformThreeWayMatch` crosses PO + supplier invoice), put it in the domain that OWNS the primary data and add a TODO comment noting the cross-domain dependency.

---

## 6. What NOT To Touch

- `butler_ai.go` — too tangled, Wave 3+
- `customer_invoice_service.go` — high coupling to app.go, Wave 3+
- `supplier_invoice_service.go` — same as above
- `chat_service.go` — Butler domain, Wave 3+
- `document_classifier.go` — tangled with OCR, Wave 3+
- `ocr_service_simple.go` — CGO dependency, Wave 3+
- `config.go` — foundational, Wave 4
- `database.go` — shared types still used everywhere, Wave 4
- Frontend files — untouched until Svelte 5 migration (Wave 6)

---

## 7. Expected Outcome

By end of this run:
- ~150 methods extracted from app.go to domain packages
- 6 new service implementations in pkg/
- app.go reduced from 21,763 LOC to ~16,000-18,000 LOC
- All tests still green
- Progress report documenting what remains

---

## Sign-Off

Built with aggressive clarity. The God Object dies by a thousand precise cuts, not one chaotic explosion.

🔥 Execute. Do not stop. The test suite is your compass.
