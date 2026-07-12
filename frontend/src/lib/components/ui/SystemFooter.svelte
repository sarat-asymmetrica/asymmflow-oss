<script lang="ts">
  interface Props {
    version?: string;
    health?: string;
    lastSync?: string;
    user?: string;
    isNavCollapsed?: boolean;
  }

  let {
    version = "v2.0 Moonshot",
    health = "Nominal",
    lastSync = "Just now",
    user = "Commander",
    isNavCollapsed = false
  }: Props = $props();

  // Reactive positioning based on nav collapse state
  let leftPosition = $derived(isNavCollapsed ? '5rem' : '16rem'); // 80px collapsed, 256px expanded
</script>

<footer class="system-footer" style="--footer-left: {leftPosition}">
  <div class="footer-left">
    <!-- Health Status with Green Dot -->
    <span class="status-group">
      <span class="status-dot"></span>
      <span class="status-text">{health}</span>
    </span>

    <span class="separator">•</span>

    <!-- Last Sync -->
    <span class="status-text">Synced: {lastSync}</span>
  </div>

  <div class="footer-right">
    <!-- Version -->
    <span class="status-text">{version}</span>

    <span class="separator">•</span>

    <!-- User Info -->
    <div class="user-group">
      <span class="status-text">{user}</span>
      <div class="user-avatar">C</div>
    </div>
  </div>
</footer>

<style>
  .system-footer {
    position: fixed;
    bottom: 0;
    left: var(--footer-left);
    right: 0;
    transition: left var(--duration-normal) var(--ease-wabi-sabi);
    height: 2rem; /* 32px */
    background-color: var(--color-paper, #fdfbf7);
    border-top: 1px solid var(--color-border, #e5e0d8);
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-left: 1.5rem;
    padding-right: 1.5rem;
    font-family: var(--font-mono, 'Courier Prime', monospace);
    font-size: 10px;
    color: var(--color-stone, #78716c);
    z-index: 10;
  }

  .footer-left,
  .footer-right {
    display: flex;
    align-items: center;
    gap: 0.75rem; /* 12px */
  }

  .status-group {
    display: flex;
    align-items: center;
    gap: 0.375rem; /* 6px */
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: var(--color-safe, #15803d);
    box-shadow: 0 0 4px rgba(21, 128, 61, 0.5);
    animation: pulse-glow 2s ease-in-out infinite;
  }

  @keyframes pulse-glow {
    0%, 100% {
      opacity: 1;
      box-shadow: 0 0 4px rgba(21, 128, 61, 0.5);
    }
    50% {
      opacity: 0.7;
      box-shadow: 0 0 8px rgba(21, 128, 61, 0.7);
    }
  }

  .status-text {
    color: var(--color-ink-light, #57534e);
    white-space: nowrap;
  }

  .separator {
    color: var(--color-stone, #78716c);
    opacity: 0.5;
    padding: 0 0.25rem; /* 4px on each side for breathing room */
  }

  .user-group {
    display: flex;
    align-items: center;
    gap: 0.5rem; /* 8px */
    padding-left: 0.75rem; /* 12px */
    border-left: 1px solid var(--color-border, #e5e0d8);
  }

  .user-avatar {
    width: 1rem; /* 16px */
    height: 1rem; /* 16px */
    border-radius: 50%;
    background-color: var(--color-stone, #78716c);
    color: var(--color-paper, #fdfbf7);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 8px;
    font-weight: bold;
  }
</style>
