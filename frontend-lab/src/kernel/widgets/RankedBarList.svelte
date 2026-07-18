<script lang="ts">
  /* Ranked bar list — the Top-N widget (top customers, top suppliers, top
   * SKUs by value). Rank + label + inline bar-fill + value, one row each.
   * Reused across 2+ dashboards, so it stays generic: any ContentClass unit. */
  import type { ContentClass } from '../descriptor'
  import type { Navigate, RankedRow } from '../hub'
  import { renderCell } from '../content'

  let {
    rows,
    unit,
    navigate,
  }: {
    rows: RankedRow[]
    unit?: ContentClass
    navigate?: Navigate
  } = $props()

  function click(row: RankedRow) {
    if (row.nav) navigate?.(row.nav)
  }
</script>

<ol class="k-rank-list">
  {#each rows as row (row.rank + row.label)}
    <li>
      <svelte:element
        this={row.nav ? 'button' : 'div'}
        role={row.nav ? 'button' : undefined}
        tabindex={row.nav ? 0 : undefined}
        class="k-rank-row"
        class:clickable={!!row.nav}
        onclick={row.nav ? () => click(row) : undefined}>
        <span class="k-rank-num">{row.rank}</span>
        <div class="k-rank-main">
          <div class="k-rank-top">
            <span class="k-rank-label">{row.label}</span>
            <span class="k-rank-value">{renderCell(unit ?? 'quantity', row.value)}</span>
          </div>
          <div class="k-rank-track">
            <span class="k-rank-fill" style:width="{Math.max(0, Math.min(100, row.pct))}%"></span>
          </div>
          {#if row.sublabel}
            <span class="k-rank-sublabel">{row.sublabel}</span>
          {/if}
        </div>
      </svelte:element>
    </li>
  {/each}
</ol>

<style>
  .k-rank-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    min-width: 0;
  }
  .k-rank-row {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    width: 100%;
    min-width: 0;
    background: none;
    border: none;
    padding: var(--k-space-xs) 0;
    font-family: inherit;
    text-align: start;
  }
  .k-rank-row.clickable {
    cursor: pointer;
  }
  .k-rank-row.clickable:hover .k-rank-label {
    color: var(--text-primary);
    text-decoration: underline;
  }
  .k-rank-num {
    flex: 0 0 20px;
    text-align: end;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(12px * var(--ui-font-scale));
    color: var(--text-muted);
  }
  .k-rank-main {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .k-rank-top {
    display: flex;
    align-items: baseline;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-rank-label {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
  }
  .k-rank-value {
    flex: 0 0 auto;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-weight: 600;
    color: var(--text-primary);
    text-align: end;
  }
  .k-rank-track {
    height: 4px;
    border-radius: var(--border-radius-pill);
    background: var(--onyx-tint);
    overflow: hidden;
  }
  .k-rank-fill {
    display: block;
    height: 100%;
    min-width: 2px;
    background: var(--k-series-1);
  }
  .k-rank-sublabel {
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
</style>
