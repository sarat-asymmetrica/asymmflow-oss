# Finance Overview — parity ledger

Old screen: `FinancialDashboard.svelte` (FinanceHub's default dashboard tab).
New: `finance-overview.hub.ts` on the Hub archetype, entity `finance-overview`.

## Built

- 4 KPIs: Revenue (money, YoY delta), Cash Balance, Accounts Receivable (nav →
  `invoices`), Net Profit (delta = margin%).
- `period` selector (FY 2024/2025/2026, default 2026) driving a full refetch —
  matches the old year selector + `GetFinancialDashboardForYear(year)`.
- **distribution** "Balance Sheet" (horizontal) — Current Assets / Non-Current
  Assets / Liabilities / Equity as one bar + legend, replacing the old
  screen's 4 separately-filled `.bs-fill` bars (recon K3a #2) with the Hub's
  existing distribution widget — no new widget type needed.
- **stat-grid** "Key Financial Ratios" — Liquidity / Solvency / Efficiency /
  Profitability sections, each item tone-thresholded via a small `threshold()`
  helper in the descriptor (mirrors the old `ratioHealth()` function almost
  exactly, per recon K3a/K3b's DONE-shape verdict).
- **distribution** "Receivables Aging" (vertical) — current/30-60/60-90/90+,
  tone-ramped success→info→warning→danger. Old screen rendered this as a
  hand-rolled vertical bar chart; same shape, now the Hub's distribution
  widget in vertical orientation.
- **comparison** "Year over Year" (span 2) — Revenue/Gross Profit/Net Profit,
  prior year vs current, BHD. Replaces the old paired-bar-with-%-badge cards.
- **callout** "Statement Check" — synthetic bank-reconciliation notices
  (mirrors `GetCashPosition().notices`).

## Ledgered, not built

- **Cash Conversion Cycle formula box.** Old screen rendered `DSO + DIO − DPO
  = CCC` as a literal equation (bespoke, no reusable shape — recon K3a/K3b
  both call this SLOT/genuinely bespoke). Per the brief, DSO/DIO/DPO/CCC are
  instead surfaced as four stat-grid tiles in the Efficiency section, which
  covers the same numbers without a one-off component. If a literal equation
  box is wanted later, it's a `bespoke` widget (`type: 'bespoke'`) — a small
  new `CccFormula.svelte` under `kernel/widgets/`, not built here to keep this
  descriptor tight.
- **Live cash overlay.** Old screen merged `GetCashPosition()` (live bank
  balance) over `cash_and_equiv` (FY snapshot) for the Cash Balance KPI. This
  descriptor's mock only carries the FY-snapshot number; the two-source merge
  is an INTEG-time concern (compose two `fetch`es before deriving the KPI
  value) — no descriptor-shape change needed, just real-binding wiring at K5.

## Deferred to K5 (division-variant conditional visibility)

`AHSDashboard.svelte` is FinanceHub's *other* dashboard-tab variant, selected
by `getDashboardVariant() === 'ahs'`. Its binding
(`GetFinancialDashboardByDivision`) returns a **strict subset** of
`FinancialDashboard`'s data — no ratios, no balance-sheet breakdown, no AR
aging buckets, no YoY (recon K3a/K3b, confirmed independently by both).
Because a `HubDescriptor` is built against one `Data` type, AHS is **not** a
second dashboard here — it's flagged as a **conditional-widget-visibility /
division-registry concern**:

- Either the Go struct grows those fields for every division (preferred —
  recon K3a's "DEFER: data-shape gap, not a widget-choice gap" verdict), and
  AHS becomes the same `finance-overview` descriptor with a `division` param
  threaded through `fetch`, OR
- the descriptor needs per-field presence checks (`has_data`-style) to hide
  the ratios/aging/YoY widgets when a division's data doesn't carry them —
  the same conditional-visibility pattern the division registry (Wave 12,
  `pkg/overlay`) already establishes for division-scoped rendering elsewhere
  in this codebase.

Do not build `AHSDashboard` as a second Hub instance; resolve the data-shape
gap first.
