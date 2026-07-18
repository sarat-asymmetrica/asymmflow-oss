<script lang="ts">
  /* List widget — a plain row list (recent orders, flagged accounts) with an
   * optional left tone accent and a right-aligned value. The generic sibling
   * to ActivityFeed (which is timestamp-oriented) and RankedBarList (which is
   * bar-oriented) — this one is for rows that are just label + detail + value. */
  import type { ListRow, Navigate } from '../hub'

  let {
    rows,
    navigate,
  }: {
    rows: ListRow[]
    navigate?: Navigate
  } = $props()

  function click(row: ListRow) {
    if (row.nav) navigate?.(row.nav)
  }
</script>

<ul class="k-list">
  {#each rows as row, i (row.label + i)}
    <li>
      <svelte:element
        this={row.nav ? 'button' : 'div'}
        class="k-list-row"
        class:clickable={!!row.nav}
        role={row.nav ? 'button' : undefined}
        tabindex={row.nav ? 0 : undefined}
        onclick={row.nav ? () => click(row) : undefined}>
        <span
          class="k-list-accent"
          style:background={row.tone ? `var(--k-tone-${row.tone}-fg)` : 'transparent'}
        ></span>
        <div class="k-list-main">
          <span class="k-list-label">{row.label}</span>
          {#if row.detail}
            <span class="k-list-detail">{row.detail}</span>
          {/if}
        </div>
        {#if row.value !== undefined}
          <span class="k-list-value">{row.value}</span>
        {/if}
      </svelte:element>
    </li>
  {/each}
</ul>

<style>
  .k-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-list-row {
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
  .k-list-row.clickable {
    cursor: pointer;
  }
  .k-list-row.clickable:hover .k-list-label {
    color: var(--text-primary);
    text-decoration: underline;
  }
  .k-list-accent {
    flex: 0 0 3px;
    align-self: stretch;
    border-radius: var(--border-radius-pill);
  }
  .k-list-main {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .k-list-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
  }
  .k-list-detail {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-muted);
  }
  .k-list-value {
    flex: 0 0 auto;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-weight: 600;
    color: var(--text-primary);
    text-align: end;
  }
</style>
