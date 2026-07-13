<script lang="ts">
  import { onMount } from "svelte";
  import { motionMs } from "$lib/motion";
  import { fade } from "svelte/transition";
  import { toast } from "$lib/stores/toasts";
  import { GetReportData, ExportReport } from "../../../wailsjs/go/main/InfraService";
  import type { ReportCategory } from "$lib/types";
  import WabiSpinner from "$lib/components/ui/WabiSpinner.svelte";

  let activeCategory: ReportCategory = $state("sales");
  let loading = $state(false);
  let exporting = $state(false);
  let reportData: Record<string, any> | null = $state(null);
  let showExportModal = $state(false);
  let exportFormat = $state("csv"); // Wave 9.6: backend implements CSV only; PDF/Excel are "coming soon" (disabled)

  const categories: Array<{ id: ReportCategory; label: string; icon: string }> = [
    { id: "sales", label: "Sales", icon: "" },
    { id: "customers", label: "Customers", icon: "" },
    { id: "operations", label: "Operations", icon: "" },
    { id: "inventory", label: "Inventory", icon: "" },
    { id: "financial", label: "Financial", icon: "" },
  ];

  async function loadReportData() {
    loading = true;
    reportData = null;
    try {
      // GetReportData only accepts a fixed period preset (week/month/quarter/year),
      // not an arbitrary date range — so there is no custom range to thread through
      // here. The trailing-month window is the intended period for this dashboard.
      reportData = await GetReportData(activeCategory, "month");
    } catch (error) {
      console.error("Failed to load report data:", error);
      toast.danger(`Failed to load ${activeCategory} report`);
      reportData = null;
    } finally {
      loading = false;
    }
  }

  // Relative bar width against the max of the same field across the list —
  // avoids baking a magic scale constant into each chart.
  function barPercent(value: number, items: any[] | undefined, key: string): number {
    const list = items || [];
    const max = list.reduce((m, item) => Math.max(m, Number(item?.[key]) || 0), 0);
    if (max <= 0) return 0;
    return Math.min(100, (Number(value) / max) * 100);
  }

  async function handleExport() {
    exporting = true;
    try {
      const path = await ExportReport(
        activeCategory,
        exportFormat,
        JSON.stringify(reportData),
      );
      toast.success(`Exported to ${path || "Documents"}`);
      showExportModal = false;
    } catch (e) {
      toast.danger("Export failed");
    } finally {
      exporting = false;
    }
  }

  onMount(loadReportData);
</script>

