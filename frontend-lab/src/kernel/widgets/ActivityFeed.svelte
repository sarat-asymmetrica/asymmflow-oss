<script lang="ts">
  /* Activity feed — a vertical timeline of recent events (audit trail,
   * recent postings). Timestamp-oriented sibling to ListWidget. */
  import type { ActivityItem, Navigate } from '../hub'

  let {
    items,
    emptyMessage,
    navigate,
  }: {
    items: ActivityItem[]
    emptyMessage?: string
    navigate?: Navigate
  } = $props()

  function click(item: ActivityItem) {
    if (item.nav) navigate?.(item.nav)
  }
</script>

{#if items.length === 0}
  <div class="k-activity-empty">{emptyMessage ?? 'Nothing here.'}</div>
{:else}
  <ul class="k-activity">
    {#each items as item, i (item.title + i)}
      <li>
        <svelte:element
          this={item.nav ? 'button' : 'div'}
          role={item.nav ? 'button' : undefined}
          tabindex={item.nav ? 0 : undefined}
          class="k-activity-row"
          class:clickable={!!item.nav}
          onclick={item.nav ? () => click(item) : undefined}>
          <span
            class="k-activity-dot"
            style:background={item.tone ? `var(--k-tone-${item.tone}-fg)` : 'var(--onyx-tint-medium)'}
          ></span>
          <div class="k-activity-main">
            <span class="k-activity-title">{item.title}</span>
            {#if item.subtitle}
              <span class="k-activity-subtitle">{item.subtitle}</span>
            {/if}
          </div>
          {#if item.timestamp}
            <span class="k-activity-time">{item.timestamp}</span>
          {/if}
        </svelte:element>
      </li>
    {/each}
  </ul>
{/if}

<style>
  .k-activity {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-activity-row {
    display: flex;
    align-items: flex-start;
    gap: var(--k-space-sm);
    width: 100%;
    min-width: 0;
    background: none;
    border: none;
    padding: var(--k-space-xs) 0;
    font-family: inherit;
    text-align: start;
  }
  .k-activity-row.clickable {
    cursor: pointer;
  }
  .k-activity-row.clickable:hover .k-activity-title {
    color: var(--text-primary);
    text-decoration: underline;
  }
  .k-activity-dot {
    flex: 0 0 8px;
    width: 8px;
    height: 8px;
    margin-top: 4px;
    border-radius: 50%;
  }
  .k-activity-main {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .k-activity-title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
  }
  .k-activity-subtitle {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-muted);
  }
  .k-activity-time {
    flex: 0 0 auto;
    font-size: calc(11px * var(--ui-font-scale));
    color: var(--text-muted);
    white-space: nowrap;
  }
  .k-activity-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--k-space-lg) 0;
    color: var(--text-muted);
    font-size: calc(12px * var(--ui-font-scale));
    text-align: center;
  }
</style>
