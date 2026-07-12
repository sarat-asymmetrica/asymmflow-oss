/**
 * Geodesic state transitions — the tactile signature of AsymmFlow.
 *
 * A UI state (any record of numbers: x, y, scale, opacity, …) is embedded as
 * a point on the unit hypersphere via a homogeneous coordinate; transitions
 * travel the geodesic (generalized SLERP) between embeddings, so EVERY
 * property advances under one shared angular parameter — they arrive
 * together, as one coherent movement, not as N parallel tweens.
 *
 * Interruption contract (Constitution §2.7): retargeting mid-flight starts a
 * new geodesic FROM THE CURRENT INTERPOLATED POINT. Nothing snaps, nothing
 * jumps, no tween-killing. Click fast — the surface flows.
 */

import { MOTION_REGIMES, type MotionRegime } from '@asymmflow/tokens';

export type StateVector<K extends string = string> = Record<K, number>;

// ── Cubic bezier easing (same curves as the CSS tokens) ──────────────────

/** Build an easing function from cubic-bezier control points (x1,y1,x2,y2). */
export function cubicBezier(x1: number, y1: number, x2: number, y2: number) {
  // Newton–Raphson on the parametric x(t), then evaluate y(t).
  const cx = 3 * x1;
  const bx = 3 * (x2 - x1) - cx;
  const ax = 1 - cx - bx;
  const cy = 3 * y1;
  const by = 3 * (y2 - y1) - cy;
  const ay = 1 - cy - by;

  const sampleX = (t: number) => ((ax * t + bx) * t + cx) * t;
  const sampleY = (t: number) => ((ay * t + by) * t + cy) * t;
  const sampleDX = (t: number) => (3 * ax * t + 2 * bx) * t + cx;

  return (x: number): number => {
    if (x <= 0) return 0;
    if (x >= 1) return 1;
    let t = x;
    for (let i = 0; i < 5; i++) {
      const err = sampleX(t) - x;
      if (Math.abs(err) < 1e-5) break;
      const d = sampleDX(t);
      if (Math.abs(d) < 1e-6) break;
      t -= err / d;
    }
    return sampleY(Math.max(0, Math.min(1, t)));
  };
}

const regimeEasings = Object.fromEntries(
  (Object.keys(MOTION_REGIMES) as MotionRegime[]).map((r) => {
    const [x1, y1, x2, y2] = MOTION_REGIMES[r].ease;
    return [r, cubicBezier(x1, y1, x2, y2)];
  }),
) as Record<MotionRegime, (x: number) => number>;

// ── Hypersphere embedding + generalized slerp ────────────────────────────

interface Embedded {
  keys: string[];
  /** Unit direction on S^n (state components + homogeneous w). */
  dir: number[];
  /** Magnitude restored after slerp. */
  mag: number;
}

/**
 * The homogeneous coordinate keeps the embedding away from the origin so a
 * zero state vector is still a valid point on the sphere (Constitution echo
 * of "no failure state on S³ — you're always somewhere valid").
 */
const HOMOGENEOUS_W = 1;

function embed(keys: string[], v: StateVector): Embedded {
  const comps = keys.map((k) => v[k] ?? 0);
  comps.push(HOMOGENEOUS_W);
  const mag = Math.hypot(...comps);
  return { keys, dir: comps.map((c) => c / mag), mag };
}

function unembed(e: Embedded): StateVector {
  const out: StateVector = {};
  const scale = e.mag;
  for (let i = 0; i < e.keys.length; i++) {
    out[e.keys[i]] = e.dir[i] * scale;
  }
  return out;
}

