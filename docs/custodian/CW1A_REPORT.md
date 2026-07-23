# CW1-A Mission Report — Key Custody Map + Recovery Rehearsal (C2)

**Wave:** Custodian 1 "The Existential Floor"
(`FABLE_CUSTODIAN_SPEC_01_THE_EXISTENTIAL_FLOOR.md`) · **Mission:** CW1-A
**Branch:** `feat/fable-custodian-w1` · **Date:** 2026-07-23

## What was built

1. **`docs/custodian/KEY_CUSTODY.md`** — one section per key surface from
   `CW10_INVENTORY.md` Part A (§1-§12), plus the mesh §11 keys and the ambient
   `.env` secrets, each with: lives-where, protects-what, recovered-how, dies-with-it,
   and a rehearsal-evidence pointer (or an explicit "not rehearsable — residue"
   note). Includes the DO-NOT-ESCROW section citing
   `mesh/docs/MESSENGER_DESIGN_CONSTITUTION.md` verbatim, and an "owner ritual"
   section with the exact developer-console steps to run
   `ExportEncryptionBackup()`/`ImportEncryptionBackup()` today.
2. **`docs/custodian/RECOVERY_ENVELOPE_TEMPLATE.md`** — five named-field sections
   (FieldCrypto master key material, hardware-ID custody decision, `.env`
   reconstruction list, mesh device-seed/rooms.json, license/slug notes) plus the
   explicit DO-NOT-ESCROW list. Zero material values — every field is `[FILL]`.
3. **`custodian_rehearsal_test.go`** (module root, `package main`) — the rehearsal
   harness, run via `go test -run 'TestCustodianRehearsal|TestScratchGuardRefusesUnsafePaths' -v .`
4. **`scripts/custodian/rehearse_recovery/main.go`** — a `go run`-able wrapper
   (`go run ./scripts/custodian/rehearse_recovery`) that shells out to the `go test`
   invocation above and streams it, satisfying the literal invocation named in the
   mission brief without needing to import unexported `package main` symbols from a
   second package (which Go does not allow).
5. **This report.**

## Why the harness is a `_test.go` file, not a standalone `go run` program

`FieldCrypto`, `ImportKeyMaterial`, `keystoreProtect`/`keystoreUnprotect`,
`keystoreAvailable` are unexported symbols of the module-root `package main`
(confirmed: `field_crypto.go:1` and `hardware_id_keystore_windows.go:3` are both
`package main`; `go.mod:1` shows module `ph_holdings_app` with these files at its
root). A separate `package main` under `scripts/custodian/` cannot import another
`package main`, and even a regular importable package could not see unexported
identifiers. The mission brief's own text anticipated this exact situation ("it may be
package main, in which case drive it via a test file in that package instead") — this
is the path taken, documented at the top of `custodian_rehearsal_test.go`.

## Red-then-green transcript (verbatim, this run)

