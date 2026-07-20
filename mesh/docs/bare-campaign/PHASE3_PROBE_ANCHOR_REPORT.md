# Phase 3 — Probe and Anchor Roles, Bare Port

**Coder:** P0-D · **Branch:** `feat/fable-bare-runtime` · **Date:** 2026-07-20
**Owns:** `mesh/kit/bare-probe.mjs`, `mesh/kit/bare-anchor.mjs`, `mesh/kit/bare-anchor-spike.mjs`,
this report. Did not touch `probe.mjs`/`anchor.mjs`/`anchor-spike.mjs`/`guide.mjs`/
`build-kit.mjs` or their spikes (rollback path, still green), `bare-guide.mjs`/
`bare-guide-spike.mjs` (P1A), `build-bare-kit.mjs` (P0-B), `host/bare-bridge*.mjs`,
`apply.mjs`, `reducer/**`, `cmd/**`, `goldens/**`. No scheduled task installed, modified,
or uninstalled on this machine (R4).

---

## 1. Verdict per role

**Probe: PORTS IN FULL for its core diagnostic job.** All four load-bearing checks (DHT
bootstrap, NAT self-diagnosis, CGNAT card, hyperswarm punch test) run under Bare with the
identical diagnostic vocabulary as `probe.mjs`, verified live against the real DHT and a real
two-process punch test (§3). Only the *optional* `--holesail` loopback spot-check does not
port today, and degrades gracefully rather than crashing (§2, §5) — the probe's core "can this
network mesh?" job is undiminished.

**Anchor: the OUTER SHAPE ports in full (resilience loop, heartbeat format, shutdown
discipline); the actual room-hosting PAYLOAD does not port yet, and cannot without touching a
file reserved to another in-flight coder.** `bare-anchor.mjs` exports the same `anchorMain()`
contract as `anchor.mjs` — capped exponential backoff, an I7-identical heartbeat line, shutdown
only via an explicit `requestShutdown()` — proven against a real (non-trivial) fixture boot
function through a real spawn pipe (§3). Its CLI entry point refuses honestly, every time,
rather than pretending to boot something it cannot (§3, §5). This is a real, load-bearing
boundary, not a shortcut: room-hosting needs `host/bare-bridge.mjs` (P1A, reserved and
in-flight).

---

## 2. Diagnostic-vocabulary conformance table

Every string below is copy-checked against `probe.mjs`'s own literal source, not
re-derived from memory.

