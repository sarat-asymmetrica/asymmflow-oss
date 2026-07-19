# Bare Sidecar Spike — Findings Report (Band 4, Mission A2 "The Corridor")

**Outcome: Deliverable B (findings report).** Bare is further along than expected —
the entire Holepunch JS stack and stdio ndjson framing work under Bare on Windows
with **zero shims**. The spike is blocked on exactly one thing: a WASI preview1
host for the Go reducer, which does not exist as an npm package today. This is
a scoped, well-characterized gap, not a dead end.

## Environment

- OS: Windows 11 Pro (10.0.26200), same machine as the rest of Mission A2.
- Node: v22.17.0 (bundled elsewhere in the mission; used here only to run npm).
- Bare: `1.30.3` (`bare-runtime-win32-x64` prebuild resolved and ran with no
  build step — `npx bare --version` → `v1.30.3`).
- Installed as devDependencies in `mesh/package.json` (I6 exception, this band
  only): `bare`, `bare-fs`, `bare-crypto`, `bare-events`, `bare-process`,
  `bare-stream` (the last four came along as transitive/explicit compat
  modules; only `bare` and `bare-process`/`bare-fs` are actually exercised by
  the scripts below — the others were installed to probe availability and are
  harmless to leave, or trim at gate discretion).

## What RAN under Bare (with transcripts)

### 1. `hello.mjs` — does Bare execute at all on this machine

```
$ npx bare host/bare-spike/hello.mjs
bare hello-world [
  'C:\\Projects\\asymmflow\\asymmflow-mesh\\mesh\\node_modules\\bare-runtime-win32-x64\\bin\\bare.exe',
  'C:\\Projects\\asymmflow\\asymmflow-mesh\\mesh\\host\\bare-spike\\hello.mjs'
]
```

PASS. No native build step, no admin rights, no separate install — `npm i -D
bare` pulled the Windows x64 prebuilt binary and it just runs.

### 2. `stdio-check.mjs` — the DP4 transport primitive (ndjson over stdio)

```
$ printf '{"id":1,"method":"hello","params":{}}\n' | npx bare host/bare-spike/stdio-check.mjs
{"event":"ready"}
{"echoed":"{\"id\":1,\"method\":\"hello\",\"params\":{}}"}
```

PASS. `bare-process`'s `process.stdin`/`process.stdout` behave exactly like
Node's stream objects for this purpose — the same newline-delimited-JSON
buffering loop used in `bridge-server.mjs`/`bridge-client.mjs` (TCP sockets)
and MESSENGER_UI_CAMPAIGN.md §1's "sidecar stdio, same frames" plan ports
verbatim. This is the actual DP4 seam and it works.

### 3. `require-check.mjs` — does the Holepunch stack load under Bare

```
$ npx bare host/bare-spike/require-check.mjs
OK   corestore
OK   autobase
OK   hyperswarm
OK   hyperdht
OK   hyperbee
OK   hyperblobs
OK   hypercore
OK   hypercore-id-encoding
OK   blind-peer
OK   blind-peering
OK   protomux-wakeup
```

PASS, all 11/11, with **no `"bare"` export condition present** in any of
these packages' installed `package.json` (checked `corestore`,
`autobase`, `hyperswarm` — all plain `"main": "index.js"`). Bare's own
`require()`/`import` implements enough of Node's core module surface
(`events`, `fs`, `crypto`, `path`, etc.) natively that these modules resolve
without a compat shim. This is the strongest positive signal in the spike —
the actual mesh dependencies (the things `mesh-node.mjs`, `social-room.mjs`,
etc. build on) are not the blocker.

### 4. `wasi-check.mjs` — the reducer's runtime dependency

```
$ npx bare host/bare-spike/wasi-check.mjs
WebAssembly typeof: object
FAIL node:wasi :: MODULE_NOT_FOUND :: Cannot find module 'node:wasi' imported from ...
FAIL wasi :: MODULE_NOT_FOUND :: Cannot find module 'wasi' imported from ...
FAIL bare-wasi :: MODULE_NOT_FOUND :: Cannot find module 'bare-wasi' imported from ...
```

`npm view bare-wasi` → `404 Not Found`. No official Holepunch WASI package
exists on npm (checked; only unrelated `wasi`-named packages like
`@bytecodealliance/preview2-shim` and `@wasmer/wasi`, neither Bare-native).

`WebAssembly` itself IS present (`typeof WebAssembly === 'object'`) — Bare has
a real WASM engine. The gap is specifically the **host-side WASI syscall
table**, which `node:wasi` provides in Node and nothing provides in Bare.

### 5. `wasm-compile-check.mjs` — does Bare's WebAssembly engine parse OUR module

```
$ npx bare host/bare-spike/wasm-compile-check.mjs
reducer.wasm bytes: 3963498
OK   WebAssembly.compile(reducer.wasm) succeeded
required import namespaces: [ 'wasi_snapshot_preview1' ]
import count: 18
```

PASS. `WebAssembly.compile` on the real `mesh/dist/reducer.wasm` (the Go
reducer `apply.mjs` drives in production) succeeds under Bare. The module's
*only* unsatisfied import namespace is `wasi_snapshot_preview1` — confirming
the blocker is precisely, and only, the WASI host.

### 6. `wasi-imports-list.mjs` — the exact shim surface needed

