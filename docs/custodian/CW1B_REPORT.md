# CW1-B Report — The Restore Drill (C1)

**Wave:** Custodian 1 "The Existential Floor" (`FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md`)
**Mission:** CW1-B, restore drill + disaster recovery runbook
**Date:** 2026-07-23 · **Branch:** `feat/fable-custodian-w1`
**Standard:** Sealed-Ship honesty — every negative control transcript kept verbatim, every unverified item named as residue, no exit-code-only assertions.

---

## 1. What was built

1. `scripts/custodian/drillrestore/main.go` — standalone `go run`-able Go harness (package `main` under its own directory, run via `go run ./scripts/custodian/drillrestore`). Exercises the REAL `pkg/infra/db` engine (`Backuper.Backup` = `VACUUM INTO`, `VerifyBackup`, `Restore`) against a synthetic SQLite database, entirely under `%TEMP%\custodian-drill\<run-ts>`.
2. `scripts/custodian/drill_mesh_restore.mjs` — standalone Node/ESM harness imitating `mesh/kit/kit-spike.mjs`'s structure (`createKitHost`/`createCommandLayer`, TCP-fallback-only, no hyperswarm). Proves "the folder is the data" via a real room, a real folder copy, and a real reopen attempt, under `%TEMP%\custodian-drill-mesh-<run-ts>-*`.
3. `docs/custodian/DISASTER_RECOVERY_RUNBOOK.md` — the human procedure.
4. One-paragraph correction to `docs/ops/BACKUP_RESTORE_PREFLIGHT_V0_1.md` (dead `%APPDATA%\AsymmFlow` path → three-step resolver, pointer to the new runbook). This is the only pre-existing file touched.

No runtime code (`backup.go`, `contract.go`, `paths.go`, `database.go`, any file under `mesh/host/` or `mesh/kit/`) was modified. `db.Restore()`/`db.VerifyBackup()` are called as a library from the drill only, never wired into `App` or the UI.

**Scenario-build choice (Go drill):** the app's real schema comes from GORM AutoMigrate on the compiled binary (~90 tables, `database.go:27-443`), not SQL files. Importing that whole surface into a throwaway drill binary would be heavy and couple the drill to unrelated schema churn. The drill instead opens a real SQLite DB through the same driver the product uses (`github.com/ncruces/go-sqlite3/driver`, `"sqlite3"`) and creates a representative subset of the tables CW10_INVENTORY.md names as restore sentinels (`customers`, `invoices`, `payments`, `settings` incl. `schema_version`), seeded with synthetic rows carrying a SHA-256 checksum. This exercises the exact backup/restore CODE PATH PH runs; it does not claim to exercise the full production schema.

## 2. Green transcript — Go restore drill (`go run ./scripts/custodian/drillrestore`)

