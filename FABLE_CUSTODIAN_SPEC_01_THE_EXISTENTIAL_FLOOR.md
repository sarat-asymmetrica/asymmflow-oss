# Custodian Wave 1 — The Existential Floor

**Status:** RATIFIED 2026-07-23 · **Orchestrator & primary gate:** Fable
**Coders:** Sonnet 5 agents, one per mission band, gated individually
**Owner:** the Commander · **Parent map:** `CUSTODIANS_LEDGER.md` (items C1, C2, C7)
**Branch:** `feat/fable-custodian-w1` off today's main
**Prior art (MANDATORY):**
1. `CUSTODIANS_LEDGER.md` — why these three items are Wave 1: they are the cheap,
   existential ones. None is a feature.
2. `pkg/infra/db/backup.go` + `backup_test.go` — the existing backup machinery; C1
   builds on what exists, it does not reinvent.
3. `field_crypto.go`, `hardware_id_keystore_windows.go`, `config.go` — the known key
   surfaces; CW1-0 completes this inventory, it does not assume it.
4. `mesh/kit/` data-folder layout + `mesh/docs/MESSENGER_DECISIONS.md` (room-key /
   true-deletion rulings) — the mesh side of the custody map.
5. Method law: [[verify-the-probe]] — every drill and every CI job must be able to
   report the opposite result; a restore drill that cannot detect a corrupt backup
   proves nothing.

---

## 0. Charter

Three deliverables, all custody, zero features:

- **C2 — the key custody map + rehearsed recovery.** After this wave, there exists a
  document that answers, for EVERY key in the system: where it lives, what it
  protects, who can recover it, and what dies with it — and the recovery procedure
  has been *executed once on a copy*, not just written.
- **C1 — the restore drill.** After this wave, there exists a drill harness + runbook,
  and the drill has been *run with a stopwatch*: backup artifact → working system with
  the data intact, plus a corrupt-backup negative control that the drill catches.
- **C7 — public CI.** After this wave, `.github/workflows/` carries the gate floor
  (Go build + vet + full test suite at minimum), proven by running every CI command
  verbatim locally; first green cloud run lands when the branch is pushed.

**Honesty boundary stated up front:** this wave runs on the dev machine against
*copies*. It proves the procedures and the instruments. What it CANNOT prove: a
restore on genuinely foreign hardware, and recovery by a person who is not the
steward. Both are recorded as residue for a later wave, not fudged here.

**Stop-and-report (owner decisions):** any change to encryption semantics or key
derivation (the wave MAPS keys, it does not touch crypto code); any backup-format
change; any CI step that would require repo secrets (the floor must run secretless —
if a test demands `ENCRYPTION_MASTER_KEY`-class material, that is a finding to
report, not a secret to upload); anything that would write to the live PH deployment
or any real backup medium.

## 1. Doctrine

- **Every instrument proves it can go red.** The restore drill must FAIL loudly on a
  deliberately corrupted backup. The recovery rehearsal must show that the WRONG key
  fails before showing that the right procedure succeeds. The CI workflow must be
  shown to fail on a deliberately broken test (locally, red commit never pushed).
- **Assert on content, never on exit codes.** A restore passes when row counts /
  checksums / a known sentinel record match the source, not when a command exits 0.
- **Copies only.** Every drill and rehearsal runs against copies in scratch
  directories. Nothing in this wave touches live data, live keystores, or the
  deployed PH machine. The drill harness refuses to run against a path that looks
  live (guard built in, guard tested).
- **Real secrets never enter the repo.** The custody map names keys and locations; it
  NEVER contains key material. Grep-gate before commit. The recovery-envelope
  procedure describes what the owner writes down offline; the repo carries the
  template, not the filled envelope.
- **Coder agents receive briefs, return evidence.** Each mission's coder reports
  red-then-green transcripts for every check. The primary gate (Fable) re-runs
  independently; a claim without a transcript is not evidence.

## 2. Missions

### CW1-0 — Inventory (orchestrator, before any coder launches)
Complete two inventories on today's main, written to `docs/custodian/CW10_INVENTORY.md`:
1. **Key surfaces:** every place key material is created, stored, or consumed —
   FieldCrypto master key path, DPAPI keystore(s), license/hardware-id keys, mesh
   room keys + registry, session/JWT secrets, any `.env`-class config. For each:
   file:line, storage location at runtime, loss consequence.
2. **Data-at-rest surfaces:** every store a restore must cover — ERP DB (engine,
   location, existing backup.go coverage), mesh data folders, keystore files,
   uploaded documents, config. For each: what the existing backup machinery covers
   vs. does not.
**Gate CW1-0:** inventory doc exists; every claim carries a file:line or command
evidence; the "NOT covered by existing backup" list is explicit.

