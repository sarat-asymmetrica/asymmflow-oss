# PH Cutover Runbook — from sovereign PH deployment to OSS AsymmFlow

Mission H deliverable (PH convergence Wave 5). This is the exact, ordered,
Commander-executable procedure that turns "the substrate is ready"
(`PH_PARITY_CERTIFICATE.md`) into a completed switch. It was dress-rehearsed
end-to-end on a PH snapshot (see `PH_CONVERGENCE_PROGRESS.md`, Wave 5) —
every step below is the rehearsed step, not a hypothesis.

**Invariants in force throughout:** real PH data never enters this
repository; all data work happens in an out-of-repo working directory on the
operator's machine. Every carried financial value reconciles to the fils or
the cutover ABORTS — a mismatch is a live bug, never a round-off.

---

## 0. Prerequisites (before freeze day)

- [ ] A build of the OSS app (`wails build -clean`) from the release commit,
      plus `phimport.exe` and `phreconcile.exe` built from the SAME commit:
      `go build -o phimport.exe ./cmd/phimport` and
      `go build -o phreconcile.exe ./cmd/phreconcile`.
- [ ] PH's sovereign `overlay.json` (divisions, TRN, bank strings, aliases,
      `"seed_sets": ["default-assets"]`) — lives only on PH machines.
- [ ] A working directory on the target machine, e.g. `C:\asymmflow_cutover\`,
      with the app in `app\` and `overlay.json` next to the executable.
- [ ] The secrets to re-enter by hand after import (encrypted settings are
      machine-bound and never carried): Mistral/OCR API keys, cloud-sync
      `DATABASE_URL` if used.
- [ ] Agreement on the freeze window with every PH user (no writes during
      cutover) and a named rollback owner.

## 1. Freeze the live system

1. Announce the freeze; all users close AsymmFlow on every machine.
2. On the primary machine, verify no process holds the DB (no fresh
   `ph_holdings.db-wal` growth; Task Manager shows no AsymmFlow).
3. Disable the background cloud sync for the window (unset
   `ENABLE_CLOUD_SYNC` or stop the sync service) so nothing writes mid-copy.

## 2. Snapshot

1. If a `ph_holdings.db-wal` sidecar exists, copy `ph_holdings.db`,
   `ph_holdings.db-wal`, `ph_holdings.db-shm` TOGETHER into the working
   directory, then checkpoint the copy:
   `PRAGMA wal_checkpoint(TRUNCATE)` (a file-copy of a WAL-mode SQLite
   without its WAL does not contain uncheckpointed commits — Mission E
   lesson). If no sidecar exists, a plain copy of the `.db` is complete.
2. Name it `source_copy.db`. This file is also the archival snapshot: it
   preserves every legacy column the migration faithfully drops (the
   import report's `column_drops` section), so nothing is ever
   unrecoverable.
3. Record its SHA-256.

## 3. Provision the destination

1. Set the destination path and run the app once against a FRESH file:
   `$env:PH_DB_PATH = 'C:\asymmflow_cutover\asymmflow.db'` then start
   `AsymmFlow.exe` (with `overlay.json` beside it, seed_sets
   `["default-assets"]` only).
2. Wait for the dashboard to render (schema + foundations provisioned),
   then close the app cleanly.
3. Verify the fresh file provisioned the full surface — the banking/FX/VAT
   suite plus `fiscal_periods` and `customer_name_mappings` must exist
   (they are criticalDeploymentModels; if any is missing the build is
   stale — STOP).

## 4. Import

1. `phimport.exe -source source_copy.db -dest asymmflow.db > import_report.json`
2. Read the JSON report END TO END. Gate conditions:
   - `unmapped` MUST be empty. A non-empty unmapped list means PH grew a
     table this runbook has never adjudicated — STOP and resolve.
   - Every `skipped` entry carries a reason and an honest row count;
     confirm nothing skipped surprises you (the expected skips are the
     adjudicated set: scratch/SSOT/shadow tables, `*_backup`,
     `intelligence_*`, `extracted_documents`, data-update mechanism,
     sessions/sync state, license keys, sqlite internals).
   - `column_drops` lists legacy columns that held data and were
     faithfully dropped (PH's own app reads none of them) — this is the
     Mission I work-list, preserved in the §2 archival snapshot.
   - Note `encrypted_settings_skipped` (re-entered in §7) and the PC-D7
     receipt transform counts.

## 5. Reconcile (THE gate)

1. `phreconcile.exe -source source_copy.db -dest asymmflow.db > reconcile_report.json`
2. Exit code 0 and `"pass": true` required: every check — counts on the
   full carry surface, money to the fils on invoices/payments/credit
   notes/supplier ledger/POs, the banking/FX/VAT suite — must match.
3. **Any mismatch ABORTS the cutover.** Diagnose against the report's
   side-by-side values; a mismatch is a bug in the source data or the
   mapping and goes back through review — never hand-adjust the
   destination.

## 6. First boot + hash recompute

1. Start the app on the imported file (`PH_DB_PATH` still set).
2. Watch the log: startup must complete with zero errors, and the
   invoice/credit-note hash backfill must recompute every blanked hash
   under the new install's salt (Mission E measured 480/480).
3. Close the app and re-run the reconciliation AGAINST THE BOOTED FILE:
   `phreconcile.exe -source source_copy.db -dest asymmflow.db` must still
   pass. Provisioning and startup "ensure" paths are writers too — the
   Mission H rehearsal's post-boot reconcile is what caught the demo
   bank-fixture seed (PC-D18). A post-boot mismatch ABORTS like any other.
4. Start the app again and spot-check in the UI: dashboard totals, one
   known invoice PDF, one supplier ledger page, bank reconciliation screen
   shows the carried statements.

## 7. Re-key secrets

1. In-app settings: re-enter the API keys and any cloud-sync credentials
   (count must equal `encrypted_settings_skipped` from §4).
2. Restart once; confirm the integrations that matter to PH (OCR inbox,
   Butler if used) come up, or are cleanly disabled.

## 8. Parallel run (recommended: 1–2 business days)

1. Users work in the NEW app; the frozen legacy install stays untouched as
   the fallback.
2. At end of window, re-run `phreconcile.exe` source↔dest — the carried
   history must still tie (new rows will show only in checks whose
   destination side grew; that divergence must consist exactly of the
   period's new business, which the operator verifies against the day's
   documents).
3. Commander sign-off ends the parallel run.

## 9. Switch (and rollback)

- **Switch:** remove the legacy binary from user machines / shortcuts;
  the OSS install is now the system of record. Archive `source_copy.db`
  (with its SHA-256) and both reports alongside the cutover date.
- **Rollback (any gate failed):** delete the imported destination file,
  unfreeze the legacy install, announce; nothing in the legacy system was
  ever modified by this procedure — the import reads the source strictly
  read-only. Re-attempt after the defect is fixed and re-reviewed.

---

## Acceptance summary (what "done" means)

| Gate | Artifact | Pass condition |
|---|---|---|
| Import accounting | `import_report.json` | 100% of source tables carried/transformed/skipped-with-reason; `unmapped` empty |
| Reconciliation | `reconcile_report.json` | all checks match to the fils; exit 0 |
| Boot | app log | zero startup errors; all hashes recomputed |
| Parallel run | re-run reconcile + operator verification | carried history still ties; day's deltas match the day's documents |
| Sign-off | Commander | explicit |
