# Wave 9 Spec-02 Status Report — One Job, One Path (Money Flows)

**Branch:** `feat/fable-wave9-2-money-flows` · **Commits:** 6 feature + 1 docs, off `main` — **NOT merged/pushed**, left for owner review.
**Gates (final commit):** `npx vite build` ✅ (exit 0) · `npx svelte-check` ✅ **0 errors / 14 warnings** (= pre-wave baseline; net **0 new errors, 0 new warnings**) · `go build ./...` ✅ · `go vet` ✅ · `go test ./...` ✅ (exit 0, all packages; new bypass test `TestUpdateSupplierInvoice_CannotBypassPaymentLifecycle` passes).

**Operating model:** Opus 4.8 orchestrator as senior designer/tech-lead (Phase-A recon work orders, constitutional review of every diff before commit, central gating never trusting a subagent's build claim, and personal correction of substandard/financial-risk details). Five Sonnet 5 `general-purpose` subagents wrote code in disjoint-file batches; the orchestrator authored B7 directly and made three financial-integrity corrections (below).

---

## Phase A — ground-truth verdicts (recon before code; anchor drift was significant)

| # | Question | Verdict |
|---|---|---|
| **A1** | Three supplier-settlement paths + what each writes | **CONFIRMED, drift corrected.** `RecordSupplierPayment`/`UpdateSupplierPayment` (SupplierPaymentsScreen), `MarkSupplierInvoicePaid` (SupplierInvoicesScreen:732), `UpdateSupplierInvoiceWithPayment` (Edit save). **All settlement is ledger-table-only** (`SupplierPayment` rows via `applySupplierInvoicePaymentState`) — **no GL journal** is ever posted for supplier settlement (only expenses/payroll post journals). `SupplierInvoicesScreen` was mounted in **OperationsHub**, not FinanceHub. |
| **A2** | FX rate source of truth | **NONE.** Four disconnected mechanisms (`CurrencyExchangeRate` table, `FXRate` table, overlay `ExchangeRateToBase`, and hardcoded per-country literals in SupplierInvoicesScreen `:585-601` / SupplierPaymentsScreen). No single callable binding returns "the" rate. `GetSupportedCurrencies` carries no rate. → used invoice-stored rate + editable field, gap reported (per B2(d)). |
| **A3** | Receipt void/reversal + apply-later backend; OSS receipt UI | **OSS HAS NO RECEIPT UI.** `PaymentsScreen` is a `Payment`-only screen. `CustomerReceipt` backend (`ApplyCustomerReceiptToInvoice`, etc.) exists but has **zero frontend callers**; no `pendingReceiptApply` store; **no void/reversal backend** (`Status:"Reversed"` is checked, never assignable). Matches ratified **PC-D10 (payments-only, DIVERGENT-INTENTIONAL)**. → B4 is a documented skip (see below). |
| **A4** | Authenticated user identity | `frontend/src/lib/stores/authContext.js` `currentUser` (from `GetCurrentUserStub`) carries `.id` = **User.ID**. Backend `getCurrentUserID()` (which stamps `CreatedBy`) prioritizes **EmployeeID**. SupplierInvoicesScreen sent `systemUser='System Admin'` (:155/:702), which the backend substituted with `getCurrentUserID()`. **The two resolvers diverge** — load-bearing for the SoD fix below. |
| **A5** | Anchors on OSS | **Heavy drift, corrected per item.** Highlights: Expenses managers were **never duplicated** on OSS (single render site — audit premise false); Dashboard has **64 raw hex / 1 token** and **no pipeline/aging widgets** (build from scratch, not "make drillable"); Orders delete uses a **screen-local** confirm, `PreviewOrderDeleteCascade` unused; SupplierInvoices Edit selects at `:1453`/`:1473` (not `:1447`/`:1467`). |

---

## Phase B / C — items

| Item | Status | Commit | Notes |
|---|---|---|---|
| **B1** Close A3 off-ledger Edit bypass (MANDATORY) | **shipped** | `2a38b18` | Edit-modal `status`/`payment_status` selects + payment-ledger checkbox removed; Edit is descriptive-only. Go `UpdateSupplierInvoice` field-masks the 5 lifecycle fields from the persisted row. New hermetic Go test proves marking Paid via Update is impossible (return + DB reload + zero `SupplierPayment`). Create form already hardcodes Pending/Unpaid — no leak. |
| **B2(a)** Supplier invoices in FinanceHub | **shipped** | `8b03c98` | Surfaced beside Supplier Payments; **moved** out of OperationsHub (not duplicated — Article III.1). Screen reused as-is (`embedded`). |
| **B2(b)** One gated settle path | **shipped** | `2a38b18` | Only `MarkSupplierInvoicePaid` (Approved-only) remains in UI. `UpdateSupplierInvoiceWithPayment` now UI-orphaned (Go kept; deprecation = owner call). |
| **B2(c)** Real approver identity | **shipped (orchestrator-corrected)** | `2a38b18` | Ghost `'System Admin'` removed; UI shows the operator + blocks if unknown (III.4). **Attribution resolved server-side** via `getCurrentUserID()` to keep SoD sound (see Decision 1). |
| **B2(d)** FX from source | **shipped + gap reported** | `2a38b18` | Fake per-country default removed; editable rate field kept. No FX source-of-truth exists (owner question 1). |
| **B2(e)** Derived header totals | **shipped** | `2a38b18` | Create-modal Subtotal/VAT read-only, driven by `recalcLineItems()`; latent bug fixed (add/remove line now recalcs). VAT math untouched (flat 10%). |
| **B2(f)** In-place 3-way-match result | **shipped** | `2a38b18` | Per-leg PO/GRN pass-fail + overall status rendered in place; re-hydrates from server; canonical `WabiModal`/`Button`. |
| **B3** Supplier payment ergonomics | **shipped (orchestrator-corrected)** | `3943378` | Create-time editable FX field (invoice-sourced default); client overpay cap + "Pay Full Outstanding"; expense settlements de-conflated (pill badges + segmented filter). **Create-then-patch `amount_bhd` hack removed** (Decision 2). |
| **B4** Receipts: on-account dead end | **SKIPPED — premise absent on OSS** | (report) | OSS has no receipt UI; `CustomerReceipt` backend has zero callers; no reversal backend; KPI/labels/buttons the audit cites don't exist here. Ratified **PC-D10 payments-only**. Building a receipts screen is net-new feature work (owner question 3), not path-unification. Honest skip (mirrors Spec-01's "B1.3 not invented"). |
| **B5** Expenses: lead with the job | **shipped** | `8fa0934` | Quick Entry first, grouped Classification vs Money & Dates (FormGroup); single manager behind a Setup disclosure. Audit's "rendered twice" premise corrected — was never duplicated on OSS. |
| **B6** Invoice creation de-cluttered | **shipped** | `f0b3618` | 12-checkbox PDF panel behind a "Customize fields" disclosure (T11); no-order refusal → explanatory empty state + "Open Orders" link. No proforma path (owner question 5). |
| **B7** FinanceHub IA symmetry | **shipped (orchestrator)** | `8b03c98` | Customer Invoices/Customer Payments (AR) then Supplier Invoices/Supplier Payments (AP); Approvals → "Expense Approvals" beside Expenses; supplier-360 drill re-pointed to Finance. |
| **C1** DashboardScreen tokenization | **shipped — deviation closed** | `d93da32` | ~64 raw hex → loaded Onyx & Ether tokens; zero bare hex; visual-parity. Spec-01 deviation closed. |
| **C2** Pipeline & aging widgets | **shipped (Spec-01 B1.3 residue closed)** | `d93da32` | Both Wave 8 YTD backends now have callers, with drill-throughs + loading/empty/error states. |
| **C3** Order delete cascade preview | **shipped** | `f0b3618` | `PreviewOrderDeleteCascade` wired ahead of the confirm; blocked-payments case surfaced (no proceed); zero-dependents message; confirm-twice preserved. |

