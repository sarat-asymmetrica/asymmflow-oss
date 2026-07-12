<script lang="ts">
  /**
   * AreaChartPage — showcase for AreaChart.
   *
   * Demonstrates:
   * - Stacked cashflow composition: Receivables / Inventory / Cash
   * - Non-stacked overlapping comparison (same three series)
   * - Randomize mid-flight proving geodesic tween interruptibility
   * - Toggleable legend on both variants
   * - Props reference table
   *
   * Note: stacked=true requires all series to share the same x values in the
   * same order — all three series here share months[], satisfying the constraint.
   */

  import { AreaChart } from '@asymmflow/charts';
  import { Button } from '@asymmflow/ui';
  import { formatCurrency } from '../../../../../charts/src/format.js';

  // ── Month labels ───────────────────────────────────────────────────────────

  const months = [
    new Date(2025, 6, 1),
    new Date(2025, 7, 1),
    new Date(2025, 8, 1),
    new Date(2025, 9, 1),
    new Date(2025, 10, 1),
    new Date(2025, 11, 1),
    new Date(2026, 0, 1),
    new Date(2026, 1, 1),
    new Date(2026, 2, 1),
    new Date(2026, 3, 1),
    new Date(2026, 4, 1),
    new Date(2026, 5, 1),
  ];

  function monthLabel(d: number | Date): string {
    if (!(d instanceof Date)) return String(d);
    return d.toLocaleDateString('en-GB', { month: 'short', year: '2-digit' });
  }

  // ── Base data — Al Mahmood Trading cashflow composition (BHD) ─────────────
  // Three components that together explain working capital movement.

  const baseReceivables = [18_400, 21_200, 23_100, 19_800, 24_600, 27_300, 25_900, 29_400, 32_200, 30_100, 33_800, 36_500];
  const baseInventory   = [14_200, 15_800, 16_400, 14_900, 17_600, 19_100, 18_300, 20_800, 22_400, 21_600, 23_900, 25_700];
  const baseCash        = [8_600,  9_400,  10_200, 8_900,  11_300, 12_800, 11_900, 13_600, 15_100, 14_200, 16_400, 18_100];

  // ── Randomize ─────────────────────────────────────────────────────────────

  function randomize(base: number[], spread = 0.22): number[] {
    return base.map((v) => Math.round(v * (1 + (Math.random() - 0.5) * spread)));
  }

  let currentReceivables = $state([...baseReceivables]);
  let currentInventory   = $state([...baseInventory]);
  let currentCash        = $state([...baseCash]);

  function handleRandomize() {
    currentReceivables = randomize(baseReceivables);
    currentInventory   = randomize(baseInventory);
    currentCash        = randomize(baseCash);
  }

  // ── Series ────────────────────────────────────────────────────────────────

  const series = $derived([
    { label: 'Receivables', points: months.map((x, i) => ({ x, y: currentReceivables[i] })) },
    { label: 'Inventory',   points: months.map((x, i) => ({ x, y: currentInventory[i] })) },
    { label: 'Cash',        points: months.map((x, i) => ({ x, y: currentCash[i] })) },
  ]);

  function bhd(n: number): string {
    return formatCurrency(n, 'BHD', { compact: true });
  }

  // ── Props reference ────────────────────────────────────────────────────────

  const propsRows = [
    { prop: 'series', type: '{ label: string; points: { x: number | Date; y: number }[] }[]', default: '—', desc: 'REQUIRED. Series data array.' },
    { prop: 'title', type: 'string', default: '—', desc: 'REQUIRED. Accessible chart name.' },
    { prop: 'stacked', type: 'boolean', default: 'false', desc: 'Stack areas. All series MUST share the same x values in order.' },
    { prop: 'height', type: 'number', default: '280', desc: 'Chart height in px.' },
    { prop: 'yFormat', type: '(n: number) => string', default: 'formatCompact', desc: 'Tick and tooltip y-value formatter.' },
    { prop: 'xFormat', type: '(x: number | Date) => string', default: 'date/number', desc: 'X-axis tick and tooltip header formatter.' },
    { prop: 'curve', type: "'monotone' | 'linear'", default: "'monotone'", desc: 'Line interpolation.' },
    { prop: 'legend', type: 'boolean', default: 'true', desc: 'Show toggleable legend above the chart.' },
    { prop: 'description', type: 'string', default: '—', desc: 'Accessible SVG description.' },
  ];
</script>

<div class="sections">

  <!-- ===== Stacked ===== -->
  <section>
    <h2 class="af-section-title">Stacked cashflow composition</h2>
    <p class="intro">
      Receivables, Inventory, and Cash stack to show total working capital and its
      composition over 12 months. Click "Randomize data" repeatedly mid-flight —
      the stacked bands re-target geodesically without snapping.
      All three series share the same month x-values (required for stacking).
    </p>
    <div class="chart-actions">
      <Button onclick={handleRandomize} variant="secondary" size="sm">Randomize data</Button>
    </div>
    <AreaChart
      {series}
      stacked={true}
      title="Al Mahmood Trading — Working Capital Composition (BHD)"
      description="Stacked area chart showing receivables, inventory and cash composition over 12 months"
      yFormat={bhd}
      xFormat={monthLabel}
      height={300}
    />
  </section>

  <!-- ===== Non-stacked ===== -->
  <section>
    <h2 class="af-section-title">Overlapping comparison</h2>
    <p class="intro">
      The same three series without stacking — useful for comparing magnitude.
      Translucent fills at 12% opacity keep all series legible when they overlap.
      Randomize again to see how non-stacked tween behaves.
    </p>
    <AreaChart
      {series}
      stacked={false}
      title="Al Mahmood Trading — Working Capital Components (overlapping)"
      yFormat={bhd}
      xFormat={monthLabel}
      height={260}
    />
  </section>

  <!-- ===== Props reference ===== -->
  <section>
    <h2 class="af-section-title">Props</h2>
    <div class="props-table-wrap">
      <table class="props-table">
        <thead>
          <tr>
            <th>Prop</th>
            <th>Type</th>
            <th>Default</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          {#each propsRows as row}
            <tr>
              <td><code>{row.prop}</code></td>
              <td class="type-cell">{row.type}</td>
              <td class="default-cell af-numeric">{row.default}</td>
              <td>{row.desc}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>

</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 72ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  .chart-actions {
    display: flex;
    gap: var(--af-space-3);
    margin-bottom: var(--af-space-4);
  }

  /* ── Props table ────────────────────────────────────────────────────────── */
  .props-table-wrap {
    overflow-x: auto;
  }

  .props-table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--af-text-sm);
  }

  .props-table th,
  .props-table td {
    text-align: left;
    padding: var(--af-space-2) var(--af-space-3);
    border-bottom: 1px solid var(--af-border);
    vertical-align: top;
  }

  .props-table th {
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-muted);
    background: var(--af-surface-raised);
  }

  .props-table td {
    color: var(--af-text-secondary);
  }

  .type-cell {
    color: var(--af-text-muted);
    font-family: 'Fira Code', 'Cascadia Code', monospace;
    font-size: 0.88em;
  }

  .default-cell {
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
  }

  code {
    font-family: 'Fira Code', 'Cascadia Code', monospace;
    font-size: 0.88em;
    background: var(--af-surface-sunken);
    padding: 1px 5px;
    border-radius: 3px;
    color: var(--af-accent-pressed);
  }
</style>
