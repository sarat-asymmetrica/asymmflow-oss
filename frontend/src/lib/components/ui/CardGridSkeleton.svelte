<script lang="ts">
  /**
   * CardGridSkeleton — content-shaped loading placeholder for a stat-card row
   * plus a panel grid (dashboards, financial summaries). Reserves the same
   * footprint as the real content so there is zero layout shift on load.
   * Shimmer is CSS animation only; killed by the global prefers-reduced-motion
   * reset in design-tokens.css.
   */
  interface Props {
    /** Number of stat cards in the top row. */
    statCards?: number;
    /** Number of panels in the grid below the stat row. */
    panels?: number;
    /** Rows of shimmer lines rendered inside each panel (list/chart placeholder). */
    panelRows?: number;
    /** Panel grid column count. */
    panelCols?: number;
  }

  let { statCards = 4, panels = 4, panelRows = 4, panelCols = 2 }: Props = $props();
</script>

<div class="cg-skeleton" role="status" aria-label="Loading dashboard">
  {#if statCards > 0}
    <div class="cg-stat-row" style="grid-template-columns: repeat({statCards}, 1fr);">
      {#each Array(statCards) as _, i (i)}
        <div class="cg-card">
          <div class="cg-block cg-shimmer" style="width: 60%; height: 12px;"></div>
          <div class="cg-block cg-shimmer" style="width: 40%; height: 24px; margin-top: 10px;"></div>
        </div>
      {/each}
    </div>
  {/if}

  <div class="cg-panel-grid" style="grid-template-columns: repeat({panelCols}, 1fr);">
    {#each Array(panels) as _, p (p)}
      <div class="cg-panel">
        <div class="cg-block cg-shimmer" style="width: 45%; height: 14px; margin-bottom: 16px;"></div>
        {#each Array(panelRows) as _, r (r)}
          <div
            class="cg-block cg-shimmer cg-panel-line"
            style="width: {r % 2 === 0 ? '90%' : '70%'};"
          ></div>
        {/each}
      </div>
    {/each}
  </div>
</div>

<style>
  .cg-skeleton {
    display: flex;
    flex-direction: column;
    gap: 20px;
    width: 100%;
  }

  .cg-stat-row {
    display: grid;
    gap: 16px;
  }

  .cg-card {
    padding: 16px;
    border-radius: var(--border-radius);
    background: var(--surface);
    border: var(--border-width, 1px) solid var(--border);
  }

  .cg-panel-grid {
    display: grid;
    gap: 16px;
  }

  .cg-panel {
    padding: 20px;
    border-radius: var(--border-radius);
    background: var(--surface);
    border: var(--border-width, 1px) solid var(--border);
    min-height: 160px;
  }

  .cg-panel-line {
    height: 12px;
    margin-bottom: 12px;
  }

  .cg-block {
    border-radius: var(--border-radius-sm);
    background: var(--surface-elevated);
    position: relative;
    overflow: hidden;
  }

  .cg-shimmer::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(255, 255, 255, 0.55) 50%,
      transparent 100%
    );
    animation: cg-shimmer-sweep 1.6s ease-in-out infinite;
    transform: translateX(-100%);
  }

  @keyframes cg-shimmer-sweep {
    100% {
      transform: translateX(100%);
    }
  }
</style>
