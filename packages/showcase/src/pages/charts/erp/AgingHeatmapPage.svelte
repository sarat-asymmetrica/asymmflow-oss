<script lang="ts">
  /**
   * AgingHeatmapPage — ERP showcase: receivables aging matrix.
   *
   * Demonstrates:
   * - 8-customer × 5-bucket matrix with realistic Gulf-trading skew
   * - Clickable-cells variant wired to toast.info()
   * - Compact 4-customer variant
   *
   * Data: Al Mahmood Trading customers — Bahrain Gulf-trading narrative.
   * Skew: most accounts Current, two problem accounts heavy in 90+.
   */

  import { AgingHeatmap } from '@asymmflow/charts';
  import { toast } from '@asymmflow/ui';

  // ─── 8-customer dataset (realistic Gulf-trading skew) ────────────────────

  const rows = [
    { label: 'Gulf Equipment Trading',  values: [12_450,     0,       0,       0,       0] },
    { label: 'Al Moayyed Contracting',  values: [ 4_875,  2_340,      0,       0,       0] },
    { label: 'Zain Bahrain BSC',        values: [ 8_910,     0,   1_200,      0,       0] },
    { label: 'Gulf Air Group',          values: [ 6_500,  8_000,  5_000,      0,       0] },
    { label: 'Ithmaar Bank',            values: [     0,     0,   2_100,  3_532,       0] },
    { label: 'Arab Banking Corp',       values: [     0,     0,       0,  4_800,  17_300] },
    { label: 'Khaleeji Commercial',     values: [ 3_890,     0,       0,      0,       0] },
    { label: 'GFH Financial Group',     values: [ 5_200,  2_000,      0,      0,       0] },
  ];

  // ─── Compact 4-customer variant ───────────────────────────────────────────

  const compactRows = [
    { label: 'National Petroleum Co.',   values: [31_200,     0,      0,     0,      0] },
    { label: 'Midal Cables',        values: [ 1_820,     0,      0,     0,      0] },
    { label: 'Seef Properties',     values: [ 4_000,  5_450,     0,     0,      0] },
    { label: 'Arabian Industries',  values: [     0,     0,  2_450,     0,      0] },
  ];

  // ─── Cell click → toast ───────────────────────────────────────────────────

  function handleCellClick(row: string, bucketIndex: number) {
    const bucketLabels = ['Current', '1–30 days', '31–60 days', '61–90 days', '90+ days'];
    const bucket = bucketLabels[bucketIndex] ?? `Bucket ${bucketIndex}`;
    const rowData = rows.find((r) => r.label === row);
    const value = rowData?.values[bucketIndex] ?? 0;

    if (value > 0) {
      toast.info(`${row} · ${bucket}: BHD ${value.toLocaleString('en-US', { minimumFractionDigits: 3 })}`);
    } else {
      toast.info(`${row} · ${bucket}: No balance outstanding`);
    }
  }

  // ─── Props table ──────────────────────────────────────────────────────────

  const propRows = [
    { name: 'rows',        type: '{ label: string; values: number[] }[]', required: true,  desc: 'Row data — each values array must match bucket count.' },
    { name: 'title',       type: 'string',                               required: true,  desc: 'ARIA region label.' },
    { name: 'buckets',     type: 'string[]',                             required: false, desc: 'Bucket header labels. Default: Current, 1–30, 31–60, 61–90, 90+.' },
    { name: 'valueFormat', type: '(n: number) => string',                required: false, desc: 'Cell value formatter. Default: formatCompact.' },
    { name: 'description', type: 'string',                               required: false, desc: 'Extended a11y description.' },
    { name: 'onCellClick', type: '(row: string, bucketIndex: number) => void', required: false, desc: 'Turns cells into <button>s when provided.' },
  ];
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1 class="page-title">AgingHeatmap</h1>
      <p class="page-desc">
        Receivables aging matrix — the CFO's morning read. Real HTML table for full
        keyboard accessibility and copy-paste. Cell intensity encodes balance severity
        against the global max (honest cross-column comparison). Severity hue tracks
        overdue risk per bucket column.
      </p>
    </div>
  </header>

  <!-- ─── Demo: Full 8-customer clickable ───────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Clickable Cells — Click any cell</h2>
      <p class="section-subtitle">
        Al Mahmood Trading · 8 accounts · Click any cell to fire a toast
      </p>
    </div>

    <AgingHeatmap
      {rows}
      title="Receivables Aging — Al Mahmood Trading, May 2026"
      description="8 key accounts across 5 aging buckets. Arab Banking Corp and Ithmaar Bank are problem accounts."
      onCellClick={handleCellClick}
    />
  </section>

  <!-- ─── Demo: Read-only full matrix ───────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Read-only Matrix</h2>
      <p class="section-subtitle">No onCellClick — hover ring still shows; no button chrome</p>
    </div>

    <AgingHeatmap
      rows={rows.slice(0, 5)}
      title="Receivables Aging — Read-only view"
    />
  </section>

  <!-- ─── Demo: Compact 4-customer ─────────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Compact Variant — 4 Accounts</h2>
      <p class="section-subtitle">Khalifa Logistics & Manama Retail Group cluster</p>
    </div>

    <div class="compact-wrap">
      <AgingHeatmap
        rows={compactRows}
        title="Compact Receivables Aging — Bahrain Steel cluster"
      />
    </div>
  </section>

  <!-- ─── Demo: Custom buckets ──────────────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Custom Buckets</h2>
      <p class="section-subtitle">Q1/Q2/Q3 quarterly aging view</p>
    </div>

    <div class="compact-wrap">
      <AgingHeatmap
        rows={[
          { label: 'Gulf Air Group',   values: [19_500, 8_200, 0] },
          { label: 'Batelco Group',    values: [ 3_221,     0, 0] },
          { label: 'Ithmaar Bank',     values: [     0, 5_632, 0] },
        ]}
        buckets={['Q1 2026', 'Q4 2025', 'Q3 2025']}
        title="Quarterly Aging View"
      />
    </div>
  </section>

  <!-- ─── Props table ─────────────────────────────────────────────────────── -->
  <section class="demo-section">
    <h2 class="section-title af-label">Props</h2>
    <div class="props-table-wrap">
      <table class="props-table">
        <thead>
          <tr>
            <th class="af-label">Prop</th>
            <th class="af-label">Type</th>
            <th class="af-label">Required</th>
            <th class="af-label">Description</th>
          </tr>
        </thead>
        <tbody>
          {#each propRows as row}
            <tr>
              <td class="prop-name af-numeric">{row.name}</td>
              <td class="prop-type">{row.type}</td>
              <td class="prop-req">{row.required ? 'Yes' : 'No'}</td>
              <td class="prop-desc">{row.desc}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </section>
</div>

<style>
  .page {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
    padding: var(--af-space-5);
    max-width: 960px;
  }

  .page-header {
    padding-block-end: var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
  }

  .page-title {
    font-family: var(--af-font-numeric);
    font-size: calc(24px * var(--af-scale, 1));
    font-weight: var(--af-weight-bold);
    color: var(--af-text);
    margin: 0 0 var(--af-space-2);
    letter-spacing: -0.02em;
  }

  .page-desc {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    line-height: var(--af-leading-relaxed);
    max-width: 640px;
    margin: 0;
  }

  .demo-section {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-3);
  }

  .section-header {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .section-title {
    color: var(--af-text-muted);
    margin: 0;
  }

  .section-subtitle {
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
    margin: 0;
  }

  .compact-wrap {
    max-width: 640px;
  }

  /* ── Props table ─────────────────────────────────────────────────────────── */
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

  .props-table thead th {
    padding: var(--af-space-2) var(--af-space-3);
    background: var(--af-surface-raised);
    color: var(--af-text-muted);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
    text-align: left;
    border-bottom: 1px solid var(--af-border-strong);
    white-space: nowrap;
  }

  .props-table tbody tr {
    border-bottom: 1px solid var(--af-border);
  }

  .props-table tbody tr:last-child {
    border-bottom: none;
  }

  .props-table td {
    padding: var(--af-space-2) var(--af-space-3);
    vertical-align: top;
  }

  .prop-name {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-sm);
    color: var(--af-accent);
    font-weight: var(--af-weight-medium);
    white-space: nowrap;
  }

  .prop-type {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xs);
    color: var(--af-text-secondary);
    white-space: nowrap;
  }

  .prop-req {
    font-size: var(--af-text-xs);
    color: var(--af-text-muted);
    text-align: center;
  }

  .prop-desc {
    color: var(--af-text-secondary);
    line-height: var(--af-leading-base);
    max-width: 340px;
  }
</style>
