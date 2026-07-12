<script lang="ts">
    import { createBubbler, stopPropagation } from 'svelte/legacy';

    const bubble = createBubbler();
    import { onMount } from "svelte";
    import { fade } from "svelte/transition";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import {
        ListUsers } from "../../../wailsjs/go/main/App";
import { ListRoles, GetAuditLogs, GetRolePermissionsList, CreateUser, UpdateUser } from "../../../wailsjs/go/main/InfraService";
    import { toast } from "../stores/toasts";

    let activeTab = $state("users"); // users, roles, audit
    let loading = $state(false);

    // Data
    let users: any[] = $state([]);
    let roles: any[] = $state([]);
    let auditLogs: any[] = $state([]);
    let searchQuery = $state("");
    let showUserModal = $state(false);
    let editingUser: any = $state(null);
    let savingUser = $state(false);
    let userForm = $state({
        username: "",
        email: "",
        password: "",
        full_name: "",
        department: "",
        job_title: "",
        role_id: "",
        is_active: true,
    });
    let permissionRole: any = $state(null);
    let rolePermissions: string[] = $state([]);
    let loadingRolePermissions = $state(false);

    async function loadData() {
        loading = true;
        try {
            if (!window.go) {
                // Mock
                users = [
                    {
                        id: 1,
                        username: "admin",
                        full_name: "Admin User",
                        role: "Administrator",
                        status: "Active",
                    },
                    {
                        id: 2,
                        username: "jdoe",
                        full_name: "John Doe",
                        role: "Sales",
                        status: "Active",
                    },
                    {
                        id: 3,
                        username: "asmith",
                        full_name: "Alice Smith",
                        role: "Operations",
                        status: "Inactive",
                    },
                ];
                roles = [
                    { id: 1, name: "Administrator" },
                    { id: 2, name: "Sales" },
                    { id: 3, name: "Operations" },
                ];
                auditLogs = [
                    {
                        ts: new Date().toISOString(),
                        action: "LOGIN",
                        user: "admin",
                        details: "Successful login",
                    },
                    {
                        ts: new Date().toISOString(),
                        action: "UPDATE_USER",
                        user: "admin",
                        details: "Updated user jdoe",
                    },
                ];
            } else {
                const [u, r] = await Promise.all([
                    ListUsers().catch(() => []),
                    ListRoles().catch(() => []),
                ]);
                users = u || [];
                roles = r || [];
            }
        } catch (e) {
            toast.danger("Load failed");
        } finally {
            loading = false;
        }
    }

    async function fetchAudit() {
        if (auditLogs.length > 0) return; // cache slightly
        loading = true;
        try {
            if (window.go) {
                auditLogs = (await GetAuditLogs(50, "all", "")) || [];
            }
        } catch (e) {
            console.error('Failed to load audit logs:', e);
            toast.danger('Failed to load audit logs');
        } finally {
            loading = false;
        }
    }

    function roleId(role: any) {
        return String(role?.id || role?.ID || "");
    }

    function roleLabel(role: any) {
        return role?.display_name || role?.name || role?.role_name || "Role";
    }

    function userRoleLabel(user: any) {
        return user?.role_name || user?.role?.display_name || user?.role?.name || user?.role || "Unassigned";
    }

    function isUserActive(user: any) {
        if (typeof user?.is_active === "boolean") return user.is_active;
        return String(user?.status || "").toLowerCase() !== "inactive";
    }

    function resetUserForm() {
        userForm = {
            username: "",
            email: "",
            password: "",
            full_name: "",
            department: "",
            job_title: "",
            role_id: roleId(roles[0]),
            is_active: true,
        };
    }

    function openCreateUser() {
        editingUser = null;
        resetUserForm();
        showUserModal = true;
    }

    function openEditUser(user: any) {
        editingUser = user;
        userForm = {
            username: user?.username || "",
            email: user?.email || "",
            password: "",
            full_name: user?.full_name || user?.display_name || "",
            department: user?.department || "",
            job_title: user?.job_title || "",
            role_id: String(user?.role_id || user?.role?.id || roleId(roles[0])),
            is_active: isUserActive(user),
        };
        showUserModal = true;
    }

    async function saveUser() {
        if (!userForm.username.trim() || !userForm.email.trim() || !userForm.full_name.trim()) {
            toast.warning("Name, username, and email are required");
            return;
        }
        if (!editingUser && !userForm.password.trim()) {
            toast.warning("Password is required for new users");
            return;
        }
        if (!userForm.role_id) {
            toast.warning("Choose a role");
            return;
        }

        savingUser = true;
        try {
            if (window.go) {
                if (editingUser) {
                    await UpdateUser(
                        String(editingUser.id),
                        userForm.full_name.trim(),
                        userForm.email.trim(),
                        userForm.department.trim(),
                        userForm.job_title.trim(),
                        userForm.role_id,
                        userForm.is_active,
                    );
                    toast.success("User updated");
                } else {
                    await CreateUser(
                        userForm.username.trim(),
                        userForm.email.trim(),
                        userForm.password,
                        userForm.full_name.trim(),
                        userForm.department.trim(),
                        userForm.job_title.trim(),
                        userForm.role_id,
                    );
                    toast.success("User created");
                }
                await loadData();
            } else if (editingUser) {
                users = users.map((user) =>
                    user.id === editingUser.id
                        ? { ...user, ...userForm, status: userForm.is_active ? "Active" : "Inactive" }
                        : user,
                );
                toast.success("User updated");
            } else {
                users = [
                    ...users,
                    {
                        id: Date.now(),
                        username: userForm.username,
                        full_name: userForm.full_name,
                        role: roles.find((role) => roleId(role) === userForm.role_id)?.name || "Staff",
                        status: "Active",
                    },
                ];
                toast.success("User created");
            }
            showUserModal = false;
        } catch (e) {
            toast.danger(`User save failed: ${String(e)}`);
        } finally {
            savingUser = false;
        }
    }

    async function openRolePermissions(role: any) {
        permissionRole = role;
        rolePermissions = [];
        loadingRolePermissions = true;
        try {
            if (window.go) {
                rolePermissions = (await GetRolePermissionsList(role?.name || roleLabel(role))) || [];
            } else if (role?.permissions) {
                rolePermissions = JSON.parse(role.permissions);
            }
        } catch (e) {
            if (role?.permissions) {
                try {
                    rolePermissions = JSON.parse(role.permissions);
                } catch {
                    rolePermissions = [];
                }
            }
            toast.danger(`Failed to load permissions: ${String(e)}`);
        } finally {
            loadingRolePermissions = false;
        }
    }

    let filteredUsers = $derived(users.filter(
        (u) =>
            !searchQuery ||
            (u.full_name || u.username || "").toLowerCase().includes(searchQuery.toLowerCase()),
    ));

    onMount(loadData);
