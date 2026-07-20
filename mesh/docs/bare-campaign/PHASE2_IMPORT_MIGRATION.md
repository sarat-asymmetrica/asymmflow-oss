# Phase 2 — Import Migration (Coder P0-B)

**UPDATE (resequenced):** the `#apply` condition map (mesh-node.mjs's
`./apply.mjs` load-time failure under Bare, blocking P1A-wasi-shim) was
inserted as the top priority ahead of the rest of this document — see §8.
It landed first; the rest of this file describes the original-scope work
that followed.

**Scope:** migrate `node:`-prefixed specifiers to the `bare`-condition import
map (`PHASE0_GATE_B3_CONDITION_MAP.md`) in the 9 files this coder owns,
per file fencing. Fix `apply-bare.mjs`'s known pack-time blocker (the
`isBare` ternary). All gates below are real transcripts run against the
actual `mesh/` tree, not a scratchpad copy — the source edits are real.

**All required gates stayed green, byte-identical to their pinned goldens.
No API non-equivalence was found on any call site actually exercised by
these files** — the one function these files rely on from `node:crypto`
(`createHash('sha256').update(...).digest('hex')`) is a verified drop-in
under `bare-crypto`. One NEW pack-time finding surfaced during verification,
outside this task's scope but worth flagging now — see §6.

---

## 1. Per-file migration table

