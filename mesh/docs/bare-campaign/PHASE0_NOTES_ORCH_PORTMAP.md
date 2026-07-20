# Phase 0 — Orchestrator note: the Node-builtin port surface

**Author:** orchestrator (Opus 4.8) · **Date:** 2026-07-20 · Produced by direct
inspection of the tree on `feat/fable-bare-runtime`, not from memory.

This is the mechanical inventory of what the Bare port must satisfy. It is
evidence for phase briefs, not a design. Companion agent notes: `PHASE0_NOTES_A_BARE.md`
(bare-* API surface), `..._B_PACKAGING.md`, `..._C_WASMEXPORT.md`, `..._D_REVERIFY.md`.

## Method

```
grep -ho "node:[a-z_]*" mesh/host/*.mjs mesh/kit/*.mjs | sort | uniq -c | sort -rn
```

## Aggregate usage (mesh/host + mesh/kit)

| count | builtin |
|---|---|
| 28 | `node:fs` |
| 27 | `node:path` |
| 23 | `node:url` |
| 18 | `node:os` |
| 11 | `node:crypto` |
| 8 | `node:child_process` |
| 7 | `node:readline` |
| 6 | `node:net` |
| 1 | `node:wasi` |
| 1 | `node:events` |

## Per-file, for the actual Phase-2/Phase-3 port targets

| file | builtins used | LOC |
|---|---|---|
| `host/bridge-server.mjs` | crypto, fs, net, path | 480 |
| `host/mesh-node.mjs` | crypto | 190 |
| `host/apply.mjs` | fs, os, path, url, **wasi** | — |
| `host/capability.mjs` | crypto | — |
| `host/social-room.mjs` | *(none)* | — |
| `host/invite-code.mjs` | *(none)* | — |
| `host/attachments.mjs` | crypto | — |
| `host/export-transcript.mjs` | (dynamic) | — |
| `kit/guide.mjs` | child_process, fs, path, readline, url | 395 |
| `kit/kit-host.mjs` | crypto, fs, path, readline, url | — |
| `kit/kit-net.mjs` | net | — |
| `kit/kit-registry.mjs` | fs, path | — |
| `kit/kit-repl.mjs` | crypto, fs, path, readline | — |
| `kit/probe.mjs` | crypto, net, url | — |
| `kit/anchor.mjs` | fs, path, url | — |

## Reading of the map (to be confirmed against P0-A's findings)

- **`node:wasi` is the single hard blocker**, exactly as the 2026-07-19 spike said, and
  it is confined to ONE file (`host/apply.mjs`). Whichever Phase-1 path wins, the blast
  radius on the host side is one module. Good news for D4.
- **`social-room.mjs` and `invite-code.mjs` use no builtins at all** — pure mesh law,
  port for free.
- **The mesh core is nearly builtin-free**: outside `apply.mjs`, the host layer needs
  only `crypto` (hashing/random/signing) plus `fs`/`path` in the bridge. The 11
  Holepunch packages already load clean under Bare (spike §3, being re-verified by P0-D).
- **The kit layer, not the mesh layer, is the port's real surface**: `readline`
  (7 uses — the Guided Path's whole question-and-answer UX) and `child_process`
  (8 uses — launching things, the anchor/probe ceremony). These have no equivalent in
  the mesh core and are exactly where the guided-path reimplementation lands. P0-A must
  answer whether `bare-readline`/`bare-subprocess` exist and what their real API shapes
  are on Windows; if `readline` has no counterpart, the guided path's line-reader is
  written directly against `bare-process`'s stdin stream (permitted — the guided path's
  UI layer is presentation, not mesh law, per campaign §3 Phase 3).
- **`node:net` (6 uses)** matters less than it looks: the bridge's TCP mode is dev-only
  and the DP4 seam is stdio. `kit-net.mjs`/`probe.mjs` are the ones that genuinely dial.
- **`node:url`'s 23 uses are almost entirely `fileURLToPath(import.meta.url)`** for
  self-locating paths — a pattern that interacts directly with the sealed-artifact
  question (a bundled artifact may have no meaningful file URL). Flagged for P0-B's
  packaging answer; this is a likely source of silent Phase-3 breakage.

## Not verified here

Whether each listed builtin has a working `bare-*` counterpart, and the Windows-specific
behavior of any of them. That is P0-A's deliverable. This note only establishes WHAT
must be satisfied, not whether it can be.
