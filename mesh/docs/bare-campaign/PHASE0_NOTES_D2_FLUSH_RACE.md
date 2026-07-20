# Phase 0 Notes — D2: The Bare Stdout Flush Race, Root-Caused

**Coder:** P0-D · **Branch:** `feat/fable-bare-runtime` · **Date:** 2026-07-20
**Scope:** root-cause the truncation finding from `PHASE0_NOTES_D_REVERIFY.md` §3. All work
done in `…/scratchpad/p0d-flush/` (outside the repo, per instruction — other agents are
actively writing `mesh/host/**`/`mesh/cmd/**`). Sole repo write is this file.
**Method note:** every rate below is a real measured count from a real run, never an
estimate. Runs use a purpose-built harness (`harness.mjs`, not committed — scratchpad only)
that spawns `bare.exe` as a Node child with `stdio: ['ignore', 'pipe', 'pipe']`, i.e. the exact
shape `guide.mjs`'s `runProbe()` already uses and DP4's sidecar bridge would use — a real OS
pipe read by a Node parent via `child.stdout.on('data', …)`, not a shell `$(...)` capture.

---

## 1. VERDICT — pipe vs TTY

**True interactive-console TTY could not be produced inside this tool environment** — verified
directly: a `bare-process` `process.stdout.isTTY` probe returned `undefined` in every
configuration attempted (direct invocation, `| cat`, redirected to a file). This tool's own
shell never hands a script a real console handle, so a TTY-only control group is not available
to me. This is stated plainly rather than assumed away — see §6.

What I *could* and did measure, at ≥30 runs each, against the real 3.96MB `reducer.wasm`:

| Stdout target | Write API | Async compile | Rate | 
|---|---|---|---|
| **Pipe** (Node parent, real production shape) | `console.log` | `await WebAssembly.compile()` | **14/30 OK (53% loss)** |
| **File redirect** (regular file, non-console, non-pipe) | `console.log` | `await WebAssembly.compile()` | **22/30 OK (27% loss)** |
| **Pipe** | `bare-process` `process.stdout.write()` | `await WebAssembly.compile()` | **0/30 OK — 30/30 HANG** (deadlock, see §2) |
| **Pipe** | `bare-process` `process.stdout.write()`, **no WASM at all** | none | **0/30 OK — 30/30 HANG** (same deadlock, WASM-independent) |
| **Pipe** | `console.log` | `new WebAssembly.Module()` (**sync**, mitigated) | **50/50 OK — 0 loss** |
| **File redirect** | `console.log` | `new WebAssembly.Module()` (**sync**, mitigated) | **30/30 OK — 0 loss** |

**Bottom line for the pipe-vs-TTY question as asked: the loss is real and severe on the
production-shaped pipe (53%), and it is NOT pipe-exclusive — it also reproduces on a plain
file redirect (27%), which rules out "cosmetic console/TTY teardown, safe on our real seam" as
the explanation.** This is not good news to report, but it is what was measured. The genuinely
good news is in §4: the mitigation is simple, cheap, and measured clean at 80/80 across both
non-console stdio targets.

There are, in fact, **two distinct bugs**, not one — see §3.

**The single most directly damning result in this investigation** is not from a synthetic
variant — it is the actual `mesh/host/bare-spike/stdio-check.mjs` script (the one the original
2026-07-19 report marked PASS, and the literal DP4 ndjson-over-stdio proof-of-concept), tested
byte-for-byte under the real production pipe topology instead of the shell-pipe test it was
originally verified with:

