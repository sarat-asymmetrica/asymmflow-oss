/**
 * Abortable Fetch Utilities
 *
 * Prevents race conditions when components unmount or params change.
 * CRITICAL for state stability!
 */

/**
 * AbortableController wrapper
 *
 * Automatically aborts previous requests when new ones are made.
 * Use this in components that fetch data on mount or param changes.
 */
export class AbortableController {
  private controller: AbortController | null = null;

  /**
   * Get abort signal for current request
   */
  get signal(): AbortSignal | undefined {
    return this.controller?.signal;
  }

  /**
   * Start a new abortable operation
   * Automatically aborts the previous one
   */
  start(): AbortSignal {
    this.abort(); // Cancel previous request
    this.controller = new AbortController();
    return this.controller.signal;
  }

  /**
   * Abort the current operation
   */
  abort(): void {
    this.controller?.abort();
    this.controller = null;
  }

  /**
   * Check if operation was aborted
   */
  isAborted(): boolean {
    return this.controller?.signal.aborted ?? false;
  }
}

/**
 * Helper to check if error is an abort error
 */
export function isAbortError(error: unknown): boolean {
  if (error instanceof Error) {
    return error.name === 'AbortError';
  }
  if (error instanceof DOMException) {
    return error.name === 'AbortError';
  }
  return false;
}

/**
 * Fetch with automatic abort on component unmount
 *
 * Usage in Svelte:
 * ```svelte
 * <script>
 *   import { onDestroy } from 'svelte';
 *   import { createAbortableFetch } from '$lib/utils/abortable';
 *
 *   const abortableFetch = createAbortableFetch();
 *
 *   async function loadData() {
 *     try {
 *       const data = await abortableFetch('/api/data');
 *       // handle data
 *     } catch (e) {
 *       if (!isAbortError(e)) {
 *         // handle real errors
 *       }
 *     }
 *   }
 *
 *   onDestroy(() => abortableFetch.abort());
 * </script>
 * ```
 */
export function createAbortableFetch() {
  const controller = new AbortableController();

  return {
    /**
     * Fetch with abort signal
     */
    async fetch(url: string, options: RequestInit = {}): Promise<Response> {
      const signal = controller.start();
      return fetch(url, { ...options, signal });
    },

    /**
     * Fetch JSON with abort signal
     */
    async fetchJson<T>(url: string, options: RequestInit = {}): Promise<T> {
      const signal = controller.start();
      const response = await fetch(url, {
        ...options,
        signal,
        headers: {
          'Accept': 'application/json',
          ...options.headers,
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return response.json();
    },

    /**
     * POST JSON with abort signal
     */
    async postJson<T>(
      url: string,
      data: unknown,
      options: RequestInit = {}
    ): Promise<T> {
      const signal = controller.start();
      const response = await fetch(url, {
        ...options,
        method: 'POST',
        signal,
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          ...options.headers,
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return response.json();
    },

    /**
     * Abort current operation
     */
    abort() {
      controller.abort();
    },

    /**
     * Get the underlying controller
     */
    controller,
  };
}
