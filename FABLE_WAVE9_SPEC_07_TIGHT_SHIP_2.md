# Wave 9 Spec 07 — Tight Ship 2 (The Report-Only Backlog, Ratified)

**Mission:** Close the well-characterized backlog the Spec-06 hunt surfaced and escalated. Every item below was found, verified, and triaged by Wave 9.6; the owner has now ruled — the authorizations in §3 are recorded here and are binding. This is the last stability wave before the repo-hygiene/cloud-push milestone and the owner-reserved Sensory & Brand wave (Wave 10, spec already on main).
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-7-tight-ship-2` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` (keep-lists) → this spec.
**Prior art:** read all six prior reports (`FABLE_WAVE9_SPEC_01..06_REPORT.md`). The Spec-06 report's C1 triage ledger and A1 stage-mapping table are load-bearing inputs to this wave — read them closely.

## 0. Read before anything

1. `CLAUDE.md` — financial semantics are stop-and-report EXCEPT the five scoped authorizations recorded in §3 (B1, B2, B3, B4, B5). Those five are authorized exactly as scoped — no wider.
2. `DESIGN_CONSTITUTION.md` — Article III (the guard ladder decides WHERE a stock movement posts — at the authorization moment), Article II #7 (integrity surfaced), Article VII.
3. The Spec-06 report — its residue list is this spec's table of contents.

**Data-sensitivity invariant:** `../ph_holdings` readable for reference; real client names/figures never enter this repo.

## 1. Operating model

Identical to Specs 02–06: Opus 4.8 orchestrator as senior designer/tech-lead; Sonnet 5 coders in file-disjoint batches; Phase-A recon first; constitutional review of every diff; central gating.

