/**
 * Quaternion core — unit quaternions on S³ for orientation animation.
 *
 * Used by @asymmflow/scenes for 3D glyph/scene orientation and exposed to
 * consumers who animate CSS rotate3d. The general state-vector geodesic
 * (any number of properties) lives in geodesic.ts; this file is the true
 * 4D rotational case.
 */

export interface Quat {
  w: number;
  x: number;
  y: number;
  z: number;
}

export const QUAT_IDENTITY: Readonly<Quat> = Object.freeze({ w: 1, x: 0, y: 0, z: 0 });

export function quat(w: number, x: number, y: number, z: number): Quat {
  return { w, x, y, z };
}

/** Unit quaternion from an axis (need not be normalized) and angle in radians. */
export function fromAxisAngle(ax: number, ay: number, az: number, angle: number): Quat {
  const len = Math.hypot(ax, ay, az) || 1;
  const half = angle / 2;
  const s = Math.sin(half) / len;
  return { w: Math.cos(half), x: ax * s, y: ay * s, z: az * s };
}

export function normalize(q: Quat): Quat {
  const len = Math.hypot(q.w, q.x, q.y, q.z) || 1;
  return { w: q.w / len, x: q.x / len, y: q.y / len, z: q.z / len };
}

/** Hamilton product a ⊗ b (apply b's rotation, then a's). */
export function multiply(a: Quat, b: Quat): Quat {
  return {
    w: a.w * b.w - a.x * b.x - a.y * b.y - a.z * b.z,
    x: a.w * b.x + a.x * b.w + a.y * b.z - a.z * b.y,
    y: a.w * b.y - a.x * b.z + a.y * b.w + a.z * b.x,
    z: a.w * b.z + a.x * b.y - a.y * b.x + a.z * b.w,
  };
}

export function conjugate(q: Quat): Quat {
  return { w: q.w, x: -q.x, y: -q.y, z: -q.z };
}

export function dot(a: Quat, b: Quat): number {
  return a.w * b.w + a.x * b.x + a.y * b.y + a.z * b.z;
}

/**
 * Spherical linear interpolation along the SHORTEST arc on S³.
 * Constant angular velocity in t ∈ [0, 1] — the property that makes
 * mid-flight retargeting seamless (re-slerp from wherever you are).
 */
export function slerp(a: Quat, b: Quat, t: number): Quat {
  let d = dot(a, b);
  // q and -q encode the same rotation; flip to take the short way around.
  let bw = b.w, bx = b.x, by = b.y, bz = b.z;
  if (d < 0) {
    d = -d;
    bw = -bw; bx = -bx; by = -by; bz = -bz;
  }

  // Nearly parallel: lerp + normalize avoids 0/0.
  if (d > 0.9995) {
    return normalize({
      w: a.w + (bw - a.w) * t,
      x: a.x + (bx - a.x) * t,
      y: a.y + (by - a.y) * t,
      z: a.z + (bz - a.z) * t,
    });
  }

  const theta = Math.acos(Math.min(1, d));
  const sinTheta = Math.sin(theta);
  const wa = Math.sin((1 - t) * theta) / sinTheta;
  const wb = Math.sin(t * theta) / sinTheta;
  return {
    w: wa * a.w + wb * bw,
    x: wa * a.x + wb * bx,
    y: wa * a.y + wb * by,
    z: wa * a.z + wb * bz,
  };
}

/** CSS transform string: rotate3d equivalent of the quaternion. */
export function toCssRotate(q: Quat): string {
  const n = normalize(q);
  const angle = 2 * Math.acos(Math.max(-1, Math.min(1, n.w)));
  const s = Math.sqrt(Math.max(0, 1 - n.w * n.w));
  if (s < 1e-6 || angle < 1e-6) return 'rotate3d(0, 0, 1, 0deg)';
  const deg = (angle * 180) / Math.PI;
  return `rotate3d(${n.x / s}, ${n.y / s}, ${n.z / s}, ${deg}deg)`;
}
