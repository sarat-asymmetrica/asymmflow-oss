<script lang="ts">
  /**
   * Wabi-Sabi Progress Indicator
   * Ink dots with breathing animation on current step
   */
  import { onMount, onDestroy } from 'svelte';
  import { WOBBLE_K, withAlpha, COLOR } from '$lib/design-system/asymmetrica';

  interface Props {
    current?: number;
    total?: number;
    labels?: string[];
    showLabels?: boolean;
  }

  let {
    current = 0,
    total = 5,
    labels = [],
    showLabels = true
  }: Props = $props();

  let breathT = $state(0);
  let animationId: number;

  function animate() {
    breathT += 0.003;
    animationId = requestAnimationFrame(animate);
  }

  onMount(() => {
    animate();
  });

  onDestroy(() => {
    if (animationId) cancelAnimationFrame(animationId);
  });

  let currentScale = $derived(1 + Math.sin(breathT * 5) * 0.15);
</script>

<div class="wabi-progress" role="progressbar" aria-valuenow={current} aria-valuemax={total}>
  {#each Array(total) as _, i}
    <div class="step" class:completed={i < current} class:active={i === current}>
      <div
        class="dot"
        style={i === current ? `transform: scale(${currentScale})` : ''}
      >
        {#if i < current}
          <span class="check">OK</span>
        {/if}
      </div>
      
      {#if showLabels && labels[i]}
        <span class="label">{labels[i]}</span>
      {/if}
    </div>
    
    {#if i < total - 1}
      <div class="line" class:filled={i < current}></div>
    {/if}
  {/each}
</div>

<style>
  .wabi-progress {
    display: flex;
    align-items: flex-start;
    justify-content: center;
    gap: 0;
  }

  .step {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    border: 1.5px solid var(--color-ink-light, #57534e);
    background: transparent;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s var(--ease-sabi);
    position: relative;
  }

  .step.completed .dot {
    background: var(--color-ink, #1c1c1c);
    border-color: var(--color-ink, #1c1c1c);
  }

  .step.active .dot {
    width: 14px;
    height: 14px;
    border-color: var(--color-ink, #1c1c1c);
    border-width: 2px;
  }

  .check {
    font-size: 8px;
    color: var(--color-paper, #fdfbf7);
    line-height: 1;
  }

  .label {
    font-family: var(--font-data, monospace);
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--color-ink-light, #57534e);
    transition: color 0.3s ease;
    max-width: 60px;
    text-align: center;
  }

  .step.completed .label,
  .step.active .label {
    color: var(--color-ink, #1c1c1c);
  }

  .line {
    width: 40px;
    height: 1px;
    background: var(--color-ink-light, #57534e);
    opacity: 0.3;
    margin-top: 6px;
    transition: all 0.3s ease;
  }

  .line.filled {
    background: var(--color-ink, #1c1c1c);
    opacity: 1;
  }

  /* Responsive */
  @media (max-width: 600px) {
    .line {
      width: 24px;
    }
    
    .label {
      font-size: 8px;
      max-width: 50px;
    }
  }
</style>
