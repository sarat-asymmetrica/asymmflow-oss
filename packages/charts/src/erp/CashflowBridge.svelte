<script lang="ts">
  /**
   * CashflowBridge — the waterfall chart for ERP cashflow analysis.
   *
   * Classic waterfall: positive bars rise (--af-success), negative bars fall
   * (--af-danger), total bars span from zero (--af-inverse-surface).
   * Thin connector lines link each bar's end level to the next start.
   * Hover: bar emphasis with dimming. Tooltip: label + value + running total.
   * Entrance: bars grow from their start level, staggered. Data change: geodesic tween.
   *
   * Constitution §4c: status colors are the encoding here — permitted.
   */

  import { untrack } from 'svelte';
  import { scaleBand, scaleLinear, max, min, niceCeil } from '../scales.js';
  import { formatCurrency, formatCompact } from '../format.js';
  import { createValuesTween } from '../valuesTween.js';
  import ChartFrame from '../ChartFrame.svelte';
  import ChartTooltip from '../ChartTooltip.svelte';
  import Axis from '../Axis.svelte';
  import type { ChartContext } from '../ChartFrame.svelte';

  // ─── Props ────────────────────────────────────────────────────────────────

  interface BridgeItem {
    label: string;
    value: number;
    isTotal?: boolean;
  }

  interface Props {
    items: BridgeItem[];
    height?: number;
    valueFormat?: (n: number) => string;
    currency?: string;
    title: string;
    description?: string;
    showConnectors?: boolean;
    showValues?: boolean;
  }

  let {
    items,
    height = 300,
    valueFormat,
    currency = 'BHD',
    title,
    description,
    showConnectors = true,
    showValues = true,
  }: Props = $props();

  // ─── Default format ────────────────────────────────────────────────────────

  function defaultFmt(n: number): string {
    return formatCurrency(n, currency, { compact: true });
  }

  const fmt = $derived(valueFormat ?? defaultFmt);

  // ─── Cumulative levels ─────────────────────────────────────────────────────

  interface BarData {
    label: string;
    value: number;
    isTotal: boolean;
    /** Y-level where the bar starts (lower value). */
    barStart: number;
    /** Y-level where the bar ends (upper value). */
    barEnd: number;
    /** Running cumulative after this bar. */
    cumulative: number;
  }

  const bars = $derived.by((): BarData[] => {
    let running = 0;
    return items.map((item) => {
      if (item.isTotal) {
        const totalVal = item.value;
        running = totalVal;
        return {
          label: item.label,
          value: item.value,
          isTotal: true,
          barStart: 0,
          barEnd: totalVal,
          cumulative: totalVal,
        };
      }
      const barStart = Math.min(running, running + item.value);
      const barEnd = Math.max(running, running + item.value);
      running += item.value;
      return {
        label: item.label,
        value: item.value,
        isTotal: false,
        barStart,
        barEnd,
        cumulative: running,
      };
    });
  });

  // ─── Y domain ─────────────────────────────────────────────────────────────

  const yMin = $derived.by(() => {
    const vals = bars.flatMap((b) => [b.barStart, b.barEnd]);
    const mn = min(vals) ?? 0;
    return mn < 0 ? mn - Math.abs(mn) * 0.05 : 0;
  });

  const yMax = $derived.by(() => {
    const vals = bars.flatMap((b) => [b.barStart, b.barEnd]);
    const mx = max(vals) ?? 0;
    return niceCeil(mx * 1.05) || 1;
  });

  // ─── Tween: track cumulative levels so bars animate together ──────────────

  const getCumulatives = (bs: BarData[]) => bs.map((b) => b.cumulative);

  // Compute initial cumulatives — untrack items read (intentional initial snapshot;
  // all subsequent updates flow through the $effect below).
  const _initCums = untrack(() => {
    const cums: number[] = [];
    let r = 0;
    for (const item of items) {
      if (item.isTotal) { r = item.value; }
      else { r += item.value; }
      cums.push(r);
    }
    return cums;
  });
  let tweenedCumulatives = $state(_initCums);

  const tween = createValuesTween(_initCums, (vals) => {
    tweenedCumulatives = vals;
  });

  $effect(() => {
    tween.to(getCumulatives(bars), { regime: 'stabilize' });
  });

  // Reconstruct animated bars from tweened cumulatives
  const animatedBars = $derived.by((): BarData[] => {
    let running = 0;
    return items.map((item, i) => {
      const animCum = tweenedCumulatives[i] ?? bars[i]?.cumulative ?? 0;
      if (item.isTotal) {
        running = animCum;
        return {
          label: item.label,
          value: item.value,
          isTotal: true,
          barStart: 0,
          barEnd: animCum,
          cumulative: animCum,
        };
      }
      const prev = running;
      running = animCum;
      return {
        label: item.label,
        value: item.value,
        isTotal: false,
        barStart: Math.min(prev, animCum),
        barEnd: Math.max(prev, animCum),
        cumulative: animCum,
      };
    });
  });

  // ─── Hover state ───────────────────────────────────────────────────────────

  let hoveredIndex = $state<number | null>(null);
  let tipX = $state(0);
  let tipY = $state(0);

  // ─── Entrance animation state ──────────────────────────────────────────────
  // Each bar tracks whether its entrance has fired (no re-fire on data update).
  let entered = $state(false);

  $effect(() => {
    if (!entered) {
      const id = setTimeout(() => { entered = true; }, 50);
      return () => clearTimeout(id);
    }
  });
