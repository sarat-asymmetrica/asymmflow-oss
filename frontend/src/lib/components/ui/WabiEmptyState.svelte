<script lang="ts">
  /**
   * Wabi-Sabi Empty State
   * Beautiful, calming placeholder when there's no data
   * "Emptiness is not nothing, it is everything waiting to happen"
   */
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { motionMs } from '../../motion';
  
  
  import { createEventDispatcher } from 'svelte';
  interface Props {
    title?: string;
    message?: string;
    icon?: 'garden' | 'mandala' | 'scroll' | 'inbox';
    action?: string;
    actionLabel?: string;
  }

  let {
    title = 'Nothing here yet',
    message = 'This space is waiting for you to fill it.',
    icon = 'garden',
    action = '',
    actionLabel = ''
  }: Props = $props();
  const dispatch = createEventDispatcher();
  
  let canvas: HTMLCanvasElement = $state();
  let ctx: CanvasRenderingContext2D | null;
  let animationId: number;
  let t = 0;
  
  // Zen garden animation - rocks and raked sand
  function drawGarden() {
    if (!ctx || !canvas) return;
    t += 0.003;
    
    const w = canvas.width;
    const h = canvas.height;
    
    ctx.clearRect(0, 0, w, h);
    
    // Raked sand lines
    ctx.strokeStyle = 'rgba(28, 28, 28, 0.08)';
    ctx.lineWidth = 1;
    
    for (let i = 0; i < 8; i++) {
      const y = 30 + i * 15;
      ctx.beginPath();
      ctx.moveTo(20, y);
      
      for (let x = 20; x < w - 20; x += 5) {
        const wave = Math.sin(x * 0.02 + t + i * 0.5) * 3;
        ctx.lineTo(x, y + wave);
      }
      ctx.stroke();
    }
    
    // Three stones (asymmetric placement)
    ctx.fillStyle = 'rgba(28, 28, 28, 0.15)';
    
    // Large stone
    ctx.beginPath();
    ctx.ellipse(w * 0.3, h * 0.6, 25, 18, 0.2, 0, Math.PI * 2);
    ctx.fill();
    
    // Medium stone
    ctx.beginPath();
    ctx.ellipse(w * 0.6, h * 0.5, 15, 12, -0.3, 0, Math.PI * 2);
    ctx.fill();
    
    // Small stone
    ctx.beginPath();
    ctx.ellipse(w * 0.75, h * 0.65, 10, 8, 0.1, 0, Math.PI * 2);
    ctx.fill();
    
    animationId = requestAnimationFrame(drawGarden);
  }
  
  // Mandala animation - rotating circles
  function drawMandala() {
    if (!ctx || !canvas) return;
    t += 0.005;
    
    const w = canvas.width;
    const h = canvas.height;
    const cx = w / 2;
    const cy = h / 2;
    
    ctx.clearRect(0, 0, w, h);
    
    // Concentric circles
    for (let i = 0; i < 4; i++) {
      const r = 20 + i * 20;
      const wobble = Math.sin(t + i) * 2;
      
      ctx.beginPath();
      ctx.arc(cx, cy, r + wobble, 0, Math.PI * 2);
      ctx.strokeStyle = `rgba(28, 28, 28, ${0.15 - i * 0.03})`;
      ctx.lineWidth = 1;
      ctx.stroke();
    }
    
    // Rotating dots
    for (let i = 0; i < 8; i++) {
      const angle = (i / 8) * Math.PI * 2 + t;
      const r = 50;
      const x = cx + Math.cos(angle) * r;
      const y = cy + Math.sin(angle) * r;
      
      ctx.beginPath();
      ctx.arc(x, y, 3, 0, Math.PI * 2);
      ctx.fillStyle = 'rgba(28, 28, 28, 0.2)';
      ctx.fill();
    }
    
    animationId = requestAnimationFrame(drawMandala);
  }
  
  // Scroll animation - unfurling paper
  function drawScroll() {
    if (!ctx || !canvas) return;
    t += 0.004;
    
    const w = canvas.width;
    const h = canvas.height;
    
    ctx.clearRect(0, 0, w, h);
    
    // Scroll body
    ctx.fillStyle = 'rgba(253, 251, 247, 0.8)';
    ctx.strokeStyle = 'rgba(28, 28, 28, 0.15)';
    ctx.lineWidth = 1;
    
    const scrollWidth = 100;
    const scrollHeight = 80;
    const x = (w - scrollWidth) / 2;
    const y = (h - scrollHeight) / 2;
    
    // Main scroll
    ctx.beginPath();
    ctx.roundRect(x, y, scrollWidth, scrollHeight, 4);
    ctx.fill();
    ctx.stroke();
    
    // Text lines
    ctx.strokeStyle = 'rgba(28, 28, 28, 0.1)';
    for (let i = 0; i < 4; i++) {
      const lineY = y + 15 + i * 15;
      const lineWidth = 60 + Math.sin(t + i) * 10;
      ctx.beginPath();
      ctx.moveTo(x + 20, lineY);
      ctx.lineTo(x + 20 + lineWidth, lineY);
      ctx.stroke();
    }
    
    animationId = requestAnimationFrame(drawScroll);
  }
  
  // Inbox animation - floating envelopes
  function drawInbox() {
    if (!ctx || !canvas) return;
    t += 0.003;
    
    const w = canvas.width;
    const h = canvas.height;
    
    ctx.clearRect(0, 0, w, h);
    
    // Inbox tray
    ctx.strokeStyle = 'rgba(28, 28, 28, 0.15)';
    ctx.lineWidth = 1.5;
    
    ctx.beginPath();
    ctx.moveTo(w * 0.25, h * 0.7);
    ctx.lineTo(w * 0.25, h * 0.85);
    ctx.lineTo(w * 0.75, h * 0.85);
    ctx.lineTo(w * 0.75, h * 0.7);
    ctx.stroke();
    
    // Empty indicator - gentle pulse
    const pulseR = 15 + Math.sin(t * 2) * 3;
    ctx.beginPath();
    ctx.arc(w / 2, h * 0.5, pulseR, 0, Math.PI * 2);
    ctx.strokeStyle = `rgba(28, 28, 28, ${0.1 + Math.sin(t * 2) * 0.05})`;
    ctx.stroke();
    
    animationId = requestAnimationFrame(drawInbox);
  }
  
  onMount(() => {
    if (canvas) {
      ctx = canvas.getContext('2d');
      canvas.width = 200;
      canvas.height = 150;
      
      const animations = { garden: drawGarden, mandala: drawMandala, scroll: drawScroll, inbox: drawInbox };
      animations[icon]();
    }
  });
  
  onDestroy(() => {
    if (animationId) cancelAnimationFrame(animationId);
  });
