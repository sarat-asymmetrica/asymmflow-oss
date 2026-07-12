<script lang="ts" module>
  export interface LegendItem {
    label: string;
    /** Defaults to the series palette color at this item's index. */
    color?: string;
  }
</script>

<script lang="ts">
  /**
   * Legend — series key, optionally interactive.
   *
   * When `toggleable`, clicking an item adds/removes its label from the
   * bindable `hidden` array — the chart filters its series accordingly.
   * Hidden items render dimmed with a hollow swatch, so the legend itself
   * shows the state.
   */

  import { seriesColor } from './palette.js';

  interface Props {
    items: LegendItem[];
    toggleable?: boolean;
    /** Labels currently hidden (bindable when toggleable). */
    hidden?: string[];
    class?: string;
  }

  let {
    items,
    toggleable = false,
    hidden = $bindable([]),
    class: className = '',
  }: Props = $props();

  function toggle(label: string) {
    if (!toggleable) return;
    hidden = hidden.includes(label)
      ? hidden.filter((l) => l !== label)
      : [...hidden, label];
  }
</script>

<div class="af-legend {className}" role={toggleable ? 'group' : undefined} aria-label={toggleable ? 'Toggle series' : undefined}>
  {#each items as item, i (item.label)}
    {@const color = item.color ?? seriesColor(i)}
    {@const isHidden = hidden.includes(item.label)}
    {#if toggleable}
      <button
        class="item interactive"
        class:hidden-item={isHidden}
        onclick={() => toggle(item.label)}
        aria-pressed={!isHidden}
      >
        <span class="swatch" class:hollow={isHidden} style:--swatch-color={color}></span>
        {item.label}
      </button>
    {:else}
      <span class="item">
        <span class="swatch" style:--swatch-color={color}></span>
        {item.label}
      </span>
    {/if}
  {/each}
</div>

<style>
  .af-legend {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--af-space-2) var(--af-space-4);
  }

  .item {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-2);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    color: var(--af-text-secondary);
    border: none;
    background: none;
    padding: 0;
  }

  .item.interactive {
    cursor: pointer;
    border-radius: var(--af-radius-sm);
    padding: 2px var(--af-space-1);
    transition: color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .item.interactive:hover {
    color: var(--af-text);
  }

  .item.hidden-item {
    color: var(--af-text-muted);
  }

  .swatch {
    width: 10px;
    height: 10px;
    border-radius: 3px;
    background: var(--swatch-color);
    flex-shrink: 0;
    transition: background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .swatch.hollow {
    background: transparent;
    box-shadow: inset 0 0 0 1.5px var(--swatch-color);
  }
</style>
