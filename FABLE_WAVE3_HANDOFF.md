# FABLE WAVE 3 HANDOFF — Make the Proof the Law

**From:** Claude Fable 5, the instance that executed Waves 1–2
**To:** the Fable 5 instance executing Wave 3
**Authorized by:** Sarat (the Commander) — autonomous execution, total
methodological freedom, clarifications welcome at any stage but not required.

This spec is different from the last two: it was written by a model that has
already worked this codebase, for a model with the same capabilities and none
of the context. Everything in here is either (a) a mission, or (b) context I
paid for that you shouldn't have to re-derive. Trust the pointers; verify the
code (things drift — `git log` is your friend).

---

## 0. Where you are standing

AsymmFlow is an offline-first, single-binary ERP **substrate** (Go + Wails
v2.11 + Svelte, pure-Go SQLite via ncruces — CGO is banned). The thesis under
test since Wave 1: **a vertical is configuration plus a thin domain package;
everything generic lives in engines; the kernel enforces the invariants
every vertical inherits.**

State of the evidence after Wave 2 (branch
`feat/fable-wave2-composition-proof`, ~85% thesis-proven — the honest audit
is in `docs/FABLE_WAVE2_PROGRESS.md`):

- **The substrate is real.** `pkg/kernel/{money,actor,approval,workflow,
  policy,evidence,text}` (pure, dependency-free; `actor.CanApprove` is the
  AI-authority boundary), `pkg/overlay` (deployment identity incl.
  `JurisdictionCode()`), `pkg/infra/events` (in-memory bus),
  `pkg/compliance` (TaxEngine registry + event hook; BH/IN/SA engines),
  `pkg/compliance/saudi` (full ZATCA Phase 2: UBL 2.1, secp256k1 XAdES,
  QR TLV, Fatoora client), `pkg/documents/numbering`,
  `pkg/finance/settlement`, `pkg/infra/db` (backup/restore),
  `pkg/infra/auth` (PIN lock).
- **The composition proof BOOTS.** `overlays/hospitality` +
  `cmd/hospitality` is a Saudi café POS that runs a full business day
  end-to-end using ONLY substrate engines — `go run ./cmd/hospitality`
  exits 0. Read it first; it is the pattern Wave 3 spreads.
- **The contradiction you exist to resolve:** the ORIGINAL trading vertical
  ignores the substrate it lives next to. Its composition root is
  `startup()` in `app.go` (~line 260–1208): ~90 hardcoded AutoMigrate
  models, unconditional trading seeds, hardcoded compliance registration
  (`compliance_bindings.go`), RBAC vocabulary baked in, ~140 root-level
  `package main` files wrapping one giant App. Hospitality proves a NEW
  vertical composes cleanly; nothing yet proves the OLD one can be brought
  onto the same law. That is Wave 3's headline.

Read in this order before writing any code:
`CLAUDE.md` → `docs/FABLE_WAVE2_PROGRESS.md` → `docs/FABLE_WAVE2_DECISIONS.md`
(the `[Mirror]` notes were written for you) → `cmd/hospitality/main.go` →
`app.go` `startup()` → `docs/FABLE_WAVE1_DECISIONS.md` if you need deeper
history.

---

## 1. Mission A (headline): decompose the trading composition root

**Goal.** The trading vertical boots through the same shape hospitality
does: an explicit composition root that wires overlay → database → bus →
compliance → engines → domain services, where the model set, seeds, and
compliance registration are driven by configuration, not hardcoded in
`startup()`.

**Non-goal.** A rewrite. The Wails v2 app is a shipping product (v2.3.0 in
pilot). This is a **strangler**: behavior-identical extraction, staged, with
the app booting green after every stage.

Staged milestones — each one independently committable, each one falsifiable:

- **A.1 — Extract the wiring seam.** Create `pkg/runtime/composition` (name
  yours to choose) holding a `CompositionRoot`-style builder: overlay load,
  DSN/pragma construction, bus + compliance registry + hook wiring. `app.go
  startup()` DELEGATES to it; `cmd/hospitality` migrates onto it too, so one
  seam serves both verticals. Falsifier: both binaries boot; `wails build`
  unaffected.
- **A.2 — Model registry.** The ~90-model AutoMigrate list becomes a
  registered model-set (trading registers trading models; hospitality
  already has its own). Byte-identical migration behavior — pin with a
  schema-dump comparison test before/after (SQLite `sqlite_master` is
  enough).
- **A.3 — Seeds behind the overlay.** Unconditional trading seeds
  (divisions, RBAC roles, demo rows) become seed functions selected by the
  overlay/vertical, preserving current default behavior exactly for
  existing deployments (default overlay → same seeds as today).
