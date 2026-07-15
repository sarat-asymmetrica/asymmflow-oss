<script lang="ts">
  /* AllocationMatchPanel — search candidate documents, multi-add them into an
   * allocation plan, edit each applied amount, and watch a running remainder
   * foot toward zero. A controlled primitive: the caller owns the `allocations`
   * array and every mutation goes back through onAdd/onAmountChange/onRemove;
   * `balanced` is bindable so a Modal footer outside the panel can gate its
   * Confirm without re-deriving the math. The panel calls NO backend — the
   * caller's VM decides what confirm does (split allocation vs receipt-apply
   * loop). Owns its own layout (L1). */
  import { formatMoney } from '../format'
  import {
    type MatchCandidate,
    type AllocationDraft,
    allocationKey,
    totalAllocated,
    remainingToAllocate,
    isFullyAllocated,
    defaultAllocationAmount,
  } from '../allocation'
  import Button from '../controls/Button.svelte'
  import SearchInput from '../controls/SearchInput.svelte'
  import EmptyState from '../controls/EmptyState.svelte'

  let {
    target,
    allocations,
    candidates,
    candidateTypeOptions,
    singleSelectTypes = [],
    tolerance = 0.001,
    loading = false,
    maxResults = 40,
    renderCandidateLabel,
    balanced = $bindable(false),
    confirmLabel,
    onAdd,
    onAmountChange,
    onRemove,
    onConfirm,
  }: {
    target: { amount: number; currency: string; label?: string }
    allocations: AllocationDraft[]
    /** Pre-fetched candidate pool (filtered client-side), OR an async search fn. */
    candidates: MatchCandidate[]
    candidateTypeOptions?: { value: string; label: string }[]
    /** Types that match one-at-a-time: adding replaces the sole allocation. */
    singleSelectTypes?: string[]
    tolerance?: number
    loading?: boolean
    maxResults?: number
    renderCandidateLabel?: (c: MatchCandidate) => string
    balanced?: boolean
    /** When set, the panel renders its own Confirm button; otherwise the caller
     * gates externally via bind:balanced. */
    confirmLabel?: string
    onAdd: (candidate: MatchCandidate, amount: number) => void
    onAmountChange: (key: string, amount: number) => void
    onRemove: (key: string) => void
    onConfirm?: (allocations: AllocationDraft[]) => void
  } = $props()

  let query = $state('')
  let typeFilter = $state('')

  const allocatedKeys = $derived(new Set(allocations.map((a) => a.key)))
  const allocated = $derived(totalAllocated(allocations))
  const remaining = $derived(remainingToAllocate(target.amount, allocations))
  const over = $derived(remaining < -tolerance)

  // Keep the bindable `balanced` in sync with the derived math.
  $effect(() => {
    balanced = isFullyAllocated(target.amount, allocations, tolerance)
  })

  const label = (c: MatchCandidate) => (renderCandidateLabel ? renderCandidateLabel(c) : c.label)

  const visibleCandidates = $derived.by(() => {
    const q = query.trim().toLowerCase()
    const filtered = candidates.filter((c) => {
      if (typeFilter && c.type !== typeFilter) return false
      if (q && !label(c).toLowerCase().includes(q)) return false
      return true
    })
    // Amount-proximity sort: the nearest-sized candidates to the remainder rise
    // first, so the obvious match is at the top (parity with the old screen).
    const r = Math.abs(remaining)
    filtered.sort((a, b) => Math.abs(a.amount - r) - Math.abs(b.amount - r))
    return filtered.slice(0, maxResults)
  })

  function add(c: MatchCandidate) {
    const amount = defaultAllocationAmount(remaining, c.amount)
    onAdd(c, amount)
  }
</script>

