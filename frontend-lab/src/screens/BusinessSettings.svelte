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
  import Stack from '$kernel/primitives/Stack.svelte'
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
      <Stack gap="md">
        <FormGrid columns={2}>
          <label class="k-field">
            <span class="k-field-label">Company Name</span>
            <input class="k-input" type="text" bind:value={vm.draft.companyName} />
          </label>

          <label class="k-field">
            <span class="k-field-label">Base Currency</span>
            <select class="k-input" bind:value={vm.draft.baseCurrency}>
              {#each currencyOptions() as opt (opt.value)}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          </label>

          <label class="k-field">
            <span class="k-field-label">Default Margin %</span>
            <input class="k-input" type="number" step="0.1" min="0" bind:value={vm.draft.defaultMarginPercent} />
          </label>

          <label class="k-field">
            <span class="k-field-label">VAT Rate %</span>
            <input class="k-input" type="number" step="0.1" min="0" bind:value={vm.draft.vatRatePercent} />
          </label>

          <label class="k-field">
            <span class="k-field-label">Fiscal Year Start</span>
            <select class="k-input" bind:value={vm.draft.fiscalYearStartMonth}>
              {#each MONTHS as month, i (month)}
                <option value={i + 1}>{month}</option>
              {/each}
            </select>
          </label>
        </FormGrid>

        {#if vm.saveError}
          <span class="bs-message bs-error">Could not save: {vm.saveError}</span>
        {:else if vm.saved}
          <span class="bs-message bs-saved">Saved.</span>
        {/if}

        <Row justify="end">
          <Button variant="primary" onclick={() => vm.save()} disabled={vm.saving}>
            {vm.saving ? 'Saving…' : 'Save Changes'}
          </Button>
        </Row>
      </Stack>
    </Card>

    <!-- R4: AI provider (Butler) key. Encrypted at rest server-side; only the
         server-masked last-4 is ever shown — the plaintext is write-only. -->
    <Card padding="lg">
      <Stack gap="md">
        <span class="k-field-label">AI Provider Key (Butler)</span>
        <span class="bs-message">
          Used by the Butler assistant. Stored encrypted on this device; only the last 4
          characters are ever shown, and the key is never displayed in full again.
        </span>

        <FormGrid columns={2}>
          <label class="k-field">
            <span class="k-field-label">Current Key</span>
            <input class="k-input" type="text" value={vm.aiKey.maskedKey} readonly />
          </label>

          <label class="k-field">
            <span class="k-field-label">{vm.aiKey.isSet ? 'Replace Key' : 'Set Key'}</span>
            <input
              class="k-input"
              type="password"
              autocomplete="off"
              placeholder="Paste the provider API key"
              bind:value={vm.aiKeyInput}
            />
          </label>
        </FormGrid>

        {#if vm.aiKeyError}
          <span class="bs-message bs-error">Could not save key: {vm.aiKeyError}</span>
        {:else if vm.aiKeySaved}
          <span class="bs-message bs-saved">API key saved (encrypted).</span>
        {/if}

        <Row justify="end">
          <Button variant="primary" onclick={() => vm.saveAIKey()} disabled={vm.aiKeySaving}>
            {vm.aiKeySaving ? 'Saving…' : 'Save Key'}
          </Button>
        </Row>
      </Stack>
    </Card>
  {/if}
</PageShell>

<style>
  /* Form controls now use the kernel k-field/k-field-label/k-input classes
   * (styles/kernel.css, L2 single-source); spacing is owned by the Stack
   * primitive. Only save-status message typography/color remains here (L1:
   * font + color via tokens only). */
  .bs-message {
    font-size: var(--modal-body-size);
    overflow-wrap: break-word;
  }
  .bs-error {
    color: var(--k-tone-danger-fg);
  }
  .bs-saved {
    color: var(--k-tone-success-fg);
  }
</style>
