# The Sealed Corridor — Campaign Report

**Date:** 2026-07-20 · **Branch:** `feat/fable-sealed-corridor` (off `750a39b`, not pushed)
**Orchestrator / technical lead / gate:** Opus 4.8 · **Coders:** Sonnet 5 ×4 · **Owner:** the Commander
**Prior line:** the Sealed Ship (`mesh/docs/bare-campaign/CAMPAIGN_REPORT.md`)

---

## 1. Verdict

**The charter is met.** Two **sealed** kits — no Node, no npm, no module
resolution outside the folder — extracted into two from-scratch directories,
each driven through its **real `run_bare_mesh.cmd` launcher**, form **one
room** and carry messages **both ways**, content-asserted in each direction,
through the unmodified reducer.

The room they share is **encrypted** (`hypercore-crypto` `randomBytes(32)`),
which no prior gate in this line had ever put on a wire.

`mesh/reducer/**` was never touched. Capability, invite and protocol-v0
semantics are unchanged. Holesail was never imported, referenced, or
considered as a transport — the ban held.

**What is NOT claimed:** every measurement here is two processes on ONE
machine. The India↔Bahrain field ceremony is owner-reserved and untouched, and
its runbook is the deliverable that hands it over.

## 2. What each mission delivered

| mission | deliverable | evidence |
|---|---|---|
| **SC-0** | port map; both packaging unknowns retired | probe re-run N=7 (6/7 AMBER, RTT 8–11 ms) with a 7/7-RED negative control; `bare-pack` proven to offload the corridor's native addons and run from hostile geography |
| **SC-1** | `bare-registry.mjs` + reopen in `ensureMessengerCore` | reopen across a real process restart 16/16; fresh-dir control 5/5; three malformed-registry controls |
| **SC-2** | `bare-net.mjs` (hyperswarm + `bare-tcp` fallback) | TCP replication both directions **16/16**; hyperswarm **11/16** measured, not gated; two controls run first |
| **SC-3a** | the two-sided ceremony in the guide | driven end-to-end through both real launchers (see §5) |
| **SC-3b** | menu [1] "Check the connection" is real | negative control 16/16 RED first; never-hangs 16/16; never-exits 5/5; probe self-test 15/15 unregressed |
| **SC-4** | sealed build gates, verifier section, field runbook | builder refuses on a missing addon (proven red-provable); verifier's corridor phase is opt-in; runbook written from a ceremony actually driven |
| **SC-5** | independent gate, own driver, own controls | §5 |

## 3. The six findings that mattered — none came from a passing test

**1. You could post, but you could not READ.** The messenger's only view was
`/rooms`, printing `last: <preview>` — the last message in canonical
`(Seq, Actor)` order, *not* the most recently received one. A message arriving
with a lower seq was **structurally invisible**, and that is not exotic: each
device seeds its seq counter from ops it has *seen*, so two people typing at
roughly the same time produce it routinely. It reproduced every round — the
founder posted seq 3 while the joiner had already posted seq 4. **Two people
on a corridor could not read each other**, in a mission whose entire point is a
two-way conversation. Found by driving the guide the way a person uses it and
asserting on what the SCREEN shows.

**2. `bare-tcp`'s `listen()` defaults its host to `localhost`** where
`node:net` defaults to all interfaces, and `kit-net.mjs` omits the argument.
A verbatim port would have shipped a LAN fallback **deaf to every machine
except itself** — 16/16 green on one machine, silently broken on two. Found by
reading the package source (`bare-tcp/index.js:545`), because **no
single-machine gate could ever have caught it.**

**3. A missing room folder does not throw.** Corestore silently creates a fresh
empty store, and with `bootstrap: null` Autobase founds a brand-new base under
a different key — so the guide would greet a **phantom empty room** as "found
your earlier conversation again." Worse than a caught exception. Found only
because the gate refused a mission that had marked itself complete with the
"never crash boot" contract inherited by reading rather than executed.

