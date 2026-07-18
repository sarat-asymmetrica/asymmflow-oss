<script lang="ts">
  /* Costing Sheet — bespoke-on-primitives (K5) full-page cost/quote workspace,
   * NOT a modal. Two modes: pick an opportunity (or start blank), then build
   * a pricing sheet whose per-line waterfall + sheet totals are the SACRED
   * math ported verbatim into costing-sheet-vm.svelte.ts (calcLine /
   * sheetTotals) — this file only composes primitives and renders (L1); it
   * computes nothing. See screens/parity/CostingSheet.parity.md for the full
   * capability census, INTEG-gap ledger, and the documented simplifications
   * (customer near-dup matching, session-restore flows dropped). */
  import { onMount } from 'svelte'
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import Grid from '$kernel/primitives/Grid.svelte'
  import Card from '$kernel/primitives/Card.svelte'
  import Stack from '$kernel/primitives/Stack.svelte'
  import Row from '$kernel/primitives/Row.svelte'
  import Toolbar from '$kernel/primitives/Toolbar.svelte'
  import FormGrid from '$kernel/primitives/FormGrid.svelte'
  import DataTable from '$kernel/primitives/DataTable.svelte'
  import Button from '$kernel/controls/Button.svelte'
  import EmptyState from '$kernel/controls/EmptyState.svelte'
  import ConfirmDialog from '$kernel/controls/ConfirmDialog.svelte'
  import CalloutWidget from '$kernel/widgets/CalloutWidget.svelte'
  import StatTileGrid from '$kernel/widgets/StatTileGrid.svelte'
  import LineItemsEditor from '$kernel/widgets/LineItemsEditor.svelte'
  import { sumField } from '$kernel/line-items'
  import type { LineColumn, LineFooterCell } from '$kernel/line-items'
  import type { ColumnSpec, StatusSpec } from '$kernel/descriptor'
  import type { Tone } from '$kernel/tones'
  import { formatMoney, formatNumber } from '$kernel/format'
  import {
    CURRENCY_OPTIONS,
    type CostingOpportunityRow,
    type CostingRevisionRow,
    type CostingSheetSummaryRow,
    type CostingLineRow,
  } from '../bridge/costing-sheet'
  import { CostingSheetViewModel, MAX_COSTING_LINE_ITEMS } from './costing-sheet-vm.svelte'

  let { embedded = false }: { embedded?: boolean } = $props()

  const vm = new CostingSheetViewModel()
  onMount(() => void vm.load())

  /* ---- Opportunity picker: full dropdown label + preview/DataTable columns ---- */

  function opportunityLabel(o: CostingOpportunityRow): string {
    const value = o.value > 0 ? formatMoney(o.value) : 'Value pending'
    return `${o.ref || 'Manual'} • ${o.customer || 'Unknown'} - ${o.project || 'New requirement'} (${o.status}) — ${value}`
  }

  const STAGE_TONES: Record<string, Tone> = {
    Pending: 'neutral',
    Qualified: 'info',
    Proposal: 'info',
    Negotiation: 'warning',
    Won: 'success',
    Lost: 'danger',
  }
  const opportunityStatus: StatusSpec<CostingOpportunityRow> = { value: (r) => r.status, tones: STAGE_TONES }
  const opportunityColumns: ColumnSpec<CostingOpportunityRow>[] = [
    { key: 'ref', label: 'Reference', content: 'code', value: (r) => r.ref || 'Manual', minWidth: 110 },
    { key: 'customer', label: 'Customer', content: 'name', value: (r) => r.customer || 'Unknown', grow: true, minWidth: 180 },
    { key: 'project', label: 'Project', content: 'text', value: (r) => r.project || 'New requirement', minWidth: 180 },
    {
      key: 'value',
      label: 'Value',
      content: 'text',
      value: (r) => (r.value > 0 ? formatMoney(r.value) : 'Value pending'),
      minWidth: 130,
    },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 110 },
  ]
  const oppId = (r: CostingOpportunityRow) => r.id

  /* ---- Revisions strip (RFQ-scoped) ---- */

  const REVISION_TONES: Record<string, Tone> = { Draft: 'neutral', Approved: 'success', Rejected: 'danger' }
  const revisionStatus: StatusSpec<CostingRevisionRow> = { value: (r) => r.status, tones: REVISION_TONES }
  const revisionColumns: ColumnSpec<CostingRevisionRow>[] = [
    { key: 'rev', label: 'Revision', content: 'code', value: (r) => `Rev ${r.revisionNumber}`, minWidth: 90 },
    {
      key: 'current',
      label: 'Current',
      content: 'text',
      value: (r) => (r.isActive ? 'Current' : ''),
      tone: (r) => (r.isActive ? 'success' : 'neutral'),
      minWidth: 90,
    },
    { key: 'status', label: 'Status', content: 'status', value: (r) => r.status, minWidth: 100 },
    { key: 'createdAt', label: 'Created', content: 'date', value: (r) => r.createdAt, minWidth: 110 },
    { key: 'createdBy', label: 'By', content: 'name', value: (r) => r.createdBy, minWidth: 130 },
    { key: 'finalPrice', label: 'Final Price', content: 'money', value: (r) => r.finalPrice, minWidth: 130 },
  ]
  const revisionId = (r: CostingRevisionRow) => String(r.id)

  /* ---- Recent costings (display-only) ---- */
  const recentColumns: ColumnSpec<CostingSheetSummaryRow>[] = [
    { key: 'ref', label: 'Ref', content: 'code', value: (r) => r.ref, minWidth: 90 },
    { key: 'customerName', label: 'Customer', content: 'name', value: (r) => r.customerName, grow: true, minWidth: 160 },
    { key: 'totalSellBHD', label: 'Amount', content: 'money', value: (r) => r.totalSellBHD, minWidth: 130 },
  ]
  const recentId = (r: CostingSheetSummaryRow) => r.ref

  /* ---- Line items — the pricing waterfall (LineItemsEditor is presentation
   * only; every value below reads vm.calc(line), never computes locally). ---- */

  const currencyOpts = CURRENCY_OPTIONS.map((c) => ({ value: c, label: c }))

  const lineColumns: LineColumn<CostingLineRow>[] = [
    { key: 'equipment', label: 'Equipment', kind: 'text', minWidth: 170, value: (l) => l.equipment, set: (l, v) => { l.equipment = String(v) } },
    { key: 'model', label: 'Model', kind: 'text', minWidth: 130, value: (l) => l.model, set: (l, v) => { l.model = String(v) } },
    { key: 'longCode', label: 'Long Code', kind: 'text', wide: true, value: (l) => l.longCode, set: (l, v) => { l.longCode = String(v) } },
    {
      key: 'detailedDescription',
      label: 'Detailed Description',
      kind: 'textarea',
      wide: true,
      value: (l) => l.detailedDescription,
      set: (l, v) => { l.detailedDescription = String(v) },
    },
    { key: 'quantity', label: 'Qty', kind: 'number', minWidth: 80, value: (l) => l.quantity, set: (l, v) => { l.quantity = Number(v) || 0 } },
    {
      key: 'currency',
      label: 'Currency',
      kind: 'select',
      minWidth: 90,
      value: (l) => l.currency,
      set: (l, v) => { l.currency = String(v) },
      options: () => currencyOpts,
    },
    {
      key: 'fobForeign',
      label: 'FOB',
      kind: 'money',
      step: '0.001',
      minWidth: 120,
      value: (l) => l.fobForeign,
      set: (l, v) => { l.fobForeign = Number(v) || 0 },
      currency: (l) => l.currency,
    },
    { key: 'freightPercent', label: 'Freight %', kind: 'percent', minWidth: 90, value: (l) => l.freightPercent, set: (l, v) => { l.freightPercent = Number(v) || 0 } },
    { key: 'customsPercent', label: 'Customs %', kind: 'percent', minWidth: 90, value: (l) => l.customsPercent, set: (l, v) => { l.customsPercent = Number(v) || 0 } },
    { key: 'handlingPercent', label: 'Handling %', kind: 'percent', minWidth: 90, value: (l) => l.handlingPercent, set: (l, v) => { l.handlingPercent = Number(v) || 0 } },
    { key: 'financePercent', label: 'Finance %', kind: 'percent', minWidth: 90, value: (l) => l.financePercent, set: (l, v) => { l.financePercent = Number(v) || 0 } },
    { key: 'insurance', label: 'Insurance', kind: 'money', step: '0.001', minWidth: 110, value: (l) => l.insurance, set: (l, v) => { l.insurance = Number(v) || 0 } },
    { key: 'otherCosts', label: 'Other Costs', kind: 'money', step: '0.001', minWidth: 110, value: (l) => l.otherCosts, set: (l, v) => { l.otherCosts = Number(v) || 0 } },
    {
      key: 'userPrice',
      label: 'Manual Price',
      kind: 'money',
      step: '0.001',
      minWidth: 130,
      value: (l) => l.userPrice,
      set: (l, v) => {
        l.userPrice = Number(v) || 0
        l.userPriceSet = l.userPrice > 0
      },
      placeholder: (l) => formatNumber(vm.calc(l).suggestedPriceUnit),
      tone: (l) => (l.userPriceSet ? 'warning' : 'neutral'),
    },
    { key: 'fobBHD', label: 'FOB (BHD)', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).fobBHD },
    { key: 'freightBHD', label: 'Freight (BHD)', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).freightBHD },
    { key: 'cf', label: 'C&F', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).cf },
    { key: 'customsBHD', label: 'Customs (BHD)', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).customsBHD },
    { key: 'landedCost', label: 'Landed Cost', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).landedCost },
    { key: 'handlingBHD', label: 'Handling (BHD)', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).handlingBHD },
    { key: 'financeBHD', label: 'Finance (BHD)', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).financeBHD },
    { key: 'totalCost', label: 'Total Cost', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).totalCost },
    { key: 'marginPercent', label: 'Margin %', kind: 'percent', minWidth: 90, value: (l) => l.marginPercent, set: (l, v) => { l.marginPercent = Number(v) || 0 } },
    { key: 'marginBHD', label: 'Margin (BHD)', kind: 'readonly', content: 'money', minWidth: 120, value: (l) => vm.calc(l).marginBHD },
    { key: 'suggestedPriceUnit', label: 'Suggested Price', kind: 'readonly', content: 'money', minWidth: 130, value: (l) => vm.calc(l).suggestedPriceUnit },
    { key: 'totalSuggestedPrice', label: 'Total Price', kind: 'readonly', content: 'money', minWidth: 140, value: (l) => vm.calc(l).totalSuggestedPrice },
  ]

  const lineFooter: LineFooterCell<CostingLineRow>[] = [
    { label: 'Subtotal', content: 'money', value: (rows) => sumField(rows, (r) => vm.calc(r).totalSuggestedPrice) },
  ]

  /* ---- Summary rail (display-only tone, not sacred) ---- */
  function profitTone(pct: number): Tone {
    if (pct < 0) return 'danger'
    if (pct < 10) return 'warning'
    return 'success'
  }

  const ORDER_TYPE_OPTIONS = ['General', 'Spareparts', 'Items + Spareparts']
  const COUNTRY_OPTIONS = ['DE', 'CH', 'FR', 'UK', 'US', 'SL', 'GR', 'IT']
  const CERTIFICATE_OPTIONS = ['Yes', 'No', 'Additional charges applicable as per OEM terms']
  const PAYMENT_TERMS_OPTIONS = [
    '100% Advance Payment with PO',
    '100% Payment Against Delivery',
    '30 days from Date of Delivery',
    '60 days from Date of Delivery',
    'Letter of Credit (LC)',
    'Project Stage Payments',
    '50% Advance + 50% Against Delivery',
  ]
  const DELIVERY_TERMS_OPTIONS = ['Ex-Works (EXW)', 'Free Carrier (FCA)', 'Delivered Duty Paid (DDP)']
  const EST_DELIVERY_OPTIONS = ['3-5 weeks', '4-6 weeks', '5-7 weeks', '7-9 weeks', '9-11 weeks', '12-14 weeks', '16-18 weeks', 'On request']
  const DOC_TYPE_OPTIONS = ['Quotation', 'Budgetary Quote', 'Budgetary Estimate', 'Technical Offer', 'Commercial Offer']
