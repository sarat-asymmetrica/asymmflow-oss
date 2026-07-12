# Codex Autonomous Execution Spec — Wave 10: Schema Bridge + TOON Format

**Date**: 2026-05-06
**From**: Claude (Opus 4.6, Senior Architect) + the maintainer
**To**: Codex (GPT-5.5, Senior Architect)
**Run Target**: Autonomous until complete
**Previous Runs**: Waves 0-9 complete. 69 commits. Extraction ERA + Schema ERA complete. 7 Cap'n Proto schemas, 138+ types, generated Go + TypeScript. Tests GREEN.
**Build Verification**: `go build ./...` and `go test ./... -count=1 -timeout 300s` after every ticket.
**Disk space**: Use `$env:GOTMPDIR='D:\go-tmp'` and `$env:GOCACHE='D:\go-cache'`.

---

## 0. Context — Why This Wave Is Hard (And Why We Do It Now)

Wave 9 created the Cap'n Proto schemas. Now we need to BRIDGE them to the existing GORM persistence layer and introduce TOON encoding at the LLM boundary.

**Architectural Decision**: Proto types become the PRIMARY interface. GORM is a persistence implementation detail. This means:

```
BEFORE (current):
  Domain Service → GORM struct → Wails JSON → Frontend

AFTER (this wave):
  Domain Service → GORM struct → Adapter → Proto message → Wails JSON → Frontend
                                    ↑
                              Wave 10 creates this
```

The adapter sits between GORM persistence and consumers (frontend, agents, API). Domain services keep working with GORM internally (we don't touch service.go files). The adapters MAP GORM structs to Proto messages at the boundary.

**TOON**: A compact encoding format that reduces LLM token usage by ~30%. We integrate it at the Butler ↔ LLM boundary so every API call from this point forward is cheaper.

---

## 1. Tickets

### Dependency Graph

```
Ticket 1 (adapter package setup) → Tickets 2-7 (per-domain adapters)
Ticket 8 (TOON research + integration) → independent
Ticket 9 (wire adapters into one pilot Wails endpoint) → after Ticket 2
Ticket 10 (progress audit) → last
```

---

### Ticket 1: Create Adapter Package Structure

**Deliverables**:
1. Create `pkg/adapter/` package
2. Create sub-packages mirroring the schema structure:
   ```
   pkg/adapter/
   ├── finance/    # GORM finance types ↔ Proto finance messages
   ├── crm/        # GORM CRM types ↔ Proto CRM messages
   ├── butler/     # GORM butler types ↔ Proto butler messages
   ├── documents/  # GORM documents types ↔ Proto documents messages
   ├── infra/      # GORM infra types ↔ Proto infra messages
   └── sync/       # GORM sync types ↔ Proto sync messages
   ```
3. Create `pkg/adapter/adapter.go` with shared conversion helpers:
   ```go
   package adapter

   import "time"

   // TimeToText converts time.Time to ISO 8601 text for Proto fields
   func TimeToText(t time.Time) string {
       if t.IsZero() { return "" }
       return t.Format(time.RFC3339)
   }

   // TextToTime converts ISO 8601 text from Proto fields to time.Time
   func TextToTime(s string) time.Time {
       t, _ := time.Parse(time.RFC3339, s)
       return t
   }

   // UintToText converts uint IDs to string for Proto Text fields
   func UintToText(id uint) string {
       return fmt.Sprintf("%d", id)
   }
   ```

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Package structure mirrors schemas/

**Commit**: `feat(codex): create adapter package structure for proto bridge`

---

### Ticket 2: Finance Adapter (GORM ↔ Proto)

**Source GORM types**: `pkg/finance/domain.go` (45 types)
**Target Proto types**: `schemas/go/finance/finance.capnp.go`

Create `pkg/adapter/finance/convert.go` with conversion functions:

```go
package finance

import (
    gorm "ph_holdings_app/pkg/finance"
    proto "ph_holdings_app/schemas/go/finance"
    "ph_holdings_app/pkg/adapter"
    capnp "capnproto.org/go/capnp/v3"
)

// InvoiceToProto converts a GORM Invoice to a Proto Invoice message
func InvoiceToProto(inv gorm.Invoice) (*proto.Invoice, error) {
    _, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
    if err != nil { return nil, err }

    p, err := proto.NewInvoice(seg)
    if err != nil { return nil, err }

    p.SetInvoiceNumber(inv.InvoiceNumber)
    p.SetCustomerId(adapter.UintToText(inv.CustomerID))
    p.SetGrandTotalBhd(inv.GrandTotalBHD)
    // ... map all fields

    return &p, nil
}

// InvoiceFromProto converts a Proto Invoice message to a GORM Invoice
func InvoiceFromProto(p proto.Invoice) (gorm.Invoice, error) {
    inv := gorm.Invoice{}
    inv.InvoiceNumber, _ = p.InvoiceNumber()
    // ... map all fields back
    return inv, nil
}
```

**IMPORTANT**: Start with the MOST USED types first:
1. Invoice, DBInvoiceItem (used everywhere)
2. Payment (financial core)
3. CompanyBankAccount, BankStatement, BankStatementLine (banking)
4. PurchaseOrder, PurchaseOrderItem (procurement)
5. Then remaining types

For complex types with embedded lists (Invoice.Items), use Cap'n Proto's List builders.

If a GORM field doesn't have a matching Proto field (or vice versa), document the gap with a comment but don't block on it.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] At least 15 core finance types have ToProto/FromProto converters
- [ ] Invoice roundtrip test: GORM → Proto → GORM preserves all fields

