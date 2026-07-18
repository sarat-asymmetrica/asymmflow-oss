<script lang="ts">
  /* Stat tile grid — sectioned small-stat tiles (e.g. inventory counts by
   * warehouse, aging buckets by age band). StatItem carries no nav today
   * (hub.ts), so `navigate` is accepted for prop-shape parity with the other
   * widgets but currently unused; tiles are always static. */
  import type { Navigate, StatItem } from '../hub'
  import { renderCell } from '../content'

  let {
    sections,
  }: {
    sections: { title?: string; items: StatItem[] }[]
    navigate?: Navigate
  } = $props()
</script>

<div class="k-stat-sections">
  {#each sections as section, i (section.title ?? i)}
    <div class="k-stat-section">
      {#if section.title}
        <h4 class="k-stat-title">{section.title}</h4>
      {/if}
      <div class="k-stat-grid">
        {#each section.items as item (item.label)}
          <div class="k-stat-tile">
            <span class="k-stat-label">{item.label}</span>
            <span
              class="k-stat-value"
              style:color={item.tone ? `var(--k-tone-${item.tone}-fg)` : undefined}>
              {renderCell(item.content ?? 'text', item.value)}
            </span>
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .k-stat-sections {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-stat-title {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    margin: 0 0 var(--k-space-sm);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-stat-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(140px, 100%), 1fr));
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-stat-tile {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .k-stat-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .k-stat-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(18px * var(--ui-font-scale));
    font-weight: 700;
    line-height: 1.1;
    color: var(--text-primary);
    overflow-wrap: anywhere;
  }
</style>
