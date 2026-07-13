<script lang="ts">
  /**
   * TableSkeleton — content-shaped loading placeholder for a header bar + N shimmer rows.
   * Reserves the same footprint the real table/list will occupy (zero layout shift).
   * Shimmer is CSS animation only; the global prefers-reduced-motion reset (design-tokens.css)
   * collapses animation-duration to ~0, so it renders static under reduced motion.
   */
  interface Props {
    rows?: number;
    cols?: number;
    /** Render a filter-bar placeholder above the header (for screens with a filter row). */
    showFilterBar?: boolean;
  }

  let { rows = 6, cols = 5, showFilterBar = false }: Props = $props();
</script>

<div class="table-skeleton" role="status" aria-label="Loading data">
  {#if showFilterBar}
    <div class="ts-filter-bar">
      <div class="ts-block ts-shimmer" style="width: 220px; height: 34px;"></div>
      <div class="ts-block ts-shimmer" style="width: 140px; height: 34px;"></div>
      <div class="ts-block ts-shimmer" style="width: 140px; height: 34px;"></div>
    </div>
  {/if}

  <div class="ts-header" style="grid-template-columns: repeat({cols}, 1fr);">
    {#each Array(cols) as _, c (c)}
      <div class="ts-block ts-shimmer ts-header-cell" style="width: {c === 0 ? '60%' : '45%'};"></div>
    {/each}
  </div>

  <div class="ts-rows">
    {#each Array(rows) as _, r (r)}
      <div class="ts-row" style="grid-template-columns: repeat({cols}, 1fr);">
        {#each Array(cols) as _, c (c)}
          <div
            class="ts-block ts-shimmer ts-cell"
            style="width: {c === 0 ? '80%' : '55%'};"
          ></div>
        {/each}
      </div>
    {/each}
  </div>
</div>

<style>
  .table-skeleton {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
  }

  .ts-filter-bar {
    display: flex;
    gap: 12px;
    padding-bottom: 12px;
    margin-bottom: 4px;
    border-bottom: var(--border-width, 1px) solid var(--border);
  }

  .ts-header,
  .ts-row {
    display: grid;
    gap: 16px;
    align-items: center;
    padding: 12px 8px;
  }

  .ts-header {
    border-bottom: var(--border-width, 1px) solid var(--border);
  }

  .ts-row {
    border-bottom: 1px solid var(--surface-elevated);
  }

  .ts-header-cell {
    height: 12px;
  }

  .ts-cell {
    height: 14px;
  }

  .ts-block {
    border-radius: var(--border-radius-sm);
    background: var(--surface-elevated);
    position: relative;
    overflow: hidden;
  }

  .ts-shimmer::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(255, 255, 255, 0.55) 50%,
      transparent 100%
    );
    animation: ts-shimmer-sweep 1.6s ease-in-out infinite;
    transform: translateX(-100%);
  }

  @keyframes ts-shimmer-sweep {
    100% {
      transform: translateX(100%);
    }
  }
</style>
