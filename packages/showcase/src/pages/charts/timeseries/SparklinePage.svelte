<script lang="ts">
  /**
   * SparklinePage — showcase for the Sparkline component.
   *
   * Demonstrates:
   * - KPI-style cards with inline sparklines (Revenue, Orders, Collections)
   * - Color variants (default chart-1, success, danger via seriesColor)
   * - Size variants (narrow/default/wide, short/default/tall)
   * - showArea on/off
   * - Edge cases: 2-point data, flat line, single point
   * - Props reference table
   */

  import { Sparkline } from '@asymmflow/charts';
  import { seriesColor } from '../../../../../charts/src/palette.js';

  // ── Sample data ────────────────────────────────────────────────────────────

  // Al Mahmood Trading — 12-month revenue pulse (BHD thousands)
  const revenueData = [42, 48, 51, 44, 53, 61, 58, 64, 70, 67, 73, 80];

  // Monthly order count
  const ordersData = [120, 134, 118, 141, 152, 148, 160, 175, 168, 182, 195, 210];

  // Collections vs outstanding ratio (percentage)
  const collectionsData = [72, 68, 75, 71, 80, 78, 82, 85, 81, 88, 90, 87];

  // Edge case: only 2 points
  const twoPoints = [30, 70];

  // Edge case: flat line
  const flatData = [50, 50, 50, 50, 50, 50];

  // Edge case: single point
  const singlePoint = [65];

  // KPI deltas for context
  const kpis = [
    {
      label: 'Revenue',
      value: 'BHD 80K',
      delta: '+9.6%',
      positive: true,
      data: revenueData,
      color: seriesColor(0),
    },
    {
      label: 'Orders',
      value: '210',
      delta: '+7.7%',
      positive: true,
      data: ordersData,
      color: seriesColor(1),
    },
    {
      label: 'Collections',
      value: '87%',
      delta: '-3.3%',
      positive: false,
      data: collectionsData,
      color: 'var(--af-danger)',
    },
  ];

  // Props table rows
  const propsRows = [
    { prop: 'data', type: 'number[]', default: '—', desc: 'Array of numeric values to plot.' },
    { prop: 'width', type: 'number', default: '120', desc: 'SVG width in px.' },
    { prop: 'height', type: 'number', default: '32', desc: 'SVG height in px.' },
    { prop: 'color', type: 'string', default: 'seriesColor(0)', desc: 'Stroke & fill color. Use CSS custom property or seriesColor(i).' },
    { prop: 'showArea', type: 'boolean', default: 'true', desc: 'Whether to fill the area below the curve (opacity 0.10).' },
    { prop: 'strokeWidth', type: 'number', default: '1.5', desc: 'Line stroke width in px.' },
    { prop: 'class', type: 'string', default: '—', desc: 'Additional CSS classes.' },
  ];
</script>

