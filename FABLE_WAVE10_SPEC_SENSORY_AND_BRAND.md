# Wave 10 Spec — Sensory & Brand (The Owner's Wave)

**Mission:** Make AsymmFlow *feel* like it belongs to the people who run their business on it. Every flow wave made the app correct; this wave makes it theirs. The work is governed by DESIGN_CONSTITUTION.md Articles I (operator's language and rituals) and IV (the sensory budget) — both already law. This is the wave those articles were written for.
**Sequencing:** runs AFTER Spec-07 (Tight Ship 2) and after the repo-hygiene/cloud-push milestone. Do not start this wave until the owner says the ship is tight.
**Repo:** `asymmflow-oss`. **Branch:** `feat/fable-wave10-sensory-brand` off `main`. Do not merge or push; leave for owner review.
**Authority documents, in order:** `CLAUDE.md` → `DESIGN_CONSTITUTION.md` (Articles I and IV are the mission itself) → `FABLE_WAVE9_UIUX_AUDIT.md` (keep-lists) → this spec.
**Prior art:** read the six prior reports (`FABLE_WAVE9_SPEC_01..06_REPORT.md`) plus Spec-07's when it exists.

**The one-sentence brief:** a trader who has used this app for a month should be able to recognize it from across the room with the logo cropped out — by its timeline, its motion, its language, and the one sound it makes.

## 0. Read before anything

1. `DESIGN_CONSTITUTION.md` Article IV — the sensory budget is a BUDGET: responsiveness first, motion second, sound a distant third. **IV.3 allows exactly ONE sound in the entire application, reserved for a deal closing as PAID.** Article I — the deal-serial spine, domain nouns, and rituals (the six-document closing set) are the raw material of ownership.
2. `CLAUDE.md` — synthetic-data invariant. **Brand slots ship tokenized and synthetic** (product wordmark, accent, names); the flagship deployment applies its own identity as an overlay on its side. No client identity enters this repo.
3. The audit keep-lists (§4, all domains) and every Wave-9.x shipped behavior are binding. This wave layers feel ON TOP of flows; it never rewires one.

## 1. Operating model

Opus 4.8 orchestrator + Sonnet 5 coders, same as Waves 9.1–9.7 — with one addition: **this is the owner's wave, so taste calls are decision points, not improvisations.** For each item marked [TASTE], build the recommended variant fully, describe the alternatives considered in the report, and flag the choice for the owner's gate review. Never silently pick an aesthetic the owner can't see described.

**Lessons inherited (do not relearn):** anchors drift — recon first · `git checkout -- frontend/dist/index.html` after builds · gate baseline: vite clean, svelte-check 0 errors/14 warnings, `go build`/`go vet` clean, `go test -count=1 ./...` green (use `-timeout 1800s`; the main package is slow under load) · monster files get one coder · bindings regen is central.

## 2. Phase A — recon (read-only; verdicts in the report)

| # | Question | Feeds |
|---|---|---|
| A1 | Interaction inventory: the canonical button/press components in `packages/` and `lib/components/ui` — what press/hover/focus/active states exist today, and where ad-hoc buttons bypass them. | B1 |
| A2 | Motion census: every CSS transition/animation in `frontend/src` (durations, easings, what they animate); does any `prefers-reduced-motion` handling exist; how modals/drawers/toasts enter and exit today. | B2 |
| A3 | The deal spine's data reality: for one deal, which linked records exist (RFQ → costing → offer → order → DN → invoice → receipt/payment) and via which foreign keys/serials; where a 360-view could already assemble the full chain server-side in ONE call vs N. What did Wave 9.x already build (customer 360 drills, serial deep-links, order traceability)? | B3 |
| A4 | Audio in Wails: can the webview play a bundled asset without an autoplay-policy block when triggered by a user's own click (the payment-posting action)? Prototype the smallest possible proof. Where user settings live (for the opt-out toggle). | B4 |
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

**B3 — The deal-spine timeline (Article I — THE signature).** [TASTE]
The ownership centerpiece: a horizontal deal timeline component rendering one deal's life as connected, serial-labeled stages — RFQ → Costing → Offer (rev N) → Order → Delivery → Invoice → PAID — each node showing its document serial + date, colored by state (done / current / pending / n-a), each node click-through to its record. Mounted where a deal is contemplated: the Order detail view and the Customer 360's deal rows (A3 tells you where assembly is cheapest; prefer ONE backend call returning the assembled chain). Missing links render honestly as gaps, never invented.
This is the thing a user sees every day that no competitor screenshot has. Build it as a canonical component with the design system's tokens — it should look inevitable, not decorated.
**AC:** for any order, one glance answers "where does this deal stand and what documents exist"; every node deep-links; chain assembly is one round-trip; renders correctly for partial chains (order with no DN yet, invoice unpaid).

**B4 — The one sound (Article IV.3 — the entire audio budget).** [TASTE]
When a customer invoice transitions to PAID (receipt fully applied — the exact event A3/A6 pinpoint), play the single application sound: short (<1s), low, warm — a settle, not a fanfare. Requirements: bundled asset (embedded, no network); triggered only in the acting user's session by their own posting click (satisfies autoplay policy per A4's proof); a Settings toggle (default ON) — the opt-out is one click; **no other sound anywhere in the application, ever** — grep-verifiably one `Audio` construction in the codebase.
**AC:** closing a deal as paid is audibly acknowledged once; muting it is trivial; the audio budget audit (one call site) passes; no sound on any other event including errors.

**B5 — Rituals & operator language (Article I).**
(a) The six-document closing set becomes visible: on the deal timeline (B3) or order detail, a compact document checklist for the deal — offer / order confirmation / delivery note / invoice / statement entry / receipt — each present-or-missing, each a link. When the last document completes and the invoice is PAID, the set renders visibly complete (a quiet full-set state — this moment is where B4's sound already lives; no extra celebration animation beyond B2's settle transition).
(b) Empty states speak operator language (A7's census): "No RFQs yet — enquiries you log land here," not "No data found." Short, domain-noun copy, one optional action link. No illustrations, no mascots.
(c) The brand slot: ONE tokenized identity block (app wordmark text + accent color + optional mark) consumed by the sidebar header, login/lock surface, and printed/PDF headers — so a deployment re-skins identity by overriding tokens/config in ONE place. Ships synthetic ("AsymmFlow").
**AC:** a closed deal's full document set is verifiable at a glance; empty states read like a colleague wrote them; rebranding a deployment = one config/token override, no source edits.

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
