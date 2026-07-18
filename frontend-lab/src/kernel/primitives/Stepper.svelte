<script lang="ts">
  /* Stepper — a linear state-machine progress rail + forward-transition action
   * buttons. The kernel's answer to the approve→post→pay / draft→submitted→
   * approved lifecycles hand-wired across finance screens (payroll runs,
   * expenses, PO/supplier-invoice approval) and, later, first-run wizards.
   * `steps` declare the ordered lifecycle; `currentKey` marks where a record
   * sits (steps before it read done, after it pending); `actions` declare each
   * forward transition and which states enable it (mirrors the old
   * disabled-unless-status-matches gating exactly). An unknown currentKey
   * leaves every step pending rather than crashing. Owns its layout (L1); the
   * host renders any per-step detail (stats, sub-forms) in the `detail` slot. */
  import type { Snippet } from 'svelte'
  import Button from '../controls/Button.svelte'

  type ActionVariant = 'primary' | 'ghost' | 'danger'

  let {
    steps,
    currentKey,
    actions = [],
    busy = false,
    detail,
  }: {
    steps: { key: string; label: string; description?: string }[]
    currentKey: string
    actions?: {
      key: string
      label: string
      /** currentKey values that enable this action. */
      enabledFrom: string[]
      onAction: () => void | Promise<void>
      variant?: ActionVariant
    }[]
    busy?: boolean
    detail?: Snippet
  } = $props()

  const currentIndex = $derived(steps.findIndex((s) => s.key === currentKey))

  function stateOf(index: number): 'done' | 'active' | 'pending' {
    if (currentIndex === -1) return 'pending'
    if (index < currentIndex) return 'done'
    if (index === currentIndex) return 'active'
    return 'pending'
  }
</script>

<div class="k-stepper">
  <ol class="k-stepper-rail">
    {#each steps as step, i (step.key)}
      {@const state = stateOf(i)}
      <li class="k-step k-step-{state}">
        <span class="k-step-marker" aria-hidden="true">{state === 'done' ? '✓' : i + 1}</span>
        <span class="k-step-text">
          <span class="k-step-label">{step.label}</span>
          {#if step.description}<span class="k-step-desc">{step.description}</span>{/if}
        </span>
      </li>
    {/each}
  </ol>

  {#if detail}
    <div class="k-stepper-detail">{@render detail()}</div>
  {/if}

  {#if actions.length}
    <div class="k-stepper-actions">
      {#each actions as action (action.key)}
        <Button
          variant={action.variant ?? 'primary'}
          disabled={busy || !action.enabledFrom.includes(currentKey)}
          onclick={() => void action.onAction()}
        >{action.label}</Button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .k-stepper {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-stepper-rail {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-md);
    list-style: none;
    margin: 0;
    padding: 0;
    min-width: 0;
  }
  .k-step {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-step-marker {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    flex-shrink: 0;
    border-radius: var(--border-radius-pill);
    border: var(--border-width) solid var(--border);
    font-size: calc(12px * var(--ui-font-scale));
    font-weight: 700;
    background: var(--surface);
    color: var(--text-muted);
  }
  .k-step-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .k-step-label {
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 600;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-step-desc {
    font-size: var(--meta-size);
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-step-done .k-step-marker {
    background: var(--k-tone-success-bg);
    border-color: transparent;
    color: var(--k-tone-success-fg);
  }
  .k-step-done .k-step-label {
    color: var(--text-primary);
  }
  .k-step-active .k-step-marker {
    background: var(--onyx);
    border-color: transparent;
    color: var(--surface);
  }
  .k-step-active .k-step-label {
    color: var(--text-primary);
  }
  .k-stepper-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-sm);
  }
</style>
