<script lang="ts">
  import type { Snippet } from 'svelte'

  type Gap = 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'

  let {
    min = '240px',
    gap = 'md',
    children,
  }: {
    /** Minimum column width; the grid auto-fits as many as the container allows. */
    min?: string
    gap?: Gap
    children: Snippet
  } = $props()
</script>

<div class="k-grid" style:gap="var(--k-space-{gap})" style:--k-grid-min={min}>
  {@render children()}
</div>

<style>
  .k-grid {
    display: grid;
    /* min(...) guard: a column can never force the grid wider than its
     * container, even when --k-grid-min exceeds the available width. */
    grid-template-columns: repeat(auto-fill, minmax(min(var(--k-grid-min), 100%), 1fr));
    min-width: 0;
  }
  .k-grid > :global(*) {
    min-width: 0;
  }
</style>
