/**
 * @asymmflow/tokens — the theme contract.
 *
 * A theme is DATA: a named assignment of `--af-*` custom properties.
 * This module defines the contract (which tokens exist, which a theme MUST
 * provide), helpers to apply/serialize themes, and the mathematical constants
 * shared by the motion engine and generative layers.
 *
 * The CSS files under css/ are the load-path default (Onyx & Ether v3);
 * this module is the programmatic seam — the style_alchemy design engine
 * generates `Theme` objects that flow through `applyTheme`/`themeToCss`.
 *
 * Constitution: packages/DESIGN_CONSTITUTION.md
 */

// ===== Mathematical constants (shared substrate) =====

/** The golden ratio — the spacing ladder and stagger rhythms converge to it. */
export const PHI = 1.618033988749895;

/**
 * The φ ladder (§4b): additive Fibonacci-style spacing scale in px.
 * Each step is the sum of the previous two; step ratio → φ.
 */
export const SPACE_LADDER = [4, 8, 12, 20, 32, 52, 84, 136] as const;

/** Modular type scale (≈1.2 minor third), px at --af-scale: 1. */
export const TYPE_SCALE = {
  xs: 11,
  sm: 13,
  md: 14,
  lg: 16,
  xl: 19,
  '2xl': 24,
  '3xl': 29,
  display: 48,
} as const;

/**
 * Three-regime motion policy (§4e). The motion engine and CSS tokens both
 * derive from this single definition.
 */
export const MOTION_REGIMES = {
  /** R1 · Explore — entrances, reveals, arrivals. */
  explore: { duration: 400, ease: [0.22, 1, 0.36, 1] as const },
  /** R2 · Optimize — micro-interactions: hover, press, toggle, focus. */
  optimize: { duration: 140, ease: [0.4, 0, 0.2, 1] as const },
  /** R3 · Stabilize — exits, settles, confirmations, collapses. */
  stabilize: { duration: 240, ease: [0.4, 0, 1, 1] as const },
  /** Spring — earned moments only (ceremonies, success). */
  spring: { duration: 500, ease: [0.34, 1.56, 0.64, 1] as const },
} as const;

export type MotionRegime = keyof typeof MOTION_REGIMES;

/** Sibling stagger rhythm in ms (§4e). */
export const MOTION_STAGGER_MS = 48;

/**
 * Hover-intent delay before a tooltip reveals, in ms. The established tooltip
 * standard (~optimize × 3.57) — long enough to ignore an incidental pass-over,
 * short enough to feel responsive on a deliberate hover. Instant on the way out.
 */
export const TOOLTIP_DELAY_MS = 500;

// ===== The token contract =====

/**
 * Color tokens a theme MUST provide. Everything else (type, space, radii,
 * motion, z, density) has constitutional defaults a theme MAY override.
 */
export const REQUIRED_COLOR_TOKENS = [
  'bg',
  'surface',
  'surface-raised',
  'surface-sunken',
  'text',
  'text-secondary',
  'text-muted',
  'text-inverse',
  'accent',
  'accent-hover',
  'accent-pressed',
  'accent-contrast',
  'accent-tint',
  'accent-tint-strong',
  'inverse-surface',
  'border',
  'border-strong',
  'tint',
  'tint-medium',
  'tint-strong',
  'success',
  'success-tint',
  'warning',
  'warning-tint',
  'danger',
  'danger-tint',
  'info',
  'info-tint',
  'glass-bg',
  'glass-border',
  'focus-ring',
  'scrim',
  'chart-1',
  'chart-2',
  'chart-3',
  'chart-4',
  'chart-5',
  'chart-6',
  'chart-7',
  'chart-8',
] as const;

/** Number of chart series slots every theme provides (--af-chart-1..N). */
export const CHART_SERIES_COUNT = 8;

/** The golden angle in degrees — chart palettes hue-walk by this step. */
export const GOLDEN_ANGLE_DEG = 137.50776405003785;

