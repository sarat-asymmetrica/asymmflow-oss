<script lang="ts">
  import { run } from 'svelte/legacy';

  import { createEventDispatcher, onMount, onDestroy } from 'svelte';

  interface Props {
    open?: boolean;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'full';
    closable?: boolean;
    header?: import('svelte').Snippet;
    children?: import('svelte').Snippet;
    footer?: import('svelte').Snippet;
  }

  let {
    open = $bindable(false),
    title = '',
    size = 'md',
    closable = true,
    header,
    children,
    footer
  }: Props = $props();

  const dispatch = createEventDispatcher();

  let modalElement: HTMLElement = $state();
  let previousFocus: HTMLElement | null = null;


  function openModal() {
    previousFocus = document.activeElement as HTMLElement;
    document.body.style.overflow = 'hidden';

    setTimeout(() => {
      // Focus first focusable element or close button
      const focusable = modalElement?.querySelector(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      ) as HTMLElement;
      focusable?.focus();
    }, 100);
  }

  function closeModal() {
    document.body.style.overflow = '';
    previousFocus?.focus();
  }

  function handleClose() {
    if (closable) {
      open = false;
      dispatch('close');
    }
  }

  function handleBackdropClick(event: MouseEvent) {
    if (closable && event.target === event.currentTarget) {
      handleClose();
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' && closable) {
      handleClose();
    }

    // Trap focus within modal
    if (event.key === 'Tab' && open) {
      const focusableElements = modalElement?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      if (!focusableElements || focusableElements.length === 0) return;

      const firstElement = focusableElements[0] as HTMLElement;
      const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

      if (event.shiftKey && document.activeElement === firstElement) {
        event.preventDefault();
        lastElement.focus();
      } else if (!event.shiftKey && document.activeElement === lastElement) {
        event.preventDefault();
        firstElement.focus();
      }
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    document.body.style.overflow = '';
  });
  run(() => {
    if (open) {
      openModal();
    } else {
      closeModal();
    }
  });
</script>

{#if open}
  <div
    class="modal-backdrop"
    onclick={handleBackdropClick}
    role="presentation"
  >
    <div
      class="modal modal-{size}"
      bind:this={modalElement}
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? 'modal-title' : undefined}
    >
      <!-- Header -->
      {#if title || closable || header}
        <header class="modal-header">
          {#if header}
            {@render header?.()}
          {:else if title}
            <h2 id="modal-title" class="modal-title">{title}</h2>
          {/if}

          {#if closable}
            <button
              class="modal-close"
              onclick={handleClose}
              aria-label="Close modal"
            >
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                <path
                  d="M5 5L15 15M15 5L5 15"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                />
              </svg>
            </button>
          {/if}
        </header>
      {/if}

      <!-- Content -->
      <div class="modal-content">
        {@render children?.()}
      </div>

      <!-- Footer -->
      {#if footer}
        <footer class="modal-footer">
          {@render footer?.()}
        </footer>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: var(--z-modal);
    padding: clamp(16px, 2.5vw, 28px);
    animation: wave10-modal-scrim-in var(--motion-base, 200ms) var(--ease-decelerate, cubic-bezier(0, 0, 0.2, 1));
  }

  .modal {
    background: var(--surface);
    border-radius: var(--border-radius-lg);
    box-shadow: var(--shadow-lg);
    display: flex;
    flex-direction: column;
    max-height: calc(100vh - 56px);
    overflow: hidden;
    animation: wave10-modal-panel-in var(--motion-base, 200ms) var(--ease-decelerate, cubic-bezier(0, 0, 0.2, 1));
  }

  @keyframes wave10-modal-scrim-in {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes wave10-modal-panel-in {
    from { opacity: 0; transform: translateY(6px) scale(0.98); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }

  .modal-sm {
    width: 100%;
    max-width: 400px;
  }

  .modal-md {
    width: 100%;
    max-width: 560px;
  }

  .modal-lg {
    width: 100%;
    max-width: 820px;
  }

  .modal-xl {
    width: 100%;
    max-width: 1040px;
  }

  .modal-full {
    width: calc(100vw - 32px);
    height: calc(100vh - 32px);
  }

  .modal-header {
    padding: 14px 16px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-shrink: 0;
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--surface);
  }

  .modal-title {
    font-family: var(--font-display);
    font-size: var(--modal-title-size, 22px);
    font-weight: var(--modal-title-weight, 500);
    line-height: 1.2;
    letter-spacing: -0.02em;
    color: var(--text-primary);
    margin: 0;
  }

  .modal-close {
    background: none;
    border: none;
    padding: 4px;
    cursor: pointer;
    color: var(--text-secondary);
    transition: color var(--transition-fast);
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
  }

  .modal-close:hover {
    color: var(--text-primary);
  }

  .modal-close:focus-visible {
    outline: 2px solid var(--brand-indigo);
    outline-offset: 2px;
  }

  .modal-content {
    padding: 14px 16px 16px;
    overflow-y: auto;
    flex: 1;
    min-height: 0;
    font-family: var(--font-body);
    font-size: var(--modal-body-size, 14px);
    line-height: var(--modal-line-height, 1.6);
  }

  .modal-footer {
    padding: 14px 16px 16px;
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    flex-shrink: 0;
  }

  :global(.modal .card:hover) {
    transform: none !important;
    box-shadow: var(--shadow-sm) !important;
  }
</style>