**4. `errno 112` is `ERROR_DISK_FULL`,** which Node surfaces as
`EINPROGRESS, unknown error`. It had already produced a careful, confident,
**wrong** conclusion (antivirus scanning large native binaries) that fit every
observation — including the one that looked most like evidence for it, a
trivial one-file copy succeeding while a 62 MB tree failed. That is capacity,
not AV. Cause: gate harnesses staging ~62–73 MB per run and leaking it; 41
abandoned kit directories totalling 2.37 GB were reclaimed, then 36 more.

**5. Every device called itself `guide`, so two machines could mint the SAME
message id.** `msgId` is `{actor}:{seq}` and each device computes its next seq
from its OWN local view, so two different physical machines both named `guide`
could independently land on the same id for two different messages. Reproduced
live: a founder's post came back `(not posted -- duplicate msgId "guide:4")`
after the joiner's post happened to take that seq. **This is silent message
loss on a real corridor** — the failure surfaces as a message that simply never
appears. It was invisible to every prior gate because a single device only ever
collides with itself; SC-3a was the first mission to put two real devices in
one room. Fixed by deriving each device's actor from its own public key
(`guide-${pubHex.slice(0,8)}`), which makes the collision structurally
impossible while keeping the identity stable across restarts.

> **Record correction:** this fix is committed inside `fb9dce8`, whose message
> describes only the `/read` gap. Four agents shared one working tree, and the
> coder's edit was swept into the orchestrator's commit — the SC-1 coder
> flagged this same pattern independently. The commit message is therefore
> incomplete, and rather than rewrite history the correction is recorded here,
> where the campaign's durable record lives.

**6. The guide double-spaced every line it printed, since Phase 2.** `write` is
`console.log` (and must stay — `stdout.write` hangs 30/30 on a real pipe),
which appends a newline to strings that already end in one. The entire menu
rendered with a blank line between every row. It survived because **every gate
in this campaign asserts with `.includes()` on substrings**, which is blind to
the blank lines between them. Green suite, broken-looking client surface.

## 4. The method lesson this campaign adds

The Sealed Ship's §4 ended at five rules. This wave earns a sixth, and it is
the through-line of all six findings above:

> **6. Assertions inherit the blind spots of the surface they read.** Every
> finding above was invisible to a green gate for the same structural reason:
> the assertion could not see the thing that was wrong. `.includes()` cannot
> see whitespace. `lastPreview` cannot see a lower-seq message. A
> single-machine socket test cannot see a `localhost` bind. A `catch` block
> cannot see a failure that never throws. Before trusting a green result, ask
> not "did it pass" but **"what could be broken and still produce this exact
> output?"**

Two corollaries earned the hard way this wave:

- **An empty-stdout HANG is evidence about the MACHINE, not the code.** A kit
  wedged mid-ceremony leaves PARTIAL output — it printed its menu before
  wedging. A process that never got scheduled leaves NONE. Re-run on a quiet
  machine before reporting it. (Both the SC-1 coder and the orchestrator
  reported this as a defect before isolating it.)
- **An asymmetric failure on a symmetric mechanism indicts the observer.**
  A→B failing while B→A passed was never a replication bug; it was the
  assertion.

## 5. The final gate (SC-5) — own driver, own controls

`kit/sealed-corridor-gate.mjs`, written by the orchestrator and deliberately
NOT built on any mission's spike: re-running the spike the coder wrote proves
the coder ran it, not that the thing works.

**It drives `run_bare_mesh.cmd`,** which every mission spike in the tree does
not — they drive `bare.exe app.bundle`, one layer below what a client
double-clicks. The launcher is not a formality: its `cd /d "%~dp0"` is what
puts the kit's CWD-relative `./data/...` inside the kit folder, and it is the
file historically mis-driven (three false failures in the prior campaign were
Git Bash mangling `cmd.exe /c` — reproduced again today).

**Controls run FIRST and gate everything else.** If any fails to go red the
driver refuses to report positive results at all, rather than noting it and
continuing.

