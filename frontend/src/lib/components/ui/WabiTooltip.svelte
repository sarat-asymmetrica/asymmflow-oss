<script lang="ts">
  /**
   * Wabi-Sabi Tooltip
   * Elegant, paper-like tooltips that appear like whispered wisdom
   */
  import { fade } from 'svelte/transition';
  import { motionMs } from '../../motion';
  
  interface Props {
    text: string;
    position?: 'top' | 'bottom' | 'left' | 'right';
    delay?: number;
    children?: import('svelte').Snippet;
  }

  let {
    text,
    position = 'top',
    delay = 300,
    children
  }: Props = $props();
  
  let visible = $state(false);
  let timeout: ReturnType<typeof setTimeout>;
  
  function show() {
    timeout = setTimeout(() => {
      visible = true;
    }, delay);
  }
  
  function hide() {
    clearTimeout(timeout);
    visible = false;
  }
</script>

<div 
  class="tooltip-wrapper"
  onmouseenter={show}
  onmouseleave={hide}
  onfocus={show}
  onblur={hide}
  role="tooltip"
>
  {@render children?.()}
  
  {#if visible && text}
    <div 
      class="tooltip {position}"
      in:fade={{ duration: motionMs(120) }}
      out:fade={{ duration: motionMs(80) }}
    >
      <span class="tooltip-text">{text}</span>
      <div class="tooltip-arrow"></div>
    </div>
  {/if}
</div>

<style>
  .tooltip-wrapper {
    position: relative;
    display: inline-flex;
  }
  
  .tooltip {
    position: absolute;
    z-index: 1000;
    padding: 8px 13px;
    background: #1c1c1c;
    border-radius: 6px;
    white-space: nowrap;
    pointer-events: none;
  }
  
  .tooltip-text {
    font-family: Georgia, serif;
    font-size: 12px;
    color: #fdfbf7;
    line-height: 1.4;
  }
  
  .tooltip-arrow {
    position: absolute;
    width: 8px;
    height: 8px;
    background: #1c1c1c;
    transform: rotate(45deg);
  }
  
  /* Positions */
  .tooltip.top {
    bottom: calc(100% + 8px);
    left: 50%;
    transform: translateX(-50%);
  }
  
  .tooltip.top .tooltip-arrow {
    bottom: -4px;
    left: 50%;
    transform: translateX(-50%) rotate(45deg);
  }
  
  .tooltip.bottom {
    top: calc(100% + 8px);
    left: 50%;
    transform: translateX(-50%);
  }
  
  .tooltip.bottom .tooltip-arrow {
    top: -4px;
    left: 50%;
    transform: translateX(-50%) rotate(45deg);
  }
  
  .tooltip.left {
    right: calc(100% + 8px);
    top: 50%;
    transform: translateY(-50%);
  }
  
  .tooltip.left .tooltip-arrow {
    right: -4px;
    top: 50%;
    transform: translateY(-50%) rotate(45deg);
  }
  
  .tooltip.right {
    left: calc(100% + 8px);
    top: 50%;
    transform: translateY(-50%);
  }
  
  .tooltip.right .tooltip-arrow {
    left: -4px;
    top: 50%;
    transform: translateY(-50%) rotate(45deg);
  }
</style>
