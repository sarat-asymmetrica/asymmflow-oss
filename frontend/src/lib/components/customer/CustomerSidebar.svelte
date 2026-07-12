<script lang="ts">
  import Card from '$lib/components/ui/Card.svelte';
  import { formatCurrency } from './customerFormatters';
  import type { main } from '../../../../wailsjs/go/models';

  interface Props {
    profile: main.CustomerFullProfile;
  }

  let { profile }: Props = $props();
</script>

<aside class="sidebar">
  <Card padding="md">
    <h3 class="sidebar-title">OVERVIEW</h3>
    <div class="info-grid">
      <div class="info-item">
        <span class="label">Customer ID</span>
        <span class="value code">{profile.customer_id}</span>
      </div>
      <div class="info-item">
        <span class="label">TRN</span>
        <span class="value code">{profile.trn || 'N/A'}</span>
      </div>
      <div class="info-item">
        <span class="label">Industry</span>
        <span class="value">{profile.industry || 'N/A'}</span>
      </div>
      <div class="info-item">
        <span class="label">Relationship</span>
        <span class="value">{profile.relation_years} years</span>
      </div>
      <div class="info-item">
        <span class="label">Location</span>
        <span class="value">{profile.city}, {profile.country}</span>
      </div>
    </div>
  </Card>

  <Card padding="md">
    <h3 class="sidebar-title">FINANCIAL</h3>
    <div class="info-grid">
      <div class="info-item">
        <span class="label">Payment Terms</span>
        <span class="value">{profile.payment_terms_days} days</span>
      </div>
      <div class="info-item">
        <span class="label">Open Exposure</span>
        <span class="value">{formatCurrency(profile.outstanding_bhd)} BHD</span>
      </div>
      <div class="info-item">
        <span class="label">Overdue</span>
        <span class="value" class:danger={profile.overdue_bhd > 0}>{formatCurrency(profile.overdue_bhd)} BHD</span>
      </div>
      <div class="info-item">
        <span class="label">Credit Blocked</span>
        <span class="value">{profile.is_credit_blocked ? 'Yes' : 'No'}</span>
      </div>
    </div>
  </Card>
</aside>

<style>
  .sidebar { display: flex; flex-direction: column; gap: 16px; }
  .sidebar-title { font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-secondary); margin: 0 0 12px 0; }

  .info-grid { display: flex; flex-direction: column; gap: 12px; }
  .info-item { display: flex; flex-direction: column; gap: 2px; }
  .label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; }
  .value { font-size: 14px; font-weight: 500; }
  .value.code { font-family: 'JetBrains Mono', monospace; color: var(--brand-indigo); }
  .value.danger { color: #EF4444; }
</style>
