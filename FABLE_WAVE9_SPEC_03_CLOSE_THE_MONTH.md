# Wave 9 Spec 03 — Close the Month (Recon + Trust) + Ratified AR/AP Follow-ups

**Mission:** Wave 9.3 from `FABLE_WAVE9_UIUX_AUDIT.md` §5 (reconciliation, actor identity, guard calibration, reporting trust) + five owner-ratified follow-ups from the Spec-02 gate.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-3-close-the-month` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` → this spec.
**Prior art:** `FABLE_WAVE9_SPEC_01_REPORT.md` + `FABLE_WAVE9_SPEC_02_REPORT.md` — read both; Spec-02's Phase A verdicts (no FX source of truth, no receipt UI, identity-resolver divergence) are load-bearing here.

## 0. Read before anything

1. `CLAUDE.md` — invariant 5 (financial semantics = stop-and-report) applies **except where this spec records an explicit owner authorization** (C1, C4 — scoped there).
2. `DESIGN_CONSTITUTION.md` — Articles III (guard ladder, no ghost actors, as-of dates — this wave IS Article III.4/III.5), II, VI.
3. `FABLE_WAVE9_UIUX_AUDIT.md` §4.6 (recon/accounting findings + **binding keep-list**), §4.5, §3 T1/T8/T9/T12, §5 Wave 9.3.
4. Both prior reports.

**Data-sensitivity invariant:** `../ph_holdings` readable for reference; real client names/figures never enter this repo.

## 1. Operating model

Identical to Spec-02, which worked: **Opus 4.8 orchestrator as senior designer/tech-lead** (Phase-A recon work orders → disjoint-file coder batches → constitutional review of every diff → central gating → personal correction of financial-risk details); **Sonnet 5 subagents** (`model: "sonnet"`, `subagent_type: "general-purpose"`) write the code.

**Lessons inherited (do not relearn):**
- Audit anchors cite deployed PH and have drifted hard every wave — verify every anchor here before coding.
- `frontend/dist/index.html` is a committed `go:embed` placeholder; `git checkout -- frontend/dist/index.html` after any build mutates it.
- Gate baseline: `npx vite build` clean; `npx svelte-check` = **0 errors / 14 warnings**; `go test ./...` green. Net-new errors/warnings = failure.
- **Identity is resolved server-side** (Spec-02 Decision 1): where attribution or SoD matters, do not trust a client-supplied user id — backend `getCurrentUserID()` is the resolver that stamps `CreatedBy`; keep both sides of any comparison on the same resolver. The UI's job is to *display* the operator and block when unknown.
- Watch file overlap between batches: B1/B2/B3 all touch the recon screens — assign the recon-screen portions of B2/B3 to the same coder as B1.

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Do `GetDepositsInTransit` / `GetOutstandingCheques` (or equivalents) exist on OSS? What do the book-bank recon + FX reval models store for dates (as-of capability)? How does the cheque register relate to bank statement lines today? | B1, B3 |
| A2 | Map `Payment` vs `CustomerReceipt` models and bindings: fields, overlap, what PC-D7's unapplied-remainder transform produces, which surface renders AR money-in today. | C3 |
| A3 | Per ghost-actor site (GRN received_by/qc_by, bank-recon finalize/match/unmatch, book-bank finalize, FX post, audit reverse, costing Prepared-By): does the backend stamp identity server-side already, or does it persist the client string? Does `GetPreparedByOptions` (Wave 8) suit costing? | B2 |
| A4 | ReportsScreen + AccountingScreen today: which categories render, what export helpers exist (PDF/CSV/print) to reuse for statement/GL export? | B5, B6 |
| A5 | `UpdateSupplierInvoiceWithPayment` caller census: frontend, Go tests, sync/importers — is removal safe? | C2 |
| A6 | Verify anchors for every B/C item; note drift. | all |

## 3. Phase B — close the month (the wave's core)

