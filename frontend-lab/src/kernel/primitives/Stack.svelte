<script lang="ts">
  import type { Snippet } from 'svelte'

  type Gap = 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  type Align = 'stretch' | 'start' | 'center' | 'end'

  let {
    gap = 'md',
    align = 'stretch',
    children,
  }: {
    gap?: Gap
    align?: Align
    children: Snippet
  } = $props()

  const alignCss: Record<Align, string> = {
    stretch: 'stretch',
    start: 'flex-start',
    center: 'center',
    end: 'flex-end',
  }
</script>

<div class="k-stack" style:gap="var(--k-space-{gap})" style:align-items={alignCss[align]}>
  {@render children()}
</div>

<style>
  .k-stack {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  /* Layout doctrine: every child is overflow-safe by construction. */
  .k-stack > :global(*) {
    min-width: 0;
  }
</style>
