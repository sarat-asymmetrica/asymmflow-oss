<script lang="ts">
    import { preventDefault } from 'svelte/legacy';

    import { createEventDispatcher, onMount, onDestroy } from "svelte";
    import { EventsOn, EventsOff } from "../../../../wailsjs/runtime/runtime";
    import { currentUser, permissions } from "$lib/stores/authContext";
    import { t } from "$lib/i18n";
    import { GetDBSyncStatus } from "../../../../wailsjs/go/main/App";
import { SyncNowWithProgress, GetDBSyncSettings } from "../../../../wailsjs/go/main/SyncServiceBinding";
    import { getUnreadNotificationsCount } from "$lib/api/collaboration";
    import { NAV_ITEMS } from "$lib/config/navItems";
    import SyncProgress from "./SyncProgress.svelte";

    const dispatch = createEventDispatcher();

    // Sync status
    let syncStatus: {
        configured: boolean;
        online: boolean;
        sync_enabled: boolean;
        last_sync: string | null;
    } = $state({
        configured: false,
        online: false,
        sync_enabled: false,
        last_sync: null
    });
    let syncLoading = $state(false);
    let statusInterval: number;
    let notificationInterval: number;
    let showSyncProgress = $state(false);
    let unreadNotifications = $state(0);

    // Poll sync status every 10 seconds
    async function checkSyncStatus() {
        try {
            const status = await GetDBSyncStatus() as Record<string, any>;
            if (status.error) {
                console.warn("Sync status unavailable:", status.error);
                return;
            }
            syncStatus = {
                configured: status.configured ?? false,
                online: status.online ?? false,
                sync_enabled: status.sync_enabled ?? false,
                last_sync: status.last_sync ?? null
            };
        } catch (e) {
            console.warn("Failed to get sync status:", e);
        }
    }

    async function loadUnreadNotifications() {
        try {
            unreadNotifications = await getUnreadNotificationsCount();
        } catch {
            unreadNotifications = 0;
        }
    }

    // Manual sync trigger with progress UI
    async function triggerSync() {
        if (syncLoading || !syncStatus.configured || !canManualSync) return;
        syncLoading = true;
        showSyncProgress = true;

        try {
            const result = await SyncNowWithProgress();
            console.log("Manual sync result:", result);
            await checkSyncStatus();
        } catch (e) {
            console.error("Manual sync failed:", e);
        } finally {
            syncLoading = false;
            // SyncProgress component auto-hides after completion
        }
    }

    onMount(() => {
        checkSyncStatus();
        loadUnreadNotifications();
        EventsOn("notifications:new", loadUnreadNotifications);
        EventsOn("notifications:updated", loadUnreadNotifications);
        statusInterval = setInterval(checkSyncStatus, 10000) as unknown as number;
        notificationInterval = setInterval(loadUnreadNotifications, 10000) as unknown as number;
        return () => {
            if (statusInterval) clearInterval(statusInterval);
            if (notificationInterval) clearInterval(notificationInterval);
            EventsOff("notifications:new", "notifications:updated");
        };
    });

    onDestroy(() => {
        if (statusInterval) clearInterval(statusInterval);
        if (notificationInterval) clearInterval(notificationInterval);
        EventsOff("notifications:new", "notifications:updated");
    });


    
    interface Props {
        // Props
        currentScreen?: string;
    }

    let { currentScreen = "dashboard" }: Props = $props();

    // Navigation Items come from the single shared source of truth
    // ($lib/config/navItems) so the sidebar order, the Alt+N shortcut order, and
    // the shell-level permission gate can never drift apart (Wave 9.5 B8).
    // Items without a permission are always visible; items with a permission are
    // HIDDEN (not disabled) if the user lacks it.
    const navItems = NAV_ITEMS;

    // Check if user has a specific permission
    function hasPermission(perm: string | null): boolean {
        if (!perm) return true; // No permission required
        const permissionList = Array.isArray($permissions) ? $permissions : [];
        if (permissionList.length === 0) return false;

        // Wildcard access (admin)
        if (permissionList.includes('*')) return true;

        // Direct match
        if (permissionList.includes(perm)) return true;

        // Resource wildcard (e.g., "finance:*" matches "finance:view")
        const [resource] = perm.split(':');
        if (permissionList.includes(`${resource}:*`)) return true;

        return false;
    }


    function navigate(screenId: string) {
        dispatch("navigate", { screen: screenId });
    }

    // Status indicator color
    let statusColor = $derived(syncStatus.online && syncStatus.sync_enabled ? "var(--status-success, #22c55e)"
                   : syncStatus.configured && !syncStatus.sync_enabled ? "var(--status-warning, #f59e0b)"
                   : syncStatus.configured ? "var(--status-warning, #f59e0b)"
                   : "var(--text-muted, #c7c7c7)");
    let statusLabel = $derived(syncStatus.configured && !syncStatus.sync_enabled ? "Cloud sync paused"
                   : syncStatus.online ? "Cloud sync active"
                   : syncStatus.configured ? "Connecting..."
                   : "Offline mode");
    let canManualSync = $derived(hasPermission("settings:update"));
    let syncTitle = $derived(canManualSync
        ? (syncStatus.sync_enabled && syncStatus.online ? "Click to sync now" : statusLabel)
        : (syncStatus.configured && syncStatus.sync_enabled ? "Background sync runs automatically" : statusLabel));
    // Filter navigation items based on permissions
    // Filters nav items for license-based role access
    let visibleNavItems = $derived(navItems.filter(item => hasPermission(item.permission)));
    // User info
    let userName = $derived($currentUser?.full_name || "Guest");
    let userInitial = $derived(userName.charAt(0).toUpperCase());
    let userRole = $derived($currentUser?.role_name || $currentUser?.role?.display_name || "User");
