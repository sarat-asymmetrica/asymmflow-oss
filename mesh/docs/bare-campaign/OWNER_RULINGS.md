# Bare Runtime Campaign — Owner Rulings (pre-cleared gates)

**Ratified:** 2026-07-20 · **Owner:** the Commander · **Orchestrator:** Opus 4.8
Campaign spec: `mesh/docs/FABLE_CAMPAIGN_BARE_RUNTIME.md` §4 stop-and-ask triggers.

These rulings were obtained BEFORE Phase 0 completed, so the campaign can run to its
authorized endpoint without a mid-flight round-trip. Anything not covered here still
stops and asks.

## R1 — Reducer fork (triggers #1, #2): PRE-AUTHORIZED, gated on byte-identity

The `go:wasmexport` reactor path may be built without a further ruling, **provided all
three conditions hold**:

1. `mesh/reducer/**` (the fold law) is **untouched**. Mesh law stays frozen (D6).
2. The existing wasip1 **command module keeps building** and its spikes stay green
   (the rollback path stays warm).
3. **Every existing golden vector folds byte-identical** through the new channel —
   same ops in, byte-for-byte identical state out, versus the Node WASI host.

Any divergence on any of the three = STOP and report. The new entry point lives in a
NEW file (e.g. `mesh/cmd/reducer-reactor/main.go`); the old one is not edited.

*Rationale: a new entry point + host channel is packaging, not semantics. Byte-identity
on the goldens is the objective proof of that claim, not an assertion.*

## R2 — Sealed-artifact identity (trigger #3): BARE-AS-LIBRARY FIRST

Prefer shipping our own sealed artifact (bare runtime binary + prebuilt bundle +
embedded prebuilds) that we control end to end. The full **Pear runtime app model**
(end user installs Pear; app ships as a `pear://` key) is taken **only if Phase 0
proves the self-shipped path impossible** — and even then, the end-user cost is
reported to the owner **before** any code commits to it.

## R3 — Dependencies (trigger #4): BUILD-TIME & READ-ONLY REFERENCE PRE-AUTHORIZED

Pre-authorized without asking: anything that **never ships inside the sealed artifact** —
reference source ported from (e.g. Node's `lib/wasi.js`, a WASI reference impl),
build-time zip/self-extractor/packaging tooling. All such use is listed in the phase report.

NOT pre-authorized (still stop-and-ask): any dependency outside the Holepunch/`bare-*`
family that lands **inside the shipped artifact**, and reuse of the DP2 NSIS installer
toolchain (raise it as a proposal if Phase 3 wants it).

## R4 — Campaign endpoint: RUN TO THE PHASE 3 GATE

Autonomous authority runs through **Phase 3 inclusive**, ending with the hostile-machine
rehearsal (clean directory / no Node, no npm, no dev tooling — extract, double-click,
full ceremony) and an honest report plus the sealed kit.

**Phase 4 is owner-gated and NOT authorized in this run:** no anchor migration on the
owner's machine (no scheduled-task uninstall/install), no corridor field ceremony.
Those wait for the owner with the artifact in hand.

## Standing constraints (unchanged, restated)

- Branch `feat/fable-bare-runtime`; **no push** until the final gate.
- Synthetic identities only, everywhere (I4 / GL-12).
- Node-line spikes stay green throughout — it is the rollback path.
- Every gate that executes a built artifact runs it from **outside the repo tree** (D5).
- Every report states what was **NOT** verified.
- No changes to the OSS ERP; this campaign is mesh-side only (trigger #5 stands).
