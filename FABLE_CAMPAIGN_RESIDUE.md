# Campaign — Residue & Tech-Debt Pass (INTEG close-out → flip-ready)

**You are the incoming orchestrator + technical lead.** INTEG waves I1–I3 are DONE and merged
to main (`b00d794`): ~160 INTEG gaps → 71 honest throws, financial core wired + Go-tested. This
campaign closes the residue and pays the ledgered tech debt, so the only thing left before the
owner's smoke checklist and the K6 flip (Task #5, still **out of scope**) is nothing at all.

Same operating model: **Sonnet 5 agents code, you gate every wave and fix what they miss.**
The two disciplines that made INTEG clean are LAW here too: **gap-if-uncertain** on posting
paths (an honest `INTEG gap:` beats a guessed wire), and **wire-and-verify** (type-gate vs
`wailsjs/*.d.ts` + a Go persistence test against scratch SQLite for anything that mutates).

**Read FIRST, in order:** `CLAUDE.md` → `frontend-lab/KERNEL.md` →
`FABLE_CAMPAIGN_INTEG_HANDOFF.md` (**the residue ledger §A–E is your work-list; the Gotchas
section will save you hours — especially GORM dest-reuse, memdb migrations, `map.goTime`, and
gotcha #6 on wails.json**) → `FABLE_WAVE_K6_PARITY.md` (scoreboard) → `FABLE_KERNEL_CAMPAIGN_LOG.md`
(INTEG section) → this file.

**Branch:** `exp/frontend-kernel` (worktree `asymmflow-lab`). Never push; merges to main go
through the owner's review gate. ⚠️ Do NOT `git merge main` casually (handoff gotcha #6 —
the wails.json repoint reverts). Validation runtime: **quarantined scratch SQLite** (owner
ruling 2026-07-15; `PH_DB_PATH` + `APPDATA` to scratch, `export` not `$env:`).

---

## Wave R1 — 🔥 The orchestrator-owned hot items (handoff §A; do these YOURSELF, not agents)

1. **`SaveCostingAsOffer`** — assemble the flat `main.CostingExportData` from the costing VM
   (header + line-items). Verify EVERY field against `models.ts` and the Go impl; wire; Go test
   asserting the created Offer's totals. The previous orchestrator correctly refused to guess —
   honor that bar.
2. **Supplier-invoice descriptor actions** — `approveSupplierInvoice` / `markSupplierInvoicePaid` /
   `performThreeWayMatch` are wired but unreachable; add the per-status actions to
   `supplier-invoices.descriptor.ts` (SoD approver from session, confirm dialogs on pay/approve).
3. **Expenses Post** — **default ruling (owner may veto at kickoff): YES, wire it on the ledger
   screen.** Posting belongs where users act, with a `ConfirmDialog` (requireReason) stating "this
   posts a GL journal entry". Wire `PostExpenseEntry` + Go test asserting the journal entry.
4. **Bank Accounts Create/Update** — **default ruling: plaintext-to-server contract.** Confirm
   exactly which fields the backend FieldCrypto handler encrypts (read the Go impl), send those
   fields plaintext over the binding so the SERVER encrypts; never pre-encrypt client-side; never
   echo IBAN/SWIFT back into logs or mock fixtures. Go test: create → read back → assert stored
   ciphertext ≠ plaintext and roundtrip decrypt works via the service.

## Wave R2 — Deferred Go persistence tests (handoff §B; bindings already wired)

`FinalizeBookBankReconciliation` · `DeleteRFQWithCascade` (cascade on confirm=true, error on
confirm=false with links) · a focused approve+reject round-trip for the two Review bindings ·
Import two-phase (nothing persists until Confirm). Follow the `integ_*_hotzone_test.go` house
style + the handoff gotchas.

## Wave R3 — The non-financial mutation tail (~40 gaps; handoff §C; agent-parallelizable)

Same disjoint-bridge-file batching as INTEG: **People** (13 PII/credential mutations — synthetic
canon only, PII care) · **Work** (14 task/project mutations) · **Deployment** (checklist/sync/
retry/export) · **Butler** (`executeButlerAction` seam — PRESERVE the AI-authority boundary:
agents draft, deterministic services approve/post) · **Business Settings** (`UpdateSettings` —
confirm key vocabulary vs the Go handler FIRST) · the untyped-patch stragglers (Costing
`UpdateCostingSheet`, Accounting `UpdateAccount`, `CreateSupplierInvoice`, 3 bank-recon patches)
— each needs its key contract confirmed against the Go side before wiring; gap-if-uncertain.

## Wave R4 — AI-provider key, encrypted at rest (handoff §D; owner-ratified)

Wire the Settings/Butler AI-key field through `SettingsService.SetSetting(key, value, encrypt=true)`
(HKDF + AES-256-GCM infra exists). Load back MASKED (last-4 only). Never log/echo the key; no
secrets in source or fixtures. Go test: set → stored encrypted → masked read.

## Wave R5 — Capture-form SLOT items (net-new UI on kernel primitives)

Bindings exist; the capture UI doesn't: **PO Receive-Items** · **GRN Receive/QC/Complete** ·
**Delivery Dispatch/Confirm** · **Invoice send/PDF/proforma**. Build on FormModal/Wizard/
LineItemsEditor — L1/L2 laws apply (tripwire tests will bite). If a kernel gap from
`FABLE_WAVE_K6_PARITY.md` §"Kernel gaps still open" blocks one of these (e.g. multi-panel,
`ColumnSpec.rowAction`), build the ENGINE feature once rather than ejecting per-screen — that
list is pre-approved engine work when a residue screen needs it.

## Wave R6 — Tech debt

1. **Bundle split** — `npm run build` warns: one 800 kB JS chunk. Evaluate route-level
   `import()` code-splitting of screens (registry is the natural seam). Wails serves from disk,
   so the win is parse/startup time, not network — measure before/after boot-to-dashboard; if
   the win is <100ms, document that and set `chunkSizeWarningLimit` with a comment instead.
   Don't split blindly.
2. **Warnings stay zero, mechanically** — svelte-check is 0 errors / 0 warnings today; add
   `--fail-on-warnings` to the `check` script so it can never drift.
3. **Gap-count tripwire** — add a tiny vitest that counts `INTEG gap:` throws in `src/bridge/`
   and asserts the number only ever DECREASES (pin current count; update the pin as waves land).
   Makes the scoreboard mechanical.
4. **Dead-mock sweep** — after R3/R5, remove mock branches for fully-wired screens where the
   mock no longer serves dev-mode value; keep the bridge seam itself (mock mode stays a feature).
5. **Handoff §E blanks stay honest** — the deliberately-blank reads (notification review cards,
   dashboard focus/alerts, audit amount) are LEDGERED non-bugs; do not invent bindings for them.

---

## Gates, per wave (unchanged law)

`npm run check` 0/0 · vitest all green · `npm run build` clean · layout gate (touched labels;
full 49-screen sweep at campaign end) · `go build ./...` + `go test ./...` for any Go touch ·
scoreboard rows flipped in `FABLE_WAVE_K6_PARITY.md` · a campaign-log entry per wave.

## The prize

`INTEG gap:` count 0 (or each survivor owner-accepted by name), every mutation Go-proven,
capture forms live, bundle debt measured-and-decided — the kernel app is flip-ready pending
only the owner's human smoke pass.
