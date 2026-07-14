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
| K0 | Baseline verified (check 0/0, test 19/19, build clean) + scaffolding | ✅ in progress |
| K1 | Ledger blitz (12 ledgers + credit-notes) | ⏳ |
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
  bridge + registry architecture. Scaffolding next.
