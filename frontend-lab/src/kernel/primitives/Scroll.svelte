<script lang="ts">
  import type { Snippet } from 'svelte'

  type Axis = 'x' | 'y' | 'both'

  let {
    axis = 'y',
    children,
  }: {
    axis?: Axis
    children: Snippet
  } = $props()
</script>

<!-- Layout doctrine: scrolling is always DECLARED, never accidental.
     Any scroll region other than PageShell's content area uses this. -->
<div
  class="k-scroll"
  style:overflow-x={axis === 'y' ? 'hidden' : 'auto'}
  style:overflow-y={axis === 'x' ? 'hidden' : 'auto'}
>
  {@render children()}
</div>

<style>
  .k-scroll {
    flex: 1;
    min-height: 0;
    min-width: 0;
  }
</style>
