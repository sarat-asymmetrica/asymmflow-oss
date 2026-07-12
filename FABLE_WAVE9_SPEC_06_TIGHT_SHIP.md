# Wave 9 Spec 06 — Tight Ship (Residue + Hardening Sweep)

**Mission:** Close every issue the Wave 9.5 orchestrator flagged, then run a general hardening sweep — adversarial bug hunt, end-to-end flow verification, swallowed-error cleanup, and targeted test hardening. The owner's directive, via the client SPOC: *deliver a polished system working E2E, all flows active*. This wave (and, if the hunt surfaces a large backlog, one successor) is the stability gate before the owner-reserved Sensory & Brand wave.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-6-tight-ship` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` → this spec.
**Prior art:** read all five prior reports (`FABLE_WAVE9_SPEC_0{1,2,3,4,5}_REPORT.md`). Spec-05 gate rulings now in force: B7a3 ratified ("Delivered" + real POD; dead `Signed` status stays removed); B9c KPI drill targets confirmed as built; activity monitoring stays in the SalesHub Admin tab (developer-gated) — do not move it; B2a enum consolidation, B4 Orders consumption, and the wmic replacement are all AUTHORIZED and are this wave's B1/B2/B4.

## 0. Read before anything

1. `CLAUDE.md` — layer model, synthetic-data invariant, security posture. Financial semantics: **stop-and-report**, except the three scoped authorizations recorded in §3 (B1 data migration, B3 schema flag, B4 hardware-ID implementation swap).
2. `DESIGN_CONSTITUTION.md` — Articles II (the Nine Patterns are your bug-hunt lens), III (guard ladder), V, VI, VII.
3. `FABLE_WAVE9_UIUX_AUDIT.md` — §2 (the 39-flow inventory across 7 domains) is C2's checklist; §4 keep-lists (ALL domains) are binding on every fix you ship.
4. All five prior reports — Waves 9.1–9.5 rebuilt most of these flows; the audit's verdicts are STALE in your favor. C2 re-scores them.

**Data-sensitivity invariant:** `../ph_holdings` readable for reference; real client names/figures never enter this repo.

## 1. Operating model

