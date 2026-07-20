# Phase 0 — Gate verification: what actually resolves under Bare

**Date:** 2026-07-20 · **Verifier:** orchestrator (Opus 4.8), independently of coder P0-B
**Subject:** P0-B's finding #3 in `PHASE0_NOTES_B2_PACKAGING_SPIKE.md` — *"neither `node:fs`
nor plain `fs` resolves under Bare at all"*.

**Why re-verified personally:** that finding would make every `node:`-importing file in
`mesh/host` and `mesh/kit` unpackable, which P0-B correctly identified as the single
biggest Phase-2 risk. A claim that large gets measured, not accepted (D1).

## Verdict: the FINDING is wrong; the RECOMMENDATION it produced is right

P0-B's stated evidence was `bare.exe -e "require('node:fs')"` returning "zero candidates".
That invocation cannot test what it appears to test: `-e` evaluates as ESM, where `require`
is not defined at all — an identical trap the orchestrator fell into while first checking
this, producing `require is not defined` for *every* specifier including `bare-fs`. Any
conclusion drawn from it is void.

Re-measured with real ESM files and `await import()`:

### Inside the mesh tree (bare-* packages installed) — `npx bare .gate-tmp/resolve-check.mjs`

| specifier | Bare | Node (control) |
|---|---|---|
| `node:fs` / `fs` | **OK** — 95 named exports | OK — 107 |
| `node:path` / `path` | **OK** — 14 | OK — 18 |
| `node:url` / `url` | **OK** — 11 | OK — 13 |
| `node:net` | **OK** — 10 | OK — 19 |
| `node:crypto` / `crypto` | **FAIL** — MODULE_NOT_FOUND | OK — 68 |
| `node:os` | **FAIL** — MODULE_NOT_FOUND | OK — 24 |
| `node:readline` | **FAIL** — MODULE_NOT_FOUND | OK — 9 |
| `bare-fs` / `bare-path` / `bare-url` | OK | FAIL (`Bare is not defined`) |
| `bare-crypto` / `bare-process` / `bare-os` | OK | FAIL (`require.addon is not a function`) |

So Bare **does** alias `node:fs`/`path`/`url`/`net` onto the corresponding `bare-*`
packages — but does **not** alias `crypto`, `os`, or `readline`, even though `bare-crypto`
and `bare-os` are installed and resolve perfectly under their own names.

### Outside any node_modules (hostile geography) — same script, scratchpad dir

**Everything fails**, `node:fs` and `bare-fs` alike — all 17 specifiers, MODULE_NOT_FOUND.

## The load-bearing insight

**Bare has no built-in modules whatsoever.** There is no internal `fs`. `node:fs` is not a
builtin being polyfilled — it is an *alias* resolved from `node_modules` on disk like any
other package. That is why the same import works in the repo and fails three directories
away, and it is the cleanest possible restatement of why this campaign exists: under Node,
`node:fs` is guaranteed; under Bare, it is a dependency that must be *sealed into the
artifact*. It also explains, precisely, why the original 2026-07-19 spike saw the whole
Holepunch stack "import clean with no shims" — those packages were sitting in
`mesh/node_modules`, and nothing was being polyfilled at all.

## Consequence: rewrite the imports anyway — P0-B's recipe stands

The corrective action P0-B derived is adopted, for stronger reasons than the one it gave:

1. `node:crypto` (11 uses), `node:os` (18), `node:readline` (7) have **no alias** and
   **must** be rewritten to `bare-crypto` / `bare-os` / `bare-readline`. This is not
   optional and is not covered by any aliasing.
2. `node:fs` / `path` / `url` / `net` do alias today, but the alias table is Bare's
   internal policy, not a contract we control, and it evidently does not cover the whole
   Node surface. Depending on it buys nothing. **Rewrite them to explicit `bare-*` too**,
   so resolution is deterministic and identical at pack time and run time.
3. Whether `bare-pack` resolves `node:`-prefixed specifiers at PACK time is a separate
   question this note does not settle — and rewriting makes it moot, which is the point.

**Binding rule for Phase 2 coders:** no `node:` specifier appears in any file destined for
the sealed artifact. Import `bare-*` explicitly, every time.

## What this does NOT establish

- Bare's alias table is not documented here beyond the 17 specifiers measured; other
  `node:` names may or may not alias. Untested names must be assumed absent.
- API *equivalence* is untested — `bare-fs` reports 95 exports vs Node's 107, `bare-crypto`
  21 vs 68. Resolution succeeding is not the same as the functions we call existing with
  the same signatures. Each rewritten call site needs its own verification.
- P0-B's other findings (`--offload-addons`, the `.bundle` format, `import.meta.asset()`,
  the four extra native addons) are NOT affected by this correction and stand as reported;
  they were verified end-to-end with transcripts in hostile geography.
