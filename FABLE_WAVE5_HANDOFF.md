# FABLE WAVE 5 HANDOFF — Peel by the Map

Written 2026-07-04 by the Fable 5 instance that ran Wave 4 ("Shrink the
God"), for the fresh instance that will run Wave 5. You inherit a measured
map, a proven peel shape, and a working two-vertical substrate. Your job is
to execute against the map — not to re-derive it — and to keep every
financial number exactly where it was.

The Commander (Sarat) is at his terminal for the duration. Ask when a
decision is his; do not ask when the map already answers it.

## 0. Where you are standing

AsymmFlow: offline-first single-binary ERP substrate. Go + Wails v2.11 +
Svelte 5, pure-Go SQLite (ncruces — CGO is banned, keep it banned). Layer
law: `pure kernel → engines → overlay → vertical`. Two verticals boot
through ONE composition seam (`pkg/runtime/composition`): the trading app
(package main — the thick one) and hospitality (`overlays/hospitality` +
`cmd/hospitality` — the thin proof vertical, now with full AND partial
ZATCA credit notes on one ICV/PIH chain).

Branch discipline so far: wave branches off main, fast-forwarded into main
at the next wave's start. Wave 4 lives on `feat/fable-wave4-app-fanout`
(8 commits, e321d8a → 11cc54e). **Confirm with the Commander that Wave 4 is
merged to main before you branch** (`git log --oneline -3 main` should show
11cc54e or later). Branch: `feat/fable-wave5-<slug>`.

REQUIRED READING, in order (≈30 min, pays for itself):
1. `docs/FABLE_WAVE4_PROGRESS.md` — what just happened + this wave's residue.
2. `docs/FABLE_WAVE4_DECISIONS.md` — W4-D1 (the reference peel shape) and
   W4-D9 (the fan-out map) are your working documents; W4-D2..D8 stop you
   from re-buying paid-for lessons.
3. `docs/FABLE_WAVE3_DECISIONS.md` + `docs/FABLE_WAVE3_PROGRESS.md` — the
   composition seam, approvals-on-kernel, ZATCA state.
4. `CLAUDE.md` + `SYNTHETIC_IDENTITY.md` — invariants and the synthetic canon.
5. Skim `pkg/infra/deletion/deletion.go` — this IS the peel shape. Every
   Mission A extraction should look like it.

The thesis you are advancing: "a vertical is configuration plus a thin
domain package." Hospitality proves it; trading's 1230-method App
(~128k LOC across 218 root files) is the remaining counterexample. Wave 4
put it at ~93%. You move it by MOVING LOGIC INWARD, never by reshuffling
method counts.

## 1. Mission A (headline): peel the cheap seams, then payroll

The map (W4-D9) is explicit. Execute in this order — it is
risk-retirement-sorted, cheapest falsification first:

**A.1 — The cheap-seams queue.** Four self-contained extractions, each a
small coherent commit, each following the W4-D1 shape (aggregate + logic
inward; host behind narrow ports; root keeps a type alias + thin App
delegates; Wails façade signatures never change):
  - Serial numbers → `pkg/crm/fulfillment` (pairs with the GRN flow that
    already lives at that seam).
  - Cheque register → `pkg/finance/*`.
  - FX revaluation → `pkg/finance/fx` (mind invariant 5 — revaluation IS
    financial arithmetic; golden the numbers before you move them).
  - Assets/device CRUD → `pkg/infra/*`.
  Then, if momentum allows: the CRM contract body-move (finish the cleanest
  existing peel as the exemplar).

**A.2 — Payroll golden tests, then the payroll peel.** Payroll
(payroll_service.go, 33 methods, 53× a.db) is the best next FULL-domain
peel, but its accrual/posting journal is sacred financial arithmetic.
Sequence is non-negotiable: FIRST write golden tests over
`postPayrollAccrualJournal` + run generation totals (representative
fixtures, exact expected numbers, committed green against the UNTOUCHED
code); ONLY THEN extract to `pkg/hr/payroll` (or `pkg/finance/payroll` —
your call, record it). The approval gate is already on the kernel (W4 A.3);
identity/notification/posting stay behind ports per the standing rule.

**A.3 — Hub ports, opportunistically.** Auth/RBAC and
collaboration/notifications are HUBS: ports, never relocation (W4-D9).
Every peel in A.1/A.2 that needs them adds to the same small port
vocabulary — reuse the deletion peel's Identity/Notifier port shapes
instead of minting new ones per package.

What Mission A is NOT: the sales-pipeline surfaces (app_sales_pipeline.go,
app_order_customer_surface.go, app_setup_documents_surface.go) and
OneDrive/ETL. They are mapped EXPENSIVE. Do not open them this wave.

## 2. Mission B: session lifecycle, honestly this time

Wave 4 deleted the write-only SessionManager (W4-D3). The real system is
the DB-backed AuthManager (auth_session.go). Mission B wires interactive
session-inactivity enforcement THROUGH AuthManager as a deliberate,
tested security change:

- Decide the policy with the Commander first (one AskUserQuestion: timeout
  duration + what "activity" means + what the user sees on expiry). This is
  UX-visible; it is his call, not yours.
- Enforcement server-side (bound-endpoint side), not just frontend timers.
- Tests: expiry actually blocks a bound call; activity actually extends;
  logout invalidates. The W4-D3 mirror rule applies — a security component
  is only real if something READS it; make the read side the test.

This connects to the standing security-dogfood plan (CIA audit); keep the
change small and composable rather than building the whole identity story.

## 3. Mission C: hospitality — bill split (and print queue if it falls out)

The proof vertical keeps graduating toward pilot-usable:

**C.1 — Bill split.** Split one open session's lines across N invoices
(by line assignment; equal-split by amount is NOT a goal — quantity/line
assignment only, so document arithmetic stays per-line exact). Each split
invoice is its own ZATCA document on the shared ICV/PIH chain; payments and
refunds compose unchanged (partial refunds already work per invoice). Gates:
closing each split still requires kernel CanApprove; agents never issue.
Watch the rounding seam: per-line VAT rounding means split invoices' totals
sum exactly to what one big invoice would carry ONLY if lines are assigned
whole — enforce that (same philosophy as W4-D6: refuse, never adjust).

**C.2 — Print queue (stretch).** A minimal spooler: kitchen tickets and
invoices enqueue print jobs (table + status + payload reference); a worker
marks printed/failed; no actual printer driver — the seam is the point.
Skip without guilt if C.1 + Mission A eat the wave.

Deliberately NOT this wave: exchange/replacement flows (credit note + new
invoice as one gesture) — needs a design conversation first.

## 4. Mission D: the mirror (continuous, not a phase)

Same discipline as Waves 3–4:
- `docs/FABLE_WAVE5_DECISIONS.md` — W5-D1…, **[Mirror]** paragraphs for
  what generalizes. Write each entry WHEN you decide, not at the end.
- `docs/FABLE_WAVE5_PROGRESS.md` at wave end — measured timeline from git
  log, honest mission status, honest thesis %, Wave 6 residue.
- Honest accounting rule from W4: the App shrinks by LOGIC moved, not
  method count (delegates stay). Say so with numbers (e.g. LOC moved into
  pkg/, files whose free functions emptied).

## 5. Invariants (unchanged, non-negotiable)

1. **No secrets in source.** Env / in-app settings only.
2. **No real client data.** Synthetic canon only (SYNTHETIC_IDENTITY.md).
   Never reintroduce real company names, tax IDs, bank details, people.
3. **No CGO.** ncruces SQLite stays.
4. **AI-authority boundary.** Agents inspect/explain/draft/recommend; only
   deterministic services approve/post/persist/delete. Every new approve or
   post flow gates through `pkg/approvals` + kernel actors (three flows do
   already: costing, delete approval, payroll).
5. **Financial semantics are sacred.** Rounding, posting order, tax
   behavior, sequence formats: stop-and-ask, never a judgment call. Golden
   the numbers BEFORE touching code that produces them (A.2 rule).
6. **Green at every commit.** `go build ./...` + relevant tests; full
   `go test ./...` before each commit that touches shared code.
7. **No silent deletions.** Dead code dies LOUDLY with a signpost NOTE
   (see W3 AuditEvent, W4 SessionManager for the pattern).
8. **Kernel purity.** No domain vocabulary in `pkg/kernel`. (Adding a
   generic status synonym to `pkg/approvals` is fine — it's an engine.)
9. No Wails v3 migration. No `packages/*` changes.

## 6. Working agreements

- Branch `feat/fable-wave5-<slug>` off main (after confirming the Wave 4
  merge). Small coherent commits; end commit messages with the Claude
  co-author line.
- **PowerShell 5.1 lies about exit codes** when stderr is redirected: run
  builds/tests through the Bash tool. `git commit -F <file>` always
  (PowerShell mangles quoted messages); write the message file via a bash
  heredoc.
- `go build ./...` needs `frontend/dist` (go:embed) — it exists; don't
  delete it.
- Wails binding regeneration happens at wave end via `wails build -clean`;
  commit regenerated bindings as a separate chore commit. If a peel moves a
  bound model into pkg/, the frontend namespace changes (e.g.
  `main.DeleteApprovalRequest` → `deletion.Request` in W4) — grep
  `frontend/src` for the old name and fix the references in the same chore
  commit; `cd frontend && npx svelte-check --tsconfig ./tsconfig.json`
  must stay at 0 errors.
- Windows test hygiene: close GORM pools in test cleanup
  (`sqlDB, _ := db.DB(); t.Cleanup(func(){ sqlDB.Close() })`) or TempDir
  removal fails with file-lock errors.
- Root test suite: ~130–180s per run. One unexplained flake was observed
  once in Wave 4 (FAIL, then green twice with -count=1; no log captured).
  If you hit it, IMMEDIATELY re-run with `-v` teed to a file and record
  the failing test in the decisions doc before anything else.
- Reference codebases `CS-Invoice` and
  `PP_Killer` are READ-ONLY; study notes must NOT enter
  this public repo.
- The task list + an early Explore subagent for any survey work — but note
  the W4 lesson: subagents can be slow/silent; don't block on one, measure
  the cheap version yourself in parallel.

## 7. Lessons you inherit (paid for in Waves 1–4 — do not re-buy)

- **The peel shape (W4-D1):** aggregate + workflow inward; host behind
  identity/notifier/executor-style ports; root type alias keeps schema,
  JSON, registry and bindings; existing end-to-end tests passing untouched
  IS the behavior-identity proof. The trampoline idiom (generic Service +
  closures back into root) is NOT extraction — don't add more of it.
- **Straggler = live bug (W4-D2):** when one call site never got a
  migration the others did, expect the original defect still live there.
- **Trace the READ side of security components (W4-D3)** before trusting
  or extending them.
- **Re-measure residue lines before executing them (W4-D4):** "unify the
  three X" written from altitude may dissolve on the ground; the honest
  deliverable is sometimes a corrected map and a recorded NO.
- **Money split across documents (W4-D6):** enforce the invariant on the
  SUM at issuance; refuse loudly; never adjust a signed document's numbers.
- **ncruces DSN**: only `?_pragma=name(value)` works; mattn-style params
  are silently ignored (this bug ran the pilot in DELETE journal mode for
  months — W3).
- **GORM golden tests**: constraints emit in map order — normalize exactly
  the nondeterministic part (see trading_models_schema_test.go).
- **Compliance hook is async** — tests poll RecentValidations with a
  timeout (pattern in overlays/hospitality/partialrefund_test.go).

## 8. Stop-and-ask registry

Ask the Commander BEFORE acting on any of these:
- Offer NN-YY numbering (`app_sales_pipeline.go` ~3377) — OneDrive folder
  coupling; standing stop-and-ask, untouched three waves running.
- Any rounding / posting-order / tax-behavior / sequence-format change
  beyond behavior-identical (payroll journal especially — see A.2).
- Schema migrations that rewrite existing rows (adding tables/indexes with
  explicit legacy-index retirement, as in W4 C.1, is fine).
- Deleting anything with live callers.
- Session-timeout policy (Mission B — explicitly his decision).
- Anything that changes the appearance of customer-facing financial
  documents (the PDF unification stays parked for exactly this reason).
- ZATCA: any change to bytes of already-issued invoice shapes. The sandbox
  round-trip stays DEFERRED unless the Commander volunteers portal OTPs +
  JDK 11–14; if he does, `pkg/compliance/saudi` (api client + GenerateCSR)
  is ready and `docs/research/ZATCA_PHASE2_RESEARCH.md` has the state.

## 9. Definition of done

Wave 5 ends green with the progress audit written:
- `go build ./...` clean; full `go test ./...` green (run via bash).
- `go run ./cmd/hospitality` exits 0 — and now demos bill split if C.1
  landed.
- `wails build -clean` exit 0, AsymmFlow.exe produced; svelte-check 0
  errors; regenerated bindings committed as a chore commit.
- Every Mission A peel: existing app-level tests untouched and green, plus
  pkg-level tests for the moved logic.
- Payroll: golden tests exist and passed against BOTH the untouched and
  the peeled code (same numbers, provably).
- `docs/FABLE_WAVE5_DECISIONS.md` + `docs/FABLE_WAVE5_PROGRESS.md` written,
  with an honest thesis % and the Wave 6 residue.
- Branch `feat/fable-wave5-<slug>` awaiting the Commander's review; nothing
  merged by you.

Build → Test → Ship. Measure, don't estimate. And when the map and the
ground disagree — the ground wins, and the mirror records why. 🌊