</script>

<aside class="sidebar">
    <!-- BRAND HEADER -->
    <div class="sidebar-header">
        <div class="logo-mark">PH</div>
        <span class="brand-name">Trading</span>
    </div>

    <!-- NAVIGATION -->
    <nav class="nav-list">
        {#each visibleNavItems as item}
            <a
                href="#{item.id}"
                class="nav-link"
                class:active={currentScreen === item.id}
                onclick={preventDefault(() => navigate(item.id))}
            >
                <span class="nav-row">
                    <span class="nav-label">{t(item.labelKey)}</span>
                    {#if item.id === "notifications" && unreadNotifications > 0}
                        <span class="nav-badge">{unreadNotifications}</span>
                    {/if}
                </span>
                {#if currentScreen === item.id}
                    <div class="active-indicator"></div>
                {/if}
            </a>
        {/each}
    </nav>
    
    <!-- SYNC STATUS -->
    <button class="sync-status" onclick={triggerSync} title={syncTitle} type="button" disabled={!canManualSync || syncLoading || !syncStatus.configured || !syncStatus.sync_enabled}>
        <div class="sync-indicator" style="background-color: {statusColor}"></div>
        <span class="sync-label">{statusLabel}</span>
        {#if syncLoading}
            <span class="sync-spinner"></span>
        {/if}
    </button>

    <!-- USER FOOTER -->
    <div class="sidebar-footer">
         <div class="user-profile">
             <div class="avatar">{userInitial}</div>
             <div class="user-info">
                 <span class="user-name">{userName}</span>
                 <span class="user-role">{userRole}</span>
             </div>
         </div>
    </div>
</aside>

<!-- Sync Progress Modal -->
<SyncProgress bind:show={showSyncProgress} title="Syncing with Cloud" />

<style>
    /* SIDEBAR - CLEAN ENTERPRISE STYLE */
    .sidebar {
        position: fixed;
        left: 0;
        top: 0;
        bottom: 0;
        width: var(--sidebar-width); /* 220px */
        background-color: var(--surface);
        border-right: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        z-index: 50;
    }

    /* HEADER */
    .sidebar-header {
        height: var(--header-height); /* 56px */
        display: flex;
        align-items: center;
        padding: 0 20px;
        border-bottom: 1px solid var(--border);
        gap: 12px;
    }

    .logo-mark {
        width: 28px;
        height: 28px;
        background: var(--carbon);
        color: var(--canvas);
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 12px;
        letter-spacing: -0.02em;
    }

    .brand-name {
        font-weight: 600;
        font-size: 14px;
        color: var(--text-primary);
        letter-spacing: -0.01em;
    }

    /* NAV LIST */
    .nav-list {
        display: flex;
        flex-direction: column;
        padding: 16px 12px;
        gap: 4px;
        flex: 1;
        overflow-y: auto;
    }

    .nav-link {
        display: flex;
        align-items: center;
        justify-content: space-between;
        text-decoration: none;
        color: var(--text-secondary);
        font-size: 14px;
        font-weight: 500;
        padding: 8px 12px;
        border-radius: var(--border-radius-sm);
        transition: all var(--transition-fast);
        position: relative;
    }

    .nav-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .nav-badge {
        min-width: 18px;
        height: 18px;
        padding: 0 6px;
        border-radius: 999px;
        background: #0f766e;
        color: white;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-size: 11px;
        font-weight: 700;
    }

    .nav-link:hover {
        background-color: var(--bg-base);
        color: var(--text-primary);
    }

    .nav-link.active {
        background-color: var(--onyx-tint-strong);
        color: var(--onyx);
        font-weight: 600;
    }

    /* FOOTER */
    .sidebar-footer {
        padding: 16px;
        border-top: 1px solid var(--border);
    }

    .user-profile {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .avatar {
        width: 32px;
        height: 32px;
        background: var(--bg-base);
        color: var(--text-secondary);
        border: 1px solid var(--border);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 12px;
        font-weight: 600;
    }

    .user-info {
        display: flex;
        flex-direction: column;
    }

    .user-name {
        font-size: 13px;
        font-weight: 600;
        color: var(--text-primary);
        line-height: 1.2;
    }

    .user-role {
        font-size: 11px;
        color: var(--text-muted);
    }

    /* SYNC STATUS INDICATOR */
    .sync-status {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        width: 100%;
        border-top: 1px solid var(--border);
        border-left: 0;
        border-right: 0;
        border-bottom: 0;
        background: transparent;
        cursor: pointer;
        transition: background-color var(--transition-fast);
        text-align: left;
        font: inherit;
    }

    .sync-status:hover {
        background-color: var(--bg-base);
    }

    .sync-status:disabled {
        cursor: default;
    }

    .sync-status:disabled:hover {
        background: transparent;
    }

    .sync-indicator {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        flex-shrink: 0;
        animation: pulse 2s ease-in-out infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }

    .sync-label {
        font-size: 11px;
        color: var(--text-muted);
        flex: 1;
    }

    .sync-spinner {
        width: 12px;
        height: 12px;
        border: 2px solid var(--border);
        border-top-color: var(--text-secondary);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @media (max-width: 1024px) {
        .sidebar {
            right: 0;
            bottom: auto;
            width: 100%;
            height: 72px;
            flex-direction: row;
            align-items: center;
            border-right: 0;
            border-bottom: 1px solid var(--border);
            overflow-x: auto;
            overflow-y: hidden;
        }

        .sidebar-header {
            height: 72px;
            flex: 0 0 auto;
            padding: 0 12px;
            border-bottom: 0;
            border-right: 1px solid var(--border);
        }

        .brand-name,
        .sync-status,
        .sidebar-footer {
            display: none;
        }

        .nav-list {
            min-width: 0;
            flex: 1;
            flex-direction: row;
            align-items: center;
            padding: 0 8px;
            gap: 4px;
            overflow-x: auto;
            overflow-y: hidden;
        }

        .nav-link {
            flex: 0 0 auto;
            min-height: 42px;
            padding: 8px 10px;
            white-space: nowrap;
        }

        .active-indicator {
            display: none;
        }
    }
</style>
