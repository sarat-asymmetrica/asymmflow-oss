<script lang="ts" module>
  export interface ChartMargin {
    top: number;
    right: number;
    bottom: number;
    left: number;
  }

  /** Geometry handed to children — everything a chart needs to scale itself. */
  export interface ChartContext {
    /** Full SVG width/height (px). */
    width: number;
    height: number;
    /** Plot area inside the margins — scales map into this. */
    innerWidth: number;
    innerHeight: number;
    margin: ChartMargin;
  }
</script>

<script lang="ts">
  /**
   * ChartFrame — the responsive stage every chart renders into.
   *
   * Owns the <svg>: width tracks the container (bind:clientWidth), height is
   * a prop. Children render inside a <g> already translated by the margin,
   * so chart code works purely in plot-area coordinates. The optional
   * `overlay` snippet renders in an HTML layer above the SVG (tooltips live
   * there — real elements, not foreignObject).
   *
   * A11y: role="img" with the title/description as the accessible name.
   * Charts are summaries — the actual figures belong in an adjacent table
   * or the surrounding page (constitution §2.6).
   */

  import type { Snippet } from 'svelte';

  interface Props {
    /** Chart height in px (width is fluid). */
    height?: number;
    margin?: Partial<ChartMargin>;
    /** Accessible name. Always provide one. */
    title: string;
    description?: string;
    /** SVG content, in plot-area coordinates. */
    children: Snippet<[ChartContext]>;
    /** HTML layer above the SVG (tooltips, annotations). */
    overlay?: Snippet<[ChartContext]>;
    class?: string;
  }

  let {
    height = 280,
    margin: marginProp,
    title,
    description,
    children,
    overlay,
    class: className = '',
  }: Props = $props();

  const margin: ChartMargin = $derived({
    top: 12,
    right: 16,
    bottom: 32,
    left: 48,
    ...marginProp,
  });

  let width = $state(0);

  const ctx: ChartContext = $derived({
    width,
    height,
    innerWidth: Math.max(0, width - margin.left - margin.right),
    innerHeight: Math.max(0, height - margin.top - margin.bottom),
    margin,
  });
</script>

<div class="af-chart-frame {className}" bind:clientWidth={width} style:height="{height}px">
  {#if width > 0}
    <svg
      {width}
      {height}
      viewBox="0 0 {width} {height}"
      role="img"
      aria-label={description ? `${title}. ${description}` : title}
    >
      <title>{title}</title>
      {#if description}
        <desc>{description}</desc>
      {/if}
      <g transform="translate({margin.left}, {margin.top})">
        {@render children(ctx)}
      </g>
    </svg>
    {#if overlay}
      <div class="overlay">
        {@render overlay(ctx)}
      </div>
    {/if}
  {/if}
</div>

<style>
  .af-chart-frame {
    position: relative;
    width: 100%;
  }

  svg {
    display: block;
    overflow: visible;
    font-family: var(--af-font-body);
  }

  .overlay {
    position: absolute;
    inset: 0;
    pointer-events: none;
  }
</style>
