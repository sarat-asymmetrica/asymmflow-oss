/**
 * escapeHtml - Prevent XSS attacks by escaping HTML special characters
 *
 * This utility is used whenever user-provided data is interpolated into
 * HTML strings (e.g., DataTable render functions).
 *
 * @param str - The string to escape
 * @returns The escaped string safe for HTML interpolation
 */
export function escapeHtml(str: string | null | undefined): string {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}
