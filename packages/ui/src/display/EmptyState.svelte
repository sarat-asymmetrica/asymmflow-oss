<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface EmptyStateProps {
    title: string;
    description?: string;
    /** Icon slot — render any SVG or Snippet here */
    icon?: Snippet;
    /** Action slot — render a Button or link */
    action?: Snippet;
    [key: string]: unknown;
  }

  let { title, description, icon, action, ...restProps }: EmptyStateProps = $props();
</script>

<div class="af-empty" {...restProps}>
  {#if icon}
    <div class="af-empty__icon" aria-hidden="true">
      {@render icon()}
    </div>
  {/if}

  <h2 class="af-empty__title">{title}</h2>

  {#if description}
    <p class="af-empty__desc">{description}</p>
  {/if}

  {#if action}
    <div class="af-empty__action">
      {@render action()}
    </div>
  {/if}
</div>

<style>
  .af-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    padding: var(--af-space-6);
    gap: var(--af-space-3);
    /* R1 explore: entrance on mount */
    animation: af-explore-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  @keyframes af-explore-in {
    from {
      opacity: 0;
      transform: translateY(12px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .af-empty {
      animation: none;
    }
  }

  .af-empty__icon {
    color: var(--af-text-muted);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-block-end: var(--af-space-2);
  }

  .af-empty__title {
    font-family: var(--af-font-display);
    font-size: var(--af-text-xl);
    font-weight: var(--af-weight-semibold);
    letter-spacing: var(--af-title-tracking);
    line-height: var(--af-leading-tight);
    color: var(--af-text);
    margin: 0;
    max-width: 36ch;
  }

  .af-empty__desc {
    font-family: var(--af-font-body);
    font-size: var(--af-text-md);
    color: var(--af-text-secondary);
    line-height: var(--af-leading-relaxed);
    margin: 0;
    max-width: 44ch;
  }

  .af-empty__action {
    margin-block-start: var(--af-space-2);
  }
</style>
