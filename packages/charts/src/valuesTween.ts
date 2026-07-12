/**
 * valuesTween.ts — chart transitions on the geodesic engine.
 *
 * Wraps @asymmflow/motion's createGeodesicTween for the chart case: an
 * ARRAY of numbers (bar heights, arc angles, line y-values) tweening as one
 * coherent state. Because all values share a single angular parameter on the
 * hypersphere, a re-sorted bar chart moves as one gesture — and a mid-flight
 * data update retargets smoothly from wherever the bars currently are.
 * Interruptible by construction; respects prefers-reduced-motion (jumps).
 *
 * Length-change rule: same-length updates tween; a different length jumps
 * (there is no meaningful geodesic between spaces of different dimension).
 */

import {
  createGeodesicTween,
  type GeodesicTween,
  type TransitionOptions,
} from '@asymmflow/motion';

export type { TransitionOptions };

export interface ValuesTween {
  /** Latest interpolated values (same array your onUpdate receives). */
  readonly current: number[];
  readonly moving: boolean;
  /** Retarget — tweens if length matches, jumps otherwise. */
  to(values: number[], opts?: TransitionOptions): void;
  /** Snap instantly. */
  jump(values: number[]): void;
  stop(): void;
}

const toState = (vals: number[]): Record<string, number> =>
  Object.fromEntries(vals.map((v, i) => [`v${i}`, v]));

const fromState = (s: Record<string, number>, n: number): number[] =>
  Array.from({ length: n }, (_, i) => s[`v${i}`] ?? 0);

export function createValuesTween(
  initial: number[],
  onUpdate: (values: number[]) => void,
): ValuesTween {
  let n = initial.length;
  let current = [...initial];
  let tween: GeodesicTween<string> = createGeodesicTween(toState(initial), (s) => {
    current = fromState(s, n);
    onUpdate(current);
  });

  const rebuild = (vals: number[]) => {
    tween.stop();
    n = vals.length;
    tween = createGeodesicTween(toState(vals), (s) => {
      current = fromState(s, n);
      onUpdate(current);
    });
    current = [...vals];
    onUpdate(current);
  };

  return {
    get current() {
      return current;
    },
    get moving() {
      return tween.moving;
    },
    to(values, opts) {
      if (values.length !== n) {
        rebuild(values);
        return;
      }
      tween.to(toState(values), opts);
    },
    jump(values) {
      if (values.length !== n) {
        rebuild(values);
        return;
      }
      tween.jump(toState(values));
    },
    stop() {
      tween.stop();
    },
  };
}
