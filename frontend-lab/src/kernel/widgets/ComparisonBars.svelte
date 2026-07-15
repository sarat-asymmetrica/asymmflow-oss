<script lang="ts">
  /* Comparison bars — base vs current period, per metric (YoY / budget vs
   * actual). Two horizontal bars per row on a shared scale, plus a %change
   * badge. Money-only today (ComparisonRow.currency) — no unit prop, unlike
   * RankedBarList, since every comparison in the descriptor is a money line. */
  import type { ComparisonRow } from '../hub'
  import { renderCell } from '../content'

  let {
    rows,
    baseLabel,
    currentLabel,
  }: {
    rows: ComparisonRow[]
    baseLabel: string
    currentLabel: string
  } = $props()

  function pctOfMax(value: number, row: ComparisonRow): number {
    const max = Math.max(Math.abs(row.base), Math.abs(row.current))
    return max > 0 ? (Math.abs(value) / max) * 100 : 0
  }

  function change(row: ComparisonRow): number | null {
    if (row.base === 0) return null
    return ((row.current - row.base) / Math.abs(row.base)) * 100
  }
</script>

<div class="k-cmp">
  <div class="k-cmp-legend">
    <span class="k-cmp-legend-item">
      <span class="k-cmp-swatch k-cmp-swatch-base"></span>{baseLabel}
    </span>
    <span class="k-cmp-legend-item">
      <span class="k-cmp-swatch k-cmp-swatch-current"></span>{currentLabel}
    </span>
  </div>
  <div class="k-cmp-rows">
    {#each rows as row (row.label)}
      {@const delta = change(row)}
      <div class="k-cmp-row">
        <div class="k-cmp-head">
          <span class="k-cmp-label">{row.label}</span>
          {#if delta !== null}
            <span
              class="k-cmp-delta"
              style:color={`var(--k-tone-${delta >= 0 ? 'success' : 'danger'}-fg)`}>
              {delta >= 0 ? '+' : ''}{delta.toFixed(1)}%
            </span>
          {/if}
        </div>
        <div class="k-cmp-bar-row">
          <span class="k-cmp-track">
            <span class="k-cmp-fill k-cmp-fill-base" style:width="{pctOfMax(row.base, row)}%"></span>
          </span>
          <span class="k-cmp-val">{renderCell('money', row.base, row.currency)}</span>
        </div>
        <div class="k-cmp-bar-row">
          <span class="k-cmp-track">
            <span class="k-cmp-fill k-cmp-fill-current" style:width="{pctOfMax(row.current, row)}%"></span>
          </span>
          <span class="k-cmp-val">{renderCell('money', row.current, row.currency)}</span>
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .k-cmp {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-cmp-legend {
    display: flex;
    gap: var(--k-space-md);
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-muted);
  }
  .k-cmp-legend-item {
    display: inline-flex;
    align-items: center;
    gap: var(--k-space-xs);
  }
  .k-cmp-swatch {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    flex-shrink: 0;
  }
  .k-cmp-swatch-base {
    background: var(--onyx-tint-medium);
  }
  .k-cmp-swatch-current {
    background: var(--k-series-1);
  }
  .k-cmp-rows {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-cmp-row {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  .k-cmp-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-cmp-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
  }
  .k-cmp-delta {
    flex-shrink: 0;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 700;
  }
  .k-cmp-bar-row {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-cmp-track {
    flex: 1 1 auto;
    min-width: 0;
    height: 8px;
    border-radius: var(--border-radius-pill);
    background: var(--onyx-tint);
    overflow: hidden;
  }
  .k-cmp-fill {
    display: block;
    height: 100%;
    min-width: 2px;
  }
  .k-cmp-fill-base {
    background: var(--onyx-tint-medium);
  }
  .k-cmp-fill-current {
    background: var(--k-series-1);
  }
  .k-cmp-val {
    flex: 0 0 auto;
    min-width: 88px;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 600;
    color: var(--text-primary);
    text-align: end;
  }
</style>
