# FABLE WAVE 6 HANDOFF — Empty the God's Pockets

Written 2026-07-04 by the Fable 5 instance that ran Wave 5 ("Peel by the
Map"), for the fresh instance that will run Wave 6. You inherit a
substrate where every CHEAP seam has already been peeled — what remains
in the trading root is either genuinely hub, genuinely expensive, or
reachable only through the two work-lists this handoff hands you. Your
job is to keep moving logic inward along those lists — and to keep every
financial number exactly where it was.

The Commander (Sarat) is available. Ask when a decision is his; do not
ask when this document already answers it. Two decisions he has ALREADY
made for this wave (do not re-ask): the payroll negative-net fix is
**refuse-to-generate** (Mission B below), and the wave headline is
**keep shrinking the god** (Mission A).

## 0. Where you are standing

AsymmFlow: offline-first single-binary ERP substrate. Go + Wails v2.11 +
Svelte 5, pure-Go SQLite (ncruces — CGO is banned, keep it banned). Layer
law: `pure kernel → engines → overlay → vertical`. Two verticals boot
through ONE composition seam (`pkg/runtime/composition`): the trading app
(package main — still thick, but ~3,450 LOC of logic lighter after Wave
5) and hospitality (`overlays/hospitality` + `cmd/hospitality` — the thin
proof vertical, now with bill split, partial+full ZATCA credit notes on
one ICV/PIH chain, and a print-spooler seam).

Wave 5 IS merged: `git log --oneline -3 main` should show b894b5a
("docs(wave5): Mission D…") or later. Branch `feat/fable-wave6-<slug>`
directly off main. Small coherent commits, fast-forward merge at the
next wave's start — same discipline as Waves 2–5.

REQUIRED READING, in order (≈30 min, pays for itself):
1. `docs/FABLE_WAVE5_PROGRESS.md` — what just happened + this wave's residue.
2. `docs/FABLE_WAVE5_DECISIONS.md` — W5-D1..D7. W5-D3 (re-measure before
   executing) and W5-D5 (port the neighbors you'd drag, inline what
   already moved) are your working rules; the rest stop you re-buying
   paid-for lessons.
3. `docs/FABLE_WAVE4_DECISIONS.md` — W4-D1 (the reference peel shape) and
   W4-D9 (the fan-out map + the standing HUB rule: ports, never
   relocation).
4. `CLAUDE.md` + `SYNTHETIC_IDENTITY.md` — invariants and the synthetic canon.
5. Skim `pkg/finance/payroll/service.go` — the largest peel so far; note
   how the four ports are cut (identity, directory, events, expense
   bridge) and how posting moved INWARD because its models were already
   in pkg/finance. Every Wave 6 extraction should ask the same question:
   where does the DATA already live?

The thesis you are advancing: "a vertical is configuration plus a thin
domain package." Hospitality proves it; trading's root (~1,230 App
methods, but audit by logic location, not method count) is the remaining
counterexample. Wave 5 put it at ~94%. You move it by MOVING LOGIC
INWARD, never by reshuffling method counts, and never by adding
trampolines (generic Service + closures back into root — W4-D1 named it,
don't add more).

## 1. Mission A (headline): the two work-lists, then ports for the pipeline

Execute in this order — risk-retirement-sorted, cheapest falsification
first. RE-MEASURE each line before executing it (W4-D4, W5-D3: the honest
peel boundary often cuts through the middle of a file, and residue lines
written from altitude sometimes dissolve on the ground).

**A.1 — Butler read paths.** The cheapest untouched seam in the W4-D9
map. Invariant 4 means Butler code paths can inspect/explain/draft but
never persist — so they carry no RBAC-mutation or notification
entanglement by construction. Survey `butler_ai_context.go` (57 methods /
~5,070 LOC), `app_butler_context.go`, `butler_grounded_fastpath.go`, and
the butler read surfaces; move the CONTEXT-BUILDING and READ logic into
`pkg/` (a `pkg/butler` or extensions of existing domain pkgs — decide
from what the data says, record it). Expect the W5-D3 pattern: some of
these files will have a hub-shaped half (session identity, event
emission) — thin delegates and ports for those, logic inward for the
rest. Model calls (Mistral) stay behind whatever seam already carries
them; do NOT couple pkg/ to an AI vendor.

**A.2 — Entity-by-entity delete extraction.** The deletion Executor
port's ~26-way dispatch (wired in Wave 4's `pkg/infra/deletion` peel) is
the explicit work-list: each case is one entity's delete logic living in
root. For each entity whose domain package already exists (customer,
product, invoice, PO, DN, GRN, serials, cheques, FX, payroll, contract…),
move the delete logic into that package and have the Executor case
delegate. `guardDeleteOrRequest` (13 call sites) stays root — it IS the
hub-facing guard. Work the list in whatever order the coupling measures
cheapest; a partial list honestly finished beats all 26 shuffled.
Remember W4-D2: any dispatch case that diverges from its siblings is
where a live bug still is — record what you find.

