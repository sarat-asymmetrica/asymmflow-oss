<script lang="ts">
  
  
  interface Props {
    /**
   * Wabi-Sabi Skeleton Loader
   * Organic shimmer effect with imperfect edges
   */
    width?: string;
    height?: string;
    rounded?: boolean;
    circle?: boolean;
    lines?: number;
    gap?: string;
  }

  let {
    width = '100%',
    height = '20px',
    rounded = false,
    circle = false,
    lines = 1,
    gap = '8px'
  }: Props = $props();
</script>

{#if lines > 1}
  <div class="skeleton-group" style="gap: {gap};">
    {#each Array(lines) as _, i}
      <div 
        class="skeleton"
        class:rounded
        class:circle
        style="
          width: {i === lines - 1 ? '70%' : width};
          height: {height};
        "
      >
        <div class="shimmer"></div>
      </div>
    {/each}
  </div>
{:else}
  <div 
    class="skeleton"
    class:rounded
    class:circle
    style="width: {width}; height: {circle ? width : height};"
  >
    <div class="shimmer"></div>
  </div>
{/if}

<style>
  .skeleton-group {
    display: flex;
    flex-direction: column;
  }

  .skeleton {
    position: relative;
    overflow: hidden;
    background: rgba(0, 0, 0, 0.06);
    border-radius: 4px;
  }

  .skeleton.rounded {
    border-radius: 8px;
  }

  .skeleton.circle {
    border-radius: 50%;
  }

  .shimmer {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      90deg,
      transparent 0%,
      rgba(255, 255, 255, 0.4) 50%,
      transparent 100%
    );
    animation: shimmer 2s ease-in-out infinite;
    transform: translateX(-100%);
  }

  @keyframes shimmer {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(100%);
    }
  }

  /* Wabi-sabi: slight imperfection in timing */
  .skeleton:nth-child(odd) .shimmer {
    animation-duration: 2.1s;
  }

  .skeleton:nth-child(3n) .shimmer {
    animation-duration: 1.9s;
  }
</style>
