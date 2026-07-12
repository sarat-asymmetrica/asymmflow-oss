<script lang="ts">
  import { SPACE_LADDER, TYPE_SCALE, onyxEther } from '@asymmflow/tokens';

  const surfaces = [
    { token: 'bg', label: 'Background · ether' },
    { token: 'surface', label: 'Surface' },
    { token: 'surface-raised', label: 'Surface raised' },
    { token: 'surface-sunken', label: 'Surface sunken' },
    { token: 'inverse-surface', label: 'Inverse · onyx' },
  ];

  const inks = [
    { token: 'text', label: 'Text' },
    { token: 'text-secondary', label: 'Text secondary' },
    { token: 'text-muted', label: 'Text muted' },
    { token: 'border', label: 'Border' },
    { token: 'border-strong', label: 'Border strong' },
  ];

  const accents = [
    { token: 'accent', label: 'Accent' },
    { token: 'accent-hover', label: 'Accent hover' },
    { token: 'accent-pressed', label: 'Accent pressed' },
    { token: 'success', label: 'Success' },
    { token: 'warning', label: 'Warning' },
    { token: 'danger', label: 'Danger' },
    { token: 'info', label: 'Info' },
  ];

  const typeRoles = [
    { cls: 'af-page-title', sample: 'Quarterly revenue, reconciled', name: '2xl · page title', size: TYPE_SCALE['2xl'] },
    { cls: 'af-section-title', sample: 'Outstanding invoices', name: 'lg · section title', size: TYPE_SCALE.lg },
    { cls: 'body', sample: 'The quotation was approved and routed to the costing sheet.', name: 'md · body', size: TYPE_SCALE.md },
    { cls: 'table-sample', sample: 'INV-2026-0411 · Gulf Equipment Trading WLL', name: 'sm · table', size: TYPE_SCALE.sm },
    { cls: 'af-label', sample: 'Payment status', name: 'xs · label', size: TYPE_SCALE.xs },
  ];

  const radii = [
    { token: '--af-radius-sm', label: 'sm · controls', px: '8px' },
    { token: '--af-radius-md', label: 'md · cards', px: '12px' },
    { token: '--af-radius-lg', label: 'lg · modals', px: '16px' },
    { token: '--af-radius-pill', label: 'pill', px: '999px' },
  ];
</script>

