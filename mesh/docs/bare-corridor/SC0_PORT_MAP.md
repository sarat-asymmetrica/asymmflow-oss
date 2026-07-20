# SC-0 — Immersion & Port Map

**Campaign:** The Sealed Corridor · **Mission:** SC-0 (orchestrator, before any code)
**Date:** 2026-07-20 · **Branch:** `feat/fable-sealed-corridor` (off `750a39b`)
**Author:** Opus 4.8 (orchestrator / technical lead / primary gate)

Gate SC-0 requires: the port map exists; every Bare substitution names its package and
its prior-art precedent; probe re-run evidence attached. All three are below.

---

## 1. Probe re-verification — RUN, not cited

The campaign spec says *"Re-verify the probe's DHT claims on today's tree (run it, don't
cite it)."* Done, on `kit/bare-probe.mjs` under the real `bare.exe`.

### 1a. The orchestrator's own wrong probe, recorded rather than discarded

The FIRST attempt at the two-process punch reported **CORRIDOR RED — "could not reach the
listener"**, which would have been a headline finding (the Sealed Ship measured 15/15,
RTT 8 ms). It was wrong, and it was wrong in this campaign line's signature way.

`bare-probe.mjs` carries its own 58-second hard watchdog. The listener was started, its
key was read, and the dial was launched *by hand* some tens of seconds later — by which
time the listener had already printed `hard watchdog fired (>58s) — forcing exit` and
died. The measurement compared a live dialer against a dead listener. Nothing about the
network had changed; the harness had a timing hole.

This is Rule 2 (vary one axis at a time) failing before Rule 1 could even apply, and it is
the orchestrator's own — the fourth such probe on this campaign line, three of them the
orchestrator's. It is recorded here because the corrections are the transferable part.

**Fix:** a driver that spawns the dial the instant the listener prints its `KEY:` line, so
the two probes' lifetimes actually overlap.

### 1b. The re-run, with a negative control

Driver: spawn `--listen`, wait for `KEY:`, immediately spawn `--dial <key>`; and a paired
NEGATIVE CONTROL that dials a **well-formed but wrong** z32 key (every character rotated 7
places through the z32 alphabet — same length, same shape, nobody listening on it).

| leg | N | result |
|---|---|---|
| self-test (hermetic, no network) | 1 | **15/15 checks, SELF-TEST GREEN** |
| single-process live diagnostics | 1 | DHT bootstrap **3/3 servers responded**; network reports **firewalled**; no public address detected; `CORRIDOR AMBER` |
| POSITIVE two-process punch | 7 | **6/7 `CORRIDOR AMBER`**, RTT **8 / 9 / 9 / 9 / 10 / 11 ms**; **1/7 `CORRIDOR RED`** — connection established but the ping/pong round trip timed out after 8000 ms |
| NEGATIVE CONTROL (wrong key) | 7 | **7/7 `CORRIDOR RED`** — punch timed out after 45000 ms, every time |

The driver is committed as `mesh/kit/bare-probe-pair-check.mjs` so this is reproducible
rather than a transcript: `node kit/bare-probe-pair-check.mjs` runs one positive pair and
one negative-control pair and refuses to report a positive result if the negative control
ever comes back green.

**The harness can report the opposite result.** The negative control never once came back
green, so the positive results are admissible as measured.

### 1c. What this changes versus the Sealed Ship's record

Two honest deltas, neither of them a regression in our code:

- **AMBER, not GREEN.** The Sealed Ship recorded a direct punch at RTT 8 ms. Today this
  machine's network self-reports as **firewalled** with **no public address detected**, so
  the same probe returns `CORRIDOR AMBER` (connected, relayed) rather than `CORRIDOR
  GREEN`. This is a property of today's network, not of the code — the punch itself
  succeeds and the RTT is in the same 8–11 ms band.