```
Custodian Wave 1 — CW1-B restore drill
Scratch root: C:\Users\schan\AppData\Local\Temp\custodian-drill\20260723_133556

== Liveness guard — negative controls ==
  [PASS-RED] guard refuses a live-looking ph_holdings.db outside the scratch root (correctly refused: refused — C:\Users\schan\AppData\Local\Temp\custodian-drill-livecheck-fake-20260723_133556\ph_holdings.db is named ph_holdings.db but sits outside this drill's scratch root (C:\Users\schan\AppData\Local\Temp\custodian-drill\20260723_133556))
  [PASS-RED] guard refuses a path under %APPDATA%\Asymmetrica (string-only, never touched) (correctly refused: refused — C:\Users\schan\AppData\Roaming\Asymmetrica\AsymmFlow-Dev\data\ph_holdings.db is under the live Asymmetrica data plane (C:\Users\schan\AppData\Roaming\Asymmetrica))
  [PASS] guard allows a path inside the drill's own scratch root

== Scenario build (synthetic DB, real driver + schema-subset) ==
  [PASS] scenario: source DB seeded with customers/invoices/payments/settings
  sentinel customer id=1 name="CUSTODIAN-DRILL-SENTINEL-20260723_133556" checksum=4aadcc67db846f118ddf65206be2dcd1a4ed0e210c18b9e155fe51a44ab81551

== Backup (real engine: pkg/infra/db.Backuper.Backup) ==
  [PASS] backup: artifact created
  backup artifact: C:\Users\schan\AppData\Local\Temp\custodian-drill\20260723_133556\source\backups\ph_holdings_20260723_133557.db
  STOPWATCH backup leg: 78.5069ms

== Restore (real engine: pkg/infra/db.Restore + VerifyBackup) — GREEN path ==
  [PASS] VerifyBackup: real (uncorrupted) artifact passes integrity_check
  [PASS] Restore: succeeds against a fresh scratch target
  [PASS] Restore: no pre-restore snapshot needed for an empty target
  STOPWATCH restore+verify leg: 25.3986ms (restore-only: 21.7957ms)
  [PASS] restored DB: PRAGMA integrity_check == ok
  [PASS] restored DB: customers row count matches source
  [PASS] restored DB: invoices row count matches source
  [PASS] restored DB: payments row count matches source
  [PASS] restored DB: sentinel record readback byte-for-byte
  [PASS] restored DB: settings.schema_version survives

== Corrupt-backup negative control — RED path (mandatory) ==
  [PASS] corrupt artifact: file still exists post-corruption (bytes flipped, not truncated)
  [PASS-RED] VerifyBackup: REFUSES the corrupted artifact (correctly refused: db: backup failed integrity check: *** in database main ***
Tree 4 page 4: btreeInitPage() returns error code 11)
  [PASS-RED] Restore: REFUSES to restore from the corrupted artifact (correctly refused: db: backup failed integrity check: *** in database main ***
Tree 4 page 4: btreeInitPage() returns error code 11)
  [PASS] corrupt-restore: target file was NOT created (Restore refused before copying)

== Summary ==
Scratch root (left in place for inspection): C:\Users\schan\AppData\Local\Temp\custodian-drill\20260723_133556
Backup leg:        78.5069ms
Restore+verify leg: 25.3986ms

RESTORE DRILL GREEN — all assertions passed, both negative controls fired red as required.
```

**Result: fully green.** Both mandatory negative controls (corrupt-artifact, liveness-guard) fired RED exactly as required, and the corrupted-artifact control shows the REAL machinery — not a drill-side check — refusing the corruption (`db: backup failed integrity check: ... btreeInitPage() returns error code 11`, from `VerifyBackup`/`Restore` in `pkg/infra/db/backup.go`).

## 3. Red transcript — mesh-folder restore drill (`node scripts/custodian/drill_mesh_restore.mjs`)

```
Custodian Wave 1 — CW1-B mesh-folder restore drill (folder IS the data)

Scratch root: C:\Users\schan\AppData\Local\Temp\custodian-drill-mesh-2026-07-23T08-05-58-172Z-gBzvff

== Scenario build (real kit host + real room) ==
  [PASS] boot: device has a real identity
  [PASS] create: room open on the drill device
  [PASS] scenario: sentinel message present before backup
  [PASS] scenario: device closed cleanly before folder copy (no live writer during backup)

== Backup (folder copy — this IS the mesh backup) ==
  [PASS] backup: keys/ subtree copied
  [PASS] backup: keys/rooms.json copied
  [PASS] backup: corestore/ subtree copied
  STOPWATCH backup (folder copy) leg: 27ms

== Destroy original ==
  [PASS] original data dir removed

== Restore (green path) — reopen via the kit registry, content-assert the sentinel ==
  STOPWATCH restore (folder copy) leg: 22ms
  [host log] could not reopen a registered room (3a7d10afa98bf04e…): Invalid device file, was modified
  [host log] device ready — actor "custodian-drill", devicePub 1b57e16f83e95b34…, bridge on 127.0.0.1:51318
  [FAIL] restore: the room comes back automatically via kit-registry.mjs
  [FAIL] restore: manifest title survives — skipped — room did not come back, see [host log] above
  [FAIL] restore: sentinel message reads back CONTENT-identical after restore — skipped — room did not come back

== Root-cause diagnostic — device-file inode guard (read-only probe) ==
  [PASS] diagnostic: allowBackup:true opens the SAME copied directory that just failed

== Negative control — restore WITHOUT data/keys/ (mandatory RED) ==
  [PASS] negative-control fixture: corestore/ present, keys/ deliberately withheld
  [PASS-RED] negative control: a NEW device identity was minted (old identity is gone with keys/) — new pubHex=5570cb884cea7af5…
  [PASS-RED] negative control: the room is NOT found (no rooms.json survived) — server.rooms does not contain the original roomKey

== Summary ==
Backup (folder copy) leg:  27ms
Restore (folder copy) leg: 22ms

MESH RESTORE DRILL RED — 3 assertion(s) failed. See [FAIL] lines above.
```