**Tests**: Create `pkg/adapter/finance/convert_test.go` with at least one roundtrip test:
```go
func TestInvoiceRoundtrip(t *testing.T) {
    original := gorm.Invoice{InvoiceNumber: "INV-2026-001", ...}
    proto, err := InvoiceToProto(original)
    require.NoError(t, err)
    back, err := InvoiceFromProto(*proto)
    require.NoError(t, err)
    assert.Equal(t, original.InvoiceNumber, back.InvoiceNumber)
}
```

**Commit**: `feat(codex): create finance GORM-Proto adapter with roundtrip tests`

---

### Ticket 3: CRM Adapter (GORM ↔ Proto)

**Source GORM types**: `pkg/crm/domain.go` (35 types)
**Target Proto types**: `schemas/go/crm/crm.capnp.go`

Create `pkg/adapter/crm/convert.go` with converters for:
1. CustomerMaster, CustomerContact
2. SupplierMaster, SupplierContact
3. Offer, OfferItem
4. Order, OrderItem
5. Opportunity
6. ProductMaster, SerialNumber
7. DeliveryNote, DeliveryNoteItem
8. GoodsReceivedNote, GRNItem
9. Remaining types

Same pattern as finance adapter. Include at least one roundtrip test.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] At least 15 core CRM types have converters

**Commit**: `feat(codex): create CRM GORM-Proto adapter with roundtrip tests`

---

### Ticket 4: Butler Adapter (GORM ↔ Proto)

**Source GORM types**: `pkg/butler/domain.go` (13 types)
**Target Proto types**: `schemas/go/butler/butler.capnp.go`

Create `pkg/adapter/butler/convert.go` for:
- ButlerResponse, ButlerAction, Intent, ButlerResolvedEntity
- PredictionRecord, Conversation, ChatMessage
- Remaining types

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Butler GORM-Proto adapter`

---

### Ticket 5: Documents Adapter (GORM ↔ Proto)

**Source types**: `pkg/documents/ports.go` types (CompanyInfo, BrandingConfig, etc.)
**Target Proto types**: `schemas/go/documents/documents.capnp.go`

Create `pkg/adapter/documents/convert.go`.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Documents GORM-Proto adapter`

---

### Ticket 6: Infra Adapter (GORM ↔ Proto)

**Source types**: `pkg/infra/` types (User, Role, Device, etc.)
**Target Proto types**: `schemas/go/infra/infra.capnp.go`

