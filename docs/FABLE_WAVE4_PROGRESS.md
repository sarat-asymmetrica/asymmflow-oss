# Fable Wave 4 — Progress Audit

Honest self-audit of Wave 4 ("Shrink the God"), written by the model that
did the work. There was no written handoff this wave: the spec was the
Wave 3 residue list, scoped with the Commander at kickoff (headline = App
fan-out; ZATCA sandbox deferred — it needs portal OTPs). Timestamps are
measured from the git log of `feat/fable-wave4-app-fanout` (all 2026-07-04,
local; the wave began ~05:45 with the Wave 3 → main merge and scoping).

## Measured timeline

| Time | Commit | What |
| --- | --- | --- |
| 06:13 | e321d8a | **A.1** delete-approval workflow → `pkg/infra/deletion` (aggregate + logic inward, host behind 3 ports) — the reference peel shape |
| 06:20 | c6f108e | **A.2** GRN numbering onto the sequence-table engine (last straggler on the buggy BEGIN EXCLUSIVE path; seed from MAX, not COUNT) |
| 06:33 | 0118e26 | **B.2** SessionManager deleted loudly (write-only security theater; AuthManager is the one session system); RateLimiter → `pkg/infra/ratelimit` |
| 06:54 | 10bf0cc | **C.1/C.2** hospitality partial line-level refunds (refund ledger, rounding guard, negative tenders) + `CreditNoteIssued` event through the compliance hook |
| 07:03 | 4f02b3e | **A.3** payroll-run approval routed through the kernel gate (agents refused; RBAC stays authority source) |

## Mission status

### Mission A — App fan-out (headline): DONE for this wave's scope

- **A.1** established the reference peel shape (W4-D1): the aggregate and
  the workflow move into `pkg/infra/deletion`; identity, notification
  delivery, and delete execution stay behind narrow ports. Existing
  end-to-end tests pass untouched — that is the behavior-identity proof.
- **A.2** finished the S4 numbering migration: GRN was the last document
  allocating via raw `BEGIN EXCLUSIVE` + max-scan (lock committed before
  use; any read error silently restarted the year at 0001). Now on the
  engine; legacy continuation pinned by test.
- **A.3** put the third money-approval flow (payroll runs) behind the
  kernel gate, and produced the measured fan-out map (W4-D9) so Wave 5
  peels by data, not vibes.
- Honest accounting: the App went from 1229 to ~1230 METHODS (delegates
  stay, by design — Wails bindings must keep their shape), but the
  delete-approval LOGIC left root, and the map now distinguishes honest
  seams from trampolines. The god-object shrinks by logic, not by method
  count; anyone auditing this wave by counting methods will be misled,
  which is why this line exists.

### Mission B: B.2 DONE; B.1 closed by measurement (no code)

- **B.2** — the in-memory SessionManager was write-only security theater
  (nothing ever read it) duplicating the real DB-backed AuthManager;
  deleted loudly with a signpost. The live RateLimiter promoted verbatim to
  `pkg/infra/ratelimit` (alias keeps call sites + tests unchanged; new pkg
  tests pin refill, key independence, and a 100-goroutine budget race).
- **B.1** — the "unify the three PDF generator paths" residue line
  dissolved under measurement (W4-D4): identity is already unified on the
  overlay; the engine's invoice renderer has zero live product callers; the
  live documents are a consistent gofpdf fleet. The remaining unification
  is a per-document visual-sign-off project — deliberately NOT smuggled
  into a refactor commit. The measured map is recorded so Wave 5 starts
  from ground truth.

### Mission C — hospitality graduation: DONE

- **C.1** partial line-level refunds: several ZATCA 381s per invoice on the
  shared ICV/PIH chain; the per-line refund ledger caps every quantity;
  invoice stays `paid` until the last quantity is credited (no invented
  status); negative tenders net into the day close; a cumulative guard
  refuses rounding-drift over-refunds (pinned by a crafted half-quantity
  case — 976 halalas against a 975-halala invoice, refused). Full-refund
  path behavior-identical, existing tests untouched.
- **C.2** `CreditNoteIssued` is its own domain event (not smuggled under
  InvoiceCreated); the compliance hook subscribes and — bug found while
  wiring — now records the ACTUAL event name instead of stamping
  `finance.invoice.created` on every validation it ever made.
- The demo binary now runs money three ways: sale, full refund, partial
  refund; day close reconciles the net drawer to the halala; exit 0.

### Mission D — the mirror: DONE

`docs/FABLE_WAVE4_DECISIONS.md` (W4-D1…D10) + this audit.

## Thesis: ~93%

Wave 3 closed the composition gap (one seam boots both verticals). Wave 4
moved the needle a smaller, honest step: every money-approval flow in the
trading app now passes the kernel's authority gate (costing, delete
approval, payroll), the proof vertical handles partial money-out with
document-chain integrity, and the App god-object has a proven peel shape
plus a measured map — but the god-object itself still stands (~1230
methods; one workflow's logic left it). What keeps the number honest:

1. The vertical-as-thin-domain-package claim still only fully holds for
   hospitality. Trading's thick root shrinks one aggregate at a time.
2. ZATCA remains gateway-unexercised (deferred with the Commander at
   kickoff — portal OTPs are human-in-the-loop).
3. The PDF fleet is consistent but not unified; unification is a visual
   sign-off project, now correctly scoped as such.

## Residue for Wave 5

- **Payroll domain peel** — the best next full-domain extraction (33
  methods, clean coupling profile), but write golden tests over its
  POSTING semantics first; financial numbers are stop-and-ask.
- **Entity-by-entity delete extraction** — the deletion Executor port's
  ~26-way dispatch is the work-list.
- **Cheap seams queue (full map in W4-D9)**: serial numbers →
  pkg/crm/fulfillment; cheque register + FX revaluation → pkg/finance/*;
  assets/device; the CRM contract body-move. Hubs (auth/RBAC,
  collaboration/notifications) stay behind PORTS — never relocate them.
- **PDF canvas unification** — per-document visual sign-off with the pilot
  (map in W4-D4); forcing function: jung-kurt/gofpdf is archived upstream.
- **ZATCA sandbox round-trip** — unchanged from Wave 3 (portal OTPs +
  JDK 11–14 for the R3.4.x validator); CSR + api client are ready.
- **Session inactivity enforcement** — if wanted, wire through AuthManager
  as a deliberate security change (W4-D3), ideally in the CIA-audit sprint.
- **Offer NN-YY numbering** — standing stop-and-ask (OneDrive coupling).
- **Hospitality**: bill split; print queue; exchange/replacement flows
  (credit note + new invoice as one gesture).
- One root-suite test flake observed once (FAIL then green twice uncached;
  no log captured). If it recurs, capture `-v` output before anything else.