### CW1-A — The custody map + recovery rehearsal (C2)
Deliverables:
- `docs/custodian/KEY_CUSTODY.md` — the map, one section per key from CW1-0: lives
  where / protects what / recovered how / dies-with-it list / rehearsal evidence link.
- `docs/custodian/RECOVERY_ENVELOPE_TEMPLATE.md` — what the owner writes down and
  seals offline (named fields, no material), including the mesh room-key doctrine
  (what is deliberately unrecoverable BY DESIGN, stated so nobody "fixes" it later).
- A rehearsal script (`scripts/custodian/rehearse_recovery.*`) that, on a COPY:
  (1) proves encrypted fields are unreadable under a wrong key (red first),
  (2) executes the documented recovery path and proves readback (green second),
  (3) for the DPAPI keystore: documents + demonstrates what survives and what dies
  across a simulated profile loss (copy-based; honest about what can't be simulated).
**Gate CW1-A (red-provable):** rehearsal transcript shows red-then-green; custody map
covers 100% of CW1-0's key list (diff-checked, not eyeballed); no-key-material grep
gate green; every "cannot rehearse on dev machine" is listed as residue, not skipped
silently.

### CW1-B — The restore drill (C1)
Deliverables:
- `scripts/custodian/drill_restore.*` — harness that: takes a backup artifact
  produced by the EXISTING machinery (pkg/infra/db/backup.go path — extend only if
  CW1-0 found uncovered stores, and then as a stop-and-report first), restores to a
  scratch target, and verifies CONTENT (row counts per critical table + sentinel
  checksums + a known record readback). Includes the liveness guard (refuses
  non-scratch targets; guard has its own negative test).
- `docs/custodian/DISASTER_RECOVERY_RUNBOOK.md` — the human procedure: blank-machine
  sequence, where backups live, stopwatch expectations (filled from the drill run,
  not invented), corrupt-backup behavior, ransomware note (offline copy discipline),
  stolen-laptop note (what the thief can and cannot read — links to KEY_CUSTODY.md).
- **The drill actually run** on the dev machine against a copy: timed, transcript
  kept, numbers in the runbook.
**Gate CW1-B (red-provable):** drill green with content assertions AND red on a
deliberately corrupted artifact (bytes flipped, not just missing file) AND red on the
liveness guard; stopwatch numbers recorded; mesh-folder restore covered (folder IS
the data — copy-back + kit reopens the room and reads a pre-backup message, content-
asserted).

### CW1-C — Public CI (C7)
Deliverables:
- `.github/workflows/gate.yml` — on push/PR to main: Go build, `go vet`, full
  `go test ./...` on `windows-latest` (the product's OS — if suite time forces a
  split, ubuntu for pure-Go + windows for OS-coupled packages, decided on measured
  timings, recorded). Secretless by doctrine. Known env-coupled or hardware-coupled
  tests (wmic/hardware-id class) handled the way the suite already handles them
  (honest skips) — CI must not invent new skips; if it needs one, stop-and-report.
- Optionally a second job for `mesh/` npm gate floor IF it runs clean without
  network-dependent flake (hyperswarm-class checks are AMBER by nature — exclude
  them explicitly and say so in a comment; the deterministic floor only).
- `docs/custodian/CI_LAW.md` — what CI enforces vs. what only the full local gate
  enforces (CI is the floor, not the gate; the wave-gate discipline stays).
- Release hygiene starter: `CHANGELOG.md` seeded from the wave history headlines +
  the tagging convention written down (no tags pushed this wave).
**Gate CW1-C:** every workflow command executed verbatim locally and green; workflow
YAML validated; a deliberately broken test shown to turn the local command red (then
reverted — never committed); doc states the CI/local-gate boundary honestly. First
cloud-green run is a POST-PUSH checklist item (push = owner-cadence), recorded as
such in the report, not claimed in advance.

### CW1-G — FINAL GATE (Fable) 
Independent re-verification: re-run the recovery rehearsal and restore drill from
their docs ALONE (the runbook is the instrument under test — if the gate needs
knowledge not in the doc, that is a finding); re-run CI commands verbatim; re-run the
no-secrets grep; full existing regression floor (go test ./..., frontend build if
touched, mesh floor if touched). Then commit train review → branch ready for owner
review. NOT merged by the orchestrator.

## 3. Report discipline

One report per mission under `docs/custodian/` (`CW1A_REPORT.md` etc.) + a wave
report `FABLE_CUSTODIAN_SPEC_01_REPORT.md` at root, Sealed-Ship honesty standard:
every gate records its negative control, every rehearsal transcript is kept verbatim,
every not-verified item (foreign hardware, non-steward operator, cloud CI run) is in
a RESIDUE section by name. The wave ends with the Wave-2 candidate list (from the
ledger, informed by what this wave surfaced) for the owner review.

*Correctness makes it work. Custody keeps it alive. 🗝️*