</script>

<PageShell {embedded} title="Costing Sheet" subtitle="Build a cost/quote sheet from an opportunity, or start blank.">
  {#snippet toolbar()}
    <Toolbar>
      <Button onclick={() => vm.load()}>Refresh</Button>
      {#if vm.formOpen}
        <Button onclick={() => vm.exportExcel()} disabled={vm.exporting}>{vm.exporting ? 'Exporting…' : 'Export Excel'}</Button>
        <Button onclick={() => vm.exportPDF()} disabled={vm.exporting}>{vm.exporting ? 'Exporting…' : 'Export PDF'}</Button>
        {#if vm.isRFQOpportunity}
          <Button onclick={() => vm.saveCosting()} disabled={vm.savingCosting}>{vm.savingCosting ? 'Saving…' : 'Save Costing'}</Button>
        {/if}
        <Button variant="primary" onclick={() => vm.requestSaveAsOffer()} disabled={vm.savingOffer}>
          {vm.savingOffer ? 'Saving…' : 'Save as Offer'}
        </Button>
        <Button onclick={() => vm.closeForm()}>Change Opportunity</Button>
      {/if}
    </Toolbar>
  {/snippet}

  {#if vm.loading}
    <EmptyState message="Loading costing workspace…" />
  {:else if vm.error}
    <EmptyState message={`Could not load costing data: ${vm.error}`}>
      {#snippet actions()}
        <Button onclick={() => vm.load()}>Retry</Button>
      {/snippet}
    </EmptyState>
  {:else if !vm.formOpen}
    <Stack gap="lg">
      {#if vm.settingsError}
        <CalloutWidget items={[{ label: 'Settings unavailable', text: `Using fallback VAT 10% / margin 20% (${vm.settingsError}).`, tone: 'warning' }]} />
      {/if}

      <Card padding="lg">
        <Stack gap="md">
          <span class="cs-hint">Select an opportunity to create a costing sheet — customer details and any structured line items are pre-filled automatically.</span>
          <label class="k-field">
            <span class="k-field-label">Opportunity / RFQ</span>
            <select class="k-input" value={vm.selectedOpportunityId} onchange={(e) => void vm.selectOpportunity(e.currentTarget.value)}>
              <option value="">— Select an opportunity —</option>
              {#each vm.opportunities as opp (opp.id)}
                <option value={opp.id}>{opportunityLabel(opp)}</option>
              {/each}
            </select>
          </label>
          <Row justify="start">
            <Button onclick={() => vm.startBlank()}>Start Blank Costing</Button>
          </Row>
        </Stack>
      </Card>

      <Card padding="none">
        {#if vm.opportunities.length === 0}
          <EmptyState message="No opportunities found. Create one in the Sales Hub first, or start a blank costing." />
        {:else}
          <DataTable
            columns={opportunityColumns}
            rows={vm.previewOpportunities}
            id={oppId}
            status={opportunityStatus}
            selectedId={vm.selectedOpportunityId}
            onSelect={(r) => void vm.selectOpportunity(r.id)}
          />
        {/if}
      </Card>

      {#if vm.recentSheets.length > 0}
        <Card padding="none">
          <DataTable columns={recentColumns} rows={vm.recentSheets} id={recentId} />
        </Card>
      {/if}
    </Stack>
  {:else}
    <Stack gap="lg">
      {#if vm.isRFQOpportunity && vm.revisions.length > 0}
        <Card padding="none">
          <DataTable
            columns={revisionColumns}
            rows={vm.revisions}
            id={revisionId}
            status={revisionStatus}
            selectedId={vm.selectedRevisionId != null ? String(vm.selectedRevisionId) : null}
            onSelect={(r) => vm.selectRevision(r)}
          />
        </Card>
        <Row justify="end" gap="sm">
          <Button onclick={() => vm.saveCosting()} disabled={vm.savingCosting}>{vm.savingCosting ? 'Saving…' : '+ New Revision'}</Button>
          {#if vm.selectedRevisionId != null && !vm.revisions.find((r) => r.id === vm.selectedRevisionId)?.isActive}
            {@const selected = vm.revisions.find((r) => r.id === vm.selectedRevisionId)}
            {#if selected}
              <Button variant="primary" onclick={() => vm.setActiveRevision(selected)}>Make Current</Button>
            {/if}
          {/if}
        </Row>
      {/if}
      {#if vm.revisionError}
        <CalloutWidget items={[{ label: 'Revision', text: vm.revisionError, tone: 'danger' }]} />
      {/if}
      {#if vm.saveError}
        <CalloutWidget items={[{ label: 'Save', text: vm.saveError, tone: 'danger' }]} />
      {/if}
      {#if vm.exportError}
        <CalloutWidget items={[{ label: 'Export', text: vm.exportError, tone: 'danger' }]} />
      {/if}
      {#if vm.lastSavedOfferNumber}
        <CalloutWidget items={[{ label: 'Saved', text: `Offer ${vm.lastSavedOfferNumber} saved.`, tone: 'success' }]} />
      {/if}

      <Grid min="640px" gap="lg">
        <Stack gap="lg">
          <Card padding="lg">
            <Stack gap="md">
              <FormGrid columns={3}>
                <label class="k-field">
                  <span class="k-field-label">Customer *</span>
                  <select class="k-input" value={vm.header.customerId} onchange={(e) => vm.selectCustomerById(e.currentTarget.value)}>
                    <option value="">Select customer…</option>
                    {#each vm.customers as c (c.id)}
                      <option value={c.id}>{c.businessName || '(unnamed)'}</option>
                    {/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Contact Person</span>
                  <input class="k-input" type="text" bind:value={vm.header.contactPerson} placeholder="Name" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">RFQ Reference</span>
                  <input class="k-input" type="text" bind:value={vm.header.rfqReference} placeholder="RFQ / enquiry ref" />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Division</span>
                  <select class="k-input" bind:value={vm.header.division}>
                    {#each vm.divisionOptions as opt (opt.value)}<option value={opt.value}>{opt.label}</option>{/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Date</span>
                  <input class="k-input" type="date" bind:value={vm.header.date} />
                </label>
                <label class="k-field">
                  <span class="k-field-label">Prepared By *</span>
                  <select class="k-input" bind:value={vm.header.preparedBy}>
                    <option value="">Select…</option>
                    {#each vm.preparedByOptions as name (name)}<option value={name}>{name}</option>{/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Doc Type</span>
                  <select class="k-input" bind:value={vm.header.quoteType}>
                    {#each DOC_TYPE_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Payment Terms</span>
                  <select class="k-input" bind:value={vm.header.paymentTerms}>
                    {#each PAYMENT_TERMS_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Delivery Terms</span>
                  <select class="k-input" bind:value={vm.header.deliveryTerms}>
                    {#each [...new Set([vm.header.deliveryTerms, ...DELIVERY_TERMS_OPTIONS])] as opt (opt)}<option value={opt}>{opt}</option>{/each}
                  </select>
                </label>
                <label class="k-field">
                  <span class="k-field-label">Est. Delivery</span>
                  <select class="k-input" bind:value={vm.header.estDelivery}>
                    {#each EST_DELIVERY_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                  </select>
                </label>
                <label class="k-field k-field-wide">
                  <span class="k-field-label">Subject</span>
                  <input class="k-input" type="text" bind:value={vm.header.subject} placeholder="Subject line for customer PDF" />
                </label>
              </FormGrid>

              <button class="cs-toggle" onclick={() => (vm.showAdvanced = !vm.showAdvanced)}>
                {vm.showAdvanced ? '▾' : '▸'} Compliance, Certificates &amp; VAT
              </button>

              {#if vm.showAdvanced}
                <FormGrid columns={3}>
                  <label class="k-field">
                    <span class="k-field-label">Order Type</span>
                    <select class="k-input" bind:value={vm.header.orderType}>
                      {#each ORDER_TYPE_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Origin</span>
                    <select class="k-input" bind:value={vm.header.countryOfOrigin}>
                      {#each COUNTRY_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">COC/COO</span>
                    <select class="k-input" bind:value={vm.header.cocCoo}>
                      {#each CERTIFICATE_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Test Cert</span>
                    <select class="k-input" bind:value={vm.header.testCertificate}>
                      {#each CERTIFICATE_OPTIONS as opt (opt)}<option value={opt}>{opt}</option>{/each}
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Install</span>
                    <select class="k-input" bind:value={vm.header.installation}>
                      <option value="Yes">Yes</option><option value="No">No</option>
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Commission</span>
                    <select class="k-input" bind:value={vm.header.commissioning}>
                      <option value="Yes">Yes</option><option value="No">No</option>
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Place of Supply</span>
                    <select class="k-input" bind:value={vm.header.placeOfSupply}>
                      <option value="Kingdom of Bahrain">Kingdom of Bahrain</option>
                      <option value="GCC">GCC Member State</option>
                      <option value="Export">Export (Outside GCC)</option>
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Tax Category</span>
                    <select class="k-input" bind:value={vm.header.taxCategory}>
                      <option value="Standard">Standard Rate</option>
                      <option value="Zero-rated">Zero-rated</option>
                      <option value="Exempt">Exempt</option>
                      <option value="Out-of-scope">Out of Scope</option>
                    </select>
                  </label>
                  <label class="k-field">
                    <span class="k-field-label">Customer TRN</span>
                    <input class="k-input" type="text" bind:value={vm.header.customerTRN} placeholder="Tax Reg. Number" />
                  </label>
                </FormGrid>
              {/if}
            </Stack>
          </Card>

          <Card padding="lg">
            <LineItemsEditor
              columns={lineColumns}
              rows={vm.lines}
              createRow={() => vm.blankLine()}
              onAdd={() => vm.addLine()}
              onRemove={(i) => vm.removeLine(i)}
              minRows={1}
              maxRows={MAX_COSTING_LINE_ITEMS}
              footer={lineFooter}
            />
          </Card>
        </Stack>

        <Stack gap="lg">
          <Card padding="lg">
            <Stack gap="md">
              <label class="k-field">
                <span class="k-field-label">Discount (BHD)</span>
                <input class="k-input" type="number" step="0.001" min="0" bind:value={vm.discount} />
              </label>
              <label class="k-field">
                <span class="k-field-label" title="Additional charges not shown on the quotation">Hidden Charges (BHD)</span>
                <input class="k-input" type="number" step="0.001" min="0" bind:value={vm.hiddenCharges} />
              </label>
              <label class="k-field">
                <span class="k-field-label">VAT %</span>
                <input class="k-input" type="number" step="0.5" min="0" max="100" bind:value={vm.vatRate} />
              </label>

              <StatTileGrid
                sections={[
                  {
                    title: 'Totals',
                    items: [
                      { label: 'Subtotal', value: vm.subtotal, content: 'money' },
                      { label: 'Grand Total', value: vm.grandTotal, content: 'money' },
                    ],
                  },
                  {
                    title: 'Profit Analysis',
                    items: [
                      { label: 'PO Expected', value: vm.netAmount, content: 'money' },
                      { label: 'PH Cost', value: vm.totalCost, content: 'money' },
                      { label: 'Profit', value: vm.profit, content: 'money', tone: profitTone(vm.profitPercent) },
                      { label: 'Profit %', value: `${vm.profitPercent.toFixed(1)}%`, content: 'text', tone: profitTone(vm.profitPercent) },
                    ],
                  },
                ]}
              />
            </Stack>
          </Card>

          <Card padding="lg">
            <Stack gap="sm">
              <span class="cs-section-title">Terms &amp; Conditions</span>
              <span class="cs-hint">Printed on a separate page in PDF exports.</span>
              <textarea class="k-input k-input-area" rows="8" bind:value={vm.termsAndConditions} placeholder="Enter terms and conditions…"></textarea>
            </Stack>
          </Card>
        </Stack>
      </Grid>
    </Stack>
  {/if}
</PageShell>

{#if vm.confirmOverwriteOpen}
  <ConfirmDialog
    title="Overwrite existing offer?"
    message={`This will overwrite offer ${vm.linkedOfferNumber} — continue?`}
    confirmLabel="Overwrite"
    danger={false}
    onConfirm={() => vm.confirmSaveAsOffer()}
    onCancel={() => vm.cancelSaveAsOffer()}
  />
{/if}

<style>
  /* Typography only (L1) — native form controls use the kernel-owned
   * k-field/k-field-label/k-input classes (single-source in styles/kernel.css);
   * spacing/grouping is FormGrid/Stack/Row's job. */
  .cs-hint {
    font-size: var(--meta-size);
    color: var(--text-secondary);
    overflow-wrap: break-word;
  }
  .cs-toggle {
    font-family: var(--font-ui);
    font-size: var(--modal-body-size);
    font-weight: 600;
    color: var(--text-secondary);
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0;
  }
  .cs-section-title {
    font-family: var(--font-display);
    font-size: var(--section-title-size);
    font-weight: var(--section-title-weight);
    overflow-wrap: break-word;
  }
</style>
