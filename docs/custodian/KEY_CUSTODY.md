# Key Custody Map

**Wave:** Custodian 1 "The Existential Floor" · Mission CW1-A (C2 — key custody map + rehearsed recovery)
**Source of truth for the key list:** `docs/custodian/CW10_INVENTORY.md` Part A §1-§12
(inventoried on branch `feat/fable-custodian-w1` off main `00f7054`). This document adds
custody/recovery/rehearsal facts on top of that inventory; it does not re-derive the
key list.
**Rehearsal evidence:** `custodian_rehearsal_test.go` (module root, package main),
runnable via `go run ./scripts/custodian/rehearse_recovery` or
`go test -run 'TestCustodianRehearsal|TestScratchGuardRefusesUnsafePaths' -v .`.
Full transcript in `docs/custodian/CW1A_REPORT.md`.

**Legend for "recovered how":** MANUAL = a human must run an export/import step in
advance and store the output; AUTOMATIC = the system heals itself with no data loss;
NONE = permanent loss on the stated trigger; BY DESIGN = deliberately unrecoverable,
do not escrow.

---

## §1. FieldCrypto master key (primary at-rest key)

- **Lives where:** never persisted — derived in memory each process start
  (`field_crypto.go:60` `NewFieldCrypto()`). Inputs on disk: the `ENCRYPTION_MASTER_KEY`
  env var if set (§12 below), else the machine hardware ID (§3); strengthened via
  PBKDF2-SHA256 600k iterations (`field_crypto.go:40,99`) when the secret isn't already
  a 64-hex-char key.
- **Protects:** employee CPR/passport/visa/permit numbers
  (`employee_compliance_service.go:42,187,257`), bank account numbers + IBANs
  (`bank_accounts_service.go:460-461`), encrypted `settings` rows including the Mistral
  API key (`settings_service.go:114-116`), and supplies the invoice HMAC salt (§8).
