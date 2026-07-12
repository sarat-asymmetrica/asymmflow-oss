/**
 * Focus Trap Utility for Accessible Modals
 * Traps keyboard focus within a container (Tab cycles, Escape to close)
 * WCAG 2.1 AA Compliant
 */

export interface FocusTrapOptions {
    onEscape?: () => void;
}

export function createFocusTrap(container: HTMLElement, options: FocusTrapOptions = {}) {
    const focusableSelectors = [
        'button:not([disabled])',
        'input:not([disabled])',
        'select:not([disabled])',
        'textarea:not([disabled])',
        'a[href]',
        '[tabindex]:not([tabindex="-1"])',
    ].join(', ');

    let previouslyFocused: HTMLElement | null = null;

    function getFocusableElements(): HTMLElement[] {
        return Array.from(container.querySelectorAll(focusableSelectors));
    }

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key === 'Escape' && options.onEscape) {
            options.onEscape();
            return;
        }

        if (event.key !== 'Tab') return;

        const focusable = getFocusableElements();
        if (focusable.length === 0) return;

        const firstElement = focusable[0];
        const lastElement = focusable[focusable.length - 1];

        if (event.shiftKey) {
            // Shift+Tab: going backwards
            if (document.activeElement === firstElement) {
                event.preventDefault();
                lastElement.focus();
            }
        } else {
            // Tab: going forwards
            if (document.activeElement === lastElement) {
                event.preventDefault();
                firstElement.focus();
            }
        }
    }

    function activate() {
        previouslyFocused = document.activeElement as HTMLElement;

        document.addEventListener('keydown', handleKeyDown);

        // Focus first focusable element
        const focusable = getFocusableElements();
        if (focusable.length > 0) {
            // Small delay to ensure modal is rendered
            setTimeout(() => focusable[0].focus(), 10);
        }
    }

    function deactivate() {
        document.removeEventListener('keydown', handleKeyDown);

        // Return focus to previously focused element
        if (previouslyFocused && previouslyFocused.focus) {
            previouslyFocused.focus();
        }
    }

    return { activate, deactivate };
}
