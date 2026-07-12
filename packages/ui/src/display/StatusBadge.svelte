<script lang="ts">
  /**
   * StatusBadge — constitution §4c compliant.
   *
   * Default emphasis is MONOCHROME: shape + weight communicate status, not color.
   * Color is a scarce resource, spent only when `emphasis="color"` is explicitly set.
   *
   * Statuses:
   *   done      — filled dot + inline check mark
   *   pending   — hollow ring
   *   attention — filled triangle (warning shape)
   *   failed    — filled stop / x mark
   */

  export type StatusKind = 'done' | 'pending' | 'attention' | 'failed';
  export type StatusEmphasis = 'mono' | 'color';

  export interface StatusBadgeProps {
    status: StatusKind;
    /** 'mono' (default) = monochrome iconography per constitution §4c */
    emphasis?: StatusEmphasis;
    size?: 'sm' | 'md';
    /** Override the displayed label; defaults to the status name */
    label?: string;
    [key: string]: unknown;
  }

  let {
    status,
    emphasis = 'mono',
    size = 'md',
    label,
    ...restProps
  }: StatusBadgeProps = $props();

  const labels: Record<StatusKind, string> = {
    done: 'Done',
    pending: 'Pending',
    attention: 'Attention',
    failed: 'Failed',
  };

  const displayLabel = $derived(label ?? labels[status]);
</script>

<span
  class="af-status af-status--{status} af-status--{emphasis} af-status--{size}"
  role="status"
  aria-label={displayLabel}
  {...restProps}
>
  <!-- Indicator mark — SVG keeps sizing predictable and crisp -->
  <span class="af-status__mark" aria-hidden="true">
    {#if status === 'done'}
      <!-- Filled dot with check mask -->
      <svg viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="6" cy="6" r="5.5" fill="currentColor"/>
        <path
          d="M3.5 6l1.8 1.8 3.2-3.2"
          stroke="var(--af-surface)"
          stroke-width="1.4"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    {:else if status === 'pending'}
      <!-- Hollow ring -->
      <svg viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="6" cy="6" r="4.75" stroke="currentColor" stroke-width="1.5"/>
      </svg>
    {:else if status === 'attention'}
      <!-- Filled triangle — attention shape -->
      <svg viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path
          d="M6 1.5L11 10.5H1L6 1.5Z"
          fill="currentColor"
        />
        <path
          d="M6 5v2.5"
          stroke="var(--af-surface)"
          stroke-width="1.3"
          stroke-linecap="round"
        />
        <circle cx="6" cy="9" r="0.65" fill="var(--af-surface)"/>
      </svg>
    {:else}
      <!-- failed — filled circle with X -->
      <svg viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="6" cy="6" r="5.5" fill="currentColor"/>
        <path
          d="M4 4l4 4M8 4l-4 4"
          stroke="var(--af-surface)"
          stroke-width="1.4"
          stroke-linecap="round"
        />
      </svg>
    {/if}
  </span>
  <span class="af-status__text">{displayLabel}</span>
</span>

<style>
  .af-status {
    display: inline-flex;
    align-items: center;
    gap: var(--af-space-1);
    font-family: var(--af-font-body);
    font-weight: var(--af-weight-semibold);
    letter-spacing: 0.02em;
    border-radius: var(--af-radius-pill);
    line-height: 1;
    /* No border by default — monochrome relies on icon + weight alone */
  }

  /* Sizes */
  .af-status--sm {
    font-size: var(--af-text-xs);
    padding: var(--af-space-1) var(--af-space-2);
    min-height: 20px;
  }
  .af-status--sm .af-status__mark {
    width: 10px;
    height: 10px;
  }

  .af-status--md {
    font-size: calc(var(--af-text-xs) * 1.09);
    padding: calc(var(--af-space-1) + 1px) var(--af-space-3);
    min-height: 24px;
  }
  .af-status--md .af-status__mark {
    width: 12px;
    height: 12px;
  }

  .af-status__mark {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .af-status__mark svg {
    width: 100%;
    height: 100%;
    display: block;
  }

  /* ===== MONO (default) — constitution §4c: monochrome is the default ===== */
  .af-status--mono {
    background: var(--af-surface-raised);
    color: var(--af-text-secondary);
    border: 1px solid var(--af-border);
  }

  /* failed gets weight 600 bump to signal severity without color */
  .af-status--mono.af-status--failed {
    color: var(--af-text);
    font-weight: var(--af-weight-bold);
    border-color: var(--af-border-strong);
  }

  /* ===== COLOR — explicit opt-in only, spent rarely ===== */
  .af-status--color.af-status--done {
    background: var(--af-success-tint);
    color: var(--af-success);
    border: 1px solid var(--af-success-tint);
  }

  .af-status--color.af-status--pending {
    background: var(--af-surface-raised);
    color: var(--af-text-secondary);
    border: 1px solid var(--af-border);
  }

  .af-status--color.af-status--attention {
    background: var(--af-warning-tint);
    color: var(--af-warning);
    border: 1px solid var(--af-warning-tint);
  }

  .af-status--color.af-status--failed {
    background: var(--af-danger-tint);
    color: var(--af-danger);
    border: 1px solid var(--af-danger-tint);
    font-weight: var(--af-weight-bold);
  }
</style>
