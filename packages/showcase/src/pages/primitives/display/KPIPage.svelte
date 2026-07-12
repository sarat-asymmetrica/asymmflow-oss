<script lang="ts">
  import { KPICard } from '@asymmflow/ui';
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">KPICard</h2>
    <p class="intro">
      Headline metrics. All financial figures use <code>.af-numeric</code> — tabular-nums, always.
      The <code>north-star</code> variant is the single hero KPI: display-size value on inverse
      surface. Use it once per dashboard to anchor the reader's eye.
    </p>
  </section>

  <!-- Realistic dashboard row: north-star + 3 KPIs -->
  <section>
    <div class="af-label section-label">Realistic dashboard row — Acme Instrumentation, Bahrain</div>
    <div class="kpi-grid">
      <KPICard
        variant="north-star"
        label="Total revenue, June"
        value="BHD 1,284,932"
        meta="Reconciled · all entities"
      />

      <KPICard
        label="Accounts receivable"
        value="BHD 312,450"
        meta="42 open invoices"
      >
        {#snippet trend()}
          <!-- Trend up — muted, monochrome, no raw color -->
          <svg viewBox="0 0 16 10" width="16" height="10" fill="none" aria-hidden="true">
            <path d="M1 9L5 4.5L8.5 7L15 1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M11 1h4v4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <span style:margin-inline-start="4px">+8.3%</span>
        {/snippet}
      </KPICard>

      <KPICard
        label="Outstanding payables"
        value="BHD 97,110"
        meta="18 due within 7 days"
      >
        {#snippet trend()}
          <svg viewBox="0 0 16 10" width="16" height="10" fill="none" aria-hidden="true">
            <path d="M1 1L5 5.5L8.5 3L15 9" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M11 9h4V5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <span style:margin-inline-start="4px">-2.1%</span>
        {/snippet}
      </KPICard>

      <KPICard
        label="Collection rate"
        value="94.7%"
        meta="30-day rolling average"
      />
    </div>
  </section>

  <!-- Variants explainer -->
  <section>
    <div class="af-label section-label">Variant comparison</div>
    <div class="grid-2">
      <KPICard
        variant="default"
        label="Default variant"
        value="BHD 84,320"
        meta="Standard surface card"
      />
      <KPICard
        variant="north-star"
        label="North-star variant"
        value="87.532%"
        meta="Display-size on inverse surface"
      />
    </div>
  </section>

  <!-- Minimal / no meta -->
  <section>
    <div class="af-label section-label">Minimal — label and value only</div>
    <div class="kpi-grid">
      <KPICard label="Transactions" value="2,841" />
      <KPICard label="Vendors" value="134" />
      <KPICard label="Avg invoice" value="BHD 4,502" />
      <KPICard label="On-time delivery" value="98.2%" />
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
    max-width: 64ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  .section-label {
    margin-block-end: var(--af-space-3);
  }

  /* Dashboard row: north-star spans 2 columns on wide, 1 on narrow */
  .kpi-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: var(--af-grid-gap);
  }

  /* Make north-star span wider when grid allows */
  .kpi-grid :global(.af-kpi--north-star) {
    grid-column: span 1;
  }

  @media (min-width: 640px) {
    .kpi-grid :global(.af-kpi--north-star) {
      grid-column: span 2;
    }
  }

  .grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--af-grid-gap);
  }
</style>
