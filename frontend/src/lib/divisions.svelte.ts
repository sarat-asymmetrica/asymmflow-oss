import { GetDivisionRegistry } from "../../wailsjs/go/main/InfraService";

export interface DivisionOption {
  key: string;
  legalName: string;
}

interface DivisionRegistryState {
  divisions: Array<{ key: string; legalName: string; aliases: string[]; dashboardVariant: string }>;
  defaultKey: string;
  companyDisplayName: string;
}

// BUILTIN synthetic fallback — the frontend mirror of overlay.BuiltinDefaults();
// the ONLY permitted division literals in frontend live code (audit-exempt,
// like Go's BuiltinDefaults). This guarantees the division selector is never
// empty: it seeds the store before the Wails binding resolves (or if it
// never does, e.g. DESIGN_MODE / a failed call).
export const BUILTIN_DIVISION_REGISTRY: DivisionRegistryState = {
  divisions: [
    { key: "Acme Instrumentation", legalName: "ACME INSTRUMENTATION W.L.L", aliases: [], dashboardVariant: "" },
    {
      key: "Beacon Controls",
      legalName: "BEACON CONTROLS W.L.L.",
      aliases: ["beacon controls wll", "beacon controls w.l.l", "beacon controls w.l.l."],
      dashboardVariant: "ahs",
    },
  ],
  defaultKey: "Acme Instrumentation",
  companyDisplayName: "Acme Instrumentation WLL",
};

let registry = $state<DivisionRegistryState>(BUILTIN_DIVISION_REGISTRY);

export function getDivisions(): DivisionOption[] {
  return registry.divisions.map((div) => ({ key: div.key, legalName: div.legalName }));
}

export function getDivisionKeys(): string[] {
  return registry.divisions.map((div) => div.key);
}

export function getDefaultDivisionKey(): string {
  return registry.defaultKey;
}

export function getCompanyDisplayName(): string {
  return registry.companyDisplayName;
}

export function getDivisionLegalName(key: string): string {
  const match = registry.divisions.find((div) => div.key === key);
  return match ? match.legalName : "";
}

export function isKnownDivision(value: string): boolean {
  return registry.divisions.some((div) => div.key === value);
}

// Returns the division's dashboard-variant key (e.g. "ahs" for a division
// with a bespoke dashboard), or "" if none. Mirrors overlay DashboardVariant.
export function getDashboardVariant(key: string): string {
  const match = registry.divisions.find((div) => div.key === key);
  return match ? match.dashboardVariant : "";
}

// Mirrors overlay.CompanyOverlay.NormalizeDivisionName exactly (Go source:
// pkg/overlay/overlay.go): case-insensitive/whitespace-trimmed match against
// each division's Key, then against its declared (already-lowercase) aliases;
// unknown strings fall back to the registry's default key.
export function normalizeDivision(raw: string): string {
  const needle = raw.trim().toLowerCase();
  for (const div of registry.divisions) {
    if (div.key.toLowerCase() === needle) {
      return div.key;
    }
    for (const alias of div.aliases) {
      if (alias === needle) {
        return div.key;
      }
    }
  }
  return registry.defaultKey;
}

export async function initDivisions(): Promise<void> {
  try {
    const response = await GetDivisionRegistry();
    if (response && Array.isArray(response.divisions) && response.divisions.length > 0) {
      registry = {
        divisions: response.divisions.map((div) => ({
          key: div.key,
          legalName: div.legalName,
          aliases: div.aliases || [],
          dashboardVariant: div.dashboardVariant || "",
        })),
        defaultKey: response.defaultKey,
        companyDisplayName: response.companyDisplayName,
      };
    }
  } catch (error) {
    // Keep the BUILTIN synthetic fallback — never leave the selector empty.
    console.error("initDivisions: failed to load division registry, keeping builtin fallback", error);
  }
}