<div class="sections">

  <!-- ===== KPI Cards ===== -->
  <section>
    <h2 class="af-section-title">KPI pulse cards</h2>
    <p class="intro">
      Sparklines sit inline inside KPI summary cards for Al Mahmood Trading.
      Each card shows the trailing 12-month trend alongside the current period value.
    </p>
    <div class="kpi-grid">
      {#each kpis as kpi}
        <div class="kpi-card">
          <div class="kpi-top">
            <span class="kpi-label af-label">{kpi.label}</span>
            <Sparkline data={kpi.data} width={120} height={32} color={kpi.color} />
          </div>
          <div class="kpi-value af-numeric">{kpi.value}</div>
          <div class="kpi-delta" class:positive={kpi.positive} class:negative={!kpi.positive}>
            {kpi.delta} vs prior year
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- ===== Color variants ===== -->
  <section>
    <h2 class="af-section-title">Color variants</h2>
    <p class="intro">
      Pass any CSS color expression. Use <code>seriesColor(i)</code> for palette slots
      or semantic tokens for contextual signaling.
    </p>
    <div class="variant-row">
      {#each [0, 1, 2, 3, 4, 5, 6, 7] as i}
        <div class="variant-item">
          <Sparkline data={revenueData} width={100} height={28} color={seriesColor(i)} />
          <span class="af-label variant-label">chart-{i + 1}</span>
        </div>
      {/each}
      <div class="variant-item">
        <Sparkline data={collectionsData} width={100} height={28} color="var(--af-success)" />
        <span class="af-label variant-label">success</span>
      </div>
      <div class="variant-item">
        <Sparkline data={collectionsData.map(v => 100 - v)} width={100} height={28} color="var(--af-danger)" />
        <span class="af-label variant-label">danger</span>
      </div>
    </div>
  </section>

  <!-- ===== Size variants ===== -->
  <section>
    <h2 class="af-section-title">Size variants</h2>
    <p class="intro">Width and height are free-form — the chart scales to fit.</p>
    <div class="size-row">
      <div class="size-item">
        <Sparkline data={revenueData} width={60} height={20} />
        <span class="af-label">60×20</span>
      </div>
      <div class="size-item">
        <Sparkline data={revenueData} width={120} height={32} />
        <span class="af-label">120×32 (default)</span>
      </div>
      <div class="size-item">
        <Sparkline data={revenueData} width={200} height={48} />
        <span class="af-label">200×48</span>
      </div>
      <div class="size-item">
        <Sparkline data={revenueData} width={300} height={64} />
        <span class="af-label">300×64</span>
      </div>
    </div>
  </section>

  <!-- ===== showArea toggle ===== -->
  <section>
    <h2 class="af-section-title">showArea on / off</h2>
    <div class="area-row">
      <div class="area-item">
        <Sparkline data={revenueData} width={160} height={40} showArea={true} />
        <span class="af-label">showArea=true (default)</span>
      </div>
      <div class="area-item">
        <Sparkline data={revenueData} width={160} height={40} showArea={false} />
        <span class="af-label">showArea=false</span>
      </div>
    </div>
  </section>

  <!-- ===== Edge cases ===== -->
  <section>
    <h2 class="af-section-title">Edge cases</h2>
    <div class="edge-row">
      <div class="edge-item">
        <Sparkline data={twoPoints} width={120} height={32} />
        <span class="af-label">2 points</span>
      </div>
      <div class="edge-item">
        <Sparkline data={flatData} width={120} height={32} />
        <span class="af-label">Flat line</span>
      </div>
      <div class="edge-item">
        <Sparkline data={singlePoint} width={120} height={32} />
        <span class="af-label">Single point</span>
      </div>
      <div class="edge-item">
        <Sparkline data={[]} width={120} height={32} />
        <span class="af-label">Empty array</span>
      </div>
    </div>
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
              <td class="af-numeric">{row.default}</td>
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

  /* ── KPI cards ──────────────────────────────────────────────────────────── */
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: var(--af-space-4);
  }

  .kpi-card {
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-space-4);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .kpi-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--af-space-3);
    min-width: 0;
  }

  .kpi-label {
    color: var(--af-text-muted);
    font-size: var(--af-text-xs);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    font-weight: var(--af-weight-semibold);
    /* Truncate gracefully rather than shove the sparkline out of the card. */
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .kpi-value {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-bold);
    font-variant-numeric: tabular-nums lining-nums;
    color: var(--af-text);
    line-height: 1;
  }

  .kpi-delta {
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    color: var(--af-text-muted);
  }

  .kpi-delta.positive { color: var(--af-success); }
  .kpi-delta.negative { color: var(--af-danger); }

  /* ── Color variants ─────────────────────────────────────────────────────── */
  .variant-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-4);
    align-items: flex-end;
  }

  .variant-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-1);
  }

  .variant-label {
    color: var(--af-text-muted);
    font-size: 10px;
  }

  /* ── Size variants ──────────────────────────────────────────────────────── */
  .size-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-5);
    align-items: flex-end;
  }

  .size-item {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  /* ── showArea ───────────────────────────────────────────────────────────── */
  .area-row {
    display: flex;
    gap: var(--af-space-6);
    align-items: flex-end;
    flex-wrap: wrap;
  }

  .area-item {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  /* ── Edge cases ─────────────────────────────────────────────────────────── */
  .edge-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--af-space-5);
    align-items: flex-end;
  }

  .edge-item {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    padding: var(--af-space-3);
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

  code {
    font-family: 'Fira Code', 'Cascadia Code', monospace;
    font-size: 0.88em;
    background: var(--af-surface-sunken);
    padding: 1px 5px;
    border-radius: 3px;
    color: var(--af-accent-pressed);
  }
</style>
