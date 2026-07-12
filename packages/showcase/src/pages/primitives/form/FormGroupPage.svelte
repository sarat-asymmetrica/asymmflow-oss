<script lang="ts">
  import { FormGroup, Input, Textarea, Select, CurrencyInput, Checkbox, Button } from '@asymmflow/ui';
  import type { SelectOption } from '@asymmflow/ui';

  // ── Quotation entry form state ───────────────────────────────────────
  let vendorName = $state('');
  let vendorEmail = $state('');
  let category = $state('');
  let currency = $state('BHD');
  let amount = $state(0);
  let vatRate = $state('5');
  let notes = $state('');
  let urgentFlag = $state(false);
  let submitted = $state(false);

  // Validation errors — only shown after submit attempt
  let errors = $state<Record<string, string>>({});

  const categoryOptions: SelectOption[] = [
    { value: 'goods', label: 'Goods purchased' },
    { value: 'services', label: 'Professional services' },
    { value: 'travel', label: 'Travel & accommodation' },
    { value: 'utilities', label: 'Utilities' },
    { value: 'capex', label: 'Capital expenditure' },
  ];

  const vatOptions: SelectOption[] = [
    { value: '0', label: '0% — Exempt' },
    { value: '5', label: '5% — Standard rate' },
  ];

  const currencyOptions: SelectOption[] = [
    { value: 'BHD', label: 'BHD — Bahraini Dinar' },
    { value: 'USD', label: 'USD — US Dollar' },
    { value: 'AED', label: 'AED — UAE Dirham' },
    { value: 'EUR', label: 'EUR — Euro' },
  ];

  const decimalsForCurrency: Record<string, number> = {
    BHD: 3, KWD: 3, OMR: 3, USD: 2, EUR: 2, AED: 2,
  };

  let decimals = $derived(decimalsForCurrency[currency] ?? 2);

  function validate(): boolean {
    const e: Record<string, string> = {};
    if (!vendorName.trim()) e.vendorName = 'Vendor name is required.';
    if (vendorEmail && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(vendorEmail)) {
      e.vendorEmail = 'Enter a valid email address.';
    }
    if (!category) e.category = 'Category is required.';
    if (amount <= 0) e.amount = 'Amount must be greater than zero.';
    errors = e;
    return Object.keys(e).length === 0;
  }

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    submitted = true;
    if (validate()) {
      // In production: dispatch to parent or call API
      alert(`Quotation submitted: ${vendorName} — ${currency} ${amount.toFixed(decimals)}`);
    }
  }

  function handleReset() {
    vendorName = '';
    vendorEmail = '';
    category = '';
    currency = 'BHD';
    amount = 0;
    vatRate = '5';
    notes = '';
    urgentFlag = false;
    errors = {};
    submitted = false;
  }
</script>

