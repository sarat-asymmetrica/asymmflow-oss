<script lang="ts">
  import type { Snippet } from 'svelte'

  type Gap = 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  type Align = 'stretch' | 'start' | 'center' | 'end' | 'baseline'
  type Justify = 'start' | 'center' | 'end' | 'between'

  let {
    gap = 'sm',
    align = 'center',
    justify = 'start',
    wrap = false,
    children,
  }: {
    gap?: Gap
    align?: Align
    justify?: Justify
    wrap?: boolean
    children: Snippet
  } = $props()

  const alignCss: Record<Align, string> = {
    stretch: 'stretch',
    start: 'flex-start',
    center: 'center',
    end: 'flex-end',
    baseline: 'baseline',
  }
  const justifyCss: Record<Justify, string> = {
    start: 'flex-start',
    center: 'center',
    end: 'flex-end',
    between: 'space-between',
  }
</script>

<div
  class="k-row"
  style:gap="var(--k-space-{gap})"
  style:align-items={alignCss[align]}
  style:justify-content={justifyCss[justify]}
  style:flex-wrap={wrap ? 'wrap' : 'nowrap'}
>
  {@render children()}
</div>

<style>
  .k-row {
    display: flex;
    flex-direction: row;
    min-width: 0;
  }
  /* Layout doctrine: min-width:0 on every flex child — the classic
   * "table blows out the page" bug is structurally impossible here. */
  .k-row > :global(*) {
    min-width: 0;
  }
</style>
