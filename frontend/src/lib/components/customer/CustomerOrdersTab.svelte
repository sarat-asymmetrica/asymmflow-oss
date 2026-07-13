<script lang="ts">
  import Card from '$lib/components/ui/Card.svelte';
  import DealTimeline from '../DealTimeline.svelte';
  import { formatDate, formatCurrency } from './customerFormatters';
  import { GetDealTimelineByOrderNumber } from '../../../../wailsjs/go/main/App';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    profile: main.CustomerFullProfile;
    openOrder: () => void;
  }

  let { profile, openOrder }: Props = $props();

  // OrderSummary rows here carry order_number/date/status/total only - no id
  // (recon: A3, Wave 10) - so the deal-spine timeline is resolved by serial
  // via GetDealTimelineByOrderNumber rather than a prop-plumbed id. Toggle
  // reveal keeps this to ONE backend call per row, only on demand.
  let expandedOrderNumber: string | null = $state(null);
  let expandedOrderId: string | null = $state(null);
  let resolvingOrderNumber: string | null = $state(null);

  async function toggleTimeline(orderNumber: string) {
    if (expandedOrderNumber === orderNumber) {
      expandedOrderNumber = null;
      expandedOrderId = null;
      return;
    }
    resolvingOrderNumber = orderNumber;
    try {
      const timeline = await GetDealTimelineByOrderNumber(orderNumber);
      expandedOrderId = timeline.order_id;
      expandedOrderNumber = orderNumber;
    } catch (e) {
      // Honest failure - no timeline to show, no fabricated data.
      expandedOrderNumber = null;
      expandedOrderId = null;
    } finally {
      resolvingOrderNumber = null;
    }
  }
</script>

<Card padding="md">
  <h3 class="section-title">RECENT ORDERS</h3>
  {#if profile.recent_orders?.length > 0}
    <div class="list">
      {#each profile.recent_orders as order}
        <div class="list-wrap">
          <div
            class="list-item"
            role="button"
            tabindex="0"
            onclick={() => openOrder()}
            onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), openOrder())}
          >
            <div class="item-main">
              <span class="item-code">{order.order_number}</span>
              <span class="item-date">{formatDate(order.order_date)}</span>
            </div>
            <div class="item-meta">
              <span class="item-status">{order.status}</span>
              <span class="item-amount">{formatCurrency(order.total_value_bhd)} BHD</span>
              <button
                type="button"
                class="timeline-toggle"
                onclick={(e) => { e.stopPropagation(); toggleTimeline(order.order_number); }}
                disabled={resolvingOrderNumber === order.order_number}
              >
                {expandedOrderNumber === order.order_number ? 'Hide timeline' : 'Timeline'}
              </button>
            </div>
          </div>
          {#if expandedOrderNumber === order.order_number && expandedOrderId}
            <div class="inline-timeline">
              <DealTimeline orderId={expandedOrderId} />
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {:else}
    <p class="empty-text">No orders found</p>
  {/if}
</Card>

<style>
  .section-title { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 16px 0; }
  .empty-text { font-size: 13px; color: var(--text-muted); font-style: italic; }

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

  .list-wrap { display: flex; flex-direction: column; }
  .timeline-toggle {
    background: none;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 3px 8px;
    font-size: 11px;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .timeline-toggle:hover:not(:disabled) { color: var(--text-primary); border-color: var(--onyx); }
  .timeline-toggle:disabled { opacity: 0.5; cursor: not-allowed; }
  .inline-timeline {
    padding: 10px 12px 4px;
    background: var(--surface-elevated);
    border-radius: 0 0 6px 6px;
    margin-top: -8px;
  }
</style>
