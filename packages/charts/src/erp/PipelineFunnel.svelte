<script lang="ts">
  /**
   * PipelineFunnel — horizontal sales pipeline funnel.
   *
   * Stages rendered as horizontal bands in SVG, centered per row, width
   * proportional to value / first stage. seriesColor(0) with stepping opacity
   * (one hue, depth via opacity — constitution monochrome confidence).
   * Trapezoid connectors (--af-tint-medium) show the narrowing flow.
   * Conversion badges between stages: '64% →' style.
   * Labels column left, values at band right edge.
   * Hover: emphasis + tooltip. Entrance: bands sweep from center. Data: tween.
   */

  import { untrack } from 'svelte';
  import ChartFrame from '../ChartFrame.svelte';
  import ChartTooltip from '../ChartTooltip.svelte';
  import { seriesColor } from '../palette.js';
  import { formatCompact, formatPercent } from '../format.js';
  import { createValuesTween } from '../valuesTween.js';
  import type { ChartContext } from '../ChartFrame.svelte';

  // ─── Props ────────────────────────────────────────────────────────────────

  interface Stage {
    label: string;
    value: number;
  }

  interface Props {
    stages: Stage[];
    height?: number;
    valueFormat?: (n: number) => string;
    title: string;
    description?: string;
    showConversion?: boolean;
  }

  let {
    stages,
    height = 300,
    valueFormat,
    title,
    description,
    showConversion = true,
  }: Props = $props();

  const fmt = $derived(valueFormat ?? formatCompact);

  // ─── Tween values ─────────────────────────────────────────────────────────

  // createValuesTween needs an initial array at construction time.
  // untrack silences the Svelte 5 reactive-capture warning — this is intentional:
  // the tween constructor runs once; all subsequent updates flow through $effect.
  const _initVals = untrack(() => stages.map((s) => s.value));
  let tweenedValues = $state(_initVals);

  const tween = createValuesTween(_initVals, (vals) => {
    tweenedValues = vals;
  });

  $effect(() => {
    const target = stages.map((s) => s.value);
    tween.to(target, { regime: 'stabilize' });
  });

  // ─── Hover state ──────────────────────────────────────────────────────────

  let hoveredIndex = $state<number | null>(null);
  let tipX = $state(0);
  let tipY = $state(0);

  // ─── Entrance ─────────────────────────────────────────────────────────────

  let entered = $state(false);

  $effect(() => {
    const id = setTimeout(() => { entered = true; }, 30);
    return () => clearTimeout(id);
  });

  // ─── Layout constants ─────────────────────────────────────────────────────

  const LABEL_COL_W = 120;
  const BAND_GAP = 6;
  const MIN_BAND_H = 28;

  // Opacity ramp: first → 1.0, last → 0.45
  function bandOpacity(i: number, n: number): number {
    if (n <= 1) return 1;
    return 1 - (i / (n - 1)) * 0.55;
  }
</script>

<ChartFrame
  {title}
  {description}
  {height}
  margin={{ left: LABEL_COL_W, right: 80, top: 12, bottom: 12 }}
  class="af-pipeline-funnel"
