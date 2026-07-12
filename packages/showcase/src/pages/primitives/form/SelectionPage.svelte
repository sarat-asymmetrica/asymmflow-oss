<script lang="ts">
  import { Select, Checkbox, Toggle } from '@asymmflow/ui';
  import type { SelectOption } from '@asymmflow/ui';

  const categoryOptions: SelectOption[] = [
    { value: 'goods', label: 'Goods purchased' },
    { value: 'services', label: 'Professional services' },
    { value: 'travel', label: 'Travel & accommodation' },
    { value: 'utilities', label: 'Utilities' },
    { value: 'capex', label: 'Capital expenditure' },
  ];

  const vatOptions: SelectOption[] = [
    { value: '0', label: '0% (exempt)' },
    { value: '5', label: '5% standard' },
    { value: '10', label: '10% luxury' },
  ];

  let category = $state('');
  let vatRate = $state('5');
  let categoryInvalid = $state(false);

  // Checkboxes
  let cbDefault = $state(false);
  let cbChecked = $state(true);
  let cbIndeterminate = $state(false);
  let cbDisabled = $state(false);
  let cbCheckedDisabled = $state(true);

  // Bulk-select demo
  const lineItems = ['Invoice #001', 'Invoice #002', 'Invoice #003', 'Invoice #004'];
  let selected = $state<boolean[]>([false, true, false, false]);
  let allChecked = $derived(selected.every(Boolean));
  let someChecked = $derived(selected.some(Boolean) && !allChecked);

  function toggleAll() {
    const next = !allChecked;
    selected = selected.map(() => next);
  }

  // Toggles
  let toggleA = $state(false);
  let toggleB = $state(true);
  let toggleC = $state(false);
  let toggleD = $state(true);
  let toggleDisabled = $state(false);
</script>

