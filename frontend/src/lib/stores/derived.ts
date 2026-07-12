/**
 * Derived Store Utilities
 *
 * Prevents manual state synchronization bugs by using reactive derivations.
 * State flows automatically, no manual updates needed!
 */

import { derived, writable, type Readable, type Writable } from 'svelte/store';

/**
 * Async derived store
 *
 * Automatically updates when dependencies change.
 * Handles loading state and errors.
 */
export function asyncDerived<T, R>(
  stores: Readable<T> | Readable<T>[],
  fn: (values: T | T[]) => Promise<R>,
  initialValue: R
): Readable<{ loading: boolean; data: R; error: Error | null }> {
  const { subscribe, set } = writable({
    loading: false,
    data: initialValue,
    error: null as Error | null,
  });

  // Track abort controller for cleanup
  let controller: AbortController | null = null;

  const storesArray = Array.isArray(stores) ? stores : [stores];

  const unsubscribe = derived(storesArray, (values) => values).subscribe(
    async (values) => {
      // Abort previous computation
      controller?.abort();
      controller = new AbortController();

      set({ loading: true, data: initialValue, error: null });

      try {
        const result = await fn(Array.isArray(stores) ? values : values[0]);

        // Only update if not aborted
        if (!controller.signal.aborted) {
          set({ loading: false, data: result, error: null });
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          set({
            loading: false,
            data: initialValue,
            error: error instanceof Error ? error : new Error(String(error)),
          });
        }
      }
    }
  );

  return {
    subscribe: (run) => {
      const unsubStore = subscribe(run);
      return () => {
        unsubStore();
        unsubscribe();
        controller?.abort();
      };
    },
  };
}

/**
 * Filtered derived store
 *
 * Example:
 * const filteredItems = filtered(items, filter, (item, filter) => item.status === filter);
 */
export function filtered<T, F>(
  items: Readable<T[]>,
  filter: Readable<F>,
  predicate: (item: T, filterValue: F) => boolean
): Readable<T[]> {
  return derived([items, filter], ([$items, $filterValue]) => {
    return $items.filter((item) => predicate(item, $filterValue));
  });
}

/**
 * Sorted derived store
 */
export function sorted<T>(
  items: Readable<T[]>,
  compareFn: (a: T, b: T) => number
): Readable<T[]> {
  return derived(items, ($items) => {
    return [...$items].sort(compareFn);
  });
}

/**
 * Paginated derived store
 */
export function paginated<T>(
  items: Readable<T[]>,
  page: Readable<number>,
  pageSize: Readable<number>
): Readable<{
  items: T[];
  page: number;
  pageSize: number;
  totalPages: number;
  totalItems: number;
}> {
  return derived([items, page, pageSize], ([$items, $page, $pageSize]) => {
    const totalItems = $items.length;
    const totalPages = Math.ceil(totalItems / $pageSize);
    const start = $page * $pageSize;
    const end = start + $pageSize;
    const pageItems = $items.slice(start, end);

    return {
      items: pageItems,
      page: $page,
      pageSize: $pageSize,
      totalPages,
      totalItems,
    };
  });
}

/**
 * Grouped derived store
 */
export function grouped<T, K extends string | number>(
  items: Readable<T[]>,
  keyFn: (item: T) => K
): Readable<Record<K, T[]>> {
  return derived(items, ($items) => {
    const groups = {} as Record<K, T[]>;

    for (const item of $items) {
      const key = keyFn(item);
      if (!groups[key]) {
        groups[key] = [];
      }
      groups[key].push(item);
    }

    return groups;
  });
}

/**
 * Aggregated derived store
 *
 * Example:
 * const stats = aggregated(items, {
 *   total: (items) => items.length,
 *   sum: (items) => items.reduce((sum, item) => sum + item.value, 0),
 *   avg: (items) => items.reduce((sum, item) => sum + item.value, 0) / items.length
 * });
 */
export function aggregated<T, R extends Record<string, unknown>>(
  items: Readable<T[]>,
  aggregators: { [K in keyof R]: (items: T[]) => R[K] }
): Readable<R> {
  return derived(items, ($items) => {
    const result = {} as R;

    for (const key in aggregators) {
      result[key] = aggregators[key]($items);
    }

    return result;
  });
}
