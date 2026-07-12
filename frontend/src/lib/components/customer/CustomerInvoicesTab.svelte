<script lang="ts">
  import Card from '$lib/components/ui/Card.svelte';
  import { formatDate, formatCurrency } from './customerFormatters';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    profile: main.CustomerFullProfile;
    openInvoice: (invoice: any) => void;
  }

  let { profile, openInvoice }: Props = $props();
</script>

<Card padding="md">
  <h3 class="section-title">RECENT INVOICES</h3>
  {#if profile.recent_invoices?.length > 0}
    <div class="list">
      {#each profile.recent_invoices as invoice}
        <div
          class="list-item"
          role="button"
          tabindex="0"
          onclick={() => openInvoice(invoice)}
          onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), openInvoice(invoice))}
        >
          <div class="item-main">
            <span class="item-code">{invoice.invoice_number}</span>
            <span class="item-date">{formatDate(invoice.invoice_date)}</span>
          </div>
          <div class="item-meta">
            <span class="item-status status-{invoice.status?.toLowerCase()}">{invoice.status}</span>
            <span class="item-amount">{formatCurrency(invoice.grand_total_bhd)} BHD</span>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <p class="empty-text">No invoices found</p>
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
  .item-status.status-paid { color: #10B981; }
  .item-status.status-overdue { color: #EF4444; }
  .item-amount { font-weight: 600; font-family: 'JetBrains Mono', monospace; }
</style>
