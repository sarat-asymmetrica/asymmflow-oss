# Phase 0 Notes — C: Go's wasip1 / go:wasmexport story (verified 2026-07-20)

Author: coder P0-C. Local Go: `go1.25.3 windows/amd64` (verified via `go version`).
Everything below is either a direct quote from a fetched doc, or a transcript from a
real build/enumerate experiment run in this session. No claim below is asserted from
training-vintage memory alone — where I could not verify something, it is listed in
§7.

---

## 1. VERDICT: the "no WASI imports needed" premise is FALSE

The Phase-1b premise as stated in the brief — that a `go:wasmexport` build needs "no
WASI import table at all" — **does not hold**. A `go:wasmexport` reactor built with
`-buildmode=c-shared` still imports **10 `wasi_snapshot_preview1` functions**. This is
smaller than a WASI *command* module (18 imports, our existing `reducer.wasm`), but it
is not zero, and it is not close to zero either — it's the Go runtime's own baseline
requirement, not something a `go:wasmexport` build can opt out of.

### Real transcript — throwaway reactor build

Built in scratchpad (`wasmexport-test/main.go`):

```go
package main

//go:wasmexport apply
func apply(a, b int32) int32 { return a + b }

func main() {}
```

```
$ GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o reactor.wasm .
EXIT 0
-rw-r--r-- 1 schan 197609 1677994 reactor.wasm   (~1.68 MB)
```

Enumerated with a small Node script (`node enumerate.mjs reactor.wasm`, Node v22.17.0,
using `WebAssembly.Module.imports`/`.exports`):

```
IMPORTS: 10
   wasi_snapshot_preview1 / sched_yield
   wasi_snapshot_preview1 / proc_exit
   wasi_snapshot_preview1 / args_get
   wasi_snapshot_preview1 / args_sizes_get
   wasi_snapshot_preview1 / clock_time_get
   wasi_snapshot_preview1 / environ_get
   wasi_snapshot_preview1 / environ_sizes_get
   wasi_snapshot_preview1 / fd_write
   wasi_snapshot_preview1 / random_get
   wasi_snapshot_preview1 / poll_oneoff
EXPORTS: 3
   _initialize (function)
   apply (function)
   memory (memory)
```

### Comparison transcript — repo's existing command-module `reducer.wasm`

Built from repo root exactly as instructed:

```
$ GOOS=wasip1 GOARCH=wasm go build -o mesh/dist/reducer.wasm ./mesh/cmd/reducer
EXIT 0
-rw-r--r-- 1 schan 197609 3963665 mesh/dist/reducer.wasm   (~3.96 MB)
```

Enumerated the same way:

```
IMPORTS: 18
   wasi_snapshot_preview1 / sched_yield
   wasi_snapshot_preview1 / proc_exit
   wasi_snapshot_preview1 / args_get
   wasi_snapshot_preview1 / args_sizes_get
   wasi_snapshot_preview1 / clock_time_get
   wasi_snapshot_preview1 / environ_get
   wasi_snapshot_preview1 / environ_sizes_get
   wasi_snapshot_preview1 / fd_write
   wasi_snapshot_preview1 / random_get
   wasi_snapshot_preview1 / poll_oneoff
   wasi_snapshot_preview1 / fd_close
   wasi_snapshot_preview1 / fd_read
   wasi_snapshot_preview1 / fd_write        (2nd entry — see note below)
   wasi_snapshot_preview1 / random_get      (2nd entry — see note below)
   wasi_snapshot_preview1 / fd_fdstat_get
   wasi_snapshot_preview1 / fd_fdstat_set_flags
   wasi_snapshot_preview1 / fd_prestat_get
   wasi_snapshot_preview1 / fd_prestat_dir_name
EXPORTS: 2
   _start (function)
   memory (memory)
```

(Note: `fd_write`/`random_get` appear twice in `WebAssembly.Module.imports()` — the
wasm binary itself declares two import entries with the same module/name, which is
legal in the wasm spec; not a script bug. Not investigated further — irrelevant to the
verdict.)