<div class="k-amp">
  <div class="k-amp-target">
    <span class="k-amp-target-label">{target.label ?? 'Amount to allocate'}</span>
    <span class="k-amp-target-value">{formatMoney(target.amount, target.currency)}</span>
  </div>

  <!-- Allocation plan -->
  <div class="k-amp-plan">
    {#if allocations.length === 0}
      <span class="k-amp-plan-empty">No documents allocated yet — search below and add matches.</span>
    {:else}
      {#each allocations as a (a.key)}
        <div class="k-amp-alloc">
          <span class="k-amp-alloc-label" title={a.label}>{a.label}</span>
          <div class="k-amp-alloc-amt">
            <input
              class="k-amp-input"
              type="number"
              step="0.001"
              value={a.amount}
              onchange={(e) => onAmountChange(a.key, Number(e.currentTarget.value) || 0)}
            />
            <span class="k-amp-alloc-max">/ {formatMoney(a.maxAmount, target.currency)}</span>
          </div>
          <button class="k-amp-remove" aria-label="Remove allocation" onclick={() => onRemove(a.key)}>×</button>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Running remainder -->
  <div
    class="k-amp-remainder"
    style:background={`var(--k-tone-${balanced ? 'success' : over ? 'danger' : 'neutral'}-bg)`}
    style:color={`var(--k-tone-${balanced ? 'success' : over ? 'danger' : 'neutral'}-fg)`}
  >
    <span class="k-amp-remainder-label">
      {#if balanced}Fully allocated{:else if over}Over-allocated{:else}Remaining{/if}
    </span>
    <span class="k-amp-remainder-value">
      {formatMoney(balanced ? allocated : Math.abs(remaining), target.currency)}
    </span>
  </div>

  <!-- Candidate search -->
  <div class="k-amp-search">
    <SearchInput bind:value={query} placeholder="Search candidate documents…" />
    {#if candidateTypeOptions?.length}
      <select class="k-amp-typesel" bind:value={typeFilter}>
        <option value="">All types</option>
        {#each candidateTypeOptions as t (t.value)}
          <option value={t.value}>{t.label}</option>
        {/each}
      </select>
    {/if}
  </div>

  <div class="k-amp-candidates">
    {#if loading}
      <EmptyState message="Searching candidates…" />
    {:else if visibleCandidates.length === 0}
      <EmptyState message="No matching candidate documents." />
    {:else}
      {#each visibleCandidates as c (c.type + ':' + c.id)}
        {@const isAllocated = allocatedKeys.has(allocationKey(c.type, c.id))}
        {@const single = singleSelectTypes.includes(c.type)}
        <div class="k-amp-cand">
          <span class="k-amp-cand-label" title={label(c)}>{label(c)}</span>
          <span class="k-amp-cand-amt">{formatMoney(c.amount, target.currency)}</span>
          <Button
            variant={isAllocated ? 'ghost' : 'primary'}
            disabled={isAllocated}
            onclick={() => add(c)}
          >{isAllocated ? 'Added' : single ? 'Select' : 'Add'}</Button>
        </div>
      {/each}
    {/if}
  </div>

  {#if confirmLabel}
    <div class="k-amp-confirm">
      <Button variant="primary" disabled={!balanced} onclick={() => onConfirm?.(allocations)}>
        {confirmLabel}
      </Button>
    </div>
  {/if}
</div>

<style>
  .k-amp {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-md);
    min-width: 0;
  }
  .k-amp-target {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-amp-target-label {
    font-size: var(--label-size);
    font-weight: var(--label-weight);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
  }
  .k-amp-target-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(18px * var(--ui-font-scale));
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
  }
  .k-amp-plan {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    min-width: 0;
  }
  .k-amp-plan-empty {
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
  }
  .k-amp-alloc {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-amp-alloc-label {
    flex: 1 1 auto;
    min-width: 0;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-amp-alloc-amt {
    display: flex;
    align-items: baseline;
    gap: var(--k-space-xs);
    flex-shrink: 0;
  }
  .k-amp-alloc-max {
    font-size: var(--meta-size);
    color: var(--text-muted);
    white-space: nowrap;
  }
  .k-amp-input {
    width: 110px;
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(13px * var(--ui-font-scale));
    text-align: right;
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 5px 8px;
    outline: none;
  }
  .k-amp-input:focus {
    border-color: var(--onyx);
  }
  .k-amp-remove {
    width: 24px;
    height: 24px;
    flex-shrink: 0;
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    background: var(--surface);
    color: var(--text-secondary);
    font-size: 15px;
    line-height: 1;
    cursor: pointer;
  }
  .k-amp-remainder {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--k-space-sm);
    padding: var(--k-space-sm) var(--k-space-md);
    border-radius: var(--border-radius);
    min-width: 0;
  }
  .k-amp-remainder-label {
    font-size: calc(13px * var(--ui-font-scale));
    font-weight: 700;
    white-space: nowrap;
  }
  .k-amp-remainder-value {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(14px * var(--ui-font-scale));
    font-weight: 700;
    white-space: nowrap;
  }
  .k-amp-search {
    display: flex;
    gap: var(--k-space-sm);
    align-items: center;
    min-width: 0;
  }
  .k-amp-typesel {
    flex-shrink: 0;
    font-family: var(--font-ui);
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 7px 8px;
    outline: none;
  }
  .k-amp-candidates {
    display: flex;
    flex-direction: column;
    gap: var(--k-space-xs);
    max-height: 260px;
    overflow-y: auto;
    min-width: 0;
  }
  .k-amp-cand {
    display: flex;
    align-items: center;
    gap: var(--k-space-sm);
    min-width: 0;
  }
  .k-amp-cand-label {
    flex: 1 1 auto;
    min-width: 0;
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .k-amp-cand-amt {
    font-family: var(--font-numeric);
    font-feature-settings: var(--font-numeric-features);
    font-size: calc(13px * var(--ui-font-scale));
    color: var(--text-secondary);
    white-space: nowrap;
    flex-shrink: 0;
  }
  .k-amp-confirm {
    display: flex;
    justify-content: flex-end;
  }
</style>
