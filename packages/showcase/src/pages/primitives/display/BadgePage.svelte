<script lang="ts">
  import { Badge, StatusBadge } from '@asymmflow/ui';
  import type { StatusKind } from '@asymmflow/ui';

  const badgeVariants = ['primary', 'neutral', 'outlined'] as const;
  const sizes = ['sm', 'md'] as const;

  const statuses: StatusKind[] = ['done', 'pending', 'attention', 'failed'];
</script>

<div class="sections">
  <section>
    <h2 class="af-section-title">Badge</h2>
    <p class="intro">
      Pill labels for categorical metadata. Three variants keep color restrained:
      <strong>primary</strong> (inverse — use once per context),
      <strong>neutral</strong> (default, most used),
      <strong>outlined</strong> (lightest touch).
    </p>
  </section>

  <!-- Badge variants -->
  <section>
    <div class="af-label section-label">Variants</div>
    <div class="row">
      {#each badgeVariants as v}
        <div class="variant-col">
          <div class="af-meta variant-name">{v}</div>
          <div class="row row--gap-2">
            {#each sizes as s}
              <Badge variant={v} size={s}>
                {s === 'sm' ? 'Small' : 'Medium'}
              </Badge>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  </section>

  <!-- Real-world labels -->
  <section>
    <div class="af-label section-label">Real-world label examples</div>
    <div class="row">
      <Badge variant="primary">New</Badge>
      <Badge variant="neutral">Draft</Badge>
      <Badge variant="neutral">Beta</Badge>
      <Badge variant="outlined">Read-only</Badge>
      <Badge variant="neutral" size="sm">v2.1.0</Badge>
      <Badge variant="primary" size="sm">Admin</Badge>
    </div>
  </section>

  <!-- StatusBadge divider -->
  <section>
    <h2 class="af-section-title">StatusBadge</h2>
    <p class="intro">
      Status indicators default to <strong>monochrome</strong> per constitution §4c —
      shape and weight carry the meaning, not color. Color is an explicit opt-in
      via <code>emphasis="color"</code>, spent rarely.
    </p>
  </section>

  <!-- Mono emphasis (default) -->
  <section>
    <div class="af-label section-label">emphasis="mono" — default, constitution compliant</div>
    <div class="demo-card">
      <div class="row">
        {#each statuses as s}
          <StatusBadge status={s} emphasis="mono" />
        {/each}
      </div>
      <div class="row" style:margin-top="var(--af-space-2)">
        {#each statuses as s}
          <StatusBadge status={s} emphasis="mono" size="sm" />
        {/each}
      </div>
    </div>
  </section>

  <!-- Color emphasis -->
  <section>
    <div class="af-label section-label">emphasis="color" — opt-in, use sparingly</div>
    <div class="demo-card">
      <div class="row">
        {#each statuses as s}
          <StatusBadge status={s} emphasis="color" />
        {/each}
      </div>
      <div class="row" style:margin-top="var(--af-space-2)">
        {#each statuses as s}
          <StatusBadge status={s} emphasis="color" size="sm" />
        {/each}
      </div>
    </div>
  </section>

  <!-- Custom labels -->
  <section>
    <div class="af-label section-label">Custom labels via label prop</div>
    <div class="demo-card">
      <div class="row">
        <StatusBadge status="done" label="Reconciled" />
        <StatusBadge status="pending" label="Awaiting approval" />
        <StatusBadge status="attention" label="Review required" />
        <StatusBadge status="failed" label="Rejected" />
      </div>
    </div>
  </section>
</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-5);
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

  .row {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--af-space-2);
  }

  .row--gap-2 {
    gap: var(--af-space-2);
  }

  .variant-col {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .variant-name {
    font-style: italic;
  }

  .demo-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }
</style>
