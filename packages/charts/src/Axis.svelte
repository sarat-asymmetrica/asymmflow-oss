<script lang="ts">
  /**
   * Axis — token-styled SVG axis with optional gridlines.
   *
   * Accepts any d3 scale (linear, band, point, time) via the AnyScale
   * structural type. Render inside ChartFrame's children snippet:
   *
   *   <Axis scale={y} orient="left" innerWidth={ctx.innerWidth}
   *         innerHeight={ctx.innerHeight} grid format={formatCompact} />
   *
   * Grammar: axis text is --af-text-secondary at xs size; gridlines are
   * --af-border and SOLID — dashes are noise (§3). The domain line is drawn
   * only on the bottom axis; left axes rely on the gridlines.
   */

  import type { AnyScale } from './scales.js';

  interface Props {
    scale: AnyScale;
    orient: 'bottom' | 'left';
    innerWidth: number;
    innerHeight: number;
    /** Desired tick count (continuous scales only; band/point show all). */
    ticks?: number;
    format?: (value: never) => string;
    /** Draw gridlines across the plot area. */
    grid?: boolean;
    /** Axis label (units, dimension). */
    label?: string;
  }

  let {
    scale,
    orient,
    innerWidth,
    innerHeight,
    ticks = 5,
    format,
    grid = false,
    label,
  }: Props = $props();

  const tickValues = $derived(
    scale.ticks ? scale.ticks(ticks) : scale.domain(),
  );

  // Band/point scales center the tick within the step.
  const offset = $derived(scale.bandwidth ? scale.bandwidth() / 2 : 0);

  function pos(v: unknown): number {
    return (scale(v as never) ?? 0) + offset;
  }

  function text(v: unknown): string {
    return format ? format(v as never) : String(v);
  }
</script>

{#if orient === 'bottom'}
  <g class="axis" transform="translate(0, {innerHeight})">
    <line class="domain" x1="0" x2={innerWidth} y1="0" y2="0" />
    {#each tickValues as v (v)}
      <g transform="translate({pos(v)}, 0)">
        {#if grid}
          <line class="grid" y1={-innerHeight} y2="0" />
        {/if}
        <text class="tick" y="20" text-anchor="middle">{text(v)}</text>
      </g>
    {/each}
    {#if label}
      <text class="label" x={innerWidth} y="34" text-anchor="end">{label}</text>
    {/if}
  </g>
{:else}
  <g class="axis">
    {#each tickValues as v (v)}
      <g transform="translate(0, {pos(v)})">
        {#if grid}
          <line class="grid" x1="0" x2={innerWidth} />
        {/if}
        <text class="tick" x="-10" dy="0.32em" text-anchor="end">{text(v)}</text>
      </g>
    {/each}
    {#if label}
      <text class="label" x="-10" y="-14" text-anchor="end">{label}</text>
    {/if}
  </g>
{/if}

<style>
  .domain {
    stroke: var(--af-border-strong);
    stroke-width: 1;
  }

  .grid {
    stroke: var(--af-border);
    stroke-width: 1;
  }

  .tick {
    fill: var(--af-text-secondary);
    font-size: var(--af-text-xs);
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
  }

  .label {
    fill: var(--af-text-muted);
    font-size: var(--af-text-xs);
    font-family: var(--af-font-body);
    font-weight: var(--af-weight-medium);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
  }
</style>
