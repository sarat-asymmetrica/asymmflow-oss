# Frontend Kernel Campaign — Orchestrator Log

Durable progress tracker for the K1–K6 full-migration campaign
(`FABLE_CAMPAIGN_FRONTEND_KERNEL.md`). Orchestrator = Opus 4.8; coders = Sonnet 5.
Branch `exp/frontend-kernel` (LOCAL-ONLY). Updated as waves land.

## Architecture decisions (orchestrator, binding for all waves)

- **Per-entity bridge modules** (`bridge/<entity>.ts`): each ledger/entity owns a
  self-contained module (types + mock + real + bridge-switched public fns) using the
  shared `bridge/runtime.ts` (`usingWails`/`pick`). Kills the god-mock and lets N
  agents build in parallel with ZERO shared-file collisions. The two pilots
  (invoices/customers) keep the legacy central `bridge/index.ts`; index.ts re-exports
  runtime + barrels the new modules.
- **Screen registry** (`screens/registry.ts`): one typed list mapping key → {label,
  group, archetype, descriptor|component}. App.svelte renders from it. This is the
  K5 nav backbone, grown wave by wave. Registry edits are orchestrator-owned (merge
  point) so parallel agents never touch it.
- **Agent output contract per screen** (collision-free — agents write only NEW files):
  1. `bridge/<entity>.ts`
  2. `screens/<entity>.descriptor.ts`
  3. `screens/parity/<Entity>.parity.md` (the PARITY_INVOICES.md method)
  Orchestrator wires the registry entry + gates + fixes.
- **Visual-diversity mandate** (owner): don't inherit the card-heavy jank. Ledger
  engine gains a declarative **summary strip** (count + money totals + status
  distribution mini-bar) when ≥2 screens want it — consistent data-viz, cheap.

## Gate bar (every wave end)
`npm run check` 0/0 · `npm run test` all green · `npm run build` clean ·
layout-detector zero-violation at 1440/900/420 · per-screen parity docs honest.

## Wave status

| Wave | Scope | Status |
|---|---|---|
| K0 | Baseline verified + scaffolding + engine spine | ✅ done |
| K1 | Ledger blitz — 12 ledgers built, gated, detector-clean; report written | ✅ done (awaiting review) |
| K2 | Entity blitz (suppliers, users, products, warehouse; fold detail views) | ⏳ |
| K3 | Hub archetype + dashboards | ⏳ |
| K4 | Bespoke screens on primitives | ⏳ |
| K5 | App shell + INTEG completion + harness | ⏳ |
| K6 | The flip | ⏳ |

## K1 screen → binding map (to be confirmed by recon)
| Screen | Likely fetch binding | Service |
|---|---|---|
| Orders | ListOrders(limit,offset) / FilterOrders | CRM |
| PurchaseOrders | ListPurchaseOrdersPaginated(page,size,status) | Infra |
| Quotations | (recon) | ? |
| RFQs | (recon) | CRM |
| Offers | GetAllOffers / ListOffersPaginated | CRM/Infra |
| DeliveryNotes | (recon — ListShipments?) | CRM |
| GRNs | ListGRNs(limit,offset,qcStatus) | CRM |
| ChequeRegister | (recon) | Finance |
| Expenses | ListExpenseEntries(status,includePaid) | Finance |
| SupplierInvoices | (recon) | Finance |
| SupplierPayments | GetAllSupplierPayments() | Finance |
| Payments | GetAllPayments(limit,offset) | Finance |
| CreditNotes | ListCreditNotes(limit,offset) | Finance |

## Log
- **K0 (2026-07-14):** Read authority chain (KERNEL/PARITY_INVOICES/spec). Verified
  baseline green. Census: 62 old screen files (~65k LOC). Established per-entity
  bridge + registry architecture. Scaffolding green.
