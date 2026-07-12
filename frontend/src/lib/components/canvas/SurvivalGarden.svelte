<script lang="ts">
    import { onMount, onDestroy } from "svelte";

    interface Props {
        runwayMonths?: number;
        burnRate?: number;
        cashBalance?: number;
        loading?: boolean;
        error?: string;
    }

    let {
        runwayMonths = 0,
        burnRate = 0,
        cashBalance = 0,
        loading = false,
        error = ""
    }: Props = $props();

    let canvas = $state<HTMLCanvasElement | null>(null);
    let ctx: CanvasRenderingContext2D | null = null;
    let animationId: number;
    let waveOffset = 0;

    onMount(() => {
        if (!canvas) return;
        ctx = canvas.getContext("2d");

        const resize = () => {
            canvas.width = canvas.offsetWidth;
            canvas.height = canvas.offsetHeight;
        };

        const render = () => {
            if (!ctx || !canvas) return;

            const w = canvas.width;
            const h = canvas.height;

            ctx.clearRect(0, 0, w, h);

            // Water level based on runway (6 months = 50% height)
            const safeRunway = Math.max(runwayMonths, 0);
            const waterLevel = Math.min(safeRunway / 12, 1.0);

            // Determine color based on runway
            let waterColor = "#15803d"; // Safe green
            if (safeRunway < 2)
                waterColor = "#ef4444"; // Danger red
            else if (safeRunway < 4) waterColor = "#fbbf24"; // Warning gold

            // Draw Water with wave
            ctx.fillStyle = waterColor;
            ctx.globalAlpha = 0.6;

            ctx.beginPath();
            ctx.moveTo(0, h);

            for (let x = 0; x <= w; x += 10) {
                const y =
                    h * (1 - waterLevel) + Math.sin(x * 0.02 + waveOffset) * 5;
                ctx.lineTo(x, y);
            }

            ctx.lineTo(w, h);
            ctx.closePath();
            ctx.fill();

            // Draw "Stones" (Expenses) at the bottom
            ctx.fillStyle = "#475569";
            ctx.globalAlpha = 1;
            ctx.beginPath();
            ctx.arc(w * 0.2, h - 10, 20, 0, Math.PI * 2);
            ctx.arc(w * 0.5, h - 15, 25, 0, Math.PI * 2);
            ctx.arc(w * 0.8, h - 8, 15, 0, Math.PI * 2);
            ctx.fill();

            waveOffset += 0.05;
            animationId = requestAnimationFrame(render);
        };

        resize();
        window.addEventListener("resize", resize);
        render();

        return () => window.removeEventListener("resize", resize);
    });

    onDestroy(() => {
        if (animationId) cancelAnimationFrame(animationId);
    });
</script>

<div class="survival-garden">
    <canvas bind:this={canvas}></canvas>

    <div class="metrics">
        <div class="metric-left">
            <span class="metric-big">{(runwayMonths ?? 0).toFixed(1)}</span>
            <span class="metric-label">MONTHS REMAINING</span>
        </div>
        <div class="metric-right">
            <span class="metric-label">BURN RATE</span>
            <span class="metric-value"
                >{(burnRate ?? 0).toLocaleString()} BHD</span
            >
            <span class="metric-label">CASH</span>
            <span class="metric-value"
                >{(cashBalance ?? 0).toLocaleString()} BHD</span
            >
        </div>
    </div>

    {#if loading}
        <p class="legend">Loading survival metrics...</p>
    {:else if error}
        <p class="legend danger">{error}</p>
    {:else}
        <p class="legend">
            Water level represents cash.<br />
            <span class="danger">Red</span> = Parched (&lt; 2 months).<br />
            <span class="safe">Green</span> = Flowing (&gt; 6 months).
        </p>
    {/if}
</div>

<style>
    .survival-garden {
        display: flex;
        flex-direction: column;
        height: 100%;
    }

    canvas {
        width: 100%;
        height: 180px;
        margin-bottom: 1.25rem;
    }

    .metrics {
        display: flex;
        justify-content: space-between;
        align-items: flex-end;
        margin-bottom: 1.25rem;
    }

    .metric-left {
        display: flex;
        flex-direction: column;
    }

    .metric-big {
        font-size: 2.25rem;
        font-weight: bold;
        font-family: var(--font-serif, Georgia, serif);
        color: var(--color-ink, #1c1c1c);
        line-height: 1;
    }

    .metric-label {
        font-family: var(--font-mono, "Courier New", monospace);
        font-size: 0.6rem;
        color: var(--color-ink-light, #57534e);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .metric-right {
        text-align: right;
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .metric-value {
        font-family: var(--font-mono, "Courier New", monospace);
        font-size: 0.9rem;
        color: var(--color-ink, #1c1c1c);
    }

    .legend {
        font-size: 0.75rem;
        line-height: 1.6;
        color: var(--color-ink-light, #57534e);
        margin: 0;
        margin-top: auto;
    }

    .legend .danger {
        color: var(--color-danger, #ef4444);
    }

    .legend .safe {
        color: var(--color-safe, #15803d);
    }
</style>