| Consumer | Result |
|---|---|
| Shell pipe (`printf '...' \| npx bare stdio-check.mjs`) — how it was originally verified | **PASS**, both frames print |
| `child_process.spawn` with `stdio: ['pipe','pipe','pipe']`, Node parent writing to `child.stdin` and reading `child.stdout` via `'data'` — **the real DP4 shape** | **`{"event":"ready"}` arrives 10/10; the `{"echoed":...}` reply — the actual message payload — is silently lost 10/10 (100%), process exits code 0, never a hang** |
| Same script, rewritten to remove the explicit `process.exit(0)` on stdin `'end'`, hoping the loop would drain naturally instead of cutting a write off | **Worse: 10/10 HANG**, confirms `process.exit()` was load-bearing for termination at all, not merely early |
| Same script, rewritten to route both writes through `console.log` instead of `bare-process`'s `process.stdout.write()` (Rule 2, §4) — everything else identical, `process.exit(0)` kept | **30/30 OK, 0 loss, 0 hangs** |

In plain terms: **the exact script this campaign already marked as a PASS silently drops
every single reply it sends, 100% of the time, when driven the way our production bridge
would actually drive it** — and the fix is not "add a delay" or "don't call exit" (both were
tried and both failed or made it worse), it is "don't use `bare-process`'s stdout stream for
the write path." Full detail in §2-4.

---

## 2. Isolation matrix (all pipe mode unless noted; ≥30 runs each; real counts)

| Variant | Description | Result |
|---|---|---|
| (a) | `console.log` × 3, exit. No wasm, no await. | **30/30 OK** |
| (b) | top-level `await Promise.resolve()`, then `console.log`. | **30/30 OK** |
| (c) | `await WebAssembly.compile()` of an 8-byte hand-built empty module (`\0asm\1\0\0\0`), then `console.log`. | **28/30 OK (2 lost, 7%)** |
| (d) | `await WebAssembly.compile()` of the real 3.96MB `reducer.wasm`, then `console.log`. | **14/30 OK (16 lost, 53%)** |
| (e) | same as (d) plus ~500 lines of output. | **22/30 OK (8 lost, 27%)** |
| (f) | same as (d) but writing via `bare-process`'s `process.stdout.write()` instead of `console.log`. | **0/30 OK — 30/30 HANG** |
| (g) | same as (d) plus an explicit `await new Promise(r => setTimeout(r, 50))` right after the compile resolves, before writing more. | **22/30 OK (8 lost, 27%) — delay did NOT fix it** |
| (h) | same as (f) but using a write-callback / `'drain'`-aware helper before exit (the most "correct" drain pattern `bare-process` exposes). | **0/5 OK — 5/5 HANG** (stopped at 5 runs once the pattern was clear) |
| (j) | `bare-process` `process.stdout.write('hello\n')` **with no `fs`, no WASM, no compile at all**. | **0/30 OK — 30/30 HANG** |
| (k) | `bare-process` imported (referenced, unused) but output goes through `console.log`, no WASM. | **30/30 OK, fast (~200-300ms/run)** |
| (l) | same as (d) but `WebAssembly.compile()` replaced with **synchronous** `new WebAssembly.Module(bytes)`. | **50/50 OK (pipe) + 30/30 OK (file) — 0 loss, 0 hangs** |

Every anomaly in (c)/(d)/(e)/(g) had the **identical shape**: stdout contained exactly the
first line (`bytes N`) and nothing written after the `await WebAssembly.compile()` — never a
partial mid-list truncation, never garbled output, always a clean stop right at the await
boundary. Every anomaly in (f)/(h)/(j) had the **identical shape**: zero bytes of stdout ever
observed, process still alive and "Responding: True" per `Get-Process`, killed only by the
harness's timeout — a true deadlock, not a fast exit racing a flush.

---

## 3. Root cause — two separate bugs, not one

### Bug A: async `WebAssembly.compile()` races Bare's own stdout teardown (partial loss, ~7-53%)