- **The punch is not 100%.** 1 of 7 positive pairs connected and then failed its
  ping/pong round trip. **N=7 is far too small to state a rate** (Rule 5: a measured rate
  needs N≥30) and no rate is claimed here. It is recorded as an observed, reproducible-
  class failure with a direct design consequence, below.

### 1d. Design consequence for SC-3 (binding on the ceremony copy)

SC-3 wires "Check the connection" to these checks. Because a single probe run can fail
while the corridor is in fact usable, **the guide must not present one probe result as a
definitive verdict about the network.** The Reception-Grade copy has to survive a false
red without either lying or panicking the human. This is a requirement on SC-3, derived
from measurement, not a style note.

---

## 2. Packaging unknowns retired (measured today, `bare-pack@2.2.1`)

The corridor needs sockets and the DHT *inside the sealed bundle*. Neither had ever been
packed — `bare-probe.mjs` has only ever been run as a raw script under `bare.exe`. Both
were tested rather than assumed.

| entry packed | result |
|---|---|
| `import 'bare-tcp'` | packs; offloads `bare-tcp.bare`, `bare-dns.bare`; **runs from a from-scratch dir outside any `node_modules`**: bound a real port |
| `import 'hyperswarm' + 'hyperdht' + 'bare-tcp'` | packs; offloads **six** addons; **runs from hostile geography**: `Hyperswarm` constructed, joined, destroyed clean |

**The six native addons a corridor kit now carries**, each at
`node_modules/<pkg>/prebuilds/win32-x64/<pkg>.bare`:

```
bare-dns   bare-inspect   bare-tcp   bare-type   sodium-native   udx-native
```

**Consequence — a REQUIRED builder hard gate.** `build-bare-kit.mjs` already refuses to
produce a kit whose `dist/reducer.wasm` failed to offload, because that failure is
*broken-but-green*: the kit boots, draws its menu, says Goodbye, and silently cannot post.
A missing `udx-native.bare` or `bare-tcp.bare` has exactly the same shape — the kit would
render its whole ceremony and silently fail to reach the other machine. **The builder must
refuse on its own**, not rely on a spike having been run first. This is an SC-2/SC-4
deliverable, not optional.

### 2a. `bare-tcp` API surface (measured under Bare, not read off a README)

```
Server, Socket, connect, constants, createConnection, createServer,
errors, isIP, isIPv4, isIPv6, socketpair
```

Loopback `createServer` + `connect` round-tripped a payload; a connect to a dead port
errored with `connection refused` (the negative control fired). It is a near drop-in for
`node:net` at exactly the surface `kit-net.mjs` uses.

### 2b. An observation, honestly under-claimed

A Bare script holding a `bare-tcp` server and socket **did not exit on its own** — an
explicit `Bare.exit(0)` was required. Pending `setTimeout` handles were *also* live in
that script, so **the cause is not isolated to `bare-tcp`** and no claim is made that it
is. It re-confirms RULE 3's practice (an explicit exit is load-bearing; never rely on
natural drain), which every file in this campaign already follows.

---

## 3. The port map

### 3.1 `kit-registry.mjs` → `kit/bare-registry.mjs` (SC-1)

| element | verdict |
|---|---|
| the `rooms.json` shape (`roomKey, storage, authorityPub, encryptionKey, bootstrap, title`, plus `lastPeer`) | **PORTS VERBATIM** |
| idempotent-by-`roomKey` `saveRoomRegistryEntry` | **PORTS VERBATIM** |
| `updateRoomRegistryPeer` (the F2 auto-reconnect stamp) | **PORTS VERBATIM** |
| corrupt registry returns `[]` and never crashes boot | **PORTS VERBATIM** — this is law, not defensiveness |
| `import { readFileSync, writeFileSync, existsSync } from 'node:fs'` | **BARE SUBSTITUTE: `#fs`** (→ `bare-fs`). Precedent: `mesh/package.json`'s condition map, landed by `PHASE0_GATE_B3_CONDITION_MAP.md`; used by `bare-bridge.mjs` and `bare-guide.mjs` today. |
| `import { join } from 'node:path'` | **BARE SUBSTITUTE: hand-rolled `joinPath`.** Precedent: `bare-bridge.mjs`'s own three-function path helper, added rather than pulling a `#path` alias for something this small. |
| the reopen LOOP itself (`kit-host.mjs` lines ~131-158) | **PORTS**, into `bare-guide.mjs`'s `ensureMessengerCore()` — `createMeshNode` + `core.registerRoom`, both already reachable in the sealed bundle |
| `kit-host.mjs`'s bundled-`node.exe` detection | **OUT OF SCOPE** — meaningless for a sealed Bare kit; there is no PATH-vs-bundled ambiguity to report on, by construction |
| `kit-host.mjs`'s `persistentActor` readline prompt | **OUT OF SCOPE for SC-1.** The guide already has its own FIFO-queue stdin layer; an actor-name question is ceremony copy and belongs to SC-3 if it is wanted at all. |

