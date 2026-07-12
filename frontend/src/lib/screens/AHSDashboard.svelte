<script lang="ts">
    import { run } from 'svelte/legacy';

    /**
     * AHSDashboard - Financial dashboard for Beacon Controls WLL (sister company)
     * E2: Shows division-filtered financial data for Beacon Controls
     */
    import { onMount } from 'svelte';
    import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
    import { toast } from '$lib/stores/toasts';
    import { GetFinancialDashboardByDivision } from '../../../wailsjs/go/main/App';
import { GetFinancialReportYears } from '../../../wailsjs/go/main/FinanceService';

    interface Props {
        embedded?: boolean;
    }

    let { embedded = false }: Props = $props();
    run(() => {
        embedded;
    });

    let loading = $state(true);
    let data: any = $state(null);
    let availableYears: number[] = $state([]);
    let selectedYear = $state(new Date().getFullYear());

    const fmtCurrency = (val: number) =>
        new Intl.NumberFormat('en-BH', { style: 'currency', currency: 'BHD', minimumFractionDigits: 3, maximumFractionDigits: 3 }).format(val || 0);

    const fmtCompact = (val: number) => {
        if (val === null || val === undefined || val === 0) return 'BHD 0';
        const abs = Math.abs(val);
        const sign = val < 0 ? '-' : '';
        if (abs >= 1000000) return `${sign}BHD ${(abs / 1000000).toFixed(2)}M`;
        if (abs >= 1000) return `${sign}BHD ${(abs / 1000).toFixed(1)}K`;
        return `${sign}BHD ${abs.toFixed(0)}`;
    };

    const fmtSignedCompact = (val: number) => {
        if (val === null || val === undefined) return 'BHD 0';
        if (val === 0) return 'BHD 0';
        return fmtCompact(val);
    };

    async function loadYears() {
        try {
            const years = await GetFinancialReportYears();
            if (years && years.length > 0) {
                availableYears = Array.from(new Set([2024, 2023, ...years])).sort((a, b) => b - a);
                selectedYear = availableYears[0] || new Date().getFullYear();
            } else {
                availableYears = [new Date().getFullYear(), 2025, 2024, 2023];
            }
        } catch (err) {
            availableYears = [new Date().getFullYear(), 2025, 2024, 2023];
        }
    }

    async function loadDashboard() {
        loading = true;
        try {
            data = await GetFinancialDashboardByDivision(selectedYear, 'Beacon Controls');
        } catch (err) {
            console.error('Failed to load AHS dashboard:', err);
            toast.danger(`Failed to load Beacon Controls data for ${selectedYear}`);
            data = { division: 'Beacon Controls', year: selectedYear, has_data: false };
        } finally {
            loading = false;
        }
    }

    run(() => {
        if (selectedYear) {
            loadDashboard();
        }
    });

    onMount(async () => {
        await loadYears();
        await loadDashboard();
    });
</script>

