# CODEX WAVE 14 HANDOFF — Svelte 5 + Go Service Architecture

**Project**: AsymmFlow (asymmflow)
**Wave**: 14 of 16
**Depends On**: Wave 13 (complete — commit `ee166f0`)
**Module**: `ph_holdings_app`

---

## RULES (READ FIRST)

1. **`go build ./...` and `go test ./... -count=1 -timeout 300s` MUST pass after EVERY Go ticket.**
2. **`cd frontend && npm run build` MUST pass after EVERY frontend ticket.**
3. **Commit after EACH ticket** with message format: `feat(codex): <description> (Wave 14, Ticket N)`
4. **NO behavior changes.** This wave is a PURE structural migration. Every method must do exactly what it did before. If a test breaks, the migration introduced a bug — fix the migration, not the test.
5. **STOP CONDITIONS**: If Wails v3 is not installable or breaks the build on Windows, skip Wails v3 tickets and document the issue. If the Svelte migration tool produces unrecoverable errors across >20 components, stop and document. If any ticket cannot be completed after 3 attempts, stop and write what went wrong.
6. **ELEVATED READ ACCESS**: You can read files across `C:\Projects\` for reference patterns.
7. **DO NOT modify domain packages** (`pkg/finance/`, `pkg/crm/`, `pkg/butler/`, `pkg/documents/`, `pkg/sync/`, `pkg/infra/`). These are stable. Only modify root-level `.go` files, `main.go`, `internal/`, and `frontend/`.

---

## CONTEXT: What Exists After Wave 13

### Current Architecture

```
main.go          → wails.Run(), binds single App struct
app.go           → App struct (30+ fields: db, services, cache, etc.)
*_service.go     → 1,196 methods on *App (ALL exposed as Wails bindings)
frontend/        → 186 Svelte 4 components, 99K LOC
  └── src/
      ├── lib/          → Shared components, stores
      ├── routes/       → Page components
      └── wailsjs/go/   → Generated Wails bindings (import from main/App)

internal/viewmodel/  → Wave 11 MVVM layer (76 types, 7 builders)
```

### Current Wails v2 Binding

```go
// main.go
Bind: []interface{}{
    app,   // ALL 1,196 methods exposed
}
```

```typescript
// frontend — current import pattern
import { CreateInvoice } from '../wailsjs/go/main/App';
```

### Method Distribution Across Services

Source files → target service mapping (approximate):

| Target Service | Source Files | ~Methods |
|---------------|-------------|----------|
| **FinanceService** | customer_invoice_service, supplier_invoice_service, expense_service, payroll_service, cheque_register_service, app_accounting_inventory, bank_*, payment_*, credit_note_*, fx_rate_* | ~250 |
| **CRMService** | app_sales_pipeline, app_order_customer_surface, delivery_note_service, purchase_order_service, grn_service, app_graph_contract_surface, product_*, serial_*, customer_*, supplier_* | ~230 |
| **ButlerService** | butler_ai_context, butler_grounded_fastpath, chat_service, predictor, batch, payment_intelligence | ~100 |
| **DocumentsService** | app_setup_documents_surface, ocr_*, document_classifier, *_pdf_service, annexure_*, excel_*, msg_parser | ~120 |
| **SyncService** | collaboration_service, sync_service_impl, db_manager (sync methods only) | ~70 |
| **InfraService** | app_auth_rbac, license_service, settings_service, email_service, file_watcher, phase7_rollout, app.go lifecycle | ~100 |
| **App (retained)** | startup, shutdown, config, db init, remaining glue | ~20 |

### Frontend Dependencies

| Metric | Value |
|--------|-------|
| Svelte version | 4.2.0 |
| Svelte components | 186 |
| Frontend LOC | ~99K |
| Files using stores (writable/readable/derived) | 13 |
| Files using on:event syntax | 113 |
| Wails binding package | `@wailsapp/wails/v2` |

---

## WAVE 14 STRATEGY

This wave has two INDEPENDENT tracks that can be done in sequence:

**Track A (Go Backend — Tickets 1-4)**: Create per-domain service structs that delegate to App. Bind alongside App. Regenerate Wails bindings. Zero frontend breakage.

**Track B (Svelte Frontend — Tickets 5-8)**: Upgrade to Svelte 5 using the automated migration tool. Update event handlers. Fix edge cases.

**Track C (Conditional — Ticket 9)**: If Wails v3 is available and stable on Windows, upgrade. If not, document and defer.

Both tracks are independently valuable. Even if Wails v3 is not available, Svelte 5 + service architecture is a major improvement.

---

## WAVE 14 TICKETS

### Ticket 1: Platform Validation + Dependency Upgrade

Research and validate available platform upgrades.

**Steps:**
1. Check if Wails v3 is available:
   ```powershell
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest 2>&1
   ```
   Document result: available/not available, version number.

2. Check Svelte 5 migration tool:
   ```powershell
   cd frontend
   npx sv migrate svelte-5 --help 2>&1
   ```
   If `sv` not available, try `npx svelte-migrate svelte-5 --help`.
   Document which tool is available.

3. Check if `pretext` Go library exists:
   ```powershell
   go list -m github.com/nicholasgasior/pretext@latest 2>&1
   ```
   (Or search for other Pretext PDF libraries.) Document result.

4. Write findings to `docs/WAVE14_PLATFORM_VALIDATION.md` with:
   - Wails v3: available/not, version, Windows compatibility notes
   - Svelte migration tool: available/not, name, version
   - Pretext: available/not, alternative approaches
   - Recommendation: which conditional tickets to execute

**Commit**: `feat(codex): validate platform availability (Wave 14, Ticket 1)`

---

### Ticket 2: Go Service Layer — Create Domain Services

**Files**: Create new files in `internal/services/`

Create 6 service structs, each holding a pointer to `*App` for delegation.

```go
// internal/services/finance_service.go
package services

