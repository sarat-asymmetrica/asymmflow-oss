# Parity Ledger — PricingScreen (old) vs Pricing (bespoke K4)

Verdicts (see `PARITY_INVOICES.md` for the full legend):

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL ledgers at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Customer sidebar list, click to select | **DONE** | `ListWidget` (label = name, detail = regime + current win rate, right value = win rate, accent tone = regime) with click routed through the standard `NavIntent` slot. `bridge/pricing.ts#fetchPricingCustomers` is genuinely mock-only — see #4. |
| 2 | Target-margin slider (5%–50%, step 1%) | **DONE** | New `kernel/controls/RangeSlider.svelte` — a plain styled `<input type="range">` wrapper (bindable value, optional formatted live label), same 0.05–0.50/step 0.01 range as the old screen. |
| 3 | Run Simulation → `SimulateMargin(customerName, proposedMargin)` | **DONE (real)** | `bridge/pricing.ts#simulateMargin` real path calls `InfraService.SimulateMargin` directly — confirmed signature `SimulateMargin(customer string, proposedMargin float64) (*main.MarginSimulation, error)` and confirmed response fields (`customer`, `proposed_margin`, `current_win_rate`, `estimated_win_rate`, `confidence`, `recommended_action`, `warning?`) against `wailsjs/go/models.ts`. This is the one genuinely real binding on the old screen and it's wired 1:1, not mocked. |
| 4 | Customer list + win rates (`overallStats.customers`) | **INTEG (ledgered, not ported-as-real)** | Confirmed in the old screen's own source: `overallStats` is a hardcoded literal object, not a fetch. This build does **not** port that array pretending it's real — `fetchPricingCustomers`'s real path throws an honest INTEG-gap error naming the gap ("no real customer/win-rate endpoint exists... wire a real source before K5"); the mock path generates a comparable adversarial dataset instead. |
| 5 | `GetPricingRecommendation` import | **DEFER (dead import, not ported)** | Confirmed unused in the old screen (imported, never called) — not carried into this bridge at all, matching the recon finding. |
| 6 | Projected win rate + strategic-impact result panel | **DONE** | `StatTileGrid` (current win rate / projected win rate, tone-flipped success↔danger by direction / confidence) + `CalloutWidget` (guidance = `recommended_action`, optional warning row when the real/mock result carries one). |
| 7 | Regime-specific guidance copy (Premium/PriceSensitive/ValueBalanced paragraph) | **DONE** | Ported verbatim as a `REGIME_GUIDANCE` lookup rendered through a neutral-tone `CalloutWidget`, same three regime strings the real `MarginSimulation`/mock data already use. |
| 8 | Header badges: overall avg margin / win rate | **DEFER** | Old header badges (`Avg Margin: 22%`, `Win Rate: 35%`) come from the same hardcoded `overallStats` object as #4 — not reproduced here since it's synthetic-on-synthetic with no real source; revisit alongside #4 if a real aggregate endpoint appears. |

## Reading

This is a financial-subject screen (margin/pricing strategy) but low actual
risk today: the only mutation-adjacent call is a read-only simulation, and
it's wired to the real binding rather than mocked. The one thing this build
refuses to do is dress up the old screen's hardcoded customer array as if it
were a real fetch — that gap is named explicitly (`fetchPricingCustomers`'s
INTEG-gap message) so whoever wires K5 knows exactly what's missing: a real
customer list with real current win rates, not just a `SimulateMargin` call
site.
