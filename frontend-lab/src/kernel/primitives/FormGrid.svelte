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

<!-- The wrapper is the size container; the grid queries it. An element
     cannot @container-query its own size — found live when FormGrid
     rendered single-column inside Modal (no ancestor container there). -->
<div class="k-form-grid-container">
  <div class="k-form-grid" data-cols={columns}>
    {@render children()}
  </div>
</div>

<style>
  .k-form-grid-container {
    container-type: inline-size;
    min-width: 0;
  }
  .k-form-grid {
    display: grid;
    gap: var(--k-space-md);
    min-width: 0;
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
