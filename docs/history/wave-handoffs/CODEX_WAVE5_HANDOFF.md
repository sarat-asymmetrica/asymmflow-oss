# Codex Autonomous Execution Spec — Wave 5: Domain Model Alias Bridge

**Date**: 2026-05-05
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Wave 0-4 complete. app.go: 21,763 → 2,135 LOC. God Object decomposed.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.

---

## 0. Context — What You're Doing And Why

The God Object is dead. Domain service facades exist in `pkg/finance/`, `pkg/crm/`, `pkg/infra/`. But the services are currently **adapter shells** — they forward to root helper functions that still use root-owned GORM models from `database.go`.

This wave converts them to **real domain packages** by:
1. Making package-owned types the source of truth (type aliases in root)
2. Moving SQL logic bodies from root helpers INTO the packages
3. Keeping Wails bindings stable throughout

Read `docs/DOMAIN_MODEL_ALIAS_PLAN.md` for full strategy. This spec implements Batch 1 + Batch 2 + the banking logic move.

---

## 1. Governance

You are governed by:
- `.codex/AGENTS.md` — your identity and rules
- `docs/OPERATING_PRINCIPLES.md` — unchained mandate
- `docs/DOMAIN_MODEL_ALIAS_PLAN.md` — the alias strategy and readiness gates

**Key constraint**: `database.go` is used by GORM's AutoMigrate. Type aliases preserve this — GORM sees the same struct shape, same table names, same tags. The alias is transparent to GORM.

---

## 2. The Mechanical Protocol (Follow Exactly)

For each model family:

```
Step 1: COMPARE
  - Read the struct in database.go
  - Read the matching struct in pkg/*/domain.go
  - Identify field differences (missing fields, different tags, different types)

Step 2: ALIGN
  - Update pkg/*/domain.go to match database.go EXACTLY
  - Same field names, same types, same GORM tags, same JSON tags
  - Include TableName() method if one exists on the root struct
  - Include any BeforeCreate/AfterCreate GORM hooks

Step 3: ALIAS
  - In database.go, replace the struct definition with a type alias:
    type Payment = finance.Payment
  - Add the import for the package
  - Remove the old struct body (but keep any standalone helper functions)

Step 4: BUILD
  - go build ./...
  - If it fails: fix import cycles, missing methods, or tag mismatches
  - Common issue: methods defined on the root type can't exist on an alias
    Solution: move the method to the package alongside the type

Step 5: TEST
  - go test ./... -count=1 -timeout 300s
  - If tests reference the old type directly, they'll still work (alias is transparent)

Step 6: COMMIT
  - One commit per model family (2-4 related types together is fine)
  - Prefix: refactor(codex): alias <ModelName> to pkg/<domain>
```

---

## 3. Tickets

### Dependency Graph

```
Ticket 1 (Payment models) ───┐
Ticket 2 (Expense models) ───┼── Ticket 5 (Banking logic move)
Ticket 3 (FX models) ────────┤
Ticket 4 (Banking models) ───┘

Ticket 6 (Fulfillment models) ── independent
Ticket 7 (Procurement models) ── independent
Ticket 8 (Progress audit) ────── last
```

Tickets 1-4 must complete before Ticket 5. Tickets 6-7 are independent of 1-5.

---

### Ticket 1: Alias Payment Models

**Models**:
- `Payment` → `finance.Payment`
- `SupplierPayment` → `finance.SupplierPayment`

**Steps**:
1. Align `pkg/finance/domain.go` Payment/SupplierPayment with `database.go` versions
2. Move any methods defined on root Payment/SupplierPayment to `pkg/finance/domain.go`
3. Replace struct definitions in `database.go` with aliases
4. Verify `pkg/finance/payment/service.go` can now use its own package's types directly

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `grep "type Payment " database.go` shows alias, not struct
- [ ] No circular imports

**Commit**: `refactor(codex): alias Payment models to pkg/finance`

---

### Ticket 2: Alias Expense Models

**Models**:
- `ExpenseCategory` → `finance.ExpenseCategory`
- `ExpenseVendor` → `finance.ExpenseVendor`
- `ExpenseEntry` → `finance.ExpenseEntry`
- `RecurringExpense` → `finance.RecurringExpense`
- `BankExpenseEntry` → `finance.BankExpenseEntry`

**Steps**: Same protocol as Ticket 1.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `pkg/finance/expense/service.go` uses package-local types

**Commit**: `refactor(codex): alias Expense models to pkg/finance`

---

### Ticket 3: Alias FX/Currency Models

**Models**:
- `CurrencyExchangeRate` → `finance.CurrencyExchangeRate`
- `FXRate` → `finance.FXRate`
- `FXRevaluation` → `finance.FXRevaluation`

**Steps**: Same protocol.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `refactor(codex): alias FX models to pkg/finance`

---

### Ticket 4: Alias Banking Models

**Models** (larger batch — the full banking family):
- `BankStatement` → `finance.BankStatement`
- `BankStatementLine` → `finance.BankStatementLine`
- `BankLinePaymentAllocation` → `finance.BankLinePaymentAllocation`
- `BankCashBalance` → `finance.BankCashBalance`
- `StatementHash` → `finance.StatementHash`
- `BookBankReconciliation` → `finance.BookBankReconciliation`
- `OutstandingCheque` → `finance.OutstandingCheque`
- `DepositInTransit` → `finance.DepositInTransit`
- `ChequeRegister` → `finance.ChequeRegister`
- `BankStatementFile` → `finance.BankStatementFile`
- `BankReconciliationAuditLog` → `finance.BankReconciliationAuditLog`

**Extra care**: These models are referenced by the banking service AND by the reconciliation matcher. Ensure all GORM hooks and TableName() methods move with the types.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] All banking models now owned by `pkg/finance`
- [ ] `pkg/finance/banking/service.go` references `finance.BankStatement` (its own package)

