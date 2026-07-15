/* Users bridge module — self-contained: types + mock + real + switch.
 * Real binding: `ListUsers()` on App (verified against
 * frontend/wailsjs/go/main/App.d.ts:1361 — also present on InfraService,
 * App is the K1/K2 convention for App-hosted list bindings). UNPAGED — the
 * server loads every non-deleted user in one call (recon-K2: no limit/offset
 * param exists on this binding at all).
 *
 * SECURITY / RBAC HOT-ZONE (recon-K2, binding): `User` has no `password`
 * field — it never round-trips (`PasswordHash` is `json:"-"` server-side).
 * This module NEVER carries a password/password_hash field on the row, in
 * the mock generator, or anywhere else. It exposes fetch ONLY — no create/
 * update/deactivate mutation is built here (K2 scope: read + summary +
 * profile only). `role_id`/`is_active` are privilege-bearing fields; real
 * mutations wire at K5 through the exact server-gated calls
 * (`CreateUser`/`UpdateUser`, both `users:*`-RBAC-gated) — never an
 * optimistic local mutation, so none is offered here to accidentally lean on. */
import { pick } from './runtime'
import { goDate, str } from './map'
import { ListUsers } from '$wails/go/main/App'

export interface UserRow {
  id: string
  username: string
  email: string
  fullName: string
  displayName: string
  department: string
  jobTitle: string
  isActive: boolean
  status: string // derived: 'Active' | 'Inactive' — User has no status field
  lastLoginAt: string
  mustChangePassword: boolean
  roleName: string
}

/* ---- mock: adversarial + deterministic (see bridge/mock.ts) ---- */
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))
function lcg(seed: number): () => number {
  let s = seed >>> 0
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0
    return s / 0xffffffff
  }
}
const pad = (n: number, w: number): string => String(n).padStart(w, '0')

const NAMES = [
  'Ahmed Al-Khalifa',
  'Fatima Hassan',
  'Mohammed Al-Sayed',
  'Layla Ibrahim',
  'Yousif Abdulla',
  'Noor Al-Amin',
  'Khalid Mansoor',
  'Sara Al-Qassimi',
  'Hassan Juma',
  'Maryam Al-Doseri',
  'محمد أحمد الخليفة', // RTL adversary
  'X',
  'Extremely Long Legal Full Name As It Appears On The National Identification Card Including Tribal Affiliation And Honorific Titles For Records Purposes Only Not For Everyday Display'.slice(0, 200), // 200-char monster
]
const DEPARTMENTS = ['Sales', 'Finance', 'Warehouse', 'Procurement', 'Operations', 'IT', '']
const JOB_TITLES = [
  'Sales Executive',
  'Finance Manager',
  'Warehouse Clerk',
  'Procurement Lead',
  'Operations Coordinator',
  'IT Administrator',
  '',
]
const ROLES = [
  'Administrator',
  'Sales Manager',
  'Sales Representative',
  'Finance Officer',
  'Warehouse Supervisor',
  'Procurement Officer',
  'Viewer',
]

let cache: UserRow[] | null = null

function generate(): UserRow[] {
  const rand = lcg(20260714 ^ 0x05e5)
  const rows: UserRow[] = []
  const n = 60
  for (let i = 1; i <= n; i++) {
    const r = rand()
    const isActive = i % 7 !== 0 // ~86% active, matches the batch's "mostly active" convention
    const fullName = i % 59 === 0 ? NAMES[12]! : i % 37 === 0 ? NAMES[10]! : NAMES[i % 10]!
    const deptIdx = i % 43 === 0 ? DEPARTMENTS.length - 1 : i % DEPARTMENTS.length
    const jobIdx = i % 43 === 0 ? JOB_TITLES.length - 1 : i % JOB_TITLES.length
    const monthIdx = Math.floor(rand() * 12)
    const day = 1 + Math.floor(rand() * 27)
    const neverLoggedIn = i % 23 === 0

    rows.push({
      id: `user-${i}`,
      username: i % 13 === 0 ? 'x' : `user.${pad(i, 3)}`,
      email:
        i % 17 === 0
          ? 'user.identity.and.access.management.department.regional.office@extremely-long-corporate-domain-name.example.com.bh'
          : `user.${pad(i, 3)}@example.bh`,
      fullName,
      displayName: fullName,
      department: DEPARTMENTS[deptIdx]!,
      jobTitle: JOB_TITLES[jobIdx]!,
      isActive,
      status: isActive ? 'Active' : 'Inactive',
      lastLoginAt: neverLoggedIn ? '' : `2026-${pad(monthIdx + 1, 2)}-${pad(day, 2)}`,
      mustChangePassword: i % 11 === 0,
      roleName: i % 31 === 0 ? '' : ROLES[i % ROLES.length]!, // empty = role-hydration-failure adversary
    })
  }
  return rows
}

async function mockFetchAll(): Promise<UserRow[]> {
  cache ??= generate()
  await sleep(250)
  return [...cache]
}

/* ---- real: fetch WIRED (no mutations — K2 scope is read-only, see header) ---- */
function mapUser(r: Record<string, unknown>): UserRow {
  const isActive = Boolean(r.is_active)
  return {
    id: str(r.id),
    username: str(r.username),
    email: str(r.email),
    fullName: str(r.full_name),
    displayName: str(r.display_name),
    department: str(r.department),
    jobTitle: str(r.job_title),
    isActive,
    status: isActive ? 'Active' : 'Inactive',
    lastLoginAt: goDate(r.last_login_at),
    mustChangePassword: Boolean(r.must_change_password),
    roleName: str(r.role_name),
  }
}

async function realFetchAll(): Promise<UserRow[]> {
  const rows = await ListUsers()
  return (rows ?? []).map((x) => mapUser(x as unknown as Record<string, unknown>))
}

/* ---- public switched API (descriptor imports THIS) ---- */
export const fetchUsers = (): Promise<UserRow[]> => pick(realFetchAll, mockFetchAll)()
