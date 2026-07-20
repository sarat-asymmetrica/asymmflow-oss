# Phase 0 — Gate result: the dual-runtime recipe (`bare` condition import map)

**Date:** 2026-07-20 · **Author:** orchestrator (Opus 4.8), run personally
**Question:** can ONE source file stay dual-runtime (Node *and* Bare) while remaining
packable by `bare-pack`, given that `bare-pack`'s static traverser refuses to walk past
any `node:` specifier (`PHASE0_NOTES_B2_PACKAGING_SPIKE.md` §8)?

**Answer: YES — via a `"bare"` condition import map in `package.json`. Verified end to
end, with negative controls.**

## Why this was asked

`mesh/host/apply-bare.mjs` (P1-A, destined for the sealed artifact) selects its filesystem
module at runtime:

```js
const fsMod = isBare ? await import('bare-fs') : await import('node:fs')
```

Correct at runtime, but `bare-pack` resolves statically and cannot distinguish `node:fs`
from a nonexistent package. A file that runs green today can therefore be unpackable —
a Phase 3 blocker hiding in code that nothing had yet tried to bundle.

## Method

Scratchpad harness (outside the repo, D5). `package.json`:

```json
{
  "type": "module",
  "imports": {
    "#fs": { "bare": "bare-fs", "default": "node:fs" }
  }
}
```

Entry file imports through the alias only: `import fs from '#fs'`.

## Results

| # | test | result |
|---|---|---|
| 1 | run directly under `bare.exe` | **OK** — `typeof readFileSync === 'function'`, read its own source |
| 2 | `bare-pack --host win32-x64 --offload-addons -o out/app.bundle entry.mjs` | **pack exit 0** |
| 3 | copy `bare.exe` + bundle + offload tree to a directory with no package.json, no source, no npm tree, and run | **`bare-fs` resolved and worked** |
| NC-A | same map, `bare` branch → package that does not exist | **pack exit 1** ✓ control fails |
| NC-B | plain `import fs from 'node:fs'`, no map | **pack exit 1** ✓ control fails |

The negative controls matter: they prove the harness can report failure, so test 2's exit 0
means something. (This session has now been bitten twice by probes that could only ever
report failure — see `PHASE0_GATE_B2_RESOLUTION.md`.)

**`bare-pack` honors the `bare` condition and resolves ONLY that branch.** The `default`
`node:fs` entry sits in the map unresolved and does not break the build — which is exactly
the property we need.

## THE RECIPE (copy-pasteable, binding for Phase 2)

In `mesh/package.json`:

```json
"imports": {
  "#fs":      { "bare": "bare-fs",      "default": "node:fs" },
  "#path":    { "bare": "bare-path",    "default": "node:path" },
  "#url":     { "bare": "bare-url",     "default": "node:url" },
  "#crypto":  { "bare": "bare-crypto",  "default": "node:crypto" },
  "#os":      { "bare": "bare-os",      "default": "node:os" },
  "#readline":{ "bare": "bare-readline","default": "node:readline" },
  "#net":     { "bare": "bare-tcp",     "default": "node:net" }
}
```

Consumers write `import fs from '#fs'` — never `node:fs`, never a runtime ternary. One
source file, both runtimes, packable.

Two caveats before this is applied wholesale:

1. **API equivalence is NOT established.** `bare-fs` exposes 95 exports vs Node's 107;
   `bare-crypto` 21 vs 68. The map guarantees *resolution*, not that the specific functions
   we call exist with the same signatures. Every migrated call site needs its own check.
2. `#net → bare-tcp` is an assumption of shape, not a verified drop-in for `node:net`.
   Verify before relying on it.

## Bonus finding — independent confirmation of the `import.meta.url` landmine

Test 3 also reproduced P0-B's finding #4 by accident. The entry read its own source via
`import.meta.url` and, from inside the bundle, that resolved to:

```
...\gate-condmap-hostile\app.bundle\entry.mjs   → ENOENT
```

i.e. a **virtual path inside the `.bundle` file**, not a real directory. This is the exact
failure mode awaiting all 23 `fileURLToPath(import.meta.url)` sites in `mesh/host` and
`mesh/kit` once they are bundled, and it confirms `import.meta.asset()` + `--offload` is
mandatory for asset lookup rather than optional. Note the process still exited 0 while
throwing — consistent with the exit-code unreliability P0-D is investigating separately.

## Not verified

- Whether `bare-pack` follows dynamic `await import()` inside a ternary at all (the
  original question about `apply-bare.mjs`'s current form). The condition map makes it
  moot, so it was not pursued — but if a dynamic import is silently IGNORED rather than
  followed, a file could pack "successfully" and fail at runtime, which is worse. Flagged.
- Behavior of the map under `--offload` (assets) as opposed to `--offload-addons`.
- Any host other than `win32-x64`.
