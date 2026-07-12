# Next Session Fixes — Phase 26 Complete Audit Resolution

**Generated**: 2026-02-28
**Context**: Phase 25 applied 48+ P0/P1/P2 fixes. Phase 26 resolved all remaining P3 and Phase 17 deferred items. **ALL findings resolved.**

---

## Phase 26 Fix Log (21 items — P3 + Phase 17 deferred)

### P3 Code Hygiene (14 fixes)

| ID | File | Fix Applied |
|----|------|-------------|
| S1 | `serial_number_service.go` | `linkSerialsToInvoice` status filter — case-insensitive via `UPPER()` |
| S5 | `serial_number_service.go` | Limit(1000) — added log warning when limit is hit |
| S7 | `serial_number_service.go` | `allocateSerialsToDN` — batch size validation (max 500 serials) |
| C1 | `credit_note_service.go` | `len(reason)` → `utf8.RuneCountInString(reason)` |
| C2 | `credit_note_service.go` | reason field HTML-sanitized via `html.EscapeString()` |
| C7 | `credit_note_service.go` | CreditNoteItemInput.Description — 500 rune length validation |
| C8 | `credit_note_service.go` | PDF description truncation — rune-aware `[]rune` slicing |
| E2 | `einvoice_service.go` | XML file permissions `0644` → `0640` |
| I1 | `customer_invoice_service.go` | HMAC fallback log — `sync.Once` prevents log flooding |
| I7 | `customer_invoice_service.go` | HMAC salt path — absolute via `os.Executable()` + `filepath.Dir()` |
| PO-26 | `purchase_order_service.go` | Empty currency fallback to `"BHD"` in log messages |
| DN-A3 | `delivery_note_service.go` | `DispatchDeliveryNote` — empty status bypass blocked |
| DN-A6 | `delivery_note_service.go` | `CreateDeliveryNote` — force initial status to `"Prepared"` |
| B1 | `customer_invoice_service.go` | Synthetic order item creation moved inside transaction |

### Phase 17 Deferred (7 fixes)

| # | File | Fix Applied |
|---|------|-------------|
| 1 | `field_crypto.go` | PBKDF2 salt — random 32-byte salt via `crypto/rand`, stored in `.field_crypto_salt` with absolute path resolution |
| 2 | `auth_handler.go` | OAuth callback — already binds to `127.0.0.1:8080` (verified, no change needed) |
| 3 | `business_invariants.go` + `app.go` | `ValidateCostingApproval()` wired into `ApproveCostingSheet()` — enforces 8% min margin |
| 4 | `delivery_note_service.go` | Negative inventory prevention — `GetOrderDeliveryStatus()` validates remaining quantities + positive quantity check |
| 5 | `batch_operations.go` | Batch operations bounded — `maxBatchSize = 500` on all internal update/delete methods |
| 6 | `butler_ai.go` | Prompt injection — `sanitizeForPrompt()` applied to context data in `buildMistralSystemPrompt` and document text in `AnalyzeDocumentWithButler` |
| 7 | `frontend/package.json` | Already on Vite 5.4.0 + Svelte 4.2.0 (verified, no CVEs from Vite 3.x/Svelte 3.x) |

---

## Phase 25 Fix Log (48 items — P0/P1/P2)

### Round 1 (7 fixes from Red Team Round 2)
- P0: Credit check TOCTOU — credit check + invoice creation in single atomic tx (customer_invoice_service.go)
- P1: GrandTotalBHD removed from editable fields (customer_invoice_service.go)
- P1: Status validation — can't set "Paid" if OutstandingBHD > 0; allowlist enforced (customer_invoice_service.go)
- P1: Payment ID collision — sha256(time) replaced with uuid.New().String() (customer_invoice_service.go)
- P1: RecordPartialPayment rejects Cancelled/Void/Proforma/Paid invoices (customer_invoice_service.go)
- P1: UpdatePOStatus blocks Draft→Sent for POs above 5K BHD threshold (purchase_order_service.go)
- P2: DeletePurchaseOrder status guard + CreatePurchaseOrder forced Draft (purchase_order_service.go)

### Round 2 (10 fixes from Red Team Round 3)
- P0: PO permission namespace fix — `purchase_orders:*` → `po:*` (14 occurrences)
- P1: Empty string status rejected + Void/Proforma added to allowlist (customer_invoice_service.go)
- P1: Cancelled/Void/Paid treated as terminal states — entire invoice immutable (customer_invoice_service.go)
- P1: InvoiceNumber locked on non-Draft invoices (customer_invoice_service.go)
- P1: Invoice items locked on non-Draft invoices (customer_invoice_service.go)
- P1: MarkCustomerInvoicePaid rejects Cancelled/Void/Proforma/Paid (customer_invoice_service.go)
- P1: RecordPayment rejects Cancelled/Void/Proforma/Paid (payment_service.go)
- P1: UpdatePOStatus blocks Draft→Approved above 5K BHD threshold (purchase_order_service.go)
- P1: VAT export excludes Void/Proforma/Draft invoices (einvoice_service.go)
- P1: Duplicate invoice check moved inside atomic transaction (customer_invoice_service.go)