---

## Decisions taken (orchestrator financial-integrity corrections)

1. **Approver attribution is server-resolved, not client-supplied (B2c).** The coder passed `$currentUser.id` to `ApproveSupplierInvoice`. But `CreatedBy` is stamped by `getCurrentUserID()` (EmployeeID-first) while `$currentUser.id` comes from `GetCurrentUserStub` (User.ID-first) — two different resolvers. Since the SoD gate compares `CreatedBy == approver`, a client User.ID could bypass creator≠approver for the same human and let a creator approve their own invoice. **Fix:** pass empty → the backend resolves via `getCurrentUserID()` (the same resolver that set `CreatedBy`), restoring the pre-wave secure behavior; the UI still identifies + gate-checks the operator. Keeps a keep-list control (the gated chain / SoD) intact.

2. **Removed the B3 create-then-patch `amount_bhd` workaround.** The coder recorded a payment at the backend's implicit 1:1 rate, then fired a second `UpdateSupplierPayment` to overwrite `amount_bhd`. That **changes posting order** and risks persisting a wrong BHD value on partial failure — squarely CLAUDE.md invariant 5 (stop-and-report), and the spec's "changing what a posting does is not [in scope]." Kept the editable FX field + overpay cap (client-side, in scope); routed non-BHD BHD-value correction through the existing Edit path; reported the backend gap (owner question 1). B2(d)'s ratified guidance was "editable field + report the gap," not a posting workaround.

