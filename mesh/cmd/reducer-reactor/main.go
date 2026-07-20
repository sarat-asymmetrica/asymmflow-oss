//go:build wasip1

// Command reducer-reactor is the go:wasmexport REACTOR packaging of the pure
// inventory reducer — Bare-runtime campaign Phase 1b (mesh/docs/bare-campaign).
// Sibling of ../reducer/main.go (the WASI *command* module), NOT a replacement
// for it: owner ruling R1 requires the command module keep building and stay
// green as the rollback path (mesh/docs/bare-campaign/OWNER_RULINGS.md).
//
// WHY A SECOND ENTRY POINT — and an honest correction to an earlier estimate.
// Phase 0 first measured a go:wasmexport reactor at only 10 wasi_snapshot_preview1
// imports versus the command module's 16, which suggested the reactor would shed
// the whole fd-table / preopen group (the six calls P0-A confirmed have no bare-fs
// backing). That measurement was taken against a TRIVIAL `add()` probe and did not
// survive contact with the real reducer. Measured against THIS file's actual build:
//
//     command module (../reducer):  16 distinct wasi_snapshot_preview1 imports
//     reactor        (this file):   15  — only fd_read is eliminated
//
// Cause, traced rather than guessed (`go list -deps`): mesh/reducer imports no
// os/fmt/log itself, but encoding/json, crypto/ed25519, crypto/sha256 and
// encoding/hex all transitively pull in Go's `os` package on wasip1, and os's
// package-level init unconditionally builds the stdio fd table — that survives
// linking whether the entry point is `func main` or a go:wasmexport reactor. The
// trivial probe measured 10 only because it imported nothing that pulls in `os`,
// so that init was dead-code-eliminated. Any WASI shim hosting this module must
// therefore budget for 15 syscalls, INCLUDING all five no-bare-fs-backing ones.
// (Full trail: docs/bare-campaign/PHASE1B_REPORT.md and PHASE0_GATE_C_VERIFICATION.md.)
//
// So the reactor's justification is NOT a smaller shim — that turned out to be
// worth exactly one syscall. It is the call pattern: instantiate once, call
// _initialize once, then call a WARM instance's exports repeatedly, which suits a
// host folding incrementally rather than replaying the whole op-log per call, and
// which retires the command channel's temp-file-per-call design along with it.
//
// ABI (documented here because header comments in this repo are load-bearing,
// same discipline ../reducer/main.go already follows):
//
//   The Go compiler's go:wasmexport lowering (verified against
//   pkg.go.dev/cmd/compile#hdr-WebAssembly_Directives and go.dev/blog/wasmexport —
//   see docs/bare-campaign/PHASE0_NOTES_C_WASMEXPORT.md §3 for the full citation
//   trail) permits `string` as an EXPORTED FUNCTION PARAMETER — the compiler
//   lowers it to an (i32 ptr, i32 len) pair automatically — but NOT as a return
//   type. That asymmetry is what forces the three-export shape below; it is not
//   a stylistic choice:
//
//     malloc(size uint32) uint32
//       The host calls this FIRST to obtain a guest-memory offset it may write
//       `size` bytes of input JSON into. The returned buffer is pinned in a
//       process-global registry (`pinned` below) so Go's own garbage collector
//       cannot reclaim it between this call and the matching free() — from
//       malloc onward, the HOST owns that allocation's lifetime, not Go's GC.
//       SENTINEL: malloc(0) returns 0 and pins nothing — 0 is never a real
//       buffer's address (bufAddr never returns 0 for a non-empty buffer), so
//       it is a safe, unambiguous "no allocation" marker. free(0, _) and
//       readBytes(0, 0) both treat it as a deliberate no-op, not an error.
//
//     apply(ptr, length uint32) uint64
//       Reads `length` bytes starting at `ptr` (bytes the host wrote after its
//       own malloc call), unmarshals them into the SAME `input{Mode,Config,Ops}`
//       struct ../reducer/main.go uses, and dispatches through the SAME
//       reducer.ApplyWithConfig / reducer.ApplyRoom functions the command module
//       calls — see the "semantics, not packaging" note below. The result is
//       marshaled to JSON, copied into a FRESH malloc'd buffer (a second,
//       independent allocation from the input one — apply never reuses or frees
//       the input buffer), and returned as a packed 64-bit "fat pointer":
//       `(outPtr << 32) | outLen`. string cannot be a go:wasmexport RESULT type
//       (see the ABI note above), so this packed-uint64 encoding is the
//       documented workaround, not a local invention — it mirrors the pattern
//       in the community write-ups cited in the Phase 0 notes.
//
//     free(ptr, size uint32)
//       Releases a pin from the registry so the buffer becomes GC-eligible
//       again. The HOST is responsible for calling this on every pointer it
//       obtained — from its own malloc call, AND from the outPtr half of every
//       apply() return value — exactly once, after it has finished reading the
//       bytes. `size` is accepted but unused (the registry is keyed by address
//       alone); it exists so the export signature reads symmetrically with
//       malloc's, and so a future pinning scheme keyed by (ptr,size) is a
//       non-breaking change. ptr==0 (the malloc(0) sentinel) is a no-op.
//
//   readBytes(ptr, length) — the input side of the same discipline, NOT
//   itself exported — never reconstructs a Go pointer from a raw wasm
//   address (that would be a uintptr->unsafe.Pointer conversion outside
//   Go's unsafe.Pointer rules, flagged by `go vet`'s unsafeptr check, and
//   unsound in general even where it happens to work today). Instead it
//   looks `ptr` up in the SAME `pinned` registry malloc populated — the
//   host can only ever have a valid `ptr` because malloc handed it one —
//   and returns a re-sliced VIEW of that exact buffer, bounds-checked
//   against `length` so a host that lies about the length can't read past
//   its own allocation. A `ptr` malloc never issued (or a `length` bigger
//   than what was allocated at `ptr`) resolves to nil, not a crash or an
//   out-of-bounds read — apply() turns that into an error result, never UB.
//
//   `_initialize` (emitted automatically by `-buildmode=c-shared` — see the
//   Phase 0 notes' §2 for the citation) must be called by the host exactly once,
//   before any of the three exports above. `main` below is never invoked by the
//   host; Go requires a `func main` to exist in `package main` regardless.
//
// SEMANTICS, NOT PACKAGING (owner ruling R1's load-bearing distinction): this
// file imports mesh/reducer UNCHANGED — not one file under mesh/reducer/** is
// touched by this campaign phase — and calls the identical ApplyWithConfig /
// ApplyRoom functions the command module calls, with the identical
// input{Mode,Config,Ops} struct, in the identical dispatch shape (compare this
// file's `apply`/`emit` against ../reducer/main.go's `main`: same two-way mode
// switch, same two function calls, same json.Marshal of the same `any`-typed
// result). Nothing here re-expresses the fold; only the byte channel differs —
// malloc/apply/free replacing stdin/stdout's read-all/write-all. The proof that
// this claim is true, not merely argued, is the byte-identity comparison against
// the command module's own output over the SAME (Config, Ops) — see
// docs/bare-campaign/PHASE1B_REPORT.md and mesh/host/reactor-parity-spike.mjs.
//
// Build: GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o mesh/dist/reducer-reactor.wasm ./mesh/cmd/reducer-reactor
package main

