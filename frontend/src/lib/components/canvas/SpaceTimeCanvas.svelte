<script lang="ts">
    import { onMount } from 'svelte';
    import type { VisualRegime } from '$lib/types/ui';

    interface Props {
        regime?: VisualRegime | null;
    }

    let { regime = null }: Props = $props();

    let canvas: HTMLCanvasElement | null = $state(null);
    let ctx: CanvasRenderingContext2D | null = null;
    let animationId: number;
    let time = 0;
    
    // Starfield for deep space
    const stars: Array<{ x: number; y: number; z: number; size: number }> = [];
    for(let i=0; i<200; i++) {
        stars.push({
            x: Math.random(),
            y: Math.random(),
            z: Math.random() * 2 + 0.1, // Depth
            size: Math.random() * 2
        });
    }

    onMount(() => {
        if (!canvas) return;
        ctx = canvas.getContext('2d');

        const render = () => {
            if (!ctx || !canvas) return;
            const w = canvas.width;
            const h = canvas.height;

            ctx.clearRect(0, 0, w, h);
            time += 0.005;

            // Draw Background Gradient based on Regime
            if (regime) {
                const gradient = ctx.createLinearGradient(0, 0, w, h);
                const c1 = regime.primary_color || '#1c1c1c';
                const c2 = regime.secondary_color || '#2d4a6f';
                gradient.addColorStop(0, hexToRgba(c1, 0.05));
                gradient.addColorStop(1, hexToRgba(c2, 0.1));
                ctx.fillStyle = gradient;
                ctx.fillRect(0, 0, w, h);
            }

            // Draw 3D Starfield / Particles
            ctx.fillStyle = hexToRgba(regime?.secondary_color || '#ffffff', 0.3);

            stars.forEach(star => {
                // Move star
                let x = (star.x - 0.5) * w;
                let y = (star.y - 0.5) * h;

                // Rotation based on time
                const cos = Math.cos(time * 0.1);
                const sin = Math.sin(time * 0.1);
                const rx = x * cos - y * sin;
                const ry = x * sin + y * cos;

                // Perspective projection
                const scale = 200 / (200 + star.z);
                const px = rx * scale + w/2;
                const py = ry * scale + h/2;

                ctx.beginPath();
                ctx.arc(px, py, star.size * scale, 0, Math.PI * 2);
                ctx.fill();
            });

            // Draw Regime-specific geometry (Grid for Cyberpunk, Flows for Fluid, etc.)
            if (regime && regime.geometry?.type === 'FluidPlane') {
                 // Draw fluid lines
                 ctx.beginPath();
                 ctx.strokeStyle = hexToRgba(regime.secondary_color, 0.1);
                 for(let i=0; i<h; i+=50) {
                     ctx.moveTo(0, i);
                     for(let x=0; x<w; x+=20) {
                         ctx.lineTo(x, i + Math.sin(x*0.01 + time + i)*20);
                     }
                 }
                 ctx.stroke();
            }

            animationId = requestAnimationFrame(render);
        };

        const resize = () => {
            canvas.width = window.innerWidth;
            canvas.height = window.innerHeight;
        };
        resize();
        window.addEventListener('resize', resize);
        render();

        return () => {
            window.removeEventListener('resize', resize);
            cancelAnimationFrame(animationId);
        };
    });

    function hexToRgba(hex: string | null | undefined, alpha: number) {
        if (!hex) return `rgba(0,0,0,${alpha})`;
        let c;
        if(/^#([A-Fa-f0-9]{3}){1,2}$/.test(hex)){
            c= hex.substring(1).split('');
            if(c.length== 3){
                c= [c[0], c[0], c[1], c[1], c[2], c[2]];
            }
            c= '0x'+c.join('');
            const num = Number(c);
            return 'rgba('+[(num>>16)&255, (num>>8)&255, num&255].join(',')+','+alpha+')';
        }
        return `rgba(0,0,0,${alpha})`;
    }
</script>

<canvas
    bind:this={canvas}
    class="fixed inset-0 pointer-events-none z-[-1]"
></canvas>
