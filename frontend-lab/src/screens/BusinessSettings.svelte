<script lang="ts">
  /* Business Settings — bespoke-on-primitives (K4 SettingsScreen split).
   * App prefs + business rules (default margin/VAT/currency/company name/
   * fiscal-year-start) consolidated into one form; see
   * screens/parity/Settings.parity.md for what else the old screen held and
   * where it went (ledgered vs retired). */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import { currencyOptions } from '../bridge/bank-accounts'
  import { BusinessSettingsViewModel } from './business-settings-vm.svelte'

  const vm = new BusinessSettingsViewModel()
  onMount(() => void vm.load())

  const MONTHS = [
    'January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December',
  ]
</script>

<PageShell title="Business Settings" subtitle="Company profile and default business rules.">
  {#if vm.loading}
    <EmptyState message="Loading settings…" />
  {:else if vm.error}
    <EmptyState message={`Could not load settings: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else}
    <Card padding="lg">
      <FormGrid columns={2}>
        <label class="bs-field">
          <span class="bs-label">Company Name</span>
          <input class="bs-input" type="text" bind:value={vm.draft.companyName} />
        </label>

        <label class="bs-field">
          <span class="bs-label">Base Currency</span>
          <select class="bs-input" bind:value={vm.draft.baseCurrency}>
            {#each currencyOptions() as opt (opt.value)}
              <option value={opt.value}>{opt.label}</option>
            {/each}
          </select>
        </label>

        <label class="bs-field">
          <span class="bs-label">Default Margin %</span>
          <input class="bs-input" type="number" step="0.1" min="0" bind:value={vm.draft.defaultMarginPercent} />
        </label>

        <label class="bs-field">
          <span class="bs-label">VAT Rate %</span>
          <input class="bs-input" type="number" step="0.1" min="0" bind:value={vm.draft.vatRatePercent} />
        </label>

        <label class="bs-field">
          <span class="bs-label">Fiscal Year Start</span>
          <select class="bs-input" bind:value={vm.draft.fiscalYearStartMonth}>
            {#each MONTHS as month, i (month)}
              <option value={i + 1}>{month}</option>
            {/each}
          </select>
        </label>
      </FormGrid>

      {#if vm.saveError}
        <p class="bs-error">Could not save: {vm.saveError}</p>
      {:else if vm.saved}
        <p class="bs-saved">Saved.</p>
      {/if}

      <div class="bs-footer">
        <Row justify="end">
          <Button variant="primary" onclick={() => vm.save()} disabled={vm.saving}>
            {vm.saving ? 'Saving…' : 'Save Changes'}
          </Button>
        </Row>
      </div>
    </Card>
  {/if}
</PageShell>

<style>
  /* Typography/control skin only (L1) — mirrors FormModal's k-field/k-input
   * look; scoped locally since FormModal's classes aren't :global. */
  .bs-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .bs-label {
    font-size: var(--modal-label-size);
    font-weight: var(--modal-label-weight);
    color: var(--text-secondary);
  }
  .bs-input {
    font-family: var(--font-ui);
    font-size: var(--modal-body-size);
    color: var(--text-primary);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--border-radius-sm);
    padding: 8px 10px;
    max-width: 100%;
    min-width: 0;
    outline: none;
    transition: border-color var(--motion-fast) var(--ease-standard);
  }
  .bs-input:focus {
    border-color: var(--onyx);
  }
  .bs-error {
    margin-top: var(--k-space-sm);
    font-size: var(--modal-body-size);
    color: #b3261e;
    overflow-wrap: break-word;
  }
  .bs-saved {
    margin-top: var(--k-space-sm);
    font-size: var(--modal-body-size);
    color: var(--k-tone-success-fg);
  }
  .bs-footer {
    margin-top: var(--k-space-md);
  }
</style>
