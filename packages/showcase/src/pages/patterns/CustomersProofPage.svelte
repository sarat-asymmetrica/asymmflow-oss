<script lang="ts">
  /**
   * CustomersProofPage — THE PROOF.
   *
   * Re-imagines the legacy CustomersScreen.svelte (~900 LOC, wailsjs deps,
   * raw CSS variables, ad-hoc modal) as a composition of @asymmflow/ui + patterns.
   *
   * What the legacy screen did — preserved here:
   *   - KPI row: total customers / active / outstanding BHD
   *   - Searchable sortable customer table with grade badges (StatusBadge)
   *   - Payment stats in tabular numerals
   *   - Row click opens a SplitDetail drawer with customer 360 + recent invoices
   *   - "New Customer" button opens a Modal with a FormGroup form
   *
   * Data: 20 rows of realistic Gulf-trading mock customers.
   * Constitution: all tokens, all regimes, full keyboard, full ARIA.
   */

  import {
    DataShell,
    SplitDetail,
    PageHeader,
  } from '@asymmflow/patterns';

  import {
    KPICard,
    StatusBadge,
    Modal,
    Button,
    FormGroup,
    Input,
    Select,
    DataTable,
  } from '@asymmflow/ui';

  import type { Column, CellContext } from '@asymmflow/ui';
  import type { StatusKind, SelectOption } from '@asymmflow/ui';

  // ─── Types ──────────────────────────────────────────────────────────────────

  interface Customer {
    id: string;
    customer_id: string;
    business_name: string;
    customer_type: string;
    payment_grade: 'A' | 'B' | 'C' | 'D';
    status: 'Active' | 'Inactive' | 'Blacklisted';
    city: string;
    country: string;
    primary_phone: string;
    primary_email: string;
    credit_limit_bhd: number;
    outstanding_bhd: number;
    avg_payment_days: number;
    total_orders_value: number;
    total_orders_count: number;
    relation_years: number;
    industry: string;
    cr_number: string;
    payment_terms_days: number;
  }

  interface RecentInvoice {
    id: string;
    number: string;
    date: string;
    amount: number;
    status: 'Paid' | 'Outstanding' | 'Overdue';
  }

  // ─── Mock: 20 Gulf-trading customers ─────────────────────────────────────────

  const customers: Customer[] = [
    { id: '01', customer_id: 'CUST-GEW001', business_name: 'Gulf Equipment Trading WLL',       customer_type: 'EC',  payment_grade: 'A', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0001', primary_email: 'accounts@getwll.bh',   credit_limit_bhd: 50_000, outstanding_bhd: 12_450, avg_payment_days: 24, total_orders_value: 214_300, total_orders_count: 38, relation_years: 7, industry: 'Industrial Equipment', cr_number: 'CR-98001', payment_terms_days: 30 },
    { id: '02', customer_id: 'CUST-AMC002', business_name: 'Al Moayyed Contracting Group',     customer_type: 'EP',  payment_grade: 'A', status: 'Active',     city: 'Riffa',      country: 'BH', primary_phone: '+973 1723 0002', primary_email: 'fin@almoayyed.bh',     credit_limit_bhd: 75_000, outstanding_bhd:  4_875, avg_payment_days: 18, total_orders_value: 389_500, total_orders_count: 62, relation_years:12, industry: 'Construction',         cr_number: 'CR-98002', payment_terms_days: 30 },
    { id: '03', customer_id: 'CUST-NPC003', business_name: 'National Petroleum Co.',    customer_type: 'EC',  payment_grade: 'A', status: 'Active',     city: 'Awali',      country: 'BH', primary_phone: '+973 1723 0003', primary_email: 'ap@natpetro.example',          credit_limit_bhd:200_000, outstanding_bhd:  0,      avg_payment_days: 12, total_orders_value: 978_200, total_orders_count:144, relation_years:18, industry: 'Oil & Gas',            cr_number: 'CR-98003', payment_terms_days: 60 },
    { id: '04', customer_id: 'CUST-NMC004', business_name: 'National Motor Company BSC',       customer_type: 'EC',  payment_grade: 'C', status: 'Active',     city: 'Salmabad',   country: 'BH', primary_phone: '+973 1723 0004', primary_email: 'accounts@nmc.bh',      credit_limit_bhd: 15_000, outstanding_bhd:  2_340, avg_payment_days: 41, total_orders_value:  87_450, total_orders_count: 23, relation_years: 4, industry: 'Automotive',           cr_number: 'CR-98004', payment_terms_days: 30 },
    { id: '05', customer_id: 'CUST-ZBH005', business_name: 'Zain Bahrain BSC',                 customer_type: 'EC',  payment_grade: 'B', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0005', primary_email: 'payables@zain.bh',     credit_limit_bhd: 40_000, outstanding_bhd:  8_910, avg_payment_days: 28, total_orders_value: 156_800, total_orders_count: 41, relation_years: 9, industry: 'Telecommunications',   cr_number: 'CR-98005', payment_terms_days: 30 },
    { id: '06', customer_id: 'CUST-GAG006', business_name: 'Gulf Air Group Holding Co.',       customer_type: 'EC',  payment_grade: 'B', status: 'Active',     city: 'Eastside',   country: 'BH', primary_phone: '+973 1723 0006', primary_email: 'finance@gulfair.bh',   credit_limit_bhd: 60_000, outstanding_bhd: 19_500, avg_payment_days: 32, total_orders_value: 243_100, total_orders_count: 55, relation_years:11, industry: 'Aviation',             cr_number: 'CR-98006', payment_terms_days: 45 },
    { id: '07', customer_id: 'CUST-ITH007', business_name: 'Ithmaar Bank BSC',                 customer_type: 'EC',  payment_grade: 'D', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0007', primary_email: 'ap@ithmaar.bh',        credit_limit_bhd: 10_000, outstanding_bhd:  5_632, avg_payment_days: 67, total_orders_value:  44_200, total_orders_count: 14, relation_years: 3, industry: 'Banking',              cr_number: 'CR-98007', payment_terms_days: 30 },
    { id: '08', customer_id: 'CUST-GSC008', business_name: 'Gulf Smelting Co.',          customer_type: 'EC',  payment_grade: 'A', status: 'Active',     city: 'Hidd',       country: 'BH', primary_phone: '+973 1723 0008', primary_email: 'finance@gulfsmelting.example',      credit_limit_bhd:150_000, outstanding_bhd:  0,      avg_payment_days:  9, total_orders_value: 712_400, total_orders_count: 98, relation_years:15, industry: 'Manufacturing',        cr_number: 'CR-98008', payment_terms_days: 60 },
    { id: '09', customer_id: 'CUST-BAT009', business_name: 'Batelco Group',                    customer_type: 'EC',  payment_grade: 'A', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0009', primary_email: 'accounts@batelco.bh',  credit_limit_bhd: 35_000, outstanding_bhd:  3_221, avg_payment_days: 21, total_orders_value: 189_600, total_orders_count: 47, relation_years:10, industry: 'Telecommunications',   cr_number: 'CR-98009', payment_terms_days: 30 },
    { id: '10', customer_id: 'CUST-ESK010', business_name: 'Eskan Bank',                       customer_type: 'EC',  payment_grade: 'C', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0010', primary_email: 'ap@eskanbank.bh',      credit_limit_bhd: 20_000, outstanding_bhd:  6_750, avg_payment_days: 45, total_orders_value:  98_300, total_orders_count: 29, relation_years: 6, industry: 'Banking',              cr_number: 'CR-98010', payment_terms_days: 30 },
    { id: '11', customer_id: 'CUST-ABC011', business_name: 'Arab Banking Corporation (ABC)',    customer_type: 'EC',  payment_grade: 'D', status: 'Inactive',   city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0011', primary_email: 'finance@abc.bh',       credit_limit_bhd:  5_000, outstanding_bhd: 22_100, avg_payment_days: 88, total_orders_value:  67_900, total_orders_count: 17, relation_years: 2, industry: 'Banking',              cr_number: 'CR-98011', payment_terms_days: 30 },
    { id: '12', customer_id: 'CUST-SEF012', business_name: 'Seef Properties WLL',              customer_type: 'SI',  payment_grade: 'B', status: 'Active',     city: 'Seef',       country: 'BH', primary_phone: '+973 1723 0012', primary_email: 'ap@seefproperties.bh', credit_limit_bhd: 30_000, outstanding_bhd:  9_450, avg_payment_days: 35, total_orders_value: 134_700, total_orders_count: 33, relation_years: 8, industry: 'Real Estate',          cr_number: 'CR-98012', payment_terms_days: 45 },
    { id: '13', customer_id: 'CUST-GFH013', business_name: 'GFH Financial Group',              customer_type: 'EC',  payment_grade: 'B', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0013', primary_email: 'finance@gfh.bh',       credit_limit_bhd: 45_000, outstanding_bhd:  7_200, avg_payment_days: 29, total_orders_value: 178_500, total_orders_count: 39, relation_years: 5, industry: 'Finance',              cr_number: 'CR-98013', payment_terms_days: 30 },
    { id: '14', customer_id: 'CUST-KGG014', business_name: 'Khaleeji Commercial Bank',         customer_type: 'EC',  payment_grade: 'C', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0014', primary_email: 'ap@khaleejibank.bh',   credit_limit_bhd: 18_000, outstanding_bhd:  3_890, avg_payment_days: 52, total_orders_value:  76_200, total_orders_count: 21, relation_years: 3, industry: 'Banking',              cr_number: 'CR-98014', payment_terms_days: 30 },
    { id: '15', customer_id: 'CUST-UNB015', business_name: 'United Nations Bahrain (UNOB)',    customer_type: 'NR',  payment_grade: 'A', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0015', primary_email: 'fin@unob.bh',          credit_limit_bhd: 25_000, outstanding_bhd:  0,      avg_payment_days: 15, total_orders_value: 112_300, total_orders_count: 28, relation_years: 6, industry: 'Government',           cr_number: 'CR-98015', payment_terms_days: 45 },
    { id: '16', customer_id: 'CUST-BAB016', business_name: 'Bahrain Airport Services (BAS)',   customer_type: 'SP',  payment_grade: 'B', status: 'Active',     city: 'Eastside',   country: 'BH', primary_phone: '+973 1723 0016', primary_email: 'accounts@bas.bh',      credit_limit_bhd: 22_000, outstanding_bhd:  4_100, avg_payment_days: 33, total_orders_value:  98_600, total_orders_count: 26, relation_years: 4, industry: 'Aviation Services',    cr_number: 'CR-98016', payment_terms_days: 30 },
    { id: '17', customer_id: 'CUST-GHC017', business_name: 'Gulf Hotels Group BSC',            customer_type: 'EC',  payment_grade: 'B', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0017', primary_email: 'finance@gulfhotels.bh', credit_limit_bhd: 28_000, outstanding_bhd:  6_300, avg_payment_days: 27, total_orders_value: 143_900, total_orders_count: 35, relation_years: 7, industry: 'Hospitality',          cr_number: 'CR-98017', payment_terms_days: 30 },
    { id: '18', customer_id: 'CUST-MNT018', business_name: 'Midal Cables Ltd.',                customer_type: 'EP',  payment_grade: 'A', status: 'Active',     city: 'Hidd',       country: 'BH', primary_phone: '+973 1723 0018', primary_email: 'ap@midalcables.bh',    credit_limit_bhd: 55_000, outstanding_bhd:  1_820, avg_payment_days: 19, total_orders_value: 267_400, total_orders_count: 51, relation_years: 9, industry: 'Manufacturing',        cr_number: 'CR-98018', payment_terms_days: 30 },
    { id: '19', customer_id: 'CUST-ABB019', business_name: 'Arabian Industries LLC',            customer_type: 'SI',  payment_grade: 'C', status: 'Inactive',   city: 'Salmabad',   country: 'BH', primary_phone: '+973 1723 0019', primary_email: 'accounts@arabinds.bh', credit_limit_bhd:  8_000, outstanding_bhd:  2_450, avg_payment_days: 58, total_orders_value:  49_100, total_orders_count: 16, relation_years: 2, industry: 'System Integration',   cr_number: 'CR-98019', payment_terms_days: 30 },
    { id: '20', customer_id: 'CUST-PHT020', business_name: 'Acme Instrumentation International WLL',     customer_type: 'PH',  payment_grade: 'A', status: 'Active',     city: 'Manama',     country: 'BH', primary_phone: '+973 1723 0020', primary_email: 'accounts@phtrading.bh', credit_limit_bhd:100_000, outstanding_bhd:  0,      avg_payment_days: 10, total_orders_value: 521_800, total_orders_count: 87, relation_years:14, industry: 'Trading',              cr_number: 'CR-98020', payment_terms_days: 60 },
  ];

  // ─── Per-customer recent invoices (keyed by customer id) ──────────────────────

  const recentInvoicesMap: Record<string, RecentInvoice[]> = {
    '01': [
      { id: 'ri-01a', number: 'INV-2026-0411', date: '2026-05-01', amount: 12_450.000, status: 'Outstanding' },
      { id: 'ri-01b', number: 'INV-2026-0398', date: '2026-04-10', amount:  7_200.500, status: 'Paid' },
      { id: 'ri-01c', number: 'INV-2026-0378', date: '2026-03-22', amount:  4_100.000, status: 'Paid' },
    ],
    '06': [
      { id: 'ri-06a', number: 'INV-2026-0406', date: '2026-04-15', amount: 19_500.000, status: 'Outstanding' },
      { id: 'ri-06b', number: 'INV-2026-0383', date: '2026-03-25', amount:  8_750.000, status: 'Paid' },
    ],
    '07': [
      { id: 'ri-07a', number: 'INV-2026-0405', date: '2026-04-12', amount:  5_632.250, status: 'Overdue' },
    ],
    '11': [
      { id: 'ri-11a', number: 'INV-2026-0401', date: '2026-04-01', amount: 22_100.000, status: 'Overdue' },
      { id: 'ri-11b', number: 'INV-2026-0372', date: '2026-03-01', amount:  9_400.000, status: 'Overdue' },
    ],
  };

  function getRecentInvoices(id: string): RecentInvoice[] {
    return recentInvoicesMap[id] ?? [];
  }

  // ─── KPIs ─────────────────────────────────────────────────────────────────────

  const totalCustomers = $derived(customers.length);
  const activeCount    = $derived(customers.filter((c) => c.status === 'Active').length);
  const totalOutstanding = $derived(
    customers.reduce((s, c) => s + c.outstanding_bhd, 0),
  );

  // ─── Customer columns ─────────────────────────────────────────────────────────

  function fmtBHD(v: unknown): string {
    if (v == null || v === 0) return 'BHD 0.000';
    return `BHD ${(v as number).toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 })}`;
  }

  const columns: Column<Customer>[] = [
    { key: 'business_name',   header: 'Company',         sortable: true },
    { key: 'customer_type',   header: 'Type',            width: '64px' },
    { key: 'payment_grade',   header: 'Grade',           width: '72px' },
    { key: 'status',          header: 'Status',          width: '100px' },
    { key: 'outstanding_bhd', header: 'Outstanding',     numeric: true,  sortable: true, width: '140px', format: fmtBHD },
    { key: 'avg_payment_days',header: 'Avg Days',        numeric: true,  sortable: true, width: '90px' },
    { key: 'city',            header: 'City',            width: '100px' },
  ];

  // StatusBadge StatusKind = 'done' | 'pending' | 'attention' | 'failed'
  const gradeStatus: Record<string, StatusKind> = { A: 'done', B: 'pending', C: 'attention', D: 'failed' };
  const customerStatus: Record<string, StatusKind> = { Active: 'done', Inactive: 'pending', Blacklisted: 'failed' };

  // ─── Selection / SplitDetail ──────────────────────────────────────────────────

  let selectedId = $state<string | null>(null);

  const selectedCustomer = $derived(
    selectedId ? customers.find((c) => c.id === selectedId) ?? null : null,
  );

  // ─── Recent invoice columns ───────────────────────────────────────────────────

  const invoiceCols: Column<RecentInvoice>[] = [
    { key: 'number', header: 'Invoice',  width: '140px' },
    { key: 'date',   header: 'Date',     width: '100px' },
    { key: 'amount', header: 'Amount',   numeric: true, width: '120px', format: fmtBHD },
    { key: 'status', header: 'Status',   width: '100px' },
  ];

  // ─── New Customer modal ───────────────────────────────────────────────────────

  let showCreate = $state(false);

  interface NewCustomerForm {
    business_name: string;
    customer_type: string;
    payment_grade: string;
    status: string;
    primary_phone: string;
    primary_email: string;
    city: string;
    country: string;
    cr_number: string;
    industry: string;
    payment_terms: string;
    credit_limit: string;
  }

  function emptyForm(): NewCustomerForm {
    return {
      business_name: '',
      customer_type: '',
      payment_grade: '',
      status: 'Active',
      primary_phone: '',
      primary_email: '',
      city: '',
      country: 'Bahrain',
      cr_number: '',
      industry: '',
      payment_terms: '',
      credit_limit: '',
    };
  }

  let form = $state<NewCustomerForm>(emptyForm());
  let formError = $state('');

  function handleCreate() {
    if (!form.business_name.trim()) {
      formError = 'Company name is required.';
      return;
    }
    formError = '';
    // In a real screen, call CreateCustomer(data); here we just close.
    showCreate = false;
    form = emptyForm();
  }

  const typeOptions: SelectOption[] = [
    { value: 'EC', label: 'End Customer' },
    { value: 'CO', label: 'Consultant' },
    { value: 'EP', label: 'Engineering / EPC' },
    { value: 'IR', label: 'Intl Reseller' },
    { value: 'NR', label: 'Natl Reseller' },
    { value: 'PB', label: 'Plant Builder' },
    { value: 'SI', label: 'System Integrator' },
    { value: 'SP', label: 'Service Provider' },
    { value: 'PH', label: 'Acme Instrumentation' },
  ];

  const gradeOptions: SelectOption[] = [
    { value: 'A', label: 'A — Premium' },
    { value: 'B', label: 'B — Standard' },
    { value: 'C', label: 'C — Watch' },
    { value: 'D', label: 'D — Restricted' },
  ];

  const statusOptions: SelectOption[] = [
    { value: 'Active', label: 'Active' },
    { value: 'Inactive', label: 'Inactive' },
    { value: 'Blacklisted', label: 'Blacklisted' },
  ];

  const termsOptions: SelectOption[] = [
    { value: 'cash', label: 'Cash' },
    { value: 'pdc',  label: 'PDC (Post-Dated Cheque)' },
    { value: 'net30', label: 'Net 30' },
    { value: 'net60', label: 'Net 60' },
    { value: 'net90', label: 'Net 90' },
  ];
</script>

<div class="proof-page">

  <!-- ─── Page header ────────────────────────────────────────────────────── -->
  <PageHeader
    title="Customers"
    breadcrumb={[{ label: 'Dashboard', href: '#' }, { label: 'Customers' }]}
  >
    {#snippet meta()}
      <span class="af-meta">Directory & Relationships</span>
    {/snippet}

    {#snippet actions()}
      <Button variant="primary" onclick={() => (showCreate = true)}>
        New Customer
      </Button>
    {/snippet}
  </PageHeader>

  <!-- ─── KPI row ────────────────────────────────────────────────────────── -->
  <div class="kpi-row" aria-label="Key metrics">
    <KPICard
      label="Total Customers"
      value={String(totalCustomers)}
      meta="registered counterparties"
    />
    <KPICard
      label="Active"
      value={String(activeCount)}
      meta="{activeCount} of {totalCustomers} customers"
    />
    <KPICard
      label="Outstanding (BHD)"
      value="BHD {totalOutstanding.toLocaleString('en-US', { minimumFractionDigits: 3, maximumFractionDigits: 3 })}"
      meta={totalOutstanding > 50_000 ? 'Above threshold — review' : 'Within normal range'}
    />
  </div>

  <!-- ─── Master-detail shell ────────────────────────────────────────────── -->
  <SplitDetail
    bind:selectedId
    drawerTitle={selectedCustomer?.business_name ?? 'Customer Detail'}
    drawerSize="lg"
  >
    {#snippet children()}
      <DataShell
        title="Customers"
        data={customers}
        {columns}
        searchableKeys={['business_name', 'customer_type', 'city', 'industry', 'cr_number']}
        pageSize={12}
        onRowClick={(row) => (selectedId = row.id)}
        label="Customer directory"
        tableMaxHeight="520px"
      >
        {#snippet cell(ctx)}
          {#if ctx.column.key === 'payment_grade'}
            <StatusBadge
              status={gradeStatus[ctx.value as string] ?? 'neutral'}
              label="Grade {ctx.value}"
            />
          {:else if ctx.column.key === 'status'}
            <StatusBadge
              status={customerStatus[ctx.value as string] ?? 'neutral'}
              label={ctx.value as string}
            />
          {:else}
            {ctx.formatted}
          {/if}
        {/snippet}

        {#snippet rowActions(row)}
          <button
            class="row-act-btn"
            aria-label="View {row.business_name}"
            onpointerdown={(e) => { e.stopPropagation(); selectedId = row.id; }}
          >
            View
          </button>
        {/snippet}
      </DataShell>
    {/snippet}

    <!-- ─── Drawer detail: customer 360 ──────────────────────────────── -->
    {#snippet detail(id)}
      {@const c = customers.find((x) => x.id === id)}
      {#if c}
        <div class="detail-body">

          <!-- Section: Identity -->
          <div class="detail-section">
            <div class="detail-section-label af-label">Identity</div>
            <div class="detail-grid">
              <div class="detail-field">
                <span class="detail-field-label af-label">Customer ID</span>
                <span class="detail-field-value af-numeric">{c.customer_id}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">CR Number</span>
                <span class="detail-field-value">{c.cr_number}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Industry</span>
                <span class="detail-field-value">{c.industry}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Relation</span>
                <span class="detail-field-value af-numeric">{c.relation_years} yrs</span>
              </div>
            </div>
          </div>

          <!-- Section: Classification -->
          <div class="detail-section">
            <div class="detail-section-label af-label">Classification</div>
            <div class="detail-badges">
              <StatusBadge status={gradeStatus[c.payment_grade] ?? 'neutral'} label="Grade {c.payment_grade}" />
              <StatusBadge status={customerStatus[c.status] ?? 'neutral'} label={c.status} />
              <span class="detail-type-chip">{c.customer_type}</span>
            </div>
          </div>

          <!-- Section: Contact -->
          <div class="detail-section">
            <div class="detail-section-label af-label">Contact</div>
            <div class="detail-grid">
              <div class="detail-field">
                <span class="detail-field-label af-label">Phone</span>
                <a class="detail-field-value detail-link" href="tel:{c.primary_phone}">{c.primary_phone}</a>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Email</span>
                <a class="detail-field-value detail-link" href="mailto:{c.primary_email}">{c.primary_email}</a>
              </div>
              <div class="detail-field detail-field--full">
                <span class="detail-field-label af-label">Location</span>
                <span class="detail-field-value">{c.city}, {c.country}</span>
              </div>
            </div>
          </div>

          <!-- Section: Financials -->
          <div class="detail-section">
            <div class="detail-section-label af-label">Financials</div>
            <div class="detail-grid">
              <div class="detail-field">
                <span class="detail-field-label af-label">Credit Limit</span>
                <span class="detail-field-value af-numeric">{fmtBHD(c.credit_limit_bhd)}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Outstanding</span>
                <span class="detail-field-value af-numeric {c.outstanding_bhd > 0 ? 'val--warn' : ''}">{fmtBHD(c.outstanding_bhd)}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Avg Payment</span>
                <span class="detail-field-value af-numeric">{c.avg_payment_days} days</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Terms</span>
                <span class="detail-field-value af-numeric">Net {c.payment_terms_days}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Total Orders</span>
                <span class="detail-field-value af-numeric">{fmtBHD(c.total_orders_value)}</span>
              </div>
              <div class="detail-field">
                <span class="detail-field-label af-label">Order Count</span>
                <span class="detail-field-value af-numeric">{c.total_orders_count}</span>
              </div>
            </div>
          </div>

          <!-- Section: Recent Invoices mini-table -->
          {#if getRecentInvoices(c.id).length > 0}
            {@const recent = getRecentInvoices(c.id)}
            <div class="detail-section">
              <div class="detail-section-label af-label">Recent Invoices</div>
              <DataTable
                data={recent}
                columns={invoiceCols}
                rowId={(r) => r.id}
                label="Recent invoices for {c.business_name}"
                maxHeight="220px"
              >
                {#snippet cell(ctx)}
                  {#if ctx.column.key === 'status'}
                    {@const s = ctx.value as string}
                    <span class="inv-status {s === 'Paid' ? 'inv-status--paid' : s === 'Overdue' ? 'inv-status--overdue' : 'inv-status--outstanding'}">{s}</span>
                  {:else}
                    {ctx.formatted}
                  {/if}
                {/snippet}
              </DataTable>
            </div>
          {:else}
            <div class="detail-section">
              <div class="detail-section-label af-label">Recent Invoices</div>
              <p class="af-meta detail-empty">No recent invoices.</p>
            </div>
          {/if}

        </div>
      {/if}
    {/snippet}

    {#snippet detailFooter(id)}
      <Button variant="ghost" onclick={() => (selectedId = null)}>Close</Button>
      <Button variant="primary">Edit Customer</Button>
    {/snippet}
  </SplitDetail>

</div>

<!-- ─── New Customer Modal ──────────────────────────────────────────────────── -->
<Modal
  bind:open={showCreate}
  title="New Customer"
  size="lg"
>
  {#snippet children()}
    <div class="modal-form">

      <!-- Identity -->
      <div class="form-section-label af-label">Identity</div>
      <div class="form-row">
        <FormGroup label="Company Name" controlId="nc-company" required error={formError}>
          {#snippet children()}
            <Input id="nc-company" bind:value={form.business_name} placeholder="Acme Instrumentation WLL" />
          {/snippet}
        </FormGroup>
        <FormGroup label="CR Number" controlId="nc-cr">
          {#snippet children()}
            <Input id="nc-cr" bind:value={form.cr_number} placeholder="CR-12345" />
          {/snippet}
        </FormGroup>
      </div>

      <div class="form-row form-row--3">
        <FormGroup label="Type" controlId="nc-type">
          {#snippet children()}
            <Select
              id="nc-type"
              options={typeOptions}
              bind:value={form.customer_type}
              placeholder="Select type"
            />
          {/snippet}
        </FormGroup>
        <FormGroup label="Grade" controlId="nc-grade">
          {#snippet children()}
            <Select
              id="nc-grade"
              options={gradeOptions}
              bind:value={form.payment_grade}
              placeholder="Select grade"
            />
          {/snippet}
        </FormGroup>
        <FormGroup label="Status" controlId="nc-status">
          {#snippet children()}
            <Select
              id="nc-status"
              options={statusOptions}
              bind:value={form.status}
            />
          {/snippet}
        </FormGroup>
      </div>

      <FormGroup label="Industry" controlId="nc-industry">
        {#snippet children()}
          <Input id="nc-industry" bind:value={form.industry} placeholder="Oil & Gas" />
        {/snippet}
      </FormGroup>

      <!-- Contact -->
      <div class="form-section-label af-label">Contact</div>
      <div class="form-row">
        <FormGroup label="Primary Phone" controlId="nc-phone">
          {#snippet children()}
            <Input id="nc-phone" type="tel" bind:value={form.primary_phone} placeholder="+973 1723 0000" />
          {/snippet}
        </FormGroup>
        <FormGroup label="Primary Email" controlId="nc-email">
          {#snippet children()}
            <Input id="nc-email" type="email" bind:value={form.primary_email} placeholder="accounts@company.bh" />
          {/snippet}
        </FormGroup>
      </div>
      <div class="form-row">
        <FormGroup label="City" controlId="nc-city">
          {#snippet children()}
            <Input id="nc-city" bind:value={form.city} placeholder="Manama" />
          {/snippet}
        </FormGroup>
        <FormGroup label="Country" controlId="nc-country">
          {#snippet children()}
            <Input id="nc-country" bind:value={form.country} />
          {/snippet}
        </FormGroup>
      </div>

      <!-- Financial -->
      <div class="form-section-label af-label">Financial</div>
      <div class="form-row">
        <FormGroup label="Payment Terms" controlId="nc-terms">
          {#snippet children()}
            <Select
              id="nc-terms"
              options={termsOptions}
              bind:value={form.payment_terms}
              placeholder="Select terms"
            />
          {/snippet}
        </FormGroup>
        <FormGroup label="Credit Limit (BHD)" controlId="nc-credit">
          {#snippet children()}
            <Input id="nc-credit" type="text" inputmode="decimal" bind:value={form.credit_limit} placeholder="0" />
          {/snippet}
        </FormGroup>
      </div>

    </div>
  {/snippet}

  {#snippet footer()}
    <Button variant="ghost" onclick={() => { showCreate = false; form = emptyForm(); formError = ''; }}>
      Cancel
    </Button>
    <Button variant="primary" onclick={handleCreate}>
      Create Customer
    </Button>
  {/snippet}
</Modal>

<style>
  /* ── Proof page layout ─────────────────────────────────────────────────── */
  .proof-page {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-4);
  }

  /* ── KPI row ─────────────────────────────────────────────────────────────── */
  .kpi-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--af-grid-gap);
  }

  /* ── Row action button ───────────────────────────────────────────────────── */
  .row-act-btn {
    display: inline-flex;
    align-items: center;
    padding: 0 var(--af-space-2);
    height: 28px;
    border: 1px solid var(--af-border-strong);
    border-radius: var(--af-radius-sm);
    background: var(--af-surface);
    color: var(--af-text-secondary);
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-medium);
    cursor: pointer;
    white-space: nowrap;
    transition:
      background var(--af-motion-optimize-duration) var(--af-motion-optimize-ease),
      color var(--af-motion-optimize-duration) var(--af-motion-optimize-ease);
  }

  .row-act-btn:hover {
    background: var(--af-surface-raised);
    color: var(--af-text);
  }

  .row-act-btn:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
  }

  /* ── Drawer detail body ──────────────────────────────────────────────────── */
  .detail-body {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-4);
  }

  .detail-section {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
    padding-block-end: var(--af-space-4);
    border-bottom: 1px solid var(--af-border);
  }

  .detail-section:last-child {
    border-bottom: none;
  }

  .detail-section-label {
    /* .af-label from base: 11px, 600, uppercase, 0.08em — semantic role clear */
    color: var(--af-text-muted);
    margin-block-end: var(--af-space-1);
  }

  .detail-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--af-space-3) var(--af-space-4);
  }

  .detail-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .detail-field--full {
    grid-column: span 2;
  }

  .detail-field-label {
    /* af-label: muted tone for secondary labels inside a drawer */
    color: var(--af-text-muted);
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
  }

  .detail-field-value {
    font-family: var(--af-font-body);
    font-size: var(--af-text-sm);
    color: var(--af-text);
    font-weight: var(--af-weight-medium);
  }

  .val--warn {
    color: var(--af-warning);
  }

  .detail-link {
    color: var(--af-accent);
    text-decoration: none;
  }

  .detail-link:hover {
    text-decoration: underline;
  }

  .detail-link:focus-visible {
    outline: 2px solid var(--af-focus-ring);
    outline-offset: 2px;
    border-radius: 2px;
  }

  /* Classification badges row */
  .detail-badges {
    display: flex;
    align-items: center;
    gap: var(--af-space-2);
    flex-wrap: wrap;
  }

  .detail-type-chip {
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    letter-spacing: var(--af-label-tracking);
    text-transform: uppercase;
    color: var(--af-text-secondary);
    background: var(--af-surface-raised);
    border: 1px solid var(--af-border);
    border-radius: var(--af-radius-pill);
    padding: 2px var(--af-space-2);
  }

  .detail-empty {
    color: var(--af-text-muted);
    font-size: var(--af-text-sm);
  }

  /* Recent invoice status — monochrome weight (§4c) */
  .inv-status {
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    letter-spacing: 0.02em;
    text-transform: uppercase;
  }

  .inv-status--paid        { color: var(--af-success); }
  .inv-status--outstanding { color: var(--af-text-secondary); }
  .inv-status--overdue     { color: var(--af-danger); }

  /* ── Modal form ──────────────────────────────────────────────────────────── */
  .modal-form {
    display: flex;
    flex-direction: column;
    gap: var(--af-space-2);
  }

  .form-section-label {
    margin-block-start: var(--af-space-3);
    margin-block-end: var(--af-space-1);
    padding-block-end: var(--af-space-1);
    border-bottom: 1px solid var(--af-border);
    color: var(--af-text-muted);
    font-family: var(--af-font-body);
    font-size: var(--af-text-xs);
    font-weight: var(--af-weight-semibold);
    text-transform: uppercase;
    letter-spacing: var(--af-label-tracking);
  }

  .form-section-label:first-child {
    margin-block-start: 0;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--af-space-3);
  }

  .form-row--3 {
    grid-template-columns: 1fr 1fr 1fr;
  }

  /* ── Reduced motion ──────────────────────────────────────────────────────── */
  @media (prefers-reduced-motion: reduce) {
    .row-act-btn {
      transition: none;
    }
  }
</style>