Identical to Spec-02..05: **Opus 4.8 orchestrator as senior designer/tech-lead** (Phase-A recon work orders → coder batches → constitutional review of every diff → central gating that never trusts a subagent's build claim → personal ownership of shared-file seams + bindings regen); **Sonnet 5 subagents** (`model: "sonnet"`, `subagent_type: "general-purpose"`) write the code.

**Lessons inherited (do not relearn):**
- Anchors drift — verify every anchor before coding (this spec's anchors were re-verified at the Spec-05 gate on 2026-07-11, but check anyway).
- `git checkout -- frontend/dist/index.html` after any build mutates the placeholder.
- Gate baseline: `npx vite build` clean · `npx svelte-check` **0 errors / 14 warnings** · `go build ./...` + `go vet ./...` clean · `go test -count=1 ./...` green. Net-new = failure.
- **Set `ENCRYPTION_MASTER_KEY` (any 64-hex value) for every `go test` run** — a bare run hangs for minutes in `getHardwareID()`'s wmic call on this Windows 11 box (the Spec-05 report proved it). B4 removes the root cause; until B4 lands, the env var is mandatory.
- `TestFileWatcher_HandlerError` is a known pre-existing timing flake (hardcoded 200ms window, misses under full-suite CPU load). B5 fixes it. Until then, a full-suite failure on ONLY that test is not a wave regression — but say so explicitly in the report, never silently ignore it.
- Monster files get ONE coder each; batches touching the same file are sequenced, never parallel. Known monsters: `OrdersScreen` (B2's target), `CostingSheetScreen`, `OffersScreen`, `WorkHub`.
- Identity/attribution resolves server-side; RBAC enforcement stays server-side, never widened for UI convenience; auth-adjacent changes get tests.

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | The two overlapping opportunity-stage vocabularies: enumerate BOTH string sets (legacy 9-stage vs current), where each is written (create/update/import paths), what `displayStage()` maps, and how many rows in a dev DB carry legacy-only strings. What does the backend validate today? | B1 |
| A2 | `LineItemsEditor.svelte` order mode: what the scaffold provides vs what OrdersScreen's inline editor does today (fields, validation, VAT/discount handling, add/remove/reorder). Diff the two behaviors precisely — B2 is behavior-preserving. | B2 |
| A3 | GRN completion signal: confirm the StockMovement-based `is_completed` derivation and the all-rejected blind spot; what a dedicated flag needs (model field, migration, backfill rule, the `canCompleteGRN` consumer). | B3 |
| A4 | `getHardwareID()` (settings_service.go:198) call-site inventory: confirm auth_handler.go:854, field_crypto.go:63 (key-derivation fallback!), settings_service.go:41. Which pure-Go WMI route is viable CGO-free (e.g. go-ole–based WMI query for `Win32_BaseBoard.SerialNumber`) vs a context-timeout'd `powershell Get-CimInstance` exec. Also inventory the OTHER wmic call sites (app_setup_documents_surface.go ~:1105, ~:1360 — diagnostics-grade). | B4 |
| A5 | Remaining `NewCache()` sites without `Stop()` (cache_test.go + ad-hoc test apps per the Spec-05 report); the `TestFileWatcher_HandlerError` wait mechanism; the undefined token names currently rendering via `var(--token, #hex)` fallbacks (`--text-danger`, B9's `.danger`, C3's 6 names) and their fallback values. | B5, B6, B7 |
| A6 | C2 feasibility: can the app be exercised live on this box (`wails dev` or a vite dev + wailsjs stub)? If yes, which flow spines are practically walkable; if no, C2 runs as static chain-of-custody tracing. Record the decision. | C2 |

## 3. Phase B — the flagged residue (all items from the Spec-05 report, ruled at the gate)

**B1 — Opportunity stage-vocabulary consolidation (AUTHORIZED data migration).** One canonical stage enum, backend-validated. Migrate legacy 9-stage rows to their canonical equivalents (A1's mapping table goes in the report verbatim); the migration is idempotent and logs each row's before→after. After migration, `displayStage()` should become a near-identity (kept as a safety net); the "legacy strings display raw via the All tab" residue dies. This is a **data-shape** change, not a money change — but if A1 reveals stage strings feeding any financial computation, STOP and report instead.
**AC:** the backend rejects non-canonical stage writes; no reachable card renders a raw legacy string; the migration mapping is in the report; migration is idempotent (safe to run twice).

**B2 — Orders onto the shared LineItemsEditor (AUTHORIZED; Costing stays the reference).** Wire OrdersScreen onto `LineItemsEditor` `mode="order"`, behavior-preserving: every field, validation, and calculation OrdersScreen's inline editor performs today survives byte-for-byte (A2's diff is your contract). All math stays in the parent screen — the component stays presentation-only (verify like Spec-05 did: the component defines zero calculation logic). ONE coder owns OrdersScreen.
**AC:** Orders line editing runs through the shared component; A2's behavior diff shows zero regressions; the component still contains no math.

**B3 — GRN completion flag (AUTHORIZED schema change).** Dedicated completion marker on the GRN model (e.g. `completed_at`), set inside the same locked transaction the Wave-9.5 idempotency guard uses. Backfill migration derives it from the existing StockMovement signal. `canCompleteGRN`/`is_completed` read the flag; the all-rejected GRN (zero accepted quantity) can now complete honestly. The row-lock idempotency guard from 9.5 stays — the flag joins it, doesn't replace it.
**AC:** an all-rejected GRN can be completed exactly once; the double-count guard still holds under the existing test; backfill is idempotent.

**B4 — Retire wmic from `getHardwareID()` (AUTHORIZED implementation swap under a HARD invariant).** The Windows path currently shells out to deprecated `wmic baseboard get serialnumber`, which stalls for minutes on Win11. **The returned value feeds licensing AND the field-crypto key-derivation fallback (field_crypto.go) — the replacement MUST return a byte-identical value for the same machine** (same source property `Win32_BaseBoard.SerialNumber`, same trimming), or existing licenses break and encrypted fields become unreadable. If byte-identity cannot be guaranteed, STOP and report — do not ship a "close enough" fingerprint.
Approach: primary = pure-Go WMI query (CGO stays banned); fallback = context-timeout'd `powershell -NoProfile Get-CimInstance Win32_BaseBoard`; last resort = the old wmic call under a short context timeout. Memoize the resolved value for the process lifetime. Add a guard test that, when wmic is present and answers within its timeout, asserts the new path returns the identical string. Sweep the two diagnostics-grade wmic sites (video controller, memory) onto CIM or a timeout — best-effort, they may not warrant byte-identity.
**AC:** a bare `go test ./...` (no env var) no longer hangs; the identity guard test passes; field-crypto and license behavior provably unchanged on the same machine.

**B5 — De-flake `TestFileWatcher_HandlerError`.** Replace the hardcoded 200ms wait with poll-until-condition + generous deadline (e.g. `require.Eventually`-style loop). Fix the wait, not the assertion.
**AC:** the test passes under full-suite CPU load — demonstrate with 3 consecutive full-suite runs in the gate.

**B6 — Finish the cache-goroutine cleanup.** `t.Cleanup(...Stop)` (or equivalent) for the remaining `NewCache()` sites A5 finds (cache_test.go + ad-hoc test apps). Mechanical.
**AC:** no test-created cache goroutine outlives its test.

**B7 — Define the missing tokens (Article VI completion).** Add the undefined token names (`--text-danger` + B9's/C3's names per A5) to theme.css **at their current fallback values** — do NOT collapse them onto the differently-valued existing `--danger`/`--success`/`--warning` (Spec-05 deliberately kept them apart; changing rendered colors is a sensory-wave decision, not yours). Then the `var(--x, #hex)` fallbacks at those sites may drop the hex. Zero visual change.
**AC:** the A5 token list resolves from theme.css; grep shows no fallback hex remaining for those names; rendered colors are pixel-identical.

## 4. Phase C — the hardening sweep

**C1 — Adversarial bug hunt (loop-until-dry).** Parallel read-only finder agents per domain (sales, inventory, finance, people/projects, shell/nav/settings), each briefed with a distinct lens drawn from what Wave 9.5's recon kept finding in the wild:
- *Backend-rejects-what-UI-sends:* payload/status strings, enum casing, required fields the UI doesn't collect (the PO-grid class of bug).
- *Unreachable states:* buttons gated on conditions that can never hold; statuses no path sets (the DN-Confirm class).
- *Silent failure:* awaited calls whose failure changes nothing visible; best-effort writes that swallow (the costing-persist class).
- *Handoff breaks:* pending-store producers with no consumer, or consumers reading fields the producer doesn't set.
- *Stale bindings:* wailsjs calls to methods that changed signature; unused-but-rendered data.
Triage ladder for every find: **fix-in-wave** (small, in-scope, non-financial, keep-list-safe — batch to coders by file ownership) vs **report-only** (financial semantics, schema beyond B1/B3, anything keep-list-adjacent). Hunt until one full round of fresh finders returns nothing new that survives your triage. Every find — fixed or not — appears in the report with its verdict.

**C2 — E2E flow verification (the SPOC's ask made checkable).** Re-score the audit's 39 flows (`FABLE_WAVE9_UIUX_AUDIT.md` §2) against TODAY'S code. Per A6's feasibility verdict: live-walk what's walkable, and for the rest run a static chain-of-custody trace — UI control → handler → wailsjs binding → Go method → persistence/status transition → return → UI refresh — recording the exact break point for anything not 🟢. Deliverable: the full 39-row table (flow · 2026-07-10 audit verdict · today's verdict · evidence · break point if any) in the report. Every 🔴 found is a C1-triage item: fix it in-wave or report why not. This table is the wave's headline artifact.

**C3 — Swallowed-error sweep (frontend).** Inventory `catch` blocks in `frontend/src` that reduce a failure to `console.*` or nothing. Classify: *legitimately optional* (graceful degradation with a sane default — leave, list) vs *must-surface* (a user action that can silently fail). Fix must-surface sites with the established pattern: visible toast + error state + retry where cheap (the B9b/B3c precedent). No new UX inventions — Article II patterns only.
**AC:** the report lists every site with its classification; must-surface sites are fixed.

**C4 — Targeted test hardening (Go).** Tests for Wave-9-built behavior that shipped without them, priority order: DN full-delivery detection + `CreateDNWithSerials` failure-cleanup; PO transition validation (legal accepted, illegal rejected, canonical strings); `GetOrderDeliveryStatusBatch` shape/empty/missing-order cases; GRN completion after B3. Hermetic, table-driven where natural, matching existing test idiom. This is coverage for what Waves 9.x BUILT — not a coverage crusade across the legacy surface.
**AC:** each listed behavior has at least one meaningful test; full suite green.

### Suggested coder batching (adjust from Phase A; respect the monster-file rule)
- Coder 1: B1 (backend enum + migration + OpportunitiesScreen raw-display cleanup)
- Coder 2: B2 (OrdersScreen — the monster; one owner, includes its C1/C3 fixes if any land there)
- Coder 3: B3 + B5 + B6 (GRN schema flag + test-infrastructure fixes)
- Coder 4: B4 (hardware ID + wmic sweep — isolated, security-adjacent, gets the guard test)
- Coder 5: B7 + C3 fixes (tokens + must-surface error sites)
- C1 finders + C2 tracers: parallel read-only agents; the orchestrator triages centrally and batches fixes to coders by file ownership
- Orchestrator: seams, bindings regen, triage ledger, constitutional review

## 5. Hard boundaries

- **No sensory/brand work** (owner-reserved next wave; Article IV.3's sound budget untouched). No new feature domains. No visual redesign — C2/C1 fixes restore intended behavior, they don't reimagine it.
- **Financial semantics: stop-and-report** — the ONLY authorized changes are B1 (stage data migration), B3 (completion flag + backfill), B4 (hardware-ID swap under byte-identity). Rounding, posting order, tax/VAT math, and payment application are untouchable.
- **Keep-lists from the audit §4 (ALL domains) + every prior wave's shipped behavior are binding.** A hardening fix that regresses a keep-list item is a failure, not a trade.
- **RBAC:** enforcement server-side, never widened; auth-adjacent changes get tests (B4 explicitly).
- **Explicitly deferred (owner-acknowledged, do NOT do):** GL opening-balance carry-forward; visa/permit tracking; allocation capacity enforcement; repo-wide CRLF normalization; WorkHub/CustomerDetailView wholesale decomposition; activity-monitoring relocation.
- **No merge, no push, no tag.**

## 6. Definition of done + status report

Done = Phase A verdicts recorded; B1–B7 shipped or explicitly skipped with reason; C1 hunted to dry (one full fresh round with no surviving new finds) with the complete triage ledger; C2's 39-row table complete with every 🔴 fixed or triaged; C3 classified + must-surface fixed; C4 tests in; gates green on the final commit (with the ENCRYPTION_MASTER_KEY caveat gone if B4 landed).

Write `FABLE_WAVE9_SPEC_06_REPORT.md`, commit it, and paste it verbatim as your final message (the established template: Phase A verdicts, per-item status + commits, the C1 triage ledger, the C2 flow table, decisions, constitution deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green — the SPOC asked for a tight ship, and a tight ship is one whose leaks are KNOWN.
