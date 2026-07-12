<script lang="ts">
  /**
   * DonutChart — proportional composition via a segmented ring.
   *
   * Segments are ordered as provided (no d3 sort). Hover expands the segment
   * by adjusting outerRadius. Angle transitions run through the geodesic tween
   * engine so mid-flight data updates retarget cleanly. Entrance sweeps in
   * from zero; reduced-motion jumps directly to final state.
   *
   * Center: displays formatted total by default; caller can override with the
   * `center` snippet. Legend + tooltip carry the labels — no text ON the donut.
   *
   * Constitution: packages/DESIGN_CONSTITUTION.md.
   */

  import type { Snippet } from 'svelte';
  import ChartFrame from '../ChartFrame.svelte';
  import Legend from '../Legend.svelte';
  import ChartTooltip from '../ChartTooltip.svelte';
  import { pie, arc as arcGen } from '../scales.js';
  import type { PieArcDatum } from '../scales.js';
  import { seriesColor } from '../palette.js';
  import { createValuesTween } from '../valuesTween.js';
  import { formatCompact, formatPercent } from '../format.js';

  // ─── Props ───────────────────────────────────────────────────────────────────

  interface Item {
    label: string;
    value: number;
  }

  interface Props {
    items: Item[];
    height?: number;
    thickness?: number;
    valueFormat?: (n: number) => string;
    title: string;
    description?: string;
    legend?: boolean;
    centerLabel?: string;
    center?: Snippet;
  }

  let {
    items,
    height = 260,
    thickness = 26,
    valueFormat = formatCompact,
    title,
    description,
    legend = true,
    centerLabel,
    center,
  }: Props = $props();

  // ─── Legend toggle state ──────────────────────────────────────────────────

  let hiddenLabels = $state<string[]>([]);

  const visibleItems = $derived(
    items.filter((it) => !hiddenLabels.includes(it.label))
  );

  const legendItems = $derived(
    items.map((it, i) => ({ label: it.label, color: seriesColor(i) }))
  );

  // ─── Geodesic tween over values ───────────────────────────────────────────

  const rawValues = $derived(visibleItems.map((it) => Math.max(0, it.value)));

  let tweenedValues = $state<number[]>([]);

  // Start with empty tween; $effect below immediately retargets to real values
  // (will jump on first call since lengths differ — that's correct behaviour).
  const tw = createValuesTween([], (vals) => {
    tweenedValues = vals;
  });

  $effect(() => {
    tw.to([...rawValues], { regime: 'stabilize' });
  });

  // ─── Computed pie geometry ────────────────────────────────────────────────

  const total = $derived(tweenedValues.reduce((s, v) => s + v, 0));

  /** d3 pie — no sort (preserve order per spec) */
  const pieGen = pie<number>().sort(null).padAngle(0.012);

  const arcs = $derived(pieGen(tweenedValues));

  // ─── Hover state ─────────────────────────────────────────────────────────

  let hoveredIndex = $state<number | null>(null);

  // ─── Tooltip state ────────────────────────────────────────────────────────

  interface TipState {
    visible: boolean;
    x: number;
    y: number;
    label: string;
    value: number;
    pct: number;
    color: string;
  }

  let tip = $state<TipState>({
    visible: false,
    x: 0,
    y: 0,
    label: '',
    value: 0,
    pct: 0,
    color: '',
  });

  function onSegEnter(
    event: MouseEvent,
    i: number,
    cx: number,
    cy: number,
    svgRect: DOMRect,
  ) {
    hoveredIndex = i;
    const origIdx = items.findIndex((it) => it.label === visibleItems[i]?.label);
    tip = {
      visible: true,
      x: event.clientX - svgRect.left,
      y: event.clientY - svgRect.top,
      label: visibleItems[i]?.label ?? '',
      value: tweenedValues[i] ?? 0,
      pct: total > 0 ? (tweenedValues[i] ?? 0) / total : 0,
      color: seriesColor(origIdx),
    };
  }

  function onSegLeave() {
    hoveredIndex = null;
    tip = { ...tip, visible: false };
  }
</script>