</script>

<div class="page">
    <header class="header">
        <div class="header-content">
            <h1>Team.</h1>
            <p class="subtitle">User Management & Security</p>
        </div>
        {#if activeTab === "users"}
            <button class="btn-primary" onclick={openCreateUser}>+ Add User</button>
        {/if}
    </header>

    <div class="tabs">
        <button
            class="tab"
            class:active={activeTab === "users"}
            onclick={() => (activeTab = "users")}>Users</button
        >
        <button
            class="tab"
            class:active={activeTab === "roles"}
            onclick={() => (activeTab = "roles")}>Roles</button
        >
        <button
            class="tab"
            class:active={activeTab === "audit"}
            onclick={() => {
                activeTab = "audit";
                fetchAudit();
            }}>Audit Logs</button
        >
    </div>

    <main class="content-area">
        {#if loading}
            <div class="loading"><WabiSpinner size="lg" /></div>
        {:else if activeTab === "users"}
            <div class="toolbar">
                <input
                    type="text"
                    placeholder="Search users..."
                    bind:value={searchQuery}
                    class="search-input"
                />
            </div>
            <div class="table-card" in:fade>
                <table>
                    <thead
                        ><tr
                            ><th>User</th><th>Role</th><th>Status</th><th
                                class="right">Actions</th
                            ></tr
                        ></thead
                    >
                    <tbody>
                        {#each filteredUsers as u}
                            <tr>
                                <td>
                                    <div class="user-cell">
                                        <div class="avatar">
                                            {(u.full_name || u.username || "U")[0]}
                                        </div>
                                        <div class="info">
                                            <div class="name">
                                                {u.full_name || u.display_name || u.username}
                                            </div>
                                            <div class="sub">@{u.username}</div>
                                        </div>
                                    </div>
                                </td>
                                <td><span class="badge role">{userRoleLabel(u)}</span></td
                                >
                                <td
                                    ><span
                                        class="dot {isUserActive(u)
                                            ? 'green'
                                            : 'red'}"
                                    ></span>
                                    {isUserActive(u) ? "Active" : "Inactive"}</td
                                >
                                <td class="right"
                                    ><button class="btn-sm" onclick={() => openEditUser(u)}>Edit</button></td
                                >
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {:else if activeTab === "roles"}
            <div class="grid-roles" in:fade>
                {#each roles as r}
                    <div class="card role-card">
                        <h3>{roleLabel(r)}</h3>
                        <p>{r.description || "Permissions managed via configuration."}</p>
                        <button class="btn-txt" onclick={() => openRolePermissions(r)}>View Permissions</button>
                    </div>
                {/each}
            </div>
        {:else if activeTab === "audit"}
            <div class="table-card" in:fade>
                <table>
                    <thead
                        ><tr
                            ><th>Timestamp</th><th>Action</th><th>User</th><th
                                >Details</th
                            ></tr
                        ></thead
                    >
                    <tbody>
                        {#each auditLogs as log}
                            <tr>
                                <td class="mono"
                                    >{new Date(log.created_at || log.timestamp || log.ts || Date.now()).toLocaleString()}</td
                                >
                                <td
                                    ><span class="badge neutral"
                                        >{log.action}</span
                                    ></td
                                >
                                <td class="bold">{log.user || log.user_id || "System"}</td>
                                <td class="dim">{log.details || log.resource || "Audit event"}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </main>
</div>

{#if showUserModal}
    <div class="modal-backdrop" role="button" tabindex="0" onclick={() => (showUserModal = false)} onkeydown={(event) => event.key === "Escape" && (showUserModal = false)}>
        <div class="modal-card" role="presentation" tabindex="-1" onclick={stopPropagation(bubble('click'))} onkeydown={stopPropagation(bubble('keydown'))}>
            <div class="modal-head">
                <div>
                    <h2>{editingUser ? "Edit User" : "Add User"}</h2>
                    <p>{editingUser ? "Update role, profile, and active status." : "Create a backend user account with a role."}</p>
                </div>
                <button class="btn-sm" onclick={() => (showUserModal = false)}>Close</button>
            </div>
            <div class="form-grid">
                <label>
                    <span>Full Name</span>
                    <input bind:value={userForm.full_name} placeholder="Full name" />
                </label>
                <label>
                    <span>Username</span>
                    <input bind:value={userForm.username} placeholder="username" disabled={Boolean(editingUser)} />
                </label>
                <label>
                    <span>Email</span>
                    <input type="email" bind:value={userForm.email} placeholder="name@company.com" />
                </label>
                <label>
                    <span>{editingUser ? "Password" : "Password *"}</span>
                    <input type="password" bind:value={userForm.password} placeholder={editingUser ? "Unchanged" : "Temporary password"} disabled={Boolean(editingUser)} />
                </label>
                <label>
                    <span>Department</span>
                    <input bind:value={userForm.department} placeholder="Sales, Operations, Finance..." />
                </label>
                <label>
                    <span>Job Title</span>
                    <input bind:value={userForm.job_title} placeholder="Role title" />
                </label>
                <label>
                    <span>Role</span>
                    <select bind:value={userForm.role_id}>
                        {#each roles as role}
                            <option value={roleId(role)}>{roleLabel(role)}</option>
                        {/each}
                    </select>
                </label>
                <label class="checkbox-row">
                    <input type="checkbox" bind:checked={userForm.is_active} />
                    <span>Active user</span>
                </label>
            </div>
            <div class="modal-actions">
                <button class="btn-sm" onclick={() => (showUserModal = false)}>Cancel</button>
                <button class="btn-primary" onclick={saveUser} disabled={savingUser}>{savingUser ? "Saving..." : "Save User"}</button>
            </div>
        </div>
    </div>
{/if}

{#if permissionRole}
    <div class="modal-backdrop" role="button" tabindex="0" onclick={() => (permissionRole = null)} onkeydown={(event) => event.key === "Escape" && (permissionRole = null)}>
        <div class="modal-card permissions-card" role="presentation" tabindex="-1" onclick={stopPropagation(bubble('click'))} onkeydown={stopPropagation(bubble('keydown'))}>
            <div class="modal-head">
                <div>
                    <h2>{roleLabel(permissionRole)}</h2>
                    <p>{permissionRole.description || "Role permissions"}</p>
                </div>
                <button class="btn-sm" onclick={() => (permissionRole = null)}>Close</button>
            </div>
            {#if loadingRolePermissions}
                <div class="loading"><WabiSpinner size="sm" /></div>
            {:else if rolePermissions.length === 0}
                <p class="dim">No permissions configured.</p>
            {:else}
                <div class="permission-grid">
                    {#each rolePermissions as permission}
                        <span class="permission-chip">{permission}</span>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
{/if}

<style>
    .page {
        padding: var(--page-padding);
        height: 100vh;
        background: var(--paper);
        color: var(--ink);
        display: flex;
        flex-direction: column;
        box-sizing: border-box;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        margin-bottom: var(--space-6);
    }
    h1 {
        font-size: var(--text-5xl);
        font-weight: var(--font-weight-light);
        margin: 0;
        letter-spacing: -0.02em;
    }
    .subtitle {
        color: var(--ink-faint);
        margin-top: var(--space-2);
    }
    .btn-primary {
        background: var(--ink);
        color: var(--paper);
        border: none;
        padding: 10px 20px;
        border-radius: var(--radius-pill);
        cursor: pointer;
    }

    .tabs {
        display: flex;
        gap: 24px;
        border-bottom: 1px solid var(--border-subtle);
        margin-bottom: 24px;
    }
    .tab {
        background: transparent;
        border: none;
        padding: 12px 0;
        font-size: 14px;
        color: var(--ink-light);
        cursor: pointer;
        position: relative;
    }
    .tab.active {
        color: var(--ink);
        font-weight: 500;
    }
    .tab.active::after {
        content: "";
        position: absolute;
        bottom: -1px;
        left: 0;
        width: 100%;
        height: 2px;
        background: var(--ink);
    }

    .content-area {
        flex: 1;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    .loading {
        display: flex;
        justify-content: center;
        padding: 40px;
    }

    .toolbar {
        display: flex;
        margin-bottom: 16px;
    }
    .search-input {
        width: 300px;
        padding: 10px;
        border: 1px solid var(--border-medium);
        border-radius: 8px;
        font-size: 13px;
    }

    .table-card {
        background: var(--paper-subtle);
        border-radius: 12px;
        border: 1px solid var(--border-subtle);
        overflow: hidden;
    }
    table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;
    }
    th {
        text-align: left;
        padding: 12px 16px;
        border-bottom: 1px solid var(--border-medium);
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink-light);
        font-weight: 500;
    }
    td {
        padding: 12px 16px;
        border-bottom: 1px solid var(--border-subtle);
        vertical-align: middle;
    }
    .right {
        text-align: right;
    }

    .user-cell {
        display: flex;
        align-items: center;
        gap: 12px;
    }
    .avatar {
        width: 32px;
        height: 32px;
        background: var(--ink);
        color: var(--paper);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 600;
        font-size: 14px;
    }
    .info .name {
        font-weight: 500;
    }
    .info .sub {
        font-size: 11px;
        color: var(--ink-light);
    }

    .badge {
        padding: 2px 8px;
        border-radius: 4px;
        border: 1px solid var(--border-medium);
        font-size: 10px;
        text-transform: uppercase;
    }
    .badge.role {
        background: var(--paper);
    }
    .badge.neutral {
        background: rgba(0, 0, 0, 0.05);
        border: none;
    }

    .dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        display: inline-block;
        margin-right: 4px;
    }
    .dot.green {
        background: #22c55e;
    }
    .dot.red {
        background: #ef4444;
    }

    .btn-sm {
        padding: 4px 12px;
        border: 1px solid var(--border-medium);
        background: var(--paper);
        border-radius: 4px;
        cursor: pointer;
        font-size: 11px;
    }

    .grid-roles {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
        gap: 16px;
    }
    .role-card {
        background: var(--paper-subtle);
        padding: 20px;
        border-radius: 12px;
        border: 1px solid var(--border-subtle);
    }
    .role-card h3 {
        margin: 0 0 8px;
        font-size: 16px;
        font-weight: 500;
    }
    .role-card p {
        font-size: 12px;
        color: var(--ink-light);
        margin-bottom: 16px;
    }
    .btn-txt {
        background: transparent;
        border: none;
        color: var(--ink);
        font-size: 12px;
        cursor: pointer;
        padding: 0;
        font-weight: 500;
    }

    .bold {
        font-weight: 500;
    }
    .dim {
        color: var(--ink-light);
    }
    .mono {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--ink-light);
    }

    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(15, 23, 42, 0.42);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 24px;
        z-index: 1000;
    }
    .modal-card {
        width: min(720px, 100%);
        max-height: 88vh;
        overflow-y: auto;
        background: var(--paper);
        border: 1px solid var(--border-medium);
        border-radius: 12px;
        padding: 24px;
        box-shadow: 0 24px 70px rgba(15, 23, 42, 0.24);
    }
    .permissions-card {
        width: min(820px, 100%);
    }
    .modal-head {
        display: flex;
        justify-content: space-between;
        gap: 16px;
        margin-bottom: 20px;
    }
    .modal-head h2 {
        margin: 0 0 4px;
        font-size: 22px;
        font-weight: 500;
    }
    .modal-head p {
        margin: 0;
        color: var(--ink-light);
        font-size: 13px;
    }
    .form-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 14px;
    }
    .form-grid label {
        display: flex;
        flex-direction: column;
        gap: 6px;
        font-size: 12px;
        color: var(--ink-light);
    }
    .form-grid input,
    .form-grid select {
        border: 1px solid var(--border-medium);
        border-radius: 8px;
        padding: 9px 10px;
        background: var(--paper);
        color: var(--ink);
    }
    .checkbox-row {
        justify-content: center;
        flex-direction: row !important;
        align-items: center;
        color: var(--ink) !important;
    }
    .checkbox-row input {
        width: auto;
    }
    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 22px;
    }
    .permission-grid {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }
    .permission-chip {
        font-family: var(--font-mono);
        font-size: 11px;
        padding: 5px 8px;
        border-radius: 6px;
        background: var(--paper-subtle);
        border: 1px solid var(--border-subtle);
    }
</style>
