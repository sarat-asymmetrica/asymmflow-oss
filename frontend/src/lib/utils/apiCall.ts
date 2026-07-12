/**
 * apiCall() - Permission-aware API error wrapper
 *
 * Wraps any Wails API call. Detects "access denied:" anywhere in backend errors
 * and shows a permission-specific toast instead of a generic "Failed to load".
 * Returns null on failure so callers can gracefully handle missing data.
 */

import { toast } from '$lib/stores/toasts';

export async function apiCall<T>(promise: Promise<T>, label = 'data'): Promise<T | null> {
  try {
    return await promise;
  } catch (err: any) {
    const message = typeof err === 'string' ? err : err?.message || (err ? String(err) : 'Unknown error');
    const lower = message.toLowerCase();

    // Backend requirePermission() returns "access denied: permission 'X' required (your role: Y)"
    // Use includes() for defense-in-depth — catches even if wrapped by AppError or prefixed
    if (lower.includes('access denied:')) {
      // The backend message already includes role info, show it directly
      const idx = lower.indexOf('access denied:');
      const detail = message.substring(idx + 'access denied:'.length).trim();
      toast.warning(`Permission denied: ${detail}`);
    } else {
      toast.danger(`Failed to load ${label}: ${message}`);
    }

    return null;
  }
}
