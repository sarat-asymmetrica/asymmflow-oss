<script lang="ts">
  /**
   * LineChart — the workhorse timeseries chart.
   *
   * Multi-series line chart with:
   * - scaleTime (Date x) or scaleLinear (number x) auto-detection
   * - Y axis left with grid; X axis bottom (solid gridlines, §3)
   * - Crosshair + per-series dots + ChartTooltip on hover
   * - Toggleable legend — hidden series excluded from rendering AND y-domain
   * - Geodesic value tween (createValuesTween, regime 'stabilize') when
   *   series shape stays the same; jump on shape change
   * - Draw-on entrance per path, staggered, reduced-motion guarded
   *
   * Constitution: no raw hex, no raw ms, --af-* tokens only,
   * prefers-reduced-motion first-class, tabular-nums on all numerals.
   */

  import ChartFrame from '../ChartFrame.svelte';
  import Axis from '../Axis.svelte';
  import Legend from '../Legend.svelte';
  import ChartTooltip from '../ChartTooltip.svelte';
  import { line as d3line, curveMonotoneX, curveLinear } from '../scales.js';
  import { seriesColor } from '../palette.js';
  import { createValuesTween } from '../valuesTween.js';
  import { formatCompact } from '../format.js';
  import {
    buildXScale,
    buildYScale,
    findNearestIndex,
    defaultXFormat,
    type Series,
    type TimePoint,
  } from './internal.js';

  export type { Series as LineSeries, TimePoint as LinePoint };

  interface Props {
    series: Series[];
    height?: number;
    yFormat?: (n: number) => string;
    xFormat?: (x: number | Date) => string;
    curve?: 'monotone' | 'linear';
    title: string;
    description?: string;
    legend?: boolean;
  }

  let {
    series,
    height = 280,
    yFormat = formatCompact,
    xFormat = defaultXFormat,
    curve = 'monotone',
    title,
    description,
    legend = true,
  }: Props = $props();

  // ── Legend state ──────────────────────────────────────────────────────────
  let hidden = $state<string[]>([]);
  const legendItems = $derived(series.map((s, i) => ({ label: s.label, color: seriesColor(i) })));

  // ── Hover state ───────────────────────────────────────────────────────────
  let tooltipVisible = $state(false);
  let hoverIndex = $state(-1);
  let tooltipX = $state(0);
  let tooltipY = $state(0);
  let frameWidth = $state(320);

  // ── Value tween ───────────────────────────────────────────────────────────
  function flatYValues(ss: Series[], hid: string[]): number[] {
    return ss
      .filter((s) => !hid.includes(s.label))
      .flatMap((s) => s.points.map((p) => p.y));
  }

  function shapeSig(ss: Series[], hid: string[]): string {
    return ss
      .filter((s) => !hid.includes(s.label))
      .map((s) => `${s.label}:${s.points.length}`)
      .join('|');
  }

  // Start with an empty tween state; the $effect below will fire on mount
  // and immediately call tween.to() with the real data. Starting empty avoids
  // accessing reactive props during $state initialisation (Svelte warning).
  let currentYValues = $state<number[]>([]);
  let lastShape      = $state('');

  const tween = createValuesTween([], (vals) => {
    currentYValues = vals;
  });

  $effect(() => {
    const sig = shapeSig(series, hidden);
    const flat = flatYValues(series, hidden);
    if (sig !== lastShape) {
      lastShape = sig;
      tween.jump(flat);
    } else {
      tween.to(flat, { regime: 'stabilize' });
    }
  });

  // Reconstruct animated series from tween values.
  // All three reactive sources (series, hidden, currentYValues) are referenced
  // at the top level so Svelte tracks them correctly.
  const visibleSeries = $derived(series.filter((s) => !hidden.includes(s.label)));

  const animatedSeries = $derived(
    buildAnimated(visibleSeries, currentYValues)
  );

  function buildAnimated(
    visible: Series[],
    yVals: number[],
  ): { label: string; points: { x: number | Date; y: number }[] }[] {
    let offset = 0;
    return visible.map((s) => {
      const pts = s.points.map((p, i) => ({ x: p.x, y: yVals[offset + i] ?? p.y }));
      offset += s.points.length;
      return { label: s.label, points: pts };
    });
  }

  let uid = $state(Math.random().toString(36).slice(2, 8));
  const APPROX_PATH_LEN = 2000;
</script>

