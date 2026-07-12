import { writable } from 'svelte/store';

/**
 * Canonical confirm primitive (Design Constitution Article III.6 + VI.2).
 *
 * Replaces native `window.confirm()` / `window.prompt()` — which are crude,
 * unstyleable, and hazardous inside the WebView shell — with ONE promise-based
 * dialog rendered by <ConfirmHost/> (mounted once in App.svelte, like toasts).
 *
 *   if (await confirm.ask({ title, message, confirmLabel, variant })) { ... }
 *
 * For actions that previously collected a reason via prompt() (e.g. rejections),
 * use askForReason — it returns the captured text alongside the decision:
 *
 *   const r = await confirm.askForReason({ title, message, reasonLabel });
 *   if (r.confirmed) { useReason(r.reason); }
 */

export type ConfirmVariant = 'primary' | 'danger' | 'warning' | 'success';

export interface ConfirmState {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  variant: ConfirmVariant;
  /** When true, the dialog shows a reason field and blocks confirm until filled if required. */
  withReason: boolean;
  reasonLabel: string;
  reasonPlaceholder: string;
  reasonRequired: boolean;
}

const initial: ConfirmState = {
  open: false,
  title: '',
  message: '',
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  variant: 'primary',
  withReason: false,
  reasonLabel: 'Reason',
  reasonPlaceholder: '',
  reasonRequired: false,
};

const { subscribe, set, update } = writable<ConfirmState>({ ...initial });

// The resolver for the currently-open dialog. Only one dialog is live at a time;
// asking again while one is open cancels the previous (resolves false / not confirmed).
let resolver: ((value: { confirmed: boolean; reason: string }) => void) | null = null;

function settle(value: { confirmed: boolean; reason: string }) {
  const r = resolver;
  resolver = null;
  update((s) => ({ ...s, open: false }));
  if (r) r(value);
}

export interface AskOptions {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: ConfirmVariant;
}

export interface AskForReasonOptions extends AskOptions {
  reasonLabel?: string;
  reasonPlaceholder?: string;
  reasonRequired?: boolean;
}

/** Ask a yes/no question. Resolves true only when the user confirms. */
function ask(options: AskOptions): Promise<boolean> {
  return askForReason({ ...options, reasonRequired: false }).then((r) => r.confirmed) as Promise<boolean>;
}

/**
 * Ask a question that also captures a reason. Resolves { confirmed, reason }.
 * When reasonRequired is true the Confirm button stays disabled until non-empty.
 */
function askForReason(
  options: AskForReasonOptions & { __withReason?: boolean },
): Promise<{ confirmed: boolean; reason: string }> {
  // Cancel any in-flight dialog first.
  if (resolver) settle({ confirmed: false, reason: '' });

  const withReason = options.__withReason ?? true;
  return new Promise((resolve) => {
    resolver = resolve;
    set({
      open: true,
      title: options.title,
      message: options.message,
      confirmLabel: options.confirmLabel || 'Confirm',
      cancelLabel: options.cancelLabel || 'Cancel',
      variant: options.variant || 'primary',
      withReason,
      reasonLabel: options.reasonLabel || 'Reason',
      reasonPlaceholder: options.reasonPlaceholder || '',
      reasonRequired: options.reasonRequired ?? false,
    });
  });
}

// Bare ask() must not render a reason field — route it through askForReason with the flag off.
function askPlain(options: AskOptions): Promise<boolean> {
  return askForReason({ ...options, __withReason: false }).then((r) => r.confirmed);
}

/** Called by <ConfirmHost/> only. */
function _resolve(confirmed: boolean, reason: string): void {
  settle({ confirmed, reason });
}

export const confirm = {
  subscribe,
  ask: askPlain,
  askForReason: (options: AskForReasonOptions) => askForReason(options),
  _resolve,
};
