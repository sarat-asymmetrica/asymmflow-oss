# Recovery Envelope Template

**What this is:** the exact set of fields the owner (or whoever is the steward at the
time) writes down and seals **offline** — a password manager entry, a printed sheet in
a locked drawer, or a hardware token. This document is a TEMPLATE: it names fields and
where to find the values. **It must never be filled in and committed to this repo, or
stored anywhere the repo's contributors/CI can see it.** The repo carries the shape of
the envelope, never its contents.

See `docs/custodian/KEY_CUSTODY.md` for the full custody map this envelope backs, and
`docs/custodian/CW10_INVENTORY.md` for the underlying key-surface inventory.

**Grep-gate note:** this file itself was checked for anything resembling real key
material before commit — see `CW1A_REPORT.md`'s grep transcript. If you fill this
template out for real, fill out a COPY outside the repository, never this file in
place.

---

## How to use this template

1. Copy this file's field list (not this file itself) into your offline vault.
2. Fill in every field marked **[FILL]**. Leave nothing as a TODO in the real
   envelope — an incomplete envelope is worse than none, because it creates false
   confidence.
3. Re-verify the envelope after every rotation event (key rotation, hardware change,
   re-installation) — stale entries are a trap.
4. Anyone who could plausibly need to invoke this (the owner, and whoever the owner
   names as a successor steward per a future C8 bus-factor wave) should know the
   envelope exists and where it lives — but the envelope's *contents* should be as
   access-restricted as the data it protects.

---

## Section 1 — FieldCrypto master key material (the highest-priority field)

Governs: employee CPR/passport/visa/permit numbers, bank accounts/IBANs, encrypted
settings (including the Mistral API key), invoice integrity hashes. See
`KEY_CUSTODY.md` §1/§2/§8.

| Field | Value | How to obtain |
|---|---|---|
| `master_key_hex` | **[FILL]** | `App.ExportEncryptionBackup()` via the Wails DevTools console procedure in `KEY_CUSTODY.md` → "Owner ritual" |
| `salt_hex` | **[FILL]** | same call, `salt_hex` field of the returned object |
| `key_version` | **[FILL]** | same call, `key_version` field (which AES key version is currently active) |
| Date exported | **[FILL, ISO 8601]** | the day you ran the export |
| Exported by (name/role) | **[FILL]** | who ran it — for audit cross-reference against `GlobalAuditLogger`'s `encryption_key_exported` entry |
| Deployment this envelope is for | **[FILL — machine name / slug]** | the `deployment.json` slug (see Section 5) this key material belongs to; an envelope from one deployment must never be imported into another |

**Re-export whenever:** `Rotate()` is called (key version changes — the OLD version's
key stays needed to decrypt old data, so re-export captures the current master/salt,
which is unaffected by rotation since rotation derives new *versions* from the same
master+salt, per `field_crypto.go:230-244`) — in practice, the master/salt pair rarely
changes once set; re-export mainly matters after a fresh install or a deliberate key
replacement.

## Section 2 — Plaintext `.hardware_id` value (custody decision required)

Governs: the fallback path if `ENCRYPTION_MASTER_KEY` is not in use — see
`KEY_CUSTODY.md` §3.

**Owner decision needed (not made by this wave):** does this deployment rely on the
hardware-ID-derived key (no `ENCRYPTION_MASTER_KEY` env var set), or on an explicit
`ENCRYPTION_MASTER_KEY`? If the former, the plaintext hardware ID value itself becomes
sensitive recovery material (it's one of the two inputs FieldCrypto would need to
re-derive the *same* key on a truly identical restore, though in practice
`ImportKeyMaterial` from Section 1 supersedes needing this at all). Record the
decision, not just the value:

| Field | Value |
|---|---|
| This deployment's key source | **[FILL: "ENCRYPTION_MASTER_KEY env var" or "hardware-ID fallback"]** |
| If hardware-ID fallback: plaintext `.hardware_id` value | **[FILL, or "N/A — Section 1 export supersedes this"]** |
| If hardware-ID fallback: is the plaintext sidecar file itself backed up anywhere? | **[FILL: yes/no + where]** |

**Recommendation:** treat Section 1's export as authoritative and sufficient; Section
2 exists so the *decision* is on record, not so a second, redundant secret is kept.