<div class="af-linechart-wrap">
  {#if legend && legendItems.length > 0}
    <div class="af-linechart-legend">
      <Legend items={legendItems} toggleable bind:hidden />
    </div>
  {/if}

  <ChartFrame {height} {title} {description}>
    {#snippet children(ctx)}
      {@const { innerWidth, innerHeight, margin } = ctx}

      {#if frameWidth !== ctx.width}
        <!-- sync frameWidth -->
      {/if}

      {#if animatedSeries.length === 0}
        <text
          x={innerWidth / 2}
          y={innerHeight / 2}
          text-anchor="middle"
          dominant-baseline="middle"
          class="empty-label"
        >No data</text>
      {:else}
        {@const xScale = buildXScale(series, innerWidth, hidden)}
        {@const yScale = buildYScale(series, innerHeight, hidden)}
        {@const curveType = curve === 'linear' ? curveLinear : curveMonotoneX}

        <!-- Y axis with grid -->
        <Axis scale={yScale} orient="left" {innerWidth} {innerHeight} grid format={yFormat} />
        <!-- X axis -->
        <Axis
          scale={xScale}
          orient="bottom"
          {innerWidth}
          {innerHeight}
          format={(v) => xFormat(v as number | Date)}
        />

        <!-- Draw-on animation styles -->
        <defs>
          <style>
            @keyframes af-lc-draw-{uid} {'{'}
              from {'{'} stroke-dashoffset: {APPROX_PATH_LEN}; {'}'}
              to   {'{'} stroke-dashoffset: 0; {'}'}
            {'}'}
            {#each animatedSeries as _, i}
              @media (prefers-reduced-motion: no-preference) {'{'}
                .af-lc-path-{uid}-{i} {'{'}
                  stroke-dasharray: {APPROX_PATH_LEN};
                  stroke-dashoffset: {APPROX_PATH_LEN};
                  animation:
                    af-lc-draw-{uid}
                    var(--af-motion-explore-duration)
                    var(--af-motion-explore-ease)
                    forwards;
                  animation-delay: calc({i} * var(--af-motion-stagger));
                {'}'}
              {'}'}
            {/each}
          </style>
        </defs>

        <!-- Series paths -->
        {#each animatedSeries as s, i}
          {@const seriesIdx = series.findIndex((orig) => orig.label === s.label)}
          {@const lineGen = d3line<{ x: number | Date; y: number }>()
            .x((p) => (xScale(p.x as never) ?? 0))
            .y((p) => yScale(p.y))
            .curve(curveType)}
          <path
            class="af-lc-path-{uid}-{i}"
            d={lineGen(s.points) ?? ''}
            fill="none"
            stroke={seriesColor(seriesIdx)}
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        {/each}

        <!-- Crosshair + hover dots -->
        {#if tooltipVisible && hoverIndex >= 0}
          {@const firstVisible = series.filter((sv) => !hidden.includes(sv.label))[0]}
          {#if firstVisible?.points[hoverIndex]}
            {@const hx = xScale(firstVisible.points[hoverIndex].x as never) ?? 0}
            <line
              x1={hx} x2={hx}
              y1="0" y2={innerHeight}
              stroke="var(--af-border-strong)"
              stroke-width="1"
            />
            {#each animatedSeries as s}
              {@const pt = s.points[hoverIndex]}
              {#if pt}
                <circle
                  cx={xScale(pt.x as never) ?? 0}
                  cy={yScale(pt.y)}
                  r="3.5"
                  fill={seriesColor(series.findIndex((orig) => orig.label === s.label))}
                  stroke="var(--af-surface)"
                  stroke-width="1.5"
                />
              {/if}
            {/each}
          {/if}
        {/if}

        <!-- Transparent capture rect — full plot area -->
        <rect
          x="0" y="0"
          width={innerWidth} height={innerHeight}
          fill="transparent"
          role="presentation"
          aria-hidden="true"
          style="cursor: crosshair;"
          onpointermove={(e) => {
            frameWidth = ctx.width;
            const svgEl = (e.currentTarget as SVGRectElement).closest('svg');
            const svgRect = svgEl?.getBoundingClientRect();
            const plotX = svgRect ? e.clientX - svgRect.left - margin.left : 0;
            const xScale2 = buildXScale(series, innerWidth, hidden);
            const idx = findNearestIndex(plotX, series, hidden, xScale2);
            hoverIndex = idx;
            tooltipVisible = idx >= 0;
            if (idx >= 0) {
              const firstVis = series.filter((sv) => !hidden.includes(sv.label))[0];
              const xPt = firstVis?.points[idx]?.x;
              tooltipX = (xPt != null ? (xScale2(xPt as never) ?? 0) : 0) + margin.left;
              const yScale2 = buildYScale(series, innerHeight, hidden);
              tooltipY = (firstVis ? yScale2(firstVis.points[idx]?.y ?? 0) : 0) + margin.top;
            }
          }}
          onpointerleave={() => { tooltipVisible = false; hoverIndex = -1; }}
        />
      {/if}
    {/snippet}

    {#snippet overlay(ctx)}
      {#if tooltipVisible && hoverIndex >= 0}
        {@const visibleSeries = series.filter((s) => !hidden.includes(s.label))}
        {#if visibleSeries.length > 0}
          {@const xVal = visibleSeries[0].points[hoverIndex]?.x}
          <ChartTooltip
            x={tooltipX}
            y={tooltipY}
            visible={tooltipVisible}
            frameWidth={ctx.width}
          >
            {#snippet children()}
              <div class="tip-header">{xVal != null ? xFormat(xVal) : ''}</div>
              {#each visibleSeries as s}
                {@const pt = s.points[hoverIndex]}
                {#if pt}
                  <div class="tip-row">
                    <span class="tip-swatch" style:background={seriesColor(series.findIndex((orig) => orig.label === s.label))}></span>
                    <span class="tip-label">{s.label}</span>
                    <span class="tip-val">{yFormat(pt.y)}</span>
                  </div>
                {/if}
              {/each}
            {/snippet}
          </ChartTooltip>
        {/if}
      {/if}
    {/snippet}
  </ChartFrame>
</div>

<style>
  .af-linechart-wrap {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    width: 100%;
  }

  .af-linechart-legend {
    padding-inline: var(--af-space-1);
  }

  .empty-label {
    fill: var(--af-text-muted);
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
  }

  /* Tooltip internals — rendered in the overlay HTML layer */
  :global(.tip-header) {
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
    font-weight: var(--af-weight-semibold);
    font-size: var(--af-text-xs);
    margin-bottom: var(--af-space-1);
    white-space: nowrap;
    color: var(--af-text-inverse);
  }

  :global(.tip-row) {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    margin-top: 2px;
  }

  :global(.tip-swatch) {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  :global(.tip-label) {
    flex: 1;
    white-space: nowrap;
    font-size: var(--af-text-xs);
    color: var(--af-text-inverse);
  }

  :global(.tip-val) {
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
    font-weight: var(--af-weight-semibold);
    font-size: var(--af-text-xs);
    margin-left: var(--af-space-2);
    color: var(--af-text-inverse);
  }
</style>
