<script lang="ts">
    import { createBubbler, stopPropagation } from 'svelte/legacy';
    import { motionMs } from "$lib/motion";

    const bubble = createBubbler();
    import { onMount } from "svelte";
    import { fade } from "svelte/transition";
    import ActionProposalCard from "../components/ui/ActionProposalCard.svelte";
    import EvidenceSourceList from "../components/ui/EvidenceSourceList.svelte";
    import KpiStatusStrip from "../components/ui/KpiStatusStrip.svelte";
    import WabiSpinner from "../components/ui/WabiSpinner.svelte";
    import { toast } from "../stores/toasts";
    import { formatNumber } from "$lib/utils/formatters";
    import type { evidence, main } from "../../../wailsjs/go/models";
    import { CreateAccount } from "../../../wailsjs/go/main/App";
    import {
        CreateJournalEntry,
        ExportBalanceSheetCSV,
        ExportCashflowEvidencePack,
        ExportGeneralLedgerCSV,
        ExportJournalCSV,
        ExportVATReturnData,
        GenerateBalanceSheet,
        GenerateProfitAndLoss,
        GetChartOfAccounts,
        GetCashflowEvidenceCommandCenter,
        GetJournalEntries,
        ListCashflowEvidenceProposalReviews,
        ReviewCashflowEvidenceProposal,
        GetPostingCoverageReport,
        GetTrialBalanceGate,
        SyncCashflowEvidenceProposalReviews,
        UpdateAccount,
    } from "../../../wailsjs/go/main/FinanceService";

    // Props
    export const embedded = false;

    // Nav
    const views = [
        { id: "dashboard", icon: "", label: "Financial Overview" },
        { id: "coa", icon: "", label: "Chart of Accounts" },
        { id: "journal", icon: "", label: "Journal Entries" },
        { id: "reports", icon: "", label: "Reports" },
    ];
    let currentView = $state("dashboard");
    let loading = $state(false);

    // Data
    let stats = $state({ assets: 0, liabilities: 0, equity: 0, cash: 0 });
    type AccountView = {
        id?: string;
        code: string;
        name: string;
        type: string;
        balance?: number;
        is_vat: boolean;
        vat_direction?: string;
        account_group?: string;
        parent_account_id?: string;
    };
    type JournalView = {
        id: string;
        date: string;
        desc: string;
        amount: number;
    };
    let accounts: AccountView[] = $state([]);
    let journals: JournalView[] = $state([]);
    let postingCoverage: any = $state(null);
    let trialBalanceGate: any = $state(null);
    let cashflowEvidence: evidence.CommandCenter | null = $state(null);
    let cashflowProposalReviews: main.CashflowEvidenceProposalReview[] = $state([]);
    let syncingCashflowReviews = $state(false);
    let reviewingCashflowProposal = $state("");
    let reportYear = $state(new Date().getFullYear());
    let reportLoading = $state("");
    let plReport: any = $state(null);
    let bsReport: any = $state(null);
    let cashflowEvidenceMetrics = $derived([
        {
            label: "Attention",
            value: formatCurrency(cashflowEvidence?.cash?.total_attention || 0),
            meta: `${formatCurrency(cashflowEvidence?.cash?.overdue_ar || 0)} overdue`,
            status: cashflowEvidence?.cash?.status || cashflowEvidence?.overall_status || "review",
        },
        {
            label: "Posting",
            value: `${cashflowEvidence?.posting?.missing_journals ?? 0} missing`,
            meta: `${cashflowEvidence?.posting?.draft_entries ?? 0} drafts`,
            status: cashflowEvidence?.posting?.status || "review",
        },
        {
            label: "Bank Match",
            value: `${cashflowEvidence?.unmatched_bank_lines ?? 0} open`,
            meta: formatCurrency(cashflowEvidence?.unmatched_bank_amount || 0),
            status: (cashflowEvidence?.unmatched_bank_lines ?? 0) > 0 ? "review" : "ready",
        },
        {
            label: "Evidence Pack",
            value: `${cashflowEvidence?.exportable_audit_items ?? 0} items`,
            meta: `${cashflowEvidence?.open_follow_up_tasks ?? 0} follow-ups`,
            status: cashflowEvidence?.overall_status || "review",
        },
    ]);

    // Voucher Form State per FORM_FIELD_DEFINITIONS.md Section 5
    let showVoucherModal = $state(false);
    let newVoucher = $state({
        type: "Payment", // Payment, Receipt, Journal, Sales, Purchase
        voucher_no: "", // Auto-gen sequence
        date: new Date().toISOString().split("T")[0],
        reference: "", // Cheque No, Transfer Ref
        lines: [{ account: "", debit: 0, credit: 0, narration: "" }],
    });
    const voucherTypes = ["Payment", "Receipt", "Journal", "Sales", "Purchase"];

    function addVoucherLine() {
        newVoucher.lines = [
            ...newVoucher.lines,
            { account: "", debit: 0, credit: 0, narration: "" },
        ];
    }

    function removeVoucherLine(index: number) {
        newVoucher.lines = newVoucher.lines.filter((_, i) => i !== index);
    }

    let voucherTotal = $derived({
        debit: newVoucher.lines.reduce(
            (sum, l) => sum + (Number(l.debit) || 0),
            0,
        ),
        credit: newVoucher.lines.reduce(
            (sum, l) => sum + (Number(l.credit) || 0),
            0,
        ),
    });
    let isBalanced =
        $derived(voucherTotal.debit === voucherTotal.credit && voucherTotal.debit > 0);

    async function createVoucher() {
        if (!isBalanced) {
            toast.warning("Debits must equal Credits");
            return;
        }

        const lines = newVoucher.lines.map((line) => {
            const account = accounts.find((acc) => acc.code === line.account);
            return {
                account_id: account?.id || "",
                account_name: account?.name || line.account,
                debit: Number(line.debit) || 0,
                credit: Number(line.credit) || 0,
                description: line.narration || newVoucher.reference || newVoucher.type,
            };
        });

        if (lines.some((line) => !line.account_id)) {
            toast.warning("Every journal line must have an account");
            return;
        }

        try {
            await CreateJournalEntry({
                entry_date: new Date(`${newVoucher.date}T00:00:00`).toISOString(),
                description: `${newVoucher.type}: ${newVoucher.reference || "Entry"}`,
                fiscal_year: new Date(newVoucher.date).getFullYear(),
                fiscal_period: new Date(newVoucher.date).getMonth() + 1,
                source_type: "manual",
                source_id: "",
                is_auto_generated: false,
                lines,
            } as any);
            showVoucherModal = false;
            newVoucher = {
                type: "Payment",
                voucher_no: "",
                date: new Date().toISOString().split("T")[0],
                reference: "",
                lines: [{ account: "", debit: 0, credit: 0, narration: "" }],
            };
            toast.success("Voucher created");
            await loadData();
        } catch (e) {
            toast.danger(`Voucher creation failed: ${String(e)}`);
        }
    }

    // === CHART OF ACCOUNTS CRUD per TECHNICAL_SPECIFICATION.md Section 2.4 ===
    let showAccountModal = $state(false);
    let editingAccount: AccountView | null = $state(null);
    let newAccount = $state({
        code: "",
        name: "",
        type: "Asset",
        parent: "",
        is_vat: false, // Bahrain VAT account flag
    });

    const accountTypes = [
        { value: "Asset", label: "Assets", group: "BS" },
        { value: "Liability", label: "Liabilities", group: "BS" },
        { value: "Equity", label: "Equity", group: "BS" },
        { value: "Revenue", label: "Revenue", group: "PL" },
        { value: "Expense", label: "Expenses", group: "PL" },
    ];

    function accountGroupForType(type: string) {
        return accountTypes.find((t) => t.value === type)?.group || "PL";
    }

    function vatDirectionForAccount(account: { type: string; is_vat: boolean }) {
        if (!account.is_vat) return "";
        return account.type === "Liability" || account.type === "Revenue"
            ? "output"
            : "input";
    }

    function normalizeBackendAccount(account: any): AccountView {
        return {
            id: account.id,
            code: account.account_code || account.code || "",
            name: account.account_name || account.name || "",
            type: account.account_type || account.type || "Asset",
            balance: Number(account.balance) || 0,
            is_vat: Boolean(account.is_vat_account || account.is_vat),
            vat_direction: account.vat_direction || "",
            account_group: account.account_group || accountGroupForType(account.account_type || account.type || "Asset"),
            parent_account_id: account.parent_account_id || "",
        };
    }

    function normalizeBackendJournal(entry: any): JournalView {
        return {
            id: entry.entry_number || entry.id,
            date: entry.entry_date ? new Date(entry.entry_date).toISOString().split("T")[0] : "",
            desc: entry.description || "Journal entry",
            amount: Number(entry.debit_total || entry.credit_total || 0),
        };
    }

    function openAddAccount() {
        editingAccount = null;
        newAccount = {
            code: "",
            name: "",
            type: "Asset",
            parent: "",
            is_vat: false,
        };
        showAccountModal = true;
    }

    function openEditAccount(account: AccountView) {
        editingAccount = account;
        newAccount = {
            code: account.code,
            name: account.name,
            type: account.type,
            parent: account.parent_account_id || "",
            is_vat: account.is_vat,
        };
        showAccountModal = true;
    }

    async function saveAccount() {
        if (!newAccount.code || !newAccount.name) {
            toast.warning("Code and Name are required");
            return;
        }

        try {
            if (editingAccount) {
                if (!editingAccount.id) {
                    toast.warning("This account is missing its backend ID");
                    return;
                }
                await UpdateAccount(editingAccount.id, {
                    account_name: newAccount.name.trim(),
                    account_type: newAccount.type,
                    is_vat_account: newAccount.is_vat,
                    vat_direction: vatDirectionForAccount(newAccount),
                    account_group: accountGroupForType(newAccount.type),
                });
                toast.success("Account updated");
            } else {
                if (accounts.find((a) => a.code === newAccount.code)) {
                    toast.warning("Account code already exists");
                    return;
                }
                await CreateAccount({
                    account_code: newAccount.code.trim(),
                    account_name: newAccount.name.trim(),
                    account_type: newAccount.type,
                    balance: 0,
                    is_active: true,
                    is_vat_account: newAccount.is_vat,
                    vat_direction: vatDirectionForAccount(newAccount),
                    parent_account_id: newAccount.parent || "",
                    account_group: accountGroupForType(newAccount.type),
                } as any);
                toast.success("Account added");
            }
            showAccountModal = false;
            await loadData();
        } catch (e) {
            toast.danger(`Account save failed: ${String(e)}`);
        }
    }

    function proposalReviewKey(proposal: evidence.ActionProposal): string {
        return [
            proposal.action || "",
            proposal.source_type || "",
            proposal.required_deterministic_service || "",
            proposal.label || "",
        ]
            .map((part) => part.trim().toLowerCase())
            .join("|");
    }

    function proposalReviewStatus(proposal: evidence.ActionProposal): string {
        return proposalReviewFor(proposal)?.status?.replace(/_/g, " ") || "";
    }

    function proposalReviewFor(proposal: evidence.ActionProposal): main.CashflowEvidenceProposalReview | undefined {
        const key = proposalReviewKey(proposal);
        return cashflowProposalReviews.find((row) => row.proposal_key === key);
    }

    function upsertProposalReview(row: main.CashflowEvidenceProposalReview) {
        cashflowProposalReviews = [
            row,
            ...cashflowProposalReviews.filter((candidate) => candidate.id !== row.id),
        ];
    }

    async function handleCashflowEvidenceAction(action: "inspect" | "draft" | "export") {
        if (action === "export") {
            try {
                const path = await ExportCashflowEvidencePack(30);
                toast.success(`Evidence pack exported: ${path}`);
            } catch (e) {
                toast.danger(`Evidence pack export failed: ${String(e)}`);
            }
            return;
        }

        const labels = {
            inspect: "Cashflow evidence sources are visible in the command center.",
            draft: "Follow-up drafting remains routed through deterministic service approval.",
        };
        toast.info(labels[action]);
    }

    async function syncCashflowProposalReviews() {
        if (!cashflowEvidence?.action_proposals?.length) {
            toast.info("No cashflow evidence proposals are available to queue.");
            return;
        }

        syncingCashflowReviews = true;
        try {
            cashflowProposalReviews = await SyncCashflowEvidenceProposalReviews(30);
            toast.success(`${cashflowProposalReviews.length} proposal review${cashflowProposalReviews.length === 1 ? "" : "s"} queued`);
        } catch (e) {
            toast.danger(`Proposal queue sync failed: ${String(e)}`);
        } finally {
            syncingCashflowReviews = false;
        }
    }

    async function reviewCashflowProposal(proposal: evidence.ActionProposal, status: "approved" | "needs_input" | "rejected") {
        let review = proposalReviewFor(proposal);
        if (!review?.id) {
            try {
                cashflowProposalReviews = await SyncCashflowEvidenceProposalReviews(30);
                review = proposalReviewFor(proposal);
            } catch (e) {
                toast.danger(`Proposal queue sync failed: ${String(e)}`);
                return;
            }
        }
        if (!review?.id) {
            toast.warning("Queue this proposal before reviewing it.");
            return;
        }

        reviewingCashflowProposal = review.id;
        try {
            const updated = await ReviewCashflowEvidenceProposal(review.id, status, "");
            upsertProposalReview(updated);
            toast.success(`Proposal marked ${status.replace(/_/g, " ")}`);
        } catch (e) {
            toast.danger(`Proposal review failed: ${String(e)}`);
        } finally {
            reviewingCashflowProposal = "";
        }
    }

    async function generateAccountingReport(type: "pl" | "balance" | "vat") {
        if (reportYear < 2000 || reportYear > 2030) {
            toast.warning("Report year must be between 2000 and 2030");
            return;
        }

        reportLoading = type;
        try {
            if (type === "pl") {
                plReport = await GenerateProfitAndLoss(reportYear);
                toast.success(`Profit & Loss generated for ${reportYear}`);
            } else if (type === "balance") {
                bsReport = await GenerateBalanceSheet(reportYear);
                toast.success(`Balance Sheet generated for ${reportYear}`);
            } else {
                const quarter = Math.floor(new Date().getMonth() / 3) + 1;
                const filePath = await ExportVATReturnData(reportYear, quarter);
                toast.success(`VAT return exported: ${filePath || `Q${quarter} ${reportYear}`}`);
            }
        } catch (e) {
            toast.danger(`Report generation failed: ${String(e)}`);
        } finally {
            reportLoading = "";
        }
    }

    // === STATEMENT EXPORTS (Wave 9.3 B5) ===
    // The accounting chain here is single-tenant (no company field on
    // ChartOfAccount/JournalEntry), so exports honor only the active
    // reportYear — there is no company scope to filter by.
    async function exportStatement(type: "balance" | "gl" | "journal") {
        if (reportYear < 2000 || reportYear > 2030) {
            toast.warning("Report year must be between 2000 and 2030");
            return;
        }

        const loadingKey = `${type}-csv`;
        reportLoading = loadingKey;
        try {
            let filePath = "";
            if (type === "balance") {
                filePath = await ExportBalanceSheetCSV(reportYear);
            } else if (type === "gl") {
                filePath = await ExportGeneralLedgerCSV(reportYear);
            } else {
                filePath = await ExportJournalCSV(reportYear);
            }
            toast.success(`Exported: ${filePath || "Documents"}`);
        } catch (e) {
            toast.danger(`Export failed: ${String(e)}`);
        } finally {
            reportLoading = "";
        }
    }

    // === BAHRAIN VAT LOGIC per TECHNICAL_SPECIFICATION.md Section 2.4 ===
    const VAT_RATE = 0.1; // Bahrain 10% VAT (as of 2022)

    // Group accounts by type for CoA view
    let accountsByType = $derived(accountTypes.map((t) => ({
        ...t,
        accounts: accounts.filter((a) => a.type === t.value),
    })));

    // VAT Summary calculation
    let vatAccounts = $derived(accounts.filter((a) => a.is_vat));
    let outputVAT = $derived(journals
        .filter((j) => j.desc.toLowerCase().includes("sales"))
        .reduce((sum, j) => sum + j.amount * VAT_RATE, 0));
    let inputVAT = $derived(journals
        .filter((j) => j.desc.toLowerCase().includes("purchase"))
        .reduce((sum, j) => sum + j.amount * VAT_RATE, 0));
    let vatPayable = $derived(outputVAT - inputVAT);

    async function loadData() {
        loading = true;
        try {
            if (!window.go) {
                stats = {
                    assets: 1250000,
                    liabilities: 450000,
                    equity: 800000,
                    cash: 120000,
                };
                accounts = [
                    {
                        code: "1000",
                        name: "Cash",
                        type: "Asset",
                        is_vat: false,
                    },
                    {
                        code: "1100",
                        name: "Accounts Receivable",
                        type: "Asset",
                        is_vat: false,
                    },
                    {
                        code: "1200",
                        name: "Inventory",
                        type: "Asset",
                        is_vat: false,
                    },
                    {
                        code: "2000",
                        name: "Accounts Payable",
                        type: "Liability",
                        is_vat: false,
                    },
                    {
                        code: "2100",
                        name: "VAT Payable (Output)",
                        type: "Liability",
                        is_vat: true,
                    },
                    {
                        code: "2110",
                        name: "VAT Receivable (Input)",
                        type: "Asset",
                        is_vat: true,
                    },
                    {
                        code: "3000",
                        name: "Owners Equity",
                        type: "Equity",
                        is_vat: false,
                    },
                    {
                        code: "3100",
                        name: "Retained Earnings",
                        type: "Equity",
                        is_vat: false,
                    },
                    {
                        code: "4000",
                        name: "Sales Revenue",
                        type: "Revenue",
                        is_vat: false,
                    },
                    {
                        code: "4100",
                        name: "Service Revenue",
                        type: "Revenue",
                        is_vat: false,
                    },
                    {
                        code: "5000",
                        name: "Cost of Goods Sold",
                        type: "Expense",
                        is_vat: false,
                    },
                    {
                        code: "5100",
                        name: "Salaries Expense",
                        type: "Expense",
                        is_vat: false,
                    },
                    {
                        code: "5200",
                        name: "Rent Expense",
                        type: "Expense",
                        is_vat: false,
                    },
                ];
                journals = [
                    {
                        id: "JE-001",
                        date: "2025-01-01",
                        desc: "Opening Balance",
                        amount: 100000,
                    },
                    {
                        id: "JE-002",
                        date: "2025-01-15",
                        desc: "Sales Invoice #101",
                        amount: 5000,
                    },
                ];
                postingCoverage = {
                    total: 14,
                    linked: 9,
                    missing: 5,
                    draft_entries: 9,
                    is_complete: false,
                    rows: [
                        { label: "Customer Invoices", total: 8, linked: 5, missing: 3, draft_entries: 5 },
                        { label: "Customer Payments", total: 3, linked: 2, missing: 1, draft_entries: 2 },
                        { label: "Supplier Invoices", total: 2, linked: 1, missing: 1, draft_entries: 1 },
                        { label: "Supplier Payments", total: 1, linked: 1, missing: 0, draft_entries: 1 },
                    ],
                };
                cashflowEvidence = {
                    window: {
                        start: new Date().toISOString(),
                        end: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
                        label: "Next 30 days",
                    },
                    cash: {
                        open_ar: 186000,
                        overdue_ar: 42000,
                        due_in_window: 74000,
                        confirmed_uninvoiced_orders: 28000,
                        weighted_pipeline: 32000,
                        total_attention: 176000,
                        overdue_ratio: 0.23,
                        priority: "high",
                        status: "review",
                    },
                    evidence_sources: [
                        { source_type: "receivables", label: "Receivables", required: 14, present: 9, missing: 5, confidence: 0.64, status: "review", priority: "high" },
                        { source_type: "banking", label: "Bank Match", required: 8, present: 6, missing: 2, confidence: 0.75, status: "review", priority: "medium" },
                        { source_type: "documents", label: "Source Docs", required: 10, present: 8, missing: 2, confidence: 0.8, status: "ready", priority: "medium" },
                    ],
                    posting: {
                        status: "review",
                        priority: "high",
                        total_sources: 14,
                        missing_journals: 5,
                        draft_entries: 9,
                        trial_balance_ready: true,
                        message: "Posting coverage has missing journals; draft entries are ready for review.",
                    },
                    unmatched_bank_lines: 2,
                    unmatched_bank_amount: 7400,
                    open_follow_up_tasks: 3,
                    exportable_audit_items: 11,
                    overall_status: "review",
                    next_action: "Review missing journals and unmatched bank lines before export.",
                } as unknown as evidence.CommandCenter;
                cashflowProposalReviews = [];
                trialBalanceGate = {
                    is_balanced: true,
                    debit_total: 120000,
                    credit_total: 120000,
                    difference: 0,
                    entry_count: 22,
                };
            } else {
                const [backendAccounts, backendJournals, coverage, trialGate] = await Promise.all([
                    GetChartOfAccounts("All"),
                    GetJournalEntries(new Date().getFullYear(), 0, null as any, 100),
                    GetPostingCoverageReport(),
                    GetTrialBalanceGate(new Date().getFullYear(), 0),
                ]);
                accounts = backendAccounts.map(normalizeBackendAccount);
                journals = backendJournals.map(normalizeBackendJournal);
                postingCoverage = coverage;
                trialBalanceGate = trialGate;
                try {
                    cashflowEvidence = await GetCashflowEvidenceCommandCenter(30);
                    cashflowProposalReviews = await ListCashflowEvidenceProposalReviews(30, false);
                } catch (evidenceError) {
                    console.error("Cashflow evidence load failed:", evidenceError);
                    cashflowEvidence = null;
                    cashflowProposalReviews = [];
                }
                stats = {
                    assets: accounts
                        .filter((account) => account.type === "Asset")
                        .reduce((sum, account) => sum + (Number(account.balance) || 0), 0),
                    liabilities: accounts
                        .filter((account) => account.type === "Liability")
                        .reduce((sum, account) => sum + (Number(account.balance) || 0), 0),
                    equity: accounts
                        .filter((account) => account.type === "Equity")
                        .reduce((sum, account) => sum + (Number(account.balance) || 0), 0),
                    cash: accounts
                        .filter((account) => account.type === "Asset" && /cash|bank/i.test(account.name))
                        .reduce((sum, account) => sum + (Number(account.balance) || 0), 0),
                };
            }
        } catch (e) {
            toast.danger(`Accounting load failed: ${String(e)}`);
        } finally {
            loading = false;
        }
    }

    onMount(loadData);

    function formatCurrency(n) {
        return Number(n).toLocaleString("en-BH", {
            style: "currency",
            currency: "BHD",
        });
    }
