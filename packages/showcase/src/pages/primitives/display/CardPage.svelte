<script lang="ts">
  import { Card } from '@asymmflow/ui';
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">Card</h2>
    <p class="intro">
      Surface containers. 1px border at rest, no shadow. The Lift (opt-in via <code>lift</code>)
      adds a single soft ambient shadow on hover. Variants express hierarchy, not decoration.
      Header and footer Snippets add structured regions.
    </p>
  </section>

  <!-- Variants -->
  <section>
    <div class="af-label section-label">Variants</div>
    <div class="grid grid--4">
      <Card variant="default" lift>
        {#snippet header()}
          <span class="af-label">Default</span>
        {/snippet}
        <p class="body-copy">Standard surface. 1px border at rest. Hover the Lift.</p>
      </Card>

      <Card variant="raised" lift>
        {#snippet header()}
          <span class="af-label">Raised</span>
        {/snippet}
        <p class="body-copy">Surface-raised background — elevated panels.</p>
      </Card>

      <Card variant="accent" lift>
        {#snippet header()}
          <span class="af-label">Accent</span>
        {/snippet}
        <p class="body-copy">2px inline-start border in onyx. Editorial emphasis, no color.</p>
      </Card>

      <Card variant="north-star">
        {#snippet header()}
          <span class="af-label" style:opacity="0.6" style:color="var(--af-text-inverse)">North Star</span>
        {/snippet}
        <p class="body-copy" style:color="var(--af-text-inverse)">
          Inverse surface. The single hero KPI card. Use once per dashboard.
        </p>
      </Card>
    </div>
  </section>

  <!-- Header + Footer -->
  <section>
    <div class="af-label section-label">With header and footer Snippets</div>
    <div class="grid grid--2">
      <Card lift>
        {#snippet header()}
          <div class="card-header-row">
            <span class="af-section-title" style:font-size="var(--af-text-lg)">Invoice detail</span>
            <span class="af-meta">INV-2026-0411</span>
          </div>
        {/snippet}
        <p class="body-copy" style:padding-block="var(--af-space-2)">
          Gulf Equipment Trading WLL · BHD 12,450.000 · 30-day net
        </p>
        {#snippet footer()}
          <div class="card-footer-row">
            <span class="af-meta">Created 11 Jun 2026</span>
            <span class="af-meta">Due 11 Jul 2026</span>
          </div>
        {/snippet}
      </Card>

      <Card variant="accent" lift>
        {#snippet header()}
          <span class="af-section-title" style:font-size="var(--af-text-lg)">Attention required</span>
        {/snippet}
        <p class="body-copy" style:padding-block="var(--af-space-2)">
          3 overdue payables require sign-off before the week close.
        </p>
        {#snippet footer()}
          <span class="af-meta">Finance · Accounts Payable</span>
        {/snippet}
      </Card>
    </div>
  </section>

  <!-- Padding override -->
  <section>
    <div class="af-label section-label">Padding variants via prop (φ ladder steps)</div>
    <div class="grid grid--3">
      <Card padding={2}>
        <span class="af-label">padding=2 (8px)</span>
        <p class="body-copy">Dense. For compact table cards or chips.</p>
      </Card>
      <Card padding={4}>
        <span class="af-label">padding=4 (20px, default)</span>
        <p class="body-copy">Default card padding token.</p>
      </Card>
      <Card padding={5}>
        <span class="af-label">padding=5 (32px)</span>
        <p class="body-copy">Generous. Dashboard panels, onboarding cards.</p>
      </Card>
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

  .grid {
    display: grid;
    gap: var(--af-grid-gap);
  }

  .grid--2 { grid-template-columns: repeat(2, 1fr); }
  .grid--3 { grid-template-columns: repeat(3, 1fr); }
  .grid--4 { grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); }

  .body-copy {
    font-size: var(--af-text-sm);
    color: var(--af-text-secondary);
    line-height: var(--af-leading-base);
    margin: 0;
  }

  .card-header-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: var(--af-space-3);
  }

  .card-footer-row {
    display: flex;
    justify-content: space-between;
  }
</style>