| File | `node:` imports found | Action | Verified-equivalent |
|---|---|---|---|
| `mesh/package.json` | — | Added `imports` map: `#fs` → `bare-fs`/`fs`, `#crypto` → `bare-crypto`/`crypto`. **Only these two** — no other alias is used by any file this coder owns, so `#path`/`#url`/`#os`/`#readline`/`#net` were deliberately NOT added (would be speculative). No new devDependencies needed: `bare-fs` and `bare-crypto` were already present in `devDependencies` from the C4/Phase-0 spikes. | N/A |
| `mesh/host/apply-bare.mjs` | dynamic ternary: `isBare ? await import('bare-fs') : await import('node:fs')` | Replaced with static `import * as fsMod from '#fs'`; removed the now-unused `isBare` local; updated the header comment to explain the packaging fix and cite the source (`PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §9, `PHASE0_GATE_B3_CONDITION_MAP.md`). | Yes — see §2 and §6 |
| `mesh/host/mesh-node.mjs` | `import { createHash } from 'node:crypto'` (1 use, `viewDigest()`) | → `'#crypto'` | Yes — see §2 |
| `mesh/host/capability.mjs` | `import { createHash } from 'node:crypto'` (2 uses: `signOp`, `inviteProof`) | → `'#crypto'` | Yes — see §2 |
| `mesh/host/attachments.mjs` | `import { createHash } from 'node:crypto'` (3 uses: `putAttachment`, `getAttachment`, `verifyAttachmentBytes`) | → `'#crypto'` | Yes — see §2 |
| `mesh/host/reissue-room.mjs` | `import { createHash } from 'node:crypto'` (1 use, deterministic invite-seed derivation) | → `'#crypto'` | Yes — see §2 |
| `mesh/host/social-room.mjs` | none | No change — file has zero `node:`/`bare-*` imports (confirmed by direct read and grep). Matches `PHASE0_NOTES_ORCH_PORTMAP.md`'s own note that this file is builtin-free. | N/A |
| `mesh/host/invite-code.mjs` | none | No change — same as above; only dependency is `hypercore-id-encoding`, already confirmed loading clean under Bare in the original C4 spike. | N/A |
| `mesh/host/export-transcript.mjs` | none | No change — pure orchestration over `node.ops()`/`.viewDigest()`/`.state()`, no direct builtin usage. | N/A |

**Total: 4 files changed (`mesh-node.mjs`, `capability.mjs`,
`attachments.mjs`, `reissue-room.mjs`) with a single-line import swap each;
1 file changed (`apply-bare.mjs`) with the ternary replaced; 1 file
(`package.json`) gained the `imports` map; 3 files needed zero changes.**
`bridge-server.mjs`, `bridge-spike.mjs`, `wasi-preview1-lite.mjs`,
`apply.mjs`, `mesh/reducer/**`, `mesh/cmd/**`, `mesh/goldens/**`,
`mesh/kit/**` were not touched, per file fencing.

---

## 2. API-equivalence findings

**The only `node:crypto` function actually called anywhere in these 4
files is `createHash(alg).update(data).digest(encoding)`** — confirmed by
direct read of every call site (listed above) before writing any test.
Nothing else from `node:crypto` (no `randomUUID`, `randomBytes`,
`timingSafeEqual`, HMAC, etc.) is used by any file this coder owns — those
are real risks per the team lead's brief, but they don't occur in this
coder's file set (they may occur in files P1A-wasi-shim or others own).

Built a scratchpad equivalence test (`crypto-equiv-test.mjs`, in
`p0b-pack/`, not the repo) exercising the exact call shape used in the real
files — `createHash('sha256').update(buffer-or-string).digest('hex')` —
against a **known NIST/FIPS 180-4 SHA-256 test vector** (not an invented
constant) as a positive control, plus a negative control (a different input
must produce a different digest, proving the test isn't tautological):

```
$ node crypto-equiv-test.mjs
runtime: node
createHash sha256("abc") = ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
matches NIST known vector: true
Buffer-input digest: b713f6d2a989e907c58516a7c8bb487792c8785ddbf367b80dc774bf33b85a85
different input digest differs from abc digest: true

$ ./bare.exe crypto-equiv-test.mjs
runtime: bare
createHash sha256("abc") = ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad
matches NIST known vector: true
Buffer-input digest: b713f6d2a989e907c58516a7c8bb487792c8785ddbf367b80dc774bf33b85a85
different input digest differs from abc digest: true
```

**Byte-identical output, both runtimes, matching the published NIST
vector.** `bare-crypto`'s `createHash('sha256')` is a verified drop-in for
this exact call shape. This is also independently confirmed by the gate
transcripts in §3: `capability.mjs`'s `signOp`/`inviteProof` (both feed
`createHash` output into `hcrypto.sign`) and `mesh-node.mjs`'s
`viewDigest()`/`attachments.mjs`'s ref hashing all produce pinned-golden-
matching digests under **both** `node host/*.mjs` and `npx bare
host/*.mjs` runs of the real reducer-fold spikes — the real signing and
hashing paths, not just an isolated probe.

**`hypercore-crypto` (used by `capability.mjs` for `keyPair`/`sign`/
`verify`) was not part of the `node:` migration** (it's a Holepunch package,
not a Node builtin, and doesn't import `node:crypto` — confirmed via `npm
view hypercore-crypto dependencies`: only `b4a`, `compact-encoding`,
`sodium-universal`). Checked anyway since capability.mjs's cryptographic
correctness rests on it: same 32-byte seed produces byte-identical
`publicKey` under Node and Bare, and `sign`/`verify` round-trip correctly
under both:

```
$ node hcrypto-test.mjs          $ ./bare.exe hcrypto-test.mjs
keyPair OK, pub: ea4a6c63e29c...  keyPair OK, pub: ea4a6c63e29c...  (identical)
sign OK, sig len: 64              sign OK, sig len: 64
verify OK: true                   verify OK: true
```

**No function used by any file in this coder's set was found to be
missing, differently-signed, or behaviorally divergent under `bare-*`.**
Nothing was silently substituted; nothing required stopping on a call site.

---

## 3. Gate transcripts

All run against the real `mesh/` tree after the edits (not a copy),
`cwd = mesh/`.

```
$ node host/bare-parity-spike.mjs
13 scenario(s) run, 0 failure(s) total.
BARE PARITY SPIKE GREEN -- the unmodified reducer folds byte-identically under Bare

$ npx bare host/bare-parity-spike.mjs
13 scenario(s) run, 0 failure(s) total.
BARE PARITY SPIKE GREEN -- the unmodified reducer folds byte-identically under Bare

$ node host/smoke.mjs
✓ boundary / state / invariant / convergence / golden — all pass
digest: 6c8c35eff1e2c04d6d46704ad7c542c2808717fae58fb1d91ceccfcbd09eb410
SMOKE GREEN ✅

$ node host/reactor-parity-spike.mjs
13 scenario(s) run, 0 check failure(s) total.
REACTOR PARITY GREEN ✅ — every golden folds byte-identically through the reactor
```

Node-line spikes whose imports were touched (all against pinned goldens):

```
$ npm run invitespike      → INVITE SPIKE GREEN ✅  (golden digests matched)
$ npm run roomspike        → ROOM SPIKE GREEN ✅    (golden digests matched)
$ npm run socialspike      → SOCIAL SPIKE GREEN ✅  (golden digest matched)
$ npm run attachspike      → ATTACH SPIKE GREEN ✅  (golden digests matched)
$ npm run transcriptspike  → TRANSCRIPT SPIKE GREEN ✅ (golden digests + bundle sha256 matched)
$ npm run reissuespike     → REISSUE SPIKE GREEN ✅ (golden digests matched)
$ npm run missionc         → MISSION C GREEN ✅     (golden digests matched)
```

**Every one of these compares against a pinned golden digest, not merely
"exit 0" — a silent behavior change would show up as a digest mismatch,
and none did.** All 11 gates (4 primary + 7 Node-line spikes) green.

---

## 4. Packages added

**None.** `bare-fs` (`^4.7.4`) and `bare-crypto` (`^1.15.3`) were already
present in `mesh/package.json`'s `devDependencies` before this task (added
in the C4/Phase-0 spike work). No `npm install` was run against the real
`mesh/` tree for this task.

---

## 5. `mesh/package.json` diff (the only non-`host/` file touched)

```diff
   "description": "Sovereign Mesh track for AsymmFlow — ...",
+  "imports": {
+    "#fs": { "bare": "bare-fs", "default": "fs" },
+    "#crypto": { "bare": "bare-crypto", "default": "crypto" }
+  },
   "scripts": {
```

Deliberately minimal — only the two aliases this coder's files actually
import. `#path`/`#url`/`#os`/`#readline`/`#net` are real needs elsewhere in
the campaign (per `PHASE0_GATE_B3_CONDITION_MAP.md`'s full recipe) but
adding them here with no consumer would be exactly the speculative-entry
anti-pattern the brief warned against; whichever coder's files need them
should add them when they do the corresponding migration, to keep the map's
history legible (one alias, one PR, one reason).

---

## 6. Additional finding — NOT part of this task's scope, flagged plainly

After the `isBare` ternary fix, I verified end-to-end that `apply-bare.mjs`
now actually **packs** (it previously failed at `bare-pack` time with
`MODULE_NOT_FOUND: node:fs`, per `PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §9 —
this was the specific blocker Task 3 asked me to fix, and it is fixed):

```
$ bare-pack --host win32-x64 --offload -o packed/entry.bundle entry.mjs
pack exit=0
```

But running the resulting bundle in true hostile geography (a from-scratch
directory: `bare.exe` + the bundle + the offloaded `bare-fs`/`bare-path`/
`bare-url` addon tree, nothing else) surfaces a **second, separate,
pre-existing** blocker — not something this migration introduced:

```
Uncaught FileError: ENOENT: no such file or directory, open
  "...\entry.bundle\dist\reducer.wasm"
    at loadModule (.../apply-bare.mjs:45:42)
```

This is the **already-documented `import.meta.url` virtual-path landmine**
from `PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §4b: `apply-bare.mjs` locates
`reducer.wasm` via `new URL('../dist/reducer.wasm', import.meta.url)`
(line 32 of the file, untouched by this migration — it is a self-location
pattern, not a `node:`-specifier issue, so it was out of this task's
scope). Once the file is actually bundled, `import.meta.url` resolves to a
path *inside* the `.bundle` file's virtual namespace, not a real directory,
so the relative `../dist/reducer.wasm` lookup misses. The verified fix
(already proven working end-to-end in hostile geography, §4c of the same
document) is to switch that one line to
`import.meta.asset('../dist/reducer.wasm')` combined with `--offload`/
`--offload-assets` at pack time.

**I did not apply this fix.** It touches the same file I own, and the
one-line change is already fully specified and pre-verified from Phase 0,
so I could — but the team lead's Task 3 scoped this coder's fix
specifically to the `node:fs` ternary, and the wasm self-location question
sits squarely in Phase 3 (sealed-kit packaging) per the campaign phase
boundaries, not Phase 2 (import migration). Flagging it here rather than
silently expanding scope; recommend either authorizing this coder to apply
the one-line `import.meta.asset()` fix now (it's isolated, well-evidenced,
and would make `apply-bare.mjs` genuinely pack-and-run clean end to end)
or carrying it as a named Phase 3 item so it isn't rediscovered cold.

---

## 7. `#apply` condition map — unblocking P1A-wasi-shim (top priority, done first)

**Problem:** `mesh-node.mjs:21` imported `applyViaWasm` from `./apply.mjs`
unconditionally. `apply.mjs` is Node-only (`node:wasi`), so under `npx bare`,
merely *loading* `mesh-node.mjs` failed: `MODULE_NOT_FOUND: node:wasi`. This
blocked the entire Bare bridge P1A-wasi-shim was building.

**Ruling followed (not improvised):** do NOT point `mesh-node.mjs` at
`apply-bare.mjs` unconditionally, even though it's proven to work under both
runtimes — `apply.mjs` is the frozen rollback path (owner ruling R1
condition 2) and must stay independently fallible from the new host. Used a
`#apply` condition map instead, same mechanism as `#fs`/`#crypto` but with
**relative file targets** instead of package names — untested territory,
verified before relying on it.

### 8a. Relative-target resolution, verified from a nested importer (not just the package root)

Built a scratchpad harness matching the real layout exactly — a
`package.json` at the root, real files at `host/apply.mjs` and
`host/apply-bare.mjs`, and (critically) the **consumer file also nested at
`host/consumer.mjs`**, since resolving from a nested importer is a different
question than resolving from the package root and the two can differ:

```json
{ "imports": { "#apply": { "bare": "./host/apply-bare.mjs", "default": "./host/apply.mjs" } } }
```

```
$ node host/consumer.mjs
resolved to: FROM_APPLY_NODE

$ ./bare.exe host/consumer.mjs
resolved to: FROM_APPLY_BARE
```

**Both branches verified, run both ways — not one branch exercised three
times and the other zero, the exact mistake flagged twice already today.**
Relative targets resolve correctly, and they resolve relative to the
package root (`package.json`'s own location), not relative to the importing
file — confirmed by the nested importer still finding `./host/apply.mjs`
correctly rather than looking for `host/host/apply.mjs`.

### 8b. Contract check — `apply.mjs` vs `apply-bare.mjs`

Read both files in full before wiring anything. Both export:

```js
export function applyViaWasm(ops, config = undefined, mode = '')
```

— same parameter names, same defaults, both plain synchronous functions
(no `async`, no Promise), both return the parsed converged `State` object
(`apply.mjs` via `JSON.parse(readFileSync(outPath, 'utf8'))`;
`apply-bare.mjs` via `JSON.parse(applyViaWasmRaw(...).toString('utf8'))`).
`apply-bare.mjs` additionally exports `applyViaWasmRaw` (not imported by
`mesh-node.mjs`, irrelevant to this contract). **Signatures and return
shape match exactly — no adaptation of `mesh-node.mjs`'s call site was
needed or made.**

### 8c. Applied to the real files

`mesh/package.json`:
```diff
   "imports": {
+    "#apply": { "bare": "./host/apply-bare.mjs", "default": "./host/apply.mjs" },
     "#fs": { "bare": "bare-fs", "default": "fs" },
     "#crypto": { "bare": "bare-crypto", "default": "crypto" }
   },
```

`mesh/host/mesh-node.mjs`: `import { applyViaWasm } from './apply.mjs'` →
`import { applyViaWasm } from '#apply'`, with a comment explaining the
independence rationale so a future editor doesn't "simplify" it back to a
direct `apply-bare.mjs` import.

### 8d. Gates — this change specifically

```
$ node -e "import('./host/mesh-node.mjs').then(()=>console.log('node OK'))"
node OK   (WASI experimental-feature warning only, expected — apply.mjs's own)

$ npx bare <loader importing ./host/mesh-node.mjs>
bare OK
```

Every Node-line spike that uses `mesh-node.mjs`, re-run after the change,
compared against pinned goldens:

```
node host/smoke.mjs        → SMOKE GREEN ✅        digest unchanged: 6c8c35ef...
npm run invitespike        → INVITE SPIKE GREEN ✅  golden digests matched
npm run roomspike          → ROOM SPIKE GREEN ✅    golden digests matched
npm run socialspike        → SOCIAL SPIKE GREEN ✅  golden digest matched
npm run attachspike        → ATTACH SPIKE GREEN ✅  golden digests matched (see note below)
npm run transcriptspike    → TRANSCRIPT SPIKE GREEN ✅ golden digests + bundle sha256 matched
npm run reissuespike       → REISSUE SPIKE GREEN ✅ golden digests matched
npm run missionc           → MISSION C GREEN ✅     golden digests matched
```

**Honesty note, not swept under the rug:** one `attachspike` run in the
middle of this batch printed a `view digest` that differed from every other
run of the same script (before and after). The **golden-checked** value in
that same spike (`state digest`, asserted via "golden: room state digest
matches pinned golden" / "golden: state projection matches pinned golden
(deep)") was byte-identical in that run and every other — only the
un-asserted, purely-`console.log`'d `view digest` (attach-spike.mjs line
157, no golden comparison exists for it) differed once, then reproduced the
original value cleanly on immediate re-run. `attachments.mjs` doesn't even
import `#apply`, and `git status` at the end of this session shows another
coder's untracked files (`mesh/host/bare-bridge.mjs`,
`PHASE0_NOTES_D2_FLUSH_RACE.md`) appearing in the same shared tree mid-
session — concurrent test runs sharing `os.tmpdir()`-based storage paths is
the likely explanation, not a regression from this change. Flagged rather
than silently dropped; not re-investigated further since it's outside this
coder's file fence and the actually-gated value never moved.

**#apply landed and unblocks P1A-wasi-shim.** Team lead notified immediately per instruction.

## 8. What is NOT verified

1. **§6's wasm self-location fix** — known, specified, previously verified
   in isolation (Phase 0 spike), but not applied or re-verified against the
   real `apply-bare.mjs` in this session (out of scope, see §6).
2. **API equivalence for `node:crypto` functions NOT called by this
   coder's files** (`randomUUID`, `randomBytes`, `timingSafeEqual`, HMAC,
   etc.) — the team lead named these as the ones "I would bet against."
   They were not encountered in any file this coder owns; whichever coder
   migrates a file that calls them still needs to check those specifically.
3. **`#path`/`#url`/`#os`/`#readline`/`#net` aliases** — not added, not
   exercised, because nothing in this coder's file set imports them. Their
   own equivalence questions (flagged in `PHASE0_GATE_B3_CONDITION_MAP.md`,
   e.g. `#net → bare-tcp` being "an assumption of shape, not a verified
   drop-in") remain exactly as open as before this task.
4. **`bridge-server.mjs`/`bridge-spike.mjs`** (owned by P1A-wasi-shim,
   file-fenced off) — not read in detail, not touched, no claim made about
   their migration status.
5. **macOS/Linux** — all gates run on Windows only, consistent with every
   prior spike in this campaign.
6. Did not run `go test ./mesh/...` (`npm run test:go`) — not in the
   required-gates list for this task and no Go source was touched.