**Reading the diff**: the reactor's 10 imports are a strict subset of the command
module's set, minus the file-descriptor-table calls (`fd_close`, `fd_read`,
`fd_fdstat_get`, `fd_fdstat_set_flags`, `fd_prestat_get`, `fd_prestat_dir_name`) that
the command build pulls in for its stdin/stdout file plumbing. The 10-import set —
`sched_yield`, `proc_exit`, `args_get`, `args_sizes_get`, `clock_time_get`,
`environ_get`, `environ_sizes_get`, `fd_write`, `random_get`, `poll_oneoff` — is the Go
**runtime's own baseline WASI dependency**, present regardless of whether the program
touches files. This tracks with the goroutine scheduler (`poll_oneoff`/`sched_yield`
for the netpoller/timer wakeups), map seeding and Go's internal fastrand
(`random_get`), panic/stderr output (`fd_write`), `os.Args`/`os.Environ` support
(`args_*`, `environ_*`), and process exit (`proc_exit`). None of this is something a Go
program can compile away by using `go:wasmexport` instead of `func main()` — it's below
the language surface, in `runtime/`.

**Consequence for the campaign**: if Phase 1b's design depends on a WASI-import-free
module (e.g. for a host that refuses to supply `wasi_snapshot_preview1`, or a
"pure compute, no syscalls" security argument), that premise needs to be revised. The
real, verified story is "smaller WASI surface, not zero WASI surface." A host — Bare's
runtime included — still needs to supply (or stub) all 10 functions above to
instantiate any stock-Go wasip1 module, reactor or command.

---

## 2. Exact build invocation + host call protocol

