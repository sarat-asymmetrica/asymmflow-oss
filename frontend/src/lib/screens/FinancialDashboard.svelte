<script lang="ts">
    /**
     * FinancialDashboard - McKinsey Standard Executive View
     * Dynamically loads data from Tally imports or falls back to FY2024 baseline
     */
    import { onMount } from 'svelte';
    import WabiSpinner from '$lib/components/ui/WabiSpinner.svelte';
    import { toast } from '$lib/stores/toasts';
    import { permissions } from '$lib/stores/authContext';
    import { GetFinancialDashboardForYear } from '../../../wailsjs/go/main/App';
import { GetFinancialReportYears, GetCashPosition } from '../../../wailsjs/go/main/FinanceService';


    interface Props {
        embedded?: boolean;
    }

    let { embedded = false }: Props = $props();

    let loading = $state(true);
    let data: any = $state(null);
    let availableYears: number[] = $state([]);
    let selectedYear = $state(new Date().getFullYear()); // Default to current year
    let noDataForYear = $state(false);
    let loadError = $state(false); // B9b: true when the backend load failed - never paper over with fabricated figures
    let liveCashTotal = $state(0); // Sum of closing balances from all bank accounts
    let cashNoticeItems: string[] = $state([]);

    const fmtCurrency = (val: number) =>
        new Intl.NumberFormat('en-BH', { style: 'currency', currency: 'BHD', minimumFractionDigits: 3, maximumFractionDigits: 3 }).format(val);

    // Compact currency for KPI cards - uses K/M for large numbers
    const fmtCompact = (val: number) => {
        if (val === null || val === undefined) return 'BHD 0';
        const abs = Math.abs(val);
        const sign = val < 0 ? '-' : '';
        if (abs >= 1000000) return `${sign}BHD ${(abs / 1000000).toFixed(2)}M`;
        if (abs >= 1000) return `${sign}BHD ${(abs / 1000).toFixed(1)}K`;
        return `${sign}BHD ${abs.toFixed(0)}`;
    };

    const fmtPct = (val: number) => `${val.toFixed(1)}%`;
    const fmtRatio = (val: number) => `${val.toFixed(2)}x`;

    // Years with demo financial data
    const AUDITED_YEARS = [2023, 2024];

    // B9a: ratio health thresholds computed from real values (display logic only,
    // touches no posting/rounding/tax math). "Accurate red beats false green" -
    // every ratio color must tell the truth, not be a hardcoded placeholder.
    type RatioHealth = 'good' | 'caution' | 'danger' | '';
    function ratioHealth(kind: string, val: number | null | undefined): RatioHealth {
        if (val === null || val === undefined || Number.isNaN(val)) return '';
        switch (kind) {
            case 'current_ratio':
                return val >= 1.5 ? 'good' : val >= 1.0 ? 'caution' : 'danger';
            case 'quick_ratio':
                return val >= 1.0 ? 'good' : val >= 0.7 ? 'caution' : 'danger';
            case 'cash_ratio':
                return val >= 0.5 ? 'good' : val >= 0.2 ? 'caution' : 'danger';
            case 'debt_to_equity':
                return val <= 1.0 ? 'good' : val <= 2.0 ? 'caution' : 'danger';
            case 'equity_ratio':
                // equity_ratio is rendered via fmtPct (e.g. 71.3 => "71.3%"), so
                // thresholds are expressed in percentage points, not 0-1 ratio units.
                return val >= 50 ? 'good' : val >= 30 ? 'caution' : 'danger';
            case 'dso':
                return val <= 45 ? 'good' : val <= 75 ? 'caution' : 'danger';
            case 'gross_margin':
                return val >= 25 ? 'good' : val >= 15 ? 'caution' : 'danger';
            case 'roe':
                return val >= 15 ? 'good' : val >= 5 ? 'caution' : 'danger';
            case 'asset_turnover':
                // Informational only - no danger state, just a positive signal above 1x.
                return val > 1 ? 'good' : '';
            default:
                return '';
        }
    }

    // B9c: KPI cards drill into the app's navigate mechanism (same convention as
    // DashboardScreen/OrdersScreen/InvoicesScreen etc.) - keyboard-accessible too.
    function navigateTo(screen: string, tab?: string) {
        window.dispatchEvent(new CustomEvent('navigateToScreen', { detail: tab ? { screen, tab } : { screen } }));
    }
    function handleKpiKeydown(event: KeyboardEvent, screen: string, tab?: string) {
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            navigateTo(screen, tab);
        }
    }


    function compactAccountRef(value: string) {
        return value
            .replace(/\b\d{8,}\b/g, (match) => `...${match.slice(-4)}`)
            .replace(/\s+/g, ' ')
            .trim();
    }

    function formatCashNotice(notice: string) {
        const clean = notice.replace(/\.$/, '').trim();
        const missingMatch = clean.match(/^No statement imported for (.+)$/i);
        if (missingMatch) {
            return `Missing statement: ${compactAccountRef(missingMatch[1])}`;
        }

        const staleMatch = clean.match(/^(.+) latest statement is ([^;]+); latest imported month for (.+) is (.+)$/i);
        if (staleMatch) {
            return `${compactAccountRef(staleMatch[1])} is behind (${staleMatch[2]} vs ${staleMatch[3]} ${staleMatch[4]})`;
        }

        return compactAccountRef(clean);
    }

    async function loadAvailableYears() {
        try {
            const years = await GetFinancialReportYears();
            if (years && years.length > 0) {
                availableYears = years;
                // Default to latest available year
                selectedYear = years[0] || new Date().getFullYear();
            } else {
                // Fallback: current year + audited years
                const curYear = new Date().getFullYear();
                availableYears = [curYear, 2025, 2024, 2023];
                selectedYear = curYear;
            }
        } catch (err) {
            console.error('Failed to load available years:', err);
            // Fallback: current year + audited years
            availableYears = [new Date().getFullYear(), 2025, 2024, 2023];
            selectedYear = 2024;
        }
    }

    async function loadDashboard() {
        loading = true;
        noDataForYear = false;
        loadError = false;
        try {
            const result = await GetFinancialDashboardForYear(selectedYear);
            data = result;

            // Check if we're using Tally data or fallback
            if (!availableYears.includes(selectedYear) && selectedYear !== 2024) {
                noDataForYear = true;
            }
        } catch (err) {
            console.error('Failed to load dashboard:', err);
            toast.danger(`Failed to load financial data for ${selectedYear}`);
            // B9b: never paper over a failed load with fabricated numbers - a fake
            // FY2024 dataset shown as "live" under a user-selected year is a lie
            // about money. Surface an explicit, visible error state instead.
            data = null;
            loadError = true;
        } finally {
            loading = false;
        }
    }


    async function handleYearChange() {
        await loadDashboard();
    }

    async function loadCashPosition() {
        try {
            const pos = await GetCashPosition();
            if (pos) {
                liveCashTotal = Number(pos.cash_balance_bhd ?? pos.total_bhd ?? 0);
                cashNoticeItems = Array.isArray(pos.notices) ? pos.notices.filter(Boolean) : [];
            }
        } catch (err) {
            console.warn('GetCashPosition not available, using fallback:', err);
            cashNoticeItems = [];
        }
    }

    onMount(async () => {
        await loadAvailableYears();
        await Promise.all([loadDashboard(), loadCashPosition()]);
    });
    let permissionList = $derived(Array.isArray($permissions) ? $permissions : []);
    let canView = $derived(permissionList.includes('*') || permissionList.includes('finance:view') || permissionList.includes('finance:*'));
    let isAuditedYear = $derived(AUDITED_YEARS.includes(selectedYear));
    let liveDataBadge = $derived(data?.source?.includes('Fresh Start') ? 'Live ERP Data' : 'Unaudited Live Data');
    let cashNoticeSummary = $derived(cashNoticeItems.length
        ? `${cashNoticeItems.length} bank ${cashNoticeItems.length === 1 ? 'account needs' : 'accounts need'} statement attention`
        : '');
    let cashNoticePreview = $derived(cashNoticeItems.slice(0, 3).map(formatCashNotice));
    let hiddenCashNoticeCount = $derived(Math.max(cashNoticeItems.length - cashNoticePreview.length, 0));
    let totalAR = $derived(data ? data.ar_current + data.ar_30_60 + data.ar_60_90 + data.ar_over_90 : 0);
    let arBarDenominator = $derived(Math.max(totalAR, 1));
    let assetBarDenominator = $derived(Math.max(Number(data?.total_assets || 0), 1));
