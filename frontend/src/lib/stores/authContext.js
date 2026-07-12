/**
 * Auth Context Store - RBAC State Management
 * 
 * Provides reactive state for:
 * - Current user information
 * - User permissions
 * - Role-based UI visibility
 * - Session state
 */

import { writable, derived } from 'svelte/store';
import { ListRoles } from '../../../wailsjs/go/main/App';
import { GetCurrentUserStub, GetUserPermissions, HasPermission } from '../../../wailsjs/go/main/InfraService';

// Current user state
export const currentUser = writable(null);
export const userLoading = writable(true);
export const userError = writable(null);

// All available roles
export const roles = writable([]);

// User permissions array
export const permissions = writable([]);

export function normalizePermissionList(value) {
    return Array.isArray(value) ? value : [];
}

/**
 * Derived store: Is user logged in?
 */
export const isAuthenticated = derived(currentUser, $user => !!$user);

/**
 * Derived store: User's role name
 */
export const userRole = derived(currentUser, $user => $user?.role_name || 'Guest');

/**
 * Derived store: Is user a manager?
 */
export const isManager = derived(currentUser, $user =>
    ['sales_manager', 'management'].includes($user?.role?.name)
);

/**
 * Derived store: Is user an accountant?
 */
export const isAccountant = derived(currentUser, $user =>
    ['accountant', 'management'].includes($user?.role?.name)
);

/**
 * Derived store: Is user management?
 */
export const isManagement = derived(currentUser, $user =>
    $user?.role?.name === 'management'
);

/**
 * Permission check function (reactive)
 * Usage: $canRead('customers') in Svelte
 */
export function createPermissionChecker() {
    let perms = [];
    permissions.subscribe(p => perms = normalizePermissionList(p));

    return (permission) => {
        const safePerms = normalizePermissionList(perms);

        // Wildcard access
        if (safePerms.includes('*')) return true;

        // Direct match
        if (safePerms.includes(permission)) return true;

        // Wildcard pattern (e.g., "customers:*" matches "customers:read")
        const [resource] = permission.split(':');
        if (safePerms.includes(`${resource}:*`)) return true;

        return false;
    };
}

/**
 * Navigation visibility based on role
 */
export const navVisibility = derived([currentUser, permissions], ([$user, $perms]) => {
    if (!$user) return {};
    const safePerms = normalizePermissionList($perms);

    const hasPermission = (perm) => {
        if (safePerms.includes('*')) return true;
        if (safePerms.includes(perm)) return true;
        const [resource] = perm.split(':');
        return safePerms.includes(`${resource}:*`);
    };

    return {
        // Sales & Operations (most users see these)
        dashboard: hasPermission('dashboard:read'),
        opportunities: hasPermission('opportunities:read'),
        costing: hasPermission('costing:read'),
        offers: hasPermission('offers:read'),
        orders: hasPermission('orders:read'),
        delivery: hasPermission('delivery:read'),
        customers: hasPermission('customers:read'),
        suppliers: hasPermission('suppliers:read'),
        followups: hasPermission('followups:read'),
        reports: hasPermission('reports:read:sales') || hasPermission('reports:read:finance'),

        // Intelligence (Sales + Management)
        pipeline: hasPermission('pipeline:read'),
        customerIntel: hasPermission('customer_intel:read'),
        competition: hasPermission('competition:read'),

        // ERP Modules (Accountant + Management)
        accounting: hasPermission('accounting:read'),
        inventory: hasPermission('inventory:read'),
        payroll: hasPermission('payroll:read'),

        // Administration (Management only)
        userManagement: hasPermission('users:read') || safePerms.includes('*'),
        authManagement: safePerms.includes('*'),
        security: safePerms.includes('*'),
        settings: hasPermission('settings:read:basic') || hasPermission('settings:read'),
    };
});

/**
 * Initialize auth context - call on app mount
 */
export async function initAuthContext() {
    userLoading.set(true);
    userError.set(null);

    try {
        // Load current user (stub for dev, will be session-based in production)
        const user = await GetCurrentUserStub();
        currentUser.set(user);

        // Load roles
        const rolesList = await ListRoles();
        roles.set(rolesList);

        // Load user permissions
        if (user?.id) {
            const perms = await GetUserPermissions(user.id);
            permissions.set(normalizePermissionList(perms));
        }

        // Only log in development mode to prevent information disclosure
        if (import.meta.env.DEV) {
            console.log('Auth context initialized:', {
                user: user?.full_name,
                role: user?.role_name,
                permissions: normalizePermissionList(await GetUserPermissions(user?.id)).length
            });
        }

    } catch (err) {
        console.error('Failed to initialize auth context:', err);
        userError.set(err.message || 'Failed to load user');

        // Set default permissions for unauthenticated state
        permissions.set([]);
    } finally {
        userLoading.set(false);
    }
}

/**
 * Refresh current user (after profile update, etc.)
 */
export async function refreshUser() {
    try {
        const user = await GetCurrentUserStub();
        currentUser.set(user);

        if (user?.id) {
            const perms = await GetUserPermissions(user.id);
            permissions.set(normalizePermissionList(perms));
        }
    } catch (err) {
        console.error('Failed to refresh user:', err);
    }
}

/**
 * Check permission via backend (for critical operations)
 */
export async function checkPermissionRemote(userId, permission) {
    try {
        return await HasPermission(userId, permission);
    } catch (err) {
        console.error('Permission check failed:', err);
        return false;
    }
}

// Export a convenience function for templates
export const can = createPermissionChecker();
