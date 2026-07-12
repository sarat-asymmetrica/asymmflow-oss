# Codex Autonomous Execution Spec — Wave 7: Banking Completion + Invoice Spine + CGO Elimination

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Waves 0-6 complete. 42 commits. database.go: 39 aliases, 51 remaining structs. Banking: 1,004 LOC real logic. Tests GREEN.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
**Disk space**: Use `$env:GOTMPDIR='D:\go-tmp'` and `$env:GOCACHE='D:\go-cache'` if C: is tight.

---

## 0. Context

Wave 6 moved banking statement CRUD and integrity helpers into `pkg/finance/banking`. Three banking areas remain deferred:
1. Matching/allocation (touches invoices, supplier invoices, expenses, payroll)
2. Reconciliation lifecycle (finalize, reopen, summary, stats)
3. `GetBalanceContinuityReport` (depends on `CompanyBankAccount`)

This wave completes all three, aliases the Invoice/CreditNote spine, and eliminates CGO by swapping `mattn/go-sqlite3` for `ncruces/go-sqlite3`.

Read `docs/WAVE6_PROGRESS.md` for current state.

---

## 1. Governance

You are governed by:
- `.codex/AGENTS.md`
- `docs/OPERATING_PRINCIPLES.md`
- `docs/DOMAIN_MODEL_ALIAS_PLAN.md`

---

## 2. Tickets

### Dependency Graph

```
Ticket 1 (Banking reconciliation lifecycle) ──┐
Ticket 2 (CompanyBankAccount port) ───────────┼── Ticket 3 (Banking matching/allocation)
                                              │
Ticket 4 (Invoice/CreditNote aliases) ────────── independent
Ticket 5 (SupplierInvoice aliases) ───────────── depends on Ticket 4
Ticket 6 (ncruces SQLite swap) ───────────────── independent of 1-5
Ticket 7 (Infra model aliases) ───────────────── independent
Ticket 8 (Progress audit) ───────────────────── last
```

---

### Ticket 1: Move Banking Reconciliation Lifecycle Into Package

**Methods to move into `pkg/finance/banking/`**:
- `ValidateStatementBalance`
- `FinalizeReconciliation`
- `ReopenReconciliation`
- `GetReconciliationSummary`
- `GetReconciliationStats`

These use the same port pattern from Wave 6 (AuthorizationPort, AuditPort). Add any additional ports needed (e.g., if `FinalizeReconciliation` needs to notify other systems, add a NotificationPort).

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All reconciliation lifecycle methods in banking service
- [ ] Root wrappers are 1-3 line delegations

**Commit**: `refactor(codex): move reconciliation lifecycle into banking service`

---

### Ticket 2: CompanyBankAccount Port + Continuity Report

**Problem**: `GetBalanceContinuityReport` depends on the root `CompanyBankAccount` type.

**Solution**: Either:
- Option A: Alias `CompanyBankAccount` to `finance.CompanyBankAccount` and move it
- Option B: Create a `BankAccountPort` interface in the banking package that provides the account data needed

Pick whichever is simpler. If `CompanyBankAccount` has few references, aliasing is cleaner. If it's heavily entangled, a port is safer.

Then move `GetBalanceContinuityReport` into the banking service.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Continuity report method in banking service

**Commit**: `refactor(codex): resolve bank account dependency and move continuity report`

---

### Ticket 3: Move Banking Matching/Allocation Into Package

**This is the most complex banking extraction.** These methods touch multiple domains:

- `AutoMatchBankLines` — matches bank lines to invoices, supplier invoices, expenses
- `ManualMatchLine` — same cross-domain references
- `UnmatchLine` — reverses a match
- `CreateSplitAllocation` — splits a bank line across multiple invoices

**Strategy**: The matching logic itself lives in `pkg/finance/banking/matcher.go`. Cross-domain references (Invoice, SupplierInvoice, Expense) should be accessed through READ-ONLY ports:

```go
// In pkg/finance/banking/ports.go
type InvoiceLookup interface {
    GetInvoiceByID(id string) (interface{}, error)  // or a banking-local MatchableInvoice type
    GetInvoiceByNumber(number string) (interface{}, error)
}

type ExpenseLookup interface {
    GetExpenseByID(id string) (interface{}, error)
}
```

This way the matching algorithm moves into the package WITHOUT importing other domain packages directly (maintaining the dependency rule from TARGET_ARCHITECTURE.md).

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] ALL matching logic in `pkg/finance/banking/matcher.go`
- [ ] Cross-domain lookups via ports, NOT direct imports of other domains
- [ ] Tolerance values, fuzzy matching thresholds preserved EXACTLY
- [ ] Root matching functions deleted (replaced by service methods)

