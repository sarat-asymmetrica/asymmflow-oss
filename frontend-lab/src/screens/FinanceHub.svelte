<script lang="ts">
  /* FinanceHub — a K5 tab-navigator: one PageShell hosting the finance screens
   * as TabShell tabs. Pure composition over ALREADY-BUILT kernel screens
   * (ledgers via DocumentLedger, the bespoke Payroll/BankRecon/BookBankRecon,
   * the finance dashboard via Hub) rendered `embedded` so they share the hub's
   * single header + scroll region. The old FinanceHub's division selector is
   * DEFERRED — the child screens carry their own division filters where
   * relevant; a hub-level company scope needs a prop contract those screens
   * don't expose yet (K5 polish / INTEG). */
  import PageShell from '$kernel/primitives/PageShell.svelte'
  import TabShell from '$kernel/primitives/TabShell.svelte'
  import DocumentLedger from '$kernel/archetypes/DocumentLedger.svelte'
  import Hub from '$kernel/archetypes/Hub.svelte'
  import Payroll from './Payroll.svelte'
  import BankReconciliation from './BankReconciliation.svelte'
  import BookBankRecon from './BookBankRecon.svelte'
  import { invoicesDescriptor } from './invoices.descriptor'
  import { paymentsDescriptor } from './payments.descriptor'
  import { supplierInvoicesDescriptor } from './supplier-invoices.descriptor'
  import { supplierPaymentsDescriptor } from './supplier-payments.descriptor'
  import { expensesDescriptor } from './expenses.descriptor'
  import { chequeRegisterDescriptor } from './cheque-register.descriptor'
  import { fxRevaluationDescriptor } from './fx-revaluation.descriptor'
  import { auditTrailDescriptor } from './audit-trail.descriptor'
  import { financeOverviewDescriptor } from './dashboards/finance-overview.hub'
  import { navigate } from '../stores/navigation.svelte'
  import type { NavIntent } from '$kernel/hub'

  let active = $state('overview')
  const nav = (intent: NavIntent) => navigate(intent.key, intent.query ? { query: intent.query } : undefined)
</script>

{#snippet t_overview()}<Hub descriptor={financeOverviewDescriptor} navigate={nav} embedded />{/snippet}
{#snippet t_invoices()}<DocumentLedger descriptor={invoicesDescriptor} embedded />{/snippet}
{#snippet t_payments()}<DocumentLedger descriptor={paymentsDescriptor} embedded />{/snippet}
{#snippet t_supplier_invoices()}<DocumentLedger descriptor={supplierInvoicesDescriptor} embedded />{/snippet}
{#snippet t_supplier_payments()}<DocumentLedger descriptor={supplierPaymentsDescriptor} embedded />{/snippet}
{#snippet t_expenses()}<DocumentLedger descriptor={expensesDescriptor} embedded />{/snippet}
{#snippet t_payroll()}<Payroll embedded />{/snippet}
{#snippet t_bank_recon()}<BankReconciliation embedded />{/snippet}
{#snippet t_cheques()}<DocumentLedger descriptor={chequeRegisterDescriptor} embedded />{/snippet}
{#snippet t_book_bank()}<BookBankRecon embedded />{/snippet}
{#snippet t_fx()}<DocumentLedger descriptor={fxRevaluationDescriptor} embedded />{/snippet}
{#snippet t_audit()}<DocumentLedger descriptor={auditTrailDescriptor} embedded />{/snippet}

<PageShell title="Finance" subtitle="Invoices, payments, payroll, reconciliation, and the finance ledgers.">
  <TabShell
    activeKey={active}
    onSelect={(k) => (active = k)}
    tabs={[
      { key: 'overview', label: 'Overview', content: t_overview },
      { key: 'invoices', label: 'Invoices', content: t_invoices },
      { key: 'payments', label: 'Payments', content: t_payments },
      { key: 'supplier-invoices', label: 'Supplier Invoices', content: t_supplier_invoices },
      { key: 'supplier-payments', label: 'Supplier Payments', content: t_supplier_payments },
      { key: 'expenses', label: 'Expenses', content: t_expenses },
      { key: 'payroll', label: 'Payroll', content: t_payroll },
      { key: 'bank-recon', label: 'Bank Recon', content: t_bank_recon },
      { key: 'cheques', label: 'Cheques', content: t_cheques },
      { key: 'book-bank', label: 'Book vs Bank', content: t_book_bank },
      { key: 'fx', label: 'FX Revaluation', content: t_fx },
      { key: 'audit', label: 'Audit Trail', content: t_audit },
    ]}
  />
</PageShell>
