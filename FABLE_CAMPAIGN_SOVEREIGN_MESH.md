# FABLE CAMPAIGN — SOVEREIGN MESH: "The Kernel Becomes Distributed Law"

Written 2026-07-04 by the Opus 4.8 instance running the Commander's
(Sarat's) strategy session, for the Fable 5 instance that will run this
campaign alongside the existing wave structure. This is the **Option C**
bet named in that session: adopt the Holepunch/Pear **data stack**
(Hypercore → Hyperbee → Autobase) for storage + multi-writer sync, with
**the AsymmFlow Go kernel compiled to WASM as the deterministic Autobase
`apply()` reducer**, Holesail/Hyperswarm as transport, and an Ed25519
capability layer above the pipe.

Commander's call (2026-07-04): **go straight to C** — do not stop at the
Holesail-sidecar-only half-measure. But run C *wave-disciplined*: the
cheapest falsification (a determinism spike on one domain) comes FIRST,
and the full-data-layer commitment is earned by that spike's evidence,
not assumed. This is C executed with the same "measure, don't estimate"
spine as Waves 2–6 — not a leap of faith.

The Commander is available. Ask when a decision is his; do not ask when
this document already answers it.

---

## 0. The thesis this campaign proves (and how it completes the others)

AsymmFlow-OSS proved: *a business app is a pure kernel + overlays.* The
PH Convergence campaign proves that at real scale. **This campaign proves
the last unification:**

> **The pure, deterministic, invariant-enforcing function that guards
> money is the same function whether it runs on one node or a thousand.**

Autobase linearizes every writer's append-only log into ONE deterministic
order every peer agrees on, then replays it through a pure `apply()`
reducer. If that reducer IS the AsymmFlow kernel (`pkg/kernel/{money,
approval,actor,policy}`), compiled to WASM, then integer-money precision,
the approval state machine, and the AI-authority boundary (`CanApprove()`
→ false for agents) stop being single-node checks and become **distributed
consensus rules enforced identically on every peer, with cryptographic
proof of the log.** Kernel = apply reducer. Overlay = WASM component.
Event bus = Autobase log. Sync = Hypercore replication. Transport =
Holesail. Identity = Ed25519 seed. One organism.

The sister repo `C:\Projects\asymm-mesh` already scaffolds these primitives
(Node, Component, Store, Tunnel, Key, Grant, Event) and the 9 sovereign
invariants. This campaign brings that substrate INTO AsymmFlow-OSS as its
own track — reference asymm-mesh's design, but keep the AsymmFlow work
self-contained here.

## 1. Why this is worth the R&D (the payoff)

- **Sovereign multi-branch, zero cloud.** A GCC trader's two/five offices
  sync peer-to-peer with no server, no static IP, no port-forwarding, no
  monthly hosting bill. Data never touches infrastructure we run.
- **Financial invariants become unanimous.** No node can accept an
  oversell or a negative invoice — the apply reducer rejects it on every
  peer, deterministically.
- **The AI firewall goes distributed.** `CanApprove()==false` for agents
  is enforced in the replication layer itself.
- **It de-risks on a proven, funded stack.** Holepunch is Tether-funded;
  Keet (P2P video/chat), PearPass (audited by Secfault), and QVAC
  (Tether's sovereign local-AI) ship on it in production today. This is
  tested infrastructure, not a research toy.

## 2. The load-bearing distinction you must internalize FIRST

**CRDTs and Autobase are different tools; the mesh needs BOTH, sorted by
conflict-shape.**

- **CRDT** = always-accept, converges, but CANNOT preserve a business
  invariant (two offline nodes both sell the last unit → PN-Counter
  merges to −1, oversold; convergent but wrong). Use for the
  **commutative ~80%**: orders (append-only), audit log, customer visits,
  notes, per-field profile/menu edits.
- **Autobase** = deterministic linearized order + pure `apply()` reducer
  = CAN enforce invariants, at the cost that a conflicting **offline write
  may be REJECTED on merge** (the reducer skips it). Use for the
  **invariant-bound ~20%**: inventory floors, credit limits, uniqueness,
  ordering. For money, "reject the oversell" is the CORRECT behavior — the
  UX surfaces a typed `Unconfirmed`/`Rejected` state (Mechanism 2), never
  a silent bad number.

Do NOT try to CRDT the invariant-bound set, and do NOT pay Autobase's
rejection cost on the commutative set. Mis-sorting either way is the
signature failure mode of this campaign.

**Delightful alignment to exploit (Mission E):** ZATCA does NOT require a
global invoice sequence — the ICV counter is **per device / EGS unit**.
A per-device append-only counter is exactly a **Hypercore** (one keypair,
one log, single-writer, `ICV = core.length`). No cross-node coordination
is needed for numbering because the mandate never asked for it. Two
independently-designed systems share one shape; use it.

## 3. Missions (risk-retirement order — cheapest falsification first)

**Mission A — The determinism spike (THE gate; do this before anything
else).** Pick ONE domain — **inventory** (PN-Counter + an Autobase floor
invariant) or **orders** (G-Set, pure append). Build the full loop end to
end for JUST that domain: a Go invariant-reducer compiled to `wasip1`, run
inside an Autobase `apply()`, over a Corestore, replicated across **3
simulated peers**. Prove: (1) all 3 peers converge byte-identical; (2) the
invariant holds under a concurrent-offline-oversell scenario (one write is
deterministically rejected on all 3); (3) golden tests pin the converged
state exactly. If this spike does not go green, STOP and report — the
whole bet rests on it. This is a spike, not a migration; touch nothing
else.

**Mission B — The conflict-shape taxonomy.** Classify every AsymmFlow
entity (invoice, invoice-item, PO, GRN, DN, serial, payment, credit-note,
customer, supplier, product, opportunity, costing, inventory, audit) into
**CRDT-safe** vs **Autobase-invariant-bound**, with the specific CRDT type
(G-Set / PN-Counter / LWW-Map / RGA) or the specific invariant the reducer
must enforce. Deliverable: `docs/MESH_CONFLICT_SHAPE_TAXONOMY.md` — a
table. This is the design that makes Mission C tractable.

**Mission C — Kernel-as-reducer, determinism-audited.** Compile
`pkg/kernel/{money,approval,actor,policy}` + the invariant checks to
`wasip1` and wire them as the Autobase apply reducer. Audit EVERY apply
path for nondeterminism — the two Go landmines: **map iteration order is
randomized (sort keys before iterating)** and **`time.Now()`/`rand` are
forbidden inside apply (timestamps come from event DATA, not wall-clock)**.
Integer money already removes float nondeterminism (keep it that way). The
existing golden-test discipline is exactly the tool — pin byte-identical
apply outputs.

**Mission D — Transport + capability layer (keep them SEPARATE).** Wire
Holesail (Node sidecar, per asymm-mesh `pkg/tunnel`) as a **dumb encrypted
pipe** — TCP *and* UDP. **CRITICAL invariant:** a Holesail connection key
CANNOT be revoked or rotated (it is a static SSH-like seed — confirmed in
the Holesail docs). Therefore NEVER use a raw Holesail key as the
permission model. The real capability/grant/identity layer is Ed25519
**Grants WITH epochs** (asymm-mesh `pkg/identity` + `Grant` primitive)
sitting ABOVE the pipe: revocation = bump the grant epoch and re-issue to
the still-trusted; old grants stop validating at the app layer even though
the old pipe still opens. Separate transport-auth from capability-auth,
always. (This is how an ephemeral ZATCA-auditor tunnel dies cleanly: kill
the Holesail process AND expire the grant epoch.)

**Mission E — Per-device ZATCA chains.** Model each device's invoice chain
as its own Hypercore; `ICV = core.length`; wire the Saudi compliance
engine (`pkg/compliance/saudi`) to read the counter from the core, not a
shared table. Financial-semantics + already-issued bytes = stop-and-ask.

**Mission F — The mirror.** `docs/MESH_DECISIONS.md` (MESH-D1…, `[Mirror]`
paragraphs) written WHEN you decide; `docs/MESH_PROGRESS.md` per wave with
honest status, the spike's measured convergence results, and residue.

## 4. Invariants (inherit Wave-6 §5; these are added and non-negotiable)

1. **Determinism is sacred.** Anything inside an Autobase `apply()` must be
   a pure, deterministic function — same inputs → byte-identical output on
   every peer, forever. No wall-clock, no `rand`, no unsorted map
   iteration, no floats. A nondeterministic apply cracks the entire trust
   model. Golden it.
2. **No cloud dependency.** If it needs a server we run to function, it
   violates the thesis. (Hyperswarm DHT bootstrap is a network dependency,
   NOT a server holding data — say "zero infrastructure you pay a monthly
   bill for," and offer self-hosted bootstrap for the paranoid.)
3. **Holesail is a dumb pipe.** Transport-auth (the pipe) and
   capability-auth (Ed25519 grants with epochs) are separate layers. Never
   conflate them.
4. **Sort by conflict-shape.** CRDT the commutative; Autobase-apply the
   invariant-bound. Never mix.
5. **CGO stays banned.** It is *why* the kernel can target `wasip1` at all.
6. **No real client data in the repo.** Synthetic canon only.

## 5. Stop-and-ask registry

- Any change to financial determinism, rounding, posting, or the bytes of
  an already-issued ZATCA invoice.
- The decision to move the WHOLE AsymmFlow data layer onto Holepunch
  (Mission A's spike informs this; the full cutover is Commander-gated).
- Adopting the Pear *shell*/*runtime* (Bare) vs keeping Wails — this
  campaign assumes **Wails stays; Holepunch supplies data + transport
  only.** Changing that is a Commander call.
- Any grant/revocation crypto design (epoch rotation) before it ships —
  revocation in P2P is the deepest crypto problem here.

## 6. Definition of done (for the first campaign wave)

- Mission A spike GREEN: 3 peers converge byte-identical; the oversell
  invariant holds (one write deterministically rejected on all peers);
  goldens pin the state.
- `docs/MESH_CONFLICT_SHAPE_TAXONOMY.md` complete.
- One domain (the spike's) runs live over Holesail transport across ≥2
  real machines with the Go-WASM reducer enforcing its invariant.
- `docs/MESH_DECISIONS.md` + `docs/MESH_PROGRESS.md` written, honest
  thesis %, residue for the next wave.
- The Commander has the evidence to decide whether the full data layer
  migrates.

## 7. Reference docs (Fable: read these — links)

**Pear / Holepunch stack**
- Pear stack overview: https://docs.pears.com/explanation/the-pears-stack/
- Runtime & languages (Bare, not Node): https://docs.pears.com/explanation/runtime-and-languages/
- Storage & distribution: https://docs.pears.com/explanation/storage-and-distribution/
- Dependencies & network: https://docs.pears.com/explanation/dependencies-and-network/
- **Autobase (the multi-writer apply reducer — THE piece):** https://docs.pears.com/reference/building-blocks/autobase
- Hypercore replicate & persist: https://docs.pears.com/how-to/store-and-replicate/replicate-and-persist-with-hypercore/
- Hyperswarm (peer discovery by topic): https://docs.pears.com/how-to/connect-to-peers/connect-to-many-peers-by-topic-with-hyperswarm/
- Corestore helper: https://docs.pears.com/reference/helpers/corestore/
- Release pipeline / OTA: https://docs.pears.com/explanation/release-pipeline/

**Holesail (transport)**
- Docs: https://docs.holesail.io/ · LLM digest: https://docs.holesail.io/llms-full.txt
- Start a server: https://docs.holesail.io/usage/start-a-holesail-server
- Connection keys (note: static, non-revocable): https://docs.holesail.io/terminology/connection-keys

**Production proof (how others applied the stack)**
- Keet (P2P messaging, 24-word seed identity): https://keet.io/
- PearPass (P2P password manager, Libsodium, audited): https://pass.pears.com/
- QVAC (Tether's sovereign local AI): https://qvac.tether.io/

**In-repo / sibling**
- `C:\Projects\asymm-mesh` — CLAUDE.md, docs/ARCHITECTURE.md, docs/SYNC.md
  (the CRDT schemas), docs/MANIFESTO.md (the 9 invariants). Reference it;
  keep AsymmFlow's mesh work self-contained here.

## 8. Honest risks (name them, don't hide them)

- **Two-runtime complexity** (Bare/JS ↔ Go/WASM). Marshalling + debugging
  across the boundary is real cost. The spike (Mission A) exists to price
  it before committing.
- **The determinism tax.** Every apply-path line gets audited. Non-trivial,
  but the golden discipline already trains for it.
- **Ecosystem maturity.** Smaller than the Postgres world — mitigated by
  Tether funding + production mileage (Keet/PearPass/QVAC) + you own the
  file on disk (Corestore), so there is no vendor to be captured by.
- **Always-on topology.** Design branches as Autobase peers (any online
  subset progresses), never hub-and-spoke (master off = all dark).

Build → Test → Ship. Measure, don't estimate. The spike is the gate; the
ground wins; the mirror records why. 🌊
