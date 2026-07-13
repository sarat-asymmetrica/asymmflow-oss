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

### B1 — Responsiveness ⏳ (Batch 2)

### B2 — One motion vocabulary ✅ (committed 64b101c)
One motion vocabulary now lives in `frontend/src/assets/design-tokens.css` — the canonical LIVE token file (imported first from `main.ts`):
```
--motion-fast: 120ms;   --motion-base: 200ms;   --motion-settle: 260ms;
--ease-standard: cubic-bezier(0.25,0.1,0.25,1);   --ease-decelerate: cubic-bezier(0,0,0.2,1);
--focus-ring-color: var(--brand-indigo);
```
The legacy `--transition-fast/base/slow` and `--easing-*` were **re-pointed to alias these** (one source of truth); `--transition-slow` came 400ms→250ms; **the spring `cubic-bezier(0.34,1.56,0.64,1)` is retired** from the vocabulary and removed from the real-surface call sites (`.animate-flourish`, `IntelligenceHub`, `CursorFollower`). A **live global `@media (prefers-reduced-motion: reduce)` reset** was added (previously effectively absent app-wide). Modal + toast enter/exit were unified onto the tokens: the global ConfirmHost modal (previously zero animation) now has a subtle 200ms decelerate entrance; the three modal impls and the toast were standardized to fade + small translate/scale, no overshoot. A latent invalid-CSS bug (two timing-functions stacked in one `animation` shorthand) was fixed in passing.
**Audit — motion tokens = one source:** ✅ a single grep at the token layer finds every duration/easing.
**Reduced-motion:** CSS reset kills all CSS motion. Svelte JS transitions (fly/fade/scale) are NOT CSS — the orchestrator added `frontend/src/lib/motion.ts` (`motionMs()`) and gated the shared primitives (toast + QuickCaptureModal). A final mechanical sweep of the remaining ~98 app-wide JS-transition sites is scheduled as the last step (see Gate results) so the "fully static" claim is honest, not assumed.

### B3 — Deal-spine timeline ✅ (committed 64b101c) — THE signature
Backend `GetDealTimeline(orderID)` + `GetDealTimelineByOrderNumber` in `invoice_traceability.go`: a **read-only 6-query assembler** (Order→Offer→RFQ→Costing→DeliveryNotes→Invoices) returning ordered `DealTimelineNode{Stage,Serial,Date,State,RecordID,RecordType}`. PAID is **derived in-memory** via the app's own `hydrateCustomerInvoicesPaymentState` (customer_invoice_payment_policy.go) — **zero writes**, `finance:view`-gated. Missing links render honestly (`pending` for a real future stage, `na` for an optional one) — never invented. `DealTimeline.svelte` is the owner-ratified compact single-row stepper (state-colored dots on a thin rule; serial + date beneath; horizontal scroll on narrow widths, never shrink-to-fit) on Onyx & Ether monochrome tokens. Mounted on `OrderDetail.svelte` (primary, preserving the existing stage-change log) and `CustomerOrdersTab.svelte` (per-row on-demand toggle). Customer360 left unmounted (it's a prediction/graph screen with no deal rows — Article I.5). Bindings regenerated.
**AC:** one glance answers "where does this deal stand"; every present node deep-links via the app's existing handoff stores; assembly is one round-trip; partial chains render.

### B4 — The one sound ✅ (committed 64b101c)
`scripts/gen_paid_sound.py` (pure stdlib) synthesizes `frontend/src/assets/sounds/paid-settle.wav` — a two-tone low "settle" (root 208Hz + fifth 312Hz, warm attack, fast decay): **18 KB, 0.42s** (budget ≤50KB/<1s). `frontend/src/lib/sound.ts` is the **only `new Audio()` in the repo** (grep-verified) — asset bundled via Vite → `go:embed` → Wails asset server, no network. It plays only when a receipt **fully settles** a customer invoice to PAID (client-side gate on the same `0.001` tolerance the settlement policy uses), only after the posting call resolves without error, only on the acting user's click. Opt-out setting `sound_on_paid_enabled` (default ON) added to the settings map + `SettingsScreen` toggle + `soundSettings` store. `.gitattributes` now pins `*.wav/ogg/mp3` binary.
**Audit — audio = one call site:** ✅. **No sound on any other event** (not errors, not saves, not arrivals).

### B5 — Rituals, operator language, brand slot ⏳ (Batch 2: B5a checklist, B5bc language+brand)

### B6 — Toast discipline ⏳ (Batch 3)

---

## Gate results
_(final)_

## Taste Ledger
_(every aesthetic decision, alternatives considered, what to review first — grows per batch)_

**What to review FIRST:** `DealTimeline.svelte` on an Order detail view (the signature) — open an order that is part-way through its life and one that's fully PAID. Then the paid-settle sound (record a receipt that fully settles an invoice) — this is the whole audio budget; re-roll it in `scripts/gen_paid_sound.py` if the character's wrong.

**B2 — motion easing [TASTE].** `--ease-standard` kept bit-identical to the pre-existing `--easing-smooth` (0.25,0.1,0.25,1) so nothing already using it shifts feel — only the vocabulary consolidates. `--ease-decelerate` = Material decelerate (0,0,0.2,1); considered ease-out-expo (0.33,1,0.68,1) but rejected as too flat-tailed ("sluggish stop") in a 120–260ms window. Spring/bounce deliberately excluded (trading desk, not a game). Toasts enter via `fly` (−8px, "arriving from above the stack"), modals via `fade`+`scale` ("growing into place") — matching what each already implied, just standardized. **Owner look:** confirm the modal/toast feel; note `IntelligenceHub`'s screen-mount is still 0.5s (curve-only was in scope) — shorten it too if you want.

**B3 — timeline shape [TASTE].** Built the owner-ratified compact dot-and-rule stepper. Alternatives considered & rejected: vertical timeline card (heavier, not "compact"); progress-bar-with-percent (hides which stage is missing); icon-per-stage (adds visual weight the brief avoids). Colors use monochrome contrast tokens, not hues — so the future green-accent can't collide with a "done = green" (there is no green in the timeline). **Owner look:** is the dot/label density right at typical order widths?

**B4 — the sound [TASTE].** A two-tone wooden "settle" (208Hz root + 312Hz fifth, warm attack, fast decay, 0.42s), not a chime/fanfare. Re-roll params are named constants at the top of `scripts/gen_paid_sound.py` (pitch via `ROOT_FREQ_HZ`/`FIFTH_RATIO`, per-tone timing/gain, warmth partial, master gain). **Owner look:** play it; if it should be lower/warmer/shorter, tweak and regenerate.

### Owner decisions requested (open questions)
1. **B4 autoplay:** `.play()` fires after the `await`-ed posting call resolves (so it never sounds on a failed post) rather than the strict zero-await pattern. For a sub-second local Wails IPC this stays inside Chromium's user-activation window, so it should play — but only real-hardware confirmation is definitive (can't verify audio headlessly). Accept, or want the zero-await variant (which risks sounding on a subsequently-failed post)?
2. **B4 second PAID path:** a separate "Apply receipt to invoice" flow can also bring an invoice to PAID; it's currently SILENT (only the "Record Payment/Receipt" submit sounds). Wiring it too = same one `Audio` construction, just a second call site — more consistent, arguably more correct. Want it wired?