**Commit**: `refactor(codex): move banking matching and allocation into service`

---

### Ticket 4: Alias Invoice and CreditNote Models

**Models** (Batch 5 from DOMAIN_MODEL_ALIAS_PLAN.md — the business spine):
- `Invoice` → `finance.Invoice`
- `DBInvoiceItem` → `finance.DBInvoiceItem`
- `CreditNote` → `finance.CreditNote`
- `CreditNoteItem` → `finance.CreditNoteItem`
- `InvoiceSequence` → `finance.InvoiceSequence`

**WARNING**: `Invoice` is the MOST referenced type (3,567 hits in the archaeological audit). Expect many compile touch-points after aliasing. Most will be transparent (alias just works), but methods defined on the root Invoice type will need to move to `pkg/finance/domain.go`.

**Extra care**:
- Invoice has PDF generation connections — alias is safe (type name unchanged) but verify PDF service files compile
- InvoiceItem may have GORM hooks or calculation methods — move them with the type
- CreditNote references Invoice — both are now in the same package, so this is clean

**Approach**:
1. Alias Invoice FIRST, alone. Fix all compile errors. Build. Test.
2. Then alias DBInvoiceItem, CreditNote, CreditNoteItem together.
3. Then InvoiceSequence.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All Invoice family types owned by `pkg/finance`
- [ ] TableName() methods moved with types
- [ ] GORM hooks moved with types

**Commit**: `refactor(codex): alias Invoice and CreditNote models to pkg/finance`

---

### Ticket 5: Alias SupplierInvoice Models

**Models**:
- `SupplierInvoice` → `finance.SupplierInvoice`
- `SupplierInvoiceItem` → `finance.SupplierInvoiceItem`

