<script lang="ts">
  /**
   * CashflowBridgePage — ERP showcase: monthly cashflow bridge.
   *
   * Demonstrates:
   * - Monthly bridge: Opening Balance → Collections → Supplier Payments →
   *   Payroll → Rent & Utilities → VAT → Closing Balance
   * - Quarter-view compact example
   * - Randomize button proving mid-flight interruptible tween
   * - showValues + showConnectors toggles
   *
   * Data: Al Mahmood Trading — Gulf-trading narrative, BHD.
   */

  import { CashflowBridge } from '@asymmflow/charts';
  import { Button } from '@asymmflow/ui';

  // ─── Monthly bridge data (May 2026, Al Mahmood Trading) ──────────────────

  interface BridgeItem {
    label: string;
    value: number;
    isTotal?: boolean;
  }

  const monthlyBase: BridgeItem[] = [
    { label: 'Opening',    value: 42_800,  isTotal: true },
    { label: 'Collections', value: 38_450 },
    { label: 'Supplier Pmts', value: -22_100 },
    { label: 'Payroll',    value: -14_250 },
    { label: 'Rent & Utils', value: -3_600 },
    { label: 'VAT',        value: -1_890 },
    { label: 'Closing',    value: 39_410, isTotal: true },
  ];

  // The tween is driven by the component; we just swap the data.
  let monthly = $state<BridgeItem[]>(monthlyBase);

  // ─── Quarter-view: Q2 2026 bridge ────────────────────────────────────────

  const quarterlyItems: BridgeItem[] = [
    { label: 'Q2 Open',   value: 31_500, isTotal: true },
    { label: 'Net Revenue', value: 67_200 },
    { label: 'Costs',     value: -48_400 },
    { label: 'Q2 Close',  value: 50_300, isTotal: true },
  ];

  // ─── Randomize ────────────────────────────────────────────────────────────

  function randomize() {
    const opening = Math.round(25_000 + Math.random() * 30_000);
    const collections = Math.round(20_000 + Math.random() * 30_000);
    const supplier = -Math.round(10_000 + Math.random() * 20_000);
    const payroll = -Math.round(8_000 + Math.random() * 10_000);
    const rent = -Math.round(2_000 + Math.random() * 4_000);
    const vat = -Math.round(500 + Math.random() * 2_000);
    const closing = opening + collections + supplier + payroll + rent + vat;

    monthly = [
      { label: 'Opening',     value: opening, isTotal: true },
      { label: 'Collections', value: collections },
      { label: 'Supplier Pmts', value: supplier },
      { label: 'Payroll',     value: payroll },
      { label: 'Rent & Utils', value: rent },
      { label: 'VAT',         value: vat },
      { label: 'Closing',     value: closing, isTotal: true },
    ];
  }

  // ─── Toggle states ─────────────────────────────────────────────────────────

  let showValues = $state(true);
  let showConnectors = $state(true);

  // ─── Props table rows ──────────────────────────────────────────────────────

  const propRows = [
    { name: 'items',          type: '{ label: string; value: number; isTotal?: boolean }[]', required: true,  desc: 'Bar data — totals span from zero, others build running cumulative.' },
    { name: 'title',          type: 'string', required: true,  desc: 'Accessible SVG title (a11y name).' },
    { name: 'height',         type: 'number', required: false, desc: 'Chart height in px. Default 300.' },
    { name: 'currency',       type: 'string', required: false, desc: 'ISO currency code for default format. Default "BHD".' },
    { name: 'valueFormat',    type: '(n: number) => string', required: false, desc: 'Override label/tooltip formatting.' },
    { name: 'description',    type: 'string', required: false, desc: 'SVG <desc> for extended a11y context.' },
    { name: 'showConnectors', type: 'boolean', required: false, desc: 'Draw level-to-level connector lines. Default true.' },
    { name: 'showValues',     type: 'boolean', required: false, desc: 'Render signed values atop each bar. Default true.' },
  ];
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1 class="page-title">CashflowBridge</h1>
      <p class="page-desc">
        Waterfall chart for cashflow analysis. Positive flows rise in success green,
        negative flows fall in danger red, total bars span from zero in inverse surface.
        Semantic color IS the encoding — constitution §4c permits this.
      </p>
    </div>
  </header>

  <!-- ─── Demo: Monthly bridge ───────────────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Monthly Bridge — May 2026</h2>
      <p class="section-subtitle">Al Mahmood Trading, Manama · BHD</p>
    </div>

    <div class="demo-controls">
      <Button variant="secondary" onclick={randomize}>Randomize (mid-flight safe)</Button>
      <label class="toggle-label">
        <input type="checkbox" bind:checked={showValues} />
        Show values
      </label>
      <label class="toggle-label">
        <input type="checkbox" bind:checked={showConnectors} />
        Show connectors
      </label>
    </div>

    <div class="chart-card">
      <CashflowBridge
        items={monthly}
        title="May 2026 Cashflow Bridge — Al Mahmood Trading"
        description="Monthly cashflow waterfall: opening balance through to closing balance including all outflows."
        height={320}
        {showValues}
        {showConnectors}
      />
    </div>
  </section>

  <!-- ─── Demo: Quarter-view ─────────────────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Quarterly View — Q2 2026</h2>
      <p class="section-subtitle">Compact 4-bar summary · Gulf Construction Co</p>
    </div>

    <div class="chart-card chart-card--compact">
      <CashflowBridge
        items={quarterlyItems}
        title="Q2 2026 Cashflow Bridge — Gulf Construction Co"
        height={220}
      />
    </div>
  </section>

  <!-- ─── Demo: Negative balance scenario ─────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Stress Scenario (Negative Balance)</h2>
      <p class="section-subtitle">Proves zero-line renders correctly · Bahrain Steel</p>
    </div>

    <div class="chart-card">
      <CashflowBridge
        items={[
          { label: 'Opening',  value: 8_200, isTotal: true },
          { label: 'Revenue',  value: 12_400 },
          { label: 'Costs',    value: -24_800 },
          { label: 'Payroll',  value: -9_500 },
          { label: 'Closing',  value: -13_700, isTotal: true },
        ]}
        title="Stress Scenario — Bahrain Steel"
        height={260}
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

  .demo-controls {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    flex-wrap: wrap;
  }

  .toggle-label {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    cursor: pointer;
    user-select: none;
  }

  .toggle-label input[type='checkbox'] {
    accent-color: var(--af-accent);
    width: 14px;
    height: 14px;
    cursor: pointer;
  }

  .chart-card {
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-space-4);
    background: var(--af-surface);
  }

  .chart-card--compact {
    max-width: 520px;
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

  @media (prefers-reduced-motion: reduce) {
    .toggle-label {
      transition: none;
    }
  }
</style>