**This RED is a real finding, not a script bug.** It is honestly reported below, not fixed and not papered over, per the mission brief's own permission ("if the mesh leg proves too entangled to script reliably, deliver what is provable and record the rest as residue — honest red/residue beats bought green") and per §0 stop-and-report (no runtime-code wiring).

## 4. Findings

### FINDING 1 (mesh, load-bearing): a plain folder copy of `data/` does not reliably restore a real room today

**What happens:** create a real room on a real `kit-host.mjs` device, post a message, close the device, copy `data/` to a new location with `cpSync(..., {recursive:true})`, delete the original, boot `createKitHost` against the copy. The device gets a working identity (keys/ survives the copy fine — proven), but the room registered in `data/keys/rooms.json` fails to reopen: `could not reopen a registered room (...): Invalid device file, was modified`.

**Root cause (confirmed, not guessed):** `hypercore-storage` (a mesh dependency, `mesh/node_modules/hypercore-storage/index.js:514-521`, using `mesh/node_modules/device-file/index.js`) writes a sentinel `CORESTORE` file inside each room's storage directory recording that directory's filesystem inode (`st.ino`) at creation time (`device-file/index.js:96,191`). On open it re-`fstat`s the directory and refuses (`"Invalid device file, was modified"`, fatal) if the inode differs. This is a deliberate move/copy-safety guard against silently forking a replicated multiwriter Hypercore store — **and it fires on every ordinary file copy**, because a copy always allocates new inodes; it is not specific to corruption.

The drill confirms this diagnosis directly: a targeted read-only probe (`new Corestore(sameCopiedDir, { allowBackup: true })`, using the `corestore` dependency directly — no mesh runtime file touched) opens the SAME directory that just failed, cleanly, every run. `hypercore-storage` threads `allowBackup` straight through from the `Corestore` constructor (`hypercore-storage/index.js:499,514`; `corestore/index.js:244`) specifically to skip the inode guard for exactly this restore scenario.

**Why this is stop-and-report, not fixed here:** the fix is `new Corestore(storage, primaryKey ? {...} : { allowBackup: true })` (or similarly threading an `allowBackup` flag) in `mesh/host/mesh-node.mjs:80` — the only place AsymmFlow constructs a room's `Corestore`. That is a runtime-code change to mesh's replication layer, explicitly out of scope for this wave (§0: "the wave MAPS keys, it does not touch crypto/replication code"; "any backup-format change" is a stop-and-report item). Recorded here for the owner and for the Wave-2 candidate list.

**Operational impact today:** a `data/` folder copy still faithfully preserves the identity (`keys/`) and the raw bytes (`corestore/`), so mesh data is not lost/corrupted by a folder copy — but it is not currently push-button restorable for rooms whose storage directory was itself copied. `docs/custodian/DISASTER_RECOVERY_RUNBOOK.md` §8 documents this honestly and tells an operator not to rely on it as a working restore until the runtime fix lands.

### FINDING 2 (Go/db engine, reassuring): corrupt-backup detection works exactly as designed

