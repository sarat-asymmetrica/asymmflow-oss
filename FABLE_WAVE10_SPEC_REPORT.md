# FABLE Wave 10 — Sensory & Brand — Status Report

**Branch:** `feat/fable-wave10-sensory-brand` (off `main`) — NOT merged, NOT pushed, NOT tagged.
**Orchestrator:** Opus 4.8. **Coders:** Sonnet 5 subagents. **Gate:** the orchestrator, per §1 of the spec.
**Mission:** make the app *feel* owned — timeline, motion, language, one sound. Flows frozen; feel layered on top.

> Status legend: ✅ shipped & gated · ⚠️ shipped with a flag for the owner · ⏳ in progress.

---

## Phase A — Recon verdicts (read-only)

Full evidence in `docs/wave10-recon/A1..A7`. Headlines:

**A1 — Interaction inventory (→B1).** Live canonical button = `frontend/src/lib/components/ui/Button.svelte` (290 uses/37 files). It has **no press transform today** and `.btn-ghost`/`.btn-secondary` have no `:active` at all. `packages/ui/src/form/Button.svelte` is the orphaned reference impl carrying the proven `transform: scale(0.985)` press — port that pattern, don't adopt the package. Focus rings are fragmented across 4 rules; no single token. 101 files use raw `<button>`; **WorkHub's 5 panels (~19 unstyled raw buttons) are the highest-value, lowest-risk convergence target**; CustomerContactsStrip:39, CustomerDetailHeader:29/:41 are the other clear ones.

**A2 — Motion census + token verdict (→B2).** Canonical LIVE token file = `frontend/src/assets/design-tokens.css` (imported first from `main.ts`; already owns `--transition-*`). The other candidate token files (`styles/design-tokens.css`, `phi-design-tokens.css`, `wabi-sabi.css`) are **dead/unimported**; `packages/tokens` is a not-yet-wired future system. **`prefers-reduced-motion` is effectively absent app-wide** (only `DataTable.svelte` has a scoped block; the 3 global resets are orphaned) — a real global reset is required. Three inconsistent modal impls (0/150/200ms); toasts use `fade` 180ms. ~20 animations >250ms + 3 spring/bounce curves flagged.

**A3 — Deal-spine data reality (→B3/B4/B5a).** FK chain: `Order.OfferID/RFQID` → Offer/RFQ; `DeliveryNote.OrderID` → Order; `Invoice.OrderID/OfferID/QuoteID/RfqID/DeliveryNoteID`; `Payment.InvoiceID`; `CustomerReceiptAllocation.ReceiptID/InvoiceID/PaymentID`. **No existing one-call deal assembler**; closest is `GetInvoiceAuditTrail` (invoice_traceability.go:29) — clone its pattern into `GetDealTimeline(orderID)`. **PAID is derived** by `customerInvoiceSettlementStatus` (customer_invoice_payment_policy.go:48) applied in `RecordPartialPayment` (customer_invoice_service.go:1974); hook B4 to that post-write point, not a status-field watch. Bindings regen = `wails generate module`. Mount points: `OrderDetail.svelte` (has a `.timeline` section already), `Customer360.svelte` + `CustomerOrdersTab.svelte`.

**A4 — Audio in Wails (→B4).** Vite asset import → `//go:embed frontend/dist` → Wails asset server = **no network** (same path fonts already use). WebView2/Chromium gesture rule: `.play()` must be the first synchronous statement in the click handler. **Zero existing audio** confirmed. Opt-out lives in the generic settings map (`app_setup_documents_surface.go` `GetSettings`, default-true gives opt-out free) + a small store for PaymentsScreen to read it.

**A5 — Perceived-latency hot spots (→B1).** Top 5: DashboardScreen, WorkHub, CustomerDetailView (+SupplierDetailView), FinancialDashboard, CRMCustomerDashboard (+CRMSupplier) — all centered-`WabiSpinner`/blank-then-pop. Recommendation: build reusable `TableSkeleton` + `CardGridSkeleton` primitives (offenders share table/list/stat-card shapes). WabiSpinner is the only loading primitive today (56 files).

**A6 — Toast census (→B6).** 826 sites/63 files; ~814 legitimately CONFIRM. Only **4 ANNOUNCE violations** (App.svelte:91 file-watch, App.svelte:627 session-expired, OpportunityDetail:342 cross-device conflict, WorkHub:972 cached-while-sync), **1 duplicate** (BankReconciliationScreen:412 toast dupes an inline banner), and a dev-only `ToastTestButton.svelte` (7 sites) to delete.

**A7 — Empty states + rituals + brand (→B5).** ~12 generic list-view empty states (highest-leverage); `WabiEmptyState.svelte` exists but is unused (the home for new copy). **No document-checklist UI today.** Brand slots: sidebar (`EnterpriseSidebar.svelte:164`) and login (`LoginScreen.svelte:96`) **hardcode a stale "PH Trading"/"PH Holdings" wordmark** (also a synthetic-identity leak) — the real gap; PDF headers (`company_branding.go` + `pkg/overlay`) and desktop shell (`wails.json`, `build/appicon.png`) are already config/token-driven.

---

## Phase B — Implementation

_(filled in as each batch gates)_

### B1 — Responsiveness ⏳
### B2 — One motion vocabulary ⏳
### B3 — Deal-spine timeline ⏳
### B4 — The one sound ⏳
### B5 — Rituals, operator language, brand slot ⏳
### B6 — Toast discipline ⏳

---

## Gate results
_(final)_

## Taste Ledger
_(final — every aesthetic decision, alternatives considered, what to review first)_
