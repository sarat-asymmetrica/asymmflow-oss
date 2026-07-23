# FP-0 — Immersion & Inventory (Field Packet campaign)

**Date:** 2026-07-23 · **Orchestrator:** Opus 4.8 · **Campaign:**
`mesh/docs/FABLE_CAMPAIGN_FIELD_PACKET.md` (f62ef37)
**Charter reminder:** packaging only — `mesh/host/**`, `mesh/kit/*.mjs`,
`mesh/reducer/**`, and the launcher are FROZEN this wave.

## 1. Prior art — read, in order

| # | Document | Status |
|---|---|---|
| 1 | `bare-corridor/CORRIDOR_RUNBOOK_SEALED.md` | read — §3a MOTW/SmartScreen, §3b `#` rule, §4 two-role ceremony, §4c `/read` law, §5 photograph discipline, §6 what-worked-means, §7 verifier protocol |
| 2 | `bare-campaign/PHASE4_CLEAN_MACHINE_VERIFICATION.md` | read — Round-1 PASS block (the FP-2 exemplar), Round-2 redefined to receptionist machine, Defender/window-flash field notes |
| 3 | `bare-campaign/CAMPAIGN_REPORT.md` §4 + §5 | read — five method rules + exit-codes-lie-at-three-layers, binding law |
| 4 | `FABLE_CAMPAIGN_SEALED_CORRIDOR.md` | read — SC-4 (runbook + build) and SC-5 (final gate 16/16 leg D both ways) are the green lights being packaged |
| 5 | `mesh/kit/build-bare-kit.mjs` | read — the sanctioned builder; hard gates enumerated in §3 below |
| 6 | Aesthetic reference (local, outside repo) | exists at the spec's path; tokens extracted in §6 |

## 2. `mesh/` drift since the corridor merge — VERDICT: BENIGN

```
$ git log --oneline 0fd87b0..HEAD -- mesh/
f62ef37 Field Packet campaign spec -- ...   (docs only, +207 lines, 1 file)
```

That is the campaign spec itself and nothing else. India W1 (`3c41f17`) and
Wave 13 (`fb27926`) merged after the corridor but touched **zero** paths under
`mesh/` — verified by the log above, not assumed. No runtime file, no kit
script, no reducer change. **No spike re-runs required.** Working tree clean
at FP-0 start.

## 3. The sanctioned builder and its hard gates

`mesh/kit/build-bare-kit.mjs` — the ONLY way FP-1 may produce the kit:

- **§0** rebuilds `dist/reducer.wasm` fresh (via `scripts/build-reducer.mjs`)
  before packing — a stale reducer cannot slip in.
- **§1** wipes the output dir — no stale-output contamination.
- **§2** real `bare-pack --host win32-x64 --offload` resolution (no hand walk).
- **§2b HARD GATE** — offloaded `dist/reducer.wasm` must exist and byte-match
  the fresh build (the broken-but-green "renders everything, posts nothing" shape).
