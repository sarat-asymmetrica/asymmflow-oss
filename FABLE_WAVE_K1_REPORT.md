# FABLE Wave K1 — Ledger Blitz — Report

**Branch:** `exp/frontend-kernel` (LOCAL-ONLY) · **Date:** 2026-07-14
**Orchestrator:** Opus 4.8 · **Coders:** 6× Sonnet 5 (2 recon + 6 build agents, gated)
**Commits:** `b885736` (engine spine) · `8103fbe` (batch 1) · this report + batch 2.

## 1. What K1 delivered

Every document-ledger-family screen of the old frontend is now a typed descriptor
rendered by the `DocumentLedger` archetype. **12 ledgers** at mock+real parity minus
explicitly-ledgered gaps:

| # | Screen | Entity | Fetch | Notable | Parity doc |
|---|---|---|---|---|---|
| 1 | Invoices (pilot) | invoices | paged | summary strip added | PARITY_INVOICES.md |
| 2 | Orders | orders | paged | delivery% tone, year/customer filters | Orders.parity.md |
| 3 | RFQs | rfqs | flat | stage machine, stage-set actions | RFQs.parity.md |
| 4 | Offers | offers | flat | computed Expired, 2-signal validity tone | Offers.parity.md |
| 5 | Purchase Orders | purchase-orders | flat | 9-state transition machine, multi-currency, >5000 approval guard | PurchaseOrders.parity.md |
| 6 | Delivery Notes | delivery-notes | flat | status chain, "2 of 3" chip | DeliveryNotes.parity.md |
| 7 | Goods Received | grns | flat | weighted acceptance-rate tone | GRNs.parity.md |
| 8 | Payments (Receipts) | payments | paged | unapplied-balance tone, row-aware Reverse | Payments.parity.md |
| 9 | Credit Notes | credit-notes | paged | **apply-confirm added**, paging fix | CreditNotes.parity.md |
| 10 | Supplier Invoices | supplier-invoices | flat | dual-status (match + payment), match-rate | SupplierInvoices.parity.md |
| 11 | Supplier Payments | supplier-payments | flat | 2-source bridge merge, source badge | SupplierPayments.parity.md |
| 12 | Cheque Register | cheque-register | flat | cheque status machine, row-aware Cancel, age tone | ChequeRegister.parity.md |

Per-screen honest parity ledgers live in `frontend-lab/src/screens/parity/` (485 lines,
the PARITY_INVOICES.md method — DONE/EQUIV/ENGINE/SLOT/INTEG/DEFER). Those docs are the
centerpiece; this report summarizes.

**The thesis, in numbers:** the 11 old ledger screens rebuilt here total **~19,700 LOC**
of hand-written Svelte. The replacement: **~2,000 LOC of typed descriptors** + the shared
kernel engine. (Bridge modules add ~1,800 LOC that is *mostly synthetic mock data* — real
adapters are ~30 lines each; the mock is adversarial fixtures, not production code.)

## 2. Engine features built (only where ≥2 screens needed them)

Built by the orchestrator as the shared, tested kernel spine; agents consumed them:

1. **Summary strip** (`LedgerDescriptor.summary` + `LedgerSummary` primitive) — declarative
   KPI metrics + a status-distribution bar over the visible rows. The **visual-diversity
   vehicle**: replaces ~10 hand-rolled card grids with one dense, tone-aware treatment.
   Used by all 12 ledgers. Computed pure (`computeSummary`, unit-tested).
2. **`ColumnSpec.tone`** — threshold/semantic cell tinting. Live on GRN acceptance %,
   Orders delivery %, Offers validity, Payments unapplied, Supplier-invoice due dates &
   payment status, Expenses payment status, Cheque age.
3. **`StatusSpec.transitions` + `nextStates()`** — declared legal state machines gating row
   actions (PO 9-state, cheque 6-state, expense 5-state, DN, RFQ, offer). Pure + tested.
4. **Filter-chip counts** — `deriveFilterOptions` returns live counts; chips show them.
5. **Row-aware forms** (`FormSpec.initial(row?)` / `submit(draft, row?)`) — surfaced by
   bldPipeline's STOP-and-report; unblocks reason-on-row capture (Payments Reverse,
   Expenses Reject, Cheque Cancel). Backward-compatible.
6. **Single-source tone palette** (`kernel/tones.ts` + `--k-tone-*`) — Badge, summary bars,
   and cell tones share one definition (L2). Zero visual regression.
7. **Infrastructure:** per-entity bridge modules (kills the god-mock, enables parallel
   agent builds), `bridge/map.ts` (shared goDate/str/num), `screens/registry.ts` (the K5
   nav backbone), registry-driven App shell with a grouped sidebar.

## 3. Anti-jank / visual-diversity wins (owner mandate)

- **Summary distribution bars** on every ledger — a live, colored status breakdown that the
  old card grids never had.
- **Tone-aware cells** draw the eye to what matters: amber unapplied balances, red overdue
  dates, threshold-colored acceptance rates.
