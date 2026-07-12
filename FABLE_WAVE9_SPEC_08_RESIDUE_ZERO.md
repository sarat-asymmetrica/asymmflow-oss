# Wave 9 Spec 08 — Residue Zero (The Deferred Ledger, Ratified)

**Mission:** Close every PH-domain item the Wave 9.x reports deferred with "owner decision" or "next wave." After this wave, the only open work on the substrate is ecosystem hardening (Spec-09) and the owner-reserved Sensory & Brand wave (Wave 10). Nothing on the deferred ledger survives.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-8-residue-zero` off `main` (Wave 9.7 merged at `b4e88fa`). Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` (keep-lists) → this spec.
**Prior art:** all seven prior reports (`FABLE_WAVE9_SPEC_01..07_REPORT.md`). Each B item below cites the report that deferred it — read that citation before coding.

## 0. Ratifications recorded (2026-07-12)

These owner decisions are now law; record them, do not relitigate:

1. **Offer.Stage stays on its own DB-CHECK vocabulary** ('RFQ','Quoted','Won','Lost','Expired'), separate from the canonical Opportunity/RFQ enum (Spec-07 §7.1 — ratified as recommended). The dormant Cap'n Proto enum also stays untouched. The comment in `stage_vocabulary.go` documenting this is now constitutional.
2. **Wave 9.7 merged to main** after independent owner-side review (`b4e88fa`); all five Spec-07 financial authorizations are shipped law.
3. **The entire deferred ledger is authorized for closure**, split across this spec (PH-domain) and Spec-09 (ecosystem). The per-item scoping in §3 below is the exact authorization — no wider.

**Data-sensitivity invariant:** `../ph_holdings` readable for reference; real client names/figures never enter this repo. B5's DB heuristic runs against an owner-provided copy OUTSIDE the repo tree; only aggregate counts enter the report.

## 1. Operating model

Identical to Specs 02–07: Opus 4.8 orchestrator as senior designer/tech-lead; Sonnet 5 coders in file-disjoint batches; Phase-A recon first; constitutional review of every diff; central gating + bindings regen.

**Lessons inherited (do not relearn):** anchors drift — verify before coding · `git checkout -- frontend/dist/index.html` after builds · gate baseline: vite clean, svelte-check 0 errors/14 warnings, `go build`/`go vet` clean, full `go test -count=1 -timeout 1800s ./...` green · `TestFileWatcher_HandlerError` failures are REAL · monster files get one coder · bindings regen central · identity/SoD resolves server-side · canonicalize BOTH sides of any invariant that compares stage/status vocabulary (the Spec-07 review lesson) · schema-golden changes are deliberate: regenerate with `-update-schema-golden` and review the diff (expect only your tables/cols).

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | GRN receive panel + PO payload today: where PO line data is assembled for `PurchaseOrdersScreen.svelte`'s receive panel (Wave 9.7 B5), whether `RequiresSerialTracking` exists on the product/inventory model server-side, and the cheapest honest path to per-line serial enforcement in the panel. What would wiring `RaiseGRNDiscrepancy`/`ResolveGRNDiscrepancy` into the panel take (both methods exist, unwired since 9.6)? | B1 |
| A2 | GL close model: how `ExportGeneralLedgerCSV` (Spec-03 B5, `statement_export_service.go`) derives per-account movement; what a minimal prior-period opening-balance carry needs (computed roll-forward at export time vs a stored period-close row); whether `reportYear` scoping gives a natural period boundary. What PH does, if anything. | B2 |
| A3 | Allocation reality: `ProjectMember.AllocationPercent` capture path (Spec-04 — captured 0–100, informational), every read site, and what a per-person cross-project total needs. Warn-vs-block precedents in the codebase for soft guards. | B3 |
| A4 | Employee compliance surface: the `EmployeeProfile` model + PeopleHub editor structure (post-9.4 re-model); where document-expiry-shaped data would live (new columns vs a child table); whether any notification/approvals hook fits expiry warnings (Article V home exists since 9.4). What PH has for visa/CPR/permit, if anything (audit says nothing — confirm). | B4 |
| A5 | Stock-history repair surface: the exact shape of pre-9.7 double-posted Adjustment movements (no `reference_id` — Spec-07 B2), what uniquely identifies a double vs a legitimate repeated adjustment (same item+qty+direction+~timestamp), and what a reversal-style repair (compensating movement, never a delete) would look like under Article III. | B5 |
| A6 | The `rfq_datas` plural no-op: confirm `app.go:1279`'s `addColumnIfNotExists("rfq_datas", …)` targets a table that never exists (live table is `rfq_data`), what column it thought it was adding, and whether the singular table already has it. | B6 |

## 3. Phase B — ratified closures

