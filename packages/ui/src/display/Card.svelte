<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface CardProps {
    /** Visual treatment of the card surface */
    variant?: 'default' | 'raised' | 'accent' | 'north-star';
    /** Enable the Lift shadow on hover (§4d) */
    lift?: boolean;
    /** Override card padding with a space token step (1–8). Defaults to --af-card-padding */
    padding?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;
    /** Optional header Snippet rendered above the padding border */
    header?: Snippet;
    /** Optional footer Snippet rendered below the content */
    footer?: Snippet;
    /** Main content */
    children?: Snippet;
    /** HTML element to render as (div or article) */
    as?: 'div' | 'article' | 'section';
    [key: string]: unknown;
  }

  let {
    variant = 'default',
    lift = false,
    padding,
    header,
    footer,
    children,
    as: Tag = 'div',
    ...restProps
  }: CardProps = $props();

  const paddingStyle = $derived(
    padding ? `var(--af-space-${padding})` : 'var(--af-card-padding)'
  );
</script>

<svelte:element
  this={Tag}
  class="af-card af-card--{variant}"
  class:af-card--lift={lift}
  style:--_card-padding={paddingStyle}
  {...restProps}
>
  {#if header}
    <div class="af-card__header">
      {@render header()}
    </div>
  {/if}

  <div class="af-card__body">
    {@render children?.()}
  </div>

  {#if footer}
    <div class="af-card__footer">
      {@render footer()}
    </div>
  {/if}
</svelte:element>

<style>
  .af-card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    overflow: hidden;
    /* R2 optimize: micro-interaction for shadow/border transitions */
    transition:
      box-shadow var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      border-color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  /* Variants */
  .af-card--raised {
    background: var(--af-surface-raised);
  }

  .af-card--accent {
    border-inline-start: 2px solid var(--af-inverse-surface);
    /* Pull inline-start border visual inward — preserve 12px total radius */
    border-start-start-radius: var(--af-radius-sm);
    border-end-start-radius: var(--af-radius-sm);
  }

  .af-card--north-star {
    background: var(--af-inverse-surface);
    border-color: transparent;
    color: var(--af-text-inverse);
  }

  /* Lift — opt-in hover shadow (§4d "the Lift") */
  .af-card--lift:hover,
  .af-card--lift:focus-within {
    box-shadow: var(--af-shadow-lift);
  }

  /* Sections */
  .af-card__header {
    padding: var(--_card-padding);
    border-block-end: 1px solid var(--af-border);
  }

  .af-card--north-star .af-card__header {
    border-block-end-color: color-mix(in srgb, var(--af-text-inverse) 12%, transparent);
  }

  .af-card__body {
    padding: var(--_card-padding);
  }

  .af-card__footer {
    padding: var(--_card-padding);
    border-block-start: 1px solid var(--af-border);
    background: var(--af-surface-raised);
  }

  .af-card--north-star .af-card__footer {
    border-block-start-color: color-mix(in srgb, var(--af-text-inverse) 12%, transparent);
    background: color-mix(in srgb, var(--af-text-inverse) 5%, transparent);
  }

  /* Reduced motion — transitions already use optimize curve which is fast;
     under reduced-motion the global backstop collapses them anyway (base.css) */
</style>
