<script lang="ts">
  /* Donut — categorical share of a whole (grade mix, supplier-type mix). The
   * anti-card widget: replaces a wall of flat count tiles with one proportion
   * chart. Coloured by the CVD-validated categorical series palette (--k-series-*,
   * fixed slot order, never cycled). A legend + values give identity, so colour
   * is never the sole channel (dataviz relief rule). Hand-rolled SVG, no lib. */
  import type { WidgetSegment } from '../hub'
  import { formatNumber } from '../format'

  let {
    segments,
    centerLabel,
  }: {
    segments: WidgetSegment[]
    centerLabel?: string | undefined
  } = $props()

  const R = 60
  const C = 2 * Math.PI * R
  const total = $derived(segments.reduce((s, x) => s + x.value, 0))
  // Precompute each arc's dash length + offset (2px visual gap between arcs).
  const arcs = $derived.by(() => {
    let acc = 0
    return segments.map((seg, i) => {
      const frac = total > 0 ? seg.value / total : 0
      const len = Math.max(0, frac * C - 2)
      const offset = -acc * C
      acc += frac
      return { seg, len, offset, slot: (i % 6) + 1 }
    })
  })
</script>

<div class="k-donut">
  <div class="k-donut-ring">
    <svg viewBox="0 0 150 150" role="img" aria-label={centerLabel ?? 'Distribution'}>
      <g transform="rotate(-90 75 75)">
        {#each arcs as a (a.seg.key)}
          <circle
            cx="75"
            cy="75"
            r={R}
            fill="none"
            stroke="var(--k-series-{a.slot})"
            stroke-width="18"
            stroke-dasharray="{a.len} {C - a.len}"
            stroke-dashoffset={a.offset}
          />
        {/each}
      </g>
      {#if total > 0 && centerLabel}
        <text x="75" y="72" class="k-donut-total" text-anchor="middle">{formatNumber(total)}</text>
        <text x="75" y="90" class="k-donut-caption" text-anchor="middle">{centerLabel}</text>
      {/if}
    </svg>
  </div>
  <ul class="k-donut-legend">
    {#each arcs as a (a.seg.key)}
      <li class="k-donut-key">
        <span class="k-donut-dot" style:background="var(--k-series-{a.slot})"></span>
        <span class="k-donut-name">{a.seg.label}</span>
        <span class="k-donut-val">{formatNumber(a.seg.value)}</span>
        <span class="k-donut-pct">{total > 0 ? Math.round((a.seg.value / total) * 100) : 0}%</span>
      </li>
    {/each}
  </ul>
</div>

<style>
  .k-donut {
    display: flex;
    align-items: center;
    gap: var(--k-space-lg);
    flex-wrap: wrap;
    min-width: 0;
  }
  .k-donut-ring {
    flex: 0 0 auto;
    width: 130px;
    height: 130px;
  }
  .k-donut-ring svg {
    width: 100%;
    height: 100%;
  }
  .k-donut-total {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: 20px;
    font-weight: 700;
    fill: var(--text-primary);
  }
  .k-donut-caption {
    font-size: 9px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    fill: var(--text-secondary);
  }
  .k-donut-legend {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    flex: 1 1 160px;
    min-width: 0;
  }
  .k-donut-key {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    font-size: calc(12px * var(--ui-font-scale));
    min-width: 0;
  }
  .k-donut-dot {
    width: 10px;
    height: 10px;
    border-radius: 3px;
    flex-shrink: 0;
  }
  .k-donut-name {
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
    flex: 1 1 auto;
  }
  .k-donut-val {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-weight: 600;
    color: var(--text-primary);
    flex-shrink: 0;
  }
  .k-donut-pct {
    color: var(--text-muted);
    flex-shrink: 0;
    min-width: 34px;
    text-align: right;
  }
</style>