- **K1 recon (2026-07-14):** Two Sonnet agents censused all 13 ledgers → recon-K1-A.md
  (sales) + recon-K1-B.md (finance) in scratchpad. Rulings: QuotationScreen is NOT a
  ledger (Excel→PDF tool → K4). ~10/13 screens want a summary strip. Finance cluster is
  the hot zone (multi-panel, dual-status, transition-gated). "Fix don't preserve" gaps:
  Expenses approve/reject/post no-confirm + hardcoded reason; CreditNotes apply no-confirm.
- **K1 engine spine (2026-07-14, commit b885736):** Built + tested + Playwright-verified
  the shared engine features (≥2 screens each): summary strip (LedgerDescriptor.summary +
  LedgerSummary primitive + distribution bar), ColumnSpec.tone, FilterChips counts,
  StatusSpec.transitions + nextStates(), single-source tone palette (tones.ts + --k-tone-*).
  Invoices pilot upgraded with a summary as the reference. bridge/map.ts extracted (goDate/
  str/num). Gates green (check 0/0, test 26, build), detector 0 violations @1440/420.
- **K1 build batch 1 (2026-07-14, commit 8103fbe):** 6 ledgers built+gated+committed —
  Orders, RFQs, Offers, PurchaseOrders, DeliveryNotes, GRNs. All reviewed by orchestrator
  (types, parity honesty, mock adversarial-ness, real mapping, INTEG-gap discipline).
  Highlights: PO 9-state transition machine w/ approval-threshold guard; GRN weighted
  acceptance-rate tone; Offers two-signal validity tone + computed Expired. Registry wired.
  Full gate green (check 0/0 212 files, test 26, build). Playwright detector CLEAN on all
  8 product screens @1440+420. Showcase (dev kitchen-sink, intentional 3000px overflow demo)
  = detector-EXEMPT, not a product screen, will not ship.
- **K1 build batch 2 (2026-07-14, done):** 6 finance ledgers built+gated — Payments
  (Receipts), CreditNotes, SupplierInvoices, SupplierPayments, ChequeRegister, Expenses.
  All reviewed by orchestrator. Highlights: Payments row-aware Reverse form (validates the
  new engine feature from an agent's hands); SupplierInvoices dual-status (match badge +
  payment toned column); SupplierPayments 2-source bridge merge; ChequeRegister cheque
  status machine + row-aware Cancel; Expenses = card-list→TABLE + all 3 "fix don't preserve"
  gaps FIXED (approve/post confirms + operator reject-reason form). All wired.
- **K1 COMPLETE (2026-07-14):** full gate green (check 0/0 224 files, test 26, build).
  Layout-detector CLEAN on all 14 product screens @1440+420. FABLE_WAVE_K1_REPORT.md written
  (per-screen parity + engine features + consolidated deferred ledger + hot-zone preservation).
  Awaiting Fable review before K2. QuotationScreen reclassified (not a ledger) → K4.

## Engine additions mid-K1 (orchestrator-built on agent findings)
- **Row-aware forms** (2026-07-14): bldPipeline (STOP-and-reported per brief) found
  FormModal never received the clicked row, so row-scoped input-capture actions
  (Cancel/Reject/Reverse with a reason, edit-prefill) couldn't use `action.form`.
  Fix: `FormSpec.initial(row?)` + `submit(draft, row?)`, threaded ActionHost→FormModal→
  FormViewModel. Row typed `unknown` (cast at descriptor; ActionHost is the `any` seam).
  Backward-compat (existing 0-arg forms still valid). Green (check 0/0, test 26). This
  unblocks batch-2 reason-on-row actions (PO Cancel, Cheque Cancel/Stale, Expenses
  Reject, Payments Reverse). RFQ's 4-button stage workaround can later fold to 1 form.

## K1 ruling: scope = the ledger spine
K1 delivers list+paging+status+filters+search+summary+simple actions to parity, and
HONESTLY LEDGERS the deep features (multi-panel composition, dual-status rows, FX/line-item/
receive forms, cross-screen handoffs, real mutations) as SLOT/INTEG/ENGINE for K5. Real
bridge = fetch wired, mutations INTEG-gap throw (proven pilot pattern). 12 real K1 ledgers
(Quotations dropped).