No gap found here. `VerifyBackup()`/`Restore()` (`pkg/infra/db/backup.go`) caught a mid-file byte-flip corruption on every run, via `PRAGMA integrity_check`, and `Restore()` refused BEFORE touching the restore target (confirmed: the corrupt-restore target file was not created). This is the real machinery, not a drill-side shortcut — the drill imports and calls the exact same exported functions `backup_test.go` already unit-tests, against a real artifact the drill's own `Backuper.Backup` produced.

### FINDING 3 (liveness guard, self-test): guard correctly refuses both named live-path shapes

The drill's own liveness guard (built specifically for this drill, not shared with any runtime code) correctly refused (a) a scratch file literally named `ph_holdings.db` sitting outside the drill's scratch root, and (b) a path string under `%APPDATA%\Asymmetrica` — the latter checked purely as a string, no I/O against the real directory. A positive control (a path inside the drill's own scratch root) was allowed, proving the guard is not simply refuse-everything.

## 5. Stopwatch numbers (from the runs above — not invented)

| Leg | Duration | Source |
|---|---|---|
| Go: Backup (`VACUUM INTO`) | 78.51 ms | this run (a prior run measured 22.27 ms — see note below) |
| Go: Verify + Restore (copy-over + journal cleanup) | 25.40 ms (restore-copy alone: 21.80 ms) | this run |
| Mesh: Backup (recursive folder copy) | 27 ms | this run (27-36 ms range across 3 runs) |
| Mesh: Restore (recursive folder copy) | 22 ms | this run (22-46 ms range across 3 runs) |

Note on Go backup-leg variance (22 ms vs 78 ms across runs on the same machine): both runs used the same tiny synthetic DB (4 tables, single-digit row counts) — the variance reflects OS/filesystem scheduling noise at this scale, not a workload difference. **These numbers do not extrapolate to a real multi-GB, ~90-table production database** — `VACUUM INTO` cost scales with page count. The runbook (§3) states this caveat explicitly; do not quote these numbers as a production restore SLA.

## 6. Residue (named, not silently skipped)

1. **Mesh folder-copy restore is not push-button working today** (FINDING 1) — needs a runtime-code change (`allowBackup` threading in `mesh-node.mjs`) that is out of scope for this wave. Recommended Wave-2 candidate.
2. **Stopwatch numbers are synthetic-scale only.** No drill run in this wave exercised a realistic production-sized database (~90 tables, GB-scale). Re-timing on a realistic data volume before quoting any restore SLA is residue.
3. **Foreign hardware restore (the FieldCrypto killer scenario) is documented but not drilled here.** CW10_INVENTORY.md's top risk — a DB restore to different hardware "succeeding" while every FieldCrypto field is silently unreadable — is covered procedurally in the runbook (§2 step 5) and is CW1-A's territory (key custody + rehearsal), not re-proven independently by this restore drill. The runbook cross-references `docs/custodian/KEY_CUSTODY.md` rather than duplicating that rehearsal.
4. **No in-app restore command exists** (`db.Restore()` remains unwired, confirmed again by this drill calling it as a library) — the runbook's manual-copy procedure (§2 step 4) is today's real operator path, honestly described as manual, not a UI feature.
5. **This wave runs entirely on the dev machine against copies**, per the wave's own honesty boundary — genuinely foreign hardware and a non-steward operator executing the runbook cold remain unverified (spec §0).

## 7. How to re-run

```powershell
# Go restore drill (no prerequisites beyond `go build` working):
go run ./scripts/custodian/drillrestore

# Mesh restore drill (needs mesh/ dependencies + the wasm reducer built once):
npm --prefix mesh install
npm --prefix mesh run build
node scripts/custodian/drill_mesh_restore.mjs
```

Both scripts print PASS/FAIL per assertion (never rely on process exit code alone — the mesh drill's exit code was inspected too, but every claim above is backed by a `[PASS]`/`[FAIL]`/`[PASS-RED]` line in the transcript, not just the final exit status).