Isolated cleanly by the (d) → (l) delta: identical script, identical output, identical stdout
target — the **only** change is `await WebAssembly.compile(bytes)` → `new
WebAssembly.Module(bytes)` (synchronous, no await, no promise). That single change took the
loss rate from 53% (pipe) / 27% (file) to **0/80 across both targets**. This isolates the
trigger to the async path specifically, not WASM compilation itself, not output volume (e),
not compile duration/size (c: 8 bytes still loses 7% of the time; d: 3.96MB loses 53% of the
time — size correlates with *rate* but is not required for the bug to fire at all), and not a
simple timing race a fixed delay can paper over (g: a 50ms delay after the await changed
nothing). The delay result is the important negative control: if this were "the process just
needs a bit more wall-clock time before exiting," 50ms — far longer than any of these scripts'
total runtime — would have fixed it. It didn't. That rules out a naive "add a setTimeout
before exit" mitigation and points at something more structural: **the completion of an async
`WebAssembly.compile()` (which Bare/V8 resolves via a background thread-pool task posted back
onto the main loop) appears to leave Bare's own `console`/stdout write path in a state where a
write issued in that same continuation can be silently dropped rather than queued**, on a
target that isn't an interactive console. I did not get further than this without instrumenting
Bare's C sources directly (out of scope for a scratchpad spike) — see §6.

### Bug B: `bare-process`'s `process.stdout.write()` deadlocks unconditionally when the parent reads via a pipe — nothing to do with WASM

Isolated cleanly by variant (j): a two-line script, no `fs`, no `WebAssembly`, importing only
`bare-process` and calling `.stdout.write()` twice. **30/30 hung**, confirmed independently by
(f) and (h) (both 100% hang, both using the same API on top of real WASM work). Variant (k) —
same `bare-process` import present but output routed through the built-in `console.log`
instead — is **30/30 OK, fast**. This isolates the deadlock to the act of calling
`bare-process`'s `stdout.write()` specifically, when that stream's underlying pipe is being
read by a Node.js parent process via `child_process.spawn`'s piped stdio — not to importing
the module, not to WASM, not to any prior state. This is not a race (a race would show a
non-zero success rate); it is a **100% reproducible deadlock** in this exact topology. §5
below links this to a known, currently-unresolved class of Windows pipe/IPC problems already
on file with the Bare maintainers.

---

## 4. Mitigation — proven, ready to be made binding

**Rule 1 (mandatory): use synchronous `new WebAssembly.Module(bytes)`, never `await
WebAssembly.compile(bytes)`, for any WASM module a Bare host script needs to load before
exiting or before its next stdout write matters.** Proven **0 losses in 80 runs** (50 pipe +
30 file redirect) against the real, full-size `reducer.wasm`, replacing a measured 53%/27%
loss rate on the exact same script with only that one line changed. This is a drop-in
replacement — `new WebAssembly.Module()` has the identical return shape
(`WebAssembly.Module.imports()`/`.exports()` work the same on its result) as the resolved
value of `WebAssembly.compile()`; the only behavioral difference is that it blocks the thread
for the duration of the parse instead of yielding to the event loop, which for a
few-megabyte module compiled once at startup (or once per warm reducer instance, per the
reactor-path discussion in `PHASE0_NOTES_C_WASMEXPORT.md`/`PHASE0_GATE_C_VERIFICATION.md`) is
not a meaningful cost.

