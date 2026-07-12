<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { WOBBLE_K, noise, withAlpha, COLOR, BREATH } from '$lib/design-system/asymmetrica';

  interface Props {
    size?: 'sm' | 'md' | 'lg';
    color?: string;
    tempo?: 'meditative' | 'calm' | 'alert';
  }

  let { size = 'md', color = COLOR.ink, tempo = 'calm' }: Props = $props();

  let canvas: HTMLCanvasElement = $state();
  let ctx: CanvasRenderingContext2D | null;
  let animationId: number;
  let breathT = 0;

  const sizes = { sm: 24, md: 40, lg: 64 };
  const tempos = {
    meditative: 0.002,
    calm: 0.005,
    alert: 0.015,
  };

  function drawWabiCircle(cx: number, cy: number, radius: number, wobble: number = 0.03) {
    if (!ctx) return;
    
    ctx.beginPath();
    const strokes = 60;

    for (let i = 0; i <= strokes; i++) {
      const theta = (i / strokes) * Math.PI * 2;
      const r = radius + Math.sin(theta * WOBBLE_K + breathT * 3) * wobble * radius;

      const x = cx + r * Math.cos(theta);
      const y = cy + r * Math.sin(theta);

      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }

    ctx.strokeStyle = withAlpha(color, 0.6);
    ctx.lineWidth = size === 'sm' ? 1.5 : size === 'md' ? 2 : 3;
    ctx.stroke();
  }

  function animate() {
    if (!ctx || !canvas) return;
    
    breathT += tempos[tempo];
    
    // Clear
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const cx = canvas.width / 2;
    const cy = canvas.height / 2;
    const baseR = (sizes[size] / 2) * 0.7;

    // Breathing radius
    const r = baseR + Math.sin(breathT) * (baseR * 0.15);

    // Draw imperfect circle
    drawWabiCircle(cx, cy, r, 0.04);

    // Inner dot (breathing)
    const dotR = 2 + Math.sin(breathT * 2) * 1;
    ctx.beginPath();
    ctx.arc(cx, cy, dotR, 0, Math.PI * 2);
    ctx.fillStyle = withAlpha(color, 0.4);
    ctx.fill();

    animationId = requestAnimationFrame(animate);
  }

  onMount(() => {
    if (canvas) {
      ctx = canvas.getContext('2d');
      canvas.width = sizes[size];
      canvas.height = sizes[size];
      animate();
    }
  });

  onDestroy(() => {
    if (animationId) cancelAnimationFrame(animationId);
  });
</script>

<div class="wabi-spinner" style="width: {sizes[size]}px; height: {sizes[size]}px;">
  <canvas bind:this={canvas}></canvas>
</div>

<style>
  .wabi-spinner {
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  canvas {
    display: block;
  }
</style>
