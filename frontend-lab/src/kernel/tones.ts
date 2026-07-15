/* The semantic tone vocabulary — ONE definition (L2). Every tone-coloured
 * surface (Badge, summary distribution bar, threshold-coloured cell) resolves
 * its colour from the --k-tone-* custom properties defined in kernel.css, so a
 * palette change happens in exactly one place. */

export type Tone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'

export const TONES: readonly Tone[] = ['neutral', 'info', 'success', 'warning', 'danger']

/** CSS var reference for a tone's foreground or background colour. */
export function toneVar(tone: Tone, which: 'fg' | 'bg'): string {
  return `var(--k-tone-${tone}-${which})`
}