## Section 3 — `.env` reconstruction list

Governs: Azure, Supabase, Mistral, and `ASYMMFLOW_MASTER_KEY` credentials — see
`KEY_CUSTODY.md` §12. Per `CW10_INVENTORY.md` Surface 6, `.env` is **not** covered by
any backup. Record where each value can be RE-OBTAINED (these are mostly re-issuable,
so the envelope records the *recovery procedure*, not the secret itself, except where
noted):

| Field | Where to re-obtain (not the value) |
|---|---|
| `AZURE_TENANT_ID` / `AZURE_CLIENT_ID` / `AZURE_CLIENT_SECRET` | **[FILL — e.g. "Azure AD App Registration, tenant X, app name Y — regenerate client secret from Azure Portal"]** |
| `SUPABASE_DB_PASSWORD` / `SUPABASE_SERVICE_KEY` / `DATABASE_URL` | **[FILL — Supabase project dashboard, project ref]**. Note: `ENABLE_CLOUD_SYNC` is off by default (`config.go:544,574`); confirm whether this deployment uses cloud sync at all before treating this as required. |
| `MISTRAL_API_KEY` | **[FILL — Mistral console account]** — fully re-issuable, low urgency |
| `ASYMMFLOW_MASTER_KEY` | **[FILL: is `ENABLE_DEVELOPER_MASTER_KEY` even turned on for this deployment? If not, N/A.]** If in use: this is NOT re-derivable — it must be recorded directly, offline, with the same discipline as Section 1. |
| `ENCRYPTION_MASTER_KEY` (if this deployment sets it explicitly rather than relying on hardware-ID fallback) | Cross-reference Section 1 — this should be the identical value as `master_key_hex` there. Do not maintain two divergent copies. |

## Section 4 — Mesh device-seed / rooms.json copy location

Governs: `mesh/` P2P identity and room keys — see `KEY_CUSTODY.md` §11.
**Do NOT include social-room or DM room keys here** — see the DO-NOT-ESCROW list
below.

| Field | Value |
|---|---|
| Anchor machine `data/keys/` location | **[FILL — path on the deployed anchor machine]** |
| Is `data/keys/` copied anywhere off that machine? | **[FILL: yes/no + where + how often]** |
| `device-seed.hex` — recovery procedure if lost | **[FILL — re-invite procedure; this wave did not build/rehearse an automated one, see `CW1A_REPORT.md` residue]** |
| Anchored (work) room `encryptionKey` entries in `rooms.json` | **[FILL: list which anchored rooms exist and whether a second device/anchor also holds a copy of each room's key — see finding in `KEY_CUSTODY.md` §11 that no re-share/escrow procedure exists in code today]** |

## Section 5 — License reissue notes

Governs: `license_keys`, `deployment.json` slug — see `KEY_CUSTODY.md` §10 and
`CW10_INVENTORY.md` Surface 6.

| Field | Value |
|---|---|
| `deployment.json` slug for this machine | **[FILL]** — losing/mismatching this makes the app resolve a *different* data directory and appear to have lost everything, even though nothing was deleted |
| License key(s) issued to this deployment | **[FILL, or "re-issuable via `seedLicenseKeys`, not sensitive"]** |
| `ASYMMFLOW_MASTER_KEY` in use? | cross-reference Section 3 |

---

## DO-NOT-ESCROW list (explicit, so nobody "fixes" this later)

Per `mesh/docs/MESSENGER_DESIGN_CONSTITUTION.md` (RATIFIED 2026-07-18) and
`KEY_CUSTODY.md`'s "Unrecoverable BY DESIGN" section:

- **Social room and DM `encryptionKey`/`bootstrap` values in `rooms.json`.** These
  belong to the room's participants, not the organization. Writing them into this
  envelope would be a topological violation of Article II and an explicit breach of
  Article IV item 5 ("Admin export or covert membership in social rooms/DMs...
  prohibited as a product feature"). **Leave them out.**
- **Crypto-epoch predecessor keys** (superseded Autobase bootstrap/encryption keys
  after a revocation re-key). Their entire purpose is to seal an epoch's history.
  **Leave them out.**

If a future wave proposes escrowing either of the above, that is an amendment to the
Design Constitution requiring explicit owner ratification — not a custody-wave
default.
