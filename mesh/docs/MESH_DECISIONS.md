# MESH — Decisions (Mission F mirror)

Decisions taken on the Sovereign Mesh track, with rationale, in the campaign's
`[Mirror]` style. Newest first. A decision here is a *default the ground earned* —
the Commander overrides any of them; the stop-and-ask registry (campaign §5) is never
auto-decided.

---

### MESH-D8 — Wave 1 transport = raw Corestore replication over TCP, Holesail carries the socket
`peer.mjs` speaks the Hypercore replication protocol over a plain TCP socket bound to
127.0.0.1; Holesail (secure connector) tunnels that socket over the DHT.
**[Mirror]** Mission D doctrine demands transport-auth ≠ capability-auth. Piping the
replication stream through a Holesail-tunneled TCP socket keeps the layers physically
separate: knowing the `hs://` connector gets you a byte pipe; ONLY an `addWriter` grant
through the linearizer gets you write access. Hyperswarm-direct (join the discoveryKey)
would work too, but Holesail is the ratified Era-3 sidecar and this proves the actual
target topology. Confirmed working over the real DHT/UDX stack between processes.

### MESH-D7 — Autobase view = the linearized OP LOG; state materialized OUTSIDE apply()
`apply()` only handles writer grants and appends op values to the view core. The
inventory state is computed by folding the whole view through the wasm reducer on read.
**[Mirror]** Autobase may undo/reapply the view during reordering, so apply must touch
nothing external (README's IMPORTANT note). Keeping apply minimal makes it trivially
deterministic; the reducer re-sorts canonically anyway, so state is stable through
interim linearization churn while the CONVERGED view still gates byte-identity (the
stronger property). Cost: O(n) refold per read — fine at spike scale; the incremental
`//go:wasmexport` reactor (MESH-D4's endpoint) is the optimization, deferred to
Mission C when the real kernel packages arrive.

### MESH-D6 — Wave 1 keeps the WASI command module (reactor still deferred)
**[Mirror]** MESH-D4's pending decision, resolved for Wave 1: the command module
survived contact with the real machinery (refold-per-read over Autobase views) without
wiring cost, so the reactor's extra complexity still isn't paid for. Re-evaluate at
Mission C where per-op marshalling volume actually grows.

### MESH-D5 — Location: a dedicated `exp/sovereign-mesh` worktree off `main`, not the frontend-flip branch
The mesh work lives in its own worktree/branch off `main` (`asymmflow-mesh`), under a
top-level `mesh/` dir.
**[Mirror]** The campaign says keep the mesh work self-contained *in the AsymmFlow
repo* (not in sibling `asymm-mesh`). The active `exp/frontend-kernel` worktree is the
K6 flip branch — mixing a Node data-stack + Go/WASM spike into it would pollute the
flip's clean diff and couple two unrelated graduations. A dedicated branch off main is
the clean, reversible home: self-contained in-repo as asked, isolated from the flip.

### MESH-D4 — Wave 0 proves the boundary with a WASI *command* module, not a `//go:wasmexport` reactor (yet)
The reducer is packaged as a stdin→stdout WASI command module for the scaffold.
**[Mirror]** Mission A's stated purpose is to *price the two-runtime boundary* before
committing. The cheapest way to prove JS↔Go/WASM marshalling is clean is a command
module driven over real file-descriptor stdio — zero manual linear-memory
marshalling, no pipe-deadlock risk, trivially goldenable. The incremental
`//go:wasmexport apply()` reactor (lower per-op overhead, wires straight into
Autobase's `apply()`) is the *right* endpoint but carries higher wiring cost; it is
Wave 1's decision, made once the boundary itself is proven. Go 1.25 supports
`//go:wasmexport`, so the door is open — we just didn't pay for it before pricing the risk.

### MESH-D3 — The spike domain is INVENTORY (floor invariant), per campaign §3
`qty ≥ 0` per SKU; a concurrent-offline oversell is deterministically rejected.
**[Mirror]** The campaign named inventory *or* orders. Inventory was chosen because it
is the *invariant-bound* case (Autobase-apply), which is the half that actually
exercises the reducer's reason to exist — rejecting a convergent-but-wrong merge.
Orders (pure append/G-Set) would prove convergence but not invariant enforcement; the
harder, more load-bearing property is the one worth spiking first.

### MESH-D2 — Canonical linearization order = (Seq, Actor, SKU, TS); TS is event data, never a clock
The reducer re-sorts ops by this total order before folding.
**[Mirror]** Autobase supplies a deterministic linearized order; the reducer models it
with an explicit total order so replay is independent of network arrival order (proven:
500 permutations → one digest). TS is included only as the *last* tie-breaker and is
carried in the op DATA — never read from a wall-clock inside apply (landmine #2). A
consequence worth internalizing: *which* conflicting write loses an oversell is itself
deterministic (the later-in-canonical-order one), identical on every peer — not the
"latest by clock". The invariant guarantees the STATE and the reject COUNT; the
canonical order fixes the specific loser.

### MESH-D1 — Convergence check = sha256 over a canonical, map-free state projection
Peers agree iff their digests match.
**[Mirror]** A digest makes the "3 peers byte-identical" gate a one-line equality
instead of a deep-compare. It is computed over a *sorted, map-free* projection (SKUs in
sorted key order) specifically to dodge landmine #1 (randomized map iteration) — the
digest must not itself be a source of nondeterminism. Integer-only quantities (landmine
#3) keep the encoded bytes stable.

---

## Stop-and-ask items NOT decided here (campaign §5 — Commander calls)
- Moving the *whole* AsymmFlow data layer onto Holepunch (Mission A's evidence informs
  it; the full cutover is gated).
- Adopting the Pear runtime (Bare) vs keeping Wails — this track assumes **Wails stays;
  Holepunch supplies data + transport only.**
- Any grant/revocation crypto (Ed25519 epoch rotation) before it ships.
- Anything touching financial determinism, rounding, posting, or already-issued ZATCA
  invoice bytes.