```
=== RUN   TestScratchGuardRefusesUnsafePaths
=== RUN   TestScratchGuardRefusesUnsafePaths/inside_scratch_root
=== RUN   TestScratchGuardRefusesUnsafePaths/nested_inside_scratch_root
=== RUN   TestScratchGuardRefusesUnsafePaths/outside_scratch_root
=== RUN   TestScratchGuardRefusesUnsafePaths/names_ph_holdings.db
=== RUN   TestScratchGuardRefusesUnsafePaths/names_ph_holdings.db_mixed_case
=== RUN   TestScratchGuardRefusesUnsafePaths/contains_hash
--- PASS: TestScratchGuardRefusesUnsafePaths (0.00s)
    --- PASS: TestScratchGuardRefusesUnsafePaths/inside_scratch_root (0.00s)
    --- PASS: TestScratchGuardRefusesUnsafePaths/nested_inside_scratch_root (0.00s)
    --- PASS: TestScratchGuardRefusesUnsafePaths/outside_scratch_root (0.00s)
    --- PASS: TestScratchGuardRefusesUnsafePaths/names_ph_holdings.db (0.00s)
    --- PASS: TestScratchGuardRefusesUnsafePaths/names_ph_holdings.db_mixed_case (0.00s)
    --- PASS: TestScratchGuardRefusesUnsafePaths/contains_hash (0.00s)
=== RUN   TestCustodianRehearsal_FieldCrypto
    custodian_rehearsal_test.go:183: scratch root: C:\Users\schan\AppData\Local\Temp\custodian-rehearsal\20260723_133245_828404
    custodian_rehearsal_test.go:194: sentinel plaintext: "CUSTODIAN-SENTINEL-1784793765831666600"
    custodian_rehearsal_test.go:221: sentinel ciphertext (base64): ATHCw1o8NHuLcTFjgdpMbTM3EsFMm8rvLndOfmZh2LF5LlBMeLD6z4cp/MeP3CRbThwAK8MMuUh6UUjbIv9HZUkhJQ==
=== RUN   TestCustodianRehearsal_FieldCrypto/RED_wrong_master_key_fails
    custodian_rehearsal_test.go:243: confirmed RED: wrong master key -> Decrypt() error = field_crypto: decryption failed: cipher: message authentication failed (plaintext NOT recovered)
=== RUN   TestCustodianRehearsal_FieldCrypto/RED_missing_salt_file_means_missing_salt_hex_fails
    custodian_rehearsal_test.go:269: confirmed RED: missing/substitute salt -> Decrypt() error = field_crypto: decryption failed: cipher: message authentication failed (plaintext NOT recovered)
    custodian_rehearsal_test.go:280: confirmed RED: reading missing salt file -> open C:\Users\schan\AppData\Local\Temp\custodian-rehearsal\20260723_133245_828404\recovery-machine\salt-that-does-not-exist.hex: The system cannot find the file specified.
=== RUN   TestCustodianRehearsal_FieldCrypto/GREEN_documented_recovery_path_round_trips
    custodian_rehearsal_test.go:318: confirmed GREEN: recovered instance decrypted sentinel byte-identical: "CUSTODIAN-SENTINEL-1784793765831666600"
--- PASS: TestCustodianRehearsal_FieldCrypto (0.07s)
    --- PASS: TestCustodianRehearsal_FieldCrypto/RED_wrong_master_key_fails (0.00s)
    --- PASS: TestCustodianRehearsal_FieldCrypto/RED_missing_salt_file_means_missing_salt_hex_fails (0.00s)
    --- PASS: TestCustodianRehearsal_FieldCrypto/GREEN_documented_recovery_path_round_trips (0.06s)
=== RUN   TestCustodianRehearsal_DPAPIKeystore
    custodian_rehearsal_test.go:345: scratch root: C:\Users\schan\AppData\Local\Temp\custodian-rehearsal\20260723_133245_902369
=== RUN   TestCustodianRehearsal_DPAPIKeystore/same_process_round_trip_succeeds
    custodian_rehearsal_test.go:367: confirmed: DPAPI (CRYPTPROTECT_LOCAL_MACHINE) round-trips within this machine/session
=== RUN   TestCustodianRehearsal_DPAPIKeystore/RED_corrupted_blob_fails
    custodian_rehearsal_test.go:391: confirmed RED: corrupted DPAPI blob (proxy for cross-machine/foreign-key) -> error = keystoreUnprotect: CryptUnprotectData failed: The data is invalid.
=== NAME  TestCustodianRehearsal_DPAPIKeystore
    custodian_rehearsal_test.go:394: RESIDUE: true cross-machine DPAPI failure (same blob, different physical machine's DPAPI master key) cannot be simulated on one dev machine and is NOT claimed here — see CW1A_REPORT.md residue list. Likewise 'profile loss' (new Windows user account, same machine): CRYPTPROTECT_LOCAL_MACHINE scope means, per the source comment in hardware_id_keystore_windows.go:16-20, any local user/process on this machine can unprotect — so a same-machine profile loss is EXPECTED to still succeed by design, but no second Windows user account was available on this dev machine to verify live.
--- PASS: TestCustodianRehearsal_DPAPIKeystore (0.02s)
    --- PASS: TestCustodianRehearsal_DPAPIKeystore/same_process_round_trip_succeeds (0.02s)
    --- PASS: TestCustodianRehearsal_DPAPIKeystore/RED_corrupted_blob_fails (0.00s)
=== RUN   TestCustodianRehearsal_NoRealSecretsInThisFile
--- PASS: TestCustodianRehearsal_NoRealSecretsInThisFile (0.00s)
PASS
ok  	ph_holdings_app	0.385s
```

**Read the order:** for `TestCustodianRehearsal_FieldCrypto`, the two `RED_*`
subtests ran and asserted failure (`t.Fatalf` would have fired and failed the whole
run had decryption ever succeeded under a wrong key or wrong salt) *before*
`GREEN_documented_recovery_path_round_trips` proved the real recovery path
(`ImportKeyMaterial`) restores the sentinel byte-identical. Same order for the DPAPI
keystore test: same-process round-trip green, then the corrupted-blob red control.
This is Go subtest execution order as written in the source, not reordered for the
report — see `custodian_rehearsal_test.go:225-320` (FieldCrypto) and
`:352-393` (DPAPI).

Also verified independently as part of this mission: `go vet .` and `go vet
./scripts/...` are both clean (no output), and `go build ./...` exits 0 with the new
files present — the harness and wrapper compile cleanly alongside the full existing
tree.

## No-real-secrets grep (before finishing)

Ran against every file this mission added:

