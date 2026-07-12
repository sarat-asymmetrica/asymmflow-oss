<script lang="ts">
  /**
   * BarChart — grouped or stacked bars, horizontal or vertical.
   *
   * Series colors come from the --af-chart-N token palette. Layout is computed
   * with d3-scale (scaleBand + scaleLinear for vertical; mirrored for
   * horizontal). Value transitions use the geodesic tween engine so rapid data
   * changes retarget smoothly mid-flight. Entrance bars rise from the baseline,
   * staggered ≤12 items at a time.
   *
   * Constitution: packages/DESIGN_CONSTITUTION.md.
   */

  import type { Snippet } from 'svelte';
  import ChartFrame from '../ChartFrame.svelte';
  import Axis from '../Axis.svelte';
  import Legend from '../Legend.svelte';
  import ChartTooltip from '../ChartTooltip.svelte';
  import { scaleBand, scaleLinear, stack, stackOffsetNone, max, sum, niceCeil } from '../scales.js';
  import { seriesColor } from '../palette.js';
  import { createValuesTween } from '../valuesTween.js';
  import { formatCompact } from '../format.js';

  // ─── Props ───────────────────────────────────────────────────────────────────

  interface Series {
    label: string;
    values: number[];
  }

  interface Props {
    categories: string[];
    series: Series[];
    mode?: 'grouped' | 'stacked';
    horizontal?: boolean;
    height?: number;
    valueFormat?: (n: number) => string;
    title: string;
    description?: string;
    legend?: boolean;
    showValues?: boolean;
  }

  let {
    categories,
    series,
    mode = 'grouped',
    horizontal = false,
    height = 280,
    valueFormat = formatCompact,
    title,
    description,
    legend = true,
    showValues = false,
  }: Props = $props();

  // ─── Legend toggle state ──────────────────────────────────────────────────

  let hiddenSeries = $state<string[]>([]);

  const visibleSeries = $derived(
    series.filter((s) => !hiddenSeries.includes(s.label))
  );

  const legendItems = $derived(
    series.map((s, i) => ({ label: s.label, color: seriesColor(i) }))
  );

  // ─── Tooltip state ────────────────────────────────────────────────────────

  interface TooltipState {
    visible: boolean;
    x: number;
    y: number;
    category: string;
    items: { label: string; value: number; color: string }[];
    total?: number;
  }

  let tip = $state<TooltipState>({
    visible: false,
    x: 0,
    y: 0,
    category: '',
    items: [],
  });

  // ─── Hover state ─────────────────────────────────────────────────────────

  /** The bar key being hovered: "seriesIndex-categoryIndex" or "stack-seriesIndex-categoryIndex" */
  let hoveredKey = $state<string | null>(null);

  function barOpacity(key: string): number {
    if (hoveredKey === null) return 1;
    return hoveredKey === key ? 1 : 0.45;
  }

  // ─── d3 layout ────────────────────────────────────────────────────────────

  /**
   * All values flattened in visible series × categories order — fed into the
   * valuesTween so transitions stay synced across mode changes.
   */
  const flatValues = $derived(
    visibleSeries.flatMap((s) => s.values.slice(0, categories.length))
  );

  // Tweened array; length tracks visibleSeries × categories.
  let tweenedFlat = $state<number[]>([]);

  // Build the tween engine with a stable empty start; the first $effect below
  // immediately calls tw.to() with the real values (jump because lengths differ).
  const tw = createValuesTween([], (vals) => {
    tweenedFlat = vals;
  });

  $effect(() => {
    tw.to([...flatValues], { regime: 'stabilize' });
  });

  /**
   * Reconstruct the tweened matrix: tweenedFlat → rows (series) × cols (categories).
   * Used in grouped layout.
   */
  const tweenedMatrix = $derived(
    visibleSeries.map((_, si) =>
      categories.map((_, ci) => tweenedFlat[si * categories.length + ci] ?? 0)
    )
  );

  // ─── Stacked layout ───────────────────────────────────────────────────────

  /** d3 stack layers over tweened values */
  const stackedData = $derived(() => {
    if (mode !== 'stacked' || visibleSeries.length === 0) return [];
    const keys = visibleSeries.map((s, i) => `s${i}`);
    const data = categories.map((_, ci) => {
      const row: Record<string, number> = {};
      visibleSeries.forEach((_, si) => {
        row[`s${si}`] = tweenedMatrix[si]?.[ci] ?? 0;
      });
      return row;
    });
    const stackGen = stack<Record<string, number>>()
      .keys(keys)
      .offset(stackOffsetNone);
    return stackGen(data);
  });

  // ─── Scale computation (receives ChartContext from ChartFrame) ────────────

  // These are derived per-render inside the template snippet — we pass
  // them down via a reactive helper.

  function computeScales(innerWidth: number, innerHeight: number) {
    const visibleMax =
      mode === 'stacked'
        ? niceCeil(
            Math.max(
              0,
              ...categories.map((_, ci) =>
                (visibleSeries.reduce((acc, _, si) => acc + (tweenedMatrix[si]?.[ci] ?? 0), 0))
              )
            )
          )
        : niceCeil(
            Math.max(
              0,
              ...visibleSeries.flatMap((_, si) =>
                categories.map((_, ci) => tweenedMatrix[si]?.[ci] ?? 0)
              )
            )
          );

    const safeMax = visibleMax === 0 ? 1 : visibleMax;

    if (!horizontal) {
      const xCat = scaleBand<string>()
        .domain(categories)
        .range([0, innerWidth])
        .paddingInner(0.25);

      const xInner = scaleBand<string>()
        .domain(visibleSeries.map((s) => s.label))
        .range([0, xCat.bandwidth()])
        .paddingInner(0.1);

      const yLin = scaleLinear()
        .domain([0, safeMax])
        .range([innerHeight, 0]);

      return { xCat, xInner, yLin, horizontal: false as const };
    } else {
      const yCat = scaleBand<string>()
        .domain(categories)
        .range([0, innerHeight])
        .paddingInner(0.25);

      const yInner = scaleBand<string>()
        .domain(visibleSeries.map((s) => s.label))
        .range([0, yCat.bandwidth()])
        .paddingInner(0.1);

      const xLin = scaleLinear()
        .domain([0, safeMax])
        .range([0, innerWidth]);

      return { yCat, yInner, xLin, horizontal: true as const };
    }
  }

  // ─── Tooltip helpers ──────────────────────────────────────────────────────

  function showTip(
    event: MouseEvent,
    categoryIndex: number,
    seriesIndex: number,
    marginLeft: number,
    marginTop: number,
    svgRect: DOMRect,
  ) {
    const frameX = event.clientX - svgRect.left;
    const frameY = event.clientY - svgRect.top;
    const cat = categories[categoryIndex];
    const items = visibleSeries.map((s, si) => ({
      label: s.label,
      value: tweenedMatrix[si]?.[categoryIndex] ?? 0,
      color: seriesColor(series.findIndex((orig) => orig.label === s.label)),
    }));
    const total =
      mode === 'stacked'
        ? items.reduce((s, it) => s + it.value, 0)
        : undefined;
    tip = {
      visible: true,
      x: frameX,
      y: frameY,
      category: cat,
      items: mode === 'grouped' ? [items[seriesIndex]] : items,
      total,
    };
  }

  function hideTip() {
    tip = { ...tip, visible: false };
    hoveredKey = null;
  }
