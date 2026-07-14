<script lang="ts">
  import type { Snippet } from 'svelte'

  type Padding = 'none' | 'md' | 'lg'

  let {
    padding = 'md',
    children,
  }: {
    padding?: Padding
    children: Snippet
  } = $props()

  const paddingCss: Record<Padding, string> = {
    none: '0',
    md: 'var(--card-padding)',
    lg: 'var(--card-padding-lg)',
  }
</script>

<div class="k-card" style:padding={paddingCss[padding]}>
  {@render children()}
</div>

<style>
  .k-card {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius);
    box-shadow: var(--shadow-sm);
    min-width: 0;
    /* Wide content scrolls inside its own region, never past the card. */
    overflow: hidden;
  }
</style>