<div class="ahs-dashboard">
    <!-- Year Selector -->
    <div class="controls">
        <div class="year-selector">
            <label for="ahs-year">Financial Year:</label>
            <select id="ahs-year" bind:value={selectedYear} class="year-select">
                {#each availableYears as year}
                    <option value={year}>FY{year}</option>
                {/each}
            </select>
        </div>
        <div class="company-badge">Beacon Controls WLL</div>
    </div>

    {#if data?.source}
        <div class="source-strip">
            <span>{data.source}</span>
            {#if data.is_audited}
                <span>Audited</span>
            {/if}
        </div>
    {/if}

    {#if loading}
        <div class="loading-container">
            <WabiSpinner size="lg" />
        </div>
    {:else if !data || !data.has_data}
        <!-- No data state -->
        <div class="no-data-container">
            <div class="no-data-icon">
                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#d1d1d6" stroke-width="1.5">
                    <path d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
            </div>
            <h3>No Data for Beacon Controls in FY{selectedYear}</h3>
            <p>
                Beacon Controls financial data will appear here once invoices and orders
                are tagged with the "Beacon Controls" division in the costing sheet.
            </p>
            <p class="hint">
                To tag data: Create a costing sheet with Division = "Beacon Controls",
                then convert it to an offer and order. The division will carry through
                to invoices automatically.
            </p>
        </div>
    {:else}
        <!-- KPI Cards -->
        <div class="kpi-grid">
            <div class="kpi-card">
                <div class="kpi-label">Revenue</div>
                <div class="kpi-value">{fmtCompact(data.revenue)}</div>
                <div class="kpi-sub">{data.is_audited ? 'Audited annual revenue' : `${data.invoice_count} invoices`}</div>
            </div>
            <div class="kpi-card">
                <div class="kpi-label">Net Result</div>
                <div class="kpi-value">{fmtSignedCompact(data.net_profit)}</div>
                <div class="kpi-sub">{data.net_profit < 0 ? 'Net loss' : 'Net profit'}</div>
            </div>
            <div class="kpi-card">
                <div class="kpi-label">Cash</div>
                <div class="kpi-value">{fmtCompact(data.cash_equivalents)}</div>
                <div class="kpi-sub">Cash and bank balances</div>
            </div>
            <div class="kpi-card">
                <div class="kpi-label">Total Assets</div>
                <div class="kpi-value">{fmtCompact(data.total_assets)}</div>
                <div class="kpi-sub">Equity: {fmtCompact(data.total_equity)}</div>
            </div>
        </div>

        <!-- Summary Table -->
        <div class="summary-section">
            <h4>Financial Summary - Beacon Controls WLL</h4>
            <table class="summary-table">
                <thead>
                    <tr>
                        <th>Metric</th>
                        <th class="number-col">Value</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Total Revenue</td>
                        <td class="number-col">{fmtCurrency(data.revenue)}</td>
                    </tr>
                    <tr>
                        <td>Cost of Sales</td>
                        <td class="number-col">{fmtCurrency(data.cost_of_sales)}</td>
                    </tr>
                    <tr>
                        <td>Gross Profit</td>
                        <td class="number-col">{fmtCurrency(data.gross_profit)}</td>
                    </tr>
                    <tr>
                        <td>Staff Costs</td>
                        <td class="number-col">{fmtCurrency(data.staff_costs)}</td>
                    </tr>
                    <tr>
                        <td>Administrative Expenses</td>
                        <td class="number-col">{fmtCurrency(data.admin_expenses)}</td>
                    </tr>
                    <tr class:overdue={data.net_profit < 0}>
                        <td>Net Result</td>
                        <td class="number-col">{fmtCurrency(data.net_profit)}</td>
                    </tr>
                    <tr>
                        <td>Trade Receivables</td>
                        <td class="number-col">{fmtCurrency(data.trade_receivables)}</td>
                    </tr>
                    <tr>
                        <td>Cash & Bank</td>
                        <td class="number-col">{fmtCurrency(data.cash_equivalents)}</td>
                    </tr>
                    <tr>
                        <td>Total Assets</td>
                        <td class="number-col">{fmtCurrency(data.total_assets)}</td>
                    </tr>
                    <tr>
                        <td>Total Liabilities</td>
                        <td class="number-col">{fmtCurrency(data.total_liabilities)}</td>
                    </tr>
                    <tr>
                        <td>Total Equity</td>
                        <td class="number-col">{fmtCurrency(data.total_equity)}</td>
                    </tr>
                </tbody>
            </table>
        </div>
    {/if}
</div>

<style>
    .ahs-dashboard {
        max-width: 1200px;
    }

    .controls {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 24px;
    }

    .year-selector {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .year-selector label {
        font-size: 13px;
        font-weight: 500;
        color: #6e6e73;
    }

    .year-select {
        padding: 6px 12px;
        border: 1px solid #e5e5e5;
        border-radius: 6px;
        font-size: 13px;
        font-family: "DM Sans", sans-serif;
        outline: none;
        cursor: pointer;
    }

    .year-select:focus {
        border-color: #1d1d1f;
    }

    .company-badge {
        padding: 6px 12px;
        background: #f0f0f5;
        border-radius: 6px;
        font-size: 12px;
        font-weight: 600;
        color: #6e6e73;
        letter-spacing: 0.02em;
    }

    .source-strip {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 18px;
        padding: 10px 14px;
        border-radius: 10px;
        background: linear-gradient(135deg, rgba(232, 245, 233, 0.9), rgba(244, 247, 240, 0.9));
        color: #3b4a3f;
        font-size: 12px;
        font-weight: 600;
    }

    .loading-container {
        display: flex;
        justify-content: center;
        padding: 80px 0;
    }

    /* No Data State */
    .no-data-container {
        text-align: center;
        padding: 60px 40px;
        background: #ffffff;
        border: 1px solid #e5e5e5;
        border-radius: 12px;
    }

    .no-data-icon {
        margin-bottom: 16px;
    }

    .no-data-container h3 {
        font-family: "Arvo", "Georgia", serif;
        font-size: 18px;
        font-weight: 400;
        color: #1d1d1f;
        margin: 0 0 12px 0;
    }

    .no-data-container p {
        font-size: 14px;
        color: #6e6e73;
        margin: 0 0 8px 0;
        max-width: 500px;
        margin-left: auto;
        margin-right: auto;
        line-height: 1.5;
    }

    .hint {
        font-size: 12px !important;
        color: #8e8e93 !important;
        padding-top: 8px;
        border-top: 1px solid #f0f0f0;
        margin-top: 16px !important;
    }

    /* KPI Cards */
    .kpi-grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 16px;
        margin-bottom: 24px;
    }

    .kpi-card {
        background: #ffffff;
        border: 1px solid #e5e5e5;
        border-radius: 12px;
        padding: 20px;
    }

    .kpi-label {
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: #8e8e93;
        margin-bottom: 8px;
    }

    .kpi-value {
        font-family: "Arvo", "Georgia", serif;
        font-size: 24px;
        font-weight: 400;
        color: #1d1d1f;
        margin-bottom: 4px;
    }

    .kpi-sub {
        font-size: 12px;
        color: #8e8e93;
    }

    /* Summary Table */
    .summary-section {
        background: #ffffff;
        border: 1px solid #e5e5e5;
        border-radius: 12px;
        padding: 20px;
    }

    .summary-section h4 {
        font-family: "Arvo", "Georgia", serif;
        font-size: 15px;
        font-weight: 400;
        color: #1d1d1f;
        margin: 0 0 16px 0;
    }

    .summary-table {
        width: 100%;
        border-collapse: collapse;
    }

    .summary-table th {
        padding: 8px 12px;
        text-align: left;
        font-size: 10px;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: #8e8e93;
        border-bottom: 1px solid #e5e5e5;
    }

    .summary-table td {
        padding: 10px 12px;
        font-size: 13px;
        color: #1d1d1f;
        border-bottom: 1px solid #f0f0f0;
    }

    .summary-table .number-col {
        text-align: right;
        font-family: "SF Mono", "Menlo", monospace;
        font-size: 13px;
    }

    .summary-table tr.overdue td {
        color: #ff3b30;
        font-weight: 500;
    }

    @media (max-width: 768px) {
        .kpi-grid {
            grid-template-columns: repeat(2, 1fr);
        }

        .controls {
            flex-direction: column;
            gap: 12px;
            align-items: flex-start;
        }
    }
</style>
