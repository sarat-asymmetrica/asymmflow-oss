# Campaign — INTEG Execution: Handoff / Residue Ledger

**Status at handoff:** Waves **I1 + I2 complete**, Customer-360 reshaped (owner ruling),
**I3 financial hot-zones mostly wired + tested**. All work on `exp/frontend-kernel`
(worktree `asymmflow-lab`, LOCAL-ONLY — never pushed). Every checkpoint gated green
(`npm run check` 0/0 (348), vitest, `npm run build`, layout detector 49/49, `go test`
hot-zones). The K6 flip is **NOT** started (owner-gated Task #5, out of scope).

Read the running record in `FABLE_KERNEL_CAMPAIGN_LOG.md` (INTEG section) and the
sign-off scoreboard in `FABLE_WAVE_K6_PARITY.md`. This file is the **residue ledger**:
what's left, and the gotchas learned.

---

## Validation doctrine (owner-ratified 2026-07-15)

The WebView2 GUI cannot be driven headlessly (Playwright hits the vite dev server = mock
mode, no `window.go`). So each hot-zone is validated by: **(a)** `npm run check` proving the
adapter↔binding contract against the generated `wailsjs/*.d.ts`, **(b)** a Go persistence/
audit/reversal test driving the real `App` binding against a **scratch SQLite** (the spec's
"Go query snippet against the scratch DB"). The owner's smoke checklist remains the human GUI
pass. `time.Time` IPC marshalling proven in I1.3 (`integ_date_bridge_test.go`).

**The repo already has hundreds of Go tests** — where a binding is covered, wire-and-verify;
else wire-and-write-test. New tests added this campaign: `integ_date_bridge_test.go`,
`integ_ar_hotzone_test.go` (ApplyCreditNote), `integ_accounting_hotzone_test.go`
(CreateJournalEntry), `integ_ap_hotzone_test.go` (DeleteSupplierPayment),
`integ_recon_hotzone_test.go` (Finalize/DeleteBankStatement).

## Operating model that worked

3 parallel Sonnet agents wired the mechanical adapters per domain (disjoint bridge files, no
`registry.ts` touch, no dev server, `npm run check` self-gate). **Critical rule that paid off:
"gap-if-uncertain" on sacred posting paths** — agents left anything they couldn't map with
certainty as an honest `INTEG gap:` rather than guess. The orchestrator wrote/ran the Go tests,
gated centrally, flipped the scoreboard, and committed.

---

## RESIDUE — remaining work (in rough priority order)

### A. 🔥 Hot-zone TODOs the orchestrator owns (agents correctly declined)
1. **`SaveCostingAsOffer`** (costing-sheet.ts) — creates/overwrites an Offer. The binding takes
   a *flat* `main.CostingExportData` struct; the frontend `CostingExportPayload` is a *nested*
   lab subset and no assembly exists. Build the CostingExportData from the costing VM state
   (header + line-items → flat fields + `lineItems[]`), then wire + write a Go test asserting the
   Offer is created with the right totals. Do NOT guess the payload — verify every field vs
   `models.ts` `CostingExportData`/`CostingExportLineItem` and the Go `SaveCostingAsOffer` impl.
2. **Supplier-invoice DESCRIPTOR consumption** — the bridge fns `approveSupplierInvoice` /
   `markSupplierInvoicePaid` / `performThreeWayMatch` are wired (SoD approver from session) but the
   `supplier-invoices.descriptor.ts` is still read-only. Add the per-status actions that call them.
3. **`Expenses` Post** (expenses.ts `realPost`) — GAPPED: it posts a real GL journal entry
   (`postExpenseJournal`), not a status flip. Owner decision: does GL-posting belong on this
   ledger screen? If yes, wire `PostExpenseEntry` + Go test (assert the journal entry is created).
4. **Bank Accounts Create/Update** (bank-accounts.ts) — GAPPED: `CreateBankAccount(struct)` /
   `UpdateBankAccount(id, patch)` carry **encrypted IBAN/SWIFT**; a plaintext patch would bypass
   server-side FieldCrypto re-encryption. Needs an encryption-safe adapter contract (confirm which
   fields the backend encrypts, pass them so the server encrypts — never pre-encrypt client-side).

### B. Deferred Go persistence tests (bindings WIRED + type-verified, test pending)
- `FinalizeBookBankReconciliation` (adjacent to the tested FinalizeReconciliation).
- `DeleteRFQWithCascade` (assert linked costing/offers cascade-removed on confirm=true; errors on
  confirm=false when links exist).
- `ReviewDeleteApprovalRequest` / `ReviewEmployeeArchiveRequest` — existing coverage in
  `app_test.go` / `employee_archive_service_test.go`; add a focused approve+reject round-trip if
  desired.
- Import two-phase (`Preview/Confirm/Discard`) — assert nothing persists until Confirm.

### C. Non-financial / operational mutations NOT in the §3 hot-zone roster (still gapped)
These were out of the explicit I3 §3 list; wire with the same pattern (agents, gap-if-uncertain),
type-gate + existing tests where present:
- **People** (people.ts) — 13 PII/credential mutations (Create/UpdateEmployeeProfile,
  RequestEmployeeArchive, GenerateLicenseKey, Create/DeleteEmployeeDocument). PII — careful.
- **Work** (work.ts) — 14 task/project mutations (Delete/Archive/ShelveCollaborativeProject,
  Create/UpdateCollaborativeTask…).
- **Deployment** (deployment.ts) — UpdatePilotDeploymentChecklistItem, TriggerCollaborativeSyncNow,
  RetryCollaborativePendingOperations, export bundle/signoff.
- **Butler** (butler.ts) — the `executeButlerAction` seam over ~23 write actions +
  ChatWithButlerPersistent / DeleteConversation / PurgeAllConversations. AI-authority boundary:
  agents may only draft; deterministic services approve/post — preserve.
- **Business Settings** (business-settings.ts) — `UpdateSettings` (confirm key vocabulary vs the Go
  handler first).
- Costing `UpdateCostingSheet` (struct-arg), Accounting `UpdateAccount` (untyped patch),
  `CreateSupplierInvoice` (struct-arg), 3 bank-recon statement/line edits (untyped patch) — all
  gapped on untyped/rich-struct args; need a confirmed key contract before wiring.

### D. AI-provider key encrypted settings (owner-ratified: encrypted in-app)
Infra EXISTS: `SettingsService.SetSetting(key, value, encrypt=true)` uses FieldCrypto (HKDF +
AES-256-GCM). Wire the Settings/Butler AI-key field to a binding that calls `SetSetting(...,true)`
and loads it back masked. Never log/echo the key; no secrets in source. (Only affects Settings/Butler.)

### E. Reads still deliberately blank (honest, ledgered — NOT bugs)
- Notifications review cards: `reviewStatus`/`requestedBy`/`reason` blank (live on the request, not
  the notification) — enrich when the review mutations' UI lands.
- main-dashboard focus/alerts/tasks, finance-overview notices, audit-trail amount — no backing
  binding; honest blank (Hub hides empty widgets).

---

## Gotchas learned (save the next instance time)

1. **GORM dest reuse** — `db.Where(...).First(&x)` where `x` still holds a prior row's primary key
   ANDs the stale PK into the query (`id='a' AND id='b'` → no rows → "record not found"). Always
   read into a FRESH struct (see the `getInvoice` helper in `integ_ar_hotzone_test.go`).
2. **memdb** — `setupTestApp` uses ncruces shared in-memory SQLite; migrate the extra tables your
   method touches (e.g. DeleteSupplierPayment re-derives the linked `supplier_invoices` status, so
   that table must exist even for an "unlinked" payment).
3. **time.Time args** — the generated `time.Time` TS class is an empty codegen stub; pass the
   RFC3339 wire string via `map.goTime(dateStr)` (UTC midnight, explicit `Z`) + a structural cast.
4. **Actor args** — bindings that take a `user`/`approvedBy`/`performedBy` string: source it from
   `actingUserId()` (session store), never trust it from the row/caller.
5. **Delete-approval guard** — `guardDeleteOrRequest` deletes directly for an admin session but
   raises a delete-request for non-admins; the test-admin (`RoleName: "admin"`, perms `["*"]`) is
   admin, so deletes go through.
6. **wails.json** — this branch points at `frontend-lab` for dev; do NOT `git merge main` casually
   (it reverts the repoint). `main.go`'s `go:embed` still points at old `frontend/` (that repoint is
   flip step 2). So `wails dev` exercises frontend-lab; `wails build` would embed the old app.
7. **Dev server + gate** — start `npm run dev -- --port 5175 --strictPort` then
   `BASE_URL=http://localhost:5175 node tests/gate.mjs "<Labels>"`. Kill stray 5175 listeners first.
8. **sqlite3 CLI** now installed (winget SQLite.SQLite 3.53.3) for ad-hoc scratch-DB inspection.

## The prize (unchanged)

Every parity row honestly `real`/`wired`, every hot-zone mutation proven against a throwaway DB
with its audit trail intact — so the only thing between the kernel and production is the owner's
smoke checklist and the K6 flip.
