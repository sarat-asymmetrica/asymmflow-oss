# SC-3a — The Two-Sided Corridor Ceremony

**Campaign:** The Sealed Corridor · **Mission:** SC-3a · **Date:** 2026-07-20
**Branch:** `feat/fable-sealed-corridor` · **Author:** Sonnet 5 coder

Gate SC-3 requires a scripted two-process ceremony through the REAL launchers
(kit A mints, kit B redeems, A grants, B posts, A reads B's message back —
content-asserted), plus negative controls run FIRST, plus N≥16 on any leg
where a hang or a race is plausible. All of that is below, along with a real
bug this gate found and fixed.

---

## 1. What was built

- `mesh/kit/bare-guide.mjs` — menu [2] grows the corridor fork. `openMessenger`
  asks the founder/joiner question (in the D3-ratified unified form the
  orchestrator landed mid-mission — see §4), then dispatches to two new
  functions:
  - `startAsFounder` — Path A: reopen-or-create, mint an invite
    (`openDmInvite`), print it grouped-in-fours, start the network
    (`ensureNetwork` → `bare-net.mjs`, dynamically imported), accept the
    other machine's pairing code and call `node.addWriter()` in-process
    (deviation #2), then the shared REPL.
  - `joinAsJoiner` — Path B: decode the pasted invite, create a mesh node in
    a STABLE `joined-<prefix>` dir, save the registry entry immediately
    (F2(a)), print this device's pairing code, start the network, offer an
    optional LAN `ip:port` hint, wait (bounded, `JOIN_WRITABLE_TIMEOUT_MS` =
    90s) for `node.writable`, then `redeemInvite`. Ports kit-repl.mjs's
    already-joined short-circuit (MSG-D25) at both entry points.
  - `reopenOrCreateRoom` / `messengerRepl` — extracted, unchanged-behavior
    helpers shared by every path that reaches a usable room.
  - `ensureNetwork` — the ONE dynamic import of `bare-net.mjs`, for the
    measured reason given in its own comment (§3 below).
- `mesh/kit/bare-corridor-spike.mjs` (NEW) — this gate.
- One appended npm script: `"sc3aspike": "npm run build && node
  kit/bare-corridor-spike.mjs"`.

## 2. Gate command

```
npm run sc3aspike
```
(equivalently `node kit/bare-corridor-spike.mjs`). Builds
`kit/bare-guide-entry.mjs` into a private `kit/.sc3a-dist` via
`build-bare-kit.mjs --require-addons=udx-native,sodium-native,bare-tcp,bare-dns`,
then drives `run_bare_mesh.cmd` (never `bare.exe app.bundle`) through
`cmd.exe /c <abs path>` with `shell:false` and `ASYMMFLOW_KIT_NONINTERACTIVE=1`
— copied from `sealed-corridor-gate.mjs`'s own `runLauncher`, per Rule 3 and
the campaign spec's own wording for this wave.

## 3. Measured evidence

**Two full runs** (2026-07-20, both at the SC-5 merge gate after the
orchestrator's session crashed mid-collection; run by the gate, not the
coder). Numbers below give both. Where they differ, §3.3 records the
isolation that explains it.

### 3.1 Negative controls (run first, per Rule 1)

| control | N | run 1 | run 2 |
|---|---|---|---|
| (i) malformed invite code — plain refusal, guide survives | 5 | OK=5/5 | OK=5/5 |
| (ii)/(iii) well-formed-but-wrong code, FAST correctness sweep (never falsely joins) | 16 | OK=16/16 | OK=16/16 |
| (ii)/(iii) well-formed-but-wrong code, SLOW honest-copy shape (exact failure text + Goodbye, never a hang) | 3 | OK=3/3 | OK=3/3 |

**Framing, stated plainly:** this gate does not have a way to sever the real
internet/DHT from inside the harness, so "unreachable network" (iii) is
exercised via a syntactically-valid invite code for a room nobody founded —
the network path for THAT specific room genuinely cannot ever complete
(nobody is listening on its derived hyperswarm topic, no writer grant will
ever arrive). This is the same scope as (ii) and is reported as one combined
control rather than a separately-severed network, which is not achievable
from this harness.

### 3.2 Positive: the two-sided ceremony

| leg | N | run 1 | run 2 |
|---|---|---|---|
| LAN-assisted (deterministic, GATED) | 16 | 15/16 (see §3.3) | **16/16** |
| hyperswarm-only (live DHT, MEASURED, not gated pass/fail — same convention as `bare-net-spike.mjs`'s own swarm leg) | 5 | 3/5 | 2/5 |

### 3.3 Run 1's single red, isolated (the gate's own Rule-2 pass)

Run 1's only failure was round 4/16: `timeout waiting for A invite intro` —
kit A never printed its invite inside the harness's **20-second** bound. That
bound was the tightest in the ceremony and the only one that absorbs the
ENTIRE cold boot (bare.exe start + Defender scanning a freshly copied
unsigned 45 MB exe + bundle load + reducer wasm + Autobase room founding);
every later wait runs in a warm process. The failure never appeared at any
other bound in either run, and run 2 — same instrument, byte-identical,
quiet machine — went 16/16. **Combined: 31/32 ceremonies green, the sole
miss at the sole cold-boot-absorbing bound.** Same disease class
`sealed-corridor-gate.mjs`'s own leg C already diagnosed and widened its
timeout for (60s → 150s, machine-load HANGs with empty stdout).

The verdict follows that precedent: the kit is healthy; the harness bound
was tight. The first wait was widened 20s → 90s AFTER both runs above (the
numbers in this file were measured with the 20s bound — the fix is for the
next reader's hour, not this report's numbers), with the measurement recorded
at the fix site.

The hyperswarm-only leg's fractions (3/5, 2/5; 5/10 combined) are consistent
with this network's SC-0 characterisation (firewalled/AMBER, 1/7 RED even on
a working corridor) and with SC-2's own 11/16 at the lower layer. Live-DHT
reds concentrated in early rounds both runs; TCP/LAN is the deterministic
path and is why it is the gated one.

Each round: kit A mints an invite and starts the network; kit B (a live,
separate `run_bare_mesh.cmd` process) redeems it via a real bidirectional
relay (B's pairing code is read off B's live stdout and typed into A's live
stdin, mid-run — no fixed stdin script can express this, same reasoning as
`bare-net-spike.mjs`'s own header); A grants via `addWriter`; B posts; A
reads B's message back via `/rooms` (content-asserted, the spec's own literal
requirement); A posts a second marker and B reads THAT back too
(content-asserted, both directions — see §5 for why a second marker was
necessary rather than checking the first one).

## 4. A live bug found and fixed, mid-mission

**The bug:** `ensureMessengerCore` hardcoded `actor: 'guide'` for every
device. `msgId` is `{actor}:{seq}` (`bare-bridge.mjs`'s own header) and
`takeSeq` computes each device's next seq from its OWN local view — so two
DIFFERENT physical machines sharing the literal actor name `'guide'` could
independently compute the SAME seq for two DIFFERENT messages. Reproduced
live at this gate's own smoke pass: a founder's second post was silently
rejected — `(not posted -- duplicate msgId "guide:4")` — after the joiner's
own post happened to land on the same seq. This was invisible in every prior
single-device gate (SC-1, SC-3b) because a device only ever collides with
itself, never with another device, until SC-3a put two real devices in one
room for the first time.

**The fix** (`bare-guide.mjs`, `ensureMessengerCore`): each device derives its
actor from its OWN public key — `guide-${keys.pubHex.slice(0, 8)}` — instead
of the shared literal. Two devices never share a keypair, so the collision is
structurally impossible; the same device's identity stays stable across
restarts (the seed persists), so `kit-registry.mjs`'s own reopen discipline
is unaffected. All THREE call sites that carried the literal (`createBridgeCore`,
`createSocialRoomNode`, `redeemInvite`) were updated together.

**Scope note:** this is a host-side identity CHOICE (what string this file
passes as its own participant), not a protocol-v0 method, reducer, or
capability/invite semantic change — the campaign's stop-and-report list does
not cover it, and `SC0_PORT_MAP.md §3.1` explicitly flagged an actor-name
question as "SC-3's [territory], if it is wanted at all." It was wanted:
SC-3a is the first mission that puts two real devices in one room.

## 5. A test-design lesson, recorded rather than silently fixed

The FIRST draft of the both-directions content check failed
(`B never saw A's exact message text`) — not a guide defect. `bare-guide.mjs`'s
`/rooms` shows only the room's SINGLE most recent message (`roomSummary`'s
`lastPreview`, not a history). Since A posts before B in the round, B's post
is always the globally-later op, so A's own check (does A see B's message?)
is meaningful — but the mirror check (does B see A's FIRST message?) can
never succeed regardless of replication, because A's first post is never the
"last" message once B has posted after it. Fixed by having A post a SECOND
marker once B's message is confirmed seen, making IT the new global-last op
for B's own check. Recorded here per this campaign's own transparency norm —
a probe that measures something other than what its author believed is the
single most common failure mode this campaign line has produced.

## 6. Coordination note

Mid-mission, `bare-guide.mjs`'s `openMessenger` question flow changed
underneath this gate (the orchestrator's own D3 fix + a `verify-clean-machine.ps1`
compatibility fix: the SAME question is now asked whether or not a room
already exists, with "connect" as the deliberate path into the corridor
ceremony rather than Enter being an accidental one). This gate's stdin
sequences were updated to match (every fixed/scripted `'2', 'skip', ...`
sequence now includes an explicit `'connect'` line before the code question).
Verified against the CURRENT file, not the one this mission started against.

## 7. What this gate did NOT verify

- **Nothing about two real machines.** Every round here is two PROCESSES on
  ONE machine (two hostile temp dirs), same scope note SC0/SC2 already gave.
  The India↔Bahrain corridor is owner-reserved field work (SC-5/field
  ceremony), untouched here.
- **The hyperswarm-only leg is measured, not gated** — live DHT is legitimately
  allowed to be flaky (SC0_PORT_MAP.md §1: up to 45s, 1/7 RED even on a working
  corridor), so this leg's fraction is reported honestly rather than failing
  the whole gate on DHT variance, matching `bare-net-spike.mjs`'s own
  precedent exactly.
- **No claim about a clean machine.** This machine has Node/npm installed
  globally; only geographic hermeticity (temp dirs outside any `node_modules`)
  is claimed, same scope note as every other gate in this campaign.
- **Menu [3]/[4] (anchor/status) and menu [1]'s punch test (check 4)** are
  unchanged/out of scope for this mission.
- **The "unreachable network" control is not a literally severed connection**
  — see §3.1's framing note.

---

*Port the proven, seal the port, prove the seal.* 🐻
