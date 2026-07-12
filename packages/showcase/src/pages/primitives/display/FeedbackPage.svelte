<script lang="ts">
  import { Spinner, Skeleton, EmptyState } from '@asymmflow/ui';

  let emptyKey = $state(0);
</script>

<div class="sections">
  <!-- SPINNER -->
  <section>
    <h2 class="af-section-title">Spinner</h2>
    <p class="intro">
      A quiet, sophisticated mark. A ring with a 137.5° golden-angle arc rotating on a
      continuous linear loop — the one allowed linear easing. <code>currentColor</code>
      so it inherits from context. Three sizes. Under
      <code>prefers-reduced-motion</code>, rotation becomes a gentle opacity pulse.
    </p>

    <div class="demo-card">
      <div class="spinner-row">
        <div class="spinner-cell">
          <Spinner size="sm" />
          <span class="af-meta">sm · 16px</span>
        </div>
        <div class="spinner-cell">
          <Spinner size="md" />
          <span class="af-meta">md · 24px</span>
        </div>
        <div class="spinner-cell">
          <Spinner size="lg" />
          <span class="af-meta">lg · 40px</span>
        </div>
      </div>
    </div>
  </section>

  <!-- Spinner in context -->
  <section>
    <div class="af-label section-label">In context — inherits currentColor</div>
    <div class="grid-2">
      <div class="demo-card context-default">
        <Spinner size="sm" />
        <span class="af-meta" style:margin-inline-start="var(--af-space-2)">Loading invoices…</span>
      </div>
      <div class="demo-card context-muted" style:color="var(--af-text-muted)">
        <Spinner size="sm" />
        <span class="af-meta" style:margin-inline-start="var(--af-space-2)">color: text-muted</span>
      </div>
      <div class="demo-card context-inverse" style:background="var(--af-inverse-surface)" style:color="var(--af-text-inverse)">
        <Spinner size="sm" />
        <span class="af-meta" style:margin-inline-start="var(--af-space-2)">on inverse surface</span>
      </div>
      <div class="demo-card context-accent" style:color="var(--af-accent)">
        <Spinner size="sm" />
        <span class="af-meta" style:margin-inline-start="var(--af-space-2)">color: accent</span>
      </div>
    </div>
  </section>

  <!-- SKELETON -->
  <section>
    <h2 class="af-section-title">Skeleton</h2>
    <p class="intro">
      Shimmer loading placeholders. Three variants: <code>text</code> (pill, inline),
      <code>rect</code> (block), <code>circle</code>. Compose freely to mirror real
      layout shapes. The shimmer uses tokens only — no raw hex in the gradient.
    </p>
  </section>

  <!-- Skeleton: card composition -->
  <section>
    <div class="af-label section-label">Composed — KPI card skeleton</div>
    <div class="grid-3">
      {#each [1, 2, 3] as _}
        <div class="demo-card skeleton-card">
          <Skeleton variant="text" width="60%" height="11px" />
          <Skeleton variant="text" width="80%" height="29px" style="margin-top: var(--af-space-2)" />
          <Skeleton variant="text" width="50%" height="11px" style="margin-top: var(--af-space-2)" />
        </div>
      {/each}
    </div>
  </section>

  <!-- Skeleton: table row -->
  <section>
    <div class="af-label section-label">Composed — table row skeleton</div>
    <div class="demo-card">
      {#each [1, 2, 3, 4] as _}
        <div class="skeleton-row">
          <Skeleton variant="circle" width="32px" height="32px" />
          <Skeleton variant="text" width="140px" height="13px" />
          <Skeleton variant="text" width="80px" height="13px" />
          <Skeleton variant="text" width="100px" height="13px" />
          <Skeleton variant="rect" width="60px" height="20px" style="border-radius: var(--af-radius-pill)" />
        </div>
      {/each}
    </div>
  </section>

  <!-- Skeleton: variants raw -->
  <section>
    <div class="af-label section-label">Variants — text / rect / circle</div>
    <div class="demo-card variant-grid">
      <div class="variant-group">
        <span class="af-meta">text</span>
        <Skeleton variant="text" width="100%" />
        <Skeleton variant="text" width="75%" />
        <Skeleton variant="text" width="85%" />
      </div>
      <div class="variant-group">
        <span class="af-meta">rect</span>
        <Skeleton variant="rect" height="60px" />
      </div>
      <div class="variant-group">
        <span class="af-meta">circle</span>
        <div style:display="flex" style:gap="var(--af-space-2)">
          <Skeleton variant="circle" width="32px" height="32px" />
          <Skeleton variant="circle" width="48px" height="48px" />
          <Skeleton variant="circle" width="64px" height="64px" />
        </div>
      </div>
    </div>
  </section>

  <!-- EMPTY STATE -->
  <section>
    <h2 class="af-section-title">EmptyState</h2>
    <p class="intro">
      Calm, generous whitespace. Icon Snippet + title + description + optional
      action. R1 explore entrance on mount. Reserved for the Edges zone — it
      earns love via patient, quiet presentation.
    </p>
  </section>

  <section>
    <div class="af-label section-label">
      Default — with icon, description, and action
      <button class="replay" onclick={() => emptyKey++}>Replay entrance</button>
    </div>
    {#key emptyKey}
      <EmptyState
        title="No invoices yet"
        description="Once you create your first invoice, it will appear here. Gulf Equipment Trading WLL is ready."
      >
        {#snippet icon()}
          <!-- Minimal monochrome inbox mark — no emoji, no icon lib -->
          <svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
            <rect x="8" y="14" width="32" height="24" rx="3" stroke="currentColor" stroke-width="1.5"/>
            <path d="M8 20l16 10 16-10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        {/snippet}
        {#snippet action()}
          <button class="empty-btn">Create first invoice</button>
        {/snippet}
      </EmptyState>
    {/key}
  </section>

  <!-- EmptyState minimal -->
  <section>
    <div class="af-label section-label">Minimal — title only</div>
    <div class="demo-card" style:background="var(--af-surface-raised)">
      <EmptyState title="No results found" />
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
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
  }

  .demo-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  /* Spinner */
  .spinner-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-6);
  }

  .spinner-cell {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--af-space-2);
  }

  .grid-2 {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--af-grid-gap);
  }

  .context-default,
  .context-muted,
  .context-inverse,
  .context-accent {
    display: flex;
    align-items: center;
    min-height: 52px;
  }

  /* Skeleton */
  .grid-3 {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: var(--af-grid-gap);
  }

  .skeleton-card {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .skeleton-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    padding: var(--af-space-3) 0;
    border-block-end: 1px solid var(--af-border);
  }

  .skeleton-row:last-child {
    border-block-end: none;
  }

  .variant-grid {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: var(--af-space-4);
  }

  .variant-group {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  /* Empty state action */
  .empty-btn {
    border: 1px solid var(--af-border-strong);
    background: transparent;
    color: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-2) var(--af-space-4);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    min-height: 44px;
  }

  .empty-btn:hover {
    background: var(--af-tint);
    border-color: var(--af-inverse-surface);
  }

  .replay {
    border: 1px solid var(--af-border);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    padding: var(--af-space-1) var(--af-space-3);
    border-radius: var(--af-radius-sm);
    cursor: pointer;
    transition:
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .replay:hover {
    border-color: var(--af-border-strong);
    background: var(--af-surface-raised);
  }
</style>
