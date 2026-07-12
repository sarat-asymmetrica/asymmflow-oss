# AsymmFlow Design Constitution

**Status:** LAW — ratified by the owner 2026-07-10. Binding on every agent and every wave from Wave 9 onward.
**Companions:** `FABLE_WAVE9_UIUX_AUDIT.md` (evidence + keep-lists), `CLAUDE.md` (repo invariants).

This document exists so that many independent build sessions produce **one application**. Wave specs are deliberately thin because they inherit this. When a spec and the constitution conflict, the constitution wins; when the constitution must bend, that is an **amendment** — flag it in the wave report and get owner sign-off, don't improvise.

Every article states the law, the canonical implementation to copy, and the named anti-pattern to reject in review. Cross-references like **T1**/**pattern #2** point into the audit.

---

## Article I — The app speaks the operator's language

The deepest form of ownership is recognition: a trading team should look at any screen and see *their* vocabulary, rituals, and geography — not a database's.

1. **Domain nouns, not schema nouns.** RFQ, Offer, REV-2, Costing, DN, GRN — the words the team says on the phone. Headline identifiers are never raw ids; the human serial leads.
2. **The deal serial is the spine.** Every business object surfaces its serial (`EH-24-26 ACME FIT` grammar — prefix, sequence, year, customer, instrument token), search resolves it first, and every generated document (quote, DN, invoice) carries it. The app mints serials; users never hand-type them.
3. **Status is a vocabulary, not free text.** Lifecycles are fixed named stages with named transitions. No free-text status fields, ever — the source data showed 123 spellings of ~5 states. (Enforced structurally by Article III.3.)
4. **Place anchors.** BHD with 3-decimal fils everywhere money renders; Sun–Thu working week in any date logic that cares; tabular figures for money columns.
5. **Job-shaped hierarchy.** A screen opens on the question its user came to answer. *Anti-patterns (audit T11): sales metrics headlining the HR profile editor; RUNWAY/BURN/MRR SaaS vocabulary for a Gulf trader; a 12-checkbox PDF panel inside invoice creation.*
6. **Rituals are first-class.** When the operator has a closing ritual (e.g. the six-document set: client PO, our quote, costing, DN, supplier invoice, our invoice), the app renders it as a checklist, not as six unrelated attachments.

## Article II — The Nine Patterns (law of flows)

Extracted from the app's own best domain (audit §2). Every new or reworked flow MUST implement whichever of these apply. In review, "does it use the pattern the sales pipeline uses?" is a pass/fail question.

| # | Pattern | Canonical implementation |
|---|---|---|
| 1 | **Context-preserving handoffs** — finish A, land inside B with the job pre-set (pending-store + nav event) | `pendingCostingOpportunity`, `pendingReceiptApply`, `pendingDNCreate` |
| 2 | **Status drives the next action** — render only the legal next step(s), never a free-jump status control | Offers action column; supplier-invoice Match→Approve→Settle |
| 3 | **Revisions, not overwrites** — first-class revision model with an explicit active/final marker | Costing revisions with `is_active`; offer Re-quote/Renew |
| 4 | **Recoverable dead-ends** — every blocked state offers a way forward, reason captured | Credit-limit override modal with required reason |
| 5 | **Guarded single-purpose conversions** — the big transition is one named action with its prerequisite | `MarkOfferWon` requires the customer PO, then auto-creates the order |
| 6 | **Crash-safe long work** — drafts survive; Resume/Discard on return | Costing localStorage draft engine + `beforeunload` |
| 7 | **Integrity surfaced, not hidden** — banners for anomalies, cascade previews before deletes, visible overridden-vs-suggested state | No-items/zero-value banners; delete-cascade preview |
| 8 | **Smart defaults** — pre-fill everything derivable from context; user edits, never re-types | Costing pre-fill from opportunity; per-division terms |
| 9 | **One confirm primitive** — promise-based confirm + double-submit guards for every irreversible action | `askActionConfirmation` (OrdersScreen) |

## Article III — One job, one path

1. **Exactly one way to do each job.** Duplicate paths to the same outcome are defects (audit T7), even when both work — they split muscle memory and audit trails. Consolidate; delete the loser.
2. **The guard ladder.** Calibrate friction to consequence, uniformly:
   - *Read / navigate:* free.
   - *Reversible write:* one click + toast.
   - *Irreversible or posting action (journal, finalize, approve):* explicit named button + the one confirm primitive stating the consequence. Never triggered by row-click or bare selection.
   - *Destructive with dependents:* confirm **plus** cascade preview; mass-destructive adds a second step.
3. **No free-jump status controls.** Status dropdowns that bypass a gated chain are banned (T4); expose only legal transitions (pattern #2).
4. **No ghost actors.** Every mutation is attributed to the authenticated user. If identity is unavailable, block or leave blank — never `'admin'` / `'System User'` (T1). An audit trail with fictional actors is worse than none.
5. **As-of dates.** Period-sensitive postings (reconciliation, revaluation, month-end) take an explicit as-of date; silently stamping `new Date()` is a defect (T8).
6. **Native `confirm()` / `prompt()` / `alert()` are banned** (T5) — crude, unstyleable, and hazardous in a WebView shell. The canonical Modal exists; use it.

## Article IV — Sensory budget (feel)

Feel is earned in this order: **responsiveness → motion → sound.** Skipping ahead is decoration.

1. **Responsiveness first.** Every interaction acknowledges within ~100ms (pressed state, optimistic update where safe). Anything over ~400ms shows progress. No amount of animation rescues a dead click.
2. **Motion values.** Entrances 120–200ms ease-out; exits faster than entrances; press state ≈ scale 0.97 with shadow compression; nothing over 300ms except a deliberate celebratory moment. Always honor `prefers-reduced-motion`.
3. **Sound is saffron.** The application budget is **one sound**, reserved for the operator's true win moment — a deal closing as *paid* — user-initiated, under 300ms, globally mutable. No arrival sounds, no error sounds, no routine-save sounds. *(Implementation belongs to the dedicated Sensory & Brand wave; the budget is law now so nobody spends it early.)*
4. **Toasts confirm, never announce.** A toast may only echo an action the user just took. Nothing arrives unbidden as a toast — arrivals belong to Article V.

## Article V — Alarm philosophy (notifications)

Attention is the scarce resource; the system's credibility is spent every time it interrupts without cause. We own both the platform and the senders, so we can enforce what phone OSes cannot.

1. **Admission control.** A notification type may exist only if it can name: *what action, by whom, by when.* If there's no required human response, it is dashboard state or a log line — not a notification.
2. **Three classes, three physics:**
   - **Task** (approvals, assigned work): persists until *done*. Reading never dismisses it. Lives in a work queue with a clear owner.
   - **Alarm** (rare + consequential: cheque bounced, committed stock unavailable, receivable crossing a threshold): requires acknowledgment, carries an owner and the money at risk, sorted by consequence. More than a handful a week means something is misclassified.
   - **Digest** (everything informational): batched, never interrupts.
3. **Chronic conditions are state, not events.** 157 stalled quotes get an aging board with thresholds — not 157 notifications. Acute events alarm; chronic conditions dashboard.
4. **Read ≠ resolved.** Marking something read may never strand an actionable item. *(Canonical violation: NotificationsScreen approvals whose Approve/Reject render only while unread — audit T3/work domain.)*

## Article VI — One engine (tokens & components)

1. **Onyx & Ether is the only token system.** Screens use semantic tokens; raw hex values, ad-hoc px shadows, and one-off spacing in screen code are review-rejected.
2. **One canonical component per primitive** — Button, Modal, DataTable, PageLayout, form controls — from the shared design system (`packages/`). Hand-rolled tables and dialogs inside screens are defects.
3. **Screens compose; the system owns primitives.** If a screen needs a primitive that doesn't exist, add it to the system first, then consume it. No screen-local forks of shared components (the two divergent line-item editors are the cautionary tale).

## Article VII — Enforcement & amendment

1. **Review is constitutional review.** The orchestrator rejects nonconforming diffs regardless of whether they "work". The owner's final gate re-checks.
2. **The audit's keep-lists (§4) are binding.** A slice touching those screens must preserve the listed behaviors.
3. **Deviations are amendments:** flagged explicitly in the wave status report with justification, effective only on owner sign-off.
4. **Repo invariants stack on top** (`CLAUDE.md`): no real client data (synthetic canon only), no secrets, financial semantics are stop-and-ask, keep the build green.
5. **Ratifications log.** Owner rulings that settle a recurring question get recorded here so future waves don't re-litigate them:
   - **Offer.Stage vocabulary (Wave 9.8 Spec-08 §0.1, ratified 2026-07-12):** `Offer.Stage` keeps its own DB-CHECK vocabulary (`'RFQ'`, `'Quoted'`, `'Won'`, `'Lost'`, `'Expired'`), separate from the canonical Opportunity/RFQ enum (Article I.3). The dormant Cap'n Proto enum also stays untouched. This is a deliberate, bounded exception, not a violation of "status is a vocabulary, not free text" — both vocabularies are still fixed and named. See `stage_vocabulary.go` for the documenting comment, now constitutional. Future waves must not re-open this.
   - **Encrypted document number excluded from collaborative sync (Wave 9.9 Spec-09 §C8(b), ratified 2026-07-12):** `EmployeeDocument.DocNumberEncrypted` carries `json:"-"` (`employee_compliance_service.go`) so the encrypted visa/CPR/permit number is NEVER placed on the collaborative-sync wire — a deliberate PII-minimization consistent with the field-crypto posture (the plaintext is only ever reconstructed locally via the DTO). Cross-device sync therefore does not propagate the document number; this is intended, not a bug. A future wave must NOT "fix" this into a sync payload (that would leak PII across the wire). If the owner ever wants the number to sync, that is a separate, explicit authorization with its own transport-encryption design — not a silent tag removal.
