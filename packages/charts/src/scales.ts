/**
 * scales.ts — the headless math layer, re-exported from d3.
 *
 * d3-scale / d3-shape / d3-array are pure functions (domain→range mapping,
 * path generation, statistics) with no DOM opinions. Components in this
 * package import from HERE, not from d3 directly, so the dependency surface
 * stays in one file.
 */

export {
  scaleLinear,
  scaleBand,
  scaleTime,
  scalePoint,
  scaleSqrt,
} from 'd3-scale';
export type { ScaleLinear, ScaleBand, ScaleTime, ScalePoint } from 'd3-scale';

export {
  line,
  area,
  arc,
  pie,
  stack,
  stackOffsetNone,
  curveMonotoneX,
  curveLinear,
  curveStepAfter,
} from 'd3-shape';
export type { Line, Area, Arc, Pie, PieArcDatum, Series } from 'd3-shape';

export { extent, max, min, sum, range, ticks } from 'd3-array';

/**
 * Minimal structural type every axis-compatible scale satisfies.
 * Lets Axis.svelte accept linear, band, point and time scales alike.
 */
export interface AnyScale {
  (value: never): number | undefined;
  domain(): unknown[];
  range(): number[];
  ticks?(count?: number): unknown[];
  bandwidth?(): number;
}

/**
 * A "pleasant" axis ceiling: rounds up to 1/2/2.5/5 × 10^k.
 * niceCeil(8_643) → 10_000; niceCeil(412) → 500.
 */
export function niceCeil(value: number): number {
  if (value <= 0) return 0;
  const exp = Math.floor(Math.log10(value));
  const base = Math.pow(10, exp);
  const m = value / base;
  const step = m <= 1 ? 1 : m <= 2 ? 2 : m <= 2.5 ? 2.5 : m <= 5 ? 5 : 10;
  return step * base;
}
