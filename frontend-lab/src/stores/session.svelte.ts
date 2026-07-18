/* Session store — the acting user + permissions for the kernel app. NEW at K5
 * (the lab had no session; BankRecon/DeploymentHub/PeopleHub used an `actor:
 * 'lab-user'` placeholder and WorkHub had no role signal — all flagged in their
 * parity docs for "K5 session"). Rune-based. `hasPermission` unifies the
 * wildcard/`resource:*` logic that was duplicated 3× in the old frontend
 * (App.svelte, EnterpriseSidebar, authContext) into ONE definition (L2).
 *
 * Populated at the shell's boot from whichever auth path wins (K5 = license
 * result). Under mock (no Wails runtime), `initMockSession()` seeds a synthetic
 * admin with full permissions so every screen + nav item renders in the lab. */

import { usingWails } from '../bridge/runtime'

export interface SessionUser {
  id: string
  fullName: string
  roleName: string
}

let currentUser = $state<SessionUser | null>(null)
let permissions = $state<string[]>([])

export function getCurrentUser(): SessionUser | null {
  return currentUser
}

/** The acting-user id for audit/mutation calls (BankRecon finalize, etc.). */
export function actingUserId(): string {
  return currentUser?.id ?? 'unknown'
}

export function getPermissions(): string[] {
  return permissions
}

export function isAuthenticated(): boolean {
  return currentUser !== null
}

/** Wildcard-aware permission check — ONE definition (L2): grants on `*`, a
 * direct match, or a `resource:*` wildcard for the permission's resource. */
export function hasPermission(perm: string): boolean {
  if (!perm) return true
  if (permissions.includes('*')) return true
  if (permissions.includes(perm)) return true
  const resource = perm.split(':')[0]
  return !!resource && permissions.includes(`${resource}:*`)
}

/** Set from a real auth result (license activation payload at K5). */
export function setSession(user: SessionUser, perms: string[]): void {
  currentUser = user
  permissions = [...perms]
}

export function clearSession(): void {
  currentUser = null
  permissions = []
}

/** Seed a synthetic admin under mock so the lab renders every screen/nav item.
 * Real Wails boots via the license gate → setSession() instead. */
export function initSession(): void {
  if (usingWails()) return // real boot sets the session from the license result
  currentUser = { id: 'lab-user', fullName: 'Lab User', roleName: 'Administrator' }
  permissions = ['*']
}
