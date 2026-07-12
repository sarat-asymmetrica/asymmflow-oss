# Fable Wave 5 — Progress Audit

Honest self-audit of Wave 5 ("Peel by the Map"), written by the model
that did the work, against the written handoff `FABLE_WAVE5_HANDOFF.md`.
Timestamps are measured from the git log of `feat/fable-wave5-peel-by-map`
(all 2026-07-04, local; the wave began ~11:20 with the Wave 4 merge
confirmation, the branch, and the Mission B policy question to the
Commander — answered: 30-minute idle, refuse + re-login).

## Measured timeline

| Time | Commit | What |
| --- | --- | --- |
| 11:34 | 9b2d612 | **A.1a** serial-number lifecycle → `pkg/crm/fulfillment` (zero ports; the two serial trampolines deleted from the generic Handlers; `text.EscapeLike` to the kernel) |
| 11:42 | f54a92f | **A.1b** cheque register + lifecycle → `pkg/finance/cheque` |
| 11:45 | 4d0b1f9 | **A.1c-pin** FX golden tests, exact-binary fixtures, green against untouched code |
| 11:51 | 8f0cb42 | **A.1c** FX rates + revaluation → `pkg/finance/fx`; goldens pass unchanged |
| 12:02 | bf4e1da | **A.1d** assets store → `pkg/infra/assets`; device fingerprint/lifecycle → `pkg/infra/device` (auth-entangled flows deliberately stay — W5-D3) |
| 12:19 | f7e5e72 | **A.1e** contract body-move → `pkg/crm/contract` (the "cleanest existing peel" was an empty stub — W5-D4; found + fixed seeds that could never complete) |
| 12:23 | e289a68 | **A.2-pin** payroll golden tests (generation totals + accrual/payout journals), green against untouched code |
| 12:35 | 1228d0a | **A.2** payroll domain → `pkg/finance/payroll` (posting inward, four ports out — W5-D5); goldens pass unchanged |
| 12:46 | c4458fa | **Mission B** 30-min interactive inactivity via AuthManager, enforced at requirePermission (W5-D6) |
| 12:57 | 3160818 | **C.1** hospitality bill split — whole-line assignment, exact sum invariant, line→invoice stamp (W5-D7) |
| 13:03 | 066fbea | **C.2 (stretch)** minimal print spooler — atomic enqueue with the document, claim/mark worker seam |

Plus the wave-end chore commit (regenerated Wails bindings + frontend
namespace fixes) after this audit.

## Mission status

### Mission A — peel the cheap seams, then payroll: DONE, full queue

All four cheap seams landed in the W4-D1 shape, plus the optional CRM
contract body-move, plus the payroll headline. Every peel: existing
app-level tests untouched and green, new pkg-level tests for the moved
logic, RBAC guards kept at root (hubs stay hubs).

- Both financial-arithmetic peels (FX, payroll) followed the golden-first
  discipline: numbers pinned in their own commit against the UNTOUCHED
  code, then the peel commit runs the same tests unchanged. Payroll's
  five accrual lines (6000/6050 debits vs 2210/2211/2212 credits,
  balanced at 2056 in the fixture), account balances, payout journal, and
  the expense mirror are all pinned with exact float64 equality.
- Free-audit finds, fixed and recorded: the contract models never minted
  IDs, so template/clause seeding ALWAYS failed after the first row
  (W5-D4); the contract "extraction" was an empty stub constructed but
  never called.
- Corrected-map NOs, recorded: device's login/setup/approve flows are the
  auth hub in a device costume — ports-not-relocation applies, so they
  stayed in root (W5-D3); payroll's expense-ledger bridge stayed behind a
  port for the same reason (W5-D5).

**Honest accounting (the W4 rule: LOGIC moved, not method count).** The
seven peeled root files went from ~3,800 lines of logic to ~335 lines of
thin delegates and type aliases — ≈3,450 LOC of domain logic left the
root package this wave; `pkg/` gained ≈4,900 lines (the moved logic plus
its new tests). The App's Wails-visible method count is ~1,232 —
essentially unchanged BY DESIGN (delegates keep the binding surface;
Mission B added a logout binding). Anyone auditing by method count will
again be misled; audit by where the bodies live.

### Mission B — session inactivity, honestly this time: DONE

