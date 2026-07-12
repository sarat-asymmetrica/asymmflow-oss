<script lang="ts">
  import Card from '$lib/components/ui/Card.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import { formatCurrency } from './customerFormatters';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    profile: main.CustomerFullProfile;
    editMode: boolean;
    editForm: any;
    saveLoading: boolean;
    cancelEdit: () => void;
    saveEdit: () => void | Promise<void>;
  }

  let { profile, editMode, editForm = $bindable(), saveLoading, cancelEdit, saveEdit }: Props = $props();
</script>

{#if editMode}
  <Card padding="md">
    <h3 class="section-title">EDIT CUSTOMER</h3>
    <div class="edit-form">
      <div class="form-row">
        <div class="form-group"><label for="customer-detail-business-name">Business Name</label><input id="customer-detail-business-name" type="text" bind:value={editForm.business_name} /></div>
        <div class="form-group"><label for="customer-detail-customer-code">Customer Code</label><input id="customer-detail-customer-code" type="text" bind:value={editForm.customer_code} /></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label for="customer-detail-customer-type">Customer Type</label><input id="customer-detail-customer-type" type="text" bind:value={editForm.customer_type} /></div>
        <div class="form-group"><label for="customer-detail-industry">Industry</label><input id="customer-detail-industry" type="text" bind:value={editForm.industry} /></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label for="customer-detail-address">Address</label><input id="customer-detail-address" type="text" bind:value={editForm.address_line1} /></div>
        <div class="form-group"><label for="customer-detail-city">City</label><input id="customer-detail-city" type="text" bind:value={editForm.city} /></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label for="customer-detail-country">Country</label><input id="customer-detail-country" type="text" bind:value={editForm.country} /></div>
        <div class="form-group"><label for="customer-detail-trn">TRN</label><input id="customer-detail-trn" type="text" bind:value={editForm.trn} /></div>
      </div>
      <div class="form-row">
        <div class="form-group"><label for="customer-detail-mobile-number">Mobile Number</label><input id="customer-detail-mobile-number" type="text" bind:value={editForm.mobile_number} placeholder="+973 3XXX XXXX" /></div>
      </div>
      <!-- Wave 9.6 Sh3: phone/email/vat_number inputs removed here. They bound
           to editForm.phone/.email/.vat_number, but CustomerFullProfile exposes
           none of those keys (so they always render blank on edit), and
           MergeCustomerUpdate never reads incoming.Phone/.Email; vat_number isn't
           a CustomerMaster column at all. Pure dead-ends that never persisted —
           removed rather than half-wired. Mobile Number above is real (MobileNumber
           is merged and CustomerFullProfile's edit source seeds it correctly). -->
      <div class="form-row">
        <div class="form-group"><label for="customer-detail-payment-grade">Payment Grade</label>
          <select id="customer-detail-payment-grade" bind:value={editForm.payment_grade}>
            <option value="A">A</option><option value="B">B</option>
            <option value="C">C</option><option value="D">D</option>
          </select>
        </div>
        <div class="form-group"><label for="customer-detail-payment-terms-days">Payment Terms (Days)</label><input id="customer-detail-payment-terms-days" type="number" bind:value={editForm.payment_terms_days} /></div>
      </div>
      <div class="form-actions">
        <Button variant="secondary" on:click={cancelEdit} disabled={saveLoading}>Cancel</Button>
        <Button variant="primary" on:click={saveEdit} disabled={saveLoading}>
          {saveLoading ? 'Saving...' : 'Save Changes'}
        </Button>
      </div>
    </div>
  </Card>
{/if}
<div class="metrics-grid">
  <Card padding="md">
    <div class="metric">
      <span class="metric-label">Total Business</span>
      <span class="metric-value">{formatCurrency(profile.total_revenue)}</span>
      <span class="metric-currency">BHD</span>
    </div>
  </Card>
  <Card padding="md">
    <div class="metric">
      <span class="metric-label">Total Orders</span>
      <span class="metric-value">{profile.total_orders}</span>
    </div>
  </Card>
  <Card padding="md">
    <div class="metric">
      <span class="metric-label">Avg Order Value</span>
      <span class="metric-value">{formatCurrency(profile.avg_order_value)}</span>
      <span class="metric-currency">BHD</span>
    </div>
  </Card>
  <Card padding="md">
    <div class="metric">
      <span class="metric-label">Win Rate</span>
      <span class="metric-value">{profile.win_rate?.toFixed(0) || 0}%</span>
    </div>
  </Card>
</div>

<Card padding="md">
  <h3 class="section-title">EXPOSURE AGING</h3>
  <div class="aging-grid">
    <div class="aging-item">
      <span class="aging-label">Current</span>
      <span class="aging-value">{formatCurrency(profile.ar_aging_buckets?.current)}</span>
      <span class="aging-currency">BHD</span>
    </div>
    <div class="aging-item">
      <span class="aging-label">31-60 Days</span>
      <span class="aging-value">{formatCurrency(profile.ar_aging_buckets?.days_30_60)}</span>
      <span class="aging-currency">BHD</span>
    </div>
    <div class="aging-item">
      <span class="aging-label">61-90 Days</span>
      <span class="aging-value">{formatCurrency(profile.ar_aging_buckets?.days_60_90)}</span>
      <span class="aging-currency">BHD</span>
    </div>
    <div class="aging-item">
      <span class="aging-label">90+ Days</span>
      <span class="aging-value danger">{formatCurrency((profile.ar_aging_buckets?.days_90_120 || 0) + (profile.ar_aging_buckets?.days_120_plus || 0))}</span>
      <span class="aging-currency danger">BHD</span>
    </div>
  </div>
</Card>

<style>
  .section-title { font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 16px 0; }

  .metrics-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 16px; }
  .metric { text-align: center; display: flex; flex-direction: column; align-items: center; }
  .metric-label { display: block; font-size: 11px; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 4px; }
  .metric-value { font-size: 24px; font-weight: 600; }
  .metric-currency { font-size: 11px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-secondary); margin-top: 4px; }

  .aging-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
  .aging-item { text-align: center; padding: 12px; background: var(--surface-elevated); border-radius: 8px; display: flex; flex-direction: column; align-items: center; }
  .aging-label { display: block; font-size: 11px; color: var(--text-muted); margin-bottom: 4px; }
  .aging-value { font-size: 16px; font-weight: 600; font-family: 'JetBrains Mono', monospace; }
  .aging-value.danger { color: #EF4444; }
  .aging-currency { font-size: 10px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-secondary); margin-top: 4px; }
  .aging-currency.danger { color: #EF4444; }

  .form-group { margin-bottom: 16px; }
  .edit-form { display: flex; flex-direction: column; gap: 12px; }
  .edit-form .form-row { display: flex; gap: 16px; }
  .edit-form .form-row .form-group { flex: 1; }
  .edit-form .form-group label { display: block; font-size: 11px; font-weight: 500; color: var(--steel, #86868B); margin-bottom: 4px; text-transform: uppercase; letter-spacing: 0.03em; }
  .edit-form .form-group input, .edit-form .form-group select { width: 100%; padding: 8px 10px; border: 1px solid var(--border, #E5E5E5); border-radius: 6px; font-size: 14px; }
  .edit-form .form-group input:focus, .edit-form .form-group select:focus { outline: none; border-color: var(--onyx, #1D1D1F); }
  .form-actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 8px; }
</style>
