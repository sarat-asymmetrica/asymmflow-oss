# AGENT GATE LEDGER — Messenger track

**Purpose:** the technical lead's quality gate findings on subagent-built missions.
Every divergence between expected quality and delivered work is recorded here so that
*every future agent* (any model, any session) inherits the standard without having to
rediscover it. Read this BEFORE coding on the messenger track; it is as binding as
`MESSENGER_DECISIONS.md`.

**Methodology:** orchestrator/technical-lead (senior model) writes mission briefs →
coder agents build → lead reviews the full diff, runs gates independently, fixes or
bounces divergences → records findings here → commits. Coder agents never commit.

Entry format: `GL-<n> [mission] — <finding>` with **Pattern to repeat** or
**Anti-pattern to avoid**.

---

## Standing standards (distilled for all coder agents)

- **Read the installed source, not the README.** Holepunch-ecosystem docs lag their
  code (proven in M4: blind-peering's README named a stale option). Verify every
  third-party API against `node_modules/<pkg>/index.js` before using it.
- **Go and JS signable/kind logic must be character-for-character semantically
  identical** — including length guards (`len > 4`, `len > 7` style). A pathological
  kind string that behaves differently across the boundary is a determinism landmine.
- **Never pass a key array as a `JSON.stringify` replacer** (it filters recursively —
  gutted a nested blob locator in M3). Pin insertion order in the object literal.
- **Skipped vs rejected taxonomy is law:** chat-domain rule failures → `skipped[]`
  (typed reason); capability/kernel law → `rejected[]`. Never crash the fold.
- **Golden discipline:** legacy goldens are byte-frozen; only goldens explicitly
  authorized by the mission brief may regenerate, and every regeneration is proven
  reproducible by a second run without the update flag.
- **Report honestly:** verbatim gate output, deviations with justification,
  uncertainty called out. An agent that says "I wasn't sure, I chose X because Y"
  is doing it right.

## Findings

**GL-1 [W-mirror-2 Mission 1: vocabulary] — Spec-gap surfacing is the standard; silent literalism is not.**
The coder implemented the brief's claim rule literally (non-authority may only
self-claim), noticed the consequence — a member could never voluntarily release
their own claim — implemented it AS SPECIFIED, and FLAGGED the product-behavior
consequence prominently in the report instead of either silently shipping it or
silently "fixing" it. **Pattern to repeat:** when the spec's letter produces a
behavior the spec's *intent* (here, Constitution Art. VI) never discussed, build
the letter, flag the gap, let the gate rule. The gate ruled: members may release
their own standing claim ("may only release own claim" skip otherwise); lead
implemented + tested + updated MSG-D17. Also repeated-worthy: the report's
honest fallback note on cross-boundary signable verification (no direct
Go-verifies-JS-bytes unit pattern exists; the spike IS the practical proof — a
field-list mismatch fails universally, not subtly). **Anti-pattern (minor):**
new decision entries were inserted mid-file "to sit near D15" — decision docs
append at the literal END, chronology beats visual adjacency; lead relocated
D16/D17.

**GL-2 [W-mirror-2 Mission 1 gate] — A view golden on a concurrent fork is a coin-flip; pin STATE, assert convergence.**
The attach spike went RED intermittently at the lead's gate re-run (agent's
2-run reproducibility had passed): the scenario forks two writers
CONCURRENTLY (both append while disconnected), so Autobase may legitimately
linearize the heads in either order — the VIEW digest is not run-deterministic
there, and the golden had pinned it since M3 (pre-existing latent flake, not
the agent's regression; their run protocol just couldn't catch it).
**Standard, now law for all spikes:** a view-digest golden is only valid when
appends are causally chained (single writer, or barriered between writers);
a genuinely-forked scenario pins the STATE digest + deep state projection
(order-independent by construction — the canonical fold is the point) and
asserts the view is CONVERGED across peers, not what order it converged to.
**And:** reproducibility proof for any scenario containing real concurrency =
3+ runs, not 2. **Watch-class:** room-spike (lines ~109-111) and invite-spike
share the concurrent-fork structure and still pin view digests; stable across
dozens of runs so far — if either EVER flakes, apply this fix, don't debug.

