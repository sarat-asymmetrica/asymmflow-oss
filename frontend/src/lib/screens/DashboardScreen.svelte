<script lang="ts">
    import { run } from 'svelte/legacy';

    import { onDestroy, onMount } from "svelte";
    import { fade } from "svelte/transition";
    import { toast } from "$lib/stores/toasts";
    import { t } from "$lib/i18n";
    import { currentUser, permissions } from "../stores/authContext";
    import { devLog } from "$lib/utils/devLog";
    import { GetDashboardStats, GetDashboardPipelineByStageYTD, GetDashboardARAgingReportYTD } from "../../../wailsjs/go/main/App";
import { ListFollowUps } from "../../../wailsjs/go/main/CRMService";
    import { EventsOff, EventsOn } from "../../../wailsjs/runtime/runtime";
    import { main, crm } from "../../../wailsjs/go/models";
    import ContextTaskModal from "../components/ContextTaskModal.svelte";
    import { listMyTasks, listTeamTasks, refreshCollaborativeWorkspace, type CollaborativeTask } from "$lib/api/collaboration";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";

    let loading = $state(true);
    let now = $state(new Date());
    let clockTimer: ReturnType<typeof setInterval> | undefined;

    function hasPermission(permission: string): boolean {
        const permissionList = Array.isArray(activePermissions) ? activePermissions : [];
        if (permissionList.includes("*") || permissionList.includes(permission)) return true;
        const [resource] = permission.split(":");
        return permissionList.includes(`${resource}:*`);
    }

    let stats = $state({
        activeRFQs: 0,
        activeOrders: 0,
        winRate: 0,
        totalRevenue: 0,
        revenueMeta: "Confirmed business",
        revenueGrowth: 0,
        activityYear: new Date().getFullYear(),
        pipelineValue: 0,
        collectionRate: 0,
        outstandingAR: 0,
        arDaysOverdue: 0,
        pendingInvoices: 0,
        cashBalance: 0,
        cashPositionNote: "",
        freshStartDate: "",
    });

    // Wave 9.2 C2: pipeline-by-stage + AR-aging widgets. Each loads independently
    // of the main dashboard stats so a slow/failed backend doesn't block the rest
    // of the page - see loadPipelineByStage / loadARAging below.
    let pipelineStages: main.SalesPipelineData[] = $state([]);
    let pipelineLoading = $state(true);
    let pipelineError = $state("");

    let arAging: main.ARAgingReport | null = $state(null);
    let agingLoading = $state(true);
    let agingError = $state("");

    let followUpTasks: crm.FollowUpTask[] = $state([]);
    let dashboardTasks: CollaborativeTask[] = $state([]);
    let activeDashboardTasks: CollaborativeTask[] = $state([]);
    let followUpSummary = $state("* No follow-up issues");
    let totalTaskSignals = $state(0);
    let taskModalOpen = $state(false);
    let taskSignalsPermissionLoaded = $state(false);
    let taskSignalRequestSeq = 0;






    async function loadData() {
        loading = true;
        try {
            const [dashStats] = await Promise.all([
                GetDashboardStats(),
                loadPipelineByStage(),
                loadARAging(),
            ]);
            if (dashStats) {
                const rawStats = dashStats as any;
                stats.activeRFQs = rawStats.active_rfqs || 0;
                stats.activeOrders = rawStats.active_orders || 0;
                stats.winRate = rawStats.win_rate || 0;
                stats.totalRevenue = rawStats.total_revenue || 0;
                stats.revenueMeta =
                    rawStats.revenue_meta ||
                    `Confirmed business FY${rawStats.activity_year || new Date().getFullYear()}`;
                stats.revenueGrowth = rawStats.month_growth || 0;
                stats.activityYear = rawStats.activity_year || new Date().getFullYear();
                stats.pipelineValue = rawStats.pipeline_value_bhd || 0;
                stats.collectionRate = rawStats.collection_rate || 0;
                stats.outstandingAR = rawStats.outstanding_ar || 0;
                stats.arDaysOverdue = rawStats.ar_days_overdue || 0;
                stats.pendingInvoices = rawStats.pending_invoices || 0;
                stats.cashBalance = rawStats.cash_balance_bhd || 0;
                stats.cashPositionNote = rawStats.cash_position_note || "";
                stats.freshStartDate = rawStats.fresh_start_date || "";
            }

        } catch (err) {
            const errorMsg = err?.message || String(err);
            toast.danger(`Failed to load dashboard data: ${errorMsg}`);
        } finally {
            loading = false;
        }

        void loadTaskSignals({ refreshRemote: true });
    }

    // loadPipelineByStage / loadARAging never throw - each swallows its own
    // failure into a widget-scoped error so Promise.all in loadData() can't be
    // short-circuited by one backend having a bad day.
    async function loadPipelineByStage() {
        pipelineLoading = true;
        pipelineError = "";
        try {
            pipelineStages = (await GetDashboardPipelineByStageYTD()) || [];
        } catch (err) {
            pipelineError = err?.message || String(err);
            pipelineStages = [];
        } finally {
            pipelineLoading = false;
        }
    }

    async function loadARAging() {
        agingLoading = true;
        agingError = "";
        try {
            arAging = await GetDashboardARAgingReportYTD();
        } catch (err) {
            agingError = err?.message || String(err);
            arAging = null;
        } finally {
            agingLoading = false;
        }
    }

    async function loadTaskSignals(options: { refreshRemote?: boolean } = {}) {
        const requestSeq = ++taskSignalRequestSeq;
        if (!canViewTasks) {
            if (activePermissions.length > 0) {
                followUpTasks = [];
                dashboardTasks = [];
            }
            return;
        }

        if (options.refreshRemote) {
            await refreshCollaborativeWorkspace({ minIntervalMs: 5_000 }).catch((err) => {
                devLog.warn("Dashboard collaborative refresh failed", err);
            });
        }

        const [legacyRows, myTaskRows] = await Promise.all([
            ListFollowUps(5).catch((err) => {
                devLog.warn("Dashboard follow-ups failed", err);
                return [];
            }),
            listMyTasks(false).catch((err) => {
                devLog.warn("Dashboard personal tasks failed", err);
                return [];
            }),
        ]);
        if (requestSeq !== taskSignalRequestSeq) return;
        followUpTasks = legacyRows || [];
        dashboardTasks = myTaskRows || [];
        if (dashboardTasks.length === 0 && Array.isArray(activePermissions) && activePermissions.includes("*")) {
            dashboardTasks = await listTeamTasks(false).catch((err) => {
                devLog.warn("Dashboard team tasks failed", err);
                return [];
            });
        }
    }



    function formatCurrency(val: number): string {
        return `${Number(val || 0).toLocaleString("en-US", {
            minimumFractionDigits: 0,
            maximumFractionDigits: 0,
        })} BHD`;
    }

    function formatCompactCurrency(val: number): string {
        const amount = Number(val || 0);
        return `${amount.toLocaleString("en-US", { maximumFractionDigits: 0 })} BHD`;
    }

    function formatCount(val: number): string {
        return Number(val || 0).toLocaleString("en-US", { maximumFractionDigits: 0 });
    }

    function formatDate(value: Date = new Date()): string {
        return value.toLocaleDateString("en-GB", {
            weekday: "long",
            day: "numeric",
            month: "long",
            year: "numeric",
        });
    }

    function getTimeGreeting(value: Date): string {
        const hour = value.getHours();
        if (hour < 12) return "Good morning";
        if (hour < 16) return "Good afternoon";
        return "Good evening";
    }

    function formatUserName(user: any): string {
        const raw =
            user?.preferred_name ||
            user?.full_name ||
            user?.display_name ||
            user?.name ||
            user?.username ||
            "";
        const cleaned = String(raw).replace(/[._-]+/g, " ").trim();
        if (!cleaned) return "there";
        const firstName = cleaned.split(/\s+/)[0];
        return firstName.charAt(0).toUpperCase() + firstName.slice(1);
    }

    function summarizeText(value: string | undefined, maxLength = 78): string {
        const compact = String(value || "").replace(/\s+/g, " ").trim();
        if (!compact) return "No issue summary";
        if (compact.length <= maxLength) return compact;
        const clipped = compact.slice(0, maxLength - 3);
        const lastSpace = clipped.lastIndexOf(" ");
        return `${clipped.slice(0, lastSpace > 42 ? lastSpace : clipped.length)}...`;
    }

    function buildFollowUpSummary(
        legacyFollowUps: crm.FollowUpTask[],
        activeTasks: CollaborativeTask[],
    ): string {
        if (legacyFollowUps.length > 0) {
            const first = legacyFollowUps[0];
            return `* ${summarizeText(first.title || first.description || first.notes || "Follow-up pending")}`;
        }
        if (activeTasks.length > 0) {
            return `* ${summarizeText(activeTasks[0].title || activeTasks[0].description || "Task pending")}`;
        }
        return "* No follow-up issues";
    }

    function taskAge(task: any): string {
        const rawDate = task.due_date || task.last_comment_at || task.created_at || task.updated_at || task.started_at;
        if (!rawDate) return "recent";
        const date = new Date(rawDate as any);
        if (Number.isNaN(date.getTime())) return "recent";
        const diffDays = Math.round((date.getTime() - Date.now()) / (24 * 60 * 60 * 1000));
        if (diffDays === 0) return "today";
        if (diffDays === 1) return "tomorrow";
        if (diffDays > 1) return `${diffDays}d`;
        if (diffDays === -1) return "yesterday";
        return `${Math.abs(diffDays)}d ago`;
    }

    function taskInitial(task: any): string {
        return (task.priority || task.status || "T").slice(0, 1).toUpperCase();
    }

    function taskMeta(task: CollaborativeTask): string {
        const pieces = [
            task.priority ? `${task.priority} priority` : "",
            task.assignee_name ? `assigned to ${task.assignee_name}` : "",
            task.status || "",
        ].filter(Boolean);
        return pieces.join(" / ") || "Task";
    }

    function navigateTo(target: Record<string, string>) {
        window.dispatchEvent(new CustomEvent("navigateToScreen", { detail: target }));
    }

    // Wave 9 B1: dashboard task rows drill into Work Hub and open the task,
    // mirroring NotificationsScreen's openNotification handoff exactly.
    const pendingTaskStorageKey = "asymmflow.pendingCollaborativeTaskId";
    function openTask(task: CollaborativeTask) {
        if (!task?.id) return;
        sessionStorage.setItem(pendingTaskStorageKey, task.id);
        navigateTo({ screen: "work" });
        window.setTimeout(() => {
            window.dispatchEvent(new CustomEvent("openCollaborativeTask", { detail: { taskID: task.id } }));
        }, 80);
    }

    // Wave 9.2 C2: pipeline stage -> Opportunities, pre-filtered by stage.
    // Mirrors the review_pipeline_won drill below (navigateTo({ screen: "opportunities", stage })),
    // which OpportunitiesScreen already consumes via its `params.stage` prop.
    function pipelineStageTone(stageName: string): string {
        const normalized = String(stageName || "").trim().toLowerCase();
        if (normalized === "won") return "positive";
        if (normalized === "lost") return "negative";
        return "neutral";
    }

    function handlePipelineStageClick(stageName: string) {
        navigateTo({ screen: "opportunities", stage: stageName });
    }

    // AR aging bucket -> Invoices, pre-filtered. C5: InvoicesScreen now also
    // supports a day-range aging-bucket filter (days_30/60/90/120_plus)
    // alongside the existing status filters, so each overdue bar narrows to
    // its OWN bucket via agingBucket, on top of the "Overdue" status filter
    // used by the existing chase_overdue/review_ar drill. The Current bucket
    // (not yet due) still opens the tab unfiltered.
    function handleAgingBucketClick(bucket: { invoiceFilter?: string; agingBucket?: string }) {
        navigateTo({
            screen: "finance",
            tab: "invoices",
            company: "Acme Instrumentation",
            ...(bucket.invoiceFilter ? { invoiceFilter: bucket.invoiceFilter } : {}),
            ...(bucket.agingBucket ? { agingBucket: bucket.agingBucket } : {}),
        });
    }

    function handleDashboardAction(actionId: string) {
        switch (actionId) {
            case "review_cash":
                // Bank Recon tab id in FinanceHub is "bank_recon" (underscore) -
                // the previous "bank-recon" value never matched, so this action
                // silently landed on the Finance dashboard tab instead.
                navigateTo({ screen: "finance", tab: "bank_recon", company: "Acme Instrumentation" });
                return;
            case "review_credit":
            case "chase_overdue":
            case "review_ar":
                navigateTo({
                    screen: "finance",
                    tab: "invoices",
                    company: "Acme Instrumentation",
                    ...(stats.arDaysOverdue > 0 ? { invoiceFilter: "Overdue" } : {}),
                });
                return;
            case "review_revenue":
                navigateTo({ screen: "finance", tab: "dashboard", company: "Acme Instrumentation" });
                return;
            case "open_followups":
                navigateTo({ screen: "work" });
                return;
            case "review_pipeline":
                navigateTo({ screen: "opportunities" });
                return;
            case "review_pipeline_won":
                navigateTo({ screen: "opportunities", stage: "Won" });
                return;
            case "review_orders":
                navigateTo({ screen: "operations" });
                return;
            default:
                toast.warning("Dashboard action is not available yet");
        }
    }

    function openTaskModal() {
        taskModalOpen = true;
    }

    async function handleTaskCreated() {
        taskModalOpen = false;
        await loadData();
    }

    onMount(() => {
        now = new Date();
        clockTimer = setInterval(() => {
            now = new Date();
        }, 60_000);
        loadData();

        EventsOn("data:refresh", (data: any) => {
            devLog.info("Dashboard received data:refresh event", data);
            loadData();
        });
        EventsOn("tasks:updated", () => void loadTaskSignals());
        EventsOn("notifications:updated", () => void loadTaskSignals());
        EventsOn("employees:updated", () => void loadTaskSignals({ refreshRemote: true }));
    });

    onDestroy(() => {
        if (clockTimer) {
            clearInterval(clockTimer);
        }
        EventsOff("data:refresh", "tasks:updated", "notifications:updated", "employees:updated");
    });
    let userName = $derived(formatUserName($currentUser));
    let greeting = $derived(getTimeGreeting(now));
    let activePermissions = $derived(Array.isArray($permissions) ? $permissions : []);
    let canViewTasks = $derived(hasPermission("tasks:view"));
    let canViewFinance = $derived(hasPermission("finance:view"));
    run(() => {
        activeDashboardTasks = dashboardTasks
            .filter((task) => !["completed", "archived", "cancelled", "canceled"].includes((task.status || "").toLowerCase()))
            .slice(0, 5);
    });
    run(() => {
        followUpSummary = buildFollowUpSummary(followUpTasks, activeDashboardTasks);
    });
    run(() => {
        totalTaskSignals = followUpTasks.length + activeDashboardTasks.length;
    });
    let pressureLevel = $derived(canViewFinance
        ? stats.cashBalance > 0
            ? "Cash position is the primary decision surface today"
            : stats.pipelineValue > stats.outstandingAR
              ? "Pipeline quality is the primary decision surface today"
              : "Commercial finance is the primary decision surface today"
        : totalTaskSignals > 0
          ? "Follow-ups and pipeline conversion are the primary decision surface today"
          : "Pipeline conversion is the primary decision surface today");
    let kpis = $derived(canViewFinance
        ? [
              {
                  label: `Revenue FY${stats.activityYear}`,
                  value: formatCompactCurrency(stats.totalRevenue),
                  meta: stats.revenueMeta,
                  tone: stats.revenueGrowth >= 0 ? "positive" : "negative",
                  delta:
                      stats.revenueGrowth === 0
                          ? "flat"
                          : `${stats.revenueGrowth > 0 ? "+" : ""}${stats.revenueGrowth.toFixed(1)}% vs last month`,
                  actionId: "review_revenue",
              },
              {
                  label: "Cash Balance",
                  value: formatCompactCurrency(stats.cashBalance),
                  meta: "Latest bank statements",
                  tone: stats.cashPositionNote ? "warning" : "positive",
                  delta: stats.cashPositionNote ? "check statements" : "current",
                  actionId: "review_cash",
              },
              {
                  label: "Accounts Receivable",
                  value: formatCompactCurrency(stats.outstandingAR),
                  meta:
                      stats.pendingInvoices > 0
                          ? `${stats.pendingInvoices} invoice/order exposure${stats.pendingInvoices === 1 ? "" : "s"} open`
                          : "No open AR",
                  tone: stats.pendingInvoices > 0 ? "negative" : "positive",
                  delta:
                      stats.arDaysOverdue > 0
                          ? `${stats.arDaysOverdue} days average overdue`
                          : "current",
                  actionId: "review_ar",
              },
              {
                  label: "Pipeline",
                  value: formatCompactCurrency(stats.pipelineValue),
                  meta: "Active commercial exposure",
                  tone: "neutral",
                  delta: "weighted by live opportunities",
                  actionId: "review_pipeline",
              },
          ]
        : [
              {
                  label: "Active RFQs",
                  value: formatCount(stats.activeRFQs),
                  meta: "Open sales opportunities",
                  tone: stats.activeRFQs > 0 ? "neutral" : "positive",
                  delta: "needs qualification",
                  actionId: "review_pipeline",
              },
              {
                  label: "Pipeline",
                  value: formatCompactCurrency(stats.pipelineValue),
                  meta: "Active quotation value",
                  tone: "neutral",
                  delta: "live opportunity data",
                  actionId: "review_pipeline",
              },
              {
                  label: "Active Orders",
                  value: formatCount(stats.activeOrders),
                  meta: "Confirmed handoffs",
                  tone: stats.activeOrders > 0 ? "positive" : "neutral",
                  delta: "operations visible",
                  actionId: "review_orders",
              },
              {
                  label: "Win Rate",
                  value: `${Math.round(stats.winRate)}%`,
                  meta: `FY${stats.activityYear} closed opportunities`,
                  tone: stats.winRate >= 20 ? "positive" : "warning",
                  delta: "won / closed",
                  actionId: "review_pipeline_won",
              },
          ]);
    let operatingFocus = $derived(canViewFinance
        ? [
              {
                  title: "Cash balance",
                  detail:
                      stats.cashPositionNote
                          ? stats.cashPositionNote
                          : `Bank cash is ${formatCurrency(stats.cashBalance)}`,
                  tone: stats.cashPositionNote ? "warning" : "clear",
                  action: "Review",
                  actionId: "review_cash",
              },
              {
                  title: "Accounts receivable",
                  detail:
                      stats.outstandingAR > 0
                          ? `Open receivables are ${formatCurrency(stats.outstandingAR)}`
                          : "No open receivables",
                  tone: stats.outstandingAR > 0 ? "warning" : "clear",
                  action: "Open",
                  actionId: "review_ar",
              },
              {
                  title: "Next follow-up",
                  detail:
                      followUpTasks.length > 0
                          ? summarizeText(followUpTasks[0].title || followUpTasks[0].description || "Follow-up pending", 96)
                          : activeDashboardTasks.length > 0
                          ? activeDashboardTasks[0].title || "Task pending"
                          : "No follow-up task queued",
                  tone: totalTaskSignals > 0 ? "info" : "clear",
                  action: "Open",
                  actionId: "open_followups",
              },
          ]
        : [
              {
                  title: "Next follow-up",
                  detail:
                      followUpTasks.length > 0
                          ? summarizeText(followUpTasks[0].title || followUpTasks[0].description || "Follow-up pending", 96)
                          : activeDashboardTasks.length > 0
                          ? activeDashboardTasks[0].title || "Task pending"
                          : "No follow-up task queued",
                  tone: totalTaskSignals > 0 ? "info" : "clear",
                  action: "Open",
                  actionId: "open_followups",
              },
              {
                  title: "Opportunity pipeline",
                  detail: `${stats.activeRFQs} active RFQs with ${formatCurrency(stats.pipelineValue)} in quoted/open value`,
                  tone: stats.activeRFQs > 0 ? "info" : "clear",
                  action: "Open",
                  actionId: "review_pipeline",
              },
              {
                  title: "Order handoff",
                  detail: `${stats.activeOrders} active orders visible for sales coordination`,
                  tone: stats.activeOrders > 0 ? "clear" : "soft",
                  action: "Open",
                  actionId: "review_orders",
              },
          ]);
    let alerts = $derived(canViewFinance
        ? [
              {
                  label: "Statements",
                  text: stats.cashPositionNote || `Cash balance ${formatCurrency(stats.cashBalance)}`,
                  tone: stats.cashPositionNote ? "warning" : "clear",
              },
              {
                  label: "Receivables",
                  text:
                      stats.pendingInvoices > 0
                          ? `${stats.pendingInvoices} invoice/order exposure${stats.pendingInvoices === 1 ? "" : "s"} open`
                          : "No open invoices or uninvoiced orders",
                  tone: stats.pendingInvoices > 0 ? "warning" : "clear",
              },
              {
                  label: "Pipeline",
                  text: `Commercial exposure at ${formatCurrency(stats.pipelineValue)}`,
                  tone: "info",
              },
              {
                  label: "Follow-ups",
                  text: followUpSummary,
                  tone: totalTaskSignals > 0 ? "info" : "soft",
              },
          ]
        : [
              {
                  label: "Follow-ups",
                  text: followUpSummary,
                  tone: totalTaskSignals > 0 ? "info" : "soft",
              },
              {
                  label: "RFQs",
                  text: `${stats.activeRFQs} active opportunities need stage discipline`,
                  tone: stats.activeRFQs > 0 ? "info" : "clear",
              },
              {
                  label: "Orders",
                  text: `${stats.activeOrders} confirmed orders in active handoff`,
                  tone: stats.activeOrders > 0 ? "info" : "clear",
              },
          ]);
    let pipelineMaxValue = $derived(
        pipelineStages.reduce((max, stage) => Math.max(max, stage.value || 0), 0),
    );
    function pipelineStageWidth(stage: main.SalesPipelineData): number {
        if (!pipelineMaxValue) return 0;
        return Math.max(4, Math.round(((stage.value || 0) / pipelineMaxValue) * 100));
    }
    let agingBuckets = $derived.by(() => {
        if (!arAging) return [];
        const raw: { key: string; label: string; amount: number; tone: string; invoiceFilter?: string; agingBucket?: string }[] = [
            { key: "current", label: "Current", amount: arAging.current, tone: "positive" },
            { key: "days_30", label: "1-30d", amount: arAging.days_30, tone: "neutral", invoiceFilter: "Overdue", agingBucket: "days_30" },
            { key: "days_60", label: "31-60d", amount: arAging.days_60, tone: "warning", invoiceFilter: "Overdue", agingBucket: "days_60" },
            { key: "days_90", label: "61-90d", amount: arAging.days_90, tone: "warning", invoiceFilter: "Overdue", agingBucket: "days_90" },
            { key: "days_120_plus", label: "90d+", amount: arAging.days_120_plus, tone: "negative", invoiceFilter: "Overdue", agingBucket: "days_120_plus" },
        ];
        const total = arAging.total || 0;
        return raw.map((bucket) => ({
            ...bucket,
            width: total > 0 ? Math.max(4, Math.round((bucket.amount / total) * 100)) : 0,
        }));
    });
    run(() => {
        if (!canViewTasks && taskSignalsPermissionLoaded) {
            taskSignalsPermissionLoaded = false;
        }
    });
    run(() => {
        if (canViewTasks && !taskSignalsPermissionLoaded) {
            taskSignalsPermissionLoaded = true;
            void loadTaskSignals({ refreshRemote: true });
        }
    });
