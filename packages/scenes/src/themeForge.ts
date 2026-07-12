/**
 * themeForge.ts — THE design engine seam.
 *
 * generateTheme(seed, opts?) → Theme
 *
 * Algorithm:
 *   1. Hash seed → hue in [0, 360°)
 *   2. Derive accent color at hue, tuned for AA contrast on white
 *   3. Build a tinted neutral ramp (hue-influenced, never pure gray — §4c)
 *   4. Derive hover/pressed darkening steps from accent
 *   5. Compute tints (alpha variants)
 *   6. Status colors: harmonized to seed hue (family stable, slight tint)
 *   7. Glass, scrim, focus derived from above
 *   8. Light mode first-class; dark mode inverts ramp properly
 *
 * Guarantees: output passes validateTheme() — all REQUIRED_COLOR_TOKENS present.
 *
 * Constitution: §4c ("tinted neutral ramp — never pure gray"), §2.3 ("themes are data").
 */

import { seededRng } from './rng.js';
import { cyrb53 } from './rng.js';
import type { Theme } from '@asymmflow/tokens';
import {
  validateTheme,
  CHART_SERIES_COUNT,
  GOLDEN_ANGLE_DEG,
} from '@asymmflow/tokens';

// ── Color math helpers ───────────────────────────────────────────────────────

/** Clamp a value to [lo, hi]. */
function clamp(v: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, v));
}

/** HSL → RGB (each in [0,1]). */
function hslToRgb(h: number, s: number, l: number): [number, number, number] {
  h = ((h % 360) + 360) % 360;
  s = clamp(s, 0, 1);
  l = clamp(l, 0, 1);

  const c = (1 - Math.abs(2 * l - 1)) * s;
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1));
  const m = l - c / 2;

  let r = 0, g = 0, b = 0;
  if (h < 60) { r = c; g = x; }
  else if (h < 120) { r = x; g = c; }
  else if (h < 180) { g = c; b = x; }
  else if (h < 240) { g = x; b = c; }
  else if (h < 300) { r = x; b = c; }
  else { r = c; b = x; }

  return [r + m, g + m, b + m];
}

