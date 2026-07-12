
<script lang="ts">
    /**
     * RegimeLoader - Three-regime aware loading indicator
     *
     * Dynamically adjusts animation complexity based on current regime:
     * - Discovery (R1 30%): 5 rings, fast chaotic motion
     * - Refinement (R2 20%): 3 rings, medium balanced motion
     * - Completion (R3 50%): 1 ring, slow stable motion
     *
     * @component
     */
    import { onMount, onDestroy } from 'svelte';
    import { currentRegime } from "../core/theme";
    import { Regime } from "../core/regime";

    

    

    
    interface Props {
        /** Canvas width in pixels */
        width?: number;
        /** Canvas height in pixels */
        height?: number;
        /** ARIA label for accessibility */
        ariaLabel?: string;
    }

    let { width = 200, height = 200, ariaLabel = "Loading indicator" }: Props = $props();

    let canvas: HTMLCanvasElement = $state();
    let ctx: CanvasRenderingContext2D | null;
    let animationId: number;
    let t = 0;

    // φ-based animation timing
    const PHI = 1.618;

    onMount(() => {
        ctx = canvas.getContext('2d');
        if (!ctx) return;

        const render = () => {
            if (!ctx || !canvas) return;
            const w = canvas.width;
            const h = canvas.height;
            const cx = w / 2;
            const cy = h / 2;

            ctx.clearRect(0, 0, w, h);

            t += 0.02;

            // Regime-based complexity adaptation
            let rings = 1;
            let speed = 1;
            let color = 'var(--accent-color, #c5a059)'; // Gold default

            if ($currentRegime === Regime.Discovery) {
                // High complexity, chaotic (30% regime)
                rings = 5;
                speed = 2;
                color = 'var(--danger-color, #ef4444)'; // Red for high variance
            } else if ($currentRegime === Regime.Refinement) {
                // Balanced (20% regime)
                rings = 3;
                speed = 1;
                color = 'var(--accent-color, #c5a059)'; // Gold for optimization
            } else if ($currentRegime === Regime.Completion) {
                // Simple, stable (50% regime)
                rings = 1;
                speed = 0.5;
                color = 'var(--safe-color, #15803d)'; // Green for stability
            }

            // Draw multiple rotating rings
            for (let i = 0; i < rings; i++) {
                const r = 30 + i * 15;
                const offset = i * (Math.PI / rings);

                ctx.beginPath();
                // Simulate 3D rotation by scaling Y axis
                const scaleY = Math.sin(t * speed + offset);

                ctx.ellipse(
                    cx, cy,
                    r,
                    r * Math.abs(scaleY),
                    t * speed + offset,
                    0,
                    2 * Math.PI
                );
                ctx.strokeStyle = color;
                ctx.lineWidth = 2;
                ctx.stroke();

                // Draw connecting nodes (except in Completion regime for simplicity)
                if ($currentRegime !== Regime.Completion) {
                    const nodes = 3 + i;
                    for (let j = 0; j < nodes; j++) {
                        const angle = (j / nodes) * 2 * Math.PI + t * speed;
                        const nx = cx + Math.cos(angle) * r;
                        const ny = cy + Math.sin(angle) * r * scaleY;

                        // Rotate point around center for tilt effect
                        const tiltedX = (nx - cx) * Math.cos(offset) - (ny - cy) * Math.sin(offset) + cx;
                        const tiltedY = (nx - cx) * Math.sin(offset) + (ny - cy) * Math.cos(offset) + cy;

                        ctx.beginPath();
                        ctx.arc(tiltedX, tiltedY, 3, 0, 2 * Math.PI);
                        ctx.fillStyle = color;
                        ctx.fill();
                    }
                }
            }

            animationId = requestAnimationFrame(render);
        };
        render();
    });

    onDestroy(() => {
        if (typeof window !== 'undefined') {
            cancelAnimationFrame(animationId);
        }
    });
</script>

<div class="regime-loader" title="Regime: {$currentRegime}" role="status" aria-label={ariaLabel}>
    <canvas bind:this={canvas} width={width} height={height} class="loader-canvas"></canvas>
    <div class="regime-label">
        {$currentRegime}
    </div>
</div>

<style>
    .regime-loader {
        display: inline-block;
        text-align: center;
    }

    .loader-canvas {
        display: block;
    }

    .regime-label {
        margin-top: 8px;
        font-family: 'Courier New', monospace;
        font-size: 0.75rem;
        color: color-mix(in srgb, var(--text-color, #6b7280) 60%, transparent);
        text-transform: uppercase;
        letter-spacing: 0.1em;
    }

    .regime-loader:focus-within {
        outline: 2px solid var(--accent-color, #c5a059);
        outline-offset: 2px;
        border-radius: 4px;
    }
</style>
