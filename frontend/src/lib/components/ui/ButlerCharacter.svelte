<script lang="ts">

    import { onMount, onDestroy } from 'svelte';
    interface Props {
        message?: string;
        sentiment?: string; // calm, alert, happy
    }

    let { message = "", sentiment = "neutral" }: Props = $props();

    let canvas = $state<HTMLCanvasElement | null>(null);
    let ctx: CanvasRenderingContext2D | null = null;
    let animationId: number;
    let time = 0;

    // Sentiment Colors
    const colors = {
        calm: "#15803d",
        alert: "#ef4444",
        happy: "#fbbf24",
        neutral: "#3b82f6"
    };

    let eyeColor = $derived(colors[sentiment] || colors.neutral);

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

            // Draw "Butler" Eye - A complex geometric construct

            // Outer Ring (Breathing)
            const breath = Math.sin(time) * 2;
            ctx.beginPath();
            ctx.arc(cx, cy, 24 + breath, 0, Math.PI * 2);
            ctx.strokeStyle = eyeColor;
            ctx.lineWidth = 2;
            ctx.setLineDash([5, 5]); // Dashed robotic look
            ctx.stroke();
            ctx.setLineDash([]); // Reset

            // Inner Iris (Scanning)
            ctx.beginPath();
            ctx.arc(cx, cy, 12, 0, Math.PI * 2);
            ctx.fillStyle = eyeColor;
            ctx.fill();

            // Pupil (Dilating)
            const dilation = Math.sin(time * 0.5) * 2;
            ctx.beginPath();
            ctx.arc(cx, cy, 4 + dilation, 0, Math.PI * 2);
            ctx.fillStyle = "#fff";
            ctx.fill();

            // Scanning Line
            const scanY = cy + Math.sin(time * 2) * 20;
            ctx.beginPath();
            ctx.moveTo(cx - 30, scanY);
            ctx.lineTo(cx + 30, scanY);
            ctx.strokeStyle = `rgba(255, 255, 255, 0.3)`;
            ctx.lineWidth = 1;
            ctx.stroke();

            animationId = requestAnimationFrame(render);
        };
        render();
    });

    onDestroy(() => cancelAnimationFrame(animationId));
</script>

<div class="bg-[var(--bg-color)] border border-[var(--border-color)] p-6 rounded-lg shadow-sm flex items-center gap-6 h-full relative overflow-hidden group">
    <!-- Canvas for dynamic eye animation -->
    <div class="relative w-20 h-20 flex-shrink-0">
        <canvas bind:this={canvas} width={80} height={80} class="w-full h-full"></canvas>
    </div>

    <div class="z-10 flex-1">
        <div class="text-xs font-mono uppercase tracking-widest opacity-40 mb-1 flex justify-between">
            <span>Butler Insight</span>
            <span class="text-[var(--accent-color)]">{sentiment.toUpperCase()}</span>
        </div>
        <div class="font-serif text-lg leading-relaxed opacity-90 relative">
            <span class="absolute -left-3 -top-1 opacity-20 text-2xl font-serif">"</span>
            {message}
            <span class="opacity-20 text-xl font-serif">"</span>
        </div>
    </div>

    <!-- Holographic Glare -->
    <div class="absolute inset-0 bg-gradient-to-r from-transparent via-[var(--accent-color)]/5 to-transparent -translate-x-full group-hover:translate-x-full transition-transform duration-1000"></div>
</div>
