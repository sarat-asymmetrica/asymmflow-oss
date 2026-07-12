# FABLE CAMPAIGN — PH CONVERGENCE: "Prove the Thesis at PH Scale"

Written 2026-07-04 by the Opus 4.8 instance running the Commander's
(Sarat's) strategy session, for the Fable 5 instance that will run this
campaign **after Wave 6 completes** (or wherever the Commander slots it).
This is a CAMPAIGN spec, not a single wave — converging a live, money-
moving, 98-table ERP onto the substrate is several waves of work. Run it
with the same discipline as Waves 2–6: small coherent commits, golden-
first for anything financial, measure don't estimate, and when the map
and the ground disagree, the ground wins and the mirror records why. 🌊

The Commander is available. Ask when a decision is his; do not ask when
this document already answers it.

---

## 0. What this campaign IS

`asymmflow-oss` was **seeded from** the live client app `ph_holdings`
("PH Trading" — an industrial/instrumentation trading ERP deployed in
Bahrain, ~8 users, real money, real audited financials). The two repos
share **no git ancestry**: OSS is a re-architected substrate (pure
kernel → engines → overlay → vertical), seeded by copying business logic
out of a **post-Phase-33, pre-April-13-2026** `ph_holdings` working tree.

Since that seed the two diverged in **opposite** directions:
- `ph_holdings` gained ~3 months of field-hardening the fork never saw
  (the ledger in §3).
- `asymmflow-oss` gained the entire kernel/overlay refactor, the
  hospitality vertical, Saudi ZATCA, pure-Go SQLite, session timeout,
  payroll refusal — none of which `ph_holdings` has.

**This campaign closes the first gap so PH Trading can run on the
substrate as Sovereign Fork #1.** The `OVERLAY_BOUNDARY_GUIDE` already
names this destiny: *"The current repo began as a serious industrial/
trading system… it should become the first real overlay proving the
kernel can support demanding operational workflows."*

## 1. The thesis this campaign proves

> **"A vertical is configuration plus a thin domain package."**

Hospitality proves it at *toy* scale. PH Trading proves it at *real*
scale. **The convergence and the thesis-validation are the same event:**
if a demanding, audited, 98-table industrial deployment can be reduced to
an `overlay.json` + a sovereign-fork config on the shared substrate, with
its invoices and financials **byte-identical** to the live system, the
thesis stops being a claim and becomes a demonstrated fact. That is the
terminal state of this campaign. Nothing less counts as done.

## 2. The Freeze Law (Commander-decided 2026-07-04 — context, not optional)

The Commander has ordered a **HARD CUT**: from 2026-07-04, all net-new
PH Trading feature/fix work lands in **OSS-trading only**, never in
`ph_holdings`. The *only* exception class is **production-down /
security-critical / data-integrity**; anything in that class may still
land in `ph_holdings`, but the SAME commit must be logged into the
Divergence Ledger (§3) the instant it is made, so even exceptions cannot
silently widen the gap. Everything not in that class → OSS-trading or it
does not happen.

**Implication for you:** the ledger below is a *snapshot as of
2026-07-04*. Before executing, `git -C <ph_holdings> log` since that date
and fold any exception-class commits into the ledger first.

## 3. The Divergence Ledger (the heart of the campaign)

Authoritative PH-side catalogue, commit-cited, from a full audit of
`ph_holdings` git history + CLAUDE.md phases. Short SHAs are in the
`ph_holdings` repo. **OSS-side verdicts marked `[seed]` are a first-pass
from 2026-07-04 and MUST be re-measured on the ground (Mission A) before
any port — the delete-guard example below shows why a blind port can
REGRESS the substrate.**

### Band 0 — The June-29 "Abhie" SPOC fixes (TOP PRIORITY, freshest, most business-specific)

| # | PH behavior | PH commit | OSS verdict `[seed]` | Tag |
|---|---|---|---|---|
| A1 | `Order`/`ExpenseEntry` custom `UnmarshalJSON` accepting date-only *and* RFC3339 (empty stays zero so GORM skips) — fixes every date-input edit save being rejected | `425c0b2` | ABSENT (OSS `parseFlexibleDate` is bank-statement only; a helper exists to reuse) | **PORT** |
| B1 | Won-import (OneDrive) populates invoice line-items + `normalizeDivisionName` division — the write-side origin of hollow invoices | `3c5127b` | Likely PRESENT — verify `BackfillInvoiceItemsFromOrders` + import path covers the division stamp | **VERIFY** |
| B2 | Env-gated hollow-invoice diagnose/repair/verify/checkpoint toolkit; healed 10 hollow invoices | `e5c0665`,`9644520` | PRESENT (startup backfill app.go:992, `deployment_audit.go` hollow-detection) | **DISSOLVED** |
| B3 | "Linked Invoices" table on order-detail (calls the zero-caller `GetInvoicesByOrder`) | `a1aae97` | VERIFY (binding may exist; UI surface likely absent — frontend port) | **PORT (UI)** |
| B4 | Draft-invoice line-item editor; **totals derived server-side** in `UpdateCustomerInvoice`; Amount read-only for Draft | `0a96926` | PARTIAL — server recompute technique present in create path; verify update path + build the editor UI | **PORT** |
| B4a | Zero-rated VAT preserved on Draft edit (recover effective rate from `VATBHD/Subtotal` before defaulting to 10%) | `70b05d2` | PARTIAL — recovery trick exists at create (customer_invoice_service.go:817); **verify the UPDATE path** | **VERIFY→PORT** |
| C | Invoice PDF: per-`Division` bank block, buyer-address fallback to `Attention*`, `"For <LegalName>"` signature, 40mm top margin, ref-number width-truncation | `b06b83f` | VERIFY (PDF service exists; division-bank filtering + fallbacks likely absent). **Customer-facing doc appearance = stop-and-ask** | **PORT (gated)** |
| D1/D2 | `folderNumberHasDigit` guard stops customer-word→folder-number collapse; `routeToOpportunity` matches on canonical key (stops OCR dup) — ~83 opportunities were being hidden | `10f96a7` | VERIFY (canonical-key logic was in the seed; the digit-guard was added after) | **VERIFY→PORT** |
| D3 | Costing→opportunity matcher as ordered passes (exact id → exact ref → same-customer last); Disconnect + Start-Fresh | `505ae76` | VERIFY (frontend costing flow) | **PORT (UI)** |
| E | `DeleteCustomer`/`DeleteSupplier` blocked when party has transactional children (+ linked opportunities, from skeptic rework); returns counts; supplier-delete UI added | `1778090`,`70b05d2` | **DIFFERENT** — OSS has `delete_approval_service.go` + `guardDeleteOrRequest` (approval-workflow route). **A blind port would REGRESS this.** Decide: empty-guard vs approval-workflow, or compose both | **DECISION** |
| F | Credit-limit override: soft-limit bypass gated to `isManagementRole` (finance *excluded*) + required reason + `CREDIT_LIMIT_OVERRIDE` audit row (AuditLog gained `ResourceID`+`Details`); chokepoint on `CreateInvoiceWithOptions` itself (not just the Orders helper) | `a4dfeaf`,`223af59`,`ca24372` | ABSENT — hard block exists, audited-override path does not. **Route through `pkg/approvals` + kernel actors, not a bespoke gate** | **PORT (via kernel)** |

### Band 1 — June deployment-readiness money/integrity invariants (second priority; touch money/data, never made the phase docs)

- **Field-mask partial-update protection** (`bf567b5`): `UpdateSupplierInvoice`/`UpdateGRN`/`UpdateCustomerContact` load-then-overlay so partial payloads don't wipe approval-trail / journal-link / 3-way-match / QC / OCR-linkage fields. → **VERIFY/PORT** (data-integrity class).
- **24 previously-unguarded frontend-bound mutators got `requirePermission`** (`bf567b5`) + internal/exported split hard-guarding the Seed* funcs to admin. → **VERIFY** against OSS RBAC coverage.
- **Hollow-invoice send-block** (`MON-007`): `SendCustomerInvoice` refuses a hollow invoice. → **VERIFY**.
- **HMAC backfill/verify** (`MON-003`, Pass-D): startup pass backfills blank invoice HMAC hashes (481 Tally invoices) + `VerifyInvoiceHash` constant-time tamper check. → **VERIFY**.
- **PO VAT computed on `SubtotalBHD`** not foreign subtotal (`MON-004`) — feeds the 5K BHD approval threshold. → **VERIFY** (financial-semantics; golden-first).
- **Non-BHD line-items mislabelled as BHD** fix (June-15 P2). → **VERIFY**.
- **Sync robustness** (`ffbe9c7`): widen remote `varchar→TEXT`, backfill NULL PKs with UUIDs before sync (SQLite tolerates, PG rejects), `SKIP_REMOTE_MIGRATION`. → relevant only if PH keeps Postgres sync; **DECISION** given OSS's Turso/CDC direction.
- **`promote` foreign-key bug** (`5545793`): `PRAGMA foreign_keys=OFF` inside a txn is a SQLite no-op — the data-promote had *never* worked. Note for the data-migration mission (§4 Mission C).

### Band 2 — The April write-policy seam (architectural DECISION, not a bugfix)

`customer_write_policy.go` / `supplier_write_policy.go` /
`bank_reconciliation_policy.go` (`35bb48c`→`3f87e3a`, 2026-04-13): a
shared seam for ID generation, seed enrichment, primary-contact policy,
product-supplier linkage, valuation fallback, bank-recon lifecycle.
**Decide whether the OSS overlay/engine layer already subsumes this
concept or whether the policy seam ports in as an engine.** This is a
"config not code" judgment call — exactly the kind the substrate exists
to make. Record the decision in the mirror.

### Band 3 — Robustness / platform / UI hardening (port opportunistically, not blocking)

Parser-robustness (ZIP byte-accounting `64799be`, RTF hex-escape
`31fdb9e`, OCR byte-conversion `c47c28f`); **7 platform-blind bugs**
(`3bc658f`) incl. field-crypto salt falling back from read-only Program
Files to AppData, `%APPDATA%` vs `~/.local/share` — **cross-check against
OSS `pkg/runtime/composition.StandardOverlayDirs`, which likely already
solves most of these**; June-20/21 UI canonicalization (one `KPICard`,
one status-colour SSOT, one `Modal`) — OSS has its own `packages/` design
system, so this is likely DISSOLVED. Connection hardening (`PingContext`
5s timeout, conn lifetimes, `statement_timeout=30s`) → **VERIFY** in
`pkg/infra/db`.

## 4. The Missions (risk-retirement order — cheapest falsification first)

**Mission A — Re-measure the ledger (produce verified verdicts).** For
every `[seed]`/VERIFY/DECISION row above, measure OSS-trading on the
ground and replace the seed verdict with a real one: DISSOLVED (substrate
already covers it), PORT (clean port needed), DECISION (OSS chose a
different path — Commander or documented judgment). Output: a corrected
ledger in `docs/PH_CONVERGENCE_LEDGER.md`. This is the honest deliverable
of the first wave — a corrected map, per W4-D4. Do NOT port anything
until its row is re-measured; the delete-guard (E) is the standing proof
that a blind port regresses the substrate.

**Mission B — Port the money/integrity-critical divergence, golden-first.**
In priority order: F (credit-override via `pkg/approvals` + kernel
actors — NOT a bespoke gate), B4/B4a (server-derived totals + zero-rated
preservation on update), Band-1 field-mask partial-update protection, PO-
VAT-on-BHD. Golden the numbers BEFORE touching producing code (W5-D2);
every ported behavior gets a pkg-level test with fake ports. Customer-
facing PDF changes (C) are **stop-and-ask** (financial-doc appearance).

**Mission C — The data-migration path.** A one-time importer: PH's real
98-table SQLite → the OSS schema (which has peeled/renamed tables — real
schema-drift reconciliation). Two hard rules: (1) **real PH data NEVER
enters this repo** — the importer is code here, the data lives only in
PH's sovereign deployment (SYNTHETIC_IDENTITY invariant). (2) The
Commander's separate historical-data backfill workstream
(`D:\ph_data_master`, the `darch` tool at `C:\Projects\darch`) should
target the OSS schema DIRECTLY, so cleaned data lands already in the
destination format — converge data-quality and engine in one pass. Mind
the `5545793` promote bug: `PRAGMA foreign_keys=OFF` inside a txn is a
no-op.

**Mission D — PH's overlay + sovereign fork.** Extract PH Trading's
reality — TRN, per-division bank details, BAPCO/Alba/EWA conventions, 10%
VAT, 8% min margin, `PH-{ROLE}-{6}` license format, AHS-vs-PH division
branding — into `overlay.json` + a sovereign-fork config. This is the
"PH Trading is now a JSON file, not a program" moment made literal. Every
company-specific fact that ends up in code instead of config is a thesis
failure — flag it.

**Mission E — Parallel-run reconciliation → cutover proof.** Run the OSS
binary on a COPY of PH's data beside the live deployment. Reconcile:
invoices byte-identical (to the fils)? P&L ties out? VAT return matches?
Only when outputs agree is the thesis proven. The pure-Go ncruces SQLite
removes the mingw-w64 cross-compile fragility — the Windows `.exe` just
builds.

**Mission F — The mirror (continuous).** `docs/PH_CONVERGENCE_DECISIONS.md`
(PC-D1…, `[Mirror]` paragraphs for what generalizes) written WHEN you
decide; `docs/PH_CONVERGENCE_PROGRESS.md` per wave (measured timeline,
honest verdict counts, honest thesis %, residue for the next wave).
Honest accounting: report behaviors verified/ported and LOC moved, not
method counts.

## 5. Invariants (inherit the standing 9 from `FABLE_WAVE6_HANDOFF.md` §5; these are added)

- **PH is live.** Nothing you do here touches the deployed `ph_holdings`.
  This repo is where PH's *future* is built; the cutover is Mission E and
  is Commander-gated.
- **Real client data never enters this repo.** Synthetic canon only.
  The migration importer is code here; PH's data is not.
- **Financial semantics are sacred, doubly so here** — you are
  reproducing an *audited* system. Rounding, posting order, tax behavior,
  sequence formats, and **the appearance of customer-facing financial
  documents**: golden-first, stop-and-ask.
- **Port through the kernel, not around it.** Where PH used a bespoke
  gate (credit override), the substrate's answer is `pkg/approvals` +
  kernel actors so the AI-authority boundary and audit trail come for
  free. Don't reintroduce god-object patterns.

## 6. Stop-and-ask registry (campaign-specific — ask the Commander first)

- Any change to the bytes/appearance of a customer-facing financial
  document (invoice/credit-note/offer PDF).
- The Band-2 write-policy-seam decision (architectural).
- The Band-0-E delete-semantics decision (empty-guard vs approval-workflow).
- The cutover itself (Mission E) and any parallel-run against real data.
- Whether PH retains Postgres sync or moves to the OSS Turso/CDC path.
- Any financial-semantics divergence discovered where OSS and PH disagree
  on a NUMBER — that is a live bug in one of them; surface it, don't
  silently pick a side (W4-D2: a straggler is a live bug).

## 7. Definition of done (the thesis, proven)

The campaign ends when:
- `docs/PH_CONVERGENCE_LEDGER.md` shows every divergence row resolved
  (DISSOLVED / PORTED+tested / DECISION-recorded).
- Every money/integrity port has golden + pkg-level tests; `go test ./...`
  green; `go build ./...` + `wails build -clean` clean; svelte-check 0.
- A PH `overlay.json` + sovereign-fork config exists; no PH-specific fact
  lives in code.
- Mission E parallel-run shows invoices byte-identical and financials
  tying out against a copy of the live data.
- `docs/PH_CONVERGENCE_PROGRESS.md` written with an honest thesis %.
- The Commander has what he needs to schedule the cutover.

When PH Trading boots on this substrate as Sovereign Fork #1 with its
numbers unchanged, the thesis is no longer a claim. That is the whole
point of everything from Wave 1 forward.

Build → Test → Ship. Measure, don't estimate. The ground wins. 🌊
