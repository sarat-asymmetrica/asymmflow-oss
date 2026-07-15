# MESH — Progress (honest status per wave)

Campaign: `FABLE_CAMPAIGN_SOVEREIGN_MESH.md`. Branch: `exp/sovereign-mesh`
(worktree `C:\Projects\asymmflow\asymmflow-mesh`). LOCAL-ONLY — never pushed.
Orchestrator = Opus 4.8. "The spike is the gate; the ground wins; the mirror records why."

---

## Wave 0 — SCAFFOLD (2026-07-15) · ✅ done, boundary proven

Goal: stand up the mesh track and **price the two-runtime boundary** (the #1
risk Mission A exists to price) before the Commander is back to run the full gate.

**Built:**
- `mesh/reducer/` — the pure, deterministic inventory apply-reducer (Go, no build
  tags, no I/O/clock/rand). Enforces the floor invariant `qty ≥ 0`; rejects the
  concurrent-offline oversell deterministically. Host unit tests incl. a
  **500-permutation convergence** proof + no-input-mutation + empty-stable.
- `mesh/cmd/reducer/` — the wasip1 packaging (`//go:build wasip1`), a WASI command
  module (stdin JSON → stdout JSON). **Compiles clean to `GOOS=wasip1 GOARCH=wasm`.**
- `mesh/host/apply.mjs` — the JS host driving the wasm via `node:wasi` (real
  file-descriptor stdin/stdout, no pipe deadlock).
- `mesh/host/smoke.mjs` — the determinism smoke over the **real Go→WASM→JS boundary**.
- `mesh/goldens/inventory_basic.json` — pinned converged state.
- `mesh/scripts/build-reducer.mjs` — cross-platform wasm build.
- Holepunch/Pear data stack + Holesail installed (`mesh/package.json`).

**Measured results (Wave-0 smoke, `npm run smoke`):**
- ✅ Boundary: JS drives the Go/WASM reducer; ops marshal in, state marshals out.
- ✅ Invariant: exactly one oversell rejected; no SKU ever < 0.
- ✅ Convergence (reducer half): 3 "peers" fed 3 different arrival orders →
  **byte-identical digest** `aa5fa416…`. Reproducible across separate processes.
- ✅ Golden: converged state matches the pinned golden.

**What Wave 0 deliberately does NOT yet prove (the honest line):**
the 3 peers here are 3 in-process invocations with shuffled input — the
**real Autobase linearization, real Hypercore replication, and real Holesail
transport across ≥2 machines are stubbed.** Wave 0 proves the *reducer + boundary*
halves of Mission A; it does not yet stand up the *replication* half. That is the
first on-site task below.

---

## Wave 1 — MISSION A, the real gate (NEXT · on-site with the Commander)

Turn the stubbed 3-peer smoke into the real thing:
1. Wrap the reducer as the actual **Autobase `apply()`** over a **Corestore**
   (`mesh/host/` → an Autobase view whose apply calls the wasm reducer). Decision
   pending: keep the stdin/stdout command module, or switch to an incremental
   `//go:wasmexport apply()` reactor (lower marshalling overhead, higher wiring
   cost). Wave 0 kept the command module to prove the boundary cheaply first.
2. Replicate 3 Autobase writers **in one process** first (3 Corestores, manual
   `replicate()` streams) → prove byte-identical convergence + the oversell
   rejection through the real linearizer.
3. Then replicate across **≥2 real machines over Holesail** (Mission D transport).
4. **Golden the converged Autobase view** (not just the reducer output).

**Gate (Definition of Done, campaign §6):** 3 peers converge byte-identical; the
oversell invariant holds (one write deterministically rejected on all peers);
goldens pin the state. If it does not go green — STOP and report.

## Wave 2+ — Missions C/D/E/F (after the gate is green)

- **C** — kernel-as-reducer, determinism-audited: compile
  `pkg/kernel/{money,approval,actor,policy}` + invariant checks to wasip1; audit
  every apply path for the two Go landmines (map-iteration order, `time.Now()`/`rand`).
- **D** — Holesail transport + Ed25519 grant-with-epochs capability layer, kept
  strictly separate (transport-auth ≠ capability-auth).
- **E** — per-device ZATCA Hypercore chains (`ICV = core.length`).
- **F** — this mirror + `MESH_DECISIONS.md`, kept honest per wave.

---

## Residue / notes for the next session

- `mesh/dist/reducer.wasm` is git-ignored (build output); `npm run build` regenerates
  it. `npm run smoke` builds then runs.
- The reducer digest is a sha256 over a canonical, map-free projection — safe to use
  as the cross-peer convergence check.
- Determinism landmines are documented inline in `mesh/reducer/reducer.go`; keep that
  discipline when the reducer grows to import the real kernel packages (Mission C).