**The trap SC-1 must not fall into.** A registry alone does not fix the "kitchen table
every run" bug. `bare-bridge.mjs`'s `createSocialRoom` wire method picks a random
`social-<hex>` storage dir, so there would be no stable directory to reopen against.
`kit-repl.mjs` already solved this with a **declared deviation** — bypass the wire method,
call `social-room.mjs` directly with a directory name the kit chooses and remembers. That
deviation ports with the registry; it is half the fix, not an implementation detail.

### 3.2 `kit-net.mjs` → `kit/bare-net.mjs` (SC-2)

| element | verdict |
|---|---|
| `roomTopic()` = **discoveryKey**, never the raw base key | **PORTS VERBATIM** — capability hygiene, not style |
| hyperswarm primary, per-topic routing via `peerInfo.topics`, best-effort fallthrough | **PORTS VERBATIM** (`hyperswarm` proven Bare-clean, and now proven *packable*, §2) |
| direct-TCP REQUIRED fallback | **PORTS**, on a substitute (below) |
| replication wire `node.store.replicate(socket)` piped onto a raw duplex | **PORTS VERBATIM** — same primitive `mesh-node.mjs` and `peer.mjs` already use. Mesh law; not re-invented. |
| `close()` destroying sockets BEFORE awaiting `server.close()` | **PORTS VERBATIM** — inverting it deadlocks forever, per that file's own comment |
| `listenTcp` resolving the ACTUAL bound port | **PORTS VERBATIM** — a first draft in the Node line persisted `peer:0` and silently poisoned auto-reconnect |
| `peerCount` / `swarmPeerCount` / `lastPeerSeenAt` honesty split | **PORTS VERBATIM** — the split exists so `/status` never reports a fabricated per-room number |
| `import net from 'node:net'` | **BARE SUBSTITUTE: `bare-tcp@2.5.2`** — the ONE pre-approved new dependency (campaign §1). Surface verified empirically (§2a). Precedent: named in `CAMPAIGN_REPORT.md` §8 item 5 as the explicit-dependency question, and in `bare-probe.mjs`'s own header as the blocker that made check 5 un-portable. |
| `hcrypto.discoveryKey`, `Hyperswarm` | **PORT VERBATIM** — already Bare-clean (11/11, `BARE_SPIKE_REPORT.md`), now also pack-clean (§2) |
| `--no-hyperswarm` degradation | **PORTS** as the `useHyperswarm: false` constructor flag it already is |
| **Holesail, any path** | **BANNED.** Not the kit transport, never was; `peer.mjs`'s era. Its optional import is literally what caused FR-1a. Reintroducing it is a stop-and-report. |

### 3.3 The `/addwriter` onboarding ceremony (SC-3)