type FinanceService struct {
    app interface{} // Use interface to avoid import cycle; cast in methods
}

func NewFinanceService(app interface{}) *FinanceService {
    return &FinanceService{app: app}
}
```

**IMPORTANT**: The service files MUST be in `package main` (same package as App) because they need to call App methods directly. Put them in the root directory alongside the existing `*_service.go` files.

**Files to create** (all in root, `package main`):

| File | Service | Delegates Methods From |
|------|---------|----------------------|
| `service_finance.go` | `FinanceService` | customer_invoice_service, supplier_invoice_service, expense_service, payroll_service, cheque_register_service, app_accounting_inventory, bank_*, payment_*, credit_note_*, fx_rate_* |
| `service_crm.go` | `CRMService` | app_sales_pipeline, app_order_customer_surface, delivery_note_service, purchase_order_service, grn_service, app_graph_contract_surface, product_*, serial_*, customer_*, supplier_* |
| `service_butler.go` | `ButlerService` | butler_ai_context, butler_grounded_fastpath, chat_service, predictor, batch, payment_intelligence |
| `service_documents.go` | `DocumentsService` | app_setup_documents_surface, ocr_*, document_classifier, *_pdf_service, annexure_*, excel_*, msg_parser |
| `service_sync.go` | `SyncService` | collaboration_service, sync_service_impl, db_manager sync methods |
| `service_infra.go` | `InfraService` | app_auth_rbac, license_service, settings_service, email_service, file_watcher, phase7_rollout |

**Each service struct pattern:**
```go
// service_finance.go
package main

// FinanceService exposes finance-domain Wails bindings.
type FinanceService struct {
    app *App
}

func NewFinanceService(app *App) *FinanceService {
    return &FinanceService{app: app}
}

// --- Customer Invoice methods ---

func (s *FinanceService) CreateCustomerInvoice(data map[string]interface{}) (interface{}, error) {
    return s.app.CreateCustomerInvoice(data)
}

func (s *FinanceService) GetCustomerInvoice(id string) (interface{}, error) {
    return s.app.GetCustomerInvoice(id)
}