>
  {#snippet children(ctx: ChartContext)}
    {@const n = stages.length}
    {@const firstVal = tweenedValues[0] || 1}
    {@const maxBandH = Math.max(MIN_BAND_H, Math.floor((ctx.innerHeight - Math.max(0, n - 1) * BAND_GAP) / n))}
    {@const totalH = n * maxBandH + Math.max(0, n - 1) * BAND_GAP}
    {@const offsetY = (ctx.innerHeight - totalH) / 2}
    {@const maxW = ctx.innerWidth}
    {@const color = seriesColor(0)}

    {#if tweenedValues.length === 0}
      <text
        x={ctx.innerWidth / 2}
        y={ctx.innerHeight / 2}
        text-anchor="middle"
        dominant-baseline="middle"
        class="empty-label"
      >No data</text>
    {:else}
    {#each tweenedValues as val, i}
      {@const stage = stages[i]}
      {@const ratio = Math.max(0, val / firstVal)}
      {@const bandW = Math.max(4, ratio * maxW)}
      {@const bandX = (maxW - bandW) / 2}
      {@const bandY = offsetY + i * (maxBandH + BAND_GAP)}
      {@const opacity = bandOpacity(i, n)}
      {@const isHovered = hoveredIndex === i}
      {@const isDimmed = hoveredIndex !== null && !isHovered}

      <!-- Trapezoid connector between stages -->
      {#if i < n - 1}
        {@const nextVal = tweenedValues[i + 1] ?? val}
        {@const nextRatio = Math.max(0, nextVal / firstVal)}
        {@const nextW = Math.max(4, nextRatio * maxW)}
        {@const nextX = (maxW - nextW) / 2}
        {@const topY = bandY + maxBandH}
        {@const botY = bandY + maxBandH + BAND_GAP}
        <polygon
          class="trapezoid"
          points="{bandX},{topY} {bandX + bandW},{topY} {nextX + nextW},{botY} {nextX},{botY}"
        />

        <!-- Conversion badge -->
        {#if showConversion}
          {@const convRatio = val > 0 ? (nextVal / val) : 0}
          {@const badgeX = maxW / 2}
          {@const badgeY = topY + BAND_GAP / 2}
          <text
            class="conversion-badge"
            x={badgeX}
            y={badgeY + 3.5}
            text-anchor="middle"
          >
            {formatPercent(convRatio)} →
          </text>
        {/if}
      {/if}

      <!-- Band group -->
      <g
        class="band-group"
        class:band-group--dimmed={isDimmed}
        class:band-group--entered={entered}
        style="--band-index: {Math.min(i, 12)}; --band-cx: {maxW / 2}px;"
        onpointerenter={() => {
          hoveredIndex = i;
          tipX = maxW / 2 + ctx.margin.left;
          tipY = bandY + maxBandH / 2 + ctx.margin.top;
        }}
        onpointerleave={() => { hoveredIndex = null; }}
        role="img"
        aria-label="{stage.label}: {fmt(val)}"
      >
        <rect
          class="band-rect"
          x={bandX}
          y={bandY}
          width={bandW}
          height={maxBandH}
          fill={color}
          fill-opacity={opacity}
          rx="3"
        />
      </g>

      <!-- Left label (outside the band, in margin area — x relative to plot area so negative) -->
      <text
        class="band-label"
        x={-8}
        y={bandY + maxBandH / 2}
        text-anchor="end"
        dominant-baseline="middle"
      >
        {stage.label}
      </text>

      <!-- Right value (at band's right edge) -->
      <text
        class="band-value"
        x={bandX + bandW + 8}
        y={bandY + maxBandH / 2}
        text-anchor="start"
        dominant-baseline="middle"
      >
        {fmt(val)}
      </text>
    {/each}
    {/if}
  {/snippet}

  {#snippet overlay(ctx: ChartContext)}
    {@const hovered = hoveredIndex !== null ? stages[hoveredIndex] : null}
    {@const firstVal = stages[0]?.value || 1}
    {@const prevVal = hoveredIndex !== null && hoveredIndex > 0 ? tweenedValues[hoveredIndex - 1] : null}
    <ChartTooltip
      x={tipX}
      y={tipY}
      visible={hoveredIndex !== null}
      frameWidth={ctx.width}
    >
      {#snippet children()}
        {#if hovered}
          <div class="tip-label">{hovered.label}</div>
          <div class="tip-value">{fmt(tweenedValues[hoveredIndex!] ?? hovered.value)}</div>
          <div class="tip-meta">
            {formatPercent((tweenedValues[hoveredIndex!] ?? hovered.value) / firstVal)} of pipeline
          </div>
          {#if prevVal !== null}
            <div class="tip-meta">
              {formatPercent((tweenedValues[hoveredIndex!] ?? hovered.value) / prevVal)} from prev
            </div>
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

  /* ── Trapezoid connectors ─────────────────────────────────────────────────── */
  .trapezoid {
    fill: var(--af-tint-medium);
    pointer-events: none;
  }

  /* ── Conversion badges ────────────────────────────────────────────────────── */
  .conversion-badge {
    fill: var(--af-text-muted);
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xs);
    font-variant-numeric: tabular-nums lining-nums;
    pointer-events: none;
  }

  /* ── Band labels ──────────────────────────────────────────────────────────── */
  .band-label {
    fill: var(--af-text);
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    pointer-events: none;
  }

  .band-value {
    fill: var(--af-text-secondary);
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-sm);
    font-variant-numeric: tabular-nums lining-nums;
    pointer-events: none;
  }

  /* ── Band hover dimming ───────────────────────────────────────────────────── */
  .band-group {
    opacity: 1;
    transition: opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
    cursor: default;
  }

  .band-group--dimmed {
    opacity: 0.4;
  }

  .band-rect {
    transition: filter var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .band-group:not(.band-group--dimmed):hover .band-rect {
    filter: brightness(1.05);
  }

  /* ── Entrance: bands sweep from center (scaleX from 0.4, transform-origin center) */
  @media (prefers-reduced-motion: no-preference) {
    .band-group .band-rect {
      transform-box: fill-box;
      transform-origin: center center;
      transform: scaleX(0.4);
      opacity: 0;
    }

    .band-group--entered .band-rect {
      animation: af-funnel-sweep var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
      animation-delay: calc(var(--band-index, 0) * var(--af-motion-stagger));
    }

    @keyframes af-funnel-sweep {
      from {
        transform: scaleX(0.4);
        opacity: 0;
      }
      to {
        transform: scaleX(1);
        opacity: 1;
      }
    }
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

  .tip-meta {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-xs);
    font-variant-numeric: tabular-nums lining-nums;
    color: color-mix(in srgb, var(--af-text-inverse) 70%, transparent);
    margin-block-start: 2px;
  }
</style>
