// wasi-preview1-lite.mjs — a Bare-native (and Node-compatible) WASI preview1
// host, scoped to EXACTLY the 16 wasi_snapshot_preview1 syscalls the Go
// reducer's wasip1 command module (mesh/dist/reducer.wasm) imports, per the
// verified import table (mesh/docs/bare-campaign/PHASE0_GATE_C_VERIFICATION.md):
// args_get, args_sizes_get, clock_time_get, environ_get, environ_sizes_get,
// fd_close, fd_fdstat_get, fd_fdstat_set_flags, fd_prestat_dir_name,
// fd_prestat_get, fd_read, fd_write, poll_oneoff, proc_exit, random_get,
// sched_yield.
//
// Why this file exists: no `bare-wasi` package exists (mesh/docs/bare-campaign/
// PHASE0_NOTES_A_BARE.md §2.4/§4 — confirmed 404 on npm, and explicitly tagged
// 🔴 Unsupported in Holepunch's own bare-node compat catalog). `node:wasi`
// obviously doesn't run under Bare either. This is the gap-filler.
//
// DUAL-RUNTIME BY CONSTRUCTION: this file has ZERO `node:`/`bare-*` imports.
// The only cross-runtime global it relies on is `Buffer` — verified present,
// unimported, in both Node (`node -e "typeof Buffer"` -> function) and Bare
// (`npx bare -e "typeof Buffer"` -> function; `typeof crypto`/`TextEncoder`
// are BOTH undefined under Bare, so those were deliberately avoided). All
// linear-memory reads/writes and all string encode/decode go through Buffer
// views over the instance's memory — nothing else is required. This is what
// makes ONE shim body correct under both hosts: the caller (apply.mjs's Bare
// counterpart, apply-bare.mjs) injects the actual I/O backing (in-memory
// buffers, no filesystem), and this file never has to know which runtime
// it's executing under.
//
// ABI reference: WebAssembly/WASI legacy preview1 witx, fetched this session
// from the `wasi-0.1` tag (the `main` branch has moved on to the 0.2+
// Component Model and no longer carries the preview1 witx tree):
//   preview1/witx/typenames.witx           — struct/enum layouts
//   preview1/witx/wasi_snapshot_preview1.witx — function signatures
// Cross-checked against the known canonical struct sizes (fdstat=24B,
// prestat=8B, subscription=48B, event=32B, iovec/ciovec=8B) — all matched,
// no surprises. Errno numbering taken verbatim from the witx enum order
// (success=0 ... notcapable=76; the ones this shim actually returns:
// badf=8, inval=28, nosys=52, notcapable=76).
//
// R3 (owner ruling): this file is authored from the witx spec + general WASI
// knowledge, not copy-pasted from any package; nothing here ships from an
// external dependency. Node's own `lib/wasi.js` was fetched too but its ABI
// logic lives in a C++ native binding, not the JS file, so it was not usable
// as a porting source (see PHASE1A_REPORT.md).
//
// Factory shape: createWASI({ args, env, fds, random }) -> { imports, setMemory }.
// This diverges from the brief's suggested `{ args, env, stdin, stdout,
// preopens }` shape because our reducer only ever touches 3 well-known fds
// (0/1/2) with no directory tree — a generic `fds: { [n]: handle }` map is
// the honest shape for that surface (see the fd-table doc below), and it is
// exactly what a future `path_open`-bearing reducer would extend, not
// replace. `preopens` isn't a parameter because we have none — see
// fd_prestat_get below for why that's implemented as an honest EBADF table,
// not a fiction.
//
// fd handle shape (all synchronous, all optional):
//   { read(dst: Buffer) -> bytesWritten,   // 0 = EOF; omit if fd is not readable
//     write(src: Buffer) -> bytesConsumed, // omit if fd is not writable
//     close() -> void }                    // omit if nothing to release
export function createWASI({ args = [], env = {}, fds = {}, random } = {}) {
  let memory = null

  function buf() {
    if (!memory) throw new Error('wasi-preview1-lite: setMemory() was never called')
    return Buffer.from(memory.buffer, 0, memory.buffer.byteLength)
  }

  // errno values per the witx `$errno` enum's declaration order (verified
  // this session against preview1/witx/typenames.witx on the wasi-0.1 tag).
  const E = { SUCCESS: 0, TOOBIG: 1, BADF: 8, FAULT: 21, INVAL: 28, IO: 29, NOSYS: 52, NOTCAPABLE: 76 }

  // The fd table is shim-local state, not a passthrough to any real
  // filesystem — honest per the campaign brief: bare-fs has openSync/
  // closeSync/readSync/writeSync/statSync, but the reducer never calls
  // path_open, so there is nothing to back with bare-fs here. Every fd this
  // shim ever knows about is one the caller injected up front.
  const table = new Map()
  for (const [k, handle] of Object.entries(fds)) table.set(Number(k), handle)
  // fd_fdstat_set_flags has no bare-fs analog at all (P0-A §5, confirmed by
  // P0-D's cross-check) — Node's fs has no "set flags on an already-open fd"
  // primitive either. Tracked purely as shim state; fd_fdstat_get reports
  // back whatever was last set here (0 until touched).
  const fdFlags = new Map()

  function encode(str) { return Buffer.from(str, 'utf8') }

  // ── args / environ ────────────────────────────────────────────────────
  function args_sizes_get(argcPtr, argvBufSizePtr) {
    const b = buf()
    const encoded = args.map(encode)
    const bufSize = encoded.reduce((s, a) => s + a.length + 1, 0) // +1 per NUL
    b.writeUInt32LE(args.length, argcPtr)
    b.writeUInt32LE(bufSize, argvBufSizePtr)
    return E.SUCCESS
  }

  function args_get(argvPtr, argvBufPtr) {
    const b = buf()
    let bufCursor = argvBufPtr
    for (let i = 0; i < args.length; i++) {
      const enc = encode(args[i])
      enc.copy(b, bufCursor)
      b[bufCursor + enc.length] = 0 // NUL terminator
      b.writeUInt32LE(bufCursor, argvPtr + i * 4)
      bufCursor += enc.length + 1
    }
    return E.SUCCESS
  }

  function environAsStrings() {
    return Object.entries(env).map(([k, v]) => `${k}=${v}`)
  }

  function environ_sizes_get(countPtr, bufSizePtr) {
    const b = buf()
    const encoded = environAsStrings().map(encode)
    const bufSize = encoded.reduce((s, e) => s + e.length + 1, 0)
    b.writeUInt32LE(encoded.length, countPtr)
    b.writeUInt32LE(bufSize, bufSizePtr)
    return E.SUCCESS
  }

  function environ_get(environPtr, environBufPtr) {
    const b = buf()
    const entries = environAsStrings()
    let bufCursor = environBufPtr
    for (let i = 0; i < entries.length; i++) {
      const enc = encode(entries[i])
      enc.copy(b, bufCursor)
      b[bufCursor + enc.length] = 0
      b.writeUInt32LE(bufCursor, environPtr + i * 4)
      bufCursor += enc.length + 1
    }
    return E.SUCCESS
  }

  // ── clock / random / scheduling ───────────────────────────────────────
  // clock_time_get: `precision` (2nd param) arrives as a BigInt (i64 in the
  // core wasm ABI) — accepted but not used to bound resolution; a JS
  // Date.now()-derived nanosecond value is monotonic-enough for a batch
  // command tool that never sleeps. `id` (realtime vs monotonic vs the two
  // cpu-time clocks) is ignored for the same reason: every clock this
  // process could ask for collapses to "current wall time" here.
  function clock_time_get(_id, _precision, timePtr) {
    const ns = BigInt(Date.now()) * 1_000_000n
    buf().writeBigUInt64LE(ns, timePtr)
    return E.SUCCESS
  }

  // Honesty note (per P0-D's decision table): Go's runtime calls random_get
  // once at process init to seed its internal map-hash seed (runtime
  // fastrand/maphash init on GOOS=wasip1) — verified by grep that neither
  // mesh/reducer/*.go nor cmd/reducer/main.go import crypto/rand or
  // math/rand outside _test.go files, so nothing in the reducer's own logic
  // ever consumes random_get's output. The digest is SHA-256 over a
  // marshaled JSON value whose map keys Go's encoding/json sorts
  // alphabetically before emission — map iteration order (the one thing the
  // hash seed affects) never reaches the output bytes. Given that, this
  // shim defaults to Math.random() (a JS global present in both runtimes,
  // no crypto import needed) rather than pulling in bare-crypto/node:crypto
  // for a value proven not to affect byte-identity. A caller who wants real
  // entropy anyway (this shim is not reducer-specific) can inject
  // `random(n)` returning a Buffer of n bytes.
  function random_get(bufPtr, bufLen) {
    const b = buf()
    if (typeof random === 'function') {
      const bytes = random(bufLen)
      bytes.copy(b, bufPtr, 0, bufLen)
    } else {
      for (let i = 0; i < bufLen; i++) b[bufPtr + i] = (Math.random() * 256) | 0
    }
    return E.SUCCESS
  }

  // No real OS thread to yield from (single JS call stack, no wasm threads
  // in this build) — a no-op success is the correct answer, not a stub.
  function sched_yield() { return E.SUCCESS }

  // proc_exit is `@witx noreturn` — it must never return control to the
  // caller. The only way to unwind out of a synchronous host-import call in
  // JS is to throw; the caller (apply-bare.mjs) catches this specific class
  // to recover the exit code, exactly mirroring how node:wasi's own
  // `returnOnExit` unwinds internally (their C++ binding does the
  // equivalent throw/catch one layer down, not visible from JS — see
  // PHASE1A_REPORT.md on why lib/wasi.js itself couldn't be read as a
  // porting source for this).
  class WASIExit extends Error {
    constructor(code) { super(`wasi proc_exit(${code})`); this.code = code }
  }
  function proc_exit(code) { throw new WASIExit(code) }

  // ── fd table ─────────────────────────────────────────────────────────
  function fd_close(fd) {
    const h = table.get(fd)
    if (!h) return E.BADF
    if (typeof h.close === 'function') h.close()
    table.delete(fd)
    fdFlags.delete(fd)
    return E.SUCCESS
  }

  // fdstat is 24 bytes: filetype(u8)@0, pad(1), flags(u16)@2, pad(4),
  // rights_base(u64)@8, rights_inheriting(u64)@16 (verified against
  // typenames.witx field order + natural C alignment; matches the
  // well-known wasi-libc __wasi_fdstat_t size of 24).
  function fd_fdstat_get(fd, ptr) {
    const h = table.get(fd)
    if (!h) return E.BADF
    const b = buf()
    const CHARACTER_DEVICE = 2 // $filetype enum position (unknown=0, block_device=1, character_device=2)
    b.writeUInt8(CHARACTER_DEVICE, ptr)
    b.writeUInt8(0, ptr + 1)
    b.writeUInt16LE(fdFlags.get(fd) ?? 0, ptr + 2)
    b.writeUInt32LE(0, ptr + 4) // padding to the u64 boundary
    // rights: not enforced (nothing in this shim's surface calls
    // fd_fdstat_set_rights or path_open), so report "everything" rather
    // than fabricate a granular set nothing checks.
    b.writeBigUInt64LE(0xffffffffffffffffn, ptr + 8)
    b.writeBigUInt64LE(0xffffffffffffffffn, ptr + 16)
    return E.SUCCESS
  }

  // No bare-fs equivalent exists for "set flags on an open fd" (P0-A §5) —
  // this is synthesized shim-local state, tracked and read back by
  // fd_fdstat_get above, not a passthrough to any real primitive.
  function fd_fdstat_set_flags(fd, flags) {
    if (!table.has(fd)) return E.BADF
    fdFlags.set(fd, flags)
    return E.SUCCESS
  }

  // No bare-fs equivalent exists at all — preopens are a WASI-only concept
  // (P0-A §5). This shim preopens NOTHING (the reducer never calls
  // path_open, so there is nothing to preopen), which means the honest
  // answer for every fd is EBADF. This is exactly the signal Go's wasip1
  // runtime uses to know where its preopen-fd probing (fd 3, 4, 5, ...)
  // ends — returning EBADF here isn't a cop-out, it's the correct preview1
  // response for "not a preopen," and it's what lets the runtime stop
  // probing immediately instead of hanging or erroring.
  function fd_prestat_get(_fd, _ptr) { return E.BADF }
  function fd_prestat_dir_name(_fd, _pathPtr, _pathLen) { return E.BADF }

  // iovec/ciovec are 8 bytes each: buf(u32 ptr)@0, buf_len(u32)@4 (verified
  // against typenames.witx — both fields are u32-sized/aligned, no padding).
  function fd_read(fd, iovsPtr, iovsLen, nreadPtr) {
    const h = table.get(fd)
    if (!h || typeof h.read !== 'function') return E.BADF
    const b = buf()
    let total = 0
    for (let i = 0; i < iovsLen; i++) {
      const base = iovsPtr + i * 8
      const ptr = b.readUInt32LE(base)
      const len = b.readUInt32LE(base + 4)
      if (len === 0) continue
      const dst = b.subarray(ptr, ptr + len)
      const n = h.read(dst)
      total += n
      if (n < len) break // short read (EOF) — WASI permits partial fulfillment
    }
    b.writeUInt32LE(total, nreadPtr)
    return E.SUCCESS
  }

  function fd_write(fd, iovsPtr, iovsLen, nwrittenPtr) {
    const h = table.get(fd)
    if (!h || typeof h.write !== 'function') return E.BADF
    const b = buf()
    let total = 0
    for (let i = 0; i < iovsLen; i++) {
      const base = iovsPtr + i * 8
      const ptr = b.readUInt32LE(base)
      const len = b.readUInt32LE(base + 4)
      if (len === 0) continue
      const src = b.subarray(ptr, ptr + len)
      const n = h.write(src)
      total += n
      if (n < len) break
    }
    b.writeUInt32LE(total, nwrittenPtr)
    return E.SUCCESS
  }

  // ── poll_oneoff ──────────────────────────────────────────────────────
  // subscription = 48 bytes: userdata(u64)@0, tag(u8)@8 [0=clock,1=fd_read,
  // 2=fd_write], union payload @16 (padded to the u64 alignment of
  // subscription_clock's members). subscription_clock: id(u32)@16,
  // timeout(u64)@24, precision(u64)@32, flags(u16)@40.
  // subscription_fd_readwrite: file_descriptor(u32)@16.
  // event = 32 bytes: userdata(u64)@0, error(u16)@8, type(u8)@10,
  // fd_readwrite{nbytes(u64)@16, flags(u16)@24}.
  // (Both sizes verified against typenames.witx field order + alignment;
  // match the well-known wasi-libc __wasi_subscription_t=48,
  // __wasi_event_t=32.)
  //
  // Honest simplification: this shim never actually blocks. Every fd this
  // shim manages has its data fully available synchronously (in-memory
  // buffers, no real async I/O) and this is a single-goroutine batch
  // command with no legitimate reason to sleep — so every subscription,
  // clock or fd, resolves as "ready" on the same call, with the clock case
  // reported as "the timeout already elapsed" and the fd case reported as
  // "at least one byte is available" (true for our fds) or BADF (for a fd
  // this shim doesn't know). If poll_oneoff is ever hit with a subscription
  // this can't honestly resolve, that would be a real gap — Phase 1's
  // parity run is what proves it does or doesn't happen for the real
  // reducer (see PHASE1A_REPORT.md; P0-D flagged this as "TO VERIFY").
  function poll_oneoff(inPtr, outPtr, nsubscriptions, nEventsPtr) {
    const b = buf()
    for (let i = 0; i < nsubscriptions; i++) {
      const subBase = inPtr + i * 48
      const userdata = b.readBigUInt64LE(subBase)
      const tag = b.readUInt8(subBase + 8)
      const evtBase = outPtr + i * 32
      b.writeBigUInt64LE(userdata, evtBase)
      let error = E.SUCCESS
      let nbytes = 0n
      if (tag === 0) {
        // clock: report the timeout as already elapsed
      } else if (tag === 1 || tag === 2) {
        const fd = b.readUInt32LE(subBase + 16)
        const h = table.get(fd)
        if (!h) error = E.BADF
        else nbytes = 1n // "at least something is ready" — true for our fds
      } else {
        error = E.INVAL
      }
      b.writeUInt16LE(error, evtBase + 8)
      b.writeUInt8(tag, evtBase + 10)
      b.writeUInt8(0, evtBase + 11)
      b.writeUInt32LE(0, evtBase + 12) // padding to the fd_readwrite u64 field
      b.writeBigUInt64LE(nbytes, evtBase + 16)
      b.writeUInt16LE(0, evtBase + 24) // eventrwflags: no hangup
    }
    b.writeUInt32LE(nsubscriptions, nEventsPtr)
    return E.SUCCESS
  }

  return {
    imports: {
      wasi_snapshot_preview1: {
        args_get, args_sizes_get,
        environ_get, environ_sizes_get,
        clock_time_get,
        fd_close, fd_fdstat_get, fd_fdstat_set_flags,
        fd_prestat_get, fd_prestat_dir_name,
        fd_read, fd_write,
        poll_oneoff,
        proc_exit,
        random_get,
        sched_yield,
      },
    },
    setMemory(m) { memory = m },
    WASIExit,
  }
}