</script>

<div class="dashboard" class:embedded>
    {#if !canView}
        <div class="loading-state">
            <p style="color: var(--text-secondary); font-size: 14px;">You don't have permission to view financial data.</p>
        </div>
    {:else if loading}
        <div class="loading-state">
            <WabiSpinner size="lg" tempo="calm" />
            <p>Loading financial data...</p>
        </div>
    {:else if loadError}
        <div class="loading-state">
            <p class="load-error-message">Financial data unavailable for FY{selectedYear} &mdash; could not load.</p>
            <button type="button" class="retry-btn" onclick={loadDashboard}>Retry</button>
        </div>
    {:else if data}
        <!-- Header -->
        <div class="dashboard-header">
            <div class="header-left">
                <h2>Financial Performance</h2>
                <span class="period-badge">{data.period}</span>
                {#if noDataForYear}
                    <span class="no-data-badge">Using FY2024 baseline</span>
                {/if}
            </div>
            <div class="header-right">
                <div class="year-selector">
                    <label for="year-select">Fiscal Year:</label>
                    <select id="year-select" bind:value={selectedYear} onchange={handleYearChange}>
                        {#each availableYears as year}
                            <option value={year}>FY{year}</option>
                        {/each}
                    </select>
                </div>
                {#if isAuditedYear}
                    <span class="audit-badge audited">Demo financial data</span>
                {:else}
                    <span class="audit-badge unaudited">{liveDataBadge}</span>
                {/if}
            </div>
        </div>

        <!-- Wave 9.3 B4: name the source so the two P&Ls don't confuse. -->
        <div class="source-authority-note" role="note">
            {#if isAuditedYear}
                <strong>Audited/filed figures.</strong> Audited demo data for this year.
            {:else}
                <strong>Live ERP figures.</strong> Real-time, unaudited — computed from current ERP records.
            {/if}
            The <em>Accounting → Reports</em> statements are built from imported Tally (filed) data and can
            differ; use those for what was filed, this for where the business stands today.
        </div>

        <!-- Primary KPIs Row -->
        <div class="kpi-row">
            <div
                class="kpi-card"
                role="button"
                tabindex="0"
                onclick={() => navigateTo('accounting')}
                onkeydown={(e) => handleKpiKeydown(e, 'accounting')}
            >
                <div class="kpi-label">Revenue</div>
                <div class="kpi-value">{fmtCompact(data.revenue)}</div>
                <div class="kpi-change" class:negative={data.revenue_yoy < 0} class:positive={data.revenue_yoy >= 0}>
                    {data.revenue_yoy >= 0 ? '+' : '-'} {fmtPct(Math.abs(data.revenue_yoy))} YoY
                </div>
            </div>
            <div
                class="kpi-card"
                role="button"
                tabindex="0"
                onclick={() => navigateTo('finance', 'invoices')}
                onkeydown={(e) => handleKpiKeydown(e, 'finance', 'invoices')}
            >
                <div class="kpi-label">Cash Balance</div>
                <div class="kpi-value">{fmtCompact(liveCashTotal > 0 ? liveCashTotal : data.cash_and_equiv)}</div>
                <div class="kpi-sub">{liveCashTotal > 0 ? 'Latest bank statements' : 'Awaiting bank statements'}</div>
            </div>
            <div
                class="kpi-card"
                role="button"
                tabindex="0"
                onclick={() => navigateTo('finance', 'invoices')}
                onkeydown={(e) => handleKpiKeydown(e, 'finance', 'invoices')}
            >
                <div class="kpi-label">Accounts Receivable</div>
                <div class="kpi-value">{fmtCompact(data.trade_receivables)}</div>
                <div class="kpi-sub">Open AR and uninvoiced orders</div>
            </div>
            <div
                class="kpi-card highlight"
                role="button"
                tabindex="0"
                onclick={() => navigateTo('accounting')}
                onkeydown={(e) => handleKpiKeydown(e, 'accounting')}
            >
                <div class="kpi-label">Net Profit</div>
                <div class="kpi-value">{fmtCompact(data.net_profit)}</div>
                <div class="kpi-sub">Margin: {fmtPct(data.net_margin)}</div>
            </div>
        </div>

        {#if cashNoticeItems.length}
            <div class="statement-note">
                <div class="statement-note-header">
                    <strong>Statement check:</strong>
                    <span>{cashNoticeSummary}</span>
                </div>
                <ul>
                    {#each cashNoticePreview as notice}
                        <li>{notice}</li>
                    {/each}
                    {#if hiddenCashNoticeCount > 0}
                        <li>{hiddenCashNoticeCount} more {hiddenCashNoticeCount === 1 ? 'account' : 'accounts'} need review.</li>
                    {/if}
                </ul>
            </div>
        {/if}

        <!-- Main Grid -->
        <div class="main-grid">
            <!-- Balance Sheet Card -->
            <div class="card">
                <div class="card-header">
                    <h3>Balance Sheet</h3>
                    <span class="card-date">As of {data.as_of_date}</span>
                </div>
                <div class="card-body">
                    <div class="bs-section">
                        <div class="bs-title">Assets</div>
                        <div class="bs-row">
                            <span>Current Assets</span>
                            <span class="bs-val">{fmtCurrency(data.current_assets)}</span>
                        </div>
                        <div class="bs-bar">
                            <div class="bs-fill blue" style="width: {data.current_assets / assetBarDenominator * 100}%"></div>
                        </div>
                        <div class="bs-row">
                            <span>Non-Current Assets</span>
                            <span class="bs-val">{fmtCurrency(data.non_current_assets)}</span>
                        </div>
                        <div class="bs-bar">
                            <div class="bs-fill purple" style="width: {data.non_current_assets / assetBarDenominator * 100}%"></div>
                        </div>
                        <div class="bs-row total">
                            <span>Total Assets</span>
                            <span class="bs-val">{fmtCurrency(data.total_assets)}</span>
                        </div>
                    </div>
                    <div class="bs-divider"></div>
                    <div class="bs-section">
                        <div class="bs-title">Liabilities & Equity</div>
                        <div class="bs-row">
                            <span>Total Liabilities</span>
                            <span class="bs-val">{fmtCurrency(data.total_liabilities)}</span>
                        </div>
                        <div class="bs-bar">
                            <div class="bs-fill red" style="width: {data.total_liabilities / assetBarDenominator * 100}%"></div>
                        </div>
                        <div class="bs-row">
                            <span>Total Equity</span>
                            <span class="bs-val">{fmtCurrency(data.total_equity)}</span>
                        </div>
                        <div class="bs-bar">
                            <div class="bs-fill green" style="width: {data.total_equity / assetBarDenominator * 100}%"></div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Key Ratios Card -->
            <div class="card">
                <div class="card-header">
                    <h3>Key Financial Ratios</h3>
                </div>
                <div class="card-body">
                    <div class="ratios-grid">
                        <div class="ratio-section">
                            <div class="ratio-title">Liquidity</div>
                            <div class="ratio-row">
                                <span>Current Ratio</span>
                                <span class="ratio-val {ratioHealth('current_ratio', data.current_ratio)}">{fmtRatio(data.current_ratio)}</span>
                            </div>
                            <div class="ratio-row">
                                <span>Quick Ratio</span>
                                <span class="ratio-val {ratioHealth('quick_ratio', data.quick_ratio)}">{fmtRatio(data.quick_ratio)}</span>
                            </div>
                            <div class="ratio-row">
                                <span>Cash Ratio</span>
                                <span class="ratio-val {ratioHealth('cash_ratio', data.cash_ratio)}">{fmtRatio(data.cash_ratio)}</span>
                            </div>
                        </div>
                        <div class="ratio-section">
                            <div class="ratio-title">Solvency</div>
                            <div class="ratio-row">
                                <span>Debt/Equity</span>
                                <span class="ratio-val {ratioHealth('debt_to_equity', data.debt_to_equity)}">{fmtRatio(data.debt_to_equity)}</span>
                            </div>
                            <div class="ratio-row">
                                <span>Equity Ratio</span>
                                <span class="ratio-val {ratioHealth('equity_ratio', data.equity_ratio)}">{fmtPct(data.equity_ratio)}</span>
                            </div>
                        </div>
                        <div class="ratio-section">
                            <div class="ratio-title">Efficiency</div>
                            <div class="ratio-row">
                                <span>DSO</span>
                                <span class="ratio-val {ratioHealth('dso', data.dso)}">{typeof data.dso === 'number' ? data.dso.toFixed(1) : data.dso} days</span>
                            </div>
                            <div class="ratio-row">
                                <span>Asset Turnover</span>
                                <span class="ratio-val {ratioHealth('asset_turnover', data.asset_turnover)}">{fmtRatio(data.asset_turnover)}</span>
                            </div>
                        </div>
                        <div class="ratio-section">
                            <div class="ratio-title">Profitability</div>
                            <div class="ratio-row">
                                <span>Gross Margin</span>
                                <span class="ratio-val {ratioHealth('gross_margin', data.gross_margin)}">{fmtPct(data.gross_margin)}</span>
                            </div>
                            <div class="ratio-row">
                                <span>ROE</span>
                                <span class="ratio-val {ratioHealth('roe', data.roe)}">{fmtPct(data.roe)}</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- AR Aging Card -->
            <div class="card">
                <div class="card-header">
                    <h3>Receivables Aging</h3>
                    <span class="card-badge warning">{fmtPct(data.ar_overdue_pct)} overdue</span>
                </div>
                <div class="card-body">
                    <div class="ar-total">
                        <span>Total AR</span>
                        <span class="ar-total-val">{fmtCurrency(totalAR)}</span>
                    </div>
                    <div class="ar-buckets">
                        <div class="ar-bucket">
                            <div class="ar-bar-wrap">
                                <div class="ar-bar green" style="height: {data.ar_current / arBarDenominator * 100}%"></div>
                            </div>
                            <div class="ar-label">0-30d</div>
                            <div class="ar-val">{fmtCurrency(data.ar_current)}</div>
                        </div>
                        <div class="ar-bucket">
                            <div class="ar-bar-wrap">
                                <div class="ar-bar lime" style="height: {data.ar_30_60 / arBarDenominator * 100}%"></div>
                            </div>
                            <div class="ar-label">31-60d</div>
                            <div class="ar-val">{fmtCurrency(data.ar_30_60)}</div>
                        </div>
                        <div class="ar-bucket">
                            <div class="ar-bar-wrap">
                                <div class="ar-bar orange" style="height: {data.ar_60_90 / arBarDenominator * 100}%"></div>
                            </div>
                            <div class="ar-label">61-90d</div>
                            <div class="ar-val">{fmtCurrency(data.ar_60_90)}</div>
                        </div>
                        <div class="ar-bucket">
                            <div class="ar-bar-wrap">
                                <div class="ar-bar red" style="height: {data.ar_over_90 / arBarDenominator * 100}%"></div>
                            </div>
                            <div class="ar-label">>90d</div>
                            <div class="ar-val">{fmtCurrency(data.ar_over_90)}</div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Working Capital Card -->
            <div class="card">
                <div class="card-header">
                    <h3>Working Capital</h3>
                </div>
                <div class="card-body">
                    <div class="wc-rows">
                        <div class="wc-row">
                            <span>Trade Receivables</span>
                            <span class="wc-val">{fmtCurrency(data.trade_receivables)}</span>
                        </div>
                        <div class="wc-row">
                            <span>Inventory</span>
                            <span class="wc-val">{fmtCurrency(data.inventory)}</span>
                        </div>
                        <div class="wc-row">
                            <span>Trade Payables</span>
                            <span class="wc-val">({fmtCurrency(data.trade_payables)})</span>
                        </div>
                        <div class="wc-row total">
                            <span>Net Working Capital</span>
                            <span class="wc-val">{fmtCurrency(data.working_capital)}</span>
                        </div>
                    </div>
                    <div class="ccc-box">
                        <div class="ccc-title">Cash Conversion Cycle</div>
                        <div class="ccc-formula">
                            <span class="ccc-item">{typeof data.dso === 'number' ? data.dso.toFixed(1) : data.dso}d DSO</span>
                            <span class="ccc-op">+</span>
                            <span class="ccc-item">{typeof data.dio === 'number' ? data.dio.toFixed(1) : data.dio}d DIO</span>
                            <span class="ccc-op">-</span>
                            <span class="ccc-item">{typeof data.dpo === 'number' ? data.dpo.toFixed(1) : data.dpo}d DPO</span>
                            <span class="ccc-op">=</span>
                            <span class="ccc-result">{typeof data.cash_conv_cycle === 'number' ? data.cash_conv_cycle.toFixed(1) : data.cash_conv_cycle}d</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- YoY Comparison -->
        <div class="card wide">
            <div class="card-header">
                <h3>Year-over-Year Comparison</h3>
                <span class="card-date">{data.period} vs {data.prior_year}</span>
            </div>
            <div class="card-body">
                <div class="yoy-grid">
                    <div class="yoy-item">
                        <div class="yoy-label">Revenue</div>
                        <div class="yoy-compare">
                            <div class="yoy-row">
                                <span class="yoy-year">{data.prior_year}</span>
                                <div class="yoy-bar-bg">
                                    <div class="yoy-bar gray" style="width: 100%"></div>
                                </div>
                                <span class="yoy-val">{fmtCurrency(data.py_revenue)}</span>
                            </div>
                            <div class="yoy-row">
                                <span class="yoy-year">{data.period}</span>
                                <div class="yoy-bar-bg">
                                    <div class="yoy-bar blue" style="width: {data.py_revenue > 0 ? Math.min(data.revenue / data.py_revenue * 100, 100) : 0}%"></div>
                                </div>
                                <span class="yoy-val">{fmtCurrency(data.revenue)}</span>
                            </div>
                        </div>
                        <div class="yoy-change" class:negative={data.revenue_yoy < 0} class:positive={data.revenue_yoy >= 0}>
                            {data.revenue_yoy >= 0 ? '+' : '-'} {fmtPct(Math.abs(data.revenue_yoy))}
                        </div>
                    </div>
                    <div class="yoy-item">
                        <div class="yoy-label">Gross Profit</div>
                        <div class="yoy-compare">
                            <div class="yoy-row">
                                <span class="yoy-year">{data.prior_year}</span>
                                <div class="yoy-bar-bg">
                                    <div class="yoy-bar gray" style="width: 100%"></div>
                                </div>
                                <span class="yoy-val">{fmtCurrency(data.py_gross_profit)}</span>
                            </div>
                            <div class="yoy-row">
                                <span class="yoy-year">{data.period}</span>
                                <div class="yoy-bar-bg">
                                    <div class="yoy-bar blue" style="width: {data.py_gross_profit > 0 ? Math.min(data.gross_profit / data.py_gross_profit * 100, 100) : 0}%"></div>
                                </div>
                                <span class="yoy-val">{fmtCurrency(data.gross_profit)}</span>
                            </div>
                        </div>
                        <div class="yoy-change" class:negative={data.gross_profit < data.py_gross_profit} class:positive={data.gross_profit >= data.py_gross_profit}>
                            {data.py_gross_profit > 0 ? `${data.gross_profit >= data.py_gross_profit ? '+' : '-'} ${fmtPct(Math.abs((data.gross_profit - data.py_gross_profit) / data.py_gross_profit * 100))}` : '—'}
                        </div>
                    </div>
                    <div class="yoy-item">
                        <div class="yoy-label">Net Profit</div>
                        <div class="yoy-compare">
                            <div class="yoy-row">
                                <span class="yoy-year">{data.prior_year}</span>
                                <div class="yoy-bar-bg">
                                    <div class="yoy-bar gray" style="width: 100%"></div>
                                </div>
                                <span class="yoy-val">{fmtCurrency(data.py_net_profit)}</span>
                            </div>
                            <div class="yoy-row">
                                <span class="yoy-year">{data.period}</span>
                                <div class="yoy-bar-bg">
                                    <div class="yoy-bar blue" style="width: {data.py_net_profit > 0 ? Math.min(data.net_profit / data.py_net_profit * 100, 100) : 0}%"></div>
                                </div>
                                <span class="yoy-val">{fmtCurrency(data.net_profit)}</span>
                            </div>
                        </div>
                        <div class="yoy-change" class:negative={data.net_profit < data.py_net_profit} class:positive={data.net_profit >= data.py_net_profit}>
                            {data.py_net_profit > 0 ? `${data.net_profit >= data.py_net_profit ? '+' : '-'} ${fmtPct(Math.abs((data.net_profit - data.py_net_profit) / data.py_net_profit * 100))}` : '—'}
                        </div>
                    </div>
                </div>
                <div class="yoy-note">
                    <!-- B9d: prose computed from live `data` instead of hardcoded per-year branches. -->
                    <strong>Note:</strong> Comparing {data.period} to {data.prior_year}. Gross margin {fmtPct(data.gross_margin)}. All figures in BHD.
                </div>
            </div>
        </div>

        <!-- Footer -->
        <div class="dashboard-footer">
            <!-- B9d: footer source is data-driven (data.period) instead of a hardcoded "FY2024" literal. -->
            <span>Source: {data?.source || `Demo financial data ${data?.period || `FY${selectedYear}`}`}</span>
            {#if isAuditedYear}
                <span>Demo financial data</span>
            {:else}
                <span>Note: {data?.period || `FY${selectedYear}`} demo data</span>
            {/if}
        </div>
    {/if}
</div>

<style>
    .dashboard {
        display: flex;
        flex-direction: column;
        gap: 20px;
        font-family: var(--font-body);
    }
    .source-authority-note {
        padding: var(--space-3, 12px) var(--space-4, 16px);
        background: var(--surface-sunken, rgba(0,0,0,0.03));
        border: 1px solid var(--border-subtle, rgba(0,0,0,0.08));
        border-left: 3px solid var(--accent, #35a66f);
        border-radius: var(--radius-md, 8px);
        color: var(--ink-muted, #5a5a5a);
        font-size: var(--text-sm, 0.85rem);
        line-height: 1.5;
    }
    .source-authority-note strong { color: var(--ink, #1a1a1a); }

    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        min-height: 400px;
        gap: 16px;
        color: var(--text-secondary);
    }

    /* B9b: visible degraded state when the load fails - no fabricated numbers. */
    .load-error-message {
        color: var(--text-danger);
        font-weight: 500;
        font-size: 14px;
        text-align: center;
        max-width: 420px;
    }

    .retry-btn {
        padding: 8px 20px;
        background: var(--carbon, #000);
        color: var(--canvas, #FFF);
        border: none;
        border-radius: 6px;
        font-family: var(--font-body);
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: background 120ms ease;
    }

    .retry-btn:hover {
        background: var(--onyx, #1D1D1F);
    }

    /* Header */
    .dashboard-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 16px;
        flex-wrap: wrap;
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

    .header-left {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
        min-width: 0;
    }

    .header-left h2 {
        margin: 0;
        font-family: var(--font-display);
        font-size: 20px;
        font-weight: 500;
        letter-spacing: 0;
        color: var(--onyx, #1D1D1F);
    }

    .header-right {
        display: flex;
        align-items: center;
        gap: 16px;
        flex-wrap: wrap;
    }

    .period-badge {
        background: var(--carbon, #000);
        color: var(--canvas, #FFF);
        padding: 4px 12px;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 600;
    }

    .no-data-badge {
        background: var(--steel, #86868B);
        color: var(--canvas, #FFF);
        padding: 4px 10px;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 500;
    }

    .year-selector {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .year-selector label {
        font-size: 12px;
        font-weight: 500;
        color: var(--steel, #86868B);
    }

    .year-selector select {
        padding: 6px 12px;
        background: var(--canvas, #FFF);
        border: 1px solid var(--border, #E5E5E5);
        border-radius: 4px;
        font-family: var(--font-body);
        font-size: 13px;
        font-weight: 500;
        color: var(--onyx, #1D1D1F);
        cursor: pointer;
        transition: all 120ms ease;
    }

    .year-selector select:hover {
        border-color: var(--onyx, #1D1D1F);
    }

    .year-selector select:focus {
        outline: none;
        border-color: var(--carbon, #000);
        box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.1);
    }

    .audit-badge {
        font-size: 11px;
        padding: 4px 10px;
        border-radius: 4px;
        font-weight: 500;
    }

    .audit-badge.audited {
        background: rgba(21, 128, 61, 0.12);
        color: #15803d;
    }

    .audit-badge.unaudited {
        background: rgba(217, 119, 6, 0.12);
        color: #d97706;
    }

    /* KPI Row */
    .kpi-row {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 16px;
    }

    .kpi-card {
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 20px;
        display: flex;
        flex-direction: column;
        gap: 10px;
        min-width: 0; /* Allow shrinking in grid */
        overflow: hidden; /* Prevent overflow */
        cursor: pointer; /* B9c: KPI cards are drillable */
        transition: box-shadow 120ms ease, border-color 120ms ease;
    }

    .kpi-card:hover {
        border-color: var(--brand-indigo, #1D1D1F);
        box-shadow: var(--shadow-lift, 0 4px 24px rgba(0, 0, 0, 0.04));
    }

    .kpi-card:focus-visible {
        outline: 2px solid var(--brand-indigo, #1D1D1F);
        outline-offset: 2px;
    }

    .kpi-card.highlight {
        background: var(--brand-indigo-tint, rgba(99, 102, 241, 0.08));
        border-color: var(--brand-indigo);
    }

    .statement-note {
        margin: 12px 0 16px;
        padding: 10px 14px;
        border: 1px solid rgba(217, 119, 6, 0.18);
        border-radius: 8px;
        background: rgba(255, 248, 232, 0.84);
        color: #7a4d0b;
        font-size: 12px;
        line-height: 1.45;
    }

    .statement-note-header {
        display: flex;
        gap: 8px;
        align-items: baseline;
        flex-wrap: wrap;
    }

    .statement-note ul {
        margin: 8px 0 0 18px;
        padding: 0;
    }

    .statement-note li + li {
        margin-top: 3px;
    }

    .kpi-label {
        font-family: var(--font-body);
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0;
        color: var(--text-muted);
        font-weight: 500;
        line-height: 1.4;
        overflow-wrap: anywhere;
    }

    .kpi-value {
        font-size: clamp(18px, 2.5vw, 28px); /* Responsive font size */
        font-weight: 500;
        font-family: var(--font-display);
        color: var(--text-primary);
        word-break: normal;
        overflow-wrap: anywhere;
        line-height: 1.18;
        letter-spacing: 0;
        max-width: 100%;
    }

    .kpi-sub {
        font-size: 13px;
        color: var(--text-secondary);
        line-height: 1.5;
        overflow-wrap: anywhere;
    }

    .kpi-change {
        font-size: 13px;
        font-weight: 500;
        line-height: 1.5;
        overflow-wrap: anywhere;
    }

    .kpi-change.positive { color: #15803d; }
    .kpi-change.negative { color: #ef4444; }

    /* Main Grid */
    .main-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 16px;
    }

    /* Cards */
    .card {
        background: var(--surface);
        border: 1px solid var(--border);
        border-radius: 12px;
        overflow: hidden;
        min-width: 0;
    }

    .card.wide {
        grid-column: span 2;
    }

    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 16px;
        padding: 16px 20px;
        background: var(--bg-base);
        border-bottom: 1px solid var(--border);
    }

    .card-header h3 {
        margin: 0;
        font-family: var(--font-display);
        font-size: 14px;
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0;
        line-height: 1.35;
        overflow-wrap: anywhere;
    }

    .card-date {
        font-size: 12px;
        color: var(--text-muted);
        overflow-wrap: anywhere;
        text-align: right;
    }

    .card-badge {
        font-size: 11px;
        padding: 3px 8px;
        border-radius: 4px;
        font-weight: 500;
    }

    .card-badge.warning {
        background: rgba(217, 119, 6, 0.15);
        color: #d97706;
    }

    .card-body {
        padding: 20px;
        min-width: 0;
    }

    /* Balance Sheet */
    .bs-section {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .bs-title {
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        color: var(--text-muted);
        margin-bottom: 4px;
    }

    .bs-row {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        font-size: 13px;
    }

    .bs-row.total {
        font-weight: 600;
        margin-top: 8px;
        padding-top: 8px;
        border-top: 1px solid var(--border);
    }

    .bs-val {
        font-family: var(--font-display);
        flex-shrink: 0;
        text-align: right;
    }

    .bs-bar {
        height: 8px;
        background: var(--bg-base);
        border-radius: 4px;
        overflow: hidden;
    }

    .bs-fill {
        height: 100%;
        border-radius: 4px;
    }

    .bs-fill.blue { background: var(--brand-indigo); }
    .bs-fill.purple { background: #8b5cf6; }
    .bs-fill.red { background: #ef4444; }
    .bs-fill.green { background: #15803d; }

    .bs-divider {
        height: 1px;
        background: var(--border);
        margin: 16px 0;
    }

    /* Ratios */
    .ratios-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 20px;
    }

    .ratio-section {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .ratio-title {
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        color: var(--text-muted);
        padding-bottom: 6px;
        border-bottom: 1px solid var(--border);
    }

    .ratio-row {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        font-size: 13px;
    }

    .ratio-val {
        font-family: var(--font-display);
        font-weight: 500;
        flex-shrink: 0;
        text-align: right;
    }

    .ratio-val.good { color: #15803d; }
    .ratio-val.caution { color: #d97706; }
    .ratio-val.danger { color: var(--text-danger); }

    /* AR Aging */
    .ar-total {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        margin-bottom: 20px;
        font-size: 14px;
    }

    .ar-total-val {
        font-weight: 600;
        font-family: var(--font-display);
        flex-shrink: 0;
        text-align: right;
    }

    .ar-buckets {
        display: flex;
        justify-content: space-around;
        align-items: flex-end;
        height: 140px;
        padding-top: 20px;
    }

    .ar-bucket {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        flex: 1;
    }

    .ar-bar-wrap {
        width: 40px;
        height: 100px;
        background: var(--bg-base);
        border-radius: 4px;
        display: flex;
        align-items: flex-end;
        overflow: hidden;
    }

    .ar-bar {
        width: 100%;
        border-radius: 4px 4px 0 0;
        transition: height 0.3s ease;
    }

    .ar-bar.green { background: #15803d; }
    .ar-bar.lime { background: #22c55e; }
    .ar-bar.orange { background: #d97706; }
    .ar-bar.red { background: #ef4444; }

    .ar-label {
        font-size: 11px;
        color: var(--text-muted);
    }

    .ar-val {
        font-size: 11px;
        font-family: var(--font-display);
        text-align: center;
        overflow-wrap: anywhere;
    }

    /* Working Capital */
    .wc-rows {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .wc-row {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        font-size: 13px;
    }

    .wc-row.total {
        font-weight: 600;
        margin-top: 8px;
        padding-top: 8px;
        border-top: 1px solid var(--border);
    }

    .wc-val {
        font-family: var(--font-display);
        flex-shrink: 0;
        text-align: right;
    }

    .ccc-box {
        margin-top: 20px;
        padding: 16px;
        background: var(--bg-base);
        border-radius: 12px;
        text-align: center;
    }

    .ccc-title {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--text-muted);
        margin-bottom: 12px;
    }

    .ccc-formula {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .ccc-item {
        font-size: 13px;
        font-family: var(--font-display);
    }

    .ccc-op {
        color: var(--text-muted);
    }

    .ccc-result {
        font-size: 16px;
        font-weight: 500;
        color: var(--brand-indigo);
        font-family: var(--font-display);
    }

    /* YoY Comparison */
    .yoy-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 24px;
    }

    .yoy-item {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .yoy-label {
        font-size: 13px;
        font-weight: 500;
        font-family: var(--font-body);
    }

    .yoy-compare {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .yoy-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .yoy-year {
        font-size: 11px;
        color: var(--text-muted);
        width: 50px;
    }

    .yoy-bar-bg {
        flex: 1;
        height: 16px;
        background: var(--bg-base);
        border-radius: 4px;
        overflow: hidden;
    }

    .yoy-bar {
        height: 100%;
        border-radius: 4px;
    }

    .yoy-bar.gray { background: #94a3b8; }
    .yoy-bar.blue { background: var(--brand-indigo); }

    .yoy-val {
        font-size: 11px;
        font-family: var(--font-display);
        width: 90px;
        text-align: right;
        overflow-wrap: anywhere;
    }

    .yoy-change {
        font-size: 13px;
        font-weight: 500;
        text-align: center;
    }

    .yoy-change.positive { color: #15803d; }
    .yoy-change.negative { color: #ef4444; }

    .yoy-note {
        margin-top: 20px;
        padding: 12px 16px;
        background: rgba(217, 119, 6, 0.1);
        border-radius: 6px;
        font-size: 12px;
        color: #92400e;
    }

    /* Footer */
    .dashboard-footer {
        display: flex;
        justify-content: space-between;
        gap: 16px;
        flex-wrap: wrap;
        font-size: 11px;
        color: var(--text-muted);
        padding-top: 16px;
        border-top: 1px solid var(--border);
    }

    @media (max-width: 1024px) {
        .kpi-row {
            grid-template-columns: repeat(2, 1fr);
        }

        .main-grid {
            grid-template-columns: 1fr;
        }

        .card.wide {
            grid-column: span 1;
        }

        .yoy-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
