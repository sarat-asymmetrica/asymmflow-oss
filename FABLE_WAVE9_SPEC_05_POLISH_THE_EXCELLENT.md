# Wave 9 Spec 05 — Polish the Excellent (The Last Flow Wave)

**Mission:** Wave 9.5 from `FABLE_WAVE9_UIUX_AUDIT.md` §5 — invest in the app's strengths: sales residuals, line-editor convergence, PO/GRN/DN finishing, nav IA, dashboard refinements — plus four items ruled at the Spec-04 gate. This is the FINAL flow wave; after it comes the owner-reserved Sensory & Brand wave. Leave the app so that wave has nothing to fix, only to delight.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave9-5-polish` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` → `FABLE_WAVE9_UIUX_AUDIT.md` → this spec.
**Prior art:** read all four prior reports (`FABLE_WAVE9_SPEC_0{1,2,3,4}_REPORT.md`). Spec-04 gate rulings now in force: `users:view` gating ratified as built; allocation stays informational (no capacity enforcement); the Article VI consistency-with-file hex cases ratified — and C3 below closes the SettingsScreen one properly.

## 0. Read before anything

1. `CLAUDE.md` — financial semantics stop-and-report (zero authorizations this wave); RBAC server-side + tests where touched.
2. `DESIGN_CONSTITUTION.md` — Articles II (all nine patterns), III, IV.1–IV.2 (responsiveness/motion — B10 is Article IV.1 made real), VI.
3. `FABLE_WAVE9_UIUX_AUDIT.md` §4.1 (sales — **the binding keep-list this wave lives inside**), §4.2 (inventory), §4.6–4.7 residuals, §5 Wave 9.5.
4. All four prior reports.

**Data-sensitivity invariant:** `../ph_holdings` readable for reference; real client names/figures never enter this repo.

## 1. Operating model

Identical to Spec-02/03/04: **Opus 4.8 orchestrator as senior designer/tech-lead**; **Sonnet 5 subagents** (`model: "sonnet"`, `subagent_type: "general-purpose"`) code in file-disjoint batches; constitutional review of every diff; central gating; orchestrator owns shared-file seams + bindings regen.

