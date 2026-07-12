<script lang="ts">

    import { onMount, onDestroy } from 'svelte';
    interface Props {
        activeTasks?: number;
        winProbability?: number;
        loading?: boolean;
        error?: string;
    }

    let {
        activeTasks = 0,
        winProbability = 0,
        loading = false,
        error = ''
    }: Props = $props();

    let canvas = $state<HTMLCanvasElement | null>(null);
    let ctx: CanvasRenderingContext2D | null = null;
    let animationId: number;
    let time = 0;

    onMount(() => {
        if (!canvas) return;
        ctx = canvas.getContext('2d');
        const render = () => {
            if (!ctx || !canvas) return;
            const w = canvas.width;
            const h = canvas.height;
            const cx = w / 2;
            const cy = h / 2;

            ctx.clearRect(0, 0, w, h);
            time += 0.05;

            const pulseCount = Math.max(activeTasks, 1);
            for (let i = 0; i < pulseCount; i++) {
                const phase = (time * 0.4) + i * (Math.PI / pulseCount);
                const radius = 30 + (Math.sin(phase) + 1) * 40;
                ctx.beginPath();
                ctx.arc(cx, cy, radius, 0, Math.PI * 2);
                ctx.strokeStyle = i % 2 === 0 ? '#15803d' : '#fbbf24';
                ctx.lineWidth = 1.5;
                ctx.globalAlpha = 0.35;
                ctx.stroke();
            }

            const beat = (Math.sin(time) + 1) / 2;
            const beatRadius = 32 + beat * 8;
            ctx.beginPath();
            ctx.arc(cx, cy, beatRadius, 0, Math.PI * 2);
            ctx.fillStyle = winProbability > 0.6 ? 'rgba(21, 128, 61, 0.35)' : 'rgba(239, 68, 68, 0.25)';
            ctx.fill();

            animationId = requestAnimationFrame(render);
        };

        const resize = () => {
            const rect = canvas.parentElement?.getBoundingClientRect();
            if (!rect) return;
            canvas.width = rect.width;
            canvas.height = rect.height;
        };
        resize();
        window.addEventListener('resize', resize);
        render();

        return () => window.removeEventListener('resize', resize);
    });

    onDestroy(() => cancelAnimationFrame(animationId));
</script>

<div class="service-card">
    <h3>Service Rhythm</h3>
    <canvas bind:this={canvas}></canvas>

    {#if loading}
        <p class="meta">Listening for signals...</p>
    {:else if error}
        <p class="meta error">{error}</p>
    {:else}
        <div class="meta-row">
            <div>
                <div class="meta-label">ACTIVE TASKS</div>
                <div class="meta-value">{activeTasks}</div>
            </div>
            <div>
                <div class="meta-label">WIN PROBABILITY</div>
                <div class="meta-value">{Math.round(winProbability * 100)}%</div>
            </div>
        </div>
    {/if}
</div>

<style>
    .service-card {
        background: rgba(255, 255, 255, 0.4);
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: 10px;
        padding: 1.25rem;
        position: relative;
        min-height: 300px;
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
    }

    h3 {
        margin: 0;
        font-family: var(--font-serif, Georgia, serif);
        font-weight: normal;
        letter-spacing: -0.3px;
        color: var(--color-ink, #1c1c1c);
    }

    canvas {
        width: 100%;
        height: 180px;
        border-radius: 8px;
    }

    .meta {
        font-family: var(--font-mono, 'Courier New', monospace);
        font-size: 0.75rem;
        color: var(--color-ink-light, #57534e);
        margin: 0;
    }

    .meta-row {
        display: flex;
        justify-content: space-between;
        gap: 1.5rem;
        font-family: var(--font-mono, 'Courier New', monospace);
        color: var(--color-ink, #1c1c1c);
    }

    .meta-label {
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--color-ink-light, #57534e);
    }

    .meta-value {
        font-size: 1rem;
    }

    .error {
        color: var(--color-danger, #ef4444);
    }
</style>