- **A.4 — Compliance registration is already config-shaped** (registry +
  overlay jurisdiction); fold `compliance_bindings.go`'s hardcoded
  registration into the composition root so there is exactly ONE place
  engines get registered.
- **A.5 (stretch, judgment call) — First domain-service peel.** Pick ONE
  root-level service cluster (candidate: delete-approval or GRN numbering,
  both small and already engine-adjacent) and move it behind a `pkg/` seam
  the way Wave 2 did numbering. Do NOT attempt the 1229-method App fan-out;
  that is Wave 4+ material.

Success statement for the progress audit: *"the trading app and the
hospitality app boot through the same composition seam, and the diff to
`startup()` is deletions and delegations only."*

## 2. Mission B: retire the A.1 engine residue

Two core, two stretch. All four are mapped with file:line pointers in the
Wave 2 study digest's public traces — re-derive quickly with grep; the
shapes are:

- **B.1 (core) — Approval routing onto `pkg/kernel/approval`.** Three
  mechanisms exist today: `assessCostingRisk`
  (`app_costing_exports_surface.go`, overlay thresholds),
  `delete_approval_service.go` (guard/request/perform), and the kernel
  package nothing uses. Promote a routing engine that expresses both
  existing flows on kernel vocabulary; rewire behavior-identically; the
  kernel's actor/authority checks must gate every approval transition
  (agents can NEVER approve — enforce with tests, both flows).
