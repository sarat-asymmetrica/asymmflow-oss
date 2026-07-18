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
