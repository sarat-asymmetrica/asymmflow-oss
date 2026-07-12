# Wave 9 Spec 02 — One Job, One Path (Money Flows)

**Mission:** Wave 9.2 from `FABLE_WAVE9_UIUX_AUDIT.md` §5 (money-flow path unification) + three owner-ratified follow-ups from the Spec-01 gate.
**Repo:** `asymmflow-oss` (this repo). **Branch:** `feat/fable-wave9-2-money-flows` off `main`. Do not merge or push; leave the branch for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` → this spec.
**Prior art:** `FABLE_WAVE9_SPEC_01_REPORT.md` — read it; this spec builds on its Phase A verdicts and inherits its lessons.

## 0. Read before anything

1. `CLAUDE.md` — repo invariants. **Invariant 5 is load-bearing this wave:** rounding, posting order, and tax behavior are stop-and-report, never judgment calls.
2. `DESIGN_CONSTITUTION.md` — Articles II (patterns), III (one job one path — this wave IS Article III), IV.4 (toasts), VI (components/tokens).
3. `FABLE_WAVE9_UIUX_AUDIT.md` §4.5 (Finance AR/AP findings + **binding keep-list**), §3 themes T1/T4/T6/T7, §5 Wave 9.2.
4. `FABLE_WAVE9_SPEC_01_REPORT.md` — especially the A3 finding and "Known residue".

**Data-sensitivity invariant:** `../ph_holdings` is readable for reference; real client names/figures never enter this repo (`SYNTHETIC_IDENTITY.md`).

## 1. Operating model — who you are

You are an **Opus 4.8 orchestrator acting as a senior designer/tech lead**. **Sonnet 5 subagents write the code** (Agent tool, `model: "sonnet"`, `subagent_type: "general-purpose"`); you write their work orders, review every diff constitutionally before commit, run all gates yourself, and personally fix small substandard details. Spec-01's cadence worked — repeat it, including:

- Coder prompts carry: exact items + acceptance criteria, corrected file anchors, quoted constitution articles, the domain keep-list, gate commands.
- Batch by **disjoint files**; sweeps that touch everyone's output run last.
- **Never trust a subagent's build claim** — gate centrally after each batch.
- Commit per item: `feat(wave9.2): <item> — <what>`. Final: `docs(wave9): spec-02 status report`.

**Lessons inherited from Spec-01 (do not relearn):**
- Screens live at `frontend/src/lib/screens/`; backend via `wailsjs/go/{main.App,CRMService,FinanceService}` bindings. Audit anchors cite deployed PH — verify every anchor here before coding.
- `frontend/dist/index.html` is a committed placeholder that `go:embed` needs; if a build mutates it, `git checkout -- frontend/dist/index.html` before committing.
- Gate baseline: `npx vite build` clean; `npx svelte-check` = **0 errors / 14 warnings**. Any net-new error or warning is a failure.
- **Go is in scope this wave** (unlike Spec-01): if you touch Go, `go test ./...` is part of the gate, and new/changed exported behavior gets tests.

## 2. Phase A — recon (read-only, do first)

Anchor drift bit Spec-01; scout before coding. Record verdicts in the report.

| # | Question | Feeds |
|---|---|---|
| A1 | Map the three supplier-settlement paths on OSS: which frontend surfaces call `RecordSupplierPayment` / `MarkSupplierInvoicePaid` / `UpdateSupplierInvoiceWithPayment` (or OSS-named equivalents), and what does each write (ledger entry? journal? invoice fields only)? | B1, B2 |
| A2 | Where does the app get FX rates today? (FXRevaluationScreen's source, any rates service/table, invoice-stored rate.) Is there one callable source of truth? | B2, B3 |
| A3 | Does a receipt void/reversal backend exist? What does PC-D7's unapplied-remainder transform expose for "apply later"? | B4 |
| A4 | How is the authenticated user's identity available to screens (session store? binding)? Confirm what supplier-invoice approval currently sends. | B1, B2 |
| A5 | Verify anchors for every Phase B/C item on this repo; note drift. | all |

## 3. Phase B — money flows (the wave's core)

**B1 — Close the A3 off-ledger bypass (MANDATORY, FIRST — HIGH severity).** The supplier-invoice Edit modal exposes free-jump `status` (~:1453) and `payment_status` (~:1473) selects; setting `payment_status='Paid'` silently marks the invoice paid with **no ledger entry, no journal, no SoD/3-way-match gate**. Remove all state-advancing controls from Edit (Article III.3); Edit may change descriptive fields only. Lifecycle advances solely through the gated Match→Approve→Settle chain. Check create-form for the same leak. If Go endpoints permit the bypass server-side (`UpdateSupplierInvoice` writing `status`/`payment_status`), field-mask them there too, with a test proving Edit can no longer settle.
**AC:** no UI path advances supplier-invoice state outside the gated chain; a test (Go or documented manual) proves marking Paid via Edit is impossible; descriptive editing still works.

**B2 — Supplier AP unification (one home, one settle path).** (a) Surface supplier invoices in FinanceHub beside Settlements so the bookkeeper's AP loop (intake→match→approve→settle) lives in one hub — reuse the existing screen; don't fork it (Article VI.3). (b) Collapse the three settlement paths into **one gated Settle action/modal**; the losers' UI entry points are removed, and their Go endpoints deprecated or re-pointed if nothing else calls them (report what you did). (c) Approver identity = authenticated user (A4), never a hardcoded string (Article III.4). (d) FX rate sourced from A2's source of truth, shown editable with the sourced default; if no source of truth exists, invoice-stored rate + editable field, and report the gap. (e) Header Subtotal/VAT derived from lines, not independently editable. (f) 3-way-match result rendered in place (per-leg pass/fail), not a vanishing spinner+toast.
**AC:** AP loop completable without leaving FinanceHub; exactly one settle path in UI and (if safely removable) in Go; approver = real user; totals can't disagree with lines.

**B3 — Supplier payment ergonomics.** Create-time FX rate field (sourced default, editable); overpay cap against the invoice's outstanding amount mirroring the AR apply flow's full/partial/over helper; separate paid-expenses rows from supplier payments in the grid (or label them unmistakably).
**AC:** non-BHD payment creation is rate-aware; overpaying an invoice is blocked with an explanatory message; the grid no longer conflates two artifact types silently.

**B4 — Receipts: the on-account dead end.** (a) "Apply unapplied balance" row action on receipts with unapplied remainder → opens the existing apply modal pre-scoped to that receipt (pattern #1; PC-D7 transforms are the backend half). (b) Void/reversal **for unapplied on-account receipts only** — if a reversal backend exists (A3), wire it; if not, implement reversal scoped strictly to receipts with zero applications (with Go tests). Anything touching applied/posted receipts: stop-and-report, do not build. (c) Fix "Avg Days to Collection" (exclude `days_to_payment: 0` receipt rows). (d) Label cleanup: the three "+ Add Receipt" buttons converge on one label; misleading Invoice#/"0d" cells on receipt rows show "—"/on-account instead.
**AC:** an on-account receipt can be applied later from the list; a mistaken unapplied receipt can be corrected; the KPI is no longer corrupted.

**B5 — Expenses: lead with the job.** One Categories/Vendors manager (currently rendered twice) moved behind a "Setup" disclosure; Quick Entry form first at the top; expense approvals reachable from within the Expenses workspace (not a distant tab group); group the entry form's 9 controls into money vs classification.
**AC:** the common job (enter an expense) is above the fold; setup is one instance, discoverable; submit→approve loop visible in one place.

**B6 — Invoice creation de-cluttered.** Move the 12-checkbox PDF field-visibility panel out of creation into a "Customize fields" disclosure at the PDF/preview step (T11). Give the no-unfulfilled-order refusal an explanatory empty state that says *why* and links to Orders. **Do not build a proforma/blank-invoice path** — that's an unratified feature decision; note it as an owner option in the report.
**AC:** invoice creation shows commercial fields only; a user with no eligible order understands what to do next.

**B7 — FinanceHub IA symmetry.** AR/AP naming made parallel (e.g. "Customer Receipts" / "Supplier Payments" — pick one symmetric vocabulary and apply it to tabs and titles); promote Approvals out of the Admin group to sit adjacent to the flows that feed it; B2's supplier-invoice tab placed in the AP cluster.
**AC:** tab vocabulary reads as one system; approvals discoverable next to Expenses/AP.

## 4. Phase C — owner-ratified follow-ups from the Spec-01 gate

**C1 — DashboardScreen tokenization pass (authorized).** Replace the screen's raw-hex stylesheet with Onyx & Ether semantic tokens (Article VI.1), including the hover/focus states Spec-01 added under deviation. **Visual-parity intent:** map each hex to the nearest semantic token; this is a re-plumbing, not a redesign. Kill the deviation.
**AC:** zero raw hex in DashboardScreen styles; screen visually equivalent (spot-check side by side); the Spec-01 deviation is closed.

**C2 — Dashboard pipeline & aging widgets (closes Spec-01's B1.3 residue).** Build the pipeline-by-stage widget on `GetDashboardPipelineByStageYTD` and the AR-aging-buckets widget on `GetDashboardARAgingReportYTD`, each with drill-throughs (stage → filtered Opportunities; bucket → filtered Invoices) using the param-threading Spec-01 built. Canonical components; company-scoped. Batch with C1 — same file, same coder.
**AC:** both Wave 8 YTD backends finally have callers; every widget segment navigates filtered-correct; loading/empty/error states present.

**C3 — Order delete cascade preview (authorized).** Wire `PreviewOrderDeleteCascade` into the Orders delete flow: the confirm modal shows what will be deleted/affected before the destructive action (Article III.2 destructive rung; pattern #7). Restores the keep-list's cascade-preview strength that OSS lacked.
**AC:** deleting an order shows its cascade first; confirm proceeds, cancel is untouched; zero-dependents case says so.

### Suggested coder batching (adjust from Phase A)
- Coder 1: B1 + B2 (supplier invoice screen + FinanceHub + Go masking) — the wave's spine, start immediately after Phase A
- Coder 2: B4 (receipts/payments AR side)
- Coder 3: B3 + B5 (supplier payments + expenses)
- Coder 4: C1 + C2 (DashboardScreen tokenization + widgets)
- Coder 5: B6 + C3 (invoices create + orders delete preview)
- Orchestrator directly: B7 (FinanceHub IA — small, cross-cutting, easier to do than to specify)

## 5. Hard boundaries

- **No Wave 9.3+ work:** recon-screen unification, as-of-date sweep, the broad actor-identity sweep (B1/B2's approver identity is this wave's only T1 fix), cheque↔bank-line linkage, People/Projects re-model — all later waves, even where adjacent.
- **Financial semantics (rounding, posting order, tax math): stop-and-report** (CLAUDE.md 5). Collapsing duplicate paths and masking bypass fields is in scope; *changing what a posting does* is not.
- **Keep-lists binding** (audit §4.5 finance AR/AP above all): the gated Match→Approve→Settle chain, the `pendingReceiptApply` handoff + apply-receipt ergonomics, confirm-twice + posted/paid locks, PH/AHS `matchesCompany` scoping. §4.7's dashboard keep-list applies to C1/C2.
- **No merge, no push, no tag.** Branch stays local for review.
- Layer model respected: UI → bindings → services; no business rules in Svelte.

## 6. Definition of done + status report

Done = Phase A recorded; B1–B7 + C1–C3 shipped or explicitly skipped with reason; gates green on final commit (`vite build` + `svelte-check` at baseline + `go test ./...` if Go touched); report written.

Write `FABLE_WAVE9_SPEC_02_REPORT.md`, commit it, and paste it verbatim as your final message. Use Spec-01's report template (phases table, per-item status+commits, decisions, constitution deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green.