- **Expenses converted from a card list to a table** — the old card layout read as legacy;
  the data is tabular.
- **Filter chips carry counts** — the deployment's real vocabulary and volume at a glance.

## 4. "Fix, don't preserve" — audit gaps closed (not inherited)

The finance census flagged three real audit/UX regressions in the old screens. Per the
owner's mandate, K1 **fixed** them and ledgered each as an intentional improvement:

1. **Expenses Approve/Post** fired with no confirmation → now confirm-gated.
2. **Expenses Reject** used a hardcoded reason string → now a row-aware reason form that
   captures the operator's actual reason (closes an audit-trail gap).
3. **Credit Notes Apply** reduced AR with no confirmation → now confirm-gated (matches the
   Reverse-Receipt standard for an analogous "reduce AR" operation).

Plus small safety tightenings, each noted in its parity doc: Delivery-Note Delete narrowed
to pre-dispatch (`Prepared`) rows; PO status filters switched to `derive` so the statuses
the old static tabs omitted (Cancelled/Pending Approval/Approved) now surface.

## 5. Financial-semantics hot-zones — PRESERVED, not reimplemented

Every money/posting/tax/reconciliation action was **ledgered, not loosely rebuilt**, so no
descriptor can be mistaken for a live financial flow. Real mutations throw honest INTEG-gap
errors naming their binding (proven pilot pattern); they wire at K5 against the quarantine
backend. Preserved untouched: PO SoD-gated Approve + Receive-Items (inventory posting) + the
>5000 threshold guard; the one AR money-in path (Record Receipt); Supplier-Invoice
Match→Approve→Pay chain; FX-aware Record Payment posting; GRN Complete idempotency; Cheque
Issue/Clear reconciliation; Expense Post-to-GL.

## 6. Consolidated deferred ledger (for K5 / later waves)

Honestly tracked, nothing silently dropped:

- **INTEG (real bindings, K5):** all create/edit/mutation flows currently throw INTEG-gap;
  cross-screen `pending*`+`navigateToScreen` handoffs (Orders→DN/PO/Invoice/Proforma,
  Offers→CostingSheet, DN→invoice) need a real nav primitive; two-phase enriched fetches
  (Orders deliveryStatus, DN order/customer join, SupplierPayments expense source).
- **ENGINE (deferred, ≥2 screens — candidates for a focused mini-wave):**
  - **Multi-panel composition** (5 screens: Payments History, Cheque Registers/Stale,
    Expenses recurring/approvals/workspace) — build the PRIMARY ledger done; secondaries
    ledgered. Biggest single gap the schema doesn't yet answer.
  - **Dual-badge status** (`secondaryStatus`) — SupplierInvoices + Expenses render the
    second dimension as a toned text column today (correct, not mis-badged).
  - **Rich confirm** (`{message, lines[], blocked}`) — Orders cascade-delete preview,
    DN dispatch/POD capture.
  - **Date-range filter primitive** — SupplierInvoices uses a derived-year stand-in.
  - **Server-backed summary binding** — Expenses summary is client-computed today.
- **SLOT (ejection components):** line-item / receive-against-PO / DN-fulfillment /
  CostingSheet create-edit forms; 3-way-match panel; bank-statement-line Clear picker;
  pipeline-trail detail (RFQ→Offer→Order→DN/GRN).
- **DEFER/backend:** RFQ `due_date` phantom field (collected, never persisted — a
  pre-existing backend gap, not fixed silently).
- **NOT a ledger — reclassified:** **QuotationScreen** is an Excel→PDF tool, not a
  document ledger → moved to K4 (bespoke).

## 7. Gate results (green)

- `npm run check` — **0 errors, 0 warnings, 224 files**
- `npm run test` — **26 passing** (+7 new: filter counts, nextStates, computeSummary)
- `npm run build` — clean
- **Layout-detector — CLEAN on all 14 product screens at 1440 + 420** (no overflow, no
  degenerate text columns) against adversarial fixtures (200-char names, 12-digit amounts,
  RTL Arabic, empty fields, unknown statuses, 200–500 rows).
- **Exempt:** `Showcase` (dev kitchen-sink; contains an intentional 3000px overflow demo;
  not a product screen, will not ship).

## 8. Orchestration notes

- 6 build agents wrote **collision-free** disjoint files (per-entity bridge + descriptor +
  parity doc); orchestrator owned every shared file (registry, engine, App).
- One engine gap (row-aware forms) was surfaced by an agent's STOP-and-report and built by
  the orchestrator mid-wave rather than worked around per-screen — the intended model.
- Every descriptor + bridge + parity doc was read and gated by the orchestrator; quality was
  uniformly high (correct snake_case Go mapping, adversarial mocks, honest INTEG-gaps).

## 9. Verdict

K1 is at flip-grade for the ledger family: list + paging + status + filters + search +
summary + safe actions at parity or better, every deep feature honestly ledgered, gates
green, detector clean. Ready for Fable review before K2 (Entity blitz).
