<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { fade } from 'svelte/transition';

  interface Props {
    open?: boolean;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    closeOnOverlay?: boolean;
    showClose?: boolean;
    children?: import('svelte').Snippet;
    footer?: import('svelte').Snippet;
  }

  let {
    open = $bindable(false),
    title = '',
    size = 'md',
    closeOnOverlay = true,
    showClose = true,
    children,
    footer
  }: Props = $props();

  const dispatch = createEventDispatcher();

  function close() {
    open = false;
    dispatch('close');
  }

  function handleOverlayClick(e: MouseEvent) {
    if (closeOnOverlay && e.target === e.currentTarget) {
      close();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) {
      close();
    }
  }

  const sizes = {
    sm: '380px',
    md: '500px',
    lg: '680px',
    xl: '820px',
  };
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="modal-overlay"
    onclick={handleOverlayClick}
    in:fade={{ duration: 200 }}
    out:fade={{ duration: 150 }}
    role="dialog"
    aria-modal="true"
    aria-labelledby="modal-title"
  >
    <div 
      class="modal-container"
      style="max-width: {sizes[size]}"
      in:fade={{ duration: 150 }}
      out:fade={{ duration: 100 }}
    >
      {#if title || showClose}
        <header class="modal-header">
          {#if title}
            <h2 id="modal-title" class="modal-title">{title}</h2>
          {/if}
          {#if showClose}
            <button class="modal-close" onclick={close} aria-label="Close modal">
              ×
            </button>
          {/if}
        </header>
      {/if}

      <div class="modal-body">
        {@render children?.()}
      </div>

      {#if footer}
        <footer class="modal-footer">
          {@render footer?.()}
        </footer>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(28, 28, 28, 0.4);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: clamp(14px, 2vw, 24px);
    z-index: 1000;
  }

  .modal-container {
    background: var(--surface, #ffffff);
    border: 1px solid rgba(0, 0, 0, 0.08);
    border-radius: 18px;
    width: 100%;
    max-height: calc(100vh - 48px);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    box-shadow:
      0 var(--space-2, 21px) var(--space-4, 55px) rgba(0, 0, 0, 0.15),
      0 0 0 1px rgba(0, 0, 0, 0.05);
    font-family: var(--font-body);
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 16px 18px 14px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--surface, #ffffff);
  }

  .modal-title {
    font-family: var(--font-display);
    font-size: var(--modal-title-size, 22px);
    font-weight: var(--modal-title-weight, 500);
    line-height: 1.2;
    letter-spacing: -0.02em;
    margin: 0;
    color: var(--text-primary, #1c1c1c);
  }

  .modal-close {
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    font-size: 24px;
    color: var(--text-secondary, #57534e);
    cursor: pointer;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s var(--ease-wabi);
  }

  .modal-close:hover {
    color: var(--text-primary, #1c1c1c);
  }

  .modal-body {
    padding: 16px 18px 18px;
    overflow-y: auto;
    flex: 1;
    font-family: var(--font-body);
    font-size: var(--modal-body-size, 14px);
    line-height: var(--modal-line-height, 1.6);
  }

  .modal-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-1, 13px);
    padding: 14px 18px 16px;
    border-top: 1px solid rgba(0, 0, 0, 0.08);
    background: rgba(0, 0, 0, 0.02);
  }

  :global(.modal-container .card:hover) {
    transform: none !important;
    box-shadow: var(--shadow-sm) !important;
  }
</style>
