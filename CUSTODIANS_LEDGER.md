# The Custodian's Ledger

**Status:** RATIFIED 2026-07-23 · **Owner:** the Commander · **Author:** Fable
**What this is:** the standing map of *post-correctness* work — what keeps AsymmFlow
alive after the code is right. Born from one owner question: *"what would a senior
architect ask for that I haven't thought of yet?"* The answer was: almost nothing in
features, almost everything in custody.

**How it is worked:** progressive waves (`FABLE_CUSTODIAN_SPEC_NN_*.md`), one wave
specced in detail at a time, reviewed and re-planned after each. This ledger stays
broad on purpose — it is the map, not the itinerary. Items are not ordered by number;
waves pull from tiers by risk.

---

## Tier 1 — The system is deployed. How does it survive contact with time?

**C1. The restore drill.** A backup that has never been restored is not a backup.
Rehearse, with a stopwatch, blank-machine → working AsymmFlow with yesterday's data —
for the ERP database, the mesh data folders, and the keystores. Include the hostile
variants: corrupted backup detected (not silently restored), ransomware scenario
(backup medium attached to infected machine), stolen-laptop scenario. Deliverable is
a rehearsed runbook + a repeatable drill harness, not a document of intentions.

**C2. Key escrow & the data-death map.** FieldCrypto on employee documents, DPAPI
keystores, `ENCRYPTION_MASTER_KEY`, mesh room keys where deletion-by-forgotten-key is
a design feature. Every one of these makes key loss permanent data loss. Deliverable:
a written custody map (every key in the system — where it lives, what it protects,
who can recover it, what dies with it) + a sealed recovery-envelope procedure for the
owner + a rehearsed recovery on a copy. Cheap to do; existential to skip.

**C3. The upgrade path for the installed base.** Real machines now run installers
(v2.3.x line, DP2). Version N+1 must reach them without a technician: forward-only
schema migrations proven against a mature copy, a rollback story for a bad release,
and an explicit rule for mixed-version machines syncing against each other (promote
the messenger's versioned-signable law to system law).

**C4. The flight recorder.** When the field hits something weird, diagnosis today is
a phone call. Generalize the kit's `VERIFY_EVIDENCE.txt` discipline to the ERP:
structured local error/event log + one-click "export diagnostic bundle" (PII-scrubbed)
a receptionist can send back.

## Tier 2 — The thesis is "the proof is the law." Where is the proof thin?

**C5. The aging harness.** Everything is measured at migration-day volumes. Build a
synthetic data-ager that inflates a copy to five years of growth (100k+ documents)
and re-run the flow gates and timing budgets against it. Find the O(n²) now, not
during a month-end close in 2028. Reusable, run per wave.

**C6. Fuzz the mouths.** Every importer eats hostile real-world files (xlsx, OCR
inbox, shop ingest) — the biggest crash surface and, on a public repo, the biggest
security surface. Property-based invariants on the kernel fold + fuzzing on every
parser that touches a file a client emailed.

**C7. Public CI.** The repo is public and builders are cloning it, but the gate floor
runs only on one machine. GitHub Actions running the suite on every push/PR turns
private discipline into public, self-enforcing law — and insures the gates themselves
against the bus. Includes tagged releases + changelog so versions exist somewhere
besides one disk.

## Tier 3 — What if it succeeds?

**C8. The bus factor & the release ceremony.** If the steward is offline for a month,
can anyone else build, gate, and release? A written, *rehearsed* release ceremony
(reproducible build → gate → sign → tag → ship) that a competent stranger could
follow. Runbooks exist for receptionists; write one for the successor.

**C9. Know what is actually used.** Post-cutover, wave-steering goes blind on which
flows the client actually lives in. A privacy-honest, local-only usage ledger
(volunteered, exportable, never phoned home — constitution-compatible) the owner can
choose to share. Data-steered waves instead of intuition-steered.

---

## Wave log

| Wave | Spec | Items | Status |
|---|---|---|---|
| 1 | `FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md` | C1 + C2 + C7 | MERGED to main `b360ab2` (2026-07-23, owner-authorized; report `FABLE_CUSTODIAN_SPEC_01_REPORT.md`) |

*Correctness makes it work. Custody keeps it alive.*
