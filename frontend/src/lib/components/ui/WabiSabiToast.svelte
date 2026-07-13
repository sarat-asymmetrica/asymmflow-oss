<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { fly, fade } from 'svelte/transition';
  import { cubicOut } from 'svelte/easing';
  import { motionMs } from '../../motion';

  // Motion vocabulary (Wave 10 B2): timing mirrors design-tokens.css --motion-base (200ms).
  // Svelte transitions run in JS and cannot read CSS custom properties or CSS cubic-bezier
  // strings directly, so the numeric value is hardcoded here — keep it equal to --motion-base.
  // cubicOut is the closest svelte/easing analog to --ease-decelerate (fast start, slow settle, no overshoot).
  const TOAST_MOTION_MS = 200;

  interface Props {
    message?: string;
    type?: 'success' | 'warning' | 'danger' | 'info';
    duration?: number;
    showBrush?: boolean;
  }

  let {
    message = '',
    type = 'info',
    duration = 4000,
    showBrush = false
  }: Props = $props();

  const dispatch = createEventDispatcher();

  let dismissTimeout: ReturnType<typeof setTimeout>;

  const toneLabel: Record<typeof type, string> = {
    success: 'Success',
    warning: 'Warning',
    danger: 'Attention',
    info: 'Update'
  };

  function dismiss() {
    dispatch('dismiss');
  }

  onMount(() => {
    if (duration > 0) {
      dismissTimeout = setTimeout(dismiss, duration);
    }
  });

  onDestroy(() => {
    if (dismissTimeout) clearTimeout(dismissTimeout);
  });
</script>

<button
  class="wabi-toast {type}"
  class:with-brush={showBrush}
  in:fly={{ duration: motionMs(TOAST_MOTION_MS), y: -8, easing: cubicOut }}
  out:fade={{ duration: motionMs(TOAST_MOTION_MS), easing: cubicOut }}
  onclick={dismiss}
  aria-label="Dismiss notification: {message}"
  data-testid="{type}-toast"
>
  <div class="toast-content">
    <span class="toast-tone">{toneLabel[type]}</span>
    <span class="toast-message">{message}</span>
  </div>
</button>

<style>
  .wabi-toast {
    position: relative;
    width: min(420px, calc(100vw - 24px));
    min-height: 76px;
    padding: 0;
    border: 1px solid rgba(29, 29, 31, 0.08);
    border-radius: 18px;
    background:
      linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(245, 245, 247, 0.96));
    color: var(--text-primary, #1d1d1f);
    text-align: left;
    cursor: pointer;
    overflow: hidden;
    box-shadow:
      0 14px 32px rgba(15, 23, 42, 0.1),
      0 2px 8px rgba(15, 23, 42, 0.04);
    transition: border-color 160ms ease, background 160ms ease;
  }

  .wabi-toast::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 5px;
    background: rgba(29, 29, 31, 0.18);
  }

  .wabi-toast.with-brush::after {
    display: none;
  }

  .toast-content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    padding: 16px 18px 16px 20px;
    padding-left: 24px;
  }

  .toast-tone {
    font-family: var(--font-display);
    font-size: 11px;
    font-weight: 500;
    line-height: 1.2;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--text-secondary, #6b7280);
  }

  .toast-message {
    display: block;
    max-width: 100%;
    font-family: var(--font-body);
    font-size: 14px;
    font-weight: 400;
    line-height: 1.55;
    letter-spacing: 0;
    color: var(--text-primary, #1d1d1f);
    white-space: normal;
    overflow-wrap: anywhere;
    word-break: break-word;
  }

  .wabi-toast.success::before {
    background: linear-gradient(180deg, rgba(22, 163, 74, 0.92), rgba(34, 197, 94, 0.62));
  }

  .wabi-toast.warning::before {
    background: linear-gradient(180deg, rgba(217, 119, 6, 0.92), rgba(251, 191, 36, 0.68));
  }

  .wabi-toast.danger::before {
    background: linear-gradient(180deg, rgba(220, 38, 38, 0.96), rgba(248, 113, 113, 0.72));
  }

  .wabi-toast.info::before {
    background: linear-gradient(180deg, rgba(2, 132, 199, 0.92), rgba(96, 165, 250, 0.66));
  }

  .wabi-toast.success {
    border-color: rgba(22, 163, 74, 0.18);
  }

  .wabi-toast.warning {
    border-color: rgba(217, 119, 6, 0.2);
  }

  .wabi-toast.danger {
    border-color: rgba(220, 38, 38, 0.2);
  }

  .wabi-toast.info {
    border-color: rgba(2, 132, 199, 0.18);
  }

  @media (max-width: 640px) {
    .wabi-toast {
      width: min(100%, calc(100vw - 24px));
      min-height: 70px;
      border-radius: 16px;
    }

    .toast-content {
      padding: 14px 16px 14px 20px;
    }
  }
</style>
