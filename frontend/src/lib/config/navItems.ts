// Single source of truth for primary navigation (Wave 9.5 B8).
// One list drives three things that had drifted apart:
//   1. EnterpriseSidebar rendering order
//   2. App.svelte Alt+N shortcut order (visible items, same order → shortcut
//      order always equals visual order)
//   3. App.svelte screenPermissions (shell-level route gate)
// Before this, the sidebar (12 items) and the Alt+N array (9 items, missing
// accounting/reports/settings) disagreed, and screenPermissions omitted
// accounting/reports entirely — so those two routes fell through the "no
// permission required" branch (a silent shell-gate bypass, same class as the
// Wave 9.4 usermanagement fix).

export interface NavItem {
    id: string;
    labelKey: string;
    icon: string;
    permission: string | null; // null = always visible / no permission required
}

export const NAV_ITEMS: NavItem[] = [
    { id: "dashboard", labelKey: "nav.dashboard", icon: "grid", permission: null },
    { id: "opportunities", labelKey: "crm.pipeline", icon: "target", permission: "offers:view" },
    { id: "operations", labelKey: "nav.operations", icon: "layers", permission: "po:view" },
    { id: "finance", labelKey: "nav.finance", icon: "bar-chart", permission: "finance:view" },
    { id: "accounting", labelKey: "nav.accounting", icon: "book", permission: "finance:view" },
    { id: "reports", labelKey: "nav.reports", icon: "file-text", permission: "reports:view" },
    { id: "work", labelKey: "nav.work", icon: "check-square", permission: null },
    { id: "people", labelKey: "nav.people", icon: "briefcase", permission: "hr:view" },
    { id: "notifications", labelKey: "nav.notifications", icon: "bell", permission: null },
    { id: "relationships", labelKey: "nav.relationships", icon: "users", permission: "customers:view" },
    { id: "intelligence", labelKey: "nav.butler", icon: "zap", permission: "intelligence:chat" },
    { id: "settings", labelKey: "nav.settings", icon: "settings", permission: "settings:view" },
];

// Routes that are reachable but not in the primary nav (deep-links, admin
// surfaces, dev-only). They still need a shell-level permission gate so a hash
// navigation can't bypass RBAC. Deployment stays a Settings sub-tab by
// deliberate decision (settings:update-gated, opened from SettingsScreen).
export const EXTRA_SCREEN_PERMISSIONS: Record<string, string | null> = {
    deployment: "settings:update",
    showcase: null, // dev only
    customer360: "customers:view",
    usermanagement: "users:view",
    rfqs: "offers:view", // was an ungated orphan route
};

// Derived: id → permission for the whole shell (nav + extras).
export const SCREEN_PERMISSIONS: Record<string, string | null> = {
    ...Object.fromEntries(NAV_ITEMS.map((i) => [i.id, i.permission])),
    ...EXTRA_SCREEN_PERMISSIONS,
};