**Rule 2 (mandatory): never call `bare-process`'s `process.stdout.write()` (or anything built
on it) when the script's stdout is a pipe read by a parent process — i.e. never, for any DP4
sidecar/bridge script, since that IS the topology. Use Bare's own built-in `console.log`
instead.** `console.log` under Bare does not touch `bare-process`'s stream implementation at
all — variant (k) proves the two are genuinely different code paths, and only the
`bare-process` one deadlocks. This is a real constraint on the DP4 design, not just a style
preference: the ndjson-over-stdio protocol that `stdio-check.mjs` proved in
`PHASE0_NOTES_D_REVERIFY.md` §2.2 was written using `bare-process`'s `process.stdout.write()`
— and §1's table above shows the actual, literal `stdio-check.mjs` file, re-tested under the
real `child_process` pipe topology (not the shell-pipe test it originally passed), loses its
reply payload 10/10 with a clean exit (not even a hang, in that particular configuration —
the "no output at all" hang shape from variants (f)/(h)/(j) and the "first write OK, later
write from inside a callback lost" shape from this real script are evidently two faces of the
same underlying `bare-process` stdout unreliability, surfacing differently depending on
exactly when the process exits relative to the write). Removing the explicit `process.exit()`
to let the loop drain naturally was also tried on this real script and made things strictly
worse (10/10 hang instead of 10/10 silent loss) — ruling out "just don't call exit early" as a
fix. Rewriting the same script to route both writes through `console.log` instead, with
`process.exit()` kept, measured **30/30 OK, 0 loss** — direct, literal confirmation that Rule
2 fixes the actual sidecar script, not just a synthetic stand-in. **Any future DP4 stdio
adapter must be built on `console.log`-equivalent primitives for output, or on a different
bare stream package entirely (`bare-pipe`/`bare-tty` were not tested here — flagged in §6),
never on bare-process's `process.stdout`.**

Combined rule, stated as the binding pattern for every Bare host script in this campaign:

> **Load WASM synchronously (`new WebAssembly.Module()`), write output via Bare's native
> `console.log`/global console, never via `bare-process`'s `process.stdout`.** This combination
> measured 0 failures in 80 runs under both of the two non-console stdio targets tested.

Not yet proven: whether `console.log`'s reliability holds at higher output volumes than tested
here when combined with the sync-compile fix (variant (e)'s 500-line volume test was only run
against the *async* compile path, where it still showed loss — it was not re-run against the
sync-compile mitigation). Flagged in §6 as the one gap in an otherwise clean mitigation proof.

---

## 5. Upstream issue search

Searched `github.com/holepunchto/bare`, `bare-process`, `bare-pipe`, `bare-tty`, `bare-kit`
issues/PRs for stdout truncation, flush-on-exit, and exit-race reports (2026-07-20).

- **No exact match** for "stdout truncated after WebAssembly.compile()" or "bare-process
  stdout.write() hangs when piped" was found as a filed issue. This appears to be a genuinely
  new observation, not a known/tracked bug — worth filing upstream per the campaign's
  give-freely ethos, with this report's variant (j)/(k)/(l) transcripts as ready-made
  repro material.