**B1 — One guided month-end reconciliation flow.** Bank recon (match) and book-bank recon (prove) are two halves of ONE monthly task, currently two unlabelled sibling tabs — and the prove half cannot balance (Deposits-in-Transit and Outstanding Cheques are displayed with NO input path; `new Date()` stamped as the rec date).
(a) Name and link the two-step sequence (e.g. "Close the month: 1 Match transactions → 2 Prove the balance") — guided handoff from step 1's finalize into step 2 (pattern #1), whether as one screen or two cross-linked tabs (orchestrator's design call; one *flow* either way).
(b) DIT + outstanding-cheque line inputs on the prove step; outstanding cheques **pre-populated from the cheque register** (A1), editable.
(c) **As-of date picker** (month-end), persisted with the rec — no silent `new Date()` (Article III.5, T8).
(d) Statement-import preview: parsed rows shown for confirm before commit.
(e) Cheque↔bank linkage (Spec-01 residue): "Mark Cleared" currently passes an empty `bankStatementLineID` — when clearing happens during recon matching, link the statement line; from the register, allow picking the matched line or route the user to recon. A cleared cheque should know which bank line cleared it.
(f) Bank-account CRUD relocated out of the recon screen (to Settings/Admin surface) if verified present here.
**AC:** a bookkeeper can complete match→prove for a chosen month-end date and the prove step can actually balance with DIT/outstanding inputs; both halves visibly belong to one flow; cheques cleared via recon carry their statement line.

**B2 — Ghost-actor sweep (T1, Article III.4).** Thread real identity through every site A3 confirms: GRN `received_by`/`qc_by`, bank-recon finalize/match/unmatch, book-bank finalize, FX post, audit-trail reverse, costing Prepared-By (wire `GetPreparedByOptions` if it fits, else current-user default). Prefer server-side resolution (see Lessons); the UI displays the operator and blocks when identity is unknown. No `'admin'` / `'System User'` / `'System'` literals remain in these paths.
**AC:** grep for the ghost literals in the swept paths returns zero; each swept action persists a real, consistent identity; unknown identity blocks with an explanatory message rather than mis-attributing.

**B3 — Guard calibration (T9, Article III.2).** (a) FX revaluation: bare row-click must NOT post — explicit "Post revaluation" button + confirm primitive stating the consequence; "Revalue All" confirmed with count + as-of date; update-rate modal shows "was X → now Y". (b) FX reval takes an as-of date (T8). (c) Audit-trail row-click opens details; Reverse becomes a distinct guarded action (currently row-click opens the REVERSE modal). (d) Project delete upgraded to match task delete's two-press guard (the one Work-domain item assigned to 9.3 by the audit).
**AC:** no posting or destructive action fires from a bare row-click anywhere touched; every guard states its consequence; FX posts carry their as-of date.

**B4 — The two-P&L problem (T12) — labeling only.** AccountingScreen's live operational P&L vs FinancialDashboard's audited/imported figures for the same year, unexplained. **Default authority ruling (owner-approved direction, refine wording freely):** label AccountingScreen output "Live books (unaudited, real-time)" and FinancialDashboard's historicals "Audited/filed figures (imported)", each with a one-line note naming the other and when to trust which. No reconciliation math, no data changes — labels and a short explanation only.
**AC:** a user seeing either P&L knows which one they're looking at and why the other differs.

