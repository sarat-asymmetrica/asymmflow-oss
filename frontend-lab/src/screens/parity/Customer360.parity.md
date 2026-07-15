# Parity Ledger — Customer360 (old) vs Customer360 (bespoke K4)

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Customer 360 info panel (name, code, key facts) | **DONE (fetch INTEG-gapped)** | Left `Card` — name/code + `StatTileGrid` "Financial" section (Lifetime Value/Avg Payment Days/Disputes). `bridge/customer-360.ts::fetchCustomer360` stands in for `GetCustomer360` (`App`), INTEG-gapped per the K4 brief. |
| 2 | Payment-regime badge | **DONE** | `REGIME_TONE` toned `Badge` next to the name (success=Prompt, info=Standard, warning=Slow, danger=AtRisk); unrecognized values fall back to neutral via `?? 'neutral'`, same fallback contract as SerialTrace's `STAGE_TONES`. |
| 3 | Recent grade predictions (grade + confidence + predicted-days) | **DONE (fetch INTEG-gapped)** | Predictions tab — `ListWidget` rows (`toListRow`), toned by grade (A=success…D=danger, unknown=neutral). Confidence and predicted-days both render in the row detail; no separate mathematical-rigor treatment. |
| 4 | `RegimeBadge` / `MathematicalRigorBadge` bespoke components | **EQUIV** | Reimplemented as a plain toned `Badge` (regime) + the confidence percentage folded into the `ListWidget` detail line — not ported verbatim, per the K4 brief's explicit instruction not to carry the consciousness-package components forward. |
| 5 | "Connections" tab — connection-count stats + related products/suppliers | **DONE (fetch INTEG-gapped)** | Connections tab — `StatTileGrid` "Network" (Total Connections, Centrality Score) + two flat `Badge` chip rows (Related Products / Related Suppliers). `bridge/customer-360.ts::fetchCustomer360Connections` stands in for `GetCustomer360Graph` (`CRMService`), INTEG-gapped. |
| 6 | Node-link relationship graph visualization | **N/A — never existed** | Recon correction (recon-K4.md): the old "Relationships" tab only ever rendered connection-count stats + two flat chip lists, no graph-viz. No new graph primitive was built or needed; this row exists only to record that the earlier K2 deferral note overstated the old screen's scope. |
| 7 | Overview tab — contact + commercial sections | **DONE** | Two `StatTileGrid` sections (Contact: person/phone/email/address; Commercial: terms/credit limit/TRN/industry/relationship years), split out from the left panel's headline financial stats so nothing is rendered twice. |
| 8 | Customer selection (old screen was routed to via a specific customer ID) | **EQUIV** | This bespoke build has no master list to select from, so a small synthetic `FilterChips` picker (`bridge/customer-360.ts::fetchCustomer360Directory`) stands in for "which customer" — lab-only scaffolding, not a stand-in for a real endpoint. At K5 this picker is replaced by real navigation (customer row → Customer 360), not ported as-is. |
| 9 | Fallback to hardcoded mock when `window.go` is absent | **EQUIV** | Superseded by the bridge's `pick(real, mock)` switch (`bridge/runtime.ts`) — the same real/mock decision point every other K4 bridge module uses, not a screen-local fallback. |

## Reading

Both real bindings (`GetCustomer360`, `GetCustomer360Graph`) are confirmed
real and mapped 1:1 in the bridge's real-side function names, but INTEG-gapped
per the K4 brief — this is customer financial history and predictive grading,
moderate-sensitivity read-only data, so the lab stays on adversarial synthetic
data end to end (including a customer with zero predictions and a customer
with zero connections, to exercise both tabs' empty states) until K5 wiring.
