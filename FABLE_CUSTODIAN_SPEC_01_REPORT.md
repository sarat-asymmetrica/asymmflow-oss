# Custodian Wave 1 — The Existential Floor — WAVE REPORT

**Date:** 2026-07-23 · **Branch:** `feat/fable-custodian-w1` (worktree, off main `00f7054`)
**Orchestrator & primary gate:** Fable · **Coders:** 3× Sonnet 5 (one per band) + 2 read-only
inventory agents · **Status:** ALL BANDS GATED GREEN — awaiting owner review; NOT merged.
**Spec:** `FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md` · **Map:** `CUSTODIANS_LEDGER.md`

## 1. What shipped (commits, in order)

| Commit | Band | Deliverable |
|---|---|---|
| `371a7c5` | CW1-0 | `docs/custodian/CW10_INVENTORY.md` — key surfaces + data-at-rest, file:line throughout |
| `0571199` | CW1-A | `KEY_CUSTODY.md` + `RECOVERY_ENVELOPE_TEMPLATE.md` + red-then-green rehearsal harness (`custodian_rehearsal_test.go`, `scripts/custodian/rehearse_recovery/`) + `CW1A_REPORT.md` |
| `cf1f646` | CW1-B | Restore drill (`scripts/custodian/drillrestore/`, `drill_mesh_restore.mjs`) + `DISASTER_RECOVERY_RUNBOOK.md` + stale ops-doc path fix + `CW1B_REPORT.md` |
| `884b617` | CW1-C | `.github/workflows/gate.yml` (public CI floor, secretless) + `CI_LAW.md` + `CHANGELOG.md` seed + `CW1C_REPORT.md` |

Zero runtime-code changes, as chartered. Only existing-file edit: the authorized
`docs/ops/BACKUP_RESTORE_PREFLIGHT_V0_1.md` dead-path correction.

## 2. Final gate (CW1-G) evidence

- `go test -count=1 ./...` fresh full suite: **GREEN, exit 0** (no FAILs; `-count=1`
  after the cache lesson below).
- Mesh floor `npm run smoke`: **GREEN** (3-peer byte-identical convergence, pinned golden).
- Frontend `npm run build`: **GREEN** (20.9s; `dist/` placeholder reverted byte-identical).
- CW1-A rehearsal independently re-run `-count=1`: wrong-key RED → missing-salt RED →
  `ImportKeyMaterial()` recovery GREEN, sentinel byte-identical; scratch-guard 6/6.
- CW1-B Go drill independently re-run: liveness-guard 2× RED + allow control, content
  assertions all green, corrupt-artifact refused BY THE REAL `VerifyBackup`
  (btreeInitPage error surfaced) — creation-time detection of mid-file corruption is
  therefore REAL, upgrading CW10's "verification gap" from unknown to bounded.
- CW1-B mesh drill independently re-run: RED reproduced (see finding W1-F1), diagnostic
  probe (raw Corestore `allowBackup:true`) opens the same copied dir; keys-withheld
  negative control fires correctly.
- Secrets hygiene: 64-hex grep clean across all new files (per-band + self-check test
  that scans its own source); `gate.yml` has zero `secrets.*` references.

**Gate-time findings by the gate itself (Rule-6 class, both fixed/handled):**
1. **The Go test cache almost laundered a gate.** First CW1-A "independent re-run" was
   served `(cached)` — a replay of the coder's own run. All gate re-runs now use
   `-count=1`. Method note for every future gate on this repo.
2. **CW1C-G1:** the mesh CI job shells out to `go build` (wasip1 reducer) but rode the
   runner image's floating Go toolchain — pinned `actions/setup-go@v5 → 1.25.0` to
   match the go job (fix committed in `884b617`).

## 3. Findings for the owner (Wave-2 candidate ledger)

