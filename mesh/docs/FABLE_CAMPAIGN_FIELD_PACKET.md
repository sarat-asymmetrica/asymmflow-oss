# Fable Campaign — The Field Packet

**Status:** DRAFT (owner ratification pending) · **Orchestrator:** Opus 4.8 (autonomous wave)
**Coders:** Sonnet 5 (or orchestrator solo — this is a small wave) · **Spec author & final gate:** Fable
**Owner:** the Commander
**Prior art (MANDATORY reading, in this order):**
1. `mesh/docs/bare-corridor/CORRIDOR_RUNBOOK_SEALED.md` — THE ceremony script. The packet
   RENDERS this; it does not reinvent it.
2. `mesh/docs/bare-campaign/PHASE4_CLEAN_MACHINE_VERIFICATION.md` — Round 1 evidence +
   the Round-2 (receptionist machine) protocol this campaign stages.
3. `mesh/docs/bare-campaign/CAMPAIGN_REPORT.md` §4 (five method rules) + §5 (exit codes
   lie at three layers) — binding law, inherited verbatim.
4. `mesh/docs/FABLE_CAMPAIGN_SEALED_CORRIDOR.md` — what the sealed kit is and how it was
   gated; SC-4/SC-5 are the green lights this campaign packages.
5. `mesh/kit/build-bare-kit.mjs` (or the builder's current name — locate it, read its
   hard gates) — the ONLY sanctioned way to produce the kit this campaign ships.
6. Aesthetic reference (LOCAL, may live outside the repo — if absent, STOP and ask):
   `C:\Projects\git_versions\asymm_all_math\asymm_mathematical_organism\design_system\showcase\show_and_sell\asymmetrica-capabilities.html`

---

## 0. Charter

The Sealed Corridor is gate-green and merged. What stands between the code and the
field is **logistics**: there is no shipping zip of the sealed kit, and the humans who
will run the ceremony have a 229-line engineer-voiced runbook instead of something a
receptionist can hold. This campaign closes that gap with three artifacts:

1. **The zip** — a fresh, drive-through-verified sealed-kit field zip cut from today's
   main, ready for a USB stick.
2. **The packet** — a single self-contained HTML (print-to-PDF clean) that renders the
   runbook's ceremony + the Round-2 verifier protocol in client-grade language and the
   Asymmetrica capabilities-page aesthetic.
3. **The staging** — everything laid out so the owner's only remaining moves are:
   copy folder to USB, schedule the call, run the ceremony.

**This is a PACKAGING campaign. It writes zero runtime code.** The sealed kit, the
guide, the verifier, and the reducer are gate-green and FROZEN for this wave. If any
mission appears to require touching `mesh/host/**`, `mesh/kit/*.mjs`, `mesh/reducer/**`,
or the launcher — **stop and report**. A packaging wave that "just fixes one thing" in
a gated runtime is how fudged gates are born.

**Rollback path, binding (inherited from SC-5):** the Node A2.1 corridor kit zip
(`AsymmFlow-Corridor-MachineB.zip`, 103.4 MB, cut 2026-07-19) stays warm and untouched.
If the sealed field zip is not gate-green by ceremony day, the ceremony runs on A2.1.
A slipped gate is a report, not a failure; a fudged gate is a failure.

## 1. Doctrine

The five method rules of `CAMPAIGN_REPORT.md` §4 are BINDING LAW, most critically here:

- **Rule 1 (verify the probe):** every check in this campaign proves it can report the
  opposite result before its verdicts count. The drive-through gate ships a negative
  control (e.g. a deliberately corrupted copy must FAIL the verifier).
- **Rule 3 (test the layer the client touches):** the zip gate extracts the actual zip
  with Windows-native tooling (`Expand-Archive`, not a repo copy) to a from-scratch
  directory OUTSIDE the repo tree and drives `run_bare_mesh.cmd` / 
  `verify_clean_machine.cmd` — never `bare.exe app.bundle` directly. Geography-hermetic:
  the FR-1 lesson (in-repo gates let module resolution escape to `mesh/node_modules`)
  applies to zips verbatim.
- **Rule 5 (sample size IS the test):** the extracted-kit ceremony check runs the
  verifier's own 16× loop. A single smoke run is inadmissible.
- **Exit codes are inadmissible as health.** Every assertion is on content
  (`VERIFY_EVIDENCE.txt` verdict line, tally line `OK=16/16`, probe-control red line).
- **`#` in any path breaks Bare addon resolution.** Every staging/extraction directory
  this campaign creates avoids `#`; the packet WARNS about it in client words (the
  runbook §3b copy is the precedent).
