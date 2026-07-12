<script lang="ts">
    import { run, self, preventDefault } from 'svelte/legacy';

    import { createEventDispatcher, onMount, tick } from "svelte";
    import OpportunityDetail from "../components/OpportunityDetail.svelte";
    import ContextTaskModal from "../components/ContextTaskModal.svelte";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import Button from "../components/ui/Button.svelte";

    import {
        GetRFQs } from "../../../wailsjs/go/main/App";
import { GetPipelineOpportunities, CreateRFQWithReference, DeleteRFQ, DeleteRFQWithCascade, ListCustomers } from "../../../wailsjs/go/main/CRMService";
    import { toast } from "../stores/toasts";
    import { confirm } from "../stores/confirm";
    import { devLog } from "$lib/utils/devLog";
    import { debounce } from "$lib/utils/debounce";
    import { pendingProjectHandoff } from "$lib/stores/navigation";

    const dispatch = createEventDispatcher();

    // Wave 9 B1.4 / B3: optional stage pre-filter from a dashboard drill-through.
    // NOTE: SalesHub.svelte (the parent) does not currently forward its own
    // `params` prop down to this component, so this only takes effect once
    // that one-line wiring is added there (out of scope for this change).
    interface Props {
        params?: { stage?: string };
    }
    let { params = {} }: Props = $props();

    let opportunities: any[] = [];
    let filteredOpportunities: any[] = $state([]);
    let selected: any = $state(null);
    let loadingList = $state(true);
    let showCreateForm = $state(false);
    let creating = $state(false);
    let customers: any[] = $state([]);

    let searchQuery = $state("");
    let debouncedSearchQuery = "";
    let activeFilter = $state("All");
    let activeYear = $state(String(new Date().getFullYear()));
    let sortBy = $state("date");
    let sortDirection = $state("desc");
    let availableYears: string[] = $state(["All"]);

    let showOpportunityModal = $state(false);
    let showDeleteConfirm = $state(false);
    let deleteTarget: any = $state(null);
    let deleting = $state(false);
    let showTaskModal = $state(false);
    let opportunityModalEl: HTMLDivElement | null = $state(null);

    let pipelineStats = $state({
        newCount: 0,
        quotedCount: 0,
        wonCount: 0,
        totalValue: 0,
        winRate: 0,
    });

    let createForm = $state({
        customer: "",
        project: "",
        rfq_ref: "",
        value: "",
        notes: "",
    });

    // Tabs match the values displayStage() actually renders on cards (B2a fix) -
    // "Pipeline" covers the collapsed deprecated-stage cluster (New/Qualified/
    // Proposal/On Hold) so every tab is guaranteed reachable by real data.
    const STATUS_OPTIONS = ["All", "Pipeline", "Quoted", "Won", "Lost"];
    const DEPRECATED_STAGE_LABELS = new Set(["New", "Qualified", "Proposal", "On Hold"]);

    // DISPLAY-ONLY safety net (Wave 9.6 B1, safe half): legacy 9-stage / ad-hoc
    // RFQ stage strings still present in stored data get bucketed into the
    // canonical tab set below so no card ever renders a raw legacy string and
    // every card is reachable from a real tab (not just "All"). This is purely
    // a label/tab mapping for the UI - it does NOT touch the stored `stage`
    // value, any write path, or the Go dashboard win-rate/pipeline-value
    // computation (app_prediction_dashboard.go), which still reads the raw
    // stored strings untouched. The backend stage-vocabulary consolidation
    // (collapsing these into a single canonical enum at the data layer) is
    // escalated to the owner as a stop-and-report migration, since that value
    // feeds a financial computation.
    const LEGACY_STAGE_DISPLAY_MAP: Record<string, string> = {
        "RFQ Received": "Pipeline",
        "Costing": "Pipeline",
        "Tender": "Pipeline",
        "Offer Sent": "Quoted",
        "Follow-up/Eval": "Quoted",
        "PO/LOI Received": "Won",
        "Order Placed": "Won",
        "In Process": "Won",
        "Delivered": "Won",
        "Closed (Payment)": "Won",
        "Closed (Lost)": "Lost",
    };

    const updateDebouncedSearch = debounce((value: string) => {
        debouncedSearchQuery = value;
        applyFilters();
    }, 300);

    run(() => {
        updateDebouncedSearch(searchQuery);
    });

    function parseYear(value: any): number {
        const numericYear = Number(value);
        if (Number.isFinite(numericYear) && numericYear >= 2000 && numericYear <= 2100) {
            return numericYear;
        }

        if (value) {
            const date = new Date(value);
            if (!Number.isNaN(date.getTime())) {
                return date.getFullYear();
            }
        }

        return new Date().getFullYear();
    }

    function getOpportunityYear(opp: any): string {
        return String(parseYear(opp?.year ?? opp?.created_at));
    }

    function buildOpportunityTitle(customerName: string, reference: string, fallback: string) {
        const customer = (customerName || "").trim();
        const ref = (reference || "").trim();
        if (customer && ref) return `${customer} / ${ref}`;
        if (customer) return customer;
        if (ref) return ref;
        return fallback || "Untitled";
    }

    function normalizeStage(value: any): string {
        const stage = String(value || "").trim();
        if (!stage) return "New";
        return stage;
    }

    function displayStage(value: any): string {
        const stage = normalizeStage(value);
        if (DEPRECATED_STAGE_LABELS.has(stage)) return "Pipeline";
        return LEGACY_STAGE_DISPLAY_MAP[stage] ?? stage;
    }

    function getOpportunityTimestamp(opp: any): number {
        const candidates = [
            opp?.updated_at,
            opp?.created_at,
            opp?.offer_date,
            opp?.received_date,
            opp?.expected_date,
        ];

        for (const candidate of candidates) {
            if (!candidate) continue;
            const parsed = new Date(candidate).getTime();
            if (!Number.isNaN(parsed) && parsed > 0) {
                return parsed;
            }
        }

        return 0;
    }

    function compareOpportunityRecency(a: any, b: any): number {
        const timestampDiff = getOpportunityTimestamp(b) - getOpportunityTimestamp(a);
        if (timestampDiff !== 0) return timestampDiff;

        const folderA = String(a?.folder_number || a?.rfq_number || "").trim();
        const folderB = String(b?.folder_number || b?.rfq_number || "").trim();
        if (folderA || folderB) {
            return folderB.localeCompare(folderA, undefined, { numeric: true, sensitivity: "base" });
        }

        return String(b?.id || "").localeCompare(String(a?.id || ""), undefined, { numeric: true, sensitivity: "base" });
    }

    function quotedSortBucket(opp: any): number {
        const stage = normalizeStage(opp?.stage || opp?.status);
        return ["Quoted", "Won", "Lost"].includes(stage) ? 1 : 0;
    }

    function normalizePipelineOpportunity(opp: any) {
        const visibleReference = opp.eh_ref || opp.folder_number || "";
        const stage = normalizeStage(opp.stage);
        return {
            ...opp,
            client: opp.customer_name,
            customer: opp.customer_name,
            project: buildOpportunityTitle(opp.customer_name, visibleReference, opp.title || opp.folder_name || opp.folder_number),
            project_name: buildOpportunityTitle(opp.customer_name, visibleReference, opp.title || opp.folder_name || opp.folder_number),
            value: Number(opp.revenue_bhd) || 0,
            status: stage,
            stage,
            rfq_number: visibleReference,
            rfq_ref: opp.eh_ref || "",
            year: parseYear(opp.year),
            _source: "pipeline",
        };
    }

    function normalizeRFQ(rfq: any) {
        const reference = rfq.rfq_ref || rfq.rfq_number || "";
        const fallbackTitle = rfq.project || rfq.project_name || "Untitled";
        const stage = normalizeStage(rfq.stage || rfq.status);
        return {
            ...rfq,
            client: rfq.client || rfq.customer || "Unknown",
            customer: rfq.client || rfq.customer || "Unknown",
            project: buildOpportunityTitle(rfq.client || rfq.customer || "", reference, fallbackTitle),
            project_name: buildOpportunityTitle(rfq.client || rfq.customer || "", reference, fallbackTitle),
            value: Number(rfq.value) || 0,
            status: stage,
            stage,
            year: parseYear(rfq.created_at),
            _source: "rfq",
        };
    }

    function refreshAvailableYears() {
        const years = [...new Set(opportunities.map(getOpportunityYear))]
            .filter(Boolean)
            .sort((a, b) => Number(b) - Number(a));
        availableYears = ["All", ...years];
        if (activeYear !== "All" && years.length > 0 && !years.includes(activeYear)) {
            activeYear = years[0];
        }
        if (years.length === 0) {
            activeYear = "All";
        }
    }

    async function loadOpportunities() {
        loadingList = true;
        try {
            const [rfqs, pipeline, custList] = await Promise.all([
                GetRFQs(200, 0).catch(() => []),
                GetPipelineOpportunities(500, 0).catch(() => []),
                ListCustomers(500, 0).catch(() => []),
            ]);

            customers = custList || [];

            const pipelineNormalized = (pipeline || []).map(normalizePipelineOpportunity);
            const liveRfqs = (rfqs || [])
                .filter((rfq: any) => String(rfq?.rfq_number || "").trim())
                .map(normalizeRFQ);

            const rfqFolders = new Set(liveRfqs.map((rfq: any) => String(rfq.rfq_number || "").trim()).filter(Boolean));
            const uniquePipeline = pipelineNormalized.filter((opp: any) => !rfqFolders.has(String(opp.folder_number || "").trim()));

            opportunities = [...uniquePipeline, ...liveRfqs].sort(compareOpportunityRecency);
            refreshAvailableYears();
            applyFilters();

            if (showOpportunityModal && selected) {
                const refreshed = opportunities.find((opp) => String(opp.id) === String(selected.id));
                selected = refreshed || selected;
            }
        } catch (err) {
            devLog.error(err);
            toast.danger("Failed to load opportunities");
        } finally {
            loadingList = false;
        }
    }

    function calculateStats(baseList: any[]) {
        const stats = {
            newCount: 0,
            quotedCount: 0,
            wonCount: 0,
            totalValue: 0,
        };
        let won = 0;
        let closed = 0;

        baseList.forEach((opp) => {
            const stage = normalizeStage(opp.stage || opp.status);
            const value = Number(opp.value) || Number(opp.revenue_bhd) || 0;

            if (DEPRECATED_STAGE_LABELS.has(stage)) stats.newCount++;
            if (stage === "Quoted") stats.quotedCount++;
            if (stage === "Won") {
                stats.wonCount++;
                won++;
                closed++;
            }
            if (stage === "Lost") {
                closed++;
            }

            stats.totalValue += value;
        });

        pipelineStats = {
            ...stats,
            winRate: closed > 0 ? (won / closed) * 100 : 0,
        };
    }

    function applyFilters() {
        let yearScoped = [...opportunities];

        if (activeYear !== "All") {
            yearScoped = yearScoped.filter((opp) => getOpportunityYear(opp) === activeYear);
        }

        calculateStats(yearScoped);

        let result = [...yearScoped];

        if (activeFilter !== "All") {
            // Filter on the SAME value the card shows (displayStage), not the raw
            // stage - otherwise a tab can return cards whose visible label disagrees
            // with the tab, or never match canonical records at all (B2a fix).
            result = result.filter((opp) => displayStage(opp.stage || opp.status) === activeFilter);
        }

        if (debouncedSearchQuery.trim()) {
            const query = debouncedSearchQuery.toLowerCase();
            result = result.filter((opp) =>
                (opp.client || opp.customer || "").toLowerCase().includes(query) ||
                (opp.project || opp.project_name || "").toLowerCase().includes(query) ||
                String(opp.rfq_number || opp.folder_number || "").toLowerCase().includes(query) ||
                String(opp.folder_name || "").toLowerCase().includes(query) ||
                String(opp.title || "").toLowerCase().includes(query) ||
                String(opp.stage || opp.status || "").toLowerCase().includes(query) ||
                String(opp.owner || opp.owner_name || "").toLowerCase().includes(query) ||
                String(opp.payment_terms || "").toLowerCase().includes(query) ||
                String(opp.delivery_terms || "").toLowerCase().includes(query) ||
                String(opp.notes || opp.comment || opp.owner_notes || "").toLowerCase().includes(query)
            );
        }

        result.sort((a, b) => {
            if (activeFilter === "All") {
                const quoteBucketDelta = quotedSortBucket(a) - quotedSortBucket(b);
                if (quoteBucketDelta !== 0) return quoteBucketDelta;
            }

            let aValue: any = 0;
            let bValue: any = 0;

            if (sortBy === "date") {
                aValue = getOpportunityTimestamp(a);
                bValue = getOpportunityTimestamp(b);
            } else if (sortBy === "value") {
                aValue = Number(a.value) || 0;
                bValue = Number(b.value) || 0;
            } else if (sortBy === "customer") {
                aValue = (a.client || a.customer || "").toLowerCase();
                bValue = (b.client || b.customer || "").toLowerCase();
            }

            if (aValue === bValue) {
                return compareOpportunityRecency(a, b);
            }

            if (sortDirection === "asc") {
                return aValue > bValue ? 1 : -1;
            }
            return aValue < bValue ? 1 : -1;
        });

        filteredOpportunities = result;
    }

    run(() => {
        activeFilter;
        activeYear;
        sortBy;
        sortDirection;
        applyFilters();
    });

    function formatCurrency(value: number) {
        return new Intl.NumberFormat("en-BH", {
            style: "currency",
            currency: "BHD",
            maximumFractionDigits: 0,
        }).format(Number(value) || 0);
    }

    function getOpportunityValue(opp: any): number {
        return Number(opp?.value ?? opp?.revenue_bhd ?? 0) || 0;
    }

    function formatOpportunityValue(opp: any): string {
        const value = getOpportunityValue(opp);
        if (value <= 0) return "Value pending";
        return formatCurrency(value);
    }

    function formatDate(value: any) {
        if (!value) return "No date";
        const parsed = new Date(value);
        if (Number.isNaN(parsed.getTime())) return "No date";
        return parsed.toLocaleDateString();
    }

    async function selectOpportunity(opp: any) {
        selected = opp;
        showOpportunityModal = true;
        await tick();
        opportunityModalEl?.focus();
    }

    function closeOpportunityModal() {
        showTaskModal = false;
        showOpportunityModal = false;
    }

    function handleOpenCostingFromDetail(event: CustomEvent) {
        const opportunity = event.detail?.opportunity || selected;
        const pendingPayload = event.detail?.pendingPayload || {
            id: String(opportunity?.id || ""),
            customer_name: opportunity?.client || opportunity?.customer || "",
            folder_number: opportunity?.folder_number || opportunity?.rfq_number || "",
            rfq_ref: opportunity?.rfq_ref || opportunity?.eh_ref || "",
            title: opportunity?.project || opportunity?.project_name || opportunity?.title || "",
        };
        if (pendingPayload?.id) {
            sessionStorage.setItem("asymmflow.pendingCostingOpportunity", JSON.stringify(pendingPayload));
        }
        showTaskModal = false;
        showOpportunityModal = false;
        dispatch("navigate", { screen: "opportunities", tab: "costing" });
        window.dispatchEvent(new CustomEvent("navigateToScreen", {
            detail: { screen: "opportunities", tab: "costing" }
        }));
    }

    // Wave 9.4 B4.1: "Start project" handoff — a won/live opportunity becomes
    // a WorkHub project in one action, with lineage (opportunity id, customer)
    // preseeded into the create composer. Mirrors the pendingDNCreate pattern
    // OrdersScreen already uses for delivery notes.
    function handleStartProject(opportunity: any) {
        if (!opportunity) return;
        const customerName = opportunity.customer_name || opportunity.client || opportunity.customer || "";
        const projectLabel = opportunity.project || opportunity.project_name || opportunity.folder_name || opportunity.folder_number || "Opportunity";
        pendingProjectHandoff.request({
            source: "opportunity",
            sourceId: String(opportunity.id || ""),
            opportunityId: String(opportunity.id || ""),
            customerId: opportunity.customer_id || undefined,
            customerName,
            suggestedName: customerName ? `${customerName} — ${projectLabel}` : projectLabel,
        });
        showOpportunityModal = false;
        window.dispatchEvent(new CustomEvent("navigateToScreen", {
            detail: { screen: "work" }
        }));
    }

    async function handleCreate() {
        if (!createForm.customer || !createForm.project) {
            toast.warning("Customer and Prospect/Project fields are required");
            return;
        }

        creating = true;
        try {
            await CreateRFQWithReference(
                createForm.customer,
                createForm.project,
                createForm.rfq_ref,
                parseFloat(createForm.value) || 0,
                createForm.notes,
            );
            toast.success("Opportunity created");
            showCreateForm = false;
            createForm = {
                customer: "",
                project: "",
                rfq_ref: "",
                value: "",
                notes: "",
            };
            await loadOpportunities();
        } catch (err) {
            toast.danger("Creation failed: " + err);
        } finally {
            creating = false;
        }
    }

    function confirmDelete(opp: any, event: MouseEvent) {
        event.stopPropagation();
        deleteTarget = opp;
        showDeleteConfirm = true;
    }

    async function handleDelete(cascade = false) {
        if (!deleteTarget) return;

        deleting = true;
        try {
            if (cascade) {
                await DeleteRFQWithCascade(deleteTarget.id, true);
            } else {
                await DeleteRFQ(deleteTarget.id);
            }
            toast.success(`Opportunity "${deleteTarget.project || "Untitled"}" deleted`);
            if (selected?.id === deleteTarget.id) {
                selected = null;
                showOpportunityModal = false;
            }
            showDeleteConfirm = false;
            deleteTarget = null;
            await loadOpportunities();
        } catch (err: any) {
            const errorMsg = err?.message || String(err);
            if (errorMsg.includes("costing sheet") || errorMsg.includes("offer")) {
                toast.warning("This opportunity has linked documents. Use 'Delete All' to remove everything.");
            } else {
                toast.danger("Delete failed: " + errorMsg);
            }
        } finally {
            deleting = false;
        }
    }

    // Article III.2 mass-destructive rung: destroying N things (this opportunity
    // PLUS every linked costing sheet and offer) must be visibly harder than
    // destroying the one opportunity alone. The plain "Delete" button keeps its
    // single confirm (the modal itself); "Delete All" routes through a second,
    // escalated, typed-reason gate via the canonical confirm store before the
    // cascade actually runs.
    async function handleCascadeDelete() {
        if (!deleteTarget) return;

        const projectLabel = deleteTarget.project || "this opportunity";
        const r = await confirm.askForReason({
            title: "Delete Everything Linked to This Opportunity?",
            message: `This permanently deletes "${projectLabel}" AND every costing sheet and offer linked to it. This cannot be undone. Type a reason to confirm you understand the full scope of what is being destroyed.`,
            confirmLabel: "Delete All, Permanently",
            variant: "danger",
            reasonLabel: "Reason for full cascade delete",
            reasonPlaceholder: "e.g. Duplicate opportunity, created in error...",
            reasonRequired: true,
        });
        if (!r.confirmed) return;

        await handleDelete(true);
    }

    onMount(() => {
        loadOpportunities().then(() => {
            // B3 360-continuity: CustomerDetailView's RFQ row drills straight into
            // the matching opportunity, mirroring CostingSheetScreen's pending-store handoff.
            const pendingOpportunityId = sessionStorage.getItem("asymmflow.pendingOpportunityId");
            if (pendingOpportunityId) {
                try {
                    const match = opportunities.find((opp) => String(opp.id) === pendingOpportunityId);
                    if (match) {
                        selectOpportunity(match);
                    } else {
                        toast.warning("Could not find that opportunity in this view.");
                    }
                } finally {
                    sessionStorage.removeItem("asymmflow.pendingOpportunityId");
                }
            }
        });

        // B3: "New RFQ for this customer" from CustomerDetailView preseeds the create form.
        const pendingRFQCustomer = sessionStorage.getItem("asymmflow.pendingRFQCustomer");
        if (pendingRFQCustomer) {
            try {
                const pending = JSON.parse(pendingRFQCustomer);
                createForm = { ...createForm, customer: pending.name || "" };
                showCreateForm = true;
            } finally {
                sessionStorage.removeItem("asymmflow.pendingRFQCustomer");
            }
        }
    });

    // B1.4: apply an incoming stage pre-filter (see the Props note above for the
    // SalesHub wiring this currently depends on).
    let lastAppliedStage = "";
    run(() => {
        const requestedStage = params?.stage;
        if (requestedStage && requestedStage !== lastAppliedStage) {
            if (STATUS_OPTIONS.includes(requestedStage)) {
                activeFilter = requestedStage;
            }
            lastAppliedStage = requestedStage;
        }
    });
