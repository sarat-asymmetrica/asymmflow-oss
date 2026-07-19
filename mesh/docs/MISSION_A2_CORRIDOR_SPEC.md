# Mission A2 — "The Corridor" (India ↔ Bahrain WAN Proof)

**Status:** RATIFIED 2026-07-19 · **Branch:** `feat/fable-mission-a2-corridor` · **Base:** `2f874c3` (main)
**Spec author / gate:** Fable (orchestrator + technical lead + primary gate)
**Builders:** Sonnet 5 coder agents, one per band
**Field window:** SPOC available same-day for the receptionist-machine test

---

## 0. Mission statement

The kitchen-table field kit proved the mesh on real hardware over LAN (MSG-D26, 2026-07-18).
Mission A2 proves the same ceremony over the open internet: **founder machine (India) ↔ receptionist
machine (PH office, Bahrain)** — live bidirectional messaging + sha256-verified file transfer over
Hyperswarm, no Postgres, no VPN, no port-forward *required* (though one is available).

This is the gate that Mission B (Postgres → Autobase data migration) sits behind. Prove the pipe,
then move the water.

## 1. Owner rulings recorded (2026-07-19)

| # | Ruling |
|---|--------|
| R1 | **Anchor over VPS.** The receptionist machine serves as the always-on **anchor peer** (full member, direct reachability, high uptime) in lieu of a VPS. VPS is demoted to a documented contingency, triggered only by a CGNAT verdict or future multi-site/uptime needs. |
| R2 | **Era-1 precedent confirmed.** The DuckDNS-exposed Postgres of Era-1 runs on the PH office network → that network already has a real public IP + working port-forward + dynamic DNS. The reachability question is pre-answered; Band 1 verifies rather than discovers. |
| R3 | **Full homelab authority.** PH office router/firewall may be modified freely (not an enterprise-managed setting). |
| R4 | **Bare does not gate the corridor.** The Bare sidecar port (DP4 opener) is Band 4, a stretch mission. Bands 1–3 must not depend on it. |
| R5 | Terminology: the anchor is **not** a blind peer — it is a full member holding room keys. `blind-peer` remains reserved for future ciphertext-only mirrors on untrusted infra. |

## 2. Non-negotiable invariants (binding on every coder)

- **I1 — No mesh-law changes.** The Go/WASM reducer, capability layer, invite semantics, and
  bridge protocol v0 are FROZEN in this mission. This is a transport/packaging/ops mission.
  Any change under `mesh/reducer/` or to `bridge-server.mjs` method semantics = stop and report.
- **I2 — Existing gates stay green.** `kit-spike.mjs` (33 checks) and `bridge-spike.mjs`
  (30 checks) must pass unmodified. New capability = new checks, never edited old ones
  (GL-11 discipline).
- **I3 — Phone-scriptable surfaces.** Anything the receptionist sees must be readable aloud
  over a phone: short words, no hex dumps on the happy path, `/status` stays the support tool.
- **I4 — Synthetic identities only** in fixtures, docs, and examples (GL-12: names come from
  SYNTHETIC_IDENTITY.md, never invented-plausible). No real names anywhere.
- **I5 — Pinned runtime.** Node **v22 LTS** (v22.17.0 is the proven local version). The kit
  bundles its own runtime; UDX native prebuilds must load from the bundled layout.
- **I6 — Zero new heavyweight deps.** The mesh package already carries everything needed
  (hyperswarm, hyperdht, corestore, holesail, blind-peer). New npm deps require gate approval.
- **I7 — Wire is E2E-encrypted on every path** (existing property; do not regress). The probe
  and anchor must never log message plaintext.

## 3. Bands

### Band 1 — The Probe 🔬 (`mesh/kit/probe.mjs`)

A single-file diagnostic the receptionist (guided by SPOC on the phone) runs FIRST. Answers
"can this network mesh?" in under a minute, before any ceremony is attempted.

Checks, in order, each with a plain-English PASS/FAIL line:

