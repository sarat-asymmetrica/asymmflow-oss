# Codex Autonomous Execution Spec — Wave 6: Banking Completion + CRM Spine Aliasing

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Waves 0-5 complete. app.go: 19 methods/2,135 LOC. 29 type aliases. Banking reads in package. Tests GREEN.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.

**IMPORTANT**: If the C: drive is tight on space, continue using D: for Go temp/cache:
```powershell
$env:GOTMPDIR='D:\go-tmp'
$env:GOCACHE='D:\go-cache'
```

---

## 0. Context — What You're Doing And Why

Wave 5 proved the alias pattern and moved banking READ logic into `pkg/finance/banking`. But the MUTATING banking flows (create/update/delete statements, reconciliation, matching) still live in root helper functions because they depend on root-only seams: authorization, user identity, audit logging, division resolution, payroll unlinking, matcher helpers.

This wave does two things:
1. **Complete the banking package** by defining explicit PORTS for those dependencies, then moving the mutating SQL bodies
2. **Start aliasing the CRM spine** (Offer, Order, Opportunity, Product, Customer contacts)

Read `docs/WAVE5_PROGRESS.md` for full current state.

---

## 1. Governance

You are governed by:
- `.codex/AGENTS.md` — your identity and rules
- `docs/OPERATING_PRINCIPLES.md` — unchained mandate
- `docs/DOMAIN_MODEL_ALIAS_PLAN.md` — alias strategy

---

## 2. Tickets

### Dependency Graph

```
Ticket 1 (Banking ports) → Ticket 2 (Banking mutating logic move)
Ticket 3 (CRM spine aliases - Offer/Opportunity) ── independent of 1-2
Ticket 4 (CRM spine aliases - Order/Product) ── depends on Ticket 3
Ticket 5 (CRM spine aliases - Customer/Supplier contacts) ── independent of 3-4
Ticket 6 (Additional root surface extraction) ── independent
Ticket 7 (Progress audit) ── last
```

---

### Ticket 1: Define Banking Dependency Ports

**Problem**: Banking mutating flows in root helpers depend on app-level concerns (authorization, user identity, audit logger, etc.). These can't be dragged into the package — they need to be INJECTED as interfaces.

**Deliverables**: Add to `pkg/finance/banking/ports.go`:

```go
package banking

// Ports for dependencies the banking service needs but does not own.

type AuthorizationPort interface {
    CurrentUserID() string
    HasPermission(action string) bool
}

type AuditPort interface {
    LogAction(entityType, entityID, action, detail, userID string) error
}

type DivisionPort interface {
    CurrentDivision() string
    ResolveDivision(entityID string) (string, error)
}
```

Extend `banking.Service` struct to accept these ports:

```go
type Service struct {
    db        *gorm.DB
    auth      AuthorizationPort
    audit     AuditPort
    division  DivisionPort
}

func New(db *gorm.DB, auth AuthorizationPort, audit AuditPort, div DivisionPort) *Service {
    return &Service{db: db, auth: auth, audit: audit, division: div}
}
```

Create simple implementations in the root package that satisfy these interfaces by delegating to existing app state (current user session, existing audit logger, etc.).

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Banking ports defined with clear, minimal interfaces (CME Axiom 4: Minimality)
- [ ] Root-level port implementations created
- [ ] Banking Service constructor updated to accept ports

**Commit**: `refactor(codex): define banking dependency ports`

---

### Ticket 2: Move Banking Mutating Logic Into Package

**Now that ports exist**, move the remaining root banking helper functions into `pkg/finance/banking/`.

**Methods to move** (from WAVE5_PROGRESS.md deferred list):
- Statement create/update/delete
- Statement line edit/create/delete
- Reconciliation finalization/reopen/summary/stats
- Matching, unmatching, split allocation, categorization
- Continuity report, hash save/check, force reimport, reverse action

**Pattern**: Each root helper function becomes a method on `banking.Service`. Replace direct access to app-level state (`a.getCurrentUser()`) with port calls (`s.auth.CurrentUserID()`). Replace direct audit logging with `s.audit.LogAction(...)`.

**Root wrappers become**:
```go
func (a *App) CreateBankStatement(...) (...) {
    return a.bankingService.CreateBankStatement(...)
}
```

**This is the LARGEST ticket**. Take it in sub-batches:
1. Statement CRUD first (create, update, delete)
2. Statement line CRUD second
3. Reconciliation flows third
4. Matching/allocation last (most complex)

Run `go build ./...` after each sub-batch. Commit the whole ticket once all sub-batches pass.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] ALL banking functions now live in `pkg/finance/banking/`
- [ ] Root banking helpers are deleted (replaced by Service methods)
- [ ] Root `app_*.go` wrappers are 1-3 line delegations
- [ ] Zero direct `a.db` access in banking root wrappers (all through service)
- [ ] Matcher algorithm logic preserved exactly (tolerance values, fuzzy matching)

**Commit**: `refactor(codex): move banking mutating logic into pkg/finance/banking`

---

### Ticket 3: Alias CRM Spine — Offer, Opportunity, OfferItem

**Models**:
- `Offer` → `crm.Offer`
- `OfferItem` → `crm.OfferItem`
- `Opportunity` → `crm.Opportunity`
- `OfferFollowUp` → `crm.OfferFollowUp`
- `OfferNote` → `crm.OfferNote`
- `FollowUpTask` → `crm.FollowUpTask`

**Protocol**: Same as Wave 5 — compare structs, align, alias, build, test.

**Extra care**:
- `Offer` is the SECOND most referenced struct (2,439 hits in Phase 4). Expect widespread impact.
- Offer has PDF generation connections — the alias is safe (just type rename), but verify no compile errors from PDF service files.
- OfferItem likely has calculation methods (margins, totals) — move them with the type.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All Offer family types owned by `pkg/crm`