// ... all other finance methods delegate to s.app.MethodName()
```

**How to build the delegation list:**
1. `grep "^func (a \*App)" <source_file>.go` to get all method signatures
2. Create a corresponding method on the service struct that calls `s.app.<method>(args...)`
3. Preserve EXACT signatures (same param names, same return types)

**This is PURE mechanical delegation. No logic changes. No param modifications.**

**To determine which file maps to which service**, use the source file mapping table above. If unsure about a method, put it in InfraService (catch-all).

**Test**: `go build ./...` passes. The new services exist but aren't bound yet.

**Commit**: `feat(codex): create domain service delegation layer (Wave 14, Ticket 2)`

---

### Ticket 3: Bind Services in Wails + Regenerate Bindings

**File**: Modify `main.go`

**Step 1**: Update main.go to create and bind all services:

```go
func main() {
    app := NewApp()

    // Create domain services
    financeService := NewFinanceService(app)
    crmService := NewCRMService(app)
    butlerService := NewButlerService(app)
    documentsService := NewDocumentsService(app)
    syncService := NewSyncServiceBinding(app)  // avoid collision with existing SyncService type
    infraService := NewInfraService(app)

    err := wails.Run(&options.App{
        // ... existing config unchanged ...
        Bind: []interface{}{
            app,             // Keep for backwards compatibility
            financeService,
            crmService,
            butlerService,
            documentsService,
            syncService,
            infraService,
        },
    })
}
```

**Step 2**: Regenerate Wails bindings:
```powershell
wails generate module
```

**Step 3**: Verify the generated bindings now include both `App` and all 6 service namespaces in `frontend/wailsjs/go/main/`.

**Test**: `go build ./...` passes. Frontend has new binding files.

**Commit**: `feat(codex): bind domain services in Wails (Wave 14, Ticket 3)`

---

### Ticket 4: Frontend Service Import Migration

**Files**: All files in `frontend/src/` that import from `wailsjs/go/main/App`

Replace imports from `App` to the appropriate domain service.

**Migration pattern:**
```typescript
// BEFORE:
import { CreateCustomerInvoice, GetCustomerInvoice } from '../../wailsjs/go/main/App';

