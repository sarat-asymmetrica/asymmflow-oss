<script lang="ts">
  /**
   * ChartTooltip — the HTML tooltip for chart hover states.
   *
   * Render inside ChartFrame's `overlay` snippet. Position with FULL-SVG
   * coordinates (px from the frame's top-left — add the margin to plot-area
   * coords). Inverse surface, xs type, R2 fade. Pointer-events: none — the
   * chart underneath keeps receiving hover.
   *
   * Clamps horizontally inside the frame and flips below the anchor when
   * too close to the top edge.
   */

  import type { Snippet } from 'svelte';

  interface Props {
    /** Anchor point in frame coordinates (px). */
    x: number;
    y: number;
    visible: boolean;
    /** Frame width (ctx.width) for horizontal clamping. */
    frameWidth: number;
    children: Snippet;
  }

  let { x, y, visible, frameWidth, children }: Props = $props();

  let tipW = $state(0);
  let tipH = $state(0);

  const GAP = 10;

  // Clamp the tooltip's left edge into the frame.
  const left = $derived(
    Math.max(4, Math.min(x - tipW / 2, frameWidth - tipW - 4)),
  );
  // Above the anchor by default; flip below when it would poke out the top.
  const top = $derived(y - tipH - GAP < 4 ? y + GAP : y - tipH - GAP);
</script>

<div
  class="af-chart-tooltip"
  class:visible
  style:left="{left}px"
  style:top="{top}px"
  bind:clientWidth={tipW}
  bind:clientHeight={tipH}
  role="status"
>
  {@render children()}
</div>

<style>
  .af-chart-tooltip {
    position: absolute;
    pointer-events: none;
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border-radius: var(--af-radius-sm);
    padding: var(--af-space-2) var(--af-space-3);
    font-size: var(--af-text-xs);
    line-height: var(--af-leading-base);
    box-shadow: var(--af-shadow-overlay);
    z-index: var(--af-z-tooltip);
    opacity: 0;
    transition: opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    max-width: 260px;
    white-space: nowrap;
  }

  .af-chart-tooltip.visible {
    opacity: 1;
  }
</style>
