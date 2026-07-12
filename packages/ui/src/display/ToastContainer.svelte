<script lang="ts">
  /**
   * ToastContainer — fixed bottom-right stack, z af-z-toast.
   *
   * Motion regimes (§4e):
   *   Entrance: R1 explore — opacity + translateY(16px) decelerate in
   *   Exit: R3 stabilize — opacity + translateY(8px) accelerate out
   *
   * A11y:
   *   success/info → role="status" (polite)
   *   warning/danger → role="alert" (assertive)
   *
   * Design:
   *   Monochrome-first — the toast is surface + border by default.
   *   Status-colored 3px inline-start border is the ONLY color, per constitution §4c.
   *   Auto-dismiss with pause-on-hover via mouseenter/mouseleave.
   */

  import { toast, type ToastItem, type ToastSeverity } from './toast.svelte.js';

  // Pause-on-hover: track which id is paused
  let paused: Set<string> = $state(new Set());

  // Animated-out ids (R3 exit playing)
  let exiting: Set<string> = $state(new Set());

  function startDismiss(item: ToastItem) {
    if (exiting.has(item.id)) return;
    // Trigger R3 exit animation
    exiting = new Set([...exiting, item.id]);
    setTimeout(() => {
      toast.dismiss(item.id);
      exiting = new Set([...exiting].filter((id) => id !== item.id));
    }, 240); // --af-motion-stabilize-duration
  }

  function scheduleAutoDismiss(item: ToastItem) {
    if (item.duration === 0) return;
    const timer = setTimeout(() => {
      if (!paused.has(item.id)) startDismiss(item);
      // If paused, the mouseleave handler re-schedules
    }, item.duration);
    return timer;
  }

  function onEnter(id: string) {
    paused = new Set([...paused, id]);
  }

  function onLeave(item: ToastItem) {
    paused = new Set([...paused].filter((p) => p !== item.id));
    // Re-kick a short auto-dismiss window after hover ends
    if (item.duration > 0) {
      setTimeout(() => startDismiss(item), 1200);
    }
  }

  // For each new toast, schedule its auto-dismiss
  $effect(() => {
    const q = toast.queue;
    for (const item of q) {
      if (item.duration > 0 && !exiting.has(item.id)) {
        scheduleAutoDismiss(item);
      }
    }
  });

  function roleFor(severity: ToastSeverity): 'status' | 'alert' {
    return severity === 'warning' || severity === 'danger' ? 'alert' : 'status';
  }

  const severityLabel: Record<ToastSeverity, string> = {
    success: 'Success',
    info: 'Information',
    warning: 'Warning',
    danger: 'Error',
  };
</script>

<div class="af-toast-region" aria-label="Notifications">
  {#each toast.queue as item (item.id)}
    <div
      class="af-toast af-toast--{item.severity}"
      class:af-toast--exiting={exiting.has(item.id)}
      role={roleFor(item.severity)}
      aria-live={roleFor(item.severity) === 'alert' ? 'assertive' : 'polite'}
      aria-atomic="true"
      onmouseenter={() => onEnter(item.id)}
      onmouseleave={() => onLeave(item)}
    >
      <span class="af-sr-only">{severityLabel[item.severity]}:</span>
      <span class="af-toast__message">{item.message}</span>
      <button
        class="af-toast__close"
        aria-label="Dismiss notification"
        onclick={() => startDismiss(item)}
      >
        <!-- Minimal × mark via SVG, no emoji, no icon lib dependency -->
        <svg viewBox="0 0 12 12" width="12" height="12" fill="none" aria-hidden="true">
          <path d="M2 2l8 8M10 2l-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
  {/each}
</div>

<style>
  /* Fixed stack — bottom-right, above everything */
  .af-toast-region {
    position: fixed;
    inset-block-end: var(--af-space-4);
    inset-inline-end: var(--af-space-4);
    z-index: var(--af-z-toast);
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    align-items: flex-end;
    pointer-events: none;
  }

  /* Individual toast */
  .af-toast {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    min-width: 280px;
    max-width: 420px;
    padding: var(--af-space-3) var(--af-space-4);
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    box-shadow: var(--af-shadow-overlay);

    /* Status accent: 3px inline-start border only — color spent sparingly */
    border-inline-start: 3px solid var(--af-border-strong);

    /* R1 entrance — explore: opacity + translateY rise */
    animation: af-toast-in var(--af-motion-explore-duration) var(--af-motion-explore-ease) both;
  }

  /* Per-severity accent colors */
  .af-toast--success { border-inline-start-color: var(--af-success); }
  .af-toast--info    { border-inline-start-color: var(--af-info); }
  .af-toast--warning { border-inline-start-color: var(--af-warning); }
  .af-toast--danger  { border-inline-start-color: var(--af-danger); }

  /* R3 exit — stabilize: accelerate out */
  .af-toast--exiting {
    animation: af-toast-out var(--af-motion-stabilize-duration) var(--af-motion-stabilize-ease) both;
    pointer-events: none;
  }

  @keyframes af-toast-in {
    from {
      opacity: 0;
      transform: translateY(16px) scale(0.97);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @keyframes af-toast-out {
    to {
      opacity: 0;
      transform: translateY(8px) scale(0.97);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .af-toast,
    .af-toast--exiting {
      animation: none;
    }
    .af-toast--exiting {
      opacity: 0;
    }
  }

  .af-toast__message {
    flex: 1;
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
    color: var(--af-text);
    line-height: var(--af-leading-base);
  }

  .af-toast__close {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: none;
    background: transparent;
    color: var(--af-text-muted);
    cursor: pointer;
    border-radius: var(--af-radius-sm);
    transition:
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .af-toast__close:hover {
    color: var(--af-text);
    background: var(--af-tint);
  }

  .af-toast__close:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 1px;
  }

  /* Touch devices: expand the icon button to the WCAG 2.5.5 tap floor.
     Square dismiss control — both axes get the floor. Desktop sizing unchanged. */
  @media (pointer: coarse) {
    .af-toast__close {
      min-width: var(--af-tap-min);
      min-height: var(--af-tap-min);
    }
  }
</style>
