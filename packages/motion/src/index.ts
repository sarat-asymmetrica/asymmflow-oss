/**
 * @asymmflow/motion — interruptible geodesic state transitions.
 * Constitution: packages/DESIGN_CONSTITUTION.md §5.
 */

export {
  quat,
  QUAT_IDENTITY,
  fromAxisAngle,
  normalize,
  multiply,
  conjugate,
  dot,
  slerp,
  toCssRotate,
} from './quaternion.js';
export type { Quat } from './quaternion.js';

export {
  cubicBezier,
  slerpState,
  createGeodesicTween,
} from './geodesic.js';
export type {
  StateVector,
  TransitionOptions,
  GeodesicTween,
} from './geodesic.js';

// Re-export the regime contract so consumers need one import.
export { MOTION_REGIMES, MOTION_STAGGER_MS, PHI } from '@asymmflow/tokens';
export type { MotionRegime } from '@asymmflow/tokens';
