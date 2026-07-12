<script lang="ts">
  /**
   * DonutChart showcase — receivables aging composition.
   *
   * Demonstrates:
   * - Receivables aging donut with center total in BHD
   * - Legend toggling (visible series re-compose)
   * - Randomize for mid-flight geodesic retargeting
   * - Small-size variant (height 180)
   * - Two-item edge case
   */

  import { DonutChart } from '@asymmflow/charts';
  import { formatCurrency, formatPercent } from '@asymmflow/charts';

  // ─── Data ─────────────────────────────────────────────────────────────────

  const agingItems = [
    { label: 'Current (0–30 days)',    value: 84_600 },
    { label: 'Overdue 31–60 days',     value: 31_200 },
    { label: 'Overdue 61–90 days',     value: 18_450 },
    { label: 'Overdue 90+ days',       value: 9_750  },
  ];

  function fmtBHD(n: number) {
    return formatCurrency(n, 'BHD', { compact: true });
  }

  // ─── Randomize ────────────────────────────────────────────────────────────

  function rand(min: number, max: number) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  }

  let items = $state(agingItems.map((it) => ({ ...it })));

  function randomize() {
    items = agingItems.map((it) => ({ ...it, value: rand(5_000, 120_000) }));
  }

  // ─── Two-item edge case ───────────────────────────────────────────────────

  const twoItems = [
    { label: 'Settled',    value: 68_400 },
    { label: 'Outstanding', value: 31_600 },
  ];

  // ─── Total display for center ─────────────────────────────────────────────

  const totalBHD = $derived(items.reduce((s, it) => s + it.value, 0));

  // Custom center section — fixed values from agingItems (not reactive to randomize)
  const overdueTotal = agingItems.slice(1).reduce((s, it) => s + it.value, 0);
  const grandTotal = agingItems.reduce((s, it) => s + it.value, 0);
</script>

<div class="sections">

  <!-- ===== SECTION 1: Main receivables aging ===== -->
  <section>
    <h2 class="af-section-title">Receivables aging composition</h2>
    <p class="intro">
      Receivables split by aging bucket — the center shows the total balance.
      Toggle legend items to isolate buckets. Click <strong>Randomize</strong>
      rapidly to see smooth geodesic angle retargeting mid-flight.
    </p>

    <div class="toolbar">
      <button class="action-btn" onclick={randomize}>Randomize</button>
      <span class="af-meta">
        Total: <span class="af-numeric">{formatCurrency(totalBHD, 'BHD')}</span>
      </span>
    </div>

    <DonutChart
      title="Receivables aging breakdown"
      description="Outstanding receivable balance segmented by aging bucket"
      {items}
      valueFormat={fmtBHD}
      centerLabel="Total receivables"
      height={280}
    />
  </section>

  <!-- ===== SECTION 2: Small size variant ===== -->
  <section>
    <h2 class="af-section-title">Small size (height 180)</h2>
    <p class="intro">
      At constrained heights the donut remains legible — padAngle and
      cornerRadius keep gaps visible; the legend + tooltip carry all labels.
    </p>

    <DonutChart
      title="Receivables aging — compact"
      description="Compact version at height 180"
      items={agingItems}
      valueFormat={fmtBHD}
      centerLabel="AR total"
      height={180}
      thickness={20}
    />
  </section>

  <!-- ===== SECTION 3: Two-item edge case ===== -->
  <section>
    <h2 class="af-section-title">Two-item edge case</h2>
    <p class="intro">
      Two segments — settled vs outstanding. Even with two items the padAngle
      and cornerRadius prevent a merged blob.
    </p>

    <DonutChart
      title="Settled vs outstanding"
      description="Proportion of receivables settled vs outstanding"
      items={twoItems}
      valueFormat={fmtBHD}
      centerLabel="Balance"
      height={240}
    />
  </section>

  <!-- ===== SECTION 4: Custom center snippet ===== -->
  <section>
    <h2 class="af-section-title">Custom center snippet</h2>
    <p class="intro">
      Providing the <code>center</code> snippet replaces the default total/label.
      Below: a two-line custom center with overdue percentage.
    </p>

    <DonutChart
      title="Receivables aging with custom center"
      description="Custom center showing overdue percentage"
      items={agingItems}
      valueFormat={fmtBHD}
      height={260}
    >
      {#snippet center()}
        <div class="custom-center">
          <div class="custom-pct">{formatPercent(grandTotal > 0 ? overdueTotal / grandTotal : 0)}</div>
          <div class="custom-lbl">overdue</div>
        </div>
      {/snippet}
    </DonutChart>
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
          <tr><td><code>items</code></td><td><code>&#123; label, value &#125;[]</code></td><td>—</td><td>Required. Order is preserved (no d3 sort).</td></tr>
          <tr><td><code>height</code></td><td><code>number</code></td><td><code>260</code></td><td>SVG height in px. Width is fluid.</td></tr>
          <tr><td><code>thickness</code></td><td><code>number</code></td><td><code>26</code></td><td>Ring thickness in px (outerR − innerR).</td></tr>
          <tr><td><code>valueFormat</code></td><td><code>(n) =&gt; string</code></td><td><code>formatCompact</code></td><td>Tooltip value formatter.</td></tr>
          <tr><td><code>title</code></td><td><code>string</code></td><td>—</td><td>Required. Accessible SVG title.</td></tr>
          <tr><td><code>description</code></td><td><code>string?</code></td><td>—</td><td>Optional SVG desc.</td></tr>
          <tr><td><code>legend</code></td><td><code>boolean</code></td><td><code>true</code></td><td>Show/hide the toggleable legend.</td></tr>
          <tr><td><code>centerLabel</code></td><td><code>string?</code></td><td>—</td><td>Caption below the total in the center hole.</td></tr>
          <tr><td><code>center</code></td><td><code>Snippet</code></td><td>—</td><td>Override the entire center content.</td></tr>
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
    gap: var(--af-space-4);
    margin-bottom: var(--af-space-4);
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

  /* Custom center */
  .custom-center {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: 2px;
  }

  .custom-pct {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-bold);
    font-variant-numeric: tabular-nums lining-nums;
    color: var(--af-danger);
    line-height: 1;
  }

  .custom-lbl {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    color: var(--af-text-muted);
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
    .action-btn { transition: none; }
  }
</style>
