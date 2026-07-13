# Wave 11 Spec — Polish & True Mirror (the QA-driven wave)

**Mission:** the first hardware review of a flagship-deployment build (2026-07-13) surfaced visual defects that the compile/test gates can never see — screens that render flat and unstyled, a broken tab layout, and synthetic identity leaking through the UI as if it were data. This wave builds a **browser-driven QA loop** (Playwright against the dev server), fixes the three known defects, and lets systematic testing find the unknowns. The mirror should show the app as it actually renders, not as the code intends.

**Sequencing:** after Wave 10 (Sensory & Brand, merged `891a6ad`) and the four post-hardware fix branches (merged 2026-07-13: second paid-sound path, window-title slot, activation-screen brand slot, console-window + i18n startup fixes). Those fixes are the pattern for this wave: hardware finds, substrate fixes, synthetically.
**Repo:** `asymmflow-oss` (PUBLIC — synthetic invariant is now world-visible law). **Branch:** `feat/fable-wave11-polish` off `main`. No merge, no push, no tag.
**Operating model:** Opus 4.8 orchestrator + Sonnet 5 coders; orchestrator is the gate. Playwright drives verification — a fix without a before/after screenshot pair is not done.
**Authority docs, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` (Articles IV, VI) → `docs/DEPLOYMENT_BRANDING.md` → the Wave 9 keep-lists + all Wave 9.x/10 shipped behavior → this spec.

**Lessons inherited (do not relearn):** anchors drift — recon first · `git checkout -- frontend/dist/index.html` after builds · gate baseline: vite clean, svelte-check 0 errors/14 warnings, `go build`/`go vet` clean, `go test -count=1 -timeout 1800s ./...` green · monster files get one coder · bindings regen central · Wave-10 audits are permanent law (motion tokens one source; exactly ONE `new Audio(` construction; zero announce-class toasts; reduced-motion fully static).

---

## 0. Phase A — recon (read-only verdicts in the report)

**A0 — The QA harness (this wave's enabling work).** Determine the repeatable recipe for driving the app in a browser: `wails dev` dev-server URL (check `wails.json` `frontend:dev:serverUrl` / devserver flags) or the vite dev server directly, whichever renders truthfully (note where Wails bindings fail in a bare browser and how to stub/tolerate: the sweep must still render every screen's LAYOUT even if data calls fail — and screens that render blank without data ARE findings). Document: launch command, login/bypass with the synthetic seed, navigation map (`NAV_ITEMS` is the source of truth), and how a subagent takes a full-page screenshot per screen. This recipe goes in the report so every later wave can rerun the sweep.

**A1 — Known #1: the division identity leak.** "Acme Instrumentation" (the synthetic division) is hardcoded as a *fallback and dropdown value* across the app — census found ~10 frontend sites (`App.svelte` nav fallbacks, `QuickCaptureModal` defaults + a hardcoded `<option>`, `global.d.ts`) and backend sites (PDF export default `case "": divisionValue = "Acme Instrumentation WLL"` in `app_costing_exports_surface.go`, logger strings in `app.go`, template path names). To a real deployment these read as ANOTHER COMPANY'S NAME in their Finance screens and documents. Recon: complete the census (grep `Acme` — separate live UI/document surfaces from test fixtures/docs/demo files, which may keep it), and verdict on the design in B1.

**A2 — Known #2: Accounting & Reports render flat/grey.** On hardware, both screens render as unstyled washes — headings as plain text, stat values floating without cards, sections missing surfaces/borders (Accounting: overview stats + cashflow/evidence panels; Reports: KPI tiles + tab strip + chart areas). Reproduce in the SYNTHETIC dev build (this must not be deployment-specific — verify), then root-cause: likely candidates are a CSS file that stopped being imported, class names orphaned by a past decomposition, styles scoped to a wrapper that no longer exists, or tokens consumed before definition. These two screens were NOT among Wave 10's five skeleton hot spots' styling work — find what they depend on that's broken.

**A3 — Known #3: People (HR) detail layout.** The Employee Detail pane's section tabs (Profile / Work / Access / Compliance) render as enormous vertical pill outlines (~200px tall ovals) instead of a compact horizontal tab strip; field blocks below misalign. Root-cause (suspect: a tab component consuming a broken/missing style contract, or flex direction/height collapse).

**A4 — The full-screen sweep (find the unknowns).** Using A0's recipe: drive EVERY `NAV_ITEMS` screen plus each screen's major tabs/sub-views; screenshot at 1440×900 and one narrower width (~1100px). Catalog every visual defect into a **Defect Ledger**: screen, element, defect class (unstyled region · overflow/clipping · misalignment · contrast failure · raw i18n key · placeholder/lorem text · synthetic-identity leak · empty-state gap · token violation), severity (P1 broken/embarrassing · P2 inconsistent · P3 nice-to-have). Screenshots (synthetic identity ONLY — never deployment identity) go in `docs/wave11-qa/` with an index.

## 1. Phase B — fixes (each verified by a before/after screenshot pair)

**B1 — Division identity becomes configuration.** One synthetic source of truth for the default division display name, config-overridable per deployment:
- Backend: the overlay (`pkg/overlay` / `overlay.json`) gains a `default_division` (display name) field with the synthetic builtin default; the PDF-export default and any live code paths consume it. Logger strings/comments switch to neutral product wording ("AsymmFlow app started"), not overlay values.
- Frontend: one `DEFAULT_DIVISION` source (fed from backend config/bindings, or a brand-slot-style module if a binding is disproportionate — orchestrator's call, justified in the report); the ~10 fallback sites and the hardcoded `<option>` consume it (options should prefer real distinct division values from data where a list exists).
- **DATA CAUTION (hard rule):** `Division` is a stored row value. This wave changes DISPLAY and DEFAULT-FOR-NEW only — zero migrations, zero rewriting of stored values, zero comparison-logic changes. If a comparison site couples to the literal string, STOP and report (that's the Spec-07 canonicalization lesson).
- Test fixtures/docs may keep "Acme" (they're synthetic canon); the AC is about live UI + generated documents.
**AC:** grep finds zero "Acme" in live UI render paths and document-generation value paths; a deployment override of `default_division` changes Finance/QuickCapture/PDF defaults with no source edit; stored data untouched.

**B2 — Accounting & Reports restored.** Fix A2's root cause. These screens must meet the Wave-10 bar: real surfaces/cards on the token layer, skeletons if they're slow-loading (they weren't in Wave 10's five — add them if A2 shows they qualify), zero layout shift, empty states in operator language. If the root cause is systemic (an import lost by an old decomposition), sweep for OTHER screens silently depending on the same thing.
**AC:** both screens render composed (cards, surfaces, spacing) in the sweep screenshots; no other screen shares the root cause unfixed.

**B3 — People detail layout fixed.** Compact horizontal tab strip (or the app's canonical tab pattern — match what other screens use, Article VI: one engine), aligned field grid below.
**AC:** Employee Detail matches the app's tab/detail idiom at both sweep widths.

**B4 — The Defect Ledger burn-down.** Fix ALL P1s; fix P2s where the fix is local and behavior-frozen (batch by screen, monster files get one coder); P3s and any P2 needing structural work are RECORDED with a one-line fix sketch for a future wave — silent deferral is a report-integrity violation.
**AC:** re-run the full A4 sweep at the end: zero P1 remains; every fixed item has its before/after pair in `docs/wave11-qa/`.

**B5 — The sweep becomes a fixture.** Commit the A0 recipe + the drive script/checklist as `docs/wave11-qa/QA_SWEEP.md` (+ any helper script under `scripts/`), so "run the mirror" is a standing capability, not a one-off.

## 2. Hard boundaries

- **Flows frozen** (standing law): display, CSS, layout, and config plumbing only. No behavior, routing, permission, or data changes. The ONLY backend change authorized is B1's additive overlay field + its consumption. Financial semantics: stop-and-report, zero authorizations.
- **No stored-data rewrites** (B1's data caution is absolute).
- **Synthetic invariant, now public:** screenshots, fixtures, examples — synthetic identity only. Deployment identity (names, colors, keys) NEVER appears in this repo, its docs, or its committed screenshots.
- **Wave-10 audits hold at final commit:** motion tokens one source · one `new Audio(` construction · zero announce-class toasts · reduced-motion static · press/focus/skeleton behavior intact.
- Keep-lists + all Wave 9.x/10 behavior binding. No new deps beyond a devDependency for Playwright if needed (justify in report; prefer the already-available playwright MCP tooling). No merge, no push, no tag.

## 3. Definition of done + report

Done = A0 recipe committed; A1–A4 verdicts; B1–B5 shipped; gates green on final commit (vite clean, svelte-check 0/14, go build/vet clean, full go test green); Wave-10 audits re-verified; final sweep shows zero P1.

Write `FABLE_WAVE11_SPEC_REPORT.md`, commit it, paste verbatim as the final message — established template + the Defect Ledger (found/fixed/deferred with severities) + the screenshot index + root-cause narratives for A2/A3 (what broke, WHEN it likely broke, and what guard now prevents regression). Severity honesty is law: if a screen is still ugly after its fix, say so.
