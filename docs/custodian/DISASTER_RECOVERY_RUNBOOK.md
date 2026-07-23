# Disaster Recovery Runbook

**Wave:** Custodian 1 "The Existential Floor" (`FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md`), Mission CW1-B
**Audience:** whoever has to bring a dead or corrupted AsymmFlow machine back to life — not necessarily the person who wrote this doc.
**Source of truth for paths:** `docs/custodian/CW10_INVENTORY.md`. **Source of truth for keys:** `docs/custodian/KEY_CUSTODY.md` — this runbook does not repeat the key-recovery detail, it tells you WHEN to go read that document.

---

## 0. Read this box first

> **Cloud sync is NOT a backup.** AsymmFlow's optional cloud sync (`ENABLE_CLOUD_SYNC`) is a partial, merge-only mirror of ~70 business tables. It does NOT carry `settings`, `sync_records`, `audit_logs`, `chart_of_accounts`/journals, chat, or any encrypted field. It cannot resurrect a machine on its own. If cloud sync is the only thing you have, you do not have a backup — you have some of the business data and none of the configuration, GL, or crypto material. (CW10_INVENTORY.md Surface 3.)

> **All local backups live on the SAME disk as the live database.** `pkg/infra/db.Backuper` writes rotated snapshots to `<dbDir>\backups\` — the same drive as `ph_holdings.db`. There is no code path that copies a backup off-machine. If the disk fails, the database AND every local backup are gone together. Off-machine copy is a **manual step** — see §5.

---

## 1. Where backups live on a deployed machine

The database path resolution is a total three-step order (`pkg/infra/deploy/paths.go:150-176`), and the backup directory sits next to whichever database path wins:

| Resolver step | Database path | Backup directory |
|---|---|---|
| 1. `PH_DB_PATH` env set | value of `PH_DB_PATH` (resolved against CWD if relative) | `<dir of PH_DB_PATH>\backups\` |
| 2. `portable.flag` next to the exe | `<exeDir>\data\ph_holdings.db` | `<exeDir>\data\backups\` |
| 3. Default (no override) | `%APPDATA%\Asymmetrica\<slug>\data\ph_holdings.db` | `%APPDATA%\Asymmetrica\<slug>\data\backups\` |

`<slug>` defaults to `AsymmFlow-Dev` unless an exe-adjacent `deployment.json` sets a different slug (`pkg/infra/deploy/paths.go:33,92-108`). **Check `deployment.json` before assuming the default slug** — a wrong slug guess resolves to a directory that looks empty, not an error.

Two things write into `backups\`: the runtime rotating backup (keep 7, `database.go:470-491`) and a pre-migrate/pre-reseed snapshot (`pre-migrate-<ts>.db`, keep 5, `contract.go:337-353`). Both are plain files named `<stem>_<timestamp>.db` — newest by filename sort is newest by time.

The legacy path `%APPDATA%\AsymmFlow` (no `Asymmetrica` vendor folder) is **dead** — nothing in the current codebase reads or writes it (`paths.go:9-10`). If you find data there, it is from a pre-Mission-DP1 install and needs a manual one-time migration, not a restore.

## 2. Blank-machine resurrection sequence

This is the order that avoids the single worst failure mode this wave found (§3 below): a DB restore that "succeeds" while every encrypted field becomes permanently unreadable.

1. **Before touching the new machine**, locate:
   - The most recent verified backup file (`<backups>\ph_holdings_<timestamp>.db`).
   - The exported FieldCrypto key material, if it exists (`ExportKeyMaterial()`/`ExportSalt()`, `field_crypto.go:270,277`) — or the ORIGINAL machine's `.field_crypto_salt` and `.hardware_id` sidecar files, if the original machine/disk is still reachable.
   - `.env` (holds `ENCRYPTION_MASTER_KEY` if that path was used instead of hardware-ID derivation, plus Azure/Supabase creds).
   - `deployment.json` (the slug — gets the new install pointed at the right data-plane directory).
   - The KEY_CUSTODY.md-documented recovery envelope, if one was sealed (`docs/custodian/RECOVERY_ENVELOPE_TEMPLATE.md`).
2. **Install the application** on the new machine (installer or portable build). Do NOT let it boot against a fresh/empty data plane yet if you have a backup to restore — an empty boot will seed a NEW hardware-ID-derived key and a NEW `schema_version` baseline that then has to be reconciled.
3. **Stop the app** if it auto-started.
4. **Place the database:**
   - Copy the chosen backup file to the resolved database path (§1) as `ph_holdings.db`. Delete any stray `ph_holdings.db-wal` / `-shm` in the target directory first — a restored snapshot must not inherit a stale journal.
   - This drill's `db.Restore()` (`pkg/infra/db/backup.go:160-179`) does this same swap safely (verify → snapshot-current → copy → drop journals) if you are scripting it; today there is no in-app button, so a manual file copy following the same order is the supported path (§4's finding explains why).
5. **Restore the key material** (this is the step people skip and then lose PII):
   - If `ENCRYPTION_MASTER_KEY` was the original master-secret path: set the same env var / `.env` entry on the new machine. Done — FieldCrypto re-derives correctly.
   - Otherwise (hardware-ID-derived, the default): you need BOTH the original `.field_crypto_salt` AND either the original `.hardware_id` plaintext sidecar or an `ImportKeyMaterial()` restore of the exported key. See `docs/custodian/KEY_CUSTODY.md` §1-3 for the exact procedure and what happens if you skip this (short version: the DB opens fine, `PRAGMA integrity_check` passes, and every encrypted field — employee CPR/passport/visa/permit, bank accounts/IBANs, encrypted settings — is permanently unreadable garbage).
6. **Restore `.env` and `deployment.json`** if the new machine doesn't already have the right ones (deployment.json controls which data-plane directory the app even looks at — wrong slug = "my data is gone" even though it's sitting right there under the old slug).
7. **Start the app.** On boot it will run the update contract (`contract.go`): schema-stamp check (refuses to open a DB stamped for a NEWER binary than the one running — this is a feature, not a bug: get a matching or newer binary), then normal boot.
8. **Verify** (content, not exit codes):
   - `PRAGMA integrity_check` = `ok`.
   - Row counts on `customers`, `invoices`, `payments`, `orders` look sane relative to what you expect.
   - Open one record known to have an encrypted field (an employee's compliance doc, or a bank account) and confirm it decrypts — this is the FieldCrypto canary. If it doesn't decrypt, STOP and go fix step 5 before doing anything else (do not let the app re-encrypt garbage over readable ciphertext).
   - `settings` table has a `schema_version` row.
9. **Re-enable cloud sync (if used) only after step 8 passes.** Syncing a broken restore outward is how one bad machine becomes two.

## 3. Stopwatch numbers (from an actual drill run — not invented)

From `scripts/custodian/drillrestore` (Go, real `pkg/infra/db` engine, synthetic representative schema — see that file's header for why a subset schema was used instead of the full ~90-table GORM catalog), run 2026-07-23 on the dev machine:

| Leg | Duration |
|---|---|
| Backup (`VACUUM INTO`) | **22.27 ms** |
| Verify + Restore (copy-over + journal cleanup) | **14.64 ms** (restore-copy alone: 7.01 ms) |

These numbers are for a small synthetic DB (a handful of rows across 4 tables) and will not extrapolate linearly to a multi-GB production database with ~90 tables — `VACUUM INTO` cost scales with page count. Treat these as a floor, not an estimate for a real deployment; re-time on a realistic data volume before quoting an SLA.

Mesh-folder leg timings (folder copy, from `scripts/custodian/drill_mesh_restore.mjs`, same run): backup (recursive folder copy) **~29-36 ms**, restore (recursive folder copy) **~23-46 ms** for a single-room scratch device. See §4 — the copy itself is fast; the REOPEN after copying is where the current gap lives.

## 4. Corrupt-backup behavior (verified, not assumed)

The drill deliberately flips a run of bytes mid-file (not a truncation) in a COPY of a real backup artifact, then calls the production `VerifyBackup()`/`Restore()` functions against it.

**Result: the real machinery caught it correctly, every time.** `VerifyBackup()` (`pkg/infra/db/backup.go:129-146`) runs `PRAGMA integrity_check` and refuses on anything but `ok`; `Restore()` calls `VerifyBackup()` FIRST, before touching the target file, so a corrupted backup is refused before any damage — the pre-existing target (if any) is left untouched. This is the same code path a real restore would hit, so: **if you restore with `db.Restore()`, a corrupted backup will not silently overwrite a working database.** A manual file copy (dragging the `.db` file over) has no such protection — always run integrity_check (e.g. `go run ./cmd/sqlite_integrity -db <path>`, see `scripts/preflight_backup_restore.ps1`) on any backup before trusting it if you are not going through `Restore()`.

## 5. Colocation / off-machine copy (manual procedure, today)

No code in this repo copies a backup off the machine it was created on. Until that changes (a Wave-2 candidate, not built here), the manual procedure is:

1. After a backup rotation (or on a schedule you choose), copy the newest file in `<dbDir>\backups\` to removable media, a network share, or a second machine.
2. Do the same for the FieldCrypto key material export (`ExportKeyMaterial`/`ExportSalt`) — see `docs/custodian/KEY_CUSTODY.md`. A DB backup without its matching key export is not a usable backup for encrypted fields.
3. Keep at least one copy that is NOT on the same disk, NOT on the same machine, and ideally not on the same site as the live install.
4. Label the copy with the timestamp already in the filename — do not rename it.

## 6. Ransomware note

Because backups are colocated by default (§0, §5), a ransomware event that encrypts the machine's disk takes the live DB AND every local rotation with it in one shot. The only defense this wave found is discipline, not code:

- Maintain an OFFLINE copy (disconnected media) on the cadence in §5 — an attached network share is not offline; anything mounted at encryption time is in scope.
- Do not rely on the 7-deep local rotation as ransomware protection — it protects against a bad migration or accidental deletion, not against an attacker with write access to the whole disk.
- If you suspect compromise, treat every backup on the affected disk as untrusted until verified (`PRAGMA integrity_check` + a manual look at row content) from the offline copy instead.

## 7. Stolen-laptop note

What a thief holding the bare disk/machine can and cannot read (see `docs/custodian/KEY_CUSTODY.md` for the full key-by-key breakdown):

- **Cannot read without more:** FieldCrypto-protected fields (employee CPR/passport/visa/permit numbers, bank accounts/IBANs, encrypted settings) — the DPAPI-wrapped `.hardware_id.dpapi` sidecar is `CRYPTPROTECT_LOCAL_MACHINE`-scoped, so it does not travel usefully off the machine's own Windows install by itself, and the FieldCrypto master key is either an env var not on disk in plaintext (if `ENCRYPTION_MASTER_KEY` was used) or hardware-ID-derived (unusable without the live machine's own hardware-ID resolution).
- **Can read if the machine itself boots (no separate device password/BitLocker assumed):** everything NOT FieldCrypto-protected — plaintext business data in most tables, plaintext `.env` credentials (Azure/Supabase/API keys — all re-issuable, but should still be rotated on theft), plaintext `.hardware_id` legacy sidecar if the migration hasn't happened yet, and — separately — mesh `data/keys/` (device seed, room keys, invite/authority keys) which is **plaintext on disk, not DPAPI-wrapped** (CW10_INVENTORY.md Finding 3). A stolen laptop with mesh set up hands the thief that device's full mesh identity and every room key it holds, unless those rooms have been re-keyed/revoked since.
- **Action on theft:** rotate `.env` credentials; consider the machine's mesh identity compromised and use the mesh revocation/re-key path (crypto-epoch successor, see KEY_CUSTODY.md) for any room that device had write access to; if `ENCRYPTION_MASTER_KEY` was ever typed into that machine's `.env`, treat FieldCrypto data as exposed too and plan a key rotation.

## 8. Mesh folder restore procedure — CURRENT LIMITATION (read before relying on this)

"The folder is the data" (CW10_INVENTORY.md Surface 7) is correct as a *storage* description, but this wave's drill (`scripts/custodian/drill_mesh_restore.mjs`) proved that **a plain recursive folder copy of `data/` does NOT reliably restore a real room today.**

Root cause (confirmed by a targeted diagnostic in the drill script): `hypercore-storage` (a mesh dependency) writes a sentinel `CORESTORE` file per room-storage directory recording that directory's filesystem inode at creation time, and refuses to reopen the room if the inode has changed — a built-in guard against silently forking a replicated multiwriter store. **Any ordinary file copy always produces new inodes**, so this guard trips on every folder-copy restore, not just a corrupted one. The dependency exposes an `allowBackup: true` option that skips this check for exactly this scenario, but `mesh/host/mesh-node.mjs` (the only place AsymmFlow constructs a room's `Corestore`) does not pass it — wiring that in is a runtime-code change, out of scope for this wave (stop-and-report, see `docs/custodian/CW1B_REPORT.md`).

**What this means operationally, today:**
- A `data/` folder copy still correctly preserves `data/keys/` (device identity + room registry, both proven to survive a copy in the drill) and the raw corestore bytes.
- Reopening a device from a COPIED `data/` directory currently fails for any room whose storage directory was copied (not the original). The device gets a working identity but the specific room comes back "unknown room" until the copy issue above is fixed in mesh-node.mjs.
- **Until that fix lands:** do not rely on a mesh folder copy as a working restore for rooms that must reopen. Treat mesh data as recoverable-in-principle (bytes are intact) but NOT push-button restorable today. This is recorded as a Wave-2 candidate, not silently worked around.
- `data/keys/` withheld (the "device loses its identity" scenario) behaves as documented: the device mints a brand-new identity and the old room registry is gone — this is correct, proven behavior, not a bug.

## 9. Cloud sync is NOT a backup (again, on purpose)

Repeating §0 here because this is the single most common operator mistake this wave's inventory found: cloud sync being "on" does not mean a backup exists. See CW10_INVENTORY.md Surface 3 for the exact table coverage gap. If your disaster-recovery plan says "we have cloud sync" and nothing else, you do not have a plan.