<div class="sections">
  <!-- ===== COLOR ===== -->
  <section>
    <h2 class="af-section-title">Color — one accent, derived neutrals</h2>
    <p class="intro">
      Sage-tinted monochrome. The neutral ramp is derived from the accent — nothing is pure
      black, white-on-grey, or random gray. Status colors exist but the default status
      language is monochrome iconography and weight.
    </p>

    <div class="af-label swatch-group-label">Surfaces</div>
    <div class="swatches">
      {#each surfaces as s}
        <div class="swatch">
          <div class="chip" style:background="var(--af-{s.token})"></div>
          <div class="swatch-meta">
            <span class="swatch-name">{s.label}</span>
            <code class="af-meta">--af-{s.token} · {onyxEther.tokens[s.token]}</code>
          </div>
        </div>
      {/each}
    </div>

    <div class="af-label swatch-group-label">Ink &amp; structure</div>
    <div class="swatches">
      {#each inks as s}
        <div class="swatch">
          <div class="chip" style:background="var(--af-{s.token})"></div>
          <div class="swatch-meta">
            <span class="swatch-name">{s.label}</span>
            <code class="af-meta">--af-{s.token} · {onyxEther.tokens[s.token]}</code>
          </div>
        </div>
      {/each}
    </div>

    <div class="af-label swatch-group-label">Accent &amp; status (spent rarely)</div>
    <div class="swatches">
      {#each accents as s}
        <div class="swatch">
          <div class="chip" style:background="var(--af-{s.token})"></div>
          <div class="swatch-meta">
            <span class="swatch-name">{s.label}</span>
            <code class="af-meta">--af-{s.token} · {onyxEther.tokens[s.token]}</code>
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- ===== TYPE ===== -->
  <section>
    <h2 class="af-section-title">Type — Space Grotesk × Rubik, scale ≈1.2</h2>
    <p class="intro">
      Display face carries titles and every financial figure (tabular numerals, always).
      Body face carries UI. Labels are the quiet backbone of enterprise density.
    </p>
    <div class="type-rows card">
      {#each typeRoles as role}
        <div class="type-row">
          <span class={role.cls}>{role.sample}</span>
          <code class="af-meta">{role.name} · {role.size}px</code>
        </div>
      {/each}
      <div class="type-row">
        <span class="af-numeric numeric-sample">BHD 1,284,932.500</span>
        <code class="af-meta">numeric · tabular-nums, always</code>
      </div>
    </div>
  </section>

  <!-- ===== SPACE ===== -->
  <section>
    <h2 class="af-section-title">Space — the φ ladder</h2>
    <p class="intro">
      Each step is the sum of the previous two; the ratio converges to φ (1.618…).
      Fibonacci rhythm, 4px-grid compatible. Ad-hoc margins are defects.
    </p>
    <div class="card ladder">
      {#each SPACE_LADDER as step, i}
        <div class="ladder-row">
          <code class="af-meta ladder-name">--af-space-{i + 1}</code>
          <div class="ladder-bar" style:width="{step}px"></div>
          <span class="af-meta af-numeric">{step}px</span>
        </div>
      {/each}
    </div>
  </section>

  <!-- ===== RADII & ELEVATION ===== -->
  <section>
    <h2 class="af-section-title">Radii &amp; elevation — borders at rest, the Lift on hover</h2>
    <p class="intro">
      Cards rest on a 1px border with no shadow. Hover earns a single soft ambient
      shadow ("the Lift" — hover the cards below). True shadows are reserved for layered surfaces.
    </p>
    <div class="radii-grid">
      {#each radii as r}
        <div class="card radius-card" style:border-radius="var({r.token})">
          <span class="swatch-name">{r.label}</span>
          <code class="af-meta">{r.px}</code>
        </div>
      {/each}
    </div>
    <div class="elevation-grid">
      <div class="card elevation-card">
        <span class="af-label">At rest</span>
        <p class="elevation-desc">1px border, no shadow</p>
      </div>
      <div class="card elevation-card lifted">
        <span class="af-label">The Lift (hover state)</span>
        <p class="elevation-desc">--af-shadow-lift</p>
      </div>
      <div class="card elevation-card overlaid">
        <span class="af-label">Layered surface</span>
        <p class="elevation-desc">--af-shadow-overlay</p>
      </div>
      <div class="card elevation-card north-star">
        <span class="af-label ns-label">North star</span>
        <p class="ns-value af-numeric">87.532%</p>
      </div>
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

  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
    transition: box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .card:hover {
    box-shadow: var(--af-shadow-lift);
  }

  /* Swatches */
  .swatch-group-label {
    margin: var(--af-space-4) 0 var(--af-space-2);
  }

  .swatches {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
    gap: var(--af-space-3);
  }

  .swatch {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-sm);
    padding: var(--af-space-2);
  }

  .chip {
    width: 44px;
    height: 44px;
    border-radius: var(--af-radius-sm);
    border: 1px solid var(--af-border);
    flex-shrink: 0;
  }

  .swatch-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .swatch-name {
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
  }

  .swatch-meta code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Type */
  .type-rows {
    display: flex;
    flex-direction: column;
  }

  .type-row {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--af-space-4);
    padding: var(--af-space-3) 0;
    border-bottom: 1px solid var(--af-border);
  }

  .type-row:last-child {
    border-bottom: none;
  }

  .body {
    font-size: var(--af-text-md);
  }

  .table-sample {
    font-size: var(--af-text-sm);
  }

  .numeric-sample {
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-semibold);
  }

  .type-row code {
    flex-shrink: 0;
  }

  /* Ladder */
  .ladder {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .ladder-row {
    display: grid;
    grid-template-columns: 110px 1fr 52px;
    align-items: center;
    gap: var(--af-space-3);
  }

  .ladder-bar {
    height: 16px;
    background: var(--af-accent);
    border-radius: 3px;
    opacity: 0.85;
  }

  /* Radii & elevation */
  .radii-grid,
  .elevation-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
    gap: var(--af-space-3);
  }

  .elevation-grid {
    margin-top: var(--af-space-3);
  }

  .radius-card {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-1);
  }

  .elevation-card {
    min-height: 110px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
  }

  .elevation-desc {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
  }

  .lifted {
    box-shadow: var(--af-shadow-lift);
  }

  .overlaid {
    box-shadow: var(--af-shadow-overlay);
    border: none;
  }

  .north-star {
    background: var(--af-inverse-surface);
    border: none;
  }

  .ns-label {
    color: var(--af-text-inverse);
    opacity: 0.6;
  }

  .ns-value {
    color: var(--af-text-inverse);
    font-size: var(--af-text-display);
    font-weight: var(--af-weight-bold);
    line-height: var(--af-leading-tight);
  }
</style>