Create `pkg/adapter/infra/convert.go`.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Infra GORM-Proto adapter`

---

### Ticket 7: Sync Adapter (GORM ↔ Proto)

**Source types**: remaining root structs in `database.go` (5 types)
**Target Proto types**: `schemas/go/syncschema/sync.capnp.go`

Create `pkg/adapter/sync/convert.go`.

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

**Commit**: `feat(codex): create Sync GORM-Proto adapter`

---

### Ticket 8: TOON Integration at Butler LLM Boundary

**Goal**: Reduce token usage for Butler ↔ LLM communication by encoding context in TOON format instead of JSON.

**Step 1**: Research TOON availability.
- Check if `github.com/toon-format/toon-go` or similar Go library exists
- Run: `go search` or check GitHub for `toon-format` organization
- Check npm `@toon-format/toon` for reference implementation details

**Step 2**: If a Go library exists:
- Add it to `go.mod`
- Create `pkg/butler/chat/toon.go` with TOON encoder for Butler context
- Wire it into the LLM call path (before sending context to Sarvam)

**Step 3**: If NO Go library exists, create a minimal encoder:
- Create `pkg/toon/encoder.go` — a lightweight TOON encoder
- TOON format is simple: key-value pairs with minimal delimiters, no quotes on simple values, no commas
- Reference the npm package source for format spec
- Target: encode Butler context structs to TOON text

**Step 4**: If TOON format spec is unclear or unavailable:
- Create `pkg/toon/encoder.go` with a simplified compact encoding:
  ```go
  // Compact format: removes JSON overhead while remaining LLM-parseable
  // {"key": "value", "num": 42} → key=value num=42
  func CompactEncode(v interface{}) string
  ```
- This achieves similar token savings without strict TOON compliance
- Document that this should be replaced with official TOON when available

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] TOON or compact encoder exists in the codebase
- [ ] Butler LLM calls use compact encoding for context (not JSON)
- [ ] Benchmark: measure token reduction on a sample Butler prompt

**Commit**: `feat(codex): integrate TOON compact encoding at Butler LLM boundary`

---

### Ticket 9: Pilot Proto Endpoint (One Wails Method)

**Goal**: Prove the adapter bridge works end-to-end by converting ONE existing Wails method to return Proto-shaped data.

**Choose**: `GetDashboardStats()` — it's read-only, high-visibility, and used on the main screen.

**Steps**:
1. Create a NEW method: `GetDashboardStatsV2() (map[string]interface{}, error)`
2. Inside, call existing `GetDashboardStats()` to get GORM data
3. Convert through adapter to Proto message
4. Marshal Proto message to JSON-compatible map for Wails
5. Return the Proto-shaped data

**DO NOT** modify the existing `GetDashboardStats()` — the V2 method is additive.

This proves:
- Adapter conversion works
- Proto-shaped data flows through Wails to frontend
- No breaking changes to existing functionality

**Checklist**:
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `GetDashboardStatsV2()` returns Proto-shaped data
- [ ] Existing `GetDashboardStats()` unchanged and working

**Commit**: `feat(codex): pilot Proto endpoint for dashboard stats`

---

### Ticket 10: Wave 10 Progress Audit

**Deliverables**:
1. Count of adapter conversion functions (ToProto + FromProto pairs)
2. Count of roundtrip tests
3. TOON/compact encoding status (library found? custom built? token savings measured?)
4. Pilot endpoint status
5. Any GORM types that couldn't be mapped to Proto (document why)
6. Write `docs/WAVE10_PROGRESS.md`

**Commit**: `docs(codex): write wave 10 progress report`

---

## 2. Adapter Writing Rules

### Naming Convention
- `XxxToProto(gorm GormType) (*proto.ProtoType, error)` — GORM → Proto
- `XxxFromProto(p proto.ProtoType) (GormType, error)` — Proto → GORM
- One file per domain: `convert.go`
- One test file per domain: `convert_test.go`

### Field Mapping Rules

| GORM Field Type | Proto Field Type | Conversion |
|----------------|-----------------|------------|
| `uint` (ID) | `Text` | `fmt.Sprintf("%d", id)` / `strconv.ParseUint` |
| `string` | `Text` | Direct (Set/Get) |
| `float64` | `Float64` | Direct |
| `int64` | `Int64` | Direct |
| `bool` | `Bool` | Direct |
| `time.Time` | `Text` | `RFC3339` format |
| `*time.Time` | `Text` | Empty string if nil |
| `gorm.DeletedAt` | `Text` | Empty string if not deleted |
| `[]ItemType` | `List(ProtoItemType)` | Loop with NewList + Set |
| `map[string]interface{}` | Skip | Document as unmappable |
| `json.RawMessage` | `Data` | Direct bytes |

### What to Skip
- GORM tags — not relevant to Proto
- GORM hooks (BeforeCreate, AfterUpdate) — persistence-only
- Methods on GORM structs — only data fields
- Fields with `map[string]interface{}` — too dynamic, skip with comment
- Fields that exist in GORM but not in Proto (or vice versa) — skip with comment

### Error Handling
- Return `error` from all conversion functions
- If a field can't be set (Cap'n Proto error), wrap and return
- Don't panic on missing fields — log and skip

---

## 3. Quality Gates

After EVERY ticket:
1. `go build ./...` exits 0
2. `go test ./... -count=1 -timeout 300s` exits 0

### Special Rules
- If Cap'n Proto generated types have unexpected API shapes, READ the generated `.capnp.go` file to understand the getter/setter pattern before writing converters
- Cap'n Proto getters return `(value, error)` for Text fields — always handle the error
- Cap'n Proto requires `capnp.NewMessage()` and segment allocation — follow the pattern from Ticket 2
- If a generated Proto type is missing (schema compilation issue), skip that converter with a TODO

---

## 4. Autonomy Contract

- Start with Ticket 1. Proceed in order.
- Tickets 2-7 (adapters) are the CORE deliverable — complete ALL of them.
- Ticket 8 (TOON) is important but independent — do it in parallel or after adapters.
- Ticket 9 (pilot endpoint) is a STRETCH — do it if time allows.
- Do NOT stop between tickets.
- STOP conditions: build fails after 3 fix attempts; test regression; disk full.
- If adapter conversion for a specific type is too complex (>10 min on one type), skip it with TODO and move to next type.

---

## 5. What NOT To Touch

- `pkg/*/domain.go` — NEVER modify GORM structs
- `pkg/*/service.go` — NEVER modify domain service logic
- `schemas/*.capnp` — NEVER modify schema definitions
- `schemas/go/*` — NEVER modify generated code
- Existing Wails methods — NEVER modify (Ticket 9 adds a NEW method)
- Frontend files — no Svelte changes

---

## 6. Expected Outcome

- `pkg/adapter/` with 6 sub-packages (finance, crm, butler, documents, infra, sync)
- ~60-80 ToProto/FromProto converter pairs covering the most-used types
- Roundtrip tests for at least 5 core types (Invoice, CustomerMaster, Order, Offer, ChatMessage)
- TOON or compact encoder at Butler LLM boundary
- One pilot V2 endpoint proving end-to-end Proto flow
- Build and tests GREEN
- Foundation ready for Wave 11 (MVVM ViewModels consume Proto types)

---

## Sign-Off

This wave is the HARDEST of the construction era. It's mechanical but demands precision — every field mapping must be correct, every Cap'n Proto API call must allocate segments properly, every roundtrip test must prove fidelity.

The payoff is ENORMOUS: after this wave, Proto types are the lingua franca. ViewModels (Wave 11) build on Proto. Agents produce Proto. Modules extend Proto. The marketplace normalizes against Proto. Everything flows through the same type system.

The bridge is the FOUNDATION. Build it right. Test it thoroughly. No shortcuts.

🏗️ Bridge the types. Encode the context. Prove the endpoint. GO.
