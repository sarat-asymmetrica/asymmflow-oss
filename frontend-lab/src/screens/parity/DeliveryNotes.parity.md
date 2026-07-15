# Parity Ledger — DeliveryNotesScreen (old) vs DeliveryNotes descriptor

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Flat list (`GetDeliveryNotes()`, no pagination) | **DONE** | Single `fetch()`, no `fetchPage` — matches the real binding's shape. |
| 2 | Status filter tabs | **DONE** | Derived chip. `Signed`/`Cancelled` exist in the backend enum but have no UI path in the old screen either — not modeled. |
| 3 | Free-text search (dn#/customer/order ref/driver/vehicle) | **DONE** | One `searchText` (L2). |
| 4 | Order/Customer client-side enrichment (`order_reference`, `customer_name`) | **ENGINE / INTEG** | Not on the `DeliveryNote` struct — old screen joins against separately-loaded Orders + Customers lists. K1's mock **synthesizes** these fields directly onto the row; real integration needs the same join (or a backend change to return them inline). Flagged, not silently faked as real. |
| 5 | "Delivery # = 2 of 3" partial-delivery chip | **DONE** | Computed cell from `deliverySeq`/`totalDeliveries`; renders `Full` for single-delivery orders. |
| 6 | Summary strip (Total, In Transit, Delivered, status mix) | **DONE** | New — old screen's stats row rebuilt as the declarative `SummarySpec`. |
| 7 | Create DN (fulfillment sub-form: ordered/prev/remaining/now qty + serials) | **LEDGER / SLOT / ENGINE** | Structurally identical to GRN's receive-form (K1-A synthesis #2) — a shared `FulfillmentLineEditor` candidate. Not built in K1; no screen action exists for it. |
| 8 | Dispatch (recoverable inline driver/vehicle capture) | **LEDGER / SLOT** | Real action needs a richer confirm-as-form (`ActionSpec.confirm` is a plain string today). Not built. |
| 9 | Confirm Delivery (POD signature capture) | **LEDGER / SLOT** | Same richer-confirm-as-form need as #8. Also triggers a "create invoice?" cross-screen handoff when the parent order becomes fully delivered — ledgered alongside Orders' handoff family. Not built. |
| 10 | Status advance (Prepared→Dispatched→InTransit→Delivered) | **DONE** (simplified) | `StatusSpec.transitions` declares the legal chain; a single "Advance Status" row action flips to the next state via a plain confirm — **not** the real Dispatch/POD-capture forms (#8/#9). Real binding throws an honest INTEG-gap naming both `DispatchDeliveryNote`/`ConfirmDeliveryNote`. |
| 11 | Delete (`DeleteDeliveryNote`) | **DONE** (mock, narrowed) | Old screen allows Delete from any status; K1 restricts it to `Prepared` (pre-dispatch) as an intentional safety improvement — not a preserved-as-is behavior. Real binding is INTEG-gapped. |
| 12 | Edit (header fields) | **SLOT** | Form archetype territory; not built in K1. |
| 13 | Generate PDF + Open exported file | **INTEG** | Same two-call pattern as Invoices #13/Offers PDF; cheap to wire once one INTEG binding is proven, not built in K1. |
| 14 | Year filter (`availableYears`, no dropdown control found) | **DEFER** | Old screen computes the array but no visible control was found in the recon — not reproduced; flagged as vestigial rather than assumed-needed. |

## Reading

K1 builds the DeliveryNotes spine at parity: flat list, status filter,
search, the partial-delivery chip, and a new summary strip. The two real
mutating flows (Dispatch, Confirm Delivery) both need capture-at-confirm
forms the kernel doesn't have yet, so K1 substitutes one honest,
deliberately-simplified "Advance Status" action along the same
`StatusSpec.transitions` chain instead of faking the real UX. The
order/customer enrichment is mocked data in K1's bridge, not a real join —
real integration needs either the same client-side join the old screen does
or a backend change. Delete is narrowed to `Prepared` rows as a stated safety
improvement, not a regression.
