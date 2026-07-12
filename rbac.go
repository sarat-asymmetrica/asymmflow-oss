package main

// ============================================================================
// RBAC MOVED TO app.go
// ============================================================================
//
// All RBAC functionality has been consolidated into app.go to avoid
// duplication and conflicts. See app.go lines 6859+ for:
//
// - Permission constants (PermCustomersView, PermInvoicesCreate, etc.)
// - SeedDefaultRoles() - creates admin/manager/staff roles
// - HasPermission(userID, permission) - checks user permissions
// - GetUserPermissions(userID) - retrieves user permissions
// - GetCurrentUserRole() - returns current user's role
// - CheckPermissionByRole(role, permission) - role-based checking
// - GetRolePermissionsList(roleName) - gets permissions for a role
//
// This file is intentionally empty to prevent duplicate declarations.
// ============================================================================