- **CRLF law:** every `.cmd`/`.bat` inside the zip is CRLF; assert it byte-level in the
  gate (A2 merge-gate finding — bare-LF `.cmd` misparses).
- **Synthetic data only, real names NEVER:** the packet and every screenshot/example in
  it uses roles ("the receptionist", "the founder", "the person in India") and made-up
  message text. No real person's name, no real company detail beyond what is already
  public in the repo. The repo is PUBLIC; the packet ships inside it.
- **The invite code is a bearer secret:** the packet must carry the runbook's
  photograph-discipline verbatim (photograph screens and `/read` outputs; NEVER the
  long invite code).

**Stop-and-report (owner decisions, not judgment calls):** any runtime-code edit (see
Charter); any new dependency; any change to the runbook's ceremony semantics (the
packet REPHRASES, it does not REDESIGN); code-signing/Authenticode (deferred at DP2 §9
— the packet documents the SmartScreen reality instead); publishing the packet anywhere
outside the repo.

## 2. Missions

### FP-0 — Immersion & inventory (orchestrator, before any artifact)
Read the prior-art list. Then inventory the ground truth on today's main:
- Locate the sealed-kit builder and its hard gates; confirm the kit build inputs
  (`app.bundle` entry, offloaded addons, `dist/reducer.wasm`) are unchanged since the
  SC-5 gate (git log on `mesh/` since `0fd87b0` — if ANYTHING under `mesh/` changed,
  say so and re-run the affected spikes before building).
- Enumerate the current `mesh/kit/dist-bare/` manifest vs. the Sealed Ship's recorded
  24-file manifest; explain every difference or confirm identity.
- Confirm the aesthetic reference file exists and extract its design tokens (palette,
  fonts, card/badge/quote idioms) into the FP-2 brief.
**Gate FP-0:** written inventory in `bare-corridor/FP0_INVENTORY.md`; any `mesh/` drift
since the corridor merge is listed with a verdict (benign / needs-spike-rerun).

### FP-1 — Cut the zip (the shipping artifact)
- **Fresh build** via the sanctioned builder from today's main — never zip the possibly
  stale `dist-bare/` as found. The builder's own hard gates (wasm offload, native-addon
  refusal) must run and pass.
- Zip layout: kit files at the ZIP ROOT (double-click `Extract All` must yield a folder
  where `run_bare_mesh.cmd` is one level down at most — no nested `dist-bare/dist-bare`).
