<script lang="ts">
    import { onMount, onDestroy } from 'svelte';

    
    interface Props {
        pipeline?: any;
        winProb?: number;
        loading?: boolean;
        error?: string;
    }

    let {
        pipeline = [],
        winProb = 0,
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
            time += 0.01;

            const totalCount = safePipeline.reduce((acc, stage) => acc + (stage?.count || 0), 0) || 1;
            let startAngle = -Math.PI / 2;

            safePipeline.forEach((stage, idx) => {
                const share = (stage?.count || 0) / totalCount;
                const endAngle = startAngle + share * Math.PI * 2;
                const radius = 80 + idx * 18;

                ctx.beginPath();
                ctx.arc(cx, cy, radius, startAngle, endAngle);
                ctx.strokeStyle = `rgba(27, 28, 28, ${0.25 + idx * 0.15})`;
                ctx.lineWidth = 12;
                ctx.lineCap = 'round';
                ctx.stroke();

                // Marker for stage name
                const labelAngle = startAngle + (endAngle - startAngle) / 2;
                const lx = cx + Math.cos(labelAngle) * (radius + 16);
                const ly = cy + Math.sin(labelAngle) * (radius + 16);
                ctx.fillStyle = 'rgba(87, 83, 78, 0.85)';
                ctx.font = "10px 'Courier New', monospace";
                ctx.textAlign = 'center';
                ctx.fillText(stage?.name || '', lx, ly);

                startAngle = endAngle;
            });

            const pulsate = Math.sin(time * 2) * 4;
            const orbRadius = 50 + pulsate;
            ctx.beginPath();
            ctx.arc(cx, cy, orbRadius, 0, Math.PI * 2);

            let color = '#3b82f6';
            if (winProb > 0.7) color = '#15803d';
            else if (winProb < 0.3) color = '#ef4444';

            const grad = ctx.createRadialGradient(cx, cy, 0, cx, cy, 100);
            grad.addColorStop(0, color);
            grad.addColorStop(1, 'transparent');
            ctx.fillStyle = grad;
            ctx.fill();

            ctx.font = "bold 32px Georgia, serif";
            ctx.fillStyle = '#1c1c1c';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(`${Math.round(winProb * 100)}%`, cx, cy);

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
    // Ensure pipeline is always an array
    let safePipeline = $derived(Array.isArray(pipeline) ? pipeline : []);
</script>

<div class="mandala-card">
    <h3>Opportunity Mandala</h3>
    <canvas bind:this={canvas}></canvas>

    <div class="legend">
        {#if loading}
            <span>Loading pipeline...</span>
        {:else if error}
            <span class="error">{error}</span>
        {:else}
            {#each safePipeline as stage (stage?.name || 'unknown')}
                <div class="legend-row">
                    <span class="label">{stage.name}</span>
                    <span class="value">{stage.count} | {stage.amount?.toLocaleString?.() || '0'} BHD</span>
                </div>
            {/each}
        {/if}
    </div>

    <p class="subtext">Win probability orb reflects orders/offer ratio.</p>
</div>

<style>
    .mandala-card {
        background: rgba(255, 255, 255, 0.4);
        border: 1px solid rgba(0, 0, 0, 0.08);
        border-radius: 10px;
        padding: 1.25rem;
        position: relative;
        min-height: 360px;
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
        height: 220px;
        border-radius: 8px;
    }

    .legend {
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
        font-family: var(--font-mono, 'Courier New', monospace);
        font-size: 0.75rem;
        color: var(--color-ink-light, #57534e);
    }

    .legend-row {
        display: flex;
        justify-content: space-between;
        border-bottom: 1px dashed rgba(0, 0, 0, 0.08);
        padding-bottom: 0.25rem;
    }

    .value {
        color: var(--color-ink, #1c1c1c);
    }

    .subtext {
        margin: 0;
        font-size: 0.7rem;
        color: var(--color-ink-light, #57534e);
        font-family: var(--font-mono, 'Courier New', monospace);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .error {
        color: var(--color-danger, #ef4444);
    }
</style>
