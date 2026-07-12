/**
 * CSRF Protection Utility for Wails App
 *
 * Defense-in-depth for critical financial operations (payments, invoices).
 * While Wails bindings are inherently secure against external requests,
 * CSRF tokens provide additional protection layer.
 *
 * Usage:
 *   import { getToken, refreshToken, clearToken } from '$lib/utils/csrf';
 *
 *   // Before critical operation
 *   const token = await getToken();
 *
 *   // Pass to backend if validation needed
 *   await SomeCriticalOperation(data, token);
 *
 *   // After successful operation
 *   clearToken();
 */

import { GetCSRFToken } from '../../../wailsjs/go/main/App';

let currentToken: string | null = null;

/**
 * Get current CSRF token, generating one if needed
 */
export async function getToken(): Promise<string> {
    if (!currentToken) {
        currentToken = await GetCSRFToken();
    }
    return currentToken;
}

/**
 * Force refresh the CSRF token (get a new one)
 */
export async function refreshToken(): Promise<string> {
    currentToken = await GetCSRFToken();
    return currentToken;
}

/**
 * Clear the current token (call after successful state-changing operation)
 * This ensures a fresh token is used for the next operation
 */
export function clearToken(): void {
    currentToken = null;
}

/**
 * Check if we have a token cached
 */
export function hasToken(): boolean {
    return currentToken !== null;
}