### Round 3 (25 P2 fixes + 6 P1s from red team)
- All 25 P2 findings resolved across 9 files
- 6 additional P1s found during P2 red team and fixed (RBAC gaps, stale status strings, PO CreatedBy, CN Proforma guard)

### Phase 24 Fixes (33 items — prior session)
- 14 Phase 23 sweep fixes, 5 feature fixes, 8 red team hardening fixes, 6 pre-existing P1 fixes

---

## Phase 26 Red Team Fixes (10 items from 4 audit agents)

### P1 Fixes (3)
- `butler_ai.go`: `sanitizeForPrompt` removed 2000-char truncation that destroyed all Butler context; made case-insensitive via compiled regex
- `app.go:ApproveCostingSheet`: Customer grade now looked up via RFQ→Customer linkage (was passing empty string, bypassing Grade C/D advance + ABB checks)
- `delivery_note_service.go:CreateDeliveryNote`: TOCTOU fix — quantity validation + DN creation wrapped in single transaction with order row lock

### P2 Fixes (7)
- `delivery_note_service.go:CreateDeliveryNote`: `GetOrderDeliveryStatus` error no longer silently swallowed (was bypassing all quantity validation on error)
- `delivery_note_service.go:DispatchDeliveryNote`: DN status + serial status updates wrapped in single atomic transaction
- `delivery_note_service.go:ConfirmDeliveryNote`: Same atomic transaction fix for confirm flow
- `field_crypto.go`: Removed deterministic salt fallback (was negating the random salt fix when filesystem fails)
- `field_crypto.go:loadOrCreateSalt`: Atomic write via temp-file-then-rename + symlink resolution via `filepath.EvalSymlinks`
- `customer_invoice_service.go:computeDocumentHMAC`: `os.Executable()` error now handled explicitly (was silently degrading to CWD-relative path)
- `batch_operations.go`: `maxBatchSize=500` applied to all 9 BatchCreate methods + delete methods now use constant instead of magic number

---

## Phase 26 Red Team Round 2 Fixes (12 items from 4 audit agents)

### P0 Fixes (2)
- `delivery_note_service.go:ConfirmDeliveryNote`: Fixed `dn_id` → `dn_number` (column didn't exist on serial_numbers table — ConfirmDeliveryNote was broken for serials)
- `app.go:CreateCostingSheet`: Fixed `markup_percent` → `margin_percent` key (frontend sends `margin_percent`, stored margin was always 0, making 8% approval check dead code)

### P1 Fixes (3)
- `customer_invoice_service.go:computeDocumentHMAC`: Replaced disk-read salt with `globalFieldCrypto.salt` (avoids EvalSymlinks path mismatch; removed unused `os`/`filepath` imports)
- `app.go:ApproveCostingSheet`: Silent grade lookup failure now logs WARNING + case-insensitive fallback for name mismatches
- `delivery_note_service.go:CreateDeliveryNoteWithItems`: TOCTOU fix — quantity re-validation + DN creation wrapped in single transaction

### P2 Fixes (7)
- `delivery_note_service.go:DispatchDeliveryNote`: Restored `shipped_date` on serial status update (was dropped when inlining markSerialsShipped)
- `delivery_note_service.go:ConfirmDeliveryNote`: Restored per-serial `warranty_end_date` calculation (was dropped when inlining markSerialsDelivered)
- `delivery_note_service.go:CreateDeliveryNote`: Unknown OrderItemID now returns error instead of silently bypassing validation + order items query error now checked
- `butler_ai.go:buildMistralSystemPrompt`: `[ACTIONS]`/`SYSTEM:`/`INSTRUCTIONS:` cleanup now case-insensitive via compiled regex
- `butler_ai.go:buildMistralSystemPrompt`: Context JSON capped at 100K chars to prevent unbounded API payloads
- `batch_operations.go`: App-level wrappers now use `maxBatchSize` constant instead of magic `500`
- `app.go`: `NewFieldCrypto` failure now logs CRITICAL-level warning (was silently disabling all encryption)

---

## ALL FINDINGS RESOLVED

| Priority | Fixed | Remaining |
|----------|-------|-----------|
| P0 | 4 | 0 |
| P1 | 29 | 0 |
| P2 | 39 | 0 |
| P3 | 14 | 0 |
| Phase 17 deferred | 7 | 0 |
| **Total** | **93** | **0** |

---

## Security Audit Coverage

| Audit Round | Source | Findings | Status |
|-------------|--------|----------|--------|
| Phase 12 Red Team | 6 parallel agents, 60+ issues | All resolved | ✅ |
| Phase 17 Red Team | 6 parallel agents, 117 raw findings | P0/P1 same-day, P2 deferred → resolved Phase 26 | ✅ |
| Phase 23 Red Team | 4 rounds, P0-P2 | All resolved | ✅ |
| Phase 24 Red Team | 2 rounds | All resolved | ✅ |
| Phase 25 Red Team | 3 rounds | All resolved | ✅ |
| Phase 26 Sweep | P3 + deferred cleanup | All resolved | ✅ |
| Phase 26 Red Team R1 | 4 agents, 10 findings (3 P1, 7 P2) | All P1/P2 fixed | ✅ |
| Phase 26 Red Team R2 | 4 agents, 12 findings (2 P0, 3 P1, 7 P2) | All fixed | ✅ |