// AFTER:
import { CreateCustomerInvoice, GetCustomerInvoice } from '../../wailsjs/go/main/FinanceService';
```

**How to determine which service:**
- Finance methods (invoice, payment, expense, payroll, bank, cheque, accounting, fx, credit note) → `FinanceService`
- CRM methods (customer, supplier, offer, order, pipeline, delivery, PO, GRN, serial, product, contract) → `CRMService`
- Butler methods (chat, butler, predict, conversation, intent) → `ButlerService`
- Document methods (OCR, PDF, classify, template, excel, msg, annexure, document) → `DocumentsService`
- Sync methods (collaboration, sync, db_sync) → `SyncService`
- Infra methods (auth, login, logout, role, permission, license, settings, email, file, phase7) → `InfraService`
- Anything unclear → keep as `App` import (safe — App still bound)

**Process:**
1. Find all files importing from `wailsjs/go/main/App`
2. For each import, categorize the method names into services
3. Split the import into per-service imports
4. If a file imports methods from multiple services, use multiple import lines

**IMPORTANT**: Some generated binding files might export both the function AND a type (e.g., `App.Invoice`). Types/models stay in the App namespace or a models file. Only FUNCTION imports move to services.

**Verify**: `cd frontend && npm run build` passes after all imports are updated.

**Commit**: `feat(codex): migrate frontend imports to domain services (Wave 14, Ticket 4)`

---

### Ticket 5: Svelte 5 Package Upgrade

**File**: `frontend/package.json`, `frontend/svelte.config.js`, `frontend/vite.config.ts`

**Step 1**: Update Svelte dependencies:
```powershell
cd frontend
npm install svelte@5 @sveltejs/vite-plugin-svelte@latest svelte-check@latest svelte-preprocess@latest --save-dev
```

If `svelte-preprocess` is no longer needed in Svelte 5 (Svelte 5 handles preprocessing natively), remove it.

**Step 2**: Update vite config if needed. Svelte 5 might require `@sveltejs/vite-plugin-svelte` v4+.

**Step 3**: Update `svelte.config.js` if the plugin API changed.

**Step 4**: Run `npm run build` to verify the build passes. If there are errors, fix the config (NOT the components — component migration is Ticket 6).

If the build has thousands of errors because Svelte 5 doesn't support v4 syntax: that's expected. Proceed to Ticket 6 (migration tool). Just ensure the toolchain is installed and configured correctly.

**Commit**: `feat(codex): upgrade Svelte 5 packages (Wave 14, Ticket 5)`

---

### Ticket 6: Svelte 5 Automated Migration

Run the Svelte 5 migration tool to convert Svelte 4 syntax to Svelte 5.

**Step 1**: Run the migration tool (use whichever was found in Ticket 1):
```powershell
cd frontend
npx sv migrate svelte-5
```
Or:
```powershell
npx svelte-migrate svelte-5
```

**Step 2**: Review what the tool changed:
```powershell
git diff --stat
```

**Step 3**: Run `npm run build` to check for remaining errors.

**The migration tool should handle:**
- `on:click` → `onclick` (and all other event handlers)
- `$:` reactive statements → `$derived()` / `$effect()`
- `export let prop` → `let { prop } = $props()`
- `$$props` / `$$restProps` → `$props()`
- Slot-based composition → snippet syntax (where applicable)
- Store auto-subscriptions may need manual review

**Step 4**: If the tool is NOT available or fails, do the migration manually for the most critical patterns:

Priority 1 (113 files): `on:click` → `onclick`, `on:change` → `onchange`, etc.
Priority 2 (13 files): `writable()` → `$state()`, `derived()` → `$derived()`
Priority 3: `$:` blocks → `$derived()` / `$effect()`

Use find-and-replace across all `.svelte` files for the mechanical patterns.

**Commit**: `feat(codex): run Svelte 5 automated migration (Wave 14, Ticket 6)`

---

### Ticket 7: Svelte 5 Manual Fixes

Fix any remaining build errors after the automated migration.

**Common issues to check:**
1. TypeScript errors from changed prop signatures
2. `$$props` / `$$restProps` not converted
3. Slot → snippet conversion issues
4. Component lifecycle changes (`onMount` should still work, but check)
5. Store subscriptions with `$store` syntax — should auto-convert, but verify
6. Transition directives — Svelte 5 uses the same syntax, should be fine
7. `bind:this` — should work unchanged
8. Custom action syntax — should work unchanged

**Process:**
1. Run `npm run build` and collect all errors
2. Fix errors one by one, grouped by error type
3. After each batch of fixes, run `npm run build` again
4. Repeat until build passes

**Also check**: `npm run check` (svelte-check) for type errors.

**Commit**: `feat(codex): fix Svelte 5 migration edge cases (Wave 14, Ticket 7)`

---

### Ticket 8: Wails v3 Migration (CONDITIONAL)

**This ticket is CONDITIONAL on Ticket 1 validation.**

**IF Wails v3 is available AND stable on Windows:**

1. Update go.mod:
   ```powershell
   go get github.com/wailsapp/wails/v3
   ```

2. Refactor main.go from Wails v2 to v3 API:
   ```go
   // BEFORE (v2):
   wails.Run(&options.App{Bind: []interface{}{app, ...}})

   // AFTER (v3):
   application := wails.NewApplication(wails.ApplicationOptions{...})
   window := application.NewWebviewWindowWithOptions(...)
   window.RegisterService(financeService)
   window.RegisterService(crmService)
   // etc.
   application.Run()
   ```

3. Update embed directive if needed
4. Regenerate bindings with `wails3 generate bindings`
5. Update frontend binding imports if paths changed
6. Verify build + run

**IF Wails v3 is NOT available or NOT stable:**

1. Document in `docs/WAVE14_PLATFORM_VALIDATION.md` why it was skipped
2. The service layer from Tickets 2-3 is already Wails v3-READY
3. When v3 releases, the migration will be a simple main.go refactor
4. Skip this ticket, proceed to Ticket 9

**Commit**: `feat(codex): migrate to Wails v3 (Wave 14, Ticket 8)` OR `docs(codex): defer Wails v3 — not available (Wave 14, Ticket 8)`

---

### Ticket 9: Full Build + Integration Verification

**Verify the entire stack builds and works:**

```powershell
# Go backend
go build ./...
go test ./... -count=1 -timeout 300s

