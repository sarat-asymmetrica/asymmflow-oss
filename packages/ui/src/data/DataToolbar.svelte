<script lang="ts">
  /**
   * DataToolbar — @asymmflow/ui
   *
   * Slot-based bar above tables.
   * Left: title + count Snippet.
   * Right: actions Snippet.
   * Glass background; sits flush with the table top.
   * Zero decoration — calm center aesthetic.
   *
   * Constitution: packages/DESIGN_CONSTITUTION.md
   */

  import type { Snippet } from 'svelte';

  interface Props {
    /** Left region: title, count, breadcrumb, etc. */
    left?: Snippet;
    /** Right region: action buttons, filters, search, etc. */
    right?: Snippet;
  }

  let { left, right }: Props = $props();
</script>

<div class="af-toolbar">
  <div class="af-toolbar__left">
    {#if left}
      {@render left()}
    {/if}
  </div>
  <div class="af-toolbar__right">
    {#if right}
      {@render right()}
    {/if}
  </div>
</div>

<style>
  .af-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    /* Row-gap gives breathing room once the right region wraps below. */
    gap: var(--af-space-3) var(--af-space-4);
    flex-wrap: wrap;
    padding: var(--af-space-3) var(--af-space-4);
    background: var(--af-glass-bg);
    backdrop-filter: var(--af-glass-blur);
    -webkit-backdrop-filter: var(--af-glass-blur);
    border: 1px solid var(--af-border);
    border-bottom: none;
    border-radius: var(--af-radius-md) var(--af-radius-md) 0 0;
    min-height: var(--af-header-height);
  }

  .af-toolbar__left {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    min-width: 0;
    /*
     * flex-basis: auto (content size) is what makes the bar WRAP the right
     * region to its own row when cramped, instead of collapsing to 0 and
     * letting the right region overlap. On wide screens it grows to push the
     * right region to the far edge, preserving the space-between look.
     */
    flex: 1 1 auto;
  }

  .af-toolbar__right {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    flex: 0 0 auto;
  }
</style>
