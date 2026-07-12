<script lang="ts">
    import { run } from 'svelte/legacy';

    import { onDestroy, createEventDispatcher } from 'svelte';
    import { createFocusTrap } from '$lib/utils/focusTrap';

    interface Props {
        open?: boolean;
        title?: string;
        size?: 'sm' | 'md' | 'lg';
        children?: import('svelte').Snippet;
        footer?: import('svelte').Snippet;
    }

    let {
        open = false,
        title = '',
        size = 'md',
        children,
        footer
    }: Props = $props();

    const dispatch = createEventDispatcher();

    let modalElement: HTMLElement = $state();
    let focusTrap: ReturnType<typeof createFocusTrap> | null = $state(null);

    function close() {
        dispatch('close');
    }

    function handleBackdropClick(event: MouseEvent) {
        if (event.target === event.currentTarget) {
            close();
        }
    }

    function handleBackdropKeydown(event: KeyboardEvent) {
        if (event.key === 'Enter' || event.key === ' ') {
            close();
        }
    }

    run(() => {
        if (open && modalElement) {
            focusTrap = createFocusTrap(modalElement, {
                onEscape: close
            });
            focusTrap.activate();
        } else if (focusTrap) {
            focusTrap.deactivate();
            focusTrap = null;
        }
    });

    onDestroy(() => {
        if (focusTrap) {
            focusTrap.deactivate();
        }
    });
</script>

{#if open}
    <div
        class="modal-backdrop"
        onclick={handleBackdropClick}
        onkeydown={handleBackdropKeydown}
        role="button"
        tabindex="-1"
        aria-label="Close modal"
    >
        <div
            bind:this={modalElement}
            class="modal modal-{size}"
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
        >
            <header class="modal-header">
                <h2 id="modal-title">{title}</h2>
                <button
                    type="button"
                    class="close-btn"
                    onclick={close}
                    aria-label="Close modal"
                >
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
                        <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                </button>
            </header>

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
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: var(--z-modal, 1000);
        animation: fadeIn 150ms ease-out;
    }

    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    @keyframes slideUp {
        from {
            opacity: 0;
            transform: translateY(16px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .modal {
        background: var(--surface, #ffffff);
        border-radius: var(--border-radius, 12px);
        max-height: 90vh;
        overflow: auto;
        box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
        animation: slideUp 200ms ease-out;
    }

    .modal-sm { width: 400px; max-width: 95vw; }
    .modal-md { width: 600px; max-width: 95vw; }
    .modal-lg { width: 900px; max-width: 95vw; }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1rem 1.5rem;
        border-bottom: 1px solid var(--border, #e5e5e5);
    }

    .modal-header h2 {
        margin: 0;
        font-family: var(--font-display);
        font-size: var(--modal-title-size, 22px);
        font-weight: var(--modal-title-weight, 500);
        line-height: 1.2;
        letter-spacing: -0.02em;
        color: var(--text-primary, #1d1d1f);
    }

    .close-btn {
        background: none;
        border: none;
        color: var(--text-secondary, #86868b);
        cursor: pointer;
        padding: 0.5rem;
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: background 120ms ease, color 120ms ease;
    }

    .close-btn:hover {
        background: var(--onyx-tint, rgba(29, 29, 31, 0.04));
        color: var(--text-primary, #1d1d1f);
    }

    .close-btn:focus {
        outline: 2px solid var(--carbon, #000);
        outline-offset: 2px;
    }

    .modal-body {
        padding: 1.5rem;
        font-family: var(--font-body);
        font-size: var(--modal-body-size, 14px);
        line-height: var(--modal-line-height, 1.6);
    }

    .modal-footer {
        padding: 1rem 1.5rem;
        border-top: 1px solid var(--border, #e5e5e5);
        display: flex;
        justify-content: flex-end;
        gap: 0.5rem;
    }
</style>
