/**
 * quaternionWalk.ts — deterministic walk on S³.
 *
 * Starting from the identity quaternion, we repeatedly multiply by small
 * random-axis rotations derived from the seeded PRNG. This produces a
 * reproducible sequence of unit quaternions that traces a path on the
 * 3-sphere — the same seed always produces the same path.
 *
 * Projection: we rotate the basis vector (0,0,1) by each quaternion to get
 * a 3D point on the unit sphere. The x,y components give us a 2D point for
 * SVG drawing; z gives subtle depth.
 *
 * Imports only @asymmflow/motion quaternion primitives — no reimplementation.
 */

import {
  QUAT_IDENTITY,
  fromAxisAngle,
  multiply,
  normalize,
  type Quat,
} from '@asymmflow/motion';

import { seededRng } from './rng.js';

export interface WalkPoint {
  x: number;
  y: number;
  z: number;
  /** The quaternion at this step, for consumers who want raw orientation. */
  q: Quat;
}

/**
 * Rotate a 3D vector by a unit quaternion via q * v * q⁻¹.
 * (Equivalent to rotating the vector in 3D.)
 */
function rotateVector(q: Quat, vx: number, vy: number, vz: number): [number, number, number] {
  // Sandwich product: q * (0, vx, vy, vz) * q⁻¹
  // Expanded form for efficiency (avoids two full quaternion multiplications):
  const tx = 2 * (q.y * vz - q.z * vy);
  const ty = 2 * (q.z * vx - q.x * vz);
  const tz = 2 * (q.x * vy - q.y * vx);
  return [
    vx + q.w * tx + q.y * tz - q.z * ty,
    vy + q.w * ty + q.z * tx - q.x * tz,
    vz + q.w * tz + q.x * ty - q.y * tx,
  ];
}

/**
 * Generate n walk points on S³ from a seed string.
 *
 * @param seed  - Deterministic seed (company name, user id, etc.)
 * @param n     - Number of points (path vertices). Typical: 12–32.
 * @param step  - Max rotation angle per step (radians). Smaller = tighter loops.
 *                Typical: 0.4–0.9. Constitutional default: 0.6 (elegant, not scattershot).
 *
 * Points are projected from S³→S² by rotating the basis vector (0,0,1),
 * then mapped to 2D by x,y. z is provided for optional depth effects.
 * All coordinates are in [-1, 1].
 */
export function walkPoints(seed: string, n: number, step = 0.6): WalkPoint[] {
  const rng = seededRng(seed);
  const points: WalkPoint[] = [];
  let q: Quat = { ...QUAT_IDENTITY };

  for (let i = 0; i < n; i++) {
    // Project current orientation: rotate z-basis by q
    const [x, y, z] = rotateVector(q, 0, 0, 1);
    points.push({ x, y, z, q: { ...q } });

    // Advance: multiply by a small random rotation
    // Axis: random unit vector from rng
    const ax = rng() * 2 - 1;
    const ay = rng() * 2 - 1;
    const az = rng() * 2 - 1;
    const angle = rng() * step; // [0, step] radians
    const rot = fromAxisAngle(ax, ay, az, angle);
    q = normalize(multiply(q, rot));
  }

  return points;
}

/**
 * Project walk points to a 2D canvas coordinate space.
 *
 * @param points  - Output of walkPoints()
 * @param width   - Canvas/SVG width
 * @param height  - Canvas/SVG height
 * @param padding - Margin from edges
 *
 * Returns {x, y} pairs scaled to the canvas.
 */
export function projectTo2D(
  points: WalkPoint[],
  width: number,
  height: number,
  padding = 0.1,
): Array<{ x: number; y: number }> {
  const p = padding;
  // Map from [-1,1] to [p*w, (1-p)*w] range
  return points.map(({ x, y }) => ({
    x: ((x + 1) / 2) * width * (1 - 2 * p) + width * p,
    y: ((y + 1) / 2) * height * (1 - 2 * p) + height * p,
  }));
}

/**
 * Build a smooth SVG path string from 2D points using quadratic Bézier
 * smoothing (each pair of midpoints becomes control/anchor). This produces
 * a single connected stroke that flows through all points without corners.
 */
export function smoothPath(pts: Array<{ x: number; y: number }>): string {
  if (pts.length < 2) return '';
  if (pts.length === 2) {
    return `M ${pts[0].x} ${pts[0].y} L ${pts[1].x} ${pts[1].y}`;
  }

  // Start at the midpoint between the first and second points
  const mid = (a: { x: number; y: number }, b: { x: number; y: number }) => ({
    x: (a.x + b.x) / 2,
    y: (a.y + b.y) / 2,
  });

  let d = '';
  const first = mid(pts[0], pts[1]);
  d += `M ${first.x.toFixed(2)} ${first.y.toFixed(2)}`;

  for (let i = 1; i < pts.length - 1; i++) {
    const ctrl = pts[i];
    const end = mid(pts[i], pts[i + 1]);
    d += ` Q ${ctrl.x.toFixed(2)} ${ctrl.y.toFixed(2)} ${end.x.toFixed(2)} ${end.y.toFixed(2)}`;
  }

  // Final segment to the last point
  const last = pts[pts.length - 1];
  d += ` L ${last.x.toFixed(2)} ${last.y.toFixed(2)}`;

  return d;
}