- Name: `AsymmFlow-SealedKit-Field-YYYYMMDD.zip`, cut to `C:\Projects\asymmflow\`
  (OUTSIDE the repo — the zip is a shipping artifact, not a commit; verify it is not
  in any tracked path / is gitignored).
- Record: file count, total size, sha256 of the zip, sha256 of `bare.exe`,
  `app.bundle`, and `dist/reducer.wasm` inside it.
**Gate FP-1 (the drive-through, red-provable):**
1. `Expand-Archive` the actual zip to a from-scratch `#`-free directory outside the
   repo tree (e.g. `%TEMP%\fp1-drive-through\`).
2. Run the extracted kit's own `verify_clean_machine.cmd`; assert on CONTENT:
   the tally line reports `OK=16/16 HANG=0/16 CONTENT_FAIL=0/16`, the probe control
   went red, and `VERIFY_EVIDENCE.txt` exists. (The dev machine will honestly report
   NOT CLEAN for the Node-free claim — that line is environmental, expected, and must
   be recorded, not suppressed. The ceremony tally is the gate.)
3. CRLF-assert every `.cmd` in the extraction, byte-level.
4. **Negative control:** repeat the extraction, deliberately corrupt `app.bundle`
   (truncate or flip bytes), re-run the verifier → it must go RED. A drive-through
   that cannot fail proves nothing.
5. Confirm extraction into a path CONTAINING `#` produces the kit's loud refusal
   (the verifier's own check), not a confusing crash.

### FP-2 — The packet (client-grade HTML, capabilities-page aesthetic)
One self-contained HTML: `mesh/docs/bare-corridor/FIELD_PACKET.html`.

**Content (three chapters, all sourced from prior art — rephrase, never redesign):**
1. **"Before the call"** — getting the folder on the machine: USB-first guidance,
   MOTW/SmartScreen reality in honest client words (runbook §3a verbatim in spirit:
   name the exact warning and the exact button; never promise "no warnings"), the
   `#`-path rule, synthetic-data-only rule.
2. **"The health check"** (Round 2 protocol) — double-click `verify_clean_machine.cmd`,
   what the screen should say (render the Round-1 PASS block as the exemplar), what to
   send back (`VERIFY_EVIDENCE.txt` + `verify-logs\`), and the Phase-4 field notes
   (Defender first-run slowness is not a failure; window-flash diagnosis line).
3. **"The conversation"** (the ceremony) — the runbook §4 two-column ceremony:
   STARTING person's steps and JOINING person's steps side by side, the both-ways
   proof (§4c, `/read` law with its one-sentence why), what to photograph (§5,
   including the never-photograph-the-invite rule), and what "it worked" means (§6:
   content both directions; a closed window and a "Goodbye" prove nothing).

**Aesthetic (from the reference, tokens extracted at FP-0):** warm paper background
(`#FAF9F7` family), Spectral light serif for headings, DM Sans body, JetBrains Mono
for anything the human will literally see on a screen (`run_bare_mesh.cmd`, the tally
line), deep-teal + gold accent system, pill badges for step states, card grid for the
two ceremony roles, quote-block idiom for the honesty notes. **Aesthetic only** — no
mode-toggle, no progress bar, no JS beyond (optionally) zero-dependency
collapse/expand; the page must read complete with JS disabled.

**Print discipline:** an `@media print` pass — sensible page breaks between chapters
(`break-inside: avoid` on ceremony cards), no hover-dependent content, dark-on-light
throughout, footer with kit version + zip sha256 (short) + date. Fonts via the same
Google Fonts link as the reference WITH honest local fallbacks (Georgia/system-ui) —
the packet must remain fully legible offline; print-to-PDF on the connected dev
machine bakes the real fonts in.

**Gate FP-2:** (a) content audit — every instruction in the packet traces to a runbook
/ Phase-4 section (list the mapping; anything invented = finding); (b) zero real names
grep (case-insensitive, against the known-names scrub list used in OSS hygiene —
roles only); (c) opens correctly file:// offline (no broken layout without network);
(d) print-to-PDF produced and eyeballed: no orphaned headings, no clipped cards, both
ceremony columns intact; PDF cut to `C:\Projects\asymmflow\` next to the zip.

### FP-3 — Field staging & the handover note (dovetail into the ceremony)
- Stage a single folder `C:\Projects\asymmflow\FIELD_KIT_STAGING\`: the FP-1 zip, the
  FP-2 PDF, and a one-screen `README_OWNER.txt` — the owner's own checklist (copy zip
  to USB → receptionist machine → extract to Desktop or `C:\corridor\` → verifier
  first → evidence comes back → then the ceremony call). Owner-facing, so plain text
  is correct here; the PACKET is the client-facing pretty layer.
- Pre-author the results ledger entry: `mesh/docs/MESSENGER_DECISIONS.md` gains a
  DRAFT entry (next MSG-D number — read the file, don't assume) titled "Field results:
  receptionist Round-2 + first India↔Bahrain sealed corridor", with empty evidence
  slots (Round-2 verdict, tally, vcruntime line, SmartScreen behavior observed,
  ceremony both-ways proof, photographs index). The field fills slots; nobody
  re-derives the schema on ceremony day. Mark it DRAFT — it is not a decision until
  the field signs it.
- The SmartScreen observation slot explicitly feeds the deferred DP2 §9 Authenticode
  decision — say so in the entry.
**Gate FP-3:** staging folder complete and self-explaining (a person who has read
NOTHING can follow README_OWNER.txt); ledger draft committed; no tracked-path
pollution by zips/PDFs (git status clean of artifacts).

### FP-4 — FINAL GATE (Fable), then the field (owner-reserved)
Fable re-verifies independently: own extraction of the actual zip in own hostile
directory, own verifier run with own negative control, packet content audit + real-name
grep re-run, print-to-PDF re-cut. Then merge.

**THEN the owner runs, in order (the orchestrator does not touch these):**
1. Round-2 clean-machine verification on the receptionist machine (evidence returns).
2. Two-machine LAN rehearsal (owner-side, per SC-5 reservation).
3. The India↔Bahrain ceremony with the field contact on the phone — packet in hand,
   runbook §2 go/no-go honored, A2.1 zip on standby.
4. Results into the FP-3 ledger draft; MSG-D entry ratified.

## 3. Report discipline

One doc per mission under `mesh/docs/bare-corridor/` (`FP0_INVENTORY.md`,
`FP1_ZIP_REPORT.md`, `FP2_PACKET_REPORT.md`, `FP3_STAGING_REPORT.md`), same honesty
standard as the Sealed Ship's 24: every gate records its negative control and its N,
every report states what was NOT verified (this campaign inherits one standing
un-verifiable: the dev machine cannot prove the Node-free claim — only the
receptionist machine can, and that is the point of the field). Retractions stay
visible. Anything learned about zip/MOTW/SmartScreen behavior is recorded for the
DP2 §9 Authenticode file.

*Pack the proven, prove the pack, hand it over. 🐻📦*