**A.3 — Sales-pipeline PORTS (preparation, not relocation).** The
expensive clusters (`app_sales_pipeline.go`, `app_order_customer_surface.go`,
`app_setup_documents_surface.go`) stay where they are this wave. What you
may do: stabilize the interfaces they'd need — extend the existing port
vocabulary (deletion's Identity/Notifier shapes, payroll's four ports)
rather than minting new ones — and, ONLY if measurement reveals a
genuinely self-contained sub-seam (a leaf with a.db-only coupling, like
the cheap seams were), peel that one leaf as a small coherent commit.
The Offer NN-YY numbering site (~app_sales_pipeline.go:3377) remains a
standing stop-and-ask — do not touch it even in passing.

What Mission A is NOT: relocating the hubs (auth/RBAC,
collaboration/notifications — ports, never relocation, W4-D9), the
OneDrive/ETL import machinery, or the PDF canvas unification (visual
sign-off project, still parked).

## 2. Mission B: the payroll refusal (Commander-decided)

Wave 5 observed and pinned — but did not change — this behavior: when an
employee's deductions exceed gross, the item's net clamps to 0 while the
accrual journal still debits full gross, so the journal does NOT balance
(debits ≠ credits). The Commander has decided the fix: **refuse to
generate**.

- `payroll.GenerateRun` refuses any run containing an item whose
  deductions exceed its gross, with a clear per-employee error naming the
  employee and the amounts (refuse the whole run — no partial
  generation, no silent skipping; same philosophy as W4-D6).
- The Wave 5 golden `TestPayrollRunGeneration_NegativeNetClampsToZero`
  pins the OLD behavior — it changes to pin the refusal. This is a
  deliberate, Commander-authorized golden change: say so in the commit
  message and in the decisions doc, because a silently edited golden is
  worse than none.
- Add a pkg-level test with fake ports for the refusal, and confirm the
  balanced-journal goldens still pass byte-identical (they never
  exercised the imbalance).
- Do NOT extend scope into capping or receivable-booking — those options
  were considered and not chosen.

## 3. Mission C: session lifecycle follow-ons (small, bounded)

From the Wave 5 residue, in priority order; each is small — stop when
they stop being small:

- **C.1 — Logout in the UI.** Wire a visible logout control to the
  existing `LogoutInteractiveSession` binding and return the UI to the
  login screen (the `auth:session-expired` handling in App.svelte shows
  the shape). No new backend behavior; svelte-check stays at 0 errors.
- **C.2 — Configurable timeout (stretch).** Read the interactive
  inactivity timeout from settings (env / in-app), default 30 minutes.
  Only if it lands cleanly on the existing Setting machinery; the
  hardcoded default is fine otherwise. Do not re-ask the Commander for a
  new default — 30 minutes stands.

