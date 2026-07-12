<script lang="ts">
  import Card from '$lib/components/ui/Card.svelte';
  import { formatDate, formatCurrency } from './customerFormatters';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    profile: main.CustomerFullProfile;
    openRFQ: (rfq: any) => void;
  }

  let { profile, openRFQ }: Props = $props();
</script>

<Card padding="md">
  <h3 class="section-title">RFQ HISTORY</h3>
  <div class="rfq-stats">
    <span>Floated: {profile.rfqs_floated}</span>
    <span>Won: {profile.rfqs_won}</span>
    <span>Win Rate: {profile.win_rate?.toFixed(0) || 0}%</span>
  </div>
  {#if profile.recent_rfqs?.length > 0}
    <div class="list">
      {#each profile.recent_rfqs as rfq}
        <div
          class="list-item"
          role="button"
          tabindex="0"
          onclick={() => openRFQ(rfq)}
          onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), openRFQ(rfq))}
        >
          <div class="item-main">
            <span class="item-code">{rfq.project}</span>
            <span class="item-date">{formatDate(rfq.created_at)}</span>
          </div>
          <div class="item-meta">
            <span class="item-status">{rfq.status}</span>
            <span class="item-amount">{formatCurrency(rfq.value)} BHD</span>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <p class="empty-text">No RFQs found</p>
  {/if}
</Card>

<style>
  .section-title { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 16px 0; }
  .empty-text { font-size: 13px; color: var(--text-muted); font-style: italic; }
  .rfq-stats { display: flex; gap: 24px; margin-bottom: 16px; font-size: 14px; color: var(--text-secondary); }

  .list { display: flex; flex-direction: column; gap: 8px; }
  .list-item { display: flex; justify-content: space-between; align-items: center; padding: 12px; background: var(--surface-elevated); border-radius: 6px; transition: all var(--transition-fast); cursor: pointer; }
  .list-item:hover { background: var(--surface-hover); }
  .list-item:focus-visible { outline: 2px solid var(--brand-indigo, #6366F1); outline-offset: -2px; }
  .item-main { display: flex; flex-direction: column; gap: 2px; }
  .item-code { font-weight: 600; font-family: 'JetBrains Mono', monospace; color: var(--brand-indigo); }
  .item-date { font-size: 12px; color: var(--text-muted); }
  .item-meta { display: flex; align-items: center; gap: 16px; }
  .item-status { font-size: 12px; padding: 2px 8px; background: var(--surface); border-radius: 4px; }
  .item-amount { font-weight: 600; font-family: 'JetBrains Mono', monospace; }
</style>