| element | verdict |
|---|---|
| `node.addWriter(key)` reached **in-process** via the rooms Map | **PORTS VERBATIM.** Protocol v0 deliberately has no become-a-writer wire method (bridge deviation #2). `bare-bridge.mjs` preserves that deviation exactly; the Bare guide holds its `createBridgeCore` in-process, so the same in-process reach is available for the same reason `kit-host.mjs` and `kit-repl.mjs` run in one process. |
| joiner side: print pairing code → wait for `node.writable` → `redeemInvite` | **PORTS VERBATIM** (`kit-repl.mjs`'s `join()`), including `waitFor(… node.base.update(); return node.writable …)` |
| pairing-code sanitisation `String(x).replace(/[<>"'`\s]/g,'')` | **PORTS VERBATIM** — WhatsApp/voice-paste tolerance is field-critical |
| the already-joined short-circuit (never show a joiner "exhausted" for doing nothing wrong) | **PORTS VERBATIM** — MSG-D25, a real field failure |
| `randomUUID`/`randomBytes` from `node:crypto` | **BARE SUBSTITUTE: `hypercore-crypto`'s `randomBytes`.** Precedent: `bare-bridge.mjs`'s `randomHex()` — already a direct dependency, already proven Bare-clean, not a `node:crypto` substitute that needed inventing. |
| `createLiveStream` (the live message feed, `room-updated` diffing by `msgId`) | **DEFERRED, declared.** Not required by any SC gate. Its identity-based-diff lesson is recorded so a later wave does not re-derive it positionally. |
| `/attach` · `/fetch` · `exportTranscript` | **OUT OF SCOPE** — no corridor gate needs them; the wire methods already exist in `bare-bridge.mjs` if a later wave wants them. |

### 3.4 A2.1 Reception-Grade guide copy (SC-3, and it is a REQUIREMENT)

| element | verdict |
|---|---|
| one double-click entry point; **clients are never handed a command line** (ruling R6) | **PORTS VERBATIM** |
| plain-question menu, paste-and-Enter, zero arguments anywhere | **PORTS VERBATIM** — already law in `bare-guide.mjs` |
| `normalizeCode` / `groupInFours` (grouped-4 codes, whitespace tolerance) | **ALREADY PORTED** byte-for-byte in `bare-guide.mjs`; SC-3 must *use* them on the invite path, which today it does not |
| the three literal verdict words + "Read this word to the person on the call:" | **PORTS VERBATIM** — `printVerdictLarge` already exists and is currently unused; SC-3 wires it |
| error fold-line convention (one plain sentence, a rule, then raw detail) | **ALREADY PORTED** (`reportError`) |
| the firewall OFFER's copy | **ALREADY PORTED**; the ACTION stays an honest stub — the real mutation is `netsh` + elevation, an outward-facing system change the machine's owner has not reviewed. Out of scope here; it stays labelled. |
| Menu [1] "Check the connection" | **CURRENTLY AN HONEST STUB → SC-3 makes it real** from `bare-probe.mjs`'s checks 1–4, with client-word verdicts, subject to §1d |
| Menu [3]/[4] anchor + status system mutation | **OUT OF SCOPE** — scheduled-task/firewall mutation stays an honest stub. Not a corridor gate; unchanged from the Sealed Ship. |

---

## 4. What SC-0 did NOT verify

Stated plainly, per D5:

- **Nothing about two real machines.** Every measurement here is single-machine (two
  processes). The corridor's actual claim — India↔Bahrain class WAN — is owner-reserved
  field work (SC-5) and is untouched.
- **No rate is established for the punch.** N=6 is evidence of a failure mode, not a rate.
- **`bare-tcp` was proven to bind, connect, echo and refuse.** It was **not** yet proven
  to carry a Corestore replication stream — that is SC-2's job and SC-2 must not assume it.
- **The six offloaded addons were proven to load and construct.** `sodium-native` and
  `udx-native` doing real cryptographic and UDP work *inside a sealed bundle over a real
  corridor* is SC-2/SC-3 evidence, not SC-0's.
- **No claim about a clean machine.** This machine has Node and npm installed globally.
  "Ran from a directory outside `node_modules`" is geographic hermeticity, which is a
  weaker guarantee, and is the only one claimed here.

---

*Port the proven, seal the port, prove the seal.* 🐻
