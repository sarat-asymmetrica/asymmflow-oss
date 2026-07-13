/**
 * Motion helpers — the JS-side companion to the CSS motion tokens in
 * `assets/design-tokens.css`.
 *
 * The global `@media (prefers-reduced-motion: reduce)` reset in design-tokens.css
 * neutralises every CSS transition/animation, but Svelte's `fade`/`fly`/`scale`
 * transitions animate inline styles over a JS timer (requestAnimationFrame) and
 * are NOT affected by that CSS reset. `motionMs()` closes that gap: pass a
 * transition's duration through it and the animation collapses to instant when
 * the user has asked for reduced motion — so the app renders fully static.
 *
 * Article IV.2 (Design Constitution): "Always honor prefers-reduced-motion."
 */

export function prefersReducedMotion(): boolean {
	return (
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(prefers-reduced-motion: reduce)').matches
	);
}

/**
 * Duration to hand a Svelte transition: the requested ms normally, or 0 when
 * the user prefers reduced motion (Svelte treats a 0ms transition as instant).
 */
export function motionMs(ms: number): number {
	return prefersReducedMotion() ? 0 : ms;
}
