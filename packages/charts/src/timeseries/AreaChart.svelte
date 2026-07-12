<script lang="ts">
  /**
   * AreaChart — overlapping or stacked area chart.
   *
   * stacked=false: overlapping translucent fills (fill-opacity 0.12) with
   * a 2px stroke line on top for each series.
   *
   * stacked=true: d3 stack over ALIGNED x samples (all series must share
   * the same x values in the same order — this is a documented constraint).
   * Stacked areas use fill-opacity 0.25 with 1.5px strokes at full color
   * on each band's top edge. Calm and legible: bands read as composition.
   *
   * Same hover/tooltip/legend/transition behavior as LineChart.
   *
   * Constitution: no raw hex, no raw ms, --af-* tokens only,
   * prefers-reduced-motion first-class, tabular-nums on all numerals.
   */

  import ChartFrame from '../ChartFrame.svelte';
  import Axis from '../Axis.svelte';
  import Legend from '../Legend.svelte';
  import ChartTooltip from '../ChartTooltip.svelte';
  import {
    line as d3line,
    area as d3area,
    stack as d3stack,
    stackOffsetNone,
    curveMonotoneX,
    curveLinear,
    max,
    scaleLinear,
  } from '../scales.js';
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
    type AnyXScale,
  } from './internal.js';

  export type { Series as AreaSeries, TimePoint as AreaPoint };

  interface Props {
    series: Series[];
    height?: number;
    yFormat?: (n: number) => string;
    xFormat?: (x: number | Date) => string;
    curve?: 'monotone' | 'linear';
    title: string;
    description?: string;
    legend?: boolean;
    /**
     * When stacked=true, all series MUST share the same x values in the same
     * positional order. Misaligned x sets produce incorrect stacking.
     */
    stacked?: boolean;
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
    stacked = false,
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

  // Start with empty tween state; $effect fires on mount with real data.
  // Avoids accessing reactive props during $state initialisation.
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

  // All three reactive sources (series, hidden, currentYValues) referenced at
  // top level so Svelte tracks them correctly.
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

  // ── Stacked path builder ──────────────────────────────────────────────────
  function buildStackedPaths(
    ss: typeof animatedSeries,
    xScale: AnyXScale,
    innerHeight: number,
    curveType: typeof curveMonotoneX,
  ): { areaPath: string; linePath: string; color: string }[] {
    if (ss.length === 0) return [];

    const n = ss[0].points.length;
    const xValues = ss[0].points.map((p) => p.x);
    const keys = ss.map((s) => s.label);

    type Row = Record<string, number>;
    const tableData: Row[] = Array.from({ length: n }, (_, i) => {
      const row: Row = {};
      for (const s of ss) {
        row[s.label] = s.points[i]?.y ?? 0;
      }
      return row;
    });

    const totals = tableData.map((row) => keys.reduce((acc, k) => acc + (row[k] ?? 0), 0));
    const yMax = max(totals) ?? 1;
    const yScale = scaleLinear().domain([0, yMax]).range([innerHeight, 0]);

    const stackGen = d3stack<Row>().keys(keys).offset(stackOffsetNone);
    const layers = stackGen(tableData);

    return layers.map((layer, li) => {
      type Datum = [number, number];
      const areaGen = d3area<Datum>()
        .x((_, i) => (xScale(xValues[i] as never) ?? 0))
        .y0((d) => yScale(d[0]))
        .y1((d) => yScale(d[1]))
        .curve(curveType);

      const lineGen = d3line<Datum>()
        .x((_, i) => (xScale(xValues[i] as never) ?? 0))
        .y((d) => yScale(d[1]))
        .curve(curveType);

      const pts = layer.map((d) => [d[0], d[1]] as Datum);

      return {
        areaPath: areaGen(pts) ?? '',
        linePath: lineGen(pts) ?? '',
        color: seriesColor(series.findIndex((orig) => orig.label === keys[li])),
      };
    });
  }

  // Stacked y-max for axis — references animatedSeries at top level.
  const stackedYMax = $derived(computeStackedMax(stacked, animatedSeries));

  function computeStackedMax(
    isStacked: boolean,
    ss: typeof animatedSeries,
  ): number {
    if (!isStacked || ss.length === 0) return 0;
    const n = ss[0].points.length;
    let stackMax = 0;
    for (let i = 0; i < n; i++) {
      const sum = ss.reduce((acc, s) => acc + (s.points[i]?.y ?? 0), 0);
      if (sum > stackMax) stackMax = sum;
    }
    return stackMax;
  }
</script>

<div class="af-areachart-wrap">
  {#if legend && legendItems.length > 0}
    <div class="af-areachart-legend">
      <Legend items={legendItems} toggleable bind:hidden />
    </div>
  {/if}

  <ChartFrame {height} {title} {description}>
    {#snippet children(ctx)}
      {@const { innerWidth, innerHeight, margin } = ctx}
      {@const xScale = buildXScale(series, innerWidth, hidden)}
      {@const curveType = curve === 'linear' ? curveLinear : curveMonotoneX}

      {#if animatedSeries.length === 0}
        <text
          x={innerWidth / 2}
          y={innerHeight / 2}
          text-anchor="middle"
          dominant-baseline="middle"
          class="empty-label"
        >No data</text>
      {:else if stacked}
        <!-- Stacked areas -->
        {@const yMax = stackedYMax > 0 ? stackedYMax : 1}
        {@const yScale = scaleLinear().domain([0, yMax]).range([innerHeight, 0])}

        <Axis scale={yScale} orient="left" {innerWidth} {innerHeight} grid format={yFormat} />
        <Axis scale={xScale} orient="bottom" {innerWidth} {innerHeight} format={(v) => xFormat(v as number | Date)} />

        <defs>
          <style>
            @keyframes af-ac-draw-{uid} {'{'}
              from {'{'} stroke-dashoffset: {APPROX_PATH_LEN}; {'}'}
              to   {'{'} stroke-dashoffset: 0; {'}'}
            {'}'}
            {#each animatedSeries as _, i}
              @media (prefers-reduced-motion: no-preference) {'{'}
                .af-ac-line-{uid}-{i} {'{'}
                  stroke-dasharray: {APPROX_PATH_LEN};
                  stroke-dashoffset: {APPROX_PATH_LEN};
                  animation:
                    af-ac-draw-{uid}
                    var(--af-motion-explore-duration)
                    var(--af-motion-explore-ease)
                    forwards;
                  animation-delay: calc({i} * var(--af-motion-stagger));
                {'}'}
              {'}'}
            {/each}
          </style>
        </defs>

        {#each buildStackedPaths(animatedSeries, xScale, innerHeight, curveType) as sp, i}
          <path d={sp.areaPath} fill={sp.color} fill-opacity="0.25" stroke="none" />
          <path
            class="af-ac-line-{uid}-{i}"
            d={sp.linePath}
            fill="none"
            stroke={sp.color}
            stroke-width="1.5"
            stroke-linecap="round"
          />
        {/each}

        <!-- Crosshair for stacked -->
        {#if tooltipVisible && hoverIndex >= 0}
          {@const firstVisible = series.filter((sv) => !hidden.includes(sv.label))[0]}
          {#if firstVisible?.points[hoverIndex]}
            {@const hx = xScale(firstVisible.points[hoverIndex].x as never) ?? 0}
            <line x1={hx} x2={hx} y1="0" y2={innerHeight} stroke="var(--af-border-strong)" stroke-width="1" />
          {/if}
        {/if}

        <!-- Capture rect -->
        <rect
          x="0" y="0" width={innerWidth} height={innerHeight}
          fill="transparent" role="presentation" aria-hidden="true"
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
              tooltipY = margin.top + innerHeight / 2;
            }
          }}
          onpointerleave={() => { tooltipVisible = false; hoverIndex = -1; }}
        />

      {:else}
        <!-- Overlapping areas -->
        {@const yScale = buildYScale(series, innerHeight, hidden)}

        <Axis scale={yScale} orient="left" {innerWidth} {innerHeight} grid format={yFormat} />
        <Axis scale={xScale} orient="bottom" {innerWidth} {innerHeight} format={(v) => xFormat(v as number | Date)} />

        <defs>
          <style>
            @keyframes af-ac-draw-{uid} {'{'}
              from {'{'} stroke-dashoffset: {APPROX_PATH_LEN}; {'}'}
              to   {'{'} stroke-dashoffset: 0; {'}'}
            {'}'}
            {#each animatedSeries as _, i}
              @media (prefers-reduced-motion: no-preference) {'{'}
                .af-ac-line-{uid}-{i} {'{'}
                  stroke-dasharray: {APPROX_PATH_LEN};
                  stroke-dashoffset: {APPROX_PATH_LEN};
                  animation:
                    af-ac-draw-{uid}
                    var(--af-motion-explore-duration)
                    var(--af-motion-explore-ease)
                    forwards;
                  animation-delay: calc({i} * var(--af-motion-stagger));
                {'}'}
              {'}'}
            {/each}
          </style>
        </defs>

        {#each animatedSeries as s, i}
          {@const seriesIdx = series.findIndex((orig) => orig.label === s.label)}
          {@const areaGen = d3area<{ x: number | Date; y: number }>()
            .x((p) => (xScale(p.x as never) ?? 0))
            .y0(innerHeight)
            .y1((p) => yScale(p.y))
            .curve(curveType)}
          {@const lineGen = d3line<{ x: number | Date; y: number }>()
            .x((p) => (xScale(p.x as never) ?? 0))
            .y((p) => yScale(p.y))
            .curve(curveType)}
          <path d={areaGen(s.points) ?? ''} fill={seriesColor(seriesIdx)} fill-opacity="0.12" stroke="none" />
          <path
            class="af-ac-line-{uid}-{i}"
            d={lineGen(s.points) ?? ''}
            fill="none"
            stroke={seriesColor(seriesIdx)}
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        {/each}

        <!-- Crosshair + dots -->
        {#if tooltipVisible && hoverIndex >= 0}
          {@const firstVisible = series.filter((sv) => !hidden.includes(sv.label))[0]}
          {#if firstVisible?.points[hoverIndex]}
            {@const hx = xScale(firstVisible.points[hoverIndex].x as never) ?? 0}
            <line x1={hx} x2={hx} y1="0" y2={innerHeight} stroke="var(--af-border-strong)" stroke-width="1" />
            {#each animatedSeries as s}
              {@const yScale2 = buildYScale(series, innerHeight, hidden)}
              {@const pt = s.points[hoverIndex]}
              {#if pt}
                <circle
                  cx={xScale(pt.x as never) ?? 0}
                  cy={yScale2(pt.y)}
                  r="3.5"
                  fill={seriesColor(series.findIndex((orig) => orig.label === s.label))}
                  stroke="var(--af-surface)"
                  stroke-width="1.5"
                />
              {/if}
            {/each}
          {/if}
        {/if}

        <!-- Capture rect -->
        <rect
          x="0" y="0" width={innerWidth} height={innerHeight}
          fill="transparent" role="presentation" aria-hidden="true"
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
              const yScale3 = buildYScale(series, innerHeight, hidden);
              tooltipY = (firstVis ? yScale3(firstVis.points[idx]?.y ?? 0) : 0) + margin.top;
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
  .af-areachart-wrap {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    width: 100%;
  }

  .af-areachart-legend {
    padding-inline: var(--af-space-1);
  }

  .empty-label {
    fill: var(--af-text-muted);
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
  }

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
