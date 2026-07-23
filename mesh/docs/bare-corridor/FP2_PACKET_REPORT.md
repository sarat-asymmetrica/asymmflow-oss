# FP-2 — The Packet · Report

**Date:** 2026-07-23 · **Coder:** Sonnet 5 (packet draft) · **Gate:** Opus 4.8
orchestrator · **Campaign:** Field Packet (`f62ef37`)

## 1. Deliverable

`mesh/docs/bare-corridor/FIELD_PACKET.html` — one self-contained file, three
chapters (**Before the call · The health check · The conversation**), zero
`<script>` tags of any kind (reads complete with JS disabled by construction),
only external reference is the Google Fonts link with honest Georgia/system-ui
fallbacks. Capabilities-page aesthetic per the FP-0 token extract: warm paper,
Spectral light headings, DM Sans body, JetBrains Mono for everything the human
literally sees on a screen, teal/gold accents, pill badges, card grid for the
two ceremony roles, gold quote-blocks for the honesty notes.

The PDF (`FIELD_PACKET.pdf`, 445,104 bytes, sha256 `A8278FD8887B4BB1CA95…`)
was cut headless-Chrome from the final HTML to `C:\Projects\asymmflow\` next
to the zip, and copied into `FIELD_KIT_STAGING\`.

## 2. Gate FP-2 results

**(a) Content audit — traceability.** Every instruction and factual claim maps
to a source section (runbook §3a/§3b/§3c/§4/§4a/§4b/§4c/§5/§6/§7 + PHASE4
Round-1 block and Round-2 field notes). Mapping table produced by the coder and
spot-verified by the gate against the runbook line by line for the ceremony
steps (step-for-step faithful, no merged/reordered/dropped steps).
**Invented-claims list: EMPTY.** One deliberate rephrase recorded: the
window-flash field note gives the client-facing instruction (photograph +
report) and omits the support-engineer terminal diagnostic — an audience
scoping, not a semantic change. The Round-1 PASS block renders verbatim with
its honest caption (Windows Sandbox, 2026-07-20).

**(b) Zero real names.** First instrument (Git Bash `grep -i`) ABORTED —
under Rule 1 a crashing probe proves nothing, so the gate switched instruments
entirely (PowerShell regex) and re-proved the probe: positive control with two
planted names hit 2/2, then the packet scanned clean **0 hits across the
12-pattern scrub list** (list held in session scratchpad only — the repo is
public, so the list itself is never committed). Roles only throughout.

**(c) Offline / file://.** Loaded with ALL network requests force-aborted
(fonts.googleapis.com + fonts.gstatic.com blocked at the browser level):
fully legible on fallback fonts, layout intact, nothing hover-dependent.

**(d) Print-to-PDF, eyeballed page by page (10 pages).** Each chapter starts
on its own page; the two ceremony role-columns render side by side, complete
and unclipped across their spread; no orphaned headings; dark screen-blocks
print with their backgrounds (`print-color-adjust: exact`); footer carries
kit line + zip sha256 (first 18) + date.

## 3. Findings fixed at gate (both by the gate, not the coder)

1. **Stale footer hash (gate's own brief at fault):** the coder was briefed
   with the FIRST zip's sha256; the shipped zip was re-cut after FP1-1's
   verifier fix. Footer corrected to `130BF121870BF47CA9`, traceability note
   updated.
2. **Print background artifact:** `@media print` overrode `body` but not
   `html`, so the warm page background printed as a beige block below the
   footer on the last page. Fixed (`html, body{background:#fff}`), PDF re-cut,
   last page re-eyeballed clean.

## 4. Not verified, stated plainly

- The PDF was cut and eyeballed on the dev machine's Chrome; a different
  PDF reader/printer may paginate differently. FP-4 re-cuts and re-eyeballs
  independently.
- The packet's client language has not yet been read by an actual
  non-technical human — the receptionist Round 2 and the ceremony are the
  real test of the words, and their observations feed back here.

**Gate FP-2: PASS** (with the two gate-fixed findings recorded above).

*Pack the proven, prove the pack, hand it over.* 🐻📦