**Depends on Ticket 4** because SupplierInvoice may reference Invoice types.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): alias SupplierInvoice models to pkg/finance`

---

### Ticket 6: Eliminate CGO — Swap mattn/go-sqlite3 for ncruces/go-sqlite3

**This is a HIGH-VALUE independent ticket.** Eliminating CGO means:
- No C compiler needed for cross-compilation
- Pure Go binary
- Simpler builds for Jordan

**Steps**:

1. In `go.mod`, replace:
   ```
   github.com/mattn/go-sqlite3 → github.com/ncruces/go-sqlite3
   ```

2. Find all import statements referencing mattn:
   ```bash
   grep -rn "mattn/go-sqlite3" --include="*.go"
   ```

3. For GORM usage, the driver import changes:
   ```go
   // BEFORE:
   import _ "github.com/mattn/go-sqlite3"
   // or
   import "gorm.io/driver/sqlite"

   // AFTER:
   import _ "github.com/ncruces/go-sqlite3/driver"
   import _ "github.com/ncruces/go-sqlite3/embed"
   ```

4. For `database/sql` direct usage:
   ```go
   // BEFORE:
   import _ "github.com/mattn/go-sqlite3"
   db, err := sql.Open("sqlite3", dsn)

   // AFTER:
   import _ "github.com/ncruces/go-sqlite3/driver"
   import _ "github.com/ncruces/go-sqlite3/embed"
   db, err := sql.Open("sqlite3", dsn)
   // Driver name stays "sqlite3" — ncruces registers the same name
   ```

5. GORM driver change:
   ```go
   // BEFORE:
   import gormsqlite "gorm.io/driver/sqlite"
   db, err := gorm.Open(gormsqlite.Open(dsn))

   // AFTER — ncruces works with GORM via the database/sql driver:
   import "gorm.io/gorm"
   import _ "github.com/ncruces/go-sqlite3/driver"
   import _ "github.com/ncruces/go-sqlite3/embed"

   sqlDB, _ := sql.Open("sqlite3", dsn)
   db, err := gorm.Open(gormsqlite.New(gormsqlite.Config{Conn: sqlDB}))
   // OR keep using gormsqlite.Open() — test which works
   ```

6. Run `go mod tidy` to remove mattn dependency.

7. Verify NO CGO:
   ```bash
   go build -tags='' ./...
   # Should succeed without CGO_ENABLED=1
   ```

**Known complications**:
- `config.go` imports mattn for command utilities — update those too
- `gen2brain/go-fitz` (PDF rendering) is ALSO CGO — DON'T remove it yet (Wave 8+ for OCR migration)
- `go-ole` is CGO on Windows — DON'T remove it (Windows-only, build-tagged)

**Goal**: Eliminate mattn/go-sqlite3 specifically. Other CGO deps remain for now.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `grep -rn "mattn/go-sqlite3" --include="*.go"` returns zero hits
- [ ] `grep "mattn" go.mod` returns zero (or only as indirect)
- [ ] SQLite operations work correctly (GORM queries, migrations)

**Commit**: `refactor(codex): replace mattn/go-sqlite3 with ncruces/go-sqlite3 (eliminate CGO)`

---

### Ticket 7: Alias Infra and System Models

**Models**:
- `CustomerMaster` → `crm.CustomerMaster`
- `SupplierMaster` → `crm.SupplierMaster`
- `CompanyBankAccount` → `finance.CompanyBankAccount` (if not done in Ticket 2)
- `Warehouse` → `crm.Warehouse`
- `InventoryItem` → `crm.InventoryItem`
- `StockMovement` → `crm.StockMovement`
- `StockAdjustment` → `crm.StockAdjustment`

**Extra care**: `CustomerMaster` has 180 references (from audit). It's a core entity but less complex than Invoice. The alias should be relatively smooth.

**Infra models to alias** (if time permits):
- `Setting` → `infra.Setting`
- `AuditLog` → `infra.AuditLog`
- `Job` → `infra.Job`

Skip `Role`, `User`, `Device`, `LicenseKey`, `SyncRecord` if they have deep entanglement with auth/startup code.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] CustomerMaster and SupplierMaster owned by `pkg/crm`
- [ ] Document any models that couldn't be aliased and why

**Commit**: `refactor(codex): alias remaining CRM and Infra models`

---

### Ticket 8: Wave 7 Progress Audit

**Deliverables**:
1. Count aliases in database.go (target: 60+)
2. Count remaining structs in database.go (target: <25)
3. Is `pkg/finance/banking/` fully complete? (all CRUD + reconciliation + matching)
4. Total `func (a *App)` across root files
5. LOC breakdown by `pkg/` package
6. CGO status: is mattn removed?
7. What remains for Wave 8 (Butler, Documents/OCR, Sync, Wails v3, Svelte 5)
8. Write `docs/WAVE7_PROGRESS.md`

**Commit**: `refactor(codex): write wave 7 progress report`

---

## 3. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0

**Ticket 3 special rules** (matching is the most complex algorithm):
- Preserve ALL tolerance values (bank matching thresholds)
- Preserve fuzzy matching logic byte-for-byte
- If matching references types from other domains, use ports (interfaces), NOT direct imports
- Test with `go test ./pkg/finance/banking/... -v` specifically

**Ticket 6 special rules** (SQLite swap):
- After the swap, run the FULL test suite (not just build) because SQL behavior differences could surface
- ncruces uses Wasm internally — ensure no `CGO_ENABLED=0` build issues
- If GORM driver wiring is tricky, check ncruces docs for GORM-specific integration patterns

---

## 4. Autonomy Contract

- Start with Ticket 1. Proceed in order.
- Do NOT stop between tickets unless a STOP condition hits.
- STOP conditions: build fails after 3 fix attempts; test regression; disk full.
- Tickets 1-3 (banking completion) are HIGHEST priority.
- Ticket 6 (ncruces) is HIGH VALUE but independent — if banking takes longer than expected, still do ncruces.
- If Invoice aliasing (Ticket 4) causes >100 compile errors that can't be resolved in 20 minutes, SKIP it and document the blocking pattern. Move to Ticket 6.

---

## 5. Expected Outcome

By end of this run:
- `pkg/finance/banking/` is FULLY COMPLETE — every banking operation is owned by the package
- Invoice and CreditNote types owned by `pkg/finance`
- SupplierInvoice types owned by `pkg/finance`
- mattn/go-sqlite3 ELIMINATED — pure Go builds
- database.go reduced to <25 remaining structs
- Clear path for Wave 8: Butler AI extraction, Documents/OCR extraction, framework upgrades

---

## Sign-Off

Wave 7 is the final major structural wave. After this:
- Banking is complete (the reference domain package)
- The business spine (Invoice, Order, Offer, Customer) is fully aliased
- CGO is eliminated from SQLite
- What remains is Butler (tangled), Documents (CGO-heavy), Sync (Turso migration), and framework upgrades (Wails v3 + Svelte 5)

Those are each their own epic. Today we finish the FOUNDATION.

🔥 Execute. Banking first. Invoice spine second. CGO last. The compiler guides you.