```
grep -rnoE '[0-9a-fA-F]{64,}' custodian_rehearsal_test.go \
  scripts/custodian/rehearse_recovery/main.go \
  docs/custodian/KEY_CUSTODY.md \
  docs/custodian/RECOVERY_ENVELOPE_TEMPLATE.md
```

Result: **no matches** (grep exit code 1 — "not found"). A second pass for
credential-shaped patterns (`-----BEGIN`, AWS-style `AKIA...`, inline
`hardware_id=<hex>`-style assignments) also found nothing. `custodian_rehearsal_test.go`
additionally self-checks this at test time
(`TestCustodianRehearsal_NoRealSecretsInThisFile`, PASS above) — every key/salt byte
string used anywhere in the harness is generated at runtime via `crypto/rand`, never
literal.

## Findings

1. **`ExportEncryptionBackup()`/`ImportEncryptionBackup()` exist and are wired as
   admin-gated Wails bindings, but have zero UI surface.** Grepped every `.svelte`
   file under `frontend-lab/src` for both names — no matches. The only generated
   references are the Wails JS binding stubs
   (`frontend-lab/wailsjs/go/main/App.d.ts:481,1207`,
   `frontend-lab/wailsjs/go/main/App.js:913,2365`, and duplicate bindings under
   `DocumentsService`). An operator today can only reach this through the Wails
   DevTools console (documented as the interim procedure in `KEY_CUSTODY.md`'s owner
   ritual) or a developer rebuild. **Recommendation for a future wave (not built this
   wave — no runtime/frontend code was touched, per the ADD-ONLY rule):** a Settings →
   Security screen with an "Export Recovery Key" button gated the same way
   (`requirePermission("*")`), with a confirmation dialog reiterating "store this
   offline, we cannot recover it for you."
2. **Whether DevTools is even reachable on the packaged production build was not
   verified.** This dev-machine-only wave never touched the deployed PH machine
   (copies-only doctrine, spec §1) and did not build/run a packaged release binary to
   check. If DevTools is disabled in release builds (common Wails/Electron practice),
   the console procedure in `KEY_CUSTODY.md` is not runnable there at all without a
   debug rebuild — flagged, not resolved.
3. **No re-share/escrow procedure exists in code for anchored (work) mesh room
   `encryptionKey` values.** Unlike FieldCrypto's Export/Import pair, there is no Go-
   or JS-side function that lets a second device/anchor re-derive or re-share an
   anchored room's key if the last holder loses `data/keys/`. This is distinct from
   the social-room/DM DO-NOT-ESCROW ruling — anchored rooms are *supposed* to be
   org-recoverable per Article II, but the mechanism doesn't exist yet. Recorded in
   `KEY_CUSTODY.md` §11 and here; not fixed (mesh runtime code, out of scope).
4. Six findings carried forward verbatim from `CW10_INVENTORY.md` (deterministic weak
   fallback keys, two hardcoded static salts, plaintext mesh key material, DPAPI
   machine-vs-user scope, partial-hash logging, no automatic escrow) are cross-
   referenced in `KEY_CUSTODY.md`'s Findings section rather than repeated in full
   here — all are crypto-semantics or logging changes, explicitly out of this wave's
   scope per spec §0's stop-and-report doctrine. None were touched.

## Stop-and-reports

