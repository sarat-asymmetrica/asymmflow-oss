/**
 * Brand slot — the ONE source of app identity.
 *
 * Consumed by: EnterpriseSidebar.svelte (sidebar header), LoginScreen.svelte
 * (login/lock surface). PDF/print headers are a separate, already
 * config-driven path — see `docs/DEPLOYMENT_BRANDING.md`.
 *
 * Branding is configuration, not code (repo law). The shipped default is
 * the synthetic "AsymmFlow" identity. A deployment re-skins by creating a
 * GITIGNORED `frontend/src/lib/brand.local.ts` that default-exports a
 * partial `BrandIdentity` — no source edits required. See
 * `docs/DEPLOYMENT_BRANDING.md` for the full recipe.
 *
 * `accentVar` is any valid CSS color value — typically a `var(--token)`
 * reference into the design-token layer, but a deployment override may use
 * a literal color instead. It must never be hardcoded to a non-synthetic
 * value in this file.
 */

export interface BrandIdentity {
  /** Full wordmark shown next to the mark, e.g. in the sidebar header and login card. */
  wordmark: string;
  /** Short glyph shown inside the badge/logo-mark (2-3 chars reads best). */
  mark: string;
  /** CSS color value for the accent — defaults to the existing brand token, never a hardcoded hex here. */
  accentVar: string;
  /**
   * Default division / trading-entity display name (Wave 11 B1). The ONE
   * frontend source for the division shown as a fallback, a new-record default,
   * and in generated-document display paths. The shipped value is the synthetic
   * "Acme Instrumentation"; a deployment overrides it via `brand.local.ts` so
   * Finance / QuickCapture / PDF defaults re-skin with NO source edit.
   *
   * NOTE (Wave 11 scope boundary): the app also uses the literal division name
   * as a COMPARISON KEY in ~20 scoping/validation/SQL sites (see
   * FABLE_WAVE11_SPEC_REPORT.md §B1). Those are NOT re-pointed here — collapsing
   * them needs a stable division-ID registry (a future wave). This slot only
   * feeds DISPLAY and DEFAULT-FOR-NEW; because the fallbacks and the comparison
   * defaults draw from the SAME value, behavior stays consistent under override.
   */
  defaultDivision: string;
}

const defaultBrand: BrandIdentity = {
  wordmark: 'AsymmFlow',
  mark: 'AF',
  accentVar: 'var(--brand-indigo)',
  defaultDivision: 'Acme Instrumentation'
};

// Optional deployment-side override. `brand.local.ts` is gitignored and does
// not exist in the shipped tree — import.meta.glob resolves to an empty
// object when the file is absent, so this is a no-op by default and only
// activates when a deployment adds the override file locally.
const localModules = import.meta.glob('./brand.local.ts', { eager: true }) as Record<
  string,
  { default?: Partial<BrandIdentity> }
>;
const localOverride = Object.values(localModules)[0]?.default ?? {};

export const brand: BrandIdentity = { ...defaultBrand, ...localOverride };