```
$ npx bare host/bare-spike/wasi-imports-list.mjs
wasi_snapshot_preview1.sched_yield
wasi_snapshot_preview1.proc_exit
wasi_snapshot_preview1.args_get
wasi_snapshot_preview1.args_sizes_get
wasi_snapshot_preview1.clock_time_get
wasi_snapshot_preview1.environ_get
wasi_snapshot_preview1.environ_sizes_get
wasi_snapshot_preview1.fd_write
wasi_snapshot_preview1.random_get
wasi_snapshot_preview1.poll_oneoff
wasi_snapshot_preview1.fd_close
wasi_snapshot_preview1.fd_read
wasi_snapshot_preview1.fd_write
wasi_snapshot_preview1.random_get
wasi_snapshot_preview1.fd_fdstat_get
wasi_snapshot_preview1.fd_fdstat_set_flags
wasi_snapshot_preview1.fd_fdstat_set_flags
wasi_snapshot_preview1.fd_prestat_get
wasi_snapshot_preview1.fd_prestat_dir_name
```

16 distinct syscalls (18 import entries, a couple repeated in the table).
This matches `apply.mjs`'s own channel design exactly (file-descriptor
stdin/stdout, no sockets, no threads, no signals): args/env introspection,
clock, random, and a minimal fd read/write/stat/close/prestat set — the
smallest possible WASI Command surface Go's `GOOS=wasip1` runtime emits for
a stdin→stdout batch tool. There is no `path_open`, no `fd_seek`, no
directory-tree walking — the reducer's own I/O contract (temp-file fds
handed in via `stdin`/`stdout`, per `apply.mjs`'s header comment) keeps the
needed surface small.

## What BLOCKED

Exactly one thing: **no WASI preview1 host implementation exists for Bare**,
neither built into the runtime nor as an npm package (`bare-wasi` doesn't
exist; `node:wasi` isn't polyfilled). Everything downstream of that — running
`apply.mjs`'s `applyViaWasm()`, and therefore `mesh-node.mjs`'s state
materialization, and therefore `bridge-server.mjs`'s methods — is unreachable
under Bare until this exists. This spike did not attempt to hand-port
`bare-bridge.mjs`/a stdio transport adapter for `bridge-server.mjs` itself,
because doing so would exercise `mesh-node.mjs` → `apply.mjs` on the very
first `roomState()`/`post()` call and hit this same wall immediately — timeboxing
here in favor of nailing down the *exact* blocker (§6 above) was the better use
of the spike's time than writing an adapter that cannot run end-to-end yet.

## Shim inventory (for a future DP4 push)

| Layer | Status under Bare | Shim needed |
|---|---|---|
| Runtime execution | PASS | none |
| stdio ndjson transport | PASS | none |
| corestore/autobase/hyperswarm/hyperdht/hyperbee/hyperblobs/hypercore/blind-peer/blind-peering/protomux-wakeup | PASS | none |
| WebAssembly engine + module parse | PASS | none |
| WASI preview1 host (16 syscalls) | **BLOCKED** | write a Bare-native WASI preview1 shim (fd_read/fd_write/fd_close/fd_fdstat_get/fd_fdstat_set_flags/fd_prestat_get/fd_prestat_dir_name on top of `bare-fs`; args_get/args_sizes_get/environ_get/environ_sizes_get from `Bare.argv`/`bare-process` env; clock_time_get/random_get/sched_yield/proc_exit are near-trivial) |

## Recommended DP4 path

**Bare-ready except the reducer's WASI host.** Two options, in the order I'd
pursue them:

1. **Port a minimal WASI preview1 shim** scoped to exactly the 16 syscalls in
   §6, built on `bare-fs`/`bare-process`. This is a bounded, few-hundred-line
   task (Node's own `lib/wasi.js` is the reference shape to port, minus
   everything the reducer doesn't call — no `path_open`, no directory walks,
   no signals). Given how narrow the surface is, this looks tractable inside
   a single future wave, not a research problem.
2. **Fallback**: if the shim proves harder than it looks once started, change
   `apply.mjs`'s channel from WASI-command-module-over-fds to a
   `//go:wasmexport apply()` build (already flagged as the "next step" in
   `apply.mjs`'s own header comment, independent of Bare) — a plain
   `WebAssembly.instantiate` with exported functions needs **no WASI import
   table at all**, sidestepping this blocker entirely rather than shimming
   around it. Worth flagging to the gate as it may be the cheaper fix
   regardless of Bare, since it also removes the current temp-file-per-call
   design in `apply.mjs`.

Until either lands, **stay on the Node sidecar** for DP4 (the current
localhost-TCP bridge already proves the exact wire shape stdio will use, per
§1's own framing) — Bare is not a regression risk to switch to later, since
everything except the reducer already works.

## Files

- `mesh/host/bare-spike/hello.mjs`
- `mesh/host/bare-spike/stdio-check.mjs`
- `mesh/host/bare-spike/require-check.mjs`
- `mesh/host/bare-spike/wasi-check.mjs`
- `mesh/host/bare-spike/wasm-compile-check.mjs`
- `mesh/host/bare-spike/wasi-imports-list.mjs`
- `mesh/host/bare-spike/BARE_SPIKE_REPORT.md` (this file)
- `mesh/package.json` (+devDependencies: bare, bare-fs, bare-crypto,
  bare-events, bare-process, bare-stream)
- `mesh/package-lock.json` (lockfile update for the above)

No files outside `mesh/host/bare-spike/` and `package.json`/`package-lock.json`
devDeps were touched (I1).