**B1 — GRN finishing (Spec-07 residue B5 + B7c).**
(a) Thread the per-line serial-tracking flag into the PO receive panel: serialized lines REQUIRE serial entry (count == qty), non-serialized lines don't show serial inputs. Follow A1's cheapest honest path — if the flag must join the PO line payload, that is authorized; no new posting logic, the backend `len(serials)==qty` check stays the enforcement of record.
(b) Wire `RaiseGRNDiscrepancy` into the receive panel: when received-now < remaining and the operator marks a rejection/short-ship, the discrepancy records through the existing backend (reason required). Resolution stays wherever `ResolveGRNDiscrepancy` naturally surfaces per A1 — do not build a new screen for it; a list in an existing operations surface is enough.
**AC:** a serialized line cannot be received without its serials; a short-ship can be recorded as a discrepancy from the same panel; no changes to posting/locking; deprecated GRNScreen stays retired.

**B2 — GL opening-balance carry-forward (Spec-03 §Q5, AUTHORIZED as an export-truth fix).**
The General Ledger export currently shows movement-within-year only. Ship A2's minimal honest carry: each account's opening balance = prior-period net movement (computed roll-forward is fine; a stored close model is NOT required unless A2 shows computation is wrong, not just slow). The CSV header note ("opening balances not carried") is removed only when they actually are.
**Hard scope:** this changes the EXPORT and any on-screen GL card that shares the derivation. It does not touch posting, journals, or any transactional table.
**AC:** a ledger exported for year N opens each account with the carried balance from years < N; year-1 opens at zero; export column semantics documented in the report.

**B3 — Allocation capacity enforcement (Spec-04 §Q5, ratified as WARN, not block).**
`AllocationPercent` stops being purely decorative: when saving a member allocation would push that person's cross-project active-allocation total above 100%, the UI warns with the actual numbers (who, what total, which projects) — but saves on confirm. Server computes the total (never trust the client's sum); no hard block, no schema change.
**AC:** over-allocation triggers the warning with true totals; confirm still saves; totals computed server-side; existing project flows byte-identical otherwise.

**B4 — Visa/CPR/permit tracking (audit :78-91 feature gap, AUTHORIZED — the one NEW feature of this wave).**
A Bahrain company legally tracks employee document expiries. Ship the minimal compliant surface per A4: document fields (CPR number + expiry; passport number + expiry; visa/permit type + number + expiry) on the employee compliance surface, admin/HR-gated, PII-conscious (these are exactly the fields FieldCrypto exists for — encrypt at rest per the existing pattern). Expiring-soon (≤60 days) surfaces through the EXISTING notification/Article-V home A4 identifies — no new alerting subsystem.
**Schema note:** adding columns/tables touching `tradingModels()` trips the schema golden — regenerate deliberately.
**AC:** documents + expiries recorded per employee, encrypted at rest, gated; expiring documents visible without opening each profile; zero payroll/money-math contact.

**B5 — Historical stock-movement repair plan (Spec-07 §7.3, REPORT + DRAFT ONLY).**
Run the Spec-07 heuristic (duplicate Adjustment movements grouped by item+qty+direction) READ-ONLY against an owner-provided copy of the live PH DB (path supplied at runtime; never committed, never inside the repo). Deliver: (a) the count + a per-item aggregate table (no client-identifying free text in the report — item IDs only); (b) a drafted, idempotent repair script that posts COMPENSATING movements (Article III — never deletes history), gated behind an explicit env flag, with a dry-run mode, following the B2-toolkit pattern from the ph_holdings campaign (diagnose/repair/verify/checkpoint). **The script is delivered NOT executed** — the owner runs it against production.
**AC:** count reported; repair script exists with dry-run + verify; zero writes performed by this wave.

**B6 — Micro-residue.**
(a) Fix the `app.go:1279` `rfq_datas` plural no-op per A6: point it at the real table if the column is genuinely missing, or delete the dead call with a comment if the singular table already has it.
(b) Record ratification §0.1 (Offer.Stage separation) wherever the constitution keeps such rulings, so no future wave re-opens it.
**AC:** no dead migration calls; the ruling is findable.

## 4. Hard boundaries

- The §3 authorizations are exact: **nothing else financial moves.** Posting, rounding, payment application, payroll math: untouchable. B2 changes derivation-for-display/export only; B5 writes NOTHING.
- B4 is the only new feature domain; it stays inside the employee-compliance surface. No general document-management system.
- No sensory/brand work (Wave 10 reserved). No ecosystem items (CRLF, decompositions, DPAPI, staticcheck — all Spec-09; do NOT drift into them even opportunistically).
- Keep-lists (audit §4) + all Wave 9.x shipped behavior binding. RBAC server-side, never widened; auth-adjacent changes get tests. PII additions (B4) get FieldCrypto, not plaintext columns.
- No merge, no push, no tag.

## 5. Definition of done + status report

Done = Phase A verdicts; B1–B6 shipped or explicitly skipped with reason; B5's count + script delivered-not-run; gates green on the final commit.

Write `FABLE_WAVE9_SPEC_08_REPORT.md`, commit it, and paste it verbatim as your final message (established template: Phase A verdicts, per-item status + commits, decisions, deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green.