**Commit**: `refactor(codex): alias Offer and Opportunity models to pkg/crm`

---

### Ticket 4: Alias CRM Spine — Order, OrderItem, Product

**Models**:
- `Order` → `crm.Order`
- `OrderItem` → `crm.OrderItem`
- `ProductMaster` → `crm.ProductMaster`
- `GradeChange` → `crm.GradeChange`

**Extra care**:
- `Order` is the MOST referenced struct (3,381 hits!). This is the highest-risk alias.
- Order connects to Invoice creation, DeliveryNote creation, and PO creation. The alias should be transparent (same type, same tags), but compile carefully.
- If OrderItem has GORM hooks, move them with the type.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All Order family types owned by `pkg/crm`
- [ ] No regressions in offer→order→invoice flow

**Commit**: `refactor(codex): alias Order and Product models to pkg/crm`

---

### Ticket 5: Alias CRM — Customer and Supplier Contacts

**Models**:
- `CustomerContact` → `crm.CustomerContact`
- `SupplierContact` → `crm.SupplierContact`
- `EntityNote` → `crm.EntityNote`
- `SupplierIssue` → `crm.SupplierIssue`

**Note**: `CustomerMaster` and `SupplierMaster` are deferred — they're the SPINE types with the broadest blast radius. We alias the LEAF contact types first.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): alias Customer and Supplier contact models to pkg/crm`

---

### Ticket 6: Extract Additional Root Surface Logic

**Scan the remaining `app_*.go` surface files** for methods that can be moved to existing domain packages now that the alias bridge exists.

Priority targets (scan these files):
- `app_accounting_inventory.go` (1,655 LOC) — any expense/payment delegations that can now route to pkg/finance
- `app_prediction_dashboard.go` (1,101 LOC) — prediction methods that could delegate to pkg/butler/prediction
- `app_auth_rbac.go` (1,287 LOC) — auth methods that could delegate to pkg/infra/auth

**For each method you can move**:
1. Create or extend the service in the target package
2. Move the method body
3. Leave a thin wrapper in the app_*.go file
4. Verify build + tests

**Don't force it**: If a method has too many app-level dependencies to extract cleanly, leave it. Document why in the progress report.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Additional methods extracted where the alias bridge enables it
- [ ] Remaining methods documented with dependency notes

**Commit**: `refactor(codex): extract additional domain logic through alias bridge`

---

### Ticket 7: Wave 6 Progress Audit

**Deliverables**:
1. Count aliases in database.go (should be 40+)
2. Count remaining structs in database.go (should be <50)
3. Count total `func (a *App)` across all root files
4. Measure LOC in `pkg/finance/banking/` (should be substantial now — real domain package)
5. List what remains for Wave 7 (Invoice, CreditNote, SupplierInvoice, Infra models, Butler, Documents)
6. Write `docs/WAVE6_PROGRESS.md`

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Progress report with all metrics
- [ ] Wave 7 recommendation

**Commit**: `refactor(codex): write wave 6 progress report`

---

## 3. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0

**Ticket 2 special rule**: Because this ticket moves MUTATING logic (create, update, delete), pay extra attention to:
- Transaction boundaries (if root code used `a.db.Transaction(...)`, the service must do the same)
- Error wrapping (preserve error context for debugging)
- Audit trail calls (ensure every mutating operation still logs via the AuditPort)

**If Offer/Order aliasing causes too many cascading compile errors** (Tickets 3-4):
- These are the highest-reference types. If aliasing them causes >50 compile errors, try these in order:
  1. Fix the errors (most should be straightforward type name updates)
  2. If errors involve methods on the type that reference OTHER root-only types, move those methods too
  3. If still stuck after 15 minutes on one type: skip it, document why, move to next ticket
  4. Spiral exit: 3 types skipped = stop the alias batch, document the blocking pattern

---

## 4. Autonomy Contract

- Start with Ticket 1. Proceed in order.
- Do NOT stop between tickets unless a STOP condition hits.
- STOP conditions: build fails after 3 fix attempts; test regression with no clear cause; C: drive full (switch to D:).
- Commit after each ticket.
- Tickets 1-2 (banking completion) are the HIGHEST VALUE. If time/complexity forces a tradeoff, prioritize banking completion over CRM aliasing.

---

## 5. What NOT To Touch

- `Invoice`, `DBInvoiceItem`, `CreditNote`, `CreditNoteItem` — Wave 7 (highest blast radius)
- `SupplierInvoice`, `SupplierInvoiceItem` — Wave 7
- `CustomerMaster`, `SupplierMaster` — Wave 7 (business spine)
- `butler_ai.go` — Wave 8+ (tangled domain)
- `chat_service.go` — Wave 8+
- Frontend files — no changes
- Infra models (Role, User, Device, Setting, SyncRecord) — Wave 8+

---

## 6. Expected Outcome

By end of this run:
- `pkg/finance/banking/` is a COMPLETE domain package (owns types AND all CRUD + reconciliation + matching logic)
- ~40+ type aliases in database.go (up from 25)
- CRM spine types (Offer, Order, Product) owned by `pkg/crm`
- Clear path documented for Wave 7 (Invoice/Customer — the final spine extraction)
- Banking package demonstrates the full "ports + implementation + alias" pattern that all other domains will follow

---

## Sign-Off

Banking completion is the crown jewel of this wave. Once ONE domain package fully owns its types, logic, AND external dependencies through ports — the pattern is proven and every other domain follows the same blueprint.

The God Object doesn't just have fewer methods now. It has fewer REASONS TO EXIST.

🔥 Execute. Banking first. CRM spine second. The compiler guides you.