</script>

<!-- ─── Legend ────────────────────────────────────────────────────────────── -->
{#if legend && series.length > 0}
  <Legend items={legendItems} toggleable bind:hidden={hiddenSeries} class="af-bar-legend" />
{/if}

<!-- ─── Chart ─────────────────────────────────────────────────────────────── -->
<ChartFrame
  {height}
  {title}
  {description}
  margin={{ left: horizontal ? 96 : 48, bottom: horizontal ? 36 : 40, right: 16, top: 12 }}
>
  {#snippet children(ctx)}
    {@const sc = computeScales(ctx.innerWidth, ctx.innerHeight)}

    <!-- Axes -->
    {#if !sc.horizontal}
      <Axis scale={sc.xCat} orient="bottom" innerWidth={ctx.innerWidth} innerHeight={ctx.innerHeight} />
      <Axis scale={sc.yLin} orient="left"   innerWidth={ctx.innerWidth} innerHeight={ctx.innerHeight} grid ticks={5} format={valueFormat} />
    {:else}
      <Axis scale={sc.yCat} orient="bottom" innerWidth={ctx.innerWidth} innerHeight={ctx.innerHeight} />
      <Axis scale={sc.xLin} orient="left"   innerWidth={ctx.innerWidth} innerHeight={ctx.innerHeight} grid ticks={5} format={valueFormat} />
    {/if}

    <!-- Bars: grouped vertical -->
    {#if !sc.horizontal && mode === 'grouped'}
      {#each categories as cat, ci}
        {@const catX = sc.xCat(cat) ?? 0}
        <g transform="translate({catX}, 0)" class="cat-group">
          {#each visibleSeries as s, si}
            {@const origIdx = series.findIndex((o) => o.label === s.label)}
            {@const barX = sc.xInner(s.label) ?? 0}
            {@const val = tweenedMatrix[si]?.[ci] ?? 0}
            {@const barH = Math.max(0, ctx.innerHeight - (sc.yLin(val) ?? ctx.innerHeight))}
            {@const barY = sc.yLin(val) ?? ctx.innerHeight}
            {@const key = `${si}-${ci}`}
            <rect
              x={barX}
              y={barY}
              width={sc.xInner.bandwidth()}
              height={barH}
              rx="3"
              fill={seriesColor(origIdx)}
              opacity={barOpacity(key)}
              class="bar"
              role="img"
              aria-label="{s.label} {cat}: {valueFormat(val)}"
              onmouseenter={(e) => {
                hoveredKey = key;
                showTip(e, ci, si, ctx.margin.left, ctx.margin.top,
                  (e.currentTarget as SVGRectElement).ownerSVGElement!.getBoundingClientRect());
              }}
              onmouseleave={hideTip}
            />
            {#if showValues && barH > 14}
              <text
                x={barX + sc.xInner.bandwidth() / 2}
                y={barY - 3}
                text-anchor="middle"
                class="bar-label"
                opacity={barOpacity(key)}
              >{valueFormat(val)}</text>
            {/if}
          {/each}
        </g>
      {/each}
    {/if}

    <!-- Bars: stacked vertical -->
    {#if !sc.horizontal && mode === 'stacked'}
      {#each stackedData() as layer, si}
        {@const origIdx = series.findIndex((o) => o.label === (visibleSeries[si]?.label ?? ''))}
        {#each categories as _cat, ci}
          {@const seg = layer[ci]}
          {@const y0 = sc.yLin(seg?.[1] ?? 0) ?? 0}
          {@const y1 = sc.yLin(seg?.[0] ?? 0) ?? 0}
          {@const barH = Math.max(0, y1 - y0)}
          {@const catX = sc.xCat(_cat) ?? 0}
          {@const key = `stack-${si}-${ci}`}
          <rect
            x={catX}
            y={y0}
            width={sc.xCat.bandwidth()}
            height={barH}
            rx="3"
            fill={seriesColor(origIdx)}
            opacity={barOpacity(key)}
            class="bar"
            role="img"
            aria-label="{visibleSeries[si]?.label} {_cat}: {valueFormat((seg?.[1] ?? 0) - (seg?.[0] ?? 0))}"
            onmouseenter={(e) => {
              hoveredKey = key;
              showTip(e, ci, si, ctx.margin.left, ctx.margin.top,
                (e.currentTarget as SVGRectElement).ownerSVGElement!.getBoundingClientRect());
            }}
            onmouseleave={hideTip}
          />
          {#if showValues && barH > 14}
            {@const segVal = (seg?.[1] ?? 0) - (seg?.[0] ?? 0)}
            <text
              x={catX + sc.xCat.bandwidth() / 2}
              y={y0 + barH / 2 + 4}
              text-anchor="middle"
              class="bar-label"
              opacity={barOpacity(key)}
            >{valueFormat(segVal)}</text>
          {/if}
        {/each}
      {/each}
    {/if}

    <!-- Bars: grouped horizontal -->
    {#if sc.horizontal && mode === 'grouped'}
      {#each categories as cat, ci}
        {@const catY = sc.yCat(cat) ?? 0}
        <g transform="translate(0, {catY})">
          {#each visibleSeries as s, si}
            {@const origIdx = series.findIndex((o) => o.label === s.label)}
            {@const barY = sc.yInner(s.label) ?? 0}
            {@const val = tweenedMatrix[si]?.[ci] ?? 0}
            {@const barW = Math.max(0, sc.xLin(val) ?? 0)}
            {@const key = `h-${si}-${ci}`}
            <rect
              x={0}
              y={barY}
              width={barW}
              height={sc.yInner.bandwidth()}
              rx="3"
              fill={seriesColor(origIdx)}
              opacity={barOpacity(key)}
              class="bar"
              role="img"
              aria-label="{s.label} {cat}: {valueFormat(val)}"
              onmouseenter={(e) => {
                hoveredKey = key;
                showTip(e, ci, si, ctx.margin.left, ctx.margin.top,
                  (e.currentTarget as SVGRectElement).ownerSVGElement!.getBoundingClientRect());
              }}
              onmouseleave={hideTip}
            />
            {#if showValues && barW > 28}
              <text
                x={barW + 4}
                y={barY + sc.yInner.bandwidth() / 2 + 4}
                class="bar-label"
              >{valueFormat(val)}</text>
            {/if}
          {/each}
        </g>
      {/each}
    {/if}

    <!-- Bars: stacked horizontal -->
    {#if sc.horizontal && mode === 'stacked'}
      {#each stackedData() as layer, si}
        {@const origIdx = series.findIndex((o) => o.label === (visibleSeries[si]?.label ?? ''))}
        {#each categories as _cat, ci}
          {@const seg = layer[ci]}
          {@const x0 = sc.xLin(seg?.[0] ?? 0) ?? 0}
          {@const x1 = sc.xLin(seg?.[1] ?? 0) ?? 0}
          {@const barW = Math.max(0, x1 - x0)}
          {@const catY = sc.yCat(_cat) ?? 0}
          {@const key = `hs-${si}-${ci}`}
          <rect
            x={x0}
            y={catY}
            width={barW}
            height={sc.yCat.bandwidth()}
            rx="3"
            fill={seriesColor(origIdx)}
            opacity={barOpacity(key)}
            class="bar"
            role="img"
            aria-label="{visibleSeries[si]?.label} {_cat}: {valueFormat((seg?.[1] ?? 0) - (seg?.[0] ?? 0))}"
            onmouseenter={(e) => {
              hoveredKey = key;
              showTip(e, ci, si, ctx.margin.left, ctx.margin.top,
                (e.currentTarget as SVGRectElement).ownerSVGElement!.getBoundingClientRect());
            }}
            onmouseleave={hideTip}
          />
        {/each}
      {/each}
    {/if}
  {/snippet}

  {#snippet overlay(ctx)}
    <ChartTooltip x={tip.x} y={tip.y} visible={tip.visible} frameWidth={ctx.width}>
      <div class="tip-cat">{tip.category}</div>
      {#each tip.items as item}
        <div class="tip-row">
          <span class="tip-swatch" style:--tc={item.color}></span>
          <span class="tip-label">{item.label}</span>
          <span class="tip-val">{valueFormat(item.value)}</span>
        </div>
      {/each}
      {#if tip.total !== undefined}
        <div class="tip-total">Total: {valueFormat(tip.total)}</div>
      {/if}
    </ChartTooltip>
  {/snippet}
</ChartFrame>

<style>
  /* Entrance: bars rise from baseline (guarded by prefers-reduced-motion) */
  @media (prefers-reduced-motion: no-preference) {
    .bar {
      animation: bar-rise var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
    }

    /* Stagger up to 12 bars */
    .cat-group .bar:nth-child(1) { animation-delay: calc(0  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(2) { animation-delay: calc(1  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(3) { animation-delay: calc(2  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(4) { animation-delay: calc(3  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(5) { animation-delay: calc(4  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(6) { animation-delay: calc(5  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(7) { animation-delay: calc(6  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(8) { animation-delay: calc(7  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(9) { animation-delay: calc(8  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(10){ animation-delay: calc(9  * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(11){ animation-delay: calc(10 * var(--af-motion-stagger)); }
    .cat-group .bar:nth-child(12){ animation-delay: calc(11 * var(--af-motion-stagger)); }
  }

  @keyframes bar-rise {
    from {
      transform: scaleY(0);
      transform-origin: bottom center;
      opacity: 0;
    }
    to {
      transform: scaleY(1);
      transform-origin: bottom center;
      opacity: 1;
    }
  }

  .bar {
    transition: opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    cursor: default;
  }

  .bar-label {
    fill: var(--af-text-secondary);
    font-size: var(--af-text-xs);
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
    pointer-events: none;
  }

  /* Tooltip internals */
  .tip-cat {
    font-weight: var(--af-weight-semibold);
    margin-bottom: 4px;
    font-size: var(--af-text-xs);
  }

  .tip-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--af-text-xs);
    white-space: nowrap;
    min-width: 0;
  }

  .tip-swatch {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    background: var(--tc);
    flex-shrink: 0;
  }

  .tip-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tip-val {
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
    font-weight: var(--af-weight-semibold);
    margin-left: var(--af-space-2);
  }

  .tip-total {
    margin-top: 4px;
    border-top: 1px solid color-mix(in srgb, currentColor 20%, transparent);
    padding-top: 4px;
    font-size: var(--af-text-xs);
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
    font-weight: var(--af-weight-semibold);
  }

  :global(.af-bar-legend) {
    margin-bottom: var(--af-space-3);
  }
</style>