None triggered. No change to encryption semantics or key derivation was needed to
build the custody map or rehearse the documented Export/Import path — the rehearsal
uses the exact existing functions (`ImportKeyMaterial`, `ExportKeyMaterial`,
`ExportSalt`, `keystoreProtect`, `keystoreUnprotect`) with zero modification. The six
crypto findings above (weak fallbacks, hardcoded salts, etc.) are documented per the
spec's explicit instruction ("the custody map documents each honestly") rather than
escalated as stop-and-reports, because the spec itself already classifies fixing them
as out-of-wave and pre-designates them for the wave report's findings ledger
(`CW10_INVENTORY.md` line 27-28, "recorded below... go to the wave report's
stop-and-report ledger").

## Residue — what could not be rehearsed on this dev machine, and why

| Residue item | Why not rehearsed here |
|---|---|
| True cross-machine DPAPI failure (same protected blob, a genuinely different machine's DPAPI master key) | One dev machine cannot host two distinct DPAPI machine-key contexts. Approximated with a bit-flipped-blob negative control (`RED_corrupted_blob_fails`), documented explicitly as a proxy, not equivalence. |
| Same-machine Windows-profile-loss DPAPI behavior (new user account, same machine) | No second Windows user account existed on this dev machine to create and test under. Reasoned from the source's own `CRYPTPROTECT_LOCAL_MACHINE` documentation (`hardware_id_keystore_windows.go:16-20`) that this should succeed by design, but "should" is not "verified live" — logged as residue in the test output itself. |
| `NewFieldCrypto()`'s own file-path resolution (`loadOrCreateSalt()` exe-adjacent/`DataDir()` candidates, the `ENCRYPTION_MASTER_KEY` env-var branch inside `NewFieldCrypto()`) | Not independently re-proven by this rehearsal; it already has its own coverage in the existing test suite (outside this mission's scope to re-verify) and has no override seam to redirect safely into a scratch dir without either mutating process-global `APPDATA`/exe-path state (risking every other test in the package that reads `deploy.DataDir()`) or editing `field_crypto.go` (banned this wave). The rehearsal instead drives `ImportKeyMaterial`/`ExportKeyMaterial`/`ExportSalt` directly — the exact functions the real recovery ritual (`ExportEncryptionBackup`/`ImportEncryptionBackup`) calls — which is faithful to the *recovery* procedure even though it doesn't touch the *original-provisioning* file-resolution code path. |
| Hardware-ID (§3) independent recovery | No independent recovery procedure exists beyond the FieldCrypto Export/Import path (which the rehearsal does cover); there is nothing separate to rehearse. |
| Mesh (§11) key recovery — device-seed re-invite, anchored-room key re-share | Requires the Node/Bare/Autobase JS runtime, which this Go-side harness does not invoke. No JS-side rehearsal was built this mission (out of scope — CW1-A's mission brief scoped the rehearsal script to FieldCrypto + DPAPI explicitly; mesh folder-level restore is CW1-B's territory, and mesh *key* recovery beyond folder copy-back has no existing procedure to rehearse per Finding 3 above). |
| Whether the owner-ritual DevTools console procedure works on a packaged/release Wails build | Not built or tested this wave (copies-only doctrine; no packaged release binary was produced or the live PH machine touched). See Finding 2. |
| Non-steward operator running the recovery procedure | Per the spec's honesty boundary (§0): this wave proves the procedure exists and works for someone with source-level and DevTools access: it does not prove a receptionist-level operator could complete it unassisted. Flagged as a wave-2-candidate concern, not solved here. |

## Coverage mapping — custody map vs. inventory (100% check)

Every key surface in `CW10_INVENTORY.md` Part A has a corresponding section in
`KEY_CUSTODY.md`, diff-checked below (not eyeballed):

| Inventory item (`CW10_INVENTORY.md`) | `KEY_CUSTODY.md` section |
|---|---|
| §1 FieldCrypto master key | §1 |
| §2 `.field_crypto_salt` | §2 |
| §3 Hardware ID + sidecars | §3 |
| §4 Windows DPAPI keystore | §4 |
| §5 Legacy SettingsService key | §5 |
| §6 OAuth token cache key + `.auth_token.json` | §6 |
| §7 Session tokens + password hashes | §7 |
| §8 Invoice integrity HMAC key | §8 |
| §9 Mistral AI API key | §9 |
| §10 License keys + developer master key | §10 |
| §11 Mesh (P2P) key material | §11 |
| §12 Ambient env/.env secrets | §12 |
| FINDINGS 1-6 (weak fallbacks, hardcoded salts, plaintext mesh keys, DPAPI scope, partial-hash logging, no auto-escrow) | `KEY_CUSTODY.md` "FINDINGS" section (all 6, cross-referenced by number) |
| "UNRECOVERABLE BY DESIGN" (social rooms/DMs, crypto-epoch predecessor keys) | `KEY_CUSTODY.md` "Unrecoverable BY DESIGN — DO NOT ESCROW" section, and mirrored in `RECOVERY_ENVELOPE_TEMPLATE.md`'s "DO-NOT-ESCROW list" |

**12 of 12 numbered key surfaces covered. 6 of 6 findings carried forward. Both
DO-NOT-ESCROW items covered in both deliverables.** No inventory item was dropped.

## Hard-rules compliance checklist

- **No existing runtime code modified.** `git status`-equivalent check: only new files
  were created (`custodian_rehearsal_test.go`,
  `scripts/custodian/rehearse_recovery/main.go`, three docs under `docs/custodian/`,
  this report). `field_crypto.go`, `settings_service.go`,
  `hardware_id_keystore_windows.go`, and every other pre-existing file are untouched.
- **No real key material committed.** Grep transcript above; all material in the
  harness is `crypto/rand`-generated at test time and never persisted outside
  `%TEMP%\custodian-rehearsal\<run-ts>\`, which `t.Cleanup` removes at the end of each
  test.
- **Copies only, scratch-guarded.** `scratchGuard()` refuses any path outside the
  scratch root, any path naming `ph_holdings.db` (case-insensitive), and any path
  containing `#`; negative-tested by `TestScratchGuardRefusesUnsafePaths` (6
  subcases, all PASS above).
- **No `git commit`/`git add` run** by this mission — changes are left in the working
  tree for the orchestrator's gate to commit.
- **No stop-and-report conditions were hit** (see Stop-and-reports section above).
