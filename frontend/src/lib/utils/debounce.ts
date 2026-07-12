/**
 * Debounce Utility for Performance Optimization
 *
 * Prevents excessive function calls during rapid user input (e.g., search).
 * Minimum delay should be 300ms for search operations.
 */

/**
 * Creates a debounced version of a function
 * The function will only execute after `delay` ms have passed since the last call
 */
export function debounce<T extends (...args: any[]) => any>(
    fn: T,
    delay: number
): (...args: Parameters<T>) => void {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    return (...args: Parameters<T>) => {
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
        timeoutId = setTimeout(() => {
            fn(...args);
            timeoutId = null;
        }, delay);
    };
}

/**
 * Creates a debounced value store (Svelte-compatible)
 * Updates to the value are debounced before notifying subscribers
 */
export function createDebouncedValue<T>(initialValue: T, delay: number = 300) {
    let value = initialValue;
    let debouncedValue = initialValue;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const subscribers: Set<(value: T) => void> = new Set();

    function set(newValue: T) {
        value = newValue;

        if (timeoutId) {
            clearTimeout(timeoutId);
        }

        timeoutId = setTimeout(() => {
            debouncedValue = value;
            subscribers.forEach(fn => fn(debouncedValue));
        }, delay);
    }

    function subscribe(fn: (value: T) => void) {
        subscribers.add(fn);
        fn(debouncedValue); // Emit current value immediately

        return () => {
            subscribers.delete(fn);
        };
    }

    function get() {
        return debouncedValue;
    }

    return { set, subscribe, get };
}

/**
 * Throttle utility - ensures function runs at most once per `delay` ms
 * Unlike debounce, throttle allows the first call through immediately
 */
export function throttle<T extends (...args: any[]) => any>(
    fn: T,
    delay: number
): (...args: Parameters<T>) => void {
    let lastCall = 0;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    return (...args: Parameters<T>) => {
        const now = Date.now();
        const timeSinceLastCall = now - lastCall;

        if (timeSinceLastCall >= delay) {
            lastCall = now;
            fn(...args);
        } else if (!timeoutId) {
            timeoutId = setTimeout(() => {
                lastCall = Date.now();
                fn(...args);
                timeoutId = null;
            }, delay - timeSinceLastCall);
        }
    };
}
