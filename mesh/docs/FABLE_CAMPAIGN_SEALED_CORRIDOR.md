# Fable Campaign — The Sealed Corridor

**Status:** RATIFIED 2026-07-20 · **Orchestrator:** Opus 4.8 (autonomous wave)
**Coders:** Sonnet 5 · **Spec author & final gate:** Fable · **Owner:** the Commander
**Prior art (MANDATORY reading, in this order):**
1. `mesh/docs/bare-campaign/CAMPAIGN_REPORT.md` — the Sealed Ship's full record,
   **especially §4's five method rules and §5 (exit codes lie at three layers)**
2. `mesh/docs/FABLE_CAMPAIGN_BARE_RUNTIME.md` — doctrine D1–D6, inherited verbatim here
3. `mesh/docs/MISSION_A2_CORRIDOR_SPEC.md` — incl. A2.1 Reception-Grade addendum + FR-1/FR-1a
4. `mesh/kit/kit-host.mjs`, `kit-net.mjs`, `kit-registry.mjs`, `kit-repl.mjs` — the Node
   kit's SOLVED designs (identity, network, room persistence, onboarding ceremony)
5. `mesh/docs/bare-campaign/PHASE3_PROBE_ANCHOR_REPORT.md` — what the Bare probe proved
6. `mesh/docs/bare-campaign/PHASE4_CLEAN_MACHINE_VERIFICATION.md` — the verifier discipline

---

## 0. Charter

Give the **sealed Bare kit** the corridor: two sealed kits on two real machines
(India↔Bahrain class WAN) form one room — invite minted on one, redeemed on the other,
messages replicating both ways through the real reducer — with the same FR-1-immunity
the Sealed Ship proved: **no Node, no npm, no module resolution outside the folder.**

This is a PORT of proven design, not invention. The Node kit line already solved every
hard design problem this campaign touches:
- **Room persistence across runs:** `kit-registry.mjs` (the sealed guide's known gap —
  each run currently creates a fresh "kitchen table" instead of rediscovering).
- **The corridor network model:** `kit-net.mjs` — **hyperswarm primary + direct-TCP
  fallback** (MSG-D24). NOTE WELL: Holesail is NOT the kit transport and never was —
  it was `peer.mjs`'s era, and its optional import is literally what caused FR-1a.
  Do not reintroduce it here.
- **The onboarding ceremony:** `kit-repl.mjs`'s in-process `/addwriter` path (protocol
  v0 deliberately has no become-a-writer wire method — deviation #2 stands).
- **The client-facing ceremony UX:** A2.1 Reception-Grade guided path (START_HERE
  vocabulary, plain questions, paste-and-Enter, verdict words) — a REQUIREMENT on this
  campaign's guide copy, not a suggestion.

What the Sealed Ship already proved under Bare (re-verify, do not trust): DHT
bootstrap + two-process hole-punch (probe, 15/15, RTT 8 ms), the full local messenger
ceremony sealed (16/16 at the merge gate and on a clean machine), the reducer
byte-identical (13/13 both runtimes), spawn-pipe stdio discipline (executable rules).

**Rollback path, binding:** the Node A2.1 corridor kit zip stays ready on ceremony
day. If the sealed corridor is not gate-green by then, the ceremony runs on A2.1 and
this campaign continues without deadline pressure. A slipped gate is a report, not a
failure; a fudged gate is a failure.

## 1. Doctrine

D1–D6 from `FABLE_CAMPAIGN_BARE_RUNTIME.md` apply verbatim. Additionally, the five
method rules from `CAMPAIGN_REPORT.md` §4 are BINDING LAW here, most critically:

- **Rule 1 (verify the probe):** every harness proves it can report the opposite
  result before its verdicts count. Every new gate ships its negative control.
- **Rule 3 (test the layer the client touches):** final gates drive `run_bare_mesh.cmd`
  through a real spawned pipe, never `bare.exe app.bundle`, never a shell pipe.
- **Rule 5 (sample size IS the test):** any check where a hang/race is plausible runs
  N≥16; any measured *rate* uses N≥30. A pass at N≤5 is inadmissible.
- **Exit codes are inadmissible as health** (§5). Every assertion is on content.
- **`#` in any path breaks Bare addon resolution** (merge-gate finding 2026-07-20) —
  every harness that creates directories avoids `#`; the guide/verifier refusal stands.

**Stop-and-report (owner decisions, not judgment calls):** any new npm dependency
beyond `bare-tcp`; any protocol-v0 method addition/semantic change; any change to
`mesh/reducer/**`, capability/invite semantics, or the launcher's exit-code contract;
any temptation to reintroduce Holesail.

## 2. Missions

