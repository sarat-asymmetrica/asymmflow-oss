<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    columns = 2,
    children,
  }: {
    /** Max columns at full width; collapses by container width, not viewport. */
    columns?: 1 | 2 | 3
    children: Snippet
  } = $props()
</script>

<div class="k-form-grid" data-cols={columns}>
  {@render children()}
</div>

<style>
  .k-form-grid {
    display: grid;
    gap: var(--k-space-md);
    min-width: 0;
    container-type: inline-size;
    grid-template-columns: 1fr;
  }
  .k-form-grid > :global(*) {
    min-width: 0;
  }
  @container (min-width: 480px) {
    .k-form-grid[data-cols='2'],
    .k-form-grid[data-cols='3'] {
      grid-template-columns: repeat(2, 1fr);
    }
  }
  @container (min-width: 760px) {
    .k-form-grid[data-cols='3'] {
      grid-template-columns: repeat(3, 1fr);
    }
  }
</style>
