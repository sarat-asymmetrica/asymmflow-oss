<script lang="ts">
  /**
   * BarChart showcase — Gulf-trading quarterly revenue data.
   *
   * Demonstrates:
   * - Grouped vertical bars with legend toggling
   * - Stacked mode switcher (same data, different layout)
   * - Horizontal variant (top customers by outstanding balance)
   * - Randomize button proving geodesic mid-flight retargeting
   * - showValues variant
   * - Single-series degenerate case
   */

  import { BarChart } from '@asymmflow/charts';
  import { formatCurrency } from '@asymmflow/charts';

  // ─── Data ─────────────────────────────────────────────────────────────────

  const quarters = ['Q1 2026', 'Q2 2026', 'Q3 2026', 'Q4 2026'];

  // Quarterly revenue by division — Gulf-trading BHD
  const baseRevenue = [
    { label: 'Industrial Supply',   values: [48_200, 52_100, 61_400, 58_800] },
    { label: 'Marine Equipment',    values: [31_500, 28_700, 34_200, 39_600] },
    { label: 'Construction Goods',  values: [22_800, 35_400, 29_100, 41_200] },
  ];

  function fmt(n: number) {
    return formatCurrency(n, 'BHD', { compact: true });
  }

  // ─── Randomize ────────────────────────────────────────────────────────────

  function rand(min: number, max: number) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  }

  let revenueSeries = $state(baseRevenue.map((s) => ({ ...s, values: [...s.values] })));

  function randomize() {
    revenueSeries = baseRevenue.map((s) => ({
      ...s,
      values: s.values.map(() => rand(15_000, 75_000)),
    }));
  }

  // ─── Mode toggle ──────────────────────────────────────────────────────────

  let mode = $state<'grouped' | 'stacked'>('grouped');

  // ─── Horizontal data — top customers by outstanding ───────────────────────

  const topCustomers = [
    { label: 'Al Mahmood Trading Co',     values: [42_300] },
    { label: 'Gulf Construction Co WLL',   values: [35_800] },
    { label: 'Bahrain Steel & Metals',     values: [28_550] },
    { label: 'Khalid Al Zayani & Sons',    values: [21_200] },
    { label: 'National Supply Corp BSC',   values: [17_900] },
  ];
  const topCustomerCats = topCustomers.map((c) => c.label);

  // ─── Single series degenerate ─────────────────────────────────────────────

  const singleSeries = [
    { label: 'Total Revenue', values: [119_500, 116_200, 124_700, 139_600] },
  ];
</script>

