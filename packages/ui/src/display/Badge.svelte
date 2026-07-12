<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface BadgeProps {
    /**
     * Visual variant.
     * - primary: inverse-surface background (accent mark)
     * - neutral: raised surface + border (default, most used)
     * - outlined: transparent bg, border only
     */
    variant?: 'primary' | 'neutral' | 'outlined';
    size?: 'sm' | 'md';
    children?: Snippet;
    [key: string]: unknown;
  }

  let {
    variant = 'neutral',
    size = 'md',
    children,
    ...restProps
  }: BadgeProps = $props();
</script>

<span
  class="af-badge af-badge--{variant} af-badge--{size}"
  {...restProps}
>
  {@render children?.()}
</span>

<style>
  .af-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--af-radius-pill);
    font-family: var(--af-font-body);
    font-weight: var(--af-weight-semibold);
    white-space: nowrap;
    line-height: 1;
    letter-spacing: 0.02em;
  }

  /* Sizes */
  .af-badge--sm {
    font-size: var(--af-text-xs);
    padding: var(--af-space-1) var(--af-space-2);
    min-height: 18px;
  }

  .af-badge--md {
    font-size: calc(var(--af-text-xs) * 1.09); /* ~12px */
    padding: calc(var(--af-space-1) + 1px) var(--af-space-3);
    min-height: 22px;
  }

  /* Variants */
  .af-badge--primary {
    background: var(--af-inverse-surface);
    color: var(--af-text-inverse);
    border: 1px solid transparent;
  }

  .af-badge--neutral {
    background: var(--af-surface-raised);
    color: var(--af-text-secondary);
    border: 1px solid var(--af-border);
  }

  .af-badge--outlined {
    background: transparent;
    color: var(--af-text-secondary);
    border: 1px solid var(--af-border-strong);
  }
</style>
