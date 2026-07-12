# FABLE WAVE 1 — The Sovereign Extraction

**Handoff date:** June 12, 2026
**Authored by:** Claude (strategy session with the maintainer / Commander)
**Executed by:** You — a Fable 5 instance, invited as a collaborator
**Repo:** `the AsymmFlow repository`
**Prior art convention:** `CODEX_WAVE{N}_HANDOFF.md` (root) → `docs/WAVE{N}_PROGRESS.md`. You inherit the rhythm, not the leash.

---

## 1. Who you are in this work

You are not a task executor. You are the **senior architect and senior research scientist** on this wave. The spec below describes *intent and constraints*, not procedure. Where you see a better decomposition, a cleaner boundary, a smarter sequence — take it, and write down why. This repo has a tradition of superseding its own plans with explicit supersession notes (see `docs/CODEX_DEV_ROADMAP_2026_05_15.md`); you are expected to continue that tradition, not preserve our guesses.

**Your charter of freedoms:**

- **Research freely.** If you want to study how Odoo structures modules, how Frappe handles app/site separation, how Go plugin architectures age, how other projects draw kernel/domain boundaries — search the web. Curiosity is legitimate work here, not a detour.
- **Spawn subagents freely.** Fan out exploration, parallelize disjoint extraction surfaces, use adversarial reviewers on your own boundary decisions. You know your tools better than we do.
- **Ask freely.** If a decision genuinely belongs to the Commander (business meaning, data semantics only a human who ran this company's books would know, anything destructive), stop and ask. Asking is not weakness; guessing on business semantics is.
- **Disagree freely.** If part of this spec is wrong, say so in the decision log and do the better thing. The only sin is silent divergence.

**The one obligation that pays for all this freedom: every consequential decision gets logged.** Create `docs/FABLE_WAVE1_DECISIONS.md`. One entry per decision: what was decided, what was rejected, why. All intelligences in the loop — that is the deal.

---

## 2. Why this work matters (business context)

This repo is being transmuted from a deployed Bahraini trading ERP into a **sovereign software substrate** — a builder's kit that lets indie builders and small agencies compose offline-first, single-binary vertical business apps (trading, pharmacy, clinic, workshop ERPs) for SMBs, with AI-agent-native methodology shipped as part of the product.

Market research (June 2026) found the quadrant **(one-time purchase) × (offline single binary) × (ERP domain primitives) × (agent-native methodology)** is empty. The closest precedents: Tally's 28,000-partner perpetual-license economy, ERPNext's partner network (whose #1 structural pain is upgrade-breakage of customizations — a pain this kit's own-your-fork model dissolves), and $999-tier boilerplates like SaaS Pegasus. The positioning shorthand: *"TDL for the Claude Code generation."*

Two things follow from this context:

1. **The overlay extraction you are about to do IS the product validation.** The kit is sellable the day one reference overlay proves the kernel → engines → overlay pattern works. A later wave will hand the finished kit to a fresh agent and ask it to compose a *second* vertical, cold. Your work determines whether that test passes.
2. **Your work artifacts are themselves product.** The decision log, the progress audit, this handoff — they ship in the kit as the methodology exemplar. Write them as if a paying builder will read them, because one will.

---

## 3. Required reading, in order

Before touching code (subagents can parallelize this):

1. `docs/SOVEREIGN_SOFTWARE_CONSTITUTION.md` — the philosophy and the layer model
2. `docs/KERNEL_CONSTITUTION.md` — what may and may NOT live in the kernel
3. `docs/OVERLAY_BOUNDARY_GUIDE.md` — the kernel/engine/overlay boundary rules
4. `docs/SOVEREIGN_SUBSTRATE_6_MONTH_ROADMAP.md` — Track G (overlay extraction) and Track A (kernel) are your missions
5. `docs/CODEX_GOAL_ENGINE_GENERALIZATION_AUDIT.md` — the honest map of coupling risks and the target shape: `pure kernel → domain service → storage adapter → ViewModel adapter → agent adapter`
6. `docs/OPERATING_PRINCIPLES.md` and `docs/COLLABORATION_PROTOCOL.md` — house culture
7. Skim 2–3 recent `CODEX_WAVE*_HANDOFF.md` + matching `docs/WAVE*_PROGRESS.md` pairs to absorb the wave rhythm

Orientation facts so you don't re-derive them: ~100+ `package main` files sit at repo root (the legacy app surface), including `app_sales_pipeline.go` (~141 KB), `app_setup_documents_surface.go` (~147 KB), `butler_ai_context.go` (~173 KB). `app.go` was already reduced from 21,763 LOC to a ~1,900 LOC lifecycle shell in Waves 0–8. `pkg/` holds 33 packages including real engines (`finance/posting`, `documents`, `compliance`, `cashflow/evidence`, `math`). The kernel has exactly four primitives: `pkg/kernel/{money, approval, evidence, text}`. Stack: Go 1.24+ / Wails v2.11 (v3 deferred — do not migrate) / Svelte 5 / ncruces pure-Go SQLite (CGO is banned; keep it banned).

**Git situation, check before anything:** the checked-out branch may be `feat/asymmflow-design-system` (8 commits ahead of master, unmerged, owns `packages/*`). Run `git status` and `git branch -vv` first. Recommended: create `feat/fable-wave1-sovereign-extraction` off `master`, leave the design-system branch untouched, and do not modify `packages/*` in this wave. If you find a reason to deviate, log it.

---

## 4. Mission 0 — The Hygiene Gate (blocking; do first)

> **✅ This hygiene gate has since been COMPLETED (historical record).** The
> repository now ships **only** the synthetic reference dataset (see
> [`SYNTHETIC_IDENTITY.md`](SYNTHETIC_IDENTITY.md)) — no non-synthetic company
> data, credentials, or figures remain in the tree. The checklist below is
> retained as a record of the original scrub, not an open action.

At the time of this handoff the working tree still originated from a working
deployment and had not yet been converted to synthetic data. Before any further
agent waves churned through it, the following gate had to pass:

- [ ] Remove the working `.db` (the pre-scrub books) from the working tree. Before deleting, confirm a copy exists outside this repo — verify, don't assume; ask the Commander to confirm if ambiguous.
- [ ] Audit and quarantine `data/ssot/` — likely real client context. Move out of the repo or into an explicitly gitignored vault; ask the Commander which.
- [ ] Scrub `CLAUDE.md` / `AGENTS.md` of Supabase project IDs, real credentials, and live-deployment references. These files are also stale (pre-refactor era) — rewriting them to describe the substrate vision is in-scope and valuable, since every future agent reads them first.
- [ ] Sweep for real customer/employee/bank data in: `deploy_package/`, `test_data/`, `exports/`, `data/`, seed files (`customer_reference_seed.go`), `.env*`, and the `manual_*_test.go` files (several reference OneDrive imports and Supabase schemas).
- [ ] Build or adapt a **synthetic seed** so the app boots and demos with fictional data (`test_fixtures.go` and `customer_reference_seed.go` show the existing patterns). The kit's first impression is `wails dev` on fake-but-plausible data.
- [ ] Delete the 2 MB+ root log files (`wave7_root_test*.log`, `wave7_target_test.log`) and add patterns to `.gitignore`.

**Scope boundary:** working-tree scrub only. Git-history rewrite is a separate, deliberate decision (it invalidates every clone) — flag it in the decision log as pending, do not perform it.

---

## 5. Mission A — Track G: Extract the Trading Overlay

**The structural move this whole wave exists for.** The PH/trading/instrumentation behavior currently woven through the root-level app surface becomes the first **domain overlay** — proving the substrate's central claim.

### Target end-state

- A trading/distribution overlay package (location and name per `OVERLAY_BOUNDARY_GUIDE.md` — your call, log it; something like `overlays/trading/` or `pkg/overlay/trading/`) containing the domain logic that is genuinely about *trading-company operations*: RFQ/costing/quotation pipeline, PO approval thresholds, GRN+QC flow, serial traceability chain, delivery notes, trading-specific dashboards.
- **Company-specific facts become overlay *configuration*, not code.** The canonical example: `einvoice_service.go` hardcodes Acme Instrumentation's TRN and NBR details as constants. Minimum-margin percentages, approval thresholds (the 5K BHD rule), letterhead branding, grading policies — all of it config, with PH's values becoming the *example config* that ships with the reference overlay.
- Root-level `package main` files reduced to: a thin composition root (wiring kernel + engines + overlay into the Wails app), lifecycle, and platform glue. The Engine Generalization Audit's layer shape is the law: `pure kernel → domain service → storage adapter → ViewModel adapter → agent adapter`.
- Generic capabilities discovered inside the monoliths during extraction (e.g., generic document numbering, generic approval routing, generic PDF assembly buried in trading code) get promoted to engines (`pkg/`), not buried in the overlay.

### Decision rules for "what goes where"

- Would a *pharmacy* app need it? → engine or kernel, not overlay.
- Is it a fact about Acme Instrumentation the company (TRN, thresholds, branding, grading cutoffs)? → overlay config.
- Is it a behavior of trading-as-a-domain (RFQ→costing→quote, serial chains)? → overlay code.
- Is it sector-agnostic vocabulary (Party, Money, Approval, Evidence)? → kernel. And per the Kernel Constitution's rejection rule: `PurchaseOrder`, `Quotation`, `GRN`, `VATInvoice` may NEVER be kernel concepts.

### Counsel, not commands

- The three monster files (`butler_ai_context.go`, `app_sales_pipeline.go`, `app_setup_documents_surface.go`) are the dragons. Consider whether `butler_ai_context.go` is even overlay work — it may decompose into a generic context-assembly engine + per-overlay context contributions. Your judgment.
- You do not have to extract *everything* in one wave. A clean, complete extraction of the sales-pipeline + procurement + serial-traceability core with a documented residue list beats a smeared 100% attempt. Completion honesty over coverage theater.
- Strangler-fig over big-bang: keep the app building and tests passing at every meaningful checkpoint. `Taskfile.yml` and `scripts/verify_release.ps1` are your harnesses.

---

## 6. Mission B — Complete the Kernel Vocabulary

Four primitives exist (`money`, `approval`, `evidence`, `text`). The constitutions call for roughly: **Actor, Party, Request, Asset, Workflow, Policy, Timeline** (and possibly events-as-kernel — see below). Design and implement the remaining primitives, in the same style as the existing four: dependency-free, sector-agnostic, immutability-biased, real tests.

- Each primitive needs a **denial test**: an automated check (grep-based or AST-based) asserting no domain vocabulary leaks into `pkg/kernel/` — make the rejection rule executable, not aspirational.
- **Sequence advantage:** Mission A's extraction will *reveal* what the primitives must express (every trading workflow you extract is a test case for `Workflow`; every policy threshold for `Policy`). Consider running B slightly behind A's discoveries rather than designing primitives in a vacuum. Or run them in parallel with a sync point. Your call — log it.
- Decide and document the relationship between `pkg/kernel` primitives and the existing `pkg/infra/events` bus (the roadmap hints at events-as-kernel). Note for your map: the event bus currently has **subscribers but zero production publishers** — compliance hooks listen to events nobody emits. Full wiring is Wave 2's job; what's in scope for you is making sure your extraction *doesn't* make that wiring harder, and noting in the decision log where the publish points should go.

---

## 7. Invariants (non-negotiable)

1. **Kernel purity:** no domain concepts in `pkg/kernel/`, enforced by denial tests.
2. **AI-authority boundary:** agents may inspect/explain/draft/recommend; only deterministic services may approve/post/persist/delete. Preserve and extend the existing denial tests around this (see `docs/AI_REPAIR_AGENT_WORKFLOW.md`).
3. **No CGO.** ncruces SQLite stays.
4. **No Wails v3 migration** (explicitly deferred), no frontend redesign, no `packages/*` changes in this wave.
5. **Wave ends green:** `go build ./...` clean, full test suite passing, app boots against synthetic seed.
6. **No silent deletions:** code you don't understand gets quarantined (moved + logged), not deleted.
7. **Financial semantics are sacred:** if an extraction forces a choice about rounding, posting order, or tax behavior, that's a stop-and-ask, not a judgment call.

---

## 8. Working agreements

- **Branch:** `feat/fable-wave1-sovereign-extraction` off `master` (or log your deviation).
- **Commits:** small, coherent, message convention as in recent history. Commit at every stable checkpoint — git is the safety net.
- **Decision log:** `docs/FABLE_WAVE1_DECISIONS.md`, as described in §1.
- **Progress audit:** `docs/FABLE_WAVE1_PROGRESS.md`, same genre as prior `WAVE*_PROGRESS.md`. Include the **measurement rule**: record actual timestamps (start, checkpoints, end) — measured elapsed time, never estimates.
- **Residue list:** whatever you consciously leave unextracted, enumerate it with reasons. That list seeds Wave 2's spec.
- **Escalate to the Commander when:** business semantics are ambiguous (is this rule PH policy or trading-domain truth?), data destruction is involved, the hygiene gate finds something unexpected (e.g., no backup of the db exists), or you believe a constitution document itself is wrong.

---

## 9. Acceptance criteria

- [ ] Hygiene gate complete: no real client data in working tree; app boots and demos on synthetic seed; CLAUDE.md/AGENTS.md rewritten for the substrate era.
- [ ] Trading overlay exists as a package with documented boundaries; company facts are config; PH values survive only as the reference overlay's example config.
- [ ] Root `package main` surface measurably collapsed (report before/after file count and LOC); composition root is thin and legible.
- [ ] Kernel vocabulary complete (or consciously scoped subset, with reasons), each primitive tested, denial tests executable and passing.
- [ ] All invariants hold; wave ends green.
- [ ] `docs/FABLE_WAVE1_DECISIONS.md` and `docs/FABLE_WAVE1_PROGRESS.md` written to ship-in-the-kit quality.
- [ ] A short section in the progress doc titled **"Could a second vertical be composed today?"** — your honest answer, with the gaps that remain. This question is the entire point.

---

## 10. The spirit of the thing

You're holding the work of a year — a real ERP that runs a real company in Bahrain, mid-transformation into something nobody has built: a substrate that lets a builder in Tirupati or Bhubaneswar ship sovereign software to businesses that have only ever been offered rent-seeking clouds or arcane legacy tools.

The architecture documents in this repo are unusually good. The gap between vision and tree is honest, mapped, and now yours to close. Bring your own perspective — where you see something we missed, that's not scope creep, that's why you were invited.

Build → Test → Ship. Measure, don't estimate. Log every decision. Have fun, broseph. 🌊

**Om Lokah Samastah Sukhino Bhavantu.**