<div class="sections">

  <!-- ===== SECTION 1: Grouped + mode switcher ===== -->
  <section>
    <h2 class="af-section-title">Quarterly revenue by division</h2>
    <p class="intro">
      Three divisions of a Gulf trading company across four quarters.
      Switch between <strong>grouped</strong> and <strong>stacked</strong> to
      compare composition vs individual performance. Click <strong>Randomize</strong>
      rapidly — bars retarget mid-flight with the geodesic tween engine.
    </p>

    <div class="toolbar">
      <div class="seg-group" role="group" aria-label="Chart mode">
        <button
          class="seg"
          class:on={mode === 'grouped'}
          onclick={() => (mode = 'grouped')}
        >Grouped</button>
        <button
          class="seg"
          class:on={mode === 'stacked'}
          onclick={() => (mode = 'stacked')}
        >Stacked</button>
      </div>
      <button class="action-btn" onclick={randomize}>Randomize</button>
    </div>

    <BarChart
      title="Quarterly revenue by division"
      description="Three divisions across Q1–Q4 2026 in BHD"
      categories={quarters}
      series={revenueSeries}
      {mode}
      valueFormat={fmt}
      height={300}
    />
  </section>

  <!-- ===== SECTION 2: Horizontal — top customers ===== -->
  <section>
    <h2 class="af-section-title">Horizontal variant — top customers by outstanding</h2>
    <p class="intro">
      Outstanding balances for the five largest accounts. Horizontal layout
      suits long category names and direct rank comparison.
    </p>

    <BarChart
      title="Top customers by outstanding balance"
      description="Outstanding receivable balance per customer in BHD"
      categories={topCustomerCats}
      series={topCustomers}
      horizontal
      valueFormat={fmt}
      height={260}
      legend={false}
    />
  </section>

  <!-- ===== SECTION 3: showValues ===== -->
  <section>
    <h2 class="af-section-title">Value labels on bars</h2>
    <p class="intro">
      <code>showValues</code> renders compact BHD figures at bar ends. Labels
      appear only when the bar is tall enough (≥14 px) to avoid collisions.
    </p>

    <BarChart
      title="Revenue Q3 with value labels"
      description="Q3 2026 revenue by division with values shown on bars"
      categories={['Q3 2026']}
      series={baseRevenue.map((s) => ({ label: s.label, values: [s.values[2]] }))}
      valueFormat={fmt}
      showValues
      height={220}
    />
  </section>

  <!-- ===== SECTION 4: Single series degenerate ===== -->
  <section>
    <h2 class="af-section-title">Single-series (degenerate case)</h2>
    <p class="intro">
      A single series — no inner band needed, bars span the full category width.
      Legend renders with one item; grouped and stacked are equivalent.
    </p>

    <BarChart
      title="Total quarterly revenue"
      description="Aggregate revenue across all divisions by quarter"
      categories={quarters}
      series={singleSeries}
      valueFormat={fmt}
      height={220}
    />
  </section>

  <!-- ===== SECTION 5: Props table ===== -->
  <section>
    <h2 class="af-section-title">API</h2>
    <div class="props-table-wrap">
      <table class="props-table">
        <thead>
          <tr>
            <th>Prop</th>
            <th>Type</th>
            <th>Default</th>
            <th>Notes</th>
          </tr>
        </thead>
        <tbody>
          <tr><td><code>categories</code></td><td><code>string[]</code></td><td>—</td><td>Required. X-axis (or Y in horizontal) labels.</td></tr>
          <tr><td><code>series</code></td><td><code>&#123; label, values &#125;[]</code></td><td>—</td><td>Required. Values aligned to categories.</td></tr>
          <tr><td><code>mode</code></td><td><code>'grouped'|'stacked'</code></td><td><code>'grouped'</code></td><td>Layout mode.</td></tr>
          <tr><td><code>horizontal</code></td><td><code>boolean</code></td><td><code>false</code></td><td>Rotate 90°.</td></tr>
          <tr><td><code>height</code></td><td><code>number</code></td><td><code>280</code></td><td>SVG height in px. Width is fluid.</td></tr>
          <tr><td><code>valueFormat</code></td><td><code>(n) =&gt; string</code></td><td><code>formatCompact</code></td><td>Axis tick + tooltip + label formatter.</td></tr>
          <tr><td><code>title</code></td><td><code>string</code></td><td>—</td><td>Required. Accessible SVG title.</td></tr>
          <tr><td><code>description</code></td><td><code>string?</code></td><td>—</td><td>Optional SVG desc for additional a11y context.</td></tr>
          <tr><td><code>legend</code></td><td><code>boolean</code></td><td><code>true</code></td><td>Show/hide the toggleable legend.</td></tr>
          <tr><td><code>showValues</code></td><td><code>boolean</code></td><td><code>false</code></td><td>Value labels at bar ends.</td></tr>
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
    line-height: var(--af-leading-relaxed);
  }

  strong {
    font-weight: var(--af-weight-semibold);
    color: var(--af-text);
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    margin-bottom: var(--af-space-4);
  }

  .seg-group {
    display: flex;
  }

  .seg {
    border: 1px solid var(--af-border-strong);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-2) var(--af-space-3);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .seg:first-child {
    border-radius: var(--af-radius-sm) 0 0 var(--af-radius-sm);
  }

  .seg:last-child {
    border-radius: 0 var(--af-radius-sm) var(--af-radius-sm) 0;
    margin-left: -1px;
  }

  .seg:hover {
    background: var(--af-surface-raised);
    color: var(--af-text);
  }

  .seg.on {
    background: var(--af-inverse-surface);
    border-color: var(--af-inverse-surface);
    color: var(--af-text-inverse);
  }

  .seg:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
    position: relative;
    z-index: 1;
  }

  .action-btn {
    border: 1px solid var(--af-border-strong);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-2) var(--af-space-4);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .action-btn:hover {
    background: var(--af-surface-raised);
    color: var(--af-text);
  }

  .action-btn:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  /* Props table */
  .props-table-wrap {
    overflow-x: auto;
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
  }

  .props-table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--af-text-sm);
  }

  .props-table th {
    padding: var(--af-space-3) var(--af-space-4);
    text-align: left;
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-muted);
    background: var(--af-surface-raised);
    border-bottom: 1px solid var(--af-border);
  }

  .props-table td {
    padding: var(--af-space-3) var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
    color: var(--af-text-secondary);
    vertical-align: top;
    line-height: var(--af-leading-relaxed);
  }

  .props-table tr:last-child td {
    border-bottom: none;
  }

  .props-table td:first-child {
    color: var(--af-text);
    font-weight: var(--af-weight-medium);
    white-space: nowrap;
  }

  code {
    font-family: 'Fira Code', 'Cascadia Code', 'Courier New', monospace;
    font-size: 0.88em;
    background: var(--af-surface-sunken);
    padding: 1px 5px;
    border-radius: 3px;
    color: var(--af-accent-pressed);
  }

  @media (prefers-reduced-motion: reduce) {
    .seg, .action-btn { transition: none; }
  }
</style>
