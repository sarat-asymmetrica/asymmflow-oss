
import { writable, derived } from 'svelte/store';
import { Quaternion } from '../math/quaternion';
import { Regime } from './regime';

// Define 4 themes as Quaternions
// Mapping strategy:
// W: Primary Hue (normalized 0-1)
// X: Primary Lightness (normalized 0-1)
// Y: Background Lightness (normalized 0-1)
// Z: Accent Hue (normalized 0-1)

export const THEME_QUATERNIONS = {
    DAVINCI: new Quaternion(0.11, 0.4, 0.95, 0.1), // Sepia/Paper like
    WABISABI: new Quaternion(0.3, 0.5, 0.2, 0.4),  // Earthy, darker
    VOID: new Quaternion(0.7, 0.1, 0.05, 0.8),     // Dark, Neon
    HOLO: new Quaternion(0.5, 0.9, 0.9, 0.6)       // Bright, Prismatic
};

export const currentRegime = writable<Regime>(Regime.Discovery);
export const targetThemeQuaternion = writable<Quaternion>(THEME_QUATERNIONS.DAVINCI);
export const currentThemeQuaternion = writable<Quaternion>(THEME_QUATERNIONS.DAVINCI);

// Helper to convert normalized quaternion components to CSS variables
export const themeVariables = derived(currentThemeQuaternion, ($q) => {
    const q = $q.normalize(); // Ensure unit quaternion for consistency
    // Simple mapping for demo purposes.
    // In a real sophisticated system, this would be a matrix transform.

    // We take absolute values because quaternion double cover property (q == -q)
    // but for color we want positive values.
    const h1 = Math.abs(q.w) * 360;
    const l1 = Math.abs(q.x) * 100;
    const l_bg = Math.abs(q.y) * 100;
    const h2 = Math.abs(q.z) * 360;

    return {
        // Raw quaternion values
        '--primary-hue': `${h1}`,
        '--primary-lightness': `${l1}%`,
        '--bg-lightness': `${l_bg}%`,
        '--accent-hue': `${h2}`,

        // Core semantic colors
        '--bg-color': `hsl(${h1}, 10%, ${l_bg}%)`,
        '--text-color': `hsl(${h1}, 10%, ${l_bg > 50 ? 10 : 90}%)`,
        '--accent-color': `hsl(${h2}, 80%, 60%)`,

        // Three-regime semantic colors (used by RegimeLoader, KintsugiError)
        '--danger-color': `hsl(0, 84%, 60%)`,      // R1 Discovery (30% high variance) - Red
        '--safe-color': `hsl(142, 71%, 45%)`,      // R3 Completion (50% stability) - Green
        '--gold-color': `hsl(43, 74%, 66%)`,       // Kintsugi gold - #fbbf24

        // UI element colors
        '--border-color': `hsl(${h1}, 10%, ${l_bg > 50 ? 80 : 20}%)`,

        // Spacing unit (φ-based: 8px base)
        '--spacing-unit': '8px',

        // Transition duration (inherited from regime.ts)
        '--transition-duration': '0.382s' // 1/φ²
    };
});