30-minute idle timeout for interactive logins, enforced server-side at
requirePermission (the chokepoint every bound call passes), DB session
rows through AuthManager's own table for the audit trail, refuse +
re-login UX via one Wails event. The W4-D3 mirror rule is satisfied by
construction: the read side is the RBAC middleware itself, and the tests
exercise bound-call behavior — expiry blocks, activity extends, logout
invalidates, a new login supersedes. Found on the way: the UserSession
table had served ONLY the OAuth flow; interactive logins never had a
session lifecycle at all.

### Mission C — hospitality: C.1 DONE, C.2 (stretch) DONE

- **C.1 bill split**: one open session → N invoices by whole-line
  assignment (quantity splitting refused, never implemented — the sum
  invariant then holds by construction and is still guarded at issuance
  per W4-D6). Each split invoice is its own ZATCA document on the shared
  ICV/PIH chain, issued in one transaction; kernel CanApprove gates it;
  agents never issue; payments and refunds compose per invoice. The split
  exposed and fixed a latent scoping hole: refunds resolved lines by
  SESSION, exact only while a session has one invoice — order lines are
  now stamped with their invoice at issuance, with an exact legacy
  fallback (W5-D7).
- **C.2 print spooler**: kitchen tickets and every invoice (close and
  split) enqueue print jobs atomically with the document; a worker claims
  FIFO per station and marks printed/failed (with requeue). No driver —
  the seam is the point.
- The demo binary now runs money FOUR ways — sale, full refund, partial
  refund, split bill — and the day close still reconciles the net drawer
  to the halala; exit 0.

### Mission D — the mirror: DONE

`docs/FABLE_WAVE5_DECISIONS.md` (W5-D1…D7) + this audit. Every decision
entry was written when decided, not at wave end.

## Thesis: ~94%

Wave 5 executed the entire W4-D9 cheap-seam queue plus the payroll
full-domain peel — the largest single block of logic yet to leave the
root (~3,450 LOC this wave). The proof vertical now handles the classic
POS money shapes (split bills included) with document-chain integrity,
and the substrate gained real security lifecycle for interactive
sessions. What keeps the number honest:

1. The trading root still holds the EXPENSIVE clusters the map priced
   correctly: the sales-pipeline surfaces (RFQ→offer→costing→order),
   setup/documents, butler context, OneDrive/ETL, and the two hubs
   (auth/RBAC, collaboration/notifications). "Vertical = configuration
   plus a thin domain package" fully holds for hospitality only.
2. ZATCA remains gateway-unexercised (portal OTPs are human-in-the-loop;
   standing deferral).
3. The deletion Executor's ~26-way dispatch is still the entity-by-entity
   work-list nobody has started.

## Residue for Wave 6

- **The expensive clusters, by the standing rules**: sales-pipeline
  surfaces and OneDrive/ETL only behind ports or with a dedicated design
  pass; hubs (auth/RBAC, collaboration/notifications) get PORTS, never
  relocation. The deletion Executor dispatch is the natural work-list for
  entity-by-entity extraction.
- **Butler read paths** — still the cheapest untouched seam (invariant 4
  means no RBAC/notification entanglement).
- **PDF canvas unification** — unchanged: a per-document visual sign-off
  project with the pilot (map in W4-D4); jung-kurt/gofpdf archived
  upstream remains the forcing function.
- **ZATCA sandbox round-trip** — unchanged (portal OTPs + JDK 11–14;
  CSR + api client ready).
- **Offer NN-YY numbering** — standing stop-and-ask, untouched four waves
  running.
- **Hospitality**: exchange/replacement flows (credit note + new invoice
  as one gesture) — still needs the design conversation first; a real
  print driver behind the C.2 seam if a pilot wants paper.
- **Mission B follow-ons** (CIA-audit sprint candidates): a logout button
  in the UI wired to LogoutInteractiveSession; configurable timeout via
  settings if the pilot asks; extending session rows to license-based
  flows.
- The accrual journal does NOT balance when an item's deductions exceed
  its gross (the net clamp absorbs the difference on the debit side only)
  — observed while building the payroll goldens, pinned as behavior
  (clamp test), NOT changed: posting semantics are stop-and-ask. Raise
  with the Commander before touching.

Build → Test → Ship. Measured, not estimated. 🌊