**Build** (verified working, this session):
```
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o reactor.wasm .
```
`-buildmode=c-shared` is the flag — not a separate "reactor" flag. Quoting
[go.dev/blog/wasmexport](https://go.dev/blog/wasmexport): "The build flag signals to
the linker not to generate the `_start` function (the entry point for a command
module), and instead generate an `_initialize` function, which performs runtime and
package initialization, along with any exported functions and their dependencies."

**Host call protocol** (from the same post, corroborated by our export list above,
which does show `_initialize` and no `_start`):
1. Instantiate the module, supplying `wasi_snapshot_preview1` imports (from
   `node:wasi`, wazero's `wasi_snapshot_preview1.MustInstantiate`, wasmtime's WASI
   support, or Bare's own equivalent).
2. Call `_initialize` **once**, before any other export. Quote: "The `_initialize`
   function must be called before any other exported functions. The `main` function
   will not be automatically invoked."
3. The instance then stays alive; call exported functions (`apply` in our test, `add`
   in the blog's example) as many times as needed — this is the reactor's whole point
   versus a command module, whose memory "is consumed by its run" (see our own comment
   in `mesh/host/apply.mjs:24-26`, which is accurate for the command-module case and
   would need to change for a reactor).

Directive syntax, verbatim from
[pkg.go.dev/cmd/compile#hdr-WebAssembly_Directives](https://pkg.go.dev/cmd/compile#hdr-WebAssembly_Directives):
```go
//go:wasmexport h
func hWasm() { ... }
```
makes `hWasm` "available outside this WebAssembly module as `h`".

---

## 3. Memory/ABI pattern for JSON in and out

Go type → wasm type translation table, quoted verbatim from the same compiler doc
(applies to both `go:wasmimport` and `go:wasmexport`):

| Go type | Wasm type |
|---|---|
| bool | i32 |
| int32, uint32 | i32 |
| int64, uint64 | i64 |
| float32 | f32 |
| float64 | f64 |
| unsafe.Pointer | i32 |
| pointer | i32 (element type restricted — see below) |
| **string** | **(i32, i32) — "only permitted as a parameter, not a result"** |

"Any other parameter types are disallowed by the compiler." Pointer element types are
restricted to bool/intN/uintN/floatN, arrays of those, or a struct that (if non-empty)
embeds `structs.HostLayout` and only contains fields of permitted types. **Slices,
generic structs, maps, and interfaces cannot cross the boundary directly** — which
matters a lot for our reducer, whose `input`/state types are exactly the kind of
regular nested Go structs `encoding/json` targets, not `HostLayout` structs.

**The critical asymmetry: string works as a parameter, not as a return type.** So:

- **JS → Go direction** (host sends the op-log JSON in): a `go:wasmexport` function can
  take `func apply(opsJSON string) ...` directly — the compiler lowers this to a
  `(ptr i32, len i32)` export signature automatically. But the host still needs
  somewhere in guest memory to put those bytes before calling — Go's GC-managed heap
  is not something a host can just poke bytes into at an arbitrary offset. The
  documented pattern (quoting the
  [YokeBlogSpace writeup](https://yokecd.github.io/blog/posts/interfacing-with-webassembly-in-go/),
  which is the only source I found with a full worked example) is: the Go side
  additionally exports its own `malloc`:
  ```go
  //go:wasmexport malloc
  func malloc(size uint32) wasm.Buffer {
      return FromSlice(make([]byte, size))
  }
  ```
  and the host calls `malloc(len(jsonBytes))`, writes the JSON into
  `instance.exports.memory.buffer` at the returned offset, then calls the real export
  (`apply(ptr, len)`).

- **Go → JS direction** (converged state JSON out): since `string` can't be a *result*
  type, the return value has to be an `i32`/`i64` "fat pointer" (packed
  address+length, or a small struct-of-two-i32s convention) that the host decodes by
  reading `memory.buffer` afterward — there is no compiler-native "return this JSON
  string" path. The YokeBlogSpace pattern packs address+length into a `uint64`.

- **Ownership/free**: quoting the search summary of that same pattern, "the host owns
  that allocation, so must call the built-in export `free` when done" — i.e. a second
  `//go:wasmexport free` is also needed, mirroring manual C-style memory management at
  the boundary, layered on top of Go's own GC underneath.

- **GC pinning concern (real, filed, and Go-1.24-era)**: quoting
  [golang/go#69584](https://github.com/golang/go/issues/69584), a wasmexport function
  with a **void** (no-return-value) signature could crash "during garbage collection
  operations when calling void-exported functions compiled to WebAssembly," reported
  against `devel go1.24-c208b91`, milestoned `Go1.24`, marked fixed (issue closed,
  "NeedsFix" → resolved). I could not independently verify from the issue text alone
  whether the fix landed in the 1.25.3 we're running locally or is still open in some
  form — flagging as unverified in §7, but the underlying lesson holds regardless:
  **exported functions returning nothing are the discovered edge case; ours would
  return a fat pointer, i.e. always a non-void signature**, which the reporter's own
  workaround ("if I add a return value... the problems seem to go away") suggests
  sidesteps it.

**Net for our reducer**: none of `encoding/json`'s `Marshal`/`Unmarshal` calls change —
they still operate on ordinary Go structs entirely inside the wasm linear memory / Go
heap, exactly as they do today. What changes is *only* the two edges: instead of
`io.ReadAll(os.Stdin)` and `os.Stdout.Write(out)`, the boundary functions become
`malloc`/`apply(ptr,len)`/`free` doing byte copies into/out of a JS-visible
`ArrayBuffer`. This is a **packaging change**, not a semantics change — expanded on in
§6.

---

## 4. Restrictions & risks

- **Reentrancy / goroutines**: quoting go.dev/blog/wasmexport: "Wasm is a
  single-threaded architecture with no parallelism. A `go:wasmexport` function can
  spawn new goroutines. But if a function creates a background goroutine, it will not
  continue executing when the `go:wasmexport` function returns, until calling back
  into the Go-based Wasm module." So goroutines survive across calls (the instance
  stays warm, per §2), but they only get CPU time while some exported function is
  executing — there is no background scheduler ticking between host calls. Our
  reducer is a synchronous pure fold; this is a non-issue for us but would matter if
  Wave-future work added timers/background workers inside the wasm module.
- **`poll_oneoff`/`sched_yield` still imported**: confirmed directly in our own
  transcript (§1) — both appear in the 10-import reactor set. The Go scheduler depends
  on them even with zero goroutines spawned by user code, because the runtime itself
  starts background machinery (GC assist scheduling, etc.) that routes through the
  netpoller/timer path on wasip1. A host that wants to omit these WASI functions
  entirely is not an option with stock Go, reactor or not.
  Quoting go.dev/blog/wasi (command-module version, but same runtime underneath): "The
  scheduler can still schedule goroutines to run concurrently, and standard
  in/out/error is non-blocking, so a goroutine can execute while another reads or
  writes, but any host function call... will cause all goroutines to block until the
  host function call has returned" — i.e. no real parallelism, single-threaded
  cooperative scheduling, consistent with wasm's model generally.
- **Type surface is narrow**: per §3's table, nested structs/slices/maps can't cross
  the ABI boundary directly — everything has to go through the malloc/ptr+len/JSON
  channel. This is fine for us (we're already JSON-serializing at the boundary in the
  command-module version) but rules out a "richer" typed API without a serialization
  layer.
- **No pointers across the 64-bit/32-bit gap**: quoting the blog, "Due to the
  unfortunate mismatch between the 64-bit architecture of the client and the 32-bit
  architecture of the host, it is not possible to pass pointers in memory" for nested
  pointer fields — reinforces that JSON-over-bytes (not a "shared struct") is the
  right mental model, matching what we already do.
- **Binary size**: our own transcript shows the trivial reactor (`add`) alone is
  ~1.68 MB, and the full reducer package (with `encoding/json` + the fold logic) as a
  command module is ~3.96 MB. I did not build a reactor variant of the real reducer in
  this session (out of scope — see §7), so I can't yet report its exact reactor size,
  but expect it to land somewhat below the 3.96 MB command build (no stdin/stdout
  fd-table code) and well above the 1.68 MB trivial case (full `encoding/json` +
  reducer logic pulled in either way).

---

## 5. TinyGo assessment (secondary — stock Go preferred either way)

Our reducer (`mesh/reducer/**`, driven from `mesh/cmd/reducer/main.go:1-72`) leans on
`encoding/json` for both `Unmarshal` (parsing `input{Config, Ops}`) and `Marshal`
(serializing the converged `State`) — both reflection-heavy stdlib paths. Per
[tinygo.org/docs/guides/compatibility](https://tinygo.org/docs/guides/compatibility/)
and corroborating search results: "TinyGo doesn't yet implement reflection APIs needed
by `encoding/json`... `encoding/json` usage compiles, but panics at runtime due to
limited support for reflection," with the community workaround being non-reflective
JSON libraries (e.g. `gjson`) rather than the stdlib package. Since rewriting the
reducer's serialization onto a non-stdlib JSON library would be a real code change to
files we're told not to touch changing well-tested logic, **TinyGo is not a viable
drop-in for this reducer as written**. This confirms rather than merely accepts the
stated preference for stock Go — it isn't a preference, it's close to a requirement
given the current `encoding/json` dependency.

---

## 6. Semantics vs. packaging — the reducer's `go:wasmexport` variant

**What a reactor variant would look like**: a new file, e.g.
`mesh/cmd/reducer-reactor/main.go`, `//go:build wasip1`, exporting three functions —
`malloc(size uint32) uint32` (returns offset into guest memory), `free(ptr, size
uint32)`, and `apply(ptr, len int32) uint64` (or two i32 out-params via a second export
for length, since string/complex results can't be direct return types per §3) — that:
1. reads `len` bytes starting at `ptr` out of its own linear memory (available inside
   Go via a helper that reinterprets a `uintptr` as a `[]byte` — standard for this
   pattern, not something new),
2. `json.Unmarshal`s into the *same* `input{Mode, Config, Ops}` struct defined today,
3. calls the *same* `reducer.ApplyWithConfig` / `reducer.ApplyRoom` functions,
   unchanged,
4. `json.Marshal`s the result,
5. `malloc`s space for the output bytes, copies them in, and returns the fat pointer
   for the host to read and then `free`.

**Does this change reducer semantics?** No — and the argument is precise, not just
asserted: `mesh/cmd/reducer/main.go:52-61` is already a thin adapter — it does
`io.ReadAll` → `json.Unmarshal` → dispatch on `in.Mode` to `reducer.ApplyWithConfig` or
`reducer.ApplyRoom` → `json.Marshal` → `os.Stdout.Write`. Every one of the fold
functions (`reducer.ApplyWithConfig`, `reducer.ApplyRoom`) lives in
`mesh/reducer/**`, is untouched by this proposal (the brief explicitly scopes a new
file, not edits to `mesh/reducer/**`), and is not aware of — does not import, does not
reference — `os.Stdin`/`os.Stdout`/WASI at all today (confirmed by reading
`mesh/cmd/reducer/main.go`: the reducer package is only ever called with fully
in-memory Go values, `in.Config` and `in.Ops`, never with a stream or file handle).
The five steps above are the **exact same five transformations** (bytes-in →
unmarshal → fold → marshal → bytes-out) that the command module performs today; only
step 1 and step 5's *channel* changes — stdin/stdout file descriptors replaced by
malloc'd linear-memory offsets addressed by a JS host directly, instead of by
`node:wasi`'s file-descriptor shim in `mesh/host/apply.mjs:59-73`. The fold logic that
determines *what state results from what ops* — the actual "law" — does not move,
does not get re-expressed, and is not even recompiled differently (same
`mesh/reducer` package, same import). This is packaging/channel only, not semantics,
and is a clean, mechanically verifiable claim: a byte-for-byte diff of
`reducer.ApplyWithConfig`'s/`reducer.ApplyRoom`'s output on identical `(Config, Ops)`
input, called from either `main.go` (command) or a hypothetical
`reducer-reactor/main.go`, must be identical, because both call the same Go function
value with the same arguments — the WASI/wasmexport machinery only decides how those
arguments got assembled from bytes and how the result gets serialized back to bytes,
never what the fold computes.

One caveat worth flagging precisely rather than glossing over: the *op-log replay
model* itself (full log every call, per `mesh/host/apply.mjs:24-26`'s comment "a
command module's memory is consumed by its run") is what the command module does
today. A reactor's whole value proposition is **incremental** apply — feeding only new
ops into a warm instance that retains prior folded state in its own memory, rather than
replaying the full log each call. That would be a real behavior change (statefulness
across calls, no longer stateless-per-invocation) — but it is a change to the *host
call pattern* (how many/which ops get sent each time), not to the *fold function*
itself, which is pure and order-dependent-only-on-what-it's-given either way. Whether
to adopt incremental replay is a design decision for a later wave, not something this
note is claiming is free or already decided.

---

## 7. Not verified

- Whether golang/go#69584 (void-wasmexport + GC crash) is actually fixed in the
  1.25.3 we run locally, versus merely closed with a documented workaround — I read
  the issue via WebFetch summarization, not the raw issue thread or the Go 1.25
  release notes' fixed-bug list. Low risk to us regardless, since any reducer export
  we'd write returns a value (a fat pointer), which the reporter's own account
  suggests avoids the trigger — but I have not built and stress-tested a real
  wasmexport reducer variant, only the trivial `add`-style function in §1.
  Recommend a Phase 1b task, not a Phase 0 blocker.
- Did not build a `go:wasmexport` variant of the actual reducer (with
  `encoding/json` + `mesh/reducer` wired in) in this session — out of scope per the
  brief (new-file-only, and Phase 0 is documentation, not implementation). §1's size
  comparison and §6's semantics argument are therefore reasoned from the existing
  command-module build plus the documented ABI, not from a second real transcript of
  the reactor-reducer specifically. Flagging so Phase 1b doesn't treat §6 as
  already-proven-by-build — it's proven by argument from the existing code + verified
  ABI rules, not by a second experiment.
- Full `go.dev/blog/wasi` content on command-vs-reactor comparison and
  `_start`/`_initialize` distinction — that post predates `go:wasmexport` (Go 1.21 era)
  and, per my fetch, doesn't cover reactor mode at all; all reactor/`_initialize`
  claims in this note come from `go.dev/blog/wasmexport` (Go 1.24) instead, which does
  cover it directly and is quoted in §2.
- Did not check whether Bare's own WASI implementation (as opposed to Node's
  `node:wasi` or wazero) actually implements all 10 baseline `wasi_snapshot_preview1`
  functions from §1 — that's Bare-specific and outside this note's Go-side scope;
  flagging as a hard dependency for whoever owns the Bare-runtime host side (P0-A/B) to
  confirm.
- Multiple-return-value and `error`-type rules for `go:wasmexport` signatures — the
  compiler doc I fetched didn't state these explicitly and I didn't find a
  second source; if the reactor design needs multiple exported return values (e.g.
  ptr AND len as two separate results, rather than one packed value), verify against
  a real build before committing to a signature.