</script>

<ChartFrame
  {title}
  {description}
  {height}
  margin={{ left: 64, right: 16, top: 16, bottom: 40 }}
  class="af-cashflow-bridge"
>
  {#snippet children(ctx: ChartContext)}
    {@const xScale = scaleBand()
      .domain(items.map((_, i) => String(i)))
      .range([0, ctx.innerWidth])
      .paddingInner(0.35)}
    {@const yScale = scaleLinear()
      .domain([yMin, yMax])
      .range([ctx.innerHeight, 0])}
    {@const bw = xScale.bandwidth()}

    {#if animatedBars.length === 0}
      <text
        x={ctx.innerWidth / 2}
        y={ctx.innerHeight / 2}
        text-anchor="middle"
        dominant-baseline="middle"
        class="empty-label"
      >No data</text>
    {/if}

    <!-- Axes -->
    <Axis
      scale={yScale}
      orient="left"
      innerWidth={ctx.innerWidth}
      innerHeight={ctx.innerHeight}
      grid
      format={(v) => formatCompact(v as number)}
      ticks={5}
    />
    <Axis
      scale={xScale}
      orient="bottom"
      innerWidth={ctx.innerWidth}
      innerHeight={ctx.innerHeight}
      format={(v) => items[Number(v)]?.label ?? ''}
    />

    <!-- Zero line when negatives exist -->
    {#if yMin < 0}
      <line
        class="zero-line"
        x1="0"
        x2={ctx.innerWidth}
        y1={yScale(0)}
        y2={yScale(0)}
      />
    {/if}

    <!-- Connector lines -->
    {#if showConnectors}
      {#each animatedBars as bar, i}
        {#if i < animatedBars.length - 1}
          {@const nextBar = animatedBars[i + 1]}
          {@const x1 = (xScale(String(i)) ?? 0) + bw}
          {@const x2 = xScale(String(i + 1)) ?? 0}
          {@const yLevel = bar.isTotal ? yScale(bar.barEnd) : yScale(bar.cumulative)}
          {@const yNext = nextBar.isTotal ? yScale(0) : yScale(bar.cumulative)}
          <line
            class="connector"
            {x1}
            {x2}
            y1={yLevel}
            y2={yNext}
          />
        {/if}
      {/each}
    {/if}

    <!-- Bars -->
    {#each animatedBars as bar, i}
      {@const x = xScale(String(i)) ?? 0}
      {@const barTopY = yScale(bar.barEnd)}
      {@const barH = Math.abs(yScale(bar.barStart) - yScale(bar.barEnd))}
      {@const isHovered = hoveredIndex === i}
      {@const isDimmed = hoveredIndex !== null && !isHovered}

      <!-- Bar rect with entrance + hover animation via CSS class -->
      <g
        class="bar-group"
        class:bar-group--dimmed={isDimmed}
        class:bar-group--entered={entered}
        style="--bar-index: {Math.min(i, 12)}; --bar-start-y: {yScale(bar.barStart)}px; --bar-h: {barH}px;"
        role="img"
        aria-label="{bar.label}: {fmt(bar.value)}"
        onpointerenter={(e) => {
          hoveredIndex = i;
          tipX = x + bw / 2 + ctx.margin.left;
          tipY = barTopY + ctx.margin.top;
        }}
        onpointerleave={() => { hoveredIndex = null; }}
      >
        <rect
          {x}
          y={barTopY}
          width={bw}
          height={Math.max(1, barH)}
          class="bar-rect"
          class:bar-rect--pos={!bar.isTotal && bar.value >= 0}
          class:bar-rect--neg={!bar.isTotal && bar.value < 0}
          class:bar-rect--total={bar.isTotal}
          rx="2"
        />

        <!-- Value label -->
        {#if showValues}
          {@const labelY = bar.value >= 0 || bar.isTotal ? barTopY - 5 : yScale(bar.barStart) + 14}
          <text
            class="bar-value"
            class:bar-value--total={bar.isTotal}
            x={x + bw / 2}
            y={labelY}
            text-anchor="middle"
          >
            {bar.isTotal
              ? fmt(bar.barEnd)
              : (bar.value >= 0 ? '+' : '') + fmt(bar.value)}
          </text>
        {/if}
      </g>
    {/each}
  {/snippet}

  {#snippet overlay(ctx: ChartContext)}
    {@const hovered = hoveredIndex !== null ? animatedBars[hoveredIndex] : null}
    <ChartTooltip
      x={tipX}
      y={tipY}
      visible={hoveredIndex !== null}
      frameWidth={ctx.width}
    >
      {#snippet children()}
        {#if hovered}
          <div class="tip-label">{hovered.label}</div>
          <div class="tip-value">
            {hovered.isTotal
              ? fmt(hovered.barEnd)
              : (hovered.value >= 0 ? '+' : '') + fmt(hovered.value)}
          </div>
          {#if !hovered.isTotal}
            <div class="tip-running">Running: {fmt(hovered.cumulative)}</div>
          {/if}
        {/if}
      {/snippet}
    </ChartTooltip>
  {/snippet}
</ChartFrame>

<style>
  /* ── Empty state ──────────────────────────────────────────────────────────── */
  .empty-label {
    fill: var(--af-text-muted);
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
  }

  /* ── Bar fills — semantic color IS the encoding here (§4c) ──────────────── */
  .bar-rect--pos   { fill: var(--af-success); }
  .bar-rect--neg   { fill: var(--af-danger); }
  .bar-rect--total { fill: var(--af-inverse-surface); }

  /* ── Hover dimming via group ──────────────────────────────────────────────── */
  .bar-group {
    opacity: 1;
    transition: opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    cursor: default;
  }

  .bar-group--dimmed {
    opacity: 0.35;
  }

  /* ── Entrance: bars scale-up from their start level ─────────────────────── */
  @media (prefers-reduced-motion: no-preference) {
    .bar-rect {
      transform-box: fill-box;
      transform-origin: bottom center;
      transform: scaleY(0);
    }

    .bar-group--entered .bar-rect {
      animation: af-bridge-rise var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
      animation-delay: calc(var(--bar-index, 0) * var(--af-motion-stagger));
    }

    @keyframes af-bridge-rise {
      from { transform: scaleY(0); }
      to   { transform: scaleY(1); }
    }
  }

  /* ── Value labels ─────────────────────────────────────────────────────────── */
  .bar-value {
    fill: var(--af-text-secondary);
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xs);
    font-variant-numeric: tabular-nums lining-nums;
    pointer-events: none;
  }

  .bar-value--total {
    fill: var(--af-text);
    font-weight: var(--af-weight-semibold);
  }

  /* ── Connector lines ─────────────────────────────────────────────────────── */
  .connector {
    stroke: var(--af-border-strong);
    stroke-width: 1;
    stroke-dasharray: none;
  }

  /* ── Zero line ───────────────────────────────────────────────────────────── */
  .zero-line {
    stroke: var(--af-border-strong);
    stroke-width: 1.5;
  }

  /* ── Tooltip ─────────────────────────────────────────────────────────────── */
  .tip-label {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-inverse);
    margin-block-end: 2px;
  }

  .tip-value {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-sm);
    font-variant-numeric: tabular-nums lining-nums;
    font-weight: var(--af-weight-semibold);
    color: var(--af-text-inverse);
  }

  .tip-running {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xs);
    font-variant-numeric: tabular-nums lining-nums;
    color: color-mix(in srgb, var(--af-text-inverse) 70%, transparent);
    margin-block-start: 2px;
  }
</style>
