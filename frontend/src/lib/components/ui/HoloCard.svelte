<script lang="ts">
  interface Props {
    className?: string;
    children?: import('svelte').Snippet;
  }

  let { className = "", children }: Props = $props();
</script>

<div class="holo-card {className}">
  {@render children?.()}
</div>

<style>
  .holo-card {
    position: relative;
    background: rgba(255, 255, 255, 0.6);
    backdrop-filter: blur(8px);
    border: 1px solid rgba(255, 255, 255, 0.4);
    box-shadow:
      0 4px 6px -1px rgba(0, 0, 0, 0.1),
      0 2px 4px -1px rgba(0, 0, 0, 0.06),
      inset 0 0 20px rgba(255, 255, 255, 0.5);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    overflow: hidden;
  }

  .holo-card:hover {
    box-shadow:
      0 10px 15px -3px rgba(0, 0, 0, 0.1),
      0 4px 6px -2px rgba(0, 0, 0, 0.05),
      inset 0 0 30px rgba(255, 255, 255, 0.8);
    transform: translateY(-2px);
  }

  /* Holographic sheen */
  .holo-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: -100%;
    width: 200%;
    height: 100%;
    background: linear-gradient(
      115deg,
      transparent 0%,
      transparent 40%,
      rgba(255, 255, 255, 0.4) 45%,
      rgba(255, 255, 255, 0.0) 50%,
      transparent 100%
    );
    transition: transform 0.6s;
    pointer-events: none;
  }

  .holo-card:hover::before {
    transform: translateX(50%);
  }
</style>
