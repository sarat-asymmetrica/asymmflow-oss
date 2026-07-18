# Parity Ledger — UserManagementScreen (old) vs Users EntityMaster descriptor

Verdicts:

- **DONE** — capability exists in the kernel pilot today
- **EQUIV** — deliberately different mechanism, same job, kernel way is better
- **ENGINE** — needs a kernel/engine feature (benefits ALL entities at once)
- **SLOT** — needs an ejection component (screen-specific, L4 territory)
- **INTEG** — needs the real Wails bindings (mock stands in today)
- **DEFER** — deliberately out of the pilot's scope, tracked

| # | Old-screen capability | Verdict | Notes |
|---|---|---|---|
| 1 | Status derivation (`User` has no `status` field, only `is_active`) | **FIXED, not preserved** | Same correctness finding as Suppliers (recon-K2 synthesis #5). Descriptor derives a 2-state `Active`/`Inactive` from `is_active` — no fabricated 3rd state. |
| 2 | + Add User (`CreateUser`, `users:create` gated, sets initial `role_id`) | **NOT BUILT — RBAC hot-zone** | Per the K2 brief: user-mutation actions are explicitly out of scope. `role_id` is a privilege-bearing field with no client-side guard beyond server RBAC; wiring belongs at K5 through the exact server-gated call, not a hand-rolled form here. No `create`/`update` mutation exists anywhere in `bridge/users.ts` — not even a mock one — so there is nothing to accidentally leak an optimistic-mutation pattern from. |
| 3 | Edit (`UpdateUser`, `users:update` gated, sets `role_id`/`is_active`/etc.) | **NOT BUILT — RBAC hot-zone** | Same reasoning as #2. `is_active` (the deactivation path — there is no dedicated delete/deactivate binding) is exactly as privilege-bearing as `role_id`; both are gated behind the same `UpdateUser` call and both are deliberately not built. |
| 4 | Password field: required+editable on create, hidden/disabled on edit | **N/A — not built** | Moot until #2/#3 land. `password`/`password_hash` never appears on `UserRow`, the mock generator, or anywhere in this descriptor — verified by grep of `bridge/users.ts`. |
| 5 | Mock-mode local-mutation fallback (`if (!window.go)` in the old screen) | **NOT CARRIED FORWARD** | Flagged by recon-K2 as an anti-pattern that must not leak into the kernel descriptor. This bridge has no mutation exports at all (mock or real), so there is no local-mutation surface to begin with. |
| 6 | Three-tab single screen (Users / Roles / Audit Logs) | **ENGINE gap** | Roles and Audit Logs are separate small datasets, not per-user profile content — same "screen composes multiple archetype instances" finding as K1's Payments/ChequeRegister/Expenses (recon-K2 #1). Out of K2 scope; Roles/Audit are their own future descriptors. |
| 7 | Role permission-chip viewer (`GetRolePermissionsList`, off the Role entity) | **SLOT** | Small, self-contained, scoped to a different entity than `users`. Not built — low priority per recon-K2. |
| 8 | No dedicated user detail/profile view historically | **EQUIV (additive, not preserved)** | The `EntityMaster` profile (Account/Access sections + Last Login KPI) is NEW capability, not a migrated one — there is no "old behavior" to parity-check it against. Judged directly against the `User` field census instead. |
| 9 | Search sweeps name/username only | **FIXED, not preserved** | `searchText` widened to `fullName`/`username`/`email`/`department`/`roleName` — trivial, every other K1/K2 screen already searches this broadly (recon-K2 #6). |
| 10 | No status/role filter chips | **FIXED, not preserved** | Both added as derived `FilterSpec`s — free given `status` and `roleName` both exist on the row; the old screen simply never built them. |
| 11 | No Delete/Deactivate action in the UI at all | **CORRECTLY NOT BUILT** | Matches old-screen behavior exactly — deactivation only ever happened via Edit→uncheck Active (itself not built, see #3). No delete action invented here. |

## Reading

This is the thinnest K2 build by design: the RBAC hot-zone ruling in the brief
means every mutating capability (create, edit, role assignment, deactivation)
is deliberately absent, not deferred-and-forgotten — there is no mock
mutation, no `FormSpec`, and no `ActionSpec` anywhere in `users.descriptor.ts`
or `bridge/users.ts` for any of it. What IS built matches or exceeds the old
screen: list, 2-state status (fixed from a fake 3-state), a role/status
summary strip with genuine visual-diversity value (a role-distribution bar is
a real RBAC-hygiene signal, and Users is the only entity in this batch with
zero money fields), widened search, and two new filters. The profile is
honestly additive — the old screen never had one — built from the `User`
field census, not from a richer old view that doesn't exist. Password/
password-hash exposure and optimistic local mutation were the two named
security invariants; both are structurally impossible here since the bridge
carries no such field and no mutation path at all.