<div class="page">
  <header class="header">
    <div class="header-content">
      <h1>Business Intelligence.</h1>
      <p class="subtitle">Reports & Analytics</p>
    </div>
    <div class="actions">
      <button class="btn-refresh" onclick={loadReportData}>Refresh</button>
      <button class="btn-primary" onclick={() => (showExportModal = true)}
        >Export Report</button
      >
    </div>
  </header>

  <div class="tabs-bar">
    {#each categories as cat}
      <button
        class="tab"
        class:active={activeCategory === cat.id}
        onclick={() => {
          activeCategory = cat.id;
          loadReportData();
        }}
      >
        <span class="icon">{cat.icon}</span> <span>{cat.label}</span>
      </button>
    {/each}
  </div>

  <main class="main-panel">
    {#if loading}
      <div class="loading"><WabiSpinner size="lg" tempo="calm" /></div>
    {:else if reportData}
      <div class="report-container" in:fade={{ duration: motionMs(400) }}>
        {#if activeCategory === "sales"}
          <div class="stats-row">
            <div class="stat-card">
              <div class="val">{((reportData.win_rate || 0) * 100).toFixed(1)}%</div>
              <div class="lbl">Win Rate</div>
            </div>
            <div class="stat-card">
              <div class="val">
                {((reportData.conversion_rate || 0) * 100).toFixed(1)}%
              </div>
              <div class="lbl">Conversion</div>
            </div>
            <div class="stat-card">
              <div class="val">{(reportData.avg_deal_size || 0).toLocaleString()} BHD</div>
              <div class="lbl">Avg Deal Size</div>
            </div>
          </div>
          <div class="chart-panel">
            <h3>Pipeline Value</h3>
            <!-- Simple Bar Chart Visualization -->
            <div class="bars">
              {#each reportData.pipeline || [] as item}
                <div class="bar-row">
                  <span class="bar-lbl">{item.stage}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill"
                      style="width: {barPercent(item.value, reportData.pipeline, 'value')}%"
                    ></div>
                  </div>
                  <span class="bar-val">{item.value.toLocaleString()}</span>
                </div>
              {/each}
            </div>
          </div>
        {:else if activeCategory === "customers"}
          <div class="stats-row">
            <div class="stat-card">
              <div class="val">{(reportData.avg_payment_days || 0).toFixed(0)} days</div>
              <div class="lbl">Avg Payment Days</div>
            </div>
            <div class="stat-card">
              <div class="val">
                {((reportData.collection_efficiency || 0) * 100).toFixed(1)}%
              </div>
              <div class="lbl">Collection Efficiency</div>
            </div>
          </div>
          <div class="chart-panel">
            <h3>Grade Distribution</h3>
            <div class="bars">
              {#each reportData.grade_distribution || [] as grade}
                <div class="bar-row">
                  <span class="bar-lbl">Grade {grade.grade}</span>
                  <div class="bar-track">
                    <div class="bar-fill" style="width: {grade.percentage}%"></div>
                  </div>
                  <span class="bar-val">{grade.count} ({grade.percentage.toFixed(0)}%)</span>
                </div>
              {/each}
              {#if !(reportData.grade_distribution || []).length}
                <p class="empty-note">No graded customers yet.</p>
              {/if}
            </div>
          </div>
          <div class="chart-panel">
            <h3>Customer Type</h3>
            <div class="bars">
              {#each reportData.type_distribution || [] as t}
                <div class="bar-row">
                  <span class="bar-lbl">{t.label}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill"
                      style="width: {barPercent(t.count, reportData.type_distribution, 'count')}%"
                    ></div>
                  </div>
                  <span class="bar-val">{t.count}</span>
                </div>
              {/each}
            </div>
          </div>
        {:else if activeCategory === "operations"}
          <div class="stats-row">
            <div class="stat-card">
              <div class="val">{reportData.avg_lead_time || 0} days</div>
              <div class="lbl">Avg Lead Time</div>
            </div>
            <div class="stat-card">
              <div class="val">{((reportData.on_time_delivery || 0) * 100).toFixed(1)}%</div>
              <div class="lbl">On-Time Delivery</div>
            </div>
            <div class="stat-card">
              <div class="val">{reportData.pending_shipments || 0}</div>
              <div class="lbl">Pending Shipments</div>
            </div>
          </div>
          <div class="chart-panel">
            <h3>Orders by Stage</h3>
            <div class="bars">
              {#each reportData.orders_by_stage || [] as s}
                <div class="bar-row">
                  <span class="bar-lbl">{s.stage}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill"
                      style="width: {barPercent(s.count, reportData.orders_by_stage, 'count')}%"
                    ></div>
                  </div>
                  <span class="bar-val">{s.count}</span>
                </div>
              {/each}
            </div>
          </div>
        {:else if activeCategory === "inventory"}
          <div class="stats-row">
            <div class="stat-card">
              <div class="val">{reportData.total_items || 0}</div>
              <div class="lbl">Total Items</div>
            </div>
            <div class="stat-card">
              <div class="val">{(reportData.total_value || 0).toLocaleString()} BHD</div>
              <div class="lbl">Total Value</div>
            </div>
            <div class="stat-card" class:danger={(reportData.low_stock_alerts || 0) > 0}>
              <div class="val">{reportData.low_stock_alerts || 0}</div>
              <div class="lbl">Low Stock Alerts</div>
            </div>
          </div>
          <div class="chart-panel">
            <h3>Stock Movements (this window)</h3>
            <div class="bars">
              {#each reportData.movements || [] as m}
                <div class="bar-row">
                  <span class="bar-lbl">{m.type}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill"
                      style="width: {barPercent(m.value, reportData.movements, 'value')}%"
                    ></div>
                  </div>
                  <span class="bar-val">{m.count} / {m.value.toLocaleString()} BHD</span>
                </div>
              {/each}
              {#if !(reportData.movements || []).length}
                <p class="empty-note">No stock movements recorded in this window.</p>
              {/if}
            </div>
          </div>
        {:else if activeCategory === "financial"}
          <div class="stats-row">
            <div class="stat-card">
              <div class="val">
                {(reportData.receivables_outstanding || 0).toLocaleString()} BHD
              </div>
              <div class="lbl">Receivables Outstanding</div>
            </div>
            <div class="stat-card">
              <div class="val">
                {(reportData.payables_outstanding || 0).toLocaleString()} BHD
              </div>
              <div class="lbl">Payables Outstanding</div>
            </div>
            <div class="stat-card">
              <div class="val">
                {(reportData.avg_monthly_revenue || 0).toLocaleString()} BHD
              </div>
              <div class="lbl">Avg Monthly Revenue</div>
            </div>
          </div>
          <div class="chart-panel">
            <h3>Collections This Period</h3>
            <div class="progress-lg">
              <div
                class="prog-fill"
                style="width: {Math.min(100, ((reportData.collected || 0) / (reportData.collection_target || 1)) * 100)}%"
              ></div>
              <span class="prog-text"
                >{(reportData.collected || 0).toLocaleString()} / {(reportData.collection_target || 0).toLocaleString()}
                BHD</span
              >
            </div>
          </div>
          <div class="chart-panel">
            <h3>Overdue Receivables by Aging</h3>
            <div class="bars">
              {#each reportData.overdue || [] as bucket}
                <div class="bar-row">
                  <span class="bar-lbl">{bucket.days}</span>
                  <div class="bar-track">
                    <div
                      class="bar-fill"
                      style="width: {barPercent(bucket.amount, reportData.overdue, 'amount')}%"
                    ></div>
                  </div>
                  <span class="bar-val">{bucket.amount.toLocaleString()} BHD</span>
                </div>
              {/each}
              {#if !(reportData.overdue || []).length}
                <p class="empty-note">No receivables overdue 30+ days.</p>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>

{#if showExportModal}
  <div class="modal-backdrop" transition:fade={{ duration: motionMs(400) }}>
    <div class="modal">
      <h3>Export Report</h3>
      <div class="form-group">
        <div id="report-format-label">Format</div>
        <div class="toggle-group" role="group" aria-labelledby="report-format-label">
          <button
            class:active={exportFormat === "csv"}
            onclick={() => (exportFormat = "csv")}>CSV</button
          >
          <button disabled title="PDF export coming soon">PDF</button>
          <button disabled title="Excel export coming soon">Excel</button>
        </div>
      </div>
      <div class="modal-actions">
        <button class="btn-ghost" onclick={() => (showExportModal = false)}
          >Cancel</button
        >
        <button
          class="btn-primary"
          onclick={handleExport}
          disabled={exporting}
        >
          {exporting ? "Exporting..." : "Download"}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    padding: var(--page-padding);
    height: 100vh;
    background: var(--paper);
    color: var(--ink);
    display: flex;
    flex-direction: column;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    margin-bottom: var(--space-4);
    flex-shrink: 0;
  }
  h1 {
    font-size: var(--text-3xl);
    font-weight: var(--font-weight-light);
    margin: 0;
    letter-spacing: -0.02em;
  }
  .subtitle {
    color: var(--ink-faint);
    margin-top: var(--space-1);
    font-size: var(--text-sm);
  }

  .actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }
  .btn-refresh {
    background: var(--paper-subtle);
    color: var(--ink);
    border: 1px solid var(--border-medium);
    padding: 8px 16px;
    border-radius: var(--radius-pill);
    cursor: pointer;
    font-size: 13px;
  }
  .btn-refresh:hover {
    border-color: var(--ink-light);
  }

  .btn-primary {
    background: var(--ink);
    color: var(--paper);
    border: none;
    padding: 8px 16px;
    border-radius: var(--radius-pill);
    cursor: pointer;
    font-size: 13px;
  }

  .tabs-bar {
    display: flex;
    gap: 8px;
    margin-bottom: var(--space-4);
    border-bottom: 1px solid var(--border-subtle);
    padding-bottom: 8px;
  }
  .tab {
    background: transparent;
    border: 1px solid transparent;
    padding: 6px 14px;
    border-radius: 20px;
    cursor: pointer;
    display: flex;
    gap: 6px;
    align-items: center;
    color: var(--ink-light);
    font-size: 13px;
  }
  .tab:hover {
    background: var(--paper-subtle);
  }
  .tab.active {
    background: var(--ink);
    color: var(--paper);
  }
  .tab .icon {
    font-size: 14px;
  }

  .main-panel {
    flex: 1;
    overflow-y: auto;
  }
  .loading {
    display: flex;
    justify-content: center;
    padding-top: 40px;
  }

  .report-container {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .stats-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--space-4);
  }
  .stat-card {
    background: var(--paper-subtle);
    padding: var(--space-4);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-subtle);
  }
  .stat-card .val {
    font-size: 24px;
    font-weight: 300;
    margin-bottom: 4px;
  }
  .stat-card .lbl {
    font-size: 10px;
    text-transform: uppercase;
    color: var(--ink-light);
    letter-spacing: 1px;
  }
  .stat-card.danger {
    background: #fee2e2;
    border-color: #fca5a5;
    color: #991b1b;
  }
  .stat-card.danger .lbl {
    color: #7f1d1d;
  }

  .chart-panel {
    background: var(--paper-subtle);
    padding: var(--space-4);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-subtle);
  }
  .chart-panel h3 {
    margin: 0 0 16px;
    font-size: 13px;
    text-transform: uppercase;
    color: var(--ink-light);
  }

  .bar-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 8px;
    font-size: 13px;
  }
  .bar-lbl {
    width: 100px;
    text-align: right;
  }
  .bar-track {
    flex: 1;
    height: 8px;
    background: rgba(0, 0, 0, 0.05);
    border-radius: 4px;
    overflow: hidden;
  }
  .bar-fill {
    height: 100%;
    background: var(--ink);
  }
  .bar-val {
    width: 80px;
    font-family: var(--font-mono);
  }
  .empty-note {
    color: var(--ink-faint);
    font-size: 12px;
    margin: 0;
  }

  .progress-lg {
    height: 24px;
    background: rgba(0, 0, 0, 0.05);
    border-radius: 12px;
    position: relative;
    overflow: hidden;
  }
  .prog-fill {
    height: 100%;
    background: #059669;
  }
  .prog-text {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 500;
    color: #fff;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  }

  /* Modal */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  .modal {
    background: var(--paper);
    padding: 32px;
    border-radius: 16px;
    width: 320px;
  }
  .toggle-group {
    display: flex;
    gap: 8px;
    margin-top: 8px;
  }
  .toggle-group button {
    flex: 1;
    padding: 8px;
    border: 1px solid var(--border-medium);
    background: transparent;
    border-radius: 6px;
    cursor: pointer;
  }
  .toggle-group button.active {
    background: var(--ink);
    color: var(--paper);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 24px;
  }
  .btn-ghost {
    background: transparent;
    border: none;
    cursor: pointer;
  }
</style>