import (
	"encoding/json"
	"sync"
	"unsafe"

	"ph_holdings_app/mesh/reducer"
)

type input struct {
	Mode   string         `json:"mode,omitempty"` // "" = business fold; "room" = Messenger room fold
	Config reducer.Config `json:"config"`         // Mission D: authorityPub enables capability enforcement
	Ops    []reducer.Op   `json:"ops"`
}

// pinned holds every live allocation keyed by its starting linear-memory
// address, so Go's GC never reclaims a buffer the host still holds a raw
// pointer to. This map's only job is to keep a Go-visible reference alive
// between malloc (or apply's own output allocation) and the matching free —
// it does not itself decide WHEN memory is released; the host does, via free.
var (
	pinMu  sync.Mutex
	pinned = map[uint32][]byte{}
)

// bufAddr returns a buffer's own starting address as a wasm32 i32-sized
// pointer. Safe under Go's non-moving wasm GC (verified: the runtime does not
// relocate live heap objects on this target), but only for the DURATION the
// buffer stays pinned — which is exactly what the `pinned` map guarantees.
//
// Callers must never hand this a zero-length buffer: &buf[0] does not exist for
// one, and taking &buf instead would yield the address of the SLICE HEADER (a
// local variable) — an address that points at no payload, that can collide with
// a real allocation or with another zero-size request, and that would then be
// used as a live key in `pinned`. malloc handles size 0 before reaching here.
func bufAddr(buf []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(&buf[0])))
}

// NULL is the ABI's zero-size / failure pointer. It is never a valid key in
// `pinned`, so free(NULL) and readBytes(NULL, _) are both safely inert.
const NULL uint32 = 0

