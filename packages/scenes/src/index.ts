/**
 * @asymmflow/scenes — the magic edges.
 *
 * Generative identity, ambient depth, and the seed→theme design engine.
 * Lives at the edges of the design system (Constitution §0):
 * login ceremonies, dashboard ambient layers, empty states, arrival moments.
 *
 * Dependency hierarchy (Constitution §1):
 *   @asymmflow/scenes → @asymmflow/motion → @asymmflow/tokens
 *
 * Nothing imports upward from here. Scenes do NOT import @asymmflow/ui.
 */

// ── Deterministic PRNG (seeded) ──────────────────────────────────────────────
export { seededRng, cyrb53, mulberry32, rngRange, rngInt } from './rng.js';

// ── Quaternion walk on S³ ────────────────────────────────────────────────────
export { walkPoints, projectTo2D, smoothPath } from './quaternionWalk.js';
export type { WalkPoint } from './quaternionWalk.js';

// ── Theme forge (design engine seam) ────────────────────────────────────────
export { generateTheme, contrastRatio } from './themeForge.js';
export type { ThemeForgeOptions } from './themeForge.js';

// ── Svelte components ────────────────────────────────────────────────────────
export { default as GlyphMark } from './GlyphMark.svelte';
export { default as AmbientField } from './AmbientField.svelte';
export { default as LoginCeremony } from './LoginCeremony.svelte';
