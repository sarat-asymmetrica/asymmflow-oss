
<script lang="ts">
    import { run } from 'svelte/legacy';

    import { onMount } from 'svelte';
    import { currentThemeQuaternion, targetThemeQuaternion, themeVariables } from './theme';
    import { Quaternion } from '../math/quaternion';
    import { ANIMATION_DURATION } from './regime';
    interface Props {
        children?: import('svelte').Snippet;
    }

    let { children }: Props = $props();

    let animationFrame: number;
    let t = 0;

    // Subscribe to target changes to trigger animation
    targetThemeQuaternion.subscribe((target) => {
        // Start animation from current to target
        const start = $currentThemeQuaternion;
        t = 0;

        const animate = () => {
            t += 0.02; // speed factor
            if (t > 1) t = 1;

            const next = Quaternion.slerp(start, target, t);
            currentThemeQuaternion.set(next);

            if (t < 1) {
                animationFrame = requestAnimationFrame(animate);
            }
        };
        cancelAnimationFrame(animationFrame);
        animationFrame = requestAnimationFrame(animate);
    });

    // Apply CSS variables to root
    run(() => {
        if (typeof document !== 'undefined') {
            const vars = $themeVariables;
            Object.entries(vars).forEach(([key, value]) => {
                document.documentElement.style.setProperty(key, value);
            });
            // Apply global transition duration
            document.documentElement.style.setProperty('--transition-duration', `${ANIMATION_DURATION.medium}s`);
        }
    });
</script>

{@render children?.()}