- **B.2 (core) — Audit convergence.** Two audit systems: live
  `infra.AuditLog` + `logAudit` (`app_auth_rbac.go` — note resourceID and
  description are currently DROPPED; fix that, it's a bug not a feature)
  and a dead second system in `security_enhancements.go`. Converge on one
  engine-backed path; delete the dead one LOUDLY (decision-log entry, not
  silence).
- **B.3 (stretch) — Excel `findColumn` de-triplication** into
  `pkg/documents/excel` (a dead started extraction already sits there —
  finish or replace it).
- **B.4 (stretch) — PDF de-contamination:** `pkg/engines/pdf_generator.go`
  carries hardcoded InvoiceData/Bahrain addresses; three generator paths
  exist. Minimum honest version: make the engine overlay-driven and port
  ONE consumer; full unification is residue.

**Explicitly out of scope, still:** Offer NN-YY numbering
(`app_sales_pipeline.go` ~3377) — coupled to OneDrive folder matching;
stop-and-ask territory. Leave it unless the Commander says otherwise.

## 3. Mission C: ZATCA hardening + hospitality graduation

Wave 2's ZATCA module is production-grade to a documented boundary. Move
the boundary:

- **C.1 — Validate against ZATCA's official sample set.** Fetch the ZATCA
  SDK samples (public; the research notes in
  `docs/research/ZATCA_PHASE2_RESEARCH.md` carry the pointers and the two
  open ❓ flags). Resolve: (a) the exact ds:Reference XPath transform
  strings; (b) QR tag-9 semantics for standard invoices. If our emitted
  XML/hash/QR disagrees with the samples, the samples win — fix and pin
  with golden tests. If sample acquisition needs a portal login, say so in
  the progress doc and mark the flag as still-open rather than guessing.
- **C.2 — CSR generation.** Onboarding needs a ZATCA-profile CSR
  (secp256k1, ZATCA-specific subject/extensions incl. the EGS serial and
  title flags). We can already build certs manually
  (`saudi.NewSelfSignedCertificate`); a `GenerateCSR` sibling completes the
  offline half of onboarding. Stdlib will refuse the curve — you'll be
  hand-assembling ASN.1 again; `crypto.go` shows every trick (RawValue
  element walks, SET re-tagging, UTF8String not PrintableString).
- **C.3 — Hospitality credit notes.** The ZATCA module supports credit
  notes (TypeCode 381, BillingReference, InstructionNote); the vertical
  doesn't expose them. Add a refund/credit-note flow to
  `overlays/hospitality` (manager-PIN + authority gated, chains ICV/PIH
  correctly, negative settlement impact lands in day close). This also
  retires the "proof vertical is happy-path-only" critique.

## 4. Mission D: the mirror (continuous, not a phase)

- `docs/FABLE_WAVE3_DECISIONS.md` — every consequential decision, with the
  `[Mirror]` annotation discipline (what would a lesser model need to be
  told). Number entries W3-D1….
- `docs/FABLE_WAVE3_PROGRESS.md` — measured timeline from the git log,
  honest thesis %, residue list for Wave 4.
- **No agentic-profiles step this wave** (Commander's call — Wave 2's
  profiles stand).
- If Mission A teaches you something that changes kernel design conclusions,
  NOW you have the second data point Wave 2 lacked: writing
  `docs/KERNEL_V2_DRAFT.md` is authorized (optional, evidence-driven).

---

## 5. Invariants (unchanged, non-negotiable)

1. No secrets in source. No real client data — synthetic canon only
   (`SYNTHETIC_IDENTITY.md`); never reintroduce real company names, tax
   IDs, bank details.
2. No CGO. ncruces SQLite stays.
3. Kernel purity: no domain vocabulary in `pkg/kernel`.
4. AI-authority boundary in product AND process: agents
   inspect/draft/recommend; only deterministic services approve, post,
   persist, delete.
5. Financial semantics are sacred: rounding, posting order, tax behavior,
   sequence formats = stop-and-ask with exact before/after numbers.
6. Green at every commit: `go build ./...` + tests. (Build needs
   `frontend/dist` present — it's go:embed'ed.)
7. No silent deletions. Delegate or record in the residue.
8. No Wails v3 migration, no `packages/*` design-system changes.

## 6. Working agreements

- Branch `feat/fable-wave3-<slug>` off the merged mainline the Commander
  gives you (ask which base if unclear — Wave 2's branch may or may not be
  merged when you start; do NOT assume).
- Small coherent commits, one capability each; decision-log entry per
  consequential choice; commit with `git commit -F <file>` (see §7).
- Reference codebases from prior waves (`CS-Invoice`,
  `PP_Killer`) remain READ-ONLY and their study notes
  must NOT enter this public repo. You likely won't need them this wave.
- Wave ends green with the progress audit written.

## 7. Lessons you inherit (paid for in Waves 1–2 — do not re-buy)

- **Boot, don't just build.** Every integration bug Wave 2 found surfaced
  in the first minute of RUNNING the composition, never statically. After
  each Mission A stage: boot both binaries.
- **Extraction = free audit.** Test the extracted engine harder than the
  original; when the original fails your test (it happened — numbering
  deadlocked), fix in the engine and log it.
- **SQLite discipline:** SELECT-FOR-UPDATE is a no-op; make transactions
  writers from their FIRST statement or they deadlock under concurrency.
  ncruces DSN pragmas: `?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)`.
  `:memory:` does not share across pooled connections — use a file-backed
  temp DB in concurrency tests, and close the GORM `sql.DB` pool in
  `t.Cleanup` or Windows `t.TempDir()` teardown fails.
- **Date bombs:** never fixed calendar dates + relative-period semantics in
  tests; derive from `time.Now().UTC()` and check period-boundary survival.
- **PowerShell 5.1 mangles quoted commit messages** — always
  `git commit -F <file>`. `2>$null` redirects can fake failures; capture
  output properly.
- **encoding/asn1 struct tags don't compose with RawValue** for
  context-specific elements — walk elements manually. X.509 strings:
  UTF8String, not PrintableString (real names have accents).
- **The compliance hook is async** (goroutine per event) — tests poll
  `RecentValidations` with a timeout, never assert immediately.
- **Root files are call sites, not precedent.** Architectural style comes
  from `pkg/` and `overlays/hospitality`, never from root `package main`.
- **Events carry the NET taxable base** in `Amount`; every compliance
  engine validates TaxAmount against it.
- **Sequencing by risk-retirement** beats mission order: find the cheapest
  step that would falsify the plan and run it first. For Mission A that is
  A.1 (the seam) — if the seam can't serve both verticals, you want to know
  in hour one.

## 8. Stop-and-ask registry

Message the Commander before: anything touching rounding/tax/posting/
sequence formats beyond behavior-identical delegation; Offer NN-YY; any
schema migration that rewrites existing rows; deleting anything that has
live callers; ZATCA sample-set conclusions that would change emitted bytes
for ALREADY-ISSUED invoice shapes.

## 9. Definition of done

- Trading + hospitality boot through one composition seam (A.1–A.4), stages
  committed separately, `startup()` diff is deletions/delegations.
- Approval routing on the kernel; audit converged; agents provably refused
  in both approval flows (B.1–B.2). Stretch B.3/B.4 as time allows.
- ZATCA sample verdict recorded (confirmed or honestly still-open); CSR
  generation; hospitality credit notes with chain + settlement integration
  (C.1–C.3).
- Decision log + progress audit with honest thesis % and Wave 4 residue.
- `go build ./...` + full `go test ./...` green; both demo binaries exit 0.

Build → Test → **Boot** → Ship. Measure, don't estimate.

— Fable 5 (Wave 2 instance), 2026-07-03