| Element | `probe.mjs` (original) | `bare-probe.mjs` (this port) | Conformance |
|---|---|---|---|
| Header banner | `Mission A2 Band 1 — The Corridor Probe` | `Mission A2 Band 1 — The Corridor Probe (Bare runtime)` | Identical + explicit runtime tag (deliberate — a field contact reading two transcripts side by side should be able to tell which one ran) |
| DHT pass line | `PASS: DHT bootstrap reachable (N/M servers responded)` | same | Byte-identical |
| DHT fail line | `FAIL: DHT bootstrap unreachable — <msg>` | same | Byte-identical |
| NAT firewalled | `INFO: this network reports as firewalled — a direct listener may not be reachable from outside` | same | Byte-identical |
| NAT clear | `PASS: this network is not firewalled (direct connections should work)` | same | Byte-identical |
| Public address line | `PUBLIC ADDRESS: <host>:<port>  <-- read this IP aloud for the CGNAT check below` | same | Byte-identical |
| CGNAT card (3 lines + framing blanks) | `CGNAT CHECK (ask SPOC to do this — takes 30 seconds):` / compare-address line / same-or-different line | same | Byte-identical |
| Listen key line | `PASS: listening — give this key to the other side:` / `  KEY: <z32 grouped>` | same | Byte-identical |
| Punch peer-connected | `PASS: a peer connected — echoing ping back` | same | Byte-identical |
| Punch dial success | `PASS: punch succeeded — RTT <n>ms (<relay hint>)` | same | Byte-identical |
| Punch fail | `FAIL: punch test — <msg>` | same | Byte-identical |
| No-punch-requested | `INFO: no --listen/--dial given — skipping the punch test (diagnostics only)` | same | Byte-identical |
| Holesail absent | `INFO: holesail check not included in this kit — skipping` | same | Byte-identical (this specific case still shares the original's exact wording) |
| Holesail present but unprovable under Bare | *(no equivalent case in the original — Node always has `node:net`)* | `INFO: holesail loopback check needs a TCP loopback primitive not yet available under Bare in this kit (no node:net equivalent wired — see PHASE3_PROBE_ANCHOR_REPORT.md) — skipping` | **New line, not a reworded original** — see §5 |
| `z32Groups()` | groups into 4s, join with a space, pass-through on odd input | identical implementation, copied | Byte-identical |
| `computeVerdict()` reason strings (RED/AMBER/GREEN, all branches) | 5 distinct reason strings | identical, copied verbatim | Byte-identical |
| Verdict words | `CORRIDOR GREEN` / `CORRIDOR AMBER` / `CORRIDOR RED` (exactly these three) | identical | Byte-identical |
| `--json` schema tag | `asymm-corridor-probe.v1` | `asymm-corridor-probe-bare.v1` | **Deliberately distinct** (§ below every other field is unchanged shape) — an ops-log consumer can tell which runtime produced a report; nothing else in the schema differs |

**Net assessment**: the vocabulary a field contact or SPOC actually hears/reads —
PASS/FAIL/INFO lines, the CGNAT card, the z32 key grouping, and the three verdict words — is
unchanged. The two additions (the "(Bare runtime)" banner tag and the new holesail-under-Bare
skip line) are both new *information*, not rewordings of existing vocabulary, and both exist
because this runtime genuinely differs from the one the original vocabulary was written for —
consistent with the charter's instruction to preserve the vocabulary, not to pretend the two
runtimes are identical when they aren't.

---

## 3. Spike results

### 3a. `bare-probe.mjs` — self-test (hermetic, no network)

```
$ npx bare kit/bare-probe.mjs --self-test
bare-probe.mjs self-test (hermetic — no network)

  [OK] z32Groups groups a 52-char key into 13 groups of 4
  [OK] z32Groups is a no-op-ish pass-through on non-multiple-of-4 input (never throws)
  [OK] RED — no DHT reachable
  [OK] RED — punch requested but no peer arrived
  [OK] AMBER — firewalled but punch OK and direct
  [OK] AMBER — not firewalled but punch is relayed
  [OK] AMBER — holesail spot-check failed alone
  [OK] GREEN — direct punch confirmed, not firewalled
  [OK] GREEN (provisional) — no punch requested, diagnostics only clean
  [OK] GREEN — holesail unattempted (skipped, either reason) never drags the verdict down
  [OK] json report: bare-specific schema tag present
  [OK] json report: verdict is one of the three exact words
  [OK] json report: checks.dht/nat/punch/holesail all present
  [OK] json report: round-trips through JSON.stringify/parse losslessly
  [OK] output: no raw 64-hex key ever gets printed by the probe's own formatting helpers

SELF-TEST GREEN
```
One fixture was added beyond the original's set ("holesail unattempted never drags the
verdict down") specifically covering the new Bare-only skip path — the original had no
case distinguishing "holesail unattempted because absent" from "holesail unattempted because
this runtime can't prove it," since under Node that second case cannot occur.

### 3b. `bare-probe.mjs` — live diagnostics, real DHT

```
$ npx bare kit/bare-probe.mjs --json --holesail
{"schema":"asymm-corridor-probe-bare.v1","timestamp":"2026-07-20T06:04:44.250Z",
 "verdict":"CORRIDOR AMBER",
 "reason":"this network reports firewalled/behind NAT — usable; the anchor port-forward path (R1/R2) is recommended",
 "checks":{"dht":{"total":3,"reachable":3},
           "nat":{"firewalled":true,"publicHost":null,"publicPort":null},
           "punch":{"attempted":false,...},"holesail":{"attempted":false,...}}}
```
Real hyperdht bootstrap (3/3 servers responded), real NAT self-diagnosis, correct verdict
computation — all live, not fixtures.

### 3c. `bare-probe.mjs` — real two-process punch test (Gate G1's own requirement: "probe runs
clean on the founder machine in both roles")

Two real `bare.exe` processes, one `--listen`, one `--dial <key>`, on this machine:

```
[listen side]
PASS: DHT bootstrap reachable (3/3 servers responded)
PASS: listening — give this key to the other side:
  KEY: ac79 cejy takm xfkg cbbq 1bb5 t5xj ep4n s4e8 yajc s5wf 7nfu ip4y
PASS: a peer connected — echoing ping back

[dial side]
PASS: DHT bootstrap reachable (3/3 servers responded)
PASS: punch succeeded — RTT 8ms (possibly relayed)
CORRIDOR AMBER   (this network is firewalled + relayed — correct verdict for this network)
```
A real hyperswarm rendezvous, real ping/pong round trip (8ms), correct verdict on both sides.

### 3d. `bare-anchor-spike.mjs` — full run

```
$ node kit/bare-anchor-spike.mjs
Phase 3 — bare-anchor spike: loop contract (real spawn pipe) + negative control + real CLI check

=== positive: real fixture boot, full loop contract ===
  [OK] anchor booted and wrote a heartbeat file
  [OK] heartbeat line format is timestamp+peers+rooms+mode, I7-identical to anchor.mjs
  [OK] anchor never exited after boot (loop is alive)
  [OK] heartbeat file advances on its own over time
  [OK] tick() was actually invoked by the heartbeat cadence (peers counter advanced)
  [OK] anchor shuts down cleanly ONLY on an explicit requestShutdown() call
  [OK] the fixture session was actually close()d on shutdown

=== retry: boot-time failure retried with backoff, then recovers ===
  [OK] a boot-time failure is logged (not swallowed) and does not crash the process
  [OK] anchor RECOVERS after retrying past deliberate boot failures (heartbeat eventually appears)
  [OK] exactly 3 boot attempts occurred (2 deliberate failures + 1 success)

=== negative control: permanently-broken boot, spike must detect it ===
  [OK] NEGATIVE CONTROL: a permanently-broken boot() correctly produces NO heartbeat file
  [OK] NEGATIVE CONTROL: the anchor is still alive (retrying) rather than crashing on permanent boot failure
  [OK] NEGATIVE CONTROL: at least one boot attempt was made and failed (fixture actually exercised)

=== §4 — RULE 4: the REAL, unmodified bare-anchor.mjs CLI entry point, through a real spawn pipe ===
  bare-anchor.mjs CLI via real spawn pipe (no args): OK=5/5 PARTIAL=0/5 TOTAL_LOSS=0/5 HANG=0/5
  [OK] §4: the real CLI entry point, spawned through a real OS pipe, refuses with its documented message every run

BARE-ANCHOR SPIKE GREEN
```

**Negative control, explicit**: the "negative control" section boots `anchorMain()` against a
fixture whose `boot()` throws on every attempt (never succeeds). The SAME assertion used in
the positive-path test ("did a heartbeat file appear?") is run again here, and the spike
asserts it correctly observed **no** heartbeat — proving this spike's own checks can see and
report failure, not merely confirm success (mirrors this coder's `spawn-pipe-harness.mjs`
`selfTest()` from Phase 2, same doctrine applied to a different module).

**Architecture note kept visible rather than smoothed over**: `bare-anchor.mjs` is Bare-only
(imports `bare-fs`/`bare-process` directly, which do not resolve under plain Node — confirmed:
`node -e "require('bare-crypto')"` throws `TypeError: require.addon is not a function`, shared
by all three native-hook packages). `spawn-pipe-harness.mjs` is Node-only by design (it is the
*parent* half of a spawn pipe). One file cannot import both. `bare-anchor-spike.mjs` is
therefore a **Node** file (like `stdio-seam-spike.mjs`) that generates small Bare fixture
scripts under `mesh/kit/` (so Bare's module resolution finds `mesh/node_modules` — the same
lesson `stdio-seam-spike.mjs` already paid for), spawns them via `runSpawnPipe`, and parses
`RESULT:` lines they print via `console.log`. Building this surfaced two real bugs in **my own
spike code**, not in Bare, both left as visible comments in the file rather than silently
fixed: (a) `spawn-pipe-harness.mjs`'s `args` array is placed *before* the script path in
`spawn(exe, [...args, scriptPath], …)`, so passing `dataDir` via `args` never reached the
fixture's `process.argv` — fixed by embedding `dataDir` as a JS string literal at fixture-
generation time (same technique `stdio-seam-spike.mjs` already uses for the WASM path); (b) an
early `RESULT:` line parser's regex was anchored with `$` immediately after the payload with
no tolerance for Bare's CRLF line endings on win32, so it matched **zero** lines against
completely healthy output — combined with a debug print that only showed
`stdout.slice(0, 500)`, this briefly looked exactly like a new Bug-A-class silent-truncation
finding. It was not: a 20-run stress test of the identical fixture via the corrected assertion
(plain substring match, not the broken regex) showed **20/20 clean, full output every time**.
The regex was then fixed (strip a trailing `\r` before matching) and the full spike re-run
clean. Recorded here per the honesty law — a promising-looking "new flush-race finding" that
turned out to be an off-by-one in the observer is itself worth keeping visible, not quietly
deleting the evidence of the false alarm.

---

## 4. Hostile-geography behaviour

`kit/bare-probe.mjs` copied to `…/scratchpad/p3-hostile/kit/`, with a **freshly, separately
`npm install`ed** `node_modules` (`bare`, `bare-process`, `bare-crypto`, `hyperdht`,
`hyperswarm`, `hypercore-id-encoding`) and a hand-written `package.json` — a from-scratch
provisioning, not a copy of `mesh/node_modules`.

```
$ npx bare kit/bare-probe.mjs --self-test      (from outside the repo tree)
SELF-TEST GREEN                                 (all 15 checks, identical to §3a)

$ npx bare kit/bare-probe.mjs --json --holesail
{"schema":"asymm-corridor-probe-bare.v1", "verdict":"CORRIDOR AMBER", ...}
                                                 (real live DHT bootstrap, correct verdict)
```
Both hermetic and live-network checks ran cleanly outside the repo tree. FR-1a is closed for
this file: every optional/unproven-under-Bare import (`holesail`, and the (never attempted)
TCP loopback primitive) is behind a `try/catch` at point of use, never a bare top-level
`import`.

**A second, real packaging landmine found and documented, not swept under the rug**: this
file's `import { randomBytes } from '#crypto'` depends on the **consuming** `package.json`
declaring the `#crypto` → `bare-crypto` import-map condition (`mesh/package.json`'s own
`imports` field). Deleting that field from the hostile-geography `package.json` and re-running
produced an immediate, unguarded crash:

```
Uncaught ModuleResolveError: PACKAGE_IMPORT_NOT_DEFINED: Package import specifier '#crypto'
is not defined by "imports" in '.../package.json'
```
exit 127. Unlike `holesail` (an optional feature, correctly lazy-guarded), `#crypto` is used
unconditionally by the probe's core DHT/punch logic — there is no graceful-degrade path for it
the way there is for holesail, because without it the probe cannot do its primary job at all.
**This is an actionable finding for `build-bare-kit.mjs` (P0-B, in flight): any packaged kit
must carry the `#crypto`/`#fs`/`#apply` import-map conditions from `mesh/package.json` into
whatever `package.json` ships alongside a Bare-native kit, or every file using a `#`-prefixed
import (this one, and likely others) hard-crashes the moment it's run from the built kit
folder rather than the dev tree** — structurally the same failure class as FR-1a, one layer
down (a missing package-level config, not a missing package).

---

## 5. Exactly what is owner-gated and unproven

- **`bare-tcp` is not an explicit `mesh/package.json` dependency.** It (and `bare-net`)
  already resolve in this tree today, but ONLY as undeclared transitive dependencies (via
  `holesail`'s own dependency chain, and via `bare`'s own `bare-subprocess`). Importing an
  undeclared transitive package directly would reproduce the exact fragility this campaign's
  binding rules exist to close, so `bare-probe.mjs` does not do it — the `--holesail` loopback
  spot-check degrades gracefully instead (§2, §4). **If `bare-tcp` is added as an explicit
  devDependency (a gate/owner decision, per the dispatch's own instruction — not mine to make
  unilaterally), the holesail loopback check can be ported in full**, and so could a future
  Bare-native TCP fallback transport for the anchor. Flagging the exact version already present
  transitively (`bare-tcp@2.5.2`) so the gate has it without re-deriving it.
- **The anchor's real room-hosting duty is entirely unproven under Bare, and cannot be proven
  without touching `host/bare-bridge.mjs` (P1A, reserved and in-flight).** `anchor.mjs`'s
  actual payload — `kit-host.mjs`'s `createKitHost()` — transitively imports
  `bridge-server.mjs`/`bridge-client.mjs`, which hardcode `node:net`/`node:crypto` for their
  OWN internal localhost RPC bridge (the seam `kit-repl.mjs`'s command layer talks over), not
  merely for `kit-net.mjs`'s optional TCP fallback. `mesh-node.mjs` itself is ALREADY
  Bare-portable (verified: resolves entirely through `#apply`/`#crypto`, and
  `host/bare-entry.mjs` already proves a real reducer fold under Bare with a byte-matched
  digest) — the reducer fold is not the blocker; the bridge is. This is a real, structural,
  correctly-identified boundary: reimplementing or bypassing that bridge here would step
  directly on P1A's reserved file and duplicate work this campaign has already assigned
  elsewhere. `bare-anchor.mjs`'s `boot` parameter is REQUIRED with no bundled default,
  specifically so that wiring the real bridge in later is a small, additive change (see that
  file's header) rather than a rewrite.
- **The scheduled-task migration itself (install/uninstall a real Windows Scheduled Task
  pointing at a sealed Bare artifact instead of `node.exe`) is Phase 4, explicitly NOT
  authorized (owner ruling R4).** Nothing in this coder's scope installs, modifies, or
  uninstalls any scheduled task on this machine. `bare-anchor.mjs` does not attempt to design
  its own scheduled-task wiring at all — that remains `install_anchor.cmd`/
  `install_anchor.ps1`'s job, untouched, and the owner's to migrate when the bridge exists.
- **A real corridor punch test between the founder role and a genuinely separate Bahrain-side
  machine was not attempted** (§3c is two local processes on this one machine, matching Gate
  G1's own literal wording — "probe runs clean on the founder machine in both roles" — not the
  full field ceremony, which per the corridor spec is explicitly not a coder task).

---

## 6. Not verified

- Whether `hypercore-crypto` (used by `kit-net.mjs`'s `roomTopic()`, not touched by this
  port) resolves under Bare — not needed by anything in this coder's scope, not checked.
- SIGINT/SIGTERM delivery to a real `bare.exe` process via `process.on('SIGINT', …)` under
  `bare-process` — attempted once this session via a Git-Bash `kill -INT`, inconclusive (the
  handler never fired within a 20s window, but Windows signal delivery through Git Bash's own
  `kill` emulation to a native Win32 process is itself an unreliable test method, not
  necessarily evidence against `bare-process`). **Not a gap unique to this port**: the
  ORIGINAL `anchor-spike.mjs` also never tests real OS signal delivery to `anchor.mjs`'s CLI —
  it tests `requestShutdown()` called directly, in-process, exactly as `bare-anchor-spike.mjs`
  does here. `bare-anchor.mjs`'s CLI wiring for `SIGINT`/`SIGTERM` → `requestShutdown()` exists
  in the same shape the original uses but is unexercised by any spike, old or new.
- `bare-tcp`/`bare-net`'s own behavior/reliability under Bare — present in the tree, never
  imported or exercised by any file in this coder's scope (§5).
- Whether the `#crypto`/`#fs`/`#apply` import-map requirement (§4's packaging landmine) affects
  any OTHER file already in the tree beyond this one — not audited beyond `bare-probe.mjs`
  itself; flagged as a build-bare-kit.mjs-relevant finding, not chased further.
- Live network conditions beyond this one machine/network (§5's corridor-test scope note).
- Whether `bare-anchor.mjs`'s resilience loop behaves correctly under REAL (not fixture) high
  churn — the retry-fixture spike exercises exactly 2 deliberate failures then a permanent
  recovery; sustained flapping (repeated fail/recover cycles) was not tested.
