<script lang="ts">
  
  

  import { createEventDispatcher } from 'svelte';
  interface Props {
    /**
   * Wabi-Sabi Card Component
   * Translucent paper with subtle depth
   */
    variant?: 'default' | 'elevated' | 'outlined' | 'ghost';
    padding?: 'none' | 'sm' | 'md' | 'lg';
    hoverable?: boolean;
    clickable?: boolean;
    header?: import('svelte').Snippet;
    children?: import('svelte').Snippet;
    footer?: import('svelte').Snippet;
  }

  let {
    variant = 'default',
    padding = 'md',
    hoverable = false,
    clickable = false,
    header,
    children,
    footer
  }: Props = $props();
  const dispatch = createEventDispatcher();

  function handleClick(e: MouseEvent) {
    if (clickable) {
      dispatch('click', e);
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (clickable && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      dispatch('click', e);
    }
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
<div
  class="wabi-card {variant} padding-{padding}"
  class:hoverable
  class:clickable
  role={clickable ? 'button' : undefined}
  tabindex={clickable ? 0 : undefined}
  onclick={handleClick}
  onkeydown={handleKeydown}
>
  {#if header}
    <header class="card-header">
      {@render header?.()}
    </header>
  {/if}

  <div class="card-body">
    {@render children?.()}
  </div>

  {#if footer}
    <footer class="card-footer">
      {@render footer?.()}
    </footer>
  {/if}
</div>

<style>
  .wabi-card {
    background: rgba(255, 255, 255, 0.4);
    border: 1px solid rgba(0, 0, 0, 0.05);
    border-radius: var(--space-1, 13px);
    transition: all 0.3s var(--ease-sabi);
    position: relative;
  }

  /* Variants */
  .wabi-card.elevated {
    background: rgba(255, 255, 255, 0.6);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05);
  }

  .wabi-card.outlined {
    background: transparent;
    border: 1px solid rgba(0, 0, 0, 0.1);
  }

  .wabi-card.ghost {
    background: transparent;
    border: none;
  }

  /* Padding */
  .wabi-card.padding-none .card-body { padding: 0; }
  .wabi-card.padding-sm .card-body { padding: var(--fib-2); }
  .wabi-card.padding-md .card-body { padding: var(--fib-3); }
  .wabi-card.padding-lg .card-body { padding: var(--fib-4); }

  /* Interactive states */
  .wabi-card.hoverable:hover,
  .wabi-card.clickable:hover {
    background: rgba(255, 255, 255, 0.7);
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.08);
    transform: translateY(-2px);
  }

  .wabi-card.clickable {
    cursor: pointer;
  }

  .wabi-card.clickable:focus-visible {
    outline: 2px solid var(--color-ink, #1c1c1c);
    outline-offset: 2px;
  }

  /* Header & Footer */
  .card-header {
    padding: var(--fib-3);
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  }

  .card-footer {
    padding: var(--fib-3);
    border-top: 1px solid rgba(0, 0, 0, 0.05);
    background: rgba(0, 0, 0, 0.02);
    border-radius: 0 0 var(--fib-2) var(--fib-2);
  }
</style>
