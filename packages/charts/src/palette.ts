/**
 * palette.ts — chart series colors, token-driven.
 *
 * Themes define --af-chart-1..8 (a golden-angle hue walk from the accent;
 * see @asymmflow/tokens REQUIRED_COLOR_TOKENS and scenes/themeForge).
 * Charts NEVER hardcode series colors — they reference these tokens, so a
 * live applyTheme() re-skins every chart on the page.
 *
 * Constitution: packages/DESIGN_CONSTITUTION.md §2.2 (one token source).
 */

import { CHART_SERIES_COUNT } from '@asymmflow/tokens';

/** Token name for series i (0-based): seriesToken(0) → '--af-chart-1'. Wraps past 8. */
export function seriesToken(i: number): string {
  const n = ((i % CHART_SERIES_COUNT) + CHART_SERIES_COUNT) % CHART_SERIES_COUNT;
  return `--af-chart-${n + 1}`;
}

/** CSS value for series i: seriesColor(0) → 'var(--af-chart-1)'. Use in SVG fill/stroke. */
export function seriesColor(i: number): string {
  return `var(${seriesToken(i)})`;
}

/**
 * Resolve the full palette to concrete color strings (for canvas, gradients,
 * or anywhere var() can't reach). Reads computed style from `el` (default
 * document root) so theme overrides apply.
 */
export function readChartPalette(el?: Element): string[] {
  const target = el ?? document.documentElement;
  const cs = getComputedStyle(target);
  const colors: string[] = [];
  for (let i = 0; i < CHART_SERIES_COUNT; i++) {
    colors.push(cs.getPropertyValue(seriesToken(i)).trim());
  }
  return colors;
}

export { CHART_SERIES_COUNT };