<div class="sections">

  <!-- Select -->
  <section>
    <h2 class="af-section-title">Select</h2>
    <p class="intro">
      Styled native <code>&lt;select&gt;</code> — browser's own scroll, keyboard, and
      platform list. A custom chevron replaces the UA arrow. Token-driven focus ring.
      Never a custom dropdown unless search is required (search is a future pattern).
    </p>

    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">default</span>
        <Select options={categoryOptions} bind:value={category} placeholder="Select category…" />
      </div>
      <div class="demo-cell">
        <span class="af-label">with value</span>
        <Select options={vatOptions} bind:value={vatRate} />
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled</span>
        <Select options={categoryOptions} value="services" disabled />
      </div>
      <div class="demo-cell">
        <span class="af-label">invalid</span>
        <Select
          options={categoryOptions}
          value=""
          invalid={categoryInvalid}
          placeholder="Select category…"
          aria-describedby="cat-err"
          onfocus={() => (categoryInvalid = false)}
          onblur={() => (categoryInvalid = !category)}
        />
        {#if categoryInvalid}
          <span id="cat-err" class="af-meta" style="color: var(--af-danger);">Category is required.</span>
        {/if}
      </div>
      <div class="demo-cell">
        <span class="af-label">children slot</span>
        <Select value="aed">
          <option value="bhd">Bahraini Dinar (BHD)</option>
          <option value="usd">US Dollar (USD)</option>
          <option value="aed">UAE Dirham (AED)</option>
          <option value="eur">Euro (EUR)</option>
        </Select>
      </div>
    </div>
  </section>

  <!-- Checkbox -->
  <section>
    <h2 class="af-section-title">Checkbox</h2>
    <p class="intro">
      Custom-drawn box — SVG tick for checked, SVG dash for indeterminate.
      The real <code>&lt;input type="checkbox"&gt;</code> is visually hidden but
      keyboard and screen-reader accessible. Touch target meets 44px floor.
    </p>

    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">unchecked</span>
        <Checkbox bind:checked={cbDefault} label="Accept terms" />
      </div>
      <div class="demo-cell">
        <span class="af-label">checked</span>
        <Checkbox bind:checked={cbChecked} label="Send remittance slip" />
      </div>
      <div class="demo-cell">
        <span class="af-label">indeterminate</span>
        <Checkbox bind:checked={cbIndeterminate} indeterminate label="Select all (mixed)" />
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled unchecked</span>
        <Checkbox bind:checked={cbDisabled} label="Automated reconciliation" disabled />
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled checked</span>
        <Checkbox bind:checked={cbCheckedDisabled} label="VAT registered" disabled />
      </div>
    </div>

    <h3 class="af-section-title" style="margin-top: var(--af-space-5); font-size: var(--af-text-md);">Bulk-select pattern</h3>
    <p class="intro">The header checkbox drives indeterminate state — the standard table bulk-select UX.</p>
    <div class="card">
      <div class="bulk-row bulk-row--header">
        <Checkbox
          checked={allChecked}
          indeterminate={someChecked}
          onCheckedChange={toggleAll}
          aria-label="Select all invoices"
        />
        <span class="af-label">Invoice</span>
        <span class="af-label" style="margin-inline-start: auto;">Status</span>
      </div>
      {#each lineItems as item, i}
        <div class="bulk-row">
          <Checkbox bind:checked={selected[i]} aria-label="Select {item}" />
          <span style="font-size: var(--af-text-sm);">{item}</span>
          <span class="af-meta" style="margin-inline-start: auto;">
            {selected[i] ? 'Selected' : 'Unselected'}
          </span>
        </div>
      {/each}
    </div>
  </section>

  <!-- Toggle -->
  <section>
    <h2 class="af-section-title">Toggle</h2>
    <p class="intro">
      Role <code>switch</code> on the underlying checkbox. Thumb uses
      <code>--af-motion-optimize-*</code> for the slide — felt as instant.
      Label position can be <code>start</code> for settings-list layouts.
    </p>

    <div class="card state-grid">
      <div class="demo-cell">
        <span class="af-label">off</span>
        <Toggle bind:checked={toggleA} label="Email notifications" />
      </div>
      <div class="demo-cell">
        <span class="af-label">on</span>
        <Toggle bind:checked={toggleB} label="Auto-reconcile" />
      </div>
      <div class="demo-cell">
        <span class="af-label">with description</span>
        <Toggle
          bind:checked={toggleC}
          label="Two-factor auth"
          description="Requires a TOTP app on sign-in"
        />
      </div>
      <div class="demo-cell">
        <span class="af-label">label-start</span>
        <Toggle
          bind:checked={toggleD}
          label="Dark mode"
          labelPosition="start"
          style="width: 100%"
        />
      </div>
      <div class="demo-cell">
        <span class="af-label">disabled</span>
        <Toggle bind:checked={toggleDisabled} label="SWIFT payments" disabled />
      </div>
    </div>

    <!-- Settings list context -->
    <h3 class="af-section-title" style="margin-top: var(--af-space-5); font-size: var(--af-text-md);">Settings list pattern</h3>
    <div class="card settings-list">
      <div class="settings-row">
        <Toggle bind:checked={toggleB} label="Auto-reconcile" description="Matches statements nightly" labelPosition="start" />
      </div>
      <div class="settings-row">
        <Toggle bind:checked={toggleC} label="Two-factor auth" description="Requires TOTP on sign-in" labelPosition="start" />
      </div>
      <div class="settings-row">
        <Toggle bind:checked={toggleD} label="Email digest" description="Weekly summary of transactions" labelPosition="start" />
      </div>
    </div>
  </section>

</div>

<style>
  .sections {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-6);
  }

  .intro {
    color: var(--af-text-secondary);
    font-size: var(--af-text-md);
    max-width: 64ch;
    margin-top: var(--af-space-2);
    margin-bottom: var(--af-space-4);
  }

  .card {
    background: var(--af-surface);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-md);
    padding: var(--af-card-padding);
  }

  .state-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: var(--af-space-4);
  }

  .demo-cell {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  /* Bulk-select rows */
  .bulk-row {
    display: flex;
    align-items: center;
    gap: var(--af-space-3);
    padding: var(--af-space-2) 0;
    border-bottom: 1px solid var(--af-border);
  }

  .bulk-row:last-child {
    border-bottom: none;
  }

  .bulk-row--header {
    padding-bottom: var(--af-space-3);
  }

  /* Settings list */
  .settings-list {
    padding: 0 var(--af-card-padding);
  }

  .settings-row {
    padding: var(--af-space-3) 0;
    border-bottom: 1px solid var(--af-border);
  }

  .settings-row:last-child {
    border-bottom: none;
  }
</style>
