# CW1-0 — Inventory: Key Surfaces & Data-at-Rest

**Wave:** Custodian 1 "The Existential Floor" (`FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md`)
**Date:** 2026-07-23 · **Produced by:** two independent read-only inventory agents,
synthesized by the orchestrator. Every claim carries file:line evidence gathered by
direct reading of today's tree (branch `feat/fable-custodian-w1` off main `00f7054`).

## Orchestrator synthesis — what the inventories mean for this wave

1. **One root of trust:** nearly all at-rest crypto keys off the machine hardware ID
   (baseboard serial, `settings_service.go:448`), with `ENCRYPTION_MASTER_KEY`
   overriding for FieldCrypto only. A hardware change without prior key export is
   permanent loss of all field-encrypted PII.
2. **The killer restore failure mode:** a DB restore to different hardware SUCCEEDS at
   the SQLite layer while every encrypted field becomes permanently undecryptable —
   `.field_crypto_salt`, `.hardware_id` sidecars, and `.env` are all OUTSIDE the
   backup engine's coverage, and the DPAPI blob is machine-scoped by design. CW1-B's
   drill MUST include this scenario; CW1-A's custody map MUST make the export
   procedure a first-class owner ritual.
3. **Backups are colocated and unverified at creation:** all rotation lives in
   `<dbDir>\backups\` on the same disk; `VerifyBackup()` runs only at restore;
   `db.Restore()` exists but nothing calls it. Disk loss = total loss today.
4. **Cloud sync is NOT a backup:** ~70-table merge-only partial mirror; cannot
   resurrect settings, GL, audit, or crypto.
5. **Six key-surface findings** (deterministic public fallback keys, hardcoded static
   salts, plaintext mesh keys, machine-scope DPAPI, partial-hash logging, no
   automatic escrow) — recorded below; fixing crypto semantics is OUT of this wave's
   scope (stop-and-report doctrine), but the custody map documents each honestly.
6. **Stale ops doc:** `docs/ops/BACKUP_RESTORE_PREFLIGHT_V0_1.md` cites the dead
   `%APPDATA%\AsymmFlow` path — CW1-B corrects the doc (docs are in scope; runtime
   code is not).
7. **Unrecoverable BY DESIGN (do not escrow):** mesh social-room/DM keys and
   crypto-epoch predecessor keys — forgetting the key IS the deletion feature
   (`MESSENGER_DESIGN_CONSTITUTION.md`). The custody map marks these explicitly so
   nobody "fixes" them later.

---

# Part A — Key & Secret Surface Inventory (agent: inv-keys)

## Root of trust (headline)
Nearly everything at-rest keys off ONE root: the machine hardware ID (baseboard serial), resolved at settings_service.go:448 (`resolveHardwareIDUncached`). Three independent AES keys + the invoice HMAC derive from it. `ENCRYPTION_MASTER_KEY` overrides it for FieldCrypto only. If the motherboard serial changes (or the app moves machines) with no exported key material, all field-encrypted PII and the OAuth token cache become permanently undecryptable — the single largest custody risk.

## 1. FieldCrypto master key (primary at-rest key)
- Created/loaded: `NewFieldCrypto()` field_crypto.go:60. Precedence: (1) ENCRYPTION_MASTER_KEY env (field_crypto.go:41,62) used directly if >=64 hex chars; (2) hardware ID fallback (field_crypto.go:65) strengthened via PBKDF2-SHA256 600k iters (field_crypto.go:40,99).
- Runtime location: derived in memory, never persisted. Inputs on disk = env var (§12) + random salt file `.field_crypto_salt` (field_crypto.go:342, exe-dir or %APPDATA%\Asymmetrica\<slug>\data, mode 0600). Per-version AES keys HKDF-SHA256 (field_crypto.go:118-133).
- Protects (AES-256-GCM): employee CPR/passport/visa/permit numbers (employee_compliance_service.go:42, encrypt :187,257; refuses plaintext fallback :92-96); bank account numbers + IBANs (bank_accounts_service.go:460-461); encrypted `settings` rows incl. Mistral key (settings_service.go:114-116); supplies the invoice HMAC salt (§8).
- Loss consequence: lose master source AND salt → every encrypted employee doc number, bank account, IBAN, encrypted setting permanently unreadable. Losing ONLY the salt is equally fatal even if hardware ID is intact (salt is non-derivable random, field_crypto.go:74-78).
- Recovery: MANUAL only — `ExportKeyMaterial()` field_crypto.go:270 + `ExportSalt()` :277, restore via `ImportKeyMaterial()` :285. No automatic backup. Rotation supported (:232), old versions kept.

## 2. .field_crypto_salt (random salt)
- loadOrCreateSalt() field_crypto.go:341; 32 random bytes :385, atomic 0600 write :392-398.
- Plaintext file, exe-dir first then DataDir. Second required input to FieldCrypto master key AND literal invoice HMAC key.
- Loss = unrecoverable loss of all FieldCrypto data + all invoice hashes stop verifying. Not re-derivable. Recovery only via ExportSalt/ImportKeyMaterial.

## 3. Hardware ID + at-rest sidecars (root identifier)
- resolveHardwareID() settings_service.go:417; live lookup :448 (Windows Get-CimInstance Win32_BaseBoard :499-500, fallback wmic :521; macOS ioreg; Linux /etc/machine-id; last-resort os.Hostname() :489). Memoized once per process :215-220.
- Persisted next to SQLite DB: `.hardware_id.dpapi` (DPAPI-wrapped, Windows steady state, :257,271-308) and `.hardware_id` (plaintext pre-migration/non-Windows fallback, :236-245, 0600; migrated then renamed `.hardware_id.migrated` :357-362).
- Root input for FieldCrypto (§1), legacy SettingsService key (§5), OAuth cache key (§6). One changed byte breaks all three (hence byte-identity effort :438-447).
- Loss: value re-resolvable while machine unchanged; a genuine hardware change makes everything keyed to the old value undecryptable. Recovery = re-resolution :431-434; none across real hardware change except FieldCrypto Export/Import.

## 4. Windows DPAPI keystore (at-rest wrapping)
- keystoreProtect/Unprotect hardware_id_keystore_windows.go:38/60, CryptProtectData with CRYPTPROTECT_LOCAL_MACHINE|CRYPTPROTECT_UI_FORBIDDEN :49.
- DPAPI master key held by OS, machine-scoped; wrapped blob = .hardware_id.dpapi. Protects only at-rest confidentiality of the sidecar, not the derived value.
- Loss: silent fallback to plaintext sidecar/live resolution (settings_service.go:330-331), no data loss on its own. Recovery = automatic fallback.

## 5. Legacy SettingsService key (migration only)
- NewSettingsService() settings_service.go:54 — SHA-256(hardwareID + "asymmetrica-salt-2025") [hardcoded static salt]. In-memory.
- Decrypts pre-FieldCrypto settings rows, transparently re-encrypts to FieldCrypto on read (:138-146). Low loss impact; derivable from hardware ID.

## 6. OAuth (MS Graph) token cache key + .auth_token.json
- getEncryptionKey() auth_handler.go:853 — SHA-256(hardwareID + "ph-holdings-auth-salt-2026") [hardcoded static salt]. AES-256-GCM encrypt/decrypt :779,810.
- Encrypted file .auth_token.json in CWD, 0600 (:659,681): MS access/refresh token + profile.
- Loss = RE-ISSUABLE (just re-login); tokens also hashed in user_sessions table (§7). Recovery = re-auth; plaintext-migration path :704-708.

## 7. Session tokens + password hashes (DB)
- Sessions stored as SHA-256(token) only, plaintext never persisted (auth_session.go:34-43,79-80); access 24h/refresh 30d (:51-54); lookups by hash. Old in-memory SessionManager deleted as dead theater (security_enhancements.go:524-533).
- Passwords bcrypt DefaultCost (device_service.go:84,201). Both in SQLite (user_sessions/users).
- Loss: none permanent (one-way); lost DB → re-login/reset, not data loss.

## 8. Invoice integrity HMAC key
- computeDocumentHMAC() customer_invoice_service.go:36 — HMAC-SHA256 keyed by globalFieldCrypto.salt (:45). Falls back to UNKEYED plain SHA-256 if FieldCrypto down (:52-56). Consumed in ZATCA XML <InvoiceHash> (einvoice_service.go:67-93).
- Key = the .field_crypto_salt bytes (§2). Salt change → stored invoice_hash values fail verification (integrity-signal loss, not data loss). Backfill guarded against poisoning (:80-90).

## 9. Mistral AI API key
- getMistralAPIKey() butler_ai.go:1313: (1) encrypted DB setting apiKeys.mistral_key via FieldCrypto (app.go:717-733); (2) settings.json at <dbDir>/settings.json (app.go:735-740, path app_setup_documents_surface.go:38); (3) MISTRAL_API_KEY env (butler_ai.go:1338). Persisted encrypted by SetAPIKeys (app_setup_documents_surface.go:636-638).
- Protects Butler AI chat + OCR (sole AI provider post-Wave 13). Loss = FULLY re-issuable (mint new key). No data loss.

## 10. License keys + developer master key
- License keys PH-{ROLE}-{6hex}, 3 crypto-random bytes (license_service.go:220-224), stored license_keys table, device-hash bound (:46-58). Hardcoded named/example seed keys :783-807 (demo, not secrets).
- Developer master key: masterKey = os.Getenv("ASYMMFLOW_MASTER_KEY") (:111), EMPTY by default, gated by ENABLE_DEVELOPER_MASTER_KEY (:194-199,313-320); deliberately never hardcoded (:108-111); grants admin, device-independent, unrevocable (:633-635).
- Env var (master) / DB (issued). Protects app access/RBAC, not data. Loss = re-issuable (seedLicenseKeys :685).

## 11. Mesh (P2P) key material — Node/Bare under mesh/
- Device identity seed: persistentSeed() kit-host.mjs:67 — 32 random bytes written PLAINTEXT hex to data/keys/device-seed.hex (:69-70); expands to device Ed25519 keypair via deviceKeys() capability.mjs:116; actor label data/keys/actor.txt (:114).
- Room keys + content encryption keys: {roomKey, authorityPub, encryptionKey(hex), bootstrap(hex)} persisted PLAINTEXT in data/keys/rooms.json (kit-registry.mjs:1-7,46-52); content key = randomBytes(32) (kit-repl.mjs:225), into Autobase (mesh-node.mjs:60-61,77,86,130).
- Authority/invite keys: Ed25519 authority keypair signs cap grants/revokes/epochs (capability.mjs:133-145); invite keys sign redemption proofs (:152-183).
- data/keys/ plaintext on disk, NOT DPAPI-wrapped. Protects mesh device identity, room membership, e2e message encryption of the org mirror.
- Loss: device-seed.hex lost = device loses mesh identity (re-invite); room encryptionKey lost = that room's mirrored history undecryptable. See "unrecoverable by design".

## 12. Ambient env/.env secrets (not app-generated)
Loaded by LoadConfig() config.go:187 (exe-dir → CWD → DataDir .env, :190-209): Azure AZURE_TENANT_ID/CLIENT_ID/CLIENT_SECRET (:230-232); Supabase SUPABASE_DB_PASSWORD/SERVICE_KEY, DATABASE_URL (embedded pw parsed :560-568,635-660); ENCRYPTION_MASTER_KEY (§1), MISTRAL_API_KEY (§9), ASYMMFLOW_MASTER_KEY (§10). All re-issuable pass-through creds; masked in logs (:393-395,460-467).

## FINDINGS (flag list — custody map documents; crypto fixes are out-of-wave)
1. Deterministic weak fallback keys. Hardware-ID failure keys FieldCrypto off literal "fallback-key-ace-engine" (field_crypto.go:68), OAuth cache off "fallback-key-asymmetrica-auth" (auth_handler.go:858), SettingsService off "fallback-key-ace-engine" (settings_service.go:49). Any machine hitting fallback derives an IDENTICAL PUBLICLY-KNOWN key — data there is effectively unprotected. Only a warning logged.
2. Two hardcoded static salt strings in the binary: "asymmetrica-salt-2025" (settings_service.go:54), "ph-holdings-auth-salt-2026" (auth_handler.go:862). With hostname fallback, reduces OAuth-cache/legacy-settings key to guessable inputs on a fallback machine.
3. Mesh key material plaintext on disk. device-seed.hex, rooms.json (room keys + content encryptionKey hex), authority/invite keys under data/keys/ are NOT DPAPI-wrapped (unlike the Go hardware-ID sidecar). Copying data/keys/ off the machine = full mesh identity + room decryption.
4. DPAPI machine scope not user scope. .hardware_id.dpapi uses CRYPTPROTECT_LOCAL_MACHINE (hardware_id_keystore_windows.go:49) → any local user/process can unprotect. Documented as intentional (:16-20).
5. Partial identifiers logged. Device HASHES logged truncated to 16 chars (license_service.go:302,341,443; device_service.go:307). Hashes not keys (low risk); master-key device-transfer CRITICAL line logs two device-hash prefixes (:341). No actual key material logged — masker redacts key-shaped fields (security_enhancements.go:470-511).
6. No automatic FieldCrypto key escrow. Only recovery for highest-value data (employee PII, bank/IBAN) is a manual ExportKeyMaterial+ExportSalt (field_crypto.go:270,277) a human must run in advance. No code auto-backs-up → unprepared hardware failure = permanent PII loss.

## UNRECOVERABLE BY DESIGN (mesh true-deletion doctrine)
- Social rooms & DMs: MESSENGER_DESIGN_CONSTITUTION.md:47-52 — "forgetting the key is true deletion of the room"; Article V (:143) makes mutual/forced forgetting a feature. A discarded social-room encryptionKey (rooms.json) makes that room's content INTENTIONALLY permanently unrecoverable — the designed privacy guarantee, not a failure.
- Crypto-epoch re-key chains: revocation mints a successor Autobase with new bootstrap + encryption key (:59-65); a discarded predecessor key permanently seals that epoch's history by design.
Map these as "unrecoverable by design — do not escrow". Everything in §1-§10 is the opposite: should be escrowed, but today only FieldCrypto has even a manual path.

---

# Part B — Data-at-Rest Inventory for Disaster Restore (agent: inv-data)

Headline: the SQLite DB is well-covered by backup machinery, but **three at-rest surfaces sit outside it, and one (field-crypto key material) can silently render the backed-up DB undecryptable on a restored machine.**

## 0. Engine & deployment layout
- **Primary engine: SQLite** (pure-Go `ncruces/go-sqlite3`, CGO banned) `config.go:42`, `contract.go:14`. No server DB in the primary path.
- **Postgres/Supabase is an OPTIONAL cloud-sync target, off by default** (`ENABLE_CLOUD_SYNC=false`) `config.go:544,574`.
- **Three-plane doctrine** `paths.go:12-17`: Code (exe dir, installer-replaced), Identity (`<slugRoot>\identity`), Data (`<slugRoot>\data`, never touched by installer).
- **Data plane root (Windows):** `%APPDATA%\Asymmetrica\<slug>\data`; else `~/.local/share/asymmetrica/<slug>/data` `paths.go:113-138`. Slug defaults `AsymmFlow-Dev` `paths.go:33`, overridden by exe-adjacent `deployment.json` `paths.go:102-108`.
- Legacy `%APPDATA%\AsymmFlow` is structurally never used (Mission DP1) `paths.go:9-10`, `app.go:202`. **STALE-DOC FLAG:** `docs/ops/BACKUP_RESTORE_PREFLIGHT_V0_1.md:29,37` still points operators at `$env:APPDATA\AsymmFlow\ph_holdings.db` — a dead path; the preflight script targets the wrong file on a current deployment.

## Surface 1 — Primary ERP database (SQLite)
- **What:** `ph_holdings.db`, ~90 GORM tables (CRM, finance/GL, inventory, procurement, payroll, banking SSOT, chat, sync bookkeeping). Catalog `database.go:27-443`.
- **Where:** total 3-step resolver `paths.go:158-176`: (1) `PH_DB_PATH` env; (2) `portable.flag` next to exe → `<exeDir>\data\ph_holdings.db`; (3) default `DataDir()\ph_holdings.db`. Filename `DBFileName` `paths.go:41`.
- **Live siblings:** `-wal`/`-shm` (WAL mode); restore/seed delete stale journals `contract.go:558-566`, `backup.go:176-177`.
- **Schema source:** GORM AutoMigrate from the compiled binary, NOT SQL files `app.go:549-572`. `migrations/` has only `003_performance_indexes.sql` — not the schema of record.
- **Covered by backup:** YES.
- **Restore must verify:** `PRAGMA integrity_check`=ok; `PRAGMA foreign_key_check`=0 rows (`BACKUP_RESTORE_PREFLIGHT_V0_1.md:44-45`); schema-stamp row `settings.key='schema_version'` (`contract.go:28,417`) ≤ restoring binary's schema else app REFUSES to open (downgrade refusal `contract.go:229-236`). Sentinels: row counts on `customers`/`invoices`/`payments`/`orders`; `settings` table must exist.

## Surface 2 — Existing backup machinery (pkg/infra/db/backup.go)
- **Engine:** `Backuper.Backup()` = `VACUUM INTO` (atomic), chmod 0600, `<dbDir>\backups\<stem>_<ts>.db`, prune to Keep=7 `backup.go:62-95`, `database.go:449`.
- **Two writers into same `backups\` dir:** (1) runtime rotating backup `database.go:470-491` (keep 7); (2) pre-migrate/pre-reseed file copy `pre-migrate-<ts>.db` `contract.go:337-353` (keep 5).
- **Triggers:** at startup `app.go:619` (frequency-gated); manual `TriggerBackup`/`BackupDatabase` `database.go:463,635`; automatically before migrate/`PH_FORCE_RESEED` `contract.go:188,250`.
- **Policy in DB settings table:** `backup_auto_enabled` (default TRUE), `backup_frequency_days` (7, clamp 1-30), `backup_last_at`, `backup_last_path` `database.go:451-544`. A separate `EnableAutoBackup` env flag (default false) `config.go:247` does NOT gate startup backup — easy confusion, two mechanisms.
- **Restore code EXISTS but is UNWIRED:** `db.Restore()` `backup.go:160-179` + `db.VerifyBackup()` `backup.go:129-146` do verify→snapshot→swap, but NO `App` method calls `db.Restore`. Restore today = manual file copy per the ops doc `:59-65`, no in-app button.
- **Verification gap:** backups verified on RESTORE only; NOT verified at CREATION — a bad `VACUUM INTO` sits in rotation until someone restores it.
- **Colocation gap:** all 12 backups live in `<dbDir>\backups\` on the same disk/machine as the live DB. Ops doc says keep one off-machine (`:65`) but NO code performs an off-machine copy. Disk loss = DB + all backups gone.

## Surface 3 — Cloud sync (usable backup? NO)
- ~70-table allow-list `db_sync_service.go:64-131`; merge-only, never deletes `:28,:19`.
- **Partial, not a replica.** NOT synced: `settings`, `sync_records`, `audit_logs`, `alerts`, `jobs`, `sync_status`, `devices`, `chart_of_accounts`/journals, chat tables, prediction records. Cannot resurrect a machine (no settings/schema-stamp/GL/audit/crypto).
- Conflict bookkeeping in local `sync_records` `database.go:318-329` (itself not synced). No round-trip test (`:70`).
- **Verdict:** partial business-data mirror; NOT a DB-backup substitute.

## Surface 4 — Field-crypto key material (CRITICAL GAP)
Encrypted fields (employee visa/CPR/permit + other sensitive values) are AES-256-GCM keyed from TWO inputs OUTSIDE the DB and OUTSIDE backup:
1. **`.field_crypto_salt`** — 32 random bytes, exe-adjacent first else `DataDir()` `field_crypto.go:341-402`; no deterministic fallback `:75-78`.
2. **Master secret** — `ENCRYPTION_MASTER_KEY` env, else the machine hardware ID `field_crypto.go:60-71`.
3. **Hardware-ID persistence** — sidecar `.hardware_id` (plaintext, legacy) + DPAPI sibling `.hardware_id.keystore`, next to DB `settings_service.go:236-244,257`, `hardware_id_keystore_windows.go`. Fresh installs write ONLY the keystore, no plaintext `hardware_id_keystore_test.go:59-71`.

**DR failure mode:** DPAPI blob is `CRYPTPROTECT_LOCAL_MACHINE`-scoped `hardware_id_keystore_windows.go:49` — undecryptable on a different machine. On restore to new hardware, `resolveHardwareID` `settings_service.go:417-435` fails the keystore, falls back to plaintext `.hardware_id` if present, else derives a NEW id → NEW key → **all encrypted fields permanently undecryptable.** A machine restored from DB backup alone, no `ENCRYPTION_MASTER_KEY` and no plaintext `.hardware_id`, loses encrypted data even though rows restored fine.
- **NOT covered by backup** — `Backuper` copies only the `.db`.
- **Restore must verify:** `ENCRYPTION_MASTER_KEY` set to original OR both `.field_crypto_salt` + original `.hardware_id` restored. Recovery API exists: `ExportKeyMaterial()`/`ExportSalt()`→`ImportKeyMaterial()` `field_crypto.go:270-309` (manual, no auto hook). Sentinel: decrypt one known `employee_documents` field post-restore.

## Surface 5 — Documents, exports, reports, inbox
- **OCR/intake inbox:** watched filesystem folder `InboxPath=<basePath>\<company>\Inbox` (OneDrive-rooted) `app_setup_documents_surface.go:989,1944,2131-2163`. Raw files OUTSIDE data plane and backup; classified records land in DB intake tables.
- **exports/reports/batch_output:** under CWD or `DataDir()` when CWD unwritable `app.go:199-224`. Transient, not backed up, regenerable.
- **Attachments:** `expense_attachments`/`costing_sheet_attachments` are DB tables `expense_service.go:48`, `costing_attachment_service.go:53` → ride inside `.db` backup. EXCEPTION: local-file-path attachments pointing at external files `costing_attachment_service.go:366`, temp `os.TempDir()\asymmflow-costing-attachments:404`. `BankStatementFile` archives PDFs/CSVs in-DB `database.go:439-440` (covered).

## Surface 6 — Config/license state to resurrect a machine
- **Settings store = DB `settings` table** `database.go:193` (schema stamp, backup policy, sync settings, encrypted secrets). In DB backup, NOT in cloud sync.
- **`.env`** — exe-adjacent/CWD/`DataDir()\.env` `config.go:190-207`; holds `ENCRYPTION_MASTER_KEY`, `DATABASE_URL`/`SUPABASE_*`, Azure creds, OneDrive paths. NOT backed up — manual resurrect. Store of record for master key when env path used.
- **`deployment.json`** (exe-adjacent) sets the slug keying the whole data plane `paths.go:52,77-90`. Wrong/lost → app resolves a different dir and appears to have "lost" all data.
- **`config.json`** (exe-adjacent) `pkg/config/config.go:79` — watcher/log tuning, non-critical.
- **License/device:** `license_keys` table (in DB, IS synced `db_sync_service.go:99`); `devices`/`device_users` (in DB, NOT synced `database.go:227-231`); hardware-ID sidecars (Surface 4). New hardware changes device identity, may require re-licensing.

## Surface 7 — Mesh data ("the folder IS the data")
Root `./data` or `--data DIR` `kit/kit-host.mjs:212`; deployed anchor `$kitDir\data` under scheduled task `AsymmFlowMeshAnchor` `kit/install_anchor.ps1:24-34`. Two sibling subtrees:
```
data/
├── keys/                       # identity — never inside corestore
│   ├── device-seed.hex         # Ed25519 device identity  (kit-host.mjs:113)
│   ├── actor.txt               # device actor label       (kit-host.mjs:114)
│   ├── rooms.json              # room registry            (kit-registry.mjs:32)
│   └── anchor.log              # anchor heartbeat         (anchor.mjs:133)
└── corestore/                  # Corestore/Autobase/Hypercore (kit-host.mjs:109)
    ├── social-<uuid>/          # rooms created            (bridge-server.mjs:355)
    └── joined-<key16>-<uuid>/  # rooms joined             (bridge-server.mjs:390)