</script>

<div class="screen-container">
    <div class="action-bar">
        <div class="kpi-row">
            <div class="mini-kpi">
                <span class="kpi-label">Pipeline</span>
                <span class="kpi-value">{formatCurrency(pipelineStats.totalValue)}</span>
            </div>
            <div class="mini-kpi">
                <span class="kpi-label">Win Rate</span>
                <span class="kpi-value">{pipelineStats.winRate.toFixed(1)}%</span>
            </div>
            <div class="mini-kpi">
                <span class="kpi-label">Open</span>
                <span class="kpi-value">{pipelineStats.newCount + pipelineStats.quotedCount}</span>
            </div>
        </div>
        <Button variant="primary" on:click={() => (showCreateForm = true)}>+ New Opportunity</Button>
    </div>

    <div class="controls-bar">
        <div class="filter-tabs">
            {#each STATUS_OPTIONS as status}
                <button
                    class="filter-tab"
                    class:active={activeFilter === status}
                    onclick={() => (activeFilter = status)}
                >
                    {status}
                </button>
            {/each}
        </div>

        <div class="right-controls">
            <select bind:value={activeYear} class="year-select" aria-label="Filter opportunities by year">
                {#each availableYears as yearOption}
                    <option value={yearOption}>{yearOption === "All" ? "All Years" : yearOption}</option>
                {/each}
            </select>

            <div class="sort-controls">
                <select bind:value={sortBy} class="sort-select">
                    <option value="date">Latest First</option>
                    <option value="value">Value</option>
                    <option value="customer">Customer</option>
                </select>
                <button
                    class="sort-direction-btn"
                    onclick={() => (sortDirection = sortDirection === "asc" ? "desc" : "asc")}
                    title={sortDirection === "asc" ? "Ascending" : "Descending"}
                >
                    {sortDirection === "asc" ? "(asc)" : "(desc)"}
                </button>
            </div>

            <div class="search-box">
                <input
                    type="text"
                    class="search-input"
                    placeholder="Search customer, project..."
                    bind:value={searchQuery}
                    aria-label="Search opportunities"
                />
            </div>
        </div>
    </div>

    <div class="list-shell">
        <div class="list-header">
            <span class="result-count">
                {filteredOpportunities.length} opportunities
                {#if activeYear !== "All"}
                    in {activeYear}
                {/if}
            </span>
        </div>

        {#if loadingList}
            <div class="loading-state"><WabiSpinner size="md" tempo="calm" /></div>
        {:else if filteredOpportunities.length === 0}
            <div class="empty-state">
                {searchQuery || activeFilter !== "All" ? "No matching opportunities found." : "No opportunities available."}
            </div>
        {:else}
            <div class="opportunity-list">
                {#each filteredOpportunities as opp}
                    <div
                        class="opportunity-card"
                        role="button"
                        tabindex="0"
                        onclick={() => selectOpportunity(opp)}
                        onkeydown={(event) => (event.key === "Enter" || event.key === " ") && selectOpportunity(opp)}
                    >
                        <div class="card-main">
                            <div class="card-identity">
                                <div class="eyebrow-row">
                                    <span class="opportunity-number">{opp.folder_number || opp.rfq_number || "Manual"}</span>
                                    <span class="year-pill">{getOpportunityYear(opp)}</span>
                                    <span class:source-pill={true} class:rfq={opp._source === "rfq"}>
                                        {opp._source === "rfq" ? "RFQ" : "Pipeline"}
                                    </span>
                                </div>
                                <h3>{opp.project || opp.project_name || "Untitled Opportunity"}</h3>
                                <p class="customer-name">{opp.client || opp.customer || "Unknown Customer"}</p>
                            </div>

                            <div class="card-summary">
                                <span class="status-badge">{displayStage(opp.stage || opp.status)}</span>
                                <span class="card-value" class:pending={getOpportunityValue(opp) <= 0}>
                                    {formatOpportunityValue(opp)}
                                </span>
                            </div>
                        </div>

                        <div class="card-foot">
                            <div class="meta-strip">
                                <span>Updated {formatDate(opp.updated_at || opp.created_at)}</span>
                                {#if opp.payment_terms}
                                    <span>{opp.payment_terms}</span>
                                {/if}
                                {#if opp.delivery_terms}
                                    <span>{opp.delivery_terms}</span>
                                {/if}
                            </div>

                            <div class="card-actions">
                                {#if opp._source === "rfq"}
                                    <button
                                        class="delete-btn"
                                        onclick={(event) => confirmDelete(opp, event)}
                                        aria-label={`Delete ${opp.project || "opportunity"}`}
                                    >
                                        Delete
                                    </button>
                                {/if}
                                <span class="view-link">View details</span>
                            </div>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

{#if showOpportunityModal && selected}
    <div
        bind:this={opportunityModalEl}
        class="modal-backdrop"
        onclick={self(closeOpportunityModal)}
        onkeydown={(event) => {
            if (event.currentTarget !== event.target) return;
            if (event.key === "Escape" || event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                closeOpportunityModal();
            }
        }}
        role="button"
        tabindex="0"
    >
        <div class="modal-card opportunity-modal" role="document">
            <div class="modal-header">
                <div>
                    <p class="modal-kicker">{selected.folder_number || selected.rfq_number || "Opportunity"}</p>
                    <h3 id="opportunity-modal-title">{selected.project || selected.project_name || "Opportunity Details"}</h3>
                </div>
                <div class="modal-header-actions">
                    <button class="ghost-btn" onclick={() => handleStartProject(selected)}>Start Project</button>
                    <button class="ghost-btn" onclick={() => showTaskModal = true}>Create Task</button>
                    <button class="close-btn" onclick={closeOpportunityModal} aria-label="Close opportunity details">Close</button>
                </div>
            </div>

            <OpportunityDetail
                opportunity={selected}
                on:updated={loadOpportunities}
                on:createCosting={handleOpenCostingFromDetail}
            />
        </div>
    </div>
{/if}

{#if selected}
    <ContextTaskModal
        open={showTaskModal}
        title="Create Opportunity Task"
        subtitle={`Link work to ${selected.project || selected.project_name || selected.folder_number || "this opportunity"}`}
        defaults={{
            customer_id: selected.customer_id,
            opportunity_id: String(selected.id || ""),
            seed_title: `Opportunity task: ${selected.project || selected.project_name || selected.folder_number || "Follow up"}`,
        }}
        on:close={() => showTaskModal = false}
        on:created={() => showTaskModal = false}
    />
{/if}

{#if showCreateForm}
    <div
        class="modal-backdrop"
        onclick={self(() => (showCreateForm = false))}
        onkeydown={(event) => {
            if (event.currentTarget !== event.target) return;
            if (event.key === "Escape" || event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                showCreateForm = false;
            }
        }}
        role="button"
        tabindex="0"
    >
        <div class="modal-card create-modal" role="document">
            <h3 id="create-modal-title" class="section-title">New Opportunity</h3>

            <form onsubmit={preventDefault(handleCreate)}>
                <div class="form-group">
                    <label class="label" for="create-opportunity-customer">Customer</label>
                    <input id="create-opportunity-customer" list="cust-list" bind:value={createForm.customer} class="input" placeholder="Search customer..." required />
                    <datalist id="cust-list">
                        {#each customers as customer}
                            <option value={customer.business_name}></option>
                        {/each}
                    </datalist>
                </div>

                <div class="form-group">
                    <label class="label" for="create-opportunity-project">Project</label>
                    <input id="create-opportunity-project" type="text" bind:value={createForm.project} class="input" placeholder="Project name" required />
                </div>

                <div class="form-group">
                    <label class="label" for="create-opportunity-reference">Reference ID</label>
                    <input id="create-opportunity-reference" type="text" bind:value={createForm.rfq_ref} class="input" placeholder="Customer RFQ / enquiry ref" />
                </div>

                <div class="form-grid">
                    <div class="form-group">
                        <label class="label" for="create-opportunity-value">Value (BHD)</label>
                        <input id="create-opportunity-value" type="number" bind:value={createForm.value} class="input" placeholder="0.00" />
                    </div>
                </div>

                <div class="form-group">
                    <label class="label" for="create-opportunity-notes">Notes</label>
                    <textarea
                        id="create-opportunity-notes"
                        bind:value={createForm.notes}
                        class="input notes-input"
                        rows="4"
                        placeholder="Add context, scope, customer request details, exclusions, or next steps..."
                    ></textarea>
                </div>

                <div class="modal-actions">
                    <Button variant="secondary" on:click={() => (showCreateForm = false)}>Cancel</Button>
                    <Button variant="primary" type="submit" disabled={creating}>
                        {creating ? "Creating..." : "Create"}
                    </Button>
                </div>
            </form>
        </div>
    </div>
{/if}

{#if showDeleteConfirm && deleteTarget}
    <div
        class="modal-backdrop"
        onclick={self(() => (showDeleteConfirm = false))}
        onkeydown={(event) => {
            if (event.currentTarget !== event.target) return;
            if (event.key === "Escape" || event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                showDeleteConfirm = false;
            }
        }}
        role="button"
        tabindex="0"
    >
        <div class="modal-card delete-modal" role="document">
            <div class="delete-header">
                <h3 id="delete-modal-title" class="section-title">Delete Opportunity?</h3>
            </div>

            <div class="delete-body">
                <p>Are you sure you want to delete this opportunity?</p>
                <div class="delete-details">
                    <div class="detail-row">
                        <span class="detail-label">Customer:</span>
                        <span class="detail-value">{deleteTarget.client || deleteTarget.customer || "Unknown"}</span>
                    </div>
                    <div class="detail-row">
                        <span class="detail-label">Project:</span>
                        <span class="detail-value">{deleteTarget.project || "Untitled"}</span>
                    </div>
                    <div class="detail-row">
                        <span class="detail-label">Value:</span>
                        <span class="detail-value">{formatOpportunityValue(deleteTarget)}</span>
                    </div>
                </div>
                <p class="warning-text">This action cannot be undone.</p>
            </div>

            <div class="modal-actions">
                <Button variant="secondary" on:click={() => (showDeleteConfirm = false)} disabled={deleting}>
                    Cancel
                </Button>
                <Button variant="danger" on:click={() => handleDelete(false)} disabled={deleting}>
                    {deleting ? "Deleting..." : "Delete"}
                </Button>
                <Button variant="danger" on:click={handleCascadeDelete} disabled={deleting}>
                    {deleting ? "..." : "Delete All (with linked docs)"}
                </Button>
            </div>
            <p class="delete-all-caution">
                "Delete All" also permanently removes every costing sheet and offer linked to this opportunity, and asks you to type a reason before it runs.
            </p>
        </div>
    </div>
{/if}

<style>
    .screen-container {
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 1rem;
        position: relative;
        overflow: hidden;
    }

    .action-bar,
    .controls-bar,
    .card-main,
    .card-foot,
    .modal-header,
    .modal-actions,
    .delete-header,
    .detail-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
    }

    .kpi-row,
    .filter-tabs,
    .right-controls,
    .sort-controls,
    .eyebrow-row,
    .meta-strip,
    .card-actions,
    .form-grid,
    .delete-details {
        display: flex;
        gap: 0.75rem;
        flex-wrap: wrap;
    }

    .mini-kpi {
        display: flex;
        flex-direction: column;
        min-width: 110px;
    }

    .kpi-label,
    .modal-kicker {
        font-size: 0.7rem;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--text-muted);
    }

    .kpi-value {
        font-size: 1.05rem;
        font-weight: 700;
        color: var(--text-primary);
    }

    .controls-bar {
        gap: 1rem;
        flex-wrap: wrap;
    }

    .filter-tab,
    .sort-select,
    .year-select,
    .sort-direction-btn,
    .search-input,
    .input,
    .delete-btn,
    .close-btn {
        border: 1px solid var(--border);
        border-radius: 12px;
        background: var(--surface);
        color: var(--text-primary);
    }

    .filter-tab {
        padding: 0.55rem 0.9rem;
        cursor: pointer;
        font-size: 0.8rem;
    }

    .filter-tab.active {
        background: var(--brand-indigo);
        color: white;
        border-color: var(--brand-indigo);
    }

    .sort-select,
    .year-select,
    .search-input,
    .input {
        padding: 0.65rem 0.8rem;
        min-height: 42px;
    }

    .search-box {
        min-width: 280px;
    }

    .search-input {
        width: 100%;
    }

    .notes-input {
        width: 100%;
        min-height: 112px;
        resize: vertical;
        line-height: 1.5;
    }

    .sort-direction-btn,
    .delete-btn,
    .close-btn {
        padding: 0.65rem 0.85rem;
        cursor: pointer;
    }

    .list-shell {
        flex: 1;
        min-height: 0;
        overflow: auto;
        padding-right: 0.25rem;
    }

    .list-header {
        margin-bottom: 0.9rem;
    }

    .result-count {
        font-size: 0.85rem;
        color: var(--text-muted);
    }

    .opportunity-list {
        display: grid;
        grid-template-columns: minmax(0, 1fr);
        gap: 0.9rem;
    }

    .opportunity-card {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 1.15rem 1.25rem;
        border-radius: 20px;
        border: 1px solid rgba(15, 23, 42, 0.08);
        background:
            linear-gradient(135deg, rgba(255, 255, 255, 0.98), rgba(244, 247, 251, 0.96)),
            radial-gradient(circle at top right, rgba(56, 189, 248, 0.09), transparent 35%);
        box-shadow: 0 16px 38px rgba(15, 23, 42, 0.08);
        cursor: pointer;
        transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
    }

    .opportunity-card:hover {
        transform: translateY(-1px);
        box-shadow: 0 22px 44px rgba(15, 23, 42, 0.12);
        border-color: rgba(37, 99, 235, 0.22);
    }

    .card-identity {
        flex: 1;
        min-width: 0;
    }

    .card-identity h3,
    .modal-header h3 {
        margin: 0.35rem 0 0.2rem;
        font-size: 1.2rem;
        line-height: 1.2;
    }

    .customer-name {
        margin: 0;
        color: var(--text-secondary);
    }

    .opportunity-number,
    .year-pill,
    .source-pill,
    .status-badge,
    .view-link {
        display: inline-flex;
        align-items: center;
        border-radius: 999px;
        padding: 0.32rem 0.65rem;
        font-size: 0.72rem;
        letter-spacing: 0.04em;
        text-transform: uppercase;
    }

    .opportunity-number {
        background: rgba(15, 23, 42, 0.08);
        color: var(--text-primary);
        font-weight: 700;
    }

    .year-pill {
        background: rgba(14, 165, 233, 0.12);
        color: #0369a1;
    }

    .source-pill {
        background: rgba(16, 185, 129, 0.12);
        color: #047857;
    }

    .source-pill.rfq {
        background: rgba(168, 85, 247, 0.12);
        color: #7c3aed;
    }

    .card-summary {
        min-width: 170px;
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        gap: 0.5rem;
    }

    .status-badge {
        background: rgba(15, 23, 42, 0.08);
        color: var(--text-primary);
        font-weight: 700;
    }

    .card-value {
        font-size: 1.15rem;
        font-weight: 800;
        color: var(--text-primary);
    }

    .card-value.pending {
        color: var(--text-muted);
        font-weight: 600;
    }

    .meta-strip {
        flex: 1;
        color: var(--text-muted);
        font-size: 0.82rem;
    }

    .view-link {
        background: rgba(37, 99, 235, 0.1);
        color: #1d4ed8;
        font-weight: 700;
    }

    .modal-backdrop {
        position: absolute;
        inset: 0;
        background: rgba(15, 23, 42, 0.45);
        backdrop-filter: blur(6px);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 1rem;
        z-index: 50;
    }

    .modal-card {
        width: min(820px, calc(100% - 2rem));
        max-height: calc(100% - 2rem);
        overflow: auto;
        background: white;
        border-radius: 24px;
        box-shadow: 0 30px 80px rgba(15, 23, 42, 0.24);
        padding: 1.25rem;
    }

    .opportunity-modal {
        width: min(860px, calc(100% - 2rem));
    }

    .modal-header {
        margin-bottom: 1rem;
        gap: 1rem;
    }

    .modal-header-actions {
        display: flex;
        gap: 0.75rem;
        align-items: center;
    }

    .ghost-btn {
        border: 1px solid var(--border);
        border-radius: 999px;
        padding: 0.65rem 1rem;
        background: white;
        font: inherit;
        color: var(--text-primary);
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 0.45rem;
        margin-bottom: 0.9rem;
    }

    .form-grid > * {
        flex: 1 1 220px;
    }

    .label,
    .detail-label {
        font-size: 0.8rem;
        font-weight: 700;
        color: var(--text-muted);
    }

    .section-title {
        margin: 0 0 1rem;
    }

    .loading-state,
    .empty-state {
        min-height: 280px;
        display: grid;
        place-items: center;
        color: var(--text-muted);
        border: 1px dashed var(--border);
        border-radius: 20px;
        background: rgba(255, 255, 255, 0.65);
    }

    .delete-modal {
        width: min(560px, 100%);
    }

    .delete-body {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }

    .delete-details {
        flex-direction: column;
        gap: 0.6rem;
        padding: 1rem;
        border-radius: 16px;
        background: rgba(15, 23, 42, 0.04);
    }

    .detail-row {
        gap: 1rem;
    }

    .warning-text {
        color: #b45309;
        margin: 0;
    }

    .delete-all-caution {
        margin: 0.5rem 0 0;
        font-size: 0.8125rem;
        color: var(--text-secondary, #6b7280);
        text-align: right;
    }

    @media (max-width: 900px) {
        .card-main,
        .card-foot,
        .action-bar,
        .controls-bar,
        .modal-header {
            flex-direction: column;
            align-items: stretch;
        }

        .card-summary {
            align-items: flex-start;
        }

        .search-box {
            min-width: 100%;
        }

        .modal-backdrop {
            padding: 0.75rem;
        }
    }
</style>