- **§2c HARD GATE** — (a) `--require-addons=` declared expectation (only an
  explicit list can catch an ABSENT addon — Rule 1); (b) byte-identity of every
  offloaded addon vs its `node_modules` source prebuild. Proven red-provable
  before being trusted (builder's own header, 2026-07-20).
- **§4** launcher written CRLF-at-write-time, `%~dp0`-relative, ASCII-only,
  exit-code propagation (one layer of lying removed, not all).
- **§4b** copies the in-kit verifier pair (`verify_clean_machine.cmd` +
  `verify-clean-machine.ps1`) into the kit.

**Sanctioned FP-1 invocation** (SC-3a/SC-4 precedent, corridor addon list from
`SC2_REPORT.md` §"require-addons"):

```
node kit/build-bare-kit.mjs --entry=kit/bare-guide-entry.mjs --require-addons=bare-tcp,udx-native,sodium-native,bare-dns
```

(The builder rebuilds the reducer itself at §0; Node v22.17.0 present.)

## 4. `mesh/kit/dist-bare/` as found vs the Sealed Ship's recorded 24

As found: **30 files, 64.7 MB**, `portable.flag` says
`entry: kit\bare-guide-entry.mjs`, `built 2026-07-20T16:12:46Z` — i.e. the
corridor-era SC-4 build, stale by three days. **It will NOT be zipped as
found** — FP-1 wipes and rebuilds through the builder (charter).

Sealed Ship recorded 24 files / 62.8 MB (`PHASE3_KIT_REPORT.md` §14a):
6 kit files (`app.bundle`, `bare.exe`, `dist/reducer.wasm`, `README_BARE_KIT.txt`,
`portable.flag`, `run_bare_mesh.cmd`) + 18 offloaded addons.

Every difference, explained:

| Δ | File | Why |
|---|---|---|
| +1 | `node_modules/bare-tcp/.../bare-tcp.bare` | SC-2 corridor network leg (TCP fallback — the one pre-approved dependency) |
| +1 | `node_modules/udx-native/.../udx-native.bare` | SC-2 hyperswarm path |
| +1 | `node_modules/bare-dns/.../bare-dns.bare` | SC-2 corridor DNS (SC0_PORT_MAP §2 measured list) |
| +1 | `node_modules/hyper-cmd-lib-keys/node_modules/sodium-native/.../sodium-native.bare` | nested prebuild sourced by bare-pack's real resolution (builder §2c(b) notes this case explicitly) |
| +1 | `verify_clean_machine.cmd` | in-kit verifier, shipped since `695243c` (builder §4b) |
| +1 | `verify-clean-machine.ps1` | same |

24 + 6 = 30. **Identity for the remaining 24 confirmed by name.** All 18
Sealed-Ship addons still present (22 addons total as found).

## 5. The in-kit verifier (what FP-1's drive-through drives)

`verify-clean-machine.ps1` phases: **A** cleanliness evidence
(node/npm/npx/bare resolvability + informational `vcruntime140.dll` line) ·
**B** `#`-path hazard warning · **C** probe control that MUST go red before
anything counts · **D** the ceremony ×16 through the REAL launcher, content-
asserted (`(posted, seq N)` + `Goodbye`), tally line
`OK=16/16 HANG=0/16 CONTENT_FAIL=0/16` · **E** optional corridor section,
env-gated (`ASYMMFLOW_VERIFY_CORRIDOR=1`), off by default — Round-2 protocol
unchanged. Evidence: `VERIFY_EVIDENCE.txt` + `verify-logs\`.

Expected-and-honest on the dev machine: phase A reports **NOT CLEAN**
(Node is installed here). That line is environmental, recorded, not suppressed;
the ceremony tally is the FP-1 gate. The Node-free claim remains provable only
on the receptionist machine — the standing un-verifiable this campaign inherits.

## 6. FP-2 design brief — tokens extracted from the aesthetic reference

Reference: `asymmetrica-capabilities.html` (local showcase, confirmed present).

**Palette:** bg `#FAF9F7` / warm `#F4F2EE` / card `#FFFFFF` / code `#1C1B22` ·
text `#1A1A1A` / `#4A4A4A` / muted `#8A8A8A` · accent (deep teal) `#1A4C5C`,
light `#E2EEF2`, mid `#236B7E` · gold `#B8860B` on `#FDF6E3` · borders
`#E4E1DB` / `#EDEAE5` · status: red `#A63D2F`/`#FCEAE7`, amber `#A16B15`/`#FEF3E0`,
green `#2D6B46`/`#E4F0EA` · shadows 3-step subtle · radius 14px / 10px.

**Type:** Spectral (300/400, italic accents) for headings — h1
`clamp(2.4–3.8rem)` weight 300, tight letter-spacing, `em` in accent teal;
DM Sans body 16px / 1.75; JetBrains Mono for anything literally on a screen.
Google Fonts link as in the reference + honest local fallbacks
(Georgia / system-ui / monospace) — legible fully offline.

**Idioms to reuse:** `.section-label` uppercase teal eyebrow · `.card` white
+ 1px border + radius 14 + shadow-sm · `.card-grid` auto-fit minmax(280px,1fr)
(the two ceremony roles) · pill badges (100px radius, uppercase 0.68rem —
critical/warning/info color pairs) · `.quote-block` gold-left-border +
`#FDF6E3` bg, Spectral italic (the honesty notes) · `.stat` tiles (serif value,
uppercase label) · `.tag` pills · dark `#1C1B22` code blocks in mono (screen
renders, the tally line).

**Explicitly NOT ported (spec):** mode-toggle machinery, progress bar,
scroll JS. Optional zero-dependency collapse only; page must read complete
with JS disabled, print dark-on-light.

## 7. Real-names gate plan (FP-2c)

The repo is public, so the scrub list itself cannot be committed here. The
gate assembles the known-real-names list (OSS-hygiene sprint precedent +
synthetic canon inversion) in the session scratchpad, greps the packet
case-insensitively, and the FP-2 report records only PASS/FAIL + the list's
cardinality — roles only in the packet ("the receptionist", "the founder",
"the person in India").

## 8. Not verified at FP-0, stated plainly

- Nothing was EXECUTED at FP-0 — no build, no verifier run. Inventory is
  file-and-git evidence only; FP-1 does the running.
- The dev machine cannot prove the Node-free claim (standing; §5).
- The stale `dist-bare/` was not re-verified — irrelevant by design, it gets
  wiped at FP-1.

**Gate FP-0: PASS** — drift enumerated with verdict (benign), builder and its
gates located, manifest delta fully explained, aesthetic tokens extracted.

*Pack the proven, prove the pack, hand it over.* 🐻📦
