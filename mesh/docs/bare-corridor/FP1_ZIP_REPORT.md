# FP-1 — The Zip (shipping artifact) · Report

**Date:** 2026-07-23 · **Orchestrator:** Opus 4.8 (drive-through run by the
orchestrator directly; no coder) · **Campaign:** Field Packet (`f62ef37`)

## 1. Fresh build — sanctioned builder, hard gates ran

```
$ node kit/build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs \
    --require-addons=bare-tcp,udx-native,sodium-native,bare-dns
...
required addons present: bare-tcp, udx-native, sodium-native, bare-dns
total: 30 file(s), 64.7 MB
```

Reducer rebuilt at builder §0; wasm-offload gate (§2b) and addon byte-identity
+ declared-requirement gates (§2c) all executed and passed. `dist-bare/` was
wiped and rebuilt — nothing stale was zipped, per charter.

## 2. FINDING FP1-1 — the frozen in-kit verifier could not pass a healthy kit

**The first drive-through went RED: `TALLY OK=0/16 CONTENT_FAIL=16/16`** on a
kit whose build gates were all green. Diagnosis (confirmed both directions
before being believed, Rule 1):

- `verify-clean-machine.ps1`'s `CEREMONY_STDIN` (5 lines: `2`, Enter,
  message, `/exit`, `5`) predates **SC-3's start-or-`connect` prompt** in the
  guide. The message text was being consumed by that prompt; nothing was ever
  posted; every run honestly reported `CONTENT_FAIL`.
- Proof the kit was healthy: one manual ceremony with the corrected 6-line
  sequence → `(posted, seq 2)` + `Goodbye`.
- Proof of the causal chain: the verifier shipped at `695243c` (Sealed Ship,
  pre-corridor) and was last touched at `1aa074b` (SC-4, which added the
  optional phase E but not the extra Enter). Round 1's sandbox 16/16 PASS
  (2026-07-20) ran against the *pre-corridor* guide; SC-5's final gate drove
  its own harness (`sealed-corridor-gate.mjs`), not the in-kit verifier. The
  defect slipped between the two instruments.

**Field consequence had this shipped:** the receptionist's Round-2 run would
have reported a healthy kit as `CONTENT_FAIL 16/16` — a false RED on the one
machine where nobody can diagnose it. The drive-through exists for exactly
this; Rule 1 lives on.

**Resolution:** verifier is FROZEN under this campaign's charter →
stop-and-report → **owner authorized the one-line fix** (one added CRLF line
in `CEREMONY_STDIN`, commit `a18196e`). The human-facing Round-2 protocol
(double-click, same tally line, same evidence folder) is unchanged. Kit
rebuilt through the sanctioned builder, zip re-cut, entire gate re-run.

## 3. The shipping zip (final cut)

| property | value |
|---|---|
| name | `AsymmFlow-SealedKit-Field-20260723.zip` |
| location | `C:\Projects\asymmflow\` — outside the repo tree; `dist-bare/` itself is confirmed gitignored, working tree clean |
| size | 24,179,118 bytes (30 files, 64.7 MB uncompressed) |
| zip sha256 | `130BF121870BF47CA9F3A0112760A6FB2A1494FEB6D5B826FF683F6EB3DB45FE` |
| `bare.exe` sha256 | `61D7F0D40CBC061F657B126D2DEB3A74E38ED46CD73F86DA0163D7E613EC3962` (45,142,016 B) |
| `app.bundle` sha256 | `5509B282824A879246FAED1700E375A4C0D43AEBE73787B31AC78E0A94A7E2A9` (2,324,665 B) |
| `dist/reducer.wasm` sha256 | `12F86EC902FBB8B7ED17D55708CDA820877D2A7AA0880927C897A35A37CE009B` (3,963,665 B) |

Layout: kit files at the ZIP ROOT — `Expand-Archive`/Extract-All yields
`run_bare_mesh.cmd` one level down, no nested `dist-bare/dist-bare`.

**Observation (recorded, not alarming):** across the two builds of this
mission, `app.bundle` was byte-identical but `reducer.wasm` was
size-identical with a different sha256 — the reducer build is not
byte-deterministic across runs. The builder's §2b gate is size-based, and
reducer correctness is content-gated elsewhere (13/13 parity both runtimes).
Stated so nobody later mistakes a hash drift for tampering.

## 4. Gate FP-1 — the drive-through (all on the ACTUAL zip, Windows-native)

Extraction: `Expand-Archive` → `%TEMP%\fp1-drive-through\` (from scratch,
`#`-free, outside the repo). 30 files.

| leg | result |
|---|---|
| 1. layout | `run_bare_mesh.cmd` at extraction root ✅ |
| 2. kit's own verifier, default protocol | **`TALLY OK=16/16 HANG=0/16 CONTENT_FAIL=0/16`**, probe control red FIRST (`verdict=CONTENT_FAIL`), `VERIFY_EVIDENCE.txt` + 17 logs present, `=== VERDICT: KIT PASS (16/16) but machine NOT clean ===` ✅ |
| 3. CRLF byte-assert | `run_bare_mesh.cmd` and `verify_clean_machine.cmd`: 0 bare LF, all CRLF ✅ |
| 4. negative control | fresh extraction, 4096 bytes randomized at `app.bundle` midpoint (size unchanged — the manifest check alone cannot see it) → `TALLY OK=0/16 CONTENT_FAIL=16/16`, `VERDICT: FAIL` ✅ RED |
| 5. `#`-path refusal | extraction into `%TEMP%\fp1-#-hazard\` → `B: FATAL ... VERDICT: NOT RUN (hazardous path)`, exit 2, before any ceremony ✅ loud, not confusing |

Supplementary (read-only, same-defect-class sweep after FP1-1): one E2-style
`menu [1]` run through the real launcher — verdict word captured
(`CORRIDOR AMBER`, consistent with this network's SC-0 characterization;
evidence-only by design, never a gate). Phase E's stdin is NOT stale.

The expected-and-honest dev-machine line — `A: MACHINE IS NOT CLEAN` (Node on
PATH here) — was recorded, not suppressed. The ceremony tally is the gate;
the Node-free claim remains field-provable only (Round 2).

## 5. Not verified, stated plainly

- **The Node-free claim** — cannot be proven on this machine; that is what
  Round 2 on the receptionist machine is for.
- **Two-machine corridor legs** — single-machine by definition here; SC-5's
  leg D (16/16 both ways) remains the standing evidence until the LAN
  rehearsal and ceremony.
- **MOTW/SmartScreen behaviour of THIS zip** — it has never traversed a
  browser download; the packet documents the expected behaviour and the
  Round-2 slot records what actually happens (feeds DP2 §9 Authenticode).
- The negative-control and hazard extractions were deleted; the clean
  drive-through extraction retained for FP-4's independent re-verification.

**Gate FP-1: PASS** — every leg on the actual artifact, red-provable, N=16.

*Pack the proven, prove the pack, hand it over.* 🐻📦