**Commit**: `refactor(codex): alias Banking models to pkg/finance`

---

### Ticket 5: Move Banking SQL Logic Into Package

**This is the BIG payoff ticket.** Now that banking models are owned by `pkg/finance`, the SQL logic can move FROM root helper functions INTO `pkg/finance/banking/`.

**What to move**:
- Find all root-level helper functions that serve the banking facade methods listed in WAVE4_PROGRESS.md
- Move their bodies into `pkg/finance/banking/service.go` (or `matcher.go`, `reconciliation.go` as appropriate)
- The root `app_*.go` methods become TRUE thin wrappers: just delegate to `a.bankingService.Method()`

**Pattern**:
```go
// BEFORE (root helper):
func (a *App) getBankStatementsHelper(bankAccountID string) ([]BankStatement, error) {
    var statements []BankStatement
    a.db.Where("bank_account_id = ?", bankAccountID).Find(&statements)
    return statements, nil
}

// AFTER (in pkg/finance/banking/service.go):
func (s *Service) GetBankStatements(bankAccountID string) ([]finance.BankStatement, error) {
    var statements []finance.BankStatement
    s.db.Where("bank_account_id = ?", bankAccountID).Find(&statements)
    return statements, nil
}

// Root wrapper becomes:
func (a *App) GetBankStatements(bankAccountID string) ([]BankStatement, error) {
    return a.bankingService.GetBankStatements(bankAccountID)
}
```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `pkg/finance/banking/` contains real SQL logic, not just forwarding
- [ ] Root banking functions are now 1-3 line delegation wrappers
- [ ] No root helper function references banking GORM models directly (package owns them now)

**Commit**: `refactor(codex): move banking SQL logic into pkg/finance/banking`

---

### Ticket 6: Alias Fulfillment Models

**Models**:
- `DeliveryNote` → `crm.DeliveryNote`
- `DeliveryNoteItem` → `crm.DeliveryNoteItem`
- `SerialNumber` → `crm.SerialNumber`

**Steps**: Same protocol as Ticket 1.

**Extra care**: SerialNumber has a state machine (registered → allocated → shipped → delivered → invoiced). Ensure all state transition methods move with the type.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Serial lifecycle methods are in `pkg/crm/domain.go` or `pkg/crm/fulfillment/`

**Commit**: `refactor(codex): alias Fulfillment models to pkg/crm`

---

### Ticket 7: Alias Procurement Models

**Models**:
- `PurchaseOrder` → `crm.PurchaseOrder`
- `PurchaseOrderItem` → `crm.PurchaseOrderItem`
- `GoodsReceivedNote` → `crm.GoodsReceivedNote`
- `GRNItem` → `crm.GRNItem`

**Steps**: Same protocol.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `pkg/crm/procurement/service.go` uses package-local types

**Commit**: `refactor(codex): alias Procurement models to pkg/crm`

---

### Ticket 8: Wave 5 Progress Audit

**Deliverables**:
1. Count how many models in `database.go` are now aliases vs original structs
2. Count lines remaining in `database.go`
3. Verify `pkg/finance/banking/` is a REAL domain package (owns types AND logic)
4. List what remains for Wave 6 (the high-coupling models: Invoice, Offer, Order, Customer)
5. Write `docs/WAVE5_PROGRESS.md`

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Progress report written with alias count, remaining struct count, recommendation for Wave 6

**Commit**: `refactor(codex): write wave 5 progress report`

---

## 4. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0
3. No import cycles (Go compiler enforces this, but double-check if build fails)
4. GORM AutoMigrate still works (aliases are transparent to GORM)

**If alias causes import cycle**:
- Most common cause: a method on the root type references another root type
- Solution: move BOTH types to the package, or break the method into a standalone function
- If stuck after 3 attempts: skip that model, document in progress report, move on

**If GORM fails on aliased type**:
- Check that TableName() method was moved with the type
- Check that GORM hooks (BeforeCreate, etc.) were moved
- Check that embedded `Base` or `gorm.Model` is identical

---

## 5. Autonomy Contract

- Start with Ticket 1. Proceed in order through Ticket 8.
- Do NOT stop between tickets unless a STOP condition hits.
- STOP conditions: build fails after 3 fix attempts on an alias; import cycle that requires architectural decision; test failure that isn't explained by the alias change.
- Commit after each ticket with the specified message.
- If a model has dependencies on ANOTHER un-aliased model (e.g., Payment references Invoice), and Invoice isn't aliased yet: leave Payment's InvoiceID as `string` (it's already a string FK), don't try to alias Invoice early.

---

## 6. What NOT To Touch

- `Invoice`, `Order`, `Offer`, `CustomerMaster`, `SupplierMaster` — Batch 5, too high-coupling for this wave
- `butler_ai.go` — completely out of scope
- Frontend files — no Svelte changes
- `config.go` — infra domain, later wave
- Wails binding generation — not needed (aliases preserve type names)

---

## 7. Expected Outcome

By end of this run:
- ~25-30 models aliased from `database.go` to domain packages
- `database.go` reduced from ~2,466 LOC to ~1,500 LOC (spine models remain)
- `pkg/finance/banking/` is a FULLY REAL domain package (owns types + logic)
- Pattern proven: alias → move logic → thin wrapper. Ready to replicate for all domains.
- Clear path documented for Wave 6 (the high-coupling Invoice/Order/Customer extraction)

---

## Sign-Off

The alias bridge is the last structural prerequisite. After this wave, every future extraction follows the same proven pattern. Make it boring. Make it repeatable. Make it GREEN.

🔥 Execute. One model family at a time. The compiler is your safety net.