- **Recovered how:** **MANUAL.** `ExportKeyMaterial()` (`field_crypto.go:270`) +
  `ExportSalt()` (`field_crypto.go:277`) must be run *in advance* and the two hex
  strings stored offline; restore via `ImportKeyMaterial(masterHex, saltHex)`
  (`field_crypto.go:285`). **This IS wired into the app as an operator-invocable
  binding** — `App.ExportEncryptionBackup()` / `App.ImportEncryptionBackup()`
  (`app_setup_documents_surface.go:815-888`), admin-only (`requirePermission("*")`),
  exposed in the Wails JS bindings (`frontend-lab/wailsjs/go/main/App.d.ts:481,1207`).
  **Finding:** no `.svelte` screen in `frontend-lab/src` calls either binding — there
  is no button/menu anywhere in the shipped UI today. An admin can only reach it via
  the Wails devtools console (`window.go.main.App.ExportEncryptionBackup()`) or a
  developer build. See the owner-ritual section below for the exact console procedure,
  and the residue list in `CW1A_REPORT.md` for the recommended UI fix (out of this
  wave's scope — no runtime code was touched).
- **Dies with it:** lose the master secret AND the salt (or lose either one alone) →
  every encrypted employee document number, bank account/IBAN, and encrypted setting
  is **permanently** unreadable; invoice integrity hashes stop verifying (§8).
- **Rehearsal evidence:** `TestCustodianRehearsal_FieldCrypto` — red (wrong master key,
  wrong/substitute salt, missing salt file all fail closed, content-asserted) then
  green (documented `ImportKeyMaterial` path recovers the sentinel byte-identical).
  See `CW1A_REPORT.md` §Transcript.

## §2. `.field_crypto_salt` (random salt)

- **Lives where:** `loadOrCreateSalt()` (`field_crypto.go:341`) — exe-adjacent
  directory first, else `deploy.DataDir()` (`%APPDATA%\Asymmetrica\<slug>\data`);
  32 random bytes (`field_crypto.go:385`), atomic write, mode 0600.
- **Protects:** it is the second required input to the FieldCrypto master key AND is
  used directly as the literal invoice-HMAC key (§8) — losing it is exactly as fatal
  as losing the master secret, even if the hardware ID is intact
  (`field_crypto.go:74-78`: no deterministic fallback exists on purpose).
- **Recovered how:** MANUAL — same `ExportSalt()`/`ImportKeyMaterial()` path as §1;
  there is no independent recovery path for the salt alone.
- **Dies with it:** identical consequence to §1 (they must be recovered together).
- **Rehearsal evidence:** covered by the same `TestCustodianRehearsal_FieldCrypto` run
  as §1 (the salt is exported/imported alongside the master key in every red and green
  case).

## §3. Hardware ID + at-rest sidecars (root identifier)

- **Lives where:** `resolveHardwareID()` (`settings_service.go:417`), live lookup at
  `:448` (Windows `Get-CimInstance Win32_BaseBoard`, wmic fallback, macOS `ioreg`,
  Linux `/etc/machine-id`, last-resort `os.Hostname()`). Persisted next to the SQLite
  DB as `.hardware_id.dpapi` (DPAPI-wrapped, Windows steady state) and/or `.hardware_id`
  (plaintext, pre-migration/non-Windows fallback, mode 0600).
- **Protects:** it is the root input for the FieldCrypto fallback (§1), the legacy
  SettingsService key (§5), and the OAuth token-cache key (§6) — one changed byte
  breaks all three.
- **Recovered how:** AUTOMATIC while the machine is unchanged (re-resolves each boot,
  memoized per process, `settings_service.go:215-220`). **NONE** across a genuine
  hardware change *except* via the FieldCrypto Export/Import path in §1 (which
  sidesteps the hardware ID entirely once `ENCRYPTION_MASTER_KEY` or exported material
  is in play).
- **Dies with it:** a real hardware change with no exported FieldCrypto material and no
  preserved `.hardware_id` sidecar → the app derives a *new* ID → a *new* key →
  everything keyed off the old ID becomes unreadable. This is the single largest
  custody risk in the system (per CW1-0 §Orchestrator synthesis item 1).
- **Rehearsal evidence:** not independently rehearsed as its own recovery path (there
  isn't one beyond §1's export/import) — see residue list in `CW1A_REPORT.md`.

## §4. Windows DPAPI keystore (at-rest wrapping)

- **Lives where:** `keystoreProtect`/`keystoreUnprotect`
  (`hardware_id_keystore_windows.go:38,60`), `CryptProtectData`/`CryptUnprotectData`
  with `CRYPTPROTECT_LOCAL_MACHINE | CRYPTPROTECT_UI_FORBIDDEN`. Wraps the
  `.hardware_id.dpapi` sidecar bytes only — it is not a new source of the identifier.
- **Protects:** at-rest confidentiality of the hardware-ID sidecar file against
  "copy the file off the machine and read it in a text editor." Does **not** protect
  against any local user/process on the *same* machine — `CRYPTPROTECT_LOCAL_MACHINE`
  is intentionally machine-scoped, not user-scoped (documented in the file's own header
  comment, `:16-20`).
- **Recovered how:** AUTOMATIC — a DPAPI failure (unavailable keystore, corrupted blob)
  silently falls back to the plaintext sidecar or live re-resolution
  (`settings_service.go:330-331`). No data loss from DPAPI failure alone.
- **Dies with it:** nothing dies with DPAPI loss by itself; it degrades gracefully to
  plaintext/re-resolution. What DOES NOT survive: moving the wrapped blob to a
  *different* physical machine — that machine's DPAPI master key differs, so the blob
  is permanently unreadable there (this is a feature of DPAPI's threat model, not a
  bug — the plaintext/live-resolution fallback is what actually saves the day on a
  restore-to-new-hardware scenario, not the DPAPI blob).
- **Rehearsal evidence:** `TestCustodianRehearsal_DPAPIKeystore` — same-machine
  protect/unprotect round-trip (green) and a corrupted-blob negative control (red,
  proxy for "foreign machine key"). **Residue:** true cross-machine DPAPI failure and
  same-machine profile-loss (different Windows user account) were NOT live-verified —
  see `CW1A_REPORT.md`.

## §5. Legacy SettingsService key (migration only)

- **Lives where:** `NewSettingsService()` (`settings_service.go:54`) —
  `SHA-256(hardwareID + "asymmetrica-salt-2025")`, in-memory only, never persisted.
- **Protects:** decrypts pre-FieldCrypto settings rows, transparently re-encrypting to
  FieldCrypto on read (`:138-146`).
- **Recovered how:** AUTOMATIC — fully re-derivable from the hardware ID (§3) at any
  time; not an independent secret.
- **Dies with it:** nothing new — its loss consequence is a strict subset of §3's.
- **Rehearsal evidence:** not rehearsed (re-derivable, no independent recovery
  procedure to prove).
- **Finding (out of wave scope, flagged not fixed):** the salt string
  `"asymmetrica-salt-2025"` is hardcoded in the binary — see Findings §1/§2 below.

## §6. OAuth (MS Graph) token cache key + `.auth_token.json`

- **Lives where:** `getEncryptionKey()` (`auth_handler.go:853`) —
  `SHA-256(hardwareID + "ph-holdings-auth-salt-2026")`. Encrypts `.auth_token.json`
  (CWD, mode 0600) holding the MS access/refresh token + profile.
- **Protects:** the cached Microsoft Graph session.
- **Recovered how:** AUTOMATIC — **re-issuable by re-login.** Tokens are also only
  ever stored hashed in `user_sessions` (§7).
- **Dies with it:** nothing permanent — worst case is one re-authentication.
- **Rehearsal evidence:** not rehearsed (fully re-issuable; no recovery procedure to
  prove — loss has no data-loss consequence).

## §7. Session tokens + password hashes (DB)

- **Lives where:** SQLite `user_sessions`/`users` tables. Sessions stored as
  `SHA-256(token)` only — plaintext token never persisted (`auth_session.go:34-43`).
  Passwords bcrypt `DefaultCost` (`device_service.go:84,201`).
- **Protects:** login sessions and password verification.
- **Recovered how:** AUTOMATIC (one-way hashes; nothing to "recover," only re-login or
  password reset).
- **Dies with it:** nothing — this data is expendable by design.
- **Rehearsal evidence:** not applicable (no recovery procedure exists or is needed).

## §8. Invoice integrity HMAC key

- **Lives where:** `computeDocumentHMAC()` (`customer_invoice_service.go:36`) — keyed
  by `globalFieldCrypto.salt`, i.e. literally the §2 salt bytes. Falls back to
  *unkeyed* plain SHA-256 if FieldCrypto is down (`:52-56`).
- **Protects:** the ZATCA e-invoice `<InvoiceHash>` integrity signal
  (`einvoice_service.go:67-93`).
- **Recovered how:** tied 1:1 to §2's recovery. If §1/§2 are recovered, this recovers
  with them (same salt bytes re-derive the same HMAC key).
- **Dies with it:** a salt change/loss makes *previously stored* `invoice_hash` values
  fail verification — an **integrity-signal loss**, not a data loss (the invoices
  themselves remain readable; only their tamper-evidence breaks). Backfill is guarded
  against poisoning (`:80-90`).
- **Rehearsal evidence:** covered transitively by §1/§2's rehearsal (same salt object).

## §9. Mistral AI API key

- **Lives where:** three places, in precedence order:
  (1) encrypted DB setting `apiKeys.mistral_key` via FieldCrypto (`app.go:717-733`);
  (2) `settings.json` at `<dbDir>/settings.json`
  (`app_setup_documents_surface.go:38,735-740`);
  (3) `MISTRAL_API_KEY` env var (`butler_ai.go:1338`).
- **Protects:** Butler AI chat + OCR — the sole AI provider post-Wave 13.
- **Recovered how:** AUTOMATIC — **fully re-issuable.** Mint a new key at the provider
  and re-enter it; no data is lost.
- **Dies with it:** nothing — worst case is Butler AI/OCR unavailable until a new key
  is entered.
- **Rehearsal evidence:** not rehearsed (re-issuable; no recovery procedure to prove).

## §10. License keys + developer master key

- **Lives where:** license keys `PH-{ROLE}-{6hex}` in the `license_keys` table,
  device-hash bound (`license_service.go:46-58,220-224`). Developer master key:
  `os.Getenv("ASYMMFLOW_MASTER_KEY")`, empty by default, gated by
  `ENABLE_DEVELOPER_MASTER_KEY` (`:111,194-199,313-320`) — deliberately never
  hardcoded.
- **Protects:** app access/RBAC gating. Not data-at-rest.
- **Recovered how:** AUTOMATIC/MANUAL-reissue — license keys are re-issuable
  (`seedLicenseKeys :685`); the master key is an env var the owner sets and knows, not
  a derived secret to "recover."
- **Dies with it:** nothing — access control, not data.
- **Rehearsal evidence:** not rehearsed (no data-loss consequence; no export/import
  procedure exists to prove).

## §11. Mesh (P2P) key material — `mesh/` (Node/Bare)

- **Lives where:** `<kitDir>\data\keys\` on the deployed anchor machine
  (`mesh/kit/kit-host.mjs:212`, `mesh/kit/install_anchor.ps1:24-34`):
  - `device-seed.hex` — 32 random bytes, **plaintext**, device Ed25519 identity seed
    (`kit-host.mjs:67-70`, expands via `capability.mjs:116`).
  - `actor.txt` — device actor label (`kit-host.mjs:114`).
  - `rooms.json` — room registry: `{roomKey, authorityPub, encryptionKey(hex),
    bootstrap(hex)}` per room, **plaintext** (`kit-registry.mjs:1-7,46-52`).
  - Authority/invite Ed25519 keypairs signing capability grants/revokes/epochs and
    invite redemption (`capability.mjs:133-183`).
  - None of `data/keys/` is DPAPI-wrapped, unlike the Go hardware-ID sidecar (§4) —
    plaintext on disk is the current state.
- **Protects:** mesh device identity, room membership, and end-to-end message
  encryption of the org mirror (anchored/work rooms — see §Unrecoverable-by-design
  below for social rooms/DMs).
- **Recovered how — ANCHORED (work) rooms:** device-seed loss = MANUAL re-invite (the
  device re-joins under a new identity; org-authority-anchored room history is
  recoverable by rejoining because authority lives with the org, not the device).
  Room `encryptionKey` loss for an **anchored** room = the org authority can, in
  principle, re-derive access because the org authority key is *in* the room manifest
  (Article II) — but no automated re-share/escrow procedure exists in code today; this
  is a MANUAL, undocumented-in-code gap. **Finding**, not fixed this wave (no mesh
  runtime code touched).
- **Dies with it (anchored rooms):** losing `data/keys/` on the last surviving device
  with no other member/anchor holding the room = that room's mirrored history is lost
  the same way a Signal/Matrix homeserver loses history if every server drops it —
  functionally permanent without a second copy.
- **Rehearsal evidence:** **NOT rehearsed this wave.** Driving the real mesh
  (Node/Bare, Autobase/Hypercore, `capability.mjs`) requires the JS/mesh runtime, which
  this Go-side harness does not invoke — recorded as residue. CW1-B (the restore
  drill) is scoped to cover "folder IS the data" copy-back for mesh; CW1-A defers the
  *key*-recovery rehearsal for anchored mesh rooms to that mission or a future wave.

## §12. Ambient `.env` / config secrets (not app-generated)

- **Lives where:** `LoadConfig()` (`config.go:187`, resolution order exe-dir → CWD →
  `DataDir()`, `:190-209`). Holds: Azure `AZURE_TENANT_ID`/`CLIENT_ID`/`CLIENT_SECRET`
  (`:230-232`); Supabase `SUPABASE_DB_PASSWORD`/`SERVICE_KEY`, `DATABASE_URL`
  (`:560-568,635-660`); `ENCRYPTION_MASTER_KEY` (§1); `MISTRAL_API_KEY` (§9);
  `ASYMMFLOW_MASTER_KEY` (§10). Also `deployment.json` (exe-adjacent, sets the slug
  keying the whole data plane, `paths.go:52,77-90`) and `config.json` (non-critical
  tuning).
- **Protects:** all of the above are re-issuable pass-through credentials except
  `ENCRYPTION_MASTER_KEY`, which — if this is the mechanism a deployment uses instead
  of the hardware-ID fallback — **is** the FieldCrypto master secret and must be
  treated with §1's custody discipline, not §9/§10's "just reissue it" discipline.
- **Recovered how:** MANUAL — `.env` is not backed up by any code path
  (`CW10_INVENTORY.md` Surface 6); an owner must keep an offline reconstruction list
  (see `RECOVERY_ENVELOPE_TEMPLATE.md`). Azure/Supabase/Mistral values are
  re-obtainable from their respective consoles; `ENCRYPTION_MASTER_KEY`, if in use, is
  **not** re-obtainable — it must be in the envelope or it is permanently lost (same
  fate as §1).
- **Dies with it:** losing `.env` alone is only fatal for the `ENCRYPTION_MASTER_KEY`
  value (if that's the deployment's chosen path over hardware-ID fallback) and for
  `deployment.json`'s slug (wrong/lost slug → app resolves a *different* data
  directory and appears to have "lost" everything, even though nothing was deleted —
  see `CW10_INVENTORY.md` Surface 6).
- **Rehearsal evidence:** the `ENCRYPTION_MASTER_KEY` case is exactly what
  `TestCustodianRehearsal_FieldCrypto` rehearses (§1). The Azure/Supabase/Mistral/slug
  cases are not independently rehearsed (either fully re-issuable, or "restore the
  string to a file" with no crypto to prove).

---

## Unrecoverable BY DESIGN — DO NOT ESCROW

Per `mesh/docs/MESSENGER_DESIGN_CONSTITUTION.md` (RATIFIED 2026-07-18), these are
**deliberately** impossible to recover. Building escrow for them would be a regression
against a ratified owner values ruling, not a fix for a gap:

- **Social rooms & DMs (Article II, lines 47-52).** "Authority = the participants
  themselves. The org authority key is *not in the room* — there is no admin who can
  be granted in later, because grants come from the room's own authority plane...
  every participant discarding the base and forgetting the key is true deletion of the
  *room*." Article IV item 5 explicitly bans "Admin export or covert membership in
  social rooms/DMs" as "impossible by topology... prohibited as a product feature
  besides." Article V item 6: "Room disposability never destroys another's evidence:
  'delete the room' is each owner discarding their *own* copy. Mutual forgetting is
  possible; forced forgetting is not."
- **Crypto-epoch predecessor keys (Article II amendment, lines 59-65).** A
  revocation-driven re-key mints a *successor* Autobase with a new bootstrap key and a
  new encryption key; a discarded predecessor key permanently seals that epoch's
  history by design. "One room, one Autobase" holds *within* an epoch, not across a
  re-key — the chain is meant to have sealed segments.

**Nobody "fixes" this later without an owner amendment to the Design Constitution.**
It is the whole point of the social/DM room class.

---

## FINDINGS (documented, not fixed — out of this wave's scope per spec §0)

Carried forward from `CW10_INVENTORY.md`'s findings list, with the custody
implication spelled out:

1. **Deterministic weak fallback keys.** If hardware-ID resolution fails, FieldCrypto
   falls back to the literal string `"fallback-key-ace-engine"` (`field_crypto.go:68`),
   OAuth cache to `"fallback-key-asymmetrica-auth"` (`auth_handler.go:858`),
   SettingsService to `"fallback-key-ace-engine"` (`settings_service.go:49`). Any
   machine that ever hits this fallback derives an **identical, publicly-known key** —
   data protected under it is effectively unencrypted. Only a log warning fires.
   Custody implication: this is a silent failure mode with no distinct "key" to
   custody — the fix is a crypto-semantics change (stop-and-report territory), not a
   documentation fix.
2. **Two hardcoded static salt strings in the binary:** `"asymmetrica-salt-2025"`
   (§5) and `"ph-holdings-auth-salt-2026"` (§6). Combined with hostname fallback, this
   reduces the OAuth-cache/legacy-settings key to guessable inputs on a fallback
   machine. Low custody impact today (both are migration-only/re-issuable surfaces)
   but flagged for the record.
3. **Mesh key material is plaintext on disk, not DPAPI-wrapped** (§11) — unlike the Go
   side's hardware-ID sidecar. Copying `data/keys/` off the machine is full mesh
   identity + room decryption in one file copy.
4. **DPAPI machine scope, not user scope** (§4) — intentional per the code's own
   header comment, documented here so it isn't mistaken for an oversight.
5. **Partial identifiers logged.** Device hashes truncated to 16 chars appear in logs
   (`license_service.go:302,341,443`, `device_service.go:307`) — hashes, not keys, so
   low risk; the masker redacts key-shaped fields (`security_enhancements.go:470-511`).
6. **No automatic FieldCrypto key escrow.** The only recovery path for the
   highest-value data (employee PII, bank/IBAN) is the MANUAL export a human must run
   *in advance* — and per this wave's finding, has no UI button to run it from. See
   the owner ritual below.

---

## Owner ritual — what to run TODAY on the deployed PH machine

**Finding stated plainly first:** `ExportEncryptionBackup()` / `ImportEncryptionBackup()`
are real, wired Wails app bindings (`app_setup_documents_surface.go:815-888`,
`frontend-lab/wailsjs/go/main/App.d.ts:481,1207`) — **but no screen in the shipped
frontend calls them.** There is no Settings-page button today. Until one is built (see
residue in `CW1A_REPORT.md`), the only way an operator can run this is the developer
console procedure below. This is a genuine gap for a non-technical steward — recorded
honestly, not worked around by touching runtime code (out of this wave's scope).

### Developer-console procedure (works today, requires DevTools access)

1. Launch the AsymmFlow desktop app and log in as a user whose role grants the `*`
   (admin/wildcard) permission — `requirePermission("*")` gates both calls.
2. Open the Wails webview's DevTools (in a dev build: right-click → Inspect, or the
   Wails dev server's browser tab if running `wails dev`; a packaged production build
   may have DevTools disabled — see note below).
3. In the DevTools Console, run:
   ```js
   const backup = await window.go.main.App.ExportEncryptionBackup();
   console.log(backup);
   ```
4. This returns `{ master_key_hex, salt_hex, key_version, warning }`. **Write down
   `master_key_hex` and `salt_hex` by hand** (or copy to a file you will immediately
   move off-machine) using the field names in `RECOVERY_ENVELOPE_TEMPLATE.md`. Do
   **not** leave them in browser console scrollback, clipboard history, or a
   screenshot tool's history.
5. Seal the envelope per `RECOVERY_ENVELOPE_TEMPLATE.md` (password manager entry,
   printed and locked away, or hardware token — owner's choice, but offline and not in
   this repo).
6. This action is audit-logged (`GlobalAuditLogger`, `"encryption_key_exported"`,
   `app_setup_documents_surface.go:833-841`) — expect an audit trail entry.

**To recover on a new/repaired machine**, once the app is installed and running there:
```js
await window.go.main.App.ImportEncryptionBackup(masterKeyHex, saltHex);
```
using the exact strings written down in step 4. This is also audit-logged
(`"encryption_key_imported"`).

**Note on packaged builds:** if the production Wails build disables DevTools (common
for release builds), this procedure is not runnable at all today on that build without
a developer re-enabling DevTools or a debug binary. **This is itself a finding**,
recorded here and in `CW1A_REPORT.md`'s residue list — verifying DevTools
availability on the actual deployed PH production binary was out of scope for this
dev-machine-only wave (would require testing on the live PH machine, which the spec's
copies-only doctrine forbids touching).

### What this does NOT cover

- It does not export `.env`, `deployment.json`, mesh keys, or the DB itself — those
  need the separate procedures in `RECOVERY_ENVELOPE_TEMPLATE.md` and CW1-B's
  restore runbook.
- It does not help with anchored mesh room key recovery (§11) — no equivalent
  export/import binding exists for the mesh side today.
