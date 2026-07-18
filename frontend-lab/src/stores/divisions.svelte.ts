/* Divisions store — the ONE source of division vocabulary for the kernel app
 * (L7: division names come from the registry, never a hardcoded literal). Ported
 * from the old frontend's `divisions.svelte.ts` (Svelte 5 runes). Seeds the
 * BUILTIN synthetic fallback so the selector is never empty, then `initDivisions()`
 * replaces it from `GetDivisionRegistry` when the real Wails runtime is present.
 * Under mock (no window.go) it silently keeps the fallback. Replaces the
 * static-mock `divisionOptions` several bridges stub until this lands. */

import { GetDivisionRegistry } from '$wails/go/main/InfraService'
import { usingWails } from '../bridge/runtime'

export interface DivisionOption {
  key: string
  legalName: string
}

interface DivisionRegistryState {
  divisions: Array<{ key: string; legalName: string; aliases: string[]; dashboardVariant: string }>
  defaultKey: string
  companyDisplayName: string
}

// BUILTIN synthetic fallback — the frontend mirror of overlay.BuiltinDefaults();
// the ONLY permitted division literals in frontend live code (audit-exempt,
// like Go's BuiltinDefaults). Synthetic identity (SYNTHETIC_IDENTITY.md).
export const BUILTIN_DIVISION_REGISTRY: DivisionRegistryState = {
  divisions: [
    { key: 'Acme Instrumentation', legalName: 'ACME INSTRUMENTATION W.L.L', aliases: [], dashboardVariant: '' },
    {
      key: 'Beacon Controls',
      legalName: 'BEACON CONTROLS W.L.L.',
      aliases: ['beacon controls wll', 'beacon controls w.l.l', 'beacon controls w.l.l.'],
      dashboardVariant: 'ahs',
    },
  ],
  defaultKey: 'Acme Instrumentation',
  companyDisplayName: 'Acme Instrumentation WLL',
}

let registry = $state<DivisionRegistryState>(BUILTIN_DIVISION_REGISTRY)

export function getDivisions(): DivisionOption[] {
  return registry.divisions.map((div) => ({ key: div.key, legalName: div.legalName }))
}

/** Division vocabulary as `{value,label}` form-select options — the ONE source
 * for every division dropdown (L2/L7). Under mock this is the BUILTIN synthetic
 * fallback; under real Wails it is the post-`initDivisions()` registry, so
 * callers MUST read it lazily (at form-open), not capture it at module-eval —
 * see invoices.descriptor.ts. Label is the division key (its short name), which
 * is what the old dropdowns showed; `legalName` is the entity's long legal name. */
export function getDivisionOptions(): { value: string; label: string }[] {
  return registry.divisions.map((div) => ({ value: div.key, label: div.key }))
}

export function getDivisionKeys(): string[] {
  return registry.divisions.map((div) => div.key)
}

export function getDefaultDivisionKey(): string {
  return registry.defaultKey
}

export function getCompanyDisplayName(): string {
  return registry.companyDisplayName
}

export function getDivisionLegalName(key: string): string {
  const match = registry.divisions.find((div) => div.key === key)
  return match ? match.legalName : ''
}

export function isKnownDivision(value: string): boolean {
  return registry.divisions.some((div) => div.key === value)
}

/** Division's dashboard-variant key (e.g. "ahs"), or "" — mirrors overlay DashboardVariant. */
export function getDashboardVariant(key: string): string {
  const match = registry.divisions.find((div) => div.key === key)
  return match ? match.dashboardVariant : ''
}

/** Mirrors overlay NormalizeDivisionName: case/whitespace-insensitive match on key
 * then aliases; unknown falls back to the registry default. */
export function normalizeDivision(raw: string): string {
  const needle = raw.trim().toLowerCase()
  for (const div of registry.divisions) {
    if (div.key.toLowerCase() === needle) return div.key
    for (const alias of div.aliases) {
      if (alias === needle) return div.key
    }
  }
  return registry.defaultKey
}

export async function initDivisions(): Promise<void> {
  if (!usingWails()) return // mock: keep the builtin synthetic fallback
  try {
    const response = await GetDivisionRegistry()
    if (response && Array.isArray(response.divisions) && response.divisions.length > 0) {
      registry = {
        divisions: response.divisions.map((div) => ({
          key: div.key,
          legalName: div.legalName,
          aliases: div.aliases || [],
          dashboardVariant: div.dashboardVariant || '',
        })),
        defaultKey: response.defaultKey,
        companyDisplayName: response.companyDisplayName,
      }
    }
  } catch (error) {
    // Keep the BUILTIN synthetic fallback — never leave the selector empty.
    console.error('initDivisions: failed to load division registry, keeping builtin fallback', error)
  }
}