**B5 — Statements out the door.** Export for Balance Sheet, General Ledger, and Journal (reuse A4's existing export machinery; CSV at minimum, PDF where the machinery already offers it) — the owner must be able to send statements to their accountant.
**AC:** BS/GL/Journal each have a working export honoring the active date range + company scope.

**B6 — Reports speak trader (T11).** (a) Fix the catalog so every category's packs render (not only under the "financial" tab). (b) Replace RUNWAY/BURN/MRR framing with trader vocabulary (cash position, receivables/payables, margin) — relabel/reframe existing computations; do not invent new metrics. (c) Non-financial categories stop dumping auto-labelled numeric KPIs — give them their real catalogs or fold them away honestly.
**AC:** no SaaS vocabulary remains; every advertised report renders or is removed.

## 4. Phase C — owner-ratified follow-ups (Spec-02 gate, 2026-07-10)

**C1 — Supplier-payment FX persistence (AUTHORIZED posting change — the only one).** Add an `exchangeRate` parameter to `RecordSupplierPayment`/`recordSupplierPayment` so a non-BHD payment posts its confirmed BHD value in **one write** (no create-then-patch), and the server overpay guard evaluates at the real rate instead of 1:1. Wire the Spec-02 create-time FX field to it. Existing rounding conventions unchanged; BHD stays 3-decimal. **Go tests required:** non-BHD posts correct `amount_bhd`; overpay guard respects the rate; omitted rate falls back to current behavior (BHD 1:1).
**AC:** tests green; the Spec-02 residue ("advisory-only FX field", "1:1 false rejections") is closed.

**C2 — Deprecate `UpdateSupplierInvoiceWithPayment` (ratified: deprecate if not useful).** If A5 confirms zero callers: remove the Go method + regenerate/prune wailsjs bindings. If anything depends on it: keep, mark deprecated in a comment pointing at the gated settle path, and report.
**AC:** the bypass-era endpoint is gone or formally deprecated; build + tests green either way.

**C3 — Receipts workspace (ratified net-new feature).** The `CustomerReceipt` backend (apply-to-invoice, PC-D7 unapplied-remainder transforms) exists with zero frontend callers. Build the AR money-in surface for it — **without creating a second parallel money-in screen** (Article III.1): one AR surface (extend/absorb the existing Customer Payments tab as A2's model map dictates) where the user can (a) record a receipt (applied and/or on-account), (b) see unapplied/on-account balances, (c) "Apply unapplied balance" → apply modal pre-scoped (pattern #1) with the full/partial/over helper ergonomics the audit praises, (d) reverse a **fully-unapplied** receipt only — wire an existing reversal backend if A2 finds one, else implement reversal scoped strictly to zero-application receipts with Go tests. Applied/posted receipt reversal: stop-and-report.
**AC:** on-account money-in is recordable, visible, applicable-later, and correctable pre-application; exactly one AR money-in path exists in the UI.

**C4 — Proforma invoice path (ratified feature).** Invoice creation currently refuses without an unfulfilled order. Add a proforma path: create an invoice-shaped document without an order, unmistakably typed/badged "Proforma". **Hard financial rule:** a proforma posts NOTHING — no revenue recognition, no AR aging, no VAT liability — until explicitly converted to a final invoice (conversion = guarded single-purpose action, pattern #5; conversion may attach an order or proceed orderless if the model allows). Numbering: distinct proforma sequence (e.g. `PF-…`) so fiscal numbering stays gapless — flag in the report if the numbering engine constrains this.
**AC:** a proforma can be created, printed/exported, and later converted; unconverted proformas appear in no financial report/aging; the fiscal number sequence is unaffected by proforma creation.

**C5 — Aging drills land on their bucket (ratified).** Add a day-range/aging-bucket filter to InvoicesScreen (30/60/90/120+), and re-point the dashboard AR-aging drills (Spec-02 C2) from the shared "Overdue" filter to the specific bucket clicked.
**AC:** clicking the 90-day bar shows only that bucket's invoices; the filter is also usable directly on InvoicesScreen.

**C6 — Supplier-invoice company scoping (Spec-02 residue).** Thread the PH/AHS company selection through the embedded SupplierInvoicesScreen in FinanceHub, consistent with `matchesCompany` scoping elsewhere (keep-list behavior).
**AC:** switching company in FinanceHub filters supplier invoices like its sibling tabs.

### Suggested coder batching (adjust from Phase A; respect file overlap)
- Coder 1: B1 + recon-screen portions of B2/B3 (bank/book-bank/cheque/FX screens are one file cluster)
- Coder 2: B2/B3 remainder (GRN, costing Prepared-By, audit-trail, project-delete guard)
- Coder 3: C3 receipts workspace (largest net-new — start early)
- Coder 4: C1 + C2 + C6 (supplier AP: Go param + deprecation + scoping)
- Coder 5: B5 + B6 + C5 (reports/exports/filters) then C4 (invoices)
- Orchestrator directly: B4 (small, judgment-heavy wording)

## 5. Hard boundaries

- **No Wave 9.4+ work** (People/projects re-model, employee identity home, payroll placement) beyond B3(d)'s project-delete guard; **no Wave 9.5 polish**; no sensory/brand work (own wave, owner-reserved).
- **Financial semantics: stop-and-report** — with exactly two scoped authorizations recorded here: C1's `exchangeRate` param and C4's proforma-posts-nothing rule. Rounding, VAT math, posting order otherwise untouched.
- **Keep-lists binding:** §4.6 (bank-recon Finalize/Reopen gating, split-allocation match modal, Fix-Debit/Credit path, Accounting date presets + statement layout, cheque next-number preview, FinancialDashboard drill-throughs) and §4.5 (gated settle chain, confirm-twice + posted/paid locks, `matchesCompany` scoping).
- **No merge, no push, no tag.** Branch stays local for review.
- Layer model respected; no business rules in Svelte.

## 6. Definition of done + status report

Done = Phase A recorded; B1–B6 + C1–C6 shipped or explicitly skipped with reason; gates green on final commit (`vite build` + `svelte-check` baseline + `go test ./...`); report written.

Write `FABLE_WAVE9_SPEC_03_REPORT.md`, commit it, and paste it verbatim as your final message (Spec-02's template: Phase A verdicts, per-item status+commits, decisions, constitution deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green.
