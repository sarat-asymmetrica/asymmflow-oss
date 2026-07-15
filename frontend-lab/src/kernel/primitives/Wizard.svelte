<script lang="ts">
  /* Wizard — a linear multi-step process with local back/next navigation and a
   * per-step content panel. Sibling to Stepper: Stepper is a state-machine RAIL
   * over a PERSISTED record (currentKey = the record's status, actions = gated
   * forward transitions); Wizard owns a LOCAL step pointer the user pages
   * through before anything is committed (OneDrive import's configure → review →
   * run; the first-run SetupWizard). Shares Stepper's numbered done/active/
   * pending rail visual, but adds Back + Next driven by the host, not a status
   * gate. The host renders the CURRENT step's content in the `content` snippet
   * (switching on `currentIndex`); Next is gated by `canAdvance`. Owns its
   * layout (L1). */
  import type { Snippet } from 'svelte'
  import Button from '../controls/Button.svelte'

  let {
    steps,
    currentIndex,
    content,
    onBack,
    onNext,
    canAdvance = true,
    backLabel = 'Back',
    nextLabel = 'Next',
    busy = false,
  }: {
    steps: { key: string; label: string; description?: string }[]
    /** The current step (0-based) — the host owns this local pointer. */
    currentIndex: number
    /** The current step's content (host switches on currentIndex). */
    content: Snippet
    onBack?: () => void
    onNext?: () => void
    /** Gate the Next button (e.g. "at least one valid path entered"). */
    canAdvance?: boolean
    backLabel?: string
    /** Next/finish label — e.g. "Start Scan" / "Import 12 Deals". */
    nextLabel?: string
    busy?: boolean
  } = $props()

  function stateOf(index: number): 'done' | 'active' | 'pending' {
    if (index < currentIndex) return 'done'
    if (index === currentIndex) return 'active'
    return 'pending'
  }
</script>

<div class="k-wizard">
  <ol class="k-wizard-rail">
    {#each steps as step, i (step.key)}
      {@const state = stateOf(i)}
      <li class="k-wstep k-wstep-{state}">
        <span class="k-wstep-marker" aria-hidden="true">{state === 'done' ? '✓' : i + 1}</span>
        <span class="k-wstep-text">
          <span class="k-wstep-label">{step.label}</span>
          {#if step.description}<span class="k-wstep-desc">{step.description}</span>{/if}
        </span>
      </li>
    {/each}
  </ol>

  <div class="k-wizard-content">
    {@render content()}
  </div>

  <div class="k-wizard-nav">
    <Button onclick={() => onBack?.()} disabled={busy || currentIndex === 0}>{backLabel}</Button>
    <Button variant="primary" onclick={() => onNext?.()} disabled={busy || !canAdvance}>
      {busy ? 'Working…' : nextLabel}
    </Button>
  </div>
</div>

<style>
  .k-wizard {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-lg);
    min-width: 0;
  }
  .k-wizard-rail {
    display: flex;
    flex-wrap: wrap;
    gap: var(--k-space-md);
    list-style: none;
    margin: 0;
    padding: 0;
    min-width: 0;
  }
  .k-wstep {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-wstep-marker {
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
  .k-wstep-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .k-wstep-label {
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 600;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-wstep-desc {
    font-size: var(--meta-size);
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-wstep-done .k-wstep-marker {
    background: var(--k-tone-success-bg);
    border-color: transparent;
    color: var(--k-tone-success-fg);
  }
  .k-wstep-done .k-wstep-label {
    color: var(--text-primary);
  }
  .k-wstep-active .k-wstep-marker {
    background: var(--onyx);
    border-color: transparent;
    color: var(--surface);
  }
  .k-wstep-active .k-wstep-label {
    color: var(--text-primary);
  }
  .k-wizard-content {
    min-width: 0;
  }
  .k-wizard-nav {
    display: flex;
    justify-content: space-between;
    gap: var(--k-space-sm);
  }
</style>