<!-- ─── Legend ────────────────────────────────────────────────────────────── -->
{#if legend && items.length > 0}
  <Legend items={legendItems} toggleable bind:hidden={hiddenLabels} class="af-donut-legend" />
{/if}

<!-- ─── Chart ─────────────────────────────────────────────────────────────── -->
<ChartFrame
  {height}
  {title}
  {description}
  margin={{ top: 8, right: 8, bottom: 8, left: 8 }}
>
  {#snippet children(ctx)}
    {@const cx = ctx.innerWidth / 2}
    {@const cy = ctx.innerHeight / 2}
    {@const outerR = Math.min(cx, cy) - 2}
    {@const innerR = Math.max(0, outerR - thickness)}
    {@const expandBy = 4}

    {#if arcs.length === 0}
      <!-- Empty state — calm centered label, mirrors the timeseries charts. -->
      <text x={cx} y={cy} text-anchor="middle" dominant-baseline="middle" class="empty-label">No data</text>
    {/if}

    <!-- Segments -->
    {#each arcs as d, i (visibleItems[i]?.label ?? i)}
      {@const origIdx = items.findIndex((it) => it.label === visibleItems[i]?.label)}
      {@const isHovered = hoveredIndex === i}
      {@const thisOuter = isHovered ? outerR + expandBy : outerR}
      {@const arcPath = arcGen<PieArcDatum<number>>()
        .innerRadius(innerR)
        .outerRadius(thisOuter)
        .cornerRadius(2)(d)}
      <path
        d={arcPath ?? ''}
        fill={seriesColor(origIdx)}
        opacity={hoveredIndex !== null && !isHovered ? 0.55 : 1}
        class="donut-seg"
        role="img"
        aria-label="{visibleItems[i]?.label}: {valueFormat(tweenedValues[i] ?? 0)} ({formatPercent(total > 0 ? (tweenedValues[i] ?? 0) / total : 0)})"
        transform="translate({cx}, {cy})"
        onmouseenter={(e) => onSegEnter(
          e,
          i,
          cx + ctx.margin.left,
          cy + ctx.margin.top,
          (e.currentTarget as SVGPathElement).ownerSVGElement!.getBoundingClientRect()
        )}
        onmouseleave={onSegLeave}
      />
    {/each}

    <!-- Center content (HTML-in-SVG via foreignObject is messy; use SVG text) -->
    {#if !center && arcs.length > 0}
      <g transform="translate({cx}, {cy})">
        <text class="center-total" text-anchor="middle" dy="-0.1em">
          {valueFormat(total)}
        </text>
        {#if centerLabel}
          <text class="center-label" text-anchor="middle" dy="1.4em">
            {centerLabel}
          </text>
        {/if}
      </g>
    {:else if center && arcs.length > 0}
      <!-- Custom center via foreignObject — constrained to inner circle -->
      <foreignObject
        x={cx - innerR * 0.7}
        y={cy - innerR * 0.5}
        width={innerR * 1.4}
        height={innerR}
      >
        {@render center()}
      </foreignObject>
    {/if}
  {/snippet}

  {#snippet overlay(ctx)}
    <ChartTooltip x={tip.x} y={tip.y} visible={tip.visible} frameWidth={ctx.width}>
      <div class="tip-row">
        <span class="tip-swatch" style:--tc={tip.color}></span>
        <span class="tip-lbl">{tip.label}</span>
      </div>
      <div class="tip-vals">
        <span class="tip-val">{valueFormat(tip.value)}</span>
        <span class="tip-pct">{formatPercent(tip.pct)}</span>
      </div>
    </ChartTooltip>
  {/snippet}
</ChartFrame>

<style>
  /* Entrance: sweep in from 0 angle — achieved by animating stroke-dasharray
     on the full-ring sentinel. Segments themselves fade in; the sweep-in
     visual comes from a clipPath technique: we animate a covering rect,
     but the simplest approach is opacity + scale-in with transform-origin center. */
  @media (prefers-reduced-motion: no-preference) {
    .donut-seg {
      animation: seg-arrive var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
    }
  }

  @keyframes seg-arrive {
    from {
      opacity: 0;
      transform: translate(var(--cx, 0), var(--cy, 0)) scale(0.88);
    }
    to {
      opacity: 1;
    }
  }

  .donut-seg {
    cursor: default;
    transition:
      opacity var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      d var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .center-total {
    font-family: var(--af-font-numeric);
    font-size: var(--af-text-2xl);
    font-weight: var(--af-weight-bold);
    font-variant-numeric: tabular-nums lining-nums;
    fill: var(--af-text);
  }

  /* Empty state — matches the timeseries charts' calm "No data" label. */
  .empty-label {
    fill: var(--af-text-muted);
    font-size: var(--af-text-sm);
    font-family: var(--af-font-body);
  }

  .center-label {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    fill: var(--af-text-muted);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
  }

  /* Tooltip internals */
  .tip-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--af-text-xs);
    margin-bottom: 4px;
  }

  .tip-swatch {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    background: var(--tc);
    flex-shrink: 0;
  }

  .tip-lbl {
    font-weight: var(--af-weight-semibold);
    white-space: nowrap;
  }

  .tip-vals {
    display: flex;
    gap: var(--af-space-3);
    font-size: var(--af-text-xs);
    font-family: var(--af-font-numeric);
    font-variant-numeric: tabular-nums lining-nums;
  }

  .tip-val {
    font-weight: var(--af-weight-semibold);
  }

  .tip-pct {
    color: color-mix(in srgb, currentColor 60%, transparent);
  }

  :global(.af-donut-legend) {
    margin-bottom: var(--af-space-3);
  }
</style>