**Lessons inherited (do not relearn):** anchors drift — verify before coding · `git checkout -- frontend/dist/index.html` after builds · gate baseline: vite clean, svelte-check 0 errors/14 warnings, `go build`/`go vet` clean, full `go test -count=1 -timeout 1800s ./...` green (the main package legitimately needs >600s under load; ENCRYPTION_MASTER_KEY is no longer required — B4/Wave-9.6 fixed the hang) · `TestFileWatcher_HandlerError` is de-flaked; a failure there is now REAL · monster files get one coder · bindings regen central · identity/SoD resolves server-side.

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Re-verify the Spec-06 stage findings: the 5 allowlists + 2 unvalidated write paths (OneDrive import, SSOT importer); exactly which literals app_prediction_dashboard.go:395-551 branches on; how many rows per legacy stage exist in a dev DB; where MarkOfferWon/MarkOfferLost update Status but not Stage. | B1 |
| A2 | Stock-adjustment double-post: confirm CreateStockAdjustment AND ApproveStockAdjustment both post a StockMovement; enumerate every caller of each (incl. Butler); what an un-approved adjustment should hold (nothing? a pending row?); how many historical doubled movements a dev DB shows. | B2 |
| A3 | Expense approval path vs the supplier-invoice SoD precedent (CreatedBy != approver): where expense approval happens, what identity is available server-side, what test pattern the supplier gate uses. | B3 |
| A4 | The `Number(vat_percent) || 10` sites: every occurrence of the `|| <rate>` fallback pattern on VAT/tax fields across frontend + any Go analog; which are reachable; what a genuine 0% invoice does today end-to-end. | B4 |
| A5 | Goods-receipt restoration surface: what remains alive of the receiving chain (reconcileInventoryReceipt, assignSerialsToGRN, GRN model + Wave-9.6 CompletedAt flag) vs what the deprecated GRNScreen alone provided; the minimal PO-side surface for receive-against-PO (the PO detail's existing structure, the Wave-9.5 status-driven buttons). | B5 |
| A6 | Payroll comp-profile: confirm the per-employee-unique model vs per-company UI mismatch and the exact clobber sequence; what the safe fix is (scope the profile per employee+company, or make the UI honestly per-employee). | B6 |
| A7 | Small-fry anchors: order manual-create's 2 non-atomic calls; DeliveryTracking's swapped CreateShipment args + nil-deref (and whether the screen is reachable); the GRN-discrepancy placeholder; employee-archive dead "pending approval" code; WorkHub admin buttons' role visibility; CreateDNWithSerials orphaned DeliveryNoteItem rows on cleanup; the two empty catches (QuotationScreen:99, UserManagementScreen:107). | B7 |

## 3. Phase B — ratified fixes

**B1 — Stage-vocabulary consolidation, THE FULL FIX (AUTHORIZED — mapping RATIFIED).** The owner has ratified the Spec-06 mapping table **as printed in the Spec-06 report**, including the two flagged rows: Follow-up/Eval → Quoted, and the four post-sale stages (PO/LOI Received, Order Placed, In Process, Delivered) → Won (fulfillment already lives on the Order — the opportunity's job ended at Won).
Ship: (a) ONE canonical stage enum, enforced on ALL write paths — the 5 allowlists unified, the 2 unvalidated importers gated; (b) the idempotent historical migration per the mapping, logging every row's before→after; (c) MarkOfferWon/MarkOfferLost also update RFQData.Stage (kills the stale-Stage pipeline deflation); (d) `displayStage()` retained as a safety net.
**The owner accepts that WinRate and PipelineValueBHD will move** — that is the point; mis-tagged rows were lying. The report MUST quote the aggregate before→after (WinRate %, PipelineValueBHD) from a dev DB so the movement is seen, not suffered.
**AC:** non-canonical writes rejected everywhere including imports; migration idempotent (safe twice); the before→after aggregates printed; win-rate no longer deflated by stale Stage.

**B2 — Stock-adjustment double-post (AUTHORIZED).** The movement posts at the **authorization moment**: `ApproveStockAdjustment` owns the StockMovement; `CreateStockAdjustment` creates the pending adjustment WITHOUT posting (Article III — posting is a guarded, named act). Butler's path follows automatically. Historical doubled movements: **report the count, do not auto-repair** (data surgery on stock history is a separate owner decision). Add the regression test: create→approve = exactly one movement; create alone = zero.
**AC:** one approved adjustment = one movement; unapproved = none; historical damage counted and reported, untouched.

**B3 — Expense SoD (AUTHORIZED).** Creator ≠ approver, enforced server-side, mirroring the supplier-invoice precedent exactly (same identity resolution, same error shape, same test pattern). UI surfaces the block reason.
**AC:** self-approval rejected server-side with a test; a second identity approves normally.

**B4 — The `|| 10` VAT fallback (AUTHORIZED — parsing bug, not a rate change).** A genuine 0 is a legal VAT rate (zero-rated/export). Replace `Number(x) || 10` with an explicit null/undefined/NaN check that preserves 0 at every A4 site. The default-when-absent stays 10 — ONLY the treatment of an explicit 0 changes. Test: a 0% invoice keeps 0% end-to-end.
**AC:** explicit 0% survives; absent still defaults to 10; every A4 site converted.

**B5 — Lean goods receipt (RECOMMENDED-RATIFIED; the owner may veto at gate).** Restore receiving as a **PO-flow action**, not a resurrected GRNScreen: on a Confirmed/Sent PO, a "Receive items" action (status-driven button, Wave-9.5 pattern) opens a focused receive panel — per-line received qty (remaining-qty defaulted — keep-list), optional serial entry for serialized lines — and posts through the EXISTING backend chain (reconcileInventoryReceipt, serial minting, the 9.5 row-lock + 9.6 CompletedAt guard). This un-severs the only stock-write and serial-mint paths and makes PO "Received" true again. The deprecated GRNScreen stays retired.
**AC:** receiving a PO updates stock and mints serials through the existing guarded backend; partial receipts work; PO "Received" reflects reality; the old screen stays gone.

**B6 — Payroll comp-profile clobber (AUTHORIZED as a data-integrity guard, NOT a payroll-math change).** Per A6: either scope the profile correctly (employee+company) with a migration, or make the save path refuse cross-division clobbers with a clear error — whichever A6 shows is the honest minimal fix. The payroll state machine and money math remain untouched.
**AC:** switching companies + save can no longer overwrite another division's profile; payroll computation byte-identical.

**B7 — Small fry (all verified by A7 first).**
(a) Order manual-create atomic (one transactional backend call; no ghost itemless orders).
(b) DeliveryTracking: fix the swapped args + nil-deref if reachable; if unreachable, retire it honestly per the constitution's retire rules (record which).
(c) GRN-discrepancy placeholder: remove the stub UI (a control that does nothing is a lie); backend CRUD only if B5 makes it reachable and cheap — else report.
(d) Employee-archive dead "pending approval" review: make the review real (it has an ApprovalsQueue home since 9.4) or remove the dead path and label the flow honestly.
(e) WorkHub project-admin buttons hidden for roles the backend rejects (visibility only — server enforcement already correct).
(f) CreateDNWithSerials failure-cleanup also removes DeliveryNoteItem rows (closes the 9.6 C4 observation).
(g) Hardware-ID persistence hardening: persist the resolved hardware ID on first successful resolution (settings/local store) and prefer the persisted value thereafter — so timeout variance on a wedged-WMI box can never flip key material between boots. (Gate finding from the 9.6 review; the Wave-9.6 resolution order is otherwise correct.)
(h) The two empty catches get minimal honest handling.

## 4. Hard boundaries

- The five §3 authorizations are exact: **nothing else financial moves.** Rounding, posting order, payment application, payroll math: untouchable.
- Historical stock-movement repair (B2's count) and any GL work: report-only.
- No sensory/brand work (Wave 10 is specced and reserved). No new feature domains beyond B5's ratified receive action.
- Keep-lists (audit §4) + all Wave 9.x shipped behavior binding. RBAC server-side, never widened; auth-adjacent changes get tests.
- **Explicitly deferred (do NOT do):** GL opening-balance carry-forward; visa/permit; allocation enforcement; CRLF normalization; WorkHub/CustomerDetailView decomposition; activity-monitoring relocation.
- No merge, no push, no tag.

## 5. Definition of done + status report

Done = Phase A verdicts; B1–B7 shipped or explicitly skipped with reason (B5 carries its own veto note if the owner overrides at gate); the B1 aggregate before→after and B2 historical count printed in the report; gates green on the final commit.

Write `FABLE_WAVE9_SPEC_07_REPORT.md`, commit it, and paste it verbatim as your final message (established template: Phase A verdicts, per-item status + commits, decisions, deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green.