**Final numbers** (2026-07-20, run at the merge gate by Fable after the
orchestrator's terminal crashed mid-collection; two full runs of the gate,
two full runs of the SC-3a spike):

| leg | result |
|---|---|
| negative controls (×3, run first) | all red as required, both runs |
| leg A — full ceremony through the REAL launcher | **16/16** (both runs; 0 hang, 0 partial) |
| leg B — persistence across a real restart | **16/16** (both runs) |
| leg C — six malformed-registry fixtures | **18/18** (both runs) |
| leg D — THE CORRIDOR, two sealed kits, both ways | **16/16** (run 2, deterministic path) |
| leg D unassisted-hyperswarm fraction (run 1, measured) | 11/16 — see finding 7 |
| SC-3a spike, LAN-assisted ceremony | run 1: 15/16 (cold-boot bound, isolated — SC3A_REPORT §3.3), run 2: **16/16** |
| SC-3a spike, hyperswarm-only (measured) | 3/5, 2/5 |

**Finding 7 — the gate's own leg D gated the wrong path** (recorded here
because it is finding class Rule-6, same as the other six, and it happened to
the INDEPENDENT gate itself): leg D's first version answered B's LAN prompt
with Enter — "keep waiting on hyperswarm" — which silently made the
campaign's decisive leg a hyperswarm-only test on a network SC-0 had already
characterised as firewalled/AMBER. It came back 11/16, every failure "B never
became writable", clean exits both sides — SC-2's measured swarm fraction,
reproduced to the digit, reported as a kit failure. The harness was gating
the ISP's firewall. Fixed at the merge gate: leg D now drives the
deterministic path the guide really offers (A's printed TCP port pasted at
B's LAN prompt, hyperswarm live alongside — the guide's real shape), and the
unassisted fraction is recorded above as the measurement it always was.
16/16 followed immediately. *Assertions inherit the blind spots — and the
environment — of the surface they read.*

## 6. Regression — unchanged suites, re-run

| suite | result |
|---|---|
| reducer parity, Node | 13/13 byte-identical |
| reducer parity, Bare | 13/13 byte-identical |
| bridge spike (protocol v0) | GREEN |
| stdio seam (both violations still proven load-bearing) | ALL GATES PASS |
| seq race | PASS |
| `go test ./mesh/...` | ok |
| `bare-guide-spike.mjs` | 17/17 |

## 7. Known gaps, stated plainly

- **No two-machine measurement.** Everything is two processes on one host.
- **hyperswarm at 11/16 on this network**, which SC-0 independently
  characterised as firewalled/AMBER. Reported, not gated; the TCP fallback
  exists for exactly this and is 16/16. Not a field-condition estimate.
- **No rate for the intermittent HANG.** Shown to be machine load, but both
  quiet-machine measurements were correctness proofs (N=3), not rates.
- **Menu [3]/[4] (anchor, status) and the firewall action remain honest
  stubs.** Unchanged from the Sealed Ship; they say so on screen.
- **The kit is not code-signed.** A SmartScreen prompt on a downloaded copy is
  expected, and the runbook says which button to press rather than promising
  there will be none.
- **No live-message stream.** You must type `/read`; messages do not appear on
  their own. The Node line's `createLiveStream` was not ported.
- **This machine has Node and npm installed.** "Ran outside `node_modules`" is
  geographic hermeticity, a weaker claim than a clean machine.

## 8. Owner-reserved, untouched

1. Receptionist-machine clean verification (Round 2).
2. Two-machine LAN rehearsal.
3. The India↔Bahrain ceremony — runbook at
   `mesh/docs/bare-corridor/CORRIDOR_RUNBOOK_SEALED.md`.
4. The A2.1 rollback decision, which the runbook states is made BEFORE
   ceremony day: if this kit is not gate-green, the ceremony runs on the Node
   A2.1 kit and this campaign continues without deadline pressure. **A slipped
   gate is a report; a fudged gate is a failure.**

---

*Port the proven, seal the port, prove the seal.* 🐻