Hospitality is deliberately QUIET this wave (Commander's headline call).
The exchange/replacement flow still needs its design conversation — if
the Commander volunteers it mid-wave, have that conversation and record
it in the decisions doc; do not build it unprompted.

## 4. Mission D: the mirror (continuous, not a phase)

Same discipline as Waves 3–5:
- `docs/FABLE_WAVE6_DECISIONS.md` — W6-D1…, **[Mirror]** paragraphs for
  what generalizes. Write each entry WHEN you decide, not at the end.
- `docs/FABLE_WAVE6_PROGRESS.md` at wave end — measured timeline from git
  log, honest mission status, honest thesis %, Wave 7 residue.
- Honest accounting rule (W4/W5): the App shrinks by LOGIC moved, not
  method count. Report LOC moved into pkg/ and which dispatch cases /
  butler files emptied; the Wave 5 audit shows the format.

## 5. Invariants (unchanged, non-negotiable)

1. **No secrets in source.** Env / in-app settings only.
2. **No real client data.** Synthetic canon only (SYNTHETIC_IDENTITY.md).
   Never reintroduce real company names, tax IDs, bank details, people.
3. **No CGO.** ncruces SQLite stays.
4. **AI-authority boundary.** Agents inspect/explain/draft/recommend;
   only deterministic services approve/post/persist/delete. Every new
   approve or post flow gates through `pkg/approvals` + kernel actors.
5. **Financial semantics are sacred.** Rounding, posting order, tax
   behavior, sequence formats: stop-and-ask, never a judgment call.
   Golden the numbers BEFORE touching code that produces them. (Mission
   B's semantics change is pre-authorized — the refusal option,
   exactly as specified, nothing adjacent.)
6. **Green at every commit.** `go build ./...` + relevant tests; full
   `go test ./...` before each commit that touches shared code.
7. **No silent deletions.** Dead code dies LOUDLY with a signpost NOTE.
8. **Kernel purity.** No domain vocabulary in `pkg/kernel`.
9. No Wails v3 migration. No `packages/*` changes.

## 6. Working agreements

- Branch `feat/fable-wave6-<slug>` off main (Wave 5 is already merged —
  verify b894b5a is in main's history, then go). Small coherent commits;
  end commit messages with the Claude co-author line.
- **PowerShell 5.1 lies about exit codes** when stderr is redirected: run
  builds/tests through the Bash tool. `git commit -F <file>` always;
  write the message file via a bash heredoc.
- `go build ./...` needs `frontend/dist` (go:embed) — it exists; don't
  delete it.
- Wails binding regeneration at wave end via `wails build -clean`;
  regenerated bindings are a separate chore commit. When a peel moves a
  bound model into pkg/, the frontend namespace changes — grep
  `frontend/src` for the old `main.X` names and fix them in the same
  chore commit (Wave 5 example: `frontend/src/lib/api/payroll.ts`);
  `cd frontend && npx svelte-check --tsconfig ./tsconfig.json` must stay
  at 0 errors.
- Windows test hygiene: close GORM pools in test cleanup
  (`sqlDB, _ := db.DB(); t.Cleanup(func(){ sqlDB.Close() })`) or TempDir
  removal fails with file-lock errors.
- Root test suite: ~130–180s per run. The Wave 4 one-time flake never
  recurred in Wave 5; if you hit an unexplained FAIL, IMMEDIATELY re-run
  with `-v` teed to a file and record the failing test in the decisions
  doc before anything else.
- Reference codebases `CS-Invoice` and
  `PP_Killer` are READ-ONLY; study notes must NOT enter
  this public repo.
- Subagents for survey work are fine, but don't block on one — measure
  the cheap version yourself in parallel (W4 lesson, still true).

## 7. Lessons you inherit (paid for in Waves 1–5 — do not re-buy)

- **The peel shape (W4-D1)** and its Wave 5 refinement (W5-D5): aggregate
  + logic inward; port the neighbors you'd otherwise drag, INLINE the
  ones whose models already moved. Port count is a cost, not a badge
  (W5-D1). Existing end-to-end tests passing untouched IS the
  behavior-identity proof.
- **A file's name is not its coupling profile (W5-D3):** grep for session
  mutation and hub helpers before executing a mapped extraction; the
  honest boundary often cuts through the middle of a file.
- **"Constructed but never called" (W5-D4):** audit extraction progress
  by where method bodies live, not by which packages appear in
  initServices. And parse-with-real-code-paths finds real bugs — the
  contract seeds had NEVER worked.
- **Golden-first for financial arithmetic (W5-D2):** exact-binary (or
  integer) fixtures make every assertion exact equality — no tolerance to
  argue about. Pin in a separate commit against untouched code.
- **Straggler = live bug (W4-D2):** a dispatch case that diverges from
  its migrated siblings is where the original defect still lives.
- **Enforce at the chokepoint (W5-D6):** middleware the calls already
  flow through beats a validator endpoints must remember to call.
- **Split children need their own key (W5-D7):** any query resolving
  children through a document's PARENT breaks the moment two documents
  share that parent. Stamp at issuance; give legacy rows an exact
  fallback.
- **Re-measure residue lines before executing (W4-D4):** the honest
  deliverable is sometimes a corrected map and a recorded NO.
- **ncruces DSN:** only `?_pragma=name(value)` works; mattn-style params
  are silently ignored.
- **GORM golden tests:** constraints emit in map order — normalize
  exactly the nondeterministic part.
- **Compliance hook is async** — tests poll RecentValidations with a
  timeout (pattern in overlays/hospitality/partialrefund_test.go).

## 8. Stop-and-ask registry

Ask the Commander BEFORE acting on any of these:
- Offer NN-YY numbering (`app_sales_pipeline.go` ~3377) — OneDrive folder
  coupling; standing stop-and-ask, untouched four waves running.
- Any rounding / posting-order / tax-behavior / sequence-format change
  beyond behavior-identical — EXCEPT the Mission B refusal, which is
  pre-authorized exactly as §2 specifies.
- Schema migrations that rewrite existing rows (adding tables/indexes
  with explicit legacy-index retirement is fine).
- Deleting anything with live callers.
- Anything that changes the appearance of customer-facing financial
  documents (the PDF unification stays parked for exactly this reason).
- ZATCA: any change to bytes of already-issued invoice shapes. The
  sandbox round-trip stays DEFERRED unless the Commander volunteers
  portal OTPs + JDK 11–14; if he does, `pkg/compliance/saudi` is ready
  and `docs/research/ZATCA_PHASE2_RESEARCH.md` has the state.
- Hospitality exchange/replacement flows — design conversation first,
  Commander-initiated.

## 9. Definition of done

Wave 6 ends green with the progress audit written:
- `go build ./...` clean; full `go test ./...` green (run via bash).
- `go run ./cmd/hospitality` exits 0 (unchanged behavior — hospitality is
  quiet this wave; it must STAY green).
- `wails build -clean` exit 0, AsymmFlow.exe produced; svelte-check 0
  errors; regenerated bindings committed as a chore commit with any
  frontend namespace fixes.
- Every Mission A extraction: existing app-level tests untouched and
  green, plus pkg-level tests for the moved logic.
- Mission B: the refusal implemented exactly as §2; the golden change
  explicit and explained; the balanced-journal goldens byte-identical.
- Mission C.1 landed (logout control); C.2 only if it stayed small.
- `docs/FABLE_WAVE6_DECISIONS.md` + `docs/FABLE_WAVE6_PROGRESS.md`
  written, with an honest thesis % and the Wave 7 residue.
- Branch `feat/fable-wave6-<slug>` awaiting the Commander's review;
  nothing merged by you.

Build → Test → Ship. Measure, don't estimate. And when the map and the
ground disagree — the ground wins, and the mirror records why. 🌊