//go:wasmexport malloc
func malloc(size uint32) uint32 {
	// A zero-size allocation has no address worth naming; the ABI defines NULL
	// as its answer (see the file header). Returning a synthesized address here
	// would register a bogus key in `pinned` that free could never match.
	if size == 0 {
		return NULL
	}
	buf := make([]byte, size)
	ptr := bufAddr(buf)
	pinMu.Lock()
	pinned[ptr] = buf
	pinMu.Unlock()
	return ptr
}

//go:wasmexport free
func free(ptr uint32, _ uint32) {
	if ptr == NULL {
		return
	}
	pinMu.Lock()
	delete(pinned, ptr)
	pinMu.Unlock()
}

// readBytes returns the first `length` bytes of the buffer registered at `ptr`,
// WITHOUT copying. The host can only legitimately pass a `ptr` that came from
// this module's own malloc, so the authoritative []byte is already in `pinned`
// — we look it up rather than reconstructing a pointer from an integer.
//
// This is deliberate, and it is the safe formulation: turning a uintptr back
// into an unsafe.Pointer (the obvious way to write this) is invalid under Go's
// unsafe.Pointer rules even where it happens to work on wasm today, and `go vet`
// flags it as `unsafeptr`. The lookup needs no unsafe at all, AND it bounds-checks
// the host's `length` claim against an allocation whose true size we know — a
// raw reinterpretation would happily slice past the end of the buffer if the
// host lied. An unknown or over-long pointer yields nil, which apply() reports
// as a normal JSON error rather than trapping the instance.
func readBytes(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}
	pinMu.Lock()
	buf, ok := pinned[ptr]
	pinMu.Unlock()
	if !ok || uint32(len(buf)) < length {
		return nil
	}
	return buf[:length]
}

// emit marshals v, mallocs a fresh output buffer sized to fit, copies the
// JSON in, and packs the result into apply's single allowed uint64 return
// value (string cannot be a go:wasmexport result type — see the file header).
func emit(v any) uint64 {
	out, err := json.Marshal(v)
	if err != nil {
		// Marshal failure on our own state/error types would be a reducer bug,
		// not a caller error; still never let apply() itself panic across the
		// wasm boundary — degrade to a minimal, always-marshalable error shape.
		out, _ = json.Marshal(map[string]string{"error": "reducer-reactor: marshal state: " + err.Error()})
	}
	// json.Marshal never actually returns an empty slice for these types (the
	// worst case is the 4-byte literal "null"), so this is defense against a
	// future v that could — not a path exercised today. Written this way so
	// the NULL discipline is total: emit() never mallocs 0 bytes and never
	// looks a NULL ptr up in `pinned` (which would resolve to a nil buf and a
	// no-op copy anyway, but going through the sentinel explicitly keeps this
	// function's contract with malloc/free/readBytes uniform everywhere).
	if len(out) == 0 {
		return 0
	}
	ptr := malloc(uint32(len(out)))
	pinMu.Lock()
	buf := pinned[ptr]
	pinMu.Unlock()
	copy(buf, out)
	return (uint64(ptr) << 32) | uint64(len(out))
}

// apply is the reactor's one real export: read the input JSON the host wrote
// via malloc, fold it through the SAME reducer functions the command module
// calls (see the file header's "semantics, not packaging" note), and hand
// back a fat pointer to the output JSON. Every error path still returns a
// normal (non-trapping) fat pointer — to a small {"error": "..."} JSON object
// — rather than aborting the wasm instance, since a reactor is meant to
// survive a bad call and serve the next one (unlike the command module, whose
// os.Exit(N) on error is fine because the whole instance is one-shot anyway).
//
//go:wasmexport apply
func apply(ptr, length uint32) uint64 {
	body := readBytes(ptr, length)

	var in input
	if len(body) > 0 {
		if err := json.Unmarshal(body, &in); err != nil {
			return emit(map[string]string{"error": "reducer-reactor: parse ops: " + err.Error()})
		}
	}

	// One wasm, two folds (Messenger Wave 1): the mode selects which law runs.
	// Identical switch to ../reducer/main.go's — this is the semantics-vs-
	// packaging claim made mechanically checkable, not just asserted in prose.
	var state any
	switch in.Mode {
	case "":
		state = reducer.ApplyWithConfig(in.Config, in.Ops)
	case "room":
		state = reducer.ApplyRoom(in.Config, in.Ops)
	default:
		return emit(map[string]string{"error": "reducer-reactor: unknown mode " + in.Mode})
	}

	return emit(state)
}

// main is never invoked by the host (the reactor's entry point is
// _initialize, generated by -buildmode=c-shared) but `package main` requires
// one to exist.
func main() {}