- **holepunchto/bare PR #160** — "Drain closing handles after the exit event" — closely
  related in *kind* (an async-close-vs-stopped-event-loop race: "closing a pipe is async
  (`uv_close` needs a loop turn); the exit event fires after the loop has already stopped...so
  a close issued from exit is queued but never completes until `bare_teardown` pumps the
  loop"). **Closed without merging on 2026-07-02** — the maintainer (kasperisager) rejected
  the approach as violating teardown contracts. Not our exact bug (this PR is about handle
  *closing*, ours is about stdout *writes*), but it establishes the maintainers are actively
  wrestling with exit-timing/event-loop-teardown races in this exact runtime, unresolved as of
  three weeks before this investigation.
- **holepunchto/bare-kit#83** — worklet-exit detection: a host process reading a worklet's IPC
  stream "gets no signal" on exit — "no EOF on the IPC stream, no callback, nothing" — because
  the worklet thread parks holding its IPC fds open. Different subsystem (bare-kit's
  worklet/IPC layer, not plain process stdio) but the same *family* of symptom: a reader on the
  other end of a Bare-owned pipe not reliably observing the writer's end-of-life.
- **holepunchto/bare-kit#92** — win32-specific: "the win32 IPC layer had no EOF path" at all;
  `bare_ipc_read` never returned end-of-stream on Windows, and a real `UV_EOF` would have
  *aborted* the process (an assertion failure) rather than being handled. The resolution
  the maintainers converged on was not a Windows-specific pipe fix — they **decided to replace
  pipes with an in-process message queue entirely**, "eliminating platform-specific IPC
  complexities." This is the most directly corroborating evidence found: it establishes that
  **Windows pipe/IPC handling in the Holepunch/Bare ecosystem is a known-immature area that its
  own maintainers are actively moving away from**, independent of and prior to this
  investigation. It does not prove the exact same code path causes bugs A and B above (bare-kit
  uses `bare-pipe`'s IPC primitives, not the process's own stdio streams), but it is strong
  circumstantial support that "Bare + Windows + pipes" is exactly the kind of seam where this
  class of bug lives.
- No issue found specifically discussing `WebAssembly.compile()` and Bare/libuv interaction —
  the WebAssembly.compile async-completion-scheduling concern (§3, Bug A) appears to be
  unexplored territory upstream as far as this search could determine.

**Recommendation**: file a new issue on `holepunchto/bare` (or `bare-process`, since Bug B is
specific to that package) with this report's variants (j) (minimal repro, no WASM) and (d)→(l)
(minimal repro, WASM-specific) attached as reproductions. Not filed by this coder — no
upstream-facing action was authorized for this spike; flagging as the clear next step.

---

## 6. Not verified

- **True interactive TTY was not testable in this sandboxed tool environment** (`isTTY`
  returned `undefined` in every configuration attempted). All evidence in this report is
  pipe-vs-file, not pipe-vs-TTY. Since production is 100% pipe-shaped (a spawned sidecar always
  has its stdout read programmatically, never watched on a live console), this gap does not
  change the mitigation recommendation, but the literal "is it TTY-exclusive" framing of the
  original question could not be answered with a genuine control group.
- **Root cause of Bug A is narrowed, not fully explained.** I identified *that* switching from
  async `.compile()` to sync `new WebAssembly.Module()` eliminates it and *that* a fixed delay
  does not, which rules out several hypotheses (pure timing, output volume, module size as a
  hard requirement) but I did not instrument Bare's C/libuv internals to name the exact
  mechanism (e.g., whether the WASM background-compile thread's completion callback is
  competing with a libuv pipe-write completion callback for the same loop tick, or something
  else). An honest "narrowed to the async-compile boundary, not root-caused below that" is the
  correct claim.
- **Bug B's exact internal mechanism** (why `bare-process`'s stream never completes even its
  first write to a Node-read pipe) was not traced into `bare-process`'s or `bare-pipe`'s
  source. The upstream corroboration in §5 (bare-kit#92) is suggestive, not proof of the same
  code path.
- **Variant (e)'s 500-line volume test was only run against the async-compile (buggy) path,
  not re-run against the sync-compile mitigation.** The mitigation is proven clean at the
  output volumes actually used in (d)/(l) (~20 lines), not at (e)'s scale. A production ndjson
  sidecar emitting many frames per session should re-verify at realistic volume before this
  mitigation is trusted at scale.
- **`bare-pipe` and `bare-tty` packages were not tested as alternative stdio primitives.** If a
  future DP4 design wants something more capable than `console.log` (e.g. binary-safe
  buffering, backpressure awareness) for the sidecar's actual write path, these are the natural
  next candidates to test — not evaluated here.
- **No upstream issue was filed.** §5's recommendation is a recommendation, not an action taken
  this session.
- Did not test whether Bug A's rate changes with concurrent WASM compiles (e.g. two sidecar
  instances compiling simultaneously) — all matrix runs were single-process, sequential.
- The §1 real-`stdio-check.mjs` re-test used `bare-process`'s `process.stdin` for **reading**
  in every variant (including the fixed one) — only the **write** side was swapped to
  `console.log`. Reading via `bare-process`'s `process.stdin` appears reliable in all these
  runs (the `'data'`/`'end'` handlers visibly fired — inferred from the process exiting via
  `process.exit(0)` inside the `'end'` handler rather than hanging, in the exit-kept variants),
  but this was not isolated and stress-tested the way the write path was. If a future DP4
  adapter needs to read framed input as well as write framed output, the read side should get
  the same rigor before being trusted at scale.
