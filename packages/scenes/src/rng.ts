/**
 * rng.ts — deterministic seeded PRNG + string hash.
 *
 * Same seed always produces the same sequence.
 * No Date.now(), no Math.random() — this module is purely deterministic.
 *
 * Algorithm: mulberry32 PRNG seeded via cyrb53 string hash.
 * Both are well-known, fast, and produce good statistical distribution
 * for our purposes (generative identity marks, theme derivation).
 */

/**
 * cyrb53 — a fast, high-quality 53-bit string hash.
 * Credit: bryc (https://github.com/bryc/code/blob/master/jshash/experimental/cyrb53.md)
 * Returns a number in [0, 2^53).
 */
export function cyrb53(str: string, seed = 0): number {
  let h1 = 0xdeadbeef ^ seed;
  let h2 = 0x41c6ce57 ^ seed;
  for (let i = 0; i < str.length; i++) {
    const ch = str.charCodeAt(i);
    h1 = Math.imul(h1 ^ ch, 2654435761);
    h2 = Math.imul(h2 ^ ch, 1597334677);
  }
  h1 = Math.imul(h1 ^ (h1 >>> 16), 2246822507);
  h1 ^= Math.imul(h2 ^ (h2 >>> 13), 3266489909);
  h2 = Math.imul(h2 ^ (h2 >>> 16), 2246822507);
  h2 ^= Math.imul(h1 ^ (h1 >>> 13), 3266489909);
  // Combine into a single 53-bit integer.
  return 4294967296 * (2097151 & h2) + (h1 >>> 0);
}

/**
 * mulberry32 — a fast, high-quality 32-bit PRNG.
 * Takes a 32-bit integer seed; returns a callable that yields numbers in [0, 1).
 */
export function mulberry32(seed: number): () => number {
  let s = seed >>> 0; // ensure 32-bit unsigned
  return function () {
    s += 0x6d2b79f5;
    let t = s;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    t = (t ^ (t >>> 14)) >>> 0;
    return t / 0x100000000;
  };
}

/**
 * Create a deterministic PRNG from a seed string.
 * Returns a function that yields numbers in [0, 1) with no external state.
 *
 * @example
 * const rng = seededRng('Acme Instrumentation');
 * const a = rng(); // always the same first value for this seed
 */
export function seededRng(seed: string): () => number {
  // Use cyrb53 to map the string to a 32-bit seed for mulberry32.
  // We fold the 53-bit hash into 32 bits via XOR of upper/lower halves.
  const hash = cyrb53(seed);
  const seed32 = ((hash >>> 0) ^ ((hash / 0x100000000) >>> 0)) >>> 0;
  return mulberry32(seed32);
}

/**
 * Draw n values uniformly from [lo, hi) using the given rng.
 */
export function rngRange(rng: () => number, lo: number, hi: number): number {
  return lo + rng() * (hi - lo);
}

/**
 * Pick a random integer in [lo, hi] inclusive.
 */
export function rngInt(rng: () => number, lo: number, hi: number): number {
  return Math.floor(rngRange(rng, lo, hi + 1));
}
