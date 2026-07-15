<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    children,
    trailing,
  }: {
    children: Snippet
    /** Right-aligned cluster (e.g. view toggles, secondary actions). */
    trailing?: Snippet
  } = $props()
</script>

<div class="k-toolbar">
  <div class="k-toolbar-main">{@render children()}</div>
  {#if trailing}
    <div class="k-toolbar-trailing">{@render trailing()}</div>
  {/if}
</div>

<style>
  .k-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--k-space-md);
    /* Anti-collapse: clusters wrap instead of squeezing each other. */
    flex-wrap: wrap;
    min-width: 0;
  }
  .k-toolbar-main,
  .k-toolbar-trailing {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    flex-wrap: wrap;
    min-width: 0;
  }
  .k-toolbar-main {
    flex: 1 1 320px;
  }
  .k-toolbar-main > :global(*),
  .k-toolbar-trailing > :global(*) {
    min-width: 0;
  }
</style>