**GL-3 [W-mirror-2 Mission 2: encryption] — Source-citation discipline at its best; the report IS part of the deliverable.**
**Patterns to repeat, all three:** (a) every third-party API claim carried a
file+line citation from the INSTALLED source, read BEFORE the README was
cross-checked — including the load-bearing find (`ViewStore.getEncryption`,
store.js:246-252) proving named view cores encrypt too, and the negative
finding (no content-key rotation API exists; boot.js:104-113 reuses the
persisted key unconditionally). (b) When the mirror golden regenerated
BYTE-IDENTICAL under encryption, the coder didn't let it silently look like
a missed step — it documented WHY (goldens pin decoded values; encryption is
a storage/transport property, invisible to a keyed peer). Explaining an
absent diff is as important as explaining a present one. (c) The
rotate-on-revoke task was investigate-and-report; the coder resisted
building any of its three options and ranked them honestly, including
labeling its own ranking as judgment rather than fact. **Anti-pattern:** the
agent went idle WITHOUT sending its final report and had to be pinged — the
code was done, the mission wasn't. The report is a first-class deliverable:
a gate cannot run on an unexplained diff, and an idle agent with undelivered
findings is indistinguishable from a crashed one. Deliver the report, THEN
go idle. **Gate outcome:** zero code changes required by the lead — first
mission this wave to pass the gate untouched; rotation ruled (room re-issue
doctrine, MSG-D18 addendum) from the coder's findings.

**GL-4 [W-mirror-2 Mission 3: evidence export] — The ledger works: GL-3's lesson arrived pre-installed.**
Second consecutive zero-fix gate. **Patterns to repeat:** (a) the report
arrived BEFORE the agent idled — GL-3, read at mission start, changed
behavior without the lead saying a word; this is the ledger doing its job
and the reason it exists. (b) The one out-of-scope file change (a read-only
`node.authorityPub` getter on mesh-node.mjs) was minimal, additive,
justified in the report, and proven harmless by the full regression floor —
the right way to handle a brief that didn't anticipate a missing accessor:
smallest possible change, loudly declared. (c) The golden was scoped to what
the mission actually proves (bundle reproducibility: stateDigest +
viewDigest + bundleSha256) instead of duplicating the state-shape golden
that room_autobase.json already owns — golden MINIMALISM is a virtue;
overlapping goldens rot in pairs. (d) The spike's forge case asserts the
subtle thing (the attacker's self-consistent signature PASSES sig-check and
only the capability-plane refold catches it) — testing the mechanism's
limits, not just its successes. **Confirmed on review (the coder's flagged
uncertainty):** `createMeshNode` with no `authorityPub` IS the correct
social-room shape (unenforced fold, no admin in the room — Art. II / MSG-D17
vocabulary); the from-scratch construction was right.

**GL-5 [W-human-3 Mission A: room re-issue] — Distrust your own passing test; the ambiguous green is the dangerous one.**
Third consecutive zero-fix gate on the coder's code. **Pattern to repeat,
the headline:** the coder's first wrong-key probe PASSED (`ops.length === 0`)
and it flagged its own green as ambiguous — "can't decrypt" and "never
replicated anything" produce the same empty result — then rebuilt the probe
to force a real block over a real replication stream and assert both
arrival AND opacity separately. A test that cannot distinguish success from
vacuous success is not a test; interrogating your own green is senior
behavior. Also repeat-worthy: (a) the self-caught seq-restart bug (canonical
order sorts by Seq first — a successor room's ops must CONTINUE the
ceremony's counter, not restart at 1; inline-commented so it can't
regress silently); (b) observed-vs-contract honesty in probe B (Autobase's
fail-closed empty view reported as an observation of this version, not
asserted as an API guarantee); (c) the honest-envelope framing of reusing
`asymm-room2.` codes as pure key transport when the grant is already
direct — reusing one proven bearer-string format beats inventing a second.
**Gate adoptions:** the coder's flagged what-about-garbage-pointers idea
became `TestManifestPredecessorNoValidationByDesign` — pinning the ABSENCE
of validation as law so a future "helpful" shape-check reads as a
deliberate change; its in-process-hermetic interpretation (no DHT/mirror
when the mission proves ceremony+crypto, not delivery) was accepted as
correct scope discipline.
