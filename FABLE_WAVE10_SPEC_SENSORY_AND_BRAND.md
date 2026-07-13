# Wave 10 Spec — Sensory & Brand (The Owner's Wave)

**Mission:** Make AsymmFlow *feel* like it belongs to the people who run their business on it. Every flow wave made the app correct; this wave makes it theirs. The work is governed by DESIGN_CONSTITUTION.md Articles I (operator's language and rituals) and IV (the sensory budget) — both already law. This is the wave those articles were written for.
**Sequencing:** GATE SATISFIED (2026-07-13). Spec-07 (Tight Ship 2), Spec-08 (Residue Zero), and Spec-09 (Ecosystem Hardening) are all shipped and merged; the repo was re-squashed to one commit and pushed to `github.com/sarat-asymmetrica/asymmflow-oss` (private, pending owner eyeball → public flip — that flip is independent of this wave). The owner has declared the ship tight; this wave is GO.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave10-sensory-brand` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` (Articles I and IV are the mission itself) → `FABLE_WAVE9_UIUX_AUDIT.md` (keep-lists) → this spec.
**Prior art:** read all nine prior reports (`FABLE_WAVE9_SPEC_01..09_REPORT.md`). Spec-09's matters most here: it decomposed the two monster screens this wave will touch — WorkHub is now root + 7 components under `frontend/src/**/components/workhub/`, CustomerDetail is root + 9 components under `components/customer/` — so A1/A5/B1 targets are the decomposed children, not monoliths. Known cosmetic nit there: `modalTask` is derived twice (WorkHub root + task modal) — flows are frozen, so leave it unless B-work touches those exact lines anyway.

**The one-sentence brief:** a trader who has used this app for a month should be able to recognize it from across the room with the logo cropped out — by its timeline, its motion, its language, and the one sound it makes.

## 0. Read before anything

1. `DESIGN_CONSTITUTION.md` Article IV — the sensory budget is a BUDGET: responsiveness first, motion second, sound a distant third. **IV.3 allows exactly ONE sound in the entire application, reserved for a deal closing as PAID.** Article I — the deal-serial spine, domain nouns, and rituals (the six-document closing set) are the raw material of ownership.
2. `CLAUDE.md` — synthetic-data invariant. **Brand slots ship tokenized and synthetic** (product wordmark, accent, names); the flagship deployment applies its own identity as an overlay on its side. No client identity enters this repo.
3. The audit keep-lists (§4, all domains) and every Wave-9.x shipped behavior are binding. This wave layers feel ON TOP of flows; it never rewires one.

## 1. Operating model

Opus 4.8 orchestrator + Sonnet 5 coders, same as Waves 9.1–9.7 — with one addition: **this is the owner's wave, so taste calls are decision points, not improvisations.** For each item marked [TASTE], build the recommended variant fully, describe the alternatives considered in the report, and flag the choice for the owner's gate review. Never silently pick an aesthetic the owner can't see described.

**Lessons inherited (do not relearn):** anchors drift — recon first · `git checkout -- frontend/dist/index.html` after builds · gate baseline: vite clean, svelte-check 0 errors/14 warnings (confirmed still the baseline at Spec-09's final gate), `go build`/`go vet` clean, `go test -count=1 ./...` green (use `-timeout 1800s`; the main package is slow under load; 84 packages ok at last gate) · monster files get one coder · bindings regen is central · **binary artifacts freeze pre-scrub content** — if B5c's brand-slot work touches printed/PDF output, regenerate the artifacts and verify with `pdftotext`, never strings/grep · line endings are law: `.gitattributes` pins LF — do not fight it.

**Token-layer reality (found in pre-flight, feeds A2/B2):** there is no single `theme.css` at a canonical path — the tree has MULTIPLE candidate token files (`design-tokens.css` ×2, `phi-design-tokens.css`, `theme.css`, `wabi-sabi.css`, `app.css`). Article VI says one engine: A2 must name the ONE canonical file where motion tokens will live and report where the others defer to it (or flag the duplication — read-only verdict, no consolidation rewiring this wave beyond adding the motion tokens in the canonical spot).

## 2. Phase A — recon (read-only; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Interaction inventory: the canonical button/press components in `packages/` and `lib/components/ui` — what press/hover/focus/active states exist today, and where ad-hoc buttons bypass them. | B1 |
| A2 | Motion census: every CSS transition/animation in `frontend/src` (durations, easings, what they animate); does any `prefers-reduced-motion` handling exist; how modals/drawers/toasts enter and exit today. | B2 |
| A3 | The deal spine's data reality: for one deal, which linked records exist (RFQ → costing → offer → order → DN → invoice → receipt/payment) and via which foreign keys/serials; where a 360-view could already assemble the full chain server-side in ONE call vs N. What did Wave 9.x already build (customer 360 drills, serial deep-links, order traceability)? | B3 |
| A4 | Audio in Wails: can the webview play a bundled asset without an autoplay-policy block when triggered by a user's own click (the payment-posting action)? Prototype the smallest possible proof. Where user settings live (for the opt-out toggle). Context: Wails **v2.11** on WebView2 (Chromium) — Chromium's autoplay policy permits playback initiated by a user gesture, so a click-triggered `Audio.play()` should pass without flags; prove it, don't assume it. | B4 |
| A5 | Perceived-latency hot spots: the 5 slowest-feeling screen opens (data-load waterfalls, spinner-vs-skeleton usage, layout shift on load); what WabiSpinner does today. | B1 |
| A6 | Toast census: every `toast.*` call site — which CONFIRM a user action vs ANNOUNCE something the user didn't do (Article IV.4 violations); which are duplicated (toast + inline state for the same event). | B6 |
| A7 | Empty states + document rituals: the empty-state copy across screens (generic vs operator language); where the six-doc closing set (offer, order confirmation, DN, invoice, statement, receipt) is visible per deal today. | B5 |

## 3. Phase B — the sensory budget, spent in constitutional order

**B1 — Responsiveness first (Article IV.1 — the cheapest luxury).**
(a) Every canonical button gets a real press state: pressed transform + shade shift, ≤100ms, defined ONCE in the component/token layer, inherited everywhere. Ad-hoc buttons found in A1 are converged onto the canonical components (mechanical, behavior-preserving).
(b) Focus visibility: one focus-ring token, keyboard-visible on every interactive element.
(c) The A5 hot spots get skeletons (content-shaped placeholders) instead of centered spinners where the layout is known; zero layout shift on load for those five screens.
**AC:** pressing any button visibly responds within one frame; the five worst screens stop "popping" on load; no behavior change anywhere.

**B2 — One motion vocabulary (Article IV.2).** [TASTE]
Define motion tokens in theme.css: 2–3 durations (e.g. `--motion-fast` ~120ms, `--motion-base` ~200ms), 2 easings (standard + decelerate), used by ALL transitions. Modals/drawers/toasts get consistent enter/exit (fade+small translate — no bounces, no springs; this is a trading desk, not a game). Status-change moments (an offer marked Won, an invoice marked Paid) get one subtle settle transition. **`prefers-reduced-motion: reduce` disables every non-essential animation — verified, not assumed.**
**AC:** one grep finds every duration/easing at the token layer; reduced-motion renders the app fully static; nothing animates longer than ~250ms.

**B3 — The deal-spine timeline (Article I — THE signature).** [TASTE — direction RATIFIED by owner 2026-07-13: **UX simplicity above all.** Comprehension and legibility beat visual richness. Recommended shape: a compact single-row horizontal stepper — small state-colored dots/nodes connected by a thin rule, serial + date as quiet two-line labels beneath each node, generous type size, zero ornamentation. Document detail (the B5a checklist) lives BELOW or ON-DEMAND, never crammed into the nodes. On narrow widths it wraps or scrolls horizontally — it never shrinks text to fit. Build this variant; alternatives go in the Taste Ledger as usual.]
The ownership centerpiece: a horizontal deal timeline component rendering one deal's life as connected, serial-labeled stages — RFQ → Costing → Offer (rev N) → Order → Delivery → Invoice → PAID — each node showing its document serial + date, colored by state (done / current / pending / n-a), each node click-through to its record. Mounted where a deal is contemplated: the Order detail view and the Customer 360's deal rows (A3 tells you where assembly is cheapest; prefer ONE backend call returning the assembled chain). Missing links render honestly as gaps, never invented.
This is the thing a user sees every day that no competitor screenshot has. Build it as a canonical component with the design system's tokens — it should look inevitable, not decorated.
**AC:** for any order, one glance answers "where does this deal stand and what documents exist"; every node deep-links; chain assembly is one round-trip; renders correctly for partial chains (order with no DN yet, invoice unpaid).

**B4 — The one sound (Article IV.3 — the entire audio budget).** [TASTE — direction RATIFIED by owner 2026-07-13: **synthesize the candidate** (e.g. a small script generating a WAV/OGG — two-tone low settle, warm attack, fast decay, <1s, ≤50KB; no downloaded/licensed samples). Ship the synthesized asset + the generator script in `scripts/`; the owner re-rolls by tweaking generator parameters at gate review if the character is wrong.]
When a customer invoice transitions to PAID (receipt fully applied — the exact event A3/A6 pinpoint), play the single application sound: short (<1s), low, warm — a settle, not a fanfare. Requirements: bundled asset (embedded, no network); triggered only in the acting user's session by their own posting click (satisfies autoplay policy per A4's proof); a Settings toggle (default ON) — the opt-out is one click; **no other sound anywhere in the application, ever** — grep-verifiably one `Audio` construction in the codebase.
**AC:** closing a deal as paid is audibly acknowledged once; muting it is trivial; the audio budget audit (one call site) passes; no sound on any other event including errors.

**B5 — Rituals & operator language (Article I).**
(a) The six-document closing set becomes visible: on the deal timeline (B3) or order detail, a compact document checklist for the deal — offer / order confirmation / delivery note / invoice / statement entry / receipt — each present-or-missing, each a link. When the last document completes and the invoice is PAID, the set renders visibly complete (a quiet full-set state — this moment is where B4's sound already lives; no extra celebration animation beyond B2's settle transition).
(b) Empty states speak operator language (A7's census): "No RFQs yet — enquiries you log land here," not "No data found." Short, domain-noun copy, one optional action link. No illustrations, no mascots.
(c) The brand slot: ONE tokenized identity block (app wordmark text + accent color + optional mark) consumed by the sidebar header, login/lock surface, and printed/PDF headers — so a deployment re-skins identity by overriding tokens/config in ONE place. Ships synthetic ("AsymmFlow" / the AHS Trading document canon — see `ahs_branding_smoke_test.go`).

**Owner ratification 2026-07-13 — the flagship deployment's identity (build the slot to carry THIS, prove it, but do not commit it):** the first sovereign deployment re-skins to its own design language — a vivid fresh green accent (the company website's header green, roughly a saturated leaf green in the `#5CB550`–`#6ABF4B` neighborhood; sample the exact value from the owner-supplied screenshot at handoff), a grey-on-white wordmark with the green as the single accent, white/light surfaces, and a company mark replacing the default app identity in (i) the sidebar header, (ii) the login/lock surface, (iii) printed/PDF document headers, and (iv) the desktop app icon. Tone: **lightly personal, not a heavy re-theme** — the accent color, wordmark, and icon carry the identity; do NOT restyle components, charts, or semantic status colors to green. Deal-stage/status colors keep their existing semantic tokens (green-accent ≠ green-means-success collisions must be checked: if the accent lands near the existing success token, the two must remain distinguishable where they co-occur, e.g. the B3 timeline).

Mechanics (this is the acceptance test of the slot itself):
- **Web-layer identity** (wordmark text, mark SVG/PNG, accent token, PDF header block) = runtime config/token override in ONE documented place. Per repo law ("branding is configuration, not code") the shipped default stays synthetic; the wave PROVES the override path with a **throwaway, gitignored local override file** (any non-synthetic values used for the proof must never be committed or screenshotted into tracked docs).
- **Desktop app icon**: Wails v2 bakes `build/appicon.png` (and Windows `build/windows/icon.ico`) at build time — icon swap is therefore a per-deployment BUILD asset, not runtime config. Document the exact swap step in the same override doc so a deployment rebrand = one config file + one build-asset swap.
- Write the override doc as `docs/DEPLOYMENT_BRANDING.md`: every slot, its file/token, and the two-step rebrand recipe. The flagship's actual override file is authored on the deployment side (ph_holdings convergence workstream), not in this repo.
**AC:** a closed deal's full document set is verifiable at a glance; empty states read like a colleague wrote them; rebranding a deployment = one config/token override + one build-asset (icon) swap, zero source edits — proven live with a gitignored throwaway override, documented in `docs/DEPLOYMENT_BRANDING.md`, with no non-synthetic identity committed.

**B6 — Toast discipline (Article IV.4).**
Fix A6's violations: toasts CONFIRM what the user just did; they never ANNOUNCE background state (that's Article V's domain — route real alarms to the notification/approvals system, delete the rest). De-duplicate toast+inline pairs (keep the inline state). Success toasts get the B2 motion treatment; error toasts persist until dismissed (existing behavior — keep).
**AC:** every remaining toast call site is a direct echo of a user action; zero announce-class toasts.

## 4. Hard boundaries

- **Flows are frozen.** No behavior, routing, permission, or data changes — B3/B5a may ADD read-only assembly endpoints but never mutate. If a sensory item seems to require a flow change, it's out of scope: report it.
- **Financial semantics: stop-and-report. Zero authorizations this wave.** (B4's trigger LISTENS to the paid transition; it must not touch how the transition happens.)
- **The sound budget is one.** Not one-per-module. One. (IV.3.)
- **No new fonts, no new color families** — existing tokens + the motion/brand tokens this spec defines. Bundle-size discipline: the audio asset ≤50KB; no animation libraries (CSS + Svelte transitions only).
- **Accessibility is law:** reduced-motion fully honored; focus visibility everywhere; the sound has no information monopoly (the visual state change carries the meaning; the sound is garnish).
- **Keep-lists + all Wave 9.x behavior binding. No merge, no push, no tag.**

## 5. Definition of done + status report

Done = Phase A verdicts; B1–B6 shipped (each [TASTE] item with its variant rationale recorded for the owner's gate); gates green on the final commit; the three audits pass (motion tokens: one source; audio: one call site; toasts: zero announce-class).

Write `FABLE_WAVE10_SPEC_REPORT.md`, commit it, and paste it verbatim as your final message (established template + a **Taste Ledger**: every aesthetic decision taken, the alternatives considered, and what the owner should look at first when reviewing the branch). Severity honesty is law — and so is taste honesty: if something built ugly, say so; the owner would rather re-roll a component than ship embarrassment.