**From the inventory (crypto semantics — out of this wave's scope by doctrine):**
1. Deterministic PUBLIC fallback keys when hardware-ID resolution fails
   (`"fallback-key-ace-engine"`, `"fallback-key-asymmetrica-auth"`) — data encrypted
   under them is effectively unprotected.
2. Two hardcoded static salts in the binary (`settings_service.go:54`, `auth_handler.go:862`).
3. Mesh key material plaintext on disk (`data/keys/` — seed, room keys, content keys);
   a stolen laptop hands over full mesh identity + room decryption.
4. DPAPI machine-scope (documented intentional; revisit vs user-scope).
5. Partial device-hash logging (low risk; noted).
6. No automatic FieldCrypto escrow — the only recovery for the highest-value data is a
   manual export a human must run IN ADVANCE.

**New this wave:**
- **W1-F1 (mesh restore is broken today):** `hypercore-storage` stamps each room dir
  with its filesystem inode and refuses reopen after ANY folder copy ("Invalid device
  file, was modified") — so "the folder IS the data" only holds if the folder never
  moves. Diagnostic-proven fix: thread `allowBackup: true` into Corestore construction
  (`mesh/host/mesh-node.mjs:80`) — one line of runtime code, stop-and-reported, NOT
  applied. Strong Wave-2 item.
- **W1-F2 (no UI for the key ritual):** `ExportEncryptionBackup`/`ImportEncryptionBackup`
  are wired, admin-gated app bindings with audit logging — but NO frontend screen calls
  them; today's procedure requires DevTools, whose availability on the packaged PH
  build is unverified. Wave-2 candidate: Settings→Security export/import screen.
- **W1-F3 (anchored-room recovery mechanism missing):** anchored (work) mesh rooms are
  supposed to be org-recoverable (Art. II) but no re-share/escrow mechanism exists —
  distinct from social rooms' intentional unrecoverability.
- **W1-F4 (CI hygiene):** frontend-lab has 7 pre-existing npm-audit vulns (1 critical,
  transitive); mesh `npm ci` emits cosmetic tar warnings unverified on a fresh runner.

## 4. Residue (named, not fudged)

- **First cloud-green run of `gate.yml` is PENDING PUSH** — only local verbatim proof
  exists. Post-push checklist: watch the first run; prime suspect if mesh goes red is
  the tar-warning behavior on a cold runner (W1-F4).
- Branch protection / required-status-checks: owner GitHub-settings action.
- Foreign-hardware restore + non-steward operator: unprovable on one dev machine —
  the receptionist-class field rehearsal inherits these (natural pairing with the
  Field Packet track's ceremony visit).
- True cross-machine DPAPI failure (approximated by blob corruption); same-machine
  profile-loss (no second Windows account available).
- Stopwatch numbers are synthetic-scale; re-time on a production-sized copy before
  quoting any SLA.
- Off-machine backup copy remains a MANUAL procedure (runbook §5); no code performs it.
- vitest/svelte-check not in CI; wails installer build belongs to release ceremony (C8).

## 5. The owner ritual (action available TODAY)

`KEY_CUSTODY.md` §"Owner ritual": run `ExportEncryptionBackup()` via DevTools console
on the deployed PH machine, hand-copy `master_key_hex` + `salt_hex` into the
`RECOVERY_ENVELOPE_TEMPLATE.md` fields, seal offline. Until this is done once, a
hardware failure on the PH machine means permanent loss of every encrypted employee
document number, bank account, and IBAN. **This is the single highest-leverage
15 minutes available in the entire custody program.**

## 6. Wave-2 candidate list (for the review)

From the ledger + this wave's findings, in rough owner-leverage order:
1. Key-surface hardening: findings 1+2+6 (kill public fallbacks: fail-closed instead;
   auto-escrow prompt/reminder; salt strategy) + W1-F2 export UI.
2. W1-F1 mesh `allowBackup` wiring + a mesh-side backup ritual (folder snapshot is
   fast — 26ms measured — once reopen works).
3. Off-machine backup automation (C1 completion: the colocation gap).
4. C3 upgrade path or C4 flight recorder as the next Tier-1 organ.

*Correctness makes it work. Custody keeps it alive. 🗝️*