<div class="sections">

  <section>
    <h2 class="af-section-title">FormGroup</h2>
    <p class="intro">
      FormGroup wires the label's <code>for</code> to the control's <code>id</code>,
      generates the <code>aria-describedby</code> id for hint and error messages,
      and applies <code>.af-label</code> typography automatically.
      The control itself stays dumb — it just accepts an <code>id</code> and
      <code>aria-describedby</code> as props.
    </p>
  </section>

  <!-- FormGroup anatomy -->
  <section>
    <h2 class="af-section-title">Anatomy</h2>
    <div class="card state-grid">
      <FormGroup label="Label only" controlId="demo-a">
        <Input id="demo-a" placeholder="Control goes here" />
      </FormGroup>

      <FormGroup label="With hint" controlId="demo-b" hint="This appears below the control.">
        <Input id="demo-b" placeholder="Control goes here" aria-describedby="demo-b-desc" />
      </FormGroup>

      <FormGroup label="With error" controlId="demo-c" error="This field has a problem.">
        <Input id="demo-c" invalid aria-describedby="demo-c-desc" value="bad value" />
      </FormGroup>

      <FormGroup label="Required field" controlId="demo-d" required hint="Asterisk is sr-accessible.">
        <Input id="demo-d" required placeholder="Required" aria-describedby="demo-d-desc" />
      </FormGroup>
    </div>
  </section>

  <!-- Composed quotation entry form -->
  <section>
    <h2 class="af-section-title">Quotation entry — composed form</h2>
    <p class="intro">
      A realistic ERP quotation form. Validation runs on submit; errors surface
      inline via <code>FormGroup</code>. The currency select drives the decimal
      precision of the amount field in real time.
    </p>

    <form class="card quotation-form" onsubmit={handleSubmit} onreset={handleReset} novalidate>
      <div class="form-header">
        <span class="af-label">New purchase quotation</span>
        <Checkbox bind:checked={urgentFlag} label="Mark as urgent" />
      </div>

      <div class="form-body">
        <!-- Row 1: Vendor -->
        <div class="form-row form-row--2col">
          <FormGroup
            label="Vendor name"
            controlId="q-vendor"
            required
            error={errors.vendorName}
          >
            <Input
              id="q-vendor"
              bind:value={vendorName}
              placeholder="e.g. Gulf Supply Co."
              invalid={!!errors.vendorName}
              aria-describedby={errors.vendorName ? 'q-vendor-desc' : undefined}
              required
            />
          </FormGroup>

          <FormGroup
            label="Vendor email"
            controlId="q-email"
            hint="Optional — for automatic PO dispatch."
            error={errors.vendorEmail}
          >
            <Input
              id="q-email"
              type="email"
              bind:value={vendorEmail}
              placeholder="accounts@supplier.com"
              invalid={!!errors.vendorEmail}
              aria-describedby="q-email-desc"
            />
          </FormGroup>
        </div>

        <!-- Row 2: Category + VAT -->
        <div class="form-row form-row--2col">
          <FormGroup
            label="Expense category"
            controlId="q-category"
            required
            error={errors.category}
          >
            <Select
              id="q-category"
              options={categoryOptions}
              bind:value={category}
              placeholder="Select category…"
              invalid={!!errors.category}
              aria-describedby={errors.category ? 'q-category-desc' : undefined}
              required
            />
          </FormGroup>

          <FormGroup
            label="VAT rate"
            controlId="q-vat"
            hint="Applied to the net amount."
          >
            <Select
              id="q-vat"
              options={vatOptions}
              bind:value={vatRate}
              aria-describedby="q-vat-desc"
            />
          </FormGroup>
        </div>

        <!-- Row 3: Currency + Amount -->
        <div class="form-row form-row--currency">
          <FormGroup
            label="Currency"
            controlId="q-currency"
          >
            <Select
              id="q-currency"
              options={currencyOptions}
              bind:value={currency}
            />
          </FormGroup>

          <FormGroup
            label="Net amount"
            controlId="q-amount"
            required
            error={errors.amount}
            hint={decimals === 3 ? '3-decimal precision (BHD/KWD/OMR)' : '2-decimal precision'}
          >
            <CurrencyInput
              id="q-amount"
              bind:value={amount}
              currency={currency}
              decimals={decimals}
              min={0}
              invalid={!!errors.amount}
              aria-describedby="q-amount-desc"
            />
          </FormGroup>
        </div>

        <!-- Row 4: Notes -->
        <FormGroup
          label="Remarks"
          controlId="q-notes"
          hint="Internal notes — not sent to vendor."
        >
          <Textarea
            id="q-notes"
            bind:value={notes}
            placeholder="Procurement reference, delivery conditions…"
            rows={3}
            autoResize
            aria-describedby="q-notes-desc"
          />
        </FormGroup>
      </div>

      <!-- Actions -->
      <div class="form-actions">
        <Button type="reset" variant="ghost">Clear form</Button>
        <Button type="submit" variant="primary">Submit quotation</Button>
      </div>

      {#if submitted && Object.keys(errors).length === 0}
        <div class="form-success" role="status">
          Quotation submitted — {vendorName}, {currency} {amount.toFixed(decimals)}.
        </div>
      {/if}
    </form>
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
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
    gap: var(--af-space-4);
  }

  /* ── Quotation form ──────────────────────────────────────────────── */
  .quotation-form {
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .form-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-bottom: var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
    margin-bottom: var(--af-space-4);
  }

  .form-body {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-4);
  }

  .form-row {
    display: grid;
    gap: var(--af-space-4);
  }

  .form-row--2col {
    grid-template-columns: 1fr 1fr;
  }

  .form-row--currency {
    grid-template-columns: 160px 1fr;
  }

  @media (max-width: 600px) {
    .form-row--2col,
    .form-row--currency {
      grid-template-columns: 1fr;
    }
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--af-space-3);
    padding-top: var(--af-space-5);
    margin-top: var(--af-space-4);
    border-top: 1px solid var(--af-border);
  }

  .form-success {
    margin-top: var(--af-space-4);
    padding: var(--af-space-3) var(--af-space-4);
    background: var(--af-accent-tint);
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    color: var(--af-accent-pressed);
    font-size: var(--af-text-sm);
    font-weight: var(--af-weight-medium);
  }
</style>
