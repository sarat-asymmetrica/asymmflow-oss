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

## Wave 1 — MISSION A, the real gate (2026-07-15) · ✅ stages 1+2 GREEN

The stubbed 3-peer smoke is now the real thing. Built (Fable-driven, per the
owner's parallel-tracks call — Opus 4.8 runs frontend INTEG concurrently):

- `mesh/host/mesh-node.mjs` — the mesh peer: Corestore + Autobase whose **view is
  the linearized op log**; `apply()` handles only writer grants + op appends
  (never external state); **state is materialized OUTSIDE apply** by folding the
  whole view through the wasm reducer. Convergence checks: raw **view digest**
  (stronger) + reducer **state digest**.
- `mesh/host/wave1-local.mjs` (`npm run wave1`) — **stage 1**: 3 real Autobase
  writers, 3 on-disk Corestores, real `replicate()` streams, and a GENUINE
  concurrent-offline causal fork (wires cut → dev-a and dev-b write blind →
  wires reconnect → linearizer merges).
- `mesh/host/peer.mjs` — a standalone peer process (host/join roles, JSON-line
  REPL: `add-writer` / `append` / `digest` / `exit`). Transport doctrine held:
  raw Corestore replication over TCP; **Holesail carries the socket** (secure
  `hs://` connector); transport-auth ≠ capability-auth.
- `mesh/host/wave1-holesail.mjs` (`npm run wave1:holesail`) — **stage 2**: two
  separate OS processes replicating through a **real Holesail tunnel over the
  real DHT/UDX stack** (same code path two machines use).

**Measured results (both stages):**
- ✅ Writer grants flow through the linearizer (`addWriter` ops) and replicate.
- ✅ Fork merge: all peers converge to a **byte-identical view**
  (`5962c1f9…`, pinned in `goldens/inventory_autobase.json`; reproducible across
  runs under the test's pinned primary keys).
- ✅ Invariant through the REAL machinery: exactly one oversell rejected, same
  canonical loser (dev-a seq 2) on every peer; no SKU < 0.
- ✅ **State digest == the Wave-0 reducer golden (`aa5fa416…`)** — same ops, same
  reducer, now arriving via real Autobase + real transport. The reducer is
  provably transport-indifferent.
- ✅ Stage 2 over Holesail: grant, appends, convergence, golden — all through the
  tunnel between two processes with independent on-disk stores.

**The honest line (what remains for the full Mission A finale):** stage 2 runs
both processes on ONE machine — the bytes traverse the real DHT/UDX transport,
but not two physical NICs/networks. The ≥2-machine run is the identical
commands on two boxes: machine 1 `npm run wave1:host`, copy the printed `hs://`
url + baseKey, machine 2 `npm run wave1:join -- --url <hs> --base-key <hex>`,
then `add-writer`/`append`/`digest` per `peer.mjs`'s header. Ceremony, not
machinery — the machinery is proven.

**Gate verdict (campaign §6):** peers converge byte-identical ✅ · oversell
deterministically rejected on all peers ✅ · goldens pin the state ✅ — **GREEN**
(with the two-physical-boxes ceremony left for when the Commander has both
machines at hand).

## Wave 2 — MISSION C, kernel-as-reducer (2026-07-16) · ✅ GREEN

The reducer now imports the **REAL kernel packages** —
`pkg/kernel/{money,approval,actor,policy}` — compiled clean to wasip1 (3.7MB
wasm) and proven through the real Autobase machinery:

- **Reducer v2** (`mesh/reducer/reducer.go` + `kernel_domains.go`): typed op
  envelope (`kind` selects the domain; `""` stays Wave-0-compatible), four
  domains folded through kernel law:
  - `inventory.move` — the Wave-0 floor invariant, unchanged.
  - `ar.limit/charge/payment` — **kernel money** integer minor-unit arithmetic;
    credit-limit invariant; currency mismatches are typed kernel errors.
  - `approval.decide` — **kernel approval + actor**: subjects start at
    pending_review; `ValidTransition` is the single truth (approved→rejected
    refused); approve/reject/supersede requires `CanApprove()`;
    needs_input/pending requires `CanPropose()`.
  - `policy.violation/override` — **kernel policy**: only an approver may
    override a standing violation, with a mandatory reason.
- **The AI-authority boundary is DISTRIBUTED LAW:** the agent is stopped at
  THREE kernel layers — `actor.New` (an agent can't even be CONSTRUCTED with
  approve authority), `approval.NewRecord` ("agent actors cannot approve"),
  `policy.Override` (CanApprove gate) — and the Mission C mesh gate proves the
  rejection lands **identically on every peer, even from the agent's own
  writer core**.
- **Determinism audit:** the kernel packages take `now time.Time` as a
  PARAMETER everywhere (no clock reads, no rand, no map-order output —
  audited by grep + the 500-permutation test over the mixed-domain op set).
  Reducer hands them `time.UnixMilli(op.TS).UTC()` — op data, never a clock.
- **Tests:** `missionc_test.go` — kernel-law invariants, agent rejection with
  kernel words, forged-authority agent, propose-level human refused,
  500-permutation convergence, input-immutability. Plus the Wave-0 suite,
  which passes UNCHANGED (envelope is backward compatible).
- **Gates:** `npm run missionc` — 3 peers (one of them the agent's), genuine
  offline fork, byte-identical views + kernel state on all peers, goldened
  (`goldens/missionc_autobase.json`, reproducible).
- **STATE SCHEMA v2:** the state digest now covers stock/ar/approvals/policies —
  Wave-0/1 STATE goldens regenerated (MESH-D9). The Wave-1 VIEW digest was
  untouched (`5962c1f9…`) — the op log didn't change, only the projection.

## Wave 3+ — Missions D/E/F (next)

- **D** — Ed25519 grant-with-epochs capability layer above the Holesail pipe
  (transport-auth ≠ capability-auth; transport half already proven in Wave 1).
- **E** — per-device ZATCA Hypercore chains (`ICV = core.length`).
- **F** — this mirror + `MESH_DECISIONS.md`, kept honest per wave.
- The `//go:wasmexport` incremental reactor (per MESH-D6) when marshalling
  volume warrants it; candidate alongside Mission D.

---

## Residue / notes for the next session

- `mesh/dist/reducer.wasm` is git-ignored (build output); `npm run build` regenerates
  it. `npm run smoke` builds then runs.
- The reducer digest is a sha256 over a canonical, map-free projection — safe to use
  as the cross-peer convergence check.
- Determinism landmines are documented inline in `mesh/reducer/reducer.go`; keep that
  discipline when the reducer grows to import the real kernel packages (Mission C).