</script>

<div class="page">
    <header class="header">
        <div class="header-content">
            <h1>Accounting.</h1>
            <p class="subtitle">General Ledger & Financials</p>
        </div>
        <div class="actions">
            {#if currentView === "journal"}
                <button
                    class="btn-sm"
                    onclick={() => exportStatement("journal")}
                    disabled={Boolean(reportLoading)}
                    >{reportLoading === "journal-csv" ? "Exporting..." : "Export CSV"}</button
                >
                <button
                    class="btn-primary"
                    onclick={() => (showVoucherModal = true)}
                    >+ New Entry</button
                >
            {:else if currentView === "coa"}
                <button class="btn-primary" onclick={openAddAccount}
                    >+ New Account</button
                >
            {/if}
        </div>
    </header>

    <div class="layout-split">
        <aside class="sidebar">
            <nav class="nav-menu">
                {#each views as v}
                    <button
                        class="nav-item"
                        class:active={currentView === v.id}
                        onclick={() => (currentView = v.id)}
                    >
                        <span class="icon">{v.icon}</span>
                        {v.label}
                    </button>
                {/each}
            </nav>

            <div class="quick-stats">
                <div class="stat-row">
                    <span class="lbl">Cash</span>
                    <span class="val">{formatCurrency(stats.cash)}</span>
                </div>
            </div>
        </aside>

        <main class="main-view">
            {#if loading}
                <div class="loading"><WabiSpinner size="lg" /></div>
            {:else if currentView === "dashboard"}
                <div class="dashboard-grid" in:fade={{ duration: motionMs(400) }}>
                    <div class="card">
                        <h3>Total Assets</h3>
                        <div class="big-num">
                            {formatCurrency(stats.assets)}
                        </div>
                    </div>
                    <div class="card">
                        <h3>Liabilities</h3>
                        <div class="big-num">
                            {formatCurrency(stats.liabilities)}
                        </div>
                    </div>
                    <div class="card">
                        <h3>Equity</h3>
                        <div class="big-num">
                            {formatCurrency(stats.equity)}
                        </div>
                    </div>
                    <div class="card ledger-card">
                        <h3>Posting Coverage</h3>
                        <div class="big-num">
                            {postingCoverage?.missing ?? 0}
                        </div>
                        <p class="card-subtitle">
                            {postingCoverage?.linked ?? 0} linked / {postingCoverage?.total ?? 0} eligible
                        </p>
                    </div>
                    <div class="card ledger-card">
                        <h3>Trial Balance</h3>
                        <div class="gate-status" class:ok={trialBalanceGate?.is_balanced}>
                            {trialBalanceGate?.is_balanced ? "Balanced" : "Review"}
                        </div>
                        <p class="card-subtitle">
                            Difference {formatCurrency(trialBalanceGate?.difference || 0)}
                        </p>
                    </div>
                    <div class="evidence-panel">
                        <div class="panel-head evidence-head">
                            <div>
                                <span>Cashflow Evidence</span>
                                <strong>{cashflowEvidence?.window?.label || "30 day window"}</strong>
                            </div>
                            <div class="evidence-status" data-status={cashflowEvidence?.overall_status || "review"}>
                                {cashflowEvidence?.overall_status || "review"}
                            </div>
                        </div>

                        <KpiStatusStrip items={cashflowEvidenceMetrics} />

                        <EvidenceSourceList sources={cashflowEvidence?.evidence_sources || []} />

                        <div class="evidence-next">
                            <span>{cashflowEvidence?.next_action || "Monitor cashflow evidence readiness."}</span>
                            <div>
                                <button type="button" disabled={syncingCashflowReviews} onclick={syncCashflowProposalReviews}>
                                    {syncingCashflowReviews ? "Queueing" : `Queue ${cashflowProposalReviews.length}`}
                                </button>
                                <button type="button" onclick={() => handleCashflowEvidenceAction("inspect")}>Inspect</button>
                                <button type="button" onclick={() => handleCashflowEvidenceAction("draft")}>Draft</button>
                                <button type="button" onclick={() => handleCashflowEvidenceAction("export")}>Export</button>
                            </div>
                        </div>

                        <div class="proposal-list">
                            {#each cashflowEvidence?.action_proposals || [] as proposal}
                                {@const review = proposalReviewFor(proposal)}
                                <ActionProposalCard
                                    {proposal}
                                    reviewLabel={proposalReviewStatus(proposal) || proposal.required_deterministic_service}
                                    hasReview={Boolean(review)}
                                    reviewing={reviewingCashflowProposal === review?.id}
                                    onApprove={() => reviewCashflowProposal(proposal, "approved")}
                                    onNeedsInput={() => reviewCashflowProposal(proposal, "needs_input")}
                                    onReject={() => reviewCashflowProposal(proposal, "rejected")}
                                />
                            {/each}
                        </div>
                    </div>
                    <div class="ledger-panel">
                        <div class="panel-head">
                            <span>Posting Readiness</span>
                            <strong>{postingCoverage?.draft_entries ?? 0} draft entries</strong>
                        </div>
                        <div class="coverage-list">
                            {#each postingCoverage?.rows || [] as row}
                                <div class="coverage-row">
                                    <span>{row.label}</span>
                                    <strong>{row.missing} missing</strong>
                                    <small>{row.linked}/{row.total}</small>
                                </div>
                            {/each}
                        </div>
                    </div>
                </div>
            {:else if currentView === "coa"}
                <div class="coa-view" in:fade={{ duration: motionMs(400) }}>
                    <!-- VAT Summary Card -->
                    <div class="vat-summary-card">
                        <h4>Bahrain VAT Summary (10%)</h4>
                        <div class="vat-row">
                            <span>Output VAT (Sales)</span>
                            <span class="mono">{formatCurrency(outputVAT)}</span
                            >
                        </div>
                        <div class="vat-row">
                            <span>Input VAT (Purchases)</span>
                            <span class="mono">-{formatCurrency(inputVAT)}</span
                            >
                        </div>
                        <div class="vat-row total">
                            <span>VAT Payable</span>
                            <span
                                class="mono {vatPayable >= 0
                                    ? 'negative'
                                    : 'positive'}"
                                >{formatCurrency(vatPayable)}</span
                            >
                        </div>
                    </div>

                    <!-- Grouped Accounts -->
                    {#each accountsByType as group}
                        <div class="account-group">
                            <div class="group-header">
                                <span class="group-title">{group.label}</span>
                                <span class="group-badge">{group.group}</span>
                                <span class="group-count"
                                    >{group.accounts.length}</span
                                >
                            </div>
                            <div class="account-list">
                                {#each group.accounts as acc}
                                    <div class="account-row">
                                        <span class="code">{acc.code}</span>
                                        <span class="name">
                                            {acc.name}
                                            {#if acc.is_vat}<span
                                                    class="vat-tag">VAT</span
                                                >{/if}
                                        </span>
                                        <div class="row-actions">
                                            <button
                                                class="btn-icon"
                                                onclick={() =>
                                                    openEditAccount(acc)}
                                                >Edit</button
                                            >
                                        </div>
                                    </div>
                                {/each}
                                {#if group.accounts.length === 0}
                                    <div class="empty-group">No accounts</div>
                                {/if}
                            </div>
                        </div>
                    {/each}
                </div>
            {:else if currentView === "journal"}
                <div class="table-container" in:fade={{ duration: motionMs(400) }}>
                    <table>
                        <thead
                            ><tr
                                ><th>ID</th><th>Date</th><th>Description</th><th
                                    class="right">Amount</th
                                ></tr
                            ></thead
                        >
                        <tbody>
                            {#each journals as j}
                                <tr>
                                    <td class="mono">{j.id}</td>
                                    <td class="mono">{j.date}</td>
                                    <td>{j.desc}</td>
                                    <td class="right mono"
                                        >{formatCurrency(j.amount)}</td
                                    >
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else if currentView === "reports"}
                <div class="reports-grid" in:fade={{ duration: motionMs(400) }}>
                    <!-- Wave 9.3 B4: name the source so the two P&Ls don't confuse. -->
                    <div class="source-authority-note" role="note">
                        <strong>Imported accounting figures.</strong> These statements are built from
                        imported Tally accounting data — the filed/reconciled books. They can differ from
                        the <em>Financial Dashboard</em>, which shows live, real-time ERP figures for the
                        current year (and audited figures for 2023–2024). Use these for what was filed;
                        use the dashboard for where the business stands today.
                    </div>
                    <div class="report-controls">
                        <label for="accounting-report-year">Fiscal Year</label>
                        <input
                            id="accounting-report-year"
                            type="number"
                            min="2000"
                            max="2030"
                            bind:value={reportYear}
                        />
                    </div>
                    <div class="report-card">
                        <div class="icon">P&L</div>
                        <h4>Profit & Loss</h4>
                        <button
                            class="btn-sm"
                            onclick={() => generateAccountingReport("pl")}
                            disabled={Boolean(reportLoading)}
                        >{reportLoading === "pl" ? "Generating..." : "Generate"}</button>
                    </div>
                    <div class="report-card">
                        <div class="icon">Balance</div>
                        <h4>Balance Sheet</h4>
                        <div class="report-card-actions">
                            <button
                                class="btn-sm"
                                onclick={() => generateAccountingReport("balance")}
                                disabled={Boolean(reportLoading)}
                            >{reportLoading === "balance" ? "Generating..." : "Generate"}</button>
                            <button
                                class="btn-sm"
                                onclick={() => exportStatement("balance")}
                                disabled={Boolean(reportLoading)}
                            >{reportLoading === "balance-csv" ? "Exporting..." : "Export CSV"}</button>
                        </div>
                    </div>
                    <div class="report-card">
                        <div class="icon">GL</div>
                        <h4>General Ledger</h4>
                        <p class="report-card-note">Posted entries, per account, for {reportYear}</p>
                        <button
                            class="btn-sm"
                            onclick={() => exportStatement("gl")}
                            disabled={Boolean(reportLoading)}
                        >{reportLoading === "gl-csv" ? "Exporting..." : "Export CSV"}</button>
                    </div>
                    <div class="report-card">
                        <div class="icon">VAT</div>
                        <h4>VAT Return</h4>
                        <button
                            class="btn-sm"
                            onclick={() => generateAccountingReport("vat")}
                            disabled={Boolean(reportLoading)}
                        >{reportLoading === "vat" ? "Exporting..." : "Generate"}</button>
                    </div>
                </div>

                {#if plReport}
                    <div class="financial-report" in:fade={{ duration: motionMs(400) }}>
                        <h4>Profit & Loss Statement - {plReport.year}</h4>
                        <div class="report-section">
                            <div class="report-line">
                                <span class="label">Sales Revenue</span>
                                <span class="value">{formatNumber(plReport.sales_revenue, 3)} {plReport.currency}</span>
                            </div>
                            <div class="report-line">
                                <span class="label">Other Income</span>
                                <span class="value">{formatNumber(plReport.other_income, 3)} {plReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Total Revenue</span>
                                <span class="value">{formatNumber(plReport.total_revenue, 3)} {plReport.currency}</span>
                            </div>
                        </div>

                        <div class="report-section">
                            <div class="report-line">
                                <span class="label">Cost of Goods Sold</span>
                                <span class="value">{formatNumber(plReport.cost_of_goods_sold, 3)} {plReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Gross Profit</span>
                                <span class="value">{formatNumber(plReport.gross_profit, 3)} {plReport.currency}</span>
                            </div>
                            <div class="report-line">
                                <span class="label">Gross Profit Margin</span>
                                <span class="value">{plReport.gross_profit_margin.toFixed(1)}%</span>
                            </div>
                        </div>

                        <div class="report-section">
                            <div class="report-line">
                                <span class="label">Operating Expenses</span>
                                <span class="value">{formatNumber(plReport.operating_expenses, 3)} {plReport.currency}</span>
                            </div>
                        </div>

                        <div class="report-section final">
                            <div class="report-line total">
                                <span class="label">Net Profit</span>
                                <span class="value net-profit">{formatNumber(plReport.net_profit, 3)} {plReport.currency}</span>
                            </div>
                            <div class="report-line">
                                <span class="label">Net Profit Margin</span>
                                <span class="value">{plReport.net_profit_margin.toFixed(1)}%</span>
                            </div>
                        </div>

                        <div class="report-meta">
                            <span>Generated: {new Date(plReport.generated_at).toLocaleString()}</span>
                            <span>Source: {plReport.invoice_count} invoices, {plReport.purchase_count} purchases</span>
                        </div>
                    </div>
                {/if}

                {#if bsReport}
                    <div class="financial-report" in:fade={{ duration: motionMs(400) }}>
                        <h4>Balance Sheet - As of {new Date(bsReport.as_of_date).toLocaleDateString()}</h4>

                        <div class="report-section">
                            <h5>Assets</h5>
                            <div class="report-line">
                                <span class="label">Cash</span>
                                <span class="value">{formatNumber(bsReport.cash, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line">
                                <span class="label">Accounts Receivable</span>
                                <span class="value">{formatNumber(bsReport.accounts_receivable, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line">
                                <span class="label">Inventory</span>
                                <span class="value">{formatNumber(bsReport.inventory, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Total Current Assets</span>
                                <span class="value">{formatNumber(bsReport.total_current_assets, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Total Assets</span>
                                <span class="value">{formatNumber(bsReport.total_assets, 3)} {bsReport.currency}</span>
                            </div>
                        </div>

                        <div class="report-section">
                            <h5>Liabilities</h5>
                            <div class="report-line">
                                <span class="label">Accounts Payable</span>
                                <span class="value">{formatNumber(bsReport.accounts_payable, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Total Current Liabilities</span>
                                <span class="value">{formatNumber(bsReport.total_current_liabilities, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Total Liabilities</span>
                                <span class="value">{formatNumber(bsReport.total_liabilities, 3)} {bsReport.currency}</span>
                            </div>
                        </div>

                        <div class="report-section final">
                            <h5>Equity</h5>
                            <div class="report-line">
                                <span class="label">Retained Earnings</span>
                                <span class="value">{formatNumber(bsReport.retained_earnings, 3)} {bsReport.currency}</span>
                            </div>
                            <div class="report-line total">
                                <span class="label">Total Equity</span>
                                <span class="value">{formatNumber(bsReport.total_equity, 3)} {bsReport.currency}</span>
                            </div>
                        </div>

                        <div class="report-meta">
                            <span>Generated: {new Date(bsReport.generated_at).toLocaleString()}</span>
                        </div>
                    </div>
                {/if}
            {/if}
        </main>
    </div>
</div>

<!-- Voucher Entry Modal -->
{#if showVoucherModal}
    <div
        class="modal-backdrop"
        role="button"
        tabindex="0"
        onclick={() => (showVoucherModal = false)}
        onkeydown={(event) =>
            (event.key === "Enter" || event.key === " ") &&
            (showVoucherModal = false)}
    >
        <div class="modal-card" role="presentation" tabindex="-1" onclick={stopPropagation(bubble('click'))} onkeydown={stopPropagation(bubble('keydown'))}>
            <h3>New Accounting Entry</h3>
            <div class="form-scroll">
                <!-- HEADER SECTION -->
                <div class="section-label">Header</div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="accounting-voucher-type">Voucher Type</label>
                        <select
                            id="accounting-voucher-type"
                            bind:value={newVoucher.type}
                            class="input-clean"
                        >
                            {#each voucherTypes as t}
                                <option>{t}</option>
                            {/each}
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="accounting-voucher-date">Date</label>
                        <input
                            id="accounting-voucher-date"
                            type="date"
                            bind:value={newVoucher.date}
                            class="input-clean"
                        />
                    </div>
                </div>
                <div class="form-group">
                    <label for="accounting-voucher-reference">Reference (Cheque No / Transfer Ref)</label>
                    <input
                        id="accounting-voucher-reference"
                        type="text"
                        bind:value={newVoucher.reference}
                        class="input-clean"
                        placeholder="e.g. CHQ-12345"
                    />
                </div>

                <!-- LINES SECTION -->
                <div class="section-label">
                    <span>Lines</span>
                    <button
                        type="button"
                        class="btn-add-line"
                        onclick={addVoucherLine}>+ Add Line</button
                    >
                </div>
                <div class="lines-table">
                    <div class="lines-header">
                        <span>Account</span>
                        <span>Debit</span>
                        <span>Credit</span>
                        <span></span>
                    </div>
                    {#each newVoucher.lines as line, i}
                        <div class="lines-row">
                            <select
                                bind:value={line.account}
                                class="input-clean"
                            >
                                <option value="">Select Account</option>
                                {#each accounts as acc}
                                    <option value={acc.code}
                                        >{acc.code} - {acc.name}</option
                                    >
                                {/each}
                            </select>
                            <input
                                type="number"
                                bind:value={line.debit}
                                class="input-clean"
                                placeholder="0.00"
                            />
                            <input
                                type="number"
                                bind:value={line.credit}
                                class="input-clean"
                                placeholder="0.00"
                            />
                            <button
                                type="button"
                                class="btn-remove"
                                onclick={() => removeVoucherLine(i)}>×</button
                            >
                        </div>
                    {/each}
                </div>

                <!-- TOTALS -->
                <div class="totals-row">
                    <span class="lbl">Totals:</span>
                    <span class="mono"
                        >Dr {formatCurrency(voucherTotal.debit)}</span
                    >
                    <span class="mono"
                        >Cr {formatCurrency(voucherTotal.credit)}</span
                    >
                    {#if isBalanced}
                        <span class="badge success">Balanced</span>
                    {:else}
                        <span class="badge warning">Unbalanced</span>
                    {/if}
                </div>
            </div>

            <div class="modal-actions">
                <button
                    class="btn-ghost"
                    onclick={() => (showVoucherModal = false)}>Cancel</button
                >
                <button
                    class="btn-primary"
                    onclick={createVoucher}
                    disabled={!isBalanced}
                >
                    Create Entry
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- Account Modal (Add/Edit) -->
{#if showAccountModal}
    <div
        class="modal-backdrop"
        role="button"
        tabindex="0"
        onclick={() => (showAccountModal = false)}
        onkeydown={(event) =>
            (event.key === "Enter" || event.key === " ") &&
            (showAccountModal = false)}
    >
        <div class="modal-card" role="presentation" tabindex="-1" onclick={stopPropagation(bubble('click'))} onkeydown={stopPropagation(bubble('keydown'))}>
            <h3>{editingAccount ? "Edit Account" : "Add Account"}</h3>

            <div class="form-row">
                <div class="form-group">
                    <label for="accounting-account-code">Account Code *</label>
                    <input
                        id="accounting-account-code"
                        type="text"
                        bind:value={newAccount.code}
                        class="input-clean"
                        placeholder="e.g. 1000"
                        disabled={!!editingAccount}
                    />
                </div>
                <div class="form-group">
                    <label for="accounting-account-type">Account Type *</label>
                    <select id="accounting-account-type" bind:value={newAccount.type} class="input-clean">
                        {#each accountTypes as t}
                            <option value={t.value}>{t.label}</option>
                        {/each}
                    </select>
                </div>
            </div>

            <div class="form-group">
                <label for="accounting-account-name">Account Name *</label>
                <input
                    id="accounting-account-name"
                    type="text"
                    bind:value={newAccount.name}
                    class="input-clean"
                    placeholder="e.g. Cash in Bank"
                />
            </div>

            <div class="form-group checkbox-group">
                <label class="checkbox-label">
                    <input type="checkbox" bind:checked={newAccount.is_vat} />
                    <span>VAT Account (for Bahrain VAT tracking)</span>
                </label>
            </div>

            <div class="modal-actions">
                <button
                    class="btn-ghost"
                    onclick={() => (showAccountModal = false)}>Cancel</button
                >
                <button class="btn-primary" onclick={saveAccount}>
                    {editingAccount ? "Update" : "Add Account"}
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
    .source-authority-note {
        grid-column: 1 / -1;
        padding: var(--space-3) var(--space-4);
        margin-bottom: var(--space-3);
        background: var(--surface-sunken, var(--paper));
        border: 1px solid var(--border-subtle);
        border-left: 3px solid var(--accent, var(--ink));
        border-radius: var(--radius-md);
        color: var(--ink-muted, var(--ink-faint));
        font-size: var(--text-sm);
        line-height: 1.5;
    }
    .source-authority-note strong { color: var(--ink); }

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
    .btn-primary {
        background: var(--ink);
        color: var(--paper);
        border: none;
        padding: 8px 20px;
        border-radius: var(--radius-pill);
        cursor: pointer;
        font-size: 13px;
    }

    .layout-split {
        display: grid;
        grid-template-columns: 240px 1fr;
        gap: var(--space-4);
        flex: 1;
        min-height: 0;
    }

    .sidebar {
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        border-right: 1px solid var(--border-subtle);
        padding-right: var(--space-3);
    }
    .nav-menu {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .nav-item {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 10px;
        border: none;
        background: transparent;
        color: var(--ink-light);
        cursor: pointer;
        border-radius: 8px;
        font-size: 13px;
        text-align: left;
    }
    .nav-item:hover {
        background: var(--paper-subtle);
        color: var(--ink);
    }
    .nav-item.active {
        background: var(--ink);
        color: var(--paper);
        font-weight: 500;
    }

    .quick-stats {
        background: var(--paper-subtle);
        padding: var(--space-4);
        border-radius: 12px;
    }
    .stat-row {
        display: flex;
        justify-content: space-between;
        font-size: 12px;
    }
    .stat-row .lbl {
        color: var(--ink-light);
    }
    .stat-row .val {
        font-weight: 600;
        font-family: var(--font-mono);
    }

    .main-view {
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        padding-left: var(--space-2);
    }
    .loading {
        display: flex;
        justify-content: center;
        padding: 40px;
    }

    /* Dashboard */
    .dashboard-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
        gap: var(--space-4);
    }
    .card {
        background: var(--paper-subtle);
        padding: var(--space-4);
        border-radius: 16px;
        border: 1px solid var(--border-subtle);
    }
    .card h3 {
        margin: 0 0 12px;
        font-size: 11px;
        text-transform: uppercase;
        color: var(--ink-light);
    }
    .big-num {
        font-size: 28px;
        font-weight: 300;
        font-family: var(--font-mono);
    }
    .ledger-card .big-num {
        color: #854d0e;
    }
    .card-subtitle {
        margin: 8px 0 0;
        font-size: 12px;
        color: var(--ink-light);
    }
    .gate-status {
        display: inline-flex;
        align-items: center;
        min-height: 34px;
        padding: 0 12px;
        border-radius: 6px;
        background: #fef9c3;
        color: #854d0e;
        font-weight: 700;
        font-size: 13px;
    }
    .gate-status.ok {
        background: #dcfce7;
        color: #166534;
    }
    .ledger-panel {
        grid-column: 1 / -1;
        background: var(--paper);
        border: 1px solid var(--border-medium);
        border-radius: 12px;
        overflow: hidden;
    }
    .evidence-panel {
        grid-column: 1 / -1;
        display: grid;
        gap: 0;
        background: var(--paper);
        border: 1px solid var(--border-medium);
        border-radius: 8px;
        overflow: hidden;
    }
    .evidence-head > div:first-child {
        display: grid;
        gap: 4px;
    }
    .evidence-status {
        min-width: 76px;
        padding: 6px 10px;
        border-radius: 6px;
        background: #fef9c3;
        color: #854d0e;
        text-align: center;
        text-transform: uppercase;
        font-size: 11px;
        font-weight: 700;
    }
    .evidence-status[data-status="ready"] {
        background: #dcfce7;
        color: #166534;
    }
    .evidence-status[data-status="blocked"],
    .evidence-status[data-status="critical"] {
        background: #fee2e2;
        color: #991b1b;
    }
    .evidence-next {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        padding: 12px 16px;
        border-top: 1px solid var(--border-subtle);
        color: var(--ink);
        font-size: 13px;
    }
    .evidence-next > span {
        min-width: 0;
    }
    .evidence-next > div {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        justify-content: flex-end;
    }
    .evidence-next button {
        min-height: 32px;
        padding: 0 12px;
        border: 1px solid var(--border-medium);
        border-radius: 6px;
        background: var(--paper-subtle);
        color: var(--ink);
        cursor: pointer;
        font-size: 12px;
    }
    .evidence-next button:hover {
        border-color: var(--ink-light);
    }
    .proposal-list {
        display: grid;
        gap: 1px;
        background: var(--border-subtle);
        border-top: 1px solid var(--border-subtle);
    }
    .panel-head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        padding: 12px 16px;
        border-bottom: 1px solid var(--border-subtle);
        font-size: 12px;
        color: var(--ink-light);
    }
    .panel-head strong {
        color: var(--ink);
        font-family: var(--font-mono);
        font-weight: 600;
    }
    .coverage-list {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 1px;
        background: var(--border-subtle);
    }
    .coverage-row {
        min-width: 0;
        background: var(--paper);
        padding: 12px;
        display: grid;
        gap: 4px;
    }
    .coverage-row span {
        font-size: 12px;
        color: var(--ink-light);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    .coverage-row strong {
        font-size: 16px;
        font-family: var(--font-mono);
    }
    .coverage-row small {
        color: var(--ink-light);
        font-size: 11px;
        font-family: var(--font-mono);
    }

    /* Tables */
    .table-container {
        background: var(--paper);
        border-radius: 12px;
        border: 1px solid var(--border-medium);
        overflow: hidden;
    }
    table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;
    }
    th {
        text-align: left;
        padding: 10px 16px;
        background: var(--paper-subtle);
        color: var(--ink-light);
        font-weight: 500;
        font-size: 11px;
        text-transform: uppercase;
        position: sticky;
        top: 0;
    }
    td {
        padding: 10px 16px;
        border-bottom: 1px solid var(--border-subtle);
    }
    .right {
        text-align: right;
    }
    .mono {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--ink-light);
    }
    .badge {
        padding: 2px 8px;
        border-radius: 4px;
        background: var(--paper-subtle);
        border: 1px solid var(--border-medium);
        font-size: 10px;
        text-transform: uppercase;
    }
    /* Reports */
    .reports-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: var(--space-4);
    }
    .report-controls {
        background: var(--paper-subtle);
        padding: 20px;
        border-radius: 12px;
        border: 1px solid var(--border-subtle);
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    .report-controls label {
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--ink-light);
    }
    .report-controls input {
        padding: 8px 10px;
        border: 1px solid var(--border-medium);
        border-radius: 6px;
        background: var(--paper);
        color: var(--ink);
    }
    .report-card {
        background: var(--paper-subtle);
        padding: 20px;
        border-radius: 12px;
        border: 1px solid var(--border-subtle);
        text-align: center;
    }
    .report-card .icon {
        font-size: 24px;
        margin-bottom: 12px;
    }
    .report-card h4 {
        margin: 0 0 16px;
        font-weight: 500;
    }
    .report-card-note {
        margin: -8px 0 12px;
        font-size: 11px;
        color: var(--ink-light);
    }
    .report-card-actions {
        display: flex;
        justify-content: center;
        gap: 8px;
    }
    .btn-sm {
        padding: 6px 16px;
        border: 1px solid var(--border-medium);
        background: var(--paper);
        border-radius: 6px;
        cursor: pointer;
        font-size: 12px;
    }
    .btn-sm:disabled {
        cursor: wait;
        opacity: 0.62;
    }

    /* Generated report display (Fix Rec2: render what generateAccountingReport returns) */
    .financial-report {
        grid-column: 1 / -1;
        margin-top: var(--space-2);
        padding: var(--space-4);
        background: var(--paper);
        border: 1px solid var(--border-medium);
        border-radius: var(--radius-lg);
    }
    .financial-report h4 {
        margin: 0 0 var(--space-3);
        font-size: var(--text-lg, 16px);
        font-weight: 600;
        color: var(--ink);
    }
    .financial-report h5 {
        margin: var(--space-3) 0 var(--space-1);
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.05em;
        color: var(--ink-light);
    }
    .report-section {
        margin-bottom: var(--space-3);
        padding-bottom: var(--space-2);
        border-bottom: 1px solid var(--border-subtle);
    }
    .report-section.final {
        border-bottom: 2px solid var(--ink);
    }
    .report-line {
        display: flex;
        justify-content: space-between;
        padding: 6px 0;
        font-size: 13px;
        font-variant-numeric: tabular-nums lining-nums;
    }
    .report-line.total {
        font-weight: 600;
        border-top: 1px solid var(--border-subtle);
        padding-top: 8px;
        margin-top: 2px;
    }
    .report-line .label {
        color: var(--ink);
    }
    .report-line .value {
        color: var(--ink);
        font-family: var(--font-mono);
    }
    .report-line .value.net-profit {
        font-weight: 700;
        font-size: 15px;
    }
    .report-meta {
        margin-top: var(--space-2);
        padding-top: var(--space-2);
        border-top: 1px solid var(--border-subtle);
        display: flex;
        justify-content: space-between;
        font-size: 11px;
        color: var(--ink-light);
    }

    /* Modal */
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.6);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        backdrop-filter: blur(4px);
    }
    .modal-card {
        background: var(--paper);
        padding: 24px;
        border-radius: 16px;
        width: 640px;
        max-width: 95%;
        max-height: 85vh;
        display: flex;
        flex-direction: column;
        box-shadow: var(--shadow-xl);
    }
    .modal-card h3 {
        margin: 0 0 16px;
        font-weight: 400;
        font-family: var(--font-heading);
    }
    .form-scroll {
        flex: 1;
        overflow-y: auto;
        max-height: 55vh;
    }
    .section-label {
        font-size: 10px;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--ink-light);
        font-weight: 500;
        margin-top: 16px;
        margin-bottom: 8px;
        padding-bottom: 4px;
        border-bottom: 1px solid var(--border-subtle);
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .section-label:first-child {
        margin-top: 0;
    }
    .form-group {
        margin-bottom: 12px;
        flex: 1;
    }
    .form-group label {
        font-size: 10px;
        text-transform: uppercase;
        color: var(--ink-light);
        display: block;
        margin-bottom: 3px;
    }
    .form-row {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }
    .input-clean {
        width: 100%;
        padding: 8px;
        border: 1px solid var(--border-medium);
        border-radius: 6px;
        box-sizing: border-box;
        font-size: 13px;
    }

    /* Lines Table */
    .btn-add-line {
        font-size: 11px;
        padding: 2px 10px;
        background: var(--paper);
        border: 1px solid var(--border-medium);
        border-radius: 4px;
        cursor: pointer;
    }
    .lines-table {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }
    .lines-header {
        display: grid;
        grid-template-columns: 2fr 1fr 1fr 32px;
        gap: 8px;
        font-size: 10px;
        text-transform: uppercase;
        color: var(--ink-light);
        padding: 0 4px;
    }
    .lines-row {
        display: grid;
        grid-template-columns: 2fr 1fr 1fr 32px;
        gap: 8px;
        align-items: center;
    }
    .btn-remove {
        background: transparent;
        border: none;
        font-size: 18px;
        cursor: pointer;
        color: var(--ink-light);
        padding: 0;
        width: 24px;
        height: 24px;
        line-height: 24px;
    }
    .btn-remove:hover {
        color: #dc2626;
    }

    /* Totals Row */
    .totals-row {
        display: flex;
        gap: 16px;
        align-items: center;
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid var(--border-medium);
        font-size: 12px;
    }
    .totals-row .lbl {
        color: var(--ink-light);
    }
    .badge.success {
        background: #dcfce7;
        color: #166534;
    }
    .badge.warning {
        background: #fef9c3;
        color: #854d0e;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--border-subtle);
    }
    .btn-ghost {
        background: transparent;
        border: none;
        cursor: pointer;
        font-size: 13px;
        padding: 8px 16px;
    }

    /* CoA View */
    .coa-view {
        display: flex;
        flex-direction: column;
        gap: var(--space-4);
    }
    .vat-summary-card {
        background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
        border-radius: var(--radius-lg);
        padding: var(--space-4);
        border: 1px solid #fcd34d;
    }
    .vat-summary-card h4 {
        margin: 0 0 var(--space-2);
        font-size: 14px;
    }
    .vat-row {
        display: flex;
        justify-content: space-between;
        padding: 6px 0;
        font-size: 13px;
    }
    .vat-row.total {
        border-top: 1px solid rgba(0, 0, 0, 0.1);
        margin-top: 8px;
        padding-top: 8px;
        font-weight: 600;
    }
    .positive {
        color: #22c55e;
    }
    .negative {
        color: #dc2626;
    }

    .account-group {
        background: var(--paper-subtle);
        border-radius: var(--radius-lg);
        overflow: hidden;
    }
    .group-header {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-2) var(--space-3);
        background: rgba(0, 0, 0, 0.03);
        border-bottom: 1px solid var(--border-subtle);
    }
    .group-title {
        font-weight: 600;
        font-size: 13px;
    }
    .group-badge {
        background: var(--ink);
        color: var(--paper);
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 10px;
        font-weight: 600;
    }
    .group-count {
        margin-left: auto;
        color: var(--ink-light);
        font-size: 12px;
    }
    .account-list {
        padding: var(--space-1);
    }
    .account-row {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        padding: 8px var(--space-2);
        border-radius: 6px;
    }
    .account-row:hover {
        background: rgba(0, 0, 0, 0.03);
    }
    .account-row .code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--ink-light);
        width: 60px;
    }
    .account-row .name {
        flex: 1;
        font-size: 13px;
    }
    .vat-tag {
        display: inline-block;
        background: #fef3c7;
        color: #92400e;
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 9px;
        font-weight: 600;
        margin-left: 8px;
    }
    .row-actions {
        display: flex;
        gap: 4px;
        opacity: 0;
        transition: opacity 0.15s;
    }
    .account-row:hover .row-actions {
        opacity: 1;
    }
    .btn-icon {
        background: transparent;
        border: none;
        cursor: pointer;
        padding: 4px;
        font-size: 14px;
    }
    .empty-group {
        padding: var(--space-2);
        color: var(--ink-light);
        font-size: 12px;
        font-style: italic;
    }
    .checkbox-group {
        margin-top: var(--space-2);
    }
    .checkbox-label {
        display: flex;
        align-items: center;
        gap: 8px;
        cursor: pointer;
        font-size: 13px;
    }

    @media (max-width: 900px) {
        .layout-split {
            grid-template-columns: 1fr;
        }
        .sidebar {
            position: static;
        }
        .coverage-list {
            grid-template-columns: repeat(2, minmax(0, 1fr));
        }
        .evidence-next {
            align-items: flex-start;
            flex-direction: column;
        }
    }

    @media (max-width: 560px) {
        .coverage-list {
            grid-template-columns: 1fr;
        }
        .form-row,
        .lines-header,
        .lines-row {
            grid-template-columns: 1fr;
        }
    }
</style>
