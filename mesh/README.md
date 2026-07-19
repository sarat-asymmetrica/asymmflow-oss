# mesh/ — The Sovereign Mesh track

> "The pure, deterministic, invariant-enforcing function that guards money is the
> same function whether it runs on one node or a thousand."

This directory is AsymmFlow's **Sovereign Mesh** track: the Go kernel compiled to
WASM as the deterministic **Autobase `apply()` reducer** over **Hypercore/Corestore**
storage, replicated across **Holesail** P2P tunnels, with an **Ed25519** capability
layer above the pipe. It is self-contained here per the campaign
(`../FABLE_CAMPAIGN_SOVEREIGN_MESH.md`); it references the sibling `asymm-mesh`
repo's design but does not depend on it.

**Status:** merged to `main` and public since 2026-07-18 (the sovereign-mesh + messenger
track landed with the field-confirmed kitchen-table test). Mission A2 ("The Corridor",
`docs/MISSION_A2_CORRIDOR_SPEC.md`) extends it toward the India↔Bahrain WAN proof.

## What's here (Wave 0 scaffold — boundary proven)

```
mesh/
├── reducer/            # PURE deterministic inventory apply-reducer (Go, host-testable)
│   ├── reducer.go      #   floor invariant qty>=0; rejects the oversell deterministically
│   └── reducer_test.go #   500-permutation convergence + invariant + no-mutation tests
├── cmd/reducer/        # wasip1 packaging (//go:build wasip1) — WASI command module
│   └── main.go         #   stdin JSON ops -> stdout JSON state
├── host/
│   ├── apply.mjs       # JS host: drives reducer.wasm via node:wasi (the two-runtime boundary)
│   └── smoke.mjs       # determinism smoke over the REAL Go->WASM->JS boundary
├── goldens/            # pinned converged states
├── scripts/
│   └── build-reducer.mjs  # cross-platform: GOOS=wasip1 GOARCH=wasm go build
├── docs/
│   ├── MESH_CONFLICT_SHAPE_TAXONOMY.md  # Mission B — CRDT vs Autobase, per entity
│   ├── MESH_PROGRESS.md                 # Mission F — honest status per wave
│   └── MESH_DECISIONS.md                # Mission F — decisions + rationale
├── dist/               # reducer.wasm (git-ignored; `npm run build` regenerates)
└── package.json        # Holepunch/Pear data stack + Holesail
```

## Run it

```bash
# from repo root — pure reducer host tests (fast, no wasm)
go test ./mesh/...

# from mesh/ — build the wasm + run the determinism smoke over the real boundary
cd mesh
npm run smoke            # builds dist/reducer.wasm, then runs host/smoke.mjs
npm run smoke:update-golden   # re-pin the golden after an intentional reducer change
```

Expected: `SMOKE GREEN ✅` — boundary works, oversell rejected, 3 shuffled peers
byte-identical, golden matches.

## The load-bearing rule (read before touching the reducer)

Anything inside the Autobase `apply()` reducer must be a **pure, deterministic**
function — same inputs → byte-identical output on every peer, forever. The four
landmines (map-iteration order, `time.Now()`/`rand`, floats, unstable
linearization) and how the reducer avoids each are documented inline in
`reducer/reducer.go`. A nondeterministic apply cracks the entire trust model. Golden it.

## Status & what's next

Wave 0 (scaffold) proved the **reducer + two-runtime boundary** halves of Mission A.
The remaining Mission A gate — real Autobase linearization over real Hypercores
replicated across real Holesail tunnels between ≥2 machines — is the next on-site
step. See `docs/MESH_PROGRESS.md` for the honest line and the wave plan.

**Stop-and-ask (campaign §5):** financial determinism/rounding/posting or
already-issued ZATCA bytes; the decision to move the *whole* data layer onto
Holepunch; adopting the Pear runtime (Bare) instead of Wails; any grant/revocation
crypto design. These are Commander calls.