/** Generalized slerp between two unit vectors of equal dimension. */
function slerpDir(a: number[], b: number[], t: number): number[] {
  let d = 0;
  for (let i = 0; i < a.length; i++) d += a[i] * b[i];
  d = Math.max(-1, Math.min(1, d));

  if (d > 0.9995) {
    // Nearly parallel — lerp + renormalize.
    const out = a.map((av, i) => av + (b[i] - av) * t);
    const len = Math.hypot(...out) || 1;
    return out.map((c) => c / len);
  }

  const theta = Math.acos(d);
  const sinTheta = Math.sin(theta);
  const wa = Math.sin((1 - t) * theta) / sinTheta;
  const wb = Math.sin(t * theta) / sinTheta;
  return a.map((av, i) => wa * av + wb * b[i]);
}

/**
 * One point along the geodesic between two state vectors.
 * Magnitude interpolates linearly; direction travels the great-circle arc.
 */
export function slerpState<K extends string>(
  from: StateVector<K>,
  to: StateVector<K>,
  t: number,
): StateVector<K> {
  const keys = Object.keys(from) as K[];
  const ea = embed(keys, from);
  const eb = embed(keys, to);
  const dir = slerpDir(ea.dir, eb.dir, t);
  const mag = ea.mag + (eb.mag - ea.mag) * t;
  return unembed({ keys, dir, mag }) as StateVector<K>;
}

// ── The interruptible tween driver ───────────────────────────────────────

export interface TransitionOptions {
  /** Motion regime — supplies duration AND easing from the token contract. */
  regime?: MotionRegime;
  /** Override duration (ms). Prefer regimes; reach for this in ceremonies only. */
  duration?: number;
  onComplete?: () => void;
}

export interface GeodesicTween<K extends string> {
  /** Current interpolated state — read each frame via onUpdate. */
  readonly current: StateVector<K>;
  readonly moving: boolean;
  /** Retarget. Mid-flight calls re-geodesic from the current point. */
  to(target: Partial<StateVector<K>>, opts?: TransitionOptions): void;
  /** Snap instantly (used by reduced-motion). */
  jump(target: Partial<StateVector<K>>): void;
  stop(): void;
}

const prefersReducedMotion = () =>
  typeof window !== 'undefined' &&
  window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;

/**
 * Create an interruptible geodesic tween over a named state vector.
 * Drives requestAnimationFrame only while moving; idle costs nothing.
 */
export function createGeodesicTween<K extends string>(
  initial: StateVector<K>,
  onUpdate: (state: StateVector<K>) => void,
): GeodesicTween<K> {
  let current: StateVector<K> = { ...initial };
  let from: StateVector<K> = { ...initial };
  let target: StateVector<K> = { ...initial };
  let startTime = 0;
  let duration: number = MOTION_REGIMES.stabilize.duration;
  let ease = regimeEasings.stabilize;
  let onComplete: (() => void) | undefined;
  let raf = 0;
  let moving = false;

  function frame(now: number) {
    const t = duration <= 0 ? 1 : Math.min(1, (now - startTime) / duration);
    current = slerpState(from, target, ease(t));
    onUpdate(current);
    if (t < 1) {
      raf = requestAnimationFrame(frame);
    } else {
      current = { ...target };
      onUpdate(current);
      moving = false;
      onComplete?.();
    }
  }

  return {
    get current() {
      return current;
    },
    get moving() {
      return moving;
    },
    to(partial, opts = {}) {
      const next = { ...target, ...partial } as StateVector<K>;
      if (prefersReducedMotion()) {
        this.jump(next);
        return;
      }
      // Interruption: depart from wherever we are right now.
      from = { ...current };
      target = next;
      const regime = opts.regime ?? 'stabilize';
      duration = opts.duration ?? MOTION_REGIMES[regime].duration;
      ease = regimeEasings[regime];
      onComplete = opts.onComplete;
      startTime = performance.now();
      if (!moving) {
        moving = true;
        raf = requestAnimationFrame(frame);
      }
    },
    jump(partial) {
      cancelAnimationFrame(raf);
      moving = false;
      current = { ...current, ...partial } as StateVector<K>;
      from = { ...current };
      target = { ...current };
      onUpdate(current);
    },
    stop() {
      cancelAnimationFrame(raf);
      moving = false;
    },
  };
}
