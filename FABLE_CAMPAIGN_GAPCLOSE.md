# Campaign — Gap-Close Pass (the last 23 → 0)

**You are the incoming orchestrator + technical lead.** INTEG (I1–I3) and the Residue pass
(R1–R5) are DONE and merged to main (`c51bf14`): ~160 INTEG gaps → **23 honest throws**, every
one named and reasoned. This campaign closes those 23. When it lands, the ONLY things left
before the K6 flip are the owner's smoke checklist and the parked R6 bundle-split decision
(both **out of scope** here).

Same operating model: **Sonnet 5 agents code, you gate every wave and fix what they miss.**
The two disciplines remain LAW: **gap-if-uncertain** (an honest throw beats a guessed wire) and
**wire-and-verify** (type-gate vs `wailsjs/*.d.ts` + a Go persistence test on scratch SQLite
for anything that mutates; for exports, assert the artifact itself — see G4).

**Read FIRST, in order:** `CLAUDE.md` → `frontend-lab/KERNEL.md` →
`FABLE_CAMPAIGN_INTEG_HANDOFF.md` (gotchas §: GORM dest-reuse, memdb migrations, `map.goTime`,
wails.json) → `FABLE_WAVE_K6_PARITY.md` (scoreboard) → `FABLE_KERNEL_CAMPAIGN_LOG.md` (INTEG +
Residue sections) → this file.

**Branch:** `exp/frontend-kernel` (worktree `asymmflow-lab`). Never push; merges to main go
through the owner's review gate. ⚠️ Do NOT `git merge main` casually. Validation runtime:
**quarantined scratch SQLite** (`PH_DB_PATH` + `APPDATA` to scratch, bash `export` not `$env:`).
⚠️ svelte-check runs INSIDE `frontend-lab/` (repo root sweeps unrelated `packages/showcase`).

---

## ★ OWNER RULINGS (2026-07-16) — ratified, not defaults. Do not re-litigate.

1. **Butler write-actions: SPLIT.** Wire the **19 draft/update-class** bindings (human arms +
   confirms in the UI; the confirming HUMAN is the acting actor — attribute via the session
   actor, never "butler"). The **4 approve-class** bindings — `ApprovePurchaseOrder`,
   `ApproveStockAdjustment`, `ApproveSupplierInvoice`, `ApproveCostingSheet` — are **permanently
   retired from butler's action vocabulary**: remove them from `butler-actions.ts` resolution;
   butler REPLIES pointing at the Approvals Queue instead. Rationale: the AI-authority boundary
   means the agent never puts an approval one click away. This mirrors the mesh's distributed
   law (see `mesh/docs/MESH_DECISIONS.md` MESH-D10).
2. **Invoice settlement: receipt-capture modal.** Replace the mark-paid stub with a small
   receipt form (amount, date, method, reference) on the Invoices screen calling
   `ApplyCustomerReceiptToInvoice` — ride the R5 `ActionSpec.modal` seam. Honest accounting,
   no status flips.
3. **Standalone invoice create: RETIRED.** Remove the create affordance from the Invoices
   screen; invoices are raised from an order (`CreateInvoiceWithOptions`, already wired there).
   EmptyState + toolbar copy point at Orders. Record in the retire ledger.
4. **Pricing win-rate list: real read-only Go aggregation endpoint.** Add a binding that
   computes per-customer win-rate from actual offer won/lost history (the old screen HARDCODED
   this list — that is the bug, not the reference). Read-only; unit-test the aggregation.

Standing defaults (owner may veto at kickoff, otherwise proceed):
- **`DeleteRFQWithCascade` stays RFQ-only** — hide the cascade-delete action for
  pipeline-sourced Opportunity rows (matches the Go binding's actual domain); no new binding.
- **`UpdateCostingSheet`** — assemble the full `CostingSheetData` struct in the costing VM,
  the same technique R1 proved for `SaveCostingAsOffer`; no narrower Go binding.

---

## Wave G1 — 🔥 Owner-ruled product changes (do these YOURSELF, not agents)

The four rulings above, in order: butler split (1) · settlement modal (2) · invoice-create
retirement (3) · win-rate endpoint (4). Each mutation gets its Go persistence test; the butler
split additionally gets a vitest asserting the 4 approve-class names can NEVER resolve to an
executable action (a boundary tripwire, like the mesh gate's agent-rejection check).

## Wave G2 — Payroll hot-zone (financial + PII)

`UpsertEmployeeCompensationProfile` — R1-grade treatment: confirm every field against the Go
struct, wire, Go test asserting persisted compensation rows; synthetic canon only, no real
names/salaries in fixtures ever. Plus the payroll **employee master list** fetch (cross-domain:
`collaboration.listEmployeeProfiles` — read-only, verify the service seam).

## Wave G3 — Known-technique wiring (agent-parallelizable, disjoint files)

- **`UpdateSettings`** — fetch-merge-write: `GetSettings` → overlay the screen's 5 fields →
  save the FULL object. Go test: unrelated top-level keys (folders, apiKeys) survive the write.
- **Notifications review ×2** — enrich the fetch mapper with `source_id`, then wire
  approve/reject through `ReviewDeleteApprovalRequest` / `ReviewEmployeeArchiveRequest`
  (persistence tests already exist from R2 — reuse them as the oracle).
- **`SyncCashflowEvidenceProposalReviews`** — review-row sync, not a GL posting; wire + test.
- **Customer status change** — fetch-merge-`UpdateCustomer` (full record). Test: status changes,
  every other field survives.

## Wave G4 — The export tail (10 sites; side-effecting, so verify the ARTIFACT)

Accounting: `ExportBalanceSheetCSV` · `ExportGeneralLedgerCSV` · `ExportJournalCSV` ·
`ExportVATReturnData` · `ExportCashflowEvidencePack`. Costing: `ExportCostingToPDF` ·
`ExportCostingToExcel` · `OpenExportedFile`. Deployment: `ExportPilotSupportBundle` ·
`ExportPilotSignoffReport`.

Law for this wave: every export Go test runs against scratch, asserts the returned path exists
under the QUARANTINED dirs, and spot-checks content (CSV header row; PDF magic bytes; VAT
export against the division-scoped emission contract). `OpenExportedFile` is the one true
side-effect (shells out to the OS): wire it behind the existing confirm affordance and do NOT
invoke it in tests — assert the path argument only.

## Wave G5 — Close-out

Gap tripwire pin → **0**. Scoreboard rows flipped in `FABLE_WAVE_K6_PARITY.md` (incl. the two
retirements from G1 in the retire ledger). Dead-mock sweep for newly fully-wired screens (keep
the bridge seam; mock mode stays a feature). Campaign-log entry. Full gates.

**Explicitly OUT of scope:** the R6 bundle split (owner-parked) · the K6 flip itself ·
handoff §E deliberately-blank reads (LEDGERED non-bugs — do not invent bindings).

---

## Gates, per wave (unchanged law)

`npm run check` 0/0 (in `frontend-lab/`) · vitest all green · `npm run build` clean · layout
gate on touched screens (full 49-screen sweep at campaign end) · `go build ./...` +
`go test ./...` for any Go touch · campaign-log entry per wave.

## The prize

`INTEG gap:` count **0** — no survivors, no asterisks. Every mutation Go-proven, every export
artifact-proven, the AI-authority boundary mechanically tripwired in the frontend too. The
kernel app is flip-ready pending only the owner's human smoke pass and the parked bundle call.
