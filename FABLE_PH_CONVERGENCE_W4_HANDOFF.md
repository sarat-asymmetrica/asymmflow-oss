# FABLE CAMPAIGN — PH CONVERGENCE WAVE 4: "Prove the Parity, Certify the Cutover"

Written 2026-07-07 by the Opus 4.8 instance running the Commander's
(Sarat's) session, for the Fable 5 instance that will run this wave
autonomously. Parent campaign: `FABLE_CAMPAIGN_PH_CONVERGENCE.md`
(read it first — the Freeze Law, the thesis, and Missions A–F are its
context). This wave is **Mission G**, the natural successor to Mission A:
Mission A re-measured the *known* divergence ledger; Mission G measures
the *unknown* — the full feature-and-flow surface — and certifies the
cutover.

The Commander is available for the narrow stop-and-ask set in §6. He is
**not** available for anything this document already decides (§3). When in
doubt between asking and measuring: measure. The ground wins.

---

## 0. What this wave IS — and is NOT

**IS:** Prove that AsymmFlow-OSS (this repo, `main` @ `801f41a`) is at
full **feature and flow parity** with the deployed PH app
(`C:\Projects\asymmflow\ph_holdings`, branch `ui-ux-hardening`, frozen at
`ca24372`), close every parity gap the audit surfaces, and emit a
**cutover-readiness certificate** the Commander reads to schedule the
switch.

**IS NOT:**
- **Not data migration / data-quality work.** The `phimport` path, the
  darch backfill, the 355-op correction patch, OCR — all deferred to a
  later dedicated wave (Mission H). The Commander's standing call: *data
  is a re-runnable dial; parity is the one-way door.* Do not touch data
  convergence here. (Exception — schema *provisioning* parity is in
  scope; see §4 G.1. Provisioning an empty table is parity. Deciding what
  rows flow into it is the deferred data wave.)
- **Not the mesh campaign** (`FABLE_CAMPAIGN_SOVEREIGN_MESH.md`). Separate
  track, separate wave.
- **Not a re-port of the ledger.** As of the 2026-07-07 audit (§1), Waves
  1–3 already shipped essentially the entire `PH_CONVERGENCE_LEDGER.md`
  PORT surface. Do not re-port shipped work; verify-then-skip.

---

## 1. Where we stand (the discovery that scopes this wave)

`docs/PH_CONVERGENCE_LEDGER.md` was Mission A's map of the **known**
divergence. A 2026-07-07 re-audit of its highest-value PORT rows found
**10 of 11 SHIPPED, 1 PARTIAL, 0 fully pending**:

| Ledger row | Status now | Evidence |
|---|---|---|
| F credit-limit override | SHIPPED | `customer_invoice_service.go:633` via `pkg/approvals`; `credit_limit_exceeded` Finding `:1042`; audit row `:1118` |
| B4/B4a Draft totals + zero-rated | SHIPPED | recompute `customer_invoice_service.go:1345`; rate recovery `:1334` |
| 1-FM field-mask | SHIPPED | load-then-overlay: `supplier_invoice_service.go:311`, `grn_service.go:290`, `app_order_customer_surface.go:1480` |
| 1-RBAC unguarded mutators | SHIPPED | permission gates added (`app_watcher.go:245`, `bank_accounts_service.go:33`, …) |
| 1-HMAC verify+backfill | SHIPPED | `VerifyInvoiceHash` `:128`, `backfillInvoiceHashesInternal` `:80`, startup `app.go:1001` |
| 1-HOLLOW send-block | SHIPPED | item-count guard `customer_invoice_service.go:1438` |
| 1-POVAT VAT basis | SHIPPED (fixed) | BHD basis `purchase_order_service.go:171` |
| A1 flexible dates | SHIPPED | `pkg/crm/order_json.go:14`, `pkg/finance/expense_json.go:14` |
| C invoice-PDF bank filter | SHIPPED | division filter `invoice_pdf_service.go:661` |
| 3-PLAT field-crypto salt | SHIPPED | `appDataDirPath()` + writability probe `field_crypto.go:352` |
| D1/D2 digit-guard | **PARTIAL** | `folderNumberHasDigit` wired (`onedrive_import_service.go:2035/668`); paired `cleanLooseOneDriveFolderNumberToken` never ported — zero hits |

**So the porting backlog is closed. The remaining risk is the surface the
ledger never mapped.** The ledger only ever tracked the June-29 SPOC
deltas + June deployment-readiness invariants. "Full feature and flow
parity" is a larger claim: *every screen, every bound flow, every report
and PDF, every startup/scheduled job* in the deployed app has a
counterpart here — or a **ratified reason** it does not. Mission G maps
and closes that.

**Freeze-Law check (parent §2):** `ph_holdings` has zero commits since
2026-07-04 (last is `ca24372`, 2026-06-29). The comparison baseline is
current; no exception-class commits to fold in. Re-verify at wave start.

---

## 2. The parity definition (internalize before measuring)

**Parity = feature and flow COVERAGE, not bug-for-bug behavioral
identity.** The substrate is allowed to be its own thing where it
deliberately chose to be. Every surface element of deployed PH classifies
as exactly one of:

- **PRESENT** — OSS has an equivalent flow; behavior matches. Parity.
- **DIVERGENT-INTENTIONAL** — OSS behaves differently *by design*; recorded
  in the divergence register with the reason and the trade-off. **Not a
  gap. Do not "fix" it.** (The Commander has pre-ratified two — §3.)
- **DIVERGENT-GAP** — OSS behaves differently and it's a genuine parity
  hole (missing validation, wrong number, absent step). Close it,
  golden-first.
- **ABSENT** — deployed PH has a flow OSS lacks entirely. Close it
  (money/integrity first) or, if it's genuinely obsolete, record why with
  Commander sign-off.
- **EXTRA** — OSS has a flow PH lacks. Note it (it's substrate value); no
  action.

The certificate's power is the honest, explicit **DIVERGENT-INTENTIONAL
register** — "everything the live app does, the substrate does, and here
are the N places it deliberately does *better*, each signed off."

---

## 3. The Commander's pre-decided calls (Fable: do NOT re-ask these)

1. **Scope:** parity + certificate ONLY. No data convergence, no mesh.
   (§0.)
2. **Behavior fork — party-delete (ledger row E):** **KEEP OSS behavior.**
   OSS orphans a party's transactional children on an admin/approved
   delete; deployed PH blocks it with a child-count guard. Record as
   **DIVERGENT-INTENTIONAL** with the honest integrity note: *an
   admin/approved delete in OSS can orphan a party's Orders/Invoices/
   Offers/Opportunities (supplier mirror: PO/SupplierInvoice/
   SupplierPayment); the substrate accepts this as a deliberate choice.*
   Do **not** port PH's `*_HAS_LINKED_RECORDS` guard. (Register it clearly
   enough that a future Commander can revisit with eyes open.)
3. **Behavior fork — bank-statement lifecycle (Band-2 row 6-8):** **KEEP
   OSS behavior.** OSS's banking engine silently auto-reverts a
   Reconciled/Verified statement to InProgress on a line change
   (`pkg/finance/banking/service.go:646`); PH refuses edits until the
   statement is explicitly reopened. Record as **DIVERGENT-INTENTIONAL**.
   Do not port PH's refuse-until-reopened lifecycle.
4. **1-SYNC (ledger row):** PH's sovereign fork adopts the OSS **Turso/CDC**
   forward path (`pkg/sync/turso/cdc.go`). The three legacy Postgres-sync
   hardenings are **moot by design** — record them as not-ported (aligns
   the fork with the substrate's forward direction), do not port them.
5. **Synthetic-canon cleanup:** the demo bank seed uses slugs
   `bank-ahli` / `bank-bbk`, which read as real Bahraini banks. Rename to
   clearly-synthetic slugs (e.g. `bank-alpha` / `bank-beta`) — invariant
   #2 (no real-world identifiers in the repo). Small, do it in G.2. No
   decision needed.
6. **C invoice-PDF appearance:** the remaining PDF sub-items are cosmetic
   (40 mm vs 50 mm top margin, ref-number width-truncation helper, buyer-
   address `Attention*` fallback, blank-signature layout). Customer-facing
   financial-document **appearance is stop-and-ask** (§6). Measure and
   propose in the certificate; change nothing without the Commander.

---

## 4. The Missions (risk-retirement order — cheapest falsification first)

### G.1 — The parity map (THE deliverable; do this first and mostly)

Produce `docs/PH_PARITY_MAP.md`: a systematic, honest classification of
deployed PH's entire surface against OSS. This is a **code comparison**,
read-only on the PH side. Two roots:
- **Deployed (source of truth):** `C:\Projects\asymmflow\ph_holdings`
  (~215K LOC, Wails v2 + Svelte, branch `ui-ux-hardening` @ `ca24372`).
- **Substrate (this repo):** `main` @ `801f41a`.

**Enumerate the surface, by domain, in parallel read passes** (the same
six-pass discipline that produced the Mission A ledger). For each domain,
list every deployed element and classify per §2:

- **Bound-method surface:** every Wails-bound method PH exposes (the App
  god-object + service files) → its OSS counterpart. PH's App is ~1229
  methods; OSS's composition root is decomposed but the App is still thick
  — expect a many-to-many map, not 1:1.
- **Frontend screens/routes:** every Svelte screen/route/modal in PH's
  `frontend/` → OSS `frontend/`. Flows, not just files (e.g. "edit-RFQ
  line-item entry," "costing Start-Fresh confirm," "linked-invoices on
  order detail").
- **Reports & documents:** every PDF/CSV/report generator (invoice PDF, PO
  PDF, offer PDF, VAT-return CSV, statements, costing exports).
- **Startup & scheduled jobs:** every boot-time backfill/migration/hook and
  any timer/watcher.
- **Fresh-provision schema parity:** does a from-zero OSS DB `AutoMigrate`
  **every table** the deployed PH schema has? The Mission E rehearsal
  already found two provisioning gaps — the banking-reconciliation suite
  (`bank_statements`, `bank_statement_lines`, `bank_accounts`, …) and
  `extracted_documents` are **not created on a fresh file**. Treat missing
  *tables* as a parity gap to CLOSE (provision them in the model-set);
  treat the *data* carry/skip as deferred (Mission H) — note it, don't
  migrate it.

Domains to cover (suggested split): finance/invoicing · procurement
(PO/GRN/supplier-invoice/payment) · CRM (customers/suppliers/contacts) ·
opportunities/costing · inventory/stock-movement · documents/OCR/inbox ·
reporting & PDF · settings/RBAC/users · sync · startup/migration.

**Output:** the matrix (domain → element → classification → evidence →
action). This is the honest deliverable of the wave even if nothing else
ships — a corrected, complete map (parent §4, W4-D4 discipline).

### G.2 — Close the known tail (the ledger residue + housekeeping)

Independent of the audit, these are already-known and can proceed
golden-first / pkg-tested. **Verify-then-skip anything G.1 proves present:**
- **D1/D2 residue:** port `cleanLooseOneDriveFolderNumberToken` (the one
  PARTIAL row) + PH's `opportunity_collapse_regression_test.go` cases.
- **B3-UI:** the linked-invoices Svelte block on order-detail (binding
  `GetInvoicesByOrder` already exists in OSS; port the `a1aae97` UI only).
- **D3:** the costing↔opportunity hardening flows
  (`findOpportunityFromPendingLaunch` ordered-passes matcher,
  `disconnectFromOpportunity`, Start-Fresh confirm modal) — frontend-only.
- **Band-3 robustness (opportunistic, security-relevant):** 3-PARSE ×3
  (the `int64(UncompressedSize64)` zip guard, the byte-dropping
  `ParseInt(hex,16,8)`, unclamped float→uint8 in predator vision) + their
  security tests; 3-CONN DSN timeouts on the online-sync path. Mirror the
  PH diffs.
- **1-FX residual:** confirm the backend populates line-item `currency`
  JSON; if blank, it's the same defect class via a different field — close
  it.
- **Synthetic bank-slug rename** (§3.5).

### G.3 — Close the ABSENT / DIVERGENT-GAP findings from G.1

Whatever the audit surfaces as genuine holes. **Money/integrity first,
golden-first, pkg-level tests with fake ports** (parent §4 Mission B
discipline). A financial-number divergence between OSS and PH is a **live
bug in one of them** — surface it, do not silently pick a side (parent §6,
W4-D2). Any NEW behavior fork beyond the two pre-ratified in §3 →
stop-and-ask.

### G.4 — The parity certificate

`docs/PH_PARITY_CERTIFICATE.md`: the Commander's cutover-readiness
artifact. Contains:
- Every surface element resolved: PRESENT / PORTED+tested /
  DIVERGENT-INTENTIONAL-registered / stop-and-ask-pending.
- **The DIVERGENT-INTENTIONAL register** — explicit, each with reason +
  trade-off (starts with: party-delete orphaning, bank-recon auto-revert,
  Turso-not-Postgres sync; plus every EXTRA where OSS does more).
- Honest coverage % (behaviors/flows verified, not method counts).
- The residual stop-and-ask list (esp. the invoice-PDF appearance items).
- A one-paragraph verdict: is the substrate ready to carry PH's flows?

### G.5 — The mirror

- `docs/PH_CONVERGENCE_DECISIONS.md` — append PC-D10… with `[Mirror]`
  paragraphs for what generalizes, written WHEN you decide.
- `docs/PH_CONVERGENCE_PROGRESS.md` — a Wave 4 section: measured timeline,
  honest classification counts, honest thesis %, residue for Mission H
  (data convergence).

---

## 5. Invariants (inherit parent §5; these are added / reaffirmed)

- **PH is live and frozen.** Nothing here touches deployed `ph_holdings`;
  read-only on that side. The cutover is Commander-gated.
- **Real client data never enters this repo.** The audit is code-only.
  Synthetic canon only (invariant #2).
- **Financial semantics are sacred, doubly so here** — you are certifying
  an *audited* system. Rounding, posting order, tax behavior, sequence
  formats, and customer-facing document appearance: golden-first,
  stop-and-ask.
- **Parity = coverage, not bug-for-bug identity** (§2). Intentional
  divergences are **recorded, never hidden, never silently "fixed."**
- **Port through the kernel, not around it** — where a gap needs new
  gating, use `pkg/approvals` + kernel actors, not a bespoke god-object
  method.
- **Keep it green** at every checkpoint: `go test ./...` (main pkg is
  slow — budget ~600 s), `go build ./...` (needs `frontend/dist` — run a
  frontend build or `wails build`; `go vet ./...` catches test-compile
  breaks), svelte-check baseline 0 errors.

## 6. Stop-and-ask registry (ask the Commander; do not decide these)

- Any change to the bytes/appearance of a customer-facing financial
  document (invoice / PO / offer / credit-note PDF, VAT-return CSV) — the
  invoice-PDF cosmetic items live here.
- Any NEW behavior fork the audit finds beyond the two pre-ratified in §3.
- Any financial-number divergence where OSS and PH disagree on a value —
  surface it as a live bug; do not pick a side.
- The cutover itself.

## 7. Definition of done

- `docs/PH_PARITY_MAP.md` complete: every deployed-PH surface element
  classified with evidence.
- Every DIVERGENT-GAP / ABSENT closed with golden + pkg-level tests, or
  recorded with Commander sign-off; the known tail (G.2) closed or
  verified-present.
- `go test ./...` green · `go build ./...` clean · svelte-check 0 ·
  (a `wails build -clean` is HARD-DENIED to the agent — the Commander runs
  it at packaging; hand-sync only the specific bindings that change).
- `docs/PH_PARITY_CERTIFICATE.md` written with an honest coverage % and an
  explicit DIVERGENT-INTENTIONAL register.
- `docs/PH_CONVERGENCE_PROGRESS.md` Wave 4 section written; Mission H
  (data convergence) residue handed forward.
- The Commander has, in one document, the evidence to schedule the
  cutover.

---

## 8. Operating notes (gotchas that cost prior sessions time)

- **CWD trap:** the Go module root is the repo root, but the harness may
  start in `frontend/`. Always pass an explicit `path:
  C:\Projects\asymmflow\asymmflow-oss` for backend searches; `Push-Location`
  to the root before `go test` / `go build`.
- **The App god-object is real:** PH's ~1229-method App means a
  bound-method comparison is many-to-many. Map by *flow*, not by method
  name.
- **Two "Wave 3"s exist — don't conflate:** `FABLE_WAVE3_PROGRESS.md` is
  the composition-law sprint; the convergence Wave 3 is
  `docs/FABLE_WAVE3_PROGRESS.md` / branch `feat/fable-ph-convergence-w3`
  (now merged to `main`). This wave is convergence Wave 4.
- **Branch:** cut `feat/fable-ph-convergence-w4` from `main` @ `801f41a`.
  Commit in small coherent steps. Never push (hard-denied); local commits
  only. Merge to `main` is Commander-gated at wave close.
- **Measure, don't estimate. The ground wins; the mirror records why.** 🌊