</script>

<div class="empty-state" in:fade={{ duration: motionMs(400) }}>
  <canvas bind:this={canvas} class="empty-canvas"></canvas>
  
  <h3 class="empty-title">{title}</h3>
  <p class="empty-message">{message}</p>
  
  {#if action && actionLabel}
    <button class="empty-action" onclick={() => dispatch('action')}>
      {actionLabel}
    </button>
  {/if}
</div>

<style>
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 34px;
    text-align: center;
    min-height: 250px;
  }
  
  .empty-canvas {
    margin-bottom: 21px;
    opacity: 0.8;
  }
  
  .empty-title {
    font-family: Georgia, serif;
    font-size: 18px;
    font-weight: normal;
    color: #1c1c1c;
    margin: 0 0 8px;
  }
  
  .empty-message {
    font-family: Georgia, serif;
    font-size: 14px;
    color: #57534e;
    margin: 0 0 21px;
    max-width: 280px;
    line-height: 1.5;
  }
  
  .empty-action {
    padding: 10px 21px;
    background: transparent;
    border: 1px solid #1c1c1c;
    border-radius: 6px;
    font-family: Georgia, serif;
    font-size: 13px;
    color: #1c1c1c;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .empty-action:hover {
    background: #1c1c1c;
    color: #fdfbf7;
  }
</style>