1. **DHT bootstrap reachability** — connect a `hyperdht` node against the default bootstrap
   set (`node1-3.hyperdht.org:49737`, outbound UDP). Report reachable count.
2. **NAT self-diagnosis** — after bootstrap, report what HyperDHT knows: are we firewalled,
   what host:port does the network see us as. Print the observed public IP prominently.
3. **CGNAT check (human-in-the-loop)** — print the observed public IP with the instruction:
   *"Compare with the router's WAN address (SPOC does this). Same = GREEN, different = CGNAT."*
   The probe cannot see the router; the card makes the comparison a 30-second phone step.
4. **Punch test** — `--listen` mode (founder machine) prints a z32 key; `--dial <key>` mode
   (Bahrain) connects over Hyperswarm, round-trips a ping payload, and reports direct vs
   relayed if discernible from the socket, plus RTT.
5. **Holesail spot-check** (`--holesail` flag, optional) — stand up a loopback echo tunnel
   to prove the holesail path independently.

**Verdict line** (the phone-scriptable output, one of):
`CORRIDOR GREEN` (all pass, direct) · `CORRIDOR AMBER` (connected but relayed/firewalled —
usable, anchor port-forward recommended) · `CORRIDOR RED` (no DHT or no punch — stop, escalate).

CLI: `node kit/probe.mjs [--listen | --dial <key>] [--holesail] [--json]`.
`--json` emits machine-readable results for the ops log.

**Gate G1:** probe runs clean on the founder machine in both roles (two local processes,
real DHT); `--json` schema stable; verdict logic covered by a hermetic self-test mode
(`--self-test`, no network) so CI stays offline-safe.

### Band 2 — Field Kit 2.0, Bahrain edition 🧳 (`mesh/kit/build-kit.mjs` + kit docs)

Adapt the kitchen-table kit for a *remote* Machine B that we never physically touch.

1. **Bundled runtime.** `build-kit.mjs` gains a `--bundle-node` option: copy the local
   `node.exe` (pinned v22 LTS) into the kit folder; `run_mesh.cmd` (and all new .cmd
   entrypoints) prefer the bundled `node.exe` over PATH. Receptionist machine needs ZERO
   installs. Verify UDX/native prebuilds resolve under the copied layout.
2. **Firewall pre-authorization.** New `setup_firewall.cmd` (run-once, self-elevating via
   PowerShell `Start-Process -Verb RunAs`): adds an inbound+outbound Windows Firewall rule
   for the bundled `node.exe` path via `netsh advfirewall`. Idempotent; prints
   `FIREWALL READY` when done. The ceremony card orders this before first run so the
   receptionist never faces a firewall dialog mid-ceremony.
3. **Remote ceremony card.** `README_CORRIDOR.txt` — the kitchen-table ceremony rewritten
   for phone/WhatsApp delivery of the invite + pairing codes (codes are z32; card includes
   the "read it in groups of four" convention and a WhatsApp-paste alternative). English,
   short sentences, numbered steps, `/status` as the universal "what do I read to you" tool.
4. **Probe included.** The built kit contains `probe.mjs` + `run_probe.cmd` +
   `run_probe_dial.cmd` so Band 1 travels inside the same zip.
5. **`/status` hardening.** Extend kit `/status` with: swarm connection count, last-seen
   peer time, transport in use (hyperswarm/tcp), and bundled-node version — each on its own
   short line (I3).

**Gate G2:** `kit-spike.mjs` still 33/33; new `kit2-spike.mjs` proves (hermetically):
bundled-node layout resolves udx native binding; `run_mesh.cmd`/`run_probe.cmd` reference the
bundled runtime; firewall script is syntactically valid + idempotent (dry-run mode
`--print-only` for the gate — no elevation in CI); built Machine-B folder contains the full
corridor file set.

### Band 3 — The Anchor ⚓ (`mesh/kit/anchor.mjs` + ops scripts)

The receptionist machine's second role: always-on full peer so the mesh converges whenever
the founder comes online, regardless of who slept when.