/** RGB (each in [0,1]) → hex string (#RRGGBB). */
function rgbToHex(r: number, g: number, b: number): string {
  const toHex = (v: number) => Math.round(clamp(v, 0, 1) * 255).toString(16).padStart(2, '0');
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

/** RGB (each in [0,1]) → HSL. */
function rgbToHsl(r: number, g: number, b: number): [number, number, number] {
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const l = (max + min) / 2;
  if (max === min) return [0, 0, l];
  const d = max - min;
  const s = d / (l > 0.5 ? 2 - max - min : max + min);
  let h: number;
  switch (max) {
    case r: h = ((g - b) / d + (g < b ? 6 : 0)) / 6; break;
    case g: h = ((b - r) / d + 2) / 6; break;
    default: h = ((r - g) / d + 4) / 6; break;
  }
  return [h * 360, s, l];
}

/** Relative luminance per WCAG 2.x (input: RGB in [0,1]). */
function relativeLuminance(r: number, g: number, b: number): number {
  const lin = (v: number) =>
    v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

/**
 * WCAG contrast ratio between two foreground/background hex colors.
 * Returns the ratio (e.g. 4.5 = AA, 7.0 = AAA).
 */
export function contrastRatio(fg: string, bg: string): number {
  const [fr, fg_, fb] = hexToRgb(fg);
  const [br, bg_, bb] = hexToRgb(bg);
  const l1 = relativeLuminance(fr, fg_, fb);
  const l2 = relativeLuminance(br, bg_, bb);
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

/** Parse a hex string (#RGB or #RRGGBB) → [r, g, b] in [0,1]. */
function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '');
  if (h.length === 3) {
    return [
      parseInt(h[0] + h[0], 16) / 255,
      parseInt(h[1] + h[1], 16) / 255,
      parseInt(h[2] + h[2], 16) / 255,
    ];
  }
  return [
    parseInt(h.slice(0, 2), 16) / 255,
    parseInt(h.slice(2, 4), 16) / 255,
    parseInt(h.slice(4, 6), 16) / 255,
  ];
}

/** Lighten or darken by adjusting HSL lightness. dl: positive = lighter, negative = darker. */
function adjustL(hex: string, dl: number): string {
  const [r, g, b] = hexToRgb(hex);
  const [h, s, l] = rgbToHsl(r, g, b);
  const [nr, ng, nb] = hslToRgb(h, s, clamp(l + dl, 0, 1));
  return rgbToHex(nr, ng, nb);
}

/** Adjust saturation of a hex color by ds (positive = more saturated). */
function adjustS(hex: string, ds: number): string {
  const [r, g, b] = hexToRgb(hex);
  const [h, s, l] = rgbToHsl(r, g, b);
  const [nr, ng, nb] = hslToRgb(h, clamp(s + ds, 0, 1), l);
  return rgbToHex(nr, ng, nb);
}

/** Create an rgba() string from a hex color and alpha. */
function hexAlpha(hex: string, alpha: number): string {
  const [r, g, b] = hexToRgb(hex);
  const ri = Math.round(r * 255);
  const gi = Math.round(g * 255);
  const bi = Math.round(b * 255);
  return `rgba(${ri}, ${gi}, ${bi}, ${alpha.toFixed(2)})`;
}

// ── Accent derivation: find a lightness that gives AA contrast on white ─────

/**
 * Given a hue and saturation, find the darkest lightness such that the color
 * achieves >= minContrast (default 4.5 = WCAG AA) on a white background.
 * Starts from a medium lightness and walks downward until contrast is met.
 */
function findAALightness(
  hue: number,
  sat: number,
  minContrast = 4.5,
): number {
  const white: [number, number, number] = [1, 1, 1];
  const whiteLum = relativeLuminance(...white);

  // Binary search in [0.1, 0.55] for the lightness that hits minContrast on white
  let lo = 0.10;
  let hi = 0.55;
  for (let i = 0; i < 20; i++) {
    const mid = (lo + hi) / 2;
    const [r, g, b] = hslToRgb(hue, sat, mid);
    const lum = relativeLuminance(r, g, b);
    const ratio = (whiteLum + 0.05) / (lum + 0.05);
    if (ratio >= minContrast) {
      lo = mid; // can go lighter and still pass
    } else {
      hi = mid; // too light, need to go darker
    }
  }
  // Use lo (the lightest that still passes)
  return lo;
}

// ── Neutral ramp: hue-tinted, never pure gray (§4c) ─────────────────────────

/**
 * Derive the full neutral ramp from the accent hue.
 * Light mode: very pale tinted backgrounds → near-black text.
 * The ramp carries a subtle hue so neutrals are warm/cool relative to the accent.
 */
function deriveNeutralRamp(
  hue: number,
  mode: 'light' | 'dark',
): {
  bg: string;
  surface: string;
  surfaceRaised: string;
  surfaceSunken: string;
  text: string;
  textSecondary: string;
  textMuted: string;
  textInverse: string;
  inverseSurface: string;
  border: string;
  borderStrong: string;
} {
  // Neutral saturation: low but present — constitutional "tinted, never pure gray"
  const nSat = 0.06;

  if (mode === 'light') {
    // Muted text gets a SEARCHED lightness, not a fixed one: the lightest
    // value that still clears 4.5:1 on white — the AA floor, guaranteed
    // for every hue (v3.1 calibration; fixed l=0.60 failed AA on most hues).
    const mutedL = findAALightness(hue, 0.07, 4.5);
    return {
      bg: rgbToHex(...hslToRgb(hue, nSat, 0.92)),
      surface: rgbToHex(...hslToRgb(hue, nSat * 0.3, 0.99)),
      surfaceRaised: rgbToHex(...hslToRgb(hue, nSat * 0.5, 0.97)),
      surfaceSunken: rgbToHex(...hslToRgb(hue, nSat, 0.88)),
      text: rgbToHex(...hslToRgb(hue, 0.20, 0.11)),
      textSecondary: rgbToHex(...hslToRgb(hue, 0.10, 0.33)),
      textMuted: rgbToHex(...hslToRgb(hue, 0.07, mutedL)),
      textInverse: rgbToHex(...hslToRgb(hue, nSat * 0.5, 0.96)),
      inverseSurface: rgbToHex(...hslToRgb(hue, 0.20, 0.11)),
      border: rgbToHex(...hslToRgb(hue, nSat, 0.82)),
      borderStrong: rgbToHex(...hslToRgb(hue, nSat * 1.5, 0.68)),
    };
  } else {
    // Dark mode: invert the ramp — dark surfaces, light text, same hue family
    return {
      bg: rgbToHex(...hslToRgb(hue, 0.12, 0.09)),
      surface: rgbToHex(...hslToRgb(hue, 0.10, 0.12)),
      surfaceRaised: rgbToHex(...hslToRgb(hue, 0.09, 0.15)),
      surfaceSunken: rgbToHex(...hslToRgb(hue, 0.14, 0.07)),
      text: rgbToHex(...hslToRgb(hue, nSat * 0.5, 0.94)),
      textSecondary: rgbToHex(...hslToRgb(hue, 0.08, 0.68)),
      textMuted: rgbToHex(...hslToRgb(hue, 0.05, 0.57)),
      textInverse: rgbToHex(...hslToRgb(hue, 0.20, 0.12)),
      inverseSurface: rgbToHex(...hslToRgb(hue, nSat * 0.5, 0.94)),
      border: rgbToHex(...hslToRgb(hue, 0.12, 0.22)),
      borderStrong: rgbToHex(...hslToRgb(hue, 0.14, 0.33)),
    };
  }
}

// ── Status colors: family stable, tinted slightly toward seed hue ────────────

/**
 * Status hues: these stay in their chromatic families (green=success, etc.)
 * but are shifted slightly toward the seed hue for harmony. The shift is
 * intentionally small: status must remain recognizable (§4c).
 */
function deriveStatus(
  seedHue: number,
  mode: 'light' | 'dark',
): {
  success: string; successTint: string;
  warning: string; warningTint: string;
  danger: string; dangerTint: string;
  info: string; infoTint: string;
} {
  const tintFactor = 0.08; // how much to shift toward seed hue
  const blend = (base: number, tFactor: number) =>
    base + (seedHue - base) * tFactor;

  const successHue = blend(130, tintFactor); // green family
  const warningHue = blend(38, tintFactor);  // amber family
  const dangerHue = blend(4, tintFactor);    // red family
  const infoHue = blend(215, tintFactor);    // blue family

  const lLight = 0.36; // lightness for light mode status (AA on white bg)
  const lDark = 0.65;  // lightness for dark mode (AA on dark bg)
  const sat = 0.60;

  const lForMode = mode === 'light' ? lLight : lDark;

  const s = (hue: number) => rgbToHex(...hslToRgb(hue, sat, lForMode));
  const t = (hue: number, alpha: number) =>
    hexAlpha(rgbToHex(...hslToRgb(hue, sat, mode === 'light' ? 0.40 : 0.60)), alpha);

  return {
    success: s(successHue),
    successTint: t(successHue, 0.10),
    warning: s(warningHue),
    warningTint: t(warningHue, 0.10),
    danger: s(dangerHue),
    dangerTint: t(dangerHue, 0.10),
    info: s(infoHue),
    infoTint: t(infoHue, 0.10),
  };
}

// ── Chart palette: golden-angle hue walk from the accent ────────────────────

/**
 * Derive the chart series palette (§4c + charts grammar): series 1 is the
 * accent itself; each subsequent series rotates the hue by the golden angle
 * (137.5°), the same irrational step the Spinner arc and quaternion walks use.
 * Saturation breathes slightly per index so neighbors never read as a ramp.
 * Every color is contrast-floored: ≥3:1 against the surface it draws on
 * (WCAG 1.4.11 non-text contrast).
 */
function deriveChartPalette(accentHex: string, mode: 'light' | 'dark'): string[] {
  const [r, g, b] = hexToRgb(accentHex);
  const [accentHue] = rgbToHsl(r, g, b);

  const colors: string[] = [accentHex];
  for (let i = 1; i < CHART_SERIES_COUNT; i++) {
    const hue = (accentHue + GOLDEN_ANGLE_DEG * i) % 360;
    const sat = 0.42 + (i % 3) * 0.09; // breathe: 0.42 / 0.51 / 0.60
    if (mode === 'light') {
      // Lightest value that still clears 3:1 on (near-)white surface
      const l = findAALightness(hue, sat, 3.0);
      colors.push(rgbToHex(...hslToRgb(hue, sat, l)));
    } else {
      // Dark mode: lift toward the readable plateau over dark surfaces
      const l = clamp(findAALightness(hue, sat, 3.0) + 0.32, 0.55, 0.74);
      colors.push(rgbToHex(...hslToRgb(hue, sat, l)));
    }
  }
  return colors;
}

// ── Public API ───────────────────────────────────────────────────────────────

export interface ThemeForgeOptions {
  mode?: 'light' | 'dark';
}

/**
 * Generate a complete, valid Theme from a seed string.
 *
 * The theme is deterministic: same seed + same mode = same theme, always.
 * The output passes validateTheme() — all REQUIRED_COLOR_TOKENS are present.
 *
 * @example
 * const theme = generateTheme('Acme Instrumentation');
 * applyTheme(theme);                    // live-themes the page
 * console.log(validateTheme(theme));    // []  ← empty = valid
 */
export function generateTheme(seed: string, opts: ThemeForgeOptions = {}): Theme {
  const mode = opts.mode ?? 'light';
  const rng = seededRng(seed);

  // Step 1: Hue selection from seed (uniform in [0, 360))
  // We use the first rng value directly — cyrb53 distributes well.
  const hue = rng() * 360;

  // Step 2: Accent saturation — fairly saturated, but not neon
  // Dark seeds (low rng value) get slightly warmer saturation
  const accentSat = 0.52 + rng() * 0.18; // [0.52, 0.70]

  // Step 3: Find AA lightness for accent
  const accentL = mode === 'light'
    ? findAALightness(hue, accentSat, 4.5)
    : clamp(findAALightness(hue, accentSat, 4.5) + 0.30, 0.45, 0.75);
  // Dark mode: accent needs to be lighter to contrast on dark backgrounds.

  const accentHex = rgbToHex(...hslToRgb(hue, accentSat, accentL));

  // Step 4: Hover and pressed states (darken by small steps in light mode,
  // lighten in dark mode to approach the accent's luminance plateau)
  const accentHover = mode === 'light'
    ? adjustL(accentHex, -0.06)
    : adjustL(accentHex, 0.06);
  const accentPressed = mode === 'light'
    ? adjustL(accentHex, -0.11)
    : adjustL(accentHex, 0.11);

  // Accent contrast text: white on dark accents, dark text on very light accents
  const [ar, ag, ab] = hexToRgb(accentHex);
  const accentLum = relativeLuminance(ar, ag, ab);
  const accentContrastHex = accentLum < 0.2
    ? '#FFFFFF'
    : rgbToHex(...hslToRgb(hue, 0.20, 0.10));

  // Step 5: Neutral ramp (tinted, hue-derived)
  const neutral = deriveNeutralRamp(hue, mode);

  // Step 6: Tints — alpha variants of the neutral text color and accent
  // (v3.1 calibration: one perceptual step stronger for sRGB panels)
  const textBase = neutral.text;
  const tint = hexAlpha(textBase, 0.05);
  const tintMedium = hexAlpha(textBase, 0.1);
  const tintStrong = hexAlpha(textBase, 0.15);
  const accentTint = hexAlpha(accentHex, 0.1);
  const accentTintStrong = hexAlpha(accentHex, 0.18);

  // Step 7: Status colors
  const status = deriveStatus(hue, mode);

  // Step 8: Glass, scrim, focus (v3.1: more opaque glass — text stays crisp
  // over blur on standard-DPI panels)
  const glassBg = hexAlpha(neutral.surfaceRaised, 0.88);
  const glassBorder = hexAlpha(neutral.text, 0.1);
  const scrim = hexAlpha(neutral.text, 0.48);
  const focusRing = accentHex;

  // Step 9: Chart series palette — golden-angle walk from the accent
  const chart = deriveChartPalette(accentHex, mode);

  const tokens: Theme['tokens'] = {
    bg: neutral.bg,
    surface: neutral.surface,
    'surface-raised': neutral.surfaceRaised,
    'surface-sunken': neutral.surfaceSunken,
    text: neutral.text,
    'text-secondary': neutral.textSecondary,
    'text-muted': neutral.textMuted,
    'text-inverse': neutral.textInverse,
    accent: accentHex,
    'accent-hover': accentHover,
    'accent-pressed': accentPressed,
    'accent-contrast': accentContrastHex,
    'accent-tint': accentTint,
    'accent-tint-strong': accentTintStrong,
    'inverse-surface': neutral.inverseSurface,
    border: neutral.border,
    'border-strong': neutral.borderStrong,
    tint,
    'tint-medium': tintMedium,
    'tint-strong': tintStrong,
    success: status.success,
    'success-tint': status.successTint,
    warning: status.warning,
    'warning-tint': status.warningTint,
    danger: status.danger,
    'danger-tint': status.dangerTint,
    info: status.info,
    'info-tint': status.infoTint,
    'glass-bg': glassBg,
    'glass-border': glassBorder,
    'focus-ring': focusRing,
    scrim,
    'chart-1': chart[0],
    'chart-2': chart[1],
    'chart-3': chart[2],
    'chart-4': chart[3],
    'chart-5': chart[4],
    'chart-6': chart[5],
    'chart-7': chart[6],
    'chart-8': chart[7],
  };

  const theme: Theme = {
    name: `forge-${seed.toLowerCase().replace(/\W+/g, '-')}-${mode}`,
    tokens,
  };

  // Invariant: must pass validation (programming error if it doesn't)
  const missing = validateTheme(theme);
  if (missing.length > 0) {
    throw new Error(`[themeForge] BUG: generated theme is missing tokens: ${missing.join(', ')}`);
  }

  return theme;
}

/**
 * Compute WCAG contrast ratio between a foreground and background hex color.
 * contrastRatio is exported above (line ~88) for ThemeForgePage AA/AAA badges.
 */