```
Resurrection = same corestore subdir + same keys/ seed + matching rooms.json entry. Delete keys/ → identity gone; delete corestore subdir → room history gone. `mesh/.gitignore:36-40` ignores the whole `data/` plane; keys/ "holds real key material." **NOT covered by any AsymmFlow backup** — the Go engine knows nothing about `mesh/data`.

## Coverage matrix
| Surface | Location (deployed) | In DB backup? | In cloud sync? | Restore assertion |
|---|---|---|---|---|
| SQLite DB | `%APPDATA%\Asymmetrica\<slug>\data\ph_holdings.db` | YES | partial | integrity ok; fk 0; schema≤binary; row counts |
| `backups\` rotation | `<dbDir>\backups\*.db` | is the backup | NO | same-disk only, no off-machine copy |
| `.field_crypto_salt` | exe-dir or `<dataDir>` | NO | NO | present + original bytes |
| `.hardware_id[.keystore]` | `<dbDir>` | NO | NO | keystore machine-bound; need plaintext or env key |
| `.env` (master key/creds) | exe-dir/CWD/`<dataDir>` | NO | NO | `ENCRYPTION_MASTER_KEY` matches original |
| `deployment.json` | exe-dir | NO (Code plane) | NO | correct slug → correct data plane |
| settings table | inside DB | YES | NO | schema stamp + backup policy survive |
| OneDrive inbox/source docs | client OneDrive | NO | NO | regenerable via re-import |
| exports/reports/batch_output | CWD or `<dataDir>` | NO | NO | transient, regenerable |
| mesh keys+corestore | `<kitDir>\data` | NO | NO | device-seed + rooms.json + corestore subdir |

## Top risks for the drill
1. **Encrypted-field wipe on new hardware** (Surface 4) — highest-impact latent failure; DB restore "succeeds" while encrypted fields are dead.
2. **All backups colocated with live DB** (Surface 2) — no off-machine copy in code; disk loss = total loss.
3. **Backups unverified at creation** — bad `VACUUM INTO` surfaces only at restore.
4. **No in-app restore command** — `db.Restore()` exists but unwired; manual swap governed by a doc citing the dead `%APPDATA%\AsymmFlow` path.
5. **Cloud sync mistaken for a backup** — partial, merge-only; cannot resurrect settings/GL/audit/crypto.

---

## Gate CW1-0 verdict (orchestrator)

- Inventory doc exists with file:line evidence throughout: **PASS**
- Explicit "NOT covered by existing backup" list: **PASS** (coverage matrix + Part B
  surfaces 4-7)
- Both agents reported completeness with named limits (per-deployment rooms.json
  contents are data, not code): **PASS**
- Findings that are OUT of wave scope (crypto semantics) are recorded for the owner,
  not silently fixed: **PASS** — Part A findings 1, 2, 3, 4 go to the wave report's
  stop-and-report ledger.
