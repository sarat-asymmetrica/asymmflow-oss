
<script lang="ts">
    import { run, createBubbler, stopPropagation } from 'svelte/legacy';

    const bubble = createBubbler();
    /**
     * Modal - Shoji-style modal window component
     *
     * Features Ma (間) principle - negative space opening animation
     * where the modal expands from center line, "pushing space out"
     *
     * ACCESSIBILITY FEATURES:
     * - Focus trap (Tab/Shift+Tab cycle within modal)
     * - Focus restoration (returns to trigger element on close)
     * - Keyboard navigation (Escape to close)
     * - ARIA attributes (role, aria-modal, aria-labelledby)
     *
     * @component
     */
    import { createEventDispatcher, onMount, tick } from "svelte";
    import { fade } from "svelte/transition";
    import { cubicOut } from "svelte/easing";
    import { motionMs } from "../../motion";

    /** Event interface for type-safe dispatch */
    interface ModalEvents {
        close: void;
    }

    const dispatch = createEventDispatcher<ModalEvents>();

    

    

    
    interface Props {
        /** Controls modal visibility */
        isOpen?: boolean;
        /** Modal title text */
        title?: string;
        /** Maximum modal width */
        maxWidth?: string; // 448px
        onclose?: () => void;
        children?: import('svelte').Snippet;
    }

    let {
        isOpen = false,
        title = "Modal",
        maxWidth = "28rem",
        onclose,
        children
    }: Props = $props();

    /** ARIA label for accessibility (external reference only) */
    export const ariaLabel: string = title;

    const PHI = 1.618;
    const DURATION_LONG = (1 / PHI) * 1000; // ≈ 618ms

    // Focus management
    let modalContainer: HTMLElement = $state();
    let previouslyFocusedElement: HTMLElement | null = null;
    let focusableElements: HTMLElement[] = [];

    function close() {
        dispatch("close");
        onclose?.();
        restoreFocus();
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Escape') {
            e.preventDefault();
            close();
        }

        // Focus trap: Tab and Shift+Tab
        if (e.key === 'Tab') {
            trapFocus(e);
        }
    }

    /**
     * Focus trap implementation
     * Keeps Tab/Shift+Tab navigation within modal
     */
    function trapFocus(e: KeyboardEvent) {
        if (!modalContainer) return;

        // Get all focusable elements
        const focusables = modalContainer.querySelectorAll<HTMLElement>(
            'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
        );

        focusableElements = Array.from(focusables);

        if (focusableElements.length === 0) return;

        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        // Shift+Tab on first element -> focus last
        if (e.shiftKey && document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
        }
        // Tab on last element -> focus first
        else if (!e.shiftKey && document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
        }
    }

    /**
     * Store currently focused element and focus modal
     */
    async function captureFocus() {
        previouslyFocusedElement = document.activeElement as HTMLElement;
        await tick();
        if (modalContainer) {
            modalContainer.focus();
        }
    }

    /**
     * Restore focus to previously focused element
     */
    function restoreFocus() {
        if (previouslyFocusedElement && previouslyFocusedElement.focus) {
            previouslyFocusedElement.focus();
        }
    }

    // Capture focus when modal opens
    run(() => {
        if (isOpen && modalContainer) {
            captureFocus();
        }
    });

    /**
     * Ma (間) Expansion Transition
     * Simulates "opening a space" rather than just appearing
     * Expands from center line vertically, then horizontally
     */
    function maExpand(_node: HTMLElement, { duration = DURATION_LONG }) {
        return {
            duration,
            css: (t: number) => {
                const eased = cubicOut(t);
                return `
                    transform: scaleY(${eased}) scaleX(${0.8 + 0.2 * eased});
                    opacity: ${eased};
                    clip-path: inset(${50 * (1 - eased)}% 0 ${50 * (1 - eased)}% 0);
                `;
            }
        };
    }

    function handleBackdropClick(e: MouseEvent) {
        // Only close if clicking the backdrop itself, not its children
        if (e.target === e.currentTarget) {
            close();
        }
    }
