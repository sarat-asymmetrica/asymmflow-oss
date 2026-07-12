<script lang="ts">
  /**
   * PipelineFunnelPage — ERP showcase: sales pipeline funnel.
   *
   * Demonstrates:
   * - Full 5-stage pipeline: Leads → Qualified → Quoted → Negotiation → Won
   * - Randomize mid-flight: proves interruptible tween
   * - showConversion off variant
   * - Minimal 3-stage case
   *
   * Data: Khalifa Logistics Gulf-trading CRM narrative, BHD.
   */

  import { PipelineFunnel } from '@asymmflow/charts';
  import { Button } from '@asymmflow/ui';

  // ─── Main pipeline data ────────────────────────────────────────────────────

  interface Stage {
    label: string;
    value: number;
  }

  const pipelineBase: Stage[] = [
    { label: 'Leads',        value: 240 },
    { label: 'Qualified',    value: 150 },
    { label: 'Quoted',       value: 96  },
    { label: 'Negotiation',  value: 51  },
    { label: 'Won',          value: 32  },
  ];

  let pipeline = $state<Stage[]>(pipelineBase);

  // ─── Minimal 3-stage ──────────────────────────────────────────────────────

  const minimalStages: Stage[] = [
    { label: 'Prospects',  value: 80 },
    { label: 'Proposals',  value: 24 },
    { label: 'Closed',     value: 11 },
  ];

  // ─── Randomize ────────────────────────────────────────────────────────────

  function randomize() {
    const leads = Math.round(180 + Math.random() * 120);
    const qualified = Math.round(leads * (0.4 + Math.random() * 0.3));
    const quoted = Math.round(qualified * (0.4 + Math.random() * 0.3));
    const neg = Math.round(quoted * (0.3 + Math.random() * 0.3));
    const won = Math.round(neg * (0.4 + Math.random() * 0.3));

    pipeline = [
      { label: 'Leads',       value: leads },
      { label: 'Qualified',   value: qualified },
      { label: 'Quoted',      value: quoted },
      { label: 'Negotiation', value: neg },
      { label: 'Won',         value: won },
    ];
  }

  // ─── Toggle states ─────────────────────────────────────────────────────────

  let showConversion = $state(true);

  // ─── Props table ──────────────────────────────────────────────────────────

  const propRows = [
    { name: 'stages',         type: '{ label: string; value: number }[]', required: true,  desc: 'Stage data — first stage is 100% width reference.' },
    { name: 'title',          type: 'string',                             required: true,  desc: 'Accessible SVG title.' },
    { name: 'height',         type: 'number',                             required: false, desc: 'Chart height in px. Default 300.' },
    { name: 'valueFormat',    type: '(n: number) => string',              required: false, desc: 'Override value label format. Default formatCompact.' },
    { name: 'description',    type: 'string',                             required: false, desc: 'SVG <desc> for extended a11y context.' },
    { name: 'showConversion', type: 'boolean',                            required: false, desc: 'Show "64% →" badges between stages. Default true.' },
  ];
</script>

<div class="page">
  <header class="page-header">
    <div>
      <h1 class="page-title">PipelineFunnel</h1>
      <p class="page-desc">
        Horizontal sales pipeline funnel. One accent color with stepping opacity —
        monochrome confidence as stages narrow. Conversion rate badges between stages.
        Interruptible geodesic tween on data updates.
      </p>
    </div>
  </header>

  <!-- ─── Demo: Full 5-stage pipeline ───────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Sales Pipeline — Khalifa Logistics</h2>
      <p class="section-subtitle">Gulf construction sector · June 2026</p>
    </div>

    <div class="demo-controls">
      <Button variant="secondary" onclick={randomize}>Randomize (mid-flight safe)</Button>
      <label class="toggle-label">
        <input type="checkbox" bind:checked={showConversion} />
        Show conversion rates
      </label>
    </div>

    <div class="chart-card">
      <PipelineFunnel
        stages={pipeline}
        title="Sales Pipeline — Khalifa Logistics, June 2026"
        description="5-stage CRM pipeline from initial leads through to won opportunities."
        height={300}
        {showConversion}
      />
    </div>
  </section>

  <!-- ─── Demo: showConversion off ─────────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Without Conversion Badges</h2>
      <p class="section-subtitle">Clean bands-only view for dense dashboards</p>
    </div>

    <div class="chart-card">
      <PipelineFunnel
        stages={pipelineBase}
        title="Pipeline — Clean View"
        height={280}
        showConversion={false}
      />
    </div>
  </section>

  <!-- ─── Demo: Minimal 3-stage ─────────────────────────────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">Minimal — 3 Stages</h2>
      <p class="section-subtitle">Manama Retail Group · simplified view</p>
    </div>

    <div class="chart-card chart-card--compact">
      <PipelineFunnel
        stages={minimalStages}
        title="Minimal Pipeline — Manama Retail Group"
        height={200}
      />
    </div>
  </section>

  <!-- ─── Demo: High-volume — proves the opacity ramp ──────────────────── -->
  <section class="demo-section">
    <div class="section-header">
      <h2 class="section-title af-label">High-Volume Pipeline — BHD Values</h2>
      <p class="section-subtitle">Gulf Construction Co · deal values in BHD thousands</p>
    </div>

    <div class="chart-card">
      <PipelineFunnel
        stages={[
          { label: 'Identified',   value: 2_840_000 },
          { label: 'Scoped',       value: 1_620_000 },
          { label: 'Proposal',     value:   890_000 },
          { label: 'Legal Review', value:   410_000 },
          { label: 'Contracted',   value:   198_000 },
        ]}
        title="Construction Deals Pipeline — Gulf Construction Co"
        height={300}
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
