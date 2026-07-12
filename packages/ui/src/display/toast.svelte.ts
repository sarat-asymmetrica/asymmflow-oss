/**
 * toast.svelte.ts — Svelte 5 rune-based toast store.
 *
 * Named with the .svelte.ts extension so Svelte's compiler processes $state.
 * No svelte/store writable — uses $state directly (runes module pattern).
 *
 * API:
 *   toast.success(message, opts?)
 *   toast.info(message, opts?)
 *   toast.warning(message, opts?)
 *   toast.danger(message, opts?)
 *   toast.dismiss(id)
 *   toast.clear()
 *
 * Access the reactive queue via toast.queue (read-only derived view).
 */

export type ToastSeverity = 'success' | 'info' | 'warning' | 'danger';

export interface ToastItem {
  id: string;
  message: string;
  severity: ToastSeverity;
  /** Auto-dismiss after ms. 0 = persistent. Default: 4500 */
  duration: number;
}

export interface ToastOptions {
  duration?: number;
}

// Unique monotonic id — no Date.now() collisions in rapid sequences
let _seq = 0;
function uid(): string {
  return `af-toast-${++_seq}`;
}

// Internal mutable state — lives in module scope, reactive via $state
let _queue: ToastItem[] = $state([]);

function add(severity: ToastSeverity, message: string, opts: ToastOptions = {}): string {
  const id = uid();
  const duration = opts.duration ?? 4500;
  _queue = [..._queue, { id, message, severity, duration }];
  return id;
}

function dismiss(id: string): void {
  _queue = _queue.filter((t) => t.id !== id);
}

function clear(): void {
  _queue = [];
}

export const toast = {
  /** Read-only reactive queue — bind this in ToastContainer */
  get queue(): ToastItem[] {
    return _queue;
  },
  success: (message: string, opts?: ToastOptions) => add('success', message, opts),
  info: (message: string, opts?: ToastOptions) => add('info', message, opts),
  warning: (message: string, opts?: ToastOptions) => add('warning', message, opts),
  danger: (message: string, opts?: ToastOptions) => add('danger', message, opts),
  dismiss,
  clear,
};
