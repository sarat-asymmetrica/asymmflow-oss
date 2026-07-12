/**
 * Fetch with Retry Logic
 *
 * Handles transient network failures with exponential backoff.
 * Essential for production resilience.
 */

export interface FetchWithRetryOptions extends RequestInit {
  retries?: number;
  retryDelay?: number;
  timeout?: number;
  onRetry?: (attempt: number, error: Error) => void;
}

/**
 * Fetch with automatic retry on failure
 *
 * @param url - URL to fetch
 * @param options - Fetch options + retry config
 * @returns Promise<Response>
 */
export async function fetchWithRetry(
  url: string,
  options: FetchWithRetryOptions = {}
): Promise<Response> {
  const {
    retries = 3,
    retryDelay = 1000,
    timeout = 10000,
    onRetry,
    ...fetchOptions
  } = options;

  for (let attempt = 0; attempt < retries; attempt++) {
    try {
      // Add timeout signal
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), timeout);

      const response = await fetch(url, {
        ...fetchOptions,
        signal: fetchOptions.signal || controller.signal,
      });

      clearTimeout(timeoutId);

      // Success - return response
      if (response.ok) {
        return response;
      }

      // Server error (5xx) - retry
      if (response.status >= 500 && attempt < retries - 1) {
        const delay = retryDelay * Math.pow(2, attempt); // Exponential backoff
        onRetry?.(attempt + 1, new Error(`Server error ${response.status}`));
        await new Promise((resolve) => setTimeout(resolve, delay));
        continue;
      }

      // Client error (4xx) - don't retry, throw immediately
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    } catch (error) {
      // Last attempt - throw error
      if (attempt === retries - 1) {
        throw error;
      }

      // AbortError from timeout or cancellation - throw immediately (no retry)
      if (error instanceof Error && error.name === 'AbortError') {
        throw error;
      }

      // Network error - retry with exponential backoff
      const delay = retryDelay * Math.pow(2, attempt);
      onRetry?.(attempt + 1, error instanceof Error ? error : new Error(String(error)));
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }

  throw new Error('Retry limit exceeded');
}

/**
 * Fetch JSON with retry
 */
export async function fetchJsonWithRetry<T>(
  url: string,
  options: FetchWithRetryOptions = {}
): Promise<T> {
  const response = await fetchWithRetry(url, {
    ...options,
    headers: {
      'Accept': 'application/json',
      ...options.headers,
    },
  });

  return response.json();
}

/**
 * POST JSON with retry
 */
export async function postJsonWithRetry<T>(
  url: string,
  data: unknown,
  options: FetchWithRetryOptions = {}
): Promise<T> {
  const response = await fetchWithRetry(url, {
    ...options,
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...options.headers,
    },
    body: JSON.stringify(data),
  });

  return response.json();
}