3. **Supplier invoices moved, not duplicated.** Rather than leave the screen in OperationsHub and also add it to FinanceHub (a duplicate path — Article III.1 defect), it was removed from OperationsHub; the supplier-360 drill-through and OperationsHub tab-count wiring were updated to match.

4. **B4 skipped, grounded in PC-D10.** The audit's receipt findings are from deployed PH (which has a receipt UI); OSS ratified payments-only. Reported honestly rather than inventing a receipt screen.

## Constitution deviations
- **None net-new.** Article VI: a handful of C1 values with no exact loaded token use `var(--invented-name, #fallback)` — the same convention Spec-01 ratified for new code and FinancialDashboard already uses; **zero bare hex** remains. This **closes** the standing Spec-01 DashboardScreen raw-hex deviation rather than opening one.
- No financial-semantics changes shipped (VAT stayed flat-10%; no rounding/posting-order/tax math touched — the one posting-order workaround a coder attempted was removed). No secrets, no real client data, layer model respected (UI → bindings → services).

## Keep-list attestation (audit §4.5 AR/AP + §4.7 dashboard)
- **Gated Match→Approve→Settle chain:** preserved and *strengthened* — the off-ledger Edit bypass is closed; the single settle path stays Approved-only; **SoD integrity restored** (Decision 1). ✅
- **`pendingReceiptApply` + apply-receipt ergonomics:** N/A on OSS (no receipt UI); untouched. ✅
- **Confirm-twice + posted/paid locks:** preserved (supplier approve confirm; orders delete double-submit guard intact, cascade preview added in front). ✅
- **PH/AHS `matchesCompany` scoping:** untouched; SupplierInvoicesScreen rendered `embedded` exactly as before (its company handling unchanged — see residue). ✅
- **§4.7 dashboard keep-list:** role-adaptive rendering, Operating Focus deep-links, existing KPI-card drills all preserved; new widgets follow the same `navigateTo` param pattern. ✅

## Known residue / follow-ups
- **Supplier-payment FX persistence (backend gap).** `RecordSupplierPayment` has no `exchangeRate` param and posts non-BHD `amount_bhd` at 1:1; the create-time field is advisory + drives the overpay check, with Edit-after-record as the correction path. Server overpay guard also runs at 1:1 (can false-reject valid non-BHD payments). Needs a backend param (owner question 1).
- **`UpdateSupplierInvoiceWithPayment`** — UI-orphaned but still live in Go; awaiting a deprecate/remove decision.
- **Supplier-invoice company scoping** — the screen isn't `company`-scoped (wasn't before either); it renders `embedded` in FinanceHub without the hub's company selector filtering it. Pre-existing; a later wave should thread company.
- **AR-aging drill granularity** — the 30/60/90/120+ buckets all drill to `invoiceFilter:"Overdue"` because InvoicesScreen has only status filters, no day-range. Needs an InvoicesScreen filter to drill per-bucket.
- **`Sidebar.svelte:32`** lists `supplier-invoices` under a children array, but that component is only used by `LayoutExample.svelte` (a demo, not the live shell) — left as-is.

## Open questions for the owner
1. **Supplier-payment FX backend param (financial-semantics).** Authorize adding an `exchangeRate` param to `RecordSupplierPayment`/`recordSupplierPayment` so non-BHD payments post the confirmed BHD value in one write (and the overpay guard uses the real rate)? This is a posting change — stop-and-report per invariant 5.
2. **`UpdateSupplierInvoiceWithPayment`** — deprecate/remove now that nothing in the UI calls it, or keep for API compatibility?
3. **Receipts (B4).** Confirm payments-only (PC-D10) is the intended end state, or schedule a receipts screen (apply-later / on-account reversal) in a later wave? The `CustomerReceipt` backend exists but is unwired.
4. **B6 proforma/blank-invoice path** — build a scratch-invoice path (unratified feature), or is the order-required + explanatory empty state the intended behavior?
5. **AR-aging bucket drills** — add a day-range filter to InvoicesScreen so each aging bucket drills to its own bucket, or accept the shared "Overdue" filter for now?

---

*Definition of done: Phase A recorded ✅ · B1–B3, B5–B7, C1–C3 shipped ✅ · B4 explicitly skipped with reason ✅ · gates green on final commit ✅ · report written ✅. Branch left local for owner review — no merge, no push, no tag.*