**Lessons inherited (do not relearn):** audit anchors drift — verify before coding · `git checkout -- frontend/dist/index.html` after builds · gate baseline `vite build` clean / `svelte-check` **0 errors / 14 warnings** / `go test ./...` green with at least one `-count=1` full run · identity resolves server-side · monster files get ONE coder (OffersScreen ~2000+ lines and OrdersScreen are this wave's monsters).

**Polish-wave discipline:** these screens are the app's crown jewels — the audit graded them 🟢 *with* these defects. Every fix must be a strict improvement: when in doubt between a clever restructure and a minimal correct fix, take the minimal fix. The keep-list is not a checklist to re-verify at the end; it is the operating envelope.

## 2. Phase A — recon (read-only, do first; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Offers: do the two expiry notions (client-computed `is_expired` vs `stage === 'Expired'`) both exist on OSS? Where does each surface? What sets stage to Expired (job? on-read)? | B1 |
| A2 | Opportunities: which create-form fields does the backend actually persist; deprecated-stage filter list; Edit-modal field census; change-log load path. | B2 |
| A3 | Costing: where the 10-line cap lives (UI array? backend?); the Save-as-Offer persist path and its failure handling; whether a standalone save endpoint exists. | B3 |
| A4 | Line editors: diff Costing's line editor vs Orders' manual editor — fields, calculations, components; what a shared component must cover. PO workspace status grid + backend legal-transition rules. GRN Complete/qc_status semantics + manual-GRN path. | B4, B5, B6 |
| A5 | DN: Confirm-Delivery signature handling, 'Signed' status reachability, serial create-then-patch path, remaining-qty→zero detection; Orders list perf (page size, N+1 delivery-status calls — count them). | B7, B10 |
| A6 | Nav: sidebar list vs Alt+N array vs screenPermissions keys; where "Relationships" label lives; Deployment reachability. FinancialDashboard: ratio thresholds, the silent-fallback dataset, YoY prose. | B8, B9 |
| A7 | Admin: conflict-tools' exact scope in UserManagementScreen; `GenerateLicenseKey`/batch signatures + what issuance needs; SettingsScreen hex census + which Onyx & Ether tokens map. The Butler flake: which test state leaks into `TestButlerSalesContextRedactsPipelineAmounts`? | C1–C4 |

## 3. Phase B — polish the excellent

**B1 — Offers residuals.** (a) Reconcile the two expiry notions — one source of truth; an offer can never render expired inside the Quoted tab. (b) After MarkOfferWon: keep the toast, add a "View Order →" affordance to the created order (pattern #1). (c) Loss-reason list: "Successfully Closed" must not sit first under *Reason for Loss* — reorder/regroup. (d) Note placeholder says "opportunity" on an offer — fix. (e) The xl View modal's editable-header-inside-a-viewer: viewer is read-only; editing routes to the edit path (orchestrator's design call on the cheapest correct shape).
**AC:** expiry is one truth everywhere; Won hands off; no viewer that silently edits.

**B2 — Opportunity intake cleanups.** (a) Deprecated stages (New/Qualified/Proposal/On Hold) leave the filter list or cards show true stage — no filter that returns mislabeled cards. (b) Create-form fields the backend discards (Priority/received_date/due_date per A2) are wired through or removed — no fields that lie. (c) Edit modal sectioned (Commercial / Classification / Financials / Notes) — sectioning only, no field redesign. (d) Change-log load errors distinguished from empty and from unauthorized. (e) Cascade "Delete All" gets a stronger guard than single Delete (two-step; Article III.2 mass-destructive rung). (f) Line-item currency editable for manual (non-OCR) items.
**AC:** filters return what they say; every visible field persists; destroying N things is visibly harder than destroying one.

**B3 — Costing finishing.** (a) Remove/raise the 10-line hard cap (seeded RFQs must not silently truncate — if a real limit exists in the backend, surface it, don't swallow). (b) Standalone "Save Costing" (persists a revision without creating an offer). (c) Save-as-Offer's silent costing-persist failure → visible warning toast (offer saved, history not — say so). (d) Save-as-Offer silently updating an existing offer → confirm first.
**AC:** no silent truncation, no silent failure, no silent overwrite; costing can be saved as itself. Draft-recovery engine, suggested-vs-override pricing, and revision model untouched (keep-list).

**B4 — One line-item editor.** Converge Orders' manual line editor and Costing's editor on one shared component (Costing's is the reference — Article VI.3). The shared component lives with the other shared components; both screens consume it; feature gaps (per A4) are closed in the component, not forked around.
**AC:** one component, two consumers, zero behavior regressions on either screen (Costing's calculator parity is sacred).

**B5 — PO workspace: legal transitions only.** Replace the flat 8-button status grid with status-driven next-action buttons (pattern #2; the backend's legal-transition rules from A4 are the source of truth — mirror them, don't invent). Kill the `querySelectorAll` DOM-poking in favor of data-driven disabled state.
**AC:** a user can only click transitions the backend will accept; no DOM spelunking.

**B6 — GRN finishing.** Complete gated on QC status and hidden once complete (no re-completable GRNs); the never-rendered manual-GRN path: wire it deliberately or delete it (decide from A4, record the decision).
**AC:** Complete appears exactly when legal; no dead path remains.

**B7 — DN: the closing moment closes.** (a) Confirm Delivery captures the recipient name (real POD instead of hardcoded 'Auto-confirmed'); 'Signed' status becomes reachable. (b) When a confirm brings the order's remaining-to-deliver to zero: offer the handoff — "Order fully delivered — create invoice?" (pattern #1). (c) Serial-path create-then-patch collapses to a single create call (its patch failure currently only console.warns).
**AC:** delivery confirmation produces evidence; the sales loop's last handoff exists; one create path.

**B8 — Nav IA (orchestrator-owned; App.svelte + sidebar).** One shared nav list drives sidebar order, Alt+N shortcut order, and header titles (Settings included — no more "Workspace" fallback); "Relationships" renamed to speak the operator's language (Article I — "CRM" or "Customers & Suppliers", pick with the sidebar's width in mind); Deployment gets a deliberate reachable home or stays a Settings sub-tab by recorded decision.
**AC:** shortcut order = visual order; every destination has a correct title; no label a trader wouldn't say.

**B9 — FinancialDashboard refinements.** (a) Ratio health colors computed from real thresholds (A6), not hardcoded green. (b) Backend failure shows an explicit degraded state — never the silent hardcoded-FY2024 dataset under a selected year (that's a lie about money; treat as [H]). (c) Revenue/NetProfit KPIs drillable like Cash/AR. (d) Hardcoded per-year YoY prose: compute or remove.
**AC:** every color tells the truth; every number is either real or visibly absent; every KPI drills.

**B10 — Responsiveness polish (Article IV.1).** Orders list: replace `ListOrders(10000,0)` + per-row delivery-status calls (~175 round-trips per open, per the audit) with paging and a batched status call (add a read-only batch endpoint if none exists — pure read, gated like its singular sibling). Serial trace gets a "recently delivered" default instead of a blank page.
**AC:** Orders opens in one breath (measure round-trips before/after — report the numbers); serial trace greets instead of shrugs.

## 4. Phase C — ruled at the Spec-04 gate

**C1 — Conflict-tools move to sales admin (ruled).** The opportunity edit-conflict resolution + activity monitoring leave UserManagementScreen for a sales-admin home (per A7 — likely a SalesHub admin tab or equivalent), permission-gated as today or tighter. UserManagementScreen keeps only user/role/access administration.
**AC:** sales tooling lives with sales; the access screen is only about access.

**C2 — License issuance surfaced (ruled).** `GenerateLicenseKey`(/batch) gets an admin-gated affordance in the Access/Users admin surface — issue, then bind via the existing Access-tab flow. No new permission widening; server-side gates preserved.
**AC:** an admin can issue and bind a license without leaving the app.

**C3 — SettingsScreen tokenization (ruled — closes the Spec-04 deviation properly).** Replace SettingsScreen's ~102 raw hex with Onyx & Ether semantic tokens, visual-parity intent, exactly like the DashboardScreen precedent (Spec-02 C1). `var(--token, #fallback)` for values with no exact token.
**AC:** zero bare hex in SettingsScreen; visually equivalent; the ratified consistency-deviation is retired.

**C4 — Kill the Butler flake (test hygiene).** Two waves have documented `TestButlerSalesContextRedactsPipelineAmounts` failing on full-suite runs and passing in isolation — cross-test state contamination. Diagnose (A7) and fix the leak (isolate the shared state, not the symptom; do NOT just add `-count=1` or reorder tests). If the root cause is genuinely out of reach this wave, document the exact mechanism in the report.
**AC:** three consecutive full-suite runs green, or a precise root-cause writeup.

### Suggested coder batching (adjust from Phase A; monster-file rule applies)
- Coder 1: B1 + B2 (Offers + Opportunities — the sales monsters, one coder)
- Coder 2: B3 + B4 (Costing + the shared line editor; B4's Orders-side consumption lands after Coder 3's B10 Orders work is in — orchestrator sequences the OrdersScreen seam)
- Coder 3: B7 + B10 (DN + Orders perf)
- Coder 4: B5 + B6 (PO workspace + GRN)
- Coder 5: B9 + C3 (FinancialDashboard + SettingsScreen tokens)
- Coder 6: C1 + C2 (admin cluster)
- Orchestrator: B8 (nav seam), C4 (flake forensics), all seams + review

## 5. Hard boundaries

- **No sensory/brand work** — no sounds, no celebratory moments, no signature-timeline visuals; that wave is owner-reserved and next. Article IV.1–IV.2 responsiveness/motion basics ARE in scope (B10); IV.3's sound budget stays untouched.
- **Financial semantics: stop-and-report.** B9's threshold math is display logic; anything deeper stops.
- **Keep-lists binding — §4.1 sales above all:** pending-store handoffs, status-driven CTAs, suggested-vs-override pricing, the revision model, credit-override-with-reason, MarkOfferWon PO capture, integrity banners + cascade preview, costing draft recovery, won/lost edit-locking, order traceability chain. Also §4.2 inventory (Order→DN handoff, PO→GRN pre-select, remaining-qty defaulting, serial trace search) and §4.7 (dashboard drills, Operating Focus).
- **Explicitly deferred (do NOT do):** visa/permit tracking; GL opening-balance carry-forward; allocation capacity enforcement; CRLF normalization; WorkHub/CustomerDetailView wholesale decomposition (opportunistic extraction only).
- **No merge, no push, no tag.**

## 6. Definition of done + status report

Done = Phase A recorded; B1–B10 + C1–C4 shipped or explicitly skipped with reason; gates green on final commit (incl. one `-count=1` full `go test` run — three if C4 claims a fix); report written.

Write `FABLE_WAVE9_SPEC_05_REPORT.md`, commit it, and paste it verbatim as your final message (the established template: Phase A verdicts, per-item status+commits, decisions, constitution deviations, keep-list attestation, residue, owner questions). Severity honesty is law: an accurate red beats a false green.
