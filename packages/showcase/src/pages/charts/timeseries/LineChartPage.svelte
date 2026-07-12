<script lang="ts">
  /**
   * LineChartPage — showcase for LineChart.
   *
   * Demonstrates:
   * - 12-month Gulf-trading revenue vs collections (BHD, Al Mahmood Trading)
   * - "Randomize data" Button proving geodesic tween interruptibility
   * - Toggleable legend
   * - Single-series variant
   * - Empty state
   * - Props reference table
   */

  import { LineChart } from '@asymmflow/charts';
  import { Button } from '@asymmflow/ui';
  import { formatCurrency } from '../../../../../charts/src/format.js';

  // ── Month labels ───────────────────────────────────────────────────────────

  const months = [
    new Date(2025, 6, 1),  // Jul 2025
    new Date(2025, 7, 1),
    new Date(2025, 8, 1),
    new Date(2025, 9, 1),
    new Date(2025, 10, 1),
    new Date(2025, 11, 1),
    new Date(2026, 0, 1),  // Jan 2026
    new Date(2026, 1, 1),
    new Date(2026, 2, 1),
    new Date(2026, 3, 1),
    new Date(2026, 4, 1),
    new Date(2026, 5, 1),  // Jun 2026
  ];

  function monthLabel(d: number | Date): string {
    if (!(d instanceof Date)) return String(d);
    return d.toLocaleDateString('en-GB', { month: 'short', year: '2-digit' });
  }

  // ── Base data (Al Mahmood Trading, Gulf Construction) ─────────────────────

  const baseRevenue = [38_200, 41_500, 44_800, 39_600, 47_200, 52_100, 49_800, 55_400, 61_000, 58_300, 63_700, 68_900];
  const baseCollections = [31_400, 35_200, 38_600, 33_900, 40_100, 44_500, 43_200, 47_800, 53_400, 50_600, 55_200, 59_700];

  // ── Randomize state ───────────────────────────────────────────────────────

  function randomize(base: number[], spread = 0.20): number[] {
    return base.map((v) => Math.round(v * (1 + (Math.random() - 0.5) * spread)));
  }

  let currentRevenue = $state([...baseRevenue]);
  let currentCollections = $state([...baseCollections]);

  function handleRandomize() {
    currentRevenue = randomize(baseRevenue);
    currentCollections = randomize(baseCollections);
  }

  // ── Series builders ───────────────────────────────────────────────────────

  const mainSeries = $derived([
    { label: 'Revenue', points: months.map((x, i) => ({ x, y: currentRevenue[i] })) },
    { label: 'Collections', points: months.map((x, i) => ({ x, y: currentCollections[i] })) },
  ]);

  const singleSeries = $derived([
    { label: 'Revenue', points: months.map((x, i) => ({ x, y: currentRevenue[i] })) },
  ]);

  const emptySeries: typeof mainSeries = [];

  // ── Format helpers ────────────────────────────────────────────────────────

  function bhd(n: number): string {
    return formatCurrency(n, 'BHD', { compact: true });
  }

  // ── Props reference ────────────────────────────────────────────────────────

  const propsRows = [
    { prop: 'series', type: '{ label: string; points: { x: number | Date; y: number }[] }[]', default: '—', desc: 'REQUIRED. Series data array.' },
    { prop: 'title', type: 'string', default: '—', desc: 'REQUIRED. Accessible chart name (role="img" aria-label).' },
    { prop: 'height', type: 'number', default: '280', desc: 'Chart height in px. Width is fluid.' },
    { prop: 'yFormat', type: '(n: number) => string', default: 'formatCompact', desc: 'Tick and tooltip y-value formatter.' },
    { prop: 'xFormat', type: '(x: number | Date) => string', default: 'date/number', desc: 'X-axis tick and tooltip header formatter.' },
    { prop: 'curve', type: "'monotone' | 'linear'", default: "'monotone'", desc: 'Line interpolation. monotone = smooth; linear = straight segments.' },
    { prop: 'description', type: 'string', default: '—', desc: 'Accessible description (SVG <desc>).' },
    { prop: 'legend', type: 'boolean', default: 'true', desc: 'Show toggleable legend above the chart.' },
  ];
</script>

<div class="sections">

  <!-- ===== Main demo ===== -->
  <section>
    <h2 class="af-section-title">Revenue vs Collections — Al Mahmood Trading</h2>
    <p class="intro">
      12-month BHD revenue and collections for Gulf Construction.
      Click "Randomize data" repeatedly mid-flight to prove geodesic tween interruptibility —
      the lines re-target from wherever they currently are, never snapping.
      Toggle series in the legend to isolate views.
    </p>

    <div class="chart-actions">
      <Button onclick={handleRandomize} variant="secondary" size="sm">Randomize data</Button>
    </div>

    <LineChart
      series={mainSeries}
      title="Al Mahmood Trading — Revenue vs Collections (BHD)"
      description="12-month trailing revenue and collections comparison for Gulf Construction"
      yFormat={bhd}
      xFormat={monthLabel}
      height={300}
    />
  </section>

  <!-- ===== Single series ===== -->
  <section>
    <h2 class="af-section-title">Single series</h2>
    <p class="intro">
      One series — the legend is still present, toggling it hides the series and
      collapses the y-domain to zero.
    </p>
    <LineChart
      series={singleSeries}
      title="Al Mahmood Trading — Revenue only"
      yFormat={bhd}
      xFormat={monthLabel}
      height={220}
    />
  </section>

  <!-- ===== Curve: linear ===== -->
  <section>
    <h2 class="af-section-title">Linear curve</h2>
    <p class="intro">
      Use <code>curve="linear"</code> when exact sample-to-sample movement matters
      more than visual smoothness.
    </p>
    <LineChart
      series={mainSeries}
      title="Al Mahmood Trading — Linear interpolation"
      yFormat={bhd}
      xFormat={monthLabel}
      curve="linear"
      height={220}
    />
  </section>

  <!-- ===== Empty state ===== -->
  <section>
    <h2 class="af-section-title">Empty state</h2>
    <p class="intro">
      When <code>series</code> is empty or all series are toggled off via the legend,
      a quiet "No data" label fills the plot area.
    </p>
    <LineChart
      series={emptySeries}
      title="Empty chart demo"
      height={180}
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
              <td class="af-numeric default-cell">{row.default}</td>
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