export type RequiredColorToken = (typeof REQUIRED_COLOR_TOKENS)[number];

/**
 * A theme: name + token values keyed WITHOUT the `--af-` prefix
 * (e.g. `{ accent: '#2F7A38' }` → `--af-accent: #2F7A38`).
 * Required color tokens must be present; any other --af-* token may be
 * overridden (fonts, radii, motion durations…).
 */
export interface Theme {
  name: string;
  tokens: Record<RequiredColorToken, string> & Record<string, string>;
}

/** Returns the required color tokens missing from a theme (empty = valid). */
export function validateTheme(theme: Theme): string[] {
  return REQUIRED_COLOR_TOKENS.filter((t) => !(t in theme.tokens));
}

/** Apply a theme to an element (defaults to <html>) by setting --af-* vars. */
export function applyTheme(
  theme: Theme,
  el: HTMLElement = document.documentElement,
): void {
  el.dataset.afTheme = theme.name;
  for (const [key, value] of Object.entries(theme.tokens)) {
    el.style.setProperty(`--af-${key}`, value);
  }
}

/** Remove a previously applied inline theme, falling back to the CSS default. */
export function clearTheme(el: HTMLElement = document.documentElement): void {
  delete el.dataset.afTheme;
  for (const prop of Array.from(el.style)) {
    if (prop.startsWith('--af-')) el.style.removeProperty(prop);
  }
}

/** Serialize a theme to a CSS rule (for build-time theme files). */
export function themeToCss(
  theme: Theme,
  selector = `[data-af-theme="${theme.name}"]`,
): string {
  const lines = Object.entries(theme.tokens).map(
    ([key, value]) => `  --af-${key}: ${value};`,
  );
  return `${selector} {\n${lines.join('\n')}\n}\n`;
}

// ===== The default theme, as data =====

/**
 * Onyx & Ether v3 — mirrors css/themes/onyx-ether.css (which is canonical for
 * the load path). Exposed as data so the design engine has a reference
 * exemplar to mutate from.
 */
export const onyxEther: Theme = {
  name: 'onyx-ether',
  tokens: {
    bg: '#E9EFE9',
    surface: '#FFFFFF',
    'surface-raised': '#F7FAF7',
    'surface-sunken': '#DEE7DE',
    text: '#18211B',
    'text-secondary': '#4C5B51',
    'text-muted': '#6E7E73',
    'text-inverse': '#F4F8F4',
    accent: '#2A7532',
    'accent-hover': '#236128',
    'accent-pressed': '#1A4C1F',
    'accent-contrast': '#FFFFFF',
    'accent-tint': 'rgba(42, 117, 50, 0.10)',
    'accent-tint-strong': 'rgba(42, 117, 50, 0.18)',
    'inverse-surface': '#18211B',
    border: '#CCD9CD',
    'border-strong': '#A3B7A8',
    tint: 'rgba(24, 33, 27, 0.05)',
    'tint-medium': 'rgba(24, 33, 27, 0.10)',
    'tint-strong': 'rgba(24, 33, 27, 0.15)',
    success: '#2A7532',
    'success-tint': 'rgba(42, 117, 50, 0.12)',
    warning: '#8F6210',
    'warning-tint': 'rgba(143, 98, 16, 0.12)',
    danger: '#AC2F25',
    'danger-tint': 'rgba(172, 47, 37, 0.12)',
    info: '#2861A6',
    'info-tint': 'rgba(40, 97, 166, 0.12)',
    'glass-bg': 'rgba(247, 250, 247, 0.88)',
    'glass-border': 'rgba(24, 33, 27, 0.10)',
    'focus-ring': '#2A7532',
    scrim: 'rgba(24, 33, 27, 0.48)',
    'chart-1': '#2A7532',
    'chart-2': '#5D55A6',
    'chart-3': '#A8741A',
    'chart-4': '#2D7670',
    'chart-5': '#94427C',
    'chart-6': '#5E7A34',
    'chart-7': '#4359B1',
    'chart-8': '#B5503F',
  },
};
