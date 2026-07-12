/**
 * internal.ts — shared helpers for LineChart and AreaChart.
 *
 * Lives inside the timeseries directory (exclusive ownership). Not re-exported
 * from the package index; used only within this family.
 */

import { scaleLinear, scaleTime, extent, max, min, niceCeil } from '../scales.js';
import type { ScaleLinear, ScaleTime } from '../scales.js';

export type AnyXScale = ScaleLinear<number, number> | ScaleTime<number, number>;

export interface TimePoint {
  x: number | Date;
  y: number;
}

export interface Series {
  label: string;
  points: TimePoint[];
}

/**
 * Detect whether the series uses Date x-values.
 * Checks the first non-empty series' first point.
 */
export function isDateSeries(series: Series[]): boolean {
  for (const s of series) {
    if (s.points.length > 0) return s.points[0].x instanceof Date;
  }
  return false;
}

/**
 * Build the x scale (time or linear) from all visible series.
 */
export function buildXScale(
  series: Series[],
  innerWidth: number,
  hidden: string[],
): AnyXScale {
  const visible = series.filter((s) => !hidden.includes(s.label));
  const allX = visible.flatMap((s) => s.points.map((p) => p.x)) as (number | Date)[];

  if (allX.length === 0) {
    return scaleLinear().domain([0, 1]).range([0, innerWidth]);
  }

  if (allX[0] instanceof Date) {
    const [lo, hi] = extent(allX as Date[]) as [Date, Date];
    return scaleTime().domain([lo, hi]).range([0, innerWidth]);
  }

  const [lo, hi] = extent(allX as number[]) as [number, number];
  return scaleLinear().domain([lo, hi]).range([0, innerWidth]);
}

/**
 * Build the y scale from all visible series.
 * Domain starts at 0 unless data goes negative (then from data min).
 */
export function buildYScale(
  series: Series[],
  innerHeight: number,
  hidden: string[],
): ScaleLinear<number, number> {
  const visible = series.filter((s) => !hidden.includes(s.label));
  const allY = visible.flatMap((s) => s.points.map((p) => p.y));

  if (allY.length === 0) {
    return scaleLinear().domain([0, 100]).range([innerHeight, 0]);
  }

  const dataMin = min(allY) ?? 0;
  const dataMax = max(allY) ?? 0;
  const yMin = dataMin < 0 ? dataMin : 0;
  const yMax = niceCeil(dataMax) || 1;

  return scaleLinear().domain([yMin, yMax]).range([innerHeight, 0]);
}

/**
 * Find the nearest x-index across all visible series for a given plot-area x.
 * Uses the first visible series as the canonical x set.
 */
export function findNearestIndex(
  plotX: number,
  series: Series[],
  hidden: string[],
  xScale: AnyXScale,
): number {
  const visible = series.filter((s) => !hidden.includes(s.label));
  if (visible.length === 0) return -1;
  const points = visible[0].points;
  if (points.length === 0) return -1;

  let best = 0;
  let bestDist = Infinity;
  for (let i = 0; i < points.length; i++) {
    const sx = xScale(points[i].x as never) ?? 0;
    const d = Math.abs(sx - plotX);
    if (d < bestDist) {
      bestDist = d;
      best = i;
    }
  }
  return best;
}

/**
 * Format an x value for tooltip headers.
 * Dates → short locale string; numbers → as-is.
 */
export function defaultXFormat(x: number | Date): string {
  if (x instanceof Date) {
    return x.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  }
  return String(x);
}
