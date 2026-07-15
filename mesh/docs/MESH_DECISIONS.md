# MESH — Decisions (Mission F mirror)

Decisions taken on the Sovereign Mesh track, with rationale, in the campaign's
`[Mirror]` style. Newest first. A decision here is a *default the ground earned* —
the Commander overrides any of them; the stop-and-ask registry (campaign §5) is never
auto-decided.

---

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