1. **Anchor mode.** `anchor.mjs`: headless (no REPL) kit-host that loads every room in the
   kit registry, joins the swarm, replicates forever. Resilient loop: retry/backoff on DHT
   loss, rejoin on network change, never exits on transient errors. Writes a heartbeat line
   (timestamp + peer count) to `anchor.log` (plaintext-free, I7).
2. **Auto-start.** `install_anchor.cmd`: registers a Windows Scheduled Task (logon trigger,
   restart-on-failure) running the bundled node + `anchor.mjs`. `uninstall_anchor.cmd`
   reverses it. Idempotent both ways.
3. **Optional direct listen.** `--listen <port>` flag: in addition to Hyperswarm, listen on
   a fixed TCP port (the kit's existing `kit-net` TCP transport) so the Era-1-style
   port-forward + DuckDNS name gives a second, DHT-free path (`/connect ph-office.duckdns.org:PORT`
   from the founder kit). Off by default; enabled during field setup since R2/R3 allow it.
4. **Status without a console.** `anchor_status.cmd`: prints the last heartbeat + log tail
   in phone-readable form (I3), so SPOC can check the anchor without touching the task.

**Gate G3:** hermetic `anchor-spike.mjs`: anchor process starts, loads a fixture room over
local TCP transport, survives a killed-and-restarted counterpart (replication resumes),
heartbeat file advances, `--listen` accepts a kit `/connect`; scheduled-task scripts pass
`--print-only` validation. Existing spikes untouched and green.

### Band 4 — Bare sidecar spike 🐻 (STRETCH — `mesh/host/bare-spike/`)

DP4 opener, explicitly non-gating (R4). Attempt: run the protocol-v0 bridge under the Bare
runtime with stdio framing instead of TCP, per the MESSENGER_UI_CAMPAIGN.md §1 "transport
swaps at DP4" seam.

- Deliverable A (success): `bare-bridge.mjs` + adapter passing the bridge-spike method
  checks under Bare, with a written delta report of every compat shim needed.
- Deliverable B (blocked): a findings report — exactly which modules/prebuilds/APIs block
  Bare on Windows today, with recommended path. **A findings report is a fully successful
  outcome for this band**; do not force it.
- Constraint: nothing outside `mesh/host/bare-spike/` + `package.json` devDeps may change (I1).

**Gate G4:** either deliverable, honestly labeled. No green-washing.

## 4. Field ceremony (post-gate, with SPOC — not a coder task)

1. SPOC runs `run_probe.cmd` on the receptionist machine (founder runs `--listen` side) → verdict word.
2. If GREEN/AMBER: `setup_firewall.cmd`, then the README_CORRIDOR ceremony → live corridor test
   (message both ways + file with sha256 spoken-verify).
3. `install_anchor.cmd`; founder goes offline/online to demonstrate anchor convergence.
4. Optional: router port-forward + DuckDNS name for the anchor's `--listen` port (R2 network).
5. Field results recorded as MSG-D27 in MESSENGER_DECISIONS.md (by the gate, not coders).

## 5. Out of scope (do not touch)

- Postgres/GORM sync layer (`sync_service_impl.go` etc.) — Mission B, later wave.
- Frontend/Correspondence screen, `mesh.ts` real-wire flip — that's W-UI-2.
- Reducer/capability/invite law (I1). ERP entity data — messenger rooms only.
- Any git history rewrite, any push. Branch stays local until owner review.

## 6. Coder dispatch plan

| Coder | Band | Writes | Parallel-safe because |
|-------|------|--------|----------------------|
| C1 | 1 | `kit/probe.mjs`, probe cmds, self-test | new files only |
| C2 | 2 | `build-kit.mjs`, `kit2-spike.mjs`, cards, cmds | kit build layer; coordinates file *names* with C1/C3 via this spec |
| C3 | 3 | `kit/anchor.mjs`, anchor cmds, `anchor-spike.mjs` | new files + additive `/status` lines |
| C4 | 4 | `host/bare-spike/**` only | isolated directory |

C2 consumes C1/C3 outputs *by filename only* (names fixed in this spec); the gate wires the
final built-kit verification after all bands land.