# Frontend
cd frontend
npm run build
npm run check

# Wails (full app build)
cd ..
wails build   # or wails3 build if v3
```

If the full Wails build fails but `go build` and `npm run build` pass independently, document the error. The Wails build may need specific environment setup.

**Also verify:**
- Generated binding files exist for all 6 services
- No TypeScript import errors in frontend
- Svelte 5 syntax passes svelte-check

**Commit**: `test(codex): verify full build after Wave 14 migration (Wave 14, Ticket 9)`

---

### Ticket 10: Progress Audit

**File**: `docs/WAVE14_PROGRESS.md`

Write the progress audit with:

1. **Commit table**: Ticket number, commit hash, summary
2. **Migration metrics**:
   - Service methods: how many methods per service
   - Frontend files migrated: Svelte 5 changes, import changes
   - Wails v3 status: migrated/deferred
   - Pretext status: available/deferred
3. **Svelte 5 migration stats**:
   - Files auto-migrated by tool
   - Files manually fixed
   - Remaining Svelte 4 patterns (if any)
4. **Breaking changes**: List any behavior changes (should be ZERO)
5. **Issues and deviations**: Document anything that didn't go as planned
6. **Final gate**: Confirm all build gates pass

Also update `docs/MASTER_PLAN.md`:
- Mark Wave 13 as `✅ DONE` with date and commit range
- Mark Wave 14 as `✅ DONE` (or `✅ PARTIAL` if Wails v3 deferred)

---

## DEPENDENCY GRAPH

```
Ticket 1 (validation) ─→ Ticket 2 (Go services) ─→ Ticket 3 (bind + bindings)
                                                   ─→ Ticket 4 (frontend imports)
Ticket 1              ─→ Ticket 5 (Svelte 5 packages) ─→ Ticket 6 (auto migration)
                                                        ─→ Ticket 7 (manual fixes)
Ticket 1              ─→ Ticket 8 (Wails v3 — conditional)
All                   ─→ Ticket 9 (verification)
All                   ─→ Ticket 10 (audit)
```

**Recommended execution order**: 1 → 2 → 3 → 5 → 6 → 7 → 4 → 8 → 9 → 10

(Do Svelte upgrade BEFORE import migration so both changes happen in one pass)

---

## QUALITY GATES

### Per-Ticket Gate (Go)
```powershell
go build ./...
go test ./... -count=1 -timeout 300s
```

### Per-Ticket Gate (Frontend)
```powershell
cd frontend
npm run build
```

### Wave Completion Gate
```powershell
go build ./...
go test ./... -count=1 -timeout 300s
cd frontend && npm run build && npm run check
```

---

## WHAT SUCCESS LOOKS LIKE

After Wave 14:

1. **Svelte 5 is live** — runes, modern reactivity, smaller bundle
2. **6 domain services** — FinanceService, CRMService, ButlerService, DocumentsService, SyncService, InfraService
3. **Frontend imports from services** — organized by domain, not one monolithic App
4. **Wails v3** (if available) — multi-window, multi-service binding
5. **Zero behavior changes** — every method does exactly what it did before
6. **All tests pass** — backend and frontend

The architecture is now:
```
Svelte 5 ($state runes) ←→ Wails Bindings ←→ DomainService ←→ App ←→ Domain Packages ←→ DB
     VIEW                    TRANSPORT          DELEGATION      GLUE       MODEL
```

The App struct becomes pure glue. Domain services are the binding contract. ViewModels (Wave 11) remain the data shape. This is ready for:
- Wave 15: i18n + Compliance (add per-locale services)
- Wave 16: Nango Integration (add external API services)
- Future: Remove App delegation entirely (services call domain packages directly)
