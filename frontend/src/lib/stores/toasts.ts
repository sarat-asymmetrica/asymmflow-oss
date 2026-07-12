import { writable } from 'svelte/store';

export type ToastType = 'success' | 'warning' | 'danger' | 'info';

export interface Toast {
  id: string;
  message: string;
  type: ToastType;
  duration: number;
  showBrush: boolean;
}

function generateId(): string {
  return `toast-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

const { subscribe, update } = writable<Toast[]>([]);

function add(options: Partial<Toast> & { message: string }): string {
  const newToast: Toast = {
    id: generateId(),
    message: options.message,
    type: options.type || 'info',
    duration: options.duration ?? 4000,
    showBrush: options.showBrush ?? true,
  };

  update((toasts) => [...toasts, newToast]);
  return newToast.id;
}

function dismiss(id: string): void {
  update(toasts => toasts.filter(t => t.id !== id));
}

function clear(): void {
  update(() => []);
}

export const toasts = {
  subscribe,
  add,
  dismiss,
  clear,
  success: (message: string, duration?: number) => add({ message, type: 'success', duration }),
  warning: (message: string, duration?: number) => add({ message, type: 'warning', duration }),
  danger: (message: string, duration?: number) => add({ message, type: 'danger', duration }),
  info: (message: string, duration?: number) => add({ message, type: 'info', duration }),
};

// Re-export convenience functions for direct import
export const toast = {
  success: (message: string, duration?: number) => add({ message, type: 'success', duration }),
  warning: (message: string, duration?: number) => add({ message, type: 'warning', duration }),
  danger: (message: string, duration?: number) => add({ message, type: 'danger', duration }),
  info: (message: string, duration?: number) => add({ message, type: 'info', duration }),
  dismiss,
  clear,
};