</script>

<svelte:window onkeydown={handleKeydown}/>

{#if isOpen}
    <!-- Backdrop -->
    <div
        class="modal-backdrop"
        onclick={handleBackdropClick}
        transition:fade={{ duration: motionMs(300) }}
        aria-hidden="true"
    >
        <!-- Modal Container -->
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
            bind:this={modalContainer}
            class="modal-container"
            onclick={stopPropagation(bubble('click'))}
            onkeydown={stopPropagation(bubble('keydown'))}
            role="dialog"
            aria-modal="true"
            aria-labelledby="modal-title"
            tabindex="-1"
            style="max-width: {maxWidth};"
            transition:maExpand={{ duration: DURATION_LONG }}
        >
            <div class="modal-content">
                <!-- Header -->
                <div class="modal-header">
                    <h2 id="modal-title" class="modal-title">
                        {title}
                    </h2>
                    <button
                        onclick={close}
                        class="close-button"
                        aria-label="Close modal"
                        type="button"
                    >
                        <span aria-hidden="true">&times;</span>
                    </button>
                </div>

                <!-- Body -->
                <div class="modal-body">
                    {@render children?.()}
                </div>

                <!-- Footer with Ma (negative space) emphasis -->
                <div class="modal-footer">
                    <button
                        onclick={close}
                        class="footer-button"
                        type="button"
                    >
                        Close
                    </button>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        inset: 0;
        z-index: 50;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 13px; /* φ-based */
        background: color-mix(in srgb, var(--bg-color, #111827) 60%, transparent);
        backdrop-filter: blur(4px);
        cursor: pointer;
    }

    .modal-backdrop:focus {
        outline: none;
    }

    .modal-container {
        position: relative;
        width: 100%;
        overflow: hidden;
        box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
        background: var(--bg-color, #ffffff);
        border-radius: 8px;
        cursor: auto;
        border: 1px solid color-mix(in srgb, var(--text-color, #e5e7eb) 20%, transparent);
    }

    .modal-container:focus {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
    }

    .modal-content {
        position: relative;
        padding: 34px; /* φ-based: 21 × φ ≈ 34 */
    }

    .modal-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 21px; /* φ-based */
    }

    .modal-title {
        font-size: 1.5rem;
        font-weight: 300;
        letter-spacing: 0.025em;
        color: var(--text-color, #111827);
        margin: 0;
    }

    .close-button {
        color: color-mix(in srgb, var(--text-color, #9ca3af) 60%, transparent);
        font-size: 1.5rem;
        line-height: 1;
        border: none;
        background: none;
        cursor: pointer;
        padding: 4px 8px;
        transition: color var(--transition-duration, 0.3s) ease;
    }

    .close-button:hover {
        color: var(--text-color, #111827);
    }

    .close-button:focus-visible {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
        border-radius: 4px;
    }

    .modal-body {
        color: color-mix(in srgb, var(--text-color, #6b7280) 90%, transparent);
        font-size: 0.875rem;
        line-height: 1.625;
        margin-bottom: 34px; /* φ-based */
    }

    /* Ma (間) - Negative Space emphasis in footer */
    .modal-footer {
        margin-top: 55px; /* φ-based: 34 × φ ≈ 55 */
        display: flex;
        justify-content: flex-end;
    }

    .footer-button {
        padding: 13px 34px; /* φ-based */
        border: 1px solid color-mix(in srgb, var(--text-color, #d1d5db) 50%, transparent);
        background: transparent;
        color: var(--text-color, #6b7280);
        font-family: 'Courier New', monospace;
        font-size: 0.75rem;
        text-transform: uppercase;
        letter-spacing: 0.1em;
        cursor: pointer;
        transition: all var(--transition-duration, 0.3s) ease;
    }

    .footer-button:hover {
        border-color: var(--text-color, #111827);
        color: var(--text-color, #111827);
    }

    .footer-button:focus-visible {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
    }
</style>
