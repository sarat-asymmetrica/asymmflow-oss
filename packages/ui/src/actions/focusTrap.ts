/**
 * Focus containment for layered surfaces (Modal, Drawer, CommandBar).
 * Constitution §2.6: every layered surface traps focus. No exceptions.
 *
 * Usage: <div use:focusTrap={{ active: open, onEscape: close }}>
 */

export interface FocusTrapOptions {
  active?: boolean;
  /** Called on Escape. The surface decides whether to close. */
  onEscape?: () => void;
  /** Focus this selector on activation; falls back to first focusable. */
  initialFocus?: string;
}

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), ' +
  'textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function focusTrap(node: HTMLElement, options: FocusTrapOptions = {}) {
  let opts = { active: true, ...options };
  let previouslyFocused: HTMLElement | null = null;

  function focusables(): HTMLElement[] {
    return Array.from(node.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
      (el) => el.offsetParent !== null || el === document.activeElement,
    );
  }

  function activate() {
    previouslyFocused = document.activeElement as HTMLElement | null;
    // Defer one frame: portal actions re-parent the node AFTER this action
    // mounts (child effects run first), and re-parenting an element that
    // contains focus silently resets focus to <body> — which would strand
    // the Escape/Tab listeners. One frame later the node has landed.
    requestAnimationFrame(() => {
      if (!opts.active || !node.isConnected) return;
      const target = opts.initialFocus
        ? node.querySelector<HTMLElement>(opts.initialFocus)
        : null;
      const el = target ?? focusables()[0];
      if (el) {
        el.focus();
      } else {
        node.setAttribute('tabindex', '-1');
        node.focus();
      }
    });
  }

  function restore() {
    previouslyFocused?.focus();
    previouslyFocused = null;
  }

  function onKeydown(e: KeyboardEvent) {
    if (!opts.active) return;

    if (e.key === 'Escape' && opts.onEscape) {
      e.stopPropagation();
      opts.onEscape();
      return;
    }

    if (e.key !== 'Tab') return;
    const items = focusables();
    if (items.length === 0) {
      e.preventDefault();
      return;
    }
    const first = items[0];
    const last = items[items.length - 1];
    const current = document.activeElement;

    if (e.shiftKey && (current === first || !node.contains(current))) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && (current === last || !node.contains(current))) {
      e.preventDefault();
      first.focus();
    }
  }

  node.addEventListener('keydown', onKeydown);
  if (opts.active) activate();

  return {
    update(next: FocusTrapOptions = {}) {
      const wasActive = opts.active;
      opts = { active: true, ...next };
      if (!wasActive && opts.active) activate();
      if (wasActive && !opts.active) restore();
    },
    destroy() {
      node.removeEventListener('keydown', onKeydown);
      if (opts.active) restore();
    },
  };
}
