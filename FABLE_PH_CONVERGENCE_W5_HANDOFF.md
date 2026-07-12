# FABLE CAMPAIGN — PH CONVERGENCE WAVE 5: "Carry Everything, Reconcile Everything" (Mission H)

Written 2026-07-07 by the Opus 4.8 instance running the Commander's
(Sarat's) session, for the Fable 5 instance that will run this wave
autonomously. Parent: `FABLE_CAMPAIGN_PH_CONVERGENCE.md`. Predecessor:
`FABLE_PH_CONVERGENCE_W4_HANDOFF.md` (Mission G — parity, now certified:
`docs/PH_PARITY_CERTIFICATE.md`, thesis ~96%, verdict "substrate is ready;
cutover is a data-and-timing exercise, not a parity risk").

Mission G proved the *engine*. Mission H fills the *tank*: a complete,
reconciled migration of PH's real data into the full OSS schema — every
table carried or explicitly skipped-with-reason, every row accounted,
reconciled to the fils. When this wave is green, the cutover is a
scheduling decision.

The Commander is available for the §6 stop-and-ask set. Do not re-ask
what §3 already decides.

---

## 0. What this wave IS — and is NOT

**IS:** Extend the Mission E / `cmd/phimport` path from "rehearsed on the
core tables" to "carries the ENTIRE PH surface into the now-complete OSS
schema, reconciled." Mission G provisioned 15 previously-missing
banking/FX/VAT tables; this wave **populates** them from PH source and
reconciles them, plus resolves every remaining carry/skip decision the
parity certificate deferred here.

**IS NOT — the data-QUALITY convergence.** Cleaning PH's data drift (the
darch `D:\ph_data_master` backfill, the 355-op correction patch, dedup,
economics-fill, broken-year adjudication) is **Mission I**, a separate
Commander-in-loop wave. Rationale (Commander's standing call): *the current
PH data is mostly workable, and data is a re-runnable dial — you can
re-migrate N times.* So Mission H carries the data **as it is today,
faithfully and completely**; Mission I improves it later. Do NOT pull
darch or the 355-op patch into this wave. Faithful first, better later.

**IS NOT** the mesh campaign (`FABLE_CAMPAIGN_SOVEREIGN_MESH.md`) or any
sync-transport work.

---

## 1. Where we stand (inherited ground)

- **Mission E rehearsal (PC-D9)** already proved the core path: fresh OSS
  DB provisioned, `cmd/phimport` carried 107 tables / 17,595 rows from a
  copy of the PH snapshot, and **25/25 reconciliation checks matched to the
  fils** (invoices 480 / 8,988,247.145 BHD, VAT, outstanding, payments,
  supplier invoices/payments, items, orders, offers, GRNs, users). Ran
  OUTSIDE the repo in `C:\Projects\asymmflow\mission_e_rehearsal\`.
- **Mission G** then found and closed the schema gaps that rehearsal
  *couldn't* have carried: 15 banking/FX/VAT tables now provision on a
  fresh DB (create-if-missing, FK-safe), and the supplier-invoice CHECK now
  accepts the values a clean 3-way match writes. So the destination schema
  is now COMPLETE where Mission E's was not.
- **The gap this wave closes:** Mission E carried the tables that *existed*
  in the destination at the time. Now that 15 more exist, plus several
  tables are still unresolved (carry/skip), the importer must be brought up
  to full-surface coverage and the reconciliation extended to match.

---

## 2. Invariants (inherit parent §5 + W4 §5; reaffirmed — these are sacred here)

- **Real PH data NEVER enters this repo.** The importer, transforms, and
  reconciler are CODE here; PH's data lives ONLY in the out-of-repo
  rehearsal dir and the sovereign deployment. Synthetic canon only in the
  repo (invariant #2). Run all data work in
  `C:\Projects\asymmflow\mission_e_rehearsal\` (or a fresh sibling), NEVER
  in the working tree.
- **Financial semantics are sacred, doubly so** — you are migrating an
  *audited* system. Every carried money value reconciles to the fils or the
  wave is not done. A number that doesn't tie is a stop-and-ask, not a
  round-off.
- **Every row is accounted.** Carried, skipped, or transformed — the
  importer's JSON report must account for 100% of source rows, and every
  skip carries a reason (the Mission E discipline: "customer_receipts is
  empty → verified honest no-op"). Silent drops are failures.
- **`PRAGMA foreign_keys=OFF` is a no-op inside a transaction** (PH bug
  `5545793`, carried note). The importer must set pragmas OUTSIDE any txn.
- **Keep it green:** `go test ./...`, `go vet ./...`, svelte-check 0, and a
  **full reconciliation pass** at every checkpoint. `wails build -clean` is
  Commander-run at packaging.

---

## 3. The Commander's pre-decided calls (do NOT re-ask)

1. **Scope:** faithful + complete migration of CURRENT data. Data-quality
   cleanup is Mission I, deferred. (§0.)
2. **Skip-with-reason tables** (certificate §"Deferred"): `intelligence_*`,
   `data_update_*`, `*_backup` → **SKIP**, each logged with a one-line
   reason in the report. Do not carry scaffolding/derived/backup tables.
3. **Reconciliation is the gate.** A wave that imports but doesn't reconcile
   the new tables is incomplete. Extend the harness, don't just extend the
   importer.

**Genuinely open (measure-and-propose, then §6 if judgment is needed):**
- `extracted_documents` (×359 in the snapshot) carry/skip — it is
  model-less/raw in PH too. Measure it (row shape, size, whether anything
  references it), then propose: carry via a raw/passthrough model, or
  skip-with-reason. Stop-and-ask only if the call is genuinely the
  Commander's.
- The OneDrive discovery-walk / disambiguation cluster (~18 fns) — matters
  only for a *live re-scan at cutover*, not for carrying existing rows.
  Assess whether cutover needs it; if not, record as Mission-I/at-cutover.

---

## 4. The Missions (risk-retirement order)

### H.1 — Full-surface import coverage (extend `cmd/phimport`)

Bring the importer from Mission-E coverage to **the complete PH table
surface** against the now-complete OSS schema. For EVERY PH source table,
the importer must do exactly one of: **carry** (map + copy, counted),
**transform** (documented mapping, e.g. PC-D7 receipts→payments), or
**skip** (reason logged). Deliverable additions:
- Populate the **15 newly-provisioned tables** from PH source:
  `bank_accounts`, `bank_statements`, `bank_statement_lines`,
  `bank_statement_files`, `statement_hashes`, `book_bank_reconciliations`,
  `deposits_in_transit`, `cheque_registers`, `outstanding_cheques`,
  `bank_reconciliation_audit_logs`, `bank_cash_balances`,
  `bank_expense_entries`, `fx_rates`, `fx_revaluations`, `vat_returns`.
  (The rehearsal noted PH itself has thin data here — ~2 statements /
  12 lines — so expect small counts, but carry them faithfully and let the
  report show the honest numbers.)
- Resolve `extracted_documents` per §3.
- Confirm the schema-drift reconciliation still holds for peeled/renamed
  tables (real divergence; two schemas, not a byte-copy).
- Every skip counted and reasoned; the report accounts for 100% of source
  rows.

### H.2 — Reconciliation extension (the gate)

Extend the Mission E reconcile harness to cover the newly-carried surface,
reconciled to the fils against the PH source:
- Bank suite: statement counts + line sums + running balances; cheque
  register totals; deposits-in-transit; bank cash balances.
- FX: `fx_rates` / `fx_revaluations` carried values match source.
- VAT return: `vat_returns` totals tie to the invoice/PO VAT already
  reconciled in Mission E.
- Re-run the FULL Mission E check set too (the 25) — provisioning changes
  must not have regressed the core. Target: **N/N to the fils**, N now
  larger than 25.

### H.3 — Full-provision rehearsal (the proof)

Repeat the Mission E procedure end-to-end on a FRESH snapshot copy, but now
through the complete schema + full importer: provision a from-zero DB with
the sovereign overlay + `seed_sets: ["default-assets"]`, run the extended
`phimport`, boot the app (zero errors, hashes recompute), and run H.2's
full reconciliation. This is the dress rehearsal of the actual cutover.
Out-of-repo. Record measured counts.

### H.4 — The cutover runbook + mirror

- `docs/PH_CUTOVER_RUNBOOK.md`: the exact, ordered, Commander-executable
  cutover procedure (freeze live → snapshot → provision → import →
  reconcile → re-key secrets → hash-recompute → parallel-run → switch),
  with the reconciliation acceptance gate and a rollback step. This is what
  turns "the substrate is ready" into "here is how we throw the switch."
- `docs/PH_CONVERGENCE_DECISIONS.md` — append PC-D16… for H decisions.
- `docs/PH_CONVERGENCE_PROGRESS.md` — Wave 5 section: measured import
  counts, reconciliation results, honest residue for Mission I.

---

## 5. Pending-items register (rounded up this session — address or carry forward)

From the parity certificate's residuals + this review. None block the
cutover; listed so nothing is lost:

**Deferred to Mission I (data quality — Commander-in-loop, re-runnable):**
- The darch `D:\ph_data_master` backfill + the 355-op correction patch
  (64 dedup, 202 economics-fill, 27 title-enrich, 9 broken-year, 27
  value-review) targeting the OSS schema directly. Adjudication ops (27
  value mismatches, 9 manual years) are Commander calls. See the session's
  data-backfill notes; artifacts in the prior scratchpad.

**Customer-facing appearance (stop-and-ask; Commander has seen):**
- Signature-block divergence across invoice/credit-note/DN/PO/offer PDFs
  (Commander chose keep-OSS; port via overlay-driven signer identity only
  if exact live-doc fidelity is later wanted).
- Costing datasheet attachment + PDF-bundle (`costing_attachment_service.go`,
  `costing_pdf_bundle_service.go`) ABSENT — decide before cutover if PH
  relies on bundled supplier datasheets on offers.

**Small code follow-ups (non-blocking, not this wave unless trivial):**
- D2 canonical-key route-dedup (`routeToOpportunity` matches folder_number
  verbatim, errors on dup — no case-insensitive/canonical dedup).
- `GetReportData` RBAC (PH guards it; OSS guards only `ExportReport`).
- `persistSecretSetting` — UI-saved API keys don't survive restart
  (env/in-memory only; AI/OCR optional, low priority).
- No-caller feature methods (`CreateOfferRevision`, `RenewOffer`,
  `UpdateOpportunityCommercialFields`, `GetUnifiedOfferThread`,
  `GetCostingsByOpportunity`, `GetInvoicesByAgingBucket`,
  `GetCustomerRelatedProducts`, `GetCustomerRelatedSuppliers`) — port when
  their UI is wired.

**Adjacent opportunity (not campaign-scoped; Commander tracking):** a
WhatsApp layer via GOWA (`go-whatsapp-web-multidevice`, Go + MCP) as a real
feature; a PH-vocabulary tender-radar giveaway. Noted, not tasked here.

---

## 6. Stop-and-ask registry

- Any carried financial value that does NOT reconcile to the fils — surface
  it as a live bug (in the source or the mapping), don't fudge it.
- `extracted_documents` carry/skip if the call is genuinely judgment.
- Any customer-facing document appearance change.
- The cutover execution itself.

## 7. Definition of done

- `cmd/phimport` carries the complete PH surface into the full OSS schema;
  the JSON report accounts for 100% of source rows (carry/transform/skip,
  every skip reasoned).
- The 15 banking/FX/VAT tables are populated and **reconciled to the fils**;
  the full Mission E core check set still passes (N/N, N>25).
- H.3 full-provision rehearsal is green: fresh DB → import → boot clean →
  reconcile, out-of-repo, measured.
- `go test ./...` green · `go vet` clean · svelte-check 0.
- `docs/PH_CUTOVER_RUNBOOK.md` written; progress + decisions mirrored.
- The Commander can schedule the cutover from the runbook alone.

---

## 8. Operating notes

- **Out-of-repo data work.** Real data touches only
  `C:\Projects\asymmflow\mission_e_rehearsal\` (or a fresh sibling), never
  the working tree. The repo receives importer/reconciler CODE only.
- **CWD trap:** Go module root is the repo root; pass explicit
  `path: C:\Projects\asymmflow\asymmflow-oss` for searches; `Push-Location`
  to root before `go test`/`go build`.
- **WAL checkpoint:** a file-copy of a WAL-mode SQLite won't reflect
  uncheckpointed commits — `PRAGMA wal_checkpoint(TRUNCATE)` before hashing
  or copying a DB (Mission E lesson).
- **Branch:** cut `feat/fable-ph-convergence-w5` from `main` AFTER the
  Commander merges W4 (this wave builds on the provisioned schema). Commit
  small; never push; merge is Commander-gated.
- **Measure, don't estimate. Every row accounted; every fils reconciled.** 🌊
