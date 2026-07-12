<script lang="ts">
  /**
   * Sparkline — the inline pulse.
   *
   * A self-contained, margin-less sparkline that sits inside table cells and
   * KPI cards. Monotone curve + optional area fill + end-dot. Draws on entry
   * with a stroke-dash animation, guarded by prefers-reduced-motion.
   *
   * No ChartFrame, no axes, no margins (2px breathing room only).
   * Constitution: no raw hex, no raw ms, prefers-reduced-motion first-class.
   */

  import { scaleLinear, line as d3line, area as d3area, curveMonotoneX } from '../scales.js';
  import { seriesColor } from '../palette.js';

  interface Props {
    data: number[];
    width?: number;
    height?: number;
    /** Defaults to seriesColor(0) = var(--af-chart-1). */
    color?: string;
    showArea?: boolean;
    strokeWidth?: number;
    class?: string;
  }

  let {
    data,
    width = 120,
    height = 32,
    color = seriesColor(0),
    showArea = true,
    strokeWidth = 1.5,
    class: className = '',
  }: Props = $props();

  // 2px breathing room on each side
  const PAD = 2;

  const xScale = $derived(
    scaleLinear()
      .domain([0, Math.max(1, data.length - 1)])
      .range([PAD, width - PAD]),
  );

  const minVal = $derived(data.length > 0 ? Math.min(...data) : 0);
  const maxVal = $derived(data.length > 0 ? Math.max(...data) : 1);

  const yScale = $derived(
    scaleLinear()
      .domain([minVal, maxVal === minVal ? minVal + 1 : maxVal])
      .range([height - PAD, PAD]),
  );

  const linePath = $derived(
    (d3line<number>()
      .x((_, i) => xScale(i))
      .y((d) => yScale(d))
      .curve(curveMonotoneX))(data) ?? ''
  );

  const areaPath = $derived(
    (d3area<number>()
      .x((_, i) => xScale(i))
      .y0(height - PAD)
      .y1((d) => yScale(d))
      .curve(curveMonotoneX))(data) ?? ''
  );

  // End-dot position
  const lastX = $derived(data.length > 0 ? xScale(data.length - 1) : 0);
  const lastY = $derived(data.length > 0 ? yScale(data[data.length - 1]) : 0);

  // Approximate path length for draw-on animation
  const approxLen = $derived(Math.sqrt(width * width + height * height) * 1.5);

  // Unique id for this sparkline instance (animation name isolation)
  let uid = $state(Math.random().toString(36).slice(2, 8));
</script>

<!--
  display: inline-block so it sits inside text flow / table cells.
  overflow: visible so the end-dot doesn't get clipped.
-->
<svg
  {width}
  {height}
  viewBox="0 0 {width} {height}"
  class="af-sparkline {className}"
  role="img"
  aria-label="Sparkline chart"
  style="display: inline-block; overflow: visible; vertical-align: middle;"
>
  <defs>
    <style>
      @keyframes af-sparkline-draw-{uid} {'{'}
        from {'{'} stroke-dashoffset: {approxLen}; {'}'}
        to   {'{'} stroke-dashoffset: 0; {'}'}
      {'}'}
      @media (prefers-reduced-motion: no-preference) {'{'}
        .af-sparkline-line-{uid} {'{'}
          stroke-dasharray: {approxLen};
          stroke-dashoffset: {approxLen};
          animation:
            af-sparkline-draw-{uid}
            var(--af-motion-explore-duration)
            var(--af-motion-explore-ease)
            forwards;
        {'}'}
      {'}'}
    </style>
  </defs>

  {#if data.length > 1}
    {#if showArea}
      <path
        d={areaPath}
        fill={color}
        opacity="0.10"
        stroke="none"
      />
    {/if}

    <path
      class="af-sparkline-line-{uid}"
      d={linePath}
      fill="none"
      stroke={color}
      stroke-width={strokeWidth}
      stroke-linecap="round"
      stroke-linejoin="round"
    />

    <!-- End-dot -->
    <circle cx={lastX} cy={lastY} r="2" fill={color} />
  {:else if data.length === 1}
    <!-- Single-point: just a centered dot -->
    <circle cx={xScale(0)} cy={yScale(data[0])} r="2" fill={color} />
  {/if}
</svg>

<style>
  .af-sparkline {
    flex-shrink: 0;
  }
</style>