### SC-0 — Immersion & port map (orchestrator, before any code)
Read the prior-art list. Produce `bare-corridor/SC0_PORT_MAP.md`: for each of
`kit-registry.mjs`, `kit-net.mjs`, the `/addwriter` ceremony, and the A2.1 guide copy —
what ports verbatim, what needs a Bare-native substitute (named), what is out of scope.
Re-verify the probe's DHT claims on today's tree (run it, don't cite it).
**Gate SC-0:** port map exists, every Bare substitution names its package and its
prior-art precedent; probe re-run evidence attached.

### SC-1 — Rooms that survive the night (`kit-registry` port)
The sealed guide reopens its rooms on boot instead of creating "kitchen table" every
run. Port `kit-registry.mjs`'s load/reopen into `bare-guide.mjs`'s
`ensureMessengerCore()` via `#fs` (no `node:` specifiers — executable rule).
**Gate SC-1 (red-provable):** spike runs the sealed guide twice against one data dir:
run 1 creates + posts, run 2 must list the SAME room and read back run 1's message
(content-asserted). Negative control: point run 2 at a fresh data dir → must NOT find
the room. N≥16 on the reopen leg (a persistence race is plausible).

### SC-2 — The network leg under Bare (`kit-net` port)
Port `kit-net.mjs`'s model: hyperswarm primary, direct-TCP fallback (`bare-tcp` — the
one pre-approved new dependency), `--no-hyperswarm`-equivalent degradation for
no-internet LAN. Replication wire = the same Corestore replication streams the Node
kit uses. Every Holepunch package imported must already be in the sealed bundle's
dependency walk or offloaded by `bare-pack` — **the builder's wasm/addon hard gates
extend to any new native addon this mission pulls in** (broken-but-green is a known
shape; the builder must refuse on its own).
**Gate SC-2 (two-process, real sockets):** spike spawns TWO sealed-kit processes in
TWO hostile directories on one machine; a room replicates A→B and B→A over (a) the
TCP fallback with DHT disabled, and (b) the hyperswarm path. Content-asserted both
directions. Harness self-tests first (can it detect a non-replicating pair? prove it
with a firewalled/wrong-key negative control). N≥16 per path.

### SC-3 — The ceremony (guide wiring + Reception-Grade copy)
The sealed guide's menu grows the real corridor flow:
- Menu [2] path A ("start a room here"): existing behavior + **mint an invite** —
  display grouped-4 invite code (A2.1 formatting precedent in `formatCode`).
- Menu [2] path B ("join the other computer's room"): paste-and-Enter invite redeem →
  network connect (SC-2) → founder-side writer grant. The become-a-writer step is
  in-process on the founder's kit (deviation #2) — design the two-sided ceremony so
  each side's guide tells its human exactly what to do next, in A2.1's plain words.
- Menu [1] "Check the connection" stops being a stub: wire the Bare probe's proven
  DHT-bootstrap/punch checks with client-word verdicts ("this computer can reach the
  internet meeting point" / cannot, plus what to do).
**Gate SC-3:** scripted two-process ceremony through the REAL launchers: kit A mints,
kit B redeems (scripted stdin), A grants, B posts, A reads B's message back —
content-asserted end-to-end. Negative controls: wrong invite code must refuse with the
guide's own plain-language error; unreachable network must produce the honest failure
copy, not a hang (N≥16, hang-plausible by definition).

### SC-4 — Seal it, verify it, write it down
- New kit build (`--entry` unchanged if possible) passes ALL existing builder hard
  gates + any SC-2 addon extensions; sealed size/manifest recorded.
- `verify_clean_machine.cmd` gains an OPTIONAL corridor section (env-gated, off by
  default: two-machine checks can't run single-machine) — the single-machine 16×
  ceremony stays the default so the receptionist Round-2 protocol is unchanged.
- `CORRIDOR_RUNBOOK_SEALED.md`: the two-human ceremony script (who reads what aloud,
  what to photograph), SmartScreen/MOTW note (USB usually quiet, download prompts),
  the `#`-path warning, and the A2.1 rollback decision point stated plainly.
**Gate SC-4:** fresh build from clean checkout; full existing regression green
(stdioseam, seqrace, bridge Node+Bare, guide 17/17, parity both runtimes, go test);
sealed two-process corridor rehearsal green from hostile geography.

### SC-5 — FINAL GATE (Fable) then the field (owner-reserved)
Fable re-verifies independently (own driver, own negative controls, honest N) and
merges. THEN the owner runs: receptionist-machine clean verification (Round 2, already
queued) → two-machine LAN rehearsal → the India↔Bahrain ceremony. The orchestrator
does not touch the field steps; their runbook is the SC-4 deliverable.

## 3. Report discipline

One doc per mission under `mesh/docs/bare-corridor/` (SC0_PORT_MAP, SC1_…, etc.), same
honesty standard as the Sealed Ship's 24: every retraction stays visible, every gate
records its negative control and its N, every report states what was NOT verified.
Anything learned about Bare itself (new defects, workarounds) is written as an
executable rule where possible and flagged for the consolidated upstream filing.

*Port the proven, seal the port, prove the seal. 🐻*