</script>

<div class="dashboard-page">
    <header class="dashboard-header">
        <div>
            <p class="brand-kicker">AsymmFlow</p>
            <h1>{greeting}, {userName}</h1>
            <p class="dashboard-subtitle">{pressureLevel}</p>
        </div>
        <time datetime={now.toISOString()}>{formatDate(now)}</time>
    </header>

    {#if loading}
        <div class="loading-state"><WabiSpinner size="lg" tempo="calm" /></div>
    {:else}
        <main class="dashboard-shell" in:fade>
            <section class="kpi-strip" aria-label={t("dashboard.kpis")}>
                {#each kpis as kpi}
                    <div
                        class="metric-card"
                        data-tone={kpi.tone}
                        role="button"
                        tabindex="0"
                        onclick={() => handleDashboardAction(kpi.actionId)}
                        onkeydown={(event) => {
                            if (event.key === "Enter" || event.key === " ") {
                                event.preventDefault();
                                handleDashboardAction(kpi.actionId);
                            }
                        }}
                        aria-label={`Open ${kpi.label}`}
                    >
                        <div class="metric-label">{kpi.label}</div>
                        <div class="metric-value">{kpi.value}</div>
                        <div class="metric-meta">
                            <span>{kpi.meta}</span>
                            <strong>{kpi.delta}</strong>
                        </div>
                    </div>
                {/each}
            </section>

            <section class="decision-grid">
                <article class="panel operating-panel">
                    <div class="panel-head">
                        <span>Operating Focus</span>
                        <strong>{totalTaskSignals} active follow-ups</strong>
                    </div>
                    <h2>{pressureLevel}</h2>
                    <div class="focus-list">
                        {#each operatingFocus as item}
                            <div class="focus-row" data-tone={item.tone}>
                                <div>
                                    <strong>{item.title}</strong>
                                    <span>{item.detail}</span>
                                </div>
                                <button
                                    class="focus-action"
                                    type="button"
                                    onclick={() => handleDashboardAction(item.actionId)}
                                    aria-label={`${item.action} ${item.title}`}
                                >
                                    {item.action}
                                </button>
                            </div>
                        {/each}
                    </div>
                </article>

                <article class="panel alerts-panel">
                    <div class="panel-head">
                        <span>{t("dashboard.alerts")}</span>
                        <strong>{alerts.filter((alert) => alert.tone !== "clear").length} live</strong>
                    </div>
                    <div class="alert-list">
                        {#each alerts as alert}
                            <div class="alert-row" data-tone={alert.tone}>
                                <strong>{alert.label}</strong>
                                <span>{alert.text}</span>
                            </div>
                        {/each}
                    </div>
                </article>

                <article class="panel pipeline-panel">
                    <div class="panel-head">
                        <span>Pipeline by Stage</span>
                        <strong>{pipelineStages.length} stages</strong>
                    </div>
                    {#if pipelineLoading}
                        <div class="widget-loading"><WabiSpinner size="sm" tempo="calm" /></div>
                    {:else if pipelineError}
                        <div class="widget-error">Failed to load pipeline stages.</div>
                    {:else if pipelineStages.length === 0}
                        <div class="widget-empty">No open pipeline stages yet.</div>
                    {:else}
                        <div class="stage-list">
                            {#each pipelineStages as stage}
                                <button
                                    class="stage-row"
                                    type="button"
                                    data-tone={pipelineStageTone(stage.stage)}
                                    onclick={() => handlePipelineStageClick(stage.stage)}
                                    aria-label={`Open ${stage.stage} opportunities`}
                                >
                                    <div class="stage-row-head">
                                        <strong>{stage.stage}</strong>
                                        <span>{formatCount(stage.count)} &middot; {formatCompactCurrency(stage.value)}</span>
                                    </div>
                                    <div class="stage-bar-track">
                                        <span class="stage-bar-fill" style={`width: ${pipelineStageWidth(stage)}%`}></span>
                                    </div>
                                </button>
                            {/each}
                        </div>
                    {/if}
                </article>

                {#if canViewFinance}
                    <article class="panel receivables-panel">
                        <div class="panel-head">
                            <span>Collections</span>
                            <strong>{stats.pendingInvoices} open</strong>
                        </div>
                        <div class="receivable-row primary">
                            <div>
                                <span>{t("finance.invoice.total")}</span>
                                <strong>{formatCurrency(stats.outstandingAR)}</strong>
                            </div>
                            <small>{stats.arDaysOverdue > 0 ? `${stats.arDaysOverdue}d average overdue` : "current"}</small>
                        </div>
                        {#if agingLoading}
                            <div class="widget-loading"><WabiSpinner size="sm" tempo="calm" /></div>
                        {:else if agingError}
                            <div class="widget-error">Failed to load AR aging.</div>
                        {:else if !arAging || arAging.total <= 0}
                            <div class="widget-empty">No aged receivables.</div>
                        {:else}
                            <div class="aging-buckets">
                                {#each agingBuckets as bucket}
                                    <button
                                        class="aging-bucket"
                                        type="button"
                                        data-tone={bucket.tone}
                                        onclick={() => handleAgingBucketClick(bucket)}
                                        aria-label={`Open ${bucket.label} receivables`}
                                    >
                                        <span class="aging-bucket-label">{bucket.label}</span>
                                        <div class="aging-bucket-track">
                                            <span class="aging-bucket-fill" style={`width: ${bucket.width}%`}></span>
                                        </div>
                                        <strong class="aging-bucket-value">{formatCompactCurrency(bucket.amount)}</strong>
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </article>
                {/if}

                <article class="panel activity-panel" class:wide={!canViewFinance}>
                    <div class="panel-head">
                        <span>Tasks</span>
                        <button class="panel-action" type="button" onclick={openTaskModal}>New Task</button>
                    </div>
                    {#if activeDashboardTasks.length > 0}
                        <div class="activity-list">
                            {#each activeDashboardTasks as task}
                                <div
                                    class="activity-row"
                                    role="button"
                                    tabindex="0"
                                    onclick={() => openTask(task)}
                                    onkeydown={(event) => {
                                        if (event.key === "Enter" || event.key === " ") {
                                            event.preventDefault();
                                            openTask(task);
                                        }
                                    }}
                                    aria-label={`Open task ${task.title}`}
                                >
                                    <span class="activity-token">{taskInitial(task)}</span>
                                    <div>
                                        <strong>{task.title}</strong>
                                        <span>{summarizeText(task.description || taskMeta(task), 110)}</span>
                                    </div>
                                    <time>{taskAge(task)}</time>
                                </div>
                            {/each}
                        </div>
                    {:else}
                        <button class="empty-state empty-action" type="button" onclick={openTaskModal}>No active tasks. Create one.</button>
                    {/if}
                </article>
            </section>
        </main>
    {/if}
</div>

<ContextTaskModal
    open={taskModalOpen}
    title="Create Task"
    subtitle="Add a follow-up from the dashboard"
    on:created={handleTaskCreated}
    on:close={() => (taskModalOpen = false)}
/>

<style>
    .dashboard-page {
        min-height: 100vh;
        padding: 22px 28px 28px;
        background: var(--bg-base);
        color: var(--text-primary);
        box-sizing: border-box;
    }

    .dashboard-header {
        display: flex;
        justify-content: space-between;
        gap: 24px;
        align-items: flex-start;
        margin-bottom: 18px;
    }

    .brand-kicker {
        margin: 0 0 12px;
        color: var(--info);
        font-size: 12px;
        font-weight: 700;
    }

    .dashboard-header h1 {
        margin: 0;
        color: var(--text-primary);
        font-size: 28px;
        font-weight: 500;
        line-height: 1.1;
    }

    .dashboard-subtitle {
        margin: 6px 0 0;
        color: var(--text-secondary);
        font-size: 12px;
        font-weight: 600;
    }

    .dashboard-header time {
        color: var(--text-muted);
        font-size: 12px;
        white-space: nowrap;
    }

    .loading-state {
        min-height: 60vh;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .dashboard-shell {
        display: flex;
        flex-direction: column;
        gap: 14px;
    }

    .kpi-strip {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 14px;
    }

    .metric-card,
    .panel {
        background: rgba(255, 255, 255, 0.78);
        border: 1px solid rgba(205, 216, 226, 0.82);
        border-radius: 8px;
        box-shadow: 0 10px 28px rgba(39, 54, 73, 0.035);
    }

    .metric-card {
        min-height: 86px;
        padding: 18px 20px 14px;
        position: relative;
        overflow: hidden;
        cursor: pointer;
        transition:
            transform 0.15s ease,
            box-shadow 0.15s ease,
            border-color 0.15s ease;
    }

    .metric-card:hover {
        transform: translateY(-1px);
        box-shadow: 0 14px 32px rgba(39, 54, 73, 0.08);
        border-color: rgba(150, 170, 195, 0.9);
    }

    .metric-card:focus-visible {
        outline: 2px solid var(--focus-glow, #79a9df);
        outline-offset: 2px;
    }

    .metric-card::before {
        content: "";
        position: absolute;
        inset: 0 0 auto 0;
        height: 3px;
        background: var(--border);
    }

    .metric-card[data-tone="positive"]::before {
        background: var(--success);
    }

    .metric-card[data-tone="negative"]::before {
        background: var(--danger);
    }

    .metric-card[data-tone="warning"]::before {
        background: var(--warning);
    }

    .metric-label,
    .panel-head span {
        color: var(--text-secondary);
        font-size: 10px;
        font-weight: 800;
        letter-spacing: 0;
        text-transform: uppercase;
    }

    .metric-value {
        margin-top: 10px;
        color: var(--text-primary);
        font-size: 24px;
        line-height: 1;
        font-weight: 700;
    }

    .metric-meta {
        display: flex;
        justify-content: space-between;
        gap: 10px;
        margin-top: 8px;
        color: var(--text-muted);
        font-size: 11px;
        line-height: 1.3;
    }

    .metric-meta strong {
        color: var(--success);
        font-weight: 700;
        text-align: right;
    }

    .metric-card[data-tone="negative"] .metric-meta strong {
        color: var(--danger);
    }

    .metric-card[data-tone="warning"] .metric-meta strong {
        color: var(--warning);
    }

    .decision-grid {
        display: grid;
        grid-template-columns: minmax(0, 1.15fr) minmax(320px, 0.85fr);
        gap: 14px;
    }

    .panel {
        padding: 20px;
        min-height: 210px;
        box-sizing: border-box;
    }

    .panel-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
        margin-bottom: 12px;
    }

    .panel-head strong {
        color: var(--text-muted);
        font-size: 11px;
        font-weight: 700;
    }

    .panel-action {
        padding: 6px 9px;
        border: 1px solid var(--border);
        border-radius: 5px;
        background: var(--surface);
        color: var(--info);
        font-size: 11px;
        font-weight: 800;
        cursor: pointer;
    }

    .panel-action:hover {
        border-color: var(--info-hover, #8fb1d5);
        background: var(--surface-elevated);
    }

    .operating-panel h2 {
        margin: 0 0 18px;
        color: var(--text-primary);
        font-size: 18px;
        line-height: 1.35;
        font-weight: 700;
    }

    .focus-list,
    .alert-list,
    .activity-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .focus-row,
    .alert-row,
    .activity-row,
    .receivable-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        border-radius: 6px;
        background: var(--surface-elevated);
    }

    .focus-row {
        padding: 12px 14px;
        border-left: 3px solid var(--border);
    }

    .focus-row[data-tone="critical"] {
        background: var(--tint-danger, #fff0f1);
        border-color: var(--danger);
    }

    .focus-row[data-tone="warning"] {
        background: var(--tint-warning, #fff7e9);
        border-color: var(--warning);
    }

    .focus-row[data-tone="info"] {
        background: var(--tint-info, #edf5ff);
        border-color: var(--info);
    }

    .focus-row strong,
    .activity-row strong,
    .receivable-row strong {
        display: block;
        color: var(--text-primary);
        font-size: 13px;
        font-weight: 800;
    }

    .focus-row span,
    .activity-row div span,
    .receivable-row span,
    .alert-row span {
        display: block;
        margin-top: 4px;
        color: var(--text-muted);
        font-size: 11px;
        line-height: 1.35;
    }

    .focus-action {
        min-width: 58px;
        padding: 7px 10px;
        border: 0;
        border-radius: 5px;
        background: var(--info);
        color: white;
        font-size: 11px;
        font-weight: 800;
        text-align: center;
        cursor: pointer;
        transition:
            background 0.15s ease,
            transform 0.15s ease,
            box-shadow 0.15s ease;
    }

    .focus-action:hover {
        background: var(--info-pressed, #1a5298);
        box-shadow: 0 6px 14px rgba(34, 101, 185, 0.16);
        transform: translateY(-1px);
    }

    .focus-action:focus-visible {
        outline: 2px solid var(--focus-glow, #79a9df);
        outline-offset: 2px;
    }

    .alert-row {
        align-items: flex-start;
        justify-content: flex-start;
        padding: 12px 14px;
        border-left: 3px solid var(--border);
    }

    .alert-row[data-tone="critical"] {
        background: var(--tint-danger, #fff0f2);
        border-color: var(--danger);
    }

    .alert-row[data-tone="warning"] {
        background: var(--tint-warning, #fff7e8);
        border-color: var(--warning);
    }

    .alert-row[data-tone="info"] {
        background: var(--tint-info, #eef6ff);
        border-color: var(--info);
    }

    .alert-row[data-tone="soft"] {
        background: var(--surface-elevated);
        border-color: var(--border);
    }

    .alert-row strong {
        min-width: 96px;
        color: var(--text-primary);
        font-size: 12px;
    }

    .receivables-panel,
    .pipeline-panel,
    .activity-panel {
        min-height: 260px;
    }

    .activity-panel.wide {
        grid-column: 1 / -1;
        min-height: 220px;
    }

    .receivable-row {
        padding: 14px;
    }

    .receivable-row.primary {
        background: var(--tint-danger, #fff7f8);
    }

    .receivable-row small {
        color: var(--danger);
        font-size: 11px;
        font-weight: 800;
        white-space: nowrap;
    }

    /* ===== Pipeline-by-stage + AR-aging widgets (Wave 9.2 C2) ===== */
    .widget-loading,
    .widget-empty {
        min-height: 90px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--text-muted);
        font-size: 12px;
    }

    .widget-error {
        min-height: 90px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--danger);
        font-size: 12px;
        text-align: center;
    }

    .stage-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .stage-row {
        display: block;
        width: 100%;
        padding: 8px 0;
        border: 0;
        background: transparent;
        text-align: left;
        cursor: pointer;
        border-radius: 4px;
        transition: background 0.15s ease;
    }

    .stage-row:hover {
        background: var(--surface-elevated);
    }

    .stage-row:focus-visible {
        outline: 2px solid var(--focus-glow, #79a9df);
        outline-offset: 2px;
    }

    .stage-row-head {
        display: flex;
        justify-content: space-between;
        align-items: baseline;
        gap: 10px;
        margin-bottom: 6px;
    }

    .stage-row-head strong {
        color: var(--text-primary);
        font-size: 13px;
        font-weight: 700;
    }

    .stage-row-head span {
        color: var(--text-muted);
        font-size: 11px;
        white-space: nowrap;
    }

    .stage-bar-track {
        height: 6px;
        border-radius: 999px;
        background: var(--border);
        overflow: hidden;
    }

    .stage-bar-fill {
        display: block;
        height: 100%;
        border-radius: inherit;
        background: var(--info);
    }

    .stage-row[data-tone="positive"] .stage-bar-fill {
        background: var(--success);
    }

    .stage-row[data-tone="negative"] .stage-bar-fill {
        background: var(--danger);
    }

    .aging-buckets {
        margin-top: 16px;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .aging-bucket {
        display: grid;
        grid-template-columns: 52px 1fr auto;
        align-items: center;
        gap: 10px;
        width: 100%;
        padding: 6px 0;
        border: 0;
        background: transparent;
        cursor: pointer;
        border-radius: 4px;
        transition: background 0.15s ease;
    }

    .aging-bucket:hover {
        background: var(--surface-elevated);
    }

    .aging-bucket:focus-visible {
        outline: 2px solid var(--focus-glow, #79a9df);
        outline-offset: 2px;
    }

    .aging-bucket-label {
        color: var(--text-secondary);
        font-size: 11px;
        font-weight: 700;
    }

    .aging-bucket-track {
        height: 6px;
        border-radius: 999px;
        background: var(--border);
        overflow: hidden;
    }

    .aging-bucket-fill {
        display: block;
        height: 100%;
        border-radius: inherit;
        background: var(--info);
    }

    .aging-bucket[data-tone="positive"] .aging-bucket-fill {
        background: var(--success);
    }

    .aging-bucket[data-tone="warning"] .aging-bucket-fill {
        background: var(--warning);
    }

    .aging-bucket[data-tone="negative"] .aging-bucket-fill {
        background: var(--danger);
    }

    .aging-bucket-value {
        color: var(--text-primary);
        font-size: 12px;
        white-space: nowrap;
    }

    .activity-row {
        padding: 10px 0;
        background: transparent;
        border-bottom: 1px solid var(--border);
        cursor: pointer;
        border-radius: 4px;
        transition: background 0.15s ease;
    }

    .activity-row:last-child {
        border-bottom: none;
    }

    .activity-row:hover {
        background: var(--surface-elevated);
    }

    .activity-row:focus-visible {
        outline: 2px solid var(--focus-glow, #79a9df);
        outline-offset: -2px;
    }

    .activity-token {
        width: 22px;
        height: 22px;
        min-width: 22px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        box-sizing: border-box;
        border-radius: 5px;
        background: var(--surface-elevated);
        color: var(--info);
        font-size: 10px;
        font-weight: 800;
        line-height: 1;
        text-align: center;
        flex: 0 0 22px;
    }

    .activity-row div {
        flex: 1;
        min-width: 0;
    }

    .activity-row time {
        color: var(--text-muted);
        font-size: 10px;
        font-weight: 700;
        white-space: nowrap;
    }

    .empty-state {
        min-height: 160px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--text-muted);
        font-size: 12px;
    }

    .empty-action {
        width: 100%;
        border: 1px dashed var(--border);
        border-radius: 6px;
        background: var(--surface-elevated);
        cursor: pointer;
    }

    .empty-action:hover {
        color: var(--info);
        border-color: var(--info-hover, #9bb9d8);
        background: var(--surface-elevated);
    }

    @media (max-width: 1180px) {
        .kpi-strip,
        .decision-grid {
            grid-template-columns: 1fr 1fr;
        }
    }

    @media (max-width: 760px) {
        .dashboard-page {
            padding: 18px;
        }

        .dashboard-header {
            flex-direction: column;
        }

        .kpi-strip,
        .decision-grid {
            grid-template-columns: 1fr;
        }

        .metric-meta,
        .focus-row,
        .receivable-row {
            align-items: flex-start;
            flex-direction: column;
        }

        .aging-bucket {
            grid-template-columns: 1fr auto;
            grid-template-areas: "label value" "track track";
            row-gap: 4px;
        }

        .aging-bucket-label {
            grid-area: label;
        }

        .aging-bucket-value {
            grid-area: value;
        }

        .aging-bucket-track {
            grid-area: track;
        }
    }
</style>
